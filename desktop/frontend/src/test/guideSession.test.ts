import { describe, it, expect, vi, beforeEach } from 'vitest'
import { waitFor } from '@testing-library/svelte'
import { guide } from '../lib/guide/guideState.svelte'
import { handleGuideAsk, pageFrom } from '../lib/guide/guideSession'
import { CATALOG_ROUTES } from '../lib/guide/catalog/data'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { NewGuideSession, AskGuide, CloseGuideSession, AnswerGuide } from './mocks/wailsApp'
import { th } from '../lib/locales/th'
import fs from 'node:fs'
import path from 'node:path'

// Two brains, one guide (docs/architecture/ui-guide-2026-09-15.md §5): the
// model session opens beside the map and is never waited for; a model turn
// that fails hands the question to the map in the same breath; and the
// window's answers to the engine's `guide` tool are what the model sees.

function answered(id: string): any {
  const call = vi.mocked(AnswerGuide).mock.calls.find((c) => c[0] === id)
  expect(call, `no answer for ${id}`).toBeTruthy()
  return JSON.parse(call![1])
}

beforeEach(async () => {
  vi.clearAllMocks()
  await guide.stop()
  document.body.innerHTML = ''
  cockpit.activeView = 'chat'
  cockpit.settingsIntent = null
  cockpit.model.provider = ''
})

describe('the model brain', () => {
  it('opens a session beside the map when the chat has a provider, and not otherwise', async () => {
    await guide.start()
    expect(NewGuideSession).not.toHaveBeenCalled()
    expect(guide.brain).toBe('map')
    await guide.stop()

    cockpit.model.provider = 'ollama'
    vi.mocked(NewGuideSession).mockResolvedValueOnce('guide-1')
    await guide.start()
    // The map answers meanwhile; the session lands when it lands.
    await waitFor(() => expect(guide.brain).toBe('model'))
    expect(guide.sessionId).toBe('guide-1')
    const index = JSON.parse(vi.mocked(NewGuideSession).mock.calls[0][0])
    expect(index.find((e: any) => e.id === 'chat.send')).toEqual({ id: 'chat.send', name: th['guide.chat.send.name'], safe: false })
  })

  it('closes the session with the guide, and a session that opens after the guide closed is closed too', async () => {
    cockpit.model.provider = 'ollama'
    vi.mocked(NewGuideSession).mockResolvedValueOnce('guide-2')
    await guide.start()
    await waitFor(() => expect(guide.sessionId).toBe('guide-2'))
    await guide.stop()
    expect(CloseGuideSession).toHaveBeenCalledWith('guide-2')
    expect(guide.sessionId).toBeNull()

    let land: (id: string) => void = () => {}
    vi.mocked(NewGuideSession).mockReturnValueOnce(new Promise((r) => (land = r)))
    await guide.start()
    await guide.stop()
    land('guide-late')
    await waitFor(() => expect(CloseGuideSession).toHaveBeenCalledWith('guide-late'))
    expect(guide.sessionId).toBeNull()
  })

  it('asks the model and says its answer where the figure stands', async () => {
    cockpit.model.provider = 'ollama'
    vi.mocked(NewGuideSession).mockResolvedValueOnce('guide-3')
    vi.mocked(AskGuide).mockResolvedValueOnce('The send button, right there.')
    await guide.start(undefined, 'chat.send')
    await waitFor(() => expect(guide.brain).toBe('model'))
    const moves = guide.moveSeq
    const says = guide.saySeq
    const reply = await guide.ask('what is this?')
    expect(AskGuide).toHaveBeenCalledWith('guide-3', 'what is this?')
    expect(reply).toBe('The send button, right there.')
    expect(guide.saySeq).toBe(says + 1)
    expect(guide.moveSeq).toBe(moves)
    expect(guide.transcript.map((m) => m.who)).toEqual(['user', 'guide'])
  })

  it('hands an explicit tour request to the prepared core route without asking the model to invent it', async () => {
    cockpit.model.provider = 'ollama'
    vi.mocked(NewGuideSession).mockResolvedValueOnce('guide-tour')
    const door = document.createElement('button')
    door.setAttribute('data-guide', 'topbar.door')
    door.getBoundingClientRect = () => ({ width: 120, height: 32 }) as DOMRect
    document.body.appendChild(door)
    await guide.start()
    await waitFor(() => expect(guide.brain).toBe('model'))

    const reply = await guide.ask('พาผมทัวหน่อย')

    expect(AskGuide).not.toHaveBeenCalled()
    expect(guide.route).toEqual({ id: 'first', at: 0 })
    expect(guide.stopId).toBe('topbar.door')
    expect(reply).toContain('ทีละจุด')
    expect(reply).toContain(th['guide.topbar.door.name'])
  })

  it('falls back to the map in the same turn when the model fails, and stays there', async () => {
    cockpit.model.provider = 'ollama'
    vi.mocked(NewGuideSession).mockResolvedValueOnce('guide-4')
    vi.mocked(AskGuide).mockRejectedValueOnce(new Error('connection refused'))
    await guide.start()
    await waitFor(() => expect(guide.brain).toBe('model'))
    const send = document.createElement('button')
    send.setAttribute('data-guide', 'chat.send')
    send.getBoundingClientRect = () => ({ width: 36, height: 36 }) as DOMRect
    document.body.appendChild(send)
    const reply = await guide.ask('ปุ่มส่งข้อความอยู่ไหน')
    expect(guide.brain).toBe('map')
    expect(reply).toBeTruthy()
    expect(guide.stopId).toBe('chat.send')
    await guide.ask('ความจำ')
    expect(AskGuide).toHaveBeenCalledTimes(1)
  })
})

describe('the window answering the guide tool', () => {
  it('keeps model navigation on the visible guide runtime, never app page setters', () => {
    const root = path.resolve(__dirname, '..', 'lib', 'guide')
    for (const file of ['guideState.svelte.ts', 'guideSession.ts']) {
      const src = fs.readFileSync(path.join(root, file), 'utf-8')
      expect(src).not.toMatch(/from ['"]\.\/pages['"]/)
      expect(src).not.toMatch(/\bopenPage\s*\(/)
      expect(src).not.toMatch(/\bsetActiveView\s*\(/)
    }
  })

  it('where: the page and what is on it', async () => {
    const context = document.createElement('div')
    context.setAttribute('data-guide-context', 'workbench.git')
    context.getBoundingClientRect = () => ({ width: 500, height: 700 }) as DOMRect
    document.body.appendChild(context)
    const el = document.createElement('button')
    el.setAttribute('data-guide', 'chat.send')
    el.getBoundingClientRect = () => ({ width: 10, height: 10 }) as DOMRect
    document.body.appendChild(el)
    await guide.start(undefined, 'chat.send')
    await handleGuideAsk({ id: 'w1', action: 'where', args: {} })
    const res = answered('w1')
    expect(res.page).toBe('chat')
    expect(res.context).toBe('workbench.git')
    expect(res.guideAt).toBe('chat.send')
    expect(res.visible).toEqual([{ id: 'chat.send', name: th['guide.chat.send.name'], safe: false }])
  })

  it('describe: the map entry in the window\'s language', async () => {
    await handleGuideAsk({ id: 'd1', action: 'describe', args: { id: 'settings.rail.brain' } })
    const res = answered('d1')
    expect(res.name).toBe(th['guide.settings.rail.brain.name'])
    expect(res.what).toBe(th['guide.settings.rail.brain.what'])
    expect(res.why).toBe(th['guide.settings.rail.brain.why'])
    expect(res.ref).toBe('§279')
    expect(res.safe).toBe(true)
  })

  it('press: refused by the window for anything the map does not mark safe', async () => {
    await handleGuideAsk({ id: 'p1', action: 'press', args: { id: 'chat.send' } })
    expect(answered('p1')).toEqual({ ok: false, error: th['guide.safeRefusal'] })
    const door = document.createElement('button')
    door.setAttribute('data-guide', 'topbar.door')
    door.getBoundingClientRect = () => ({ width: 120, height: 32 }) as DOMRect
    const clicked = vi.fn()
    door.addEventListener('click', clicked)
    document.body.appendChild(door)
    await handleGuideAsk({ id: 'p2', action: 'press', args: { id: 'topbar.door' } })
    expect(answered('p2')).toEqual({ ok: true })
    await new Promise((r) => setTimeout(r, 0))
    expect(clicked).toHaveBeenCalledTimes(1)
  })

  it('point and goto: an unknown id or page is an error, a known one moves the guide', async () => {
    await guide.start()
    await handleGuideAsk({ id: 'x1', action: 'point', args: { id: 'nowhere.at.all' } })
    expect(answered('x1').ok).toBe(false)
    const send = document.createElement('button')
    send.setAttribute('data-guide', 'chat.send')
    send.getBoundingClientRect = () => ({ width: 36, height: 36 }) as DOMRect
    document.body.appendChild(send)
    await handleGuideAsk({ id: 'x2', action: 'point', args: { id: 'chat.send' } })
    expect(answered('x2').ok).toBe(true)
    expect(guide.stopId).toBe('chat.send')
    await handleGuideAsk({ id: 'x3', action: 'goto', args: { page: 'moon' } })
    expect(answered('x3').ok).toBe(false)
    const footer = document.createElement('button')
    footer.setAttribute('data-guide', 'sidebar.footer')
    footer.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(footer)
    await handleGuideAsk({ id: 'x4', action: 'goto', args: { page: 'settings.models' } })
    expect(answered('x4')).toEqual({
      ok: true,
      status: 'guiding',
      destination: 'settings.models',
      page: 'chat',
      nextStepId: 'sidebar.footer',
    })
    expect(cockpit.activeView).toBe('chat')
    expect(cockpit.settingsIntent).toBeNull()
  })

  it('names places in the app’s one vocabulary, and refuses anything else', () => {
    expect(pageFrom('chat')).toBe('chat')
    expect(pageFrom('settings.models')).toBe('settings.models')
    expect(pageFrom('capability.skills')).toBe('capability.skills')
    expect(pageFrom('office')).toBe('office')
    // A map id is accepted too — the place it lives in is meant.
    expect(pageFrom('settings.rail.brain')).toBe('settings.models')
    expect(pageFrom('nope')).toBeNull()
    expect(pageFrom('settings.nosuchsection')).toBeNull()
  })

  it('find: searches catalog targets and concepts without context dump', async () => {
    await handleGuideAsk({ id: 'f1', action: 'find', args: { query: 'ต่อสมอง' } })
    const res = answered('f1')
    expect(res.stopId).toBeTruthy()
    expect(res.sentence).toBeTruthy()
    expect(res.confidence).toBeGreaterThan(0)
  })

  it('route: returns deterministic navigation steps and availability', async () => {
    const footer = document.createElement('button')
    footer.setAttribute('data-guide', 'sidebar.footer')
    footer.getBoundingClientRect = () => ({ width: 180, height: 36 }) as DOMRect
    document.body.appendChild(footer)
    await handleGuideAsk({ id: 'r1', action: 'route', args: { id: 'settings.rail.brain' } })
    const res = answered('r1')
    expect(res.targetId).toBe('settings.rail.brain')
    expect(res.stepsRemaining).toBeDefined()
    expect(res.reachable).toBe(true)
  })

  it('tours and tour expose prepared core routes and start one by id or intent', async () => {
    await handleGuideAsk({ id: 'ts1', action: 'tours', args: {} })
    const listed = answered('ts1')
    expect(listed.tours.map((tour: any) => tour.id)).toEqual(CATALOG_ROUTES.map((route) => route.id))

    const door = document.createElement('button')
    door.setAttribute('data-guide', 'topbar.door')
    door.getBoundingClientRect = () => ({ width: 120, height: 32 }) as DOMRect
    document.body.appendChild(door)
    await guide.start()
    await handleGuideAsk({ id: 'ts2', action: 'tour', args: { query: 'พาทัว Aetox' } })
    expect(answered('ts2')).toMatchObject({ ok: true, route: 'first', currentStopId: 'topbar.door' })
    expect(guide.route).toEqual({ id: 'first', at: 0 })
  })
})
