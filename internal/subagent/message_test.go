package subagent

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

type finishingBoundaryProvider struct {
	started chan struct{}
	release chan struct{}
	calls   int
}

func (*finishingBoundaryProvider) Name() string { return "message-boundary-test" }

func (p *finishingBoundaryProvider) Complete(ctx context.Context, req model.Request) (model.Response, error) {
	p.calls++
	if p.calls == 1 {
		close(p.started)
		select {
		case <-p.release:
		case <-ctx.Done():
			return model.Response{}, ctx.Err()
		}
		return model.Response{Text: "first answer before the update"}, nil
	}
	last := req.Messages[len(req.Messages)-1].Content
	return model.Response{Text: "continued with: " + last}, nil
}

func TestMessageSteersARunningDelegateAndRefusesAFinishedOne(t *testing.T) {
	runner := NewDelegations()
	steered := make(chan string, 1)
	release := make(chan struct{})
	task := runner.start(delegation{
		profile: "explore",
		label:   "inspect",
		steer: func(text string) {
			steered <- text
		},
	}, func(context.Context, *runningTask) skill.Output {
		<-release
		return skill.Output{Name: "task_result", Content: "done", Success: true}
	})

	if err := runner.message(task.id, "  use the new filename  "); err != nil {
		t.Fatalf("message: %v", err)
	}
	select {
	case got := <-steered:
		if got != "use the new filename" {
			t.Fatalf("steered text = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("the running delegate never received the update")
	}

	close(release)
	select {
	case <-task.done:
	case <-time.After(time.Second):
		t.Fatal("delegate did not finish")
	}
	if err := runner.message(task.id, "too late"); err == nil || !strings.Contains(err.Error(), "finished") {
		t.Fatalf("message after finish = %v, want a finished-task refusal", err)
	}
}

func TestMessageDoesNotMasqueradeAsAnAskMainAnswer(t *testing.T) {
	runner := NewDelegations()
	release := make(chan struct{})
	task := runner.start(delegation{
		profile: "explore",
		label:   "inspect",
		steer:   func(string) {},
	}, func(context.Context, *runningTask) skill.Output {
		<-release
		return skill.Output{Name: "task_result", Content: "done", Success: true}
	})
	task.setParked(&pendingAsk{question: "which file?", answer: make(chan string, 1)})
	if err := runner.message(task.id, "config.prod.json"); err == nil || !strings.Contains(err.Error(), "action=answer") {
		t.Fatalf("message while ask_main is parked = %v, want direction to answer", err)
	}
	task.setParked(nil)
	close(release)
	<-task.done
}

func TestCollectYieldsToAParentInterjectionWithoutEndingTheTask(t *testing.T) {
	runner := NewDelegations()
	release := make(chan struct{})
	task := runner.start(delegation{profile: "explore", label: "inspect", steer: func(string) {}},
		func(context.Context, *runningTask) skill.Output {
			<-release
			return skill.Output{Name: "task_result", Content: "done", Success: true}
		})
	wake := make(chan struct{}, 1)
	wake <- struct{}{}
	tool := &taskResultTool{runner: runner, parentInterjection: wake}
	out, err := tool.ExecuteTool(context.Background(), map[string]any{"task_id": task.id})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if !out.Success || !strings.Contains(out.Content, "COLLECT PAUSED") {
		t.Fatalf("collect output = %#v, want a successful pause", out)
	}
	if task.finished() || task.wasCollected() {
		t.Fatal("yielding collect finished or collected the still-running task")
	}
	close(release)
	<-task.done
	collected, ask, interrupted, err := runner.collect(context.Background(), task.id, nil)
	if err != nil || ask != nil || interrupted || collected != task {
		t.Fatalf("collect after completion = task %p ask %v interrupted %v err %v", collected, ask, interrupted, err)
	}
}

func TestMessageAtTheFinishingBoundaryStartsAContinuationTurn(t *testing.T) {
	isolate(t)
	provider := &finishingBoundaryProvider{started: make(chan struct{}), release: make(chan struct{})}
	registry := skill.NewDefaultRegistry(skill.RegistryOptions{SandboxRoot: t.TempDir()})
	runner := NewDelegations()
	packed := NewTaskTools(TaskOptions{
		Provider: provider, Model: "message-boundary-test", Registry: registry, Delegations: runner,
	})[0].(*delegationTool)

	started, err := packed.ExecuteTool(context.Background(), map[string]any{
		"action": "start", "agent": "explore", "description": "boundary",
		"prompt": "finish a small job and report",
	})
	if err != nil || !started.Success {
		t.Fatalf("start = %#v, %v", started, err)
	}
	id := taskIDOf(t, started)
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("delegate never reached its first provider call")
	}

	msg, err := packed.ExecuteTool(context.Background(), map[string]any{
		"action": "message", "task_id": id, "message": "use config.prod.json instead",
	})
	if err != nil || !msg.Success {
		t.Fatalf("message = %#v, %v", msg, err)
	}
	close(provider.release)

	done, err := packed.ExecuteTool(context.Background(), map[string]any{"action": "collect", "task_id": id})
	if err != nil || !done.Success {
		t.Fatalf("collect = %#v, %v", done, err)
	}
	if provider.calls != 2 {
		t.Fatalf("provider calls = %d, want the original turn plus one continuation", provider.calls)
	}
	if !strings.Contains(done.Content, "use config.prod.json instead") {
		t.Fatalf("continued result lost the update: %q", done.Content)
	}
}

func TestTaskPackExposesMessageAsItsOwnPermission(t *testing.T) {
	tools := NewTaskTools(TaskOptions{})
	packed, ok := tools[0].(*delegationTool)
	if !ok {
		t.Fatalf("task tool = %T", tools[0])
	}
	if !slices.Contains(packed.Actions(), "task_message") {
		t.Fatalf("task actions = %v, missing task_message", packed.Actions())
	}
	schema := string(packed.ToolDefinition().Function.Parameters)
	for _, want := range []string{`"message"`, `"task_id"`} {
		if !strings.Contains(schema, want) {
			t.Fatalf("task schema missing %s: %s", want, schema)
		}
	}
}
