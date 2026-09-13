package engine

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
)

// SupportedThinkLevelsFor answers for the provider/model an agent's editor is
// pointing at, so a level saved on an agent is one the dispatch will honour.
//
// The empty/empty case is the one that shipped broken for a minute: it means
// "whatever the chat is on", and provider.Normalize("") answers with the
// fallback provider's name rather than "", so normalizing before the test made
// the inherit branch unreachable — the editor drew "this model has no thinking
// level" over a chat sitting at max (caught in the dev app, 13 ก.ย. 2026).
func TestSupportedThinkLevelsForInheritsTheChatsProviderAndModel(t *testing.T) {
	t.Cleanup(func() { model.SetModelCatalog(nil) })
	model.SetModelCatalog(&model.ModelCatalog{Source: "test", Models: map[string]model.ModelFacts{
		"deepseek/deepseek-v4": {
			Context: 128000, ToolCall: true, Reasoning: true, ReasoningToggle: true,
			ReasoningLevels: []string{"low", "high", "max"},
			Input:           []string{"text"}, Output: []string{"text"},
		},
	}})

	a := &Engine{}
	a.cur().cfg = config.Config{ModelProvider: "deepseek", ModelName: "deepseek-v4"}

	inherited := a.SupportedThinkLevelsFor("", "")
	if len(inherited) == 0 {
		t.Fatal("inheriting both answered no levels; the chat's own model has them")
	}
	named := a.SupportedThinkLevelsFor("deepseek", "deepseek-v4")
	if len(named) != len(inherited) {
		t.Fatalf("inheriting gave %v, naming the same pair gave %v", inherited, named)
	}

	// A model with no dial answers an empty list, never nil: the frontend reads
	// .length on it mid-render.
	if levels := a.SupportedThinkLevelsFor("ollama", "llama-nobody-has"); levels == nil {
		t.Fatal("a model with no dial answered nil; JSON null crashes the picker")
	}
}
