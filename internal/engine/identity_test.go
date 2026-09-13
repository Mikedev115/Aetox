package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// The identity layer is one folder per head since 14 ก.ย. 2026 (§266): what
// these pin is that the split loses nobody a file, and that the two heads
// never read each other's.

func identityEngine(t *testing.T) *Engine {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	return &Engine{ctx: context.Background(), emit: func(string, ...any) {}}
}

// The flat set both heads shared until the split is copied into BOTH heads'
// folders on first use, and the flat files are gone afterwards: neither head
// reads a word less than it did the day before, and there is one place a
// file can be.
func TestFlatIdentitySetIsGivenToBothHeadsOnce(t *testing.T) {
	a := identityEngine(t)
	root, _ := os.LookupEnv("AETOX_DATA_ROOT")
	flat := filepath.Join(root, "identity")
	if err := os.MkdirAll(flat, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flat, "identity.md"), []byte("be kind"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flat, "notes.txt"), []byte("not a layer"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, head := range []string{"assistant", "coding"} {
		files, err := a.ListIdentityFiles(head)
		if err != nil {
			t.Fatalf("%s: %v", head, err)
		}
		if len(files) != 1 || files[0].Name != "identity.md" {
			t.Fatalf("%s got %v, want the migrated identity.md", head, files)
		}
		if text, _ := a.ReadIdentityFile(head, "identity.md"); text != "be kind" {
			t.Fatalf("%s read %q", head, text)
		}
	}
	if _, err := os.Stat(filepath.Join(flat, "identity.md")); !os.IsNotExist(err) {
		t.Fatalf("flat identity.md still there after the split")
	}
	if _, err := os.Stat(filepath.Join(flat, "notes.txt")); err != nil {
		t.Fatalf("a non-markdown file in the folder was touched: %v", err)
	}
}

// A file saved on one head is that head's alone; the other head still has
// what it had. And a head nobody has is refused, not created.
func TestIdentityFilesAreKeptPerHead(t *testing.T) {
	a := identityEngine(t)
	if err := a.SaveIdentityFile("coding", "identity.md", "you write code"); err != nil {
		t.Fatal(err)
	}
	if text, _ := a.ReadIdentityFile("assistant", "identity.md"); text != "" {
		t.Fatalf("assistant read the coder's file: %q", text)
	}
	files, _ := a.ListIdentityFiles("assistant")
	if len(files) != 0 {
		t.Fatalf("assistant lists %v, want nothing", files)
	}
	if err := a.DeleteIdentityFile("coding", "identity.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ListIdentityFiles("editor"); err == nil {
		t.Fatal("an unknown head was accepted")
	}
	if _, err := a.ReadIdentityFile("coding", "../secrets"); err == nil {
		t.Fatal("path traversal was accepted")
	}
}

// The pre-2026-07 single file is still honoured, into both heads.
func TestLegacyAetoxMdBecomesEveryHeadsContext(t *testing.T) {
	a := identityEngine(t)
	root, _ := os.LookupEnv("AETOX_DATA_ROOT")
	if err := os.WriteFile(filepath.Join(root, "AETOX.md"), []byte("old brief"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, head := range []string{"assistant", "coding"} {
		if text, err := a.ReadIdentityFile(head, "context.md"); err != nil || text != "old brief" {
			t.Fatalf("%s context.md = %q, %v", head, text, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "AETOX.md")); !os.IsNotExist(err) {
		t.Fatal("AETOX.md left behind")
	}
}
