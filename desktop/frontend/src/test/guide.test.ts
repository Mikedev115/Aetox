import { describe, it, expect, beforeEach, vi } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { render, fireEvent, waitFor } from '@testing-library/svelte'
import { tick } from 'svelte'
import { compactGuideReply, guide, modelTourChoice } from '../lib/guide/guideState.svelte'
import { mapPick } from '../lib/guide/mapPick'
import { GUIDE_ROUTES } from '../lib/guide/routes'
import { GUIDE_MAP } from '../lib/guide/map'
import { isShortcut, shortcutLabel } from '../lib/shortcuts'
import { restingSpot } from '../lib/guide/walk'
import { guidePrefs, setGuidePref, GUIDE_SIZE_DEFAULT, GUIDE_SIZE_MIN, GUIDE_SIZE_MAX } from '../lib/guide/guidePrefs.svelte'
import { arrivalFor, contextualGuideStarts, greetingFor, offeredWalkCategories, offeredWalks, pageDetail } from '../lib/guide/greeting'
import { watchInspector, watchPlace } from '../lib/guide/placeWatch'
import Guide from '../lib/guide/Guide.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { th } from '../lib/locales/th'
import { en } from '../lib/locales/en'
import { zh } from '../lib/locales/zh'
import { PAGE_IDS } from '../lib/rooms'
import { visibleGuideElement } from '../lib/guide/where'

// The guide on the desk (docs/architecture/ui-guide-2026-09-15.md §4): one
// figure that walks to a mapped element, says the map's words, presses only
// what the map marks safe, and — while it is open — turns a click on any
// mapped element into a question about it.

function mapped(id: string, rect?: Partial<DOMRect>): HTMLButtonElement {
  const el = document.createElement('button')
  el.setAttribute('data-guide', id)
  if (rect) {
    el.getBoundingClientRect = () => ({ left: 0, top: 0, right: 0, bottom: 0, width: 0, height: 0, x: 0, y: 0, toJSON() {}, ...rect }) as DOMRect
  }
  document.body.appendChild(el)
  return el
}

beforeEach(async () => {
  await guide.stop()
  document.body.innerHTML = ''
  localStorage.clear()
  cockpit.activeView = 'chat'
  cockpit.desk = 'assistant'
  cockpit.model.provider = ''
  vi.restoreAllMocks()
})

describe('the guide store', () => {
  it('chooses the visible copy when a responsive layout renders the same guide target twice', () => {
    const clipped = mapped('workbench.add_tab', {
      left: window.innerWidth - 1,
      right: window.innerWidth + 199,
      top: 100,
      bottom: 140,
      width: 200,
      height: 40,
    })
    const visible = mapped('workbench.add_tab', {
      left: 100,
      right: 300,
      top: 100,
      bottom: 140,
      width: 200,
      height: 40,
    })

    expect(visibleGuideElement('workbench.add_tab')).not.toBe(clipped)
    expect(visibleGuideElement('workbench.add_tab')).toBe(visible)
  })

  it('collapses an exact or truncated repeated model reply', () => {
    const answer = 'ได้เลยครับ เดี๋ยวผมพาเดินดูทีละจุด โดยคุณเป็นคนกดเองนะครับ'
    expect(compactGuideReply(`${answer} ${answer}`)).toBe(answer)
    expect(compactGuideReply(`${answer} ${answer.slice(0, -12)}`)).toBe(answer)
  })

  it('turns a model JSON tour choice into a prepared route instead of visible JSON', () => {
    expect(modelTourChoice('{"tour":"พาเปิดแมพโครงสร้างโปรเจกต์"}')).toEqual({
      recognized: true,
      routeId: 'code_map',
    })
    expect(modelTourChoice('```json\n{"route":"brain"}\n```')).toEqual({
      recognized: true,
      routeId: 'brain',
    })
    expect(modelTourChoice('คำตอบปกติ')).toEqual({ recognized: false, routeId: null })
  })

  it('opens on a route at its first stop and closes to nothing', async () => {
    mapped(GUIDE_ROUTES.first.stops[0], { width: 120, height: 32 })
    expect(guide.on).toBe(false)
    await guide.start('first')
    expect(guide.on).toBe(true)
    expect(guide.route?.id).toBe('first')
    expect(guide.stopId).toBe(GUIDE_ROUTES.first.stops[0])
    expect(guide.brain).toBe('map')
    await guide.stop()
    expect(guide.on).toBe(false)
    expect(guide.route).toBeNull()
    expect(guide.stopId).toBeNull()
    expect(guide.transcript).toEqual([])
  })

  it('turns an opened right panel into a step-by-step tool tour', async () => {
    const page = document.createElement('div')
    page.setAttribute('data-guide-place', 'chat')
    page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(page)
    // The global inspector toggle stays visible after the panel opens. The
    // tour must prefer the deeper `+` door instead of pointing back at the
    // toggle and asking the user to close the panel again.
    mapped('topbar.inspector_btn', { width: 36, height: 36 })
    mapped('workbench.add_tab', { width: 36, height: 36 })
    await guide.start()

    guide.inspectorChanged(true)

    await waitFor(() => expect(guide.route?.id).toBe('workbench_panel'))
    expect(guide.heading).toBe('topbar.tab.terminal')
    expect(guide.awaiting).toBe('workbench.add_tab')
    expect(guide.stopId).toBe('workbench.add_tab')
    expect(guide.sentence).toContain(th['guide.workbench.panel.opened'])
    expect(guide.sentence).toContain(th['guide.topbar.tab.terminal.what'])
  })

  it('steps a route both ways without wrapping past the beginning', async () => {
    const stops = GUIDE_ROUTES.first.stops
    // This test is about timeline indexing; navigation itself is exercised
    // against one real visible door at a time below.
    for (const stop of stops) mapped(stop, { width: 120, height: 32 })
    await guide.start('first')
    guide.next()
    await waitFor(() => expect(guide.stopId).toBe(stops[1]))
    guide.prev()
    await waitFor(() => expect(guide.stopId).toBe(stops[0]))
    guide.prev()
    await waitFor(() => expect(guide.stopId).toBe(stops[0]))
    // Nothing is kept: asking to be shown around starts at the beginning,
    // however far a previous walk got. Resuming silently is how somebody
    // presses "show me around" and lands on the MCP page (15 ก.ย. 2026).
    expect(localStorage.getItem('guideRoute')).toBeNull()
    await guide.stop()
    await guide.start('first')
    expect(guide.stopId).toBe(stops[0])
    expect(guide.route).toEqual({ id: 'first', at: 0 })
  })

  it('treats a real click on the current route control as completion and advances automatically', async () => {
    const stops = GUIDE_ROUTES.first.stops
    mapped(stops[0], { width: 120, height: 32 })
    mapped(stops[1], { width: 180, height: 36 })
    await guide.start('first')

    guide.advance(stops[0])

    await waitFor(() => expect(guide.stopId).toBe(stops[1]))
    expect(guide.route).toEqual({ id: 'first', at: 1 })
  })

  it('waits for a required field to contain a value, then acknowledges it and continues', async () => {
    const name = document.createElement('input')
    name.setAttribute('data-guide', 'team.name_input')
    name.getBoundingClientRect = () => ({ width: 320, height: 38 }) as DOMRect
    document.body.appendChild(name)
    mapped('team.lead_select', { width: 140, height: 34 })
    await guide.start()
    await guide.ask('พาไปตั้งทีมหน่อย')

    guide.completeInput('team.name_input')
    await new Promise((resolve) => setTimeout(resolve, 600))
    expect(guide.stopId).toBe('team.name_input')

    name.value = 'ทีมทดสอบ'
    guide.completeInput('team.name_input')

    await waitFor(() => expect(guide.stopId).toBe('team.lead_select'), { timeout: 1500 })
    expect(guide.sentence).toContain('เรียบร้อย')
    expect(guide.sentence).toContain(th['guide.team.lead_select.name'])
    expect(guide.transcript.at(-1)?.text).toContain('เรียบร้อย')
    expect(guide.transcript.at(-1)?.stopId).toBe('team.lead_select')
  })

  it('finishes a one-stop informational route as soon as its page is reached', async () => {
    const page = document.createElement('div')
    page.setAttribute('data-guide-place', 'artifacts')
    page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(page)
    mapped('artifacts.header', { width: 180, height: 48 })

    await guide.start('artifacts')

    expect(guide.route).toBeNull()
    expect(guide.stopId).toBe('artifacts.header')
    expect(guide.sentence).toContain(th['guide.routeAlreadyHere'].split('{route}')[0])
    expect(guide.sentence).toContain(th['guide.routeComplete'].split('?')[0])
  })

  it('appends each new stop to a tour that started as a typed question', async () => {
    mapped('capability.mcp.header', { width: 180, height: 36 })
    mapped('capability.skills.header', { width: 180, height: 36 })
    await guide.start()

    await guide.ask('พาไปดูห้องความสามารถ')
    expect(guide.transcript.at(-1)?.stopId).toBe('capability.mcp.header')

    guide.next()
    await waitFor(() => expect(guide.stopId).toBe('capability.skills.header'))
    await waitFor(() => expect(guide.transcript.at(-1)?.stopId).toBe('capability.skills.header'))
    expect(guide.transcript.at(-1)?.text).toBe(th['guide.capability.skills.header.what'])
  })

  it('starts a prepared route from the room already open instead of replaying old doors', async () => {
    const page = document.createElement('div')
    page.setAttribute('data-guide-place', 'settings.appearance')
    page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(page)
    mapped('settings.rail.brain', { width: 220, height: 40 })

    await guide.start('first')

    expect(guide.route).toEqual({ id: 'first', at: GUIDE_ROUTES.first.stops.indexOf('settings.rail.brain') })
    expect(guide.stopId).toBe('settings.rail.brain')
    // The rail is visible on Appearance, but the Brain page has not been
    // reached until the person presses it.
    expect(guide.awaiting).toBe('settings.rail.brain')
  })

  it('answers prepared chips from the map without turning them into model questions', async () => {
    mapped('settings.rail.brain', { width: 220, height: 40 })
    await guide.start(undefined, 'settings.rail.brain')
    const before = guide.transcript.length

    const answer = guide.answerPreset('common')

    expect(answer).toBe(th['guide.settings.rail.brain.common'])
    expect(guide.sentence).toBe(answer)
    expect(guide.transcript).toHaveLength(before)
  })

  it('keeps prepared chip answers visible inside a question-started guide conversation', async () => {
    const page = document.createElement('div')
    page.setAttribute('data-guide-place', 'settings.models')
    page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(page)
    mapped('settings.brain.hero', { width: 500, height: 120 })

    await guide.start()
    await guide.ask('ต่อสมองยังไง')
    const before = guide.transcript.length

    const answer = guide.answerPreset('recommend')

    expect(answer).toBe(th['guide.settings.brain.hero.recommend'])
    expect(guide.transcript).toHaveLength(before + 1)
    expect(guide.transcript.at(-1)).toMatchObject({
      who: 'guide',
      text: answer,
      stopId: 'settings.brain.hero',
    })
  })

  it('every walk bumps moveSeq once — a route step and a model point share one door', async () => {
    // Both stops on screen, so neither walk is a search for a missing target.
    mapped('topbar.door', { width: 120, height: 32 })
    mapped('chat.send', { width: 36, height: 36 })
    await guide.start(undefined, 'topbar.door')
    const before = guide.moveSeq
    await guide.goTo('chat.send')
    expect(guide.moveSeq).toBe(before + 1)
    expect(guide.stopId).toBe('chat.send')
    expect(guide.sentence).toBe('')
    await guide.goTo('chat.send', 'a sentence of its own')
    expect(guide.moveSeq).toBe(before + 2)
    expect(guide.sentence).toBe('a sentence of its own')
  })

  it('refuses to press what the map does not mark safe, and never clicks it', async () => {
    const send = mapped('chat.send', { width: 36, height: 36 })
    const clicked = vi.fn()
    send.addEventListener('click', clicked)
    await guide.start(undefined, 'chat.send')
    const res = guide.press()
    expect(clicked).not.toHaveBeenCalled()
    expect(res.ok).toBe(false)
    expect(res.message).toBe(th['guide.safeRefusal'])
    // By id too — the model's press names its target.
    expect(guide.press('chat.send').ok).toBe(false)
    await new Promise((r) => setTimeout(r, 0))
    expect(clicked).not.toHaveBeenCalled()
  })

  it('presses a safe element for the user, by the current stop or by id', async () => {
    const door = mapped('topbar.door', { width: 120, height: 32 })
    const clicked = vi.fn()
    door.addEventListener('click', clicked)
    await guide.start(undefined, 'topbar.door')
    expect(guide.press()).toEqual({ ok: true, message: th['guide.pressed'] })
    expect(guide.press('topbar.door').ok).toBe(true)
    // The click lands after the click that asked for it has finished.
    expect(clicked).not.toHaveBeenCalled()
    await new Promise((r) => setTimeout(r, 0))
    expect(clicked).toHaveBeenCalledTimes(2)
    expect(guide.press('sidebar.projects')).toEqual({ ok: false, message: th['guide.notOnScreen'] })
  })

  it('can try a visible starter once, then requires Next instead of clicking it twice', async () => {
    const starter = mapped('chat.starter', { width: 280, height: 90 })
    const clicked = vi.fn()
    starter.addEventListener('click', clicked)
    await guide.start(undefined, 'chat.starter')
    guide.transcript = [{ who: 'guide', text: guide.sentence, stopId: 'chat.starter' }]
    const { container } = render(Guide)

    const take = await waitFor(() => {
      const button = Array.from(container.querySelectorAll<HTMLButtonElement>('.say-command'))
        .find((item) => item.textContent?.includes(th['guide.tryThisPrompt']))
      expect(button).toBeTruthy()
      return button!
    })
    await fireEvent.click(take)
    await waitFor(() => expect(clicked).toHaveBeenCalledTimes(1))
    expect(guide.actedStopId).toBe('chat.starter')
    expect(container.textContent).not.toContain(th['guide.tryThisPrompt'])
  })

  it('uses the explicit guide button as confirmation before sending a prepared turn', async () => {
    cockpit.model.provider = 'ollama'
    const send = mapped('chat.send', { width: 36, height: 36 })
    const clicked = vi.fn()
    send.addEventListener('click', clicked)
    await guide.start(undefined, 'chat.send')
    guide.transcript = [{ who: 'guide', text: guide.sentence, stopId: 'chat.send' }]
    const { container } = render(Guide)

    const confirm = await waitFor(() => {
      const button = Array.from(container.querySelectorAll<HTMLButtonElement>('.say-command'))
        .find((item) => item.textContent?.includes(th['guide.sendThis']))
      expect(button).toBeTruthy()
      return button!
    })
    expect(clicked).not.toHaveBeenCalled()
    await fireEvent.click(confirm)
    await waitFor(() => expect(clicked).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(guide.sentence).toBe(th['guide.chat.waitingForAnswer']))
    expect(guide.stopId).toBeNull()
  })

  it('continues from a clicked starter to Send, then acknowledges the sent turn', async () => {
    mapped('chat.starter', { width: 280, height: 90 })
    mapped('chat.send', { width: 36, height: 36 })
    await guide.start(undefined, 'chat.starter')

    guide.advance('chat.starter')
    await waitFor(() => expect(guide.stopId).toBe('chat.send'))
    expect(guide.sentence).toBe(th['guide.chat.starter.readyToSend'])

    guide.advance('chat.send')
    await waitFor(() => expect(guide.sentence).toBe(th['guide.chat.waitingForAnswer']))
    expect(guide.stopId).toBeNull()
  })

  it('finishes the prepared Assistant-page tour at Send with the waiting explanation', async () => {
    const stops = GUIDE_ROUTES.assistant_desk.stops
    for (const stop of stops) mapped(stop, { width: 180, height: 40 })
    await guide.start('assistant_desk')

    for (let index = 0; index < stops.length - 1; index++) {
      guide.advance(stops[index])
      await waitFor(() => expect(guide.stopId).toBe(stops[index + 1]))
    }
    guide.advance(stops.at(-1)!)

    await waitFor(() => expect(guide.route).toBeNull())
    expect(guide.stopId).toBeNull()
    expect(guide.sentence).toBe(th['guide.chat.waitingForAnswer'])
  })

  it('diverts a send rehearsal to brain setup when no model is configured', async () => {
    mapped('sidebar.footer', { width: 180, height: 40 })
    const send = mapped('chat.send', { width: 36, height: 36 })
    const clicked = vi.fn()
    send.addEventListener('click', clicked)
    await guide.start(undefined, 'chat.send')
    guide.takeMe()

    await waitFor(() => expect(guide.route?.id).toBe('brain'))
    expect(clicked).not.toHaveBeenCalled()
    expect(guide.sentence).toBe(th['guide.brainMissing'])
  })
})

describe('the map brain', () => {
  it('answers ten Thai questions with the right stop', () => {
    const cases: [string, string][] = [
      ['ความจำอยู่ไหน', 'settings.head.tab.memory'],
      ['ปุ่มส่งคืออะไร', 'chat.send'],
      ['ต่อสมอง', 'settings.brain.hero'],
      ['สร้างโปรเจกต์', 'sidebar.create_project'],
      ['สลับหัว', 'topbar.door'],
      ['ช่องพิมพ์', 'composer.input'],
      ['ตั้งค่าอวตาร', 'settings.rail.avatar'],
      ['ดูประวัติ', 'sidebar.history'],
      ['ตั้งค่าเสียง', 'settings.rail.voice'],
      ['รอบแรก', GUIDE_ROUTES.first.stops[0]],
    ]
    for (const [q, id] of cases) {
      expect(mapPick(q, { stopId: null, route: null }).stopId, q).toBe(id)
    }
  })

  it('answers ten English questions with the right stop', () => {
    const cases: [string, string][] = [
      ['where is memory', 'settings.head.tab.memory'],
      ['what is send button', 'chat.send'],
      ['connect brain', 'settings.brain.hero'],
      ['create project', 'sidebar.create_project'],
      ['switch head', 'topbar.door'],
      ['message input', 'composer.input'],
      ['avatar settings', 'settings.rail.avatar'],
      ['search history', 'sidebar.search'],
      ['voice settings', 'settings.rail.voice'],
      ['tour', GUIDE_ROUTES.first.stops[0]],
    ]
    for (const [q, id] of cases) {
      expect(mapPick(q, { stopId: null, route: null }).stopId, q).toBe(id)
    }
  })

  it('walks a route on "next" and explains the current stop on "what is this"', () => {
    const next = mapPick('ต่อไป', { stopId: GUIDE_ROUTES.first.stops[0], route: { id: 'first', at: 0 } })
    expect(next.stopId).toBe(GUIDE_ROUTES.first.stops[1])
    expect(next.route).toEqual({ id: 'first', at: 1 })
    const here = mapPick('นี่คืออะไร', { stopId: 'chat.send', route: null })
    expect(here.stopId).toBe('chat.send')
    expect(here.sentence).toContain(th['guide.chat.send.name'])
  })

  it('points at the model page when it does not understand — the one place a brain is connected', () => {
    const res = mapPick('สภาพอากาศวันนี้', { stopId: null, route: null })
    expect(res.stopId).toBe('settings.rail.brain')
    expect(res.sentence).toBe(th['guide.unknownWithBrainHint'])
  })

  it('knows the code map as a system concept rather than asking the model to invent a route', () => {
    const res = mapPick('เทคโนโลยีแมพโครงสร้างโปรเจกต์ทำงานยังไง', { stopId: null, route: null })
    expect(res.stopId).toBe('workbench.repo_map_view')
    expect(res.sentence).toContain(th['guide.concept.code_map.what'])
  })
})

describe('desk-aware greeting', () => {
  it('explains the project map on Code and agent/team controls on Assistant', () => {
    cockpit.desk = 'coding'
    expect(pageDetail('chat').what).toBe(th['guide.room.chat.coding.what'])
    expect(greetingFor('chat', 4)).toContain('ไม่กี่ call')

    cockpit.desk = 'assistant'
    expect(pageDetail('chat').what).toBe(th['guide.room.chat.assistant.what'])
    expect(greetingFor('chat', 4)).toContain('เลือกทีม')
  })

  it('describes the active right-hand pane instead of repeating its parent desk', () => {
    cockpit.desk = 'coding'
    const git = greetingFor('chat', 24, 'workbench.git')
    expect(git).toContain(th['guide.topbar.tab.diff.name'])
    expect(git).toContain(th['guide.workbench.git.ask'])
    expect(git).not.toContain(th['guide.room.chat.coding.name'])
    expect(git).not.toContain('24')

    cockpit.desk = 'assistant'
    const terminal = greetingFor('chat', 24, 'workbench.terminal')
    expect(terminal).toContain(th['guide.topbar.tab.terminal.name'])
    expect(terminal).toContain(th['guide.workbench.terminal.ask'])
    expect(terminal).not.toContain(th['guide.room.chat.assistant.name'])
  })

  it('does not offer the tab that is already in front', () => {
    const page = document.createElement('div')
    page.setAttribute('data-guide-place', 'chat')
    page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    const pane = document.createElement('div')
    pane.setAttribute('data-guide-context', 'workbench.git')
    pane.getBoundingClientRect = () => ({ width: 500, height: 700 }) as DOMRect
    page.appendChild(pane)
    document.body.appendChild(page)

    const ids = contextualGuideStarts('chat', 'coding').map((choice) => choice.id)
    expect(ids).not.toContain('topbar.tab.diff')
    expect(ids).toContain('topbar.tab.terminal')
    expect(ids).toContain('topbar.tab.git_log')
  })
})

describe('the guide on screen', () => {
  it('stands beside the target with the bubble on the far side, and says the map\'s words', async () => {
    vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(1200)
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(800)
    mapped('sidebar.projects', { left: 20, top: 300, right: 180, bottom: 340, width: 160, height: 40 })
    await guide.start(undefined, 'sidebar.projects')
    const { container } = render(Guide)
    await waitFor(() => expect(container.querySelector('.say-name')?.textContent).toBe(th['guide.sidebar.projects.name']))
    const wrap = container.querySelector<HTMLElement>('.guide-mascot-wrap')!
    // The target is at the left edge: the figure stands to its right, and the
    // bubble opens to the figure's right — away from the target.
    await waitFor(() => expect(wrap.style.transform).toContain('translate(196px'))
    expect(wrap.classList.contains('flip')).toBe(true)
    await waitFor(() => expect(container.querySelector('.say-body')?.textContent).toContain(th['guide.sidebar.projects.what'].slice(0, 8)), { timeout: 4000 })
    expect(container.querySelector('.say-name')?.textContent).toBe(th['guide.sidebar.projects.name'])
    expect(container.querySelector('.say-tag')?.textContent).toBe(th['guide.canPress'])
    expect(container.querySelector('.say-map-badge')?.textContent).toBe(th['guide.fromMap'])
  })

  it('leaves a plain click alone, and answers a Shift+click', async () => {
    const search = mapped('sidebar.search', { width: 100, height: 20 })
    mapped('sidebar.projects', { width: 120, height: 32 })
    const clicked = vi.fn()
    search.addEventListener('click', clicked)
    await guide.start(undefined, 'sidebar.projects')
    const { unmount } = render(Guide)
    await tick()

    // A plain press is the person using their own app. The button works and
    // the guide says nothing about it.
    await fireEvent.click(search)
    expect(clicked).toHaveBeenCalledTimes(1)
    expect(guide.stopId).toBe('sidebar.projects')

    // Shift+click is the ask: the guide answers, and the button does NOT fire
    // — finding out what Send does should not send anything.
    await fireEvent.click(search, { shiftKey: true })
    expect(clicked).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(guide.stopId).toBe('sidebar.search'))

    await fireEvent.keyDown(window, { key: 'Escape' })
    await waitFor(() => expect(guide.on).toBe(false))
    unmount()
    await fireEvent.click(search, { shiftKey: true })
    expect(clicked).toHaveBeenCalledTimes(2) // closed: even Shift is just a click
  })

  it('explains runtime skill and MCP rows from their own facts', async () => {
    const skill = mapped('capability.item.skill.installed.invoice', { width: 240, height: 90 })
    skill.dataset.guidePage = 'capability.skills'
    skill.dataset.guideName = 'invoice'
    skill.dataset.guideWhat = 'Invoice is a skill that checks totals and tax.'
    skill.dataset.guideWhy = 'It keeps the same checks across jobs.'
    skill.dataset.guideCommon = 'Use it for invoice review.'
    skill.dataset.guideRecommend = 'Keep it for recurring finance work.'
    const clicked = vi.fn()
    skill.addEventListener('click', clicked)

    await guide.start()
    const { container } = render(Guide)
    await tick()
    await fireEvent.click(skill, { shiftKey: true })

    expect(clicked).not.toHaveBeenCalled()
    await waitFor(() => expect(guide.stopId).toBe('capability.item.skill.installed.invoice'))
    await waitFor(() => expect(container.querySelector('.say-name')?.textContent).toBe('invoice'))
    await waitFor(() => expect(container.querySelector('.say-body')?.textContent).toContain('checks totals and tax'), { timeout: 3000 })
  })

  it('says a model\'s answer where it stands, without walking', async () => {
    mapped('topbar.door', { left: 500, top: 20, right: 600, bottom: 50, width: 100, height: 30 })
    await guide.start(undefined, 'topbar.door')
    const { container } = render(Guide)
    await waitFor(() => expect(container.querySelector('.say-name')?.textContent).toBe(th['guide.topbar.door.name']))
    const moves = guide.moveSeq
    guide.say('this is the door')
    await waitFor(() => expect(container.querySelector('.say-body')?.textContent).toContain('this is the door'), { timeout: 3000 })
    expect(guide.moveSeq).toBe(moves)
  })

  it('keeps the take-me click inside the guide so an opened menu stays open', async () => {
    const footer = mapped('sidebar.footer', { width: 200, height: 40 })
    mapped('chat.send', { width: 36, height: 36 })
    footer.addEventListener('click', () => {
      mapped('account.settings', { width: 180, height: 32 })
    })

    const originalMatchMedia = globalThis.matchMedia
    globalThis.matchMedia = (() => ({ matches: true })) as unknown as typeof matchMedia
    await guide.start('brain')
    const { container, unmount } = render(Guide)
    const escaped = vi.fn()
    const outsideListener = (event: MouseEvent) => {
      if ((event.target as HTMLElement | null)?.closest('.say-take')) escaped()
    }
    window.addEventListener('click', outsideListener)

    try {
      const take = await waitFor(() => {
        const button = container.querySelector<HTMLButtonElement>('.say-take')
        expect(button).not.toBeNull()
        return button!
      })
      await fireEvent.click(take)

      expect(escaped).not.toHaveBeenCalled()
      await waitFor(() => expect(guide.awaiting).toBe('account.settings'))
    } finally {
      window.removeEventListener('click', outsideListener)
      unmount()
      globalThis.matchMedia = originalMatchMedia
    }
  })
})

describe('arriving', () => {
  it('is already at its resting spot on the very first frame', async () => {
    vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(1240)
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(860)
    await guide.start()
    const { container } = render(Guide)
    // Not (0,0): the figure used to render in the corner and then ride the
    // 800ms walk transition across the window, which reads as sliding in from
    // nowhere every time it opens.
    const wrap = container.querySelector<HTMLElement>('.guide-mascot-wrap')!
    const home = restingSpot({ width: 1240, height: 860 })
    expect(wrap.style.transform.replace(/\s+/g, '')).toBe(`translate(${home.x}px,${home.y}px)`)
  })

  it('fades in without animating the transform that says where it stands', () => {
    // An entrance keyframe touching `transform` would override the inline one
    // and snap the figure to the corner — the bug it was meant to replace.
    const css = fs.readFileSync(path.resolve(__dirname, '../lib/guide/Guide.svelte'), 'utf-8')
    const frames = css.slice(css.indexOf('@keyframes guide-arrive'))
    const block = frames.slice(0, frames.indexOf('}', frames.indexOf('{')) + 1)
    expect(block).toContain('opacity')
    expect(block).not.toContain('transform')
  })

  it('keeps a narrow dock at the normal chat width and centres it in viewport coordinates', () => {
    const css = fs.readFileSync(path.resolve(__dirname, '../lib/guide/Guide.svelte'), 'utf-8')
    const dock = css.slice(css.indexOf('.docked .say {'), css.indexOf('.docked .say::after'))
    expect(dock).toContain('left: calc(50vw - var(--guide-x))')
    expect(dock).toContain('width: min(var(--guide-bubble-width, 420px), calc(100vw - 24px))')
    expect(dock).toContain('translateX(-50%)')
    expect(dock).toContain('var(--guide-y)')
  })
})

describe('the doors into the guide', () => {
  it('F1 opens it from anywhere and closes it again, and no other chord answers to it', () => {
    const f1 = (over: Partial<KeyboardEventInit> = {}) =>
      new KeyboardEvent('keydown', { key: 'F1', code: 'F1', ...over })
    expect(isShortcut(f1(), 'guide')).toBe(true)
    // A function key carries no layout risk — the reason shortcuts.ts warns
    // about letters — but a modifier still has to disqualify it.
    expect(isShortcut(f1({ ctrlKey: true }), 'guide')).toBe(false)
    expect(isShortcut(f1({ shiftKey: true }), 'guide')).toBe(false)
    expect(isShortcut(new KeyboardEvent('keydown', { key: 'F2', code: 'F2' }), 'guide')).toBe(false)
    expect(shortcutLabel('guide')).toBe('F1')
  })

  it('every door the tour has, the guide has beside it', () => {
    // The map is what the guide knows, so its own doors are in it: a door the
    // map does not carry is one the guide cannot explain when asked about it.
    for (const id of ['account.guide', 'settings.about.guide_btn']) {
      expect(GUIDE_MAP.find((e) => e.id === id), id).toBeTruthy()
      expect(th[`guide.${id}.name` as keyof typeof th], id).toBeTruthy()
    }
  })
})

describe('what it says when it arrives', () => {
  it('has a page-specific common-use explanation for every signed page', () => {
    const locales = { th, en, zh } as const
    for (const page of PAGE_IDS) {
      const key = `guide.page.${page}.use`
      const askKey = `guide.page.${page}.ask`
      for (const [locale, messages] of Object.entries(locales)) {
        expect(messages[key as keyof typeof messages], `${locale}: ${key}`).toBeTruthy()
        expect(messages[askKey as keyof typeof messages], `${locale}: ${askKey}`).toBeTruthy()
      }
      const detail = pageDetail(page)
      expect(detail.name, `${page} name`).toBeTruthy()
      expect(detail.use, `${page} common use`).toBeTruthy()
      expect(detail.ask, `${page} question`).toBeTruthy()
      expect(arrivalFor(page), `${page} arrival`).toContain(detail.use)
      expect(arrivalFor(page), `${page} arrival question`).toContain(detail.ask)
    }
  })

  it('names the room it finds you in and asks what is unclear', () => {
    // jsdom gives every element a zero rect unless one is stubbed, and
    // explainableOnScreen counts only what has size — so the stub IS the test.
    mapped('chat.send', { width: 36, height: 36 })
    mapped('topbar.door', { width: 120, height: 32 })
    const inChat = greetingFor('chat')
    expect(inChat).toContain(th['guide.room.chat.assistant.name'])
    expect(inChat).toContain('2') // the two mapped things on screen
    expect(inChat).toMatch(/[?？]|ไหม|หรือเปล่า/)

    document.body.innerHTML = ''
    const bare = greetingFor('settings.models')
    expect(bare).toContain(th['guide.room.settings'])
    expect(bare).not.toContain('0') // never promises what it cannot point at
  })

  it('still asks when it does not know the room', () => {
    const unknown = greetingFor(null)
    expect(unknown).toBe(th['guide.greetPlain'])
    expect(unknown).toMatch(/[?？]|ไหม/)
  })

  it('opens with a short welcome, then explains the room named by its SIGN on request', async () => {
    // The page says where it is; the guide reads that, not the app's store.
    const page = document.createElement('div')
    page.setAttribute('data-guide-place', 'settings.models')
    page.getBoundingClientRect = () => ({ width: 800, height: 600 }) as DOMRect
    document.body.appendChild(page)
    await guide.start()
    expect(guide.route).toBeNull()
    expect(guide.stopId).toBeNull()
    expect(guide.welcoming).toBe(true)
    expect(guide.sentence).toBe(th['guide.welcome'])
    expect(guide.sentence).toContain('Shift')

    guide.describeCurrentPage()

    expect(guide.welcoming).toBe(false)
    expect(guide.sentence).toContain(th['guide.room.settings'])
  })
})

describe('being led there, one press at a time', () => {
  // The screen is what path.ts reads, so the test builds screens: the sidebar
  // as it looks on the chat page, then what a press reveals.
  const onChat = () => {
    document.body.innerHTML = ''
    mapped('sidebar.footer', { width: 200, height: 40 })
    mapped('chat.send', { width: 36, height: 36 })
  }

  it('points at the door instead of teleporting to a page you cannot see', async () => {
    onChat()
    await guide.start(undefined, 'settings.rail.brain')
    // Not standing at the target — standing at the button that leads there.
    expect(guide.stopId).toBe('sidebar.footer')
    expect(guide.awaiting).toBe('sidebar.footer')
    expect(guide.heading).toBe('settings.rail.brain')
    expect(guide.stepsLeft).toBeGreaterThan(1)
  })

  it('recognises a folded sidebar and leads through its visible toggle first', async () => {
    document.body.innerHTML = ''
    const toggle = mapped('topbar.sidebar_btn', { width: 36, height: 36 })
    toggle.setAttribute('aria-expanded', 'false')
    // Just like the production compact rail, its footer still has a real
    // rectangle. The explicit expanded state must outrank that rectangle.
    mapped('sidebar.footer', { width: 57, height: 40 })

    await guide.start(undefined, 'settings.rail.brain')
    expect(guide.stopId).toBe('topbar.sidebar_btn')
    expect(guide.awaiting).toBe('topbar.sidebar_btn')
    expect(guide.heading).toBe('settings.rail.brain')

    // The real toggle press reveals the sidebar; the guide re-reads the DOM
    // and moves to the account menu instead of asking to fold it again.
    guide.advance('topbar.sidebar_btn')
    setTimeout(() => toggle.setAttribute('aria-expanded', 'true'), 20)

    await waitFor(() => expect(guide.awaiting).toBe('sidebar.footer'))
    expect(guide.stopId).toBe('sidebar.footer')
    expect(guide.heading).toBe('settings.rail.brain')
  })

  it('moves on when the awaited button is pressed, and re-reads the screen each time', async () => {
    onChat()
    await guide.start(undefined, 'settings.rail.brain')
    expect(guide.awaiting).toBe('sidebar.footer')

    // Pressing it opens the menu — which is what makes the gear reachable.
    mapped('account.settings', { width: 180, height: 32 })
    guide.advance('sidebar.footer')
    await waitFor(() => expect(guide.awaiting).toBe('account.settings'), { timeout: 2000 })
    expect(guide.heading).toBe('settings.rail.brain')

    // Pressing THAT exposes the Settings rail. A visible rail is still a door
    // until its destination page has a sign, so the guide now asks for that
    // final real press instead of declaring arrival between frames.
    mapped('settings.rail.brain', { width: 220, height: 40 })
    guide.advance('account.settings')
    await waitFor(() => expect(guide.awaiting).toBe('settings.rail.brain'), { timeout: 2000 })
    expect(guide.stopId).toBe('settings.rail.brain')
    expect(guide.heading).toBe('settings.rail.brain')
  })

  it('verifies each real screen change and explains the page before continuing', async () => {
    const sign = (page: string) => {
      const el = document.createElement('div')
      el.setAttribute('data-guide-place', page)
      el.getBoundingClientRect = () => ({ width: 800, height: 600 }) as DOMRect
      document.body.appendChild(el)
    }

    onChat()
    sign('chat')
    await guide.start('brain')
    expect(guide.awaiting).toBe('sidebar.footer')

    // The actual button handler opens the account menu after Guide's capture
    // listener has started waiting.
    guide.advance('sidebar.footer')
    setTimeout(() => mapped('account.settings', { width: 180, height: 32 }), 20)
    await waitFor(() => expect(guide.awaiting).toBe('account.settings'))

    // Settings opens on General. The guide narrates that arrival and points
    // to Brain instead of assuming the final page is already open.
    guide.advance('account.settings')
    setTimeout(() => {
      document.body.innerHTML = ''
      sign('settings.general')
      mapped('settings.rail.brain', { width: 220, height: 40 })
    }, 20)
    await waitFor(() => expect(guide.awaiting).toBe('settings.rail.brain'))
    expect(guide.sentence).toBe(arrivalFor('settings.general'))

    // Only the rail's real click reaches the model page. Arrival explains the
    // page and the first useful landmark together.
    guide.advance('settings.rail.brain')
    setTimeout(() => {
      document.body.innerHTML = ''
      sign('settings.models')
      mapped('settings.brain.hero', { width: 500, height: 80 })
    }, 20)
    await waitFor(() => expect(guide.stopId).toBe('settings.brain.hero'))
    expect(guide.awaiting).toBeNull()
    expect(guide.sentence).toContain(arrivalFor('settings.models'))
    expect(guide.sentence).toContain(th['guide.settings.brain.hero.what'])
  })

  it('accepts an already-open destination instead of asking to click its active rail again', async () => {
    const sign = document.createElement('div')
    sign.setAttribute('data-guide-place', 'capability.mine')
    sign.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(sign)
    mapped('capability.rail.mcp', { width: 220, height: 40 })

    // Recreate the short transition state from the real bug: navigation was
    // planned while the heading had no box, then the page itself arrived.
    guide.on = true
    guide.heading = 'capability.mcp.header'
    guide.awaiting = 'capability.rail.mcp'
    guide.stopId = 'capability.rail.mcp'
    guide.sentence = ''

    guide.advance('capability.rail.mcp')

    expect(guide.awaiting).toBeNull()
    expect(guide.heading).toBeNull()
    expect(guide.stopId).toBe('capability.mcp.header')
    expect(guide.sentence).toContain(th['guide.capability.mcp.header.what'])
    expect(guide.sentence).not.toBe(th['guide.stepNoChange'])
  })

  it('take me there advances exactly one safe door per request', async () => {
    const clicked: string[] = []
    const sign = (page: string) => {
      const el = document.createElement('div')
      el.setAttribute('data-guide-place', page)
      el.getBoundingClientRect = () => ({ width: 800, height: 600 }) as DOMRect
      document.body.appendChild(el)
    }

    onChat()
    sign('chat')
    const footer = document.querySelector<HTMLElement>('[data-guide="sidebar.footer"]')!
    footer.addEventListener('click', () => {
      clicked.push('sidebar.footer')
      const settings = mapped('account.settings', { width: 180, height: 32 })
      settings.addEventListener('click', () => {
        clicked.push('account.settings')
        document.body.innerHTML = ''
        sign('settings.general')
        const rail = mapped('settings.rail.brain', { width: 220, height: 40 })
        rail.addEventListener('click', () => {
          clicked.push('settings.rail.brain')
          document.body.innerHTML = ''
          sign('settings.models')
          mapped('settings.brain.hero', { width: 500, height: 80 })
        })
      })
    })

    await guide.start('brain')
    guide.takeMe()

    await waitFor(() => expect(guide.awaiting).toBe('account.settings'))
    expect(clicked).toEqual(['sidebar.footer'])
    expect(guide.escorting).toBe(false)

    guide.takeMe()
    await waitFor(() => expect(guide.awaiting).toBe('settings.rail.brain'))
    expect(clicked).toEqual(['sidebar.footer', 'account.settings'])
    expect(guide.escorting).toBe(false)

    guide.takeMe()
    await waitFor(() => expect(guide.stopId).toBe('settings.brain.hero'), { timeout: 5000 })
    expect(clicked).toEqual(['sidebar.footer', 'account.settings', 'settings.rail.brain'])
    expect(guide.escorting).toBe(false)
    expect(guide.awaiting).toBeNull()
  })

  it('ignores presses it did not ask for, and keeps heading where it was', async () => {
    onChat()
    await guide.start(undefined, 'settings.rail.brain')
    guide.advance('chat.send') // the person did something else entirely
    await new Promise((r) => setTimeout(r, 30))
    expect(guide.awaiting).toBe('sidebar.footer')
    expect(guide.heading).toBe('settings.rail.brain')
  })

  it('gives up the walk when asked about something else', async () => {
    onChat()
    await guide.start(undefined, 'settings.rail.brain')
    expect(guide.heading).toBe('settings.rail.brain')
    guide.explain('chat.send') // Shift+click elsewhere
    await waitFor(() => expect(guide.stopId).toBe('chat.send'))
    expect(guide.heading).toBeNull()
    expect(guide.awaiting).toBeNull()
  })

  it('walks straight to anything already on screen', async () => {
    onChat()
    await guide.start(undefined, 'chat.send')
    expect(guide.stopId).toBe('chat.send')
    expect(guide.awaiting).toBeNull()
  })

  it('offers named walks to press, taken from the routes themselves', () => {
    const walks = offeredWalks()
    expect(walks.map((w) => w.id)).toEqual(Object.keys(GUIDE_ROUTES))
    for (const w of walks) expect(w.label.trim().length).toBeGreaterThan(0)
    expect(walks.find((w) => w.id === 'first')?.label).toBe(th['guide.walk.first'])
  })

  it('groups every prepared walk under one clear top-level topic', () => {
    const categories = offeredWalkCategories()
    expect(categories.map((category) => category.id)).toEqual(['setup', 'use', 'understand'])
    const grouped = categories.flatMap((category) => offeredWalks(category.id).map((walk) => walk.id))
    expect(grouped).toHaveLength(Object.keys(GUIDE_ROUTES).length)
    expect(new Set(grouped)).toEqual(new Set(Object.keys(GUIDE_ROUTES)))
    for (const category of categories) {
      expect(category.label.trim()).not.toBe('')
      expect(offeredWalks(category.id).length).toBeGreaterThan(0)
    }
  })

  it('never mixes Assistant-only and Code-only walks', () => {
    const assistant = offeredWalks('use', 'assistant').map((walk) => walk.id)
    const coding = offeredWalks('use', 'coding').map((walk) => walk.id)

    expect(assistant).toContain('assistant_desk')
    expect(assistant).toContain('workbench_panel')
    expect(assistant).not.toContain('code_desk')
    expect(assistant).not.toContain('code_map')

    expect(coding).toContain('code_desk')
    expect(coding).toContain('code_map')
    expect(coding).not.toContain('assistant_desk')
    expect(coding).not.toContain('workbench_panel')
  })

  it('offers coding-workbench actions on Code instead of the app-wide starter menu', () => {
    const starts = contextualGuideStarts('chat', 'coding')
    expect(starts.map((choice) => choice.id)).toEqual([
      'code_desk',
      'code_map',
      'topbar.tab.terminal',
      'topbar.tab.editor',
      'topbar.tab.diff',
      'topbar.tab.git_log',
      'topbar.tab.pr',
      'topbar.tab.browser',
    ])
    expect(starts.every((choice) => choice.label.trim().length > 0)).toBe(true)
  })

  it('offers Assistant actions without any Code-only entry', () => {
    const starts = contextualGuideStarts('chat', 'assistant')
    const ids = starts.map((choice) => choice.id)
    expect(ids).toContain('assistant_desk')
    expect(ids).not.toContain('code_desk')
    expect(ids).not.toContain('code_map')
  })
})

describe('every door into the guide opens the same way', () => {
  // A door that dives straight into a walk drops somebody into the middle of a
  // tour they never chose — which is what the owner kept seeing as "เด้ง"
  // (15 ก.ย. 2026). The greeting asks first and offers the walks as buttons;
  // only pressing one of those buttons starts a walk.
  it('no component starts a route directly — the walk buttons are the only way in', () => {
    const src = path.resolve(__dirname, '..')
    const offenders: string[] = []
    const walk = (dir: string) => {
      for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
        const full = path.join(dir, e.name)
        if (e.isDirectory()) walk(full)
        else if (/\.(svelte|ts)$/.test(e.name) && !full.includes(`${path.sep}test${path.sep}`)) {
          // Guide.svelte itself is where the offered buttons live, and the
          // tour's accept is an explicit yes to a walk that was named.
          if (e.name === 'Guide.svelte' || e.name === 'guideState.svelte.ts') continue
          for (const m of fs.readFileSync(full, 'utf-8').matchAll(/guide\.start\(\s*['"]/g)) {
            offenders.push(`${path.relative(src, full)} — ${m[0]}`)
          }
        }
      }
    }
    walk(src)
    expect(offenders, `these open a walk without asking first:\n${offenders.join('\n')}`).toEqual([])
  })
})

describe('the screen moving under it', () => {
  // The guide reads where it is off the page's own sign and nothing else, so
  // the test moves signs around rather than calling any app setter. What it is
  // pinning: an open guide must never be left talking about a page the person
  // has already left (owner, 15 ก.ย. 2026: *"เวลากดไปหน้าต่างๆขณะไกด์ มันยัง
  // ค้างแบบนี้อยู่ คือมันควรรู้ตัวด้วยสิว่าตอนนี้อยู่หน้าไหน"*).

  it('asks the way again from wherever the person has got to', async () => {
    mapped('sidebar.footer', { width: 200, height: 40 })
    await guide.start(undefined, 'settings.rail.brain')
    expect(guide.awaiting).toBe('sidebar.footer')

    // They opened the menu themselves, by their own route. The door the guide
    // was pointing at has gone with it.
    document.body.innerHTML = ''
    mapped('account.settings', { width: 180, height: 32 })
    guide.placeChanged('chat')

    await waitFor(() => expect(guide.awaiting).toBe('account.settings'))
    expect(guide.heading).toBe('settings.rail.brain')
  })

  it('skips an obsolete route stop when the person changes rooms themselves', async () => {
    mapped('topbar.door', { width: 120, height: 32 })
    await guide.start('first')
    expect(guide.route).toEqual({ id: 'first', at: 0 })

    document.body.innerHTML = ''
    mapped('settings.rail.brain', { width: 220, height: 40 })
    guide.placeChanged('settings.appearance')

    await waitFor(() => expect(guide.stopId).toBe('settings.rail.brain'))
    expect(guide.route).toEqual({ id: 'first', at: GUIDE_ROUTES.first.stops.indexOf('settings.rail.brain') })
  })

  it('introduces the real next button when a tour starts from another room', async () => {
    const sign = document.createElement('div')
    sign.setAttribute('data-guide-place', 'settings.voice')
    sign.getBoundingClientRect = () => ({ width: 800, height: 600 }) as DOMRect
    document.body.appendChild(sign)
    mapped('settings.rail.brain', { width: 220, height: 40 })

    await guide.start('first')

    expect(guide.route).toEqual({ id: 'first', at: GUIDE_ROUTES.first.stops.indexOf('settings.rail.brain') })
    expect(guide.stopId).toBe('settings.rail.brain')
    expect(guide.sentence).toContain(th['guide.settings.rail.brain.name'])
    expect(guide.sentence).not.toContain(th['guide.topbar.door.name'])
  })

  it('stops pointing at a button that is no longer there, and greets the new page', async () => {
    mapped('chat.send', { width: 36, height: 36 })
    await guide.start(undefined, 'chat.send')
    guide.route = { id: 'first', at: 0 }
    const said = guide.saySeq

    document.body.innerHTML = '' // the page they were on is gone
    guide.placeChanged('settings.models')

    expect(guide.stopId).toBeNull()
    expect(guide.awaiting).toBeNull()
    expect(guide.route).toBeNull() // a walk through a page nobody is on any more
    expect(guide.place).toBe('settings.models')
    expect(guide.saySeq).toBe(said + 1)
    expect(guide.sentence).toBe(greetingFor('settings.models'))
  })

  it('says nothing when what it is standing beside is still there', async () => {
    mapped('chat.send', { width: 36, height: 36 })
    await guide.start(undefined, 'chat.send')
    const said = guide.saySeq
    const moved = guide.moveSeq

    guide.placeChanged('chat') // a panel opened somewhere; the button is fine

    expect(guide.stopId).toBe('chat.send')
    expect(guide.saySeq).toBe(said)
    expect(guide.moveSeq).toBe(moved)
  })

  it('keeps the first welcome quiet across navigation, then follows pages after explanation starts', async () => {
    await guide.start()
    const said = guide.saySeq

    guide.placeChanged('settings.models')

    expect(guide.stopId).toBeNull()
    expect(guide.place).toBe('settings.models')
    expect(guide.saySeq).toBe(said)
    expect(guide.sentence).toBe(th['guide.welcome'])

    guide.describeCurrentPage()
    expect(guide.saySeq).toBe(said + 1)
    expect(guide.sentence).toBe(greetingFor('settings.models'))

    guide.placeChanged('office')
    expect(guide.place).toBe('office')
    expect(guide.saySeq).toBe(said + 2)
    expect(guide.sentence).toBe(greetingFor('office'))
  })

  it('re-introduces chat when the same page switches from assistant to coding', async () => {
    const here = document.createElement('div')
    here.setAttribute('data-guide-place', 'chat')
    here.setAttribute('data-guide-context', 'assistant')
    here.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(here)
    await guide.start()
    guide.describeCurrentPage()
    const said = guide.saySeq

    cockpit.desk = 'coding'
    guide.placeChanged('chat', true)

    expect(guide.saySeq).toBe(said + 1)
    expect(guide.sentence).toContain(th['guide.room.chat.coding.name'])
    expect(guide.sentence).toContain(th['guide.room.chat.coding.what'])
  })

  it('introduces the pane reached from the right-hand tab instead of looping back to Code', async () => {
    cockpit.desk = 'coding'
    const page = document.createElement('div')
    page.setAttribute('data-guide-place', 'chat')
    page.setAttribute('data-guide-context', 'coding')
    page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    const pane = document.createElement('div')
    pane.setAttribute('data-guide-context', 'workbench.terminal')
    pane.getBoundingClientRect = () => ({ width: 500, height: 700 }) as DOMRect
    page.appendChild(pane)
    document.body.appendChild(page)
    const target = mapped('topbar.tab.terminal', { width: 180, height: 32 })

    await guide.start(undefined, 'topbar.tab.terminal')
    target.remove()
    guide.placeChanged('chat', true)

    expect(guide.stopId).toBeNull()
    expect(guide.sentence).toContain(th['guide.topbar.tab.terminal.name'])
    expect(guide.sentence).toContain(th['guide.workbench.terminal.ask'])
    expect(guide.sentence).not.toContain(th['guide.room.chat.coding.name'])
  })

  it('ends an Assistant-only tour when the desk switches to Code', async () => {
    const here = document.createElement('div')
    here.setAttribute('data-guide-place', 'chat')
    here.setAttribute('data-guide-context', 'assistant')
    here.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
    document.body.appendChild(here)
    mapped('composer.who', { width: 180, height: 36 })
    await guide.start('assistant_desk')
    expect(guide.route?.id).toBe('assistant_desk')

    cockpit.desk = 'coding'
    here.setAttribute('data-guide-context', 'coding')
    guide.placeChanged('chat', true)

    expect(guide.route).toBeNull()
    expect(guide.stopId).toBeNull()
    expect(guide.sentence).toContain(th['guide.room.chat.coding.name'])
    expect(guide.sentence).not.toContain(th['guide.room.chat.assistant.name'])
  })

  it('does not react to its own footsteps — one press is one walk', async () => {
    mapped('sidebar.footer', { width: 200, height: 40 })
    await guide.start(undefined, 'settings.rail.brain')
    const moved = guide.moveSeq

    // The press it asked for lands, which is exactly what changes the page.
    // Both the press and the sign would send it walking; only one may.
    document.body.innerHTML = ''
    mapped('account.settings', { width: 180, height: 32 })
    guide.advance('sidebar.footer')
    guide.placeChanged('chat')

    await waitFor(() => expect(guide.awaiting).toBe('account.settings'), { timeout: 2000 })
    await new Promise((r) => setTimeout(r, 120))
    expect(guide.moveSeq).toBe(moved + 1)
  })
})

describe('the sign watcher', () => {
  const sign = (page: string) => {
    const el = document.createElement('div')
    el.setAttribute('data-guide-place', page)
    el.getBoundingClientRect = () =>
      ({ left: 0, top: 0, right: 800, bottom: 600, width: 800, height: 600, x: 0, y: 0, toJSON() {} }) as DOMRect
    document.body.appendChild(el)
    return el
  }
  const settle = () => new Promise((r) => setTimeout(r, 260))

  it('speaks up when the page changes, and stays quiet when it has not', async () => {
    const here = sign('chat')
    const seen: (string | null)[] = []
    const off = watchPlace((p) => seen.push(p))

    here.setAttribute('data-guide-place', 'settings.models')
    await settle()
    expect(seen).toEqual(['settings.models'])

    // Busy pages mutate constantly — a chat streaming, a list growing. None of
    // that is a change of place, and none of it may reach the guide.
    for (let i = 0; i < 20; i++) document.body.appendChild(document.createElement('span'))
    await settle()
    expect(seen).toEqual(['settings.models'])

    off()
    here.setAttribute('data-guide-place', 'office')
    await settle()
    expect(seen).toEqual(['settings.models']) // stopped means stopped
  })

  it('says null when the last sign goes away', async () => {
    sign('chat')
    const seen: (string | null)[] = []
    const off = watchPlace((p) => seen.push(p))
    document.body.innerHTML = ''
    await settle()
    expect(seen).toEqual([null])
    off()
  })

  it('reports a desk context change even while the page remains chat', async () => {
    const here = sign('chat')
    here.setAttribute('data-guide-context', 'assistant')
    const seen: string[] = []
    const off = watchPlace((page, contextChanged) => seen.push(`${page}:${contextChanged}`))

    here.setAttribute('data-guide-context', 'coding')
    await settle()

    expect(seen).toEqual(['chat:true'])
    off()
  })

  it('reports when the Code inspector opens and closes without changing pages', async () => {
    const seen: boolean[] = []
    const off = watchInspector((open) => seen.push(open))
    const add = mapped('workbench.add_tab', { width: 36, height: 36 })

    await settle()
    expect(seen).toEqual([true])

    add.remove()
    await settle()
    expect(seen).toEqual([true, false])
    off()
  })
})

// A guide you can pick up and put somewhere else.
//
// The line the drag must not cross: it decides where the figure WAITS, never
// where it goes. Walking to the thing it is explaining is the whole job
// (owner, 15 ก.ย. 2026: *"อยากให้ลากได้ครับ … แต่พอถึงเวลาอธิบายมันควรจะเดิน
// ไปเดินมาได้ เพราะมันคือไกด์"*).
describe('moving it by hand', () => {
  const grab = async (el: HTMLElement, from: [number, number], to: [number, number]) => {
    ;(el as any).setPointerCapture = () => {}
    await fireEvent.pointerDown(el, { button: 0, clientX: from[0], clientY: from[1] })
    await fireEvent.pointerMove(el, { clientX: to[0], clientY: to[1] })
    await fireEvent.pointerUp(el, {})
    await tick()
  }
  const spotOf = (c: HTMLElement) => {
    const m = c.querySelector<HTMLElement>('.guide-mascot-wrap')!.style.transform.match(/translate\((-?\d+)px,\s*(-?\d+)px\)/)
    return { x: Number(m![1]), y: Number(m![2]) }
  }

  beforeEach(() => {
    setGuidePref('spot', null)
    setGuidePref('size', GUIDE_SIZE_DEFAULT)
    vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(1240)
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(860)
  })

  it('goes where it is put, and waits there the next time it is opened', async () => {
    await guide.start()
    const { container, unmount } = render(Guide)
    const figure = container.querySelector<HTMLElement>('.ranked-guide')!
    const before = spotOf(container)

    await grab(figure, [before.x + 20, before.y + 20], [before.x + 20 - 300, before.y + 20 - 200])

    const after = spotOf(container)
    expect(after.x).toBe(before.x - 300)
    expect(after.y).toBe(before.y - 200)
    expect(guidePrefs.spot).toEqual(after)

    // Opened again tomorrow: the same spot, on the first frame, with no walk
    // across the window to get there.
    unmount()
    const second = render(Guide)
    expect(spotOf(second.container)).toEqual(after)
    second.unmount()
  })

  it('a press that does not move it is not a drag — the frame buttons still work', async () => {
    await guide.start()
    const { container } = render(Guide)
    const figure = container.querySelector<HTMLElement>('.ranked-guide')!
    const before = spotOf(container)

    await grab(figure, [before.x + 10, before.y + 10], [before.x + 12, before.y + 11]) // 2px: a click

    expect(spotOf(container)).toEqual(before)
    expect(guidePrefs.spot).toBeNull()
  })

  it('resizes from the compass corner with the companion limits and remembers', async () => {
    await guide.start()
    const { container } = render(Guide)
    const grip = container.querySelector<HTMLElement>('.guide-grip')!
    ;(grip as any).setPointerCapture = () => {}
    const figure = container.querySelector<HTMLElement>('.ranked-guide')!

    expect(figure.style.width).toBe(`${GUIDE_SIZE_DEFAULT}px`)
    await fireEvent.pointerDown(grip, { clientX: 100, clientY: 100, pointerId: 2, button: 0 })
    await fireEvent.pointerMove(grip, { clientX: 140, clientY: 120, pointerId: 2 })
    expect(guidePrefs.size).toBe(GUIDE_SIZE_DEFAULT + 40)
    expect(figure.style.width).toBe(`${GUIDE_SIZE_DEFAULT + 40}px`)

    await fireEvent.pointerMove(grip, { clientX: 2000, clientY: 100, pointerId: 2 })
    expect(guidePrefs.size).toBe(GUIDE_SIZE_MAX)
    await fireEvent.pointerMove(grip, { clientX: -2000, clientY: -2000, pointerId: 2 })
    expect(guidePrefs.size).toBe(GUIDE_SIZE_MIN)
    await fireEvent.pointerUp(grip, { pointerId: 2 })

    expect(JSON.parse(localStorage.getItem('guidePrefs') ?? '{}').size).toBe(GUIDE_SIZE_MIN)
  })

  it('still walks to what it explains, and comes back to where it was put', async () => {
    const button = mapped('chat.send', { left: 500, top: 300, width: 36, height: 36 })
    await guide.start()
    const { container } = render(Guide)
    const figure = container.querySelector<HTMLElement>('.ranked-guide')!
    const before = spotOf(container)
    await grab(figure, [before.x, before.y], [before.x - 400, before.y - 100])
    const parked = spotOf(container)

    // Asked about something: it leaves the parking spot for the target.
    guide.explain('chat.send')
    await waitFor(() => expect(spotOf(container).x).not.toBe(parked.x), { timeout: 2000 })

    // And when the thing it was standing beside is gone, home is where the
    // hand left it — never the corner it would have picked for itself.
    button.remove()
    guide.placeChanged('settings.models')
    await waitFor(() => expect(spotOf(container)).toEqual(parked), { timeout: 2000 })
  })
})

// Arriving is one measurement; staying is a watch. The page a guide has just
// opened is exactly where boxes keep moving afterwards.
describe('staying on what it points at', () => {
  const rect = (el: HTMLElement, left: number, top: number, width = 36, height = 36) => {
    el.getBoundingClientRect = () =>
      ({ left, top, right: left + width, bottom: top + height, width, height, x: left, y: top, toJSON() {} }) as DOMRect
  }
  const wrapY = (c: HTMLElement) => {
    const match = c.querySelector<HTMLElement>('.guide-mascot-wrap')?.style.transform.match(/translate\(\d+px,([-\d.]+)px\)/)
    return match ? match[1] : ''
  }

  it('follows the button when it moves after the page has settled', async () => {
    const button = mapped('chat.send', { left: 500, top: 300, width: 36, height: 36 })
    await guide.start(undefined, 'chat.send')
    const { container } = render(Guide)
    await waitFor(() => expect(wrapY(container)).toBe('282'), { timeout: 2000 })

    // A card above it finished loading and pushed it down the page.
    rect(button, 500, 700)
    document.dispatchEvent(new Event('scroll'))

    await waitFor(() => expect(wrapY(container)).toBe('680'), { timeout: 2000 })
  })

  it('lets go when the button disappears without the page changing', async () => {
    const button = mapped('chat.send', { left: 500, top: 300, width: 36, height: 36 })
    await guide.start(undefined, 'chat.send')
    const { container } = render(Guide)
    await waitFor(() => expect(guide.stopId).toBe('chat.send'))

    // A menu closing, a card collapsing: no sign moves for this, so only the
    // box itself can report it.
    button.remove()
    document.dispatchEvent(new Event('scroll'))

    await waitFor(() => expect(guide.stopId).toBeNull(), { timeout: 2000 })
  })
})

describe('avatar chat display and assistant standard quality', () => {
  it('shows only a short first greeting and waits for Explain this page before the full introduction', async () => {
    const originalMatchMedia = globalThis.matchMedia
    globalThis.matchMedia = (() => ({ matches: true })) as unknown as typeof matchMedia
    try {
      const page = document.createElement('div')
      page.setAttribute('data-guide-place', 'chat')
      page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
      document.body.appendChild(page)
      await guide.start()
      const { container } = render(Guide)

      await waitFor(() => expect(container.textContent).toContain(th['guide.welcome'].split('\n')[0]))
      const shiftTip = container.querySelector<HTMLButtonElement>('.say-shift-tip')
      expect(shiftTip).not.toBeNull()
      expect(shiftTip?.textContent).toContain('Shift')
      expect(shiftTip?.textContent).toContain(th['guide.shiftClickButton'])
      expect(container.querySelectorAll('.say-recommended-routes .say-command')).toHaveLength(6)
      expect(container.querySelector('.say-recommended-routes .say-command-list')?.classList.contains('say-start-grid')).toBe(true)
      expect(container.textContent).not.toContain(th['guide.room.chat.assistant.what'])
      const describe = Array.from(container.querySelectorAll<HTMLButtonElement>('.say-command'))
        .find((button) => button.textContent?.includes(th['guide.describeThisPage']))
      expect(describe).toBeTruthy()

      await fireEvent.click(describe!)
      await waitFor(() => expect(container.textContent).toContain(th['guide.room.chat.assistant.what']))
      expect(container.textContent).not.toContain(th['guide.welcome'].split('\n')[0])
    } finally {
      globalThis.matchMedia = originalMatchMedia
    }
  })

  it('automatically flips to the right when parked near the left window boundary', async () => {
    vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(1240)
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(860)
    setGuidePref('spot', { x: 120, y: 300 })
    await guide.start()
    const { container } = render(Guide)
    const wrap = container.querySelector<HTMLElement>('.guide-mascot-wrap')!
    expect(wrap.classList.contains('flip')).toBe(true)
  })

  it('renders conversational dialogue with markdown in chat stream', async () => {
    await guide.start()
    const { container } = render(Guide)
    guide.transcript = [
      { who: 'user', text: 'ปุ่มนี้ทำอะไร' },
      { who: 'guide', text: '**ปุ่มส่งข้อความ** สำหรับส่งงานให้ *ผู้ช่วย*' },
    ]
    await waitFor(() => expect(container.querySelector('.say-chat-stream')).not.toBeNull())
    expect(container.querySelector('.say-bubble.user')?.textContent).toContain('ปุ่มนี้ทำอะไร')
    const guideBubble = container.querySelector('.say-bubble.guide')
    expect(guideBubble?.innerHTML).toContain('<strong>')
    expect(guideBubble?.innerHTML).toContain('<em>')
  })

  it('offers the route Next control once the current point is already on screen', async () => {
    mapped('topbar.door', { width: 120, height: 32 })
    await guide.start()
    const { container } = render(Guide)
    guide.transcript = [
      { who: 'user', text: 'พาผมทัวหน่อย' },
      { who: 'guide', text: 'เริ่มทัวร์รอบแรกครับ' },
    ]
    await guide.beginRoute('first')

    await waitFor(() => expect(container.querySelector('.say-route-actions')).not.toBeNull())
    expect(container.querySelector('.say-step-actions')).toBeNull()
    expect(container.querySelector('.say-route-actions')?.textContent ?? '').toContain(th['guide.next'])
  })

  it('expands the conversation surface and can collapse it again', async () => {
    await guide.start()
    const { container } = render(Guide)

    const expand = await waitFor(() => {
      const button = container.querySelector<HTMLButtonElement>('button[aria-label="ขยายไกด์"]')
      expect(button).not.toBeNull()
      return button!
    })
    expect(container.querySelector('.say')?.classList.contains('expanded')).toBe(false)

    await fireEvent.click(expand)
    expect(container.querySelector('.say')?.classList.contains('expanded')).toBe(true)
    const collapse = container.querySelector<HTMLButtonElement>('button[aria-label="ย่อไกด์"]')
    expect(collapse?.getAttribute('aria-expanded')).toBe('true')

    await fireEvent.click(collapse!)
    expect(container.querySelector('.say')?.classList.contains('expanded')).toBe(false)

    await fireEvent.dblClick(container.querySelector('.say-head')!)
    expect(container.querySelector('.say')?.classList.contains('expanded')).toBe(true)
  })

  it('folds to a title bar without closing or losing the current guide point', async () => {
    mapped('capability.mcp.header', { width: 220, height: 40 })
    await guide.start(undefined, 'capability.mcp.header')
    const { container } = render(Guide)

    const fold = await waitFor(() => {
      const button = container.querySelector<HTMLButtonElement>('button[aria-label="พับไกด์ไม่ให้บังหน้าจอ"]')
      expect(button).not.toBeNull()
      return button!
    })
    await waitFor(() => expect(container.querySelector('.say-body')).not.toBeNull())

    await fireEvent.click(fold)
    expect(container.querySelector('.say')?.classList.contains('collapsed')).toBe(true)
    expect(container.querySelector<HTMLButtonElement>('button[aria-label="เปิดไกด์กลับมา"]')).not.toBeNull()
    expect(guide.stopId).toBe('capability.mcp.header')

    await fireEvent.click(container.querySelector<HTMLButtonElement>('button[aria-label="เปิดไกด์กลับมา"]')!)
    expect(container.querySelector('.say')?.classList.contains('collapsed')).toBe(false)
    expect(container.querySelector('.say-body')).not.toBeNull()
  })

  it('opens settings from a tab attached to the chat bubble without replacing the chat', async () => {
    await guide.start()
    const { container } = render(Guide)
    const gear = await waitFor(() => {
      const button = container.querySelector<HTMLButtonElement>('.say-settings-tab')
      expect(button).not.toBeNull()
      return button!
    })

    await fireEvent.click(gear)
    const chat = container.querySelector('.say')
    const panel = container.querySelector('.guide-settings-menu')
    expect(chat).not.toBeNull()
    expect(panel).not.toBeNull()
    expect(chat?.contains(panel)).toBe(true)
    expect(container.querySelector('.gp-menu')).not.toBeNull()
    expect(gear.getAttribute('aria-expanded')).toBe('true')

    await fireEvent.click(gear)
    expect(container.querySelector('.guide-settings-menu')).toBeNull()
  })

  it('shows three readable step commands at a time and rotates the rest', async () => {
    mapped('capability.mcp.header', { width: 240, height: 40 })
    const originalMatchMedia = globalThis.matchMedia
    globalThis.matchMedia = (() => ({ matches: true })) as unknown as typeof matchMedia
    try {
      await guide.start(undefined, 'capability.mcp.header')
      const { container } = render(Guide)

      await waitFor(() => expect(container.querySelectorAll('.say-command-list .say-command')).toHaveLength(3))

      const firstPage = container.querySelector('.say-command-list')?.textContent
      const rotate = container.querySelector<HTMLButtonElement>('.say-command-rotate')
      expect(rotate).not.toBeNull()
      await fireEvent.click(rotate!)
      const secondPage = container.querySelector('.say-command-list')?.textContent
      expect(secondPage).not.toBe(firstPage)
      await fireEvent.click(rotate!)
      expect(container.querySelector('.say-command-list')?.textContent).toBe(firstPage)
    } finally {
      globalThis.matchMedia = originalMatchMedia
    }
  })

  it('follows the three questions about a point with topics from other categories', async () => {
    mapped('capability.mcp.header', { width: 240, height: 40 })
    const originalMatchMedia = globalThis.matchMedia
    globalThis.matchMedia = (() => ({ matches: true })) as unknown as typeof matchMedia
    try {
      await guide.start(undefined, 'capability.mcp.header')
      const { container } = render(Guide)

      await waitFor(() => expect(container.querySelectorAll('.say-step-actions .say-command')).toHaveLength(3))
      expect(container.querySelector('.say-step-actions')?.textContent).toContain(th['guide.command.explainMore'])
      expect(container.querySelector('.say-step-actions')?.textContent).toContain(th['guide.command.commonUse'])
      expect(container.querySelector('.say-step-actions')?.textContent).toContain(th['guide.command.recommend'])

      await fireEvent.click(container.querySelector<HTMLButtonElement>('.say-step-actions .say-command-rotate')!)
      const otherTopics = container.querySelector('.say-step-actions')?.textContent ?? ''
      expect(otherTopics).toContain(th['guide.walk.brain'])
      expect(otherTopics).toContain(GUIDE_ROUTES.assistant_desk.name)
      expect(otherTopics).toContain(th['guide.walk.memory'])
    } finally {
      globalThis.matchMedia = originalMatchMedia
    }
  })

  it('keeps prepared route suggestions visible after a question becomes a transcript', async () => {
    await guide.start()
    guide.describeCurrentPage()
    guide.transcript = [
      { who: 'user', text: 'Aetox มีไว้ทำไม' },
      { who: 'guide', text: 'Aetox ช่วยคุณทำงานผ่านผู้ช่วยและเครื่องมือในแอป' },
    ]
    const { container } = render(Guide)

    await waitFor(() => expect(container.querySelector('.say-chat-stream')).not.toBeNull())
    const routes = container.querySelector('.say-recommended-routes')
    expect(routes).not.toBeNull()
    // With no signed page this falls back to the three top-level categories.
    expect(routes?.querySelectorAll('.say-command')).toHaveLength(3)
  })

  it('shows six page-aware starting choices while keeping overflow on the rotate button', async () => {
    const originalMatchMedia = globalThis.matchMedia
    globalThis.matchMedia = (() => ({ matches: true })) as unknown as typeof matchMedia
    try {
      const page = document.createElement('div')
      page.setAttribute('data-guide-place', 'chat')
      page.getBoundingClientRect = () => ({ width: 900, height: 700 }) as DOMRect
      document.body.appendChild(page)
      cockpit.desk = 'coding'
      await guide.start()
      const { container } = render(Guide)

      await waitFor(() => expect(container.querySelectorAll('.say-recommended-routes .say-command')).toHaveLength(6))
      expect(container.querySelector('.say-recommended-routes .say-command-list')?.classList.contains('say-start-grid')).toBe(true)
      expect(container.querySelector('.say-recommended-routes .say-command-rotate')).not.toBeNull()
      expect(container.querySelector('.say-step-actions')).toBeNull()
    } finally {
      globalThis.matchMedia = originalMatchMedia
    }
  })

  it('opens one main topic into three concrete guide questions and can return', async () => {
    await guide.start()
    guide.describeCurrentPage()
    guide.transcript = [{ who: 'guide', text: guide.sentence }]
    const { container } = render(Guide)

    await waitFor(() => expect(container.querySelectorAll('.say-category')).toHaveLength(3))
    const setup = Array.from(container.querySelectorAll<HTMLButtonElement>('.say-category'))
      .find((button) => button.textContent?.includes(th['guide.walkCategory.setup']))!
    await fireEvent.click(setup)

    expect(container.querySelector('.say-command-head')?.textContent).toContain(th['guide.walkCategory.setup'])
    expect(container.querySelectorAll('.say-recommended-routes .say-command')).toHaveLength(3)
    expect(container.querySelector('.say-recommended-routes')?.textContent).toContain(th['guide.walk.brain'])
    expect(container.querySelector('.say-recommended-routes')?.textContent).toContain(th['guide.walk.team'])
    expect(container.querySelector('.say-recommended-routes')?.textContent).toContain(th['guide.walk.connections'])

    await fireEvent.click(container.querySelector<HTMLButtonElement>('.say-command-back')!)
    expect(container.querySelectorAll('.say-category')).toHaveLength(3)
  })

  it('points at the requested button without drawing a screen-size-dependent outline', async () => {
    mapped('sidebar.footer', { left: 20, top: 500, right: 220, bottom: 540, width: 200, height: 40 })
    await guide.start('brain')
    const { container } = render(Guide)

    const target = await waitFor(() => {
      const pointer = container.querySelector<HTMLElement>('.guide-target-anchor.awaiting')
      expect(pointer).not.toBeNull()
      return pointer!
    })
    expect(target.style.left).toBe('120px')
    expect(target.textContent).toContain(th['guide.clickHere'])

    // After the press the quiet arrow keeps the explanation anchored to its
    // subject, but the explicit click instruction goes away.
    guide.awaiting = null
    const quiet = await waitFor(() => {
      const pointer = container.querySelector<HTMLElement>('.guide-target-anchor')
      expect(pointer).not.toBeNull()
      return pointer!
    })
    expect(quiet.classList.contains('awaiting')).toBe(false)
    expect(quiet.textContent).not.toContain(th['guide.clickHere'])
  })

  it('keeps the target anchor visually borderless and uses only the arrow marker', () => {
    const source = fs.readFileSync(path.resolve(__dirname, '../lib/guide/Guide.svelte'), 'utf-8')
    const anchorCss = source.match(/\.guide-target-anchor\s*\{([\s\S]*?)\}/)?.[1] ?? ''
    expect(anchorCss).not.toMatch(/\bborder\s*:/)
    expect(anchorCss).not.toMatch(/\bbox-shadow\s*:/)
    expect(source).toContain('<Icon name="arrowDown"')
    expect(source).not.toContain('guide-target-ring')
  })

})
