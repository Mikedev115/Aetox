// The + menu (Palette.svelte, §257): one door for everything that goes into
// the message. Three things used to be three buttons — attach, "/" presets,
// Ctrl+K "everything" — and the third repeated the first two and then the
// model chip beside it (owner, 12 ก.ย.: "ปุ่มนี้มันทำงานซ้ำซ้อน"). These tests
// hold what the merge promised: every group is here, the search runs across
// all of them, a narrowing "/" or "@" shows only its group and can be taken
// off again, and nothing that belongs to the model chip is in this list.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Palette from '../lib/Palette.svelte'
import { setLocale } from '../lib/i18n.svelte'
import { workbench } from '../lib/stores/workbench.svelte'
import { ListPromptPresets, ListChairs } from './mocks/wailsApp'

const noop = () => {}
const base = { oninsert: noop, onmention: noop, onattach: noop, onmic: noop, onclose: noop }

beforeEach(() => {
  setLocale('th')
  workbench.tabs = []
  vi.mocked(ListPromptPresets).mockResolvedValue([
    { name: 'landing', description: 'แลนดิ้งเพจไฟล์เดียว', path: '', builtin: true },
    { name: 'review', description: 'รีวิวหาบั๊กจริง', path: '', builtin: true },
  ] as any)
  vi.mocked(ListChairs).mockResolvedValue([
    { name: 'video', description: 'ตัดต่อ ซับ ทำคลิปสั้น' },
    { name: 'doc', description: 'เอกสารและสไลด์' },
  ] as any)
})

describe('the + menu', () => {
  it('lists every group: files as tiles, presets, agents — and the open tabs a drag could stage', async () => {
    workbench.tabs = [
      { id: 'web-1', kind: 'browser', name: 'aetox.dev', url: 'https://aetox.dev' },
      { id: 'web-2', kind: 'browser', name: 'หน้าใหม่' }, // no URL yet: nothing to read
      { id: 'f-1', kind: 'file', name: 'Chat.svelte', path: 'src/Chat.svelte' },
      { id: 'artifacts', kind: 'artifacts', name: 'ผลงาน' }, // not a thing that attaches
    ] as any
    const { container } = render(Palette, base)
    await waitFor(() => expect(screen.getByText('/landing')).toBeTruthy())
    await waitFor(() => expect(screen.getByText('@video')).toBeTruthy())

    // The four kinds, as tiles, each carrying what happens to that kind.
    const tiles = Array.from(container.querySelectorAll('.pal-tile'))
    expect(tiles.map((t) => t.querySelector('.nm')?.textContent)).toEqual(['รูปภาพ', 'เอกสาร', 'วิดีโอ และเสียง', 'ไฟล์อื่น'])
    expect(tiles[1].getAttribute('title')).toContain('docx')

    // Only the tabs Workbench.svelte's canDrag would let go.
    expect(screen.getByText('aetox.dev')).toBeTruthy()
    expect(screen.getByText('Chat.svelte')).toBeTruthy()
    expect(screen.queryByText('หน้าใหม่')).toBeNull()
    expect(screen.queryByText('ผลงาน')).toBeNull()

    // Nothing that belongs to the model chip.
    expect(screen.queryByText('สลับโมเดล…')).toBeNull()
    expect(screen.queryByText('ระดับการอนุมัติ')).toBeNull()
  })

  it('a tile asks for its own dialog filter; a preset writes "/name " into the message', async () => {
    const attached: string[] = []
    const inserted: string[] = []
    const { container } = render(Palette, { ...base, onattach: (g: string) => attached.push(g), oninsert: (s: string) => inserted.push(s) })
    await waitFor(() => expect(screen.getByText('/landing')).toBeTruthy())

    const tiles = Array.from(container.querySelectorAll('.pal-tile'))
    for (const tile of tiles) await fireEvent.click(tile)
    expect(attached).toEqual(['image', 'document', 'media', ''])

    await fireEvent.click(screen.getByText('/landing'))
    // A trailing space: the user types their arguments straight after.
    expect(inserted).toEqual(['/landing '])
  })

  it('an agent addresses the message; mid-turn the group is not offered at all', async () => {
    const mentioned: string[] = []
    render(Palette, { ...base, onmention: (n: string) => mentioned.push(n) })
    await waitFor(() => expect(screen.getByText('@video')).toBeTruthy())
    await fireEvent.click(screen.getByText('@video'))
    expect(mentioned).toEqual(['video'])

    // What is typed mid-turn goes INTO the running turn, which has no door
    // to a worker — same refusal as the "@" roster.
    const { container } = render(Palette, { ...base, mentions: false })
    await waitFor(() => expect(container.querySelector('.pal-tile')).not.toBeNull())
    expect(container.querySelector('[class*="pal-row"] .pal-label')?.textContent).not.toBe('@video')
    expect(Array.from(container.querySelectorAll('.pal-label')).map((l) => l.textContent)).not.toContain('@video')
  })

  it('"/" opens on the presets alone, says so, and Backspace on an empty search widens it again', async () => {
    const { container } = render(Palette, { ...base, focus: 'prompts' })
    await waitFor(() => expect(screen.getByText('/landing')).toBeTruthy())
    expect(container.querySelector('.pal-tile')).toBeNull()
    expect(screen.queryByText('@video')).toBeNull()
    expect(container.querySelector('.pal-mode')?.textContent).toBe('/ ในช่องว่าง')

    await fireEvent.keyDown(container.querySelector('.pal-search')!, { key: 'Backspace' })
    await waitFor(() => expect(container.querySelector('.pal-tile')).not.toBeNull())
    expect(container.querySelector('.pal-mode')).toBeNull()
  })

  it('typing searches across every group at once', async () => {
    const { container } = render(Palette, base)
    await waitFor(() => expect(screen.getByText('@video')).toBeTruthy())

    await fireEvent.input(container.querySelector('.pal-search')!, { target: { value: 'วิดีโอ' } })
    // The media tile (by its name), and nothing that does not say วิดีโอ.
    await waitFor(() => expect(screen.queryByText('/landing')).toBeNull())
    expect(screen.getByText('วิดีโอ และเสียง')).toBeTruthy()
    expect(screen.queryByText('รูปภาพ')).toBeNull()
    expect(screen.queryByText('@video')).toBeNull()

    await fireEvent.input(container.querySelector('.pal-search')!, { target: { value: 'ตัดต่อ' } })
    await waitFor(() => expect(screen.getByText('@video')).toBeTruthy())
  })

  it('the shortcuts sit behind the footer link, read off shortcuts.ts', async () => {
    const { container } = render(Palette, base)
    await waitFor(() => expect(screen.getByText('/landing')).toBeTruthy())
    expect(screen.queryByText('Ctrl+,')).toBeNull()

    await fireEvent.click(container.querySelector('.pal-keys')!)
    await waitFor(() => expect(screen.getByText('Ctrl+,')).toBeTruthy())
    // The row that names this very menu.
    expect(screen.getByText('Ctrl+K')).toBeTruthy()
    // Shortcut rows are readouts, not buttons.
    expect(screen.getByText('Ctrl+,').closest('.pal-row')?.classList.contains('static')).toBe(true)
  })

  it('Escape closes', async () => {
    let closed = false
    const { container } = render(Palette, { ...base, onclose: () => (closed = true) })
    await fireEvent.keyDown(container.querySelector('.pal-search')!, { key: 'Escape' })
    expect(closed).toBe(true)
  })
})
