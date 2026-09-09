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
import { describe, it, expect, beforeEach } from 'vitest'
import { cockpit, applyStanceUpdate } from '../lib/stores/cockpit.svelte'

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
