package model

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The store's recorded expiry is a belief and the provider's 401 is the fact:
// on 2026-09-12 the ChatGPT backend answered token_expired to a token whose
// claim said a week remained, and the user was told to sign in again while
// holding a refresh token that would have fixed it. A 401 on a signed-in
// provider now renews once and re-sends; the token that goes out the second
// time is whatever TokenSource yields after the refresh.
func TestResponsesRenewsOnceOn401(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"token_expired"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseLines(`data: {"type":"response.output_text.delta","delta":"ok"}`, `data: {"type":"response.completed","response":{}}`)))
	}))
	defer server.Close()

	token := "stale"
	refreshed := 0
	p, _ := NewResponsesProvider(ResponsesConfig{
		Provider: "codex", Model: "m", BaseURL: server.URL,
		TokenSource:  func(context.Context) (string, error) { return token, nil },
		TokenRefresh: func(context.Context) (string, error) { refreshed++; token = "fresh"; return token, nil },
	})
	resp, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "ok" || refreshed != 1 {
		t.Fatalf("text=%q refreshed=%d", resp.Text, refreshed)
	}
	if strings.Join(seen, ",") != "Bearer stale,Bearer fresh" {
		t.Fatalf("requests carried %v", seen)
	}
}

// When the renewal itself fails there is nothing left to try, and the sentence
// the user sees is the provider's own 401 — that one really does mean sign in
// again. One request, not two.
func TestResponses401StandsWhenRenewalFails(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"token_expired"}}`))
	}))
	defer server.Close()

	p, _ := NewResponsesProvider(ResponsesConfig{
		Provider: "codex", Model: "m", BaseURL: server.URL,
		TokenSource:  func(context.Context) (string, error) { return "stale", nil },
		TokenRefresh: func(context.Context) (string, error) { return "", errors.New("refresh token revoked") },
	})
	_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err == nil || !strings.Contains(err.Error(), "Sign in again") {
		t.Fatalf("err = %v; want the 401 sentence", err)
	}
	if hits != 1 {
		t.Fatalf("server hit %d times; a failed renewal should not re-send", hits)
	}
}

// The same rule on the chat/completions wire: the key path (no TokenRefresh)
// still gets exactly one request, and the sign-in path gets its one retry.
func TestOpenAICompatibleRenewsOnceOn401(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"token expired"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()

	token := "stale"
	p, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		Provider: "github-copilot", Model: "m", BaseURL: server.URL,
		TokenSource:  func(context.Context) (string, error) { return token, nil },
		TokenRefresh: func(context.Context) (string, error) { token = "fresh"; return token, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Text != "ok" || strings.Join(seen, ",") != "Bearer stale,Bearer fresh" {
		t.Fatalf("text=%q requests=%v", resp.Text, seen)
	}

	seen = nil
	keyed, _ := NewOpenAICompatibleProvider(OpenAICompatibleConfig{Provider: "github-copilot", Model: "m", BaseURL: server.URL, APIKey: "stale"})
	if _, err := keyed.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}}); err == nil {
		t.Fatal("a pasted key has nothing to renew; the 401 should stand")
	}
	if len(seen) != 1 {
		t.Fatalf("key path sent %d requests; want 1", len(seen))
	}
}
