import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Chat from '../lib/Chat.svelte'
import { cockpit, applyToolEvent, resetBackgroundWork } from '../lib/stores/cockpit.svelte'
import { isDelegation } from '../lib/types'
import { setLocale } from '../lib/i18n.svelte'
import { GuideTopics, SwitchApprovalMode, SupportedThinkLevels, BackgroundTasks } from './mocks/wailsApp'
import { workbench } from '../lib/stores/workbench.svelte'

const baseProps = {
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

const step = (label: string, state: 'run' | 'done' | 'err') => ({ label, state, startedAt: 0, secs: 1 })

beforeEach(() => {
  setLocale('en')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  // The turn timeline and the background tray draw the same card now (§105.5),
  // so a leftover background task would be counted as one of the turn's own.
  cockpit.backgroundTasks = []
  cockpit.backgroundSteps = []
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
})

// turn.ToolEvent carries the outcome as a field. It used to be a formatted
// string the frontend matched the Thai word "สำเร็จ" against, so translating
// that word would have marked every call failed.
describe('tool events from the engine', () => {
  beforeEach(() => { cockpit.toolSteps = [] })

  it('reads success, the failure reason, and line counts off the event', () => {
    applyToolEvent({ action: 'call', name: 'write', subject: 'internal/skill/edit.go' })
    expect(cockpit.toolSteps[0]).toMatchObject({ label: 'write internal/skill/edit.go', state: 'run' })

    applyToolEvent({ action: 'result', name: 'write', ok: true, added: 9, removed: 0 })
    expect(cockpit.toolSteps[0]).toMatchObject({ state: 'done', added: 9 })
    expect(cockpit.toolSteps[0].removed).toBeUndefined() // 0 shows nothing

    applyToolEvent({ action: 'call', name: 'web_fetch', subject: 'https://openai.com/codex' })
    applyToolEvent({ action: 'result', name: 'web_fetch', ok: false, error: 'HTTP 403' })
    expect(cockpit.toolSteps[1]).toMatchObject({ state: 'err', error: 'HTTP 403' })
  })

  // The engine announces a call twice: once while its arguments stream, once
  // when it runs. One row, one clock.
  it('keeps one row when the streamed call is announced again on execution', () => {
    applyToolEvent({ action: 'call', name: 'write', subject: 'landing.html', added: 1 })
    applyToolEvent({ action: 'call', name: 'write', subject: 'landing.html', added: 40 })
    applyToolEvent({ action: 'call', name: 'write', subject: 'landing.html' })
    expect(cockpit.toolSteps).toHaveLength(1)
    expect(cockpit.toolSteps[0].added).toBe(40) // the counter climbs in place

    // ...but a genuine second call to the same file, after the first finished,
    // is its own row.
    applyToolEvent({ action: 'result', name: 'write', ok: true, added: 40, removed: 0 })
    applyToolEvent({ action: 'call', name: 'write', subject: 'landing.html' })
    expect(cockpit.toolSteps).toHaveLength(2)
  })

  it('drops the label to the bare tool name when a call has no subject', () => {
    applyToolEvent({ action: 'call', name: 'todo_write' })
    expect(cockpit.toolSteps[0].label).toBe('todo_write')
  })

  // The join undone. All three facts were on the event the whole time and the
  // store was flattening them into one string, which is what made the row read
  // like a log line — and which threw `act` away entirely, so a `browser` row
  // could say the browser was busy and not one thing more.
  it('keeps the name, the act and the subject as their own fields', () => {
    applyToolEvent({ action: 'call', ref: 'c1', name: 'browser', act: 'click', subject: '#submit' })
    expect(cockpit.toolSteps[0]).toMatchObject({
      name: 'browser', act: 'click', subject: '#submit',
      // And the joined string as well: it is still a row's identity on an
      // engine that sends no call id, and still what a stored turn reads back.
      label: 'browser #submit',
    })
  })

  // A row born from the streaming announcement knows the tool and nothing else
  // — the action is inside arguments the model is still writing. Taking the
  // later answer is what keeps that row from staying on the pack's own verb.
  it('lets the act arrive after the row exists', () => {
    applyToolEvent({ action: 'call', ref: 'c2', name: 'change' })
    expect(cockpit.toolSteps[0].act).toBeUndefined()
    applyToolEvent({ action: 'call', ref: 'c2', name: 'change', act: 'delete', subject: 'old.go' })
    expect(cockpit.toolSteps).toHaveLength(1)
    expect(cockpit.toolSteps[0]).toMatchObject({ act: 'delete', subject: 'old.go' })
  })

  // ข้อ 02: the results existed inside the tool's text output, where no window
  // could read them, so a search row said how long it took and nothing about
  // what it found.
  it('takes a search\'s results off the result event', () => {
    applyToolEvent({ action: 'call', ref: 's1', name: 'web_search', subject: 'rsc' })
    applyToolEvent({
      action: 'result', ref: 's1', name: 'web_search', subject: 'rsc', ok: true, count: 2,
      links: [
        { title: 'RFC', url: 'https://react.dev/rfc' },
        { title: 'Notes', url: 'https://go.dev/blog' },
      ],
    })
    expect(cockpit.toolSteps[0].links).toHaveLength(2)
    expect(cockpit.toolSteps[0].links?.[0]).toMatchObject({ title: 'RFC' })
    // Nothing has been read in full yet.
    expect(cockpit.toolSteps[0].links?.[0].opened).toBeUndefined()

    // The fetch that earns the badge lands calls later, which is why the badge
    // cannot ride on the search's own event.
    applyToolEvent({ action: 'call', ref: 'f1', name: 'web_fetch', subject: 'https://react.dev/rfc' })
    applyToolEvent({ action: 'result', ref: 'f1', name: 'web_fetch', subject: 'https://react.dev/rfc', ok: true })
    expect(cockpit.toolSteps[0].links?.[0].opened).toBe(true)
    expect(cockpit.toolSteps[0].links?.[1].opened).toBeUndefined()
  })

  // Argument order is the model's choice: when a write's "content" streams
  // before its "path" the row appears unnamed and has to name itself later.
  // Keyed on the label, the arrival of the name drew a second row.
  it('names a row that started before its subject arrived, without splitting it', () => {
    applyToolEvent({ action: 'call', ref: 'call_1', name: 'write', added: 12 })
    expect(cockpit.toolSteps).toHaveLength(1)
    expect(cockpit.toolSteps[0]).toMatchObject({ label: 'write', added: 12 })

    applyToolEvent({ action: 'call', ref: 'call_1', name: 'write', added: 260 })
    applyToolEvent({ action: 'call', ref: 'call_1', name: 'write', subject: 'landing.html', added: 402 })
    expect(cockpit.toolSteps).toHaveLength(1)
    expect(cockpit.toolSteps[0]).toMatchObject({ label: 'write landing.html', added: 402 })

    applyToolEvent({ action: 'result', ref: 'call_1', name: 'write', subject: 'landing.html', ok: true, added: 402 })
    expect(cockpit.toolSteps).toHaveLength(1)
    expect(cockpit.toolSteps[0]).toMatchObject({ label: 'write landing.html', state: 'done' })
  })

  // §59: the model's narration between calls and a round's thinking duration
  // land as rows of their own kind — born finished, never counted as tools.
  it('keeps narration and thinking as their own closed rows', () => {
    applyToolEvent({ action: 'note', name: '', text: '  กำลังไล่หา config  ' })
    applyToolEvent({ action: 'thinking', name: '', secs: 12 })
    applyToolEvent({ action: 'note', name: '', text: '   ' }) // blank narration is noise
    expect(cockpit.toolSteps).toHaveLength(2)
    expect(cockpit.toolSteps[0]).toMatchObject({ kind: 'note', label: 'กำลังไล่หา config', state: 'done' })
    expect(cockpit.toolSteps[1]).toMatchObject({ kind: 'thinking', secs: 12, state: 'done' })
  })

  // An answer the user typed over arrives on the same channel as narration and
  // is not the same thing: it is a finished reply, and it kept its own kind so
  // the chat can draw it as one.
  it('keeps an answer an interjection re-placed as prose, not as a narration row', () => {
    applyToolEvent({ action: 'said', name: '', text: '  ## สรุป\n\n- ข้อหนึ่ง  ' })
    applyToolEvent({ action: 'said', name: '', text: '  ' }) // nothing said is nothing to keep
    expect(cockpit.toolSteps).toHaveLength(1)
    expect(cockpit.toolSteps[0]).toMatchObject({
      kind: 'said', label: '## สรุป\n\n- ข้อหนึ่ง', state: 'done',
    })
  })

  // Two writes in flight at once are two rows, even though both start unnamed
  // and share a tool name.
  it('keeps concurrent calls apart by ref', () => {
    applyToolEvent({ action: 'call', ref: 'call_1', name: 'write', added: 3 })
    applyToolEvent({ action: 'call', ref: 'call_2', name: 'write', added: 5 })
    expect(cockpit.toolSteps).toHaveLength(2)

    applyToolEvent({ action: 'result', ref: 'call_2', name: 'write', subject: 'b.html', ok: true })
    expect(cockpit.toolSteps[1]).toMatchObject({ label: 'write b.html', state: 'done' })
    expect(cockpit.toolSteps[0].state).toBe('run')
  })

  // A sub-agent outlives the turn that started it, so its steps keep arriving
  // after the live block is gone. toolSteps is cleared at both ends of a turn,
  // so without a second home those rows land in the NEXT turn's timeline and
  // are drawn as work the user's new question caused.
  describe('a delegate still working after its turn ended', () => {
    beforeEach(() => { cockpit.backgroundSteps = [] })

    it('keeps its steps out of the next turn and in the background list', () => {
      // Turn 1: the delegation opens and its first step lands in the timeline.
      applyToolEvent({ action: 'call', ref: 'task_call', name: 'task', agent: 'explore' })
      applyToolEvent({ action: 'call', ref: 'g1', parent: 'task_call', name: 'grep', subject: 'Resolve' })
      expect(cockpit.toolSteps).toHaveLength(2)
      expect(cockpit.backgroundSteps).toHaveLength(0)

      // The turn ends: the answer went out, the live lists are cleared.
      cockpit.toolSteps = []

      // The delegate is still working, and says so. The result for g1 closes
      // nothing — that row went into turn 1's snapshot and is not here to
      // close — but the new call has somewhere to go that is not turn 2.
      applyToolEvent({ action: 'result', ref: 'g1', parent: 'task_call', name: 'grep', ok: true })
      applyToolEvent({ action: 'call', ref: 'r1', parent: 'task_call', name: 'read', subject: 'turn/executor.go' })
      expect(cockpit.toolSteps).toHaveLength(0)
      expect(cockpit.backgroundSteps).toHaveLength(1)
      expect(cockpit.backgroundSteps[0]).toMatchObject({ label: 'read turn/executor.go', state: 'run' })
    })

    it('still puts a delegate started in this turn into this turn', () => {
      applyToolEvent({ action: 'call', ref: 'task_call', name: 'task', agent: 'doc' })
      applyToolEvent({ action: 'call', ref: 'r1', parent: 'task_call', name: 'read', subject: 'README.md' })
      expect(cockpit.backgroundSteps).toHaveLength(0)
      expect(cockpit.toolSteps).toHaveLength(2)
    })

    // The tray's rows come from the engine's register, and the first background
    // event is what turns the poll on — without this a reloaded window would
    // never learn that anything is running.
    it('starts the tray poll when the first background event arrives', async () => {
      vi.mocked(BackgroundTasks).mockResolvedValue([
        { id: 'task_9', agent: 'explore', label: 'x', startedAt: new Date().toISOString(), toolCalls: 3, state: 'running', collected: false },
      ] as any)
      applyToolEvent({ action: 'call', ref: 'r9', parent: 'ghost_task', name: 'read', subject: 'a.go' })
      await vi.waitFor(() => expect(cockpit.backgroundTasks).toHaveLength(1))
      expect(cockpit.backgroundTasks[0]).toMatchObject({ id: 'task_9', state: 'running' })
    })
  })
})

describe('tool timeline collapsing', () => {
  it('collapses a finished turn behind a count, next to the thinking toggle', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54', reasoning: 'hmm', thinkSecs: 34,
        steps: [step('web_fetch', 'done'), step('web_fetch', 'err'), step('read', 'done')],
      }] as any,
    })

    const toggles = container.querySelectorAll('.meta-row .reasoning-toggle')
    expect(toggles.length).toBe(2)
    expect(toggles[0].textContent).toContain('Thought for 34s')
    expect(toggles[1].textContent).toContain('Used 3 tools')
    expect(toggles[1].textContent).toContain('1 failed') // failures stay visible while collapsed
    expect(container.querySelector('.tool-step')).toBeNull()

    await fireEvent.click(toggles[1])
    expect(container.querySelectorAll('.tool-step').length).toBe(3)
  })

  // The live panel is not the finished one above: it is open from the first
  // reasoning token (livePanel starts on 'think') and it is what a thinking
  // model writes into for most of a turn. It draws through the pacer now, which
  // means an action writes its text rather than Svelte — and an action that
  // failed to attach would leave the panel blank instead of throwing.
  it('shows live reasoning the moment there is any, in full', () => {
    const { container } = render(Chat, {
      ...baseProps,
      awaitingReply: true,
      reasoningText: 'ผู้ใช้ถามเรื่องสตรีม ต้องไปดูว่าวาดตรงไหน',
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
    })
    // First sight is never typed out — whatever had already streamed when this
    // block appeared is on screen at once.
    expect(container.querySelector('.reasoning-body')?.textContent)
      .toBe('ผู้ใช้ถามเรื่องสตรีม ต้องไปดูว่าวาดตรงไหน')
  })

  // The two panels swap, they never stack: one slot, one open at a time.
  it('opening the tool list closes the thinking it replaces', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54', reasoning: 'hmm', thinkSecs: 34,
        steps: [step('web_fetch', 'done')],
      }] as any,
    })
    const [think, tools] = container.querySelectorAll('.meta-row .reasoning-toggle')

    await fireEvent.click(think)
    expect(container.querySelector('.reasoning-body')).toBeTruthy()

    await fireEvent.click(tools)
    expect(container.querySelector('.reasoning-body')).toBeNull()
    expect(container.querySelectorAll('.tool-step').length).toBe(1)

    await fireEvent.click(tools) // clicking the open one closes it again
    expect(container.querySelector('.tool-step')).toBeNull()
  })

  // Every sentence stays on screen while the turn runs, not only the latest.
  // Keeping one line was the old compromise (§59) and it cost the rest: prose
  // the reader was already reading left the screen the moment the model said
  // its next thing, and a table or a chart written mid-turn went with it.
  it('keeps every sentence of the turn on screen, each with its own work', () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [
        { kind: 'note', label: 'first thought', state: 'done', startedAt: 0 },
        { kind: 'note', label: 'scanning the loop', state: 'done', startedAt: 0 },
        step('browser_read', 'run'),
      ] as any,
    })
    const said = [...container.querySelectorAll('.phase-say')].map((el) => el.textContent?.trim())
    expect(said).toEqual(['first thought', 'scanning the loop'])
    // The running row belongs to the sentence that announced it, not to a
    // separate list at the bottom of the bubble.
    const phases = container.querySelectorAll('.phase')
    // The row says what the agent is DOING — the tool's own name is not on it
    // any more (lib/toolFace.ts). Which row it is, is still the point here.
    expect(phases[1].querySelector('.tool-step')?.textContent).toContain('Read page')
    expect(phases[0].querySelector('.tool-step')).toBeNull()
  })

  // In the finished timeline the narration and thinking rows render in place,
  // but the "used N tools" count stays a count of tools.
  it('renders narration and thinking inside the finished timeline, uncounted', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        steps: [
          { kind: 'thinking', label: '', secs: 8, state: 'done', startedAt: 0 },
          { kind: 'note', label: 'reading the loop first', state: 'done', startedAt: 0 },
          step('read', 'done'),
        ],
      }] as any,
    })
    const toggle = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
      .find((b) => b.textContent?.includes('Used 1'))
    expect(toggle).toBeTruthy()
    await fireEvent.click(toggle!)
    expect(container.querySelector('.tool-note')?.textContent).toBe('reading the loop first')
    expect(container.querySelector('.tool-think')?.textContent).toContain('8')
  })

  // Owner, 16 ส.ค.: interjecting mid-turn looked like it wrecked everything the
  // agent had already answered. It was drawn as a note — raw source at --fs-xs
  // in --text-muted — and then filed behind the tools toggle. It is prose, so it
  // is drawn as prose, in the bubble, both while the turn runs and after.
  it('draws an answer the user typed over as markdown in the live block', () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [
        { kind: 'said', label: '## สรุป\n\nเรียบร้อยครับ', state: 'done', startedAt: 0 },
        step('grep', 'run'),
      ] as any,
    })
    const said = container.querySelector('.phase-say')
    expect(said?.querySelector('h2')?.textContent).toBe('สรุป')
    // Not as a narration row, and never inside the tool timeline. Every phase
    // draws its prose this way now, so the answer needs no special case to
    // keep its markdown — which is what the Demoted flag was working around.
    expect(container.querySelector('.tool-note')).toBeNull()
    expect(said?.closest('.tool-step')).toBeNull()
  })

  it('keeps that answer in the bubble after the turn ends, uncounted as a tool', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'เช็คแล้วครับ', time: '10:54',
        steps: [
          { kind: 'said', label: '## สรุป\n\nเรียบร้อยครับ', state: 'done', startedAt: 0 },
          step('grep', 'done'),
        ],
      }] as any,
    })
    expect(container.querySelector('.said-block h2')?.textContent).toBe('สรุป')
    // The answer that ended the turn is still the body below it.
    expect(container.textContent).toContain('เช็คแล้วครับ')

    const toggle = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
      .find((b) => b.textContent?.includes('Used 1'))
    expect(toggle).toBeTruthy() // one grep, not two "tools"
    await fireEvent.click(toggle!)
    // Opening the tools panel shows the grep and nothing else — an answer is
    // not a step, and a paragraph in that list is a paragraph pretending to be
    // one.
    expect(container.querySelectorAll('.tool-step')).toHaveLength(1)
    expect(container.querySelectorAll('.said-block')).toHaveLength(1)
  })

  // The button and the number on it have to be asked in the same terms. They
  // were not: the button appeared for any own step, the count excluded
  // narration and thinking, and a turn that only narrated offered "Used 0
  // tools" — a control whose whole promise is that there is nothing behind it.
  it('offers no tool toggle for a turn that only narrated', () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        steps: [{ kind: 'note', label: 'looking at the sidebar first', state: 'done', startedAt: 0 }],
      }] as any,
    })

    const toggles = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
    expect(toggles.some((b) => b.textContent?.includes('Used 0'))).toBe(false)
    // And no empty bar left standing where the toggle would have been.
    expect(container.querySelector('.meta-row')).toBeNull()
  })

  it('offers no tool toggle mid-turn either, on the same narration', () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [{ kind: 'note', label: 'looking at the sidebar first', state: 'done', startedAt: 0 }] as any,
    })

    const toggles = [...container.querySelectorAll('.reasoning-toggle')]
    expect(toggles.some((b) => b.textContent?.includes('Used 0'))).toBe(false)
  })

  // Nothing folds while the turn runs. The rule used to be the other half of
  // this — only what was still going stayed out — which took a call off the
  // screen in the frame its result landed: a skill that answered in half a
  // second was never readable at all (owner, 7 ก.ย.). The cap that folding used
  // to provide is the scrolling window on the box now.
  it('keeps every call on screen while the turn is running', () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [step('web_fetch', 'done'), step('web_fetch', 'done'), step('browser_read', 'run')],
    })

    expect(container.querySelectorAll('.tool-step').length).toBe(3)
    // The count is still said — as a line, not as a control. With nothing
    // hidden there is nothing a fold could promise.
    expect(container.querySelector('button.phase-head')).toBeNull()
    expect(container.querySelector('.phase-head')?.textContent).toContain('Used 3 tools')
    expect(container.querySelector('.tool-box.live-window')).toBeTruthy()
  })

  // And the other half, unchanged: a turn that is over is a record, and a
  // record is read by opening it.
  it('folds the whole list behind the count once the turn is over', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        parts: [{ kind: 'text', text: 'done' }],
        steps: [step('web_fetch', 'done'), step('web_fetch', 'done'), step('browser_read', 'done')],
      }] as any,
    })

    expect(container.querySelector('.tool-step')).toBeNull()
    // The header is the control, and it is the only one: no summary row above
    // it saying the same thing a second time.
    const toggles = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
    expect(toggles.some((b) => b.textContent?.includes('Used'))).toBe(false)

    const head = container.querySelector('button.phase-head')
    expect(head?.textContent).toContain('Used 3 tools')
    await fireEvent.click(head!)
    expect(container.querySelectorAll('.tool-step').length).toBe(3)
    // A record is not windowed: the cap is about a thing in motion.
    expect(container.querySelector('.tool-box.live-window')).toBeNull()
  })

  // The last call coming back is the signal — not the next sentence, and not
  // the end of the turn (owner, 7 ก.ย.: "พอมันรัน tool เสร็จ ถึงตัวสุดท้าย ก่อน
  // จะพูดประโยคถัดไป มันก็พับลงอย่างนุ่มนวล"). The model has said nothing yet
  // here and the turn is still running.
  it('folds a stretch of work when its last call comes back, mid-turn', async () => {
    const working = [
      { label: 'read a.go', ref: 'call_a', state: 'done', startedAt: 0, secs: 1 },
      { label: 'read b.go', ref: 'call_b', state: 'run', startedAt: 0 },
    ]
    const { container, rerender } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: working as any,
    })
    // One list, no control, while a call is still out.
    expect(container.querySelector('button.phase-head')).toBeNull()

    // The last one comes back. The model has said nothing yet and the turn is
    // still running: this alone is the signal.
    await rerender({
      toolSteps: [working[0], { ...working[1], state: 'done', secs: 1 }],
    } as any)

    // Nothing vanished in the frame the result landed: the rows are still
    // there, still open, and the header has become the control.
    const head = () => container.querySelector('button.phase-head')
    expect(head()).toBeTruthy()
    expect(head()!.getAttribute('aria-expanded')).toBe('true')
    expect(container.querySelectorAll('.tool-step').length).toBe(2)

    await new Promise((r) => setTimeout(r, 700))
    await tick()
    expect(head()!.getAttribute('aria-expanded')).toBe('false')
  })

  // The half that must never move: a stretch with a call still out is one list,
  // whole, capped and scrolling, with a line over it rather than a control.
  it('never folds a stretch that still has a call out', async () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [
        { label: 'read a.go', ref: 'call_a', state: 'done', startedAt: 0, secs: 1 },
        { label: 'read b.go', ref: 'call_b', state: 'done', startedAt: 0, secs: 1 },
        { label: 'run go test', ref: 'call_c', state: 'run', startedAt: 0 },
      ] as any,
    })

    await new Promise((r) => setTimeout(r, 700))
    await tick()
    expect(container.querySelectorAll('.tool-step').length).toBe(3)
    expect(container.querySelector('button.phase-head')).toBeNull()
    // The cap he compared to the thinking panel, still on.
    expect(container.querySelector('.tool-box.live-window')).toBeTruthy()
  })

  // The view unmounts Chat whole, so opening a file mid-turn and coming back
  // rebuilds every fold map from nothing. Without a guard the effect met a
  // stretch that had folded ten seconds earlier, found no record of it, and
  // played the whole movement again — every single time (owner: "ไปหน้าอื่นแล้ว
  // กลับมามันจะพับให้ดูอีกรอบ ผมลองแล้วเป็นซ้ำๆ"). The app folds what it watched
  // arrive, and nothing else.
  it('does not replay the fold for work that ended before it was mounted', async () => {
    const done = [
      { label: 'read a.go', ref: 'call_a', state: 'done', startedAt: 0, secs: 1 },
      { label: 'read b.go', ref: 'call_b', state: 'done', startedAt: 0, secs: 1 },
    ]
    // A remount over a turn already in flight: the rows arrive finished.
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: done as any,
    })
    await tick()

    // Closed and still from the first frame — nothing to hand over, so nothing
    // is opened in order to be shut again.
    const head = container.querySelector('button.phase-head')
    expect(head?.getAttribute('aria-expanded')).toBe('false')
    expect(container.querySelector('.phase-fold')).toBeNull()

    await new Promise((r) => setTimeout(r, 700))
    await tick()
    expect(head?.getAttribute('aria-expanded')).toBe('false')
    expect(container.querySelector('.phase-fold')).toBeNull()
  })

  // The other half of that guard: work still in flight on the first frame has
  // its ending ahead of it, so it is adopted the ordinary way and folds.
  it('still folds work that was in flight when it was mounted', async () => {
    const { container, rerender } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [{ label: 'read a.go', ref: 'call_a', state: 'run', startedAt: 0 }] as any,
    })
    await tick()
    expect(container.querySelector('button.phase-head')).toBeNull()

    await rerender({
      toolSteps: [{ label: 'read a.go', ref: 'call_a', state: 'done', startedAt: 0, secs: 1 }],
    } as any)
    const head = () => container.querySelector('button.phase-head')
    expect(head()?.getAttribute('aria-expanded')).toBe('true')

    await new Promise((r) => setTimeout(r, 700))
    await tick()
    expect(head()?.getAttribute('aria-expanded')).toBe('false')
  })

  // A sentence is what opens a phase, so two rounds of tools can arrive under
  // one. The beat has to be spent asking again, not on the answer it had when
  // it was armed.
  it('calls off the fold when the same stretch starts working again', async () => {
    const quiet = [{ label: 'read a.go', ref: 'call_a', state: 'done', startedAt: 0, secs: 1 }]
    const { container, rerender } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: quiet as any,
    })
    await rerender({
      toolSteps: [...quiet, { label: 'read b.go', ref: 'call_b', state: 'run', startedAt: 0 }],
    } as any)

    await new Promise((r) => setTimeout(r, 700))
    await tick()
    expect(container.querySelectorAll('.tool-step').length).toBe(2)
    expect(container.querySelector('button.phase-head')).toBeNull()
  })

  // The frame the reply lands used to take the rows with it. They are drawn
  // outside the fold while the turn runs -- nothing folds mid-turn -- and the
  // fold they were handed to started closed, so three rows became a count in
  // one frame with nothing saying where they had gone (owner, 7 ก.ย.: "ตอนมัน
  // รัน tool เสร็จ มันควรจะพับเก็บอย่างสุภาพ ... ตอนนี้มันแบบ พึ๊บไปเลย").
  //
  // So the record inherits the state the live block was in, at the live
  // window's own height so the handover moves nothing, and shuts itself a beat
  // later.
  it('hands the running rows to the fold open, then folds them shut', async () => {
    const steps = [
      { label: 'read a.go', ref: 'call_a', state: 'done', startedAt: 0, secs: 1 },
      { label: 'read b.go', ref: 'call_b', state: 'done', startedAt: 0, secs: 1 },
    ]
    // Watched arriving, because that is the only work the app folds.
    const { container, rerender } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [steps[0], { ...steps[1], state: 'run', secs: undefined }] as any,
    })
    await rerender({ toolSteps: steps } as any)
    expect(container.querySelectorAll('.tool-step').length).toBe(2)

    // The turn ends: the live block goes and the finished bubble takes its
    // place, drawing the same phase.
    await rerender({
      awaitingReply: false,
      toolSteps: [],
      messages: [
        { role: 'user', text: 'go', time: '10:54' },
        { role: 'agent', text: 'done', time: '10:55', parts: [{ kind: 'text', text: 'done' }], steps },
      ],
    } as any)

    const head = () => container.querySelector('button.phase-head')
    expect(head()?.getAttribute('aria-expanded')).toBe('true')
    expect(container.querySelectorAll('.tool-step').length).toBe(2)
    // Still at the height it had a frame ago, so the swap moves nothing.
    expect(container.querySelector('.tool-box.live-window')).toBeTruthy()

    await new Promise((r) => setTimeout(r, 700))
    await tick()
    expect(head()?.getAttribute('aria-expanded')).toBe('false')
  })

  // The beat is the reader's to interrupt. Opening a phase inside it used to be
  // undone a moment later by a timer armed before the click.
  it('leaves a phase alone once the reader has touched it', async () => {
    const steps = [{ label: 'read a.go', ref: 'call_a', state: 'done', startedAt: 0, secs: 1 }]
    const { container, rerender } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [{ label: 'read a.go', ref: 'call_a', state: 'run', startedAt: 0 }] as any,
    })
    await rerender({ toolSteps: steps } as any)
    await rerender({
      awaitingReply: false,
      toolSteps: [],
      messages: [
        { role: 'user', text: 'go', time: '10:54' },
        { role: 'agent', text: 'done', time: '10:55', parts: [{ kind: 'text', text: 'done' }], steps },
      ],
    } as any)

    // Shut it by hand, then open it again -- both inside the beat.
    const head = container.querySelector('button.phase-head')!
    await fireEvent.click(head)
    await fireEvent.click(head)
    expect(head.getAttribute('aria-expanded')).toBe('true')

    await new Promise((r) => setTimeout(r, 700))
    await tick()
    expect(head.getAttribute('aria-expanded')).toBe('true')
    // And it is the full list now, not the running turn's window.
    expect(container.querySelector('.tool-box.live-window')).toBeNull()
  })

  // The fold is keyed on the engine's own call id rather than on where the
  // phase happens to be drawn. Keyed by position it belonged to a slot instead
  // of to a phase, so the live block and the finished bubble were two different
  // keys for one stretch of work — everything the reader opened during a turn
  // shut itself the instant the reply landed.
  it('keeps a fold with its own phase when the transcript shifts', async () => {
    const agentMsg = (text: string, ref: string) => ({
      role: 'agent', text, time: '10:54',
      parts: [{ kind: 'text', text }],
      steps: [{ label: 'read a.go', ref, state: 'done', startedAt: 0, secs: 1 }],
    })
    const { container, rerender } = render(Chat, {
      ...baseProps,
      messages: [agentMsg('first', 'call_a')] as any,
    })
    await fireEvent.click(container.querySelector('button.phase-head')!)
    expect(container.querySelectorAll('.tool-step').length).toBe(1)

    // A newer turn arrives in front of it: index 0 is somebody else's phase now.
    await rerender({ messages: [agentMsg('newer', 'call_b'), agentMsg('first', 'call_a')] } as any)
    const heads = [...container.querySelectorAll('button.phase-head')]
    expect(heads.map((h) => h.getAttribute('aria-expanded'))).toEqual(['false', 'true'])
  })

  // Each number next to the thing it is about. The first version put both on
  // one line above the phase, which left the count and a four-paragraph plan
  // between the reader and the two rows being counted: by the time the eye
  // reached them the summary had scrolled away (owner, 29 ส.ค., off his own
  // screen). Thinking happens before the sentence, work after it.
  it('puts the thinking above the sentence and the work below it', () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [
        { kind: 'thinking', label: '', secs: 8, state: 'done', startedAt: 0 },
        { kind: 'note', label: 'ขั้นที่ 1 เก็บข้อมูลดิบก่อน', state: 'done', startedAt: 0 },
        step('shell', 'run'),
      ] as any,
    })

    const phase = container.querySelector('.phase')!
    const order = [...phase.children]
      .map((el) => [...el.classList].find((c) => c.startsWith('phase-')))
      .filter(Boolean)
    expect(order).toEqual(['phase-think', 'phase-say', 'phase-head'])
    expect(phase.querySelector('.phase-think')?.textContent).toContain('8')
    // And the thinking is a line, never a control: what it would open is the
    // reasoning text, which is one blob for the whole turn and belongs to no
    // single stretch. The work line under the prose is a line too for as long
    // as the stretch has a call out — which is why one is still running here.
    expect(phase.querySelectorAll('button').length).toBe(0)
  })

  // The half a person can lose work over. A delegate runs for minutes while the
  // agent narrates on, and nothing may take it off the screen — not even now
  // that the agent's OWN rows fold as soon as its last call comes back. The two
  // are different things: a tool row is a receipt and folds into a count, a
  // delegation is somebody else's work with a face on it, and "ซับเอเจน 1 ตัว"
  // cannot stand in for that.
  it('never folds a delegation that is still working', () => {
    cockpit.backgroundTasks = []
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [
        step('read a.go', 'done'),
        { label: 'task start', ref: 'call_9', delegation: true, agent: 'ผู้ช่วยค้นหา', state: 'run', startedAt: 0 },
      ] as any,
    })

    expect(container.querySelector('.bgw-card')).toBeTruthy()
    // The agent's own finished read may fold; the card may not go in with it.
    expect(container.querySelector('.phase-fold .bgw-card')).toBeNull()
  })

  // And the one the owner actually reported: the moment a delegate's report is
  // the thing to read was the moment its card left the screen for the collapsed
  // count above it (7 ก.ย.: "แล้วซับเอเจน จะไม่พับเองหลังจากทำงานเสร็จ"). It
  // stays for the rest of the turn now, and past the beat that folds the
  // agent's own rows away.
  it('keeps a delegation that has finished on screen for the rest of the turn', () => {
    cockpit.backgroundTasks = [{
      id: 'task_7', agent: 'ผู้ช่วยค้นหา', label: 'หาไฟล์', startedAt: new Date().toISOString(),
      toolCalls: 2, tokens: 0, tokensIn: 0, tokensOut: 0, cachedIn: 0, cacheReported: false,
      state: 'done', elapsedMs: 4_000, collected: true,
    }] as any
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [
        { label: 'task start', ref: 'call_7', task: 'task_7', delegation: true, agent: 'ผู้ช่วยค้นหา', state: 'done', startedAt: 0 },
      ] as any,
    })

    expect(container.querySelector('.bgw-card')).toBeTruthy()
    expect(container.querySelector('button.phase-head')).toBeNull()
  })
})

// The block that marks the live call. It exists twice on purpose — a bar that
// travels between rows, and a per-row fallback in CSS — and the whole safety of
// that arrangement is that exactly one of them is ever showing. `glide-on` is
// the switch, so it is what these check.
describe('the block on the live tool row', () => {
  beforeEach(() => { cockpit.toolSteps = [] })

  it('hands the live row to the travelling bar when one call is running', async () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [step('browser_read', 'run')],
    })
    await tick()

    const list = container.querySelector('.tool-steps')
    expect(list?.querySelector('.tool-glide')).not.toBeNull()
    expect(list?.classList.contains('glide-on')).toBe(true)
  })

  // Providers do issue parallel calls, and one bar cannot be in two places. The
  // per-row block has to take over, which it only does with the class off.
  it('falls back to per-row blocks when two calls run at once', async () => {
    const { container } = render(Chat, {
      ...baseProps, awaitingReply: true,
      messages: [{ role: 'user', text: 'go', time: '10:54' }] as any,
      toolSteps: [step('browser_read', 'run'), step('web_fetch', 'run')],
    })
    await tick()

    expect(container.querySelector('.tool-steps')?.classList.contains('glide-on')).toBe(false)
  })
})

// The approval mode is switched from the model menu. Shift+Tab is the
// shortcut, and it must never be able to REACH full-access: that mode never
// prompts again, so turning it on stays a deliberate pick from the menu.
describe('approval mode on the composer', () => {
  // The mode had a glyph on the chip until the model name moved back beside it
  // and three marks at 14px stopped reading as three facts (owner's call). It
  // is named in words in the menu, which is where it is changed — so the chip
  // must not grow one back without that decision being revisited.
  it('keeps the approval glyph off the chip, whatever the mode', () => {
    for (const approval of ['ask', 'unsafe-only', 'full-access']) {
      const { container } = render(Chat, {
        ...baseProps, messages: [] as any,
        model: { ...baseProps.model, approval },
      })
      expect(container.querySelector('.model-chip .mode-ic')).toBeNull()
    }
  })

  // What the chip does carry: which provider is answering, and which of its
  // models. The name was off for a while on width grounds and is back by the
  // owner's call — reading it should not cost a menu. Width is handled by
  // truncation, not by omission (see the vendor-prefix test below).
  it('carries the provider as a mark and names the model', () => {
    const { container } = render(Chat, { ...baseProps, messages: [] as any })
    const chip = container.querySelector('.model-chip')!
    expect(chip.querySelector('.pv svg, .pv .pv-letter')).toBeTruthy()
    expect(chip.textContent).toContain('v4')
    // The full id stays in the title, which is what the truncated name points at.
    expect(chip.getAttribute('title')).toBe('v4')
  })

  // OpenRouter and Together spell their ids `vendor/model`. The mark beside the
  // text already says who made it, so repeating the vendor spends width on the
  // half of the string that is never the answer to "which model is this".
  it('drops the vendor prefix from the name but keeps it in the title', () => {
    const { container } = render(Chat, {
      ...baseProps, messages: [] as any,
      model: { ...baseProps.model, modelName: 'deepseek/deepseek-r1' },
    })
    const chip = container.querySelector('.model-chip')!
    expect(chip.querySelector('.t')!.textContent).toBe('deepseek-r1')
    expect(chip.getAttribute('title')).toBe('deepseek/deepseek-r1')
  })

  // The menu below only draws a think-level row when there are two levels to
  // pick between, so keying the badge off model.thinkLevel alone advertised a
  // setting the menu then offered no way to move.
  it('badges the think level only when the menu can actually change it', async () => {
    SupportedThinkLevels.mockResolvedValueOnce(['high'])
    const one = render(Chat, { ...baseProps, messages: [] as any }).container
    await tick(); await tick()
    expect(one.querySelector('.model-chip .lvl')).toBeNull()

    SupportedThinkLevels.mockResolvedValueOnce(['low', 'high'])
    const two = render(Chat, { ...baseProps, messages: [] as any }).container
    await tick(); await tick()
    expect(two.querySelector('.model-chip .lvl')?.textContent).toContain('high')
  })

  it('shift+tab toggles ask↔unsafe-only and only ever tightens full-access', async () => {
    SwitchApprovalMode.mockClear()
    render(Chat, { ...baseProps, messages: [] as any })
    await fireEvent.keyDown(window, { key: 'Tab', shiftKey: true })
    expect(SwitchApprovalMode).toHaveBeenCalledWith('unsafe-only')

    SwitchApprovalMode.mockClear()
    render(Chat, {
      ...baseProps, messages: [] as any,
      model: { ...baseProps.model, approval: 'full-access' },
    })
    await fireEvent.keyDown(window, { key: 'Tab', shiftKey: true })
    expect(SwitchApprovalMode).toHaveBeenCalledWith('ask')
    expect(SwitchApprovalMode).not.toHaveBeenCalledWith('full-access')
  })
})

// A sub-agent's tool calls arrive on the same channel as the main agent's, told
// apart only by ToolEvent.parent (§44.5). The timeline has to show whose work is
// whose, and two delegates running the same tool must not share a row.
describe('sub-agent tool events', () => {
  beforeEach(() => { cockpit.toolSteps = [] })

  it('keeps a delegate’s row separate from an identical call by the main agent', () => {
    // The `task` row first, as the engine sends it: a delegate's parent is the
    // ref of the call that spawned it (executor.go stamps both from call.ID),
    // and it is that row's presence in this turn that says the work is this
    // turn's — see listFor.
    applyToolEvent({ action: 'call', name: 'task', ref: 'task_1', agent: 'explore' })
    applyToolEvent({ action: 'call', name: 'grep', subject: 'needle', ref: 'main_1' })
    applyToolEvent({ action: 'call', name: 'grep', subject: 'needle', ref: 'sub_1', parent: 'task_1' })
    expect(cockpit.toolSteps.length).toBe(3)
    expect(cockpit.toolSteps[1].parent).toBeUndefined()
    expect(cockpit.toolSteps[2].parent).toBe('task_1')

    // Each result lands on its own row.
    applyToolEvent({ action: 'result', name: 'grep', subject: 'needle', ref: 'sub_1', parent: 'task_1', ok: true })
    expect(cockpit.toolSteps[2].state).toBe('done')
    expect(cockpit.toolSteps[1].state).toBe('run')
  })

  it('carries the sub-agent name and brief onto the task row', () => {
    applyToolEvent({
      action: 'call', name: 'task', subject: 'find every caller', ref: 'task_1',
      agent: 'explore', brief: 'search internal/ for callers of Resolve and list the paths',
    })
    expect(cockpit.toolSteps[0].agent).toBe('explore')
    expect(cockpit.toolSteps[0].brief).toContain('callers of Resolve')
  })

  // The row is usually born from the streaming progress announcement, fired
  // while the model is still writing the call's arguments — no agent, no
  // brief, no kind yet. The executor's own call event arrives after, carrying
  // all three, and must land on the SAME row. Dropping them was a live turn
  // reading "ซับเอเจน 1 ตัว" for a job the doc agent did, while the
  // persisted parts said agent/doc — two answers from one turn.
  it('fills in the delegation facts when they arrive after the row was born', () => {
    applyToolEvent({ action: 'call', name: 'task', ref: 'task_1' })
    expect(cockpit.toolSteps[0].agent).toBeUndefined()

    applyToolEvent({
      action: 'call', name: 'task', subject: 'ทำสรุป 5 ข้อ', ref: 'task_1',
      agent: 'doc', agentKind: 'agent', brief: 'สร้างไฟล์เอกสาร .docx',
    })
    expect(cockpit.toolSteps.length).toBe(1)
    expect(cockpit.toolSteps[0].agent).toBe('doc')
    expect(cockpit.toolSteps[0].agentKind).toBe('agent')
    expect(cockpit.toolSteps[0].brief).toContain('.docx')
  })

  // The delegation block is the whole point: one titled group per sub-agent,
  // with its own steps inside it and the brief the main agent wrote. A flat list
  // of rows cannot say who did what.
  it('draws a delegation as its own named block with the delegate’s steps inside', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      // The live rows only exist during a turn, which is when a delegate runs.
      awaitingReply: true,
      // 'run' because a finished step collapses behind the toggle; the live rows
      // are the ones on screen while a delegate is actually working.
      toolSteps: [
        {
          label: 'task find every caller', ref: 'task_1', agent: 'explore',
          brief: 'search internal/ for callers of Resolve', state: 'run', startedAt: Date.now(),
        },
        { label: 'grep needle', parent: 'task_1', state: 'run', startedAt: Date.now() },
        { label: 'read hay.txt', parent: 'task_1', state: 'run', startedAt: Date.now() },
      ] as any,
      messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
    })

    const block = container.querySelector('.bgw-card')
    expect(block).toBeTruthy()
    // The rows are the default for a delegate that is working — no click. With
    // them on screen the headline names the JOB rather than repeating the newest
    // row three lines above itself, which is what the separate job line under it
    // used to be for and why it is dropped here.
    // Shut by default while it works: the newest row is already the card's
    // HEADLINE, so an open list would print it twice — and four delegates each
    // holding their list open is the wall this fold exists to stop.
    expect(block?.querySelector('.bgw-open')?.getAttribute('aria-expanded')).toBe('false')
    await fireEvent.click(block?.querySelector('.bgw-open') as HTMLElement)
    expect(block?.querySelector('.bgw-agent')?.textContent).toContain('explore')
    expect(block?.textContent).toContain('find every caller')
    expect(block?.querySelector('.bgw-brief')).toBeNull()
    expect(block?.querySelector('.bgw-longbrief')?.textContent).toContain('callers of Resolve')
    // The delegate's tools live inside the block, not in the agent's own list.
    expect(block?.querySelectorAll('.bgw-work .tool-step').length).toBe(2)
    // Not in the AGENT's own list — which now has to be said by excluding the
    // delegate's, because a sub-agent's turn is drawn with the same phase
    // blocks the transcript uses and therefore has a .tool-steps of its own.
    const ownRows = [...container.querySelectorAll('.tool-steps > .tool-step')]
      .filter((r) => !r.closest('.bgw-work'))
    expect(ownRows.length).toBe(0)
  })

  // Two delegates run at once and their events interleave on one channel — the
  // reason ToolEvent.parent exists at all. Each block must hold its own steps.
  it('keeps two concurrent delegations in separate blocks', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      awaitingReply: true,
      toolSteps: [
        { label: 'task hunt callers', ref: 't1', agent: 'explore', brief: 'brief one', state: 'run', startedAt: Date.now() },
        { label: 'task rename them', ref: 't2', agent: 'general', brief: 'brief two', state: 'run', startedAt: Date.now() },
        { label: 'grep alpha', parent: 't1', state: 'run', startedAt: Date.now() },
        { label: 'edit beta.go', parent: 't2', state: 'run', startedAt: Date.now() },
        { label: 'grep gamma', parent: 't1', state: 'run', startedAt: Date.now() },
      ] as any,
      messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
    })

    const blocks = container.querySelectorAll('.bgw-card')
    expect(blocks.length).toBe(2)
    for (const b of blocks) await fireEvent.click(b.querySelector('.bgw-open') as HTMLElement)
    expect(blocks[0].querySelector('.bgw-agent')?.textContent).toContain('explore')
    expect(blocks[1].querySelector('.bgw-agent')?.textContent).toContain('general')
    expect(blocks[0].querySelectorAll('.bgw-work .tool-step').length).toBe(2)
    expect(blocks[1].querySelectorAll('.bgw-work .tool-step').length).toBe(1)
    // No cross-contamination: the second delegate's edit is not in the first block.
    expect(blocks[0].textContent).not.toContain('edit beta.go')
    expect(blocks[1].textContent).not.toContain('grep alpha')
    expect(blocks[0].querySelector('.bgw-longbrief')?.textContent).toContain('brief one')
    expect(blocks[1].querySelector('.bgw-longbrief')?.textContent).toContain('brief two')
  })

  // One delegate hired, two cards on screen. The second was the agent's own
  // `task collect` — the call it makes to sit and wait for the delegate to
  // finish — and every action of delegation is packed under the one name, so
  // its label reads "task" exactly like the row that hired somebody. Judged by
  // the label it became a nameless second sub-agent standing next to the one it
  // was waiting on, and the count said two workers.
  it('does not draw a task collect as a second sub-agent', () => {
    const { container } = render(Chat, {
      ...baseProps,
      awaitingReply: true,
      toolSteps: [
        {
          label: 'task สำรวจฟีเจอร์และเส้นทางเว็บ', ref: 't1', agent: 'explore',
          brief: 'สำรวจโปรเจกต์แบบอ่านอย่างเดียว', delegation: true,
          state: 'run', startedAt: Date.now(),
        },
        { label: 'read frontend/index.html', parent: 't1', state: 'run', startedAt: Date.now() },
        // No agent, no brief, nothing running underneath it: the engine's `false`
        // is the only thing on a collect that says what it is.
        { label: 'task', ref: 't2', delegation: false, state: 'run', startedAt: Date.now() },
      ] as any,
      messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
    })

    const blocks = container.querySelectorAll('.bgw-card')
    expect(blocks.length).toBe(1)
    expect(blocks[0].querySelector('.bgw-agent')?.textContent).toContain('explore')
    // It is the agent's own work, and shows as the one row it is.
    const own = container.querySelectorAll('.tool-steps > .tool-step')
    expect(own.length).toBe(1)
    expect(own[0].textContent).toContain('Delegate')
  })

  // The flag is three-valued and the middle value is load-bearing: a row born
  // from the streaming announcement has no arguments to read an action off yet,
  // and neither has a turn stored before the engine said. Those keep the old
  // guess, which is right about every delegation and wrong only about the
  // actions that are not one — the correct way round for a fallback.
  it('falls back to the label only while nobody has said', () => {
    const node = (step: Record<string, unknown>, children: unknown[] = []) =>
      ({ step, children } as any)
    expect(isDelegation(node({ label: 'task', delegation: false }))).toBe(false)
    expect(isDelegation(node({ label: 'task', delegation: true }))).toBe(true)
    expect(isDelegation(node({ label: 'task' }))).toBe(true)
    expect(isDelegation(node({ label: 'task ทำสรุป', agent: 'doc' }))).toBe(true)
    expect(isDelegation(node({ label: 'read a.go' }))).toBe(false)
    // A delegate's own steps are the proof of a delegation nobody named.
    expect(isDelegation(node({ label: 'task' }, [{ label: 'grep x' }]))).toBe(true)
    // Narration is prose and may well open with the word.
    expect(isDelegation(node({ kind: 'note', label: 'task ต่อไปผมจะ…' }))).toBe(false)
  })

  // The `task` tool call finishes the instant the worker is spawned, so the row
  // it leaves behind says "done, 8s" for the whole job. The card is drawn from
  // the engine's register instead — the same source the tray reads.
  describe('a delegation card follows the register, not its task row', () => {
    const registered = (over: Record<string, unknown> = {}) => ({
      id: 'task_1', agent: 'deepresearch', label: 'ตรวจ CRM ไทย 20 เจ้า',
      startedAt: new Date(Date.now() - 92_000).toISOString(),
      toolCalls: 27, state: 'running', collected: false, ...over,
    })
    // The row exactly as the engine leaves it: closed, eight seconds, a tick.
    const spawnedRow = {
      label: 'task ตรวจ CRM ไทย 20 เจ้า', ref: 'call_1', task: 'task_1',
      agent: 'deepresearch', agentKind: 'agent', state: 'done', secs: 8, startedAt: Date.now() - 92_000,
    }

    it('keeps the clock running while the worker is still on its tools', () => {
      cockpit.backgroundTasks = [registered()] as any
      const { container } = render(Chat, {
        ...baseProps,
        awaitingReply: true,
        toolSteps: [spawnedRow, { label: 'web_fetch crm.co.th', parent: 'call_1', task: 'task_1', state: 'done', startedAt: 0, secs: 2 }] as any,
        messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
      })

      const card = container.querySelector('.bgw-card')
      expect(card?.className).toContain('run')
      expect(card?.querySelector('.bgw-badge.run')).toBeTruthy()
      // Counted off the delegation's own start, not frozen at the spawn's 8s.
      expect(card?.querySelector('.bgw-clock')?.textContent?.trim()).toBe('1m 32s')
      // And it stays where the user can watch it, instead of collapsing behind
      // the "used N agents" toggle the moment it starts.
      expect(card?.querySelector('.bgw-open')).toBeTruthy()
    })

    // Finished work is folded away wherever it sits: behind the phase header in
    // a turn that recorded its sequence, behind the "Agents" count in one
    // stored before parts existed. Only what is still running shows itself. So
    // both readings below open the fold first, whichever fold it is — the
    // card's state is the same question in both layouts.
    const openFinished = async (container: Element) => {
      const toggle = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
        .find((b) => b.textContent?.includes('Agents'))
      if (toggle) await fireEvent.click(toggle)
      const head = container.querySelector('button.phase-head[aria-expanded="false"]')
      if (head) await fireEvent.click(head)
      return container.querySelector('.tool-steps .bgw-card')
    }

    it('shows the delegation’s real total once the register says it finished', async () => {
      cockpit.backgroundTasks = [registered({ state: 'done', elapsedMs: 214_000, collected: true })] as any
      const { container } = render(Chat, {
        ...baseProps,
        awaitingReply: true,
        toolSteps: [spawnedRow] as any,
        messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
      })
      const card = await openFinished(container)
      expect(card?.querySelector('.bgw-clock')?.textContent?.trim()).toBe('3m 34s')
      expect(card?.querySelector('.bgw-badge.run')).toBeNull()
    })

    // A turn read back from the database has no register entry — the row is all
    // there is, and it still has to draw.
    it('falls back to the row when the register has never heard of the task', async () => {
      cockpit.backgroundTasks = []
      const { container } = render(Chat, {
        ...baseProps,
        awaitingReply: true,
        toolSteps: [spawnedRow] as any,
        messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
      })
      const card = await openFinished(container)
      expect(card?.querySelector('.bgw-clock')?.textContent?.trim()).toBe('8s')
    })

    // The other half of that fallback, and the half that was wrong. A row left
    // saying "run" is what a turn killed mid-flight leaves behind, and off a
    // live turn there is nobody left who could ever close it: the spinner ran
    // until the session was switched. Owner, 22 ส.ค., under a turn that had
    // just died of a dropped DeepSeek connection: "ทำไมมันค้างแบบนี้".
    it('stops spinning a row the register cannot vouch for, and says why', async () => {
      cockpit.backgroundTasks = []
      const { container } = render(Chat, {
        ...baseProps,
        awaitingReply: false,
        messages: [{
          role: 'agent', text: 'เทิร์นตายกลางคัน', time: '10:54',
          steps: [{ ...spawnedRow, state: 'run', secs: undefined }],
        }] as any,
      })
      const card = await openFinished(container)
      expect(card?.className).not.toContain('run')
      expect(card?.querySelector('.bgw-badge.run')).toBeNull()
      // No frozen clock either: the spawn's number was never the job's.
      // Absent rather than blank: the caption line draws the clock or it does
      // not, so "no number" is a missing element and not an empty one.
      expect(card?.querySelector('.bgw-clock')).toBeNull()
      // And the ✗ explains itself, rather than blaming a delegate that never
      // got the chance to fail.
      expect(card?.querySelector('.tool-err')?.textContent).toContain('turn ended')
    })

    // The exception that keeps the rule honest: inside a live turn the register
    // is a poll behind the row, and for that window the row is the better
    // answer. Closing the card there would blink a ✗ over work that is fine.
    it('still trusts a running row while its own turn is live', () => {
      cockpit.backgroundTasks = []
      const { container } = render(Chat, {
        ...baseProps,
        awaitingReply: true,
        toolSteps: [{ ...spawnedRow, state: 'run', secs: undefined }] as any,
        messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
      })
      const card = container.querySelector('.bgw-card')
      expect(card?.className).toContain('run')
      expect(card?.querySelector('.bgw-badge.run')).toBeTruthy()
    })

    // The id is only ever on the events from *inside* the delegate: the `task`
    // call completes before the register has a handle to give.
    it('takes the delegation id off the first step its worker runs', () => {
      cockpit.toolSteps = []
      applyToolEvent({ action: 'call', ref: 'call_1', name: 'task', agent: 'deepresearch' })
      expect(cockpit.toolSteps[0].task).toBeUndefined()
      applyToolEvent({ action: 'call', ref: 'r1', parent: 'call_1', task: 'task_1', name: 'web_fetch', subject: 'crm.co.th' })
      expect(cockpit.toolSteps[0].task).toBe('task_1')
    })

    // One delegation, one card. Reported as "แล้วจะแสดงซ้อนกันทำไมอ่ะ" with the
    // worker drawn once in the previous turn's message and once in the tray —
    // the case a filter over the live list alone cannot catch, because that list
    // was emptied when the collecting turn began.
    it('drops a tray row for a delegation the transcript is already drawing', async () => {
      cockpit.backgroundTasks = [registered()] as any
      const { container } = render(Chat, {
        ...baseProps,
        awaitingReply: true,
        toolSteps: [{ label: 'task_result task_1', state: 'run', startedAt: Date.now() }] as any,
        messages: [{
          role: 'agent', text: 'ส่งงานให้ซับเอเจนแล้ว', time: '10:54',
          steps: [spawnedRow],
        }] as any,
      })
      expect(container.querySelectorAll('.bgw .bgw-card').length).toBe(0)
      // Still reachable where the work was started, which is the whole reason
      // the tray row is redundant.
      const card = await openFinished(container)
      expect(card?.querySelector('.bgw-badge.run')).toBeTruthy()
    })

    // Unless it is stuck: the tray row is the only one with somewhere to type.
    it('keeps the tray row when the delegate is parked on a question', async () => {
      cockpit.backgroundTasks = [registered({ state: 'waiting', question: 'เอาไฟล์ไหน' })] as any
      const { container } = render(Chat, {
        ...baseProps,
        awaitingReply: false,
        messages: [{ role: 'agent', text: 'ส่งงานแล้ว', time: '10:54', steps: [spawnedRow] }] as any,
      })
      expect(container.querySelector('.bgw .bgw-answer')).toBeTruthy()
    })

    // The card can only ask the register if somebody fetched it. The poll used
    // to start on a *background* event or at the end of the turn, so a delegate
    // working inside its turn had nothing to read and the clock stayed frozen
    // for exactly as long as the user was watching it.
    it('starts polling the register as soon as a delegate works inside the turn', async () => {
      cockpit.toolSteps = []
      resetBackgroundWork()
      vi.mocked(BackgroundTasks).mockResolvedValue([
        { id: 'task_1', agent: 'explore', label: 'ตรวจไฟล์ทดสอบ', startedAt: new Date().toISOString(), toolCalls: 9, state: 'running', collected: false },
      ] as any)
      applyToolEvent({ action: 'call', ref: 'call_1', name: 'task', agent: 'explore' })
      applyToolEvent({ action: 'call', ref: 'r1', parent: 'call_1', task: 'task_1', name: 'list' })
      await vi.waitFor(() => expect(cockpit.backgroundTasks).toHaveLength(1))
    })
  })

  // A child whose task row is not in the list must still be visible. It happens
  // on a persisted turn and on out-of-order arrival, and silently dropping a row
  // means work that vanished.
  it('shows an orphaned delegate step rather than dropping it', () => {
    const { container } = render(Chat, {
      ...baseProps,
      awaitingReply: true,
      toolSteps: [
        { label: 'grep orphan', parent: 'gone', state: 'run', startedAt: Date.now() },
      ] as any,
      messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
    })
    expect(container.textContent).toContain('grep orphan')
  })

  it('counts sub-agents separately from tools in the collapsed line', () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        steps: [
          { label: 'read a.txt', state: 'done', startedAt: 0 },
          // agentKind as the engine stamps it (turn.ToolEvent.AgentKind, from
          // subagent.KindOf): explore lives in the subagents home, so it is a
          // helper. The frontend never derives this from the name.
          { label: 'task find every caller', ref: 'task_1', agent: 'explore', agentKind: 'helper', state: 'done', startedAt: 0 },
          { label: 'grep needle', parent: 'task_1', state: 'done', startedAt: 0 },
          { label: 'task_result task_1', state: 'done', startedAt: 0 },
        ],
      }] as any,
    })
    // Two separate toggles, like thinking and tools are separate: what the agent
    // did itself, and what it handed to someone else.
    const toggles = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
      .map((b) => b.textContent ?? '')
    expect(toggles.length).toBe(2)
    // Two tools of its own (read, task_result) — the delegate's grep is counted
    // inside its block — and one sub-agent.
    expect(toggles[0]).toContain('Used 2 tools')
    expect(toggles[1]).toContain('Sub-agents: 1')
    expect(toggles[0]).not.toContain('Sub-agents')
  })

  // The bug the owner reported on 2026-08-20: a `task` call that resolved
  // nobody — the tool was not even built — was drawn as "ซับเอเจน 1 ตัว ·
  // ล้มเหลว 1", so the screen named a kind of worker that had never been asked
  // for anything. The engine leaves agentKind empty exactly then, and unstamped
  // used to fall into the helper pile because that is where every delegation
  // sat before the two kinds were split.
  it('counts a delegation that resolved nobody as neither kind', () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        steps: [
          { label: 'task make the deck', ref: 'task_1', agent: 'deck', state: 'err', startedAt: 0 },
        ],
      }] as any,
    })
    const toggles = [...container.querySelectorAll('.meta-row .reasoning-toggle')]
      .map((b) => b.textContent ?? '')
    expect(toggles.length).toBe(1)
    expect(toggles[0]).toContain('Handed out: 1')
    expect(toggles[0]).not.toContain('Sub-agents')
    expect(toggles[0]).not.toContain('Agents')
    // And it is still counted as failed, which is the part that was right.
    expect(toggles[0]).toContain('failed')
  })

  // ข้อ 02: the card under a search row. Drawn through the reply's tools panel
  // because that path renders a finished timeline unfolded — the live block
  // folds what has finished, which is the right behaviour and the wrong fixture.
  it('draws what a search found under the row that ran it', async () => {
    const steps = [
      {
        label: 'web_search react server components', name: 'web_search',
        subject: 'react server components', state: 'done', startedAt: 0, secs: 2, count: 3,
        links: [
          { title: 'Server Components RFC', url: 'https://react.dev/rfc', opened: true },
          { title: 'Go 1.24 release notes', url: 'https://www.go.dev/blog/go1.24' },
          { title: 'Rendering: Server Components', url: 'https://nextjs.org/docs' },
        ],
      },
    ]
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{ role: 'agent', text: 'done', time: '10:54', steps }] as any,
    })
    const toolsBtn = container.querySelector('.meta-row .reasoning-toggle') as HTMLElement
    toolsBtn.click()
    await tick()

    const card = container.querySelector('.search-card')
    expect(card).not.toBeNull()
    // The query, in quotes, straight off the row's own subject.
    expect(card?.querySelector('.search-head .q')?.textContent).toContain('react server components')
    const hits = card!.querySelectorAll('.search-hit')
    expect(hits.length).toBe(3)
    expect(hits[0].querySelector('.ttl')?.textContent).toBe('Server Components RFC')
    // The host without the `www.` nobody reads, and its initials in place of a
    // favicon nobody is going to fetch over the network.
    expect(hits[1].querySelector('.dom')?.textContent).toBe('go.dev')
    expect(hits[1].querySelector('.fav')?.textContent).toBe('GO')
    // Only the one the agent went back and read wears the badge.
    expect(hits[0].querySelector('.was-read')).not.toBeNull()
    expect(hits[1].querySelector('.was-read')).toBeNull()
    expect(card?.querySelector('.search-foot')?.textContent)
      .toContain('3 results · 1 read in full · 2s')

    // One click, one tab. The row opens its own tab and the chat's link
    // handler met the same anchor on the way up and opened it again — two
    // tabs on one page from a single click (7 ก.ย.).
    workbench.tabs.length = 0
    ;(hits[1] as HTMLElement).click()
    await tick()
    const opened = workbench.tabs.filter((t) => t.kind === 'browser')
    expect(opened.map((t) => t.url)).toEqual(['https://www.go.dev/blog/go1.24'])
  })

  // Opening one panel closes the other, and each shows only its own kind.
  it('keeps the tools panel and the sub-agents panel apart', async () => {
    const steps = [
      { label: 'read a.txt', state: 'done', startedAt: 0 },
      { label: 'task hunt', ref: 't1', agent: 'explore', brief: 'go hunt', state: 'done', startedAt: 0 },
      { label: 'grep needle', parent: 't1', state: 'done', startedAt: 0 },
    ]
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{ role: 'agent', text: 'done', time: '10:54', steps }] as any,
    })
    const [toolsBtn, subsBtn] = [...container.querySelectorAll('.meta-row .reasoning-toggle')] as HTMLElement[]

    toolsBtn.click()
    await tick()
    expect(container.querySelector('.bgw-card')).toBeNull()
    // A step with only the old joined label still draws both halves: the verb
    // off its first word, the subject off the rest (toolSubject).
    expect(container.textContent).toContain('a.txt')
    expect(container.querySelectorAll('.tool-steps .tool-step').length).toBe(1)

    subsBtn.click()
    await tick()
    // The tools panel closed with it — one slot, same as thinking.
    expect(container.querySelectorAll('.tool-steps > .tool-step').length).toBe(0)
    const block = container.querySelector('.bgw-card')
    expect(block).toBeTruthy()
    expect(block?.querySelector('.bgw-longbrief')?.textContent).toContain('go hunt')
    // Finished, so its steps are folded away — see the fold's own test below.
    expect(block?.textContent).not.toContain('grep needle')
  })

  // Four finished delegations with their steps all open turned the transcript
  // into a wall ("มันติดกันจนดูยังไงไม่รู้"). A finished delegation's steps are
  // a record nobody asked to re-read; a running one's are the evidence it is
  // alive. Same rows, opposite defaults.
  describe('the step fold on a delegation', () => {
    const delegation = (state: string) => [
      { label: 'task hunt', ref: 't1', agent: 'explore', brief: 'go hunt', state, startedAt: Date.now() },
      { label: 'grep needle', parent: 't1', state, startedAt: Date.now() },
      { label: 'read hay.txt', parent: 't1', state, startedAt: Date.now() },
    ]

    // Folded while it works too, now that the card has a headline that moves:
    // the newest row is the top line of the card, so holding the whole list
    // open under it printed that row twice and put four concurrent delegations
    // back into the wall this fold was added to stop.
    it('is folded while the delegate works, with the live row as the headline', () => {
      const { container } = render(Chat, {
        ...baseProps, awaitingReply: true,
        toolSteps: delegation('run') as any,
        messages: [{ role: 'agent', text: 'done', time: '10:54' }] as any,
      })
      expect(container.querySelector('.bgw-now')?.textContent?.trim()).toBe('read hay.txt')
      expect(container.querySelectorAll('.bgw-work .tool-step').length).toBe(0)
    })

    it('is folded once it has finished, and says how many are behind it', async () => {
      const { container } = render(Chat, {
        ...baseProps,
        messages: [{ role: 'agent', text: 'done', time: '10:54', steps: delegation('done') }] as any,
      })
      const subsBtn = [...container.querySelectorAll('.meta-row .reasoning-toggle')].at(-1) as HTMLElement
      subsBtn.click()
      await tick()

      const fold = container.querySelector('.bgw-open') as HTMLElement
      expect(fold.textContent).toContain('See every step')
      // The count moved to the caption line beside the name, where it sits
      // with the clock: the control at the foot is a door, and labelling a
      // door with a number made it read as the summary it is not.
      expect(container.querySelector('.bgw-who')?.textContent).toContain('2 tools')
      expect(container.querySelectorAll('.bgw-work .tool-step').length).toBe(0)

      fold.click()
      await tick()
      // The delegate's work is drawn as a TURN now, not a flat row list, so its
      // finished calls sit behind their own phase header one level in — the
      // same fold the transcript uses, which is the whole point of drawing a
      // sub-agent's turn the way a turn is drawn.
      const inner = container.querySelector('.bgw-work button.phase-head') as HTMLElement
      inner.click()
      await tick()
      expect(container.querySelectorAll('.bgw-work .tool-step').length).toBe(2)
    })
  })

  // เอเจน and ซับเอเจน are different piles (COMPANY.md §4), and the chip
  // used to lump them: a turn where the doc agent made the file read "ซับเอเจน
  // 1 ตัว". The kind rides on the step (stamped by the engine from which home
  // the profile lives in), and each pile gets its own toggle and panel. A step
  // with no kind — every turn persisted before the split — stays a helper.
  it('counts an agent-kind delegation apart from the sub-agents', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        steps: [
          { label: 'read a.txt', state: 'done', startedAt: 0 },
          { label: 'task make the report', ref: 't1', agent: 'doc', agentKind: 'agent', brief: 'write the meeting summary', state: 'done', startedAt: 0 },
          { label: 'task hunt callers', ref: 't2', agent: 'explore', agentKind: 'helper', brief: 'find the callers', state: 'done', startedAt: 0 },
        ],
      }] as any,
    })
    const toggles = [...container.querySelectorAll('.meta-row .reasoning-toggle')] as HTMLElement[]
    const labels = toggles.map((b) => b.textContent ?? '')
    expect(labels.length).toBe(3)
    expect(labels[0]).toContain('Used 1 tools')
    expect(labels[1]).toContain('Agents: 1')
    expect(labels[2]).toContain('Sub-agents: 1')

    // Each toggle opens only its own pile.
    toggles[1].click()
    await tick()
    let blocks = [...container.querySelectorAll('.bgw-card')]
    expect(blocks.length).toBe(1)
    expect(blocks[0].querySelector('.bgw-agent')?.textContent).toContain('doc')

    toggles[2].click()
    await tick()
    blocks = [...container.querySelectorAll('.bgw-card')]
    expect(blocks.length).toBe(1)
    expect(blocks[0].querySelector('.bgw-agent')?.textContent).toContain('explore')
  })
})
