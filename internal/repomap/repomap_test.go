package repomap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A little repo with a clear center of gravity: two Go files import pkg/core,
// one Svelte file imports lib/util, and one markdown file has headings. The
// map must put the referenced files first and show real symbols with line
// numbers.
func seedRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/demo\n\ngo 1.22\n")
	write("pkg/core/core.go", `package core

// Answer is the load-bearing function.
func Answer() int { return 42 }

type Engine struct {
	Ready bool
}
`)
	write("a.go", "package main\n\nimport \"example.com/demo/pkg/core\"\n\nfunc main() { core.Answer() }\n")
	write("b.go", "package main\n\nimport \"example.com/demo/pkg/core\"\n\nfunc helper() int { return core.Answer() }\n")
	write("lib/util.ts", "export function slugify(s: string): string { return s }\nexport const VERSION = () => 3\n")
	write("App.svelte", "<script>\nimport { slugify } from './lib/util'\n</script>\n")
	write("docs/GUIDE.md", "# Guide\n\nprose\n\n## Install\n\n```sh\n# not a heading\n```\n")
	return root
}

func TestBuildRanksReferencedFilesFirst(t *testing.T) {
	out, err := Build(context.Background(), Options{Root: seedRepo(t)})
	if err != nil {
		t.Fatal(err)
	}
	core := strings.Index(out, "pkg/core/core.go")
	util := strings.Index(out, "lib/util.ts")
	app := strings.Index(out, "App.svelte")
	if core < 0 || util < 0 {
		t.Fatalf("referenced files missing from map:\n%s", out)
	}
	// core.go is imported by two files, util.ts by one, App.svelte by none —
	// the render order must say so.
	if !(core < util) {
		t.Errorf("core.go (2 refs) should rank above util.ts (1 ref):\n%s", out)
	}
	if app >= 0 && app < util {
		t.Errorf("App.svelte (0 refs) should not outrank util.ts:\n%s", out)
	}
	if !strings.Contains(out, "(referenced by 2)") {
		t.Errorf("core.go's two importers should be counted:\n%s", out)
	}
}

func TestBuildShowsRealSymbolsWithLines(t *testing.T) {
	out, err := Build(context.Background(), Options{Root: seedRepo(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"func Answer() int",       // Go func, brace trimmed
		"type Engine struct",      // Go type
		"export function slugify", // ts function
		"## Install",              // md heading outside the fence
	} {
		if !strings.Contains(out, want) {
			t.Errorf("map is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "not a heading") {
		t.Errorf("a # inside a code fence was mapped as a heading:\n%s", out)
	}
	// Answer is defined on line 4 of core.go and the map must say which line,
	// or "read only the range you need" has nothing to aim at.
	if !strings.Contains(out, "4: func Answer() int") {
		t.Errorf("symbol lines should carry their line number:\n%s", out)
	}
}

func TestBuildRespectsTheBudget(t *testing.T) {
	root := seedRepo(t)
	out, err := Build(context.Background(), Options{Root: root, Budget: 30})
	if err != nil {
		t.Fatal(err)
	}
	// 30 tokens ≈ 120 chars: room for the header and roughly one file. The
	// contract is whole files within roughly one file-block of the line, never
	// the whole map regardless.
	if len(out) > 30*4+400 {
		t.Errorf("a 30-token budget rendered %d chars:\n%s", len(out), out)
	}
	if !strings.Contains(out, "more files under the budget line") {
		t.Errorf("a cut map must say what it cut:\n%s", out)
	}
	full, err := Build(context.Background(), Options{Root: root, Budget: 100000})
	if err != nil {
		t.Fatal(err)
	}
	if len(full) <= len(out) {
		t.Errorf("a bigger budget should render a bigger map (%d vs %d)", len(full), len(out))
	}
}

func TestBuildSkipsIgnoredAndDotDirs(t *testing.T) {
	root := seedRepo(t)
	for _, rel := range []string{"node_modules/big/dep.ts", ".git/objects/junk.md"} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("export function hidden() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := Build(context.Background(), Options{Root: root, Ignore: map[string]bool{"node_modules": true}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "hidden") || strings.Contains(out, "node_modules") || strings.Contains(out, ".git") {
		t.Errorf("ignored directories leaked into the map:\n%s", out)
	}
}

func TestBuildRefusesAMissingRoot(t *testing.T) {
	if _, err := Build(context.Background(), Options{Root: filepath.Join(t.TempDir(), "nope")}); err == nil {
		t.Fatal("a root that does not exist must be an error, not an empty map")
	}
}

// The declRules languages, one probe each: a map line with the right symbol
// proves the family's scanner sees declarations; this is deliberately not a
// parser test suite — a line scan owns no grammar to pin.
func TestBuildScansTheMainstreamLanguages(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// One folder each, or the per-directory breadth cap (maxFilesPerDir) would
	// cut the probe list before the scanners it exists to exercise ever ran.
	write("jvm/A.java", "public class Invoice {\n    public void settle() {}\n}\n")
	write("rust/lib.rs", "pub fn parse_all(input: &str) -> Vec<Token> {\n    todo!()\n}\npub struct Token {}\n")
	write("php/app.php", "<?php\nclass Router {\n    public function dispatch($req) {}\n}\n")
	write("ruby/worker.rb", "class Worker\n  def perform(job)\n  end\nend\n")
	write("swift/View.swift", "public struct ProfileView {\n    func render() {}\n}\n")
	write("c/core.c", "struct buffer { int n; };\nint flush_buffer(struct buffer *b)\n{\n    return 0;\n}\n")

	out, err := Build(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"public class Invoice",
		"pub fn parse_all",
		"class Router",
		"def perform",
		"public struct ProfileView",
		"int flush_buffer",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("map is missing %q:\n%s", want, out)
		}
	}
}

// Python's import graph: an absolute import resolves through the suffix walk
// (the package root lives under backend/, as it does in real repos), and the
// imported module outranks its importer.
func TestBuildResolvesPythonImports(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("backend/app/db/gate.py", "def open_gate(cfg):\n    pass\n")
	write("backend/app/api/routes.py", "from app.db.gate import open_gate\n\ndef register(app):\n    pass\n")
	write("backend/app/api/admin.py", "from app.db.gate import open_gate\n\ndef admin(app):\n    pass\n")
	write("backend/app/api/util.py", "from .routes import register\n\ndef helper():\n    pass\n")

	out, err := Build(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	gate := strings.Index(out, "backend/app/db/gate.py")
	routes := strings.Index(out, "backend/app/api/routes.py")
	if gate < 0 {
		t.Fatalf("gate.py missing from map:\n%s", out)
	}
	if !strings.Contains(out, "backend/app/db/gate.py  (referenced by 2)") {
		t.Errorf("absolute imports should credit gate.py twice:\n%s", out)
	}
	if routes >= 0 && routes < gate {
		t.Errorf("gate.py (2 refs) should outrank routes.py (1 ref):\n%s", out)
	}
}

// The graph face: same ranking as the text map, real edges between kept
// nodes, and a Go package-level import landing on one spokesperson file
// rather than fanning out to lines nobody wrote.
func TestGraphSharesRankingAndResolvesEdges(t *testing.T) {
	nodes, edges, total, err := Graph(context.Background(), Options{Root: seedRepo(t)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total < 6 {
		t.Fatalf("total = %d, want every parsed file counted", total)
	}
	if len(nodes) == 0 || nodes[0].Path != "pkg/core/core.go" {
		t.Fatalf("the most-referenced file must lead the graph as it leads the map, got %+v", nodes)
	}
	find := func(path string) int {
		for i, n := range nodes {
			if n.Path == path {
				return i
			}
		}
		return -1
	}
	core, app, util := find("pkg/core/core.go"), find("App.svelte"), find("lib/util.ts")
	if core < 0 || app < 0 || util < 0 {
		t.Fatalf("expected nodes missing: %+v", nodes)
	}
	has := func(from, to int) bool {
		for _, e := range edges {
			if e.From == from && e.To == to {
				return true
			}
		}
		return false
	}
	if !has(find("a.go"), core) || !has(find("b.go"), core) {
		t.Errorf("Go imports should land on the package spokesperson: %+v", edges)
	}
	if !has(app, util) {
		t.Errorf("the svelte import of lib/util should be an edge: %+v", edges)
	}
	for _, e := range edges {
		if e.From == e.To {
			t.Errorf("self edge slipped through: %+v", e)
		}
	}
}

// The aliased imports every Next/Vite project actually writes: `@/lib/db`
// resolves by suffix, same walk as Python absolutes — without it a frontend
// draws as dots with no lines, which is how the gap was found.
func TestGraphEdgesCarrySourceEvidence(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("lib/util.ts", "export function work(): void {}\n")
	write("app.ts", "// heading\nimport { work } from './lib/util'\nwork()\n")

	nodes, edges, _, err := Graph(context.Background(), Options{Root: root}, AllNodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Fatalf("edges = %+v, want one import", edges)
	}
	edge := edges[0]
	if nodes[edge.From].Path != "app.ts" || nodes[edge.To].Path != "lib/util.ts" {
		t.Fatalf("edge endpoints = %+v through %+v", edge, nodes)
	}
	if edge.Kind != "import" || edge.Strength != "syntactic" || edge.EvidenceLine != 2 || edge.Producer != "repomap" {
		t.Fatalf("edge evidence = %+v", edge)
	}

	facts, files, capped, err := Facts(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 || facts[0].From.Path != "app.ts" || facts[0].From.Line != 2 || facts[0].To.Path != "lib/util.ts" {
		t.Fatalf("facts = %+v", facts)
	}
	// The target anchor is the file, not a line nobody read.
	if facts[0].To.Line != 0 {
		t.Errorf("an import proves the importer's line, not the target's: %+v", facts[0].To)
	}
	if capped || len(files) != 2 {
		t.Errorf("files = %v capped = %v, want both files the walk saw", files, capped)
	}
}

// One file that writes two references to the same target is ONE file depending
// on it. Counting written references instead would let a file naming two
// symbols of one package outrank a file the project genuinely leans on — and
// it would move every number and the whole order of the map, which is a
// display tool the project is measuring the adoption of.
func TestRefsCountImportingFilesNotWrittenReferences(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("lib/util.ts", "export const a = 1\nexport const b = 2\n")
	write("app.ts", "import { a } from './lib/util'\nimport { b } from './lib/util'\n")
	write("other.ts", "import { a } from './lib/util'\n")

	nodes, _, _, err := Graph(context.Background(), Options{Root: root}, AllNodes)
	if err != nil {
		t.Fatal(err)
	}
	refs := make(map[string]int)
	for _, n := range nodes {
		refs[n.Path] = n.Refs
	}
	if refs["lib/util.ts"] != 2 {
		t.Errorf("util.ts has 2 importing files, not 3 written references: %+v", nodes)
	}
	// The evidence list keeps every written reference, which is the half that
	// is meant to: two lines in app.ts are two pieces of evidence.
	facts, _, _, err := Facts(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 3 {
		t.Errorf("facts = %+v, want all three written references", facts)
	}
}

// Strength is decided where the evidence is, not where the syntax was read: a
// `dir#Name` target that resolved to a file declaring the name is resolved,
// while a package directory is the map's own convention and says so.
func TestStructuralStrengthSaysHowTheTargetWasEstablished(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/demo\n\ngo 1.22\n")
	write("pkg/core/core.go", "package core\n\ntype Request struct{}\n")
	write("main.go", "package main\n\nimport \"example.com/demo/pkg/core\"\n\nfunc main() { var _ core.Request }\n")
	write("blank.go", "package main\n\nimport _ \"example.com/demo/pkg/core\"\n")

	facts, _, _, err := Facts(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	byFrom := make(map[string]Fact)
	for _, fact := range facts {
		byFrom[fact.From.Path] = fact
	}
	if got := byFrom["main.go"].Strength; got != "resolved" {
		t.Errorf("a selector that resolved to the declaring file is resolved, got %q", got)
	}
	if got := byFrom["blank.go"].Strength; got != "heuristic" {
		t.Errorf("a blank import landing on a package directory is a convention, got %q", got)
	}
}

func TestBuildResolvesAliasImports(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("frontend/src/lib/db.ts", "export function connect(): void {}\n")
	write("frontend/src/app/page.tsx", "import { connect } from '@/lib/db'\n\nexport default function Page() { return null }\n")
	write("frontend/src/app/admin.tsx", "import { connect } from '@/lib/db'\n\nexport default function Admin() { return null }\n")

	out, err := Build(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "frontend/src/lib/db.ts  (referenced by 2)") {
		t.Errorf("@/ imports should credit db.ts twice:\n%s", out)
	}
}

// A Go import names a package, and a package is several files: the credit
// has to land on the file whose name the importer actually wrote, or the
// map ranks a package's biggest file over its most-used one — on this
// repository that was openai_compatible.go over types.go, n8n_tools.go over
// skill.go, for a year. The bare package target is spent only when no
// selector resolved (a blank import).
func TestBuildCreditsTheGoFileWhoseSymbolIsUsed(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/demo\n\ngo 1.22\n")
	write("pkg/core/types.go", "package core\n\ntype Request struct{}\n")
	// The bigger file of the package, which the old tie-break would have
	// chosen to speak for it.
	write("pkg/core/big.go", "package core\n\ntype A struct{}\ntype B struct{}\ntype C struct{}\nfunc Big() {}\n")
	write("pkg/util/hide_windows.go", "package util\n\nfunc Hide() {}\n")
	write("pkg/util/hide_other.go", "package util\n\nfunc Hide() {}\n")
	write("a.go", "package main\n\nimport \"example.com/demo/pkg/core\"\n\nfunc main() { var _ core.Request }\n")
	write("b.go", "package main\n\nimport \"example.com/demo/pkg/core\"\n\nfunc b() { var _ core.Request }\n")
	write("c.go", "package main\n\nimport _ \"example.com/demo/pkg/core\"\n")
	write("d.go", "package main\n\nimport \"example.com/demo/pkg/util\"\n\nfunc d() { util.Hide() }\n")

	nodes, edges, _, err := Graph(context.Background(), Options{Root: root}, AllNodes)
	if err != nil {
		t.Fatal(err)
	}
	refs := make(map[string]int)
	index := make(map[string]int)
	for i, n := range nodes {
		refs[n.Path] = n.Refs
		index[n.Path] = i
	}
	// a.go and b.go use core.Request → types.go; c.go's blank import falls
	// back to the package and credits both files once.
	if refs["pkg/core/types.go"] != 3 {
		t.Errorf("types.go: got %d refs, want 3 (two selectors + one blank-import fallback): %+v", refs["pkg/core/types.go"], nodes)
	}
	if refs["pkg/core/big.go"] != 1 {
		t.Errorf("big.go: got %d refs, want 1 (only the blank-import fallback): %+v", refs["pkg/core/big.go"], nodes)
	}
	if nodes[0].Path != "pkg/core/types.go" {
		t.Errorf("the used file must lead the map, not the biggest: %+v", nodes[:2])
	}
	// A build-tagged pair declaring the same name: each earns the reference.
	if refs["pkg/util/hide_windows.go"] != 1 || refs["pkg/util/hide_other.go"] != 1 {
		t.Errorf("both declarers of Hide should be credited: %+v", nodes)
	}
	has := func(from, to string) bool {
		for _, e := range edges {
			if e.From == index[from] && e.To == index[to] {
				return true
			}
		}
		return false
	}
	if !has("a.go", "pkg/core/types.go") {
		t.Errorf("the graph's line should land on the file declaring the name: %+v", edges)
	}
	if has("a.go", "pkg/core/big.go") {
		t.Errorf("a.go never used big.go; no line should be drawn to it: %+v", edges)
	}
}

// Test files import what they test and what mocks it, which is not what the
// product leans on: the first real frontend this walk met ranked
// test/mocks/wailsApp.ts second in the project on 122 test importers.
func TestBuildDoesNotCountTestImporters(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/demo\n\ngo 1.22\n")
	write("lib/store.ts", "export function load() {}\n")
	write("test/mocks/app.ts", "export const mock = () => 1\n")
	write("test/store.test.ts", "import { load } from '../lib/store'\nimport { mock } from './mocks/app'\n")
	write("test/other.test.ts", "import { mock } from './mocks/app'\n")
	write("App.svelte", "<script>\nimport { load } from './lib/store'\n</script>\n")
	write("pkg/core/core.go", "package core\n\nfunc Answer() int { return 42 }\n")
	write("pkg/core/core_test.go", "package core_test\n\nimport \"example.com/demo/pkg/core\"\n\nfunc use() { core.Answer() }\n")

	out, err := Build(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "lib/store.ts  (referenced by 1)") {
		t.Errorf("store.ts has one product importer (App.svelte); the test import must not count:\n%s", out)
	}
	if strings.Contains(out, "test/mocks/app.ts  (referenced by") {
		t.Errorf("a mock imported only by tests has no incoming references:\n%s", out)
	}
	if strings.Contains(out, "pkg/core/core.go  (referenced by") {
		t.Errorf("a _test.go importer must not credit the package:\n%s", out)
	}
	// The graph still draws the test's imports — a picture of what imports
	// what is a different question from what the product leans on.
	nodes, edges, _, err := Graph(context.Background(), Options{Root: root}, AllNodes)
	if err != nil {
		t.Fatal(err)
	}
	index := make(map[string]int)
	for i, n := range nodes {
		index[n.Path] = i
	}
	drawn := false
	for _, e := range edges {
		if e.From == index["test/store.test.ts"] && e.To == index["lib/store.ts"] {
			drawn = true
		}
	}
	if !drawn {
		t.Errorf("the graph should still draw the test's import: %+v", edges)
	}
}

// A folder the repository's own .gitignore names is not the project: a
// build output, a virtualenv, a second checkout parked beside the first —
// this repository's map spent one of its ten files on a gitignored copy of
// itself, and the copy's imports inflated the original's counts.
func TestBuildSkipsGitignoredDirs(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "# comment\n/copy/\nvenv\n*.log\n!keep\nsrc/gen/\n")
	write("main.ts", "export function main() {}\n")
	write("copy/main.ts", "export function copied() {}\n")
	write("deep/venv/lib.py", "def venv_only(): pass\n")
	write("deep/copy/nested.ts", "export function nestedCopy() {}\n")
	write("src/gen/out.ts", "export function generated() {}\n")

	out, err := Build(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"copy/main.ts", "deep/venv/lib.py"} {
		if strings.Contains(out, gone) {
			t.Errorf("%s is gitignored and must not be mapped:\n%s", gone, out)
		}
	}
	// `/copy/` is anchored to the root: a nested folder of the same name is
	// not what the line says.
	if !strings.Contains(out, "deep/copy/nested.ts") {
		t.Errorf("a root-anchored pattern must not skip a nested folder of the same name:\n%s", out)
	}
	// A pattern with an inner slash is left to the tools that need a real
	// matcher; the walk acts only on directory names.
	if !strings.Contains(out, "src/gen/out.ts") {
		t.Errorf("src/gen/ is not a plain directory name and is not acted on:\n%s", out)
	}
}

// When mapping a subfolder of a Go module, repomap must discover the ancestor
// go.mod, trim the rootPrefix so imports within the subtree resolve cleanly,
// and drop edges that point outside the mapped subtree.
func TestSubfolderGraphResolvesGoImports(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/demo\n\ngo 1.22\n")
	write("pkg/core/core.go", "package core\n\ntype Request struct{}\n")
	write("pkg/app/app.go", "package app\n\nimport \"example.com/demo/pkg/core\"\n\nfunc Use() { var r core.Request; _ = r }\n")
	write("other/thing.go", "package other\n\nfunc Thing() {}\n")

	nodes, edges, total, err := Graph(context.Background(), Options{Root: filepath.Join(root, "pkg")}, AllNodes)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("expected 2 mapped files in pkg, got %d", total)
	}

	byPath := make(map[string]Node)
	index := make(map[string]int)
	for i, n := range nodes {
		byPath[n.Path] = n
		index[n.Path] = i
	}

	coreNode, ok := byPath["core/core.go"]
	if !ok {
		t.Fatalf("core/core.go missing from graph: %+v", nodes)
	}
	if coreNode.Refs != 1 {
		t.Errorf("core/core.go should have 1 incoming ref, got %d", coreNode.Refs)
	}

	if _, ok := byPath["other/thing.go"]; ok {
		t.Errorf("other/thing.go is outside pkg and must not be in graph")
	}

	hasEdge := false
	for _, e := range edges {
		if e.From == index["app/app.go"] && e.To == index["core/core.go"] {
			hasEdge = true
		}
	}
	if !hasEdge {
		t.Errorf("expected edge from app/app.go to core/core.go, got: %+v", edges)
	}

	out, err := Build(context.Background(), Options{Root: filepath.Join(root, "pkg")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "core/core.go  (referenced by 1)") {
		t.Errorf("Build output must show core/core.go with (referenced by 1):\n%s", out)
	}
	coreIdx := strings.Index(out, "core/core.go")
	appIdx := strings.Index(out, "app/app.go")
	if coreIdx < 0 || appIdx < 0 || coreIdx > appIdx {
		t.Errorf("core/core.go should rank above app/app.go:\n%s", out)
	}
}

func TestDeepSubfolderMapping(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/demo\n\ngo 1.22\n")
	write("internal/model/types.go", "package model\n\ntype Config struct{}\n")
	write("internal/model/sub/sub.go", "package sub\n\nimport \"example.com/demo/internal/model\"\n\nfunc Build() model.Config { return model.Config{} }\n")
	write("internal/turn/executor.go", "package turn\n\nimport \"example.com/demo/internal/model\"\n\nfunc Exec() model.Config { return model.Config{} }\n")

	nodes, edges, _, err := Graph(context.Background(), Options{Root: filepath.Join(root, "internal", "model")}, AllNodes)
	if err != nil {
		t.Fatal(err)
	}
	byPath := make(map[string]Node)
	index := make(map[string]int)
	for i, n := range nodes {
		byPath[n.Path] = n
		index[n.Path] = i
	}
	if n, ok := byPath["types.go"]; !ok || n.Refs != 1 {
		t.Errorf("types.go should have 1 ref from sub/sub.go, got node: %+v", n)
	}
	hasEdge := false
	for _, e := range edges {
		if e.From == index["sub/sub.go"] && e.To == index["types.go"] {
			hasEdge = true
		}
	}
	if !hasEdge {
		t.Errorf("expected edge from sub/sub.go to types.go, got: %+v", edges)
	}
}

func TestSubfolderGitignoreInheritance(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/demo\n\ngo 1.22\n")
	write(".gitignore", "venv\n/output\n")
	write("pkg/good.go", "package pkg\nfunc Good() {}\n")
	write("pkg/venv/bad.go", "package venv\nfunc Bad() {}\n")
	write("pkg/output/keep.go", "package output\nfunc Keep() {}\n")
	write("pkg/.gitignore", "/localout\n")
	write("pkg/localout/skip.go", "package localout\nfunc Skip() {}\n")

	out, err := Build(context.Background(), Options{Root: filepath.Join(root, "pkg")})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "venv/bad.go") {
		t.Errorf("pkg/venv should be ignored by ancestor .gitignore anywhere rule:\n%s", out)
	}
	if !strings.Contains(out, "output/keep.go") {
		t.Errorf("pkg/output should NOT be ignored because ancestor /output was atRoot only:\n%s", out)
	}
	if strings.Contains(out, "localout/skip.go") {
		t.Errorf("pkg/localout should be ignored by pkg/.gitignore atRoot rule:\n%s", out)
	}
}

