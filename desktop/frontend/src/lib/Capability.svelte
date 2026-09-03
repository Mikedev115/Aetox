<script lang="ts">
  // ความสามารถ: what the assistant is made of, and what it could be made of.
  //
  // **The room exists to announce, not to organise.** The owner's problem was
  // that a person using Aetox had no way to find out it speaks MCP at all — the
  // register sat three levels inside Settings and the word appeared nowhere a
  // reader would pass. Three settings pages (เครื่องมือ, MCP servers, สกิล) were
  // between them one answer to one question, cut into thirds, each one having to
  // point at the other two: `settings.skillsDesc` literally ends by saying the
  // tools are on the page before it.
  //
  // So the announcement is done by three things on this page, and NOT by the
  // word in the nav — "ความสามารถ" tells a newcomer less than "โปรเจกต์" does:
  //
  //   1. **The lede.** One sentence naming MCP and สกิล as prose, in the first
  //      thing the eye lands on. A tab label has to be clicked to be read; a
  //      sentence does not.
  //   2. **ชั้นวาง before ของคุณ.** The room opens on the servers you do NOT
  //      have. A register opening on your own empty list is how the old page
  //      failed — it could only ever confirm what you already knew.
  //   3. **Each card says what Aetox is missing**, not what the server has
  //      (`why` in mcpShelf.ts). Claude's Customize page carries an install
  //      count for the same job; Aetox has no registry to count anything, and a
  //      hand-written sentence about the gap is the better answer at eight
  //      entries anyway.
  //
  // **What this room does NOT do: edit.** Adding from the shelf happens here
  // because that is the announcement completing itself. Everything after that —
  // the form, the token, the per-desk targets, removing a server — stays in
  // Settings' register, which already has all of it. A second editor for one
  // config file is two places to fix a bug in, and the room does not need one to
  // do its job.
  import { onMount } from 'svelte'
  import {
    ListMCPServers, SaveMCPServer, ListExternalSkills, ListTools,
    StartMCPSignIn, CompleteMCPSignIn, CancelMCPSignIn, MCPSignInStatus,
  } from '../../wailsjs/go/main/App'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
  import { setActiveView, openSettingsAt, startChatWith } from './stores/cockpit.svelte'
  import { t, type TKey } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import { MCP_PRESETS, needsPaste, presetConfig, type MCPPreset } from './mcpShelf'

  let { onClose }: { onClose: () => void } = $props()

  type Kind = 'mcp' | 'skills' | 'builtin'
  type View = 'shelf' | 'mine'

  let kind = $state<Kind>('mcp')
  // Which half of a register is showing. One variable for both tabs rather than
  // one each: the choice a person makes here is "am I browsing or auditing",
  // and it is the same question on both, so carrying it across is what they
  // mean by having answered it.
  let view = $state<View>('shelf')

  type MCPRow = { name: string; disabled: boolean; status: string; tools: number; for?: string[] }
  type SkillRow = { name: string; description: string; dir: string; bundled?: boolean }
  type ToolRow = { name: string; description: string; source: string; category: string }

  let servers = $state<MCPRow[]>([])
  let skills = $state<SkillRow[]>([])
  let tools = $state<ToolRow[]>([])
  let busy = $state('')
  let error = $state('')
  // The preset an OAuth sign-in is in flight for, and the URL it is
  // waiting on, or null. Only one at a time — same reasoning as
  // Settings.svelte's signInPrompt: a second click means giving up on the
  // first attempt, not wanting two browser tabs racing for one redirect.
  let signInPreset = $state<MCPPreset | null>(null)
  let signInURL = $state('')
  // Nothing is drawn from an empty list until the engine has actually answered:
  // a shelf that flashes "you have nothing" and then fills in has told the
  // reader something untrue, and this room's whole job is the first impression.
  let loaded = $state(false)

  // Same order and the same keys as the register in Settings, read off Go's own
  // grouping (internal/skill/category.go) rather than restated — the grouping
  // the user sees and the one the engine knows have to be one list.
  const TOOL_CATEGORIES = ['files', 'shell', 'deliverables', 'media', 'web', 'code', 'agent'] as const

  const toolGroups = $derived(
    TOOL_CATEGORIES
      .map((key) => ({ key, items: tools.filter((s) => (s.category || 'agent') === key) }))
      .filter((g) => g.items.length > 0),
  )

  // The headline number. Every tool the assistant can actually run, whatever it
  // came from — which is the only count that answers "what can it do", and the
  // reason this page does not print three separate ones.
  const toolTotal = $derived(tools.length)
  const taken = (name: string) => servers.some((s) => s.name.toLowerCase() === name.toLowerCase())
  const targetsOf = (s: MCPRow) => s.for ?? []

  async function load() {
    const [m, k, tl] = await Promise.all([ListMCPServers(), ListExternalSkills(), ListTools()])
    servers = m as MCPRow[]
    skills = k as SkillRow[]
    tools = tl as ToolRow[]
    loaded = true
  }

  onMount(load)

  // Add from the shelf.
  //
  // A preset whose header still wants a pasted token cannot be finished in one
  // press, so it does not pretend to be: it hands the user to the register's
  // form, which already knows how to open with the header names filled in.
  // Saving a server here that could never connect is the failure the shelf was
  // built to stop making.
  //
  // An oauth preset is the same failure with a different fix: its header
  // (`${connect:name}`) already reads exactly like a one-click preset's, so
  // needsPaste alone would wave it through and save a server with nothing
  // behind that header yet. addOAuth is what actually earns the one-click
  // promise for it — checking first whether a sign-in already happened.
  async function add(p: MCPPreset) {
    if (p.oauth) {
      await addOAuth(p)
      return
    }
    if (p.headers?.length && needsPaste(p.headers)) {
      openSettingsAt('mcp')
      return
    }
    busy = p.name
    error = ''
    try {
      await SaveMCPServer('', await presetConfig(p))
      await load()
    } catch (err) {
      error = String(err)
    } finally {
      busy = ''
    }
  }

  // The three-state sign-in this drives (not started / waiting / signed in)
  // mirrors Settings.svelte's "browser kind" AI-provider flow exactly —
  // BrowserOpenURL hands the authorize page to the system browser rather
  // than navigating the webview, and this call blocks until the redirect
  // lands or abandonSignIn cancels it. Already-signed-in is the fast path:
  // the credential is already in the store from a previous visit, so there
  // is nothing to wait for and the server saves immediately.
  async function addOAuth(p: MCPPreset) {
    busy = p.name
    error = ''
    try {
      const status = await MCPSignInStatus(p.name)
      if (!status.signed_in) {
        signInPreset = p
        const prompt = await StartMCPSignIn(p.name, p.url ?? '')
        signInURL = prompt.url
        if (prompt.url) BrowserOpenURL(prompt.url)
        await CompleteMCPSignIn(p.name)
        signInPreset = null
      }
      await SaveMCPServer('', await presetConfig(p))
      await load()
    } catch (err) {
      error = String(err)
      signInPreset = null
    } finally {
      busy = ''
    }
  }

  // Re-opens the same authorize page — for when the browser tab was closed
  // by accident, or the user wants to look again before approving. The
  // already-captured URL, not a second StartMCPSignIn: that would register a
  // second client and cancel the first pending sign-in out from under the
  // addOAuth call still blocked waiting on it.
  function reopenSignIn() {
    if (signInURL) BrowserOpenURL(signInURL)
  }

  async function abandonSignIn(p: MCPPreset) {
    await CancelMCPSignIn(p.name)
    signInPreset = null
    signInURL = ''
    busy = ''
  }

  // The two doors out of this room, and they are doors rather than copies of
  // what is behind them (see the header note).
  const toRegister = (section: string) => openSettingsAt(section)

  async function ask(promptKey: TKey) {
    onClose()
    await startChatWith(t(promptKey))
  }

  function toRoster() {
    setActiveView('office')
  }

  const KINDS: { id: Kind; label: TKey }[] = [
    { id: 'mcp', label: 'capability.tabMcp' },
    { id: 'skills', label: 'capability.tabSkills' },
    { id: 'builtin', label: 'capability.tabBuiltin' },
  ]
</script>

<div class="page-shell">
  <header class="page-head">
    <button class="settings-back" onclick={onClose}><Icon name="arrowLeft" size={14} /> {t('settings.backToApp')}</button>
    <div class="page-title">
      <h2>{t('desk.capability')}</h2>
      <!-- The announcement. It says MCP and สกิล out loud, in a sentence, above
           everything else on the page — which is the whole reason the room was
           built (header note, point 1). -->
      <p>{t('capability.intro', { n: String(toolTotal) })}</p>
    </div>
  </header>

  <div class="page-body">
    <div class="settings-inner">

      <div class="cap-bar">
        <div class="seg" role="tablist" aria-label={t('capability.kindsLabel')}>
          {#each KINDS as k (k.id)}
            <button type="button" role="tab" aria-selected={kind === k.id}
              class:on={kind === k.id} onclick={() => (kind = k.id)}>{t(k.label)}</button>
          {/each}
        </div>
        <!-- เครื่องมือในตัว is an inventory, not a register: there is no shelf of
             built-in tools to browse, so the pair steps out of the way rather
             than offering a choice where one half leads nowhere. -->
        {#if kind !== 'builtin'}
          <div class="seg" role="tablist" aria-label={t('capability.viewsLabel')}>
            <button type="button" role="tab" aria-selected={view === 'shelf'}
              class:on={view === 'shelf'} onclick={() => (view = 'shelf')}>{t('capability.shelf')}</button>
            <button type="button" role="tab" aria-selected={view === 'mine'}
              class:on={view === 'mine'} onclick={() => (view = 'mine')}>{t('capability.mine')}</button>
          </div>
        {/if}
      </div>

      {#if error}
        <div class="cap-error">{error}</div>
      {/if}

      <!-- ================= MCP ================= -->
      {#if kind === 'mcp' && view === 'shelf'}
        <p class="cap-note">{t('capability.mcpShelfNote')}</p>
        <div class="cap-grid">
          {#each MCP_PRESETS as p (p.name)}
            {@const have = taken(p.name)}
            <article class="cap-card" class:have>
              <div class="cap-card-head">
                <span class="cap-id">{p.name}</span>
                {#if p.headers?.length && needsPaste(p.headers)}
                  <span class="cap-tag">{t('capability.needsKey')}</span>
                {/if}
                {#if p.oauth}
                  <span class="cap-tag">{t('capability.needsSignIn')}</span>
                {/if}
                {#if !p.url}
                  <span class="cap-tag local">{t('capability.onThisMachine')}</span>
                {/if}
                <span class="cap-grow"></span>
                {#if have}
                  <span class="cap-have"><Icon name="check" size={13} /> {t('capability.added')}</span>
                {:else if signInPreset?.name === p.name}
                  <div class="cap-signin">
                    <span class="cap-signin-wait">{t('settings.signInWaiting')}</span>
                    <button class="linkish" onclick={reopenSignIn}>{t('settings.signInOpenPage')}</button>
                    <button class="ctrl" onclick={() => abandonSignIn(p)}>{t('settings.signInCancel')}</button>
                  </div>
                {:else}
                  <button class="ctrl" disabled={busy === p.name} onclick={() => add(p)}>
                    {busy === p.name ? t('capability.adding') : t('capability.add')}
                  </button>
                {/if}
              </div>
              <!-- `why`, not `desc`: what Aetox is missing, which is the question
                   a person reading a shelf actually has. -->
              <p class="cap-why">{p.why}</p>
            </article>
          {/each}
        </div>
        <p class="cap-foot">{t('capability.mcpShelfFoot')}</p>
      {/if}

      {#if kind === 'mcp' && view === 'mine'}
        {#if loaded && servers.length === 0}
          <div class="cap-empty">
            <h3>{t('capability.noServers')}</h3>
            <p>{t('capability.noServersBody')}</p>
            <div class="cap-acts">
              <button class="ctrl ctrl-primary" onclick={() => (view = 'shelf')}>{t('capability.goShelf')}</button>
            </div>
          </div>
        {:else}
          <div class="cap-rows">
            <div class="cap-rhead">
              <span>{t('capability.colServer')}</span>
              <span>{t('capability.colFor')}</span>
              <span class="num">{t('capability.colTools')}</span>
            </div>
            {#each servers as s (s.name)}
              <div class="cap-row">
                <span class="cap-who">
                  <span class="dot" class:up={!s.disabled && s.status === 'ok'} class:off={s.disabled}></span>
                  <span class="cap-id">{s.name}</span>
                </span>
                <span class="cap-chips">
                  {#each targetsOf(s) as id (id)}<span class="cap-chip">{id}</span>{:else}
                    <span class="cap-chip none">{t('settings.mcpForNobody')}</span>
                  {/each}
                </span>
                <span class="num">{s.tools}</span>
              </div>
            {/each}
          </div>
          <p class="cap-foot">
            {t('capability.registerNote')}
            <button class="linkish" onclick={() => toRegister('mcp')}>{t('capability.openRegister')}</button>
          </p>
        {/if}
      {/if}

      <!-- ================= สกิล ================= -->
      {#if kind === 'skills' && view === 'shelf'}
        <!-- The honest empty shelf. MCP has eight curated entries because
             mcpShelf.ts was written; skills have no such list, so this tab says
             that plainly instead of drawing one it does not have. Filling it is
             a curation job — pick the skills Aetox is good at and write a line
             each — not an engineering one, and inventing entries to make the tab
             look finished would be the shelf breaking its own promise. -->
        <div class="cap-empty">
          <h3>{t('capability.noSkillShelf')}</h3>
          <p>{t('capability.noSkillShelfBody')}</p>
          <div class="cap-acts">
            <button class="ctrl ctrl-primary" onclick={() => ask('settings.aiFindSkillPrompt')}>
              <Icon name="messageSquare" size={13} /> {t('settings.aiFindSkillTitle')}
            </button>
            <button class="ctrl" onclick={() => toRegister('skills')}>{t('settings.skillInstall')}</button>
          </div>
        </div>
      {/if}

      {#if kind === 'skills' && view === 'mine'}
        {#if loaded && skills.length === 0}
          <div class="cap-empty">
            <h3>{t('settings.noSkills')}</h3>
            <p>{t('capability.noSkillsBody')}</p>
            <div class="cap-acts">
              <button class="ctrl" onclick={() => toRegister('skills')}>{t('settings.skillInstall')}</button>
            </div>
          </div>
        {:else}
          <div class="cap-grid">
            {#each skills as s (s.name)}
              <article class="cap-card">
                <div class="cap-card-head">
                  <span class="cap-id">{s.name}</span>
                  {#if s.bundled}<span class="cap-tag">{t('settings.skillBundled')}</span>{/if}
                </div>
                <p class="cap-why">{s.description || '—'}</p>
              </article>
            {/each}
          </div>
          <p class="cap-foot">
            {t('capability.registerNote')}
            <button class="linkish" onclick={() => toRegister('skills')}>{t('capability.openSkills')}</button>
          </p>
        {/if}
      {/if}

      <!-- ================= เครื่องมือในตัว ================= -->
      {#if kind === 'builtin'}
        <p class="cap-note">{t('capability.builtinNote')}</p>
        <div class="cap-inv">
          {#each toolGroups as g (g.key)}
            <div class="cap-cell">
              <span class="g">{t(('settings.toolGroup_' + g.key) as TKey)}</span>
              <span class="num">{g.items.length}</span>
            </div>
          {/each}
        </div>
        <p class="cap-foot">
          {t('capability.registerNote')}
          <button class="linkish" onclick={() => toRegister('tools')}>{t('capability.openTools')}</button>
        </p>
      {/if}

      <!-- The two things this room deliberately does not hold, said once at the
           bottom where a reader who is looking for them will be (desks.ts). -->
      <div class="cap-elsewhere">
        <p>
          {t('capability.agentsElsewhere')}
          <button class="linkish" onclick={toRoster}>{t('settings.teamOpenPage')}</button>
        </p>
        <p class="dim">{t('capability.subagentsNote')}</p>
      </div>

    </div>
  </div>
</div>

<style>
  .cap-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    padding-bottom: 14px;
    border-bottom: 1px solid var(--border-subtle);
    margin-bottom: 16px;
  }
  .seg {
    display: flex;
    gap: 2px;
    background: var(--surface-sunken);
    border: 1px solid var(--border-subtle);
    border-radius: 9px;
    padding: 3px;
  }
  .seg button {
    border: 0;
    background: none;
    color: var(--text-muted);
    font: inherit;
    font-size: 13px;
    padding: 5px 13px;
    border-radius: 6px;
    cursor: pointer;
  }
  .seg button:hover { color: var(--text-primary); }
  .seg button.on {
    background: var(--surface-raised);
    color: var(--text-primary);
    font-weight: 500;
  }
  .seg button:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 1px; }

  .cap-note {
    margin: 0 0 12px;
    color: var(--text-secondary);
    font-size: 13px;
  }
  .cap-foot {
    margin: 14px 0 0;
    color: var(--text-muted);
    font-size: 12.5px;
  }
  .cap-error {
    margin-bottom: 12px;
    padding: 9px 12px;
    border-radius: 8px;
    background: var(--status-danger-bg);
    color: var(--status-danger);
    font-size: 12.5px;
  }

  .cap-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(330px, 1fr));
    gap: 10px;
  }
  .cap-card {
    display: flex;
    flex-direction: column;
    gap: 7px;
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: 10px;
    padding: 13px 14px;
  }
  .cap-card:hover { border-color: var(--border-default); }
  .cap-card.have { opacity: .72; }
  .cap-card-head {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .cap-grow { flex: 1; }
  .cap-id {
    font-family: var(--font-mono, monospace);
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
    overflow-wrap: anywhere;
  }
  .cap-why {
    margin: 0;
    color: var(--text-muted);
    font-size: 12.5px;
    line-height: 1.55;
  }
  .cap-have {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    color: var(--status-success);
    font-size: 12.5px;
  }
  .cap-tag {
    font-size: 10.5px;
    letter-spacing: .04em;
    text-transform: uppercase;
    padding: 1px 6px;
    border-radius: 4px;
    border: 1px solid var(--border-default);
    color: var(--text-muted);
    background: var(--surface-sunken);
  }
  .cap-tag.local {
    color: var(--status-warn);
    border-color: var(--status-warn);
    background: transparent;
  }

  .cap-signin {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .cap-signin-wait {
    font-size: 12px;
    color: var(--text-muted);
  }

  .cap-rows {
    border: 1px solid var(--border-subtle);
    border-radius: 10px;
    overflow: hidden;
  }
  .cap-rhead,
  .cap-row {
    display: grid;
    grid-template-columns: minmax(0, 1.4fr) minmax(0, 2fr) 70px;
    gap: 12px;
    align-items: center;
    padding: 10px 14px;
  }
  .cap-rhead {
    background: var(--surface-sunken);
    border-bottom: 1px solid var(--border-subtle);
    font-size: 11px;
    letter-spacing: .08em;
    text-transform: uppercase;
    color: var(--text-dim);
  }
  .cap-row + .cap-row { border-top: 1px solid var(--border-subtle); }
  .cap-row:hover { background: var(--surface-row-hover); }
  .cap-who { display: flex; align-items: center; gap: 8px; min-width: 0; }
  .dot {
    width: 7px; height: 7px; border-radius: 50%; flex: none;
    background: var(--status-warn);
  }
  .dot.up { background: var(--status-success); }
  .dot.off { background: var(--text-dim); }
  .cap-chips { display: flex; gap: 4px; flex-wrap: wrap; }
  .cap-chip {
    font-size: 11px;
    color: var(--text-secondary);
    background: var(--surface-raised);
    border: 1px solid var(--border-subtle);
    border-radius: 5px;
    padding: 0 6px;
    line-height: 18px;
  }
  .cap-chip.none { color: var(--text-dim); background: transparent; border-style: dashed; }
  .num {
    font-variant-numeric: tabular-nums;
    font-size: 12.5px;
    color: var(--text-secondary);
    text-align: right;
  }

  .cap-empty {
    border: 1px dashed var(--border-default);
    border-radius: 10px;
    padding: 24px 22px;
    display: flex;
    flex-direction: column;
    gap: 9px;
    background: var(--surface-sunken);
  }
  .cap-empty h3 { margin: 0; font-size: 15px; font-weight: 600; color: var(--text-primary); }
  .cap-empty p { margin: 0; max-width: 62ch; color: var(--text-muted); font-size: 13px; }
  .cap-acts { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 4px; }

  .cap-inv {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 8px;
  }
  .cap-cell {
    display: flex;
    align-items: center;
    gap: 10px;
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: 9px;
    padding: 11px 13px;
  }
  .cap-cell .g { flex: 1; font-size: 13px; color: var(--text-primary); }

  .cap-elsewhere {
    margin-top: 26px;
    padding-top: 16px;
    border-top: 1px solid var(--border-subtle);
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .cap-elsewhere p { margin: 0; font-size: 12.5px; color: var(--text-secondary); }
  .cap-elsewhere p.dim { color: var(--text-muted); }

  .linkish {
    border: 0;
    background: none;
    padding: 0;
    font: inherit;
    color: var(--accent);
    cursor: pointer;
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .linkish:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 2px; }
</style>
