package cognitive

import (
	"context"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/turn"
)

func sawCompactionRequest(p *toolLoopProvider) bool {
	for _, req := range p.requests {
		if len(req.Messages) > 0 && strings.Contains(req.Messages[0].Content, "compacting a long conversation") {
			return true
		}
	}
	return false
}

// The chars say there is plenty of room; the provider, which tokenized the
// request, says it is at 85% of the window. The provider decides.
func TestCompactionTrustsTheProvidersCountOverTheChars(t *testing.T) {
	provider := &toolLoopProvider{responses: []model.Response{
		{ToolCalls: []model.ToolCall{{ID: "c1", Type: "function",
			Function: model.FunctionCall{Name: "read", Arguments: `{"path":"a.txt"}`}}},
			Usage: &model.Usage{PromptTokens: 85_000, CompletionTokens: 10}},
		{Text: "summary of earlier turns"},
		{Text: "done"},
	}}
	// MaxChars 400_000 is a 100k-token window at the budget's four chars a
	// token; the history here is a few hundred chars, nowhere near 80% of it.
	agent := NewAgent(AgentConfig{Provider: provider, Model: "m", MaxChars: 400_000})
	// Enough turns that there is something older than the kept-recent tail to
	// summarize; the sizes stay tiny, which is the point.
	seed := make([]model.Message, 0, 8)
	for i := 0; i < 4; i++ {
		seed = append(seed,
			model.Message{Role: model.RoleUser, Content: "earlier question"},
			model.Message{Role: model.RoleAssistant, Content: "earlier answer"},
		)
	}
	agent.RestoreHistory(seed)

	_, _, err := agent.RespondWithTools(
		context.Background(),
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		"read a.txt",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) { return "short", nil, nil },
		nil,
		turn.TurnOptions{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sawCompactionRequest(provider) {
		t.Fatal("the provider counted 85k of a 100k window and nothing was summarized — the chars overruled the tokenizer")
	}
}

// The mirror: the chars cross 80% of the budget, the provider says the request
// was a tenth of the window. Summarizing here would throw away history the
// model still has room for, and pay a model call to do it.
func TestCompactionDoesNotFireOnCharsWhenTheProviderSaysThereIsRoom(t *testing.T) {
	provider := &toolLoopProvider{responses: []model.Response{
		{ToolCalls: []model.ToolCall{{ID: "c1", Type: "function",
			Function: model.FunctionCall{Name: "read", Arguments: `{"path":"big.txt"}`}}},
			Usage: &model.Usage{PromptTokens: 120, CompletionTokens: 10}},
		{Text: "done", Usage: &model.Usage{PromptTokens: 700, CompletionTokens: 5}},
	}}
	// A 1,250-token window. The seed sits under the 80% line in chars; the
	// tool output then pushes the chars past it, which used to summarize.
	agent := NewAgent(AgentConfig{Provider: provider, Model: "m", MaxChars: 5000})
	seed := make([]model.Message, 0, 8)
	for i := 0; i < 4; i++ {
		seed = append(seed,
			model.Message{Role: model.RoleUser, Content: strings.Repeat("old question ", 20)},
			model.Message{Role: model.RoleAssistant, Content: strings.Repeat("old answer ", 20)},
		)
	}
	agent.RestoreHistory(seed)

	_, _, err := agent.RespondWithTools(
		context.Background(),
		[]model.ToolDefinition{{Type: "function", Function: model.ToolFunction{Name: "read", Parameters: []byte(`{"type":"object"}`)}}},
		"read big.txt",
		func(_ context.Context, _ model.ToolCall) (string, []model.Image, error) {
			return strings.Repeat("huge tool output ", 150), nil, nil
		},
		nil,
		turn.TurnOptions{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sawCompactionRequest(provider) {
		t.Fatal("the provider counted 120 of 1,250 tokens and the history was summarized anyway — the chars overruled the tokenizer")
	}
}

// Between measurements the fill moves with the chars, at the ratio the
// measurement itself implies, in both directions.
func TestWindowFillCarriesTheMeasurementForward(t *testing.T) {
	agent := NewAgent(AgentConfig{Provider: &toolLoopProvider{}, Model: "m", MaxChars: 40_000})
	if _, _, measured := agent.WindowFill(); measured {
		t.Fatal("nothing has been measured yet, and WindowFill claimed otherwise")
	}
	agent.RestoreHistory([]model.Message{{Role: model.RoleUser, Content: strings.Repeat("x", 3_000)}})
	_, chars, _ := agent.ContextUsage()
	// The provider says those chars were 1,000 tokens: three chars a token.
	agent.fill = windowFill{promptTokens: 1_000, chars: chars}

	used, window, measured := agent.WindowFill()
	if !measured || window != 10_000 || used != 1_000 {
		t.Fatalf("right after measuring: used=%d window=%d measured=%v, want 1000/10000/true", used, window, measured)
	}
	// The ratio is the measurement's own: the system prompt is in the chars
	// too, so it lands a little over three, and 300 chars come out a little
	// under 100 tokens — the high side, which is the safe side.
	agent.context.AddMessage(model.Message{Role: model.RoleTool, Content: strings.Repeat("y", 300)})
	if used, _, _ = agent.WindowFill(); used < 1_090 || used > 1_100 {
		t.Errorf("300 chars more at about three a token: used=%d, want 1090..1100", used)
	}
	agent.ClearContext()
	if _, _, measured = agent.WindowFill(); measured {
		t.Error("the history was cleared and the old measurement is still being trusted")
	}
}

// A summary's own request is not the conversation, and its usage must not be
// mistaken for a measurement of it: the summarizer is sent the old messages,
// which is nearly the whole history, so taking its count would report the
// conversation as full right after it was emptied.
func TestEphemeralRequestsDoNotMeasureTheWindow(t *testing.T) {
	provider := &toolLoopProvider{responses: []model.Response{
		{Text: "a summary", Usage: &model.Usage{PromptTokens: 9_000, CompletionTokens: 50}},
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "m", MaxChars: 40_000})
	if _, err := agent.RespondEphemeral(context.Background(), "summarize this", turn.TurnOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, _, measured := agent.WindowFill(); measured {
		t.Fatal("an ephemeral prompt's usage was taken as the conversation's fill")
	}
}
