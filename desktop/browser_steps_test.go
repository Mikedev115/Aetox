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
