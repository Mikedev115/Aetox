package bootstrap

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Mikedev115/Aetox/internal/safety"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// The road the desktop dropdown and the CLI's /approval take: App.SetApprovalMode
// after the engine is up. The parent executor always followed it; the delegate
// tools were built with the boot-time mode and did not (2026-09-14, found by a
// model that sent a helper to run the shell its own gate had just refused).
// This is the wiring pin: switch after boot, hire a delegate through the same
// registry the model uses, and the delegate's write must ask.
func TestSwitchingTheApprovalModeAfterBootReachesDelegates(t *testing.T) {
	var mu sync.Mutex
	var asked []string
	cfg := testConfig(t)
	cfg.ModelName = "aetox-subagent:test"
	cfg.ApprovalMode = string(safety.ApprovalFullAccess)
	res, err := Engine(cfg, Options{Approve: func(_ context.Context, command, _ string) (bool, error) {
		mu.Lock()
		asked = append(asked, command)
		mu.Unlock()
		return false, nil
	}})
	if err != nil {
		t.Fatalf("Engine: %v", err)
	}
	dispatcher := skill.NewDispatcher(res.Registry)
	hire := func(id string) {
		t.Helper()
		ctx := turn.WithCallID(context.Background(), id)
		start, handled, err := dispatcher.ExecuteTool(ctx, "task", map[string]any{
			"description": "write the summary", "prompt": "สำรวจโฟลเดอร์แล้วเขียนไฟล์สรุป", "agent": "general",
		})
		if !handled || err != nil || !start.Success {
			t.Fatalf("task start: handled=%v err=%v out=%s %s", handled, err, start.Content, start.Stderr)
		}
		var taskID string
		for _, word := range strings.Fields(strings.ReplaceAll(start.Content, "\"", " ")) {
			if strings.HasPrefix(word, "task_") {
				taskID = strings.Trim(word, ".,")
			}
		}
		if taskID == "" {
			t.Fatalf("no task id in %q", start.Content)
		}
		if _, _, err := dispatcher.ExecuteTool(ctx, "task", map[string]any{"action": "collect", "task_id": taskID}); err != nil {
			t.Fatalf("collect: %v", err)
		}
	}

	hire("call_full")
	mu.Lock()
	before := len(asked)
	mu.Unlock()
	if before != 0 {
		t.Fatalf("at full access the delegate asked %d times: %q", before, asked)
	}

	res.App.SetApprovalMode(safety.ApprovalAsk)
	hire("call_ask")
	mu.Lock()
	defer mu.Unlock()
	if len(asked) == 0 {
		t.Fatal("SetApprovalMode(ask) after boot moved the parent's gate only; the delegate's write still ran unasked")
	}
	if !strings.Contains(strings.Join(asked, "\n"), "write") {
		t.Errorf("the approver was asked, but not for the write: %q", asked)
	}
}
