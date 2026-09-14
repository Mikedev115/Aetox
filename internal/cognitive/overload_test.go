package cognitive

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// What the Codex backend wrote into a customer's worker stream four times on
// 14 ก.ย. 2026, wrapped the way responses.go wraps it: the sentence, and the
// fact that it is the provider shedding load rather than refusing.
func codexOverloaded() error {
	return &model.ProviderOverloadedError{
		Provider: "codex",
		Err:      fmt.Errorf("codex: Our servers are currently overloaded. Please try again later."),
	}
}

// The backoff is seconds in production and the point of the tests is the
// bookkeeping, not the clock.
func withFastOverloadBackoff(t *testing.T) {
	t.Helper()
	old := overloadBackoff
	overloadBackoff = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { overloadBackoff = old })
}

// The customer's turn: the worker was hired, its round was built, and the
// backend said "overloaded" inside a 200 — past the transport's retry — and
// the turn ended on it. Now it is asked again, and the second ask is the same
// question, not a fresh one.
func TestAnOverloadedProviderIsAskedAgainOnTheSameRound(t *testing.T) {
	withFastOverloadBackoff(t)
	callsTool := func(model.Request) (model.Response, error) {
		return model.Response{ToolCalls: []model.ToolCall{{
			ID: "1", Type: "function",
			Function: model.FunctionCall{Name: "read", Arguments: `{"path":"a.md"}`},
		}}}, nil
	}
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		callsTool,
		func(model.Request) (model.Response, error) { return model.Response{}, codexOverloaded() },
		func(model.Request) (model.Response, error) { return model.Response{Text: "เสร็จแล้วครับ"}, nil },
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	// No listener, on purpose: the customer's case was a worker nobody was
	// watching, and the retry must not depend on a window being there.
	reply, err := toolTurn(context.Background(), agent, "ทดสอบ Worker", turn.TurnOptions{})
	if err != nil {
		t.Fatalf("an overload still ends the turn: %v", err)
	}
	if reply != "เสร็จแล้วครับ" {
		t.Errorf("reply = %q, want the answer from after the backoff", reply)
	}
	if len(provider.seen) != 3 {
		t.Fatalf("provider was called %d times, want 3 (tool round, overload, resumed round)", len(provider.seen))
	}
	// The resumed round is the failed round asked again: same messages, tool
	// result included. A retry that lost the tool result would be a fresh turn
	// wearing the old one's task ID.
	failed, resumed := provider.seen[1].Messages, provider.seen[2].Messages
	if len(failed) != len(resumed) {
		t.Fatalf("the resumed round sent %d messages, the failed round sent %d — the retry rebuilt the round", len(resumed), len(failed))
	}
	for i := range failed {
		if failed[i].Content != resumed[i].Content {
			t.Errorf("message %d differs between the failed round and its retry:\n failed: %q\nresumed: %q", i, failed[i].Content, resumed[i].Content)
		}
	}
}

// Bounded, and the provider's own sentence is what the bound ends with — it
// said "try again later", and after three tries in twenty seconds that is
// what the user should read.
func TestAProviderThatStaysOverloadedEndsWithItsOwnSentence(t *testing.T) {
	withFastOverloadBackoff(t)
	always := func(model.Request) (model.Response, error) { return model.Response{}, codexOverloaded() }
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){always, always, always, always, always}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	_, err := toolTurn(context.Background(), agent, "ทดสอบ Worker", turn.TurnOptions{})
	if err == nil {
		t.Fatal("a provider that never recovered reported success")
	}
	if !strings.Contains(err.Error(), "currently overloaded") {
		t.Errorf("error = %v, want the provider's own words kept", err)
	}
	if len(provider.seen) != 1+maxOverloadRetries {
		t.Errorf("provider was called %d times, want %d (the first ask plus its retries)", len(provider.seen), 1+maxOverloadRetries)
	}
}

// The hold is told to whoever is listening, on the rate-limit row, marked so
// the row can say "overloaded" rather than "hit its limit" — the customer's
// report went looking for a quota problem that was never there.
func TestAnOverloadHoldIsReportedAsOneAndNotAsALimit(t *testing.T) {
	withFastOverloadBackoff(t)
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		func(model.Request) (model.Response, error) { return model.Response{}, codexOverloaded() },
		func(model.Request) (model.Response, error) { return model.Response{Text: "ตอบ"}, nil },
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	var heard []turn.LimitWait
	if _, err := toolTurn(context.Background(), agent, "ทดสอบ", turn.TurnOptions{
		OnLimitWait: func(w turn.LimitWait) { heard = append(heard, w) },
	}); err != nil {
		t.Fatal(err)
	}
	if len(heard) != 2 || !heard[0].Waiting || heard[1].Waiting {
		t.Fatalf("listener heard %+v, want a start then an end", heard)
	}
	start := heard[0]
	if !start.Overloaded || start.Provider != "codex" || start.Attempt != 1 || start.Of != maxOverloadRetries {
		t.Errorf("the hold was reported as %+v, want overloaded, codex, attempt 1 of %d", start, maxOverloadRetries)
	}
	if start.ResetAt.IsZero() || start.ResetAt.After(time.Now().Add(time.Second)) {
		t.Errorf("ResetAt = %v, want the moment of the next ask, seconds away at most", start.ResetAt)
	}
	if !heard[1].Overloaded {
		t.Errorf("the end was reported as %+v, want it marked overloaded like its start", heard[1])
	}
}

// Stop during the backoff ends the turn, the way it does during every other
// hold. A backoff that outlived the user's Stop would be the app looking hung
// for the seconds it took to notice.
func TestStopDuringAnOverloadBackoffEndsTheTurn(t *testing.T) {
	old := overloadBackoff
	overloadBackoff = []time.Duration{time.Hour}
	t.Cleanup(func() { overloadBackoff = old })
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		func(model.Request) (model.Response, error) { return model.Response{}, codexOverloaded() },
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	done := make(chan error, 1)
	go func() { _, err := toolTurn(ctx, agent, "ทดสอบ", turn.TurnOptions{}); done <- err }()
	select {
	case err := <-done:
		if err == nil || ctx.Err() == nil {
			t.Errorf("turn ended with %v, want the cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the turn sat in its backoff through a Stop")
	}
}

// The plain path — no tools on the table — goes through completeWithReconnect
// and gets the same rule, because a worker whose profile carries no tools is
// still a worker the backend can shed.
func TestAnOverloadOnAPlainAnswerIsAskedAgainToo(t *testing.T) {
	withFastOverloadBackoff(t)
	provider := &scriptedProvider{steps: []func(model.Request) (model.Response, error){
		func(model.Request) (model.Response, error) { return model.Response{}, codexOverloaded() },
		func(model.Request) (model.Response, error) { return model.Response{Text: "ตอบ"}, nil },
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "gpt-5.6-luna", SystemPrompt: "sys", MaxChars: 1_000_000})

	reply, err := agent.Respond(context.Background(), "สวัสดี", turn.TurnOptions{})
	if err != nil {
		t.Fatalf("an overload on a plain answer still ends the turn: %v", err)
	}
	if reply != "ตอบ" || len(provider.seen) != 2 {
		t.Errorf("reply = %q after %d calls, want the answer from the second ask", reply, len(provider.seen))
	}
}

// The backoff has a shape, and the shape is the reason it is not the drop's
// one-second-per-attempt: a base that grows, and a jitter on top of it so a
// thousand clients told "overloaded" in the same second do not all ask again
// in the same second.
func TestOverloadWaitGrowsAndJitters(t *testing.T) {
	for attempt, base := range overloadBackoff {
		for range 50 {
			wait := overloadWait(attempt)
			if wait < base || wait > base+base/2 {
				t.Fatalf("overloadWait(%d) = %v, want within [%v, %v]", attempt, wait, base, base+base/2)
			}
		}
	}
	last := overloadBackoff[len(overloadBackoff)-1]
	if wait := overloadWait(99); wait < last {
		t.Errorf("overloadWait past the table = %v, want at least the last base %v", wait, last)
	}
}
