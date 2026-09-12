<script lang="ts">
  // ตั้งค่า › ส่วนบุคคล › อวตาร — the page for the assistant's mascot.
  //
  // The owner's point (12 ก.ย.): a mascot with four dials and no page to turn
  // them is a mascot nobody can change. This is that page: a live preview, the
  // on-screen switch, and one row per dial, each cell drawn as the WHOLE
  // outcome rather than a swatch — the same rule the agent-face picker in
  // Settings follows, because a colour chip promises a quarter of what a hue
  // moves. Every row reads the same catalogue the rig draws from, so a shell
  // or a top light appended in palette.ts / parts.ts shows up here untouched.
  //
  // Words come from avatarText.ts (temporary, see its note); choices go to
  // avatarPrefs.svelte.ts, which the companion reads.
  import Mascot from './Mascot.svelte'
  import { i18n } from '../i18n.svelte'
  import { SHELL } from './palette'
  import { FACE, TOP } from './parts'
  import { HUES } from '../agentFace'
  import { avatarText } from './avatarText'
  import { avatarPrefs, setAvatarPrefs, resetAvatarPrefs, isDefaultPrefs, assistantOptions } from './avatarPrefs.svelte'
  import { companion, setCompanionOn } from './companionSetting.svelte'

  const text = $derived(avatarText(i18n.locale))
  const opts = $derived(assistantOptions(avatarPrefs))
  const PREVIEW_POSES = ['idle', 'greeting', 'typing', 'answering', 'success'] as const
  let previewPose = $state<(typeof PREVIEW_POSES)[number]>('idle')
</script>

<h2>{text.title}</h2>
<p class="d muted avatar-blurb">{text.blurb}</p>

<div class="group-head"><span class="group-title">{text.preview}</span></div>
<div class="settings-card avatar-card">
  <div class="avatar-stage">
    <Mascot {...opts} pose={previewPose} size={184} sway />
  </div>
  <div class="avatar-poses" role="tablist" aria-label={text.preview}>
    {#each PREVIEW_POSES as p (p)}
      <button type="button" class="ctrl tiny" class:on={previewPose === p} role="tab" aria-selected={previewPose === p} onclick={() => (previewPose = p)}>
        {text.poses[p]}
      </button>
    {/each}
  </div>
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

<div class="group-head">
  <span class="group-title">{text.shell}</span>
</div>
<div class="settings-card avatar-card">
  <div class="pp-field">
    <div class="ag-parts">
      {#each SHELL as sh (sh.id)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.shell === sh.id} title={sh.label} aria-label={sh.label} onclick={() => setAvatarPrefs({ shell: sh.id })}>
          <Mascot {...opts} shell={sh.id} size={56} />
        </button>
      {/each}
    </div>
  </div>

  <div class="pp-field">
    <span class="eyebrow">{text.hue}</span>
    <div class="ag-parts">
      <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.hue === null} title={text.hueBrand} aria-label={text.hueBrand} onclick={() => setAvatarPrefs({ hue: null })}>
        <Mascot {...opts} hue={undefined} size={44} />
      </button>
      {#each HUES as h (h)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.hue === h} title={`${h}°`} aria-label={`${h}°`} onclick={() => setAvatarPrefs({ hue: h })}>
          <Mascot {...opts} hue={h} size={44} />
        </button>
      {/each}
    </div>
  </div>

  <div class="pp-field">
    <span class="eyebrow">{text.top}</span>
    <div class="ag-parts">
      {#each TOP as t (t.id)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.top === t.id} title={t.label} aria-label={t.label} onclick={() => setAvatarPrefs({ top: t.id })}>
          <Mascot {...opts} top={t.id} size={44} />
        </button>
      {/each}
    </div>
  </div>

  <div class="pp-field">
    <span class="eyebrow">{text.face}</span>
    <div class="ag-parts">
      {#each FACE.filter((f) => f.identity) as f (f.id)}
        <button type="button" class="ag-part avatar-part" class:on={avatarPrefs.face === f.id} title={f.label} aria-label={f.label} onclick={() => setAvatarPrefs({ face: f.id })}>
          <Mascot {...opts} face={f.id} size={44} />
        </button>
      {/each}
    </div>
  </div>

  {#if !isDefaultPrefs(avatarPrefs)}
    <div class="avatar-reset">
      <button type="button" class="ctrl tiny" onclick={resetAvatarPrefs}>{text.reset}</button>
    </div>
  {/if}
</div>

<p class="d muted avatar-note">{text.agentsNote}</p>

<style>
  .avatar-blurb { margin: -6px 0 14px; max-width: 64ch; }
  .avatar-card { padding-bottom: 8px; }
  .avatar-stage { display: flex; justify-content: center; padding: 18px 0 6px; }
  .avatar-poses { display: flex; gap: 6px; justify-content: center; flex-wrap: wrap; padding: 0 16px 12px; }
  .avatar-poses .on { border-color: var(--interactive); background: var(--surface-raised); }
  .avatar-card .pp-field { padding: 10px 16px; }
  .avatar-card .pp-field + .pp-field { border-top: 1px solid var(--border-subtle); }
  .avatar-part { padding: 4px; }
  .avatar-part :global(.mascot) { display: block; }
  .avatar-reset { display: flex; justify-content: flex-end; padding: 6px 16px 8px; }
  .avatar-note { margin: 12px 2px 0; }
</style>
