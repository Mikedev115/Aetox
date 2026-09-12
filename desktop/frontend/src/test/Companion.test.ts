import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import Companion from '../lib/mascot/Companion.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { reportOf, REPORT_MAX } from '../lib/mascot/presence'
import { companion, setCompanionOn } from '../lib/mascot/companionSetting.svelte'
import { voice } from '../lib/mascot/voice.svelte'

// The assistant sitting on the screen: what it does is read off the cockpit's
// live turn, what it says is only what the model said. The drawing is
// mascot.test.ts's business; this guards the seam and the two rules the owner
// gave the companion — it never shows a command, and it never turns.

beforeEach(() => {
  localStorage.clear()
  cockpit.awaitingReply = false
  cockpit.agentStatus = ''
  cockpit.toolSteps = []
  cockpit.streamingText = ''
  cockpit.reasoningText = ''
  cockpit.ask = null
  cockpit.chat = []
  voice.mic = false
  voice.speaking = false
})

const mascot = (c: HTMLElement) => c.querySelector('.companion .mascot')!
/** Mount, and let the wave of arrival pass (HELLO_MS) under fake timers. */
async function arrived() {
  vi.useFakeTimers()
  const r = render(Companion)
  await vi.advanceTimersByTimeAsync(1900)
  return r
}

describe('the companion', () => {
  // It arrives with a wave, then rests: sways, never turns to the pointer.
  it('waves on arrival, then rests, sways and does not follow the pointer', async () => {
    vi.useFakeTimers()
    const { container } = render(Companion)
    await vi.advanceTimersByTimeAsync(50)
    expect(mascot(container).classList.contains('pose-greeting')).toBe(true)
    await vi.advanceTimersByTimeAsync(1900)
    expect(mascot(container).classList.contains('pose-idle')).toBe(true)
    expect(mascot(container).classList.contains('sway')).toBe(true)
    expect(mascot(container).classList.contains('settle')).toBe(true)
    expect(container.querySelector('.say')).toBeNull()
    vi.useRealTimers()
  })

  // A running tool is a card beside the head, and no words: the bubble stays
  // shut until the model says something.
  it('shows the running tool as a pose, never as text', async () => {
    cockpit.awaitingReply = true
    cockpit.toolSteps = [{ name: 'search', label: 'grep TODO', state: 'run', startedAt: 0 }] as any
    const { container } = await arrived()
    expect(mascot(container).classList.contains('pose-searchFiles')).toBe(true)
    expect(container.querySelector('.say')).toBeNull()
    vi.useRealTimers()
  })

  // The companion's own moments, none of them the chat's: being dragged is
  // walking; a turn that failed shows the alert where success shows the
  // check; the mic and the reader are listening and answering; left alone
  // long enough it goes to its charger, and a turn wakes it.
  it('walks when dragged, minds a failed turn, hears the mic, dozes off', async () => {
    const { container } = await arrived()
    const grab = container.querySelector('.grab')!
    ;(grab as any).setPointerCapture = () => {}
    await fireEvent.pointerDown(grab, { clientX: 500, clientY: 400, pointerId: 1, button: 0 })
    await fireEvent.pointerMove(grab, { clientX: 300, clientY: 200, pointerId: 1 })
    await vi.advanceTimersByTimeAsync(20)
    expect(mascot(container).classList.contains('pose-walk')).toBe(true)
    // held still: it stands and floats, facing the way it was going
    await vi.advanceTimersByTimeAsync(200)
    expect(mascot(container).classList.contains('pose-idle')).toBe(true)
    expect(mascot(container).classList.contains('settle')).toBe(false)
    await fireEvent.pointerMove(grab, { clientX: 280, clientY: 200, pointerId: 1 })
    await vi.advanceTimersByTimeAsync(20)
    expect(mascot(container).classList.contains('pose-walk')).toBe(true)
    await fireEvent.pointerUp(grab, { pointerId: 1 })
    await vi.advanceTimersByTimeAsync(40)
    expect(mascot(container).classList.contains('pose-idle')).toBe(true)
    expect(mascot(container).classList.contains('settle')).toBe(true)

    voice.mic = true
    await vi.advanceTimersByTimeAsync(20)
    expect(mascot(container).classList.contains('pose-listening')).toBe(true)
    voice.mic = false
    voice.speaking = true
    await vi.advanceTimersByTimeAsync(20)
    expect(mascot(container).classList.contains('pose-answering')).toBe(true)
    voice.speaking = false

    // a tool failed mid-turn and nothing runs yet: debugging, not thinking
    cockpit.awaitingReply = true
    cockpit.toolSteps = [{ name: 'shell', label: 'shell npm test', state: 'err', startedAt: 0 }] as any
    await vi.advanceTimersByTimeAsync(20)
    expect(mascot(container).classList.contains('pose-debugging')).toBe(true)
    // the turn ends badly: the alert, then rest
    cockpit.chat = [{ role: 'agent', text: 'x', failed: true }] as any
    cockpit.awaitingReply = false
    await vi.advanceTimersByTimeAsync(20)
    expect(mascot(container).classList.contains('pose-error')).toBe(true)
    await vi.advanceTimersByTimeAsync(2300)
    expect(mascot(container).classList.contains('pose-idle')).toBe(true)

    // five quiet minutes: the charger; a turn wakes it
    await vi.advanceTimersByTimeAsync(5 * 60_000 + 100)
    expect(mascot(container).classList.contains('pose-recharge')).toBe(true)
    cockpit.awaitingReply = true
    cockpit.toolSteps = []
    cockpit.reasoningText = 'hmm'
    await vi.advanceTimersByTimeAsync(20)
    expect(mascot(container).classList.contains('pose-thinking')).toBe(true)
    vi.useRealTimers()
  })

  it('types out the headline of what the model reported, and shuts while a tool runs', async () => {
    cockpit.awaitingReply = true
    cockpit.toolSteps = [
      { kind: 'note', label: 'อ่าน config แล้ว มี 3 ค่าที่ยังไม่ตั้ง\nรายละเอียดยาว ๆ ที่ไม่ต้องขึ้น', state: 'done', startedAt: 0 },
    ] as any
    const { container } = await arrived()
    await vi.advanceTimersByTimeAsync(2000)
    expect(container.querySelector('.say')?.textContent).toContain('อ่าน config แล้ว มี 3 ค่าที่ยังไม่ตั้ง')
    expect(container.querySelector('.say')?.textContent).not.toContain('รายละเอียด')
    // a tool starts: icon only
    cockpit.toolSteps = [...cockpit.toolSteps, { name: 'read', label: 'read config.yaml', state: 'run', startedAt: 0 }] as any
    await vi.advanceTimersByTimeAsync(100)
    expect(container.querySelector('.say')).toBeNull()
    expect(mascot(container).classList.contains('pose-reading')).toBe(true)
    vi.useRealTimers()
  })

  // A delegate's narration is the delegate's; the assistant's bubble does not
  // relay it.
  it('ignores a sub-agent\'s narration', async () => {
    cockpit.awaitingReply = true
    cockpit.toolSteps = [{ kind: 'note', label: 'delegate says hi', state: 'done', parent: 'task-1', startedAt: 0 }] as any
    const { container } = render(Companion)
    await waitFor(() => expect(mascot(container)).toBeTruthy())
    expect(container.querySelector('.say')).toBeNull()
  })

  it('shows the headline of the answer as it streams, not its tail', async () => {
    cockpit.awaitingReply = true
    cockpit.streamingText = '## สตอรีบอร์ด TikTok แนวตั้ง\n' + 'x'.repeat(100) + 'ท้ายจริง'
    const { container } = await arrived()
    const said = container.querySelector('.say')!.textContent ?? ''
    expect(said).toContain('สตอรีบอร์ด TikTok แนวตั้ง')
    expect(said).not.toContain('ท้ายจริง')
    expect(said).not.toContain('#')
    expect(mascot(container).classList.contains('pose-answering')).toBe(true)
    vi.useRealTimers()
  })

  // A press that does not travel is a click, and a click is a reaction — a
  // moment of one of the reaction poses with a hop, then back to rest.
  it('reacts to a click with a pose, and to a drag with a move', async () => {
    const { container } = await arrived()
    const grab = container.querySelector('.grab')!
    ;(grab as any).setPointerCapture = () => {}
    await fireEvent.pointerDown(grab, { clientX: 500, clientY: 400, pointerId: 1, button: 0 })
    await fireEvent.pointerUp(grab, { pointerId: 1 })
    await vi.advanceTimersByTimeAsync(50)
    const cls = mascot(container).className
    expect(cls).toMatch(/pose-(greeting|cheer|helping|wink)/)
    expect(cls).toContain('hop')
    await vi.advanceTimersByTimeAsync(1700)
    expect(mascot(container).classList.contains('pose-idle')).toBe(true)
    expect(localStorage.getItem('companionPos')).toBeNull()
    vi.useRealTimers()
  })

  // The × on the frame puts it away; the account menu's switch is the way back.
  it('hides on its × and comes back through the setting', async () => {
    setCompanionOn(true)
    const { container } = render(Companion)
    await waitFor(() => expect(mascot(container)).toBeTruthy())
    await fireEvent.click(container.querySelector('.hide')!)
    expect(companion.on).toBe(false)
    expect(localStorage.getItem('companionOn')).toBe('off')
    setCompanionOn(true)
    expect(companion.on).toBe(true)
  })

  it('remembers where it was dragged', async () => {
    const { container } = render(Companion)
    await waitFor(() => expect(mascot(container)).toBeTruthy())
    const grab = container.querySelector('.grab')!
    ;(grab as any).setPointerCapture = () => {}
    await fireEvent.pointerDown(grab, { clientX: 500, clientY: 400, pointerId: 1, button: 0 })
    await fireEvent.pointerMove(grab, { clientX: 300, clientY: 200, pointerId: 1 })
    await fireEvent.pointerUp(grab, { pointerId: 1 })
    const saved = JSON.parse(localStorage.getItem('companionPos') ?? '{}')
    expect(saved.x).toBeGreaterThanOrEqual(8)
    expect(saved.y).toBeGreaterThanOrEqual(8)
  })
})

describe('what the bubble may say', () => {
  it('is empty while nothing has been said', () => {
    expect(reportOf({ awaiting: true })).toBe('')
    expect(reportOf({ awaiting: false, note: 'old news' })).toBe('')
  })
  it('prefers the question, then the answer, then the note', () => {
    expect(reportOf({ awaiting: true, note: 'n', streamingText: 's', question: 'q?' })).toBe('q?')
    expect(reportOf({ awaiting: true, note: 'n', streamingText: 's' })).toBe('s')
    expect(reportOf({ awaiting: true, note: 'n' })).toBe('n')
  })
  it('keeps only the first line, cut short, marks stripped, and nothing while busy', () => {
    const long = '**' + 'ก'.repeat(200) + '**\nบรรทัดสอง'
    const out = reportOf({ awaiting: true, streamingText: long })
    expect(out.length).toBe(REPORT_MAX)
    expect(out.endsWith('…')).toBe(true)
    expect(out).not.toContain('*')
    expect(out).not.toContain('บรรทัดสอง')
    expect(reportOf({ awaiting: true, busy: true, note: 'พูดไว้ก่อนหน้า' })).toBe('')
    expect(reportOf({ awaiting: true, busy: true, streamingText: 'กำลังตอบ' })).toBe('กำลังตอบ')
  })
})
