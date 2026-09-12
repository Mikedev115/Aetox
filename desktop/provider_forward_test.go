package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Mikedev115/Aetox/internal/credentials"
	"github.com/Mikedev115/Aetox/internal/engine/remote"
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

	a := newTestApp(t)
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

	a := newTestApp(t)
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

// With the engine on a host (§248 phase 3) the request the screen is asked
// to sign was built by another machine's process. A credential rides only
// to where the screen itself knows the provider lives — the catalog's
// endpoint, the sign-in's, a custom row's as recorded here, loopback — and
// a request anywhere else goes out bare. On this machine nothing changes.
func TestOnAHostACredentialRidesOnlyToWhereTheScreenKnowsTheProviderLives(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	if err := credentials.Set("groq", "gsk-screen"); err != nil {
		t.Fatal(err)
	}
	if err := oauth.Set("codex", oauth.Credential{Type: "oauth", Access: "tok", Endpoint: "https://chatgpt.example/backend"}); err != nil {
		t.Fatal(err)
	}
	if err := rememberCustomEndpoint("my-gateway", "https://gw.example.com/v1"); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t)
	a.engine = newLocalEngine(a, a.client)
	at := func(raw string) *url.URL { u, _ := url.Parse(raw); return u }

	// At home: anywhere.
	if !a.credentialMayRide("groq", at("https://evil.example/v1/chat")) {
		t.Error("on this machine the signer refused a URL; the engine here is ours")
	}

	a.engine.retarget(engineTarget{mode: modeRemote, host: remoteHostFor("box")})
	<-a.engine.kick // the test has no supervisor running; take the kick back
	cases := []struct {
		provider, url string
		want          bool
	}{
		{"groq", "https://api.groq.com/openai/v1/chat/completions", true},
		{"groq", "https://evil.example/v1/chat/completions", false},
		{"groq", "http://127.0.0.1:1234/v1/chat/completions", true},
		{"groq", "http://localhost:11434/v1/chat/completions", true},
		{"codex", "https://chatgpt.example/backend/responses", true},
		{"codex", "https://chatgpt.example.attacker.net/backend/responses", false},
		{"my-gateway", "https://gw.example.com/v1/chat/completions", true},
		{"my-gateway", "https://gw.example.com.evil/v1/chat/completions", false},
		{"nobody-knows", "https://somewhere.example/v1", false},
	}
	for _, c := range cases {
		if got := a.credentialMayRide(c.provider, at(c.url)); got != c.want {
			t.Errorf("%s → %s: mayRide = %v; want %v", c.provider, c.url, got, c.want)
		}
	}

	// And on the wire: the refused request reaches the server with no key.
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()
	// httptest listens on 127.0.0.1, which is loopback and allowed; so the
	// URL is disguised as a name the screen does not know, resolved by the
	// transport's own dialer back to the server.
	network := &http.Transport{DialContext: func(ctx context.Context, netw, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, netw, server.Listener.Addr().String())
	}}
	rt := a.providerTransport("groq", "")(network)
	req, _ := http.NewRequest(http.MethodPost, "http://groq-lookalike.example/openai/v1/chat/completions", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if gotAuth != "" {
		t.Errorf("the key rode to a host the screen does not know: Authorization %q", gotAuth)
	}
	req2, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:1/openai/v1/chat/completions", nil)
	resp2, err := rt.RoundTrip(req2)
	if err != nil {
		t.Fatalf("RoundTrip loopback: %v", err)
	}
	resp2.Body.Close()
	if gotAuth != "Bearer gsk-screen" {
		t.Errorf("a loopback runtime did not get the key: Authorization %q", gotAuth)
	}
}

func remoteHostFor(name string) remote.Host { return remote.Host{Name: name, Target: "user@" + name} }
