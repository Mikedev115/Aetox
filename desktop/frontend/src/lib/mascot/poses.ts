// What the mascot is doing — one row per pose, data rather than classes.
//
// The owner's Character & Behavior Sheet names thirteen (Greeting … Recharge);
// the rest were asked for on 12 ก.ย. 2026 — searching documents, data and
// files, answering, listening, walking. A pose reaches exactly four things:
// the light on the face screen, where each hand is, what the hands hold, and
// the card floating beside the head. It never reaches a colour or a part that
// says who this is (see palette.ts).
//
// Hands are targets in the front view — (x, y, depth), depth being how far in
// front of the body the hand is (the laptop's lid is at 11). The arm is ONE
// capsule of fixed length on a ball shoulder (rig.ts, mascot.css): it turns
// towards the target and the hand sits at its end, so a target says which way
// the arm points and nothing else. That was the owner's call, twice over —
// first against a stretchy arm that reached any target and went through the
// laptop and the head, then against the elbow that replaced it: one joint is
// the cute one.

export type Hand = readonly [x: number, y: number, depth: number]

export type Pose = {
  /** FACE row id. null = the role's own identity face (neutral / focused). */
  face: string | null
  hands: { L: Hand; R: Hand }
  /** Feet tucked under a sitting body, standing on legs, or folded up
   *  under it — the soles pulled in against the body, a robot powered down. */
  legs: 'tuck' | 'stand' | 'fold'
  /** PROP row id · 'role' = the role's laptop · null = empty hands. */
  prop: string | null
  /** PANEL row id beside the head. */
  panel: string | null
  /** PANEL row id drawn on the ground under the body. */
  ground?: string
  /** MARK id. */
  mark?: string
  /** Where the head rests when nothing is being looked at, in degrees. */
  turn: number
  /** How far mouse-follow may swing the head around `turn`. A raised hand
   *  sits beside an ear; letting the head swing that ear round to meet it
   *  is what made "the hand goes through the ear". Default ±75. */
  look?: number
}

/** Shoulders sit on the body's edge. */
export const SHOULDER = { L: [23.5, 45] as const, R: [40.5, 45] as const }
/** Shoulder to wrist. The hand's sphere sits just past it. Short, as the
 *  sheet draws it: the hands rest on the lid's lower sides, not far from the
 *  body. */
export const ARM = 8

/** Holding the laptop by the lower sides of its lid, as the sheet draws it (depth 11). */
const REST = { L: [21.5, 52.5, 11] as const, R: [42.5, 52.5, 11] as const }
/** Typing: on the base at the lid's edges, where the fingers still show
 *  from the front — the keys themselves are behind the lid. */
const KEYS = { L: [22, 54, 10] as const, R: [42, 54, 10] as const }
/** Raised beside the head: the arm points at the ear's lower edge (the ears
 *  span y 22–36) and the hand lands at cheek height in front of it — a
 *  short-armed robot does not wave above its own head. */
const UP_R: Hand = [51.5, 36.5, 5]
const UP_L: Hand = [12.5, 36.5, 5]

const POSE_ROWS = {
  greeting:    { face: 'happy',    hands: { L: REST.L, R: UP_R },               legs: 'tuck',  prop: 'role', panel: 'hi',        turn: 12,  look: 25 },
  idle:        { face: null,       hands: REST,                                 legs: 'tuck',  prop: 'role', panel: null,        turn: 14 },
  thinking:    { face: 'thinking', hands: { L: REST.L, R: [44, 41, 11] },       legs: 'tuck',  prop: 'role', panel: 'dots',      turn: -12 },
  typing:      { face: 'focused',  hands: KEYS,                                 legs: 'tuck',  prop: 'role', panel: null,        mark: 'flick', turn: 14 },
  reading:     { face: 'thinking', hands: { L: [24, 52, 8], R: [40, 52, 8] },   legs: 'tuck',  prop: 'doc',  panel: null,        turn: 6 },
  research:    { face: null,       hands: REST,                                 legs: 'tuck',  prop: 'role', panel: 'web',       turn: 10 },
  searchDocs:  { face: 'curious',  hands: { L: [24, 52, 8], R: [40, 52, 8] },   legs: 'tuck',  prop: 'doc',  panel: 'docsearch', turn: 8 },
  searchData:  { face: 'focused',  hands: KEYS,                                 legs: 'tuck',  prop: 'role', panel: 'search',    turn: -6 },
  searchFiles: { face: null,       hands: REST,                                 legs: 'tuck',  prop: 'role', panel: 'files',     turn: 10 },
  answering:   { face: 'happy',    hands: { L: REST.L, R: [51, 42, 8] },        legs: 'tuck',  prop: 'role', panel: 'answer',    turn: 10,  look: 30 },
  asking:      { face: 'curious',  hands: { L: REST.L, R: [44, 41, 11] },       legs: 'tuck',  prop: 'role', panel: 'answer',    turn: -6 },
  planning:    { face: 'happy',    hands: REST,                                 legs: 'tuck',  prop: 'role', panel: 'plan',      turn: -6 },
  coding:      { face: 'focused',  hands: KEYS,                                 legs: 'tuck',  prop: 'role', panel: 'code',      mark: 'flick', turn: 6 },
  debugging:   { face: 'curious',  hands: { L: REST.L, R: UP_R },               legs: 'tuck',  prop: 'role', panel: 'alert',     turn: 10,  look: 25 },
  presenting:  { face: 'happy',    hands: { L: REST.L, R: UP_R },               legs: 'tuck',  prop: 'role', panel: 'chart',     turn: 10,  look: 25 },
  helping:     { face: 'happy',    hands: { L: REST.L, R: [51, 42, 8] },        legs: 'tuck',  prop: 'role', panel: 'heart',     turn: 8,   look: 30 },
  success:     { face: 'excited',  hands: { L: UP_L, R: UP_R },                 legs: 'tuck',  prop: 'role', panel: 'done',      mark: 'sparkle', turn: 0, look: 30 },
  // Asleep on its charger with everything folded in (owner, 12 ก.ย.: "พับแขน
  // ขากลับ แล้วหลับ"): the arms cross over the belly, the feet fold up under.
  recharge:    { face: 'dim',      hands: { L: [27.5, 51, 5], R: [36.5, 51, 5] }, legs: 'fold', prop: null,   panel: 'zzz',       ground: 'charger', turn: 18 },
  walk:        { face: null,       hands: { L: [20, 55, 0], R: [44, 55, 0] },   legs: 'stand', prop: null,   panel: null,        turn: 70 },
  listening:   { face: null,       hands: REST,                                 legs: 'tuck',  prop: 'role', panel: 'mic',       turn: -8 },
  // Reactions to being clicked (Companion.svelte) — a moment each, no words.
  cheer:       { face: 'excited',  hands: { L: UP_L, R: UP_R },                 legs: 'tuck',  prop: 'role', panel: null,        mark: 'sparkle', turn: 0, look: 30 },
  wink:        { face: 'wink',     hands: { L: REST.L, R: UP_R },               legs: 'tuck',  prop: 'role', panel: null,        turn: 12,  look: 25 },
  // An agent whose job failed (the cartoon face's `err` ring, 12 ก.ย.): the
  // alert card of `debugging`, but no hand up — a hand on the chin, looking at
  // what went wrong, not waving about it. Distinct from `debugging` on purpose:
  // that is the assistant chasing a bug, this is a worker reporting one.
  error:       { face: 'curious',  hands: { L: REST.L, R: [44, 41, 11] },       legs: 'tuck',  prop: 'role', panel: 'alert',     turn: -8 },
} satisfies Record<string, Pose>

export type PoseId = keyof typeof POSE_ROWS

/** The table, with every row widened to the full Pose shape so a caller may
 *  read an optional field off any row. `satisfies` above keeps the key union. */
export const POSE: Record<PoseId, Pose> = POSE_ROWS

export const DEFAULT_LOOK = 75
