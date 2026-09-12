// Whether the assistant sits on the screen at all — and whether it speaks.
//
// (companionSetting, not companion: Windows cannot tell this file from
// Companion.svelte by case, and neither can the TypeScript program.)
//
// Two switches. `on` has two doors: the account menu's row
// (CompanionSwitch.svelte) and the × on the companion's own hover frame; the
// companion mounts and unmounts on it (App.svelte). `voice` has three: the
// row under it on the avatar page, the speaker on the same hover frame, and
// a click on the figure while it is talking (which stops that read, not the
// switch). Voice is on by default (owner, 12 ก.ย. 2026: "ตั้งเป็นเปิดเป็น
// ค่าเริ่มต้น") and means nothing while the companion is off — it is the
// figure that speaks, so a hidden figure is a silent one.
//
// Remembered in localStorage for now — the same place the window keeps its
// other per-viewer conveniences — rather than in the Go config, because
// config.go and the settings surface are mid-change in another session today
// (12 ก.ย. 2026). Moving them into the config later is a one-line read here
// and nothing anywhere else.

const KEY = 'companionOn'
const VOICE_KEY = 'companionVoice'

function seed(key: string): boolean {
  try {
    return localStorage.getItem(key) !== 'off'
  } catch {
    return true
  }
}

export const companion = $state<{ on: boolean; voice: boolean }>({ on: seed(KEY), voice: seed(VOICE_KEY) })

export function setCompanionOn(on: boolean): void {
  companion.on = on
  remember(KEY, on)
}

export function setCompanionVoice(on: boolean): void {
  companion.voice = on
  remember(VOICE_KEY, on)
}

function remember(key: string, on: boolean): void {
  try {
    localStorage.setItem(key, on ? 'on' : 'off')
  } catch {
    // Not remembered, still switched for this session.
  }
}
