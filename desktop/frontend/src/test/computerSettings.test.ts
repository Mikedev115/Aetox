import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import {
  ComputerControlOn, SetComputerControlOn, GrantedComputerApps, RevokeComputerApp,
} from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

// การใช้คอมพิวเตอร์ — the register, tested for the two things it is easy to get
// wrong and that neither TypeScript nor svelte-check would catch.
//
// The first is what the page IS. The direction doc (§4.2) warns that reading it
// as a list of applications produces a dead register: rows are REACHES, and a
// row exists because a mechanism reaches it. So the three rows with no mechanism
// behind them have to stay on screen saying why, rather than being hidden until
// they work. connections.go carries the same rule in a long comment, learned
// twice the hard way.
//
// The second is the default. This ships off, and a page that drew the switch on
// because a binding threw would be handing out a permission nobody granted.

const openSection = async (container: HTMLElement, label: string) => {
  const items = Array.from(container.querySelectorAll('.settings-nav-item'))
  const item = items.find((el) => el.textContent?.trim() === label)
    ?? items.find((el) => el.textContent?.includes(label))
  if (!item) throw new Error(`nav item "${label}" not found`)
  await fireEvent.click(item)
}

beforeEach(() => {
  cockpit.settingsIntent = null
  vi.mocked(ComputerControlOn).mockResolvedValue(false)
  vi.mocked(GrantedComputerApps).mockResolvedValue([])
})

describe('the computer-use page', () => {
  it('ships with the switch off', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    await waitFor(() => expect(container.textContent).toContain('โปรแกรมใดก็ได้'))
    const box = container.querySelector('.mswitch input') as HTMLInputElement
    expect(box.checked).toBe(false)
  })

  it('keeps the reaches that do not work yet on the page, saying why', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    await waitFor(() => expect(container.textContent).toContain('Google Chrome'))
    // A row that vanishes in its broken state is a dead end, not a tidy UI.
    expect(container.textContent).toContain('Microsoft Edge')
    expect(container.textContent).toContain('Microsoft Excel')
    // And each says WHY, rather than being greyed out with no reason.
    expect(container.textContent).toContain('ส่วนขยายเบราว์เซอร์')
    expect(container.textContent).toContain('สเปรดชีต')
  })

  it('turns the reach on through the binding rather than optimistically', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')
    await waitFor(() => expect(container.textContent).toContain('โปรแกรมใดก็ได้'))

    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    await fireEvent.change(container.querySelector('.mswitch input') as HTMLInputElement)

    await waitFor(() => expect(vi.mocked(SetComputerControlOn)).toHaveBeenCalledWith(true))
    // Re-read after writing: the switch shows what Go says, never what the
    // click assumed. A preference file that would not write leaves the control
    // reading the truth instead of a state nothing persisted.
    await waitFor(() => expect(vi.mocked(ComputerControlOn).mock.calls.length).toBeGreaterThan(1))
  })

  it('shows the programs the user has allowed, and lets them be taken back', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    vi.mocked(GrantedComputerApps).mockResolvedValue(['notepad', 'winword'])

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // This list is the half neither rival has: their app approvals expire with
    // the session, so there is nothing to draw. Ours are written down precisely
    // so this can exist.
    await waitFor(() => expect(container.textContent).toContain('notepad'))
    expect(container.textContent).toContain('winword')

    const revoke = screen.getAllByText('ถอนสิทธิ์')
    expect(revoke.length).toBe(2)
    await fireEvent.click(revoke[0])
    await waitFor(() => expect(vi.mocked(RevokeComputerApp)).toHaveBeenCalledWith('notepad'))
  })

  it('says nothing is allowed yet rather than showing an empty box', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    await waitFor(() => expect(container.textContent).toContain('ยังไม่ได้อนุญาตโปรแกรมไหน'))
  })

  it('hides the granted list while the reach is off, because it grants nothing', async () => {
    vi.mocked(GrantedComputerApps).mockResolvedValue(['notepad'])
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    await waitFor(() => expect(container.textContent).toContain('โปรแกรมใดก็ได้'))
    // The master switch is off, so a list of programs Aetox "may" drive would
    // be describing a permission that is not in force.
    expect(container.textContent).not.toContain('ถอนสิทธิ์')
  })
})
