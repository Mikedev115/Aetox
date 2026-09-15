import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { SIZE, BUBBLE, standBeside, restingSpot, walkTurn, walkMs, type Rect } from '../lib/guide/walk'
import { GUIDE_MAP } from '../lib/guide/map'
import { CAPABILITY_PAGES, SETTINGS_SECTIONS, PAGE_IDS } from '../lib/rooms'

// The gait (walk.ts) is arithmetic over rectangles, so it is tested as
// arithmetic — the rules it enforces are the ones a screenshot cannot pin:
// the card never covers what it points at, and the figure never leaves the
// window.

const WIN = { width: 1240, height: 860 }
const rect = (left: number, top: number, w = 120, h = 36): Rect => ({
  left, top, right: left + w, bottom: top + h, width: w, height: h,
})

describe('where the figure stands', () => {
  it('stands right of a target on the left, with the card opening further right', () => {
    const s = standBeside(rect(20, 300), WIN)
    expect(s.x).toBeGreaterThan(140) // past the target's right edge
    expect(s.flip).toBe(true) // card opens right, away from the target
    expect(s.x + SIZE + BUBBLE).toBeLessThanOrEqual(WIN.width)
  })

  it('stands left of a target on the right, with the card opening further left', () => {
    const s = standBeside(rect(1100, 300), WIN)
    expect(s.x).toBeLessThan(1100)
    expect(s.flip).toBe(false) // card opens left, away from the target
    expect(s.x).toBeGreaterThanOrEqual(0)
  })

  it('never covers the target it points at, wherever the target is', () => {
    for (let left = 0; left <= WIN.width - 120; left += 40) {
      const t = rect(left, 400)
      const s = standBeside(t, WIN)
      const card = s.flip
        ? { left: s.x + SIZE + 14, right: s.x + SIZE + 14 + BUBBLE }
        : { left: s.x - 14 - BUBBLE, right: s.x - 14 }
      const overlapsTarget = card.left < t.right && card.right > t.left
      // The one allowed exception: a window too narrow for the card on either
      // side. Then the card overlaps rather than hanging off the screen.
      const roomEitherSide =
        t.right + 16 + SIZE + 14 + BUBBLE <= WIN.width || t.left - SIZE - 16 - 14 - BUBBLE >= 0
      if (roomEitherSide) expect(overlapsTarget, `target at ${left}`).toBe(false)
    }
  })

  it('keeps the figure inside the window and clear of the top bar', () => {
    for (const t of [rect(600, 0), rect(600, 840), rect(0, 0), rect(1120, 830)]) {
      const s = standBeside(t, WIN)
      expect(s.x).toBeGreaterThanOrEqual(0)
      expect(s.y).toBeGreaterThanOrEqual(48)
      expect(s.x + SIZE).toBeLessThanOrEqual(WIN.width)
      expect(s.y + SIZE).toBeLessThanOrEqual(WIN.height)
    }
  })

  it('hangs the card below the figure only when it stands high up', () => {
    expect(standBeside(rect(600, 20), WIN).below).toBe(true)
    expect(standBeside(rect(600, 600), WIN).below).toBe(false)
  })

  it('rests in the lower right with room for the card beside it', () => {
    const h = restingSpot(WIN)
    expect(h.x + SIZE).toBeLessThanOrEqual(WIN.width)
    expect(h.x - BUBBLE).toBeGreaterThan(0) // the card opens left, on screen
    expect(h.y + SIZE).toBeLessThanOrEqual(WIN.height)
  })

  it('turns the way it walks, and does not walk at all under reduced motion', () => {
    expect(walkTurn(-300)).toBeLessThan(0)
    expect(walkTurn(300)).toBeGreaterThan(0)
    expect(walkMs(0, false)).toBeGreaterThan(0)
    expect(walkMs(900, false)).toBeLessThanOrEqual(800)
    expect(walkMs(900, false)).toBeGreaterThan(walkMs(10, false))
    expect(walkMs(900, true)).toBe(0)
  })
})

// The bug this guards: the map named the capability room's `mcp` and
// `builtins` pages, which have never existed (`mine` and `tools`), and
// Capability.arriveAt drops an id it does not know without a word — so the
// guide walked to the room, landed on the wrong page and reported the button
// missing. The types in lib/rooms.ts stop a new one being written; this stops
// the lists themselves drifting from the rails they describe.
describe('the map may only name pages the rooms actually have', () => {
  const read = (rel: string) => fs.readFileSync(path.resolve(__dirname, '..', rel), 'utf-8')

  it('every place a map row names is a real place', () => {
    for (const e of GUIDE_MAP) {
      expect(PAGE_IDS, `${e.id} names place "${e.page}"`).toContain(e.page)
    }
  })

  // The other half of the same contract: a place the map sends people to has
  // to be able to say it has arrived, or the guide waits for a sign that never
  // comes and falls back to "not on screen".
  it('every place the map names wears a sign somewhere in the UI', () => {
    const src = path.resolve(__dirname, '..')
    const signed = new Set<string>()
    const walk = (dir: string) => {
      for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
        const full = path.join(dir, e.name)
        if (e.isDirectory()) walk(full)
        else if (e.name.endsWith('.svelte')) {
          const text = fs.readFileSync(full, 'utf-8')
          for (const m of text.matchAll(/data-guide-place="([^"]+)"/g)) {
            // A sign built from a variable — "settings.{active}" — covers
            // every id with that prefix; the rail's own ids are checked
            // against rooms.ts separately.
            const v = m[1]
            if (v.includes('{')) signed.add(v.slice(0, v.indexOf('{')))
            else signed.add(v)
          }
        }
      }
    }
    walk(src)
    const covered = (id: string) => [...signed].some((s) => (s.endsWith('.') ? id.startsWith(s) : s === id))
    const orphans = [...new Set(GUIDE_MAP.map((e) => e.page))].filter((p) => !covered(p))
    expect(orphans, `places the map names but nothing signs: ${orphans.join(', ')}`).toEqual([])
  })

  it('rooms.ts still lists what the rooms draw', () => {
    const railIds = [...read('lib/Capability.svelte').matchAll(/\{ id: '([a-z_]+)', labelKey:/g)].map((m) => m[1])
    expect(railIds.length, 'the RAIL regex found nothing — Capability.svelte changed shape').toBeGreaterThan(8)
    for (const id of railIds) expect(CAPABILITY_PAGES, `capability rail row "${id}"`).toContain(id)

    // Scoped to the `sections` array: the file has other { id, label } lists
    // (the agent-look states, for one) that are not rail rows. A nav row is
    // the one that carries an icon.
    const settings = read('lib/Settings.svelte')
    const navStart = settings.indexOf('const sections:')
    expect(navStart, 'the sections array is gone from Settings.svelte').toBeGreaterThan(0)
    const navBlock = settings.slice(navStart, settings.indexOf('\n  function ', navStart))
    const navIds = [...navBlock.matchAll(/\{ id: '([a-z_]+)', label: [^\n]*?icon: /g)].map((m) => m[1])
    expect(navIds.length, 'the sections regex found nothing — Settings.svelte changed shape').toBeGreaterThan(10)
    for (const id of navIds) expect(SETTINGS_SECTIONS, `settings section "${id}"`).toContain(id)
  })
})
