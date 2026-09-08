<script lang="ts">
  import { onMount, tick } from 'svelte'
  import Terminal from '../Terminal.svelte'
  import FileEditor from '../FileEditor.svelte'
  import FilesPane from './FilesPane.svelte'
  import BrowserPane from './BrowserPane.svelte'
  import ExternalFilePane from './ExternalFilePane.svelte'
  import SheetPane from './SheetPane.svelte'
  import ImagePane from './ImagePane.svelte'
  import MediaPane from './MediaPane.svelte'
  import PdfPane from './PdfPane.svelte'
  import SlidesPane from './SlidesPane.svelte'
  import DeckRoom from './DeckRoom.svelte'
  import CuttingRoom from './CuttingRoom.svelte'
  import GitPane from './GitPane.svelte'
  import PRPane from './PRPane.svelte'
  import RepoMapPane from './RepoMapPane.svelte'
  import { fileURL } from '../fileUrl'
  import { cockpit } from '../stores/cockpit.svelte'
  import {
    workbench, activateTab, closeTab, removeTab,
    openFilesTab, openBrowserTab, openTerminalTab, openDecksTab, openGitTab, openPRTab, openRepoMapTab, openCuttingRoomTab, openFileTab, routeDeskEvent, detachTab,
    reportDeskTabs,
    openUrlInWorkbench, saveWorkbenchSnapshot, resolveAddressBarInput, labelForUrl,
    deviceList, loadDevices,
    setTabDragPayload, TAB_DRAG_MIME,
    type WorkbenchTab,
  } from '../stores/workbench.svelte'
  import { busy, busyWork, layerOn, loadBusySignal, toggleBusyLayer } from '../stores/busySignal.svelte'
  import { TerminalShells, BrowserBack, BrowserForward, BrowserReload, BrowserOpenDevTools, BrowserSetDevice, BrowserDetach } from '../../../wailsjs/go/main/App'
  import { pagePick, startPagePick, stopPagePick, type PickMode } from './pagePick.svelte'
  import { EventsOn } from '../../../wailsjs/runtime/runtime'
  import { t, type TKey } from '../i18n.svelte'
  import { isShortcut, shortcutLabel } from '../shortcuts'
  import Icon from '../Icon.svelte'
  import { sidle } from '../fold'
  import type { IconName } from '../icons'

  const tabIcon: Record<string, IconName> = { terminal: 'keyboard', browser: 'globe', files: 'copy', file: 'fileText', decks: 'layoutList', cutroom: 'scissors' }

  // Chrome DevTools' default device presets. CSS viewport sizes — BrowserPane
  // turns one into a real window of that aspect + a matching page zoom.
  // The device list is Go's (desktop/browser_device.go), not ours.
  //
  // It was eight literals here until the agent got a `device` action and needed
  // the same eight. Two lists would have drifted the first time one gained a
  // phone, and the menu and the agent would have been offering different
  // machines under the same name.
  //
  // Only the sizes come across. What a device IS — its pixel ratio, its user
  // agent, whether it has a touch screen — is applied by Go through the engine,
  // and is not something this file should be able to have an opinion about.
  const DEVICES = $derived(deviceList.rows)
  $effect(() => {
    void loadDevices()
  })

  let shells = $state<{ name: string; path: string }[]>([])
  let menuOpen = $state(false)
  // The browser toolbar's overflow menu, and which section inside it is open.
  //
  // One menu where there were four buttons and a native <select> (owner, 8 ก.ย.:
  // *"เราห่อรวมใน สามจุดได้ไหม"*). Its sections are disclosures rather than
  // fly-outs: eight device sizes and four busy-signal layers flat would be a
  // fifteen-row menu, and a second floating layer over a native browser window
  // is the one thing this panel has learnt not to build.
  let moreOpen = $state(false)
  let moreSection = $state<'' | 'size' | 'busy'>('')
  // Every menu this toolbar can open, as one fact for the browser pane.
  //
  // A native WebView2 window composites ABOVE everything the app draws, so a
  // DOM menu over the pane is invisible unless the page hides first. The pane
  // does hide for one thing it can measure — something covering the CENTRE of
  // its box — and a menu hanging off the toolbar covers the top and never the
  // centre. That is exactly why ไฟบอกสถานะ opened onto nothing: the checklist
  // was drawn, correctly, underneath the page. The panel has to SAY a menu is
  // open; it cannot be inferred.
  const panelMenuOpen = $derived(menuOpen || moreOpen)
  let urlDraft = $state('')

  const activeTab = $derived(workbench.tabs.find((t) => t.id === workbench.activeId))
  const hasActiveTask = $derived(cockpit.task.steps.some((s) => s.status === 'active'))

  // ── ไฟบอกสถานะ (§174) ──────────────────────────────────────────────
  //
  // Three of the four layers are drawn here; the fourth is drawn inside the
  // page itself, because it has to be (desktop/browser_marks.go).
  //
  // Each is a switch AND a fact, never one of them. The switch says what the
  // panel may draw, the fact says whether there is anything to draw, and a
  // layer that lit on the switch alone would be a light that means "you left
  // this on" rather than "the agent is working".
  //
  // The three read different facts on purpose. The border and the tab mark
  // follow `running`, which is one call in flight and nothing else. The action
  // bar follows `seen` — the browser has been touched at some point this turn
  // — because mounting it resizes the native page underneath, and doing that
  // twice per call would reflow a page in the middle of being read (see
  // busyWork.seen).
  const busyGlow = $derived(layerOn('edgeGlow') && busyWork.running)
  const busyBar = $derived(layerOn('actionBar') && busyWork.seen)
  const busyDot = $derived(layerOn('tabDot') && busyWork.running)

  // The browser actions the bar has words for. Anything else lands on `other`,
  // which is a real sentence rather than a blank: an action added to the
  // browser tool and not to the dictionary should read as work being done, not
  // as the signal having broken.
  const BUSY_ACTS = new Set(['open', 'read', 'click', 'type', 'scroll', 'capture', 'tabs', 'wait', 'back', 'dialog', 'console', 'network', 'hover', 'drag', 'key', 'upload'])

  /** What the bar says: the action in words, with the thing it is being done to.
   *
   *  Two forms per action rather than one phrase with the tense bolted on, so
   *  each language writes its own sentence — Thai puts the tense at both ends
   *  ("กำลังกด X" / "กด X แล้ว") and English changes the verb.
   *
   *  The collapse at the end is what lets one phrase serve an action that names
   *  something and one that does not: `click` has no subject, so "{subject}"
   *  resolves to nothing and the space it left goes with it. */
  const busyText = $derived.by(() => {
    const act = BUSY_ACTS.has(busyWork.act) ? busyWork.act : 'other'
    // A wait that has reported its progress gets its own phrase, because it is
    // the only action with something to say while it is still going: ten
    // minutes of an unchanging line is indistinguishable from a hang, and the
    // reasonable thing to do about a hang is press Stop. Only while running,
    // and only once a number has arrived — a wait shorter than the first tick
    // reads exactly as it did before, which is nearly all of them.
    if (busyWork.running && act === 'wait' && busyWork.waitTotal > 0) {
      return t('workbench.busyRun.waitProgress', {
        subject: busyWork.subject,
        elapsed: String(busyWork.waitElapsed),
        total: String(busyWork.waitTotal),
      })
        .replace(/\s+/g, ' ')
        .trim()
    }
    const key = `workbench.busy${busyWork.running ? 'Run' : 'End'}.${act}` as TKey
    return t(key, { subject: busyWork.subject }).replace(/\s+/g, ' ').trim()
  })

  $effect(() => {
    urlDraft = activeTab?.url ?? ''
  })

  // Autosave the layout for the bound session on every tab change (open/close/
  // navigate/activate) — snapshot reads workbench state reactively.
  $effect(() => {
    saveWorkbenchSnapshot()
    // Same trigger, same reason: the agent's view of the desk has to change
    // when the desk does, including when the user is the one who changed it.
    reportDeskTabs()
  })

  onMount(() => {
    TerminalShells().then((s) => (shells = s))
    // Read once, here rather than when the checklist opens. The panel draws
    // from these on the first browser call of the session, which is long before
    // anybody has a reason to open the menu — and layerOn's shipped-default
    // fallback is meant to cover the milliseconds of a round trip, not a user
    // who turned a layer off yesterday.
    void loadBusySignal()
    // The ways the agent reaches this desk, all through one door (§187.3):
    // every event carries its session, and routeDeskEvent is the single place
    // that decides live desk vs a background chat's saved one. A new desk
    // surface subscribes here and answers "whose desk" in the router's switch,
    // or it draws nothing — never a policy improvised per handler again.
    const offs = ['open-browser', 'close-browser', 'open-file', 'close-file', 'focus-tab', 'open-terminal', 'open-media', 'open-cutroom'].map((kind) =>
      EventsOn(`workbench:${kind}`, (payload: Record<string, unknown>) => routeDeskEvent(kind, payload)),
    )
    return () => offs.forEach((off) => off())
  })

  function openDefaultTerminal() {
    if (shells.length === 0) return
    menuOpen = false
    const saved = localStorage.getItem('defaultShell')
    openTerminalTab(shells.find((s) => s.path === saved) ?? shells[0])
  }

  function pick(fn: () => void) {
    menuOpen = false
    fn()
  }

  /** Device-size preset for the active browser tab; '' = fill the pane.
   *
   * Two halves, and they are two because they happen in two places. The size is
   * ours: BrowserPane shrinks the native window to it, which is what makes the
   * pane look like a phone. Everything else about being a phone — the user
   * agent, the touch screen, the pixel ratio — only exists inside the engine,
   * so Go sets it there. Doing only the first half is what made a mobile preset
   * show the desktop page in a narrow window. */
  function setViewport(name: string) {
    if (!activeTab) return
    activeTab.viewport = DEVICES.find((d) => d.name === name)
    BrowserSetDevice(activeTab.id, name)
  }

  async function navigate() {
    const u = urlDraft.trim()
    if (!u) return
    // Async because Go decides whether this is a place or a search — one line
    // of code and one round trip, in exchange for the address bar behaving the
    // way every other address bar does. See resolveAddressBarInput.
    const { url, fallback } = await resolveAddressBarInput(u)
    if (!url) return
    let tab = activeTab
    if (!tab || tab.kind !== 'browser') {
      const id = openBrowserTab()
      tab = workbench.tabs.find((x) => x.id === id)
      if (!tab) return
    }
    // Before the URL: BrowserPane's effect watches `url` and reads `fallback`
    // as it fires, so the arming has to already be there when it does.
    tab.fallback = fallback
    tab.url = url
    tab.name = labelForUrl(url)
  }

  function browserCmd(fn: (id: string) => Promise<void>) {
    const tab = activeTab
    if (tab?.kind === 'browser' && tab.url) fn(tab.id)
  }

  /** Pull the active browser tab out into a window of its own.
   *
   * Two halves, in this order and not the other: Go reparents the native window
   * first, and only then does the tab leave the strip. Removing the chip first
   * would unmount BrowserPane, and an unmounted pane is a tab whose bounds and
   * visibility nobody is driving — for the fraction of a second before the
   * detach lands, the window would be a child of the app with nothing telling
   * it where to sit.
   *
   * It does NOT close anything. detachTab takes the chip off the strip and out
   * of the saved layout; the window and its page carry on, which is the whole
   * point of the button. */
  async function detachTabInOwnWindow(id: string) {
    await BrowserDetach(id)
    detachTab(id)
  }

  // Same gate as browserCmd: a tab still on its start page has no native
  // window to inject anything into.
  function togglePick(mode: PickMode) {
    const tab = activeTab
    if (tab?.kind === 'browser' && tab.url) startPagePick(tab.id, mode)
  }

  // Lets a file/browser tab be dragged into the chat composer to attach its
  // content — see Chat.svelte's ondrop, which reads this same MIME type.
  //
  // A browser tab still on its start page is deliberately not draggable: it has
  // no native window and therefore no text, and offering the drag anyway ended
  // in `no browser tab "web-2"` after the user had already let go.
  // `!!tab.path` is not belt-and-braces: a drop that failed outright also opens
  // a file tab, and that one has no path (openDropError). Dragging it staged an
  // attachment with an empty path, which then went into the transcript as
  // "[attachment: … read it with read] " with nothing after it — a wasted turn
  // and a stored line that re-renders broken on every reopen.
  const canDrag = (tab: WorkbenchTab) =>
    (tab.kind === 'file' && !!tab.path) || (tab.kind === 'browser' && !!tab.url)
  function onTabDragStart(e: DragEvent, tab: WorkbenchTab) {
    if (!canDrag(tab)) return
    const ref = tab.kind === 'file' ? tab.path ?? '' : tab.id
    setTabDragPayload(e, tab.kind as 'file' | 'browser', ref, tab.name)
  }

  // ---- the desk as a drop target ----
  // Whatever the user is already holding lands here and opens: a file the agent
  // just made, dragged off its card in the chat; a page dragged off a real
  // browser's tab strip; anything at all out of Explorer.
  //
  // The window-level watch is not decoration. A browser tab's page is a real OS
  // window composited over this pane (see BrowserPane), so while one is up the
  // DOM underneath cannot see a drag at all — the pointer is over another
  // window. Noticing the drag anywhere in the app and lowering that window for
  // its duration is what makes this pane reachable; without it, dropping onto a
  // desk with a page open would silently do nothing.
  let dragging = $state(false) // something droppable is in flight over the app
  let overDesk = $state(false) // ...and it is over this panel right now
  let dragIdle: ReturnType<typeof setTimeout> | undefined

  function droppable(dt: DataTransfer | null): boolean {
    if (!dt) return false
    return dt.types.includes('Files') || dt.types.includes('text/uri-list') || dt.types.includes(TAB_DRAG_MIME)
  }

  function onWindowDragOver(e: DragEvent) {
    if (!droppable(e.dataTransfer)) return
    dragging = true
    // A drag that started in Explorer and wanders back out of the window fires
    // no dragend here — it was never ours to end. So the end of one is inferred
    // from dragover falling quiet (Chromium repeats it while the pointer is
    // inside, ~every 350ms even when still).
    clearTimeout(dragIdle)
    dragIdle = setTimeout(endDrag, 800)
  }

  function endDrag() {
    clearTimeout(dragIdle)
    dragging = false
    overDesk = false
  }

  function onDeskDragOver(e: DragEvent) {
    if (!droppable(e.dataTransfer)) return
    e.preventDefault() // the one thing that makes a drop land at all
    e.dataTransfer!.dropEffect = 'copy'
    overDesk = true
  }

  function onDeskDragLeave(e: DragEvent) {
    // Crossing between children fires leave/enter pairs; only a leave that
    // actually exits the panel counts.
    if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node | null)) overDesk = false
  }

  // Text dragged out of a page is a selection, not an address. Only something
  // that already reads as a URL gets opened as one — https://foo, E:\a\b.html,
  // or a bare host — so dropping three highlighted words doesn't navigate.
  function looksLikeUrl(s: string): boolean {
    if (/\s/.test(s)) return false
    return /^[a-z][a-z0-9+.-]*:/i.test(s) || /^[a-z]:[\\/]/i.test(s) || /^[\w-]+(\.[\w-]+)+(:\d+)?([/?#].*)?$/.test(s)
  }

  async function onDeskDrop(e: DragEvent) {
    const dt = e.dataTransfer
    endDrag()
    if (!dt) return
    // A dropped OS file carries no readable path in the DOM. Wails resolves
    // those natively and App.svelte routes them here by drop coordinates
    // (OnFileDrop) — this handler must not swallow the event first.
    if (dt.types.includes('Files')) return
    e.preventDefault()

    const card = dt.getData(TAB_DRAG_MIME)
    if (card) {
      try {
        const { kind, ref, label } = JSON.parse(card) as { kind: 'file' | 'browser'; ref: string; label: string }
        if (kind === 'file') await openFileTab(ref, label)
        else activateTab(ref) // a browser tab dragged inside its own desk: just focus it
      } catch {
        // Not our payload after all — nothing to open.
      }
      return
    }

    const url = (dt.getData('text/uri-list') || dt.getData('text/plain') || '')
      .split('\n')
      .map((l) => l.trim())
      .find((l) => l && !l.startsWith('#')) // uri-list comments
    if (url && looksLikeUrl(url)) {
      const addr = await resolveAddressBarInput(url)
      if (addr.url) openUrlInWorkbench(addr.url, addr.fallback)
    }
  }

  // Slide an open dropdown back inside the window.
  //
  // The + button rides at the end of the tab strip, so where it sits depends on
  // how many tabs are open and how wide the panel is. `.plus-menu` hangs from
  // its left edge, and at 230px wide it ran off the right of the window as soon
  // as a tab or two had pushed the button over: the owner's screenshot showed
  // "หน้าพัฒนาโค้ด", Git, Pull requests and แผน with their labels sheared off at
  // the window frame. Nothing in the DOM clips it (.insp-tabs deliberately has
  // no overflow-x for exactly this reason) — the clip is the window edge, which
  // no ancestor style can fix.
  //
  // A shift, not a flip. Flipping to right:0 would tear the menu away from the
  // button it belongs to on every narrow panel; moving it the few pixels it
  // overhangs keeps it attached and keeps the rows readable. The shift is
  // bounded by the room actually available on the left, so a panel too narrow
  // to hold the menu at all loses the same edge it would have lost anyway
  // rather than losing the other one instead.
  const menuGutter = 8
  function keepOnScreen(el: HTMLElement) {
    const fit = () => {
      el.style.marginLeft = '0px'
      const box = el.getBoundingClientRect()
      const overhang = box.right - (window.innerWidth - menuGutter)
      if (overhang <= 0) return
      const room = Math.max(box.left - menuGutter, 0)
      el.style.marginLeft = `${-Math.min(overhang, room)}px`
    }
    fit()
    window.addEventListener('resize', fit)
    return { destroy: () => window.removeEventListener('resize', fit) }
  }

  function closeMenuOnOutsideClick(e: MouseEvent) {
    const el = e.target as HTMLElement
    if (!el.closest('.plus-menu-wrap')) menuOpen = false
    if (!el.closest('.more-wrap')) closeMore()
  }

  function closeMore() {
    moreOpen = false
    moreSection = ''
  }

  /** Run a browser command from inside the menu, after the menu has gone.
   *
   * The order is the whole function, and it is not tidiness. Opening this menu
   * HIDES the page — it has to, or the menu is drawn underneath a native window
   * — and a hidden WebView2 refuses work that needs a live view: `capture` has
   * always known this (browser_capture.go says an SW_HIDE window produces no
   * frames), and DevTools is the same kind of call. Clicking เครื่องมือนักพัฒนา
   * did nothing at all for exactly that reason, from the moment the menu shipped.
   *
   * `await tick()` is what makes it work rather than what makes it likely: the
   * pane's visibility effect runs in that flush and issues BrowserSetVisible,
   * and every one of these calls lands on the browser host's single FIFO queue
   * — so the show is queued before the command, and runs before it. */
  async function menuCmd(fn: (id: string) => Promise<void>) {
    const tab = activeTab
    closeMore()
    if (tab?.kind !== 'browser' || !tab.url) return
    await tick()
    await fn(tab.id)
  }

  /** Open the overflow menu, or one of its sections.
   *
   * Sections are exclusive: opening one closes the other, so the menu never
   * grows past a screen and the thing just clicked is always in view. */
  function toggleMoreSection(name: 'size' | 'busy') {
    moreSection = moreSection === name ? '' : name
    if (name === 'busy' && moreSection === 'busy') void loadBusySignal()
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') { menuOpen = false; closeMore(); if (pagePick.tabId) stopPagePick(); return }
    if (isShortcut(e, 'browserTab')) { e.preventDefault(); openBrowserTab() }
    else if (isShortcut(e, 'filesTab')) { e.preventDefault(); openFilesTab() }
    // Only reaches here while the app's own webview has focus. Once the page
    // has it, the chord is the page's to see — which is why the injected
    // overlay listens for Escape itself rather than trusting this handler.
    else if (isShortcut(e, 'pickElement')) { e.preventDefault(); togglePick('pick') }
    else if (isShortcut(e, 'drawOnPage')) { e.preventDefault(); togglePick('draw') }
  }
</script>

<svelte:window
  onclick={menuOpen || moreOpen ? closeMenuOnOutsideClick : undefined} onkeydown={onKeydown}
  ondragover={onWindowDragOver} ondrop={endDrag} ondragend={endDrag}
/>

<!-- Every way to open a tab, written once.
     
     This list existed twice: in the + menu, and again in the panel an empty
     desk shows. The two were the same seven rows with the same desk gate, and
     they drifted the moment one of them was edited — a group heading added to
     the menu on 31 ส.ค. simply was not there on the empty desk, which is the
     copy the owner happened to be looking at.

     `pick` closes the menu before opening; on the empty desk there is no menu
     open, so the same call does the same thing in both places.

     โค้ด desk only for the last three: a working tree is what that desk is
     held inside, and the storefront has no project to report on (§161.4). The
     heading sits INSIDE that gate rather than above it, so it leaves with the
     rows it names. It exists because the gate was invisible: on another desk
     the list was simply shorter, with nothing saying which three had gone or
     why. The four above get no heading of their own — they have nothing to
     explain, and a label reading "ทั่วไป" is a line you read and get nothing
     back from. -->
{#snippet tabChoices()}
  <button class="plus-menu-item" disabled={shells.length === 0} onclick={openDefaultTerminal}><span class="ic"><Icon name="keyboard" size={14} /></span> {t('workbench.terminalMenu')}</button>
  <button class="plus-menu-item" onclick={() => pick(openBrowserTab)}><span class="ic"><Icon name="globe" size={14} /></span> {t('workbench.browserMenu')} <span class="kbd">{shortcutLabel('browserTab')}</span></button>
  <button class="plus-menu-item" onclick={() => pick(openFilesTab)}><span class="ic"><Icon name="copy" size={14} /></span> {t('workbench.filesTab')} <span class="kbd">{shortcutLabel('filesTab')}</span></button>
  <button class="plus-menu-item" onclick={() => pick(openDecksTab)}><span class="ic"><Icon name="layoutList" size={14} /></span> {t('workbench.decksTab')}</button>
  {#if cockpit.desk === 'coding'}
    <div class="plus-menu-head">{t('workbench.codeGroup')}</div>
    <button class="plus-menu-item" onclick={() => pick(openGitTab)}><span class="ic"><Icon name="gitBranch" size={14} /></span> {t('workbench.gitTab')}</button>
    <button class="plus-menu-item" onclick={() => pick(openPRTab)}><span class="ic"><Icon name="gitBranch" size={14} /></span> {t('workbench.prTab')}</button>
    <button class="plus-menu-item" onclick={() => pick(openRepoMapTab)}><span class="ic"><Icon name="graph" size={14} /></span> {t('workbench.repoMapTab')}</button>
  {/if}
  <!-- Same gate the code group draws with, one coordinate over: the room is
       the editor's, so its row exists where that chair is sat (§85). The
       heading is inside the gate for the code group's own reason - it leaves
       with the row it names. -->
  {#if cockpit.chair === 'editor'}
    <div class="plus-menu-head">{t('workbench.videoGroup')}</div>
    <button class="plus-menu-item" onclick={() => pick(openCuttingRoomTab)}><span class="ic"><Icon name="scissors" size={14} /></span> {t('workbench.cutroomTab')}</button>
  {/if}
{/snippet}

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- Drop target for the whole panel; the tabs and panes inside stay the real
     interactive elements. Keyboard users reach every one of these by the +
     menu, Ctrl+T / Ctrl+P, and the chat's file cards. -->
<div class="wb" class:busy-glow={busyGlow} ondragover={onDeskDragOver} ondragleave={onDeskDragLeave} ondrop={onDeskDrop}>
  <div class="insp-tabs">
    <!-- The hairline half of จุดบนแท็บที่กำลังใช้: a lit line along the
         strip's own border, saying the row above it is where to look. Absolute
         so it lies ON the border rather than pushing it down — a strip that
         grew a pixel when the agent started working would move every tab, and
         resize the native browser window underneath. -->
    {#if busyDot}<span class="busy-strip-line" aria-hidden="true"></span>{/if}
    {#each workbench.tabs as tab (tab.id)}
      <button
        class="tab" class:active={workbench.activeId === tab.id} title={tab.name} onclick={() => activateTab(tab.id)}
        draggable={canDrag(tab)}
        ondragstart={(e) => onTabDragStart(e, tab)}
        transition:sidle
      >
        <span class="ic"><Icon name={tabIcon[tab.kind] ?? 'fileText'} size={13} /></span>
        <span class="label">{tab.name}</span>
        <!-- Breathing on the one tab the agent is working. tab.id is the id the
             engine minted (web-agent-N) and busyWork.tab is that same id come
             back on the tool event, so this is an identity check and not a
             guess. Empty tab means the engine could not say which — no dot
             anywhere, and the border light carries the message alone. -->
        {#if busyDot && busyWork.tab && tab.id === busyWork.tab}
          <span class="busy-tab-dot" aria-hidden="true"></span>
        {/if}
        <span
          class="tab-close" role="button" tabindex="0" aria-label={t('workbench.close', { name: tab.name })}
          onclick={(e) => { e.stopPropagation(); closeTab(tab) }}
          onkeydown={(e) => e.key === 'Enter' && closeTab(tab)}
        ><Icon name="x" size={12} /></span>
      </button>
    {/each}
    <div class="plus-menu-wrap">
      <button class="icobtn tiny plus-btn" aria-label={t('workbench.addTab')} data-tip={t('workbench.addTab')} onclick={() => (menuOpen = !menuOpen)}><Icon name="plus" size={14} /></button>
      {#if menuOpen}
        <div class="plus-menu" use:keepOnScreen>{@render tabChoices()}</div>
      {/if}
    </div>
  </div>
  {#if activeTab?.kind === 'browser'}
  <div class="insp-addr">
    <button class="icobtn tiny" aria-label={t('workbench.back')} data-tip={t('workbench.back')} onclick={() => browserCmd(BrowserBack)}><Icon name="arrowLeft" size={14} /></button>
    <button class="icobtn tiny" aria-label={t('workbench.forward')} data-tip={t('workbench.forward')} onclick={() => browserCmd(BrowserForward)}><Icon name="arrowRight" size={14} /></button>
    <button class="icobtn tiny" aria-label={t('workbench.reload')} data-tip={t('workbench.reload')} onclick={() => browserCmd(BrowserReload)}><Icon name="rotateCw" size={14} /></button>
    <input
      class="insp-url" placeholder={t('workbench.urlPlaceholder')} bind:value={urlDraft}
      onkeydown={(e) => e.key === 'Enter' && navigate()}
    />
    <button class="icobtn tiny" aria-label={t('workbench.go')} data-tip={t('workbench.go')} onclick={navigate}><Icon name="externalLink" size={14} /></button>
    <span class="insp-sep" aria-hidden="true"></span>
    <!-- Pointing at the page is a way of talking to the agent, not a way of
         inspecting the page, so these two stay in the open while everything
         else went into the menu: they are MODES, they carry a lit state you
         have to be able to see, and each has a keyboard shortcut. A mode you
         cannot tell is on is a mode you turn on twice. -->
    <button
      class="icobtn tiny" class:active={pagePick.tabId === activeTab?.id && pagePick.mode === 'pick'}
      aria-label={t('workbench.pick')} data-tip={`${t('workbench.pick')} · ${shortcutLabel('pickElement')}`}
      onclick={() => togglePick('pick')}
    ><Icon name="pointer" size={14} /></button>
    <button
      class="icobtn tiny" class:active={pagePick.tabId === activeTab?.id && pagePick.mode === 'draw'}
      aria-label={t('workbench.draw')} data-tip={`${t('workbench.draw')} · ${shortcutLabel('drawOnPage')}`}
      onclick={() => togglePick('draw')}
    ><Icon name="pencil" size={14} /></button>
    <span class="insp-sep" aria-hidden="true"></span>
    <!-- Everything that is not a mode: the page's size, what the panel draws
         while the agent works, the developer tools, and the way out into a
         window of its own. Four buttons and a native <select> until 8 ก.ย.,
         two of which had grown the same ↗ glyph meaning different things. -->
    <span class="more-wrap">
      <button
        class="icobtn tiny tip-r" class:active={moreOpen}
        aria-label={t('workbench.more')} data-tip={t('workbench.more')}
        aria-expanded={moreOpen} aria-haspopup="true"
        onclick={() => (moreOpen ? closeMore() : (moreOpen = true))}
      ><Icon name="ellipsisVertical" size={14} /></button>
      {#if moreOpen}
        <div class="more-menu" role="menu" aria-label={t('workbench.more')} use:keepOnScreen>
          <button class="more-item" role="menuitem" aria-expanded={moreSection === 'size'} onclick={() => toggleMoreSection('size')}>
            <Icon name="monitor" size={14} />
            <span class="more-txt">{t('workbench.viewport')}</span>
            <span class="more-val">{activeTab?.viewport?.name ?? t('workbench.viewportFill')}</span>
          </button>
          {#if moreSection === 'size'}
            <div class="more-sub">
              <button class="more-item sub" role="menuitemradio" aria-checked={!activeTab?.viewport} onclick={() => { setViewport(''); closeMore() }}>
                <span class="tick">{#if !activeTab?.viewport}<Icon name="check" size={12} />{/if}</span>
                <span class="more-txt">{t('workbench.viewportFill')}</span>
              </button>
              {#each DEVICES as d}
                <button class="more-item sub" role="menuitemradio" aria-checked={activeTab?.viewport?.name === d.name} onclick={() => { setViewport(d.name); closeMore() }}>
                  <span class="tick">{#if activeTab?.viewport?.name === d.name}<Icon name="check" size={12} />{/if}</span>
                  <span class="more-txt">{d.name}</span>
                  <span class="more-val">{d.w}×{d.h}</span>
                </button>
              {/each}
            </div>
          {/if}
          <button class="more-item" role="menuitem" aria-expanded={moreSection === 'busy'} onclick={() => toggleMoreSection('busy')}>
            <Icon name="sparkles" size={14} />
            <span class="more-txt">{t('workbench.busySignal')}</span>
          </button>
          {#if moreSection === 'busy'}
            <div class="more-sub">
              <p class="more-head">{t('workbench.busySignalHint')}</p>
              {#each busy.layers as layer (layer.id)}
                <button
                  class="more-item sub" class:on={layer.on}
                  role="menuitemcheckbox" aria-checked={layer.on}
                  onclick={() => void toggleBusyLayer(layer.id, !layer.on)}
                >
                  <span class="tick">{#if layer.on}<Icon name="check" size={12} />{/if}</span>
                  <span class="more-txt"><b>{layer.label}</b><span class="more-note">{layer.note}</span></span>
                </button>
              {/each}
            </div>
          {/if}
          <span class="more-sep" aria-hidden="true"></span>
          <button class="more-item" role="menuitem" onclick={() => void menuCmd(BrowserOpenDevTools)}>
            <Icon name="wrench" size={14} />
            <span class="more-txt">{t('workbench.devtools')}</span>
          </button>
          <button class="more-item" role="menuitem" onclick={() => void menuCmd(detachTabInOwnWindow)}>
            <Icon name="externalLink" size={14} />
            <span class="more-txt">{t('workbench.detach')}</span>
          </button>
        </div>
      {/if}
    </span>
  </div>
  <!-- แถบบอกการกระทำ: what is being done, and to what, in words.
       Words only. The first draft had a small spinner turning inside it and the
       owner took it straight back out — *"ฝากเอาอนิเมชั่นออกเลย ไม่จำเป็น"*.
       The border light already says "still going" and says it better; a second
       thing turning next to the sentence is decoration competing with the one
       part of this panel that has to be read.
       So the dot is a dot: lit while a call is in flight, out between them. -->
  {#if busyBar}
    <div class="busy-bar" class:live={busyWork.running} role="status" aria-live="polite">
      <span class="busy-bar-dot" aria-hidden="true"></span>
      <span class="busy-bar-txt">{busyText}</span>
    </div>
  {/if}
  {/if}

  <div class="insp-body">
    {#if workbench.tabs.length === 0}
      <div class="insp-start">{@render tabChoices()}</div>
    {/if}
    {#each workbench.tabs as tab (tab.id)}
      <!-- A terminal's slot must never scroll (.term-host): xterm scrolls its
           own scrollback, and a slot scrollbar is not just redundant — it is
           the fuel of a resize feedback loop. The bar appearing steals ~15px
           of width, the pane's ResizeObserver refits, the PTY resize makes
           ConPTY replay its whole screen, the replay nudges content height,
           the bar leaves, and around again — every lap smearing a copy of the
           screen into the pane. An engine boot under that loop painted
           hundreds of half-shifted duplicate lines while the log file it
           tailed stayed byte-for-byte clean (2026-08-12, twice in one
           morning). -->
      <div class="insp-slot" class:term-host={tab.kind === 'terminal'} style="display:{workbench.activeId === tab.id ? 'block' : 'none'}">
        {#if tab.kind === 'terminal'}
          <Terminal sessionId={tab.id} onExit={() => removeTab(tab.id)} />
        {:else if tab.kind === 'files'}
          <FilesPane />
        {:else if tab.kind === 'decks'}
          <DeckRoom />
        {:else if tab.kind === 'pr'}
          <PRPane />
        {:else if tab.kind === 'git'}
          <GitPane />
        {:else if tab.kind === 'repomap'}
          <RepoMapPane />
        {:else if tab.kind === 'cutroom'}
          <CuttingRoom />
        {:else if tab.kind === 'file'}
          <!-- Keyed on rev so a re-read actually lands on screen: FileEditor
               copies `content` into its own state once and this pane never
               unmounts on its own, so without the key a file the agent had
               rewritten would keep showing the previous turn's bytes. Nothing
               renders until the first read resolves, rather than flashing an
               empty editor at every open. -->
          {#key tab.rev}
            {#if tab.view === 'image'}
              <ImagePane src={fileURL(tab.path ?? '')} name={tab.name} path={tab.path ?? ''} />
            {:else if tab.view === 'video' || tab.view === 'audio'}
              <MediaPane path={tab.path ?? ''} name={tab.name} kind={tab.view} />
            {:else if tab.view === 'pdf'}
              <PdfPane path={tab.path ?? ''} name={tab.name} />
            {:else if tab.sheet}
              <SheetPane path={tab.path ?? ''} preview={tab.sheet} />
            {:else if tab.unreadable}
              <ExternalFilePane path={tab.path ?? ''} reason={tab.unreadable} name={tab.name} />
            {:else if tab.deck && tab.content !== undefined}
              <SlidesPane path={tab.path ?? ''} name={tab.name} content={tab.content} active={workbench.activeId === tab.id} />
            {:else if tab.content !== undefined}
              <FileEditor path={tab.path ?? ''} content={tab.content} />
            {/if}
          {/key}
        {:else}
          <BrowserPane tab={tab} active={workbench.activeId === tab.id} menuOpen={panelMenuOpen} dragging={dragging} />
        {/if}
      </div>
    {/each}
    <!-- โต๊ะเงา: background chats' live pages (workbench.foreign). Each pane is
         mounted so its native window exists — the agent that owns it is still
         browsing — but never shown here: active={false} keeps the window
         hidden, and display:none keeps the slot out of this desk's layout.
         The tab itself is on its own session's saved desk, waiting there. -->
    {#each workbench.foreign as tab (tab.id)}
      <div class="foreign-host" aria-hidden="true">
        <BrowserPane tab={tab} active={false} menuOpen={false} dragging={false} />
      </div>
    {/each}
  </div>

  {#if hasActiveTask}
    <div class="insp-foot">
      <button class="stopbtn"><Icon name="square" size={14} /> {t('workbench.stopTask')}</button>
    </div>
  {/if}

  {#if dragging}
    <!-- Up from the moment something is picked up anywhere in the app, not just
         once it is over the panel: the point is to answer "can I put this here?"
         while the user is still deciding where to aim. -->
    <div class="wb-drop" class:over={overDesk}>
      <div class="wb-drop-card">
        <Icon name="plus" size={20} />
        <span class="wb-drop-title">{t('workbench.dropHere')}</span>
        <span class="wb-drop-sub">{t('workbench.dropHint')}</span>
      </div>
    </div>
  {/if}
</div>

<style>
  /* .inspector is the flex column; this fills it and gives the overlay below
     something to be absolute against. */
  .wb { flex: 1; min-height: 0; display: flex; flex-direction: column; position: relative; }

  /* A shadow host has no box on purpose: BrowserPane reads a zero rect, keeps
     the native window hidden, and this desk's layout never learns it exists. */
  .foreign-host { display: none; }

  .wb-drop {
    position: absolute;
    inset: 6px;
    z-index: 5;
    pointer-events: none; /* the drag events belong to .wb underneath */
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px dashed var(--border-default);
    border-radius: var(--r-md);
    background: color-mix(in srgb, var(--surface-sunken) 82%, transparent);
    opacity: 0.75;
    transition: opacity 120ms ease, border-color 120ms ease;
  }
  .wb-drop.over { opacity: 1; border-color: var(--accent); }

  .wb-drop-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    text-align: center;
    padding: 0 24px;
    color: var(--text-secondary);
  }
  .wb-drop.over .wb-drop-card { color: var(--text-primary); }
  .wb-drop-title { font-size: var(--fs-md); }
  .wb-drop-sub { font-size: var(--fs-xs); color: var(--text-dim); max-width: 30ch; }
</style>
