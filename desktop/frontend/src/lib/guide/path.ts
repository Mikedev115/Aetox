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

import { GUIDE_MAP, guideEntry } from './map'
import { CATALOG_TARGETS } from './catalog/data'
import type { NavigationResolution } from './catalog/types'
import { roomOf, type PageId } from '../rooms'
import { currentPage, onScreen as visible } from './where'
import { t } from '../i18n.svelte'

export type { NavigationResolution }

/**
 * The doors into each room, outermost first — the presses a person makes to
 * get there from a chat window.
 *
 * Six rows, not a chain on each of 157 map entries: what varies between two
 * buttons in Settings is not how you reach Settings. The rail row for the
 * exact section is appended by `railDoorFor` below, read off the map itself.
 */
// Every room below is entered through the left sidebar. The sidebar can be
// folded while its children remain mounted outside the viewport, so the first
// honest door is always the top-bar toggle. When the sidebar is already open,
// the deeper visible button wins and this step is naturally skipped.
const SIDEBAR_DOOR = 'topbar.sidebar_btn'
const SIDEBAR_ENTRY_ROOMS = new Set(['settings', 'capability', 'office', 'artifacts'])

const ROOM_DOORS: Record<string, string[]> = {
  // The assistant desk is the visible way back to Chat from another room.
  chat: [SIDEBAR_DOOR, 'sidebar.desk.assistant'],
  // The gear lives inside the account menu, so the menu is pressed first.
  settings: [SIDEBAR_DOOR, 'sidebar.footer', 'account.settings'],
  capability: [SIDEBAR_DOOR, 'sidebar.desk.capability'],
  office: [SIDEBAR_DOOR, 'sidebar.desk.office'],
  artifacts: [SIDEBAR_DOOR, 'sidebar.desk.artifacts'],
}

/** The compact 57 px sidebar still leaves its footer icon inside the
 * viewport, so geometry alone cannot tell whether the full navigation is
 * usable. The toggle exposes the state semantically for the guide and screen
 * readers; no component store is coupled into the route graph. */
function sidebarIsFolded(): boolean {
  if (typeof document === 'undefined') return false
  return document.querySelector<HTMLElement>(`[data-guide="${SIDEBAR_DOOR}"]`)?.getAttribute('aria-expanded') === 'false'
}

function targetNeedsExpandedSidebar(id: string, page: PageId): boolean {
  return id.startsWith('sidebar.')
    || id.startsWith('account.')
    || SIDEBAR_ENTRY_ROOMS.has(roomOf(page))
}

function doorVisible(id: string): boolean {
  if (sidebarIsFolded() && id !== SIDEBAR_DOOR && (id.startsWith('sidebar.') || id.startsWith('account.'))) return false
  return visible(id)
}

/** A full-screen room can hide the global doors needed by the destination.
 * Leave that room first, then recompute the route from the screen that really
 * appeared. Keeping this as a one-step escape (rather than appending a blind
 * plan) preserves the guide's "one real press, then look again" rule. */
const ROOM_EXITS: Partial<Record<string, string>> = {
  settings: 'settings.back',
  capability: 'room.back',
  office: 'room.back',
  artifacts: 'room.back',
  projects: 'room.back',
  videowork: 'room.back',
}

function visibleRoomExit(destination: PageId): string | null {
  const here = currentPage()
  if (!here || roomOf(here) === roomOf(destination)) return null
  const exit = ROOM_EXITS[roomOf(here)]
  return exit && visible(exit) ? exit : null
}

/** Controls whose job is to change the current page. When the page sign is
 * briefly absent during a transition, these must still be treated as doors,
 * never as proof that their destination has already been reached. */
function isPageDoor(id: string): boolean {
  return id === 'account.settings'
    || id === 'settings.back'
    || id === 'room.back'
    || id.startsWith('sidebar.desk.')
    || id.includes('.rail.')
}

export type PageNavigationResolution = {
  page: PageId
  reachable: boolean
  available: boolean
  nextStepId: string | null
  stepsRemaining: string[]
  blockedReason?: string
}

function chainToPage(page: PageId): string[] {
  const chain = [...(ROOM_DOORS[roomOf(page)] ?? [])]
  const rail = railDoorFor(page)
  if (rail && !chain.includes(rail)) chain.push(rail)
  return chain
}

function remainingFromVisible(chain: string[]): string[] {
  let deepest = -1
  for (let i = 0; i < chain.length; i++) {
    if (doorVisible(chain[i])) deepest = i
  }
  return deepest < 0 ? chain : chain.slice(deepest)
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
  const catalogEntry = CATALOG_TARGETS.find((e) => e.id === targetId)
  const target = catalogEntry || guideEntry(targetId)
  if (!target) return null

  // Room identity outranks a measurable box left mounted behind a full-screen
  // room. Settings keeps the chat shell alive underneath; its sidebar buttons
  // still have rectangles, but a person cannot see or press them until they
  // leave Settings.
  const exit = visibleRoomExit(target.page)
  if (exit) return exit

  // A folded sidebar can leave compact icons measurable. Opening it is still
  // the first teaching step: otherwise the arrow names controls that are
  // missing their labels and the user cannot see the route being described.
  if (sidebarIsFolded() && targetNeedsExpandedSidebar(targetId, target.page) && visible(SIDEBAR_DOOR)) {
    return SIDEBAR_DOOR
  }

  // A navigation control can be visible before its destination is reached:
  // the Team rail is on General, the Capability door is on Chat. In that
  // state the control itself is the next press, not an already-reached stop.
  const here = currentPage()
  if (visible(targetId)) {
    if (catalogEntry?.policy.actionType === 'click' && isPageDoor(targetId) && here !== target.page) return targetId
    return null
  }

  // 1. Check in-page navigation preconditions (via).
  //
  // A `via` list is ordered from the outer door to the control nearest the
  // destination. Several doors can remain visible at once: after the right
  // inspector opens, both its toggle and the `+` button are on screen. Picking
  // the first visible item sends the person back to the inspector toggle and
  // can even close the panel they just opened. The deepest visible door is the
  // next honest press — the same rule used by the room chain below.
  if (catalogEntry?.navigation?.via) {
    let deepestVisible: string | null = null
    for (const viaId of catalogEntry.navigation.via) {
      if (viaId === targetId) break
      if (doorVisible(viaId)) deepestVisible = viaId
    }
    if (deepestVisible) return deepestVisible
  }

  // 2. Inter-room doors and rail doors
  const chain = [...(ROOM_DOORS[roomOf(target.page)] ?? [])]
  const rail = railDoorFor(target.page)
  // An active rail cannot reveal anything else on the page that is already
  // open. Treating it as a route here creates the loop the user saw on MCP:
  // point at the selected row -> click it -> "the screen did not change".
  if (rail && rail !== targetId && here !== target.page) chain.push(rail)

  if (catalogEntry?.navigation?.via) {
    for (const viaId of catalogEntry.navigation.via) {
      if (!chain.includes(viaId) && viaId !== targetId) chain.push(viaId)
    }
  }

  // The DEEPEST visible door, not the first.
  let next: string | null = null
  for (const step of chain) {
    if (step === targetId) break
    if (doorVisible(step)) next = step
  }
  return next
}

/**
 * Everything the guide would have you press to reach the target from here,
 * assuming each press lands. For telling the person how far it is — never for
 * walking blind; the walk itself asks `nextStepTo` again every time.
 */
export function stepsTo(targetId: string): string[] {
  const catalogEntry = CATALOG_TARGETS.find((e) => e.id === targetId)
  const target = catalogEntry || guideEntry(targetId)
  if (!target) return []

  const exit = visibleRoomExit(target.page)
  if (exit) return [exit, ...chainToPage(target.page)]

  if (sidebarIsFolded() && targetNeedsExpandedSidebar(targetId, target.page)) {
    if (targetId.startsWith('sidebar.')) return [SIDEBAR_DOOR]
    if (targetId.startsWith('account.')) return [SIDEBAR_DOOR, 'sidebar.footer']
  }

  const here = currentPage()
  if (visible(targetId)) {
    if (catalogEntry?.policy.actionType === 'click' && isPageDoor(targetId) && here !== target.page) return [targetId]
    return []
  }

  const chain = [...(ROOM_DOORS[roomOf(target.page)] ?? [])]
  const rail = railDoorFor(target.page)
  if (rail && rail !== targetId && here !== target.page) chain.push(rail)

  if (catalogEntry?.navigation?.via) {
    for (const viaId of catalogEntry.navigation.via) {
      if (!chain.includes(viaId) && viaId !== targetId) chain.push(viaId)
    }
  }

  return remainingFromVisible(chain.filter((s) => s !== targetId))
}

/**
 * Deterministic navigation resolution returning full availability, next step,
 * semantic preconditions, and explanation if blocked.
 */
export function resolveNavigation(targetId: string): NavigationResolution {
  const catalogEntry = CATALOG_TARGETS.find((e) => e.id === targetId)
  const target = catalogEntry || guideEntry(targetId)

  if (!target) {
    return {
      targetId,
      reachable: false,
      available: false,
      nextStepId: null,
      stepsRemaining: [],
      preconditionMet: false,
      blockedReason: t('guide.unknownTarget' as any) || ('Unknown target: ' + targetId),
    }
  }

  const isAvailable = visible(targetId)
    && !(sidebarIsFolded() && targetNeedsExpandedSidebar(targetId, target.page))

  const pre = checkPrecondition(targetId)

  const here = currentPage()
  const visibleDoorStillNeedsPress = isAvailable
    && catalogEntry?.policy.actionType === 'click'
    && isPageDoor(targetId)
    && here !== target.page

  if (isAvailable && !visibleDoorStillNeedsPress) {
    return {
      targetId,
      reachable: pre.ok,
      available: true,
      nextStepId: null,
      stepsRemaining: [],
      preconditionMet: pre.ok,
      blockedReason: pre.ok ? undefined : pre.reason,
    }
  }

  const nextStep = nextStepTo(targetId)
  const remaining = stepsTo(targetId)
  // A theoretical chain is not enough. Guidance may only promise a route
  // when its next physical button is on screen right now.
  const reachable = nextStep !== null

  let blockedReason: string | undefined = undefined
  if (!pre.ok) {
    blockedReason = pre.reason
  } else if (catalogEntry?.navigation?.condition?.notOnScreenMessageKey) {
    blockedReason = t(catalogEntry.navigation.condition.notOnScreenMessageKey as any)
  } else if (catalogEntry?.navigation?.prerequisiteActionKey) {
    blockedReason = t(catalogEntry.navigation.prerequisiteActionKey as any)
  }

  return {
    targetId,
    reachable,
    available: false,
    nextStepId: nextStep,
    stepsRemaining: remaining,
    preconditionMet: pre.ok && !catalogEntry?.navigation?.via?.length,
    blockedReason,
  }
}

/**
 * Resolve the next real press needed to reach a page.
 *
 * A page destination is deliberately different from an element destination:
 * the Settings "Brain" rail button lives on every Settings section, but
 * merely standing beside that button has not reached settings.models. For a
 * page goal the rail is therefore a door that must be pressed.
 */
export function resolvePageNavigation(page: PageId): PageNavigationResolution {
  if (currentPage() === page) {
    return {
      page,
      reachable: true,
      available: true,
      nextStepId: null,
      stepsRemaining: [],
    }
  }

  const exit = visibleRoomExit(page)
  if (exit) {
    return {
      page,
      reachable: true,
      available: false,
      nextStepId: exit,
      stepsRemaining: [exit, ...chainToPage(page)],
    }
  }

  const chain = chainToPage(page)
  if (sidebarIsFolded() && SIDEBAR_ENTRY_ROOMS.has(roomOf(page)) && visible(SIDEBAR_DOOR)) {
    return {
      page,
      reachable: true,
      available: false,
      nextStepId: SIDEBAR_DOOR,
      stepsRemaining: remainingFromVisible(chain),
    }
  }
  let nextStepId: string | null = null
  for (const step of chain) {
    if (doorVisible(step)) nextStepId = step
  }
  const stepsRemaining = remainingFromVisible(chain)
  return {
    page,
    reachable: nextStepId !== null,
    available: false,
    nextStepId,
    stepsRemaining,
    blockedReason: nextStepId ? undefined : t('guide.navigationBlocked' as any),
  }
}

export type NavigationProgress = {
  progressed: boolean
  page: PageId | null
}

/**
 * Wait for evidence that the user's press actually changed the UI.
 *
 * The old walk advanced after a fixed 220 ms. Slow pages could therefore be
 * skipped and buttons that did nothing were treated as successful. This loop
 * reads the same page signs and visible guide ids used by routing, and only
 * advances when one of those facts changes.
 */
export async function waitForNavigationProgress(
  resolve: () => Pick<NavigationResolution, 'available' | 'nextStepId'> | Pick<PageNavigationResolution, 'available' | 'nextStepId'>,
  awaitedId: string,
  beforePage: PageId | null,
  timeoutMs = 1800,
): Promise<NavigationProgress> {
  if (typeof document === 'undefined') return { progressed: true, page: null }
  const started = Date.now()
  while (Date.now() - started < timeoutMs) {
    // Let the button's own click handler run after Guide's capture listener.
    await new Promise((done) => setTimeout(done, 32))
    const page = currentPage()
    const nav = resolve()
    if (page !== beforePage || nav.available || nav.nextStepId !== awaitedId || !visible(awaitedId)) {
      return { progressed: true, page }
    }
  }
  return { progressed: false, page: currentPage() }
}
