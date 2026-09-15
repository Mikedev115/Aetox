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

import { PLACE_ATTR, isPageId, roomOf, type PageId } from '../rooms'

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

/** Is this exactly where we are? */
export function isHere(page: PageId): boolean {
  return currentPage() === page
}
