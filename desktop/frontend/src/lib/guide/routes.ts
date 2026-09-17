// The guide's routes: a walk is a list of map ids, in order. Nothing else.
//
// THIS FILE IS THE TIMELINE, and it is meant to be edited. To change what the
// guide shows and in what order, reorder these strings — no other file needs
// to know. The rules that keep an edit safe:
//
//   1. Every id must exist in map.ts (guideMap.test.ts fails otherwise), and
//      the guide can only stand beside something that is ON SCREEN when it
//      gets there — `page` in the map row is what takes it to the right page,
//      so a stop whose element only exists after a click needs the click's own
//      id before it.
//   2. Stops on one page, then the next page — every page change costs the
//      walk a beat while the page settles (pages.ts waits for the element).
//   3. What the figure SAYS at a stop is not here: it is the map's `what`,
//      from docs/GUIDE-MAP.md through the locales. Change the words there.
//   4. A walk always begins at stop 1. Nothing is remembered between openings
//      — the guide keeps no state at all — so the first stop is the one every
//      person who asks to be shown around will see.

import { CATALOG_ROUTES } from './catalog/data'
import { t } from '../i18n.svelte'

export type GuideRouteId =
  | 'first'
  | 'brain'
  | 'team'
  | 'memory'
  | 'capabilities'
  | 'assistant_desk'
  | 'code_desk'
  | 'workbench_panel'
  | 'code_map'
  | 'connections'
  | 'tools'
  | 'artifacts'
  | 'support'

export type GuideDesk = 'assistant' | 'coding'

/** The two desks share the same guide engine, not the same tours. Keeping the
 * ownership here gives the opening deck, follow-up topics, and live desk
 * switching one answer for whether a route belongs on the current desk. */
const DESK_ROUTES: Record<GuideDesk, ReadonlySet<GuideRouteId>> = {
  assistant: new Set<GuideRouteId>(['assistant_desk', 'workbench_panel']),
  coding: new Set<GuideRouteId>(['code_desk', 'code_map']),
}

const DESK_SPECIFIC_ROUTES = new Set<GuideRouteId>([
  ...DESK_ROUTES.assistant,
  ...DESK_ROUTES.coding,
])

export function routeAllowedForDesk(routeId: GuideRouteId, desk: GuideDesk): boolean {
  return !DESK_SPECIFIC_ROUTES.has(routeId) || DESK_ROUTES[desk].has(routeId)
}

export type GuideRoute = {
  id: GuideRouteId
  name: string
  stops: string[]
}

const DEFAULT_ROUTE_NAMES: Record<string, string> = {
  first: 'รอบแรก',
  brain: 'ต่อสมอง',
  team: 'ตั้งทีมพนักงาน',
  memory: 'ระบบเรียนรู้',
  capabilities: 'ห้องความสามารถ',
  assistant_desk: 'พาไกด์หน้าผู้ช่วย',
  code_desk: 'พาไกด์หน้าโค้ด',
  workbench_panel: 'พาดูแผงเครื่องมือ',
  code_map: 'เปิดแผนที่โค้ด',
  connections: 'เชื่อมโปรแกรมภายนอก',
  tools: 'ตั้งค่าเครื่องมือ',
  artifacts: 'ตามหาผลงานเก่า',
  support: 'สนับสนุน Aetox',
}

export const GUIDE_ROUTES: Record<GuideRouteId, GuideRoute> = Object.fromEntries(
  CATALOG_ROUTES.map((r) => [
    r.id,
    {
      id: r.id as GuideRouteId,
      name: t(r.nameKey as any) || DEFAULT_ROUTE_NAMES[r.id] || r.id,
      stops: r.stops,
    },
  ])
) as Record<GuideRouteId, GuideRoute>

export type GuideRouteChoice = {
  id: GuideRouteId
  name: string
  description: string
  steps: number
}

/** Fixed tours the runtime can execute. Models may choose one; they never
 * build its stops or carry navigation state themselves. */
export function guideRouteChoices(): GuideRouteChoice[] {
  return CATALOG_ROUTES.map((route) => ({
    id: route.id as GuideRouteId,
    name: t(route.nameKey as any) || DEFAULT_ROUTE_NAMES[route.id] || route.id,
    description: t(route.descKey as any) || '',
    steps: route.stops.length,
  }))
}

/** Resolve the user's words to one prepared tour. Specific tours win over
 * “show me around” when both ideas occur in the same request. */
export function routeForIntent(raw: string): GuideRouteId | null {
  const query = raw.trim().toLowerCase()
  if (!query) return null
  if (query in GUIDE_ROUTES) return query as GuideRouteId

  const ordered = CATALOG_ROUTES.slice().sort((a, b) => {
    if (a.id === 'first') return 1
    if (b.id === 'first') return -1
    return 0
  })
  for (const route of ordered) {
    if (route.synonyms?.some((synonym) => query.includes(synonym.toLowerCase()))) {
      return route.id as GuideRouteId
    }
  }
  return null
}
