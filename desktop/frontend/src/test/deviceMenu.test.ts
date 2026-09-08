// Picking a device is two halves, and for as long as the menu existed only one
// of them happened.
//
// The size was ours: BrowserPane shrinks the native window to it, so the pane
// looks like a phone and the page's own media queries fire at 375 for real.
// Everything else about being a phone — the user agent, the touch screen, the
// pixel ratio — lives inside the engine, and nothing was ever telling it. So a
// site with a real mobile build kept serving the desktop one into a narrow
// window, which reads as a broken responsive layout and is not: it is the wrong
// page, rendered correctly.
//
// These two tests are the two halves. The first is that the rows come from Go
// rather than from a list in the frontend (there were eight literals here until
// the agent needed the same eight). The second is that choosing one reaches the
// engine at all.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import { tick } from 'svelte'
import Workbench from '../lib/workbench/Workbench.svelte'
import { workbench } from '../lib/stores/workbench.svelte'
import { BrowserSetDevice } from './mocks/wailsApp'
import { setLocale } from '../lib/i18n.svelte'

beforeEach(() => {
  vi.clearAllMocks()
  setLocale('en')
  workbench.tabs.length = 0
  workbench.activeId = ''
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect() {} })
})

/** A browser tab, which is what puts the device picker on screen. */
function withBrowserTab() {
  workbench.tabs.push({ id: 'web-1', kind: 'browser', name: 'page', url: 'http://localhost:8080/' })
  workbench.activeId = 'web-1'
}

/** Open the overflow menu and expand its size section.
 *
 * A native <select> until 8 ก.ย., when the four toolbar buttons and the picker
 * were folded into one ⋮ menu. The two halves this file is about did not
 * change; only the way a person reaches them did.
 */
async function sizeRows(container: HTMLElement): Promise<HTMLButtonElement[]> {
  await fireEvent.click(container.querySelector('.more-wrap button') as HTMLButtonElement)
  await tick()
  const opener = container.querySelector('.more-menu .more-item') as HTMLButtonElement
  await fireEvent.click(opener)
  await vi.waitFor(() => {
    if (container.querySelectorAll('.more-sub .more-item').length < 2) throw new Error('the device list has not arrived')
  })
  return Array.from(container.querySelectorAll('.more-sub .more-item')) as HTMLButtonElement[]
}

/** The label and the size shown on one row. */
function rowText(el: HTMLElement): string {
  const name = el.querySelector('.more-txt')?.textContent?.trim() ?? ''
  const size = el.querySelector('.more-val')?.textContent?.trim() ?? ''
  return size ? `${name} (${size})` : name
}

describe('the device picker', () => {
  it('builds its rows from the list Go owns', async () => {
    withBrowserTab()
    const { container } = render(Workbench)

    const rows = (await sizeRows(container)).map(rowText)

    // The first row is เต็มแผง, which is the absence of a device rather than
    // one of them, and so is the only row this file is allowed to know about.
    expect(rows.slice(1)).toEqual(['iPhone SE (375×667)', 'Desktop (1280×800)'])
  })

  it('tells the engine which device, not just the window', async () => {
    withBrowserTab()
    const { container } = render(Workbench)

    const rows = await sizeRows(container)
    await fireEvent.click(rows[1])
    await tick()

    // The half that was missing. Without this the pane is a narrow desktop.
    expect(vi.mocked(BrowserSetDevice)).toHaveBeenCalledWith('web-1', 'iPhone SE')
    // And the half that was already there: the window really is phone shaped.
    expect(workbench.tabs[0].viewport).toEqual({ name: 'iPhone SE', w: 375, h: 667, dpr: 2, mobile: true })
  })

  it('going back to เต็มแผง clears the emulation too', async () => {
    withBrowserTab()
    const { container } = render(Workbench)

    // Picking a size closes the menu, so going back means opening it again —
    // which is the gesture a person makes too.
    await fireEvent.click((await sizeRows(container))[1])
    await tick()
    await fireEvent.click((await sizeRows(container))[0])
    await tick()

    expect(vi.mocked(BrowserSetDevice).mock.calls.at(-1)).toEqual(['web-1', ''])
    expect(workbench.tabs[0].viewport).toBeUndefined()
  })
})
