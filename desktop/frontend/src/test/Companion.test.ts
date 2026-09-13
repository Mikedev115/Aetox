import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Companion from '../lib/mascot/Companion.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { reportOf, spokenOf, REPORT_MAX } from '../lib/mascot/presence'
import { companion, setCompanionOn, setCompanionVoice, setCompanionGreet, setCompanionSize, bubbleMetrics, SIZE_DEFAULT, SIZE_MIN, SIZE_MAX } from '../lib/mascot/companionSetting.svelte'
import { voice } from '../lib/mascot/voice.svelte'
import { speech, stopSpeech } from '../lib/speech.svelte'
import { StartSpeech } from './mocks/wailsApp'
import { profile } from '../lib/stores/profile.svelte'
import { t } from '../lib/i18n.svelte'

// The assistant sitting on the screen: what it does is read off the cockpit's
// live turn, what it says is what the model said — plus the one line of ours,
// the room's greeting as an empty chat arrives. The drawing is
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
  cockpit.activeView = 'chat'
  cockpit.desk = ''
  cockpit.chair = ''
  cockpit.space = ''
  cockpit.openSession = 's1'
  profile.name = ''
  profile.loaded = true
  voice.mic = false
  voice.speaking = false
  stopSpeech()
  setCompanionVoice(true)
  // Off in every test but its own: the mock read never ends, and a greeting
  // spoken at every mount would hold the bubble and the answering pose.
  setCompanionGreet(false)
  vi.mocked(StartSpeech).mockClear()
})

const mascot = (c: HTMLElement) => c.querySelector('.companion .mascot')!
/** Mount, and let the wave of arrival (HELLO_MS) and the greeting it says on
 *  an empty chat (GREET_MS) both pass under fake timers. */
async function arrived() {
  vi.useFakeTimers()
  const r = render(Companion)
  await vi.advanceTimersByTimeAsync(4100)
  return r
}

describe('the companion', () => {
  // It arrives with a wave and, on an empty chat, the room's greeting; then it
  // rests: sways, never turns to the pointer.
  it('waves on arrival, then rests, sways and does not follow the pointer', async () => {
    vi.useFakeTimers()
    const { container } = render(Companion)
    await vi.advanceTimersByTimeAsync(50)
    expect(mascot(container).classList.contains('pose-greeting')).toBe(true)
    await vi.advanceTimersByTimeAsync(4100)
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

    // five quiet minutes: asleep on the pillow; a message startles it awake
    // (a jolt, hands up) and only then does the work pose take over
    await vi.advanceTimersByTimeAsync(5 * 60_000 + 100)
    expect(mascot(container).classList.contains('pose-recharge')).toBe(true)
    cockpit.awaitingReply = true
    cockpit.toolSteps = []
    cockpit.reasoningText = 'hmm'
    await vi.advanceTimersByTimeAsync(40)
    expect(mascot(container).classList.contains('pose-startled')).toBe(true)
    expect(mascot(container).classList.contains('hop')).toBe(true)
    await vi.advanceTimersByTimeAsync(800)
    expect(mascot(container).classList.contains('pose-thinking')).toBe(true)
    // asleep again and poked: a slow stretch, then rest — no cheer
    cockpit.awaitingReply = false
    cockpit.reasoningText = ''
    await vi.advanceTimersByTimeAsync(2300 + 5 * 60_000 + 100)
    expect(mascot(container).classList.contains('pose-recharge')).toBe(true)
    await fireEvent.pointerDown(grab, { clientX: 500, clientY: 400, pointerId: 2, button: 0 })
    await fireEvent.pointerUp(grab, { pointerId: 2 })
    await vi.advanceTimersByTimeAsync(40)
    expect(mascot(container).classList.contains('pose-wake')).toBe(true)
    await vi.advanceTimersByTimeAsync(1400)
    expect(mascot(container).classList.contains('pose-idle')).toBe(true)
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

describe('the greeting', () => {
  // The same line the room prints above its cards, with the name, once, as
  // the empty chat arrives — and gone on its own.
  it("says the room's question with the user's name when an empty chat comes on screen, then falls silent", async () => {
    profile.name = 'mike'
    vi.useFakeTimers()
    const { container } = render(Companion)
    await vi.advanceTimersByTimeAsync(1500)
    expect(container.querySelector('.say')?.textContent).toBe(t('start.assistant.headlineNamed', { name: 'mike' }))
    expect(mascot(container).classList.contains('pose-greeting')).toBe(true)
    await vi.advanceTimersByTimeAsync(3000)
    expect(container.querySelector('.say')).toBeNull()
    expect(mascot(container).classList.contains('pose-idle')).toBe(true)
    vi.useRealTimers()
  })

  it("greets again on a switch to another empty room, in that room's words", async () => {
    profile.name = 'mike'
    const { container } = await arrived()
    expect(container.querySelector('.say')).toBeNull()
    cockpit.desk = 'coding'
    cockpit.openSession = 's2'
    await vi.advanceTimersByTimeAsync(1500)
    expect(container.querySelector('.say')?.textContent).toBe(t('start.coding.headlineNamed', { name: 'mike' }))
    vi.useRealTimers()
  })

  it("leaves a chair's opening to the room, and stops the moment a turn starts", async () => {
    cockpit.chair = 'doc'
    vi.useFakeTimers()
    const { container } = render(Companion)
    await vi.advanceTimersByTimeAsync(1500)
    expect(container.querySelector('.say')).toBeNull()
    cockpit.chair = ''
    cockpit.openSession = 's3'
    await vi.advanceTimersByTimeAsync(1500)
    expect(container.querySelector('.say')?.textContent).toBe(t('start.assistant.headline'))
    cockpit.awaitingReply = true
    await vi.advanceTimersByTimeAsync(50)
    expect(container.querySelector('.say')).toBeNull()
    vi.useRealTimers()
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
  it('is spoken as the same headline, cut at a word and without the ellipsis', () => {
    expect(spokenOf('# หัวข้อ' + String.fromCharCode(10) + 'ตัวเนื้อยาว ๆ')).toBe('หัวข้อ')
    const words = Array.from({ length: 40 }, (_, i) => 'คำ' + i).join(' ')
    const out = spokenOf(words)
    expect(out.length).toBeLessThanOrEqual(REPORT_MAX)
    expect(out.endsWith('…')).toBe(false)
    expect(words.startsWith(out + ' ')).toBe(true)
    // no space to cut at: the bubble's hard cut, still without the ellipsis
    expect(spokenOf('ก'.repeat(200))).toBe('ก'.repeat(REPORT_MAX))
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

// The voice (12 ก.ย. 2026): the on-screen chat's finished answer and a
// blocking question are read through the window's one player; nothing
// else is — a long run is quiet until it reports — and anything that means
// "the user is talking now" stops it. Failures are silent here.
describe('what it says out loud', () => {
  const spoken = () => vi.mocked(StartSpeech).mock.calls.map((c) => c[0])

  it('reads the finished answer, holds its headline up while reading, and not a failed or stopped turn', async () => {
    const { container } = await arrived()
    cockpit.awaitingReply = true
    // narration between tools is shown, never read
    cockpit.toolSteps = [{ kind: 'note', label: 'กำลังอ่านไฟล์', state: 'done', startedAt: 0 }] as any
    await vi.advanceTimersByTimeAsync(50)
    expect(spoken()).toEqual([])
    cockpit.chat = [{ role: 'user', text: 'q' }, { role: 'agent', text: '# สรุป' + String.fromCharCode(10) + 'เสร็จแล้วครับ' }] as any
    cockpit.toolSteps = []
    cockpit.awaitingReply = false
    await vi.advanceTimersByTimeAsync(50)
    // the headline only — the answer's body is the ฟัง button's read, not
    // the companion's (owner, 13 ก.ย.: "พูดแค่สรุปสิ่งที่ทำ")
    expect(spoken()).toEqual(['สรุป'])
    expect(speech.key).toBe('companion')
    await vi.advanceTimersByTimeAsync(500)
    expect(container.querySelector('.say')?.textContent).toBe('สรุป')
    stopSpeech()
    await vi.advanceTimersByTimeAsync(50)
    expect(container.querySelector('.say')).toBeNull()

    cockpit.awaitingReply = true
    await vi.advanceTimersByTimeAsync(50)
    cockpit.chat = [{ role: 'agent', text: 'x', failed: true }] as any
    cockpit.awaitingReply = false
    await vi.advanceTimersByTimeAsync(50)
    cockpit.awaitingReply = true
    await vi.advanceTimersByTimeAsync(50)
    cockpit.chat = [{ role: 'agent', text: 'x', failed: true, stopped: true }] as any
    cockpit.awaitingReply = false
    await vi.advanceTimersByTimeAsync(50)
    expect(spoken().length).toBe(1)
    vi.useRealTimers()
  })

  it('reads a question it is blocked on, once', async () => {
    await arrived()
    cockpit.awaitingReply = true
    cockpit.ask = { question: 'จะให้ลบไหม?', options: [] } as any
    await vi.advanceTimersByTimeAsync(50)
    cockpit.ask = { question: 'จะให้ลบไหม?', options: [] } as any
    await vi.advanceTimersByTimeAsync(50)
    expect(spoken()).toEqual(['จะให้ลบไหม?'])
    vi.useRealTimers()
  })

  it('stops when the user talks instead: a new message, the mic, another chat, the switch, a click', async () => {
    const { container } = await arrived()
    const speakNow = async () => {
      cockpit.awaitingReply = true
      await vi.advanceTimersByTimeAsync(50)
      cockpit.chat = [{ role: 'agent', text: 'คำตอบ' }] as any
      cockpit.awaitingReply = false
      await vi.advanceTimersByTimeAsync(50)
      expect(speech.key).toBe('companion')
    }
    await speakNow()
    cockpit.awaitingReply = true
    await vi.advanceTimersByTimeAsync(50)
    expect(speech.key).toBe('')
    cockpit.awaitingReply = false
    await vi.advanceTimersByTimeAsync(50)

    await speakNow()
    voice.mic = true
    await vi.advanceTimersByTimeAsync(50)
    expect(speech.key).toBe('')
    voice.mic = false

    await speakNow()
    cockpit.openSession = 's9'
    await vi.advanceTimersByTimeAsync(50)
    expect(speech.key).toBe('')

    await speakNow()
    const grab = container.querySelector('.grab')!
    ;(grab as any).setPointerCapture = () => {}
    await fireEvent.pointerDown(grab, { clientX: 500, clientY: 400, pointerId: 1, button: 0 })
    await fireEvent.pointerUp(grab, { pointerId: 1 })
    await vi.advanceTimersByTimeAsync(50)
    expect(speech.key).toBe('')
    expect(companion.voice).toBe(true)

    await speakNow()
    const n = spoken().length
    await fireEvent.click(container.querySelector('.mute')!)
    await vi.advanceTimersByTimeAsync(50)
    expect(companion.voice).toBe(false)
    expect(speech.key).toBe('')
    expect(localStorage.getItem('companionVoice')).toBe('off')
    // switched off, a finished answer is not read
    cockpit.awaitingReply = true
    await vi.advanceTimersByTimeAsync(50)
    cockpit.chat = [{ role: 'agent', text: 'เงียบ' }] as any
    cockpit.awaitingReply = false
    await vi.advanceTimersByTimeAsync(50)
    expect(spoken().length).toBe(n)
    vi.useRealTimers()
  })

  // The greeting is spoken too, after a beat — and once, with the name, when
  // the name lands just after the nameless greeting was shown.
  it('speaks the greeting once, with the name that arrives a beat later — behind its own switch', async () => {
    setCompanionGreet(true)
    vi.useFakeTimers()
    render(Companion)
    await vi.advanceTimersByTimeAsync(100)
    profile.name = 'mike'
    await vi.advanceTimersByTimeAsync(1000)
    expect(spoken()).toEqual([t('start.assistant.headlineNamed', { name: 'mike' })])
    vi.useRealTimers()
    expect(localStorage.getItem('companionGreet')).toBe('on')
  })

  it('is silent when the engine refuses', async () => {
    vi.mocked(StartSpeech).mockRejectedValueOnce(new Error('ไม่มีเสียง'))
    await arrived()
    cockpit.awaitingReply = true
    await vi.advanceTimersByTimeAsync(50)
    cockpit.chat = [{ role: 'agent', text: 'คำตอบ' }] as any
    cockpit.awaitingReply = false
    await vi.advanceTimersByTimeAsync(50)
    expect(speech.key).toBe('')
    expect(voice.speaking).toBe(false)
    vi.useRealTimers()
  })

  it('is on by default and remembered off', () => {
    expect(companion.voice).toBe(true)
    setCompanionVoice(false)
    expect(localStorage.getItem('companionVoice')).toBe('off')
  })

  // The corner of the hover frame resizes the figure, between the caps, and
  // the number is kept for both places (owner, 13 ก.ย. 2026: "ขยายใหญ่และ
  // เล็กลงได้ … เอาเพดานสูงสุดด้วย อย่าลืมเพดานเล็กสุด").
  it('resizes from the corner grip, within its caps, and remembers', async () => {
    setCompanionSize(SIZE_DEFAULT)
    const { container } = await arrived()
    const grip = container.querySelector('.grip')!
    ;(grip as any).setPointerCapture = () => {}
    const box = () => container.querySelector('.companion') as HTMLElement
    expect(box().style.width).toBe(`${SIZE_DEFAULT}px`)
    await fireEvent.pointerDown(grip, { clientX: 100, clientY: 100, pointerId: 2, button: 0 })
    await fireEvent.pointerMove(grip, { clientX: 140, clientY: 120, pointerId: 2 })
    expect(companion.size).toBe(SIZE_DEFAULT + 40)
    expect(box().style.width).toBe(`${SIZE_DEFAULT + 40}px`)
    // past the ceiling it stops; back past the floor it stops there too
    await fireEvent.pointerMove(grip, { clientX: 2000, clientY: 100, pointerId: 2 })
    expect(companion.size).toBe(SIZE_MAX)
    await fireEvent.pointerMove(grip, { clientX: -2000, clientY: -2000, pointerId: 2 })
    expect(companion.size).toBe(SIZE_MIN)
    await fireEvent.pointerUp(grip, { pointerId: 2 })
    expect(localStorage.getItem('companionSize')).toBe(String(SIZE_MIN))
    // a grip press is not a click on the figure: no reaction
    expect(mascot(container).classList.contains('pose-cheer')).toBe(false)
    setCompanionSize(SIZE_DEFAULT)
  })

  // The bubble follows the figure (owner, 13 ก.ย. 2026: "ให้มันขยายตาม แต่
  // จำกัดความยาวให้มันสูงขึ้นแทน"): at the default size it is what it was —
  // 12px on a card at most 300px wide; bigger, the type grows at three
  // quarters of the figure's rate while the width is held to a cap, so a
  // long answer folds into more lines. Smaller, the type shrinks a little
  // and the width stays. The same numbers as companion_draw.go bubbleMetrics.
  it("scales the bubble's type with the figure and holds its width", async () => {
    expect(bubbleMetrics(SIZE_DEFAULT)).toEqual({ font: 12, maxW: 300 })
    expect(bubbleMetrics(160)).toEqual({ font: 16.8, maxW: 334 })
    expect(bubbleMetrics(SIZE_MAX)).toEqual({ font: 23.8, maxW: 380 })
    expect(bubbleMetrics(SIZE_MIN)).toEqual({ font: 11, maxW: 300 })
    expect(bubbleMetrics(9999)).toEqual(bubbleMetrics(SIZE_MAX))

    setCompanionSize(200)
    const { container } = await arrived()
    const box = container.querySelector('.companion') as HTMLElement
    const m = bubbleMetrics(200)
    expect(box.style.getPropertyValue('--say-font')).toBe(`${m.font}px`)
    expect(box.style.getPropertyValue('--say-max')).toBe(`${m.maxW}px`)
    setCompanionSize(SIZE_DEFAULT)
    await tick()
    expect(box.style.getPropertyValue('--say-font')).toBe('12px')
    expect(box.style.getPropertyValue('--say-max')).toBe('300px')
  })
})
