// Every transition in the app takes its length from the token layer.
//
// The tokens were there first (styles/type.css) and the app ignored them. On
// 7 ก.ย. 2026 style.css held 54 `transition:` declarations and 28 different
// lengths between them, while `--dur-press`, `--dur-arrive` and the rest sat
// one file away being spent by the onboarding screen and the tool row and
// nobody else. That is the failure this test exists to make loud: not a system
// that was missing, but a system that was easy to walk past.
//
// It guards TRANSITIONS only. An `animation:` is often a decorative loop with a
// period of its own — a beam crossing an edge in 3s, a tile breathing at 1.7s —
// and those are drawings rather than answers to "did the press register". The
// four that ARE answers already hold tokens (--beam-in/out, --swap-*), and a
// rule wide enough to cover both would have to carry a list of exceptions,
// which is the thing it was written to stop.
//
// A DELAY may still be a literal. It is not a length of movement — it is how
// long the app waits before starting one, which is a per-place decision (the
// .35s a tooltip waits before it believes you meant to hover).
import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'

const css = readFileSync('src/style.css', 'utf8')
const tokens = readFileSync('src/styles/type.css', 'utf8')

/** Every `transition:` declaration in the sheet, body only. */
function declarations(): string[] {
  return [...css.matchAll(/transition:([^;]+);/g)].map((m) => m[1])
}

describe('the motion token layer', () => {
  it('is what every transition in the app asks for its length', () => {
    const literal: string[] = []
    for (const body of declarations()) {
      for (const part of body.split(',')) {
        // Position, not presence. The first time value in a part is the
        // duration and the second is the delay, so a literal is a violation
        // only when nothing token-shaped came before it.
        const time = part.search(/(?<![\w-])(\.\d+s|\d+(\.\d+)?m?s)/)
        const token = part.search(/var\(--/)
        if (time !== -1 && (token === -1 || token > time)) literal.push(part.trim())
      }
    }
    // `transition:none` and the var() forms leave nothing behind.
    expect(literal).toEqual([])
  })

  it('names the two jobs the app was writing out by hand', () => {
    // The counts that decided the values are in the comment beside them; what
    // matters here is that the names exist to be reached for.
    expect(tokens).toMatch(/--dur-tint:\s*120ms/)
    expect(tokens).toMatch(/--dur-glide:\s*180ms/)
  })

  it('is what the popovers arrive on, all ten of them', () => {
    // Ten menus, one surface, written out ten times — and until 7 ก.ย. not one
    // of them had an arrival. The list is the guard: a menu added below and
    // left off it pops into place, which is what every one of these used to do.
    const arrival = css.slice(css.indexOf('@keyframes menu-in'))
    const rule = arrival.slice(0, arrival.indexOf('}', arrival.indexOf('animation:menu-in')))
    for (const menu of [
      'focus-menu', 'attach-menu', 'stance-menu', 'branch-menu', 'model-menu',
      'ctx-menu', 'summary-menu', 'busy-menu', 'plus-menu',
    ]) {
      expect(rule).toContain(`.${menu}`)
    }
    // It grows out of the edge it hangs off, or it is drifting in from nowhere.
    expect(arrival).toMatch(/transform-origin:bottom center/)
    expect(arrival).toMatch(/transform-origin:top center/)
  })

  it('spends them where the app actually moves', () => {
    // A token nothing uses is a token that will be missed again. These numbers
    // are floors, not fixtures: they say the sweep happened, and they do not
    // break when the next hover row joins.
    const uses = (name: string) => css.split(`var(${name})`).length - 1
    expect(uses('--dur-tint')).toBeGreaterThan(20)
    expect(uses('--dur-glide')).toBeGreaterThan(3)
    expect(uses('--dur-press')).toBeGreaterThan(3)
  })
})
