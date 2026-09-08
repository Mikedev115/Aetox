// Text-only size scale. Multiplies every --fs-* step in styles/type.css by one
// factor, written to the root element so the whole app follows.
//
// Deliberately separate from systemZoom (systemFont.svelte.ts), which scales
// the entire rendered UI — text, padding, icon boxes — as one unit. The two
// answer different complaints: "the letters are too small to read" wants this,
// "everything is too small on this monitor" wants zoom. They compose, so the
// px readout the zoom control shows has to account for this factor too.

// `as const` so labelKey stays a literal and t() type-checks it as a real
// message key — same shape as UI_FONTS in uiFont.svelte.ts.
// The whole ladder moved down one rung on 8 ก.ย. 2026. What the owner had
// actually been running for weeks was แน่น (0.92) — i.e. the app shipped a
// step larger than the size its own author reads it at, and every fresh
// install started too big. So the step he was on became มาตรฐาน, and แน่น went
// below it; สบายตา and ใหญ่ followed rather than staying put, because a ladder
// with a 17% gap in the middle and 9% either side is not a ladder.
// initTypeScale migrates a name stored before this change to the step that
// carries the same factor, so nobody's app resizes itself overnight.
export const TYPE_SCALES = [
  { value: 'compact', labelKey: 'settings.typeScaleCompact', scale: 0.84 },
  { value: 'default', labelKey: 'settings.typeScaleDefault', scale: 0.92 },
  { value: 'comfortable', labelKey: 'settings.typeScaleComfortable', scale: 1 },
  { value: 'large', labelKey: 'settings.typeScaleLarge', scale: 1.1 },
] as const

export type TypeScaleName = (typeof TYPE_SCALES)[number]['value']

const STORAGE_KEY = 'aetox-type-scale'
// Bumped when the ladder itself moves. Absent means "written before the move",
// which is the only way to tell a stored `compact` that meant 0.92 from one a
// user picked knowing it means 0.84.
const LADDER_KEY = 'aetox-type-scale-ladder'
const LADDER = '2'
// Old name -> the step that now carries the factor it used to. ใหญ่ has no
// exact heir (1.18 is off the top of the new ladder) and keeps the top rung.
const MOVED: Record<string, TypeScaleName> = {
  compact: 'default',
  default: 'comfortable',
  comfortable: 'large',
  large: 'large',
}
const DEFAULT_NAME: TypeScaleName = 'default'
const BY_NAME = new Map(TYPE_SCALES.map((s) => [s.value, s]))

/** The factor a machine that has never touched this control renders at.
 *  systemFont.svelte.ts divides it back out of the px readout, so the number
 *  Settings shows on a fresh install is the one the owner tuned. */
export const DEFAULT_TYPE_SCALE = BY_NAME.get(DEFAULT_NAME)!.scale

export const typeScale = $state<{ name: TypeScaleName; scale: number }>({
  name: DEFAULT_NAME,
  scale: DEFAULT_TYPE_SCALE,
})

export function applyTypeScale(name: TypeScaleName): void {
  const step = BY_NAME.get(name) ?? BY_NAME.get(DEFAULT_NAME)!
  typeScale.name = step.value
  typeScale.scale = step.scale
  document.documentElement.style.setProperty('--fs-scale', String(step.scale))
  try {
    localStorage.setItem(STORAGE_KEY, step.value)
  } catch {
    // storage unavailable — the scale still applies for this run
  }
}

/** Call once before mount so text doesn't reflow one frame after paint. */
export function initTypeScale(): void {
  let saved: string | null = null
  let ladder: string | null = null
  try {
    saved = localStorage.getItem(STORAGE_KEY)
    ladder = localStorage.getItem(LADDER_KEY)
  } catch {
    // storage unavailable — fall through to the default
  }
  // A pick made against the old ladder is honoured as a SIZE, not as a name:
  // somebody who chose 0.92 gets 0.92, whatever the step is called now.
  if (saved && ladder !== LADDER) saved = MOVED[saved] ?? saved
  applyTypeScale(BY_NAME.has(saved as TypeScaleName) ? (saved as TypeScaleName) : DEFAULT_NAME)
  try {
    localStorage.setItem(LADDER_KEY, LADDER)
  } catch {
    // storage unavailable — the migration just runs again next time
  }
}
