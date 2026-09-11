// The engine started again under the window (§248 phase 2): the chat on
// screen is put back in front of the new engine — which opened on a fresh
// session, the way a launch does — and what the window shows is read again.
// Once per restart, and not at all for a reconnect that restarted nothing.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { LoadSessionAnyProject, GetModelInfo } from './mocks/wailsApp'
import { cockpit, resyncAfterEngineRestart } from '../lib/stores/cockpit.svelte'

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.openSession = ''
  cockpit.chat = []
  cockpit.sessionError = ''
  vi.mocked(LoadSessionAnyProject).mockResolvedValue([
    { role: 'user', text: 'สวัสดี', time: '' },
    { role: 'agent', text: 'สวัสดีครับ', time: '' },
  ] as any)
})

// The restart count is what tells one restart from the next, and the store
// remembers the last one it acted on across these tests — so each test names
// a count the one before it did not.
describe('resyncAfterEngineRestart', () => {
  it('reopens the chat on screen in the new engine and re-reads the window', async () => {
    cockpit.openSession = 's-on-screen'
    await resyncAfterEngineRestart(1)
    expect(LoadSessionAnyProject).toHaveBeenCalledWith('s-on-screen')
    expect(GetModelInfo).toHaveBeenCalled()
    expect(cockpit.chat.length).toBe(2)
  })

  it('does it once per restart, and not again for the same one', async () => {
    cockpit.openSession = 's-on-screen'
    await resyncAfterEngineRestart(2)
    await resyncAfterEngineRestart(2)
    expect(LoadSessionAnyProject).toHaveBeenCalledTimes(1)
    await resyncAfterEngineRestart(3)
    expect(LoadSessionAnyProject).toHaveBeenCalledTimes(2)
  })

  it('has nothing to reopen with no chat on screen, and still re-reads', async () => {
    await resyncAfterEngineRestart(4)
    expect(LoadSessionAnyProject).not.toHaveBeenCalled()
    expect(GetModelInfo).toHaveBeenCalled()
  })

  it('says why when the chat cannot be reopened, rather than showing nothing', async () => {
    cockpit.openSession = 's-gone'
    vi.mocked(LoadSessionAnyProject).mockRejectedValueOnce(new Error('session not found'))
    await resyncAfterEngineRestart(5)
    expect(cockpit.sessionError).toContain('session not found')
  })
})
