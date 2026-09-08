// The wording Tab types belongs to one conversation (2026-09-08).
//
// It rode across every switch until this was written: a chat that had just been
// offered wording kept it on `cockpit`, the user clicked another chat, and Tab
// in that composer typed an answer to a question asked somewhere they were no
// longer looking. The third time this shape of bug has been found — the desk
// tab (§187) and the undo chip were the first two — and all three are the same
// omission: state that Go always knew the owner of, held on a store the window
// never re-keyed on the way across.
//
// The arrival half was already right (applyPreparedReplies drops a wording
// stamped with another session), which is exactly why it went unnoticed: the
// guarded door was the one everybody looked at.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  cockpit, selectSession, applyPreparedReplies, preparedText, queuedMessages,
} from '../lib/stores/cockpit.svelte'
import { PendingUndo } from './mocks/wailsApp'
import type { Session } from '../lib/types'

const A = '20260908-120000.001'
const B = '20260908-120000.002'

const two = [{ text: 'เอาทางแรก' }, { text: 'เอาทางสอง' }]

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.chat = []
  cockpit.openSession = ''
  cockpit.turnSession = ''
  cockpit.awaitingReply = false
  cockpit.parked = {}
  cockpit.sessions = []
  cockpit.prepared = []
  cockpit.preparedAt = 0
  queuedMessages.length = 0
  vi.mocked(PendingUndo).mockResolvedValue([] as any)
})

describe('prepared wording belongs to its own chat', () => {
  it('does not follow the user into another conversation', async () => {
    cockpit.openSession = A
    applyPreparedReplies({ sessionId: A, data: two } as any)
    expect(preparedText()).toBe('เอาทางแรก')

    await selectSession({ id: B } as Session)

    // The bug: Tab here used to type A's answer. Nothing is on offer in B,
    // which is the correct state for a chat that was never asked anything.
    expect(cockpit.prepared).toHaveLength(0)
    expect(preparedText()).toBe('')
  })

  it('is still there when the user comes back', async () => {
    cockpit.openSession = A
    applyPreparedReplies({ sessionId: A, data: two } as any)

    await selectSession({ id: B } as Session)
    await selectSession({ id: A } as Session)

    // Parked rather than dropped, unlike the undo chip beside it: the chip can
    // be asked for again, and a wording is pushed once and stored nowhere, so
    // dropping it would let a glance at another chat destroy an offer that cost
    // a model call to write.
    expect(cockpit.prepared).toHaveLength(2)
    expect(preparedText()).toBe('เอาทางแรก')
  })

  it('comes back on the option the user had reached, not the first one', async () => {
    cockpit.openSession = A
    applyPreparedReplies({ sessionId: A, data: two } as any)
    cockpit.preparedAt = 1

    await selectSession({ id: B } as Session)
    await selectSession({ id: A } as Session)

    // Tab's second press moved to the other wording. Coming back to the first
    // one would be the chat quietly undoing a keypress the user made.
    expect(preparedText()).toBe('เอาทางสอง')
  })

  it('does not resurrect a wording in a chat that has none of its own', async () => {
    cockpit.openSession = A
    applyPreparedReplies({ sessionId: A, data: two } as any)

    await selectSession({ id: B } as Session)
    // B is asked nothing and offered nothing, twice over.
    await selectSession({ id: A } as Session)
    await selectSession({ id: B } as Session)

    expect(cockpit.prepared).toHaveLength(0)
  })

  it('lets a chat that took its wording come back with nothing on offer', async () => {
    cockpit.openSession = A
    applyPreparedReplies({ sessionId: A, data: two } as any)
    // The user typed their own answer instead, so the offer was put down.
    cockpit.prepared = []
    cockpit.preparedAt = 0

    await selectSession({ id: B } as Session)
    await selectSession({ id: A } as Session)

    expect(cockpit.prepared).toHaveLength(0)
  })
})
