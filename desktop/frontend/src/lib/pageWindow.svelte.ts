// How a long grid is drawn: a window over the list, grown as the reader
// reaches the end of it.
//
// ห้องความสามารถ draws 494 skill cards on ของคุณ (the bundled shelf is that
// large), and every card is a real DOM subtree: 13,700 nodes for one page,
// built in one frame, on WebView2. The owner watched it land and asked for
// the load to be shaped rather than fixed one page at a time ("ทำสถาปัตยกรรม
// การโหลดดีๆ แบ่งโหลด ไม่งั้นแย่ … หน้านี้ทั้งหมดเลย MCP อีก", 13 ก.ย. 2026).
//
// This is the whole mechanism: one number per grid. The list itself is
// already in memory — the engine hands over every skill in one call and that
// is cheap — so nothing here fetches; what it spends is DOM, and DOM is what
// it rations. A grid under its step draws whole and never pays for the
// machinery.
//
// The sentinel below the last row grows the window when it comes into view,
// 400px early so the next rows exist before the reader reaches them, and the
// same block carries a plain button: a window that can ONLY be grown by
// scrolling is a window a keyboard cannot open (DESIGN.md §4 — no state
// reachable by one route only).
export function pageWindow(step = 48) {
  let shown = $state(step)
  return {
    get shown() {
      return shown
    },
    /** The slice to draw. Returns the array itself when it all fits. */
    take<T>(rows: readonly T[]): readonly T[] {
      return rows.length <= shown ? rows : rows.slice(0, shown)
    },
    /** Whether anything is being held back — draw the sentinel when true. */
    more(total: number) {
      return total > shown
    },
    /** How many are held back, for the button's own words. */
    rest(total: number) {
      return Math.max(0, total - shown)
    },
    grow(total: number) {
      shown = Math.min(total, shown + step)
    },
    /** Back to one step. Called when the list underneath changes identity. */
    reset() {
      shown = step
    },
  }
}

export type PageWindow = ReturnType<typeof pageWindow>

/**
 * Calls `onMore` when the node is near the viewport. A Svelte action so the
 * observer lives and dies with the sentinel it is attached to.
 */
export function nearViewport(node: HTMLElement, onMore: () => void) {
  let cb = onMore
  if (typeof IntersectionObserver === 'undefined') return { update(next: () => void) { cb = next } }
  const io = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) cb()
    },
    // Generous on both sides: a sentinel the reader jumped past (End key, a
    // fast wheel) still counts as reached.
    { rootMargin: '600px 0px' },
  )
  io.observe(node)
  return {
    update(next: () => void) {
      cb = next
    },
    destroy() {
      io.disconnect()
    },
  }
}
