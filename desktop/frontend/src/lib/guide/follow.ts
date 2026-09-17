// Keeping the ring on the thing it is around.
//
// The figure measures its target once, when it arrives. That is enough for
// exactly as long as nothing moves — and a page that has just opened is the
// place where things move most: a card finishes loading, a font swaps, an
// image arrives above the button, a list grows. A second later the ring is a
// rectangle around empty space and the figure is pointing at nothing, on a
// page it genuinely has arrived at (owner, 15 ก.ย. 2026, with a screenshot of
// exactly that: *"ทำไมเวลากดเปลี่ยนหน้ามันยังไม่รู้ตัวครับ"*).
//
// The same lesson as the transcript's floor, one element smaller: WATCH, do not
// sample. `placeWatch` notices the page changing; this notices the BUTTON
// moving, which is a different event and a more common one — most of the time
// the page is right and the box under the ring is not where it was.
//
import { rectOnScreen } from './where'

// It also answers the harder half. A sign only changes when the page does, so
// a button that disappears WITHIN a page — a menu closing, a card collapsing,
// a row filtered away — moves no sign at all. Here that is simply a box with
// no size, and the caller is told the target is gone.

/** A box that has not moved by this much has not moved. Sub-pixel layout is
 *  routine and re-placing the figure on it would be a permanent flinch. */
const MOVED_PX = 1

/** The shape `walk.standBeside` reads, so a watched box can be handed
 *  straight to it without a second measurement of the same element. */
export type Box = { left: number; top: number; right: number; bottom: number; width: number; height: number }

/**
 * Watch `el`'s box for as long as the guide stands beside it.
 *
 * `onMove` is called with the new box when it moves or resizes, and `onGone`
 * when it stops having a box at all — which the caller must treat as the
 * target leaving, never as a box at the origin.
 *
 * Every source of movement it can hear: the element resizing, the page
 * reflowing under it (body), anything scrolling anywhere (capture, so a
 * scroller inside the page counts), and the window resizing. Coalesced onto one
 * animation frame, because a burst of all four is one movement.
 */
export function watchBox(
  el: HTMLElement,
  onMove: (box: Box) => void,
  onGone: () => void,
): () => void {
  if (typeof document === 'undefined') return () => {}

  const read = (): Box => {
    const r = el.getBoundingClientRect()
    return { left: r.left, top: r.top, right: r.right, bottom: r.bottom, width: r.width, height: r.height }
  }
  const same = (a: Box, b: Box) =>
    Math.abs(a.left - b.left) <= MOVED_PX &&
    Math.abs(a.top - b.top) <= MOVED_PX &&
    Math.abs(a.width - b.width) <= MOVED_PX &&
    Math.abs(a.height - b.height) <= MOVED_PX

  let last = read()
  let frame = 0
  let stopped = false

  const check = () => {
    frame = 0
    if (stopped) return
    const now = read()
    // No size at all: it is not somewhere else, it is not there. Told apart
    // from a move because the two want opposite answers — follow it, or let go
    // of it.
    if (!rectOnScreen(now) || !el.isConnected) {
      stopped = true
      onGone()
      return
    }
    if (same(now, last)) return
    last = now
    onMove(now)
  }

  const nudge = () => {
    if (!frame && !stopped) frame = requestAnimationFrame(check)
  }

  const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(nudge) : null
  ro?.observe(el)
  if (document.body) ro?.observe(document.body)
  // Capture, because the thing that scrolled is usually a panel inside the
  // page and a scroll event does not bubble to the window from one.
  document.addEventListener('scroll', nudge, { capture: true, passive: true })
  window.addEventListener('resize', nudge)

  return () => {
    stopped = true
    if (frame) cancelAnimationFrame(frame)
    ro?.disconnect()
    document.removeEventListener('scroll', nudge, true)
    window.removeEventListener('resize', nudge)
  }
}
