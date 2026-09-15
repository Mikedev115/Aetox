import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, fireEvent, waitFor } from '@testing-library/svelte'
import { tick } from 'svelte'
import { guide } from '../lib/guide/guideState.svelte'
import { mapPick } from '../lib/guide/mapPick'
import { GUIDE_ROUTES } from '../lib/guide/routes'
import { GUIDE_MAP } from '../lib/guide/map'
import { isShortcut, shortcutLabel } from '../lib/shortcuts'
import { greetingFor, offeredWalks } from '../lib/guide/greeting'
import Guide from '../lib/guide/Guide.svelte'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { th } from '../lib/locales/th'

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
  cockpit.model.provider = ''
  vi.restoreAllMocks()
})

describe('the guide store', () => {
  it('opens on a route at its first stop and closes to nothing', async () => {
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

  it('steps a route both ways, wrapping, and remembers where it was left', async () => {
    await guide.start('first')
    const stops = GUIDE_ROUTES.first.stops
    guide.next()
    await waitFor(() => expect(guide.stopId).toBe(stops[1]))
    guide.prev()
    await waitFor(() => expect(guide.stopId).toBe(stops[0]))
    guide.prev()
    await waitFor(() => expect(guide.stopId).toBe(stops[stops.length - 1]))
    // Nothing is kept: asking to be shown around starts at the beginning,
    // however far a previous walk got. Resuming silently is how somebody
    // presses "show me around" and lands on the MCP page (15 ก.ย. 2026).
    expect(localStorage.getItem('guideRoute')).toBeNull()
    await guide.stop()
    await guide.start('first')
    expect(guide.stopId).toBe(stops[0])
    expect(guide.route).toEqual({ id: 'first', at: 0 })
  })

  it('every walk bumps moveSeq once — a route step and a model point share one door', async () => {
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
    const send = mapped('chat.send')
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
    const door = mapped('topbar.door')
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
})

describe('the map brain', () => {
  it('answers ten Thai questions with the right stop', () => {
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
      ['connect brain', 'settings.rail.brain'],
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
})

describe('the guide on screen', () => {
  it('stands beside the target with the bubble on the far side, and says the map\'s words', async () => {
    vi.spyOn(window, 'innerWidth', 'get').mockReturnValue(1200)
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(800)
    mapped('sidebar.projects', { left: 20, top: 300, right: 180, bottom: 340, width: 160, height: 40 })
    await guide.start(undefined, 'sidebar.projects')
    const { container } = render(Guide)
    await waitFor(() => expect(container.querySelector('.guide-ring')).not.toBeNull())
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

  it('says a model\'s answer where it stands, without walking', async () => {
    mapped('topbar.door', { left: 500, top: 20, right: 600, bottom: 50, width: 100, height: 30 })
    await guide.start(undefined, 'topbar.door')
    const { container } = render(Guide)
    await waitFor(() => expect(container.querySelector('.guide-ring')).not.toBeNull())
    const moves = guide.moveSeq
    guide.say('this is the door')
    await waitFor(() => expect(container.querySelector('.say-body')?.textContent).toContain('this is the door'), { timeout: 3000 })
    expect(guide.moveSeq).toBe(moves)
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
  it('names the room it finds you in and asks what is unclear', () => {
    // jsdom gives every element a zero rect unless one is stubbed, and
    // explainableOnScreen counts only what has size — so the stub IS the test.
    mapped('chat.send', { width: 36, height: 36 })
    mapped('topbar.door', { width: 120, height: 32 })
    const inChat = greetingFor('chat')
    expect(inChat).toContain(th['guide.room.chat'])
    expect(inChat).toContain('2') // the two mapped things on screen
    expect(inChat).toMatch(/[?？]|ไหม|หรือเปล่า/)

    document.body.innerHTML = ''
    const bare = greetingFor('settings')
    expect(bare).toContain(th['guide.room.settings'])
    expect(bare).not.toContain('0') // never promises what it cannot point at
  })

  it('still asks when it does not know the room', () => {
    const unknown = greetingFor('some-new-room-nobody-mapped')
    expect(unknown).toBe(th['guide.greetPlain'])
    expect(unknown).toMatch(/[?？]|ไหม/)
  })

  it('greets by room when opened to be asked, rather than walking a route', async () => {
    cockpit.activeView = 'settings'
    await guide.start()
    expect(guide.route).toBeNull()
    expect(guide.stopId).toBeNull()
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

  it('moves on when the awaited button is pressed, and re-reads the screen each time', async () => {
    onChat()
    await guide.start(undefined, 'settings.rail.brain')
    expect(guide.awaiting).toBe('sidebar.footer')

    // Pressing it opens the menu — which is what makes the gear reachable.
    mapped('account.settings', { width: 180, height: 32 })
    guide.advance('sidebar.footer')
    await waitFor(() => expect(guide.awaiting).toBe('account.settings'), { timeout: 2000 })
    expect(guide.heading).toBe('settings.rail.brain')

    // Pressing THAT opens Settings, where the target itself is on screen.
    mapped('settings.rail.brain', { width: 220, height: 40 })
    guide.advance('account.settings')
    await waitFor(() => expect(guide.awaiting).toBeNull(), { timeout: 2000 })
    expect(guide.stopId).toBe('settings.rail.brain')
    expect(guide.heading).toBeNull()
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
})
