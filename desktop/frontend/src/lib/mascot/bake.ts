// Frames of the mascot, baked for a surface that has no browser.
//
// The companion on the desktop is a Win32 window drawn by Go
// (desktop/companion_windows.go) — no WebView2, by the owner's choice, so
// nothing out there can run rig.ts or mascot.css. This file is how the same
// mascot gets there anyway: the app's own webview draws it, exactly as
// Mascot.svelte does, and hands over pictures.
//
// Not a second implementation of the motion. mascot.css stays the only place
// anything moves; what happens here is that a frame is taken of it. The
// markup is mounted for real (off screen), every CSS animation on it is
// moved to the moment wanted with the Web Animations API and paused, and
// the browser's own computed `transform` and `opacity` of every part —
// keyframes, easing, the `sin(var(--t))` trigonometry, all of it evaluated
// by the engine that owns it — are written into a copy as inline style. The
// copy has no class and no stylesheet left in it, so it draws the same in an
// <img>, where it is rasterised to a PNG at the monitor's scale.
//
// What a frame is: one pose, at one head turn, at one moment of its loop,
// eyes open or shut. A pose that moves continuously (breathing, a wave, the
// walk) is sampled PHASES times over one period and the Go side blends the
// neighbours, so four pictures read as a loop. The walk is sampled at every
// WALK_STEP degrees of heading as well, because on the desktop the head is
// turned by the drag (companion_walk.go) and not by CSS.
//
// The set is keyed and cached on disk per look and per scale (companion.go),
// and versioned by what it was drawn from: change a part or a keyframe and
// the old cache is simply another hash.
import cssText from './mascot.css?raw'
import { ICONS, type IconName } from '../icons'
import { POSE, type PoseId } from './poses'
import { handVars, mascotSVG, resolveMascot, type MascotOptions } from './rig'

/** Samples per loop of a pose. */
export const PHASES = 4
/** The walk is sampled finer: legs cross-dissolving between four pictures
 *  read as a blur, eight read as steps. */
export const WALK_PHASES = 8
export function phasesOf(pose: string): number {
  return pose === 'walk' ? WALK_PHASES : PHASES
}
/** Degrees between walk headings. 360 / WALK_STEP frames per phase. */
export const WALK_STEP = 30
/** The figure is drawn 64 units wide; a frame keeps this much around it so a
 *  raised hand or the pillow is not cut at the edge. */
export const PAD = 8
export const BOX = 64 + 2 * PAD
/** The logical size the companion is drawn at (Companion.svelte SIZE). A frame
 *  is FIGURE * BOX / 64 logical pixels square before the monitor's scale. */
export const FIGURE = 104
/** The icons the desktop frame draws on hover: the hide and the voice buttons. */
export const FRAME_ICONS = ['x', 'volume2', 'volumeX'] as const satisfies readonly IconName[]
export const ICON_PX = 11

export type Blink = 'open' | 'shut'

/** One frame to bake. `turn` is only carried by the walk; every other pose is
 *  drawn at its own rest turn. */
export type FrameSpec =
  | { kind: 'pose'; pose: PoseId; phase: number; blink: Blink }
  | { kind: 'walk'; turn: number; phase: number }
  | { kind: 'icon'; icon: IconName }

/** The key a frame is stored and asked for by. Filename-safe on purpose:
 *  `idle-p0-open`, `walk-t90-p2`, `icon-x`. The Go side composes these. */
export function keyOf(f: FrameSpec): string {
  switch (f.kind) {
    case 'pose':
      return `${f.pose}-p${f.phase}-${f.blink}`
    case 'walk':
      return `walk-t${f.turn}-p${f.phase}`
    case 'icon':
      return `icon-${f.icon}`
  }
}

export function parseKey(key: string): FrameSpec | null {
  let m = /^icon-(\w+)$/.exec(key)
  if (m) return m[1] in ICONS ? { kind: 'icon', icon: m[1] as IconName } : null
  m = /^walk-t(-?\d+)-p(\d+)$/.exec(key)
  if (m) return { kind: 'walk', turn: Number(m[1]), phase: Number(m[2]) }
  m = /^(\w+)-p(\d+)-(open|shut)$/.exec(key)
  if (m && m[1] in POSE && m[1] !== 'walk') return { kind: 'pose', pose: m[1] as PoseId, phase: Number(m[2]), blink: m[3] as Blink }
  return null
}

/** Every frame the desktop body can ask for. The shut-eyed frame is one per
 *  pose: a blink is a fifth of a second, nobody sees which phase it fell on. */
export function allFrames(): FrameSpec[] {
  const out: FrameSpec[] = []
  for (const pose of Object.keys(POSE) as PoseId[]) {
    if (pose === 'walk') continue
    for (let phase = 0; phase < PHASES; phase++) out.push({ kind: 'pose', pose, phase, blink: 'open' })
    out.push({ kind: 'pose', pose, phase: 0, blink: 'shut' })
  }
  for (let turn = -180 + WALK_STEP; turn <= 180; turn += WALK_STEP) {
    for (let phase = 0; phase < WALK_PHASES; phase++) out.push({ kind: 'walk', turn, phase })
  }
  for (const icon of FRAME_ICONS) out.push({ kind: 'icon', icon })
  return out
}

/** The frames wanted first, so the body can appear before the rest is baked:
 *  the pose it is in now, the rest pose, the icons. */
export function firstFrames(pose: string): FrameSpec[] {
  const want = new Set<string>()
  const out: FrameSpec[] = []
  const add = (f: FrameSpec): void => {
    const k = keyOf(f)
    if (!want.has(k)) {
      want.add(k)
      out.push(f)
    }
  }
  for (const p of [pose, 'idle']) {
    if (!(p in POSE) || p === 'walk') continue
    for (let phase = 0; phase < PHASES; phase++) add({ kind: 'pose', pose: p as PoseId, phase, blink: 'open' })
    add({ kind: 'pose', pose: p as PoseId, phase: 0, blink: 'shut' })
  }
  for (const icon of FRAME_ICONS) add({ kind: 'icon', icon })
  return out
}

/** How long one loop of a pose lasts, and where in the pose's own time the
 *  loop starts — read off mascot.css. Breathing (3.6s) is the loop of a pose
 *  at rest; the walk cycle is .62s; a wave is .7s each way; typing hands tick
 *  at .18s; the sleeper lies down over .9s first, then snores at 3.2s; waking
 *  and startling are one-shot and are sampled across their length. */
export function loopOf(pose: string): { period: number; offset: number } {
  switch (pose) {
    case 'walk':
      return { period: 0.62, offset: 0 }
    case 'greeting':
      return { period: 1.4, offset: 0 }
    case 'typing':
    case 'coding':
    case 'searchData':
      return { period: 0.36, offset: 0 }
    case 'recharge':
      return { period: 3.2, offset: 0.9 }
    case 'wake':
      return { period: 1.2, offset: 0 }
    case 'startled':
      return { period: 0.5, offset: 0 }
    default:
      return { period: 3.6, offset: 0 }
  }
}

/** The moment, in seconds of the pose's own time, that a phase is a picture of. */
export function momentOf(pose: string, phase: number): number {
  const { period, offset } = loopOf(pose)
  return offset + (period * phase) / phasesOf(pose)
}

/** Where a blink is shut in its keyframes (mascot.css ms-blink: 97%). */
const BLINK_SHUT = 0.97

/** Versions the cache: the drawing of every pose plus the motion sheet. Any
 *  change to a part, a palette, a pose or a keyframe changes it. */
export function rigVersion(opts: MascotOptions): string {
  let h = 2166136261
  const mix = (s: string): void => {
    for (let i = 0; i < s.length; i++) {
      h ^= s.charCodeAt(i)
      h = Math.imul(h, 16777619)
    }
  }
  mix(cssText)
  // Gradient ids are numbered per drawing (rig.ts) and are not the drawing.
  for (const pose of Object.keys(POSE)) mix(mascotSVG(resolveMascot({ ...opts, pose, size: FIGURE })).replace(/\bms\d+/g, 'ms'))
  return (h >>> 0).toString(16).padStart(8, '0')
}

/** The identity of one baked set: what it looks like, at what scale, from
 *  which drawing. Frames from two different hashes never mix. */
export function setHash(opts: MascotOptions, scale: number): string {
  return `${rigVersion(opts)}-${Math.round(scale * 100)}`
}

// ---- the snapshot ---------------------------------------------------------

/** Mounts the markup the way Mascot.svelte does, seeks every animation on it
 *  to the frame's moment, and returns a standalone SVG document with every
 *  part's computed transform and opacity written inline. Needs a real
 *  layout engine (a browser); under jsdom the computed values come back
 *  empty and the document is still well-formed. */
export function snapshotSVG(opts: MascotOptions, f: FrameSpec, doc: Document = document): string {
  if (f.kind === 'icon') return iconSVG(f.icon)
  const pose = f.kind === 'walk' ? 'walk' : f.pose
  const m = resolveMascot({ ...opts, pose, size: FIGURE })
  const turn = f.kind === 'walk' ? f.turn : m.pose.turn
  const host = doc.createElement('span')
  host.className = `mascot pose-${m.poseId}`
  host.setAttribute('style', `--t:${turn}deg; ${handVars(m)}; --ms-blink:5s; width:${FIGURE}px; height:${FIGURE}px; position:fixed; left:-${FIGURE * 4}px; top:0; pointer-events:none`)
  host.innerHTML = `<svg viewBox="0 0 64 64" style="--ms-phase:0s">${mascotSVG(m)}</svg>`
  doc.body.appendChild(host)
  try {
    seek(host, momentOf(pose, f.phase), f.kind === 'pose' && f.blink === 'shut')
    const live = host.querySelector('svg') as SVGSVGElement
    const copy = live.cloneNode(true) as SVGSVGElement
    inlineComputed(live, copy, doc.defaultView ?? window)
    const px = Math.round((FIGURE * BOX) / 64)
    copy.setAttribute('xmlns', 'http://www.w3.org/2000/svg')
    copy.setAttribute('viewBox', `${-PAD} ${-PAD} ${BOX} ${BOX}`)
    copy.setAttribute('width', String(px))
    copy.setAttribute('height', String(px))
    copy.removeAttribute('style')
    return new XMLSerializer().serializeToString(copy)
  } finally {
    host.remove()
  }
}

/** Every animation under the host is paused at `at` seconds — except the
 *  blink, which is put where its keyframes shut the eyes, or where they are
 *  open; a blink is not a phase of the pose. */
function seek(host: Element, at: number, shut: boolean): void {
  const anims = typeof host.getAnimations === 'function' ? host.getAnimations({ subtree: true }) : []
  for (const a of anims) {
    const target = (a.effect as KeyframeEffect | null)?.target as Element | null
    a.pause()
    if (target?.classList.contains('ms-blink')) {
      const d = Number((a.effect as KeyframeEffect).getComputedTiming().duration) || 5000
      a.currentTime = shut ? d * BLINK_SHUT : 0
    } else {
      a.currentTime = at * 1000
    }
  }
}

/** The properties mascot.css animates or turns: nothing else moves. */
function inlineComputed(live: Element, copy: Element, view: Window): void {
  const from = live.querySelectorAll('*')
  const to = copy.querySelectorAll('*')
  from.forEach((el, i) => {
    const cs = view.getComputedStyle(el)
    const transform = cs.transform
    const opacity = cs.opacity
    const moved = transform && transform !== 'none'
    const faded = opacity !== '' && opacity !== '1'
    if (!moved && !faded) return
    const c = to[i]
    let style = c.getAttribute('style') ?? ''
    if (moved) style += `;transform:${transform};transform-origin:${cs.transformOrigin};transform-box:${cs.transformBox}`
    if (faded) style += `;opacity:${opacity}`
    c.setAttribute('style', style.replace(/^;/, ''))
    c.removeAttribute('class')
  })
}

function iconSVG(icon: IconName): string {
  const px = ICON_PX
  return (
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="${px}" height="${px}" fill="none" stroke="#fff"` +
    ` stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">${ICONS[icon]}</svg>`
  )
}

// ---- the picture ----------------------------------------------------------

export type BakedFrame = { key: string; w: number; h: number; png: string }

/** Rasterises one frame at `scale` pixels per logical pixel; png is base64. */
export async function bakeFrame(opts: MascotOptions, f: FrameSpec, scale: number): Promise<BakedFrame> {
  const svg = snapshotSVG(opts, f)
  const logical = f.kind === 'icon' ? ICON_PX : (FIGURE * BOX) / 64
  const px = Math.max(1, Math.round(logical * scale))
  const img = new Image()
  await new Promise<void>((res, rej) => {
    img.onload = () => res()
    img.onerror = () => rej(new Error(`companion frame ${keyOf(f)} did not draw`))
    img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
  })
  const canvas = document.createElement('canvas')
  canvas.width = px
  canvas.height = px
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('companion frame: no 2d context')
  ctx.drawImage(img, 0, 0, px, px)
  const url = canvas.toDataURL('image/png')
  return { key: keyOf(f), w: px, h: px, png: url.slice(url.indexOf(',') + 1) }
}

/** Bakes frames in order, handing them over `batch` at a time and yielding to
 *  the page between batches — two hundred rasterisations in one go would
 *  hold the window for a second. Stops early if `stop` says so (the look
 *  changed, the body was closed). */
export async function bakeFrames(
  opts: MascotOptions,
  frames: FrameSpec[],
  scale: number,
  deliver: (frames: BakedFrame[]) => Promise<void>,
  batch = 20,
  stop: () => boolean = () => false,
): Promise<number> {
  let done = 0
  for (let i = 0; i < frames.length; i += batch) {
    if (stop()) break
    const out: BakedFrame[] = []
    for (const f of frames.slice(i, i + batch)) out.push(await bakeFrame(opts, f, scale))
    await deliver(out)
    done += out.length
    await new Promise((r) => setTimeout(r, 0))
  }
  return done
}
