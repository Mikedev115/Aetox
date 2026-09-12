// The one thing that makes the mascot alive between poses: it looks at you.
//
// A Svelte action. While it is ON, the pointer's position relative to the
// mascot becomes a head turn — clamped to the pose's own `look` range so a
// raised hand is never swung into an ear — damped a little so the head does
// not snap, and after the pointer has been still for a while the head glances
// somewhere on its own and then comes back. All of it writes ONE custom
// property, --t; mascot.css does the rest.
//
// Off, it costs nothing: no listener, no frame loop, and the --t the template
// wrote stands. One loop per mascot that has this on, and only while it is
// on. Tiles in a roster never get it: forty heads tracking one cursor is a
// horror film, not a company.
import type { Action } from 'svelte/action'

export type LookAtOptions = {
  /** Degrees the head rests at (the pose's own turn). */
  base: number
  /** Degrees it may swing either way around `base`. */
  range: number
  /** Master switch. */
  on?: boolean
}

/** How far (px) the pointer can be from the mascot's centre and still be
 *  something it looks at. Beyond it the head returns to rest. */
const REACH_PX = 900
/** Pixels of pointer offset per degree of turn. */
const PX_PER_DEG = 4
/** Damping per frame — 0.12 settles a 60° swing in about half a second. */
const EASE = 0.12
/** After this long without pointer movement the mascot may glance on its own. */
const STILL_MS = 2500

export const lookAt: Action<HTMLElement, LookAtOptions> = (node, opts) => {
  let base = opts.base
  let range = opts.range
  let running = false
  let pointer: number | null = null
  let lastMove = 0
  let goal = base
  let cur = base
  let frame = 0
  const reduced = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

  const clampGoal = (t: number): number => Math.max(base - range, Math.min(base + range, t))
  const write = (): void => node.style.setProperty('--t', `${cur.toFixed(2)}deg`)

  const onMove = (e: PointerEvent): void => {
    const r = node.getBoundingClientRect()
    const dx = e.clientX - (r.left + r.width / 2)
    const dy = e.clientY - (r.top + r.height / 2)
    if (Math.hypot(dx, dy) > REACH_PX) {
      pointer = null
      return
    }
    pointer = clampGoal(base + dx / PX_PER_DEG)
    lastMove = performance.now()
  }

  const tick = (): void => {
    const now = performance.now()
    if (pointer !== null && now - lastMove < STILL_MS) goal = pointer
    // Left alone, glance somewhere now and then, and come back.
    else if (Math.random() < 0.006) goal = clampGoal(base + (Math.random() - 0.5) * 50)
    else if (Math.random() < 0.004) goal = base
    const next = reduced ? goal : cur + (goal - cur) * EASE
    if (Math.abs(next - cur) > 0.05) {
      cur = next
      write()
    }
    frame = requestAnimationFrame(tick)
  }

  const start = (): void => {
    if (running) return
    running = true
    cur = goal = base
    window.addEventListener('pointermove', onMove, { passive: true })
    frame = requestAnimationFrame(tick)
  }
  const stop = (): void => {
    if (!running) return
    running = false
    cancelAnimationFrame(frame)
    window.removeEventListener('pointermove', onMove)
    // Back to rest — on the same inline declaration the template writes, so
    // there is no flash of the stylesheet's 0deg before Svelte's next render.
    cur = goal = base
    write()
  }

  if (opts.on ?? true) start()

  return {
    update(next: LookAtOptions) {
      base = next.base
      range = next.range
      if (!(next.on ?? true)) {
        stop()
        return
      }
      start()
      // A new pose has a new rest angle: re-clamp what we were aiming at, and
      // write now — Svelte may have just rewritten the style attribute.
      goal = clampGoal(goal)
      if (pointer !== null) pointer = clampGoal(pointer)
      write()
    },
    destroy: stop,
  }
}
