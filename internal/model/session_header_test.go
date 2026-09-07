package model

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The headers a gateway reads before it looks at the body at all.
//
// opencode's Go plan answers 400 "Request is missing x-opencode-session and
// cannot be routed efficiently" to a request without one — no model reached,
// no token spent — so this is not a nicety that can regress quietly: losing it
// takes the whole provider down. Asserted against a real server rather than on
// the struct, for the reason wire_shape_test.go states: a header that is set
// on the wrong object looks fine from inside.

// captureHeaders answers one request with `reply` and hands back what was sent.
func captureHeaders(t *testing.T, reply string) (*httptest.Server, func() []http.Header) {
	t.Helper()
	var seen []http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		seen = append(seen, r.Header.Clone())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	return srv, func() []http.Header { return seen }
}

const okCompletion = `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`

func askTwice(t *testing.T, p *OpenAICompatibleProvider, model string) {
	t.Helper()
	for i := 0; i < 2; i++ {
		req := Request{Model: model, Messages: []Message{{Role: RoleUser, Content: "hi"}}}
		if _, err := p.Complete(context.Background(), req); err != nil {
			t.Fatalf("complete %d: %v", i+1, err)
		}
	}
}

func TestOpencodeSendsStableSessionHeader(t *testing.T) {
	for _, provider := range []string{"opencode", "opencode-go"} {
		t.Run(provider, func(t *testing.T) {
			srv, headers := captureHeaders(t, okCompletion)
			defer srv.Close()

			p, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
				Provider: provider, Model: "glm-5.3-flash", APIKey: "k", BaseURL: srv.URL})
			if err != nil {
				t.Fatalf("provider: %v", err)
			}
			askTwice(t, p, "glm-5.3-flash")

			sent := headers()
			if len(sent) != 2 {
				t.Fatalf("expected two requests, saw %d", len(sent))
			}
			first := sent[0].Get("x-opencode-session")
			if first == "" {
				t.Fatalf("no x-opencode-session; the gateway 400s this before reading the body")
			}
			// Stable is the whole ask: the header is what lets them route a
			// conversation to the same place and reuse its prompt cache, and a
			// fresh id per turn would be worse than useless.
			if second := sent[1].Get("x-opencode-session"); second != first {
				t.Errorf("session id changed between turns: %q then %q", first, second)
			}
			if ua := sent[0].Get("User-Agent"); !strings.HasPrefix(ua, "Aetox/") {
				t.Errorf("User-Agent = %q, want Aetox/<version> — they ask a client to name itself, not its HTTP library", ua)
			}
		})
	}
}

// Two chats open at once are two conversations, and the engine behind each is
// its own provider instance — so the ids must not collide.
func TestSessionHeaderDiffersPerProviderInstance(t *testing.T) {
	srv, headers := captureHeaders(t, okCompletion)
	defer srv.Close()

	for i := 0; i < 2; i++ {
		p, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
			Provider: "opencode-go", Model: "glm-5.3-flash", APIKey: "k", BaseURL: srv.URL})
		if err != nil {
			t.Fatalf("provider %d: %v", i+1, err)
		}
		req := Request{Model: "glm-5.3-flash", Messages: []Message{{Role: RoleUser, Content: "hi"}}}
		if _, err := p.Complete(context.Background(), req); err != nil {
			t.Fatalf("complete %d: %v", i+1, err)
		}
	}

	sent := headers()
	if len(sent) != 2 {
		t.Fatalf("expected two requests, saw %d", len(sent))
	}
	if a, b := sent[0].Get("x-opencode-session"), sent[1].Get("x-opencode-session"); a == b {
		t.Errorf("two engines shared one session id (%q)", a)
	}
}

// Every other row asks for no such header, and sending one there would be
// inventing a contract nobody published.
func TestNoSessionHeaderWhereTheCatalogNamesNone(t *testing.T) {
	srv, headers := captureHeaders(t, okCompletion)
	defer srv.Close()

	p, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		Provider: "deepseek", Model: "deepseek-v4-flash", APIKey: "k", BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	req := Request{Model: "deepseek-v4-flash", Messages: []Message{{Role: RoleUser, Content: "hi"}}}
	if _, err := p.Complete(context.Background(), req); err != nil {
		t.Fatalf("complete: %v", err)
	}

	sent := headers()
	if len(sent) != 1 {
		t.Fatalf("expected one request, saw %d", len(sent))
	}
	if got := sent[0].Get("x-opencode-session"); got != "" {
		t.Errorf("deepseek was sent an opencode header: %q", got)
	}
}
