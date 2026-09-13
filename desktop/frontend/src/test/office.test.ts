// เอเจนเฉพาะทาง (COMPANY.md §4). The page is a roster and a feed, and the one
// claim worth pinning is what the roster shows: the tools each chair *gets*,
// after the office ceiling — not the ones its file asked for. This is the row a
// person checks the ceiling on, so a page that echoed the request back would
// quietly defeat the structure it is reporting on.
//
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Office from '../lib/Office.svelte'
import {
  ListChairs, ListReceivedJobs, LoadSessionAnyProject, NewChairSessionAt, ListTeams,
} from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

const chair = (over: Record<string, unknown> = {}) => ({
  name: 'doc', description: 'เก้าอี้ร่างเอกสาร', tools: ['doc_write', 'read', 'pdf_read'],
  builtin: true, jobs: 0, lastUsed: '', ...over,
})

const job = (over: Record<string, unknown> = {}) => ({
  id: 1, chair: 'doc', sessionId: '20260805-090000.000',
  request: 'ทำเอกสารสรุปยอดเดือนนี้', answer: 'เขียนเสร็จแล้ว',
  toolSeq: 'pdf_read>doc_write', toolCount: 2, durationMs: 4200,
  outcome: 'unknown', time: new Date().toISOString(), ...over,
})


beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'office'
  cockpit.settingsIntent = null
  vi.mocked(ListChairs).mockResolvedValue([chair()] as any)
  vi.mocked(ListReceivedJobs).mockResolvedValue([] as any)
  // Teams reach this page as a chip on the card (§256). The seeded team,
  // everybody on it, is the shape these card tests mock.
  vi.mocked(ListTeams).mockImplementation(async () => [{
    name: 'ผู้ช่วยในคอมพิวเตอร์', desk: 'specialized', description: '', invalid: '',
    missing: [], members: await ListChairs(), path: '',
  }] as any)
})

describe('the office roster', () => {
  // A card is a face, not an inventory (2026-08-07). The tool chips were six
  // per card and five of the six were identical on every card — the office
  // ceiling hands everyone the same set — so the list took half the card to say
  // nothing about who anyone is. What the card answers now is who this is and
  // what they make; the tools moved to the editor behind the gear, which is
  // also the only place they can be changed.
  it('shows who a chair is, and no longer lists their tools', async () => {
    vi.mocked(ListChairs).mockResolvedValue([chair({ tools: ['doc_write', 'read'] })] as any)
    render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getAllByText('doc').length).toBeGreaterThan(0))
    expect(screen.getByText('เก้าอี้ร่างเอกสาร')).toBeTruthy()
    expect(screen.queryByText('doc_write')).toBeNull()
    expect(screen.queryByText('read')).toBeNull()
  })

  // Every agent arrives with a face, including one whose profile names no icon
  // — the roster derives it from what they produce (desktop/office.go). A card
  // that could render blank would be the feature shipping broken for everyone
  // who never opens the editor.
  it('draws a mark for every chair', async () => {
    vi.mocked(ListChairs).mockResolvedValue([chair({ icon: 'fileText' })] as any)
    const { container } = render(Office, { onClose: () => {} })

    await waitFor(() => expect(container.querySelector('.mascot svg')).toBeTruthy())
    // The icon is on the ears, and a tile in the roster does not move.
    expect(container.querySelector('.mascot .ms-earL .ms-badge path')).toBeTruthy()
    expect(container.querySelector('.mascot.still')).toBeTruthy()
  })

  // The half of that promise the icon cannot keep. A mascot is drawn from the
  // NAME, and the badge on its ears is the only part `icon:` decides — so a
  // profile that names none still arrives as somebody, wearing the logo,
  // rather than an empty square. This is the case that made a drawn face
  // worth having over a stored picture: it is the shape of every agent a user
  // writes themselves.
  it('draws a face for a chair whose profile names no icon', async () => {
    vi.mocked(ListChairs).mockResolvedValue([chair({ icon: '' })] as any)
    const { container } = render(Office, { onClose: () => {} })

    await waitFor(() => expect(container.querySelector('.mascot svg')).toBeTruthy())
    expect(container.querySelector('.mascot .ms-earL .ms-badge')?.innerHTML).toContain('M 116.0,742.5')
  })

  it('says plainly when a chair has never been handed anything', async () => {
    render(Office, { onClose: () => {} })
    await waitFor(() => expect(screen.getByText('ยังไม่เคยรับงาน')).toBeTruthy())
  })

  it('counts the work a chair has done', async () => {
    vi.mocked(ListChairs).mockResolvedValue([chair({ jobs: 3, lastUsed: new Date().toISOString() })] as any)
    const { container } = render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('3')).toBeTruthy())
    // In the card's body since 30 ส.ค., not in the foot. It is a fact about the
    // agent like the sentence above it; the foot is the card's actions, and a
    // number sharing that row is what kept the chat button down to an icon.
    expect(container.querySelector('.chair-stat')?.textContent).toMatch(/3\s*งาน/)
  })

  // The talking room and nothing else (owner, 13 ก.ย. 2026: "หน้านั้นจะเป็น
  // เลือกคุยอย่างเดียว"). Configuring — the gear that was on every card, the
  // hiring button, the folder link — is ตั้งค่า › เอเจนเฉพาะทาง's, and this
  // page keeps one door there. Pinned by absence as much as presence: a gear
  // creeping back onto the card is the regression.
  it('keeps one door to settings and no gear on the cards', async () => {
    const { container } = render(Office, { onClose: () => {} })
    await screen.findByText('เก้าอี้ร่างเอกสาร')

    expect(screen.queryByLabelText('ตั้งค่า')).toBeNull()
    expect(screen.queryByText('เพิ่มเอเจนเฉพาะทาง')).toBeNull()
    expect(container.querySelector('.office-note')).toBeNull()

    await fireEvent.click(screen.getByText('ตั้งค่าเอเจนเฉพาะทาง'))
    expect(cockpit.activeView).toBe('settings')
    expect(sessionStorage.getItem('aetox.settingsSection')).toBe('team')
  })

  // Walking into the room (§85): the card's chat button opens a session bound
  // to that agent, and the view moves to the chat that session now owns.
  // In words, and naming the agent. It was a 13px sparkles icon sharing the
  // foot with the job count until 30 ส.ค. — the smallest thing on the card,
  // wearing a mark that means "chat" to nobody, on a page whose whole purpose
  // is walking in and talking to a specialist. Read by its visible text here on
  // purpose: an aria-label would pass this test with the icon back.
  it('opens a direct chat from a button that says so in words', async () => {
    vi.mocked(NewChairSessionAt).mockResolvedValue('20260805-100000.000' as any)
    const { container } = render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('คุยกับ doc')).toBeTruthy())
    expect(container.querySelector('.chair-talk')?.textContent?.trim()).toBe('คุยกับ doc')
    await fireEvent.click(screen.getByText('คุยกับ doc'))

    // At the office; the engine finds the team that seats the chair.
    await waitFor(() => expect(vi.mocked(NewChairSessionAt).mock.calls[0]).toEqual(['specialized', 'doc', '']))
    expect(cockpit.activeView).toBe('chat')
    expect(cockpit.chair).toBe('doc')
    expect(cockpit.desk).toBe('specialized')
  })
})

// Whether the main assistant may hand each teammate work — the one thing that
// decides if anyone on this page ever gets used, and until 31 ส.ค. the roster
// could not say it. It lived on the settings page as a column of switches over
// a list of rows, which meant the page you open to LOOK at your team and the
// page that decides whether the team works were two different pages.
// Teams touch this page in one way only (§256, and the owner on 12 ก.ย.:
// "คนอยู่หน้าแรก ทีมอยู่ตั้งค่า"): a chip on the card naming the teams an agent
// is on. No switch, no band, no section per team — each person is drawn once,
// and everything about a roster is in ตั้งค่า › ทีมเอเจน.
describe('the roster and teams', () => {
  it('draws each agent once and names the teams that list it', async () => {
    vi.mocked(ListChairs).mockResolvedValue([chair(), chair({ name: 'fixer', builtin: false })] as any)
    vi.mocked(ListTeams).mockImplementation(async () => [
      { name: 'ผู้ช่วยในคอมพิวเตอร์', desk: 'specialized', description: '', invalid: '', missing: [],
        members: [chair()], path: '' },
      { name: 'ทีมโค้ด', desk: 'coding', description: '', invalid: '', missing: [],
        members: [chair(), chair({ name: 'fixer', builtin: false })], path: '' },
    ] as any)
    const { container } = render(Office, { onClose: () => {} })

    await waitFor(() => expect(container.querySelectorAll('.chair-card.agc').length).toBe(2))
    const cards = Array.from(container.querySelectorAll('.chair-card.agc'))
    expect(cards[0].querySelector('.chair-stat.teams')?.textContent).toContain('ผู้ช่วยในคอมพิวเตอร์ · ทีมโค้ด')
    expect(cards[1].querySelector('.chair-stat.teams')?.textContent).toContain('ทีมโค้ด')
    expect(container.querySelector('.mswitch')).toBeNull()
    expect(container.querySelector('.team-sec')).toBeNull()
  })

  it('still draws the roster when the teams cannot be read', async () => {
    vi.mocked(ListTeams).mockRejectedValue(new Error('unavailable'))
    const { container } = render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('เก้าอี้ร่างเอกสาร')).toBeTruthy())
    expect(container.querySelector('.chair-stat.teams')).toBeNull()
  })

  it('has nothing of its own about teams beyond the chip', async () => {
    render(Office, { onClose: () => {} })
    await waitFor(() => expect(screen.getByText('เก้าอี้ร่างเอกสาร')).toBeTruthy())
    expect(screen.queryByText(/จัดทีมที่/)).toBeNull()
    expect(screen.queryByText('สร้างทีม')).toBeNull()
  })
})

describe('the received-work feed', () => {
  it('shows what came in, from whom, and what it cost', async () => {
    vi.mocked(ListReceivedJobs).mockResolvedValue([job()] as any)
    render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('ทำเอกสารสรุปยอดเดือนนี้')).toBeTruthy())
    expect(screen.getByText('เรียกเครื่องมือ 2 ครั้ง')).toBeTruthy()
    expect(screen.getByText('4.2s')).toBeTruthy()
  })

  // The job row carries the caller's session id, and that is the only link
  // between a delivered file and the conversation that asked for it. The whole
  // row is the door — a boxed button repeated down the right edge was the
  // loudest thing on a page whose subject is the line beside it.
  //
  // Since §158 that walk crosses a door: the caller is usually the assistant
  // and this room is behind ทีม. Nothing here has to know that — loading a
  // session re-reads its desk and the door follows (refreshDesk).
  it('walks back to the chat that sent the job', async () => {
    vi.mocked(ListReceivedJobs).mockResolvedValue([job()] as any)
    vi.mocked(LoadSessionAnyProject).mockResolvedValue([] as any)
    render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByLabelText('ไปที่แชท')).toBeTruthy())
    await fireEvent.click(screen.getByLabelText('ไปที่แชท'))

    await waitFor(() => expect(vi.mocked(LoadSessionAnyProject).mock.calls[0][0]).toBe('20260805-090000.000'))
    expect(cockpit.activeView).toBe('chat')
  })

  // A duration is only worth a slot when it says something. Every row printing
  // "0.0s" was six copies of "this was instant" competing with the line that
  // says what the job actually was.
  it('leaves out a duration too small to mean anything', async () => {
    vi.mocked(ListReceivedJobs).mockResolvedValue([job({ durationMs: 40 })] as any)
    render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('ทำเอกสารสรุปยอดเดือนนี้')).toBeTruthy())
    expect(screen.queryByText('0.0s')).toBeNull()
  })

  // Grouped by calendar day, so the eye can skip a day it does not want —
  // rather than reading "2 วัน" printed once per row all the way down.
  it('files the feed under the day it came in', async () => {
    vi.mocked(ListReceivedJobs).mockResolvedValue([job()] as any)
    render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('วันนี้')).toBeTruthy())
  })

  // The filter is the question this list is asked once more than one teammate
  // has worked, and furniture before that.
  it('filters the feed by teammate', async () => {
    vi.mocked(ListReceivedJobs).mockResolvedValue([
      job(), job({ id: 2, chair: 'sheet', brief: 'รวมยอดค่าใช้จ่าย' }),
    ] as any)
    render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('ทั้งหมด')).toBeTruthy())
    await fireEvent.click(screen.getByText('sheet'))

    expect(screen.getByText('รวมยอดค่าใช้จ่าย')).toBeTruthy()
    expect(screen.queryByText('ทำเอกสารสรุปยอดเดือนนี้')).toBeNull()
  })

  it('does not draw a filter when one teammate is the whole feed', async () => {
    vi.mocked(ListReceivedJobs).mockResolvedValue([job()] as any)
    render(Office, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('ทำเอกสารสรุปยอดเดือนนี้')).toBeTruthy())
    expect(screen.queryByText('ทั้งหมด')).toBeNull()
  })

  it('has an empty state that says what to do rather than nothing', async () => {
    render(Office, { onClose: () => {} })
    await waitFor(() => expect(screen.getByText(/ยังไม่มีงานส่งเข้ามา/)).toBeTruthy())
  })
})

// Teams (§256): the page is one section per roster. A user team draws its
// own members under its own head, its chat door seats the agent at the
// team's desk, and the editor writes through the one door the engine has.
