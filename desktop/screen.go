package main

// This window as the engine sees it (§248 A6, B1): engine.Screen, implemented
// by an adapter rather than by App itself, because every exported method of
// App is a Wails binding and a method that returns a func (ProviderTransport)
// or takes an interface (WindowTools) has no place on that wire.
//
// The desk questions — DefaultModel, Probe, ModelResident — are the ones the
// engine cannot answer without a provider key. The screen holds the key
// (internal/credentials), asks the provider, and hands back only the answer.
// In phase 2 these are the `screen.*` calls the engine makes across the
// socket, and the engine on the other end is exactly as ignorant of the key
// as it is today.

import (
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
	"github.com/Mikedev115/Aetox/internal/signer"
	"github.com/Mikedev115/Aetox/internal/skill"
)

type appScreen struct{ app *App }

func (s appScreen) Emit(event string, data any) { s.app.emitEvent(event, data) }

func (s appScreen) ProviderTransport(provider, wireFormat string) model.Transport {
	return s.app.providerTransport(provider, wireFormat)
}

func (s appScreen) ProviderEndpoint(provider string) string { return oauth.Endpoint(provider) }

func (s appScreen) WindowTools(sess engine.Session) []skill.Skill {
	return []skill.Skill{
		// One tool for the browser, nine actions inside it (browser_tool.go).
		// The old per-action names are still what `tools:` and `categories:`
		// speak — they moved from being tools to being the actions' keys.
		&browserSkill{app: s.app, session: sess},
		// Driving programs on this machine (computer_tool.go). Offered always;
		// whether a session gets it is the engine's switch, not the window's.
		newComputerSkill(s.app, sess),
		// The guide walking the UI (guide_tool.go). Desk guide gets it exclusively.
		newGuideSkill(s.app, sess),
	}
}

// AgentTab is the agent's live browsing tab, peeked rather than taken — see
// agentTabPeek for the message taking it would swallow.
func (s appScreen) AgentTab() string { return s.app.agentTabPeek() }

// modelSees says whether the model of the chat on screen can read a picture:
// the gate the browser and the machine share before putting an image on the
// wire. A model with no eyes gets the path and the tool that reads it.
func (a *App) modelSees() bool {
	info := a.api.GetModelInfo()
	return model.ResolveVision(info.Provider, info.ModelName)
}

// deskEvent raises one workbench event, stamped with the chat it happened in
// — the screen's copy of the engine's door (desk_events.go), for the tools
// that act here.
func (a *App) deskEvent(sessionID, event string, payload map[string]string) {
	if payload == nil {
		payload = map[string]string{}
	}
	payload["sessionId"] = sessionID
	a.emitEvent("workbench:"+event, payload)
}

func (s appScreen) DefaultModel(provider, baseURL string) string {
	return model.ResolveDefaultModel(provider, baseURL, resolveAPIKeyForProvider(provider))
}

func (s appScreen) Probe(provider, modelName, baseURL, wireFormat string) (string, error) {
	return signer.Probe(provider, modelName, baseURL, resolveAPIKeyForProvider(provider), wireFormat)
}

func (s appScreen) ModelResident(provider, baseURL, modelName string) bool {
	return model.LocalModelResident(provider, baseURL, resolveAPIKeyForProvider(provider), modelName)
}

var _ engine.Screen = appScreen{}
