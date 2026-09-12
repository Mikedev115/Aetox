// Settings › ทีม (§256): the one place a team is made and edited, beside the
// agent editor and never inside it. What is pinned: the list draws every
// team with its members and its own switches, the editor writes through the
// engine's one door with exactly what was ticked, and the roster page's
// intents land on the right team.
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
const switches = (team: string, workers: { name: string; on: boolean }[], off = false) => ({
  team, agents: { off, tokens: 0, workers: workers.map((w) => ({ ...w, for: '', agent: true })) },
  helpers: { off: false, tokens: 0, workers: [] }, tokens: 0,
})

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.settingsIntent = null
  vi.mocked(ListChairs).mockResolvedValue([chair(), chair({ name: 'fixer', builtin: false })] as any)
  vi.mocked(ListTeams).mockImplementation(async () => [
    { name: '', desk: 'specialized', description: '', default: true, invalid: '', missing: [],
      members: [chair()], path: '', delegateOff: false },
    team(),
  ] as any)
  vi.mocked(DelegateSwitches).mockImplementation(async (name: string) =>
    switches(name, name === '' ? [{ name: 'doc', on: true }] : [{ name: 'fixer', on: false }]) as any)
})

describe('Settings › ทีม', () => {
  it('draws every team with its members and each member\'s reach on that team', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(container.querySelectorAll('.team-card').length).toBe(2))
    const cards = container.querySelectorAll('.team-card')
    expect(cards[0].textContent).toContain('ทีมผู้ช่วย')
    expect(cards[0].textContent).toContain('doc')
    expect(cards[1].textContent).toContain('ทีมโค้ด')
    expect(cards[1].textContent).toContain('โค้ด')
    expect(cards[1].textContent).toContain('fixer')
    // The switches are the team's: fixer is off on ทีมโค้ด, doc on on the default.
    await waitFor(() => expect(cards[1].querySelectorAll('.team-member .mswitch input').length).toBe(1))
    expect((cards[1].querySelector('.team-member .mswitch input') as HTMLInputElement).checked).toBe(false)
    expect((cards[0].querySelector('.team-member .mswitch input') as HTMLInputElement).checked).toBe(true)
    // The default team cannot be edited; a user team can.
    expect(cards[0].querySelector('[aria-label="แก้ไขทีม"]')).toBeNull()
    expect(cards[1].querySelector('[aria-label="แก้ไขทีม"]')).toBeTruthy()
  })

  it('flips a member\'s reach on the team it sits under, not everywhere', async () => {
    vi.mocked(SetAgentOff).mockResolvedValue(switches('ทีมโค้ด', [{ name: 'fixer', on: true }]) as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(container.querySelectorAll('.team-card')[1]?.querySelector('.team-member .mswitch input')).toBeTruthy())
    const input = container.querySelectorAll('.team-card')[1].querySelector('.team-member .mswitch input') as HTMLInputElement
    await fireEvent.click(input)
    await waitFor(() => expect(vi.mocked(SetAgentOff)).toHaveBeenCalledWith('ทีมโค้ด', 'fixer', false))
  })

  it('flips the team\'s own delegation switch by name', async () => {
    vi.mocked(SetDelegateOff).mockResolvedValue(switches('ทีมโค้ด', [{ name: 'fixer', on: false }], true) as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(container.querySelectorAll('.team-card')[1]?.querySelector('.team-card-head .mswitch input')).toBeTruthy())
    const input = container.querySelectorAll('.team-card')[1].querySelector('.team-card-head .mswitch input') as HTMLInputElement
    await fireEvent.click(input)
    await waitFor(() => expect(vi.mocked(SetDelegateOff)).toHaveBeenCalledWith('ทีมโค้ด', 'agents', true))
  })

  it('saves a new team through the engine door with what was ticked', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(screen.getByText('สร้างทีม')).toBeTruthy())
    await fireEvent.click(screen.getByText('สร้างทีม'))
    // The house form: mset-field + .ctrl, the seg-ctrl choice, conn-chip ticks —
    // nothing of this page's own (owner: "ดูมาตรฐานหน้าอื่นครับ").
    const name = container.querySelector('.mset-field input.ctrl') as HTMLInputElement
    await fireEvent.input(name, { target: { value: 'ทีมเอกสาร' } })
    await fireEvent.click(screen.getByText('โค้ด', { selector: '.seg-ctrl .seg-btn' }))
    // The coding desk says what it hands the team, before anybody saves.
    expect(screen.getByText(/ถือเชลล์/)).toBeTruthy()
    await fireEvent.click(container.querySelector('.conn-targets .conn-chip') as HTMLElement)
    await fireEvent.click(screen.getByText('บันทึกทีม'))

    await waitFor(() => expect(vi.mocked(SaveTeam).mock.calls[0]).toEqual(['ทีมเอกสาร', 'coding', '', ['doc']]))
  })

  it('opens on the team the roster page pointed at, with its members ticked', async () => {
    cockpit.settingsIntent = { section: 'teams', team: 'ทีมโค้ด' }
    const { container } = render(TeamSettings)

    await waitFor(() => expect(screen.getByText('แก้ไขทีม', { selector: 'h2' })).toBeTruthy())
    const name = container.querySelector('.mset-field input.ctrl') as HTMLInputElement
    expect(name.value).toBe('ทีมโค้ด')
    expect(name.disabled).toBe(true) // the name is the folder
    const ticked = Array.from(container.querySelectorAll('.conn-chip.on')).map((n) => n.textContent?.trim())
    expect(ticked).toEqual(['fixer'])
    expect(cockpit.settingsIntent).toBeNull() // consumed once
  })

  it('deletes through a confirm that says the people stay', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(container.querySelectorAll('.team-card').length).toBe(2))
    await fireEvent.click(container.querySelectorAll('.team-card')[1].querySelector('[aria-label="ลบทีม"]') as HTMLElement)
    expect(screen.getByText(/เอเจนในทีมยังอยู่ครบ/)).toBeTruthy()
    expect(vi.mocked(DeleteTeam)).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByText('ลบทีม', { selector: 'button.confirm, .confirm-dialog button, button' }))
    await waitFor(() => expect(vi.mocked(DeleteTeam)).toHaveBeenCalledWith('ทีมโค้ด'))
  })
})
