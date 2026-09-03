package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Generic MCP server sign-in.
//
// Every other flow in this package is written against one named provider,
// with its authorize/token endpoints and (for codex) its client id typed in
// as constants. That does not scale to "whichever MCP server the shelf
// offers next" — a fixed list of hand-written flows is exactly the pattern
// internal/connect's own package comment describes replacing once for
// GitHub-shaped bindings. So this file knows the name of exactly nothing: it
// discovers where a server wants its OAuth conducted (RFC 9728, then RFC
// 8414) and registers itself as a client on the fly (RFC 7591) rather than
// carrying a client id Aetox would otherwise have to obtain from each vendor
// by hand.
//
// That trade has a real edge, found by actually probing the six OAuth-only
// MCP servers under consideration on 2026-09-03: three of them (semgrep,
// grafana, netlify) answer a registration_endpoint and this flow signs into
// them with no setup at all. Three (elevenlabs, vercel, shopify) answer full
// authorization-server metadata with no registration_endpoint — a real
// server, genuinely OAuth-only, that this flow genuinely cannot reach without
// Aetox registering a fixed client id with that vendor first, which is a
// business step and not a coding one. StartMCPOAuth says so plainly rather
// than failing on some unrelated error, so the difference is diagnosable
// rather than mysterious.

// resourceMetadataRef pulls resource_metadata="..." out of a WWW-Authenticate
// challenge, per RFC 9728 §5.1 — the pointer a protected MCP server hands
// back on an unauthenticated request, naming where to read the rest.
var resourceMetadataRef = regexp.MustCompile(`resource_metadata="([^"]+)"`)

// protectedResourceMetadata is the RFC 9728 document at that pointer.
type protectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
}

// authorizationServerMetadata is the RFC 8414 document the authorization
// server itself publishes — only the three fields a sign-in needs.
type authorizationServerMetadata struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	// RegistrationEndpoint is RFC 7591's door. Its absence is not a discovery
	// failure — it is the answer "this server does not support dynamic
	// registration", and StartMCPOAuth reports it as such.
	RegistrationEndpoint string `json:"registration_endpoint"`
}

type mcpOAuthMeta struct {
	AuthorizationEndpoint string
	TokenEndpoint         string
	RegistrationEndpoint  string
}

// discoverMCPOAuth finds where an MCP server wants its OAuth conducted, from
// nothing but the server's own URL. Three hops: an unauthenticated request to
// the server itself names the resource-metadata document, that document names
// the authorization server, and the authorization server's own well-known
// document names the endpoints a sign-in needs.
func discoverMCPOAuth(ctx context.Context, resourceURL string) (*mcpOAuthMeta, error) {
	metaURL, err := resourceMetadataURL(ctx, resourceURL)
	if err != nil {
		return nil, err
	}
	var resource protectedResourceMetadata
	if err := getJSON(ctx, metaURL, &resource); err != nil {
		return nil, fmt.Errorf("reading %s: %w", metaURL, err)
	}
	if len(resource.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("%s named no authorization server", metaURL)
	}

	asMeta, err := discoverAuthorizationServer(ctx, resource.AuthorizationServers[0])
	if err != nil {
		return nil, err
	}
	if asMeta.AuthorizationEndpoint == "" || asMeta.TokenEndpoint == "" {
		return nil, fmt.Errorf("%s is missing an authorization or token endpoint", resource.AuthorizationServers[0])
	}
	return &mcpOAuthMeta{
		AuthorizationEndpoint: asMeta.AuthorizationEndpoint,
		TokenEndpoint:         asMeta.TokenEndpoint,
		RegistrationEndpoint:  asMeta.RegistrationEndpoint,
	}, nil
}

// resourceMetadataURL gets the RFC 9728 resource-metadata URL a server names
// on an unauthenticated request's 401. A GET is enough — every server this
// was verified against challenges on any method, MCP's own POST included.
func resourceMetadataURL(ctx context.Context, resourceURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", clientUserAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("%s did not challenge for authorization (got %s) — nothing to discover", resourceURL, resp.Status)
	}
	m := resourceMetadataRef.FindStringSubmatch(resp.Header.Get("WWW-Authenticate"))
	if m == nil {
		return "", fmt.Errorf("%s returned 401 with no resource_metadata reference — this server may not support MCP's OAuth discovery", resourceURL)
	}
	return m[1], nil
}

// wellKnownCandidates lists the RFC 8414 document URLs to try, in the order
// the RFC prefers: path-insertion (the well-known segment sits at the
// authorization server's root, ahead of its own path) before path-suffix (the
// document lives under the server's own path instead) — and
// oauth-authorization-server before openid-configuration, since every server
// probed while this was written used the former.
func wellKnownCandidates(as string) []string {
	u, err := url.Parse(as)
	if err != nil || u.Host == "" {
		return nil
	}
	origin := u.Scheme + "://" + u.Host
	path := strings.TrimSuffix(u.Path, "/")

	var out []string
	for _, name := range []string{"oauth-authorization-server", "openid-configuration"} {
		if path != "" {
			out = append(out, origin+"/.well-known/"+name+path)
		}
		out = append(out, origin+"/.well-known/"+name)
		if path != "" {
			out = append(out, strings.TrimSuffix(as, "/")+"/.well-known/"+name)
		}
	}
	return out
}

func discoverAuthorizationServer(ctx context.Context, as string) (*authorizationServerMetadata, error) {
	var lastErr error
	for _, candidate := range wellKnownCandidates(as) {
		var meta authorizationServerMetadata
		if err := getJSON(ctx, candidate, &meta); err != nil {
			lastErr = err
			continue
		}
		if meta.AuthorizationEndpoint != "" && meta.TokenEndpoint != "" {
			return &meta, nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no well-known document answered")
	}
	return nil, fmt.Errorf("could not find %s's OAuth metadata: %w", as, lastErr)
}

func getJSON(ctx context.Context, target string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", clientUserAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	return readJSON(resp, out)
}

// registerMCPClient performs RFC 7591 dynamic client registration and returns
// the client id the authorization server minted. Always a public client —
// token_endpoint_auth_method "none" — because this is a PKCE flow run from a
// desktop app with no way to keep a client secret, the same shape every
// public MCP client registers as.
func registerMCPClient(ctx context.Context, registrationEndpoint, redirectURI string) (string, error) {
	body, err := json.Marshal(struct {
		RedirectURIs            []string `json:"redirect_uris"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
		ClientName              string   `json:"client_name"`
	}{
		RedirectURIs:            []string{redirectURI},
		TokenEndpointAuthMethod: "none",
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		ClientName:              "Aetox",
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", clientUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	if err := readJSON(resp, &out); err != nil {
		return "", fmt.Errorf("registering with %s: %w", registrationEndpoint, err)
	}
	if out.ClientID == "" {
		return "", fmt.Errorf("registering with %s: no client_id in response", registrationEndpoint)
	}
	return out.ClientID, nil
}

// StartMCPOAuth discovers, registers, and opens a browser sign-in for one MCP
// server. serverName is what the credential is stored under — the same id
// `${connect:serverName}` reads back in an MCP server's headers
// (internal/bootstrap/config.go).
func StartMCPOAuth(ctx context.Context, serverName, resourceURL string) (*Pending, error) {
	meta, err := discoverMCPOAuth(ctx, resourceURL)
	if err != nil {
		return nil, err
	}
	if meta.RegistrationEndpoint == "" {
		return nil, fmt.Errorf("%s does not support automatic sign-in — it has no dynamic client registration, so it needs a client id Aetox has not registered with it", resourceURL)
	}

	lb, err := StartLoopback(0, "/callback")
	if err != nil {
		return nil, err
	}
	clientID, err := registerMCPClient(ctx, meta.RegistrationEndpoint, lb.RedirectURI)
	if err != nil {
		lb.Close()
		return nil, err
	}
	verifier, challenge, err := NewPKCE()
	if err != nil {
		lb.Close()
		return nil, err
	}
	state, err := randomString(32)
	if err != nil {
		lb.Close()
		return nil, err
	}

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", lb.RedirectURI)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)

	return &Pending{
		URL:           meta.AuthorizationEndpoint + "?" + q.Encode(),
		Verifier:      verifier,
		State:         state,
		provider:      serverName,
		lb:            lb,
		tokenEndpoint: meta.TokenEndpoint,
		clientID:      clientID,
	}, nil
}

// FinishMCPOAuth waits for the browser redirect, exchanges the code, and
// stores the credential under the server name StartMCPOAuth was given.
func FinishMCPOAuth(ctx context.Context, pending *Pending) error {
	if pending == nil || pending.lb == nil {
		return errors.New("no sign-in in progress")
	}
	defer pending.Cancel()

	code, state, err := pending.lb.Wait(ctx)
	if err != nil {
		return err
	}
	// Empty state tolerated, not required — see FinishCodex's identical
	// comment: some authorization servers do not echo it back, and rejecting
	// those outright would fail a sign-in that came back from the real
	// redirect. Anything present that mismatches is fatal.
	if state != "" && state != pending.State {
		return errors.New("authorization code does not match this sign-in — start again")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", pending.lb.RedirectURI)
	form.Set("client_id", pending.clientID)
	form.Set("code_verifier", pending.Verifier)

	tokens, err := mcpOAuthToken(ctx, pending.tokenEndpoint, form)
	if err != nil {
		return err
	}
	return Set(pending.provider, Credential{
		Type:          "oauth",
		Access:        tokens.AccessToken,
		Refresh:       tokens.RefreshToken,
		ExpiresAt:     tokens.expiresAt(),
		TokenEndpoint: pending.tokenEndpoint,
		ClientID:      pending.clientID,
		Label:         pending.provider,
	})
}

// refreshMCPOAuth renews a credential StartMCPOAuth minted — the generic
// counterpart to refreshCodex, usable for any server signed into through the
// discovery flow above rather than a fixed, compiled-in one. token.go.Token
// reaches this when a credential carries a TokenEndpoint but no entry in
// refreshers.
func refreshMCPOAuth(ctx context.Context, cred Credential) (Credential, error) {
	if strings.TrimSpace(cred.Refresh) == "" {
		return Credential{}, errors.New("this sign-in has no refresh token — sign in again")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.Refresh)
	form.Set("client_id", cred.ClientID)

	tokens, err := mcpOAuthToken(ctx, cred.TokenEndpoint, form)
	if err != nil {
		return Credential{}, err
	}
	next := cred
	next.Access = tokens.AccessToken
	next.ExpiresAt = tokens.expiresAt()
	if tokens.RefreshToken != "" {
		// Not every server rotates the refresh token on use; keep the old one
		// when a fresh one is not handed back rather than losing it.
		next.Refresh = tokens.RefreshToken
	}
	return next, nil
}

func mcpOAuthToken(ctx context.Context, tokenEndpoint string, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", clientUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	var tokens tokenResponse
	if err := readJSON(resp, &tokens); err != nil {
		return tokenResponse{}, fmt.Errorf("exchanging token at %s: %w", tokenEndpoint, err)
	}
	if tokens.AccessToken == "" {
		return tokenResponse{}, errors.New("sign-in returned no access token")
	}
	return tokens, nil
}
