package turn

import (
	"context"
	"testing"

	"github.com/Mikedev115/Aetox/internal/command"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/think"
)

// dialWatchingAgent plays the tool loop: it reads the depth dial the way
// cognitive.Agent.buildRequest does, once per "round", and between two rounds
// it lets the test press the dial from the UI side.
type dialWatchingAgent struct {
	press func()
	seen  []think.Level
}

func (a *dialWatchingAgent) SupportsToolCalling() bool { return true }

func (a *dialWatchingAgent) RespondWithTools(
	_ context.Context, _ []model.ToolDefinition, _ string,
	_ func(context.Context, model.ToolCall) (string, []model.Image, error),
	_ func(string) error, opts TurnOptions,
) (string, bool, error) {
	a.seen = append(a.seen, opts.EffectiveThinkLevel()) // round 1's request
	a.press()
	a.seen = append(a.seen, opts.EffectiveThinkLevel()) // round 2's request
	return "done", false, nil
}

func (a *dialWatchingAgent) RespondStream(
	_ context.Context, _ string, _ func(string) error, _ func(string) error, _ TurnOptions,
) (string, bool, error) {
	return "", true, nil
}

func (a *dialWatchingAgent) Respond(_ context.Context, _ string, _ TurnOptions) (string, error) {
	return "", nil
}

func (a *dialWatchingAgent) RespondEphemeral(_ context.Context, _ string, _ TurnOptions) (string, error) {
	return "", nil
}

// The depth dial is read on every round the loop builds, not copied once when
// the turn opens. A press under a running turn used to park a whole engine
// rebuild for the boundary — over a request field the loop rewrites anyway.
func TestThinkLevelPressedMidTurnReachesTheNextRound(t *testing.T) {
	agent := &dialWatchingAgent{}
	exec := NewExecutor(ExecutorOptions{
		Agent:       agent,
		Dispatcher:  oneToolDispatcher{},
		TurnOptions: TurnOptions{ThinkLevel: think.LevelLow},
	})
	agent.press = func() { exec.SetThinkLevel(think.LevelHigh) }

	if _, err := exec.Execute(
		context.Background(), "คิดให้หนักกว่านี้",
		command.Intent{Raw: "คิดให้หนักกว่านี้", Kind: command.KindConversation},
		func(string) {}, nil, nil,
	); err != nil {
		t.Fatalf("Execute() = %v", err)
	}
	if len(agent.seen) != 2 || agent.seen[0] != think.LevelLow || agent.seen[1] != think.LevelHigh {
		t.Fatalf("levels seen per round = %v, want [low high]: the press must reach the round after it, not the next turn", agent.seen)
	}

	// And the turn after it opens on the pressed level, with no press needed.
	agent.seen = nil
	agent.press = func() {}
	if _, err := exec.Execute(
		context.Background(), "ต่อ",
		command.Intent{Raw: "ต่อ", Kind: command.KindConversation},
		func(string) {}, nil, nil,
	); err != nil {
		t.Fatalf("second Execute() = %v", err)
	}
	if len(agent.seen) != 2 || agent.seen[0] != think.LevelHigh {
		t.Fatalf("next turn opened on %v, want the pressed level to stand", agent.seen)
	}
}

// Nobody has pressed the dial: EffectiveThinkLevel is the plain field, so a
// caller that builds TurnOptions by hand (delegates, the CLI) is unchanged.
func TestEffectiveThinkLevelFallsBackToTheField(t *testing.T) {
	if got := (TurnOptions{ThinkLevel: think.LevelNoThinking}).EffectiveThinkLevel(); got != think.LevelNoThinking {
		t.Fatalf("EffectiveThinkLevel() = %q, want the field when no dial is installed", got)
	}
}
