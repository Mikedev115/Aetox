package model

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The shape the ChatGPT backend actually answers with, cut down to the fields
// read here. Captured 13 ก.ย. 2026: luna stops at max, sol goes on to ultra,
// gpt-5.5 stops at xhigh, and a hidden row (codex-auto-review) rides along.
const codexModelsBody = `{"models":[
 {"slug":"gpt-5.6-sol","visibility":"list","default_reasoning_level":"low",
  "service_tiers":[{"id":"priority","name":"Fast","description":"1.5x speed, increased usage"}],
  "supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"},{"effort":"max"},{"effort":"ultra"}]},
 {"slug":"gpt-5.6-luna","visibility":"list","default_reasoning_level":"medium",
  "service_tiers":[{"id":"priority","name":"Fast","description":"1.5x speed, increased usage"}],
  "supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"},{"effort":"max"}]},
 {"slug":"gpt-5.5","visibility":"list","default_reasoning_level":"medium",
  "supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"}]},
 {"slug":"codex-auto-review","visibility":"hide","default_reasoning_level":"medium",
  "supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"},{"effort":"max"}]}
]}`

func TestCodexServiceTiersComeFromTheBackendPerModel(t *testing.T) {
	withCodexStatement(t)

	tiers := SupportedServiceTiers("codex", "gpt-5.6-luna")
	if len(tiers) != 1 || tiers[0].ID != "priority" || tiers[0].Name != "Fast" || !strings.Contains(tiers[0].Description, "1.5x") {
		t.Fatalf("luna service tiers = %+v, want the backend's Fast 1.5x row", tiers)
	}
	if got := NormalizeServiceTier("codex", "gpt-5.6-luna", "fast"); got != "priority" {
		t.Fatalf("fast alias = %q, want priority", got)
	}
	if got := NormalizeServiceTier("codex", "gpt-5.5", "priority"); got != "" {
		t.Fatalf("unsupported gpt-5.5 tier = %q, want standard", got)
	}
	if got := NormalizeServiceTier("openai", "gpt-5.6-luna", "priority"); got != "" {
		t.Fatalf("non-Codex tier = %q, want standard", got)
	}
}

func withCodexStatement(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/models") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(codexModelsBody))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { SetResponsesModelFacts(nil) })
	if _, err := DiscoverResponsesModels(context.Background(), "codex", srv.URL, nil, "tok"); err != nil {
		t.Fatalf("DiscoverResponsesModels: %v", err)
	}
}

// The ladder the picker offers for a Codex model is the one the backend
// states for that slug — not a table of ours. Owner, 13 ก.ย. 2026, on finding
// max missing for luna: "ควรมีตามที่เขาส่งมานะ ไม่ใช่ตั้งเอง".
func TestCodexLadderIsWhatTheBackendStatesPerModel(t *testing.T) {
	withCodexStatement(t)

	cases := []struct {
		model  string
		levels []string
		def    string
	}{
		{"gpt-5.6-luna", []string{"off", "low", "medium", "high", "xhigh", "max"}, "medium"},
		{"gpt-5.6-sol", []string{"off", "low", "medium", "high", "xhigh", "max", "ultra"}, "low"},
		{"gpt-5.5", []string{"off", "low", "medium", "high", "xhigh"}, "medium"},
		// Hidden on the picker, still stated: a slug typed by hand keeps its ladder.
		{"codex-auto-review", []string{"off", "low", "medium", "high", "xhigh", "max"}, "medium"},
	}
	for _, tc := range cases {
		caps := ResolveThinkingCapabilities("codex", tc.model)
		if !caps.Supported || strings.Join(caps.Levels, ",") != strings.Join(tc.levels, ",") {
			t.Errorf("%s: levels = %v, want %v", tc.model, caps.Levels, tc.levels)
		}
		if caps.Default != tc.def {
			t.Errorf("%s: default = %q, want the backend's %q", tc.model, caps.Default, tc.def)
		}
		if caps.Source != "responses-models" {
			t.Errorf("%s: source = %q — answered from a table, not from the backend", tc.model, caps.Source)
		}
		// Every stated rung goes on the wire as itself; off never does.
		for _, l := range caps.Levels {
			if l == "off" {
				if _, sent := caps.Wire[l]; sent {
					t.Errorf("%s: off has a wire value", tc.model)
				}
				continue
			}
			if caps.Wire[l] != l {
				t.Errorf("%s: wire[%q] = %q, want the rung itself", tc.model, l, caps.Wire[l])
			}
		}
	}
}

// A saved depth the slug does not state folds onto the nearest one it does,
// so switching from sol (ultra) to gpt-5.5 (xhigh) cannot 400 the next turn.
func TestCodexUnstatedDepthFoldsOntoTheNearestStatedRung(t *testing.T) {
	withCodexStatement(t)
	caps := ResolveThinkingCapabilities("codex", "gpt-5.5")
	for from, want := range map[string]string{"ultra": "xhigh", "max": "xhigh", "none": "off", "minimal": "low"} {
		if got := caps.Aliases[from]; got != want {
			t.Errorf("gpt-5.5: %q folds onto %q, want %q", from, got, want)
		}
	}
	luna := ResolveThinkingCapabilities("codex", "gpt-5.6-luna")
	if luna.Aliases["ultra"] != "max" {
		t.Errorf("luna: ultra folds onto %q, want max — the deepest rung it states", luna.Aliases["ultra"])
	}
	if _, aliased := luna.Aliases["max"]; aliased {
		t.Error("luna: max is stated and must not be folded away")
	}
}

// Before the list has ever been fetched the hand table answers — a shallow
// ladder every slug accepts — and it says so in Source.
func TestCodexFallsBackToTheTableOnlyWhenNothingWasStated(t *testing.T) {
	SetResponsesModelFacts(nil)
	caps := ResolveThinkingCapabilities("codex", "gpt-5.6-luna")
	if caps.Source != "responses-reasoning-effort" {
		t.Fatalf("source = %q, want the fallback table", caps.Source)
	}
	if containsLevel(caps.Levels, "max") {
		t.Error("the fallback offers max, which it cannot promise for every slug")
	}
}

// The statement survives a restart: written beside model-catalog.json on
// discovery, read back by the same install call the entry points already make.
func TestCodexStatementIsRememberedAcrossRuns(t *testing.T) {
	root := t.TempDir()
	InstallCachedResponsesModelFacts(root) // nothing there yet; remembers the root
	t.Cleanup(func() {
		responsesFactsMu.Lock()
		responsesFactsRoot = ""
		responsesFactsMu.Unlock()
	})
	withCodexStatement(t)

	SetResponsesModelFacts(nil) // "restart"
	if _, known := ResponsesModelFactsFor("gpt-5.6-luna"); known {
		t.Fatal("cleared, yet still known")
	}
	InstallCachedCatalog(root)
	f, known := ResponsesModelFactsFor("gpt-5.6-luna")
	if !known || !containsLevel(f.ReasoningLevels, "max") || f.DefaultReasoningLevel != "medium" {
		t.Fatalf("after restart: known=%v facts=%+v", known, f)
	}
}
