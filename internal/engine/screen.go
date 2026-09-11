package engine

// The engine's view of the window (§248 A6, B1).
//
// Everything the engine ever needs from the screen fits in a handful of calls:
// send an event to whoever is watching, sign a provider request with a
// credential the engine does not hold, lend the tools that only make sense
// where a window is — the browser tab, the mouse — and answer the few
// questions about a provider that need its key: which model is its default,
// does a ping reach it, is a local model resident yet. Today the screen is
// the desktop's App in the same process (desktop/screen.go) and the calls are
// method calls; when the engine is on another machine the same calls cross a
// socket, and nothing here changes.
//
// The engine holds no key and asks for none: the desk questions are answered
// by the screen with its key, and only the answer comes back.

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// Screen is what the engine asks of the window.
type Screen interface {
	// Emit sends one event to the window — the same names emitEvent carries,
	// with the payload Wails would marshal.
	Emit(event string, data any)
	// ProviderTransport is the credential for a provider in a wire format,
	// as the model layer's Transport seam (desktop/provider_forward.go).
	ProviderTransport(provider, wireFormat string) model.Transport
	// ProviderEndpoint is the base URL a sign-in pinned for a provider, or ""
	// for the catalog's. Not a secret, but the sign-in is the screen's store.
	ProviderEndpoint(provider string) string
	// WindowTools are the packs that act on the window's machine — browser
	// and computer — built for one session. The engine decides which of them
	// a session actually gets (workbenchSkills); the screen only offers.
	WindowTools(s Session) []skill.Skill

	// DefaultModel is the model a provider answers with when asked, on this
	// endpoint, with the screen's key — "" when it cannot say. The engine
	// asks when a switch lands on a provider with no model remembered.
	DefaultModel(provider, baseURL string) string
	// Probe is one 1-token completion through the same client chat uses, so
	// endpoint, key and wire format are proven at once; the label on success,
	// the provider's real failure otherwise. The queued-switch preflight.
	Probe(provider, modelName, baseURL, wireFormat string) (string, error)
	// ModelResident says whether a local runtime has the named model in
	// memory — the end of the wait the loading notice covers.
	ModelResident(provider, baseURL, modelName string) bool
}

// Session is the little a window tool needs to know about the chat it acts
// for: which one, so its desk events land there, and where its files are.
type Session interface {
	ID() string
	Root() string
}

func (c *conversation) ID() string   { return c.id }
func (c *conversation) Root() string { return strings.TrimSpace(c.cfg.SandboxRoot) }

// noScreen is the engine with nobody watching: an Engine built by struct
// literal in a test, or — in phase 2 — a process whose screen has not
// connected yet. Events go nowhere, requests go out unsigned, no window tools
// are lent, and every desk question is answered with "cannot say". Nothing
// here fails; a screen arriving later fills every blank.
type noScreen struct{}

func (noScreen) Emit(string, any) {}
func (noScreen) ProviderTransport(string, string) model.Transport {
	return func(network http.RoundTripper) http.RoundTripper { return network }
}
func (noScreen) ProviderEndpoint(string) string                       { return "" }
func (noScreen) WindowTools(Session) []skill.Skill                    { return nil }
func (noScreen) DefaultModel(string, string) string                   { return "" }
func (noScreen) Probe(string, string, string, string) (string, error) { return "", errNoScreen }
func (noScreen) ModelResident(string, string, string) bool            { return false }

// screenOf is the Screen this engine talks to, never nil.
func (a *Engine) screenOf() Screen {
	if a.screen != nil {
		return a.screen
	}
	return noScreen{}
}

// errNoScreen is what a desk question earns when nobody is watching: the
// engine cannot prove a provider it holds no key for.
var errNoScreen = errors.New("no screen is connected to answer for the provider")
