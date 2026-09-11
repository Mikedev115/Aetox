package main

// The screen's half of the model credential (§248 A3).
//
// The engine builds its provider clients without a key: bootstrap.Engine is
// handed a model.Transport instead, and the wire clients then attach no
// credential of their own. This file is that transport for the case where the
// screen and the engine are one process — it signs each request from the
// screen's own stores at the moment the request goes out, which is the same
// moment applyAuth used to do it, and so a token that expires mid-session is
// refreshed here per attempt exactly as before.
//
// Nothing about this is provisional. When the engine is a process of its own
// the transport on ITS side becomes a proxy that carries the unsigned request
// to the screen, and the screen's forwarder does what signedTransport does
// below: resolve a sign-in or a key for the provider named, put it on the
// request, send it. The engine never learns what was put on.

import (
	"fmt"
	"net/http"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

// providerTransport is the credential for provider, in the given wire format,
// as a model.Transport for bootstrap.Options.ProviderTransport.
func (a *App) providerTransport(canonical, wireFormat string) model.Transport {
	header, prefix := model.AuthScheme(canonical, wireFormat)
	return func(network http.RoundTripper) http.RoundTripper {
		return &signedTransport{provider: canonical, header: header, prefix: prefix, network: network}
	}
}

// signedTransport signs one provider's requests and hands them to the network.
//
// A sign-in (internal/oauth) outranks a pasted key for the same provider: the
// user did the more deliberate thing, and the two would otherwise disagree
// silently. Both are read per request rather than at construction — the token
// because it expires, the key because the settings page can change it under a
// running chat and the next request should carry the new one.
type signedTransport struct {
	provider string
	// header and prefix are how this wire format carries a key
	// (model.AuthScheme); an empty header is a wire format with no key.
	header, prefix string
	network        http.RoundTripper
}

func (t *signedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Cloned rather than edited: the caller still owns the request it handed
	// us, and a RoundTripper that modifies one is a documented mistake. Clone
	// carries Body and GetBody across, which is what the retry above needs.
	req = req.Clone(req.Context())
	// Extra headers the provider's credentials require — Copilot refuses a
	// request that does not identify an editor client, the ChatGPT backend
	// routes on an account id. Not a credential themselves, and wanted on
	// the key path as much as the sign-in path, as the factory always sent them.
	for name, value := range oauth.Headers(t.provider) {
		req.Header.Set(name, value)
	}
	if source := oauth.TokenSource(t.provider); source != nil {
		token, err := source(req.Context())
		if err != nil {
			return nil, fmt.Errorf("%s sign-in: %w", t.provider, err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return t.network.RoundTrip(req)
	}
	if t.header != "" {
		if key := resolveAPIKeyForProvider(t.provider); key != "" {
			req.Header.Set(t.header, t.prefix+key)
		}
	}
	return t.network.RoundTrip(req)
}
