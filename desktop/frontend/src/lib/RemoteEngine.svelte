<script lang="ts">
  // Settings › เครื่องระยะไกล (§248 phase 3): the machines the engine can
  // run on, and where it is now. A row is a host; connecting is one press,
  // and the road it starts — probe, send the engine, start it, open the
  // tunnel — is narrated by the status card, not here. The bindings behind
  // this page are the screen's own, so it works while the engine is not
  // reachable: that is when "กลับมาเครื่องนี้" matters most.
  import { onMount } from 'svelte'
  import {
    RemoteHosts, SaveRemoteHost, ForgetRemoteHost, ConnectRemote, DisconnectRemote,
    StopRemoteEngine, RemoteEngineLog,
  } from '../../wailsjs/go/main/App'
  import type { main } from '../../wailsjs/go/models'
  import { t } from './i18n.svelte'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import { engine } from './stores/engine.svelte'

  let view = $state<main.RemoteHostsView>({ this: {}, active: '', hosts: [], ssh: '', sshError: '', engine: '' } as unknown as main.RemoteHostsView)

  // The machine in one line, as the Go side spells it (machine.Info.Line):
  // system · CPUs · memory · arch. The name is the row's title, not part
  // of the line.
  function specLine(m: { version?: string; os?: string; cpus?: number; memBytes?: number; arch?: string } | undefined): string {
    if (!m) return ''
    const parts: string[] = []
    if (m.version) parts.push(m.version)
    else if (m.os) parts.push(m.os)
    if (m.cpus) parts.push(`${m.cpus} CPU`)
    if (m.memBytes) parts.push(`${Math.round(m.memBytes / 2 ** 30)} GB`)
    if (m.arch) parts.push(m.arch)
    return parts.join(' · ')
  }
  let name = $state('')
  let target = $state('')
  let root = $state('')
  let editing = $state('')
  let error = $state('')
  let busy = $state('')
  let logFor = $state('')
  let logText = $state('')
  let forgetting = $state('')

  async function load() {
    try {
      view = await RemoteHosts()
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    }
  }
  onMount(() => { void load() })
  // A connection that lands writes the engine's version into the row.
  $effect(() => {
    void engine.status.state
    void engine.status.mode
    void load()
  })

  const status = $derived(engine.status)

  async function save() {
    error = ''
    busy = 'save'
    try {
      await SaveRemoteHost(name, target, root)
      name = target = root = ''
      editing = ''
      await load()
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    } finally {
      busy = ''
    }
  }

  function edit(h: main.RemoteHostView) {
    editing = h.name
    name = h.name
    target = h.target
    root = h.root
  }

  async function act(what: string, fn: () => Promise<unknown>) {
    error = ''
    busy = what
    try {
      await fn()
      await load()
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    } finally {
      busy = ''
    }
  }

  async function showLog(h: main.RemoteHostView) {
    if (logFor === h.name) { logFor = ''; return }
    logFor = h.name
    logText = '…'
    try {
      logText = (await RemoteEngineLog(h.name)) || t('settings.remoteLogEmpty')
    } catch (err) {
      logText = err instanceof Error ? err.message : String(err)
    }
  }

  function when(iso: string): string {
    if (!iso) return t('settings.remoteNeverUsed')
    const d = new Date(iso)
    return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
  }
</script>

<h2>{t('settings.remote')}</h2>
<p class="muted set-sub">{t('settings.remoteDesc')}</p>

<div class="settings-card">
  <div class="set-row">
    <span class="set-txt">
      <span class="t">
        {#if status.mode === 'remote'}
          {t('settings.remoteNowHost', { host: status.host })}
        {:else if status.mode === 'attach'}
          {t('settings.remoteNowAttach')}
        {:else}
          {t('settings.remoteNowLocal')}
        {/if}
      </span>
      <span class="d">
        {#if status.state !== 'connected' && status.detail}{status.detail}{:else if view.sshError}{view.sshError}{:else if view.ssh}{t('settings.remoteSSH', { path: view.ssh })}{/if}
      </span>
    </span>
    {#if status.mode === 'remote'}
      <button class="ctrl" disabled={busy !== ''} onclick={() => act('disconnect', DisconnectRemote)}>{t('settings.remoteDisconnect')}</button>
    {/if}
  </div>
  <div class="set-row">
    <span class="set-txt">
      <span class="d">{view.engine === 'beside' ? t('settings.remoteEngineBeside') : t('settings.remoteEngineRelease')}</span>
    </span>
  </div>
</div>

<h3 class="set-h3">{t('settings.remoteHosts')}</h3>
<div class="settings-card">
  <!-- This machine first: the one row that is always there, so the list
       reads as "the computers the engine can run on" and not as a list of
       elsewhere. -->
  <div class="set-row">
    <span class="set-txt">
      <span class="t">
        {t('settings.remoteThisMachine')}
        {#if view.this?.hostname}<span class="tag">{view.this.hostname}</span>{/if}
        {#if status.mode !== 'remote'}<span class="mcp-badge">{t('settings.remoteActive')}</span>{/if}
      </span>
      <span class="d">{specLine(view.this) + (view.this?.cpu ? ' · ' + view.this.cpu : '')}</span>
    </span>
    {#if status.mode === 'remote'}
      <button class="ctrl ctrl-primary" disabled={busy !== ''} onclick={() => act('disconnect', DisconnectRemote)}>{t('settings.remoteUseThis')}</button>
    {/if}
  </div>
  {#if view.hosts.length === 0}
    <div class="set-row"><span class="set-txt"><span class="d muted">{t('settings.remoteNoHosts')}</span></span></div>
  {/if}
  {#each view.hosts as h (h.name)}
    <div class="set-row">
      <span class="set-txt">
        <span class="t">
          {h.name}
          {#if h.name !== h.target}<span class="tag">{h.target}</span>{/if}
          {#if view.active === h.name}<span class="mcp-badge">{t('settings.remoteActive')}</span>{/if}
        </span>
        <span class="d">
          {#if h.spec}{h.spec}{:else}{h.target}{/if}
        </span>
        <span class="d">
          {#if h.root}{h.root} · {/if}
          {#if h.version}{t('settings.remoteVersion', { version: h.version, arch: h.arch })} · {/if}
          {when(h.lastUsed)}
        </span>
      </span>
      <span class="mcp-row-actions">
        {#if view.active === h.name}
          <button class="ctrl" disabled={busy !== ''} onclick={() => act('disconnect', DisconnectRemote)}>{t('settings.remoteDisconnect')}</button>
        {:else}
          <button class="ctrl ctrl-primary" disabled={busy !== '' || !!view.sshError} onclick={() => act('connect', () => ConnectRemote(h.name))}>{t('settings.remoteConnect')}</button>
        {/if}
        <button class="ctrl" onclick={() => edit(h)}>{t('settings.remoteEdit')}</button>
        <button class="ctrl" disabled={busy !== ''} onclick={() => showLog(h)}>{t('settings.remoteLog')}</button>
        <button class="ctrl" disabled={busy !== '' || !h.version} onclick={() => act('stop', () => StopRemoteEngine(h.name))}>{t('settings.remoteStop')}</button>
        <button class="ctrl ctrl-danger" disabled={view.active === h.name} onclick={() => (forgetting = h.name)}>{t('settings.remoteForget')}</button>
      </span>
    </div>
    {#if logFor === h.name}
      <div class="set-row"><pre class="remote-log">{logText}</pre></div>
    {/if}
  {/each}
</div>

<h3 class="set-h3">{editing ? t('settings.remoteEditTitle', { name: editing }) : t('settings.remoteAdd')}</h3>
<div class="settings-card">
  <div class="set-row remote-form">
    <input class="ctrl" bind:value={target} placeholder={t('settings.remoteTarget')} spellcheck="false" aria-label={t('settings.remoteTarget')} />
    <input class="ctrl" bind:value={name} placeholder={t('settings.remoteName')} spellcheck="false" aria-label={t('settings.remoteName')} />
    <input class="ctrl" bind:value={root} placeholder={t('settings.remoteRoot')} spellcheck="false" aria-label={t('settings.remoteRoot')} />
    <span class="mcp-row-actions">
      <button class="ctrl ctrl-primary" disabled={busy !== '' || !target.trim()} onclick={save}>{t('settings.remoteSave')}</button>
      {#if editing}
        <button class="ctrl" onclick={() => { editing = ''; name = target = root = '' }}>{t('settings.cancel')}</button>
      {/if}
    </span>
  </div>
  {#if error}
    <div class="set-row"><span class="set-txt"><span class="d remote-error">{error}</span></span></div>
  {/if}
  <div class="set-row"><span class="set-txt"><span class="d muted">{t('settings.remoteNote')}</span></span></div>
</div>

{#if forgetting}
  <ConfirmDialog
    title={t('settings.remoteForgetTitle')}
    message={t('settings.remoteForgetMsg')}
    detail={forgetting}
    confirmLabel={t('settings.remoteForget')}
    onConfirm={() => { const n = forgetting; forgetting = ''; void act('forget', () => ForgetRemoteHost(n)) }}
    onCancel={() => (forgetting = '')}
  />
{/if}
