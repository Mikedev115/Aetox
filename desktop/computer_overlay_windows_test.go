//go:build windows

package main

// The light is drawn in Go rather than by a browser, so its geometry is
// something a test can actually check — which matters more here than it looks.
// This is the only signal the user gets while a window they cannot see is being
// driven, and every way it can be wrong is silent: a band drawn one pixel wide,
// a head that never moves, a middle that is not transparent. None of those would
// error, and all of them would leave somebody unable to tell.

import (
	"image/color"
	"testing"
)

func alphaAt(t *testing.T, w, h int, phase float64, x, y int) uint8 {
	t.Helper()
	img := overlayFrame(w, h, phase)
	_, _, _, a := img.At(x, y).RGBA()
	return uint8(a >> 8)
}

func TestTheMiddleOfTheScreenIsUntouched(t *testing.T) {
	const w, h = 400, 300
	if a := alphaAt(t, w, h, 0, w/2, h/2); a != 0 {
		t.Fatalf("the centre of the screen has alpha %d; this is a frame, not a curtain", a)
	}
	// And just inside the band, so an off-by-one in the depth loop shows up.
	if a := alphaAt(t, w, h, 0, overlayBand+2, h/2); a != 0 {
		t.Errorf("the band bleeds %d past its own width", a)
	}
}

func TestTheEdgeIsAlwaysLitSomewhat(t *testing.T) {
	const w, h = 400, 300
	// Every edge carries the base glow even where the head is not, so the frame
	// is a frame at all times rather than a dot travelling round a dark screen.
	for _, p := range []struct {
		name string
		x, y int
	}{
		{"top", w / 2, 0},
		{"bottom", w / 2, h - 1},
		{"left", 0, h / 2},
		{"right", w - 1, h / 2},
	} {
		if a := alphaAt(t, w, h, 0.5, p.x, p.y); a < 20 {
			t.Errorf("the %s edge is at alpha %d, which is not visible against a bright window", p.name, a)
		}
	}
}

func TestTheHeadTravels(t *testing.T) {
	const w, h = 400, 300
	// Brightest point at the top-left at phase 0, and no longer brightest there
	// a third of a lap later. A head that does not move is a static border, and
	// a static border stops being read within seconds.
	start := alphaAt(t, w, h, 0, 2, 0)
	later := alphaAt(t, w, h, 0.33, 2, 0)
	if start <= later {
		t.Fatalf("the light did not move: alpha at the start corner was %d at phase 0 and %d at phase 0.33", start, later)
	}
}

func TestTheLightIsPremultiplied(t *testing.T) {
	const w, h = 200, 150
	// UpdateLayeredWindow composites the bitmap as given. A colour channel above
	// its own alpha is unpremultiplied data, and Windows draws that as a bright
	// halo along every soft edge rather than as a glow.
	img := overlayFrame(w, h, 0.2)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.RGBA)
			if c.R > c.A || c.G > c.A || c.B > c.A {
				t.Fatalf("pixel (%d,%d) is rgba(%d,%d,%d,%d): a channel is brighter than its alpha, so it is not premultiplied",
					x, y, c.R, c.G, c.B, c.A)
			}
		}
	}
}

func TestTheLingerOutlastsOneAction(t *testing.T) {
	// A click and its answer are well under a second. If the light went out
	// between two of them the edge of the screen would flash rather than glow,
	// which reads as a fault and is the light show the beam block in style.css
	// records being taken off the sidebar and the delegation cards.
	if overlayLinger < 1500*1e6 {
		t.Errorf("overlayLinger is %s, short enough that a run of actions strobes", overlayLinger)
	}
}
