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
 *  taken after the frame that changed them.
 *
 *  A DATA ATTRIBUTE AND NOT A CLASS, which is not a style choice — it is the
 *  only thing that works. A row's class is a template expression carrying its
 *  own state (`class="tool-step f-{fam} h-{slot} {s.state}"`), so the frame a
 *  call comes back in, Svelte rewrites that attribute whole and every class an
 *  action put there is gone with it. The focus block would vanish the instant
 *  the row it was on finished, which is the exact moment it is supposed to be
 *  handing over rather than disappearing. `data-depth` is nobody else's, so it
 *  survives, and CSS reads it with an attribute selector at the same weight.
 */
// The deepest step drawn. Past this the window's own mask has taken over and a
// further one would be a distinction nobody can see.
const FLOOR = 3

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
    // THE FOCUS IS A BAND, NOT A ROW, because a turn is not one call at a time.
    // The agent groups its read-only calls and fires them together
    // (internal/cognitive/agent.go, "running %d read-only tool calls
    // together"), so five searches land in one frame and five rows are out at
    // once. A ladder that lit only the newest of them dimmed four rows every
    // bit as live as the one it lit, and a whole group arriving together is
    // what reads as the list going off all at once (owner, 7 ก.ย.:
    // *"จู่ๆก็ตู้มมาทีเดียว ยาวรวดเดียวอ่ะ"*).
    //
    // So everything still out is depth 0 and the ladder counts up from the
    // FIRST of them. With nothing out — the gap between one group and the next,
    // which is most of a turn — the floor of the list holds the focus, so the
    // band never blinks off between calls.
    let top = rows.length - 1
    for (let i = 0; i < rows.length; i++) {
      if (rows[i].classList.contains('run')) { top = i; break }
    }
    for (let i = 0; i < rows.length; i++) {
      const el = rows[i]
      const want = live ? String(i >= top ? 0 : Math.min(top - i, FLOOR)) : ''
      if (el.dataset.depth === want) continue
      if (want) el.dataset.depth = want
      else delete el.dataset.depth
    }
  }
  const schedule = () => { if (!frame) frame = requestAnimationFrame(paint) }

  // Rows arrive, and rows stop running; both move the band. The second is an
  // ATTRIBUTE change — a row carries its own state in its class — so watching
  // childList alone left the band sitting on a group that had already come
  // back. Filtered to `class` so this cannot see its own writes: what it writes
  // is data-depth, and a filter that let that through would be a loop.
  const rows = new MutationObserver(schedule)
  rows.observe(node, { childList: true, subtree: true, attributes: true, attributeFilter: ['class'] })
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
