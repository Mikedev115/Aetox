package main

// Coming back to the project you were in.
//
// The window remembered the desk it was last at (config.ModelPreference.LastDesk)
// and it always remembered which projects had been OPENED — touchProject, the
// sidebar's list. What nothing remembered was which one the app was standing in,
// so leaving the app mid-project and typing the next instruction on the way back
// landed it on no project at all (owner, 11 ก.ย.).
//
// What these pin down is the size of that memory: the folder comes back, a launch
// with nothing remembered stays outside every project, a remembered folder that
// is gone is forgotten rather than refused, and stepping out is written down as
// deliberately as stepping in.

import (
	"os"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// lastRemembered is the preference as the next launch would read it.
func lastRemembered(t *testing.T) string {
	t.Helper()
	pref, _, err := config.LoadModelPreference()
	if err != nil {
		t.Fatalf("LoadModelPreference: %v", err)
	}
	return pref.LastProject
}

func TestALaunchComesBackToTheProjectItLeftOffIn(t *testing.T) {
	a := newJobApp(t)
	root := t.TempDir()
	rememberProject(root)

	if !a.openAtRememberedProject() {
		t.Fatal("the remembered project was not opened")
	}
	if !a.projectFocused {
		t.Error("the app came back with no project focused")
	}
	if got := a.cfg.SandboxRoot; got != root {
		t.Errorf("the next chat is born in %q, not in %q", got, root)
	}
	// Standing in a folder is what the sidebar's list is a list of, so the folder
	// it came back to has to be on it.
	if !projectPaths(a)[a.cfg.SandboxRoot] {
		t.Error("the project it came back to is missing from the sidebar's list")
	}
}

func TestALaunchWithNothingRememberedStaysOutsideEveryProject(t *testing.T) {
	a := newJobApp(t)
	if got := lastRemembered(t); got != "" {
		t.Fatalf("a fresh install already has a project remembered: %q", got)
	}

	if a.openAtRememberedProject() {
		t.Error("opened a project nobody had chosen")
	}
	if a.projectFocused || a.cfg.SandboxRoot != "" {
		t.Errorf("a launch with nothing remembered came back standing in %q", a.cfg.SandboxRoot)
	}
}

func TestARememberedFolderThatIsGoneIsForgotten(t *testing.T) {
	a := newJobApp(t)
	root := t.TempDir()
	rememberProject(root)
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("removing the project folder: %v", err)
	}

	if a.openAtRememberedProject() {
		t.Error("opened a folder that is not there any more")
	}
	if a.projectFocused {
		t.Error("focused on a project it could not open")
	}
	// Forgotten by the function that found it dead, not by whichever fallback the
	// caller happens to run: otherwise every start from here on asks the same
	// question and answers it the same way.
	if got := lastRemembered(t); got != "" {
		t.Errorf("the dead folder is still remembered as %q", got)
	}
}

func TestLeavingAProjectIsRememberedAsNoProject(t *testing.T) {
	a := newJobApp(t)
	root := t.TempDir()
	a.cur().cfg.SandboxRoot = root
	a.takeProject()
	a.enterProject(root)
	if got := lastRemembered(t); got != root {
		t.Fatalf("standing in %q was remembered as %q", root, got)
	}

	if _, err := a.ClearProjectFocus(); err != nil {
		t.Fatalf("ClearProjectFocus: %v", err)
	}
	if got := lastRemembered(t); got != "" {
		t.Errorf("stepping out was remembered as %q — the next launch would drag the user back in", got)
	}
}

func TestForgettingAProjectAlsoForgetsBeingInIt(t *testing.T) {
	a := newJobApp(t)
	root := t.TempDir()
	a.enterProject(root)
	if got := lastRemembered(t); got != root {
		t.Fatalf("opening %q remembered %q", root, got)
	}

	if _, err := a.ForgetProject(root); err != nil {
		t.Fatalf("ForgetProject: %v", err)
	}
	if got := lastRemembered(t); got != "" {
		t.Errorf("a forgotten project is still what the next launch opens (%q)", got)
	}
}

func TestForgettingAnotherProjectKeepsThisOneRemembered(t *testing.T) {
	a := newJobApp(t)
	here, other := t.TempDir(), t.TempDir()
	a.enterProject(here)
	a.touchProject(other)

	if _, err := a.ForgetProject(other); err != nil {
		t.Fatalf("ForgetProject: %v", err)
	}
	if got := lastRemembered(t); got != here {
		t.Errorf("forgetting another project moved the next launch to %q", got)
	}
}

// The call site, not the helper: a door that opens a project and forgets to write
// the memory down would leave the feature working in every test above and not in
// the app.
func TestOpeningAProjectPathRemembersIt(t *testing.T) {
	a := newJobApp(t)
	root := t.TempDir()

	if _, err := a.OpenProjectPath(root); err != nil {
		t.Fatalf("OpenProjectPath: %v", err)
	}
	if got := lastRemembered(t); got != root {
		t.Errorf("opening %q remembered %q", root, got)
	}
}
