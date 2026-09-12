// The one door colour comes through.
//
// Every part of the mascot is drawn from this record and nothing else, which
// is what lets the owner's standing rule hold by construction rather than by
// review: the colour is an IDENTITY, it answers who this is, and nothing about
// what the mascot is doing may touch it (agentFace.ts tells the story of the
// three repaints that taught that). A pose reaches the screen light, the arms,
// the top indicator and the floating card; it never reaches a hue.
//
// Two dials, both identity: the HUE, which colours the cap, the ears, the
// soles and the light on the screen — an agent a user drops in tomorrow gets
// its hue the same instant it gets a name (coverHue), the bundled assistant is
// the brand's 218 — and the SHELL, which is what the body is made of. The
// sheet's robot is white; the owner asked (12 ก.ย.) for the same robot in
// other finishes, so a shell is a catalogue row here like a part is in
// parts.ts: named by id, appended never inserted, and the accent stays the
// hue's own.

export type Palette = {
  /** Head shell, lit side to terminator. */
  shell: string
  shellMid: string
  shellEdge: string
  /** Rim light on the shadow side — a cool bounce, not a second light. */
  shellRim: string
  /** Cap, ear rings, soles, cards' ink. */
  primary: string
  primaryUp: string
  primaryDn: string
  /** The OLED face screen and its bezel. */
  screen: string
  screenUp: string
  bezel: string
  /** Light ON the screen — never a pupil. */
  eye: string
  eyeGlow: string
  /** The laptop. */
  slab: string
  slabUp: string
  slabDn: string
  /** Joints, ground shadow. */
  joint: string
  ink: string
  /** Floating cards. */
  card: string
  cardLine: string
}

/** What the body is made of: saturation and the three lightnesses of the
 *  shell, in the mascot's own hue. `accent` shifts the cap and ears when the
 *  shell itself carries the hue, so they still read against it. */
export type Shell = {
  id: string
  label: string
  /** Shell saturation %, and lightness % for lit / mid / edge. */
  sat: number
  lit: number
  mid: number
  edge: number
  /** Lightness of the accent (cap, ears, soles) — lit / base / deep. */
  accent: readonly [number, number, number]
}

// Append only. The first row is the sheet's robot and the default.
export const SHELL: Shell[] = [
  { id: 'white', label: 'ขาว', sat: 20, lit: 97, mid: 90, edge: 74, accent: [66, 52, 38] },
  { id: 'pastel', label: 'พาสเทล', sat: 55, lit: 92, mid: 82, edge: 64, accent: [60, 46, 32] },
  { id: 'colour', label: 'สีเต็ม', sat: 70, lit: 66, mid: 54, edge: 38, accent: [90, 84, 70] },
  { id: 'dark', label: 'ดำ', sat: 18, lit: 34, mid: 24, edge: 13, accent: [70, 58, 44] },
]

export const DEFAULT_SHELL = 'white'

export function shellOf(id: string | undefined): Shell {
  return SHELL.find((s) => s.id === id) ?? (SHELL.find((s) => s.id === DEFAULT_SHELL) as Shell)
}

export function palette(hue: number, shellId: string = DEFAULT_SHELL): Palette {
  const sh = shellOf(shellId)
  const [aUp, a, aDn] = sh.accent
  return {
    shell: `hsl(${hue} ${sh.sat}% ${sh.lit}%)`,
    shellMid: `hsl(${hue} ${sh.sat + 2}% ${sh.mid}%)`,
    shellEdge: `hsl(${hue} ${sh.sat + 4}% ${sh.edge}%)`,
    shellRim: `hsl(${hue} 60% ${Math.min(92, sh.lit + 4)}%)`,
    primary: `hsl(${hue} 72% ${a}%)`,
    primaryUp: `hsl(${hue} 78% ${aUp}%)`,
    primaryDn: `hsl(${hue} 70% ${aDn}%)`,
    screen: `hsl(${hue} 40% 7%)`,
    screenUp: `hsl(${hue} 34% 14%)`,
    bezel: `hsl(${hue} 22% 26%)`,
    eye: `hsl(${hue} 100% 88%)`,
    eyeGlow: `hsl(${hue} 100% 72%)`,
    slab: `hsl(${hue} 10% 48%)`,
    slabUp: `hsl(${hue} 10% 60%)`,
    slabDn: `hsl(${hue} 14% 30%)`,
    joint: `hsl(${hue} 30% 22%)`,
    ink: `hsl(${hue} 30% 12%)`,
    card: `hsl(${hue} 30% 99%)`,
    cardLine: `hsl(${hue} 50% 70%)`,
  }
}
