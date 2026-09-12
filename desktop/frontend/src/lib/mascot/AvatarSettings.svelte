<script lang="ts">
  // ตั้งค่า › ส่วนบุคคล › อวตาร — the assistant's mascot, laid out like a game's
  // character screen (owner, 12 ก.ย.: "ทำหน้าเหมือนตั้งค่าเกม แบบแยกส่วน อวตาร
  // อยู่ตรงกลาง แล้วชี้เป็นส่วน ๆ อันไหนคือปรับอันไหน").
  //
  // The figure sits in the middle. Four numbered marks are pinned on the parts
  // themselves — the crown, an ear, the body, the screen — and a dashed lead
  // runs from each to the panel that changes it, so nobody has to guess which
  // row is which. Every cell in a panel is the whole outcome, still (a picker
  // of twenty-five breathing together stuttered), and reads the same catalogue
  // the rig draws from: a shell or a top light appended in palette.ts /
  // parts.ts appears here untouched.
  //
  // Personas — three slots to keep a look in and switch between — exist for
  // the agents a user will design later: a persona is a look ready to be
  // handed to one. The banner above says whose avatar this is: the
  // assistant's, the same on every desk (COMPANY.md, one face).
  //
  // Words come from avatarText.ts (temporary, see its note); choices go to
  // avatarPrefs.svelte.ts, which the companion reads.
  import Mascot from './Mascot.svelte'
  import Icon from '../Icon.svelte'
  import { i18n } from '../i18n.svelte'
  import { SHELL } from './palette'
  import { FACE, TOP } from './parts'
  import { HUES } from '../agentFace'
  import { avatarText } from './avatarText'
  import {
    avatarPrefs, setAvatarPrefs, resetAvatarPrefs, isDefaultPrefs, assistantOptions,
    personas, savePersona, usePersona, clearPersona, wornPersona, PERSONA_SLOTS,
  } from './avatarPrefs.svelte'
  import { companion, setCompanionOn } from './companionSetting.svelte'

  const text = $derived(avatarText(i18n.locale))
  const opts = $derived(assistantOptions(avatarPrefs))
  const PREVIEW_POSES = ['idle', 'greeting', 'typing', 'answering', 'success'] as const
  let previewPose = $state<(typeof PREVIEW_POSES)[number]>('idle')
  const worn = $derived(wornPersona(avatarPrefs))

  /** The four marks on the figure, in the figure's own 300×300 box: where the
   *  part is on a 250px mascot drawn 25px in from the left, 16px down. Each
   *  lead runs from the mark to the side its panel is on. */
  const MARKS = [
    { n: 1, x: 150, y: 34, side: 'left' },   // the crown
    { n: 2, x: 228, y: 122, side: 'right' }, // an ear
    { n: 3, x: 90, y: 128, side: 'left' },   // the body
    { n: 4, x: 176, y: 110, side: 'right' }, // the screen
  ] as const
</script>

<h2>{text.title}</h2>
<p class="d muted avatar-blurb">{text.blurb}</p>

<div class="avatar-main">
  <span class="star"><Icon name="sparkles" size={14} /></span>
  <span>{text.mainNote}</span>
</div>

<div class="stage">
  <!-- ① top light — ③ body -->
  <div class="panel tl">
    <h3><span class="n">1</span>{text.top}</h3>
    <p class="hint">{text.parts.top}</p>
    <div class="ag-parts">
      {#each TOP as t (t.id)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.top === t.id} title={t.label} aria-label={t.label} onclick={() => setAvatarPrefs({ top: t.id })}>
          <Mascot {...opts} top={t.id} size={52} still />
        </button>
      {/each}
    </div>
  </div>
  <div class="panel bl">
    <h3><span class="n">3</span>{text.shell}</h3>
    <p class="hint">{text.parts.shell}</p>
    <div class="ag-parts">
      {#each SHELL as sh (sh.id)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.shell === sh.id} title={sh.label} aria-label={sh.label} onclick={() => setAvatarPrefs({ shell: sh.id })}>
          <Mascot {...opts} shell={sh.id} size={52} still />
        </button>
      {/each}
    </div>
  </div>

  <!-- the figure, with its marks and leads -->
  <div class="figure">
    <svg class="leads" viewBox="0 0 300 300" preserveAspectRatio="none" aria-hidden="true">
      <g fill="none" stroke="var(--interactive)" stroke-width="1.5" stroke-dasharray="4 4" opacity=".75">
        {#each MARKS as m (m.n)}
          <path d="M{m.x} {m.y} H {m.side === 'left' ? 0 : 300}" />
        {/each}
      </g>
    </svg>
    <div class="fig-box"><Mascot {...opts} pose={previewPose} size={250} sway /></div>
    {#each MARKS as m (m.n)}
      <span class="mark" style="left:{m.x - 10}px; top:{m.y - 10}px" aria-hidden="true">{m.n}</span>
    {/each}
    <div class="poses" role="tablist" aria-label={text.preview}>
      {#each PREVIEW_POSES as p (p)}
        <button type="button" class="ctrl tiny" class:on={previewPose === p} role="tab" aria-selected={previewPose === p} onclick={() => (previewPose = p)}>
          {text.poses[p]}
        </button>
      {/each}
    </div>
    {#if !isDefaultPrefs(avatarPrefs)}
      <button type="button" class="ctrl tiny reset" onclick={resetAvatarPrefs}>{text.reset}</button>
    {/if}
  </div>

  <!-- ② accent — ④ resting face -->
  <div class="panel tr">
    <h3><span class="n">2</span>{text.hue}</h3>
    <p class="hint">{text.parts.hue}</p>
    <div class="ag-parts">
      <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.hue === null} title={text.hueBrand} aria-label={text.hueBrand} onclick={() => setAvatarPrefs({ hue: null })}>
        <Mascot {...opts} hue={undefined} size={44} still />
      </button>
      {#each HUES as h (h)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.hue === h} title={`${h}°`} aria-label={`${h}°`} onclick={() => setAvatarPrefs({ hue: h })}>
          <Mascot {...opts} hue={h} size={44} still />
        </button>
      {/each}
    </div>
  </div>
  <div class="panel br">
    <h3><span class="n">4</span>{text.face}</h3>
    <p class="hint">{text.parts.face}</p>
    <div class="ag-parts">
      {#each FACE.filter((f) => f.identity) as f (f.id)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.face === f.id} title={f.label} aria-label={f.label} onclick={() => setAvatarPrefs({ face: f.id })}>
          <Mascot {...opts} face={f.id} size={52} still />
        </button>
      {/each}
    </div>
  </div>
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
            <button type="button" class="ctrl tiny pri" disabled={worn === i} onclick={() => usePersona(i)}>{text.use}</button>
          {/if}
          <button type="button" class="ctrl tiny" onclick={() => savePersona(i)}>{text.save}</button>
          {#if slot}
            <button type="button" class="ctrl tiny" onclick={() => clearPersona(i)}>{text.clear}</button>
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

<style>
  .avatar-blurb { margin: -6px 0 14px; max-width: 64ch; }
  .avatar-main {
    display: flex; gap: 10px; align-items: flex-start; margin: 0 0 18px; padding: 10px 14px; border-radius: 12px;
    background: color-mix(in srgb, var(--interactive) 12%, transparent); border: 1px solid color-mix(in srgb, var(--interactive) 35%, transparent);
    font-size: var(--fs-sm);
  }
  .avatar-main .star { color: var(--interactive); display: inline-flex; margin-top: 2px; }

  /* ---- the stage ---- */
  .stage { display: grid; grid-template-columns: minmax(0, 1fr) 300px minmax(0, 1fr); grid-template-rows: auto auto; gap: 18px 0; align-items: start; }
  .panel { background: var(--surface-panel); border: 1px solid var(--border-subtle); border-radius: 14px; padding: 12px 14px; }
  .panel.tl { grid-column: 1; grid-row: 1; margin-right: 18px; }
  .panel.bl { grid-column: 1; grid-row: 2; margin-right: 18px; }
  .panel.tr { grid-column: 3; grid-row: 1; margin-left: 18px; }
  .panel.br { grid-column: 3; grid-row: 2; margin-left: 18px; }
  .panel h3 { margin: 0 0 2px; font-size: var(--fs-md); display: flex; align-items: center; gap: 8px; }
  .n, .mark { width: 20px; height: 20px; border-radius: 50%; background: var(--interactive); color: var(--text-on-interactive); font-size: 11px; font-weight: 700; display: inline-flex; align-items: center; justify-content: center; }
  .hint { color: var(--text-muted); font-size: var(--fs-xs); margin: 0 0 10px; }
  .avatar-part { padding: 3px; }
  .avatar-part :global(.mascot) { display: block; }

  .figure { grid-column: 2; grid-row: 1 / span 2; position: relative; display: flex; flex-direction: column; align-items: center; padding-top: 16px; }
  .figure .leads { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; overflow: visible; }
  .fig-box { position: relative; }
  .mark { position: absolute; box-shadow: 0 0 0 3px var(--surface-app); }
  .poses { display: flex; gap: 6px; margin-top: 6px; flex-wrap: wrap; justify-content: center; }
  .poses .on { border-color: var(--interactive); background: var(--surface-raised); }
  .reset { margin-top: 8px; }

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
  .acts .pri { background: var(--interactive); color: var(--text-on-interactive); border-color: var(--interactive); }
  .acts .pri:disabled { opacity: .6; cursor: default; }
  .avatar-note { margin: 12px 2px 0; }

  /* narrow: the figure first, then the panels in a column */
  @media (max-width: 900px) {
    .stage { grid-template-columns: minmax(0, 1fr); grid-template-rows: none; }
    .figure { grid-column: 1; grid-row: 1; }
    .figure .leads { display: none; }
    .panel { grid-column: 1 !important; grid-row: auto !important; margin: 0 !important; }
    .slots { grid-template-columns: 1fr; }
  }
</style>
