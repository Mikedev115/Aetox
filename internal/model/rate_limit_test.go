package model

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The ChatGPT backend's 429 names the moment the plan refills. That has always
// been printed; since 14 ก.ย. 2026 it is also a fact the turn can act on
// (cognitive.askAgainAfterLimit), which means the error has to carry it as a
// time rather than as prose.
func TestCodex429CarriesItsResetAsAFact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"usage_limit_reached","plan_type":"plus","resets_in_seconds":7200}}`))
	}))
	defer server.Close()

	p, _ := NewResponsesProvider(ResponsesConfig{Provider: "codex", Model: "m", BaseURL: server.URL, APIKey: "tok"})
	before := time.Now()
	_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err == nil {
		t.Fatal("a 429 was reported as success")
	}
	// The sentence is unchanged — a caller that will not wait shows what it
	// always has.
	if !strings.Contains(err.Error(), "plus plan's limit is used up") || !strings.Contains(err.Error(), "2 hours") {
		t.Errorf("err = %v; want the plan and the wait in words", err)
	}
	resetAt, ok := RateLimitResetAt(err)
	if !ok {
		t.Fatalf("err = %v carries no reset", err)
	}
	if got := resetAt.Sub(before); got < 2*time.Hour-time.Minute || got > 2*time.Hour+time.Minute {
		t.Errorf("reset is %s away, want ~2h", got)
	}
}

// A body that says nothing about when, beside headers that do: the x-codex-*
// windows ride every reply, and a window sitting at zero with a reset stated
// is the same fact by another route.
func TestCodex429FallsBackToTheSpentWindowsReset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		h := w.Header()
		h.Set("x-codex-primary-used-percent", "100")
		h.Set("x-codex-primary-window-minutes", "300")
		h.Set("x-codex-primary-reset-after-seconds", "5400")
		// The weekly window still has room, so it is not what refused the turn
		// and must not be the reset that is counted to.
		h.Set("x-codex-secondary-used-percent", "40")
		h.Set("x-codex-secondary-window-minutes", "10080")
		h.Set("x-codex-secondary-reset-after-seconds", "300000")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	p, _ := NewResponsesProvider(ResponsesConfig{Provider: "codex", Model: "m", BaseURL: server.URL, APIKey: "tok"})
	before := time.Now()
	_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	resetAt, ok := RateLimitResetAt(err)
	if !ok {
		t.Fatalf("err = %v carries no reset; the spent window stated one", err)
	}
	if got := resetAt.Sub(before); got < 90*time.Minute-time.Minute || got > 90*time.Minute+time.Minute {
		t.Errorf("reset is %s away, want ~90m (the spent window, not the weekly one)", got)
	}
}

// No reset anywhere is not a wait: the sentence stands alone and the turn ends
// with it, as it did before.
func TestA429WithNoResetIsNotARateLimitFact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
	}))
	defer server.Close()

	p, _ := NewResponsesProvider(ResponsesConfig{Provider: "codex", Model: "m", BaseURL: server.URL, APIKey: "tok"})
	_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if _, ok := RateLimitResetAt(err); ok {
		t.Errorf("err = %v claims a reset nobody stated", err)
	}
	if err == nil || !strings.Contains(err.Error(), "plan limit reached") {
		t.Errorf("err = %v; want the old sentence", err)
	}
}

// opencode-go meters the same five-hour window and states it nowhere on the
// refusing reply — no Retry-After, no headers. The reset is on GET /usage, so
// the provider asks, authorised the way the refused turn was, and takes the
// earliest reset among the windows the gateway marks rate-limited.
func TestOpencodeGo429AsksUsageForTheSpentWindowsReset(t *testing.T) {
	resetAt := time.Now().Add(90 * time.Minute).UTC().Truncate(time.Second)
	var usageAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/zen/go/v1/usage":
			usageAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{"usage":{` +
				`"rolling":{"status":"rate-limited","percent":100,"resetsAt":"` + resetAt.Format(time.RFC3339) + `"},` +
				`"weekly":{"status":"ok","percent":40,"resetsAt":"` + time.Now().Add(72*time.Hour).UTC().Format(time.RFC3339) + `"},` +
				`"monthly":{"status":"ok","percent":10,"resetsAt":"` + time.Now().Add(400*time.Hour).UTC().Format(time.RFC3339) + `"}}}`))
		default:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
		}
	}))
	defer server.Close()

	p, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{Provider: "opencode-go", Model: "m", BaseURL: server.URL + "/zen/go/v1", APIKey: "k", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewOpenAICompatibleProvider: %v", err)
	}
	_, err = p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	got, ok := RateLimitResetAt(err)
	if !ok {
		t.Fatalf("err = %v carries no reset; /usage stated one", err)
	}
	if !got.Equal(resetAt) {
		t.Errorf("reset = %s, want the rolling window's %s (not the weekly one)", got, resetAt)
	}
	if usageAuth != "Bearer k" {
		t.Errorf("/usage was asked with %q; want the same credential the turn carried", usageAuth)
	}
	if !strings.Contains(err.Error(), "window is used up") {
		t.Errorf("err = %v; want a sentence that says the window is spent", err)
	}
}

// A 429 the gateway will not date — /usage down, or no window rate-limited —
// keeps the sentence it always had, and the turn ends with it.
func TestOpencodeGo429WithoutADatedWindowIsNotWaitedOn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/zen/go/v1/usage" {
			_, _ = w.Write([]byte(`{"usage":{"rolling":{"status":"ok","percent":50,"resetsAt":"2099-01-01T00:00:00Z"}}}`))
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer server.Close()

	p, _ := NewOpenAICompatibleProvider(OpenAICompatibleConfig{Provider: "opencode-go", Model: "m", BaseURL: server.URL + "/zen/go/v1", APIKey: "k", Timeout: 5 * time.Second})
	_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if _, ok := RateLimitResetAt(err); ok {
		t.Errorf("err = %v claims a reset no rate-limited window stated", err)
	}
	if err == nil || !strings.Contains(err.Error(), "rate limiting this key") {
		t.Errorf("err = %v; want the old sentence", err)
	}
}

// Every other host on this wire keeps its sentence: the owner scoped the wait
// to the two providers that meter a plan by the hour, and a host that merely
// says Retry-After is not one of them.
func TestOtherHosts429IsNotARateLimitFact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer server.Close()

	p, _ := NewOpenAICompatibleProvider(OpenAICompatibleConfig{Provider: "custom", Model: "m", BaseURL: server.URL, APIKey: "k", Timeout: 5 * time.Second})
	_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if _, ok := RateLimitResetAt(err); ok {
		t.Errorf("err = %v was typed as a wait; only codex and opencode-go are", err)
	}
	if err == nil || !strings.Contains(err.Error(), "Try again in 60 minutes") {
		t.Errorf("err = %v; want the Retry-After sentence as before", err)
	}
}
