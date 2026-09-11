package engine

import (
	"context"
	"net/http"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// A stand-in window: lends whatever tools it is given, signs nothing, and
// records what it was asked to show.
type fakeScreen struct {
	tools  []skill.Skill
	events []string
}

func (f *fakeScreen) Emit(event string, _ any) { f.events = append(f.events, event) }
func (f *fakeScreen) ProviderTransport(string, string) model.Transport {
	return func(n http.RoundTripper) http.RoundTripper { return n }
}
func (f *fakeScreen) ProviderEndpoint(string) string                       { return "" }
func (f *fakeScreen) WindowTools(Session) []skill.Skill                    { return f.tools }
func (f *fakeScreen) DefaultModel(string, string) string                   { return "" }
func (f *fakeScreen) Probe(string, string, string, string) (string, error) { return "", errNoScreen }
func (f *fakeScreen) ModelResident(string, string, string) bool            { return false }
func (f *fakeScreen) AgentTab() string                                     { return "" }
func (f *fakeScreen) RenderDeck(context.Context, string, DeckRender) (DeckRendered, error) {
	return DeckRendered{}, errNoScreen
}

// switchOnComputer turns the computer-control switch on for the duration of a
// test.
//
// It writes the preference file directly rather than calling the binding,
// because the binding re-applies the whole config to rebuild the engine (the
// switch decides whether the tool is registered at all) and the tests here
// build their registry by hand. The data root is a temp dir by then, so the
// file written here is thrown away with the test and is never the one
// belonging to whoever is running it.
func switchOnComputer(t *testing.T) {
	t.Helper()
	if err := config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.ComputerControlOn = true
		return nil
	}); err != nil {
		t.Fatalf("could not turn computer control on for the test: %v", err)
	}
}

// The engine needs nothing more of a window than the three calls on Screen
// (§248 A6): with another Screen installed, the browser and the machine come
// from it — and the engine's own switch still decides whether the machine is
// handed to the model at all.
func TestTheEngineTakesItsWindowToolsFromTheScreen(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	// stubTool is desk_test.go's: a name and nothing else.
	window := &fakeScreen{tools: []skill.Skill{&stubTool{name: "browser"}, &stubTool{name: computerToolName}}}
	a := seed(&Engine{cfg: config.Config{SandboxRoot: t.TempDir()}, screen: window}, newConversation())

	names := map[string]bool{}
	for _, s := range a.workbenchSkills(a.cur(), t.TempDir()) {
		names[s.Name()] = true
	}
	if !names["browser"] {
		t.Error("the browser the screen lent is not among the session's tools")
	}
	if names[computerToolName] {
		t.Error("the machine reached the model with the switch off — the switch is the engine's, whoever lends the tool")
	}

	switchOnComputer(t)
	names = map[string]bool{}
	for _, s := range a.workbenchSkills(a.cur(), t.TempDir()) {
		names[s.Name()] = true
	}
	if !names[computerToolName] {
		t.Error("with the switch on, the machine the screen lent is missing")
	}
}
