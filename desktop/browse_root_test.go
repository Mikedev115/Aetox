package main

// Looking at a folder must stay smaller than opening one.
//
// The bug this comes from was not that the file tree could show a folder — it
// was everything else the one available button did on the way (owner, 7 ก.ย.:
// "หน้าผู้ช่วยหลักอันนี้ไม่ควรจำนะครับ ผมกดเลือกแล้วมันจำตำแหน่งนี้เลย"). So the
// tests are mostly about what does NOT happen: the engine does not move, the
// recent-projects table does not grow, and the folder does not survive a focus
// switch.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// browsingApp is unfocused, the way the ผู้ช่วย door runs.
func browsingApp(t *testing.T) (*App, string) {
	t.Helper()
	home := t.TempDir()
	return seed(&App{cfg: config.Config{SandboxRoot: home}}, newConversation()), home
}

func TestNoProjectAndNoBrowsingIsAnEmptyTree(t *testing.T) {
	a, _ := browsingApp(t)
	if got := a.ProjectTree(); len(got) != 0 {
		t.Errorf("ProjectTree() = %v, want empty", got)
	}
	if got := a.BrowseRoot(); got != "" {
		t.Errorf("BrowseRoot() = %q, want empty", got)
	}
}

func TestBrowsingShowsTheFolderWithoutOpeningIt(t *testing.T) {
	a, home := browsingApp(t)
	looked := t.TempDir()
	if err := os.MkdirAll(filepath.Join(looked, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(looked, "notes", "one.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.browseRoot = looked // what the dialog would have set

	nodes := treeByPath(a.ProjectTree())
	if _, ok := nodes["notes/one.md"]; !ok {
		t.Fatalf("the browsed folder's files are not in the tree: %+v", nodes)
	}
	if a.projectFocused {
		t.Error("browsing focused a project — the assistant's reach is OpenSandbox: !projectFocused")
	}
	if root := a.cur().cfg.SandboxRoot; root != home {
		t.Errorf("the engine moved to %q; browsing must not touch it", root)
	}
	if got := a.BrowseRoot(); got != looked {
		t.Errorf("BrowseRoot() = %q, want %q", got, looked)
	}
}

func TestStopBrowsingEmptiesTheTree(t *testing.T) {
	a, _ := browsingApp(t)
	a.browseRoot = t.TempDir()
	a.StopBrowsing()
	if got := a.BrowseRoot(); got != "" {
		t.Errorf("BrowseRoot() = %q after StopBrowsing", got)
	}
	if got := a.ProjectTree(); len(got) != 0 {
		t.Errorf("ProjectTree() = %v, want empty", got)
	}
}

// The two ways focus changes, and neither may carry the browsed folder across.
// A view that outlives the mode it belongs to is the "remembering" all of this
// is here to prevent, one step quieter.
func TestFocusSwitchesDropTheBrowsedFolder(t *testing.T) {
	a, _ := browsingApp(t)
	a.browseRoot = t.TempDir()
	a.takeProject()
	if a.browseRoot != "" {
		t.Errorf("taking a project kept browseRoot = %q", a.browseRoot)
	}

	a.projectFocused = false
	a.browseRoot = t.TempDir()
	a.focusNone()
	if a.browseRoot != "" {
		t.Errorf("clearing focus kept browseRoot = %q", a.browseRoot)
	}
}

// A focused project already has an answer to "what am I looking at", and the
// refusal is what keeps the tree from having two.
func TestBrowseFolderRefusesWhileAProjectIsOpen(t *testing.T) {
	a, _ := browsingApp(t)
	a.projectFocused = true
	if _, err := a.BrowseFolder(); err == nil {
		t.Error("BrowseFolder() was allowed while a project is focused")
	}
}

// BrowseRoot answers for the panel, and the panel only draws its "just looking"
// bar when there is no project. A stale string from an earlier roam must not
// make a project's tree claim to be one.
func TestBrowseRootIsSilentWhileFocused(t *testing.T) {
	a, _ := browsingApp(t)
	a.browseRoot = t.TempDir()
	a.projectFocused = true
	if got := a.BrowseRoot(); got != "" {
		t.Errorf("BrowseRoot() = %q while focused, want empty", got)
	}
}
