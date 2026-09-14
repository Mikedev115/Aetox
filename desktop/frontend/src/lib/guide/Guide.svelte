<script lang="ts">
  import { onMount, tick } from 'svelte'
  import Mascot from '../mascot/Mascot.svelte'
  import Icon from '../Icon.svelte'
  import { guide } from './guideState.svelte'
  import { GUIDE_MAP } from './map'
  import { GUIDE_ROUTES } from './routes'
  import { companion } from '../mascot/companionSetting.svelte'
  import { speak, stopSpeechIf } from '../speech.svelte'
  import { t } from '../i18n.svelte'

  const SIZE = 72
  let x = $state(400)
  let y = $state(300)
  let turn = $state<number | undefined>(undefined)
  let pose = $state<'idle' | 'walk' | 'presenting' | 'helping' | 'asking' | 'thinking' | 'answering'>('idle')
  let flip = $state(false)
  let below = $state(false)
  let ring = $state<{ l: number; t: number; w: number; h: number } | null>(null)
  let shown = $state('')
  let typing = $state(false)
  let askInput = $state('')
  let seq = 0

  const stop = $derived(guide.stopId ? GUIDE_MAP.find((e) => e.id === guide.stopId) : null)
  const stopName = $derived(guide.stopId ? t(`guide.${guide.stopId}.name` as any) : '')
  const stopWhat = $derived(guide.stopId ? t(`guide.${guide.stopId}.what` as any) : '')
  const stopWhy = $derived(guide.stopId ? t(`guide.${guide.stopId}.why` as any) : '')

  function sleep(ms: number) {
    return new Promise((r) => setTimeout(r, ms))
  }

  async function say(text: string) {
    const my = ++seq
    shown = ''
    typing = true
    for (const ch of text) {
      if (my !== seq) return
      shown += ch
      await sleep(ch === ' ' ? 20 : 10)
    }
    typing = false
  }

  async function moveToTarget(stopId: string, customSentence?: string) {
    seq++
    shown = ''
    typing = false
    ring = null
    stopSpeechIf('guide')

    await tick()
    // Short wait for any page route / DOM switch to render
    await sleep(60)
    await tick()

    const el = document.querySelector<HTMLElement>('[data-guide=' + JSON.stringify(stopId) + ']')
    const winW = window.innerWidth
    const winH = window.innerHeight

    let tx = winW / 2 - SIZE / 2
    let ty = winH / 2 - SIZE / 2

    if (el) {
      try {
        el.scrollIntoView({ block: 'center', behavior: 'smooth' })
      } catch {
        // ignore scroll error
      }
      await sleep(100)
      const r = el.getBoundingClientRect()
      ring = {
        l: Math.max(0, r.left - 4),
        t: Math.max(0, r.top - 4),
        w: r.width + 8,
        h: r.height + 8,
      }

      const right = r.right + 16
      const roomRight = right + SIZE + 24 < winW
      tx = roomRight ? right : Math.max(8, r.left - SIZE - 16)
      ty = Math.min(winH - SIZE - 16, Math.max(48, r.top + r.height / 2 - SIZE / 2))
    }

    const dx = tx - x
    pose = 'walk'
    turn = dx < 0 ? -70 : 70
    x = tx
    y = ty

    const standsRight = ring ? tx > ring.l : false
    const roomR = tx + SIZE + 12 + 340 <= winW
    const roomL = tx - 12 - 340 >= 0
    flip = standsRight ? roomR : !roomL
    below = ty < 260

    const reduced = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
    const walkDuration = reduced ? 0 : Math.min(800, 250 + Math.abs(dx) * 0.8)
    await sleep(walkDuration)

    turn = undefined
    const entry = GUIDE_MAP.find((e) => e.id === stopId)
    pose = entry ? (entry.safe ? 'presenting' : 'helping') : 'thinking'

    const sentence = customSentence || (stopId ? t(`guide.${stopId}.what` as any) : '')
    if (sentence) {
      void say(sentence)
      if (companion.voice) {
        speak('guide', sentence, () => {})
      }
    }
  }

  async function handlePress() {
    stopSpeechIf('guide')
    const res = guide.press()
    if (res.message) {
      await say(res.message)
      if (companion.voice) {
        speak('guide', res.message, () => {})
      }
    }
  }

  async function onAskSubmit(e: Event) {
    e.preventDefault()
    const q = askInput.trim()
    if (!q) return
    askInput = ''
    stopSpeechIf('guide')
    const answer = await guide.ask(q)
    if (guide.stopId) {
      await moveToTarget(guide.stopId, answer)
    } else {
      await say(answer)
    }
  }

  function onNext() {
    guide.next()
  }

  function onPrev() {
    guide.prev()
  }

  function onClose() {
    stopSpeechIf('guide')
    guide.stop()
  }

  $effect(() => {
    const sid = guide.stopId
    if (guide.on && sid) {
      void moveToTarget(sid)
    }
  })

  onMount(() => {
    document.body.classList.add('guide-mode')

    const onDocClick = (e: MouseEvent) => {
      if (!guide.on) return
      const target = e.target as HTMLElement | null
      if (!target) return
      if (target.closest('.guide-ui')) return

      const guideEl = target.closest<HTMLElement>('[data-guide]')
      if (guideEl) {
        const id = guideEl.getAttribute('data-guide')
        if (id) {
          e.preventDefault()
          e.stopPropagation()
          guide.goTo(id)
        }
      }
    }

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('click', onDocClick, true)
    window.addEventListener('keydown', onKeyDown)

    return () => {
      document.body.classList.remove('guide-mode')
      document.removeEventListener('click', onDocClick, true)
      window.removeEventListener('keydown', onKeyDown)
      stopSpeechIf('guide')
    }
  })
</script>

<!-- Top Guide Mode Banner -->
<div class="guide-banner guide-ui" role="region" aria-label="Guide banner">
  <div class="guide-banner-inner">
    <Icon name="compass" size={14} />
    <span class="guide-banner-text">{t('guide.modeBanner')}</span>
    <button type="button" class="guide-banner-close" onclick={onClose} aria-label="Close guide">
      <Icon name="x" size={14} />
    </button>
  </div>
</div>

<!-- Target Ring Spotlight -->
{#if ring}
  <div class="guide-ring" style="left:{ring.l}px;top:{ring.t}px;width:{ring.w}px;height:{ring.h}px"></div>
{/if}

<!-- Guide Mascot & Bubble -->
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

  {#if guide.offering}
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
  {:else if shown || typing}
    <div class="say">
      {#if stop}
        <div class="say-head">
          <span class="say-name">{stopName}</span>
          <span class="say-tag" class:no={!stop.safe}>
            {stop.safe ? t('guide.canPress') : t('guide.cannotPress')}
          </span>
          <span class="say-map-badge">{t('guide.fromMap')}</span>
        </div>
      {/if}

      <div class="say-body">{shown}{#if typing}<span class="cur"></span>{/if}</div>

      {#if stop && !typing && stopWhy}
        <div class="say-why">
          <span class="k">{t('guide.whyTitle')}</span>
          {stopWhy}
          {#if stop.ref}<span class="ref">{stop.ref}</span>{/if}
        </div>
      {/if}

      {#if !typing}
        <div class="say-foot">
          {#if stop}
            <code class="id">{stop.id}</code>
            {#if stop.safe}
              <button type="button" class="ctrl mini primary" onclick={handlePress}>{t('guide.pressForMe')}</button>
            {/if}
          {/if}
          {#if guide.route}
            <button type="button" class="ctrl mini" onclick={onPrev}>{t('guide.prev')}</button>
            <button type="button" class="ctrl mini" onclick={onNext}>{t('guide.next')}</button>
          {/if}
        </div>
      {/if}

      <form class="say-ask" onsubmit={onAskSubmit}>
        <input bind:value={askInput} placeholder={t('guide.askPlaceholder')} />
        <button type="submit" class="ctrl mini"><Icon name="sendHorizontal" size={12} /></button>
        <button type="button" class="ctrl mini close-btn" onclick={onClose} aria-label="Close"><Icon name="x" size={12} /></button>
      </form>
    </div>
  {/if}
</div>

<style>
  :global(body.guide-mode) {
    cursor: help !important;
  }
  :global(body.guide-mode .guide-ui, body.guide-mode .guide-ui *) {
    cursor: default !important;
  }

  .guide-banner {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 34px;
    background: var(--badge-amber-bg);
    color: var(--badge-amber-text);
    border-bottom: 1px solid var(--badge-amber-border);
    z-index: 10001;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25);
    font-size: var(--fs-xs);
    font-weight: 600;
  }
  .guide-banner-inner {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    max-width: 1200px;
    padding: 0 14px;
  }
  .guide-banner-text {
    flex: 1;
    text-align: center;
  }
  .guide-banner-close {
    border: 0;
    background: transparent;
    color: currentColor;
    cursor: pointer;
    display: grid;
    place-items: center;
    padding: 4px;
    border-radius: var(--r-xs);
  }
  .guide-banner-close:hover {
    background: rgba(255, 255, 255, 0.1);
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
    pointer-events: auto;
  }
  .guide-mascot-wrap.walking {
    transition-timing-function: cubic-bezier(.35, .05, .3, 1);
  }

  @media (prefers-reduced-motion: reduce) {
    .guide-ring {
      animation: none;
      transition: none;
    }
    .guide-mascot-wrap {
      transition: none;
    }
  }

  .ranked-guide {
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
