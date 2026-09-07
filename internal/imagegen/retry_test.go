package imagegen

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The measured case: the transport fails once, and the same call made a moment
// later works. Before this, that cost a whole model turn — a failed tool call,
// an apology, a re-planned prompt, a second call.
func TestATransportFailureIsRetriedOnce(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			// Hang up mid-request: the client sees a transport error, not a status.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Error("no hijacker")
				return
			}
			conn, _, err := hj.Hijack()
			if err == nil {
				_ = conn.Close()
			}
			return
		}
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"seen":"` + string(body) + `"}`))
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, strings.NewReader(`{"prompt":"a cat"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := doWithRetry(srv.Client(), req)
	if err != nil {
		t.Fatalf("doWithRetry gave up: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)

	if hits.Load() != 2 {
		t.Errorf("server saw %d requests, want 2", hits.Load())
	}
	// The body has to survive the retry — the first attempt consumed it, and a
	// second request with an empty body is a 400 wearing a retry's clothes.
	if !strings.Contains(string(got), `a cat`) {
		t.Errorf("the retry sent no body: %s", got)
	}
}

// A vendor's own answer is never retried. A 429 means slow down, and hammering
// it is how a rate limit becomes a ban.
func TestAStatusAnswerIsNotRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := doWithRetry(srv.Client(), req)
	if err != nil {
		t.Fatalf("a 429 became an error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if hits.Load() != 1 {
		t.Errorf("server saw %d requests, want 1 — a status answer is not a transport failure", hits.Load())
	}
}

// A cancelled turn ends the call. Retrying it would keep working for somebody
// who has already stopped waiting.
func TestACancelledCallIsNotRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, err := hj.Hijack()
			if err == nil {
				_ = conn.Close()
			}
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	// Cancelled the moment the first attempt fails.
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	if _, err := doWithRetry(srv.Client(), req); err == nil {
		t.Fatal("a cancelled call reported success")
	}
	if hits.Load() > 1 {
		t.Errorf("server saw %d requests — a cancelled call must not be retried", hits.Load())
	}
	cancel()
}
