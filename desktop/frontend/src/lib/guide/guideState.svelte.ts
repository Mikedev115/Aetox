// The guide's state: one figure, one stop, one sentence, and which brain is
// answering. Everything the figure does on screen (Guide.svelte) is a reaction
// to a field here, so the two doors into a move — a route step and a model's
// `point` — cannot walk the figure twice.
//
// Two brains, one guide (docs/architecture/ui-guide-2026-09-15.md §0): the map
// answers alone until a model session opens, and answers again the moment a
// model turn fails. The person sees one figure that got better, never two.
import { GUIDE_MAP, guideElement, guideEntry, guideText } from './map'
import { GUIDE_ROUTES, routeAllowedForDesk, routeForIntent, type GuideRouteId } from './routes'
import { mapPick } from './mapPick'
import { arrivalFor, greetingFor } from './greeting'
import { roomOf, type PageId } from '../rooms'
import { currentPage, onScreen } from './where'
import { presetAnswer, type GuidePresetAnswer } from './presetAnswers'
import { guidePrefs } from './guidePrefs.svelte'
import { resolveNavigation, resolvePageNavigation, waitForNavigationProgress } from './path'
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

/** Small local models occasionally echo the same short answer twice. Keep the
 * longer copy when the second half is an exact or truncated repeat. */
export function compactGuideReply(raw: string): string {
  const text = raw.trim()
  if (text.length < 64) return text
  for (let at = 32; at <= text.length - 32; at++) {
    const left = text.slice(0, at).trim()
    const right = text.slice(at).trim()
    const shorter = Math.min(left.length, right.length)
    const longer = Math.max(left.length, right.length)
    if (shorter / longer < 0.65) continue
    if (left === right || left.startsWith(right)) return left
    if (right.startsWith(left)) return right
  }
  return text
}

/** A small model may return its tour choice as plain JSON instead of making
 * the guide tool call. Treat that as a routing decision, never as copy to
 * print in the chat. The program still owns every stop in the chosen route. */
export function modelTourChoice(raw: string): { recognized: boolean; routeId: GuideRouteId | null } {
  const text = raw.trim().replace(/^```(?:json)?\s*/i, '').replace(/\s*```$/, '')
  if (!text.startsWith('{') || !text.endsWith('}')) return { recognized: false, routeId: null }
  try {
    const value = JSON.parse(text) as Record<string, unknown>
    const requested = typeof value.tour === 'string'
      ? value.tour
      : typeof value.route === 'string'
        ? value.route
        : ''
    if (!requested) return { recognized: false, routeId: null }
    return { recognized: true, routeId: routeForIntent(requested) }
  } catch {
    return { recognized: false, routeId: null }
  }
}

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
  /** A manual first open starts as a polite introduction, not a page dump.
   * The person can explicitly ask for the full page explanation from the
   * first prepared action shown under the greeting. */
  welcoming = $state(false)
  /** Bumped when the figure should walk to stopId and speak `sentence`. */
  moveSeq = $state(0)
  /** Bumped when the figure should speak `sentence` where it stands. */
  saySeq = $state(0)
  /** While leading someone to a stop they cannot see yet: the id they are
   *  being asked to press, and where that is leading. Null when the guide is
   *  simply standing beside something and explaining it. */
  awaiting = $state<string | null>(null)
  heading = $state<string | null>(null)
  /** A page goal is not the same as its rail button: the rail is a door that
   * must still be pressed before that page has been reached. */
  headingPage = $state<PageId | null>(null)
  /** True while "take me there" is pressing one safe navigation door. */
  escorting = $state(false)
  /** A direct control has already been tried at this stop. Keep the guide from
   * toggling the same menu or sending the same draft twice. Moving to another
   * stop clears it. */
  actedStopId = $state<string | null>(null)
  /** How many presses are left, for saying so. */
  stepsLeft = $state(0)
  /** The page the guide believes it is standing on, kept current by the sign
   *  watcher (placeWatch.ts). Read off the screen, never told by the app. */
  place = $state<PageId | null>(null)

  /** A requested door press is being verified. The screen changing during
   *  that check is already handled by `advance`, so `placeChanged` sits still
   *  rather than reacting twice (and cancelling the route midway). A counter
   *  and not a flag: two verifications can briefly overlap. */
  private moving = 0
  private escortTimer: ReturnType<typeof setTimeout> | null = null
  /** A route stop was genuinely pressed and is about to hand off to the next
   * stop. Keep the disappearing control from being mistaken for a lost page
   * during the same DOM update (starter cards are the common example). */
  private completingDirect: string | null = null
  private completionTimer: ReturnType<typeof setTimeout> | null = null

  private finishEscort() {
    this.escorting = false
    if (this.escortTimer) clearTimeout(this.escortTimer)
    this.escortTimer = null
  }

  /** A visible route control is itself evidence that its teaching step was
   * completed. Continue after the real click finishes; requiring a second
   * "Next" press made the guide ignore the action it had just watched. */
  private completeDirectRouteStop(id: string, evidence: 'click' | 'input', delay = 0) {
    if (!this.on || !this.route || this.awaiting || this.heading || this.headingPage) return
    if (this.stopId !== id) return
    const entry = guideEntry(id)
    if (!entry) return
    if (evidence === 'click' && entry.actionType !== 'click') return
    if (evidence === 'input' && entry.actionType !== 'focus' && entry.actionType !== 'input') return
    const routeId = this.route.id
    const routeAt = this.route.at
    this.completingDirect = id
    if (this.completionTimer) clearTimeout(this.completionTimer)
    this.completionTimer = setTimeout(() => {
      this.completionTimer = null
      const stillCurrent = this.on
        && this.route?.id === routeId
        && this.route?.at === routeAt
        && this.stopId === id
      this.completingDirect = null
      if (stillCurrent) this.next(true)
    }, delay)
  }

  /** Some controls are a lesson with one obvious next physical action. A
   * starter fills the draft; leaving the guide on the vanished card makes the
   * person hunt for what comes next. Hold that target for one render, then
   * point at Send without submitting on the person's behalf. */
  private continueDirectly(id: string, nextId: string, sentence: string, delay = 120) {
    if (!this.on || this.route || this.heading || this.headingPage || this.stopId !== id) return
    this.completingDirect = id
    if (this.completionTimer) clearTimeout(this.completionTimer)
    this.completionTimer = setTimeout(() => {
      this.completionTimer = null
      const stillCurrent = this.on && !this.route && !this.heading && !this.headingPage && this.stopId === id
      this.completingDirect = null
      if (stillCurrent) void this.goTo(nextId, sentence)
    }, delay)
  }

  /** Sending is the end of the rehearsal, so acknowledge the real click and
   * release the target instead of leaving the figure pointing at a button
   * that has already changed into Stop. */
  private acknowledgeSend(id: string, delay = 120) {
    if (!this.on || this.route || id !== 'chat.send' || this.stopId !== id) return
    this.completingDirect = id
    if (this.completionTimer) clearTimeout(this.completionTimer)
    this.completionTimer = setTimeout(() => {
      this.completionTimer = null
      const stillCurrent = this.on && !this.route && this.stopId === id
      this.completingDirect = null
      if (!stillCurrent) return
      this.stopId = null
      this.actedStopId = null
      const reply = t('guide.chat.waitingForAnswer' as never)
      this.say(reply)
      this.recordCurrentStep(reply, id)
    }, delay)
  }

  /** Input is route evidence only when it now contains a usable value. Focus
   * alone teaches where the field is; it must not pretend the user filled it. */
  completeInput(id: string) {
    const el = guideElement(id) as (HTMLElement & { value?: string; disabled?: boolean }) | null
    if (!el || el.disabled || !(typeof el.value === 'string') || !el.value.trim()) return
    if (!this.route && id === 'composer.input' && this.stopId === id) {
      this.continueDirectly(id, 'chat.send', t('guide.composer.readyToSend' as never), 550)
      return
    }
    // A short debounce lets a person finish the word before the figure walks.
    this.completeDirectRouteStop(id, 'input', 550)
  }

  /** Re-anchor a prepared route to the screen the person is actually on.
   *
   * Route stops are outcomes, not a blind playback cursor. If somebody opens
   * Settings themselves while the first-tour route is still pointing at the
   * account footer, replaying that old door is nonsense. Prefer the next stop
   * on the exact page, then the next stop in the same room. Never rewind a
   * route merely because the person moved backwards. */
  private routeIndexForPlace(page: PageId | null, from: number): number | null {
    if (!this.route || !page) return null
    const stops = GUIDE_ROUTES[this.route.id].stops
    const exact = stops.findIndex((id, index) => index >= from && GUIDE_MAP.find((entry) => entry.id === id)?.page === page)
    if (exact >= 0) return exact
    const room = roomOf(page)
    const sameRoom = stops.findIndex((id, index) => {
      const targetPage = GUIDE_MAP.find((entry) => entry.id === id)?.page
      return index >= from && !!targetPage && roomOf(targetPage) === room
    })
    return sameRoom >= 0 ? sameRoom : null
  }

  private reanchorRoute(page: PageId | null): boolean {
    if (!this.route) return false
    const at = this.routeIndexForPlace(page, this.route.at)
    if (at === null) return false
    const routeId = this.route.id
    const stopId = GUIDE_ROUTES[routeId].stops[at]
    const nav = resolveNavigation(stopId)
    // A matching page name is not enough if that page is still loading or its
    // required control does not exist in this state. In that case let the
    // ordinary lost-target path greet the page instead of keeping a dead tour.
    if (!nav.available && !nav.nextStepId) return false
    this.route = { id: routeId, at }
    void this.goTo(stopId)
    return true
  }

  private pressEscortStep() {
    const id = this.awaiting
    if (!this.escorting || !id) return
    const entry = guideEntry(id)
    if (!entry?.safe) {
      this.finishEscort()
      this.say(t('guide.safeRefusal'))
      return
    }
    const el = guideElement(id)
    if (!el || !onScreen(id)) {
      this.finishEscort()
      this.say(t('guide.notOnScreen'))
      return
    }

    // Start verification before the app handles the click. The document
    // listener ignores this synthetic press, while the real button handler
    // still runs normally.
    this.advance(id)
    this.pressing = true
    try {
      el.click()
    } finally {
      this.pressing = false
    }
  }

  /** Whether the current control can be operated from the guide. Ordinary
   * controls must be map-safe. A confirmation-required control (Send) is
   * offered only as an explicit, clearly named button in the guide; pressing
   * that button is the user's action-time confirmation. */
  canTakeCurrent(): boolean {
    if (this.awaiting) return true
    if (!this.stopId || this.actedStopId === this.stopId) return false
    const entry = guideEntry(this.stopId)
    return !!entry
      && (entry.actionType === 'click' || entry.actionType === 'focus')
      && (entry.safe || !!entry.confirmationRequired)
      && onScreen(entry.id)
  }

  /** Move a known route forward by exactly one verified safe press. */
  takeMe() {
    if (this.awaiting) {
      if (!this.heading && !this.headingPage) return
      this.escorting = true
      // The click on this Guide button must finish before opening an app menu;
      // otherwise that same click bubbles as an outside-click and closes the
      // menu immediately after it opens.
      if (this.escortTimer) clearTimeout(this.escortTimer)
      this.escortTimer = setTimeout(() => {
        this.escortTimer = null
        this.pressEscortStep()
      }, 0)
      return
    }

    const id = this.stopId
    if (!id || !this.canTakeCurrent()) return
    // A first-run machine can still explore every control, but sending a real
    // turn without a brain only produces an error. Divert at the last honest
    // moment and teach the missing setup instead of pretending the demo ran.
    if (id === 'chat.send' && !cockpit.model.provider) {
      void this.beginRoute('brain').then(() => this.say(t('guide.brainMissing')))
      return
    }
    const entry = guideEntry(id)
    const el = guideElement(id)
    if (!entry || !el) return
    this.escorting = true
    if (this.escortTimer) clearTimeout(this.escortTimer)
    this.escortTimer = setTimeout(() => {
      this.escortTimer = null
      this.pressing = true
      try {
        if (entry.actionType === 'focus') el.focus()
        else el.click()
        this.actedStopId = id
        if (id === 'chat.send' && !this.route) this.acknowledgeSend(id)
        else this.completeDirectRouteStop(id, 'click')
      } finally {
        this.pressing = false
        this.finishEscort()
      }
    }, 0)
  }

  /** The figure appears with the one-time offer after the tour. */
  offerTour() {
    this.on = true
    this.offering = true
    this.brain = 'map'
    this.stopId = null
    this.actedStopId = null
    this.completingDirect = null
    if (this.completionTimer) clearTimeout(this.completionTimer)
    this.completionTimer = null
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
    this.finishEscort()
    this.actedStopId = null
    this.completingDirect = null
    if (this.completionTimer) clearTimeout(this.completionTimer)
    this.completionTimer = null
    this.on = true
    this.offering = false
    this.transcript = []
    this.streamingText = ''
    this.asking = false
    this.welcoming = false
    this.openModel()
    if (routeId && GUIDE_ROUTES[routeId]) {
      const status = await this.beginRoute(routeId, '', initialStopId)
      if (status === 'arrived' && this.stopId && this.isInformationalFinal(this.stopId)) {
        const stopId = this.stopId
        this.settleInformationalFinal(stopId, `${t('guide.routeAlreadyHere' as never, { route: GUIDE_ROUTES[routeId].name, stop: guideText(stopId, 'name') })}\n\n${guideText(stopId, 'what')}`)
      } else {
        this.say(this.routeStartReply(routeId))
      }
    } else if (initialStopId) {
      this.route = null
      await this.goTo(initialStopId)
    } else {
      // A manual open is a hello first. The detailed room description is an
      // explicit prepared choice so merely asking for help never floods the
      // bubble with a long tour before the person has chosen one.
      this.route = null
      this.stopId = null
      this.place = currentPage()
      this.welcoming = true
      this.say(t('guide.welcome' as never))
    }
  }

  /** Turn the short first-open welcome into the complete explanation for the
   * room that is actually visible now. */
  describeCurrentPage() {
    if (!this.on) return
    this.welcoming = false
    this.place = currentPage() ?? this.place
    this.say(greetingFor(this.place))
  }

  /** Describe the step that is actually in front of the person. A prepared
   * route may re-anchor to their current room, so naming `route.stops[0]`
   * here can contradict the arrow on screen. */
  private routeStartReply(routeId: GuideRouteId): string {
    const route = GUIDE_ROUTES[routeId]
    const current = this.stopId ?? route.stops[this.route?.at ?? 0]
    const intro = t('guide.routeStarted' as never, {
      route: route.name,
      stop: guideText(current, 'name'),
    })
    const detail = guideText(current, 'what').trim()
    return detail ? `${intro}\n\n${detail}` : intro
  }

  private isInformationalFinal(id: string): boolean {
    if (!this.route) return false
    const route = GUIDE_ROUTES[this.route.id]
    const entry = guideEntry(id)
    return this.route.at === route.stops.length - 1 && entry?.actionType === 'none'
  }

  /** A page heading is proof of arrival, not another task for the user. Keep
   * pointing at it so the prepared explanations remain available, but close
   * the route so a fake "Next" button cannot appear after the destination. */
  private settleInformationalFinal(id: string, intro = ''): string {
    if (!this.isInformationalFinal(id)) return ''
    const detail = (intro || this.sentence || guideText(id, 'what')).trim()
    const reply = `${detail}${detail ? '\n\n' : ''}${t('guide.routeComplete' as never)}`
    this.route = null
    this.say(reply)
    return reply
  }

  /** Hand a prepared route to the runtime. The caller chooses only its id;
   * this store owns the stop list, progress, navigation checks, and pacing. */
  async beginRoute(
    routeId: GuideRouteId,
    sentence = '',
    initialStopId?: string,
  ): Promise<'arrived' | 'guiding' | 'blocked'> {
    this.finishEscort()
    const route = GUIDE_ROUTES[routeId]
    if (!route) return 'blocked'
    this.route = { id: routeId, at: 0 }
    const livePage = currentPage()
    const at = initialStopId ? 0 : (this.routeIndexForPlace(livePage, 0) ?? 0)
    this.route = { id: routeId, at }
    return this.goTo(initialStopId ?? route.stops[at], sentence)
  }

  /** The model brain, opened beside the map and never waited for: the map
   *  answers the first question while the session is still being built, and
   *  a session that fails to open leaves the map in charge. A model is a
   *  provider the chat already has — the guide has no dial of its own. */
  private openModel() {
    this.brain = 'map'
    this.sessionId = null
    if (!cockpit.model.provider && !guidePrefs.provider) return
    const index = GUIDE_MAP.map((e) => ({
      id: e.id,
      name: t(`guide.${e.id}.name` as any) || e.id,
      safe: e.safe,
    }))
    NewGuideSession(JSON.stringify(index), guidePrefs.provider, guidePrefs.model, guidePrefs.think)
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
    this.finishEscort()
    if (this.completionTimer) clearTimeout(this.completionTimer)
    this.completionTimer = null
    this.completingDirect = null
    this.on = false
    this.offering = false
    this.stopId = null
    this.actedStopId = null
    this.route = null
    this.transcript = []
    this.streamingText = ''
    this.asking = false
    this.welcoming = false
    this.sentence = ''
    this.awaiting = null
    this.heading = null
    this.headingPage = null
    this.stepsLeft = 0
    this.place = null
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

  /**
   * Go to a stop — by leading, not by teleporting.
   *
   * If the target is not on screen, the guide points at the button that leads
   * toward it and waits for the person to press it (`awaiting`). Each press
   * asks the question again, so the screen moving under the walk is expected
   * rather than a problem. When no visible door leads there it says so and
   * waits; it never changes the app's page state on the person's behalf.
   */
  async goTo(id: string, sentence = ''): Promise<'arrived' | 'guiding' | 'blocked'> {
    this.welcoming = false
    const entry = guideEntry(id)
    if (!entry) return 'blocked'
    if (this.stopId !== id) {
      this.actedStopId = null
      if (this.completionTimer) clearTimeout(this.completionTimer)
      this.completionTimer = null
      this.completingDirect = null
    }
    if (this.route) {
      const i = GUIDE_ROUTES[this.route.id].stops.indexOf(id)
      if (i >= 0) this.route = { id: this.route.id, at: i }
    }
    const nav = resolveNavigation(id)
    if (nav.nextStepId) {
      this.heading = id
      this.headingPage = null
      this.awaiting = nav.nextStepId
      this.stepsLeft = nav.stepsRemaining.length
      this.stopId = nav.nextStepId
      this.sentence = sentence || nav.blockedReason || ''
      this.place = currentPage()
      this.moveSeq++
      return 'guiding'
    }

    if (!nav.available) {
      this.finishEscort()
      this.heading = id
      this.headingPage = null
      this.awaiting = null
      this.stepsLeft = 0
      this.stopId = null
      this.place = currentPage()
      this.say(sentence || nav.blockedReason || t('guide.navigationBlocked'))
      return 'blocked'
    }

    this.awaiting = null
    this.finishEscort()
    this.heading = null
    this.headingPage = null
    this.stepsLeft = 0
    this.stopId = id
    this.sentence = sentence || nav.blockedReason || ''
    this.place = currentPage()
    if (this.stopId !== id) return 'blocked' // somewhere else was asked for meanwhile
    this.moveSeq++
    return 'arrived'
  }

  /** Lead to a whole page through its visible doors. The model's `goto`
   * command uses this same runtime; page requests never call app setters. */
  async goToPage(page: PageId, sentence = ''): Promise<'arrived' | 'guiding' | 'blocked'> {
    this.welcoming = false
    const nav = resolvePageNavigation(page)
    if (nav.available) {
      this.finishEscort()
      this.heading = null
      this.headingPage = null
      this.awaiting = null
      this.stepsLeft = 0
      this.stopId = null
      this.place = page
      this.say(sentence || arrivalFor(page))
      return 'arrived'
    }

    if (nav.nextStepId) {
      this.heading = null
      this.headingPage = page
      this.awaiting = nav.nextStepId
      this.stepsLeft = nav.stepsRemaining.length
      this.stopId = nav.nextStepId
      this.sentence = sentence
      this.place = currentPage()
      this.moveSeq++
      return 'guiding'
    }

    this.finishEscort()
    this.heading = null
    this.headingPage = page
    this.awaiting = null
    this.stepsLeft = 0
    this.stopId = null
    this.place = currentPage()
    this.say(sentence || nav.blockedReason || t('guide.navigationBlocked'))
    return 'blocked'
  }

  /** Finish a verified navigation press from the screen that exists now.
   *
   * Page headings are informational landmarks. If their page sign says we
   * are already there, that is sufficient arrival evidence even during the
   * brief frame before the heading receives a measurable rectangle. This is
   * deliberately limited to `actionType: none`; a form field or action on the
   * same page may still require another real press to reveal it. */
  private continueAfterNavigation(
    heading: string | null,
    headingPage: PageId | null,
    beforePage: PageId | null,
    page: PageId | null,
  ) {
    const pageIntro = page && page !== beforePage ? arrivalFor(page) : ''
    if (heading) {
      const entry = guideEntry(heading)
      const nav = resolveNavigation(heading)
      const informationalPageReached = entry?.actionType === 'none' && page === entry.page
      const arrived = nav.available || informationalPageReached
      const detail = arrived ? guideText(heading, 'what') : ''
      const narration = [pageIntro, detail].filter(Boolean).join('\n\n')

      if (!nav.available && informationalPageReached) {
        this.finishEscort()
        this.heading = null
        this.headingPage = null
        this.awaiting = null
        this.stepsLeft = 0
        this.stopId = heading
        this.sentence = narration
        this.place = page
        this.moveSeq++
        this.recordCurrentStep(detail, heading)
        this.settleInformationalFinal(heading)
        return
      }

      void this.goTo(heading, narration).then((status) => {
        if (status === 'arrived') {
          this.recordCurrentStep(detail || guideText(heading, 'what'), heading)
          this.settleInformationalFinal(heading)
        }
      })
      return
    }

    if (headingPage) {
      void this.goToPage(headingPage, pageIntro).then((status) => {
        if (status === 'arrived') this.recordCurrentStep(pageIntro)
      })
    }
  }

  /**
   * An ordinary press, while the guide happens to be up.
   *
   * It comments on exactly one thing: the button it just asked somebody to
   * press. That is the walk moving on. Every other press is a person using
   * their own app, and the guide says nothing about it — a guide that
   * announces each click is a guide people close (owner, 15 ก.ย. 2026:
   * *"เวลาผู้ใช้กดเองควรจะกดได้เลย"*).
   */
  advance(id: string) {
    if (!this.heading && !this.headingPage) {
      if (!this.route && id === 'chat.starter' && this.stopId === id) {
        this.continueDirectly(id, 'chat.send', t('guide.chat.starter.readyToSend' as never))
        return
      }
      if (!this.route && id === 'chat.send' && this.stopId === id) {
        this.acknowledgeSend(id)
        return
      }
      this.completeDirectRouteStop(id, 'click')
      return
    }
    if (id !== this.awaiting) return
    const heading = this.heading
    const headingPage = this.headingPage
    const beforePage = currentPage()
    const headingEntry = heading ? guideEntry(heading) : null
    const destinationAlreadyOpen = heading
      ? resolveNavigation(heading).available
        || (headingEntry?.actionType === 'none' && beforePage === headingEntry.page)
      : !!headingPage && resolvePageNavigation(headingPage).available

    // A selected rail may already have landed before the guide receives the
    // press (or the route may have been started on that page). In that case a
    // second click cannot produce a page change; accept the page/target as the
    // evidence and continue instead of displaying the misleading retry error.
    if (destinationAlreadyOpen) {
      this.finishEscort()
      this.continueAfterNavigation(heading, headingPage, beforePage, beforePage)
      return
    }
    // Guide's listener runs in capture phase, before the button handler. Wait
    // for real DOM evidence instead of assuming that a fixed delay succeeded.
    this.moving++
    void waitForNavigationProgress(
      () => heading ? resolveNavigation(heading) : resolvePageNavigation(headingPage!),
      id,
      beforePage,
    ).then((progress) => {
      this.moving--
      if (!this.on || this.heading !== heading || this.headingPage !== headingPage) return
      if (!progress.progressed) {
        const livePage = currentPage()
        const liveEntry = heading ? guideEntry(heading) : null
        const reachedAfterWait = heading
          ? resolveNavigation(heading).available
            || (liveEntry?.actionType === 'none' && livePage === liveEntry.page)
          : !!headingPage && resolvePageNavigation(headingPage).available
        if (reachedAfterWait) {
          this.finishEscort()
          this.continueAfterNavigation(heading, headingPage, beforePage, livePage)
          return
        }
        this.finishEscort()
        this.sentence = t('guide.stepNoChange')
        this.moveSeq++
        return
      }
      // One request is one visible step. Stop here so the person can see and
      // understand the next button before asking to be taken onward again.
      this.finishEscort()
      this.continueAfterNavigation(heading, headingPage, beforePage, progress.page)
    })
  }

  /**
   * The screen moved — a room opened, a section changed, a page went away.
   *
   * Three answers, and which one it gives is the whole behaviour:
   *
   *  - Leading somebody → ask the way again. `nextStepTo` reads the DOM, so
   *    asking from the new screen is all it takes: somebody who wandered off
   *    is led onward from where they now are, and somebody who arrived by
   *    their own route is simply told they are there.
   *  - Standing beside something that is no longer on screen → stop pointing
   *    at it and greet the new place. This is the thing the owner kept seeing:
   *    a bubble still naming a button from the page before.
   *  - Anything else → say nothing. A guide that announces every page you open
   *    is a guide you close; it only speaks when what it last said stopped
   *    being true.
   */
  placeChanged(page: PageId | null, contextChanged = false) {
    if (!this.on || this.offering || this.moving > 0) return
    const oldPlace = this.place
    this.place = page

    // Assistant and Code are separate desks even though both use the `chat`
    // page shell. A tour prepared for one must not survive a head switch and
    // continue teaching its controls on the other. Drop only desk-specific
    // routes; shared setup/about routes remain valid across the switch.
    if (contextChanged && this.route) {
      const desk = cockpit.desk === 'coding' ? 'coding' : 'assistant'
      if (!routeAllowedForDesk(this.route.id, desk)) {
        this.finishEscort()
        this.route = null
        this.stopId = null
        this.actedStopId = null
        this.heading = null
        this.headingPage = null
        this.awaiting = null
        this.stepsLeft = 0
        this.welcoming = false
        if (page) this.say(greetingFor(page))
        return
      }
    }

    // While the short welcome is showing, changing screens only updates the
    // room that "Explain this page" will describe. It must not start speaking
    // a long page description by itself.
    if (this.welcoming) return

    if (this.heading) {
      void this.goTo(this.heading)
      return
    }
    if (this.headingPage) {
      void this.goToPage(this.headingPage)
      return
    }

    if (this.stopId) {
      const stopPage = guideEntry(this.stopId)?.page
      // A globally mounted button can remain measurable behind another room.
      // Room identity outranks that stale rectangle. During a prepared route,
      // continue from the first relevant stop on the new screen.
      if (page && stopPage && roomOf(page) !== roomOf(stopPage)) {
        if (this.route && this.reanchorRoute(page)) return
        this.lostTarget()
        return
      }
      if (!onScreen(this.stopId)) this.lostTarget()
      return
    }

    // Free navigation without a stop: greet each new page the user visits
    if (page && (page !== oldPlace || contextChanged)) {
      this.say(greetingFor(page))
    }
  }

  /** Opening the right panel is entering a teachable sub-room, not merely
   * changing the window width. A prepared route keeps priority because it is
   * already leading to a specific tool; otherwise opening the panel starts a
   * real step-by-step tour instead of leaving the guide talking about its
   * toggle button. Code gets the project tools; Assistant gets the general
   * terminal/browser/files/index/slides/artifacts route. */
  inspectorChanged(open: boolean) {
    if (!open || !this.on || this.offering || this.moving > 0) return
    if (currentPage() !== 'chat') return
    if (this.route || this.heading || this.headingPage) return
    if (this.stopId && this.stopId !== 'topbar.inspector_btn') return
    this.finishEscort()
    this.stopId = null
    this.awaiting = null
    this.stepsLeft = 0
    this.heading = null
    this.headingPage = null
    this.welcoming = false
    this.place = 'chat'
    const routeId: GuideRouteId = cockpit.desk === 'coding' ? 'code_desk' : 'workbench_panel'
    const firstStop = GUIDE_ROUTES[routeId].stops[0]
    const firstDetail = guideText(firstStop, 'what').trim()
    const overview = t('guide.workbench.panel.opened' as never)
    const narration = firstDetail ? `${overview}\n\n${guideText(firstStop, 'name')}: ${firstDetail}` : overview
    void this.beginRoute(routeId, narration).then((status) => {
      if (status === 'blocked') this.say(t('guide.navigationBlocked'))
    })
  }

  /**
   * What it was standing beside stopped being on screen.
   *
   * Reached two ways, and the second is why this is its own door: the page
   * changed under it (above), or the BUTTON went away while the page stayed —
   * a menu closing, a card collapsing, a row filtered out. No sign moves for
   * that second kind, so nothing but the box itself can report it (follow.ts).
   */
  lostTarget() {
    if (!this.on || this.moving > 0 || this.completingDirect) return
    // Being led is not the same as standing: the door it was pointing at
    // disappearing usually means the person pressed it, so ask the way again
    // rather than giving up on where they were going.
    if (this.heading) {
      void this.goTo(this.heading)
      return
    }
    if (this.headingPage) {
      void this.goToPage(this.headingPage)
      return
    }
    if (!this.stopId) return
    this.stopId = null
    this.sentence = ''
    this.awaiting = null
    this.stepsLeft = 0
    // Off the route too: it was a walk through a page that is no longer in
    // front of us, and next()/prev() from there would teleport.
    this.route = null
    // The screen first, the last known place second. The box can go before the
    // sign watcher's beat has passed, and a fresh reading of the DOM beats a
    // value from 150ms ago — but a document with nothing signed at all answers
    // null, and there `place` is the better of the two.
    this.place = currentPage() ?? this.place
    this.say(greetingFor(this.place))
  }

  /** Say something where the figure stands. */
  say(text: string) {
    this.sentence = text
    this.saySeq++
  }

  /** Continue a question-started tour as a real conversation. The first
   * answer is already in the transcript; each later door and destination is
   * appended here so the visible chat never stays frozen on step one. */
  private recordCurrentStep(explicit = '', fallbackId: string | null = this.stopId) {
    if (this.transcript.length === 0) return
    const id = this.stopId ?? fallbackId
    const text = (explicit || (id ? guideText(id, 'what') : '')).trim()
    if (!text) return
    const last = [...this.transcript].reverse().find((item) => item.who === 'guide')
    if (last?.text.trim() === text && last.stopId === (id ?? undefined)) return
    this.transcript.push({ who: 'guide', text, stopId: id ?? undefined })
  }

  /** Answer a prepared chip from the product map. No model call: the wording
   * and recommendation stay stable and remain truthful offline. */
  answerPreset(kind: GuidePresetAnswer, id: string | null = this.stopId): string {
    if (!id) return ''
    const answer = presetAnswer(kind, id)
    this.say(answer)
    // A question-started guide renders its transcript instead of the animated
    // single-message surface. Put deterministic chip answers into that same
    // conversation, or the button works but its answer is drawn behind the
    // visible chat and appears to do nothing.
    this.recordCurrentStep(answer, id)
    return answer
  }

  /** "What is this?" — Shift+click, the deliberate ask. It abandons a walk in
   *  progress on purpose: the person has pointed at something else, and being
   *  dragged back to where the guide was heading is not guidance. */
  explain(id: string) {
    this.finishEscort()
    this.heading = null
    this.headingPage = null
    this.awaiting = null
    this.stepsLeft = 0
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
    this.welcoming = false
    this.finishEscort()
    this.transcript.push({ who: 'user', text: q, stopId: this.stopId ?? undefined })

    // Explicit tour intent belongs to the deterministic core even when a
    // model is connected. The model chooses prepared tours for fuzzy requests;
    // it never needs to reconstruct a route the program already knows.
    const routeId = routeForIntent(q)
    if (routeId) {
      const status = await this.beginRoute(routeId)
      let reply = this.routeStartReply(routeId)
      if (status === 'arrived' && this.stopId && this.isInformationalFinal(this.stopId)) {
        const stopId = this.stopId
        reply = this.settleInformationalFinal(stopId, `${t('guide.routeAlreadyHere' as never, { route: GUIDE_ROUTES[routeId].name, stop: guideText(stopId, 'name') })}\n\n${guideText(stopId, 'what')}`)
      } else {
        this.say(reply)
      }
      this.transcript.push({ who: 'guide', text: reply, stopId: this.stopId ?? undefined })
      return reply
    }

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
        const choice = modelTourChoice(answer || '')
        if (choice.recognized) {
          if (choice.routeId) {
            const status = await this.beginRoute(choice.routeId)
            let reply = this.routeStartReply(choice.routeId)
            if (status === 'arrived' && this.stopId && this.isInformationalFinal(this.stopId)) {
              const stopId = this.stopId
              reply = this.settleInformationalFinal(stopId, `${t('guide.routeAlreadyHere' as never, { route: GUIDE_ROUTES[choice.routeId].name, stop: guideText(stopId, 'name') })}\n\n${guideText(stopId, 'what')}`)
            } else {
              this.say(reply)
            }
            this.transcript.push({ who: 'guide', text: reply, stopId: this.stopId ?? undefined })
            return reply
          }
          // It was definitely a machine directive, but not one this build
          // knows. Do not expose JSON to the person; let the local map answer.
          this.brain = 'map'
        }
        const text = compactGuideReply(answer || '')
        if (!choice.recognized && text) {
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

  next(completed = false) {
    if (!this.route) return
    this.finishEscort()
    const r = GUIDE_ROUTES[this.route.id]
    if (this.route.at >= r.stops.length - 1) {
      const completedStop = this.stopId
      const reply = completedStop === 'chat.send'
        ? t('guide.chat.waitingForAnswer' as never)
        : t('guide.routeComplete' as never)
      this.route = null
      this.stopId = null
      this.heading = null
      this.headingPage = null
      this.awaiting = null
      this.stepsLeft = 0
      this.say(reply)
      this.recordCurrentStep(reply, completedStop)
      return
    }
    // Some stops only exist in the matching state: starter cards disappear
    // after the first message, and the team picker is hidden while speaking
    // directly to one agent. Walk past a truly absent optional stop instead
    // of stranding the tour on a control this screen cannot render.
    let at = this.route.at + 1
    while (at < r.stops.length - 1) {
      const nav = resolveNavigation(r.stops[at])
      if (nav.available || nav.nextStepId) break
      at++
    }
    this.route = { id: this.route.id, at }
    const destination = r.stops[at]
    const detail = guideText(destination, 'what')
    const narration = completed
      ? `${t('guide.stepCompleted' as never, { name: guideText(destination, 'name') })}${detail ? `\n\n${detail}` : ''}`
      : ''
    void this.goTo(destination, narration).then((status) => {
      if (status === 'arrived') {
        // A typed question renders the transcript, not the single-message
        // copy animated by say(). Keep the completion acknowledgement in that
        // visible transcript too; otherwise the guide did advance, but looked
        // as if it had simply replaced one explanation with another.
        this.recordCurrentStep(narration || guideText(destination, 'what'), destination)
        this.settleInformationalFinal(destination)
      }
    })
  }

  prev() {
    if (!this.route) return
    this.finishEscort()
    const r = GUIDE_ROUTES[this.route.id]
    if (this.route.at <= 0) return
    const at = this.route.at - 1
    this.route = { id: this.route.id, at }
    const destination = r.stops[at]
    void this.goTo(destination).then((status) => {
      if (status === 'arrived') this.recordCurrentStep(guideText(destination, 'what'), destination)
    })
  }

  /** Press an element for the user. The map's `safe` flag is the gate, and it
   *  is checked here — the one place that clicks — for both the bubble's button
   *  and the model's `press` action. */
  press(id: string | null = this.stopId): { ok: boolean; message: string } {
    if (!id) return { ok: false, message: '' }
    const entry = guideEntry(id)
    if (!entry || !entry.safe) {
      return { ok: false, message: t('guide.safeRefusal') }
    }
    const el = guideElement(id)
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
