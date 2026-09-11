package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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

	a := newTestApp()
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

	a := newTestApp()
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
