import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import {
  ListSubagentProfiles, ReadSubagentProfile, LearnedEntries,
  DelegateSwitches, ListMCPServers, ListExternalSkills, ListTools, ListChairs,
} from './mocks/wailsApp'

describe('Agent Face Contexts Preview in Settings', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    cockpit.settingsIntent = null
    vi.mocked(DelegateSwitches).mockRejectedValue(new Error('unavailable'))
    vi.mocked(ListMCPServers).mockResolvedValue([])
    vi.mocked(ListExternalSkills).mockResolvedValue([])
    vi.mocked(ListTools).mockResolvedValue([])

    vi.mocked(ListSubagentProfiles).mockResolvedValue([
      { name: 'backend', description: 'งานหลังบ้าน', prompt: 'role', builtin: true, desk: 'specialized' },
    ] as any)
    vi.mocked(ListChairs).mockResolvedValue([{ name: 'backend' }] as any)

    vi.mocked(ReadSubagentProfile).mockResolvedValue(
      '---\ndescription: งานหลังบ้าน\n---\nโค้ดหลังบ้าน' as any,
    )
    vi.mocked(LearnedEntries).mockResolvedValue([])
  })

  it('renders context preview toggle button and toggles preview panel', async () => {
    cockpit.settingsIntent = { section: 'team', agent: 'backend' }

    const { container } = render(Settings, { onClose: () => {} })

    // Wait for agent editor to open
    await waitFor(() => expect(screen.getByText('ตั้งค่าเอเจน')).toBeTruthy())

    // Check for "ดูในบริบทต่างๆ" toggle button
    const toggleBtn = await screen.findByRole('button', { name: /ดูในบริบทต่างๆ/ })
    expect(toggleBtn).toBeDefined()
    expect(container.querySelector('.ag-contexts-panel')).toBeNull()

    // Click to expand context preview
    await fireEvent.click(toggleBtn)

    // Panel should now be rendered
    await waitFor(() => {
      expect(container.querySelector('.ag-contexts-panel')).not.toBeNull()
      expect(screen.getByText('ตัวอย่างเมื่อปรากฏในจุดต่างๆ ของแอป')).toBeDefined()
    })

    // Button text switches to "ซ่อนตัวอย่างบริบท"
    expect(screen.getByRole('button', { name: /ซ่อนตัวอย่างบริบท/ })).toBeDefined()

    // Click again to collapse
    await fireEvent.click(screen.getByRole('button', { name: /ซ่อนตัวอย่างบริบท/ }))
    await waitFor(() => {
      expect(container.querySelector('.ag-contexts-panel')).toBeNull()
    })
  })

  it('displays the three contexts and allows switching states', async () => {
    cockpit.settingsIntent = { section: 'team', agent: 'backend' }

    const { container } = render(Settings, { onClose: () => {} })

    await waitFor(() => expect(screen.getByText('ตั้งค่าเอเจน')).toBeTruthy())

    const toggleBtn = await screen.findByRole('button', { name: /ดูในบริบทต่างๆ/ })
    await fireEvent.click(toggleBtn)

    await waitFor(() => {
      expect(container.querySelector('.ag-contexts-panel')).not.toBeNull()
    })

    // 1. Office context card (38px)
    expect(screen.getByText(/ในห้องทำงานและทีม/)).toBeDefined()
    // 2. Chat context card (34px)
    expect(screen.getByText(/ในแชทและแถบงาน/)).toBeDefined()
    // 3. Composer context card (20px)
    expect(screen.getByText(/ในช่องพิมพ์และแท็ก/)).toBeDefined()

    // Interactive state switching in Chat context card
    const thinkBtn = screen.getByRole('button', { name: 'กำลังคิด…' })
    await fireEvent.click(thinkBtn)

    // Verify chat face has state="think"
    const chatFace = container.querySelector('.ag-context-card .agent-face.think')
    expect(chatFace).not.toBeNull()

    // Click Working
    const workBtn = screen.getByRole('button', { name: 'กำลังทำ…' })
    await fireEvent.click(workBtn)
    expect(container.querySelector('.ag-context-card .agent-face.work')).not.toBeNull()

    // Click Done
    const doneBtn = screen.getByRole('button', { name: 'สำเร็จ' })
    await fireEvent.click(doneBtn)
    expect(container.querySelector('.ag-context-card .agent-face.done')).not.toBeNull()

    // Toggle Office context card to disabled / off
    const offBtn = screen.getByRole('button', { name: 'ปิดใช้งาน' })
    await fireEvent.click(offBtn)
    expect(container.querySelector('.ag-context-card .agent-face.off')).not.toBeNull()
  })
})
