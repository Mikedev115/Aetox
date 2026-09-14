// Page navigation for the guide: reaches any page or section where a data-guide
// element lives, using the app's existing view and overlay setters. No new
// router, no restructuring of App.svelte.
import { tick } from 'svelte'
import { cockpit, setActiveView, openSettingsAt, openCapabilityAt } from '../stores/cockpit.svelte'
import type { GuidePage } from './map'

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** Navigate to the page described by GuidePage and wait until the DOM is ready.
 *  Times out after 2000ms if target selector is supplied and not found. */
export async function openPage(p: GuidePage, targetSelector?: string): Promise<void> {
  if (p.view === 'chat') {
    if (cockpit.activeView !== 'chat') {
      setActiveView('chat')
    }
  } else if (p.view === 'settings') {
    openSettingsAt(p.rail)
  } else if (p.view === 'capability') {
    openCapabilityAt(p.page)
  } else if (p.view === 'office') {
    if (targetSelector && targetSelector.includes('team.')) {
      openSettingsAt('teams')
    } else if (targetSelector && targetSelector.includes('office.new_agent_btn')) {
      openSettingsAt('agents')
    } else {
      setActiveView('office')
    }
  } else if (p.view === 'artifacts') {
    if (targetSelector && targetSelector.includes('studio.')) {
      openSettingsAt('studio')
    } else {
      setActiveView('artifacts')
    }
  }

  await tick()

  if (!targetSelector) return

  const start = Date.now()
  while (Date.now() - start < 2000) {
    if (typeof document !== 'undefined' && document.querySelector(targetSelector)) {
      return
    }
    await sleep(40)
    await tick()
  }
}
