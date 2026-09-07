import { motionStill } from './fold'

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
/** How long the window takes to travel to a new floor, and the jump below which
 *  it does not bother. */
const RIDE_MS = 460
const RIDE_MIN_PX = 48

export function toolWindow(node: HTMLElement, opts: boolean | WindowOpts = true) {
  const read = (o: boolean | WindowOpts) =>
    typeof o === 'boolean' ? { on: o, follow: true } : { on: o.on !== false, follow: o.follow !== false }
  let { on, follow } = read(opts)
  let frame = 0
  let ride = 0
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
    if (pinned) rideDown()
    const fromBottom = node.scrollHeight - node.scrollTop - node.clientHeight
    node.classList.toggle('fade-top', node.scrollTop > 2)
    node.classList.toggle('fade-bot', fromBottom > 2)
  }
  const schedule = () => { if (!frame) frame = requestAnimationFrame(paint) }

  /** The window travelling to its new floor rather than teleporting there.
   *
   *  One row arriving moves the floor by a row and is not worth animating — the
   *  row's own arrival already says what happened. A BATCH is different: the
   *  agent groups its read-only calls and runs them together
   *  (internal/cognitive/agent.go), so five rows land in one frame and the floor
   *  drops by a screenful between two paints. Teleporting there is the page
   *  taking the wire's rhythm, which streamPace.ts already refused for text and
   *  which the owner asked for here in the same words: *"ต่อให้ตู้มมาทีเดียว
   *  หน้าบ้านก็ควรสมูธครับ"*.
   *
   *  cubicInOut, the curve `settle` uses, for the reason written there: it
   *  starts slow, moves, and arrives slowly, which is the whole of what gently
   *  means to an eye. Retargeting mid-ride is correct and deliberate — another
   *  batch landing while this one is still travelling should extend the
   *  journey, not queue a second one behind it. */
  const rideDown = () => {
    const to = node.scrollHeight - node.clientHeight
    const from = node.scrollTop
    const gap = to - from
    if (ride) { cancelAnimationFrame(ride); ride = 0 }
    if (gap <= 0) return
    if (motionStill() || gap < RIDE_MIN_PX) {
      node.scrollTop = to
      lastTop = node.scrollTop
      return
    }
    const start = performance.now()
    const step = (now: number) => {
      const t = Math.min((now - start) / RIDE_MS, 1)
      const e = t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2
      node.scrollTop = from + gap * e
      lastTop = node.scrollTop
      ride = t < 1 ? requestAnimationFrame(step) : 0
    }
    ride = requestAnimationFrame(step)
  }

  const onScroll = () => {
    // Our own ride fires these too, and treating them as the reader's would
    // have the ride cancel and restart itself on every frame of the ride.
    if (ride) { lastTop = node.scrollTop; return }
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
      if (ride) cancelAnimationFrame(ride)
    },
  }
}
