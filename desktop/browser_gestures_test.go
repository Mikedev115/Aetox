package main

// The rest of the mouse and the keyboard, pinned without a webview.
//
// Every gesture is a pure plan (browser_keys.go) and a pure answer
// (browser_gestures.go) around one engine call that cannot run here. What can
// be pinned is the shape of the plan — which is the part that fails silently
// in a browser, by pressing the wrong thing in the right place — and the words
// of the answer, which are what stop "ลากแล้ว" from becoming "เสร็จแล้ว".

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func shapeOf(t *testing.T, plan []engineCall) []string {
	t.Helper()
	var out []string
	for _, c := range plan {
		var p map[string]any
		if err := json.Unmarshal([]byte(c.Params), &p); err != nil {
			t.Fatalf("%s params are not JSON: %v\n%s", c.Method, err, c.Params)
		}
		switch c.Method {
		case "Input.dispatchMouseEvent":
			s := p["type"].(string)
			if b, ok := p["button"]; ok {
				s += ":" + b.(string)
			}
			if n, ok := p["clickCount"].(float64); ok {
				s += fmt.Sprintf("x%d", int(n))
			}
			if _, ok := p["buttons"]; ok {
				s += "+held"
			}
			out = append(out, s)
		case "Input.dispatchKeyEvent":
			out = append(out, p["type"].(string)+":"+p["key"].(string))
		default:
			out = append(out, c.Method)
		}
	}
	return out
}

func TestMousePressCountsAndButtons(t *testing.T) {
	// A double-click is two presses with the engine's own click counter at 1
	// then 2 — that is how the page tells it from two singles. The pointer
	// moves there first so the page has a hover state and a hit-test target.
	got := shapeOf(t, mousePress(point{10, 20}, "left", 2))
	want := []string{"mouseMoved", "mousePressed:leftx1", "mouseReleased:leftx1", "mousePressed:leftx2", "mouseReleased:leftx2"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("double-click = %v\nwant %v", got, want)
	}
	right := shapeOf(t, mousePress(point{1, 1}, "right", 1))
	if right[1] != "mousePressed:rightx1" {
		t.Errorf("right click must press the right button, got %v", right)
	}
	if bad := shapeOf(t, mousePress(point{1, 1}, "sideways", 9)); bad[1] != "mousePressed:leftx1" || len(bad) != 1+2*3 {
		t.Errorf("an unknown button is left and a count past three is three, got %v", bad)
	}
	if one := mouseClick(3, 4); len(one) != 3 {
		t.Errorf("mouseClick is still one left press, got %d calls", len(one))
	}
}

func TestMouseDragSweepsWithTheButtonHeld(t *testing.T) {
	// Press, N moves with `buttons:1` (without it the moves are hover and
	// the page sees a click that never travelled), release at the end.
	plan := mouseDrag(point{0, 0}, point{120, 0}, 4)
	got := shapeOf(t, plan)
	want := []string{"mouseMoved", "mousePressed:leftx1", "mouseMoved:left+held", "mouseMoved:left+held", "mouseMoved:left+held", "mouseMoved:left+held", "mouseReleased:leftx1"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("drag = %v\nwant %v", got, want)
	}
	if plan[1].Pause != afterPress {
		t.Errorf("the press must pause before the first move (Docs anchors the selection a frame later), got %v", plan[1].Pause)
	}
	if plan[2].Pause != dragStep {
		t.Errorf("each move is one frame apart, got %v", plan[2].Pause)
	}
	var last map[string]any
	_ = json.Unmarshal([]byte(plan[len(plan)-1].Params), &last)
	if last["x"] != float64(120) {
		t.Errorf("the release must be at the destination, got %v", last)
	}
	if dragSteps(point{0, 0}, point{10, 0}) != 6 || dragSteps(point{0, 0}, point{9000, 0}) != 24 {
		t.Errorf("steps clamp to [6,24], got %d and %d", dragSteps(point{0, 0}, point{10, 0}), dragSteps(point{0, 0}, point{9000, 0}))
	}
}

func TestMouseWheelIsNotchesThatSumToTheDistance(t *testing.T) {
	plan := mouseWheel(point{5, 5}, 0, 450)
	var sum float64
	notches := 0
	for _, c := range plan[1:] {
		var p map[string]any
		_ = json.Unmarshal([]byte(c.Params), &p)
		if p["type"] != "mouseWheel" {
			t.Fatalf("expected mouseWheel, got %v", p["type"])
		}
		sum += p["deltaY"].(float64)
		notches++
		if c.Pause != afterNotch {
			t.Errorf("a notch pauses for the list to render, got %v", c.Pause)
		}
	}
	if notches != 5 || sum < 449.9 || sum > 450.1 {
		t.Errorf("450px is five notches summing to 450, got %d notches summing to %v", notches, sum)
	}
}

func TestChordsCarryModifiersAndDropTheCharacter(t *testing.T) {
	k, said, err := parseChord("ctrl+shift+ArrowRight")
	if err != nil {
		t.Fatal(err)
	}
	if k.Modifiers != modCtrl|modShift || k.Key != "ArrowRight" || said != "ctrl+shift+ArrowRight" {
		t.Errorf("parseChord = %+v %q", k, said)
	}
	// ctrl+a is a command: keydown without the character, or the page
	// selects everything and then types an "a" over it.
	plan := keyPress(mustChord(t, "ctrl+a"))
	var down map[string]any
	_ = json.Unmarshal([]byte(plan[0].Params), &down)
	if down["type"] != "rawKeyDown" || down["modifiers"] != float64(modCtrl) {
		t.Errorf("ctrl+a must be rawKeyDown with modifiers 2, got %v", down)
	}
	if _, has := down["text"]; has {
		t.Errorf("ctrl+a must not carry a character, got %v", down)
	}
	// shift keeps it, uppercased.
	plan = keyPress(mustChord(t, "shift+a"))
	_ = json.Unmarshal([]byte(plan[0].Params), &down)
	if down["type"] != "keyDown" || down["text"] != "A" || down["modifiers"] != float64(modShift) {
		t.Errorf("shift+a must be keyDown 'A' with modifiers 8, got %v", down)
	}
	// Aliases and case.
	for in, want := range map[string]string{"esc": "Escape", "RETURN": "Enter", "pgdn": "PageDown", "cmd+c": "meta+c", "Control+End": "ctrl+End"} {
		if _, said, err := parseChord(in); err != nil || said != want {
			t.Errorf("parseChord(%q) = %q, %v; want %q", in, said, err, want)
		}
	}
	// Every named key has a code the page can act on, and only Enter and
	// Space produce a character.
	for name, k := range namedKeys {
		if k.VK == 0 {
			t.Errorf("%s has no virtual-key code", name)
		}
		if (k.Text != "") != (name == "Enter" || name == "Space") {
			t.Errorf("%s text = %q", name, k.Text)
		}
	}
}

func mustChord(t *testing.T, s string) engineKey {
	t.Helper()
	k, _, err := parseChord(s)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestChordPlanRefusesBeforeTheEngine(t *testing.T) {
	if _, _, err := chordPlan("ctrl+a Bogus"); err == nil {
		t.Error("an unknown key must refuse the whole plan")
	} else if !strings.Contains(err.Error(), "Bogus") {
		t.Errorf("the refusal must name the key, got %v", err)
	}
	if _, _, err := chordPlan("hyper+a"); err == nil || !strings.Contains(err.Error(), "modifier") {
		t.Errorf("an unknown modifier must be refused as one, got %v", err)
	}
	plan, said, err := chordPlan("ctrl+a Delete")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 4 || strings.Join(said, " ") != "ctrl+a Delete" {
		t.Errorf("two chords are four events named back, got %d and %v", len(plan), said)
	}
	if plan[1].Pause != afterChord || plan[3].Pause != afterChord {
		t.Errorf("every release pauses for the page's frame, got %v %v", plan[1].Pause, plan[3].Pause)
	}
}

func TestCSSToEngineFollowsTheZoom(t *testing.T) {
	// Device presets zoom the view; the engine's pointer takes the view's
	// pixels, so a CSS point has to be scaled by the preset or it lands off
	// by exactly the preset. Zero means "never set", which is 1.
	if p := cssToEngine(point{100, 50}, 2); p.X != 200 || p.Y != 100 {
		t.Errorf("zoom 2 = %v", p)
	}
	if p := cssToEngine(point{100, 50}, 0); p.X != 100 || p.Y != 50 {
		t.Errorf("zoom unset = %v", p)
	}
}

func TestTargetFromReadsQuotedNumbers(t *testing.T) {
	// Models quote numbers (tool_runs, twelve times for ref); a coordinate
	// read off a picture arrives the same way and must not become zero.
	tg := targetFrom(map[string]any{"x": "412", "y": 88.5}, "ref", "find", "x", "y")
	if !tg.HasXY || tg.X != 412 || tg.Y != 88.5 {
		t.Errorf("targetFrom = %+v", tg)
	}
	if !targetFrom(map[string]any{"x": 1}, "ref", "find", "x", "y").empty() {
		t.Error("x without y is not a point")
	}
	if tg := targetFrom(map[string]any{"toFind": " แทรก "}, "toRef", "toFind", "toX", "toY"); tg.Find != "แทรก" {
		t.Errorf("the second target reads its own names, got %+v", tg)
	}
}

func TestPointerAnswersSayWhereAndWhatTheyCannotSee(t *testing.T) {
	byPoint := browserTarget{X: 300, Y: 200, HasXY: true}
	res := browserActResult{Found: true, CX: 300, CY: 200, Under: "canvas.grid", CanvasShare: 0.5}
	got := pointerMessage("left", 1, byPoint, res)
	for _, want := range []string{"(300, 200)", "canvas.grid", "capture"} {
		if !strings.Contains(got, want) {
			t.Errorf("a click by point must name the point, what was under it, and capture on a painted page; missing %q in %s", want, got)
		}
	}
	right := pointerMessage("right", 1, browserTarget{Ref: 3}, browserActResult{Found: true, Ref: 3, Tag: "text", Label: "Title", CX: 10, CY: 10})
	if !strings.Contains(right, "คลิกขวา") || !strings.Contains(right, "Escape") {
		t.Errorf("a right click must say so and name Escape for the menu it cannot see, got %s", right)
	}
	if !strings.Contains(pointerMessage("left", 2, browserTarget{Ref: 1}, browserActResult{Ref: 1, Tag: "p"}), "ดับเบิลคลิก") {
		t.Error("a double click must say so")
	}
	keys := keyActMessage([]string{"ctrl+b"}, browserActResult{Focus: "div.kix(editable)", CanvasShare: 0.6})
	if !strings.Contains(keys, "ctrl+b") || !strings.Contains(keys, "div.kix(editable)") || !strings.Contains(keys, "capture") {
		t.Errorf("a chord must name the keys, the focus, and capture on a painted page, got %s", keys)
	}
	up := uploadMessage("page-1.png", 2048, browserActResult{Ref: 4, Tag: "input", Label: "รูป", Multiple: true, Accept: "image/*"})
	for _, want := range []string{"page-1.png", "2 KB", "ยังไม่ได้อัปโหลด", "image/*", "หลายไฟล์"} {
		if !strings.Contains(up, want) {
			t.Errorf("upload's answer missing %q: %s", want, up)
		}
	}
}

func TestPointScriptsMeasureInTheTopViewport(t *testing.T) {
	// A point script reports and acts on nothing; the centre it reports
	// walks up through frames, so a shape in a same-origin iframe is where
	// the top viewport says it is.
	for name, js := range map[string]string{
		"point":  pointScript("tok", 5),
		"under":  underScript("tok", 10, 20),
		"focus":  focusScript("tok"),
		"view":   viewportScript("tok"),
		"state":  stateScript("tok"),
		"upload": fileInputScript("tok", 2),
	} {
		if !strings.Contains(js, "aetoxReport(") {
			t.Errorf("%s does not report", name)
		}
		for _, act := range []string{"el.click()", ".set.call(", "textContent=", "insertText"} {
			if strings.Contains(js, act) {
				t.Errorf("%s acts on the page (%s); it must only measure", name, act)
			}
		}
	}
	if !strings.Contains(pointScript("tok", 5), "frameElement") {
		t.Error("pointScript does not walk up through frames")
	}
	if !strings.Contains(underScript("tok", 1, 2), "inside=x>=0&&y>=0&&x<=extra.vw&&y<=extra.vh") {
		t.Error("underScript does not check the viewport")
	}
}

func TestCursorTravelIsBounded(t *testing.T) {
	if d := cursorTravel(point{0, 0}, point{3000, 3000}, true); d != cursorTravelMax {
		t.Errorf("a long trip is capped at %v, got %v", cursorTravelMax, d)
	}
	if d := cursorTravel(point{0, 0}, point{10, 0}, true); d < 120*1e6 {
		t.Errorf("a short hop is still a visible move, got %v", d)
	}
	if d := cursorTravel(point{0, 0}, point{900, 900}, false); d != 120*1e6 {
		t.Errorf("from nowhere the sprite appears, got %v", d)
	}
}

// find for text that is not a control: the deepest element carrying the
// words is the one meant, one match is tagged as ref 1 for the point script,
// and Aetox's own overlays are never a match.
func TestFindTextScriptTagsTheDeepestSingleMatch(t *testing.T) {
	js := findTextScript("tok", "hover me")
	for _, want := range []string{
		`removeAttribute('data-aetox-ref')`,      // stale refs go first, as in every read
		`hits[a].contains(hits[b])`,              // an ancestor of a match is not the match
		`one.setAttribute('data-aetox-ref','1')`, // exactly one becomes ref 1
		`indexOf("__aetox")===0)continue`,        // the cursor and the ring are not the page
		`"hover me"`,                             // the text travels as a JSON literal
	} {
		if !strings.Contains(js, want) {
			t.Errorf("findTextScript missing %q", want)
		}
	}
	if strings.Contains(js, "el.click()") {
		t.Error("findTextScript must only tag, never act")
	}
}
