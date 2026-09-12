package main

import (
	"math"
	"testing"
	"time"
)

// The desktop drag is Companion.svelte's drag: these are that component's
// rules, checked on the port.

func TestNearAngleGoesTheShortWayRound(t *testing.T) {
	cases := []struct{ h, prev, want float64 }{
		{350, 10, -10},
		{-170, 170, 190},
		{90, 0, 90},
		{180, 0, 180}, // the seam goes the positive way
		{0, 350, 360},
	}
	for _, c := range cases {
		if got := nearAngle(c.h, c.prev); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("nearAngle(%v, %v) = %v, want %v", c.h, c.prev, got, c.want)
		}
	}
}

func TestWalkTurnReadsTheScreensAxes(t *testing.T) {
	// right is +90, left -90, down the screen (towards the viewer) 0, up 180
	if h := walkTurn(10, 0, 0, 3); math.Abs(h-90) > 1e-9 {
		t.Errorf("right: %v", h)
	}
	if h := walkTurn(-10, 0, 0, 3); math.Abs(h+90) > 1e-9 {
		t.Errorf("left: %v", h)
	}
	if h := walkTurn(0, 10, 0, 3); math.Abs(h) > 1e-9 {
		t.Errorf("down: %v", h)
	}
	if h := walkTurn(0, -10, 0, 3); math.Abs(math.Abs(h)-180) > 1e-9 {
		t.Errorf("up: %v", h)
	}
	if h := walkTurn(1, 1, 45, 3); h != 45 {
		t.Errorf("a short step keeps the heading: %v", h)
	}
}

// The head may turn no faster than walkTurnDegS, and a wobble of the hand
// does not retarget it.
func TestWalkerTurnsNoFasterThanTheLimit(t *testing.T) {
	now := time.Unix(0, 0)
	w := newWalker(100, 100, 0, 1, now)
	// one 16ms frame stepping right: at most 240°/s × .016 = 3.84°
	now = now.Add(16 * time.Millisecond)
	w.move(110, 100, now)
	if w.heading <= 0 || w.heading > walkTurnDegS*0.016+1e-9 {
		t.Fatalf("heading after one frame: %v", w.heading)
	}
	// keep stepping right for a second: it gets there
	for i := 0; i < 60; i++ {
		now = now.Add(16 * time.Millisecond)
		w.move(w.targetX+10, 100, now)
	}
	if math.Abs(w.heading-90) > 1e-6 {
		t.Fatalf("heading after a second: %v", w.heading)
	}
	// a 20° wobble is not a new direction; a turn-around is
	now = now.Add(16 * time.Millisecond)
	w.move(w.targetX+10, w.targetY+3.6, now) // ≈ 70°
	if w.want != 90 {
		t.Fatalf("a wobble retargeted the walk to %v", w.want)
	}
	now = now.Add(16 * time.Millisecond)
	w.move(w.targetX-10, w.targetY, now)
	if math.Mod(math.Mod(w.want, 360)+360, 360) != 270 {
		t.Fatalf("turning round did not retarget: %v", w.want)
	}
}

// The figure follows the hand a little behind it and settles on it; held
// still it stops walking after walkStopMs.
func TestWalkerFollowsBehindTheHandAndStops(t *testing.T) {
	now := time.Unix(0, 0)
	w := newWalker(0, 0, 0, 1, now)
	w.move(100, 0, now)
	x, _, moved := w.tick(now)
	if !moved || x <= 0 || x >= 100 {
		t.Fatalf("first frame lands at %v, want between the two", x)
	}
	for i := 0; i < 40; i++ {
		x, _, _ = w.tick(now)
	}
	if x != 100 {
		t.Fatalf("did not settle on the hand: %v", x)
	}
	if !w.moving {
		t.Fatal("not walking right after a step")
	}
	w.tick(now.Add(walkStopMs * time.Millisecond))
	if w.moving {
		t.Fatal("still walking after the hand stood still")
	}
	// scale: a 3px step is a direction at 100% and not at 175%
	w1 := newWalker(0, 0, 0, 1, now)
	w1.move(3, 0, now.Add(time.Millisecond))
	w2 := newWalker(0, 0, 0, 1.75, now)
	w2.move(3, 0, now.Add(time.Millisecond))
	if !w1.hasWant || w2.hasWant {
		t.Fatalf("step threshold not scaled: %v %v", w1.hasWant, w2.hasWant)
	}
}

func TestWalkerEndRewindsWholeCircles(t *testing.T) {
	w := newWalker(0, 0, 0, 1, time.Unix(0, 0))
	w.heading = 370
	_, _, h := w.end()
	if math.Abs(h-10) > 1e-9 {
		t.Fatalf("heading after end: %v", h)
	}
	w.heading = -190
	if _, _, h = w.end(); math.Abs(h-170) > 1e-9 {
		t.Fatalf("heading after end: %v", h)
	}
}

func TestWalkHeadingKeyRoundsToBakedHeadings(t *testing.T) {
	cases := []struct {
		h    float64
		want int
	}{{0, 0}, {14, 0}, {16, 30}, {-16, -30}, {179, 180}, {-179, 180}, {-165, -150}, {370, 0}, {95, 90}}
	for _, c := range cases {
		if got := walkHeadingKey(c.h, 30); got != c.want {
			t.Errorf("walkHeadingKey(%v) = %d, want %d", c.h, got, c.want)
		}
	}
}
