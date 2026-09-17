// A history row is a navigation control, not a command to start one engine
// rebuild per click. These tests hold the first IPC call open and press more
// rows into it, which is the exact window that used to bootstrap A, B and C
// together and let whichever response finished last decide what was on screen.
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { cockpit, selectGlobalSession, selectSession, sendUserMessage } from '../lib/stores/cockpit.svelte'
import {
  GetModelInfo, ListSessions, ListSessionsForDoor, LoadSession, LoadSessionAnyProject,
  OpenSession, PendingUndo, SendMessage,
} from './mocks/wailsApp'

type ResolveTranscript = (rows: any[]) => void

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.chat = []
  cockpit.sessions = []
  cockpit.history = []
  cockpit.spaceHistory = []
  cockpit.openSession = 'home'
  cockpit.openingSession = ''
  cockpit.sessionError = ''
  cockpit.awaitingReply = false
  cockpit.parked = {}
})

describe('session switch queue', () => {
  it('loads one session at a time and commits only the latest click', async () => {
    const waiting = new Map<string, ResolveTranscript>()
    vi.mocked(LoadSessionAnyProject).mockImplementation((id: string) =>
      new Promise((resolve) => waiting.set(id, resolve as ResolveTranscript)) as never,
    )
    cockpit.history = [
      { id: 'a', title: 'A', ago: '' },
      { id: 'b', title: 'B', ago: '' },
      { id: 'c', title: 'C', ago: '' },
    ]

    const a = selectGlobalSession(cockpit.history[0])
    const b = selectGlobalSession(cockpit.history[1])
    const c = selectGlobalSession(cockpit.history[2])

    expect(cockpit.openingSession).toBe('c')
    expect(cockpit.history.find((s) => s.id === 'c')?.active).toBe(true)
    expect(vi.mocked(OpenSession).mock.calls.map(([id]) => id)).toEqual(['a'])
    expect(vi.mocked(OpenSession).mock.calls[0][2]).toBe(true)
    expect(vi.mocked(LoadSessionAnyProject).mock.calls.map(([id]) => id)).toEqual(['a'])

    waiting.get('a')?.([{ role: 'agent', text: 'answer A', time: '10:00' }])
    await vi.waitFor(() => {
      expect(vi.mocked(LoadSessionAnyProject).mock.calls.map(([id]) => id)).toEqual(['a', 'c'])
    })
    expect(vi.mocked(OpenSession).mock.calls.map(([id]) => id)).toEqual(['a', 'c'])
    // B was replaced before it crossed IPC, and A's late answer never flashed.
    expect(cockpit.chat).toEqual([])

    waiting.get('c')?.([{ role: 'agent', text: 'answer C', time: '10:01' }])
    await Promise.all([a, b, c])

    expect(cockpit.openSession).toBe('c')
    expect(cockpit.openingSession).toBe('')
    expect(cockpit.chat.map((m) => m.text)).toEqual(['answer C'])
  })

  it('coalesces a repeated click on the request already loading', async () => {
    let finish!: ResolveTranscript
    vi.mocked(LoadSessionAnyProject).mockImplementation(() =>
      new Promise((resolve) => { finish = resolve as ResolveTranscript }) as never,
    )
    const row = { id: 'a', title: 'A', ago: '' }

    const first = selectGlobalSession(row)
    const second = selectGlobalSession(row)
    expect(OpenSession).toHaveBeenCalledTimes(1)
    expect(LoadSessionAnyProject).toHaveBeenCalledTimes(1)

    finish([{ role: 'agent', text: 'only once', time: '10:00' }])
    await Promise.all([first, second])

    expect(LoadSessionAnyProject).toHaveBeenCalledTimes(1)
    expect(OpenSession).toHaveBeenCalledTimes(1)
    // These answers arrived inside OpenSession. A second call here means the
    // frontend rebuilt the old post-open IPC ladder.
    expect(GetModelInfo).toHaveBeenCalledTimes(1)
    expect(PendingUndo).toHaveBeenCalledTimes(1)
    expect(ListSessions).toHaveBeenCalledTimes(1)
    expect(ListSessionsForDoor).toHaveBeenCalledTimes(1)
    expect(cockpit.openSession).toBe('a')
    expect(cockpit.chat.map((m) => m.text)).toEqual(['only once'])
  })

  it('restores the current row when the latest target is refused', async () => {
    cockpit.history = [
      { id: 'home', title: 'Home', ago: '', active: true },
      { id: 'gone', title: 'Gone', ago: '' },
    ]
    vi.mocked(LoadSessionAnyProject).mockRejectedValue(new Error('ไม่พบเซสชันนี้'))

    await selectGlobalSession(cockpit.history[1])

    expect(cockpit.openingSession).toBe('')
    expect(cockpit.openSession).toBe('home')
    expect(cockpit.history.find((s) => s.id === 'home')?.active).toBe(true)
    expect(cockpit.sessionError).toBe('ไม่พบเซสชันนี้')
  })

  it('does not send a message to the old engine cursor while the target opens', async () => {
    let finish!: ResolveTranscript
    vi.mocked(LoadSessionAnyProject).mockImplementation(() =>
      new Promise((resolve) => { finish = resolve as ResolveTranscript }) as never,
    )

    const opening = selectGlobalSession({ id: 'next', title: 'Next', ago: '' })
    await sendUserMessage('must stay in the composer')

    expect(SendMessage).not.toHaveBeenCalled()
    finish([])
    await opening
  })

  it('keeps project session rows working while a dev backend lacks OpenSession', async () => {
    vi.mocked(OpenSession).mockRejectedValueOnce(
      new TypeError("window.go.main.App.OpenSession is not a function"),
    )
    vi.mocked(LoadSession).mockResolvedValue([
      { role: 'agent', text: 'opened through the compatible door', time: '10:00' },
    ] as never)

    await selectSession({ id: 'code-session', title: 'Code', ago: '', mode: 'coding' })

    expect(OpenSession).toHaveBeenCalledTimes(1)
    expect(LoadSession).toHaveBeenCalledWith('code-session')
    expect(LoadSessionAnyProject).not.toHaveBeenCalled()
    expect(cockpit.openSession).toBe('code-session')
    expect(cockpit.chat.map((message) => message.text)).toEqual([
      'opened through the compatible door',
    ])
    expect(cockpit.openingSession).toBe('')
  })
})
