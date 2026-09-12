<script lang="ts">
  // Settings › ทีม (DECISIONS §256): where a team is made, named to a desk,
  // given its members, and where its two switches live. Its own page beside
  // เอเจน, never a field on the agent editor — owner, 12 ก.ย.: "ตั้งค่าทีมแยก
  // กับเอเจน". An agent's editor says who the agent is; which lists name it is
  // the team's business and is edited here.
  //
  // Its own file rather than a snippet in Settings.svelte for the reason
  // AvatarSettings is: that file is the shape of every other page, and a team
  // is not a profile — a card here is a roster, not a person.
  //
  // The roster page (Office.svelte) draws the same teams as the rooms you walk
  // into and sends its gear and its "สร้างทีม" here through settingsIntent —
  // one editor, two doors, the same rule the agent editor lives by.
  import { onMount } from 'svelte'
  import {
    ListChairs, ListTeams, SaveTeam, DeleteTeam, OpenTeamsFolder,
    DelegateSwitches, SetDelegateOff, SetAgentOff,
  } from '../../wailsjs/go/main/App'
  import { main } from '../../wailsjs/go/models'
  import { cockpit, setActiveView } from './stores/cockpit.svelte'
  import { setShell } from './shell.svelte'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import { lookOf } from './mascot/agentLook'

  let teams = $state<main.TeamCard[]>([])
  // The whole roster, for the editor's tick list: every agent may be on any
  // team, whatever team it is on today.
  let chairs = $state<main.Chair[]>([])
  // Each team's switches, keyed by team name ('' is the default). Absent
  // rather than fatal when a read fails: the list is still drawn, without the
  // switch it cannot honour.
  let switches = $state<Record<string, main.DelegateSettings>>({})
  let loaded = $state(false)
  let error = $state('')
  let busy = $state('')

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
    loaded = true
  }

  const teamLabel = (tm: main.TeamCard) => (tm.default ? t('office.teamDefault') : tm.name)
  const deskLabel = (desk: string) => (desk === 'coding' ? t('office.teamDeskCoding') : t('office.teamDeskSpecialized'))

  // One member's reach on ONE team. The same agent may be in reach on one
  // team and switched off on another (config.TeamSwitches) — which is the
  // whole reason the switch sits on the member row inside the team card.
  function reachOf(team: main.TeamCard, name: string): { on: boolean; off: boolean } | null {
    const s = switches[team.name]
    if (!s) return null
    const w = s.agents.workers.find((x) => x.name === name)
    return w ? { on: w.on, off: s.agents.off } : null
  }
  async function toggleTeam(team: main.TeamCard) {
    const s = switches[team.name]
    if (!s || busy) return
    busy = 'team:' + team.name
    try {
      switches = { ...switches, [team.name]: await SetDelegateOff(team.name, 'agents', s.agents.off === false) }
    } finally {
      busy = ''
    }
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

  // The editor: name, desk, a sentence, and a tick beside every agent on the
  // roster. One shape for new and existing; the name is fixed once a folder
  // exists, because it is the folder (and what sessions key on).
  type TeamDraft = { name: string; desk: string; description: string; members: string[]; isNew: boolean; path: string }
  let editing = $state<TeamDraft | null>(null)
  let editError = $state('')
  function newTeam() {
    editing = { name: '', desk: 'specialized', description: '', members: [], isNew: true, path: '' }
    editError = ''
  }
  function editTeam(tm: main.TeamCard) {
    editing = {
      name: tm.name, desk: tm.desk, description: tm.description,
      members: tm.members.map((c) => c.name), isNew: false, path: tm.path ?? '',
    }
    editError = ''
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
    try {
      await SaveTeam(editing.name.trim(), editing.desk, editing.description, editing.members)
      editing = null
      await load()
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

  // The roster page's doors land here with the editor already open on the
  // right team — or on a blank one. Consumed once and cleared, like the agent
  // editor's intent: one that survived into the next plain visit would reopen
  // an editor nobody asked for.
  onMount(async () => {
    const intent = cockpit.settingsIntent
    if (intent && intent.section === 'teams') cockpit.settingsIntent = null
    await load()
    if (!intent || intent.section !== 'teams') return
    if (intent.createTeam) newTeam()
    else if (intent.team) {
      const tm = teams.find((x) => x.name === intent.team)
      if (tm) editTeam(tm)
    }
  })
</script>

{#if editing}
  <h2>{editing.isNew ? t('office.newTeam') : t('office.teamEdit')}</h2>
  <p class="muted set-sub">{t('settings.teamEditDesc')}</p>
  <div class="pp-bar">
    <button class="ctrl" onclick={() => (editing = null)} disabled={busy !== ''}><Icon name="arrowLeft" size={14} /> {t('settings.agentBack')}</button>
    <div class="pp-bar-gap"></div>
    {#if !editing.isNew}
      <button class="ctrl ctrl-danger" disabled={busy !== ''} onclick={() => editing && deleteTeam(editing.name, editing.path)}>{t('office.teamDelete')}</button>
    {/if}
    <button class="ctrl ctrl-primary" onclick={saveTeam} disabled={busy !== '' || !editing.name.trim()}>{t('office.teamSave')}</button>
  </div>
  <div class="settings-card team-editor">
    <div class="team-form">
      <label class="team-field">
        <span class="eyebrow">{t('office.teamName')}</span>
        <input class="ctrl" type="text" bind:value={editing.name} disabled={!editing.isNew}
          placeholder={t('office.teamNamePlaceholder')} spellcheck="false" />
        {#if editing.isNew}<span class="d muted">{t('office.teamNameHint')}</span>{/if}
      </label>
      <div class="team-field">
        <span class="eyebrow">{t('office.teamDesk')}</span>
        <div class="team-desks">
          {#each ['specialized', 'coding'] as desk (desk)}
            <button type="button" class="pill" class:on={editing.desk === desk}
              onclick={() => { if (editing) editing.desk = desk }}>{deskLabel(desk)}</button>
          {/each}
        </div>
        {#if editing.desk === 'coding'}<span class="d muted">{t('office.teamDeskCodingNote')}</span>{/if}
      </div>
      <label class="team-field">
        <span class="eyebrow">{t('office.teamDescription')}</span>
        <input class="ctrl" type="text" bind:value={editing.description} />
      </label>
      <div class="team-field">
        <span class="eyebrow">{t('office.teamPick')}</span>
        <div class="team-pick">
          {#each chairs as c (c.name)}
            {@const on = editing.members.includes(c.name)}
            <label class="team-tick" class:on>
              <input type="checkbox" checked={on} onchange={(e) => tickMember(c.name, (e.currentTarget as HTMLInputElement).checked)} />
              <AgentMascot name={c.name} {...lookOf(c)} size={22} />
              <span class="t">{c.name}</span>
            </label>
          {/each}
        </div>
        <span class="d muted">{t('office.teamPickHint')}</span>
      </div>
      {#if editError}<div class="mset-error">{editError}</div>{/if}
    </div>
  </div>
{:else}
  <h2>{t('settings.teams')}</h2>
  <p class="muted set-sub">{t('settings.teamsDesc')}</p>
  <div class="pp-bar">
    <button class="ctrl" onclick={newTeam}><Icon name="plus" size={13} /> {t('office.newTeam')}</button>
    <button class="ctrl" onclick={() => load()}>{t('settings.refresh')}</button>
    <button class="ctrl" onclick={() => OpenTeamsFolder()}>{t('settings.teamsFolder')}</button>
    <div class="pp-bar-gap"></div>
    <!-- The rooms are on the roster page behind the storefront; this page
         configures. The door goes with the room (§86). -->
    <button class="ctrl" onclick={() => { setShell('assistant'); setActiveView('office') }}>{t('settings.teamOpenPage')} <Icon name="arrowRight" size={13} /></button>
  </div>
  {#if error}<div class="mset-error">{error}</div>{/if}

  <!-- One card per team: its head (name, desk, sentence, the team's own
       switch, the gear and the bin for a user team), then its members as rows
       with each one's reach ON THIS TEAM. -->
  {#each teams as tm (tm.name)}
    {@const s = switches[tm.name]}
    <div class="settings-card team-card" class:invalid={!!tm.invalid}>
      <div class="set-row team-card-head">
        <span class="team-ic"><Icon name="users" size={16} /></span>
        <div class="set-txt">
          <div class="t">
            {teamLabel(tm)}
            <span class="chip">{deskLabel(tm.desk)}</span>
            <span class="team-count">{t('office.teamMembersCount', { n: tm.members.length })}</span>
          </div>
          <div class="d">
            {#if tm.default}{t('office.teamDefaultNote')}{:else if tm.description}{tm.description}{/if}
          </div>
          {#if tm.invalid}<div class="d team-bad">{t('office.teamInvalid', { reason: tm.invalid })}</div>{/if}
          {#if tm.missing.length > 0}<div class="d team-bad">{t('office.teamMissing', { names: tm.missing.join(', ') })}</div>{/if}
        </div>
        <div class="ag-actions">
          {#if s && !tm.invalid}
            <label class="mswitch" title={t('office.teamDelegate')}>
              <input type="checkbox" checked={!s.agents.off} disabled={busy !== ''}
                aria-label={t('office.teamDelegate')} onchange={() => toggleTeam(tm)} />
              <span></span>
            </label>
          {/if}
          {#if !tm.default}
            <button class="icobtn tiny tip-l" aria-label={t('office.teamEdit')} data-tip={t('office.teamEdit')}
              onclick={() => editTeam(tm)}><Icon name="settings" size={13} /></button>
            <button class="icobtn tiny tip-l" aria-label={t('office.teamDelete')} data-tip={t('office.teamDelete')}
              onclick={() => deleteTeam(tm.name, tm.path ?? '')}><Icon name="trash" size={13} /></button>
          {/if}
        </div>
      </div>
      {#each tm.members as c (c.name)}
        {@const reach = reachOf(tm, c.name)}
        <div class="set-row team-member">
          <AgentMascot name={c.name} {...lookOf(c)} size={24} />
          <div class="set-txt">
            <div class="t">{c.name}</div>
            {#if c.description}<div class="d">{c.description}</div>{/if}
          </div>
          {#if reach}
            <div class="ag-actions">
              <label class="mswitch" title={t('settings.agentReachTip')}>
                <input type="checkbox" checked={reach.on && !reach.off} disabled={reach.off || busy !== ''}
                  aria-label={t('settings.agentReach')} onchange={() => toggleMember(tm, c.name, reach.on)} />
                <span></span>
              </label>
            </div>
          {/if}
        </div>
      {/each}
      {#if loaded && tm.members.length === 0}
        <div class="set-row"><div class="set-txt"><div class="d">{tm.default ? t('office.noChairs') : t('office.teamEmpty')}</div></div></div>
      {/if}
    </div>
  {/each}
  <p class="muted set-sub">{t('office.teamsNote')}</p>
{/if}

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
