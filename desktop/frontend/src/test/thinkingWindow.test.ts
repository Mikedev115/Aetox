// The live thinking must be a window, not a wall.
//
// Owner, 6 ก.ย., with a screenshot of Claude Code mid-thought: "ทำ UI ตอน
// โมเดลคิดยาวๆ ประมาณนี้ได้ไหม". A reasoning model writing for two minutes put
// two minutes of grey text into the transcript and pushed the composer off the
// bottom of the screen. The panel is capped and scrolls inside itself now: the
// tail follows, scrolling up inside it stops the follow, and the finished
// panel on a past reply is still drawn in full.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics } from './mocks/wailsApp'

const baseProps = {
  task: { title: '', steps: [] } as any,
  awaitingReply: true,
  agentStatus: '',
  toolSteps: [] as any[],
  streamingText: '',
  reasoningText: ['บรรทัดแรก', 'บรรทัดสอง', 'บรรทัดสาม'].join(String.fromCharCode(10)),
  onSend: () => {},
  onSwitchProvider: async () => {},
  onSwitchThinkLevel: async () => {},
  onSwitchModel: async () => {},
  onSubmitAPIKey: async () => {},
  model: { provider: 'deepseek', modelName: 'v4', thinkLevel: 'high', approval: 'ask', wireFormat: '' } as any,
  messages: [{ role: 'user', text: 'ไปสิ', time: '22:19' }] as any,
}

beforeEach(() => {
  setLocale('en')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  streamed = ''
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
})

/** jsdom has no layout: the box's two read-only dimensions are pinned by hand
 *  so "there is more in it than it can show" is a fact the code can read. */
function thinkBox(container: HTMLElement, scrollHeight: number, clientHeight: number): HTMLDivElement {
  const el = container.querySelector('.reasoning-body.live') as HTMLDivElement
  Object.defineProperty(el, 'scrollHeight', { configurable: true, value: scrollHeight })
  Object.defineProperty(el, 'clientHeight', { configurable: true, value: clientHeight })
  return el
}

/** One animation frame — the paint, and the follow that rides on it. */
const frame = () => new Promise((r) => requestAnimationFrame(() => setTimeout(r, 0)))

/** One more piece of reasoning off the wire, drawn. */
let streamed = ''
async function chunk(rerender: (props: any) => Promise<void> | void, text: string) {
  streamed = streamed ? streamed + String.fromCharCode(10) + text : text
  await rerender({ ...baseProps, reasoningText: baseProps.reasoningText + String.fromCharCode(10) + streamed })
  await tick()
  await frame()
}

describe('live thinking window', () => {
  it('draws the reasoning of a finished reply in full, not windowed', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      awaitingReply: false,
      reasoningText: '',
      messages: [
        { role: 'user', text: 'ไปสิ', time: '22:19' },
        { role: 'assistant', text: 'เสร็จแล้ว', time: '22:20', reasoning: 'คิดยาวมาก' },
      ] as any,
    })
    await tick()
    const toggle = container.querySelector('.reasoning-toggle') as HTMLButtonElement
    await fireEvent.click(toggle)
    await tick()
    const body = container.querySelector('.reasoning-body') as HTMLElement
    expect(body).not.toBeNull()
    expect(body.classList.contains('live')).toBe(false)
  })

  it('follows its own tail while pinned and lets go when scrolled up', async () => {
    const { container, rerender } = render(Chat, baseProps)
    await tick()
    const el = thinkBox(container, 400, 100)

    // a chunk lands: the window scrolls itself to the bottom, and marks itself
    // clipped so the fade is painted
    await chunk(rerender, 'อีกบรรทัด')
    expect(el.scrollTop).toBe(400)
    expect(el.classList.contains('clipped')).toBe(true)

    // the reader scrolls up inside the box — the next chunk must leave it alone
    el.scrollTop = 200
    await fireEvent.scroll(el)
    await chunk(rerender, 'และอีกบรรทัด')
    expect(el.scrollTop).toBe(200)

    // back at the floor, it follows again
    el.scrollTop = 300
    await fireEvent.scroll(el)
    await chunk(rerender, 'บรรทัดสุดท้าย')
    expect(el.scrollTop).toBe(400)
  })

  it('leaves a short thought unmasked — nothing is clipped', async () => {
    const { container } = render(Chat, baseProps)
    await tick()
    const el = thinkBox(container, 100, 100)
    await frame()
    expect(el.classList.contains('clipped')).toBe(false)
  })
})
