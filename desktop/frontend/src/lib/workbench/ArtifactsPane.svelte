<script lang="ts">
  // พาเนลชิ้นงานในเซสชัน (Session Artifacts)
  // รวบรวมแผนงาน (Plan), เอกสารสรุป (Walkthrough), หน้าเว็บจำลอง (UI Mockup), รูปภาพ และไฟล์ผลงานทั้งหมด
  // ที่สร้างขึ้นในเซสชันปัจจุบัน แสดงผลพรีวิวได้ทันที พร้อมเปิดในแท็บแยกหรือเปิดโฟลเดอร์ภายนอกได้
  import { onMount, tick } from 'svelte'
  import { cockpit } from '../stores/cockpit.svelte'
  import { workbench, openFileTab, openPlanTab, openUrlInWorkbench, artifactSelection } from '../stores/workbench.svelte'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import type { IconName } from '../icons'
  import { fileURL } from '../fileUrl'
  import { renderMarkdown } from '../markdown'
  import PlanPane from './PlanPane.svelte'
  import SheetPane from './SheetPane.svelte'
  import {
    ReadFile, ReadWorkbook, SessionEdits, ListArtifactsIn, OpenArtifact, CurrentSessionID,
  } from '../../../wailsjs/go/main/App'
  import type { ooxml } from '../../../wailsjs/go/models'

  export type ArtifactKind = 'plan' | 'doc' | 'page' | 'image' | 'sheet' | 'code' | 'other'

  export interface SessionArtifactItem {
    id: string
    name: string
    path: string
    kind: ArtifactKind
    size?: number
    modified?: string
    sublabel?: string
    isPlan?: boolean
  }

  let artifacts = $state<SessionArtifactItem[]>([])
  let chosenId = $state<string>('')
  let activeContent = $state<string>('')
  let sheetPreview = $state<ooxml.WorkbookPreview | null>(null)
  let kindFilter = $state<string>('all')
  let htmlViewMode = $state<'preview' | 'code'>('preview')
  let loading = $state(true)
  let contentLoading = $state(false)
  let copied = $state(false)
  let copyTimer: ReturnType<typeof setTimeout> | undefined

  const plan = $derived(cockpit.plan)
  const planSteps = $derived(plan?.steps ?? [])
  const planDoneCount = $derived(planSteps.filter((st) => st.state === 'done' || st.state === 'failed').length)
  const hasPlan = $derived(!!plan && (planSteps.length > 0 || !!plan.title || (plan.sections && plan.sections.length > 0)))

  const chosenItem = $derived(artifacts.find((a) => a.id === chosenId))

  const filteredArtifacts = $derived.by(() => {
    if (kindFilter === 'all') return artifacts
    return artifacts.filter((a) => a.kind === kindFilter)
  })

  // Kind icons
  const kindIcons: Record<ArtifactKind, IconName> = {
    plan: 'compass',
    doc: 'fileText',
    page: 'globe',
    image: 'image',
    sheet: 'chartColumn',
    code: 'fileCode',
    other: 'package',
  }

  function detectKind(path: string): ArtifactKind {
    const ext = path.split('.').pop()?.toLowerCase() ?? ''
    if (['md', 'markdown', 'txt'].includes(ext)) return 'doc'
    if (['html', 'htm'].includes(ext)) return 'page'
    if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'avif', 'ico'].includes(ext)) return 'image'
    if (['xlsx', 'csv'].includes(ext)) return 'sheet'
    if (['go', 'ts', 'js', 'py', 'json', 'css', 'sql', 'sh', 'ps1', 'rs', 'c', 'cpp'].includes(ext)) return 'code'
    return 'other'
  }

  function normPath(p: string): string {
    return p.trim().replace(/\\/g, '/').toLowerCase()
  }

  function formatBytes(bytes?: number): string {
    if (!bytes || bytes <= 0) return ''
    if (bytes < 1024) return `${bytes} B`
    const kb = bytes / 1024
    if (kb < 1024) return `${kb.toFixed(kb >= 10 ? 0 : 1)} KB`
    const mb = kb / 1024
    return `${mb.toFixed(mb >= 10 ? 0 : 1)} MB`
  }

  function fileBasename(p: string): string {
    return normPath(p).split('/').pop() || normPath(p)
  }

  function findMatchingIndex(items: SessionArtifactItem[], targetPath: string): number {
    const targetNorm = normPath(targetPath)
    const targetBase = fileBasename(targetPath)
    return items.findIndex((it) => {
      if (it.isPlan) return false
      const itNorm = normPath(it.path)
      if (itNorm === targetNorm) return true
      if (fileBasename(it.path) === targetBase) return true
      return false
    })
  }

  function matchArtifact(target: string): SessionArtifactItem | undefined {
    if (!target) return undefined
    const t = normPath(target)
    const tBase = fileBasename(target)
    return artifacts.find((it) => {
      if (it.id === target) return true
      if (target === 'session-plan' && it.isPlan) return true
      const itNorm = normPath(it.path)
      return itNorm === t || fileBasename(it.path) === tBase
    })
  }

  async function loadArtifacts() {
    loading = true
    try {
      const items: SessionArtifactItem[] = []

      // 1. Current Session Plan
      if (hasPlan && plan) {
        items.push({
          id: 'session-plan',
          name: plan.title || t('chat.planCard'),
          path: '',
          kind: 'plan',
          isPlan: true,
          sublabel: t('artifactsPane.stepsCount', {
            done: String(planDoneCount),
            total: String(planSteps.length),
          }),
        })
      }

      const curSessionId = (await CurrentSessionID().catch(() => '')) || ''

      // 2. Produced files from chat messages
      for (const m of cockpit.chat) {
        if (m.producedFiles?.length) {
          for (const rawPath of m.producedFiles) {
            const at = findMatchingIndex(items, rawPath)
            const name = rawPath.split(/[\\/]/).pop() || rawPath
            if (at >= 0) {
              if (rawPath.length > items[at].path.length) items[at].path = rawPath
            } else {
              items.push({
                id: `file-${normPath(rawPath)}`,
                name,
                path: rawPath,
                kind: detectKind(rawPath),
                sublabel: rawPath.includes('/') ? rawPath.split('/').slice(-2, -1)[0] : '',
              })
            }
          }
        }
      }

      // 3. Edits made in this session
      if (curSessionId) {
        const edits = await SessionEdits(curSessionId).catch(() => ({ files: [], total: 0 }))
        for (const ef of edits.files ?? []) {
          if (!ef.gone) {
            const at = findMatchingIndex(items, ef.path)
            const name = ef.label || ef.path.split(/[\\/]/).pop() || ef.path
            if (at >= 0) {
              if (ef.path.length > items[at].path.length) items[at].path = ef.path
              if (ef.dir && !items[at].sublabel) items[at].sublabel = ef.dir
            } else {
              items.push({
                id: `file-${normPath(ef.path)}`,
                name,
                path: ef.path,
                kind: detectKind(ef.path),
                sublabel: ef.dir || '',
              })
            }
          }
        }

        // 4. Output directory sweep for this session
        const gallery = await ListArtifactsIn('all').catch(() => ({ files: [], range: 'all', total: 0 }))
        for (const art of gallery.files ?? []) {
          if (art.sessionId === curSessionId) {
            const at = findMatchingIndex(items, art.path)
            if (at >= 0) {
              items[at].path = art.path
              items[at].size = art.size
              items[at].modified = art.modified
              if (art.folder) items[at].sublabel = art.folder
            } else {
              items.push({
                id: `file-${normPath(art.path)}`,
                name: art.name,
                path: art.path,
                kind: detectKind(art.path),
                size: art.size,
                modified: art.modified,
                sublabel: art.folder || '',
              })
            }
          }
        }
      }

      artifacts = items

      // Keep chosen item or choose artifactSelection.target, or choose the first one
      const targetMatch = artifactSelection.target ? items.find(
        (it) => it.id === artifactSelection.target || normPath(it.path) === normPath(artifactSelection.target) || (artifactSelection.target === 'session-plan' && it.isPlan)
      ) : undefined
      if (targetMatch) {
        chosenId = targetMatch.id
      } else if (!items.some((it) => it.id === chosenId)) {
        chosenId = items[0]?.id ?? ''
      }
    } finally {
      loading = false
    }
  }

  async function loadActiveContent(item?: SessionArtifactItem) {
    activeContent = ''
    sheetPreview = null
    if (!item || item.isPlan || !item.path) return

    contentLoading = true
    try {
      if (item.kind === 'sheet' && item.path.toLowerCase().endsWith('.xlsx')) {
        sheetPreview = await ReadWorkbook(item.path)
      } else if (item.kind === 'doc' || item.kind === 'page' || item.kind === 'code') {
        activeContent = await ReadFile(item.path)
      }
    } catch {
      activeContent = ''
    } finally {
      contentLoading = false
    }
  }

  $effect(() => {
    // Reload artifacts when plan or chat updates
    void cockpit.plan
    void cockpit.chat.length
    void loadArtifacts()
  })

  $effect(() => {
    // When artifactSelection.target changes, select matching artifact
    if (artifactSelection.target && artifacts.length > 0) {
      const match = matchArtifact(artifactSelection.target)
      if (match && chosenId !== match.id) {
        chosenId = match.id
      }
    }
  })

  $effect(() => {
    // When chosenId changes, load content
    const item = artifacts.find((a) => a.id === chosenId)
    void loadActiveContent(item)
  })

  onMount(() => {
    const onSelect = (e: Event) => {
      const detail = (e as CustomEvent<string>).detail
      if (!detail) return
      const match = matchArtifact(detail)
      if (match) {
        chosenId = match.id
      }
    }
    window.addEventListener('select-artifact', onSelect)
    return () => window.removeEventListener('select-artifact', onSelect)
  })

  function selectArtifact(id: string) {
    chosenId = id
  }

  function openInTab(item: SessionArtifactItem) {
    if (item.isPlan) {
      openPlanTab()
    } else if (item.kind === 'page' && htmlViewMode === 'preview') {
      openUrlInWorkbench(fileURL(item.path))
    } else {
      void openFileTab(item.path, item.name)
    }
  }

  async function copyPath(path: string) {
    if (!path) return
    await navigator.clipboard.writeText(path)
    copied = true
    if (copyTimer) clearTimeout(copyTimer)
    copyTimer = setTimeout(() => {
      copied = false
    }, 2000)
  }

  async function openFolder(path: string) {
    if (!path) return
    try {
      await OpenArtifact(path)
    } catch {
      // ignore
    }
  }

  const availableKinds = $derived.by(() => {
    const set = new Set<string>()
    for (const a of artifacts) set.add(a.kind)
    return set
  })
</script>

<div class="art-pane">
  <!-- Left sidebar: Artifact list -->
  <aside class="art-list">
    <div class="art-head">
      <span class="art-title">
        <Icon name="package" size={15} />
        <b>{t('artifactsPane.title')}</b>
      </span>
      {#if artifacts.length > 0}
        <span class="art-count-badge">{artifacts.length}</span>
      {/if}
      <button
        type="button"
        class="art-icon-btn"
        aria-label={t('artifactsPane.refresh')}
        title={t('artifactsPane.refresh')}
        onclick={loadArtifacts}
      >
        <Icon name="refreshCw" size={12} />
      </button>
    </div>

    <!-- Filter chips -->
    {#if artifacts.length > 0}
      <div class="art-filters">
        <button
          type="button"
          class="art-filter-chip"
          class:active={kindFilter === 'all'}
          onclick={() => (kindFilter = 'all')}
        >
          {t('artifactsPane.filterAll')}
        </button>
        {#if availableKinds.has('plan')}
          <button
            type="button"
            class="art-filter-chip"
            class:active={kindFilter === 'plan'}
            onclick={() => (kindFilter = 'plan')}
          >
            {t('artifactsPane.filterPlan')}
          </button>
        {/if}
        {#if availableKinds.has('doc')}
          <button
            type="button"
            class="art-filter-chip"
            class:active={kindFilter === 'doc'}
            onclick={() => (kindFilter = 'doc')}
          >
            {t('artifactsPane.filterDocs')}
          </button>
        {/if}
        {#if availableKinds.has('page')}
          <button
            type="button"
            class="art-filter-chip"
            class:active={kindFilter === 'page'}
            onclick={() => (kindFilter = 'page')}
          >
            {t('artifactsPane.filterPages')}
          </button>
        {/if}
        {#if availableKinds.has('image')}
          <button
            type="button"
            class="art-filter-chip"
            class:active={kindFilter === 'image'}
            onclick={() => (kindFilter = 'image')}
          >
            {t('artifactsPane.filterImages')}
          </button>
        {/if}
        {#if availableKinds.has('sheet')}
          <button
            type="button"
            class="art-filter-chip"
            class:active={kindFilter === 'sheet'}
            onclick={() => (kindFilter = 'sheet')}
          >
            {t('artifactsPane.filterSheets')}
          </button>
        {/if}
      </div>
    {/if}

    <!-- Artifact list items -->
    <div class="art-items">
      {#if loading}
        <div class="art-empty">{t('artifactsPane.refresh')}...</div>
      {:else if artifacts.length === 0}
        <div class="art-empty">
          <Icon name="package" size={28} />
          <p class="art-empty-title">{t('artifactsPane.empty')}</p>
          <p class="art-empty-sub">{t('artifactsPane.emptyHint')}</p>
        </div>
      {:else}
        {#each filteredArtifacts as item (item.id)}
          <button
            type="button"
            class="art-item"
            class:active={item.id === chosenId}
            onclick={() => selectArtifact(item.id)}
          >
            <span class="art-item-icon kind-{item.kind}">
              <Icon name={kindIcons[item.kind]} size={14} />
            </span>
            <div class="art-item-info">
              <span class="art-item-name" title={item.name}>{item.name}</span>
              {#if item.sublabel || item.size}
                <span class="art-item-meta">
                  {item.sublabel}{item.sublabel && item.size ? ' · ' : ''}{formatBytes(item.size)}
                </span>
              {/if}
            </div>
          </button>
        {/each}
      {/if}
    </div>
  </aside>

  <!-- Right Stage: Preview & Action toolbar -->
  <main class="art-stage">
    {#if chosenItem}
      <!-- Stage header toolbar -->
      <div class="art-stage-bar">
        <div class="art-stage-title">
          <span class="art-item-icon kind-{chosenItem.kind}">
            <Icon name={kindIcons[chosenItem.kind]} size={14} />
          </span>
          <span class="art-stage-name" title={chosenItem.name}>{chosenItem.name}</span>
          <span class="art-kind-pill kind-{chosenItem.kind}">
            {chosenItem.isPlan
              ? t('artifactsPane.filterPlan')
              : chosenItem.kind === 'doc'
                ? 'Markdown'
                : chosenItem.kind === 'page'
                  ? 'HTML / UI'
                  : chosenItem.kind === 'image'
                    ? 'Image'
                    : chosenItem.kind === 'sheet'
                      ? 'Sheet'
                      : 'File'}
          </span>
        </div>

        <div class="art-stage-actions">
          <!-- HTML Mode Toggle -->
          {#if chosenItem.kind === 'page'}
            <div class="art-view-modes">
              <button
                type="button"
                class="art-mode-btn"
                class:active={htmlViewMode === 'preview'}
                onclick={() => (htmlViewMode = 'preview')}
              >
                {t('artifactsPane.viewModePreview')}
              </button>
              <button
                type="button"
                class="art-mode-btn"
                class:active={htmlViewMode === 'code'}
                onclick={() => (htmlViewMode = 'code')}
              >
                {t('artifactsPane.viewModeCode')}
              </button>
            </div>
          {/if}

          <!-- Open in Tab -->
          <button
            type="button"
            class="art-act-btn"
            title={t('artifactsPane.openTab')}
            onclick={() => openInTab(chosenItem)}
          >
            <Icon name="externalLink" size={13} />
            <span>{t('artifactsPane.openTab')}</span>
          </button>

          <!-- Copy Path -->
          {#if chosenItem.path}
            <button
              type="button"
              class="art-act-btn"
              class:success={copied}
              title={copied ? t('artifactsPane.copiedPath') : t('artifactsPane.copyPath')}
              onclick={() => copyPath(chosenItem.path)}
            >
              <Icon name={copied ? 'check' : 'copy'} size={13} />
              <span>{copied ? t('artifactsPane.copiedPath') : t('artifactsPane.copyPath')}</span>
            </button>
            <button
              type="button"
              class="art-act-btn"
              title={t('artifactsPane.openFolder')}
              onclick={() => openFolder(chosenItem.path)}
            >
              <Icon name="folder" size={13} />
            </button>
          {/if}
        </div>
      </div>

      <!-- Stage content body -->
      <div class="art-stage-body">
        {#if chosenItem.isPlan}
          <div class="art-plan-wrap">
            <PlanPane embedded={true} />
          </div>
        {:else if contentLoading}
          <div class="art-loading">
            <Icon name="refreshCw" size={20} />
            <span>{t('deckRoom.loading')}</span>
          </div>
        {:else if chosenItem.kind === 'page'}
          {#if htmlViewMode === 'preview'}
            <div class="art-iframe-box">
              <iframe
                class="art-iframe"
                src={fileURL(chosenItem.path)}
                title={chosenItem.name}
                sandbox="allow-scripts allow-forms allow-same-origin allow-popups"
              ></iframe>
            </div>
          {:else}
            <pre class="art-code"><code>{activeContent}</code></pre>
          {/if}
        {:else if chosenItem.kind === 'doc'}
          <div class="art-doc-wrap markdown-body">
            {@html renderMarkdown(activeContent)}
          </div>
        {:else if chosenItem.kind === 'image'}
          <div class="art-img-wrap">
            <img src={fileURL(chosenItem.path)} alt={chosenItem.name} />
          </div>
        {:else if chosenItem.kind === 'sheet' && sheetPreview}
          <SheetPane path={chosenItem.path} preview={sheetPreview} />
        {:else if chosenItem.kind === 'code' || activeContent}
          <pre class="art-code"><code>{activeContent}</code></pre>
        {:else}
          <div class="art-unreadable">
            <Icon name="fileText" size={32} />
            <p>{chosenItem.name}</p>
            <button type="button" class="art-act-btn" onclick={() => openInTab(chosenItem)}>
              {t('workbench.openExternally')}
            </button>
          </div>
        {/if}
      </div>
    {:else}
      <div class="art-empty">
        <Icon name="package" size={32} />
        <p class="art-empty-title">{t('artifactsPane.pick')}</p>
      </div>
    {/if}
  </main>
</div>

<style>
  .art-pane {
    display: flex;
    width: 100%;
    height: 100%;
    min-height: 0;
    overflow: hidden;
    background: var(--surface-app);
  }

  /* Left list */
  .art-list {
    width: 220px;
    flex: none;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--border-default);
    background: var(--surface-panel);
    min-height: 0;
  }

  .art-head {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border-default);
    flex: none;
  }

  .art-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--fs-sm, 13px);
    color: var(--text-primary);
  }

  .art-count-badge {
    padding: 1px 6px;
    border-radius: 999px;
    background: var(--surface-raised);
    color: var(--text-dim);
    font-size: 11px;
    font-weight: 500;
  }

  .art-icon-btn {
    margin-left: auto;
    appearance: none;
    background: none;
    border: 0;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px;
    border-radius: var(--r-xs, 3px);
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }
  .art-icon-btn:hover {
    color: var(--text-primary);
    background: var(--surface-raised);
  }

  /* Filters */
  .art-filters {
    display: flex;
    gap: 4px;
    padding: 6px 8px;
    overflow-x: auto;
    flex: none;
    border-bottom: 1px solid var(--border-subtle);
    scrollbar-width: none;
  }
  .art-filters::-webkit-scrollbar {
    display: none;
  }

  .art-filter-chip {
    appearance: none;
    background: none;
    border: 1px solid transparent;
    border-radius: var(--r-sm, 4px);
    color: var(--text-muted);
    cursor: pointer;
    font: inherit;
    font-size: 11px;
    padding: 2px 7px;
    white-space: nowrap;
    transition: all 120ms ease;
  }
  .art-filter-chip:hover {
    color: var(--text-primary);
    background: var(--surface-sunken);
  }
  .art-filter-chip.active {
    background: var(--surface-raised);
    border-color: var(--border-strong);
    color: var(--text-primary);
    font-weight: 500;
  }

  /* Items list */
  .art-items {
    flex: 1;
    overflow-y: auto;
    padding: 6px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 0;
  }

  .art-item {
    width: 100%;
    text-align: left;
    appearance: none;
    background: none;
    border: 1px solid transparent;
    border-radius: var(--r-md, 6px);
    padding: 7px 8px;
    cursor: pointer;
    font: inherit;
    display: flex;
    align-items: center;
    gap: 8px;
    transition: background 120ms ease;
  }
  .art-item:hover {
    background: var(--surface-sunken);
  }
  .art-item.active {
    background: var(--surface-raised);
    border-color: var(--border-strong);
  }

  .art-item-icon {
    flex: none;
    width: 26px;
    height: 26px;
    border-radius: var(--r-sm, 4px);
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--surface-sunken);
    color: var(--text-muted);
  }

  .art-item-icon.kind-plan {
    background: color-mix(in srgb, var(--accent, #3b82f6) 18%, transparent);
    color: var(--accent, #3b82f6);
  }
  .art-item-icon.kind-doc {
    background: color-mix(in srgb, #10b981 18%, transparent);
    color: #10b981;
  }
  .art-item-icon.kind-page {
    background: color-mix(in srgb, #8b5cf6 18%, transparent);
    color: #8b5cf6;
  }
  .art-item-icon.kind-image {
    background: color-mix(in srgb, #ec4899 18%, transparent);
    color: #ec4899;
  }
  .art-item-icon.kind-sheet {
    background: color-mix(in srgb, #06b6d4 18%, transparent);
    color: #06b6d4;
  }

  .art-item-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .art-item-name {
    font-size: var(--fs-xs, 12px);
    font-weight: 500;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .art-item-meta {
    font-size: 10.5px;
    color: var(--text-dim);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Right Stage */
  .art-stage {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--surface-app);
  }

  .art-stage-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border-default);
    background: var(--surface-panel);
    flex: none;
  }

  .art-stage-title {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .art-stage-name {
    font-size: var(--fs-sm, 13px);
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .art-kind-pill {
    padding: 1px 7px;
    border-radius: 999px;
    font-size: 10.5px;
    font-weight: 500;
    background: var(--surface-sunken);
    color: var(--text-dim);
  }

  .art-stage-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: none;
  }

  .art-view-modes {
    display: flex;
    background: var(--surface-sunken);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-sm, 4px);
    padding: 1px;
  }

  .art-mode-btn {
    appearance: none;
    background: none;
    border: 0;
    color: var(--text-muted);
    font: inherit;
    font-size: 11px;
    padding: 3px 8px;
    border-radius: 3px;
    cursor: pointer;
  }
  .art-mode-btn.active {
    background: var(--surface-raised);
    color: var(--text-primary);
    font-weight: 500;
  }

  .art-act-btn {
    appearance: none;
    background: var(--surface-raised);
    border: 1px solid var(--border-default);
    border-radius: var(--r-sm, 4px);
    color: var(--text-primary);
    cursor: pointer;
    font: inherit;
    font-size: 11px;
    padding: 4px 8px;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    transition: all 120ms ease;
  }
  .art-act-btn:hover {
    background: var(--surface-hover, var(--surface-raised));
    border-color: var(--border-strong);
  }
  .art-act-btn.success {
    color: #10b981;
    border-color: #10b981;
  }

  .art-stage-body {
    flex: 1;
    min-height: 0;
    overflow: auto;
    position: relative;
    display: flex;
    flex-direction: column;
  }

  .art-plan-wrap {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }

  .art-doc-wrap {
    padding: 16px 20px;
    max-width: 820px;
    margin: 0 auto;
    width: 100%;
  }

  .art-iframe-box {
    flex: 1;
    min-height: 0;
    width: 100%;
    height: 100%;
    background: #ffffff;
  }

  .art-iframe {
    width: 100%;
    height: 100%;
    border: none;
    display: block;
  }

  .art-code {
    margin: 0;
    padding: 16px;
    font-family: var(--font-mono, monospace);
    font-size: 12px;
    line-height: 1.6;
    color: var(--text-primary);
    overflow: auto;
    flex: 1;
  }

  .art-img-wrap {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    overflow: auto;
    background: repeating-conic-gradient(var(--surface-sunken) 0% 25%, transparent 0% 50%) 50% / 16px 16px;
  }
  .art-img-wrap img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
    border-radius: var(--r-sm, 4px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
  }

  .art-empty,
  .art-loading,
  .art-unreadable {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    padding: 32px 16px;
    color: var(--text-muted);
    gap: 8px;
    margin: auto 0;
  }

  .art-empty-title {
    font-size: var(--fs-sm, 13px);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
  }

  .art-empty-sub {
    font-size: 11.5px;
    color: var(--text-dim);
    max-width: 240px;
    line-height: 1.5;
    margin: 0;
  }
</style>
