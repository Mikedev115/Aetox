package model

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// The Transport seam (§248 A3): a provider built with one holds no key,
// attaches no key, and needs none — the transport is the credential. What it
// signs is exactly what reaches the provider, and a retry re-enters it.
func TestTransportOwnsTheCredentialAndTheWireClientAttachesNone(t *testing.T) {
	var seen atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer signed-by-screen" {
			t.Errorf("provider saw Authorization %q, want the transport's", got)
		}
		if seen.Add(1) == 1 {
			// One retryable failure, so the second attempt proves the signer
			// runs per attempt rather than once per request.
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	var signed atomic.Int32
	sign := func(network http.RoundTripper) http.RoundTripper {
		return roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Authorization"); got != "" {
				t.Errorf("the wire client attached Authorization %q itself; with a Transport it must attach nothing", got)
			}
			signed.Add(1)
			req = req.Clone(req.Context())
			req.Header.Set("Authorization", "Bearer signed-by-screen")
			return network.RoundTrip(req)
		})
	}

	// groq: an OpenAI-compatible row that requires a key, so the transport
	// standing in for one is the thing under test.
	p, err := NewProvider(ProviderOptions{
		Provider:  "groq",
		Model:     "llama-test",
		BaseURL:   server.URL,
		Transport: sign,
	})
	if err != nil {
		t.Fatalf("a Transport must stand in for the key groq requires: %v", err)
	}
	resp, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Text != "ok" {
		t.Fatalf("got %q", resp.Text)
	}
	if signed.Load() != 2 || seen.Load() != 2 {
		t.Fatalf("signed %d, served %d — the retry must pass through the signer again", signed.Load(), seen.Load())
	}
}

// Without a Transport the old contract stands: a provider that requires a key
// still refuses to be built without one. The seam relaxes nothing for the
// path that does not use it.
func TestWithoutATransportAKeyIsStillRequired(t *testing.T) {
	for _, provider := range []string{"groq", "openai", "anthropic"} {
		_, err := NewProvider(ProviderOptions{Provider: provider, Model: "m"})
		if !errors.Is(err, ErrMissingAPIKey) {
			t.Errorf("%s without key or transport: got %v, want ErrMissingAPIKey", provider, err)
		}
		if _, err := NewProvider(ProviderOptions{Provider: provider, Model: "m", Transport: func(n http.RoundTripper) http.RoundTripper { return n }}); err != nil {
			t.Errorf("%s with a transport: %v", provider, err)
		}
	}
}

// The Anthropic wire format carries its key in x-api-key, not Authorization:
// with a Transport the client must leave that header out too, and AuthScheme
// must tell the signer which header this wire format wants.
func TestAnthropicWireClientLeavesTheKeyHeaderToTheTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "signed" {
			t.Errorf("x-api-key %q, want the transport's", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`))
	}))
	defer server.Close()

	header, prefix := AuthScheme("anthropic", "")
	if header != "x-api-key" || prefix != "" {
		t.Fatalf("AuthScheme(anthropic) = %q %q", header, prefix)
	}
	if h, pfx := AuthScheme("groq", ""); h != "Authorization" || pfx != "Bearer " {
		t.Fatalf("AuthScheme(groq) = %q %q", h, pfx)
	}
	if h, _ := AuthScheme("ollama", ""); h != "" {
		t.Fatalf("AuthScheme(ollama) = %q, want none", h)
	}

	p, err := NewProvider(ProviderOptions{
		Provider: "anthropic", Model: "claude-test", BaseURL: server.URL,
		Transport: func(network http.RoundTripper) http.RoundTripper {
			return roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get(header) != "" {
					t.Errorf("client attached %s itself", header)
				}
				req = req.Clone(req.Context())
				req.Header.Set(header, prefix+"signed")
				return network.RoundTrip(req)
			})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if resp.Text != "ok" {
		t.Fatalf("got %q", resp.Text)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
