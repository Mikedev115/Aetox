// The folding transition, checked where it can be: its own configuration.
//
// What it does on screen is not testable here, and the part that would silently
// rot is not the easing — it is the reduced-motion guard. svelte/transition does
// not consult the setting, and the CSS half of the app does, so a fold that
// ignored it would be the one animation in the app that keeps moving after the
// user asked everything to stop.
import { describe, it, expect, afterEach } from 'vitest'
import { fold, settle, sidle, SETTLE_MS } from '../lib/fold'

const el = () => document.createElement('div')
const withMotionSetting = (reduce: boolean) => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: (query: string) => ({ matches: reduce && query.includes('reduce'), media: query }),
  })
}

afterEach(() => {
  Reflect.deleteProperty(window, 'matchMedia')
})

describe('the folding transition', () => {
  it('folds over time by default', () => {
    withMotionSetting(false)
    expect(fold(el()).duration).toBe(240)
  })

  it('collapses to no time at all when motion is unwelcome', () => {
    withMotionSetting(true)
    expect(fold(el()).duration).toBe(0)
  })

  // The one thing a delay is for here: staging an arrival in two beats. It
  // rides on this transition rather than on a timer of its own so the guard
  // above covers it too — asked to stop moving, the app must not answer with a
  // pause instead.
  it('can hold an arrival back a beat, and never once motion is unwelcome', () => {
    withMotionSetting(false)
    expect(fold(el(), { delay: 260 }).delay).toBe(260)
    withMotionSetting(true)
    expect(fold(el(), { delay: 260 }).delay).toBe(0)
  })

  // Read per call, not captured at module load: the setting can change while
  // the app is open, and a value read once would need a restart to take effect.
  it('reads the setting again rather than remembering the first answer', () => {
    withMotionSetting(false)
    expect(fold(el()).duration).toBe(240)
    withMotionSetting(true)
    expect(fold(el()).duration).toBe(0)
  })

  // The test environment has no matchMedia at all, and neither will some host
  // the app is embedded in one day. Missing must mean "animate", not "throw".
  it('animates when the platform cannot say', () => {
    Reflect.deleteProperty(window, 'matchMedia')
    expect(fold(el()).duration).toBe(240)
  })

  it('lets a caller ask for a different length', () => {
    withMotionSetting(false)
    expect(fold(el(), { duration: 90 }).duration).toBe(90)
  })

})

// The slower one, for the single movement the owner asked to be able to watch:
// a stretch of work folding away once its last call has come back.
describe('the settling fold', () => {
  const tall = () => {
    const node = el()
    Object.defineProperty(node, 'offsetHeight', { value: 120, configurable: true })
    return node
  }

  // Twice a fold's length, on purpose. Four rounds of "ค่อยๆ ดุนุ่มๆ" ended
  // here, and a duration that drifts back towards fold's is the regression.
  it('takes longer than a fold, and lets a caller say how much longer', () => {
    withMotionSetting(false)
    expect(settle(tall()).duration).toBe(SETTLE_MS)
    expect(SETTLE_MS).toBeGreaterThan(fold(el()).duration as number)
    expect(settle(tall(), { duration: 0 }).duration).toBe(0)
  })

  // The two things that make it read as gentle rather than merely slow: it
  // fades as it shrinks, so the last rows LEAVE instead of being trimmed off
  // against the row below, and it is eased at both ends.
  it('fades as it shrinks, and eases at both ends', () => {
    withMotionSetting(false)
    const t = settle(tall())
    expect(t.css!(1)).toContain('opacity:1')
    expect(t.css!(0)).toContain('opacity:0')
    // cubicInOut: slow away from both ends, which cubicOut is not.
    expect(t.easing!(0.5)).toBeCloseTo(0.5, 5)
    expect(t.easing!(0.1)).toBeLessThan(0.1)
    expect(t.easing!(0.9)).toBeGreaterThan(0.9)
  })

  // A gap belongs to the container, not to the block folding inside it, so it
  // stays whole for as long as the child exists: without this the height
  // reaches zero and THEN the gap goes, in one frame — the jump the whole
  // movement was slowed down to remove.
  it('unwinds the container gap along with the height', () => {
    withMotionSetting(false)
    const css = settle(tall(), { gap: 8 }).css!
    expect(css(1)).toContain('height:120px')
    expect(css(1)).toContain('margin-bottom:0px')
    expect(css(0)).toContain('height:0px')
    expect(css(0)).toContain('margin-bottom:-8px')
  })

  // The guard covers the second transition too, which is exactly where a rule
  // gets left behind.
  it('collapses to no time at all when motion is unwelcome', () => {
    withMotionSetting(true)
    expect(settle(tall()).duration).toBe(0)
  })
})

// The horizontal one, for the strip where an un-animated removal is worst: a
// tab leaving takes its width with it, and every tab to its right jumps.
describe('the sidling tab transition', () => {
  const chip = (width = 140, pad = 12) => {
    const node = el()
    Object.defineProperty(node, 'offsetWidth', { value: width, configurable: true })
    node.style.paddingLeft = `${pad}px`
    node.style.paddingRight = `${pad}px`
    return node
  }

  it('grows and shrinks by its own width, and fades with it', () => {
    withMotionSetting(false)
    const css = sidle(chip()).css!
    expect(css(1)).toContain('width:140px')
    expect(css(1)).toContain('opacity:1')
    expect(css(0)).toContain('width:0px')
    expect(css(0)).toContain('opacity:0')
  })

  // The floor that puts the flick back in the last frame: with border-box the
  // padding is a width the element cannot go under, so a tab told to be 0px
  // wide stands at 24px and then vanishes. The padding has to leave with it.
  it('takes its padding down with the width, so nothing is left standing', () => {
    withMotionSetting(false)
    const css = sidle(chip(140, 12)).css!
    expect(css(1)).toContain('padding-left:12px')
    expect(css(1)).toContain('padding-right:12px')
    expect(css(0)).toContain('padding-left:0px')
    expect(css(0)).toContain('padding-right:0px')
  })

  // Same reason settle unwinds its own: a flex parent holds the whole gap for
  // as long as the child exists, so the last 2px would land as a jump.
  it('unwinds the strip gap along with the width', () => {
    withMotionSetting(false)
    const css = sidle(chip(), { gap: 2 }).css!
    expect(css(1)).toContain('margin-right:0px')
    expect(css(0)).toContain('margin-right:-2px')
  })

  // No stylesheet in jsdom, so this is the fallback rather than the token — but
  // the point of the check is that a length is asked for by name at all, and
  // that the guard covers this transition too.
  it('runs at the app’s length for a length changing, and stops when asked', () => {
    withMotionSetting(false)
    expect(sidle(chip()).duration).toBe(180)
    withMotionSetting(true)
    expect(sidle(chip()).duration).toBe(0)
  })
})
