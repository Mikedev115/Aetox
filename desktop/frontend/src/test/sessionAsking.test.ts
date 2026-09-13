// A chat stopped on a question, said everywhere the user might be instead of
// only in the transcript that holds the card (owner, 12 ก.ย. 2026: *"เวลา
// เอเจนถาม มันไม่มีอะไรแจ้งเตือนเลย เซสชั่นนั้นจะนิ่งและเงียบไป"*).
//
// Three claims: the row knows (sessionAsking), the topbar knows which chat
// (askingElsewhere), and a question for a chat this window holds nothing for
// is parked rather than dropped — because the engine is blocked on it with no
// deadline, and a dropped question is a turn that sits until Stop.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  cockpit, sessionAsking, sessionWorking, askingElsewhere, applyAskUser, applyAskDone,
} from '../lib/stores/cockpit.svelte'
import { attend, attention, attentionOn, setFocusProbe } from '../lib/stores/attention.svelte'
import { RequestAttention } from './mocks/wailsApp'
import { emptyTurnSpend } from '../lib/types'
import type { Session, ParkedTurn } from '../lib/types'

const row = (over: Partial<Session> = {}): Session =>
  ({ id: 's1', title: 'งานยาว', ago: '', ...over })

const parked = (over: Partial<ParkedTurn> = {}): ParkedTurn => ({
  chat: [], awaitingReply: true, agentStatus: '', toolSteps: [],
  turnFiles: [], turnProposals: [], streamingText: '', reasoningText: '',
  modelLoading: null, ask: null, driving: null, todos: [], turnSpend: emptyTurnSpend(), queued: [], ...over,
})

const q = { question: 'ไปต่อไหม?', options: ['ไปต่อ', 'หยุด'] }

beforeEach(() => {
  cockpit.awaitingReply = false
  cockpit.turnSession = ''
  cockpit.openSession = ''
  cockpit.activeView = 'chat'
  cockpit.parked = {}
  cockpit.ask = null
  cockpit.sessions = []
  cockpit.history = []
  cockpit.spaceHistory = []
  attention.layers = []
  attention.loaded = false
  setFocusProbe(null)
  vi.mocked(RequestAttention).mockClear()
})

describe('the asking dot on a chat row', () => {
  it('marks a parked chat stopped on a question, and outranks working', () => {
    cockpit.openSession = 'other'
    cockpit.parked['s1'] = parked({ ask: q })
    expect(sessionAsking(row())).toBe(true)
    // Still working by every flag — the row must choose, and it chooses the
    // state the user has to act on.
    expect(sessionWorking(row())).toBe(true)
  })

  it('marks the chat on screen while its card is up', () => {
    cockpit.openSession = 's1'
    cockpit.ask = q
    expect(sessionAsking(row())).toBe(true)
    expect(sessionAsking(row({ id: 'other' }))).toBe(false)
  })

  it('says nothing for a working chat that is not asking', () => {
    cockpit.openSession = 'other'
    cockpit.parked['s1'] = parked()
    expect(sessionAsking(row())).toBe(false)
  })
})

describe('the topbar strip', () => {
  it('names the parked chat that is asking, from whichever list has its title', () => {
    cockpit.openSession = 'other'
    cockpit.history = [row({ id: 's1', title: 'สรุปสัญญา' })]
    cockpit.parked['s1'] = parked({ ask: q })
    expect(askingElsewhere()).toEqual([{ id: 's1', title: 'สรุปสัญญา', question: 'ไปต่อไหม?' }])
  })

  it('is empty while the asking card is the one on screen', () => {
    cockpit.openSession = 's1'
    cockpit.ask = q
    expect(askingElsewhere()).toEqual([])
  })

  it('counts the open chat once another page is over its card', () => {
    cockpit.openSession = 's1'
    cockpit.ask = q
    cockpit.activeView = 'settings'
    expect(askingElsewhere().map((a) => a.id)).toEqual(['s1'])
  })
})

describe('a question for a chat this window holds nothing for', () => {
  it('is parked as asking rather than dropped', () => {
    cockpit.openSession = 'other'
    // Not awaiting, not parked, not the turn cursor: forLiveTurn's drop case.
    applyAskUser({ sessionId: 'ghost', data: q })
    expect(cockpit.parked['ghost']?.ask).toEqual(q)
    expect(sessionAsking(row({ id: 'ghost' }))).toBe(true)
    // Not drawn into the chat on screen — that is the failure the drop exists
    // to prevent, and it still must not happen.
    expect(cockpit.ask).toBeNull()
  })

  it('clears the way any other question does', () => {
    cockpit.openSession = 'other'
    applyAskUser({ sessionId: 'ghost', data: q })
    applyAskDone({ sessionId: 'ghost', data: null })
    expect(cockpit.parked['ghost']?.ask).toBeNull()
  })

  it('still drops an unstamped or empty question — there is nothing to park', () => {
    cockpit.openSession = 'other'
    applyAskUser({ sessionId: 'ghost', data: { question: '', options: [] } })
    expect(cockpit.parked['ghost']).toBeUndefined()
  })
})

describe('reaching outside the window', () => {
  // The chime is WebAudio; jsdom has none. A stand-in AudioContext counts the
  // notes so the test can hear whether it was asked to play.
  let played = 0
  const fakeAudio = () => {
    played = 0
    const node = { connect: () => node, start: () => {}, stop: () => {}, gain: { setValueAtTime() {}, linearRampToValueAtTime() {}, exponentialRampToValueAtTime() {} }, type: '', frequency: { value: 0 } }
    ;(globalThis as any).AudioContext = class {
      state = 'running'
      currentTime = 0
      destination = node
      resume() {}
      createOscillator() { played++; return node }
      createGain() { return node }
    }
  }
  const flush = () => new Promise((r) => setTimeout(r, 0))

  it('asks Go every time, with the chat’s name and what it wants', async () => {
    setFocusProbe(() => true)
    attend('ask', true, 'งานยาว')
    expect(RequestAttention).toHaveBeenCalledTimes(1)
    expect(vi.mocked(RequestAttention).mock.calls[0][0]).toBe('ask')
    expect(vi.mocked(RequestAttention).mock.calls[0][1]).toBe('งานยาว')
    expect(vi.mocked(RequestAttention).mock.calls[0][2]).not.toBe('')

    attend('done', false)
    expect(RequestAttention).toHaveBeenCalledTimes(2)
    // A chat with no name yet still gets one on the notification.
    expect(vi.mocked(RequestAttention).mock.calls[1][1]).not.toBe('')
  })

  // The webview's idea of focus used to gate everything; a webview wrong
  // about its own focus kept every signal in. Go's answer is read too, and
  // either saying "away" is enough.
  it('chimes for a finished turn on screen when the desktop says the user is away', async () => {
    fakeAudio()
    setFocusProbe(() => true)
    vi.mocked(RequestAttention).mockResolvedValueOnce(true)
    attend('done', true)
    await flush()
    expect(played).toBe(2)
  })

  it('a finished turn on screen, window in front by both accounts, is silent', async () => {
    fakeAudio()
    setFocusProbe(() => true)
    attend('done', true)
    await flush()
    expect(played).toBe(0)
  })

  it('the document’s own guess still counts when Go says nothing', async () => {
    fakeAudio()
    setFocusProbe(() => false)
    vi.mocked(RequestAttention).mockRejectedValueOnce(new Error('engine gone'))
    attend('done', true)
    await flush()
    expect(played).toBe(2)
  })

  it('respects the chime switch', async () => {
    fakeAudio()
    attention.layers = [{ id: 'flash', label: '', note: '', on: true }, { id: 'chime', label: '', note: '', on: false }]
    attention.loaded = true
    setFocusProbe(() => false)
    attend('ask', true)
    await flush()
    expect(played).toBe(0)
  })

  it('treats an unread switch as on — a question before the settings load still counts', () => {
    expect(attentionOn('flash')).toBe(true)
    expect(attentionOn('chime')).toBe(true)
    expect(attentionOn('toast')).toBe(true)
  })

  it('a question arriving reaches out, named after its chat', () => {
    cockpit.sessions = [row()]
    cockpit.openSession = 's1'
    cockpit.awaitingReply = true
    cockpit.turnSession = 's1'
    applyAskUser({ sessionId: 's1', data: q })
    expect(RequestAttention).toHaveBeenCalledTimes(1)
    expect(vi.mocked(RequestAttention).mock.calls[0][1]).toBe('งานยาว')
  })
})
