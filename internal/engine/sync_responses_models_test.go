package engine

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
)

// SyncResponsesModelFacts bridges §261.3's remote engine gap: the statement is
// fetched by the screen process, and sent across RPC to the engine process.
func TestEngineSyncResponsesModelFactsUpdatesLevelsAndPersistsToDisk(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", tempDir)

	// Ensure clean starting state: clear facts so codex resolves from fallback table.
	model.SetResponsesModelFacts(nil)
	t.Cleanup(func() { model.SetResponsesModelFacts(nil) })

	a := &Engine{}

	// Baseline: without facts, gpt-5.6-luna on codex falls back to table (capped at xhigh, no max).
	baseline := a.SupportedThinkLevelsFor("codex", "gpt-5.6-luna")
	if slices.Contains(baseline, "max") {
		t.Fatalf("expected baseline without facts not to contain max, got %v", baseline)
	}

	// Now sync facts from screen.
	facts := []model.ResponsesModelFacts{
		{
			Slug:                  "gpt-5.6-luna",
			ReasoningLevels:       []string{"low", "medium", "high", "xhigh", "max"},
			DefaultReasoningLevel: "medium",
		},
		{
			Slug:                  "gpt-5.6-sol",
			ReasoningLevels:       []string{"low", "medium", "high", "xhigh", "max", "ultra"},
			DefaultReasoningLevel: "low",
		},
	}

	if err := a.SyncResponsesModelFacts(facts); err != nil {
		t.Fatalf("SyncResponsesModelFacts failed: %v", err)
	}

	// 1. In-memory levels should now reflect the backend statement.
	lunaLevels := a.SupportedThinkLevelsFor("codex", "gpt-5.6-luna")
	wantLuna := []string{"off", "low", "medium", "high", "xhigh", "max"}
	if strings.Join(lunaLevels, ",") != strings.Join(wantLuna, ",") {
		t.Errorf("luna levels = %v, want %v", lunaLevels, wantLuna)
	}

	solLevels := a.SupportedThinkLevelsFor("codex", "gpt-5.6-sol")
	wantSol := []string{"off", "low", "medium", "high", "xhigh", "max", "ultra"}
	if strings.Join(solLevels, ",") != strings.Join(wantSol, ",") {
		t.Errorf("sol levels = %v, want %v", solLevels, wantSol)
	}

	// 2. File should be persisted to disk at DataRoot/responses-models.json.
	dataFile := filepath.Join(tempDir, "responses-models.json")
	if _, err := os.Stat(dataFile); err != nil {
		t.Fatalf("responses-models.json was not created on disk: %v", err)
	}

	// 3. Simulate process restart: clear in-memory facts and load from disk.
	model.SetResponsesModelFacts(nil)
	if slices.Contains(a.SupportedThinkLevelsFor("codex", "gpt-5.6-luna"), "max") {
		t.Fatal("expected max to disappear after in-memory facts cleared")
	}

	// Re-install from disk cache.
	model.InstallCachedResponsesModelFacts(tempDir)
	restartedLevels := a.SupportedThinkLevelsFor("codex", "gpt-5.6-luna")
	if !slices.Contains(restartedLevels, "max") {
		t.Errorf("expected max to be restored after loading cached file, got %v", restartedLevels)
	}
}
