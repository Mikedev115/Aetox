import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit, cancelTurn } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics } from './mocks/wailsApp'

vi.mock('../lib/stores/cockpit.svelte', async () => {
  const actual = await vi.importActual('../lib/stores/cockpit.svelte') as any
  return {
    ...actual,
    cancelTurn: vi.fn(),
  }
})

const baseProps = {
  task: { title: '', steps: [] } as any,
  awaitingReply: true,
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

describe('long-running tool execution UX feedback', () => {
  beforeEach(() => {
    setLocale('th')
    cockpit.chat = []
    cockpit.backgroundTasks = []
    cockpit.backgroundSteps = []
    vi.mocked(GuideTopics).mockResolvedValue([] as any)
    vi.mocked(cancelTurn).mockClear()
  })

  it('renders quick-stop button for active running tool step', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{ role: 'user', text: 'run test', time: '10:00' }] as any,
      toolSteps: [
        { label: 'shell npm test', state: 'run', startedAt: Date.now(), secs: 2 },
      ] as any,
    })

    const stopBtn = container.querySelector('.tool-quick-stop')
    expect(stopBtn).not.toBeNull()
    expect(stopBtn?.getAttribute('title')).toBe('หยุดคำสั่งนี้')

    if (stopBtn) {
      await fireEvent.click(stopBtn)
      expect(cancelTurn).toHaveBeenCalled()
    }
  })

  it('marks secs with is-slow when running tool takes >= 15 seconds', async () => {
    const fifteenSecsAgo = Date.now() - 16000
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{ role: 'user', text: 'download', time: '10:00' }] as any,
      toolSteps: [
        { label: 'shell gh run download', state: 'run', startedAt: fifteenSecsAgo, secs: 16 },
      ] as any,
    })

    const secsEl = container.querySelector('.secs')
    expect(secsEl?.classList.contains('is-slow')).toBe(true)
  })

  it('shows stalled pulsing class when tool execution exceeds 30 seconds', () => {
    const thirtyFiveSecsAgo = Date.now() - 35000
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{ role: 'user', text: 'build', time: '10:00' }] as any,
      toolSteps: [
        { label: 'shell wails build', state: 'run', startedAt: thirtyFiveSecsAgo, secs: 35 },
      ] as any,
    })

    const secsEl = container.querySelector('.secs')
    expect(secsEl?.classList.contains('is-slow')).toBe(true)
    expect(secsEl?.classList.contains('is-stalled')).toBe(true)
  })
})
