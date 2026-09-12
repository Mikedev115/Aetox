package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/credentials"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

// The screen signs; the engine's client does not (§248 A3). A provider built
// through bootstrap's door with the screen's transport carries the key from
// credentials.json on the wire and none in its own configuration.
func TestScreenTransportSignsFromTheCredentialStore(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	if err := credentials.Set("groq", "gsk-screen"); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	a := &App{}
	// No APIKey: the engine side never holds one, and the transport is what
	// satisfies a provider that requires a key.
	p, err := model.NewProvider(model.ProviderOptions{
		Provider: "groq", Model: "m", BaseURL: server.URL,
		Transport: a.providerTransport("groq", ""),
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if _, err := p.Complete(context.Background(), model.Request{Messages: []model.Message{{Role: model.RoleUser, Content: "hi"}}}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if gotAuth != "Bearer gsk-screen" {
		t.Fatalf("provider saw Authorization %q; want the key from credentials.json", gotAuth)
	}
}

// A sign-in outranks a pasted key for the same provider, and its extra
// headers ride along — the rule the factory used to apply, now applied where
// the credential lives.
func TestScreenTransportPrefersTheSignIn(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	if err := credentials.Set("codex", "sk-ignored"); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	if err := oauth.Set("codex", oauth.Credential{Type: "oauth", Access: "tok", Account: "acct_9"}); err != nil {
		t.Fatalf("seed sign-in: %v", err)
	}
	var got http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{}}\n\n"))
	}))
	defer server.Close()

	a := &App{}
	p, err := model.NewProvider(model.ProviderOptions{
		Provider: "codex", Model: "m", BaseURL: server.URL,
		Transport: a.providerTransport("codex", ""),
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if _, err := p.Complete(context.Background(), model.Request{Messages: []model.Message{{Role: model.RoleUser, Content: "hi"}}}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got.Get("Authorization") != "Bearer tok" {
		t.Fatalf("Authorization %q; want the sign-in's token over the pasted key", got.Get("Authorization"))
	}
	if got.Get("chatgpt-account-id") != "acct_9" {
		t.Fatalf("chatgpt-account-id %q; want the account header the sign-in requires", got.Get("chatgpt-account-id"))
	}
}

// The store said the token had a week left; the backend said token_expired
// (2026-09-12). On the screen's transport the wire client holds no token to
// renew, so the signer is what renews once and re-sends the same body. The
// generic refresher stands in for ChatGPT's here because it reads its token
// endpoint from the credential rather than a compiled-in constant.
func TestScreenTransportRenewsOnceOn401(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())

	var seen []string
	var bodies []int64
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"fresh","refresh_token":"r2","expires_in":3600}`))
	})
	mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		bodies = append(bodies, r.ContentLength)
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"token_expired"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// A week of claimed life, so Token() alone would never refresh it.
	err := oauth.Set("openai-compatible", oauth.Credential{
		Type: "oauth", Access: "stale", Refresh: "r1", ClientID: "c",
		TokenEndpoint: server.URL + "/token",
		ExpiresAt:     time.Now().Add(7 * 24 * time.Hour).UnixMilli(),
	})
	if err != nil {
		t.Fatalf("seed sign-in: %v", err)
	}

	a := &App{}
	p, err := model.NewProvider(model.ProviderOptions{
		Provider: "openai-compatible", Model: "m", BaseURL: server.URL,
		Transport: a.providerTransport("openai-compatible", ""),
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	resp, err := p.Complete(context.Background(), model.Request{Messages: []model.Message{{Role: model.RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "ok" {
		t.Fatalf("text = %q", resp.Text)
	}
	if strings.Join(seen, ",") != "Bearer stale,Bearer fresh" {
		t.Fatalf("requests carried %v; want the stale token once, then the renewed one", seen)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] || bodies[1] <= 0 {
		t.Fatalf("bodies %v; the re-send must carry the same request", bodies)
	}
	if cred, _ := oauth.Get("openai-compatible"); cred.Access != "fresh" || cred.Refresh != "r2" {
		t.Fatalf("store holds access=%q refresh=%q; the renewal must be written back", cred.Access, cred.Refresh)
	}
}
