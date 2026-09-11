<script lang="ts">
  // The engine is a process beside this window now (§248 phase 2), and a
  // process can be starting, gone, or coming back. This card says which,
  // in the same shell as the update and capability cards, because to the
  // user it is the same kind of news: something is happening to the app
  // that they did not ask for and cannot see.
  //
  // It renders nothing while the engine is connected, which is nearly
  // always — and nothing for the first second of a start, so a launch does
  // not flash a card at every user every day. A restart is shown at once:
  // the chat just stopped answering, and a card saying why beats a screen
  // that looks frozen.
  import { onMount } from 'svelte'
  import { EventsOn } from '../../wailsjs/runtime/runtime'
  import { EngineStatus as engineStatus, RestartEngine } from '../../wailsjs/go/main/App'
  import type { main } from '../../wailsjs/go/models'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'

  let status = $state<main.EngineStatus>({ state: 'connected', detail: '', restarts: 0, pid: 0, address: '' } as main.EngineStatus)
  let settled = $state(false)
  let hidden = $state(false)

  // heard: an event has arrived. The mount-time read is for a card that
  // mounted after the event went by; an event that lands first is newer
  // than the read's answer and must not be overwritten by it.
  let heard = false
  onMount(() => {
    const off = EventsOn('engine:status', (st: main.EngineStatus) => { heard = true; status = st })
    void engineStatus().then((st) => { if (st && !heard) status = st })
    // The launch's own starting state is not news; a state after this is.
    const timer = setTimeout(() => (settled = true), 1500)
    return () => { off(); clearTimeout(timer) }
  })

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
  const show = $derived(!hidden && down && (settled || status.state !== 'starting'))
  const title = $derived(
    status.state === 'failed'
      ? t('engine.failed')
      : status.state === 'reconnecting'
        ? t('engine.reconnecting')
        : status.state === 'restarting'
          ? t('engine.restarting')
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
        {#if status.restarts > 0}
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
    {:else}
      {#if status.detail}<div class="upd-err">{status.detail}</div>{/if}
      <div class="upd-note">{t('engine.failedNote')}</div>
      <div class="upd-actions">
        <button class="upd-btn upd-btn-go" onclick={() => void RestartEngine()}>{t('engine.restart')}</button>
      </div>
    {/if}
  </div>
{/if}
