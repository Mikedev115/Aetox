package main

// The door. Every `computer` action passes through here before any COM pointer
// is touched, and nothing downstream re-asks any of these questions.
//
// The rules come from three places and it is worth keeping the provenance,
// because each one was argued somewhere and none of them is a preference:
//
//   - docs/architecture/computer-use-2026-09-07.md §6 — the six lines this
//     feature does not cross. Two of them live here: Aetox does not drive Aetox,
//     and a credential is never typed.
//   - The rival study (Codex, Claude Desktop, 2026-09-08). Both ship this
//     capability OFF, both refuse to drive a terminal, and Claude caps browsers
//     at view-only. All three land here for the same reason they landed there:
//     a reach that can drive a terminal is a shell with extra steps, and a
//     reach that can drive a browser is the browser tool with worse aim.
//   - desktop/workspace.go:186 — the house rule that there is no "allow once".
//     A grant this door hands out is written down where the user can see it and
//     take it back, never held in a variable that forgets.
//
// The tier table below is deliberately a table of NAMES, not of behaviours. It
// answers "what kind of program is this", and the single decision function
// underneath answers "so what". Splitting them is what makes the whole thing
// testable without a desktop.

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Mikedev115/Aetox/internal/config"
)

// reachTier is how much of a program this feature will ever touch, decided by
// what kind of program it is rather than by who is asking.
type reachTier int

const (
	// tierFull — read it, press it, type into it. Everything not named below.
	tierFull reachTier = iota
	// tierWarned — full control, but the approval card says out loud what this
	// particular yes is worth, because these programs reach the whole machine.
	tierWarned
	// tierElsewhere — refused, with the tool that does this properly named in
	// the refusal. Not "unsupported": a wrong tool is being reached for, and
	// saying which is the right one is the whole value of the refusal.
	tierElsewhere
	// tierNever — refused outright, no route offered.
	tierNever
)

// terminals get tierElsewhere, and the refusal points at `shell`.
//
// Codex refuses these outright and Claude caps them at click-only; we refuse
// and redirect, which is the same safety with a better next step. Driving a
// terminal by clicking is a shell with a worse interface and none of the audit
// trail — internal/turn/executor.go's shell path records what ran, and a
// keystroke sprayed at a console window records nothing.
var reachTerminals = map[string]bool{
	"cmd": true, "powershell": true, "pwsh": true, "conhost": true,
	"windowsterminal": true, "wt": true, "openconsole": true,
	"bash": true, "sh": true, "git-bash": true, "mintty": true,
	"ubuntu": true, "wsl": true, "wslhost": true,
}

// browsers get tierElsewhere, and the refusal points at `browser`.
//
// Row 2 of the direction doc's table is the user's own Chrome, reached by an
// extension that does not exist yet. Until it does, a browser window read
// through the accessibility tree is the bad version of a thing we already do
// well: §7 of that doc says so in as many words — "the accessibility tree of a
// modern web app is not a DOM, and 'badly' is how a reach becomes something the
// owner has to apologise for."
var reachBrowsers = map[string]bool{
	"chrome": true, "msedge": true, "firefox": true, "brave": true,
	"opera": true, "vivaldi": true, "arc": true, "librewolf": true,
	"iexplore": true, "safari": true,
}

// These reach the whole machine, so the approval card says so. Not blocked —
// the user may well mean it — but a yes here is worth more than a yes to a
// text editor and the card must not pretend otherwise. Claude Desktop shows the
// same warning on the same list, and it is the right call.
var reachBroadReach = map[string]string{
	"explorer":       "หน้าต่างนี้อ่านและเขียนไฟล์ได้ทุกที่ในเครื่อง",
	"systemsettings": "หน้าต่างนี้เปลี่ยนการตั้งค่าของ Windows ได้",
	"control":        "หน้าต่างนี้เปลี่ยนการตั้งค่าของ Windows ได้",
	"regedit":        "หน้าต่างนี้แก้รีจิสทรีได้ ซึ่งทำให้ Windows พังได้",
	"mmc":            "หน้าต่างนี้เป็นเครื่องมือผู้ดูแลระบบ",
	"taskmgr":        "หน้าต่างนี้ปิดโปรแกรมอะไรก็ได้ในเครื่อง",
}

// Aetox's own program names. The direction doc's §6.5 in one map: "an agent
// that can click its own approval dialog has no approval dialog."
//
// Checked by process id first — that is the answer that cannot be faked by a
// rename — and by name second, because a second copy of Aetox running beside
// this one is still Aetox and clicking its dialog is the same hole.
var reachSelfNames = map[string]bool{
	"aetox": true, "aetox-desktop": true, "desktop": true,
}

// appTier answers what kind of program this is. Input is an executable path or
// bare name; case and the .exe suffix do not matter.
func appTier(exePath string) (reachTier, string) {
	name := exeKey(exePath)
	switch {
	case name == "":
		// A window whose program cannot be identified is not driven. Naming
		// what may be touched is the whole mechanism; an unnamed target is
		// outside it by construction.
		return tierNever, "ไม่รู้ว่าหน้าต่างนี้เป็นของโปรแกรมไหน จึงไม่แตะ"
	case reachSelfNames[name]:
		return tierNever, "Aetox ไม่ขับ Aetox — เอเจนต์ที่กดหน้าต่างอนุมัติของตัวเองได้ ก็เท่ากับไม่มีหน้าต่างอนุมัติ"
	case reachTerminals[name]:
		return tierElsewhere, "shell"
	case reachBrowsers[name]:
		return tierElsewhere, "browser"
	}
	if why, ok := reachBroadReach[name]; ok && why != "" {
		return tierWarned, why
	}
	return tierFull, ""
}

// exeKey normalises an executable to the key the tables use: base name, no
// extension, lower case. "C:\Windows\System32\cmd.exe" and "CMD" are the same
// program and the tables must not need to know both spellings.
func exeKey(exePath string) string {
	s := strings.TrimSpace(exePath)
	if s == "" {
		return ""
	}
	s = filepath.Base(strings.ReplaceAll(s, `\`, "/"))
	s = strings.TrimSuffix(strings.ToLower(s), ".exe")
	return s
}

// reachTarget is one window the tool has been asked to touch, as the guard sees
// it. Filled in by the platform half (computer_reach_windows.go).
type reachTarget struct {
	HWND  uintptr
	PID   int32
	Exe   string // full path when known, bare name otherwise
	Title string
}

// Label is what the approval card and every refusal call this window. Title
// first because that is what the user is looking at; the program name is what
// the permission is actually keyed on, so both appear.
func (t reachTarget) Label() string {
	name := exeKey(t.Exe)
	title := strings.TrimSpace(t.Title)
	switch {
	case title != "" && name != "":
		return fmt.Sprintf("%s (%s)", title, name)
	case title != "":
		return title
	case name != "":
		return name
	}
	return "หน้าต่างที่ไม่มีชื่อ"
}

// actingActions are the ones that change something on the user's screen. The
// split matters twice: reading is what the วางแผน stance keeps, and only these
// take the screen lock and raise the takeover banner.
var computerActingActions = map[string]bool{
	"focus": true, "click": true, "type": true, "close": true,
}

func computerIsActing(action string) bool {
	return computerActingActions[strings.ToLower(strings.TrimSpace(action))]
}

// ---------------------------------------------------------------------------
// The one decision
// ---------------------------------------------------------------------------

// guardReach is the single gate. It answers with nil or with a refusal already
// worded for the model — never with a bare error, and never with a boolean the
// caller has to turn into a sentence of its own.
//
// It does NOT ask the user anything. Approval is a separate step that runs
// after this returns nil, because the two questions are different: this one is
// "is this the kind of thing Aetox does at all", and that one is "does this
// user want it done here". A card offered for something that would be refused
// afterwards is the pattern askWorkspaceWiden calls out — "a question whose yes
// cannot be honoured trains the user to distrust the question."
func guardReach(on bool, selfPID int32, action string, t reachTarget) error {
	if !on {
		return refuse(
			"การใช้คอมพิวเตอร์ยังปิดอยู่",
			"เปิดได้ที่ ตั้งค่า → การใช้คอมพิวเตอร์ แล้วสั่งใหม่อีกครั้ง")
	}
	if t.PID != 0 && t.PID == selfPID {
		return refuse(
			"หน้าต่างนี้เป็นของ Aetox เอง",
			"Aetox ไม่ขับ Aetox — ถ้าต้องการทำอะไรในแอปนี้ บอกมาตรง ๆ ได้เลย")
	}

	tier, note := appTier(t.Exe)
	switch tier {
	case tierNever:
		return refuse(note, "")
	case tierElsewhere:
		switch note {
		case "shell":
			return refuse(
				fmt.Sprintf("%s เป็นหน้าต่างเทอร์มินัล จึงไม่ขับผ่านการคลิก", t.Label()),
				"ใช้ `shell` แทน — คำสั่งที่รันจะถูกบันทึกไว้ ส่วนคีย์ที่พ่นใส่หน้าต่างคอนโซลไม่ถูกบันทึกอะไรเลย")
		case "browser":
			return refuse(
				fmt.Sprintf("%s เป็นเบราว์เซอร์ จึงไม่ขับผ่าน accessibility tree", t.Label()),
				"ใช้ `browser` แทน — มันเห็น DOM จริงและเล็งได้แม่นกว่ามาก")
		}
		return refuse(fmt.Sprintf("%s ไม่ได้ขับผ่านเครื่องมือนี้", t.Label()), "")
	}
	return nil
}

// broadReachWarning is the extra line the approval card carries for a program
// that reaches the whole machine. Empty for everything else.
func broadReachWarning(exePath string) string {
	if tier, note := appTier(exePath); tier == tierWarned {
		return note
	}
	return ""
}

// ---------------------------------------------------------------------------
// The switch
// ---------------------------------------------------------------------------

// computerControlOn reads the setting. Positive-by-absence — absent means OFF,
// which is what this feature ships as and what both rivals ship theirs as. The
// spelling rule is config.go's: name it so that the zero value is what the
// product does out of the box.
func computerControlOn() bool {
	pref, ok, _ := config.LoadModelPreference()
	if !ok {
		return false
	}
	return pref.ComputerControlOn
}

// ComputerControlOn reports the switch for the settings page.
func (a *App) ComputerControlOn() bool { return computerControlOn() }

// SetComputerControlOn persists it.
func (a *App) SetComputerControlOn(on bool) error {
	return config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.ComputerControlOn = on
		return nil
	})
}

// ---------------------------------------------------------------------------
// The screen lock
// ---------------------------------------------------------------------------

// screenLock keeps one chat at a time in charge of the machine.
//
// Both rivals do this and both do it for the same reason: two agents clicking
// in the same window produce a state neither one predicted, and the second one
// reports success for a click the first one's dialog swallowed. Claude Code
// takes a machine-wide lock at the first computer-use action and holds it for
// the life of the session; ours is narrower on purpose — held for the turn, not
// the session — because a chat that finished acting has no claim on the screen,
// and a user who switched chats should not have to close one to use the other.
type screenLock struct {
	mu      sync.Mutex
	holder  string // conversation id
	holding string // what it is doing, for the message the loser gets
}

func (l *screenLock) take(sessionID, doing string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.holder != "" && l.holder != sessionID {
		return refuse(
			fmt.Sprintf("อีกแชตหนึ่งกำลังคุมหน้าจออยู่ (%s)", l.holding),
			"รอให้เทิร์นนั้นจบ หรือกดหยุดมันก่อน แล้วค่อยสั่งใหม่")
	}
	l.holder = sessionID
	l.holding = doing
	return nil
}

func (l *screenLock) release(sessionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.holder == sessionID {
		l.holder = ""
		l.holding = ""
	}
}

func (l *screenLock) heldBy() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.holder
}
