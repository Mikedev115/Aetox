// A model switch asked for while the turn is still running.
//
// The dials used to refuse: "เอเจนกำลังทำงานอยู่ — รอให้เสร็จ หรือกดหยุดก่อน แล้วค่อย
// สลับโมเดล", which is a decision handed back to the person who had just made
// it. They queue now (DECISIONS §232) — Go parks the config and swaps the
// engine at the turn boundary — and that moves the whole question onto this
// screen: a switch that is going to happen later has to be visible, has to name
// what it will be, and has to be takeable back.
//
// The one rule these tests exist to hold: the chip says what is answering RIGHT
// NOW. A queued model's name on it would be read as the model doing the
// talking, which is the exact confusion the queue was supposed to remove.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'

const cancelled = vi.fn(async () => {})

const baseProps = (pending: Record<string, string> | null) => ({
  task: { title: '', steps: [] } as never,
  messages: [] as never,
  // The state the queue only ever exists in.
  awaitingReply: true,
  agentStatus: '',
  toolSteps: [] as never,
  streamingText: '',
  reasoningText: '',
  onSend: () => {},
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onCancelPendingModel: cancelled,
  onSubmitAPIKey: async () => {},
  model: {
    provider: 'deepseek', modelName: 'deepseek-chat', thinkLevel: 'high',
    approval: 'ask', wireFormat: '', warning: '', contextUsed: 0, contextMax: 0,
    pending,
  } as never,
})

const openMenu = async () => {
  const chip = document.querySelector('.model-chip') as HTMLButtonElement
  expect(chip).toBeTruthy()
  await fireEvent.click(chip)
}

beforeEach(() => {
  vi.clearAllMocks()
  Element.prototype.scrollIntoView = () => {}
  cockpit.chat = []
  cockpit.activeView = 'chat'
})

describe('a model switch queued behind the running turn', () => {
  it('names the queued model in the menu, and leaves the chip naming the one answering', async () => {
    render(Chat, baseProps({
      provider: 'deepseek', modelName: 'deepseek-reasoner', thinkLevel: 'high', wireFormat: '',
    }) as never)
    await openMenu()

    const queued = await waitFor(() => {
      const el = document.querySelector('.mm-queued')
      expect(el).toBeTruthy()
      return el as HTMLElement
    })
    expect(queued.textContent).toContain('deepseek-reasoner')

    // The chip itself: still the model that is talking. Its own <span class="t">
    // rather than the whole chip, because the queued mark lives on it too and
    // carries the other name in a title attribute.
    const name = document.querySelector('.model-chip .t') as HTMLElement
    expect(name.textContent).toContain('deepseek-chat')
    expect(name.textContent).not.toContain('reasoner')
    // And it says, without words, that something is waiting.
    expect(document.querySelector('.model-chip .queued')).toBeTruthy()
  })

  it('takes the switch back when the queued row is dismissed', async () => {
    render(Chat, baseProps({
      provider: 'deepseek', modelName: 'deepseek-reasoner', thinkLevel: 'high', wireFormat: '',
    }) as never)
    await openMenu()

    const x = await waitFor(() => {
      const el = document.querySelector('.mm-queued .q-x')
      expect(el).toBeTruthy()
      return el as HTMLButtonElement
    })
    await fireEvent.click(x)

    expect(cancelled).toHaveBeenCalledTimes(1)
  })

  // The preflight, which is the other half of "เตรียมรอเลย": the switch is
  // rehearsed against the real endpoint while the old turn keeps answering, so
  // a key that was never set says so here rather than at the boundary.
  it('shows the preflight verdict, and the failing provider’s own words', async () => {
    render(Chat, baseProps({
      provider: 'deepseek', modelName: 'deepseek-reasoner', thinkLevel: 'high', wireFormat: '',
      check: 'failed', note: 'invalid api key',
    }) as never)
    await openMenu()

    const why = await waitFor(() => {
      const el = document.querySelector('.mm-queued-why')
      expect(el).toBeTruthy()
      return el as HTMLElement
    })
    expect(why.textContent).toContain('invalid api key')
    expect(document.querySelector('.mm-queued .q-s.bad')).toBeTruthy()
  })

  // Proven reachable reads as quiet reassurance, not as an alarm — and the
  // queued row still says what it always said.
  it('marks a proved switch without a reason line', async () => {
    render(Chat, baseProps({
      provider: 'deepseek', modelName: 'deepseek-reasoner', thinkLevel: 'high', wireFormat: '',
      check: 'ready', note: 'deepseek-reasoner · 412ms',
    }) as never)
    await openMenu()

    await waitFor(() => expect(document.querySelector('.mm-queued .q-s.ok')).toBeTruthy())
    expect(document.querySelector('.mm-queued-why')).toBeNull()
  })

  // Nothing queued is the ordinary state, including mid-turn: the park slot in
  // Go is shared with every other door that rebuilds an engine, and most of what
  // lands in it moves no dial. A row drawn from a parked MCP toggle would be the
  // app announcing a decision nobody made.
  it('draws nothing when no switch is waiting', async () => {
    render(Chat, baseProps(null) as never)
    await openMenu()

    await waitFor(() => expect(document.querySelector('.model-menu')).toBeTruthy())
    expect(document.querySelector('.mm-queued')).toBeNull()
    expect(document.querySelector('.model-chip .queued')).toBeNull()
  })
})
