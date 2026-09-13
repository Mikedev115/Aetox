<script lang="ts">
  // The two chips at the composer's foot that say who this chat is with
  // (§85, §256): WHO answers, and which TEAM the chat hires from. Two chips
  // since 13 ก.ย. — one chip carried both, and the moment a chat had a team
  // the team's name replaced the assistant's, so it read as talking TO the
  // team (owner: "ตรงนี้ไม่ควรแสดงทีมเอเจน … หรือทำแยกดี แยกช่องเป็นช่องทีม
  // ไม่มีทีมก็ขึ้นว่าไม่มีทีม"). Each chip now answers one question, like the
  // folder, branch and shell chips beside them, and each menu holds one kind
  // of row: people under the first, teams under the second — "talk to doc"
  // and "hire from ทีมร้าน" were one list, and they are not one decision.
  //
  // Picking anything opens a NEW session — a desk, a chair or a team is fixed
  // for a session's life (setStation), so a switch here is a door to a fresh
  // one, never a dial on this one. The one live control is the door's own
  // delegation switch, under the team the chat hires from, because that is
  // the one fact about a roster somebody changes mid-chat.
  //
  // Its own file so it can be tested like TeamSettings, and because Chat.svelte
  // has no room for a third menu's worth of state.
  import { ListTeams, DelegateSwitches, SetDelegateOff, AgentBlocked } from '../../wailsjs/go/main/App'
  import { engine } from '../../wailsjs/go/models'
  import { cockpit, newChairSession, newTeamSession, setActiveView, openSettingsAt } from './stores/cockpit.svelte'
  import { t, i18n } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import Mascot from './mascot/Mascot.svelte'
  import { headOf, headOptions } from './mascot/avatarPrefs.svelte'
  import { lookOf } from './mascot/agentLook'
  import { avatarText } from './mascot/avatarText'

  // Which menu is up: at most one, like every other chip on the row.
  let open = $state<'' | 'who' | 'team'>('')
  // The desk decides which teams are offered — the storefront's behind the
  // assistant, the workshop's on the coding desk — and which head fronts
  // the chat (avatarPrefs headOf): its face AND its name. It was the door's
  // icon before the heads had faces (owner: "ไอคอนต้องเปลี่ยนหน้าโค้ด ไม่งั้น
  // UX พัง"), and one word, "ผู้ช่วยหลัก", on both desks until 14 ก.ย. 2026 —
  // the two heads had just been split and the chip still called the coder
  // by the assistant's name (owner: "ทำไมมันเขียนว่าผู้ช่วยหลัก มันคือโค้ด
  // เนี่ยแยกกันแล้ว"). The name is the avatar page's own card name for the
  // head (avatarText heads), so the chip and the page agree.
  const desk = $derived(cockpit.desk === 'coding' ? 'coding' : 'specialized')
  const head = $derived(headOf(desk))
  const headName = $derived(avatarText(i18n.locale).heads[head].name)

  let teams = $state<engine.TeamCard[]>([])
  // The team the chat hires from, and its people — the list the WHO menu
  // offers, the same list the engine offers `task`.
  const current = $derived(teams.find((x) => x.name === cockpit.team) ?? null)
  const members = $derived(current ? current.members : [])
  // The chair's card, for the face on the chip: the same face the roster and
  // the menu draw. Null until the teams have loaded, when the chip wears the
  // plain glyph for that instant.
  const chairCard = $derived(members.find((c) => c.name === cockpit.chair) ?? null)
  // Which teammates cannot work yet, by the same answer the roster's veil reads
  // (AgentLock.svelte): a locked row opens the roster instead of a session
  // they cannot use. Read when the menu opens, not held: the answer only
  // changes by installing something, which is a trip to another page and back.
  let blocked = $state<Record<string, boolean>>({})
  // The door's switch and what it costs, read when the menu opens: half of it
  // is a measurement (what `task` costs the block right now), and a cached
  // one would be right the day it was cached.
  let delegate = $state<engine.DelegateSettings | null>(null)
  let busy = $state(false)

  // The teams are read at mount and whenever the station changes, so the chip
  // can wear the chair's face and the team's name before any menu opens; the
  // locks and the switch wait for a menu, being dearer and only shown there.
  async function loadTeams() {
    try {
      teams = await ListTeams(desk)
    } catch {
      teams = []
    }
  }
  async function loadMenu() {
    await loadTeams()
    try {
      const veils = await Promise.all(members.map((c) => AgentBlocked(c.name)))
      blocked = Object.fromEntries(members.map((c, i) => [c.name, veils[i] ?? false]))
    } catch {
      blocked = {}
    }
    try {
      delegate = await DelegateSwitches(cockpit.team)
    } catch {
      delegate = null
    }
  }
  $effect(() => {
    // Once at mount and again on a station change; the reads are listed so
    // the effect tracks them, and the load itself runs outside the tracked
    // scope.
    void [cockpit.desk, cockpit.team, cockpit.chair]
    void loadTeams()
  })

  async function toggle(which: 'who' | 'team') {
    const next = open === which ? '' : which
    open = next
    if (next) await loadMenu()
  }
  // A press outside closes the menu — mousedown, not click, because the other
  // chips on the row stop their click from bubbling (they close Chat's own
  // menus themselves), and a menu that only closes by choosing is a menu that
  // sticks. Escape too.
  function onOutside(e: MouseEvent) {
    if (open && !(e.target as HTMLElement).closest('.station-pick')) open = ''
  }
  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) open = ''
  }

  // WHO: back to the assistant on the chat's team, or to one of its people.
  async function pickAssistant() {
    open = ''
    if (cockpit.chair) await newTeamSession(desk, cockpit.team)
  }
  async function pickChair(c: engine.Chair) {
    open = ''
    if (blocked[c.name]) { setActiveView('office'); return }
    if (cockpit.chair !== c.name) await newChairSession(c.name, desk, cockpit.team)
  }
  // TEAM: walking onto another roster is a new chat, like walking to a chair.
  // No team is a roster too — the empty one — and its row is a door like the
  // others (owner: "ปุ่มเลือกไม่มีทีมเอเจนไม่เห็นมี"): the chat then hires
  // nobody and the assistant does everything itself, and it reopens that way.
  async function pickTeam(name: string) {
    open = ''
    if (name === cockpit.team && !cockpit.chair) return
    await newTeamSession(desk, name)
  }
  // ตั้งค่า › ทีมเอเจน, opened on THIS desk's side — from the code page the
  // page must land on ฝั่งโค้ด, not the assistant's (owner: "พอกดจัดการทีม
  // มันพาไปฝั่งผู้ช่วย แทนที่จะเป็นฝั่งโค้ด").
  function manageTeams() {
    open = ''
    cockpit.settingsIntent = { section: 'teams', side: desk }
    openSettingsAt('teams')
  }

  // The DOOR's switch: the assistant door's on the assistant desk, the code
  // door's on the coding desk. How many of the current team are in its reach
  // is the number beside the team's name — readable before a click.
  const door = $derived(delegate ? (desk === 'coding' ? delegate.code : delegate.agents) : null)
  const doorRow = $derived(
    desk === 'coding'
      ? ({ kind: 'code', label: 'chat.delegateCode', on: 'chat.delegateCodeOn', off: 'chat.delegateCodeOff' } as const)
      : ({ kind: 'agents', label: 'chat.delegateAgents', on: 'chat.delegateAgentsOn', off: 'chat.delegateAgentsOff' } as const),
  )
  const inReach = $derived(
    delegate && door && !door.off && current
      ? current.members.filter((c) => delegate!.agents.workers.find((w) => w.name === c.name)?.on).length
      : 0,
  )
  // Flipping it re-bootstraps the engine, so the menu stays open and the row
  // stays disabled until the answer comes back — a switch that looks instant
  // and is not is a switch people press twice.
  async function toggleDoor() {
    if (!door || busy) return
    busy = true
    try {
      delegate = await SetDelegateOff(doorRow.kind, !door.off)
    } finally {
      busy = false
    }
  }
</script>

<svelte:window onmousedown={onOutside} onkeydown={onKey} />

<!-- WHO answers. The chair's face when the chat is with a chair; the door's
     own icon when it is with the assistant. -->
<div class="focus-pick station-pick station-who">
  {#if open === 'who'}
    <div class="focus-menu">
      <button type="button" class="focus-item" class:on={!cockpit.chair} onclick={pickAssistant}>
        <!-- The desk's head, as its own face (14 ก.ย. 2026) — the same one
             on the wall and the floating figure, not a desk glyph. -->
        <span class="ic"><Mascot {...headOptions(head)} size={16} still /></span> {headName}
      </button>
      {#if members.length > 0}<div class="menu-sep"></div>{/if}
      {#each members as c (c.name)}
        {@const locked = blocked[c.name] ?? false}
        <!-- `on` sits on the ROW: the row is what lights up, so it is also what
             knows it is the current one. A locked teammate is not disabled: a
             dead row says "no" and nothing else, where the roster's card says
             which tool is missing and offers to fetch it. -->
        <div class="agent-row" class:on={cockpit.chair === c.name}>
          <button type="button" class="focus-item" class:locked
            title={locked ? t('lock.body') : c.description} onclick={() => pickChair(c)}>
            <AgentMascot name={c.name} {...lookOf(c)} size={20} /><span class="t">{c.name}</span>
            {#if locked}<span class="focus-locked"><Icon name="wrench" size={12} /></span>{/if}
          </button>
        </div>
      {/each}
      {#if !cockpit.team}
        <div class="folder-note">{t('chat.whoNoTeam')}</div>
      {:else if members.length === 0}
        <div class="folder-note">{t('chat.teamEmpty')}</div>
      {/if}
      <div class="folder-note">{t('chat.agentSwitchNote')}</div>
    </div>
  {/if}
  <button type="button" class="focus-chip focus-btn" aria-expanded={open === 'who'} title={t('chat.pickWho')}
    onclick={() => toggle('who')}>
    <span class="ic">
      {#if cockpit.chair && chairCard}<AgentMascot name={chairCard.name} {...lookOf(chairCard)} size={14} />
      {:else if cockpit.chair}<Icon name="bot" size={13} />
      {:else}<Mascot {...headOptions(head)} size={15} still />{/if}
    </span>
    <span class="t">{cockpit.chair || headName}</span>
    <span class="caret"><Icon name={open === 'who' ? 'chevronUp' : 'chevronDown'} size={12} /></span>
  </button>
</div>

<!-- The TEAM the chat hires from. Not drawn behind a chair: a chair hires
     nobody, so there is no team to show. "ไม่มีทีม" is a state, dim and
     still a door, not an absence. -->
{#if !cockpit.chair}
  <div class="focus-pick station-pick station-team" class:none={!cockpit.team}>
    {#if open === 'team'}
      <div class="focus-menu">
        {#each teams as tm (tm.name)}
          {@const here = tm.name === cockpit.team}
          <div class="agent-row team-row" class:on={here} class:invalid={!!tm.invalid}>
            <button type="button" class="focus-item" title={tm.invalid || tm.description} onclick={() => pickTeam(tm.name)}>
              <span class="ic"><Icon name="users" size={14} /></span>
              <span class="t">{tm.name}</span>
              <span class="team-n">
                {#if here && delegate}{t('chat.teamReach', { n: inReach, m: tm.members.length })}{:else}{tm.members.length}{/if}
              </span>
            </button>
          </div>
          {#if here && door}
            <!-- The door's switch, under the team it applies to, saying what it
                 costs — measured, not remembered (App.DelegateSwitches). -->
            <button type="button" class="focus-item delegate-row" class:on={!door.off}
              role="switch" aria-checked={!door.off} disabled={busy} onclick={toggleDoor}>
              <span class="ic"><Icon name="userRound" size={14} /></span>
              <span class="t">{t(doorRow.label)}</span>
              <span class="mswitch-face"></span>
            </button>
            <div class="folder-note">{t(door.off ? doorRow.off : doorRow.on, { n: door.tokens.toLocaleString() })}</div>
          {/if}
        {/each}
        {#if teams.length === 0}
          <div class="folder-note">{t(desk === 'coding' ? 'chat.noTeamsHere' : 'chat.noTeamsAssistant')}</div>
        {/if}
        <div class="agent-row team-row no-team" class:on={!cockpit.team}>
          <button type="button" class="focus-item" title={t('chat.pickNoTeamTip')} onclick={() => pickTeam('')}>
            <span class="ic"><Icon name="userRound" size={14} /></span>
            <span class="t">{t('chat.pickNoTeam')}</span>
          </button>
        </div>
        <div class="menu-sep"></div>
        <button type="button" class="focus-item" onclick={manageTeams}>
          <span class="ic"><Icon name="settings" size={14} /></span> {t('chat.manageTeams')}
        </button>
        {#if teams.length > 0}<div class="folder-note">{t('chat.teamSwitchNote')}</div>{/if}
      </div>
    {/if}
    <button type="button" class="focus-chip focus-btn" aria-expanded={open === 'team'} title={t('chat.pickTeam')}
      onclick={() => toggle('team')}>
      <span class="ic"><Icon name="users" size={13} /></span>
      <span class="t">{cockpit.team || t('chat.noTeam')}</span>
      <span class="caret"><Icon name={open === 'team' ? 'chevronUp' : 'chevronDown'} size={12} /></span>
    </button>
  </div>
{/if}
