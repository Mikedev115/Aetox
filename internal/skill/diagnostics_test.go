package skill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Folder mode without needing a language server installed: these two paths are
// decided before any server is contacted, and they are the ones a model hits
// first when it points diagnostics at the wrong place.
func TestDiagnosticsFolderWithNothingToCheck(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("# hi\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	s := &diagnosticsSkill{root: root}

	out, err := s.ExecuteTool(context.Background(), map[string]any{"path": "."})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if !out.Success {
		t.Error("Success = false; a folder with nothing checkable is an answer, not a failure")
	}
	if !strings.Contains(out.Content, "nothing checked") {
		t.Errorf("Content = %q, want it to say nothing was checked rather than imply everything is clean", out.Content)
	}
}

// The exclusions grep and glob already use have to apply here too: node_modules
// holds more TypeScript than the project does, and none of it is the answer.
func TestDiagnosticsFolderSkipsDependencyTrees(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "pkg", "index.ts"), []byte("export const x = 1\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	s := &diagnosticsSkill{root: root}

	out, err := s.ExecuteTool(context.Background(), map[string]any{"path": "."})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	// Only node_modules holds a checkable file, so a walk that skipped it
	// correctly has nothing left to check.
	if !strings.Contains(out.Content, "nothing checked") {
		t.Errorf("Content = %q, want node_modules skipped", out.Content)
	}
}

func TestDiagnosticsRejectsEscape(t *testing.T) {
	s := &diagnosticsSkill{root: t.TempDir()}
	if _, err := s.ExecuteTool(context.Background(), map[string]any{"path": "../../etc"}); err == nil {
		t.Fatal("a path outside the sandbox must be refused")
	}
}

// The last sentence of the `errors` line in codebase_pack.go moved to Guidance on
// 16 ก.ย. to pay for what the `trace` line needed, and it was the only statement
// a model ever read about what a clean answer is NOT — the copy that repeats it is
// diagnostics.go's own inner ToolDefinition, which no registry returns. Guidance
// is delivered once, with the first result, which is the result most likely to be
// "(no problems)". Pinned so the next edit to the pack cannot drop it in silence.
func TestTheErrorsActStillSaysWhatAnUncheckedFileIsNot(t *testing.T) {
	guidance := (&codebaseSkill{}).Guidance(map[string]any{"action": "errors"})
	if guidance == "" {
		t.Fatal("the `errors` act teaches nothing, so nothing says what \"(no problems)\" is not")
	}
	for _, want := range []string{"no server is installed", "not checked"} {
		if !strings.Contains(guidance, want) {
			t.Errorf("the guidance never mentions %q: %q", want, guidance)
		}
	}
}
