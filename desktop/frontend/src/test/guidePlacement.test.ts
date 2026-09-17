import { describe, it, expect } from 'vitest'
import { standBesideReal, walkMs, GEOMETRY, type Rect, type Viewport, type Size } from '../lib/guide/walk'

const rect = (left: number, top: number, w = 120, h = 36): Rect => ({
  left,
  top,
  right: left + w,
  bottom: top + h,
  width: w,
  height: h,
})

describe('Real-measurement Placement & Responsive Viewport Matrix', () => {
  const largeWin: Viewport = { width: 1920, height: 1080 }
  const standardWin: Viewport = { width: 1280, height: 800 }
  const tabletWin: Viewport = { width: 800, height: 600 }
  const mobileWin: Viewport = { width: 480, height: 800 }
  const narrowWin: Viewport = { width: 375, height: 667 }

  it('standard desktop: places figure and bubble side-by-side without target overlap', () => {
    const bubbleSize: Size = { width: 350, height: 180 }
    const target = rect(20, 300)
    const res = standBesideReal(target, bubbleSize, standardWin)

    expect(res.mode).toBe('side')
    expect(res.flip).toBe(true) // opens to the right
    expect(res.x).toBeGreaterThan(target.right)

    const bubbleLeft = res.x + GEOMETRY.FIGURE_SIZE + GEOMETRY.GAP_BUBBLE
    const bubbleRight = bubbleLeft + bubbleSize.width
    expect(bubbleLeft).toBeGreaterThan(target.right)
    expect(bubbleRight).toBeLessThanOrEqual(standardWin.width)
  })

  it('narrow mobile / split screen (< 720px): falls back to docked mode', () => {
    const bubbleSize: Size = { width: 320, height: 200 }
    const target = rect(100, 200)
    const res = standBesideReal(target, bubbleSize, mobileWin)

    expect(res.mode).toBe('docked')
    expect(res.dock).toBeDefined()
    expect(res.dock!.width).toBeLessThanOrEqual(mobileWin.width)
    expect(res.ring).toBeDefined()
  })

  it('extreme narrow viewport (375px): docks cleanly without overflowing screen edges', () => {
    const bubbleSize: Size = { width: 300, height: 180 }
    const target = rect(50, 150)
    const res = standBesideReal(target, bubbleSize, narrowWin)

    expect(res.mode).toBe('docked')
    expect(res.dock!.left).toBeGreaterThanOrEqual(0)
    expect(res.dock!.width).toBe(narrowWin.width - GEOMETRY.EDGE_PADDING * 2)
  })

  it('centered target in constrained viewport (no room left or right): switches to docked fallback', () => {
    // Window width is 750 (just above 720px breakpoint), but target is in center (left=280, w=190)
    // Left space = 280 - 16 - 72 - 14 = 178 < 350 (does not fit left)
    // Right space = 750 - (470 + 16 + 72 + 14) = 178 < 350 (does not fit right)
    const constrainedWin: Viewport = { width: 750, height: 800 }
    const target = rect(280, 300, 190, 40)
    const bubbleSize: Size = { width: 350, height: 200 }

    const res = standBesideReal(target, bubbleSize, constrainedWin)
    expect(res.mode).toBe('docked')
  })

  it('long text with tall bubble (> 300px): maintains safe distance from top bar', () => {
    const tallBubble: Size = { width: 350, height: 380 }
    const target = rect(100, 100)
    const res = standBesideReal(target, tallBubble, largeWin)

    expect(res.y).toBeGreaterThanOrEqual(GEOMETRY.TOP_SAFE)
    expect(res.below).toBe(true) // hangs below to avoid top bar clipping
  })

  it('docks a tall bubble when a middle target leaves no safe space above or below', () => {
    const viewport: Viewport = { width: 1280, height: 720 }
    const tallBubble: Size = { width: 420, height: 470 }
    const middleTarget = rect(28, 350, 220, 40)

    const res = standBesideReal(middleTarget, tallBubble, viewport)

    expect(res.mode).toBe('docked')
    expect(res.dockPosition).toBe('top')
    expect(res.dock?.top).toBeDefined()
  })

  it('docked adaptive placement: docks top when target is at bottom to avoid occlusion', () => {
    const bubbleSize: Size = { width: 320, height: 180 }
    // Target is composer at bottom of screen
    const bottomTarget = rect(20, 720, 300, 48)
    const res = standBesideReal(bottomTarget, bubbleSize, mobileWin) // 480x800

    expect(res.mode).toBe('docked')
    expect(res.dockPosition).toBe('top')
    expect(res.dock?.top).toBeDefined()
    expect(res.dock?.bottom).toBeUndefined()
    // Dock is at top, so it does not overlap the bottom target!
    expect(res.dock!.top! + res.dock!.height!).toBeLessThan(bottomTarget.top)
  })

  it('docked adaptive placement: docks bottom when target is in upper/middle screen', () => {
    const bubbleSize: Size = { width: 320, height: 180 }
    // Target is topbar or header
    const topTarget = rect(20, 60, 200, 40)
    const res = standBesideReal(topTarget, bubbleSize, mobileWin)

    expect(res.mode).toBe('docked')
    expect(res.dockPosition).toBe('bottom')
    expect(res.dock?.bottom).toBeDefined()
    expect(res.dock?.top).toBeUndefined()
  })

  it('full viewport matrix: figure and dock remain within bounds across all screen dimensions', () => {
    const viewports: Viewport[] = [
      { width: 320, height: 568 }, // iPhone SE
      { width: 375, height: 667 }, // iPhone 8
      { width: 412, height: 915 }, // Pixel / Galaxy
      { width: 768, height: 1024 }, // iPad portrait
      { width: 1024, height: 768 }, // iPad landscape
      { width: 1280, height: 800 }, // Desktop standard
      { width: 1920, height: 1080 }, // Desktop wide
    ]
    const bubbleSize: Size = { width: 300, height: 160 }

    for (const win of viewports) {
      // Test target at top-left
      const tTop = rect(20, 60, 80, 32)
      const resTop = standBesideReal(tTop, bubbleSize, win)
      expect(resTop.x).toBeGreaterThanOrEqual(0)
      expect(resTop.x + GEOMETRY.FIGURE_SIZE).toBeLessThanOrEqual(win.width)
      expect(resTop.y).toBeGreaterThanOrEqual(GEOMETRY.TOP_SAFE)
      expect(resTop.y + GEOMETRY.FIGURE_SIZE).toBeLessThanOrEqual(win.height)

      // Test target at bottom-center
      const tBottom = rect(win.width / 2 - 50, win.height - 60, 100, 36)
      const resBottom = standBesideReal(tBottom, bubbleSize, win)
      expect(resBottom.x).toBeGreaterThanOrEqual(0)
      expect(resBottom.x + GEOMETRY.FIGURE_SIZE).toBeLessThanOrEqual(win.width)
      expect(resBottom.y).toBeGreaterThanOrEqual(GEOMETRY.TOP_SAFE)
      expect(resBottom.y + GEOMETRY.FIGURE_SIZE).toBeLessThanOrEqual(win.height)
    }
  })

  it('accessibility: prefers-reduced-motion returns 0ms walk time', () => {
    expect(walkMs(500, true)).toBe(0)
    expect(walkMs(500, false)).toBeGreaterThan(0)
  })
})
