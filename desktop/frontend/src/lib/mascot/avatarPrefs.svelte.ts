// What the assistant's mascot looks like — the user's four choices.
//
// The identity dials the rig offers (palette.ts, parts.ts): the body's finish,
// the accent hue, the light on top, the face it rests on. Everything else is
// the assistant's own (the logo on its ears, the laptop with the mark) and is
// not a choice, because those are what make it the assistant and not an
// agent — COMPANY.md's one face.
//
// Held in localStorage for now, like companionSetting.svelte.ts and for the
// same reason (config.go is mid-change elsewhere on 12 ก.ย. 2026). The identity
// layer is where this belongs; moving it is a read/write here and a Go field,
// nothing in the page or the companion changes.
import type { MascotOptions } from './rig'
import { DEFAULT_SHELL, shellOf } from './palette'
import { FACE, TOP, row } from './parts'

export type AvatarPrefs = {
  shell: string
  /** Accent hue in degrees, or null for the brand's own. */
  hue: number | null
  top: string
  face: string
}

const KEY = 'avatarPrefs'
const PERSONA_KEY = 'avatarPersonas'
/** How many personas a person may keep. Three, as the owner asked (12 ก.ย.):
 *  "บุคลิก 1 - 2 - 3" — the looks an agent designed later may be given. */
export const PERSONA_SLOTS = 3

export const DEFAULT_PREFS: AvatarPrefs = { shell: DEFAULT_SHELL, hue: null, top: 'orb', face: 'neutral' }

/** Unknown ids land on the default, never on an error — a stored preference
 *  outlives the catalogue row it named. */
function sane(p: Partial<AvatarPrefs> | null | undefined): AvatarPrefs {
  return {
    shell: shellOf(p?.shell).id,
    hue: typeof p?.hue === 'number' && Number.isFinite(p.hue) ? ((p.hue % 360) + 360) % 360 : null,
    top: row(TOP, p?.top, DEFAULT_PREFS.top).id,
    face: FACE.some((f) => f.identity && f.id === p?.face) ? (p!.face as string) : DEFAULT_PREFS.face,
  }
}

function seed(): AvatarPrefs {
  try {
    const raw = localStorage.getItem(KEY)
    return sane(raw ? (JSON.parse(raw) as Partial<AvatarPrefs>) : null)
  } catch {
    return { ...DEFAULT_PREFS }
  }
}

export const avatarPrefs = $state<AvatarPrefs>(seed())

export function setAvatarPrefs(patch: Partial<AvatarPrefs>): void {
  const next = sane({ ...avatarPrefs, ...patch })
  avatarPrefs.shell = next.shell
  avatarPrefs.hue = next.hue
  avatarPrefs.top = next.top
  avatarPrefs.face = next.face
  try {
    localStorage.setItem(KEY, JSON.stringify(next))
  } catch {
    // Not remembered, still applied for this session.
  }
}

export function resetAvatarPrefs(): void {
  setAvatarPrefs(DEFAULT_PREFS)
}

export function isDefaultPrefs(p: AvatarPrefs = avatarPrefs): boolean {
  return p.shell === DEFAULT_PREFS.shell && p.hue === null && p.top === DEFAULT_PREFS.top && p.face === DEFAULT_PREFS.face
}

/** The assistant's slots, for Mascot.svelte: the role's own plus these. */
export function assistantOptions(p: AvatarPrefs = avatarPrefs): MascotOptions & { hue?: number } {
  return { shell: p.shell, top: p.top, face: p.face, ...(p.hue === null ? {} : { hue: p.hue }) }
}

// ---- personas ---------------------------------------------------------------
// A persona is a saved set of the four choices, kept in one of three slots.
// The assistant wears one at a time; the point of keeping them is the agents a
// user will design later — a persona is a look ready to be handed to one.

export type Persona = AvatarPrefs | null

function seedPersonas(): Persona[] {
  const empty: Persona[] = Array.from({ length: PERSONA_SLOTS }, () => null)
  try {
    const raw = localStorage.getItem(PERSONA_KEY)
    if (!raw) return empty
    const list = JSON.parse(raw) as unknown
    if (!Array.isArray(list)) return empty
    return empty.map((_, i) => (list[i] && typeof list[i] === 'object' ? sane(list[i] as Partial<AvatarPrefs>) : null))
  } catch {
    return empty
  }
}

export const personas = $state<{ slots: Persona[] }>({ slots: seedPersonas() })

function keepPersonas(): void {
  try {
    localStorage.setItem(PERSONA_KEY, JSON.stringify(personas.slots))
  } catch {
    // Not remembered, still held for this session.
  }
}

/** Save what the assistant wears now into a slot. */
export function savePersona(slot: number): void {
  if (slot < 0 || slot >= PERSONA_SLOTS) return
  personas.slots[slot] = sane({ ...avatarPrefs })
  keepPersonas()
}

/** Wear a saved persona. An empty slot changes nothing. */
export function usePersona(slot: number): void {
  const p = personas.slots[slot]
  if (p) setAvatarPrefs(p)
}

export function clearPersona(slot: number): void {
  if (slot < 0 || slot >= PERSONA_SLOTS) return
  personas.slots[slot] = null
  keepPersonas()
}

/** Which slot the current look matches, or -1. */
export function wornPersona(p: AvatarPrefs = avatarPrefs): number {
  return personas.slots.findIndex((s) => s && s.shell === p.shell && s.hue === p.hue && s.top === p.top && s.face === p.face)
}
