// The chat list in the storefront's column, from the outside.
//
// Two rules are worth pinning. The first is that the list is grouped by
// calendar day: fourteen rows each stamped "3 วันที่แล้ว" is not a list a user
// scans, it is one they read, and the header is what lets the eye skip. The
// second is that the search box is summoned rather than permanent — it and the
// new-session button shrank to two icons on the row above the list, and a
// regression that re-pins them as blocks would take a third of the column's
// height back without anyone noticing.
//
// Search results rank by match, not by date, so they are deliberately NOT
// grouped: a "วันนี้" header printed three times down one list would be
// claiming an order the results do not have.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte'
import Sidebar from '../lib/Sidebar.svelte'
import {
  SessionMode, CurrentSessionID, LoadSession, LoadSessionAnyProject,
  OpenProjectPath, OpenSession,
} from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setShell } from '../lib/shell.svelte'
import type { Session } from '../lib/types'

const daysAgo = (n: number, hour = 12): string => {
  const d = new Date()
  d.setDate(d.getDate() - n)
  d.setHours(hour, 0, 0, 0)
  return d.toISOString()
}

const chat = (id: string, title: string, updatedAt: string): Session =>
  ({ id, title, ago: '', updatedAt })

// The DAY headings only. The column also carries section headings — ที่ปักหมุด
// and โปรเจกต์, which arrived on 30 ส.ค. — and they wear .sect precisely because
// they are a different kind of thing: furniture that is there whether or not
// anything is under it, where a day heading is generated from the rows it
// heads. Every assertion in this file is about the second kind.
const dayHeads = (): string[] =>
  Array.from(document.querySelectorAll('.sess-day-head:not(.sect)')).map((e) => e.textContent?.trim() ?? '')

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.removeItem('chatsPinned')
  cockpit.desk = ''
  cockpit.activeView = 'chat'
  cockpit.history = []
  cockpit.sessions = []
  cockpit.spaces = []
  cockpit.space = ''
  cockpit.spaceHistory = []
  cockpit.historyFault = null
  cockpit.openSession = ''
  cockpit.openingSession = ''
  cockpit.sessionError = ''
  Object.assign(cockpit.project, { focused: false, name: '', path: '', branch: '' })
  setShell('assistant')
  vi.mocked(CurrentSessionID).mockResolvedValue('20260805-120000.000')
  vi.mocked(SessionMode).mockResolvedValue('')
})

describe('the chat list', () => {
  it('groups rows under the day they belong to, in the list order', () => {
    cockpit.history.push(
      chat('a', 'วันนี้เช้า', daysAgo(0, 9)),
      chat('b', 'เมื่อคืน', daysAgo(1, 23)),
      chat('c', 'อาทิตย์ก่อน', daysAgo(4)),
      chat('d', 'เดือนก่อน', daysAgo(20)),
      chat('e', 'นานมาก', daysAgo(200)),
    )
    render(Sidebar, { onOpenSettings: () => {} })

    expect(dayHeads()).toEqual([
      'วันนี้', 'เมื่อวาน', '7 วันที่ผ่านมา', '30 วันที่ผ่านมา', 'เก่ากว่านั้น',
    ])
  })

  it('opens one header per run, not one per row', () => {
    cockpit.history.push(
      chat('a', 'หนึ่ง', daysAgo(0, 9)),
      chat('b', 'สอง', daysAgo(0, 10)),
      chat('c', 'สาม', daysAgo(0, 11)),
    )
    render(Sidebar, { onOpenSettings: () => {} })

    expect(dayHeads()).toEqual(['วันนี้'])
    expect(document.querySelectorAll('.sess-row').length).toBe(3)
  })

  // A chat that predates the stamp still belongs somewhere. Dropping it would
  // be losing a row to a formatting detail.
  it('keeps a row with no timestamp, at the far end', () => {
    cockpit.history.push(chat('a', 'ไม่มีเวลา', ''))
    render(Sidebar, { onOpenSettings: () => {} })

    expect(dayHeads()).toEqual(['เก่ากว่านั้น'])
    expect(screen.getByText('ไม่มีเวลา')).toBeTruthy()
  })

  // The row above the list holds the field itself, not an icon that reveals
  // one: two glyphs alone left half the row empty, and this is the control
  // that says what the column below it is.
  it('draws the search field on the row above the list', () => {
    render(Sidebar, { onOpenSettings: () => {} })

    const field = document.querySelector('.side-actions .side-search input')
    expect(field).toBeTruthy()
    expect(field?.getAttribute('placeholder')).toBe('ค้นหาประวัติ…')
  })

  // Search is global; creation belongs to the section it affects. This keeps a
  // project folder and a general new chat as two explicit doors.
  it('puts project and chat actions on their own stable headings', () => {
    render(Sidebar, { onOpenSettings: () => {} })

    expect(screen.queryByText('เริ่มเซสชันใหม่')).toBeNull()
    expect(document.querySelectorAll('.side-actions button')).toHaveLength(0)
    const labels = Array.from(document.querySelectorAll('.rail-section-actions button'))
      .map((b) => b.getAttribute('aria-label'))
    expect(labels).toEqual(['สร้างโปรเจกต์', 'เมนูโปรเจกต์', 'เริ่มเซสชันใหม่', 'เมนูแชท'])
  })

  // The row's whole job. Every other way into a session switched the view to
  // the chat; this one loaded the conversation and left the user looking at
  // whatever page they were on — the row lit up, nothing else moved, and the
  // only reading available from the outside was that the row does not work.
  it('shows the chat when a row is clicked from another page', async () => {
    vi.mocked(LoadSessionAnyProject).mockResolvedValue([] as never)
    cockpit.activeView = 'settings'
    cockpit.history.push(chat('a', 'เปิดอันนี้', daysAgo(0, 9)))
    render(Sidebar, { onOpenSettings: () => {} })

    await fireEvent.click(screen.getByText('เปิดอันนี้'))

    expect(cockpit.activeView).toBe('chat')
  })

  // The engine writes a sentence for each of its seven refusals — the folder
  // moved, the desk file is gone, the session is not in this project. An
  // unhandled rejection here used to swallow every one of them, which made a
  // session that cannot open indistinguishable from a row that is not wired up.
  it('says why when the engine refuses to open the session', async () => {
    vi.mocked(LoadSessionAnyProject).mockRejectedValue(
      new Error('ไม่พบโปรเจกต์ของเซสชันนี้ (โฟลเดอร์อาจถูกย้ายหรือลบไปแล้ว)'),
    )
    cockpit.sessionError = ''
    cockpit.history.push(chat('err-session', 'เปิดไม่ได้', daysAgo(0, 9)))
    render(Sidebar, { onOpenSettings: () => {} })

    await fireEvent.click(screen.getByText('เปิดไม่ได้'))

    expect(cockpit.sessionError).toContain('โฟลเดอร์อาจถูกย้าย')
  })

  // The title is the only thing on a row that says what the chat was about, so
  // it gets the line — the chip, the age and the hover buttons share the one
  // below it. They used to sit on the title's line, all of them flex:none, and
  // in a 280px column the title was squeezed to a couple of words.
  it('gives the title a line of its own, above the chip and the age', () => {
    cockpit.history.push({ ...chat('a', 'ค้นไฟล์ทั้งเครื่องแล้วสรุปให้ที', daysAgo(0, 9)), agent: 'ผู้ช่วย' })
    render(Sidebar, { onOpenSettings: () => {} })

    const row = document.querySelector('.sess-row')
    expect(row?.querySelector('.sess-line')?.textContent?.trim()).toBe('ค้นไฟล์ทั้งเครื่องแล้วสรุปให้ที')
    expect(row?.querySelector('.sess-meta')?.textContent).toContain('ผู้ช่วย')
    // The overflow menu lives on the meta line, in space nothing else wants.
    expect(row?.querySelector('.sess-meta .sess-acts .row-more')).toBeTruthy()
  })

  it('keeps assistant session actions inside one overflow menu', async () => {
    cockpit.history.push(chat('assistant-chat', 'งานในหน้าผู้ช่วย', daysAgo(0, 9)))
    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    expect(container.querySelector('.sess-row .sess-pin')).toBeNull()
    expect(container.querySelector('.sess-row .sess-exp')).toBeNull()
    expect(container.querySelector('.sess-row .sess-del')).toBeNull()
    const trigger = container.querySelector('.sess-row .row-more') as HTMLElement
    const scroll = trigger.closest('.scroll') as HTMLElement
    vi.spyOn(trigger, 'getBoundingClientRect').mockReturnValue({ top: 400, bottom: 424 } as DOMRect)
    vi.spyOn(scroll, 'getBoundingClientRect').mockReturnValue({ top: 100, bottom: 450 } as DOMRect)
    await fireEvent.click(trigger)

    expect(screen.getByText('ปักหมุดแชทนี้')).toBeTruthy()
    expect(screen.getByText('Markdown')).toBeTruthy()
    expect(screen.getByText('JSON')).toBeTruthy()
    expect(screen.getByText('ลบแชท')).toBeTruthy()
    expect(container.querySelector('.sess-row-menu')?.classList.contains('up')).toBe(true)
  })

})

// Starting work in a project, from the project's own row.
//
// Until this there was one way in: click the project's name. That works —
// opening a project starts a fresh session on it — but every other list in the
// app treats a name as "show me this", so the action was hidden inside a
// gesture that reads as navigation (owner, 16 ส.ค., pointing at the row:
// "เนี้ยมันไม่มี").
describe('the project rows in the workshop column', () => {
  it('starts with lightweight project cards and opens only the chosen project', async () => {
    setShell('code')
    cockpit.projects = [
      { key: 'a', name: 'frontend', folder: 'frontend', description: 'หน้าร้าน', sessions: 4, path: 'D:/work/frontend', active: false },
      { key: 'b', name: 'senior-architect-agent', folder: 'agent', description: '', sessions: 2, path: 'D:/work/agent', active: false },
    ] as any
    vi.mocked(OpenProjectPath).mockResolvedValue({ focused: true, name: 'frontend', path: 'D:/work/frontend' } as any)

    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    expect(screen.getByText('เลือกโปรเจกต์')).toBeTruthy()
    expect(container.querySelectorAll('.project-card').length).toBe(2)
    expect(container.querySelector('.project-session-row')).toBeNull()

    await fireEvent.click(container.querySelectorAll('.project-card-open')[0])
    expect(vi.mocked(OpenProjectPath)).toHaveBeenCalledWith('D:/work/frontend')
    expect(cockpit.activeView).toBe('chat')
  })

  it('offers description editing immediately and leaves an empty project compact', async () => {
    setShell('code')
    cockpit.projects = [
      { key: 'a', name: 'Aetox', folder: 'Aetox', description: '', sessions: 3, path: 'D:/Aetox', active: false },
    ] as any

    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    expect(container.querySelector('.project-card-description')).toBeNull()
    expect(screen.queryByText('เพิ่มคำอธิบายว่าโปรเจกต์นี้ทำอะไร')).toBeNull()
    const add = container.querySelector('.project-card-edit-button') as HTMLButtonElement
    expect(add.title).toBe('เพิ่มคำอธิบายว่าโปรเจกต์นี้ทำอะไร')
    await fireEvent.click(add)
    expect(container.querySelector('.project-card-edit textarea')).toBeTruthy()
  })

  it('is not in the storefront column, which focuses no project at all', () => {
    setShell('assistant')
    cockpit.projects = [{ key: 'a', name: 'frontend', path: 'D:/work/frontend', active: false }] as any

    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    expect(container.querySelector('.project-switcher')).toBeNull()
  })

  it('opens a nested project session through the queued session door', async () => {
    setShell('code')
    Object.assign(cockpit.project, { focused: true, name: 'Aetox', path: 'D:/Aetox', branch: 'main' })
    cockpit.projects = [
      { key: 'a', name: 'Aetox', folder: 'Aetox', description: '', sessions: 2, path: 'D:/Aetox', active: true },
    ] as any
    cockpit.sessions = [
      { ...chat('draft', 'แชทใหม่', daysAgo(0, 9)), active: true, mode: 'coding' },
      { ...chat('older', 'เซสชันที่ต้องเปิด', daysAgo(0, 8)), mode: 'coding' },
    ]
    cockpit.openSession = 'draft'
    vi.mocked(SessionMode).mockResolvedValue('coding')
    vi.mocked(LoadSession).mockResolvedValue([
      { role: 'agent', text: 'กลับมาแล้ว', time: '10:00' },
    ] as never)

    render(Sidebar, { onOpenSettings: () => {} })
    await fireEvent.click(screen.getByText('เซสชันที่ต้องเปิด'))

    await waitFor(() => expect(OpenSession).toHaveBeenCalled())
    expect(vi.mocked(OpenSession).mock.calls[0][0]).toBe('older')
    expect(vi.mocked(OpenSession).mock.calls[0][2]).toBe(false)
    expect(LoadSession).toHaveBeenCalledWith('older')
  })

  it('pins a project session to the top and remembers the pin', async () => {
    setShell('code')
    Object.assign(cockpit.project, { focused: true, name: 'Aetox', path: 'D:/Aetox', branch: 'main' })
    cockpit.projects = [
      { key: 'a', name: 'Aetox', folder: 'Aetox', description: '', sessions: 2, path: 'D:/Aetox', active: true },
    ] as any
    cockpit.sessions = [
      { ...chat('newer', 'งานล่าสุด', daysAgo(0, 9)), mode: 'coding' },
      { ...chat('keeper', 'งานที่ต้องปักหมุด', daysAgo(2, 9)), mode: 'coding' },
    ]

    render(Sidebar, { onOpenSettings: () => {} })
    await fireEvent.click(screen.getAllByLabelText('เพิ่มเติม')[1])
    await fireEvent.click(screen.getByText('ปักหมุดแชทนี้'))

    await waitFor(() => {
      const titles = Array.from(document.querySelectorAll('.project-session-title')).map((node) => node.textContent)
      expect(titles).toEqual(['งานที่ต้องปักหมุด', 'งานล่าสุด'])
      expect(JSON.parse(localStorage.getItem('chatsPinned') ?? '{}')).toEqual({ keeper: true })
    })
    await fireEvent.click(screen.getAllByLabelText('เพิ่มเติม')[0])
    expect(screen.getByText('เอาหมุดออก')).toBeTruthy()
    expect(screen.getByText('Markdown')).toBeTruthy()
    expect(screen.getByText('JSON')).toBeTruthy()
  })

  it('shows the reason in the project sidebar when a session cannot open', async () => {
    setShell('code')
    Object.assign(cockpit.project, { focused: true, name: 'Aetox', path: 'D:/Aetox', branch: 'main' })
    cockpit.projects = [
      { key: 'a', name: 'Aetox', folder: 'Aetox', description: '', sessions: 1, path: 'D:/Aetox', active: true },
    ] as any
    cockpit.sessions = [{ ...chat('gone', 'เปิดไม่ได้', daysAgo(0, 8)), mode: 'coding' }]
    vi.mocked(OpenSession).mockRejectedValueOnce(new Error('ไม่พบเซสชันนี้'))

    render(Sidebar, { onOpenSettings: () => {} })
    await fireEvent.click(screen.getByText('เปิดไม่ได้'))

    expect((await screen.findByRole('alert')).textContent).toContain('ไม่พบเซสชันนี้')
  })

  it('opens the project menu inside a raised card instead of clipping it', async () => {
    setShell('code')
    cockpit.projects = [
      { key: 'a', name: 'index', folder: 'index', description: '', sessions: 0, path: 'D:/index', active: false },
    ] as any

    const { container } = render(Sidebar, { onOpenSettings: () => {} })
    await fireEvent.click(screen.getByText('โปรเจกต์ทั้งหมด'))
    await fireEvent.click(container.querySelector('.project-card-menu-button') as HTMLButtonElement)

    const card = container.querySelector('.project-card')
    expect(card?.classList.contains('menu-open')).toBe(true)
    expect(card?.querySelector('.project-card-menu .plus-menu-item')?.textContent).toContain('เอาออกจากรายการ')
  })
})

// The working ring, in the column that nests its chats under a project.
//
// It was written for .sess-row and only .sess-row — the flat list the
// storefront draws. The workshop draws no flat list at all: every chat there
// hangs under its project as a .proj-group-sess, so a turn left running had no
// mark anywhere in that column, and the one thing the ring exists for — walking
// away from a turn and still knowing it is going — did not work on the side of
// the app where the long turns actually happen (owner, 22 ส.ค.).
describe('the working ring on a workshop project chat', () => {
  beforeEach(() => {
    cockpit.turnSession = ''
    cockpit.parked = {}
  })

  it('rings the chat the turn is running in, and no other', () => {
    setShell('code')
    Object.assign(cockpit.project, { focused: true, name: 'senior-architect-agent', path: 'D:/work/agent', branch: 'main' })
    cockpit.projects = [
      { key: 'b', name: 'senior-architect-agent', folder: 'agent', description: '', sessions: 2, path: 'D:/work/agent', active: true },
    ] as any
    cockpit.sessions.push(
      { ...chat('s1', 'ลองแก้ไฟล์ให้ผมหน่อย', daysAgo(0, 9)) },
      { ...chat('s2', 'เทสครับแก้ไฟล์แล้ว', daysAgo(0, 10)) },
    )
    cockpit.turnSession = 's1'

    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    const rows = Array.from(container.querySelectorAll('.project-session-row'))
    expect(rows.length).toBe(2)
    expect(rows[0].classList.contains('working')).toBe(true)
    expect(rows[1].classList.contains('working')).toBe(false)
  })

  it('rings a chat left working in the background, which is the whole point', () => {
    setShell('code')
    Object.assign(cockpit.project, { focused: true, name: 'senior-architect-agent', path: 'D:/work/agent', branch: 'main' })
    cockpit.projects = [
      { key: 'b', name: 'senior-architect-agent', folder: 'agent', description: '', sessions: 1, path: 'D:/work/agent', active: true },
    ] as any
    cockpit.sessions.push(
      { ...chat('s1', 'งานยาว', daysAgo(0, 9)) },
    )
    // Parked: the user switched away while the turn was still going, so it is
    // no longer the engine's session and only its own record says it is alive.
    cockpit.parked = { s1: { awaitingReply: true } } as any

    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    expect(container.querySelector('.project-session-row')?.classList.contains('working')).toBe(true)
  })
})

// The 7 ก.ย. 2026 failure, from the outside: an empty column that means "your
// chats cannot be read" must not look like the empty column that means "you
// have no chats". Those were the same picture for as long as the list existed,
// and the owner read the only reading the window offered — that 77 sessions
// had been erased.
describe('a history the store could not open', () => {
  it('says so above the list instead of drawing an ordinary empty column', () => {
    cockpit.historyFault = { failed: true, tooNew: true, have: 18, known: 17, message: 'schema 18 > 17' }

    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    const banner = container.querySelector('.hist-fault')
    expect(banner).not.toBeNull()
    // Both numbers, because "update the app" is only actionable when the user
    // can see which way round the mismatch goes.
    expect(banner?.textContent).toContain('18')
    expect(banner?.textContent).toContain('17')
  })

  it('says nothing when the store is fine and the history is simply empty', () => {
    cockpit.historyFault = null

    const { container } = render(Sidebar, { onOpenSettings: () => {} })

    expect(container.querySelector('.hist-fault')).toBeNull()
  })
})
