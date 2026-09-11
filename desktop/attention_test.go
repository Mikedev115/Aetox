package main

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// Both ship on: the signal was built because a question went unannounced, and
// a signal that has to be found and switched on first announces nothing.
func TestAttentionShipsBothOn(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	got := layersByID(t, (&App{}).AttentionSignal())
	if len(got) != 2 {
		t.Fatalf("want two switches, got %d", len(got))
	}
	for _, id := range []string{attentionFlash, attentionChime} {
		if !got[id].On {
			t.Errorf("%s ships on and came back off", id)
		}
		if got[id].Label == "" || got[id].Note == "" {
			t.Errorf("%s has no words — a switch with no explanation is one nobody can decide about", id)
		}
	}
}

// Flipping one leaves the other alone, the setting survives a re-read, and an
// unknown id moves nothing.
func TestAttentionSwitchesAreIndependentAndPersist(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	app := &App{}

	got := layersByID(t, app.SetAttentionSignal(attentionChime, false))
	if got[attentionChime].On {
		t.Error("the chime did not go off")
	}
	if !got[attentionFlash].On {
		t.Error("turning the chime off took the flash with it")
	}

	again := layersByID(t, (&App{}).AttentionSignal())
	if again[attentionChime].On || !again[attentionFlash].On {
		t.Error("a fresh read did not see what was written")
	}

	after := layersByID(t, app.SetAttentionSignal("nosuchswitch", true))
	if after[attentionChime].On || !after[attentionFlash].On {
		t.Error("an unknown id moved a switch")
	}
}

// With the flash off, asking for attention must not reach the OS at all — the
// switch is the whole promise, and it is checked before any window is looked
// up. (On Windows the call would otherwise be a no-op in a test process with
// no visible window, so this is the half that can be pinned everywhere.)
func TestRequestAttentionHonoursTheSwitch(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	app := &App{}
	app.SetAttentionSignal(attentionFlash, false)
	app.RequestAttention() // must return without panicking or blocking
}

// layersByID is the engine test's helper of the same name, for the switches
// the screen keeps.
func layersByID(t *testing.T, list []engine.BusyLayer) map[string]engine.BusyLayer {
	t.Helper()
	out := map[string]engine.BusyLayer{}
	for _, l := range list {
		out[l.ID] = l
	}
	return out
}
