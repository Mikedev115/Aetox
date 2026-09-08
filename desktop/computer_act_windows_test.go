//go:build windows

package main

// The key batch, tested as a batch rather than as a parser.
//
// This is the one piece of the removed tool that survives, and the removed tool
// had unit tests over exactly this and shipped broken anyway (DECISIONS.md §22).
// So the assertions here are deliberately about the SHAPE of what reaches
// Windows — how many events, in what order, with what set on them — rather than
// about the parser agreeing with itself. A modifier released before the key it
// modifies is a combination that silently does the wrong thing, and no amount of
// checking that "ctrl" maps to 0x11 would catch it.

import (
	"strings"
	"testing"
)

func TestASingleKeyIsOneDownAndOneUp(t *testing.T) {
	ev, err := keyEvents("enter")
	if err != nil {
		t.Fatalf("enter would not parse: %v", err)
	}
	if len(ev) != 2 {
		t.Fatalf("enter produced %d events, want 2", len(ev))
	}
	if ev[0].wVk != 0x0D || ev[0].dwFlags != 0 {
		t.Errorf("first event is not a plain key-down: %+v", ev[0])
	}
	if ev[1].wVk != 0x0D || ev[1].dwFlags != keyEventKeyUp {
		t.Errorf("second event is not the matching key-up: %+v", ev[1])
	}
}

func TestModifiersComeUpInReverseOrder(t *testing.T) {
	ev, err := keyEvents("ctrl+shift+s")
	if err != nil {
		t.Fatalf("ctrl+shift+s would not parse: %v", err)
	}
	// ctrl↓ shift↓ s↓ s↑ shift↑ ctrl↑
	if len(ev) != 6 {
		t.Fatalf("ctrl+shift+s produced %d events, want 6", len(ev))
	}
	want := []struct {
		vk uint16
		up bool
	}{
		{0x11, false}, {0x10, false}, {'S', false},
		{'S', true}, {0x10, true}, {0x11, true},
	}
	for i, w := range want {
		gotUp := ev[i].dwFlags&keyEventKeyUp != 0
		if ev[i].wVk != w.vk || gotUp != w.up {
			t.Errorf("event %d is vk=%#x up=%v, want vk=%#x up=%v", i, ev[i].wVk, gotUp, w.vk, w.up)
		}
	}
}

// The scancode is the difference between this and what was removed. An
// application reading raw input — a game, a terminal, a remote-desktop client —
// sees the scancode and not the virtual key, and a zero there is a keystroke
// that arrives looking like nothing was pressed.
func TestEveryEventCarriesAScancode(t *testing.T) {
	ev, err := keyEvents("ctrl+a")
	if err != nil {
		t.Fatalf("ctrl+a would not parse: %v", err)
	}
	for i, e := range ev {
		if e.wScan == 0 {
			t.Errorf("event %d (vk %#x) has no scancode", i, e.wVk)
		}
	}
}

func TestAnUnknownKeyIsNamedRatherThanIgnored(t *testing.T) {
	_, err := keyEvents("ctrl+banana")
	if err == nil {
		t.Fatal("an unknown key was accepted")
	}
	got := explainReach("type", err)
	if !strings.Contains(got, "banana") {
		t.Errorf("the refusal does not say which key it did not know: %q", got)
	}
	// And it teaches the vocabulary, so the next attempt is not another guess.
	if !strings.Contains(got, "ctrl+shift+s") {
		t.Errorf("the refusal does not show the shape that works: %q", got)
	}
}

func TestOnlyAHoldableKeyMayComeBeforeThePlus(t *testing.T) {
	_, err := keyEvents("s+ctrl")
	if err == nil {
		t.Fatal("a letter was accepted as a modifier")
	}
	got := explainReach("type", err)
	if !strings.Contains(got, "ctrl, alt, shift, win") {
		t.Errorf("the refusal does not name what a modifier is: %q", got)
	}
}

func TestAnEmptyComboIsRefusedWithAnExample(t *testing.T) {
	_, err := keyEvents("   ")
	if err == nil {
		t.Fatal("an empty combination was accepted")
	}
	if got := explainReach("type", err); !strings.Contains(got, "ctrl+s") {
		t.Errorf("the refusal gives no example: %q", got)
	}
}

func TestFunctionKeysAndArrowsAreKnown(t *testing.T) {
	for name, want := range map[string]uint16{
		"f1": 0x70, "f12": 0x7B, "up": 0x26, "down": 0x28,
		"home": 0x24, "end": 0x23, "esc": 0x1B, "tab": 0x09,
	} {
		got, ok := vkOf(name)
		if !ok {
			t.Errorf("%s is not a key this tool knows", name)
			continue
		}
		if got != want {
			t.Errorf("%s is %#x, want %#x", name, got, want)
		}
	}
}

// The INPUT struct's size is the one thing here that fails silently and
// completely: SendInput takes an array and a stride, so a struct one field too
// short makes it read every event after the first out of the middle of its
// neighbour. The removed tool got this right; this test is here so a refactor
// cannot get it wrong.
func TestTheInputStructIsTheSizeWindowsExpects(t *testing.T) {
	const win32INPUTSizeOnAMD64 = 40
	if got := sizeOfKeyboardInput(); got != win32INPUTSizeOnAMD64 {
		t.Fatalf("keyboardInput is %d bytes, but Windows reads INPUT as %d on this architecture",
			got, win32INPUTSizeOnAMD64)
	}
}
