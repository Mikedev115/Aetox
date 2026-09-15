// The floor of the transcript, tested without six thousand lines of chat
// around it.
//
// The rule this file exists for: a height that moves with NO store change
// behind it still moves the floor. A picture finishing its load, a code block
// laying out, a stretch of tool rows folding away over a third of a second —
// the component hears about none of those, which is why following store
// changes alone left the view hanging above the bottom, and why the fold
// "ไม่สมูท" (owner, 15 ก.ย. 2026).
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { BottomStick } from '../lib/chatBottom.svelte'

/** A scroller with the geometry jsdom has no layout to give it. */
function scroller(scrollHeight: number, clientHeight = 400): HTMLDivElement {
  const el = document.createElement('div')
  let height = scrollHeight
  Object.defineProperty(el, 'scrollHeight', { configurable: true, get: () => height })
  Object.defineProperty(el, 'clientHeight', { configurable: true, get: () => clientHeight })
  ;(el as any).grow = (to: number) => {
    height = to
    fireResizes()
  }
  return el
}

/** ResizeObserver, under this test's control — the global stub in setup.ts is
 *  a no-op, which would leave the whole point of the module untested. */
let observers: (() => void)[] = []
const fireResizes = () => observers.forEach((cb) => cb())

beforeEach(() => {
  observers = []
  vi.stubGlobal(
    'ResizeObserver',
    class {
      cb: () => void
      constructor(cb: () => void) {
        this.cb = cb
        observers.push(cb)
      }
      observe() {}
      disconnect() {
        observers = observers.filter((o) => o !== this.cb)
      }
    },
  )
})

describe('the transcript floor', () => {
  it('follows a height that moved on its own — no store change, no paced frame', () => {
    const el = scroller(1000)
    const stick = new BottomStick()
    stick.attach(el, el)
    stick.paint()
    expect(el.scrollTop).toBe(1000)

    // A picture loaded. Nothing told the component; the observer did.
    ;(el as any).grow(1600)
    expect(el.scrollTop).toBe(1600)
  })

  it('stays where the reader put it when they have let go', () => {
    const el = scroller(1000)
    const stick = new BottomStick()
    stick.attach(el, el)
    el.scrollTop = 600
    stick.onScroll()
    el.scrollTop = 200 // reading something above
    stick.onScroll()
    expect(stick.following).toBe(false)

    ;(el as any).grow(1600)
    expect(el.scrollTop).toBe(200)
  })

  it('a fold shrinking the transcript keeps the floor, and is not read as letting go', () => {
    const el = scroller(1600)
    const stick = new BottomStick()
    stick.attach(el, el)
    stick.paint()

    // The rows fold away: the content shrinks under a scroller that was at the
    // bottom of it. The browser clamps scrollTop, which looks exactly like an
    // upward scroll and must not be taken for one.
    ;(el as any).grow(1200)
    el.scrollTop = 1200
    stick.onScroll()
    expect(stick.following).toBe(true)
    expect(el.scrollTop).toBe(1200)
  })

  it('never moves the scroller after it is unwired', () => {
    const el = scroller(1000)
    const stick = new BottomStick()
    const off = stick.attach(el, el)
    stick.paint()
    off()

    el.scrollTop = 0
    ;(el as any).grow(2000)
    stick.paint()
    expect(el.scrollTop).toBe(0)
  })

  it('follow() is the explicit ask: back to the newest line and following again', () => {
    const el = scroller(1000)
    const stick = new BottomStick()
    stick.attach(el, el)
    el.scrollTop = 600
    stick.onScroll()
    el.scrollTop = 100
    stick.onScroll()
    expect(stick.following).toBe(false)

    stick.follow()
    expect(stick.following).toBe(true)
    expect(el.scrollTop).toBe(1000)
  })
})
