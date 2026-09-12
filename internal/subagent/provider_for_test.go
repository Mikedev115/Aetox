package subagent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/safety"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// A profile may name the provider it thinks with (owner, 12 ก.ย. 2026: "ควร
// เลือกได้แม้แต่ผู้ให้บริการ และเลือกโมเดลได้ ทั้งเอเจนและซับเอเจน"). The delegate then
// runs on a client the host builds for that provider — on the model the
// profile names, else that provider's default — and never on the session's.
// A provider the host cannot build is a failed tool call the model can read.

// recordingProvider is the §45 kind of fake: a provider edge case (which
// client answered, and for which model), everything else on the path real.
type recordingProvider struct {
	name string
	mu   sync.Mutex
	got  []string
}

func (p *recordingProvider) Name() string              { return p.name }
func (p *recordingProvider) SupportsToolCalling() bool { return true }
func (p *recordingProvider) Complete(_ context.Context, req model.Request) (model.Response, error) {
	p.mu.Lock()
	p.got = append(p.got, req.Model)
	p.mu.Unlock()
	return model.Response{Text: "done"}, nil
}
func (p *recordingProvider) models() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.got...)
}

func runExploreOn(t *testing.T, opts TaskOptions) skill.Output {
	t.Helper()
	registry := skill.NewDefaultRegistry(skill.RegistryOptions{SandboxRoot: t.TempDir()})
	opts.Registry = registry
	opts.ApprovalMode = safety.ApprovalFullAccess
	for _, tool := range NewTaskTools(opts) {
		if err := registry.Register(tool, skill.SourceBuiltin); err != nil {
			t.Fatalf("register %s: %v", tool.Name(), err)
		}
	}
	ctx := turn.WithCallID(context.Background(), "call_1")
	dispatcher := skill.NewDispatcher(registry)
	start, _, err := dispatcher.ExecuteTool(ctx, "task", map[string]any{
		"description": "x", "prompt": "report the files", "agent": "explore",
	})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !start.Success {
		return start
	}
	out, _, err := dispatcher.ExecuteTool(ctx, "task", map[string]any{"action": "collect", "task_id": taskIDOf(t, start)})
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	return out
}

func TestADelegateThinksOnTheProviderItsProfileNames(t *testing.T) {
	isolate(t)
	if err := Save("explore", "---\ndescription: ค้นไฟล์\nprovider: other\n---\nFind things.\n"); err != nil {
		t.Fatalf("shadow: %v", err)
	}
	session := &recordingProvider{name: "session"}
	other := &recordingProvider{name: "other"}
	var asked []string
	out := runExploreOn(t, TaskOptions{
		Provider: session, Model: "session-model",
		ProviderFor: func(name string) (model.Provider, string, error) {
			asked = append(asked, name)
			return other, "other-default", nil
		},
	})
	if !out.Success {
		t.Fatalf("task failed: %q", out.Content)
	}
	if strings.Join(asked, ",") != "other" {
		t.Fatalf("the host was asked for %v, want the profile's provider once", asked)
	}
	if got := other.models(); len(got) == 0 || got[0] != "other-default" {
		t.Fatalf("the delegate ran on %v, want the named provider's default model", got)
	}
	if got := session.models(); len(got) != 0 {
		t.Fatalf("the session's provider was also used: %v", got)
	}
}

func TestADelegateNamesBothProviderAndModel(t *testing.T) {
	isolate(t)
	if err := Save("explore", "---\ndescription: ค้นไฟล์\nprovider: other\nmodel: small-fast\n---\nFind things.\n"); err != nil {
		t.Fatalf("shadow: %v", err)
	}
	other := &recordingProvider{name: "other"}
	out := runExploreOn(t, TaskOptions{
		Provider: &recordingProvider{name: "session"}, Model: "session-model",
		ProviderFor: func(string) (model.Provider, string, error) { return other, "other-default", nil },
	})
	if !out.Success {
		t.Fatalf("task failed: %q", out.Content)
	}
	if got := other.models(); len(got) == 0 || got[0] != "small-fast" {
		t.Fatalf("the delegate ran on %v, want the profile's own model", got)
	}
}

func TestAProviderTheHostCannotBuildFailsTheCallOutLoud(t *testing.T) {
	isolate(t)
	if err := Save("explore", "---\ndescription: ค้นไฟล์\nprovider: nowhere\n---\nFind things.\n"); err != nil {
		t.Fatalf("shadow: %v", err)
	}
	session := &recordingProvider{name: "session"}
	out := runExploreOn(t, TaskOptions{
		Provider: session, Model: "session-model",
		ProviderFor: func(string) (model.Provider, string, error) { return nil, "", errors.New("no key for nowhere") },
	})
	if out.Success {
		t.Fatal("a provider that could not be built still ran the delegate")
	}
	if !strings.Contains(out.Content, "nowhere") || !strings.Contains(out.Content, "no key") {
		t.Fatalf("the failure does not say which provider or why: %q", out.Content)
	}
	if got := session.models(); len(got) != 0 {
		t.Fatalf("the delegate silently ran on the session's provider: %v", got)
	}
}

// A host with no factory (the CLI, every older test) keeps the session's
// provider: the line is noted, nothing fails.
func TestWithoutAFactoryTheProviderLineIsIgnored(t *testing.T) {
	isolate(t)
	if err := Save("explore", "---\ndescription: ค้นไฟล์\nprovider: other\n---\nFind things.\n"); err != nil {
		t.Fatalf("shadow: %v", err)
	}
	session := &recordingProvider{name: "session"}
	out := runExploreOn(t, TaskOptions{Provider: session, Model: "session-model"})
	if !out.Success {
		t.Fatalf("task failed: %q", out.Content)
	}
	if got := session.models(); len(got) == 0 || got[0] != "session-model" {
		t.Fatalf("the delegate ran on %v, want the session's model", got)
	}
}
