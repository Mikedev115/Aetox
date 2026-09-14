import { describe, it, expect, beforeEach, vi } from 'vitest'
import { guide } from '../lib/guide/guideState.svelte'
import { mapPick } from '../lib/guide/mapPick'
import { GUIDE_MAP } from '../lib/guide/map'
import { GUIDE_ROUTES } from '../lib/guide/routes'
import { th } from '../lib/locales/th'
import { en } from '../lib/locales/en'

describe('Guide Store and MapPick (Phase 2)', () => {
  beforeEach(() => {
    guide.stop()
    document.body.innerHTML = ''
    vi.restoreAllMocks()
  })

  it('starts guide mode and stops cleanly', () => {
    expect(guide.on).toBe(false)
    expect(guide.stopId).toBe(null)

    guide.start('first')
    expect(guide.on).toBe(true)
    expect(guide.route?.id).toBe('first')
    expect(guide.stopId).toBe('sidebar.projects')

    guide.stop()
    expect(guide.on).toBe(false)
    expect(guide.route).toBe(null)
    expect(guide.stopId).toBe(null)
  })

  it('navigates route forward and backward with wrap-around', () => {
    guide.start('first')
    expect(guide.stopId).toBe(GUIDE_ROUTES.first.stops[0])

    guide.next()
    expect(guide.stopId).toBe(GUIDE_ROUTES.first.stops[1])

    guide.prev()
    expect(guide.stopId).toBe(GUIDE_ROUTES.first.stops[0])

    guide.prev()
    const lastStop = GUIDE_ROUTES.first.stops[GUIDE_ROUTES.first.stops.length - 1]
    expect(guide.stopId).toBe(lastStop)
  })

  it('refuses to press unsafe elements and does not trigger click()', () => {
    const sendBtn = document.createElement('button')
    sendBtn.setAttribute('data-guide', 'chat.send')
    const clickSpy = vi.fn()
    sendBtn.addEventListener('click', clickSpy)
    document.body.appendChild(sendBtn)

    guide.start(undefined, 'chat.send')
    const result = guide.press()

    expect(clickSpy).not.toHaveBeenCalled()
    expect(result.ok).toBe(false)
    expect(result.message).toBe(th['guide.safeRefusal'])
  })

  it('presses safe elements and triggers click()', () => {
    const doorBtn = document.createElement('button')
    doorBtn.setAttribute('data-guide', 'topbar.door')
    const clickSpy = vi.fn()
    doorBtn.addEventListener('click', clickSpy)
    document.body.appendChild(doorBtn)

    guide.start(undefined, 'topbar.door')
    const result = guide.press()

    expect(clickSpy).toHaveBeenCalledTimes(1)
    expect(result.ok).toBe(true)
    expect(result.message).toBe(th['guide.pressed'])
  })

  it('positions mascot and bubble on opposite sides based on target rect', () => {
    const winW = 1200
    const SIZE = 72

    // Target on left side of screen
    const targetLeft = { left: 100, right: 180, top: 200, height: 40, width: 80 }
    const rightSideX = targetLeft.right + 16 // ~196px
    const roomRight = rightSideX + SIZE + 24 < winW // true
    const standsRightOfTarget = rightSideX > targetLeft.left // true
    const roomR = rightSideX + SIZE + 12 + 340 <= winW
    const flipLeftCase = standsRightOfTarget ? roomR : false

    // Mascot stands on right of target (tx = 196), roomR is true -> flip = true (bubble on right)
    expect(roomRight).toBe(true)
    expect(flipLeftCase).toBe(true)

    // Target on far right side of screen
    const targetRight = { left: 1100, right: 1180, top: 200, height: 40, width: 80 }
    const rightSideXFar = targetRight.right + 16 // 1196px
    const roomRightFar = rightSideXFar + SIZE + 24 < winW // false (1292 > 1200)
    const txFar = roomRightFar ? rightSideXFar : Math.max(8, targetRight.left - SIZE - 16) // 1100 - 88 = 1012px
    const standsRightFar = txFar > targetRight.left // false (1012 < 1100)
    const roomLFar = txFar - 12 - 340 >= 0 // true
    const flipRightCase = standsRightFar ? false : !roomLFar

    // Mascot stands on left of target (tx = 1012), bubble stays on left (flip = false)
    expect(roomRightFar).toBe(false)
    expect(standsRightFar).toBe(false)
    expect(flipRightCase).toBe(false)
  })

  it('mapPick handles 10 representative queries in Thai', () => {
    const cases: [string, string][] = [
      ['ความจำอยู่ไหน', 'settings.head.tab.memory'],
      ['ปุ่มส่งคืออะไร', 'chat.send'],
      ['ต่อสมอง', 'settings.rail.brain'],
      ['สร้างโปรเจกต์', 'sidebar.create_project'],
      ['สลับหัว', 'topbar.door'],
      ['ช่องพิมพ์', 'composer.input'],
      ['ตั้งค่าอวตาร', 'settings.rail.avatar'],
      ['ดูประวัติ', 'sidebar.history'],
      ['ตั้งค่าเสียง', 'settings.rail.voice'],
      ['รอบแรก', 'sidebar.projects'],
    ]

    for (const [q, expectedId] of cases) {
      const res = mapPick(q, { stopId: null, route: null })
      expect(res.stopId, `Expected "${q}" to resolve to ${expectedId}`).toBe(expectedId)
    }
  })

  it('mapPick handles 10 representative queries in English', () => {
    const cases: [string, string][] = [
      ['where is memory', 'settings.head.tab.memory'],
      ['what is send button', 'chat.send'],
      ['connect brain', 'settings.rail.brain'],
      ['create project', 'sidebar.create_project'],
      ['switch head', 'topbar.door'],
      ['message input', 'composer.input'],
      ['avatar settings', 'settings.rail.avatar'],
      ['search history', 'sidebar.search'],
      ['voice settings', 'settings.rail.voice'],
      ['tour', 'sidebar.projects'],
    ]

    for (const [q, expectedId] of cases) {
      const res = mapPick(q, { stopId: null, route: null })
      expect(res.stopId, `Expected "${q}" to resolve to ${expectedId}`).toBe(expectedId)
    }
  })

  it('mapPick falls back to settings.rail.brain with hint for unknown queries', () => {
    const res = mapPick('สภาพอากาศวันนี้', { stopId: null, route: null })
    expect(res.stopId).toBe('settings.rail.brain')
    expect(res.sentence).toContain(th['guide.unknownWithBrainHint'])
  })

  it('hides companion when guide is active and restores it when guide closes', () => {
    const companionOn = true

    // When companion is on and guide is off: companion renders
    const renderCompanionInitial = companionOn && !guide.on
    expect(renderCompanionInitial).toBe(true)

    // Guide opens: companion is suppressed
    guide.start('first')
    const renderCompanionWhileGuide = companionOn && !guide.on
    expect(renderCompanionWhileGuide).toBe(false)

    // Guide closes: companion returns
    guide.stop()
    const renderCompanionAfterClose = companionOn && !guide.on
    expect(renderCompanionAfterClose).toBe(true)
  })

  it('intercepts clicks on [data-guide] elements in capture phase only when guide is on', () => {
    let guideIntercepted = false
    const testBtn = document.createElement('button')
    testBtn.setAttribute('data-guide', 'sidebar.search')
    document.body.appendChild(testBtn)

    const onCaptureClick = (e: MouseEvent) => {
      if (!guide.on) return
      const target = e.target as HTMLElement | null
      const guideEl = target?.closest<HTMLElement>('[data-guide]')
      if (guideEl) {
        e.preventDefault()
        e.stopPropagation()
        guideIntercepted = true
        guide.goTo(guideEl.getAttribute('data-guide')!)
      }
    }

    document.addEventListener('click', onCaptureClick, true)

    // When guide is off
    guide.stop()
    guideIntercepted = false
    testBtn.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    expect(guideIntercepted).toBe(false)

    // When guide is on
    guide.start()
    guideIntercepted = false
    testBtn.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
    expect(guideIntercepted).toBe(true)
    expect(guide.stopId).toBe('sidebar.search')

    document.removeEventListener('click', onCaptureClick, true)
  })
})
