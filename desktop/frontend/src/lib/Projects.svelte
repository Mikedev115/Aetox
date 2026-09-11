<script lang="ts">
  // โปรเจกต์ (COMPANY.md §84, DECISIONS §90): chats grouped, with files that
  // ride into every session inside.
  //
  // Two levels, not two columns. The list answers "which project" and nothing
  // else, so it is cards a person scans; the detail answers "what is in this
  // one", which is the context folder and the chats. Putting both on screen at
  // once made the list a narrow sidebar — the wrong shape for the question
  // asked most often, which is finding the project you meant.
  //
  // Read live off the disk for the same reason ผลงาน is: a project exists
  // because its folder exists, and an index behind this page would start
  // disagreeing with the disk the first time the user moved one.
  //
  // There is no description on a card, deliberately. Aetox has nowhere to store
  // one — the folder is the whole record — and inventing a file convention to
  // hold a sentence would be a second source of truth for the sake of a subtitle.
  // The card carries what is already true: how many chats, how much context,
  // when it last changed.
  import { onMount } from 'svelte'
  import { Spaces, CreateSpace, DeleteSpace, OpenSpaceFolder, SessionsInSpace, AddSpaceContext, AddSpaceContextFiles, RemoveSpaceContext } from '../../wailsjs/go/main/App'
  import { main } from '../../wailsjs/go/models'
  import { agoLabel, cockpit, newSpaceSession, selectGlobalSession, sendUserMessage, sessionUnread, sessionWorking, setActiveView } from './stores/cockpit.svelte'
  import { t } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import { coverHue } from './coverHue'
  import { startersFor } from './starters'
  import { stashDraft } from './composerDraft'

  let { onClose }: { onClose: () => void } = $props()

  let projects = $state<main.Space[]>([])
  let chats = $state<main.SessionMeta[]>([])
  let openName = $state('')
  let loaded = $state(false)
  let creating = $state(false)
  let draftName = $state('')
  let query = $state('')
  let sortBy = $state<'updated' | 'name'>('updated')
  let error = $state('')
  let busy = $state('')
  let confirmFile = $state('')
  // The project a delete has been asked for, held while the dialog is up.
  let confirmProject = $state('')
  // The first line of a chat that does not exist yet (see startChat).
  let firstLine = $state('')
  // The … beside the project's name: open folder, delete. Deleting is the one
  // gesture on this page that cannot be walked back, and until 12 ก.ย. it sat
  // in the context card's footer next to เปิดโฟลเดอร์ — a card about files
  // carrying the one button that removes the project. It is about the project,
  // so it lives with the project's name, behind a menu rather than in the open.
  let menuOpen = $state(false)
  // What the project's chats open with (starters.ts): the six cards a blank
  // chat inside a project deals from. Drawn here too, because this page is
  // where a person decides what to ask, and the cards were one click further
  // in than that decision.

  const open = $derived(projects.find((p) => p.name === openName))
  const starters = $derived(open ? startersFor({ desk: 'assistant', chair: '', space: open.name }).starters : [])
  const shown = $derived(
    projects
      .filter((p) => p.name.toLowerCase().includes(query.trim().toLowerCase()))
      // The backend hands them back newest first; sorting by name is the only
      // reordering this page does, so it is the only branch here.
      .sort((a, b) => (sortBy === 'name' ? a.name.localeCompare(b.name) : 0)),
  )

  async function refresh() {
    projects = (await Spaces()) ?? []
    loaded = true
    if (openName && !projects.some((p) => p.name === openName)) openName = ''
  }

  onMount(refresh)

  async function enter(name: string) {
    openName = name
    chats = (await SessionsInSpace(name)) ?? []
  }

  async function create() {
    const name = draftName.trim()
    if (!name) return
    error = ''
    busy = 'create'
    try {
      await CreateSpace(name)
      draftName = ''
      creating = false
      await refresh()
      await enter(name)
    } catch (err) {
      error = String(err)
    }
    busy = ''
  }

  // Files are copied in, so the project keeps working when the original is
  // moved (see AddSpaceContext). The list comes back from the disk that was
  // just changed rather than being guessed at here.
  async function addFiles(name: string) {
    error = ''
    busy = 'add'
    try {
      const files = await AddSpaceContext(name)
      applyContext(name, files ?? [])
    } catch (err) {
      error = String(err)
    }
    busy = ''
  }

  // Files dropped from Explorer onto the context card. App.svelte owns the one
  // drop handler and routes here by where the drop landed; the copy is the
  // same one the picker makes.
  async function addDropped(paths: string[]) {
    if (!open || paths.length === 0) return
    error = ''
    busy = 'add'
    try {
      applyContext(open.name, (await AddSpaceContextFiles(open.name, paths)) ?? [])
    } catch (err) {
      error = String(err)
    }
    busy = ''
  }

  onMount(() => {
    const onDrop = (e: Event) => void addDropped((e as CustomEvent<string[]>).detail ?? [])
    window.addEventListener('space-context-drop', onDrop)
    return () => window.removeEventListener('space-context-drop', onDrop)
  })

  // Two-step, the same gesture ผลงาน uses: the first click arms the row, the
  // second one deletes. These are the user's own files.
  async function removeFile(name: string, file: string) {
    if (confirmFile !== file) {
      confirmFile = file
      return
    }
    confirmFile = ''
    error = ''
    try {
      applyContext(name, (await RemoveSpaceContext(name, file)) ?? [])
    } catch (err) {
      error = String(err)
    }
  }

  // The list is applied at once, from the disk that was just changed; the
  // dates come with the next full read, which is asked for rather than guessed
  // at — a copy made a moment ago is "เมื่อกี้" either way.
  function applyContext(name: string, files: string[]) {
    projects = projects.map((p) => (p.name === name ? main.Space.createFrom({ ...p, contextFiles: files }) : p))
    void refresh()
  }

  // The file's date, as the row draws it: '' when the disk did not say.
  function fileAgo(space: main.Space, file: string): string {
    const stamp = space.contextModified?.[file]
    return stamp ? agoLabel(stamp) : ''
  }

  // The assistant's last words, flattened to one line: a reply is markdown
  // with headings, lists and tables in it, and the row has one line of small
  // type. Marks go, words stay; a table's rule line is all marks.
  function oneLine(text: string): string {
    return text.replace(/[#*_`>|]+|-{2,}|:-+:?/g, ' ').replace(/\s+/g, ' ').trim()
  }

  // Two characters of the name on the swatch — the whole name is the heading
  // beside it. Code points rather than UTF-16 units so a Thai vowel mark is not
  // cut from its consonant.
  function monogram(name: string): string {
    return [...name].slice(0, 2).join('')
  }

  // Deleting a project is the one gesture here that cannot be walked back, so
  // it goes through the app's one dialog rather than the arm-then-click gesture
  // the context files use: a file can be added again from the original, a
  // folder of them cannot. What survives is said in the dialog, not discovered
  // afterwards — the chats stay, the originals stay, the copies do not.
  async function deleteProject(name: string) {
    confirmProject = ''
    error = ''
    busy = 'delete'
    try {
      await DeleteSpace(name)
      openName = ''
      chats = []
      await refresh()
    } catch (err) {
      error = String(err)
    }
    busy = ''
  }

  async function openFolder(name: string) {
    error = ''
    try {
      await OpenSpaceFolder(name)
    } catch (err) {
      error = String(err)
      await refresh()
    }
  }

  // Opening a chat inside a project is the point of the room, so it leaves the
  // room — through the store, never the binding directly. Calling
  // NewSessionInSpace here was the bug: the engine opened the new session while
  // the window kept showing the one already on screen, so the chat the click
  // had just created was unreachable and the project looked like it had
  // vanished. newSpaceSession is the door that moves both.
  //
  // With a first line typed, it is sent as the chat's first message: the box
  // on this page is a composer, not a button that looks like one (it was, until
  // 12 ก.ย., and a field that opens an empty chat when you press Enter in it
  // reads as a field that lost what you typed). Empty, it opens a blank chat,
  // which is what the button did.
  //
  // The session is opened BEFORE the view changes hands, so the composer that
  // mounts behind this page reads the session it is about to draw — the
  // starter path below files text under that id and needs it to exist first.
  async function startChat(name: string) {
    const text = firstLine.trim()
    error = ''
    busy = 'chat'
    try {
      await newSpaceSession(name)
      if (text) void sendUserMessage(text)
      firstLine = ''
      setActiveView('chat')
      onClose()
    } catch (err) {
      error = String(err)
    }
    busy = ''
  }

  // A starter card: the same new chat, with the card's prompt in the composer
  // rather than sent. Two of the six end in ": " and want the user's own words
  // after them; none is sent unread on a click, which is the rule the blank
  // chat's cards already follow (pickStarter, Chat.svelte).
  async function startWith(name: string, prompt: string) {
    error = ''
    busy = 'chat'
    try {
      await newSpaceSession(name)
      stashDraft(cockpit.openSession, prompt)
      setActiveView('chat')
      onClose()
    } catch (err) {
      error = String(err)
    }
    busy = ''
  }

  function onFirstLineKey(e: KeyboardEvent, name: string) {
    if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
      e.preventDefault()
      void startChat(name)
    }
  }

  async function openChat(chat: main.SessionMeta) {
    onClose()
    setActiveView('chat')
    await selectGlobalSession({ id: chat.id, title: chat.title, ago: '' })
  }
</script>

<svelte:window onclick={() => (menuOpen = false)} />

<div class="page-shell">
  <!-- The title is the level you are on. Inside a project the room's own name
       and blurb would be the second and third heading in a row saying nothing
       about the project you opened — so the project takes the title, and the
       breadcrumb above it is both the trail and the way back. -->
  <header class="page-head">
    {#if open}
      <!-- One line of navigation, not three. The arrow is the trail's first
           step (back to the list), the app is one level further out and gets
           the far edge and the key that already closes the page. Three ways
           back stacked over the title was the first cut, and every one of
           them was a heading about somewhere else. -->
      <nav class="proj-crumbline">
        <button class="settings-back" onclick={() => (openName = '')}><Icon name="arrowLeft" size={14} /> {t('desk.projects')}</button>
        <span class="proj-crumb-sep">/</span>
        <span class="proj-crumb-here">{open.name}</span>
        <span class="proj-crumb-grow"></span>
        <button class="proj-esc" onclick={onClose}>{t('settings.backToApp')} <kbd>Esc</kbd></button>
      </nav>
      <!-- The project's own colour, the same one its card wears on the list
           (coverHue): the page used to drop it at the door, and a name alone
           over two columns of grey was nobody's project in particular. -->
      <div class="page-title proj-title">
        <span class="proj-swatch" style="--h:{coverHue(open.name)}"><span class="pp-mono">{monogram(open.name)}</span></span>
        <div class="proj-title-text">
          <h2>{open.name}</h2>
          <p class="proj-stats">
            <span>{t('projects.chatCount', { n: chats.length })}</span>
            <span class="proj-stat-sep">·</span>
            <span>{t('projects.fileCount', { n: open.contextFiles.length })}</span>
            <span class="proj-stat-sep">·</span>
            <span>{agoLabel(open.updatedAt)}</span>
            <span class="proj-stat-sep">·</span>
            <span class="proj-stat-path" title={open.path}>{open.path}</span>
          </p>
        </div>
        <span class="row-menu-wrap">
          <button type="button" class="row-more proj-more" aria-label={t('sidebar.rowMenu')} title={t('sidebar.rowMenu')}
            aria-haspopup="menu" aria-expanded={menuOpen} onclick={(e) => { e.stopPropagation(); menuOpen = !menuOpen }}>
            <Icon name="ellipsisVertical" size={15} />
          </button>
          {#if menuOpen}
            <div class="plus-menu right" role="menu">
              <button type="button" class="plus-menu-item" role="menuitem" onclick={() => { menuOpen = false; openFolder(open.name) }}>
                <span class="ic"><Icon name="folderOpen" size={14} /></span>
                {t('projects.openFolder')}
              </button>
              <button type="button" class="plus-menu-item danger" role="menuitem" disabled={busy === 'delete'}
                onclick={() => { menuOpen = false; confirmProject = open.name }}>
                <span class="ic"><Icon name="trash" size={14} /></span>
                {t('projects.delete')}
              </button>
            </div>
          {/if}
        </span>
      </div>
    {:else}
      <button class="settings-back" onclick={onClose}><Icon name="arrowLeft" size={14} /> {t('settings.backToApp')}</button>
      <div class="page-title">
        <h2>{t('desk.projects')}</h2>
        <p>{t('projects.intro')}</p>
      </div>
    {/if}
  </header>

  <div class="page-body">
    <div class="settings-inner wide">
      {#if error}<div class="page-error">{error}</div>{/if}

      {#if open}
        <!-- One project: the work on the left, what the project carries on the
             right. The title and the trail live in the page header above. -->
        <div class="proj-detail">
          <section class="proj-work">
            <!-- A composer, one line high: the first thing said in the new
                 chat, or nothing and a blank one. Enter sends; Shift+Enter is
                 a newline like the real one. -->
            <div class="proj-composer">
              <span class="proj-composer-ic"><Icon name="sparkles" size={15} /></span>
              <!-- svelte-ignore a11y_autofocus -->
              <textarea class="proj-first" rows="1" autofocus bind:value={firstLine} placeholder={t('projects.firstLine')}
                disabled={busy === 'chat'} onkeydown={(e) => onFirstLineKey(e, open.name)}></textarea>
              <button class="send" aria-label={t('projects.newChat')} title={t('projects.newChat')}
                disabled={busy === 'chat'} onclick={() => startChat(open.name)}><Icon name="sendHorizontal" size={15} /></button>
            </div>

            <div class="proj-starters">
              {#each starters as s (s.titleKey)}
                <button class="proj-starter" disabled={busy === 'chat'} onclick={() => startWith(open.name, t(s.promptKey))}>
                  <Icon name={s.icon} size={15} />
                  <span>{t(s.titleKey)}</span>
                </button>
              {/each}
            </div>

            <div class="settings-group-label eyebrow">{t('projects.chats')}</div>
            {#if chats.length === 0}
              <div class="proj-none">{t('projects.noChats')}</div>
            {:else}
              <ul class="proj-chats">
                {#each chats as chat (chat.id)}
                  <li>
                    <!-- Same dot the sidebar draws, on the same two facts:
                         green while a turn is running in this chat, amber once
                         one has finished and nobody has opened it. The page is
                         a list of the project's conversations, so it is one of
                         the places you walk back to in order to find out.
                         Two lines: the title is the user's first sentence and
                         five of them in one project start the same way, so the
                         second line is what the assistant last said. -->
                    <button class:working={sessionWorking(chat)} class:unread={sessionUnread(chat)}
                      onclick={() => openChat(chat)}>
                      <Icon name="messageSquare" size={13} />
                      <span class="proj-chat-text">
                        <span class="proj-chat-title">{chat.title}</span>
                        {#if chat.snippet}<span class="proj-chat-snip">{oneLine(chat.snippet)}</span>{/if}
                      </span>
                      {#if sessionWorking(chat)}
                        <span class="dot green" role="img" title={t('sidebar.chatWorking')} aria-label={t('sidebar.chatWorking')}></span>
                      {:else if sessionUnread(chat)}
                        <span class="dot amber" role="img" title={t('sidebar.chatUnread')} aria-label={t('sidebar.chatUnread')}></span>
                      {/if}
                      <span class="proj-chat-ago">{agoLabel(chat.updatedAt)}</span>
                    </button>
                  </li>
                {/each}
              </ul>
            {/if}
          </section>

          <!-- The rail is one card, not three. The reference has Instructions
               and Scheduled beside Context; neither is ours to put here yet —
               recurring work is the ระบบออโตเมชั่น room (COMPANY.md §7, still
               ⏳), and a second door to a room that does not exist is a promise
               the app cannot keep.
               data-context-drop is how App.svelte's one drop handler finds the
               card: files dropped on it are copied in. -->
          <aside class="proj-rail">
            <div class="proj-rail-card" data-context-drop>
              <div class="proj-rail-head">
                <h4>{t('projects.context')}</h4>
                {#if open.contextFiles.length > 0}
                  <span class="proj-rail-count">{t('projects.fileCount', { n: open.contextFiles.length })}</span>
                {/if}
                <button class="icobtn tiny" aria-label={t('projects.addFiles')} title={t('projects.addFiles')}
                  disabled={busy === 'add'} onclick={() => addFiles(open.name)}><Icon name="plus" size={14} /></button>
              </div>

              {#if open.contextFiles.length === 0}
                <button class="proj-drop" disabled={busy === 'add'} onclick={() => addFiles(open.name)}>
                  <Icon name="fileText" size={18} />
                  <span>{t('projects.contextEmpty')}</span>
                </button>
              {:else}
                <ul class="proj-files">
                  {#each open.contextFiles as file (file)}
                    <li>
                      <Icon name="fileText" size={13} />
                      <span class="proj-file-name" title={file}>{file}</span>
                      <!-- The date steps aside for the delete on hover: they
                           share the row's end, and the one you can act on wins
                           while you are pointing at it. -->
                      <span class="proj-file-ago">{fileAgo(open, file)}</span>
                      <button
                        class="proj-file-del" class:confirm={confirmFile === file}
                        aria-label={t('projects.removeFile')}
                        onclick={() => removeFile(open.name, file)}
                      >
                        {#if confirmFile === file}{t('sidebar.confirmDelete')}{:else}<Icon name="x" size={11} />{/if}
                      </button>
                    </li>
                  {/each}
                </ul>
                <button class="proj-drop slim" disabled={busy === 'add'} onclick={() => addFiles(open.name)}>
                  <Icon name="upload" size={14} />
                  <span>{t('projects.dropHint')}</span>
                </button>
              {/if}

              <div class="proj-rail-feet">
                <button class="linkish proj-rail-foot" onclick={() => openFolder(open.name)}>
                  {t('projects.openFolder')}
                </button>
                <span class="proj-hint">{t('projects.contextHint')}</span>
              </div>
            </div>
          </aside>
        </div>
      {:else}
        <!-- The list, in the same card system the preset gallery and the roster
             use (pp-*): one visual language across every shelf in the app. A
             project has no picture to ship, so its cover is a colour derived
             from its own name — the gallery reads as a gallery on the day it is
             created, with nothing in the installer. -->
        {#if projects.length > 0}
          <div class="proj-bar">
            <label class="proj-search">
              <Icon name="search" size={13} />
              <input bind:value={query} placeholder={t('projects.search')} />
            </label>
            <div class="proj-sort">
              <span>{t('projects.sortBy')}</span>
              <select class="ctrl" bind:value={sortBy}>
                <option value="updated">{t('projects.sortUpdated')}</option>
                <option value="name">{t('projects.sortName')}</option>
              </select>
            </div>
          </div>
        {/if}

        <div class="pp-grid proj-gallery">
          {#if creating}
            <!-- Creating is a card in the same slot the new-project card sits
                 in, with the cover already wearing the colour the name will
                 keep. The first thing a person makes here should look like the
                 thing they are making, not like a form about it. -->
            <form class="pp-card proj-draft" onsubmit={(e) => { e.preventDefault(); create() }}>
              <span class="pp-cover" style="--h:{coverHue(draftName || t('desk.projects'))}">
                <span class="pp-mono">{draftName || '—'}</span>
              </span>
              <div class="pp-body">
                <!-- svelte-ignore a11y_autofocus -->
                <input
                  class="ctrl" autofocus bind:value={draftName} placeholder={t('projects.namePlaceholder')}
                  onkeydown={(e) => { if (e.key === 'Escape') { creating = false; draftName = '' } }}
                />
                <span class="pp-desc">{t('projects.createHint')}</span>
                <div class="proj-draft-actions">
                  <button class="ctrl ctrl-primary" type="submit" disabled={busy === 'create' || !draftName.trim()}>
                    {busy === 'create' ? t('settings.saving') : t('projects.createShort')}
                  </button>
                  <button class="ctrl" type="button" onclick={() => { creating = false; draftName = '' }}>
                    {t('settings.cancel')}
                  </button>
                </div>
              </div>
            </form>
          {:else}
            <button class="pp-card pp-new" onclick={() => (creating = true)}>
              <span class="pp-plus">+</span>
              <span class="pp-newtxt">{t('projects.create')}</span>
            </button>
          {/if}

          {#each shown as p (p.name)}
            <!-- The delete is a sibling of the card, not a child: the card is
                 itself a button, and a button inside a button is not markup a
                 browser will honour. The wrapper is what the two share. -->
            <div class="proj-cardwrap">
              <button class="pp-card proj-card" onclick={() => enter(p.name)} title={p.path}>
                <span class="pp-cover" style="--h:{coverHue(p.name)}">
                  <span class="pp-mono">{p.name}</span>
                </span>
                <div class="pp-body">
                  <span class="pp-title">{p.name}</span>
                  <span class="pp-desc">
                    {#if p.chats === 0 && p.contextFiles.length === 0}
                      {t('projects.cardEmpty')}
                    {:else}
                      {[
                        p.chats > 0 ? t('projects.chatCount', { n: p.chats }) : '',
                        p.contextFiles.length > 0 ? t('projects.fileCount', { n: p.contextFiles.length }) : '',
                      ].filter(Boolean).join(' · ')}
                    {/if}
                  </span>
                </div>
                <span class="proj-card-foot">{agoLabel(p.updatedAt)}</span>
              </button>
              <!-- Named per card rather than a row of identical "delete"
                   buttons: this is the label a screen reader reads out, and
                   "delete project" nine times over says nothing about which. -->
              <button
                class="proj-card-del"
                aria-label={t('projects.deleteNamed', { name: p.name })}
                title={t('projects.delete')}
                disabled={busy === 'delete'}
                onclick={() => (confirmProject = p.name)}
              ><Icon name="x" size={13} /></button>
            </div>
          {/each}
        </div>

        <!-- Only ever about the search. The first cut showed this whenever the
             grid was empty, so a brand-new install read "no project matches that
             search" before anything had been searched for. -->
        {#if loaded && query.trim() !== '' && shown.length === 0}
          <div class="proj-none">{t('projects.noMatches')}</div>
        {/if}
        {#if loaded && projects.length === 0 && !creating}
          <p class="page-note proj-first">{t('projects.empty')}</p>
        {/if}
      {/if}
    </div>
  </div>
</div>

{#if confirmProject}
  {@const name = confirmProject}
  <ConfirmDialog
    title={t('projects.confirmDeleteTitle')}
    message={t('projects.confirmDeleteMessage')}
    detail={name}
    confirmLabel={t('settings.confirmDeleteAction')}
    onConfirm={() => deleteProject(name)}
    onCancel={() => (confirmProject = '')}
  />
{/if}
