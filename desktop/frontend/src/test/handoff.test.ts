// สรุปแล้วไปเริ่มแชทใหม่ (§282): compaction the user can see. The button
// under the last reply asks the engine for the chat as a list of points, the
// card puts every point up ticked, and the ticked ones open a new chat that
// starts from exactly those. These pin the three moments — asking, picking,
// arriving — and the one rule the card shares with every other piece of
// per-chat state: it belongs to the chat it was drafted for.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit, restoreTranscript } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics, DraftHandoff, ContinueInNewSession, LoadSession } from './mocks/wailsApp'

const baseProps = {
  messages: [] as any[],
  task: { title: '', steps: [] } as any,
  awaitingReply: false,
  agentStatus: '',
  toolSteps: [] as any[],
  streamingText: '',
  reasoningText: '',
  onSend: () => {},
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onCancelPendingModel: async () => {},
  onSubmitAPIKey: async () => {},
  model: { provider: 'deepseek', modelName: 'v4', thinkLevel: 'high', approval: 'ask', wireFormat: '' } as any,
}

// A chat with one finished exchange: the button lives under the last reply.
const talked = {
  ...baseProps,
  messages: [
    { role: 'user', text: 'ทำหน้า landing ให้หน่อย', time: '10:00' },
    { role: 'agent', text: 'ได้เลย ทำแล้ว', time: '10:01', id: 7 },
  ] as any[],
}

const points = ['กำลังทำหน้า landing ด้วย Go', 'ตัดสินใจใช้ SQLite', 'ค้าง: หน้า pricing ยังไม่ได้เขียน']

const flush = async () => { await tick(); await new Promise((r) => setTimeout(r, 0)); await tick() }

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.toolSteps = []
  cockpit.ask = null
  cockpit.handoff = null
  cockpit.openSession = 'sess-A'
  cockpit.awaitingReply = false
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
  vi.mocked(DraftHandoff).mockResolvedValue(points)
  vi.mocked(ContinueInNewSession).mockResolvedValue('sess-B')
  vi.mocked(LoadSession).mockResolvedValue([] as any)
})

describe('asking for the list', () => {
  it('the button under the last reply drafts the open chat and shows every point ticked', async () => {
    const { container } = render(Chat, talked)
    await tick()
    const btn = container.querySelector('.handoff-btn') as HTMLButtonElement
    expect(btn).toBeTruthy()
    expect(btn.disabled).toBe(false)

    await fireEvent.click(btn)
    expect(DraftHandoff).toHaveBeenCalledWith('sess-A')
    await flush()

    const ticks = Array.from(container.querySelectorAll('.handoff-tick')) as HTMLInputElement[]
    expect(ticks.map((t) => t.checked)).toEqual([true, true, true])
    expect(container.querySelector('.handoff-pick')?.textContent).toContain('ค้าง: หน้า pricing')
    // The go button counts what is ticked.
    expect(container.querySelector('.handoff-go')?.textContent).toContain('3')
  })

  it('is gated like ตอบใหม่: dimmed while a reply is streaming', async () => {
    const { container } = render(Chat, { ...talked, awaitingReply: true })
    await tick()
    expect((container.querySelector('.handoff-btn') as HTMLButtonElement).disabled).toBe(true)
  })

  it('says why when the engine refuses, and offers to try again', async () => {
    vi.mocked(DraftHandoff).mockRejectedValue(new Error('ยังไม่มีอะไรให้สรุป'))
    const { container } = render(Chat, talked)
    await tick()
    await fireEvent.click(container.querySelector('.handoff-btn') as HTMLButtonElement)
    await flush()
    expect(container.querySelector('.handoff-error')?.textContent).toContain('ยังไม่มีอะไรให้สรุป')
    expect(container.querySelector('.handoff-go')).toBeTruthy()
    expect(container.querySelectorAll('.handoff-tick').length).toBe(0)
  })
})

describe('picking and going', () => {
  it('sends only the ticked points and lands in the chat the engine opened', async () => {
    const { container } = render(Chat, talked)
    await tick()
    await fireEvent.click(container.querySelector('.handoff-btn') as HTMLButtonElement)
    await flush()

    const ticks = Array.from(container.querySelectorAll('.handoff-tick')) as HTMLInputElement[]
    await fireEvent.click(ticks[1])
    await tick()
    expect(container.querySelector('.handoff-go')?.textContent).toContain('2')

    await fireEvent.click(container.querySelector('.handoff-go') as HTMLButtonElement)
    await flush()
    expect(ContinueInNewSession).toHaveBeenCalledWith('sess-A', [points[0], points[2]])
    // Arrival goes through the same door as any reopened session.
    expect(LoadSession).toHaveBeenCalledWith('sess-B')
    expect(cockpit.handoff).toBeNull()
  })

  it('cannot go with nothing ticked', async () => {
    const { container } = render(Chat, talked)
    await tick()
    await fireEvent.click(container.querySelector('.handoff-btn') as HTMLButtonElement)
    await flush()
    for (const t of Array.from(container.querySelectorAll('.handoff-tick'))) await fireEvent.click(t)
    await tick()
    expect((container.querySelector('.handoff-go') as HTMLButtonElement).disabled).toBe(true)
    expect(ContinueInNewSession).not.toHaveBeenCalled()
  })

  it('อยู่แชทนี้ต่อ drops the list and changes nothing', async () => {
    const { container } = render(Chat, talked)
    await tick()
    await fireEvent.click(container.querySelector('.handoff-btn') as HTMLButtonElement)
    await flush()
    await fireEvent.click(container.querySelector('.handoff-cancel') as HTMLButtonElement)
    await tick()
    expect(cockpit.handoff).toBeNull()
    expect(container.querySelector('.handoff-pick')).toBeNull()
    expect(ContinueInNewSession).not.toHaveBeenCalled()
  })

  it('belongs to the chat it was drafted for: another chat on screen does not show it', async () => {
    cockpit.handoff = { session: 'sess-A', points, picked: [true, true, true], busy: false, error: '' }
    cockpit.openSession = 'sess-C'
    const { container } = render(Chat, talked)
    await tick()
    expect(container.querySelector('.handoff-pick')).toBeNull()
  })
})

describe('the continued chat', () => {
  it('opens on a card naming the origin and listing the points, not a bubble', async () => {
    const rows = restoreTranscript([
      { role: 'handoff', text: '- กำลังทำหน้า landing\n- ค้าง: หน้า pricing', time: '10:05', id: 9,
        origin: { id: 'sess-A', title: 'ทำหน้า landing ให้หน่อย' } } as any,
    ])
    expect(rows[0].role).toBe('handoff')
    expect(rows[0].origin?.title).toBe('ทำหน้า landing ให้หน่อย')

    const { container } = render(Chat, { ...baseProps, messages: rows })
    await tick()
    const card = container.querySelector('.handoff-card')
    expect(card).toBeTruthy()
    expect(container.querySelector('.msg.user')).toBeNull()
    const items = Array.from(container.querySelectorAll('.handoff-points li')).map((li) => li.textContent?.trim())
    expect(items).toEqual(['กำลังทำหน้า landing', 'ค้าง: หน้า pricing'])

    // The origin's name is the way back to it.
    const origin = container.querySelector('.handoff-origin') as HTMLButtonElement
    expect(origin.textContent?.trim()).toBe('ทำหน้า landing ให้หน่อย')
    await fireEvent.click(origin)
    await flush()
    expect(LoadSession).toHaveBeenCalledWith('sess-A')
  })

  it('words the gap when the origin has since been deleted', async () => {
    const rows = restoreTranscript([
      { role: 'handoff', text: '- point', time: '10:05', id: 9, origin: { id: 'sess-A', title: '' } } as any,
    ])
    const { container } = render(Chat, { ...baseProps, messages: rows })
    await tick()
    expect(container.querySelector('.handoff-origin')?.textContent?.trim()).toBe('a chat that has since been deleted')
  })
})
