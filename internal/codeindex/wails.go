package codeindex

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const wailsProducer = "wails-v2"

// wailsFactsVersion is the version of this adapter's reading, stored with the
// facts under wailsProducer. Bump it whenever a change here would make facts
// already in a cache wrong — a shape it now accepts, a pairing rule, which line
// a hop is recorded at — because nothing else in the cache records which build
// produced a row, and a stale bridge fact reads exactly like a fresh one.
const wailsFactsVersion = "1"

var (
	wailsExportRE   = regexp.MustCompile(`^\s*export\s+(?:async\s+)?function\s+([A-Za-z_$][\w$]*)\s*\(`)
	wailsDispatchRE = regexp.MustCompile(`window\[['"]go['"]\]\[['"]([^'"]+)['"]\]\[['"]([^'"]+)['"]\]\[['"]([^'"]+)['"]\]`)
	wailsImportRE   = regexp.MustCompile(`(?s)import\s*\{([^}]*)\}\s*from\s*['"]([^'"]*wailsjs/go/[^'"]+)['"]`)
)

type wailsBinding struct {
	path       string
	exportName string
	exportLine int
	dispatch   Location
	pkg        string
	receiver   string
	method     string
}

type namedImport struct {
	exported string
	local    string
}

// scanWails reads both halves of the generated boundary and stores what it
// could pair. What it could not pair is returned as a count rather than
// dropped: the two halves are a generated file the project did not write, and
// "this project has no Wails" is a different statement from "these generated
// files are not the shape this adapter reads".
func (i *Indexer) scanWails(ctx context.Context) ([]Fact, int, bool, error) {
	bindings, unpaired, found, err := i.wailsBindings(ctx)
	if err != nil {
		return nil, 0, false, err
	}
	if !found {
		return nil, 0, false, nil
	}
	// Only the methods the generated bindings actually name. The Go half of
	// this walk used to parse every .go file in the project on every index,
	// and on Aetox that alone is most of a minute's work thrown away: the
	// bindings name a few hundred receivers, not two thousand files.
	wanted := make(map[string]bool, len(bindings))
	for _, binding := range bindings {
		wanted[binding.method] = true
	}
	methods, err := i.goMethods(ctx, wanted)
	if err != nil {
		return nil, unpaired, true, err
	}
	factsBySource := make(map[string][]Fact)
	unknown := unpaired
	for _, binding := range bindings {
		key := binding.pkg + "." + binding.receiver + "." + binding.method
		locations := methods[key]
		if len(locations) != 1 {
			// Zero means the Go half is gone or was renamed; more than one
			// means the name is not the identity this adapter assumed. Either
			// way the hop is unknown, never guessed at.
			unknown++
			continue
		}
		fact := Fact{
			From:     binding.dispatch,
			To:       locations[0],
			Relation: RelationWailsRPC,
			Strength: StrengthResolved,
			Producer: wailsProducer,
		}
		factsBySource[binding.path] = append(factsBySource[binding.path], fact)
	}
	frontendFacts, frontendUnknown, err := i.frontendWailsCalls(ctx, bindings)
	if err != nil {
		return nil, unknown, true, err
	}
	unknown += frontendUnknown
	for source, facts := range frontendFacts {
		factsBySource[source] = append(factsBySource[source], facts...)
	}
	var all []Fact
	for _, facts := range factsBySource {
		sortFacts(facts)
		all = append(all, facts...)
	}
	if err := i.store.ReplaceProducer(wailsProducer, wailsFactsVersion, unknown, factsBySource); err != nil {
		return nil, unknown, true, err
	}
	sortFacts(all)
	return all, unknown, true, nil
}

// wailsBindings pairs each exported function in a generated binding with the
// dispatch inside its body. The counts are the point: `unpaired` is every
// dispatch this adapter did not turn into a binding, whatever shape the export
// around it had, and `found` is whether there are generated Wails files here at
// all.
func (i *Indexer) wailsBindings(ctx context.Context) (map[string]wailsBinding, int, bool, error) {
	bindings := make(map[string]wailsBinding)
	found := false
	unpaired := 0
	err := walkSource(i.root, func(path, rel string, entry os.DirEntry) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slash := filepath.ToSlash(rel)
		if filepath.Ext(path) != ".js" || !strings.Contains(slash, "/wailsjs/go/") {
			return nil
		}
		found = true
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(body), "\n")
		dispatchLines := make(map[int]bool)
		for at, line := range lines {
			if wailsDispatchRE.MatchString(line) {
				dispatchLines[at] = true
			}
		}
		for lineNo := 0; lineNo < len(lines); lineNo++ {
			export := wailsExportRE.FindStringSubmatch(lines[lineNo])
			if export == nil {
				continue
			}
			paired := false
			sawDispatch := false
			for at := lineNo + 1; at < len(lines); at++ {
				dispatch := wailsDispatchRE.FindStringSubmatch(lines[at])
				if dispatch != nil {
					// One defect, one count: a body whose export name and
					// dispatch name disagree is a dispatch nobody paired, and
					// the leftover-dispatch tally below is what reports it.
					sawDispatch = true
					if dispatch[3] != export[1] {
						break
					}
					paired = true
					binding := wailsBinding{
						path:       slash,
						exportName: export[1],
						exportLine: lineNo + 1,
						dispatch: Location{
							Path: slash,
							Line: at + 1,
							Name: export[1],
						},
						pkg:      dispatch[1],
						receiver: dispatch[2],
						method:   dispatch[3],
					}
					bindings[slash+"#"+export[1]] = binding
					delete(dispatchLines, at)
					break
				}
				if strings.TrimSpace(lines[at]) == "}" {
					break
				}
			}
			if !paired && !sawDispatch {
				// An export with no dispatch in its body at all: a generated
				// shape this adapter does not read.
				unpaired++
			}
		}
		// Whatever is left is a dispatch nobody paired: an export shape this
		// adapter does not read, or a body whose export name and dispatch name
		// disagree.
		unpaired += len(dispatchLines)
		return nil
	})
	return bindings, unpaired, found, err
}

// goMethods finds the Go method each generated binding means. wanted is the
// set of method names the bindings name; a file that contains none of them
// cannot hold one, and is not parsed. The check is textual on purpose — it is
// a filter in front of the parser, not a second parser, and a file it lets
// through wrongly costs one parse and no correctness.
func (i *Indexer) goMethods(ctx context.Context, wanted map[string]bool) (map[string][]Location, error) {
	methods := make(map[string][]Location)
	if len(wanted) == 0 {
		return methods, nil
	}
	err := walkSource(i.root, func(path, rel string, entry os.DirEntry) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !holdsAnyName(source, wanted) {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, source, 0)
		if err != nil || file == nil {
			return nil
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			receiver := receiverType(fn.Recv.List[0].Type)
			if receiver == "" {
				continue
			}
			key := file.Name.Name + "." + receiver + "." + fn.Name.Name
			methods[key] = append(methods[key], Location{
				Path: filepath.ToSlash(rel),
				Line: fset.Position(fn.Name.Pos()).Line,
				Name: fn.Name.Name,
			})
		}
		return nil
	})
	return methods, err
}

// holdsAnyName reports whether the source mentions any of the names at all.
func holdsAnyName(source []byte, wanted map[string]bool) bool {
	if !bytes.Contains(source, []byte("func (")) {
		return false
	}
	for name := range wanted {
		if bytes.Contains(source, []byte(name)) {
			return true
		}
	}
	return false
}

func receiverType(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverType(value.X)
	case *ast.IndexExpr:
		return receiverType(value.X)
	case *ast.IndexListExpr:
		return receiverType(value.X)
	default:
		return ""
	}
}

// frontendWailsCalls finds the first half of the boundary: a frontend file
// importing a name from a generated module and calling it.
//
// An import that resolves to no generated file is passed over in silence. The
// generated tree also holds modules that name no binding at all — Aetox's own
// frontend imports `wailsjs/go/models` in thirty-odd files to name the Go
// types on the wire — and counting those as "relationships that could not be
// established" would put a permanent, growing number in front of the model on
// a project where nothing is wrong. A module that IS there and does not export
// the name is a different thing, and is counted.
func (i *Indexer) frontendWailsCalls(ctx context.Context, bindings map[string]wailsBinding) (map[string][]Fact, int, error) {
	facts := make(map[string][]Fact)
	unknown := 0
	err := walkSource(i.root, func(path, rel string, entry os.DirEntry) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".svelte" {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if strings.Contains(slash, "/wailsjs/") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// The cheap gate before the regexes: a file that never mentions the
		// generated tree cannot import from it, and most of a frontend does not.
		if !bytes.Contains(body, []byte("wailsjs/go")) {
			return nil
		}
		text := string(body)
		for _, match := range wailsImportRE.FindAllStringSubmatchIndex(text, -1) {
			// An import written inside a comment is not an import. Aetox's own
			// frontend keeps one commented-out binding call in
			// lib/services/cockpit.ts, and counting it put a permanent "1
			// relationship could not be established" in front of the model on
			// a project where nothing was wrong.
			if inComment(text, match[0]) {
				continue
			}
			names := text[match[2]:match[3]]
			importPath := text[match[4]:match[5]]
			bindingPath := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(slash), filepath.FromSlash(importPath))))
			if filepath.Ext(bindingPath) == "" {
				bindingPath += ".js"
			}
			if !i.exists(bindingPath) {
				continue
			}
			for _, named := range parseNamedImports(names) {
				binding, ok := bindings[bindingPath+"#"+named.exported]
				if !ok {
					unknown++
					continue
				}
				callLines := findCallLines(text, named.local, lineOf(text, match[1]))
				for _, line := range callLines {
					facts[slash] = append(facts[slash], Fact{
						From:     Location{Path: slash, Line: line, Name: named.local},
						To:       Location{Path: binding.path, Line: binding.exportLine, Name: binding.exportName},
						Relation: RelationReference,
						Strength: StrengthResolved,
						Producer: wailsProducer,
					})
				}
			}
		}
		return nil
	})
	return facts, unknown, err
}

func (i *Indexer) exists(rel string) bool {
	info, err := os.Stat(filepath.Join(i.root, filepath.FromSlash(rel)))
	return err == nil && !info.IsDir()
}

func parseNamedImports(text string) []namedImport {
	var imports []namedImport
	for _, raw := range strings.Split(text, ",") {
		fields := strings.Fields(strings.TrimSpace(raw))
		if len(fields) == 1 && fields[0] != "type" {
			imports = append(imports, namedImport{exported: fields[0], local: fields[0]})
		}
		if len(fields) == 3 && fields[1] == "as" {
			imports = append(imports, namedImport{exported: fields[0], local: fields[2]})
		}
	}
	return imports
}

// findCallLines returns the lines that call the name. Comment lines are passed
// over: a doc comment mentioning a binding is not a call, and a hop stamped
// "resolved" has to be a line somebody could point at. A match inside a string
// literal is still counted, which is the shape this cannot see without a real
// parser — the caller's file is frontend source, and the alternative is not
// reading it at all.
func findCallLines(text, name string, importLine int) []int {
	callRE := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\(`)
	var lines []int
	seen := make(map[int]bool)
	for _, match := range callRE.FindAllStringIndex(text, -1) {
		// Decided per MATCH, not per line: a call written after the `//` of a
		// trailing comment sits on a line that starts with code, and a rule
		// that asks only about the line start would call it a call.
		if inComment(text, match[0]) {
			continue
		}
		line := lineOf(text, match[0])
		if line == importLine || seen[line] {
			continue
		}
		seen[line] = true
		lines = append(lines, line)
	}
	sort.Ints(lines)
	return lines
}

// inComment reports whether the text at offset sits in a comment — a line that
// is a comment, or a trailing one after code on the same line.
func inComment(text string, offset int) bool {
	if offset < 0 || offset > len(text) {
		return false
	}
	lineStart := strings.LastIndex(text[:offset], "\n") + 1
	line := text[lineStart:]
	if end := strings.Index(line, "\n"); end >= 0 {
		line = line[:end]
	}
	if isFrontendComment(line) {
		return true
	}
	// A trailing comment: everything after `//` on this line is not code, so a
	// match starting past it is prose.
	if at := strings.Index(line, "//"); at >= 0 {
		return offset-lineStart >= at
	}
	return false
}

// isFrontendComment is the comment shapes of the frontend languages this
// adapter reads: .ts, .tsx, .js, .jsx and .svelte. Deliberately not Go's rules —
// a `#` here is a private class field, not a comment.
func isFrontendComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range []string{"//", "/*", "*", "<!--"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

func lineOf(text string, offset int) int {
	return strings.Count(text[:offset], "\n") + 1
}

func sortFacts(facts []Fact) {
	sort.Slice(facts, func(a, b int) bool {
		left := facts[a]
		right := facts[b]
		if left.From.Path != right.From.Path {
			return left.From.Path < right.From.Path
		}
		if left.From.Line != right.From.Line {
			return left.From.Line < right.From.Line
		}
		if left.To.Path != right.To.Path {
			return left.To.Path < right.To.Path
		}
		if left.To.Line != right.To.Line {
			return left.To.Line < right.To.Line
		}
		return left.Relation < right.Relation
	})
}

func walkSource(root string, visit func(path, rel string, entry os.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (strings.HasPrefix(name, ".") || ignoredSourceDir(name)) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		return visit(path, rel, entry)
	})
}

func ignoredSourceDir(name string) bool {
	switch strings.ToLower(name) {
	case "node_modules", "vendor", "dist", "build", ".svelte-kit", "coverage":
		return true
	default:
		return false
	}
}
