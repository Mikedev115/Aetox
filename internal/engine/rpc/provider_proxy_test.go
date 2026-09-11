package rpc

// The provider request across the wire (§248 rule 3): the key is at the
// provider and nowhere in what the engine sent; the body comes back
// byte-exact and streamed; a cancel on the engine's side reaches the
// provider; a retry above the proxy reopens through it; no screen fails the
// request after the wait, not never.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// signWithKey is the screen's signer for these tests: it asserts the engine
// sent no credential, then puts one on.
func signWithKey(t *testing.T, key string) Signer {
	return func(provider, wireFormat string) model.Transport {
		return func(network http.RoundTripper) http.RoundTripper {
			return roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if got := req.Header.Get("x-api-key"); got != "" {
					t.Errorf("the engine's request carried a key %q — it must hold none", got)
				}
				req = req.Clone(req.Context())
				req.Header.Set("x-api-key", key)
				return network.RoundTrip(req)
			})
		}
	}
}

// proxied is an engine behind the wire whose screen signs with key, and the
// transport the engine's model client would be built on.
func proxied(t *testing.T, key string) (*Server, http.RoundTripper) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	srv := NewServer(testToken, engine.NewEngine)
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	c := NewClient(ClientOptions{})
	ServeProvider(c, signWithKey(t, key))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Connect(ctx, "tcp", strings.TrimPrefix(hs.URL, "http://"), testToken); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	network := http.DefaultTransport.(*http.Transport).Clone()
	network.ResponseHeaderTimeout = 5 * time.Second
	return srv, srv.Screen().ProviderTransport("anthropic", "anthropic")(network)
}

func TestAProviderRequestIsSignedOnTheScreenAndStreamedBack(t *testing.T) {
	var sawKey atomic.Value
	var sawBody atomic.Value
	pieces := []string{"data: {\"text\":\"สวัส\"}\n\n", "data: {\"text\":\"ดี\"}\n\n", "data: [DONE]\n\n"}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawKey.Store(r.Header.Get("x-api-key"))
		body, _ := io.ReadAll(r.Body)
		sawBody.Store(string(body))
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("anthropic-ratelimit-requests-remaining", "41")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)
		for _, p := range pieces {
			_, _ = io.WriteString(w, p)
			flusher.Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	t.Cleanup(provider.Close)

	_, transport := proxied(t, "sk-secret")
	req, _ := http.NewRequest("POST", provider.URL+"/v1/messages", strings.NewReader(`{"model":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("anthropic-ratelimit-requests-remaining"); got != "41" {
		t.Errorf("the quota header did not cross: %q", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading the body: %v", err)
	}
	if string(body) != strings.Join(pieces, "") {
		t.Errorf("body = %q, want the provider's stream byte for byte", body)
	}
	if sawKey.Load() != "sk-secret" {
		t.Errorf("the provider saw key %v, want the screen's", sawKey.Load())
	}
	if sawBody.Load() != `{"model":"x"}` {
		t.Errorf("the provider got body %v", sawBody.Load())
	}
}

// The engine cancelling the request — the Stop button — reaches the provider
// as a closed connection, not as a stream that runs to its end for nobody.
func TestCancellingTheEngineRequestStopsTheStream(t *testing.T) {
	gone := make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		flusher := w.(http.Flusher)
		for i := 0; ; i++ {
			if _, err := io.WriteString(w, "data: tick\n\n"); err != nil {
				close(gone)
				return
			}
			flusher.Flush()
			select {
			case <-r.Context().Done():
				close(gone)
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	}))
	t.Cleanup(provider.Close)

	_, transport := proxied(t, "k")
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "POST", provider.URL, strings.NewReader("{}"))
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	buf := make([]byte, 64)
	if _, err := resp.Body.Read(buf); err != nil {
		t.Fatalf("first read: %v", err)
	}
	cancel()
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("the provider never noticed the engine gave up")
	}
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Error("the body read to a clean end after the request was cancelled")
	}
}

// The retry layer above the proxy sees the 429 and reopens; the second open
// is signed again and the body is replayed. This runs the real model client
// over the proxy, the way the engine's does.
func TestARetryReopensThroughTheProxy(t *testing.T) {
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-retry" {
			t.Errorf("provider saw key %q", r.Header.Get("x-api-key"))
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "hi") {
			t.Errorf("attempt %d body = %s — the replay lost the request", calls.Load()+1, body)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	t.Cleanup(provider.Close)

	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	srv := NewServer(testToken, engine.NewEngine)
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	c := NewClient(ClientOptions{})
	// groq's key goes in Authorization; the signer here uses x-api-key for the
	// provider to check, which is fine — the wire format only says what the
	// screen would do, and this screen does this.
	ServeProvider(c, signWithKey(t, "sk-retry"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Connect(ctx, "tcp", strings.TrimPrefix(hs.URL, "http://"), testToken); err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	p, err := model.NewProvider(model.ProviderOptions{
		Provider:  "groq",
		Model:     "llama-test",
		BaseURL:   provider.URL,
		Transport: srv.Screen().ProviderTransport("groq", ""),
	})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	resp, err := p.Complete(context.Background(), model.Request{Messages: []model.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Text != "ok" {
		t.Errorf("got %q", resp.Text)
	}
	if calls.Load() != 2 {
		t.Errorf("the provider was called %d times, want the 429 and the retry", calls.Load())
	}
}

// No screen is a wait, then a failure the turn records — never a request
// sent unsigned, and never a hang.
func TestNoScreenFailsTheRequestAfterTheWait(t *testing.T) {
	prev := screenWait
	screenWait = 200 * time.Millisecond
	t.Cleanup(func() { screenWait = prev })
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	srv := NewServer(testToken, engine.NewEngine)
	network := http.DefaultTransport.(*http.Transport).Clone()
	network.ResponseHeaderTimeout = 5 * time.Second
	transport := srv.Screen().ProviderTransport("anthropic", "anthropic")(network)

	req, _ := http.NewRequest("POST", "http://127.0.0.1:1/v1/messages", strings.NewReader("{}"))
	start := time.Now()
	_, err := transport.RoundTrip(req)
	if !errors.Is(err, ErrNoScreen) {
		t.Fatalf("err = %v, want ErrNoScreen", err)
	}
	if time.Since(start) < screenWait {
		t.Error("the proxy gave up before the wait for a screen was over")
	}
}
