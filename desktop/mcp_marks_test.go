package main

import (
	"os"
	"regexp"
	"testing"
)

// The same guard providers_test.go puts on providerMarks.ts, on the other map.
//
// Two lists in two languages that have to agree and nothing comparing them: the
// servers ห้องสมุด offers live in mcpShelf.ts, their brand marks live in
// mcpMarks.ts, and a missing key is silent — the room draws a lettered tile,
// which is a legitimate state, so nothing looks broken. `codex` shipped as a
// grey "C" beside fourteen real marks for exactly that reason.
//
// Read from Go rather than asserted in vitest for the reason that test gives:
// the authority is the shelf, and a frontend test would have to restate it and
// could then drift from it.
//
// noMark is the escape hatch, and it is deliberately a list of names rather
// than a rule: a service with no published vector is a fact about that service,
// and writing it down here is what stops the next person assuming it was an
// oversight and going looking again.
//
// Empty since 2026-09-05: deepwiki, the one name that lived here, now carries a
// mark traced from its own PNG (see the note in mcpMarks.ts).
var noMark = map[string]string{}

func TestEveryShelfServerHasABrandMarkOrASayWhyNot(t *testing.T) {
	const (
		shelfPath = "frontend/src/lib/mcpShelf.ts"
		marksPath = "frontend/src/lib/mcpMarks.ts"
	)
	shelf, err := os.ReadFile(shelfPath)
	if err != nil {
		t.Fatalf("read %s: %v", shelfPath, err)
	}
	marksRaw, err := os.ReadFile(marksPath)
	if err != nil {
		t.Fatalf("read %s: %v", marksPath, err)
	}

	// `{ name: 'github', desc: ...` — the one shape every preset is written in.
	named := regexp.MustCompile(`\{\s*name:\s*'([a-z0-9-]+)'`)
	var presets []string
	for _, m := range named.FindAllStringSubmatch(string(shelf), -1) {
		presets = append(presets, m[1])
	}
	if len(presets) == 0 {
		t.Fatalf("parsed no presets out of %s — that file changed shape and this test is now blind", shelfPath)
	}

	// Keys sit at one indent and are followed by a backtick, quoted only when
	// the name is not a bare JavaScript identifier.
	// `  github: { plate: ...` — quoted only when the shelf name is not a bare
	// JavaScript identifier, which `cloudflare-docs` is not.
	keyed := regexp.MustCompile(`(?m)^  '?([a-z0-9-]+)'?: \{`)
	marks := make(map[string]bool)
	for _, m := range keyed.FindAllStringSubmatch(string(marksRaw), -1) {
		marks[m[1]] = true
	}
	if len(marks) == 0 {
		t.Fatalf("parsed no marks out of %s — that file changed shape and this test is now blind", marksPath)
	}

	for _, p := range presets {
		if marks[p] || noMark[p] != "" {
			continue
		}
		t.Errorf("shelf offers %q with no entry in %s and no line in noMark — it renders as a lettered tile "+
			"and nobody decided that", p, marksPath)
	}

	// The other direction, which is the one that rots quietly: a mark for a
	// server that left the shelf is dead weight nobody will ever see, and it
	// keeps a trademark in the binary for no reason at all.
	onShelf := make(map[string]bool, len(presets))
	for _, p := range presets {
		onShelf[p] = true
	}
	for name := range marks {
		if !onShelf[name] {
			t.Errorf("%s carries a mark for %q, which is not on the shelf any more", marksPath, name)
		}
	}
	for name := range noMark {
		if !onShelf[name] {
			t.Errorf("noMark still excuses %q, which is not on the shelf any more", name)
		}
	}
}
