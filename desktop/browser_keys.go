package main

// The engine half of an action: keystrokes and pointer gestures that go in as
// trusted input.
//
// typeScript (browser.go) decides, on the live element, whether a value can be
// written into it or whether the page will only listen to a keyboard. This
// file is what happens in the second case, and — since 6 ก.ย. — every other
// gesture a person makes with a mouse or a keyboard that a script inside the
// page cannot fake. The events are the engine's own: Chrome DevTools Protocol
// `Input.insertText` for runs of text, `Input.dispatchKeyEvent` for the keys
// that mean something, `Input.dispatchMouseEvent` for the pointer — so the
// page sees trusted events, which is the one thing a script inside the page
// can never give it. Google Docs, Sheets and Slides, Notion, Monaco and
// CodeMirror all keep their document in their own memory and read it back from
// exactly this kind of event; a DOM write reaches none of them (5 ก.ย., see
// typeScript).
//
// Newline and tab are keys and not characters, on purpose. In a document a
// newline is Enter and a tab is Tab, which is what insertText would have
// produced anyway; in a spreadsheet Enter commits the cell and moves down and
// Tab commits and moves right, and `insertText("\t")` would have put a tab
// character inside the cell instead. The one plan serves both, because both
// are what a person's fingers do.
//
// Everything below the executors is a pure function from an intent to a list
// of engine calls, so the shape of every gesture is pinned by a test that
// needs no webview (browser_keys_test.go). The executors are the only part
// that touches a tab.
//
// Portable in the sense tabView.callEngine is portable: on Windows this is
// CDP; a host that cannot answer says so through the error, and the tool
// reports that it did nothing rather than that it did.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// engineCall is one protocol method with its parameters, ready to send, and
// how long to let the page breathe after it.
//
// Pause exists because of column A. The owner's sheet on 5 ก.ย. (23:02) came
// out with every number right and every first cell of a row missing its
// first letters — "แอปเปิ้ล" as "เปิ้ล", "กล้วย" as "ย", "องุ่น" gone — while the
// cells after a Tab were whole. After Enter, Sheets commits the cell, moves
// the selection down and re-arms its editor, and it does the last part a
// frame later; a key that lands inside that frame opens the editor on a
// sink that is still being replaced, and the text sent behind it is read by
// an editor that was initialised from the middle of it. A keyboard cannot
// type inside that frame; the engine can, and did.
type engineCall struct {
	Method string
	Params string
	Pause  time.Duration
}

const (
	// afterMove is the pause after Enter or Tab — the page has a selection
	// to move and an editor to re-arm before the next run.
	afterMove = 120 * time.Millisecond
	// afterOpen is the pause between a run's first key, which opens an
	// editor that was not open, and the text that follows it.
	afterOpen = 60 * time.Millisecond
	// afterChord is the pause after a named key or a chord. An arrow in a
	// sheet moves the selection and re-arms the editor the way Enter does;
	// a shortcut like ctrl+b re-renders the selection. Same frame, same
	// reason as afterMove, shorter because nothing is typed behind it.
	afterChord = 60 * time.Millisecond
	// afterPress is the pause between mousedown and the first move of a
	// drag. Docs anchors a selection on mousedown in a handler that runs a
	// frame later; a move that arrives before it is a move from nowhere.
	afterPress = 60 * time.Millisecond
	// dragStep is the pause between two moves of a drag — one frame, so the
	// page sees a sweep and not a jump.
	dragStep = 16 * time.Millisecond
	// afterNotch is the pause between two notches of the wheel, so a
	// virtualised list renders what the last notch revealed before the next.
	afterNotch = 40 * time.Millisecond
)

// Modifier bits, spelled the way Input.dispatchKeyEvent spells them.
const (
	modAlt   = 1
	modCtrl  = 2
	modMeta  = 4
	modShift = 8
)

// engineKey is a key the plan can press: the DOM key and code, the Windows
// virtual-key code the engine wants alongside them, the character the key
// produces when it produces one, and the modifiers held while it is pressed.
// A key with Text is sent as keyDown, which makes the page see keypress and
// input as well as keydown; one without is rawKeyDown, which is keydown alone.
// Same split as every automation driver makes, because it is the split the
// engine makes.
type engineKey struct {
	Key, Code string
	VK        int
	Text      string
	Modifiers int
}

var (
	keyEnter = engineKey{Key: "Enter", Code: "Enter", VK: 13, Text: "\r"}
	keyTab   = engineKey{Key: "Tab", Code: "Tab", VK: 9}
)

// namedKeys is every key a chord can name, by its canonical spelling. The
// spelling is the DOM `key` value, which is what a model has seen in every
// keydown handler it has ever read.
var namedKeys = map[string]engineKey{
	"Enter":      keyEnter,
	"Tab":        keyTab,
	"Escape":     {Key: "Escape", Code: "Escape", VK: 27},
	"Backspace":  {Key: "Backspace", Code: "Backspace", VK: 8},
	"Delete":     {Key: "Delete", Code: "Delete", VK: 46},
	"Insert":     {Key: "Insert", Code: "Insert", VK: 45},
	"Space":      {Key: " ", Code: "Space", VK: 32, Text: " "},
	"ArrowLeft":  {Key: "ArrowLeft", Code: "ArrowLeft", VK: 37},
	"ArrowUp":    {Key: "ArrowUp", Code: "ArrowUp", VK: 38},
	"ArrowRight": {Key: "ArrowRight", Code: "ArrowRight", VK: 39},
	"ArrowDown":  {Key: "ArrowDown", Code: "ArrowDown", VK: 40},
	"Home":       {Key: "Home", Code: "Home", VK: 36},
	"End":        {Key: "End", Code: "End", VK: 35},
	"PageUp":     {Key: "PageUp", Code: "PageUp", VK: 33},
	"PageDown":   {Key: "PageDown", Code: "PageDown", VK: 34},
	"F1":         {Key: "F1", Code: "F1", VK: 112},
	"F2":         {Key: "F2", Code: "F2", VK: 113},
	"F3":         {Key: "F3", Code: "F3", VK: 114},
	"F4":         {Key: "F4", Code: "F4", VK: 115},
	"F5":         {Key: "F5", Code: "F5", VK: 116},
	"F6":         {Key: "F6", Code: "F6", VK: 117},
	"F7":         {Key: "F7", Code: "F7", VK: 118},
	"F8":         {Key: "F8", Code: "F8", VK: 119},
	"F9":         {Key: "F9", Code: "F9", VK: 120},
	"F10":        {Key: "F10", Code: "F10", VK: 121},
	"F11":        {Key: "F11", Code: "F11", VK: 122},
	"F12":        {Key: "F12", Code: "F12", VK: 123},
}

// keyAliases are the other spellings people use, lower-cased, to a canonical
// name. Modifier aliases live in modifierBits.
var keyAliases = map[string]string{
	"esc": "Escape", "return": "Enter", "del": "Delete", "ins": "Insert",
	"up": "ArrowUp", "down": "ArrowDown", "left": "ArrowLeft", "right": "ArrowRight",
	"pgup": "PageUp", "pgdn": "PageDown", "pgdown": "PageDown",
	"spacebar": "Space",
}

var modifierBits = map[string]int{
	"ctrl": modCtrl, "control": modCtrl,
	"shift": modShift,
	"alt":   modAlt, "option": modAlt,
	"meta": modMeta, "cmd": modMeta, "command": modMeta, "win": modMeta, "windows": modMeta,
}

// lookupKey finds a named key by any spelling, case-insensitively, and hands
// back its canonical name for the answer.
func lookupKey(name string) (engineKey, string, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if canon, ok := keyAliases[lower]; ok {
		return namedKeys[canon], canon, true
	}
	for canon, k := range namedKeys {
		if strings.ToLower(canon) == lower {
			return k, canon, true
		}
	}
	return engineKey{}, "", false
}

// parseChord turns "ctrl+shift+ArrowRight" into one key with its modifiers,
// and hands back the normalised spelling so the answer says what was pressed
// in one consistent form. The last token is the key: a named key, or exactly
// one printable character. Everything before it is a modifier.
//
// Shift on a letter uppercases the character, because that is what shift
// does; ctrl, alt and meta strip the character entirely (see keyPress), because
// ctrl+a is a command and not an "a".
func parseChord(chord string) (engineKey, string, error) {
	chord = strings.TrimSpace(chord)
	if chord == "" {
		return engineKey{}, "", fmt.Errorf("empty key")
	}
	// "+" itself as the key: a trailing "+" after modifiers, or alone.
	parts := strings.Split(chord, "+")
	if strings.HasSuffix(chord, "+") && len(parts) >= 2 && parts[len(parts)-1] == "" {
		parts = append(parts[:len(parts)-2], "+")
	}
	mods, names := 0, []string{}
	for _, m := range parts[:len(parts)-1] {
		bit, ok := modifierBits[strings.ToLower(strings.TrimSpace(m))]
		if !ok {
			return engineKey{}, "", fmt.Errorf("%q is not a modifier (ctrl, shift, alt, meta)", m)
		}
		mods |= bit
		names = append(names, modifierName(bit))
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	var k engineKey
	var canon string
	if nk, name, ok := lookupKey(last); ok {
		k, canon = nk, name
	} else {
		rs := []rune(last)
		if len(rs) != 1 {
			return engineKey{}, "", fmt.Errorf("%q is not a key name — one character, or one of %s", last, keyNamesList())
		}
		r := rs[0]
		if mods&modShift != 0 && r >= 'a' && r <= 'z' {
			r = r - 'a' + 'A'
		}
		k, canon = charKey(r), string(r)
	}
	k.Modifiers = mods
	said := strings.Join(append(names, canon), "+")
	return k, said, nil
}

func modifierName(bit int) string {
	switch bit {
	case modCtrl:
		return "ctrl"
	case modShift:
		return "shift"
	case modAlt:
		return "alt"
	}
	return "meta"
}

// keyNamesList is the vocabulary, for the refusal that names it.
func keyNamesList() string {
	return "Enter, Tab, Escape, Backspace, Delete, Insert, Space, ArrowLeft/Right/Up/Down, Home, End, PageUp, PageDown, F1-F12"
}

// chordPlan is the sequence of engine calls for a run of chords separated by
// whitespace — "ctrl+a Delete" is two presses — with a pause on every release
// so the page has its frame. A bad chord fails the whole plan before anything
// is sent: half a shortcut sequence is worse than none.
func chordPlan(keys string) (plan []engineCall, said []string, err error) {
	for _, chord := range strings.Fields(keys) {
		k, name, err := parseChord(chord)
		if err != nil {
			return nil, nil, err
		}
		press := keyPress(k)
		press[len(press)-1].Pause = afterChord
		plan = append(plan, press...)
		said = append(said, name)
	}
	if len(plan) == 0 {
		return nil, nil, fmt.Errorf("no keys given")
	}
	return plan, said, nil
}

// keystrokePlan is the sequence of engine calls that types text: every run of
// ordinary characters as one real key press for its first character and one
// insertText for the rest, every newline as Enter, every tab as Tab, and one
// more Enter at the end when asked. "\r\n" is one Enter, not two, because
// Windows text arrives that way.
//
// The first character is a key and not text because of what Google Sheets
// did with a run that was all text (5 ก.ย., second live run): the keys reached
// the cell editor, Tab and Enter moved the selection exactly as a keyboard's
// would, and the words between them vanished — B1 held the third run and
// nothing else. A selected cell that is not being edited ignores insertText,
// which is an IME commit with no key behind it; what opens the editor is a
// keydown with a printable character, the way a person's first keystroke
// does. Once the editor is open the rest of the run can go in as text. One
// key press per run is the whole cost, and it is what a keyboard does anyway.
func keystrokePlan(text string, enter bool) []engineCall {
	var plan []engineCall
	var run []rune
	// paced appends a key press whose release is followed by a pause.
	paced := func(k engineKey, pause time.Duration) {
		press := keyPress(k)
		press[len(press)-1].Pause = pause
		plan = append(plan, press...)
	}
	flush := func() {
		if len(run) == 0 {
			return
		}
		if len(run) == 1 {
			plan = append(plan, keyPress(charKey(run[0]))...)
		} else {
			paced(charKey(run[0]), afterOpen)
			p, _ := json.Marshal(map[string]string{"text": string(run[1:])})
			plan = append(plan, engineCall{Method: "Input.insertText", Params: string(p)})
		}
		run = run[:0]
	}
	rs := []rune(text)
	for i := 0; i < len(rs); i++ {
		switch rs[i] {
		case '\r':
			if i+1 < len(rs) && rs[i+1] == '\n' {
				i++
			}
			flush()
			paced(keyEnter, afterMove)
		case '\n':
			flush()
			paced(keyEnter, afterMove)
		case '\t':
			flush()
			paced(keyTab, afterMove)
		default:
			run = append(run, rs[i])
		}
	}
	flush()
	if enter {
		paced(keyEnter, afterMove)
	}
	return plan
}

// charKey is one printable character as a key: the character is both the
// DOM key and the text the key produces, so the page sees keydown, keypress
// and input the way it does for a real keystroke.
//
// Every key carries a virtual-key code, and for a character that has no
// fixed one the code is a letter's. The third live run on the sheet is why:
// a Thai first character sent with no code reached the cell editor while it
// was open and typed fine, and did nothing to a cell that was not being
// edited — the same keydown with VK 13 (Enter) opened the editor every time.
// Sheets decides "is this a character key" from keyCode, as Closure's key
// handling always has, and keyCode 0 is not one. A real Thai keystroke has a
// letter's code too — ห sits on the H key — and the character itself comes
// from the keypress, which the engine builds from Text and not from the code.
// So a letter code is what a keyboard sends, and which letter does not matter.
func charKey(r rune) engineKey {
	k := engineKey{Key: string(r), Text: string(r)}
	switch {
	case r >= 'a' && r <= 'z':
		k.VK = int(r - 'a' + 'A')
		k.Code = "Key" + string(r-'a'+'A')
	case r >= 'A' && r <= 'Z':
		k.VK = int(r)
		k.Code = "Key" + string(r)
	case r >= '0' && r <= '9':
		k.VK = int(r)
		k.Code = "Digit" + string(r)
	case r == ' ':
		k.VK = 32
		k.Code = "Space"
	default:
		letter := 'A' + (r % 26)
		k.VK = int(letter)
		k.Code = "Key" + string(letter)
	}
	return k
}

// keyPress is one key going down and coming back up.
//
// A key held with ctrl, alt or meta loses its character: ctrl+a is a command
// and a page that received keydown, keypress AND an inserted "a" would select
// everything and then overwrite it. Shift keeps it — shift+a is "A", and the
// caller has already uppercased it. This is the rule every automation driver
// applies, because the engine builds the keypress from Text.
func keyPress(k engineKey) []engineCall {
	text := k.Text
	if k.Modifiers&(modCtrl|modAlt|modMeta) != 0 {
		text = ""
	}
	down := map[string]any{"type": "rawKeyDown", "key": k.Key}
	if text != "" {
		down["type"] = "keyDown"
		down["text"] = text
		down["unmodifiedText"] = text
	}
	up := map[string]any{"type": "keyUp", "key": k.Key}
	// Code and the virtual-key codes only where the key has them; a
	// character key with none is sent without, not with a made-up zero.
	for _, m := range []map[string]any{down, up} {
		if k.Code != "" {
			m["code"] = k.Code
		}
		if k.VK != 0 {
			m["windowsVirtualKeyCode"] = k.VK
			m["nativeVirtualKeyCode"] = k.VK
		}
		if k.Modifiers != 0 {
			m["modifiers"] = k.Modifiers
		}
	}
	d, _ := json.Marshal(down)
	u, _ := json.Marshal(up)
	return []engineCall{
		{Method: "Input.dispatchKeyEvent", Params: string(d)},
		{Method: "Input.dispatchKeyEvent", Params: string(u)},
	}
}

// point is a position in CSS pixels of the top-level viewport — the unit the
// page measures in and the unit the model reads off a capture.
type point struct{ X, Y float64 }

func (p point) String() string { return fmt.Sprintf("(%.0f, %.0f)", p.X, p.Y) }

// cssToEngine converts a viewport point to what the engine wants.
//
// Input.dispatchMouseEvent takes device-independent pixels of the view, and
// with the tab's zoom factor at anything but 1 (the device presets,
// BrowserSetZoom) a CSS pixel is that many of them. clickByMouse ignored the
// factor until 6 ก.ย., which put every SVG click under a preset off by the
// preset's own scale; nothing had clicked SVG under a preset yet, so nothing
// had noticed.
func cssToEngine(p point, zoom float64) point {
	if zoom > 0 && zoom != 1 {
		return point{X: p.X * zoom, Y: p.Y * zoom}
	}
	return p
}

func mouseEvent(kind string, p point, extra map[string]any) engineCall {
	m := map[string]any{"type": kind, "x": p.X, "y": p.Y}
	for k, v := range extra {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return engineCall{Method: "Input.dispatchMouseEvent", Params: string(b)}
}

// mouseMove is the pointer arriving somewhere without pressing — a hover.
func mouseMove(p point) []engineCall {
	return []engineCall{mouseEvent("mouseMoved", p, nil)}
}

// mouseButtons is what a caller may ask for. Middle exists because the
// engine has it; nothing in the tool offers it yet.
var mouseButtons = map[string]bool{"left": true, "right": true, "middle": true}

// mousePress is a click at a point: the pointer moves there first (so the page
// has its hover state and a hit-test target, which Slides needs before a
// mousedown means anything), then presses and releases `count` times with the
// engine's own click counter, which is how a double- and triple-click are
// told apart from two singles. The button is left, right or middle.
func mousePress(p point, button string, count int) []engineCall {
	if !mouseButtons[button] {
		button = "left"
	}
	if count < 1 {
		count = 1
	}
	if count > 3 {
		count = 3
	}
	calls := mouseMove(p)
	for i := 1; i <= count; i++ {
		calls = append(calls,
			mouseEvent("mousePressed", p, map[string]any{"button": button, "clickCount": i}),
			mouseEvent("mouseReleased", p, map[string]any{"button": button, "clickCount": i}),
		)
	}
	return calls
}

// mouseClick is one real left click at a point — the gesture that makes an
// editor take the keyboard when a script's `focus()` did not. Real because it
// comes from the engine: an editor that focuses its hidden proxy on mousedown
// does so for this and not for `el.click()`.
func mouseClick(x, y float64) []engineCall {
	return mousePress(point{X: x, Y: y}, "left", 1)
}

// dragSteps is how many moves a drag is made of: one per twelve pixels,
// between six and twenty-four. Enough for a page to see a sweep, few enough
// that a screen-wide drag is under half a second of round trips.
func dragSteps(from, to point) int {
	d := math.Hypot(to.X-from.X, to.Y-from.Y)
	n := int(d / 12)
	if n < 6 {
		n = 6
	}
	if n > 24 {
		n = 24
	}
	return n
}

// mouseDrag presses at one point, sweeps to another, and releases: the way a
// person moves a thing or selects the text under the sweep. `buttons:1` on the
// moves is what tells the engine the left button is still down — without it
// the moves are hover and the page sees a click that never travelled.
func mouseDrag(from, to point, steps int) []engineCall {
	if steps < 1 {
		steps = 1
	}
	calls := mouseMove(from)
	press := mouseEvent("mousePressed", from, map[string]any{"button": "left", "clickCount": 1})
	press.Pause = afterPress
	calls = append(calls, press)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		p := point{X: from.X + (to.X-from.X)*t, Y: from.Y + (to.Y-from.Y)*t}
		mv := mouseEvent("mouseMoved", p, map[string]any{"button": "left", "buttons": 1})
		mv.Pause = dragStep
		calls = append(calls, mv)
	}
	calls = append(calls, mouseEvent("mouseReleased", to, map[string]any{"button": "left", "clickCount": 1}))
	return calls
}

// wheelNotch is what one click of a mouse wheel scrolls, in CSS pixels — the
// figure Chromium itself uses for a notch.
const wheelNotch = 100

// mouseWheel spins the wheel over a point: the distance in notches, each a
// real wheel event, with a pause between so a list that renders on wheel has
// rendered before the next. Positive dy scrolls down.
func mouseWheel(p point, dx, dy float64) []engineCall {
	ticks := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy)) / wheelNotch))
	if ticks < 1 {
		ticks = 1
	}
	var calls []engineCall
	calls = append(calls, mouseMove(p)...)
	for i := 0; i < ticks; i++ {
		ev := mouseEvent("mouseWheel", p, map[string]any{"deltaX": dx / float64(ticks), "deltaY": dy / float64(ticks)})
		ev.Pause = afterNotch
		calls = append(calls, ev)
	}
	return calls
}

// runEngine sends a plan to a tab, honouring every pause, and stops at the
// first refusal naming the method. What went in before the refusal is in the
// page; the caller reports the gesture as a whole as not done, which is the
// honest reading of "some of it".
func (a *App) runEngine(ctx context.Context, id string, plan []engineCall) error {
	host, err := a.browserHostLazy()
	if err != nil {
		return err
	}
	if !host.live(id) {
		return fmt.Errorf("no browser tab %q", id)
	}
	for _, c := range plan {
		if _, err := callEngineOn(ctx, host, id, c.Method, c.Params); err != nil {
			return fmt.Errorf("%s: %w", c.Method, err)
		}
		if c.Pause > 0 {
			time.Sleep(c.Pause)
		}
	}
	return nil
}

// enginePoint is a viewport point in the engine's unit for this tab.
func (a *App) enginePoint(id string, p point) point {
	if a.browsers == nil {
		return p
	}
	return cssToEngine(p, a.browsers.tab(id).zoomFactor())
}

// typeByKeys types text into whatever the tab has focused, clicking first at
// the element when the page script reported that focusing it left nothing
// editable focused. clicked says whether that click was needed, so the answer
// can mention it: a click moves the caret, and a model appending to a
// document should know the caret was placed by a click at the element's
// centre rather than left where it was.
func (a *App) typeByKeys(ctx context.Context, id string, res browserActResult, text string, enter bool) (clicked bool, err error) {
	if !res.Active && res.CX > 0 && res.CY > 0 {
		if err := a.runEngine(ctx, id, mousePress(a.enginePoint(id, point{res.CX, res.CY}), "left", 1)); err != nil {
			return false, err
		}
		clicked = true
		// An editor that takes focus on a click does the taking in its own
		// handlers, some of them a frame later. Cheap next to a round trip.
		time.Sleep(120 * time.Millisecond)
	}
	return clicked, a.runEngine(ctx, id, keystrokePlan(text, enter))
}

// clickByMouse is one real left click at a point, for an element a script's
// `el.click()` cannot reach. Google Slides, 5 ก.ย.: the title placeholder is
// SVG text, SVGElement has no click(), and Slides selects a shape on a real
// mousedown — so the model's click landed nowhere, its typing went into the
// keyboard sink with no shape selected, and the slide stayed "คลิกเพื่อเพิ่มชื่อ".
func (a *App) clickByMouse(ctx context.Context, id string, x, y float64) error {
	return a.pressByMouse(ctx, id, point{x, y}, "left", 1)
}

// pressByMouse is a click of any button, any count, at a point.
func (a *App) pressByMouse(ctx context.Context, id string, p point, button string, count int) error {
	return a.runEngine(ctx, id, mousePress(a.enginePoint(id, p), button, count))
}

// hoverByMouse moves the pointer to a point and leaves it there.
func (a *App) hoverByMouse(ctx context.Context, id string, p point) error {
	return a.runEngine(ctx, id, mouseMove(a.enginePoint(id, p)))
}

// dragByMouse sweeps from one point to another with the left button held.
func (a *App) dragByMouse(ctx context.Context, id string, from, to point) error {
	return a.runEngine(ctx, id, mouseDrag(a.enginePoint(id, from), a.enginePoint(id, to), dragSteps(from, to)))
}

// wheelByMouse spins the wheel over a point.
func (a *App) wheelByMouse(ctx context.Context, id string, p point, dx, dy float64) error {
	return a.runEngine(ctx, id, mouseWheel(a.enginePoint(id, p), dx, dy))
}

// sendKeys presses chords at the page's focus. The plan is built — and
// refused — before the engine is touched.
func (a *App) sendKeys(ctx context.Context, id string, keys string) (said []string, err error) {
	plan, said, err := chordPlan(keys)
	if err != nil {
		return nil, err
	}
	return said, a.runEngine(ctx, id, plan)
}

// dragDuration is how long a drag's sweep takes on the wire, so the trail
// drawn on the page can keep step with the pointer.
func dragDuration(from, to point) time.Duration {
	return afterPress + time.Duration(dragSteps(from, to))*dragStep
}
