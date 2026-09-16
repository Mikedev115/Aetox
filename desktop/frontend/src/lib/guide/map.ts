// The map of the app's own UI: every data-guide id, which page it lives on,
// whether the guide may press it, and which DECISIONS.md section justifies its
// design.
//
// Powered by the Guide Catalog (./catalog/data.ts) as the single source of truth,
// connecting UI targets, system-level concepts, evidence, action policies,
// and navigation requirements.
//
// Source of truth for words is docs/GUIDE-MAP.md, rendered through locales
// (lib/locales/{th,en,zh}.ts under guide.<id>.name, guide.<id>.what, guide.<id>.why).

import { t } from '../i18n.svelte'
import type { PageId } from '../rooms'
import { CATALOG_TARGETS, GUIDE_CATALOG } from './catalog/data'
export { GUIDE_CATALOG, CATALOG_TARGETS }
export type { CatalogEntry, TargetCatalogEntry, ConceptCatalogEntry } from './catalog/types'

/** Where a thing lives, in the app's one vocabulary for places (lib/rooms.ts)
 *  — the same string the page stamps on itself and that guide/where.ts reads
 *  back. One name, four readers, no second list to drift. */
export type GuidePage = PageId

export type GuideEntry = {
  id: string
  page: GuidePage
  /** For a head's own page: which head's editor to open on arrival. */
  head?: 'assistant' | 'coding'
  safe: boolean
  ref?: string
  synonyms?: string[]
}

/** The map's words for one id in the UI's language — '' for a why the map
 *  does not have, rather than the key t() answers with. A missing why is an
 *  honest blank (docs/GUIDE-MAP.md: nothing is written that DECISIONS.md
 *  does not say), and the guide must show a blank, not a key. */
export function guideText(id: string, field: 'name' | 'what' | 'why'): string {
  const key = `guide.${id}.${field}`
  const v = t(key as any)
  return v === key ? '' : v
}

/** Legacy-compatible array of UI target entries, derived 1:1 from the Guide Catalog. */
export const GUIDE_MAP: GuideEntry[] = CATALOG_TARGETS.map((t) => ({
  id: t.id,
  page: t.page,
  head: t.head,
  safe: t.policy.safe,
  ref: t.evidence?.ref,
  synonyms: t.synonyms,
}))
