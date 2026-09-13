// เกี่ยวกับคุณ (14 ก.ย. 2026): the one layer every desk and every agent reads
// the same — the person. Your name, context.md, USER.md and the session review
// that writes it, in one place, and NOT on การเรียนรู้ or คำสั่งประจำตัว any
// more. What is pinned here is the page's own two fields; the memory block and
// the review switch are pinned where they were tested before, repointed
// (learningReview.test.ts, sessionReview.test.ts).
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'
import Settings from '../lib/Settings.svelte'
import {
  ListIdentityFiles, ReadIdentityFile, SaveIdentityFile, UserName, SetUserName, LearnedScopeInfos,
} from './mocks/wailsApp'
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
  vi.mocked(ListIdentityFiles).mockResolvedValue([{ name: 'identity.md' }, { name: 'context.md' }] as any)
  vi.mocked(ReadIdentityFile).mockResolvedValue('# บริบทผู้ใช้\n\n- ทำ Aetox อยู่')
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

  it('edits context.md as one field and writes it by name', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'เกี่ยวกับคุณ')
    const box = await waitFor(() => container.querySelector('textarea[aria-label="สิ่งที่ควรรู้เกี่ยวกับคุณ"]') as HTMLTextAreaElement)
    await waitFor(() => expect(box.value).toContain('ทำ Aetox อยู่'))
    expect(ReadIdentityFile).toHaveBeenCalledWith('context.md')
    // Nothing to save until it changes.
    const save = Array.from(container.querySelectorAll('.you-save .ctrl-primary'))[0] as HTMLButtonElement
    expect(save.disabled).toBe(true)
    await fireEvent.input(box, { target: { value: '# บริบทผู้ใช้\n\n- ทำ Aetox อยู่\n- เครื่อง Windows' } })
    expect(save.disabled).toBe(false)
    await fireEvent.click(save)
    await waitFor(() => expect(SaveIdentityFile).toHaveBeenCalledWith('context.md', expect.stringContaining('เครื่อง Windows')))
    await waitFor(() => expect(save.disabled).toBe(true))
  })

  // A first-time user has no context.md: the field is empty with the
  // template as its placeholder, and never an error.
  it('opens empty, not broken, when the file does not exist yet', async () => {
    vi.mocked(ListIdentityFiles).mockResolvedValue([] as any)
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'เกี่ยวกับคุณ')
    const box = await waitFor(() => container.querySelector('textarea[aria-label="สิ่งที่ควรรู้เกี่ยวกับคุณ"]') as HTMLTextAreaElement)
    expect(box.value).toBe('')
    expect(box.placeholder).toContain('บริบทผู้ใช้')
    expect(ReadIdentityFile).not.toHaveBeenCalled()
    expect(container.querySelector('.mset-error')).toBeNull()
  })

  // The user's file left คำสั่งประจำตัว: one editor per file.
  it('is the only editor of context.md — คำสั่งประจำตัว no longer lists it', async () => {
    const { container } = render(Settings, { onClose: () => {} })
    await openSection(container, 'คำสั่งประจำตัว')
    await waitFor(() => expect(screen.getByText('identity.md')).toBeTruthy())
    expect(screen.queryByText('context.md')).toBeNull()
  })
})
