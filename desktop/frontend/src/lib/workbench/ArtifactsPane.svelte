<script lang="ts">
  // พาเนลชิ้นงานในเซสชัน (Session Artifacts)
  //
  // **ชิ้นงาน = สิ่งที่ Aetox *ยื่นให้* ไม่ใช่สิ่งที่มัน *สร้างระหว่างทาง*.**
  //
  // The first version of this pane listed everything: what a turn flagged,
  // what the session edited, and every file a sweep found in the chat's output
  // folder. The last of those is what buried it — every screenshot a browsing
  // turn takes lands in output/<session>/work, every picture fetched on the
  // way lands beside it, and the plan sat at the top of a list of twelve
  // pictures nobody asked for (owner, 12 ก.ย., with a screenshot of exactly
  // that: *"มันไม่ควรแสดงทุกอย่างดิตรงนี้"*). Source code had already been cut
  // the day before, for the same reason from the other side (*"ตรงนี้ไม่ควร
  // แสดงโค้ดสิ"*).
  //
  // So the rule is no longer about extensions, it is about the SIGNAL. A thing
  // is here because one of three mechanisms said the chat handed it over:
  //
  //   - the plan and its reports live in the app's own database (plan.go,
  //     plan_report.go) — they are the work's before and after;
  //   - a tool flagged a finished file for the user (skill.Output.Artifacts):
  //     a document `write` landed in the output folder, a deck or a page the
  //     chat put on the desk or opened in the browser. Captures no longer set
  //     that flag, which is the engine-side half of this change;
  //   - nothing else. No sweep, no session edits. A picture the chat drew, a
  //     screenshot, a sheet, the project's own files — each has a home already
  //     (ผลงาน, the card under the answer, FileChangeReview), and a third copy
  //     here told nobody anything new.
  //
  // Five groups, drawn only when they have something in them: แผน / รายงาน /
  // เอกสาร / สไลด์ / หน้าเว็บ. The first three open on the stage beside the
  // list. The last two are SEND-OFF rows: a deck already has a room that pages
  // it and a page already has a browser tab that renders it, and a second
  // preview of either inside a 320px pane was the thing the owner could not
  // read. The row says where it goes, and goes there.
  //
  // Antigravity keeps the same pair at the centre of it — implementation_plan
  // before, walkthrough after (docs/antigravity-study) — and that study is
  // where "the pane shows the plan and the report, and that is enough" came
  // from.
  import { onMount } from 'svelte'
  import { cockpit } from '../stores/cockpit.svelte'
  import { openFileTab, openPlanTab, openUrlInWorkbench, artifactSelection, isDeck } from '../stores/workbench.svelte'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import type { IconName } from '../icons'
  import { fileURL } from '../fileUrl'
  import { renderMarkdown } from '../markdown'
  import PlanPane from './PlanPane.svelte'
  import ReportPane from './ReportPane.svelte'
  import { ReadFile, OpenArtifact } from '../../../wailsjs/go/main/App'
  import type { PlanReport } from '../types'

  // Whether this is the tab in front — the same test Workbench uses for this
  // slot's `display`, rather than a second way of asking that could disagree
  // with it. Defaults to true: a pane rendered on its own is the pane being
  // looked at, which is what the tests do and what a single-pane caller means
  // (the same prop, for the same reason, as GitPane).
  let { active = true }: { active?: boolean } = $props()

  // The five groups, in the order they are drawn. No 'image', no 'sheet', no
  // 'code' — see the note at the head of this file.
  type ArtifactKind = 'plan' | 'report' | 'doc' | 'deck' | 'page'

  interface SessionArtifactItem {
    id: string
    name: string
    kind: ArtifactKind
    /** Project-relative, for the file kinds; '' for the plan and the reports. */
    path: string
    meta?: string
    isPlan?: boolean
    report?: PlanReport
  }

  let artifacts = $state<SessionArtifactItem[]>([])
  let chosenId = $state<string>('')
  let activeContent = $state<string>('')
  let loading = $state(true)
  let contentLoading = $state(false)
  let copied = $state(false)
  let copyTimer: ReturnType<typeof setTimeout> | undefined

  const plan = $derived(cockpit.plan)
  const planSteps = $derived(plan?.steps ?? [])
  const planDoneCount = $derived(planSteps.filter((st) => st.state === 'done' || st.state === 'failed').length)
  const hasPlan = $derived(!!plan && (planSteps.length > 0 || !!plan.title || (plan.sections && plan.sections.length > 0)))

  const chosenItem = $derived(artifacts.find((a) => a.id === chosenId))

  const kindIcons: Record<ArtifactKind, IconName> = {
    plan: 'compass',
    report: 'fileText',
    doc: 'fileText',
    deck: 'layoutList',
    page: 'globe',
  }

  const groupOrder: ArtifactKind[] = ['plan', 'report', 'doc', 'deck', 'page']
  const groupLabel: Record<ArtifactKind, string> = {
    plan: 'artifactsPane.groupPlan',
    report: 'artifactsPane.groupReports',
    doc: 'artifactsPane.groupDocs',
    deck: 'artifactsPane.groupDecks',
    page: 'artifactsPane.groupPages',
  }
  /** The list as drawn: each group that has anything, with its rows. */
  const groups = $derived(
    groupOrder
      .map((kind) => ({ kind, items: artifacts.filter((a) => a.kind === kind) }))
      .filter((g) => g.items.length > 0),
  )
  /** A deck or a page is sent to its own room rather than drawn here. */
  const sendsOff = (kind: ArtifactKind) => kind === 'deck' || kind === 'page'

  /** What sort of handed-over file this is, or null for one this pane does
   *  not list. The flag decided it was handed over; this only sorts it. An
   *  .html is a deck or a page, and the file has to be read to say which. */
  function docKind(path: string): 'doc' | 'html' | null {
    const ext = path.split('.').pop()?.toLowerCase() ?? ''
    if (['md', 'markdown', 'txt', 'docx', 'pdf'].includes(ext)) return 'doc'
    if (['html', 'htm'].includes(ext)) return 'html'
    return null
  }

  function normPath(p: string): string {
    return p.trim().replace(/\\/g, '/').toLowerCase()
  }

  function fileBasename(p: string): string {
    return p.split(/[\\/]/).pop() || p
  }

  function matchArtifact(target: string): SessionArtifactItem | undefined {
    if (target === 'session-plan') return artifacts.find((it) => it.isPlan)
    const tNorm = normPath(target)
    const tBase = fileBasename(target)
    return artifacts.find((it) => it.id === target || normPath(it.path) === tNorm || (it.path !== '' && fileBasename(it.path) === tBase))
  }

  function reportMeta(r: PlanReport): string {
    const parts: string[] = []
    const d = new Date(r.at)
    if (!Number.isNaN(d.getTime())) {
      parts.push(d.toLocaleString(undefined, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }))
    }
    if (r.stopped) {
      parts.push(r.stopped)
    } else {
      parts.push(t('artifactsPane.stepsShort', { done: String(r.done + r.failed), total: String(r.total) }))
    }
    if (r.failed > 0) parts.push(t('artifactsPane.reportFailed', { n: String(r.failed) }))
    const mins = Math.floor((r.elapsedSecs ?? 0) / 60)
    if (mins > 0) parts.push(t('chat.planMinutes', { n: String(mins) }))
    return parts.join(' · ')
  }

  async function loadArtifacts() {
    loading = true
    try {
      const items: SessionArtifactItem[] = []

      // 1. The plan.
      if (hasPlan && plan) {
        const state = plan.running
          ? t('bgw.running')
          : planSteps.length > 0 && planDoneCount >= planSteps.length
            ? t('chat.planDoneBadge')
            : t('chat.planReadyBadge')
        const meta = [
          plan.version > 1 ? `#${plan.version}` : '',
          planSteps.length > 0 ? t('artifactsPane.stepsShort', { done: String(planDoneCount), total: String(planSteps.length) }) : '',
          state,
        ].filter(Boolean).join(' · ')
        items.push({ id: 'session-plan', name: plan.title || t('chat.planCard'), path: '', kind: 'plan', isPlan: true, meta })
      }

      // 2. The reports, newest round first — the one somebody coming back to
      //    the chat wants is the last one written.
      for (const r of [...cockpit.planReports].sort((a, b) => b.run - a.run)) {
        items.push({
          id: `report-${r.run}`,
          name: t('artifactsPane.round', { n: String(r.run) }),
          path: '',
          kind: 'report',
          report: r,
          meta: reportMeta(r),
        })
      }

      // 3. What the turns handed over. The flag is the whole test (see the
      //    note at the head of this file); an .html is read to tell a deck
      //    from a page, the way a file tab tells them apart (isDeck).
      const seen = new Set<string>()
      const pending: Promise<void>[] = []
      for (const m of cockpit.chat) {
        for (const rawPath of m.producedFiles ?? []) {
          const key = normPath(rawPath)
          if (!rawPath || seen.has(key)) continue
          const sort = docKind(rawPath)
          if (!sort) continue
          seen.add(key)
          const name = fileBasename(rawPath)
          const dir = rawPath.replace(/\\/g, '/').split('/').slice(-2, -1)[0] ?? ''
          const meta = dir && dir !== name ? dir : ''
          if (sort === 'doc') {
            items.push({ id: `file-${key}`, name, path: rawPath, kind: 'doc', meta })
            continue
          }
          const item: SessionArtifactItem = { id: `file-${key}`, name, path: rawPath, kind: 'page', meta }
          items.push(item)
          pending.push(
            ReadFile(rawPath)
              .then((content) => {
                if (!isDeck(rawPath, content)) return
                item.kind = 'deck'
                const slides = (content.match(/<(?:section|div)[^>]*\bclass\s*=\s*(?:"[^"]*\bslide\b|'[^']*\bslide\b)/gi) ?? []).length
                item.meta = slides > 0 ? t('artifactsPane.slidesCount', { n: String(slides) }) : meta
              })
              .catch(() => {}),
          )
        }
      }
      await Promise.all(pending)

      artifacts = items

      // Keep the chosen item, or take artifactSelection.target, or the first
      // row that opens on the stage — never a send-off row, which has nothing
      // to draw here.
      const targetMatch = artifactSelection.target ? matchArtifact(artifactSelection.target) : undefined
      if (targetMatch && !sendsOff(targetMatch.kind)) {
        chosenId = targetMatch.id
      } else if (!items.some((it) => it.id === chosenId)) {
        chosenId = items.find((it) => !sendsOff(it.kind))?.id ?? ''
      }
    } finally {
      loading = false
    }
  }

  async function loadActiveContent(item?: SessionArtifactItem) {
    activeContent = ''
    if (!item || item.kind !== 'doc' || !item.path) return
    const ext = item.path.split('.').pop()?.toLowerCase() ?? ''
    // .docx and .pdf are not readable here; the stage offers to open them
    // where they can be read.
    if (!['md', 'markdown', 'txt'].includes(ext)) return
    contentLoading = true
    try {
      activeContent = await ReadFile(item.path)
    } catch {
      activeContent = ''
    } finally {
      contentLoading = false
    }
  }

  $effect(() => {
    // Nothing is read while this tab is behind another one.
    //
    // The pane stays mounted when it is not the tab in front — Workbench draws
    // every slot with `display:none` and only the active one with `display:block`
    // — so this effect used to run for the whole turn: every message written and
    // every step the plan marked, each one a read of the disk, for a list nobody
    // had on screen.
    if (!active) return
    void cockpit.openSession // the list belongs to the chat on screen
    void cockpit.plan // its checklist row is one of the rows
    void cockpit.planReports // and so is every report
    void cockpit.awaitingReply // a turn just ended: what it handed over is known now
    void loadArtifacts()
  })

  $effect(() => {
    // When artifactSelection.target changes, select the matching artifact
    if (artifactSelection.target && artifacts.length > 0) {
      const match = matchArtifact(artifactSelection.target)
      if (match && !sendsOff(match.kind) && chosenId !== match.id) {
        chosenId = match.id
      }
    }
  })

  $effect(() => {
    const item = artifacts.find((a) => a.id === chosenId)
    void loadActiveContent(item)
  })

  onMount(() => {
    const onSelect = (e: Event) => {
      const detail = (e as CustomEvent<string>).detail
      if (!detail) return
      const match = matchArtifact(detail)
      if (match && !sendsOff(match.kind)) {
        chosenId = match.id
      }
    }
    window.addEventListener('select-artifact', onSelect)
    return () => window.removeEventListener('select-artifact', onSelect)
  })

  /** A row pressed. The three stage kinds select; the two send-off kinds go
   *  where they are read — the deck to the slides room (a file tab that pages
   *  it, the same tab `desk open` makes), the page to a browser tab rendered
   *  (the same tab `browser open` makes). */
  function pressRow(item: SessionArtifactItem) {
    if (item.kind === 'deck') {
      void openFileTab(item.path, item.name)
    } else if (item.kind === 'page') {
      openUrlInWorkbench(fileURL(item.path))
    } else {
      chosenId = item.id
    }
  }

  function openInTab(item: SessionArtifactItem) {
    if (item.isPlan) {
      openPlanTab()
    } else if (item.kind === 'report') {
      // A report has no file; the tab it opens in is the plan's.
      openPlanTab()
    } else {
      void openFileTab(item.path, item.name)
    }
  }

  /** What the copy button puts on the clipboard: the report as markdown, or
   *  a file's path. */
  async function copyChosen(item: SessionArtifactItem) {
    let text = item.path
    if (item.kind === 'report' && item.report) {
      const r = item.report
      text = `# ${r.title} — ${t('artifactsPane.round', { n: String(r.run) })}\n\n` +
        (r.sections ?? []).map((s) => `**${s.heading}**\n${s.body}`).join('\n\n')
    }
    if (!text) return
    await navigator.clipboard.writeText(text)
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

  const kindPill = (kind: ArtifactKind) =>
    kind === 'plan' ? t('artifactsPane.groupPlan')
      : kind === 'report' ? t('artifactsPane.groupReports')
        : kind === 'doc' ? t('artifactsPane.groupDocs')
          : kind === 'deck' ? t('artifactsPane.groupDecks')
            : t('artifactsPane.groupPages')
</script>

<div class="art-pane">
<div class="art-body">
  {#if !loading && artifacts.length === 0}
    <!-- Nothing handed over yet. One column: a list with nothing in it beside
         a stage with nothing on it is two empty boxes saying the same thing. -->
    <main class="art-stage art-stage-alone">
      <div class="art-head">
        <span class="art-title">
          <Icon name="package" size={15} />
          <b>{t('artifactsPane.title')}</b>
        </span>
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
      <div class="art-empty">
        <Icon name="compass" size={32} />
        <p class="art-empty-title">{t('artifactsPane.empty')}</p>
        <p class="art-empty-sub">{t('artifactsPane.emptyHint')}</p>
      </div>
    </main>
  {:else}
  <!-- The list -->
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

    <div class="art-items">
      {#if loading && artifacts.length === 0}
        <div class="art-empty">{t('artifactsPane.refresh')}...</div>
      {:else}
        {#each groups as group (group.kind)}
          <div class="art-group">
            <div class="art-group-head">
              <span>{t(groupLabel[group.kind] as any)}</span>
              {#if group.items.length > 1}
                <span class="art-group-count">{group.items.length}</span>
              {/if}
            </div>
            {#each group.items as item (item.id)}
              <button
                type="button"
                class="art-item"
                class:active={item.id === chosenId}
                class:sendoff={sendsOff(item.kind)}
                onclick={() => pressRow(item)}
              >
                <span class="art-item-icon kind-{item.kind}">
                  <Icon name={kindIcons[item.kind]} size={14} />
                </span>
                <div class="art-item-info">
                  <span class="art-item-name" title={item.name}>{item.name}</span>
                  {#if item.meta}
                    <span class="art-item-meta">{item.meta}</span>
                  {/if}
                </div>
                {#if sendsOff(item.kind)}
                  <span class="art-handoff">
                    <Icon name="externalLink" size={11} />
                    <span>{item.kind === 'deck' ? t('artifactsPane.toSlides') : t('artifactsPane.toBrowser')}</span>
                  </span>
                {/if}
              </button>
            {/each}
          </div>
        {/each}
      {/if}
    </div>
  </aside>

  <!-- The stage -->
  <main class="art-stage">
    {#if chosenItem}
      <div class="art-stage-bar">
        <div class="art-stage-title">
          <span class="art-item-icon kind-{chosenItem.kind}">
            <Icon name={kindIcons[chosenItem.kind]} size={14} />
          </span>
          <span class="art-stage-name" title={chosenItem.name}>{chosenItem.name}</span>
          <span class="art-kind-pill kind-{chosenItem.kind}">{kindPill(chosenItem.kind)}</span>
        </div>

        <div class="art-stage-actions">
          <button
            type="button"
            class="art-act-btn"
            title={t('artifactsPane.openTab')}
            onclick={() => openInTab(chosenItem)}
          >
            <Icon name="externalLink" size={12} />
            <span>{t('artifactsPane.openTab')}</span>
          </button>
          {#if !chosenItem.isPlan}
            <button
              type="button"
              class="art-act-btn"
              class:success={copied}
              title={copied ? t('artifactsPane.copiedPath') : chosenItem.kind === 'report' ? t('chat.copyCode') : t('artifactsPane.copyPath')}
              onclick={() => copyChosen(chosenItem)}
            >
              <Icon name={copied ? 'check' : 'copy'} size={12} />
              <span>{copied ? t('artifactsPane.copiedPath') : chosenItem.kind === 'report' ? t('chat.copyCode') : t('artifactsPane.copyPath')}</span>
            </button>
          {/if}
          {#if chosenItem.path}
            <button
              type="button"
              class="art-act-btn"
              title={t('artifactsPane.openFolder')}
              onclick={() => openFolder(chosenItem.path)}
            >
              <Icon name="folder" size={12} />
              <span>{t('artifactsPane.openFolder')}</span>
            </button>
          {/if}
        </div>
      </div>

      <div class="art-stage-body">
        {#if chosenItem.isPlan}
          <div class="art-plan-wrap">
            <PlanPane embedded={true} />
          </div>
        {:else if chosenItem.kind === 'report' && chosenItem.report}
          <div class="art-plan-wrap">
            <ReportPane report={chosenItem.report} />
          </div>
        {:else if contentLoading}
          <div class="art-loading">
            <Icon name="refreshCw" size={20} />
            <span>{t('deckRoom.loading')}</span>
          </div>
        {:else if activeContent}
          <div class="art-doc-wrap markdown-body">
            {@html renderMarkdown(activeContent)}
          </div>
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
  {/if}
</div>
</div>

<style>
  /* TWO PANES, ONE COLUMN WHEN THERE IS NO ROOM FOR TWO.
     The list is a fixed 220px beside the stage — which reads as a design
     until the pane is narrow, and then the stage is a sliver: the inspector's
     own floor is 320px (App.svelte, panels.inspector), so at the width this
     pane is allowed to be, the thing you clicked was a ~100px column beside a
     220px list. Clicking a row looked like clicking nothing (owner, 11 ก.ย.:
     *"ในนี้กดไม่ได้ ผมดูไม่ได้"* — about a page he had just watched the agent
     build).

     Below 560px (220 list + 340 stage, the width under which a stage is not
     worth looking at) the pane turns into a column: the list is a band across
     the top and the stage takes every pixel under it. The band sizes to its
     rows up to half the pane — a list of six rows is six rows tall, not a
     fixed 240px with three of them cut off (the first wrapped version did
     exactly that, and the send-off rows this list is now for were the ones
     below the fold). */
  .art-pane {
    width: 100%;
    height: 100%;
    min-height: 0;
    overflow: hidden;
    background: var(--surface-app);
    /* The pane is the container its own layout is asked about, because the
       width that matters here is this element's and not the viewport's. Named
       for the same reason .composer .box names its own: a query that says what
       it is measuring cannot be answered by a container somebody else added
       around it later (style.css, `container-name:composer`). */
    container-type: inline-size;
    container-name: artifacts;
  }

  /* The flex box is a child of the container and not the container itself,
     and that is load-bearing: a container query is answered by an ANCESTOR,
     so a rule on .art-pane inside `@container artifacts` never matches — the
     first cut of this column layout put `flex-direction` there, the list went
     full-width (a descendant, so its rule applied) and the stage stayed a
     sliver to its right. */
  .art-body {
    display: flex;
    width: 100%;
    height: 100%;
    min-height: 0;
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

  @container artifacts (max-width: 559px) {
    .art-body {
      flex-direction: column;
    }
    .art-list {
      width: 100%;
      flex: 0 1 auto;
      max-height: 52%;
      border-right: 0;
      border-bottom: 1px solid var(--border-default);
    }
    .art-stage {
      flex: 1 1 0;
      width: 100%;
    }
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

  /* Items list */
  .art-items {
    flex: 1;
    overflow-y: auto;
    padding: 4px 6px 6px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 0;
  }

  /* A group is a heading and its rows. The heading is set in the dimmest ink
     because it is a label, not content — the rows under it are what the eye
     is for. */
  .art-group {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding-bottom: 4px;
  }
  .art-group-head {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 10.5px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-dim);
    padding: 8px 8px 3px;
  }
  .art-group-count {
    padding: 0 5px;
    border-radius: 999px;
    background: var(--surface-raised);
    font-weight: 500;
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

  /* The send-off tag: where this row goes when pressed. Quiet at rest and
     lit on hover, so the list reads as one list and the difference shows
     when the hand is over it. */
  .art-handoff {
    margin-left: auto;
    flex: none;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 10.5px;
    color: var(--text-dim);
    padding: 2px 6px;
    border-radius: 999px;
    border: 1px solid var(--border-subtle);
    white-space: nowrap;
  }
  .art-item.sendoff:hover .art-handoff {
    color: var(--text-primary);
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
  .art-item-icon.kind-report {
    background: color-mix(in srgb, var(--status-warn, #f59e0b) 18%, transparent);
    color: var(--status-warn, #f59e0b);
  }
  .art-item-icon.kind-doc {
    background: color-mix(in srgb, #10b981 18%, transparent);
    color: #10b981;
  }
  .art-item-icon.kind-deck,
  .art-item-icon.kind-page {
    background: color-mix(in srgb, #8b5cf6 18%, transparent);
    color: #8b5cf6;
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

  /* Right Stage. 340px of basis is the other half of the rule above: it is
     what a document or a card needs to be worth looking at, and it is the
     number that decides where the column layout takes over. */
  .art-stage {
    flex: 1 1 340px;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--surface-app);
  }
  .art-stage-alone {
    flex: 1 1 100%;
    width: 100%;
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
    flex-wrap: wrap;
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
    white-space: nowrap;
  }

  .art-stage-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: none;
    flex-wrap: wrap;
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
    box-sizing: border-box;
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
    max-width: 260px;
    line-height: 1.5;
    margin: 0;
  }
</style>
