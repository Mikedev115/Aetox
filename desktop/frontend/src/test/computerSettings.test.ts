import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import {
  ComputerControlOn, SetComputerControlOn, GrantedComputerApps, RevokeComputerApp,
  OpenComputerApps, AllowComputerApp, BrowseForComputerApp,
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
  vi.mocked(OpenComputerApps).mockResolvedValue([])
})

describe('the computer-use page', () => {
  it('ships with the switch off', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    await waitFor(() => expect(container.textContent).toContain('อนุญาตให้ควบคุมคอมพิวเตอร์'))
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
    await waitFor(() => expect(container.textContent).toContain('อนุญาตให้ควบคุมคอมพิวเตอร์'))

    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    await fireEvent.change(container.querySelector('.mswitch input') as HTMLInputElement)

    await waitFor(() => expect(vi.mocked(SetComputerControlOn)).toHaveBeenCalledWith(true))
    // Re-read after writing: the switch shows what Go says, never what the
    // click assumed. A preference file that would not write leaves the control
    // reading the truth instead of a state nothing persisted.
    await waitFor(() => expect(vi.mocked(ComputerControlOn).mock.calls.length).toBeGreaterThan(1))
  })

  it('lets the user pick a program from the windows that are open', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    vi.mocked(OpenComputerApps).mockResolvedValue([
      { name: 'notepad', title: 'บันทึกย่อ', allowed: false, blocked: '', warn: '', icon: '' },
      { name: 'chrome', title: 'หน้าเว็บ', allowed: false, blocked: 'browser', warn: '', icon: '' },
    ] as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // Chosen here, with nothing waiting, rather than answered in a hurry while
    // an agent is parked on the reply.
    await waitFor(() => expect(container.textContent).toContain('notepad'))
    await fireEvent.click(screen.getByText('อนุญาต'))
    await waitFor(() => expect(vi.mocked(AllowComputerApp)).toHaveBeenCalledWith('notepad'))
  })

  it('shows a browser in the list and says which tool does it instead', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    vi.mocked(OpenComputerApps).mockResolvedValue([
      { name: 'chrome', title: 'หน้าเว็บ', allowed: false, blocked: 'browser', warn: '', icon: '' },
    ] as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // Shown rather than hidden: a user who cannot find Chrome here learns
    // nothing, one who finds it with the reason learns the shape of the product.
    await waitFor(() => expect(container.textContent).toContain('chrome'))
    expect(container.textContent).toContain('`browser`')
    // And it offers no button, because there is nothing to grant.
    expect(screen.queryByText('อนุญาต')).toBeNull()
  })

  it('keeps an allowed program visible after its window closes', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    vi.mocked(GrantedComputerApps).mockResolvedValue(['winword'])
    vi.mocked(OpenComputerApps).mockResolvedValue([])

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // The grant is still in force whether or not the program is running, so
    // hiding it would hide a permission the user still has.
    await waitFor(() => expect(container.textContent).toContain('winword'))
    await fireEvent.click(screen.getByText('ถอนสิทธิ์'))
    await waitFor(() => expect(vi.mocked(RevokeComputerApp)).toHaveBeenCalledWith('winword'))
  })

  it('puts each program own icon on its row', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    vi.mocked(OpenComputerApps).mockResolvedValue([
      { name: 'notepad', title: 'บันทึกย่อ', allowed: false, blocked: '', warn: '', icon: 'data:image/png;base64,AAA' },
    ] as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // The picture the taskbar shows, so a person picks by recognising rather
    // than by reading a filename.
    await waitFor(() => {
      const img = container.querySelector('img.prog-icon') as HTMLImageElement
      expect(img?.src).toContain('data:image/png;base64,AAA')
    })
  })

  it('keeps the row aligned when a program has no icon to give', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    vi.mocked(OpenComputerApps).mockResolvedValue([
      { name: 'oldapp', title: 'โปรแกรมเก่า', allowed: false, blocked: '', warn: '', icon: '' },
    ] as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // A placeholder of the same size, not nothing: a list whose rows shift by
    // twenty pixels depending on whether an icon could be read is a list that
    // reads as broken.
    await waitFor(() => expect(container.querySelector('.prog-icon-none')).toBeTruthy())
  })

  it('can add a program that is not running, from the disk', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // A list of what happens to be open is not the whole question: a person
    // setting this up thinks in terms of the programs they use, and making them
    // launch one first in order to permit it is working around the interface.
    await waitFor(() => expect(screen.getByText('เลือกจากเครื่อง…')).toBeTruthy())
    await fireEvent.click(screen.getByText('เลือกจากเครื่อง…'))
    await waitFor(() => expect(vi.mocked(BrowseForComputerApp)).toHaveBeenCalled())
  })

  it('does not call a browser row "not available yet"', async () => {
    vi.mocked(ComputerControlOn).mockResolvedValue(true)
    vi.mocked(OpenComputerApps).mockResolvedValue([
      { name: 'chrome', title: 'หน้าเว็บ', allowed: false, blocked: 'browser', warn: '', icon: '' },
    ] as any)

    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    // It is not a missing feature. It is a job another tool already does
    // better, and a badge saying otherwise contradicts the sentence beside it.
    await waitFor(() => expect(container.textContent).toContain('ใช้เครื่องมืออื่น'))
  })

  it('offers nothing to pick while the switch is off', async () => {
    vi.mocked(GrantedComputerApps).mockResolvedValue(['notepad'])
    vi.mocked(OpenComputerApps).mockResolvedValue([
      { name: 'notepad', title: 'บันทึกย่อ', allowed: true, blocked: '', warn: '', icon: '' },
    ] as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'การใช้คอมพิวเตอร์')

    await waitFor(() => expect(container.textContent).toContain('อนุญาตให้ควบคุมคอมพิวเตอร์'))
    // Off means the model does not have the tool at all, so a list of programs
    // it "may" drive would be describing a permission that is not in force.
    expect(container.textContent).not.toContain('ถอนสิทธิ์')
    expect(screen.queryByText('อนุญาต')).toBeNull()
  })
})
