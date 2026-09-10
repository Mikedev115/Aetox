<script lang="ts">
  // Pull requests, as a room.
  //
  // The `pr` tool answers the model in sentences; this answers the person in
  // rows, from the same fetcher (internal/github, through desktop/pr_room.go).
  // What it adds that the tool cannot is seeing without asking: a CI result on
  // screen while you work is a different thing from one you have to remember to
  // go and request.
  //
  // Shaped after GitPane deliberately — same header, same collapsed rows, same
  // fetch-on-expand — because it is the same kind of question about the same
  // repository, and two rooms that answer alike should look alike.
  //
  // Nothing here computes a diff. GitHub hands back its own unified patch per
  // file and CodeDiff draws it, so a pull request's hunks look identical to the
  // ones under a chat row and to the ones in the Git pane.
  import { onMount } from 'svelte'
  import {
    PullRequests,
    PullRequestsState,
    PullRequestFiles,
    PullRequestChecks,
    CreatePullRequest,
    SuggestPRDetails,
    ReviewPullRequest,
  } from '../../../wailsjs/go/main/App'
  import { main, github } from '../../../wailsjs/go/models'
  import { cockpit } from '../stores/cockpit.svelte'
  import { t } from '../i18n.svelte'
  import Icon from '../Icon.svelte'
  import CodeDiff from '../CodeDiff.svelte'

  let room = $state<main.PRRoom | null>(null)
  let loaded = $state(false)
  let filterState = $state<'open' | 'closed'>('open')
  let open = $state<Record<number, boolean>>({})
  let files = $state<Record<number, github.PRFile[]>>({})
  let loading = $state<Record<number, boolean>>({})
  let openFile = $state<Record<string, boolean>>({})
  let checks = $state<Record<string, github.CheckRun[]>>({})

  // AI Review state per PR
  let reviewing = $state<Record<number, boolean>>({})
  let reviews = $state<Record<number, string>>({})
  let openReview = $state<Record<number, boolean>>({})

  const items = $derived(room?.items ?? [])

  // Opening one from here.
  let opening = $state(false)
  let form = $state({ title: '', head: '', base: '', body: '', draft: false })
  let submitting = $state(false)
  let drafting = $state(false)
  let formError = $state('')

  function startOpening() {
    form = { title: '', head: cockpit.project.branch || '', base: '', body: '', draft: false }
    formError = ''
    opening = true
  }

  async function handleAIDraft() {
    drafting = true
    formError = ''
    try {
      const sug = await SuggestPRDetails(form.head, form.base)
      if (sug?.title) form.title = sug.title
      if (sug?.body) form.body = sug.body
    } catch (err: any) {
      formError = String(err?.message ?? err)
    } finally {
      drafting = false
    }
  }

  async function handleAIReview(num: number) {
    if (reviews[num]) {
      openReview[num] = !openReview[num]
      return
    }
    reviewing[num] = true
    try {
      const res = await ReviewPullRequest(num)
      reviews[num] = res
      openReview[num] = true
    } catch (err: any) {
      reviews[num] = String(err?.message ?? err)
      openReview[num] = true
    } finally {
      reviewing[num] = false
    }
  }

  async function submit() {
    if (!form.title.trim() || !form.head.trim() || submitting) return
    submitting = true
    formError = ''
    try {
      const created = await CreatePullRequest(form.title, form.head, form.base, form.body, form.draft)
      if (created.error) {
        formError = created.error
        return
      }
      opening = false
      filterState = 'open'
      await refresh()
    } finally {
      submitting = false
    }
  }

  async function setFilter(next: 'open' | 'closed') {
    if (filterState === next) return
    filterState = next
    loaded = false
    await refresh()
  }

  async function refresh() {
    room = await PullRequestsState(filterState)
    loaded = true
    for (const pr of room?.items ?? []) void loadChecks(pr.headSHA)
  }

  async function loadChecks(sha: string) {
    if (!sha || checks[sha]) return
    checks[sha] = await PullRequestChecks(sha)
  }

  function fileKey(n: number, path: string): string {
    return n + ':' + path
  }

  function toggleFile(n: number, path: string) {
    const k = fileKey(n, path)
    openFile[k] = !openFile[k]
  }

  async function toggle(pr: github.PullRequest) {
    const n = pr.number
    open[n] = !open[n]
    if (!open[n] || files[n] || loading[n]) return
    loading[n] = true
    try {
      files[n] = await PullRequestFiles(n)
    } finally {
      loading[n] = false
    }
  }

  function verdict(sha: string): 'fail' | 'running' | 'pass' | '' {
    const runs = checks[sha]
    if (!runs || runs.length === 0) return ''
    if (runs.some((r) => r.status === 'completed' && !passed(r))) return 'fail'
    if (runs.some((r) => r.status !== 'completed')) return 'running'
    return 'pass'
  }

  function passed(r: github.CheckRun): boolean {
    return r.conclusion === 'success' || r.conclusion === 'neutral' || r.conclusion === 'skipped'
  }

  function failedNames(sha: string): string {
    return (checks[sha] ?? [])
      .filter((r) => r.status === 'completed' && !passed(r))
      .map((r) => `${r.name} (${r.conclusion || 'failed'})`)
      .join('\n')
  }

  onMount(refresh)
</script>

<div class="pr-pane">
  <div class="pr-head">
    <span class="repo"><Icon name="gitBranch" size={13} /> {room?.repo || '—'}</span>
    {#if loaded && !room?.reason}
      <span class="count">{t('prPane.count', { count: String(items.length) })}</span>
    {/if}
    <button type="button" class="icobtn" title={t('prPane.refresh')} onclick={refresh}>
      <Icon name="loaderCircle" size={13} />
    </button>
    {#if loaded && !room?.reason}
      <button type="button" class="icobtn" title={t('prPane.newTitle')} onclick={startOpening}>
        <Icon name="plus" size={13} />
      </button>
    {/if}
  </div>

  <p class="pr-scope">{t('prPane.scope')}</p>

  {#if loaded && !room?.reason}
    <!-- Open vs Closed Tabs -->
    <div class="pr-tabs" role="tablist">
      <button
        type="button"
        class="pr-tab"
        class:active={filterState === 'open'}
        onclick={() => setFilter('open')}
      >
        <Icon name="check" size={12} />
        <span>{t('prPane.tabOpen')}</span>
      </button>
      <button
        type="button"
        class="pr-tab"
        class:active={filterState === 'closed'}
        onclick={() => setFilter('closed')}
      >
        <Icon name="package" size={12} />
        <span>{t('prPane.tabClosed')}</span>
      </button>
    </div>
  {/if}

  {#if opening}
    <div class="pr-form">
      <div class="pr-title-row">
        <input class="pr-in" placeholder={t('prPane.newTitleField')} bind:value={form.title} />
        <button
          type="button"
          class="pr-ai-draft-btn"
          disabled={drafting}
          onclick={handleAIDraft}
          title={t('prPane.aiDraft')}
        >
          <Icon name="sparkles" size={12} />
          <span>{drafting ? t('prPane.aiDrafting') : t('prPane.aiDraft')}</span>
        </button>
      </div>

      <div class="pr-branch-row">
        <input class="pr-in mono" placeholder={t('prPane.newHead')} bind:value={form.head} />
        <span class="arrow">→</span>
        <input class="pr-in mono" placeholder={t('prPane.newBase')} bind:value={form.base} />
      </div>

      <textarea class="pr-in pr-body" rows="4" placeholder={t('prPane.newBody')} bind:value={form.body}></textarea>

      <label class="pr-draft"><input type="checkbox" bind:checked={form.draft} /> {t('prPane.newDraft')}</label>

      {#if formError}<p class="pr-error">{formError}</p>{/if}

      <div class="pr-form-buttons">
        <button type="button" class="pr-cancel" onclick={() => (opening = false)}>{t('prPane.newCancel')}</button>
        <button
          type="button"
          class="pr-submit"
          disabled={submitting || !form.title.trim() || !form.head.trim()}
          onclick={submit}
        >{submitting ? t('prPane.newOpening') : t('prPane.newConfirm')}</button>
      </div>
      <p class="pr-hint">{t('prPane.newPushFirst')}</p>
    </div>
  {/if}

  {#if !loaded}
    <p class="pr-empty">{t('prPane.loading')}</p>
  {:else if room?.reason}
    <p class="pr-empty">{room.reason}</p>
    {#if !room.connected}
      <p class="pr-empty hint">{t('prPane.connect')}</p>
    {/if}
  {:else if items.length === 0}
    <!-- Smart Empty State -->
    <div class="pr-smart-empty">
      <div class="pr-empty-icon">
        <Icon name="gitBranch" size={18} />
      </div>
      <div class="pr-empty-branch">
        <Icon name="gitBranch" size={11} />
        <span>{cockpit.project.branch || 'main'}</span>
      </div>
      <p class="pr-empty-title">
        {filterState === 'open' ? t('prPane.emptyNoOpen') : t('prPane.emptyNoClosed')}
      </p>
      {#if filterState === 'open'}
        <button
          type="button"
          class="pr-empty-cta"
          onclick={startOpening}
        >
          <Icon name="plus" size={13} />
          <span>{t('prPane.createFromBranch')}</span>
        </button>
        <button
          type="button"
          class="pr-empty-alt"
          onclick={() => setFilter('closed')}
        >
          {t('prPane.viewClosed')} →
        </button>
      {:else}
        <button
          type="button"
          class="pr-empty-alt"
          onclick={() => setFilter('open')}
        >
          ← {t('prPane.viewOpen')}
        </button>
      {/if}
    </div>
  {:else}
    <div class="pr-list">
      {#each items as pr (pr.number)}
        <div class="pr-row">
          <button type="button" class="pr-title" onclick={() => toggle(pr)}>
            <span class="chev"><Icon name={open[pr.number] ? 'chevronDown' : 'chevronRight'} size={12} /></span>
            <span class="num">#{pr.number}</span>
            <span class="ttl">{pr.title}</span>
            {#if pr.state === 'closed'}
              <span class="tag closed">{t('prPane.closed')}</span>
            {:else if pr.draft}
              <span class="tag">{t('prPane.draft')}</span>
            {/if}
            {#if verdict(pr.headSHA)}
              <span class="ci {verdict(pr.headSHA)}" title={failedNames(pr.headSHA)}>
                {verdict(pr.headSHA) === 'fail' ? '✗' : verdict(pr.headSHA) === 'running' ? '⋯' : '✓'}
              </span>
            {/if}
            <span class="stat"><span class="add">+{pr.additions}</span> <span class="del">-{pr.deletions}</span></span>
          </button>
          <div class="pr-branches">{pr.headRef} → {pr.baseRef}</div>
          {#if open[pr.number]}
            {#if loading[pr.number]}
              <p class="pr-empty">{t('prPane.loading')}</p>
            {:else}
              {#each files[pr.number] ?? [] as f (f.path)}
                <div class="pr-file">
                  <button
                    type="button" class="pr-file-head"
                    aria-expanded={!!openFile[fileKey(pr.number, f.path)]}
                    onclick={() => toggleFile(pr.number, f.path)}
                  >
                    <span class="chev"><Icon name={openFile[fileKey(pr.number, f.path)] ? 'chevronDown' : 'chevronRight'} size={12} /></span>
                    <span class="st">{f.status}</span>
                    <span class="path">{f.path}</span>
                    <span class="stat"><span class="add">+{f.additions}</span> <span class="del">-{f.deletions}</span></span>
                  </button>
                  {#if openFile[fileKey(pr.number, f.path)]}
                    {#if f.patch}
                      <CodeDiff diff={f.patch} />
                    {:else}
                      <p class="pr-empty">{t('prPane.noPatch')}</p>
                    {/if}
                  {/if}
                </div>
              {/each}
            {/if}

            <div class="pr-actions-row">
              <button
                type="button"
                class="pr-ai-review-btn"
                disabled={reviewing[pr.number]}
                onclick={() => handleAIReview(pr.number)}
              >
                <Icon name="sparkles" size={12} />
                <span>{reviewing[pr.number] ? t('prPane.aiReviewing') : t('prPane.aiReview')}</span>
              </button>
              <a class="pr-link" href={pr.url} target="_blank" rel="noreferrer">{t('prPane.openOnGitHub')}</a>
            </div>

            {#if openReview[pr.number] && reviews[pr.number]}
              <div class="pr-ai-review-box">
                <div class="pr-ai-review-head">
                  <span>
                    <Icon name="sparkles" size={12} /> {t('prPane.aiReviewTitle')}
                  </span>
                  <button
                    type="button"
                    class="icobtn tiny"
                    onclick={() => (openReview[pr.number] = false)}
                  >
                    <Icon name="check" size={12} />
                  </button>
                </div>
                <div class="pr-ai-review-content">{reviews[pr.number]}</div>
              </div>
            {/if}
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
