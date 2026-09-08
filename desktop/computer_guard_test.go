package main

// The guard's tests, and the one in here that is not like the others.
//
// Most of this file is ordinary: a table of programs, an expected answer. The
// exception is TestNoReachFailureIsJustFailed, which exists because of a
// specific historical event rather than a general principle. The tool this work
// replaces reported every failure as some variant of the word "failed", the
// owner hit one live, the transcript could not say which of four situations had
// occurred, and the commit that removed it says "deleted rather than debugged"
// (50584e5). That test is the thing standing between this attempt and the same
// ending, so it is written to fail loudly when a new code is added without a
// sentence rather than to pass quietly when one is.

import (
	"strings"
	"testing"
)

func TestTheSwitchIsTheFirstQuestion(t *testing.T) {
	err := guardReach(false, 1234, "read", reachTarget{PID: 999, Exe: "notepad.exe", Title: "ไม่มีชื่อ"})
	if err == nil {
		t.Fatal("a reach went through with the feature switched off")
	}
	got := explainReach("read", err)
	// The refusal has to carry the way out. A NO that does not say where the
	// switch is trains the user to ask a human instead of the app.
	if !strings.Contains(got, "ตั้งค่า") {
		t.Errorf("the refusal does not say where to turn it on: %q", got)
	}
}

func TestAetoxDoesNotDriveAetox(t *testing.T) {
	// By process id — the answer a rename cannot change.
	err := guardReach(true, 4242, "click", reachTarget{PID: 4242, Exe: "something-else.exe", Title: "Aetox"})
	if err == nil {
		t.Fatal("the agent was allowed to reach a window belonging to its own process")
	}

	// And by name, for a second copy of Aetox running beside this one.
	err = guardReach(true, 4242, "click", reachTarget{PID: 77, Exe: `C:\Program Files\Aetox\Aetox.exe`, Title: "Aetox"})
	if err == nil {
		t.Fatal("the agent was allowed to reach another copy of Aetox")
	}
}

func TestTerminalsAndBrowsersAreSentToTheToolThatDoesThemProperly(t *testing.T) {
	for _, tc := range []struct {
		exe, wants string
	}{
		{`C:\Windows\System32\cmd.exe`, "shell"},
		{"powershell.exe", "shell"},
		{"WindowsTerminal.exe", "shell"},
		{"pwsh", "shell"},
		{`C:\Program Files\Google\Chrome\Application\chrome.exe`, "browser"},
		{"msedge.exe", "browser"},
		{"firefox", "browser"},
	} {
		err := guardReach(true, 1, "type", reachTarget{PID: 2, Exe: tc.exe, Title: "x"})
		if err == nil {
			t.Errorf("%s was allowed; it should be refused", tc.exe)
			continue
		}
		got := explainReach("type", err)
		// The refusal names the right tool. This is the whole difference
		// between "unsupported" and a redirect: the model has somewhere to go.
		if !strings.Contains(got, "`"+tc.wants+"`") {
			t.Errorf("refusing %s did not name `%s` as the way to do it: %q", tc.exe, tc.wants, got)
		}
	}
}

func TestAnUnidentifiedWindowIsNeverTouched(t *testing.T) {
	if err := guardReach(true, 1, "read", reachTarget{PID: 2, Exe: "", Title: "mystery"}); err == nil {
		t.Fatal("a window whose program could not be identified was allowed")
	}
}

func TestOrdinaryProgramsGoThrough(t *testing.T) {
	for _, exe := range []string{"notepad.exe", `C:\Program Files\Microsoft Office\WINWORD.EXE`, "calc"} {
		if err := guardReach(true, 1, "click", reachTarget{PID: 2, Exe: exe, Title: "x"}); err != nil {
			t.Errorf("%s was refused: %v", exe, err)
		}
	}
}

func TestProgramsThatReachTheWholeMachineSaySoOnTheCard(t *testing.T) {
	for _, exe := range []string{"explorer.exe", "regedit", "SystemSettings.exe", "taskmgr"} {
		if broadReachWarning(exe) == "" {
			t.Errorf("%s carries no warning; approving it looks the same as approving a text editor", exe)
		}
		// Warned is not blocked — the user may well mean it.
		if err := guardReach(true, 1, "click", reachTarget{PID: 2, Exe: exe, Title: "x"}); err != nil {
			t.Errorf("%s was blocked outright; the design warns rather than blocks: %v", exe, err)
		}
	}
	if broadReachWarning("notepad.exe") != "" {
		t.Error("a text editor carries a broad-reach warning, which makes the real ones mean less")
	}
}

func TestExeKeyIgnoresPathCaseAndSuffix(t *testing.T) {
	for _, in := range []string{`C:\Windows\System32\CMD.EXE`, "cmd.exe", "CMD", `\\server\share\cmd.Exe`, "/usr/bin/cmd"} {
		if got := exeKey(in); got != "cmd" {
			t.Errorf("exeKey(%q) = %q, want cmd — the tables must not need two spellings of one program", in, got)
		}
	}
}

func TestReadingIsNotActing(t *testing.T) {
	for _, a := range []string{"list_apps", "read", "capture"} {
		if computerIsActing(a) {
			t.Errorf("%s counts as acting; it would take the screen lock and raise the takeover banner for a look", a)
		}
	}
	for _, a := range []string{"click", "type", "focus", "close"} {
		if !computerIsActing(a) {
			t.Errorf("%s does not count as acting; it would change the screen with no lock and no banner", a)
		}
	}
}

func TestOneChatAtATimeHasTheScreen(t *testing.T) {
	var l screenLock
	if err := l.take("chat-a", "พิมพ์ใน Notepad"); err != nil {
		t.Fatalf("the first taker was refused: %v", err)
	}
	// The same chat asking again is not a second holder.
	if err := l.take("chat-a", "คลิกปุ่มบันทึก"); err != nil {
		t.Fatalf("the holder was refused its own lock: %v", err)
	}
	err := l.take("chat-b", "อะไรก็ตาม")
	if err == nil {
		t.Fatal("two chats held the screen at once")
	}
	// The loser is told what the winner is doing, not merely that it lost.
	if got := explainReach("click", err); !strings.Contains(got, "คลิกปุ่มบันทึก") {
		t.Errorf("the refused chat is not told what the holder is doing: %q", got)
	}
	l.release("chat-b") // not the holder; must not free it
	if l.heldBy() != "chat-a" {
		t.Fatal("a chat that never held the lock released it")
	}
	l.release("chat-a")
	if l.heldBy() != "" {
		t.Fatal("the lock survived its holder releasing it")
	}
	if err := l.take("chat-b", "ตาของฉันแล้ว"); err != nil {
		t.Fatalf("the lock stayed taken after release: %v", err)
	}
}

// The test this file exists for.
//
// Every failure the reach can produce must arrive as a sentence that changes
// what to do next. A new HRESULT added to the constants without a case in
// explainHRESULT fails here, which is the point: the classifier is the feature,
// not the decoration.
func TestNoReachFailureIsJustFailed(t *testing.T) {
	codes := map[uint32]string{
		hrElementNotAvailable: "read the window again",
		hrElementNotEnabled:   "disabled",
		hrNotSupported:        "does not accept",
		hrNoInterface:         "does not accept",
		hrNoClickablePoint:    "no point",
		hrAccessDenied:        "higher privileges",
		hrTimeout:             "busy",
		hrCallRejected:        "busy",
		hrServerNotResponding: "busy",
		hrRPCUnavailable:      "closed",
		hrRPCCallFailed:       "closed",
		hrClassNotRegistered:  "not available on this machine",
		hrInvalidArg:          "defect in aetox",
	}
	seen := map[string]bool{}
	for code, want := range codes {
		got := explainReach("click", hresult{code: code, what: "test"})
		if got == "" {
			t.Errorf("HRESULT %#08x explains to nothing", code)
			continue
		}
		if unclassified(got) {
			t.Errorf("HRESULT %#08x falls through to the raw fallback: %q", code, got)
		}
		if !strings.Contains(strings.ToLower(got), want) {
			t.Errorf("HRESULT %#08x = %q, expected it to mention %q", code, got, want)
		}
		seen[got] = true
	}
	// Distinct situations must read as distinct situations. If two codes
	// collapse to one sentence, the transcript is back where the old tool was.
	if len(seen) < 7 {
		t.Errorf("only %d distinct sentences for %d codes; too many failures read alike", len(seen), len(codes))
	}
}

// The specific sentence the removed tool could not say. `GetCursorPos failed`
// and `SendInput delivered 0 of 4` were the same Windows error 5, and the
// post-mortem could not tell a locked screen from a privilege problem because
// neither message mentioned either.
func TestALockedScreenSaysSoInsteadOfFailing(t *testing.T) {
	got := explainReach("type", win32Error{call: "SendInput", code: winAccessDenied})
	for _, want := range []string{"locked", "Nothing was clicked or typed"} {
		if !strings.Contains(got, want) {
			t.Errorf("the no-input-desktop sentence is missing %q: %q", want, got)
		}
	}
	// And it must not be confused with the privilege sentence, which is a
	// different fix for the user.
	if strings.Contains(got, "higher privileges") {
		t.Error("a locked screen is being reported as a privilege problem; those have different fixes")
	}
}

func TestAGuardRefusalReadsAsARuleNotAsAWindowsError(t *testing.T) {
	got := explainReach("click", refuse("เหตุผล", "ทางออก"))
	if strings.Contains(got, "did not go through") || strings.Contains(got, "Windows") {
		t.Errorf("a rule this project chose is being worded as a failure of the machine: %q", got)
	}
	if !strings.Contains(got, "ทางออก") {
		t.Errorf("the refusal dropped its hint: %q", got)
	}
}
