package cognitive

import (
	"context"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/think"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// A picture from one call must not land between two results of the same round.
//
// Anthropic requires every tool_result answering an assistant turn to sit at
// the front of the reply, and a picture is carried as an ordinary user message
// (there is no image block a tool_result can hold on all three dialects). Added
// beside its own call, it split the group: measured 2026-09-07, a round of
// `browser capture` then `browser console` on deepseek's default (Anthropic)
// wire format died on `messages.52: tool_use ids were found without
// tool_result blocks immediately after` and took 187 seconds of finished work
// with it.
func TestToolLoopKeepsPicturesBehindEveryResultOfTheRound(t *testing.T) {
	provider := &toolLoopProvider{
		responses: []model.Response{
			{ToolCalls: []model.ToolCall{
				{ID: "call_capture", Type: "function", Function: model.FunctionCall{
					Name: "browser", Arguments: `{"action":"capture"}`}},
				{ID: "call_console", Type: "function", Function: model.FunctionCall{
					Name: "browser", Arguments: `{"action":"console"}`}},
			}},
			{Text: "ทั้งสองอย่างเรียบร้อย"},
		},
	}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxToolCalls: 4})

	_, _, err := agent.RespondWithTools(
		context.Background(),
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{
			Name: "browser", Parameters: []byte(`{"type":"object"}`)}}},
		"เปิดหน้าเว็บแล้วดูทั้งภาพและ console",
		func(_ context.Context, call model.ToolCall) (string, []model.Image, error) {
			// Only the first call brings a picture back, which is the shape
			// that breaks: a picture on the LAST call of a round was always
			// fine, and that is why this went unseen for so long.
			if call.ID == "call_capture" {
				return "captured", []model.Image{{MediaType: "image/png", Data: []byte("\x89PNG fake")}}, nil
			}
			return "no console errors", nil, nil
		},
		nil,
		turn.TurnOptions{ThinkLevel: think.LevelMedium},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(provider.requests) < 2 {
		t.Fatalf("expected a second round, saw %d requests", len(provider.requests))
	}

	// Read the transcript the model was handed the second time.
	var order []string
	for _, msg := range provider.requests[1].Messages {
		switch {
		case msg.Role == model.RoleTool:
			order = append(order, "result:"+msg.ToolCallID)
		case len(msg.Images) > 0:
			order = append(order, "picture")
		}
	}
	want := []string{"result:call_capture", "result:call_console", "picture"}
	if len(order) != len(want) {
		t.Fatalf("transcript is %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("transcript is %v, want %v", order, want)
		}
	}
}
