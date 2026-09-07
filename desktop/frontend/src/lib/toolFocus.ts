/** Which row of a live list is the one being read, and how far behind it every
 *  other one is.
 *
 *  A delegate's list is the longest-running thing in the product and it was
 *  drawn flat: twenty rows at one size and one ink, with the newest at the
 *  bottom and nothing saying so. The reader had to find the live row before
 *  they could read it, on a list that moves — which is the work the card is
 *  supposed to be doing for them. Owner, 7 ก.ย.: *"แบบพอใช้ tool นี้อยู่ก็ซูม
 *  แล้วขยายออกตอนเปลี่ยนไปงานต่อไป"*.
 *
 *  So the list has a focus: the row at the floor is read at body size, and the
 *  ones above it step down and dim as they recede. In a vertical list this
 *  shape is a wheel picker; the principle underneath it is focus + context.
 *
 *  NOTHING IS SCALED, and that is a decision this codebase took before this
 *  file existed: a transform-scaled row resamples its own text at fractional
 *  sizes on the way and the eye reads that as blur rather than as movement
 *  (style.css at .bgw-card.bgw-in, and fold.ts records the same finding for
 *  unroll). The sizes here are DISCRETE — one step, landed on — so every glyph
 *  is crisp in every frame, and what actually travels between them is opacity
 *  and colour, neither of which costs a layout.
 *
 *  An action rather than component state, for the reason toolGlide and
 *  toolWindow are actions: what it needs is a count of rows it does not own,
 *  taken after the frame that changed them. Four steps is the whole ladder —
 *  past the third the window's own mask has taken over, and a fifth would be a
 *  distinction nobody can see.
 */
const DEPTHS = ['d0', 'd1', 'd2', 'd3']

export function toolFocus(node: HTMLElement, on = true) {
  let live = on
  let frame = 0

  const paint = () => {
    frame = 0
    // Both, because a delegate's narration sits in the same column as its
    // calls (§59) and recedes with them. A search card is neither and is
    // skipped, which is correct: it belongs to the row above it and takes that
    // row's depth by sitting under it.
    const rows = node.querySelectorAll<HTMLElement>('.tool-step, .tool-note')
    for (let i = 0; i < rows.length; i++) {
      const el = rows[i]
      const want = live ? DEPTHS[Math.min(rows.length - 1 - i, DEPTHS.length - 1)] : ''
      // Written down on the element rather than read back off classList: a row
      // whose depth has not changed must not have its class removed and re-added,
      // and this is the cheapest way to know that without touching the DOM.
      if (el.dataset.depth === want) continue
      if (el.dataset.depth) el.classList.remove(el.dataset.depth)
      if (want) el.classList.add(want)
      el.dataset.depth = want
    }
  }
  const schedule = () => { if (!frame) frame = requestAnimationFrame(paint) }

  // Rows arrive and rows change state; both move the floor.
  const rows = new MutationObserver(schedule)
  rows.observe(node, { childList: true, subtree: true })
  paint()

  return {
    update(next = true) {
      if (next === live) return
      live = next
      schedule()
    },
    destroy() {
      rows.disconnect()
      if (frame) cancelAnimationFrame(frame)
    },
  }
}
