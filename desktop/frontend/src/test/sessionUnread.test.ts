// The dot that says a turn finished while you were somewhere else.
//
// The ring answered one question — which chat is still going — and went out
// the moment the turn ended, at which point the row looked exactly like the
// forty idle rows above it. So the one thing the app lets you do, walk away
// from a long turn, had no ending: you had to open rows and look (owner, 8
// ก.ย.: "บางทีงานที่ทำไว้เสร็จแล้วอ่ะ บางทีมันไม่มีอะไรแจ้งเตือนเลยว่าเสร็จแล้ว").
//
// Driven through the same public doors the two-chats test uses — send, walk
// away, finish — because that is the only sequence the mark is ever set by.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  cockpit, sendUserMessage, selectSession, sessionWorking, sessionUnread,
  applyAgentDone, queuedMessages,
} from '../lib/stores/cockpit.svelte'
import { SendMessage, PendingUndo } from './mocks/wailsApp'
import type { Session } from '../lib/types'

const A = '20260908-100000.001'
const B = '20260908-100000.002'

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.chat = []
  cockpit.awaitingReply = false
  cockpit.turnSession = ''
  cockpit.openSession = ''
  cockpit.parked = {}
  cockpit.unread = {}
  cockpit.sessions = []
  queuedMessages.length = 0
  localStorage.removeItem('chatsUnread')
  vi.mocked(PendingUndo).mockResolvedValue([] as any)
})

function heldReply(): { resolve: (text: string) => void; promise: Promise<any> } {
  let resolve!: (text: string) => void
  const promise = new Promise<any>((r) => {
    resolve = (text: string) => r({ text, parts: [] })
  })
  return { resolve, promise }
}

describe('the finished-and-unread dot', () => {
  it('goes on when a turn ends in a chat the user walked away from', async () => {
    cockpit.openSession = A
    const turnA = heldReply()
    vi.mocked(SendMessage).mockReturnValueOnce(turnA.promise)
    const sendA = sendUserMessage('งานยาว')
    await Promise.resolve()

    await selectSession({ id: B } as Session)
    // Still going: green, not amber. One dot, and working wins it.
    expect(sessionWorking({ id: A })).toBe(true)
    expect(sessionUnread({ id: A })).toBe(false)

    turnA.resolve('เสร็จแล้วครับ')
    await sendA

    expect(sessionWorking({ id: A })).toBe(false)
    expect(sessionUnread({ id: A })).toBe(true)
    // And only that chat. The one the user is looking at has nothing to say.
    expect(sessionUnread({ id: B })).toBe(false)
  })

  it('says nothing about a turn that finished in front of the user', async () => {
    cockpit.openSession = A
    const turn = heldReply()
    vi.mocked(SendMessage).mockReturnValueOnce(turn.promise)
    const send = sendUserMessage('ถามสั้นๆ')
    await Promise.resolve()
    turn.resolve('ตอบแล้ว')
    await send

    expect(sessionUnread({ id: A })).toBe(false)
  })

  it('goes out when the chat is opened, and only then', async () => {
    cockpit.openSession = A
    const turnA = heldReply()
    vi.mocked(SendMessage).mockReturnValueOnce(turnA.promise)
    const sendA = sendUserMessage('งานยาว')
    await Promise.resolve()
    await selectSession({ id: B } as Session)
    turnA.resolve('เสร็จแล้วครับ')
    await sendA
    expect(sessionUnread({ id: A })).toBe(true)

    // Walking somewhere else is not reading it.
    await selectSession({ id: '20260908-100000.003' } as Session)
    expect(sessionUnread({ id: A })).toBe(true)

    await selectSession({ id: A } as Session)
    expect(sessionUnread({ id: A })).toBe(false)
  })

  // The window that reattached after a reload ends its turns here instead —
  // its live detail died with the previous webview, so the dot is the only
  // thing left saying the work happened at all.
  it('marks a reattached turn that ends off screen', async () => {
    cockpit.openSession = B
    cockpit.parked[A] = { awaitingReply: true } as any

    await applyAgentDone({ sessionId: A })

    expect(sessionUnread({ id: A })).toBe(true)
  })

  // Closing the window is not reading the answer, so the mark has to survive
  // the reload — it is written where a fresh window will find it.
  it('keeps the mark across a reload', async () => {
    cockpit.openSession = B
    cockpit.parked[A] = { awaitingReply: true } as any
    await applyAgentDone({ sessionId: A })

    const held = JSON.parse(localStorage.getItem('chatsUnread') ?? '{}')
    expect(held[A]).toBe(true)

    // ...and a chat that has been read is gone from it, not stored as `false`.
    await selectSession({ id: A } as Session)
    expect(JSON.parse(localStorage.getItem('chatsUnread') ?? '{}')[A]).toBeUndefined()
  })
})
