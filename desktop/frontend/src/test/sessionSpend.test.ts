// The bill that survives a refresh.
//
// The composer had exactly one spend figure and it was per-turn: reset when the
// next turn starts, and gone entirely when the webview reloads, because it only
// ever lived in the window. So refreshing took the number to zero and read as
// the app throwing the bill away (owner, 7 ก.ย.: "มันชอบหายตอนรีเฟรชและชอบไป
// เริ่มใหม่ ทั้งที่ควรจะผูกกับเซสชั่น ... ค่าใช้จ่ายอะไรก็รีเฟรชหมด").
//
// Nothing was ever lost. Every round had been in token_usage the whole time,
// filed under the session id — the window simply never read it back. These
// tests hold that line: the chat's total is a READ of that table, so it cannot
// be reset by a reload, and it must never be added to locally, which is the
// shortcut that would put the same round in twice.
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit, applyUsageRound, refreshSessionSpend } from '../lib/stores/cockpit.svelte'
import { emptyTurnSpend, emptySessionSpend } from '../lib/types'
import { SessionSpend, GetContextBreakdown, EnabledProviders, ListModelsForProvider } from './mocks/wailsApp'

// The owner's real chat, 7 ก.ย.: twenty rounds, 46% of the input served from
// the provider's cache.
const total = (over: Partial<ReturnType<typeof emptySessionSpend>> = {}) => ({
  ...emptySessionSpend(),
  in: 444043, out: 6804, cached: 204416, cacheReported: true, rounds: 20,
  ...over,
})

const breakdown = {
  usedTokens: 36998, maxTokens: 1000000, measured: true, cachedTokens: 36864,
  slices: [
    { key: 'system', tokens: 4982 },
    { key: 'tools', tokens: 27910 },
    { key: 'messages', tokens: 4106 },
    { key: 'free', tokens: 963002 },
  ],
}

const props = () => ({
  messages: [],
  task: { elapsed: '', steps: [] },
  model: {
    provider: 'aetox', modelName: 'test', thinkLevel: '', contextUsed: 0,
    contextMax: 0, approval: 'ask' as const, wireFormat: '', warning: '', pending: null,
  },
  awaitingReply: false,
  agentStatus: '',
  toolSteps: [],
  streamingText: '',
  reasoningText: '',
  onSend: () => {},
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onCancelPendingModel: async () => {},
  onSubmitAPIKey: async () => {},
})

const openPanel = async () => {
  const button = await screen.findByRole('button', { name: /Context window|หน้าต่างคอนเท็กซ์/ })
  button.click()
}

beforeEach(() => {
  vi.clearAllMocks()
  EnabledProviders.mockResolvedValue(['aetox'])
  ListModelsForProvider.mockResolvedValue(['test'])
  GetContextBreakdown.mockResolvedValue(breakdown as any)
  SessionSpend.mockResolvedValue(emptySessionSpend() as any)
  cockpit.openSession = 'sess_1'
  cockpit.turnSpend = emptyTurnSpend()
  cockpit.sessionSpend = emptySessionSpend()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('the chat total in the context panel', () => {
  // The reload case, which is the whole reason this exists: the turn's tally is
  // legitimately empty — this window has watched no rounds — and the number the
  // user was looking at a second ago must still be on screen.
  it('stands on its own when the turn tally is empty, as it is after a reload', async () => {
    cockpit.sessionSpend = total()

    render(Chat, props())
    await openPanel()

    expect(await screen.findByText(/This chat, all of it|แชทนี้ทั้งหมด/)).toBeTruthy()
    // And the turn's block is absent rather than drawn as a row of zeros: a
    // zero reads as a measurement, and this window has not made one.
    expect(screen.queryByText(/Spent this turn|เทิร์นนี้ใช้ไปแล้ว/)).toBeNull()
    expect(screen.getByText('444.0k')).toBeTruthy()
    expect(screen.getByText('6.8k')).toBeTruthy()
  })

  // Rounds, not messages. Twenty rounds under a three-message chat is a tool
  // loop, and without the line it reads as the meter having lost its mind.
  it('says how many model rounds the total is made of', async () => {
    cockpit.sessionSpend = total()

    render(Chat, props())
    await openPanel()

    expect(await screen.findByText(/20 (model rounds|รอบโมเดล|次模型调用)/)).toBeTruthy()
  })

  it('draws both blocks mid-turn, labelled apart', async () => {
    cockpit.sessionSpend = total()
    cockpit.turnSpend = { ...emptyTurnSpend(), in: 269300, out: 4700, cached: 105000, cacheReported: true }

    render(Chat, props())
    await openPanel()

    expect(await screen.findByText(/Spent this turn|เทิร์นนี้ใช้ไปแล้ว/)).toBeTruthy()
    expect(screen.getByText(/This chat, all of it|แชทนี้ทั้งหมด/)).toBeTruthy()
    // The turn is a part of the chat, so both figures are on screen at once and
    // the reader has to be able to tell which is which.
    expect(screen.getByText('269.3k')).toBeTruthy()
    expect(screen.getByText('444.0k')).toBeTruthy()
  })

  // A chat that has genuinely spent nothing draws no card, same as before. The
  // absence is the honest rendering; a block of zeros is not.
  it('draws nothing at all for a chat that has spent nothing', async () => {
    render(Chat, props())
    await openPanel()

    await screen.findByText(/37.0k \/ 1000.0k/)
    expect(screen.queryByText(/This chat, all of it|แชทนี้ทั้งหมด/)).toBeNull()
    expect(screen.queryByText(/Spent this turn|เทิร์นนี้ใช้ไปแล้ว/)).toBeNull()
  })

  // Money only when every round could be priced — the same rule the turn block
  // has always followed, asked separately of each block so an unpriceable turn
  // cannot suppress a total that is priced.
  it('withholds money from the block that cannot account for all of it', async () => {
    cockpit.sessionSpend = total({ cost: 0.0154, unpriced: 3 })

    render(Chat, props())
    await openPanel()

    await screen.findByText(/This chat, all of it|แชทนี้ทั้งหมด/)
    expect(screen.queryByText('$0.0154')).toBeNull()
    expect(screen.getByText(/No published rate|ยังไม่มีราคาประกาศ|没有公布价格/)).toBeTruthy()
  })
})

describe('where the chat total comes from', () => {
  it('is read from the database, never added up in the window', async () => {
    SessionSpend.mockResolvedValue(total() as any)

    await refreshSessionSpend()

    expect(SessionSpend).toHaveBeenCalledWith('sess_1')
    expect(cockpit.sessionSpend.in).toBe(444043)
    expect(cockpit.sessionSpend.rounds).toBe(20)
  })

  // The round is already in the table — recordTokenUsage writes the row before
  // emitUsageRound announces it — so the answer is to ask again, not to add.
  // Adding would double the round the read is about to return.
  it('re-reads after a round instead of adding it', async () => {
    vi.useFakeTimers()
    SessionSpend.mockResolvedValue(total({ in: 30000, out: 500, rounds: 1 }) as any)

    applyUsageRound({ session: 'sess_1', in: 30000, out: 500 })
    await vi.advanceTimersByTimeAsync(1200)

    expect(SessionSpend).toHaveBeenCalledTimes(1)
    // Not 60000, which is what a window that both added and re-read would show.
    expect(cockpit.sessionSpend.in).toBe(30000)
    // The live turn tally is untouched by any of this; it is still its own count.
    expect(cockpit.turnSpend.in).toBe(30000)
  })

  // A turn with four delegates lands rounds faster than anyone can read them,
  // and each one would otherwise be a Go round-trip.
  it('coalesces a burst of rounds into one read', async () => {
    vi.useFakeTimers()
    SessionSpend.mockResolvedValue(total() as any)

    for (let i = 0; i < 8; i++) applyUsageRound({ session: 'sess_1', in: 1000, out: 10 })
    await vi.advanceTimersByTimeAsync(1200)

    expect(SessionSpend).toHaveBeenCalledTimes(1)
  })

  it('ignores a round another chat spent', async () => {
    vi.useFakeTimers()

    applyUsageRound({ session: 'sess_2', in: 90000, out: 5000 })
    await vi.advanceTimersByTimeAsync(1200)

    // A background conversation's total is re-read when the user arrives at it,
    // not written under the composer of the chat they are actually looking at.
    expect(SessionSpend).not.toHaveBeenCalled()
  })

  // The user can switch chats inside the await. A late answer landing afterwards
  // would put one conversation's bill under another's composer, which is the
  // exact bug clearLive's turnSpend reset exists to prevent.
  it('drops an answer that arrives after the user has walked away', async () => {
    let release: (v: any) => void = () => {}
    SessionSpend.mockReturnValue(new Promise((r) => { release = r }) as any)

    const pending = refreshSessionSpend()
    cockpit.openSession = 'sess_2'
    release(total())
    await pending

    expect(cockpit.sessionSpend.in).toBe(0)
  })

  // "The query did not come back" and "this chat has spent nothing" are
  // different sentences, and only one of them belongs under the composer.
  it('keeps the last good figure when the read fails', async () => {
    cockpit.sessionSpend = total()
    SessionSpend.mockRejectedValue(new Error('engine not ready'))

    await refreshSessionSpend()

    expect(cockpit.sessionSpend.in).toBe(444043)
  })
})
