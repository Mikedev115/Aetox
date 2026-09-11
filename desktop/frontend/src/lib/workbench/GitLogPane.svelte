<script lang="ts">
  // The history, as a room (§250).
  //
  // GitPane answers "where does my repository stand right now". This answers
  // the one it cannot: "how did it get here" — the commits, newest first,
  // grouped by the day they landed, each one openable down to the hunks.
  //
  // Everything here is paid for by what the user has looked at, and only that
  // (owner, 12 ก.ย.: "อย่าให้มันหนักเกิน แบ่งโหลดด้วย"):
  //
  // - A page is fifty commits from one git process. The next fifty come when
  //   the button at the bottom is pressed, never on scroll — a timeline of a
  //   thousand commits is an ordinary repository, and the last nine hundred
  //   are nobody's question until they are.
  // - A row's files arrive when the row is opened; a file's hunks when the
  //   file is. Both are kept once fetched, so closing and reopening costs
  //   nothing.
  // - No timer. History moves when a commit lands, so the first page is
  //   re-read on exactly three occasions: the tab coming to the front, the git
  //   room reporting a commit (codeStatus.commitsLanded), and the end of a
  //   turn — the one way the agent itself can move HEAD. A re-read that finds
  //   the same HEAD keeps every page already loaded rather than throwing them
  //   away and starting over.
  // - Search and the type chips filter what is loaded, in the window. They
  //   are not a query to git: a search over a thousand commits is a different
  //   feature with a different cost, and the pane says so when a search finds
  //   nothing in what it has.
  import { untrack } from 'svelte'
  import { GitLog, GitCommitChanges, GitCommitFileDiff } from '../../../wailsjs/go/main/App'
  import { main } from '../../../wailsjs/go/models'
  import { cockpit, sendUserMessage, setActiveView } from '../stores/cockpit.svelte'
  import { openFileTab, openGitTab } from '../stores/workbench.svelte'
  import { codeStatus } from '../stores/codeStatus.svelte'
  import { t, i18n } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import CodeDiff from '../CodeDiff.svelte'
  import { parseSubject, groupByDay, CHIP_TYPES, type CommitType, type ParsedSubject } from './gitLog'

  let { active = true }: { active?: boolean } = $props()

  const PAGE = 50

  let commits = $state<main.GitCommit[]>([])
  let more = $state(false)
  let loaded = $state(false)
  let loadingMore = $state(false)
  let open = $state<Record<string, boolean>>({})
  let files = $state<Record<string, main.GitFileChange[]>>({})
  let filesLoading = $state<Record<string, boolean>>({})
  let openFile = $state<Record<string, boolean>>({})
  let diffs = $state<Record<string, string>>({})
  let diffLoading = $state<Record<string, boolean>>({})
  let filter = $state<'all' | CommitType>('all')
  let query = $state('')
  let copied = $state<string | null>(null)

  const branch = $derived(cockpit.project.branch || '')

  // ---- reading ----------------------------------------------------------

  // One first-page read at a time, for the same reason GitPane guards its
  // poll: two in flight would race to write `commits`, and the older could
  // land last.
  let reading = false
  // Set when a reason to re-read arrived while the tab was behind another;
  // spent the moment it comes to the front. A read nobody is looking at is
  // the load this room was asked not to carry.
  let stale = false

  async function refresh() {
    if (reading) return
    reading = true
    try {
      const page = await GitLog('', PAGE)
      const fresh = page?.commits ?? []
      // Same HEAD, same page: keep every page already loaded. Only a moved
      // HEAD — a commit landed, or the branch changed under us — resets the
      // list to the new first page.
      if (loaded && fresh[0]?.hash === commits[0]?.hash && fresh.length === Math.min(commits.length, PAGE)) return
      commits = fresh
      more = !!page?.more
      loaded = true
    } finally {
      reading = false
    }
  }

  async function loadMore() {
    const last = commits[commits.length - 1]
    if (!last || loadingMore) return
    loadingMore = true
    try {
      const page = await GitLog(last.hash, PAGE)
      commits = [...commits, ...(page?.commits ?? [])]
      more = !!page?.more
    } finally {
      loadingMore = false
    }
  }

  $effect(() => {
    if (!active) return
    untrack(() => {
      if (!loaded || stale) {
        stale = false
        void refresh()
      }
    })
  })

  // A commit from the git room, or the end of a turn: HEAD may have moved.
  let lastLanded = codeStatus.commitsLanded
  let wasWorking = false
  $effect(() => {
    const landed = codeStatus.commitsLanded
    const working = cockpit.awaitingReply
    const reason = landed !== lastLanded || (wasWorking && !working)
    lastLanded = landed
    wasWorking = working
    if (!reason) return
    untrack(() => {
      if (active) void refresh()
      else stale = true
    })
  })

  async function toggle(hash: string) {
    if (open[hash]) {
      open[hash] = false
      return
    }
    open[hash] = true
    if (files[hash] === undefined) {
      filesLoading[hash] = true
      try {
        files[hash] = (await GitCommitChanges(hash)) ?? []
      } finally {
        filesLoading[hash] = false
      }
    }
  }

  const fileKey = (hash: string, path: string) => `${hash}::${path}`

  async function toggleFile(hash: string, path: string) {
    const key = fileKey(hash, path)
    if (openFile[key]) {
      openFile[key] = false
      return
    }
    openFile[key] = true
    if (diffs[key] === undefined) {
      diffLoading[key] = true
      try {
        diffs[key] = (await GitCommitFileDiff(hash, path)) ?? ''
      } finally {
        diffLoading[key] = false
      }
    }
  }

  async function copyHash(hash: string) {
    try {
      await navigator.clipboard.writeText(hash)
      copied = hash
      setTimeout(() => { if (copied === hash) copied = null }, 1500)
    } catch {
      /* clipboard refused; the hash is on screen to select by hand */
    }
  }

  function ask(c: main.GitCommit) {
    setActiveView('chat')
    void sendUserMessage(t('gitlog.askPrompt', { hash: c.short, subject: c.subject }))
  }

  // ---- reading the subject ----------------------------------------------

  type Row = main.GitCommit & ParsedSubject & { date: Date }

  const rows = $derived<Row[]>(commits.map((c) => ({ ...c, ...parseSubject(c.subject), date: new Date(c.at) })))

  const counts = $derived(rows.reduce((acc, r) => {
    acc[r.type] = (acc[r.type] ?? 0) + 1
    return acc
  }, {} as Record<CommitType, number>))

  const visible = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return rows.filter((r) =>
      (filter === 'all' || r.type === filter) &&
      (!q || r.subject.toLowerCase().includes(q) || r.hash.startsWith(q) || r.author.toLowerCase().includes(q)),
    )
  })

  const days = $derived(groupByDay(visible))

  const filtering = $derived(filter !== 'all' || query.trim() !== '')

  function dayLabel(d: Date): string {
    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    const that = new Date(d.getFullYear(), d.getMonth(), d.getDate())
    const ago = Math.round((today.getTime() - that.getTime()) / 86_400_000)
    const date = new Intl.DateTimeFormat(i18n.locale === 'th' ? 'th-TH' : i18n.locale, { day: 'numeric', month: 'short', year: 'numeric' }).format(d)
    if (ago === 0) return `${t('gitlog.today')} · ${date}`
    if (ago === 1) return `${t('gitlog.yesterday')} · ${date}`
    return date
  }

  const hhmm = (d: Date) => `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  const name = (path: string) => path.split('/').pop() ?? path
  const dir = (path: string) => {
    const cut = path.lastIndexOf('/')
    return cut < 0 ? '' : path.slice(0, cut)
  }
</script>

<div class="gitlog">
  <div class="gp-head">
    <span class="gp-where">
      <Icon name="gitBranch" size={13} />
      {#if branch}<b>{branch}</b>{/if}
      <span class="gp-arrow">→</span>
      <span>{t('gitlog.title')}</span>
    </span>
    <span class="gp-right">
      <label class="gl-search">
        <Icon name="search" size={12} />
        <input type="text" bind:value={query} placeholder={t('gitlog.search')} aria-label={t('gitlog.search')} />
      </label>
      <button class="icobtn tiny" aria-label={t('git.refresh')} data-tip={t('git.refresh')} onclick={() => { loaded = false; void refresh() }}>
        <Icon name="loaderCircle" size={13} />
      </button>
    </span>
  </div>

  <div class="gp-note">{t('git.codeDeskOnly')}</div>

  {#if loaded && commits.length > 0}
    <div class="gl-chips" role="group" aria-label={t('gitlog.filter')}>
      <button class="gl-chip" class:on={filter === 'all'} onclick={() => (filter = 'all')}>
        {t('gitlog.all')}<span class="n">{rows.length}</span>
      </button>
      {#each CHIP_TYPES as kind}
        {#if counts[kind]}
          <button class="gl-chip" class:on={filter === kind} onclick={() => (filter = kind)}>
            {kind === 'other' ? t('gitlog.other') : kind}<span class="n">{counts[kind]}</span>
          </button>
        {/if}
      {/each}
    </div>
  {/if}

  <div class="gl-scroll">
    {#if !loaded}
      <div class="gp-empty">{t('gitlog.loading')}</div>
    {:else if commits.length === 0}
      <div class="gp-empty">{t('gitlog.empty')}</div>
    {:else}
      {#if !filtering && codeStatus.gitChangedCount > 0}
        <!-- The working tree, as the node above HEAD: what the next commit will
             be, and the door to the room that makes it. -->
        <button class="gl-ghost" onclick={openGitTab}>
          <span class="dot"></span>
          <span>{t('gitlog.uncommitted', { n: codeStatus.gitChangedCount })}</span>
          <span class="gp-stat"><span class="add">+{codeStatus.gitAdded}</span><span class="del">-{codeStatus.gitRemoved}</span></span>
          <span class="go">{t('gitlog.openGit')} <Icon name="arrowRight" size={12} /></span>
        </button>
      {/if}

      {#if visible.length === 0}
        <div class="gp-empty">{more ? t('gitlog.noMatchMore') : t('gitlog.noMatch')}</div>
      {/if}

      {#each days as day (day.key)}
        <div class="gl-day">{dayLabel(day.date)} · {t('gitlog.commits', { n: day.rows.length })}</div>
        <div class="gl-rail">
          {#each day.rows as c (c.hash)}
            <div class="gl-commit">
              <button
                class="gl-row" class:open={open[c.hash]} class:head={!filtering && c.hash === commits[0]?.hash} class:merge={c.merge}
                aria-expanded={!!open[c.hash]} title={c.subject}
                onclick={() => toggle(c.hash)}
              >
                <span class="dot"></span>
                <span class="gl-hash">{c.short}</span>
                <span class="gl-tag {c.merge ? 'merge' : c.type}">{c.merge ? 'merge' : (c.word || t('gitlog.other'))}</span>
                <span class="gl-subj">{c.text}</span>
                <span class="gl-time">{hhmm(c.date)}</span>
                <span class="gp-stat"><span class="add">+{c.added}</span><span class="del">-{c.removed}</span></span>
              </button>

              {#if open[c.hash]}
                <div class="gl-detail">
                  <div class="gl-meta">
                    <span class="who">{c.author}</span>
                    {#if c.scope}<span class="gl-scope">{c.scope}</span>{/if}
                    <span>{t('gitlog.files', { n: c.files })}</span>
                    <span class="full">{c.hash.slice(0, 12)}</span>
                    <button class="gl-act" onclick={() => copyHash(c.hash)}>
                      <Icon name="copy" size={11} /> {copied === c.hash ? t('gitlog.copied') : t('gitlog.copyHash')}
                    </button>
                    <button class="gl-act ask" onclick={() => ask(c)}>
                      <Icon name="sparkles" size={11} /> {t('gitlog.ask')}
                    </button>
                  </div>

                  {#if filesLoading[c.hash]}
                    <div class="gp-skel" role="status" aria-label={t('gitlog.loading')}>
                      <i class="w1"></i><i class="w2"></i><i class="w3"></i>
                    </div>
                  {:else}
                    <div class="gl-files">
                      {#each files[c.hash] ?? [] as f (f.path)}
                        {@const key = fileKey(c.hash, f.path)}
                        <div class="gp-file">
                          <button
                            class="gp-row" class:busy={diffLoading[key]}
                            aria-expanded={!!openFile[key]} aria-busy={diffLoading[key] || undefined}
                            onclick={() => toggleFile(c.hash, f.path)}
                          >
                            <span class="gp-caret">
                              {#if diffLoading[key]}
                                <i class="gp-ring"></i>
                              {:else}
                                <Icon name={openFile[key] ? 'chevronDown' : 'chevronRight'} size={12} />
                              {/if}
                            </span>
                            <span class="gp-name">{name(f.path)}</span>
                            {#if dir(f.path)}<span class="gp-dir">{dir(f.path)}</span>{/if}
                            <span class="gp-stat">
                              <span class="add">+{f.added ?? 0}</span><span class="del">-{f.removed ?? 0}</span>
                            </span>
                            <span class="gp-badge {f.status}">{f.status}</span>
                          </button>
                          {#if openFile[key]}
                            <div class="gp-diff">
                              {#if diffLoading[key]}
                                <div class="gp-skel" role="status" aria-label={t('git.loading')}>
                                  <i class="w1"></i><i class="w2"></i><i class="w3"></i>
                                </div>
                              {:else if diffs[key]}
                                <CodeDiff diff={diffs[key]} />
                                {#if f.status !== 'D'}
                                  <button class="gp-open" onclick={() => openFileTab(f.path, name(f.path))}>
                                    <Icon name="fileCode" size={12} /> {t('git.openFile')}
                                  </button>
                                {/if}
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
              {/if}
            </div>
          {/each}
        </div>
      {/each}

      {#if more}
        <button class="gl-more" disabled={loadingMore} onclick={loadMore}>
          {loadingMore ? t('gitlog.loading') : t('gitlog.loadMore', { n: PAGE })}
        </button>
      {:else if !filtering}
        <div class="gl-end">{t('gitlog.end')}</div>
      {/if}
    {/if}
  </div>
</div>
