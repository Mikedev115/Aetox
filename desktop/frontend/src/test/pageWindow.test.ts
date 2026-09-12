// The window a long grid is drawn through (lib/pageWindow.svelte.ts): one
// number, grown a step at a time, never past the list, reset per visit. And
// the room's two loads: the MCP half on open, the skill half only once a
// skill page is opened — 494 skill folders are not walked for an MCP test.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import { pageWindow } from '../lib/pageWindow.svelte'
import Capability from '../lib/Capability.svelte'
import { ListMCPServers, ListExternalSkills, ListTools, PlacementTargets, ListSubagentProfiles, TestMCPServer } from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

describe('pageWindow', () => {
  it('draws a short list whole, and a long one a step at a time', () => {
    const w = pageWindow(3)
    const short = [1, 2]
    expect(w.take(short)).toBe(short) // the array itself, no copy
    expect(w.more(short.length)).toBe(false)
    const long = [1, 2, 3, 4, 5, 6, 7]
    expect(w.take(long)).toEqual([1, 2, 3])
    expect(w.more(7)).toBe(true)
    expect(w.rest(7)).toBe(4)
    w.grow(7)
    expect(w.take(long)).toEqual([1, 2, 3, 4, 5, 6])
    w.grow(7)
    expect(w.take(long)).toBe(long) // never past the list
    expect(w.more(7)).toBe(false)
    w.reset()
    expect(w.take(long)).toEqual([1, 2, 3])
  })
})

describe('the two loads', () => {
  const server = (over: Record<string, unknown> = {}) =>
    ({ name: 'firecrawl', disabled: false, status: 'idle', tools: 0, for: [], url: 'https://mcp.firecrawl.dev/v2/mcp', ...over })
  const skill = (name: string) => ({ name, description: 'x', dir: 'C:/skills/' + name })
  beforeEach(() => {
    vi.clearAllMocks()
    cockpit.activeView = 'capability'
    vi.mocked(ListMCPServers).mockResolvedValue([server()] as any)
    vi.mocked(ListExternalSkills).mockResolvedValue([] as any)
    vi.mocked(ListSubagentProfiles).mockResolvedValue([] as any)
    vi.mocked(PlacementTargets).mockResolvedValue([] as any)
    vi.mocked(ListTools).mockResolvedValue([] as any)
  })
  const rail = (label: string) =>
    Array.from(document.querySelectorAll<HTMLElement>('.settings-nav-item')).find((x) => x.textContent?.includes(label))!

  it('walks the skill folders only once a skill page is opened, and not for an MCP test', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(document.querySelector('.office-grid')).toBeTruthy())
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalled()) // the probe
    await waitFor(() => expect(document.querySelectorAll('.cap-act.testing').length).toBe(0))
    expect(ListExternalSkills).not.toHaveBeenCalled()
    await fireEvent.click(rail('สกิลของคุณ'))
    await waitFor(() => expect(ListExternalSkills).toHaveBeenCalledTimes(1))
    await fireEvent.click(rail('MCP server ของคุณ'))
    await fireEvent.click(screen.getByText('ทดสอบทั้งหมด'))
    await waitFor(() => expect(TestMCPServer).toHaveBeenCalledTimes(2))
    expect(ListExternalSkills).toHaveBeenCalledTimes(1)
  })

  it('draws a long skill grid a window at a time, with the rest behind a button', async () => {
    vi.mocked(ListExternalSkills).mockResolvedValue(Array.from({ length: 130 }, (_, i) => skill('s' + i)) as any)
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(document.querySelector('.office-grid')).toBeTruthy())
    await fireEvent.click(rail('สกิลของคุณ'))
    // jsdom's IntersectionObserver reports every target as visible at once,
    // so the sentinel opens one step by itself: 48 + 48.
    await waitFor(() => expect(document.querySelectorAll('.chair-card.agc').length).toBe(96))
    const more = screen.getByText('ดูเพิ่มอีก 34')
    await fireEvent.click(more)
    await waitFor(() => expect(document.querySelectorAll('.chair-card.agc').length).toBe(130))
    expect(document.querySelector('.cap-more')).toBeNull()
  })
})
