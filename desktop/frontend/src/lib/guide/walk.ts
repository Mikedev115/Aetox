// Where the figure stands, which way it faces, and how long it takes to get
// there. Pure arithmetic over rectangles — no DOM, no state, no Svelte — so
// the gait can be tuned and argued about on its own, and every rule below has
// a test instead of a screenshot.
//
// This file answers one question: given a thing to point at and a window to
// stand in, where does the figure go. It does NOT decide *what* to point at
// (routes.ts), *how to reach the page it is on* (pages.ts), or what is said
// there (the map's words). Those three were one lump inside Guide.svelte
// until 15 ก.ย. 2026; splitting them is what lets any one of them be changed
// without reading the other two.

/** The figure's box. 72 is the companion's own size on screen. */
export const SIZE = 72

/** The bubble's width — `.say` in Guide.svelte is `width: 350px`. The two
 *  numbers have to agree: this one decides which SIDE the bubble opens on,
 *  that one decides how wide it actually is, and a disagreement puts a card
 *  half off the screen. Changing the width means changing both. */
export const BUBBLE = 350

/** Clear of the target, and clear of the window's edges. */
const GAP = 16
const BUBBLE_GAP = 14
const EDGE = 8
/** Below this the bubble would run off the top, so it hangs under the figure
 *  instead. Roughly the bubble's own tallest ordinary height. */
const BUBBLE_DROP = 260
/** The top bar's own height plus the guide's pill — never stand over it. */
const TOP_SAFE = 48

export type Rect = { left: number; top: number; right: number; bottom: number; width: number; height: number }
export type Viewport = { width: number; height: number }

export type Stance = {
  /** Where the figure's box goes, in window coordinates. */
  x: number
  y: number
  /** True when the bubble opens to the figure's RIGHT (`.flip` in the CSS). */
  flip: boolean
  /** True when the bubble hangs below the figure instead of above. */
  below: boolean
  /** The ring drawn around the target. */
  ring: { l: number; t: number; w: number; h: number }
}

/**
 * Stand beside `target`, with the bubble on the side AWAY from it.
 *
 * The rule that matters: the card must never cover the button it is about.
 * So the side is chosen by where the BUBBLE fits, not by where the figure
 * fits — standing right of the target is only useful if the card can then
 * open further right. When neither side has room for the card (a target in a
 * narrow window), the figure takes whichever side it fits on and the card
 * opens back over the target, because a card half outside the window is worse
 * than a card overlapping what it describes.
 */
export function standBeside(target: Rect, win: Viewport): Stance {
  const rightOf = target.right + GAP
  const leftOf = Math.max(EDGE, target.left - SIZE - GAP)
  const fitsRight = rightOf + SIZE + BUBBLE_GAP + BUBBLE <= win.width
  const fitsLeft = target.left - SIZE - GAP - BUBBLE_GAP - BUBBLE >= 0

  let x: number
  let flip: boolean
  if (fitsRight) {
    x = rightOf
    flip = true
  } else if (fitsLeft) {
    x = leftOf
    flip = false
  } else {
    x = rightOf + SIZE + (GAP + EDGE) < win.width ? rightOf : leftOf
    flip = x - BUBBLE_GAP - BUBBLE < 0
  }

  const y = Math.min(win.height - SIZE - GAP, Math.max(TOP_SAFE, target.top + target.height / 2 - SIZE / 2))
  return {
    x,
    y,
    flip,
    below: y < BUBBLE_DROP,
    ring: { l: Math.max(0, target.left - 4), t: Math.max(0, target.top - 4), w: target.width + 8, h: target.height + 8 },
  }
}

/**
 * Where the figure waits when it has nothing to point at — the lower right,
 * clear of the composer, the corner the companion also favours. Far enough in
 * that the bubble, which opens leftward from here, stays on screen.
 */
export function restingSpot(win: Viewport): { x: number; y: number } {
  return {
    x: Math.max(EDGE, win.width - SIZE - 140),
    y: Math.max(TOP_SAFE, win.height - SIZE - 120),
  }
}

/**
 * Keep a spot the USER chose inside the window.
 *
 * The same edges `restingSpot` respects, applied to a place a hand put the
 * figure rather than a place this file picked. Two reasons it has to be here
 * and not in the drag handler: a window that was resized since the spot was
 * saved would otherwise open the guide off screen, and the top bar is out of
 * bounds for the same reason it is out of bounds for everything else — the
 * figure standing over it covers controls it cannot itself replace.
 */
export function insideWindow(p: { x: number; y: number }, win: Viewport): { x: number; y: number } {
  return {
    x: Math.max(EDGE, Math.min(win.width - SIZE - EDGE, p.x)),
    y: Math.max(TOP_SAFE, Math.min(win.height - SIZE - EDGE, p.y)),
  }
}

/** Which way the head turns for a walk of `dx` — the rig's own walk angles. */
export function walkTurn(dx: number): number {
  return dx < 0 ? -70 : 70
}

/**
 * How long the walk takes. Distance-proportional with a floor, so a short hop
 * still reads as a step and a walk across the window never drags: the eye has
 * to be able to follow it to the target, which is the only reason the figure
 * walks instead of appearing.
 *
 * `reduced` is prefers-reduced-motion: no walk at all, the figure is simply
 * there — motion is the thing that setting is about, and the guide has no
 * business insisting on it.
 */
export function walkMs(dx: number, reduced: boolean): number {
  if (reduced) return 0
  return Math.min(800, 250 + Math.abs(dx) * 0.8)
}
