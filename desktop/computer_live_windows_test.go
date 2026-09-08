//go:build windows

package main

// The gate DECISIONS.md §22 set for bringing desktop control back.
//
//	"If desktop control comes back, it comes back with a live end-to-end test on
//	 a real desktop, not just unit tests over the input-buffer builders — that
//	 was the gap that let a broken tool ship."
//
// This is that test, and the post-mortem of the removed tool sharpened it in one
// way worth stating. §22's own status line says the tool was "live-verified on
// the dev box", and it still failed in the installed application. So did the
// console-window bug five hours earlier, whose commit says why: "dev never
// showed it because `wails dev` has a console the children inherit." Twice, dev
// mode was the thing that hid the failure.
//
// So this test does not stand alone. It proves the mechanism end to end on a
// real desktop, in a real process, against a real application — and the release
// checklist in docs/DECISIONS.md §239 still requires the same round trip in the
// INSTALLED exe before the feature is offered to anyone. This is the part a
// machine can check; that is the part only running the real thing can.
//
// It opens its own Notepad and closes it again. Nothing here ever touches a
// window the person running it was using: driving somebody's open work from a
// test suite is the one thing worse than not testing at all.
//
//	AETOX_LIVE_COMPUTER=1 go test ./desktop -run TestLiveComputer -v

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func liveComputerGate(t *testing.T) {
	t.Helper()
	if os.Getenv("AETOX_LIVE_COMPUTER") == "" {
		t.Skip("set AETOX_LIVE_COMPUTER=1 to drive a real window (opens and closes its own Notepad)")
	}
}

// openOwnWindow starts a program this test owns and waits for the window it
// puts up. Returns the window and a function that closes it, so a failed
// assertion never leaves an application sitting on somebody's screen.
//
// The window is found by watching for one that was not there before, NOT by
// matching the child process id, and that is not defensive coding — it is a
// fact about Windows 11 that this test discovered the hard way. `notepad.exe`
// in System32 is a stub for a packaged Store application: starting it returns a
// process that hands the work to a different one and exits, so the window that
// appears belongs to a pid this test never had. A pid match found nothing, three
// times, against a Notepad that was plainly on screen.
//
// charmap is the first choice for the same reason stated positively: it is still
// a plain Win32 program with a real editable field and real buttons, which is
// what this test needs to exercise both halves of the reach.
func openOwnWindow(t *testing.T) (reachTarget, func()) {
	t.Helper()

	before := map[uintptr]bool{}
	if windows, err := reachListWindows(); err == nil {
		for _, w := range windows {
			before[w.HWND] = true
		}
	}

	var started *exec.Cmd
	for _, program := range []string{"charmap.exe", "notepad.exe"} {
		cmd := exec.Command(program)
		if err := cmd.Start(); err == nil {
			started = cmd
			break
		}
	}
	if started == nil {
		t.Skip("neither charmap nor notepad would start on this machine")
	}

	var found reachTarget
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) && found.HWND == 0 {
		time.Sleep(250 * time.Millisecond)
		windows, err := reachListWindows()
		if err != nil {
			continue
		}
		for _, w := range windows {
			if before[w.HWND] {
				continue
			}
			switch exeKey(w.Exe) {
			case "charmap", "notepad":
				found = w
			}
		}
	}

	stop := func() {
		if found.HWND != 0 {
			_ = reachCloseWindow(found.HWND)
		}
		if started.Process != nil {
			_ = started.Process.Kill()
			_, _ = started.Process.Wait()
		}
	}
	if found.HWND == 0 {
		stop()
		t.Skip("the program started but no new window appeared for this test to drive")
	}
	t.Logf("driving %s", found.Label())
	return found, stop
}

// The round trip the old tool could not do, and the single assertion this whole
// rebuild is answerable to: type Thai into a real application and read it back
// character for character.
//
// Thai specifically, and not because the project is Thai. It is the case that
// separates the two mechanisms: synthesized keystrokes go through a keyboard
// layout, and a layout that cannot produce these characters produces something
// else without failing. Handing the string to the provider through the Value
// pattern has no layout in the middle of it, so this either matches exactly or
// the mechanism is wrong.
func TestLiveComputerTypesThaiAndReadsItBack(t *testing.T) {
	liveComputerGate(t)
	target, stop := openOwnWindow(t)
	defer stop()

	// A declined raise is logged, never fatal. Windows refuses a foreground
	// change whenever it judges it a focus steal, and the reach is built so that
	// nothing below depends on the raise having worked: pressing and typing go
	// through the control. A test that failed here would be asserting a rule
	// about presentation and calling it a broken mechanism.
	if err := reachFocusWindow(target.HWND); err != nil {
		t.Logf("Windows would not raise the window, carrying on: %s", explainReach("focus", err))
	}

	nodes, _, err := reachReadWindow(target.HWND, "", reachReadCap)
	if err != nil {
		t.Fatalf("could not read the window: %s", explainReach("read", err))
	}
	// By control TYPE, never by the role string. The role comes back in the
	// user's language — this machine answers "แก้ไข", not "edit" — so a test
	// that searched the role for "edit" skipped itself against a window that
	// plainly had one. That is what reachNode.Kind exists for.
	var editor reachNode
	for _, n := range nodes {
		if (n.Kind == ctrlEdit || n.Kind == ctrlDocument) && n.Enabled && !n.Password && len(n.RuntimeID) > 0 {
			editor = n
			break
		}
	}
	if len(editor.RuntimeID) == 0 {
		// A fact about the program, not about the reach: a WinUI application can
		// expose its text area as something with no Value pattern at all.
		// Skipping rather than failing keeps this test honest about what it
		// proved.
		t.Skipf("this window exposes no editable field; rows were: %s", rowSummary(nodes))
	}

	const want = "สวัสดีครับ ทดสอบพิมพ์ภาษาไทย ๑๒๓"
	if err := reachType(target.HWND, editor.RuntimeID, want); err != nil {
		t.Fatalf("could not type into the field: %s", explainReach("type", err))
	}
	settle()

	got, err := reachReadBack(target.HWND, editor.RuntimeID)
	if err != nil {
		t.Fatalf("could not read the field back: %s", explainReach("read", err))
	}
	if strings.TrimRight(got, "\r\n") != want {
		t.Fatalf("what came back is not what went in:\n  sent %q\n  read %q", want, got)
	}
	t.Logf("typed and read back %d Thai characters exactly", len([]rune(want)))
}

func TestLiveComputerPressesARealControl(t *testing.T) {
	liveComputerGate(t)
	target, stop := openOwnWindow(t)
	defer stop()

	// A declined raise is logged, never fatal. Windows refuses a foreground
	// change whenever it judges it a focus steal, and the reach is built so that
	// nothing below depends on the raise having worked: pressing and typing go
	// through the control. A test that failed here would be asserting a rule
	// about presentation and calling it a broken mechanism.
	if err := reachFocusWindow(target.HWND); err != nil {
		t.Logf("Windows would not raise the window, carrying on: %s", explainReach("focus", err))
	}
	nodes, _, err := reachReadWindow(target.HWND, "", reachReadCap)
	if err != nil {
		t.Fatalf("could not read the window: %s", explainReach("read", err))
	}
	var button reachNode
	for _, n := range nodes {
		if n.Kind == ctrlButton && n.Enabled && n.Name != "" && len(n.RuntimeID) > 0 {
			button = n
			break
		}
	}
	if len(button.RuntimeID) == 0 {
		t.Skipf("no pressable control in this window; rows were: %s", rowSummary(nodes))
	}
	if err := reachClick(target.HWND, button.RuntimeID); err != nil {
		t.Fatalf("could not press %q: %s", button.Name, explainReach("click", err))
	}
	t.Logf("pressed %q through its own provider, with no cursor and no coordinate", button.Name)
}

// A window that closes is the proof that `close` is a request rather than a
// demolition: Notepad with nothing typed into it closes on WM_CLOSE, and the
// same message on a Notepad with unsaved text puts up a dialog instead. Only
// the first is asserted here, because the second needs a human to answer it.
func TestLiveComputerAsksAWindowToClose(t *testing.T) {
	liveComputerGate(t)
	target, stop := openOwnWindow(t)
	defer stop()

	if err := reachCloseWindow(target.HWND); err != nil {
		t.Fatalf("could not ask the window to close: %s", explainReach("close", err))
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := reachFindWindow(target.HWND); err != nil {
			t.Log("the window went away on its own, which is what a close request should do")
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatal("the window is still there five seconds after being asked to close")
}

func rowSummary(nodes []reachNode) string {
	var b strings.Builder
	for i, n := range nodes {
		if i > 6 {
			b.WriteString("…")
			break
		}
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(n.Role + " " + n.Name)
	}
	return b.String()
}
