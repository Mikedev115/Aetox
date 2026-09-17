package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

// fakeMCPServer stands in for a real MCP server plus its authorization
// server, both served from the same httptest.Server so a test needs exactly
// one address. resourceMetadataPath/asMetadataPath let a test place the
// well-known documents wherever it wants — TestWellKnownCandidatesCoverRealShapes
// exercises the shapes actually seen in the wild.
type fakeMCPServer struct {
	srv                  *httptest.Server
	registrationEndpoint string // empty disables DCR — the elevenlabs/vercel/shopify shape
	registerCalls        int
	tokenCalls           int
	refreshCalls         int
	clientSecret         string
	resourceScopes       []string
	authorizationScopes  []string
	// issuedRefresh, when set, is returned as refresh_token from the
	// authorization_code exchange, so TestFinishAndThenRefresh can drive a
	// refresh afterward without a second server.
	nextAccessToken string
}

func newFakeMCPServer(t *testing.T, withDCR bool) *fakeMCPServer {
	t.Helper()
	f := &fakeMCPServer{nextAccessToken: "at_1"}
	mux := http.NewServeMux()

	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource"`, f.srv.URL))
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		meta := map[string]any{
			"authorization_servers": []string{f.srv.URL},
		}
		if len(f.resourceScopes) > 0 {
			meta["scopes_supported"] = f.resourceScopes
		}
		_ = json.NewEncoder(w).Encode(meta)
	})
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		meta := map[string]any{
			"authorization_endpoint": f.srv.URL + "/authorize",
			"token_endpoint":         f.srv.URL + "/token",
		}
		if f.clientSecret != "" {
			meta["token_endpoint_auth_methods_supported"] = []string{"client_secret_post"}
		} else {
			meta["token_endpoint_auth_methods_supported"] = []string{"none"}
		}
		if withDCR {
			meta["registration_endpoint"] = f.srv.URL + "/register"
		}
		if len(f.authorizationScopes) > 0 {
			meta["scopes_supported"] = f.authorizationScopes
		}
		_ = json.NewEncoder(w).Encode(meta)
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		f.registerCalls++
		var in struct {
			TokenEndpointAuthMethod string `json:"token_endpoint_auth_method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wantMethod := "none"
		if f.clientSecret != "" {
			wantMethod = "client_secret_post"
		}
		if in.TokenEndpointAuthMethod != wantMethod {
			http.Error(w, "wrong token endpoint auth method", http.StatusBadRequest)
			return
		}
		out := map[string]any{"client_id": "client_abc", "token_endpoint_auth_method": wantMethod}
		if f.clientSecret != "" {
			out["client_secret"] = f.clientSecret
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("token endpoint: parse form: %v", err)
		}
		if f.clientSecret != "" && r.Form.Get("client_secret") != f.clientSecret {
			http.Error(w, `{"message":"Required parameter: client_secret"}`, http.StatusUnprocessableEntity)
			return
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			f.tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": f.nextAccessToken, "refresh_token": "rt_1", "expires_in": 3600,
			})
		case "refresh_token":
			f.refreshCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "at_refreshed", "expires_in": 3600,
			})
		default:
			http.Error(w, "unknown grant_type", http.StatusBadRequest)
		}
	})

	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	if withDCR {
		f.registrationEndpoint = f.srv.URL + "/register"
	}
	return f
}

func TestDiscoverMCPOAuthFullChain(t *testing.T) {
	f := newFakeMCPServer(t, true)
	meta, err := discoverMCPOAuth(context.Background(), f.srv.URL+"/mcp")
	if err != nil {
		t.Fatalf("discoverMCPOAuth: %v", err)
	}
	if meta.AuthorizationEndpoint != f.srv.URL+"/authorize" {
		t.Errorf("AuthorizationEndpoint = %q", meta.AuthorizationEndpoint)
	}
	if meta.TokenEndpoint != f.srv.URL+"/token" {
		t.Errorf("TokenEndpoint = %q", meta.TokenEndpoint)
	}
	if meta.RegistrationEndpoint != f.srv.URL+"/register" {
		t.Errorf("RegistrationEndpoint = %q", meta.RegistrationEndpoint)
	}
}

// Three shapes found by probing the first-party servers on 2026-09-13,
// each of which the discovery of 2026-09-03 turned away:
//   - Figma answers GET /mcp with 405 and challenges only a POST;
//   - Stripe writes resource_metadata=… without quotes;
//   - Atlassian challenges with no resource_metadata at all and publishes no
//     protected-resource document, only RFC 8414 metadata at its origin.
func TestDiscoverMCPOAuthRealShapes(t *testing.T) {
	serve := func(challenge func(w http.ResponseWriter, r *http.Request), withPRM bool) *httptest.Server {
		mux := http.NewServeMux()
		var srv *httptest.Server
		mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) { challenge(w, r) })
		if withPRM {
			mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"authorization_servers": []string{srv.URL}})
			})
		}
		mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"authorization_endpoint": srv.URL + "/authorize",
				"token_endpoint":         srv.URL + "/token",
				"registration_endpoint":  srv.URL + "/register",
			})
		})
		srv = httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		return srv
	}
	cases := map[string]struct {
		challenge func(srvURL string) func(w http.ResponseWriter, r *http.Request)
		withPRM   bool
	}{
		"figma: POST-only, GET is 405": {func(srvURL string) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource",scope="mcp:connect"`, srvURL))
				w.WriteHeader(http.StatusUnauthorized)
			}
		}, true},
		"stripe: unquoted resource_metadata": {func(srvURL string) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("WWW-Authenticate", "Bearer resource_metadata="+srvURL+"/.well-known/oauth-protected-resource")
				w.WriteHeader(http.StatusUnauthorized)
			}
		}, true},
		"atlassian: no pointer, no document, AS at origin": {func(srvURL string) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token"`)
				w.WriteHeader(http.StatusUnauthorized)
			}
		}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var srv *httptest.Server
			srv = serve(func(w http.ResponseWriter, r *http.Request) { tc.challenge(srv.URL)(w, r) }, tc.withPRM)
			meta, err := discoverMCPOAuth(context.Background(), srv.URL+"/mcp")
			if err != nil {
				t.Fatalf("discoverMCPOAuth: %v", err)
			}
			if meta.RegistrationEndpoint != srv.URL+"/register" {
				t.Errorf("RegistrationEndpoint = %q", meta.RegistrationEndpoint)
			}
		})
	}
}

// A server that does not challenge at all is still nothing to discover.
func TestDiscoverMCPOAuthNeedsAChallenge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(srv.Close)
	if _, err := discoverMCPOAuth(context.Background(), srv.URL+"/mcp"); err == nil || !strings.Contains(err.Error(), "did not challenge") {
		t.Fatalf("err = %v; want the no-challenge sentence", err)
	}
}

// The elevenlabs/vercel/shopify shape, found by actually probing them on
// 2026-09-03: a real authorization server, real endpoints, no
// registration_endpoint. StartMCPOAuth must say so rather than fail on some
// unrelated step (a nil client id reaching the authorize URL, say).
func TestStartMCPOAuthRefusesWithoutDynamicRegistration(t *testing.T) {
	isolateStore(t)
	f := newFakeMCPServer(t, false)
	_, err := StartMCPOAuth(context.Background(), "no-dcr-server", f.srv.URL+"/mcp")
	if err == nil {
		t.Fatal("expected an error — this server has no registration_endpoint")
	}
	if got := err.Error(); !strings.Contains(got, "does not support automatic sign-in") {
		t.Errorf("error = %q; want it to name the actual reason", got)
	}
}

func TestStartMCPOAuthRequestsRefreshWithoutExtraIdentityScopes(t *testing.T) {
	isolateStore(t)
	f := newFakeMCPServer(t, true)
	// This is Vercel's published shape: the protected resource needs openid,
	// while the authorization server additionally offers offline_access (and
	// broader identity scopes the MCP resource did not ask for).
	f.resourceScopes = []string{"openid"}
	f.authorizationScopes = []string{"openid", "email", "profile", "offline_access"}

	pending, err := StartMCPOAuth(context.Background(), "vercel", f.srv.URL+"/mcp")
	if err != nil {
		t.Fatalf("StartMCPOAuth: %v", err)
	}
	t.Cleanup(pending.Cancel)

	parsed, err := url.Parse(pending.URL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	got := strings.Fields(parsed.Query().Get("scope"))
	want := []string{"openid", "offline_access"}
	if !slices.Equal(got, want) {
		t.Fatalf("scope = %v; want %v (without email/profile)", got, want)
	}
}

func TestStartThenFinishMCPOAuthStoresCredential(t *testing.T) {
	isolateStore(t)
	f := newFakeMCPServer(t, true)
	// Supabase's DCR returns a secret even though the desktop asked for a
	// public client, then requires that secret at the token endpoint.
	f.clientSecret = "secret_abc"

	pending, err := StartMCPOAuth(context.Background(), "test-server", f.srv.URL+"/mcp")
	if err != nil {
		t.Fatalf("StartMCPOAuth: %v", err)
	}
	if f.registerCalls != 1 {
		t.Fatalf("registerCalls = %d; want 1", f.registerCalls)
	}

	// Stand in for the browser landing back on the loopback, exactly as
	// loopback_test.go does for the other flows — the authorize page itself
	// (pending.URL) is never actually rendered here; the fake server's job
	// ends at handing out a client_id, and the redirect below is what the
	// user's own "allow" click on the real page would produce.
	redirect := pending.lb.RedirectURI + "?code=auth_code_1&state=" + pending.State
	resp, err := http.Get(redirect)
	if err != nil {
		t.Fatalf("simulating the redirect: %v", err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := FinishMCPOAuth(ctx, pending); err != nil {
		t.Fatalf("FinishMCPOAuth: %v", err)
	}
	if f.tokenCalls != 1 {
		t.Fatalf("tokenCalls = %d; want 1", f.tokenCalls)
	}

	cred, ok := Get("test-server")
	if !ok {
		t.Fatal("credential was not stored under the server name")
	}
	if cred.Type != "oauth" || cred.Access != "at_1" || cred.Refresh != "rt_1" {
		t.Fatalf("stored credential = %+v; want the tokens the fake server issued", cred)
	}
	if cred.TokenEndpoint != f.srv.URL+"/token" || cred.ClientID != "client_abc" || cred.ClientSecret != "secret_abc" || cred.ClientAuthMethod != "client_secret_post" {
		t.Fatalf("stored credential is missing what a later refresh needs: %+v", cred)
	}
	if cred.ExpiresAt == 0 {
		t.Fatal("expires_in from the token response did not become an ExpiresAt")
	}
}

func TestFinishMCPOAuthRejectsMismatchedState(t *testing.T) {
	isolateStore(t)
	f := newFakeMCPServer(t, true)
	pending, err := StartMCPOAuth(context.Background(), "test-server", f.srv.URL+"/mcp")
	if err != nil {
		t.Fatalf("StartMCPOAuth: %v", err)
	}

	resp, err := http.Get(pending.lb.RedirectURI + "?code=auth_code_1&state=not-the-real-state")
	if err != nil {
		t.Fatalf("simulating the redirect: %v", err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := FinishMCPOAuth(ctx, pending); err == nil {
		t.Fatal("a mismatched state was accepted")
	}
	if _, ok := Get("test-server"); ok {
		t.Fatal("a credential was stored despite the state mismatch")
	}
}

// The generic path token.go.Token falls back to when a credential has no
// compiled-in refresher — see the TokenEndpoint branch added to Token().
func TestRefreshMCPOAuthRenewsAndPersists(t *testing.T) {
	isolateStore(t)
	f := newFakeMCPServer(t, true)
	f.clientSecret = "secret_abc"

	stale := Credential{
		Type: "oauth", Access: "at_stale", Refresh: "rt_1",
		ExpiresAt:        time.Now().Add(-time.Minute).UnixMilli(),
		TokenEndpoint:    f.srv.URL + "/token",
		ClientID:         "client_abc",
		ClientSecret:     "secret_abc",
		ClientAuthMethod: "client_secret_post",
	}
	if err := Set("test-server", stale); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := Token(context.Background(), "test-server")
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if got != "at_refreshed" {
		t.Fatalf("Token = %q; want the refreshed access token", got)
	}
	if f.refreshCalls != 1 {
		t.Fatalf("refreshCalls = %d; want 1", f.refreshCalls)
	}
	cred, _ := Get("test-server")
	// The fake token endpoint's refresh branch does not send back a new
	// refresh_token — the old one must survive rather than being wiped.
	if cred.Refresh != "rt_1" {
		t.Fatalf("refresh token was lost across a refresh that did not rotate it: %+v", cred)
	}
}

func TestWellKnownCandidatesCoverRealShapes(t *testing.T) {
	// grafana's shape, verified live 2026-09-03: the authorization server IS
	// the resource URL (path and all), and only the path-insertion form
	// answers.
	got := wellKnownCandidates("https://mcp.grafana.com/mcp")
	want := "https://mcp.grafana.com/.well-known/oauth-authorization-server/mcp"
	if !slices.Contains(got, want) {
		t.Errorf("candidates = %v; want %q among them", got, want)
	}

	// semgrep's shape: the authorization server is its own root domain, no
	// path at all — the root form is the only one that makes sense.
	got = wellKnownCandidates("https://login.semgrep.dev/")
	want = "https://login.semgrep.dev/.well-known/oauth-authorization-server"
	if !slices.Contains(got, want) {
		t.Errorf("candidates = %v; want %q among them", got, want)
	}
}
