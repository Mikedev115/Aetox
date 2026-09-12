import { describe, it, expect } from 'vitest'
import { POSE } from '../lib/mascot/poses'
import {
  PHASES,
  WALK_STEP,
  PAD,
  BOX,
  FRAME_ICONS,
  allFrames,
  firstFrames,
  keyOf,
  parseKey,
  momentOf,
  loopOf,
  rigVersion,
  setHash,
  snapshotSVG,
} from '../lib/mascot/bake'

// The desktop body draws the mascot from pictures the app bakes (bake.ts).
// What is guarded here is the contract between the two sides — which frames
// exist, what they are called, what a picture is a picture of — and that a
// snapshot is a standalone drawing with nothing left in it for a stylesheet
// to do. The pictures themselves need a real engine and are looked at in
// the app, not here.

describe('the frame set', () => {
  it('covers every pose, one blink each, the walk at every heading, and the frame icons', () => {
    const frames = allFrames()
    const keys = frames.map(keyOf)
    expect(new Set(keys).size).toBe(keys.length)
    const poses = Object.keys(POSE).filter((p) => p !== 'walk')
    for (const p of poses) {
      for (let phase = 0; phase < PHASES; phase++) expect(keys).toContain(`${p}-p${phase}-open`)
      expect(keys).toContain(`${p}-p0-shut`)
      expect(keys).not.toContain(`${p}-p1-shut`)
    }
    for (let turn = -180 + WALK_STEP; turn <= 180; turn += WALK_STEP) expect(keys).toContain(`walk-t${turn}-p${PHASES - 1}`)
    for (const icon of FRAME_ICONS) expect(keys).toContain(`icon-${icon}`)
    expect(frames.length).toBe(poses.length * (PHASES + 1) + (360 / WALK_STEP) * PHASES + FRAME_ICONS.length)
  })

  it('names frames the way the Go side composes them, and reads its own names back', () => {
    for (const f of allFrames()) expect(parseKey(keyOf(f))).toEqual(f)
    expect(parseKey('walk-p0-open')).toBeNull()
    expect(parseKey('nothing-p0-open')).toBeNull()
    expect(parseKey('icon-bogus')).toBeNull()
    expect(parseKey('../x')).toBeNull()
  })

  it('puts the pose it is in and the rest pose first, once each, with the icons', () => {
    const first = firstFrames('thinking').map(keyOf)
    expect(first.slice(0, PHASES)).toEqual([0, 1, 2, 3].map((p) => `thinking-p${p}-open`))
    expect(first).toContain('idle-p0-shut')
    expect(first).toContain('icon-x')
    expect(new Set(first).size).toBe(first.length)
    // walk is the drag's, never a first frame; an unknown pose is just idle
    expect(firstFrames('walk')).toEqual(firstFrames('nothing'))
  })
})

describe('what a phase is a picture of', () => {
  it('samples a loop evenly over its period, from its offset', () => {
    expect(momentOf('idle', 0)).toBe(0)
    expect(momentOf('idle', 2)).toBeCloseTo(1.8)
    expect(momentOf('walk', 1)).toBeCloseTo(0.155)
    // the sleeper is only asleep once it has lain down (ms-lie, .9s forwards)
    expect(momentOf('recharge', 0)).toBeCloseTo(0.9)
    expect(loopOf('greeting').period).toBeCloseTo(1.4)
  })
})

describe('the set hash', () => {
  it('changes with the look and with the scale, and is filename-safe', () => {
    const a = setHash({}, 1.75)
    expect(a).toMatch(/^[0-9a-f]{8}-175$/)
    expect(setHash({}, 1)).not.toBe(a)
    expect(setHash({ shell: 'dark' }, 1.75)).not.toBe(a)
    expect(rigVersion({})).toBe(rigVersion({}))
  })
})

describe('a snapshot', () => {
  it('is a standalone SVG document padded around the figure, with no class left to style', () => {
    const svg = snapshotSVG({}, { kind: 'pose', pose: 'idle', phase: 0, blink: 'open' })
    expect(svg.startsWith('<svg')).toBe(true)
    expect(svg).toContain('xmlns="http://www.w3.org/2000/svg"')
    expect(svg).toContain(`viewBox="${-PAD} ${-PAD} ${BOX} ${BOX}"`)
    expect(svg).toContain('<path')
    expect(svg).not.toContain('class="mascot')
    // and leaves nothing behind in the page it was drawn in
    expect(document.querySelector('.mascot')).toBeNull()
  })

  it("draws the walk at the heading asked for, not the pose table's rest turn", () => {
    const a = snapshotSVG({}, { kind: 'walk', turn: 90, phase: 0 })
    const b = snapshotSVG({}, { kind: 'walk', turn: -90, phase: 0 })
    expect(a).not.toBe(b)
  })

  it('draws an icon from the app\'s own set', () => {
    const svg = snapshotSVG({}, { kind: 'icon', icon: 'x' })
    expect(svg).toContain('viewBox="0 0 24 24"')
    expect(svg).toContain('<path d="M18 6 6 18"')
  })
})
