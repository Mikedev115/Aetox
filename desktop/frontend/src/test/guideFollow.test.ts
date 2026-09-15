// The ring stays on the button, or it is not a ring around anything.
//
// The guide measures its target when it arrives, and a page that has just
// opened is exactly where boxes keep moving afterwards. This is the watch that
// replaces that one measurement (owner, 15 ก.ย. 2026, screenshot: a ring around
// empty space on a page the guide had genuinely reached).
import { describe, it, expect, beforeEach } from 'vitest'
import { watchBox, type Box } from '../lib/guide/follow'

const settle = () => new Promise((r) => setTimeout(r, 40))

function boxed(box: Box): HTMLElement {
  const el = document.createElement('div')
  el.getBoundingClientRect = () => ({ ...box, x: box.left, y: box.top, toJSON() {} }) as DOMRect
  document.body.appendChild(el)
  return el
}

const at = (left: number, top: number, width = 40, height = 20): Box => ({
  left,
  top,
  right: left + width,
  bottom: top + height,
  width,
  height,
})

beforeEach(() => {
  document.body.innerHTML = ''
})

describe('following the box it points at', () => {
  it('reports the new box when the page moves under it', async () => {
    const el = boxed(at(10, 10))
    const moves: Box[] = []
    const off = watchBox(el, (b) => moves.push(b), () => {})

    // A card above it finished loading: same button, 200px lower.
    el.getBoundingClientRect = () => ({ ...at(10, 210), x: 10, y: 210, toJSON() {} }) as DOMRect
    document.dispatchEvent(new Event('scroll'))
    await settle()

    expect(moves.map((m) => m.top)).toEqual([210])
    off()
  })

  it('says nothing about sub-pixel drift — a permanent flinch is worse than a pixel', async () => {
    const el = boxed(at(10, 10))
    let moved = 0
    const off = watchBox(el, () => moved++, () => {})

    el.getBoundingClientRect = () => ({ ...at(10.4, 10.6), x: 10.4, y: 10.6, toJSON() {} }) as DOMRect
    document.dispatchEvent(new Event('scroll'))
    await settle()

    expect(moved).toBe(0)
    off()
  })

  it('a box with no size is the target LEAVING, not a box at the origin', async () => {
    const el = boxed(at(10, 10))
    const moves: Box[] = []
    let gone = 0
    const off = watchBox(el, (b) => moves.push(b), () => gone++)

    // The menu it lived in closed. No sign changed, no page changed.
    el.getBoundingClientRect = () => ({ ...at(0, 0, 0, 0), x: 0, y: 0, toJSON() {} }) as DOMRect
    document.dispatchEvent(new Event('scroll'))
    await settle()

    expect(gone).toBe(1)
    expect(moves).toEqual([]) // never reported as a move to (0,0)
    off()
  })

  it('an element taken out of the document has left, whatever its last rect said', async () => {
    const el = boxed(at(10, 10))
    let gone = 0
    const off = watchBox(el, () => {}, () => gone++)

    el.remove()
    document.dispatchEvent(new Event('scroll'))
    await settle()

    expect(gone).toBe(1)
    off()
  })

  it('says nothing more once it is let go of', async () => {
    const el = boxed(at(10, 10))
    let moved = 0
    let gone = 0
    const off = watchBox(el, () => moved++, () => gone++)
    off()

    el.getBoundingClientRect = () => ({ ...at(400, 400), x: 400, y: 400, toJSON() {} }) as DOMRect
    document.dispatchEvent(new Event('scroll'))
    window.dispatchEvent(new Event('resize'))
    await settle()

    expect(moved + gone).toBe(0)
  })
})
