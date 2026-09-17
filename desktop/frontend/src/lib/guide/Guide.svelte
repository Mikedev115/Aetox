<script lang="ts">
  // The guide on screen: the companion's body with the fourth rank on its
  // corner, walking to whatever the store's stopId names and saying the
  // store's sentence beside it. Everything here reacts to guideState — the
  // figure never decides where to go, only how to get there and where the
  // bubble fits (on the side AWAY from the target, so the card never covers
  // the button it is about; below the figure when it stands near the top).
  //
  // While the guide is open a click on any mapped element asks about it
  // instead of pressing it (plan §4.4). The pill at the top says so, and
  // Esc or × ends it; the guide's own controls are exempt (.guide-ui).
  import { onMount, tick, untrack } from 'svelte'
  import Mascot from '../mascot/Mascot.svelte'
  import Icon from '../Icon.svelte'
  import { guide } from './guideState.svelte'
  import { contextualGuideStarts, offeredWalkCategories, offeredWalks, type GuideContextStart, type GuideWalkCategoryId } from './greeting'
  import GuidePanel from './GuidePanel.svelte'
  import { guidePrefs, setGuidePref } from './guidePrefs.svelte'
  import { clampSize } from '../mascot/companionSetting.svelte'
  import { GUIDE_MAP, guideElement, guideEntry, guideText, GUIDE_CATALOG, type ConceptCatalogEntry } from './map'
  import { GUIDE_ROUTES } from './routes'
  import { BUBBLE, standBesideReal, restingSpot, insideWindow, walkTurn, walkMs, GEOMETRY, type Rect } from './walk'
  import { watchBox } from './follow'
  import { speak, stopSpeechIf } from '../speech.svelte'
  import { t } from '../i18n.svelte'
  import { handleGuideAsk } from './guideSession'
  import { watchInspector, watchPlace } from './placeWatch'
  import { currentPage } from './where'
  import { renderMarkdown } from '../markdown'
  import { cockpit } from '../stores/cockpit.svelte'
  import { EventsOn } from '../../../wailsjs/runtime/runtime'
  import { chooseGuideSettingsLayout, type GuideSettingsLayout } from './settingsLayout'

  // The guide has its own remembered size, but uses the same caps and resize
  // behaviour as the companion.
  const SIZE = $derived(guidePrefs.size)

  // Placed on the FIRST frame, not moved into place after it. x and y used to
  // start at 0, so the figure rendered in the top-left corner and then rode the
  // 800ms walk transition across the whole window to its resting spot — which
  // reads as a thing sliding in from nowhere, every single time it is opened
  // (owner, 15 ก.ย. 2026: "กดไกด์แล้วเหมือนมันลอยออกมาจากตรงไหนไม่รู้").
  // It now appears where it belongs and fades in on the spot; walking is for
  // going somewhere, not for arriving.
  const first = typeof window === 'undefined' ? { x: 0, y: 0 } : homeSpot()
  let x = $state(first.x)
  let y = $state(first.y)
  let placed = $state(typeof window !== 'undefined')
  let turn = $state<number | undefined>(undefined)
  let pose = $state<'idle' | 'walk' | 'presenting' | 'helping' | 'asking' | 'thinking' | 'answering'>('asking')
  let flip = $state(false)
  let below = $state(false)
  let placementMode = $state<'side' | 'docked'>('side')
  let dockPosition = $state<'bottom' | 'top'>('bottom')
  let sayEl = $state<HTMLElement | null>(null)
  let showConcept = $state(false)
  let expanded = $state(false)
  let collapsed = $state(false)

  // Responsive flip and below: ensures the bubble NEVER runs off screen
  const effectiveFlip = $derived.by(() => {
    if (placementMode === 'docked') return false
    const winW = typeof window !== 'undefined' ? window.innerWidth : 1200
    const leftSpace = x - GEOMETRY.GAP_BUBBLE
    const rightSpace = winW - (x + SIZE + GEOMETRY.GAP_BUBBLE)
    if (leftSpace < BUBBLE && rightSpace >= leftSpace) return true
    if (rightSpace < BUBBLE && leftSpace > rightSpace) return false
    return flip
  })

  const effectiveBelow = $derived.by(() => {
    if (placementMode === 'docked') return false
    const winH = typeof window !== 'undefined' ? window.innerHeight : 800
    if (y < GEOMETRY.BUBBLE_DROP) return true
    if (y > winH - GEOMETRY.BUBBLE_DROP) return false
    return below
  })

  let chatStreamEl = $state<HTMLElement | null>(null)
  function scrollToBottom() {
    tick().then(() => {
      if (chatStreamEl) {
        chatStreamEl.scrollTop = chatStreamEl.scrollHeight
      }
    })
  }

  $effect(() => {
    if (guide.transcript.length || guide.streamingText) {
      scrollToBottom()
    }
  })

  function clearTranscript() {
    guide.transcript = []
  }

  function toggleExpanded() {
    collapsed = false
    expanded = !expanded
    scrollToBottom()
  }

  /** Minimise the conversation to its title bar without dismissing the guide.
   * The guide keeps its route and answer in memory, so opening it again is an
   * instant return to the same point rather than a restart. */
  function toggleCollapsed() {
    collapsed = !collapsed
    if (collapsed) {
      expanded = false
      face = 'talk'
    }
  }

  function onBubbleHeadDoubleClick(e: MouseEvent) {
    if ((e.target as HTMLElement).closest('button')) return
    if (collapsed) toggleCollapsed()
    else toggleExpanded()
  }

  let ring = $state<{ l: number; t: number; w: number; h: number } | null>(null)
  type TargetMarker = { x: number; y: number; below: boolean; left: boolean; right: boolean }

  /** Reduce the target rectangle to one reliable point. The old marker used
   * the whole target box as a fixed-position anchor; a wide row or an
   * off-screen duplicate could therefore put its label beyond the window.
   * The marker now aims at the visible centre and keeps its own glyph inside
   * the viewport even during panel animation. */
  function markerForTarget(target: NonNullable<typeof ring>): TargetMarker {
    const width = typeof window === 'undefined' ? target.l + target.w + 48 : window.innerWidth
    const height = typeof window === 'undefined' ? target.t + target.h + 48 : window.innerHeight
    const edge = 24
    const below = target.t < 42
    const rawX = target.l + target.w / 2
    const rawY = below ? target.t + target.h : target.t
    const x = Math.max(edge, Math.min(Math.max(edge, width - edge), rawX))
    const y = Math.max(edge, Math.min(Math.max(edge, height - edge), rawY))
    return {
      x,
      y,
      below,
      left: x < 92,
      right: x > width - 130,
    }
  }
  let shown = $state('')
  let typing = $state(false)
  let askInput = $state('')
  /** Which face of the bubble is showing. */
  let face = $state<'talk' | 'settings'>('talk')
  let settingsLayout = $state<GuideSettingsLayout>({
    placement: 'right',
    tabEdge: 'right',
    align: 'end',
    maxHeight: 460,
  })
  // The landing menu is an overview: six choices let somebody see the shape
  // of the page without having to spin the list immediately. Once a walk is
  // in progress, keep the smaller three-action deck so the chat stays easy to
  // read beside the thing being explained.
  const START_CHOICES_PER_PAGE = 6
  const STEP_COMMANDS_PER_PAGE = 3
  let walkCategory = $state<GuideWalkCategoryId | null>(null)
  let walkPage = $state(0)
  let stepCommandPage = $state(0)
  let seq = 0
  // A transcript already renders the completed answer from the store. `say()`
  // may still be animating the hidden single-message copy, which must not hide
  // the useful next actions underneath the visible reply.
  const canOfferActions = $derived(!guide.asking && (!typing || guide.transcript.length > 0))
  async function layoutSettingsMenu() {
    await tick()
    if (!sayEl || typeof window === 'undefined') return
    const r = sayEl.getBoundingClientRect()
    settingsLayout = chooseGuideSettingsLayout(
      { left: r.left, right: r.right, top: r.top, bottom: r.bottom },
      { width: window.innerWidth, height: window.innerHeight },
    )
  }

  function toggleSettings() {
    face = face === 'settings' ? 'talk' : 'settings'
    if (face === 'settings') void layoutSettingsMenu()
  }

  function pageOf<T>(items: T[], page: number, perPage: number): T[] {
    if (items.length <= perPage) return items
    const pages = Math.ceil(items.length / perPage)
    const start = (page % pages) * perPage
    return items.slice(start, start + perPage)
  }

  function chooseWalkCategory(id: GuideWalkCategoryId) {
    walkCategory = id
    walkPage = 0
  }

  function leaveWalkCategory() {
    walkCategory = null
    walkPage = 0
  }

  function startWalk(id: Parameters<typeof guide.start>[0]) {
    walkPage = 0
    void guide.start(id)
  }

  function startContextChoice(choice: GuideContextStart) {
    walkPage = 0
    if (choice.kind === 'page') {
      guide.describeCurrentPage()
    } else if (choice.kind === 'route') {
      void guide.start(choice.id as Parameters<typeof guide.start>[0])
    } else {
      void guide.goTo(choice.id)
    }
  }

  function activeGuideDesk() {
    return cockpit.desk === 'coding' ? 'coding' as const : 'assistant' as const
  }

  /** Keep the opening deck useful and visually stable: page-specific actions
   * come first, then routes from different categories fill any empty slots.
   * That gives the requested 2 × 3 overview even on a sparse page, while a
   * dense page can still rotate through its remaining contextual actions. */
  function landingChoices(contextChoices: GuideContextStart[]): GuideContextStart[] {
    if (!guide.welcoming) return [...contextChoices]
    const choices: GuideContextStart[] = [
      { id: 'current', kind: 'page', label: t('guide.describeThisPage' as never) },
      ...contextChoices,
    ]
    if (choices.length >= START_CHOICES_PER_PAGE) return choices

    const seen = new Set(choices.map((choice) => `${choice.kind}:${choice.id}`))
    const addRoute = (route: ReturnType<typeof offeredWalks>[number]) => {
      const key = `route:${route.id}`
      if (seen.has(key) || choices.length >= START_CHOICES_PER_PAGE) return
      seen.add(key)
      choices.push({ id: route.id, kind: 'route', label: route.label })
    }

    // First pass keeps the extras diverse; second pass fills unusually sparse
    // pages without inventing generic or dead buttons.
    for (const category of offeredWalkCategories()) {
      const route = offeredWalks(category.id, activeGuideDesk()).find((candidate) => !seen.has(`route:${candidate.id}`))
      if (route) addRoute(route)
    }
    for (const route of offeredWalks(undefined, activeGuideDesk())) addRoute(route)
    return choices
  }

  function explainShiftClick() {
    guide.say(t('guide.shiftClickDetail' as never))
  }

  type StepCommand = 'take' | 'explain' | 'common' | 'recommend' | 'skip'
  type OtherTopic = ReturnType<typeof offeredWalks>[number]
  type StepDeckAction =
    | { id: string; kind: 'command'; command: StepCommand }
    | { id: string; kind: 'topic'; topic: OtherTopic }

  function stepCommands(): StepCommand[] {
    const commands: StepCommand[] = guide.canTakeCurrent()
      ? ['take', 'explain', 'common', 'recommend']
      : ['explain', 'common', 'recommend']
    if (guide.route) commands.push('skip')
    return commands
  }

  /** Follow the three questions about this control with a deliberately mixed
   * set of routes: one setup topic, one everyday-use topic and one system
   * topic. That keeps the deck useful after the immediate question is
   * answered instead of making every rotation another wording of the same
   * thing. A route containing the current stop is skipped as "more of this". */
  function otherTopics(): OtherTopic[] {
    const current = guide.stopId
    return offeredWalkCategories()
      .map((category) => offeredWalks(category.id, activeGuideDesk()).find((walk) => {
        if (walk.id === guide.route?.id) return false
        return !current || !GUIDE_CATALOG.some((entry) => entry.kind === 'route' && entry.id === walk.id && entry.stops.includes(current))
      }))
      .filter((walk): walk is OtherTopic => !!walk)
  }

  function stepDeckActions(): StepDeckAction[] {
    const commands = stepCommands()
    const immediateCommands = commands.filter((command) => command !== 'skip')
    const trailingCommands = commands.filter((command) => command === 'skip')
    return [
      ...immediateCommands.map((command) => ({ id: `command:${command}`, kind: 'command' as const, command })),
      ...otherTopics().map((topic) => ({ id: `topic:${topic.id}`, kind: 'topic' as const, topic })),
      ...trailingCommands.map((command) => ({ id: `command:${command}`, kind: 'command' as const, command })),
    ]
  }

  function stepCommandLabel(id: StepCommand): string {
    if (id === 'take') {
      if (guide.escorting) return t('guide.takingYou')
      if (guide.stopId === 'chat.send') {
        return cockpit.model.provider ? t('guide.sendThis') : t('guide.connectBrainFirst')
      }
      if (guide.stopId === 'chat.starter') return t('guide.tryThisPrompt')
      return t('guide.takeMe')
    }
    if (id === 'explain') return t('guide.command.explainMore')
    if (id === 'common') return t('guide.command.commonUse')
    if (id === 'recommend') return t('guide.command.recommend')
    return t('guide.command.skip')
  }

  function runStepCommand(id: StepCommand) {
    if (id === 'take') {
      guide.takeMe()
      return
    }
    if (id === 'skip') {
      guide.next()
      return
    }
    guide.answerPreset(id, guide.stopId)
  }

  // Re-evaluate after every move, dock, expand and bubble-side change. Opening
  // settings changes only the attached controls, never the chat placement.
  $effect(() => {
    void face
    void x
    void y
    void placementMode
    void expanded
    void effectiveFlip
    void effectiveBelow
    untrack(() => void layoutSettingsMenu())
  })

  const stop = $derived(guide.stopId ? guideEntry(guide.stopId) : null)
  const stopName = $derived(guide.stopId ? guideText(guide.stopId, 'name') : '')
  const relatedConcept = $derived.by(() => {
    if (!guide.stopId) return null
    return GUIDE_CATALOG.find((e): e is ConceptCatalogEntry => e.kind === 'concept' && e.relatedTargets.includes(guide.stopId!)) ?? null
  })

  function sleep(ms: number) {
    return new Promise((r) => setTimeout(r, ms))
  }

  function reduced(): boolean {
    return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
  }

  function viewport() {
    return { width: window.innerWidth, height: window.innerHeight }
  }

  // Where it waits when it has nothing to point at: the spot the user dragged
  // it to, if they have ever moved it, and otherwise the corner walk.ts picks.
  // One answer, asked by all three places that need it — the first frame, the
  // walk that finds no target, and the step back after a stop goes away — so a
  // figure that was moved cannot walk home to somewhere else.
  function homeSpot(): { x: number; y: number } {
    const win = viewport()
    const own = guidePrefs.spot
    return own ? insideWindow(own, win, SIZE) : restingSpot(win, SIZE)
  }

  function home() {
    ;({ x, y } = homeSpot())
    placed = true
  }

  async function say(text: string) {
    const my = ++seq
    if (shown === text) {
      typing = false
      return
    }
    shown = ''
    typing = true
    if (reduced()) {
      shown = text
      typing = false
      return
    }
    for (const ch of text) {
      if (my !== seq) return
      shown += ch
      await sleep(ch === ' ' ? 20 : 10)
    }
    typing = false
  }

  function voice(text: string) {
    if (guidePrefs.voice && text) speak('guide', text, () => {})
  }

  // Where to stand for a target rect: walk.ts decides, this only records it.
  function placeBeside(r: Rect) {
    const bubbleSize = sayEl ? { width: sayEl.offsetWidth, height: sayEl.offsetHeight } : { width: BUBBLE, height: 200 }
    const st = standBesideReal(r, bubbleSize, { width: window.innerWidth, height: window.innerHeight }, SIZE)
    ring = st.ring
    flip = st.flip
    below = st.below
    placementMode = st.mode
    dockPosition = st.dockPosition ?? 'bottom'
    return { tx: st.x, ty: st.y }
  }

  // Dragging the figure.
  //
  // It is a guide, so it walks to whatever it is explaining and stands beside
  // it — that is its job and a drag must never take it away. What a drag
  // decides is the OTHER place: where it waits when there is nothing to point
  // at. So the hand sets `home`, and the walk still owns every journey
  // (owner, 15 ก.ย. 2026: *"อยากให้ลากได้ครับ … แต่พอถึงเวลาอธิบายมันควรจะ
  // เดินไปเดินมาได้ เพราะมันคือไกด์"*).
  //
  // The threshold is what keeps the three little buttons on the frame usable:
  // under it the press was a click and nothing moved at all.
  const CLICK_PX = 4
  let dragging = $state(false)
  /** Bumped when a hand takes hold. A walk in flight compares it and gives up
   *  its finishing correction — the hand outranks a journey that was decided
   *  before it started. */
  let grabbed = 0
  let drag: { dx: number; dy: number; x0: number; y0: number; moved: boolean } | null = null

  function onGrab(e: PointerEvent) {
    if (e.button !== 0) return
    drag = { dx: e.clientX - x, dy: e.clientY - y, x0: e.clientX, y0: e.clientY, moved: false }
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }

  function onDrag(e: PointerEvent) {
    if (!drag) return
    if (!drag.moved && Math.hypot(e.clientX - drag.x0, e.clientY - drag.y0) < CLICK_PX) return
    if (!drag.moved) {
      drag.moved = true
      dragging = true
      grabbed++
      // Whatever it was doing, it is being carried now: no walk pose, no turn.
      turn = undefined
      if (pose === 'walk') pose = 'idle'
    }
    ;({ x, y } = insideWindow({ x: e.clientX - drag.dx, y: e.clientY - drag.dy }, viewport(), SIZE))
    placed = true
  }

  function onDrop() {
    if (!drag) return
    const moved = drag.moved
    drag = null
    if (!moved) return // a click on the figure, which is not a control
    dragging = false
    pose = 'asking'
    // Remembered, because the whole point of moving it is that it stays moved.
    setGuidePref('spot', { x, y })
  }

  // Same resize contract as the companion: drag the bottom-right corner,
  // clamp to the shared minimum/maximum, and remember the result.
  let resizing = $state(false)
  let grip: { x0: number; y0: number; size0: number } | null = null

  function onGripDown(e: PointerEvent) {
    if (e.button !== 0) return
    e.stopPropagation()
    grip = { x0: e.clientX, y0: e.clientY, size0: SIZE }
    resizing = true
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }

  function onGripMove(e: PointerEvent) {
    if (!grip) return
    e.stopPropagation()
    const d = Math.max(e.clientX - grip.x0, e.clientY - grip.y0)
    const next = clampSize(grip.size0 + d)
    if (next === SIZE) return
    setGuidePref('size', next)
    ;({ x, y } = insideWindow({ x, y }, viewport(), next))
  }

  function onGripUp(e: PointerEvent) {
    if (!grip) return
    e.stopPropagation()
    grip = null
    resizing = false
    if (guidePrefs.spot) setGuidePref('spot', { x, y })
  }

  async function moveToTarget(stopId: string, sentence: string) {
    // A walk takes up to a second, and an answer can land inside it — the
    // model's own `point` walks the figure and the reply arrives after. The
    // walk must not then speak over the newer words, so it remembers what had
    // been said when it set off and stays quiet if that moved.
    const saidAtStart = guide.saySeq
    seq++
    shown = ''
    typing = false
    stopSpeechIf('guide')
    await tick()
    let el = guideElement(stopId)
    if (el) {
      // Instant, not smooth: a rect read while a smooth scroll is still on
      // its way is the button's position halfway there, and the figure then
      // stands beside where it was.
      try {
        el.scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'auto' })
      } catch {
        // not scrollable — it is where it is
      }
      await tick()
      el = guideElement(stopId)
    }
    let tx: number
    let ty: number
    if (el) {
      ;({ tx, ty } = placeBeside(el.getBoundingClientRect()))
    } else {
      ring = null
      home()
      tx = x
      ty = y
    }
    const heldAt = grabbed
    const dx = tx - x
    if (placed && !reduced()) {
      pose = 'walk'
      turn = walkTurn(dx)
    }
    x = tx
    y = ty
    placed = true
    await sleep(walkMs(dx, reduced()))
    turn = undefined
    // The page may have settled under the walk — a section that finished
    // loading, a list that grew. Stand beside where the target is NOW.
    if (el && guide.stopId === stopId && grabbed === heldAt && !dragging) {
      const again = guideElement(stopId)
      if (again) {
        const r = again.getBoundingClientRect()
        if (ring && (Math.abs(r.left - 4 - ring.l) > 2 || Math.abs(r.top - 4 - ring.t) > 2)) {
          ;({ tx, ty } = placeBeside(r))
          x = tx
          y = ty
        }
      }
    }
    const entry = guideEntry(stopId)
    pose = !el ? 'thinking' : entry?.safe ? 'presenting' : 'helping'
    if (guide.saySeq !== saidAtStart) return
    const text = sentence || (el ? guideText(stopId, 'what') : t('guide.notOnScreen'))
    void say(text)
    voice(text)
  }

  async function onAskSubmit(e: Event) {
    e.preventDefault()
    const q = askInput.trim()
    if (!q) return
    askInput = ''
    stopSpeechIf('guide')
    pose = 'thinking'
    scrollToBottom()
    await guide.ask(q)
    pose = 'asking'
    scrollToBottom()
  }

  function onClose() {
    stopSpeechIf('guide')
    face = 'talk'
    void guide.stop()
  }

  // A walk: the store bumped moveSeq after identifying the visible stop.
  $effect(() => {
    const n = guide.moveSeq
    if (!n || !guide.on) return
    const id = guide.stopId
    const sentence = guide.sentence
    if (id) untrack(() => void moveToTarget(id, sentence))
  })

  // Nothing to stand beside any more: no ring, and the figure walks back to its
  // corner. The store drops stopId when the page it was pointing at is gone
  // (guideState.placeChanged), and a highlight box left drawn around where a
  // button used to be is the guide insisting on a screen that has moved on.
  async function stepBack() {
    ring = null
    const to = homeSpot()
    const dx = to.x - x
    if (Math.abs(dx) < 2 && Math.abs(to.y - y) < 2) return // already there
    // The words are left alone on purpose: whatever cleared the stop says
    // something in the same breath, and the walk back must not race it.
    if (!reduced()) {
      pose = 'walk'
      turn = walkTurn(dx)
    }
    x = to.x
    y = to.y
    await sleep(walkMs(dx, reduced()))
    turn = undefined
    if (pose === 'walk') pose = 'asking'
  }

  $effect(() => {
    if (guide.on && !guide.stopId) untrack(() => void stepBack())
  })

  // Once it has arrived, it STAYS on the thing it is pointing at.
  //
  // Arriving measures the target once, and a page that has just opened is
  // where boxes move most — a card finishing its load, a font swapping, an
  // image arriving above the button. A second later the ring was a rectangle
  // around empty space on a page the guide really had reached, which is what
  // "ทำไมเวลากดเปลี่ยนหน้ามันยังไม่รู้ตัว" looked like from the outside
  // (owner, 15 ก.ย. 2026). Re-runs on every walk, so it is always watching the
  // element the figure is actually beside.
  $effect(() => {
    const id = guide.stopId
    void guide.moveSeq
    if (!id || !guide.on) return
    const el = guideElement(id)
    if (!el) return
    return watchBox(
      el,
      (box) => {
        // The hand outranks the measurement: somebody holding the figure is
        // not asking it to chase a button.
        if (dragging) return
        const { tx, ty } = placeBeside(box)
        x = tx
        y = ty
      },
      () => guide.lostTarget(),
    )
  })

  // Words without a walk: the model's answer, the map's refusal, a press.
  $effect(() => {
    const n = guide.saySeq
    if (!n || !guide.on) return
    const text = guide.sentence
    untrack(() => {
      pose = 'answering'
      void say(text)
      voice(text)
    })
  })

  // While the model works the figure thinks; its streamed words show as they come.
  $effect(() => {
    if (guide.asking) {
      pose = 'thinking'
      if (guide.streamingText) {
        shown = guide.streamingText
        typing = true
      }
    }
  })

  onMount(() => {
    document.body.classList.add('guide-mode')
    if (!placed) home()

    // Two gestures, and the difference between them is the whole rule:
    //
    //   click        — using the app. The guide does not touch the event at
    //                  all; it only notices when the press is the one it just
    //                  asked for, and moves the walk on.
    //   Shift+click  — asking about the thing. THAT one is taken: somebody
    //                  holding Shift wants to know what the button is, not to
    //                  set it off, and sending a message to find out what the
    //                  send button does is a poor trade.
    //
    // An open guide used to swallow every click, which made the app read-only
    // while it was up (owner, 15 ก.ย. 2026: "เวลาผู้ใช้กดเองควรจะกดได้เลย …
    // หรือเราทำให้ แบบ กดชิฟแล้วคลิก จะเป็นการถามไกด์ก็ได้ว่าปุ่มนี้คืออะไร").
    // Capture phase, because suppressing an action has to happen before the
    // element's own handler — but the plain-click path returns without
    // touching the event, so an ordinary press is exactly as it was.
    const onDocClick = (e: MouseEvent) => {
      if (!guide.on || guide.pressing) return
      const target = e.target as HTMLElement | null
      if (!target || target.closest('.guide-ui')) return
      const id = target.closest<HTMLElement>('[data-guide]')?.getAttribute('data-guide')
      if (!id) return
      if (e.shiftKey) {
        e.preventDefault()
        e.stopPropagation()
        guide.explain(id)
      } else {
        guide.advance(id)
      }
    }
    const onDocInput = (e: Event) => {
      if (!guide.on || guide.pressing) return
      const target = e.target as HTMLElement | null
      if (!target || target.closest('.guide-ui')) return
      const id = target.closest<HTMLElement>('[data-guide]')?.getAttribute('data-guide')
      if (id) guide.completeInput(id)
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    const onResize = () => {
      if (guide.stopId && guide.on) guide.moveSeq++
      void layoutSettingsMenu()
    }

    // The guide's one sense of place: the signs on the pages, watched rather
    // than asked about, so using the app under an open guide moves the guide
    // too (placeWatch.ts).
    const offPlace = watchPlace((page, contextChanged) => guide.placeChanged(page, contextChanged))
    const offInspector = watchInspector((open) => guide.inspectorChanged(open))

    const offGuide = EventsOn('screen:guide', (ask: any) => {
      void handleGuideAsk(ask)
    })
    const offChunk = EventsOn('agent:chunk', (ev: any) => {
      if (guide.sessionId && ev?.sessionId === guide.sessionId) {
        guide.onChunk(ev?.data?.text || '', !!ev?.data?.replace)
      }
    })

    document.addEventListener('click', onDocClick, true)
    document.addEventListener('input', onDocInput, true)
    window.addEventListener('keydown', onKeyDown)
    window.addEventListener('resize', onResize)

    return () => {
      document.body.classList.remove('guide-mode')
      document.removeEventListener('click', onDocClick, true)
      document.removeEventListener('input', onDocInput, true)
      window.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('resize', onResize)
      offPlace()
      offInspector()
      offGuide()
      offChunk()
      stopSpeechIf('guide')
    }
  })
</script>

<!-- The pill that says the window is in guide mode, and the one way out besides Esc -->
<div class="guide-banner guide-ui" role="status">
  <Icon name="compass" size={13} />
  <span class="guide-banner-text">{t('guide.modeBanner')}</span>
  <button type="button" class="guide-banner-close" onclick={onClose} aria-label={t('guide.close')}>
    <Icon name="x" size={13} />
  </button>
</div>

<!-- A stop can stay selected after the user has already pressed it so the
     explanation and prepared answers still know their subject. The target
     marker is deliberately only an arrow: outlining the whole element made
     the guide depend on every screen width and every responsive layout. -->
{#if ring}
  {@const marker = markerForTarget(ring)}
  <div
    class="guide-target-anchor guide-ui"
    class:awaiting={!!guide.awaiting}
    class:label-below={marker.below}
    class:label-left={marker.left}
    class:label-right={marker.right}
    style="left:{marker.x}px;top:{marker.y}px"
    aria-hidden="true"
  >
    <span class="guide-target-label">
      {#if guide.awaiting}<span class="guide-target-text">{t('guide.clickHere')}</span>{/if}
      <span class="guide-target-arrow"><Icon name="arrowDown" size={19} /></span>
    </span>
  </div>
{/if}

<div
  class="guide-mascot-wrap guide-ui"
  class:flip={effectiveFlip}
  class:below={effectiveBelow}
  class:docked={placementMode === 'docked'}
  class:docked-top={placementMode === 'docked' && dockPosition === 'top'}
  class:docked-bottom={placementMode === 'docked' && dockPosition === 'bottom'}
  class:walking={pose === 'walk'}
  class:dragging
  class:resizing
  style="transform:translate({x}px,{y}px);--guide-x:{x}px;--guide-y:{y}px"
>
  <!-- The figure is the handle. Not the whole wrap: the bubble is under the
       same roof and text in it has to stay selectable, and the three frame
       buttons have to stay pressable — which the 4px threshold in onDrag is
       what protects. -->
  <span
    class="ranked-guide"
    style="width:{SIZE}px;height:{SIZE}px"
    role="img"
    aria-label={t('rank.guide')}
    onpointerdown={onGrab}
    onpointermove={onDrag}
    onpointerup={onDrop}
    onpointercancel={onDrop}
  >
    <Mascot role="assistant" accent="amber" top="bar" face="happy" prop="none" shell="pastel" {pose} {turn} size={SIZE} sway />
    <!-- Decoration on the figure now that the figure itself is named: two
         nested things both announcing "ไกด์" is one more than a reader needs. -->
    <span
      class="rank-corner rank-guide guide-grip"
      title={t('guide.resize')}
      role="presentation"
      onpointerdown={onGripDown}
      onpointermove={onGripMove}
      onpointerup={onGripUp}
      onpointercancel={onGripUp}
    >
      <Icon name="compass" size={14} />
    </span>
  </span>

  <!-- The frame the companion wears, for the same reasons: a figure you can
       dismiss, silence, or open the settings of, without hunting for where
       those live (owner, 15 ก.ย. 2026: "ให้มันมีขอบเหมือนผู้ช่วยด้วยดิ แบบกด
       ปิดได้ ปิดเสียงได้ เพิ่มปุ่มฟันเฟือง"). -->
  <span class="guide-frame"></span>
  <button type="button" class="g-btn g-mute" class:off={!guidePrefs.voice}
          title={guidePrefs.voice ? t('guide.set.voiceOff') : t('guide.set.voiceOn')}
          aria-label={guidePrefs.voice ? t('guide.set.voiceOff') : t('guide.set.voiceOn')}
          aria-pressed={!guidePrefs.voice}
          onclick={() => { setGuidePref('voice', !guidePrefs.voice); if (!guidePrefs.voice) stopSpeechIf('guide') }}>
    <Icon name={guidePrefs.voice ? 'volume2' : 'volumeX'} size={11} />
  </button>
  <button type="button" class="g-btn g-hide" title={t('guide.close')} aria-label={t('guide.close')} onclick={onClose}>
    <Icon name="x" size={11} />
  </button>

  {#if guide.offering}
    <div class="say" bind:this={sayEl}
         data-settings-placement={settingsLayout.placement}
         data-settings-tab={settingsLayout.tabEdge}
         data-settings-align={settingsLayout.align}>
      <button type="button" class="say-settings-tab" class:on={face === 'settings'}
              title={t('guide.set.title')} aria-label={t('guide.set.title')}
              aria-haspopup="menu" aria-expanded={face === 'settings'}
              onclick={toggleSettings}>
        <Icon name="settings" size={12} />
      </button>
      {#if face === 'settings'}
        <div class="guide-settings-menu" style="--panel-max-h: {settingsLayout.maxHeight}px">
          <GuidePanel onClose={() => (face = 'talk')} onPick={(id) => { face = 'talk'; void guide.goTo(id) }} />
        </div>
      {/if}
      <div class="say-head">
        <div class="say-head-main">
          <span class="say-name">{t('rank.guide')}</span>
        </div>
        <div class="say-head-tools">
          <button type="button" class="say-head-btn close" onclick={onClose} aria-label={t('guide.close')}>
            <Icon name="x" size={12} />
          </button>
        </div>
      </div>
      <div class="say-body">{t('guide.offerPrompt')}</div>
      <div class="say-foot say-offer-foot">
        <button type="button" class="ctrl mini primary" onclick={() => guide.acceptOffer()}>
          <Icon name="check" size={11} />
          <span>{t('guide.offerYes')}</span>
        </button>
        <button type="button" class="ctrl mini" onclick={() => guide.declineOffer()}>{t('guide.offerNo')}</button>
      </div>
    </div>
  {:else if shown || typing || guide.asking || guide.transcript.length > 0}
    <div class="say" class:expanded class:collapsed bind:this={sayEl}
         data-settings-placement={settingsLayout.placement}
         data-settings-tab={settingsLayout.tabEdge}
         data-settings-align={settingsLayout.align}>
      <button type="button" class="say-settings-tab" class:on={face === 'settings'}
              title={t('guide.set.title')} aria-label={t('guide.set.title')}
              aria-haspopup="menu" aria-expanded={face === 'settings'}
              onclick={toggleSettings}>
        <Icon name="settings" size={12} />
      </button>
      {#if face === 'settings'}
        <div class="guide-settings-menu" style="--panel-max-h: {settingsLayout.maxHeight}px">
          <GuidePanel onClose={() => (face = 'talk')} onPick={(id) => { face = 'talk'; void guide.goTo(id) }} />
        </div>
      {/if}
      <div class="say-head" role="group" aria-label={stopName || t('rank.guide')} ondblclick={onBubbleHeadDoubleClick}>
        <div class="say-head-main">
          <span class="say-name">{stopName || t('rank.guide')}</span>
          {#if stop}
            <span class="say-tag" class:no={!stop.safe && !stop.confirmationRequired}>
              {stop.safe ? t('guide.canPress') : stop.confirmationRequired ? t('guide.confirmPress') : t('guide.cannotPress')}
            </span>
            {#if guide.brain === 'map'}<span class="say-map-badge">{t('guide.fromMap')}</span>{/if}
          {/if}
        </div>
        <div class="say-head-tools">
          {#if guide.transcript.length > 0}
            <button type="button" class="say-head-btn" title={t('guide.clearChat')} aria-label={t('guide.clearChat')} onclick={clearTranscript}>
              <Icon name="rotateCw" size={11} />
            </button>
          {/if}
          <button
            type="button"
            class="say-head-btn fold-control"
            title={collapsed ? t('guide.unfold') : t('guide.fold')}
            aria-label={collapsed ? t('guide.unfold') : t('guide.fold')}
            aria-expanded={!collapsed}
            onclick={toggleCollapsed}
          >
            <Icon name={collapsed ? 'chevronDown' : 'chevronUp'} size={13} />
          </button>
          {#if !collapsed}
          <button
            type="button"
            class="say-head-btn window-control"
            title={expanded ? t('guide.collapse') : t('guide.expand')}
            aria-label={expanded ? t('guide.collapse') : t('guide.expand')}
            aria-expanded={expanded}
            onclick={toggleExpanded}
          >
            <Icon name={expanded ? 'restore' : 'square'} size={13} />
          </button>
          {/if}
          <button type="button" class="say-head-btn close" onclick={onClose} aria-label={t('guide.close')}>
            <Icon name="x" size={14} />
          </button>
        </div>
      </div>

      {#if guide.transcript.length > 0}
        <div class="say-chat-stream" bind:this={chatStreamEl}>
          {#each guide.transcript as msg, i}
            <div class="say-chat-msg {msg.who}">
              {#if msg.who === 'user'}
                <div class="say-bubble user">{msg.text}</div>
              {:else}
                <div class="say-bubble guide markdown-body">
                  {@html renderMarkdown(msg.text)}
                </div>
              {/if}
            </div>
          {/each}
          {#if guide.asking && guide.streamingText}
            <div class="say-chat-msg guide">
              <div class="say-bubble guide markdown-body">
                {@html renderMarkdown(guide.streamingText)}<span class="cur"></span>
              </div>
            </div>
          {:else if guide.asking}
            <div class="say-chat-msg guide">
              <div class="say-bubble guide thinking">
                <span class="cur"></span>
              </div>
            </div>
          {/if}
        </div>
      {:else}
        <div class="say-body markdown-body">
          {@html renderMarkdown(shown)}{#if typing || guide.asking}<span class="cur"></span>{/if}
        </div>

        {#if guide.welcoming && canOfferActions}
          <button type="button" class="say-shift-tip" onclick={explainShiftClick}>
            <Icon name="pointer" size={14} />
            <span class="say-keycap">Shift</span>
            <span>{t('guide.shiftClickButton' as never)}</span>
          </button>
        {/if}

        {#if relatedConcept && canOfferActions}
          <div class="say-concept-box">
            <button type="button" class="ctrl mini concept-toggle" onclick={() => (showConcept = !showConcept)}>
              💡 {guideText(relatedConcept.id, 'name')}
            </button>
            {#if showConcept}
              <div class="say-concept-body">
                {guideText(relatedConcept.id, 'what')}
                {#if guideText(relatedConcept.id, 'why')}
                  <div class="say-concept-why">{guideText(relatedConcept.id, 'why')}</div>
                {/if}
              </div>
            {/if}
          </div>
        {/if}

      {/if}

      <!-- A reply must always leave a useful next move. This deliberately sits
           outside the transcript/single-message fork: asking a question
           should not make the prepared routes disappear. -->
      {#if !guide.stopId && !guide.heading && !guide.headingPage && canOfferActions}
        {@const contextStarts = contextualGuideStarts(currentPage(), cockpit.desk === 'coding' ? 'coding' : 'assistant')}
        {@const firstChoices = landingChoices(contextStarts)}
        {@const categories = offeredWalkCategories()}
        {@const selectedCategory = categories.find((entry) => entry.id === walkCategory)}
        {@const walks = walkCategory ? offeredWalks(walkCategory, activeGuideDesk()) : []}
        <div class="say-command-deck say-recommended-routes">
          <div class="say-command-head">
            {#if walkCategory && firstChoices.length === 0}
              <button type="button" class="say-command-back" onclick={leaveWalkCategory}>
                <Icon name="chevronLeft" size={11} />
                <span>{t('guide.walkCategory.back')}</span>
              </button>
              <span>{selectedCategory?.label}</span>
            {:else}
              <span>{t('guide.walkCategory.title')}</span>
            {/if}
            {#if firstChoices.length > START_CHOICES_PER_PAGE || (firstChoices.length === 0 && walkCategory && walks.length > START_CHOICES_PER_PAGE)}
              <button type="button" class="say-command-rotate" title={t('guide.command.rotate')} aria-label={t('guide.command.rotate')}
                      onclick={() => (walkPage += 1)}>
                <Icon name="rotateCw" size={12} />
              </button>
            {/if}
          </div>
          <div class="say-command-list" class:say-start-grid={firstChoices.length >= 4 || walks.length >= 4}>
            {#if firstChoices.length > 0}
              {#each pageOf(firstChoices, walkPage, START_CHOICES_PER_PAGE) as choice (choice.kind + ':' + choice.id)}
                <button type="button" class="say-walk-chip say-command ctrl mini" onclick={() => startContextChoice(choice)}>
                  <span class="say-walk-dot"></span>
                  <span>{choice.label}</span>
                </button>
              {/each}
            {:else if walkCategory}
              {#each pageOf(walks, walkPage, START_CHOICES_PER_PAGE) as w (w.id)}
                <button type="button" class="say-walk-chip say-command ctrl mini" onclick={() => startWalk(w.id)}>
                  <span class="say-walk-dot"></span>
                  <span>{w.label}</span>
                </button>
              {/each}
            {:else}
              {#each categories as category (category.id)}
                <button type="button" class="say-walk-chip say-command say-category ctrl mini" onclick={() => chooseWalkCategory(category.id)}>
                  <span class="say-walk-dot"></span>
                  <span>{category.label}</span>
                  <Icon name="chevronRight" size={12} />
                </button>
              {/each}
            {/if}
          </div>
        </div>
      {/if}

      <!-- Route controls stay visible in both reading modes. A tour often
           starts from a typed question, which creates a transcript; hiding
           the next real button at that moment made the tour text-only. -->
      {#if guide.stopId && canOfferActions}
        {#if guide.awaiting}
          <div class="say-step">
            <Icon name="pointer" size={13} />
            <span>{stopName ? t('guide.stepPressNamed', { name: stopName }) : t('guide.stepPress')}</span>
            {#if guide.stepsLeft > 1}<span class="say-step-left">{t('guide.stepLeft', { n: String(guide.stepsLeft) })}</span>{/if}
          </div>
        {/if}
        {#if !guide.route}
          {@const actions = stepDeckActions()}
          <div class="say-command-deck say-step-actions">
            <div class="say-command-head">
              <span>{t('guide.command.now')}</span>
              {#if actions.length > STEP_COMMANDS_PER_PAGE}
                <button type="button" class="say-command-rotate" title={t('guide.command.rotate')} aria-label={t('guide.command.rotate')}
                        onclick={() => (stepCommandPage += 1)}>
                  <Icon name="rotateCw" size={12} />
                </button>
              {/if}
            </div>
            <div class="say-command-list">
              {#each pageOf(actions, stepCommandPage, STEP_COMMANDS_PER_PAGE) as action (action.id)}
                {#if action.kind === 'command'}
                  <button
                    type="button"
                    class="say-walk-chip say-command ctrl mini"
                    class:say-take={action.command === 'take'}
                    disabled={action.command === 'take' && guide.escorting}
                    onpointerdown={(e) => e.stopPropagation()}
                    onclick={(e) => { e.preventDefault(); e.stopImmediatePropagation(); runStepCommand(action.command) }}
                  >
                    <span class="say-walk-dot"></span>
                    <span>{stepCommandLabel(action.command)}</span>
                  </button>
                {:else}
                  <button
                    type="button"
                    class="say-walk-chip say-command say-other-topic ctrl mini"
                    onpointerdown={(e) => e.stopPropagation()}
                    onclick={(e) => { e.preventDefault(); e.stopImmediatePropagation(); startWalk(action.topic.id) }}
                  >
                    <span class="say-walk-dot"></span>
                    <span>{action.topic.label}</span>
                  </button>
                {/if}
              {/each}
            </div>
          </div>
        {/if}
      {/if}

      {#if guide.route && guide.stopId && canOfferActions}
        {@const activeRoute = GUIDE_ROUTES[guide.route.id]}
        <div class="say-route-guide">
          <div class="say-route-progress">
            {t('guide.routeProgress' as never, {
              current: String(guide.route.at + 1),
              total: String(activeRoute.stops.length),
              route: activeRoute.name,
            })}
          </div>
          <div class="say-walks say-route-actions">
            {#if guide.route.at > 0 && !guide.awaiting}
              <button type="button" class="say-walk-chip ctrl mini" onclick={() => guide.prev()}>
                <Icon name="chevronLeft" size={11} />
                <span>{t('guide.prev')}</span>
              </button>
            {/if}
            {#if guide.awaiting && guide.canTakeCurrent()}
              <button
                type="button"
                class="say-walk-chip say-take ctrl mini primary"
                disabled={guide.escorting}
                onpointerdown={(e) => e.stopPropagation()}
                onclick={(e) => { e.preventDefault(); e.stopImmediatePropagation(); guide.takeMe() }}
              >
                <Icon name="pointer" size={11} />
                <span>{guide.escorting ? t('guide.takingYou') : t('guide.takeMe')}</span>
              </button>
            {:else if !guide.awaiting}
              <button type="button" class="say-walk-chip ctrl mini primary" onclick={() => guide.next()}>
                <span class="say-walk-dot"></span>
                <span>{guide.route.at >= activeRoute.stops.length - 1 ? t('guide.finishTour' as never) : t('guide.next')}</span>
              </button>
            {/if}
          </div>
        </div>
      {/if}

      <form class="say-ask" onsubmit={onAskSubmit}>
        <input id="guide-ask" bind:value={askInput} placeholder={t('guide.askPlaceholder')} disabled={guide.asking} />
        <button type="submit" class="ctrl mini primary" aria-label={t('guide.ask')} disabled={guide.asking || !askInput.trim()}>
          <Icon name="sendHorizontal" size={12} />
        </button>
      </form>
    </div>
  {/if}
</div>

<style>
  /* No cursor override: the app is fully usable while the guide is up, and a
     help cursor over a button that really does press would be a lie. */

  /* the pill: top centre, clear of the top bar's ends, where nothing sits */
  .guide-banner {
    position: fixed;
    top: 6px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 10001;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 4px 6px 4px 12px;
    border-radius: 999px;
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    border: 1px solid var(--badge-amber-border);
    box-shadow: 0 4px 14px rgba(0, 0, 0, 0.25);
    font-size: var(--fs-xs);
    font-weight: 600;
    white-space: nowrap;
  }
  .guide-banner-close {
    border: 0;
    background: transparent;
    color: currentColor;
    cursor: pointer;
    display: grid;
    place-items: center;
    padding: 3px;
    border-radius: 50%;
  }
  .guide-banner-close:hover {
    background: rgba(255, 255, 255, 0.12);
  }

  /* This box is only an invisible anchor. The arrow points at the target's
     centre; there is intentionally no outline whose size could drift when a
     responsive layout changes under the guide. */
  .guide-target-anchor {
    position: fixed;
    z-index: 9998;
    width: 0;
    height: 0;
    box-sizing: border-box;
    pointer-events: none;
  }
  .guide-target-label {
    position: absolute;
    left: 50%;
    bottom: calc(100% + 3px);
    transform: translateX(-50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
    width: max-content;
    max-width: 180px;
  }
  .guide-target-text {
    display: inline-flex;
    align-items: center;
    padding: 6px 9px;
    border: 1px solid var(--badge-amber-border);
    border-radius: var(--r-md);
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    box-shadow: 0 5px 18px rgba(0, 0, 0, .4);
    font-size: var(--fs-md);
    font-weight: 700;
    line-height: 1.2;
    white-space: nowrap;
  }
  .guide-target-arrow {
    display: grid;
    place-items: center;
    color: var(--badge-amber-text);
    filter: drop-shadow(0 2px 4px rgba(0, 0, 0, .7));
    animation: guide-target-point-down 1s ease-in-out infinite;
  }
  .guide-target-anchor.label-below .guide-target-label {
    top: calc(100% + 3px);
    bottom: auto;
    flex-direction: column-reverse;
  }
  .guide-target-anchor.label-below .guide-target-arrow {
    animation-name: guide-target-point-up;
  }
  .guide-target-anchor.label-left .guide-target-label {
    left: 0;
    transform: none;
  }
  .guide-target-anchor.label-right .guide-target-label {
    left: auto;
    right: 0;
    transform: none;
  }
  @keyframes guide-target-point-down {
    50% { transform: translateY(3px); }
  }
  @keyframes guide-target-point-up {
    0%, 100% { transform: rotate(180deg); }
    50% { transform: rotate(180deg) translateY(3px); }
  }

  .guide-mascot-wrap {
    position: fixed;
    left: 0;
    top: 0;
    z-index: 10000;
    transition: transform var(--dur-walk, 800ms) cubic-bezier(.4, .1, .2, 1);
    will-change: transform;
    /* Arriving is a fade on the spot, never a journey: the figure is already
       where it should be on its first frame, so nothing about the entrance
       needs to move it. OPACITY ONLY here — this element's transform is the
       inline one that says where it stands, and an animation touching
       transform would override it and snap the figure to the corner, which is
       the exact bug this replaced. The scale belongs to the child. */
    animation: guide-arrive var(--dur-arrive, 300ms) ease-out;
    pointer-events: auto;
  }
  .guide-mascot-wrap.walking {
    transition-timing-function: cubic-bezier(.35, .05, .3, 1);
  }
  /* Under a hand there is no easing at all: the walk's 800ms transition would
     drag the figure along behind the pointer, which reads as lag rather than
     as weight. The transition comes back on release, for the next journey. */
  .guide-mascot-wrap.dragging {
    transition: none;
  }
  .guide-mascot-wrap.resizing {
    transition: none;
  }
  .guide-mascot-wrap.dragging .ranked-guide {
    cursor: grabbing;
  }

  @keyframes guide-arrive {
    from { opacity: 0; }
  }
  @keyframes guide-arrive-pop {
    from { transform: scale(.88); }
  }

  @media (prefers-reduced-motion: reduce) {
    .guide-mascot-wrap {
      transition: none;
      animation: none;
    }
    .ranked-guide {
      animation: none;
    }
    .guide-target-arrow {
      animation: none;
    }
  }

  .ranked-guide {
    animation: guide-arrive-pop var(--dur-arrive, 300ms) cubic-bezier(.2, .9, .3, 1.2);
    position: relative;
    display: inline-block;
    line-height: 0;
    cursor: grab;
    /* The pointer stream is the whole mechanism, and on a touch screen the
       browser hands the gesture to the scroller instead unless this says not
       to. */
    touch-action: none;
  }
  .ranked-guide :global(.mascot) {
    display: block;
  }
  .rank-corner {
    position: absolute;
    right: -6px;
    bottom: -2px;
    width: 26px;
    height: 26px;
    border-radius: 50%;
    box-sizing: border-box;
    display: grid;
    place-items: center;
    border: 2px solid var(--surface-panel);
  }
  .rank-guide {
    color: var(--badge-amber-text);
    background: var(--badge-amber-bg);
    outline: 1px solid var(--badge-amber-border);
  }
  .guide-grip {
    cursor: nwse-resize;
    touch-action: none;
    transition: box-shadow .15s, transform .15s;
  }
  .guide-grip:hover,
  .guide-mascot-wrap.resizing .guide-grip {
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--badge-amber-border) 45%, transparent);
    transform: scale(1.06);
  }

  /* The frame and its three buttons — the companion's own affordance, at the
     guide's size. Visible on hover or focus, and the mute stays lit when it is
     off so a silent figure says it is silent. */
  .guide-frame {
    position: absolute; inset: -6px; border: 1px dashed var(--border-subtle);
    border-radius: 14px; opacity: 0; transition: opacity .15s; pointer-events: none;
  }
  .g-btn {
    position: absolute; width: 22px; height: 22px; border-radius: 50%;
    border: 1px solid var(--border-subtle); background: var(--surface-raised);
    color: var(--text-muted); display: grid; place-items: center; padding: 0;
    cursor: pointer; opacity: 0; transition: opacity .15s, color .15s;
  }
  .g-btn:hover { color: var(--text-primary); }
  .g-hide { top: -12px; right: -12px; }
  .g-mute { top: -12px; left: -12px; }
  .g-mute.off { opacity: 1; }
  .guide-mascot-wrap:hover .guide-frame,
  .guide-mascot-wrap:hover .g-btn,
  .guide-mascot-wrap.resizing .guide-frame,
  .g-btn:focus-visible { opacity: 1; }
  .guide-mascot-wrap.dragging .guide-frame,
  .guide-mascot-wrap.dragging .g-btn { opacity: 0; }

  .say {
    position: absolute;
    right: calc(100% + 14px);
    bottom: 24px;
    width: var(--guide-bubble-width, 420px);
    max-width: min(var(--guide-bubble-width, 420px), calc(100vw - 32px));
    background: color-mix(in srgb, var(--surface-raised) 94%, var(--surface-panel));
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    color: var(--text-primary);
    border: 1px solid var(--border-default);
    border-radius: 16px;
    padding: 12px 14px;
    font-size: var(--fs-lg);
    line-height: 1.55;
    box-shadow: 0 10px 32px rgba(0, 0, 0, 0.4);
    animation: say-in 180ms cubic-bezier(0.16, 1, 0.3, 1);
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
  }
  .say::after {
    content: '';
    position: absolute;
    right: -6px;
    bottom: 16px;
    width: 12px;
    height: 12px;
    background: var(--surface-raised);
    border-right: 1px solid var(--border-subtle);
    border-top: 1px solid var(--border-subtle);
    transform: rotate(45deg);
    border-radius: 2px;
  }
  .flip .say {
    right: auto;
    left: calc(100% + 14px);
  }
  .flip .say::after {
    right: auto;
    left: -6px;
    border-right: 0;
    border-top: 0;
    border-left: 1px solid var(--border-subtle);
    border-bottom: 1px solid var(--border-subtle);
  }
  .below .say {
    bottom: auto;
    top: 4px;
  }
  .below .say::after {
    bottom: auto;
    top: 22px;
  }
  .docked .say {
    /* The parent is transformed to the mascot's screen position, so fixed
       descendants are fixed to that transformed box, not the viewport. Put
       the dock in viewport coordinates by subtracting the parent's offset. */
    position: absolute;
    /* A target as wide as the composer cannot fit a bubble on either side.
       Keep the ordinary readable chat width and centre it in the viewport;
       the former full-width dock looked like a second composer laid over the
       real one. */
    left: calc(50vw - var(--guide-x));
    right: auto;
    width: min(var(--guide-bubble-width, 420px), calc(100vw - 24px));
    max-width: calc(100vw - 24px);
    max-height: calc(100vh - 68px);
    overflow-y: auto;
    border-radius: 16px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.45);
    z-index: 10000;
    animation: none;
  }
  .docked-bottom .say {
    top: calc(100vh - 12px - var(--guide-y));
    bottom: auto;
    transform: translate(-50%, -100%);
  }
  .docked-top .say {
    top: calc(56px - var(--guide-y));
    bottom: auto;
    transform: translateX(-50%);
  }
  .docked .say::after {
    display: none;
  }
  /* A long guide conversation can become a real reading surface without
     turning the mascot into a separate screen. The parent itself is moved by
     transform, so viewport coordinates are translated back through --guide-x
     and --guide-y rather than relying on position:fixed inside that transform. */
  .guide-mascot-wrap .say.expanded {
    left: calc(50vw - var(--guide-x));
    right: auto;
    top: calc(48px - var(--guide-y));
    bottom: auto;
    width: min(760px, calc(100vw - 24px));
    max-width: calc(100vw - 24px);
    height: min(680px, calc(100vh - 64px));
    max-height: calc(100vh - 64px);
    transform: translateX(-50%);
    animation: none;
    z-index: 10002;
  }
  .guide-mascot-wrap .say.expanded::after {
    display: none;
  }
  /* Folding is intentionally different from the square maximise button:
     only the title bar remains, so the guide stops covering the work while
     its current route and conversation stay alive. */
  .guide-mascot-wrap .say.collapsed {
    width: min(300px, calc(100vw - 24px));
    min-height: 0;
    height: auto;
    max-height: none;
    overflow: visible;
    padding: 9px 10px;
  }
  .say.collapsed > :not(.say-head) {
    display: none !important;
  }
  .say.collapsed .say-head {
    margin-bottom: 0;
  }
  .say.collapsed .say-tag,
  .say.collapsed .say-map-badge {
    display: none;
  }
  .say-concept-box {
    margin-top: 6px;
    padding-top: 6px;
    border-top: 1px dashed var(--border-subtle);
  }
  .concept-toggle {
    background: var(--surface-sunken);
    font-size: 0.85em;
  }
  .say-concept-body {
    margin-top: 4px;
    font-size: var(--fs-2xs);
    background: var(--surface-sunken);
    padding: 6px 8px;
    border-radius: 6px;
    color: var(--text-secondary);
    line-height: 1.4;
  }
  .say-concept-why {
    margin-top: 4px;
    color: var(--badge-amber-text);
  }
  @keyframes say-in {
    from {
      opacity: 0;
      transform: translateY(4px) scale(0.97);
    }
    to {
      opacity: 1;
      transform: none;
    }
  }

  /* Settings belong to the conversation surface, not the mascot frame. The
     round tab sits just outside the bubble and opens a separate attached
     window; the conversation remains mounted and visible underneath. */
  .say-settings-tab {
    position: absolute;
    z-index: 4;
    width: 24px;
    height: 24px;
    padding: 0;
    display: grid;
    place-items: center;
    border-radius: 50%;
    border: 1px solid var(--border-default);
    background: var(--surface-raised);
    color: var(--text-muted);
    box-shadow: 0 3px 10px rgba(0, 0, 0, .28);
    cursor: pointer;
  }
  .say[data-settings-tab='left'] > .say-settings-tab {
    left: -12px;
    right: auto;
    bottom: 18px;
    top: auto;
  }
  .say[data-settings-tab='right'] > .say-settings-tab {
    left: auto;
    right: -12px;
    bottom: 18px;
    top: auto;
  }
  .say[data-settings-tab='top'] > .say-settings-tab {
    left: auto;
    right: 18px;
    top: -12px;
    bottom: auto;
  }
  .say[data-settings-tab='bottom'] > .say-settings-tab {
    left: auto;
    right: 18px;
    top: auto;
    bottom: -12px;
  }
  .say-settings-tab:hover,
  .say-settings-tab.on {
    color: var(--badge-amber-text);
    border-color: var(--badge-amber-border);
  }
  .guide-settings-menu {
    position: absolute;
    width: min(292px, calc(100vw - 24px));
    max-height: var(--panel-max-h, min(62vh, 460px));
    overflow-y: auto;
    box-sizing: border-box;
    padding: 11px;
    border: 1px solid var(--border-default);
    border-radius: 13px;
    background: color-mix(in srgb, var(--surface-raised) 96%, var(--surface-panel));
    color: var(--text-primary);
    box-shadow: 0 12px 30px rgba(0, 0, 0, .38);
    animation: say-in 160ms cubic-bezier(.16, 1, .3, 1);
    z-index: 3;
  }
  .say[data-settings-placement='right'] > .guide-settings-menu {
    left: calc(100% + 10px);
    right: auto;
  }
  .say[data-settings-placement='left'] > .guide-settings-menu {
    left: auto;
    right: calc(100% + 10px);
  }
  .say[data-settings-placement='left'][data-settings-align='start'] > .guide-settings-menu,
  .say[data-settings-placement='right'][data-settings-align='start'] > .guide-settings-menu {
    top: 0;
    bottom: auto;
  }
  .say[data-settings-placement='left'][data-settings-align='end'] > .guide-settings-menu,
  .say[data-settings-placement='right'][data-settings-align='end'] > .guide-settings-menu {
    top: auto;
    bottom: 0;
  }
  .say[data-settings-placement='above'] > .guide-settings-menu {
    left: 0;
    right: auto;
    top: auto;
    bottom: calc(100% + 10px);
    width: min(292px, 100%);
  }
  .say[data-settings-placement='below'] > .guide-settings-menu {
    left: 0;
    right: auto;
    top: calc(100% + 10px);
    bottom: auto;
    width: min(292px, 100%);
  }
  .say[data-settings-placement='inside'] > .guide-settings-menu {
    left: auto;
    right: 10px;
    top: 44px;
    bottom: auto;
    width: min(292px, calc(100% - 20px));
  }

  .say-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 8px;
  }
  .say-head-main {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    flex: 1;
  }
  .say-head-tools {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .say-head-btn {
    background: transparent;
    border: 1px solid transparent;
    border-radius: 6px;
    color: var(--text-muted);
    cursor: pointer;
    display: grid;
    place-items: center;
    padding: 0;
    width: 24px;
    height: 24px;
    transition: all .15s;
  }
  .say-head-btn:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
    border-color: var(--border-subtle);
  }
  .say-head-btn.close:hover {
    color: var(--accent-error, #f87171);
  }
  .say-name {
    font-weight: 700;
  }
  .say-tag {
    font-size: var(--fs-2xs);
    border-radius: 999px;
    padding: 1px 8px;
    background: var(--badge-cyan-bg);
    color: var(--badge-cyan-text);
    border: 1px solid var(--badge-cyan-border);
  }
  .say-tag.no {
    background: var(--surface-sunken);
    color: var(--text-muted);
    border-color: var(--border-subtle);
  }
  .say-map-badge {
    margin-left: auto;
    font-size: var(--fs-2xs);
    color: var(--text-dim);
  }

  /* Chat conversation stream */
  .say-chat-stream {
    max-height: 280px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 2px 0 6px;
    margin-bottom: 6px;
    overscroll-behavior: contain;
  }
  .say.expanded .say-chat-stream {
    flex: 1;
    min-height: 0;
    max-height: none;
  }
  .say-chat-msg {
    display: flex;
    flex-direction: column;
    max-width: 100%;
  }
  .say-chat-msg.user {
    align-items: flex-end;
  }
  .say-chat-msg.guide {
    align-items: flex-start;
  }
  .say-bubble.user {
    background: var(--surface-sunken);
    color: var(--text-primary);
    border: 1px solid var(--border-subtle);
    border-radius: 12px 12px 2px 12px;
    padding: 6px 10px;
    font-size: var(--fs-lg);
    max-width: 88%;
    word-break: break-word;
  }
  .say-bubble.guide {
    background: transparent;
    color: var(--text-primary);
    padding: 2px 0;
    font-size: var(--fs-lg);
    line-height: 1.5;
    max-width: 100%;
  }
  .say-bubble.guide.markdown-body :global(p),
  .say-body.markdown-body :global(p) {
    margin: 0 0 0.5em;
  }
  .say-bubble.guide.markdown-body :global(p:last-child),
  .say-body.markdown-body :global(p:last-child) {
    margin-bottom: 0;
  }
  .say-bubble.guide.markdown-body :global(code),
  .say-body.markdown-body :global(code) {
    display: inline-flex;
    align-items: center;
    min-height: 19px;
    padding: 0 6px;
    border: 1px solid var(--border-default);
    border-radius: 6px;
    background: var(--surface-hover);
    color: var(--text-primary);
    box-shadow: inset 0 -1px 0 color-mix(in srgb, var(--text-muted) 28%, transparent);
    font: 600 var(--fs-2xs)/1 var(--sans);
    vertical-align: 1px;
  }
  .say-body {
    margin-bottom: 6px;
    max-height: 280px;
    overflow-y: auto;
  }

  .say-shift-tip {
    width: 100%;
    min-height: 42px;
    margin-top: 4px;
    padding: 8px 10px;
    display: flex;
    align-items: center;
    gap: 8px;
    border: 1px solid var(--badge-amber-border);
    border-radius: 9px;
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    font: 600 var(--fs-md)/1.35 var(--sans);
    text-align: left;
    cursor: pointer;
  }
  .say-shift-tip:hover {
    border-color: color-mix(in srgb, var(--badge-amber-text) 60%, var(--badge-amber-border));
    background: color-mix(in srgb, var(--badge-amber-bg) 84%, var(--badge-amber-text));
  }
  .say-keycap {
    flex: 0 0 auto;
    min-width: 40px;
    padding: 3px 7px;
    border: 1px solid currentColor;
    border-radius: 6px;
    background: color-mix(in srgb, var(--surface-raised) 72%, transparent);
    box-shadow: 0 2px 0 color-mix(in srgb, currentColor 35%, transparent);
    text-align: center;
    font-size: var(--fs-2xs);
    letter-spacing: .02em;
  }

  /* Being led somewhere: the instruction reads as an instruction, in the
     guide's own colour, and sits above the map's explanation rather than
     inside it — one is what to do now, the other is what the thing is. */
  .say-step {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    margin-top: 8px;
    padding: 6px 9px;
    border-radius: 9px;
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    border: 1px solid var(--badge-amber-border);
    font-size: var(--fs-lg);
    font-weight: 600;
  }
  .say-step-left {
    margin-left: auto;
    font-weight: 500;
    opacity: 0.75;
  }
  .say-take {
    white-space: nowrap;
  }
  .say-step-actions {
    margin-top: 6px;
  }
  .say-command-deck {
    width: 100%;
    margin-top: 10px;
    padding-top: 8px;
    border-top: 1px solid var(--border-subtle);
  }
  .say-command-head {
    min-height: 24px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 6px;
    color: var(--text-secondary);
    font-size: var(--fs-md);
    font-weight: 600;
  }
  .say-command-rotate {
    width: 24px;
    height: 24px;
    padding: 0;
    display: grid;
    place-items: center;
    border: 1px solid var(--border-subtle);
    border-radius: 7px;
    background: var(--surface-sunken);
    color: var(--text-muted);
    cursor: pointer;
  }
  .say-command-rotate:hover {
    color: var(--badge-amber-text);
    border-color: var(--badge-amber-border);
  }
  .say-command-back {
    min-height: 24px;
    padding: 0 7px 0 4px;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    border: 1px solid var(--border-subtle);
    border-radius: 7px;
    background: var(--surface-sunken);
    color: var(--text-muted);
    font: inherit;
    cursor: pointer;
  }
  .say-command-back:hover {
    color: var(--text-primary);
    border-color: var(--border-default);
  }
  .say-command-list {
    display: grid;
    grid-template-columns: 1fr;
    gap: 6px;
    width: 100%;
  }
  .say-command-list.say-start-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .say-start-grid .say-command.ctrl.mini {
    min-height: 52px;
    padding: 8px 10px;
  }
  .say-command.ctrl.mini {
    width: 100%;
    min-height: 42px;
    padding: 9px 12px;
    justify-content: flex-start;
    text-align: left;
    white-space: normal;
    line-height: 1.35;
    font-size: var(--fs-lg);
    color: var(--text-primary);
    border-color: var(--border-default);
  }
  .say-walks {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 10px;
  }
  .say-route-guide {
    width: 100%;
    margin-top: 10px;
    padding-top: 9px;
    border-top: 1px solid var(--border-subtle);
  }
  .say-route-progress {
    color: var(--text-muted);
    font-size: var(--fs-xs);
    font-weight: 600;
  }
  .say-route-actions {
    margin-top: 7px;
  }

  .say-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-top: 10px;
  }
  .say-foot-nav {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
  }
  .say-offer-foot {
    justify-content: flex-end;
  }
  .say-walk-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 5px 10px;
    border-radius: 8px;
  }
  .say-walk-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--text-dim);
    transition: background 0.15s;
  }
  .say-walk-chip:hover .say-walk-dot {
    background: var(--badge-amber-text);
  }
  .say-category :global(svg) {
    margin-left: auto;
    color: var(--text-dim);
  }

  .say-ask {
    display: flex;
    gap: 6px;
    margin-top: 10px;
    padding-top: 8px;
    border-top: 1px solid var(--border-subtle);
  }
  .say-ask input {
    flex: 1;
    background: var(--surface-sunken);
    border: 1px solid var(--border-subtle);
    border-radius: 6px;
    color: var(--text-primary);
    padding: 4px 8px;
    font: inherit;
    font-size: var(--fs-lg);
  }
  .say-ask input:focus {
    outline: none;
    border-color: var(--badge-amber-border);
  }

  @media (max-width: 560px) {
    .say-command-list.say-start-grid {
      grid-template-columns: 1fr;
    }
  }

  .ctrl.mini {
    padding: 4px 10px;
    font-size: var(--fs-xs);
    font-weight: 500;
    border-radius: 8px;
    border: 1px solid var(--border-subtle);
    background: var(--surface-sunken);
    color: var(--text-secondary);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    justify-content: center;
    transition: all 0.15s cubic-bezier(0.16, 1, 0.3, 1);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }
  .ctrl.mini:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
    border-color: var(--border-default);
    transform: translateY(-1px);
    box-shadow: 0 3px 8px rgba(0, 0, 0, 0.2);
  }
  .ctrl.mini:active {
    transform: translateY(0);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }
  .ctrl.mini.primary {
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    border-color: var(--badge-amber-border);
    font-weight: 600;
  }
  .ctrl.mini.primary:hover {
    background: color-mix(in srgb, var(--badge-amber-bg) 85%, var(--badge-amber-text));
    border-color: var(--badge-amber-text);
    box-shadow: 0 3px 10px color-mix(in srgb, var(--badge-amber-border) 45%, transparent);
  }

  .cur {
    display: inline-block;
    width: 1px;
    height: 1em;
    background: var(--badge-amber-text);
    vertical-align: -2px;
    margin-left: 1px;
    animation: cur 1s steps(2) infinite;
  }
  @keyframes cur {
    50% {
      opacity: 0;
    }
  }
</style>
