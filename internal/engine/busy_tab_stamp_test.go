package engine

// Which page it is happening on — the fact ไฟบอกสถานะ could not draw without.
//
// Before this, the panel could tell that the browser was busy and nothing more,
// so the only honest thing it could do was light the whole panel with five tab
// chips sitting in it. The stamp is what turns "the agent is working" into "the
// agent is working HERE", and it is put on by the engine because the engine is
// the side that holds the tool event — and it asks the screen which tab, since
// the tabs are the screen's (§248 B1, Screen.AgentTab).

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/turn"
)

// tabScreen is a window whose agent tab is a fixed answer.
type tabScreen struct {
	testScreen
	tab string
}

func (s tabScreen) AgentTab() string { return s.tab }

// eventsFrom captures what the window would have been sent.
func eventsFrom(app *Engine) *[]turn.ToolEvent {
	seen := &[]turn.ToolEvent{}
	app.emit = func(_ string, data ...any) {
		if len(data) == 0 {
			return
		}
		stamped, ok := data[0].(SessionEvent[turn.ToolEvent])
		if !ok {
			return
		}
		*seen = append(*seen, stamped.Data)
	}
	return seen
}

func TestABrowserCallIsStampedWithTheTabItIsWorking(t *testing.T) {
	app := &Engine{screen: tabScreen{tab: "web-agent-2"}}
	seen := eventsFrom(app)
	conv := &conversation{id: "s1"}

	app.recordToolAction(conv, turn.ToolEvent{Action: "call", Ref: "c1", Name: "browser", Act: "click"})

	if len(*seen) != 1 {
		t.Fatalf("want one event, got %d", len(*seen))
	}
	if got := (*seen)[0].Tab; got != "web-agent-2" {
		t.Errorf("Tab = %q, want the tab the agent is working", got)
	}
}

// Every other tool gets nothing. A write stamped with a browser tab would have
// the panel light a page that nobody touched.
func TestOnlyTheBrowserGetsATabStamp(t *testing.T) {
	app := &Engine{screen: tabScreen{tab: "web-agent-1"}}
	seen := eventsFrom(app)

	app.recordToolAction(&conversation{id: "s1"},
		turn.ToolEvent{Action: "call", Ref: "c1", Name: "write", Subject: "a.go"})

	if len(*seen) != 1 {
		t.Fatalf("want one event, got %d", len(*seen))
	}
	if got := (*seen)[0].Tab; got != "" {
		t.Errorf("a write was stamped with tab %q", got)
	}
}

// No tab yet is a real state, and it is the state the very first `open` of a
// session is in. The panel reads an empty stamp as "light yourself, point at
// nothing", which is honest; a guessed id would point at somebody else's page.
// An engine with no screen at all answers the same way.
func TestABrowserCallBeforeAnyTabIsStampedWithNothing(t *testing.T) {
	for name, app := range map[string]*Engine{
		"no tab":    {screen: tabScreen{}},
		"no screen": {},
	} {
		seen := eventsFrom(app)

		app.recordToolAction(&conversation{id: "s1"},
			turn.ToolEvent{Action: "call", Ref: "c1", Name: "browser", Act: "open"})

		if len(*seen) != 1 {
			t.Fatalf("%s: want one event, got %d", name, len(*seen))
		}
		if got := (*seen)[0].Tab; got != "" {
			t.Errorf("%s: Tab = %q with no browser open at all", name, got)
		}
	}
}
