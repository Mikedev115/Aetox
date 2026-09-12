// The body, and the one function that draws it.
//
// This is the fixed anatomy of the blueprint — cap, head shell, face screen,
// body connector, two arms with hands, two legs with feet — assembled around
// the slots parts.ts fills and the targets poses.ts sets. It emits markup
// ONCE per (identity, pose); everything that then moves — the head turning,
// breathing, blinking, a hand waving, legs walking — is CSS in mascot.css
// keyed off the classes set here and one custom property, --t.
//
// Two ideas hold the drawing together:
//
//  • Every part mounted on the head declares its azimuth φ on the sphere
//    (face 0°, ears ∓90°, grille 180°). Its screen x is 32 + R·sin(φ + t), its
//    visible width |cos(φ + t)|, and it is in front when cos(φ + t) > 0. The
//    CSS does that arithmetic from --t, so turning costs no JavaScript and a
//    part added later needs only to say where on the head it sits.
//
//  • Held objects and hands have DEPTH in front of the body, and project as
//    x' = 32 + (x − 32)·cos t + depth·sin t. The laptop is three planes at
//    three depths, so from the side it shows its thickness; the hands sit at
//    the depth of what they hold, so they move with it instead of through it.
//
// Draw order is depth order, and CSS cannot reorder the DOM — so the parts
// that cross from behind the body to in front of it as the head turns (the
// ears, the arms) are drawn twice, a `far` copy under the body and a `near`
// copy over it, and the copies swap by opacity on the sign of sin t.
import { accentOf, palette, shellOf, type Palette } from './palette'
import { FACE, MARK, PANEL, PROP, TOP, isBadge, icon, row, type BadgeId, type Face, type Part } from './parts'
import { ARM, DEFAULT_LOOK, POSE, SHOULDER, type Hand, type Pose, type PoseId } from './poses'

/** What a caller may say about the mascot. Everything is optional; the empty
 *  object is the assistant at rest. */
export type MascotOptions = {
  /** A hue in degrees is a colour at full chroma — an agent's, off its name.
   *  Given, it wins over `accent`. */
  hue?: number
  /** An ACCENT row id (palette.ts): the hue and how much of it. Neither this
   *  nor `hue` is the default row, the mark's own white and black. */
  accent?: string
  /** What the body is made of — a SHELL row id (palette.ts). */
  shell?: string
  top?: string
  /** Ear badge: an ICONS id or 'logo'. `badgeR` defaults to `badge`. */
  badge?: string
  badgeR?: string
  /** Identity face — the FACE row this role rests on. */
  face?: string
  /** The role's laptop (or any PROP id a pose with prop:'role' should hold). */
  prop?: string
  pose?: PoseId | string
  /** Pixels the mascot will be drawn at; below 48 the drawing drops detail. */
  size?: number
}

export type Mascot = {
  hue: number
  /** 0 monochrome … 1 full colour (palette.ts Accent). */
  chroma: number
  shell: string
  p: Palette
  pose: Pose
  poseId: PoseId
  top: Part
  badgeL: BadgeId
  badgeR: BadgeId
  face: Face
  prop: Part | null
  panel: Part | null
  ground: Part | null
  mark: ((p: Palette) => string) | null
  /** Highlights, far copies and the back grille exist only at this size. */
  detail: boolean
  /** Real light: a blur filter behind the screen light and the orb. One big
   *  mascot may afford it; a roster of tiles may not. */
  glow: boolean
}

/** The size under which the drawing drops what a tile cannot show anyway. */
export const DETAIL_MIN_PX = 48
/** The size from which the screen light gets a real glow filter. High on
 *  purpose: a blur under a moving group is recomputed every frame, so the
 *  companion (104px, swaying) makes do with the gradient halo and only a
 *  big still-ish preview pays for the filter. */
export const GLOW_MIN_PX = 160

/** The brand's own hue — the blue of the UI's accent. The assistant's default
 *  look is the ACCENT row 'ink', which carries this hue at no chroma. */
export const BRAND_HUE = 218

export function resolveMascot(o: MascotOptions = {}): Mascot {
  const acc = accentOf(o.accent)
  const hue = o.hue ?? acc.hue
  const chroma = o.hue === undefined ? acc.chroma : 1
  const poseId: PoseId = o.pose && o.pose in POSE ? (o.pose as PoseId) : 'idle'
  const pose = POSE[poseId]
  const propId = pose.prop === 'role' ? (o.prop ?? 'laptopA') : pose.prop
  const badgeL: BadgeId = isBadge(o.badge) ? o.badge : 'logo'
  const shell = shellOf(o.shell).id
  return {
    hue,
    chroma,
    shell,
    p: palette(hue, shell, chroma),
    pose,
    poseId,
    top: row(TOP, o.top, 'orb'),
    badgeL,
    badgeR: isBadge(o.badgeR) ? o.badgeR : badgeL,
    face: row(FACE, pose.face ?? o.face, 'neutral'),
    prop: propId ? row(PROP, propId, 'laptopA') : null,
    panel: pose.panel ? row(PANEL, pose.panel, 'dots') : null,
    ground: pose.ground ? row(PANEL, pose.ground, 'charger') : null,
    mark: pose.mark ? (MARK[pose.mark] ?? null) : null,
    detail: (o.size ?? 38) >= DETAIL_MIN_PX,
    glow: (o.size ?? 38) >= GLOW_MIN_PX,
  }
}

/** The pose's hand targets as custom properties for the root element — the
 *  arms read them from there (mascot.css), and the root is what transitions
 *  them when the pose changes. Every renderer of the markup sets these
 *  beside --t: Mascot.svelte, the lab sheet, the companion page. */
export function handVars(m: Pick<Mascot, 'pose'>): string {
  const { L, R } = m.pose.hands
  return `--lhx:${L[0]};--lhy:${L[1]};--ld:${L[2]};--rhx:${R[0]};--rhy:${R[1]};--rd:${R[2]}`
}

/** The look range presence may use around the pose's own turn, in degrees. */
export function lookRange(pose: Pose): number {
  return pose.look ?? DEFAULT_LOOK
}

let uid = 0

export function mascotSVG(m: Mascot): string {
  const { p, detail } = m
  // Gradient ids must be unique per instance on the page, or every mascot
  // takes the first one's colours.
  const g = `ms${++uid}`
  const hl = (d: string, w = 0.9, o = 0.85): string =>
    detail ? `<path d="${d}" fill="none" stroke="#fff" stroke-width="${w}" opacity="${o}" stroke-linecap="round"/>` : ``

  let s =
    `<defs>` +
    // The shell: a specular hot-spot top-left, the lit colour, the terminator, and
    // a darker edge — one light, from the same corner every highlight below assumes.
    `<radialGradient id="${g}sh" cx=".34" cy=".26" r=".8"><stop offset="0" stop-color="#fff"/><stop offset=".18" stop-color="${p.shell}"/><stop offset=".55" stop-color="${p.shellMid}"/><stop offset=".86" stop-color="${p.shellEdge}"/><stop offset="1" stop-color="${p.joint}" stop-opacity=".55"/></radialGradient>` +
    // Soft things: a specular bloom, an occlusion shadow, the screen light's halo.
    `<radialGradient id="${g}sp"><stop offset="0" stop-color="#fff" stop-opacity=".9"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></radialGradient>` +
    `<radialGradient id="${g}ao"><stop offset="0" stop-color="${p.ink}" stop-opacity=".42"/><stop offset="1" stop-color="${p.ink}" stop-opacity="0"/></radialGradient>` +
    `<radialGradient id="${g}halo"><stop offset="0" stop-color="${p.eyeGlow}" stop-opacity=".55"/><stop offset="1" stop-color="${p.eyeGlow}" stop-opacity="0"/></radialGradient>` +
    `<linearGradient id="${g}sc" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="${p.screenUp}"/><stop offset="1" stop-color="${p.screen}"/></linearGradient>` +
    // Glass over the screen: a diagonal sheet of light, strongest top-left.
    `<linearGradient id="${g}gl" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#fff" stop-opacity=".22"/><stop offset=".45" stop-color="#fff" stop-opacity=".04"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient>` +
    // Bevelled blue: lit above, shaded below — cap, ear rings, soles.
    `<linearGradient id="${g}cap" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="${p.primaryUp}"/><stop offset=".55" stop-color="${p.primary}"/><stop offset="1" stop-color="${p.primaryDn}"/></linearGradient>` +
    `<radialGradient id="${g}orb" cx=".35" cy=".3" r=".8"><stop offset="0" stop-color="#fff"/><stop offset=".22" stop-color="${p.primaryUp}"/><stop offset="1" stop-color="${p.primaryDn}"/></radialGradient>` +
    `<linearGradient id="${g}slab" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="${p.slabUp}"/><stop offset="1" stop-color="${p.slab}"/></linearGradient>` +
    `<radialGradient id="${g}gnd"><stop offset="0" stop-color="${p.ink}" stop-opacity=".3"/><stop offset=".7" stop-color="${p.ink}" stop-opacity=".12"/><stop offset="1" stop-color="${p.ink}" stop-opacity="0"/></radialGradient>` +
    (m.glow ? `<filter id="${g}glow" x="-40%" y="-40%" width="180%" height="180%"><feGaussianBlur stdDeviation="1.3"/></filter>` : ``) +
    `</defs>`

  // Ground shadow, outside ms-all: the body lifts when it walks, its shadow stays.
  s += `<ellipse cx="32" cy="60.6" rx="18" ry="3" fill="url(#${g}gnd)"/>`
  if (m.ground) s += m.ground.svg(p, g)
  s += `<g class="ms-all">`
  // ---- behind the body: far ears, far arms, legs, torso
  if (detail) {
    s += ear(m, 'L', 'far', g) + ear(m, 'R', 'far', g)
    s += arm(p, g, 'L', 'far', hl) + arm(p, g, 'R', 'far', hl)
  }
  s += leg(p, g, 25, 'L', m.pose.legs, hl) + leg(p, g, 39, 'R', m.pose.legs, hl)
  s +=
    `<g class="ms-torso"><rect x="24" y="43" width="16" height="9.5" rx="4.5" fill="url(#${g}sh)"/>` +
    (detail ? `<ellipse cx="32" cy="46.5" rx="9" ry="3.2" fill="url(#${g}ao)"/>` : ``) +
    `<ellipse cx="32" cy="44.6" rx="7" ry="2.6" fill="${p.joint}"/><ellipse cx="32" cy="44" rx="5.2" ry="1.8" fill="${p.primaryDn}"/></g>`
  // ---- the indicator, under the cap so its stem reads as going INTO the head
  s += `<g class="ms-top">${m.top.svg(p, g)}</g>`
  // ---- head shell, lit from top-left; the light does not turn with the head
  s += `<circle cx="32" cy="27" r="19.5" fill="url(#${g}sh)"/>`
  if (detail) {
    // the bloom of the light on a glossy shell, the shadow the cap casts, and
    // the cool rim on the far side — the three things that make a ball read as a ball
    s += `<ellipse cx="24" cy="15.5" rx="7.5" ry="4.6" fill="url(#${g}sp)" transform="rotate(-28 24 15.5)"/>`
    s += `<ellipse cx="32" cy="15.5" rx="11" ry="3.2" fill="url(#${g}ao)"/>`
    s += `<path d="M45.4 14.8a19.5 19.5 0 0 1-9 30.6" fill="none" stroke="${p.shellRim}" stroke-width="1.6" stroke-linecap="round" opacity=".7"/>`
  }
  s += hl('M17.5 17.4a17.5 17.5 0 0 1 9-8.2', 1.8, 0.9)
  // grille on the back (φ = 180)
  if (detail) s += `<g class="ms-back"><g stroke="${p.bezel}" stroke-width="1.4" stroke-linecap="round"><path d="M26.5 24h11"/><path d="M26.5 27.5h11"/><path d="M28.5 31h7"/></g></g>`
  // cap
  s += `<g class="ms-cap"><path d="M22.4 11.6c3.8-6.2 15.4-6.2 19.2 0l-.9 3.6c-4.4-4.4-13-4.4-17.4 0z" fill="url(#${g}cap)"/>` +
       (detail ? `<path d="M23.3 15c4.4-4.2 13-4.2 17.4 0" fill="none" stroke="${p.primaryDn}" stroke-width=".8" opacity=".6"/>` : ``) +
       `${hl('M25.2 10.2c2.4-2 6-2.9 9.4-2.7', 0.9, 0.6)}</g>`
  // face screen (φ = 0)
  const light = m.face.svg(p, g)
  s +=
    `<g class="ms-face"><rect x="17.5" y="17.5" width="29" height="20.5" rx="9.5" fill="${p.bezel}"/>` +
    (detail ? `<rect x="17.5" y="17.5" width="29" height="20.5" rx="9.5" fill="none" stroke="${p.ink}" stroke-width=".8" opacity=".5"/>` : ``) +
    `<rect x="18.3" y="18.3" width="27.4" height="18.9" rx="8.8" fill="url(#${g}sc)"/>` +
    // the light's halo on the glass, then (big only) the light itself blurred under itself
    (detail ? `<ellipse cx="26" cy="28" rx="7" ry="6" fill="url(#${g}halo)"/><ellipse cx="38" cy="28" rx="7" ry="6" fill="url(#${g}halo)"/>` : ``) +
    (m.glow ? `<g class="ms-eyes" filter="url(#${g}glow)" opacity=".9">${light}</g>` : ``) +
    `<g class="ms-eyes">${light}</g>` +
    (detail ? `<rect x="18.3" y="18.3" width="27.4" height="18.9" rx="8.8" fill="url(#${g}gl)"/>` : ``) +
    `</g>`
  // near ears
  s += ear(m, 'L', 'near', g) + ear(m, 'R', 'near', g)
  // ---- in front of the body: what the hands hold, then the near arms over it
  if (m.prop) s += (detail ? `<ellipse cx="32" cy="52" rx="12" ry="3" fill="url(#${g}ao)"/>` : ``) + `<g class="ms-prop">${m.prop.svg(p, g)}</g>`
  s += arm(p, g, 'L', 'near', hl) + arm(p, g, 'R', 'near', hl)
  s += `</g>`
  // ---- floating UI: neither turns nor breathes
  if (m.mark) s += m.mark(p)
  if (m.panel) s += m.panel.svg(p, g)
  return s
}

// 4/6 · An ear module: ring, dark centre, badge. The left ear is at 12.5, the
// right at 51.5 — both on the sphere's edge, which is why in the front view
// they are thin plates hugging the head and not discs facing the viewer.
function ear(m: Mascot, side: 'L' | 'R', layer: 'far' | 'near', g: string): string {
  const { p } = m
  const cx = side === 'L' ? 12.5 : 51.5
  const badge = side === 'L' ? m.badgeL : m.badgeR
  const near = layer === 'near'
  return (
    `<g class="ms-ear ms-ear${side} ${layer}">` +
    `<circle cx="${cx}" cy="29" r="7" fill="${p.primaryDn}"/>` +
    `<circle cx="${cx}" cy="29" r="5.8" fill="url(#${g}cap)"/>` +
    `<circle cx="${cx}" cy="29" r="4.3" fill="${p.screen}"/>` +
    (near && m.detail ? `<circle cx="${cx}" cy="29" r="4.3" fill="none" stroke="${p.ink}" stroke-width=".7" opacity=".6"/>` +
      `<path d="M${cx - 5.3} 25.5a7 7 0 0 1 6.4-3.4" fill="none" stroke="#fff" stroke-width="1" opacity=".55" stroke-linecap="round"/>` : ``) +
    (near ? `<g class="ms-badge">${icon(badge, cx, 29, 6.2, '#fff', 2.6)}</g>` : ``) +
    `</g>`
  )
}

// 10/11 · A leg with its foot: a white ball under a blue cap (the hip), then
// the foot on its blue sole. Tucked under a sitting body, or standing.
function leg(p: Palette, g: string, cx: number, side: 'L' | 'R', mode: 'tuck' | 'stand', hl: (d: string, w?: number, o?: number) => string): string {
  const thigh = (cy: number): string =>
    `<circle cx="${cx}" cy="${cy}" r="3.6" fill="url(#${g}sh)" stroke="${p.shellEdge}" stroke-width=".5"/>` +
    `<ellipse cx="${cx}" cy="${cy - 2.6}" rx="2.4" ry="1.1" fill="${p.primaryDn}"/>`
  const foot =
    mode === 'tuck'
      ? thigh(52.2) +
        // the foot comes forward, sole to the viewer: a blue disc with a lit rim
        `<ellipse cx="${cx}" cy="57.8" rx="5.6" ry="2.6" fill="${p.primaryDn}"/><ellipse cx="${cx}" cy="57.2" rx="4.6" ry="1.9" fill="${p.primary}"/>` +
        `<ellipse cx="${cx}" cy="55.2" rx="5" ry="3" fill="url(#${g}sh)"/>` +
        hl(`M${cx - 3} 54a3.6 1.9 0 0 1 4.2-1.1`)
      : thigh(51) +
        `<ellipse cx="${cx}" cy="58" rx="5" ry="2.4" fill="${p.primaryDn}"/><ellipse cx="${cx}" cy="56.4" rx="4.6" ry="3" fill="url(#${g}sh)"/>` +
        hl(`M${cx - 2.8} 55a3.4 1.9 0 0 1 4-1.1`)
  return `<g class="ms-leg ms-leg${side}">${foot}</g>`
}

// 8/9 · An arm with one joint: a ball socket at the shoulder, one white
// capsule, a white sphere for the hand. Capsule and sphere wear the shell's
// own gradient, so an arm is the same material as the head, with a lit side
// and a dark rim. The CSS turns the whole thing at the shoulder towards the
// hand's projected target (mascot.css); the length never changes.
function arm(p: Palette, g: string, side: 'L' | 'R', layer: 'far' | 'near', hl: (d: string, w?: number, o?: number) => string): string {
  const [sx, sy] = SHOULDER[side]
  const wx = sx + ARM
  const r = 2.4
  // The target is read off the root (handVars), not written here: the root
  // outlives a redraw, so when the pose changes the numbers slide from the
  // old target to the new one and the arm swings instead of jumping.
  const v = side === 'L' ? 'l' : 'r'
  return (
    `<g class="ms-arm ms-arm${side} ${layer}" style="--sx:${sx};--sy:${sy};--hx:var(--${v}hx);--hy:var(--${v}hy);--d:var(--${v}d);transform-origin:${sx}px ${sy}px">` +
    `<g class="ms-upper" style="transform-origin:${sx}px ${sy}px">` +
    `<rect x="${sx - r}" y="${sy - r}" width="${ARM + 2 * r}" height="${2 * r}" rx="${r}" fill="url(#${g}sh)" stroke="${p.shellEdge}" stroke-width=".5"/>` +
    `<circle cx="${sx}" cy="${sy}" r="2.1" fill="${p.joint}"/>` +
    hl(`M${sx - 1.2} ${sy - 1}a1.5 1.5 0 0 1 1.7-.3`, 0.6, 0.5) +
    `<g class="ms-palm">` +
    `<circle cx="${wx + 0.8}" cy="${sy}" r="3.5" fill="url(#${g}sh)" stroke="${p.shellEdge}" stroke-width=".55"/>` +
    hl(`M${wx - 1.2} ${sy - 1.7}a2.5 2.5 0 0 1 2.5-1.2`, 0.8) +
    `</g></g></g>`
  )
}
