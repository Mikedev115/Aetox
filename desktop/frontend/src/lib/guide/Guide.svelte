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
  import { offeredWalks } from './greeting'
  import GuidePanel from './GuidePanel.svelte'
  import { guidePrefs, setGuidePref } from './guidePrefs.svelte'
  import { GUIDE_MAP, guideText } from './map'
  import { SIZE, standBeside, restingSpot, walkTurn, walkMs } from './walk'
  import { speak, stopSpeechIf } from '../speech.svelte'
  import { t } from '../i18n.svelte'
  import { handleGuideAsk } from './guideSession'
  import { EventsOn } from '../../../wailsjs/runtime/runtime'

  // Placed on the FIRST frame, not moved into place after it. x and y used to
  // start at 0, so the figure rendered in the top-left corner and then rode the
  // 800ms walk transition across the whole window to its resting spot — which
  // reads as a thing sliding in from nowhere, every single time it is opened
  // (owner, 15 ก.ย. 2026: "กดไกด์แล้วเหมือนมันลอยออกมาจากตรงไหนไม่รู้").
  // It now appears where it belongs and fades in on the spot; walking is for
  // going somewhere, not for arriving.
  const first = typeof window === 'undefined'
    ? { x: 0, y: 0 }
    : restingSpot({ width: window.innerWidth, height: window.innerHeight })
  let x = $state(first.x)
  let y = $state(first.y)
  let placed = $state(typeof window !== 'undefined')
  let turn = $state<number | undefined>(undefined)
  let pose = $state<'idle' | 'walk' | 'presenting' | 'helping' | 'asking' | 'thinking' | 'answering'>('asking')
  let flip = $state(false)
  let below = $state(false)
  let ring = $state<{ l: number; t: number; w: number; h: number } | null>(null)
  let shown = $state('')
  let typing = $state(false)
  let askInput = $state('')
  /** Which face of the bubble is showing. */
  let face = $state<'talk' | 'settings'>('talk')
  let seq = 0

  const stop = $derived(guide.stopId ? GUIDE_MAP.find((e) => e.id === guide.stopId) : null)
  const stopName = $derived(guide.stopId ? guideText(guide.stopId, 'name') : '')
  const stopWhy = $derived(guide.stopId ? guideText(guide.stopId, 'why') : '')

  function sleep(ms: number) {
    return new Promise((r) => setTimeout(r, ms))
  }

  function reduced(): boolean {
    return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
  }

  function home() {
    ;({ x, y } = restingSpot({ width: window.innerWidth, height: window.innerHeight }))
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
  function placeBeside(r: DOMRect) {
    const st = standBeside(r, { width: window.innerWidth, height: window.innerHeight })
    ring = st.ring
    flip = st.flip
    below = st.below
    return { tx: st.x, ty: st.y }
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
    let el = document.querySelector<HTMLElement>('[data-guide=' + JSON.stringify(stopId) + ']')
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
      el = document.querySelector<HTMLElement>('[data-guide=' + JSON.stringify(stopId) + ']')
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
    if (el && guide.stopId === stopId) {
      const again = document.querySelector<HTMLElement>('[data-guide=' + JSON.stringify(stopId) + ']')
      if (again) {
        const r = again.getBoundingClientRect()
        if (ring && (Math.abs(r.left - 4 - ring.l) > 2 || Math.abs(r.top - 4 - ring.t) > 2)) {
          ;({ tx, ty } = placeBeside(r))
          x = tx
          y = ty
        }
      }
    }
    const entry = GUIDE_MAP.find((e) => e.id === stopId)
    pose = !el ? 'thinking' : entry?.safe ? 'presenting' : 'helping'
    if (guide.saySeq !== saidAtStart) return
    const text = sentence || (el ? guideText(stopId, 'what') : t('guide.notOnScreen'))
    void say(text)
    voice(text)
  }

  async function handlePress() {
    stopSpeechIf('guide')
    const res = guide.press()
    if (res.message) {
      void say(res.message)
      voice(res.message)
    }
  }

  async function onAskSubmit(e: Event) {
    e.preventDefault()
    const q = askInput.trim()
    if (!q) return
    askInput = ''
    stopSpeechIf('guide')
    pose = 'thinking'
    await guide.ask(q)
  }

  function onClose() {
    stopSpeechIf('guide')
    face = 'talk'
    void guide.stop()
  }

  // A walk: the store bumped moveSeq after openPage settled the page.
  $effect(() => {
    const n = guide.moveSeq
    if (!n || !guide.on) return
    const id = guide.stopId
    const sentence = guide.sentence
    if (id) untrack(() => void moveToTarget(id, sentence))
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
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    const onResize = () => {
      if (guide.stopId && guide.on) guide.moveSeq++
    }

    const offGuide = EventsOn('screen:guide', (ask: any) => {
      void handleGuideAsk(ask)
    })
    const offChunk = EventsOn('agent:chunk', (ev: any) => {
      if (guide.sessionId && ev?.sessionId === guide.sessionId) {
        guide.onChunk(ev?.data?.text || '', !!ev?.data?.replace)
      }
    })

    document.addEventListener('click', onDocClick, true)
    window.addEventListener('keydown', onKeyDown)
    window.addEventListener('resize', onResize)

    return () => {
      document.body.classList.remove('guide-mode')
      document.removeEventListener('click', onDocClick, true)
      window.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('resize', onResize)
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

{#if ring}
  <div class="guide-ring" style="left:{ring.l}px;top:{ring.t}px;width:{ring.w}px;height:{ring.h}px"></div>
{/if}

<div
  class="guide-mascot-wrap guide-ui"
  class:flip
  class:below
  class:walking={pose === 'walk'}
  style="transform:translate({x}px,{y}px)"
>
  <span class="ranked-guide" style="width:{SIZE}px;height:{SIZE}px">
    <Mascot role="assistant" accent="amber" top="bar" face="happy" prop="none" shell="pastel" {pose} {turn} size={SIZE} sway />
    <span class="rank-corner rank-guide" title={t('rank.guide')} role="img" aria-label={t('rank.guide')}>
      <Icon name="compass" size={14} />
    </span>
  </span>

  <!-- The frame the companion wears, for the same reasons: a figure you can
       dismiss, silence, or open the settings of, without hunting for where
       those live (owner, 15 ก.ย. 2026: "ให้มันมีขอบเหมือนผู้ช่วยด้วยดิ แบบกด
       ปิดได้ ปิดเสียงได้ เพิ่มปุ่มฟันเฟือง"). -->
  <span class="guide-frame"></span>
  <button type="button" class="g-btn g-gear" class:on={face === 'settings'}
          title={t('guide.set.title')} aria-label={t('guide.set.title')}
          onclick={() => (face = face === 'settings' ? 'talk' : 'settings')}>
    <Icon name="settings" size={11} />
  </button>
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

  {#if face === 'settings'}
    <div class="say say-panel">
      <GuidePanel onClose={() => (face = 'talk')} onPick={(id) => { face = 'talk'; void guide.goTo(id) }} />
    </div>
  {:else if guide.offering}
    <div class="say">
      <div class="say-head">
        <span class="say-name">{t('rank.guide')}</span>
      </div>
      <div class="say-body">{t('guide.offerPrompt')}</div>
      <div class="say-foot">
        <button type="button" class="ctrl mini primary" onclick={() => guide.acceptOffer()}>{t('guide.offerYes')}</button>
        <button type="button" class="ctrl mini" onclick={() => guide.declineOffer()}>{t('guide.offerNo')}</button>
      </div>
    </div>
  {:else if shown || typing || guide.asking}
    <div class="say">
      {#if stop}
        <div class="say-head">
          <span class="say-name">{stopName}</span>
          <span class="say-tag" class:no={!stop.safe}>{stop.safe ? t('guide.canPress') : t('guide.cannotPress')}</span>
          {#if guide.brain === 'map'}<span class="say-map-badge">{t('guide.fromMap')}</span>{/if}
        </div>
      {/if}

      <div class="say-body">{shown}{#if typing || guide.asking}<span class="cur"></span>{/if}</div>

      <!-- Being led: what to press, and how far there is to go. The button is
           offered too, for somebody who would rather be taken than walk. -->
      {#if guide.awaiting && !typing}
        <div class="say-step">
          <Icon name="pointer" size={13} />
          <span>{t('guide.stepPress')}</span>
          {#if guide.stepsLeft > 1}<span class="say-step-left">{t('guide.stepLeft', { n: String(guide.stepsLeft) })}</span>{/if}
        </div>
      {/if}

      <!-- A question with no visible answers is a search box: name the walks. -->
      {#if !guide.stopId && !typing && !guide.asking}
        <div class="say-walks">
          {#each offeredWalks() as w (w.id)}
            <button type="button" class="ctrl mini" onclick={() => guide.start(w.id)}>{w.label}</button>
          {/each}
        </div>
      {/if}

      {#if stop && !typing && !guide.asking && stopWhy}
        <div class="say-why">
          <span class="k">{t('guide.whyTitle')}</span>{stopWhy}{#if stop.ref}<span class="ref">{stop.ref}</span>{/if}
        </div>
      {/if}

      {#if !typing && !guide.asking}
        <div class="say-foot">
          {#if stop}
            <code class="id">{stop.id}</code>
            {#if stop.safe}
              <button type="button" class="ctrl mini primary" onclick={handlePress}>{t('guide.pressForMe')}</button>
            {/if}
          {/if}
          {#if guide.route}
            <button type="button" class="ctrl mini" onclick={() => guide.prev()}>{t('guide.prev')}</button>
            <button type="button" class="ctrl mini" onclick={() => guide.next()}>{t('guide.next')}</button>
          {/if}
        </div>
      {/if}

      <form class="say-ask" onsubmit={onAskSubmit}>
        <input id="guide-ask" bind:value={askInput} placeholder={t('guide.askPlaceholder')} disabled={guide.asking} />
        <button type="submit" class="ctrl mini" aria-label={t('guide.ask')} disabled={guide.asking}><Icon name="sendHorizontal" size={12} /></button>
        <button type="button" class="ctrl mini close-btn" onclick={onClose} aria-label={t('guide.close')}><Icon name="x" size={12} /></button>
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

  .guide-ring {
    position: fixed;
    pointer-events: none;
    border: 2px solid var(--badge-amber-text);
    border-radius: 10px;
    box-shadow: 0 0 0 4px rgba(229, 169, 60, 0.25), 0 0 24px rgba(229, 169, 60, 0.4);
    animation: guide-breathe 1.6s ease-in-out infinite;
    transition: left var(--dur-walk, 800ms), top var(--dur-walk, 800ms), width var(--dur-walk, 800ms), height var(--dur-walk, 800ms);
    z-index: 9998;
  }
  @keyframes guide-breathe {
    50% {
      box-shadow: 0 0 0 7px rgba(229, 169, 60, 0.15), 0 0 32px rgba(229, 169, 60, 0.5);
    }
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

  @keyframes guide-arrive {
    from { opacity: 0; }
  }
  @keyframes guide-arrive-pop {
    from { transform: scale(.88); }
  }

  @media (prefers-reduced-motion: reduce) {
    .guide-ring {
      animation: none;
      transition: none;
    }
    .guide-mascot-wrap {
      transition: none;
      animation: none;
    }
    .ranked-guide {
      animation: none;
    }
  }

  .ranked-guide {
    animation: guide-arrive-pop var(--dur-arrive, 300ms) cubic-bezier(.2, .9, .3, 1.2);
    position: relative;
    display: inline-block;
    line-height: 0;
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

  /* The frame and its three buttons — the companion's own affordance, at the
     guide's size. Visible on hover or focus, and the mute stays lit when it is
     off so a silent figure says it is silent. */
  .guide-frame {
    position: absolute; inset: -6px; border: 1px dashed var(--border-subtle);
    border-radius: 14px; opacity: 0; transition: opacity .15s; pointer-events: none;
  }
  .g-btn {
    position: absolute; width: 20px; height: 20px; border-radius: 50%;
    border: 1px solid var(--border-subtle); background: var(--surface-raised);
    color: var(--text-muted); display: grid; place-items: center; padding: 0;
    cursor: pointer; opacity: 0; transition: opacity .15s, color .15s;
  }
  .g-btn:hover { color: var(--text-primary); }
  .g-hide { top: -10px; right: -10px; }
  .g-mute { top: -10px; left: -10px; }
  .g-gear { bottom: -10px; left: -10px; }
  .g-gear.on { opacity: 1; color: var(--badge-amber-text); border-color: var(--badge-amber-border); }
  .g-mute.off { opacity: 1; }
  .guide-mascot-wrap:hover .guide-frame,
  .guide-mascot-wrap:hover .g-btn,
  .g-btn:focus-visible { opacity: 1; }

  .say {
    position: absolute;
    right: calc(100% + 14px);
    bottom: 24px;
    width: 350px;
    max-width: 350px;
    background: var(--surface-raised);
    color: var(--text-primary);
    border: 1px solid var(--border-subtle);
    border-radius: 14px;
    padding: 12px 14px;
    font-size: var(--fs-sm);
    line-height: 1.45;
    box-shadow: 0 8px 28px rgba(0, 0, 0, 0.35);
    animation: say-in var(--dur-tint, 120ms) ease-out;
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

  /* The settings face is taller and scrolls rather than growing off screen. */
  .say-panel { max-height: min(62vh, 460px); overflow-y: auto; }

  .say-head {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
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

  .say-body {
    margin-bottom: 6px;
  }

  .say-why {
    margin-top: 8px;
    padding-top: 8px;
    border-top: 1px dashed var(--border-subtle);
    color: var(--text-muted);
    font-size: var(--fs-xs);
  }
  .say-why .k {
    color: var(--badge-amber-text);
    font-weight: 600;
    margin-right: 4px;
  }
  .say-why .ref {
    font-family: var(--font-mono, monospace);
    font-size: 0.85em;
    opacity: 0.8;
    margin-left: 4px;
  }

  /* Being led somewhere: the instruction reads as an instruction, in the
     guide's own colour, and sits above the map's explanation rather than
     inside it — one is what to do now, the other is what the thing is. */
  .say-step {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 8px;
    padding: 6px 9px;
    border-radius: 9px;
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    border: 1px solid var(--badge-amber-border);
    font-size: var(--fs-xs);
    font-weight: 600;
  }
  .say-step-left {
    margin-left: auto;
    font-weight: 500;
    opacity: 0.75;
  }
  .say-walks {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 10px;
  }

  .say-foot {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 10px;
  }
  .say-foot .id {
    margin-right: auto;
    font-family: var(--font-mono, monospace);
    font-size: 0.85em;
    color: var(--text-dim);
    background: var(--surface-sunken);
    border: 1px solid var(--border-subtle);
    border-radius: 4px;
    padding: 1px 4px;
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
    font-size: var(--fs-xs);
  }
  .say-ask input:focus {
    outline: none;
    border-color: var(--badge-amber-border);
  }

  .ctrl.mini {
    padding: 2px 8px;
    font-size: var(--fs-xs);
    border-radius: 6px;
    border: 1px solid var(--border-default);
    background: var(--surface-sunken);
    color: var(--text-secondary);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }
  .ctrl.mini:hover {
    background: var(--surface-hover);
    color: var(--text-primary);
  }
  .ctrl.mini.primary {
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    border-color: var(--badge-amber-border);
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
