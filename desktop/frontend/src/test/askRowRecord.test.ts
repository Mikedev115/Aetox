// What was asked, and what we answered (owner, 7 ก.ย.: *"ควรจะดูย้อนหลังได้ว่า
// ถามอะไรและเราตอบอะไร"*).
//
// The `ask_user` card is drawn inside the live turn and is gone the instant it
// is answered, so the timeline row is the only trace the exchange leaves behind.
// It used to carry neither half of it — a bare "ถามคุณ · 2s" — because the
// question was an argument nothing named a row by, and the answer never left the
// tool's own text output.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { cockpit, applyToolEvent, restoreTranscript } from '../lib/stores/cockpit.svelte'
import { setLocale } from '../lib/i18n.svelte'
import Chat from '../lib/Chat.svelte'
import { GuideTopics } from './mocks/wailsApp'

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

const QUESTION = 'จะยิงรอบสองหรือเพิ่มเครื่องมือ'
const ANSWER = 'ยิงรอบสอง'

beforeEach(() => {
  setLocale('en')
  cockpit.chat = []
  cockpit.todos = []
  cockpit.ask = null
  cockpit.toolSteps = []
  cockpit.backgroundTasks = []
  cockpit.backgroundSteps = []
  vi.mocked(GuideTopics).mockResolvedValue([] as any)
})

describe('the row an ask_user call leaves behind', () => {
  it('takes the question off the call and the answer off the result', () => {
    applyToolEvent({ action: 'call', name: 'ask_user', subject: QUESTION })
    expect(cockpit.toolSteps[0]).toMatchObject({ subject: QUESTION, state: 'run' })
    // Nothing to answer with yet — the call has not returned.
    expect(cockpit.toolSteps[0].answer).toBeUndefined()

    applyToolEvent({ action: 'result', name: 'ask_user', ok: true, answer: ANSWER })
    expect(cockpit.toolSteps[0]).toMatchObject({ state: 'done', subject: QUESTION, answer: ANSWER })
  })

  // A reopened session rebuilds its rows from the stored parts alone. This is
  // the half that has to survive a restart, because the live card does not.
  // Through restoreTranscript rather than the fold itself: that is the door a
  // reopened session actually comes through.
  it('rebuilds both halves from a stored turn', () => {
    const [reopened] = restoreTranscript([{
      role: 'agent', text: 'done', time: '10:54',
      parts: [{ kind: 'tool', tool: { name: 'ask_user', subject: QUESTION, answer: ANSWER, ok: true, secs: 2 } }],
    }] as never)
    expect(reopened.steps?.[0]).toMatchObject({ name: 'ask_user', subject: QUESTION, answer: ANSWER })
  })

  it('draws the question and the answer on the row', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        steps: [{
          label: 'ask_user ' + QUESTION, name: 'ask_user', subject: QUESTION,
          answer: ANSWER, state: 'done', startedAt: 0, secs: 2,
        }],
      }] as any,
    })
    // The tool list is folded on a finished turn; open it.
    const toggles = container.querySelectorAll('.meta-row .reasoning-toggle')
    await fireEvent.click(toggles[toggles.length - 1])

    const row = container.querySelector('.tool-step') as HTMLElement
    expect(row.textContent).toContain(QUESTION)
    expect(row.textContent).toContain(ANSWER)
  })

  // Every other tool's row is unchanged: the answer is drawn only where there
  // is one, so nothing gains an empty label.
  it('says nothing about answers on a row that asked nothing', async () => {
    const { container } = render(Chat, {
      ...baseProps,
      messages: [{
        role: 'agent', text: 'done', time: '10:54',
        steps: [{ label: 'read app.go', name: 'read', subject: 'app.go', state: 'done', startedAt: 0, secs: 1 }],
      }] as any,
    })
    const toggles = container.querySelectorAll('.meta-row .reasoning-toggle')
    await fireEvent.click(toggles[toggles.length - 1])
    expect(container.querySelector('.tool-step .answer')).toBeNull()
    expect(container.querySelector('.tool-step .answered')).toBeNull()
  })
})
