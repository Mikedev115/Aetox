package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The coding desk's projects go under a visible folder in the home directory,
// apart from the assistant desk's spaces in DataRoot — the owner asked that
// the two never share a place (คนละชั้นคนละตำแหน่ง).
func TestCreateCodeProjectLandsUnderHomeNotDataRoot(t *testing.T) {
	home := t.TempDir()
	data := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("AETOX_DATA_ROOT", data)

	a := &Engine{}
	if got, want := a.CodeProjectsDir(), filepath.Join(home, codeProjectsDirName); got != want {
		t.Fatalf("default dir = %q, want %q", got, want)
	}
	path, err := a.CreateCodeProject("ร้านกาแฟ-api")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if want := filepath.Join(home, codeProjectsDirName, "ร้านกาแฟ-api"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		t.Errorf("folder was not made: %v", err)
	}
	if _, err := os.Stat(filepath.Join(path, "context")); err == nil {
		t.Error("a coding project must not carry the assistant's context/ folder")
	}
	if strings.HasPrefix(path, data) {
		t.Errorf("landed inside DataRoot %q", data)
	}
	// A second project of the same name is refused, not reused.
	if _, err := a.CreateCodeProject("ร้านกาแฟ-api"); err == nil {
		t.Error("a taken name must be refused")
	}
}

// Same name rules as a space: what cannot be a folder cannot be a project.
func TestCreateCodeProjectRefusesBadNames(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := &Engine{}
	for _, name := range []string{"", "  ", "..", "a/b", `a\b`, "x:y", strings.Repeat("ย", 65)} {
		if _, err := a.CreateCodeProject(name); err == nil {
			t.Errorf("name %q was accepted", name)
		}
	}
}

// The chosen parent is remembered and used for the next project; empty puts
// the default back. Projects already made are not moved.
func TestSetCodeProjectsDirIsTheParentOfTheNextProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := &Engine{}

	chosen := t.TempDir()
	if err := a.SetCodeProjectsDir(chosen); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := a.CodeProjectsDir(); got != filepath.Clean(chosen) {
		t.Fatalf("dir = %q, want %q", got, chosen)
	}
	path, err := a.CreateCodeProject("one")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if filepath.Dir(path) != filepath.Clean(chosen) {
		t.Errorf("project went to %q, not under %q", path, chosen)
	}
	if err := a.SetCodeProjectsDir("relative/dir"); err == nil {
		t.Error("a relative parent must be refused")
	}
	if err := a.SetCodeProjectsDir(""); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if got, want := a.CodeProjectsDir(), filepath.Join(home, codeProjectsDirName); got != want {
		t.Errorf("after reset dir = %q, want %q", got, want)
	}
}
