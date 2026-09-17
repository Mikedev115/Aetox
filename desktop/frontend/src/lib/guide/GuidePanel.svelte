<script lang="ts">
  // Compact nested settings for the guide. The gear opens this as a floating
  // menu beside the figure; each top-level row drills into one focused page.
  import Icon from '../Icon.svelte'
  import { t, i18n, localeNames, type Locale } from '../i18n.svelte'
  import {
    guidePrefs,
    setGuidePref,
    resetGuidePrefs,
    GUIDE_SIZE_MIN,
    GUIDE_SIZE_MAX,
    type ThinkLevel,
  } from './guidePrefs.svelte'
  import { GUIDE_PRESETS, presetLabelKey } from './presets'
  import { cockpit, openSettingsAt } from '../stores/cockpit.svelte'
  import {
    EnabledProviders,
    ListModelsForProvider,
    SupportedThinkLevelsFor,
    ListTTSEngines,
    ListTTSVoices,
    RefreshTTSVoices,
    TTSStatus,
  } from '../../../wailsjs/go/main/App'
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime'

  let { onPick, onClose }: { onPick: (mapId: string) => void; onClose: () => void } = $props()

  type Page = 'root' | 'brain' | 'talking' | 'appearance' | 'walks'
  let page = $state<Page>('root')
  let providers = $state<string[]>([])
  let models = $state<string[]>([])
  let thinkLevels = $state<string[]>([])
  type VoiceState = 'idle' | 'checking' | 'ready' | 'engine' | 'language'
  let voiceState = $state<VoiceState>('idle')
  let voiceReason = $state('')
  let voiceEngine = $state('')
  let voiceCheckSeq = 0

  const effectiveProvider = $derived(guidePrefs.provider || cockpit.model.provider || '')
  const effectiveModel = $derived(guidePrefs.provider ? guidePrefs.model : (cockpit.model.modelName || ''))
  const brainNow = $derived(
    guidePrefs.provider
      ? `${guidePrefs.provider}${guidePrefs.model ? ' · ' + guidePrefs.model : ''}`
      : cockpit.model.provider
        ? t('guide.set.brainShared', { name: cockpit.model.provider })
        : t('guide.set.brainNone'),
  )
  const languageNow = $derived(
    guidePrefs.lang === 'auto'
      ? t('guide.set.langAuto', { name: localeNames[i18n.locale] ?? i18n.locale })
      : (localeNames[guidePrefs.lang] ?? guidePrefs.lang),
  )
  const voiceLocale = $derived(guidePrefs.lang === 'auto' ? i18n.locale : guidePrefs.lang)
  const voiceLanguageName = $derived(localeNames[voiceLocale] ?? voiceLocale)
  const pageTitle = $derived(
    page === 'brain'
      ? t('guide.set.brain')
      : page === 'talking'
        ? t('guide.set.talking')
        : page === 'appearance'
          ? t('guide.set.appearance')
          : page === 'walks'
            ? t('guide.set.shortcuts')
            : t('guide.set.title'),
  )

  $effect(() => {
    EnabledProviders()
      .then((p) => (providers = p ?? []))
      .catch(() => (providers = []))
  })

  async function checkVoice(refresh = false, locale = voiceLocale): Promise<void> {
    const seq = ++voiceCheckSeq
    voiceState = 'checking'
    voiceReason = ''
    try {
      const [status, engines] = await Promise.all([TTSStatus(), ListTTSEngines()])
      if (seq !== voiceCheckSeq) return
      voiceEngine = engines.find((engine) => engine.active)?.id ?? ''
      if (status) {
        voiceState = 'engine'
        voiceReason = status
        return
      }
      const voices = refresh ? await RefreshTTSVoices() : await ListTTSVoices()
      if (seq !== voiceCheckSeq) return
      const lang = locale.toLowerCase()
      voiceState = voices.some((voice) => (voice.lang ?? '').toLowerCase().startsWith(lang))
        ? 'ready'
        : voiceEngine === 'windows'
          ? 'language'
          : 'engine'
    } catch (err) {
      if (seq !== voiceCheckSeq) return
      voiceState = 'engine'
      voiceReason = String(err)
    }
  }

  function openVoiceSetup(): void {
    if (voiceState === 'language' && voiceEngine === 'windows') {
      BrowserOpenURL('ms-settings:speech')
      return
    }
    onClose()
    openSettingsAt('voice')
  }

  $effect(() => {
    if (page !== 'talking') return
    const locale = voiceLocale
    void checkVoice(false, locale)
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

  $effect(() => {
    const p = effectiveProvider
    const m = effectiveModel
    if (!p) {
      thinkLevels = []
      return
    }
    SupportedThinkLevelsFor(p, m)
      .then((levels) => (thinkLevels = levels ?? []))
      .catch(() => (thinkLevels = []))
  })
</script>

<div class="gp">
  <div class="gp-head">
    <span class="gp-head-start">
      {#if page !== 'root'}
        <button type="button" class="gp-icon" onclick={() => (page = 'root')} aria-label={t('guide.set.back')}>
          <Icon name="arrowLeft" size={13} />
        </button>
      {:else}
        <span class="gp-mark"><Icon name="settings" size={13} /></span>
      {/if}
      <strong>{pageTitle}</strong>
    </span>
    <button type="button" class="gp-icon" onclick={onClose} aria-label={t('guide.close')}>
      <Icon name="x" size={13} />
    </button>
  </div>

  {#if page === 'root'}
    <div class="gp-menu" role="menu" aria-label={t('guide.set.title')}>
      <button type="button" role="menuitem" onclick={() => (page = 'brain')}>
        <span class="gp-menu-icon"><Icon name="cpu" size={14} /></span>
        <span class="gp-menu-copy"><strong>{t('guide.set.brain')}</strong><small>{brainNow}</small></span>
        <Icon name="chevronRight" size={13} />
      </button>
      <button type="button" role="menuitem" onclick={() => (page = 'talking')}>
        <span class="gp-menu-icon"><Icon name="volume2" size={14} /></span>
        <span class="gp-menu-copy"><strong>{t('guide.set.talking')}</strong><small>{guidePrefs.voice ? t('guide.set.voiceOn') : t('guide.set.voiceOff')} · {languageNow}</small></span>
        <Icon name="chevronRight" size={13} />
      </button>
      <button type="button" role="menuitem" onclick={() => (page = 'appearance')}>
        <span class="gp-menu-icon"><Icon name="square" size={14} /></span>
        <span class="gp-menu-copy"><strong>{t('guide.set.appearance')}</strong><small>{guidePrefs.size} px · {t('guide.set.appearanceHint')}</small></span>
        <Icon name="chevronRight" size={13} />
      </button>
      <button type="button" role="menuitem" onclick={() => (page = 'walks')}>
        <span class="gp-menu-icon"><Icon name="compass" size={14} /></span>
        <span class="gp-menu-copy"><strong>{t('guide.set.shortcuts')}</strong><small>{t('guide.set.walksHint')}</small></span>
        <Icon name="chevronRight" size={13} />
      </button>
    </div>

    <button type="button" class="gp-reset" onclick={resetGuidePrefs}>
      <Icon name="rotateCcw" size={11} />
      <span>{t('guide.set.reset')}</span>
    </button>
  {:else if page === 'brain'}
    <div class="gp-brain-card"><Icon name="cpu" size={12} /><span>{brainNow}</span></div>
    <div class="gp-form">
      <label class="gp-row">
        <span>{t('guide.set.provider')}</span>
        <select id="guide-provider" value={guidePrefs.provider} onchange={(e) => { setGuidePref('provider', e.currentTarget.value); setGuidePref('model', '') }}>
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
      {#if thinkLevels.length > 0}
        <label class="gp-row">
          <span>{t('settings.agentThinkPick')}</span>
          <select id="guide-think" value={guidePrefs.think} onchange={(e) => setGuidePref('think', e.currentTarget.value as ThinkLevel)}>
            <option value="">{guidePrefs.provider ? t('guide.set.modelDefault') : t('settings.agentThinkInherit')}</option>
            {#each thinkLevels as lv (lv)}<option value={lv}>{lv}</option>{/each}
            {#if guidePrefs.think && !thinkLevels.includes(guidePrefs.think)}<option value={guidePrefs.think}>{guidePrefs.think}</option>{/if}
          </select>
        </label>
      {/if}
    </div>
    <p class="gp-note">{t('guide.set.brainNote')}</p>
  {:else if page === 'talking'}
    <div class="gp-form">
      <label class="gp-row">
        <span>{t('guide.set.lang')}</span>
        <select id="guide-lang" value={guidePrefs.lang} onchange={(e) => setGuidePref('lang', e.currentTarget.value as 'auto' | Locale)}>
          <option value="auto">{t('guide.set.langAuto', { name: localeNames[i18n.locale] ?? i18n.locale })}</option>
          {#each Object.entries(localeNames) as [code, name] (code)}<option value={code}>{name}</option>{/each}
        </select>
      </label>
      <label class="gp-check">
        <span><strong>{t('guide.set.voice')}</strong><small>{guidePrefs.voice ? t('guide.set.voiceOn') : t('guide.set.voiceOff')}</small></span>
        <input id="guide-voice" type="checkbox" checked={guidePrefs.voice} onchange={(e) => setGuidePref('voice', e.currentTarget.checked)} />
      </label>
      {#if voiceState === 'checking'}
        <div class="gp-voice-note soft" role="status">
          <Icon name="loaderCircle" size={12} />
          <span>{t('guide.set.voiceChecking')}</span>
        </div>
      {:else if voiceState === 'language' || voiceState === 'engine'}
        <div class="gp-voice-note" role="status">
          <Icon name="volumeX" size={12} />
          <span>
            <strong>{voiceState === 'language'
              ? t('guide.set.voiceMissingLanguage', { language: voiceLanguageName })
              : t('guide.set.voiceUnavailable')}</strong>
            {#if voiceReason}<small>{voiceReason}</small>{/if}
          </span>
          <div class="gp-voice-actions">
            <button type="button" class="gp-voice-primary" onclick={openVoiceSetup}>
              {voiceState === 'language' ? t('guide.set.voiceInstall') : t('guide.set.voiceConfigure')}
            </button>
            <button type="button" onclick={() => checkVoice(true)}>{t('guide.set.voiceRecheck')}</button>
          </div>
        </div>
      {/if}
    </div>
  {:else if page === 'appearance'}
    <div class="gp-form">
      <label class="gp-size">
        <span><strong>{t('guide.set.size')}</strong><b>{guidePrefs.size} px</b></span>
        <input id="guide-size" type="range" min={GUIDE_SIZE_MIN} max={GUIDE_SIZE_MAX} value={guidePrefs.size} oninput={(e) => setGuidePref('size', Number(e.currentTarget.value))} />
        <small>{GUIDE_SIZE_MIN}–{GUIDE_SIZE_MAX} px</small>
      </label>
    </div>
    <p class="gp-note">{t('guide.set.dragResizeHint')}</p>
  {:else if page === 'walks'}
    <div class="gp-presets">
      {#each GUIDE_PRESETS as p (p.id)}
        <button type="button" class:lead={p.lead} onclick={() => onPick(p.to)}>
          {#if p.lead}<Icon name="sparkles" size={11} />{:else}<span class="gp-dot"></span>{/if}
          <span>{t(presetLabelKey(p) as never)}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .gp { display: flex; flex-direction: column; gap: 9px; }
  .gp-head { display: flex; align-items: center; justify-content: space-between; min-height: 28px; padding-bottom: 7px; border-bottom: 1px solid var(--border-subtle); }
  .gp-head-start { display: inline-flex; align-items: center; gap: 7px; min-width: 0; color: var(--text-primary); }
  .gp-head-start strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: var(--fs-sm); }
  .gp-mark { width: 24px; height: 24px; display: grid; place-items: center; color: var(--badge-amber-text); }
  .gp-icon { width: 24px; height: 24px; display: grid; place-items: center; padding: 0; border: 1px solid transparent; border-radius: 6px; background: transparent; color: var(--text-muted); cursor: pointer; }
  .gp-icon:hover { color: var(--text-primary); background: var(--surface-hover); border-color: var(--border-subtle); }
  .gp-menu { display: grid; gap: 5px; }
  .gp-menu > button { width: 100%; min-height: 52px; display: grid; grid-template-columns: 26px minmax(0, 1fr) 14px; align-items: center; gap: 8px; padding: 7px 9px; border: 1px solid transparent; border-radius: 9px; background: var(--surface-sunken); color: var(--text-primary); text-align: left; cursor: pointer; }
  .gp-menu > button:hover { background: var(--surface-hover); border-color: var(--border-subtle); }
  .gp-menu > button > :global(svg:last-child) { color: var(--text-dim); }
  .gp-menu-icon { width: 26px; height: 26px; display: grid; place-items: center; color: var(--badge-amber-text); }
  .gp-menu-copy { min-width: 0; display: grid; gap: 2px; }
  .gp-menu-copy strong { font-size: var(--fs-xs); font-weight: 600; }
  .gp-menu-copy small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-dim); font-size: var(--fs-2xs); }
  .gp-brain-card { display: flex; align-items: center; gap: 7px; padding: 7px 9px; border: 1px solid var(--badge-amber-border); border-radius: 8px; background: color-mix(in srgb, var(--badge-amber-bg) 55%, var(--surface-sunken)); color: var(--badge-amber-text); font-size: var(--fs-xs); }
  .gp-form { display: grid; gap: 7px; padding: 9px; border: 1px solid var(--border-subtle); border-radius: 10px; background: color-mix(in srgb, var(--surface-sunken) 42%, transparent); }
  .gp-row { display: grid; gap: 5px; color: var(--text-muted); font-size: var(--fs-xs); }
  .gp-row select { width: 100%; min-width: 0; padding: 6px 8px; border: 1px solid var(--border-default); border-radius: 6px; background: var(--surface-panel); color: var(--text-primary); font: inherit; outline: none; }
  .gp-row select:focus { border-color: var(--badge-amber-border); box-shadow: 0 0 0 2px color-mix(in srgb, var(--badge-amber-border) 25%, transparent); }
  .gp-check { min-height: 48px; display: flex; align-items: center; justify-content: space-between; gap: 10px; cursor: pointer; }
  .gp-check > span { display: grid; gap: 2px; }
  .gp-check strong { color: var(--text-primary); font-size: var(--fs-xs); }
  .gp-check small { color: var(--text-dim); font-size: var(--fs-2xs); }
  .gp-check input { width: 16px; height: 16px; accent-color: var(--badge-amber-text); }
  .gp-voice-note { display: grid; grid-template-columns: 16px minmax(0, 1fr); align-items: start; gap: 6px 8px; padding: 8px; border: 1px solid var(--badge-amber-border); border-radius: 8px; background: color-mix(in srgb, var(--badge-amber-bg) 42%, var(--surface-sunken)); color: var(--badge-amber-text); font-size: var(--fs-2xs); line-height: 1.4; }
  .gp-voice-note.soft { color: var(--text-dim); border-color: var(--border-subtle); background: transparent; }
  .gp-voice-note > span { display: grid; gap: 2px; min-width: 0; }
  .gp-voice-note strong { color: var(--text-primary); font-size: var(--fs-xs); }
  .gp-voice-note small { overflow-wrap: anywhere; color: var(--text-dim); }
  .gp-voice-actions { grid-column: 1 / -1; display: flex; gap: 6px; flex-wrap: wrap; }
  .gp-voice-actions button { padding: 5px 8px; border: 1px solid var(--border-default); border-radius: 6px; background: var(--surface-panel); color: var(--text-secondary); font: inherit; cursor: pointer; }
  .gp-voice-actions button:hover { color: var(--text-primary); background: var(--surface-hover); }
  .gp-voice-actions .gp-voice-primary { color: var(--badge-amber-text); border-color: var(--badge-amber-border); background: var(--badge-amber-bg); font-weight: 600; }
  .gp-size { display: grid; gap: 8px; }
  .gp-size > span { display: flex; align-items: center; justify-content: space-between; font-size: var(--fs-xs); }
  .gp-size strong { color: var(--text-primary); }
  .gp-size b { color: var(--badge-amber-text); font-weight: 600; }
  .gp-size input { width: 100%; accent-color: var(--badge-amber-text); }
  .gp-size small { color: var(--text-dim); font-size: var(--fs-2xs); }
  .gp-note { margin: 0; color: var(--text-dim); font-size: var(--fs-2xs); line-height: 1.45; }
  .gp-presets { display: grid; gap: 6px; }
  .gp-presets button { min-height: 36px; display: flex; align-items: center; gap: 7px; padding: 7px 10px; border: 1px solid var(--border-subtle); border-radius: 8px; background: var(--surface-sunken); color: var(--text-secondary); cursor: pointer; text-align: left; }
  .gp-presets button:hover { color: var(--text-primary); background: var(--surface-hover); border-color: var(--border-default); }
  .gp-presets button.lead { color: var(--badge-amber-text); background: color-mix(in srgb, var(--badge-amber-bg) 65%, var(--surface-sunken)); border-color: var(--badge-amber-border); }
  .gp-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--text-dim); }
  .gp-reset { align-self: flex-start; display: inline-flex; align-items: center; gap: 5px; padding: 4px 7px; border: 0; background: transparent; color: var(--text-dim); font-size: var(--fs-2xs); cursor: pointer; }
  .gp-reset:hover { color: var(--text-primary); }
</style>
