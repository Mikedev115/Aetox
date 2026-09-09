package main

// A tab that has left the panel for a window of its own.
//
// The owner's rule, 8 ก.ย.: *"ถ้าแยกมาแล้วคือแยกกับเซสชั่น"* — once out, the page
// is separate from the chat it came from and outlives it. Everything below is
// that sentence, checked: the window is asked for, the raises that used to go
// through a desk stop going through one, and the × on the new frame comes back
// in as a close by the user rather than as a window that quietly vanished.

import (
	"strings"
	"testing"
)

// setMeta is what the message bridge does when a page reports itself. Written
// here because no production caller needs a setter — the only writer is the
// bridge, in place.
func setMeta(t *browserTab, title, url string) {
	t.metaMu.Lock()
	t.title, t.url = title, url
	t.metaMu.Unlock()
}

func detachedHost(t *testing.T) (*App, *fakeBackend, *fakeView) {
	t.Helper()
	app := &App{}
	b := &fakeBackend{}
	view := &fakeView{}
	tab := &browserTab{}
	app.browsers = &browserHost{
		app: app, backend: b,
		tabs:  map[string]*browserTab{"web-agent-1": tab},
		views: map[string]tabView{"web-agent-1": view},
	}
	return app, b, view
}

// The window is asked for with the page's own name on it, because a row of
// untitled windows is what alt-tab shows otherwise — and finding this page
// later is the entire point of pulling it out.
func TestDetachAsksForAWindowNamedAfterThePage(t *testing.T) {
	app, b, view := detachedHost(t)
	tab := app.browsers.tab("web-agent-1")
	setMeta(tab, "ALMxIMPACT Tennis Booking", "http://localhost:8080/overview")
	tab.rememberBounds(10, 20, 900, 700)

	app.browsers.detach("web-agent-1")
	b.drain()

	if !view.detached {
		t.Fatal("the view was never asked for a window of its own")
	}
	if view.detachTitle != "ALMxIMPACT Tennis Booking" {
		t.Errorf("window title = %q, want the page's own", view.detachTitle)
	}
	// The size it had in the panel, so the page does not reflow on the way out.
	if view.detachSize != [2]int{900, 700} {
		t.Errorf("window size = %v, want the rectangle it already had", view.detachSize)
	}
	if !tab.isDetached() {
		t.Error("the tab does not know it left")
	}
}

// A page with no title yet still gets a name: the address, and failing that
// anything at all. An empty caption is a window the user cannot pick out.
func TestDetachNamesAnUntitledPageAnyway(t *testing.T) {
	app, b, view := detachedHost(t)
	setMeta(app.browsers.tab("web-agent-1"), "", "https://example.com/thing")

	app.browsers.detach("web-agent-1")
	b.drain()

	if view.detachTitle == "" {
		t.Error("an untitled page produced an untitled window")
	}
}

// Raising is the one thing that has to change, and it changes at every caller.
// A detached tab has no chip to make active and no pane to un-hide, so an
// open-browser event would not raise it — it would draw it a NEW chip on
// whatever desk happens to be on screen, which is the leak §187 closed arriving
// from the other side.
func TestRaisingADetachedTabTouchesNoDesk(t *testing.T) {
	app, b, view := detachedHost(t)
	events := captureEvents(app)
	app.browsers.tab("web-agent-1").markDetached()

	app.raiseDetached("web-agent-1")
	b.drain()

	for _, e := range events.all() {
		if strings.HasPrefix(e.Name, "workbench:") {
			t.Errorf("a detached tab raised through the desk: %s", e.Name)
		}
	}
	if len(view.visible) == 0 || !view.visible[len(view.visible)-1] {
		t.Error("the window was never brought forward")
	}
}

// The × on the new frame. It is the only close that starts outside Aetox, and
// it has to arrive as the user closing a page — not as a crash, and not as the
// agent's own tidy-up, because only one of those three is a reason to reopen.
func TestClosingTheDetachedWindowReportsTheUserClosedIt(t *testing.T) {
	// The wiring, first: a host built the real way has somewhere for this to go.
	// Without it the × on a detached frame would destroy the window and tell
	// nobody, and the agent would go on holding a page that is gone.
	if newBrowserHost(&App{}).onUserClose == nil {
		t.Fatal("newBrowserHost left the detached window's close unwired")
	}

	app, _, _ := detachedHost(t)
	var closed []string
	app.browsers.onUserClose = func(id string) { closed = append(closed, id) }

	app.browsers.windowClosed("web-agent-1")

	if len(closed) != 1 || closed[0] != "web-agent-1" {
		t.Fatalf("closed = %v, want the tab whose window went", closed)
	}
}

// desk_list stops mentioning it (it left the desk) and `browser tabs list` says
// where it went, because the two answer different questions: what is in the
// panel, and what pages the agent is holding.
func TestTheTabListSaysWhichPagesLeftThePanel(t *testing.T) {
	app, _, _ := detachedHost(t)
	app.browsers.agentOrder = []string{"web-agent-1"}
	setMeta(app.browsers.tab("web-agent-1"), "Booking", "http://localhost:8080/overview")
	app.browsers.tab("web-agent-1").markDetached()

	list := app.agentTabList()
	if !strings.Contains(list, "หน้าต่างแยก") {
		t.Errorf("the list does not say the page is in a window of its own:\n%s", list)
	}
}

// A detached window can still be photographed, whatever the panel thinks of it.
//
// The chip leaves the strip the moment the window is given (detachTab ->
// removeTab), BrowserPane's onDestroy hides the tab it no longer owns, and
// win32Tab.setVisible ignores a hide on a detached window — so the tab's
// `hidden` flag latched true on a window sitting in front of the user, with no
// chip left to ever clear it. capture read that flag literally and refused
// every photograph for the rest of the run, blaming a chat the window no
// longer belongs to. BrowserDetach's promise is that the tools go on working;
// this is the one that had stopped.
func TestCaptureStillPhotographsADetachedWindow(t *testing.T) {
	app, b, _ := detachedHost(t)
	seed(app, &conversation{id: "on-screen"})
	tab := app.browsers.tab("web-agent-1")
	app.browsers.agentID, app.browsers.agentOrder = "web-agent-1", []string{"web-agent-1"}
	app.browsers.detach("web-agent-1")
	b.drain()
	app.BrowserSetVisible("web-agent-1", false) // the pane unmounting behind the detach
	b.drain()
	if !tab.isHidden() {
		t.Fatal("the setup no longer reproduces the panel's hide")
	}

	// owner != the chat on screen: the shape that used to answer "แชตนี้ไม่ได้
	// อยู่บนจอ" without so much as asking the window.
	_, err := (&browserCaptureSkill{app: app, owner: "another-chat"}).capture(t.Context(), false, false)
	if err != nil && strings.Contains(err.Error(), "ไม่ได้อยู่บนจอ") {
		t.Errorf("capture refused a detached window as a hidden one: %v", err)
	}
}
