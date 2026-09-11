package main

// What a type by keystrokes sends, pinned without a webview.
//
// The plan is the part that can be wrong quietly: a tab sent as text puts a
// tab character inside a spreadsheet cell instead of moving to the next one,
// an Enter without its "\r" reaches a page as keydown alone and never makes a
// paragraph, "\r\n" sent as two Enters doubles every blank line in text that
// came from Windows, and a run sent as text alone is ignored by a spreadsheet
// cell that is not yet being edited. None of those fail; they type the wrong
// thing.

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestKeystrokePlanSendsTabAndNewlineAsKeys(t *testing.T) {
	// Each run opens with a real key press — Google Sheets ignores insertText
	// on a cell that is not being edited, and a printable keydown is what
	// opens the editor — and continues as text. Tab and newline are keys.
	plan := keystrokePlan("หัวข้อ\tคำอธิบาย\r\nA", false)
	var shape []string
	for _, c := range plan {
		var p map[string]any
		if err := json.Unmarshal([]byte(c.Params), &p); err != nil {
			t.Fatalf("%s params are not JSON: %v\n%s", c.Method, err, c.Params)
		}
		switch c.Method {
		case "Input.insertText":
			shape = append(shape, "text:"+p["text"].(string))
		case "Input.dispatchKeyEvent":
			shape = append(shape, p["type"].(string)+":"+p["key"].(string))
		default:
			t.Fatalf("unexpected method %s", c.Method)
		}
	}
	want := []string{
		"keyDown:ห", "keyUp:ห", "text:ัวข้อ",
		"rawKeyDown:Tab", "keyUp:Tab",
		"keyDown:ค", "keyUp:ค", "text:ำอธิบาย",
		"keyDown:Enter", "keyUp:Enter",
		"keyDown:A", "keyUp:A",
	}
	if strings.Join(shape, " ") != strings.Join(want, " ") {
		t.Errorf("plan = %v\nwant %v", shape, want)
	}
}

func TestCharKeyCarriesTheCharacterAndALetterCode(t *testing.T) {
	// A Thai character has no fixed virtual-key code, and a keydown with code
	// 0 does not open a spreadsheet cell for editing (third live run, 5 ก.ย.).
	// A real ห comes off the H key; any letter's code will do, and the
	// character itself still travels as key and text.
	var thai, latin map[string]any
	_ = json.Unmarshal([]byte(keyPress(charKey('ห'))[0].Params), &thai)
	_ = json.Unmarshal([]byte(keyPress(charKey('a'))[0].Params), &latin)
	if thai["type"] != "keyDown" || thai["text"] != "ห" || thai["key"] != "ห" {
		t.Errorf("a Thai letter must go as keyDown with itself as key and text, got %v", thai)
	}
	if vk, _ := thai["windowsVirtualKeyCode"].(float64); vk < 'A' || vk > 'Z' {
		t.Errorf("a character with no code of its own rides on a letter key, got %v", thai)
	}
	if latin["windowsVirtualKeyCode"] != float64('A') || latin["code"] != "KeyA" || latin["text"] != "a" {
		t.Errorf("an ASCII letter carries its code and VK, got %v", latin)
	}
}

func TestKeystrokePlanEnterCarriesItsCharacter(t *testing.T) {
	// keyDown with text is what makes the page see keypress and input as well
	// as keydown; rawKeyDown is keydown alone. Enter needs the former to make
	// a paragraph in an editor, Tab needs nothing but the key.
	plan := keystrokePlan("", true)
	if len(plan) != 2 {
		t.Fatalf("enter alone should be one key down and up, got %d calls", len(plan))
	}
	var down map[string]any
	_ = json.Unmarshal([]byte(plan[0].Params), &down)
	if down["type"] != "keyDown" || down["text"] != "\r" || down["windowsVirtualKeyCode"] != float64(13) {
		t.Errorf("Enter keyDown must carry text \\r and VK 13, got %s", plan[0].Params)
	}
	var tab map[string]any
	_ = json.Unmarshal([]byte(keyPress(keyTab)[0].Params), &tab)
	if tab["type"] != "rawKeyDown" || tab["windowsVirtualKeyCode"] != float64(9) {
		t.Errorf("Tab must be rawKeyDown with VK 9, got %s", keyPress(keyTab)[0].Params)
	}
	if _, has := tab["text"]; has {
		t.Errorf("Tab must not carry a character, got %s", keyPress(keyTab)[0].Params)
	}
}

func TestKeystrokePlanPausesWhereThePageNeedsAFrame(t *testing.T) {
	// Column A, 5 ก.ย. 23:02: every first cell of a row lost its first letters
	// because the run after Enter arrived inside the frame Sheets uses to
	// move the selection and re-arm its editor. The pause rides on the key
	// release after Enter and Tab, and on the first key of a run that has
	// text behind it; text itself never pauses.
	plan := keystrokePlan("ab\tc\nd", false)
	var got []string
	for _, c := range plan {
		var p map[string]any
		_ = json.Unmarshal([]byte(c.Params), &p)
		name := c.Method
		if c.Method == "Input.dispatchKeyEvent" {
			name = p["type"].(string) + ":" + p["key"].(string)
		}
		if c.Pause > 0 {
			name += "+pause"
		}
		got = append(got, name)
	}
	want := []string{
		"keyDown:a", "keyUp:a+pause", "Input.insertText",
		"rawKeyDown:Tab", "keyUp:Tab+pause",
		"keyDown:c", "keyUp:c",
		"keyDown:Enter", "keyUp:Enter+pause",
		"keyDown:d", "keyUp:d",
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("plan = %v\nwant %v", got, want)
	}
}

func TestKeystrokePlanOfNothingIsNothing(t *testing.T) {
	if plan := keystrokePlan("", false); len(plan) != 0 {
		t.Errorf("empty text with no enter should send nothing, got %v", plan)
	}
}

func TestTypedByKeysMessageSaysWhatReadCannotSee(t *testing.T) {
	// The sentence this exists for. A type into Google Sheets that ends with
	// "ใช้ read ดูผล" sends the model to a read that shows the toolbar, and
	// the toolbar looks the same whether the cells are full or empty.
	painted := typedByKeysMessage(`textarea ref 19`, "a\tb\n", false, false, browserActResult{Active: true, CanvasShare: 0.6, Focus: "textarea.sink", FocusBefore: "body"})
	if !strings.Contains(painted, "capture") || strings.Contains(painted, "ใช้ read ดูผล") {
		t.Errorf("on a canvas page the answer must send the model to capture, not read, got: %s", painted)
	}
	if !strings.Contains(painted, "Tab") || !strings.Contains(painted, "Enter") {
		t.Errorf("text with tabs and newlines must say they went in as keys, got: %s", painted)
	}
	plain := typedByKeysMessage(`div "editor" (ref 3)`, "hello", true, true, browserActResult{Focus: "div#editor(editable)"})
	if !strings.Contains(plain, "ใช้ read ดูผล") || !strings.Contains(plain, "Enter") {
		t.Errorf("a plain editor still gets read as the check and Enter named, got: %s", plain)
	}
	if !strings.Contains(plain, "คลิกจริง") {
		t.Errorf("a click that placed the caret must be mentioned, got: %s", plain)
	}
	if !strings.Contains(plain, "5 ตัวอักษร") {
		t.Errorf("the count is what the model can compare with what it sent, got: %s", plain)
	}
	// Keys go to the page's focus, not to a ref. Naming that element is what
	// lets the model tell a sheet from a textarea the sheet ignores.
	if !strings.Contains(painted, "textarea.sink") || !strings.Contains(painted, "body") {
		t.Errorf("the answer must name where focus was and where it went, got: %s", painted)
	}
	kept := typedByKeysMessage(`textarea ref 25`, "x", false, false, browserActResult{Active: true, Kept: true, Focus: "div#waffle-rich-text-editor.cell-input(editable)"})
	if !strings.Contains(kept, "ไม่ย้าย focus") || !strings.Contains(kept, "cell-input") {
		t.Errorf("a type that left focus with the page's editor must say so and name it, got: %s", kept)
	}
	lost := typedByKeysMessage(`textarea ref 2`, "x", false, false, browserActResult{Active: false, CX: -900, Focus: "body"})
	if !strings.Contains(lost, "อาจไม่ใช่ที่ตั้งใจ") {
		t.Errorf("focus that never landed and could not be clicked must be said out loud, got: %s", lost)
	}
}
