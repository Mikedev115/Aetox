// The native window holds still while a panel handle is dragged, and follows
// once when the handle is let go (BrowserPane reflow, §283). The owner's ask on
// 14 ก.ย.: "รอผู้ใช้ลากเสร็จแล้ว เบาเซอร์ค่อยตามไป".
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render } from '@testing-library/svelte'
import BrowserPane from '../lib/workbench/BrowserPane.svelte'
import { panelDrag } from '../lib/panelDrag.svelte'
import { BrowserSetBounds, BrowserSetVisible } from './mocks/wailsApp'

const FRAME = { top: 3, right: 3, bottom: 3, left: 12 }
let pane = { x: 100, y: 50, width: 400, height: 900 }
// What the pane sends for the current box: framed, at devicePixelRatio 1.
const boxOf = (p: typeof pane) => [
  p.x + FRAME.left,
  p.y + FRAME.top,
  p.width - FRAME.left - FRAME.right,
  p.height - FRAME.top - FRAME.bottom,
]

beforeEach(() => {
  vi.clearAllMocks()
  window.devicePixelRatio = 1
  panelDrag.active = false
  pane = { x: 100, y: 50, width: 400, height: 900 }
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect() {} })
  HTMLElement.prototype.getBoundingClientRect = () => ({
    ...pane, top: pane.y, left: pane.x, bottom: pane.y + pane.height, right: pane.x + pane.width, toJSON: () => '',
  }) as DOMRect
})

async function mount() {
  const props = { tab: { id: 'web-1', kind: 'browser' as const, name: 'x', url: 'https://a.test' }, active: true, menuOpen: false, dragging: false }
  const r = render(BrowserPane, props)
  await vi.waitFor(() => expect(BrowserSetVisible).toHaveBeenCalledWith('web-1', true))
  await vi.waitFor(() => expect(BrowserSetBounds).toHaveBeenCalled())
  BrowserSetBounds.mockClear()
  BrowserSetVisible.mockClear()
  return { ...r, props }
}

// A pane change while dragging: the component's own ResizeObserver never fires
// under jsdom, so the drag's frames are played by resizing the window, which
// the pane also listens to — and which it also reads as a drag.
function frame(next: Partial<typeof pane>) {
  Object.assign(pane, next)
  window.dispatchEvent(new Event('resize'))
}

describe('the native window during a drag', () => {
  it('holds still while a panel handle is dragged and follows once when it is let go', async () => {
    await mount()
    panelDrag.active = true
    frame({ width: 500 })
    frame({ width: 600 })
    frame({ width: 700 })
    await new Promise((r) => setTimeout(r, 200)) // past the window-resize quiet period
    expect(BrowserSetBounds).not.toHaveBeenCalled()
    // Growing: the page stays where it was, in view.
    expect(BrowserSetVisible).not.toHaveBeenCalledWith('web-1', false)

    panelDrag.active = false
    await vi.waitFor(() => expect(BrowserSetBounds).toHaveBeenCalledTimes(1))
    expect(BrowserSetBounds).toHaveBeenLastCalledWith('web-1', ...boxOf(pane))
  })

  it('hides a window the shrinking pane can no longer hold, and shows it again in place at the end', async () => {
    await mount()
    panelDrag.active = true
    frame({ width: 300 })
    await vi.waitFor(() => expect(BrowserSetVisible).toHaveBeenLastCalledWith('web-1', false))
    expect(BrowserSetBounds).not.toHaveBeenCalled()
    // Wobbling back out does not blink it on: hidden stays hidden until the end.
    frame({ width: 450 })
    await new Promise((r) => setTimeout(r, 200))
    expect(BrowserSetVisible).toHaveBeenLastCalledWith('web-1', false)

    panelDrag.active = false
    await vi.waitFor(() => expect(BrowserSetVisible).toHaveBeenLastCalledWith('web-1', true))
    expect(BrowserSetBounds).toHaveBeenCalledTimes(1)
    expect(BrowserSetBounds).toHaveBeenLastCalledWith('web-1', ...boxOf(pane))
  })

  it("reads the window's own resize as a drag and follows once it goes quiet", async () => {
    await mount()
    frame({ width: 500 })
    frame({ width: 550 })
    expect(BrowserSetBounds).not.toHaveBeenCalled()
    await vi.waitFor(() => expect(BrowserSetBounds).toHaveBeenCalledTimes(1), { timeout: 1000 })
    expect(BrowserSetBounds).toHaveBeenLastCalledWith('web-1', ...boxOf(pane))
  })
})
