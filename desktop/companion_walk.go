package main

// How the desktop body walks when dragged — Companion.svelte's drag, ported.
//
// In the app window the drag is the component's: it faces the way it is
// going and follows the hand a little behind it, the way a thing on legs
// would, rather than being pinned under the cursor. On the desktop the hand
// is a Win32 mouse (companion_windows.go) and the figure is a window, so the
// same arithmetic has to live here. The constants and their reasons are the
// component's (mascot/Companion.svelte, "where it sits"): the owner's first
// look at the walk was "เร็ว เหวี่ยง" — it whipped round on every wobble of
// the hand — so the direction is the latest real step's, held until a
// clearly different one comes, and the head turns towards it no faster than
// walkTurnDegS, the short way round. The body closes walkFollow of the
// remaining distance to the hand per frame.
//
// Angles are the head's own degrees (mascot.css --t): 0 faces the viewer,
// +90 walks right, -90 left, 180 away up the screen. Pixels are physical;
// the thresholds are logical and scaled by the caller.

import (
	"math"
	"time"
)

const (
	// Degrees per second the head may turn while walking.
	walkTurnDegS = 240
	// How much of the remaining distance to the hand is closed per frame.
	walkFollow = 0.28
	// A step shorter than this (logical px) has no direction worth reading.
	walkStepPx = 2
	// A new direction must differ from the one being turned to by more than
	// this before it replaces it — a hand wobbles, a walker does not.
	walkRetargetDeg = 25
	// No movement for this long while held is standing, not walking: it
	// floats in place, facing the way it was going.
	walkStopMs = 200
)

// walker is one drag in progress: where the hand wants the figure, where the
// figure is on its way there, and which way it faces.
type walker struct {
	// Heading in degrees, and the direction being turned towards once there
	// is one.
	heading float64
	want    float64
	hasWant bool
	// The hand's spot for the figure (already clamped by the caller), and the
	// figure's actual spot easing towards it.
	targetX, targetY float64
	x, y             float64
	at               time.Time
	tickAt           time.Time
	lastStep         time.Time
	moving           bool
	scale            float64
}

// newWalker starts a drag with the figure at (x, y) and the head at heading
// — "start walking from where the head already is" — with scale physical
// pixels per logical one.
func newWalker(x, y, heading, scale float64, now time.Time) *walker {
	if scale <= 0 {
		scale = 1
	}
	return &walker{heading: heading, targetX: x, targetY: y, x: x, y: y, at: now, tickAt: now, scale: scale}
}

// move is the hand asking for the figure at (tx, ty). Everything reads the
// figure's own displacement, not the hand's: pressed against an edge it is
// not walking, so it stands there, and it does not turn to face a hand
// sliding along the wall.
func (w *walker) move(tx, ty float64, now time.Time) {
	mx, my := tx-w.targetX, ty-w.targetY
	w.targetX, w.targetY = tx, ty
	dt := math.Min(0.05, now.Sub(w.at).Seconds())
	w.at = now
	step := math.Hypot(mx, my)
	if step >= 0.5 {
		w.moving = true
		w.lastStep = now
	}
	if step >= walkStepPx*w.scale {
		prev := 0.0
		if w.hasWant {
			prev = w.want
		}
		h := walkTurn(mx, my, prev, 0)
		if !w.hasWant || math.Abs(h-w.want) > walkRetargetDeg {
			w.want = h
			w.hasWant = true
		}
	}
	if w.hasWant {
		goal := nearAngle(w.want, w.heading)
		turn := walkTurnDegS * dt
		w.heading += math.Max(-turn, math.Min(turn, goal-w.heading))
	}
}

// tick is one frame of following: the figure eases towards the hand's spot
// and stops walking once the hand has been still for walkStopMs. Reports the
// figure's spot and whether it moved.
//
// The easing is walkFollow per 60Hz frame, as in the app, but measured in
// time: a frame that comes late closes more of the distance, so the figure
// trails the hand by the same beat whatever the frame rate.
func (w *walker) tick(now time.Time) (x, y float64, moved bool) {
	if w.moving && now.Sub(w.lastStep) >= walkStopMs*time.Millisecond {
		w.moving = false
	}
	dt := now.Sub(w.tickAt).Seconds()
	w.tickAt = now
	if dt <= 0 || dt > 0.25 {
		dt = 1.0 / 60
	}
	follow := 1 - math.Pow(1-walkFollow, dt*60)
	nx := w.x + (w.targetX-w.x)*follow
	ny := w.y + (w.targetY-w.y)*follow
	if math.Hypot(w.targetX-nx, w.targetY-ny) < 0.5 {
		nx, ny = w.targetX, w.targetY
	}
	moved = nx != w.x || ny != w.y
	w.x, w.y = nx, ny
	return nx, ny, moved
}

// end is the release: the figure lands on the hand's spot and whole circles
// are rewound, so the turn home is the short way round.
func (w *walker) end() (x, y, heading float64) {
	w.x, w.y = w.targetX, w.targetY
	w.moving = false
	w.heading = math.Mod(math.Mod(w.heading+180, 360)+360, 360) - 180
	return w.x, w.y, w.heading
}

// walkTurn is the heading for a step (dx, dy), or prev if the step is too
// short to read (presence.ts walkTurn).
func walkTurn(dx, dy, prev, minPx float64) float64 {
	if math.Hypot(dx, dy) < minPx {
		return prev
	}
	return nearAngle(math.Atan2(dx, dy)*180/math.Pi, prev)
}

// nearAngle is the angle equal to h (mod 360) nearest prev — so stepping
// from prev towards it is the short way round. On the seam (180° apart) it
// goes the positive way (presence.ts nearAngle).
func nearAngle(h, prev float64) float64 {
	r := math.Mod(math.Mod(h-prev, 360)+360, 360)
	if r > 180 {
		r -= 360
	}
	return prev + r
}

// walkHeadingKey rounds a heading to the baked headings (bake.ts WALK_STEP),
// in (-180, 180].
func walkHeadingKey(heading float64, step int) int {
	h := int(math.Floor(heading/float64(step)+0.5)) * step
	h = ((h+180)%360+360)%360 - 180
	if h == -180 {
		h = 180
	}
	return h
}
