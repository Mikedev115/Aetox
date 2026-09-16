<script lang="ts">
  // The working tree, as a room (DECISIONS §161.4).
  //
  // The chat timeline answers "what did that call change" — hunk by hunk, under
  // the row that made it. This answers the other question, the one nothing in
  // the window held: "where does my repository stand right now". A turn is not
  // a session and a session is not an afternoon; by the third turn the second
  // question has no answer on screen at all.
  //
  // Nothing here computes a diff. `GitFileDiff` builds it with the same differ
  // the chat's fold-out uses (internal/skill/hunk.go) and CodeDiff draws it with
  // the same component, so a file's hunks look identical wherever you meet them.
  // Two renderers for one thing is how they drift apart.
  //
  // Rows are collapsed on arrival and fetched on expand. A working tree of forty
  // files is an ordinary state, and forty `git show` calls to draw a list nobody
  // has looked at yet is work done on the chance it is wanted.
  import { onMount, onDestroy, untrack } from 'svelte'
  import {
    GitWorkingTree,
    GitFileDiff,
    GitCommitFiles,
    GitSuggestCommitMessage,
    GitSuggestSplitCommits,
    GitSplitCancel,
    GitBranches,
    GitSwitchBranch,
    GitCreateBranch,
    GetProjectStatus,
  } from '../../../wailsjs/go/main/App'
  import { EventsOn } from '../../../wailsjs/runtime/runtime'
  import { engine } from '../../../wailsjs/go/models'
  import { cockpit, sendUserMessage, setActiveView } from '../stores/cockpit.svelte'
  import { openFileTab, openGitLogTab } from '../stores/workbench.svelte'
  import { updateCodeStatusFromGitTree, noteCommitLanded } from '../stores/codeStatus.svelte'
  import { assessDangerousFile, type DangerousFileAssessment } from './gitSecurity'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import CodeDiff from '../CodeDiff.svelte'

  // Whether this is the tab in front — the same test Workbench uses for this
  // slot's `display`, rather than a second way of asking that could disagree
  // with it. Defaults to true: a pane rendered on its own is the pane being
  // looked at, which is what the tests do and what a single-pane caller means.
  let { active = true }: { active?: boolean } = $props()

  let files = $state<engine.GitFileChange[]>([])
  let loaded = $state(false)
  // The last read failed — git ran out of its budget, most likely — and the
  // rows on screen are the tree as of the read before it. Said on the face of
  // the pane rather than drawn as "nothing changed" (owner, 12 ก.ย.).
  let stale = $state(false)
  // How long the last read took, which is what paces the next one.
  let lastReadMs = 0
  let open = $state<Record<string, boolean>>({})
  let diffs = $state<Record<string, string>>({})
  let loading = $state<Record<string, boolean>>({})

  // Mode: 'split' (AI Smart Split) vs 'manual' (Manual Selection / Single Commit)
  let mode = $state<'split' | 'manual'>('split')

  // Manual commit state
  let manualMessage = $state('')
  let selectedFiles = $state<Record<string, boolean>>({})
  let generatingMessage = $state(false)
  let committing = $state(false)
  let alert = $state<{ type: 'err' | 'success' | 'info'; text: string; showAskAssistant?: boolean } | null>(null)

  // Smart split: one card per proposed commit. The engine answers with the
  // groups at once — titles and files, messages still empty — and writes the
  // messages one group at a time afterwards, streaming each into its card
  // through git:split:chunk / git:split:message / git:split:done. So the
  // first card reads while the second is being written, and there is no
  // clock to lose to: the run ends when the model is done or when the user
  // stops it.
  //
  // `id` is the group's index in the engine's answer, which is what its
  // events are addressed by. It stays put when a committed card drops out of
  // the list — the array index does not, and a message keyed by that landed
  // on the next card once.
  type SplitCard = {
    id: number
    title: string
    files: string[]
    message: string
    pick: Record<string, boolean>
    // 'writing': the model is on it, text arriving. 'model': the model wrote
    // it. 'fallback': nobody did — the message is a placeholder and `reason`
    // says why, on the card, so it is never mistaken for the model's.
    // 'committed': successfully committed.
    state: 'writing' | 'model' | 'fallback' | 'committed'
    reason?: string
  }
  let splitCards = $state<SplitCard[]>([])
  let analyzingSplit = $state(false)
  const writingAny = $derived(splitCards.some((c) => c.state === 'writing'))
  let committingGroupId = $state<number | null>(null)
  let committingAll = $state(false)
  let committingProgressText = $state('')

  // Branch dropdown state in GitPane
  let branchMenuOpen = $state(false)
  let branches = $state<engine.GitBranch[]>([])
  let branchQuery = $state('')
  let branchBusy = $state('')
  let branchError = $state('')
  let branchInput = $state<HTMLInputElement | null>(null)

  const shownBranches = $derived(
    branches.filter((b) => b.name.toLowerCase().includes(branchQuery.trim().toLowerCase())),
  )
  const newBranchName = $derived(branchQuery.trim())
  const branchNameTaken = $derived(branches.some((b) => b.name === newBranchName))
  const canCreateBranch = $derived(
    newBranchName !== '' && !newBranchName.startsWith('-') && !branchNameTaken,
  )

  function startBranchCreate() {
    if (newBranchName === '') {
      branchInput?.focus()
      return
    }
    if (canCreateBranch) void pickBranch(newBranchName, true)
  }

  async function toggleBranchMenu() {
    branchMenuOpen = !branchMenuOpen
    if (!branchMenuOpen) return
    branchQuery = ''
    branchError = ''
    try {
      branches = (await GitBranches()) ?? []
    } catch {
      branches = []
    }
  }

  async function pickBranch(name: string, create: boolean) {
    if (branchBusy) return
    branchBusy = name
    branchError = ''
    try {
      await (create ? GitCreateBranch(name) : GitSwitchBranch(name))
      branchMenuOpen = false
    } catch (err) {
      branchError = String(err)
      branches = (await GitBranches().catch(() => [])) ?? []
    } finally {
      branchBusy = ''
      Object.assign(cockpit.project, await GetProjectStatus())
      await refresh()
    }
  }

  function closeBranchMenuOnOutside(e: MouseEvent) {
    if (!branchMenuOpen) return
    const el = e.target as HTMLElement | null
    if (el && !el.closest('.gp-branch-pick')) {
      branchMenuOpen = false
    }
  }

  // Events that arrive before the groups do — the goroutine starts the
  // moment the engine returns, and its first chunk can beat the promise to
  // this side — wait here and are replayed once the cards exist.
  type SplitEvent = { index: number; text?: string; message?: string; source?: string; reason?: string }
  let earlySplitEvents: Array<{ kind: 'chunk' | 'message'; p: SplitEvent }> = []
  const cardById = (id: number) => splitCards.find((c) => c.id === id)
  function applySplitChunk(p: SplitEvent) {
    const c = cardById(p.index)
    if (!c) {
      if (analyzingSplit) earlySplitEvents.push({ kind: 'chunk', p })
      return
    }
    if (c.state === 'writing') c.message += p.text ?? ''
  }
  function applySplitMessage(p: SplitEvent) {
    const c = cardById(p.index)
    if (!c) {
      if (analyzingSplit) earlySplitEvents.push({ kind: 'message', p })
      return
    }
    if (p.message) c.message = p.message
    c.state = p.source === 'fallback' ? 'fallback' : 'model'
    c.reason = p.reason
  }
  const offSplitChunk = EventsOn('git:split:chunk', applySplitChunk)
  const offSplitMessage = EventsOn('git:split:message', applySplitMessage)
  const offSplitDone = EventsOn('git:split:done', () => {
    for (const c of splitCards) {
      if (c.state === 'writing') {
        c.state = 'fallback'
        c.reason = t('git.splitNoMessage')
      }
    }
  })
  onDestroy(() => {
    offSplitChunk()
    offSplitMessage()
    offSplitDone()
    if (writingAny) void GitSplitCancel()
  })

  const branch = $derived(cockpit.project.branch || '')
  const totals = $derived(files.reduce(
    (acc, f) => ({ added: acc.added + (f.added ?? 0), removed: acc.removed + (f.removed ?? 0) }),
    { added: 0, removed: 0 },
  ))

  const selectedCount = $derived(files.filter((f) => selectedFiles[f.path]).length)
  const allSelected = $derived(files.length > 0 && files.every((f) => selectedFiles[f.path]))

  // The banner and the note to the assistant are for secrets, keys, dumps and
  // binaries. The app's own attachments are held out of the tick like those,
  // but they are not a warning — nothing leaked, the file is simply not the
  // project's — so they get the row badge and nothing louder.
  const dangerousFiles = $derived(
    files
      .map((f) => ({ file: f, assessment: assessDangerousFile(f.path) }))
      .filter((item): item is { file: engine.GitFileChange; assessment: DangerousFileAssessment } => item.assessment !== null && item.assessment.category !== 'app')
  )

  async function handleAskAssistantAboutDangerousFiles() {
    if (dangerousFiles.length === 0) return
    const fileList = dangerousFiles
      .map((df) => {
        const catKey = `git.dangerCat.${df.assessment.category}` as const
        const reasonStr = t(catKey as any) || df.assessment.reason
        return `- ${df.file.path} (${reasonStr})`
      })
      .join('\n')
    const prompt = t('git.askAssistantPrompt', { files: fileList })
    setActiveView('chat')
    void sendUserMessage(prompt)
  }

  function handleAskAssistantAboutGitError() {
    const prompt = t('git.askAssistantGitPrompt')
    setActiveView('chat')
    void sendUserMessage(prompt)
  }

  async function refresh() {
    const started = Date.now()
    try {
      files = (await GitWorkingTree()) ?? []
    } catch {
      // Keep the last tree. A read that failed is not a tree that is clean,
      // and the poll will ask again — later than it would have, see below.
      stale = true
      lastReadMs = Date.now() - started
      return
    }
    lastReadMs = Date.now() - started
    stale = false
    loaded = true
    updateCodeStatusFromGitTree(files)
    // Cleanup diffs for removed files
    for (const path of Object.keys(diffs)) {
      if (!files.some((f) => f.path === path)) {
        delete diffs[path]
        delete open[path]
        delete selectedFiles[path]
      }
    }
    // Select all by default for newly loaded files, EXCEPT dangerous/sensitive files for safety
    for (const f of files) {
      if (selectedFiles[f.path] === undefined) {
        selectedFiles[f.path] = assessDangerousFile(f.path) === null
      }
    }
    // A card whose files are all committed (or gone) leaves the list; the
    // others keep their id, message and ticks.
    if (splitCards.length > 0) {
      for (const c of splitCards) c.files = c.files.filter((p) => files.some((f) => f.path === p))
      splitCards = splitCards.filter((c) => c.files.length > 0)
    }
  }

  // Live while this room is the one being looked at (owner, 2026-09-11:
  // "แสดงแบบเรียลไทม์ไม่ใช่ต้องมาคอยกดรีเองบ่อยๆ").
  //
  // The tree moves for reasons this pane never hears about: the agent edits files
  // mid-turn, and so does everything else on the machine — an editor, a
  // formatter, a build, git itself in a terminal beside it. Re-reading only when
  // the tab was opened and when a turn ended therefore spent its life one edit
  // behind, with the button in the corner as the only way to catch up. So while
  // this is the tab in front it re-reads on a timer, and coming back to the tab
  // re-reads at once rather than waiting out the first tick.
  //
  // Two bounds, both about not spending the machine on a pane nobody is reading:
  // the timer exists only while this is the tab in front (every other slot is
  // `display: none`), and each tick is skipped while the window itself is hidden.
  const POLL_MS = 2000
  // The tick is paced by the read, not the other way round: a read that took
  // four seconds is followed by no fewer than sixteen of quiet, so a git that
  // is slow today — a scanner, a loaded disk, a repository on a share — gets a
  // pane that reads it less often rather than one that queues process after
  // process behind it. Two seconds when git is quick, never more than a
  // quarter minute.
  const POLL_MAX_MS = 15000
  const nextTickMs = () => Math.min(POLL_MAX_MS, Math.max(POLL_MS, lastReadMs * 4))

  // One read at a time. `git status` on a big tree outlasts a two-second tick,
  // and two reads in flight would race to write `files` — with the older tree
  // able to land last. The guard is deliberately not on `refresh` itself: a press
  // of the button must never be swallowed by a tick that happens to be out.
  let reading = false
  // The read in flight, so a tick that lands during one can wait for it and
  // pace itself by how long it took, rather than reading again two seconds
  // after a read that has not answered yet.
  let inflight: Promise<void> | null = null

  async function poll() {
    // Never while this pane is the thing changing the tree: a commit in flight
    // would have the rows it is committing pulled out from under it.
    if (reading || committing || committingAll || committingGroupId !== null || analyzingSplit) return
    reading = true
    inflight = (async () => {
      try {
        await refresh()
      } finally {
        reading = false
        inflight = null
      }
    })()
    await inflight
  }

  // A desk restored from a saved layout can put this tab behind the one in front,
  // and the count on the tab strip is owed an answer either way — so the very
  // first read does not wait to be looked at.
  onMount(() => {
    if (!active) void poll()
  })

  $effect(() => {
    if (!active) return
    // untrack: what `poll` reads is a reason to skip a tick, never a reason to
    // tear the timer down and build it again.
    untrack(() => void poll())
    // A timeout chain rather than setInterval, so each wait can be sized by the
    // read that came before it.
    let id: ReturnType<typeof setTimeout> | undefined
    let stopped = false
    const tick = async () => {
      if (stopped) return
      if (inflight) await inflight
      else if (document.visibilityState === 'visible') await poll()
      if (stopped) return
      id = setTimeout(tick, nextTickMs())
    }
    id = setTimeout(tick, nextTickMs())
    return () => {
      stopped = true
      if (id !== undefined) clearTimeout(id)
    }
  })

  let wasWorking = false
  $effect(() => {
    const working = cockpit.awaitingReply
    if (wasWorking && !working) void poll()
    wasWorking = working
  })

  function toggleSelectAll() {
    const next = !allSelected
    for (const f of files) {
      selectedFiles[f.path] = next
    }
  }

  function toggleFileSelection(path: string, e: MouseEvent) {
    e.stopPropagation()
    selectedFiles[path] = !selectedFiles[path]
  }

  async function toggle(path: string) {
    if (open[path]) {
      open[path] = false
      return
    }
    open[path] = true
    if (diffs[path] === undefined) {
      loading[path] = true
      try {
        diffs[path] = (await GitFileDiff(path)) ?? ''
      } finally {
        loading[path] = false
      }
    }
  }

  async function handleGenerateMessage() {
    const chosen = files.filter((f) => selectedFiles[f.path]).map((f) => f.path)
    if (chosen.length === 0) {
      alert = { type: 'err', text: t('git.noFilesSelected') }
      return
    }
    generatingMessage = true
    alert = null
    try {
      const msg = await GitSuggestCommitMessage(chosen)
      if (msg) manualMessage = msg
    } catch (err: any) {
      alert = { type: 'err', text: String(err?.message ?? err) }
    } finally {
      generatingMessage = false
    }
  }

  async function handleManualCommit() {
    const trimmed = manualMessage.trim()
    if (!trimmed) {
      alert = { type: 'err', text: t('git.noCommitMessage') }
      return
    }
    const chosen = files.filter((f) => selectedFiles[f.path]).map((f) => f.path)
    if (chosen.length === 0) {
      alert = { type: 'err', text: t('git.noFilesSelected') }
      return
    }

    committing = true
    alert = null
    try {
      const res = await GitCommitFiles(trimmed, chosen)
      if (res && res.outcome === 'committed') {
        noteCommitLanded()
        manualMessage = ''
        if (res.warning) {
          alert = {
            type: 'success',
            text: `${t('git.commitSuccessWithWarning')}: ${res.warning}`,
            showAskAssistant: true,
          }
        } else {
          alert = { type: 'success', text: t('git.commitSuccess') }
          setTimeout(() => { if (alert?.type === 'success' && !alert.showAskAssistant) alert = null }, 4000)
        }
      } else {
        alert = { type: 'info', text: t('git.noChangesToCommit') }
        setTimeout(() => { if (alert?.type === 'info') alert = null }, 4000)
      }
    } catch (err: any) {
      alert = {
        type: 'err',
        text: t('git.commitFailed', { error: String(err?.message ?? err) }),
        showAskAssistant: true,
      }
    } finally {
      committing = false
      await refresh()
    }
  }

  async function handleAnalyzeSplit() {
    analyzingSplit = true
    alert = null
    earlySplitEvents = []
    try {
      const groups = (await GitSuggestSplitCommits()) ?? []
      splitCards = groups.map((g, i) => ({
        id: i,
        title: g.title,
        files: g.files,
        message: g.message ?? '',
        pick: Object.fromEntries(g.files.map((fp) => [fp, assessDangerousFile(fp) === null])),
        state: g.source === 'fallback' ? 'fallback' : g.message ? 'model' : 'writing',
        reason: g.reason,
      }))
      const early = earlySplitEvents
      earlySplitEvents = []
      for (const e of early) (e.kind === 'chunk' ? applySplitChunk : applySplitMessage)(e.p)
    } catch (err: any) {
      alert = { type: 'err', text: String(err?.message ?? err) }
    } finally {
      analyzingSplit = false
    }
  }

  // Stop the model mid-run. What it wrote so far stays in the cards, editable;
  // the cards it had not reached say so instead of waiting forever.
  async function handleCancelSplit() {
    await GitSplitCancel()
    for (const c of splitCards) {
      if (c.state === 'writing') {
        c.state = 'fallback'
        c.reason = t('git.splitCancelled')
      }
    }
  }

  async function handleCommitGroup(card: SplitCard) {
    const msg = card.message.trim()
    if (!msg) {
      alert = { type: 'err', text: t('git.noCommitMessage') }
      return
    }
    const chosen = card.files.filter((p) => card.pick[p] === true)
    if (chosen.length === 0) {
      alert = { type: 'err', text: t('git.noFilesSelected') }
      return
    }

    committingGroupId = card.id
    alert = null
    try {
      const res = await GitCommitFiles(msg, chosen)
      if (res && res.outcome === 'committed') {
        card.state = 'committed'
        noteCommitLanded()
        if (res.warning) {
          alert = {
            type: 'success',
            text: `${card.title}: ${t('git.commitSuccessWithWarning')}: ${res.warning}`,
            showAskAssistant: true,
          }
        } else {
          alert = { type: 'success', text: `${card.title}: ${t('git.commitSuccess')}` }
          setTimeout(() => { if (alert?.type === 'success' && !alert.showAskAssistant) alert = null }, 4000)
        }
      } else {
        alert = { type: 'info', text: `${card.title}: ${t('git.noChangesToCommit')}` }
        setTimeout(() => { if (alert?.type === 'info') alert = null }, 4000)
      }
    } catch (err: any) {
      alert = {
        type: 'err',
        text: t('git.commitFailed', { error: String(err?.message ?? err) }),
        showAskAssistant: true,
      }
    } finally {
      committingGroupId = null
      await refresh()
    }
  }

  async function handleCommitAllGroups() {
    if (splitCards.length === 0) return
    committingAll = true
    alert = null
    let landedAny = false
    const warnings: string[] = []
    try {
      const total = splitCards.length
      for (let i = 0; i < splitCards.length; i++) {
        const card = splitCards[i]
        const msg = card.message.trim()
        const chosen = card.files.filter((p) => card.pick[p] === true)
        if (chosen.length > 0 && msg) {
          committingGroupId = card.id
          committingProgressText = t('git.committingGroup', { current: i + 1, total })
          const res = await GitCommitFiles(msg, chosen)
          if (res && res.outcome === 'committed') {
            card.state = 'committed'
            noteCommitLanded()
            landedAny = true
            if (res.warning) {
              warnings.push(res.warning)
            }
          }
        }
      }
      if (landedAny) {
        if (warnings.length > 0) {
          alert = {
            type: 'success',
            text: `${t('git.commitSuccessWithWarning')}: ${warnings.join('; ')}`,
            showAskAssistant: true,
          }
        } else {
          alert = { type: 'success', text: t('git.commitSuccess') }
          setTimeout(() => { if (alert?.type === 'success' && !alert.showAskAssistant) alert = null }, 4000)
        }
      } else {
        alert = { type: 'info', text: t('git.noChangesToCommit') }
        setTimeout(() => { if (alert?.type === 'info') alert = null }, 4000)
      }
    } catch (err: any) {
      alert = {
        type: 'err',
        text: t('git.commitFailed', { error: String(err?.message ?? err) }),
        showAskAssistant: true,
      }
    } finally {
      committingGroupId = null
      committingAll = false
      committingProgressText = ''
      await refresh()
    }
  }

  // The message box grows with the message — a subject line and a few
  // bullets, not one line squeezed into an input.
  const messageRows = (msg: string) => Math.min(8, Math.max(2, msg.split('\n').length + 1))

  const name = (path: string) => path.split('/').pop() ?? path
  const dir = (path: string) => {
    const cut = path.lastIndexOf('/')
    return cut < 0 ? '' : path.slice(0, cut)
  }
</script>

<svelte:window onclick={closeBranchMenuOnOutside} />

<div class="gitpane">
  <div class="gp-head">
    <span class="gp-where">
      <div class="gp-branch-pick">
        {#if branchMenuOpen}
          <div class="branch-menu">
            <div class="branch-search">
              <Icon name="search" size={13} />
              <!-- svelte-ignore a11y_autofocus -->
              <input
                type="text" bind:value={branchQuery} bind:this={branchInput} autofocus
                placeholder={t('branch.searchOrNew')} aria-label={t('branch.searchOrNew')}
                onkeydown={(e) => {
                  if (e.key === 'Escape') { branchMenuOpen = false; return }
                  if (e.key !== 'Enter') return
                  if (shownBranches.length === 1) void pickBranch(shownBranches[0].name, false)
                  else if (canCreateBranch) void pickBranch(branchQuery.trim(), true)
                }}
              />
            </div>
            {#if branchError}
              <div class="branch-error">{branchError}</div>
            {/if}
            <div class="branch-list">
              {#each shownBranches as b (b.name)}
                <button
                  type="button" class="branch-item" class:on={b.current}
                  disabled={!!branchBusy}
                  onclick={() => void pickBranch(b.name, false)}
                >
                  <span class="ic"><Icon name="gitBranch" size={13} /></span>
                  <span class="nm">{b.name}</span>
                  {#if b.current}<span class="tick"><Icon name="check" size={13} /></span>{/if}
                </button>
              {/each}
              {#if shownBranches.length === 0 && newBranchName !== ''}
                <div class="branch-none">{t('branch.none')}</div>
              {/if}
            </div>
            <button
              type="button" class="branch-item create"
              disabled={!!branchBusy || (newBranchName !== '' && !canCreateBranch)}
              title={branchNameTaken ? t('branch.exists') : undefined}
              onclick={startBranchCreate}
            >
              <span class="ic"><Icon name="plus" size={13} /></span>
              <span class="nm">
                {#if newBranchName === ''}{t('branch.createNew')}
                {:else}{t('branch.create')} “{newBranchName}”{/if}
              </span>
            </button>
          </div>
        {/if}
        <button
          type="button"
          class="gp-branch-btn"
          title={t('branch.title')}
          aria-label={t('branch.title')}
          onclick={(e) => { e.stopPropagation(); void toggleBranchMenu() }}
        >
          {#if branchBusy}
            <i class="gp-ring"></i>
          {:else}
            <Icon name="gitBranch" size={12} />
          {/if}
          <b>{branch || '—'}</b>
          <span class="caret"><Icon name={branchMenuOpen ? 'chevronUp' : 'chevronDown'} size={10} /></span>
        </button>
      </div>
      <span class="gp-arrow">→</span>
      <span>{t('git.workingTree')}</span>
    </span>
    <span class="gp-right">
      {#if files.length}
        <span class="gp-totals"><span class="add">+{totals.added}</span><span class="del">-{totals.removed}</span></span>
      {/if}
      <button class="icobtn tiny" aria-label={t('git.refresh')} data-tip={t('git.refresh')} onclick={refresh}>
        <Icon name="loaderCircle" size={13} />
      </button>
    </span>
  </div>

  <div class="gp-note">{t('git.codeDeskOnly')}</div>

  {#if stale}
    <div class="gp-stale" role="status">{t('git.readSlow')}</div>
  {/if}

  {#if loaded && files.length > 0}
    <!-- Dangerous File Warning Banner (if any detected) -->
    {#if dangerousFiles.length > 0}
      <div class="gp-danger-banner">
        <div class="gp-danger-main">
          <div class="gp-danger-icon" aria-hidden="true">
            <Icon name="alertTriangle" size={16} />
          </div>
          <div class="gp-danger-info">
            <div class="gp-danger-title">
              {t('git.dangerousWarningTitle', { n: dangerousFiles.length })}
            </div>
            <div class="gp-danger-desc">
              {t('git.dangerousWarningDesc')}
            </div>
            <div class="gp-danger-chips">
              {#each dangerousFiles as df}
                <span class="gp-danger-chip" title={t(`git.dangerCat.${df.assessment.category}` as any) || df.assessment.reason}>
                  <Icon name="alertTriangle" size={10} />
                  <span>{name(df.file.path)}</span>
                </span>
              {/each}
            </div>
          </div>
        </div>

        <button
          type="button"
          class="gp-ask-assistant-btn"
          onclick={handleAskAssistantAboutDangerousFiles}
          title={t('git.askAssistant')}
        >
          <Icon name="sparkles" size={13} />
          <span>{t('git.askAssistant')}</span>
        </button>
      </div>
    {/if}

    <!-- Commit Controls Area -->
    <div class="gp-commit-area">
      <!-- Mode Switch -->
      <div class="gp-modes" role="tablist">
        <button
          type="button"
          class="gp-mode-tab"
          class:active={mode === 'split'}
          onclick={() => (mode = 'split')}
        >
          <Icon name="sparkles" size={12} />
          <span>{t('git.modeSplit')}</span>
        </button>
        <button
          type="button"
          class="gp-mode-tab"
          class:active={mode === 'manual'}
          onclick={() => (mode = 'manual')}
        >
          <Icon name="check" size={12} />
          <span>{t('git.modeManual')}</span>
        </button>
      </div>

      {#if alert}
        <div class="gp-alert {alert.type}">
          <div class="gp-alert-main">
            <span class="gp-alert-msg">{alert.text}</span>
            {#if alert.type === 'success'}
              <!-- The commit just made is the newest row of the other room. -->
              <button type="button" class="gp-alert-link" onclick={openGitLogTab}>{t('git.viewTimeline')} →</button>
            {/if}
          </div>
          {#if alert.showAskAssistant}
            <button
              type="button"
              class="gp-alert-assistant-btn"
              onclick={handleAskAssistantAboutGitError}
              title={t('git.askAssistantGit')}
            >
              <Icon name="sparkles" size={12} />
              <span>{t('git.askAssistantGit')}</span>
            </button>
          {/if}
        </div>
      {/if}

      {#if mode === 'split'}
        <!-- Smart Split Section -->
        <div class="gp-split-section">
          {#if splitCards.length === 0}
            <button
              type="button"
              class="gp-commit-btn"
              disabled={analyzingSplit}
              onclick={handleAnalyzeSplit}
            >
              {#if analyzingSplit}
                <i class="gp-ring"></i>
                <span>{t('git.analyzingSplit')}</span>
              {:else}
                <Icon name="sparkles" size={13} />
                <span>{t('git.smartSplit')}</span>
              {/if}
            </button>
            {#if analyzingSplit}
              <div class="gp-split-skel-list">
                <div class="gp-split-skel-card">
                  <div class="gp-skel"><i class="w1"></i></div>
                  <div class="gp-skel"><i class="w2"></i><i class="w3"></i></div>
                </div>
                <div class="gp-split-skel-card">
                  <div class="gp-skel"><i class="w2"></i></div>
                  <div class="gp-skel"><i class="w3"></i><i class="w1"></i></div>
                </div>
              </div>
            {/if}
          {:else}
            <div class="gp-split-top">
              <span class="gp-title">{t('git.splitGroups', { n: splitCards.length })}</span>
              <span class="gp-split-top-actions">
                {#if writingAny}
                  <button type="button" class="gp-split-redo-btn" onclick={handleCancelSplit}>
                    <Icon name="x" size={11} />
                    <span>{t('git.splitCancel')}</span>
                  </button>
                {:else}
                  <button
                    type="button"
                    class="gp-split-redo-btn"
                    disabled={analyzingSplit || committingAll || committingGroupId !== null}
                    onclick={handleAnalyzeSplit}
                    title={t('git.splitRedo')}
                  >
                    {#if analyzingSplit}
                      <i class="gp-ring"></i>
                      <span>{t('git.analyzingSplit')}</span>
                    {:else}
                      <Icon name="sparkles" size={11} />
                      <span>{t('git.splitRedo')}</span>
                    {/if}
                  </button>
                {/if}
                <button
                  type="button"
                  class="gp-split-all-btn"
                  disabled={committingAll || committingGroupId !== null || writingAny}
                  onclick={handleCommitAllGroups}
                >
                  {#if committingAll}
                    <i class="gp-ring"></i>
                    <span>{committingProgressText || t('git.committing')}</span>
                  {:else}
                    <Icon name="check" size={12} />
                    <span>{t('git.commitAllGroups', { n: splitCards.length })}</span>
                  {/if}
                </button>
              </span>
            </div>

            {#if analyzingSplit}
              <div class="gp-split-skel-list">
                <div class="gp-split-skel-card">
                  <div class="gp-skel"><i class="w1"></i></div>
                  <div class="gp-skel"><i class="w2"></i><i class="w3"></i></div>
                </div>
              </div>
            {/if}

            <div class="gp-split-cards">
              {#each splitCards as card (card.id)}
                <div class="gp-split-card" class:writing={card.state === 'writing'} class:committed={card.state === 'committed'}>
                  <div class="gp-split-card-head">
                    <span class="gp-split-title">
                      <Icon name="package" size={12} />
                      {card.title}
                      {#if card.files.some((fp) => assessDangerousFile(fp) !== null)}
                        <span class="gp-row-danger-badge" title={t('git.dangerousWarningDesc')}>
                          <Icon name="alertTriangle" size={10} />
                          <span>{t('git.dangerousBadge')}</span>
                        </span>
                      {/if}
                    </span>
                    <span class="gp-stat">
                      <span class="add">{card.files.length} {t('chat.filesChanged', { n: card.files.length })}</span>
                    </span>
                  </div>

                  <textarea
                    class="gp-split-msg-input"
                    rows={messageRows(card.message)}
                    readonly={card.state === 'writing' || card.state === 'committed'}
                    bind:value={card.message}
                    placeholder={card.state === 'writing' ? t('git.splitWriting') : t('git.commitPlaceholder')}
                  ></textarea>
                  {#if card.state === 'writing'}
                    <span class="gp-split-note writing">
                      <Icon name="sparkles" size={10} />
                      <span>{t('git.splitWriting')}</span>
                    </span>
                  {:else if card.state === 'fallback'}
                    <span class="gp-split-note fallback">
                      <Icon name="alertTriangle" size={10} />
                      <span>{t('git.splitFallback', { reason: card.reason ?? '' })}</span>
                    </span>
                  {/if}

                  <div class="gp-split-files">
                    {#each card.files as fp}
                      <label class="gp-split-file-item">
                        <input
                          type="checkbox"
                          checked={card.pick[fp] === true}
                          disabled={card.state === 'committed'}
                          onchange={(e) => {
                            card.pick[fp] = (e.currentTarget as HTMLInputElement).checked
                          }}
                        />
                        <span>{name(fp)}</span>
                        {#if assessDangerousFile(fp)}
                          {@const danger = assessDangerousFile(fp)}
                          <span class="gp-row-danger-badge" title={t(`git.dangerCat.${danger?.category}` as any) || danger?.reason}>
                            <Icon name="alertTriangle" size={10} />
                            <span>{t('git.dangerousBadge')}</span>
                          </span>
                        {/if}
                      </label>
                    {/each}
                  </div>

                  <div class="gp-split-card-foot">
                    {#if card.state === 'committed'}
                      <span class="gp-split-done-chip">
                        <Icon name="check" size={12} />
                        <span>{t('git.commitSuccess')}</span>
                      </span>
                    {:else}
                      <button
                        type="button"
                        class="gp-split-commit-btn"
                        disabled={committingGroupId === card.id || committingAll || card.state === 'writing'}
                        onclick={() => handleCommitGroup(card)}
                      >
                        {#if committingGroupId === card.id}
                          <i class="gp-ring"></i>
                          <span>{t('git.committing')}</span>
                        {:else}
                          <Icon name="check" size={11} />
                          <span>{t('git.commitGroup')}</span>
                        {/if}
                      </button>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {:else}
        <!-- Manual Commit Section -->
        <div class="gp-manual-box">
          <div class="gp-input-wrap">
            <textarea
              class="gp-msg-input"
              bind:value={manualMessage}
              placeholder={t('git.commitPlaceholder')}
              onkeydown={(e) => {
                if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                  e.preventDefault()
                  void handleManualCommit()
                }
              }}
            ></textarea>

            <button
              type="button"
              class="gp-gen-btn"
              disabled={generatingMessage || selectedCount === 0}
              onclick={handleGenerateMessage}
              title={t('git.generate')}
            >
              {#if generatingMessage}
                <i class="gp-ring"></i>
                <span>{t('git.generating')}</span>
              {:else}
                <Icon name="sparkles" size={11} />
                <span>{t('git.generate')}</span>
              {/if}
            </button>
          </div>

          <button
            type="button"
            class="gp-commit-btn"
            disabled={committing || selectedCount === 0 || !manualMessage.trim()}
            onclick={handleManualCommit}
          >
            {#if committing}
              <i class="gp-ring"></i>
              <span>{t('git.committing')}</span>
            {:else}
              <Icon name="check" size={13} />
              <span>
                {t('git.commitSelected', { n: selectedCount })}
              </span>
            {/if}
          </button>
        </div>
      {/if}
    </div>
  {/if}

  {#if !loaded}
    <div class="gp-empty">{t('git.loading')}</div>
  {:else if files.length === 0}
    <div class="gp-empty">{t('git.clean')}</div>
  {:else}
    <div class="gp-list-head">
      <label class="gp-checkbox-label">
        <input
          type="checkbox"
          checked={allSelected}
          onchange={toggleSelectAll}
        />
        <span>{t('git.changesCount', { n: files.length })}</span>
      </label>
      <span class="gp-sel-info">
        {selectedCount} / {files.length} {t('git.selectAll')}
      </span>
    </div>

    <div class="gp-list">
      {#each files as f (f.path)}
        <div class="gp-file">
          <button
            class="gp-row" class:busy={loading[f.path]}
            aria-expanded={!!open[f.path]} aria-busy={loading[f.path] || undefined}
            onclick={() => toggle(f.path)}
          >
            <input
              type="checkbox"
              class="gp-row-checkbox"
              checked={selectedFiles[f.path] ?? false}
              onclick={(e) => toggleFileSelection(f.path, e)}
            />
            <span class="gp-caret">
              {#if loading[f.path]}
                <i class="gp-ring"></i>
              {:else}
                <Icon name={open[f.path] ? 'chevronDown' : 'chevronRight'} size={12} />
              {/if}
            </span>
            <span class="gp-name">{name(f.path)}</span>
            {#if assessDangerousFile(f.path)}
              {@const danger = assessDangerousFile(f.path)}
              <span class="gp-row-danger-badge" class:app={danger?.category === 'app'} title={t(`git.dangerCat.${danger?.category}` as any) || danger?.reason}>
                <Icon name={danger?.category === 'app' ? 'package' : 'alertTriangle'} size={10} />
                <span>{danger?.category === 'app' ? t('git.appOwnedBadge') : t('git.dangerousBadge')}</span>
              </span>
            {/if}
            {#if dir(f.path)}<span class="gp-dir">{dir(f.path)}</span>{/if}
            <span class="gp-stat">
              <span class="add">+{f.added ?? 0}</span><span class="del">-{f.removed ?? 0}</span>
            </span>
            <span class="gp-badge {f.status}">{f.status}</span>
          </button>
          {#if open[f.path]}
            <div class="gp-diff">
              {#if loading[f.path]}
                <div class="gp-skel" role="status" aria-label={t('git.loading')}>
                  <i class="w1"></i><i class="w2"></i><i class="w3"></i>
                </div>
              {:else if diffs[f.path]}
                <CodeDiff diff={diffs[f.path]} />
                <button class="gp-open" onclick={() => openFileTab(f.path, name(f.path))}>
                  <Icon name="fileCode" size={12} /> {t('git.openFile')}
                </button>
              {:else}
                <div class="gp-empty small">{t('git.noDiff')}</div>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
