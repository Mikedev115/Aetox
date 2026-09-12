<script lang="ts">
  // The assistant, sitting on the screen.
  //
  // One mascot in one fixed layer over the whole window — not a row in the
  // chat, not a hero on the welcome screen (the owner, 12 ก.ย.: "ไม่อยากให้มัน
  // แสดงมั่วแบบนี้ … ลากไปลากมาได้ ไม่ยึดติด อยู่บนจอเรา"). It goes where the
  // user drags it, remembers the spot, and stays inside the window. It does
  // not turn to follow the pointer; it sways, breathes and blinks in place.
  //
  // What it does is presence's word (mascot/presence.ts) read off the cockpit's
  // live state — the on-screen chat's turn — so a tool that runs is a card
  // beside its head and nothing more. What it SAYS is the bubble: the model's
  // own narration between tools, the tail of the answer as it streams, or the
  // question it is blocked on. Never a command, never a phrase of ours.
  //
  // Cost: one SVG drawn once per pose, CSS for everything that moves. Nothing
  // here runs per frame.
  import Mascot from './Mascot.svelte'
  import Icon from '../Icon.svelte'
  import { presenceOf, reportOf, walkTurn } from './presence'
  import { POSE, type PoseId } from './poses'
  import { cockpit } from '../stores/cockpit.svelte'
  import { setCompanionOn } from './companionSetting.svelte'
  import { avatarPrefs, assistantOptions } from './avatarPrefs.svelte'
  import { voice } from './voice.svelte'

  const SIZE = 104
  const MARGIN = 8
  const POS_KEY = 'companionPos'
  /** How long the success card stays after a turn ends. */
  const SUCCESS_MS = 2200
  /** Characters typed per tick, and the tick, when a report arrives. */
  const TYPE_STEP = 2
  const TYPE_MS = 28
  /** A click is a click if the pointer moved less than this; more is a drag. */
  const CLICK_PX = 4
  /** How long a reaction to a click lasts, and which it may be. */
  const REACT_MS = 1600
  const REACTIONS = ['greeting', 'cheer', 'helping', 'wink'] as const
  /** A wave on arrival — the app opened, or the switch was turned on. */
  const HELLO_MS = 1800
  /** Left alone this long with nothing to do, it goes to its charger. */
  const DOZE_MS = 5 * 60_000

  // ---- where it sits ------------------------------------------------------
  // Dragged, it walks: it faces the way it is going and it follows the hand
  // a little behind it, the way a thing on legs would, rather than being
  // pinned under the cursor. The owner's first look at it was "เร็ว เหวี่ยง" —
  // it whipped round on every wobble of the hand — so the direction is read
  // off a smoothed velocity, not the last two events, and the head may turn
  // no faster than TURN_DEG_S; the body eases to where the hand is with
  // FOLLOW per frame. Both loops run only while a drag is in progress.
  //
  // While it walks the head is driven here, every event, with the CSS
  // easing off (`snap`): the heading starts from wherever the head is at
  // the press and is stepped from there, so a press never spins it to a
  // stale direction ("ทำไมตอนกดจะย้ายมันหมุน"), and a heading that wound up
  // past a full circle on a looping drag is rewound unseen on release, a
  // frame before the easing comes back to turn it home.
  let pos = $state(seed())
  let dragging = $state(false)
  let drag: {
    dx: number; dy: number; x0: number; y0: number; moved: boolean
    lx: number; ly: number; vx: number; vy: number; at: number
    target: { x: number; y: number }; raf: number
  } | null = null
  /** Which way it walks while dragged, in the head's own degrees — right,
   *  left, away up the screen, or towards the viewer down it. */
  let heading = $state(0)
  /** Degrees per second the head may turn while walking. */
  const TURN_DEG_S = 240
  /** How much of the remaining distance to the hand is closed per frame. */
  const FOLLOW = 0.28
  /** Below this smoothed speed (px per event) the direction is kept. */
  const STILL_PX = 1.2
  /** No movement for this long while held is standing, not walking: it
   *  floats in place, facing the way it was going (owner: "คลิกค้างอยู่ที่เดิม
   *  ควรจะลอยอยู่เฉย ๆ"). */
  const STOP_MS = 140
  let moving = $state(false)
  let moveTimer: ReturnType<typeof setTimeout> | undefined

  function seed(): { x: number; y: number } {
    try {
      const raw = localStorage.getItem(POS_KEY)
      if (raw) {
        const p = JSON.parse(raw) as { x: number; y: number }
        if (Number.isFinite(p.x) && Number.isFinite(p.y)) return p
      }
    } catch {
      // No stored spot, or storage unavailable: the default below is the answer.
    }
    // Bottom-right, above the composer — the corner the eye rests in last.
    return { x: window.innerWidth - SIZE - 28, y: window.innerHeight - SIZE - 118 }
  }
  function clamp(p: { x: number; y: number }): { x: number; y: number } {
    return {
      x: Math.max(MARGIN, Math.min(window.innerWidth - SIZE - MARGIN, p.x)),
      y: Math.max(MARGIN, Math.min(window.innerHeight - SIZE - MARGIN, p.y)),
    }
  }
  function onDown(e: PointerEvent): void {
    if (e.button !== 0) return
    // Start walking from where the head already is.
    heading = POSE[pose as PoseId]?.turn ?? 0
    drag = {
      dx: e.clientX - pos.x, dy: e.clientY - pos.y, x0: e.clientX, y0: e.clientY, moved: false,
      lx: e.clientX, ly: e.clientY, vx: 0, vy: 0, at: performance.now(), target: { ...pos }, raf: 0,
    }
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }
  function onMove(e: PointerEvent): void {
    if (!drag) return
    if (!drag.moved && Math.hypot(e.clientX - drag.x0, e.clientY - drag.y0) < CLICK_PX) return
    drag.moved = true
    dragging = true
    moving = true
    clearTimeout(moveTimer)
    moveTimer = setTimeout(() => (moving = false), STOP_MS)
    // Smoothed velocity → the direction; the head turns towards it at a
    // bounded rate (the CSS transition on --t eases the steps between events).
    drag.vx = drag.vx * 0.7 + (e.clientX - drag.lx) * 0.3
    drag.vy = drag.vy * 0.7 + (e.clientY - drag.ly) * 0.3
    drag.lx = e.clientX
    drag.ly = e.clientY
    const now = performance.now()
    const dt = Math.min(0.05, (now - drag.at) / 1000)
    drag.at = now
    if (Math.hypot(drag.vx, drag.vy) >= STILL_PX) {
      const want = walkTurn(drag.vx, drag.vy, heading, 0)
      const step = TURN_DEG_S * dt
      heading += Math.max(-step, Math.min(step, want - heading))
    }
    drag.target = clamp({ x: e.clientX - drag.dx, y: e.clientY - drag.dy })
    if (!drag.raf) drag.raf = requestAnimationFrame(follow)
  }
  function follow(): void {
    if (!drag) return
    drag.raf = 0
    const { target } = drag
    const nx = pos.x + (target.x - pos.x) * FOLLOW
    const ny = pos.y + (target.y - pos.y) * FOLLOW
    if (Math.hypot(target.x - nx, target.y - ny) < 0.5) {
      pos = { ...target }
      return
    }
    pos = { x: nx, y: ny }
    drag.raf = requestAnimationFrame(follow)
  }
  function onUp(): void {
    if (!drag) return
    const wasClick = !drag.moved
    if (drag.raf) cancelAnimationFrame(drag.raf)
    if (drag.moved) pos = { ...drag.target }
    drag = null
    clearTimeout(moveTimer)
    moving = false
    if (wasClick) {
      dragging = false
      react()
      return
    }
    // Rewind whole circles now, while the head still snaps; next frame the
    // easing is back and the turn home is the short way round.
    heading = ((((heading + 180) % 360) + 360) % 360) - 180
    requestAnimationFrame(() => (dragging = false))
    try {
      localStorage.setItem(POS_KEY, JSON.stringify(pos))
    } catch {
      // A spot that could not be remembered is still the spot for this session.
    }
  }

  // ---- being clicked ------------------------------------------------------
  // A moment of one of the reactions, with a hop, then back to whatever it
  // was doing. No words: the bubble is for what the model says (presence.ts).
  let reaction = $state<string>('')
  let hop = $state(false)
  let reactTimer: ReturnType<typeof setTimeout> | undefined
  function react(): void {
    const next = REACTIONS[Math.floor(Math.random() * REACTIONS.length)]
    reaction = next === reaction ? REACTIONS[(REACTIONS.indexOf(next) + 1) % REACTIONS.length] : next
    hop = false
    requestAnimationFrame(() => (hop = true))
    clearTimeout(reactTimer)
    reactTimer = setTimeout(() => {
      reaction = ''
      hop = false
    }, REACT_MS)
  }
  $effect(() => () => {
    clearTimeout(reactTimer)
    clearTimeout(moveTimer)
  })

  // ---- the feed ------------------------------------------------------------
  // Everything above is reported to Go (companion.go, SetCompanionState) on
  // every change, so a surface outside this window — the transparent desktop
  // window to come — draws the same assistant from the same presence. Reached
  // through the runtime's own binding table rather than the generated
  // wailsjs import on purpose: those generated files carry other sessions'
  // uncommitted regeneration today; the import replaces this line when they
  // land. Outside Wails (tests, a plain browser) there is nothing to report to.
  type FeedState = { pose: string; report: string; on: boolean; prefs: { shell: string; accent: string; top: string; face: string } }
  const feed = (): ((s: FeedState) => Promise<void>) | undefined =>
    (window as unknown as { go?: { main?: { App?: { SetCompanionState?: (s: FeedState) => Promise<void> } } } }).go?.main?.App?.SetCompanionState
  $effect(() => {
    const send = feed()
    if (!send) return
    const s: FeedState = { pose, report, on: true, prefs: { shell: avatarPrefs.shell, accent: avatarPrefs.accent, top: avatarPrefs.top, face: avatarPrefs.face } }
    void send(s).catch(() => {})
  })
  $effect(() => () => {
    const send = feed()
    if (send) void send({ pose: 'idle', report: '', on: false, prefs: { shell: avatarPrefs.shell, accent: avatarPrefs.accent, top: avatarPrefs.top, face: avatarPrefs.face } }).catch(() => {})
  })
  $effect(() => {
    const keep = (): void => {
      pos = clamp(pos)
    }
    window.addEventListener('resize', keep)
    return () => window.removeEventListener('resize', keep)
  })
  // Near the left edge the bubble has no room on the left; it flips over.
  const flip = $derived(pos.x < 300)

  // ---- what it is doing ---------------------------------------------------
  const running = $derived(cockpit.toolSteps.filter((s) => s.state === 'run' && !s.parent).map((s) => s.name ?? ''))
  // A tool of this turn that failed, with nothing running after it yet.
  const failed = $derived(cockpit.awaitingReply && running.length === 0 && cockpit.toolSteps.some((s) => s.state === 'err' && !s.parent))
  const livePose = $derived(
    presenceOf({
      awaiting: cockpit.awaitingReply,
      status: cockpit.agentStatus,
      running,
      streaming: !!cockpit.streamingText,
      reasoning: !!cockpit.reasoningText,
      asking: !!cockpit.ask,
      mic: voice.mic,
      speaking: voice.speaking,
      failed,
    }),
  )
  // A turn that just ended earns a card for a moment, then rest: the check
  // when it answered, the alert when it did not (a stop is neither).
  let justDone = $state(false)
  let endedBadly = $state(false)
  let wasAwaiting = false
  $effect(() => {
    const now = cockpit.awaitingReply
    if (wasAwaiting && !now) {
      const last = cockpit.chat[cockpit.chat.length - 1]
      endedBadly = !!last?.failed && !last.stopped
      justDone = true
      const t = setTimeout(() => (justDone = false), SUCCESS_MS)
      wasAwaiting = now
      return () => clearTimeout(t)
    }
    wasAwaiting = now
  })
  // Arriving: one wave, then whatever the chat is doing.
  let hello = $state(true)
  $effect(() => {
    const t = setTimeout(() => (hello = false), HELLO_MS)
    return () => clearTimeout(t)
  })
  // Dozing: nothing to do and nobody about for a while → the charger. Any
  // turn, click or drag wakes it; the timer restarts whenever the pose it
  // would otherwise wear changes.
  let dozing = $state(false)
  $effect(() => {
    void livePose
    void reaction
    void dragging
    dozing = false
    if (cockpit.awaitingReply || dragging || reaction) return
    const t = setTimeout(() => (dozing = true), DOZE_MS)
    return () => clearTimeout(t)
  })
  const pose = $derived(
    reaction ? reaction
    : dragging && moving ? 'walk'
    : hello ? 'greeting'
    : !cockpit.awaitingReply && justDone ? (endedBadly ? 'error' : 'success')
    : dozing && !cockpit.awaitingReply ? 'recharge'
    : livePose,
  )

  // ---- what it says -------------------------------------------------------
  // The model's latest narration of its own — a delegate's rows carry `parent`
  // and are its story, not the assistant's.
  const note = $derived.by(() => {
    for (let k = cockpit.toolSteps.length - 1; k >= 0; k--) {
      const s = cockpit.toolSteps[k]
      if ((s.kind === 'note' || s.kind === 'said') && !s.parent && s.label) return s.label
    }
    return ''
  })
  const report = $derived(
    reportOf({
      awaiting: cockpit.awaitingReply,
      busy: running.length > 0,
      note,
      streamingText: cockpit.streamingText,
      question: cockpit.ask?.question,
    }),
  )
  // A narration is typed out; the answer's headline is shown as it is, since
  // it is already arriving letter by letter. Whole on any change of source.
  let shown = $state('')
  let typing = $state(false)
  let lastReport = ''
  $effect(() => {
    const text = report
    if (text === lastReport) return
    lastReport = text
    if (!text || cockpit.streamingText) {
      shown = text
      typing = false
      return
    }
    let i = 0
    typing = true
    shown = ''
    const t = setInterval(() => {
      i = Math.min(text.length, i + TYPE_STEP)
      shown = text.slice(0, i)
      if (i >= text.length) {
        typing = false
        clearInterval(t)
      }
    }, TYPE_MS)
    return () => clearInterval(t)
  })
</script>

<div
  class="companion"
  class:dragging
  class:flip
  style="transform:translate({pos.x}px,{pos.y}px); width:{SIZE}px; height:{SIZE}px"
  aria-hidden="true"
>
  {#if shown}
    <div class="say">{shown}{#if typing || cockpit.streamingText}<span class="cur"></span>{/if}</div>
  {/if}
  <!-- The frame shows on hover: a border to say "this is a thing you can
       hold", and the one control, which puts the companion away until the
       account menu brings it back. -->
  <div class="frame"></div>
  <button class="hide" type="button" title="ซ่อน" aria-label="ซ่อน" onclick={() => setCompanionOn(false)}><Icon name="x" size={11} /></button>
  <!-- A handle, not a control: it has nothing to activate, only somewhere to be. -->
  <div class="grab" role="presentation" onpointerdown={onDown} onpointermove={onMove} onpointerup={onUp} onpointercancel={onUp}>
    <Mascot {...assistantOptions(avatarPrefs)} {pose} turn={dragging ? heading : undefined} snap={dragging} size={SIZE} sway {hop} />
  </div>
</div>

<style>
  /* Above the full-window pages (settings, office, gallery sit at 50) — it is
     on the screen, not on a page — and below menus, the palette and dialogs
     (55–70), which are the things a person is actually doing. */
  .companion { position: fixed; left: 0; top: 0; z-index: 52; touch-action: none; user-select: none; }
  .grab { cursor: grab; position: relative; }
  .dragging .grab { cursor: grabbing; }
  /* the hover frame and its × — present only while the pointer is near */
  .frame { position: absolute; inset: -6px; border: 1px dashed var(--border-subtle); border-radius: 14px; opacity: 0; transition: opacity .15s; pointer-events: none; }
  .hide { position: absolute; top: -12px; right: -12px; width: 22px; height: 22px; border-radius: 50%; border: 1px solid var(--border-subtle); background: var(--surface-raised); color: var(--text-muted); display: flex; align-items: center; justify-content: center; cursor: pointer; opacity: 0; transition: opacity .15s; padding: 0; }
  .hide:hover { color: var(--text-primary); }
  .companion:hover .frame, .companion:hover .hide, .hide:focus-visible { opacity: 1; }
  .dragging .frame, .dragging .hide { opacity: 0; }
  /* the report: a small card that exists only while there is something said */
  .say {
    position: absolute; right: calc(100% + 10px); bottom: 30px;
    width: max-content; max-width: 300px;
    background: var(--surface-raised); color: var(--text-primary);
    border: 1px solid var(--border-subtle); border-radius: 12px; padding: 8px 11px;
    font-size: var(--fs-xs); line-height: 1.4;
    box-shadow: 0 6px 18px rgb(0 0 0 / 0.3);
    overflow-wrap: anywhere;
    pointer-events: none;
    animation: say-in 0.18s ease-out;
  }
  .say::after {
    content: ''; position: absolute; right: -6px; bottom: 12px; width: 12px; height: 12px;
    background: var(--surface-raised); border-right: 1px solid var(--border-subtle); border-top: 1px solid var(--border-subtle);
    transform: rotate(45deg); border-radius: 2px;
  }
  .flip .say { right: auto; left: calc(100% + 10px); }
  .flip .say::after { right: auto; left: -6px; border-right: 0; border-top: 0; border-left: 1px solid var(--border-subtle); border-bottom: 1px solid var(--border-subtle); }
  @keyframes say-in { from { opacity: 0; transform: translateY(4px) scale(0.96); } to { opacity: 1; transform: none; } }
  .cur { display: inline-block; width: 1px; height: 1em; background: var(--accent); vertical-align: -2px; margin-left: 1px; animation: cur 1s steps(2) infinite; }
  @keyframes cur { 50% { opacity: 0; } }
  @media (prefers-reduced-motion: reduce) { .say { animation: none; } .cur { animation: none; } }
</style>
