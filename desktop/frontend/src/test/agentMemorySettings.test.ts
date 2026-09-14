import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import {
  MemoryScopeInfo,
  ListSubagentProfiles, ReadSubagentProfile, LearnedEntries,
  AddLearnedEntry, SaveLearnedEntry, OpenAgentHome,
  DelegateSwitches, ListMCPServers, ListExternalSkills, ListTools, ListChairs,
} from './mocks/wailsApp'

describe('Agent Memory in Settings', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    cockpit.settingsIntent = null
    vi.mocked(DelegateSwitches).mockRejectedValue(new Error('unavailable'))
    vi.mocked(ListMCPServers).mockResolvedValue([])
    vi.mocked(ListExternalSkills).mockResolvedValue([])
    vi.mocked(ListTools).mockResolvedValue([])

    vi.mocked(ListSubagentProfiles).mockResolvedValue([
      { name: 'deck', description: 'ทำสไลด์', prompt: 'role', builtin: true, desk: 'specialized' },
    ] as any)
    vi.mocked(ListChairs).mockResolvedValue([{ name: 'deck' }] as any)

    vi.mocked(ReadSubagentProfile).mockResolvedValue(
      '---\ndescription: ทำสไลด์\n---\nสร้างสไลด์หนึ่งชุด' as any,
    )
  })

  it('renders memory box with action buttons even when agent has 0 memories', async () => {
    vi.mocked(LearnedEntries).mockResolvedValue([])
    cockpit.settingsIntent = { section: 'team', agent: 'deck' }

    render(Settings, { onClose: () => {} })

    // Wait for the agent editor pane to open
    await waitFor(() => expect(screen.getByRole('tablist', { name: 'ตั้งค่าพนักงาน' })).toBeTruthy())

    // Switch to knowledge tab
    const tab = await screen.findByRole('tab', { name: /^สกิล$/ })
    await fireEvent.click(tab)

    // In agent editor pane: check memory box empty message
    await waitFor(() => {
      expect(screen.getByText('ยังไม่มีอะไรที่คุณอนุมัติให้จำ')).toBeDefined()
    })

    // Action buttons must be present
    const addBtn = screen.getByRole('button', { name: /เพิ่มความจำ/ })
    const folderBtn = screen.getByRole('button', { name: /เปิดโฟลเดอร์เอเจน/ })

    expect(addBtn).toBeDefined()
    expect(folderBtn).toBeDefined()

    // Clicking OpenAgentHome calls backend
    await fireEvent.click(folderBtn)
    expect(OpenAgentHome).toHaveBeenCalledWith('deck')
  })

  it('allows adding a new memory directly via inline form', async () => {
    vi.mocked(LearnedEntries).mockResolvedValue([])
    cockpit.settingsIntent = { section: 'team', agent: 'deck' }

    const { container } = render(Settings, { onClose: () => {} })

    await waitFor(() => expect(screen.getByRole('tablist', { name: 'ตั้งค่าพนักงาน' })).toBeTruthy())

    // Switch to knowledge tab
    const tab = await screen.findByRole('tab', { name: /^สกิล$/ })
    await fireEvent.click(tab)

    const addBtn = await screen.findByRole('button', { name: /เพิ่มความจำ/ })
    await fireEvent.click(addBtn)

    // Textarea should appear
    const textarea = container.querySelector('textarea.mem-input') as HTMLTextAreaElement
    expect(textarea).toBeDefined()

    // Type a new fact
    await fireEvent.input(textarea, { target: { value: 'Always use strict TypeScript' } })

    // Click Save in memory actions
    const saveBtn = container.querySelector('.mem-actions .ctrl-primary') as HTMLButtonElement
    await fireEvent.click(saveBtn)

    expect(AddLearnedEntry).toHaveBeenCalledWith('deck', 'Always use strict TypeScript')
  })

  it('displays existing memories with inline edit and delete buttons', async () => {
    vi.mocked(LearnedEntries).mockResolvedValue([
      'Preference: Thai output',
      'Fact: Fast slide generator',
    ])
    cockpit.settingsIntent = { section: 'team', agent: 'deck' }

    const { container } = render(Settings, { onClose: () => {} })

    await waitFor(() => expect(screen.getByRole('tablist', { name: 'ตั้งค่าพนักงาน' })).toBeTruthy())

    // Switch to knowledge tab
    const tab = await screen.findByRole('tab', { name: /^สกิล$/ })
    await fireEvent.click(tab)

    await waitFor(() => {
      expect(screen.getByText('Preference: Thai output')).toBeDefined()
      expect(screen.getByText('Fact: Fast slide generator')).toBeDefined()
    })

    // Find delete buttons (.mem-forget)
    const deleteButtons = container.querySelectorAll('.mem-row .mem-forget')
    expect(deleteButtons.length).toBe(2)

    // Click delete on the first line
    await fireEvent.click(deleteButtons[0])
    expect(SaveLearnedEntry).toHaveBeenCalledWith('deck', 0, '')
  })

  // The head page's block, on a delegate (14 ก.ย. 2026, owner: "พนักงานหรือเอเจน
  // ทุกตัวควรจะมีความจำแยกแบบนี้ CSS มาตรฐานเดียวกัน"): the same heading, the
  // scope head with its face and the file badge, the meter from the scope's
  // own info, and the same rows.
  it("draws the delegate's memory in the head page's own block, with its meter", async () => {
    vi.mocked(LearnedEntries).mockResolvedValue(['Preference: Thai output'])
    vi.mocked(MemoryScopeInfo).mockResolvedValue({ scope: 'deck', bytes: 1200, maxBytes: 8192, full: false } as any)
    cockpit.settingsIntent = { section: 'team', agent: 'deck' }
    const { container } = render(Settings, { onClose: () => {} })
    await waitFor(() => expect(screen.getByRole('tablist', { name: 'ตั้งค่าพนักงาน' })).toBeTruthy())
    await fireEvent.click(await screen.findByRole('tab', { name: /^สกิล$/ }))
    const head = await waitFor(() => { const h = container.querySelector('.mem-desk .mem-scope[data-mem-scope="deck"]'); expect(h).toBeTruthy(); return h! })
    expect(head.querySelector('.mem-scope-name')?.textContent?.trim()).toBe('deck')
    expect(head.querySelector('.mem-badge-file')?.textContent).toBe('agents/deck/MEMORY.md')
    expect(head.querySelector('.scope-face .mascot')).toBeTruthy()
    await waitFor(() => expect(head.querySelector('.mem-cap-num')?.textContent).toContain('1,200 / 8,192'))
    expect(screen.getByText('ความจำของdeck')).toBeTruthy()
    expect(container.querySelectorAll('.mem-desk .mem-row').length).toBe(1)
  })

  // A ลูกมือ has the tab too (fddd0b15) — and the panel behind it. It sat
  // inside the agent-only block, so the tab opened on a page that had never
  // asked for the file (owner, 14 ก.ย. 2026: "ทำไมโล่งแบบนี้").
  it("draws a helper's memory behind its ความจำ tab, loaded on open", async () => {
    vi.mocked(ListSubagentProfiles).mockResolvedValue([
      { name: 'deck', description: 'ทำสไลด์', prompt: 'role', builtin: true, desk: 'specialized' },
      { name: 'explore', description: 'ค้นไฟล์', prompt: 'role', builtin: true },
    ] as any)
    vi.mocked(LearnedEntries).mockResolvedValue(['ไฟล์ทดสอบอยู่ใน src/test'])
    vi.mocked(MemoryScopeInfo).mockResolvedValue({ scope: 'explore', bytes: 40, maxBytes: 8192, full: false } as any)
    const { container } = render(Settings, { onClose: () => {} })
    // Through the ลูกมือ page's cog — the helper door; an intent naming an
    // agent opens through the agent door and would make it a พนักงาน.
    const items = Array.from(container.querySelectorAll('.settings-nav-item'))
    await fireEvent.click(items.find((el) => el.textContent?.trim() === 'ลูกมือ')!)
    await waitFor(() => expect(screen.getByText('ค้นไฟล์')).toBeTruthy())
    await fireEvent.click(container.querySelector('.office-grid button[aria-label="ตั้งค่า"]')!)
    await waitFor(() => expect(container.querySelector('#ag-panel-identity')).toBeTruthy())
    expect(screen.queryByRole('tab', { name: /^สกิล$/ })).toBeNull() // a helper has no reach tabs
    await fireEvent.click(await screen.findByRole('tab', { name: /^ความจำ$/ }))
    await waitFor(() => expect(LearnedEntries).toHaveBeenCalledWith('explore'))
    const head = await waitFor(() => { const h = container.querySelector('.mem-desk .mem-scope[data-mem-scope="explore"]'); expect(h).toBeTruthy(); return h! })
    expect(head.querySelector('.mem-badge-file')?.textContent).toBe('agents/explore/MEMORY.md')
    expect(container.querySelectorAll('.mem-desk .mem-row').length).toBe(1)
    expect(container.querySelector('#ag-panel-memory')?.classList.contains('on')).toBe(true)
  })
})
