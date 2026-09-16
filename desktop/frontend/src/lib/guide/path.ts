// How you get there BY HAND — the button to press next, not the page to jump to.
//
// The guide used to reach a stop by calling the app's own setters: the page
// simply changed. That teaches nothing. Somebody watching wants to know which
// button takes them there, and a screen that rearranges itself without a press
// leaves them exactly where they started — not knowing (owner, 15 ก.ย. 2026:
// *"ตอนกดพาไปนั่นนี่ ควรจะค่อยๆพาคลิกทีละจุดไม่ใช่เด้งไปเลย ไม่งั้นผู้ใช้ไม่รู้ว่า
// ควรจะกดไปตรงไหนครับ"*).
//
// The shape that makes this robust: **only ever compute the NEXT press.** Never
// a plan of five steps to follow blindly — the screen moves under every click
// (a menu opens, a room loads, a rail appears), so a plan made before the first
// click is stale by the second. Ask again after each one and the answer is
// always something that is on screen right now and really does lead onward.
//
// It also means this file needs no notion of "where we are". There is no walk
// state to get out of step with the DOM: the DOM is the state.

import { GUIDE_MAP } from './map'
import { CATALOG_TARGETS } from './catalog/data'
import { roomOf, type PageId } from '../rooms'
import { onScreen as visible } from './where'
import { t } from '../i18n.svelte'

/**
 * The doors into each room, outermost first — the presses a person makes to
 * get there from a chat window.
 *
 * Six rows, not a chain on each of 157 map entries: what varies between two
 * buttons in Settings is not how you reach Settings. The rail row for the
 * exact section is appended by `railDoorFor` below, read off the map itself.
 */
const ROOM_DOORS: Record<string, string[]> = {
  chat: [],
  // The gear lives inside the account menu, so the menu is pressed first.
  settings: ['sidebar.footer', 'account.settings'],
  capability: ['sidebar.desk.capability'],
  office: ['sidebar.desk.office'],
  artifacts: ['sidebar.desk.artifacts'],
}

/**
 * The rail button that opens the page a target sits on, found in the map
 * rather than listed here: it is the `*.rail.*` entry whose own page is the
 * same one. So a section that gets a new rail id, or moves, needs no edit —
 * and a section with no mapped rail button simply has no rail step.
 */
function railDoorFor(page: PageId): string | null {
  return GUIDE_MAP.find((e) => e.page === page && /\.rail\./.test(e.id))?.id ?? null
}

/**
 * Check semantic DOM preconditions for a target without coupling to component stores.
 */
export function checkPrecondition(targetId: string): { ok: boolean; reason?: string } {
  const entry = CATALOG_TARGETS.find((e) => e.id === targetId)
  if (!entry?.navigation?.condition) return { ok: true }
  if (typeof document === 'undefined') return { ok: true }

  const cond = entry.navigation.condition
  if (cond.selector) {
    const el = document.querySelector(cond.selector)
    if (!el) {
      return {
        ok: false,
        reason: cond.notOnScreenMessageKey ? t(cond.notOnScreenMessageKey as any) : undefined,
      }
    }
    if (cond.attr && cond.value !== undefined) {
      if (el.getAttribute(cond.attr) !== cond.value) {
        return {
          ok: false,
          reason: cond.notOnScreenMessageKey ? t(cond.notOnScreenMessageKey as any) : undefined,
        }
      }
    }
  }

  return { ok: true }
}

/**
 * The next thing to press on the way to `targetId`, or null when the target
 * itself is reachable (press it, or simply stand beside it and explain).
 *
 * Evaluates in-page flow preconditions (via) as well as room/rail doors.
 * Returns only ids that are visible right now, so what the guide points at is
 * always something the person can actually put a finger on.
 */
export function nextStepTo(targetId: string): string | null {
  if (visible(targetId)) return null
  const catalogEntry = CATALOG_TARGETS.find((e) => e.id === targetId)
  const target = catalogEntry || GUIDE_MAP.find((e) => e.id === targetId)
  if (!target) return null

  // 1. Check in-page navigation preconditions (via)
  // If a precondition button is visible on screen right now, that is the next press!
  if (catalogEntry?.navigation?.via) {
    for (const viaId of catalogEntry.navigation.via) {
      if (viaId === targetId) break
      if (visible(viaId)) return viaId
    }
  }

  // 2. Inter-room doors and rail doors
  const chain = [...(ROOM_DOORS[roomOf(target.page)] ?? [])]
  const rail = railDoorFor(target.page)
  if (rail && rail !== targetId) chain.push(rail)

  if (catalogEntry?.navigation?.via) {
    for (const viaId of catalogEntry.navigation.via) {
      if (!chain.includes(viaId) && viaId !== targetId) chain.push(viaId)
    }
  }

  // The DEEPEST visible door, not the first.
  let next: string | null = null
  for (const step of chain) {
    if (step === targetId) break
    if (visible(step)) next = step
  }
  return next
}

/**
 * Everything the guide would have you press to reach the target from here,
 * assuming each press lands. For telling the person how far it is — never for
 * walking blind; the walk itself asks `nextStepTo` again every time.
 */
export function stepsTo(targetId: string): string[] {
  if (visible(targetId)) return []
  const catalogEntry = CATALOG_TARGETS.find((e) => e.id === targetId)
  const target = catalogEntry || GUIDE_MAP.find((e) => e.id === targetId)
  if (!target) return []

  const chain = [...(ROOM_DOORS[roomOf(target.page)] ?? [])]
  const rail = railDoorFor(target.page)
  if (rail && rail !== targetId) chain.push(rail)

  if (catalogEntry?.navigation?.via) {
    for (const viaId of catalogEntry.navigation.via) {
      if (!chain.includes(viaId) && viaId !== targetId) chain.push(viaId)
    }
  }

  return chain.filter((s) => s !== targetId)
}
