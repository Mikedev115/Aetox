package engine

import (
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
)

// "Never answered" and "answered, states no limits" both arrive as an empty
// slice and must not collapse into one state on screen: the first is "not
// known yet, chat once and it appears", the second is "this provider does not
// report a quota". Only the presence of the key tells them apart. The screen
// composes the account card from this (desktop/providers.go); what the engine
// answers is what it saw on the headers of turns.
func TestQuotaKnownSeparatesSilenceFromAbsence(t *testing.T) {
	app := &Engine{}

	if _, known := app.ProviderQuotas("groq"); known {
		t.Error("QuotaKnown = true for a provider that has never answered a turn")
	}

	app.rememberQuotas("groq", nil)
	quotas, known := app.ProviderQuotas("groq")
	if !known {
		t.Error("QuotaKnown = false after the provider answered; want true with no windows")
	}
	if len(quotas) != 0 {
		t.Errorf("Quotas = %+v; want none", quotas)
	}
}

func TestRememberQuotasNormalizesTheProviderName(t *testing.T) {
	app := &Engine{}
	q := model.Quota{Window: "week", RemainingPercent: 12, ObservedAt: time.Now()}
	// The clients report whatever name they were built with; the panel looks
	// up canonical names.
	app.rememberQuotas("claude", []model.Quota{q})

	got, known := app.ProviderQuotas("anthropic")
	if !known || len(got) != 1 || got[0].RemainingPercent != 12 {
		t.Fatalf("anthropic quotas = %+v (known=%v); want the window filed under the alias", got, known)
	}
}

// A window the screen fetched with the key — OpenRouter beside its credits —
// lands in the same sink as the headers, so the next card reads it back.
func TestNoteProviderQuotasFeedsTheSameSink(t *testing.T) {
	app := &Engine{}
	app.NoteProviderQuotas("openrouter", []model.Quota{{Window: "day", RemainingPercent: 40}})
	got, known := app.ProviderQuotas("openrouter")
	if !known || len(got) != 1 || got[0].RemainingPercent != 40 {
		t.Fatalf("quotas = %+v (known=%v)", got, known)
	}
	app.NoteProviderQuotas("openrouter", nil)
	if got, _ := app.ProviderQuotas("openrouter"); len(got) != 1 {
		t.Error("an empty note overwrote a window the provider had stated")
	}
}
