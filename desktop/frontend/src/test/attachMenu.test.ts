// The + button on the composer, and the ways into its menu.
//
// On 31 ส.ค. an owner could not attach a .docx. Nothing in the attachment path
// refused it: the copy takes any file and `read` opens Office documents. The
// only thing missing was the extension, from one hand-written pattern string
// inside the native dialog's collapsed filter dropdown — a place where a type
// that is absent and a type the app cannot take look exactly the same. So the
// list moved into the app. On 12 ก.ย. the same button became the one door for
// everything that goes into the message (§257): the "/" button is gone, Ctrl+K
// opens this, and "/" typed into an empty box opens it on the presets.
//
// These tests hold the composer's side of that: the button opens and closes
// the menu, every way in lands on the same component, the file tiles still ask
// the Go side for their own filter, and the menu closes its neighbour rather
// than drawing over it.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics, ListChairs, ListPromptPresets, PickAttachments } from './mocks/wailsApp'

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

beforeEach(() => {
  setLocale('en')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  cockpit.backgroundTasks = []
  cockpit.backgroundSteps = []
  cockpit.pendingImages = []
  cockpit.pendingFiles = []
  cockpit.pendingContexts = []
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
  vi.mocked(ListChairs).mockResolvedValue([] as any)
  vi.mocked(ListPromptPresets).mockResolvedValue([] as any)
  vi.mocked(PickAttachments).mockClear()
  vi.mocked(PickAttachments).mockResolvedValue([] as any)
})

function plus(container: HTMLElement): HTMLButtonElement {
  return container.querySelector('.attach-pick > .icobtn') as HTMLButtonElement
}

function tiles(container: HTMLElement): HTMLButtonElement[] {
  return Array.from(container.querySelectorAll('.palette .pal-tile'))
}

async function openMenu(container: HTMLElement) {
  await fireEvent.click(plus(container))
  await tick()
}

describe('the + button', () => {
  it('opens its menu and closes it again', async () => {
    const { container } = render(Chat, baseProps as any)
    expect(container.querySelector('.palette')).toBeNull()

    await openMenu(container)
    expect(container.querySelector('.palette')).not.toBeNull()
    // Nothing narrowed: the button is the whole menu.
    expect(container.querySelector('.pal-mode')).toBeNull()

    await fireEvent.click(plus(container))
    await tick()
    expect(container.querySelector('.palette')).toBeNull()
  })

  // The whole point of moving the list out of the dialog: what the app takes
  // is readable without opening anything, and each tile says what the
  // assistant will do with that kind of file rather than only naming it.
  it('names every kind this app accepts, and what happens to it', async () => {
    const { container } = render(Chat, baseProps as any)
    await openMenu(container)

    expect(tiles(container).map((r) => r.querySelector('.nm')?.textContent)).toEqual([
      'Images', 'Documents', 'Video and audio', 'Other files',
    ])
    for (const tile of tiles(container)) {
      expect(tile.getAttribute('title')?.trim()).toBeTruthy()
    }
    // The type that started this has to be visible without opening a dialog.
    expect(tiles(container)[1].getAttribute('title')).toContain('docx')
    // Drag and Ctrl+V are the other two ways in, and neither has a button.
    expect(container.querySelector('.palette .pal-foot')?.textContent).toContain('Ctrl+V')
  })

  // Both menus open at once, drawn on top of each other (owner, 31 ส.ค., with
  // the screenshot). Every trigger on this row stops its click reaching the
  // outside-click closer — it has to, or the click that opens a menu closes it
  // again — so nothing was clearing the neighbour.
  it('closes the menu beside it rather than drawing over it', async () => {
    const { container } = render(Chat, baseProps as any)

    await fireEvent.click(container.querySelector('.stance-chip') as HTMLButtonElement)
    await tick()
    expect(container.querySelector('.stance-menu')).not.toBeNull()

    await openMenu(container)
    expect(container.querySelector('.palette')).not.toBeNull()
    expect(container.querySelector('.stance-menu')).toBeNull()

    // ...and back, so neither one is the privileged half of the pair.
    await fireEvent.click(container.querySelector('.stance-chip') as HTMLButtonElement)
    await tick()
    expect(container.querySelector('.palette')).toBeNull()
    expect(container.querySelector('.stance-menu')).not.toBeNull()
  })

  // A tile that opened the same unfiltered dialog as its neighbour would put
  // the user back in front of the dropdown this menu exists to replace.
  it('asks the dialog for its own tile, and closes behind itself', async () => {
    const { container } = render(Chat, baseProps as any)

    for (const [index, group] of ['image', 'document', 'media', ''].entries()) {
      await openMenu(container)
      await fireEvent.click(tiles(container)[index])
      await tick()

      expect(vi.mocked(PickAttachments).mock.calls.at(-1)?.[0]).toBe(group)
      expect(container.querySelector('.palette')).toBeNull()
    }
    expect(vi.mocked(PickAttachments)).toHaveBeenCalledTimes(4)
  })

  it('Ctrl+K is the same menu, and "/" in an empty box is the same menu on the presets', async () => {
    const { container } = render(Chat, baseProps as any)

    await fireEvent.keyDown(window, { key: 'k', code: 'KeyK', ctrlKey: true })
    await tick()
    expect(container.querySelector('.palette')).not.toBeNull()
    expect(container.querySelector('.pal-mode')).toBeNull()
    await fireEvent.keyDown(window, { key: 'k', code: 'KeyK', ctrlKey: true })
    await tick()
    expect(container.querySelector('.palette')).toBeNull()

    const input = container.querySelector('textarea.input') as HTMLTextAreaElement
    await fireEvent.keyDown(input, { key: '/' })
    await tick()
    expect(container.querySelector('.palette')).not.toBeNull()
    expect(container.querySelector('.pal-mode')).not.toBeNull()
    expect(container.querySelector('.palette .pal-tile')).toBeNull()
  })

  // The "/" button is gone; there is one glyph button on the row and it is +.
  it('has no "/" button left on the row', () => {
    const { container } = render(Chat, baseProps as any)
    expect(container.querySelector('.composer .icobtn.slash')).toBeNull()
    expect(plus(container).textContent).toBe('+')
  })
})
