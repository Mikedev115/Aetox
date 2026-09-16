// The map's second face: the same analysis the model reads as budgeted text,
// handed to the UI as nodes and edges. One walk, one ranking, one ignore list
// — the owner's requirement was that the system's view "ซิงค์กับ repo_map
// ที่โมเดลเขียนไว้", and the only sync that cannot drift is sharing the
// computation itself (analyze/rank above), never a cached copy of its output.
//
// What differs is only what each reader can afford: the model pays per token,
// so Build cuts to ~1k and eight symbols a file; a screen pays per pixel, so
// Graph keeps whole files and real edges and cuts only the node COUNT, because
// a thousand circles is not a picture of anything.
package repomap

import (
	"context"
	"math"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultMaxNodes is how many files the graph keeps when the caller has no
// opinion. Sixty is where a force-laid graph still reads as districts and hubs
// rather than as felt — the right first sentence, not the only one the reader
// is allowed to hear.
const DefaultMaxNodes = 60

// AllNodes, passed as maxNodes, lifts the ceiling entirely: every file the walk
// mapped becomes a node. There is no number to invent here, because the honest
// ceiling was always the project's own size — and a caller that wants the whole
// thing should not have to guess how big the whole thing is to ask for it.
const AllNodes = -1

// Node is one file the graph kept, ranked the same way the text map ranks.
type Node struct {
	Path    string `json:"path"`
	Dir     string `json:"dir"`
	Refs    int    `json:"refs"`
	Symbols int    `json:"symbols"`
}

// Edge points from the importing node to the imported one, by index into the
// node list. Indices rather than paths because the consumer is a renderer,
// and a renderer joining strings to find its endpoints is doing this
// package's job with worse tools.
type Edge struct {
	From         int    `json:"from"`
	To           int    `json:"to"`
	Kind         string `json:"kind"`
	Strength     string `json:"strength"`
	EvidenceLine int    `json:"evidenceLine"`
	Producer     string `json:"producer"`
}

// Location is one evidence anchor in a structural fact.
type Location struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// Fact is one evidence-backed file relationship found by the structural walk.
type Fact struct {
	From     Location `json:"from"`
	To       Location `json:"to"`
	Kind     string   `json:"kind"`
	Strength string   `json:"strength"`
	Producer string   `json:"producer"`
}

// Graph analyzes root and returns the top maxNodes files with the import
// edges that run between them, plus how many mapped files exist in total —
// the number that tells the reader how much of the repository the picture is.
func Graph(ctx context.Context, opts Options, maxNodes int) ([]Node, []Edge, int, error) {
	switch {
	case maxNodes == AllNodes:
		maxNodes = math.MaxInt
	case maxNodes <= 0:
		maxNodes = DefaultMaxNodes
	}
	a, err := analyze(ctx, opts)
	if err != nil {
		return nil, nil, 0, err
	}
	a.rank()

	// A file with nothing but an outgoing import (an entry point, a thin
	// wiring file) is invisible to the ranking — no symbols, no incoming refs
	// — and it is exactly the arrow-tail a graph exists to draw. Counted here
	// so selection can keep it.
	spokesperson := goSpokespersons(a)
	endpoint := func(l link) string {
		if !l.pkg {
			return l.to
		}
		return spokesperson[l.to]
	}

	outDeg := make(map[string]int)
	for _, l := range a.links {
		if endpoint(l) != "" {
			outDeg[l.from]++
		}
	}

	// Reserve for what the tree can actually supply, never for the ceiling:
	// AllNodes is math.MaxInt, and a slice asked to reserve that much panics.
	nodes := make([]Node, 0, min(maxNodes, len(a.files)))
	index := make(map[string]int)
	for _, f := range a.files {
		if len(nodes) >= maxNodes {
			break
		}
		if len(f.symbols) == 0 && f.refs == 0 && outDeg[f.rel] == 0 {
			continue
		}
		index[f.rel] = len(nodes)
		nodes = append(nodes, Node{
			Path:    f.rel,
			Dir:     filepath.ToSlash(filepath.Dir(f.rel)),
			Refs:    f.refs,
			Symbols: len(f.symbols),
		})
	}

	edges := resolveEdges(a, index, endpoint)
	return nodes, edges, a.total, nil
}

// FactVersion is the version of this analyzer's reading, and a cache stores it
// beside the facts because the facts cannot say it themselves.
//
// Bump it for any change to what this package decides: which targets the parse
// records, how they resolve, what strength they get. A cached fact carries the
// content stamps of the two files it joins and nothing about the code that
// decided it existed, so without this a fact written by an older build is
// served as fresh — and a number measured against that cache cannot be told
// apart from a number measured against the build that wrote it.
const FactVersion = "1"

// Facts returns every resolved structural relationship with the line that
// proves it, the files the walk saw, and whether the walk was cut short. Unlike
// Graph it has no display ceiling and does not rank or render anything.
//
// The file list is returned because a producer that stores these facts in a
// cache needs to know which sources it is responsible for: a file whose
// imports were all removed produces no facts at all, and "no facts" is not the
// same statement as "this file is not part of the project". The capped flag is
// returned for the same reason `Graph` renders "(walk capped)": a deadline or
// the file ceiling means the list is partial, and a caller that stores it must
// not call the result complete.
func Facts(ctx context.Context, opts Options) ([]Fact, []string, bool, error) {
	a, err := analyze(ctx, opts)
	if err != nil {
		return nil, nil, false, err
	}
	spokesperson := goSpokespersons(a)
	facts := make([]Fact, 0, len(a.links))
	for _, link := range a.links {
		to := link.to
		if link.pkg {
			to = spokesperson[link.to]
		}
		if to == "" || to == link.from {
			continue
		}
		facts = append(facts, Fact{
			From: Location{Path: link.from, Line: link.line},
			// No line: an import names a file, and the line that proves it is the
			// importer's own. Inventing line 1 for the target would print a
			// number nobody read beside the numbers that were.
			To:       Location{Path: to},
			Kind:     "import",
			Strength: link.strength,
			Producer: "repomap",
		})
	}
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].From.Path != facts[j].From.Path {
			return facts[i].From.Path < facts[j].From.Path
		}
		if facts[i].To.Path != facts[j].To.Path {
			return facts[i].To.Path < facts[j].To.Path
		}
		return facts[i].From.Line < facts[j].From.Line
	})
	files := make([]string, 0, len(a.files))
	for _, f := range a.files {
		files = append(files, f.rel)
	}
	return facts, files, a.capped, nil
}

// resolveEdges turns the analysis's links into drawn lines between kept
// nodes. A link already names a file — the one a script or Python import
// wrote, or the Go file that declares the name the importer used — so the
// line lands where the reference actually goes. Only a Go import that
// resolved to no symbol still names a package directory, and that line lands
// on the package's fullest file — a spokesperson, chosen deterministically,
// because fanning one import out to every file of the package would draw
// edges nobody wrote.
func resolveEdges(a *analysis, index map[string]int, endpoint func(link) string) []Edge {
	var edges []Edge
	seen := make(map[[2]int]bool)
	for _, l := range a.links {
		to := endpoint(l)
		if to == "" || to == l.from {
			continue
		}
		fi, ok := index[l.from]
		if !ok {
			continue
		}
		ti, ok := index[to]
		if !ok {
			continue
		}
		key := [2]int{fi, ti}
		if seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, Edge{
			From:         fi,
			To:           ti,
			Kind:         "import",
			Strength:     l.strength,
			EvidenceLine: l.line,
			Producer:     "repomap",
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	return edges
}

// goSpokespersons picks, per directory, the non-test Go file with the most
// symbols (ties to the shorter then earlier path) — where a package-level
// import edge lands when none of the importer's selectors resolved to a file.
func goSpokespersons(a *analysis) map[string]string {
	best := make(map[string]*file)
	for _, f := range a.files {
		if !strings.HasSuffix(f.rel, ".go") || strings.HasSuffix(f.rel, "_test.go") {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(f.rel))
		cur := best[dir]
		if cur == nil ||
			len(f.symbols) > len(cur.symbols) ||
			(len(f.symbols) == len(cur.symbols) && f.rel < cur.rel) {
			best[dir] = f
		}
	}
	out := make(map[string]string, len(best))
	for dir, f := range best {
		out[dir] = f.rel
	}
	return out
}
