// What the heads' mascots look like — the user's four choices, per head.
//
// The identity dials the rig offers (palette.ts, parts.ts): the body's finish,
// the accent hue, the light on top, the face it rests on. Everything else is
// the head's own template (roles.ts: the logo on the assistant's ears, the
// `>_` on the coder's, the laptop each holds) and is not a choice, because
// those are what make it a head and not an agent.
//
// Two heads since 14 ก.ย. 2026 (owner: "แยกอวตาร … ถ้าไปหน้าโค้ดก็โหลดอวตารอีกตัว
// ถ้าหน้าผู้ช่วยก็อีกตัว"). Until then one set dressed the assistant on every
// desk, and a user who had given the code desk its own model, prompt and
// think level still met the same face there — the one person in the app
// without a face of its own. A head is keyed by the desk it heads
// (`cockpit.desk`: assistant | coding), and the assistant's store keeps its
// old key, so a build from before this reads its user's look unchanged.
//
// Held in localStorage for now, like companionSetting.svelte.ts and for the
// same reason (config.go is mid-change elsewhere on 12 ก.ย. 2026). The identity
// layer is where this belongs; moving it is a read/write here and a Go field,
// nothing in the page or the companion changes.
import type { MascotOptions } from './rig'
import { ACCENT, DEFAULT_ACCENT, DEFAULT_SHELL, SHELL, accentNearHue } from './palette'
import { FACE, TOP, row } from './parts'
import { roleOf } from './roles'

export type AvatarPrefs = {
  shell: string
  /** An ACCENT row id (palette.ts) — the hue and how much of it. */
  accent: string
  top: string
  face: string
}

/** A head is the main assistant of one desk, named by that desk. */
export type HeadId = 'assistant' | 'coding'
export const HEADS: HeadId[] = ['assistant', 'coding']

/** The desk a session is at → the head that fronts it. Anything that is not
 *  the code desk is the assistant's: the two desks are the two heads. */
export function headOf(desk: string | undefined): HeadId {
  return desk === 'coding' ? 'coding' : 'assistant'
}

/** Which template (roles.ts) each head is drawn from. */
export const HEAD_ROLE: Record<HeadId, string> = { assistant: 'assistant', coding: 'code' }

const KEY: Record<HeadId, string> = { assistant: 'avatarPrefs', coding: 'avatarPrefs.coding' }
const PERSONA_KEY = 'avatarPersonas'

export const DEFAULT_PREFS: AvatarPrefs = { shell: DEFAULT_SHELL, accent: DEFAULT_ACCENT, top: 'orb', face: 'neutral' }
/** Each head's starting look: the owner's character sheet (14 ก.ย. 2026,
 *  "ค่าเริ่มต้นอ่ะ เอาสีตามนี้เลย") — both white in Aetox blue, and the coder
 *  told apart by the `code` template's own top and face: chevrons, focused. */
export const DEFAULT_HEAD_PREFS: Record<HeadId, AvatarPrefs> = {
  assistant: DEFAULT_PREFS,
  coding: { shell: 'white', accent: 'brand', top: 'chevrons', face: 'focused' },
}

/** What a store may hold: today's four ids, or the accent as the hue in
 *  degrees it was before the accents had names (a build of 12 ก.ย. 2026). */
type Stored = Partial<AvatarPrefs> & { hue?: number | null }

/** Unknown ids land on the head's default, never on an error — a stored
 *  preference outlives the catalogue row it named. */
function sane(p: Stored | null | undefined, def: AvatarPrefs = DEFAULT_PREFS): AvatarPrefs {
  const accent =
    typeof p?.accent === 'string' ? row(ACCENT, p.accent, def.accent)
    : typeof p?.hue === 'number' && Number.isFinite(p.hue) ? accentNearHue(p.hue)
    : row(ACCENT, def.accent, DEFAULT_ACCENT)
  return {
    shell: row(SHELL, p?.shell, def.shell).id,
    accent: accent.id,
    top: row(TOP, p?.top, def.top).id,
    face: FACE.some((f) => f.identity && f.id === p?.face) ? (p!.face as string) : def.face,
  }
}

function seed(head: HeadId): AvatarPrefs {
  try {
    const raw = localStorage.getItem(KEY[head])
    return sane(raw ? (JSON.parse(raw) as Stored) : null, DEFAULT_HEAD_PREFS[head])
  } catch {
    return { ...DEFAULT_HEAD_PREFS[head] }
  }
}

export const heads = $state<Record<HeadId, AvatarPrefs>>({ assistant: seed('assistant'), coding: seed('coding') })
/** The assistant head's look — the name every caller from the one-head days
 *  uses, and still the right one for anything that is not about the desk. */
export const avatarPrefs = heads.assistant

export function prefsOf(head: HeadId): AvatarPrefs {
  return heads[head]
}

export function setAvatarPrefs(patch: Partial<AvatarPrefs>, head: HeadId = 'assistant'): void {
  const cur = heads[head]
  const next = sane({ ...cur, ...patch }, DEFAULT_HEAD_PREFS[head])
  cur.shell = next.shell
  cur.accent = next.accent
  cur.top = next.top
  cur.face = next.face
  try {
    localStorage.setItem(KEY[head], JSON.stringify(next))
  } catch {
    // Not remembered, still applied for this session.
  }
}

/** Back to the head's own default; with no head named, both. */
export function resetAvatarPrefs(head?: HeadId): void {
  for (const h of head ? [head] : HEADS) setAvatarPrefs(DEFAULT_HEAD_PREFS[h], h)
}

export function isDefaultPrefs(p: AvatarPrefs = avatarPrefs, head: HeadId = 'assistant'): boolean {
  const d = DEFAULT_HEAD_PREFS[head]
  return p.shell === d.shell && p.accent === d.accent && p.top === d.top && p.face === d.face
}

/** A look's four slots, for Mascot.svelte — no template, so the caller's
 *  role (or the assistant's, Mascot's default) supplies the rest. A persona
 *  slot is drawn this way. */
export function assistantOptions(p: AvatarPrefs = avatarPrefs): MascotOptions {
  return { shell: p.shell, accent: p.accent, top: p.top, face: p.face }
}

/** A head, whole: its template's slots (the badge on the ears, what it
 *  holds) under its four chosen ones. What the chat wall, the companion
 *  and the avatar stage draw. */
export function headOptions(head: HeadId): MascotOptions {
  return { ...roleOf(HEAD_ROLE[head]).parts, ...assistantOptions(heads[head]) }
}

// ---- personas ---------------------------------------------------------------
// A persona is a saved set of the four choices. The list is as long as the
// user makes it — three, then six fixed slots (12 ก.ย.), then none at all
// (13 ก.ย.: "ให้เขากด + ไปได้เรื่อยๆ"): + saves what is worn as one more,
// and removing one closes the gap rather than leaving a hole. One list for
// both heads — a persona is a look, and either head may wear it; the point
// of keeping them is the agents a user will design later.

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

/** Keep what a head wears now as one more persona; returns its index. */
export function addPersona(head: HeadId = 'assistant'): number {
  personas.slots.push(sane({ ...heads[head] }))
  keepPersonas()
  return personas.slots.length - 1
}

/** Overwrite a persona with what a head wears now. */
export function savePersona(slot: number, head: HeadId = 'assistant'): void {
  if (slot < 0 || slot >= personas.slots.length) return
  personas.slots[slot] = sane({ ...heads[head] })
  keepPersonas()
}

/** Dress a head in a saved persona. An empty slot changes nothing. */
export function usePersona(slot: number, head: HeadId = 'assistant'): void {
  const p = personas.slots[slot]
  if (p) setAvatarPrefs(p, head)
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

/** Which slot a look matches, or -1. */
export function wornPersona(p: AvatarPrefs = avatarPrefs): number {
  return personas.slots.findIndex((s) => s.shell === p.shell && s.accent === p.accent && s.top === p.top && s.face === p.face)
}
