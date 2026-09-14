package main

// --report <file>: one JSON line per turn, for whoever runs the console from
// a script and wants the numbers without opening aetox.db — the harness
// benchmark's second layer (harness-bench-2026-09-14.md §4) is exactly this
// list: rounds, tokens in/out/cached, cost, seconds, what was called. Appended,
// never truncated, so a run of many turns and a run of one look the same.

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Mikedev115/Aetox/internal/version"
)

// turnReport is the record of one turn.
type turnReport struct {
	Version  string  `json:"version"`
	Started  string  `json:"started"`
	Seconds  float64 `json:"seconds"`
	Outcome  string  `json:"outcome"` // done · cancelled · failed
	Error    string  `json:"error,omitempty"`
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Think    string  `json:"think,omitempty"`
	Approval string  `json:"approval"`
	Desk     string  `json:"desk"`
	Root     string  `json:"root"`
	Session  string  `json:"session"`
	Message  string  `json:"message"`
	Answer   string  `json:"answer"`
	// Rounds is how many times the model was called — one usage event each.
	Rounds int        `json:"rounds"`
	Tokens tokenTally `json:"tokens"`
	Cost   costTally  `json:"cost"`
	// Tools is every call the agent made, in order, sub-agents' included.
	Tools []toolCall `json:"tools"`
	// Questions is how many times the turn waited on the terminal — an
	// ask_user, or an approval under the ask gate.
	Questions int `json:"questions"`
}

type tokenTally struct {
	In     int `json:"in"`
	Out    int `json:"out"`
	Cached int `json:"cached"`
	// CacheReported is false when no round said anything about a cache —
	// then Cached is unknown, not zero.
	CacheReported bool `json:"cacheReported"`
}

type costTally struct {
	USD float64 `json:"usd"`
	// Priced is false when any round had no published rate — the sum is
	// then a floor, not the bill.
	Priced bool `json:"priced"`
}

type toolCall struct {
	Name    string `json:"name"`
	Act     string `json:"act,omitempty"`
	Subject string `json:"subject,omitempty"`
	// Parent is set for a call made inside a sub-agent.
	Parent string `json:"parent,omitempty"`
}

// tally is what the screen counts during a turn, under the screen's lock.
type tally struct {
	rounds    int
	tokens    tokenTally
	cost      costTally
	tools     []toolCall
	toolIndex map[string]int // ref → index in tools, so a later, fuller event updates the row
	questions int
}

func newTally() tally {
	return tally{cost: costTally{Priced: true}, toolIndex: map[string]int{}}
}

func (t *tally) round(in, out, cached int, cacheReported bool, usd float64, priced bool) {
	t.rounds++
	t.tokens.In += in
	t.tokens.Out += out
	t.tokens.Cached += cached
	t.tokens.CacheReported = t.tokens.CacheReported || cacheReported
	t.cost.USD += usd
	if !priced {
		t.cost.Priced = false
	}
}

func (t *tally) call(ref string, c toolCall) {
	if ref != "" {
		if i, ok := t.toolIndex[ref]; ok {
			if c.Act != "" || c.Subject != "" {
				t.tools[i] = c
			}
			return
		}
		t.toolIndex[ref] = len(t.tools)
	}
	t.tools = append(t.tools, c)
}

// appendReport writes one line. A report that cannot be written is said on
// stderr and does not fail the turn — the answer already went out.
func appendReport(path string, r turnReport) error {
	r.Version = version.Current
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func stamp(t time.Time) string { return t.Format(time.RFC3339) }
