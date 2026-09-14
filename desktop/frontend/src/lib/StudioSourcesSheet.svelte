<script lang="ts">
  // "หาวัตถุดิบเพิ่ม" — the sheet that slides in from the right when someone
  // wants more material on the shelf, the same door the MCP page uses for
  // adding a server (Capability.svelte's .cap-sheet). It was a third section
  // at the foot of the Settings page before, below the fold and never found
  // (owner, 12 ก.ย.: "ไม่ใช่ไปอยู่ข้างล่างจนหาไม่เจอ").
  //
  // Two ways in, in the order a person has them: a folder they already own,
  // and pages where they can go and get one. Links, never downloads Aetox
  // performs — studioSources.ts says why.
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
  import { STUDIO_SOURCES } from './studioSources'
  import { t, type TKey } from './i18n.svelte'
  import Icon from './Icon.svelte'

  let {
    imported,
    scanning = false,
    onAddFolder,
    onClose,
  }: {
    /** Whether a source's folder is already a shelf, by its expected folder name. */
    imported: (folder?: string) => boolean
    scanning?: boolean
    onAddFolder: () => void
    onClose: () => void
  } = $props()

  let sheetEl: HTMLDivElement | undefined
  $effect(() => { sheetEl?.querySelector<HTMLElement>('button, [href]')?.focus() })
  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') { e.stopPropagation(); onClose() }
  }
</script>

<div class="studio-sheet-overlay" role="presentation" onkeydown={onKey}>
  <button class="studio-sheet-backdrop" aria-label={t('settings.studioClose')} onclick={onClose}></button>
  <div class="studio-sheet" role="dialog" aria-modal="true" aria-labelledby="studio-sheet-title" bind:this={sheetEl}>
    <div class="studio-sheet-head">
      <h3 id="studio-sheet-title">{t('settings.studioSources')}</h3>
      <button class="icobtn" aria-label={t('settings.studioClose')} onclick={onClose}><Icon name="x" size={15} /></button>
    </div>

    <section class="studio-sheet-sec">
      <div class="studio-sheet-row">
        <span class="studio-mark"><Icon name="folderOpen" size={20} /></span>
        <div class="studio-sheet-txt">
          <div class="t">{t('settings.studioAddOwn')}</div>
          <div class="d">{t('settings.studioAddOwnDesc')}</div>
        </div>
        <button class="ctrl ctrl-primary" data-guide="studio.import_btn" disabled={scanning} onclick={() => { onAddFolder(); onClose() }}><Icon name="plus" size={14} /> {t('settings.studioAdd')}</button>
      </div>
    </section>

    <div class="ag-band">
      <span class="lab">{t('settings.studioSourcesGet')}</span><span class="n">{STUDIO_SOURCES.length}</span>
      <span class="rule"></span>
    </div>
    <p class="office-note">{t('settings.studioSourcesDesc')}</p>

    <section class="studio-sheet-sec">
      {#each STUDIO_SOURCES as src (src.url)}
        <div class="studio-sheet-row">
          <span class="studio-mark"><Icon name="globe" size={20} /></span>
          <div class="studio-sheet-txt">
            <div class="t">{src.name} <span class="studio-tr">{src.where}</span></div>
            <div class="d">{t(src.desc)}</div>
            <div class="chair-chips">
              <span class="chip" class:mine={src.rights === 'cc0'} class:deny={src.rights === 'unknown'}>{t(`settings.studioRights.${src.rights}` as TKey)}</span>
              {#if imported(src.folder)}<span class="chip mine">{t('settings.studioImported')}</span>{/if}
            </div>
          </div>
          <button class="ctrl" onclick={() => BrowserOpenURL(src.url)}><Icon name="externalLink" size={13} /> {t('settings.studioOpenSource')}</button>
        </div>
      {/each}
    </section>
  </div>
</div>
