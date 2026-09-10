// Test for Session Review and Habits UI in Settings.svelte
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import {
  LearningEnabled, ListPendingChanges, ListDecidedChanges,
  SessionReviewAuto, SetSessionReviewAuto, RunSessionReview, ListRecurringRequests,
  DismissRecurringRequest,
} from './mocks/wailsApp'

const openSection = async (container: HTMLElement, label: string) => {
  const item = Array.from(container.querySelectorAll('.settings-nav-item'))
    .find((el) => el.textContent?.includes(label))
  if (!item) throw new Error(`nav item "${label}" not found`)
  await fireEvent.click(item)
}

const LEARNING = 'การเรียนรู้'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(LearningEnabled).mockResolvedValue(true)
  vi.mocked(SessionReviewAuto).mockResolvedValue(false)
  vi.mocked(ListPendingChanges).mockResolvedValue([] as any)
  vi.mocked(ListDecidedChanges).mockResolvedValue([] as any)
  vi.mocked(ListRecurringRequests).mockResolvedValue([
    { text: 'เช็คกำลังไฟ GPU', count: 3, normalized: 'กินไฟ gpu' },
  ] as any)
})

describe('session review and habits in settings', () => {
  it('renders recurring requests and toggles session review auto', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, LEARNING)

    // Verify session review button is present on memory subtab
    await waitFor(() => expect(screen.getByText(/ทบทวนเซสชันปัจจุบันเดี๋ยวนี้/)).toBeTruthy())

    // Click RunSessionReview
    vi.mocked(RunSessionReview).mockResolvedValue(1)
    await fireEvent.click(screen.getByText(/ทบทวนเซสชันปัจจุบันเดี๋ยวนี้/))
    expect(RunSessionReview).toHaveBeenCalled()

    // Toggle session review auto
    const switches = container.querySelectorAll('.mswitch input') as NodeListOf<HTMLInputElement>
    expect(switches.length).toBeGreaterThanOrEqual(2)
    // The second switch is session review auto
    await fireEvent.click(switches[1])
    expect(SetSessionReviewAuto).toHaveBeenCalledWith(true)

    // Switch to Habits subtab
    const habitsTabBtn = screen.getByText(/คำสั่งที่สั่งบ่อย \(Habits\)/)
    await fireEvent.click(habitsTabBtn)

    // Verify habits card appears
    await waitFor(() => expect(screen.getByText('เช็คกำลังไฟ GPU')).toBeTruthy())
    expect(screen.getByText(/3 ครั้ง/)).toBeTruthy()

    // Test dismiss button
    const dismissBtns = screen.getAllByText(/ลบ \/ ละเว้น/)
    expect(dismissBtns.length).toBeGreaterThan(0)
    await fireEvent.click(dismissBtns[0])
    expect(DismissRecurringRequest).toHaveBeenCalledWith('กินไฟ gpu', 'เช็คกำลังไฟ GPU')
  })
})
