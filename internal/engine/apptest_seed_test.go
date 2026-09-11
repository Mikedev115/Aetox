package engine

import (
	"context"
	"net/http"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// seed puts a conversation on the Engine's screen, for tests that used to build
// one by setting Engine fields directly.
//
// Those fields moved into `conversation` on 2026-08-19 (desktop/conversation.go)
// because none of them was ever a property of the app. The tests that set them
// were saying "an app whose open chat is this" all along; this says it in the
// words the code now uses, and nothing about what they assert changed.
//
// Since §248 B1 the window is another package, and an engine test has no
// window: seed installs testScreen unless the test brought its own, so what
// the desktop's screen lends and answers is lent and answered here too.
func seed(a *Engine, conv *conversation) *Engine {
	// The chat inherits the app's config unless the test gave it one of its own.
	// That is what production does — a conversation is built from the template
	// in Engine.cfg (applyConfig) — and it keeps every test that sets `cfg:` on the
	// Engine literal saying what it has always said, now that the readers ask the
	// conversation rather than the app (DECISIONS §155).
	if conv.cfg.SandboxRoot == "" && conv.cfg.ModelProvider == "" {
		conv.cfg = a.cfg
	}
	if a.screen == nil {
		a.screen = testScreen{a}
	}
	a.convs = newConversations()
	a.convs.show(conv)
	return a
}

// testScreen is the window an engine test runs against: no packs (the
// browser and the machine are the window's, in desktop/), no events (the
// a.emit seam is the recorder), and the desk questions answered the way the
// screen answers them — without a key, which is all a test ever has.
type testScreen struct{ e *Engine }

// Emit is nothing: an engine test has no window to reach, and a test that
// wants the events installs the a.emit seam, which emitEvent tries first.
func (s testScreen) Emit(string, any) {}
func (s testScreen) ProviderTransport(string, string) model.Transport {
	return func(n http.RoundTripper) http.RoundTripper { return n }
}
func (s testScreen) ProviderEndpoint(string) string    { return "" }
func (s testScreen) WindowTools(Session) []skill.Skill { return nil }
func (s testScreen) AgentTab() string                  { return "" }
func (s testScreen) RenderDeck(context.Context, string, DeckRender) (DeckRendered, error) {
	return DeckRendered{}, errNoScreen
}
func (s testScreen) DefaultModel(provider, baseURL string) string {
	return model.ResolveDefaultModel(provider, baseURL, "")
}
func (s testScreen) ModelResident(provider, baseURL, name string) bool {
	return model.LocalModelResident(provider, baseURL, "", name)
}

// Probe is the real ping, keyless: what the tests reach for is an endpoint
// that refuses, which needs no key to be found out.
func (s testScreen) Probe(provider, modelName, baseURL, wireFormat string) (string, error) {
	p, err := model.NewProvider(model.ProviderOptions{
		Provider: provider, Model: modelName, BaseURL: baseURL, Timeout: 5 * time.Second, WireFormat: wireFormat,
		Transport: func(n http.RoundTripper) http.RoundTripper { return n },
	})
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := p.Complete(ctx, model.Request{Model: modelName, Messages: []model.Message{{Role: model.RoleUser, Content: "ping"}}, MaxTokens: 1}); err != nil {
		return "", err
	}
	return modelName, nil
}
