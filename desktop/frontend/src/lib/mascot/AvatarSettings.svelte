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
  // Personas — six slots to keep a look in and switch between — exist for
  // the agents a user will design later: a persona is a look ready to be
  // handed to one. The banner above says whose avatar this is: the
  // assistant's, the same on every desk (COMPANY.md, one face).
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
    avatarPrefs, setAvatarPrefs, resetAvatarPrefs, isDefaultPrefs, assistantOptions,
    personas, savePersona, usePersona, clearPersona, wornPersona, PERSONA_SLOTS,
  } from './avatarPrefs.svelte'
  import { companion, setCompanionOn } from './companionSetting.svelte'

  const text = $derived(avatarText(i18n.locale))
  const opts = $derived(assistantOptions(avatarPrefs))
  const POSES = Object.keys(POSE) as PoseId[]
  let previewPose = $state<PoseId>('idle')
  const worn = $derived(wornPersona(avatarPrefs))
  const FACES = FACE.filter((f) => f.identity)

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

  <div class="avatar-main">
    <span class="star"><Icon name="sparkles" size={14} /></span>
    <span>{text.mainNote}</span>
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
            <button type="button" class="cell" class:on={avatarPrefs.top === t.id} title={t.label} aria-label={t.label} onclick={() => setAvatarPrefs({ top: t.id })}>
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
            <button type="button" class="cell" class:on={avatarPrefs.shell === sh.id} title={sh.label} aria-label={sh.label} onclick={() => setAvatarPrefs({ shell: sh.id })}>
              <Mascot {...opts} shell={sh.id} size={48} still />
            </button>
          {/each}
        </div>
      </div>
    </div>

    <!-- the figure; the leads above are measured from this box -->
    <div class="figure">
      <div class="fig-box" bind:this={figEl}><Mascot {...opts} pose={previewPose} size={FIG_PX} sway /></div>
      {#if !isDefaultPrefs(avatarPrefs)}
        <button type="button" class="chip reset" onclick={resetAvatarPrefs}>{text.reset}</button>
      {/if}
    </div>

    <!-- right: accent, resting face -->
    <div class="col right">
      <div class="panel p-accent" bind:this={panelEl.accent}>
        <h3>{text.hue}</h3>
        <p class="hint">{text.parts.hue}</p>
        <div class="cells">
          {#each ACCENT as a (a.id)}
            <button type="button" class="cell" class:on={avatarPrefs.accent === a.id} title={a.label} aria-label={a.label} onclick={() => setAvatarPrefs({ accent: a.id })}>
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
            <button type="button" class="cell" class:on={avatarPrefs.face === f.id} title={f.label} aria-label={f.label} onclick={() => setAvatarPrefs({ face: f.id })}>
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
          {#if slot}
            <Mascot {...assistantOptions(slot)} size={64} still />
          {:else}
            <div class="empty">{text.personaEmpty}</div>
          {/if}
          <div class="meta">
            <b>{text.persona} {i + 1}</b>
            <span>{worn === i ? text.worn : slot ? '' : text.personaEmpty}</span>
          </div>
          <div class="acts">
            {#if slot}
              <button type="button" class="chip pri" disabled={worn === i} onclick={() => usePersona(i)}>{text.use}</button>
            {/if}
            <button type="button" class="chip" onclick={() => savePersona(i)}>{text.save}</button>
            {#if slot}
              <button type="button" class="chip" onclick={() => clearPersona(i)}>{text.clear}</button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  </div>

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
  </div>

  <p class="d muted avatar-note">{text.agentsNote}</p>
</div>

<style>
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

  /* ---- personas ---- */
  .personas { padding: 12px 16px 14px; }
  .personas .hint { margin: 0 0 12px; }
  .slots { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
  .slot { display: flex; align-items: center; gap: 12px; background: var(--surface-raised); border: 1px solid var(--border-subtle); border-radius: 12px; padding: 10px 12px; min-width: 0; }
  .slot.worn { border-color: var(--interactive); }
  .slot :global(.mascot) { flex: none; }
  .empty { width: 64px; height: 64px; flex: none; border: 1px dashed var(--border-subtle); border-radius: 12px; display: flex; align-items: center; justify-content: center; color: var(--text-dim); font-size: var(--fs-xs); }
  .meta { flex: 1; min-width: 0; }
  .meta b { display: block; font-size: var(--fs-sm); }
  .meta span { font-size: var(--fs-xs); color: var(--text-muted); }
  .acts { display: flex; flex-direction: column; gap: 4px; }
  .avatar-note { margin: 12px 2px 0; }

  /* Narrower than three comfortable columns: the figure on top, the four
     panels in two columns beneath it, and no leads (the measure() rule). */
  @container (max-width: 880px) {
    .stage { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
    .figure { grid-column: 1 / -1; grid-row: 1; }
    .col { justify-content: flex-start; }
    .col.left { grid-column: 1; grid-row: 2; }
    .col.right { grid-column: 2; grid-row: 2; }
    .slots { grid-template-columns: 1fr; }
  }
  @container (max-width: 560px) {
    .stage { grid-template-columns: minmax(0, 1fr); }
    .col.left { grid-column: 1; grid-row: 2; }
    .col.right { grid-column: 1; grid-row: 3; }
  }
</style>
