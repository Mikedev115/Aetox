import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { GUIDE_MAP } from '../lib/guide/map'
import { openPage } from '../lib/guide/pages'
import { cockpit } from '../lib/stores/cockpit.svelte'
import { th } from '../lib/locales/th'
import { en } from '../lib/locales/en'
import { zh } from '../lib/locales/zh'

function findSvelteFiles(dir: string): string[] {
  let files: string[] = []
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files = files.concat(findSvelteFiles(full))
    } else if (entry.name.endsWith('.svelte')) {
      files.push(full)
    }
  }
  return files
}

// The invariant is that the three copies of the map agree — the ids in the
// .svelte files, the rows in map.ts, and the rows in docs/GUIDE-MAP.md — not
// that there are a particular number of them. A hard count is a test that has
// to be edited every time a button is mapped, which trains whoever is mapping
// buttons to edit the test without reading it. The floor stays, so the map
// cannot quietly empty out.
const MAP_FLOOR = 150

describe('GUIDE_MAP and UI data-guide integrity', () => {
  it('carries no duplicate ids, and does not shrink below the floor', () => {
    expect(GUIDE_MAP.length).toBeGreaterThanOrEqual(MAP_FLOOR)
    expect(new Set(GUIDE_MAP.map((e) => e.id)).size).toBe(GUIDE_MAP.length)
  })

  it('every data-guide attribute in .svelte matches GUIDE_MAP 1:1', () => {
    const frontendSrc = path.resolve(__dirname, '..')
    const svelteFiles = findSvelteFiles(frontendSrc)
    const foundTags = new Set<string>()

    const tagRegex = /data-guide="([^"]+)"/g
    for (const file of svelteFiles) {
      const content = fs.readFileSync(file, 'utf-8')
      let m: RegExpExecArray | null
      while ((m = tagRegex.exec(content)) !== null) {
        foundTags.add(m[1])
      }
    }

    const mapIds = new Set(GUIDE_MAP.map((e) => e.id))

    const missingInSvelte = [...mapIds].filter((id) => !foundTags.has(id))
    const extraInSvelte = [...foundTags].filter((id) => !mapIds.has(id))

    expect(missingInSvelte, `GUIDE_MAP entries missing in .svelte: ${missingInSvelte.join(', ')}`).toEqual([])
    expect(extraInSvelte, `Extra data-guide tags in .svelte not in GUIDE_MAP: ${extraInSvelte.join(', ')}`).toEqual([])
    expect(foundTags.size).toBe(mapIds.size)
  })

  it('every entry has guide.<id>.name and guide.<id>.what in all 3 locales (th, en, zh)', () => {
    const locales = {
      th: th as Record<string, string>,
      en: en as Record<string, string>,
      zh: zh as Record<string, string>,
    }

    for (const entry of GUIDE_MAP) {
      const nameKey = `guide.${entry.id}.name`
      const whatKey = `guide.${entry.id}.what`

      for (const [lang, dict] of Object.entries(locales)) {
        const nameVal = dict[nameKey]
        const whatVal = dict[whatKey]

        expect(nameVal, `Missing or empty ${nameKey} in ${lang}`).toBeDefined()
        expect(nameVal?.trim().length, `Empty ${nameKey} in ${lang}`).toBeGreaterThan(0)

        expect(whatVal, `Missing or empty ${whatKey} in ${lang}`).toBeDefined()
        expect(whatVal?.trim().length, `Empty ${whatKey} in ${lang}`).toBeGreaterThan(0)
      }
    }
  })

  it('every ref (§<num>) corresponds to a real section heading in docs/DECISIONS.md', () => {
    const repoRoot = path.resolve(__dirname, '../../../..')
    const decisionsPath = path.join(repoRoot, 'docs/DECISIONS.md')
    expect(fs.existsSync(decisionsPath), `docs/DECISIONS.md must exist at ${decisionsPath}`).toBe(true)

    const decisionsContent = fs.readFileSync(decisionsPath, 'utf-8')

    for (const entry of GUIDE_MAP) {
      if (!entry.ref) continue
      const match = entry.ref.match(/^§([0-9.]+)$/)
      expect(match, `Invalid ref format "${entry.ref}" on ${entry.id}`).toBeTruthy()
      const sectionNum = match![1]

      // Matches: ## 279. or ### 97.3
      const pattern = new RegExp(`(^|\\n)#{2,3}\\s+${sectionNum.replace('.', '\\.')}([.\\s]|$)`, 'm')
      expect(
        pattern.test(decisionsContent),
        `Section ${entry.ref} referenced by ${entry.id} not found in docs/DECISIONS.md`,
      ).toBe(true)
    }
  })

  it('every row in docs/GUIDE-MAP.md corresponds to GUIDE_MAP 1:1', () => {
    const repoRoot = path.resolve(__dirname, '../../../..')
    const guideMapDocPath = path.join(repoRoot, 'docs/GUIDE-MAP.md')
    expect(fs.existsSync(guideMapDocPath), `docs/GUIDE-MAP.md must exist at ${guideMapDocPath}`).toBe(true)

    const guideMapContent = fs.readFileSync(guideMapDocPath, 'utf-8')
    const docIds: string[] = []
    for (const line of guideMapContent.split('\n')) {
      const m = line.match(/^\|\s*`([^`]+)`\s*\|/)
      if (m) docIds.push(m[1])
    }

    const mapIds = GUIDE_MAP.map((e) => e.id)
    expect(docIds).toEqual(mapIds)
  })

  it('safe: true must never end with .send, .stop, .save, .delete, or .toggle', () => {
    const forbiddenSuffixes = ['.send', '.stop', '.save', '.delete', '.toggle']

    for (const entry of GUIDE_MAP) {
      if (entry.safe) {
        for (const suffix of forbiddenSuffixes) {
          expect(
            entry.id.endsWith(suffix),
            `safe: true entry "${entry.id}" must not end with "${suffix}"`,
          ).toBe(false)
        }
      }
    }
  })

  describe('openPage navigation', () => {
    it('reaches every kind of page through the app’s own setters, and a Settings page through its intent', async () => {
      await openPage({ view: 'chat' })
      expect(cockpit.activeView).toBe('chat')

      await openPage({ view: 'settings', rail: 'general' })
      expect(cockpit.activeView).toBe('settings')
      expect(cockpit.settingsIntent).toEqual({ section: 'general' })

      // A head's page is the intent naming the head — Settings opens it.
      await openPage({ view: 'settings', rail: 'main', head: 'coding' })
      expect(cockpit.settingsIntent).toEqual({ section: 'main', head: 'coding' })

      await openPage({ view: 'capability', page: 'mcp' })
      expect(cockpit.activeView).toBe('capability')

      await openPage({ view: 'office' })
      expect(cockpit.activeView).toBe('office')

      await openPage({ view: 'artifacts' })
      expect(cockpit.activeView).toBe('artifacts')
    })

    it('waits for the element after a page change, and gives up in place', async () => {
      const el = document.createElement('div')
      el.setAttribute('data-guide', 'chat.send')
      cockpit.activeView = 'settings'
      setTimeout(() => document.body.appendChild(el), 150)
      await openPage({ view: 'chat' }, '[data-guide="chat.send"]')
      expect(document.body.contains(el)).toBe(true)
      el.remove()
      const t0 = Date.now()
      await openPage({ view: 'chat' }, '[data-guide="chat.send"]')
      expect(Date.now() - t0).toBeLessThan(1500)
    })
  })
})
