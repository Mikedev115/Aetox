// The guide's own settings — the brain it thinks with, the language it speaks,
// whether it reads aloud.
//
// Settings, not memory. The guide deliberately keeps nothing between openings
// (no conversation, no place in a walk — guideState says why), and this is the
// one thing that is not an exception to that rule but a different kind of
// thing: a choice the user made about the tool, the same as a theme. It
// persists because a setting that forgets is a setting nobody uses.
//
// Its own module so the panel, the session and the greeting all read one
// source, and so adding a knob is a line here plus a row in the panel — never
// a hunt through the component (owner, 15 ก.ย. 2026: *"ทำให้เพิ่มได้ง่ายด้วย
// ในอนาคต ทำสถาปัตยกรรมให้ดี"*).

import type { Locale } from '../i18n.svelte'

const KEY = 'guidePrefs'

export type ThinkLevel = 'low' | 'medium' | 'high'

export type GuidePrefs = {
  /** '' means "whatever the chat is using" — the ordinary case, and the one
   *  that keeps working when the user changes provider somewhere else. */
  provider: string
  model: string
  think: ThinkLevel
  /** 'auto' follows the app's language; anything else overrides it, for a
   *  person who reads the UI in one language and would rather be guided in
   *  another (owner, 15 ก.ย. 2026: "หน้าตั้งค่ามีภาษาด้วยก็ดี"). */
  lang: 'auto' | Locale
  /** Read answers aloud, through the window's one player. */
  voice: boolean
  /** Where the user dragged the figure to, or null for the corner it picks
   *  itself. A setting and not a memory, by the same test as everything else
   *  here: it is a choice about the tool, made on purpose, and a figure that
   *  wanders back to the corner every time you open it is one you move every
   *  time you open it (owner, 15 ก.ย. 2026: *"อยากให้ลากได้ครับ"*). Cleared by
   *  the panel's reset, along with the rest. */
  spot: { x: number; y: number } | null
}

const DEFAULTS: GuidePrefs = { provider: '', model: '', think: 'low', lang: 'auto', voice: false, spot: null }

function load(): GuidePrefs {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return { ...DEFAULTS }
    const v = JSON.parse(raw) as Partial<GuidePrefs>
    // Field by field, with the default standing in for anything unrecognised:
    // a stored blob from an older shape must never put a junk provider or an
    // invalid think level onto a request.
    return {
      provider: typeof v.provider === 'string' ? v.provider : DEFAULTS.provider,
      model: typeof v.model === 'string' ? v.model : DEFAULTS.model,
      think: v.think === 'medium' || v.think === 'high' || v.think === 'low' ? v.think : DEFAULTS.think,
      lang: typeof v.lang === 'string' ? (v.lang as GuidePrefs['lang']) : DEFAULTS.lang,
      voice: typeof v.voice === 'boolean' ? v.voice : DEFAULTS.voice,
      // Both numbers or nothing: half a spot is not a spot, and a NaN here
      // would put the figure somewhere no clamp can bring it back from.
      spot:
        v.spot && typeof v.spot.x === 'number' && typeof v.spot.y === 'number' && Number.isFinite(v.spot.x) && Number.isFinite(v.spot.y)
          ? { x: v.spot.x, y: v.spot.y }
          : DEFAULTS.spot,
    }
  } catch {
    return { ...DEFAULTS }
  }
}

export const guidePrefs = $state<GuidePrefs>(load())

/** Change one setting and remember it. One door, so nothing writes the store
 *  without also writing the disk. */
export function setGuidePref<K extends keyof GuidePrefs>(key: K, value: GuidePrefs[K]): void {
  guidePrefs[key] = value
  try {
    localStorage.setItem(KEY, JSON.stringify({ ...guidePrefs }))
  } catch {
    // storage unavailable — the choice holds for this run and no longer
  }
}

export function resetGuidePrefs(): void {
  for (const k of Object.keys(DEFAULTS) as (keyof GuidePrefs)[]) setGuidePref(k, DEFAULTS[k])
}

/** Is the guide running on a brain of its own, or the chat's? */
export function usesOwnBrain(): boolean {
  return guidePrefs.provider.trim() !== ''
}
