// Whether the assistant sits on the screen at all.
//
// (companionSetting, not companion: Windows cannot tell this file from
// Companion.svelte by case, and neither can the TypeScript program.)
//
// One switch, two doors: the account menu's row (CompanionSwitch.svelte) and
// the × on the companion's own hover frame. Both flip this; the companion
// mounts and unmounts on it (App.svelte). Remembered in localStorage for
// now — the same place the window keeps its other per-viewer conveniences —
// rather than in the Go config, because config.go and the settings surface
// are mid-change in another session today (12 ก.ย. 2026). Moving it into
// the config later is a one-line read here and nothing anywhere else.

const KEY = 'companionOn'

function seed(): boolean {
  try {
    return localStorage.getItem(KEY) !== 'off'
  } catch {
    return true
  }
}

export const companion = $state<{ on: boolean }>({ on: seed() })

export function setCompanionOn(on: boolean): void {
  companion.on = on
  try {
    localStorage.setItem(KEY, on ? 'on' : 'off')
  } catch {
    // Not remembered, still switched for this session.
  }
}
