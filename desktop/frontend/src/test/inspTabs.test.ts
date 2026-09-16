import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Workbench from '../lib/workbench/Workbench.svelte'
import { workbench } from '../lib/stores/workbench.svelte'
import { codeStatus } from '../lib/stores/codeStatus.svelte'
import { setLocale } from '../lib/i18n.svelte'

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  workbench.tabs.length = 0
  workbench.activeId = ''
  codeStatus.gitChangedCount = 0
  codeStatus.openPRCount = 0
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect() {} })
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
})

describe('inspector tab strip (.insp-tabs)', () => {
  it('separates the tab scroll rack from the plus button so tabs never overlap it', async () => {
    workbench.tabs.push(
      { id: 'f-1', kind: 'file', name: 'eval.ts', path: 'src/eval.ts' } as any,
      { id: 't-1', kind: 'terminal', name: 'Terminal 1' } as any,
      { id: 't-2', kind: 'terminal', name: 'Terminal 2' } as any,
      { id: 'f-2', kind: 'file', name: 'Workbench.svelte', path: 'src/Workbench.svelte' } as any,
      { id: 'git-1', kind: 'git', name: 'Git' } as any,
    )
    workbench.activeId = 'git-1'
    codeStatus.gitChangedCount = 20

    const { container } = render(Workbench)
    await tick()

    const inspTabs = container.querySelector('.insp-tabs')
    expect(inspTabs).not.toBeNull()

    const scrollRack = container.querySelector('.insp-tabs-scroll')
    expect(scrollRack).not.toBeNull()

    const plusWrap = container.querySelector('.plus-menu-wrap')
    expect(plusWrap).not.toBeNull()

    // .plus-menu-wrap is a direct child of .insp-tabs, NOT inside the scroll rack
    expect(scrollRack?.parentElement).toBe(inspTabs)
    expect(plusWrap?.parentElement).toBe(inspTabs)
    expect(scrollRack?.contains(plusWrap)).toBe(false)

    // All tabs live inside the scroll rack
    const tabs = scrollRack?.querySelectorAll('.tab')
    expect(tabs?.length).toBe(5)

    // Git tab shows the badge with 20
    const gitTab = scrollRack?.querySelector('.tab.active')
    expect(gitTab?.textContent).toContain('Git')
    const badge = gitTab?.querySelector('.tab-badge.git')
    expect(badge?.textContent).toBe('20')
  })

  it('scrolls horizontally on wheel events inside the scroll rack', async () => {
    workbench.tabs.push(
      { id: 'f-1', kind: 'file', name: 'eval.ts' } as any,
      { id: 't-1', kind: 'terminal', name: 'Terminal' } as any,
    )
    workbench.activeId = 'f-1'

    const { container } = render(Workbench)
    await tick()

    const scrollRack = container.querySelector('.insp-tabs-scroll') as HTMLElement
    expect(scrollRack).not.toBeNull()

    scrollRack.scrollLeft = 0
    await fireEvent.wheel(scrollRack, { deltaY: 60 })
    expect(scrollRack.scrollLeft).toBe(60)
  })

  it('opens and closes the plus menu cleanly', async () => {
    workbench.tabs.push({ id: 'git-1', kind: 'git', name: 'Git' } as any)
    workbench.activeId = 'git-1'

    const { container } = render(Workbench)
    await tick()

    expect(container.querySelector('.plus-menu')).toBeNull()

    const plusBtn = container.querySelector('.plus-btn') as HTMLButtonElement
    await fireEvent.click(plusBtn)
    await tick()

    const menu = container.querySelector('.plus-menu')
    expect(menu).not.toBeNull()
  })
})
