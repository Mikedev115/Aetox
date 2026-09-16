package skill

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The seam the model actually reaches: the packed tool, its sandbox, its real
// indexer, and the answer it would read. The point of the call is the one
// relationship no other act in this pack can see — a frontend call goes through
// a generated Wails binding into a Go method, and the evidence for both hops is
// a file and a line.
func TestCodebaseTraceReportsTheWailsBridgeWithEvidence(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedTraceProject(t, root)

	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: root})
	out, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action":     "trace",
		"path":       "desktop/frontend/src/Pane.svelte",
		"name":       "GetRepoMapGraph",
		"direction":  "between",
		"targetPath": "desktop/app.go",
	})
	if err != nil {
		t.Fatalf("trace failed: %v\n%s", err, out.Content)
	}
	for _, want := range []string{
		"outcome=complete",
		"facts=2",
		"index: built for this answer",
		"desktop/frontend/src/Pane.svelte:5",
		"desktop/frontend/wailsjs/go/main/App.js:1",
		"[reference · resolved · wails-v2]",
		"desktop/app.go:5",
		"[wails_rpc · resolved · wails-v2]",
	} {
		if !strings.Contains(out.Content, want) {
			t.Errorf("trace output is missing %q:\n%s", want, out.Content)
		}
	}
	if out.ResultCount != 2 {
		t.Errorf("ResultCount = %d, want the two hops the timeline draws", out.ResultCount)
	}
}

// A hop that was never established is said out loud. The count in the answer is
// what stops "no hops" from reading as "nothing is connected".
func TestCodebaseTraceSaysHowManyHopsItCouldNotEstablish(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedTraceProject(t, root)
	// A binding the generated dispatch body does not name: the adapter must
	// refuse it rather than join the two ends by name.
	writeTraceFile(t, root, "desktop/frontend/wailsjs/go/main/App.js",
		"export function Ghost(arg1) {\n  return window['go']['main']['App']['SomethingElse'](arg1);\n}\n")

	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: root})
	out, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action": "trace",
		"path":   "desktop/frontend/wailsjs/go/main/App.js",
		"name":   "Ghost",
	})
	if err != nil {
		t.Fatalf("trace failed: %v\n%s", err, out.Content)
	}
	if !strings.Contains(out.Content, "could not be established") {
		t.Errorf("an unanswered binding must be reported, not dropped:\n%s", out.Content)
	}
	// And it must not read as a project with no Wails in it: there IS an
	// app.js under wailsjs/go, this adapter simply cannot read what it says.
	if strings.Contains(out.Content, "outcome=unsupported") {
		t.Errorf("a generated binding this adapter cannot pair is unknown, not absent:\n%s", out.Content)
	}
}

// The unfocused desk is the whole machine, and an index of "wherever the
// session stands" is a wrong answer ranked confidently — the refusal repo_map
// makes, for the same reason.
func TestCodebaseTraceRefusesAnUnfocusedSession(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: t.TempDir(), OpenSandbox: true})

	_, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action": "trace",
		"path":   "anything.go",
		"name":   "Anything",
	})
	if err == nil || !strings.Contains(err.Error(), "no project is focused") {
		t.Fatalf("unfocused trace must refuse and name the fix, got %v", err)
	}
}

func seedTraceProject(t *testing.T, root string) {
	t.Helper()
	writeTraceFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeTraceFile(t, root, "desktop/app.go", "package main\n\ntype App struct{}\n\nfunc (a *App) GetRepoMapGraph(max int) {}\n")
	writeTraceFile(t, root, "desktop/frontend/wailsjs/go/main/App.js",
		"export function GetRepoMapGraph(arg1) {\n  return window['go']['main']['App']['GetRepoMapGraph'](arg1);\n}\n")
	writeTraceFile(t, root, "desktop/frontend/src/Pane.svelte", `<script lang="ts">
  import { GetRepoMapGraph } from '../wailsjs/go/main/App'

  async function load(): Promise<void> {
    await GetRepoMapGraph(-1)
  }
</script>
`)
}

func writeTraceFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The direction is not decoration on a hit test: the same seed read the other
// way has to answer about the other half of the boundary.
func TestCodebaseTraceWalksCallersTheOtherWay(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedTraceProject(t, root)

	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: root})
	out, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action":    "trace",
		"path":      "desktop/frontend/wailsjs/go/main/App.js",
		"name":      "GetRepoMapGraph",
		"direction": "callers",
		"depth":     1,
	})
	if err != nil {
		t.Fatalf("trace failed: %v\n%s", err, out.Content)
	}
	if !strings.Contains(out.Content, "desktop/frontend/src/Pane.svelte:5") {
		t.Errorf("callers must walk back to the frontend call:\n%s", out.Content)
	}
	if strings.Contains(out.Content, "desktop/app.go:5") {
		t.Errorf("callers must not walk forward into the Go method:\n%s", out.Content)
	}
}

// A path that does not exist is not a file with no connections. The tool fails
// and says which path, rather than answering the empty question.
func TestCodebaseTraceFailsOnAPathThatIsNotThere(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedTraceProject(t, root)

	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: root})
	_, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action": "trace",
		"path":   "desktop/pane.svelte",
		"name":   "GetRepoMapGraph",
	})
	if err == nil || !strings.Contains(err.Error(), "desktop/pane.svelte") {
		t.Fatalf("a missing path must fail and name itself, got %v", err)
	}
}

// A depth the budget cut is reported, and the header states the walk that
// happened rather than the one that was asked for.
func TestCodebaseTraceReportsAClippedDepth(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedTraceProject(t, root)

	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: root})
	out, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action": "trace",
		"path":   "desktop/frontend/src/Pane.svelte",
		"name":   "GetRepoMapGraph",
		"depth":  9,
	})
	if err != nil {
		t.Fatalf("trace failed: %v\n%s", err, out.Content)
	}
	if !strings.Contains(out.Content, "TRUNCATED: depth budget reached") {
		t.Errorf("a clamped depth must say so:\n%s", out.Content)
	}
	if !strings.Contains(out.Content, "(depth=3)") {
		t.Errorf("the header must state the depth walked, not the one asked for:\n%s", out.Content)
	}
	if !out.Truncated {
		t.Error("the timeline's truncation flag must follow the answer")
	}
}

// An import hop names a file and no line inside it, so the evidence line for
// that hop is the importer's own and the destination is printed as a path.
func TestCodebaseTracePrintsAnImportHopWithoutAnInventedLine(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedTraceProject(t, root)
	writeTraceFile(t, root, "lib/util.ts", "export function work(): void {}\n")
	writeTraceFile(t, root, "app.ts", "import { work } from './lib/util'\n")

	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: root})
	out, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action": "trace",
		"path":   "app.ts",
		"name":   "work",
		"depth":  1,
	})
	if err != nil {
		t.Fatalf("trace failed: %v\n%s", err, out.Content)
	}
	if !strings.Contains(out.Content, "app.ts:1") {
		t.Errorf("the importer's own line is the evidence:\n%s", out.Content)
	}
	if strings.Contains(out.Content, "lib/util.ts:0") || !strings.Contains(out.Content, "-> lib/util.ts [import") {
		t.Errorf("an import's destination is the file, with no line invented for it:\n%s", out.Content)
	}
}

// An answer does not carry the command that produced it: the model is sent
// Content and nothing else. So the direction has to be in the answer, because
// it decides how every arrow below reads — the same seed walked the other way
// answers the other half of the boundary, and the two renderings are otherwise
// identical.
func TestCodebaseTraceNamesTheDirectionInTheAnswer(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedTraceProject(t, root)

	registry := NewDefaultRegistry(RegistryOptions{SandboxRoot: root})
	out, _, err := NewDispatcher(registry).ExecuteTool(context.Background(), "codebase", map[string]any{
		"action":    "trace",
		"path":      "desktop/frontend/src/Pane.svelte",
		"name":      "GetRepoMapGraph",
		"direction": "callers",
	})
	if err != nil {
		t.Fatalf("trace failed: %v\n%s", err, out.Content)
	}
	if !strings.Contains(out.Content, "trace callers GetRepoMapGraph in ") {
		t.Errorf("the answer must say which way it walked:\n%s", out.Content)
	}
}
