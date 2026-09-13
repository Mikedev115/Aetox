<script lang="ts">
  // เอเจนเฉพาะทาง (COMPANY.md §4): the roster, and the work the team has taken in.
  //
  // Two lists and no state of its own. A chair is a profile file, and the jobs
  // feed is a query over `jobs` — the rows every delegation already writes. The
  // page was specified as "a roster plus a feed, no new state, no inbox", and
  // that is exactly as much as this does: it reads, and it lets you walk to the
  // conversation a job came from.
  //
  // It was split in two and moved behind the ทีม door for about an hour on
  // 2026-08-20 (§158) and the owner sent it straight back: the page is where you
  // walk in to talk to a specialist, and that belongs beside the assistant. The
  // name is the only thing that stayed changed.
  //
  // Since 13 ก.ย. 2026 it is the talking room and nothing else (owner: "หน้า
  // นั้นจะเป็นเลือกคุยอย่างเดียว"). The gear on every card, the hiring button,
  // the folder link and two paragraphs of signposts all left for ตั้งค่า ›
  // เอเจนเฉพาะทาง, which is a page again; this page keeps one door there.
  // Configuring lived here for a day (12 ก.ย.: "คนอยู่หน้าแรก ทีมอยู่ตั้งค่า")
  // and it read as three pages pointing at each other — the editor still sat
  // in Settings, in a room with no row in the rail.
  import { onMount } from 'svelte'
  import {
    ListChairs, ListReceivedJobs, AgentGate, ListTeams,
  } from '../../wailsjs/go/main/App'
  import { engine, subagent } from '../../wailsjs/go/models'
  import { agoLabel, newChairSession, selectGlobalSession, setActiveView, openSettingsAt } from './stores/cockpit.svelte'
  import { t, type TKey } from './i18n.svelte'
  import { dayBucket } from './dayBucket'
  import Icon from './Icon.svelte'
  import AgentLock from './AgentLock.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import RankPip from './RankPip.svelte'
  import { lookOf } from './mascot/agentLook'

  let { onClose }: { onClose: () => void } = $props()

  let chairs = $state<engine.Chair[]>([])
  let jobs = $state<engine.ReceivedJob[]>([])
  let loaded = $state(false)
  // Which teammate's work the feed is showing. Empty is everyone, and the
  // filter row is only drawn once more than one name is in the feed — a
  // one-choice filter is furniture.
  let who = $state('')
  // Whether each teammate can work, and why not. Answered in Go
  // (App.AgentGate) so this page, ห้องงานวิดีโอ and the chat menu all draw the
  // same verdict on the same agent.
  let gates = $state<Record<string, engine.AgentGate>>({})
  // The roster waits for the verdicts rather than drawing ahead of them. A card
  // that appears usable and is veiled a moment later has already told the
  // reader something untrue; the empty moment is shorter than the wrong one.
  let gated = $state(false)

  async function loadNeeds(roster: engine.Chair[]) {
    const answers = await Promise.all(roster.map((c) => AgentGate(c.name)))
    gates = Object.fromEntries(roster.map((c, i) => [c.name, answers[i]]))
    gated = true
  }
  // Which teams name each agent (§256) — a chip on the card, nothing more.
  // Teams are configured in ตั้งค่า › ทีมเอเจน; this page is the people, each
  // drawn once, whichever lists they are on. Read alongside the roster and
  // never awaited by the cards: a slow read must not hold up the faces.
  let teamsOf = $state<Record<string, string[]>>({})
  async function loadTeamChips() {
    try {
      const next: Record<string, string[]> = {}
      for (const tm of await ListTeams('')) {
        for (const m of tm.members) (next[m.name] ??= []).push(tm.name)
      }
      teamsOf = next
    } catch {
      teamsOf = {}
    }
  }

  onMount(async () => {
    const [roster, feed] = await Promise.all([ListChairs(), ListReceivedJobs(30), loadTeamChips()])
    chairs = roster
    jobs = feed
    await loadNeeds(roster)
    loaded = true
  })
  // Walking from a job to the conversation that sent it. The job row carries
  // the caller's session id, which is the only link there is — and the only one
  // there needs to be, since the file it produced went to that session's folder.
  async function openSource(job: engine.ReceivedJob) {
    if (!job.sessionId) return
    // The view moves first and the transcript follows. Loading a session
    // switches project, workbench and history behind it — leaving the user
    // looking at this page until all of that lands reads as a dead click.
    setActiveView('chat')
    await selectGlobalSession({ id: job.sessionId, title: '', ago: '' })
  }

  // A duration is only worth a slot when it says something. Every row printing
  // "0.0s" is six copies of "this was instant" competing with the line that
  // says what the job was.
  function secs(ms: number): string {
    return ms >= 100 ? `${(ms / 1000).toFixed(1)}s` : ''
  }

  // The face a job wears is its author's, resolved off the roster so one agent
  // cannot show two faces on one page. Only what the profile CHOSE needs
  // resolving — the badge on its ears and, if its owner said so, its hue,
  // shell, top light and resting face. The robot underneath is drawn from the
  // name, so a job whose profile has since been deleted keeps the same colour
  // and loses nothing but its badge.
  const faces = $derived(new Map(chairs.map((c) => [c.name, lookOf(c)])))
  function jobFace(name: string) {
    return faces.get(name) ?? {}
  }

  // Who is in the feed, in the order the roster lists them — so the filter row
  // reads in the same order as the cards above it.
  const feedNames = $derived.by(() => {
    const seen = new Set(jobs.map((j) => j.chair))
    const ordered = chairs.map((c) => c.name).filter((n) => seen.has(n))
    for (const n of seen) if (!ordered.includes(n)) ordered.push(n)
    return ordered
  })

  // Grouped by calendar day, because a column of "2 วัน" on every row is a wall
  // rather than a list. The rows arrive newest-first from Go, so consecutive
  // runs are already the groups.
  const feedGroups = $derived.by(() => {
    const out: { key: TKey; items: engine.ReceivedJob[] }[] = []
    for (const j of jobs) {
      if (who && j.chair !== who) continue
      const key = dayBucket(j.time)
      const last = out.at(-1)
      if (last && last.key === key) last.items.push(j)
      else out.push({ key, items: [j] })
    }
    return out
  })
  // Walking into an agent's room (§85): a fresh session bound to that agent —
  // its tools, its memory, its prompt. The view moves first for the same
  // reason openSource's does: a click that waits for a bootstrap before
  // showing anything reads as a dead click. The engine picks the team that
  // seats the chair (App.seatingTeam): this page does not know teams.
  async function talkTo(chair: engine.Chair) {
    setActiveView('chat')
    await newChairSession(chair.name)
  }

</script>

<div class="page-shell">
  <header class="page-head">
    <button class="settings-back" onclick={onClose}><Icon name="arrowLeft" size={14} /> {t('settings.backToApp')}</button>
    <div class="page-title">
      <h2>{t('desk.office')}</h2>
      <p>{t('office.intro')}</p>
    </div>
  </header>

  <div class="page-body">
    <div class="settings-inner">
      <!-- One door out, on the section and not on every card: everything
           about a person that is not talking to them — model, tools, the file,
           hiring a new one — is ตั้งค่า › เอเจนเฉพาะทาง's. -->
      <div class="sec-head">
        <div class="eyebrow section-label">{t('office.roster')}</div>
        <span class="ag-reach"></span>
        <button class="ctrl" onclick={() => openSettingsAt('team')}><Icon name="settings" size={13} /> {t('office.configure')} <Icon name="arrowRight" size={13} /></button>
      </div>

      <!-- A face, not an inventory. The tool chips were six per card and five
           of the six were the same on every card — the office ceiling hands
           everyone the same set, so the list said nothing about who anyone is
           while taking half the card to say it. What is left is what the card
           is for: who this is, what they make, and whether they have done any
           of it. The tools are on the settings page, which is also the only
           place they can be changed.

           No switch, no band, no team section (owner, 12 ก.ย.: "คนอยู่หน้าแรก
           ทีมอยู่ตั้งค่า"). Whether the assistant may hand an agent work is a
           fact about a TEAM now, and it is switched in ตั้งค่า › ทีมเอเจน; a
           switch here would be the same fact in a second place. -->
      <div class="office-grid">
        {#each gated ? chairs : [] as c (c.name)}
          {@const locked = gates[c.name]?.blocked ?? false}
          <div class="chair-card agc" class:locked>
            <div class="chair-body">
              <div class="chair-who">
                <AgentMascot name={c.name} {...lookOf(c)} size={38} />
                <!-- The rank under the name (RankPip), in the name's own
                     column so neither clips the other: a roster is the one
                     page that mixes who you may talk to, so each face says
                     which level it is before where it came from. -->
                <span class="chair-id">
                  <span class="chair-name">{c.name}</span>
                  <RankPip tier="agent" />
                </span>
                <!-- On the head, not in the chips row (owner, 13 ก.ย. 2026:
                     "แปะหัวด้วยชัดๆ อันไหนมากับแอป"): which faces came with
                     the app and which the user hired is the first thing to
                     know about a roster that mixes both, so it sits beside
                     the name where the eye already is. -->
                {#if c.builtin}<span class="chip builtin">{t('office.builtin')}</span>{/if}
              </div>
              <p class="chair-desc">{c.description}</p>
              <!-- Only facts that DIFFER between agents and matter to someone
                   choosing whom to talk to: the work it has done, and since
                   §256 the teams that name it. Whether its file is edited is
                   a fact about configuring, and lives with the configuring. -->
              <div class="chair-chips">
                {#if c.jobs > 0}
                  <span class="chair-stat"><span class="n">{c.jobs}</span> {t('office.jobsDone')} · {agoLabel(c.lastUsed ?? '')}</span>
                {:else}
                  <span class="chair-stat idle">{t('office.neverUsed')}</span>
                {/if}
                {#if (teamsOf[c.name] ?? []).length > 0}
                  <span class="chair-stat teams" title={t('office.teamsOfTip')}><Icon name="users" size={11} /> {(teamsOf[c.name] ?? []).join(' · ')}</span>
                {/if}
              </div>
            </div>
            <!-- The one thing this page is for: walking in and talking to a
                 specialist. Named with the agent, not "this agent", because
                 that is what walking in is — and the row cannot overflow. -->
            <div class="chair-foot">
              <button class="chair-talk" onclick={() => talkTo(c)}>
                <Icon name="messageSquare" size={14} />
                <span class="t">{t('office.chatWith', { name: c.name })}</span>
              </button>
            </div>
            <AgentLock agent={c.name} label={c.name} gate={gates[c.name] ?? null}
              onInstalled={() => loadNeeds(chairs)} />
          </div>
        {/each}
        {#if loaded && chairs.length === 0}
          <div class="chair-card empty"><div class="chair-body"><p class="chair-desc">{t('office.noChairs')}</p></div></div>
        {/if}
      </div>

      <div class="sec-head feed-head">
        <div class="eyebrow section-label">{t('office.feed')}</div>
        <!-- Filtering is the question this list is actually asked once more
             than one teammate has worked: "what has doc been doing?". Drawn
             only when the answer could differ from the whole list. -->
        {#if feedNames.length > 1}
          <div class="feed-filter">
            <button class="pill" class:on={who === ''} onclick={() => (who = '')}>{t('office.filterAll')}</button>
            {#each feedNames as n (n)}
              <button class="pill" class:on={who === n} onclick={() => (who = n)}>{n}</button>
            {/each}
          </div>
        {/if}
      </div>
      <!-- One row per job, one line each. The row is the door — a boxed button
           repeated down the right edge was the loudest thing on a page whose
           subject is the left-hand line it sat beside. -->
      {#each feedGroups as g (g.key)}
        <div class="feed-day">{t(g.key)}</div>
        <div class="settings-card feed-card">
          {#each g.items as j (j.id)}
            <button class="job-row" disabled={!j.sessionId}
              aria-label={t('office.openSource')} onclick={() => openSource(j)}>
              <!-- The same face as the card above it. The feed names who did
                   the work, so drawing them a second way here would make one
                   agent two people on one page. -->
              <AgentMascot name={j.chair} {...jobFace(j.chair)} size={22} />
              <!-- The line the caller wrote, not the arguments the tool call
                   carried. `request` is the machine's copy and stays available
                   on hover for anyone who wants it. -->
              <span class="job-brief" title={j.request}>{j.brief || j.request}</span>
              {#if j.outcome === 'good'}<span class="ok"><Icon name="thumbsUp" size={12} /></span>{/if}
              {#if j.outcome === 'bad'}<span class="bad"><Icon name="thumbsDown" size={12} /></span>{/if}
              <span class="job-meta">{t('office.toolCalls', { n: j.toolCount })}</span>
              {#if secs(j.durationMs)}<span class="job-meta">{secs(j.durationMs)}</span>{/if}
              {#if j.sessionId}
                <span class="job-go">{t('office.openSource')}</span>
                <Icon name="chevronRight" size={14} />
              {/if}
            </button>
          {/each}
        </div>
      {/each}
      {#if loaded && feedGroups.length === 0}
        <div class="settings-card feed-card">
          <div class="set-row"><div class="set-txt"><div class="d">{who ? t('office.noJobsFor', { name: who }) : t('office.noJobs')}</div></div></div>
        </div>
      {/if}
    </div>
  </div>
</div>
