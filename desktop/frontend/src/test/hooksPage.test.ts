// Hooks: the fourth heading of ห้องความสามารถ (14 ก.ย. 2026), and the first
// screen hooks.json ever had. internal/hook ran the user's command around a
// tool call for six weeks with the file edited by hand or not at all. What is
// pinned here: the page shows the file (rows, and a file that will not parse
// says so instead of vanishing), the sheet is the one form, and every change
// is one write of the WHOLE list — the engine loads the file, not a row.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent, within } from '@testing-library/svelte'
import Capability from '../lib/Capability.svelte'
import {
  Hooks, SaveHooks, ListMCPServers, ListExternalSkills, ListSubagentProfiles, PlacementTargets, ListTools,
} from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

const guard = { event: 'PreToolUse', matcher: 'shell', command: 'python hooks/guard.py', blocking: true }
const fmt = { event: 'PostToolUse', matcher: 'write', command: 'gofmt -l .', blocking: false }

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'capability'
  cockpit.capabilityIntent = null
  vi.mocked(ListMCPServers).mockResolvedValue([] as any)
  vi.mocked(ListExternalSkills).mockResolvedValue([] as any)
  vi.mocked(ListSubagentProfiles).mockResolvedValue([] as any)
  vi.mocked(PlacementTargets).mockResolvedValue([] as any)
  vi.mocked(ListTools).mockResolvedValue([] as any)
  vi.mocked(Hooks).mockResolvedValue({ path: 'C:/x/hooks.json', hooks: [] } as any)
})

const openHooks = async (rows: unknown[] = [], err = '') => {
  vi.mocked(Hooks).mockResolvedValue({ path: 'C:/x/hooks.json', hooks: rows, err } as any)
  const r = render(Capability, { onClose: () => {} })
  await waitFor(() => expect(document.querySelector('.office-grid')).toBeTruthy())
  const row = Array.from(document.querySelectorAll<HTMLElement>('.settings-nav-item')).find((x) => x.textContent?.includes('Hooks ของคุณ'))!
  await fireEvent.click(row)
  await waitFor(() => expect(Hooks).toHaveBeenCalled())
  await waitFor(() => expect(screen.getByRole('heading', { level: 2 }).textContent).toBe('Hooks ของคุณ'))
  return r
}
const sheet = () => document.querySelector('.cap-sheet') as HTMLElement

describe('the register', () => {
  it('is its own heading on the rail, under the three registers', async () => {
    await openHooks()
    const labels = Array.from(document.querySelectorAll('.settings-nav .settings-group-label')).map((x) => x.textContent?.trim())
    expect(labels).toEqual(['MCP', 'สกิล', 'เครื่องมือในตัว', 'Hooks'])
  })

  it('shows each hook as when · what · command, and blocking as a word', async () => {
    await openHooks([guard, fmt])
    const rows = Array.from(document.querySelectorAll('.tool-row'))
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('ก่อนรันเครื่องมือ')
    expect(rows[0].textContent).toContain('shell')
    expect(rows[0].textContent).toContain('python hooks/guard.py')
    expect(within(rows[0] as HTMLElement).getByText('บล็อก')).toBeTruthy()
    expect(rows[1].textContent).toContain('หลังรันเครื่องมือ')
    expect(within(rows[1] as HTMLElement).queryByText('บล็อก')).toBeNull()
    expect(screen.getByText('2 hook')).toBeTruthy()
    expect(screen.getByText('C:/x/hooks.json')).toBeTruthy()
  })

  it('says so when there are none, and reads a blank matcher as every tool', async () => {
    await openHooks()
    expect(screen.getByText(/ยังไม่มี hook/)).toBeTruthy()
    vi.mocked(Hooks).mockResolvedValue({ path: 'C:/x/hooks.json', hooks: [{ ...fmt, matcher: '*' }] } as any)
    await fireEvent.click(screen.getByText('เพิ่ม hook'))
    // Opening the sheet does not reload; a save does. Read the row through
    // a save of the same file instead.
    await fireEvent.input(sheet().querySelector('textarea')!, { target: { value: 'echo hi' } })
    await fireEvent.click(within(sheet()).getByText('เพิ่ม'))
    await waitFor(() => expect(screen.getByText('ทุกเครื่องมือ')).toBeTruthy())
  })

  // Bootstrap logs a broken file and runs without hooks (§57); the page is
  // where a person finds that out, with the rows it could still read.
  it('a file that will not parse is announced, not hidden', async () => {
    await openHooks([], 'invalid character at line 3')
    expect(screen.getByText(/อ่านไม่ออก/).textContent).toContain('invalid character at line 3')
  })
})

describe('the sheet, the one form', () => {
  it('adds a hook by writing the whole list, defaulting to before and every tool', async () => {
    await openHooks([guard])
    await fireEvent.click(screen.getByText('เพิ่ม hook'))
    const s = sheet()
    expect(s).toBeTruthy()
    expect((s.querySelector('select') as HTMLSelectElement).value).toBe('PreToolUse')
    // Nothing to run, nothing to save.
    const add = within(s).getByText('เพิ่ม') as HTMLButtonElement
    expect(add.disabled).toBe(true)
    await fireEvent.input(s.querySelector('textarea')!, { target: { value: '  gofmt -l .  ' } })
    expect(add.disabled).toBe(false)
    await fireEvent.click(add)
    await waitFor(() => expect(SaveHooks).toHaveBeenCalledTimes(1))
    expect(vi.mocked(SaveHooks).mock.calls[0][0]).toEqual([
      guard,
      { event: 'PreToolUse', matcher: '*', command: 'gofmt -l .', blocking: false },
    ])
    await waitFor(() => expect(document.querySelector('.cap-sheet')).toBeNull())
  })

  it('edits a row in place and the hint follows the event', async () => {
    await openHooks([guard, fmt])
    await fireEvent.click(document.querySelectorAll('.tool-row')[1])
    const s = sheet()
    expect((s.querySelector('select') as HTMLSelectElement).value).toBe('PostToolUse')
    expect(s.textContent).toContain('ตีกลับให้โมเดลแก้')
    await fireEvent.change(s.querySelector('select')!, { target: { value: 'PreToolUse' } })
    expect(s.textContent).toContain('ถูกปฏิเสธ')
    await fireEvent.click(s.querySelector('.mswitch input')!)
    await fireEvent.click(within(s).getByText('บันทึก'))
    await waitFor(() => expect(SaveHooks).toHaveBeenCalledTimes(1))
    expect(vi.mocked(SaveHooks).mock.calls[0][0]).toEqual([guard, { ...fmt, event: 'PreToolUse', blocking: true }])
  })

  it('removes a row only after the dialog, and writes the rest', async () => {
    await openHooks([guard, fmt])
    await fireEvent.click(document.querySelectorAll('.tool-row')[0])
    await fireEvent.click(within(sheet()).getByText('ลบ'))
    expect(SaveHooks).not.toHaveBeenCalled()
    await fireEvent.click(document.querySelector('.confirm-go')!)
    await waitFor(() => expect(SaveHooks).toHaveBeenCalledWith([fmt]))
  })

  // The engine refuses what hook.Validate refuses; the page shows why on the
  // sheet rather than closing it over a row that was never written.
  it('keeps the sheet open with the engine\'s reason when the save is refused', async () => {
    vi.mocked(SaveHooks).mockRejectedValueOnce(new Error('hook 1: no command'))
    await openHooks([])
    await fireEvent.click(screen.getByText('เพิ่ม hook'))
    await fireEvent.input(sheet().querySelector('textarea')!, { target: { value: 'x' } })
    await fireEvent.click(within(sheet()).getByText('เพิ่ม'))
    await waitFor(() => expect(within(sheet()).getByText(/no command/)).toBeTruthy())
  })
})
