package cognitive

import (
	"context"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/think"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// The usage reporter must fire once per API response — including every round
// of a tool loop, not just the final one — with the provider's real numbers.
// This is the contract the desktop's Usage stats page depends on.
func TestUsageReporterFiresPerToolLoopRound(t *testing.T) {
	provider := &toolLoopProvider{
		responses: []model.Response{
			{
				ToolCalls: []model.ToolCall{{
					ID: "call_1", Type: "function",
					Function: model.FunctionCall{Name: "read", Arguments: `{"path":"a.txt"}`},
				}},
				Usage: &model.Usage{PromptTokens: 11, CompletionTokens: 3},
			},
			{
				Text:  "done",
				Usage: &model.Usage{PromptTokens: 29, CompletionTokens: 7},
			},
		},
	}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxToolCalls: 4})

	var got []model.Usage
	agent.SetUsageReporter(func(u model.Usage) { got = append(got, u) })

	_, _, err := agent.RespondWithTools(
		context.Background(),
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		"read a.txt",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "ok", nil, nil },
		nil,
		turn.TurnOptions{ThinkLevel: think.LevelMedium},
	)
	if err != nil {
		t.Fatalf("RespondWithTools: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("reporter fired %d times, want 2 (one per API round): %+v", len(got), got)
	}
	if got[0].PromptTokens != 11 || got[0].CompletionTokens != 3 ||
		got[1].PromptTokens != 29 || got[1].CompletionTokens != 7 {
		t.Fatalf("reporter received wrong numbers: %+v", got)
	}
}

// Respond (the non-tool path) reports too.
func TestUsageReporterFiresOnPlainRespond(t *testing.T) {
	provider := &toolLoopProvider{
		responses: []model.Response{{Text: "hi", Usage: &model.Usage{PromptTokens: 5, CompletionTokens: 2}}},
	}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model"})

	var got []model.Usage
	agent.SetUsageReporter(func(u model.Usage) { got = append(got, u) })

	if _, err := agent.Respond(context.Background(), "hello", turn.TurnOptions{ThinkLevel: think.LevelLow}); err != nil {
		t.Fatalf("Respond: %v", err)
	}
	if len(got) != 1 || got[0].PromptTokens != 5 || got[0].CompletionTokens != 2 {
		t.Fatalf("reporter = %+v, want one report of 5/2", got)
	}
}

// Every round that sent the conversation carries the guess Aetox made for
// that request beside the provider's count — the pair engine.promptCalibration
// fits. A round that sent something else over the conversation (a title) is
// left unstamped, because its count describes a different request.
func TestUsageCarriesTheRequestEstimate(t *testing.T) {
	provider := &toolLoopProvider{
		responses: []model.Response{
			{
				ToolCalls: []model.ToolCall{{
					ID: "call_1", Type: "function",
					Function: model.FunctionCall{Name: "read", Arguments: `{"path":"a.txt"}`},
				}},
				Usage: &model.Usage{PromptTokens: 11, CompletionTokens: 3},
			},
			{Text: "done", Usage: &model.Usage{PromptTokens: 29, CompletionTokens: 7}},
			{Text: "a title", Usage: &model.Usage{PromptTokens: 31, CompletionTokens: 2}},
		},
	}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxToolCalls: 4, SystemPrompt: "you are a test system prompt"})

	var got []model.Usage
	agent.SetUsageReporter(func(u model.Usage) { got = append(got, u) })

	tools := []model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}}
	_, _, err := agent.RespondWithTools(
		context.Background(), tools, "read a.txt",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "ok", nil, nil },
		nil, turn.TurnOptions{ThinkLevel: think.LevelMedium},
	)
	if err != nil {
		t.Fatalf("RespondWithTools: %v", err)
	}
	if _, err := agent.RespondEphemeral(context.Background(), "title this", turn.TurnOptions{}); err != nil {
		t.Fatalf("RespondEphemeral: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("reporter fired %d times, want 3: %+v", len(got), got)
	}
	first, second, title := got[0].Estimate, got[1].Estimate, got[2].Estimate
	if first.IsZero() || first.System == 0 || first.Tools == 0 || first.Messages == 0 {
		t.Errorf("first round estimate = %+v; want system, tools and message parts all guessed", first)
	}
	// Same system prompt and tool block; the tool call and its result joined
	// the conversation in between.
	if second.System != first.System || second.Tools != first.Tools {
		t.Errorf("fixed part moved between rounds: %+v -> %+v", first, second)
	}
	if second.Messages <= first.Messages {
		t.Errorf("messages did not grow across the tool round: %d -> %d", first.Messages, second.Messages)
	}
	if !title.IsZero() {
		t.Errorf("ephemeral round stamped %+v; its count describes a request the meter never draws", title)
	}
}
