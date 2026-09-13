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
  import Icon from './Icon.svelte'
  import { fold } from './fold'
  import { tick } from 'svelte'
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
  // The form exists only while someone is filling it in — a press on
  // เพิ่มเครื่อง or แก้ไข opens it as a dialog, บันทึก or ยกเลิก closes it
  // (owner, 13 ก.ย. 2026: "ทำเป็นปุ่มเพิ่มเครื่อง แล้วค่อยแสดงหน้าให้กรอก").
  // Three empty fields on a page that is mostly read, not written, made
  // the page look like a form.
  let formOpen = $state(false)
  let targetEl = $state<HTMLInputElement | null>(null)
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
      closeForm()
      await load()
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    } finally {
      busy = ''
    }
  }

  function openForm(h?: main.RemoteHostView) {
    error = ''
    editing = h?.name ?? ''
    name = h?.name ?? ''
    target = h?.target ?? ''
    root = h?.root ?? ''
    opener = document.activeElement
    formOpen = true
    void tick().then(() => targetEl?.focus())
  }

  // Who opened the dialog gets the focus back when it closes, as
  // ConfirmDialog does — the press was on a button in the list.
  let opener: Element | null = null

  function closeForm() {
    formOpen = false
    if (opener instanceof HTMLElement && opener.isConnected) opener.focus()
    opener = null
    editing = ''
    name = target = root = ''
    error = ''
  }

  // Enter in a field saves, Escape closes, Tab stays inside the dialog —
  // the keys a three-field dialog owes. Enter is read only on an input:
  // on a button it is that button's own press.
  function formKey(e: KeyboardEvent) {
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); closeForm(); return }
    if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
      e.preventDefault()
      if (target.trim() && busy === '') void save()
      return
    }
    if (e.key !== 'Tab' || !(e.currentTarget instanceof HTMLElement)) return
    const focusable = Array.from(e.currentTarget.querySelectorAll<HTMLElement>('input, button:not(:disabled)'))
      .filter((el) => !el.classList.contains('confirm-backdrop'))
    if (focusable.length === 0) return
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus() }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus() }
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

<div class="remote-head">
  <h3 class="set-h3">{t('settings.remoteHosts')}</h3>
  <button class="ctrl ctrl-primary" onclick={() => openForm()}>
    <Icon name="plus" size={14} /> {t('settings.remoteAdd')}
  </button>
</div>
<!-- A card per machine (owner picked แบบ C from the lab, 13 ก.ย. 2026): the
     head is what you read — icon, name, where it runs, the one verb — and
     the foot is what you rarely touch: when it was last used, and the quiet
     maintenance buttons. Five equal buttons on one line was the list the
     owner called "ชิดเกินไป". -->
<div class="settings-card rm-card">
  <div class="rm-card-head">
    <span class="rm-ic"><Icon name="monitor" size={18} /></span>
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
</div>
{#if view.hosts.length === 0}
  <div class="rm-empty">{t('settings.remoteNoHosts')}</div>
{/if}
{#each view.hosts as h (h.name)}
  <div class="settings-card rm-card" class:rm-active={view.active === h.name}>
    <div class="rm-card-head">
      <span class="rm-ic"><Icon name="server" size={18} /></span>
      <span class="set-txt">
        <span class="t">
          {h.name}
          {#if h.name !== h.target}<span class="tag">{h.target}</span>{/if}
          {#if view.active === h.name}<span class="mcp-badge">{t('settings.remoteActive')}</span>{/if}
        </span>
        <span class="d">{#if h.spec}{h.spec}{:else}{h.target}{/if}</span>
      </span>
      {#if view.active === h.name}
        <button class="ctrl" disabled={busy !== ''} onclick={() => act('disconnect', DisconnectRemote)}>{t('settings.remoteDisconnect')}</button>
      {:else}
        <button class="ctrl ctrl-primary" disabled={busy !== '' || !!view.sshError} onclick={() => act('connect', () => ConnectRemote(h.name))}>{t('settings.remoteConnect')}</button>
      {/if}
    </div>
    <div class="rm-card-foot">
      <span class="d rm-meta">
        {#if h.root}{h.root} · {/if}
        {#if h.version}{t('settings.remoteVersion', { version: h.version, arch: h.arch })} · {/if}
        {when(h.lastUsed)}
      </span>
      <span class="rm-quiet">
        <button class="ctrl" onclick={() => openForm(h)}>{t('settings.remoteEdit')}</button>
        <button class="ctrl" class:on={logFor === h.name} disabled={busy !== ''} onclick={() => showLog(h)}>{t('settings.remoteLog')}</button>
        <button class="ctrl" disabled={busy !== '' || !h.version} onclick={() => act('stop', () => StopRemoteEngine(h.name))}>{t('settings.remoteStop')}</button>
        <button class="ctrl ctrl-danger" disabled={view.active === h.name} onclick={() => (forgetting = h.name)}>{t('settings.remoteForget')}</button>
      </span>
    </div>
    {#if logFor === h.name}
      <div class="rm-card-log" transition:fold><pre class="remote-log">{logText}</pre></div>
    {/if}
  </div>
{/each}

{#if formOpen}
  <!-- The form is a dialog, not a card under the list: with many machines the
       list is long and a card at its foot is off screen when the press was at
       its head (owner, 13 ก.ย. 2026: "เวลาเครื่องเยอะ … ทำเป็นป็อปอัปขึ้นก็ได้").
       Same overlay as ConfirmDialog — a plain fixed layer, not <dialog>, for
       the reason written there. Escape is stopped here so one press closes
       the form and not Settings behind it; Tab stays inside. -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div class="confirm-overlay" role="dialog" tabindex="-1" aria-modal="true" aria-labelledby="remote-form-title" onkeydown={formKey}>
    <button class="confirm-backdrop" aria-label={t('settings.cancel')} onclick={closeForm}></button>
    <div class="confirm-card remote-dialog">
      <div class="remote-formhead">
        <span class="t" id="remote-form-title">{editing ? t('settings.remoteEditTitle', { name: editing }) : t('settings.remoteAdd')}</span>
        <span class="d">{t('settings.remoteFormSub')}</span>
      </div>
      <div class="pp-edit">
        <label class="pp-field">
          <span class="eyebrow">{t('settings.remoteTargetLabel')}</span>
          <input class="ctrl key-input" bind:value={target} bind:this={targetEl} placeholder="user@host" spellcheck="false" autocomplete="off" />
          <span class="d muted">{t('settings.remoteTarget')}</span>
        </label>
        <label class="pp-field">
          <span class="eyebrow">{t('settings.remoteNameLabel')}</span>
          <input class="ctrl key-input" bind:value={name} placeholder={t('settings.remoteNamePlaceholder')} spellcheck="false" autocomplete="off" />
          <span class="d muted">{t('settings.remoteName')}</span>
        </label>
        <label class="pp-field">
          <span class="eyebrow">{t('settings.remoteRootLabel')}</span>
          <input class="ctrl key-input" bind:value={root} placeholder="~/projects/app" spellcheck="false" autocomplete="off" />
          <span class="d muted">{t('settings.remoteRoot')}</span>
        </label>
      </div>
      {#if error}
        <div class="mset-error">{error}</div>
      {/if}
      <div class="remote-note">
        <Icon name="shield" size={14} />
        <span>{t('settings.remoteNote')}</span>
      </div>
      <div class="confirm-actions">
        <button class="ctrl" disabled={busy !== ''} onclick={closeForm}>{t('settings.cancel')}</button>
        <button class="ctrl ctrl-primary" disabled={busy !== '' || !target.trim()} onclick={save}>
          {busy === 'save' ? t('settings.saving') : t('settings.remoteSave')}
        </button>
      </div>
    </div>
  </div>
{/if}

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
