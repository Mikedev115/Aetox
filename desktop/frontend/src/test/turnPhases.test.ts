import { describe, it, expect } from 'vitest'
import { phasesOf } from '../lib/turnPhases'
import type { ToolStep } from '../lib/types'

const tool = (label: string, extra: Partial<ToolStep> = {}): ToolStep =>
  ({ label, state: 'done', startedAt: 0, secs: 1, ...extra }) as ToolStep
const note = (label: string): ToolStep => ({ label, kind: 'note', state: 'done', startedAt: 0 }) as ToolStep
const thinking = (secs: number): ToolStep => ({ label: '', kind: 'thinking', state: 'done', startedAt: 0, secs }) as ToolStep
const asked = (label: string, time = '19:03'): ToolStep => ({ label, kind: 'asked', time, state: 'done', startedAt: 0 }) as ToolStep

describe('phasesOf', () => {
  // The shape the window used to throw away: four stretches of work, each with
  // its own sentence and its own clock, flattened into one answer over two
  // summed numbers.
  it('starts a new phase at every sentence the model wrote', () => {
    const phases = phasesOf([
      thinking(41),
      note('ขอไล่ดูตรงที่ประกอบ path ก่อนครับ'),
      tool('grep placedWrite'),
      tool('read place.go'),
      thinking(96),
      note('เจอแล้ว เดี๋ยวแก้'),
      tool('edit place.go'),
      thinking(54),
      note('แก้แล้วครับ'),
    ])

    expect(phases.map((p) => p.say)).toEqual([
      'ขอไล่ดูตรงที่ประกอบ path ก่อนครับ',
      'เจอแล้ว เดี๋ยวแก้',
      'แก้แล้วครับ',
    ])
    expect(phases.map((p) => p.thinkSecs)).toEqual([41, 96, 54])
    expect(phases.map((p) => p.steps.length)).toEqual([2, 1, 0])
  })

  // The number on the old lump has to still be derivable from the pieces, or
  // the split is a different measurement rather than the same one told
  // honestly. Owner's own turn: 41+96+84+54 = 275, 9+14+8+5 = 36.
  it('keeps the totals the collapsed row used to show', () => {
    const phases = phasesOf([
      thinking(41), note('a'), ...Array.from({ length: 9 }, (_, i) => tool(`t${i}`)),
      thinking(96), note('b'), ...Array.from({ length: 14 }, (_, i) => tool(`u${i}`)),
      thinking(84), note('c'), ...Array.from({ length: 8 }, (_, i) => tool(`v${i}`)),
      thinking(54), note('d'), ...Array.from({ length: 5 }, (_, i) => tool(`w${i}`)),
    ])
    expect(phases.reduce((n, p) => n + p.thinkSecs, 0)).toBe(275)
    expect(phases.reduce((n, p) => n + p.steps.length, 0)).toBe(36)
  })

  // A delegate's rows arrive while the main agent has moved on to the next
  // sentence. Filed by arrival they would sit under a stretch of work they had
  // nothing to do with.
  it('files a sub-agent’s rows under the phase that hired it', () => {
    const phases = phasesOf([
      note('ให้เอเจนไปหาให้'),
      tool('task start', { ref: 'call-1', delegation: true }),
      note('ระหว่างรอ ขอดูโค้ดเอง'),
      tool('read app.go'),
      tool('grep foo', { parent: 'call-1' }),
    ])
    expect(phases).toHaveLength(2)
    expect(phases[0].steps.map((s) => s.label)).toEqual(['task start', 'grep foo'])
    expect(phases[1].steps.map((s) => s.label)).toEqual(['read app.go'])
  })

  // The live half. Prose still arriving is a phase, not a bubble the next tool
  // call erases.
  it('makes the still-streaming sentence a trailing open phase', () => {
    const phases = phasesOf([note('รอบแรก'), tool('read a.go')], 'กำลังจะแก้ตรงนี้')
    expect(phases.map((p) => p.streaming)).toEqual([false, true])
    expect(phases.at(-1)?.say).toBe('กำลังจะแก้ตรงนี้')
  })

  // Thought about, said nothing yet. The seconds must not be quietly added to
  // the stretch above, which finished before that thinking started.
  it('does not backdate thinking onto the phase before it', () => {
    const phases = phasesOf([note('รอบแรก'), tool('read a.go'), thinking(12)])
    expect(phases).toHaveLength(2)
    expect(phases[0].thinkSecs).toBe(0)
    expect(phases[1]).toMatchObject({ say: '', thinkSecs: 12, streaming: false })
  })

  // An answer an interjection re-placed is prose like any other prose now.
  it('treats a demoted answer as an ordinary phase', () => {
    const phases = phasesOf([
      { label: '## เสร็จแล้ว', kind: 'said', state: 'done', startedAt: 0 } as ToolStep,
      note('อ๋อ เดี๋ยวทำต่อ'),
    ])
    expect(phases.map((p) => p.say)).toEqual(['## เสร็จแล้ว', 'อ๋อ เดี๋ยวทำต่อ'])
  })

  it('opens a phase for a turn that called a tool without saying anything', () => {
    const phases = phasesOf([tool('read a.go')])
    expect(phases).toHaveLength(1)
    expect(phases[0]).toMatchObject({ say: '', thinkSecs: 0 })
  })

  it('has no phases at all for a turn that did nothing', () => {
    expect(phasesOf([])).toEqual([])
  })
})

// The user speaking is a boundary too, and the strongest one a turn has:
// everything after it happened because of it. Before PartAsked there was no row
// for it at all — the window moved the bubble below the live block with CSS,
// which reads right for one interruption and wrong for the second, because
// everything the model says during a turn is inside that one block.
describe('phasesOf with the user cutting in', () => {
  it('opens a phase at the message and gives it the answer that follows', () => {
    const phases = phasesOf([
      thinking(8),
      note('ครับ ขอไล่ดูก่อน'),
      tool('grep placedWrite'),
      asked('ผมหมายถึงตัวโปรแกรมครับ', '19:03'),
      thinking(23),
      note('อ๋อ เข้าใจแล้วครับ'),
    ])

    expect(phases).toHaveLength(2)
    expect(phases[0].say).toBe('ครับ ขอไล่ดูก่อน')
    expect(phases[0].asked).toBeUndefined()
    // The message opens the phase and the model's reply IS that phase's prose —
    // not a third stretch. They are one exchange.
    expect(phases[1].asked).toBe('ผมหมายถึงตัวโปรแกรมครับ')
    expect(phases[1].askedTime).toBe('19:03')
    expect(phases[1].say).toBe('อ๋อ เข้าใจแล้วครับ')
    // The thinking happened between the two, so it belongs to the phase that
    // answers rather than to the one that was interrupted.
    expect(phases[0].thinkSecs).toBe(8)
    expect(phases[1].thinkSecs).toBe(23)
  })

  // Two messages a minute apart are two messages. Folded into one phase they
  // would share a bubble and a timestamp, and the second minute would be gone.
  it('keeps two messages typed in a row as two phases', () => {
    const phases = phasesOf([
      note('กำลังทำครับ'),
      asked('ไม่ใช่แบบนั้นน', '19:03'),
      asked('คือ iGPU ผมกากนั่นแหละ', '19:04'),
    ])

    expect(phases.map((p) => p.asked)).toEqual([undefined, 'ไม่ใช่แบบนั้นน', 'คือ iGPU ผมกากนั่นแหละ'])
    expect(phases.map((p) => p.askedTime)).toEqual([undefined, '19:03', '19:04'])
  })

  // The one thing a new boundary must not do: take a delegate's rows away from
  // the `task` call that hired it. A sub-agent goes on working while the user
  // types and while the model answers them, so its rows arrive AFTER the
  // interruption and belong to a phase that opened before it. homeOfRef is what
  // keeps them there, and a phase boundary must not be able to move them.
  it("leaves a sub-agent's rows with the task row that hired it", () => {
    const phases = phasesOf([
      note('ขอส่งให้เอเจนเขียนก่อน'),
      tool('task start', { ref: 'call-1', delegation: true }),
      asked('เปลี่ยนเป็นภาษาไทยด้วยนะ'),
      note('ได้ครับ เดี๋ยวบอกให้'),
      tool('grep foo', { parent: 'call-1' }),
      tool('write out.md', { parent: 'call-1' }),
    ])

    expect(phases).toHaveLength(2)
    expect(phases[0].steps.map((s) => s.label)).toEqual(['task start', 'grep foo', 'write out.md'])
    expect(phases[1].steps).toEqual([])
    expect(phases[1].asked).toBe('เปลี่ยนเป็นภาษาไทยด้วยนะ')
  })

  // A message that lands while nothing else is open must not inherit the phase
  // above it, and must not be dropped for having no prose of its own.
  it('stands alone when the model says nothing after it', () => {
    const phases = phasesOf([asked('เอาแบบนี้แหละ')])
    expect(phases).toHaveLength(1)
    expect(phases[0].asked).toBe('เอาแบบนี้แหละ')
    expect(phases[0].say).toBe('')
  })
})
