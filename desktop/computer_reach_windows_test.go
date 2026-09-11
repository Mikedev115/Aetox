//go:build windows

package main

// The reach, tested against the actual desktop this is running on.
//
// Same argument as uia_windows_test.go's, one layer up: DECISIONS.md §22's
// amendment says a rebuilt desktop reach "comes back with a live end-to-end
// test on a real desktop, not just unit tests over the input-buffer builders."
// A window list assembled from a fake EnumWindows would prove the loop and
// nothing about whether the filter admits real windows or the process lookup
// survives a process this user may not open.
//
// Everything here skips rather than fails when there is no interactive desktop
// or nothing suitable is open. A build agent with an empty session is a true
// fact about that machine; the release gate is the owner running the list in
// docs, on the installed exe, not this file.

import (
	"bytes"
	"image/png"
	"os"
	"strings"
	"testing"
)

// firstReachable finds a window this test may legitimately read: not Aetox, not
// a terminal, not a browser. Returns ok=false when the desktop has none, which
// is the ordinary state of a build agent.
func firstReachable(t *testing.T) (reachTarget, bool) {
	t.Helper()
	windows, err := reachListWindows()
	if err != nil {
		t.Skipf("no reachable desktop here: %v", err)
	}
	for _, w := range windows {
		if tier, _ := appTier(w.Exe); tier == tierFull || tier == tierWarned {
			return w, true
		}
	}
	return reachTarget{}, false
}

func TestListingSeesRealWindowsAndNeverItself(t *testing.T) {
	windows, err := reachListWindows()
	if err != nil {
		t.Skipf("no reachable desktop here: %v", err)
	}
	if len(windows) == 0 {
		t.Skip("no top-level windows open on this machine")
	}

	self := int32(os.Getpid())
	for _, w := range windows {
		if w.PID == self {
			t.Errorf("the list included this very process: %s", w.Label())
		}
		// The filter's whole job. A window with no title or no process is one
		// the taskbar would not show either, and listing it hands the model
		// noise it has no way to act on.
		if strings.TrimSpace(w.Title) == "" {
			t.Errorf("a window with no title got through the filter: pid=%d exe=%q", w.PID, w.Exe)
		}
		if w.PID == 0 {
			t.Errorf("a window with no process got through the filter: %q", w.Title)
		}
	}
	t.Logf("saw %d windows; first is %s", len(windows), windows[0].Label())
}

func TestAWindowCanBeFoundAgainByItsHandle(t *testing.T) {
	w, ok := firstReachable(t)
	if !ok {
		t.Skip("no reachable window open on this machine")
	}
	again, err := reachFindWindow(w.HWND)
	if err != nil {
		t.Fatalf("a window listed a moment ago could not be found again: %v", err)
	}
	if again.PID != w.PID {
		t.Errorf("the handle resolved to a different process: %d then %d", w.PID, again.PID)
	}
}

func TestReadingARealWindowNumbersWhatItFinds(t *testing.T) {
	w, ok := firstReachable(t)
	if !ok {
		t.Skip("no reachable window open on this machine")
	}
	nodes, total, err := reachReadWindow(w.HWND, "", reachReadCap)
	if err != nil {
		t.Skipf("%s would not be read on this machine: %v", w.Label(), err)
	}
	if len(nodes) == 0 {
		t.Skipf("%s exposed nothing actionable", w.Label())
	}

	// Refs are 1..N, in order, with no gaps. Everything downstream — the ref
	// table, the click that has not been written yet, the model's own counting —
	// assumes exactly this.
	for i, n := range nodes {
		if n.Ref != i+1 {
			t.Fatalf("ref %d is at position %d; the numbering has a gap", n.Ref, i+1)
		}
	}
	if total < len(nodes) {
		t.Errorf("the total (%d) is smaller than what was returned (%d)", total, len(nodes))
	}
	if len(nodes) > reachReadCap {
		t.Errorf("the read returned %d rows, over its own cap of %d", len(nodes), reachReadCap)
	}

	// A runtime id on every row. Without one a ref cannot be resolved again
	// after the window repaints, which is the whole reason the table stores ids
	// rather than pointers.
	missing := 0
	for _, n := range nodes {
		if len(n.RuntimeID) == 0 {
			missing++
		}
	}
	if missing == len(nodes) {
		t.Error("not one row carried a runtime id; no ref could survive a repaint")
	}
	t.Logf("%s: %d of %d rows, first is %q %q", w.Label(), len(nodes), total, nodes[0].Role, nodes[0].Name)
}

func TestAFilteredReadNumbersOnlyWhatItKept(t *testing.T) {
	w, ok := firstReachable(t)
	if !ok {
		t.Skip("no reachable window open on this machine")
	}
	all, _, err := reachReadWindow(w.HWND, "", reachReadCap)
	if err != nil || len(all) == 0 {
		t.Skip("nothing to filter on this machine")
	}
	// Filter on something out of the first row's own name, so at least one row
	// must survive.
	needle := strings.TrimSpace(all[0].Name)
	if needle == "" {
		t.Skip("the first row has no name to filter on")
	}
	kept, _, err := reachReadWindow(w.HWND, needle, reachReadCap)
	if err != nil {
		t.Fatalf("the filtered read failed: %v", err)
	}
	if len(kept) == 0 {
		t.Fatalf("filtering on %q kept nothing, though it came from a row that was there", needle)
	}
	if len(kept) > len(all) {
		t.Errorf("the filter widened the read: %d rows became %d", len(all), len(kept))
	}
	for i, n := range kept {
		if n.Ref != i+1 {
			t.Fatalf("a filtered read numbered its rows %d at position %d; every read starts at 1", n.Ref, i+1)
		}
	}
}

func TestCapturingARealWindowProducesAPicture(t *testing.T) {
	w, ok := firstReachable(t)
	if !ok {
		t.Skip("no reachable window open on this machine")
	}
	data, err := reachCaptureWindow(w.HWND)
	if err != nil {
		t.Skipf("%s would not be photographed on this machine: %v", w.Label(), err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("the capture is not a readable PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() < 8 || b.Dy() < 8 {
		t.Fatalf("the capture is %dx%d, which is not a window", b.Dx(), b.Dy())
	}

	// A window that drew nothing comes back as a uniform rectangle, and the
	// blank-image failure browser_shot.go documents is exactly that. Any two
	// different pixels is enough to tell "it drew" from "it did not"; asserting
	// more would be asserting what a particular app looks like.
	first := img.At(b.Min.X, b.Min.Y)
	varied := false
	for y := b.Min.Y; y < b.Max.Y && !varied; y += 7 {
		for x := b.Min.X; x < b.Max.X; x += 7 {
			if img.At(x, y) != first {
				varied = true
				break
			}
		}
	}
	if !varied {
		t.Errorf("the capture of %s is a single flat colour — the window rendered nothing into it", w.Label())
	}
	t.Logf("captured %s at %dx%d, %d KB", w.Label(), b.Dx(), b.Dy(), len(data)/1024)
}
