// The browser toolbar's overflow menu, and the bug that made it necessary.
//
// ไฟบอกสถานะ opened onto nothing. The checklist was drawn — correctly, in the
// DOM, in the right place — and then the tab's native WebView2 window
// composited over the top of it, because a native window sits above everything
// the app paints. The pane does hide itself for something covering the CENTRE
// of its box, and a menu hanging off the toolbar covers the top and never the
// centre, so nothing ever hid the page — and the button only exists on a
// browser tab, so there was no case in which it worked.
//
// The fix is not a bigger hit-test. It is that the panel has to SAY a menu is
// open, which is what it already did for the + menu and had to be told to do
// for the second one. That wiring is what the last test here guards: the
// failure has no symptom in a unit test, because everything renders perfectly.
import { readFileSync } from 'node:fs'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Workbench from '../lib/workbench/Workbench.svelte'
import { workbench } from '../lib/stores/workbench.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { BrowserOpenDevTools } from './mocks/wailsApp'

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  workbench.tabs.length = 0
  workbench.activeId = ''
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect() {} })
  workbench.tabs.push({ id: 'web-1', kind: 'browser', name: 'page', url: 'http://localhost:8080/' })
  workbench.activeId = 'web-1'
})

const openMenu = async (container: HTMLElement) => {
  await fireEvent.click(container.querySelector('.more-wrap button') as HTMLButtonElement)
  await tick()
}

const labels = (container: HTMLElement) =>
  Array.from(container.querySelectorAll('.more-menu > .more-item')).map(
    (el) => el.querySelector('.more-txt')?.textContent?.trim() ?? '',
  )

describe('the browser toolbar', () => {
  it('keeps only the two modes in the open, and folds the rest into one menu', async () => {
    const { container } = render(Workbench)

    // Pointing and drawing stay out: they are modes with a lit state you have
    // to be able to see, and each has a keyboard shortcut.
    expect(container.querySelector('.insp-addr')?.textContent).toBeDefined()
    expect(container.querySelectorAll('.more-menu').length).toBe(0)

    await openMenu(container)
    expect(labels(container)).toEqual(['Screen size', 'Working signal', 'DevTools', 'Open in its own window'])
  })

  it('shows what size the page is now without being opened twice', async () => {
    const { container } = render(Workbench)
    await openMenu(container)

    const row = container.querySelector('.more-menu > .more-item') as HTMLElement
    expect(row.querySelector('.more-val')?.textContent?.trim()).toBe('Fill panel')
  })

  it('opens one section at a time', async () => {
    const { container } = render(Workbench)
    await openMenu(container)
    const rows = Array.from(container.querySelectorAll('.more-menu > .more-item')) as HTMLButtonElement[]

    await fireEvent.click(rows[0])
    await tick()
    expect(container.querySelectorAll('.more-sub').length).toBe(1)

    // The other section replaces it rather than joining it: eight sizes and
    // four layers open at once is a menu taller than the panel.
    await fireEvent.click(rows[1])
    await tick()
    expect(container.querySelectorAll('.more-sub').length).toBe(1)
  })

  it('closes the menu before it runs a command, or the page is still hidden', async () => {
    const { container } = render(Workbench)
    await openMenu(container)

    // Opening the menu hides the page — it has to, or the menu is drawn under
    // a native window — and a hidden WebView2 refuses work that needs a live
    // view. เครื่องมือนักพัฒนา did nothing at all for exactly that reason.
    let menuWasStillOpen: boolean | null = null
    vi.mocked(BrowserOpenDevTools).mockImplementation(async () => {
      menuWasStillOpen = !!container.querySelector('.more-menu')
    })

    const rows = Array.from(container.querySelectorAll('.more-menu > .more-item')) as HTMLButtonElement[]
    await fireEvent.click(rows[2])
    await tick()
    await tick()

    expect(BrowserOpenDevTools).toHaveBeenCalledWith('web-1')
    expect(menuWasStillOpen).toBe(false)
  })

  it('tells the pane a menu is open, or the menu is drawn under the page', () => {
    const src = readFileSync('src/lib/workbench/Workbench.svelte', 'utf8')

    // Every flag that opens something over the pane has to reach this one
    // derived value...
    const derived = src.match(/const panelMenuOpen = \$derived\(([^)]*)\)/)?.[1] ?? ''
    for (const flag of ['menuOpen', 'moreOpen']) {
      expect(derived).toContain(flag)
    }
    // ...and that value, not any single flag, is what the pane is handed.
    expect(src).toContain('menuOpen={panelMenuOpen}')
  })
})
