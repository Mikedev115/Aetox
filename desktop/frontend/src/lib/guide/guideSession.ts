import { AnswerGuide } from '../../../wailsjs/go/main/App'
import { GUIDE_MAP, guideText, type GuidePage } from './map'
import { isPageId } from '../rooms'
import { openPage } from './pages'
import { currentPage } from './where'
import { guide } from './guideState.svelte'
import { cockpit } from '../stores/cockpit.svelte'
import { t } from '../i18n.svelte'

import { guideCore } from './core'

export function getVisibleGuideElements(): { id: string; name: string; safe: boolean }[] {
  return guideCore.where().visible
}

/** The place a `goto` names — a PageId, or the id of anything on the map,
 *  whose own place is meant. Checked, because the model supplies it. */
export function pageFrom(arg: string): GuidePage | null {
  const s = arg.trim()
  if (!s) return null
  if (isPageId(s)) return s
  return GUIDE_MAP.find((e) => e.id === s)?.page ?? null
}

export async function handleGuideAsk(ask: { id: string; action: string; args?: Record<string, any> }) {
  const { id, action, args = {} } = ask
  try {
    switch (action.toLowerCase().trim()) {
      case 'where': {
        const { page, visible } = guideCore.where()
        const res = {
          page: page ?? currentPage() ?? String(cockpit.activeView),
          guideAt: guide.stopId ?? '',
          visible,
        }
        await AnswerGuide(id, JSON.stringify(res))
        break
      }
      case 'describe': {
        const targetId = String(args.id || '')
        const fact = guideCore.describe(targetId)
        const entry = GUIDE_MAP.find((e) => e.id === targetId)
        const res = {
          id: targetId,
          name: fact?.name || guideText(targetId, 'name') || targetId,
          what: fact?.what || guideText(targetId, 'what'),
          why: fact?.why || guideText(targetId, 'why'),
          ref: fact?.ref || entry?.ref || '',
          safe: fact?.safe ?? entry?.safe ?? false,
          page: entry?.page ?? '',
        }
        await AnswerGuide(id, JSON.stringify(res))
        break
      }
      case 'point': {
        const targetId = String(args.id || '')
        if (!GUIDE_MAP.some((e) => e.id === targetId)) {
          await AnswerGuide(id, JSON.stringify({ ok: false, error: 'unknown id: ' + targetId }))
          break
        }
        await guide.goTo(targetId)
        await AnswerGuide(id, JSON.stringify({ ok: true, page: currentPage() ?? String(cockpit.activeView) }))
        break
      }
      case 'goto': {
        const page = pageFrom(String(args.page || ''))
        if (!page) {
          await AnswerGuide(id, JSON.stringify({ ok: false, error: 'unknown page: ' + String(args.page || '') }))
          break
        }
        await openPage(page)
        await AnswerGuide(id, JSON.stringify({ ok: true, page: currentPage() ?? String(cockpit.activeView) }))
        break
      }
      case 'press': {
        // The map's safe flag is checked in guide.press — the one place that
        // clicks — so the window, not the model's prompt, is what refuses.
        const res = guide.press(String(args.id || ''))
        await AnswerGuide(id, JSON.stringify(res.ok ? { ok: true } : { ok: false, error: res.message || 'not safe' }))
        break
      }
      default:
        await AnswerGuide(id, JSON.stringify({ ok: false, error: 'unknown action: ' + action }))
    }
  } catch (err: any) {
    await AnswerGuide(id, JSON.stringify({ ok: false, error: err?.message || String(err) }))
  }
}
