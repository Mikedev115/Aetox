import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/svelte'
import AgentMascot from '../lib/mascot/AgentMascot.svelte'
import { lookOf, hueOf } from '../lib/mascot/agentLook'
import { coverHue } from '../lib/coverHue'

// The agent's mascot: the cartoon person's contract (name, icon, size, off,
// hue, state) on the robot. What is guarded is the seam — a profile row
// becomes the same props on every surface, a card's word becomes a pose, and
// a tile in a roster does not move while the one that is working does.

describe('lookOf', () => {
  it('carries the icon, the colour and the identity dials, parsed', () => {
    expect(lookOf({ icon: 'zap', hue: '210', accent: 'copper', shell: 'dark', top: 'bar', face: 'focused' }))
      .toEqual({ icon: 'zap', hue: 210, accent: 'copper', shell: 'dark', top: 'bar', face: 'focused' })
  })

  // Blank is the ordinary case — a profile nobody opened — and blank must stay
  // absent, so the mascot's own defaults apply rather than an empty string.
  it('leaves out what the file did not say', () => {
    expect(lookOf({ icon: '', hue: '', shell: ' ', top: undefined })).toEqual({})
    expect(lookOf(undefined)).toEqual({})
  })

  // A hand-written number that is not a hue lands on the derived colour,
  // never on grey: undefined, not NaN and not 0.
  it('refuses a hue that is not 0..360', () => {
    expect(hueOf('0')).toBe(0)
    expect(hueOf('360')).toBe(0)
    expect(hueOf(400)).toBe(40)
    expect(hueOf(-30)).toBe(330)
    for (const bad of ['', ' ', 'abc', '361', '1000', '-1', '12.5', '0x10']) expect(hueOf(bad), bad).toBeUndefined()
    expect(lookOf({ hue: 'red' }).hue).toBeUndefined()
  })

  // The cartoon's wardrobe is read no more; a file that still names a haircut
  // is not an error and not a prop.
  it('ignores hair and accessory', () => {
    expect(lookOf({ hair: 'bob', accessory: 'glasses' } as any)).toEqual({})
  })
})

describe('AgentMascot', () => {
  const mascot = (props: Record<string, unknown>) => render(AgentMascot, { name: 'deck', ...props } as any).container.querySelector('.mascot')!

  it('wears the badge the profile names, or the logo', () => {
    const ear = (props: Record<string, unknown>) => mascot(props).querySelector('.ms-earL .ms-badge')!.innerHTML
    expect(ear({ icon: 'search' })).toContain('<circle cx="11" cy="11" r="8">')
    expect(ear({})).toContain('M 116.0,742.5')
    expect(ear({ icon: 'not-an-icon' })).toContain('M 116.0,742.5')
  })

  // The colour is the agent's: off the name unless its file says otherwise,
  // and a string from the file is as good as a number.
  it('takes its hue from the name, or from the file', () => {
    expect(mascot({}).innerHTML).toContain(`hsl(${coverHue('deck')} `)
    expect(mascot({ hue: 150 }).innerHTML).toContain('hsl(150 ')
    expect(mascot({ hue: '150' }).innerHTML).toContain('hsl(150 ')
    expect(mascot({ hue: 'nope' }).innerHTML).toContain(`hsl(${coverHue('deck')} `)
  })

  // An accent named in the file (or handed over from a persona) beats the
  // name's hue; a degree in the file still beats the accent (rig.ts).
  it('wears a named accent over the hue of the name, and a degree over both', () => {
    expect(mascot({ accent: 'copper' }).innerHTML).toContain('hsl(22 ')
    expect(mascot({ accent: 'copper' }).innerHTML).not.toContain(`hsl(${coverHue('deck')} `)
    expect(mascot({ accent: 'copper', hue: 150 }).innerHTML).toContain('hsl(150 ')
  })

  it('speaks the card\'s five words as poses', () => {
    expect(mascot({}).classList.contains('pose-idle')).toBe(true)
    expect(mascot({ state: 'think' }).classList.contains('pose-thinking')).toBe(true)
    expect(mascot({ state: 'work' }).classList.contains('pose-typing')).toBe(true)
    expect(mascot({ state: 'done' }).classList.contains('pose-success')).toBe(true)
    expect(mascot({ state: 'err' }).classList.contains('pose-error')).toBe(true)
  })

  // A roster tile is still; the worker whose job is alive moves — the one
  // thing the cartoon person did that the owner asked to keep (7 ก.ย.).
  it('is still unless the job is alive, or told otherwise', () => {
    expect(mascot({}).classList.contains('still')).toBe(true)
    expect(mascot({ state: 'done' }).classList.contains('still')).toBe(true)
    expect(mascot({ state: 'err' }).classList.contains('still')).toBe(true)
    expect(mascot({ state: 'think' }).classList.contains('still')).toBe(false)
    expect(mascot({ state: 'work' }).classList.contains('still')).toBe(false)
    expect(mascot({ state: 'work', still: true }).classList.contains('still')).toBe(true)
    expect(mascot({ still: false }).classList.contains('still')).toBe(false)
  })

  it('drains one the assistant may not hand work to', () => {
    expect(mascot({ off: true }).classList.contains('off')).toBe(true)
    expect(mascot({}).classList.contains('off')).toBe(false)
  })

  // The identity dials reach the drawing: a dark shell is a different body
  // colour, and the size sets the box.
  it('passes shell and size through', () => {
    const dark = mascot({ shell: 'dark', size: 22 })
    expect(dark.innerHTML).not.toBe(mascot({ size: 22 }).innerHTML)
    expect((dark as HTMLElement).style.width).toBe('22px')
    expect(dark.classList.contains('lite')).toBe(true)
  })
})
