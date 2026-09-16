package codeindex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreKeepsOnlyFactsBothEndpointsStillMatch(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "a.go", "package demo\n\nfunc A() { B() }\n")
	writeFile(t, root, "b.go", "package demo\n\nfunc B() {}\n")

	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	fact := Fact{
		From:     Location{Path: "a.go", Line: 3, Name: "A"},
		To:       Location{Path: "b.go", Line: 3, Name: "B"},
		Relation: RelationCall,
		Strength: StrengthResolved,
		Producer: "test-analyzer",
	}
	if err := store.ReplaceProducer("test-analyzer", "1", 0, map[string][]Fact{"a.go": {fact}}); err != nil {
		t.Fatal(err)
	}
	store.Close()

	store, err = Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	got, stale, err := store.FactsBetween("a.go", "b.go")
	if err != nil {
		t.Fatal(err)
	}
	if stale != 0 || len(got) != 1 || got[0] != fact {
		t.Fatalf("fresh facts = %+v, stale = %d", got, stale)
	}

	// The TARGET changing is enough: a rename at the far end moves the line
	// this fact points at, even though the source file is untouched.
	writeFile(t, root, "b.go", "package demo\n\nfunc B(value int) {}\n")
	got, stale, err = store.FactsBetween("a.go", "b.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 || stale != 1 {
		t.Fatalf("a changed target returned %+v, stale = %d", got, stale)
	}
}

// A file whose facts are all gone — its imports removed, or the file deleted —
// must stop having rows. A per-file replace cannot do that: with no facts to
// write there is nothing to replace, and the row it left behind is then counted
// as staleness on every read for the life of the cache.
func TestReplaceProducerDropsRowsForSourcesThatStoppedProducing(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "a.go", "package demo\n")
	writeFile(t, root, "b.go", "package demo\n")
	writeFile(t, root, "c.go", "package demo\n")

	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	first := Fact{From: Location{Path: "a.go", Line: 1}, To: Location{Path: "b.go"}, Relation: RelationImport, Strength: StrengthSyntactic, Producer: "repomap"}
	second := Fact{From: Location{Path: "c.go", Line: 1}, To: Location{Path: "b.go"}, Relation: RelationImport, Strength: StrengthSyntactic, Producer: "repomap"}
	if err := store.ReplaceProducer("repomap", "1", 0, map[string][]Fact{"a.go": {first}, "c.go": {second}}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceProducer("repomap", "1", 0, map[string][]Fact{"c.go": {second}}); err != nil {
		t.Fatal(err)
	}

	left, stale, err := store.FactsFrom("a.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 || stale != 0 {
		t.Fatalf("a.go = %+v stale=%d, want the row gone rather than stale", left, stale)
	}
}

func TestReplaceProducerLeavesOtherAnalyzersAlone(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "a.go", "package demo\n")
	writeFile(t, root, "b.go", "package demo\n")
	writeFile(t, root, "c.go", "package demo\n")
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	structural := Fact{From: Location{Path: "a.go", Line: 1}, To: Location{Path: "b.go"}, Relation: RelationImport, Strength: StrengthSyntactic, Producer: "repomap"}
	bridge := Fact{From: Location{Path: "a.go", Line: 1}, To: Location{Path: "c.go"}, Relation: RelationWailsRPC, Strength: StrengthResolved, Producer: "wails"}
	if err := store.ReplaceProducer("repomap", "1", 0, map[string][]Fact{"a.go": {structural}}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceProducer("wails", "1", 0, map[string][]Fact{"a.go": {bridge}}); err != nil {
		t.Fatal(err)
	}

	facts, stale, err := store.FactsFrom("a.go")
	if err != nil {
		t.Fatal(err)
	}
	if stale != 0 || len(facts) != 2 {
		t.Fatalf("facts = %+v, stale = %d", facts, stale)
	}
}

// The paths are a trust boundary: a fact whose far end points outside the
// project is refused before anything is written, whether it arrived as the
// source or as the destination.
func TestStoreRefusesPathsOutsideTheProject(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "a.go", "package demo\n")
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	good := Location{Path: "a.go", Line: 1}
	cases := map[string]Fact{
		"as the source": {From: Location{Path: "../outside.go", Line: 1}, To: good, Relation: RelationImport, Strength: StrengthSyntactic, Producer: "p"},
		"as the target": {From: good, To: Location{Path: "../outside.go"}, Relation: RelationImport, Strength: StrengthSyntactic, Producer: "p"},
	}
	for name, fact := range cases {
		bySource := map[string][]Fact{"a.go": {fact}}
		if fact.From.Path != "a.go" {
			bySource = map[string][]Fact{"../outside.go": {fact}}
		}
		if err := store.ReplaceProducer("p", "1", 0, bySource); err == nil {
			t.Errorf("%s: a path outside the project was accepted", name)
		}
	}
}

// A producer hands over its whole reading at once. An empty list for a file it
// did read is a producer bug, and saying so is cheaper than a silent row.
func TestReplaceProducerRefusesAnEmptySourceEntry(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "a.go", "package demo\n")
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.ReplaceProducer("p", "1", 0, map[string][]Fact{"a.go": {}}); err == nil {
		t.Fatal("an empty entry was accepted")
	}
	if err := store.ReplaceProducer("  ", "1", 0, map[string][]Fact{"a.go": {{From: Location{Path: "a.go", Line: 1}, To: Location{Path: "a.go", Line: 1}, Relation: RelationImport, Strength: StrengthSyntactic, Producer: "p"}}}); err == nil {
		t.Fatal("a blank producer was accepted")
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A row written by one build of an analyzer is not the answer of another. The
// facts themselves cannot say so — they carry the content stamps of their two
// endpoints and nothing about the code that decided they exist — which makes the
// version stored beside them the whole of the difference between a reading taken
// by this build and the same reading taken by the build before it.
func TestStoreDistinguishesTheVersionThatWroteItsRows(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	root := t.TempDir()
	writeFile(t, root, "a.go", "package demo\n")
	writeFile(t, root, "b.go", "package demo\n")
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	fact := Fact{
		From: Location{Path: "a.go", Line: 1}, To: Location{Path: "b.go"},
		Relation: RelationImport, Strength: StrengthSyntactic, Producer: "p",
	}
	if err := store.ReplaceProducer("p", "2", 0, map[string][]Fact{"a.go": {fact}}); err != nil {
		t.Fatal(err)
	}
	if done, err := store.Indexed("p", "2"); err != nil || !done {
		t.Fatalf("the version that wrote the rows reads as indexed: done=%v err=%v", done, err)
	}
	if done, err := store.Indexed("p", "1"); err != nil || done {
		t.Fatalf("a row from another build answered as this one's: done=%v err=%v", done, err)
	}
}
