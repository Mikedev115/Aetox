import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { personas, addPersona, clearPersonas, setAvatarPrefs, resetAvatarPrefs } from '../lib/mascot/avatarPrefs.svelte'
import { SHELL, ACCENT } from '../lib/mascot/palette'
import { FACE, TOP } from '../lib/mascot/parts'
import {
  ListSubagentProfiles, ReadSubagentProfile, SaveAgentProfile, LearnedEntries,
  DelegateSwitches, ListMCPServers, ListExternalSkills, ListTools, ListChairs,
} from './mocks/wailsApp'

// ตั้งค่า › เอเจน › the look: the mascot's four identity dials and the badge,
// drawn as whole robots in the picker, written to the agent's own .md, and a
// persona the user saved on the avatar page worn in one press. What is
// guarded is the seam between this page and the file — what a cell does to
// the draft, what a save writes, what the old fields turn into — and the
// rule that no cell in a picker moves.

// The look lives on its own tab of the editor (owner, 12 ก.ย.: "ควรทำหน้า
// อวตารแยก") — every test here opens the editor and steps onto it.
const openEditor = async () => {
  cockpit.settingsIntent = { section: 'team', agent: 'backend' }
  const r = render(Settings, { onClose: () => {} })
  await waitFor(() => expect(screen.getByRole('tablist', { name: 'ตั้งค่าพนักงาน' })).toBeTruthy())
  await fireEvent.click(screen.getByRole('tab', { name: 'อวตาร' }))
  await waitFor(() => expect(r.container.querySelector('#ag-panel-avatar.on')).toBeTruthy())
  return r
}
const savedFile = async (): Promise<string> => {
  await fireEvent.click(screen.getByText('บันทึก'))
  await waitFor(() => expect(vi.mocked(SaveAgentProfile)).toHaveBeenCalled())
  return vi.mocked(SaveAgentProfile).mock.calls.at(-1)![1]
}
const rowCells = (container: HTMLElement, eyebrow: string): HTMLButtonElement[] => {
  const field = Array.from(container.querySelectorAll('.pp-field')).find((f) => f.querySelector('.eyebrow')?.textContent === eyebrow)!
  return Array.from(field.querySelectorAll<HTMLButtonElement>('.ag-part, .ag-icon'))
}

beforeEach(() => {
  vi.unstubAllGlobals()
  localStorage.clear()
  resetAvatarPrefs()
  clearPersonas()
  cockpit.settingsIntent = null
  vi.mocked(DelegateSwitches).mockRejectedValue(new Error('unavailable'))
  vi.mocked(ListMCPServers).mockResolvedValue([])
  vi.mocked(ListExternalSkills).mockResolvedValue([])
  vi.mocked(ListTools).mockResolvedValue([])
  vi.mocked(LearnedEntries).mockResolvedValue([])
  vi.mocked(ListSubagentProfiles).mockResolvedValue([
    { name: 'backend', description: 'งานหลังบ้าน', prompt: 'role', builtin: false, desk: 'specialized', icon: 'terminal', accent: 'copper' },
  ] as any)
  vi.mocked(ListChairs).mockResolvedValue([{ name: 'backend', icon: 'terminal', accent: 'copper' }] as any)
  vi.mocked(ReadSubagentProfile).mockResolvedValue('---\ndescription: งานหลังบ้าน\n---\nโค้ดหลังบ้าน' as any)
})

describe('the agent look editor', () => {
  it('offers every catalogue row as a still robot, and the badges as glyphs', async () => {
    const { container } = await openEditor()
    // One "choose for me" cell ahead of the catalogue on each row.
    expect(rowCells(container, 'สีตัว').length).toBe(SHELL.length + 1)
    expect(rowCells(container, 'สี').length).toBe(ACCENT.length + 1)
    expect(rowCells(container, 'ไฟบนหัว').length).toBe(TOP.length + 1)
    expect(rowCells(container, 'หน้าประจำตัว').length).toBe(FACE.filter((f) => f.identity).length + 1)
    expect(rowCells(container, 'ไอคอนบนหู').length).toBeGreaterThan(7)
    // Every cell in every row is the whole mascot, and none of them moves.
    const cells = container.querySelectorAll('.ag-part .mascot')
    expect(cells.length).toBeGreaterThan(SHELL.length + ACCENT.length + TOP.length)
    for (const m of cells) expect(m.classList.contains('still')).toBe(true)
    // The one being faced may breathe.
    expect(container.querySelector('.ag-avatar-stage > .mascot')?.classList.contains('still')).toBe(false)
    // The look is not on ตัวตน any more: the name field is, the rows are not.
    await fireEvent.click(screen.getByRole('tab', { name: 'ตัวตน' }))
    expect(container.querySelector('#ag-panel-identity.on')).toBeTruthy()
    expect(container.querySelector('#ag-panel-identity.on .ag-parts')).toBeNull()
  })

  it('writes what was picked as ids, and nothing that was not', async () => {
    const { container } = await openEditor()
    await fireEvent.click(rowCells(container, 'สีตัว').find((b) => b.title === 'ดำ')!)
    await fireEvent.click(rowCells(container, 'ไฟบนหัว').find((b) => b.title === 'แถบไฟ')!)
    await fireEvent.click(rowCells(container, 'สี').find((b) => b.title === 'ทองแดง')!)
    await fireEvent.click(rowCells(container, 'ไอคอนบนหู').find((b) => b.title === 'search')!)
    const file = await savedFile()
    expect(file).toContain('shell: dark')
    expect(file).toContain('top: bar')
    expect(file).toContain('accent: copper')
    expect(file).toContain('icon: search')
    expect(file).not.toMatch(/^face:/m)
    expect(file).not.toMatch(/^hue:/m)
  })

  // A file that still names a haircut loads, and loses the line on save —
  // the migration off the cartoon face is the next time its owner presses
  // บันทึก. A degree it carries is kept until a colour is picked, because
  // rig.ts lets a degree win over an accent.
  it('drops hair and accessory on save, and keeps a hue until a colour is picked', async () => {
    vi.mocked(ReadSubagentProfile).mockResolvedValue(
      '---\ndescription: งานหลังบ้าน\nhair: sidePart\naccessory: glasses\nhue: 210\n---\nโค้ดหลังบ้าน' as any,
    )
    const first = await openEditor()
    expect(screen.getByText(/210°/)).toBeTruthy()
    let file = await savedFile()
    expect(file).not.toContain('hair:')
    expect(file).not.toContain('accessory:')
    expect(file).toContain('hue: 210')

    // Saving closes the editor; open it again and pick a colour.
    first.unmount()
    const { container } = await openEditor()
    await fireEvent.click(rowCells(container, 'สี').find((b) => b.title === 'มินต์')!)
    file = await savedFile()
    expect(file).toContain('accent: mint')
    expect(file).not.toMatch(/^hue:/m)
  })

  it('wears a saved persona in one press, and keeps the agent\'s own badge', async () => {
    addPersona()
    setAvatarPrefs({ shell: 'dark', accent: 'gold', top: 'bar', face: 'focused' })
    addPersona()
    expect(personas.slots[1]).toBeTruthy()
    const { container } = await openEditor()
    await fireEvent.click(rowCells(container, 'ไอคอนบนหู').find((b) => b.title === 'search')!)
    // A card per saved look, no empties: the second wears the persona with
    // THIS agent's badge, and its ใช้ button puts the four dials on the draft.
    const slots = container.querySelectorAll('.ag-slot')
    expect(slots.length).toBe(2)
    expect(container.querySelectorAll('.ag-slot.empty').length).toBe(0)
    expect(slots[1].querySelector('.mascot .ms-earL .ms-badge')?.innerHTML).toContain('<circle cx="11" cy="11" r="8">')
    const use = screen.getByRole('button', { name: 'บุคลิก 2 — ใช้' })
    await fireEvent.click(use)
    expect(slots[1].classList.contains('worn')).toBe(true)
    expect((use as HTMLButtonElement).disabled).toBe(true)
    const file = await savedFile()
    expect(file).toContain('shell: dark')
    expect(file).toContain('accent: gold')
    expect(file).toContain('top: bar')
    expect(file).toContain('face: focused')
    expect(file).toContain('icon: search')
  })

  it('says when nothing is saved, and where to save one', async () => {
    const { container } = await openEditor()
    expect(container.querySelectorAll('.ag-slot').length).toBe(1)
    expect(container.querySelectorAll('.ag-slot.empty').length).toBe(1)
    expect(screen.getByText(/บันทึกและแก้บุคลิกได้ที่/)).toBeTruthy()
  })

  // The preview across the app: the same robot at the office's, the chat's
  // and the composer's size, and the chat one wearing the card's five words.
  it('previews the card states as poses', async () => {
    const { container } = await openEditor()
    await fireEvent.click(await screen.findByRole('button', { name: /ดูในบริบทต่างๆ/ }))
    await waitFor(() => expect(container.querySelector('.ag-contexts-panel')).not.toBeNull())
    await fireEvent.click(screen.getByRole('button', { name: 'กำลังคิด…' }))
    expect(container.querySelector('.ag-context-card .mascot.pose-thinking')).not.toBeNull()
    await fireEvent.click(screen.getByRole('button', { name: 'กำลังทำ…' }))
    const typing = container.querySelector('.ag-context-card .mascot.pose-typing')!
    expect(typing).not.toBeNull()
    expect(typing.classList.contains('still')).toBe(false)
    await fireEvent.click(screen.getByRole('button', { name: 'ผิดพลาด' }))
    expect(container.querySelector('.ag-context-card .mascot.pose-error')).not.toBeNull()
    await fireEvent.click(screen.getByRole('button', { name: 'ปิดใช้งาน' }))
    expect(container.querySelector('.ag-context-card .mascot.off')).not.toBeNull()
  })
})
