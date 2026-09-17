// Where the figure stands, which way it faces, and how long it takes to get
// there. Pure arithmetic over rectangles — no DOM, no state, no Svelte — so
// the gait can be tuned and argued about on its own, and every rule below has
// a test instead of a screenshot.
//
// Powered by the unified Geometry Policy (./geometry.ts) as the single source of truth.

import {
  GEOMETRY,
  calculatePlacement,
  type Rect,
  type Viewport,
  type Size,
  type PlacementMode,
  type DockPosition,
  type PlacementResult,
} from './geometry'
export { GEOMETRY, calculatePlacement }
export type { Rect, Viewport, Size, PlacementMode, DockPosition, PlacementResult }

/** The figure's box. 72 is the companion's own size on screen. */
export const SIZE = GEOMETRY.FIGURE_SIZE

/** The bubble's width — `.say` in Guide.svelte is `width: 350px`. */
export const BUBBLE = GEOMETRY.BUBBLE_DEFAULT_WIDTH

/** Clear of the target, and clear of the window's edges. */
const GAP = GEOMETRY.GAP_TARGET
const BUBBLE_GAP = GEOMETRY.GAP_BUBBLE
const EDGE = GEOMETRY.EDGE_PADDING
/** Below this the bubble would run off the top, so it hangs under the figure
 *  instead. Roughly the bubble's own tallest ordinary height. */
const BUBBLE_DROP = GEOMETRY.BUBBLE_DROP
/** The top bar's own height plus the guide's pill — never stand over it. */
const TOP_SAFE = GEOMETRY.TOP_SAFE


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
 * fits.
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
 * Enhanced placement calculation supporting real-measured bubble size and docked callout fallback.
 */
export function standBesideReal(target: Rect, bubbleSize: Size, win: Viewport, figureSize: number = SIZE): PlacementResult {
  return calculatePlacement(target, bubbleSize, win, figureSize)
}

/**
 * Where the figure waits when it has nothing to point at — the lower right,
 * clear of the composer, the corner the companion also favours. Far enough in
 * that the bubble, which opens leftward from here, stays on screen.
 */
export function restingSpot(win: Viewport, figureSize: number = SIZE): { x: number; y: number } {
  return {
    x: Math.max(EDGE, win.width - figureSize - 140),
    y: Math.max(TOP_SAFE, win.height - figureSize - 120),
  }
}

/**
 * Keep a spot the USER chose inside the window.
 */
export function insideWindow(p: { x: number; y: number }, win: Viewport, figureSize: number = SIZE): { x: number; y: number } {
  return {
    x: Math.max(EDGE, Math.min(win.width - figureSize - EDGE, p.x)),
    y: Math.max(TOP_SAFE, Math.min(win.height - figureSize - EDGE, p.y)),
  }
}

/** Which way the head turns for a walk of `dx` — the rig's own walk angles. */
export function walkTurn(dx: number): number {
  return dx < 0 ? -70 : 70
}

/**
 * How long the walk takes. Distance-proportional with a floor.
 */
export function walkMs(dx: number, reduced: boolean): number {
  if (reduced) return 0
  return Math.min(800, 250 + Math.abs(dx) * 0.8)
}
