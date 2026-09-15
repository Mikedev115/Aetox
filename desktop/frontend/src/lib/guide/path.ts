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

import { GUIDE_MAP, type GuidePage } from './map'

/** On screen and pressable right now. */
function visible(id: string): boolean {
  if (typeof document === 'undefined') return false
  const el = document.querySelector<HTMLElement>('[data-guide=' + JSON.stringify(id) + ']')
  if (!el) return false
  const r = el.getBoundingClientRect()
  return r.width > 0 && r.height > 0
}

/**
 * The doors into each room, outermost first — the presses a person makes to
 * get there from a chat window.
 *
 * Six rows, not a chain on each of 157 map entries: what varies between two
 * buttons in Settings is not how you reach Settings. The rail row for the
 * exact section is appended by `railDoorFor` below, read off the map itself.
 */
const ROOM_DOORS: Record<GuidePage['view'], string[]> = {
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
function railDoorFor(page: GuidePage): string | null {
  const sameRoom = GUIDE_MAP.filter((e) => {
    if (e.page.view !== page.view) return false
    if (page.view === 'settings' && e.page.view === 'settings') return e.page.rail === page.rail
    if (page.view === 'capability' && e.page.view === 'capability') return e.page.page === page.page
    return false
  })
  return sameRoom.find((e) => /\.rail\./.test(e.id))?.id ?? null
}

/**
 * The next thing to press on the way to `targetId`, or null when the target
 * itself is reachable (press it, or simply stand beside it and explain).
 *
 * Returns only ids that are visible right now, so what the guide points at is
 * always something the person can actually put a finger on. When nothing on
 * the way is visible — a room reached from a screen the map does not cover —
 * it answers null and the caller falls back to opening the page outright,
 * because arriving unexplained still beats not arriving.
 */
export function nextStepTo(targetId: string): string | null {
  if (visible(targetId)) return null
  const target = GUIDE_MAP.find((e) => e.id === targetId)
  if (!target) return null

  const chain = [...ROOM_DOORS[target.page.view]]
  const rail = railDoorFor(target.page)
  if (rail && rail !== targetId) chain.push(rail)

  // The DEEPEST visible door, not the first. The chain is ordered
  // outermost-inward, and pressing one does not make it disappear — the
  // account menu's own button is still on screen once the menu is open. Taking
  // the first visible one would therefore point at it again forever, and the
  // second press would close what the first press opened. The deepest one that
  // is on screen is the furthest the person has already got.
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
  const target = GUIDE_MAP.find((e) => e.id === targetId)
  if (!target) return []
  const chain = [...ROOM_DOORS[target.page.view]]
  const rail = railDoorFor(target.page)
  if (rail && rail !== targetId) chain.push(rail)
  return chain.filter((s) => s !== targetId)
}
