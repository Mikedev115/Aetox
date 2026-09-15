// The guide's state: one figure, one stop, one sentence, and which brain is
// answering. Everything the figure does on screen (Guide.svelte) is a reaction
// to a field here, so the two doors into a move — a route step and a model's
// `point` — cannot walk the figure twice.
//
// Two brains, one guide (docs/architecture/ui-guide-2026-09-15.md §0): the map
// answers alone until a model session opens, and answers again the moment a
// model turn fails. The person sees one figure that got better, never two.
import { GUIDE_MAP } from './map'
import { GUIDE_ROUTES, type GuideRouteId } from './routes'
import { openPage } from './pages'
import { mapPick } from './mapPick'
import { greetingFor } from './greeting'
import { t } from '../i18n.svelte'
import { cockpit } from '../stores/cockpit.svelte'
import { NewGuideSession, AskGuide, CloseGuideSession } from '../../../wailsjs/go/main/App'

export type TranscriptItem = {
  who: 'user' | 'guide'
  text: string
  stopId?: string
}

// The guide keeps NOTHING between openings — not the conversation, not where
// a walk was left. It used to remember the latter, and that was a mistake with
// a name: pressing "พาเดินดู" resumed at the stop a previous walk ended on, so
// the owner pressed it and landed on the MCP page with no idea why (15 ก.ย.
// 2026). Somebody asking to be shown around means from the beginning; a resume
// worth having is one the guide OFFERS, not one it imposes.

// How long a model turn may take before the map answers instead. One
// question, a few tool calls — but a small local model loads on its first
// turn (qwen3:8b on Ollama: ~10 s before the first token here), and a brain
// that is warming up is not a brain that is gone.
const MODEL_WAIT_MS = 45000

class GuideStore {
  on = $state(false)
  brain = $state<'map' | 'model'>('map')
  stopId = $state<string | null>(null)
  route = $state<{ id: GuideRouteId; at: number } | null>(null)
  transcript = $state<TranscriptItem[]>([])
  offering = $state(false)
  sessionId = $state<string | null>(null)
  /** A question is out to the model. */
  asking = $state(false)
  streamingText = $state('')
  /** What the figure says at the current stop — '' means the map's `what`. */
  sentence = $state('')
  /** Bumped when the figure should walk to stopId and speak `sentence`. */
  moveSeq = $state(0)
  /** Bumped when the figure should speak `sentence` where it stands. */
  saySeq = $state(0)

  /** The figure appears with the one-time offer after the tour. */
  offerTour() {
    this.on = true
    this.offering = true
    this.brain = 'map'
    this.stopId = null
    this.route = null
    this.sentence = ''
  }

  acceptOffer() {
    this.offering = false
    void this.start('first')
  }

  declineOffer() {
    this.offering = false
    void this.stop()
  }

  /** Open the guide: on a route, at one stop, or at the route's first stop. */
  async start(routeId?: GuideRouteId, initialStopId?: string) {
    this.on = true
    this.offering = false
    this.transcript = []
    this.streamingText = ''
    this.asking = false
    this.openModel()
    if (routeId && GUIDE_ROUTES[routeId]) {
      // Always the first stop. See the note on the missing ROUTE_KEY above.
      this.route = { id: routeId, at: 0 }
      await this.goTo(initialStopId ?? GUIDE_ROUTES[routeId].stops[0])
    } else if (initialStopId) {
      this.route = null
      await this.goTo(initialStopId)
    } else {
      // Opened to be asked. It greets by the room the user is standing in and
      // asks what is unclear there — somebody coming to look, not a search box
      // with a face (greeting.ts).
      this.route = null
      this.stopId = null
      this.say(greetingFor(String(cockpit.activeView)))
    }
  }

  /** The model brain, opened beside the map and never waited for: the map
   *  answers the first question while the session is still being built, and
   *  a session that fails to open leaves the map in charge. A model is a
   *  provider the chat already has — the guide has no dial of its own. */
  private openModel() {
    this.brain = 'map'
    this.sessionId = null
    if (!cockpit.model.provider) return
    const index = GUIDE_MAP.map((e) => ({
      id: e.id,
      name: t(`guide.${e.id}.name` as any) || e.id,
      safe: e.safe,
    }))
    NewGuideSession(JSON.stringify(index))
      .then((sid) => {
        if (!this.on || !sid) {
          if (sid) void CloseGuideSession(sid).catch(() => {})
          return
        }
        this.sessionId = sid
        this.brain = 'model'
      })
      .catch(() => {
        this.brain = 'map'
      })
  }

  async stop() {
    this.on = false
    this.offering = false
    this.stopId = null
    this.route = null
    this.transcript = []
    this.streamingText = ''
    this.asking = false
    this.sentence = ''
    const sid = this.sessionId
    this.sessionId = null
    if (sid) {
      try {
        await CloseGuideSession(sid)
      } catch {
        // a turn still running holds the row; the sweep on the next open
        // takes it (engine openDatabase)
      }
    }
  }

  /** Walk to a stop and say `sentence` there ('' = the map's own words). */
  async goTo(id: string, sentence = '') {
    const entry = GUIDE_MAP.find((e) => e.id === id)
    if (!entry) return
    this.stopId = id
    this.sentence = sentence
    if (this.route) {
      const i = GUIDE_ROUTES[this.route.id].stops.indexOf(id)
      if (i >= 0) this.route = { id: this.route.id, at: i }
    }
    await openPage(entry.page, '[data-guide=' + JSON.stringify(id) + ']')
    if (this.stopId !== id) return // somewhere else was asked for meanwhile
    this.moveSeq++
  }

  /** Say something where the figure stands. */
  say(text: string) {
    this.sentence = text
    this.saySeq++
  }

  /** A click on any mapped element while the guide is open ALSO explains it.
   *  The click itself is never taken away — see the listener in Guide.svelte
   *  for why holding the app still was the wrong trade. */
  explain(id: string) {
    void this.goTo(id)
  }

  onChunk(text: string, replace: boolean) {
    this.streamingText = replace ? text : this.streamingText + text
  }

  /** One question, answered by whichever brain is there. The model's answer
   *  is spoken where the figure stands — its own `point` has already walked
   *  it; the map's answer walks the figure itself. */
  async ask(question: string): Promise<string> {
    const q = question.trim()
    if (!q) return ''
    this.transcript.push({ who: 'user', text: q, stopId: this.stopId ?? undefined })
    if (this.brain === 'model' && this.sessionId) {
      const sid = this.sessionId
      this.asking = true
      this.streamingText = ''
      try {
        const answer = await Promise.race([
          AskGuide(sid, q),
          new Promise<never>((_, reject) => setTimeout(() => reject(new Error('timeout')), MODEL_WAIT_MS)),
        ])
        this.asking = false
        if (!this.on) return ''
        const text = (answer || '').trim()
        if (text) {
          this.transcript.push({ who: 'guide', text, stopId: this.stopId ?? undefined })
          this.say(text)
          return text
        }
      } catch {
        // the map takes over for this question and the rest of this opening
        this.brain = 'map'
      }
      this.asking = false
      this.streamingText = ''
    }
    const result = mapPick(q, { stopId: this.stopId, route: this.route })
    if (result.route !== undefined) this.route = result.route
    if (result.stopId) {
      await this.goTo(result.stopId, result.sentence)
    } else {
      this.say(result.sentence)
    }
    this.transcript.push({ who: 'guide', text: result.sentence, stopId: result.stopId ?? undefined })
    return result.sentence
  }

  next() {
    if (!this.route) return
    const r = GUIDE_ROUTES[this.route.id]
    const at = (this.route.at + 1) % r.stops.length
    this.route = { id: this.route.id, at }
    void this.goTo(r.stops[at])
  }

  prev() {
    if (!this.route) return
    const r = GUIDE_ROUTES[this.route.id]
    const at = (this.route.at - 1 + r.stops.length) % r.stops.length
    this.route = { id: this.route.id, at }
    void this.goTo(r.stops[at])
  }

  /** Press an element for the user. The map's `safe` flag is the gate, and it
   *  is checked here — the one place that clicks — for both the bubble's button
   *  and the model's `press` action. */
  press(id: string | null = this.stopId): { ok: boolean; message: string } {
    if (!id) return { ok: false, message: '' }
    const entry = GUIDE_MAP.find((e) => e.id === id)
    if (!entry || !entry.safe) {
      return { ok: false, message: t('guide.safeRefusal') }
    }
    const el = document.querySelector<HTMLElement>('[data-guide=' + JSON.stringify(id) + ']')
    if (!el) return { ok: false, message: t('guide.notOnScreen') }
    // After the click that asked for it has finished: a menu the press opens
    // would otherwise be closed by that same click landing "outside" it. And
    // marked as the guide's own, so the click-to-explain listener lets it
    // through to the button (Guide.svelte).
    setTimeout(() => {
      this.pressing = true
      try {
        el.click()
      } finally {
        this.pressing = false
      }
    }, 0)
    return { ok: true, message: t('guide.pressed') }
  }

  /** True while the guide itself is clicking a button for the user. */
  pressing = false
}

export const guide = new GuideStore()
