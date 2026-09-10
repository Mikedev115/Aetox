package model

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/oauth"
)

func TestAntigravityQuotaSummaryWithWeeklyBuckets(t *testing.T) {
	var gotAuth, gotUA, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"groups": [
				{
					"displayName": "Gemini Models",
					"buckets": [
						{
							"bucketId": "gemini-weekly",
							"displayName": "Weekly Limit Remaining",
							"window": "weekly",
							"resetTime": "2030-01-07T00:00:00Z",
							"remainingFraction": 0.64
						},
						{
							"bucketId": "gemini-5h",
							"displayName": "Five Hour Limit Remaining",
							"window": "5h",
							"resetTime": "2030-01-01T05:00:00Z",
							"remainingFraction": 0.82
						}
					]
				},
				{
					"displayName": "Claude and GPT models",
					"buckets": [
						{
							"bucketId": "3p-weekly",
							"displayName": "Weekly Limit Remaining",
							"window": "weekly",
							"resetTime": "2030-01-07T00:00:00Z",
							"remainingFraction": 0.69
						},
						{
							"bucketId": "3p-5h",
							"displayName": "Five Hour Limit Remaining",
							"window": "5h",
							"resetTime": "2030-01-01T05:00:00Z",
							"remainingFraction": 1.0
						}
					]
				}
			]
		}`))
	}))
	defer srv.Close()

	got, err := FetchBalance(context.Background(), "antigravity", srv.URL, "test-token")
	if err != nil {
		t.Fatalf("FetchBalance: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("gotAuth = %q, want Bearer test-token", gotAuth)
	}
	if gotUA != "antigravity/2.12.2" {
		t.Errorf("gotUA = %q, want antigravity/2.12.2", gotUA)
	}
	if gotPath != "/v1internal:retrieveUserQuotaSummary" {
		t.Errorf("gotPath = %q, want /v1internal:retrieveUserQuotaSummary", gotPath)
	}
	if len(got.Quotas) != 4 {
		t.Fatalf("len(got.Quotas) = %d, want 4", len(got.Quotas))
	}
	expected := []struct {
		window string
		pct    float64
	}{
		{"gemini", 82},
		{"gemini_week", 64},
		{"claude", 100},
		{"claude_week", 69},
	}
	for i, exp := range expected {
		if got.Quotas[i].Window != exp.window {
			t.Errorf("Quotas[%d].Window = %q, want %q", i, got.Quotas[i].Window, exp.window)
		}
		if got.Quotas[i].RemainingPercent != exp.pct {
			t.Errorf("Quotas[%d].RemainingPercent = %v, want %v", i, got.Quotas[i].RemainingPercent, exp.pct)
		}
		if !got.Quotas[i].HasReset() {
			t.Errorf("Quotas[%d] has no reset time parsed", i)
		}
	}
}

func TestAntigravityQuotaReadsGeminiAndClaude(t *testing.T) {
	var gotAuth, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, ":retrieveUserQuotaSummary") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"defaultAgentModelId": "gemini-3.8-flash-high",
			"models": {
				"gemini-3.8-flash-high": {
					"displayName": "Gemini 3.8 Flash (High)",
					"quotaInfo": {
						"remainingFraction": 0.85,
						"resetTime": "2030-01-01T05:00:00Z"
					}
				},
				"gemini-3.1-pro-low": {
					"displayName": "Gemini 3.1 Pro (Low)",
					"quotaInfo": {
						"remainingFraction": 0.85,
						"resetTime": "2030-01-01T05:00:00Z"
					}
				},
				"claude-sonnet-4-6": {
					"displayName": "Claude Sonnet 4.6 (Thinking)",
					"quotaInfo": {
						"remainingFraction": 0.95,
						"resetTime": "2030-01-01T06:00:00Z"
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	got, err := FetchBalance(context.Background(), "antigravity", srv.URL, "test-token")
	if err != nil {
		t.Fatalf("FetchBalance: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("gotAuth = %q, want Bearer test-token", gotAuth)
	}
	if gotUA != "antigravity/2.12.2" {
		t.Errorf("gotUA = %q, want antigravity/2.12.2", gotUA)
	}
	if got.Kind != "subscription" {
		t.Errorf("kind = %q, want subscription", got.Kind)
	}
	if got.HasAmount {
		t.Error("a subscription reported an amount")
	}
	if !got.Sufficient {
		t.Error("account marked insufficient with 85% remaining")
	}

	if len(got.Quotas) != 2 {
		t.Fatalf("len(got.Quotas) = %d, want 2 (gemini and claude)", len(got.Quotas))
	}

	// First is Gemini
	if got.Quotas[0].Window != "gemini" {
		t.Errorf("Quotas[0].Window = %q, want gemini", got.Quotas[0].Window)
	}
	if got.Quotas[0].RemainingPercent != 85 {
		t.Errorf("Quotas[0].RemainingPercent = %v, want 85", got.Quotas[0].RemainingPercent)
	}
	if !got.Quotas[0].HasReset() {
		t.Error("Quotas[0] resetAt was not parsed")
	}

	// Second is Claude
	if got.Quotas[1].Window != "claude" {
		t.Errorf("Quotas[1].Window = %q, want claude", got.Quotas[1].Window)
	}
	if got.Quotas[1].RemainingPercent != 95 {
		t.Errorf("Quotas[1].RemainingPercent = %v, want 95", got.Quotas[1].RemainingPercent)
	}
}

func TestAntigravityQuotaExhaustedMarksInsufficient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"defaultAgentModelId": "gemini-3.8-flash-high",
			"models": {
				"gemini-3.8-flash-high": {
					"displayName": "Gemini 3.8 Flash (High)",
					"quotaInfo": {
						"remainingFraction": 0,
						"resetTime": "2030-01-01T05:00:00Z"
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	got, err := FetchBalance(context.Background(), "antigravity", srv.URL, "test-token")
	if err != nil {
		t.Fatalf("FetchBalance: %v", err)
	}
	if got.Sufficient {
		t.Error("all quotas exhausted but sufficient = true")
	}
	if len(got.Quotas) != 1 {
		t.Fatalf("len(got.Quotas) = %d, want 1", len(got.Quotas))
	}
	if got.Quotas[0].RemainingPercent != 0 {
		t.Errorf("RemainingPercent = %v, want 0", got.Quotas[0].RemainingPercent)
	}
}

func TestLiveAntigravityQuota(t *testing.T) {
	if root := realDataRoot(); root != "" {
		t.Setenv("AETOX_DATA_ROOT", root)
	}
	cred, ok := oauth.Get("antigravity")
	if !ok || cred.Access == "" {
		t.Skip("skipping live test: no stored antigravity credentials")
	}

	got, err := FetchBalance(context.Background(), "antigravity", cred.Endpoint, cred.Access)
	if err != nil {
		t.Fatalf("Live FetchBalance failed: %v", err)
	}
	t.Logf("Live Antigravity Quota: Kind=%s, Sufficient=%v, Quotas=%d", got.Kind, got.Sufficient, len(got.Quotas))
	for i, q := range got.Quotas {
		t.Logf("  [%d] Window=%s, Remaining=%.1f%%, ResetAt=%v", i, q.Window, q.RemainingPercent, q.ResetAt)
	}
	if len(got.Quotas) == 0 {
		t.Error("expected at least one quota window from live Antigravity endpoint")
	}
}
