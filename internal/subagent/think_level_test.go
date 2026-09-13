package subagent

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/think"
)

// A profile may name how deep it thinks (owner, 13 ก.ย. 2026: "เอเจนและ
// ซับเอเจน ทำให้เราปรับระดับความคิดได้"). Until then every delegate thought at
// the chat's level — whatever the picker happened to be on when the job was
// dispatched — which made a file-search helper think at ultra on a chat that
// was planning, and a planner think at low on a chat that was skimming.

// effortProvider is the §45 fake with one more edge: it says it reasons, so
// the effort the delegate asked for actually reaches the request.
type effortProvider struct {
	recordingProvider
	efforts []string
	mu      sync.Mutex
}

func (p *effortProvider) SupportsReasoning() bool { return true }
func (p *effortProvider) Complete(ctx context.Context, req model.Request) (model.Response, error) {
	p.mu.Lock()
	effort := ""
	if req.Reasoning != nil {
		effort = req.Reasoning.Effort
	}
	p.efforts = append(p.efforts, effort)
	p.mu.Unlock()
	return p.recordingProvider.Complete(ctx, req)
}
func (p *effortProvider) effort() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return strings.Join(p.efforts, ",")
}

func TestADelegateThinksAtTheLevelItsProfileNames(t *testing.T) {
	isolate(t)
	if err := Save("explore", "---\ndescription: ค้นไฟล์\nthink: low\n---\nFind things.\n"); err != nil {
		t.Fatalf("shadow: %v", err)
	}
	session := &effortProvider{recordingProvider: recordingProvider{name: "session"}}
	out := runExploreOn(t, TaskOptions{
		Provider: session, Model: "session-model",
		ThinkLevel: think.LevelHigh, // the chat is thinking hard; the helper should not
	})
	if !out.Success {
		t.Fatalf("task failed: %q", out.Content)
	}
	if got := session.effort(); got != "low" {
		t.Fatalf("the delegate asked for effort %q, want the profile's own low", got)
	}
}

func TestADelegateWithNoThinkLineInheritsTheChats(t *testing.T) {
	isolate(t)
	if err := Save("explore", "---\ndescription: ค้นไฟล์\n---\nFind things.\n"); err != nil {
		t.Fatalf("shadow: %v", err)
	}
	session := &effortProvider{recordingProvider: recordingProvider{name: "session"}}
	out := runExploreOn(t, TaskOptions{Provider: session, Model: "session-model", ThinkLevel: think.LevelHigh})
	if !out.Success {
		t.Fatalf("task failed: %q", out.Content)
	}
	if got := session.effort(); got != "high" {
		t.Fatalf("the delegate asked for effort %q, want the chat's high", got)
	}
}

// The level is checked against the delegate's OWN provider and model — not the
// chat's. Codex has ultra; a DeepSeek helper does not, and a level the model
// lacks folds to that model's default rather than going out as a word the
// endpoint rejects.
func TestAThinkLevelTheDelegatesModelLacksFoldsToItsDefault(t *testing.T) {
	isolate(t)
	// The dial is read from the installed catalog, which a unit test does not
	// have; install the one row this test is about. Nothing to restore: a unit
	// test process starts with none installed.
	t.Cleanup(func() { model.SetModelCatalog(nil) })
	model.SetModelCatalog(&model.ModelCatalog{Source: "test", Models: map[string]model.ModelFacts{
		"deepseek/deepseek-v4": {
			Context: 128000, ToolCall: true, Reasoning: true, ReasoningToggle: true,
			ReasoningLevels: []string{"low", "high", "max"},
			Input:           []string{"text"}, Output: []string{"text"},
		},
	}})
	if err := Save("explore", "---\ndescription: ค้นไฟล์\nprovider: deepseek\nmodel: deepseek-v4\nthink: ultra\n---\nFind things.\n"); err != nil {
		t.Fatalf("shadow: %v", err)
	}
	other := &effortProvider{recordingProvider: recordingProvider{name: "deepseek"}}
	out := runExploreOn(t, TaskOptions{
		Provider: &recordingProvider{name: "session"}, Model: "session-model",
		ProviderFor: func(string) (model.Provider, string, error) { return other, "deepseek-v4", nil },
	})
	if !out.Success {
		t.Fatalf("task failed: %q", out.Content)
	}
	got := other.effort()
	if got == "" || got == "ultra" {
		t.Fatalf("the delegate asked for effort %q, want deepseek-v4's own default for a level it lacks", got)
	}
	if !model.SupportsThinkingLevel("deepseek", "deepseek-v4", got) {
		t.Fatalf("effort %q is not one deepseek-v4 has: %v", got, model.SupportedThinkingLevels("deepseek", "deepseek-v4"))
	}
}

func TestThinkIsReadFromTheFrontmatterLowercased(t *testing.T) {
	p := parse("x", "---\ndescription: d\nthink: XHigh\n---\nbody\n")
	if p.Think != "xhigh" {
		t.Fatalf("Think = %q, want xhigh", p.Think)
	}
	if p := parse("y", "---\ndescription: d\n---\nbody\n"); p.Think != "" {
		t.Fatalf("a file with no think: line reads %q, want empty (inherit)", p.Think)
	}
}
