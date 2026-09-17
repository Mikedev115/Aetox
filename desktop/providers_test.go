package main

import (
	"net/http"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
)

// A local runtime has no wallet and no window, and must not be reported as an
// error just because there was nothing to fetch.
func TestLocalProviderAccountIsNotAnError(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp(t)
	got := a.ProviderAccountFor("ollama")
	if got.Error != "" {
		t.Errorf("Error = %q; want empty — there was nothing to fetch", got.Error)
	}
	if got.Balance.Kind != "free" {
		t.Errorf("Kind = %q; want free", got.Balance.Kind)
	}
	if got.Balance.HasAmount {
		t.Error("a local runtime reported an amount")
	}
}

// The key desk is the screen's: a key saved here reaches the engine only as
// a rebuilt provider that signs through the transport, never as a value.
func TestSetAPIKeyFilesTheKeyAndTellsTheEngine(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp(t)
	if a.HasAPIKey("groq") {
		t.Fatal("a fresh store claims a key for groq")
	}
	if _, err := a.SetAPIKey("groq", "gsk-test-key-long-enough-to-hint"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}
	if !a.HasAPIKey("groq") {
		t.Error("the key just saved is not found")
	}
	if hint := a.APIKeyHint("groq"); hint != "••••hint" {
		t.Errorf("APIKeyHint = %q, want the last four behind dots", hint)
	}
	if _, err := a.SetAPIKey("groq", "   "); err == nil {
		t.Error("an empty key was accepted")
	}
}

func TestDesktopNoteQuotasFeedsProviderAccount(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp(t)
	h := make(http.Header)
	h.Set("x-codex-primary-used-percent", "20")
	h.Set("x-codex-primary-window-minutes", "300")
	resp := &http.Response{
		StatusCode: 200,
		Header:     h,
	}
	model.NoteQuotas("codex", resp)
	acct := a.ProviderAccountFor("codex")
	if !acct.QuotaKnown {
		t.Fatal("QuotaKnown = false after NoteQuotas on desktop side")
	}
	if len(acct.Quotas) != 1 || acct.Quotas[0].Window != "5h" || acct.Quotas[0].RemainingPercent != 80 {
		t.Fatalf("unexpected quotas: %+v", acct.Quotas)
	}
}
