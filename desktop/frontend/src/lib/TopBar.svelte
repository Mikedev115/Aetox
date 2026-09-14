<script lang="ts">
  import { t } from './i18n.svelte'
  import { cockpit, newSession, switchShell, askingElsewhere, selectGlobalSession, setActiveView } from './stores/cockpit.svelte'
  import { shell, SHELLS, offeredShells } from './shell.svelte'
  import { shortcutLabel } from './shortcuts'
  import Wordmark from './Wordmark.svelte'
  import Icon from './Icon.svelte'
  import Mascot from './mascot/Mascot.svelte'
  import RankedFace from './RankedFace.svelte'
  import { headOf, headOptions } from './mascot/avatarPrefs.svelte'
  import SessionStrip from './SessionStrip.svelte'
  import { openArtifactsTab } from './stores/workbench.svelte'
  import { codeStatus } from './stores/codeStatus.svelte'
  import { engine } from './stores/engine.svelte'

  let {
    inspectorCollapsed, onToggleInspector, sidebarCollapsed, onToggleSidebar,
  }: {
    inspectorCollapsed: boolean
    onToggleInspector: () => void
    sidebarCollapsed: boolean
    onToggleSidebar: () => void
  } = $props()

  // What this conversation is, in the band that had nothing in it.
  //
  // The row was one toggle, a flex spacer, and three corner buttons — chrome at
  // both ends and a deliberate void between (owner, 2026-08-14, next to a
  // reference where the same band carries the conversation's name).
  //
  // Read off the history list rather than held separately: refreshSessions
  // already marks which row is live, and a second copy of "what is this chat
  // called" is the one that goes stale the moment a title is rewritten.
  //
  // Empty until the first turn, on purpose. A session is only titled once it
  // has been spoken to, and a placeholder there would name a conversation that
  // has not happened.
  const title = $derived(cockpit.sessions.find((s) => s.active)?.title ?? '')

  // The door switcher (§86). On the wordmark rather than in the room list,
  // because it does not change which room you are in — it changes which
  // building you are standing in, and the two questions must not look like one
  // control. Same place, and the same gesture, as ChatGPT↔Codex.
  let doorOpen = $state(false)
  const current = $derived(SHELLS.find((s) => s.name === shell.name) ?? SHELLS[0])
  // While a press is walking, the door wears the TARGET's name and a spinner
  // where the caret was: it says where it is going, not where it was. The
  // chrome has already switched (switchShellNow sets the shell first), so
  // `current` is usually the target already — the spinner is what says the
  // chat has not caught up yet.
  const going = $derived(SHELLS.find((s) => s.desk && s.desk === cockpit.walkingTo))


  // The chat that has stopped on a question while the user is somewhere else.
  //
  // Drawn here, in the band's empty middle, because this band is the one
  // thing on screen from every chat and every file tab — the card itself is
  // inside one chat's transcript, and the sidebar row can be scrolled away or
  // folded. One strip even when two chats are asking: it names the first and
  // counts the rest, and answering the first puts the next one here.
  const asking = $derived(askingElsewhere())

  async function goAnswer() {
    const target = asking[0]
    if (!target) return
    if (target.id === cockpit.openSession) {
      // The card is on screen already, behind another page.
      setActiveView('chat')
      return
    }
    await selectGlobalSession({ id: target.id, title: target.title, ago: '' })
  }

  function closeOnOutsideClick(e: MouseEvent) {
    if (!(e.target as HTMLElement).closest('.brand')) doorOpen = false
  }

  async function pick(name: (typeof SHELLS)[number]['name']) {
    doorOpen = false
    await switchShell(name)
  }
</script>

<!-- The window frame with the panel's own column drawn inside it: filled while
     that panel is showing, empty once it's collapsed. Reads as "which side,
     and is it there" at a glance, which two near-identical block glyphs did
     not. The right-hand button mirrors the same icon. -->
{#snippet panelIcon(showing: boolean, mirrored: boolean)}
  <svg
    viewBox="0 0 16 16" width="16" height="16" aria-hidden="true"
    fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"
    style={mirrored ? 'transform:scaleX(-1)' : ''}
  >
    <rect x="1.65" y="2.9" width="12.7" height="10.2" rx="2.2" />
    <path d="M6.1 3.4V12.6" />
    {#if showing}
      <rect x="2.4" y="3.65" width="3" height="8.7" rx="1.1" fill="currentColor" stroke="none" opacity=".55" />
    {/if}
  </svg>
{/snippet}

<svelte:window onclick={doorOpen ? closeOnOutsideClick : undefined} />

{#snippet doorItemInner(s: (typeof SHELLS)[number], walking: boolean)}
  <span class="ic" class:face={!!s.desk}>
    {#if walking}<span class="walk-spin"><Icon name="loaderCircle" size={15} /></span>
    {:else if s.desk}<RankedFace tier="head" size={38}><Mascot {...headOptions(headOf(s.desk))} pose="idle" size={38} still /></RankedFace>
    {:else}<Icon name={s.icon} size={15} />{/if}
  </span>
  <span class="txt">
    <span class="t">{t(s.labelKey)}</span>
    <span class="d">{walking ? t('shell.opening') : t(s.blurbKey)}</span>
  </span>
  {#if shell.name === s.name && !walking}<span class="tick"><Icon name="check" size={13} /></span>{/if}
{/snippet}

<div class="brand">
  <button
    type="button" class="brand-btn" data-guide="topbar.door" aria-haspopup="menu" aria-expanded={doorOpen}
    aria-label={t('shell.switch')} onclick={() => (doorOpen = !doorOpen)}
  >
    <Wordmark height={20} />
    <span class="brand-door" class:walking={!!going}>{t((going ?? current).labelKey)}</span>
    <span class="brand-caret">
      {#if going}<span class="walk-spin"><Icon name="loaderCircle" size={12} /></span>{:else}<Icon name={doorOpen ? 'chevronUp' : 'chevronDown'} size={12} />{/if}
    </span>
  </button>
  {#if doorOpen}
    <div class="door-menu" role="menu">
      <!-- offeredShells, not SHELLS: a door that is built but has nothing
           behind it yet stays off the menu rather than showing as a button
           that disappoints (shell.svelte's `offered`). -->
      {#each offeredShells() as s (s.name)}
        {@const walking = going?.name === s.name}
        {#if s.name === 'assistant'}
          <button type="button" class="door-item" data-guide="topbar.door.assistant" class:on={shell.name === s.name} role="menuitem" onclick={() => pick(s.name)}>
            {@render doorItemInner(s, walking)}
          </button>
        {:else if s.name === 'code'}
          <button type="button" class="door-item" data-guide="topbar.door.coding" class:on={shell.name === s.name} role="menuitem" onclick={() => pick(s.name)}>
            {@render doorItemInner(s, walking)}
          </button>
        {:else}
          <button type="button" class="door-item" class:on={shell.name === s.name} role="menuitem" onclick={() => pick(s.name)}>
            {@render doorItemInner(s, walking)}
          </button>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<div class="topbar">
  <button
    class="icobtn tip-l" data-guide="topbar.sidebar_btn" aria-label={sidebarCollapsed ? t('topbar.showSidebar') : t('topbar.hideSidebar')}
    data-tip="{t('topbar.toggleSidebarTip')} · {shortcutLabel('toggleSidebar')}" onclick={onToggleSidebar}
  >
    {@render panelIcon(!sidebarCollapsed, false)}
  </button>
  <span class="topbar-space" data-guide="topbar.space" style={cockpit.space ? '' : 'display:none'}>{cockpit.space || ''}</span>
  <!-- Left of the spacer, not centred in it: a centred title moves every time
       its own length changes, and it collides with the corner buttons on a
       narrow window. Against the toggle it has a fixed address. -->
  {#if title}<span class="topbar-title" data-guide="topbar.tab.chat" title={title}>{title}</span>{/if}
  <!-- Where the engine is, when it is not here (§248 phase 3). The window
       looks the same on a host — that is the design — so this is the one
       line that says the code, the terminal and the chats on screen are
       another machine's. A button, because the answer to "where am I" is
       usually followed by "take me to the page about it". -->
  {#if engine.status.mode === 'remote'}
    <button type="button" class="host-badge" title={t('engine.onHost', { host: engine.status.host })} onclick={() => setActiveView('settings')}>
      <Icon name="server" size={12} />
      <span>{engine.status.host}</span>
    </button>
  {/if}
  <span class="spacer"></span>
  {#if asking.length > 0}
    <button type="button" class="ask-strip" onclick={goAnswer} title={asking[0].question}>
      <span class="dot ask" aria-hidden="true"></span>
      <span class="who">{asking[0].title || t('topbar.askingUnnamed')}</span>
      <span class="what">{asking.length > 1 ? t('topbar.askingMore', { n: asking.length - 1 }) : t('topbar.askingYou')}</span>
      <span class="go">{t('topbar.askingGo')}</span>
    </button>
    <span class="spacer"></span>
  {/if}

  <!-- tip-r on both: these sit flush against the window's right edge, so a
       centred (or left-anchored) tooltip gets clipped by it. -->
  <div class="winbtns">
    <!-- What this conversation holds — plan first, and the room's sources and
         repo state as those land. Leftmost of the three so the two panel
         toggles stay adjacent: they are the same kind of act (show a rail),
         and this one is not. -->
    <button
      class="icobtn tip-r" aria-label={t('workbench.artifactsTab')}
      data-tip={t('workbench.artifactsTab')} onclick={() => openArtifactsTab()}
    ><Icon name="package" size={15} /></button>
    <SessionStrip />
    <!-- Always, not only while the sidebar is away (owner, 12 ส.ค.). It was
         conditional on the reasoning that the sidebar's header carries this
         otherwise, and that two + buttons on one line is one of them saying
         nothing — true about the pixels, wrong about the act. Starting a new
         chat is the one thing you do from anywhere, and a button that is
         sometimes there is one you have to look for every time: you check the
         corner, find nothing, remember the panel, open the panel. A control at
         a fixed address costs a duplicate; one that moves costs a search on
         every use. -->
    <button
      class="icobtn tip-r" aria-label={t('sidebar.newSession')}
      data-tip="{t('sidebar.newSession')} · {shortcutLabel('newSession')}" onclick={newSession}
    ><Icon name="plus" size={15} /></button>
    <button
      class="icobtn tip-r" data-guide="topbar.inspector_btn" aria-label={inspectorCollapsed ? t('topbar.showPanel') : t('topbar.hidePanel')}
      data-tip="{t('topbar.toggleInspectorTip')} · {shortcutLabel('toggleInspector')}" onclick={onToggleInspector}
    >
      {@render panelIcon(!inspectorCollapsed, true)}
      {#if inspectorCollapsed && (codeStatus.gitChangedCount > 0 || codeStatus.openPRCount > 0)}
        <span class="panel-notice-dot" aria-hidden="true"></span>
      {/if}
    </button>
  </div>
</div>
