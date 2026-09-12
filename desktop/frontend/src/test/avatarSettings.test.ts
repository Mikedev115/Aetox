import { describe, it, expect, beforeEach } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import AvatarSettings from '../lib/mascot/AvatarSettings.svelte'
import Companion from '../lib/mascot/Companion.svelte'
import { avatarPrefs, setAvatarPrefs, resetAvatarPrefs, assistantOptions, DEFAULT_PREFS } from '../lib/mascot/avatarPrefs.svelte'
import { SHELL } from '../lib/mascot/palette'
import { TOP } from '../lib/mascot/parts'
import { setLocale } from '../lib/i18n.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'

// ตั้งค่า › อวตาร: the four dials, drawn as whole outcomes, writing one store
// the companion reads. What is guarded is that a choice on the page IS what
// the companion wears, and that a stale or hostile stored value lands on a
// default rather than on a broken mascot.

beforeEach(() => {
  localStorage.clear()
  resetAvatarPrefs()
  setLocale('th')
  cockpit.awaitingReply = false
  cockpit.toolSteps = []
  cockpit.streamingText = ''
  cockpit.ask = null
})

describe('avatar preferences', () => {
  it('start at the sheet\'s robot: white, brand hue, orb, neutral', () => {
    expect(avatarPrefs).toMatchObject(DEFAULT_PREFS)
    expect(assistantOptions()).toEqual({ shell: 'white', top: 'orb', face: 'neutral' })
  })

  it('remember a choice and hand it to the companion', async () => {
    setAvatarPrefs({ shell: 'colour', hue: 150, top: 'chevrons' })
    expect(JSON.parse(localStorage.getItem('avatarPrefs')!)).toMatchObject({ shell: 'colour', hue: 150, top: 'chevrons' })
    const { container } = render(Companion)
    await waitFor(() => expect(container.querySelector('.companion .mascot')).toBeTruthy())
    const svg = container.querySelector('.companion .mascot')!.innerHTML
    expect(svg).toContain('hsl(150 ')
    expect(svg).not.toContain('hsl(218 ')
    // chevrons, not the orb
    expect(svg).toContain('M27 2l3.2 3-3.2 3')
  })

  // A preference that names a row the catalogue no longer has — or never had —
  // is the default, not an exception in the middle of drawing the assistant.
  it('land unknown or hostile values on the defaults', () => {
    setAvatarPrefs({ shell: 'chrome', top: '<script>', face: 'happy', hue: Number.NaN })
    expect(avatarPrefs).toMatchObject(DEFAULT_PREFS)
    setAvatarPrefs({ hue: -30 })
    expect(avatarPrefs.hue).toBe(330)
  })
})

describe('the avatar page', () => {
  it('offers every shell and top light the catalogue has, and marks the current one', async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelector('h2')?.textContent).toBe('อวตาร'))
    const parts = container.querySelectorAll('.ag-part')
    // shells + (brand + 12 hues) + tops + 2 identity faces
    expect(parts.length).toBe(SHELL.length + 13 + TOP.length + 2)
    expect(container.querySelectorAll('.ag-part.on').length).toBe(4)
    expect(container.querySelector('.avatar-reset')).toBeNull()
    // every cell is still; only the preview moves — twenty-five breathing
    // together was the page the owner called กระตุก
    for (const cell of parts) expect(cell.querySelector('.mascot')!.classList.contains('still'), 'still').toBe(true)
    expect(container.querySelector('.avatar-stage .mascot')!.classList.contains('still')).toBe(false)
    expect(container.querySelector('.avatar-stage .mascot')!.classList.contains('sway')).toBe(true)
  })

  it('writes a click straight into the preferences and shows the reset', async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelector('.ag-part')).toBeTruthy())
    const dark = container.querySelector('.ag-part[title="ดำ"]')!
    await fireEvent.click(dark)
    expect(avatarPrefs.shell).toBe('dark')
    await waitFor(() => expect(container.querySelector('.avatar-reset')).toBeTruthy())
    await fireEvent.click(container.querySelector('.avatar-reset button')!)
    expect(avatarPrefs.shell).toBe('white')
  })

  it('carries the on-screen switch', async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelector('.mswitch input')).toBeTruthy())
    const box = container.querySelector('.mswitch input') as HTMLInputElement
    expect(box.checked).toBe(true)
    await fireEvent.click(box)
    expect(localStorage.getItem('companionOn')).toBe('off')
  })
})
