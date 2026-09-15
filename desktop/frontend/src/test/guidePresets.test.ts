import { describe, it, expect, beforeEach, vi } from 'vitest'
import { GUIDE_PRESETS, presetLabelKey } from '../lib/guide/presets'
import { GUIDE_MAP } from '../lib/guide/map'
import { guidePrefs, setGuidePref, resetGuidePrefs } from '../lib/guide/guidePrefs.svelte'
import { th } from '../lib/locales/th'
import { en } from '../lib/locales/en'
import { zh } from '../lib/locales/zh'

// The guide's settings and its ready-made walks. Both exist to be extended —
// a preset is one row, a setting is one field — so what is tested is that
// adding one cannot half-land: a preset pointing nowhere, or a label that
// exists in one language only.

describe('the ready-made walks', () => {
  it('every preset goes somewhere the map actually knows', () => {
    for (const p of GUIDE_PRESETS) {
      expect(GUIDE_MAP.find((e) => e.id === p.to), `preset "${p.id}" points at "${p.to}"`).toBeTruthy()
    }
  })

  it('every preset is named in all three languages', () => {
    for (const p of GUIDE_PRESETS) {
      const key = presetLabelKey(p) as keyof typeof th
      for (const [lang, dict] of [['th', th], ['en', en], ['zh', zh]] as const) {
        expect((dict as Record<string, string>)[key]?.trim(), `${key} in ${lang}`).toBeTruthy()
      }
    }
  })

  it('carries no duplicate ids', () => {
    expect(new Set(GUIDE_PRESETS.map((p) => p.id)).size).toBe(GUIDE_PRESETS.length)
  })
})

describe('the guide\'s own settings', () => {
  beforeEach(() => {
    localStorage.clear()
    resetGuidePrefs()
  })

  it('starts on the chat\'s brain, and remembers a choice', () => {
    expect(guidePrefs.provider).toBe('')
    setGuidePref('provider', 'ollama')
    setGuidePref('model', 'qwen3:8b')
    expect(JSON.parse(localStorage.getItem('guidePrefs')!)).toMatchObject({ provider: 'ollama', model: 'qwen3:8b' })
  })

  it('refuses junk out of storage rather than putting it on a request', async () => {
    localStorage.setItem('guidePrefs', JSON.stringify({ provider: 'ollama', think: 'ludicrous', lang: 7 }))
    // A fresh module registry, so the file's own load() runs against that blob.
    vi.resetModules()
    const fresh = await import('../lib/guide/guidePrefs.svelte')
    expect(fresh.guidePrefs.provider).toBe('ollama')
    expect(fresh.guidePrefs.think).toBe('low') // not the stored nonsense
    expect(fresh.guidePrefs.lang).toBe('auto')
  })

  it('survives storage being unavailable', () => {
    const real = Storage.prototype.setItem
    Storage.prototype.setItem = () => {
      throw new Error('quota')
    }
    try {
      expect(() => setGuidePref('voice', true)).not.toThrow()
      expect(guidePrefs.voice).toBe(true) // held for this run
    } finally {
      Storage.prototype.setItem = real
    }
  })
})
