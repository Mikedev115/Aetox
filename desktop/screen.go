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
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
	"github.com/Mikedev115/Aetox/internal/skill"
)

type appScreen struct{ app *App }

func (s appScreen) Emit(event string, data any) { s.app.emitEvent(event, data) }

func (s appScreen) ProviderTransport(provider, wireFormat string) model.Transport {
	return s.app.providerTransport(provider, wireFormat)
}

func (s appScreen) ProviderEndpoint(provider string) string { return oauth.Endpoint(provider) }

func (s appScreen) WindowTools(sess engine.Session) []skill.Skill {
	// The packs still live in the engine package until browser_*.go and
	// computer_*.go move here; WindowPacks goes with them.
	return engine.WindowPacks(s.app.eng, sess)
}

func (s appScreen) DefaultModel(provider, baseURL string) string {
	return model.ResolveDefaultModel(provider, baseURL, resolveAPIKeyForProvider(provider))
}

func (s appScreen) Probe(provider, modelName, baseURL, wireFormat string) (string, error) {
	return probeProvider(provider, modelName, baseURL, resolveAPIKeyForProvider(provider), wireFormat)
}

func (s appScreen) ModelResident(provider, baseURL, modelName string) bool {
	return model.LocalModelResident(provider, baseURL, resolveAPIKeyForProvider(provider), modelName)
}

// probeProvider is the ping itself: a 1-token completion through the same
// client chat uses, so endpoint, key and wire format are all proven at once.
// The screen's own ping, so it may hold the key it is proving; the sign-in,
// if any, is looked up here for the same reason the CLI does it — the model
// layer reads no credential store (§248 A3).
func probeProvider(canonical, modelName, baseURL, apiKey, wireFormat string) (string, error) {
	p, err := model.NewProvider(model.ProviderOptions{
		Provider:         canonical,
		Model:            modelName,
		APIKey:           apiKey,
		BaseURL:          baseURL,
		Timeout:          15 * time.Second,
		WireFormat:       wireFormat,
		TokenSource:      oauth.TokenSource(canonical),
		Headers:          oauth.Headers(canonical),
		SignedInEndpoint: oauth.Endpoint(canonical),
	})
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	start := time.Now()
	_, err = p.Complete(ctx, model.Request{
		Model:     modelName,
		Messages:  []model.Message{{Role: model.RoleUser, Content: "ping"}},
		MaxTokens: 1,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s · %dms", modelName, time.Since(start).Milliseconds()), nil
}

var _ engine.Screen = appScreen{}

// unused guard so the http import stays honest when the transport moves.
var _ http.RoundTripper = (*signedTransport)(nil)
