// Package repomap builds a token-budgeted outline of a project: every file
// ranked by how many other files reference it, each shown with its top-level
// symbols and the line they are defined on.
//
// The mechanism is Aider's (docs/aider-study/README.md §1) reduced to what a
// first version has to prove: parse → rank → fit a budget. It exists so the
// model can see the SHAPE of a repository for ~1k tokens instead of paying
// 30k to read one architecture file whole — the 7-day baseline this package
// was built against counted 18 reads of 50KB or more (BASELINE.md).
//
// Deliberately a library with no imports from internal/skill: the tool wrapper
// lives there and depends on this, never the other way round, so the parse →
// rank → fit core stays extractable on its own (README.md ระดับ 5). Everything
// host-specific — which directories are off limits, where the root is — comes
// in through Options.
//
// Parsing is stdlib go/parser for Go and line-level pattern scans for
// everything else, by choice: this repository is pure Go and tree-sitter's Go
// bindings are all cgo (EXECUTION.md ทางแยก ก/ข/ค). A scan that misses an
// exotic definition costs one line of the map; a cgo dependency costs the
// build everywhere, forever. If the measured win justifies richer parsing the
// upgrade happens behind Build's signature.
package repomap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultBudget is Aider's default map size, kept as-is on purpose: it is the
// one number in this design with field history behind it, and the point of the
// first version is to measure, not to tune.
const DefaultBudget = 1000

const (
	// maxFileBytes skips anything too big to be source someone maintains.
	// Generated bundles and vendored blobs are exactly the files whose symbols
	// would drown the map in unreferenced noise.
	maxFileBytes = 512 * 1024
	// maxFiles bounds the walk, not correctness: a tree that large is not going
	// to fit any budget anyway, and the render says when the cap was hit.
	maxFiles = 10000
	// maxSymbolChars keeps one pathological one-liner from spending the whole
	// budget. Long enough for any honest signature.
	maxSymbolChars = 120
)

// Options carries what the host decides and Build must not.
type Options struct {
	// Root is the absolute directory to map. The caller has already resolved
	// and authorized it — this package never widens a path.
	Root string
	// Budget is the approximate token ceiling for the rendered map,
	// DefaultBudget when zero. Approximate at 4 chars/token, the same rate the
	// tool-block budget uses, and for the same reason: the number is about
	// order of magnitude and must not depend on which model is loaded.
	Budget int
	// Ignore lists directory names never entered, on top of dot-dirs which are
	// always skipped. The skill layer passes its own list so "what the map
	// sees" and "what grep sees" cannot drift apart.
	Ignore map[string]bool
}

// Symbol is one definition worth a line in the map.
type Symbol struct {
	Line int
	Text string
	// Public marks what other files can see — exported in Go, `export`ed in a
	// script, top-two heading levels in markdown. When a file has more symbols
	// than its share of the map, these are the ones that survive the cut: the
	// map exists for cross-file navigation, and a private helper is exactly
	// the line another file never needs.
	Public bool
}

// maxSymbolsPerFile is each file's share of the map. Without it the first big
// file eats the whole budget: the first real run of this package rendered ONE
// file of 1,150 — 53 symbols of openai_compatible.go and nothing else — which
// is a table of contents for a book that lists one chapter's every paragraph.
// Breadth beats depth in a map; read opens the file that needs depth.
//
// Eight, down from a first guess of fifteen: at fifteen the default budget
// held three files of the measured 1,150, at eight it holds seven packages —
// and seven packages is a shape, three files is a keyhole.
const maxSymbolsPerFile = 8

// file is one mapped file with everything ranking needs.
type file struct {
	rel     string
	symbols []Symbol
	// defs are the exported names this file declares, Go only — what a
	// `dir#Name` target resolves against. Nil for every other language, whose
	// imports already name a file.
	defs []string
	// refs counts the OTHER, non-test files that reference this one. For Go
	// that is the files whose `pkg.Name` selectors land on a name declared
	// here; a file that imports the package and writes no resolvable selector
	// (a blank import, a name the scan missed) credits every file of the
	// package instead, the way the first version credited all of them.
	refs int
}

// rawEdge is one import as the walk saw it: which file wrote it, and the
// target key it wrote (a rel file path for script imports, a rel package
// directory or `dir#Name` for Go, a module path for Python). Kept raw because
// resolution needs the finished file list.
type rawEdge struct {
	from     string
	target   string
	line     int
	strength string
}

// link is one resolved reference: from a file to a file, or — when a Go
// import resolved to no symbol — from a file to a package directory, which
// rank spreads over the package and Graph draws to its spokesperson. One
// resolution serves both faces (resolve below), so the count on a node and
// the lines drawn to it can never come from two different readings of the
// same import.
// linkKey is what "one file references another" means: the pair, and whether
// the destination is a package directory rather than a file. Not the link
// itself — a link now carries the line and strength of ONE written reference,
// and two of them are still one dependency.
type linkKey struct {
	from string
	to   string
	pkg  bool
}

type link struct {
	from     string
	to       string
	line     int
	strength string
	// pkg marks `to` as a directory, not a file.
	pkg bool
}

// analysis is everything one walk of the tree learned, shared by the model's
// rendered map (Build) and the UI's node graph (Graph) so the two can never
// disagree about what the repository looks like — same ignore rules, same
// parsers, same ranking inputs, one walk.
type analysis struct {
	files  []*file
	byRel  map[string]*file
	edges  []rawEdge
	links  []link
	total  int
	capped bool
}

func analyze(ctx context.Context, opts Options) (*analysis, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("not a folder to map: %s", opts.Root)
	}

	goModule, rootPrefix, ignoredAnywhere, ignoredAtRoot := projectRoot(root)
	a := &analysis{byRel: make(map[string]*file)}

	stopped := false
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if ctx.Err() != nil {
			// A deadline mid-walk ends the walk, not the map: whatever parsed
			// by now renders, marked as partial. The first project this tool
			// met in the wild held a Python venv the ignore list did not know,
			// and the call ran 105 seconds — a map that arrives late enough is
			// indistinguishable from a tool that hangs, and the model's answer
			// to that was to call it again and pay twice.
			stopped = true
			return filepath.SkipAll
		}
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || opts.Ignore[name] || ignoredAnywhere[name] {
				return filepath.SkipDir
			}
			if ignoredAtRoot[name] && filepath.Dir(path) == root {
				return filepath.SkipDir
			}
			return nil
		}
		lang := languageOf(path)
		if lang == langNone {
			return nil
		}
		if a.total >= maxFiles {
			a.capped = true
			return filepath.SkipAll
		}
		a.total++
		info, infoErr := d.Info()
		if infoErr != nil || info.Size() > maxFileBytes {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		symbols, targets, defs := parse(lang, rel, src, goModule, rootPrefix)
		f := &file{rel: rel, symbols: symbols, defs: defs}
		a.files = append(a.files, f)
		a.byRel[rel] = f
		for _, target := range targets {
			a.edges = append(a.edges, rawEdge{
				from:     rel,
				target:   target.key,
				line:     target.line,
				strength: target.strength,
			})
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if stopped {
		if len(a.files) == 0 {
			return nil, ctx.Err()
		}
		a.capped = true
	}
	a.resolve()
	return a, nil
}

// resolve turns the walk's raw edges into links, once, for both faces.
//
// Per importing file, the symbol targets go first: each `dir#Name` that a
// file of dir declares becomes a link to that file, and a package whose
// symbols resolved at all is done — its bare package target is dropped, so a
// file using only `model.Request` credits types.go and nothing else in
// internal/model. A bare target left over resolves the way it always did: to
// the file a script or Python import names, else to the package directory.
//
// Resolving one bare target can cost a scan of every mapped file (a
// dependency name falls through resolveTargetFile to a suffix search), and a
// project importing "context" from fifty files once paid that fifty times
// over, twice. Memoised per distinct target.
func (a *analysis) resolve() {
	// A name declared by several files of one package is a build-tagged
	// pair (hide_windows.go / hide_other.go both declare HideConsole), and
	// the importer means whichever one its platform builds — so each earns
	// the reference rather than the alphabetically first.
	defs := make(map[string]map[string][]*file)
	for _, f := range a.files {
		if len(f.defs) == 0 || strings.HasSuffix(f.rel, "_test.go") {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(f.rel))
		byName := defs[dir]
		if byName == nil {
			byName = make(map[string][]*file)
			defs[dir] = byName
		}
		for _, name := range f.defs {
			byName[name] = append(byName[name], f)
		}
	}

	resolved := make(map[string]string)
	bare := func(target string) string {
		if to, ok := resolved[target]; ok {
			return to
		}
		to := ""
		if f, ok := resolveTargetFile(a.byRel, target); ok {
			to = f.rel
		}
		resolved[target] = to
		return to
	}

	seen := make(map[link]bool)
	add := func(l link) {
		if l.from == l.to || seen[l] {
			return
		}
		seen[l] = true
		a.links = append(a.links, l)
	}
	// Two passes over the same edges, in file order: symbols first so the
	// bare pass knows which packages are already spoken for. The walk emits a
	// file's edges together, so "already spoken for" is per file.
	hit := make(map[[2]string]bool)
	for _, e := range a.edges {
		dir, name, ok := strings.Cut(e.target, symbolSep)
		if !ok {
			continue
		}
		for _, f := range defs[dir][name] {
			hit[[2]string{e.from, dir}] = true
			// A `dir#Name` target landed on the file that DECLARES the name, so
			// the link is as resolved as this map gets — whatever the parse
			// could say about a package path it had not looked up yet. The
			// strength a bare target keeps is decided below, where how it landed
			// is known.
			add(link{from: e.from, to: f.rel, line: e.line, strength: "resolved"})
		}
	}
	for _, e := range a.edges {
		if strings.Contains(e.target, symbolSep) || hit[[2]string{e.from, e.target}] {
			continue
		}
		if to := bare(e.target); to != "" {
			add(link{from: e.from, to: to, line: e.line, strength: e.strength})
			continue
		}
		// Nothing declared the name and no file answered the path: the line is
		// drawn to the package directory, which is this map's own convention for
		// "this import, spread over the package" — a guess, and labelled one.
		add(link{from: e.from, to: e.target, line: e.line, strength: "heuristic", pkg: true})
	}
}

// rank counts incoming references and sorts files most-load-bearing first —
// the shared ordering both faces of the map present.
//
// Test files reference nothing here. A mock that every test imports is the
// most-imported file of a frontend and the least load-bearing one — the
// first real frontend this walk met ranked test/mocks/wailsApp.ts second in
// the project — and the map's question is what the PRODUCT leans on. The
// graph still draws a test's imports (Graph), because a picture of what
// imports what is a different question.
func (a *analysis) rank() {
	goFilesOf := make(map[string][]*file)
	for _, f := range a.files {
		if strings.HasSuffix(f.rel, ".go") && !strings.HasSuffix(f.rel, "_test.go") {
			dir := filepath.ToSlash(filepath.Dir(f.rel))
			goFilesOf[dir] = append(goFilesOf[dir], f)
		}
	}
	// One credit per importing FILE, not per written reference. Several
	// references from one file to one target are one file depending on another
	// (the field's own contract), and a file naming two symbols of one package
	// is one importer, not two. What each written line proves belongs to the
	// evidence list (Facts), and the two must not be the same count: ranking by
	// reference count would let a file that writes `config.Config` and
	// `config.ConfigOptions` outrank one more of the project depends on.
	credited := make(map[linkKey]bool)
	for _, l := range a.links {
		if isTestFile(l.from) {
			continue
		}
		key := linkKey{from: l.from, to: l.to, pkg: l.pkg}
		if credited[key] {
			continue
		}
		credited[key] = true
		if l.pkg {
			for _, f := range goFilesOf[l.to] {
				f.refs++
			}
			continue
		}
		if f, ok := a.byRel[l.to]; ok {
			f.refs++
		}
	}
	// Most-referenced first, then the file with more to say, then path so two
	// runs of the same tree render the same map — a map that reorders itself
	// between calls reads like the project changed when it did not.
	sort.Slice(a.files, func(i, j int) bool {
		if a.files[i].refs != a.files[j].refs {
			return a.files[i].refs > a.files[j].refs
		}
		if len(a.files[i].symbols) != len(a.files[j].symbols) {
			return len(a.files[i].symbols) > len(a.files[j].symbols)
		}
		return a.files[i].rel < a.files[j].rel
	})
}

// Build walks root, parses what it recognizes, ranks files by incoming
// references and renders the highest-ranked into the budget. The error is only
// ever about reaching root at all — an unparseable file is a file with no
// symbols, not a failed map.
func Build(ctx context.Context, opts Options) (string, error) {
	if opts.Budget <= 0 {
		opts.Budget = DefaultBudget
	}
	a, err := analyze(ctx, opts)
	if err != nil {
		return "", err
	}
	a.rank()
	return render(a.files, a.total, a.capped, opts.Budget), nil
}

// scriptExts is the order an extensionless import is tried against files that
// exist, most common first. Mirrors languageOf's script list.
var scriptExts = []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".svelte", ".vue"}

// isTestFile recognises the test and mock files of the languages the walk
// parses, by the names their tools require or their communities settled on:
// Go's `_test.go`, `*.test.ts` / `*.spec.ts` and the folders a test runner
// walks, pytest's `test_*.py` / `*_test.py` / conftest.
func isTestFile(rel string) bool {
	if strings.HasSuffix(rel, "_test.go") {
		return true
	}
	base := rel[strings.LastIndex(rel, "/")+1:]
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return true
	}
	if strings.HasSuffix(base, ".py") &&
		(strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") || base == "conftest.py") {
		return true
	}
	for _, seg := range strings.Split(rel, "/") {
		switch seg {
		case "test", "tests", "__tests__", "mocks", "__mocks__", "e2e", "testdata", "spec":
			return true
		}
	}
	return false
}

// resolveTargetFile finds the file an import target names, trying the exact
// path first (the import carried its extension), then each script extension,
// then the directory's index file — the same order a bundler resolves in,
// because that is the order authors write against — then Python's spellings,
// and last a suffix match: a Python absolute import names its path from a
// package root ("app/db/gate"), and where that root sits in the repo
// ("backend/") is sys.path's secret. Ending-with is the whole of what this
// package can honestly know about it, and the shortest candidate wins so two
// runs of one tree agree.
func resolveTargetFile(byRel map[string]*file, target string) (*file, bool) {
	if f, ok := byRel[target]; ok {
		return f, true
	}
	for _, ext := range scriptExts {
		if f, ok := byRel[target+ext]; ok {
			return f, true
		}
	}
	for _, ext := range scriptExts {
		if f, ok := byRel[target+"/index"+ext]; ok {
			return f, true
		}
	}
	if f, ok := byRel[target+".py"]; ok {
		return f, true
	}
	if f, ok := byRel[target+"/__init__.py"]; ok {
		return f, true
	}
	var best *file
	for rel, f := range byRel {
		if !suffixMatches(rel, target) {
			continue
		}
		if best == nil || len(rel) < len(best.rel) || (len(rel) == len(best.rel) && rel < best.rel) {
			best = f
		}
	}
	return best, best != nil
}

// suffixMatches answers whether rel is target seen from some unknown package
// or alias root: Python's spellings and the script ones, because a tsconfig
// alias hides its root exactly the way sys.path does.
func suffixMatches(rel, target string) bool {
	if strings.HasSuffix(rel, "/"+target+".py") || strings.HasSuffix(rel, "/"+target+"/__init__.py") {
		return true
	}
	for _, ext := range scriptExts {
		if strings.HasSuffix(rel, "/"+target+ext) || strings.HasSuffix(rel, "/"+target+"/index"+ext) {
			return true
		}
	}
	return false
}

// render fits ranked files into the budget, whole files at a time: a file cut
// mid-symbol-list would look complete and be wrong, which is the one failure a
// map must not have.
func render(files []*file, total int, capped bool, budget int) string {
	var b strings.Builder
	shown := 0
	spent := 0
	budgetChars := budget * 4

	header := fmt.Sprintf("repo map — %d files, ranked by incoming references\n", total)
	if capped {
		header = fmt.Sprintf("repo map — first %d files (walk capped), ranked by incoming references\n", total)
	}
	spent += len(header)

	// Go's package-level import credit gives every file of a hot package the
	// same score, and on the first real run the whole budget went to six files
	// of internal/model — a map of one package wearing the name of the repo.
	// Two files may speak for a directory; the map's job is to say WHICH
	// packages matter, and read/glob take it from there.
	const maxFilesPerDir = 2
	perDir := make(map[string]int)

	for _, f := range files {
		if len(f.symbols) == 0 && f.refs == 0 {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(f.rel))
		if perDir[dir] >= maxFilesPerDir {
			continue
		}
		var fb strings.Builder
		fb.WriteString("\n")
		fb.WriteString(f.rel)
		if f.refs > 0 {
			fmt.Fprintf(&fb, "  (referenced by %d)", f.refs)
		}
		fb.WriteString("\n")
		show := f.symbols
		if len(show) > maxSymbolsPerFile {
			show = publicFirst(f.symbols, maxSymbolsPerFile)
		}
		for _, s := range show {
			fmt.Fprintf(&fb, "  %4d: %s\n", s.Line, s.Text)
		}
		if cut := len(f.symbols) - len(show); cut > 0 {
			fmt.Fprintf(&fb, "        … +%d more symbols\n", cut)
		}
		if spent+fb.Len() > budgetChars && shown > 0 {
			break
		}
		b.WriteString(fb.String())
		spent += fb.Len()
		shown++
		perDir[dir]++
	}

	rest := total - shown
	tail := ""
	if rest > 0 {
		tail = fmt.Sprintf("\n… %d more files under the budget line — glob lists them, read opens them\n", rest)
	}
	return header + b.String() + tail
}

// publicFirst is the cut when a file is over its share: publics claim the
// room first, privates fill what is left, and the survivors render in source
// order regardless — a map that reorders a file's API reads like a different
// file.
func publicFirst(symbols []Symbol, cap int) []Symbol {
	kept := make([]Symbol, 0, cap)
	for _, s := range symbols {
		if s.Public {
			kept = append(kept, s)
			if len(kept) == cap {
				return kept
			}
		}
	}
	for _, s := range symbols {
		if !s.Public {
			kept = append(kept, s)
			if len(kept) == cap {
				break
			}
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].Line < kept[j].Line })
	return kept
}

// projectRoot finds the nearest ancestor with go.mod (the module boundary)
// and returns:
//   - goModule: the module path, or "" outside a Go module.
//   - rootPrefix: relative slash-separated path from the module root down to root ("" if root is the module root or outside a Go module).
//   - ignoredAnywhere: directory names git-ignored at any depth (combining root's .gitignore and ancestor .gitignores up to the module root).
//   - ignoredAtRoot: directory names git-ignored at the mapped root only (from root's own .gitignore).
//
// An ancestor's .gitignore is consulted for anywhere-ignores only: an ancestor's
// `/output` means output at that ancestor's root, not at root/output.
func projectRoot(root string) (goModule, rootPrefix string, ignoredAnywhere, ignoredAtRoot map[string]bool) {
	ignoredAnywhere = make(map[string]bool)
	ignoredAtRoot = make(map[string]bool)
	parseGitignoreFile(filepath.Join(root, ".gitignore"), ignoredAnywhere, ignoredAtRoot)

	var modDir string
	var ancestors []string
	dir := root
	for {
		if mod := readGoModFile(filepath.Join(dir, "go.mod")); mod != "" {
			modDir = dir
			goModule = mod
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached filesystem root
			break
		}
		ancestors = append(ancestors, parent)
		dir = parent
	}

	if modDir == "" {
		return "", "", ignoredAnywhere, ignoredAtRoot
	}

	if modDir != root {
		if rel, err := filepath.Rel(modDir, root); err == nil && rel != "." {
			rootPrefix = filepath.ToSlash(rel)
		}
	}

	for _, anc := range ancestors {
		parseGitignoreFile(filepath.Join(anc, ".gitignore"), ignoredAnywhere, nil)
		if anc == modDir {
			break
		}
	}

	return goModule, rootPrefix, ignoredAnywhere, ignoredAtRoot
}

func readGoModFile(path string) string {
	src, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

func parseGitignoreFile(path string, anywhere, atRoot map[string]bool) {
	src, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		if strings.ContainsAny(line, "*?[]") {
			continue
		}
		rooted := strings.HasPrefix(line, "/")
		name := strings.Trim(line, "/")
		if name == "" || strings.Contains(name, "/") {
			continue
		}
		if rooted {
			if atRoot != nil {
				atRoot[name] = true
			}
		} else {
			if anywhere != nil {
				anywhere[name] = true
			}
		}
	}
}

