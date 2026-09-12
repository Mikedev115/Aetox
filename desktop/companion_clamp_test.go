package main

import "testing"

// The desktop body may be dragged anywhere on the desktop, and its figure can
// never leave it: the middle of the canvas stays inside the work area it is
// nearest, while the transparent margin around it may hang over the edge.
func TestCompanionClampKeepsFigureInsideWorkArea(t *testing.T) {
	area := winRect{Left: 2560, Top: 0, Right: 4480, Bottom: 1032}
	closed := [4]bool{}
	// canvas 400×220, figure 200×100 centred → margins 100, 60
	cases := []struct {
		name         string
		x, y         int
		wantX, wantY int
	}{
		{"inside stays", 3000, 400, 3000, 400},
		{"margin may hang past the right edge", 4150, 400, 4150, 400},
		{"figure past the right edge", 4400, 400, 4480 - 200 - 100, 400},
		{"figure past the bottom (taskbar)", 3000, 1000, 3000, 1032 - 100 - 60},
		{"figure above and left", 2400, -100, 2560 - 100, -60},
	}
	for _, c := range cases {
		x, y := clampInner(c.x, c.y, 400, 220, 200, 100, area, closed)
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
	x, y := clampInner(2450, 400, 400, 220, 200, 100, area, open)
	if x != 2450 || y != 400 {
		t.Errorf("figure crossing the shared edge was held: got %d,%d", x, y)
	}
	x, _ = clampInner(-200, 400, 400, 220, 200, 100, area, open)
	if x != -100 {
		t.Errorf("the outer edge still holds: got %d want -100", x)
	}
}
