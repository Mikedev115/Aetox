import { GUIDE_MAP, type GuideEntry } from './map'
import { GUIDE_ROUTES, type GuideRouteId } from './routes'
import { openPage } from './pages'
import { mapPick } from './mapPick'
import { t } from '../i18n.svelte'

export type TranscriptItem = {
  who: 'user' | 'guide'
  text: string
  stopId?: string
}

class GuideStore {
  on = $state(false)
  brain = $state<'map' | 'model'>('map')
  stopId = $state<string | null>(null)
  route = $state<{ id: GuideRouteId; at: number } | null>(null)
  transcript = $state<TranscriptItem[]>([])
  speaking = $state(false)
  offering = $state(false)

  offerTour() {
    this.on = true
    this.offering = true
    this.stopId = 'sidebar.projects'
    this.goTo('sidebar.projects')
  }

  acceptOffer() {
    this.offering = false
    this.start('first')
  }

  declineOffer() {
    this.offering = false
    this.stop()
  }

  start(routeId?: GuideRouteId, initialStopId?: string) {
    this.on = true
    this.offering = false
    this.transcript = []
    if (routeId && GUIDE_ROUTES[routeId]) {
      this.route = { id: routeId, at: 0 }
      const firstStop = initialStopId ?? GUIDE_ROUTES[routeId].stops[0]
      this.goTo(firstStop)
    } else if (initialStopId) {
      this.route = null
      this.goTo(initialStopId)
    } else {
      this.route = null
      this.stopId = 'sidebar.projects'
      this.goTo('sidebar.projects')
    }
  }

  stop() {
    this.on = false
    this.offering = false
    this.stopId = null
    this.route = null
    this.transcript = []
  }

  async goTo(id: string) {
    this.stopId = id
    const entry = GUIDE_MAP.find((e) => e.id === id)
    if (entry) {
      await openPage(entry.page)
    }
  }

  async ask(question: string): Promise<string> {
    const q = question.trim()
    if (!q) return ''
    this.transcript.push({ who: 'user', text: q, stopId: this.stopId ?? undefined })

    const result = mapPick(q, { stopId: this.stopId, route: this.route })
    if (result.route !== undefined) {
      this.route = result.route
    }
    if (result.stopId) {
      await this.goTo(result.stopId)
    }

    this.transcript.push({ who: 'guide', text: result.sentence, stopId: result.stopId ?? undefined })
    return result.sentence
  }

  next() {
    if (!this.route) return
    const r = GUIDE_ROUTES[this.route.id]
    const nextAt = (this.route.at + 1) % r.stops.length
    this.route = { id: this.route.id, at: nextAt }
    const nextStop = r.stops[nextAt]
    this.goTo(nextStop)
  }

  prev() {
    if (!this.route) return
    const r = GUIDE_ROUTES[this.route.id]
    const prevAt = (this.route.at - 1 + r.stops.length) % r.stops.length
    this.route = { id: this.route.id, at: prevAt }
    const prevStop = r.stops[prevAt]
    this.goTo(prevStop)
  }

  press(): { ok: boolean; message: string } {
    if (!this.stopId) return { ok: false, message: '' }
    const entry = GUIDE_MAP.find((e) => e.id === this.stopId)
    if (!entry || !entry.safe) {
      return { ok: false, message: t('guide.safeRefusal') }
    }
    const el = document.querySelector<HTMLElement>('[data-guide=' + JSON.stringify(this.stopId) + ']')
    if (el) {
      el.click()
      return { ok: true, message: t('guide.pressed') }
    }
    return { ok: false, message: '' }
  }
}

export const guide = new GuideStore()
