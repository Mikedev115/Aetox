// Test for the Session Review switch in Settings.svelte. Habits left for
// ห้องความสามารถ › ชุดคำสั่ง on 14 ก.ย. 2026 (promptsPage.test.ts).
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import {
  LearningEnabled, ListPendingChanges, ListDecidedChanges,
  SessionReviewAuto, SetSessionReviewAuto, RunSessionReview,
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
})

describe('session review in settings', () => {
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
})
