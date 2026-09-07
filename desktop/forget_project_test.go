package main

// Taking a project off the list.
//
// The list only ever grew — touchProject inserts on every open and nothing ever
// deleted — so a folder opened once by mistake stayed in the column for good
// (owner, 7 ก.ย.: "ทำไมลบโปรเจกต์ไม่ได้หน้าโค้ด", about a Downloads row he had
// picked from the file panel an hour earlier).
//
// What these pin down is the size of the act: the row goes, and nothing else.

import (
	"testing"
)

func projectPaths(a *App) map[string]bool {
	out := map[string]bool{}
	for _, p := range a.RecentProjects() {
		out[p.RootPath] = true
	}
	return out
}

func TestForgetProjectRemovesOnlyThatRow(t *testing.T) {
	a := newJobApp(t)
	keep, drop := t.TempDir(), t.TempDir()
	a.touchProject(keep)
	a.touchProject(drop)

	if _, err := a.ForgetProject(drop); err != nil {
		t.Fatalf("ForgetProject: %v", err)
	}
	left := projectPaths(a)
	if left[drop] {
		t.Errorf("%s is still on the list", drop)
	}
	if !left[keep] {
		t.Errorf("%s left the list too", keep)
	}
}

// Forgetting the folder you are standing in has to step out of it as well, or
// the app is running inside a project its own list no longer names and the only
// way back is the folder dialog.
func TestForgettingTheOpenProjectStepsOutOfIt(t *testing.T) {
	a := newJobApp(t)
	root := t.TempDir()
	// The engine's own root, set the way a bootstrapped session would have it:
	// retargetTemplate only moves what the NEXT chat is born with, and applyConfig
	// (which needs a real engine) is what copies it here.
	a.cur().cfg.SandboxRoot = root
	a.takeProject()
	a.touchProject(root)

	status, err := a.ForgetProject(root)
	if err != nil {
		t.Fatalf("ForgetProject: %v", err)
	}
	if a.projectFocused || status.Focused {
		t.Error("still focused on a project that is no longer on the list")
	}
	if projectPaths(a)[root] {
		t.Error("the row survived")
	}
}

// Another project's row is not touched by stepping out of this one, and neither
// is the standing of the app when the folder being forgotten is not the open
// one.
func TestForgettingAnotherProjectLeavesThisOneOpen(t *testing.T) {
	a := newJobApp(t)
	here, elsewhere := t.TempDir(), t.TempDir()
	a.cur().cfg.SandboxRoot = here
	a.takeProject()
	a.touchProject(here)
	a.touchProject(elsewhere)

	if _, err := a.ForgetProject(elsewhere); err != nil {
		t.Fatalf("ForgetProject: %v", err)
	}
	if !a.projectFocused {
		t.Error("forgetting another project dropped focus on this one")
	}
	if got := a.cur().cfg.SandboxRoot; got != here {
		t.Errorf("the engine moved to %q", got)
	}
}

func TestForgetProjectNeedsAPath(t *testing.T) {
	a := newJobApp(t)
	if _, err := a.ForgetProject("   "); err == nil {
		t.Error("an empty path was accepted")
	}
}
