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
    ListMCPServers, SaveMCPServer, ListExternalSkills, ListTools, PlacementTargets,
    StartMCPSignIn, CompleteMCPSignIn, CancelMCPSignIn, MCPSignInStatus,
  } from '../../wailsjs/go/main/App'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
  import { setActiveView, openSettingsAt, startChatWith } from './stores/cockpit.svelte'
  import { t, type TKey } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import { coverHue } from './coverHue'
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

  type TargetRow = { id: string; name: string; detail?: string; kind: string }

  let servers = $state<MCPRow[]>([])
  let skills = $state<SkillRow[]>([])
  let tools = $state<ToolRow[]>([])
  let targets = $state<TargetRow[]>([])
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

  // **The built-in tab must not count MCP tools, and it did.** Every tool a
  // server bridges in arrives with an empty category, so `|| 'agent'` swept all
  // of them into การทำงานของผู้ช่วย — 43 of them on the owner's machine, which
  // made that one bucket read 56 while the six real ones totalled 24. A tab
  // whose question is "with nothing added, what can it already do" was
  // answering it with the things that were added.
  const builtinTools = $derived(tools.filter((s) => s.source !== 'mcp'))

  const toolGroups = $derived(
    TOOL_CATEGORIES
      .map((key) => ({ key, items: builtinTools.filter((s) => (s.category || 'agent') === key) }))
      .filter((g) => g.items.length > 0),
  )

  // The headline number. Every tool the assistant can actually run, whatever it
  // came from — which is the only count that answers "what can it do", and the
  // reason this page does not print three separate ones.
  const toolTotal = $derived(tools.length)
  const taken = (name: string) => servers.some((s) => s.name.toLowerCase() === name.toLowerCase())
  const targetsOf = (s: MCPRow) => s.for ?? []

  // What a placement id is called. The chips printed the raw id — `agent:editor`,
  // `agent:deepresearch` — which is the engine's address for a thing, not its name.
  // Settings' register has always resolved these through PlacementTargets; the
  // room was showing the same rows in a different vocabulary.
  const targetName = (id: string) => targets.find((x) => x.id === id)?.name ?? id

  // A server that has not been reached yet reports `tools: 0`, and a bare 0 in
  // a column headed เครื่องมือ reads as "this server offers nothing" rather
  // than "nobody has asked it yet". The dash says the second thing, which is
  // the true one.
  //
  // Keyed off the count and NOT off `status`: gating on `status === 'ok'` hid
  // notion's real 41, because a server can have answered a tool list on a
  // previous run and still not be reading `ok` right now. A number that was
  // actually reported is a fact worth printing; only the absence of one is
  // what the dash is for.
  const toolLabel = (s: MCPRow) => (s.tools > 0 ? String(s.tools) : '—')
  const isUp = (s: MCPRow) => !s.disabled && s.status === 'ok'

  async function load() {
    const [m, k, tl, tg] = await Promise.all([
      ListMCPServers(), ListExternalSkills(), ListTools(), PlacementTargets(),
    ])
    servers = m as MCPRow[]
    skills = k as SkillRow[]
    tools = tl as ToolRow[]
    targets = tg as TargetRow[]
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

      <!-- Two controls, and they have to LOOK like two.
           Drawn as a matched pair of segments they read as one bar of five
           buttons — the owner's first look at the real page bounced off exactly
           that. They are not siblings: the first picks the subject, the second
           picks which half of that subject you are looking at, so the second is
           subordinate and is drawn that way — pushed to the far end, quieter,
           and underlined rather than boxed. -->
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
          <div class="subseg" role="tablist" aria-label={t('capability.viewsLabel')}>
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
                <!-- The mark. Not a fetched brand logo — that would mean a file
                     per server, kept alive forever, and somebody else's
                     trademark in our binary. coverHue is the app's answer to
                     "a named thing in a gallery needs a face", already used by
                     โปรเจกต์, ชุดคำสั่ง and the roster; this is the fourth
                     gallery, so it wears the same mark rather than inventing a
                     fifth idea. Same hash, so a server keeps its colour. -->
                <span class="cap-mark" style="--h:{coverHue(p.name)}" aria-hidden="true">
                  {p.name.slice(0, 2)}
                </span>
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
                  <span class="cap-mark sm" style="--h:{coverHue(s.name)}" aria-hidden="true">
                    {s.name.slice(0, 2)}
                  </span>
                  <span class="dot" class:up={isUp(s)} class:off={s.disabled}></span>
                  <span class="cap-id">{s.name}</span>
                </span>
                <span class="cap-chips">
                  {#each targetsOf(s) as id (id)}<span class="cap-chip">{targetName(id)}</span>{:else}
                    <span class="cap-chip none">{t('settings.mcpForNobody')}</span>
                  {/each}
                </span>
                <span class="num">{toolLabel(s)}</span>
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
                  <span class="cap-mark" style="--h:{coverHue(s.name)}" aria-hidden="true">
                    {s.name.replace(/^aetox-/, '').slice(0, 2)}
                  </span>
                  <span class="cap-id">{s.name}</span>
                  <!-- A LABEL, not the explanation. `settings.skillBundled` is a
                       three-clause sentence about overriding a bundled skill —
                       correct in the register where it sits under one row, and
                       absurd here, where it drew a bordered paragraph on all
                       twenty-five cards saying the same thing twenty-five
                       times. The chip says which kind of skill this is; the
                       register is where the sentence belongs. -->
                  {#if s.bundled}<span class="cap-tag">{t('capability.bundled')}</span>{/if}
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
        <!-- **Names, not counts.** This drew seven boxes with a number in each,
             and the owner's reaction to the real page was that he could not
             read it — "ขนาดผู้สร้างแม่งยังงง". He was right, and the fault was
             the form: the tab asks "with nothing added, what can it already
             do", and a count answers "how many", which is a question nobody
             walked in with. The names are short, there are thirty-seven of
             them, and reading `read` `web_fetch` `doc_write` under a heading IS
             the answer. The count stays as a quiet total per group, where it is
             context rather than the content. -->
        <p class="cap-note">{t('capability.builtinNote', { n: String(builtinTools.length) })}</p>
        <div class="cap-cats">
          {#each toolGroups as g (g.key)}
            <section class="cap-cat">
              <h3>
                {t(('settings.toolGroup_' + g.key) as TKey)}
                <span class="num">{g.items.length}</span>
              </h3>
              <div class="cap-tools">
                {#each g.items as it (it.name)}
                  <span class="cap-tool" title={it.description}>{it.name}</span>
                {/each}
              </div>
            </section>
          {/each}
        </div>
        <!-- NOT registerNote. That sentence offers editing, deleting and
             choosing who carries a server — none of which is true of a tool
             that ships inside the binary, so on this tab it was simply false. -->
        <p class="cap-foot">
          {t('capability.builtinFoot')}
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
  /* Neither segment may stretch: .cap-bar is a flex row, and a .seg allowed to
     grow left a slab of empty chrome after เครื่องมือในตัว that made the two
     controls look like one wide bar with a gap in it. */
  .seg { flex: none; }

  /* The subordinate control (see the markup note): pushed to the far end, no
     box of its own, and the choice marked by an underline. Different enough
     that the eye reads two questions rather than five buttons. */
  .subseg {
    flex: none;
    margin-left: auto;
    display: flex;
    gap: 4px;
  }
  .subseg button {
    border: 0;
    background: none;
    color: var(--text-muted);
    font: inherit;
    font-size: 13px;
    padding: 5px 4px 6px;
    border-bottom: 2px solid transparent;
    cursor: pointer;
  }
  .subseg button:hover { color: var(--text-primary); }
  .subseg button.on {
    color: var(--text-primary);
    font-weight: 500;
    border-bottom-color: var(--accent);
  }
  .subseg button:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 1px; }

  /* The mark a named thing wears, same gradient formula as .pp-cover in
     style.css so the four galleries agree — only smaller and square, because a
     104px banner on a shelf card would be the loudest thing on the page. */
  .cap-mark {
    flex: none;
    width: 26px;
    height: 26px;
    border-radius: 7px;
    display: grid;
    place-items: center;
    font-family: var(--font-mono, monospace);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: -.02em;
    text-transform: lowercase;
    color: hsl(var(--h) 60% 84%);
    background:
      radial-gradient(120% 120% at 20% 0%, hsl(var(--h) 70% 42% / .55), transparent 60%),
      linear-gradient(135deg, hsl(var(--h) 45% 22%), hsl(calc(var(--h) + 40) 45% 15%));
  }
  .cap-mark.sm { width: 20px; height: 20px; border-radius: 6px; font-size: 9.5px; }

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

  /* The inventory, as names under headings. Columns rather than one long list:
     seven short groups side by side is a page you take in at a glance, which a
     single 37-row column is not. */
  .cap-cats {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 10px;
    align-items: start;
  }
  .cap-cat {
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: 10px;
    padding: 12px 13px 13px;
  }
  .cap-cat h3 {
    margin: 0 0 9px;
    display: flex;
    align-items: baseline;
    gap: 8px;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }
  .cap-cat h3 .num { margin-left: auto; color: var(--text-dim); font-size: 12px; }
  .cap-tools { display: flex; flex-wrap: wrap; gap: 5px; }
  .cap-tool {
    font-family: var(--font-mono, monospace);
    font-size: 11.5px;
    color: var(--text-secondary);
    background: var(--surface-sunken);
    border: 1px solid var(--border-subtle);
    border-radius: 5px;
    padding: 1px 7px;
    line-height: 19px;
  }

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
