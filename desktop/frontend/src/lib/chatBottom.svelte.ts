// Standing at the foot of the transcript — and letting go of it.
//
// A chat has one default: the newest line is the one you are meant to be
// looking at. Everything here exists to make that true without ever trapping
// somebody who wants to read back (owner, 15 ก.ย. 2026: *"แชทมันควรจะพามาอยู่
// ล่างสุดโดยอัตโนมัติสิ ไม่ใช่ล็อคนะ … ผู้ใช้ยังเลื่อนไปนั่นนี่ได้ แต่ค่าเริ่มต้น
// อ่ะ ควรจะอยู่ล่างสิ"*).
//
// Three jobs, and the middle one is what was missing:
//
//  1. WHO lets go — the reader, by scrolling up, however little.
//  2. WHEN to re-measure — every time the CONTENT's height moves, and not only
//     when a store changed. The component used to follow store changes and
//     paced frames, which covers a stream of words and nothing else: a picture
//     finishing its load, a code block laying out, a stretch of tool rows
//     folding away over 300ms — all of those move the floor with no store
//     change at all, and the view was left hanging above it. That is the whole
//     of "บางทีแสดงผลผิดเพี้ยน ผิดตำแหน่งและจังหวะ", and the fold that "ไม่สมูท"
//     is the same thing in motion: a height animating under a scroller nobody
//     was watching.
//  3. WHAT re-arms it — sending a message, opening a chat, pressing the button.
//     A person who sends something has said where they want to be looking.
//
// Its own module because those are three different tunings — a band in pixels,
// an observer, and a list of moments — and because a scroller this fiddly is
// worth testing without rendering six thousand lines of chat around it.

/** How near the floor counts as being on it, for re-arming. Generous, because
 *  it is only ever reached by somebody scrolling DOWN: they are on their way
 *  back and the last few pixels are not a decision. */
const REPIN_BAND = 80

export class BottomStick {
  /** Following the newest line. The jump button is this state, drawn. */
  following = $state(true)

  private el: HTMLElement | null = null
  private lastTop = 0
  private ro: ResizeObserver | null = null

  /**
   * Wire the scroller, and whatever inside it changes height.
   *
   * Both are observed and for different reasons: the content grows (a message,
   * a picture, a fold closing), and the scroller itself shrinks (the composer
   * growing under it, the window resizing). Either moves the floor.
   *
   * Returns the way to unwire, which the caller must use — an observer that
   * outlives the element keeps it alive.
   */
  attach(scroller: HTMLElement, content?: HTMLElement | null): () => void {
    this.el = scroller
    this.lastTop = scroller.scrollTop
    if (typeof ResizeObserver !== 'undefined') {
      this.ro = new ResizeObserver(() => this.paint())
      this.ro.observe(scroller)
      if (content) this.ro.observe(content)
    }
    return () => {
      this.ro?.disconnect()
      this.ro = null
      if (this.el === scroller) this.el = null
    }
  }

  /**
   * The reader moved. Any upward movement lets go, however little.
   *
   * The pin itself only ever scrolls DOWN, so up can only be the reader — and
   * it must not be judged by the re-pin band below: a touchpad scrolls a few px
   * per event, which never cleared 80px before the next chunk snapped the view
   * back down. That loop read as "เลื่อนขึ้นไม่ได้เลย".
   *
   * The `fromBottom` guard covers the one non-reader way scrollTop can drop:
   * the transcript shrinking (a fold, an undo, a session switch) clamps it —
   * but a clamp lands AT the floor, and a real upward scroll never does.
   */
  onScroll() {
    const el = this.el
    if (!el) return
    const fromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    if (el.scrollTop < this.lastTop - 1 && fromBottom > 2) this.following = false
    else if (fromBottom < REPIN_BAND) this.following = true
    this.lastTop = el.scrollTop
  }

  /**
   * Sit on the floor, if that is where we are meant to be.
   *
   * Idempotent and cheap to call from anywhere — a store change, a painted
   * frame, an observed resize. `lastTop` is written with the value the browser
   * settled on rather than the one asked for, so the scroll event this raises
   * is recognised as our own and never reads as the reader letting go.
   */
  paint() {
    const el = this.el
    if (!el || !this.following) return
    el.scrollTop = el.scrollHeight
    this.lastTop = el.scrollTop
  }

  /**
   * Take me to the newest line and keep me there — the answer to an explicit
   * act, never to something arriving on its own.
   *
   * Sending is the important caller. Somebody who types into a running turn is
   * asking to see what happens next, and until this existed their message
   * landed somewhere below the fold while the screen they were looking at sat
   * still — the app reading, correctly, as frozen (owner, 15 ก.ย. 2026:
   * *"แชทที่ส่งระหว่างเทิน ไม่ควรค้างแบบนี้ครับ"*).
   */
  follow() {
    this.following = true
    this.paint()
  }
}
