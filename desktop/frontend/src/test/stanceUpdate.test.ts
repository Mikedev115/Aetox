// The mode chip, when nobody pressed it.
//
// Every other way the dial moves is a click: the user opens the picker, or
// presses ลงมือ on a finished plan. `setStance` and `startPlanRun` both re-ask
// the engine right after, so the chip cannot be wrong for longer than a call.
//
// The assistant narrowing its OWN turn into วางแผน (internal/mode/plan_mode.go)
// has no click in front of it, which is why it needs an event at all — the same
// reason `model:switched` has one. Without this the engine would stop handing
// over `write` while the composer went on drawing ลงมือ, and that exact failure
// has already shipped once: StartPlanRun crossed the session into ลงมือ inside
// the engine and the window kept saying วางแผน (desktop/goal_run.go).
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  cockpit,
  applyStanceUpdate,
  newSessionAt,
  newChairSession,
} from '../lib/stores/cockpit.svelte'
import { CurrentSessionID, Stance } from './mocks/wailsApp'

const A = '20260909-120000.001'
const B = '20260909-120000.002'

beforeEach(() => {
  cockpit.openSession = A
  cockpit.stance = ''
})

describe('the assistant moving its own dial', () => {
  it('moves the chip for the chat on screen', () => {
    applyStanceUpdate({ sessionId: A, data: 'plan' })
    expect(cockpit.stance).toBe('plan')
  })

  it('leaves the chip alone when the turn is running in another chat', () => {
    applyStanceUpdate({ sessionId: B, data: 'plan' })
    // Nothing is parked for B either, and it costs nothing: the engine and the
    // sessions row both already hold the new value, so arriveAt reads it back
    // the moment that chat is opened (§234 classifies `stance` as re-read).
    expect(cockpit.stance).toBe('')
  })

  it('reads the engine’s answer rather than assuming the value', () => {
    // The engine is the one that decides; the window draws what it is told. A
    // stance this build does not implement comes back normalized, and the chip
    // must show what is actually in force rather than what was attempted.
    applyStanceUpdate({ sessionId: A, data: '' })
    expect(cockpit.stance).toBe('')
  })
})

// The dial on a chat that was never opened.
//
// A new session is born at ลงมือ on the engine side — startNewSession
// (desktop/sessions.go) carries the desk and the chair across and resets
// everything else — so the window has to arrive at ลงมือ too. It did not:
// afterNewSession is the one door with no refreshDesk at the tail, and
// refreshDesk was the only place the stance was ever re-read. Leave a วางแผน
// chat, press "แชทใหม่", and the chip went on saying วางแผน over an engine that
// was handing the model `write`, `change` and `shell`.
//
// §234 had already filed `stance` under "dropped, then asked for again"
// (cockpitBelongsToAChat.test.ts) — the answer was decided, and one door was
// not honouring it. The drop lives in arriveAt now, which every door passes
// through, so the next door added cannot forget the tail either.
describe('opening a new chat', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    cockpit.awaitingReply = false
    cockpit.openSession = A
    cockpit.space = ''
    cockpit.desk = 'assistant'
    cockpit.chair = ''
    vi.mocked(CurrentSessionID).mockResolvedValue(B)
  })

  it('puts the dial back to ลงมือ, where the engine already is', async () => {
    cockpit.stance = 'plan'

    await newSessionAt('coding')

    expect(cockpit.stance).toBe('')
  })

  it('does the same through the office door, which sets no coordinates of its own', async () => {
    cockpit.stance = 'plan'

    await newChairSession('automation')

    expect(cockpit.stance).toBe('')
  })

  it('asks the engine rather than assuming ลงมือ', async () => {
    // The drop alone would read right on a new chat and quietly wrong on every
    // other arrival — a reopened วางแผน chat has to come back on วางแผน. So the
    // drop is followed by a read, and this is the assertion that keeps the read
    // from being deleted as redundant.
    cockpit.stance = 'plan'
    vi.mocked(Stance).mockResolvedValue('consult')

    await newSessionAt('coding')
    await vi.waitFor(() => expect(cockpit.stance).toBe('consult'))

    expect(vi.mocked(Stance)).toHaveBeenCalled()
  })
})
