import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { GUIDE_CATALOG, CATALOG_TARGETS } from '../lib/guide/catalog/data'
import { GUIDE_MAP } from '../lib/guide/map'
import { th } from '../lib/locales/th'
import { en } from '../lib/locales/en'
import { zh } from '../lib/locales/zh'

describe('Guide Catalog Source of Truth & Contracts', () => {
  it('CATALOG_TARGETS matches GUIDE_MAP 1:1 in order and ids', () => {
    expect(CATALOG_TARGETS.length).toBe(GUIDE_MAP.length)
    for (let i = 0; i < CATALOG_TARGETS.length; i++) {
      expect(CATALOG_TARGETS[i].id).toBe(GUIDE_MAP[i].id)
      expect(CATALOG_TARGETS[i].page).toBe(GUIDE_MAP[i].page)
      expect(CATALOG_TARGETS[i].policy.safe).toBe(GUIDE_MAP[i].safe)
      expect(CATALOG_TARGETS[i].evidence?.ref).toBe(GUIDE_MAP[i].ref)
    }
  })

  it('every catalog concept references valid DECISIONS sections', () => {
    const repoRoot = path.resolve(__dirname, '../../../..')
    const decisionsPath = path.join(repoRoot, 'docs/DECISIONS.md')
    const decisionsContent = fs.readFileSync(decisionsPath, 'utf-8')

    const concepts = GUIDE_CATALOG.filter((e) => e.kind === 'concept')
    expect(concepts.length).toBeGreaterThanOrEqual(9)

    for (const c of concepts) {
      expect(c.evidence?.ref).toBeTruthy()
      const match = c.evidence!.ref!.match(/^§([0-9.]+)$/)
      expect(match, `Invalid ref format "${c.evidence?.ref}" on concept ${c.id}`).toBeTruthy()
      const sectionNum = match![1]
      const pattern = new RegExp(`(^|\\n)#{2,3}\\s+${sectionNum.replace('.', '\\.')}([.\\s]|$)`, 'm')
      expect(pattern.test(decisionsContent), `Section ${c.evidence?.ref} on ${c.id} not found in DECISIONS.md`).toBe(true)
    }
  })

  it('every catalog route has valid stops, references valid DECISIONS sections, and has th/en/zh translations', () => {
    const repoRoot = path.resolve(__dirname, '../../../..')
    const decisionsPath = path.join(repoRoot, 'docs/DECISIONS.md')
    const decisionsContent = fs.readFileSync(decisionsPath, 'utf-8')

    const routes = GUIDE_CATALOG.filter((e) => e.kind === 'route') as any[]
    expect(routes.length).toBeGreaterThanOrEqual(3)

    const targetIds = new Set(CATALOG_TARGETS.map((t) => t.id))
    const locales = { th, en, zh }

    for (const r of routes) {
      expect(r.stops.length).toBeGreaterThan(0)
      for (const stop of r.stops) {
        expect(targetIds.has(stop), `Route ${r.id} references stop ${stop} which does not exist in CATALOG_TARGETS`).toBe(true)
      }

      if (r.evidence?.ref) {
        const match = r.evidence.ref.match(/^§([0-9.]+)$/)
        expect(match, `Invalid ref format "${r.evidence.ref}" on route ${r.id}`).toBeTruthy()
        const sectionNum = match![1]
        const pattern = new RegExp(`(^|\\n)#{2,3}\\s+${sectionNum.replace('.', '\\.')}([.\\s]|$)`, 'm')
        expect(pattern.test(decisionsContent), `Section ${r.evidence.ref} on route ${r.id} not found in DECISIONS.md`).toBe(true)
      }

      for (const [lang, dict] of Object.entries(locales)) {
        const d = dict as Record<string, string>
        expect(d[r.nameKey]?.trim(), `Missing ${r.nameKey} in ${lang}`).toBeTruthy()
        expect(d[r.descKey]?.trim(), `Missing ${r.descKey} in ${lang}`).toBeTruthy()
      }
    }
  })

  it('the team tour targets the real form while navigation supplies its doors', () => {
    const team = GUIDE_CATALOG.find((entry) => entry.kind === 'route' && entry.id === 'team')
    expect(team?.kind).toBe('route')
    if (!team || team.kind !== 'route') return
    expect(team.stops).toEqual([
      'team.name_input',
      'team.lead_select',
      'team.save_btn',
    ])
    // Rail and New Team are navigation prerequisites. Keeping them out of the
    // timeline prevents the tour from pausing on a door before it is pressed.
    expect(team.stops).not.toContain('settings.rail.teams')
    expect(team.stops).not.toContain('team.new_btn')
    expect(team.stops).not.toContain('team.add_member_btn')
    const name = GUIDE_CATALOG.find((entry) => entry.kind === 'target' && entry.id === 'team.name_input')
    expect(name?.kind).toBe('target')
    if (name?.kind === 'target') expect(name.navigation?.via).toEqual(['team.new_btn'])
  })

  it('the brain tour teaches the common ChatGPT/Codex door before local and custom providers', () => {
    const brain = GUIDE_CATALOG.find((entry) => entry.kind === 'route' && entry.id === 'brain')
    expect(brain?.kind).toBe('route')
    if (!brain || brain.kind !== 'route') return
    expect(brain.stops).toEqual([
      'settings.brain.hero',
      'settings.brain.provider.codex',
      'settings.brain.provider.ollama',
      'settings.brain.add_provider',
    ])
    expect(th['guide.settings.rail.brain.common']).toContain('ChatGPT')
    expect(th['guide.settings.rail.brain.common']).toContain('Codex')
    expect(th['guide.settings.rail.brain.recommend']).toContain('OpenRouter')
  })

  it('destination tours stop on page content while the graph supplies room and rail doors', () => {
    const stops = (id: string) => {
      const route = GUIDE_CATALOG.find((entry) => entry.kind === 'route' && entry.id === id)
      return route?.kind === 'route' ? route.stops : []
    }

    expect(stops('capabilities')).toEqual([
      'capability.mcp.header',
      'capability.skills.header',
      'capability.builtins.header',
      'capability.computer.header',
      'capability.connections.header',
      'capability.prompts.header',
    ])
    expect(stops('connections')).toEqual(['capability.connections.header'])
    expect(stops('memory')).toEqual(['settings.head.memory_panel'])
    expect(stops('artifacts')).toEqual(['artifacts.header'])
  })

  it('every catalog concept has full translations in th, en, zh', () => {
    const concepts = GUIDE_CATALOG.filter((e) => e.kind === 'concept')
    const locales = { th, en, zh }

    for (const c of concepts) {
      const nameKey = `guide.${c.id}.name`
      const whatKey = `guide.${c.id}.what`
      const whyKey = `guide.${c.id}.why`

      for (const [lang, dict] of Object.entries(locales)) {
        const d = dict as Record<string, string>
        expect(d[nameKey]?.trim(), `Missing ${nameKey} in ${lang}`).toBeTruthy()
        expect(d[whatKey]?.trim(), `Missing ${whatKey} in ${lang}`).toBeTruthy()
        expect(d[whyKey]?.trim(), `Missing ${whyKey} in ${lang}`).toBeTruthy()
      }
    }
  })

  it('no catalog target with safe: true ends with dangerous suffixes', () => {
    const dangerous = ['.send', '.stop', '.save', '.delete', '.toggle']
    for (const target of CATALOG_TARGETS) {
      if (target.policy.safe) {
        for (const suffix of dangerous) {
          expect(target.id.endsWith(suffix), `Target ${target.id} with safe:true has dangerous suffix ${suffix}`).toBe(false)
        }
      }
    }
  })
})
