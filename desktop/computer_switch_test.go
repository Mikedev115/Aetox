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

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/skill"
)

func holdsComputer(a *App, t *testing.T) bool {
	t.Helper()
	for _, s := range a.workbenchSkills(a.cur(), t.TempDir()) {
		if s.Name() == computerToolName {
			return true
		}
	}
	return false
}

func TestTheModelIsNotGivenTheToolUntilItIsSwitchedOn(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	app := seed(&App{cfg: config.Config{SandboxRoot: t.TempDir()}}, newConversation())

	// A fresh install. Not "present and refusing" — absent.
	if holdsComputer(app, t) {
		t.Fatal("a fresh install hands the model a tool for driving the machine that nobody granted")
	}

	switchOnComputer(t)
	if !holdsComputer(app, t) {
		t.Fatal("the switch is on and the tool is still not handed over")
	}

	if err := config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.ComputerControlOn = false
		return nil
	}); err != nil {
		t.Fatalf("could not switch it back off: %v", err)
	}
	if holdsComputer(app, t) {
		t.Fatal("switching it back off left the tool in the block")
	}
}

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
	s := newComputerSkill(seed(&App{cfg: config.Config{SandboxRoot: t.TempDir()}}, newConversation()), nil)

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
