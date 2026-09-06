// The answer the user has not typed yet, waiting in the composer for Tab
// (desktop/prepared_reply.go).
//
// What these pin down is the line the feature lives or dies on: it may write
// into the box, and it may never SEND. Everything else here is about the box
// staying the user's — a suggestion that survives being typed over, or that
// appears on top of a sentence somebody is writing, is worse than no
// suggestion at all.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit, applyPreparedReplies, clearPrepared, nextPrepared, preparedText } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics } from './mocks/wailsApp'

const onSend = vi.fn()

const baseProps = {
  messages: [{ role: 'user', text: 'เอาไงดี', time: '19:45' }] as any[],
  task: { title: '', steps: [] } as any,
  awaitingReply: false,
  agentStatus: '',
  toolSteps: [] as any[],
  streamingText: '',
  reasoningText: '',
  onSend,
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onCancelPendingModel: async () => {},
  onSubmitAPIKey: async () => {},
  model: { provider: 'deepseek', modelName: 'v4', thinkLevel: 'high', approval: 'ask', wireFormat: '' } as any,
}

const two = [{ text: 'เอาทางแรก ยิงรอบสองไปเลย' }, { text: 'เอาทางสอง เพิ่มเครื่องมือดีกว่า' }]

const box = (c: Element) => c.querySelector('.composer .input') as HTMLTextAreaElement
const ghost = (c: Element) => c.querySelector('.composer .ghost') as HTMLElement | null

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.toolSteps = []
  cockpit.ask = null
  cockpit.openSession = ''
  clearPrepared()
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
})

describe('a reply prepared for the user', () => {
  it('waits in the composer as dim text without touching the draft', async () => {
    applyPreparedReplies(two)
    const { container } = render(Chat, baseProps)
    await tick()

    expect(ghost(container)?.textContent).toBe(two[0].text)
    // The wording is drawn, not typed: nothing has been put in the box yet, so
    // pressing Enter now sends nothing the user did not write.
    expect(box(container).value).toBe('')
  })

  it('is taken by Tab, and is still only a draft — nothing is sent', async () => {
    applyPreparedReplies(two)
    const { container } = render(Chat, baseProps)
    await tick()

    await fireEvent.keyDown(box(container), { key: 'Tab' })
    await tick()

    expect(box(container).value).toBe(two[0].text)
    // The whole point of Tab rather than a button that answers for you.
    expect(onSend).not.toHaveBeenCalled()
    // Taken, so there is nothing left to draw underneath.
    expect(ghost(container)).toBeNull()
  })

  it('swaps to the next wording when Tab is pressed again', async () => {
    applyPreparedReplies(two)
    const { container } = render(Chat, baseProps)
    await tick()

    await fireEvent.keyDown(box(container), { key: 'Tab' })
    await tick()
    await fireEvent.keyDown(box(container), { key: 'Tab' })
    await tick()

    expect(box(container).value).toBe(two[1].text)
    expect(onSend).not.toHaveBeenCalled()
  })

  // Cycling would mean a third press puts back the first option with nothing
  // saying it has been round — on a control that edits a message about to be
  // sent.
  it('stops at the last wording rather than cycling back to the first', () => {
    applyPreparedReplies(two)
    expect(nextPrepared()).toBe(two[1].text)
    expect(nextPrepared()).toBe('')
    expect(preparedText()).toBe(two[1].text)
  })

  it('gets out of the way as soon as the user writes their own words', async () => {
    applyPreparedReplies(two)
    const { container } = render(Chat, baseProps)
    await tick()

    await fireEvent.input(box(container), { target: { value: 'ไม่เอาทั้งสองอัน' } })
    await tick()

    expect(ghost(container)).toBeNull()
    expect(cockpit.prepared).toHaveLength(0)
    // And having been declined, it does not come back when the box is emptied.
    await fireEvent.input(box(container), { target: { value: '' } })
    await tick()
    expect(ghost(container)).toBeNull()
  })

  it('leaves Tab alone when the box holds the user\'s own writing', async () => {
    applyPreparedReplies(two)
    const { container } = render(Chat, baseProps)
    await tick()

    await fireEvent.input(box(container), { target: { value: 'ผมพิมพ์เอง' } })
    await tick()
    await fireEvent.keyDown(box(container), { key: 'Tab' })
    await tick()

    expect(box(container).value).toBe('ผมพิมพ์เอง')
  })

  it('is put down by Escape', async () => {
    applyPreparedReplies(two)
    const { container } = render(Chat, baseProps)
    await tick()

    await fireEvent.keyDown(box(container), { key: 'Escape' })
    await tick()

    expect(ghost(container)).toBeNull()
    expect(box(container).value).toBe('')
  })

  // Typing during a turn goes INTO the running turn (Interject), so a wording
  // offered there would be offering to interrupt the work with an answer to
  // the question before last.
  it('is not drawn while the chat is mid-turn', async () => {
    applyPreparedReplies(two)
    const { container } = render(Chat, { ...baseProps, awaitingReply: true })
    await tick()

    expect(ghost(container)).toBeNull()
  })

  // The event is stamped with the chat that produced it. A wording restored
  // into a composer under somebody else's conversation is a message with no
  // visible question.
  it('is dropped when it belongs to a chat the window is not showing', () => {
    cockpit.openSession = 'on-screen'
    applyPreparedReplies({ sessionId: 'somewhere-else', data: two } as any)
    expect(cockpit.prepared).toHaveLength(0)

    applyPreparedReplies({ sessionId: 'on-screen', data: two } as any)
    expect(cockpit.prepared).toHaveLength(2)
  })

  it('survives a payload with junk in it', () => {
    applyPreparedReplies(undefined as never)
    expect(cockpit.prepared).toEqual([])
    applyPreparedReplies([{ text: '' }, { text: 'เอาอันนี้' }] as any)
    expect(cockpit.prepared).toEqual([{ text: 'เอาอันนี้' }])
  })
})
