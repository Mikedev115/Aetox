package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
//
// The quotes are optional. RFC 9728 writes them, and so do Figma, Sentry
// and Vercel; Stripe (mcp.stripe.com, probed 2026-09-13) writes
// `resource_metadata=https://…` bare, and a regex that insisted on the
// quotes turned a working server into "may not support MCP's OAuth
// discovery".
var resourceMetadataRef = regexp.MustCompile(`resource_metadata="?([^",\s]+)"?`)

// initializeProbe is the smallest request a Streamable HTTP MCP server will
// answer: a JSON-RPC initialize. The challenge is fetched with it because a
// server may serve MCP on POST only — Figma's answers GET with 405 and
// challenges only a POST — and a probe that only ever asked with GET read
// that 405 as "did not challenge for authorization".
const initializeProbe = `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"aetox","version":"0"}}}`

// protectedResourceMetadata is the RFC 9728 document at that pointer.
type protectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

// authorizationServerMetadata is the RFC 8414 document the authorization
// server itself publishes — only the fields a sign-in needs.
type authorizationServerMetadata struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	// RegistrationEndpoint is RFC 7591's door. Its absence is not a discovery
	// failure — it is the answer "this server does not support dynamic
	// registration", and StartMCPOAuth reports it as such.
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
}

type mcpOAuthMeta struct {
	AuthorizationEndpoint string
	TokenEndpoint         string
	RegistrationEndpoint  string
	ClientAuthMethod      string
	Scopes                []string
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
	as, resourceScopes, err := authorizationServerFor(ctx, resourceURL, metaURL)
	if err != nil {
		return nil, err
	}

	asMeta, err := discoverAuthorizationServer(ctx, as)
	if err != nil {
		return nil, err
	}
	if asMeta.AuthorizationEndpoint == "" || asMeta.TokenEndpoint == "" {
		return nil, fmt.Errorf("%s is missing an authorization or token endpoint", as)
	}
	authMethod, err := chooseMCPClientAuth(asMeta.TokenEndpointAuthMethodsSupported)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", as, err)
	}
	return &mcpOAuthMeta{
		AuthorizationEndpoint: asMeta.AuthorizationEndpoint,
		TokenEndpoint:         asMeta.TokenEndpoint,
		RegistrationEndpoint:  asMeta.RegistrationEndpoint,
		ClientAuthMethod:      authMethod,
		Scopes:                selectMCPOAuthScopes(resourceScopes, asMeta.ScopesSupported),
	}, nil
}

// selectMCPOAuthScopes asks only for scopes the protected resource itself
// named. When that resource is OpenID-based and the authorization server
// supports offline_access, include it so an hour-long access token is paired
// with a refresh token instead of becoming an unrecoverable stale sign-in.
//
// Deliberately do not copy every scope from the authorization server: fields
// such as email/profile are capabilities it can offer, not permissions this
// MCP resource said it needs.
func selectMCPOAuthScopes(resourceScopes, authorizationScopes []string) []string {
	if len(resourceScopes) == 0 {
		return nil
	}
	supported := make(map[string]bool, len(authorizationScopes))
	for _, scope := range authorizationScopes {
		supported[scope] = true
	}
	selected := make([]string, 0, len(resourceScopes)+1)
	seen := map[string]bool{}
	for _, scope := range resourceScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" || seen[scope] || (len(supported) > 0 && !supported[scope]) {
			continue
		}
		selected = append(selected, scope)
		seen[scope] = true
	}
	if seen["openid"] && supported["offline_access"] {
		selected = append(selected, "offline_access")
	}
	return selected
}

// chooseMCPClientAuth prefers a public client when the authorization server
// supports one, then the form-body secret Supabase publishes, then HTTP Basic.
// An omitted list predates this metadata field on several working servers and
// retains the old public-client behaviour.
func chooseMCPClientAuth(methods []string) (string, error) {
	if len(methods) == 0 {
		return "none", nil
	}
	for _, wanted := range []string{"none", "client_secret_post", "client_secret_basic"} {
		for _, offered := range methods {
			if offered == wanted {
				return wanted, nil
			}
		}
	}
	return "", fmt.Errorf("no supported token endpoint authentication method in %v", methods)
}

// resourceMetadataURL gets the RFC 9728 resource-metadata URL a server names
// on an unauthenticated request's 401. A GET is enough — every server this
// was verified against challenges on any method, MCP's own POST included.
// resourceMetadataURL asks the server for its challenge and returns the
// resource_metadata pointer in it, or "" when the server challenged without
// one — which is a real shape (Atlassian, mcp.atlassian.com/v1/mcp, probed
// 2026-09-13) and not a failure: authorizationServerFor knows where to look
// next. Only a server that does not challenge at all is nothing to discover.
func resourceMetadataURL(ctx context.Context, resourceURL string) (string, error) {
	challenge := func(method string, body io.Reader) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, resourceURL, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", clientUserAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
		}
		return httpClient.Do(req)
	}
	resp, err := challenge(http.MethodPost, strings.NewReader(initializeProbe))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		resp.Body.Close()
		if resp, err = challenge(http.MethodGet, nil); err != nil {
			return "", err
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("%s did not challenge for authorization (got %s) — nothing to discover", resourceURL, resp.Status)
	}
	m := resourceMetadataRef.FindStringSubmatch(resp.Header.Get("WWW-Authenticate"))
	if m == nil {
		return "", nil
	}
	return m[1], nil
}

// authorizationServerFor names the authorization server for a resource:
// the one its protected-resource document names, read from the pointer in
// the challenge when there was one and from the RFC 9728 well-known
// locations under the resource when there was not; and, when no such
// document exists at all, the resource's own origin — the shape MCP's
// 2025-03-26 authorization spec described, which Atlassian still serves
// (no protected-resource document anywhere, but a full RFC 8414 document at
// its origin).
func authorizationServerFor(ctx context.Context, resourceURL, metaURL string) (string, []string, error) {
	candidates := []string{}
	if metaURL != "" {
		candidates = append(candidates, metaURL)
	} else {
		candidates = append(candidates, protectedResourceCandidates(resourceURL)...)
	}
	var lastErr error
	for _, c := range candidates {
		var resource protectedResourceMetadata
		if err := getJSON(ctx, c, &resource); err != nil {
			lastErr = fmt.Errorf("reading %s: %w", c, err)
			continue
		}
		if len(resource.AuthorizationServers) == 0 {
			return "", nil, fmt.Errorf("%s named no authorization server", c)
		}
		return resource.AuthorizationServers[0], resource.ScopesSupported, nil
	}
	if metaURL != "" {
		return "", nil, lastErr
	}
	u, err := url.Parse(resourceURL)
	if err != nil || u.Host == "" {
		return "", nil, fmt.Errorf("%s is not a URL an authorization server can be derived from", resourceURL)
	}
	return u.Scheme + "://" + u.Host, nil, nil
}

// protectedResourceCandidates lists where RFC 9728 §3.1 says a resource's
// metadata may live when the challenge did not say: the well-known segment
// with the resource's path inserted after it, then at the origin.
func protectedResourceCandidates(resourceURL string) []string {
	u, err := url.Parse(resourceURL)
	if err != nil || u.Host == "" {
		return nil
	}
	origin := u.Scheme + "://" + u.Host
	path := strings.TrimSuffix(u.Path, "/")
	var out []string
	if path != "" {
		out = append(out, origin+"/.well-known/oauth-protected-resource"+path)
	}
	return append(out, origin+"/.well-known/oauth-protected-resource")
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

// mcpClientRegistration is what a dynamic registration minted. Aetox prefers
// a public PKCE client when the metadata offers one. Supabase and Figma do not,
// and return a client_secret that their token endpoints require. Keeping the
// optional secret leaves the public-client path unchanged for the other servers.
type mcpClientRegistration struct {
	ClientID                string `json:"client_id"`
	ClientSecret            string `json:"client_secret"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method"`
}

// registerMCPClient performs RFC 7591 dynamic client registration and returns
// the client credentials the authorization server minted. authMethod was
// selected from the server's own metadata; a returned secret travels through
// exchange and refresh because the selected method requires it.
func registerMCPClient(ctx context.Context, registrationEndpoint, redirectURI, authMethod string) (mcpClientRegistration, error) {
	body, err := json.Marshal(struct {
		RedirectURIs            []string `json:"redirect_uris"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
		ClientName              string   `json:"client_name"`
	}{
		RedirectURIs:            []string{redirectURI},
		TokenEndpointAuthMethod: authMethod,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		ClientName:              "Aetox",
	})
	if err != nil {
		return mcpClientRegistration{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registrationEndpoint, bytes.NewReader(body))
	if err != nil {
		return mcpClientRegistration{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", clientUserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return mcpClientRegistration{}, err
	}
	var out mcpClientRegistration
	if err := readJSON(resp, &out); err != nil {
		return mcpClientRegistration{}, fmt.Errorf("registering with %s: %w", registrationEndpoint, err)
	}
	if out.ClientID == "" {
		return mcpClientRegistration{}, fmt.Errorf("registering with %s: no client_id in response", registrationEndpoint)
	}
	if out.TokenEndpointAuthMethod == "" {
		out.TokenEndpointAuthMethod = authMethod
	}
	if out.TokenEndpointAuthMethod != "none" && out.ClientSecret == "" {
		return mcpClientRegistration{}, fmt.Errorf("registering with %s: %s selected but no client_secret in response", registrationEndpoint, out.TokenEndpointAuthMethod)
	}
	return out, nil
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
	registration, err := registerMCPClient(ctx, meta.RegistrationEndpoint, lb.RedirectURI, meta.ClientAuthMethod)
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
	q.Set("client_id", registration.ClientID)
	q.Set("redirect_uri", lb.RedirectURI)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	if len(meta.Scopes) > 0 {
		q.Set("scope", strings.Join(meta.Scopes, " "))
	}

	return &Pending{
		URL:              meta.AuthorizationEndpoint + "?" + q.Encode(),
		Verifier:         verifier,
		State:            state,
		provider:         serverName,
		lb:               lb,
		tokenEndpoint:    meta.TokenEndpoint,
		clientID:         registration.ClientID,
		clientSecret:     registration.ClientSecret,
		clientAuthMethod: registration.TokenEndpointAuthMethod,
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
	form.Set("code_verifier", pending.Verifier)

	tokens, err := mcpOAuthToken(ctx, pending.tokenEndpoint, form, pending.clientID, pending.clientSecret, pending.clientAuthMethod)
	if err != nil {
		return err
	}
	return Set(pending.provider, Credential{
		Type:             "oauth",
		Access:           tokens.AccessToken,
		Refresh:          tokens.RefreshToken,
		ExpiresAt:        tokens.expiresAt(),
		TokenEndpoint:    pending.tokenEndpoint,
		ClientID:         pending.clientID,
		ClientSecret:     pending.clientSecret,
		ClientAuthMethod: pending.clientAuthMethod,
		Label:            pending.provider,
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

	tokens, err := mcpOAuthToken(ctx, cred.TokenEndpoint, form, cred.ClientID, cred.ClientSecret, cred.ClientAuthMethod)
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

func mcpOAuthToken(ctx context.Context, tokenEndpoint string, form url.Values, clientID, clientSecret, authMethod string) (tokenResponse, error) {
	// Credentials written before client_auth_method existed can still refresh.
	// A stored secret implies the form-body method; no secret is the public path.
	if authMethod == "" {
		if clientSecret != "" {
			authMethod = "client_secret_post"
		} else {
			authMethod = "none"
		}
	}
	switch authMethod {
	case "none":
		form.Set("client_id", clientID)
	case "client_secret_post":
		form.Set("client_id", clientID)
		form.Set("client_secret", clientSecret)
	case "client_secret_basic":
		// Applied to the request below; credentials do not belong in the body.
	default:
		return tokenResponse{}, fmt.Errorf("unsupported token endpoint authentication method %q", authMethod)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", clientUserAgent)
	if authMethod == "client_secret_basic" {
		req.SetBasicAuth(clientID, clientSecret)
	}

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
