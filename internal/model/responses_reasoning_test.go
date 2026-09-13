package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// A reasoning model on this endpoint is told to keep nothing (store:false), so
// the thinking it did comes back encrypted and has to ride back out with the
// next request. Not doing it was costing the owner's Codex sessions most of
// their prompt cache — 43% hit on gpt-5.6-luna deep in a session against 97%
// for DeepSeek on the same machine, measured 13 ก.ย. 2026.

const rawReasoning = `{"type":"reasoning","id":"rs_abc","encrypted_content":"gAAAAAB-secret","summary":[{"type":"summary_text","text":"**Thinking**"}],"content":[]}`

func assistantWithReasoning(provider, text string) Message {
	return Message{
		Role:           RoleAssistant,
		Content:        text,
		ReasoningItems: []ReasoningItem{{Provider: provider, Raw: json.RawMessage(rawReasoning)}},
	}
}

// The block goes back byte for byte. An `encrypted_content` re-encoded through
// a struct this package invented is no longer the thing the server issued, so
// the test is on the bytes and not on a decoded field.
func TestReasoningRidesBackVerbatimAndAheadOfTheTurnItBelongsTo(t *testing.T) {
	_, input := convertMessagesToResponses("codex", true, []Message{
		{Role: RoleUser, Content: "17*23?"},
		assistantWithReasoning("codex", "391"),
		{Role: RoleUser, Content: "and doubled?"},
	})

	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(body), `"gAAAAAB-secret"`) {
		t.Fatalf("the encrypted block did not survive to the wire: %s", body)
	}

	// Order is the order the model produced them: its thinking, then what it
	// said. The server matches the input against what it issued, so a block
	// after the message it belongs to is as wrong as no block at all.
	var items []map[string]any
	if err := json.Unmarshal(body, &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var kinds []string
	for _, it := range items {
		kind, _ := it["type"].(string)
		if role, ok := it["role"].(string); ok && role != "" {
			kind += ":" + role
		}
		kinds = append(kinds, kind)
	}
	got := strings.Join(kinds, ",")
	const want = "message:user,reasoning,message:assistant,message:user"
	if got != want {
		t.Fatalf("item order %q, want %q", got, want)
	}
}

// The safety property. A chat whose model was switched mid-way carries blocks
// another endpoint issued, and handing one of those to this one fails the whole
// request — so the filter is on the way out, not a hope that history is clean.
func TestAnotherProvidersReasoningIsNeverReplayed(t *testing.T) {
	_, input := convertMessagesToResponses("codex", true, []Message{
		{Role: RoleUser, Content: "hi"},
		assistantWithReasoning("anthropic", "hello"),
		{Role: RoleUser, Content: "again"},
	})
	body, _ := json.Marshal(input)
	if strings.Contains(string(body), "gAAAAAB-secret") {
		t.Fatalf("a block from another wire format was sent to this one: %s", body)
	}
}

// A `reasoning` item in a request that carries no `reasoning` field is an item
// the server never asked for. The turn that switches thinking off must not drag
// the previous turn's blocks along.
func TestNoReasoningAskedMeansNoReasoningReplayed(t *testing.T) {
	_, input := convertMessagesToResponses("codex", false, []Message{
		{Role: RoleUser, Content: "hi"},
		assistantWithReasoning("codex", "hello"),
		{Role: RoleUser, Content: "again"},
	})
	body, _ := json.Marshal(input)
	if strings.Contains(string(body), "gAAAAAB-secret") {
		t.Fatalf("thinking was replayed into a request that asked for none: %s", body)
	}
}

// buildResponsesRequest is the real door, and it decides `replayReasoning` from
// the effort it is about to send — the two must agree or the pairing above is
// decorative.
func TestTheRequestReplaysExactlyWhenItAsksToThink(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "hi"},
		assistantWithReasoning("codex", "hello"),
		{Role: RoleUser, Content: "again"},
	}
	for _, tc := range []struct {
		name      string
		reasoning *ReasoningConfig
	}{
		{"thinking on", &ReasoningConfig{Effort: "medium"}},
		{"thinking off", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := buildResponsesRequest("codex", "gpt-5.6-luna", Request{
				Model: "gpt-5.6-luna", Messages: msgs, Reasoning: tc.reasoning,
			})
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			body, _ := json.Marshal(req.Input)
			replayed := strings.Contains(string(body), "gAAAAAB-secret")
			asked := req.Reasoning != nil && req.Reasoning.Effort != ""
			if replayed != asked {
				t.Fatalf("asked to think = %v but replayed thinking = %v", asked, replayed)
			}
		})
	}
}

// A message carrying nothing must not grow an empty item, and the ordinary
// path — every provider that never sets the field — has to be untouched.
func TestAMessageWithNoReasoningIsUnchanged(t *testing.T) {
	before, input := convertMessagesToResponses("codex", true, []Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, Content: "hello"},
	})
	if len(input) != 2 {
		t.Fatalf("input grew to %d items on a history with no thinking in it", len(input))
	}
	_ = before
	if items := (Message{}).ReasoningItemsFor("codex"); items != nil {
		t.Fatalf("an empty message answered %v", items)
	}
}
