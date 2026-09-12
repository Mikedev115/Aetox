package main

import (
	"errors"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

// providerFor is the factory a delegate uses when its profile names a
// provider of its own (`provider:` in AGENT.md — owner, 12 ก.ย. 2026: "ควร
// เลือกได้แม้แต่ผู้ให้บริการ และเลือกโมเดลได้ ทั้งเอเจนและซับเอเจน"). It builds the
// same kind of client the chat runs on: endpoint and key resolved from this
// screen's stores, a sign-in outranking a pasted key, the request signed by
// the transport rather than by a key the engine holds (§248). The chat's own
// provider keeps the chat's wire format and timeout; any other provider gets
// its catalog default wire format, because the config keeps one wire format
// — the chat's — and a DeepSeek agent under an OpenAI chat should speak
// DeepSeek's default, not OpenAI's.
//
// The second value is the provider's default model, for a profile that names
// the provider and leaves the model to it: the same lookup the settings page
// and the CLI use, so "OpenRouter, whatever is default" means the same thing
// everywhere. An empty answer there is an error the model can read — a local
// runtime with nothing loaded is the usual cause, and naming the endpoint is
// what sends the user to the right place.
func (a *App) providerFor(cfg config.Config) func(string) (model.Provider, string, error) {
	return func(name string) (model.Provider, string, error) {
		canonical := model.NormalizeProvider(name)
		if canonical == "" {
			return nil, "", errors.New("no provider named")
		}
		if _, ok := model.LookupProviderInfo(canonical); !ok {
			return nil, "", errors.New("unknown provider " + name)
		}
		baseURL := resolveBaseURLForProvider(canonical)
		wireFormat := ""
		timeout := 30 * time.Second
		if cfg.ModelTimeoutSec > 0 {
			timeout = time.Duration(cfg.ModelTimeoutSec) * time.Second
		}
		if canonical == model.NormalizeProvider(cfg.ModelProvider) {
			wireFormat = cfg.ModelWireFormat
		}
		defModel := strings.TrimSpace(model.ResolveDefaultModel(canonical, baseURL, resolveAPIKeyForProvider(canonical)))
		if defModel == "" {
			return nil, "", errors.New(canonical + " offered no model at " + baseURL + " — start its server and load a model")
		}
		p, err := model.NewProvider(model.ProviderOptions{
			Provider:         canonical,
			Model:            defModel,
			BaseURL:          baseURL,
			Timeout:          timeout,
			WireFormat:       wireFormat,
			Transport:        a.screenOf().ProviderTransport(canonical, wireFormat),
			SignedInEndpoint: oauth.Endpoint(canonical),
		})
		if err != nil {
			return nil, "", err
		}
		return p, defModel, nil
	}
}
