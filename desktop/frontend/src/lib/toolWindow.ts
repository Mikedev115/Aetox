/** How a window behaves. `follow` rides the bottom as rows arrive, which is
 *  what a list being written to wants and what a list that is already finished
 *  does not: a search's eight results all existed in the same millisecond, and
 *  scrolling one to the bottom would hide the best answer to see the worst. */
export type WindowOpts = { on?: boolean; follow?: boolean }

/** The window a live tool list runs inside.
 *
 * A turn used to hide every call the moment it finished, so the only row on
 * screen was the one running this second: a skill that answered in half a
 * second appeared and was gone before it could be read, and the count above it
 * was the only evidence left (owner, 7 ก.ย.: "ตอนสกิลอ่ะ ตอนใช้งานมันจะพับเก็บ
 * ใช่ไหม ทำให้มีดีเลย์ก่อนพับนิดนึงได้ไหม").
 *
 * Nothing folds mid-turn now — every call the phase has made stays in the list
 * — and this is what stops that from becoming a wall: the list is capped and
 * scrolls inside itself, exactly as the thinking panel has since it had the
 * same problem (.reasoning-body.live in style.css). The page stops growing, the
 * newest row keeps arriving at the floor, and every earlier one is still in the
 * box to be scrolled back to. Clipped, never dropped.
 *
 * An action rather than component state for the reason toolGlide is one: what
 * it needs is a measurement of rows it does not own, taken after the frame that
 * changed them.
 */
export function toolWindow(node: HTMLElement, opts: boolean | WindowOpts = true) {
  const read = (o: boolean | WindowOpts) =>
    typeof o === 'boolean' ? { on: o, follow: true } : { on: o.on !== false, follow: o.follow !== false }
  let { on, follow } = read(opts)
  let frame = 0
  // Whether the reader is riding the bottom. Scrolling UP inside the box to
  // re-read a row must not be undone by the next call arriving; coming back to
  // the floor picks the follow up again. Same rule as the thinking window, and
  // the same 24px floor — the box is five rows tall, so a wider band would mean
  // "anywhere".
  let pinned = true
  let lastTop = 0
  if (!follow) pinned = false

  // Which ends have something beyond them. Two classes rather than one, because
  // the fade has to say WHICH way there is more: a list masked at the bottom
  // while it is pinned there would be dimming the row that is running.
  const paint = () => {
    frame = 0
    if (!on) {
      node.classList.remove('fade-top', 'fade-bot')
      return
    }
    if (pinned) {
      node.scrollTop = node.scrollHeight
      lastTop = node.scrollTop
    }
    const fromBottom = node.scrollHeight - node.scrollTop - node.clientHeight
    node.classList.toggle('fade-top', node.scrollTop > 2)
    node.classList.toggle('fade-bot', fromBottom > 2)
  }
  const schedule = () => { if (!frame) frame = requestAnimationFrame(paint) }

  const onScroll = () => {
    const fromBottom = node.scrollHeight - node.scrollTop - node.clientHeight
    if (node.scrollTop < lastTop - 1 && fromBottom > 2) pinned = false
    // Never re-pins a list that does not follow: reaching the bottom of a
    // finished result list is reading it, not asking to be dragged along.
    else if (follow && fromBottom < 24) pinned = true
    lastTop = node.scrollTop
    schedule()
  }
  node.addEventListener('scroll', onScroll, { passive: true })

  // Rows arrive, rows change state, and a label can wrap when the pane is
  // resized — all three change what "the bottom" is.
  const rows = new MutationObserver(schedule)
  rows.observe(node, { childList: true, subtree: true, characterData: true })
  const size = new ResizeObserver(schedule)
  size.observe(node)
  paint()

  return {
    update(next: boolean | WindowOpts = true) {
      const read2 = read(next)
      on = read2.on
      follow = read2.follow
      if (!follow) pinned = false
      schedule()
    },
    destroy() {
      node.removeEventListener('scroll', onScroll)
      rows.disconnect()
      size.disconnect()
      if (frame) cancelAnimationFrame(frame)
    },
  }
}
