// ชุดคำสั่ง — the heading of ห้องความสามารถ a user writes themselves (14 ก.ย.
// 2026), two rows: the preset gallery and editor that were ตั้งค่า › ชุดคำสั่ง,
// and the recurring requests that were the Habits tab of ตั้งค่า › การเรียนรู้.
// The tests came from Settings.test.ts and sessionReview.test.ts with them.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Capability from '../lib/Capability.svelte'
import {
  ListPromptPresets, SavePromptPreset, ListRecurringRequests, DismissRecurringRequest, SynthesizeHabit,
  ListMCPServers, ListSubagentProfiles, PlacementTargets, ListTools,
} from './mocks/wailsApp'
import { cockpit } from '../lib/stores/cockpit.svelte'

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.activeView = 'capability'
  cockpit.capabilityIntent = null
  vi.mocked(ListMCPServers).mockResolvedValue([] as any)
  vi.mocked(ListSubagentProfiles).mockResolvedValue([] as any)
  vi.mocked(PlacementTargets).mockResolvedValue([] as any)
  vi.mocked(ListTools).mockResolvedValue([] as any)
  vi.mocked(ListPromptPresets).mockResolvedValue([
    // Bundled presets ship cover art; a user preset may have none yet.
    { name: 'landing', description: 'สร้างแลนดิ้งเพจ', body: 'ทำแลนดิ้งเพจ $ARGUMENTS', path: '', builtin: true, image: 'data:image/svg+xml;base64,PHN2Zy8+' },
    { name: 'mine', description: 'ชุดคำสั่งของผม', body: 'ของผมเอง', path: 'C:/prompts/mine.md', builtin: false, image: '' },
  ] as any)
  vi.mocked(ListRecurringRequests).mockResolvedValue([
    { text: 'เช็คกำลังไฟ GPU', count: 3, normalized: 'กินไฟ gpu' },
  ] as any)
})

const rail = () => Array.from(document.querySelectorAll<HTMLElement>('.settings-nav-item'))
const open = async (row: string) => {
  const r = render(Capability, { onClose: () => {} })
  await waitFor(() => expect(PlacementTargets).toHaveBeenCalled())
  await fireEvent.click(rail().find((x) => x.textContent?.includes(row))!)
  return r
}
const openPresets = () => open('ชุดคำสั่งของคุณ')
const openHabits = () => open('คำสั่งที่สั่งบ่อย')

describe('where the heading is', () => {
  // Between the tool register and the two reaches: after what came with the
  // app, before what goes out of it.
  it('is one heading of two rows, after เครื่องมือในตัว and before การใช้คอมพิวเตอร์', async () => {
    render(Capability, { onClose: () => {} })
    await waitFor(() => expect(PlacementTargets).toHaveBeenCalled())
    const groups = Array.from(document.querySelectorAll('.settings-nav .settings-group-label')).map((x) => x.textContent?.trim())
    expect(groups.indexOf('ชุดคำสั่ง')).toBe(groups.indexOf('เครื่องมือในตัว') + 1)
    expect(groups.indexOf('การใช้คอมพิวเตอร์')).toBe(groups.indexOf('ชุดคำสั่ง') + 1)
    const rows = rail().map((x) => x.textContent?.trim())
    expect(rows.indexOf('คำสั่งที่สั่งบ่อย')).toBe(rows.indexOf('ชุดคำสั่งของคุณ') + 1)
  })
})

describe('ชุดคำสั่งของคุณ', () => {
  it('is a card gallery, badging the bundled ones', async () => {
    const { container } = await openPresets()

    await waitFor(() => expect(container.querySelectorAll('.pp-card').length).toBe(3)) // 2 presets + "new"
    expect(screen.getByText('สร้างแลนดิ้งเพจ')).toBeTruthy()
    expect(screen.getAllByText('มากับแอป')).toHaveLength(1)
    // Shipped cover renders as a real image; the one without falls back to the
    // generated cover rather than a broken <img>.
    expect(container.querySelectorAll('.pp-cover img').length).toBe(1)
    expect(container.querySelectorAll('.pp-cover .pp-mono').length).toBe(1)
    // The door that does not go stale: a card that asks the assistant.
    expect(screen.getByText('ให้ผู้ช่วยเขียนชุดคำสั่งให้')).toBeTruthy()
  })

  it('clicking a preset card opens its full text for editing', async () => {
    const { container } = await openPresets()
    await waitFor(() => expect(container.querySelectorAll('.pp-card').length).toBe(3))

    const card = Array.from(container.querySelectorAll('.pp-card'))
      .find((el) => el.textContent?.includes('/landing'))!
    await fireEvent.click(card)

    const body = container.querySelector('.pp-textarea') as HTMLTextAreaElement
    expect(body).toBeTruthy()
    expect(body.value).toBe('ทำแลนดิ้งเพจ $ARGUMENTS')
    // A bundled preset says what saving will do rather than refusing the edit.
    expect(screen.getByText(/สร้างเป็นของคุณทับไว้/)).toBeTruthy()
    // Its name is fixed; a new preset is where you get to choose one.
    expect((container.querySelector('.pp-field input.ctrl') as HTMLInputElement).disabled).toBe(true)
  })

  // An empty 300px box tells you nothing about what belongs in it.
  it('a new preset opens on a starter skeleton, not a blank box', async () => {
    const { container } = await openPresets()
    await waitFor(() => expect(container.querySelector('.pp-new')).toBeTruthy())

    await fireEvent.click(container.querySelector('.pp-new')!)
    const body = container.querySelector('.pp-textarea') as HTMLTextAreaElement
    expect(body.value).toContain('$ARGUMENTS')
    expect(body.value.length).toBeGreaterThan(80)
    expect(body.placeholder).toBeTruthy()
    // The one token a preset cannot work without gets its own button.
    expect(screen.getByText('+ $ARGUMENTS')).toBeTruthy()
  })

  // Leaving the editor with unsaved work is the same class of loss as a
  // delete, so it goes through the room's confirm gate rather than just
  // closing.
  it('asks before leaving an edited preset', async () => {
    const { container } = await openPresets()
    await waitFor(() => expect(container.querySelector('.pp-new')).toBeTruthy())
    await fireEvent.click(container.querySelector('.pp-new')!)
    const body = container.querySelector('.pp-textarea') as HTMLTextAreaElement
    await fireEvent.input(body, { target: { value: 'something new' } })

    await fireEvent.click(screen.getByText('กลับไปหน้ารวม'))
    await waitFor(() => expect(screen.getByText('ทิ้งสิ่งที่แก้ไว้?')).toBeTruthy())
    // Still in the editor until the dialog is answered.
    expect(container.querySelector('.pp-textarea')).toBeTruthy()
  })

  it('saves through the binding and returns to the gallery', async () => {
    const { container } = await openPresets()
    await waitFor(() => expect(container.querySelector('.pp-new')).toBeTruthy())
    await fireEvent.click(container.querySelector('.pp-new')!)

    await fireEvent.input(container.querySelector('.pp-field input.ctrl')!, { target: { value: 'deploy' } })
    await fireEvent.click(screen.getByText('บันทึก'))
    await waitFor(() => expect(vi.mocked(SavePromptPreset)).toHaveBeenCalled())
    expect(vi.mocked(SavePromptPreset).mock.calls[0][0]).toBe('deploy')
    await waitFor(() => expect(container.querySelector('.pp-textarea')).toBeNull())
  })
})

describe('คำสั่งที่สั่งบ่อย', () => {
  it('lists the recurring requests with their counts, and dismisses one', async () => {
    await openHabits()

    await waitFor(() => expect(screen.getByText('เช็คกำลังไฟ GPU')).toBeTruthy())
    expect(screen.getByText(/3 ครั้ง/)).toBeTruthy()

    const dismissBtns = screen.getAllByText(/ลบ \/ ละเว้น/)
    expect(dismissBtns.length).toBeGreaterThan(0)
    await fireEvent.click(dismissBtns[0])
    expect(DismissRecurringRequest).toHaveBeenCalledWith('กินไฟ gpu', 'เช็คกำลังไฟ GPU')
    // Gone from the list without a round trip.
    await waitFor(() => expect(screen.queryByText('เช็คกำลังไฟ GPU')).toBeNull())
  })

  // The thing a person does with a habit is stop typing it: the row one up,
  // opened on a new preset whose body is the request.
  it('turns a habit into a new preset in the editor one row up', async () => {
    const { container } = await openHabits()
    await waitFor(() => expect(screen.getByText('เช็คกำลังไฟ GPU')).toBeTruthy())

    await fireEvent.click(screen.getByText(/แปลงเป็นชุดคำสั่ง/))
    await waitFor(() => expect(container.querySelector('.pp-textarea')).toBeTruthy())
    expect((container.querySelector('.pp-textarea') as HTMLTextAreaElement).value).toBe('เช็คกำลังไฟ GPU')
    expect((container.querySelector('.pp-field input.ctrl') as HTMLInputElement).value).toBe('เช็คกำลังไฟ-gpu')
    expect(container.querySelector('.settings-nav-item.active')?.textContent?.trim()).toBe('ชุดคำสั่งของคุณ')
  })

  // The draft lands in the learning queue, not on this page — so the page
  // says where it went instead of reloading a list that will not show it.
  it('says where a synthesized draft went', async () => {
    vi.mocked(SynthesizeHabit).mockResolvedValue(7 as any)
    await openHabits()
    await waitFor(() => expect(screen.getByText('เช็คกำลังไฟ GPU')).toBeTruthy())

    await fireEvent.click(screen.getByText(/วิเคราะห์และสร้างสกิล/))
    await waitFor(() => expect(SynthesizeHabit).toHaveBeenCalledWith('', 'เช็คกำลังไฟ GPU'))
    await waitFor(() => expect(screen.getByText('ส่งข้อเสนอไปยังรายการรออนุมัติแล้ว')).toBeTruthy())
  })

  it('says so when nothing recurs yet', async () => {
    vi.mocked(ListRecurringRequests).mockResolvedValue([] as any)
    await openHabits()
    await waitFor(() => expect(screen.getByText(/ยังไม่พบคำสั่งที่สั่งซ้ำ/)).toBeTruthy())
  })
})
