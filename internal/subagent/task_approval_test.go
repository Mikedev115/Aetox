package subagent

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Mikedev115/Aetox/internal/safety"
)

// The gate a delegate runs under is the one the user has NOW, not the one the
// options were built with.
//
// Found on 2026-09-14 by running the coding desk under unsafe-only with the
// shell refused: the model sent a helper to run the same `go test`, the helper
// ran it at full access without a question, and the answer said so out loud.
// The parent executor's mode had been switched after bootstrap; the task
// tool's copy had not. This walks that exact sequence with the scripted
// provider: full access, a delegate writes a file and nobody is asked; the
// mode moves to ask; a second delegate's write must come to the approver.
func TestDelegateRunsUnderTheApprovalModeTheUserHasNow(t *testing.T) {
	var mu sync.Mutex
	var asked []string
	mode := safety.ApprovalFullAccess

	f := newTaskFixtureWith(t, "aetox-subagent:test", func(opts *TaskOptions) {
		opts.CurrentApprovalMode = func() safety.ApprovalMode { return mode }
		opts.Approve = func(_ context.Context, command, _ string) (bool, error) {
			mu.Lock()
			asked = append(asked, command)
			mu.Unlock()
			return false, nil
		}
	})
	hire := func(id string) {
		t.Helper()
		start := f.callTask(t, id, map[string]any{
			"description": "write the summary",
			"prompt":      "สำรวจโฟลเดอร์แล้วเขียนไฟล์สรุป",
			"agent":       "general",
		})
		if !start.Success {
			t.Fatalf("start failed: %s", start.Stderr)
		}
		f.collect(t, taskIDOf(t, start))
	}

	hire("call_full")
	mu.Lock()
	before := len(asked)
	mu.Unlock()
	if before != 0 {
		t.Fatalf("at full access the delegate asked %d times: %q", before, asked)
	}

	mode = safety.ApprovalAsk
	hire("call_ask")
	mu.Lock()
	defer mu.Unlock()
	if len(asked) == 0 {
		t.Fatal("mode switched to ask after the tools were built, and the delegate's write still ran unasked")
	}
	if !strings.Contains(strings.Join(asked, "\n"), "write") {
		t.Errorf("the approver was asked, but not for the write: %q", asked)
	}
}

// Nil keeps the static reading: a host that never switches modes builds the
// tools once and every delegate runs under that mode, as before.
func TestDelegateFallsBackToTheStaticModeWithoutALiveReader(t *testing.T) {
	tool := &taskTool{opts: TaskOptions{ApprovalMode: safety.ApprovalUnsafeOnly}}
	if got := tool.approvalModeNow(); got != safety.ApprovalUnsafeOnly {
		t.Fatalf("approvalModeNow = %q, want the static unsafe-only", got)
	}
	tool.opts.CurrentApprovalMode = func() safety.ApprovalMode { return safety.ApprovalAsk }
	if got := tool.approvalModeNow(); got != safety.ApprovalAsk {
		t.Fatalf("approvalModeNow = %q, want the live ask", got)
	}
}
