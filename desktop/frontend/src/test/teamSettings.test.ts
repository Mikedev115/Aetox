// Settings › ทีมเอเจน (§256): the one home of a roster, in การตั้งค่าโมเดล's
// master–detail shape — a two-way switch between ฝั่งผู้ช่วย and ฝั่งโค้ด at
// the top, that side's teams in the rail, one roster in the pane. What is
// pinned: the switch at the top picks what the page SHOWS (it is not an
// on/off — the door's on/off lives in the chat's team menu); the rail names
// the shown side's teams with their reach tally; the pane draws the picked
// roster with its members' switches; every team — the seeded ทีมเอเจน
// included — is editable and deletable; the door to a new team is visible on
// either side; the form has no desk choice of its own (the side above already
// made it) and offers "+ เอเจน" to make a missing agent and come back; the
// editor writes through the engine's one door with exactly what was ticked;
// and the roster page's intents land on the right team.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import TeamSettings from '../lib/TeamSettings.svelte'
import { ListChairs, ListTeams, SaveTeam, DeleteTeam, DelegateSwitches, SetAgentOff } from './mocks/wailsApp'
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
const sideTab = (container: HTMLElement, side: 'side-assistant' | 'side-code') =>
  container.querySelector(`.team-side-pick .seg-btn.${side}`) as HTMLButtonElement

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.settingsIntent = null
  sessionStorage.clear()
  vi.mocked(ListChairs).mockResolvedValue([chair(), chair({ name: 'fixer', builtin: false })] as any)
  vi.mocked(ListTeams).mockImplementation(async () => [seed(), codeTeam()] as any)
  vi.mocked(DelegateSwitches).mockImplementation(async (name: string) =>
    switches(name, name === 'ทีมเอเจน' ? [{ name: 'doc', on: true }] : [{ name: 'fixer', on: false }]) as any)
})

describe('Settings › ทีมเอเจน', () => {
  it('opens on ฝั่งผู้ช่วย, with the switch at the top counting both sides and the rail showing one', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(1))
    // The switch: a tablist, not two on/offs — no switch input anywhere above the card.
    expect(container.querySelector('.team-side-pick[role="tablist"]')).toBeTruthy()
    expect(container.querySelector('.team-side-pick .mswitch')).toBeNull()
    expect(sideTab(container, 'side-assistant').classList.contains('selected')).toBe(true)
    expect(sideTab(container, 'side-assistant').textContent).toContain('1')
    expect(sideTab(container, 'side-code').textContent).toContain('1')
    // The rail: the shown side's header, its team with the reach tally, its door.
    expect(container.querySelector('.mset-side .team-side.side-assistant')).toBeTruthy()
    expect(container.querySelector('.mset-side .team-side.side-code')).toBeNull()
    expect(rail(container)[0].textContent).toContain('ทีมเอเจน')
    await waitFor(() => expect(rail(container)[0].textContent).toContain('1/1'))
    expect(container.querySelectorAll('.mset-side .team-add').length).toBe(1)
    // Opens on the side's first team; the seed is a team like any other — a
    // gear, and no master switch of its own in the pane.
    expect(rail(container)[0].classList.contains('selected')).toBe(true)
    expect(pane(container).querySelector('.mset-name')?.textContent).toBe('ทีมเอเจน')
    expect(pane(container).querySelector('.desk-badge.side-assistant')?.textContent).toContain('ฝั่งผู้ช่วย')
    expect(screen.getByText('แก้ไขทีม', { selector: 'button' })).toBeTruthy()
    expect(pane(container).querySelector('.team-reach')).toBeNull()
    await waitFor(() => expect(pane(container).querySelectorAll('.team-member').length).toBe(1))
    expect((pane(container).querySelector('.team-member .mswitch input') as HTMLInputElement).checked).toBe(true)
  })

  it('switches to ฝั่งโค้ด: only its teams in the rail, its first team in the pane', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(1))
    await fireEvent.click(sideTab(container, 'side-code'))
    await waitFor(() => expect(rail(container)[0]?.textContent).toContain('ทีมโค้ด'))
    expect(rail(container).length).toBe(1)
    expect(sideTab(container, 'side-code').classList.contains('selected')).toBe(true)
    expect(container.querySelector('.mset-side .team-side.side-code')).toBeTruthy()
    expect(rail(container)[0].textContent).toContain('0/1')
    expect(pane(container).querySelector('.desk-badge.side-code')?.textContent).toContain('ฝั่งโค้ด')
  })

  it('says a side has no team yet, and still offers its door', async () => {
    vi.mocked(ListTeams).mockImplementation(async () => [seed()] as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(1))
    // The nudge, while the seed is the only team on the side (drawn once the
    // switches have loaded, a tick after the rail).
    await waitFor(() => expect(container.querySelector('.team-callout')).toBeTruthy())
    await fireEvent.click(sideTab(container, 'side-code'))
    await waitFor(() => expect(container.querySelector('.mset-side .team-side-empty')).toBeTruthy())
    expect(rail(container).length).toBe(0)
    expect(container.querySelectorAll('.mset-side .team-add').length).toBe(1)
    expect(container.querySelector('.team-title .team-new')).toBeTruthy()
  })

  it('cools the rows under a side whose door is off, and says where the door is', async () => {
    vi.mocked(DelegateSwitches).mockImplementation(async (name: string) =>
      switches(name, name === 'ทีมเอเจน' ? [{ name: 'doc', on: true }] : [{ name: 'fixer', on: true }], { agents: false, code: true }) as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container)[0]?.textContent).toContain('1/1'))
    expect(container.querySelector('.team-door-off')).toBeNull()
    await fireEvent.click(sideTab(container, 'side-code'))
    await waitFor(() => expect(container.querySelector('.team-door-off')?.textContent).toContain('เมนูทีมในแชท'))
    expect(rail(container)[0].textContent).toContain('0/1')
    await waitFor(() => expect(pane(container).querySelector('.team-members.cool')).toBeTruthy())
    expect((pane(container).querySelector('.team-member .mswitch input') as HTMLInputElement).disabled).toBe(true)
  })

  it("flips a member's reach on the team it sits under, not everywhere", async () => {
    vi.mocked(SetAgentOff).mockResolvedValue(switches('ทีมโค้ด', [{ name: 'fixer', on: true }]) as any)
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(1))
    await fireEvent.click(sideTab(container, 'side-code'))
    await waitFor(() => expect(pane(container).querySelector('.team-member .mswitch input')).toBeTruthy())
    await fireEvent.click(pane(container).querySelector('.team-member .mswitch input') as HTMLInputElement)
    await waitFor(() => expect(vi.mocked(SetAgentOff)).toHaveBeenCalledWith('ทีมโค้ด', 'fixer', false))
  })

  it('makes a new team on the side shown, with no desk choice of its own', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(1))
    await fireEvent.click(sideTab(container, 'side-code'))
    await waitFor(() => expect(container.querySelector('.mset-side .team-add')).toBeTruthy())
    await fireEvent.click(container.querySelector('.mset-side .team-add') as HTMLElement)
    await waitFor(() => expect(pane(container).querySelector('.mset-name')?.textContent).toBe('สร้างทีม'))
    // The side is a badge in the head — the switch above chose it (owner:
    // "เราเลือกข้างบนอยู่แล้ว จะมีปุ่มนี้ทำไม") — and the form says what
    // that side hands the team.
    expect(pane(container).querySelector('.desk-badge.side-code')).toBeTruthy()
    expect(pane(container).querySelector('.team-desk-pick')).toBeNull()
    expect(pane(container).querySelector('[role="radiogroup"]')).toBeNull()
    expect(screen.getByText(/ถือเชลล์/)).toBeTruthy()
  })

  it('saves a new team through the engine door with what was ticked', async () => {
    const { container } = render(TeamSettings)

    await waitFor(() => expect(rail(container).length).toBe(1))
    await fireEvent.click(sideTab(container, 'side-code'))
    await waitFor(() => expect(container.querySelector('.team-title .team-new')).toBeTruthy())
    await fireEvent.click(container.querySelector('.team-title .team-new') as HTMLElement)
    // The house form: mset-field + .ctrl, conn-chip ticks — nothing of this
    // page's own (owner: "ดูมาตรฐานหน้าอื่นครับ").
    const name = pane(container).querySelector('.mset-field input.ctrl') as HTMLInputElement
    await fireEvent.input(name, { target: { value: 'ทีมเอกสาร' } })
    await fireEvent.click(pane(container).querySelector('.conn-targets .conn-chip') as HTMLElement)
    await fireEvent.click(screen.getByText('บันทึกทีม'))

    await waitFor(() => expect(vi.mocked(SaveTeam).mock.calls[0]).toEqual(['ทีมเอกสาร', 'coding', '', ['doc']]))
  })

  it('offers "+ เอเจน" in the tick list only when given the door, parks the draft, and takes it back with the new agent ticked', async () => {
    // No door handed in: no chip.
    const bare = render(TeamSettings)
    await waitFor(() => expect(bare.container.querySelector('.team-title .team-new')).toBeTruthy())
    await fireEvent.click(bare.container.querySelector('.team-title .team-new') as HTMLElement)
    expect(bare.container.querySelector('.team-add-agent')).toBeNull()
    bare.unmount()

    const onNewAgent = vi.fn()
    const { container, unmount } = render(TeamSettings, { props: { onNewAgent } })
    await waitFor(() => expect(container.querySelector('.team-title .team-new')).toBeTruthy())
    await fireEvent.click(container.querySelector('.team-title .team-new') as HTMLElement)
    const name = pane(container).querySelector('.mset-field input.ctrl') as HTMLInputElement
    await fireEvent.input(name, { target: { value: 'ทีมร้าน' } })
    await fireEvent.click(pane(container).querySelector('.conn-targets .conn-chip') as HTMLElement) // doc
    await fireEvent.click(container.querySelector('.team-add-agent') as HTMLElement)
    expect(onNewAgent).toHaveBeenCalledTimes(1)
    unmount()

    // Back from the agent editor with the agent just made: the draft is
    // where it was, the new agent ticked beside what was ticked before.
    vi.mocked(ListChairs).mockResolvedValue([chair(), chair({ name: 'fixer', builtin: false }), chair({ name: 'sales', builtin: false })] as any)
    cockpit.settingsIntent = { section: 'teams', agent: 'sales' }
    const back = render(TeamSettings, { props: { onNewAgent } })
    await waitFor(() => expect(pane(back.container).querySelector('.mset-name')?.textContent).toBe('สร้างทีม'))
    expect((pane(back.container).querySelector('.mset-field input.ctrl') as HTMLInputElement).value).toBe('ทีมร้าน')
    const ticked = Array.from(pane(back.container).querySelectorAll('.conn-chip.on')).map((n) => n.textContent?.trim())
    expect(ticked).toEqual(['doc', 'sales'])
    expect(cockpit.settingsIntent).toBeNull()
    expect(sessionStorage.getItem('aetox.teamDraft')).toBeNull() // taken once
  })

  it('opens on the team the roster page pointed at — its side, its editor, its members ticked', async () => {
    cockpit.settingsIntent = { section: 'teams', team: 'ทีมโค้ด' }
    const { container } = render(TeamSettings)

    await waitFor(() => expect(pane(container).querySelector('.mset-name')?.textContent).toBe('แก้ไขทีม'))
    expect(sideTab(container, 'side-code').classList.contains('selected')).toBe(true)
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
