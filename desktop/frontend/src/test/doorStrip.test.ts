// The composer while a desk switch is on its way (13 ก.ย. 2026). A door press
// opens a new session, and on a machine where git answers slowly that is
// seconds of a button that looks dead — "กดกลับหน้าผู้ใช้ไม่ได้" (it had; the
// walk was queued). The strip the plan run already uses says where the window
// is going, and nothing about why (owner: "เอาแค่โหลดพอ"); the placeholder
// says typing is fine; the empty room steps back. All of it from one field,
// `cockpit.walkingTo`, so it cannot disagree with the door and the desk row.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render } from '@testing-library/svelte'
import { tick } from 'svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics, ListChairs, ListPromptPresets } from './mocks/wailsApp'

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
  setLocale('th')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  cockpit.plan = null
  cockpit.walkingTo = ''
  cockpit.backgroundTasks = []
  cockpit.backgroundSteps = []
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
  vi.mocked(ListChairs).mockResolvedValue([] as any)
  vi.mocked(ListPromptPresets).mockResolvedValue([] as any)
})

describe('the composer while a desk switch is on its way', () => {
  it('shows the strip with the target desk, the waiting placeholder, and steps the empty room back', async () => {
    const { container } = render(Chat, baseProps)
    expect(container.querySelector('.cbox-door')).toBeNull()

    cockpit.walkingTo = 'coding'
    await tick()
    const strip = container.querySelector('.cbox-door')!
    expect(strip).toBeTruthy()
    expect(strip.textContent).toContain('กำลังเปิดโต๊ะโค้ด')
    // Loading, only: no reason, no count, no button (owner: เอาแค่โหลดพอ).
    expect(strip.textContent).not.toMatch(/git|วิ\)/)
    expect(strip.querySelector('button')).toBeNull()
    expect(strip.querySelector('.cbox-run-track i.indet')).toBeTruthy()
    expect(container.querySelector('.composer .box')?.classList.contains('running')).toBe(true)
    expect((container.querySelector('textarea.input') as HTMLTextAreaElement).placeholder).toBe('พิมพ์ได้เลย จะส่งเมื่อโต๊ะเปิดแล้ว')
    expect(container.querySelector('.empty-state.walking')).toBeTruthy()

    cockpit.walkingTo = ''
    await tick()
    expect(container.querySelector('.cbox-door')).toBeNull()
    expect(container.querySelector('.empty-state.walking')).toBeNull()
    expect(container.querySelector('.composer .box')?.classList.contains('running')).toBe(false)
  })

  it('yields to the plan run strip rather than stacking a second one', async () => {
    cockpit.plan = { running: true, title: 'แผน', steps: [] } as any
    const { container } = render(Chat, baseProps)
    cockpit.walkingTo = 'assistant'
    await tick()
    expect(container.querySelector('.cbox-door')).toBeNull()
    expect(container.querySelectorAll('.cbox-run').length).toBe(1)
  })
})
