//go:build windows

package main

import "testing"

// The desktop body may be dragged anywhere on the desktop, and its figure can
// never leave it: the figure stays inside the work area it is nearest.
func TestCompanionClampKeepsFigureInsideWorkArea(t *testing.T) {
	area := winRect{Left: 2560, Top: 0, Right: 4480, Bottom: 1032}
	closed := [4]bool{}
	cases := []struct {
		name         string
		x, y         int
		wantX, wantY int
	}{
		{"inside stays", 3000, 400, 3000, 400},
		{"past the right edge", 4400, 400, 4480 - 200, 400},
		{"past the bottom (taskbar)", 3000, 1000, 3000, 1032 - 100},
		{"above and left", 2400, -100, 2560, 0},
	}
	for _, c := range cases {
		x, y := clampRect(c.x, c.y, 200, 100, area, closed)
		if x != c.wantX || y != c.wantY {
			t.Errorf("%s: got %d,%d want %d,%d", c.name, x, y, c.wantX, c.wantY)
		}
	}
}

// An edge with another monitor beyond it does not hold the figure: it may
// cross, or its centre could never reach the other side.
func TestCompanionClampLeavesSharedEdgesOpen(t *testing.T) {
	area := winRect{Left: 0, Top: 0, Right: 2560, Bottom: 1516}
	open := [4]bool{false, false, true, false} // a monitor to the right
	x, y := clampRect(2450, 400, 200, 100, area, open)
	if x != 2450 || y != 400 {
		t.Errorf("figure crossing the shared edge was held: got %d,%d", x, y)
	}
	x, _ = clampRect(-200, 400, 200, 100, area, open)
	if x != 0 {
		t.Errorf("the outer edge still holds: got %d want 0", x)
	}
}
