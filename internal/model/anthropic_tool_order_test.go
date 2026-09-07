package model

import "testing"

// The dialect's own ordering rule, held here as well as at the source.
//
// Anthropic requires every tool_result answering an assistant turn to sit at
// the FRONT of the reply turn. Aetox carries a tool's picture as an ordinary
// user message — there is no image block a tool_result can hold on all three
// dialects — so anything that puts one between two results is a 400 naming a
// message index and nothing else. cognitive/agent.go no longer produces that
// shape; this is what stops the next thing that does from costing a turn.
func TestAnthropicPutsEveryToolResultBeforeTheRestOfTheTurn(t *testing.T) {
	_, out := convertMessagesToAnthropic([]Message{
		{Role: RoleUser, Content: "เปิดหน้าเว็บแล้วดูทั้งภาพและ console"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{
			{ID: "call_capture", Function: FunctionCall{Name: "browser", Arguments: `{"action":"capture"}`}},
			{ID: "call_console", Function: FunctionCall{Name: "browser", Arguments: `{"action":"console"}`}},
		}},
		{Role: RoleTool, ToolCallID: "call_capture", Content: "captured"},
		{Role: RoleUser, Content: "[the image returned by browser follows]",
			Images: []Image{{MediaType: "image/png", Data: []byte("\x89PNG fake")}}, ImagesFromTool: true},
		{Role: RoleTool, ToolCallID: "call_console", Content: "no console errors"},
	})

	if len(out) != 3 {
		t.Fatalf("expected user, assistant, user; got %d turns", len(out))
	}
	reply := out[2]
	if reply.Role != "user" {
		t.Fatalf("the reply turn is %q, not user", reply.Role)
	}

	var types []string
	for _, b := range reply.Content {
		types = append(types, b.Type)
	}
	if len(types) < 2 || types[0] != "tool_result" || types[1] != "tool_result" {
		t.Fatalf("blocks are %v; both results must come first", types)
	}
	// Order among the results is the model's own, and must survive the move.
	if reply.Content[0].ToolUseID != "call_capture" || reply.Content[1].ToolUseID != "call_console" {
		t.Errorf("results were reordered: %q then %q",
			reply.Content[0].ToolUseID, reply.Content[1].ToolUseID)
	}
	// And nothing is dropped on the way: the picture still follows.
	var sawImage bool
	for _, b := range reply.Content {
		if b.Type == "image" {
			sawImage = true
		}
	}
	if !sawImage {
		t.Errorf("the picture was lost; blocks are %v", types)
	}
}

// A turn that was already in order is handed back untouched — the common case,
// and the one that must not change shape.
func TestAnthropicLeavesAnOrderedTurnAlone(t *testing.T) {
	_, out := convertMessagesToAnthropic([]Message{
		{Role: RoleAssistant, ToolCalls: []ToolCall{
			{ID: "call_read", Function: FunctionCall{Name: "read", Arguments: `{}`}},
		}},
		{Role: RoleTool, ToolCallID: "call_read", Content: "alpha"},
		{Role: RoleUser, Content: "แล้วไงต่อ"},
	})
	if len(out) != 2 {
		t.Fatalf("expected assistant, user; got %d turns", len(out))
	}
	var types []string
	for _, b := range out[1].Content {
		types = append(types, b.Type)
	}
	if len(types) != 2 || types[0] != "tool_result" || types[1] != "text" {
		t.Fatalf("blocks are %v, want [tool_result text]", types)
	}
}
