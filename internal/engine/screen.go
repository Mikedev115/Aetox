package engine

// The engine's view of the window (§248 A6).
//
// Everything the engine ever needs from the screen fits in three calls: send
// an event to whoever is watching, sign a provider request with a credential
// the engine does not hold, and lend the tools that only make sense where a
// window is — the browser tab, the mouse. Today the screen is this very Engine
// and the calls are method calls; the interface exists so that the engine half
// can leave this package with its side of the contract intact, and so that
// when the engine is on another machine the same three calls cross a socket.
//
// Not *Engine itself, on purpose. Every exported method of Engine is a Wails
// binding, and a method that returns a func (ProviderTransport) or takes an
// interface (WindowTools) has no place on that wire — appScreen is the
// adapter, and Engine has no exported method the engine's contract would need.

import (
	"strings"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// Screen is what the engine asks of the window.
type Screen interface {
	// Emit sends one event to the window — the same 37 names emitEvent
	// carries, with the payload Wails would marshal.
	Emit(event string, data any)
	// ProviderTransport is the credential for a provider in a wire format,
	// as the model layer's Transport seam (provider_forward.go).
	ProviderTransport(provider, wireFormat string) model.Transport
	// WindowTools are the packs that act on the window's machine — browser
	// and computer — built for one session. The engine decides which of them
	// a session actually gets (workbenchSkills); the screen only offers.
	WindowTools(s Session) []skill.Skill
}

// Session is the little a window tool needs to know about the chat it acts
// for: which one, so its desk events land there, and where its files are.
type Session interface {
	ID() string
	Root() string
}

func (c *conversation) ID() string   { return c.id }
func (c *conversation) Root() string { return strings.TrimSpace(c.cfg.SandboxRoot) }

// appScreen is this window as the engine sees it.
type appScreen struct{ app *Engine }

func (s appScreen) Emit(event string, data any) { s.app.emitEvent(event, data) }

func (s appScreen) ProviderTransport(provider, wireFormat string) model.Transport {
	return s.app.providerTransport(provider, wireFormat)
}

func (s appScreen) WindowTools(sess Session) []skill.Skill {
	return []skill.Skill{
		// One tool for the browser, nine actions inside it (browser_tool.go).
		// The old per-action names are still what `tools:` and `categories:`
		// speak — they moved from being tools to being the actions' keys.
		&browserSkill{app: s.app, session: sess},
		// Driving programs on this machine (computer_tool.go). Offered always;
		// whether a session gets it is the engine's switch, not the window's.
		newComputerSkill(s.app, sess),
	}
}

// screenOf is the Screen this engine talks to: the one a test installed, else
// this window. Lazy, so an Engine built by struct literal has one without asking.
func (a *Engine) screenOf() Screen {
	if a.screen != nil {
		return a.screen
	}
	return appScreen{a}
}
