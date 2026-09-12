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
import { DEFAULT_ACCENT, DEFAULT_SHELL, accentNearHue, accentOf, shellOf } from './palette'
import { FACE, TOP, row } from './parts'

export type AvatarPrefs = {
  shell: string
  /** An ACCENT row id (palette.ts) — the hue and how much of it. */
  accent: string
  top: string
  face: string
}

const KEY = 'avatarPrefs'
const PERSONA_KEY = 'avatarPersonas'
export const DEFAULT_PREFS: AvatarPrefs = { shell: DEFAULT_SHELL, accent: DEFAULT_ACCENT, top: 'orb', face: 'neutral' }

/** What a store may hold: today's four ids, or the accent as the hue in
 *  degrees it was before the accents had names (a build of 12 ก.ย. 2026). */
type Stored = Partial<AvatarPrefs> & { hue?: number | null }

/** Unknown ids land on the default, never on an error — a stored preference
 *  outlives the catalogue row it named. */
function sane(p: Stored | null | undefined): AvatarPrefs {
  const accent =
    typeof p?.accent === 'string' ? accentOf(p.accent)
    : typeof p?.hue === 'number' && Number.isFinite(p.hue) ? accentNearHue(p.hue)
    : accentOf(DEFAULT_ACCENT)
  return {
    shell: shellOf(p?.shell).id,
    accent: accent.id,
    top: row(TOP, p?.top, DEFAULT_PREFS.top).id,
    face: FACE.some((f) => f.identity && f.id === p?.face) ? (p!.face as string) : DEFAULT_PREFS.face,
  }
}

function seed(): AvatarPrefs {
  try {
    const raw = localStorage.getItem(KEY)
    return sane(raw ? (JSON.parse(raw) as Stored) : null)
  } catch {
    return { ...DEFAULT_PREFS }
  }
}

export const avatarPrefs = $state<AvatarPrefs>(seed())

export function setAvatarPrefs(patch: Partial<AvatarPrefs>): void {
  const next = sane({ ...avatarPrefs, ...patch })
  avatarPrefs.shell = next.shell
  avatarPrefs.accent = next.accent
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
  return p.shell === DEFAULT_PREFS.shell && p.accent === DEFAULT_PREFS.accent && p.top === DEFAULT_PREFS.top && p.face === DEFAULT_PREFS.face
}

/** The assistant's slots, for Mascot.svelte: the role's own plus these. */
export function assistantOptions(p: AvatarPrefs = avatarPrefs): MascotOptions {
  return { shell: p.shell, accent: p.accent, top: p.top, face: p.face }
}

// ---- personas ---------------------------------------------------------------
// A persona is a saved set of the four choices. The list is as long as the
// user makes it — three, then six fixed slots (12 ก.ย.), then none at all
// (13 ก.ย.: "ให้เขากด + ไปได้เรื่อยๆ"): + saves what is worn as one more,
// and removing one closes the gap rather than leaving a hole. The assistant
// wears one at a time; the point of keeping them is the agents a user will
// design later — a persona is a look ready to be handed to one.

export type Persona = AvatarPrefs

function seedPersonas(): Persona[] {
  try {
    const raw = localStorage.getItem(PERSONA_KEY)
    if (!raw) return []
    const list = JSON.parse(raw) as unknown
    if (!Array.isArray(list)) return []
    // A store from the six-slot build has nulls where a slot was empty.
    return list.filter((p) => p && typeof p === 'object').map((p) => sane(p as Stored))
  } catch {
    return []
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

/** Keep what the assistant wears now as one more persona; returns its index. */
export function addPersona(): number {
  personas.slots.push(sane({ ...avatarPrefs }))
  keepPersonas()
  return personas.slots.length - 1
}

/** Overwrite a persona with what the assistant wears now. */
export function savePersona(slot: number): void {
  if (slot < 0 || slot >= personas.slots.length) return
  personas.slots[slot] = sane({ ...avatarPrefs })
  keepPersonas()
}

/** Wear a saved persona. An empty slot changes nothing. */
export function usePersona(slot: number): void {
  const p = personas.slots[slot]
  if (p) setAvatarPrefs(p)
}

/** Drop a persona; the ones after it move up. */
export function removePersona(slot: number): void {
  if (slot < 0 || slot >= personas.slots.length) return
  personas.slots.splice(slot, 1)
  keepPersonas()
}

/** Forget every persona (tests, and a store that should start over). */
export function clearPersonas(): void {
  personas.slots.length = 0
  keepPersonas()
}

/** Which slot the current look matches, or -1. */
export function wornPersona(p: AvatarPrefs = avatarPrefs): number {
  return personas.slots.findIndex((s) => s.shell === p.shell && s.accent === p.accent && s.top === p.top && s.face === p.face)
}
