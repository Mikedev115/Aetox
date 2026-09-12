import { describe, it, expect } from 'vitest'
import { FACE, PANEL, PROP, TOP, isBadge, row } from '../lib/mascot/parts'
import { POSE, ARM, type PoseId } from '../lib/mascot/poses'
import { resolveMascot, mascotSVG, DETAIL_MIN_PX, GLOW_MIN_PX } from '../lib/mascot/rig'
import { palette, SHELL, shellOf } from '../lib/mascot/palette'
import { ROLE, roleOf, roleOptions } from '../lib/mascot/roles'
import { presenceOf, TOOL_POSE } from '../lib/mascot/presence'

// The mascot is a catalogue plus a table plus one drawing function, and the
// rules guarded here are the ones the owner set while watching it drawn — each
// after a screenshot of it being broken. They are what lets a part, a pose or a
// signal be added later without the drawing quietly coming apart again.

const drawn = (o: Parameters<typeof resolveMascot>[0]) => mascotSVG(resolveMascot(o))
const count = (svg: string) => (svg.match(/<(path|rect|circle|ellipse)\b/g) ?? []).length

describe('mascot catalogue', () => {
  it('gives every row a unique id in every slot', () => {
    for (const list of [TOP, FACE, PROP, PANEL]) {
      const ids = list.map((r) => r.id)
      expect(new Set(ids).size).toBe(ids.length)
    }
  })

  // The blueprint's six expressions, by name, in order — a picker shows the
  // identity ones and presence lights the rest. Appending is fine; renaming
  // or reordering these would change what a stored `face:` means.
  it('keeps the six blueprint expressions ahead of anything appended', () => {
    expect(FACE.slice(0, 6).map((f) => f.id)).toEqual(['neutral', 'focused', 'happy', 'thinking', 'excited', 'curious'])
    expect(FACE.filter((f) => f.identity).map((f) => f.id)).toEqual(['neutral', 'focused'])
  })

  // A hand-written profile with a typo lands on a part, never on an error.
  it('falls back by id, never by position', () => {
    expect(row(TOP, 'no-such-thing', 'orb').id).toBe('orb')
    expect(row(FACE, undefined, 'neutral').id).toBe('neutral')
    expect(resolveMascot({ pose: 'no-such-pose' }).poseId).toBe('idle')
    expect(resolveMascot({ badge: 'not-an-icon' }).badgeL).toBe('logo')
  })

  // The badge is the app's own icon vocabulary — the id an agent already
  // declares as `icon:` — or the real logo. Nothing else is a badge.
  it('accepts only ICONS ids or the logo as an ear badge', () => {
    expect(isBadge('search')).toBe(true)
    expect(isBadge('logo')).toBe(true)
    expect(isBadge('avatar.png')).toBe(false)
    expect(isBadge(undefined)).toBe(false)
  })
})

describe('mascot poses', () => {
  it('names only parts that exist', () => {
    for (const [id, pose] of Object.entries(POSE)) {
      if (pose.face) expect(FACE.some((f) => f.id === pose.face), `${id}.face`).toBe(true)
      if (pose.prop && pose.prop !== 'role') expect(PROP.some((p) => p.id === pose.prop), `${id}.prop`).toBe(true)
      if (pose.panel) expect(PANEL.some((p) => p.id === pose.panel), `${id}.panel`).toBe(true)
      if (pose.ground) expect(PANEL.some((p) => p.id === pose.ground), `${id}.ground`).toBe(true)
    }
  })

  // Owner, 12 ก.ย.: "ผมไม่อยากให้มือมันยืด ควรจะมีความยาวที่เราจำกัด" — and then
  // "เอาศอกออกเลย เอาแขนแค่ข้อเดียว". The arm is one capsule of ARM on a ball
  // shoulder; whatever a pose asks for, that is the arm that is drawn.
  it('draws every arm as one fixed capsule, no elbow', () => {
    for (const id of Object.keys(POSE) as PoseId[]) {
      const svg = drawn({ pose: id, size: 76 })
      const arms = svg.match(/<g class="ms-arm [^"]*"/g) ?? []
      expect(arms.length, id).toBe(4) // far + near, two sides
      expect(svg, id).not.toContain('ms-fore')
      const capsules = svg.match(new RegExp(`<rect x="[\\d.]+" y="[\\d.]+" width="${ARM + 4.8}" height="4.8" rx="2.4"`, 'g')) ?? []
      expect(capsules.length, id).toBe(4)
    }
  })

  // A raised hand beside the head sits at the ear's lower edge (the ears span
  // y 22–36) and is drawn in front of it. Above that it collides with the
  // ear the moment the head turns — the second thing the owner sent back.
  it('never raises a hand above the ear line', () => {
    for (const [id, pose] of Object.entries(POSE)) {
      expect(pose.hands.L[1], `${id}.L`).toBeGreaterThanOrEqual(36)
      expect(pose.hands.R[1], `${id}.R`).toBeGreaterThanOrEqual(36)
    }
  })

  // Every pose that lifts a hand beside the head limits how far the head may
  // follow the pointer, or mouse-follow swings that ear into the hand.
  it('limits the look range of every gesture pose', () => {
    for (const [id, pose] of Object.entries(POSE)) {
      const raised = pose.hands.R[0] >= 50 || pose.hands.L[0] <= 14
      if (raised) expect(pose.look, `${id}.look`).toBeLessThanOrEqual(30)
    }
  })
})

describe('mascot drawing', () => {
  it('draws the same markup for the same inputs', () => {
    const a = drawn({ hue: 150, badge: 'search', pose: 'typing', size: 76 })
    const b = drawn({ hue: 150, badge: 'search', pose: 'typing', size: 76 })
    // Gradient ids are per instance; everything else must match.
    expect(a.replace(/ms\d+/g, 'ms')).toBe(b.replace(/ms\d+/g, 'ms'))
  })

  // The colour is an identity. The body's colours in any pose are exactly the
  // body's colours at rest — a pose may add a card in its own colours, it may
  // not tint the person.
  it('never lets a pose change the body\'s colours', () => {
    const body = palette(218)
    const bodyColours = [body.shell, body.shellMid, body.shellEdge, body.primary, body.primaryDn, body.screen, body.eye]
    for (const id of Object.keys(POSE) as PoseId[]) {
      const svg = drawn({ pose: id, size: 76 })
      for (const c of bodyColours) expect(svg, `${id} lost ${c}`).toContain(c)
      expect(svg, `${id} tinted`).not.toMatch(/hsl\((?!218 )/)
    }
  })

  // Only hue reaches the palette; two agents differ by hue and nothing else.
  it('derives every colour from the hue alone', () => {
    const a = drawn({ hue: 30, size: 76 })
    const b = drawn({ hue: 300, size: 76 })
    expect(a.replace(/hsl\(30 /g, 'hsl(H ').replace(/ms\d+/g, 'ms')).toBe(b.replace(/hsl\(300 /g, 'hsl(H ').replace(/ms\d+/g, 'ms'))
  })

  // The ear wears the badge the caller named — the same glyph as the button.
  it('draws the named badge on both ears, or the logo', () => {
    const search = drawn({ badge: 'search', size: 76 })
    expect(search).toContain('<circle cx="11" cy="11" r="8" />')
    const logo = drawn({ size: 76 })
    expect(logo).toContain('M 116.0,742.5')
  })

  // A tile in a roster cannot show a highlight, and it does not turn, so the
  // copies that only exist to be swapped in by a turn are not drawn at all.
  it('drops the far copies and highlights below the detail size', () => {
    const small = drawn({ pose: 'idle', size: DETAIL_MIN_PX - 1 })
    const big = drawn({ pose: 'idle', size: DETAIL_MIN_PX })
    expect(small).not.toContain(' far')
    expect(small).not.toContain('ms-back')
    expect(big).toContain('ms-earL far')
    expect(count(small)).toBeLessThan(count(big))
  })

  // Filters are the one thing that makes eight of these on one page slow, so
  // a roster tile or a chat-row mascot never carries one; the one big mascot
  // the user faces may afford a single blur behind its screen light.
  it('uses a filter only on a big mascot, and only the glow', () => {
    for (const id of Object.keys(POSE) as PoseId[]) {
      expect(drawn({ pose: id, size: GLOW_MIN_PX - 1 }), id).not.toContain('<filter')
      const big = drawn({ pose: id, size: GLOW_MIN_PX })
      expect((big.match(/<filter\b/g) ?? []).length, id).toBe(1)
      expect(big).toContain('feGaussianBlur')
    }
  })

  // The user's own markup never reaches the SVG: a badge is looked up, not
  // interpolated, so an id cannot smuggle an element in.
  it('interpolates nothing a caller typed', () => {
    const svg = drawn({ badge: '<script>', top: '"><b>', face: '<x/>', prop: '</g>', size: 76 })
    expect(svg).not.toContain('<script>')
    expect(svg).not.toContain('"><b>')
    expect(svg).not.toContain('<x/>')
  })
})

describe('mascot finishes and roles', () => {
  // The body's finish is a catalogue row like a part: appended, named, and
  // the first row is the sheet's white robot.
  it('keeps the shells unique and white first', () => {
    const ids = SHELL.map((s) => s.id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(ids[0]).toBe('white')
    expect(shellOf('no-such-finish').id).toBe('white')
    expect(resolveMascot({ shell: 'colour' }).shell).toBe('colour')
  })

  // A finish recolours the body and only the body: the screen stays dark and
  // the light on it stays light, whatever the shell, or the face is lost.
  it('changes the body with the shell, never the screen', () => {
    for (const sh of SHELL) {
      const p = palette(218, sh.id)
      expect(p.screen).toBe(palette(218).screen)
      expect(p.eye).toBe(palette(218).eye)
      const svg = drawn({ shell: sh.id, size: 76 })
      expect(svg).toContain(p.shell)
    }
    expect(palette(218, 'colour').shell).not.toBe(palette(218, 'white').shell)
  })

  // A role is a named set of slots — the sheet's two robots are two rows —
  // and an agent's icon lands on the ears over whichever role it starts from.
  it('fills the slots from a role, then the icon, then the caller', () => {
    expect(new Set(ROLE.map((r) => r.id)).size).toBe(ROLE.length)
    expect(roleOf('nope').id).toBe('assistant')
    expect(roleOptions('code', undefined)).toMatchObject({ top: 'chevrons', badge: 'terminal', face: 'focused', prop: 'laptopTerm' })
    expect(roleOptions('assistant', 'search')).toMatchObject({ badge: 'search', badgeR: 'search', top: 'orb' })
    expect(roleOptions('assistant', 'not-an-icon')).toMatchObject({ badge: 'logo' })
    expect(roleOptions('code', 'search', { top: 'none', shell: 'dark' })).toMatchObject({ top: 'none', badge: 'search', shell: 'dark', prop: 'laptopTerm' })
  })
})

describe('presence', () => {
  it('lets the user\'s own actions win', () => {
    expect(presenceOf({ awaiting: true, running: ['read'], mic: true })).toBe('listening')
    expect(presenceOf({ awaiting: false, speaking: true })).toBe('answering')
    expect(presenceOf({ awaiting: true, streaming: true, asking: true })).toBe('asking')
  })

  it('reads the innermost running tool', () => {
    expect(presenceOf({ awaiting: true, running: ['task', 'web_search'] })).toBe('research')
    expect(presenceOf({ awaiting: true, running: ['search'] })).toBe('searchFiles')
    expect(presenceOf({ awaiting: true, running: ['read'] })).toBe('reading')
    expect(presenceOf({ awaiting: true, running: ['shell'] })).toBe('coding')
    expect(presenceOf({ awaiting: true, running: ['something_new'] })).toBe('typing')
  })

  it('answers while prose streams, thinks before it, rests after', () => {
    expect(presenceOf({ awaiting: true, streaming: true })).toBe('answering')
    expect(presenceOf({ awaiting: true, reasoning: true })).toBe('thinking')
    expect(presenceOf({ awaiting: true, status: 'กำลังคิดคำตอบ...' })).toBe('thinking')
    expect(presenceOf({ awaiting: false, justDone: true })).toBe('success')
    expect(presenceOf({ awaiting: false })).toBe('idle')
  })

  it('maps every tool to a pose that exists', () => {
    for (const [tool, pose] of Object.entries(TOOL_POSE)) expect(pose in POSE, tool).toBe(true)
  })
})
