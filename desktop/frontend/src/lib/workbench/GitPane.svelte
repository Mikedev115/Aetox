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
  import { onMount } from 'svelte'
  import {
    GitWorkingTree,
    GitFileDiff,
    GitCommitFiles,
    GitSuggestCommitMessage,
    GitSuggestSplitCommits,
  } from '../../../wailsjs/go/main/App'
  import { main } from '../../../wailsjs/go/models'
  import { cockpit, sendUserMessage, setActiveView } from '../stores/cockpit.svelte'
  import { openFileTab } from '../stores/workbench.svelte'
  import { updateCodeStatusFromGitTree } from '../stores/codeStatus.svelte'
  import { assessDangerousFile, type DangerousFileAssessment } from './gitSecurity'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import CodeDiff from '../CodeDiff.svelte'

  let files = $state<main.GitFileChange[]>([])
  let loaded = $state(false)
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
  let alert = $state<{ type: 'err' | 'success'; text: string } | null>(null)

  // Smart split state
  let splitGroups = $state<main.GitCommitGroup[]>([])
  let analyzingSplit = $state(false)
  let splitGroupFiles = $state<Record<number, Record<string, boolean>>>({})
  let groupMessages = $state<Record<number, string>>({})
  let committingGroupIdx = $state<number | null>(null)
  let committingAll = $state(false)

  const branch = $derived(cockpit.project.branch || '')
  const totals = $derived(files.reduce(
    (acc, f) => ({ added: acc.added + (f.added ?? 0), removed: acc.removed + (f.removed ?? 0) }),
    { added: 0, removed: 0 },
  ))

  const selectedCount = $derived(files.filter((f) => selectedFiles[f.path]).length)
  const allSelected = $derived(files.length > 0 && files.every((f) => selectedFiles[f.path]))

  const dangerousFiles = $derived(
    files
      .map((f) => ({ file: f, assessment: assessDangerousFile(f.path) }))
      .filter((item): item is { file: main.GitFileChange; assessment: DangerousFileAssessment } => item.assessment !== null)
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

  async function refresh() {
    files = (await GitWorkingTree()) ?? []
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
    // Filter remaining split groups
    if (splitGroups.length > 0) {
      splitGroups = splitGroups
        .map((g) => ({
          ...g,
          files: g.files.filter((p) => files.some((f) => f.path === p)),
        }))
        .filter((g) => g.files.length > 0)
    }
  }

  onMount(refresh)

  let wasWorking = false
  $effect(() => {
    const working = cockpit.awaitingReply
    if (wasWorking && !working) void refresh()
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
      await GitCommitFiles(trimmed, chosen)
      manualMessage = ''
      alert = { type: 'success', text: t('git.commitSuccess') }
      setTimeout(() => { if (alert?.type === 'success') alert = null }, 4000)
    } catch (err: any) {
      alert = { type: 'err', text: t('git.commitFailed', { error: String(err?.message ?? err) }) }
    } finally {
      committing = false
      await refresh()
    }
  }

  async function handleAnalyzeSplit() {
    analyzingSplit = true
    alert = null
    try {
      const groups = (await GitSuggestSplitCommits()) ?? []
      splitGroups = groups
      groupMessages = {}
      splitGroupFiles = {}
      for (let i = 0; i < groups.length; i++) {
        groupMessages[i] = groups[i].message
        splitGroupFiles[i] = {}
        for (const fp of groups[i].files) {
          splitGroupFiles[i][fp] = assessDangerousFile(fp) === null
        }
      }
    } catch (err: any) {
      alert = { type: 'err', text: String(err?.message ?? err) }
    } finally {
      analyzingSplit = false
    }
  }

  async function handleCommitGroup(idx: number) {
    const g = splitGroups[idx]
    if (!g) return
    const msg = (groupMessages[idx] ?? g.message).trim()
    if (!msg) {
      alert = { type: 'err', text: t('git.noCommitMessage') }
      return
    }
    const chosen = g.files.filter((p) => splitGroupFiles[idx]?.[p] === true)
    if (chosen.length === 0) {
      alert = { type: 'err', text: t('git.noFilesSelected') }
      return
    }

    committingGroupIdx = idx
    alert = null
    try {
      await GitCommitFiles(msg, chosen)
      alert = { type: 'success', text: `${g.title}: ${t('git.commitSuccess')}` }
      setTimeout(() => { if (alert?.type === 'success') alert = null }, 4000)
    } catch (err: any) {
      alert = { type: 'err', text: t('git.commitFailed', { error: String(err?.message ?? err) }) }
    } finally {
      committingGroupIdx = null
      await refresh()
    }
  }

  async function handleCommitAllGroups() {
    if (splitGroups.length === 0) return
    committingAll = true
    alert = null
    try {
      for (let i = 0; i < splitGroups.length; i++) {
        const g = splitGroups[i]
        const msg = (groupMessages[i] ?? g.message).trim()
        const chosen = g.files.filter((p) => splitGroupFiles[i]?.[p] === true)
        if (chosen.length > 0 && msg) {
          committingGroupIdx = i
          await GitCommitFiles(msg, chosen)
        }
      }
      alert = { type: 'success', text: t('git.commitSuccess') }
      setTimeout(() => { if (alert?.type === 'success') alert = null }, 4000)
    } catch (err: any) {
      alert = { type: 'err', text: t('git.commitFailed', { error: String(err?.message ?? err) }) }
    } finally {
      committingGroupIdx = null
      committingAll = false
      await refresh()
    }
  }

  const name = (path: string) => path.split('/').pop() ?? path
  const dir = (path: string) => {
    const cut = path.lastIndexOf('/')
    return cut < 0 ? '' : path.slice(0, cut)
  }
</script>

<div class="gitpane">
  <div class="gp-head">
    <span class="gp-where">
      <Icon name="gitBranch" size={13} />
      {#if branch}<b>{branch}</b>{/if}
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
        <div class="gp-alert {alert.type}">{alert.text}</div>
      {/if}

      {#if mode === 'split'}
        <!-- Smart Split Section -->
        <div class="gp-split-section">
          {#if splitGroups.length === 0}
            <button
              type="button"
              class="gp-commit-btn"
              disabled={analyzingSplit}
              onclick={handleAnalyzeSplit}
            >
              <Icon name="sparkles" size={13} />
              <span>{analyzingSplit ? t('git.analyzingSplit') : t('git.smartSplit')}</span>
            </button>
          {:else}
            <div class="gp-split-top">
              <span class="gp-title">{t('git.splitGroups', { n: splitGroups.length })}</span>
              <button
                type="button"
                class="gp-split-all-btn"
                disabled={committingAll || committingGroupIdx !== null}
                onclick={handleCommitAllGroups}
              >
                <Icon name="check" size={12} />
                <span>{t('git.commitAllGroups', { n: splitGroups.length })}</span>
              </button>
            </div>

            <div class="gp-split-cards">
              {#each splitGroups as g, i}
                <div class="gp-split-card">
                  <div class="gp-split-card-head">
                    <span class="gp-split-title">
                      <Icon name="package" size={12} />
                      {g.title}
                      {#if g.files.some((fp) => assessDangerousFile(fp) !== null)}
                        <span class="gp-row-danger-badge" title={t('git.dangerousWarningDesc')}>
                          <Icon name="alertTriangle" size={10} />
                          <span>{t('git.dangerousBadge')}</span>
                        </span>
                      {/if}
                    </span>
                    <span class="gp-stat">
                      <span class="add">{g.files.length} {t('chat.filesChanged', { n: g.files.length })}</span>
                    </span>
                  </div>

                  <input
                    type="text"
                    class="gp-split-msg-input"
                    bind:value={groupMessages[i]}
                    placeholder={t('git.commitPlaceholder')}
                  />

                  <div class="gp-split-files">
                    {#each g.files as fp}
                      <label class="gp-split-file-item">
                        <input
                          type="checkbox"
                          checked={splitGroupFiles[i]?.[fp] === true}
                          onchange={(e) => {
                            if (!splitGroupFiles[i]) splitGroupFiles[i] = {}
                            splitGroupFiles[i][fp] = (e.currentTarget as HTMLInputElement).checked
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
                    <button
                      type="button"
                      class="gp-split-commit-btn"
                      disabled={committingGroupIdx === i || committingAll}
                      onclick={() => handleCommitGroup(i)}
                    >
                      <Icon name="check" size={11} />
                      <span>{committingGroupIdx === i ? t('git.committing') : t('git.commitGroup')}</span>
                    </button>
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
              <Icon name="sparkles" size={11} />
              <span>{generatingMessage ? t('git.generating') : t('git.generate')}</span>
            </button>
          </div>

          <button
            type="button"
            class="gp-commit-btn"
            disabled={committing || selectedCount === 0 || !manualMessage.trim()}
            onclick={handleManualCommit}
          >
            <Icon name="check" size={13} />
            <span>
              {committing
                ? t('git.committing')
                : t('git.commitSelected', { n: selectedCount })}
            </span>
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
              <span class="gp-row-danger-badge" title={t(`git.dangerCat.${danger?.category}` as any) || danger?.reason}>
                <Icon name="alertTriangle" size={10} />
                <span>{t('git.dangerousBadge')}</span>
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
