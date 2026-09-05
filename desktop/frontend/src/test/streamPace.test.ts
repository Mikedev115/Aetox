// What the pacer must guarantee, driven on a clock this file owns.
//
// The complaint it answers is about rhythm ("บางทีโมเดลสตรีมแบบติดๆขัดๆ"), and
// rhythm is the one thing a unit test cannot look at. So the tests below pin
// the four decisions that produce it instead: a burst is held rather than
// painted, a frame paints once however many chunks landed, a block seen for the
// first time is never typed out, and an authoritative replace is not paced at
// all. Get those wrong and the rhythm is wrong; get them right and what is left
// is a number to taste, which is DRAIN_MS.
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { pacedStream, pacedText } from '../lib/streamPace'

// A hand-driven rAF. Nothing here waits on real frames: jsdom's own clock would
// make every assertion a race, and the point of the pacer is exactly WHICH
// frame a letter lands on.
let clock = 0
let pending = new Map<number, FrameRequestCallback>()
let nextId = 1
const realRaf = globalThis.requestAnimationFrame
const realCancel = globalThis.cancelAnimationFrame
const realNow = performance.now.bind(performance)

beforeEach(() => {
  clock = 0
  pending = new Map()
  nextId = 1
  globalThis.requestAnimationFrame = ((cb: FrameRequestCallback) => {
    const id = nextId++
    pending.set(id, cb)
    return id
  }) as typeof requestAnimationFrame
  globalThis.cancelAnimationFrame = ((id: number) => void pending.delete(id)) as typeof cancelAnimationFrame
  performance.now = () => clock
})

afterEach(() => {
  globalThis.requestAnimationFrame = realRaf
  globalThis.cancelAnimationFrame = realCancel
  performance.now = realNow
})

/** Frames for a stretch of wall clock. Once the pacer has caught up it stops
 *  asking for any, so this becomes a plain wait — which is what the gap
 *  between two bursts actually is. */
function drain(ms: number): void {
  const until = clock + ms
  while (clock < until) frame()
}

/** One animation frame, 16ms later — the callbacks queued before it, and only
 *  those, so a loop that re-arms itself does not run away inside one call. */
function frame(ms = 16): void {
  clock += ms
  const due = [...pending.values()]
  pending.clear()
  for (const cb of due) cb(clock)
}

const LINE = 'ผมไล่ดูโค้ดของสตรีมให้แล้วครับ พบว่ามันวาดทุกก้อนที่มาถึง'
const LONG = LINE.repeat(4)

function mount(text: string, onPaint?: () => void) {
  const host = document.createElement('div')
  const action = pacedStream(host, { text, onPaint })
  // Trimmed: marked ends a paragraph with a newline, which is markup and not
  // something a reader sees.
  return { host, action, said: () => (host.textContent ?? '').trim() }
}

describe('the stream pacer', () => {
  it('shows a block whole the first time it sees it', () => {
    const { said } = mount(LONG)
    // A session parked mid-reply and switched back to arrives with everything
    // it had. Typing that out would be a story about the network, told late.
    expect(said()).toBe(LONG)
    expect(pending.size).toBe(0)
  })

  it('holds a burst back instead of painting it where it landed', () => {
    const { action, said } = mount('ก')
    action.update({ text: 'ก' + LONG })
    // The lump has arrived. Nothing has been drawn: the wire does not get to
    // decide when the page moves.
    expect(said()).toBe('ก')

    frame()
    const first = said().length
    expect(first).toBeGreaterThan(1)
    expect(first).toBeLessThan(LONG.length)

    frame()
    expect(said().length).toBeGreaterThan(first)
  })

  it('has the whole burst on screen by its deadline', () => {
    const { action, said } = mount('ก')
    action.update({ text: 'ก' + LONG })
    // MIN_WINDOW_MS is 120; ten frames is 160ms, so nothing may still be held back.
    for (let i = 0; i < 10; i++) frame()
    expect(said()).toBe('ก' + LONG)
  })

  it('paints once per frame however many chunks landed in it', () => {
    const onPaint = vi.fn()
    const { action } = mount('เริ่ม', onPaint)
    onPaint.mockClear()

    let text = 'เริ่ม'
    for (let i = 0; i < 20; i++) {
      text += LINE
      action.update({ text, onPaint })
    }
    // Twenty chunks, no frame between them: the markdown pass ran zero times.
    // This is the whole efficiency claim — the old code ran it twenty times to
    // show nineteen states nobody could see.
    expect(onPaint).not.toHaveBeenCalled()

    frame()
    expect(onPaint).toHaveBeenCalledTimes(1)
  })

  it('takes an authoritative replace whole, and stops pacing when it does', () => {
    const { action, said } = mount('ครึ่ง')
    action.update({ text: 'ครึ่งประโยคที่กำลังจะถูกทิ้ง' })
    // The engine's delivery is not a continuation of what streamed — pacing can
    // only add letters, so a text that takes one back is taken whole.
    action.update({ text: 'คำตอบจริงที่เอนจินส่งมาแทน' })
    expect(said()).toBe('คำตอบจริงที่เอนจินส่งมาแทน')
    expect(pending.size).toBe(0)
  })

  it('stops asking for frames once it has caught up', () => {
    const { action } = mount('ก')
    action.update({ text: 'ก' + LONG })
    for (let i = 0; i < 12; i++) frame()
    // A reply spends most of its life waiting on a tool. A loop that idles
    // through that keeps the compositor awake for nothing.
    expect(pending.size).toBe(0)
  })

  it('lets go of the frame loop when the element does', () => {
    const { action, said } = mount('ก')
    action.update({ text: 'ก' + LONG })
    action.destroy()
    const held = said()
    frame()
    expect(said()).toBe(held)
    expect(pending.size).toBe(0)
  })

  it('spreads a burst over the pause that came before it', () => {
    // The first version drained every lump in a fixed 140ms, which is shorter
    // than the pause between lumps — so the text sprinted, stopped, sprinted.
    // Smoother inside each lump and stop-start all the same, which is what the
    // owner saw: "ตอนตอบเปลี่ยนนิดนึง". The window has to learn the cadence.
    const { action, said } = mount('ก')
    let text = 'ก'
    for (let i = 0; i < 5; i++) {
      text += LINE
      action.update({ text })
      drain(350)
    }

    text += LINE
    action.update({ text })
    // 144ms in, a burst that used to be finished is still arriving, because
    // the gaps said it has about 350ms of room and not 140.
    drain(130)
    expect(said().length).toBeLessThan(text.length)
    // And it is still finished well before the next burst would land.
    drain(300)
    expect(said()).toBe(text)
  })

  it('never trails further than the ceiling, whatever the model pauses for', () => {
    const { action, said } = mount('ก')
    let text = 'ก'
    // Five seconds of silence mid-thought must not buy a five-second tail.
    for (let i = 0; i < 6; i++) {
      text += LINE
      action.update({ text })
      drain(5000)
    }
    text += LINE
    action.update({ text })
    drain(420) // MAX_WINDOW_MS is 400
    expect(said()).toBe(text)
  })

  it('paces reasoning the same way, drawn as text', () => {
    // The panel the reader watches for most of a thinking turn. It costs
    // nothing to draw, which is exactly why it was left unpaced at first — and
    // it stuttered anyway, because the jitter was never in the drawing.
    const host = document.createElement('div')
    const action = pacedText(host, { text: 'เริ่มคิด' })
    expect(host.textContent).toBe('เริ่มคิด')

    action.update({ text: 'เริ่มคิด' + LONG })
    expect(host.textContent).toBe('เริ่มคิด')

    frame()
    expect((host.textContent ?? '').length).toBeGreaterThan('เริ่มคิด'.length)
    drain(200)
    expect(host.textContent).toBe('เริ่มคิด' + LONG)
  })
})
