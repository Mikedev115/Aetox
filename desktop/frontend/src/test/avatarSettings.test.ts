import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, waitFor, fireEvent } from '@testing-library/svelte'
import AvatarSettings from '../lib/mascot/AvatarSettings.svelte'
import Companion from '../lib/mascot/Companion.svelte'
import { avatarPrefs, setAvatarPrefs, resetAvatarPrefs, assistantOptions, DEFAULT_PREFS, personas, addPersona, savePersona, usePersona, removePersona, clearPersonas, wornPersona } from '../lib/mascot/avatarPrefs.svelte'
import { SHELL, ACCENT } from '../lib/mascot/palette'
import { TOP, FACE } from '../lib/mascot/parts'
import { POSE } from '../lib/mascot/poses'
import { setLocale } from '../lib/i18n.svelte'
import { cockpit, SETTINGS_SECTION_KEY } from '../lib/stores/cockpit.svelte'
import { setCompanionVoice } from '../lib/mascot/companionSetting.svelte'
import { stopSpeech } from '../lib/speech.svelte'
import { TTSStatus, ListTTSVoices, StartSpeech } from './mocks/wailsApp'

// ตั้งค่า › อวตาร: the four dials, drawn as whole outcomes, writing one store
// the companion reads. What is guarded is that a choice on the page IS what
// the companion wears, and that a stale or hostile stored value lands on a
// default rather than on a broken mascot.

beforeEach(() => {
  localStorage.clear()
  resetAvatarPrefs()
  clearPersonas()
  setLocale('th')
  cockpit.awaitingReply = false
  cockpit.toolSteps = []
  cockpit.streamingText = ''
  cockpit.ask = null
  sessionStorage.clear()
  stopSpeech()
  setCompanionVoice(true)
  vi.mocked(TTSStatus).mockResolvedValue('')
  vi.mocked(ListTTSVoices).mockResolvedValue([{ id: 'p', name: 'Pattara', lang: 'th-TH', gender: 'Male', active: false }] as any)
  vi.mocked(StartSpeech).mockClear()
})

describe('avatar preferences', () => {
  it('start at the mark\'s own robot: white, ink, orb, neutral', () => {
    expect(avatarPrefs).toMatchObject(DEFAULT_PREFS)
    expect(assistantOptions()).toEqual({ shell: 'white', accent: 'ink', top: 'orb', face: 'neutral' })
  })

  it('remember a choice and hand it to the companion', async () => {
    setAvatarPrefs({ shell: 'colour', accent: 'mint', top: 'chevrons' })
    expect(JSON.parse(localStorage.getItem('avatarPrefs')!)).toMatchObject({ shell: 'colour', accent: 'mint', top: 'chevrons' })
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
    setAvatarPrefs({ shell: 'chrome', top: '<script>', face: 'happy', accent: 'plaid' })
    expect(avatarPrefs).toMatchObject(DEFAULT_PREFS)
  })

  // A store written before the accents had names held the hue in degrees;
  // it lands on the nearest named colour, and null (the old brand default)
  // on today's default.
  it('read a hue stored by the earlier build as the nearest accent', async () => {
    localStorage.setItem('avatarPrefs', JSON.stringify({ shell: 'colour', hue: 152, top: 'orb', face: 'neutral' }))
    // …and the six-slot build kept nulls where a slot was empty: they go.
    localStorage.setItem('avatarPersonas', JSON.stringify([null, { shell: 'white', hue: null, top: 'bar', face: 'focused' }, null]))
    vi.resetModules()
    const fresh = await import('../lib/mascot/avatarPrefs.svelte')
    expect(fresh.avatarPrefs).toMatchObject({ shell: 'colour', accent: 'mint' })
    expect(fresh.personas.slots.length).toBe(1)
    expect(fresh.personas.slots[0]).toMatchObject({ shell: 'white', accent: 'ink', top: 'bar' })
  })
})

// The page opens on its first sub-menu, the design — the stage, the poses
// and the personas; the main avatar's switches are behind the second.
async function openMain(container: HTMLElement): Promise<void> {
  await waitFor(() => expect(container.querySelectorAll('.set-subtab').length).toBe(2))
  await fireEvent.click(container.querySelectorAll('.set-subtab')[1])
}

describe('the avatar page', () => {
  it("splits into the design and the main avatar's switches, and opens on the design", async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelectorAll('.set-subtab').length).toBe(2))
    const tabs = container.querySelectorAll('.set-subtab')
    expect(tabs[0].textContent).toContain('ออกแบบอวตาร')
    expect(tabs[1].textContent).toContain('ตั้งค่าอวตารหลัก')
    expect(tabs[0].classList.contains('active')).toBe(true)
    expect(container.querySelector('.stage')).toBeTruthy()
    expect(container.querySelector('.mswitch input')).toBeNull()
    await fireEvent.click(tabs[1])
    await waitFor(() => expect(container.querySelector('.mswitch input')).toBeTruthy())
    expect(container.querySelector('.stage')).toBeNull()
    expect(tabs[1].classList.contains('active')).toBe(true)
  })

  it('offers every shell and top light the catalogue has, and marks the current one', async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelector('h2')?.textContent).toBe('อวตาร'))
    const parts = container.querySelectorAll('.cell')
    // shells + accents + tops + the identity faces — every row the catalogue has
    expect(parts.length).toBe(SHELL.length + ACCENT.length + TOP.length + FACE.filter((f) => f.identity).length)
    expect(FACE.filter((f) => f.identity).length).toBeGreaterThanOrEqual(5)
    expect(container.querySelectorAll('.cell.on').length).toBe(4)
    expect(container.querySelector('.avatar-reset')).toBeNull()
    // every cell is still; only the preview moves — twenty-five breathing
    // together was the page the owner called กระตุก
    for (const cell of parts) expect(cell.querySelector('.mascot')!.classList.contains('still'), 'still').toBe(true)
    expect(container.querySelector('.fig-box .mascot')!.classList.contains('still')).toBe(false)
    expect(container.querySelector('.fig-box .mascot')!.classList.contains('sway')).toBe(true)
  })

  it('writes a click straight into the preferences and shows the reset', async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelector('.cell')).toBeTruthy())
    const dark = container.querySelector('.cell[title="ดำ"]')!
    await fireEvent.click(dark)
    expect(avatarPrefs.shell).toBe('dark')
    await waitFor(() => expect(container.querySelector('.reset')).toBeTruthy())
    await fireEvent.click(container.querySelector('.reset')!)
    expect(avatarPrefs.shell).toBe('white')
  })

  // The stage: one panel per part, a lead overlay measured off the real boxes
  // (jsdom has no layout, so every box is 0×0 and the stacked-layout rule
  // draws none — the overlay itself and the banner are what can be checked).
  it('has one panel per part, the lead overlay, and says whose avatar it is', async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelectorAll('.panel').length).toBe(4))
    expect(container.querySelector('.stage > .leads')).toBeTruthy()
    expect(container.querySelector('.panel .n')).toBeNull()
    expect(container.querySelector('.avatar-main')?.textContent).toContain('อวตารหลักของ Aetox')
    // every pose the rig has, under the stage, each with a word in this language
    const chips = Array.from(container.querySelectorAll('.poses .chip'))
    expect(chips.length).toBe(Object.keys(POSE).length)
    for (const c of chips) expect(Object.keys(POSE), c.textContent!).not.toContain(c.textContent!.trim())
    await fireEvent.click(chips.find((c) => c.textContent!.trim() === 'เดิน')!)
    await waitFor(() => expect(container.querySelector('.fig-box .mascot')!.classList.contains('pose-walk')).toBe(true))
  })

  // Personas: no fixed count — + keeps what is worn as one more, remove
  // closes the gap, and the + card will not keep a twin of a look it has.
  it('keeps as many personas as + is pressed, and knows which one is worn', async () => {
    const { container } = render(AvatarSettings)
    await waitFor(() => expect(container.querySelector('.slot.add')).toBeTruthy())
    expect(container.querySelectorAll('.slot:not(.add)').length).toBe(0)
    const add = container.querySelector('.slot.add') as HTMLButtonElement
    await fireEvent.click(add)
    expect(personas.slots.length).toBe(1)
    expect(wornPersona()).toBe(0)
    await waitFor(() => expect(add.disabled).toBe(true)) // the default look is kept now
    setAvatarPrefs({ shell: 'colour', accent: 'mint' })
    await waitFor(() => expect(add.disabled).toBe(false))
    await fireEvent.click(add)
    expect(personas.slots.length).toBe(2)
    expect(personas.slots[1]).toMatchObject({ shell: 'colour', accent: 'mint' })
    expect(wornPersona()).toBe(1)
    await waitFor(() => expect(container.querySelectorAll('.slot')[1].classList.contains('worn')).toBe(true))
    expect(JSON.parse(localStorage.getItem('avatarPersonas')!)[1]).toMatchObject({ shell: 'colour', accent: 'mint' })
    resetAvatarPrefs()
    expect(wornPersona()).toBe(0)
    usePersona(1)
    expect(avatarPrefs.shell).toBe('colour')
    // remove the first: the second moves up, and the + card follows the list
    await fireEvent.click(container.querySelectorAll('.slot')[0].querySelectorAll('.acts button')[2])
    expect(personas.slots.length).toBe(1)
    expect(personas.slots[0]).toMatchObject({ shell: 'colour', accent: 'mint' })
    await waitFor(() => expect(container.querySelectorAll('.slot').length).toBe(2))
    expect(container.querySelectorAll('.slot')[1].classList.contains('add')).toBe(true)
    usePersona(5)
    expect(avatarPrefs.shell).toBe('colour') // a slot that is not there changes nothing
  })

  it('carries the on-screen switch', async () => {
    const { container } = render(AvatarSettings)
    await openMain(container)
    await waitFor(() => expect(container.querySelector('.mswitch input')).toBeTruthy())
    const box = container.querySelector('.mswitch input') as HTMLInputElement
    expect(box.checked).toBe(true)
    await fireEvent.click(box)
    expect(localStorage.getItem('companionOn')).toBe('off')
  })

  // The voice row: on by default, and the one place that says why the
  // companion is silent when it is — the engine's own reason, or no voice
  // for the UI's language — with the way to ตั้งค่า › เสียง beside it.
  it('carries the voice switch, on by default, with nothing to say when a voice speaks the language', async () => {
    const { container } = render(AvatarSettings)
    await openMain(container)
    await waitFor(() => expect(vi.mocked(ListTTSVoices)).toHaveBeenCalled())
    const box = container.querySelector('.voice-row .mswitch input') as HTMLInputElement
    expect(box.checked).toBe(true)
    await waitFor(() => expect(container.querySelector('.voice-note')).toBeNull())
    await fireEvent.click(container.querySelector('.voice-row .chip')!)
    await waitFor(() => expect(vi.mocked(StartSpeech)).toHaveBeenCalled())
    // hello has its own switch, one step under the voice, gone with it
    const greet = container.querySelector('.greet-row .mswitch input') as HTMLInputElement
    expect(greet.checked).toBe(true)
    await fireEvent.click(greet)
    expect(localStorage.getItem('companionGreet')).toBe('off')
    await fireEvent.click(box)
    expect(localStorage.getItem('companionVoice')).toBe('off')
    expect(container.querySelector('.voice-row .chip')).toBeNull()
    expect(container.querySelector('.greet-row')).toBeNull()
  })

  it("says the engine's own reason when it cannot run, and points at the voice page", async () => {
    vi.mocked(TTSStatus).mockResolvedValue('ไม่พบ PowerShell ในเครื่อง')
    const { container } = render(AvatarSettings)
    await openMain(container)
    await waitFor(() => expect(container.querySelector('.voice-note:not(.soft)')).toBeTruthy())
    expect(container.querySelector('.voice-note')?.textContent).toContain('ไม่พบ PowerShell ในเครื่อง')
    await fireEvent.click(container.querySelector('.voice-note .link')!)
    expect(sessionStorage.getItem(SETTINGS_SECTION_KEY)).toBe('voice')
    expect(cockpit.activeView).toBe('settings')
  })

  it('warns when no installed voice speaks the UI language', async () => {
    vi.mocked(ListTTSVoices).mockResolvedValue([{ id: 'z', name: 'Zira', lang: 'en-US', gender: 'Female', active: false }] as any)
    const { container } = render(AvatarSettings)
    await openMain(container)
    await waitFor(() => expect(container.querySelector('.voice-note:not(.soft)')).toBeTruthy())
    expect(container.querySelector('.voice-note')?.textContent).toContain('ยังไม่มีเสียงสำหรับภาษาที่ใช้อยู่')
  })
})
