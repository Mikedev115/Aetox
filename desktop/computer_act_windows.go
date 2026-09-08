//go:build windows

package main

// The acting half: pressing, typing, raising, closing.
//
// Every verb here has two possible mechanisms and the choice between them is
// the whole design, so it is worth stating once rather than four times:
//
//	**Through the element, wherever the element will take it.** UIA's Invoke,
//	Toggle, SelectionItem and Value patterns are the control's own verbs. They
//	need no cursor, no coordinate, no focus and no foreground window; they
//	cannot miss; and they are unaffected by whatever the user has dragged over
//	the window in the meantime. This is what `click` and `type` use.
//
//	**Through the input queue, only where nothing else reaches.** SendInput is
//	what the removed tool used for everything, and it is what `keys` still uses,
//	because a keyboard shortcut is addressed to whatever has focus rather than
//	to a control. It is the mechanism with all the failure modes: integrity
//	level, input desktop, focus stealing. Those are classified in
//	computer_errors.go rather than reported as "failed", which is the single
//	change that most separates this from what came before.
//
// Foreground: the owner chose the Codex-on-Windows model, so an acting turn
// raises the window it is working on and the user is told the machine is being
// driven. That is a product decision, not a technical necessity — the UIA half
// would work perfectly well on a window nobody can see, which is exactly why
// raising has to be deliberate: a change made in a window the user never saw is
// a change they cannot check.

import (
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procAttachThreadInput   = user32.NewProc("AttachThreadInput")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procPostMessageW        = user32.NewProc("PostMessageW")
	procSendInput           = user32.NewProc("SendInput")
	procMapVirtualKeyW      = user32.NewProc("MapVirtualKeyW")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
)

// swRestore and wmClose are already declared in browser_windows.go, which owns
// the package's Win32 constant block for the same reason it owns user32 itself.
const (
	inputKeyboard      = 1
	keyEventKeyUp      = 0x0002
	keyEventUnicode    = 0x0004
	keyEventScanCode   = 0x0008
	keyEventExtendedKy = 0x0001
	mapVKToVSC         = 0
)

// keyboardInput is the full INPUT union, sized for x64 so a batch of them can
// be handed to SendInput as one array. The trailing padding is the mouse arm of
// the union, which is wider than the keyboard arm; getting it wrong makes
// SendInput read the second event out of the middle of the first.
type keyboardInput struct {
	typ        uint32
	_          uint32 // union alignment on x64
	wVk        uint16
	wScan      uint16
	dwFlags    uint32
	time       uint32
	dwExtraInf uintptr
	_          [8]byte
}

// sizeOfKeyboardInput exists for its test. The struct's size is the stride
// SendInput walks the array by, so a field too few or too many makes every
// event after the first be read out of the middle of its neighbour — a failure
// with no error and no symptom except keys that do not arrive.
func sizeOfKeyboardInput() int { return int(unsafe.Sizeof(keyboardInput{})) }

type winPoint struct{ X, Y int32 }

// ---------------------------------------------------------------------------
// Raising
// ---------------------------------------------------------------------------

// reachFocusWindow brings a window to the front.
//
// SetForegroundWindow alone fails on any modern Windows unless the calling
// process already owns the foreground — the OS refuses, silently, returning
// zero with no last-error worth reading. Attaching this thread's input queue to
// the foreground window's for the duration is the documented way around it, and
// it is done and undone in the same call so nothing is left attached to a
// process that may exit.
func reachFocusWindow(hwnd uintptr) error {
	if min, _, _ := procIsIconic.Call(hwnd); min != 0 {
		procShowWindow.Call(hwnd, swRestore)
	}

	fg, _, _ := procGetForegroundWindow.Call()
	if fg == hwnd {
		return nil
	}

	self, _, _ := procGetCurrentThreadID.Call()
	var fgPID uint32
	fgThread, _, _ := procGetWindowThreadProcessID.Call(fg, uintptr(unsafe.Pointer(&fgPID)))

	attached := false
	if fg != 0 && fgThread != 0 && fgThread != self {
		if ok, _, _ := procAttachThreadInput.Call(self, fgThread, 1); ok != 0 {
			attached = true
		}
	}
	ok, _, err := procSetForegroundWindow.Call(hwnd)
	if attached {
		procAttachThreadInput.Call(self, fgThread, 0)
	}
	if ok == 0 {
		// Zero here is THREE different situations and the first version of this
		// code collapsed two of them, which is the exact fault this project was
		// built to stop making. A locked screen and a higher-integrity window
		// both set a last error; Windows declining a focus steal sets none, and
		// mapping that silence to "access denied" told the live test the machine
		// was locked while the window sat plainly on screen.
		code := errnoOf(err)
		if code == 0 {
			code = winForegroundDeclined
		}
		return win32Error{call: "SetForegroundWindow", code: code}
	}
	return nil
}

// reachCloseWindow asks a window to close, the way its own title-bar × does.
//
// PostMessage rather than SendMessage, and WM_CLOSE rather than DestroyWindow:
// this is a REQUEST, and a program with unsaved work is entitled to answer it
// with a "save changes?" dialog instead of closing. Destroying the window would
// take that dialog away from the user along with their work. Posting also
// returns immediately, so a program that puts up that dialog does not park the
// tool call until somebody answers it.
func reachCloseWindow(hwnd uintptr) error {
	ok, _, err := procPostMessageW.Call(hwnd, wmClose, 0, 0)
	if ok == 0 {
		return win32Error{call: "PostMessage(WM_CLOSE)", code: errnoOf(err)}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Acting on one element
// ---------------------------------------------------------------------------

// findByRuntimeID walks a window's tree for the element a ref stands for.
//
// This is the cost of storing ids rather than pointers, paid once per action:
// one FindAll and a comparison per element, which measured in the low tens of
// milliseconds on a real window. What it buys is that a ref cannot dangle — the
// element is either found now, at the moment of acting, or it is honestly gone.
func findByRuntimeID(a *iUIAutomation, hwnd uintptr, want []int32) (*iUIAutomationElement, error) {
	if len(want) == 0 {
		return nil, refuse("ref นี้ไม่มีรหัสอ้างอิงของ element", "อ่านหน้าต่างใหม่แล้วลองอีกครั้ง")
	}
	root, err := a.elementFromHandle(hwnd)
	if err != nil {
		return nil, err
	}
	defer root.release()

	cond, err := a.createTrueCondition()
	if err != nil {
		return nil, err
	}
	defer cond.release()

	arr, err := root.findAll(scopeDescendants, cond)
	if err != nil {
		return nil, err
	}
	defer arr.release()

	n, err := arr.length()
	if err != nil {
		return nil, err
	}
	for i := int32(0); i < n; i++ {
		el, err := arr.at(i)
		if err != nil {
			return nil, err
		}
		got, err := el.runtimeID()
		if err != nil || !sameRuntimeID(got, want) {
			el.release()
			continue
		}
		return el, nil // caller releases
	}
	return nil, hresult{code: hrElementNotAvailable, what: "the element behind that ref is gone"}
}

func sameRuntimeID(a, b []int32) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// reachClick presses one element through whichever verb its provider offers.
//
// The order is deliberate and is not "try everything until something works":
// Invoke is a press, Toggle is a press on a thing that has two states,
// SelectionItem is a press on a thing in a list. A control supports exactly one
// of them in almost every case, and the legacy fallback is last because
// DoDefaultAction is defined as "whatever double-clicking would do", which is
// the vagueness the three above exist to remove.
func reachClick(hwnd uintptr, runtimeID []int32) error {
	return uia().do(func(a *iUIAutomation) error {
		el, err := findByRuntimeID(a, hwnd, runtimeID)
		if err != nil {
			return err
		}
		defer el.release()

		if enabled, err := el.isEnabled(); err == nil && !enabled {
			return hresult{code: hrElementNotEnabled, what: "that control is disabled"}
		}

		var last error
		for _, try := range []func() error{el.invoke, el.toggle, el.selectItem, el.doDefaultAction} {
			err := try()
			if err == nil {
				return nil
			}
			var hr hresult
			// Only "this control does not do that" is worth trying the next verb
			// for. A disabled control or a vanished one answers the same way to
			// all four, and running through them all turns one clear failure
			// into four seconds of silence.
			if !asHresult(err, &hr) || (hr.code != hrNotSupported && hr.code != hrNoInterface) {
				return err
			}
			last = err
		}
		return last
	})
}

// reachType puts text into one element through its Value pattern.
//
// Nothing is typed. The string is handed to the provider, which is why Thai,
// emoji and anything else arrive as themselves rather than as a sequence of
// synthesized keystrokes interpreted through whatever layout happens to be
// active — the half of the old tool that was hardest to trust and impossible to
// verify from a transcript.
//
// The password check is here as well as in the tool, and the duplication is on
// purpose: this is the last place before the text reaches Windows, and §6.2 of
// the direction doc is a line the design does not cross rather than a message
// the UI shows.
func reachType(hwnd uintptr, runtimeID []int32, text string) error {
	return uia().do(func(a *iUIAutomation) error {
		el, err := findByRuntimeID(a, hwnd, runtimeID)
		if err != nil {
			return err
		}
		defer el.release()

		if pw, err := el.isPassword(); err == nil && pw {
			return refuse(
				"ช่องนี้เป็นช่องรหัสผ่าน Aetox ไม่พิมพ์รหัสผ่านให้",
				"บอกผู้ใช้ให้พิมพ์เอง")
		}
		if enabled, err := el.isEnabled(); err == nil && !enabled {
			return hresult{code: hrElementNotEnabled, what: "that field is disabled"}
		}
		return el.setValue(text)
	})
}

// reachReadBack answers what one element holds now, so an action can report
// what it did rather than what it attempted.
func reachReadBack(hwnd uintptr, runtimeID []int32) (string, error) {
	var got string
	err := uia().do(func(a *iUIAutomation) error {
		el, err := findByRuntimeID(a, hwnd, runtimeID)
		if err != nil {
			return err
		}
		defer el.release()
		v, err := el.currentValue()
		if err != nil {
			return err
		}
		got = v
		return nil
	})
	return got, err
}

func asHresult(err error, out *hresult) bool {
	if hr, ok := err.(hresult); ok {
		*out = hr
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

// reachKeys is the one place SendInput survives from the old tool, and it is
// here because a shortcut has no element to address: ctrl+s is a message to
// whatever holds focus, which is a property of the desktop rather than of a
// control.
//
// Two things are different from the version that was removed. The scancode is
// filled in from MapVirtualKey, so applications reading raw input (games, some
// terminals, remote-desktop clients) see a real key rather than a virtual one
// with a zero scancode. And a short send is classified rather than reported as
// a count: "SendInput delivered 0 of 4 events" is the sentence the post-mortem
// could not act on.
func reachKeys(combo string) error {
	events, err := keyEvents(combo)
	if err != nil {
		return err
	}
	sent, _, callErr := procSendInput.Call(
		uintptr(len(events)),
		uintptr(unsafe.Pointer(&events[0])),
		unsafe.Sizeof(events[0]))
	if int(sent) != len(events) {
		code := errnoOf(callErr)
		if code == 0 {
			code = winAccessDenied
		}
		return win32Error{call: "SendInput", code: code}
	}
	return nil
}

// vkNames is the spoken vocabulary of keys. Same list the removed tool had,
// which was the part of it that worked; kept verbatim so anything a user or a
// model learned about the old one still holds.
var vkNames = map[string]uint16{
	"ctrl": 0x11, "control": 0x11, "alt": 0x12, "shift": 0x10, "win": 0x5B,
	"enter": 0x0D, "return": 0x0D, "tab": 0x09, "esc": 0x1B, "escape": 0x1B,
	"space": 0x20, "backspace": 0x08, "delete": 0x2E, "del": 0x2E,
	"insert": 0x2D, "home": 0x24, "end": 0x23, "pageup": 0x21, "pagedown": 0x22,
	"up": 0x26, "down": 0x28, "left": 0x25, "right": 0x27,
}

func vkOf(name string) (uint16, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if vk, ok := vkNames[name]; ok {
		return vk, true
	}
	if len(name) == 1 {
		c := name[0]
		if c >= 'a' && c <= 'z' {
			return uint16(0x41 + (c - 'a')), true
		}
		if c >= '0' && c <= '9' {
			return uint16(0x30 + (c - '0')), true
		}
	}
	if len(name) >= 2 && name[0] == 'f' {
		switch name {
		case "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12":
			n := name[1:]
			v := 0
			for _, d := range n {
				v = v*10 + int(d-'0')
			}
			return uint16(0x70 + v - 1), true
		}
	}
	return 0, false
}

func isModifier(vk uint16) bool {
	return vk == 0x11 || vk == 0x12 || vk == 0x10 || vk == 0x5B
}

// keyEvents turns "ctrl+shift+s" into the press/release batch that means it.
// Modifiers go down in the order written and come up in reverse, which is what
// a person's hands do and what every application expects.
func keyEvents(combo string) ([]keyboardInput, error) {
	parts := strings.Split(strings.TrimSpace(combo), "+")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return nil, refuse("ไม่ได้บอกว่าจะกดปุ่มอะไร", "เช่น `enter`, `ctrl+s`, `ctrl+shift+n`")
	}
	var mods []uint16
	var main uint16
	for i, p := range parts {
		vk, ok := vkOf(p)
		if !ok {
			return nil, refuse(
				"ไม่รู้จักปุ่มชื่อ "+strings.TrimSpace(p),
				"ใช้ชื่อแบบ enter, tab, esc, f5, ลูกศร, ตัวอักษรเดี่ยว หรือรวมด้วย + เช่น ctrl+shift+s")
		}
		if i < len(parts)-1 {
			if !isModifier(vk) {
				return nil, refuse(
					"ปุ่ม "+strings.TrimSpace(p)+" ไม่ใช่ปุ่มค้าง จึงมาก่อน + ไม่ได้",
					"ปุ่มค้างมีแค่ ctrl, alt, shift, win")
			}
			mods = append(mods, vk)
			continue
		}
		main = vk
	}

	out := make([]keyboardInput, 0, len(mods)*2+2)
	for _, m := range mods {
		out = append(out, keyDown(m))
	}
	out = append(out, keyDown(main), keyUp(main))
	for i := len(mods) - 1; i >= 0; i-- {
		out = append(out, keyUp(mods[i]))
	}
	return out, nil
}

func keyDown(vk uint16) keyboardInput { return keyEvent(vk, 0) }
func keyUp(vk uint16) keyboardInput   { return keyEvent(vk, keyEventKeyUp) }

func keyEvent(vk uint16, flags uint32) keyboardInput {
	scan, _, _ := procMapVirtualKeyW.Call(uintptr(vk), mapVKToVSC)
	return keyboardInput{
		typ:     inputKeyboard,
		wVk:     vk,
		wScan:   uint16(scan),
		dwFlags: flags,
	}
}

// reachCursor answers where the pointer is. Not used to aim anything — the
// whole point of the UIA half is that nothing is aimed by coordinate — but it
// is the cheapest probe that fails exactly when there is no input desktop,
// which is what the takeover check wants to know before it raises a banner.
func reachCursor() (int, int, error) {
	var p winPoint
	ok, _, err := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if ok == 0 {
		code := errnoOf(err)
		if code == 0 {
			code = winAccessDenied
		}
		return 0, 0, win32Error{call: "GetCursorPos", code: code}
	}
	return int(p.X), int(p.Y), nil
}

// settle gives a window a moment to redraw after being acted on, so a read that
// follows sees the result rather than the state before it. Bounded and short:
// this is a courtesy to the UI thread, not a synchronisation mechanism, and
// anything that needs real waiting should read again and look.
func settle() { time.Sleep(120 * time.Millisecond) }

var _ = syscall.Errno(0) // keep syscall imported for errnoOf's type switch
