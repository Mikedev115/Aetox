<script lang="ts">
  import {
    cockpit, openFolder, openProject, searchSessions, selectSession,
    deleteSession, exportChat, forgetProject, saveProjectMeta, setActiveView,
    sessionAsking, sessionUnread, sessionWorking,
  } from './stores/cockpit.svelte'
  import {
    CodeProjectsDir, CreateCodeProject, PickCodeProjectsDir,
  } from '../../wailsjs/go/main/App'
  import type { RecentProject, Session } from './types'
  import { t } from './i18n.svelte'
  import { errText } from './errText'
  import Icon from './Icon.svelte'

  let { pinnedChats }: { pinnedChats: Record<string, boolean> } = $props()

  let pickerOpen = $state(false)
  let allProjects = $state(false)
  let projectQuery = $state('')
  let sessionQuery = $state('')
  let searchTimer: ReturnType<typeof setTimeout> | undefined

  let editing = $state('')
  let nameDraft = $state('')
  let descriptionDraft = $state('')
  let metaBusy = $state(false)
  let metaError = $state('')

  let creating = $state(false)
  let createName = $state('')
  let createBusy = $state(false)
  let createError = $state('')
  let projectsDir = $state('')

  let confirmDelete = $state('')
  let projectMenu = $state('')
  let sessionMenu = $state('')
  let sessionMenuUp = $state('')
  let forgetConfirm = $state('')

  const activeProject = $derived(cockpit.projects.find((p) => p.active)
    ?? cockpit.projects.find((p) => p.path === cockpit.project.path))
  const filteredProjects = $derived(cockpit.projects.filter((p) => {
    const q = projectQuery.trim().toLocaleLowerCase()
    return !q || `${p.name} ${p.folder ?? ''} ${p.description ?? ''} ${p.path}`.toLocaleLowerCase().includes(q)
  }))
  const shownProjects = $derived(allProjects ? filteredProjects : filteredProjects.slice(0, 3))
  // A pin is useful only when it changes where the row lives. Keep each half
  // in the engine's existing order, while lifting pinned sessions above the
  // rest of the currently focused project's list.
  const shownSessions = $derived([
    ...cockpit.sessions.filter((session) => pinnedChats[session.id]),
    ...cockpit.sessions.filter((session) => !pinnedChats[session.id]),
  ])

  $effect(() => {
    if (!cockpit.project.focused) pickerOpen = true
  })

  function showPicker(showAll = false): void {
    pickerOpen = true
    allProjects = showAll
    projectQuery = ''
    editing = ''
    metaError = ''
  }

  async function choose(project: RecentProject): Promise<void> {
    sessionQuery = ''
    metaError = ''
    setActiveView('chat')
    await openProject(project.path)
    if (cockpit.project.focused && cockpit.project.path === project.path) {
      pickerOpen = false
      allProjects = false
    }
  }

  async function newChat(): Promise<void> {
    if (!cockpit.project.path) return
    setActiveView('chat')
    await openProject(cockpit.project.path)
  }

  function searchProjectSessions(): void {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => void searchSessions(sessionQuery), 180)
  }

  function edit(project: RecentProject): void {
    editing = project.key
    nameDraft = project.name
    descriptionDraft = project.description ?? ''
    metaError = ''
  }

  async function save(project: RecentProject): Promise<void> {
    if (metaBusy) return
    metaBusy = true
    metaError = ''
    try {
      await saveProjectMeta(project.path, nameDraft, descriptionDraft)
      editing = ''
    } catch (error) {
      metaError = errText(error)
    }
    metaBusy = false
  }

  async function removeProject(project: RecentProject): Promise<void> {
    if (forgetConfirm !== project.key) {
      forgetConfirm = project.key
      return
    }
    projectMenu = ''
    forgetConfirm = ''
    await forgetProject(project.path)
  }

  async function toggleCreate(): Promise<void> {
    creating = !creating
    createError = ''
    if (creating && !projectsDir) {
      try { projectsDir = await CodeProjectsDir() } catch (error) { createError = errText(error) }
    }
  }

  async function pickProjectsDir(): Promise<void> {
    try { projectsDir = await PickCodeProjectsDir() } catch (error) { createError = errText(error) }
  }

  async function createProject(): Promise<void> {
    const name = createName.trim()
    if (!name || createBusy) return
    createBusy = true
    createError = ''
    try {
      const path = await CreateCodeProject(name)
      creating = false
      createName = ''
      setActiveView('chat')
      await openProject(path)
      pickerOpen = false
      allProjects = false
    } catch (error) {
      createError = errText(error)
    }
    createBusy = false
  }

  async function removeSession(session: Session): Promise<void> {
    if (confirmDelete !== session.id) {
      confirmDelete = session.id
      return
    }
    confirmDelete = ''
    sessionMenu = ''
    await deleteSession(session)
  }

  function exportSession(session: Session, format: 'markdown' | 'json'): void {
    sessionMenu = ''
    void exportChat(session, format)
  }

  function toggleSessionPin(session: Session): void {
    pinnedChats[session.id] = !pinnedChats[session.id]
  }

  function toggleSessionMenu(event: MouseEvent, session: Session): void {
    if (sessionMenu === session.id) {
      sessionMenu = ''
      sessionMenuUp = ''
      return
    }
    const trigger = event.currentTarget as HTMLElement
    const viewport = trigger.closest('.scroll')?.getBoundingClientRect()
    const rect = trigger.getBoundingClientRect()
    const roomBelow = (viewport?.bottom ?? window.innerHeight) - rect.bottom
    const roomAbove = rect.top - (viewport?.top ?? 0)
    sessionMenuUp = roomBelow < 196 && roomAbove > roomBelow ? session.id : ''
    sessionMenu = session.id
    confirmDelete = ''
  }
</script>

<div class="scroll project-focus-scroll" data-guide="sidebar.projects">
  <button type="button" class="project-switcher" class:empty={!cockpit.project.focused}
    aria-expanded={pickerOpen} onclick={() => pickerOpen ? (pickerOpen = false) : showPicker(false)}>
    <span class="project-switcher-icon"><Icon name={cockpit.project.focused ? 'folderOpen' : 'folder'} size={17} /></span>
    <span class="project-switcher-copy">
      <span class="project-switcher-name">{activeProject?.name || t('sidebar.selectProject')}</span>
      {#if activeProject?.folder && activeProject.name !== activeProject.folder}
        <span class="project-switcher-folder">{activeProject.folder}</span>
      {/if}
    </span>
    {#if cockpit.project.focused && cockpit.project.branch}
      <span class="project-switcher-branch"><Icon name="gitBranch" size={10} />{cockpit.project.branch}</span>
    {/if}
    <Icon name={pickerOpen ? 'chevronUp' : 'chevronDown'} size={14} />
  </button>

  {#if pickerOpen}
    <section class="project-picker">
      <div class="project-picker-head">
        <strong>{allProjects ? t('sidebar.allProjects') : t('sidebar.recentProjects')}</strong>
        <span class="project-picker-actions">
          <button type="button" aria-label={t('sidebar.openExisting')} title={t('sidebar.openExisting')} onclick={openFolder}><Icon name="folder" size={13} /></button>
          <button type="button" aria-label={t('sidebar.createProject')} title={t('sidebar.createProject')} class:on={creating} onclick={toggleCreate}><Icon name="plus" size={13} /></button>
        </span>
      </div>

      {#if allProjects}
        <label class="project-search">
          <Icon name="search" size={14} />
          <input bind:value={projectQuery} placeholder={t('sidebar.searchProjects')} aria-label={t('sidebar.searchProjects')} />
        </label>
      {/if}

      {#if creating}
        <form class="proj-create project-create-card" onsubmit={(event) => { event.preventDefault(); void createProject() }}>
          <!-- svelte-ignore a11y_autofocus -->
          <input class="proj-create-name" autofocus bind:value={createName}
            placeholder={t('sidebar.createProjectName')} aria-label={t('sidebar.createProjectName')}
            onkeydown={(event) => { if (event.key === 'Escape') creating = false }} />
          <div class="proj-create-where" title={projectsDir}>
            <span class="ic"><Icon name="folder" size={11} /></span>
            <span class="p">{projectsDir}{projectsDir ? (projectsDir.includes('\\') ? '\\' : '/') : ''}<b>{createName.trim() || t('sidebar.createProjectName')}</b></span>
            <button type="button" class="proj-create-move" onclick={pickProjectsDir}>{t('sidebar.createProjectMove')}</button>
          </div>
          {#if createError}<div class="proj-create-err">{createError}</div>{/if}
          <div class="proj-create-acts">
            <button type="submit" class="proj-create-go" disabled={createBusy || !createName.trim()}>{createBusy ? t('settings.saving') : t('sidebar.createAndChat')}</button>
            <button type="button" class="proj-create-no" onclick={() => (creating = false)}>{t('settings.cancel')}</button>
          </div>
        </form>
      {/if}

      {#if metaError}<div class="proj-create-err project-meta-error">{metaError}</div>{/if}
      <div class="project-card-list">
        {#each shownProjects as project (project.key)}
          <article class="project-card" class:active={project.active} class:menu-open={projectMenu === project.key}>
            {#if editing === project.key}
              <form class="project-card-edit" onsubmit={(event) => { event.preventDefault(); void save(project) }}>
                <input bind:value={nameDraft} placeholder={t('sidebar.projectDisplayName')} aria-label={t('sidebar.projectDisplayName')} />
                <textarea rows="2" bind:value={descriptionDraft} placeholder={t('sidebar.projectDescription')} aria-label={t('sidebar.projectDescription')}></textarea>
                <div class="project-card-edit-actions">
                  <button type="submit" disabled={metaBusy}>{metaBusy ? t('settings.saving') : t('settings.save')}</button>
                  <button type="button" onclick={() => (editing = '')}>{t('settings.cancel')}</button>
                </div>
              </form>
            {:else}
              <button type="button" class="project-card-open" onclick={() => void choose(project)}>
                <span class="project-card-folder"><Icon name={project.active ? 'folderOpen' : 'folder'} size={17} /></span>
                  <span class="project-card-copy">
                    <span class="project-card-name">{project.name}</span>
                    <span class="project-card-path" title={project.path}>{project.folder || project.name}</span>
                    {#if project.description}
                      <span class="project-card-description">{project.description}</span>
                    {/if}
                    <span class="project-card-meta"><span>{t('sidebar.sessionCount', { n: project.sessions ?? 0 })}</span><span>·</span><span>{project.ago}</span></span>
                  </span>
                </button>
              <span class="project-card-tools">
                  <button type="button" class="project-card-edit-button"
                    aria-label={t('sidebar.editProject')}
                    title={project.description ? t('sidebar.editProject') : t('sidebar.projectDescriptionEmpty')}
                    onclick={() => edit(project)}>
                    <Icon name={project.description ? 'pencil' : 'plus'} size={13} />
                  </button>
                {#if allProjects}
                  <button type="button" class="project-card-menu-button" aria-label={t('sidebar.rowMenu')} title={t('sidebar.rowMenu')}
                    aria-expanded={projectMenu === project.key} onclick={() => { projectMenu = projectMenu === project.key ? '' : project.key; forgetConfirm = '' }}><Icon name="ellipsisVertical" size={13} /></button>
                  {#if projectMenu === project.key}
                    <span class="project-card-menu plus-menu" role="menu">
                      <button type="button" class="plus-menu-item danger" role="menuitem" title={t('sidebar.forgetProjectNote')} onclick={() => void removeProject(project)}>
                        <span class="ic"><Icon name="x" size={13} /></span>{forgetConfirm === project.key ? t('sidebar.confirmDelete') : t('sidebar.forgetProject')}
                      </button>
                    </span>
                  {/if}
                {/if}
              </span>
            {/if}
          </article>
        {/each}
      </div>

      {#if shownProjects.length === 0}
        <div class="project-picker-empty">{allProjects ? t('sidebar.noProjectMatches') : t('sidebar.noProjects')}</div>
      {/if}

      {#if !allProjects}
        <button type="button" class="project-all-button" onclick={() => showPicker(true)}>
          <span>{t('sidebar.allProjects')}</span><Icon name="chevronRight" size={14} />
        </button>
      {/if}
    </section>
  {:else if cockpit.project.focused}
    <section class="project-focused">
      {#if activeProject?.description}<p class="project-focused-description">{activeProject.description}</p>{/if}
      <button type="button" class="project-new-chat" onclick={newChat}>
        <Icon name="plus" size={15} />{t('sidebar.newChat')}
      </button>
      <label class="project-session-search">
        <Icon name="search" size={14} />
        <input bind:value={sessionQuery} oninput={searchProjectSessions} placeholder={t('sidebar.searchSessions')} aria-label={t('sidebar.searchSessions')} />
      </label>
      <div class="project-session-head">{t('sidebar.sessions')}</div>
      {#if cockpit.sessionError}
        <div class="project-session-error" role="alert">
          <Icon name="alertTriangle" size={13} />
          <span>{cockpit.sessionError}</span>
        </div>
      {/if}
      <div class="project-session-list">
        {#each shownSessions as session (session.id)}
          <div class="project-session-row" class:active={session.active} class:pinned={pinnedChats[session.id]} class:menu-open={sessionMenu === session.id} class:working={sessionWorking(session)} class:unread={sessionUnread(session)} class:asking={sessionAsking(session)} class:opening={cockpit.openingSession === session.id}>
            <button type="button" class="project-session-open" aria-busy={cockpit.openingSession === session.id} onclick={() => void selectSession(session)}>
              <span class="project-session-title">{session.title}</span>
              <span class="project-session-meta">
                <span>{session.ago}</span>
                {#if cockpit.openingSession === session.id}<Icon name="loaderCircle" size={11} />
                {:else if sessionAsking(session)}<span class="dot ask" title={t('sidebar.chatAsking')}></span>
                {:else if sessionWorking(session)}<span class="dot green" title={t('sidebar.chatWorking')}></span>
                {:else if sessionUnread(session)}<span class="dot amber" title={t('sidebar.chatUnread')}></span>{/if}
              </span>
            </button>
            <span class="project-session-actions">
              <button type="button" class="project-session-menu-button" aria-label={t('sidebar.rowMenu')}
                aria-expanded={sessionMenu === session.id}
                onclick={(event) => toggleSessionMenu(event, session)}>
                <Icon name="ellipsisVertical" size={13} />
              </button>
              {#if sessionMenu === session.id}
                <span class="project-session-menu plus-menu" class:up={sessionMenuUp === session.id} role="menu">
                  <button type="button" class="plus-menu-item" role="menuitem" onclick={() => { toggleSessionPin(session); sessionMenu = '' }}>
                    <span class="ic"><Icon name="pin" size={13} /></span>
                    {pinnedChats[session.id] ? t('sidebar.unpinChat') : t('sidebar.pinChat')}
                  </button>
                  <button type="button" class="plus-menu-item" role="menuitem" onclick={() => exportSession(session, 'markdown')}>
                    <span class="ic"><Icon name="download" size={13} /></span>Markdown
                  </button>
                  <button type="button" class="plus-menu-item" role="menuitem" onclick={() => exportSession(session, 'json')}>
                    <span class="ic"><Icon name="download" size={13} /></span>JSON
                  </button>
                  <button type="button" class="plus-menu-item danger" role="menuitem" onclick={() => void removeSession(session)}>
                    <span class="ic"><Icon name="x" size={13} /></span>
                    {confirmDelete === session.id ? t('sidebar.confirmDelete') : t('sidebar.deleteSession')}
                  </button>
                </span>
              {/if}
            </span>
          </div>
        {/each}
        {#if cockpit.sessions.length === 0}
          <div class="sess-empty">{sessionQuery.trim() ? t('sidebar.noMatches') : t('sidebar.noHistory')}</div>
        {/if}
      </div>
      <button type="button" class="project-all-link" onclick={() => showPicker(true)}>
        <Icon name="arrowLeft" size={13} />{t('sidebar.allProjects')}
      </button>
    </section>
  {/if}
</div>
