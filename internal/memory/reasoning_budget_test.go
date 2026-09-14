package memory

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
)

// Replayed thinking is sent on every later request, so it has to be counted
// against the budget that decides when to trim. It was not, for the few hours
// between the replay landing and this test: a browser session spends thirty-odd
// tool rounds and one measured block is 1,420 characters, so the budget was
// blind to roughly ten thousand tokens the provider was charging for — and a
// budget that reads low stops trimming exactly when it should not.
func TestReplayedThinkingIsCountedAgainstTheBudget(t *testing.T) {
	block := json.RawMessage(`{"type":"reasoning","id":"rs_x","encrypted_content":"` +
		strings.Repeat("A", 1400) + `"}`)

	plain := []model.Message{{Role: model.RoleAssistant, Content: "done"}}
	withThinking := []model.Message{{
		Role:           model.RoleAssistant,
		Content:        "done",
		ReasoningItems: []model.ReasoningItem{{Provider: "codex", Raw: block}},
	}}

	bare, carried := totalChars(plain), totalChars(withThinking)
	if carried <= bare {
		t.Fatalf("a message carrying %d characters of thinking measured %d, same as one carrying none (%d)",
			len(block), carried, bare)
	}
	if grew := carried - bare; grew != len(block) {
		t.Fatalf("the block is %d characters and the budget grew by %d", len(block), grew)
	}
}
