package turn

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/hook"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/safety"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// echoDispatcher answers every tool call with a fixed, successful result, so a
// test can see exactly what the hooks added on top of it.
type echoDispatcher struct{}

func (echoDispatcher) Execute(context.Context, string) (skill.Output, bool, error) {
	return skill.Output{}, false, nil
}

func (echoDispatcher) ToolDefinitions() []model.ToolDefinition { return nil }

func (echoDispatcher) ExecuteTool(_ context.Context, name string, _ map[string]any) (skill.Output, bool, error) {
	return skill.Output{Name: name, Content: "edited", RawOutput: "edited", Success: true}, true, nil
}

// A PostToolUse hook that ran the tests after an edit has a verdict, and the
// model is the one reader who can act on it. It rides in the receipt under its
// own key: beside the tool's output, never inside it, so the tool's own words
// stay exactly what the tool said.
func TestAfterHookVerdictRidesInTheReceipt(t *testing.T) {
	cmd := `echo "go test ./x: FAILED"`
	if runtime.GOOS == "windows" {
		cmd = `Write-Output "go test ./x: FAILED"`
	}
	hooks := hook.NewRunner(hook.Config{Hooks: []hook.Hook{{
		Event: hook.PostToolUse, Matcher: "edit", Command: cmd,
	}}}, t.TempDir())
	e := NewExecutor(ExecutorOptions{
		Agent:        &toolAwareAgent{},
		Dispatcher:   echoDispatcher{},
		Hooks:        hooks,
		ApprovalMode: safety.ApprovalFullAccess,
	})
	args := map[string]any{"path": "x/y.go", "replace": "z"}

	out, handled, err := e.executeTool(context.Background(), "edit", args)
	if err != nil || !handled {
		t.Fatalf("executeTool: handled=%v err=%v", handled, err)
	}
	if out.AfterHook != "go test ./x: FAILED" {
		t.Fatalf("AfterHook = %q, want the hook's verdict", out.AfterHook)
	}
	if out.RawOutput != "edited" || out.Content != "edited" {
		t.Errorf("the tool's own output was touched: raw=%q content=%q", out.RawOutput, out.Content)
	}

	var receipt map[string]any
	if err := json.Unmarshal([]byte(e.modelToolReceipt(context.Background(), "edit", args, out, nil)), &receipt); err != nil {
		t.Fatalf("receipt is not JSON: %v", err)
	}
	if receipt["after_hook"] != "go test ./x: FAILED" {
		t.Errorf("receipt after_hook = %v, want the verdict the model has to read", receipt["after_hook"])
	}
	if receipt["output"] != "edited" {
		t.Errorf("receipt output = %v, want the tool's own words untouched", receipt["output"])
	}
}

// A blocking PostToolUse hook rejects the result: the tool's own words stay
// exactly what the tool said, but the call reports failed and the receipt the
// model reads turns into an error carrying the hook's reason, so the model
// fixes the work instead of moving on having believed it done.
func TestBlockingPostToolUseHookRejectsTheResult(t *testing.T) {
	cmd := `echo "lint failed"; exit 1`
	if runtime.GOOS == "windows" {
		cmd = `Write-Output "lint failed"; exit 1`
	}
	hooks := hook.NewRunner(hook.Config{Hooks: []hook.Hook{{
		Event: hook.PostToolUse, Matcher: "*", Blocking: true, Command: cmd,
	}}}, t.TempDir())
	e := NewExecutor(ExecutorOptions{
		Agent:        &toolAwareAgent{},
		Dispatcher:   echoDispatcher{},
		Hooks:        hooks,
		ApprovalMode: safety.ApprovalFullAccess,
	})
	args := map[string]any{"path": "x/y.go"}

	out, handled, err := e.executeTool(context.Background(), "edit", args)
	if err != nil || !handled {
		t.Fatalf("executeTool: handled=%v err=%v", handled, err)
	}
	if out.Success {
		t.Error("a result a blocking hook rejected still reported success")
	}
	if !strings.Contains(out.Stderr, "rejected by a PostToolUse hook: lint failed") {
		t.Errorf("Stderr = %q, want the rejection with the hook's reason first", out.Stderr)
	}
	if out.Content != "edited" || out.RawOutput != "edited" {
		t.Errorf("the tool's own output was touched: raw=%q content=%q", out.RawOutput, out.Content)
	}

	var receipt map[string]any
	if err := json.Unmarshal([]byte(e.modelToolReceipt(context.Background(), "edit", args, out, nil)), &receipt); err != nil {
		t.Fatalf("receipt is not JSON: %v", err)
	}
	if receipt["status"] != string(TurnStatusError) {
		t.Errorf("receipt status = %v, want error for a rejected result", receipt["status"])
	}
	if receipt["after_hook"] != "lint failed" {
		t.Errorf("receipt after_hook = %v, want the hook's reason", receipt["after_hook"])
	}
	if receipt["output"] != "edited" {
		t.Errorf("receipt output = %v, want the tool's own words untouched", receipt["output"])
	}
}

// Hooks that say nothing add nothing: no key, no tokens.
func TestSilentAfterHookLeavesTheReceiptAlone(t *testing.T) {
	cmd := `true`
	if runtime.GOOS == "windows" {
		cmd = `exit 0`
	}
	hooks := hook.NewRunner(hook.Config{Hooks: []hook.Hook{{
		Event: hook.PostToolUse, Matcher: "*", Command: cmd,
	}}}, t.TempDir())
	e := NewExecutor(ExecutorOptions{
		Agent:        &toolAwareAgent{},
		Dispatcher:   echoDispatcher{},
		Hooks:        hooks,
		ApprovalMode: safety.ApprovalFullAccess,
	})
	args := map[string]any{"path": "x/y.go"}
	out, _, _ := e.executeTool(context.Background(), "write", args)
	if out.AfterHook != "" {
		t.Fatalf("AfterHook = %q for a silent hook", out.AfterHook)
	}
	var receipt map[string]any
	_ = json.Unmarshal([]byte(e.modelToolReceipt(context.Background(), "write", args, out, nil)), &receipt)
	if _, present := receipt["after_hook"]; present {
		t.Error("a silent hook still put an after_hook key in the receipt")
	}
}
