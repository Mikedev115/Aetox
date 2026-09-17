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
import { visibleGuideElement } from './where'
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
  actionType: 'click' | 'focus' | 'input' | 'none'
  confirmationRequired?: boolean
  ref?: string
  synonyms?: string[]
}

/** The map's words for one id in the UI's language — '' for a why the map
 *  does not have, rather than the key t() answers with. A missing why is an
 *  honest blank (docs/GUIDE-MAP.md: nothing is written that DECISIONS.md
 *  does not say), and the guide must show a blank, not a key. */
export type GuideTextField = 'name' | 'what' | 'why' | 'common' | 'recommend'

/** A catalogue entry describes controls that ship with the app. Skill and
 * MCP rows are different: their names and descriptions arrive from the
 * engine (or from the curated shelf), so enumerating them in this file would
 * immediately drift. Those rows carry the same facts on their DOM node.
 * Reading them here makes Shift+click work for every current and future row
 * without creating a second skill/MCP catalogue inside the guide. */
export function guideElement(id: string): HTMLElement | null {
  return visibleGuideElement(id)
}

const DYNAMIC_FIELD: Record<GuideTextField, keyof DOMStringMap> = {
  name: 'guideName',
  what: 'guideWhat',
  why: 'guideWhy',
  common: 'guideCommon',
  recommend: 'guideRecommend',
}

/** Return a static map row or an informational row declared by the visible
 * UI. Dynamic rows are deliberately never pressable: Shift+click asks about
 * a card/chip and must not install, remove, or toggle the item underneath. */
export function guideEntry(id: string): GuideEntry | null {
  const fixed = GUIDE_MAP.find((entry) => entry.id === id)
  if (fixed) return fixed
  const el = guideElement(id)
  const page = el?.dataset.guidePage
  if (!el?.dataset.guideName || !page) return null
  return {
    id,
    page: page as PageId,
    safe: false,
    actionType: 'none',
  }
}

export function guideText(id: string, field: GuideTextField): string {
  const key = `guide.${id}.${field}`
  const v = t(key as any)
  if (v !== key) return v
  return guideElement(id)?.dataset[DYNAMIC_FIELD[field]] ?? ''
}

/** Legacy-compatible array of UI target entries, derived 1:1 from the Guide Catalog. */
export const GUIDE_MAP: GuideEntry[] = CATALOG_TARGETS.map((t) => ({
  id: t.id,
  page: t.page,
  head: t.head,
  safe: t.policy.safe,
  actionType: t.policy.actionType,
  confirmationRequired: t.policy.confirmationRequired,
  ref: t.evidence?.ref,
  synonyms: t.synonyms,
}))
