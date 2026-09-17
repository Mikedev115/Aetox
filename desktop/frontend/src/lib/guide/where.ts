// Where the guide is standing — read off the page, never asked of the app.
//
// The page stamps its own name on itself (`data-guide-place`, a PageId from
// lib/rooms.ts) and this module reads it. That is the whole module, and the
// smallness is the point: the guide has no idea how any room is built, which
// store holds its state, or that `cockpit` exists. A room added tomorrow is
// visible to the guide the moment it wears a sign, with nothing here to edit
// (owner, 15 ก.ย. 2026: *"ไกด์ไม่ผูกกับหน้าไหนเลย"*).
//
// It replaced `cockpit.activeView`, which could say "settings" but never
// *which* settings page — so the guide could not tell the model page from the
// memory page, and greeted both the same way.

import { CONTEXT_ATTR, PLACE_ATTR, isPageId, roomOf, type PageId } from '../rooms'

/**
 * The place on screen now, or null when nothing has said.
 *
 * The DEEPEST sign wins. Places nest — Settings draws its shell and then the
 * open section inside it — and the innermost one is the specific answer;
 * anything above it is the room it sits in, which `roomOf` can still recover.
 *
 * An unknown value is treated as no answer rather than trusted: the sign is
 * written by hand in a component, and a typo there should degrade the guide's
 * greeting, never put a place into circulation that nothing else knows.
 */
export function currentPage(): PageId | null {
  if (typeof document === 'undefined') return null
  const signs = document.querySelectorAll<HTMLElement>(`[${PLACE_ATTR}]`)
  let best: { page: PageId; depth: number } | null = null
  for (const el of signs) {
    const raw = el.getAttribute(PLACE_ATTR)
    if (!raw || !isPageId(raw)) continue
    const r = el.getBoundingClientRect()
    if (r.width <= 0 || r.height <= 0) continue // a room rendered but not shown
    let depth = 0
    for (let n: HTMLElement | null = el; n; n = n.parentElement) depth++
    if (!best || depth > best.depth) best = { page: raw, depth }
  }
  return best?.page ?? null
}

/** The room the guide is in — `settings.models` → `settings`. */
export function currentRoom(): string | null {
  const page = currentPage()
  return page ? roomOf(page) : null
}

/** The visible variant inside a page, such as assistant/coding on `chat`. */
export function currentContext(): string {
  if (typeof document === 'undefined') return ''
  const signs = document.querySelectorAll<HTMLElement>(`[${CONTEXT_ATTR}]`)
  let best: { value: string; depth: number } | null = null
  for (const el of signs) {
    const value = el.getAttribute(CONTEXT_ATTR) || ''
    const r = el.getBoundingClientRect()
    if (!value || r.width <= 0 || r.height <= 0) continue
    let depth = 0
    for (let n: HTMLElement | null = el; n; n = n.parentElement) depth++
    if (!best || depth > best.depth) best = { value, depth }
  }
  return best?.value ?? ''
}

/** Is this exactly where we are? */
export function isHere(page: PageId): boolean {
  return currentPage() === page
}

/** A real piece of a target has to be inside the viewport. Merely having a
 * layout box is not enough: drawers and duplicated menu content can remain
 * mounted just beyond the window, which used to leave the arrow as a tiny
 * amber sliver on the screen edge. */
export function rectOnScreen(r: Pick<DOMRect, 'left' | 'top' | 'width' | 'height'>): boolean {
  if (r.width <= 0 || r.height <= 0) return false
  if (typeof window === 'undefined') return true
  const viewportWidth = window.innerWidth || document.documentElement?.clientWidth || 0
  const viewportHeight = window.innerHeight || document.documentElement?.clientHeight || 0
  if (viewportWidth <= 0 || viewportHeight <= 0) return true
  // Real DOMRects always provide left/top. A few light-weight hosts and our
  // deterministic DOM tests expose only width/height, which means origin.
  const left = Number.isFinite(r.left) ? r.left : 0
  const top = Number.isFinite(r.top) ? r.top : 0
  const right = left + r.width
  const bottom = top + r.height
  const visibleWidth = Math.min(right, viewportWidth) - Math.max(left, 0)
  const visibleHeight = Math.min(bottom, viewportHeight) - Math.max(top, 0)
  // Eight pixels is enough for a compact icon to be genuinely usable, while
  // rejecting the one-pixel remnant of a panel animating out of the window.
  return visibleWidth >= Math.min(8, r.width) && visibleHeight >= Math.min(8, r.height)
}

function elementActuallyVisible(el: HTMLElement): boolean {
  const style = typeof getComputedStyle === 'function' ? getComputedStyle(el) : null
  if (style?.display === 'none' || style?.visibility === 'hidden' || style?.visibility === 'collapse') return false
  if (style?.opacity === '0') return false
  return rectOnScreen(el.getBoundingClientRect())
}

/** Return the visible copy of a guide target. Some Svelte snippets are
 * intentionally rendered in both the empty panel and its + menu; selecting
 * the first DOM copy can therefore select an off-screen clone. */
export function visibleGuideElement(id: string): HTMLElement | null {
  if (typeof document === 'undefined') return null
  const selector = '[data-guide=' + JSON.stringify(id) + ']'
  return Array.from(document.querySelectorAll<HTMLElement>(selector)).find(elementActuallyVisible) ?? null
}

/**
 * Is that mapped element on screen and pressable right now?
 *
 * The same question `currentPage` asks, one element down: not "which page" but
 * "is this thing there". One definition, because the walk (path.ts), the store
 * and the figure all have to agree on what "on screen" means — an element that
 * is in the document but laid out at nothing is not something a person can
 * press, and a guide standing beside it is standing beside nothing.
 */
export function onScreen(id: string): boolean {
  return visibleGuideElement(id) !== null
}
