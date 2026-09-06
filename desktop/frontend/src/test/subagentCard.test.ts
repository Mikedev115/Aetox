// The delegation card, rebuilt around the work instead of the name.
//
// The card it replaces led with WHO, and who is the one thing about a
// delegation that cannot change: a name, a pill repeating that name's state,
// and a clock across the top, the job under them, and the work itself behind a
// fold. Nothing on it moved. Adding a portrait to the corner of that changed
// nothing either — owner, over exactly that version: *"แทบไม่ต่างจากอันเก่า"*,
// after the diagnosis it came from: *"ทั้งที่มีอวตารแล้วทำไมยังเงียบแบบเดิม
// อยู่"*. เงียบ. Silent, not ugly.
//
// So what these pin is the inversion, and the two things it is built out of:
// the line that changes every few seconds, and the sentence a finished
// delegate leaves behind. Both are DERIVED from rows that were already being
// stored, which is why a turn written months ago has them too.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics } from './mocks/wailsApp'
import { currentStep, tally } from '../lib/delegateWork'
import type { BackgroundTask, ToolStep } from '../lib/types'

const step = (over: Partial<ToolStep> = {}): ToolStep =>
  ({ label: 'read a.go', state: 'done', startedAt: 0, ...over }) as ToolStep

describe('what a delegate touched', () => {
  it('tells a file it read from a file it changed', () => {
    expect(tally([
      step({ label: 'read a.go' }),
      step({ label: 'read b.go' }),
      step({ label: 'edit c.go', added: 9, removed: 2 }),
    ])).toEqual({ read: 2, wrote: 1 })
  })

  // The engine stamps a write three ways (added/removed/diff) and stamps a read
  // no way at all, so `wrote` asks the row and `read` asks the tool's name.
  // Any of the three stamps is enough: an edit that only deletes lines carries
  // no `added`, and a tool added next year carries all of them without anybody
  // remembering this file exists.
  it('takes any of the engine’s three marks for a write', () => {
    expect(tally([step({ label: 'edit only-removed.go', removed: 4 })]).wrote).toBe(1)
    expect(tally([step({ label: 'write from-diff.go', diff: '@@ -1 +1 @@' })]).wrote).toBe(1)
  })

  // Counted per file, not per call. A delegate that read one file four times
  // read one file, and "อ่าน 4 ไฟล์" over a card would be inflating the only
  // numbers on it.
  it('counts a file once however often it was opened', () => {
    expect(tally([
      step({ label: 'read gate.py' }),
      step({ label: 'read gate.py' }),
      step({ label: 'read gate.py' }),
    ])).toEqual({ read: 1, wrote: 0 })
  })

  // A grep carries `count` — "how much came back in the tool's own unit" —
  // which is why the read side is a list of tool names and not that field: 40
  // matches across the repo is not 40 files anybody read. A name this build
  // does not know counts as neither, the same fallback workerFace already
  // follows: undercounting is checkable, guessing is not.
  it('counts neither for a tool that is not a read and did not write', () => {
    expect(tally([
      step({ label: 'grep needle', count: 40 }),
      step({ label: 'shell go build' }),
    ])).toEqual({ read: 0, wrote: 0 })
  })

  it('leaves narration and thinking out of it', () => {
    expect(tally([
      step({ kind: 'note', label: 'read the docs first' }),
      step({ kind: 'thinking', label: '' }),
      step({ label: 'read real.go' }),
    ])).toEqual({ read: 1, wrote: 0 })
  })
})

describe('the line a card is showing', () => {
  it('is the newest row still running', () => {
    expect(currentStep([
      step({ label: 'read a.go', state: 'done' }),
      step({ label: 'read b.go', state: 'run' }),
    ])?.label).toBe('read b.go')
  })

  // A delegate between two calls has no running row at all, and a card that
  // blanked for that second would flicker on every step boundary. The last
  // finished row holds the line until the next one starts.
  it('holds the last finished row when nothing is running', () => {
    expect(currentStep([
      step({ label: 'read a.go', state: 'done' }),
      step({ label: 'read b.go', state: 'done' }),
    ])?.label).toBe('read b.go')
  })

  it('is nothing at all before the first row arrives', () => {
    expect(currentStep([])).toBeUndefined()
  })
})

const baseProps = {
  messages: [{ role: 'agent', text: 'ok', time: '10:54' }] as any,
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
  onSubmitAPIKey: async () => {},
  model: { provider: 'ollama', modelName: 'gemma4:31b', thinkLevel: 'high', approval: 'ask', wireFormat: '' } as any,
}

const delegation = {
  label: 'task ตรวจ SKILL.md ให้ตรงกับโค้ด', ref: 'call_1', task: 'task_1',
  delegation: true, agent: 'deepresearch', agentKind: 'agent',
  brief: 'กางข้อกล่าวอ้างออกทีละข้อ แล้วหาโค้ดที่ยืนยันหรือหักล้าง',
  state: 'run', startedAt: Date.now(),
}

const registered = (over: Partial<BackgroundTask> = {}): BackgroundTask => ({
  id: 'task_1', agent: 'deepresearch', label: 'ตรวจ SKILL.md ให้ตรงกับโค้ด',
  startedAt: new Date().toISOString(), toolCalls: 3, tokens: 0,
  tokensIn: 0, tokensOut: 0, cachedIn: 0, cacheReported: false,
  state: 'running', collected: false, ...over,
})

// A finished delegation folds away with the rest of the finished work, wherever
// that fold happens to be — behind the phase header in a turn that recorded its
// sequence, behind the "Agents" count in one stored before phases existed.
const openFinished = async (container: Element) => {
  const toggle = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
    .find((b) => b.textContent?.includes('เอเจน'))
  if (toggle) await fireEvent.click(toggle)
  const head = container.querySelector('button.phase-head[aria-expanded="false"]')
  if (head) await fireEvent.click(head)
  return container.querySelector('.tool-steps .bgw-card')
}

beforeEach(() => {
  setLocale('th')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  cockpit.backgroundRuns = []
  cockpit.backgroundSteps = []
  cockpit.backgroundTasks = [registered()]
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
})

describe('the delegation card in the transcript', () => {
  it('leads with the row the delegate is on, not with its name', () => {
    const { container } = render(Chat, {
      ...baseProps,
      toolSteps: [
        delegation,
        { label: 'read profile.go', parent: 'call_1', task: 'task_1', state: 'done', startedAt: 0 },
        { label: 'read dispatcher.go', parent: 'call_1', task: 'task_1', state: 'run', startedAt: 0 },
      ],
    } as any)

    const card = container.querySelector('.bgw-card.run')
    expect(card?.querySelector('.bgw-now')?.textContent?.trim()).toBe('read dispatcher.go')
    // The name is still on the card — it moved to the caption under the line,
    // written the way the user would write it (an เอเจน has one address).
    expect(card?.querySelector('.bgw-who')?.textContent).toContain('@deepresearch')
  })

  // The brief is the whole reason a delegate did what it did, and the one thing
  // the user never otherwise sees: it is written by the main agent, not typed
  // by them. Promoting the live line must not cost it — which is exactly why
  // the "collapse to one line" shape was turned down.
  it('keeps the job and the brief it was handed while it works', () => {
    const { container } = render(Chat, {
      ...baseProps,
      toolSteps: [delegation, { label: 'read dispatcher.go', parent: 'call_1', task: 'task_1', state: 'run', startedAt: 0 }],
    } as any)

    const told = container.querySelector('.bgw-told')
    expect(told?.querySelector('.bgw-brief')?.textContent).toContain('ตรวจ SKILL.md')
    expect(told?.querySelector('.bgw-longbrief')?.textContent).toContain('ยืนยันหรือหักล้าง')
  })

  // A finished card used to say nothing but a tool count. This is the one
  // sentence somebody actually wants out of a delegate, and it is derived, so
  // no turn stored before today is missing it.
  it('leads with what it came back with once the work is over', async () => {
    cockpit.backgroundTasks = [registered({ state: 'done', elapsedMs: 48_000 })]
    const { container } = render(Chat, {
      ...baseProps,
      toolSteps: [
        delegation,
        { label: 'read a.go', parent: 'call_1', task: 'task_1', state: 'done', startedAt: 0 },
        { label: 'read b.go', parent: 'call_1', task: 'task_1', state: 'done', startedAt: 0 },
        { label: 'edit c.go', parent: 'call_1', task: 'task_1', state: 'done', startedAt: 0, added: 4, removed: 1 },
      ],
    } as any)

    const card = await openFinished(container)
    const now = card?.querySelector('.bgw-now')?.textContent?.trim()
    expect(now).toContain('อ่าน 2 ไฟล์')
    expect(now).toContain('แก้ 1 ไฟล์')
    // And the job it was given is still there, one line down.
    expect(card?.querySelector('.bgw-told .bgw-brief')?.textContent).toContain('ตรวจ SKILL.md')
  })

  // A delegate that only searched, or only ran a build, has no tally to state.
  // Falling back to the job is the honest resting headline; "อ่าน 0 ไฟล์"
  // would be a measurement of nothing, and the card would have said it twice.
  it('falls back to the job when there is no tally to state', async () => {
    cockpit.backgroundTasks = [registered({ state: 'done', elapsedMs: 3_000 })]
    const { container } = render(Chat, {
      ...baseProps,
      toolSteps: [delegation, { label: 'grep needle', parent: 'call_1', task: 'task_1', state: 'done', startedAt: 0, count: 12 }],
    } as any)

    const card = await openFinished(container)
    expect(card?.querySelector('.bgw-now')?.textContent?.trim()).toBe('ตรวจ SKILL.md ให้ตรงกับโค้ด')
    // Not printed twice in two sizes: the rail below drops the job line when
    // the headline is already showing it.
    expect(card?.querySelector('.bgw-told .bgw-brief')).toBeNull()
    expect(card?.querySelector('.bgw-told .bgw-longbrief')).toBeTruthy()
  })

  // Asked for and not begun is its own thing, and the register is the only
  // place that knows. A ticking clock over a worker that has not started is the
  // same lie the tray was fixed of.
  it('says nothing is happening yet on a delegation still in the line', () => {
    cockpit.backgroundTasks = [registered({ state: 'queued', toolCalls: 0 })]
    const { container } = render(Chat, { ...baseProps, toolSteps: [delegation] } as any)

    const card = container.querySelector('.bgw-card.is-queued')
    expect(card?.querySelector('.bgw-now')?.textContent?.trim()).toBe('ตรวจ SKILL.md ให้ตรงกับโค้ด')
    expect(card?.querySelector('.bgw-clock')).toBeNull()
    expect(card?.querySelector('.bgw-who')?.textContent).toContain('รอคิว')
  })

  // The beam is a CSS gradient: it says nothing to a screen reader and stops
  // moving under prefers-reduced-motion. The word is what those two readers
  // have instead, so promoting the live line must not take it away.
  it('still says "working" in words while it works', () => {
    const { container } = render(Chat, { ...baseProps, toolSteps: [delegation] } as any)
    expect(container.querySelector('.bgw-card.run .bgw-badge.run')?.textContent).toContain('กำลังทำงาน')
  })

  // The spend the register has been holding all along and the transcript's card
  // never said (§163.3) — the tray's has shown it since the brake landed, and
  // these two are deliberately one card.
  it('shows what it has spent when the register can say', () => {
    cockpit.backgroundTasks = [registered({ tokensIn: 41_200, tokensOut: 830 })]
    const { container } = render(Chat, { ...baseProps, toolSteps: [delegation] } as any)
    const meta = container.querySelector('.bgw-card .bgw-meta')?.textContent ?? ''
    expect(meta).toContain('41.2k')
    expect(meta).toContain('830')
  })

  it('says nothing about spend before the first round comes back', () => {
    const { container } = render(Chat, { ...baseProps, toolSteps: [delegation] } as any)
    expect(container.querySelector('.bgw-card .bgw-meta')).toBeNull()
  })
})
