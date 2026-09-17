package main

// The credential desk (§248 B1): every binding that needs a provider's key
// in its hand — to list its models, ping it, read its balance, say whether a
// key is there, save one — lives here on the screen, where the key is. The
// engine never sees one; what it needs of a provider it asks through
// engine.Screen (screen.go), and what these doors need of the engine they
// ask through its bindings.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Mikedev115/Aetox/internal/cliagent"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/credentials"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
	"github.com/Mikedev115/Aetox/internal/signer"
)

// resolveAPIKeyForProvider is the screen's one door to a provider key: the
// store, else the provider's environment variable. The engine reaches the
// provider through the transport this signs (provider_forward.go) and reads
// no key of its own (§248 A4).
func resolveAPIKeyForProvider(canonicalProvider string) string {
	return credentials.KeyFor(canonicalProvider)
}

// AddCustomProvider creates a provider row of the user's own: an
// OpenAI-compatible endpoint under a name they chose, with its key filed
// under that name so the next endpoint they add cannot overwrite it. The row
// is enabled on the way out, since nobody adds one to keep it hidden. Returns
// the id the row will be known by everywhere else.
//
// The key is optional here only because the page it lands on has a key field
// of its own — a user who pastes it later is not refused a row today. keyFrom
// names a provider whose saved key should be copied when apiKey is empty:
// the "+" under a card's Base URL saves that card as a new row, and the key
// on the card is one the frontend can only ever see the tail of, so the copy
// has to happen here. "" copies nothing.
func (a *App) AddCustomProvider(name, baseURL, apiKey, keyFrom string) (string, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" && strings.TrimSpace(keyFrom) != "" {
		key = resolveAPIKeyForProvider(model.NormalizeProvider(keyFrom))
	}
	id, err := a.api.AddCustomProviderRow(name, baseURL)
	if err != nil {
		return "", err
	}
	// After the row exists, so the key's provider name normalizes to it.
	if key != "" {
		if err := credentials.Set(id, key); err != nil {
			return "", err
		}
	}
	// The screen's own note of where this row points, for the signer's
	// check when the engine is on a host (credentialMayRide).
	if err := rememberCustomEndpoint(id, baseURL); err != nil {
		return "", err
	}
	return id, nil
}

// RemoveCustomProvider deletes a user-added row and everything filed under
// its name — the key the screen holds, and the base URL override and the
// model last picked the engine holds — and returns the refreshed enabled
// list.
func (a *App) RemoveCustomProvider(id string) ([]string, error) {
	next, err := a.api.RemoveCustomProviderRow(id)
	if err != nil {
		return nil, err
	}
	// A key left under a name that is about to mean nothing would sit in the
	// secrets file forever, readable by nobody and deletable from nowhere.
	if err := credentials.Forget(id); err != nil {
		return nil, err
	}
	if err := forgetCustomEndpoint(id); err != nil {
		return nil, err
	}
	return next, nil
}

// ListModelsForProvider answers "what can I pick here": live API discovery
// first, then the catalog already on this disk, then the static recommended
// list. An empty result means "no known models" — the frontend should offer a
// free-text input for a custom model id.
//
// The middle step is the one worth explaining. Asking the endpoint is the only
// answer that is a fact, and it is also the step that fails for ordinary
// reasons: a key issued for one region on another region's host (Alibaba's
// Model Studio does exactly this, and answers 401), an offline laptop, a base
// URL typed with a typo. The chain used to fall from that straight onto one
// hard-coded name per provider, so the picker showed a shelf of ONE — while
// model-catalog.json, sitting in the same data root and read by the price
// column two lines away, described 54 models for that same provider.
//
// It stays below live discovery and can never override it: the catalog says
// what models.dev publishes, not what this account is entitled to on this
// endpoint today.
func (a *App) ListModelsForProvider(providerName string) []string {
	canonical := model.NormalizeProvider(providerName)
	baseURL := a.api.ProviderBaseURL(canonical)
	apiKey := resolveAPIKeyForProvider(canonical)
	if choices, err := model.ModelChoicesWithEndpointAndAPIKey(canonical, baseURL, apiKey); err == nil && len(choices) > 0 {
		if canonical == "codex" {
			if localRoot, err := config.DataRoot(); err == nil && localRoot != "" {
				if rows, err := model.LoadResponsesModelFacts(localRoot); err == nil && len(rows) > 0 {
					go func() {
						_ = a.api.SyncResponsesModelFacts(rows)
					}()
				}
			}
		}
		return choices
	}
	if choices := a.api.CatalogModelChoices(canonical); len(choices) > 0 {
		return choices
	}
	if choices := model.ModelChoices(canonical); choices != nil {
		return choices
	}
	return []string{}
}

// ServiceTiersFor returns the processing lanes advertised by the Codex model
// catalog discovered on this machine. Keeping this next to model discovery
// means the UI never has to guess which models support Fast mode or hard-code
// a speed multiplier that can differ by model.
func (a *App) ServiceTiersFor(providerName, modelName string) []model.ServiceTier {
	return model.SupportedServiceTiers(providerName, modelName)
}

// ProviderAccountFor answers for one provider: the Settings card the user
// opened, or the one provider actually in use for the profile menu.
//
// One at a time on purpose. Fetching every enabled provider at once meant
// spending a round trip on companies the user was not talking to, to fill rows
// that no longer exist — the menu shows the provider in use and nothing else.
//
// Never returns an error. A provider being unreachable is a fact about that
// provider, carried in the Error field, not a reason to blank the panel.
func (a *App) ProviderAccountFor(providerName string) engine.ProviderAccount {
	return a.providerAccount(providerName)
}

func (a *App) providerAccount(providerName string) engine.ProviderAccount {
	canonical := model.NormalizeProvider(providerName)
	account := engine.ProviderAccount{
		Provider:     canonical,
		ExpectsQuota: model.StatesQuota(canonical),
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	balance, err := model.FetchBalance(
		ctx, canonical,
		a.api.ProviderBaseURL(canonical),
		resolveAPIKeyForProvider(canonical),
	)
	account.Balance = balance
	if err != nil {
		account.Error = err.Error()
	}

	// What the engine saw on the headers of turns: the window is the engine's
	// to observe, the balance the screen's to fetch.
	quotas, known := a.api.ProviderQuotas(canonical)
	account.Quotas, account.QuotaKnown = quotas, known

	// Two providers serve their window from an endpoint rather than on the
	// headers of turns: OpenRouter states it beside the credits, and the
	// OpenCode Go plan answers all three of its windows at /usage. Both have
	// an answer before any turn has run, which is the whole point — a fresh
	// subscription should not have to spend a turn to show what is left.
	if len(balance.Quotas) > 0 {
		account.Quotas = balance.Quotas
		account.QuotaKnown, account.QuotaFetched = true, true
		a.api.NoteProviderQuotas(canonical, balance.Quotas)
	}
	return account
}

// TestProviderConnection proves a provider is actually reachable by running a
// minimal 1-token completion through the same client chat uses — endpoint,
// key, and wire format all verified in one shot. modelName picks which model
// to ping, so a model can be proven before switching to it; empty falls back
// to the active model for this provider, else the catalog default. Returns the
// latency label on success; the error carries the provider's real failure
// message.
func (a *App) TestProviderConnection(providerName, modelName string) (string, error) {
	canonical := model.NormalizeProvider(providerName)
	baseURL := a.api.ProviderBaseURL(canonical)
	apiKey := resolveAPIKeyForProvider(canonical)
	fallback, wireFormat := a.api.ActiveModelFor(canonical)
	if fallback == "" {
		fallback = model.ResolveDefaultModel(canonical, baseURL, apiKey)
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		modelName = fallback
	}
	return signer.Probe(canonical, modelName, baseURL, apiKey, wireFormat)
}

// HasAPIKey reports whether a key-requiring provider already has resolvable
// credentials — a cached key, an env var, or a sign-in. Always true for
// providers that don't need any.
func (a *App) HasAPIKey(providerName string) bool {
	canonical := model.NormalizeProvider(providerName)
	if !model.RequiresAPIKey(canonical) {
		return true
	}
	// A signed-in provider has no key to find and never will — asking the user
	// for one would be asking for something that does not exist.
	if oauth.Has(canonical) {
		return true
	}
	return resolveAPIKeyForProvider(canonical) != ""
}

// APIKeyHint is the last few characters of the key this provider would
// actually be called with, for a field that is otherwise blank once a key is
// saved. The row went green, the placeholder said "already set", and neither
// told the owner *which* key was sitting there — so a key pasted into the
// wrong row looked exactly like a key pasted into the right one.
//
// It reads through resolveAPIKeyForProvider rather than the credential store
// alone, so what is shown is what would be sent: a pasted key shadows an
// environment one, and the hint follows that precedence instead of describing
// a key the engine would not use.
//
// Four characters, and only from a key long enough that four is a small part
// of it. Below that the whole string is dots — a hint is meant to distinguish
// two keys the owner already holds, not to reconstruct one over someone's
// shoulder. Signed-in providers return "" because there is no key to hint at.
func (a *App) APIKeyHint(providerName string) string {
	canonical := model.NormalizeProvider(providerName)
	if oauth.Has(canonical) {
		return ""
	}
	key := strings.TrimSpace(resolveAPIKeyForProvider(canonical))
	if key == "" {
		return ""
	}
	const reveal = 4
	r := []rune(key)
	if len(r) < reveal*3 {
		return strings.Repeat("•", 8)
	}
	return strings.Repeat("•", 4) + string(r[len(r)-reveal:])
}

// ProviderReady answers the question the sidebar dot is actually asking: can
// this provider be used right now?
//
// HasAPIKey was standing in for it and is not the same question. It returns
// true for anything that needs no key, which is every local runtime — so LM
// Studio and Ollama showed a green dot whether or not a server was listening,
// on a page that said "no models found" two inches to the right.
//
// What "ready" can honestly mean differs by kind, and the split is about what
// can be checked for free:
//
//   - A keyed provider: a key is set, or a sign-in exists. Whether that key
//     still works is only knowable by spending a request against it, and a
//     settings page that bills the user for opening it is worse than one that
//     says "configured".
//   - A local runtime: the server is answering. This costs a connection to
//     localhost, which is free, so there is no excuse for guessing — and it is
//     precisely the case that was lying.
//   - Aetox's own engine: always, it is built in.
func (a *App) ProviderReady(providerName string) bool {
	canonical := model.NormalizeProvider(providerName)
	if model.RequiresAPIKey(canonical) {
		return a.HasAPIKey(canonical)
	}
	if canonical == "aetox" {
		return true
	}
	// A provider served by an external program is ready when the program is
	// found and signed in — on the engine's machine, which is where it runs
	// (cli_engine.go); the registry itself is a package the screen can read.
	// Asked before the catalog's fallback-model check below, which would
	// answer "yes" for it: a fallback model name says nothing about whether
	// the program exists.
	if _, ok := cliagent.For(canonical); ok {
		return a.api.ExternalEngineStatus(canonical).Ready
	}
	// The same judgement the rest of the app makes about a local runtime — can
	// a model be got out of it — rather than a second definition of "up".
	return model.ResolveDefaultModel(canonical, a.api.ProviderBaseURL(canonical), "") != ""
}

// SetAPIKey persists an API key for a provider and, if it's the active
// provider, immediately re-bootstraps the engine — which picks the key up
// through the screen's signing transport, never from its config (§248 A4).
func (a *App) SetAPIKey(providerName, apiKey string) (engine.ModelInfo, error) {
	canonical := model.NormalizeProvider(providerName)
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return engine.ModelInfo{}, fmt.Errorf("API key cannot be empty")
	}
	if err := credentials.Set(canonical, key); err != nil {
		return engine.ModelInfo{}, err
	}
	return a.api.ProviderCredentialChanged(canonical)
}
