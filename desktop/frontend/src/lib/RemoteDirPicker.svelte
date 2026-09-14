<script lang="ts">
  // The folder picker for an engine on a host (§248 phase 3). The native
  // dialog opens this machine's disks; when the engine is on a host the
  // folder to open is there, and the only thing that can list it is the
  // engine (ListDir / HomeDir). Folders only — this picks a project, not a
  // file — and the pick goes out through the same door a native pick did:
  // the caller opens the path.
  //
  // The same overlay as ConfirmDialog, for the same reasons it is not a
  // <dialog>.
  import { onMount } from 'svelte'
  import { ListDir, HomeDir } from '../../wailsjs/go/main/App'
  import type { engine as eng } from '../../wailsjs/go/models'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'

  let {
    host,
    start = '',
    title = '',
    onPick,
    onCancel,
  }: {
    host: string
    start?: string
    /** The dialog's own title when a Go door raised the picker in place of
     *  the native dialog (screen:pickdir); the default names the host. */
    title?: string
    onPick: (path: string) => void
    onCancel: () => void
  } = $props()

  let listing = $state<eng.DirListing | null>(null)
  let path = $state('')
  let typed = $state('')
  let error = $state('')
  let busy = $state(false)
  let showHidden = $state(false)
  let inputEl = $state<HTMLInputElement | null>(null)

  /** Lists p; answers whether it could. */
  async function go(p: string): Promise<boolean> {
    busy = true
    try {
      const l = await ListDir(p)
      listing = l
      path = l.path
      typed = l.path
      error = ''
      return true
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
      return false
    } finally {
      busy = false
    }
  }

  onMount(() => {
    void (async () => {
      // A start the door suggested may not exist yet on the host — the
      // folder new projects go to, before the first one is made there. The
      // native dialog opens somewhere sensible then, and so does this: the
      // home, with the box empty rather than holding a path that failed.
      if (!start || !(await go(start))) {
        await go(await HomeDir().catch(() => ''))
      }
      inputEl?.focus()
    })()
  })

  const rows = $derived((listing?.entries ?? []).filter((e) => showHidden || !e.hidden))

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.stopPropagation()
      onCancel()
    }
  }
  function onPathKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      void go(typed)
    }
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
  class="confirm-overlay"
  role="dialog"
  tabindex="-1"
  aria-modal="true"
  aria-labelledby="picker-title"
  onkeydown={onKeydown}
>
  <button class="confirm-backdrop" aria-label={t('settings.cancel')} onclick={onCancel}></button>
  <div class="confirm-card picker-card">
    <h3 id="picker-title" class="confirm-title">{title || t('picker.title', { host })}</h3>
    {#if title}<p class="muted picker-host">{t('picker.title', { host })}</p>{/if}
    <div class="picker-path">
      <button class="ctrl picker-up" disabled={!listing?.parent || busy} onclick={() => listing?.parent && go(listing.parent)} aria-label={t('picker.up')} title={t('picker.up')}>
        <Icon name="cornerLeftUp" size={14} />
      </button>
      <input
        class="ctrl picker-input"
        bind:this={inputEl}
        bind:value={typed}
        onkeydown={onPathKeydown}
        spellcheck="false"
        aria-label={t('picker.pathLabel')}
      />
    </div>
    {#if error}
      <div class="picker-error">{error}</div>
    {/if}
    <div class="picker-list" role="listbox" aria-label={t('picker.folders')}>
      {#each rows as e (e.path)}
        <button class="picker-row" class:hidden-row={e.hidden} role="option" aria-selected="false" ondblclick={() => go(e.path)} onclick={() => go(e.path)}>
          <Icon name="folder" size={14} />
          <span class="picker-name">{e.name}</span>
        </button>
      {:else}
        <div class="picker-empty muted">{busy ? '…' : t('picker.empty')}</div>
      {/each}
      {#if listing?.truncated}
        <div class="picker-empty muted">{t('picker.truncated')}</div>
      {/if}
    </div>
    <label class="picker-hidden">
      <input type="checkbox" bind:checked={showHidden} />
      <span>{t('picker.hidden')}</span>
    </label>
    <div class="confirm-actions">
      <button class="ctrl confirm-cancel" onclick={onCancel}>{t('settings.cancel')}</button>
      <button class="ctrl ctrl-primary confirm-go" disabled={!path || busy} onclick={() => onPick(path)}>{t('picker.pick')}</button>
    </div>
  </div>
</div>

<style>
  /* Under a door's own title, the host the folder is on — the line the
     default title carries on its own. */
  .picker-host { margin: -6px 0 6px; font-size: var(--fs-sm); }
</style>
