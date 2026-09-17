package main

// What a yes turns into, tested against the gate that will actually read it.
//
// The rule this file writes is only worth anything if internal/turn's existing
// permission gate resolves it the way this design assumed. So these tests do not
// check the struct's fields — they hand the rule to safety.PermissionConfig and
// ask it the questions the executor will ask.

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/safety"
)

func TestOneYesCoversTheProgramNotTheVerb(t *testing.T) {
	cfg := safety.PermissionConfig{Rules: []safety.PermissionRule{reachRuleFor("notepad.exe")}}

	// The four names the executor judges a `computer` call by (skill.Unpack
	// turns the pack call into one of these). A user who said "yes, drive
	// Notepad" said it once, not once per verb.
	for _, tool := range []string{"computer_read", "computer_click", "computer_click_at", "computer_scroll", "computer_drag", "computer_type", "computer_capture"} {
		action, matched := cfg.Resolve(tool, []string{"notepad", "ref=3"})
		if !matched {
			t.Errorf("%s on notepad matched no rule; the user would be asked again", tool)
			continue
		}
		if action != safety.PermissionAllow {
			t.Errorf("%s on notepad resolved to %q, want allow", tool, action)
		}
	}
}

func TestAYesForOneProgramIsNotAYesForAnother(t *testing.T) {
	cfg := safety.PermissionConfig{Rules: []safety.PermissionRule{reachRuleFor("notepad.exe")}}
	if _, matched := cfg.Resolve("computer_click", []string{"winword", "ref=1"}); matched {
		t.Fatal("granting Notepad also granted Word — the pattern is too loose to be a per-program grant")
	}
}

func TestAProgramWithTheSameFilenameAtAnotherPathDoesNotInheritTheGrant(t *testing.T) {
	// Filename-only grants let an unrelated executable call itself notepad.exe
	// and inherit Notepad's reach. The path is fingerprinted into the identity.
	a := reachRuleFor(`C:\Windows\System32\notepad.exe`)
	b := reachRuleFor(`C:\Program Files\WindowsApps\Notepad\Notepad.EXE`)
	if a.Pattern == b.Pattern {
		t.Errorf("two executables with the same filename made one grant: %q", a.Pattern)
	}
	if !strings.HasPrefix(a.Pattern, "notepad@") || !strings.HasSuffix(a.Pattern, "*") {
		t.Errorf("path-backed identity has an unexpected shape: %q", a.Pattern)
	}
}

func TestTheGrantIsWrittenDownAndCanBeTakenBack(t *testing.T) {
	// A data root of its own: this writes a real permissions.json, and the
	// user's own must never be the thing under test.
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())

	app := newTestApp(t)
	// Empty, never nil: a nil slice marshals to JSON null, the settings page
	// does .length on it, and Svelte aborts the render (ARCHITECTURE.md §34).
	// The shipped state — off, nothing granted — is exactly when nil shows up.
	if got := app.GrantedComputerApps(); got == nil || len(got) != 0 {
		t.Fatalf("a fresh install grants %v (nil=%v), want an empty list", got, got == nil)
	}
	if reachGranted("notepad.exe") {
		t.Fatal("a program is granted before anyone was asked")
	}

	const notepadPath = `C:\Windows\System32\notepad.exe`
	if err := config.UpdatePermissions(func(cfg *safety.PermissionConfig) error {
		cfg.Rules = append(cfg.Rules, reachRuleFor(notepadPath))
		return nil
	}); err != nil {
		t.Fatalf("writing the grant failed: %v", err)
	}

	if !reachGranted(notepadPath) {
		t.Error("the grant did not take effect")
	}
	if reachGranted(`C:\Temp\notepad.exe`) {
		t.Error("a different notepad.exe inherited the grant")
	}
	got := app.GrantedComputerApps()
	if len(got) != 1 || got[0] != "notepad" {
		t.Fatalf("the settings list shows %v, want [notepad] — a grant nobody can see is a grant nobody can take back", got)
	}

	if err := app.RevokeComputerApp("notepad"); err != nil {
		t.Fatalf("revoking failed: %v", err)
	}
	if reachGranted(notepadPath) {
		t.Error("the program is still granted after being revoked")
	}
	if got := app.GrantedComputerApps(); len(got) != 0 {
		t.Errorf("the settings list still shows %v after revoke", got)
	}
}

func TestRevokingOneProgramLeavesEveryOtherRuleAlone(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())

	mine := safety.PermissionRule{Tool: "shell", Pattern: "git *", Action: safety.PermissionAllow}
	if err := config.UpdatePermissions(func(cfg *safety.PermissionConfig) error {
		cfg.Rules = append(cfg.Rules, mine,
			reachRuleFor(`C:\Windows\System32\notepad.exe`),
			reachRuleFor(`C:\Windows\System32\calc.exe`))
		return nil
	}); err != nil {
		t.Fatalf("writing failed: %v", err)
	}

	app := newTestApp(t)
	if err := app.RevokeComputerApp("notepad"); err != nil {
		t.Fatalf("revoking failed: %v", err)
	}

	cfg, err := config.LoadPermissions()
	if err != nil {
		t.Fatalf("reading back failed: %v", err)
	}
	var sawShell, sawCalc, sawNotepad bool
	for _, r := range cfg.Rules {
		switch {
		case r.Tool == "shell":
			sawShell = true
		case grantedComputerName(r.Pattern) == "calc":
			sawCalc = true
		case grantedComputerName(r.Pattern) == "notepad":
			sawNotepad = true
		}
	}
	if !sawShell {
		t.Error("revoking a program deleted a rule the user wrote by hand for another tool")
	}
	if !sawCalc {
		t.Error("revoking one program revoked another")
	}
	if sawNotepad {
		t.Error("the revoked program is still in the file")
	}
}
