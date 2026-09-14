import { GUIDE_MAP, type GuideEntry } from './map'
import { GUIDE_ROUTES, type GuideRouteId } from './routes'
import { t } from '../i18n.svelte'
import { th } from '../locales/th'
import { en } from '../locales/en'

export type GuideState = {
  stopId: string | null
  route: { id: GuideRouteId; at: number } | null
}

export type GuidePickResult = {
  stopId: string | null
  sentence: string
  route?: { id: GuideRouteId; at: number } | null
}

/** Strip common prefixes and filler words so keyword search focuses on intent. */
function cleanQuery(raw: string): string {
  let s = raw.trim().toLowerCase()
  // English filler removals
  s = s.replace(/\b(where is the|where is|where's the|where's|what is the|what is|how to find the|how to find|show me the|show me|find the|find)\b/gi, ' ')
  s = s.replace(/\b(button|page|tab|menu|icon|section|settings|the)\b/gi, ' ')
  // Thai filler removals
  s = s.replace(/(อยู่ที่ไหน|อยู่ไหน|คืออะไร|ช่วยหา|เปิดดู|พาไปดู|พาไปที่|พาไป|หา)/g, ' ')
  s = s.replace(/(ปุ่ม|หน้า|เมนู|แท็บ|ไอคอน)/g, ' ')
  return s.trim()
}

export function mapPick(query: string, state: GuideState): GuidePickResult {
  const q = query.trim()
  if (!q) {
    if (state.stopId) {
      const name = t(`guide.${state.stopId}.name` as any)
      const what = t(`guide.${state.stopId}.what` as any)
      return { stopId: state.stopId, sentence: `${name}: ${what}`, route: state.route }
    }
    return { stopId: 'settings.rail.brain', sentence: t('guide.unknownWithBrainHint'), route: null }
  }

  const qLower = q.toLowerCase()

  // 1. Route commands
  if (/(^|\s)(รอบแรก|พาดู|ทัวร์|first|tour|walk)(\s|$)/i.test(qLower)) {
    const route = GUIDE_ROUTES.first
    const stopId = route.stops[0]
    return {
      stopId,
      route: { id: 'first', at: 0 },
      sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
    }
  }
  if (/(^|\s)(ต่อสมอง|ใส่คีย์|brain|api key|provider)(\s|$)/i.test(qLower)) {
    const route = GUIDE_ROUTES.brain
    const stopId = route.stops[0]
    return {
      stopId,
      route: { id: 'brain', at: 0 },
      sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
    }
  }
  if (/(^|\s)(ตั้งทีม|พนักงาน|team|staff)(\s|$)/i.test(qLower)) {
    const route = GUIDE_ROUTES.team
    const stopId = route.stops[0]
    return {
      stopId,
      route: { id: 'team', at: 0 },
      sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
    }
  }

  // 2. Step forward / back in active route
  if (state.route && /(^|\s)(ต่อไป|ถัดไป|next)(\s|$)/i.test(qLower)) {
    const r = GUIDE_ROUTES[state.route.id]
    const nextAt = (state.route.at + 1) % r.stops.length
    const stopId = r.stops[nextAt]
    return {
      stopId,
      route: { id: state.route.id, at: nextAt },
      sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
    }
  }
  if (state.route && /(^|\s)(ก่อนหน้า|ย้อนกลับ|prev|previous|back)(\s|$)/i.test(qLower)) {
    const r = GUIDE_ROUTES[state.route.id]
    const prevAt = (state.route.at - 1 + r.stops.length) % r.stops.length
    const stopId = r.stops[prevAt]
    return {
      stopId,
      route: { id: state.route.id, at: prevAt },
      sentence: `${t(`guide.${stopId}.name` as any)} — ${t(`guide.${stopId}.what` as any)}`,
    }
  }

  // 3. Explain current item if query asks "what is this" without naming a feature
  if (/^(นี่คืออะไร|ปุ่มนี้คืออะไร|ตรงนี้คืออะไร|อันนี้คืออะไร|what is this|what is here|what's this)\??$/i.test(qLower)) {
    if (state.stopId) {
      const name = t(`guide.${state.stopId}.name` as any)
      const what = t(`guide.${state.stopId}.what` as any)
      return { stopId: state.stopId, sentence: `${name}: ${what}`, route: state.route }
    }
  }

  // 4. Keyword search across GUIDE_MAP entries
  const keyword = cleanQuery(q)
  const kw = keyword || qLower

  let bestEntry: GuideEntry | null = null
  let bestScore = 0

  for (const entry of GUIDE_MAP) {
    let score = 0

    // Compare with synonyms
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

    // Compare with localized names (th and en)
    const nameKey = `guide.${entry.id}.name` as keyof typeof th
    const thName = (th[nameKey] ?? '').toLowerCase()
    const enName = (en[nameKey] ?? '').toLowerCase()

    for (const n of [thName, enName]) {
      if (!n) continue
      if (n === kw || n === qLower) {
        score = Math.max(score, 100 + n.length)
      } else if (kw && (n.includes(kw) || kw.includes(n))) {
        score = Math.max(score, 50 + Math.min(n.length, kw.length))
      } else if (qLower.includes(n) || n.includes(qLower)) {
        score = Math.max(score, 35 + Math.min(n.length, qLower.length))
      }
    }

    // Compare with ID parts
    const idParts = entry.id.toLowerCase().split('.')
    for (const part of idParts) {
      if (part === kw) {
        score = Math.max(score, 45)
      } else if (kw && kw.length > 2 && part.includes(kw)) {
        score = Math.max(score, 25)
      }
    }

    if (score > bestScore) {
      bestScore = score
      bestEntry = entry
    }
  }

  if (bestEntry && bestScore >= 20) {
    const name = t(`guide.${bestEntry.id}.name` as any)
    const what = t(`guide.${bestEntry.id}.what` as any)
    return {
      stopId: bestEntry.id,
      sentence: `${name} — ${what}`,
      route: null,
    }
  }

  // 5. Fallback: point to brain setting with recommendation hint
  return {
    stopId: 'settings.rail.brain',
    sentence: t('guide.unknownWithBrainHint'),
    route: null,
  }
}
