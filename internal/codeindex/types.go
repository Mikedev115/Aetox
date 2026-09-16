// Package codeindex stores evidence-backed relationships derived from source code.
// Every fact names the source locations that support it and the analyzer that
// produced it. The source tree remains authoritative; the database is a cache.
package codeindex

import "time"

// Strength says how directly a fact is supported.
type Strength string

const (
	StrengthSyntactic Strength = "syntactic"
	StrengthResolved  Strength = "resolved"
	StrengthPossible  Strength = "possible"
	StrengthObserved  Strength = "observed"
	StrengthHeuristic Strength = "heuristic"
)

// Relation is the kind of connection between two source locations.
type Relation string

const (
	RelationImport        Relation = "import"
	RelationCall          Relation = "call"
	RelationReference     Relation = "reference"
	RelationWailsRPC      Relation = "wails_rpc"
	RelationTestReference Relation = "test_reference"
)

// Location identifies one source symbol or site. Lines are 1-based.
type Location struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	EndLine int    `json:"endLine,omitempty"`
	Name    string `json:"name,omitempty"`
}

// Evidence is the source range and producer behind a fact.
type Evidence struct {
	Location Location `json:"location"`
	Strength Strength `json:"strength"`
	Producer string   `json:"producer"`
}

// Fact is one directional relationship in the code graph.
type Fact struct {
	From     Location `json:"from"`
	To       Location `json:"to"`
	Relation Relation `json:"relation"`
	Strength Strength `json:"strength"`
	Producer string   `json:"producer"`
}

// Evidence returns the two source anchors carried by a fact.
func (f Fact) Evidence() []Evidence {
	return []Evidence{
		{Location: f.From, Strength: f.Strength, Producer: f.Producer},
		{Location: f.To, Strength: f.Strength, Producer: f.Producer},
	}
}

// Outcome distinguishes a complete answer from a missing or clipped one.
type Outcome string

const (
	OutcomeComplete Outcome = "complete"
	// OutcomePartial means a cap was reached. The facts returned are still
	// valid; only the walk is short, and Reason names what stopped it.
	OutcomePartial Outcome = "partial"
	// OutcomeUnsupported means this analyzer does not read this kind of project.
	OutcomeUnsupported Outcome = "unsupported"
	// OutcomeUnavailable means a piece the answer depends on is not there: an
	// analyzer that cannot run, or a path that does not exist. It is the answer
	// for a misspelled file, and it must never be reported as "no relationships"
	// — an empty answer and an unchecked one are different statements.
	OutcomeUnavailable Outcome = "unavailable"
	OutcomeFailed      Outcome = "failed"
)

// Direction controls which way Trace walks the fact graph.
type Direction string

const (
	DirectionCallers Direction = "callers"
	DirectionCallees Direction = "callees"
	DirectionBetween Direction = "between"
)

// Budget applies once to the whole trace, never once per edge.
type Budget struct {
	MaxRequests int
	MaxFacts    int
	MaxDepth    int
	MaxDuration time.Duration
}

// DefaultBudget is the bounded surface Wave A exposes.
func DefaultBudget() Budget {
	return Budget{
		MaxRequests: 40,
		MaxFacts:    120,
		MaxDepth:    3,
		MaxDuration: 8 * time.Second,
	}
}

// DefaultDepth is how many hops a direction follows when the caller has no
// opinion, and it lives here rather than in the tool because it is a fact about
// the traversal, not about the model's schema.
//
// It was in the tool for one afternoon, and the split produced a real bug the
// first time the API was called directly: `between` asks for a path to a named
// place and one hop almost never arrives, while `callers` and `callees`
// multiply per level. Two places deciding the same default is two answers to
// one question, and the caller with the schema is not the one that can be
// wrong about the graph.
//
// One hop was the default for the first day, and it was one hop short of the
// relationship this act exists for: a frontend call reaches a generated Wails
// binding, and the binding is what reaches the Go method. Asked for one hop,
// the answer ended inside a file nobody wrote — which is exactly where `symbol`
// already ends — so the one question no other act can answer came back looking
// like the one every other act does.
//
// Two was the fix that afternoon, and on this project it is still short. Run
// against Aetox's own tree on 2026-09-16, `callees` at two hops from
// `desktop/frontend/src/lib/workbench/RepoMapPane.svelte` answers
// `desktop/engine_forwarders_gen.go:366` — a forwarder nobody wrote by hand —
// and the file that declares the method is the third hop. A project that
// generates one layer instead of two crosses in fewer, and a default cannot
// know in advance which of the two it is looking at. The honest one is the
// deepest the budget allows: a walk stops on its own when a level has no edges
// left, so depth it does not need costs nothing, while one hop less than the
// crossing costs the whole answer.
//
// It takes no direction because it no longer answers differently by direction.
// A caller that wants a shallower, cheaper answer asks for one depth.
func DefaultDepth() int {
	return DefaultBudget().MaxDepth
}

// TraceRequest identifies one source and the direction to traverse.
type TraceRequest struct {
	Path       string
	Name       string
	Direction  Direction
	TargetPath string
	TargetName string
	Depth      int
}

// Result is returned by indexing and tracing operations.
type Result struct {
	Outcome Outcome
	Facts   []Fact
	// Unknown counts relationships a producer could see but could not
	// establish. It is reported, never dropped: a hop nobody could prove is a
	// gap in the answer, and a gap that is not counted reads as absent.
	Unknown int
	// Stale counts cached rows dropped because an endpoint no longer matches
	// what was recorded when they were written.
	Stale    int
	Requests int
	// Depth is how many levels the walk actually entered: the clamp against the
	// budget decides the ceiling, and the walk stops before it whenever a level
	// has no edges left. The caller renders this and not what it was allowed: a
	// header stating three hops over a walk of one is the same lie as a cap that
	// does not say it bit.
	Depth int
	// FileLevelHops counts the hops below that are about the FILE rather than
	// about the name that was asked about: an import is written by a file and
	// names no symbol, so following one answers "these files are connected"
	// and not "this name reaches that one". A caller that does not say which
	// of the two it is reporting has answered a different question.
	FileLevelHops int
	// Sources is how many files a full-project producer read.
	Sources int
	// Rebuilt says the index was built or rebuilt for this answer. A question
	// about an unchanged project answers without it, and a caller reading the
	// cost of a trace needs to tell those two apart.
	Rebuilt bool
	// Reads is how many cache reads the walk spent, which is the unit the
	// request budget bounds.
	Reads     int
	Truncated bool
	Reason    string
}
