// Package signer is the screen's half of the model credential (§248 A3),
// lifted out of the desktop so every screen signs the same way.
//
// The engine builds its provider clients without a key: bootstrap.Engine is
// handed a model.Transport instead, and the wire clients then attach no
// credential of their own. This package is that transport — it signs each
// request from the screen's own stores at the moment the request goes out,
// so a token that expires mid-session is refreshed per attempt. The desktop
// window (desktop/provider_forward.go) and the console (cmd/aetox) are two
// screens with two credential stores and one way of putting a credential on
// a request; before this package the console was the one host that still
// handed a key straight to the provider, which meant a second code path for
// the same wire and a second place to get the 401 retry wrong.
//
// The engine never learns what was put on. When it runs on another machine
// (§248 phase 3) the request crosses to the screen unsigned and is signed
// here on the way out — which is why MayRide exists: a host that has been
// taken over could ask the screen to sign a request addressed to a server of
// its own, and the screen decides whether the destination may carry the key.
package signer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

// Options is what a screen brings to the signer: where its keys are, and
// whether a destination may carry one.
type Options struct {
	// Provider is the canonical provider id the transport signs for.
	Provider string
	// WireFormat is the runtime format in use, which decides how a key rides
	// (model.AuthScheme) — an Anthropic-format endpoint wants x-api-key, an
	// OpenAI-compatible one a Bearer.
	WireFormat string
	// Key answers with the pasted key for a provider, "" when there is none.
	// Read per request, because the settings page can change it under a
	// running chat and the next request should carry the new one.
	Key func(provider string) string
	// MayRide says whether this request's destination may carry the
	// credential at all; nil means always.
	MayRide func(provider string, u *url.URL) bool
}

// Transport is the credential for opts.Provider, in the given wire format,
// as a model.Transport for bootstrap.Options.ProviderTransport.
func Transport(opts Options) model.Transport {
	header, prefix := model.AuthScheme(opts.Provider, opts.WireFormat)
	return func(network http.RoundTripper) http.RoundTripper {
		return &signedTransport{opts: opts, header: header, prefix: prefix, network: network}
	}
}

// signedTransport signs one provider's requests and hands them to the network.
//
// A sign-in (internal/oauth) outranks a pasted key for the same provider: the
// user did the more deliberate thing, and the two would otherwise disagree
// silently. Both are read per request rather than at construction — the token
// because it expires, the key because it can change under a running chat.
type signedTransport struct {
	opts Options
	// header and prefix are how this wire format carries a key
	// (model.AuthScheme); an empty header is a wire format with no key.
	header, prefix string
	network        http.RoundTripper
}

func (t *signedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Cloned rather than edited: the caller still owns the request it handed
	// us, and a RoundTripper that modifies one is a documented mistake. Clone
	// carries Body and GetBody across, which is what the retry below needs.
	req = req.Clone(req.Context())
	if t.opts.MayRide != nil && !t.opts.MayRide(t.opts.Provider, req.URL) {
		return t.network.RoundTrip(req)
	}
	// Extra headers the provider's credentials require — Copilot refuses a
	// request that does not identify an editor client, the ChatGPT backend
	// routes on an account id. Not a credential themselves, and wanted on
	// the key path as much as the sign-in path, as the factory always sent them.
	for name, value := range oauth.Headers(t.opts.Provider) {
		req.Header.Set(name, value)
	}
	if source := oauth.TokenSource(t.opts.Provider); source != nil {
		token, err := source(req.Context())
		if err != nil {
			return nil, fmt.Errorf("%s sign-in: %w", t.opts.Provider, err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := t.network.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusUnauthorized || req.GetBody == nil {
			return resp, err
		}
		// The provider's 401 outranks the expiry the store recorded — the
		// ChatGPT backend has answered token_expired to a token whose claim
		// said a week remained. Renew once and send the same request again;
		// this is the signer's job, not the wire client's, because on this
		// path the wire client holds no token to renew (§248 A3). A renewal
		// that fails leaves the original 401 to be reported: that one means
		// sign in again.
		token, rerr := oauth.Refresh(req.Context(), t.opts.Provider)
		if rerr != nil {
			return resp, nil
		}
		body, berr := req.GetBody()
		if berr != nil {
			return resp, nil
		}
		resp.Body.Close()
		req = req.Clone(req.Context())
		req.Body = body
		req.Header.Set("Authorization", "Bearer "+token)
		return t.network.RoundTrip(req)
	}
	if t.header != "" && t.opts.Key != nil {
		if key := t.opts.Key(t.opts.Provider); key != "" {
			req.Header.Set(t.header, t.prefix+key)
		}
	}
	return t.network.RoundTrip(req)
}

// Probe is the ping itself: a 1-token completion through the same client
// chat uses, so endpoint, key and wire format are all proven at once. The
// screen's own ping, so it may hold the key it is proving; the sign-in, if
// any, is looked up here for the same reason the transport does it — the
// model layer reads no credential store (§248 A3). The label on success,
// the provider's real failure otherwise.
func Probe(canonical, modelName, baseURL, apiKey, wireFormat string) (string, error) {
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
