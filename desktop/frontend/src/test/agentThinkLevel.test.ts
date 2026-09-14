import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import {
  ListSubagentProfiles, ReadSubagentProfile, SaveAgentProfile,
  SupportedThinkLevelsFor, EnabledProviders, SupportedProviders, ListModelsForProvider,
  DelegateSwitches, ListMCPServers, ListExternalSkills, ListTools, ListChairs,
} from './mocks/wailsApp'

// The depth dial on an agent's สมอง tab (owner, 13 ก.ย. 2026: "เอเจนและซับเอเจน
// ทำให้เราปรับระดับความคิดได้"). Its own file rather than another block inside
// Settings.test.ts, which is being edited elsewhere at the time of writing.
//
// Worth stating why this file exists at all: the dial shipped with no frontend
// coverage, and the reason was invisible. Settings.test.ts asserts the สมอง tab
// has exactly two selects, and that assertion kept passing after a third was
// added — because SupportedThinkLevelsFor is mocked empty there, and an empty
// ladder hides the row. The feature was untested and the suite looked fine.
describe('an agent’s thinking level', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    cockpit.settingsIntent = null
    vi.mocked(DelegateSwitches).mockRejectedValue(new Error('unavailable'))
    vi.mocked(ListMCPServers).mockResolvedValue([])
    vi.mocked(ListExternalSkills).mockResolvedValue([])
    vi.mocked(ListTools).mockResolvedValue([])
    vi.mocked(SupportedProviders).mockResolvedValue(['codex', 'ollama'] as any)
    vi.mocked(EnabledProviders).mockResolvedValue(['codex', 'ollama'] as any)
    vi.mocked(ListModelsForProvider).mockResolvedValue(['gpt-5.6-luna'] as any)
    vi.mocked(ListSubagentProfiles).mockResolvedValue([
      { name: 'deck', description: 'ทำสไลด์', prompt: 'role', builtin: true, desk: 'specialized' },
    ] as any)
    vi.mocked(ListChairs).mockResolvedValue([{ name: 'deck' }] as any)
    vi.mocked(ReadSubagentProfile).mockResolvedValue(
      '---\ndescription: ทำสไลด์\n---\nสร้างสไลด์หนึ่งชุด' as any,
    )
  })

  const openBrainTab = async () => {
    cockpit.settingsIntent = { section: 'team', agent: 'deck' }
    const { container } = render(Settings, { onClose: () => {} })
    await waitFor(() => expect(screen.getByRole('tablist', { name: 'ตั้งค่าพนักงาน' })).toBeTruthy())
    await fireEvent.click(screen.getByRole('tab', { name: 'สมอง' }))
    return container
  }

  const brainSelects = (container: HTMLElement) =>
    Array.from(container.querySelectorAll<HTMLSelectElement>('#ag-panel-brain select.ctrl'))

  it('offers the levels the engine states for this model, and writes the pick to the file', async () => {
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue(['off', 'low', 'medium', 'high'] as any)
    const container = await openBrainTab()

    // Three rows on the tab now: where it thinks, on what, and how deep.
    await waitFor(() => expect(brainSelects(container).length).toBe(3))
    const think = brainSelects(container)[2]
    expect(Array.from(think.options).map((o) => o.value)).toEqual(['', 'off', 'low', 'medium', 'high'])

    // Blank is inheriting, and inheriting writes no line at all — a file full
    // of lines restating the default is a file whose default can never change.
    await fireEvent.change(think, { target: { value: 'low' } })
    await fireEvent.click(screen.getByText('บันทึก'))
    await waitFor(() => expect(vi.mocked(SaveAgentProfile)).toHaveBeenCalled())
    expect(vi.mocked(SaveAgentProfile).mock.calls.at(-1)![1]).toContain('think: low')
  })

  it('is not drawn at all for a model with no dial', async () => {
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue([] as any)
    const container = await openBrainTab()

    await waitFor(() => expect(brainSelects(container).length).toBe(2))
    expect(screen.queryByText('ระดับความคิด')).toBeNull()
  })

  // A profile that already names a level keeps its row even where the level
  // cannot be honoured, so switching the model pin to look at something does
  // not silently eat a line its author wrote.
  it('keeps a level the file already carries, with a warning, on a model that has none', async () => {
    vi.mocked(SupportedThinkLevelsFor).mockResolvedValue([] as any)
    vi.mocked(ReadSubagentProfile).mockResolvedValue(
      '---\ndescription: ทำสไลด์\nthink: high\n---\nสร้างสไลด์หนึ่งชุด' as any,
    )
    const container = await openBrainTab()

    await waitFor(() => expect(brainSelects(container).length).toBe(3))
    expect(brainSelects(container)[2].value).toBe('high')
    expect(screen.getByText(/ไม่มีระดับความคิด/)).toBeTruthy()

    await fireEvent.click(screen.getByText('บันทึก'))
    await waitFor(() => expect(vi.mocked(SaveAgentProfile)).toHaveBeenCalled())
    expect(vi.mocked(SaveAgentProfile).mock.calls.at(-1)![1]).toContain('think: high')
  })
})
