package main

// The switch, tested where it actually decides something.
//
// It used to refuse the calls; now it removes the tool. The owner's words on
// 9 ก.ย.: "แยก tool เลยนะ ถ้าไม่เปิด ก็ไม่เอาระบบนี้". Both behaviours look the
// same from the settings page and they are not the same at all, so this is the
// difference written down:
//
//	refusing  — the model reads the block, believes it may drive the machine,
//	            plans around that, calls, and is told no. A wasted round trip
//	            and a plan built on a premise that was never true.
//	removing  — the model plans with what it has.
//
// And the block itself: ~280 tokens on every request of every session, for a
// capability that ships off.
//
// The removing is the engine's (workbenchSkills), whoever lends the tool, and
// internal/engine/screen_test.go is where that is checked. What is left here
// is the screen's half: how the switch reads, and that the tool itself never
// finds a way past it.

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// The setting is spelled so that ABSENT means what the product ships. config.go
// states the rule; this is the one place it can be checked against the actual
// zero value rather than against the comment.
func TestAbsentMeansOff(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	if computerControlOn() {
		t.Fatal("with no preference file at all, computer control reads as on")
	}
	var pref config.ModelPreference
	if pref.ComputerControlOn {
		t.Fatal("the zero value of the field is on, so an upgrade would grant it silently")
	}
}

// Every action, refused for the same reason, when the switch is off. Belt and
// braces on purpose: the tool is not registered, so this path should be
// unreachable in the app — but a profile or a test that constructs the skill
// directly must not find a way past the switch.
func TestTheToolRefusesEveryActionWhileOff(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	s := newComputerSkill(newTestApp(), nil)

	for _, call := range skill.PackedCalls(computerToolName) {
		out, err := s.ExecuteTool(t.Context(), map[string]any{"action": call.Action})
		if err == nil {
			t.Errorf("%s went through with the switch off", call.Action)
			continue
		}
		if out.Success {
			t.Errorf("%s reported success and an error at once", call.Action)
		}
		if !strings.Contains(out.Content, "ตั้งค่า") {
			t.Errorf("%s refused without saying where the switch is: %q", call.Action, out.Content)
		}
	}
}

// switchOnComputer turns the computer-control switch on for the duration of a
// test. It writes the preference file directly rather than calling the
// binding, because the binding rebuilds the engine, and the tests here build
// their registry by hand. The data root is a temp dir by then, so the file
// written here is thrown away with the test.
func switchOnComputer(t *testing.T) {
	t.Helper()
	if err := config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.ComputerControlOn = true
		return nil
	}); err != nil {
		t.Fatalf("could not turn computer control on for the test: %v", err)
	}
}
