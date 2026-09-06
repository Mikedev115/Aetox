package main

// เมาส์ ลูกศรเลื่อน และแรงกระเพื่อมบนหน้าเว็บ, checked without a browser.
//
// These are scripts, so what can be pinned here is what the script SAYS, and
// the three things it says are the three rules the layer would be wrong
// without: it takes the previous mark down before putting one up, it mounts
// clear of the page's own tree, and it asks the machine about motion rather
// than the switch. Each of those failing is silent in a browser — a stack of
// stale rings, a mark that drifts, an animation on a machine that asked for
// none — so they are worth a test that runs in a second.

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
)

func TestMarkScriptsClearBeforeTheyDraw(t *testing.T) {
	for name, js := range map[string]string{
		"click":  markRippleScript(120, 240),
		"scroll": markScrollScript("down", 0),
	} {
		mount := strings.Index(js, "function aetoxMarkMount")
		clear := strings.Index(js, "aetoxMarkClear();\n    var root")
		if mount < 0 || clear < 0 || clear < mount {
			t.Errorf("%s: the mount does not begin by clearing what is already there", name)
		}
	}
}

// documentElement, never a node inside the page. A transform or a filter
// anywhere up the ancestor chain makes a new containing block, and
// position:fixed inside one stops being fixed to the viewport — the mark then
// drifts with the page instead of pointing at it. Plenty of sites put
// transform:translateZ(0) on body, so body is not far enough up.
func TestMarksMountAtTheRootAndStayFixed(t *testing.T) {
	for name, js := range map[string]string{
		"click":  markRippleScript(10, 20),
		"scroll": markScrollScript("up", 0),
	} {
		if !strings.Contains(js, "var root=document.documentElement;") {
			t.Errorf("%s: does not mount on the document element", name)
		}
		if !strings.Contains(js, "position:fixed") {
			t.Errorf("%s: the mark is not fixed to the viewport", name)
		}
		if !strings.Contains(js, "pointer-events:none") {
			t.Errorf("%s: the mark can be clicked, and would eat the user's click", name)
		}
	}
}

// The switch says what Aetox would LIKE to draw; the machine says whether it
// moves. Both scripts have to ask the page itself, because the page is the one
// with the user's setting.
func TestMarksAskThePageAboutMotion(t *testing.T) {
	for name, js := range map[string]string{
		"click":  markRippleScript(300, 80),
		"scroll": markScrollScript("bottom", 0),
	} {
		if !strings.Contains(js, "prefers-reduced-motion: reduce") {
			t.Errorf("%s: never asks the machine whether it may move", name)
		}
		if !strings.Contains(js, "aetoxMarkQuiet()") {
			t.Errorf("%s: asks and does not act on the answer", name)
		}
	}
}

// A jump gets two chevrons and a screen at a time gets one, so the four
// directions read as two distances without a word being written — the words are
// the action bar's job, on a page whose language is not the app's.
func TestScrollMarkPointsTheRightWay(t *testing.T) {
	for _, tc := range []struct {
		to      string
		place   string
		turn    string
		chevron string
	}{
		{"down", "bottom:44px", "rotate(45deg)", "i<1"},
		{"bottom", "bottom:44px", "rotate(45deg)", "i<2"},
		{"up", "top:44px", "rotate(-135deg)", "i<1"},
		{"top", "top:44px", "rotate(-135deg)", "i<2"},
	} {
		js := markScrollScript(tc.to, 0)
		for _, want := range []string{tc.place, tc.turn, tc.chevron} {
			if !strings.Contains(js, want) {
				t.Errorf("scroll %s: script is missing %q", tc.to, want)
			}
		}
	}
}

// The switch gates DRAWING and nothing else. Clearing has to run either way, or
// turning the layer off mid-run would leave the last mark on the page for the
// rest of its life.
func TestPageMarksSwitchGatesDrawingOnly(t *testing.T) {
	on := &App{cfg: config.Config{}}
	if !on.pageMarksOn() {
		t.Error("the layer ships on and reports off")
	}
	off := &App{cfg: config.Config{BusyPageMarksOff: true}}
	if off.pageMarksOn() {
		t.Error("the layer was switched off and reports on")
	}
	// Neither door panics with no browser host behind it, which is the state
	// every one of this package's tests constructs an App in — and, more to the
	// point, the state a session is in before anybody has opened a page.
	off.markPageClick(AgentTabID("web-agent-1"), point{4, 5})
	off.markPageScroll(AgentTabID("web-agent-1"), "down", 0)
	off.clearPageMarks(AgentTabID("web-agent-1"))
	on.markPageClick(AgentTabID("web-agent-1"), point{4, 5})
	on.markPageScroll(AgentTabID("web-agent-1"), "down", 0)
}

// Nothing of Aetox's own in the photograph. The ring sits directly over the
// control it points at, and a model handed that picture has no way to know the
// circle is not part of the site.
func TestClearScriptRemovesTheMarkByID(t *testing.T) {
	js := clearMarksScript("tok-1")
	if !strings.Contains(js, markElementID) {
		t.Errorf("the clear does not name the mark it is meant to remove")
	}
	if strings.Contains(js, "createElement") {
		t.Error("the clear draws something")
	}
	// The report is what makes this one different from every other mark script,
	// and it is what capture waits on. Without it the only thing between a stale
	// ring and the photograph is a sleep that was put there for the raise.
	if !strings.Contains(js, "aetoxReport(\"tok-1\",0,null)") {
		t.Errorf("the clear never says it is done:\n%s", js)
	}
	// Reported AFTER the removal, or it would be answering for work it had not
	// done yet.
	if strings.Index(js, "removeChild") > strings.Index(js, "aetoxReport(\"tok-1\"") {
		t.Error("it reports before it removes")
	}
}

// The cursor is the one mark that is not an action: it stays, so it must not
// go through the mount that clears the previous mark, and it must come off
// before a capture with the trail beside it.
func TestCursorLivesBesideTheMarksAndDiesBeforeACapture(t *testing.T) {
	js := cursorMoveScript(40, 50, 200*time.Millisecond, true)
	if strings.Contains(js, "aetoxMarkMount(") {
		t.Error("the cursor must not be mounted through aetoxMarkMount, which clears the ring it should outlive")
	}
	for _, want := range []string{"position:fixed", "pointer-events:none", "document.documentElement", "createElementNS", "prefers-reduced-motion"} {
		if !strings.Contains(js, want) {
			t.Errorf("cursor script missing %q", want)
		}
	}
	clear := clearMarksScript("tok")
	for _, id := range []string{markElementID, cursorElementID, trailElementID} {
		if !strings.Contains(clear, id) {
			t.Errorf("clearMarksScript does not remove %s, so a capture would photograph it", id)
		}
	}
	drag := cursorDragScript(point{1, 2}, point{30, 40}, 300*time.Millisecond)
	if !strings.Contains(drag, "createElement(\"canvas\")") || !strings.Contains(drag, "devicePixelRatio") {
		t.Error("the trail is a canvas at device pixel ratio")
	}
	if !strings.Contains(cursorShowScript(3, 4), "aetoxCursorTo(3,4,0,false)") {
		t.Error("cursorShowScript must place the sprite without travel")
	}
}

// The layer off is no sprite and no wait; the position is remembered either
// way so the sprite comes back where it should when the layer returns.
func TestCursorMoveHonoursTheSwitch(t *testing.T) {
	app := &App{}
	app.cfg.BusyPageMarksOff = true
	app.browsers = &browserHost{app: app, tabs: map[string]*browserTab{"web-agent-1": {}}, views: map[string]tabView{"web-agent-1": &fakeView{}}}
	if wait := app.markCursorMove("web-agent-1", point{10, 10}, true); wait != 0 {
		t.Errorf("with marks off a click must not wait for a sprite, got %v", wait)
	}
	if x, y, ok := app.browsers.tab("web-agent-1").cursor(); !ok || x != 10 || y != 10 {
		t.Errorf("the position is remembered regardless, got %v %v %v", x, y, ok)
	}
}

// The click mark is a ripple leaving the point, not a light around the box.
// Two things are pinned: no halo anywhere (a glow is what grows back one
// shadow at a time), and the mark placed AT the coordinates rather than around
// a measured rect — which is what lets a click by x,y on a canvas have any
// feedback at all.
func TestClickMarkIsARippleFromThePoint(t *testing.T) {
	js := markRippleScript(120, 240)
	if strings.Contains(js, "box-shadow") {
		t.Error("the click mark is wearing a glow again")
	}
	if !strings.Contains(js, "left:120px;top:240px;width:0;height:0;") {
		t.Errorf("the ripple is not mounted at the click point: %s", js)
	}
	if !strings.Contains(js, "wave(0);wave(150);") {
		t.Error("one wave reads as a circle that appeared; two read as something leaving the point")
	}
	// Nothing may come to rest visible. element.animate() with the default
	// fill drops the element back to its own style when it finishes, so every
	// animated piece starts at opacity 0 and the page is left clean.
	for _, want := range []string{`border:2.5px solid "+AETOX_ACC+";border-radius:50%;opacity:0;`, `background:"+AETOX_ACC+";opacity:0;`} {
		if !strings.Contains(js, want) {
			t.Errorf("a ripple piece can come to rest visible, missing %q", want)
		}
	}
}

// The sprite is an ordinary arrow, and the three places that have to agree on
// its point are what this pins.
//
// The owner had the logo for a day and sent it back — *"เอาเมาส์ปกติก็ได้ ไม่เอา
// แบบนี้"* — so what is drawn is the shape an operating system draws, at 1:1
// with no scale and no rotation. That flatness is the point of the test as much
// as the shape is: the group transform puts the tip on the hotspot, the
// transform-origin squeezes about it on a press, and the translate moves the
// sprite to a page coordinate by it. Any one of them drifting means a pointer
// that points somewhere other than where the click goes, and the fewer numbers
// stand between the path and the pixel the fewer places that can happen.
func TestCursorIsAnArrowAndAgreesOnItsTip(t *testing.T) {
	js := cursorMoveScript(10, 20, 200*time.Millisecond, true)
	tip := fmt.Sprintf("%g,%g", cursorTipX, cursorTipY)
	for _, want := range []string{
		"AETOX_ARROW=",
		"M 0,0 L 0,17.1",                  // the point is the path's own origin
		`"translate(` + tip + `)"`,        // and the only transform puts it on the hotspot
		`viewBox:"0 0 16 24"`,             // a box the 12.1x19.7 shape fits inside with its casing
		`"stroke-width":"1.5"`,            // one outline, stroked in ink and filled white
		`"stroke-linejoin":"round"`,       // so the 20-degree tip is a point, not a spike
		`fill:"#ffffff",stroke:AETOX_INK`, // body light, border dark: legible on any page
		fmt.Sprintf("transform-origin:%gpx %gpx", cursorTipX, cursorTipY),
		fmt.Sprintf(`"translate("+(x-%g)+"px,"+(y-%g)+"px)"`, cursorTipX, cursorTipY),
		cursorInk,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("cursor sprite missing %q", want)
		}
	}
	// The geometry the logo needed is gone rather than merely unused. A stray
	// scale or rotate left behind is a number nothing reads until the day
	// somebody edits the path and cannot work out why the tip sits wrong.
	for _, gone := range []string{"scale(0.04703)", "rotate(-14)", "AETOX_LOGO", "evenodd"} {
		if strings.Contains(js, gone) {
			t.Errorf("the logo sprite's %q survived the change back to an arrow", gone)
		}
	}
	// Built attribute by attribute. innerHTML is the one thing a page with
	// Trusted Types enforced refuses outright, and this sprite exists to be
	// seen on exactly those sites.
	if strings.Contains(js, ".innerHTML") {
		t.Error("the sprite is assembled with innerHTML, which the strictest sites refuse")
	}
}

// The arrow is the size a system arrow is, and it fits the box it is given.
//
// Both halves matter and they pull against each other: a sprite drawn larger
// than its viewBox is silently clipped, and one drawn much smaller than 20px
// stops reading as the mouse pointer it is imitating. The path is in CSS pixels
// precisely so this can be checked by reading it.
func TestCursorArrowIsSystemSizedAndFitsItsBox(t *testing.T) {
	var maxX, maxY float64
	for _, pair := range regexp.MustCompile(`(\d+(?:\.\d+)?),(\d+(?:\.\d+)?)`).FindAllStringSubmatch(cursorArrowPath, -1) {
		x, _ := strconv.ParseFloat(pair[1], 64)
		y, _ := strconv.ParseFloat(pair[2], 64)
		maxX, maxY = math.Max(maxX, x), math.Max(maxY, y)
	}
	if maxY < 16 || maxY > 24 {
		t.Errorf("the arrow is %gpx tall; a system pointer is about 20", maxY)
	}
	// The casing is stroked half in and half out, so the ink reaches 0.75px
	// past the outline on the far side, and the inset moves the whole shape.
	const casing = 0.75
	if got, limit := maxX+cursorTipX+casing, 16.0; got > limit {
		t.Errorf("the arrow reaches %gpx across a %gpx box and will be clipped", got, limit)
	}
	if got, limit := maxY+cursorTipY+casing, 24.0; got > limit {
		t.Errorf("the arrow reaches %gpx down a %gpx box and will be clipped", got, limit)
	}
}
