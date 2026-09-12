// The companion's body on the desktop, as the brain in this window reaches it.
//
// Companion.svelte is the brain either way: it decides the pose, what the
// bubble says, when to sleep. With `place` set to the desktop
// (companionSetting.svelte.ts) the figure is a Win32 window drawn by Go
// (desktop/companion_windows.go) instead of this window's DOM, and this
// file is the seam: it opens and closes that window, bakes the frames it
// draws from (bake.ts), and hands its clicks and drags back to the brain.
//
// Reached through the runtime's own binding table rather than the generated
// wailsjs import (see Companion.svelte's feed for why); outside Wails —
// tests, a plain browser — there is no body, and `up` stays false.

import { EventsOn } from '../../../wailsjs/runtime/runtime'
import type { MascotOptions } from './rig'
import { allFrames, bakeFrames, firstFrames, keyOf, setHash, type FrameSpec } from './bake'

type Frame = { key: string; w: number; h: number; png: string }
type Bindings = {
  OpenCompanionWindow?: (x: number, y: number, size: number) => Promise<boolean>
  CloseCompanionWindow?: () => Promise<void>
  CompanionSpriteKeys?: (hash: string) => Promise<string[]>
  CompanionSprites?: (hash: string, frames: Frame[]) => Promise<unknown>
}

const POS_KEY = 'companionDesktopPos'

export type BodyInput =
  | { kind: 'click' | 'dragStart' | 'dragEnd' | 'hide' | 'mute' }
  | { kind: 'moved'; x: number; y: number }
  | { kind: 'bake'; scale: number }
  | { kind: 'resize'; size: number }

export const desktopBody = $state<{ up: boolean; scale: number; baking: boolean }>({ up: false, scale: 0, baking: false })

function app(): Bindings {
  return (window as unknown as { go?: { main?: { App?: Bindings } } }).go?.main?.App ?? {}
}

function savedPos(): { x: number; y: number } {
  try {
    const raw = localStorage.getItem(POS_KEY)
    if (raw) {
      const p = JSON.parse(raw) as { x: number; y: number }
      if (Number.isFinite(p.x) && Number.isFinite(p.y)) return p
    }
  } catch {
    // no spot remembered: Go picks the corner
  }
  return { x: -1, y: -1 }
}

export function rememberPos(x: number, y: number): void {
  try {
    localStorage.setItem(POS_KEY, JSON.stringify({ x, y }))
  } catch {
    // still the spot for this session; Go keeps the window where it is
  }
}

/** Sends the figure out to the desktop. False when the platform cannot, in
 *  which case the brain keeps drawing it here. */
export async function openBody(size: number): Promise<boolean> {
  const open = app().OpenCompanionWindow
  if (!open) return false
  const p = savedPos()
  try {
    desktopBody.up = await open(p.x, p.y, size)
  } catch {
    desktopBody.up = false
  }
  return desktopBody.up
}

export async function closeBody(): Promise<void> {
  desktopBody.up = false
  desktopBody.scale = 0
  try {
    await app().CloseCompanionWindow?.()
  } catch {
    // the window is the Go side's to close; nothing to do here if it cannot
  }
}

/** Listens to the body's doings for as long as the returned function is not
 *  called. */
export function onBodyInput(handler: (input: BodyInput) => void): () => void {
  try {
    return EventsOn('companion:input', (d: BodyInput) => handler(d))
  } catch {
    return () => {}
  }
}

let bakeRun = 0

/** Bakes the frames the body lacks for a scale, the pose it is in first so
 *  it appears at once, then the rest in the background. A newer call (the
 *  look changed, another scale) stops an older one. */
export async function bakeFor(opts: MascotOptions, scale: number, pose: string): Promise<void> {
  const keys = app().CompanionSpriteKeys
  const put = app().CompanionSprites
  if (!keys || !put || scale <= 0) return
  const run = ++bakeRun
  const hash = setHash(opts, scale)
  let have: Set<string>
  try {
    have = new Set(await keys(hash))
  } catch {
    return
  }
  const first = firstFrames(pose)
  const seen = new Set(first.map(keyOf))
  const rest = allFrames().filter((f) => !seen.has(keyOf(f)))
  const missing: FrameSpec[] = [...first, ...rest].filter((f) => !have.has(keyOf(f)))
  if (missing.length === 0) return
  desktopBody.baking = true
  try {
    await bakeFrames(
      opts,
      missing,
      scale,
      async (frames) => {
        const err = await put(hash, frames)
        if (err) throw new Error(String(err))
      },
      20,
      () => run !== bakeRun,
    )
  } catch {
    // a frame that would not bake is a frame the body draws without
  } finally {
    if (run === bakeRun) desktopBody.baking = false
  }
}

/** The bubble's and frame's colours, off this window's stylesheet, so the
 *  desktop matches the app's theme. */
export function themeColors(): { bg: string; fg: string; muted: string; border: string; accent: string } {
  if (typeof getComputedStyle !== 'function') return { bg: '', fg: '', muted: '', border: '', accent: '' }
  const cs = getComputedStyle(document.documentElement)
  const v = (name: string): string => cs.getPropertyValue(name).trim()
  return { bg: v('--surface-raised'), fg: v('--text-primary'), muted: v('--text-muted'), border: v('--border-subtle'), accent: v('--accent') }
}

/** The words of a text as the body should break lines between them — Thai
 *  has no spaces, and Go has no dictionary. */
export function wordsOf(text: string, locale: string): string[] {
  if (!text) return []
  try {
    const seg = new Intl.Segmenter(locale, { granularity: 'word' })
    return [...seg.segment(text)].map((s) => s.segment)
  } catch {
    return text.split(/(\s+)/).filter(Boolean)
  }
}
