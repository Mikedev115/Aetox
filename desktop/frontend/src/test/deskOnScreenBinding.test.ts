// The desk is bound to the chat on screen, not to the chat the engine is on.
//
// Those are the same chat until another one is read while one works
// (arriveAt), and then they are not: CurrentSessionID names the WORKING chat.
// refreshSessions runs at the tail of every door and handed that id to the
// desk, which undid the switch the door had just made — the viewed chat's tabs
// were saved under the working chat's id, and the working agent's pages became
// "live" (openAgentBrowserTabFor judges by the bound id), so a page another chat
// was browsing drew itself on the strip of the chat being read (owner, 7 ก.ย.:
// "เว็บไซต์แสดงผลทั้งที่เปิดหน้าอื่น … มันควรผูกกับแชทและเซสชั่น").
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { cockpit, refreshSessions } from '../lib/stores/cockpit.svelte'
import {
  workbench, openFilesTab, adoptWorkbenchSession, routeDeskEvent,
} from '../lib/stores/workbench.svelte'
import { CurrentSessionID, ListSessions } from './mocks/wailsApp'

const saved = (id: string) =>
  JSON.parse(localStorage.getItem(`aetox-workbench:${id}`) ?? '{"tabs":[]}') as {
    tabs: { kind: string; id?: string; mine?: boolean }[]
  }

beforeEach(async () => {
  vi.clearAllMocks()
  localStorage.clear()
  workbench.tabs.length = 0
  workbench.activeId = ''
  workbench.foreign.length = 0
  vi.mocked(ListSessions).mockResolvedValue([] as any)
})

describe('the desk follows the chat on screen', () => {
  it('stays with the chat being read while another chat works', async () => {
    // Reading `viewing`, with the file tree open on its desk...
    cockpit.openSession = 'viewing'
    await adoptWorkbenchSession('viewing')
    openFilesTab()
    // ...while the engine is busy in `working`.
    vi.mocked(CurrentSessionID).mockResolvedValue('working')

    await refreshSessions()

    // The working agent opens a page. It is that chat's, and it parks there —
    // never on the strip of the chat being read.
    routeDeskEvent('open-browser', { sessionId: 'working', id: 'web-agent-1', url: 'https://a.test' })
    expect(workbench.tabs.map((t) => t.kind)).toEqual(['files'])
    expect(workbench.foreign.map((t) => t.id)).toEqual(['web-agent-1'])
    expect(saved('working').tabs).toEqual([
      { kind: 'browser', id: 'web-agent-1', mine: true, url: 'https://a.test', name: 'a.test' },
    ])
    // And the viewed chat's desk was not written under the working chat's id.
    expect(saved('working').tabs.some((t) => t.kind === 'files')).toBe(false)
  })

  it('still takes the engine’s id when nothing is on screen yet', async () => {
    cockpit.openSession = ''
    vi.mocked(CurrentSessionID).mockResolvedValue('booting')

    await refreshSessions()

    routeDeskEvent('open-browser', { sessionId: 'booting', id: 'web-agent-1', url: 'https://a.test' })
    expect(workbench.tabs.map((t) => t.id)).toEqual(['web-agent-1'])
  })
})
