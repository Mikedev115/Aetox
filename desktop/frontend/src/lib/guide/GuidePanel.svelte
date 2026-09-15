<script lang="ts">
  // The guide's settings, as the bubble's second face.
  //
  // One face explains what you are looking at; this one is the guide itself —
  // the brain it thinks with, the language it answers in, whether it reads
  // aloud, and the walks it can take you on. A separate component because the
  // bubble is already the busiest thing in the feature, and because adding a
  // setting should mean editing a panel, not a figure that also walks and
  // talks (owner, 15 ก.ย. 2026: *"อีกหน้าตั้งค่าไกด์ … ทำให้เพิ่มได้ง่ายด้วย
  // ในอนาคต"*).
  //
  // Everything it writes goes through guidePrefs, and everything it offers
  // comes from presets.ts — so a new knob is one field there plus one row
  // here, and a new shortcut is one row in presets.ts and nothing here at all.
  import Icon from '../Icon.svelte'
  import { t, i18n, localeNames, type Locale } from '../i18n.svelte'
  import { guidePrefs, setGuidePref, resetGuidePrefs, type ThinkLevel } from './guidePrefs.svelte'
  import { GUIDE_PRESETS, presetLabelKey } from './presets'
  import { cockpit } from '../stores/cockpit.svelte'
  import { EnabledProviders, ListModelsForProvider } from '../../../wailsjs/go/main/App'

  let { onPick, onClose }: { onPick: (mapId: string) => void; onClose: () => void } = $props()

  let providers = $state<string[]>([])
  let models = $state<string[]>([])

  // The brain the guide will actually use, said plainly: its own when one is
  // pinned, the chat's otherwise. A settings page that shows a blank where the
  // answer is "the same as over there" teaches nothing.
  const brainNow = $derived(
    guidePrefs.provider
      ? `${guidePrefs.provider}${guidePrefs.model ? ' · ' + guidePrefs.model : ''}`
      : cockpit.model.provider
        ? t('guide.set.brainShared', { name: cockpit.model.provider })
        : t('guide.set.brainNone'),
  )

  $effect(() => {
    EnabledProviders()
      .then((p) => (providers = p ?? []))
      .catch(() => (providers = []))
  })

  $effect(() => {
    const p = guidePrefs.provider
    if (!p) {
      models = []
      return
    }
    ListModelsForProvider(p)
      .then((m) => (models = m ?? []))
      .catch(() => (models = []))
  })

  const THINK: ThinkLevel[] = ['low', 'medium', 'high']
</script>

<div class="gp">
  <div class="gp-head">
    <span class="gp-title"><Icon name="settings" size={13} /> {t('guide.set.title')}</span>
    <button type="button" class="gp-x" onclick={onClose} aria-label={t('guide.set.back')}>
      <Icon name="arrowLeft" size={13} />
    </button>
  </div>

  <!-- Shortcuts first: it is what most people opened this for, and every one
       of them is a place on the map, so they work with no model at all. -->
  <div class="gp-sect">{t('guide.set.shortcuts')}</div>
  <div class="gp-presets">
    {#each GUIDE_PRESETS as p (p.id)}
      <button type="button" class="ctrl mini" class:primary={p.lead} onclick={() => onPick(p.to)}>
        {t(presetLabelKey(p) as never)}
      </button>
    {/each}
  </div>

  <div class="gp-sect">{t('guide.set.brain')}</div>
  <div class="gp-now">{brainNow}</div>
  <label class="gp-row">
    <span>{t('guide.set.provider')}</span>
    <select
      id="guide-provider"
      value={guidePrefs.provider}
      onchange={(e) => {
        setGuidePref('provider', e.currentTarget.value)
        setGuidePref('model', '')
      }}
    >
      <option value="">{t('guide.set.sameAsChat')}</option>
      {#each providers as p (p)}<option value={p}>{p}</option>{/each}
    </select>
  </label>
  {#if guidePrefs.provider}
    <label class="gp-row">
      <span>{t('guide.set.model')}</span>
      <select id="guide-model" value={guidePrefs.model} onchange={(e) => setGuidePref('model', e.currentTarget.value)}>
        <option value="">{t('guide.set.modelDefault')}</option>
        {#each models as m (m)}<option value={m}>{m}</option>{/each}
      </select>
    </label>
  {/if}
  <label class="gp-row">
    <span>{t('guide.set.think')}</span>
    <select id="guide-think" value={guidePrefs.think} onchange={(e) => setGuidePref('think', e.currentTarget.value as ThinkLevel)}>
      {#each THINK as lv (lv)}<option value={lv}>{t(`guide.set.think.${lv}` as never)}</option>{/each}
    </select>
  </label>
  <p class="gp-note">{t('guide.set.brainNote')}</p>

  <div class="gp-sect">{t('guide.set.talking')}</div>
  <label class="gp-row">
    <span>{t('guide.set.lang')}</span>
    <select id="guide-lang" value={guidePrefs.lang} onchange={(e) => setGuidePref('lang', e.currentTarget.value as 'auto' | Locale)}>
      <option value="auto">{t('guide.set.langAuto', { name: localeNames[i18n.locale] ?? i18n.locale })}</option>
      {#each Object.entries(localeNames) as [code, name] (code)}<option value={code}>{name}</option>{/each}
    </select>
  </label>
  <label class="gp-row gp-check">
    <input id="guide-voice" type="checkbox" checked={guidePrefs.voice} onchange={(e) => setGuidePref('voice', e.currentTarget.checked)} />
    <span>{t('guide.set.voice')}</span>
  </label>

  <button type="button" class="ctrl mini gp-reset" onclick={resetGuidePrefs}>{t('guide.set.reset')}</button>
</div>

<style>
  .gp { display: flex; flex-direction: column; gap: 6px; }
  .gp-head { display: flex; align-items: center; gap: 8px; margin-bottom: 2px; }
  .gp-title { display: inline-flex; align-items: center; gap: 6px; font-weight: 700; }
  .gp-x {
    margin-left: auto; border: 0; background: none; color: var(--text-muted);
    cursor: pointer; padding: 2px; border-radius: var(--r-xs); display: grid; place-items: center;
  }
  .gp-x:hover { color: var(--text-primary); background: var(--surface-hover); }
  .gp-sect {
    margin-top: 6px; font-size: var(--fs-2xs); font-weight: 700; letter-spacing: .04em;
    text-transform: uppercase; color: var(--text-dim);
  }
  .gp-presets { display: flex; flex-wrap: wrap; gap: 5px; }
  .gp-now { font-size: var(--fs-xs); color: var(--badge-amber-text); }
  .gp-row { display: flex; align-items: center; gap: 8px; font-size: var(--fs-xs); }
  .gp-row > span { flex: none; min-width: 74px; color: var(--text-muted); }
  .gp-row select {
    flex: 1; min-width: 0; background: var(--surface-sunken); color: var(--text-primary);
    border: 1px solid var(--border-subtle); border-radius: 6px; padding: 3px 6px;
    font: inherit; font-size: var(--fs-xs);
  }
  .gp-check { gap: 6px; }
  .gp-check > span { min-width: 0; color: var(--text-primary); }
  .gp-note { margin: 2px 0 0; font-size: var(--fs-2xs); color: var(--text-dim); line-height: 1.4; }
  .gp-reset { align-self: flex-start; margin-top: 8px; }
</style>
