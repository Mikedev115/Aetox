import { describe, it, expect, beforeEach, vi } from 'vitest'
import { guideCore } from '../lib/guide/core'
import { th } from '../lib/locales/th'

describe('GuideCore Deterministic Engine & Fact Pack', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('answers route commands in multiple languages deterministically', () => {
    // Thai
    const resTh = guideCore.find('ต่อสมอง')
    expect(resTh.route?.id).toBe('brain')
    expect(resTh.isExact).toBe(true)

    // English
    const resEn = guideCore.find('tour')
    expect(resEn.route?.id).toBe('first')
    expect(resEn.isExact).toBe(true)

    // Chinese
    const resZh = guideCore.find('连接大脑')
    expect(resZh.route?.id).toBe('brain')
    expect(resZh.isExact).toBe(true)
  })

  it('matches system concepts and points to related targets', () => {
    const res = guideCore.find('สแตนซ์คืออะไร')
    expect(res.conceptId).toBe('concept.stance')
    expect(res.stopId).toBe('composer.stance')
    expect(res.isExact).toBe(true)
  })

  it('describes entries with verified decisions ref and locales', () => {
    const desc = guideCore.describe('composer.stance')
    expect(desc).not.toBeNull()
    expect(desc?.id).toBe('composer.stance')
    expect(desc?.ref).toBe('§106')
    expect(desc?.safe).toBe(true)

    const conceptDesc = guideCore.describe('concept.desk_separation')
    expect(conceptDesc).not.toBeNull()
    expect(conceptDesc?.ref).toBe('§86')
  })

  it('builds compact targeted fact packs without token bloat', () => {
    const pack = guideCore.buildFactPack('ระดับการลงมือ', 'composer.input')
    expect(pack.retrievedFacts.length).toBeLessThanOrEqual(3)
    expect(pack.retrievedFacts.length).toBeGreaterThanOrEqual(1)
    expect(pack.visibleTargets.length).toBeLessThanOrEqual(10)

    const factsJson = JSON.stringify(pack)
    // Compact fact pack must be well under 2KB
    expect(factsJson.length).toBeLessThan(2000)
  })

  it('press refuses unsafe buttons and enforces safety gate', () => {
    const unsafeRes = guideCore.press('chat.send')
    expect(unsafeRes.ok).toBe(false)
    expect(unsafeRes.message).toBe(th['guide.safeRefusal'])

    // Safe button not on screen
    const safeMissingRes = guideCore.press('topbar.door')
    expect(safeMissingRes.ok).toBe(false)
    expect(safeMissingRes.message).toBe(th['guide.notOnScreen'])

    // Safe button on screen
    const btn = document.createElement('button')
    btn.setAttribute('data-guide', 'topbar.door')
    const clickHandler = vi.fn()
    btn.addEventListener('click', clickHandler)
    document.body.appendChild(btn)

    const safeRes = guideCore.press('topbar.door')
    expect(safeRes.ok).toBe(true)
  })
})
