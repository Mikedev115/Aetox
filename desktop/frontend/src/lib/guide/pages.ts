// Opening a place outright — the one module that knows how this app is wired.
//
// Everything else about the guide speaks PageId and nothing else: the map, the
// signs on each page, the door chains, the greeting. This file is the single
// adapter between that vocabulary and `cockpit`'s setters, so when the app
// reorganises its views there is exactly one place to follow it.
//
// It is also the guide's LAST resort, not its usual way to travel. Walking
// somebody to a button teaches them where it is; changing the page under them
// teaches nothing (path.ts). This runs when no door is visible — arriving
// unexplained still beats not arriving — and for the model's own `goto`.
import { tick } from 'svelte'
import { cockpit, setActiveView, openCapabilityAt } from '../stores/cockpit.svelte'
import { PLACE_ATTR, roomOf, type PageId, type CapabilityPage, type SettingsSection } from '../rooms'
import { currentPage } from './where'

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// How long to wait for the place to answer. A view swap mounts a whole page
// and its data; standing still needs no patience at all.
const SETTLE_AFTER_CHANGE_MS = 2000
const SETTLE_IN_PLACE_MS = 250

/** The section half of a place: `settings.models` → `models`. */
function sectionOf(page: PageId): string {
  const dot = page.indexOf('.')
  return dot < 0 ? '' : page.slice(dot + 1)
}

/**
 * Open `page`, and wait until it says it is there.
 *
 * The wait watches the page's own sign (where.ts) rather than a selector the
 * caller invented, so "has it arrived" has one answer across the guide. A
 * caller that also needs a particular element on screen passes it as
 * `targetSelector` and the wait covers both.
 */
export async function openPage(page: PageId, targetSelector?: string, head?: 'assistant' | 'coding'): Promise<void> {
  const before = currentPage()
  const room = roomOf(page)
  const section = sectionOf(page)

  switch (room) {
    case 'settings':
      cockpit.settingsIntent = { section: section as SettingsSection, ...(head ? { head } : {}) }
      setActiveView('settings')
      break
    case 'capability':
      openCapabilityAt(section as CapabilityPage)
      break
    default:
      if (cockpit.activeView !== room) setActiveView(room)
  }

  await tick()
  if (typeof document === 'undefined') return

  const budget = before === page ? SETTLE_IN_PLACE_MS : SETTLE_AFTER_CHANGE_MS
  const start = Date.now()
  while (Date.now() - start < budget) {
    // Either piece of evidence is enough, and neither is required of the
    // other: the sign says we are there, or the thing we came for is on
    // screen. Demanding both means a page that has not been signed yet — or a
    // test with no signs at all — burns the whole budget before every step,
    // which the walk feels as a stall.
    if (currentPage() === page) return
    if (targetSelector && document.querySelector(targetSelector)) return
    // Nothing in this document ever says where it is — a room not signed yet,
    // or a test harness with no pages in it. There is no arrival to wait for,
    // and waiting anyway is a stall before every step of a walk.
    if (!targetSelector && !document.querySelector(`[${PLACE_ATTR}]`)) return
    await sleep(40)
    await tick()
  }
}
