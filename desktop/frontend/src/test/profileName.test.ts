// The name in the sidebar footer, and where it goes.
//
// Three things the owner asked for on 12 ก.ย. 2026, each of which is a
// behaviour here rather than a look: the footer's gear goes to settings
// without opening the menu first; the menu opens with the name as text, not a
// focused field; and a name is saved as it is typed, so closing the menu by
// clicking elsewhere — which unmounts the field before blur — cannot lose it.
// The fourth is that the name is read by more than the footer: the empty chat
// greets by it (starters.headlineFor).
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import Sidebar from '../lib/Sidebar.svelte'
import { UserName, SetUserName, ProviderAccountFor, CurrentSessionID, SessionMode } from './mocks/wailsApp'
import { profile } from '../lib/stores/profile.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { setShell } from '../lib/shell.svelte'
import { startersFor, headlineFor } from '../lib/starters'
import { t } from '../lib/i18n.svelte'

const room = (desk: string, space = '') => startersFor({ desk, chair: '', space })

beforeEach(() => {
  vi.clearAllMocks()
  cockpit.desk = ''
  cockpit.activeView = 'chat'
  cockpit.history.length = 0
  setShell('assistant')
  vi.mocked(CurrentSessionID).mockResolvedValue('20260826-120000.000')
  vi.mocked(SessionMode).mockResolvedValue('')
  vi.mocked(ProviderAccountFor).mockResolvedValue(null as never)
  vi.mocked(UserName).mockResolvedValue('mike')
  profile.name = ''
  profile.loaded = false
})

describe('the footer name', () => {
  it('the gear opens settings and leaves the menu closed', async () => {
    const onOpenSettings = vi.fn()
    const { container } = render(Sidebar, { onOpenSettings })
    ;(container.querySelector('.side-gear') as HTMLElement).click()
    expect(onOpenSettings).toHaveBeenCalledTimes(1)
    expect(container.querySelector('.profile-menu')).toBeNull()
  })

  it('the menu opens with the name as text; the field appears only when the name is clicked', async () => {
    const { container } = render(Sidebar, { onOpenSettings: () => {} })
    ;(container.querySelector('.side-footer') as HTMLElement).click()
    await waitFor(() => expect(container.querySelector('.profile-head .name-text')?.textContent?.trim()).toBe('mike'))
    expect(container.querySelector('.name-input')).toBeNull()

    ;(container.querySelector('.name-text') as HTMLElement).click()
    await waitFor(() => expect(container.querySelector('.name-input')).toBeTruthy())
    expect(document.activeElement).toBe(container.querySelector('.name-input'))
  })

  it('is saved as it is typed, not only on Enter or blur', async () => {
    const { container } = render(Sidebar, { onOpenSettings: () => {} })
    ;(container.querySelector('.side-footer') as HTMLElement).click()
    await waitFor(() => expect(container.querySelector('.name-text')).toBeTruthy())
    ;(container.querySelector('.name-text') as HTMLElement).click()
    await waitFor(() => expect(container.querySelector('.name-input')).toBeTruthy())
    const input = container.querySelector('.name-input') as HTMLInputElement

    await fireEvent.input(input, { target: { value: 'mike w' } })
    expect(SetUserName).toHaveBeenLastCalledWith('mike w')
    expect(profile.name).toBe('mike w')
  })
})

describe('the greeting', () => {
  it('addresses the person by the footer name on the desks that greet one, and not on a project', () => {
    profile.name = 'mike'
    expect(headlineFor(room('assistant'), profile.name, t)).toBe(t('start.assistant.headlineNamed', { name: 'mike' }))
    expect(headlineFor(room('assistant'), profile.name, t)).toContain('mike')
    expect(headlineFor(room('coding'), profile.name, t)).toContain('mike')
    expect(headlineFor(room('assistant', 'Aetox'), profile.name, t)).toBe(t('start.project.headline'))
  })

  it('asks the nameless question when no name was typed', () => {
    expect(headlineFor(room('assistant'), '  ', t)).toBe(t('start.assistant.headline'))
    expect(headlineFor(room('assistant'), '', t)).not.toContain('{name}')
  })
})
