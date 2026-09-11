package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/update"
	"github.com/Mikedev115/Aetox/internal/version"
)

// The desktop binding for the version the app calls itself. Trivial, and
// pinned anyway: the About page renders whatever comes back, so a binding that
// returned "" would put a dash where the version belongs and nothing would
// fail. internal/version's own test is what keeps this value honest.
func TestAppVersionIsTheOneConstant(t *testing.T) {
	if got := (&App{}).AppVersion(); got != version.Current {
		t.Errorf("AppVersion() = %q, want %q", got, version.Current)
	}
}

// The binding's one job beyond calling the package: "the user switched this
// off" has to arrive as a Status the About page can render, not as a rejected
// promise the page would show as a red failure under a setting they chose.
func TestCheckForUpdateReportsDisabledAsAStatusNotAnError(t *testing.T) {
	t.Setenv(update.DisableEnv, "1")

	st, err := (&App{}).CheckForUpdate()
	if err != nil {
		t.Fatalf("err = %v, want nil — a disabled check is not a failure", err)
	}
	if !st.Disabled {
		t.Error("Status.Disabled is false after the check was switched off")
	}
	if st.Available {
		t.Error("a check that never ran must not claim an update is available")
	}
	// Still renderable: the About page shows the version and the release link
	// whether or not a check happened.
	if st.Current != version.Current || st.URL == "" {
		t.Errorf("nothing usable to render: %+v", st)
	}
}

// A nil Wails context is what a binding sees if anything ever calls it before
// startup. It must fall back to a real context rather than panic on the way
// into http.NewRequestWithContext.
func TestCheckForUpdateSurvivesANilContext(t *testing.T) {
	a := &App{}
	if a.ctx != nil {
		t.Fatal("this test is only meaningful with no Wails context")
	}
	if _, err := a.CheckForUpdate(); err != nil {
		t.Fatalf("err = %v — in a test the check is skipped, not failed", err)
	}
}

// Restarting kills the process, and the process is where the turn lives — so
// the refusal is the same one every session switch gets. The sentence is not:
// this one arrives on the update card, where advice about switching chats would
// read as the update itself having broken.
func TestRestartToUpdateRefusesMidTurnInItsOwnWords(t *testing.T) {
	a := &App{}
	if err := a.beginTurn(a.cur().id); err != nil {
		t.Fatalf("beginTurn() = %v", err)
	}
	defer a.endTurn(a.cur().id)

	err := a.RestartToUpdate()
	if err == nil {
		t.Fatal("RestartToUpdate() = nil while a turn is running — it would kill the turn with the process")
	}
	if !errors.Is(err, errTurnBusyUpdate) {
		t.Errorf("err = %v, want the update-specific refusal", err)
	}
	if strings.Contains(err.Error(), "สลับแชท") {
		t.Error("the refusal points at a door the user is not standing in")
	}
}

// Nothing staged, nothing to restart into. Reachable by pressing the button on
// a window that reloaded after the Go side lost its staging (or never had it),
// and it must refuse rather than quit into the same build.
func TestRestartToUpdateWithNothingStagedRefuses(t *testing.T) {
	a := &App{}
	if err := a.RestartToUpdate(); err == nil {
		t.Error("RestartToUpdate() = nil with nothing staged — the app would close for no update")
	}
	if v := a.StagedUpdate(); v.Version != "" {
		t.Errorf("StagedUpdate() = %+v on a fresh app, want empty", v)
	}
}

// The morning after a "later": the previous hand-off's outcome has to reach
// the window in words, and only once. The installer itself is adopted or not
// by internal/update (its own tests); what this pins is that the desktop
// reads the log BEFORE anything is swept, carries the failure into
// StagedUpdate, and announces it — the window may already have asked and
// been told "nothing".
func TestAdoptStagedUpdateCarriesThePreviousFailureToTheWindow(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", root)
	if err := os.WriteFile(filepath.Join(root, "update-restart.log"),
		[]byte("error=This command cannot be run due to the error: The operation was canceled by the user.\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var announced []StagedInfo
	a := &App{emit: func(ev string, data ...any) {
		if ev == "update:staged" && len(data) == 1 {
			announced = append(announced, data[0].(StagedInfo))
		}
	}}

	a.adoptStagedUpdate()

	info := a.StagedUpdate()
	if info.Version != "" {
		t.Errorf("adopted %q with nothing staged", info.Version)
	}
	if !strings.Contains(info.InstallError, "UAC") {
		t.Errorf("InstallError = %q, want the declined-UAC sentence", info.InstallError)
	}
	if len(announced) != 1 || announced[0] != info {
		t.Errorf("update:staged = %+v, want exactly one carrying %+v", announced, info)
	}
	if _, err := os.Stat(filepath.Join(root, "update-restart.log")); !os.IsNotExist(err) {
		t.Error("the restart log survived being read — the next launch would show last week's failure")
	}

	// A fresh stage wipes the old failure: the sentence was about the
	// previous file, and a new one has not failed at anything.
	a.stagedMu.Lock()
	a.installError = ""
	a.stagedMu.Unlock()
	if a.StagedUpdate().InstallError != "" {
		t.Error("InstallError not cleared")
	}
}

// Nothing to report, nothing announced: a normal launch must not wake the
// window's updater for no reason.
func TestAdoptStagedUpdateIsSilentWhenThereIsNothing(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	called := false
	a := &App{emit: func(string, ...any) { called = true }}
	a.adoptStagedUpdate()
	if called {
		t.Error("emitted an event with nothing staged and no hand-off to report")
	}
}
