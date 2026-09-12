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
  import { onMount } from 'svelte'
  // The hiring door opens the agents' home. Since the homes split, which
  // folder a file lands in is which kind it is — a chair file dropped into the
  // sub-agents' folder would wake up sick.
  import {
    ListChairs, ListReceivedJobs, OpenAgentsFolder, OpenTeamsFolder, AgentGate,
    DelegateSwitches, SetDelegateOff, ListTeams, SaveTeam, DeleteTeam,
  } from '../../wailsjs/go/main/App'
  import { main, subagent } from '../../wailsjs/go/models'
  import { agoLabel, cockpit, newChairSession, selectGlobalSession, setActiveView } from './stores/cockpit.svelte'
  import { t, type TKey } from './i18n.svelte'
  import { dayBucket } from './dayBucket'
  import Icon from './Icon.svelte'
  import AgentLock from './AgentLock.svelte'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import { lookOf } from './mascot/agentLook'

  let { onClose }: { onClose: () => void } = $props()

  let chairs = $state<main.Chair[]>([])
  let jobs = $state<main.ReceivedJob[]>([])
  let loaded = $state(false)
  // Which teammate's work the feed is showing. Empty is everyone, and the
  // filter row is only drawn once more than one name is in the feed — a
  // one-choice filter is furniture.
  let who = $state('')
  // Whether each teammate can work, and why not. Answered in Go
  // (App.AgentGate) so this page, ห้องงานวิดีโอ and the chat menu all draw the
  // same verdict on the same agent.
  let gates = $state<Record<string, main.AgentGate>>({})
  // The roster waits for the verdicts rather than drawing ahead of them. A card
  // that appears usable and is veiled a moment later has already told the
  // reader something untrue; the empty moment is shorter than the wrong one.
  let gated = $state(false)

  async function loadNeeds(roster: main.Chair[]) {
    const answers = await Promise.all(roster.map((c) => AgentGate(c.name)))
    gates = Object.fromEntries(roster.map((c, i) => [c.name, answers[i]]))
    gated = true
  }
  // The page is organised by TEAM since 12 ก.ย. (§256): every roster the
  // assistant or the code desk can hire from, the default one first, each
  // with its own members, its own switches and its own doors. Teams are
  // folders like agents are, read from disk on every visit for the same
  // reason the roster is.
  let teams = $state<main.TeamCard[]>([])
  // Each team's switches, keyed by team name ('' is the default). Absent
  // rather than fatal when a read fails: the deck is still drawn, as one
  // group with no switch, which is exactly the page as it stood before.
  let switches = $state<Record<string, main.DelegateSettings>>({})
  let delegateBusy = $state('')
  async function loadTeams() {
    try {
      teams = await ListTeams('')
    } catch {
      teams = []
    }
    const answers = await Promise.all(teams.map(async (tm) => {
      try { return await DelegateSwitches(tm.name) } catch { return null }
    }))
    const next: Record<string, main.DelegateSettings> = {}
    teams.forEach((tm, i) => { if (answers[i]) next[tm.name] = answers[i]! })
    switches = next
  }

  // One agent's reach on ONE team, looked up in that team's agents block. The
  // same agent may be in reach on one team and switched off on another —
  // the switches were split per team on purpose (config.TeamSwitches).
  function reachOf(team: main.TeamCard, name: string): { on: boolean; off: boolean } | null {
    const s = switches[team.name]
    if (!s) return null
    const w = s.agents.workers.find((x) => x.name === name)
    return w ? { on: w.on, off: s.agents.off } : null
  }
  function reaches(team: main.TeamCard, c: main.Chair): boolean {
    const w = reachOf(team, c.name)
    return !!w && w.on && !w.off
  }
  const teamLabel = (tm: main.TeamCard) => (tm.default ? t('office.teamDefault') : tm.name)
  const deskLabel = (desk: string) => (desk === 'coding' ? t('office.teamDeskCoding') : t('office.teamDeskSpecialized'))

  async function toggleTeam(team: main.TeamCard) {
    const s = switches[team.name]
    if (!s || delegateBusy) return
    delegateBusy = team.name
    try {
      switches = { ...switches, [team.name]: await SetDelegateOff(team.name, 'agents', s.agents.off === false) }
    } finally {
      delegateBusy = ''
    }
  }

  // The split each team's deck is drawn in. Only drawn as two groups when
  // there is a switch to answer with and both groups have somebody in them —
  // a heading over an empty deck is a label for nothing, and on a fresh
  // install (where delegation ships off) it would put every card under
  // "ยังไม่ได้เปิด" with an empty group above it.
  const onDuty = (team: main.TeamCard) => team.members.filter((c) => reaches(team, c))
  const offDuty = (team: main.TeamCard) => team.members.filter((c) => !reaches(team, c))
  const banded = (team: main.TeamCard) => !!switches[team.name] && onDuty(team).length > 0 && offDuty(team).length > 0

  // The team editor: name, desk, a sentence, and a tick beside every agent
  // on the roster. One shape for new and existing; the name is fixed once a
  // folder exists, because it is the folder (and what sessions key on).
  type TeamDraft = { name: string; desk: string; description: string; members: string[]; isNew: boolean }
  let teamEditing = $state<TeamDraft | null>(null)
  let teamError = $state('')
  let teamBusy = $state(false)
  function newTeam() {
    teamEditing = { name: '', desk: 'specialized', description: '', members: [], isNew: true }
    teamError = ''
  }
  function editTeam(tm: main.TeamCard) {
    teamEditing = {
      name: tm.name, desk: tm.desk, description: tm.description,
      members: tm.members.map((c) => c.name), isNew: false,
    }
    teamError = ''
  }
  function tickMember(name: string, on: boolean) {
    if (!teamEditing) return
    const rest = teamEditing.members.filter((m) => m !== name)
    teamEditing.members = on ? [...rest, name] : rest
  }
  async function saveTeam() {
    if (!teamEditing || teamBusy) return
    teamBusy = true
    teamError = ''
    try {
      await SaveTeam(teamEditing.name.trim(), teamEditing.desk, teamEditing.description, teamEditing.members)
      teamEditing = null
      await loadTeams()
    } catch (err) {
      teamError = String(err)
    } finally {
      teamBusy = false
    }
  }
  // Deleting a team is deleting a list — the people stay — so the sentence
  // on the dialog says so, and names the folder for checking.
  let pendingConfirm = $state<{ title: string; message: string; detail: string; confirmLabel: string; run: () => void } | null>(null)
  function deleteTeam(tm: main.TeamCard) {
    pendingConfirm = {
      title: t('office.confirmTeamDeleteTitle'),
      message: t('office.confirmTeamDeleteMessage'),
      detail: tm.path || tm.name,
      confirmLabel: t('office.confirmTeamDeleteAction'),
      run: async () => {
        try {
          await DeleteTeam(tm.name)
        } catch (err) {
          teamError = String(err)
        }
        await loadTeams()
      },
    }
  }
  function runPendingConfirm() {
    const req = pendingConfirm
    pendingConfirm = null
    req?.run()
  }

  onMount(async () => {
    const [roster, feed] = await Promise.all([ListChairs(), ListReceivedJobs(30), loadTeams()])
    chairs = roster
    jobs = feed
    await loadNeeds(roster)
    loaded = true
  })

  // Walking from a job to the conversation that sent it. The job row carries
  // the caller's session id, which is the only link there is — and the only one
  // there needs to be, since the file it produced went to that session's folder.
  async function openSource(job: main.ReceivedJob) {
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
    const out: { key: TKey; items: main.ReceivedJob[] }[] = []
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
  // showing anything reads as a dead click.
  async function talkTo(chair: main.Chair, team: main.TeamCard) {
    setActiveView('chat')
    // Seated by the team the card sits under, at that team's desk: the same
    // agent walks into the office from ทีมผู้ช่วย and into the workshop from a
    // coding team, and the desk decides what it holds there (§256).
    await newChairSession(chair.name, team.desk, team.name)
  }

  // The two doors into the shared profile editor (Settings holds the one
  // implementation; two copies of an editor is how they drift). The intent
  // carries kind='agent' by construction — it comes off this roster — so the
  // editor saves through the agents' door without ever reading a file to
  // decide what something is.
  //
  // 'team' is เอเจน; 'agents' is the ซับเอเจน page next to it in Settings.
  // Sending 'agents' still opened the right editor — the handler forces
  // kind='agent' — so the header read ตั้งค่าเอเจน and everything about the
  // form was correct. What it got wrong was the page *underneath*: closing the
  // editor put the user on the ซับเอเจน roster, a page they had not asked
  // for and could not have reached from here, with no way back to the team
  // they came from. A back button that lands somewhere else is worse than no
  // back button.
  function configure(c: main.Chair) {
    cockpit.settingsIntent = { section: 'team', agent: c.name }
    setActiveView('settings')
  }
  function createAgent() {
    cockpit.settingsIntent = { section: 'team', createAgent: true }
    setActiveView('settings')
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
      <!-- The hiring doors are controls on the section, not cards in the grid.
           As a card it was a 180px dashed box holding the first slot, so the
           first thing the eye landed on was the space where nobody is — and it
           pushed a real teammate onto a row of their own. -->
      <div class="sec-head">
        <div class="eyebrow section-label">{t('office.roster')}</div>
        <span class="ag-reach"></span>
        <button class="ctrl" onclick={newTeam}><Icon name="users" size={13} /> {t('office.newTeam')}</button>
        <button class="ctrl" onclick={createAgent}><Icon name="plus" size={13} /> {t('office.newAgent')}</button>
      </div>

      <!-- The team editor, above the decks it changes. One form for a new
           team and an existing one; the name locks once the folder exists. -->
      {#if teamEditing}
        <div class="settings-card team-editor">
          <div class="team-form">
            <label class="team-field">
              <span class="eyebrow">{t('office.teamName')}</span>
              <input class="ctrl" type="text" bind:value={teamEditing.name} disabled={!teamEditing.isNew}
                placeholder={t('office.teamNamePlaceholder')} spellcheck="false" />
              {#if teamEditing.isNew}<span class="d muted">{t('office.teamNameHint')}</span>{/if}
            </label>
            <div class="team-field">
              <span class="eyebrow">{t('office.teamDesk')}</span>
              <div class="team-desks">
                {#each ['specialized', 'coding'] as desk (desk)}
                  <button type="button" class="pill" class:on={teamEditing.desk === desk}
                    onclick={() => { if (teamEditing) teamEditing.desk = desk }}>{deskLabel(desk)}</button>
                {/each}
              </div>
              {#if teamEditing.desk === 'coding'}<span class="d muted">{t('office.teamDeskCodingNote')}</span>{/if}
            </div>
            <label class="team-field">
              <span class="eyebrow">{t('office.teamDescription')}</span>
              <input class="ctrl" type="text" bind:value={teamEditing.description} />
            </label>
            <div class="team-field">
              <span class="eyebrow">{t('office.teamPick')}</span>
              <div class="team-pick">
                {#each chairs as c (c.name)}
                  {@const on = teamEditing.members.includes(c.name)}
                  <label class="team-tick" class:on>
                    <input type="checkbox" checked={on} onchange={(e) => tickMember(c.name, (e.currentTarget as HTMLInputElement).checked)} />
                    <AgentMascot name={c.name} {...lookOf(c)} size={22} />
                    <span class="t">{c.name}</span>
                  </label>
                {/each}
              </div>
              <span class="d muted">{t('office.teamPickHint')}</span>
            </div>
            {#if teamError}<div class="folder-error">{teamError}</div>{/if}
            <div class="team-actions">
              <button class="ctrl" onclick={() => (teamEditing = null)} disabled={teamBusy}>{t('office.teamCancel')}</button>
              <button class="ctrl ctrl-primary" onclick={saveTeam} disabled={teamBusy || !teamEditing.name.trim()}>{t('office.teamSave')}</button>
            </div>
          </div>
        </div>
      {/if}

      <!-- A face, not an inventory. The tool chips were six per card and five
           of the six were the same on every card — the office ceiling hands
           everyone the same set, so the list said nothing about who anyone is
           while taking half the card to say it. What is left is what the card
           is for: who this is, what they make, and whether they have done any
           of it. The tools are one click away behind the gear, which is also
           the only place they can be changed. -->
      <!-- One card, drawn once per band of every team. A snippet rather than a
           copy because the decks differ in nothing except which agents are in
           them, and a second copy is a second thing to keep true. -->
      {#snippet chairCard(c: main.Chair, team: main.TeamCard)}
          {@const locked = gates[c.name]?.blocked ?? false}
          <!-- No switch, and the card never cools (owner, 31 ส.ค.): *"มันเหมือน
               ไม่เปิดใช้งาน ทั้งที่มันก็แชทได้ปกติ"*.
               The switch and the drained card were both answering "may the MAIN
               assistant hand this one work", and both were read as "this agent
               is off" — a card that greys beside an off switch says disabled in
               two ways at once, and the face made it worse: a person with the
               colour pulled out of them reads as gone, not as undelegated.
               The band this card sits under already carries that answer, with a
               count and a line saying the chat still opens, so the state is on
               screen once instead of three times. Changing it is the gear,
               which is where the rest of this agent's settings already live. -->
          <div class="chair-card agc" class:locked>
            <div class="chair-body">
              <div class="chair-who">
                <AgentMascot name={c.name} {...lookOf(c)} size={38} />
                <span class="chair-name">{c.name}</span>
              </div>
              <p class="chair-desc">{c.description}</p>
              <!-- What this agent has actually done, as a quiet line inside the
                   card rather than a column of the foot (owner, 30 ส.ค.). Only
                   facts that DIFFER between agents are drawn here — a badge
                   every card carries is a badge that says nothing. -->
              <div class="chair-chips">
                {#if c.overrides}<span class="chip mine">{t('office.overrides')}</span>{/if}
                {#if c.jobs > 0}
                  <span class="chair-stat"><span class="n">{c.jobs}</span> {t('office.jobsDone')} · {agoLabel(c.lastUsed ?? '')}</span>
                {:else}
                  <span class="chair-stat idle">{t('office.neverUsed')}</span>
                {/if}
              </div>
            </div>
            <!-- The one thing this page is for: walking in and talking to a
                 specialist. Named with the agent, not "this agent", because
                 that is what walking in is — and the row cannot overflow. -->
            <div class="chair-foot">
              <button class="chair-talk" onclick={() => talkTo(c, team)}>
                <Icon name="messageSquare" size={14} />
                <span class="t">{t('office.chatWith', { name: c.name })}</span>
              </button>
              <button class="icobtn tiny tip-l" aria-label={t('settings.agentConfigure')}
                data-tip={t('settings.agentConfigure')} onclick={() => configure(c)}>
                <Icon name="settings" size={13} />
              </button>
            </div>
            <AgentLock agent={c.name} label={c.name} gate={gates[c.name] ?? null}
              onInstalled={() => loadNeeds(chairs)} />
          </div>
      {/snippet}

      <!-- One section per team (§256). Its head carries the facts about the
           roster — where it works, how many, whether the assistant may hand
           it work — and its deck is split by the one thing this page can
           change: which band an agent sits in IS its delegation state on
           THIS team, so no card needs a badge for it. -->
      {#each teams as tm (tm.name)}
        {@const s = switches[tm.name]}
        <section class="team-sec" class:invalid={!!tm.invalid}>
          <div class="team-head">
            <span class="team-ic"><Icon name="users" size={15} /></span>
            <div class="team-title">
              <span class="team-name">{teamLabel(tm)}</span>
              <span class="chip">{deskLabel(tm.desk)}</span>
              <span class="team-count">{t('office.teamMembersCount', { n: tm.members.length })}</span>
            </div>
            {#if tm.description}<span class="team-desc">{tm.description}</span>{/if}
            <span class="rule"></span>
            {#if s && !tm.invalid}
              <span class="ag-reach">
                {#if s.agents.off}
                  {t('office.reachNone')}
                {:else}
                  {t('office.reachSome', { n: onDuty(tm).length, total: tm.members.length })}
                {/if}
              </span>
              <label class="mswitch" title={t('office.teamDelegate')}>
                <input
                  type="checkbox" checked={!s.agents.off} disabled={delegateBusy !== ''}
                  aria-label={t('office.teamDelegate')} onchange={() => toggleTeam(tm)}
                />
                <span></span>
              </label>
            {/if}
            {#if !tm.default}
              <button class="icobtn tiny tip-l" aria-label={t('office.teamEdit')} data-tip={t('office.teamEdit')}
                onclick={() => editTeam(tm)}><Icon name="settings" size={13} /></button>
              <button class="icobtn tiny tip-l" aria-label={t('office.teamDelete')} data-tip={t('office.teamDelete')}
                onclick={() => deleteTeam(tm)}><Icon name="trash" size={13} /></button>
            {/if}
          </div>
          {#if tm.default}
            <p class="team-note">{t('office.teamDefaultNote')}</p>
          {/if}
          {#if tm.invalid}
            <p class="team-note bad">{t('office.teamInvalid', { reason: tm.invalid })}</p>
          {/if}
          {#if tm.missing.length > 0}
            <p class="team-note bad">{t('office.teamMissing', { names: tm.missing.join(', ') })}</p>
          {/if}
          {#if banded(tm)}
            <div class="ag-band">
              <span class="lab">{t('office.bandOn')}</span><span class="n">{onDuty(tm).length}</span>
              <span class="rule"></span>
            </div>
            <div class="office-grid">
              {#each gated ? onDuty(tm) : [] as c (c.name)}{@render chairCard(c, tm)}{/each}
            </div>
            <div class="ag-band">
              <span class="lab">{t('office.bandOff')}</span><span class="n">{offDuty(tm).length}</span>
              <span class="rule"></span>
              <span class="say">{t('office.bandOffNote')}</span>
            </div>
            <div class="office-grid">
              {#each gated ? offDuty(tm) : [] as c (c.name)}{@render chairCard(c, tm)}{/each}
            </div>
          {:else}
            <div class="office-grid">
              {#each gated ? tm.members : [] as c (c.name)}{@render chairCard(c, tm)}{/each}
              {#if loaded && tm.members.length === 0}
                <div class="chair-card empty"><div class="chair-body"><p class="chair-desc">{tm.default ? t('office.noChairs') : t('office.teamEmpty')}</p></div></div>
              {/if}
            </div>
          {/if}
        </section>
      {/each}
      <p class="office-note">
        {t('office.hiringNote')}
        <button class="linklike" onclick={() => OpenAgentsFolder()}>{t('office.openAgentsFolder')}</button>
        · {t('office.teamsNote')}
        <button class="linklike" onclick={() => OpenTeamsFolder()}>{t('office.openTeamsFolder')}</button>
      </p>

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

      <!-- Where the rest of them are. This page is the roster — who takes work
           and what they have done — and it is not every profile the engine
           runs: the assistant's own delegates never sit here. Saying so is what
           keeps two pages from reading as one list that disagrees with itself.
           It sits at the foot because it is a footnote: mid-page it was a wall
           of prose between the team and their work. -->
      <p class="office-note foot">{t('office.settingsNote')}</p>
    </div>
  </div>
</div>

{#if pendingConfirm}
  {@const req = pendingConfirm}
  <ConfirmDialog
    title={req.title}
    message={req.message}
    detail={req.detail}
    confirmLabel={req.confirmLabel}
    onConfirm={runPendingConfirm}
    onCancel={() => (pendingConfirm = null)}
  />
{/if}
