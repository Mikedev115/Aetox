package main

// A tab whose engine has gone gets a new engine, not a funeral.
//
// On 6 ก.ย. the browser process behind web-agent-1 ended at 01:47 while the
// tab stayed in the host's map. Every call after that was refused with
// ERROR_INVALID_STATE, the tool reported "the browser engine refused what
// Aetox asked it to do" fifteen times in twenty minutes, and the pane showed
// a black rectangle with the page's title still on the strip. Nothing in the
// program could notice, because nothing was listening for it.
//
// These tests drive that event through the seam the fix added
// (tabCallbacks.onEngineGone) and check the four things that have to be true
// afterwards: the dead view is destroyed, a new one is created on the same
// page at the same rect, the tab keeps its identity for everyone holding it,
// and the agent is told once.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func openForRevive(t *testing.T) (*fakeBackend, *browserHost, *browserTab, *fakeView) {
	t.Helper()
	b := &fakeBackend{}
	h := &browserHost{backend: b, tabs: map[string]*browserTab{}, views: map[string]tabView{}}
	h.open("web-agent-1", "https://a.example/", "", 1, 2, 3, 4)
	b.drain()
	tab := h.tab("web-agent-1")
	if tab == nil {
		t.Fatal("open did not register the tab")
	}
	first, _ := h.views["web-agent-1"].(*fakeView)
	if first == nil {
		t.Fatal("open did not register a view")
	}
	return b, h, tab, first
}

func TestEngineGoneRevivesTheTabInPlace(t *testing.T) {
	b, h, tab, first := openForRevive(t)

	// The frontend placed the tab and the agent moved on to a second page —
	// both facts a fresh engine has to be given back.
	h.onTab("web-agent-1", func(v tabView, tb *browserTab) {
		tb.rememberBounds(10, 20, 300, 400)
		v.setBounds(10, 20, 300, 400)
	})
	h.onTab("web-agent-1", func(v tabView, tb *browserTab) { tb.goTo(v, "https://b.example/") })
	b.drain()

	b.engineOf("web-agent-1").onEngineGone(errors.New("the browser engine's process exited"))
	if !tab.isDead() {
		t.Fatal("the death was not recorded")
	}
	if first.destroyed {
		t.Fatal("the revive ran on the engine's own thread; it must be queued")
	}
	b.drain()

	if !first.destroyed {
		t.Error("the dead view was not destroyed")
	}
	if h.tab("web-agent-1") != tab {
		t.Error("the tab lost its identity across the revive")
	}
	second := h.views["web-agent-1"]
	if second == nil || second == tabView(first) {
		t.Fatal("no new view was put behind the tab")
	}
	if len(b.opens) != 2 {
		t.Fatalf("expected the original open and one revive, got %d opens", len(b.opens))
	}
	if got := b.opens[1]; got.url != "https://b.example/" || got.bounds != [4]int{10, 20, 300, 400} {
		t.Errorf("revive asked for %+v; want the last page at the last rect", got)
	}
	if tab.isDead() {
		t.Error("the tab is still marked dead after a successful revive")
	}
	note := tab.takeReviveNote()
	if !strings.Contains(note, "https://b.example/") {
		t.Errorf("the agent's note should name the page put back, got %q", note)
	}
	if tab.takeReviveNote() != "" {
		t.Error("the note was available twice; it is meant to be said once")
	}
}

// A complaint from the engine that was replaced must not kill the new one.
// The old webview's queued calls can still be refused after the revive, and
// each refusal arrives through the callbacks that engine was given.
func TestALateWordFromTheOldEngineIsIgnored(t *testing.T) {
	b, h, tab, _ := openForRevive(t)
	old := b.engineOf("web-agent-1")
	old.onEngineGone(errors.New("gone"))
	b.drain()
	if tab.isDead() {
		t.Fatal("precondition: the revive should have cleared the death")
	}
	live := h.views["web-agent-1"]

	old.onEngineGone(errors.New("still gone"))
	b.drain()

	if tab.isDead() {
		t.Error("a late report from the replaced engine marked the live tab dead")
	}
	if h.views["web-agent-1"] != live {
		t.Error("a late report from the replaced engine replaced the live view")
	}
	if len(b.opens) != 2 {
		t.Errorf("expected no third engine, got %d opens", len(b.opens))
	}
}

// open() on a tab that is registered but dead — the open beat the queued
// revive to the host thread — revives it on the page this open asked for.
func TestOpenOnADeadTabRevivesItOnTheNewPage(t *testing.T) {
	b, h, tab, first := openForRevive(t)
	tab.markDead(errors.New("gone"))

	h.open("web-agent-1", "https://c.example/", "", 5, 6, 7, 8)
	b.drain()

	if !first.destroyed {
		t.Error("the dead view was not destroyed")
	}
	if tab.isDead() {
		t.Error("open left the tab dead")
	}
	if len(b.opens) != 2 || b.opens[1].url != "https://c.example/" || b.opens[1].bounds != [4]int{5, 6, 7, 8} {
		t.Errorf("open should have revived on its own page and rect, got %+v", b.opens)
	}
}

// A caller waiting on a navigation when the engine dies keeps waiting on the
// same latch, and the revived engine's completion is what wakes it — with the
// page, not with an error to retry.
func TestARevivedEngineCompletesThePendingWait(t *testing.T) {
	b, h, tab, _ := openForRevive(t)
	tab.armNavigation()
	h.onTab("web-agent-1", func(v tabView, tb *browserTab) { tb.goTo(v, "https://b.example/") })
	b.drain()

	waited := make(chan error, 1)
	go func() { waited <- tab.awaitNavigation(context.Background(), 5*time.Second) }()

	b.engineOf("web-agent-1").onEngineGone(errors.New("gone"))
	b.drain()
	select {
	case err := <-waited:
		t.Fatalf("the wait ended at the revive, before any page arrived: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	// The new engine loads the page.
	b.engineOf("web-agent-1").onNavDone(h.views["web-agent-1"], true)
	select {
	case err := <-waited:
		if err != nil {
			t.Errorf("the revived navigation should satisfy the wait, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the wait did not see the revived engine's completion")
	}
}

// An engine that keeps dying is given up, the way closeTab gives up a view
// that died: out of the map, the agent told the page is gone, and a wait in
// progress ended with the reason rather than after its timeout.
func TestReviveGivesUpWhenTheEngineKeepsDying(t *testing.T) {
	b := &fakeBackend{}
	app := &App{}
	h := &browserHost{app: app, backend: b, tabs: map[string]*browserTab{}, views: map[string]tabView{}}
	app.browsers = h
	h.open("web-agent-1", "https://a.example/", "", 1, 2, 3, 4)
	b.drain()
	tab := h.tab("web-agent-1")

	for i := 0; i < reviveBudget; i++ {
		b.engineOf("web-agent-1").onEngineGone(errors.New("gone"))
		b.drain()
		if tab.isDead() {
			t.Fatalf("revive %d should have been within budget", i+1)
		}
	}

	tab.armNavigation()
	waited := make(chan error, 1)
	go func() { waited <- tab.awaitNavigation(context.Background(), 10*time.Second) }()

	b.engineOf("web-agent-1").onEngineGone(errors.New("gone for the third time"))
	b.drain()

	if h.tab("web-agent-1") != nil {
		t.Error("a tab whose engine cannot stay alive should be closed, not kept")
	}
	if len(b.opens) != 1+reviveBudget {
		t.Errorf("expected exactly %d engines, got %d", 1+reviveBudget, len(b.opens))
	}
	if _, err := app.agentTab(); !errors.Is(err, errAgentTabGone) {
		t.Errorf("the agent should be told its page is gone (closedByApp), got %v", err)
	}
	select {
	case err := <-waited:
		if err == nil || !strings.Contains(err.Error(), "could not be brought back") {
			t.Errorf("the wait should end naming the engine, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the wait outlived the give-up; it should have ended at once")
	}
}

// A tab the user closed between the death and the revive stays closed.
func TestReviveDoesNothingForATabAlreadyClosed(t *testing.T) {
	b, h, tab, first := openForRevive(t)
	b.engineOf("web-agent-1").onEngineGone(errors.New("gone"))
	// closeTab's registry half, as the user's × would do it before the queue turns.
	h.mu.Lock()
	delete(h.tabs, "web-agent-1")
	delete(h.views, "web-agent-1")
	h.mu.Unlock()
	first.destroyed = true

	b.drain()

	if len(b.opens) != 1 {
		t.Errorf("a closed tab was revived: %+v", b.opens)
	}
	if !tab.isDead() {
		t.Error("nothing should have touched the closed tab's state")
	}
}
