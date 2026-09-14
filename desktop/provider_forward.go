package main

// The screen's half of the model credential (§248 A3).
//
// The engine builds its provider clients without a key: bootstrap.Engine is
// handed a model.Transport instead, and the wire clients then attach no
// credential of their own. The transport itself — sign-in over key, refresh
// once on a 401, the extra headers a provider's credential needs — lives in
// internal/signer, shared with the console screen (cmd/aetox). What is this
// window's alone is here: where its keys are (internal/credentials) and
// whether a destination may carry one (credentialMayRide).
//
// When the engine is a process of its own the transport on ITS side becomes
// a proxy that carries the unsigned request to the screen, and the screen's
// forwarder does what the signer does: resolve a sign-in or a key for the
// provider named, put it on the request, send it. The engine never learns
// what was put on.

import (
	"net"
	"net/url"
	"strings"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
	"github.com/Mikedev115/Aetox/internal/signer"
)

// providerTransport is the credential for provider, in the given wire format,
// as a model.Transport for bootstrap.Options.ProviderTransport.
func (a *App) providerTransport(canonical, wireFormat string) model.Transport {
	return signer.Transport(signer.Options{
		Provider:   canonical,
		WireFormat: wireFormat,
		Key:        resolveAPIKeyForProvider,
		MayRide:    a.credentialMayRide,
	})
}

// credentialMayRide reports whether a credential for provider may be put on
// a request to this URL.
//
// With the engine a child of this process the answer is yes: it runs on
// this machine's ground, and a URL it built came from settings on this
// disk. With the engine on a host over ssh (§248 phase 3) the request was
// built by another machine's process, and a host that has been taken over
// could ask the screen to sign one addressed to a server of its own — the
// one way decision 3's key could still leave this machine. So there a
// credential rides only to where the screen itself knows the provider
// lives: the catalog's endpoints, the sign-in's endpoint, a custom row's
// endpoint as recorded when it was added here, and loopback — a runtime on
// this machine, which is what a host's "localhost" means once the request
// is sent from here. Anywhere else the request goes out bare and fails at
// the far end with the provider's own words, rather than carrying the key
// to whoever asked. An engine attached by hand (AETOX_ENGINE_ADDR) that
// named another machine in its hello is on a host too (elsewhere).
func (a *App) credentialMayRide(provider string, u *url.URL) bool {
	if a.engine == nil || !a.engine.elsewhere() {
		return true
	}
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	if host == "localhost" || host == "::1" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()) {
		return true
	}
	known := []string{oauth.Endpoint(provider)}
	if info, ok := model.LookupProviderInfo(provider); ok {
		known = append(known, info.BaseURL, info.AltBaseURL)
	}
	known = append(known, customProviderEndpoints(provider)...)
	for _, k := range known {
		if k == "" {
			continue
		}
		if ku, err := url.Parse(k); err == nil && strings.EqualFold(ku.Hostname(), host) {
			return true
		}
	}
	debuglog.Msg("provider: %s asked for a credential on %s://%s, which the screen does not know for it — sent unsigned", provider, u.Scheme, u.Host)
	return false
}
