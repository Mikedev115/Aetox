import { describe, it, expect } from 'vitest'
import {
  HAIR, HAIR_DERIVED, ACCESSORY, ACCESSORY_CHOICES, ACCESSORY_DERIVED,
  HUES, PROP, PROP_ICONS, PROP_MIN_PX, faceOf, resolveFace, faceSVG,
} from '../lib/agentFace'

// The wardrobe is a catalogue a profile may point into by name, which makes
// these ids part of the file format rather than an implementation detail. What
// is guarded here is the small set of rules that lets a part be ADDED later
// without repainting anybody or breaking a file somebody already wrote.
describe('agent face wardrobe', () => {
  // A duplicate id makes `hair: bob` ambiguous, and the loser would be
  // whichever one happens to sit later in the array.
  it('gives every part a unique id', () => {
    const hair = HAIR.map((h) => h.id)
    const acc = ACCESSORY.map((a) => a.id)
    expect(new Set(hair).size).toBe(hair.length)
    expect(new Set(acc).size).toBe(acc.length)
  })

  // The whole reason a face can be derived instead of stored. If this ever
  // stops holding, an agent's face changes under them between two launches.
  it('draws the same face for a name every time', () => {
    const a = faceSVG(resolveFace('research', 88, { icon: 'search' }))
    const b = faceSVG(resolveFace('research', 88, { icon: 'search' }))
    expect(a).toBe(b)
  })

  // Two hashes, not one — sharing coverHue would tie every agent of a colour to
  // the same haircut. This is the cheapest observable proof they are separate.
  it('varies hair independently of hue', () => {
    const shapes = new Set(['doc', 'github', 'sheet', 'editor'].map((n) => resolveFace(n, 88).hair.id))
    expect(shapes.size).toBeGreaterThan(1)
  })

  // These fields arrive from a .md written by hand by somebody who cannot see
  // this file. A typo has to land on the derived face, never on a blank square.
  it('falls back to the derived part when a profile names one that does not exist', () => {
    const typo = resolveFace('research', 88, { hair: 'mohawk' })
    expect(typo.hair.id).toBe(resolveFace('research', 88).hair.id)
    expect(faceSVG(typo)).toContain('<ellipse')
  })

  it('honours a part the profile does name', () => {
    expect(resolveFace('research', 88, { hair: 'beanie' }).hair.id).toBe('beanie')
    expect(resolveFace('research', 88, { accessory: 'headphones' }).accessory.id).toBe('headphones')
  })

  // The override exists so the seven bundled agents can be spread across the
  // wheel by hand; coverHue clusters them, and two of them land four degrees
  // apart. Nothing about coverHue changes for anyone who does not set this.
  it('lets a profile choose its own hue', () => {
    expect(resolveFace('research', 88, { hue: 28 }).hue).toBe(28)
    expect(resolveFace('research', 88).hue).toBe(resolveFace('research', 24).hue)
  })

  // Below the threshold the prop is a few pixels of noise beside the head, and
  // the row it sits in is the @ menu, where the name is doing the work anyway.
  it('drops the prop below the small size', () => {
    expect(resolveFace('research', PROP_MIN_PX, { icon: 'search' }).prop).toBe('search')
    expect(resolveFace('research', PROP_MIN_PX - 1, { icon: 'search' }).prop).toBe('')
  })

  // An icon no prop is drawn for is the ordinary case, not an error: `icon:` is
  // the profile's MARK and this map only covers the ones worth holding.
  it('draws no prop for an icon it has none for', () => {
    expect(resolveFace('research', 88, { icon: 'userRound' }).prop).toBe('')
    expect(Object.keys(PROP).length).toBeGreaterThan(0)
  })

  // The bug this guards is the one the editor shipped with for a month: it kept
  // its own hand-typed list of fifteen marks to offer, this map drew seven, and
  // the overlap was four — so eleven of the fifteen choices changed nothing at
  // all on screen and the page said so nowhere. One list now, and every entry on
  // it has to make a visible difference.
  it('offers exactly the marks it can draw, and every one of them changes the face', () => {
    expect(PROP_ICONS).toEqual(Object.keys(PROP))
    const bare = faceSVG(resolveFace('x', 88))
    for (const icon of PROP_ICONS) {
      expect(faceSVG(resolveFace('x', 88, { icon })), icon).not.toBe(bare)
    }
  })

  // `bot` is what a profile naming no icon resolves to (desktop/office.go,
  // chairIcon). A prop for it would put the same object in the hands of every
  // agent whose owner never chose one, which is the opposite of what an unset
  // field means.
  it('leaves the mark an unstyled profile gets empty-handed', () => {
    expect(PROP_ICONS).not.toContain('bot')
    expect(resolveFace('x', 88, { icon: 'bot' }).prop).toBe('')
  })

  // The picker's own list, and the one place the catalogue is not offered
  // whole: `none` sits in ACCESSORY twice so half a roster wears nothing, and a
  // picker drawing "ไม่มี" twice looks like a bug rather than like weighting.
  it('offers each accessory once, without losing the weighting duplicate', () => {
    expect(ACCESSORY_CHOICES.some((a) => a.id === 'none2')).toBe(false)
    expect(ACCESSORY_CHOICES.map((a) => a.id)).toEqual([...new Set(ACCESSORY_CHOICES.map((a) => a.id))])
    expect(ACCESSORY.length).toBeGreaterThan(ACCESSORY_CHOICES.length)
  })

  // **The numbers below are the roster's identity, not a fact about the file.**
  //
  // `pick` chooses an index, so these two lengths are inputs to every derived
  // face. Change one and every agent that never chose a haircut gets a new one
  // on the next launch — the seven that ship included. That is what `pickOnly`
  // exists to prevent, and this is the test that notices when a new part
  // forgets the flag: parts may be added forever, this pool may not move.
  it('keeps the derived pool frozen at what the roster was built from', () => {
    expect(HAIR_DERIVED.length).toBe(8)
    expect(ACCESSORY_DERIVED.length).toBe(4)
    expect(HAIR.length).toBeGreaterThan(HAIR_DERIVED.length)
    expect(HAIR_DERIVED.every((h) => !h.pickOnly)).toBe(true)
  })

  // The other half of that: a part added after the roster is still choosable by
  // name, on every surface, or the picker would be offering a face the file
  // cannot hold.
  it('honours a part the hash can never land on', () => {
    expect(resolveFace('research', 88, { hair: 'headscarf' }).hair.id).toBe('headscarf')
    expect(resolveFace('research', 88, { accessory: 'beard' }).accessory.id).toBe('beard')
  })

  // One reader for the four strings a profile writes, because seven surfaces
  // draw an agent and a second reading of the same fields is how one person
  // ends up with two faces. A hue is the only one that needs converting, and a
  // hand-written file is exactly where a bad one comes from.
  it('reads a profile row into overrides, and refuses a hue that is not one', () => {
    expect(faceOf({ icon: 'zap', hair: 'bob', accessory: 'glasses', hue: '210' }))
      .toEqual({ icon: 'zap', hair: 'bob', accessory: 'glasses', hue: 210 })
    expect(faceOf({ hue: '0' }).hue).toBe(0)
    expect(faceOf({ hue: '360' }).hue).toBe(0)
    for (const bad of ['', '  ', 'blue', '-30', '400', '12.5', '1e2']) {
      expect(faceOf({ hue: bad }).hue, bad).toBeUndefined()
    }
    expect(faceOf(undefined).icon).toBeUndefined()
  })

  // A colour the picker offers has to be a colour the reader accepts — the same
  // "one list" rule PROP_ICONS is built on, one row down the form.
  it('offers only hues the reader takes back', () => {
    for (const h of HUES) expect(faceOf({ hue: String(h) }).hue, String(h)).toBe(h)
  })

  // Every part combination has to produce a person. A hair that forgot to draw
  // the head, or a size that dropped the body, would ship as an empty tile on
  // somebody's roster and nowhere else.
  it('draws a whole person for every part in the catalogue', () => {
    for (const h of HAIR) {
      for (const a of ACCESSORY) {
        const svg = faceSVG(resolveFace('x', 88, { hair: h.id, accessory: a.id, icon: 'search' }))
        expect(svg, `${h.id}/${a.id}`).toContain('<ellipse')
        expect(svg, `${h.id}/${a.id}`).toContain('#14161a')
      }
    }
  })
})

// What the agent is DOING, which is the only thing here that is not the person.
//
// The face had eyes and a mouth and nothing else, so the mouth carried every
// emotion alone — and "concentrating" came out as an O, which is the shape
// every emoji set uses for *startled* (owner, 6 ก.ย.: "ไม่เอาอ้าปากแบบนั้นดิ").
// Working is now a scene: brows, a grin, a laptop big enough to change the
// tile's silhouette, and the screen's light on the jaw.
describe('an agent that is working', () => {
  const idle = (o = {}) => faceSVG(resolveFace('นักวิจัย', 76, o))
  const busy = (o = {}) => faceSVG(resolveFace('นักวิจัย', 76, { state: 'work', ...o }))

  // The roster is full of agents nobody has hired. None of them may change.
  it('leaves a face with no state exactly as it was', () => {
    const s = idle()
    expect(s).not.toContain('af-kit')
    expect(s).not.toContain('af-glow')
    expect(s).not.toContain('clipPath')
    // The brows belong to the expression, not to the person.
    expect(s.match(/stroke-linecap="round"/g) ?? []).toHaveLength(1)
  })

  it('opens a laptop and lights the face', () => {
    const s = busy()
    expect(s).toContain('af-kit')
    expect(s).toContain('af-glow')
    expect(s).toContain('clipPath')
  })

  // A round highlight is a sparkle; a rectangular one is the screen itself,
  // reflected. It is the cheapest thing on this face and the one that says
  // where the agent is looking.
  it('puts the screen in its eyes as a rectangle, not a dot', () => {
    expect(busy()).toContain('<rect x="24.6" y="26.1"')
    expect(idle()).not.toContain('<rect x="24.6"')
  })

  // The prop's box is x 40..62, y 40..64 — which is where the machine now is.
  it('puts the identity prop down while the hands are busy', () => {
    expect(resolveFace('research', 88, { icon: 'search' }).prop).toBe('search')
    expect(resolveFace('research', 88, { icon: 'search', state: 'work' }).prop).toBe('')
  })

  // The glasses accessory is two r=6 lenses centred on the eyes, so its rim
  // tops out at y=21. A brow on the comfortable line sat on the frame and
  // vanished; a brow in the hair's own colour vanished into a fringe. Both
  // failures were silent, and both are one number.
  it('keeps the brows clear of the glasses and of the hair colour', () => {
    const s = busy({ accessory: 'glasses' })
    expect(s).toContain('M23.4 20.2')
    expect(s).toContain('M34.2 19.8')
    // Drawn in the eyes' near-black, never in p.dark, which is what a fringe
    // is made of.
    expect(s).toContain('d="M23.4 20.2q3.2-2.6 6.4-.4" stroke="#14161a"')
  })

  // Rule 3 at the top of agentFace.ts, and it has to survive the new field:
  // the same agent in the same state draws the same markup, twice in one page
  // included — which is what makes the jaw clip's shared id safe.
  it('stays deterministic, and clips the jaw to the head it was cut for', () => {
    expect(busy()).toBe(busy())
    const small = faceSVG(resolveFace('x', 76, { state: 'work', hair: 'curly' }))
    const big = faceSVG(resolveFace('x', 76, { state: 'work', hair: 'sidePart' }))
    const idOf = (s: string) => s.match(/<clipPath id="([^"]+)"/)?.[1]
    expect(idOf(small)).toBeTruthy()
    expect(idOf(small)).not.toBe(idOf(big))
  })

  // Every hair in the catalogue has to survive being sat behind a laptop.
  it('draws a whole working person for every haircut', () => {
    for (const h of HAIR) {
      const s = faceSVG(resolveFace('x', 76, { hair: h.id, accessory: 'glasses', state: 'work' }))
      expect(s, h.id).toContain('af-kit')
      expect(s, h.id).toContain('M23.4 20.2')
    }
  })
})
