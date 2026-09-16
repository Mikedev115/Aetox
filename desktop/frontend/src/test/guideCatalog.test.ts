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
    expect(concepts.length).toBeGreaterThanOrEqual(5)

    for (const c of concepts) {
      expect(c.evidence?.ref).toBeTruthy()
      const match = c.evidence!.ref!.match(/^§([0-9.]+)$/)
      expect(match, `Invalid ref format "${c.evidence?.ref}" on concept ${c.id}`).toBeTruthy()
      const sectionNum = match![1]
      const pattern = new RegExp(`(^|\\n)#{2,3}\\s+${sectionNum.replace('.', '\\.')}([.\\s]|$)`, 'm')
      expect(pattern.test(decisionsContent), `Section ${c.evidence?.ref} on ${c.id} not found in DECISIONS.md`).toBe(true)
    }
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
