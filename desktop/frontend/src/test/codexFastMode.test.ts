import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, waitFor } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import {
  ListModelsForProvider,
  ServiceTiersFor,
  SwitchServiceTier,
} from './mocks/wailsApp'

const props = {
  task: { title: '', steps: [] } as never,
  messages: [] as never,
  awaitingReply: false,
  agentStatus: '',
  toolSteps: [] as never,
  streamingText: '',
  reasoningText: '',
  onSend: () => {},
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onCancelPendingModel: async () => {},
  onSubmitAPIKey: async () => {},
  model: {
    provider: 'codex', modelName: 'gpt-5.6-luna', thinkLevel: 'low',
    serviceTier: '', approval: 'ask', wireFormat: '', warning: '', pending: null,
    contextUsed: 0, contextMax: 0,
  },
}

beforeEach(() => {
  vi.clearAllMocks()
  Element.prototype.scrollIntoView = () => {}
  cockpit.chat = []
  cockpit.activeView = 'chat'
  vi.mocked(ListModelsForProvider).mockResolvedValue(['gpt-5.6-luna'] as never)
  vi.mocked(ServiceTiersFor).mockResolvedValue([{
    id: 'priority', name: 'Fast', description: '1.5x speed, increased usage',
  }] as never)
})

describe('Codex Fast mode', () => {
  it('uses the model catalog multiplier and sends the priority tier', async () => {
    render(Chat, props as never)
    await fireEvent.click(document.querySelector('.model-chip') as HTMLButtonElement)

    const fastTrigger = await waitFor(() => {
      const row = Array.from(document.querySelectorAll('.mm-row'))
        .find((item) => item.textContent?.includes('โหมดความเร็ว') || item.textContent?.includes('Speed mode'))
      expect(row).toBeTruthy()
      return row!.querySelector('.updrop-trigger') as HTMLButtonElement
    })
    await fireEvent.click(fastTrigger)

    const fast = await waitFor(() => {
      const option = Array.from(document.querySelectorAll('.updrop-opt'))
        .find((item) => item.textContent?.includes('Fast 1.5×')) as HTMLButtonElement
      expect(option).toBeTruthy()
      return option
    })
    await fireEvent.click(fast)

    await waitFor(() => expect(SwitchServiceTier).toHaveBeenCalledWith('priority'))
  })
})
