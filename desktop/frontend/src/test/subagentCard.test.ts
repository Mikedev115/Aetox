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
  onCancelPendingModel: async () => {},
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

// The one list in the app that had no cap on it, which is exactly backwards: a
// delegate is the worker that can run for twenty minutes, so its list is the
// one that grows longest. Same window the thinking panel has had since it had
// the same problem — capped and scrolling while it is being written to, whole
// once it is a record.
describe('the delegate’s own tool list', () => {
  const withChildren = [
    delegation,
    { label: 'read a.go', parent: 'call_1', task: 'task_1', state: 'done', startedAt: 0, secs: 1 },
    { label: 'read b.go', parent: 'call_1', task: 'task_1', state: 'run', startedAt: 0 },
  ]

  it('caps itself while the delegate is still working', async () => {
    const { container } = render(Chat, { ...baseProps, toolSteps: withChildren } as any)
    await fireEvent.click(container.querySelector('.bgw-open') as HTMLElement)
    expect(container.querySelector('.bgw-work.live-window')).toBeTruthy()
  })

  // Read off `state`, not off which turn is on screen: a delegation outlives
  // the turn that started it, and a stopped one is a record wherever it sits.
  it('shows itself whole once the delegate has stopped', async () => {
    cockpit.backgroundTasks = [registered({ state: 'done', elapsedMs: 9_000 })]
    const { container } = render(Chat, { ...baseProps, toolSteps: withChildren } as any)
    await fireEvent.click(container.querySelector('.bgw-open') as HTMLElement)
    expect(container.querySelector('.bgw-work')).toBeTruthy()
    expect(container.querySelector('.bgw-work.live-window')).toBeNull()
  })
})

// A tool row is a thing the agent did, and folds into a count the way a receipt
// does. A delegation is somebody ELSE'S work — a face, and a brief they were
// handed — and "ซับเอเจน 2 ตัว" cannot stand in for that: it names how many,
// which is the least interesting fact about them. So the fold is the agent's
// own rows and only those, on a live turn and on one read back a week later.
describe('a delegation is never folded', () => {
  const storedTurn = {
    role: 'agent', text: 'ok', time: '10:54',
    parts: [{ kind: 'text', text: 'ok' }],
    steps: [
      { label: 'read a.go', ref: 'call_0', state: 'done', startedAt: 0, secs: 1 },
      { ...delegation, state: 'done' },
    ],
  }

  it('is on screen the moment a finished turn is drawn, with nothing to click', () => {
    cockpit.backgroundTasks = []
    const { container } = render(Chat, { ...baseProps, awaitingReply: false, messages: [storedTurn] } as any)

    expect(container.querySelector('.bgw-card')).toBeTruthy()
    // The agent's own row is still behind the count — that half is unchanged.
    expect(container.querySelector('.tool-step')).toBeNull()
    expect(container.querySelector('button.phase-head')?.getAttribute('aria-expanded')).toBe('false')
  })

  it('stays on screen when the reader closes the fold again', async () => {
    cockpit.backgroundTasks = []
    const { container } = render(Chat, { ...baseProps, awaitingReply: false, messages: [storedTurn] } as any)
    const head = container.querySelector('button.phase-head') as HTMLElement

    await fireEvent.click(head)
    expect(container.querySelectorAll('.tool-step').length).toBe(1)
    expect(container.querySelector('.bgw-card')).toBeTruthy()
    await fireEvent.click(head)
    expect(container.querySelector('.tool-step')).toBeNull()
    expect(container.querySelector('.bgw-card')).toBeTruthy()
  })
})

// The agent sitting and waiting on somebody else. A real call, and not work:
// what it redeems has its own card, its own face and its own clock right below
// it, so once it comes back the row is the card said twice and a count pushed
// up by one.
describe('the row that waits for a delegate', () => {
  // `delegation: false` is what the engine sends: only STARTING one hires
  // anybody, so a collect arrives explicitly saying it did not (delegationOf in
  // internal/turn/executor.go). Without it the row's label alone would make the
  // window guess it was a delegation and draw a second card.
  const collect = (over = {}) => ({
    label: 'task', name: 'task', act: 'collect', ref: 'call_2', delegation: false,
    task: 'task_1', state: 'done', startedAt: 0, secs: 13, ...over,
  })

  it('is on screen while the agent is actually blocked on it', () => {
    const { container } = render(Chat, {
      ...baseProps,
      toolSteps: [delegation, collect({ state: 'run', secs: undefined })],
    } as any)
    const rows = [...container.querySelectorAll('.tool-step')].map((r) => r.textContent)
    expect(rows.join(' ')).toContain('รอผลงาน')
  })

  it('is gone once it comes back, and takes its place in the count with it', () => {
    cockpit.backgroundTasks = [registered({ state: 'done', elapsedMs: 13_000 })]
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: false,
      messages: [{
        role: 'agent', text: 'ok', time: '10:54',
        parts: [{ kind: 'text', text: 'ok' }],
        steps: [{ ...delegation, state: 'done' }, collect()],
      }] as any,
    })
    expect(container.textContent).not.toContain('รอผลงาน')
    // One delegation and nothing else: the collect is not a tool the agent used.
    expect(container.querySelector('.phase-head')?.textContent).not.toContain('เครื่องมือ')
  })

  // The thirteen seconds were the delegate's all along. They belong on the
  // delegate's card, not on an anonymous row above it — and the card's own row
  // cannot supply them, because a `task` start returns the instant the worker
  // is spawned.
  it('hands its seconds to the card of the delegate it waited for', () => {
    cockpit.backgroundTasks = []
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: false,
      messages: [{
        role: 'agent', text: 'ok', time: '10:54',
        parts: [{ kind: 'text', text: 'ok' }],
        steps: [{ ...delegation, state: 'done', secs: 0 }, collect()],
      }] as any,
    })
    expect(container.querySelector('.bgw-card .bgw-clock')?.textContent?.trim()).toBe('13s')
  })
})
