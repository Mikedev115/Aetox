package main

// Several moves in one call, checked without a page.
//
// What can run here is the bookkeeping the batch exists for: every step goes
// through the same gate a single call does, the answer numbers every step,
// the first failure stops the rest and says so, and the steps that never ran
// are named. tabs (which reads this process's own bookkeeping) is the one
// action that runs for real; everything that touches a page refuses in words,
// which is exactly the failure the batch has to report well.

import (
	"strings"
	"testing"
)

func TestStepsStopAtTheFirstFailureAndNameTheRest(t *testing.T) {
	app := hostWithTabs(t, "web-agent-1", []string{"web-agent-1"}, "web-agent-1")
	browser := &browserSkill{app: app}
	out, err := browser.run(t.Context(), map[string]any{"steps": []any{
		map[string]any{"action": "tabs", "act": "list"},
		map[string]any{"action": "nonesuch"},
		map[string]any{"action": "capture"},
	}})
	if err == nil {
		t.Fatal("a batch with a failing step must fail")
	}
	for _, want := range []string{"1. tabs:", "2. ✗", "nonesuch", "3. capture: ยังไม่ได้ทำ"} {
		if !strings.Contains(out.Content, want) {
			t.Errorf("the batch answer must carry %q, got:\n%s", want, out.Content)
		}
	}
	if out.Success {
		t.Error("a batch with a failing step is not a success")
	}
	if out.RawOutput != out.Content {
		t.Error("RawOutput drifted from Content")
	}
}

// A profile that may not type cannot type through a batch.
func TestStepsPassEveryStepThroughTheGate(t *testing.T) {
	app := hostWithTabs(t, "web-agent-1", []string{"web-agent-1"}, "web-agent-1")
	narrowed := (&browserSkill{app: app}).Narrow([]string{"browser_read", "browser_tabs"}).(*browserSkill)
	out, err := narrowed.run(t.Context(), map[string]any{"steps": []any{
		map[string]any{"action": "tabs", "act": "list"},
		map[string]any{"action": "type", "text": "x"},
	}})
	if err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("a step this profile may not use must be refused as one, got %v\n%s", err, out.Content)
	}
	if !strings.Contains(out.Content, "1. tabs:") {
		t.Errorf("the step before the refusal still answers, got:\n%s", out.Content)
	}
}

// The batch is described once, as a signature, and taught once, as guidance.
func TestStepsAreInTheBlockAndTheGuidance(t *testing.T) {
	def := (&browserSkill{}).ToolDefinition()
	if !strings.Contains(def.Function.Description, "`steps`") {
		t.Error("the block does not offer steps")
	}
	g := (&browserSkill{}).Guidance(map[string]any{"steps": []any{map[string]any{"action": "read"}}})
	if !strings.Contains(g, "find") || !strings.Contains(g, "first step that fails") {
		t.Errorf("the batch guidance must teach aiming by text and the stop rule, got: %s", g)
	}
}

// The batch stops when the page moves out from under a step that cannot see
// that it moved.
//
// `tabs select` is the move used here because it is the one navigation that
// happens for real without a page: it changes which document every later step
// works, which is exactly what makes ref 3 mean something else. The click
// after it must not be attempted — a stale ref does not fail, it presses
// whatever now carries that number and reports success, and a batch that does
// that is worse than one that stops.
func TestStepsStopWhenThePageMovesUnderARef(t *testing.T) {
	app := hostWithTabs(t, "web-agent-1", []string{"web-agent-1", "web-agent-2"},
		"web-agent-1", "web-agent-2")
	browser := &browserSkill{app: app}
	out, err := browser.run(t.Context(), map[string]any{"steps": []any{
		map[string]any{"action": "tabs", "act": "select", "id": "web-agent-2"},
		map[string]any{"action": "click", "ref": 3},
		map[string]any{"action": "read"},
	}})
	if err != nil {
		t.Fatalf("stopping is a report, not a failure: %v\n%s", err, out.Content)
	}
	if !out.Success {
		t.Error("the steps that ran, ran — the batch is not a failure")
	}
	for _, want := range []string{"1. tabs:", "2. click: ยังไม่ได้ทำ", "3. read: ยังไม่ได้ทำ", "หยุดก่อนขั้นที่ 2"} {
		if !strings.Contains(out.Content, want) {
			t.Errorf("the answer must carry %q, got:\n%s", want, out.Content)
		}
	}
}

// A step that does not aim at the old page goes on. Stopping every batch at
// every navigation would take back most of what the batch is for: open → read
// is two steps and one call, and the whole point is that the read wants the
// page the open arrived at.
func TestStepsThatAimAtNothingRunOnAfterThePageMoves(t *testing.T) {
	app := hostWithTabs(t, "web-agent-1", []string{"web-agent-1", "web-agent-2"},
		"web-agent-1", "web-agent-2")
	browser := &browserSkill{app: app}
	out, err := browser.run(t.Context(), map[string]any{"steps": []any{
		map[string]any{"action": "tabs", "act": "select", "id": "web-agent-2"},
		map[string]any{"action": "tabs", "act": "list"},
	}})
	if err != nil {
		t.Fatalf("nothing here fails: %v\n%s", err, out.Content)
	}
	if strings.Contains(out.Content, "หยุดก่อนขั้น") {
		t.Errorf("a step aimed at no page must not be stopped, got:\n%s", out.Content)
	}
	if !strings.Contains(out.Content, "2. tabs:") {
		t.Errorf("the second step must have run, got:\n%s", out.Content)
	}
}

// What counts as aiming at a page that may be gone. `find` is the one aim
// that survives, because it is resolved against whatever page it lands on.
func TestStepAimsBlind(t *testing.T) {
	for _, c := range []struct {
		name string
		step any
		want bool
	}{
		{"a ref", map[string]any{"action": "click", "ref": 3}, true},
		{"a quoted ref", map[string]any{"action": "click", "ref": "3"}, true},
		{"a point", map[string]any{"action": "click", "x": 100, "y": 200}, true},
		{"a drag's far end", map[string]any{"action": "drag", "find": "card", "toX": 10, "toY": 20}, true},
		{"text", map[string]any{"action": "click", "find": "ตกลง"}, false},
		{"text with a dead ref alongside", map[string]any{"action": "click", "find": "ตกลง", "ref": 3}, false},
		{"no aim at all", map[string]any{"action": "read"}, false},
		{"a scroll with no target", map[string]any{"action": "scroll", "to": "bottom"}, false},
		{"not an object", "read", false},
	} {
		if got := stepAimsBlind(c.step); got != c.want {
			t.Errorf("stepAimsBlind(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

// The batch is taught by the actions that lead into it, not by itself.
//
// This is the trap the guidance file already diagnosed for `wait` and for
// `tabs`: advice keyed to an action arrives on the first use of that action,
// so advice about batching that is keyed to `steps` reaches only a model that
// already batches. The trigger has to ride on what every browser session
// calls.
func TestTheBatchIsTaughtBeforeItIsUsed(t *testing.T) {
	browser := &browserSkill{}
	for _, action := range []string{"open", "click"} {
		g := browser.Guidance(map[string]any{"action": action})
		if !strings.Contains(g, "`steps`") {
			t.Errorf("the guidance for %s never mentions the batch, so nobody hears about it: %s", action, g)
		}
	}
}
