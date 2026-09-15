package cognitive

import (
	"context"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
)

type compactMockProvider struct {
	model.Provider
	summaryResponse string
}

func (p *compactMockProvider) Complete(_ context.Context, _ model.Request) (model.Response, error) {
	return model.Response{Text: p.summaryResponse}, nil
}

func (p *compactMockProvider) Name() string {
	return "mock-compact"
}

func TestCompactContextOnDemand(t *testing.T) {
	provider := &compactMockProvider{
		summaryResponse: "Summary of earlier research into files and architecture.",
	}
	agent := NewAgent(AgentConfig{
		Provider: provider,
		Model:    "test-model",
		MaxChars: 100_000,
	})

	// Add messages: 1 user, 1 assistant with tool call, 1 tool result (>400 chars),
	// followed by several turns to exceed compactKeepRecent (6) + 4.
	toolContent := strings.Repeat("read output content\n", 30) // ~600 chars
	messages := []model.Message{
		{Role: model.RoleUser, Content: "Hello, check file A"},
		{Role: model.RoleAssistant, Content: "Checking file A", ToolCalls: []model.ToolCall{
			{ID: "call_1", Type: "function", Function: model.FunctionCall{Name: "read"}},
		}},
		{Role: model.RoleTool, Name: "read", ToolCallID: "call_1", Content: toolContent},
		{Role: model.RoleAssistant, Content: "File A looks good."},
	}

	for i := 2; i <= 6; i++ {
		messages = append(messages,
			model.Message{Role: model.RoleUser, Content: "Now check next item"},
			model.Message{Role: model.RoleAssistant, Content: "All items verified."},
		)
	}

	agent.RestoreHistory(messages)

	swept, summarized, err := agent.CompactContext(context.Background())
	if err != nil {
		t.Fatalf("unexpected error during CompactContext: %v", err)
	}
	if swept == 0 {
		t.Errorf("expected old tool result to be swept, got swept = 0")
	}
	if !summarized {
		t.Errorf("expected older conversation turns to be summarized, got summarized = false")
	}

	items, _, summaries := agent.MaintenanceStats()
	if items != swept {
		t.Errorf("MaintenanceStats swept items mismatch: got %d, expected %d", items, swept)
	}
	if summaries == 0 {
		t.Errorf("MaintenanceStats summaries should be > 0, got %d", summaries)
	}
}
