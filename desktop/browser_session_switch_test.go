package main

// What a session switch, a reload and a hidden window must not cost the agent.
//
// 6 ก.ย. 21:05–21:07, one chat working a Canva page while the owner moved
// between sessions: two captures of eight seconds each answered nothing, the
// tab then vanished and the agent was told it had "no page open" — the reason
// (somebody closed it) eaten by a decorative lookup before the step that
// needed it ran — and, after a switch back, a user-owned web-4 sat on the same
// URL beside the agent's chipless web-agent-2.

import (
	"strings"
	"testing"
)

// The notice that the user closed the page reaches the step that has no page,
// not the bookkeeping around it. Every "which tab is this" stamp on the way
// used to take agentTab(), which consumes the notice.
func TestStepsStillSayTheUserClosedThePage(t *testing.T) {
	app := hostWithTabs(t, "web-agent-1", []string{"web-agent-1"}, "web-agent-1")
	app.BrowserClose("web-agent-1") // the × on the strip, mid-batch

	browser := &browserSkill{app: app}
	_, err := browser.run(t.Context(), map[string]any{"steps": []any{
		map[string]any{"action": "type", "ref": 1, "text": "1080"},
	}})
	if err == nil {
		t.Fatal("a step on a closed page must fail")
	}
	if !strings.Contains(err.Error(), "closed while you worked") {
		t.Errorf("the step was not told the user closed the page: %v", err)
	}
}

// The same, one door over: a single call's before/after bookkeeping.
func TestASingleCallStillSaysTheUserClosedThePage(t *testing.T) {
	app := hostWithTabs(t, "web-agent-1", []string{"web-agent-1"}, "web-agent-1")
	app.BrowserClose("web-agent-1")

	browser := &browserSkill{app: app}
	_, err := browser.run(t.Context(), map[string]any{"action": "type", "ref": 1, "text": "x"})
	if err == nil || !strings.Contains(err.Error(), "closed while you worked") {
		t.Errorf("the call was not told the user closed the page: %v", err)
	}
}

// A hidden window is refused in words, and the words say which of the two
// hidden states this is — because the next move differs.
func TestCaptureNamesWhyAHiddenViewCannotBePhotographed(t *testing.T) {
	bg := captureHiddenErr(true).Error()
	if !strings.Contains(bg, "ไม่ได้อยู่บนจอ") || !strings.Contains(bg, "read") {
		t.Errorf("a background chat's refusal must say the chat is off screen and name read: %s", bg)
	}
	fg := captureHiddenErr(false).Error()
	if !strings.Contains(fg, "tabs select") || strings.Contains(fg, "ไม่ได้อยู่บนจอ") {
		t.Errorf("an on-screen chat's refusal must point at tabs select, not blame the session: %s", fg)
	}
}

// waitShown returns as soon as the raise lands, and gives up on time when it
// never does — it is what keeps the on-screen case from paying the full wait.
func TestWaitShownReturnsWhenTheTabIsShown(t *testing.T) {
	a := &App{}
	tab := &browserTab{hidden: true}
	go func() {
		tab.visMu.Lock()
		tab.hidden = false
		tab.visMu.Unlock()
	}()
	a.waitShown(t.Context(), tab, 2e9)
	if tab.isHidden() {
		t.Fatal("waitShown returned with the tab still hidden")
	}
	still := &browserTab{hidden: true}
	a.waitShown(t.Context(), still, 60e6)
	if !still.isHidden() {
		t.Fatal("a tab nobody showed must still be hidden")
	}
}

// A fresh id is never one a tab already answers to, whatever the counter says.
func TestMintedAgentTabIDSkipsTheOnesAlreadyOpen(t *testing.T) {
	app := hostWithTabs(t, "", nil, "web-agent-1", "web-agent-2")
	agentBrowserSeq = 0 // the process restarted; the window brought two chips back
	if got := app.mintAgentTabID(); got != "web-agent-3" {
		t.Fatalf("minted %q, want web-agent-3 past the two already open", got)
	}
}
