<script lang="ts">
  // Settings › ทีมเอเจน (DECISIONS §256): the one home of a roster — where a
  // team is made, named to a desk, given its members, and where both of its
  // switches live. The people are on the roster page; this page never draws
  // a chat door (owner, 12 ก.ย.: "คนอยู่หน้าแรก ทีมอยู่ตั้งค่า").
  //
  // The shape is การตั้งค่าโมเดล's, because it is the same kind of page: a
  // short list of things on the left, the one you picked on the right
  // (.mset / .mset-side / .mset-detail). A single tall card of every team and
  // every member was the first cut, and the owner sent it back ("เอามาอยู่
  // โดดๆแบบนี้ได้ไง") — a list with switches down one edge said nothing about
  // what a team IS. Here the rail is the teams, and the pane is one roster at
  // a time: who it hires, at which desk, whether the assistant may hand it
  // work, and — for a team of the user's — the form that changes it.
  //
  // Making a team has to look possible at a glance (owner: "ดูแล้วเพิ่มง่าย
  // เห็นแล้วรู้ว่าอ๋อ เพิ่มได้"), so the door is drawn three times on purpose:
  // the primary button beside the title, the last row of the rail, and — on a
  // machine with no team of the user's yet — a callout under the shipped team
  // saying what a team is for and offering to make the first one.
  //
  // Its own file rather than a snippet in Settings.svelte for the reason
  // AvatarSettings is: a team is not a profile, and that file already holds
  // the shape of every other page. The roster page and the chat's picker send
  // their doors here through settingsIntent — one editor, several doors.
  import { onMount } from 'svelte'
  import {
    ListChairs, ListTeams, SaveTeam, DeleteTeam, OpenTeamsFolder,
    DelegateSwitches, SetAgentOff,
  } from '../../wailsjs/go/main/App'
  import { main } from '../../wailsjs/go/models'
  import { cockpit, setActiveView } from './stores/cockpit.svelte'
  import { setShell } from './shell.svelte'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import { lookOf } from './mascot/agentLook'

  // The one door out of this page: to the agent editor, for an agent that
  // does not exist yet. Settings owns that editor, so it hands the door in.
  let { onNewAgent }: { onNewAgent?: () => void } = $props()

  let teams = $state<main.TeamCard[]>([])
  // The whole roster, for the editor's tick list: any agent may be on any
  // team, whatever team it is on today.
  let chairs = $state<main.Chair[]>([])
  // Each team's switches, keyed by team name ('' is the default). Absent
  // rather than fatal when a read fails: the pane is still drawn, without the
  // switch it cannot honour.
  let switches = $state<Record<string, main.DelegateSettings>>({})
  let loaded = $state(false)
  let error = $state('')
  let busy = $state('')

  // Which team the pane shows. '' is ทีมผู้ช่วย, which is also where the page
  // opens: it is the team every machine has.
  let selected = $state('')
  const current = $derived(teams.find((x) => x.name === selected) ?? null)
  // Which SIDE the page shows — the assistant's teams or the code door's —
  // chosen by the two-way switch at the top (owner, 13 ก.ย.: "เป็นแบบฝั่ง
  // ผู้ช่วยและฝั่งโค้ด แยกกัน" — a switch between the two, not an on/off).
  // One side at a time: the rail lists that side's teams and its door, the
  // pane the team picked there. Nothing else on this page is an on/off but
  // a member's own reach on its team.
  let side = $state<'specialized' | 'coding'>('specialized')
  const onSide = $derived(teams.filter((tm) => tm.desk === side))
  function showSide(desk: string) {
    side = desk === 'coding' ? 'coding' : 'specialized'
    editing = null
    const first = teams.find((tm) => tm.desk === side)
    selected = first ? first.name : ''
  }
  // The door's own on/off is not on this page — it is the row in the chat's
  // team menu — but a door that is off cools the rows under it here, so the
  // page never shows a switch that does nothing. Read off any team's answer;
  // the engine reports both doors on every one.
  const sideOff = $derived.by(() => {
    const any = Object.values(switches)[0]
    return { specialized: any ? any.agents.off : false, coding: any ? any.code.off : false }
  })

  async function load() {
    try {
      const [roster, list] = await Promise.all([ListChairs(), ListTeams('')])
      chairs = roster
      teams = list
    } catch (err) {
      error = String(err)
      teams = []
    }
    const answers = await Promise.all(teams.map(async (tm) => {
      try { return await DelegateSwitches(tm.name) } catch { return null }
    }))
    const next: Record<string, main.DelegateSettings> = {}
    teams.forEach((tm, i) => { if (answers[i]) next[tm.name] = answers[i]! })
    switches = next
    if (!teams.some((x) => x.name === selected && x.desk === side)) selected = teams.find((x) => x.desk === side)?.name ?? ''
    loaded = true
  }

  const teamLabel = (tm: main.TeamCard) => tm.name
  // The two sides a team can be on, in the order the doors sit on the
  // wordmark: the storefront first, the workshop second. Icon and colour are
  // the desk's own (desks.ts), so a team reads as belonging to a door the
  // user already knows.
  const SIDES = [
    { desk: 'specialized', cls: 'side-assistant', icon: 'sparkles' as const, label: 'settings.teamSideAssistant' as const, note: 'settings.teamDeskAssistantNote' as const },
    { desk: 'coding', cls: 'side-code', icon: 'fileCode' as const, label: 'settings.teamSideCode' as const, note: 'office.teamDeskCodingNote' as const },
  ]
  const sideOf = (desk: string) => SIDES.find((s) => s.desk === desk) ?? SIDES[0]
  const curSide = $derived(sideOf(side))
  // How many of a team are in the assistant's reach right now — the number
  // the rail shows beside each team, so the state is readable before a click.
  function inReach(tm: main.TeamCard): number {
    const s = switches[tm.name]
    if (!s || sideOff[tm.desk as 'specialized' | 'coding']) return 0
    return tm.members.filter((c) => s.agents.workers.find((w) => w.name === c.name)?.on).length
  }

  // One member's reach on ONE team. The same agent may be in reach on one
  // team and switched off on another (config.TeamSwitches) — which is the
  // whole reason the switch sits on the member row inside the team's pane.
  function reachOf(team: main.TeamCard, name: string): { on: boolean; off: boolean } | null {
    const s = switches[team.name]
    if (!s) return null
    const w = s.agents.workers.find((x) => x.name === name)
    return w ? { on: w.on, off: sideOff[team.desk as 'specialized' | 'coding'] } : null
  }
  async function toggleMember(team: main.TeamCard, name: string, on: boolean) {
    if (busy) return
    busy = 'member:' + team.name + ':' + name
    try {
      switches = { ...switches, [team.name]: await SetAgentOff(team.name, name, on) }
    } finally {
      busy = ''
    }
  }

  // The editor, drawn in the pane in place of the team: name, desk, a
  // sentence, and a tick beside every agent on the roster. One shape for new
  // and existing; the name is fixed once a folder exists, because it is the
  // folder (and what sessions key on).
  type TeamDraft = { name: string; desk: string; description: string; members: string[]; isNew: boolean; path: string }
  let editing = $state<TeamDraft | null>(null)
  let editError = $state('')
  // "+ เอเจน" in the tick list: the agent wanted is not on the roster yet.
  // The draft is parked (this page unmounts while the agent editor is up)
  // and picked back up on return, with the new agent ticked — so making an
  // agent mid-form costs nothing typed so far.
  const DRAFT_KEY = 'aetox.teamDraft'
  function goNewAgent() {
    if (!editing || !onNewAgent) return
    try { sessionStorage.setItem(DRAFT_KEY, JSON.stringify(editing)) } catch { /* the draft is just not kept */ }
    onNewAgent()
  }
  function takeParkedDraft(): TeamDraft | null {
    try {
      const raw = sessionStorage.getItem(DRAFT_KEY)
      sessionStorage.removeItem(DRAFT_KEY)
      return raw ? (JSON.parse(raw) as TeamDraft) : null
    } catch {
      return null
    }
  }
  function newTeam(desk: string = side) {
    editing = { name: '', desk, description: '', members: [], isNew: true, path: '' }
    editError = ''
  }
  function editTeam(tm: main.TeamCard) {
    editing = {
      name: tm.name, desk: tm.desk, description: tm.description,
      members: tm.members.map((c) => c.name), isNew: false, path: tm.path ?? '',
    }
    editError = ''
  }
  function pick(name: string) {
    editing = null
    selected = name
  }
  function tickMember(name: string, on: boolean) {
    if (!editing) return
    const rest = editing.members.filter((m) => m !== name)
    editing.members = on ? [...rest, name] : rest
  }
  async function saveTeam() {
    if (!editing || busy) return
    busy = 'save'
    editError = ''
    const name = editing.name.trim()
    try {
      await SaveTeam(name, editing.desk, editing.description, editing.members)
      const desk = editing.desk
      editing = null
      showSide(desk)
      await load()
      selected = name
    } catch (err) {
      editError = String(err)
    } finally {
      busy = ''
    }
  }

  // Deleting a team is deleting a list — the people stay — so the sentence
  // on the dialog says so, and names the folder for checking.
  let pendingConfirm = $state<{ title: string; message: string; detail: string; confirmLabel: string; run: () => void } | null>(null)
  function deleteTeam(name: string, path: string) {
    pendingConfirm = {
      title: t('office.confirmTeamDeleteTitle'),
      message: t('office.confirmTeamDeleteMessage'),
      detail: path || name,
      confirmLabel: t('office.confirmTeamDeleteAction'),
      run: async () => {
        try {
          await DeleteTeam(name)
          editing = null
          selected = ''
        } catch (err) {
          error = String(err)
        }
        await load()
      },
    }
  }
  function runPendingConfirm() {
    const req = pendingConfirm
    pendingConfirm = null
    req?.run()
  }

  // The roster page's and the picker's doors land here on the right team —
  // or on a blank form. Consumed once and cleared, like the agent editor's
  // intent: one that survived into the next plain visit would reopen a form
  // nobody asked for.
  onMount(async () => {
    const intent = cockpit.settingsIntent
    if (intent && intent.section === 'teams') cockpit.settingsIntent = null
    await load()
    const parked = takeParkedDraft()
    if (parked) {
      showSide(parked.desk)
      editing = parked
      if (intent?.agent && chairs.some((c) => c.name === intent.agent)) tickMember(intent.agent, true)
      return
    }
    if (!intent || intent.section !== 'teams') return
    if (intent.createTeam) newTeam(side)
    else if (intent.team && teams.some((x) => x.name === intent.team)) {
      const tm = teams.find((x) => x.name === intent.team)
      if (tm) {
        showSide(tm.desk)
        selected = tm.name
        editTeam(tm)
      }
    }
  })
</script>

<div class="team-title">
  <div>
    <h2>{t('settings.teams')}</h2>
    <p class="muted set-sub">
      {t('settings.teamsDesc')}
      <button class="linklike" onclick={() => { setShell('assistant'); setActiveView('office') }}>{t('settings.teamOpenPage')} <Icon name="arrowRight" size={12} /></button>
    </p>
  </div>
  <!-- The door, where the eye lands first — on the side being shown. -->
  <button class="ctrl ctrl-primary team-new" onclick={() => newTeam(side)} disabled={!!editing?.isNew}><Icon name="plus" size={14} /> {t('office.newTeam')}</button>
</div>
{#if error}<div class="mset-error">{error}</div>{/if}

<!-- The switch between the two sides, above everything — the approval
     row's own two-way control, wearing each desk's icon and colour. It picks
     what the page SHOWS; the door's on/off lives in the chat's team menu. -->
<div class="seg-ctrl team-side-pick" role="tablist" aria-label={t('settings.teamSides')}>
  {#each SIDES as s (s.desk)}
    <button type="button" class="seg-btn {s.cls}" class:selected={side === s.desk}
      role="tab" aria-selected={side === s.desk} onclick={() => showSide(s.desk)}>
      <Icon name={s.icon} size={13} /> {t(s.label)}
      <span class="team-tally">{teams.filter((tm) => tm.desk === s.desk).length}</span>
    </button>
  {/each}
</div>
{#if sideOff[side]}
  <p class="muted set-hint team-door-off">{t(side === 'coding' ? 'settings.sideCodeOff' : 'settings.sideAssistantOff')} {t('settings.sideOffWhere')}</p>
{/if}

<div class="settings-card mset team-set">
  <!-- The rail: every team, the default first, each with the one number
       worth reading before a click — how many of it the assistant may hand
       work to — and the door once more at the foot. -->
  <aside class="mset-side">
    <!-- One side's teams, each with the one number worth reading before a
         click — how many of it the door may hand work to — and the side's
         door at the foot, so an empty side still reads as a place a team can
         be made (owner: "ดูแล้วเพิ่มง่าย เห็นแล้วรู้ว่าอ๋อ เพิ่มได้"). -->
    <div class="settings-group-label eyebrow team-side {curSide.cls}">
      <Icon name={curSide.icon} size={12} /> {t(curSide.label)}
    </div>
    {#each onSide as tm (tm.name)}
      <button class="mset-prov team-row {curSide.cls}" class:selected={selected === tm.name && !editing?.isNew}
        class:invalid={!!tm.invalid} onclick={() => pick(tm.name)}>
        <Icon name="users" size={15} />
        <span class="mset-prov-name">{teamLabel(tm)}</span>
        <span class="team-tally" title={t('settings.teamTallyTip')}>{inReach(tm)}/{tm.members.length}</span>
      </button>
    {/each}
    {#if loaded && onSide.length === 0}
      <div class="team-side-empty">{t('settings.teamSideEmpty')}</div>
    {/if}
    <button class="mset-prov team-add {curSide.cls}" class:selected={!!editing?.isNew}
      onclick={() => newTeam(side)}>
      <Icon name="plus" size={14} /> {t('settings.teamNewHere')}
    </button>
  </aside>

  <div class="mset-detail">
    {#if editing}
      <!-- The form. The same fields the model page's custom endpoint draws
           (mset-field + eyebrow + hint + .ctrl), the same two-way choice the
           approval row draws (seg-ctrl), and the audience chips the MCP page
           ticks agents with (conn-chip). Nothing of this page's own: a form
           that looks like no other form is a form somebody has to learn. -->
      <div class="mset-head">
        <Icon name="users" size={22} />
        <span class="mset-name">{editing.isNew ? t('office.newTeam') : t('office.teamEdit')}</span>
        <!-- The side is a fact from the switch above, not a choice down here
             (owner: "เราเลือกข้างบนอยู่แล้ว จะมีปุ่มนี้ทำไม"). -->
        <span class="desk-badge {sideOf(editing.desk).cls}"><Icon name={sideOf(editing.desk).icon} size={12} /> {t(sideOf(editing.desk).label)}</span>
      </div>
      <p class="muted set-hint">{t('settings.teamEditDesc')} {t(sideOf(editing.desk).note)}</p>
      <div class="mset-field">
        <div class="eyebrow">{t('office.teamName')}</div>
        {#if editing.isNew}<div class="muted set-hint">{t('office.teamNameHint')}</div>{/if}
        <!-- svelte-ignore a11y_autofocus -->
        <input class="ctrl key-input" type="text" bind:value={editing.name} disabled={!editing.isNew}
          placeholder={t('office.teamNamePlaceholder')} spellcheck="false" autofocus={editing.isNew} />
      </div>
      <div class="mset-field">
        <div class="eyebrow">{t('office.teamDescription')}</div>
        <input class="ctrl key-input" type="text" bind:value={editing.description} placeholder={t('settings.teamDescriptionPlaceholder')} />
      </div>
      <div class="mset-field">
        <div class="eyebrow">{t('office.teamPick')} <span class="team-picked">{editing.members.length}</span></div>
        <div class="muted set-hint">{t('office.teamPickHint')}</div>
        <div class="conn-targets">
          {#each chairs as c (c.name)}
            {@const on = editing.members.includes(c.name)}
            <button type="button" class="conn-chip agent" class:on aria-pressed={on}
              title={c.description} onclick={() => tickMember(c.name, !on)}>
              <AgentMascot name={c.name} {...lookOf(c)} size={16} />
              {c.name}
            </button>
          {/each}
          {#if onNewAgent}
            <button type="button" class="conn-chip team-add-agent" title={t('settings.teamAddAgentTip')} onclick={goNewAgent}>
              <Icon name="plus" size={13} /> {t('settings.teamAddAgent')}
            </button>
          {/if}
        </div>
      </div>
      {#if editError}<div class="mset-error">{editError}</div>{/if}
      <div class="mset-keyrow team-actions">
        <button class="ctrl ctrl-primary" onclick={saveTeam} disabled={busy !== '' || !editing.name.trim()}>{t('office.teamSave')}</button>
        <button class="ctrl" onclick={() => { editing = null }} disabled={busy !== ''}>{t('office.teamCancel')}</button>
        <div class="pp-bar-gap"></div>
        {#if !editing.isNew}
          <button class="ctrl ctrl-danger" disabled={busy !== ''} onclick={() => editing && deleteTeam(editing.name, editing.path)}>{t('office.teamDelete')}</button>
        {/if}
      </div>

    {:else if current}
      {@const tm = current}
      {@const s = switches[tm.name]}
      <!-- One roster. Its head is who it is; the card under it is the one
           decision about it (may the assistant hand it work); the rows are
           the people, each with that decision for them alone. -->
      <div class="mset-head">
        <Icon name="users" size={22} />
        <span class="mset-name">{teamLabel(tm)}</span>
        <!-- The side, as a badge with the desk's icon and colour — the one
             fact about a team that decides what its members hold. -->
        <span class="desk-badge {sideOf(tm.desk).cls}"><Icon name={sideOf(tm.desk).icon} size={12} /> {t(sideOf(tm.desk).label)}</span>
        <div class="pp-bar-gap"></div>
        <button class="ctrl" onclick={() => editTeam(tm)}><Icon name="settings" size={13} /> {t('office.teamEdit')}</button>
      </div>
      <p class="muted set-hint">
        {#if tm.description}{tm.description}{:else}{t('settings.teamNoDescription')}{/if}
      </p>
      {#if tm.invalid}<div class="mset-error">{t('office.teamInvalid', { reason: tm.invalid })}</div>{/if}
      {#if tm.missing.length > 0}<div class="mset-error">{t('office.teamMissing', { names: tm.missing.join(', ') })}</div>{/if}

      <div class="mset-field">
        <div class="eyebrow">
          {t('office.teamPick')} <span class="team-picked">{tm.members.length}</span>
          {#if s && !sideOff[tm.desk as 'specialized' | 'coding']}<span class="team-picked muted">· {t('settings.teamInReach', { n: inReach(tm) })}</span>{/if}
        </div>
        {#if tm.members.length === 0}
          <p class="muted set-hint">{t('office.teamEmpty')}</p>
        {/if}
        <div class="team-members" class:cool={sideOff[tm.desk as 'specialized' | 'coding']}>
          {#each tm.members as c (c.name)}
            {@const reach = reachOf(tm, c.name)}
            <div class="team-member" class:off={!!reach && !(reach.on && !reach.off)}>
              <AgentMascot name={c.name} {...lookOf(c)} size={28} />
              <div class="team-member-txt">
                <div class="t">{c.name}</div>
                {#if c.description}<div class="d">{c.description}</div>{/if}
              </div>
              {#if reach}
                <label class="mswitch" title={t('settings.agentReachTip')}>
                  <input type="checkbox" checked={reach.on && !reach.off} disabled={reach.off || busy !== ''}
                    aria-label={t('settings.agentReach')} onchange={() => toggleMember(tm, c.name, reach.on)} />
                  <span></span>
                </label>
              {/if}
            </div>
          {/each}
        </div>
      </div>

      <!-- The nudge, only while the user has no team of their own: what a
           team is for, in one sentence each, and the door. Gone the moment
           the first team exists — a tip for a thing already done is noise. -->
      {#if loaded && onSide.length <= 1}
        <div class="team-callout">
          <div class="team-callout-txt">
            <div class="t">{t('settings.teamFirstTitle')}</div>
            <div class="d">{t('settings.teamFirstBody')}</div>
          </div>
          <button class="ctrl ctrl-primary" onclick={() => newTeam(side)}><Icon name="plus" size={14} /> {t('office.newTeam')}</button>
        </div>
      {/if}
    {:else if loaded}
      <p class="muted set-hint">{t('office.teamEmpty')}</p>
    {/if}
  </div>
</div>
<p class="muted set-sub">
  {t('office.teamsNote')}
  <button class="linklike" onclick={() => OpenTeamsFolder()}>{t('settings.teamsFolder')}</button>
</p>

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
