import { guideCore } from './core'
import type { GuideRouteId } from './routes'

export type GuideState = {
  stopId: string | null
  route: { id: GuideRouteId; at: number } | null
}

export type GuidePickResult = {
  stopId: string | null
  sentence: string
  route?: { id: GuideRouteId; at: number } | null
}

/**
 * Deterministic query and intent picker.
 * Delegates to GuideCore for unified target, concept, route, and locale resolution.
 */
export function mapPick(query: string, state: GuideState): GuidePickResult {
  const res = guideCore.find(query, state.stopId, state.route)
  return {
    stopId: res.stopId,
    sentence: res.sentence,
    route: res.route !== undefined ? res.route : undefined,
  }
}
