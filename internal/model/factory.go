package model

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	pvdr "github.com/Mikedev115/Aetox/internal/provider"
)

// Transport is the door through which a credential reaches a model request
// without this package ever holding it (§248). It is handed the network
// transport the client would have used and answers with the one it will use:
// the desktop wraps it in one that signs each request from the credential
// store, and a remote engine will hand back one that never dials at all and
// carries the request to the screen instead. When a provider is built with a
// Transport, it attaches no credential of its own and requires none — the
// transport is the credential.
type Transport func(network http.RoundTripper) http.RoundTripper

type ProviderOptions struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
	Timeout  time.Duration
	// Transport, when set, owns authentication — see the type. APIKey and
	// TokenSource are then ignored by the wire clients and not required.
	Transport Transport
	// TokenSource, Headers and SignedInEndpoint are what a sign-in
	// (internal/oauth) contributes for the provider, resolved by the CALLER:
	// this package reads no credential store, so that the engine half of the
	// app can link it without being able to reach one. TokenSource outranks
	// APIKey (the user did the more deliberate thing); Headers ride every
	// request (the ChatGPT backend routes on an account id); SignedInEndpoint
	// is a base URL the sign-in pinned, which wins over the catalog default
	// but never over a URL the user typed.
	TokenSource      func(context.Context) (string, error)
	Headers          map[string]string
	SignedInEndpoint string
	// TokenRefresh is consulted once when the provider answers 401 to a token
	// TokenSource handed out: it renews the token whatever the recorded expiry
	// said, and the request is sent again. The recorded expiry is a belief and
	// the 401 is the provider's word (oauth.Refresh). Nil means no second try —
	// the key path, and a sign-in with nothing to renew.
	TokenRefresh func(context.Context) (string, error)
	// WireFormat picks between a provider's two wire formats when it has one
	// (ProviderMetadata.AltRuntime/AltBaseURL) — e.g. "openai-compatible" or
	// "anthropic" for DeepSeek. Empty uses the catalog's default runtime.
	// Unknown values, or a provider with no alt, are ignored (default wins).
	WireFormat string
	// Locale ("th", "en") is consumed by the noop runtime alone — Aetox's own
	// built-in provider, which is an onboarding surface rather than a model and
	// therefore has to speak the user's language (ARCHITECTURE.md §40). Every
	// other runtime below ignores it.
	Locale string
}

func NewProvider(opts ProviderOptions) (Provider, error) {
	provider := NormalizeProvider(opts.Provider)
	if provider == "" {
		provider = "aetox"
	}

	info, ok := LookupProviderInfo(provider)
	if !ok {
		return nil, fmt.Errorf("unsupported model provider: %q", provider)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	// A Transport is the credential; nothing else is asked for.
	requireAPIKey := info.RequiresAPIKey && opts.Transport == nil

	runtime := resolveRuntime(info, opts.WireFormat)
	baseURL := opts.BaseURL
	if info.AltRuntime != "" && runtime == info.AltRuntime {
		// The default-format URL is wrong for the alt wire format — only a
		// user-customized URL survives the switch.
		if baseURL == "" || baseURL == info.BaseURL {
			baseURL = info.AltBaseURL
		}
	} else if info.AltBaseURL != "" && baseURL == info.AltBaseURL {
		// Symmetric: a stale alt-format URL under the default wire format.
		baseURL = info.BaseURL
	}

	// Every provider nobody has signed into arrives with a nil TokenSource,
	// which is exactly the path that existed before sign-ins did.
	tokenSource := opts.TokenSource
	if endpoint := strings.TrimSpace(opts.SignedInEndpoint); endpoint != "" && (baseURL == "" || baseURL == info.BaseURL) {
		// Some sign-ins name the host the account is served from. A base URL
		// the user typed themselves still wins over it.
		baseURL = endpoint
	}
	opts.BaseURL = baseURL

	switch runtime {
	case string(pvdr.RuntimeNoop):
		noop := NewNoopProvider(opts.Model)
		noop.Locale = opts.Locale
		return noop, nil
	case string(pvdr.RuntimeOllama):
		return NewOllamaProvider(OllamaConfig{
			Model:     opts.Model,
			BaseURL:   opts.BaseURL,
			Timeout:   timeout,
			Transport: opts.Transport,
		})
	case string(pvdr.RuntimeOpenAICompatible):
		return NewOpenAICompatibleProvider(OpenAICompatibleConfig{
			Provider:      provider,
			Model:         opts.Model,
			APIKey:        opts.APIKey,
			BaseURL:       opts.BaseURL,
			Timeout:       timeout,
			RequireAPIKey: &requireAPIKey,
			TokenSource:   tokenSource,
			TokenRefresh:  opts.TokenRefresh,
			Headers:       opts.Headers,
			Transport:     opts.Transport,
		})
	case string(pvdr.RuntimeResponses):
		return NewResponsesProvider(ResponsesConfig{
			Provider:     provider,
			Model:        opts.Model,
			APIKey:       opts.APIKey,
			BaseURL:      opts.BaseURL,
			Timeout:      timeout,
			TokenSource:  tokenSource,
			TokenRefresh: opts.TokenRefresh,
			Headers:      opts.Headers,
			Transport:    opts.Transport,
		})
	case string(pvdr.RuntimeAnthropic):
		return NewAnthropicProvider(AnthropicConfig{
			Provider:  provider,
			Model:     opts.Model,
			APIKey:    opts.APIKey,
			BaseURL:   opts.BaseURL,
			Timeout:   timeout,
			Transport: opts.Transport,
		})
	case string(pvdr.RuntimeExternalCLI):
		// A placeholder on purpose — see ExternalCLIProvider. The turn itself
		// never comes through here.
		return NewExternalCLIProvider(provider, opts.Model), nil
	default:
		return nil, fmt.Errorf("unsupported model provider: %q", provider)
	}
}

// resolveRuntime is the wire format a provider will actually speak: its
// catalog default, or its alt when the preference names that one. Unknown
// values, and a provider with no alt, fall back to the default.
func resolveRuntime(info ProviderMetadata, wireFormat string) string {
	if wf := strings.TrimSpace(wireFormat); wf != "" && info.AltRuntime != "" && wf == info.AltRuntime {
		return info.AltRuntime
	}
	return info.Runtime
}

// AuthScheme is how a request to this provider, in this wire format, carries
// an API key: the header and the prefix in front of the key. An empty header
// means the wire format carries no key at all (Ollama, the built-in). This is
// what a Transport needs to sign a request the wire client left unsigned, and
// it is the same knowledge the wire clients use — kept in one place so the
// two cannot disagree.
func AuthScheme(providerName, wireFormat string) (header, prefix string) {
	provider := NormalizeProvider(providerName)
	info, ok := LookupProviderInfo(provider)
	if !ok {
		return "", ""
	}
	switch resolveRuntime(info, wireFormat) {
	case string(pvdr.RuntimeAnthropic):
		return "x-api-key", ""
	case string(pvdr.RuntimeOpenAICompatible), string(pvdr.RuntimeResponses):
		return "Authorization", "Bearer "
	default:
		return "", ""
	}
}
