// The two station chips at the composer's foot (§256, 13 ก.ย.): WHO answers,
// and which TEAM the chat hires from. What is pinned: the two chips are two
// questions — the team's name never replaces the assistant's; the assistant
// wears the desk's icon (the code page's own, not the spark); "ไม่มีทีม" is a
// visible state and still a door; the team chip is not drawn behind a chair;
// the WHO menu lists the current team's people and the TEAM menu lists the
// desk's teams with the door's switch under the current one; picking anything
// opens a NEW session; and "จัดการทีม" lands ตั้งค่า › ทีมเอเจน on this
// desk's side.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import StationPick from '../lib/StationPick.svelte'
import { ListTeams, DelegateSwitches, SetDelegateOff, AgentBlocked, NewTeamSession, NewChairSessionAt, SessionTeam } from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { ICONS } from '../lib/icons'

const chair = (over: Record<string, unknown> = {}) => ({
  name: 'doc', description: 'เก้าอี้ร่างเอกสาร', tools: [], builtin: true, jobs: 0, lastUsed: '', ...over,
})
const team = (over: Record<string, unknown> = {}) => ({
  name: 'ผู้ช่วยในคอมพิวเตอร์', desk: 'specialized', description: 'ทีมที่แอปตั้งให้', invalid: '', missing: [],
  members: [chair(), chair({ name: 'sheet' })], path: 'C:/teams/ผู้ช่วยในคอมพิวเตอร์/TEAM.md', ...over,
})
const switches = (over: { agents?: boolean; code?: boolean; on?: string[] } = {}) => ({
  team: 'ผู้ช่วยในคอมพิวเตอร์',
  agents: { off: over.agents ?? false, tokens: 90, workers: [{ name: 'doc', for: '', agent: true, on: (over.on ?? ['doc']).includes('doc') }, { name: 'sheet', for: '', agent: true, on: (over.on ?? ['doc']).includes('sheet') }] },
  code: { off: over.code ?? false, tokens: 40, workers: [] },
  helpers: { off: false, tokens: 0, workers: [] }, tokens: 0,
})
const who = (c: HTMLElement) => c.querySelector('.station-who .focus-chip') as HTMLButtonElement
const teamChip = (c: HTMLElement) => c.querySelector('.station-team .focus-chip') as HTMLButtonElement
const menu = (c: HTMLElement, which: 'who' | 'team') => c.querySelector(`.station-${which} .focus-menu`) as HTMLElement
// Icon.svelte inlines the path, so an icon is told apart by its path.
const wears = (el: HTMLElement, icon: keyof typeof ICONS) => el.innerHTML.includes(ICONS[icon].slice(0, 40))

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.desk = 'specialized'
  cockpit.chair = ''
  cockpit.team = 'ผู้ช่วยในคอมพิวเตอร์'
  cockpit.settingsIntent = null
  vi.mocked(ListTeams).mockImplementation(async () => [team()] as any)
  vi.mocked(DelegateSwitches).mockImplementation(async () => switches() as any)
  vi.mocked(AgentBlocked).mockResolvedValue(false as any)
  vi.mocked(SessionTeam).mockResolvedValue('ผู้ช่วยในคอมพิวเตอร์' as any)
})

describe('station chips', () => {
  it("draws two chips: the assistant with the assistant head's face and name, and the team by name", async () => {
    const { container } = render(StationPick)

    expect(who(container).textContent).toContain('ผู้ช่วย')
    expect(who(container).textContent).not.toContain('ผู้ช่วยในคอมพิวเตอร์')
    expect(teamChip(container).textContent).toContain('ผู้ช่วยในคอมพิวเตอร์')
    expect(container.querySelector('.station-team.none')).toBeNull()
    // The desk's head, as a face (14 ก.ย. 2026), not the door's icon: the
    // assistant wears the orb, and nothing of the coder's chevrons.
    const face = who(container).querySelector('.mascot')!
    expect(face).toBeTruthy()
    expect(face.innerHTML).not.toContain('ms-chev')
    expect(wears(who(container), 'sparkles')).toBe(false)
  })

  it("wears the code head's face AND name on the coding desk — the avatar page's card name, not the assistant's", async () => {
    cockpit.desk = 'coding'
    cockpit.team = ''
    const { container } = render(StationPick)

    expect(who(container).textContent).toContain('โค้ด')
    expect(who(container).textContent).not.toContain('ผู้ช่วย')
    const face = who(container).querySelector('.mascot')!
    expect(face.innerHTML).toContain('ms-chev')
    expect(wears(who(container), 'fileCode')).toBe(false)
  })

  it('says "ยังไม่มีทีม" on the team chip, dimmed, and still opens a menu that says where to make one', async () => {
    cockpit.desk = 'coding'
    cockpit.team = ''
    vi.mocked(ListTeams).mockImplementation(async () => [] as any)
    const { container } = render(StationPick)

    expect(teamChip(container).textContent).toContain('ยังไม่มีทีมช่วย')
    expect(container.querySelector('.station-team.none')).toBeTruthy()
    await fireEvent.click(teamChip(container))
    await waitFor(() => expect(menu(container, 'team')).toBeTruthy())
    expect(menu(container, 'team').textContent).toContain('ยังไม่มีทีมฝั่งโค้ด')
    expect(screen.getByText('จัดการทีม')).toBeTruthy()
    // No team is the roster this chat is on: its row is lit.
    expect(menu(container, 'team').querySelector('.team-row.no-team.on')).toBeTruthy()
  })

  it('TEAM menu: no team is a row too — picking it opens a chat that hires nobody', async () => {
    vi.mocked(NewTeamSession).mockResolvedValue('20260913-100003.000' as any)
    const { container } = render(StationPick)

    await fireEvent.click(teamChip(container))
    await waitFor(() => expect(menu(container, 'team').querySelector('.team-row.no-team')).toBeTruthy())
    const row = menu(container, 'team').querySelector('.team-row.no-team') as HTMLElement
    expect(row.classList.contains('on')).toBe(false) // this chat is on ผู้ช่วยในคอมพิวเตอร์
    expect(row.textContent).toContain('ไม่ใช้ทีมช่วย')
    await fireEvent.click(row.querySelector('.focus-item') as HTMLElement)
    await waitFor(() => expect(vi.mocked(NewTeamSession)).toHaveBeenCalledWith('specialized', ''))
  })

  it('hides the team chip behind a chair, and wears the chair’s face', async () => {
    cockpit.chair = 'doc'
    const { container } = render(StationPick)

    expect(who(container).textContent).toContain('doc')
    expect(teamChip(container)).toBeNull()
    // The face lands once the team file has been read.
    await waitFor(() => expect(who(container).querySelector('.ic .mascot, .ic svg')).toBeTruthy())
  })

  it('WHO menu: the assistant, then the current team’s people; picking one opens a chair session on this team', async () => {
    vi.mocked(NewChairSessionAt).mockResolvedValue('20260913-100000.000' as any)
    const { container } = render(StationPick)

    await fireEvent.click(who(container))
    await waitFor(() => expect(menu(container, 'who').querySelectorAll('.agent-row').length).toBe(2))
    const m = menu(container, 'who')
    expect(m.querySelector('.focus-item.on')?.textContent).toContain('ผู้ช่วย')
    expect(m.textContent).not.toContain('ผู้ช่วยในคอมพิวเตอร์') // people only — no team rows here
    expect(m.querySelector('.delegate-row')).toBeNull()
    await fireEvent.click(screen.getByText('sheet'))
    await waitFor(() => expect(vi.mocked(NewChairSessionAt).mock.calls[0]).toEqual(['specialized', 'sheet', 'ผู้ช่วยในคอมพิวเตอร์']))
    expect(menu(container, 'who')).toBeNull() // closed on pick
  })

  it('WHO menu: back to the assistant from a chair opens a team session', async () => {
    cockpit.chair = 'doc'
    vi.mocked(NewTeamSession).mockResolvedValue('20260913-100001.000' as any)
    const { container } = render(StationPick)

    await fireEvent.click(who(container))
    await waitFor(() => expect(menu(container, 'who')).toBeTruthy())
    await fireEvent.click(screen.getByText('ผู้ช่วย', { selector: '.focus-item' }))
    await waitFor(() => expect(vi.mocked(NewTeamSession)).toHaveBeenCalledWith('specialized', 'ผู้ช่วยในคอมพิวเตอร์'))
  })

  it('TEAM menu: the desk’s teams with the reach beside the current one and the door’s switch under it', async () => {
    vi.mocked(ListTeams).mockImplementation(async () => [team(), team({ name: 'ทีมร้าน', members: [chair()] })] as any)
    vi.mocked(SetDelegateOff).mockResolvedValue(switches({ agents: true }) as any)
    const { container } = render(StationPick)

    await fireEvent.click(teamChip(container))
    await waitFor(() => expect(menu(container, 'team').querySelectorAll('.team-row:not(.no-team)').length).toBe(2))
    const rows = Array.from(menu(container, 'team').querySelectorAll('.team-row:not(.no-team)'))
    expect(rows[0].classList.contains('on')).toBe(true)
    await waitFor(() => expect(rows[0].textContent).toContain('มอบงานได้ 1/2'))
    expect(rows[1].textContent).toContain('ทีมร้าน')
    expect(rows[1].textContent).toContain('1')
    // The door's switch, on, under the current team only; flipping asks the
    // assistant door (this is the assistant desk).
    const sw = menu(container, 'team').querySelector('.delegate-row') as HTMLButtonElement
    expect(sw.getAttribute('aria-checked')).toBe('true')
    expect(menu(container, 'team').textContent).toContain('90 token')
    await fireEvent.click(sw)
    await waitFor(() => expect(vi.mocked(SetDelegateOff)).toHaveBeenCalledWith('agents', true))
    await waitFor(() => expect(sw.getAttribute('aria-checked')).toBe('false'))
    expect(menu(container, 'team')).toBeTruthy() // a switch is not a pick: the menu stays
  })

  it('TEAM menu: picking another team opens a session hiring from it; the current one is a no-op', async () => {
    vi.mocked(ListTeams).mockImplementation(async () => [team(), team({ name: 'ทีมร้าน', members: [chair()] })] as any)
    vi.mocked(NewTeamSession).mockResolvedValue('20260913-100002.000' as any)
    const { container } = render(StationPick)

    await fireEvent.click(teamChip(container))
    await waitFor(() => expect(menu(container, 'team').querySelectorAll('.team-row:not(.no-team)').length).toBe(2))
    await fireEvent.click(screen.getByText('ผู้ช่วยในคอมพิวเตอร์', { selector: '.team-row .focus-item .t' }))
    expect(vi.mocked(NewTeamSession)).not.toHaveBeenCalled()
    await fireEvent.click(teamChip(container))
    await waitFor(() => expect(menu(container, 'team').querySelectorAll('.team-row:not(.no-team)').length).toBe(2))
    await fireEvent.click(screen.getByText('ทีมร้าน', { selector: '.team-row .focus-item .t' }))
    await waitFor(() => expect(vi.mocked(NewTeamSession)).toHaveBeenCalledWith('specialized', 'ทีมร้าน'))
  })

  it('the code page asks the code door, and "จัดการทีม" lands on ฝั่งโค้ด', async () => {
    cockpit.desk = 'coding'
    cockpit.team = 'ทีมโค้ด'
    vi.mocked(ListTeams).mockImplementation(async () => [team({ name: 'ทีมโค้ด', desk: 'coding', members: [chair({ name: 'fixer' })] })] as any)
    vi.mocked(DelegateSwitches).mockImplementation(async () => ({ ...switches({ code: true }), team: 'ทีมโค้ด' }) as any)
    const { container } = render(StationPick)

    await fireEvent.click(teamChip(container))
    await waitFor(() => expect(menu(container, 'team').querySelector('.delegate-row')).toBeTruthy())
    const sw = menu(container, 'team').querySelector('.delegate-row') as HTMLButtonElement
    expect(sw.textContent).toContain('โค้ดมอบงานให้ทีม')
    expect(sw.getAttribute('aria-checked')).toBe('false')
    await fireEvent.click(screen.getByText('จัดการทีม'))
    expect(cockpit.settingsIntent).toEqual({ section: 'teams', side: 'coding' })
    expect(cockpit.activeView).toBe('settings')
  })

  it('one menu at a time, and a press outside closes it', async () => {
    const { container } = render(StationPick)

    await fireEvent.click(who(container))
    await waitFor(() => expect(menu(container, 'who')).toBeTruthy())
    await fireEvent.click(teamChip(container))
    await waitFor(() => expect(menu(container, 'team')).toBeTruthy())
    expect(menu(container, 'who')).toBeNull()
    await fireEvent.mouseDown(document.body)
    await waitFor(() => expect(menu(container, 'team')).toBeNull())
  })
})
