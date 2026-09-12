<script lang="ts">
  // The engine is a process beside this window now (§248 phase 2), and a
  // process can be starting, gone, or coming back — or, since phase 3, on
  // another machine at the end of an ssh tunnel. This card says which, in
  // the same shell as the update and capability cards, because to the user
  // it is the same kind of news: something is happening to the app that
  // they did not ask for and cannot see.
  //
  // It renders nothing while the engine is connected, which is nearly
  // always — and nothing for the first second of a start, so a launch does
  // not flash a card at every user every day. A restart is shown at once:
  // the chat just stopped answering, and a card saying why beats a screen
  // that looks frozen. The road to a host is shown at once too, step by
  // step: the user pressed the button that started it.
  import { onMount } from 'svelte'
  import { EventsOn } from '../../wailsjs/runtime/runtime'
  import { RestartEngine, DisconnectRemote } from '../../wailsjs/go/main/App'
  import type { main } from '../../wailsjs/go/models'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import { engine, applyEngineStatus, loadEngineStatus } from './stores/engine.svelte'

  let settled = $state(false)
  let hidden = $state(false)

  onMount(() => {
    const off = EventsOn('engine:status', (st: main.EngineStatus) => applyEngineStatus(st))
    void loadEngineStatus()
    // The launch's own starting state is not news; a state after this is.
    const timer = setTimeout(() => (settled = true), 1500)
    return () => { off(); clearTimeout(timer) }
  })

  const status = $derived(engine.status)
  const remote = $derived(status.mode === 'remote')

  // A new state un-hides the card: a failure after someone waved off a
  // restart notice is news they have not had.
  let said = $state('')
  $effect(() => {
    if (status.state !== said) {
      said = status.state
      hidden = false
    }
  })

  const down = $derived(status.state !== 'connected')
  const show = $derived(!hidden && down && (settled || remote || status.state !== 'starting'))
  const title = $derived(
    status.state === 'failed'
      ? (remote ? t('engine.failedRemote', { host: status.host }) : t('engine.failed'))
      : status.state === 'reconnecting'
        ? t('engine.reconnecting')
        : status.state === 'restarting'
          ? t('engine.restarting')
          : remote
            ? t('engine.goingTo', { host: status.host })
            : t('engine.starting'),
  )
</script>

{#if show}
  <div class="upd-card" role="status" aria-live="polite">
    <div class="upd-head">
      <div class="upd-icon">
        <Icon name={status.state === 'failed' ? 'alertTriangle' : 'refreshCw'} size={20} />
      </div>
      <div class="upd-headings">
        <div class="upd-title">{title}</div>
        {#if status.state !== 'failed' && status.detail}
          <div class="upd-date">{status.detail}</div>
        {:else if status.restarts > 0}
          <div class="upd-date">{t('engine.restarts', { n: String(status.restarts) })}</div>
        {/if}
      </div>
      <button class="upd-x" aria-label={t('engine.hide')} onclick={() => (hidden = true)}>
        <Icon name="x" size={14} />
      </button>
    </div>

    {#if status.state !== 'failed'}
      <div class="upd-progress">
        <div class="upd-bar indeterminate" role="progressbar" aria-label={title}>
          <div class="upd-bar-fill"></div>
        </div>
      </div>
      {#if remote}
        <div class="upd-actions">
          <button class="upd-btn" onclick={() => void DisconnectRemote()}>{t('engine.useLocal')}</button>
        </div>
      {/if}
    {:else}
      {#if status.detail}<div class="upd-err">{status.detail}</div>{/if}
      <div class="upd-note">{remote ? t('engine.failedRemoteNote') : t('engine.failedNote')}</div>
      <div class="upd-actions">
        <button class="upd-btn upd-btn-go" onclick={() => void RestartEngine()}>{remote ? t('engine.retryRemote') : t('engine.restart')}</button>
        {#if remote}
          <button class="upd-btn" onclick={() => void DisconnectRemote()}>{t('engine.useLocal')}</button>
        {/if}
      </div>
    {/if}
  </div>
{/if}
