<script lang="ts">
  // ตั้งค่า › ส่วนบุคคล › อวตาร — the assistant's mascot, laid out like a game's
  // character screen (owner, 12 ก.ย.: "ทำหน้าเหมือนตั้งค่าเกม แบบแยกส่วน อวตาร
  // อยู่ตรงกลาง แล้วชี้เป็นส่วน ๆ อันไหนคือปรับอันไหน").
  //
  // The figure sits in the middle of the stage, on the centre line and at
  // mid-height whatever the panels around it grow to; the four panels take
  // the four corners, and a dashed lead runs from the part itself — the
  // crown, an ear, the body, the screen — to the panel that changes it, so
  // nobody has to guess which row is which. The stage is a container query,
  // not a viewport one: what matters is the width this page was given, and
  // below a comfortable three-column width the figure goes on top and the
  // panels line up beneath it in two, then one (owner: "คิดถึงจอหลายขนาดด้วย").
  //
  // Every cell in a panel is the whole outcome, still (a picker of forty
  // breathing together stuttered), and reads the same catalogue the rig draws
  // from: a shell, an accent, a top light or a face appended in palette.ts /
  // parts.ts appears here untouched. Under the stage, every pose the rig has,
  // to see the look move.
  //
  // Personas — as many saved looks as the user wants, a + card at the end
  // of the row — exist for the agents a user will design later: a persona is
  // a look ready to be handed to one, and either head may wear one.
  //
  // Two heads since 14 ก.ย. 2026 (owner: "แยกอวตาร … ถ้าไปหน้าโค้ดก็โหลดอวตารอีกตัว"):
  // the assistant desk's and the code desk's, each with its own four dials
  // (avatarPrefs.svelte.ts `heads`). The two cards under the banner say
  // which one the stage is dressing — cards, not a dropdown, because the
  // point is seeing them side by side and seeing that they differ. The
  // template each is drawn from (roles.ts) is the head's own and not a
  // choice: the `>_` on the coder's ears is what says it is the coder.
  //
  // Two sub-menus (owner, 13 ก.ย.: "แยก ตั้งค่าอวตารหลัก กับ ออกแบบอวตาร"), the
  // same .set-subtabs bar ตั้งค่า › การเรียนรู้ uses: the first, and the one
  // the page opens on, is the stage, the poses and the personas; the second
  // the switches for the main avatar — on the screen, where, whether it
  // talks. Apart, the switches no longer sit under a forty-cell picker
  // nobody scrolls past twice; design first because that is what the page
  // is for (owner: "เอาออกแบบอวตารไว้ก่อนตั้งค่าอวตาร").
  //
  // Words come from avatarText.ts (temporary, see its note); choices go to
  // avatarPrefs.svelte.ts, which the companion reads.
  import Mascot from './Mascot.svelte'
  import Icon from '../Icon.svelte'
  import { i18n } from '../i18n.svelte'
  import { ACCENT, SHELL } from './palette'
  import { FACE, TOP } from './parts'
  import { POSE, type PoseId } from './poses'
  import { avatarText } from './avatarText'
  import {
    heads, HEADS, headOptions, setAvatarPrefs, resetAvatarPrefs, isDefaultPrefs, assistantOptions,
    personas, addPersona, savePersona, usePersona, removePersona, wornPersona, type HeadId,
  } from './avatarPrefs.svelte'
  import { companion, setCompanionOn, setCompanionVoice, setCompanionGreet, setCompanionPlace, setCompanionSize, SIZE_DEFAULT, SIZE_MIN, SIZE_MAX } from './companionSetting.svelte'
  import { desktopBody } from './desktopBody.svelte'
  import { speech, speak, stopSpeechIf } from '../speech.svelte'
  import { openSettingsAt } from '../stores/cockpit.svelte'
  import { TTSStatus, ListTTSVoices } from '../../../wailsjs/go/main/App'

  const text = $derived(avatarText(i18n.locale))
  // Which head the stage is dressing. Opens on the assistant's.
  let editing = $state<HeadId>('assistant')
  const prefs = $derived(heads[editing])
  const opts = $derived(headOptions(editing))
  let tab = $state<'design' | 'main'>('design')
  const POSES = Object.keys(POSE) as PoseId[]
  let previewPose = $state<PoseId>('idle')
  const worn = $derived(wornPersona(prefs))
  const FACES = FACE.filter((f) => f.identity)

  // The voice's notice. The companion is silent when it cannot speak (owner,
  // 12 ก.ย.: "ทำงานไม่ได้ก็เงียบไป มีแจ้งเตือนหน้านี้"), so this page is where
  // the reason is said: the engine's own refusal (TTSStatus, the same line
  // ตั้งค่า › เสียง shows), or — the quieter case — an engine that runs but has
  // no voice for the UI's language, which on a fresh Windows is the usual
  // one: Thai UI, English voices only. Asked once per visit and again when
  // the switch is turned on; the Windows check starts PowerShell, so nothing
  // is shown until it answers.
  let voiceNote = $state<'' | 'checking' | 'engine' | 'lang'>('')
  let voiceReason = $state('')
  async function checkVoice(): Promise<void> {
    voiceNote = 'checking'
    voiceReason = ''
    try {
      const status = await TTSStatus()
      if (status) {
        voiceNote = 'engine'
        voiceReason = status
        return
      }
      const voices = await ListTTSVoices()
      const lang = i18n.locale.toLowerCase()
      const speaks = voices.some((v) => (v.lang ?? '').toLowerCase().startsWith(lang))
      voiceNote = speaks ? '' : 'lang'
    } catch (err) {
      voiceNote = 'engine'
      voiceReason = String(err)
    }
  }
  $effect(() => {
    if (companion.voice) void checkVoice()
  })
  const TRY_KEY = 'avatar-try'
  const trying = $derived(speech.key === TRY_KEY)
  function tryVoice(): void {
    if (trying) {
      stopSpeechIf(TRY_KEY)
      return
    }
    void speak(TRY_KEY, text.voiceTryText, (err) => {
      voiceNote = 'engine'
      voiceReason = err
    })
  }
  $effect(() => () => stopSpeechIf(TRY_KEY))

  // The leads: a straight line from the part on the figure to the panel that
  // changes it, measured off the real boxes rather than drawn where the
  // layout is hoped to be (owner: "เอาผูกกันเลย ไม่ใช่ไปขนาน"). Anchors are
  // points on the 64-unit mascot; the rest is getBoundingClientRect.
  const FIG_PX = 250
  type PartKey = 'top' | 'accent' | 'shell' | 'face'
  const ANCHORS: { key: PartKey; x: number; y: number }[] = [
    { key: 'top', x: 32, y: 4.4 },     // the orb on the crown
    { key: 'accent', x: 51.5, y: 29 }, // the right ear ring
    { key: 'shell', x: 22, y: 44 },    // the body, low left
    { key: 'face', x: 44, y: 34 },     // the screen's lower corner
  ]
  let stageEl: HTMLDivElement | undefined = $state()
  let figEl: HTMLDivElement | undefined = $state()
  const panelEl: Partial<Record<PartKey, HTMLDivElement>> = {}
  let leads = $state<{ x1: number; y1: number; x2: number; y2: number }[]>([])
  let stageBox = $state({ w: 0, h: 0 })

  function measure(): void {
    if (!stageEl || !figEl) return
    const st = stageEl.getBoundingClientRect()
    const fg = figEl.getBoundingClientRect()
    stageBox = { w: st.width, h: st.height }
    const k = FIG_PX / 64
    const next: typeof leads = []
    for (const a of ANCHORS) {
      const panel = panelEl[a.key]
      if (!panel) continue
      const pr = panel.getBoundingClientRect()
      // Stacked (narrow) layout: the panels sit under the figure, in its
      // column, and a lead would cross everything to reach them — draw none.
      if (pr.left < fg.right && pr.right > fg.left) continue
      const x1 = fg.left - st.left + a.x * k
      const y1 = fg.top - st.top + a.y * k
      const left = pr.right <= fg.left + 1
      const x2 = (left ? pr.right : pr.left) - st.left
      const y2 = Math.min(Math.max(y1, pr.top - st.top + 22), pr.bottom - st.top - 22)
      next.push({ x1, y1, x2, y2 })
    }
    leads = next
  }
  $effect(() => {
    if (!stageEl) return
    const ro = new ResizeObserver(() => measure())
    ro.observe(stageEl)
    window.addEventListener('resize', measure)
    const raf = requestAnimationFrame(measure)
    return () => {
      ro.disconnect()
      window.removeEventListener('resize', measure)
      cancelAnimationFrame(raf)
    }
  })
</script>

<div class="avatar-page">
  <h2>{text.title}</h2>
  <p class="d muted avatar-blurb">{text.blurb}</p>

  <div class="set-subtabs" role="tablist" aria-label={text.title}>
    <button type="button" class="set-subtab" role="tab" aria-selected={tab === 'design'} class:active={tab === 'design'} onclick={() => (tab = 'design')}>
      <Icon name="palette" size={14} />
      <span>{text.tabDesign}</span>
    </button>
    <button type="button" class="set-subtab" role="tab" aria-selected={tab === 'main'} class:active={tab === 'main'} onclick={() => (tab = 'main')}>
      <Icon name="slidersHorizontal" size={14} />
      <span>{text.tabMain}</span>
    </button>
  </div>

  {#if tab === 'main'}
  <div class="settings-card">
    <div class="set-row">
      <div class="set-txt">
        <div class="t">{text.onScreen}</div>
        <div class="d">{text.onScreenDesc}</div>
      </div>
      <label class="mswitch">
        <input type="checkbox" checked={companion.on} aria-label={text.onScreen} onchange={(e) => setCompanionOn(e.currentTarget.checked)} />
        <span></span>
      </label>
    </div>
    <!-- Where it lives: this window, or a window of its own on the desktop
         (companionSetting.svelte.ts `place`). A choice, not a replacement —
         the owner wanted both kept and the cost of each said (13 ก.ย. 2026:
         "ทำเป็นตัวเลือก … เขียนรายละเอียดบอกก็พอว่ากินทรัพยากรไม่เท่ากัน"). -->
    <div class="set-row place-row" class:dim={!companion.on}>
      <div class="set-txt">
        <div class="t">{text.place}</div>
        <div class="d">{text.placeDesc}</div>
        <ul class="place-list">
          <li class:now={companion.place === 'window'}><b>{text.placeWindow}</b><span>{text.placeWindowDesc}</span></li>
          <li class:now={companion.place === 'desktop'}><b>{text.placeDesktop}</b><span>{text.placeDesktopDesc}</span></li>
        </ul>
        {#if desktopBody.baking}
          <div class="voice-note soft" role="status">{text.placeBaking}</div>
        {/if}
      </div>
      <div class="seg place-seg" role="radiogroup" aria-label={text.place}>
        <button type="button" class="seg-btn" class:active={companion.place === 'window'} role="radio" aria-checked={companion.place === 'window'} onclick={() => setCompanionPlace('window')}>
          <span class="ic"><Icon name="square" size={13} /></span>{text.placeWindow}
        </button>
        <button type="button" class="seg-btn" class:active={companion.place === 'desktop'} role="radio" aria-checked={companion.place === 'desktop'} onclick={() => setCompanionPlace('desktop')}>
          <span class="ic"><Icon name="monitor" size={13} /></span>{text.placeDesktop}
        </button>
      </div>
    </div>
    <!-- The size: set by dragging the figure's corner, shown here as the
         number it is, with the way back to the default. -->
    <div class="set-row size-row" class:dim={!companion.on}>
      <div class="set-txt">
        <div class="t">{text.size} <span class="tag">{companion.size} px</span></div>
        <div class="d">{text.sizeDesc.replace('{min}', String(SIZE_MIN)).replace('{max}', String(SIZE_MAX))}</div>
      </div>
      {#if companion.size !== SIZE_DEFAULT}
        <button type="button" class="chip" onclick={() => setCompanionSize(SIZE_DEFAULT)}>{text.sizeReset}</button>
      {/if}
    </div>
    <!-- The voice: a row under the figure's own, since it is the figure that
         talks (a hidden companion is a silent one). Below it, why it cannot,
         when it cannot; and a way to hear it, so "on" can be checked here. -->
    <div class="set-row voice-row" class:dim={!companion.on}>
      <div class="set-txt">
        <div class="t">{text.voice}</div>
        <div class="d">{text.voiceDesc}</div>
        {#if companion.voice && voiceNote}
          <div class="voice-note" class:soft={voiceNote === 'checking'} role="status">
            {#if voiceNote === 'checking'}
              {text.voiceChecking}
            {:else}
              <Icon name="alertTriangle" size={13} />
              <span>
                {voiceNote === 'engine' ? text.voiceNoEngine : text.voiceNoLang}
                {#if voiceNote === 'engine'}{voiceReason}{/if}
                <button type="button" class="link" onclick={() => openSettingsAt('voice')}>{text.voiceSettings}</button>
              </span>
            {/if}
          </div>
        {/if}
      </div>
      <div class="set-ctrl voice-ctrl">
        {#if companion.voice}
          <button type="button" class="chip" class:on={trying} onclick={tryVoice}>{trying ? '■' : ''} {text.voiceTry}</button>
        {/if}
        <label class="mswitch">
          <input type="checkbox" checked={companion.voice} aria-label={text.voice} onchange={(e) => setCompanionVoice(e.currentTarget.checked)} />
          <span></span>
        </label>
      </div>
    </div>
    <!-- One step under the voice: whether hello is among what it says. -->
    {#if companion.voice}
      <div class="set-row greet-row" class:dim={!companion.on}>
        <div class="set-txt">
          <div class="t">{text.greet}</div>
          <div class="d">{text.greetDesc}</div>
        </div>
        <label class="mswitch">
          <input type="checkbox" checked={companion.greet} aria-label={text.greet} onchange={(e) => setCompanionGreet(e.currentTarget.checked)} />
          <span></span>
        </label>
      </div>
    {/if}
  </div>

  {:else}
  <div class="avatar-main">
    <span class="star"><Icon name="sparkles" size={14} /></span>
    <span>{text.mainNote}</span>
  </div>

  <!-- which head is on the stage -->
  <div class="who" role="tablist" aria-label={text.mainNote}>
    {#each HEADS as h (h)}
      <button type="button" class="who-card" class:on={editing === h} role="tab" aria-selected={editing === h} onclick={() => (editing = h)}>
        <span class="who-face"><Mascot {...headOptions(h)} size={72} still /></span>
        <span class="who-txt">
          <span class="who-name">{text.heads[h].name}</span>
          <span class="who-where">{text.heads[h].where}</span>
        </span>
        {#if editing === h}<span class="who-now">{text.designing}</span>{/if}
      </button>
    {/each}
  </div>

  <div class="stage" bind:this={stageEl}>
    <svg class="leads" width={stageBox.w} height={stageBox.h} aria-hidden="true">
      {#each leads as l, i (i)}
        <line x1={l.x1} y1={l.y1} x2={l.x2} y2={l.y2} />
        <circle cx={l.x1} cy={l.y1} r="4" />
        <circle cx={l.x2} cy={l.y2} r="3" />
      {/each}
    </svg>

    <!-- left: top light, body -->
    <div class="col left">
      <div class="panel p-top" bind:this={panelEl.top}>
        <h3>{text.top}</h3>
        <p class="hint">{text.parts.top}</p>
        <div class="cells">
          {#each TOP as t (t.id)}
            <button type="button" class="cell" class:on={prefs.top === t.id} title={t.label} aria-label={t.label} onclick={() => setAvatarPrefs({ top: t.id }, editing)}>
              <Mascot {...opts} top={t.id} size={48} still />
            </button>
          {/each}
        </div>
      </div>
      <div class="panel p-shell" bind:this={panelEl.shell}>
        <h3>{text.shell}</h3>
        <p class="hint">{text.parts.shell}</p>
        <div class="cells">
          {#each SHELL as sh (sh.id)}
            <button type="button" class="cell" class:on={prefs.shell === sh.id} title={sh.label} aria-label={sh.label} onclick={() => setAvatarPrefs({ shell: sh.id }, editing)}>
              <Mascot {...opts} shell={sh.id} size={48} still />
            </button>
          {/each}
        </div>
      </div>
    </div>

    <!-- the figure; the leads above are measured from this box -->
    <div class="figure">
      <div class="fig-box" bind:this={figEl}><Mascot {...opts} pose={previewPose} size={FIG_PX} sway /></div>
      {#if !isDefaultPrefs(prefs, editing)}
        <button type="button" class="chip reset" onclick={() => resetAvatarPrefs(editing)}>{text.reset}</button>
      {/if}
    </div>

    <!-- right: accent, resting face -->
    <div class="col right">
      <div class="panel p-accent" bind:this={panelEl.accent}>
        <h3>{text.hue}</h3>
        <p class="hint">{text.parts.hue}</p>
        <div class="cells">
          {#each ACCENT as a (a.id)}
            <button type="button" class="cell" class:on={prefs.accent === a.id} title={a.label} aria-label={a.label} onclick={() => setAvatarPrefs({ accent: a.id }, editing)}>
              <Mascot {...opts} accent={a.id} size={48} still />
            </button>
          {/each}
        </div>
      </div>
      <div class="panel p-face" bind:this={panelEl.face}>
        <h3>{text.face}</h3>
        <p class="hint">{text.parts.face}</p>
        <div class="cells">
          {#each FACES as f (f.id)}
            <button type="button" class="cell" class:on={prefs.face === f.id} title={f.label} aria-label={f.label} onclick={() => setAvatarPrefs({ face: f.id }, editing)}>
              <Mascot {...opts} face={f.id} size={48} still />
            </button>
          {/each}
        </div>
      </div>
    </div>
  </div>

  <!-- every pose the rig has, to see the look move -->
  <div class="poses" role="tablist" aria-label={text.preview}>
    {#each POSES as p (p)}
      <button type="button" class="chip" class:on={previewPose === p} role="tab" aria-selected={previewPose === p} onclick={() => (previewPose = p)}>
        {text.poses[p] ?? p}
      </button>
    {/each}
  </div>

  <div class="group-head"><span class="group-title">{text.personas}</span></div>
  <div class="settings-card personas">
    <p class="hint">{text.personasNote}</p>
    <div class="slots">
      {#each personas.slots as slot, i (i)}
        <div class="slot" class:worn={worn === i}>
          <Mascot {...assistantOptions(slot)} size={64} still />
          <div class="meta">
            <b>{text.persona} {i + 1}</b>
            <span>{worn === i ? text.worn : ''}</span>
          </div>
          <div class="acts">
            <button type="button" class="chip pri" disabled={worn === i} onclick={() => usePersona(i, editing)}>{text.use}</button>
            <button type="button" class="chip" disabled={worn === i} onclick={() => savePersona(i, editing)}>{text.save}</button>
            <button type="button" class="chip" onclick={() => removePersona(i)}>{text.remove}</button>
          </div>
        </div>
      {/each}
      <!-- The + card, always last: what is worn now becomes one more
           persona. Pressed while the look is already kept, it says so
           instead of keeping a twin. -->
      <button type="button" class="slot add" disabled={worn >= 0} onclick={() => addPersona(editing)}>
        <span class="empty"><Icon name="plus" size={22} /></span>
        <span class="meta">
          <b>{text.personaAdd}</b>
          <span>{worn >= 0 ? text.worn : text.personaAddDesc}</span>
        </span>
      </button>
    </div>
  </div>

  {/if}
</div>

<style>
  /* The two places sit as a pair sized to their words: the shared .seg
     stretches its tabs to a column and cuts them, which is right for a
     row of many tabs and wrong for two names that must both read. */
  .place-seg { flex: none; }
  .place-seg .seg-btn { flex: none; min-width: 0; padding: 6px 14px; overflow: visible; }
  .place-row.dim .set-txt, .size-row.dim .set-txt { opacity: .55; }
  .place-list { list-style: none; margin: 6px 0 0; padding: 0; display: grid; gap: 4px; }
  .place-list li { display: grid; grid-template-columns: auto 1fr; gap: 8px; align-items: baseline; font-size: var(--fs-xs); color: var(--text-muted); line-height: 1.45; }
  .place-list b { font-weight: 600; color: var(--text-secondary); white-space: nowrap; }
  .place-list li.now b { color: var(--text-primary); }
  .place-list li.now span { color: var(--text-secondary); }
  /* This page is a stage, not a column of prose: it asks the settings pane
     for more than the 760px a settings row wants (style.css .settings-inner,
     which reads --content-max), and lays itself out by the width it gets. */
  :global(.settings-inner:has(.avatar-page)) { --content-max: 1120px; }
  .avatar-page { container-type: inline-size; }

  .avatar-blurb { margin: -6px 0 14px; max-width: 64ch; }
  .avatar-main {
    display: flex; gap: 10px; align-items: flex-start; margin: 0 0 18px; padding: 10px 14px; border-radius: 12px;
    background: color-mix(in srgb, var(--interactive) 12%, transparent); border: 1px solid color-mix(in srgb, var(--interactive) 35%, transparent);
    font-size: var(--fs-sm);
  }
  .avatar-main .star { color: var(--interactive); display: inline-flex; margin-top: 2px; }

  /* ---- the two heads: which one the stage is dressing ---- */
  .who { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin: 0 0 18px; }
  .who-card {
    appearance: none; font: inherit; text-align: left; color: inherit; cursor: pointer; position: relative;
    display: flex; align-items: center; gap: 14px; padding: 12px 14px; border-radius: 14px; min-width: 0;
    border: 1px solid var(--border-subtle); background: var(--surface-panel);
  }
  .who-card:hover { border-color: var(--border-strong); }
  .who-card.on { border-color: var(--interactive); background: var(--surface-raised); box-shadow: 0 0 0 3px color-mix(in srgb, var(--interactive) 18%, transparent); }
  .who-face { flex: none; }
  .who-face :global(.mascot) { display: block; }
  .who-txt { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
  .who-name { font-size: var(--fs-lg); font-weight: 600; }
  .who-where { font-size: var(--fs-xs); color: var(--text-muted); }
  .who-now { position: absolute; top: 10px; right: 12px; font-size: var(--fs-2xs); color: var(--interactive); font-weight: 600; }

  /* ---- the stage: two columns of panels with the figure between them ---- */
  .stage {
    position: relative; display: grid; grid-template-columns: minmax(0, 1fr) 300px minmax(0, 1fr);
    gap: 18px; align-items: stretch;
  }
  .stage > .leads { position: absolute; left: 0; top: 0; pointer-events: none; overflow: visible; z-index: 1; }
  .stage > .leads line { stroke: var(--interactive); stroke-width: 1.5; stroke-dasharray: 5 4; opacity: .8; }
  .stage > .leads circle { fill: var(--interactive); stroke: var(--surface-app); stroke-width: 2; }
  /* Each side is its own column, its two panels pushed to the corners, so a
     tall panel on one side never drags the other side's panels down with it. */
  .col { display: flex; flex-direction: column; justify-content: space-between; gap: 18px; min-width: 0; }
  .panel { background: var(--surface-panel); border: 1px solid var(--border-subtle); border-radius: 14px; padding: 12px 14px; min-width: 0; }
  .panel h3 { margin: 0 0 2px; font-size: var(--fs-md); }
  .hint { color: var(--text-muted); font-size: var(--fs-xs); margin: 0 0 10px; }
  .cells { display: flex; flex-wrap: wrap; gap: 6px; }
  .cell {
    padding: 3px; display: grid; place-items: center; border-radius: var(--r-sm); cursor: pointer;
    border: 1px solid var(--border-subtle); background: var(--surface-sunken);
  }
  .cell:hover { border-color: var(--border-strong); }
  .cell.on { border-color: var(--interactive); background: var(--surface-raised); }
  .cell :global(.mascot) { display: block; }

  /* The figure: on the centre line, at mid-height of whatever the columns grow to. */
  .figure { align-self: center; justify-self: center; display: flex; flex-direction: column; align-items: center; gap: 8px; }
  .fig-box { position: relative; }

  .chip {
    appearance: none; font: inherit; font-size: var(--fs-xs); padding: 4px 10px; border-radius: 999px; cursor: pointer;
    border: 1px solid var(--border-subtle); background: var(--surface-sunken); color: var(--text-secondary);
  }
  .chip:hover { border-color: var(--border-strong); color: var(--text-primary); }
  .chip.on { border-color: var(--interactive); background: var(--surface-raised); color: var(--text-primary); }
  .chip.pri { background: var(--interactive); color: var(--text-on-interactive); border-color: var(--interactive); }
  .chip.pri:disabled { opacity: .6; cursor: default; }
  .poses { display: flex; gap: 6px; margin: 16px 0 28px; flex-wrap: wrap; justify-content: center; }
  /* ---- the voice row ---- */
  .voice-row.dim .set-txt, .greet-row.dim .set-txt { opacity: .55; }
  .greet-row { padding-left: 22px; }
  .voice-ctrl { display: flex; align-items: center; gap: 10px; }
  .voice-note {
    display: flex; gap: 6px; align-items: flex-start; margin-top: 8px; padding: 6px 10px; border-radius: 8px;
    font-size: var(--fs-xs); line-height: 1.45; color: var(--status-warn);
    background: color-mix(in srgb, var(--status-warn) 10%, transparent); border: 1px solid color-mix(in srgb, var(--status-warn) 30%, transparent);
  }
  .voice-note :global(.icon) { flex: none; margin-top: 2px; }
  .voice-note.soft { color: var(--text-muted); background: transparent; border-color: var(--border-subtle); }
  .voice-note .link { appearance: none; background: none; border: 0; padding: 0; font: inherit; color: var(--interactive); text-decoration: underline; cursor: pointer; }

  /* ---- personas ---- */
  .personas { padding: 12px 16px 14px; }
  .personas .hint { margin: 0 0 12px; }
  .slots { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
  .slot { display: flex; align-items: center; gap: 12px; background: var(--surface-raised); border: 1px solid var(--border-subtle); border-radius: 12px; padding: 10px 12px; min-width: 0; }
  .slot.worn { border-color: var(--interactive); }
  .slot :global(.mascot) { flex: none; }
  .empty { width: 64px; height: 64px; flex: none; border: 1px dashed var(--border-subtle); border-radius: 12px; display: flex; align-items: center; justify-content: center; color: var(--text-dim); font-size: var(--fs-xs); }
  .slot.add { appearance: none; font: inherit; text-align: left; border-style: dashed; cursor: pointer; color: inherit; }
  .slot.add:hover:not(:disabled) { border-color: var(--border-strong); }
  .slot.add:hover:not(:disabled) .empty { border-color: var(--interactive); color: var(--interactive); }
  .slot.add:disabled { cursor: default; opacity: .6; }
  .chip:disabled:not(.pri) { opacity: .5; cursor: default; }
  .meta { flex: 1; min-width: 0; }
  .meta b { display: block; font-size: var(--fs-sm); }
  .meta span { font-size: var(--fs-xs); color: var(--text-muted); }
  .acts { display: flex; flex-direction: column; gap: 4px; }

  /* Narrower than three comfortable columns: the figure on top, the four
     panels in two columns beneath it, and no leads (the measure() rule). */
  @container (max-width: 880px) {
    .stage { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
    .figure { grid-column: 1 / -1; grid-row: 1; }
    .col { justify-content: flex-start; }
    .col.left { grid-column: 1; grid-row: 2; }
    .col.right { grid-column: 2; grid-row: 2; }
    .slots { grid-template-columns: 1fr; }
    .who { grid-template-columns: 1fr; }
  }
  @container (max-width: 560px) {
    .stage { grid-template-columns: minmax(0, 1fr); }
    .col.left { grid-column: 1; grid-row: 2; }
    .col.right { grid-column: 1; grid-row: 3; }
  }
</style>
