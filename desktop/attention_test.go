package main

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// All three ship on: the signal was built because a question went unannounced,
// and a signal that has to be found and switched on first announces nothing.
func TestAttentionShipsAllOn(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	got := layersByID(t, (&App{}).AttentionSignal())
	if len(got) != 3 {
		t.Fatalf("want three switches, got %d", len(got))
	}
	for _, id := range []string{attentionToast, attentionFlash, attentionChime} {
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
	if !got[attentionFlash].On || !got[attentionToast].On {
		t.Error("turning the chime off took another switch with it")
	}

	again := layersByID(t, (&App{}).AttentionSignal())
	if again[attentionChime].On || !again[attentionFlash].On || !again[attentionToast].On {
		t.Error("a fresh read did not see what was written")
	}

	after := layersByID(t, app.SetAttentionSignal("nosuchswitch", true))
	if after[attentionChime].On || !after[attentionFlash].On || !after[attentionToast].On {
		t.Error("an unknown id moved a switch")
	}

	toast := layersByID(t, app.SetAttentionSignal(attentionToast, false))
	if toast[attentionToast].On || !toast[attentionFlash].On {
		t.Error("the notification switch is not its own")
	}
}

// A process with no main window is never "away": there is nothing to come
// back to, so nothing is asked of the OS and the answer is false — the window
// then falls back on its own focus guess. Pinned with every switch off as
// well, so the call is known to return without reaching for a window on any
// platform.
func TestRequestAttentionWithNoWindowIsNotAway(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	app := &App{}
	if app.RequestAttention("done", "แชต", "ทำงานเสร็จแล้ว") {
		t.Error("a test process with no window reported the user as away")
	}
	app.SetAttentionSignal(attentionFlash, false)
	app.SetAttentionSignal(attentionToast, false)
	if app.RequestAttention("ask", "", "") {
		t.Error("with every switch off the answer changed")
	}
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
