// Single source of truth for guide geometry, placement, and responsive calculations.
// Used by TypeScript calculations (walk.ts) and exported to CSS custom properties.

export const GEOMETRY = {
  FIGURE_SIZE: 72,
  BUBBLE_DEFAULT_WIDTH: 420,
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
export type DockPosition = 'bottom' | 'top'

export type PlacementResult = {
  x: number
  y: number
  flip: boolean
  below: boolean
  mode: PlacementMode
  dockPosition?: DockPosition
  ring: { l: number; t: number; w: number; h: number }
  dock?: {
    left: number
    right: number
    top?: number
    bottom?: number
    width: number
    height?: number
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
 * 5. In docked mode, if the target is in the lower portion of the screen, dock to top;
 *    otherwise dock to bottom, ensuring the target remains fully visible.
 */
export function calculatePlacement(
  target: Rect,
  bubbleSize: Size,
  win: Viewport,
  figureSize: number = GEOMETRY.FIGURE_SIZE,
): PlacementResult {
  const bubbleW = bubbleSize.width || GEOMETRY.BUBBLE_DEFAULT_WIDTH
  const bubbleH = bubbleSize.height || 200
  const figure = Math.max(1, figureSize || GEOMETRY.FIGURE_SIZE)

  const ring = {
    l: Math.max(0, target.left - 4),
    t: Math.max(0, target.top - 4),
    w: target.width + 8,
    h: target.height + 8,
  }

  const isNarrowViewport = win.width < GEOMETRY.DOCK_BREAKPOINT_WIDTH

  const rightOf = target.right + GEOMETRY.GAP_TARGET
  const leftOf = Math.max(GEOMETRY.EDGE_PADDING, target.left - figure - GEOMETRY.GAP_TARGET)
  const fitsRight = rightOf + figure + GEOMETRY.GAP_BUBBLE + bubbleW <= win.width
  const fitsLeft = target.left - figure - GEOMETRY.GAP_TARGET - GEOMETRY.GAP_BUBBLE - bubbleW >= 0

  const figureY = Math.min(
    win.height - figure - GEOMETRY.GAP_TARGET,
    Math.max(GEOMETRY.TOP_SAFE, target.top + target.height / 2 - figure / 2),
  )
  // These offsets mirror Guide.svelte: ordinary bubbles end 24px above the
  // figure's bottom, while a below bubble begins 4px below its top.
  const fitsAbove = figureY + figure - 24 - bubbleH >= GEOMETRY.TOP_SAFE
  const fitsBelow = figureY + 4 + bubbleH <= win.height - GEOMETRY.EDGE_PADDING

  // 1. Docked fallback for narrow screens, horizontal squeeze, or a bubble
  // that fits neither above nor below the target. Width alone used to choose
  // side mode and let a tall guide card fall off the bottom of the window.
  if (isNarrowViewport || (!fitsRight && !fitsLeft) || (!fitsAbove && !fitsBelow)) {
    const dockHeight = Math.min(bubbleH + 24, win.height * 0.45)
    // Avoid covering target: if target is in the lower half/boundary, dock at top; else dock at bottom
    const dockAtBottom = target.bottom < win.height - dockHeight - 20
    const dockPosition: DockPosition = dockAtBottom ? 'bottom' : 'top'

    let figX = rightOf
    if (figX + figure > win.width - GEOMETRY.EDGE_PADDING) {
      figX = Math.max(GEOMETRY.EDGE_PADDING, target.left - figure - GEOMETRY.GAP_TARGET)
    }

    return {
      x: figX,
      y: figureY,
      flip: false,
      below: true,
      mode: 'docked',
      dockPosition,
      ring,
      dock: {
        left: GEOMETRY.EDGE_PADDING,
        right: GEOMETRY.EDGE_PADDING,
        top: dockAtBottom ? undefined : GEOMETRY.TOP_SAFE + 8,
        bottom: dockAtBottom ? GEOMETRY.EDGE_PADDING : undefined,
        width: win.width - GEOMETRY.EDGE_PADDING * 2,
        height: dockHeight,
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
    x = rightOf + figure + (GEOMETRY.GAP_TARGET + GEOMETRY.EDGE_PADDING) < win.width ? rightOf : leftOf
    flip = x - GEOMETRY.GAP_BUBBLE - bubbleW < 0
  }

  const below = !fitsAbove && fitsBelow

  return {
    x,
    y: figureY,
    flip,
    below,
    mode: 'side',
    ring,
  }
}
