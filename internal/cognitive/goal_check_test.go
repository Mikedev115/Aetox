package cognitive

import (
	"context"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/think"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// มุ่งเป้า, and the reason it is a hook rather than an engine.
//
// §106.10 declined this mode on 2026-08-14, and one of the two objections was
// about shape: it "filters nothing and adds a control loop in the turn executor,
// with a verification call every round and a cost that multiplies". That was
// true of the design on the table then. What the loop already had — and nobody
// had looked at in this light — is a branch where a turn that was going to end
// is kept alive: the one an interjection takes when the user types under a
// finished answer. This is that branch with a different author.
//
// So these tests are about the seam, not about any particular checker. What
// belongs to the caller — who checks, how, and when to give up — is deliberately
// not in this package, exactly as what a tool DOES is not.

// A turn the checker sends back keeps working, and the answer it was about to
// hand over is written down before the verdict is. Skipping that would leave the
// next round arguing with a verdict about an answer missing from its own
// history.
func TestAGoalCheckCanSendTheTurnBack(t *testing.T) {
	provider := &toolLoopProvider{responses: []model.Response{
		{Text: "I have done the first step."},
		{Text: "Now every step is done."},
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxToolCalls: 6})

	asked := 0
	answers := []string{}
	final, _, err := agent.RespondWithTools(
		context.Background(),
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		"carry out the plan",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "", nil, nil },
		nil,
		turn.TurnOptions{
			ThinkLevel: think.LevelMedium,
			OnGoalCheck: func(answer string) string {
				asked++
				answers = append(answers, answer)
				if asked == 1 {
					return "Step 2 of the plan has not been done. Keep going."
				}
				return ""
			},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asked != 2 {
		t.Fatalf("the check ran %d times; it must run once per attempt to finish", asked)
	}
	if final != "Now every step is done." {
		t.Errorf("the turn ended on %q rather than on the round the check let through", final)
	}
	// Asked about what the model SAID, so a check can be about the answer and
	// not only about what ran.
	if answers[0] != "I have done the first step." {
		t.Errorf("the check was handed %q rather than the answer about to be given", answers[0])
	}

	// The order in context: the demoted answer, then the verdict as a user turn.
	var roles []string
	var texts []string
	for _, m := range agent.ContextMessages() {
		roles = append(roles, string(m.Role))
		texts = append(texts, m.Content)
	}
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "I have done the first step.") {
		t.Error("the answer the check sent back was never written into the history it argues with")
	}
	verdictAt, answerAt := -1, -1
	for i, txt := range texts {
		if strings.Contains(txt, "Step 2 of the plan has not been done") {
			verdictAt = i
		}
		if txt == "I have done the first step." {
			answerAt = i
		}
	}
	if answerAt < 0 || verdictAt < 0 || answerAt > verdictAt {
		t.Errorf("the verdict must follow the answer it is about: answer at %d, verdict at %d", answerAt, verdictAt)
	}
	if roles[verdictAt] != string(model.RoleUser) {
		t.Errorf("the verdict went in as %q; it is somebody telling the model it is not done, which is a user turn", roles[verdictAt])
	}
}

// A turn with no check set behaves exactly as it did before this existed. The
// hook is nil for every stance but one, so this is the path nearly every turn in
// the app takes.
func TestWithoutAGoalCheckTheTurnEndsWhereItAlwaysDid(t *testing.T) {
	provider := &toolLoopProvider{responses: []model.Response{{Text: "done"}}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxToolCalls: 4})

	final, _, err := agent.RespondWithTools(
		context.Background(),
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		"answer me",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "", nil, nil },
		nil,
		turn.TurnOptions{ThinkLevel: think.LevelMedium},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if final != "done" {
		t.Errorf("the turn ended on %q", final)
	}
	if len(provider.requests) != 1 {
		t.Errorf("a turn with no check made %d provider calls; it should have made one", len(provider.requests))
	}
}

// THE CEILING IS NOT OPTIONAL. §106.10 names it as one of the four parts this
// mode cannot be built without, and it is the one the hook itself must not be
// trusted to supply: a checker that never says yes is exactly the failure mode a
// long unattended run has, and it is indistinguishable from a hard question.
//
// MaxToolCalls is that ceiling and it already existed — it bounds the loop
// whatever keeps it alive, an interjection and a verdict alike. This pins that
// the goal check cannot outrun it, which is what makes the whole mode safe to
// hand to somebody who has walked away.
func TestAGoalCheckThatNeverSaysYesStillHitsTheCeiling(t *testing.T) {
	responses := make([]model.Response, 0, 20)
	for i := 0; i < 20; i++ {
		responses = append(responses, model.Response{Text: "still working"})
	}
	provider := &toolLoopProvider{responses: responses}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxToolCalls: 5})

	asked := 0
	_, _, err := agent.RespondWithTools(
		context.Background(),
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		"carry out the plan",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "", nil, nil },
		nil,
		turn.TurnOptions{
			ThinkLevel:  think.LevelMedium,
			OnGoalCheck: func(string) string { asked++; return "not done yet" },
		},
	)
	// Whether it comes back as an error or as the last answer does not matter
	// here and is the loop's own business. What matters is that it STOPPED.
	_ = err
	if asked > 5 {
		t.Fatalf("the check ran %d times against a ceiling of 5 — a checker that never says yes would run forever", asked)
	}
	if len(provider.requests) > 5 {
		t.Fatalf("%d provider calls against a ceiling of 5", len(provider.requests))
	}
}

// A cancelled turn does not get sent back. Stop is the brake on this mode as it
// is on every other, and a check that ran after the user pressed it would keep a
// turn alive past the moment they ended it.
func TestACancelledTurnIsNotSentBack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &toolLoopProvider{
		responses: []model.Response{{Text: "half done"}, {Text: "more"}},
		beforeReply: func(call int) {
			if call == 1 {
				cancel()
			}
		},
	}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxToolCalls: 6})

	asked := 0
	_, _, _ = agent.RespondWithTools(
		ctx,
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		"carry out the plan",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "", nil, nil },
		nil,
		turn.TurnOptions{
			ThinkLevel:  think.LevelMedium,
			OnGoalCheck: func(string) string { asked++; return "keep going" },
		},
	)
	if len(provider.requests) > 2 {
		t.Errorf("the turn made %d calls after being cancelled", len(provider.requests))
	}
}
