// Deterministic Core for the Guide Runtime:
// Handles where, find, describe, route, press, and targeted fact pack retrieval.
// Guarantees 100% functionality without a model, and supplies minimal, high-signal
// fact packs when delegating to the model adapter.

import { GUIDE_CATALOG, CATALOG_TARGETS } from './catalog/data'
import type { CatalogEntry, TargetCatalogEntry, ConceptCatalogEntry, RouteCatalogEntry } from './catalog/types'
import { GUIDE_MAP, guideElement, guideEntry, guideText } from './map'
import { GUIDE_ROUTES, routeForIntent, type GuideRouteId } from './routes'
import { currentPage, onScreen } from './where'
import { nextStepTo, stepsTo, checkPrecondition, resolveNavigation, type NavigationResolution } from './path'
import { t } from '../i18n.svelte'
import { th } from '../locales/th'
import { en } from '../locales/en'
import { zh } from '../locales/zh'

export type FactItem = {
  id: string
  kind: 'target' | 'concept' | 'route'
  name: string
  what: string
  why: string
  ref?: string
  safe?: boolean
}

export type FactPack = {
  page: string | null
  guideAt: string | null
  visibleTargets: Array<{ id: string; name: string; safe: boolean }>
  retrievedFacts: FactItem[]
}

export type FindResult = {
  stopId: string | null
  sentence: string
  route?: { id: GuideRouteId; at: number } | null
  isExact: boolean
  confidence: number
  conceptId?: string
}

function cleanQuery(raw: string): string {
  let s = raw.trim().toLowerCase()
  // English filler removals
  s = s.replace(/\b(where is the|where is|where's the|where's|what is the|what is|how to find the|how to find|show me the|show me|find the|find)\b/gi, ' ')
  s = s.replace(/\b(button|page|tab|menu|icon|section|settings|the)\b/gi, ' ')
  // Thai filler removals
  s = s.replace(/(อยู่ที่ไหน|อยู่ไหน|คืออะไร|ช่วยหา|เปิดดู|พาไปดู|พาไปที่|พาไป|หา)/g, ' ')
  s = s.replace(/(ปุ่ม|หน้า|เมนู|แท็บ|ไอคอน)/g, ' ')
  // Chinese filler removals
  s = s.replace(/(在哪里|在哪|是什么|怎么找|帮我找|带我去看|带我找|查看|打开|寻找)/g, ' ')
  s = s.replace(/(按钮|页面|菜单|标签|图标)/g, ' ')
  return s.trim()
}

export class GuideCore {
  /**
   * Inspection: read DOM state, current page, and currently visible targets.
   */
  where(): { page: string | null; visible: Array<{ id: string; name: string; safe: boolean }> } {
    const page = currentPage()
    if (typeof document === 'undefined') return { page, visible: [] }

    const nodes = document.querySelectorAll<HTMLElement>('[data-guide]')
    const visible: Array<{ id: string; name: string; safe: boolean }> = []
    const seen = new Set<string>()

    for (const el of nodes) {
      const id = el.getAttribute('data-guide')
      if (!id || seen.has(id)) continue
      seen.add(id)
      const r = el.getBoundingClientRect()
      if (r.width > 0 && r.height > 0) {
        const entry = CATALOG_TARGETS.find((e) => e.id === id)
        const dynamic = guideEntry(id)
        visible.push({
          id,
          name: guideText(id, 'name') || id,
          safe: entry?.policy.safe ?? dynamic?.safe ?? false,
        })
      }
    }

    return { page, visible }
  }

  /**
   * Describe a target, concept, or route using exact grounded facts and evidence.
   */
  describe(id: string): FactItem | null {
    const entry = GUIDE_CATALOG.find((e) => e.id === id)
    if (!entry) {
      const dynamic = guideEntry(id)
      if (!dynamic) return null
      return {
        id,
        kind: 'target',
        name: guideText(id, 'name') || id,
        what: guideText(id, 'what'),
        why: guideText(id, 'why'),
        safe: dynamic.safe,
      }
    }

    let name = ''
    let what = ''
    let why = ''

    if (entry.kind === 'route') {
      name = t(entry.nameKey as any) || entry.id
      what = t(entry.descKey as any) || ''
      why = entry.evidence?.ref ? `Spec reference: ${entry.evidence.ref}` : ''
    } else {
      name = guideText(id, 'name') || id
      what = guideText(id, 'what')
      why = guideText(id, 'why')
    }

    return {
      id,
      kind: entry.kind,
      name,
      what,
      why,
      ref: entry.evidence?.ref,
      safe: entry.kind === 'target' ? entry.policy.safe : false,
    }
  }

  /**
   * Determine navigation steps, availability, preconditions, and next button for any target.
   */
  route(targetId: string): NavigationResolution {
    return resolveNavigation(targetId)
  }

  /**
   * Deterministic search and intent matching against catalog targets and concepts.
   */
  find(query: string, currentStopId?: string | null, activeRoute?: { id: GuideRouteId; at: number } | null): FindResult {
    const q = query.trim()
    if (!q) {
      if (currentStopId) {
        const item = this.describe(currentStopId)
        if (item) {
          return {
            stopId: currentStopId,
            sentence: `${item.name}: ${item.what}`,
            route: activeRoute,
            isExact: true,
            confidence: 100,
          }
        }
      }
      return {
        stopId: 'settings.rail.brain',
        sentence: t('guide.unknownWithBrainHint'),
        route: null,
        isExact: false,
        confidence: 0,
      }
    }

    const qLower = q.toLowerCase()

    // 1. Prepared route commands. The route resolver owns the vocabulary;
    // neither this core nor a model gets a second copy of the tour map.
    const routeId = routeForIntent(qLower)
    if (routeId) {
      const route = GUIDE_ROUTES[routeId]
      const stopId = route.stops[0]
      return {
        stopId,
        route: { id: routeId, at: 0 },
        sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
        isExact: true,
        confidence: 100,
      }
    }

    // 2. Step forward / back in active route
    if (activeRoute && /(^|\s)(ต่อไป|ถัดไป|next|下一步)(\s|$)/i.test(qLower)) {
      const r = GUIDE_ROUTES[activeRoute.id]
      const nextAt = (activeRoute.at + 1) % r.stops.length
      const stopId = r.stops[nextAt]
      return {
        stopId,
        route: { id: activeRoute.id, at: nextAt },
        sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
        isExact: true,
        confidence: 100,
      }
    }
    if (activeRoute && /(^|\s)(ก่อนหน้า|ย้อนกลับ|prev|previous|back|上一步)(\s|$)/i.test(qLower)) {
      const r = GUIDE_ROUTES[activeRoute.id]
      const prevAt = (activeRoute.at - 1 + r.stops.length) % r.stops.length
      const stopId = r.stops[prevAt]
      return {
        stopId,
        route: { id: activeRoute.id, at: prevAt },
        sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
        isExact: true,
        confidence: 100,
      }
    }

    // 3. "What is this" query
    if (/^(นี่คืออะไร|ปุ่มนี้คืออะไร|ตรงนี้คืออะไร|อันนี้คืออะไร|what is this|what is here|what's this|这是什么|当前按钮是什么)\??$/i.test(qLower)) {
      if (currentStopId) {
        const item = this.describe(currentStopId)
        if (item) {
          return {
            stopId: currentStopId,
            sentence: `${item.name}: ${item.what}`,
            route: activeRoute,
            isExact: true,
            confidence: 100,
          }
        }
      }
    }

    // 4. Keyword search across catalog (targets and concepts) in th, en, zh
    const kw = cleanQuery(q) || qLower
    let bestEntry: CatalogEntry | null = null
    let bestScore = 0

    const localeDicts = [th, en, zh]

    for (const entry of GUIDE_CATALOG) {
      let score = 0

      // Synonyms matching
      if (entry.synonyms) {
        for (const syn of entry.synonyms) {
          const sLower = syn.toLowerCase()
          if (sLower === kw || sLower === qLower) {
            score = Math.max(score, 100 + sLower.length)
          } else if (kw && (sLower.includes(kw) || kw.includes(sLower))) {
            score = Math.max(score, 40 + Math.min(sLower.length, kw.length))
          } else if (qLower.includes(sLower) || sLower.includes(qLower)) {
            score = Math.max(score, 30 + Math.min(sLower.length, qLower.length))
          }
        }
      }

      // Localized names in th, en, zh
      const nameKey = entry.kind === 'route' ? entry.nameKey : `guide.${entry.id}.name`
      for (const dict of localeDicts) {
        const localized = ((dict as Record<string, string>)[nameKey] ?? '').toLowerCase()
        if (!localized) continue
        if (localized === kw || localized === qLower) {
          score = Math.max(score, 100 + localized.length)
        } else if (kw && (localized.includes(kw) || kw.includes(localized))) {
          score = Math.max(score, 50 + Math.min(localized.length, kw.length))
        } else if (qLower.includes(localized) || localized.includes(qLower)) {
          score = Math.max(score, 35 + Math.min(localized.length, qLower.length))
        }
      }

      // ID parts matching
      const idParts = entry.id.toLowerCase().split('.')
      for (const part of idParts) {
        if (part === kw) {
          score = Math.max(score, 45)
        } else if (kw.length > 2 && part.includes(kw)) {
          score = Math.max(score, 25)
        }
      }

      if (score > bestScore || (score === bestScore && (entry.kind === 'concept' || entry.kind === 'route'))) {
        bestScore = score
        bestEntry = entry
      }
    }

    if (bestEntry && bestScore >= 20) {
      if (bestEntry.kind === 'route') {
        const routeId = bestEntry.id as GuideRouteId
        const stopId = bestEntry.stops[0]
        const name = t(bestEntry.nameKey as any) || bestEntry.id
        const what = t(bestEntry.descKey as any) || ''
        return {
          stopId,
          route: { id: routeId, at: 0 },
          sentence: `${name} — ${what}`,
          isExact: bestScore >= 50,
          confidence: bestScore,
        }
      }

      const name = t(`guide.${bestEntry.id}.name` as any)
      const what = t(`guide.${bestEntry.id}.what` as any)

      if (bestEntry.kind === 'concept') {
        const targetId = bestEntry.relatedTargets[0] || null
        return {
          stopId: targetId,
          conceptId: bestEntry.id,
          sentence: `${name} — ${what}`,
          route: null,
          isExact: bestScore >= 50,
          confidence: bestScore,
        }
      }

      return {
        stopId: bestEntry.id,
        sentence: `${name} — ${what}`,
        route: null,
        isExact: bestScore >= 50,
        confidence: bestScore,
      }
    }

    return {
      stopId: 'settings.rail.brain',
      sentence: t('guide.unknownWithBrainHint'),
      route: null,
      isExact: false,
      confidence: 0,
    }
  }

  /**
   * Build targeted, high-precision fact pack for model context (avoiding 157-item dump).
   */
  buildFactPack(query: string, currentStopId?: string | null): FactPack {
    const { page, visible } = this.where()
    const retrievedFacts: FactItem[] = []
    const seen = new Set<string>()

    // Current stop fact
    if (currentStopId) {
      const f = this.describe(currentStopId)
      if (f) {
        retrievedFacts.push(f)
        seen.add(f.id)
      }
    }

    // Top match fact
    const found = this.find(query, currentStopId)
    if (found.stopId && !seen.has(found.stopId)) {
      const f = this.describe(found.stopId)
      if (f) {
        retrievedFacts.push(f)
        seen.add(f.id)
      }
    }
    if (found.conceptId && !seen.has(found.conceptId)) {
      const c = this.describe(found.conceptId)
      if (c) {
        retrievedFacts.push(c)
        seen.add(c.id)
      }
    }

    return {
      page,
      guideAt: currentStopId || null,
      visibleTargets: visible.slice(0, 10), // only up to 10 visible items
      retrievedFacts: retrievedFacts.slice(0, 3), // top 3 facts
    }
  }

  /**
   * Safe action execution: strictly enforces action policy and semantic conditions.
   */
  press(id: string): { ok: boolean; message: string } {
    if (!id) return { ok: false, message: '' }

    const entry = CATALOG_TARGETS.find((e) => e.id === id)
    if (!entry || !entry.policy.safe) {
      return { ok: false, message: t('guide.safeRefusal') }
    }

    const pre = checkPrecondition(id)
    if (!pre.ok) {
      return { ok: false, message: pre.reason || t('guide.notOnScreen') }
    }

    if (typeof document === 'undefined') return { ok: false, message: '' }
    const el = guideElement(id)
    if (!el) return { ok: false, message: t('guide.notOnScreen') }

    setTimeout(() => {
      try {
        el.click()
      } catch {
        // click failed safely
      }
    }, 0)

    return { ok: true, message: t('guide.pressed') }
  }
}

export const guideCore = new GuideCore()
