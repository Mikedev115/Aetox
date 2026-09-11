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
  it('asks the OS to flash only when the window is not focused', () => {
    setFocusProbe(() => true)
    attend('ask', true)
    expect(RequestAttention).not.toHaveBeenCalled()

    setFocusProbe(() => false)
    attend('ask', true)
    expect(RequestAttention).toHaveBeenCalledTimes(1)
  })

  it('respects the flash switch', () => {
    attention.layers = [{ id: 'flash', label: '', note: '', on: false }, { id: 'chime', label: '', note: '', on: true }]
    attention.loaded = true
    setFocusProbe(() => false)
    attend('done', false)
    expect(RequestAttention).not.toHaveBeenCalled()
  })

  it('treats an unread switch as on — a question before the settings load still counts', () => {
    expect(attentionOn('flash')).toBe(true)
    expect(attentionOn('chime')).toBe(true)
  })

  it('a question arriving reaches out; a finished turn on screen does not', () => {
    setFocusProbe(() => false)
    cockpit.openSession = 's1'
    cockpit.awaitingReply = true
    cockpit.turnSession = 's1'
    applyAskUser({ sessionId: 's1', data: q })
    expect(RequestAttention).toHaveBeenCalledTimes(1)

    vi.mocked(RequestAttention).mockClear()
    setFocusProbe(() => true)
    attend('done', true)
    expect(RequestAttention).not.toHaveBeenCalled()
  })
})
