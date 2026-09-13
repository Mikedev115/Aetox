// เกี่ยวกับคุณ (14 ก.ย. 2026): the one layer every desk and every agent reads
// the same — the person. Your name, USER.md and the session review that writes
// it, in one place, and NOT on การเรียนรู้ any more. context.md sat here for a
// morning and went back to the head's own set (ตัวหลัก › ตัวตน, mainHeads.test.ts).
// What is pinned here is the page's own field; the memory block and the review
// switch are pinned where they were tested before, repointed
// (learningReview.test.ts, sessionReview.test.ts).
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import { UserName, SetUserName, LearnedScopeInfos } from './mocks/wailsApp'
import { profile } from '../lib/stores/profile.svelte'

const openSection = async (container: HTMLElement, label: string) => {
  const item = Array.from(container.querySelectorAll('.settings-nav-item')).find((el) => el.textContent?.includes(label))
  if (!item) throw new Error(`nav item "${label}" not found`)
  await fireEvent.click(item)
}

beforeEach(() => {
  vi.clearAllMocks()
  profile.name = ''
  profile.loaded = false
  vi.mocked(LearnedScopeInfos).mockResolvedValue([] as any)
  vi.mocked(UserName).mockResolvedValue('mike')
})

describe('เกี่ยวกับคุณ', () => {
  it('sits under ส่วนบุคคล and shows the name the footer knows', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    const labels = Array.from(container.querySelectorAll('.settings-nav-item')).map((n) => (n.textContent ?? '').trim())
    expect(labels).toContain('เกี่ยวกับคุณ')
    await openSection(container, 'เกี่ยวกับคุณ')
    const name = await waitFor(() => container.querySelector('input[aria-label="ชื่อของคุณ"]') as HTMLInputElement)
    await waitFor(() => expect(name.value).toBe('mike'))
    // Written the footer's way: through the one store, on change.
    await fireEvent.input(name, { target: { value: 'Mike D' } })
    await fireEvent.change(name)
    expect(SetUserName).toHaveBeenCalledWith('Mike D')
    expect(profile.name).toBe('Mike D')
  })
})
