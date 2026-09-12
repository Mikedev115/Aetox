import { describe, it, expect, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'

// The mascot's two homes in the chat, and the seam between the window's live
// signals and the pose it wears (mascot/presence.ts). What is guarded is the
// wiring — that the signals Chat already holds reach the mascot — not the
// drawing, which mascot.test.ts covers.

const baseProps = {
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
}
const model = { provider: 'deepseek', modelName: 'deepseek-flash', thinkLevel: 'low', approval: 'ask', wireFormat: '' }
const reply = (text: string) => ({ role: 'agent', text, time: '10:54' })

beforeEach(() => {
  setLocale('th')
  cockpit.chat = []
  cockpit.ask = null
})

describe('the mascot in the chat', () => {
  // An empty chat greets. The one mascot that follows the pointer is this one.
  it('stands on the welcome screen, greeting, in the brand hue', async () => {
    const { container } = render(Chat, { ...baseProps, model: model as any, messages: [] as any })
    await waitFor(() => expect(container.querySelector('.presence-hero .mascot')).toBeTruthy())
    const m = container.querySelector('.presence-hero .mascot')!
    expect(m.classList.contains('pose-greeting')).toBe(true)
    expect(m.innerHTML).toContain('hsl(218 ')
    expect(container.querySelector('.presence-row')).toBeNull()
  })

  // A live turn with nothing concrete yet is thinking.
  it('sits at the head of the live bubble while a turn runs', async () => {
    const { container } = render(Chat, {
      ...baseProps, model: model as any, messages: [reply('hi')] as any, awaitingReply: true, agentStatus: 'กำลังคิดคำตอบ...',
    })
    await waitFor(() => expect(container.querySelector('.presence-row .mascot')).toBeTruthy())
    expect(container.querySelector('.presence-row .mascot')!.classList.contains('pose-thinking')).toBe(true)
    expect(container.querySelector('.presence-hero')).toBeNull()
  })

  // The running tool decides the pose — the wiring from toolSteps through
  // presenceOf to the class on the mascot.
  it('reads what the running tool is doing', async () => {
    const { container } = render(Chat, {
      ...baseProps, model: model as any, messages: [reply('hi')] as any, awaitingReply: true,
      toolSteps: [{ name: 'read', label: 'read notes.md', state: 'run' }] as any,
    })
    await waitFor(() => expect(container.querySelector('.presence-row .mascot.pose-reading')).toBeTruthy())
  })

  it('answers while the reply streams', async () => {
    const { container } = render(Chat, {
      ...baseProps, model: model as any, messages: [reply('hi')] as any, awaitingReply: true, streamingText: 'สวัส',
    })
    await waitFor(() => expect(container.querySelector('.presence-row .mascot.pose-answering')).toBeTruthy())
  })

  // A question the model is blocked on outranks whatever else is on screen.
  it('asks when the model is blocked on a question', async () => {
    cockpit.ask = { question: 'ใช้ไฟล์ไหน', options: [] }
    const { container } = render(Chat, {
      ...baseProps, model: model as any, messages: [reply('hi')] as any, awaitingReply: true, streamingText: 'x',
    })
    await waitFor(() => expect(container.querySelector('.presence-row .mascot.pose-asking')).toBeTruthy())
  })
})
