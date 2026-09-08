// The plan is drawn from the row, not from what the model typed
// (desktop/plan.go).
//
// Until 2026-09-08 a plan reached the screen exactly one way: the model wrote
// the whole document inside a ```plan fence, and the renderer put a border round
// it. That is also why revising one cost the document a second time — 70% of
// every byte of plan ever written on the owner's machine was a rewrite
// (TOKEN-AUDIT.md, PLAN REWRITES).
//
// Now the tool holds it and the window draws it, which moves two questions the
// fence answered by accident: the card has to come back when the chat is
// reopened, and it must not follow the user into another conversation. Both are
// §234's rule, and `plan` is classified there as dropped-and-re-read.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  cockpit, selectSession, applyPlanUpdate, queuedMessages, startPlanRun,
} from '../lib/stores/cockpit.svelte'
import { PendingUndo, SessionPlan, StartPlanRun, Stance, SendMessage } from './mocks/wailsApp'
import type { Session, Plan } from '../lib/types'

const A = '20260908-140000.001'
const B = '20260908-140000.002'

const plan = (over: Partial<Plan> = {}): Plan => ({
  title: 'ยกเพดานการรอเป็น 600 วิ',
  sections: [
    { heading: 'What is there now', body: 'waitMax is 60s' },
    { heading: 'What to change', body: '1. raise it' },
  ],
  version: 1,
  updated: '2026-09-08T14:00:00Z',
  ...over,
})

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.chat = []
  cockpit.openSession = ''
  cockpit.turnSession = ''
  cockpit.awaitingReply = false
  cockpit.parked = {}
  cockpit.sessions = []
  cockpit.plan = null
  queuedMessages.length = 0
  vi.mocked(PendingUndo).mockResolvedValue([] as any)
  vi.mocked(SessionPlan).mockResolvedValue({ title: '', sections: [], version: 0, updated: '' } as any)
})

describe('the plan belongs to its conversation', () => {
  it('takes an update for the chat on screen', () => {
    cockpit.openSession = A
    applyPlanUpdate({ sessionId: A, data: plan() } as any)
    expect(cockpit.plan?.title).toBe('ยกเพดานการรอเป็น 600 วิ')
    expect(cockpit.plan?.sections).toHaveLength(2)
  })

  it('drops one raised by a chat working in the background', () => {
    cockpit.openSession = A
    applyPlanUpdate({ sessionId: B, data: plan({ title: 'somebody else' }) } as any)
    // Not parked either, and it costs nothing: the row is already written by
    // the time the event goes out, so opening B finds its plan in the database.
    expect(cockpit.plan).toBeNull()
  })

  it('is re-read for the chat being opened, never carried across', async () => {
    cockpit.openSession = A
    applyPlanUpdate({ sessionId: A, data: plan() } as any)

    await selectSession({ id: B } as Session)

    // B has never had a plan. Carrying A's across is the bug §234 is about, and
    // it would be a particularly bad one here: a plan is a document somebody is
    // about to act on.
    expect(cockpit.plan).toBeNull()
    expect(vi.mocked(SessionPlan)).toHaveBeenCalledWith(B)
  })

  it('comes back when the chat is reopened, because the engine still has it', async () => {
    cockpit.openSession = A
    applyPlanUpdate({ sessionId: A, data: plan() } as any)

    await selectSession({ id: B } as Session)
    vi.mocked(SessionPlan).mockResolvedValue(plan() as any)
    await selectSession({ id: A } as Session)

    // The half the fence used to give for free: a plan written into a message
    // is still there when the transcript loads. Drawing from state means the
    // window has to go and ask, and this is that read.
    expect(cockpit.plan?.title).toBe('ยกเพดานการรอเป็น 600 วิ')
  })

  it('reads a chat with no plan as no card at all', async () => {
    cockpit.openSession = A
    vi.mocked(SessionPlan).mockResolvedValue({ title: '', sections: [], version: 0, updated: '' } as any)
    await selectSession({ id: B } as Session)
    // null rather than an empty plan: "no plan" and "a plan with nothing in it"
    // draw differently, and only the first is true here.
    expect(cockpit.plan).toBeNull()
  })

  it('keeps an amend’s marks so the user can see what moved', () => {
    cockpit.openSession = A
    applyPlanUpdate({ sessionId: A, data: plan() } as any)
    applyPlanUpdate({
      sessionId: A,
      data: plan({ version: 2, changed: ['What to change'] }),
    } as any)

    expect(cockpit.plan?.version).toBe(2)
    expect(cockpit.plan?.changed).toEqual(['What to change'])
  })
})

// STARTING A RUN MOVES THE DIAL, and the window has to be told.
//
// StartPlanRun crosses the session from วางแผน into ลงมือ inside the engine —
// without that the run carries no write, no shell, not even `plan_step`. But the
// dial reads `cockpit.stance`, which until now was only ever written when the
// user worked the dial themselves. So the engine ran in ลงมือ while the screen
// still said วางแผน: the run worked and the window lied about which mode it was
// in. Found on a real run, from a screenshot of the composer.
describe('starting a run', () => {
  it('brings the dial back in step with the engine', async () => {
    cockpit.openSession = A
    cockpit.stance = 'plan'
    vi.mocked(StartPlanRun).mockResolvedValue({ refusal: '', started: true } as any)
    vi.mocked(Stance).mockResolvedValue('' as any) // ลงมือ is the empty stance

    await startPlanRun()

    expect(cockpit.stance).toBe('')
    expect(vi.mocked(Stance)).toHaveBeenCalled()
  })

  it('says nothing and moves nothing when the engine refuses', async () => {
    cockpit.openSession = A
    cockpit.stance = 'plan'
    vi.mocked(StartPlanRun).mockResolvedValue({ refusal: 'ยังไม่มีแผนในห้องนี้' } as any)

    const refusal = await startPlanRun()

    expect(refusal).toBe('ยังไม่มีแผนในห้องนี้')
    // The dial must not move for a run that never started — a stance nothing is
    // enforcing is the failure setStance's own guard exists to avoid.
    expect(cockpit.stance).toBe('plan')
    expect(vi.mocked(SendMessage)).not.toHaveBeenCalled()
  })
})
