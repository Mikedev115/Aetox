// The one door colour comes through.
//
// Every part of the mascot is drawn from this record and nothing else, which
// is what lets the owner's standing rule hold by construction rather than by
// review: the colour is an IDENTITY, it answers who this is, and nothing about
// what the mascot is doing may touch it (the cartoon faces this replaced were
// repainted three times before that rule was learned). A pose reaches the
// screen light, the arms,
// the top indicator and the floating card; it never reaches a hue.
//
// Two dials, both identity: the ACCENT — a hue and how much of it, which
// colours the cap, the ears, the soles and the light on the screen — and the
// SHELL, which is what the body is made of. An agent a user drops in tomorrow
// gets its hue the same instant it gets a name (coverHue), at full colour;
// the bundled assistant rests on the mark's own two tones (owner, 12 ก.ย.:
// "โลโก้ Aetox เป็นสีขาวและดำ อวตารเริ่มต้นก็ควรจะโทนประมาณนั้น") — a white
// robot with black cap, ears and soles, a white light on its screen. The
// sheet's robot is white; the owner asked for the same robot in other
// finishes, so a shell is a catalogue row here like a part is in parts.ts:
// named by id, appended never inserted, and so is an accent.

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
 *  shell, in the mascot's own hue. `accent` is how light the cap and ears are
 *  when they carry a colour, `ink` when they carry none — a light grey cap on
 *  a white body, a silver one on a black body. Light, not black: the first
 *  cut had the cap and soles near black and the owner's word for it was
 *  "ดำมืด ไม่สดใส" — the two tones are white and the dark of the screen, and
 *  the metal between them stays bright. */
export type Shell = {
  id: string
  label: string
  /** Shell saturation %, and lightness % for lit / mid / edge. */
  sat: number
  lit: number
  mid: number
  edge: number
  /** Lightness of a coloured accent (cap, ears, soles) — lit / base / deep. */
  accent: readonly [number, number, number]
  /** The same three when the accent is monochrome. */
  ink: readonly [number, number, number]
}

// Append only. The first row is the sheet's robot and the default.
export const SHELL: Shell[] = [
  { id: 'white', label: 'ขาว', sat: 20, lit: 97, mid: 90, edge: 74, accent: [66, 52, 38], ink: [80, 66, 48] },
  { id: 'pastel', label: 'พาสเทล', sat: 55, lit: 92, mid: 82, edge: 64, accent: [60, 46, 32], ink: [78, 64, 46] },
  { id: 'colour', label: 'สีเต็ม', sat: 70, lit: 66, mid: 54, edge: 38, accent: [90, 84, 70], ink: [96, 90, 78] },
  { id: 'dark', label: 'ดำ', sat: 18, lit: 34, mid: 24, edge: 13, accent: [70, 58, 44], ink: [90, 80, 64] },
  // A saturated dark body — navy, maroon, forest — with a light accent.
  { id: 'deep', label: 'เข้ม', sat: 55, lit: 42, mid: 32, edge: 18, accent: [90, 82, 68], ink: [90, 80, 64] },
]

export const DEFAULT_SHELL = 'white'

export function shellOf(id: string | undefined): Shell {
  return SHELL.find((s) => s.id === id) ?? (SHELL.find((s) => s.id === DEFAULT_SHELL) as Shell)
}

/** The accent: a hue, and how much of it. `chroma` scales every saturation
 *  the palette has and slides the cap between its `ink` and `accent`
 *  lightness — 1 is the full colour, 0 is the mark's own monochrome, and the
 *  values between are the muted metals and slates a wheel of pure hues
 *  cannot say. */
export type Accent = {
  id: string
  label: string
  hue: number
  chroma: number
}

// Append only. The first row is the logo's white and black, with the faintest
// cool cast so the greys read as clean metal, not ash; it was the default
// until 13 ก.ย. 2026, when the owner asked for the second — the Aetox blue he
// had been running on his own machine (avatarPrefs: white shell, brand
// accent, orb, neutral) — to be what a fresh install starts on. The wheel
// runs red to rose at roughly twenty degrees, then the muted ones.
export const ACCENT: Accent[] = [
  { id: 'ink', label: 'ขาวดำ', hue: 218, chroma: 0.08 },
  { id: 'brand', label: 'น้ำเงิน Aetox', hue: 218, chroma: 1 },
  { id: 'red', label: 'แดง', hue: 0, chroma: 1 },
  { id: 'coral', label: 'ส้มอิฐ', hue: 14, chroma: 1 },
  { id: 'orange', label: 'ส้ม', hue: 30, chroma: 1 },
  { id: 'amber', label: 'อำพัน', hue: 45, chroma: 1 },
  { id: 'yellow', label: 'เหลือง', hue: 60, chroma: 1 },
  { id: 'lime', label: 'เขียวมะนาว', hue: 85, chroma: 1 },
  { id: 'green', label: 'เขียว', hue: 120, chroma: 1 },
  { id: 'mint', label: 'มินต์', hue: 150, chroma: 1 },
  { id: 'teal', label: 'เขียวน้ำทะเล', hue: 170, chroma: 1 },
  { id: 'cyan', label: 'ฟ้าอมเขียว', hue: 188, chroma: 1 },
  { id: 'sky', label: 'ฟ้า', hue: 202, chroma: 1 },
  { id: 'indigo', label: 'คราม', hue: 245, chroma: 1 },
  { id: 'violet', label: 'ม่วง', hue: 268, chroma: 1 },
  { id: 'purple', label: 'ม่วงแดง', hue: 290, chroma: 1 },
  { id: 'magenta', label: 'บานเย็น', hue: 310, chroma: 1 },
  { id: 'pink', label: 'ชมพู', hue: 330, chroma: 1 },
  { id: 'rose', label: 'กุหลาบ', hue: 348, chroma: 1 },
  { id: 'slate', label: 'เทาน้ำเงิน', hue: 218, chroma: 0.3 },
  { id: 'copper', label: 'ทองแดง', hue: 22, chroma: 0.55 },
  { id: 'gold', label: 'ทอง', hue: 46, chroma: 0.8 },
  { id: 'olive', label: 'เขียวขี้ม้า', hue: 80, chroma: 0.45 },
]

export const DEFAULT_ACCENT = 'brand'

export function accentOf(id: string | undefined): Accent {
  return ACCENT.find((a) => a.id === id) ?? (ACCENT.find((a) => a.id === DEFAULT_ACCENT) as Accent)
}

/** The full-colour row nearest a hue in degrees — for a preference that was
 *  stored as a hue before the accents had names. */
export function accentNearHue(hue: number): Accent {
  const h = ((hue % 360) + 360) % 360
  let best = accentOf('brand')
  let dist = 361
  for (const a of ACCENT) {
    if (a.chroma < 1) continue
    const d = Math.min(Math.abs(a.hue - h), 360 - Math.abs(a.hue - h))
    if (d < dist) { dist = d; best = a }
  }
  return best
}

const pct = (n: number): number => Math.round(n * 10) / 10

export function palette(hue: number, shellId: string = DEFAULT_SHELL, chroma = 1): Palette {
  const sh = shellOf(shellId)
  const c = Math.min(1, Math.max(0, chroma))
  const sat = (s: number): number => pct(s * c)
  const mix = (ink: number, colour: number): number => pct(ink + (colour - ink) * c)
  const aUp = mix(sh.ink[0], sh.accent[0])
  const a = mix(sh.ink[1], sh.accent[1])
  const aDn = mix(sh.ink[2], sh.accent[2])
  return {
    shell: `hsl(${hue} ${sat(sh.sat)}% ${sh.lit}%)`,
    shellMid: `hsl(${hue} ${sat(sh.sat + 2)}% ${sh.mid}%)`,
    shellEdge: `hsl(${hue} ${sat(sh.sat + 4)}% ${sh.edge}%)`,
    shellRim: `hsl(${hue} ${sat(60)}% ${Math.min(92, sh.lit + 4)}%)`,
    primary: `hsl(${hue} ${sat(72)}% ${a}%)`,
    primaryUp: `hsl(${hue} ${sat(78)}% ${aUp}%)`,
    primaryDn: `hsl(${hue} ${sat(70)}% ${aDn}%)`,
    screen: `hsl(${hue} ${sat(40)}% 7%)`,
    screenUp: `hsl(${hue} ${sat(34)}% 14%)`,
    bezel: `hsl(${hue} ${sat(22)}% 26%)`,
    // A monochrome light is whiter than a coloured one, or it reads as grey.
    eye: `hsl(${hue} ${sat(100)}% ${mix(97, 88)}%)`,
    eyeGlow: `hsl(${hue} ${sat(100)}% ${mix(90, 72)}%)`,
    slab: `hsl(${hue} ${sat(10)}% 48%)`,
    slabUp: `hsl(${hue} ${sat(10)}% 60%)`,
    slabDn: `hsl(${hue} ${sat(14)}% 30%)`,
    joint: `hsl(${hue} ${sat(30)}% 22%)`,
    ink: `hsl(${hue} ${sat(30)}% 12%)`,
    card: `hsl(${hue} ${sat(30)}% 99%)`,
    cardLine: `hsl(${hue} ${sat(50)}% 70%)`,
  }
}
