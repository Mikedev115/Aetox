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
  // The session review writes USER.md, so since 14 ก.ย. 2026 its switch and
  // its button live on เกี่ยวกับคุณ, with the file — not on การเรียนรู้.
  it('runs and toggles the session review from เกี่ยวกับคุณ', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'เกี่ยวกับคุณ')
    await waitFor(() => expect(screen.getByText(/ทบทวนเซสชันปัจจุบันเดี๋ยวนี้/)).toBeTruthy())

    vi.mocked(RunSessionReview).mockResolvedValue(1)
    await fireEvent.click(screen.getByText(/ทบทวนเซสชันปัจจุบันเดี๋ยวนี้/))
    expect(RunSessionReview).toHaveBeenCalled()

    // The page's one switch.
    const switches = container.querySelectorAll('.mswitch input') as NodeListOf<HTMLInputElement>
    expect(switches.length).toBe(1)
    await fireEvent.click(switches[0])
    expect(SetSessionReviewAuto).toHaveBeenCalledWith(true)
    // And it is gone from การเรียนรู้.
    await openSection(container, LEARNING)
    expect(screen.queryByText(/ทบทวนเซสชันปัจจุบันเดี๋ยวนี้/)).toBeNull()
  })

  it('renders recurring requests on the habits subtab', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, LEARNING)
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
