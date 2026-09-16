// Single source of truth for guide geometry, placement, and responsive calculations.
// Used by TypeScript calculations (walk.ts) and exported to CSS custom properties.

export const GEOMETRY = {
  FIGURE_SIZE: 72,
  BUBBLE_DEFAULT_WIDTH: 350,
  BUBBLE_MIN_WIDTH: 260,
  GAP_TARGET: 16,
  GAP_BUBBLE: 14,
  EDGE_PADDING: 8,
  BUBBLE_DROP: 260,
  TOP_SAFE: 48,
  DOCK_BREAKPOINT_WIDTH: 720,
} as const

export type Rect = {
  left: number
  top: number
  right: number
  bottom: number
  width: number
  height: number
}

export type Viewport = {
  width: number
  height: number
}

export type Size = {
  width: number
  height: number
}

export type PlacementMode = 'side' | 'docked'

export type PlacementResult = {
  x: number
  y: number
  flip: boolean
  below: boolean
  mode: PlacementMode
  ring: { l: number; t: number; w: number; h: number }
  dock?: {
    left: number
    right: number
    bottom: number
    width: number
  }
}

/**
 * Calculate the optimal position for the figure and speech bubble given the
 * target element's rectangle, the bubble's measured size, and the viewport.
 *
 * Invariants:
 * 1. The bubble must never overlap the target it is explaining (unless in docked fallback).
 * 2. The bubble must never overflow outside the viewport.
 * 3. The figure must never stand over the top bar (y >= TOP_SAFE) or outside the viewport.
 * 4. In narrow viewports (< 720px) or when neither side fits the bubble, fallback to 'docked' mode.
 */
export function calculatePlacement(
  target: Rect,
  bubbleSize: Size,
  win: Viewport,
): PlacementResult {
  const bubbleW = bubbleSize.width || GEOMETRY.BUBBLE_DEFAULT_WIDTH
  const bubbleH = bubbleSize.height || 200

  const ring = {
    l: Math.max(0, target.left - 4),
    t: Math.max(0, target.top - 4),
    w: target.width + 8,
    h: target.height + 8,
  }

  const isNarrowViewport = win.width < GEOMETRY.DOCK_BREAKPOINT_WIDTH

  const rightOf = target.right + GEOMETRY.GAP_TARGET
  const leftOf = Math.max(GEOMETRY.EDGE_PADDING, target.left - GEOMETRY.FIGURE_SIZE - GEOMETRY.GAP_TARGET)
  const fitsRight = rightOf + GEOMETRY.FIGURE_SIZE + GEOMETRY.GAP_BUBBLE + bubbleW <= win.width
  const fitsLeft = target.left - GEOMETRY.FIGURE_SIZE - GEOMETRY.GAP_TARGET - GEOMETRY.GAP_BUBBLE - bubbleW >= 0

  // 1. Docked fallback for narrow screens or when both sides cannot fit the bubble
  if (isNarrowViewport || (!fitsRight && !fitsLeft)) {
    const dockHeight = Math.min(bubbleH + 24, win.height * 0.45)
    const dockAtBottom = target.bottom < win.height - dockHeight - 20

    let figX = rightOf
    if (figX + GEOMETRY.FIGURE_SIZE > win.width - GEOMETRY.EDGE_PADDING) {
      figX = Math.max(GEOMETRY.EDGE_PADDING, target.left - GEOMETRY.FIGURE_SIZE - GEOMETRY.GAP_TARGET)
    }

    const figY = Math.min(
      win.height - GEOMETRY.FIGURE_SIZE - GEOMETRY.GAP_TARGET,
      Math.max(GEOMETRY.TOP_SAFE, target.top + target.height / 2 - GEOMETRY.FIGURE_SIZE / 2),
    )

    return {
      x: figX,
      y: figY,
      flip: false,
      below: true,
      mode: 'docked',
      ring,
      dock: {
        left: GEOMETRY.EDGE_PADDING,
        right: GEOMETRY.EDGE_PADDING,
        bottom: dockAtBottom ? GEOMETRY.EDGE_PADDING : win.height - dockHeight,
        width: win.width - GEOMETRY.EDGE_PADDING * 2,
      },
    }
  }

  // 2. Side placement
  let x: number
  let flip: boolean
  if (fitsRight) {
    x = rightOf
    flip = true
  } else if (fitsLeft) {
    x = leftOf
    flip = false
  } else {
    x = rightOf + GEOMETRY.FIGURE_SIZE + (GEOMETRY.GAP_TARGET + GEOMETRY.EDGE_PADDING) < win.width ? rightOf : leftOf
    flip = x - GEOMETRY.GAP_BUBBLE - bubbleW < 0
  }

  const y = Math.min(
    win.height - GEOMETRY.FIGURE_SIZE - GEOMETRY.GAP_TARGET,
    Math.max(GEOMETRY.TOP_SAFE, target.top + target.height / 2 - GEOMETRY.FIGURE_SIZE / 2),
  )

  const below = y < (bubbleH > 240 ? bubbleH + 20 : GEOMETRY.BUBBLE_DROP)

  return {
    x,
    y,
    flip,
    below,
    mode: 'side',
    ring,
  }
}
