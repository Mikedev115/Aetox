import { describe, it, expect, beforeEach } from 'vitest'
import { render } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { toolSubject } from '../lib/toolFace'

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

describe('parallel tool fan-out card', () => {
  beforeEach(() => {
    cockpit.chat = []
    cockpit.toolSteps = []
  })

  it('fans out multiple tool calls into parallel-card matching search-card pattern', () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{ role: 'user', text: 'run tests', time: '10:50' }] as any,
      awaitingReply: true,
      toolSteps: [
        { label: 'shell go test ./... -count=1', name: 'shell', subject: 'go test ./... -count=1', state: 'run', startedAt: Date.now() },
        { label: 'git status', name: 'git', subject: 'status', state: 'run', startedAt: Date.now() },
      ],
    })

    const card = container.querySelector('.parallel-card')
    expect(card).toBeTruthy()
    expect(card?.classList.contains('search-card')).toBe(true)

    const head = card?.querySelector('.parallel-head')
    expect(head).toBeTruthy()
    expect(head?.textContent).toContain('2')

    const hits = card?.querySelectorAll('.parallel-hit')
    expect(hits?.length).toBe(2)

    const foot = card?.querySelector('.parallel-foot')
    expect(foot).toBeTruthy()
  })

  it('resolves shell command for shell_output with no subject', () => {
    const steps: any[] = [
      { name: 'shell', subject: 'go test ./... -count=1', state: 'done', startedAt: 1000 },
      { name: 'shell', act: 'output', subject: '', state: 'run', startedAt: 2000 },
    ]

    const resolved = toolSubject(steps[1], steps)
    expect(resolved).toBe('go test ./... -count=1')
  })
})
