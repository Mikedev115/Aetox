import { AnswerGuide } from '../../../wailsjs/go/main/App'
import { GUIDE_MAP, guideText, type GuidePage } from './map'
import { openPage } from './pages'
import { guide } from './guideState.svelte'
import { cockpit } from '../stores/cockpit.svelte'
import { t } from '../i18n.svelte'

export function getVisibleGuideElements(): { id: string; name: string; safe: boolean }[] {
  if (typeof document === 'undefined') return []
  const nodes = document.querySelectorAll<HTMLElement>('[data-guide]')
  const out: { id: string; name: string; safe: boolean }[] = []
  const seen = new Set<string>()
  for (const el of nodes) {
    const id = el.getAttribute('data-guide')
    if (!id || seen.has(id)) continue
    seen.add(id)
    const rect = el.getBoundingClientRect()
    if (rect.width > 0 && rect.height > 0) {
      const entry = GUIDE_MAP.find((e) => e.id === id)
      out.push({
        id,
        name: t(`guide.${id}.name` as any) || id,
        safe: entry?.safe ?? false,
      })
    }
  }
  return out
}

/** The page a `goto` names: chat · settings.<rail> · capability.<page> ·
 *  office · artifacts — or a map id, whose own page is meant. */
export function pageFrom(arg: string): GuidePage | null {
  const s = arg.trim()
  if (!s) return null
  const byId = GUIDE_MAP.find((e) => e.id === s)
  if (byId) return byId.page
  if (s === 'chat') return { view: 'chat' }
  if (s === 'office') return { view: 'office' }
  if (s === 'artifacts') return { view: 'artifacts' }
  const m = /^(settings|capability)\.([a-z_]+)$/.exec(s)
  if (m && m[1] === 'settings') return { view: 'settings', rail: m[2] }
  if (m && m[1] === 'capability') return { view: 'capability', page: m[2] }
  return null
}

export async function handleGuideAsk(ask: { id: string; action: string; args?: Record<string, any> }) {
  const { id, action, args = {} } = ask
  try {
    switch (action.toLowerCase().trim()) {
      case 'where': {
        const visible = getVisibleGuideElements()
        const res = {
          page: String(cockpit.activeView),
          guideAt: guide.stopId ?? '',
          visible,
        }
        await AnswerGuide(id, JSON.stringify(res))
        break
      }
      case 'describe': {
        const targetId = String(args.id || '')
        const entry = GUIDE_MAP.find((e) => e.id === targetId)
        const res = {
          id: targetId,
          name: guideText(targetId, 'name') || targetId,
          what: guideText(targetId, 'what'),
          why: guideText(targetId, 'why'),
          ref: entry?.ref || '',
          safe: entry?.safe ?? false,
          page: entry?.page ? JSON.stringify(entry.page) : '',
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
        await AnswerGuide(id, JSON.stringify({ ok: true, page: String(cockpit.activeView) }))
        break
      }
      case 'goto': {
        const page = pageFrom(String(args.page || ''))
        if (!page) {
          await AnswerGuide(id, JSON.stringify({ ok: false, error: 'unknown page: ' + String(args.page || '') }))
          break
        }
        await openPage(page)
        await AnswerGuide(id, JSON.stringify({ ok: true, page: String(cockpit.activeView) }))
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
