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
  // question it is blocked on. Never a command, and one phrase of ours only:
  // the room's own greeting, spoken once as an empty chat comes on screen
  // (see "what it says"). Everything else in the bubble is the model's.
  //
  // Since 12 ก.ย. 2026 it also speaks — the on-screen chat's finished answer
  // and a question it is blocked on, out loud, through the same engine the
  // ฟัง button uses (see "what it says out loud").
  //
  // Cost: one SVG drawn once per pose, CSS for everything that moves. Nothing
  // here runs per frame.
  import { untrack } from 'svelte'
  import Mascot from './Mascot.svelte'
  import Icon from '../Icon.svelte'
  import { presenceOf, reportOf, headlineOf, walkTurn, nearAngle } from './presence'
  import { POSE, type PoseId } from './poses'
  import { cockpit } from '../stores/cockpit.svelte'
  import { companion, setCompanionOn, setCompanionVoice, setCompanionSize, clampSize } from './companionSetting.svelte'
  import { desktopBody, openBody, closeBody, onBodyInput, bakeFor, rememberPos, themeColors, wordsOf } from './desktopBody.svelte'
  import { speech, speak, stopSpeechIf } from '../speech.svelte'
  import { avatarPrefs, assistantOptions } from './avatarPrefs.svelte'
  import { voice } from './voice.svelte'
  import { profile } from '../stores/profile.svelte'
  import { startersFor, headlineFor } from '../starters'
  import { t, i18n } from '../i18n.svelte'

  /** The figure's size, logical px — the user's, dragged at the corner of
   *  the hover frame (companionSetting.svelte.ts size). */
  const SIZE = $derived(companion.size)
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
  /** How long the greeting stays in the bubble when an empty chat arrives. */
  const GREET_MS = 4000
  /** How long the greeting waits before it is spoken: the user's name can
   *  arrive a beat after launch and re-key the greeting, and a voice that
   *  had already started on the nameless one would say it twice. */
  const GREET_SAY_MS = 400
  /** Left alone this long with nothing to do, it goes to sleep. */
  const DOZE_MS = 5 * 60_000
  /** Getting up after being poked, and the jolt of being woken by a message. */
  const WAKE_MS = 1300
  const STARTLE_MS = 750

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
    /** The direction being turned towards, in degrees, once there is one. */
    want: number | null; at: number
    target: { x: number; y: number }; raf: number
  } | null = null
  /** Which way it walks while dragged, in the head's own degrees — right,
   *  left, away up the screen, or towards the viewer down it. */
  let heading = $state(0)
  /** Degrees per second the head may turn while walking. */
  const TURN_DEG_S = 240
  /** How much of the remaining distance to the hand is closed per frame. */
  const FOLLOW = 0.28
  /** A step shorter than this has no direction worth reading. */
  const STEP_PX = 2
  /** A new direction must differ from the one being turned to by more than
   *  this before it replaces it — a hand wobbles, a walker does not. */
  const RETARGET_DEG = 25
  /** No movement for this long while held is standing, not walking: it
   *  floats in place, facing the way it was going (owner: "คลิกค้างอยู่ที่เดิม
   *  ควรจะลอยอยู่เฉย ๆ"). */
  const STOP_MS = 200
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
      want: null, at: performance.now(), target: { ...pos }, raf: 0,
    }
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }
  function onMove(e: PointerEvent): void {
    if (!drag) return
    if (!drag.moved && Math.hypot(e.clientX - drag.x0, e.clientY - drag.y0) < CLICK_PX) return
    drag.moved = true
    dragging = true
    // Where it may go: the hand's spot, kept inside the window. At an edge
    // the grip is re-anchored to the clamped spot, so the way back starts
    // the moment the hand turns round instead of only once the hand is back
    // where it was when the edge was hit — that dead stretch was the "ติด ๆ"
    // the owner saw at the sides of the window.
    const t = clamp({ x: e.clientX - drag.dx, y: e.clientY - drag.dy })
    drag.dx = e.clientX - t.x
    drag.dy = e.clientY - t.y
    // Everything below reads the mascot's own displacement, not the hand's:
    // pressed against an edge it is not walking, so it stands there and
    // floats, and it does not turn to face a hand sliding along the wall.
    const mx = t.x - drag.target.x
    const my = t.y - drag.target.y
    drag.target = t
    const now = performance.now()
    const dt = Math.min(0.05, (now - drag.at) / 1000)
    drag.at = now
    const step = Math.hypot(mx, my)
    if (step >= 0.5) {
      moving = true
      clearTimeout(moveTimer)
      moveTimer = setTimeout(() => (moving = false), STOP_MS)
    }
    // The direction is the latest real step's, held until a clearly
    // different one comes along (not a blend of recent steps: a blend
    // sweeps through every angle between two directions when the hand
    // turns round, and the head chased that sweep the long way about).
    // The head turns towards it no faster than TURN_DEG_S, the short way.
    if (step >= STEP_PX) {
      const h = walkTurn(mx, my, drag.want ?? 0, 0)
      if (drag.want === null || Math.abs(h - drag.want) > RETARGET_DEG) drag.want = h
    }
    if (drag.want !== null) {
      const goal = nearAngle(drag.want, heading)
      const turn = TURN_DEG_S * dt
      heading += Math.max(-turn, Math.min(turn, goal - heading))
    }
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
      // Clicked while talking: that is "shush" (the read ends, the switch
      // stays as it was), with the same hop as any other click.
      hush()
      react()
      return
    }
    // Rewind whole circles now, while the head still snaps; next frame the
    // easing is back and the turn home is the short way round.
    heading = ((((heading + 180) % 360) + 360) % 360) - 180
    // A timeout, not a frame: a frame never comes while the window is hidden.
    setTimeout(() => (dragging = false), 20)
    try {
      localStorage.setItem(POS_KEY, JSON.stringify(pos))
    } catch {
      // A spot that could not be remembered is still the spot for this session.
    }
  }

  // ---- being resized ---------------------------------------------------------
  // The corner of the hover frame, bottom-right, is a handle: dragging it
  // grows or shrinks the figure about its top-left, between SIZE_MIN and
  // SIZE_MAX, and the number is kept for both places. The bubble, the frame
  // and the buttons are laid out from the figure's box, so they follow.
  let resizing = $state(false)
  let grip: { x0: number; y0: number; size0: number } | null = null
  function onGripDown(e: PointerEvent): void {
    if (e.button !== 0) return
    e.stopPropagation()
    grip = { x0: e.clientX, y0: e.clientY, size0: SIZE }
    resizing = true
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  }
  function onGripMove(e: PointerEvent): void {
    if (!grip) return
    const d = Math.max(e.clientX - grip.x0, e.clientY - grip.y0)
    const next = clampSize(grip.size0 + d)
    if (next !== SIZE) {
      setCompanionSize(next)
      pos = clamp(pos)
    }
  }
  function onGripUp(): void {
    grip = null
    resizing = false
  }

  // ---- being clicked ------------------------------------------------------
  // A moment of one of the reactions, with a hop, then back to whatever it
  // was doing. No words: the bubble is for what the model says (presence.ts).
  let reaction = $state<string>('')
  let hop = $state(false)
  let reactTimer: ReturnType<typeof setTimeout> | undefined
  function react(): void {
    // Asleep, a poke does not get a cheer; it gets a slow stretch.
    if (asleep) {
      wakeUp('wake')
      return
    }
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
  type FeedState = {
    pose: string
    report: string
    on: boolean
    prefs: { shell: string; accent: string; top: string; face: string }
    shown: string
    words: string[]
    cursor: boolean
    theme: { bg: string; fg: string; muted: string; border: string; accent: string }
    muted: boolean
    hop: number
    size: number
  }
  const feed = (): ((s: FeedState) => Promise<void>) | undefined =>
    (window as unknown as { go?: { main?: { App?: { SetCompanionState?: (s: FeedState) => Promise<void> } } } }).go?.main?.App?.SetCompanionState
  /** Clicks reacted to, counted, so the body hops with each (companion.go Hop). */
  let hops = $state(0)
  const prefsOf = () => ({ shell: avatarPrefs.shell, accent: avatarPrefs.accent, top: avatarPrefs.top, face: avatarPrefs.face })
  $effect(() => {
    const send = feed()
    if (!send) return
    const s: FeedState = {
      pose,
      report: said,
      on: true,
      prefs: prefsOf(),
      shown,
      words: wordsOf(shown, i18n.locale),
      cursor: typing || !!cockpit.streamingText,
      theme: themeColors(),
      muted: !companion.voice,
      hop: hops,
      size: SIZE,
    }
    void send(s).catch(() => {})
  })
  $effect(() => () => {
    const send = feed()
    if (send) void send({ pose: 'idle', report: '', on: false, prefs: prefsOf(), shown: '', words: [], cursor: false, theme: themeColors(), muted: !companion.voice, hop: hops, size: SIZE }).catch(() => {})
  })

  // ---- the body on the desktop ------------------------------------------------
  // With `place` set to the desktop, the figure is Go's window (desktopBody
  // .svelte.ts) and this component draws nothing — but decides everything
  // still: the body reports its clicks and drags here and is told the pose
  // and the words back through the feed above. If the window cannot be
  // opened (another platform), the figure stays in here as if the switch
  // had not been thrown.
  const wantsDesktop = $derived(companion.on && companion.place === 'desktop')
  $effect(() => {
    if (!wantsDesktop) {
      // Always, not only when this page opened it: a page that reloaded
      // (dev, or a crash of the webview) never ran its cleanup, and the
      // body it left behind would sit there until the app closed.
      void closeBody()
      return
    }
    void openBody(untrack(() => SIZE))
    return () => void closeBody()
  })
  $effect(() => {
    if (!wantsDesktop) return
    return onBodyInput((input) => {
      switch (input.kind) {
        case 'click':
          hush()
          hops++
          react()
          break
        case 'dragStart':
          dragging = true
          break
        case 'dragEnd':
          dragging = false
          break
        case 'moved':
          rememberPos(input.x, input.y)
          break
        case 'hide':
          setCompanionOn(false)
          break
        case 'mute':
          setCompanionVoice(!companion.voice)
          break
        case 'bake':
          desktopBody.scale = input.scale
          break
        case 'resize':
          setCompanionSize(input.size)
          break
      }
    })
  })
  // The frames the body draws from, baked for its monitor's scale and for
  // the look chosen — again whenever either changes.
  $effect(() => {
    const scale = desktopBody.scale
    const opts = assistantOptions(avatarPrefs)
    if (!wantsDesktop || !desktopBody.up || scale <= 0) return
    void bakeFor(opts, scale, untrack(() => pose))
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
      // The answer, read aloud: only a reply that arrived (not a failure, not
      // a stop), and only here — the turn of the chat on screen, which is
      // the one this flag follows (a chat working off screen ends parked).
      if (last?.role === 'agent' && !last.failed && last.text) untrack(() => say(last.text))
      const t = setTimeout(() => (justDone = false), SUCCESS_MS)
      wasAwaiting = now
      return () => clearTimeout(t)
    }
    // A new message going out: the user is talking now, so it stops.
    if (!wasAwaiting && now) untrack(hush)
    wasAwaiting = now
  })
  // Arriving: one wave, then whatever the chat is doing.
  let hello = $state(true)
  $effect(() => {
    const t = setTimeout(() => (hello = false), HELLO_MS)
    return () => clearTimeout(t)
  })
  // Dozing: nothing to do and nobody about for a while → the pillow. Any
  // turn, click or drag wakes it; the timer restarts whenever the pose it
  // would otherwise wear changes. `asleep` mirrors `dozing` outside the
  // reactive graph so the effect can ask "was it asleep?" without depending
  // on the answer.
  //
  // Waking is not a cut (owner: "ค่อย ๆ ลุก … ส่งข้อความตอนนอนควรจะตกใจแล้วลุก
  // มาทำงาน"): a message while it sleeps startles it — a jolt, both hands
  // up, then straight to work; anything else (a click, a drag) gets a slow
  // stretch first. The lean itself eases over .9s in mascot.css either way.
  let dozing = $state(false)
  let asleep = false
  let waking = $state<'' | 'wake' | 'startled'>('')
  let wakeTimer: ReturnType<typeof setTimeout> | undefined
  function wakeUp(how: 'wake' | 'startled'): void {
    dozing = false
    asleep = false
    waking = how
    if (how === 'startled') {
      hop = false
      requestAnimationFrame(() => (hop = true))
    }
    clearTimeout(wakeTimer)
    wakeTimer = setTimeout(() => {
      waking = ''
      hop = false
    }, how === 'startled' ? STARTLE_MS : WAKE_MS)
  }
  $effect(() => {
    void livePose
    void reaction
    void dragging
    void waking
    const busy = cockpit.awaitingReply || dragging || !!reaction
    if (asleep) wakeUp(cockpit.awaitingReply ? 'startled' : 'wake')
    else dozing = false
    if (busy || waking) return
    const t = setTimeout(() => {
      dozing = true
      asleep = true
    }, DOZE_MS)
    return () => clearTimeout(t)
  })
  $effect(() => () => clearTimeout(wakeTimer))
  // ---- the greeting -----------------------------------------------------
  // The one line of ours it speaks, declared here because the pose below
  // waves while it is said. When an empty chat comes on
  // screen — a new chat, a desk switched to, the app opening on one — it says
  // the question the room prints above the cards, with the user's name where
  // the desk greets a person (starters.headlineFor), and waves. Not a second
  // phrase: the same string the room shows, from the same function, so the
  // two cannot drift. Keyed on the room and the name, so switching between
  // two empty rooms greets again and a name that loads a beat after launch
  // corrects the greeting rather than missing it; a chair's opening is the
  // agent's own and is left to the room. It ends on its own (GREET_MS) and
  // anything the model says takes the bubble over at once (owner, 12 ก.ย.:
  // "ทำให้อวตารพูด mike วันนี้มีอะไรให้ช่วยไหม ออกมาด้วย").
  const emptyRoom = $derived(
    cockpit.activeView === 'chat' && cockpit.chat.length === 0 && !cockpit.awaitingReply && !cockpit.chair
      ? `${cockpit.desk}|${cockpit.space}|${cockpit.openSession}|${profile.name}`
      : '',
  )
  let greeting = $state('')
  $effect(() => {
    if (!emptyRoom) {
      greeting = ''
      return
    }
    const text = headlineFor(startersFor({ desk: cockpit.desk, chair: '', space: cockpit.space }), profile.name, t)
    greeting = text
    const tm = setTimeout(() => (greeting = ''), GREET_MS)
    // Said out loud too — the wave with nothing behind it read as a bug
    // (owner, 12 ก.ย.: "ทำไมมันเงียบ"). The one phrase of ours the voice
    // carries, behind its own switch; the bubble holds the line until the
    // voice is done with it.
    const sp = setTimeout(() => untrack(() => { if (companion.greet) say(text) }), GREET_SAY_MS)
    return () => {
      clearTimeout(tm)
      clearTimeout(sp)
    }
  })

  const pose = $derived(
    reaction ? reaction
    : waking ? waking
    : dragging && moving ? 'walk'
    : hello || greeting ? 'greeting'
    : !cockpit.awaitingReply && justDone ? (endedBadly ? 'error' : 'success')
    : dozing && !cockpit.awaitingReply ? 'recharge'
    : livePose,
  )

  // ---- what it says out loud ---------------------------------------------
  // The voice (owner, 12 ก.ย. 2026: "เพิ่มให้มันพูดได้ ใช้ TTS ในระบบเลย"): the
  // on-screen chat's finished answer, a question it is blocked on, and the
  // room's greeting as an empty chat arrives (see "the greeting"), read
  // through the window's one player (lib/speech.svelte.ts) with the engine
  // and voice of ตั้งค่า › เสียง — the same read the ฟัง button makes, started
  // for the user. NOT the narration between tools: a long run says a line
  // per round, and a voice that read every one would still be on round three
  // when the report came, or talk over itself ("คิดเผื่อตอนมันทำงานยาว …
  // เสียงชนกัน"). So a run is quiet until it reports, and the bubble carries
  // the rounds meanwhile.
  //
  // One read at a time, newest wins (the player's rule), and it stops the
  // moment the user is speaking instead: a new message, the mic, another
  // chat brought on screen, the switch, or a click on the figure. Silent on
  // any failure — no voice installed, an engine that cannot run — because a
  // report is not the place for an error; the avatar page says why
  // (AvatarSettings.svelte), and the ฟัง button still shows its own.
  const VOICE_KEY = 'companion'
  const talking = $derived(speech.key === VOICE_KEY)
  /** The text being read, for the bubble; '' once the read ends. */
  let spoken = $state('')
  function say(text: string): void {
    if (!companion.voice) return
    spoken = text
    void speak(VOICE_KEY, text)
  }
  function hush(): void {
    stopSpeechIf(VOICE_KEY)
  }
  $effect(() => {
    if (!talking) spoken = ''
  })
  // A question it is blocked on is read as it appears — the one moment in a
  // run the user has to come back for.
  let lastAsked = ''
  $effect(() => {
    const q = cockpit.ask?.question ?? ''
    if (q && q !== lastAsked) untrack(() => say(q))
    lastAsked = q
  })
  // untrack: hush reads the player's key, and an effect that tracked it
  // would run again as a read began — and end it.
  $effect(() => {
    void cockpit.openSession
    untrack(hush)
  })
  $effect(() => {
    if (voice.mic || !companion.voice) untrack(hush)
  })
  $effect(() => () => hush())

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
  // While it reads the answer the bubble holds the answer's first line — the
  // headline the stream showed, kept up until the voice is done with it. The
  // greeting fills the bubble only while the model has nothing to say.
  const said = $derived(report || (talking && spoken ? headlineOf(spoken) : '') || greeting)
  // A narration is typed out; the answer's headline is shown as it is, since
  // it is already arriving letter by letter. Whole on any change of source.
  let shown = $state('')
  let typing = $state(false)
  let lastReport = ''
  $effect(() => {
    const text = said
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

{#if !desktopBody.up}
<div
  class="companion"
  class:dragging
  class:resizing
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
  <!-- The corner: a grip to resize by. -->
  <div class="grip" role="presentation" onpointerdown={onGripDown} onpointermove={onGripMove} onpointerup={onGripUp} onpointercancel={onGripUp}></div>
  <button class="hide" type="button" title="ซ่อน" aria-label="ซ่อน" onclick={() => setCompanionOn(false)}><Icon name="x" size={11} /></button>
  <!-- The other control on the frame: the voice, on or off. The same switch
       as the avatar page's row; here because "make it stop talking" is
       wanted where the talking is. -->
  <button class="mute" class:off={!companion.voice} type="button" title={companion.voice ? 'ปิดเสียง' : 'เปิดเสียง'} aria-label={companion.voice ? 'ปิดเสียง' : 'เปิดเสียง'} aria-pressed={!companion.voice} onclick={() => setCompanionVoice(!companion.voice)}><Icon name={companion.voice ? 'volume2' : 'volumeX'} size={11} /></button>
  <!-- A handle, not a control: it has nothing to activate, only somewhere to be. -->
  <div class="grab" role="presentation" onpointerdown={onDown} onpointermove={onMove} onpointerup={onUp} onpointercancel={onUp}>
    <Mascot {...assistantOptions(avatarPrefs)} {pose} turn={dragging ? heading : undefined} snap={dragging} size={SIZE} sway {hop} />
  </div>
</div>
{/if}

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
  .mute { position: absolute; top: -12px; left: -12px; width: 22px; height: 22px; border-radius: 50%; border: 1px solid var(--border-subtle); background: var(--surface-raised); color: var(--text-muted); display: flex; align-items: center; justify-content: center; cursor: pointer; opacity: 0; transition: opacity .15s; padding: 0; }
  .mute:hover { color: var(--text-primary); }
  /* Muted, it stays visible: a figure that will not talk should say so. */
  .mute.off { opacity: 1; color: var(--text-muted); }
  .companion:hover .frame, .companion:hover .hide, .companion:hover .mute, .companion:hover .grip, .resizing .frame, .resizing .grip, .hide:focus-visible, .mute:focus-visible { opacity: 1; }
  .dragging .frame, .dragging .hide, .dragging .mute, .dragging .grip { opacity: 0; }
  /* the corner grip: two short strokes, the way a window's corner says "pull here" */
  .grip {
    position: absolute; right: -6px; bottom: -6px; width: 16px; height: 16px;
    cursor: nwse-resize; opacity: 0; transition: opacity .15s;
    background: linear-gradient(135deg, transparent 0 55%, var(--text-muted) 55% 62%, transparent 62% 75%, var(--text-muted) 75% 82%, transparent 82%);
    border-bottom-right-radius: 14px;
  }
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
