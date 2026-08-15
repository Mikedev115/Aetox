<script lang="ts">
  // Background tasks: the agent's dev servers, watch builds and log tails
  // (shell with run_in_background), mirrored live from the Go job registry.
  // The list arrives by push — App.svelte applies every background:changed
  // event into the cockpit store — so this pane only draws and acts.
  import { onMount } from 'svelte'
  import { StopBackgroundTask } from '../../../wailsjs/go/main/App'
  import {
    cockpit, refreshBackgroundTasks, clearFinishedBackgroundTasks, visibleBackgroundTasks,
  } from '../stores/cockpit.svelte'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import type { BackgroundTask } from '../types'

  let error = $state('')
  // Stop is destructive to a live process — same two-step confirm the session
  // list and Artifacts use: first click arms, second click fires.
  let confirmId = $state('')

  const rows = $derived(visibleBackgroundTasks())
  const running = $derived(rows.filter((r) => r.running))
  const finished = $derived(rows.filter((r) => !r.running))

  // One clock for every running row, ticking only while something runs. A
  // running row's elapsed is the Go snapshot plus what the local clock has
  // added since the snapshot arrived; a finished row's is frozen by Go.
  let now = $state(Date.now())
  $effect(() => {
    if (running.length === 0) return
    const id = setInterval(() => (now = Date.now()), 1000)
    return () => clearInterval(id)
  })

  function liveMs(task: BackgroundTask): number {
    if (!task.running) return task.elapsedMs
    return task.elapsedMs + Math.max(0, now - cockpit.backgroundTasksAt)
  }

  function fmtElapsed(ms: number): string {
    const s = Math.max(0, Math.floor(ms / 1000))
    if (s < 60) return `${s}s`
    const m = Math.floor(s / 60)
    if (m < 60) return `${m}m ${String(s % 60).padStart(2, '0')}s`
    return `${Math.floor(m / 60)}h ${String(m % 60).padStart(2, '0')}m`
  }

  function stateLabel(task: BackgroundTask): string {
    const el = fmtElapsed(liveMs(task))
    if (task.running) return t('bgTasks.stateRunning', { t: el })
    if (task.killed) return t('bgTasks.stateKilled', { t: el })
    if (task.exitError) return t('bgTasks.stateExitError', { t: el })
    return t('bgTasks.stateDone', { t: el })
  }

  async function stop(task: BackgroundTask) {
    if (confirmId !== task.id) {
      confirmId = task.id
      return
    }
    confirmId = ''
    error = ''
    try {
      await StopBackgroundTask(task.id)
      // The process dying pushes background:changed; nothing to mutate here.
    } catch (err) {
      error = t('bgTasks.stopFailed', { err: String(err) })
    }
  }

  onMount(refreshBackgroundTasks)
</script>

<div class="insp-scroll">
  <div style="padding:8px">
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px">
      <span class="muted tp-sub">{t('bgTasks.tab')}</span>
      {#if finished.length > 0}
        <button class="ctrl" onclick={clearFinishedBackgroundTasks}>
          <Icon name="x" size={13} /> {t('bgTasks.clearFinished')}
        </button>
      {/if}
    </div>

    {#if error}<div class="muted tp-sub bg-error">{error}</div>{/if}

    {#if running.length > 0}
      <div class="eyebrow" style="margin:12px 0 4px"><Icon name="loaderCircle" size={13} /> {t('bgTasks.running')} ({running.length})</div>
      {#each running as task (task.id)}
        <div class="bg-row">
          <div class="bg-main">
            <div class="bg-cmd" title={task.command}>{task.command}</div>
            <div class="muted tp-sub">{task.id} · {stateLabel(task)}</div>
          </div>
          <button
            class="icobtn tiny tip-l"
            class:armed={confirmId === task.id}
            aria-label={confirmId === task.id ? t('bgTasks.stopConfirm') : t('bgTasks.stop')}
            data-tip={confirmId === task.id ? t('bgTasks.stopConfirm') : t('bgTasks.stop')}
            onclick={() => stop(task)}
            onmouseleave={() => (confirmId = confirmId === task.id ? '' : confirmId)}
          >
            <Icon name="square" size={13} />
          </button>
        </div>
      {/each}
    {/if}

    {#if finished.length > 0}
      <div class="eyebrow" style="margin:12px 0 4px"><Icon name="clock" size={13} /> {t('bgTasks.finished')} ({finished.length})</div>
      {#each finished as task (task.id)}
        <div class="bg-row done">
          <div class="bg-main">
            <div class="bg-cmd" title={task.command}>{task.command}</div>
            <div class="muted tp-sub">{task.id} · {stateLabel(task)}</div>
            {#if !task.killed && task.exitError}
              <div class="muted tp-sub bg-error">{task.exitError}</div>
            {/if}
          </div>
        </div>
      {/each}
    {/if}

    {#if rows.length === 0}
      <div class="empty">{t('bgTasks.empty')}</div>
    {/if}
  </div>
</div>

<style>
  .bg-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border-radius: 6px;
  }
  .bg-row.done { opacity: 0.75; }
  .bg-main { flex: 1; min-width: 0; }
  .bg-cmd {
    font-family: var(--font-mono, monospace);
    font-size: var(--fs-sm);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .bg-error { color: var(--danger, #d66); }
  .icobtn.armed { color: var(--danger, #d66); }
</style>
