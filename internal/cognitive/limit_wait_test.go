package cognitive

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// What the ChatGPT backend says when a Codex plan's five-hour window is spent,
// wrapped the way responses.go wraps it: the sentence, and the reset as a fact.
func planSpentUntil(resetAt time.Time) error {
	return &model.RateLimitError{
		Provider: "codex",
		ResetAt:  resetAt,
		Err:      fmt.Errorf("codex: the plus plan's limit is used up. It resets in %s.", time.Until(resetAt).Round(time.Second)),
	}
}

func withLimitWaits(t *testing.T, listener func(turn.LimitWait)) turn.TurnOptions {
	t.Helper()
	old := limitWaitSlack
	limitWaitSlack = 10 * time.Millisecond
	t.Cleanup(func() { limitWaitSlack = old })
	return turn.TurnOptions{OnLimitWait: listener}
}

func toolTurn(ctx context.Context, a *Agent, msg string, opts turn.TurnOptions) (string, error) {
	reply, _, err := a.RespondWithTools(
		ctx,
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		msg,
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "", nil, nil },
		nil,
		opts,
	)
	return reply, err
}

// The owner's ask, 14 ก.ย. 2026: the five-hour window runs out mid-run, the app
// says it is waiting, and when the window refills the run carries on — with
// the tool results it had, not from a blank retry.
func TestASpentPlanWindowIsWaitedOutAndTheRunResumes(t *testing.T) {
	callsTool := func(model.Request) (model.Response, error) {
		return model.Response{ToolCalls: []model.ToolCall{{
			ID: "1", Type: "function",
			Function: model.FunctionCall{Name: "read", Arguments: `{"path":"a.md"}`},
		}}}, nil
	}
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		callsTool,
		func(model.Request) (model.Response, error) {
			return model.Response{}, planSpentUntil(time.Now().Add(20 * time.Millisecond))
		},
		func(model.Request) (model.Response, error) {
			return model.Response{Text: "เสร็จแล้วครับ"}, nil
		},
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	var heard []turn.LimitWait
	reply, err := toolTurn(context.Background(), agent, "ทำต่อให้จบ",
		withLimitWaits(t, func(w turn.LimitWait) { heard = append(heard, w) }))
	if err != nil {
		t.Fatalf("a spent window still ends the turn: %v", err)
	}
	if reply != "เสร็จแล้วครับ" {
		t.Errorf("reply = %q, want the answer from after the wait", reply)
	}
	if len(provider.seen) != 3 {
		t.Fatalf("provider was called %d times, want 3 (tool round, refusal, resumed round)", len(provider.seen))
	}
	// The wait is announced, then ended — the window draws the countdown from
	// the first and clears it on the second.
	if len(heard) != 2 || !heard[0].Waiting || heard[1].Waiting {
		t.Fatalf("listener heard %+v, want a start then an end", heard)
	}
	if heard[0].Provider != "codex" || heard[0].ResetAt.IsZero() {
		t.Errorf("the start names no provider or reset: %+v", heard[0])
	}
	// Resumed, not restarted: the tool round's result is still in what the
	// model is asked after the wait.
	var toolResults int
	for _, msg := range provider.seen[2].Messages {
		if msg.Role == model.RoleTool {
			toolResults++
		}
	}
	if toolResults == 0 {
		t.Error("the resumed round carried no tool results — the run was restarted, not resumed")
	}
}

// Nobody watching, nobody waiting. A delegate's rounds are not on screen, and
// a terminal that has not wired a countdown must not sit silent for hours —
// without a listener the turn ends with the provider's own sentence, exactly
// as before.
func TestNoListenerMeansNoWait(t *testing.T) {
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		func(model.Request) (model.Response, error) {
			return model.Response{}, planSpentUntil(time.Now().Add(time.Hour))
		},
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	start := time.Now()
	_, err := toolTurn(context.Background(), agent, "ทำต่อให้จบ", turn.TurnOptions{})
	if err == nil {
		t.Fatal("the turn reported success on a refusal nobody could wait out")
	}
	var limit *model.RateLimitError
	if !errors.As(err, &limit) {
		t.Errorf("error = %v, want the provider's own RateLimitError to reach the caller", err)
	}
	if len(provider.seen) != 1 || time.Since(start) > time.Second {
		t.Errorf("provider was called %d times over %s; want once, immediately", len(provider.seen), time.Since(start))
	}
}

// A weekly window that says "resets in four days" is not something a chat can
// wait for. Past the ceiling the sentence is the right ending.
func TestAResetBeyondTheCeilingIsNotWaitedFor(t *testing.T) {
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		func(model.Request) (model.Response, error) {
			return model.Response{}, planSpentUntil(time.Now().Add(maxLimitWait + time.Hour))
		},
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	var heard int
	_, err := toolTurn(context.Background(), agent, "ทำต่อให้จบ",
		withLimitWaits(t, func(turn.LimitWait) { heard++ }))
	if err == nil {
		t.Fatal("a four-day wait was started")
	}
	if heard != 0 {
		t.Errorf("listener heard %d events for a wait that must not happen", heard)
	}
}

// Stop during the wait is Stop, not a failure: the desktop tells the two apart
// by the error, and would otherwise draw a retry box over a button the user
// chose to press.
func TestStopDuringTheWaitEndsAsStop(t *testing.T) {
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		func(model.Request) (model.Response, error) {
			return model.Response{}, planSpentUntil(time.Now().Add(time.Hour))
		},
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ended bool
	_, err := toolTurn(ctx, agent, "ทำต่อให้จบ", turn.TurnOptions{OnLimitWait: func(w turn.LimitWait) {
		if w.Waiting {
			cancel()
		} else {
			ended = true
		}
	}})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if !ended {
		t.Error("the wait's end was not reported after Stop")
	}
}

// The plain route — a model that cannot call tools — waits the same way.
func TestTheToolLessRouteWaitsOutTheWindowToo(t *testing.T) {
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		func(model.Request) (model.Response, error) {
			return model.Response{}, planSpentUntil(time.Now().Add(20 * time.Millisecond))
		},
		func(model.Request) (model.Response, error) { return model.Response{Text: "ตอบแล้ว"}, nil },
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	var heard int
	reply, err := agent.Respond(context.Background(), "สวัสดี", withLimitWaits(t, func(turn.LimitWait) { heard++ }))
	if err != nil {
		t.Fatalf("Respond: %v", err)
	}
	if reply != "ตอบแล้ว" || heard != 2 {
		t.Errorf("reply = %q, heard = %d; want the resumed answer and a start+end", reply, heard)
	}
}
