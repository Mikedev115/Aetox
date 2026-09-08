// The model row when the provider has no list to offer.
//
// Owner, 9 ก.ย.: "บั๊คเลือกโมเดลไม่ได้" — a screenshot of the composer menu
// with ระดับการอนุมัติ and ผู้ให้บริการ each carrying a dropdown and โมเดล
// carrying dead text. Nothing was broken about the switch: Ollama's server was
// simply not running, so /api/tags answered nothing, and the `ollama` catalog
// row deliberately has no fallback model (a local runtime serves whatever was
// pulled, so any name written there is a guess). ListModelsForProvider returned
// [], which its own doc comment says means "offer a free-text input" — and the
// composer answered it with a <span>. The one row the menu exists to change was
// the only one that could not be changed.
//
// The same shape covers every provider that gates its list: xAI answers 403
// until a team has credits, Model Studio 401 on the wrong region's host.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { ListModelsForProvider } from './mocks/wailsApp'

const switched: string[] = []

const baseProps = {
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
  onSwitchModel: async (m: string) => { switched.push(m) },
  onCancelPendingModel: async () => {},
  onSubmitAPIKey: async () => {},
  model: {
    provider: 'ollama', modelName: 'qwen3.5:9b',
    thinkLevel: 'high', approval: 'full-access', wireFormat: '',
  } as never,
}

async function openMenu() {
  const chip = document.querySelector('.model-chip') as HTMLButtonElement
  expect(chip).toBeTruthy()
  await fireEvent.click(chip)
}

beforeEach(() => {
  vi.clearAllMocks()
  switched.length = 0
  Element.prototype.scrollIntoView = () => {}
  cockpit.chat = []
  cockpit.activeView = 'chat'
  vi.mocked(ListModelsForProvider).mockResolvedValue([] as never)
})

describe('the model row with no discoverable list', () => {
  it('offers a box to type the id in, prefilled with the model in use', async () => {
    render(Chat, baseProps as never)
    await openMenu()

    await waitFor(() => {
      const box = document.querySelector('.mm-idbox input') as HTMLInputElement
      expect(box).toBeTruthy()
      expect(box.value).toBe('qwen3.5:9b')
    })
    // The dead end this replaces.
    expect(document.querySelector('.mm-static')).toBeNull()
  })

  it('switches to whatever was typed', async () => {
    render(Chat, baseProps as never)
    await openMenu()

    const box = await waitFor(() => {
      const el = document.querySelector('.mm-idbox input') as HTMLInputElement
      expect(el).toBeTruthy()
      return el
    })
    await fireEvent.input(box, { target: { value: 'llama4.2:8b' } })
    const use = document.querySelector('.mm-idbox button') as HTMLButtonElement
    await fireEvent.click(use)

    await waitFor(() => expect(switched).toEqual(['llama4.2:8b']))
  })

  // Why there is a box rather than a list, and the way back to a list — a
  // server started after the menu was opened must not need the menu closed.
  it('names the provider and asks again on request', async () => {
    render(Chat, baseProps as never)
    await openMenu()

    await waitFor(() => expect(document.querySelector('.mm-nolist')?.textContent).toContain('ollama'))
    vi.mocked(ListModelsForProvider).mockResolvedValue(['qwen3.5:9b', 'llama4.2:8b'] as never)
    await fireEvent.click(document.querySelector('.mm-nolist button') as HTMLButtonElement)

    await waitFor(() => {
      expect(document.querySelector('.mm-idbox')).toBeNull()
      expect(document.querySelector('.mm-nolist')).toBeNull()
    })
  })

  // A list that is merely slow must not draw the empty shape on its way in.
  it('says it is loading rather than showing the box first', async () => {
    let release: (v: string[]) => void = () => {}
    vi.mocked(ListModelsForProvider).mockReturnValue(
      new Promise<string[]>((r) => { release = r }) as never,
    )
    render(Chat, baseProps as never)
    await openMenu()

    await waitFor(() => expect(document.querySelector('.mm-static')?.textContent).toContain('กำลังโหลด'))
    expect(document.querySelector('.mm-idbox')).toBeNull()
    release(['qwen3.5:9b', 'llama4.2:8b'])
    await waitFor(() => {
      const triggers = Array.from(document.querySelectorAll('.updrop-trigger'))
      expect(triggers.some((b) => b.textContent?.includes('qwen3.5:9b'))).toBe(true)
    })
    expect(document.querySelector('.mm-static')).toBeNull()
  })
})
