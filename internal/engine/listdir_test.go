package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDirListsFoldersOnlyWithHiddenOnesLast(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"src", ".git", "Docs", "zeta"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &Engine{}
	l, err := a.ListDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range l.Entries {
		names = append(names, e.Name)
	}
	want := []string{"Docs", "src", "zeta", ".git"}
	if len(names) != len(want) {
		t.Fatalf("entries = %v; want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("entries = %v; want %v", names, want)
		}
	}
	if !l.Entries[3].Hidden || l.Entries[0].Hidden {
		t.Error("hidden flag wrong")
	}
	if l.Parent != filepath.Dir(root) || l.Path != root {
		t.Errorf("path/parent = %q/%q", l.Path, l.Parent)
	}
	if l.Entries[1].Path != filepath.Join(root, "src") {
		t.Errorf("entry path = %q", l.Entries[1].Path)
	}
}

func TestListDirRefusesWhatIsNotAFolder(t *testing.T) {
	a := &Engine{}
	if _, err := a.ListDir(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("a missing folder listed")
	}
	f := filepath.Join(t.TempDir(), "file")
	_ = os.WriteFile(f, []byte("x"), 0o644)
	if _, err := a.ListDir(f); err == nil {
		t.Error("a file listed")
	}
	// Empty means home.
	l, err := a.ListDir("")
	if err != nil || l.Path != a.HomeDir() {
		t.Errorf("ListDir(\"\") = %q, %v; want home %q", l.Path, err, a.HomeDir())
	}
}
