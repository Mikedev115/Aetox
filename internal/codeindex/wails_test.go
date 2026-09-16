package codeindex

import (
	"context"
	"reflect"
	"testing"
)

func TestIndexWailsLinksFrontendCallThroughGeneratedBindingToGo(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)

	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.IndexWails(context.Background())
	if result.Outcome != OutcomeComplete || result.Unknown != 0 {
		t.Fatalf("index result = %+v", result)
	}
	frontend, stale, err := indexer.store.FactsFrom("desktop/frontend/src/Pane.svelte")
	if err != nil {
		t.Fatal(err)
	}
	if stale != 0 || len(frontend) != 1 {
		t.Fatalf("frontend facts = %+v stale=%d", frontend, stale)
	}
	if frontend[0].Relation != RelationReference || frontend[0].From.Line != 5 || frontend[0].To.Path != "desktop/frontend/wailsjs/go/main/App.js" || frontend[0].To.Line != 1 {
		t.Fatalf("frontend bridge = %+v", frontend[0])
	}

	binding, stale, err := indexer.store.FactsFrom("desktop/frontend/wailsjs/go/main/App.js")
	if err != nil {
		t.Fatal(err)
	}
	if stale != 0 || len(binding) != 1 {
		t.Fatalf("binding facts = %+v stale=%d", binding, stale)
	}
	if binding[0].Relation != RelationWailsRPC || binding[0].From.Line != 2 || binding[0].To.Path != "desktop/app.go" || binding[0].To.Line != 5 {
		t.Fatalf("RPC bridge = %+v", binding[0])
	}
}

func TestTraceUsesOneBudgetAcrossTheWholePath(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	budget := DefaultBudget()
	budget.MaxFacts = 1
	indexer, err := NewIndexer(root, budget)
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path:      "desktop/frontend/src/Pane.svelte",
		Name:      "GetRepoMapGraph",
		Direction: DirectionCallees,
		Depth:     3,
	})
	if result.Outcome != OutcomePartial || !result.Truncated || result.Reason != "fact budget reached" {
		t.Fatalf("trace result = %+v", result)
	}
	if len(result.Facts) != 1 {
		t.Fatalf("facts = %+v", result.Facts)
	}
}

func TestTraceBetweenFindsTheWailsPath(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path:       "desktop/frontend/src/Pane.svelte",
		Name:       "GetRepoMapGraph",
		Direction:  DirectionBetween,
		TargetPath: "desktop/app.go",
		TargetName: "GetRepoMapGraph",
		Depth:      3,
	})
	if result.Outcome != OutcomeComplete || len(result.Facts) != 2 {
		t.Fatalf("trace result = %+v", result)
	}
	if result.Facts[0].Relation != RelationReference || result.Facts[1].Relation != RelationWailsRPC {
		t.Fatalf("path = %+v", result.Facts)
	}
}

func seedWailsProject(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, root, "desktop/app.go", "package main\n\ntype App struct{}\n\nfunc (a *App) GetRepoMapGraph(max int) {}\n")
	writeFile(t, root, "desktop/frontend/wailsjs/go/main/App.js", "export function GetRepoMapGraph(arg1) {\n  return window['go']['main']['App']['GetRepoMapGraph'](arg1);\n}\n")
	writeFile(t, root, "desktop/frontend/src/Pane.svelte", `<script lang="ts">
  import { GetRepoMapGraph } from '../wailsjs/go/main/App'

  async function load() {
    await GetRepoMapGraph(-1)
  }
</script>
`)
}

// A generated binding whose shape this adapter cannot read is UNKNOWN, not a
// project without Wails. Collapsing the two would let a generator upgrade turn
// every crossed-boundary path into "nothing here", silently.
func TestIndexWailsReportsUnpairedBindingsAsUnknown(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	writeFile(t, root, "desktop/frontend/wailsjs/go/main/Ghost.js",
		"export function Ghost(arg1) {\n  return window['go']['main']['App']['SomethingElse'](arg1);\n}\n")
	writeFile(t, root, "desktop/frontend/wailsjs/go/main/Also.js",
		"export function Also() {\n  console.log('no dispatch here')\n}\n")

	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.IndexWails(context.Background())
	if result.Outcome == OutcomeUnsupported {
		t.Fatal("there are generated bindings here; the answer is unknown, not unsupported")
	}
	if result.Unknown != 2 {
		t.Fatalf("unknown = %d, want the two exports that could not be paired: %+v", result.Unknown, result)
	}
}

func TestIndexWailsSaysUnsupportedOnlyWithNoGeneratedBindings(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, root, "main.go", "package main\n\nfunc main() {}\n")

	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.IndexWails(context.Background())
	if result.Outcome != OutcomeUnsupported || result.Unknown != 0 {
		t.Fatalf("a project with no Wails must say so plainly: %+v", result)
	}
}

// A path that is not there is never "nothing is connected to it". The whole
// lookup is a string key into an index, so a typo, a wrong case on Windows, a
// directory or a moved file all come back as silence indistinguishable from a
// file nobody references — the one direction a reader cannot check.
func TestTraceRefusesAStartPathThatIsNotThere(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	// A directory is not a file, and a missing between-target is reported too.
	for _, request := range []TraceRequest{
		{Path: "desktop/missing.go", Name: "GetRepoMapGraph", Direction: DirectionCallees},
		{Path: "desktop/frontend", Name: "GetRepoMapGraph", Direction: DirectionCallees},
		{
			Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
			Direction: DirectionBetween, TargetPath: "desktop/nowhere.go",
		},
	} {
		result := indexer.Trace(context.Background(), request)
		if result.Outcome != OutcomeUnavailable {
			t.Errorf("%+v: outcome = %s, want unavailable", request, result.Outcome)
		}
		if len(result.Facts) != 0 || result.Reason == "" {
			t.Errorf("%+v: an unavailable path must say which one: %+v", request, result)
		}
	}

	// A path the machine accepts under another spelling: Windows resolves it,
	// so it must never come back as "this file has no connections". Either the
	// walk lands on the same file — the index is keyed by the spelling the
	// walk found — or the answer is that there is no such file. Silence is the
	// one thing it may not be.
	right := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionCallees, Depth: 2,
	})
	wrong := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionCallees, Depth: 2,
	})
	if wrong.Outcome == OutcomeUnavailable {
		return
	}
	if len(right.Facts) == 0 || !reflect.DeepEqual(right.Facts, wrong.Facts) {
		t.Errorf("a differently-spelled path gave a different answer:\nright = %+v\nwrong = %+v", right, wrong)
	}
}

// A depth the budget cut is a clipped answer, and the result carries the depth
// that was actually walked: a caller printing the number it asked for over a
// shorter walk is the same lie as a cap that does not say it bit.
func TestTraceSaysWhenTheDepthWasClamped(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionCallees, Depth: DefaultBudget().MaxDepth + 4,
	})
	if !result.Truncated || result.Reason != "depth budget reached" {
		t.Fatalf("a clamped depth must say so: %+v", result)
	}
	if result.Depth != DefaultBudget().MaxDepth {
		t.Fatalf("Depth = %d, want the depth actually walked", result.Depth)
	}

	full := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionCallees, Depth: 1,
	})
	if full.Truncated || full.Depth != 1 {
		t.Fatalf("a depth inside the budget is not a clip: %+v", full)
	}
}

// targetName on a structural hop: the import names a file and no symbol, so
// arriving at that file IS arriving, and the answer must not blame the depth
// for a rule that could never match.
func TestTraceBetweenReachesATargetFileAcrossAnImport(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "lib/util.ts", "export function work(): void {}\n")
	writeFile(t, root, "app.ts", "import { work } from './lib/util'\n")
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "app.ts", Name: "work", Direction: DirectionBetween,
		TargetPath: "lib/util.ts", TargetName: "work", Depth: 3,
	})
	if result.Outcome != OutcomeComplete || len(result.Facts) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Facts[0].Relation != RelationImport {
		t.Fatalf("path = %+v", result.Facts)
	}
}

// A generated module that names no binding is passed over in silence, and the
// number in front of the model stays a statement about something being wrong.
func TestIndexWailsIgnoresModulesThatNameNoBinding(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	writeFile(t, root, "desktop/frontend/wailsjs/go/models.ts", "export namespace engine {\n  export class Status {}\n}\n")
	writeFile(t, root, "desktop/frontend/src/Types.ts",
		"import { engine } from '../wailsjs/go/models'\n\nexport type S = engine.Status\n")

	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.IndexWails(context.Background())
	if result.Outcome != OutcomeComplete || result.Unknown != 0 {
		t.Fatalf("a type-only module is not an unresolved relationship: %+v", result)
	}
}

// A generator that emits arrow functions is a shape this adapter cannot read,
// and that is unknown — never a project with nothing to say.
func TestIndexWailsCountsAnExportShapeItCannotRead(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	writeFile(t, root, "desktop/frontend/wailsjs/go/main/Arrow.js",
		"export const Arrow = (a) => window['go']['main']['App']['Arrow'](a)\n")

	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.IndexWails(context.Background())
	if result.Outcome == OutcomeUnsupported || result.Unknown == 0 {
		t.Fatalf("an unreadable generated shape must be counted: %+v", result)
	}
}

// A doc comment is not a call. A hop stamped resolved has to be a line somebody
// could point at.
func TestCallSitesSkipCommentLines(t *testing.T) {
	source := "import { Run } from '../wailsjs/go/main/App'\n// Run is what this pane needs\nconst x = 1\nRun(2)\n"
	lines := findCallLines(source, "Run", 1)
	if len(lines) != 1 || lines[0] != 4 {
		t.Fatalf("call lines = %v, want only the real call on line 4", lines)
	}
}

// The second question about an unchanged project must not rebuild anything.
// This is the difference between a tool and a demo: on Aetox itself the first
// version spent the whole query budget indexing and then answered from an
// expired clock, so every trace returned "time budget reached" with no hops.
func TestSecondTraceOnAnUnchangedProjectDoesNoIndexWork(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	request := TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionBetween, TargetPath: "desktop/app.go", Depth: 3,
	}
	first := indexer.Trace(context.Background(), request)
	if first.Outcome != OutcomeComplete || len(first.Facts) != 2 {
		t.Fatalf("first trace = %+v", first)
	}
	second := indexer.Trace(context.Background(), request)
	if second.Outcome != OutcomeComplete || len(second.Facts) != 2 {
		t.Fatalf("second trace = %+v", second)
	}
	if second.Stale != 0 {
		t.Fatalf("an untouched project reported stale rows: %+v", second)
	}
	if second.Sources != 0 {
		t.Errorf("the second trace walked the project again (Sources = %d)", second.Sources)
	}
	if second.Rebuilt {
		t.Error("the second trace rebuilt an index nothing had invalidated")
	}
	if !first.Rebuilt {
		t.Error("the first trace answered from an index it never built")
	}
}

// A file that changed since the index is rebuilt, not answered around: the row
// whose file moved on is exactly the row that would have been the answer.
func TestTraceRebuildsWhenAFileChangedUnderIt(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	request := TraceRequest{
		Path: "desktop/frontend/wailsjs/go/main/App.js", Name: "GetRepoMapGraph",
		Direction: DirectionCallees, Depth: 1,
	}
	if got := indexer.Trace(context.Background(), request); got.Outcome != OutcomeComplete {
		t.Fatalf("first trace = %+v", got)
	}

	// The Go method moves three lines down; the cached hop points at the old
	// line, so it stops matching and the index has to be rebuilt.
	writeFile(t, root, "desktop/app.go", "package main\n\ntype App struct{}\n\n// moved\n// moved\n// moved\nfunc (a *App) GetRepoMapGraph(max int) {}\n")

	after := indexer.Trace(context.Background(), request)
	if after.Outcome != OutcomeComplete || len(after.Facts) != 1 {
		t.Fatalf("after the change = %+v", after)
	}
	if after.Facts[0].To.Line != 8 {
		t.Fatalf("the hop still points at the old line: %+v", after.Facts[0])
	}
}

// The note about file-level hops describes the hops being returned. A
// `between` walk looks at every edge it passes and prints only one path; a
// count taken while walking put "61 of these hop(s)" over an answer of three.
func TestFileLevelHopsCountsWhatIsReturned(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionBetween, TargetPath: "desktop/app.go", Depth: 3,
	})
	if result.Outcome != OutcomeComplete || len(result.Facts) != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.FileLevelHops != 0 {
		t.Fatalf("both hops here name the symbol at both ends, got %d", result.FileLevelHops)
	}

	// An import hop has no name at its destination, and is counted.
	writeFile(t, root, "lib.ts", "export const a = 1\n")
	writeFile(t, root, "uses.ts", "import { a } from './lib'\n")
	imports := indexer.Trace(context.Background(), TraceRequest{
		Path: "uses.ts", Name: "a", Direction: DirectionCallees, Depth: 1,
	})
	if imports.FileLevelHops != 1 {
		t.Fatalf("an import hop is file-level: %+v", imports)
	}
}

// A file created after the index has no rows and no stamp, which is exactly
// what a file with no relationships looks like. Answering "nothing is
// connected to this" about a file nobody has looked at is the same wrong
// answer the missing-path check exists to prevent, one step further out.
func TestTraceIndexesAFileCreatedAfterTheIndex(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "lib.ts", "export const a = 1\n")
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	if got := indexer.Trace(context.Background(), TraceRequest{
		Path: "lib.ts", Name: "a", Direction: DirectionCallees,
	}); got.Outcome != OutcomeComplete {
		t.Fatalf("first trace = %+v", got)
	}

	writeFile(t, root, "uses.ts", "import { a } from './lib'\n")
	fresh := indexer.Trace(context.Background(), TraceRequest{
		Path: "uses.ts", Name: "a", Direction: DirectionCallees, Depth: 1,
	})
	if fresh.Outcome != OutcomeComplete || len(fresh.Facts) != 1 {
		t.Fatalf("a file created after the index was answered from an index that never saw it: %+v", fresh)
	}
	if !fresh.Rebuilt {
		t.Error("the index had to be rebuilt to answer this, and the result does not say so")
	}
}

// A file this index does not read — a binary, a huge generated bundle, a type
// it does not parse — says so, rather than answering with silence.
func TestTraceSaysWhenAFileIsNotOneTheIndexReads(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "notes.txt", "not source anybody indexes\n")
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "notes.txt", Name: "anything", Direction: DirectionCallees,
	})
	if result.Outcome != OutcomeUnsupported {
		t.Fatalf("result = %+v", result)
	}
	if result.Reason == "" {
		t.Error("an unsupported file must say which one and why")
	}
}

// The default depth is a fact about the traversal and lives with the traversal.
// It was in the tool for one afternoon, and the first direct call to the API
// proved the split wrong: `between` with the tool's default answered a
// three-hop path, and the same request through the API answered one hop and
// called it a miss.
func TestBetweenDefaultsToEnoughHopsWithoutTheCallerAsking(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionBetween, TargetPath: "desktop/app.go",
	})
	if result.Outcome != OutcomeComplete || len(result.Facts) != 2 {
		t.Fatalf("between with no depth given = %+v", result)
	}
	if result.Depth != len(result.Facts) {
		t.Fatalf("a found path reports the hops of the path: depth=%d, facts=%d", result.Depth, len(result.Facts))
	}

	// A fan-out also defaults without being told, and the default has to be
	// deep enough to cross: the relationship this act exists for is the call,
	// the binding, and the method, and a default ending inside the binding
	// answers what `symbol` already answers. The default is the budget's own
	// ceiling, so a project that generates one layer has a level left over —
	// and the result reports the levels it entered, which is three here over a
	// path of two hops.
	fanOut := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionCallees,
	})
	if fanOut.Depth != 3 {
		t.Fatalf("a fan-out reports the levels it entered, got %d", fanOut.Depth)
	}
	if len(fanOut.Facts) != 2 {
		t.Fatalf("and it reaches the Go method: %+v", fanOut.Facts)
	}
	if fanOut.Facts[1].Relation != RelationWailsRPC {
		t.Fatalf("the second hop is the generated binding: %+v", fanOut.Facts)
	}
}

// A binding call that is commented out is not a relationship, and neither is an
// import written inside a comment. Aetox's own frontend keeps one of each, and
// counting them put a permanent "1 relationship could not be established" in
// front of the model on a project where nothing was wrong.
func TestCommentedOutImportsAndCallsAreNotRelationships(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	writeFile(t, root, "desktop/frontend/src/Commented.ts", `//   import { GetRepoMapGraph } from '../wailsjs/go/main/App'
// import { NoSuchBinding } from '../wailsjs/go/main/App'

export function keep(): void {
  // GetRepoMapGraph(-1)
}
`)
	writeFile(t, root, "desktop/frontend/src/Trailing.svelte", `<script lang="ts">
  import { GetRepoMapGraph } from '../wailsjs/go/main/App'
  const note = 1 // GetRepoMapGraph(2)
  GetRepoMapGraph(3)
</script>
`)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.IndexWails(context.Background())
	if result.Unknown != 0 {
		t.Fatalf("a commented-out mention counted as a relationship: %+v", result)
	}
	if result.Outcome != OutcomeComplete {
		t.Fatalf("outcome = %s, want complete on a project with nothing wrong", result.Outcome)
	}

	facts, _, err := indexer.store.FactsFrom("desktop/frontend/src/Trailing.svelte")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts[0].From.Line != 4 {
		t.Fatalf("the real call is the only hop: %+v", facts)
	}
	none, _, err := indexer.store.FactsFrom("desktop/frontend/src/Commented.ts")
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("a file whose only mentions are comments has no hops: %+v", none)
	}
}

// A file's facts are written per line, and a walk that arrives at one function in
// a generated file must follow the edges written AT that function. Without it,
// arriving anywhere in `desktop/forwarders.go` answered with the other
// forwarder's edge as well — one hop that answers the question, buried among
// every other function's edges, and on a real generated forwarder 101 of them.
func TestWalkFollowsTheEdgesWrittenAtTheLineItArrivedOn(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedForwarderProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "Alpha",
		Direction: DirectionCallees, Depth: 3,
	})
	if result.Outcome != OutcomeComplete || len(result.Facts) != 3 {
		t.Fatalf("result = %+v", result)
	}
	last := result.Facts[len(result.Facts)-1]
	if last.To.Path != "lib/a.go" || last.From.Line != 7 {
		t.Fatalf("the third hop is the edge written at line 7: %+v", result.Facts)
	}
	for _, fact := range result.Facts {
		if fact.To.Path == "lib/b.go" {
			t.Fatalf("the other forwarder's edge was followed: %+v", result.Facts)
		}
	}
}

// The same rule read the other way. Arriving at a file through a file-level hop,
// the walk is still asking about the symbol it started from and not about the
// file: otherwise "who reaches this method" becomes "who reaches this file",
// which on a generated forwarder is every binding in the dispatch.
func TestWalkKeepsTheNameItIsFollowingAcrossAFileLevelHop(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedForwarderProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "lib/a.go", Name: "Alpha",
		Direction: DirectionCallers, Depth: 2,
	})
	if result.Outcome != OutcomeComplete || len(result.Facts) != 2 {
		t.Fatalf("result = %+v", result)
	}
	second := result.Facts[1]
	if second.From.Path != "desktop/frontend/wailsjs/go/main/App.js" || second.From.Name != "Alpha" {
		t.Fatalf("the second hop is the binding that calls this method: %+v", result.Facts)
	}
	for _, fact := range result.Facts {
		if fact.From.Name == "Beta" {
			t.Fatalf("another binding was read as reaching this symbol: %+v", result.Facts)
		}
	}
}

// A cache written by an older analyzer is not this build's answer. Nothing in a
// fact row records which code decided it existed — only the content stamps of
// its two endpoints — so the version kept beside a producer's rows is the whole
// of the difference between a reading taken by this build and one taken by the
// build before a change.
func TestTraceRebuildsWhenTheAnalyzerVersionChanged(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	seedWailsProject(t, root)
	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	request := TraceRequest{
		Path: "desktop/frontend/src/Pane.svelte", Name: "GetRepoMapGraph",
		Direction: DirectionCallees, Depth: 3,
	}
	if first := indexer.Trace(context.Background(), request); len(first.Facts) == 0 {
		t.Fatalf("the seeded project must answer before anything changes: %+v", first)
	}
	// The same rows, marked as written by a build other than this one: what an
	// upgraded analyzer finds in a cache it wrote yesterday.
	if err := indexer.store.MarkIndexed(structuralProducer, "0", 0); err != nil {
		t.Fatal(err)
	}
	rebuilt := indexer.Trace(context.Background(), request)
	if !rebuilt.Rebuilt {
		t.Fatalf("an older reading must be rebuilt rather than answered from: %+v", rebuilt)
	}
	if len(rebuilt.Facts) == 0 {
		t.Fatalf("the rebuilt answer is empty: %+v", rebuilt)
	}
	// And the control, so that this is measuring the version and not the second
	// call: with this build's version on the rows again, the cache answers.
	settled := indexer.Trace(context.Background(), request)
	if settled.Rebuilt {
		t.Fatalf("an unchanged project answers from the cache: %+v", settled)
	}
}

// seedForwarderProject is a generated forwarder file with two functions in it,
// each reaching a different package, plus the frontend call that arrives at one
// of them. Two edges out of one file is the whole point: a walk that cannot say
// which one belongs to the function in hand follows both.
func seedForwarderProject(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, root, "lib/a.go", "package lib\n\ntype Alpha struct{}\n")
	writeFile(t, root, "lib/b.go", "package lib\n\ntype Beta struct{}\n")
	writeFile(t, root, "desktop/forwarders.go",
		"package main\n\nimport \"example.com/app/lib\"\n\ntype App struct{}\n\n"+
			"func (a *App) Alpha() lib.Alpha { return lib.Alpha{} }\n"+
			"func (a *App) Beta() lib.Beta { return lib.Beta{} }\n")
	writeFile(t, root, "desktop/frontend/wailsjs/go/main/App.js",
		"export function Alpha(arg1) {\n  return window['go']['main']['App']['Alpha'](arg1);\n}\n"+
			"export function Beta(arg1) {\n  return window['go']['main']['App']['Beta'](arg1);\n}\n")
	writeFile(t, root, "desktop/frontend/src/Pane.svelte", `<script lang="ts">
  import { Alpha } from '../wailsjs/go/main/App'

  async function load(): Promise<void> {
    await Alpha(-1)
  }
</script>
`)
}

// A walk clipped by the fact budget still counts what it is returning, because
// that count is what produces "N of these hop(s) are file-level". A clipped
// answer is the one most likely to be nothing but file-level hops — on Aetox's
// own tree a name the boundary adapter cannot pair returns 120 import edges of
// one generated file — and reporting zero there drops the note that separates
// "these files are connected" from "this name reaches that one" exactly when
// the reader needs it.
func TestAClippedWalkStillCountsItsFileLevelHops(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "app.ts", "import { a } from './a'\nimport { b } from './b'\nimport { c } from './c'\n")
	writeFile(t, root, "a.ts", "export function a(): void {}\n")
	writeFile(t, root, "b.ts", "export function b(): void {}\n")
	writeFile(t, root, "c.ts", "export function c(): void {}\n")

	indexer, err := NewIndexer(root, Budget{MaxFacts: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "app.ts", Name: "a", Direction: DirectionCallees,
	})
	if !result.Truncated || result.Reason != "fact budget reached" {
		t.Fatalf("the fixture must be clipped by the fact budget: %+v", result)
	}
	if len(result.Facts) != 2 {
		t.Fatalf("facts = %d, want the budget the walk was given", len(result.Facts))
	}
	if result.FileLevelHops != len(result.Facts) {
		t.Fatalf("FileLevelHops = %d over %d file-level hops: the count is what says they are not about the name",
			result.FileLevelHops, len(result.Facts))
	}
}

// The depth in the answer is the levels the walk entered, not the ones it was
// allowed. A file with nothing behind it ends the walk at the first level, and
// a header stating three hops over a walk of one is the same lie as a cap that
// does not say it bit.
func TestTraceReportsTheLevelsItActuallyEntered(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "alone.go", "package demo\n\nfunc Alone() {}\n")

	indexer, err := NewIndexer(root, DefaultBudget())
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()

	result := indexer.Trace(context.Background(), TraceRequest{
		Path: "alone.go", Name: "Alone", Direction: DirectionCallees,
	})
	if len(result.Facts) != 0 {
		t.Fatalf("the fixture has no edge to follow: %+v", result.Facts)
	}
	if result.Depth != 1 {
		t.Fatalf("Depth = %d, want the one level the walk entered", result.Depth)
	}
}
