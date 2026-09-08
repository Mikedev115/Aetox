// `desk focus` — the verb the surface never had.
//
// Bringing something already on the desk back to the front used to be answered
// one way per kind and, for most kinds, not at all: the browser had `tabs
// select`, a file had `desk open` a second time (which re-reads it and rebuilds
// the pane, throwing away the scroll position and anything half-typed), and a
// terminal or a git pane had nothing. This is the one answer for all of them,
// and it is one assignment to activeId — nothing is created, read or closed.
import { readFileSync } from 'node:fs'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  workbench, openFileTab, openFilesTab, adoptWorkbenchSession, switchWorkbenchSession,
  routeDeskEvent, reportDeskTabs,
} from '../lib/stores/workbench.svelte'
import { ReadFile, WorkbenchTabsChanged } from './mocks/wailsApp'

// Through the door every desk event takes (§187.3), never around it.
const focus = (sessionId: string, tab: string, path = '') =>
  routeDeskEvent('focus-tab', { sessionId, tab, path })

const saved = (id: string) =>
  JSON.parse(localStorage.getItem(`aetox-workbench:${id}`) ?? '{"tabs":[],"activeIdx":-1}') as {
    tabs: { kind: string; path?: string; id?: string }[]
    activeIdx: number
  }

beforeEach(() => {
  vi.clearAllMocks()
  workbench.tabs.length = 0
  workbench.activeId = ''
  workbench.foreign.length = 0
  localStorage.clear()
  vi.mocked(ReadFile).mockResolvedValue('เนื้อไฟล์' as any)
})

describe('desk focus', () => {
  it('moves the view to a tab that a path could never have named', async () => {
    await adoptWorkbenchSession('on-screen')
    openFilesTab()
    await openFileTab('note.md')
    expect(workbench.activeId).toBe('file-note.md')

    focus('on-screen', 'files')

    expect(workbench.activeId).toBe('files')
    // Nothing opened, nothing closed: the same two tabs, in the same order.
    expect(workbench.tabs.map((t) => t.id)).toEqual(['files', 'file-note.md'])
  })

  it('takes a file by its path, the way open and close do', async () => {
    await adoptWorkbenchSession('on-screen')
    await openFileTab('output/s1/deck.html')
    openFilesTab()

    focus('on-screen', 'file-output/s1/deck.html', 'output/s1/deck.html')

    expect(workbench.activeId).toBe('file-output/s1/deck.html')
    // And the file was not read again — that is `open`'s job, and doing it
    // here would cost the user their scroll position for a switch.
    expect(ReadFile).toHaveBeenCalledTimes(1)
  })

  it('does nothing for a tab that closed between the report and the event', async () => {
    await adoptWorkbenchSession('on-screen')
    await openFileTab('note.md')

    // The Go side judged against a mirror that is one report behind. An
    // activeId pointing at a tab that is gone is a strip with no pane under it.
    focus('on-screen', 'file-vanished.md', 'vanished.md')

    expect(workbench.activeId).toBe('file-note.md')
  })

  it('parks a background chat’s focus on its own desk, not on the one on screen', async () => {
    await adoptWorkbenchSession('on-screen')
    await openFileTab('note.md')
    localStorage.setItem('aetox-workbench:background', JSON.stringify({
      tabs: [
        { kind: 'browser', name: 'localhost', url: 'http://localhost:5173', id: 'web-agent-1', mine: true },
        { kind: 'file', name: 'deck.html', path: 'output/background/deck.html', mine: true },
      ],
      activeIdx: 1,
    }))

    focus('background', 'web-agent-1')

    // The desk on screen never moved.
    expect(workbench.activeId).toBe('file-note.md')
    // The background chat's did, where its user will find it.
    expect(saved('background').activeIdx).toBe(0)
    // And Go was told, or desk_list in that chat would still mark the file.
    expect(WorkbenchTabsChanged).toHaveBeenLastCalledWith('background', [
      { kind: 'browser', name: 'localhost', path: '', url: 'http://localhost:5173', mine: true, id: 'web-agent-1', active: true },
      { kind: 'file', name: 'deck.html', path: 'output/background/deck.html', url: '', mine: true, id: 'file-output/background/deck.html', active: false },
    ])

    // Opening that chat finds it in front — which is what the tool promised.
    await switchWorkbenchSession('background')
    expect(workbench.activeId).toBe('web-agent-1')
  })

  it('reports which tab the user is looking at, on every change', async () => {
    await adoptWorkbenchSession('on-screen')
    await openFileTab('note.md')
    openFilesTab()

    // The window pushes the mirror from an effect over the store (Workbench.
    // svelte); this is that push, called directly.
    reportDeskTabs()

    const [, tabs] = vi.mocked(WorkbenchTabsChanged).mock.calls.at(-1)!
    expect(tabs.map((t: any) => [t.id, t.active])).toEqual([
      ['file-note.md', false],
      ['files', true],
    ])
  })
})

// The door has two halves and they are in different files: the window
// subscribes to a list of event names, and the router answers a switch of
// them. Nothing tied the two together, so a surface could be routed
// perfectly and never subscribed — every unit test passes, and the agent's
// tool reports success for an event no window is listening to. That is the
// mistake this change nearly shipped with, caught by hand.
describe('the desk door', () => {
  const read = (p: string) => readFileSync(new URL(p, import.meta.url), 'utf8')

  it('subscribes to every kind the router answers', () => {
    const routed = read('../lib/stores/workbench.svelte.ts')
      .split('export function routeDeskEvent')[1]
      .split('\n}')[0]
    const handled = [...routed.matchAll(/case '([^']+)':/g)].map((m) => m[1])

    const subscribed = read('../lib/workbench/Workbench.svelte')
      .match(/const offs = \[([^\]]*)\]/)![1]
      .match(/'([^']+)'/g)!
      .map((s) => s.slice(1, -1))

    expect(handled.length).toBeGreaterThan(0)
    expect([...handled].sort()).toEqual([...subscribed].sort())
  })
})
