// Whether the assistant sits on the screen at all — whether it speaks — and
// whether it says hello.
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
// figure that speaks, so a hidden figure is a silent one. `greet` sits under
// `voice`: whether the room's greeting is among what it says out loud (owner,
// 12 ก.ย.: "ตั้งค่าอีกชั้นนึงว่าจะให้พูดทักทายด้วยไหม") — on by default, moot
// while voice is off.
//
// `place` is where the figure is drawn (owner, 13 ก.ย. 2026: "ทำเป็นตัวเลือก …
// ว่าจะแสดงในแอปหรือทั่วเดสก์ท็อป"): 'window' is this window's DOM, the
// default; 'desktop' is a small window of its own on the desktop, drawn by
// Go from baked frames (desktopBody.svelte.ts) — it can be dragged to any
// monitor and stays when the app is minimised, for a few MB more and a
// first bake of a few seconds. The brain (Companion.svelte) is the same
// either way.
//
// Remembered in localStorage for now — the same place the window keeps its
// other per-viewer conveniences — rather than in the Go config, because
// config.go and the settings surface are mid-change in another session today
// (12 ก.ย. 2026). Moving them into the config later is a one-line read here
// and nothing anywhere else.

const KEY = 'companionOn'
const VOICE_KEY = 'companionVoice'
const GREET_KEY = 'companionGreet'
const PLACE_KEY = 'companionPlace'
const SIZE_KEY = 'companionSize'

/** How big the figure is drawn, in logical px — the same number in the
 *  window and on the desktop. Set by dragging the corner of its hover frame
 *  (owner, 13 ก.ย. 2026: "ขยายใหญ่และเล็กลงได้ … เอาเพดานสูงสุดด้วย อย่าลืม
 *  เพดานเล็กสุด"). The floor keeps the face readable; the ceiling keeps it a
 *  figure on the screen, not a screen. */
export const SIZE_DEFAULT = 104
export const SIZE_MIN = 64
export const SIZE_MAX = 240

export function clampSize(n: number): number {
  if (!Number.isFinite(n)) return SIZE_DEFAULT
  return Math.round(Math.max(SIZE_MIN, Math.min(SIZE_MAX, n)))
}

/** The bubble's type and width at a figure size — the same rule the desktop
 *  body draws by (companion_draw.go bubbleMetrics). At SIZE_DEFAULT the
 *  bubble is what it always was: 12px on a card at most 300px wide. Bigger,
 *  the type grows at three quarters of the figure's rate (owner, 13 ก.ย. 2026:
 *  "ให้มันขยายตาม") while the card's width is held back — a long line folds
 *  into more lines rather than stretching across the screen ("จำกัดความยาว
 *  ให้มันสูงขึ้นแทน"). Padding, radius and the gap to the figure are ems of
 *  the type, so the card keeps its proportions. */
export const BUBBLE_FONT = 12
export const BUBBLE_MAX_W = 300
const BUBBLE_FONT_RATE = 0.75
const BUBBLE_FONT_MIN = 11
const BUBBLE_FONT_MAX = 24
const BUBBLE_W_RATE = 0.6
const BUBBLE_W_CAP = 380

export function bubbleMetrics(size: number): { font: number; maxW: number } {
  const k = clampSize(size) / SIZE_DEFAULT
  const font = Math.min(BUBBLE_FONT_MAX, Math.max(BUBBLE_FONT_MIN, BUBBLE_FONT * (1 + BUBBLE_FONT_RATE * (k - 1))))
  const maxW = Math.min(BUBBLE_W_CAP, Math.max(BUBBLE_MAX_W, BUBBLE_MAX_W + BUBBLE_W_RATE * (clampSize(size) - SIZE_DEFAULT)))
  return { font: Math.round(font * 10) / 10, maxW: Math.round(maxW) }
}

function seedSize(): number {
  try {
    const raw = localStorage.getItem(SIZE_KEY)
    return raw ? clampSize(Number(raw)) : SIZE_DEFAULT
  } catch {
    return SIZE_DEFAULT
  }
}

export type CompanionPlace = 'window' | 'desktop'

function seedPlace(): CompanionPlace {
  try {
    return localStorage.getItem(PLACE_KEY) === 'desktop' ? 'desktop' : 'window'
  } catch {
    return 'window'
  }
}

function seed(key: string): boolean {
  try {
    return localStorage.getItem(key) !== 'off'
  } catch {
    return true
  }
}

export const companion = $state<{ on: boolean; voice: boolean; greet: boolean; place: CompanionPlace; size: number }>({
  on: seed(KEY),
  voice: seed(VOICE_KEY),
  greet: seed(GREET_KEY),
  place: seedPlace(),
  size: seedSize(),
})

export function setCompanionSize(size: number): void {
  companion.size = clampSize(size)
  try {
    localStorage.setItem(SIZE_KEY, String(companion.size))
  } catch {
    // Not remembered, still this size for the session.
  }
}

export function setCompanionPlace(place: CompanionPlace): void {
  companion.place = place
  try {
    localStorage.setItem(PLACE_KEY, place)
  } catch {
    // Not remembered, still switched for this session.
  }
}

export function setCompanionOn(on: boolean): void {
  companion.on = on
  remember(KEY, on)
}

export function setCompanionVoice(on: boolean): void {
  companion.voice = on
  remember(VOICE_KEY, on)
}

export function setCompanionGreet(on: boolean): void {
  companion.greet = on
  remember(GREET_KEY, on)
}

function remember(key: string, on: boolean): void {
  try {
    localStorage.setItem(key, on ? 'on' : 'off')
  } catch {
    // Not remembered, still switched for this session.
  }
}
