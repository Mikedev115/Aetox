// Where native child windows are drawn, in CSS px from the window's top-left.
//
// A browser tab is a real Win32 window glued over its pane (BrowserPane), and
// a native window composites over EVERY DOM layer — z-index is a fact about
// the DOM and the compositor never asks it. So an overlay that must stay in
// view over the whole window, the companion first of all, cannot be drawn on
// top of a tab; it can only stay out of the tab's rectangle. This is the
// register it reads. The pane writes its rect while the tab is on screen and
// clears it the moment it is not, so a rect here means "covered, now".
//
// Owner, 15 ก.ย. 2026: the browser pane over the avatar, "ซึ่งไม่ปกติ".

export type NativeRect = { x: number; y: number; w: number; h: number }

const rects = $state<Record<string, NativeRect>>({})

export const nativeRects = {
  /** Every native rectangle on screen right now. */
  get all(): NativeRect[] {
    return Object.values(rects)
  },
}

/** A native window is at `r`, or (null) gone from the screen. */
export function setNativeRect(id: string, r: NativeRect | null): void {
  if (r === null) {
    if (id in rects) delete rects[id]
    return
  }
  const cur = rects[id]
  if (cur && cur.x === r.x && cur.y === r.y && cur.w === r.w && cur.h === r.h) return
  rects[id] = r
}

/** Does the box at p (size×size) overlap r, with `gap` px of air kept round it? */
export function overlaps(p: { x: number; y: number }, size: number, r: NativeRect, gap: number): boolean {
  return p.x + size + gap > r.x && p.x - gap < r.x + r.w && p.y + size + gap > r.y && p.y - gap < r.y + r.h
}

/**
 * The nearest spot for a size×size box that is inside `win` and clear of
 * every rect. The ways out of each rect the box overlaps — left, right,
 * above, below — that fit in the window are the candidates; one clear of
 * every rect wins over one that only trades rects, and the shortest of
 * those wins. A box with nowhere to go — a tab filling the window — is left
 * where it is; under the tab is the only place there is.
 */
export function keepOut(
  p: { x: number; y: number },
  size: number,
  win: { w: number; h: number },
  gap: number,
  list: NativeRect[] = nativeRects.all,
): { x: number; y: number } {
  const hit = list.filter((r) => overlaps(p, size, r, gap))
  if (hit.length === 0) return p
  const fits = (q: { x: number; y: number }) => q.x >= gap && q.y >= gap && q.x + size + gap <= win.w && q.y + size + gap <= win.h
  const clear = (q: { x: number; y: number }) => !list.some((r) => overlaps(q, size, r, gap))
  const far = (q: { x: number; y: number }) => Math.hypot(q.x - p.x, q.y - p.y)
  const ways = hit.flatMap((r) => [
    { x: r.x - size - gap, y: p.y },
    { x: r.x + r.w + gap, y: p.y },
    { x: p.x, y: r.y - size - gap },
    { x: p.x, y: r.y + r.h + gap },
  ]).filter(fits)
  if (ways.length === 0) return p
  const free = ways.filter(clear)
  const pick = (free.length > 0 ? free : ways).sort((a, b) => far(a) - far(b))[0]
  return pick
}
