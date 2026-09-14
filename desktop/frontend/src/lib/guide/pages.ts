// Reaching the page a map entry lives on, with the app's own setters — no
// router of its own, no restructuring of App.svelte. A Settings page is
// reached through cockpit.settingsIntent, which Settings takes whether it is
// already open or about to be; a head's page is the intent naming the head.
import { tick } from 'svelte'
import { cockpit, setActiveView, openCapabilityAt } from '../stores/cockpit.svelte'
import type { GuidePage } from './map'

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// How long to wait for the element after a page change, and after none. A
// view swap mounts a whole page and its data; a section that is already on
// screen either has the element or never will.
const SETTLE_AFTER_CHANGE_MS = 2000
const SETTLE_IN_PLACE_MS = 250

/** Open the page and, when a selector is given, wait until it is on screen. */
export async function openPage(p: GuidePage, targetSelector?: string): Promise<void> {
  const wasView = cockpit.activeView
  switch (p.view) {
    case 'chat':
      if (cockpit.activeView !== 'chat') setActiveView('chat')
      break
    case 'settings':
      cockpit.settingsIntent = { section: p.rail, ...(p.head ? { head: p.head } : {}) }
      setActiveView('settings')
      break
    case 'capability':
      openCapabilityAt(p.page)
      break
    case 'office':
      if (cockpit.activeView !== 'office') setActiveView('office')
      break
    case 'artifacts':
      if (cockpit.activeView !== 'artifacts') setActiveView('artifacts')
      break
  }
  await tick()
  if (!targetSelector || typeof document === 'undefined') return
  const budget = wasView !== cockpit.activeView || p.view === 'settings' ? SETTLE_AFTER_CHANGE_MS : SETTLE_IN_PLACE_MS
  const start = Date.now()
  while (Date.now() - start < budget) {
    if (document.querySelector(targetSelector)) return
    await sleep(40)
    await tick()
  }
}
