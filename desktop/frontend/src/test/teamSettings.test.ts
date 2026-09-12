// Settings › ทีมเอเจน (§256): the one home of a roster, in การตั้งค่าโมเดล's
// master–detail shape — teams in the rail split by side, one roster in the
// pane. What is pinned: the two DOOR switches sit above everything and are
// the only master switches there are; the rail names every team under its
// side with its reach tally; the pane draws the picked roster with its
// members' switches; every team — the seeded ทีมเอเจน included — is editable
// and deletable; the door to a new team is visible on both sides; the editor
// writes through the engine's one door with exactly what was ticked; and the
// roster page's intents land on the right team.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import TeamSettings from '../lib/TeamSettings.svelte'
import { ListChairs, ListTeams, SaveTeam, DeleteTeam, DelegateSwitches, SetAgentOff, SetDelegateOff } from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

const chair = (over: Record<string, unknown> = {}) => ({
  name: 'doc', description: 'เก้าอี้ร่างเอกสาร', tools: [], builtin: true, jobs: 0, lastUsed: '', ...over,
})
const seed = () => ({
  name: 'ทีมเอเจน', desk: 'specialized', description: 'ทีมที่แอปตั้งให้ตอนติดตั้ง แก้หรือลบได้', invalid: '',
  missing: [], members: [chair()], path: 'C:/teams/ทีมเอเจน/TEAM.md',
})
const codeTeam = (over: Record<string, unknown> = {}) => ({
  name: 'ทีมโค้ด', desk: 'coding', description: 'แก้โค้ด', invalid: '',
  missing: [], members: [chair({ name: 'fixer', builtin: false })], path: 'C:/teams/ทีมโค้ด/TEAM.md', ...over,
})
const switches = (team: string, workers: { name: string; on: boolean }[], doors = { agents: false, code: false }) => ({
  team,
  agents: { off: doors.agents, tokens: 0, workers: workers.map((w) => ({ ...w, for: '', agent: true })) },
  code: { off: doors.code, tokens: 0, workers: [] },
  helpers: { off: false, tokens: 0, workers: [] }, tokens: 0,
})
const rail = (container: HTMLElement) => Array.from(container.querySelectorAll('.mset-side .team-row'))
const pane = (container: HTMLElement) => container.querySelector('.mset-detail') as HTMLElement
const doorSwitch = (container: HTMLElement, side: 'side-assistant' | 'side-code') =>
  container.querySelector(`.team-sides .${side} .mswitch input`) as HTMLInputElement

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.settingsIntent = null
  vi.mocked(ListChairs).mockResolvedValue([chair(), chair({ name: 'fixer', builtin: false })] as any)
  vi.mocked(ListTeams).mockImplementation(async () => [seed(), codeTeam()] as any)
  vi.mocked(DelegateSwitches).mockImplementation(async (name: string) =>
    switches(name, name === 'ทีมเอเจน' ? [{ name: 'doc', on: true }] : [{ name: 'fixer', on: false }]) as any)
})

describe('Settings › ทีมเอเจน', () => {
  it('draws the two door switches above everything, and one team under each side', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(2))
    expect(doorSwitch(container, 'side-assistant')).toBeTruthy()
    expect(doorSwitch(container, 'side-code')).toBeTruthy()
    // The rail: each side's header, its team with the reach tally, its door.
    expect(container.querySelector('.mset-side .team-side.side-assistant')).toBeTruthy()
    expect(container.querySelector('.mset-side .team-side.side-code')).toBeTruthy()
    expect(rail(container)[0].textContent).toContain('ทีมเอเจน')
    await waitFor(() => expect(rail(container)[0].textContent).toContain('1/1'))
    expect(rail(container)[1].textContent).toContain('ทีมโค้ด')
    expect(rail(container)[1].textContent).toContain('0/1')
    expect(container.querySelectorAll('.mset-side .team-add').length).toBe(2)
    // Opens on the first team; the seed is a team like any other — a gear,
    // and no master switch of its own in the pane.
    expect(rail(container)[0].classList.contains('selected')).toBe(true)
    expect(pane(container).querySelector('.mset-name')?.textContent).toBe('ทีมเอเจน')
    expect(pane(container).querySelector('.desk-badge.side-assistant')?.textContent).toContain('ฝั่งผู้ช่วย')
    expect(screen.getByText('แก้ไขทีม', { selector: 'button' })).toBeTruthy()
    expect(pane(container).querySelector('.team-reach')).toBeNull()
    await waitFor(() => expect(pane(container).querySelectorAll('.team-member').length).toBe(1))
    expect((pane(container).querySelector('.team-member .mswitch input') as HTMLInputElement).checked).toBe(true)
  })

  it('says a side has no team yet, and still offers its door', async () => {
    vi.mocked(ListTeams).mockImplementation(async () => [seed()] as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(1))
    expect(container.querySelector('.mset-side .team-side-empty')).toBeTruthy()
    expect(container.querySelectorAll('.mset-side .team-add').length).toBe(2)
    expect(container.querySelector('.team-title .team-new')).toBeTruthy()
    // The nudge, while the seed is the only team (drawn once the switches
    // have loaded, a tick after the rail).
    await waitFor(() => expect(container.querySelector('.team-callout')).toBeTruthy())
  })

  it("flips a door's switch by side, and the rows under that side cool", async () => {
    vi.mocked(SetDelegateOff).mockResolvedValue(switches('ทีมโค้ด', [{ name: 'fixer', on: false }], { agents: false, code: true }) as any)
    const { container } = render(TeamSettings)

    // Loaded through — the tally is read off the switches, so its presence
    // says the page holds them and a flip has something to update.
    await waitFor(() => expect(rail(container)[1]?.textContent).toContain('0/1'))
    await fireEvent.click(doorSwitch(container, 'side-code'))
    await waitFor(() => expect(vi.mocked(SetDelegateOff)).toHaveBeenCalledWith('code', true))
    // The answer lands a tick later; the switch reads it back before the rail
    // is asked to show the cooled roster.
    await waitFor(() => expect(doorSwitch(container, 'side-code').checked).toBe(false))
    await fireEvent.click(rail(container)[1])
    await waitFor(() => expect(pane(container).querySelector('.team-members.cool')).toBeTruthy())
    // The assistant's door was not touched.
    expect(doorSwitch(container, 'side-assistant').checked).toBe(true)
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

  it("opens a code-side door with the code desk already chosen", async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(container.querySelectorAll('.mset-side .team-add').length).toBe(2))
    await fireEvent.click(container.querySelectorAll('.mset-side .team-add')[1])
    await waitFor(() => expect(pane(container).querySelector('.team-desk-pick .seg-btn.side-code.selected')).toBeTruthy())
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

  it('deletes the seeded team too, through a confirm that says the people stay', async () => {
    cockpit.settingsIntent = { section: 'teams', team: 'ทีมเอเจน' }
    render(TeamSettings)

    await waitFor(() => expect(screen.getByText('ลบทีม', { selector: 'button.ctrl-danger' })).toBeTruthy())
    await fireEvent.click(screen.getByText('ลบทีม', { selector: 'button.ctrl-danger' }))
    expect(screen.getByText(/เอเจนในทีมยังอยู่ครบ/)).toBeTruthy()
    expect(vi.mocked(DeleteTeam)).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByText('ลบทีม', { selector: '.confirm-go' }))
    await waitFor(() => expect(vi.mocked(DeleteTeam)).toHaveBeenCalledWith('ทีมเอเจน'))
  })
})
