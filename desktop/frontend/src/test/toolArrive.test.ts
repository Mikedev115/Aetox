// Rows that land in the same frame, dealt out instead of dropped.
//
// The agent groups its read-only calls and runs them together, so a batch of
// five is an ordinary event and not an edge case. What these pin is the one
// thing the action decides: which rows belong to the same delivery, and where
// in it each of them sits.
import { describe, it, expect } from 'vitest'
import { toolArrive } from '../lib/toolArrive'

const list = () => {
  const el = document.createElement('div')
  document.body.append(el)
  return el
}
const row = (kind = 'tool-step') => {
  const el = document.createElement('div')
  el.className = kind
  return el
}
// MutationObserver delivers on the microtask queue; one turn of the macrotask
// queue is past it either way.
const settled = () => new Promise((done) => setTimeout(done, 0))
const beat = (el: Element) => (el as HTMLElement).style.getPropertyValue('--i')

describe('a batch of rows arriving', () => {
  it('deals out everything that landed together', async () => {
    const el = list()
    const action = toolArrive(el)
    const rows = [row(), row(), row()]
    el.append(...rows)
    await settled()

    // The first of a batch has nothing to wait for; the rest queue behind it.
    expect(beat(rows[0])).toBe('')
    expect(beat(rows[1])).toBe('1')
    expect(beat(rows[2])).toBe('2')
    action.destroy()
  })

  // A row that arrives on its own is the ordinary case and must not be held
  // back: the delay exists to spread a delivery, and one row is not one.
  it('leaves a row that arrived alone alone', async () => {
    const el = list()
    const action = toolArrive(el)
    const one = row()
    el.append(one)
    await settled()

    expect(beat(one)).toBe('')
    action.destroy()
  })

  // Narration rides in the same column as the calls (§59) and takes a beat with
  // them; a search card belongs to the row above it and takes that row's.
  it('counts narration and skips what belongs to a row', async () => {
    const el = list()
    const action = toolArrive(el)
    const first = row()
    const card = row('search-card')
    const note = row('tool-note')
    el.append(first, card, note)
    await settled()

    expect(beat(card)).toBe('')
    expect(beat(note)).toBe('1')
    action.destroy()
  })

  // Eight rows is already the longest any arrival here is allowed to hold the
  // screen. A batch of twenty must not be eleven hundred milliseconds.
  it('stops the wait growing on a long batch', async () => {
    const el = list()
    const action = toolArrive(el)
    const rows = Array.from({ length: 12 }, () => row())
    el.append(...rows)
    await settled()

    expect(beat(rows[7])).toBe('7')
    expect(beat(rows[11])).toBe('7')
    action.destroy()
  })

  // The rows a turn was read back with are already there when this attaches, so
  // a record does not deal itself out — only what arrives after does.
  it('does not deal out rows that were there before it looked', async () => {
    const el = list()
    const old = [row(), row(), row()]
    el.append(...old)
    const action = toolArrive(el)
    await settled()

    expect(old.map(beat)).toEqual(['', '', ''])
    action.destroy()
  })
})
