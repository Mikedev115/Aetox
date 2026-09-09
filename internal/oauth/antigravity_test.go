package oauth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestAntigravityMethodIsRestricted(t *testing.T) {
	method, ok := MethodFor("antigravity")
	if !ok {
		t.Fatal("MethodFor(antigravity) returned false")
	}
	if method.Risk != RiskRestricted {
		t.Fatalf("method.Risk = %q; want %q", method.Risk, RiskRestricted)
	}
	if method.Kind != "browser" {
		t.Fatalf("method.Kind = %q; want browser", method.Kind)
	}
	if !strings.Contains(method.Note, "Antigravity") {
		t.Fatalf("method.Note = %q; want it to mention Antigravity", method.Note)
	}
}

func TestStartAntigravityBuildsValidOAuthRequest(t *testing.T) {
	pending, err := StartAntigravity()
	if err != nil {
		t.Fatalf("StartAntigravity: %v", err)
	}
	defer pending.Cancel()

	if pending.provider != "antigravity" {
		t.Fatalf("pending.provider = %q; want antigravity", pending.provider)
	}
	if pending.Verifier == "" {
		t.Fatal("pending.Verifier is empty; want PKCE verifier")
	}
	if pending.State == "" {
		t.Fatal("pending.State is empty")
	}

	u, err := url.Parse(pending.URL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if u.Host != "accounts.google.com" {
		t.Fatalf("u.Host = %q; want accounts.google.com", u.Host)
	}
	q := u.Query()
	if q.Get("client_id") != antigravityClientID {
		t.Fatalf("client_id = %q; want %q", q.Get("client_id"), antigravityClientID)
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q; want S256", q.Get("code_challenge_method"))
	}
	if q.Get("code_challenge") == "" {
		t.Fatal("code_challenge is empty")
	}
	if q.Get("state") != pending.State {
		t.Fatalf("state mismatch: URL has %q, pending has %q", q.Get("state"), pending.State)
	}
	if !strings.Contains(q.Get("redirect_uri"), "/oauth-callback") {
		t.Fatalf("redirect_uri = %q; want /oauth-callback path", q.Get("redirect_uri"))
	}
}

func TestRefreshAntigravity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			http.Error(w, "bad grant_type", 400)
			return
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			http.Error(w, "bad refresh_token", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "new-access",
			"refresh_token": "new-refresh",
			"expires_in": 3600,
			"token_type": "Bearer"
		}`))
	}))
	defer server.Close()

	// Redirect antigravityToken calls for this test by replacing HTTP client or calling test server
	// Test refresh with missing refresh token
	_, err := refreshAntigravity(context.Background(), Credential{Type: "oauth", Access: "a"})
	if err == nil {
		t.Fatal("refreshAntigravity succeeded with empty refresh token")
	}
}

func TestDailyCloudCode(t *testing.T) {
	if os.Getenv("AETOX_LIVE_TESTS") == "" {
		t.Skip("skipping live Antigravity test; set AETOX_LIVE_TESTS=1 to run")
	}
	cred, ok := Get("antigravity")
	if !ok || cred.Access == "" {
		t.Skip("no stored antigravity credential")
	}

	// Test calling loadCodeAssist on daily-cloudcode-pa.googleapis.com
	endpoint := "https://daily-cloudcode-pa.googleapis.com/v1internal"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		endpoint+":loadCodeAssist", strings.NewReader(`{"metadata":{"ideType":"ANTIGRAVITY","platform":"PLATFORM_UNSPECIFIED"}}`))
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cred.Access)
	req.Header.Set("User-Agent", "antigravity/2.12.2")

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("daily-cloudcode HTTP error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	t.Logf("daily-cloudcode Status: %d %s", resp.StatusCode, resp.Status)
	t.Logf("daily-cloudcode Body: %s", string(body))

	// Discover available models
	modelsReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		endpoint+":fetchAvailableModels", strings.NewReader(`{}`))
	modelsReq.Header.Set("Content-Type", "application/json")
	modelsReq.Header.Set("Authorization", "Bearer "+cred.Access)
	modelsReq.Header.Set("User-Agent", "antigravity/2.12.2")
	if modelsResp, err := httpClient.Do(modelsReq); err == nil {
		mBytes, _ := io.ReadAll(modelsResp.Body)
		modelsResp.Body.Close()
		t.Logf("fetchAvailableModels: %s", string(mBytes))
	}

	// Now test streamGenerateContent
	genURL := "https://daily-cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse"
	genBody := `{
		"model": "gemini-3.8-flash-high",
		"project": "aicode-consumers",
		"request": {
			"contents": [
				{
					"role": "user",
					"parts": [{"text": "Say hello in Thai in one sentence"}]
				}
			]
		}
	}`
	genReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, genURL, strings.NewReader(genBody))
	if err != nil {
		t.Fatalf("genReq: %v", err)
	}
	genReq.Header.Set("Content-Type", "application/json")
	genReq.Header.Set("Accept", "text/event-stream")
	genReq.Header.Set("Authorization", "Bearer "+cred.Access)
	genReq.Header.Set("User-Agent", "antigravity/2.12.2")

	genResp, err := httpClient.Do(genReq)
	if err != nil {
		t.Fatalf("streamGenerateContent error: %v", err)
	}
	defer genResp.Body.Close()
	genBytes, _ := io.ReadAll(genResp.Body)
	t.Logf("streamGenerateContent Status: %d %s", genResp.StatusCode, genResp.Status)
	t.Logf("streamGenerateContent Body: %s", string(genBytes))
}



