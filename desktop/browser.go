package main

// Native in-app browser: each browser tab is a real webview of whatever engine
// the OS ships (WebView2, WebKitGTK, WKWebView), positioned over the dock's
// browser pane. This exists because iframes can't render sites that send
// X-Frame-Options/CSP deny (YouTube, Google, anything with bot checks), and
// because the AI needs to read real page content (BrowserGetText).
//
// This file is everything that does not care which engine that is: the injected
// scripts, the message bridge's security model, tab bookkeeping, and every
// Wails binding. One platform file behind it supplies the engine —
// browser_windows.go today, browser_linux.go and browser_darwin.go later. See
// PLATFORM-SUPPORT.md for the file map and ARCHITECTURE.md §48 for why the
// bindings themselves are never behind a build tag.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/statereport"
)

// tabView is one platform's live webview for one tab. Every method is called
// on the thread that owns the webview — that is, from inside hostBackend.do.
type tabView interface {
	navigate(url string)
	eval(js string)
	setBounds(x, y, w, h int)
	// setVisible(true) both shows the view and raises it: on Windows two
	// webviews in the same top-level window composite independently, so a tab
	// that is merely shown can stay behind the app's own webview — loaded,
	// painting, invisible.
	setVisible(visible bool)
	// detach gives this tab a window of its own: out of the app, with a frame,
	// a taskbar button and the user in charge of where it sits. Called once per
	// tab and never undone — a detached page is separate from the session it
	// came from and outlives it (the owner’s rule, 8 ก.ย.).
	//
	// Nothing about how the agent reaches the page changes, which is why this
	// is one method rather than a mode: the tools talk to the engine, not to a
	// rectangle. A host with no such concept may do nothing, and the tab simply
	// stays in the panel.
	detach(title string, w, h int)
	// setShape cuts the view to a phone screen: a corner radius, and a notch
	// taken out of the top, both in device pixels. All zero is a plain
	// rectangle, which is what every tab that is not emulating a phone gets.
	//
	// It is a property of the WINDOW rather than something drawn over it,
	// because a native view composites above everything the app paints — a
	// bezel in the DOM would be a bezel behind the page. A host with no way to
	// cut a window may do nothing, and the emulation is a rectangle again.
	setShape(radius, notchW, notchH, notchY int)
	setZoom(factor float64)
	openDevTools()
	destroy()
	// capture asks the engine for a PNG of the visible page. Called on the
	// webview thread like everything else here, but answered on the channel
	// rather than returned: the engine delivers its answer through the same
	// message pump this thread is running, so a caller that blocked here would
	// be blocking the thing it is waiting for. See browser_shot.go.
	capture() <-chan shotResult
	// callEngine runs one of the engine's own protocol methods and hands back
	// its answer as JSON. Same threading and same channel-not-return reason as
	// capture.
	//
	// It exists because exporting a deck needs three things no portable
	// vocabulary covers — print this page to PDF, photograph this rectangle,
	// measure where the slides are — and inventing a portable spelling for each
	// would be inventing an abstraction over one implementation. On Windows
	// these are Chrome DevTools Protocol methods (browser_windows.go). A future
	// WebKit host answers the same three questions its own way or not at all,
	// and the caller finds out through an error rather than through a lie.
	//
	// Unlike capture, printing does NOT need the page on screen: it is a
	// separate pipeline from compositing. That is the whole reason a deck can be
	// exported from a webview nobody ever sees (deck_render.go).
	callEngine(method, paramsJSON string) <-chan engineReply
	// sendBehind drops this tab to the bottom of the Z order, leaving it laid
	// out and painting with the app's own webview drawn over it. The one caller
	// is the deck export, which needs a page that composites (so a capture has
	// frames) and that nobody sees (so nothing flashes). See browser_windows.go
	// for why the obvious alternatives — hidden, or parked off-screen — both
	// come back blank.
	sendBehind()
}

// engineReply is one finished (or failed) engine call, in the portable
// vocabulary tabView is written in — browser.go must not name any one engine's
// types. JSON is the protocol's own answer object, undecoded.
type engineReply struct {
	JSON string
	Err  error
}

// tabCallbacks are the portable reactions a platform host wires into a tab it
// creates.
type tabCallbacks struct {
	// onMessage carries one postMessage envelope plus the sending frame's real
	// origin as the engine reports it — never what the page claims. See
	// aetoxMsg for why that distinction is the whole security model.
	onMessage func(raw, source string)
	// onNavDone fires when a navigation finishes. ok is whether the page
	// actually loaded, as opposed to the engine stopping on its own error
	// page; view is passed in because this can fire before the caller has
	// finished storing it.
	onNavDone func(view tabView, ok bool)
	// onEngineError carries the engine's own complaint about a call we made —
	// a refused COM call, a controller that would not create, a bad browser
	// path. Not a page error: the page is what onNavDone reports on.
	//
	// This existed as a log line and nothing else, which is how §127.8 stayed
	// invisible for a week. The engine said "This method can only be called
	// from the thread that created the object" every single time, into a file
	// nobody was reading, while the agent was handed "page did not finish
	// loading" and reasonably concluded the network was bad — then told the
	// user so. The tool's answer has to be able to carry what the engine said,
	// or the agent is guessing about its own tools.
	onEngineError func(err error)
	// onWindowClosed fires when somebody closes a DETACHED tab’s own window.
	// It is the one way a tab can end that Aetox’s own strip never sees, and
	// without it that × would leave the agent holding a page that is gone —
	// the orphan §127 spent a week on, arrived at from the other direction.
	onWindowClosed func()
	// onEngineGone says the engine behind this tab is gone for good — its
	// browser process exited, or the webview has been closed under us — and
	// nothing this view is asked from now on will succeed. Distinct from
	// onEngineError because the two want opposite responses: an error is
	// something to report about a call, this is something to REPLACE, and the
	// host answers it by putting a new engine behind the same tab (revive).
	//
	// Until 6 ก.ย. there was no such channel. The browser process behind an
	// agent's tab ended at 01:47, the tab stayed in the map, and every call for
	// the next twenty minutes was refused with the same sentence while the
	// tool reported a page that would not load.
	onEngineGone func(err error)
}

// hostBackend is one platform's webview host.
type hostBackend interface {
	// start brings the host up — its owning thread, message pump, window
	// class, whatever the platform needs. Idempotent; blocks until ready.
	start() error
	// do runs fn on the thread that owns the webviews.
	//
	// ALWAYS asynchronous, on every platform. Windows has a dedicated STA
	// thread for this; GTK and Cocoa require the webviews on the app's *main*
	// thread, which makes the obvious dispatch_sync/g_main_context_invoke_sync
	// spelling deadlock — browserSnapshot calls do() and then blocks up to
	// five seconds waiting for the page to answer, and that is the path the
	// agent reads pages through. ARCHITECTURE.md §48 Decision 3.
	do(fn func())
	// openTab creates a webview at the given physical-pixel bounds and starts
	// it navigating. Called from inside do, so it is already on the owning
	// thread. Returns nil if the platform could not create it, having logged
	// why.
	openTab(id, url string, x, y, w, h int, cb tabCallbacks) tabView
}

// aetoxMsg is the JSON envelope pages post back to Go via the platform's script
// bridge (see metaScript / textScript).
//
// SECURITY: any page loaded in the tab can call that bridge itself, at any
// time, with an arbitrary __aetox envelope — it is not exclusive to our own
// injected scripts. Two checks guard against that (see onMessage): the "meta"
// case cross-checks the claimed URL against the sending frame's real origin as
// reported by the engine itself (a page cannot forge that), so a page can't
// make the address bar show a URL it isn't actually at (phishing-enabling
// spoof). The "text" case additionally requires a per-request Token minted by
// BrowserGetText, so a page can't preempt/replay a fake page-content response
// into the AI agent's read path. Neither check stops a page from lying within
// its own real DOM/title — that's inherent to any "agent reads a live page"
// feature and is a prompt-injection risk to be handled by treating fetched
// page text as untrusted data, not by this transport.
type aetoxMsg struct {
	Aetox    string           `json:"__aetox"`
	Title    string           `json:"title,omitempty"`
	URL      string           `json:"url,omitempty"`
	Text     string           `json:"text,omitempty"`
	Token    string           `json:"token,omitempty"`
	Elements []browserElement `json:"elements,omitempty"`
	Images   []browserImage   `json:"images,omitempty"`
	// "text": how many there really were, and how much of the page could not
	// be entered at all. The lists above stop at a cap; these do not, so the
	// read can say what it left out instead of presenting a truncated list as
	// the whole page. See textScript.
	ElementsTotal int `json:"elementsTotal,omitempty"`
	ImagesTotal   int `json:"imagesTotal,omitempty"`
	Frames        int `json:"frames,omitempty"`
	// "pick" only: what the user pointed at, whether they left the mode without
	// pointing at anything, and whether they drew — in which case the marks are
	// still on the page waiting to be photographed. See browser_pick.go.
	Picks     []browserPick `json:"picks,omitempty"`
	Cancelled bool          `json:"cancelled,omitempty"`
	Drawn     bool          `json:"drawn,omitempty"`
	// "wait": whether what was waited for turned up before the deadline.
	// "act": whether the ref a click or a type was aimed at matched anything,
	// and what it turned out to be. Sent BEFORE the action runs, so a click
	// that navigates cannot destroy the document before its own report gets
	// out. See aetoxActJS.
	Found bool   `json:"found,omitempty"`
	Ref   int    `json:"ref,omitempty"`
	Tag   string `json:"tag,omitempty"`
	Label string `json:"label,omitempty"`
	// "act" from a type only: which way the text goes in ("value" or "keys"),
	// whether focusing the element left an editable thing focused, and where
	// the element is on screen for the real click that fixes it when it did
	// not. See typeScript and browser_keys.go.
	Mode   string  `json:"mode,omitempty"`
	Active bool    `json:"active,omitempty"`
	CX     float64 `json:"cx,omitempty"`
	CY     float64 `json:"cy,omitempty"`
	// Where the page's focus was before the type touched anything, and where
	// it ended up, each as one short descriptor. See aetoxDescribe.
	FocusBefore string `json:"focusBefore,omitempty"`
	Focus       string `json:"focus,omitempty"`
	// Kept: the target was a hidden proxy and the page already had an editor
	// focused, so focus was left where the page put it.
	Kept bool `json:"kept,omitempty"`
	// Mouse ("act" from a click): the element is not HTML and has no click()
	// — the click has to be a real pointer at cx, cy. See clickScript.
	Mouse bool `json:"mouse,omitempty"`
	// "act" from a point, under, focus, viewport or state script: what is
	// under a point, the viewport in CSS px and its pixel ratio, how far the
	// page is scrolled, how many interactive elements it has, and — for a
	// file input — whether it is one and what it accepts. See aetoxPointJS.
	Under   string  `json:"under,omitempty"`
	VW      int     `json:"vw,omitempty"`
	VH      int     `json:"vh,omitempty"`
	DPR     float64 `json:"dpr,omitempty"`
	ScrollX int     `json:"scrollX,omitempty"`
	ScrollY int     `json:"scrollY,omitempty"`
	Count   int     `json:"count,omitempty"`
	// "act" from refMarkScript: which refs were drawn onto the page, and (in
	// Count above) how many were in the frame before the cap. See
	// browser_refmarks.go — the key printed under a marked capture is built
	// from these.
	Marks []int `json:"marks,omitempty"`
	// "act" from findTextScript: the plain-text matches, described, when
	// there were several to choose between.
	Matches   []string `json:"matches,omitempty"`
	FileInput bool     `json:"fileInput,omitempty"`
	Multiple  bool     `json:"multiple,omitempty"`
	Accept    string   `json:"accept,omitempty"`
	// "text" and "act": how much of the viewport is canvas, and how many. A
	// page that paints its content is a page whose text is only its chrome.
	// See aetoxCanvasJS.
	CanvasShare float64 `json:"canvasShare,omitempty"`
	CanvasCount int     `json:"canvasCount,omitempty"`
	// "log": one of the page's own recorders, read on demand. Armed travels
	// with the entries deliberately — an empty buffer and a page the recorder
	// never reached are the same list, and only one of them is evidence. See
	// browser_log.go.
	Kind    string            `json:"kind,omitempty"`
	Log     []browserLogEntry `json:"log,omitempty"`
	Dropped int               `json:"dropped,omitempty"`
	Armed   bool              `json:"armed,omitempty"`
	// "dialog": which of alert/confirm/prompt the page called, what it said, and
	// what we answered on its behalf. See dialogScript.
	Dialog  string `json:"dialog,omitempty"`
	Message string `json:"message,omitempty"`
	Answer  string `json:"answer,omitempty"`
}

// browserElement is one clickable/typeable element found on the page, tagged
// with a data-aetox-ref attribute so a later browser_click/browser_type call
// can find the same node again by ref.
type browserElement struct {
	Ref  int    `json:"ref"`
	Tag  string `json:"tag"`
	Role string `json:"role,omitempty"`
	Text string `json:"text"`
	// Focused: this is where the page's keyboard goes right now. Hidden: an
	// input or textarea the page keeps out of sight, which is a keyboard proxy
	// and not a field. Both exist because of Google Sheets on 5 ก.ย.: the list
	// showed `[20] textbox: ""` and `[25] textarea: ""` with nothing to tell
	// them apart, the model picked the textarea, and the keys went into an
	// off-screen Trix editor while the cell editor — [20], focused all along —
	// got nothing. See typeScript.
	Focused bool `json:"focused,omitempty"`
	Hidden  bool `json:"hidden,omitempty"`
}

// browserImage is one meaningful image found on the page — its absolute URL
// and alt text, so the model can show it in chat with markdown ![alt](src).
type browserImage struct {
	Src string `json:"src"`
	Alt string `json:"alt,omitempty"`
}

// browserSnapshot is the result of one textScript round trip: page text plus
// the interactive elements and images found on it.
type browserSnapshot struct {
	Text     string
	Elements []browserElement
	Images   []browserImage
	// ElementsTotal and ImagesTotal are how many the page really had. They are
	// counted past the caps the lists above stop at, which is the whole point:
	// a list of 150 and a page of 150 look identical otherwise, and the reader
	// of a silently-cut list has no way to know it is missing the button it is
	// looking for.
	ElementsTotal int
	ImagesTotal   int
	// BlockedFrames is how many cross-origin frames this page has. Their
	// contents cannot be reached from JS by any means, so they are counted and
	// reported rather than quietly omitted.
	BlockedFrames int
	// CanvasShare is how much of the viewport is covered by <canvas>, 0..1,
	// and CanvasCount how many there are. Google Docs, Sheets and Slides,
	// Figma, a chart — all paint what they show, and none of what they paint
	// is in Text. Reported so a read can say that its text is the page's
	// chrome and not its content, instead of presenting the toolbar as the
	// document. See aetoxCanvasJS.
	CanvasShare float64
	CanvasCount int
}

// metaScript reports the page's real title and URL back over the bridge. The
// call itself is the one part of these scripts that is engine-specific —
// bridgePost is `window.chrome.webview.postMessage` on WebView2 and
// `window.webkit.messageHandlers.aetox.postMessage` on both WebKits — so it
// comes from the platform file and everything below is shared.
func metaScript() string {
	return bridgePost + `(JSON.stringify({__aetox:"meta",title:document.title,url:location.href}))`
}

// aetoxScanJS is the walker every page script shares: the document roots this
// page really has, and the way to find a tagged element inside any of them.
//
// `document` alone was the whole search until 2026-08-22, and `document` is not
// the page. An element inside a shadow root is invisible to
// document.querySelectorAll, and so is one inside an iframe — which is where
// checkout forms, embedded editors and most component-library controls live.
//
// The failure was silent and, worse, disguised as a different one: `read` came
// back with no elements, and read's own guidance teaches the model that an
// empty page means "not finished loading, use wait". So the honest response to
// a page we could not see was to wait for it, and then wait again.
//
// A cross-origin frame cannot be entered from JS by any technique. Those are
// counted rather than opened, so a read can report how much of the page it was
// unable to reach instead of implying there was nothing there.
//
// The node budget is what keeps this inside browserSnapshot's five-second
// deadline on a page with a very large DOM. It bounds the walk rather than the
// collecting, because the walk is the part whose cost the caller cannot
// predict — and `cut` is reported for the same reason every other cap here is.
const aetoxScanJS = `
  var AETOX_BUDGET=20000;
  function aetoxScan(){
    var roots=[document],blocked=0,seen=0;
    for(var i=0;i<roots.length;i++){
      var all;
      try{all=roots[i].querySelectorAll('*');}catch(e){continue;}
      for(var j=0;j<all.length;j++){
        if(++seen>AETOX_BUDGET)return{roots:roots,blocked:blocked,cut:true};
        var el=all[j];
        if(el.shadowRoot)roots.push(el.shadowRoot);
        if(el.tagName==='IFRAME'){
          var d=null;
          try{d=el.contentDocument;}catch(e){d=null;}
          if(d&&d.body)roots.push(d);else blocked++;
        }
      }
    }
    return{roots:roots,blocked:blocked,cut:false};
  }
  function aetoxFind(ref){
    var s=aetoxScan();
    for(var i=0;i<s.roots.length;i++){
      var el;
      try{el=s.roots[i].querySelector('[data-aetox-ref="'+ref+'"]');}catch(e){el=null;}
      if(el)return el;
    }
    return null;
  }
`

// aetoxTextJS is what "the text of this page" means, in one place.
//
// document.body.innerText is not it, and it is short in two different ways.
// An iframe's document is a separate one and its text is in none of its
// parent's. A shadow root's text is in none of its host's either: innerText is
// built from an element's light-DOM descendants, and shadow content is not
// among them however plainly it is on screen. Measured on a fixture page
// 2026-08-22 — a host whose shadow root renders a paragraph reports
// host.innerText === "" and document.body.innerText does not contain a word of
// it, while the shadow root's own children report it normally. That last part
// is what this function is built on.
//
// One definition because `read` and `wait` both need the same answer or they
// contradict each other: a `wait` for a word `read` can see would poll until
// its deadline and then report the word absent, which is the worst kind of
// wrong here because an absent word reads as a page still loading, and the
// response to that is to wait again.
//
// The cost is the whole walk, on every poll, and it was measured before being
// accepted rather than argued about. On en.wikipedia.org/wiki/World_War_II
// (16,844 nodes): aetoxScan 3.755 ms, document.body.innerText 7.08 ms. The
// scan is cheaper than the innerText call `wait` was already paying five times
// a second, so this is not a new order of cost — and AETOX_BUDGET still bounds
// the pathological page.
const aetoxTextJS = `
  function aetoxText(){
    var roots=aetoxScan().roots;
    var t=document.body?(document.body.innerText||""):"";
    for(var i=1;i<roots.length;i++){
      var r=roots[i];
      /* A document root (a same-origin frame) has a body; a shadow root does
         not, and is read through its own children — each of which reports
         rendered text normally even though its host reports none. */
      if(r.body){if(r.body.innerText)t+='\n'+r.body.innerText;continue;}
      for(var j=0;j<r.children.length;j++){
        var c=r.children[j];
        if(c.innerText)t+='\n'+c.innerText;
      }
    }
    return t;
  }
`

// aetoxCanvasJS measures how much of what is on screen is painted rather than
// written: the viewport area under visible <canvas> elements, as a share of
// the viewport.
//
// It exists because of two sessions on 5 ก.ย. A model typed twelve sections
// into a Google Doc and two rows into a Google Sheet, read the pages back, saw
// its text in the DOM, and told the user the work was done. Both documents
// were empty. Docs and Sheets keep the document in their own memory and paint
// it onto canvas; the DOM around the canvas is menus, rulers, and the one
// hidden element that catches the keyboard. A read of that DOM is a read of
// the toolbar, and nothing in it said so — so the model reported on a document
// it had never seen, in the same words it uses for one it has.
//
// A measurement rather than a list of hostnames, for the reason
// visionModelMarkers gives: every editor that paints is one this file has not
// heard of yet. Overlapping canvases can sum past the viewport, so the share
// is capped at 1 — the question is "is this page mostly paint", not "how many
// layers".
//
// canvasAppShare is where "some canvas on the page" becomes "this page paints
// its content". A fifth of the viewport: Docs with its side panel open, Sheets
// with Gemini's panel taking half the window, both clear it; a page with a
// chart in a corner does not, and for that page the text really is the page.
const canvasAppShare = 0.2

const aetoxCanvasJS = `
  function aetoxCanvas(roots){
    var vw=window.innerWidth||0,vh=window.innerHeight||0;
    if(vw<=0||vh<=0)return{share:0,count:0};
    var area=0,count=0;
    for(var i=0;i<roots.length;i++){
      var cs;
      try{cs=roots[i].querySelectorAll('canvas');}catch(e){continue;}
      for(var j=0;j<cs.length;j++){
        var r=cs[j].getBoundingClientRect();
        var w=Math.min(r.right,vw)-Math.max(r.left,0),h=Math.min(r.bottom,vh)-Math.max(r.top,0);
        if(w<=0||h<=0)continue;
        count++;area+=w*h;
      }
    }
    return{share:Math.min(1,area/(vw*vh)),count:count};
  }
`

// interactiveSel is what counts as a control, in one place.
//
// There were two copies of this string until 8 ก.ย. — textScript's and
// aetoxCountInteractive's — which is the second-place-answering-one-question
// this file's neighbours keep warning about. Widening one and not the other
// would have left the page's own count of controls answering the old question
// while `read` answered the new one.
//
// **The ARIA roles below are the change, and they are not a nicety.** The list
// was the HTML controls plus role=button and role=link, which is every control
// a page written in HTML has and about half the controls a page written in a
// component framework has. Angular Material — which is Google's entire product
// line, so Docs, Sheets, and NotebookLM — builds a tab as `<div role="tab">`
// wrapping a `<span class="mdc-tab__text-label">`. Neither matched, so the tab
// was in no read, had no ref, and could not be clicked by name.
//
// What that cost, from the owner's log of 8 ก.ย. 14:37: `click find:"เว็บไซต์"`
// refused (no such text), `read` (the control is not in it), `click
// find:"แหล่งข้อมูล"` refused with ten non-interactive text nodes listed —
// including `span.mdc-tab__text-label "แหล่งข้อมูล"`, the label of the very tab
// that was wanted — then `capture`, then `click x:114,y:119` measured off the
// picture by eye. Five calls and a coordinate to press a tab, and the repeat
// detector filed the failure as a tool that keeps breaking the same way.
//
// It is also why `capture marks=true` would NOT have rescued that turn: marks
// draw the refs a read stamped, so they inherit this list's blind spots exactly.
// A control that is in no read is in no picture either.
//
// Everything added is a role that IS a control by definition — something a
// person presses, picks, ticks or types into. Roles that only describe
// structure (tablist, menu, listbox, group) stay out: they contain controls,
// they are not controls, and listing them would put a box round the container
// and its contents both.
const interactiveSel = `a[href],button,input,select,textarea,[contenteditable="true"],` +
	`[role="button"],[role="link"],[role="tab"],[role="checkbox"],[role="radio"],` +
	`[role="switch"],[role="menuitem"],[role="menuitemcheckbox"],[role="menuitemradio"],` +
	`[role="option"],[role="combobox"],[role="textbox"],[role="searchbox"],` +
	`[role="slider"],[role="spinbutton"],[role="treeitem"],svg text`

// textScript reads page text and, in the same pass, tags every visible
// interactive element with a data-aetox-ref so browser_click/browser_type can
// target it later. Refs are reassigned fresh each call.
//
// filter, when non-empty, tags only the elements whose text contains it,
// case-insensitively. It exists because the cap below is a real ceiling on a
// real page and not a theoretical one: 150 refs are assigned in DOM order, and
// on most sites the first 150 elements are the nav and the sidebar, so the
// control the model is actually looking for can sit past the cap on every read
// no matter how many times it reads. Raising the cap would buy that with
// context on every read; a filter is paid only by the read that needs it.
//
// Both caps now count past themselves. What stopped is said out loud in the
// tool output rather than left for the model to discover by finding nothing —
// the same rule skill_view's end marker was written for
// (internal/skill/progressive.go), applied to the tool that was breaking it.
func textScript(token, filter string) string {
	return fmt.Sprintf(`(function(){%s%s%s%s
  var want=%s.trim().toLowerCase();
  var scan=aetoxScan();
  var roots=scan.roots;
  var act=aetoxDeepActive();
  /* svg text is here for the editors that draw their document as SVG —
     Google Slides, diagrams — where a title placeholder is a <text> node
     and nothing else on the page names it. innerText is an HTMLElement
     property, so the label below falls through to textContent for these. */
  var sel='%s';
  /* Stale refs from the previous read are cleared first, in every root. A
     filtered read tags far fewer nodes than an unfiltered one, so without this
     a ref could resolve to a node the last read tagged and this one did not. */
  for(var r=0;r<roots.length;r++){
    var stale;
    try{stale=roots[r].querySelectorAll('[data-aetox-ref]');}catch(e){continue;}
    for(var q=0;q<stale.length;q++)stale[q].removeAttribute('data-aetox-ref');
  }
  var out=[],elTotal=0;
  for(var r2=0;r2<roots.length;r2++){
    var els;
    try{els=roots[r2].querySelectorAll(sel);}catch(e){continue;}
    for(var i=0;i<els.length;i++){
      var el=els[i];
      var rect=el.getBoundingClientRect();
      if(rect.width<=0||rect.height<=0)continue;
      var txt=(el.innerText||el.value||el.getAttribute('aria-label')||el.getAttribute('placeholder')||(el instanceof SVGElement?el.textContent:'')||'').trim().replace(/\s+/g,' ').slice(0,80);
      if(el instanceof SVGElement&&!txt)continue;
      if(el.tagName==='SELECT'){
        var op=[];
        for(var k=0;k<el.options.length&&k<8;k++)op.push(el.options[k].text.trim());
        txt=((txt?txt+' ':'')+'[options: '+op.join(' | ')+']').slice(0,200);
      }
      if(want&&txt.toLowerCase().indexOf(want)<0)continue;
      elTotal++;
      if(out.length>=150)continue;
      var ref=out.length+1;
      el.setAttribute('data-aetox-ref',String(ref));
      var hid=(el.tagName==="INPUT"||el.tagName==="TEXTAREA")&&aetoxTypeMode(el)==="keys";
      out.push({ref:ref,tag:el.tagName.toLowerCase(),role:el.getAttribute('role')||'',text:txt,focused:el===act,hidden:hid});
    }
  }
  var imgs=[],seenSrc={},imgTotal=0;
  for(var r3=0;r3<roots.length;r3++){
    var imels;
    try{imels=roots[r3].querySelectorAll('img[src]');}catch(e){continue;}
    for(var j=0;j<imels.length;j++){
      var im=imels[j];
      var ir=im.getBoundingClientRect();
      if(ir.width<64||ir.height<64)continue; /* skip icons/trackers */
      var src=im.currentSrc||im.src||'';
      if(!src||src.indexOf('data:')===0||seenSrc[src])continue;
      seenSrc[src]=1;
      imgTotal++;
      if(imgs.length>=20)continue;
      imgs.push({src:src.slice(0,600),alt:(im.alt||'').trim().replace(/\s+/g,' ').slice(0,120)});
    }
  }
  var text=aetoxText();
  var cv=aetoxCanvas(roots);
  %s(JSON.stringify({__aetox:"text",token:%q,title:document.title,url:location.href,text:text.slice(0,200000),elements:out,images:imgs,elementsTotal:elTotal,imagesTotal:imgTotal,frames:scan.blocked,canvasShare:cv.share,canvasCount:cv.count}));
})()`, aetoxScanJS, aetoxTextJS, aetoxCanvasJS, aetoxTypeModeJS, mustJSONString(filter), interactiveSel, bridgePost, token)
}

// browserActResult is what a click or a type says about the element it was
// aimed at. Reported before the action, never after: see aetoxActJS.
type browserActResult struct {
	Found bool
	Ref   int
	Tag   string
	Label string
	// Set by a type only. Mode is "value" when the page script set the
	// element's value itself, "keys" when the text still has to go in as
	// keystrokes from the engine (browser_keys.go). Active says whether
	// focusing the element left something editable focused; CX, CY is the
	// element's centre in CSS pixels, for the real click that takes focus when
	// it did not. CanvasShare is aetoxCanvas's answer for this page.
	Mode        string
	Active      bool
	CX, CY      float64
	CanvasShare float64
	// FocusBefore and Focus name the element the page had focused before the
	// type and after it took focus — "textarea#x.cls[aria-label](editable)".
	// The second is where the keystrokes will go, which is a fact the model
	// should have: on a page whose editor is a hidden sink, the element the
	// model aimed at and the element that gets the keys are not the same.
	FocusBefore, Focus string
	// Kept is true when the target was a hidden proxy and the page already
	// held the keyboard in an editor, so the script left focus alone and the
	// keys went to that editor. See typeScript.
	Kept bool
	// Mouse is true when a click's target is not an HTML element and the
	// script did not click it: the click still has to happen, as a real
	// pointer at CX, CY. See clickScript.
	Mouse bool
	// The page's answer about a point or about itself: what is under the
	// point (aetoxDescribe), the viewport in CSS px with its pixel ratio and
	// scroll offset, how many interactive elements it holds, the document
	// title, and for a file input whether it is one and what it accepts.
	Under   string
	VW, VH  int
	DPR     float64
	ScrollX int
	ScrollY int
	Count   int
	Matches []string
	// The refs refMarkScript drew onto the page, with Count holding how many
	// were in the frame before its cap.
	Marks     []int
	Title     string
	URL       string
	FileInput bool
	Multiple  bool
	Accept    string
}

// aetoxActJS is the one sentence a click and a type both have to say: whether
// the ref they were given matched anything on this page.
//
// It exists because of a real turn, on 2026-08-22, recorded in tool_runs. The
// model sent {"action":"click","ref":"1"}, the quoted number came through the
// desktop's own intArg as 0, and clickScript's `if(!el)return` swallowed the
// miss without a sound — so the tool answered "คลิก ref 0 แล้ว", ok=1, and the
// page had not been touched. The model read the page, saw nothing had changed,
// clicked again, reopened the page, clicked again: six rounds of a loop whose
// exit was a sentence nobody was saying. The type coercion is fixed too, and it
// is the smaller half. A tool that reports success for work it did not do turns
// every bug upstream of it into a loop.
//
// **Reported before the click, not after.** A click can navigate, and a
// navigation tears down the document that would have sent the message. Sending
// first means the report is about what was aimed at rather than about what
// happened, which is the half that was missing — whether the action landed at
// all is what `read` is for, and the guidance already says to read afterwards.
func aetoxActJS() string {
	return `
  function aetoxReport(token,ref,el,extra){
    var m={__aetox:"act",token:token,url:location.href,ref:ref,
      found:!!el,
      tag:el?el.tagName.toLowerCase():"",
      label:el?String(el.innerText||el.value||el.getAttribute('aria-label')||'').trim().replace(/\s+/g,' ').slice(0,80):""};
    if(extra)for(var k in extra)m[k]=extra[k];
    ` + bridgePost + `(JSON.stringify(m));
  }
`
}

// clickScript clicks the element tagged with the given ref (see textScript).
//
// aetoxFind rather than document.querySelector, because textScript now tags
// nodes inside shadow roots and same-origin frames as well. A ref handed out by
// a read this tool could not then act on would be worse than not handing it out.
//
// An element that is not HTML — SVG text in a Slides deck, a shape in a
// diagram — has no click() to call and answers only to a real pointer, so
// the report says so (mouse:true) with the element's centre, and Go clicks
// there through the engine instead (clickByMouse). Reported before the
// scroll for the reason above; the centre is measured after it, which is why
// the second report exists: it carries the coordinates the click needs,
// and only on the path where nothing can navigate before it is sent.
func clickScript(token string, ref int) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s
  var el=aetoxFind(%d);
  var mouse=!!el&&!(el instanceof HTMLElement);
  if(mouse){
    try{el.scrollIntoView({block:"center"});}catch(e){}
    var p=aetoxPagePoint(el);
    aetoxReport(%s,%d,el,{mouse:true,cx:p.x,cy:p.y});
    return;
  }
  aetoxReport(%s,%d,el);
  if(!el)return;
  el.scrollIntoView({block:"center"});
  el.click();
})()`, aetoxScanJS, aetoxActJS(), aetoxPointJS, ref, string(tok), ref, string(tok), ref)
}

// aetoxPointJS is how a place on the page is measured and named.
//
// aetoxPagePoint is an element's centre in CSS pixels of the TOP viewport —
// getBoundingClientRect is relative to the element's own frame, so for an
// element inside a same-origin iframe the frame's own rect is added, all the
// way up. The engine's pointer events take top-viewport coordinates and
// nothing else, so a centre measured any other way clicks the wrong place by
// exactly the frame's offset.
//
// aetoxUnder is the reverse: the deepest element at a point, descending into
// shadow roots and same-origin frames the way a pointer would. It skips
// Aetox's own overlays because those are pointer-events:none, which is one
// more reason they have to be.
//
// aetoxView is the viewport as the model needs to know it to aim: its size in
// CSS pixels (what a capture's pixels map onto), its pixel ratio (why the
// capture has more pixels than that), and how far it is scrolled.
const aetoxPointJS = `
  function aetoxPagePoint(el){
    var r=el.getBoundingClientRect();
    var x=r.left+r.width/2,y=r.top+r.height/2;
    var d=el.ownerDocument,guard=0;
    while(d&&d!==document&&guard++<10){
      var f=null;
      try{f=d.defaultView&&d.defaultView.frameElement;}catch(e){f=null;}
      if(!f)break;
      var fr=f.getBoundingClientRect();
      x+=fr.left;y+=fr.top;
      d=f.ownerDocument;
    }
    return{x:x,y:y};
  }
  function aetoxUnder(x,y){
    var el=null;
    try{el=document.elementFromPoint(x,y);}catch(e){return null;}
    var guard=0;
    while(el&&guard++<10){
      var inner=null;
      if(el.shadowRoot&&el.shadowRoot.elementFromPoint){
        try{inner=el.shadowRoot.elementFromPoint(x,y);}catch(e){inner=null;}
        if(inner===el)inner=null;
      }else if(el.tagName==="IFRAME"){
        var doc=null;
        try{doc=el.contentDocument;}catch(e){doc=null;}
        if(doc){var fr=el.getBoundingClientRect();try{inner=doc.elementFromPoint(x-fr.left,y-fr.top);}catch(e){inner=null;}}
      }
      if(!inner)break;
      el=inner;
    }
    return el;
  }
  function aetoxView(){
    return{vw:window.innerWidth||0,vh:window.innerHeight||0,dpr:window.devicePixelRatio||1,scrollX:Math.round(window.scrollX||0),scrollY:Math.round(window.scrollY||0)};
  }
  function aetoxCountInteractive(){
    var roots=aetoxScan().roots,n=0;
    var sel='` + interactiveSel + `';
    for(var i=0;i<roots.length;i++){try{n+=roots[i].querySelectorAll(sel).length;}catch(e){}}
    return n;
  }
`

// pointScript measures a ref: where its centre is in the top viewport,
// whether it is HTML (a script can click it) or not (only a pointer can),
// and the viewport it sits in. Acts on nothing; the gesture is the engine's.
func pointScript(token string, ref int) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s%s%s
  var el=aetoxFind(%d),extra=aetoxView();
  if(el){
    try{el.scrollIntoView({block:"center",behavior:"instant"});}catch(e){}
    var p=aetoxPagePoint(el);
    extra.cx=p.x;extra.cy=p.y;
    extra.mouse=!(el instanceof HTMLElement);
    extra.under=aetoxDescribe(el);
  }
  extra.canvasShare=aetoxCanvas(aetoxScan().roots).share;
  aetoxReport(%s,%d,el,extra);
})()`, aetoxScanJS, aetoxActJS(), aetoxCanvasJS, aetoxTypeModeJS, aetoxPointJS, ref, string(tok), ref)
}

// underScript names what is at a viewport point, and refuses a point outside
// the viewport before anything is pressed there: found is "inside", not
// "something was hit", because a pointer over an empty patch of canvas is a
// perfectly good target and an off-screen point is not one at all.
func underScript(token string, x, y float64) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s%s%s
  var x=%g,y=%g,extra=aetoxView();
  var inside=x>=0&&y>=0&&x<=extra.vw&&y<=extra.vh;
  var el=inside?aetoxUnder(x,y):null;
  extra.found=inside;extra.cx=x;extra.cy=y;
  extra.under=el?aetoxDescribe(el):"";
  extra.canvasShare=aetoxCanvas(aetoxScan().roots).share;
  aetoxReport(%s,0,el,extra);
})()`, aetoxScanJS, aetoxActJS(), aetoxCanvasJS, aetoxTypeModeJS, aetoxPointJS, x, y, string(tok))
}

// focusScript says where the page's keyboard goes right now — the one fact a
// `key` or a target-less `type` has to name, because keystrokes go to focus
// and not to a ref (§226).
func focusScript(token string) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s%s%s
  var a=aetoxDeepActive(),extra=aetoxView();
  extra.focus=aetoxDescribe(a);
  extra.active=aetoxActiveIsEditable();
  extra.canvasShare=aetoxCanvas(aetoxScan().roots).share;
  aetoxReport(%s,0,a,extra);
})()`, aetoxScanJS, aetoxActJS(), aetoxCanvasJS, aetoxTypeModeJS, aetoxPointJS, string(tok))
}

// viewportScript is the viewport alone, for capture: what its pixels map to.
func viewportScript(token string) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s
  aetoxReport(%s,0,null,aetoxView());
})()`, aetoxScanJS, aetoxActJS(), aetoxPointJS, string(tok))
}

// stateScript is the page in four numbers and a name, cheap enough to ask
// before and after every action: URL and title, how many interactive
// elements it has, and where its focus is. The difference between two of
// these is the change note an action hands back in place of "ใช้ read ดูผล"
// — measured on Playwright's MCP as the single change that let a model chain
// a dozen actions without stopping to look.
func stateScript(token string) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s%s
  var extra=aetoxView();
  extra.title=document.title;
  extra.count=aetoxCountInteractive();
  extra.focus=aetoxDescribe(aetoxDeepActive());
  extra.found=true;
  aetoxReport(%s,0,null,extra);
})()`, aetoxScanJS, aetoxActJS(), aetoxTypeModeJS, aetoxPointJS, string(tok))
}

// findTextScript is `find` for text that is not a control: a heading, a word
// in a paragraph, a label with nothing to press. The interactive read is
// tried first (resolveTarget); this is what runs when it found nothing, so a
// hover over "hover me" or a double-click on a word works the way it would
// for a person, who never asked whether the thing under the pointer was a
// button. The owner's first live run (6 ก.ย.) failed both of those.
//
// The deepest elements whose text contains the words are the matches — a
// paragraph inside a section inside main all contain it, and the paragraph
// is the one meant. Exactly one is tagged as ref 1 so the point script can
// find it; several are described back so the model can choose by ref after
// a read, or by a longer text. Aetox's own overlays are skipped, since they
// are the one thing on the page that is not the page.
func findTextScript(token, text string) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s%s
  var want=%s.trim().toLowerCase();
  var roots=aetoxScan().roots,hits=[],seen=0;
  for(var r=0;r<roots.length;r++){
    var stale;
    try{stale=roots[r].querySelectorAll('[data-aetox-ref]');}catch(e){continue;}
    for(var q=0;q<stale.length;q++)stale[q].removeAttribute('data-aetox-ref');
  }
  if(want){
    for(var r2=0;r2<roots.length&&seen<AETOX_BUDGET;r2++){
      var all;
      try{all=roots[r2].querySelectorAll('*');}catch(e){continue;}
      for(var i=0;i<all.length&&seen<AETOX_BUDGET;i++){
        seen++;
        var el=all[i],tag=el.tagName;
        if(tag==="SCRIPT"||tag==="STYLE"||tag==="NOSCRIPT"||tag==="HTML"||tag==="BODY"||tag==="HEAD")continue;
        if(el.id&&String(el.id).indexOf("__aetox")===0)continue;
        var t=String(el.textContent||"");
        if(t.toLowerCase().indexOf(want)<0)continue;
        var rc=el.getBoundingClientRect();
        if(rc.width<=0||rc.height<=0)continue;
        hits.push(el);
      }
    }
  }
  var deep=[];
  for(var a=0;a<hits.length;a++){
    var inner=false;
    for(var b=0;b<hits.length;b++){if(a!==b&&hits[a].contains(hits[b])){inner=true;break;}}
    if(!inner)deep.push(hits[a]);
  }
  var extra={count:deep.length,matches:[]};
  for(var d=0;d<deep.length&&d<12;d++){
    var s=String(deep[d].innerText||deep[d].textContent||"").trim().replace(/\s+/g," ").slice(0,60);
    extra.matches.push(aetoxDescribe(deep[d])+' "'+s+'"');
  }
  var one=deep.length===1?deep[0]:null;
  if(one)one.setAttribute('data-aetox-ref','1');
  aetoxReport(%s,0,one,extra);
})()`, aetoxScanJS, aetoxActJS(), aetoxTypeModeJS, aetoxPointJS, mustJSONString(text), string(tok))
}

// fileInputScript says whether a ref is a file input, and what it takes.
// Acts on nothing: the file goes in through the engine (DOM.setFileInputFiles),
// which is the only door a page lets a file through.
func fileInputScript(token string, ref int) string {
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s
  var el=aetoxFind(%d),extra={};
  if(el){
    extra.fileInput=(el.tagName==="INPUT"&&String(el.type).toLowerCase()==="file");
    extra.multiple=!!el.multiple;
    extra.accept=String(el.getAttribute("accept")||"");
    try{el.scrollIntoView({block:"center",behavior:"instant"});}catch(e){}
    var p=aetoxPagePoint(el);
    extra.cx=p.x;extra.cy=p.y;
  }
  aetoxReport(%s,%d,el,extra);
})()`, aetoxScanJS, aetoxActJS(), aetoxPointJS, ref, string(tok), ref)
}

// typeScript is the page half of a type, and it decides which of two ways the
// text goes in.
//
// **value** — a visible input, textarea or select. The value is set through
// the native setter (so React/Vue-controlled inputs pick it up) and
// input+change are fired; a SELECT matches the text against option value or
// label instead. enter additionally presses Enter — synthetic keydown first
// (skipped requestSubmit if the page preventDefault'ed it), then the form's
// requestSubmit, because untrusted KeyboardEvents never trigger the browser's
// own implicit submission. This was the whole of the tool until 5 ก.ย.
//
// **keys** — a contenteditable, or an input the page keeps hidden as its
// keyboard proxy. The script only focuses the element and reports; the text is
// then typed by the engine as real keystrokes (browser_keys.go). It has to be,
// because of what the value path did to Google Docs and Google Sheets that day:
// `el.textContent=val` on Docs' editor container — a contenteditable div whose
// children are the canvas tiles — replaced the tiles with the text, which the
// next `read` then found in the DOM and reported back as the document. Sheets'
// hidden textarea took the value and Sheets, which listens for keys and not
// for values, never looked at it. Both tools said "พิมพ์แล้ว", both reads
// agreed, and the user opened two empty documents. An editor that keeps its
// document in its own memory accepts exactly one kind of input, and a DOM
// write is not it.
//
// The mode is decided here, on the live element, and not by hostname: a
// hidden proxy input is a shape (opacity 0, a few pixels, parked off-screen),
// and the shape is what every such editor shares. The report carries the mode
// so Go knows whether it still has work to do, and carries whether focus
// actually landed on something editable, because a proxy that the page
// focuses for itself on a real click will not be focused by `el.focus()` from
// a script — Go then clicks for real at cx, cy first.
func typeScript(token string, ref int, text string, enter bool) string {
	encoded, _ := json.Marshal(text)
	enterJS := ""
	if enter {
		enterJS = `
  var ke={key:"Enter",code:"Enter",keyCode:13,which:13,bubbles:true,cancelable:true};
  var notHandled=el.dispatchEvent(new KeyboardEvent("keydown",ke));
  el.dispatchEvent(new KeyboardEvent("keyup",ke));
  if(notHandled&&el.form&&typeof el.form.requestSubmit==="function"){el.form.requestSubmit();}`
	}
	tok, _ := json.Marshal(token)
	return fmt.Sprintf(`(function(){%s%s%s%s
  var el=aetoxFind(%d);
  var mode=aetoxTypeMode(el),extra={mode:mode};
  if(el&&mode==="keys"){
    /* Focus before the report, and only here: focusing cannot navigate, and
       the report has to say whether focus landed. The value path reports
       first and acts after, as click does. */
    try{el.scrollIntoView({block:"center"});}catch(e){}
    extra.focusBefore=aetoxDescribe(aetoxDeepActive());
    /* A hidden input is the page's keyboard proxy, and a page that already
       holds the keyboard somewhere editable has told us where its keys go.
       Google Sheets, 5 ก.ย.: the cell editor was focused, the model aimed at
       the one textarea the read listed, focusing that moved the keyboard into
       an off-screen Trix editor, and the sheet got nothing. So a proxy target
       does not take focus away from an editor that has it. A contenteditable
       target always takes it: the model named the editor itself. */
    var proxy=(el.tagName==="INPUT"||el.tagName==="TEXTAREA");
    if(proxy&&aetoxActiveIsEditable()&&aetoxDeepActive()!==el){extra.kept=true;}
    else{aetoxTakeFocus(el);}
    var rk=el.getBoundingClientRect();
    extra.active=aetoxActiveIsEditable();
    extra.focus=aetoxDescribe(aetoxDeepActive());
    extra.cx=rk.left+rk.width/2;extra.cy=rk.top+rk.height/2;
    extra.canvasShare=aetoxCanvas(aetoxScan().roots).share;
  }
  aetoxReport(%s,%d,el,extra);
  if(!el||mode==="keys")return;
  el.focus();
  var val=%s;
  if(el.tagName==="SELECT"){
    var want=val.trim().toLowerCase();
    for(var i=0;i<el.options.length;i++){
      var o=el.options[i];
      if(o.value.toLowerCase()===want||o.text.trim().toLowerCase()===want){
        Object.getOwnPropertyDescriptor(window.HTMLSelectElement.prototype,"value").set.call(el,o.value);
        break;
      }
    }
  } else if(el.tagName==="INPUT"||el.tagName==="TEXTAREA"){
    var proto=el.tagName==="TEXTAREA"?window.HTMLTextAreaElement.prototype:window.HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto,"value").set.call(el,val);
  } else {
    el.textContent=val;
  }
  el.dispatchEvent(new Event("input",{bubbles:true}));
  el.dispatchEvent(new Event("change",{bubbles:true}));%s
})()`, aetoxScanJS, aetoxActJS(), aetoxCanvasJS, aetoxTypeModeJS, ref, string(tok), ref, encoded, enterJS)
}

// aetoxTypeModeJS is the decision typeScript's comment describes, plus the two
// things the keys path needs from the page: focus taken the way a script can
// take it, and an honest answer about whether that worked.
//
// aetoxActiveIsEditable follows document.activeElement down through shadow
// roots and same-origin frames, because that is where the editable really is
// in the editors this exists for: Docs' keyboard target is a contenteditable
// inside an about:blank iframe, and focusing the visible editor is what makes
// Docs focus that. A cross-origin frame stops the walk, and an IFRAME is not
// editable, so the answer is "no" and Go clicks for real.
//
// aetoxTakeFocus puts the caret at the end of what is there, for a plain
// contenteditable and for an input, so the keystrokes that follow append
// rather than prepend — the reading a model has of "type this into the
// editor". Not for an element with canvas inside it: a DOM selection inside
// Docs' tile container means nothing to Docs and could mean something wrong.
const aetoxTypeModeJS = `
  function aetoxTypeMode(el){
    if(!el)return "";
    var tag=el.tagName;
    if(tag==="SELECT")return "value";
    if(tag!=="INPUT"&&tag!=="TEXTAREA")return "keys";
    var r=el.getBoundingClientRect(),cs=getComputedStyle(el);
    var hidden=parseFloat(cs.opacity)===0||cs.visibility==="hidden"||r.width<4||r.height<4||r.right<0||r.bottom<0;
    return hidden?"keys":"value";
  }
  function aetoxTakeFocus(el){
    try{el.focus();}catch(e){}
    try{
      if(el.tagName==="INPUT"||el.tagName==="TEXTAREA"){var n=el.value.length;el.setSelectionRange(n,n);return;}
      if(el.querySelector('canvas'))return;
      var d=el.ownerDocument,s=d.getSelection();
      if(!s)return;
      var rg=d.createRange();rg.selectNodeContents(el);rg.collapse(false);
      s.removeAllRanges();s.addRange(rg);
    }catch(e){}
  }
  function aetoxDeepActive(){
    var a=document.activeElement,guard=0;
    while(a&&guard++<20){
      var inner=null;
      if(a.shadowRoot&&a.shadowRoot.activeElement)inner=a.shadowRoot.activeElement;
      else if(a.tagName==="IFRAME"){try{inner=a.contentDocument?a.contentDocument.activeElement:null;}catch(e){inner=null;}}
      if(!inner||inner===a)break;
      a=inner;
    }
    return a;
  }
  function aetoxActiveIsEditable(){
    var a=aetoxDeepActive();
    if(!a||a===document.body)return false;
    if(a.tagName==="INPUT"||a.tagName==="TEXTAREA")return true;
    return !!a.isContentEditable;
  }
  function aetoxDescribe(a){
    if(!a||a===document.body)return "body";
    var s=a.tagName.toLowerCase();
    try{
      if(a.id)s+="#"+a.id;
      var cls=String(a.getAttribute('class')||'').trim().split(/\s+/).filter(Boolean).slice(0,2);
      if(cls.length)s+="."+cls.join(".");
      var al=a.getAttribute('aria-label');
      if(al)s+="["+al.slice(0,40)+"]";
      if(a.isContentEditable)s+="(editable)";
    }catch(e){}
    return s;
  }
`

// sameOrigin reports whether a and b share a scheme+host — used to check a
// page's claimed URL against its real origin as reported by the engine.
func sameOrigin(a, b string) bool {
	ua, err1 := url.Parse(a)
	ub, err2 := url.Parse(b)
	if err1 != nil || err2 != nil || ua.Scheme == "" || ua.Scheme != ub.Scheme {
		return false
	}
	// file: URLs have no host, so the host check below would reject every
	// local page. The check's purpose is stopping a page from spoofing the
	// address bar as a trusted SITE — a file page claiming some other local
	// path can't do that, so scheme match is enough for file↔file.
	if ua.Scheme == "file" {
		return true
	}
	return ua.Host != "" && ua.Host == ub.Host
}

// newMessageToken mints a per-request nonce for BrowserGetText, so a stray or
// forged "text" message can't be mistaken for the response to a specific call.
func newMessageToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// browserTab is the portable bookkeeping for one tab: latches, meta, zoom,
// visibility, the pick in progress. Everything about a tab EXCEPT the webview.
//
// The webview used to be a field here, and that is what made §127.8 possible:
// anything holding a *browserTab could reach the engine directly, on whatever
// goroutine it happened to be on, and WebView2 answers that by refusing the
// call silently. The views now live in browserHost.views and are handed out
// only inside onTab, which runs on the thread that owns them. There is no
// second way to get one, so the call that broke does not compile.
type browserTab struct {
	// One navigation's completion latch, replaced on every new navigation so a
	// reused tab can be awaited again. Guarded because navCompleted closes it
	// from the host thread while a tool call waits on it: without the mutex,
	// re-arming races the close and a waiter can hold the previous latch.
	navMu   sync.Mutex
	navDone chan struct{} // closed after the current navigation completes
	navOnce *sync.Once

	metaMu sync.Mutex
	title  string
	url    string
	// navOK is the last completed navigation's real outcome. False means the
	// window is showing the engine's own error page — a state that used to be
	// indistinguishable from a loaded page, since navigation-completed fires
	// either way.
	navOK bool

	visMu  sync.Mutex
	hidden bool // BrowserSetVisible(false); nav-completed re-glue must not surface hidden tabs

	// fbMu guards fallback: the other scheme to try, once, if the navigation
	// this tab was just given fails. Set only from the address bar, and only
	// for a scheme Aetox chose rather than one the user typed (address.go).
	//
	// Taken rather than read, because a fallback is spent the moment it is
	// used: without that, a page that fails, falls back, and fails again would
	// bounce between the two schemes for as long as the tab is open.
	fbMu     sync.Mutex
	fallback string

	zoomMu sync.Mutex
	zoom   float64 // device-size emulation (see BrowserSetZoom); 0 = never set

	// Where the agent's cursor is, in CSS pixels of the viewport, and whether
	// it has been anywhere yet. The sprite itself lives in the page and dies
	// with the document; this is what draws it back on the next one, at the
	// spot the user last saw it. See browser_marks.go, aetoxCursorJS.
	curMu    sync.Mutex
	curX     float64
	curY     float64
	curKnown bool

	textMu    sync.Mutex
	textCh    chan browserSnapshot
	textToken string // token BrowserGetText is currently waiting on; empty = none pending

	// What the last read left behind for click and type to aim at. Every read
	// reassigns refs from 1 and strips the ones before it, so "ref 11" means
	// nothing except in the numbering of the read that produced it — and a
	// FILTERED read tags only its matches, which is a short list with its own
	// numbers on a page that never changed.
	//
	// Kept because the tool could not otherwise say why a ref missed. It used
	// to answer every miss with "refs expire when the page changes", which is
	// true and was the wrong cause: the page was the same, a narrower read had
	// simply replaced the numbering underneath it.
	refMu     sync.Mutex
	refCount  int    // how many elements the last read tagged
	refFilter string // the filter that read used; "" for none
	refRead   bool   // whether any read has happened on this page yet

	pickMu sync.Mutex
	// pickToken is the token the live point-at-the-page mode will answer with;
	// empty = no mode running. Unlike textToken there is no channel waiting on
	// it — a pick arrives as an event whenever the user gets round to pointing,
	// or never. See browser_pick.go.
	pickToken string

	waitMu    sync.Mutex
	waitCh    chan bool
	waitToken string

	logMu    sync.Mutex
	logCh    chan browserLogReport
	logToken string // token browserLog is waiting on; empty = none pending

	actMu    sync.Mutex
	actCh    chan browserActResult
	actToken string // token a click/type is waiting on; empty = none pending

	// dlgMu guards what the page's dialogs said since anyone last looked. A
	// dialog cannot block here — see dialogScript — so the only way the agent
	// ever learns one happened is that the next answer it gets mentions it.
	dlgMu   sync.Mutex
	dialogs []string

	engMu sync.Mutex
	// engErr is the engine's last complaint about a call made SINCE this tab's
	// current navigation was armed. Cleared by armNavigation, so it is always
	// about the navigation being waited on and never a leftover from the last
	// one. See tabCallbacks.onEngineError for why it exists at all.
	engErr error
	// dead is why the engine behind this tab is gone, or nil while it lives.
	// Set by engineGone, cleared by a revive that put a new engine in place.
	// A tab that is dead answers nothing; a tab that was dead and is not now
	// has a note to give (reviveNote). Guarded by engMu with engErr, because
	// they are read together by the waiter that decides what to say.
	dead error
	// reviveNote is told once, to the next answer the agent gets from this
	// tab: its page was put back by a fresh engine, so whatever was typed and
	// not sent, scrolled to, or picked is gone. Same shape as dialogs.
	reviveNote string
	// revives is when each revive happened, so a tab whose engine cannot stay
	// alive is given up rather than rebuilt forever. See reviveAllowed.
	revives []time.Time
	// gen counts the engines this tab has had. Each set of callbacks carries
	// the generation it was made for, so a complaint from an engine that has
	// already been replaced cannot be mistaken for the current one dying — a
	// late refusal from the old webview would otherwise revive a live tab.
	gen int

	// wantMu guards where this tab was last asked to go and how big it was —
	// everything a fresh engine needs to put the same page back in the same
	// place. Written by every navigation and every bounds change; read only
	// by revive. Kept on the tab rather than asked of the engine because the
	// engine that knew is the one that is gone.
	wantMu    sync.Mutex
	wantURL   string
	bounds    [4]int
	hasBounds bool
	// detached: this tab is a window of its own now and no longer part of any
	// desk. Guarded by wantMu with the two above because it is the same kind
	// of fact — what this tab should be, as opposed to what its engine is.
	detached bool

	// shotMu guards what this tab's last capture looked like.
	//
	// A capture used to hand back a picture and nothing else, which meant two
	// captures of an unchanged page were two identical pictures with nothing
	// saying they were identical. A model reading the second one cannot tell
	// "the edit did not land" from "nothing here was going to change", and it
	// reads it as the first: on 28 ส.ค. three of one chat's thirteen captures
	// came back byte-for-byte identical, and each one sent the agent round to
	// edit again — the deck's opening slide was rewritten four times over a
	// picture that had never changed.
	//
	// The sum is of the PNG bytes and not of the pixels. That can only ever
	// MISS a duplicate, never invent one: an encoder that wrote the same image
	// two ways leaves both captures reported as new, which is exactly what
	// every capture did before this existed. A false "nothing changed" would
	// be the dangerous direction, and this cannot produce one.
	shotMu   sync.Mutex
	shotSum  [sha256.Size]byte
	shotPath string // where the remembered capture was written
	shotHave bool   // whether shotSum means anything yet
	shotSame int    // how many captures in a row have come back identical
}

// lastShot reports whether a capture just taken is byte-for-byte the one this
// tab last handed back, where that one was written, and how many captures in a
// row have now come back identical.
//
// It counts as well as answers, because "the same picture again" and "the same
// picture for the third time" are different facts about how a turn is going,
// and only the tab is in a position to know either.
func (t *browserTab) lastShot(sum [sha256.Size]byte) (where string, inARow int, same bool) {
	t.shotMu.Lock()
	defer t.shotMu.Unlock()
	if !t.shotHave || t.shotSum != sum {
		return "", 0, false
	}
	t.shotSame++
	return t.shotPath, t.shotSame, true
}

// rememberShot makes this capture the one the next is measured against, and
// ends whatever run of identical ones came before it.
func (t *browserTab) rememberShot(sum [sha256.Size]byte, where string) {
	t.shotMu.Lock()
	defer t.shotMu.Unlock()
	t.shotSum, t.shotPath, t.shotHave, t.shotSame = sum, where, true, 0
}

// noteEngineError records what the engine said. Called from whatever thread the
// engine chose; the mutex is the whole of the synchronisation.
func (t *browserTab) noteEngineError(err error) {
	if err == nil {
		return
	}
	t.engMu.Lock()
	t.engErr = err
	t.engMu.Unlock()
}

func (t *browserTab) engineError() error {
	t.engMu.Lock()
	defer t.engMu.Unlock()
	return t.engErr
}

// remember records the page this tab was asked for, so a revive can ask for
// it again. Every navigation goes through goTo, which calls this; open()
// calls it for the first page, which the backend navigates to itself.
func (t *browserTab) remember(url string) {
	t.wantMu.Lock()
	defer t.wantMu.Unlock()
	t.wantURL = url
}

// rememberBounds records where the tab's window is, in the same physical
// pixels BrowserSetBounds is given.
// markDetached records that this tab now lives in a window of its own, and
// detachSize is the rectangle to give that window.
//
// The flag is read by everything that used to assume a tab is a rectangle
// inside the panel — the session teardown that would have closed it, the desk
// event that would have drawn a chip for it, the capture that would have told
// the model to run `tabs select` to un-hide it. None of those is wrong about a
// tab in the strip; all of them are wrong about this one.
func (t *browserTab) markDetached() {
	t.wantMu.Lock()
	defer t.wantMu.Unlock()
	t.detached = true
}

func (t *browserTab) isDetached() bool {
	if t == nil {
		return false
	}
	t.wantMu.Lock()
	defer t.wantMu.Unlock()
	return t.detached
}

// detachSize is the size the window opens at: the rectangle the tab had in the
// panel when it was pulled out, so the page it is showing does not reflow the
// moment it leaves. Zero for a tab that was never placed, which lets the
// platform pick its own default.
func (t *browserTab) detachSize() (w, h int) {
	t.wantMu.Lock()
	defer t.wantMu.Unlock()
	if !t.hasBounds {
		return 0, 0
	}
	return t.bounds[2], t.bounds[3]
}

func (t *browserTab) rememberBounds(x, y, w, h int) {
	t.wantMu.Lock()
	defer t.wantMu.Unlock()
	t.bounds, t.hasBounds = [4]int{x, y, w, h}, true
}

// wanted is what a fresh engine should be given: the last page asked for and
// the last rect the tab was placed at.
func (t *browserTab) wanted() (url string, x, y, w, h int) {
	t.wantMu.Lock()
	defer t.wantMu.Unlock()
	return t.wantURL, t.bounds[0], t.bounds[1], t.bounds[2], t.bounds[3]
}

// goTo is the one way a tab is navigated after it exists: it records the
// destination and then asks the view for it, so revive always knows where
// the tab was going. Called on the host thread like every view call.
func (t *browserTab) goTo(v tabView, url string) {
	t.remember(url)
	v.navigate(url)
}

// markDead records that the engine behind this tab is gone. Reports whether
// this is news — a dead engine tends to say so more than once (ProcessFailed,
// then a refusal for every call already queued), and only the first should
// start a revive.
func (t *browserTab) markDead(why error) bool {
	if why == nil {
		return false
	}
	t.engMu.Lock()
	defer t.engMu.Unlock()
	if t.dead != nil {
		return false
	}
	t.dead = why
	return true
}

// deadWhy is why the engine is gone, or nil while it lives.
func (t *browserTab) deadWhy() error {
	t.engMu.Lock()
	defer t.engMu.Unlock()
	return t.dead
}

func (t *browserTab) isDead() bool { return t.deadWhy() != nil }

// nextGen numbers the engine about to be created for this tab.
func (t *browserTab) nextGen() int {
	t.engMu.Lock()
	defer t.engMu.Unlock()
	t.gen++
	return t.gen
}

func (t *browserTab) generation() int {
	t.engMu.Lock()
	defer t.engMu.Unlock()
	return t.gen
}

// reviveBudget is how many times, within reviveWindow, a tab's engine may be
// replaced before the tab is given up instead. One death is the case this
// exists for; a second within a minute means whatever is killing the engine
// is still there, and a third webview would only be a third corpse.
const (
	reviveBudget = 2
	reviveWindow = time.Minute
)

// reviveAllowed reports whether one more revive fits the budget, and counts
// it if so. Only revive calls this.
func (t *browserTab) reviveAllowed(now time.Time) bool {
	t.engMu.Lock()
	defer t.engMu.Unlock()
	recent := t.revives[:0]
	for _, at := range t.revives {
		if now.Sub(at) < reviveWindow {
			recent = append(recent, at)
		}
	}
	t.revives = recent
	if len(t.revives) >= reviveBudget {
		return false
	}
	t.revives = append(t.revives, now)
	return true
}

// revived clears the death and leaves the sentence the next answer carries.
func (t *browserTab) revived(url string) {
	t.engMu.Lock()
	defer t.engMu.Unlock()
	why := t.dead
	t.dead = nil
	t.reviveNote = fmt.Sprintf("เอนจินเบราว์เซอร์ของแท็บนี้ล่ม (%v) และถูกเปิดขึ้นใหม่แล้ว หน้า %s ถูกโหลดกลับมา แต่สิ่งที่พิมพ์ค้างไว้ ตำแหน่งที่เลื่อนถึง และ ref จากการอ่านครั้งก่อนหายไป อ่านหน้าใหม่ก่อนทำต่อ", why, url)
}

// takeReviveNote hands back the revive sentence and forgets it, so it is said
// once — to the first answer after the engine came back — and not again.
func (t *browserTab) takeReviveNote() string {
	t.engMu.Lock()
	defer t.engMu.Unlock()
	note := t.reviveNote
	t.reviveNote = ""
	return note
}

// giveUp ends whatever wait is on this tab, with the death left in place so
// the waiter names it. Called when a revive could not be done.
func (t *browserTab) giveUp() {
	t.setNavOK(false)
	once, done := t.latch()
	once.Do(func() { close(done) })
}

// navPending reports whether a navigation is being waited on — the latch is
// armed and nothing has closed it yet.
func (t *browserTab) navPending() bool {
	_, done := t.latch()
	select {
	case <-done:
		return false
	default:
		return true
	}
}

func (t *browserTab) isHidden() bool {
	t.visMu.Lock()
	defer t.visMu.Unlock()
	return t.hidden
}

func (t *browserTab) meta() (title, url string) {
	t.metaMu.Lock()
	defer t.metaMu.Unlock()
	return t.title, t.url
}

// noteDialog remembers one dialog the page raised. Bounded, because a page in a
// loop can raise them forever and the point is to tell the agent something
// happened, not to transcribe an attack.
func (t *browserTab) noteDialog(line string) {
	t.dlgMu.Lock()
	defer t.dlgMu.Unlock()
	if len(t.dialogs) < 8 {
		t.dialogs = append(t.dialogs, line)
	}
}

// takeDialogs hands back what the page said and forgets it, so one dialog is
// reported once rather than on every action for the rest of the session.
func (t *browserTab) takeDialogs() []string {
	t.dlgMu.Lock()
	defer t.dlgMu.Unlock()
	out := t.dialogs
	t.dialogs = nil
	return out
}

// dialogNote is the sentence an action appends when the page said something
// while it was working, or "" when it did not.
func (a *App) dialogNote(id AgentTabID) string {
	if a.browsers == nil {
		return ""
	}
	t := a.browsers.tab(string(id))
	if t == nil {
		return ""
	}
	var out string
	// The engine coming back is told here as well: it is the same kind of
	// sentence — something happened to this page while nobody was asking —
	// and the callers that append this are exactly the ones that should carry
	// it. See engineNote for the callers that only want that half.
	if note := t.takeReviveNote(); note != "" {
		out += "\n" + note
	}
	lines := t.takeDialogs()
	if len(lines) == 0 {
		return out
	}
	return out + "\nหน้าเว็บขึ้นกล่องข้อความ:\n" + strings.Join(lines, "\n")
}

// engineNote is the revive sentence alone, for the answers that do not carry
// dialogs — open and read, which are where an agent most often meets a page
// it has to start over on.
func (a *App) engineNote(id AgentTabID) string {
	if a.browsers == nil {
		return ""
	}
	t := a.browsers.tab(string(id))
	if t == nil {
		return ""
	}
	if note := t.takeReviveNote(); note != "" {
		return "\n" + note
	}
	return ""
}

// armNavigation readies the tab for one more navigation, so the caller that
// is about to call view.navigate can await *that* one.
//
// Without it a reused tab answers instantly with the previous page's verdict:
// the latch is closed from the first load and never reopens, so "did the page
// I just asked for arrive" becomes "did the page before it arrive". Called
// before navigate, never after — arming late would drop a completion that beat
// the arm and hang the wait until its timeout.
// latch hands back the current navigation latch, creating one if this tab has
// never had it set. The zero-value tab is a real state — tests build one, and
// a completion can arrive before open() finishes storing its fields — so the
// pair is resolved here rather than assumed to exist at every use.
// zoomFactor is the tab's device-emulation zoom, or 0 when none was ever
// set. nil-safe, because the tab a pointer action aims at can have closed
// between the lookup and the press.
func (t *browserTab) zoomFactor() float64 {
	if t == nil {
		return 0
	}
	t.zoomMu.Lock()
	defer t.zoomMu.Unlock()
	return t.zoom
}

// rememberCursor and cursor are the agent's pointer position across
// documents: written by every pointer action, read by navCompleted to redraw
// the sprite on the page that replaced the one it was drawn on.
func (t *browserTab) rememberCursor(x, y float64) {
	if t == nil {
		return
	}
	t.curMu.Lock()
	t.curX, t.curY, t.curKnown = x, y, true
	t.curMu.Unlock()
}

func (t *browserTab) cursor() (x, y float64, ok bool) {
	if t == nil {
		return 0, 0, false
	}
	t.curMu.Lock()
	defer t.curMu.Unlock()
	return t.curX, t.curY, t.curKnown
}

func (t *browserTab) latch() (*sync.Once, chan struct{}) {
	t.navMu.Lock()
	defer t.navMu.Unlock()
	if t.navDone == nil {
		t.navDone = make(chan struct{})
	}
	if t.navOnce == nil {
		t.navOnce = &sync.Once{}
	}
	return t.navOnce, t.navDone
}

func (t *browserTab) armNavigation() {
	t.setNavOK(false)
	t.navMu.Lock()
	t.navDone, t.navOnce = make(chan struct{}), &sync.Once{}
	t.navMu.Unlock()
	// Cleared here so anything the engine says from now on belongs to the
	// navigation about to be asked for, and a wait that times out can tell the
	// difference between "the site is slow" and "we made a call the engine
	// threw away".
	t.engMu.Lock()
	t.engErr = nil
	t.engMu.Unlock()
	// And the same for what we think the page IS, which is the half that was
	// wrong in production before anyone noticed.
	//
	// meta arrives from the page a beat after it loads, so `open` polls until it
	// is non-empty. On a fresh tab that works. On a REUSED one the previous
	// page's title and URL are already sitting there, so the poll succeeds on
	// its first read and `open` reports the page it just left — seen in the log
	// as "เปิดแล้ว: Example Domain" for a navigation to x.com. Every reused open
	// since tab reuse shipped has been naming the wrong page, and
	// parseBrowserOpened has been filing those into the visited-pages panel.
	t.metaMu.Lock()
	t.title, t.url = "", ""
	t.metaMu.Unlock()
}

func (t *browserTab) setNavOK(ok bool) {
	t.metaMu.Lock()
	t.navOK = ok
	t.metaMu.Unlock()
}

// setFallback arms the one retry the next navigation is allowed. An empty
// string disarms it, which is what every navigation with a scheme of its own
// passes.
func (t *browserTab) setFallback(url string) {
	t.fbMu.Lock()
	defer t.fbMu.Unlock()
	t.fallback = url
}

// takeFallback returns the armed retry and disarms it in the same breath.
func (t *browserTab) takeFallback() string {
	t.fbMu.Lock()
	defer t.fbMu.Unlock()
	url := t.fallback
	t.fallback = ""
	return url
}

func (t *browserTab) navLoaded() bool {
	t.metaMu.Lock()
	defer t.metaMu.Unlock()
	return t.navOK
}

// awaitNavigation blocks until the tab's first navigation completes, then
// reports whether the page actually loaded. Waiting alone is not enough:
// "navigation completed" only says the engine stopped, not that it stopped on
// the page that was asked for, so a caller that trusted navDone reported
// success over a File-not-found page and kept working from it.
func (t *browserTab) awaitNavigation(ctx context.Context, timeout time.Duration) error {
	_, done := t.latch()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(timeout):
		// If the engine complained about a call we made during this navigation,
		// that is the answer and it is OURS, not the page's.
		//
		// A plain errors.New, deliberately, where the line below is a
		// statereport: the two sentences describe different worlds. "The site
		// is slow tonight" is a report about a moment and nothing to correct.
		// "The engine refused our call" is a defect in this program, and
		// marking it as weather is how it survived a week of being logged
		// every twenty seconds — §127.8, where the agent read the weather
		// report it was handed and told the user the network was bad.
		if engErr := t.engineError(); engErr != nil {
			return fmt.Errorf("the browser engine refused what Aetox asked it to do: %w", engErr)
		}
		// Both are reports about a page at a moment — slow tonight, down
		// tonight — not behaviour to correct (statereport, not errors.New):
		// left unmarked, three of these became a permanent-memory card about
		// avoiding a localhost URL whose server simply was not running yet.
		return statereport.New("page did not finish loading")
	}
	// The latch closed because the engine is gone and could not be replaced
	// (giveUp), not because a page arrived. Named as ours, like the refusal
	// above: the page did nothing wrong.
	if why := t.deadWhy(); why != nil {
		return fmt.Errorf("the browser engine behind this tab died and could not be brought back: %w", why)
	}
	if !t.navLoaded() {
		return statereport.New("page failed to load — not found, or unreachable")
	}
	return nil
}

type browserHost struct {
	app     *App
	backend hostBackend

	mu   sync.Mutex
	tabs map[string]*browserTab
	// views holds the live webviews, and is the reason browserTab no longer
	// does. Nothing outside this file reads it, and onTab is the only thing
	// that hands one out — always from inside backend.do, which is the whole
	// point (see browserTab's own comment, and browser_shot.go's header for
	// what the engine does when that rule is broken).
	views map[string]tabView
	// onUserClose is how a close that Aetox did not start gets back into the
	// app: the × on a detached window’s own frame. Set by the App that owns
	// this host, because closeTab lives there.
	onUserClose func(id string)
	// Two fields because there are two questions, and one field answering both
	// is what put the agent's keystrokes on the user's page.
	//
	// lastID is "the one on screen": most recently opened *or shown*, rewritten
	// by BrowserSetVisible every time a tab is raised, including by the user
	// clicking between their own. The frontend asks this; nothing the agent does
	// may.
	//
	// agentID is "the one the agent is working" and agentOrder is every tab the
	// agent owns, oldest first. Only a web-agent- tab reaches either, so raising
	// a user's tab cannot move them and the agent cannot lose track of itself.
	//
	// Two fields for the same reason lastID and agentID are two: "which of mine
	// am I working" and "which are mine" are different questions. The agent had
	// exactly one tab until 2026-08-17, which fused a rule about OWNERSHIP (the
	// agent's tabs are its own, never the user's) with a rule about COUNT (there
	// is one of them) — and only the first was ever load-bearing. The prefix
	// separates the agent's tabs from the user's at any number.
	//
	// agentID is not cleared by whoever closes a tab: one the user closed is gone
	// from tabs, and agentTab reads that as no tab, which is also how `open`
	// learns to mint a fresh one. agentOrder IS pruned, because a list is read
	// and a list that names a dead tab is a list that lies.
	lastID     string
	agentID    string
	agentOrder []string
	// goneID and goneWhy remember that a page the agent was working stopped
	// existing, and who ended it, so the next thing the agent asks of the
	// browser can say so instead of answering "you have no page open" as though
	// the agent had forgotten to open one. Owner, 22 ส.ค.: a click in the
	// browser is a user-side action, and the model is meant to hear about it
	// and carry on.
	//
	// The id is half the record and not decoration. Without it this was a bare
	// bool that anything could raise, which could only answer "did something get
	// closed recently" — while the question being asked is "what happened to the
	// page I am holding". The two come apart the moment the agent opens a new
	// page: a bool set before that survives the reopen and lands on the first
	// action of a fresh, perfectly live tab. So the id is cleared the moment the
	// agent has a page again (see open), and the reason is written only by the
	// call that actually removed a tab (see closeTab).
	goneID  string
	goneWhy closeReason
}

func newBrowserHost(app *App) *browserHost {
	return &browserHost{
		app:     app,
		backend: newHostBackend(),
		tabs:    map[string]*browserTab{},
		views:   map[string]tabView{},
		// The one close that starts outside Aetox: the × on a detached window.
		onUserClose: func(id string) { app.closeTab(id, closedByUser) },
	}
}

func (h *browserHost) start() error { return h.backend.start() }

func (h *browserHost) tab(id string) *browserTab {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tabs[id]
}

// open creates the webview for a tab, on the thread that owns webviews.
func (h *browserHost) open(id, url, fallback string, x, y, w, hgt int) {
	debuglog.Msg("browser.open(%s): queueing (url=%s)", id, url)
	h.backend.do(func() {
		debuglog.Msg("browser.open(%s): running on the host thread", id)
		if t := h.tab(id); t != nil {
			// Open, but with no engine behind it: the browser process ended
			// and nothing has put a new one there yet (engineGone queues a
			// revive; this open may simply have got here first). Reviving on
			// the page THIS call asked for is better than on the page the tab
			// last had, and it is what the caller is about to wait on.
			if t.isDead() {
				debuglog.Msg("browser.open(%s): the tab's engine is gone; reviving it on %s", id, url)
				t.remember(url)
				t.rememberBounds(x, y, w, hgt)
				h.revive(id, t)
				return
			}
			// Not a failure, and not a navigation either: the tab is already
			// embedded, so there is nothing here to create. A reused tab was
			// navigated by workbenchOpenBrowser before its desk event went out,
			// and the frontend's handler calls BrowserOpen to RAISE a tab as
			// well as to create one — so this return is how the common case
			// ends, several times per session.
			//
			// It says so out loud because the silence was read as a hang. On
			// 5 ก.ย. a run's log ended on the line above and the missing
			// "embed ok" was diagnosed as a deadlock in this function; the
			// actual cause was the process restarting twelve seconds later,
			// which the next log file states on its fourth line. A branch that
			// logs its entry and not its exit leaves the log ending on a
			// sentence that reads as unfinished work, and whoever reads it next
			// has to disprove this function before looking anywhere else.
			debuglog.Msg("browser.open(%s): tab already open, nothing to create", id)
			return
		}
		tab := &browserTab{navDone: make(chan struct{}), navOnce: &sync.Once{}, fallback: fallback}
		tab.remember(url)
		tab.rememberBounds(x, y, w, hgt)
		view := h.backend.openTab(id, url, x, y, w, hgt, h.callbacks(id, tab))
		if view == nil {
			return // the backend has already logged why
		}

		h.mu.Lock()
		h.tabs[id] = tab
		h.views[id] = view
		h.lastID = id
		if isAgentTabID(id) {
			h.agentID = id
			if !slices.Contains(h.agentOrder, id) {
				h.agentOrder = append(h.agentOrder, id)
			}
			// The agent has a page again, so whatever happened to the last one
			// has stopped being news. Cleared HERE rather than when the message
			// is read, because a record that outlives the reopen is a sentence
			// about a tab that no longer exists, delivered against one that
			// does — which is the whole failure this pair replaced.
			h.goneID, h.goneWhy = "", 0
		}
		h.mu.Unlock()
	})
}

// callbacks is what the backend is handed for a tab's engine, written once
// because a tab now gets an engine more than once: at open, and again at
// every revive. Two copies of these closures would be two places for the
// engine-gone wiring to be forgotten.
func (h *browserHost) callbacks(id string, tab *browserTab) tabCallbacks {
	gen := tab.nextGen()
	return tabCallbacks{
		onMessage:     func(raw, source string) { h.onMessage(id, tab, raw, source) },
		onNavDone:     func(v tabView, ok bool) { h.navCompleted(id, tab, v, ok) },
		onEngineError: tab.noteEngineError,
		onEngineGone:  func(err error) { h.engineGone(id, tab, gen, err) },
		// The × on a detached window’s own frame. Routed to the same door the
		// strip’s × uses, so a page that goes away goes away once, in one
		// vocabulary, and the agent is told the user closed it.
		onWindowClosed: func() { h.windowClosed(id) },
	}
}

// windowClosed is a detached window’s frame being closed by the user.
//
// It goes through closeTab rather than destroying anything here, for the reason
// the close-reason vocabulary exists at all: what the agent must be told is not
// “the window is gone” but WHO ended it. A page the user shut is not a page
// that crashed and is not a page the agent closed, and only one of the three is
// a reason to try again.
func (h *browserHost) windowClosed(id string) {
	if h.onUserClose != nil {
		h.onUserClose(id)
	}
}

// detach hands a tab its own window. Idempotent: a second call raises the
// window that is already out, which is the useful reading of a second press.
//
// The title travels because only this side knows it — the view has the page,
// the tab has what the page called itself, and the window needs a name for the
// taskbar the moment it appears rather than at the next navigation.
func (h *browserHost) detach(id string) {
	h.onTab(id, func(v tabView, t *browserTab) {
		title, url := t.meta()
		if title == "" {
			title = url
		}
		if title == "" {
			title = "Aetox"
		}
		w, hgt := t.detachSize()
		v.detach(title, w, hgt)
		t.markDetached()
	})
}

// engineGone is the portable half of "the engine behind this tab is gone".
//
// Called from whatever thread the engine chose — ProcessFailed and the error
// callback both arrive on the host thread, but nothing here depends on that.
// It records the death once and queues the revive rather than doing it in
// place: this may be running inside the engine's own callback, and destroying
// that engine's window from inside its callback is not a place to be.
func (h *browserHost) engineGone(id string, tab *browserTab, gen int, why error) {
	if tab.generation() != gen {
		return // an engine this tab no longer has
	}
	if !tab.markDead(why) {
		return // the same death, reported again
	}
	debuglog.Msg("browser tab %s: engine gone (%v); reviving", id, why)
	// So a wait that times out before the revive lands names the engine
	// rather than the weather.
	tab.noteEngineError(why)
	h.backend.do(func() { h.revive(id, tab) })
}

// revive puts a new engine behind a tab whose engine is gone: the same id,
// the same *browserTab, the same page and rect, a fresh webview. On the host
// thread — callers reach it through do, or are already inside one.
//
// The tab keeps its identity on purpose. Everything that holds a tab — the
// frontend's strip, the agent's agentID, a workbenchOpenBrowser mid-wait — is
// holding the id and the *browserTab, and both stay valid across the revive.
// The wait in particular: if a navigation was pending when the engine died,
// its latch is kept and the revived engine's completion closes it, so the
// caller that asked for the page gets the page, later than it hoped, rather
// than an error it has to retry.
//
// It gives up rather than looping. A second death inside reviveWindow says
// the cause is still there, and the honest answer is the one closeTab gives
// for a view that died: the page is gone, and the agent is told so.
func (h *browserHost) revive(id string, tab *browserTab) {
	if !tab.isDead() {
		return // an earlier revive (or an open) already did this
	}
	h.mu.Lock()
	registered := h.tabs[id] == tab
	old := h.views[id]
	if registered {
		delete(h.views, id)
	}
	h.mu.Unlock()
	if !registered {
		return // closed in the meantime; nothing to put back
	}
	if old != nil {
		old.destroy()
	}

	url, x, y, w, hgt := tab.wanted()
	fail := func(why string) {
		debuglog.Msg("browser tab %s: %s; giving the tab up", id, why)
		tab.giveUp()
		if h.app != nil {
			h.app.closeTab(id, closedByApp)
			return
		}
		h.mu.Lock()
		delete(h.tabs, id)
		h.mu.Unlock()
	}
	if !tab.reviveAllowed(time.Now()) {
		fail("the engine died again within a minute")
		return
	}
	// A wait already in progress keeps its latch (see above); otherwise arm
	// one, so the revived navigation is awaitable like any other.
	if !tab.navPending() {
		tab.armNavigation()
	}
	debuglog.Msg("browser tab %s: reviving on %s", id, url)
	view := h.backend.openTab(id, url, x, y, w, hgt, h.callbacks(id, tab))
	if view == nil {
		fail("a new engine could not be created")
		return
	}
	h.mu.Lock()
	h.views[id] = view
	h.mu.Unlock()
	if tab.isHidden() {
		view.setVisible(false)
	}
	tab.revived(url)
	debuglog.Msg("browser tab %s: engine back", id)
}

// agentTabPrefix marks the ids workbenchOpenBrowser mints. It is the only thing
// that distinguishes a tab the agent opened from one the user did, so it is
// tested in exactly one place — here — and everything downstream reads agentID.
const agentTabPrefix = "web-agent-"

func isAgentTabID(id string) bool { return strings.HasPrefix(id, agentTabPrefix) }

// navCompleted is the portable half of "a navigation finished". It takes the
// view rather than reading tab.view because a fast first navigation can land
// before open() has stored it.
func (h *browserHost) navCompleted(id string, tab *browserTab, view tabView, ok bool) {
	// The guess was wrong, and there is another one to try.
	//
	// This is the second half of address.go's `guess`, and it is the half that
	// makes guessing a scheme honest: `localhost:8443` is served over https as
	// often as `example.com` is still served over plain http, and neither the
	// address bar nor the person typing into it can know which. Chrome and Edge
	// both do exactly this, and it is why they appear to always be right.
	//
	// Before the latch, before visibility, before meta: nothing downstream
	// should hear about a navigation that is about to be replaced, or the tab
	// surfaces the engine's error page for the length of one round trip and the
	// address bar reads back a URL the user is not going to end up on.
	if !ok {
		if fb := tab.takeFallback(); fb != "" {
			debuglog.Msg("browser.nav(%s): the guessed scheme failed, falling back to %s", id, fb)
			tab.goTo(view, fb)
			return
		}
	}
	// Recorded before navDone is closed, so a waiter that wakes on it reads
	// this navigation's outcome and not the previous one's.
	tab.setNavOK(ok)
	once, done := tab.latch()
	once.Do(func() { close(done) })

	// Raise the tab now that the page has rendered. The frontend's
	// browser:meta handler used to be the only thing doing this, which made
	// visibility depend on page JS delivering a message that passes the origin
	// check — never true for file:// before the sameOrigin fix, and fragile in
	// general: the page stayed loaded but composited invisibly behind the
	// app's own webview.
	tab.visMu.Lock()
	hidden := tab.hidden
	tab.visMu.Unlock()
	if !hidden {
		view.setVisible(true)
	}

	// Engines keep zoom per origin, so a cross-site navigation drops the
	// device-emulation factor back to 1 — re-assert it here.
	tab.zoomMu.Lock()
	z := tab.zoom
	tab.zoomMu.Unlock()
	if z > 0 {
		view.setZoom(z)
	}
	view.eval(metaScript())

	// The cursor the user was watching died with the old document. Put it
	// back where they last saw it, on the new one — a pointer that vanished
	// on every navigation would read as the agent having let go of the mouse.
	if h.app != nil && h.app.pageMarksOn() {
		if x, y, ok := tab.cursor(); ok {
			view.eval(cursorShowScript(x, y))
		}
	}
}

// onMessage handles one postMessage envelope from a tab's page. source is the
// sending frame's real origin as the engine reports it — trustworthy, unlike
// anything else in the message, which any page script can set freely.
func (h *browserHost) onMessage(id string, tab *browserTab, raw string, source string) {
	var m aetoxMsg
	if err := json.Unmarshal([]byte(raw), &m); err != nil || m.Aetox == "" {
		return
	}
	switch m.Aetox {
	case "meta":
		// A page can claim any url it likes in the envelope; only trust it if
		// it matches where the engine says the message actually came from —
		// otherwise a page could make the address bar show a URL it isn't at.
		if !sameOrigin(source, m.URL) {
			return
		}
		tab.metaMu.Lock()
		tab.title, tab.url = m.Title, m.URL
		tab.metaMu.Unlock()
		if h.app.ctx != nil {
			h.app.emitEvent("browser:meta:"+id, map[string]string{"title": m.Title, "url": m.URL})
		}
	case "text":
		tab.textMu.Lock()
		ch := tab.textCh
		expectedToken := tab.textToken
		tab.textCh = nil
		tab.textToken = ""
		tab.textMu.Unlock()
		// Reject if nothing is waiting, the token doesn't match this specific
		// BrowserGetText call (stops stale/forged messages from a page), or
		// the claimed url doesn't match the real sending origin.
		if ch == nil || m.Token == "" || m.Token != expectedToken || !sameOrigin(source, m.URL) {
			return
		}
		ch <- browserSnapshot{
			Text: m.Text, Elements: m.Elements, Images: m.Images,
			ElementsTotal: m.ElementsTotal, ImagesTotal: m.ImagesTotal, BlockedFrames: m.Frames,
			CanvasShare: m.CanvasShare, CanvasCount: m.CanvasCount,
		}
	case "act":
		tab.actMu.Lock()
		ch := tab.actCh
		expectedToken := tab.actToken
		tab.actCh = nil
		tab.actToken = ""
		tab.actMu.Unlock()
		if ch == nil || m.Token == "" || m.Token != expectedToken || !sameOrigin(source, m.URL) {
			return
		}
		ch <- browserActResult{
			Found: m.Found, Ref: m.Ref, Tag: m.Tag, Label: m.Label,
			Mode: m.Mode, Active: m.Active, CX: m.CX, CY: m.CY, CanvasShare: m.CanvasShare,
			FocusBefore: m.FocusBefore, Focus: m.Focus, Kept: m.Kept, Mouse: m.Mouse,
			Under: m.Under, VW: m.VW, VH: m.VH, DPR: m.DPR, ScrollX: m.ScrollX, ScrollY: m.ScrollY,
			Count: m.Count, Matches: m.Matches, Marks: m.Marks, Title: m.Title, URL: m.URL,
			FileInput: m.FileInput, Multiple: m.Multiple, Accept: m.Accept,
		}
	case "log":
		tab.logMu.Lock()
		ch := tab.logCh
		expectedToken := tab.logToken
		tab.logCh = nil
		tab.logToken = ""
		tab.logMu.Unlock()
		// Same three gates as "text", for the same three reasons: nothing
		// waiting, a token that is not the one this call minted, or a page
		// claiming to be somewhere it is not.
		if ch == nil || m.Token == "" || m.Token != expectedToken || !sameOrigin(source, m.URL) {
			return
		}
		ch <- browserLogReport{Kind: m.Kind, Entries: m.Log, Dropped: m.Dropped, Armed: m.Armed}
	case "wait":
		tab.waitMu.Lock()
		ch, want := tab.waitCh, tab.waitToken
		tab.waitCh, tab.waitToken = nil, ""
		tab.waitMu.Unlock()
		// Same two checks the text case makes, for the same reason: a page can
		// post this envelope itself, and a forged one must not end a wait the
		// real script is still running.
		if ch == nil || m.Token == "" || m.Token != want || !sameOrigin(source, m.URL) {
			return
		}
		ch <- m.Found
	case "dialog":
		// No token here, and that is deliberate: nobody asked for this message,
		// the page raised it. The origin check is what keeps a frame from
		// putting words in another page's mouth, and the text is quoted rather
		// than obeyed — it is a report about the page, never an instruction.
		if !sameOrigin(source, m.URL) {
			return
		}
		tab.noteDialog(fmt.Sprintf("- %s(%q) — Aetox ตอบว่า %s", m.Dialog, m.Message, m.Answer))
	case "pick":
		// Origin before token, deliberately: a message from the wrong origin
		// must not consume the token the real page is still going to answer
		// with. Order the other way round, and any page in any frame could end
		// a pick the user is halfway through.
		if !sameOrigin(source, m.URL) || !tab.claimPick(m.Token) {
			return
		}
		h.app.emitEvent("browser:pick:"+id, map[string]any{
			"url": m.URL, "cancelled": m.Cancelled, "drawn": m.Drawn, "picks": m.Picks,
		})
	}
}

// ---------------------------------------------------------------------------
// Wails bindings
// ---------------------------------------------------------------------------
//
// Every method below exists on every platform, and none of them is behind a
// build tag. desktop/frontend/wailsjs/go/main/App.d.ts is generated from this
// set and committed; a platform that dropped one would regenerate that file
// without it and break BrowserPane.svelte's imports at vite build time.
// ARCHITECTURE.md §48 Decision 2.

func (a *App) browserHostLazy() (*browserHost, error) {
	a.terminalsMu.Lock()
	if a.browsers == nil {
		a.browsers = newBrowserHost(a)
	}
	h := a.browsers
	a.terminalsMu.Unlock()
	return h, h.start()
}

// onTab runs fn against one tab's webview on the thread that owns webviews.
//
// It is the ONLY way to reach a tabView, and that is the design rather than a
// convenience: an engine call made from any other goroutine is not slow and not
// racy, it is refused outright, and the refusal arrives as a page that never
// finishes loading twenty seconds later (§127.8). Holding the views in the host
// instead of on browserTab is what makes this the only door — there is no field
// left to reach past it.
//
// The view handed to fn is valid for the duration of fn and no longer. Nothing
// in Go stops a caller squirrelling it away to call later; what this prevents
// is the accident, which is the one that happened.
//
// The lookup happens HERE, on the host thread, not at the call site. open()
// only registers a tab at the very end of its own queued command, so anything
// that checked on the caller's goroutine found nil for every call made in the
// moments after BrowserOpen and dropped it silently — which is what left a
// freshly opened tab's window at the rect the pane had before the address bar
// existed, covering the toolbar until something forced a resize. do() is FIFO,
// so by the time this runs the open ahead of it has finished.
func (h *browserHost) onTab(id string, fn func(tabView, *browserTab)) {
	h.backend.do(func() {
		h.mu.Lock()
		v, t := h.views[id], h.tabs[id]
		h.mu.Unlock()
		if v != nil && t != nil {
			fn(v, t)
		}
	})
}

func (a *App) onTab(id string, fn func(tabView, *browserTab)) {
	if host, err := a.browserHostLazy(); err == nil {
		host.onTab(id, fn)
	}
}

// live reports whether id names a tab that still has a webview — the question
// browserTab.view == nil used to answer at call sites. A tab without a view was
// never a tab; it was an openTab that failed and got stored anyway.
func (h *browserHost) live(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tabs[id] != nil && h.views[id] != nil
}

// BrowserOpen creates a native browser tab at the given physical-pixel bounds.
//
// fallback is the other scheme to try if this URL fails to load, and it is
// empty for every caller that was handed a scheme rather than choosing one.
// Only the address bar fills it (Address.Fallback).
func (a *App) BrowserOpen(id, url, fallback string, x, y, w, h int) error {
	host, err := a.browserHostLazy()
	if err != nil {
		return err
	}
	host.open(id, url, fallback, x, y, w, h)
	return nil
}

// BrowserNavigate loads a URL in an existing tab. See BrowserOpen for fallback.
func (a *App) BrowserNavigate(id, url, fallback string) {
	a.onTab(id, func(v tabView, t *browserTab) {
		// Armed before the navigation starts: nav-completed can fire from the
		// engine's thread before this call returns.
		t.setFallback(fallback)
		t.goTo(v, url)
	})
}

// BrowserSetBounds moves/resizes a tab's view (physical pixels, relative to
// the main window client area).
//
// A hidden tab stays hidden through a move. The frontend re-glues bounds for
// reasons that have nothing to do with visibility — a window resize, the
// inspector reopening at another width — and a backend whose move also shows
// (win32's did, until 7 ก.ย.) turned each of those into the page surfacing
// over settings, over a dialog, over another chat. Said here, once, for every
// backend: only BrowserSetVisible(true) puts a tab on screen.
func (a *App) BrowserSetBounds(id string, x, y, w, h int) {
	a.onTab(id, func(v tabView, t *browserTab) {
		t.rememberBounds(x, y, w, h)
		v.setBounds(x, y, w, h)
		if t.isHidden() {
			v.setVisible(false)
		}
	})
}

// BrowserSetZoom scales the page inside a tab — this is what makes the device
// presets real emulation rather than a small window: with the tab's view sized
// to deviceWidth*factor, a zoom of `factor` leaves the page a CSS viewport of
// exactly deviceWidth, so its media queries fire as they would on that device.
// 1 = no emulation (the pane-filling default).
func (a *App) BrowserSetZoom(id string, factor float64) {
	if factor <= 0 {
		factor = 1
	}
	a.onTab(id, func(v tabView, t *browserTab) {
		t.zoomMu.Lock()
		t.zoom = factor
		t.zoomMu.Unlock()
		v.setZoom(factor)
	})
}

// raiseDetached brings a detached window to the front.
//
// It is the whole of what “raise this tab” means once a tab has left the
// panel. Every other raise in this app is a desk event — the window makes the
// chip active and the pane un-hides the view — and a detached tab has no chip
// and no pane, so an event would be addressed to nobody, and (worse) the
// handler would draw it a new chip on whatever desk is on screen.
func (a *App) raiseDetached(id string) {
	a.onTab(id, func(v tabView, _ *browserTab) { v.setVisible(true) })
}

// detachedTab reports whether this id names a tab that has left the panel. Read
// by every raise in the app, which is why it is one call and not a field lookup
// at four sites.
func (a *App) detachedTab(id string) bool {
	if a.browsers == nil {
		return false
	}
	return a.browsers.tab(id).isDetached()
}

// BrowserDetach gives a tab a window of its own, and it never comes back.
//
// The owner’s ask, 8 ก.ย.: *“เราเพิ่มปุ่มแยก ให้เบราว์เซอร์มันรันแยกออกมาเลยได้ไหม
// ไม่อยากให้ผูกอยู่แค่กับใน Aetox ... แต่ Aetox ก็ตามไปควบคุมได้นะ”* — with the rule
// that settles every question below: once out, it is separate from the session
// it came from and outlives it.
//
// So this is a one-way door, and deliberately. A window that could be pushed
// back into the strip would need a session to be pushed back INTO, and the tab
// no longer has one — its chat may have been closed an hour ago. “Come back”
// for a detached page is opening it again, which the agent and the address bar
// can both already do.
//
// Nothing about reaching the page changes. The tools drive the engine, and the
// engine does not know which window it is in: read, click, type, capture and
// the ref marks all keep working, from any chat, exactly as before.
func (a *App) BrowserDetach(id string) {
	if host, err := a.browserHostLazy(); err == nil {
		host.detach(id)
	}
}

// BrowserSetVisible shows/hides a tab (hidden when its dock tab is inactive or
// the settings overlay is open — a native view always floats above the UI).
func (a *App) BrowserSetVisible(id string, visible bool) {
	a.onTab(id, func(v tabView, t *browserTab) {
		t.visMu.Lock()
		t.hidden = !visible
		t.visMu.Unlock()
		if visible {
			a.browsers.mu.Lock()
			a.browsers.lastID = id
			a.browsers.mu.Unlock()
		}
		v.setVisible(visible)
	})
}

// BrowserBack / BrowserForward / BrowserReload drive history via script — not
// every engine wrapper exposes GoBack/GoForward, and the script works on all
// of them.
func (a *App) BrowserBack(id string)    { a.browserEval(id, "history.back()") }
func (a *App) BrowserForward(id string) { a.browserEval(id, "history.forward()") }
func (a *App) BrowserReload(id string)  { a.browserEval(id, "location.reload()") }

// BrowserOpenDevTools opens the engine's own DevTools on a tab — find-in-page,
// console, network, element inspection and its screenshot tools, none of which
// are worth reimplementing in our toolbar.
// BrowserSetScreenShape cuts a tab to the shape of the device it is emulating.
//
// Sizes arrive in device pixels because that is the unit a window has, and the
// pane is the only thing that knows the scale it is drawing the phone at — a
// 932-tall iPhone in a 700-tall panel is drawn at 0.75, and its corners are
// 0.75 of a corner. Passing CSS pixels here and scaling on this side would put
// that arithmetic in two places.
func (a *App) BrowserSetScreenShape(id string, radius, notchW, notchH, notchY int) {
	a.onTab(id, func(v tabView, _ *browserTab) { v.setShape(radius, notchW, notchH, notchY) })
}

func (a *App) BrowserOpenDevTools(id string) {
	a.onTab(id, func(v tabView, _ *browserTab) { v.openDevTools() })
}

func (a *App) browserEval(id, js string) {
	a.onTab(id, func(v tabView, _ *browserTab) { v.eval(js) })
}

// CloseAllBrowserTabs destroys the native browser views this process is still
// holding, except the ones a turn that is still running is working in.
//
// Called by the frontend right after it (re)loads (App.svelte onMount). The Go
// backend outlives a webview reload — a `wails dev` Vite HMR full-reload, or a
// plain Ctrl+R — which wipes the JS-side `workbench` store without running
// BrowserPane's onDestroy, leaving the native view behind with nothing left to
// reposition or close it. It just floats, stuck at its last bounds. Sweeping
// those is what this is for, and on a genuine fresh start `a.browsers` is nil
// and it is a no-op.
//
// **The exception is new, and the reasoning it replaces was wrong.** This used
// to close everything, on the grounds that *"a freshly loaded frontend owns
// zero workbench tabs by definition"*. True of the frontend, false of the app:
// the engine keeps working across a reload, so a turn in flight can be holding
// half a dozen tabs of its own — and reloading the window killed them all
// mid-task, after which the agent walked its own list and was told, correctly
// and uselessly, that every one of them had been closed (owner, 24 ส.ค.).
//
// Only while a turn is running. With nothing working, an agent tab is as
// orphaned as any other and the sweep should take it — otherwise the leftovers
// of a turn that died would sit there for the rest of the app's life with
// nothing left to close them.
func (a *App) CloseAllBrowserTabs() {
	if a.browsers == nil {
		return
	}
	keep := map[string]bool{}
	if a.anyTurnRunning() {
		for _, id := range a.agentTabs() {
			keep[id] = true
		}
	}
	h := a.browsers
	h.mu.Lock()
	ids := make([]string, 0, len(h.tabs))
	for id := range h.tabs {
		if !keep[id] {
			ids = append(ids, id)
		}
	}
	h.mu.Unlock()
	for _, id := range ids {
		// The app tidying up after a reload, which is not the user and must not
		// be reported as one: the agent is told its page is gone, not that
		// somebody closed it.
		a.closeTab(id, closedByApp)
	}
	// Spared is not shown. The frontend that just loaded owns no panes, so a
	// kept window has nothing to hide or move it until a pane adopts it — and
	// the chat that owns it is not necessarily the one the window comes back
	// on. Left as it was, it sat composited over whatever chat loaded, at the
	// bounds it had before the reload, with nothing able to reach it (the
	// owner's "แสดงผลค้าง", 7 ก.ย.). Hidden here, it waits like a parked page:
	// the pane that adopts it (restoreWorkbench, by id) shows it again, and the
	// agent still browsing it never notices, since a hidden tab navigates.
	for id := range keep {
		a.BrowserSetVisible(id, false)
	}
}

// closeReason says who ended a tab.
//
// It has no zero value on purpose. A tab can stop existing for three different
// reasons and the agent needs to be told a different thing about each, so a
// close that does not say which is not a close this program knows how to
// report — and it should not compile.
//
// The rule this replaces was prose: "BrowserClose is the frontend's door, so
// every call to it is the user." True when it was written and false five weeks
// later, because a lifecycle hook in the window and a sweep in the engine both
// reached it and neither is a user. See
// docs/architecture/browser-tab-lifetime-2026-08-25.md.
type closeReason int

const (
	closedByUser  closeReason = iota + 1 // the × on the tab strip
	closedByAgent                        // the agent's own `browser tabs close`
	closedByApp                          // a sweep, a teardown, a view that died
)

// String names the reason for the log, which is the one reader that has to
// tell the three apart after the fact.
func (r closeReason) String() string {
	switch r {
	case closedByUser:
		return "user"
	case closedByAgent:
		return "agent"
	case closedByApp:
		return "app"
	}
	return fmt.Sprintf("closeReason(%d)", int(r))
}

// BrowserClose is the × on the tab strip, and nothing else. It is a Wails
// binding, so the window is its only caller; every close inside the engine
// names its own reason through closeTab.
func (a *App) BrowserClose(id string) {
	a.closeTab(id, closedByUser)
}

// BrowserCloseForTeardown is the window closing a tab for a reason that is not
// a person, and it exists because until 2026-08-27 the window had no way to say
// so.
//
// closedByApp was defined for exactly this ("a sweep, a teardown, a view that
// died") and had only ever been reachable from inside Go. The frontend's single
// door hardcoded closedByUser, so a session switch — which discards the whole
// strip and rebuilds it from the next session's saved layout — had no honest
// call to make. It made none, and the native window was orphaned: still
// composited over the chat, at the bounds it last had, with no pane left alive
// to hide or move it (owner's screenshot, 27 ส.ค.).
//
// The reason matters as much as the close. closedByUser tells the agent
// somebody shut its page, which is the sentence browser-tab-lifetime-2026-08-25
// was written to stop us saying by accident; closedByApp tells it the page is
// gone, which is true, and which is what a session switch actually did.
func (a *App) BrowserCloseForTeardown(id string) {
	a.closeTab(id, closedByApp)
}

// closeTab is the close itself, with nothing said about who asked for it.
//
// Two things are true at once and they are why this is not four lines: the
// registry entry has to go NOW, so the agent stops being told about a tab it
// can no longer use, and the native window cannot go now, because destroying it
// only happens on the browser thread (backend.do posts a message). Deleting
// both together — which is what this did — meant a window whose destroy had not
// run yet was already unreachable: no id in `views`, so nothing could hide it,
// move it or try again. It just sat there, black, over the pane.
//
// So `tabs` and `agentOrder` are dropped immediately (live() reads `tabs`, so
// the tab is closed the instant this returns) and `views` is dropped by the
// queued func, after the window is actually gone.
func (a *App) closeTab(id string, why closeReason) {
	host, err := a.browserHostLazy()
	if err != nil {
		return
	}
	host.mu.Lock()
	v := host.views[id]
	_, wasOpen := host.tabs[id]
	delete(host.tabs, id)
	// Said in the log, with who asked. Every open logs itself and no close
	// did, so a tab that vanished between two tool calls — web-agent-1 on
	// 6 ก.ย. 21:07, closed by nobody the log could name — left only the
	// agent's "no page open" to reason backwards from.
	debuglog.Msg("browser.close(%s): by %s (open=%v, view=%v)", id, why, wasOpen, v != nil)
	// Written by the call that actually removed the tab, and only that one.
	// A second pass over an id already gone writes nothing, so re-entering
	// this function cannot overwrite the reason the first pass gave — which
	// matters because closing a tab tells the window, the window drops the
	// chip, and dropping the chip used to come straight back round here. That
	// echo is a no-op now by construction rather than by a guard somebody has
	// to remember to keep.
	if wasOpen && isAgentTabID(id) {
		host.goneID, host.goneWhy = id, why
	}
	// The current tab falls back to whatever is left rather than to nothing,
	// so closing one of several does not strand the agent mid-task.
	host.agentOrder = slices.DeleteFunc(host.agentOrder, func(open string) bool { return open == id })
	if host.agentID == id {
		host.agentID = ""
		if len(host.agentOrder) > 0 {
			host.agentID = host.agentOrder[len(host.agentOrder)-1]
		}
	}
	if v == nil {
		delete(host.views, id)
	}
	host.mu.Unlock()
	if v != nil {
		host.backend.do(func() {
			v.destroy()
			host.mu.Lock()
			delete(host.views, id)
			host.mu.Unlock()
		})
	}
	// The window is told, which it never was. `workbench:open-browser` had no
	// partner, so a tab closed from this side stayed on the strip forever —
	// pointing at a native view that no longer existed, and with the pane
	// latched open (BrowserPane's `opened`) so nothing would ever re-open it.
	// The file side has said both halves all along (workbench:close-file).
	if wasOpen || v != nil {
		a.deskEvent("", "close-browser", map[string]string{"id": id})
	}
}

// BrowserGetText returns the visible text content of a tab's current page —
// this is the read-path the AI agent uses to work with the browser.
func (a *App) BrowserGetText(id string) (string, error) {
	// No filter: this is the frontend's own read of the whole page, not the
	// agent's search for one control on it.
	snap, err := a.browserSnapshot(id, "")
	if err != nil {
		return "", err
	}
	return snap.Text, nil
}

// browserSnapshot reads page text plus the interactive elements tagged by
// textScript, in one round trip. Used by BrowserGetText and browser_read.
// noteRefs records what a read left for click and type to aim at. Called on
// every read, including a filtered one, because a filtered read is the case
// that made this worth keeping.
func (t *browserTab) noteRefs(count int, filter string) {
	if t == nil {
		return
	}
	t.refMu.Lock()
	t.refCount, t.refFilter, t.refRead = count, filter, true
	t.refMu.Unlock()
}

func (t *browserTab) refs() (count int, filter string, read bool) {
	if t == nil {
		return 0, "", false
	}
	t.refMu.Lock()
	defer t.refMu.Unlock()
	return t.refCount, t.refFilter, t.refRead
}

func (a *App) browserSnapshot(id, filter string) (browserSnapshot, error) {
	host, err := a.browserHostLazy()
	if err != nil {
		return browserSnapshot{}, err
	}
	t := host.tab(id)
	if t == nil {
		return browserSnapshot{}, fmt.Errorf("no browser tab %q", id)
	}

	token := newMessageToken()
	ch := make(chan browserSnapshot, 1)
	t.textMu.Lock()
	t.textCh = ch
	t.textToken = token
	t.textMu.Unlock()

	// This blocks below, so do() must not: see hostBackend.do.
	host.onTab(id, func(v tabView, _ *browserTab) { v.eval(textScript(token, filter)) })

	select {
	case snap := <-ch:
		t.noteRefs(len(snap.Elements), filter)
		return snap, nil
	case <-time.After(5 * time.Second):
		t.textMu.Lock()
		t.textCh = nil
		t.textToken = ""
		t.textMu.Unlock()
		return browserSnapshot{}, fmt.Errorf("page did not respond (still loading?)")
	}
}

// browserActOn runs one ref-aimed script and collects the report it sends
// before acting. answered is false when the page never said anything, which is
// a third outcome and not a failure: the report is a courtesy the page performs
// and a page can be busy, gone, or refusing to run scripts at all.
func (a *App) browserActOn(id string, build func(token string) string) (res browserActResult, answered bool, err error) {
	return a.browserActOnFor(id, build, 2*time.Second)
}

// browserActOnFor is browserActOn with the caller's own patience. The change
// note asks the page about itself right after an action that may be
// navigating away, and a page mid-unload never answers; waiting the full two
// seconds for that silence, on every click, would cost more than the note
// saves. That caller asks for less and reads silence as "the page is going".
func (a *App) browserActOnFor(id string, build func(token string) string, wait time.Duration) (res browserActResult, answered bool, err error) {
	host, err := a.browserHostLazy()
	if err != nil {
		return browserActResult{}, false, err
	}
	t := host.tab(id)
	if t == nil {
		return browserActResult{}, false, fmt.Errorf("no browser tab %q", id)
	}

	token := newMessageToken()
	ch := make(chan browserActResult, 1)
	t.actMu.Lock()
	t.actCh = ch
	t.actToken = token
	t.actMu.Unlock()

	host.onTab(id, func(v tabView, _ *browserTab) { v.eval(build(token)) })

	select {
	case r := <-ch:
		return r, true, nil
	// Short, because the report is sent before the click rather than after it:
	// nothing here is waiting on a page to finish loading or a handler to run.
	case <-time.After(wait):
		t.actMu.Lock()
		t.actCh = nil
		t.actToken = ""
		t.actMu.Unlock()
		return browserActResult{}, false, nil
	}
}

// browserClickRef clicks the element tagged with ref by the most recent
// browser_read snapshot (see textScript), and reports what it was aimed at.
func (a *App) browserClickRef(id string, ref int) (browserActResult, bool, error) {
	return a.browserActOn(id, func(token string) string { return clickScript(token, ref) })
}

// BrowserClickRef is the binding the frontend was generated against. It keeps
// its signature because the frontend has no use for the report; the tool, which
// has to tell the model whether the ref matched anything, calls the one above.
func (a *App) BrowserClickRef(id string, ref int) error {
	_, _, err := a.browserClickRef(id, ref)
	return err
}

// browserTypeRef sets an input/textarea/select/contenteditable's value, tagged
// with ref by the most recent browser_read snapshot (see textScript), and
// reports what it was aimed at. enter presses Enter afterwards (for forms with
// no submit button).
func (a *App) browserTypeRef(id string, ref int, text string, enter bool) (browserActResult, bool, error) {
	return a.browserActOn(id, func(token string) string { return typeScript(token, ref, text, enter) })
}

// BrowserTypeRef is the binding the frontend was generated against; see
// BrowserClickRef for why it keeps its shape.
func (a *App) BrowserTypeRef(id string, ref int, text string, enter bool) error {
	_, _, err := a.browserTypeRef(id, ref, text, enter)
	return err
}
