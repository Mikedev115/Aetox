// ตัวหลัก (14 ก.ย. 2026): the two heads a person actually talks to, each with a
// page in the shape every specialist agent already has. The feedback that
// started this was "ไม่รู้ว่าตัวหลักปรับแต่งได้" — a thing with no page reads as
// unconfigurable. What is pinned: two cards and never a third, the agent
// editor's tab bar, the doors on each tab, and that what is about the person
// is NOT here (เกี่ยวกับคุณ). The memory tab is pinned in learningReview.test.ts.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import { ListModes, LearnedScopeInfos, LearnedEntries, ListPendingChanges } from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

const openSection = async (container: HTMLElement, label: string) => {
  const items = Array.from(container.querySelectorAll('.settings-nav-item'))
  const item = items.find((el) => el.textContent?.trim() === label) ?? items.find((el) => el.textContent?.includes(label))
  if (!item) throw new Error(`nav item "${label}" not found`)
  await fireEvent.click(item)
}
const cards = (c: HTMLElement) => Array.from(c.querySelectorAll<HTMLElement>('.main-card'))
const tabs = (c: HTMLElement) => Array.from(c.querySelectorAll<HTMLElement>('.ag-tabs-bar [role="tab"]'))

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'settings'
  cockpit.capabilityIntent = null
  vi.mocked(ListModes).mockResolvedValue([
    { name: 'assistant', description: 'โต๊ะผู้ช่วย, ทำได้ทุกอย่างบนเครื่อง' },
    { name: 'coding', description: 'โต๊ะเขียนโค้ด' },
    { name: 'specialized', description: 'โต๊ะเอเจน' },
  ] as any)
  vi.mocked(LearnedScopeInfos).mockResolvedValue([{ scope: '', orphan: false }, { scope: 'mode:coding', orphan: false, projectsUnder: true }] as any)
  vi.mocked(LearnedEntries).mockImplementation(async (scope: string) => (scope === '' ? ['a', 'b'] : []) as any)
  vi.mocked(ListPendingChanges).mockResolvedValue([] as any)
})

describe('the list', () => {
  it('draws the two heads as agent cards with the desk file\'s own line, and no third', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    const names = cards(container).map((c) => c.querySelector('.chair-name')?.textContent?.trim())
    expect(names).toEqual(['ผู้ช่วย', 'โค้ด'])
    // The face is the head's own rig, not an icon, with its rank on the
    // corner (RankedFace) like every other face in the app.
    expect(cards(container)[0].querySelector('.mascot')).toBeTruthy()
    expect(cards(container)[0].querySelector('.rank-corner.rank-head')).toBeTruthy()
    await waitFor(() => expect(cards(container)[0].textContent).toContain('ทำได้ทุกอย่างบนเครื่อง'))
    // No memory count on the card (owner, 14 ก.ย.: "จำไว้ 0 บรรทัด เอาออก");
    // the number is on the memory tab, one click in.
    expect(cards(container)[0].textContent).not.toContain('บรรทัด')
    // The person's layer is named as elsewhere, never drawn here.
    expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ตัวหลัก')
    expect(container.textContent).not.toContain('USER.md')
    expect(container.querySelector('.office-note .linklike')?.textContent).toBe('เกี่ยวกับคุณ')
  })

  it('counts what waits, on the card and on the rail', async () => {
    vi.mocked(ListPendingChanges).mockResolvedValue([
      { id: 1, kind: 'memory', scope: 'mode:coding', op: 'add', body: 'x', state: 'pending' },
      { id: 2, kind: 'memory', scope: 'user:profile', op: 'add', body: 'y', state: 'pending' },
    ] as any)
    cockpit.pendingLearned = 2
    const { container } = render(Settings, { onClose: () => {} })
    const row = Array.from(container.querySelectorAll('.settings-nav-item')).find((el) => el.textContent?.includes('ตัวหลัก'))!
    expect(row.querySelector('.nav-count')?.textContent?.trim()).toBe('2')
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await waitFor(() => expect(cards(container)[1].textContent).toContain('รออนุมัติ 1'))
    expect(cards(container)[0].textContent).not.toContain('รออนุมัติ')
    cockpit.pendingLearned = 0
  })
})

describe('a head\'s page', () => {
  it('opens from the card in the agent editor\'s shape, and crosses to the other head', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await fireEvent.click(cards(container)[0].querySelector('.icobtn')!)
    await waitFor(() => expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ตั้งค่า ผู้ช่วย'))
    expect(tabs(container).map((t) => t.textContent?.trim())).toEqual(['ตัวตน', 'การเข้าถึง', 'ความจำ'])
    expect(container.querySelector('.main-head .rank-corner.rank-head')).toBeTruthy()
    expect(container.querySelector('.main-head .mascot')).toBeTruthy()
    // The desk file's line on the first tab, and where the persona still is.
    expect(container.querySelector('.ag-tab-panel.on')?.textContent).toContain('ทำได้ทุกอย่างบนเครื่อง')
    expect(container.querySelector('.ag-tab-panel.on')?.textContent).toContain('modes/assistant.md')

    await fireEvent.click(Array.from(container.querySelectorAll('.pp-bar .ctrl')).find((b) => b.textContent?.includes('ไปที่ โค้ด'))!)
    await waitFor(() => expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('ตั้งค่า โค้ด'))
    expect(container.querySelector('.ag-tab-panel.on')?.textContent).toContain('modes/coding.md')

    await fireEvent.click(Array.from(container.querySelectorAll('.pp-bar .ctrl'))[0])
    await waitFor(() => expect(cards(container).length).toBe(2))
  })

  // Reach is doors into ห้องความสามารถ, at the page that answers for the
  // desk — the room stays the one place any of it is changed.
  it('reach doors into the capability room at the matching page', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'ตัวหลัก')
    await waitFor(() => expect(cards(container).length).toBe(2))
    await fireEvent.click(cards(container)[1].querySelector('.icobtn')!)
    await fireEvent.click(tabs(container).find((t) => t.textContent?.includes('การเข้าถึง'))!)
    const rows = Array.from(container.querySelectorAll('.ag-tab-panel.on .set-row'))
    expect(rows.map((r) => r.querySelector('.t')?.textContent?.trim())).toEqual([
      'ตั้งค่า MCP ฝั่งผู้ช่วยและโค้ด', 'สกิลของคุณ', 'ทะเบียนเครื่องมือ',
    ])
    await fireEvent.click(rows[0].querySelector('.ctrl')!)
    expect(cockpit.capabilityIntent).toEqual({ page: 'desks' })
    expect(cockpit.activeView).toBe('capability')
  })

})
