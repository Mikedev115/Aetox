// External conversations must stay visually distinct after they are opened in
// Aetox history. The mark is the real service logo shared with Connections,
// and an ordinary Aetox conversation carries no badge at all.
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, waitFor } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { GuideTopics } from './mocks/wailsApp'

const message = { role: 'agent', text: 'เรียบร้อยครับ', id: 1, time: '2 นาที' }
const props = {
  task: { title: '', steps: [] },
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
  model: { provider: 'codex', modelName: 'gpt-test', thinkLevel: 'medium', approval: 'ask', wireFormat: '' },
  messages: [message],
} as any

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
  cockpit.openingSession = ''
  cockpit.chat = [message] as any
  cockpit.transport = ''
})

describe('external chat mark', () => {
  for (const [transport, label] of [['telegram', 'Telegram'], ['discord', 'Discord']] as const) {
    it(`shows the real ${label} mark on its bubbles`, async () => {
      cockpit.transport = transport
      const { container } = render(Chat, props)

      await waitFor(() => expect(container.querySelector('.msg-transport')).not.toBeNull())
      const badge = container.querySelector('.msg-transport')!
      expect(badge.textContent).toContain(label)
      expect(badge.querySelector('.connection-mark')?.getAttribute('data-connection-mark')).toBe(transport)
    })
  }

  it('does not label an ordinary Aetox conversation as external', async () => {
    const { container } = render(Chat, props)
    await waitFor(() => expect(container.querySelector('.msg.bot')).not.toBeNull())
    expect(container.querySelector('.msg-transport')).toBeNull()
  })
})
