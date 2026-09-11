package main

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// What this window lends the engine is the real packs — the seam changes
// nothing for the app as shipped (§248 A6, B1).
func TestThisWindowLendsTheRealPacks(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp()
	tools := appScreen{a}.WindowTools(stubSession{root: t.TempDir()})
	var browser, machine bool
	for _, s := range tools {
		switch s.Name() {
		case "browser":
			browser = true
		case "computer":
			machine = true
		}
	}
	if !browser || !machine {
		t.Fatalf("this window lends browser=%v machine=%v; want both", browser, machine)
	}
}

// The desk questions are answered without the engine ever holding a key: a
// provider that needs none answers, a closed port is found out, and a runtime
// that is not listening has nothing resident.
func TestTheScreenAnswersTheDeskQuestions(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp()
	var screen engine.Screen = appScreen{a}
	if got := screen.DefaultModel("aetox", ""); got == "" {
		t.Error("the built-in provider has a default model and the screen did not name it")
	}
	if _, err := screen.Probe("groq", "llama-test", "http://127.0.0.1:1", ""); err == nil {
		t.Error("a ping at a closed port succeeded")
	}
	if screen.ModelResident("ollama", "http://127.0.0.1:1", "x") {
		t.Error("a model is resident on a runtime that is not listening")
	}
}

type stubSession struct{ root string }

func (s stubSession) ID() string   { return "test-session" }
func (s stubSession) Root() string { return s.root }
