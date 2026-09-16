package codeindex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/repomap"
)

// structuralProducer is the name the import facts are stored under. Their
// version is repomap's own (repomap.FactVersion), because that is the package
// that decides which facts exist.
const structuralProducer = "repomap"

// analyzer names one producer's reading together with the version of the code
// that wrote it. The two travel as one value because the version describes the
// rows stored under the name, and a name on its own cannot say whether they
// were written by the build that is asking.
type analyzer struct {
	name    string
	version string
}

// refreshTimeout is the budget for building the index, and it is separate from
// the budget for asking a question.
//
// They were one number, and on a real project that made the tool useless: a
// query spent its whole eight seconds rebuilding the index and then answered
// from an expired clock, so `codebase trace` on Aetox itself returned
// "time budget reached" with no hops at all, every time. The index is built
// once per project and reused; the question is the part that has to be fast.
const refreshTimeout = 60 * time.Second

// Indexer coordinates structural and semantic producers for one repository.
type Indexer struct {
	root   string
	store  *Store
	budget Budget
}

// NewIndexer opens the repository cache with one whole-query budget.
func NewIndexer(root string, budget Budget) (*Indexer, error) {
	defaults := DefaultBudget()
	if budget.MaxRequests <= 0 {
		budget.MaxRequests = defaults.MaxRequests
	}
	if budget.MaxFacts <= 0 {
		budget.MaxFacts = defaults.MaxFacts
	}
	if budget.MaxDepth <= 0 {
		budget.MaxDepth = defaults.MaxDepth
	}
	if budget.MaxDuration <= 0 {
		budget.MaxDuration = defaults.MaxDuration
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	store, err := Open(abs)
	if err != nil {
		return nil, err
	}
	return &Indexer{root: abs, store: store, budget: budget}, nil
}

// Close releases the derived cache.
func (i *Indexer) Close() error {
	if i == nil {
		return nil
	}
	return i.store.Close()
}

// Index runs every producer over the project. It is the expensive call, it is
// idempotent, and Trace runs it only when the answer would otherwise be built
// on nothing or on rows whose files have changed.
func (i *Indexer) Index(ctx context.Context) Result {
	indexCtx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()

	var result Result
	result.Rebuilt = true
	result.Requests = 2
	structural := i.IndexStructural(indexCtx)
	if structural.Outcome == OutcomeFailed {
		structural.Requests = result.Requests
		return structural
	}
	result.Sources = structural.Sources
	result.Facts = append(result.Facts, structural.Facts...)

	wails := i.IndexWails(indexCtx)
	if wails.Outcome == OutcomeFailed {
		wails.Requests = result.Requests
		return wails
	}
	result.Facts = append(result.Facts, wails.Facts...)
	result.Unknown = wails.Unknown
	sortFacts(result.Facts)

	switch {
	case structural.Outcome == OutcomePartial || wails.Outcome == OutcomePartial:
		result.Outcome = OutcomePartial
		result.Truncated = true
		result.Reason = "a producer stopped early"
	case wails.Outcome == OutcomeUnsupported:
		// Nothing to cross here, and that is not a failure of the walk. Only
		// said when the walk itself found nothing, which the caller decides.
		result.Outcome = OutcomeComplete
	default:
		result.Outcome = OutcomeComplete
	}
	return result
}

// IndexStructural refreshes the evidence-bearing import facts. It hands the
// store its whole reading of the project at once, so a file whose imports were
// all removed stops having rows rather than leaving stale ones behind.
func (i *Indexer) IndexStructural(ctx context.Context) Result {
	facts, files, capped, err := repomap.Facts(ctx, repomap.Options{Root: i.root})
	if err != nil {
		return Result{Outcome: OutcomeFailed, Reason: err.Error()}
	}
	if ctx.Err() != nil {
		return Result{Outcome: OutcomePartial, Truncated: true, Reason: "time budget reached"}
	}
	bySource := make(map[string][]Fact, len(files))
	stored := make([]Fact, 0, len(facts))
	for _, fact := range facts {
		converted := Fact{
			From:     Location{Path: fact.From.Path, Line: fact.From.Line},
			To:       Location{Path: fact.To.Path, Line: fact.To.Line},
			Relation: Relation(fact.Kind),
			Strength: Strength(fact.Strength),
			Producer: fact.Producer,
		}
		bySource[converted.From.Path] = append(bySource[converted.From.Path], converted)
		stored = append(stored, converted)
	}
	if err := i.store.ReplaceProducer(structuralProducer, repomap.FactVersion, 0, bySource); err != nil {
		return Result{Outcome: OutcomeFailed, Facts: stored, Reason: err.Error()}
	}
	if err := i.store.ReplaceSources(files); err != nil {
		return Result{Outcome: OutcomeFailed, Facts: stored, Reason: err.Error()}
	}
	sortFacts(stored)
	result := Result{Outcome: OutcomeComplete, Facts: stored, Sources: len(files)}
	if capped {
		// The walk hit the file ceiling or a deadline: the shape is right and
		// the list is short, which is exactly the difference a caller storing
		// this must be told about.
		result.Outcome = OutcomePartial
		result.Truncated = true
		result.Reason = "the project walk was cut short"
	}
	return result
}

// IndexWails refreshes the deterministic Wails v2 boundary facts.
func (i *Indexer) IndexWails(ctx context.Context) Result {
	facts, unknown, supported, err := i.scanWails(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return Result{
				Outcome:   OutcomePartial,
				Facts:     facts,
				Unknown:   unknown,
				Truncated: true,
				Reason:    "time budget reached",
			}
		}
		return Result{Outcome: OutcomeFailed, Facts: facts, Unknown: unknown, Reason: err.Error()}
	}
	if !supported {
		// Recorded as run-and-found-nothing: without this the next question
		// would read the same absence as "never indexed" and walk the project
		// again to reach the same answer.
		if err := i.store.MarkIndexed(wailsProducer, wailsFactsVersion, unknown); err != nil {
			return Result{Outcome: OutcomeFailed, Reason: err.Error()}
		}
		return Result{Outcome: OutcomeUnsupported, Reason: "no supported Wails v2 bindings found"}
	}
	outcome := OutcomeComplete
	if unknown > 0 {
		outcome = OutcomePartial
	}
	return Result{Outcome: outcome, Facts: facts, Unknown: unknown}
}

// ---- asking a question ----

type traceNode struct {
	path string
	// name is the symbol this node is being read as. A hop that names nothing at
	// its far end — an import — does not change it: the walk is following one
	// name and a file-level hop says only that two files are connected.
	name string
	// line is where the fact that brought the walk here landed on this node, and
	// 0 at the start, where no fact brought it anywhere. It is what makes a node
	// narrowable once the name has nothing left to say: the structural producer
	// writes its facts per file and names no symbol, so for a file with one edge
	// per function the line is the only thing that says which edge is about the
	// name in hand.
	line int
}

type traceParent struct {
	previous traceNode
	fact     Fact
}

// Trace answers what a name connects to.
//
// The index is built only when there is nothing to answer from, and rebuilt
// only when a row it holds no longer matches its files. An unchanged project
// therefore costs one read: no walk, no parse, no language server. That is what
// the content stamps are FOR — a cache that had to be rebuilt on every question
// would make the stamps pointless and the tool too slow to call.
func (i *Indexer) Trace(ctx context.Context, request TraceRequest) Result {
	prepared, err := i.prepare(request)
	if err != nil {
		return Result{Outcome: OutcomeFailed, Reason: err.Error()}
	}
	if prepared.outcome != "" {
		return Result{Outcome: prepared.outcome, Reason: prepared.reason}
	}
	request = prepared.request

	built, err := i.indexed()
	if err != nil {
		return Result{Outcome: OutcomeFailed, Reason: err.Error()}
	}
	builtNow := false
	if !built {
		if result := i.Index(ctx); result.Outcome == OutcomeFailed {
			return result
		}
		builtNow = true
	}
	// A file the walk never mapped — created, renamed, or moved since the last
	// index — has no rows and no stamp, so it is indistinguishable from a file
	// with no relationships. Ask which of the two it is, and index again before
	// answering "nothing is connected to this" about a file nobody looked at.
	unmapped, err := i.unmapped(ctx, request)
	if err != nil {
		return Result{Outcome: OutcomeFailed, Reason: err.Error()}
	}
	if unmapped != "" {
		if result := i.Index(ctx); result.Outcome == OutcomeFailed {
			return result
		}
		builtNow = true
		unmapped, err = i.unmapped(ctx, request)
		if err != nil {
			return Result{Outcome: OutcomeFailed, Reason: err.Error()}
		}
	}
	if unmapped != "" {
		return Result{
			Outcome: OutcomeUnsupported,
			Reason:  unmapped + " is not a file this index reads (it walks source files it can parse)",
		}
	}
	// Read after the index exists, never before: on a project's first question
	// the analyzer has not run yet, and asking the store what it could not
	// establish would answer about a pass that never happened.
	result := i.walkWithUnknown(ctx, request)
	result.Rebuilt = builtNow
	if result.Stale > 0 {
		// Something the cache holds has moved on disk. Rebuild once and walk
		// again rather than answering from a graph with holes in it — the
		// stale rows are exactly the ones that would have been the answer.
		if rebuilt := i.Index(ctx); rebuilt.Outcome == OutcomeFailed {
			return rebuilt
		}
		result = i.walkWithUnknown(ctx, request)
		result.Rebuilt = true
		result.Reason = joinReasons(result.Reason, "the index was rebuilt because a file changed")
	}
	return result
}

// unmapped returns the first of the request's paths that the walk that built
// this cache never mapped, or "" when both are known.
func (i *Indexer) unmapped(ctx context.Context, request TraceRequest) (string, error) {
	for _, path := range []string{request.Path, request.TargetPath} {
		if path == "" || path == "." {
			continue
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		mapped, err := i.store.IndexedSource(path)
		if err != nil {
			return "", err
		}
		if !mapped {
			return path, nil
		}
	}
	return "", nil
}

func (i *Indexer) walkWithUnknown(ctx context.Context, request TraceRequest) Result {
	unknown, err := i.store.Unknown(wailsProducer)
	if err != nil {
		return Result{Outcome: OutcomeFailed, Reason: err.Error()}
	}
	return i.walk(ctx, request, unknown)
}

type preparedRequest struct {
	request TraceRequest
	outcome Outcome
	reason  string
}

// prepare validates the request and brings its paths to the project's own
// spelling, before any index work happens. A path that is not a file here has
// its own answer, asked first: every lookup below is a string key into an
// index, so a typo, a directory, or a moved file would otherwise return the
// same silence as a file nobody references.
func (i *Indexer) prepare(request TraceRequest) (preparedRequest, error) {
	return i.validate(request)
}

func (i *Indexer) validate(request TraceRequest) (preparedRequest, error) {
	request.Path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(request.Path)))
	request.Name = strings.TrimSpace(request.Name)
	request.TargetPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(request.TargetPath)))
	request.TargetName = strings.TrimSpace(request.TargetName)
	if request.Path == "." || request.Path == "" || request.Name == "" {
		return preparedRequest{outcome: OutcomeFailed, reason: "path and name are required"}, nil
	}
	if !i.exists(request.Path) {
		return preparedRequest{outcome: OutcomeUnavailable, reason: request.Path + " is not a file in this project"}, nil
	}
	request.Path = i.projectSpelling(request.Path)
	if request.Direction == "" {
		request.Direction = DirectionCallees
	}
	if request.Direction != DirectionCallers && request.Direction != DirectionCallees && request.Direction != DirectionBetween {
		return preparedRequest{outcome: OutcomeFailed, reason: fmt.Sprintf("unknown direction %q", request.Direction)}, nil
	}
	if request.Direction == DirectionBetween {
		if request.TargetPath == "." || request.TargetPath == "" {
			return preparedRequest{outcome: OutcomeFailed, reason: "targetPath is required for direction=between"}, nil
		}
		if !i.exists(request.TargetPath) {
			return preparedRequest{outcome: OutcomeUnavailable, reason: request.TargetPath + " is not a file in this project"}, nil
		}
		request.TargetPath = i.projectSpelling(request.TargetPath)
	}
	if request.Depth <= 0 {
		request.Depth = DefaultDepth()
	}
	return preparedRequest{request: request}, nil
}

func (i *Indexer) indexed() (bool, error) {
	for _, producer := range []analyzer{
		{name: structuralProducer, version: repomap.FactVersion},
		{name: wailsProducer, version: wailsFactsVersion},
	} {
		done, err := i.store.Indexed(producer.name, producer.version)
		if err != nil {
			return false, err
		}
		if !done {
			return false, nil
		}
	}
	return true, nil
}

// walk reads the cached facts and follows them. It touches the disk only to
// check whether a file still matches what was stored.
func (i *Indexer) walk(ctx context.Context, request TraceRequest, wailsUnknown int) Result {
	bounded, cancel := context.WithTimeout(ctx, i.budget.MaxDuration)
	defer cancel()

	depth := request.Depth
	clamped := false
	if depth > i.budget.MaxDepth {
		depth = i.budget.MaxDepth
		clamped = true
	}

	start := traceNode{path: request.Path, name: request.Name}
	frontier := []traceNode{start}
	seenNodes := map[traceNode]bool{start: true}
	seenFacts := make(map[Fact]bool)
	parents := make(map[traceNode]traceParent)

	result := Result{
		Outcome: OutcomeComplete,
		Unknown: wailsUnknown,
	}
	if clamped {
		result.Truncated = true
		result.Reason = "depth budget reached"
	}

	for level := 0; level < depth && len(frontier) > 0; level++ {
		// Counted as the loop goes, so every return below already carries the
		// levels this walk entered rather than the ones it was allowed.
		result.Depth = level + 1
		var next []traceNode
		for _, node := range frontier {
			if bounded.Err() != nil {
				return clipped(result, "time budget reached", request.Direction)
			}
			if result.Reads >= i.budget.MaxRequests {
				return clipped(result, "request budget reached", request.Direction)
			}
			facts, stale, err := i.factsForNode(node, request.Direction)
			result.Reads++
			result.Stale += stale
			if err != nil {
				result.Outcome = OutcomeFailed
				result.Reason = err.Error()
				return result
			}
			following := preferNamedFacts(facts, node, request.Direction)
			for _, fact := range following {
				if seenFacts[fact] {
					continue
				}
				seenFacts[fact] = true
				result.Facts = append(result.Facts, fact)
				if len(result.Facts) >= i.budget.MaxFacts {
					return clipped(result, "fact budget reached", request.Direction)
				}
				neighbour := traceNode{path: fact.To.Path, name: fact.To.Name, line: fact.To.Line}
				if request.Direction == DirectionCallers {
					neighbour = traceNode{path: fact.From.Path, name: fact.From.Name, line: fact.From.Line}
				}
				// A file-level hop says two files are connected and says nothing
				// about a symbol, so it does not change which name the walk is
				// following. Letting it go was what turned the reverse walk from one
				// method into every binding in the generated dispatch: at the file it
				// arrived at, "what reaches this symbol" became "what reaches this
				// file".
				if neighbour.name == "" {
					neighbour.name = node.name
				}
				if !seenNodes[neighbour] {
					seenNodes[neighbour] = true
					parents[neighbour] = traceParent{previous: node, fact: fact}
					next = append(next, neighbour)
				}
				if request.Direction == DirectionBetween && targetReached(neighbour, fact, request) {
					result.Facts = pathFacts(start, neighbour, parents)
					result.FileLevelHops = countFileLevelHops(result.Facts, request.Direction)
					return result
				}
			}
		}
		frontier = next
	}
	if request.Direction == DirectionBetween {
		result.Outcome = OutcomePartial
		result.Reason = fmt.Sprintf("no path from %s:%s to %s within %d hop(s)",
			request.Path, request.Name, request.TargetPath, result.Depth)
	}
	// Counted from the hops being returned, never from the hops examined on
	// the way: a `between` walk looks at hundreds of edges and prints three,
	// and a note describing the examination would put a number in front of a
	// reader that matches nothing they can see.
	result.FileLevelHops = countFileLevelHops(result.Facts, request.Direction)
	return result
}

func countFileLevelHops(facts []Fact, direction Direction) int {
	fileLevel := 0
	for _, fact := range facts {
		if factIsFileLevel(fact, direction) {
			fileLevel++
		}
	}
	return fileLevel
}

// projectSpelling returns the path as the project itself spells it, element by
// element, or the path unchanged when it cannot be found. Only ever called for
// a path the filesystem already accepted, so on a case-sensitive filesystem it
// is the identity — a wrong-case name there is a different file, and rewriting
// it to a name the caller did not write would be the tool inventing a path.
func (i *Indexer) projectSpelling(rel string) string {
	parts := strings.Split(rel, "/")
	resolved := make([]string, 0, len(parts))
	dir := i.root
	for _, part := range parts {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return rel
		}
		match := ""
		for _, entry := range entries {
			if entry.Name() == part {
				match = entry.Name()
				break
			}
			if match == "" && strings.EqualFold(entry.Name(), part) {
				match = entry.Name()
			}
		}
		if match == "" {
			return rel
		}
		resolved = append(resolved, match)
		dir = filepath.Join(dir, match)
	}
	return strings.Join(resolved, "/")
}

func (i *Indexer) factsForNode(node traceNode, direction Direction) ([]Fact, int, error) {
	if direction == DirectionCallers {
		return i.store.FactsTo(node.path)
	}
	return i.store.FactsFrom(node.path)
}

// factIsFileLevel says this hop is a statement about two files rather than
// about the name in hand: the near end carries no name of its own.
func factIsFileLevel(fact Fact, direction Direction) bool {
	if direction == DirectionCallers {
		return fact.To.Name == ""
	}
	return fact.From.Name == ""
}

// preferNamedFacts narrows a node's edges to the ones about the name in hand.
//
// A node is a (file, name) pair, and a file has edges for everything in it:
// `desktop/engine_forwarders_gen.go` has one per forwarder, 101 of them, and a
// walk that follows all of them buries the one hop that answers the question
// and spends the fact budget doing it. Three things can narrow that, best
// first:
//
//  1. an edge whose near end names this symbol — exact evidence about the name;
//  2. an edge written AT the line the walk arrived on, which is the only thing
//     left once the near end names nothing. A file's facts are per line, so the
//     line says which site in the file is in hand even when the fact cannot;
//  3. the file's every edge. Nothing has narrowed anything, which is where a
//     walk starts: the caller named a path and a name and no fact has brought
//     the reader anywhere yet, so the file's own edges are the whole answer.
func preferNamedFacts(facts []Fact, node traceNode, direction Direction) []Fact {
	var exact, atLine, structural []Fact
	for _, fact := range facts {
		location := fact.From
		if direction == DirectionCallers {
			location = fact.To
		}
		if location.Name == "" {
			if node.line > 0 && location.Line == node.line {
				atLine = append(atLine, fact)
				continue
			}
			structural = append(structural, fact)
			continue
		}
		if node.name != "" && location.Name == node.name {
			exact = append(exact, fact)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	if len(atLine) > 0 {
		return atLine
	}
	return structural
}

// targetReached answers "is this the place we were asked to arrive at".
//
// A hop that names nothing at its far end — an import — counts as arriving at
// the target FILE and nothing more specific, whichever name was asked for.
// Structural facts name no symbol at either end, so a path whose last hop is an
// import lands on the target file and no further. That is the honest reading —
// the caller sees the import hop and can see that it names no function — and it
// is better than refusing the only producer Wave A has, or reporting the miss as
// a depth problem when the depth was never the reason. The arriving fact is what
// says which of the two happened, because a node carries the name it is being
// read as rather than the name its file happens to have.
func targetReached(node traceNode, arriving Fact, request TraceRequest) bool {
	if node.path != request.TargetPath {
		return false
	}
	if request.TargetName == "" || node.name == request.TargetName {
		return true
	}
	return factIsFileLevel(arriving, request.Direction)
}

func pathFacts(start, target traceNode, parents map[traceNode]traceParent) []Fact {
	var reverse []Fact
	for target != start {
		parent, ok := parents[target]
		if !ok {
			return nil
		}
		reverse = append(reverse, parent.fact)
		target = parent.previous
	}
	facts := make([]Fact, len(reverse))
	for index := range reverse {
		facts[len(reverse)-1-index] = reverse[index]
	}
	return facts
}

func clipped(result Result, reason string, direction Direction) Result {
	result.Outcome = OutcomePartial
	result.Truncated = true
	result.Reason = reason
	// Counted here as well, and for the reason the count exists: a walk clipped
	// by the fact budget is the one most likely to be returning nothing but
	// file-level hops — 120 import edges of one generated forwarder on Aetox's
	// own tree — and leaving the count at zero drops the note that says so
	// exactly where the reader needs it most.
	result.FileLevelHops = countFileLevelHops(result.Facts, direction)
	return result
}

func joinReasons(existing, added string) string {
	if existing == "" {
		return added
	}
	return existing + "; " + added
}
