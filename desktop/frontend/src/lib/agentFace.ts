// The wardrobe an เอเจน's face is assembled from.
//
// A catalogue rather than a drawing. The first version of this lived as one
// chain of {#if hair === 4} inside AgentFace.svelte, which worked and would
// have stopped working the first time somebody wanted a ninth haircut: adding
// one meant editing a branch in the middle of a render, and offering one to a
// user meant a number with no name. Here a part is a row, adding one is
// appending to an array, and every part carries the label the picker shows —
// which is a picker that now exists, in Settings' ตัวตน section, reading these
// arrays directly so that what can be chosen and what can be drawn are the same
// list.
//
// Three rules hold this together and all of them matter more than the drawings:
//
//  1. **A part is identified by its id, never its position.** The hash picks an
//     INDEX, so reordering these arrays would repaint the whole roster — but an
//     agent that names its part in its own file names the id, and that keeps
//     working whatever the order becomes. Append rather than insert.
//  2. **A part added after the roster existed is `pickOnly`.** Appending is
//     safe for the file format and not for the derived faces: the index is
//     taken modulo the list's LENGTH, so growing a pool of eight to fourteen
//     hands a new haircut to every agent that never chose one. See DERIVED,
//     below the catalogue — the pool is frozen, the picker is not.
//  3. **Nothing here reads a file or a store.** Given a name it returns the
//     same face on every machine, forever, which is what lets the face be
//     derived instead of stored (see AgentFace.svelte).
import { coverHue } from './coverHue'

// The one palette every part is drawn from, so a new part cannot invent a
// colour that sits outside the agent's own family.
export type Palette = {
  skin: string
  skinShade: string
  shirt: string
  dark: string
  darkUp: string
  bright: string
  glass: string
}

export function palette(hue: number): Palette {
  return {
    skin: `hsl(${hue} 42% 78%)`,
    skinShade: `hsl(${hue} 36% 66%)`,
    shirt: `hsl(${hue} 38% 42%)`,
    dark: `hsl(${hue} 46% 23%)`,
    darkUp: `hsl(${hue} 48% 31%)`,
    bright: `hsl(${hue} 58% 86%)`,
    glass: `hsl(${hue} 28% 40%)`,
  }
}

// The head every hair is cut for. A part may override it — the long and curly
// cuts sit around a slightly smaller head — but only through this shape, so a
// new part cannot quietly move the eyes off the face.
export type Head = { cy: number; rx: number; ry: number }
const HEAD: Head = { cy: 26, rx: 15.5, ry: 16.5 }

export type Hair = {
  id: string
  label: string
  /** Drawn under the head. Volume that frames the face lives here. */
  behind?: (p: Palette) => string
  /** Drawn over the head. Ordinary hair lives here. */
  front?: (p: Palette) => string
  /** Ears show unless a cut covers them. */
  ears?: boolean
  head?: Head
  /** Offered in the picker, never landed on by the hash. See DERIVED below —
   *  this is what lets the catalogue grow without repainting the roster. */
  pickOnly?: boolean
}

// Append only. See rule 1 above.
export const HAIR: Hair[] = [
  {
    id: 'sidePart',
    label: 'แสกข้าง',
    front: (p) => `<path d="M16.5 25C16.5 12 24 9 32 9s15.5 3 15.5 16c0-7-4-10-12-10-5 0-8 2-11 2-4 0-6.5 3-8 8z" fill="${p.dark}"/>`,
  },
  {
    id: 'bun',
    label: 'มวย',
    front: (p) =>
      `<circle cx="32" cy="7.5" r="5" fill="${p.dark}"/>` +
      `<path d="M17 24C17 12 24 10 32 10s15 2 15 14c0-7-5-10-15-10s-15 3-15 10z" fill="${p.dark}"/>`,
  },
  {
    id: 'bob',
    label: 'บ๊อบ',
    front: (p) => `<path d="M15 30C15 12 23 8.5 32 8.5S49 12 49 30v6h-5V21c-4-4-20-4-24 0v15h-5z" fill="${p.dark}"/>`,
  },
  {
    id: 'curly',
    label: 'หยิก',
    ears: false,
    head: { cy: 27, rx: 14, ry: 15 },
    behind: (p) =>
      `<circle cx="20" cy="16" r="7.5" fill="${p.dark}"/>` +
      `<circle cx="32" cy="11" r="8.5" fill="${p.dark}"/>` +
      `<circle cx="44" cy="16" r="7.5" fill="${p.dark}"/>` +
      `<circle cx="17" cy="24" r="5.5" fill="${p.dark}"/>` +
      `<circle cx="47" cy="24" r="5.5" fill="${p.dark}"/>`,
  },
  {
    id: 'cap',
    label: 'หมวกแก๊ป',
    front: (p) =>
      `<path d="M16 21C16 10 23 6.5 32 6.5S48 10 48 21z" fill="${p.dark}"/>` +
      `<path d="M4 18h13v5H6a2.5 2.5 0 0 1 0-5z" fill="${p.darkUp}"/>` +
      `<rect x="15" y="19.5" width="34" height="4.5" rx="2.2" fill="${p.darkUp}"/>`,
  },
  {
    id: 'beanie',
    label: 'หมวกไหมพรม',
    front: (p) =>
      `<path d="M16 22C16 11 23 7 32 7s16 4 16 15z" fill="${p.dark}"/>` +
      `<rect x="15" y="20.5" width="34" height="5.5" rx="2.7" fill="${p.darkUp}"/>` +
      `<circle cx="32" cy="5" r="3.2" fill="${p.darkUp}"/>`,
  },
  {
    id: 'long',
    label: 'ผมยาว',
    ears: false,
    head: { cy: 27, rx: 14.5, ry: 15.5 },
    behind: (p) => `<path d="M14 44V26c0-11 8-17 18-17s18 6 18 17v18h-6V26c0-7-5-11-12-11s-12 4-12 11v18z" fill="${p.dark}"/>`,
  },
  {
    id: 'neat',
    label: 'เรียบ',
    front: (p) => `<path d="M17 25C17 12 25 9.5 32 9.5S47 12 47 25c-1-9-7-12-15-12s-14 3-15 12z" fill="${p.dark}"/>`,
  },
  // ── Everything below arrived after the roster did, so all of it is
  // `pickOnly`. Choosable by name, never derived — see DERIVED, below the
  // catalogue, for the whole of why.
  // Spikes drawn into the HAIRLINE rather than above the head, and that is the
  // one rule these six were re-cut under. A silhouette outside the head sits on
  // the tile, and the tile's own gradient is within three points of lightness of
  // p.dark — so a spike that pokes above the skull is invisible at any size,
  // while a jagged fringe against the forehead is legible at 20px. The parts
  // that genuinely have to live outside (a tail, beads, an afro) take p.darkUp
  // instead, which is what the cap's brim and the beanie's band already do.
  {
    id: 'spiky',
    label: 'ตั้งแหลม',
    pickOnly: true,
    front: (p) =>
      `<path d="M17 25C17 12 24 8 32 8s15 4 15 17l-3.8-5.5-3.8 5.5-3.8-5.5-3.8 5.5-3.8-5.5-3.8 5.5-3.8-5.5-3.4 5.5z" fill="${p.dark}"/>`,
  },
  {
    id: 'ponytail',
    label: 'หางม้า',
    pickOnly: true,
    behind: (p) =>
      `<path d="M44 19c8 2 13 7.5 13 16 0 8-3.5 13.5-8.5 15.5 3.5-6.5 3-15 0-20z" fill="${p.darkUp}"/>`,
    front: (p) =>
      `<path d="M16.5 25C16.5 12 24 9 32 9s15.5 3 15.5 16c-1-8-6.5-11-15.5-11s-14.5 3-15.5 11z" fill="${p.dark}"/>`,
  },
  {
    id: 'braids',
    label: 'เปีย',
    pickOnly: true,
    ears: false,
    behind: (p) =>
      `<circle cx="15.5" cy="32" r="4" fill="${p.darkUp}"/><circle cx="15.5" cy="39" r="3.5" fill="${p.darkUp}"/>` +
      `<circle cx="48.5" cy="32" r="4" fill="${p.darkUp}"/><circle cx="48.5" cy="39" r="3.5" fill="${p.darkUp}"/>`,
    front: (p) =>
      `<path d="M16.5 27C16.5 13 24 9 32 9s15.5 4 15.5 18c0-8-4.5-11-11-11-3.5 0-5 1.5-8 1.5-3.5 0-6 2.5-7 7.5z" fill="${p.dark}"/>`,
  },
  {
    id: 'afro',
    label: 'ทรงฟู',
    pickOnly: true,
    ears: false,
    head: { cy: 27, rx: 14, ry: 15 },
    behind: (p) => `<ellipse cx="32" cy="21" rx="19" ry="17" fill="${p.darkUp}"/>`,
  },
  {
    id: 'headscarf',
    label: 'ผ้าคลุมผม',
    pickOnly: true,
    ears: false,
    behind: (p) =>
      `<path d="M13 46V29c0-11 8.5-19 19-19s19 8 19 19v17h-7.5V29c0-7-5-12-11.5-12S20.5 22 20.5 29v17z" fill="${p.darkUp}"/>`,
    front: (p) =>
      `<path d="M16.5 27C16.5 15 23.5 10.5 32 10.5S47.5 15 47.5 27c-2-9-7-12-15.5-12S18.5 18 16.5 27z" fill="${p.darkUp}"/>`,
  },
  // No hair at all, and a real choice rather than the absence of one: leaving
  // the field blank means "derive it", which lands on one of the eight above.
  { id: 'bald', label: 'ไม่มีผม', pickOnly: true },
]

export type Accessory = {
  id: string
  label: string
  svg?: (p: Palette) => string
  /** As Hair.pickOnly. */
  pickOnly?: boolean
}

// `none` appears twice on purpose, and it is the same rule the card's chips are
// drawn under: a badge every card carries is a badge that says nothing. Half a
// roster wearing nothing is what makes the glasses mean something.
export const ACCESSORY: Accessory[] = [
  { id: 'none', label: 'ไม่มี' },
  {
    id: 'glasses',
    label: 'แว่น',
    svg: (p) =>
      `<circle cx="26" cy="27" r="6" fill="none" stroke="${p.glass}" stroke-width="2"/>` +
      `<circle cx="38" cy="27" r="6" fill="none" stroke="${p.glass}" stroke-width="2"/>` +
      `<path d="M31 27h2" stroke="${p.glass}" stroke-width="2"/>`,
  },
  { id: 'none2', label: 'ไม่มี' },
  {
    id: 'headphones',
    label: 'หูฟัง',
    svg: (p) =>
      `<path d="M15 26a17 17 0 0 1 34 0" fill="none" stroke="${p.darkUp}" stroke-width="3.4"/>` +
      `<rect x="10.5" y="24" width="8" height="12" rx="3.5" fill="${p.darkUp}"/>` +
      `<rect x="45.5" y="24" width="8" height="12" rx="3.5" fill="${p.darkUp}"/>`,
  },
  // Added after the roster, so `pickOnly` — same rule as the hair below `neat`.
  // These draw BEFORE the mouth (faceSVG), which is what lets a beard sit under
  // a smile instead of swallowing it.
  {
    id: 'shades',
    label: 'แว่นดำ',
    pickOnly: true,
    svg: () =>
      `<rect x="20" y="23" width="11.5" height="8" rx="2" fill="#14161a"/>` +
      `<rect x="32.5" y="23" width="11.5" height="8" rx="2" fill="#14161a"/>` +
      `<path d="M31.5 25.5h1" stroke="#14161a" stroke-width="2.4"/>`,
  },
  {
    id: 'moustache',
    label: 'หนวด',
    pickOnly: true,
    svg: (p) =>
      `<path d="M32 33c-2-2.6-6.2-2.2-7.4 1 2.6 1.7 5.4 1.1 7.4-1z" fill="${p.dark}"/>` +
      `<path d="M32 33c2-2.6 6.2-2.2 7.4 1-2.6 1.7-5.4 1.1-7.4-1z" fill="${p.dark}"/>`,
  },
  {
    id: 'beard',
    label: 'เครา',
    pickOnly: true,
    svg: (p) =>
      `<path d="M18.5 28.5c0 9.5 6 15 13.5 15s13.5-5.5 13.5-15c-1.5 6.5-6 9.5-13.5 9.5s-12-3-13.5-9.5z" fill="${p.dark}"/>`,
  },
]

/** What the accessory picker offers. Not ACCESSORY itself: `none` sits in that
 *  array twice so half a roster wears nothing (see the note above it), and a
 *  picker showing "ไม่มี" twice reads as a bug rather than as weighting. HAIR
 *  needs no such trim and is offered whole. */
export const ACCESSORY_CHOICES: Accessory[] = ACCESSORY.filter((a) => a.id !== 'none2')

// The pool the HASH may land on — and the reason it is not the whole catalogue.
//
// `pick` below chooses an INDEX, so the list's LENGTH is part of every derived
// face. Rule 1 at the top of this file says a part is identified by its id and
// that adding one is appending a row, and both halves are true; what neither
// says is that appending changes `length`, and so changes the haircut of every
// agent that never chose one. Six new cuts would have handed the whole roster
// new faces on the next launch — the seven that ship included — which is the
// promise this file is built on ("the same face on every machine, forever")
// breaking quietly, for a change that only ever meant to ADD.
//
// So the pool is frozen at what the roster was derived from, and everything
// added since is `pickOnly`: offered by name, never landed on. The pickers show
// the whole catalogue; the hash sees only these.
const derived = <T extends { pickOnly?: boolean }>(list: T[]) => list.filter((x) => !x.pickOnly)
export const HAIR_DERIVED: Hair[] = derived(HAIR)
export const ACCESSORY_DERIVED: Accessory[] = derived(ACCESSORY)

/** The hues the colour row offers: twelve even steps around the wheel, which is
 *  as fine as this palette can be told apart at 40px — the shirt and the skin
 *  sit near 40% saturation, so two hues 15° apart arrive as the same person.
 *  Choosing none leaves it to coverHue, where every agent's colour has come
 *  from since before there was a face to put it on. */
export const HUES: number[] = [0, 30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330]

// What the agent is holding, keyed by the `icon:` its profile already declares.
// Nothing new is asked of whoever writes the file, and an icon this map does not
// know draws no prop rather than a guess: a profile is written by hand by
// somebody who cannot see this list.
//
// This map is also the PICKER, through PROP_ICONS below, and that is what fixed
// the bug it was allowed to have for a month. The editor kept its own hand-typed
// list of fifteen marks to offer; this map drew seven; the overlap was four. So
// eleven of the fifteen choices changed nothing at all on the roster — you
// picked "terminal", saved, and got back the same empty-handed face — while
// `zap`, `gitBranch` and `slidersHorizontal`, worn by three of the seven agents
// shipping in the box, could not be chosen at all. Two lists that had to agree
// and nothing making them.
//
// So: a prop added here appears in the picker in the same commit, and a mark
// that cannot be drawn cannot be offered. `bot` is deliberately absent — it is
// what chairIcon() hands back for a profile that names nothing, so drawing it
// would put a robot in the hands of every agent whose owner never chose.
export const PROP: Record<string, (p: Palette) => string> = {
  search: (p) =>
    `<circle cx="50" cy="50" r="8" fill="none" stroke="${p.bright}" stroke-width="3.2"/>` +
    `<path d="M56 56l5.5 5.5" stroke="${p.bright}" stroke-width="3.2" stroke-linecap="round"/>`,
  fileText: (p) =>
    `<path d="M43 44h11l7 7v13H43z" fill="${p.bright}"/>` +
    `<path d="M54 44v7h7" fill="${p.skinShade}"/>` +
    `<path d="M47 55h11M47 60h8" stroke="${p.dark}" stroke-width="2.2" stroke-linecap="round"/>`,
  clapperboard: (p) =>
    `<rect x="40" y="47" width="22" height="15" rx="2.5" fill="${p.bright}"/>` +
    `<path d="M40 51.5h22" stroke="${p.dark}" stroke-width="2.6"/>` +
    `<path d="M44 47l3.5 4.5M50 47l3.5 4.5M56 47l3.5 4.5" stroke="${p.dark}" stroke-width="2.2"/>`,
  chartColumn: (p) =>
    `<rect x="43" y="52" width="5" height="10" rx="2" fill="${p.bright}"/>` +
    `<rect x="50.5" y="46" width="5" height="16" rx="2" fill="${p.bright}"/>` +
    `<rect x="58" y="40" width="5" height="22" rx="2" fill="${p.bright}"/>`,
  gitBranch: (p) =>
    `<circle cx="46" cy="46" r="3.8" fill="${p.bright}"/>` +
    `<circle cx="46" cy="60" r="3.8" fill="${p.bright}"/>` +
    `<circle cx="60" cy="46" r="3.8" fill="${p.bright}"/>` +
    `<path d="M46 49.8v6.4M49.8 46h6.4" stroke="${p.bright}" stroke-width="2.6"/>`,
  zap: (p) => `<path d="M53 40l-10 15h7.5l-2.5 11 11-15.5h-7.5z" fill="${p.bright}"/>`,
  slidersHorizontal: (p) =>
    `<path d="M42 50h20M42 60h20" stroke="${p.bright}" stroke-width="2.8" stroke-linecap="round"/>` +
    `<circle cx="51" cy="50" r="3.6" fill="${p.bright}"/>` +
    `<circle cx="56" cy="60" r="3.6" fill="${p.bright}"/>`,
  // The nine below are the marks the editor was already offering and the face
  // could not draw. Same box as the seven above — roughly x 40..62, y 40..64,
  // beside the shoulder — and the same two colours, so a new prop cannot invent
  // a palette of its own.
  //
  // No arc flags anywhere on purpose: every shape here is a rect, a circle, an
  // ellipse or straight lines. A knob on a puzzle piece drawn as an `a` command
  // is one wrong sweep flag away from a shape nobody notices is inside out at
  // 38px, and there is no way to see that in a diff.
  layoutList: (p) =>
    `<rect x="41" y="45" width="21" height="17" rx="2.6" fill="${p.bright}"/>` +
    `<rect x="44" y="48.5" width="5" height="4.5" rx="1.2" fill="${p.dark}"/>` +
    `<rect x="44" y="55.5" width="5" height="4.5" rx="1.2" fill="${p.dark}"/>` +
    `<path d="M52.5 50.7h6.5M52.5 57.7h6.5" stroke="${p.dark}" stroke-width="2.2" stroke-linecap="round"/>`,
  fileCode: (p) =>
    `<path d="M43 44h11l7 7v13H43z" fill="${p.bright}"/>` +
    `<path d="M54 44v7h7" fill="${p.skinShade}"/>` +
    `<path d="M50 54.5l-2.8 3.5 2.8 3.5M55 54.5l2.8 3.5-2.8 3.5" fill="none" stroke="${p.dark}" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>`,
  terminal: (p) =>
    `<rect x="40" y="45" width="22" height="17" rx="2.6" fill="${p.bright}"/>` +
    `<path d="M40 49.5h22" stroke="${p.dark}" stroke-width="2.2"/>` +
    `<path d="M44.5 53.5l3 2.6-3 2.6" fill="none" stroke="${p.dark}" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>` +
    `<path d="M51 58.7h6.5" stroke="${p.dark}" stroke-width="2.2" stroke-linecap="round"/>`,
  globe: (p) =>
    `<circle cx="51" cy="53" r="10.5" fill="${p.bright}"/>` +
    `<path d="M40.5 53h21" stroke="${p.dark}" stroke-width="2"/>` +
    `<ellipse cx="51" cy="53" rx="4.6" ry="10.5" fill="none" stroke="${p.dark}" stroke-width="2"/>`,
  brain: (p) =>
    `<circle cx="47" cy="50" r="6.2" fill="${p.bright}"/>` +
    `<circle cx="55" cy="50" r="6.2" fill="${p.bright}"/>` +
    `<rect x="45.5" y="51" width="11" height="10" rx="3.4" fill="${p.bright}"/>` +
    `<path d="M51 45.5V61" stroke="${p.dark}" stroke-width="1.9"/>` +
    `<path d="M47 50.6q2 1.6 4 0M51 55.6q2 1.6 4 0" fill="none" stroke="${p.dark}" stroke-width="1.7" stroke-linecap="round"/>`,
  palette: (p) =>
    `<ellipse cx="50.5" cy="53" rx="11" ry="10" fill="${p.bright}"/>` +
    // The thumb hole, filled with the shirt it sits on rather than cut out:
    // this prop is drawn over the body, and a real hole would show the tile.
    `<circle cx="54.5" cy="58" r="3.2" fill="${p.shirt}"/>` +
    `<circle cx="46" cy="49" r="1.9" fill="${p.dark}"/><circle cx="52" cy="47.5" r="1.9" fill="${p.dark}"/>` +
    `<circle cx="57" cy="51.5" r="1.9" fill="${p.dark}"/><circle cx="45.5" cy="56" r="1.9" fill="${p.dark}"/>`,
  package: (p) =>
    `<path d="M51 41.5l11 5.6v11.3l-11 5.6-11-5.6V47.1z" fill="${p.bright}"/>` +
    `<path d="M40 47.1l11 5.6 11-5.6" fill="none" stroke="${p.dark}" stroke-width="2" stroke-linejoin="round"/>` +
    `<path d="M51 52.7V64" stroke="${p.dark}" stroke-width="2"/>`,
  compass: (p) =>
    `<circle cx="51" cy="53" r="10.5" fill="${p.bright}"/>` +
    `<path d="M57 47l-3.2 8.2-8.2 3.2 3.2-8.2z" fill="${p.dark}"/>`,
  puzzle: (p) =>
    // The right knob sits at 59, not at the piece's own edge: 64 is the whole
    // viewBox, and a knob drawn past it comes back as a flat cut that nobody
    // reads as a puzzle piece.
    `<rect x="40" y="47" width="19" height="15" rx="2" fill="${p.bright}"/>` +
    `<circle cx="48.5" cy="47" r="3.4" fill="${p.bright}"/>` +
    `<circle cx="59" cy="55" r="3.4" fill="${p.bright}"/>`,
  // Twelve more, on the owner's ask for more to choose from. Free in a way the
  // hair was not: a prop is only ever drawn for an icon a profile NAMES, so
  // adding one cannot move anybody's derived face (see DERIVED above).
  wrench: (p) =>
    `<circle cx="56.5" cy="46.5" r="5.4" fill="none" stroke="${p.bright}" stroke-width="3.4"/>` +
    `<path d="M52.7 50.3L42.5 60.5" stroke="${p.bright}" stroke-width="4" stroke-linecap="round"/>`,
  scissors: (p) =>
    `<circle cx="45" cy="59.5" r="3.3" fill="none" stroke="${p.bright}" stroke-width="2.4"/>` +
    `<circle cx="53.5" cy="59.5" r="3.3" fill="none" stroke="${p.bright}" stroke-width="2.4"/>` +
    `<path d="M43.5 56.8L57 41M55 56.8L41.5 41" stroke="${p.bright}" stroke-width="2.6" stroke-linecap="round"/>`,
  pencil: (p) =>
    `<path d="M58 41l4 4-13.5 13.5-5.5 1.5 1.5-5.5z" fill="${p.bright}"/>` +
    `<path d="M44.5 54.5l5 5-6.5 1.5z" fill="${p.dark}"/>`,
  shield: (p) =>
    `<path d="M51 41l10 4v8c0 6-4 9.6-10 11-6-1.4-10-5-10-11v-8z" fill="${p.bright}"/>` +
    `<path d="M46.5 52.5l3.2 3.2 6-6.4" fill="none" stroke="${p.dark}" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"/>`,
  messageSquare: (p) =>
    `<rect x="40" y="44" width="22" height="15" rx="3" fill="${p.bright}"/>` +
    `<path d="M46 58.5l-1 5.5 6.5-5.5z" fill="${p.bright}"/>` +
    `<path d="M45 49h12M45 54h8" stroke="${p.dark}" stroke-width="2.2" stroke-linecap="round"/>`,
  monitor: (p) =>
    `<rect x="40" y="43" width="22" height="15" rx="2.4" fill="${p.bright}"/>` +
    `<rect x="43" y="46" width="16" height="9" rx="1.2" fill="${p.dark}"/>` +
    `<path d="M51 58v4M46 62.5h10" stroke="${p.bright}" stroke-width="2.6" stroke-linecap="round"/>`,
  image: (p) =>
    `<rect x="40" y="44" width="22" height="17" rx="2.4" fill="${p.bright}"/>` +
    `<circle cx="46.5" cy="50" r="2.4" fill="${p.dark}"/>` +
    `<path d="M41 60.5l6-7 5 5 4-3.5 5.5 5.5z" fill="${p.dark}"/>`,
  mic: (p) =>
    `<rect x="47.5" y="40" width="7" height="13" rx="3.5" fill="${p.bright}"/>` +
    `<path d="M43.5 50.5c0 4.4 3.4 7.6 7.5 7.6s7.5-3.2 7.5-7.6" fill="none" stroke="${p.bright}" stroke-width="2.6" stroke-linecap="round"/>` +
    `<path d="M51 58v5" stroke="${p.bright}" stroke-width="2.6" stroke-linecap="round"/>`,
  folder: (p) =>
    `<path d="M40 45h8l2.5 3H62v14H40z" fill="${p.bright}"/>` +
    `<path d="M40 52.5h22" stroke="${p.dark}" stroke-width="2"/>`,
  graph: (p) =>
    `<circle cx="44" cy="59" r="3.6" fill="${p.bright}"/>` +
    `<circle cx="58.5" cy="59" r="3.6" fill="${p.bright}"/>` +
    `<circle cx="51.5" cy="44" r="3.6" fill="${p.bright}"/>` +
    `<path d="M46.5 56.5l3.5-9M56 56.5L53 47.5M47.6 59h7.3" stroke="${p.bright}" stroke-width="2.2"/>`,
  clock: (p) =>
    `<circle cx="51" cy="52.5" r="10.5" fill="${p.bright}"/>` +
    `<path d="M51 45.5v7.5l5 3" fill="none" stroke="${p.dark}" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"/>`,
  keyboard: (p) =>
    `<rect x="39.5" y="46" width="23" height="14" rx="2.4" fill="${p.bright}"/>` +
    `<path d="M43 50.5h1.5M47.5 50.5H49M52 50.5h1.5M56.5 50.5H58" stroke="${p.dark}" stroke-width="2.4" stroke-linecap="round"/>` +
    `<path d="M46 56h10" stroke="${p.dark}" stroke-width="2.4" stroke-linecap="round"/>`,
}

/** The marks a face can actually hold, in the order the picker offers them.
 *  Read off PROP rather than written out again — see the note above it. */
export const PROP_ICONS: string[] = Object.keys(PROP)

// Below this the prop is four pixels of noise competing with the head, so it is
// dropped rather than shrunk. A legible 24px row is worth more than a complete
// one.
export const PROP_MIN_PX = 32

// Deliberately not coverHue. Sharing one hash would tie every cyan agent to the
// same haircut, and the two roster names that already land four degrees apart
// would arrive as the same person twice.
export function wardrobeHash(s: string): number {
  let h = 2166136261
  for (const ch of s) {
    h ^= ch.codePointAt(0)!
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

/** What a profile may say about its own face. Every field is optional and an
 *  unknown value falls back to the derived one, never to an error: these arrive
 *  from a .md written by hand by somebody who cannot see this file. */
export type FaceOverrides = {
  hue?: number
  hair?: string
  accessory?: string
  icon?: string
  /** Not a wardrobe field — see FaceState. It rides here because this is the
   *  bag AgentFace already spreads, and a second props channel for one string
   *  would be one more thing for a caller to forget. faceOf() never sets it:
   *  a profile describes a person, not a moment. */
  state?: FaceState
}

/** The face fields exactly as a profile writes them, and as every row the Go
 *  side hands the pages carries them: four optional strings, `hue` included,
 *  because that is what a `key: value` line in a .md is. */
export type FaceFields = { icon?: string; hair?: string; accessory?: string; hue?: string }

/** One row → what AgentFace takes.
 *
 *  One function because there are seven surfaces that draw an agent, and a page
 *  converting the row its own way is exactly the drift this file spent a
 *  commit removing. A hue that is blank, mistyped or out of range comes back
 *  undefined rather than as a colour: these are hand-written files, and a
 *  wrong number must land on the derived colour, never on grey. */
export function faceOf(f: FaceFields | undefined): FaceOverrides {
  const raw = (f?.hue ?? '').trim()
  const n = /^\d{1,3}$/.test(raw) ? Number(raw) : NaN
  return {
    icon: f?.icon,
    hair: f?.hair,
    accessory: f?.accessory,
    hue: Number.isFinite(n) && n <= 360 ? n % 360 : undefined,
  }
}

/** What the agent is DOING, which is the only thing here that is not the
 *  person. Hair, accessory, hue and prop answer "who is this"; this answers
 *  "what is happening", and it is the one field a caller passes per render
 *  rather than reading off a profile.
 *
 *  Empty is the face this file drew before any of this existed, unchanged to
 *  the pixel. A roster of agents nobody has hired looks exactly as it did. */
export type FaceState = '' | 'think' | 'work' | 'done' | 'err'

export type Face = {
  hue: number
  p: Palette
  hair: Hair
  accessory: Accessory
  head: Head
  ears: boolean
  prop: string
  smiles: boolean
  state: FaceState
}

// Named out of the whole catalogue, derived out of the frozen pool. Two lists
// rather than one because they answer different questions: "is this a part I
// know" is asked of everything that exists, "which part does this name get" of
// only what the roster was built on.
function pick<T extends { id: string }>(all: T[], pool: T[], id: string | undefined, index: number): T {
  if (id) {
    const named = all.find((x) => x.id === id)
    if (named) return named
  }
  return pool[index % pool.length]
}

export function resolveFace(name: string, size: number, o: FaceOverrides = {}): Face {
  const hue = o.hue ?? coverHue(name)
  const w = wardrobeHash(name)
  const hair = pick(HAIR, HAIR_DERIVED, o.hair, w)
  const accessory = pick(ACCESSORY, ACCESSORY_DERIVED, o.accessory, w >>> 3)
  const state = o.state ?? ''
  return {
    hue,
    p: palette(hue),
    hair,
    accessory,
    head: hair.head ?? HEAD,
    ears: hair.ears ?? true,
    // A working agent has its hands on the laptop, so it is not also holding a
    // magnifying glass — and it could not anyway: the prop's box (x 40..62,
    // y 40..64) is where the machine now is. The identity mark comes back the
    // moment the work ends.
    prop: state !== 'work' && size >= PROP_MIN_PX && o.icon && o.icon in PROP ? o.icon : '',
    smiles: w % 3 !== 0,
    state,
  }
}

// The same person under the screen's light. Not a brighter version of the
// palette — a COOLER one, six degrees round the wheel, because a monitor is the
// one light source in the room that is bluer than the room. Three tones only:
// what the light lands on is the jaw, the neck and the top of the shirt.
//
// Flat, with a hard edge, and that is a decision rather than a shortcut: a
// gradient or a blur at 26px is one smear of mud, while an edge stays an edge
// at any size. It is also how flat illustration has always drawn a face lit by
// a screen at night — two tones, no falloff.
function litBy(hue: number): { skin: string; shade: string; shirt: string } {
  const h = (hue + 354) % 360
  return { skin: `hsl(${h} 62% 88%)`, shade: `hsl(${h} 46% 76%)`, shirt: `hsl(${h} 44% 52%)` }
}

// The lit part of the face has to stop at the jaw, and the jaw is an ellipse.
//
// A clip is the only honest way to say that: the alternatives were an arc
// command (this file draws no arcs, for the reason written above PROP) and an
// inscribed ellipse, which cannot follow a jaw that narrows as fast as this one
// does — measured, it spills two units either side of the chin and reads as two
// lamps rather than one light.
//
// The id is derived from the head's own numbers, so two faces wearing the same
// head share it and it means the same shape in both. That is the property that
// matters: a duplicated id here can only ever resolve to geometry identical to
// the one it duplicates. A counter would be unique and would also break the
// rule at the top of this file — the same name must draw the same markup, on
// every machine, forever, including twice in one page.
function jawClip(head: Head): string {
  return `af-jaw-${head.cy}-${head.rx}-${head.ry}`.replace(/\./g, '_')
}

/** The screen, the light it throws, and the face that is looking at it.
 *
 *  Drawn big on purpose. The prop catalogue's box is beside the shoulder and
 *  its floor is PROP_MIN_PX, so a laptop drawn to that scale would be dropped
 *  at exactly the size this is for — the 26px head of a delegation card. What
 *  survives at 26px is not detail, it is SILHOUETTE: the machine has to change
 *  the shape of the whole tile or it may as well not be there.
 *
 *  Narrower than the shoulders, so the shirt shows either side. That one gap is
 *  the difference between a person holding a laptop and a head floating over a
 *  screen. */
function workScene(p: Palette): string {
  return (
    `<g class="af-glow"><ellipse cx="32" cy="43" rx="17" ry="6" fill="${p.bright}" opacity=".18"/></g>` +
    `<g class="af-kit">` +
    `<path d="M13 60l3-8h32l3 8z" fill="${p.glass}"/>` +
    `<rect x="15" y="44.5" width="34" height="14.5" rx="2.4" fill="${p.dark}"/>` +
    `<rect x="16.4" y="45.9" width="31.2" height="11.7" rx="1.6" fill="${p.darkUp}"/>` +
    `<circle cx="32" cy="51.8" r="2.4" fill="${p.bright}" opacity=".7"/>` +
    `<rect x="9" y="59.2" width="46" height="4.6" rx="1.9" fill="${p.dark}"/>` +
    `<rect x="9" y="59.2" width="46" height="1.4" rx=".7" fill="${p.bright}" opacity=".5"/>` +
    `</g>`
  )
}

/** The whole character as one markup string, so the component stays a frame and
 *  a test can assert on parts without mounting anything.
 *
 *  Three groups carry class names, and they are the whole of the animation
 *  contract with style.css: `af-rig` is the person (it leans and types),
 *  `af-pupils` is what looks around, `af-eye` is what blinks. Nothing in here
 *  moves by itself — this function returns the same markup for the same face,
 *  and CSS decides whether any of it is alive. */
export function faceSVG(f: Face): string {
  const { p, head } = f
  const working = f.state === 'work'
  const lit = litBy(f.hue)
  const jaw = jawClip(head)

  let s = `<g class="af-rig">`
  s +=
    `<path d="M3 64c0-13 12-19 29-19s29 6 29 19z" fill="${working ? lit.shirt : p.shirt}"/>` +
    `<path d="M25 45l7 8 7-8z" fill="${working ? lit.skin : p.skin}"/>` +
    `<rect x="27.5" y="35" width="9" height="11" fill="${working ? lit.shade : p.skinShade}"/>`
  if (f.ears) {
    s += `<circle cx="16.5" cy="28" r="3.4" fill="${p.skinShade}"/><circle cx="47.5" cy="28" r="3.4" fill="${p.skinShade}"/>`
  }
  if (f.hair.behind) s += f.hair.behind(p)
  s += `<ellipse cx="32" cy="${head.cy}" rx="${head.rx}" ry="${head.ry}" fill="${p.skin}"/>`
  if (working) {
    // Tilted a little: the machine sits slightly right of centre in the tile,
    // so the light reaches a shade further up that cheek. Two units of tilt is
    // invisible as a fact and is what stops the edge reading as a drawn line.
    const top = head.cy + head.ry * 0.33
    s +=
      `<clipPath id="${jaw}"><ellipse cx="32" cy="${head.cy}" rx="${head.rx}" ry="${head.ry}"/></clipPath>` +
      `<g clip-path="url(#${jaw})">` +
      `<path d="M12 ${round(top)}L52 ${round(top - 2.2)}V46H12z" fill="${lit.skin}"/></g>`
  }
  if (f.hair.front) s += f.hair.front(p)
  if (working) {
    // The brows are new, and they are the reason this face can be eager at all.
    // It had eyes and a mouth and nothing else, so the mouth was carrying every
    // emotion on its own — which is how "concentrating" came out as an O, the
    // shape every emoji set uses for *startled*. Raised and curved is the
    // difference between surprised and glad to be here.
    //
    // Drawn high, short and inboard, in the eyes' own near-black rather than
    // the hair's colour, and every one of those is a measurement rather than a
    // taste. The glasses accessory is two r=6 lenses centred on the eyes, so
    // its rim tops out at y=21 — a brow on the comfortable line sat exactly on
    // the frame and disappeared. Above 20.2 it clears the rim. The hair colour
    // was the other half of the same disappearance: a fringe is `p.dark`, and
    // a `p.dark` brow drawn over it is not drawn at all.
    s +=
      `<path d="M23.4 20.2q3.2-2.6 6.4-.4" stroke="#14161a" stroke-width="1.8" stroke-linecap="round" fill="none"/>` +
      `<path d="M34.2 19.8q3.2-2.2 6.4 .4" stroke="#14161a" stroke-width="1.8" stroke-linecap="round" fill="none"/>`
  }
  s += `<g class="af-pupils">`
  s += working
    ? `<circle class="af-eye" cx="26" cy="27.4" r="2.9" fill="#14161a"/>` +
      `<circle class="af-eye" cx="38" cy="27.4" r="2.9" fill="#14161a"/>` +
      // The catchlight, and it is a RECTANGLE. A round one is a generic
      // sparkle; a rectangular one is the screen itself, reflected — the
      // cheapest thing on this face and the one that says where the agent is
      // looking. Portrait lighting has used an eye-light from below for exactly
      // this reason: lit from underneath, the eyes go dead without it.
      `<rect x="24.6" y="26.1" width="2.6" height="1.9" rx=".4" fill="#e8feff"/>` +
      `<rect x="36.6" y="26.1" width="2.6" height="1.9" rx=".4" fill="#e8feff"/>`
    : `<circle class="af-eye" cx="26" cy="27" r="2.6" fill="#14161a"/>` +
      `<circle class="af-eye" cx="38" cy="27" r="2.6" fill="#14161a"/>`
  s += `</g>`
  if (f.accessory.svg) s += f.accessory.svg(p)
  s += working
    ? `<path d="M27.6 33.9q4.4 5.2 8.8 0z" fill="#14161a"/>`
    : f.smiles
      ? `<path d="M29 35q3 2.5 6 0" stroke="#14161a" stroke-width="1.9" fill="none" stroke-linecap="round"/>`
      : `<path d="M29 35h6" stroke="#14161a" stroke-width="1.9" stroke-linecap="round"/>`
  if (f.prop) s += PROP[f.prop](p)
  s += `</g>`
  if (working) s += workScene(p)
  return s
}

// Coordinates land in the markup, so they are trimmed rather than printed at
// float precision: `31.400000000000002` in a path is noise in every diff of
// every face for the life of the file.
function round(n: number): number {
  return Math.round(n * 100) / 100
}
