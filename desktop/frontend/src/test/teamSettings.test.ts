// Settings › ทีมเอเจน (§256): the one home of a roster, in การตั้งค่าโมเดล's
// master–detail shape — teams in the rail, one roster in the pane. What is
// pinned: the rail names every team with its reach tally, the pane draws the
// picked roster with its switch and its members' switches, the door to a new
// team is visible three ways, the editor writes through the engine's one
// door with exactly what was ticked, and the roster page's intents land on
// the right team.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import TeamSettings from '../lib/TeamSettings.svelte'
import { ListChairs, ListTeams, SaveTeam, DeleteTeam, DelegateSwitches, SetAgentOff, SetDelegateOff } from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

const chair = (over: Record<string, unknown> = {}) => ({
  name: 'doc', description: 'เก้าอี้ร่างเอกสาร', tools: [], builtin: true, jobs: 0, lastUsed: '', ...over,
})
const team = (over: Record<string, unknown> = {}) => ({
  name: 'ทีมโค้ด', desk: 'coding', description: 'แก้โค้ด', default: false, invalid: '',
  missing: [], members: [chair({ name: 'fixer', builtin: false })], path: 'C:/teams/ทีมโค้ด/TEAM.md',
  delegateOff: false, ...over,
})
const defaultTeam = () => ({
  name: '', desk: 'specialized', description: '', default: true, invalid: '', missing: [],
  members: [chair()], path: '', delegateOff: false,
})
const switches = (team: string, workers: { name: string; on: boolean }[], off = false) => ({
  team, agents: { off, tokens: 0, workers: workers.map((w) => ({ ...w, for: '', agent: true })) },
  helpers: { off: false, tokens: 0, workers: [] }, tokens: 0,
})
const rail = (container: HTMLElement) => Array.from(container.querySelectorAll('.mset-side .team-row'))
const pane = (container: HTMLElement) => container.querySelector('.mset-detail') as HTMLElement

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.settingsIntent = null
  vi.mocked(ListChairs).mockResolvedValue([chair(), chair({ name: 'fixer', builtin: false })] as any)
  vi.mocked(ListTeams).mockImplementation(async () => [defaultTeam(), team()] as any)
  vi.mocked(DelegateSwitches).mockImplementation(async (name: string) =>
    switches(name, name === '' ? [{ name: 'doc', on: true }] : [{ name: 'fixer', on: false }]) as any)
})

describe('Settings › ทีมเอเจน', () => {
  it('lists every team in the rail with its reach tally, and opens on the shipped team', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(2))
    expect(rail(container)[0].textContent).toContain('ทีมเอเจน')
    // The tally lands a tick after the rail: it is read off the switches.
    await waitFor(() => expect(rail(container)[0].textContent).toContain('1/1'))
    expect(rail(container)[1].textContent).toContain('ทีมโค้ด')
    expect(rail(container)[1].textContent).toContain('0/1')
    expect(rail(container)[0].classList.contains('selected')).toBe(true)
    // The pane: the shipped team, its desk, its note, its member with the
    // member's own switch — and no gear, because it cannot be edited.
    expect(pane(container).querySelector('.mset-name')?.textContent).toBe('ทีมเอเจน')
    expect(pane(container).textContent).toContain('มากับแอป')
    expect(pane(container).querySelector('.desk-badge.side-assistant')?.textContent).toContain('ฝั่งผู้ช่วย')
    expect(pane(container).textContent).toContain('แก้สมาชิกไม่ได้')
    await waitFor(() => expect(pane(container).querySelectorAll('.team-member').length).toBe(1))
    expect((pane(container).querySelector('.team-member .mswitch input') as HTMLInputElement).checked).toBe(true)
    expect(screen.queryByText('แก้ไขทีม', { selector: 'button' })).toBeNull()
  })

  it('shows the door to a new team three ways while the user has none, and once they do, two', async () => {
    vi.mocked(ListTeams).mockImplementation(async () => [defaultTeam()] as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(container.querySelector('.team-callout')).toBeTruthy())
    expect(container.querySelector('.team-title .team-new')).toBeTruthy()
    // One door per side, so the code side reads as a place a team can be
    // made even while it is empty (owner: 'แยกชัดๆ อันไหนฝั่งผู้ช่วย อันไหนฝั่งโค้ด').
    expect(container.querySelectorAll('.mset-side .team-add').length).toBe(2)
    expect(container.querySelector('.mset-side .team-side.side-code')).toBeTruthy()
    expect(container.querySelector('.mset-side .team-side-empty')).toBeTruthy()
    expect(screen.getByText(/อยากได้ทีมของคุณเอง/)).toBeTruthy()

    vi.mocked(ListTeams).mockImplementation(async () => [defaultTeam(), team()] as any)
    const second = render(TeamSettings)
    await waitFor(() => expect(rail(second.container).length).toBe(2))
    expect(second.container.querySelector('.team-callout')).toBeNull()
  })

  it('picks a team from the rail and draws that roster, gear included', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(2))
    await fireEvent.click(rail(container)[1])
    await waitFor(() => expect(pane(container).querySelector('.mset-name')?.textContent).toBe('ทีมโค้ด'))
    expect(pane(container).querySelector('.desk-badge.side-code')?.textContent).toContain('ฝั่งโค้ด')
    expect(pane(container).textContent).toContain('แก้โค้ด')
    expect(pane(container).querySelector('.team-member')?.textContent).toContain('fixer')
    // fixer is off on ทีมโค้ด: its row cools and its switch is unticked.
    expect((pane(container).querySelector('.team-member .mswitch input') as HTMLInputElement).checked).toBe(false)
    expect(pane(container).querySelector('.team-member.off')).toBeTruthy()
    expect(screen.getByText('แก้ไขทีม', { selector: 'button' })).toBeTruthy()
  })

  it("flips a member's reach on the team it sits under, not everywhere", async () => {
    vi.mocked(SetAgentOff).mockResolvedValue(switches('ทีมโค้ด', [{ name: 'fixer', on: true }]) as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(2))
    await fireEvent.click(rail(container)[1])
    await waitFor(() => expect(pane(container).querySelector('.team-member .mswitch input')).toBeTruthy())
    await fireEvent.click(pane(container).querySelector('.team-member .mswitch input') as HTMLInputElement)
    await waitFor(() => expect(vi.mocked(SetAgentOff)).toHaveBeenCalledWith('ทีมโค้ด', 'fixer', false))
  })

  it("flips the team's own delegation switch by name", async () => {
    vi.mocked(SetDelegateOff).mockResolvedValue(switches('ทีมโค้ด', [{ name: 'fixer', on: false }], true) as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(2))
    await fireEvent.click(rail(container)[1])
    await waitFor(() => expect(pane(container).querySelector('.team-reach .mswitch input')).toBeTruthy())
    await fireEvent.click(pane(container).querySelector('.team-reach .mswitch input') as HTMLInputElement)
    await waitFor(() => expect(vi.mocked(SetDelegateOff)).toHaveBeenCalledWith('ทีมโค้ด', 'agents', true))
  })

  it('saves a new team through the engine door with what was ticked', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(container.querySelector('.team-title .team-new')).toBeTruthy())
    await fireEvent.click(container.querySelector('.team-title .team-new') as HTMLElement)
    // The house form: mset-field + .ctrl, the seg-ctrl choice, conn-chip ticks —
    // nothing of this page's own (owner: "ดูมาตรฐานหน้าอื่นครับ").
    const name = pane(container).querySelector('.mset-field input.ctrl') as HTMLInputElement
    await fireEvent.input(name, { target: { value: 'ทีมเอกสาร' } })
    await fireEvent.click(screen.getByText('โค้ด', { selector: '.seg-ctrl .seg-btn' }))
    // The coding desk says what it hands the team, before anybody saves.
    expect(screen.getByText(/ถือเชลล์/)).toBeTruthy()
    await fireEvent.click(pane(container).querySelector('.conn-targets .conn-chip') as HTMLElement)
    await fireEvent.click(screen.getByText('บันทึกทีม'))

    await waitFor(() => expect(vi.mocked(SaveTeam).mock.calls[0]).toEqual(['ทีมเอกสาร', 'coding', '', ['doc']]))
  })

  it('opens on the team the roster page pointed at, in its editor, with its members ticked', async () => {
    cockpit.settingsIntent = { section: 'teams', team: 'ทีมโค้ด' }
    const { container } = render(TeamSettings)

    await waitFor(() => expect(pane(container).querySelector('.mset-name')?.textContent).toBe('แก้ไขทีม'))
    const name = pane(container).querySelector('.mset-field input.ctrl') as HTMLInputElement
    expect(name.value).toBe('ทีมโค้ด')
    expect(name.disabled).toBe(true) // the name is the folder
    const ticked = Array.from(pane(container).querySelectorAll('.conn-chip.on')).map((n) => n.textContent?.trim())
    expect(ticked).toEqual(['fixer'])
    expect(cockpit.settingsIntent).toBeNull() // consumed once
  })

  it('deletes through a confirm that says the people stay', async () => {
    cockpit.settingsIntent = { section: 'teams', team: 'ทีมโค้ด' }
    render(TeamSettings)

    await waitFor(() => expect(screen.getByText('ลบทีม', { selector: 'button.ctrl-danger' })).toBeTruthy())
    await fireEvent.click(screen.getByText('ลบทีม', { selector: 'button.ctrl-danger' }))
    expect(screen.getByText(/เอเจนในทีมยังอยู่ครบ/)).toBeTruthy()
    expect(vi.mocked(DeleteTeam)).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByText('ลบทีม', { selector: '.confirm-go' }))
    await waitFor(() => expect(vi.mocked(DeleteTeam)).toHaveBeenCalledWith('ทีมโค้ด'))
  })
})
