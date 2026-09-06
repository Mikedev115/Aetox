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
  //   2. **ห้องสมุด before ของคุณ.** The room opens on the servers you do NOT
  //      have. A register opening on your own empty list is how the old page
  //      failed — it could only ever confirm what you already knew.
  //      (Called ชั้นวาง until 4 ก.ย. 2026. A shelf is where a thing is put
  //      down; a library is where you go to look one up, which is what this
  //      half is for — owner's word, and the better one.)
  //   3. **Each card says what Aetox is missing**, not what the server has
  //      (`why` in mcpShelf.ts). Claude's Customize page carries an install
  //      count for the same job; Aetox has no registry to count anything, and a
  //      hand-written sentence about the gap is the better answer at eight
  //      entries anyway.
  //
  // **This room is where an MCP server lives now.** Add it from ห้องสมุด, say
  // who carries it, switch it on and off, test it, throw it away — all here.
  //
  // That reverses what this comment used to say, twice, in one day, and the
  // path is worth keeping because the second version was the wrong lesson:
  //
  //   1. "This room does NOT edit." One config file, one editor. Clean rule.
  //   2. Then the placement panel came here, and the rule was rewritten to
  //      "edits exactly one field" — which sounds principled and is not. The
  //      room was already a second editor at that point; it was just one with
  //      its arms cut off, so a person standing on a row had to hold a map of
  //      which half of the job lives on which page. The owner found it by
  //      trying to delete a server he was looking straight at.
  //   3. So: the row's actions come with the panel. What does NOT come is the
  //      FORM — address, key, env, cwd, timeout, allowed tools. Eleven fields
  //      copied is the duplication rule 1 was actually about, and none of it is
  //      what this room is for. แก้ไข opens the register and says so in the
  //      label rather than pretending.
  //
  // Nothing here is a lookalike. `.reg-head`, `.mcp-targets`, its group heading
  // and เลือกทั้งหมด, `.plus-menu` and ConfirmDialog are read out of what the
  // register already uses, and every write goes through the same binding it
  // calls. Two rows that mean the same thing must look the same and write the
  // same way, or they are two features (DESIGN.md §1, §5).
  //
  // The one part that is NOT the register's is the control inside that panel:
  // `.cap-pick` chips instead of `.mcp-place` switch rows. Borrowing stops
  // where the borrowed thing is wrong for the room — the register lays its
  // switches out as full-width cells, which on the owner's window put 200px of
  // nothing between a name and its own switch and made one server ten rows
  // tall. The frame is still theirs; the thing you press is not.
  import { onMount } from 'svelte'
  import {
    ListMCPServers, SaveMCPServer, ListExternalSkills, ListTools, PlacementTargets,
    SetMCPServerTargets, ListSubagentProfiles, InstallSkillFromGitHub,
    ToggleMCPServer, TestMCPServer, RemoveMCPServer,
    StartMCPSignIn, CompleteMCPSignIn, CancelMCPSignIn, MCPSignInStatus,
  } from '../../wailsjs/go/main/App'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
  import { setActiveView, openSettingsAt, startChatWith } from './stores/cockpit.svelte'
  import { t, type TKey } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import AgentFace from './AgentFace.svelte'
  import { faceOf } from './agentFace'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import { NAV } from './desks'
  import type { IconName } from './icons'
  import { coverHue } from './coverHue'
  import McpMark from './McpMark.svelte'
  import { MCP_PRESETS, needsPaste, presetConfig, type MCPPreset } from './mcpShelf'
  import { SKILL_PRESETS, type SkillPreset } from './skillShelf'

  let { onClose }: { onClose: () => void } = $props()

  type Kind = 'mcp' | 'skills' | 'builtin'
  type View = 'shelf' | 'mine'

  let kind = $state<Kind>('mcp')
  // Which half of a register is showing. One variable for both tabs rather than
  // one each: the choice a person makes here is "am I browsing or auditing",
  // and it is the same question on both, so carrying it across is what they
  // mean by having answered it.
  let view = $state<View>('shelf')

  // `url` and `command` are the row's second line — where this server actually
  // is. The room used to draw neither, because a three-column table had nowhere
  // to put an address; the register's row has always had the slot.
  type MCPRow = {
    name: string; disabled: boolean; status: string; tools: number
    for?: string[]; url?: string; command?: string[]
  }
  type SkillRow = { name: string; description: string; dir: string; bundled?: boolean }
  type ToolRow = { name: string; description: string; source: string; category: string }

  type TargetRow = { id: string; name: string; detail?: string; kind: string }

  let servers = $state<MCPRow[]>([])
  let skills = $state<SkillRow[]>([])
  let tools = $state<ToolRow[]>([])
  let targets = $state<TargetRow[]>([])
  // The roster, loaded for one reason: an agent's switch wears that agent's own
  // face. PlacementTarget does not carry `icon:` and should not — the roster is
  // where a profile's own drawing already lives, and Settings' panel reads it
  // from exactly here. All three fields of it: a profile may name its hair and
  // its glasses too, and half a face is a different person.
  let agents = $state<{ name: string; icon?: string; hair?: string; accessory?: string; hue?: string }[]>([])
  // Which server's placement panel is open. One at a time, like the register:
  // eleven switches under every row at once is the wall the register avoided.
  let openRow = $state('')
  // Which row's ⋯ menu is down, and the delete waiting for a yes.
  let menuFor = $state('')
  let confirmRemove = $state('')
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

  const isUp = (s: MCPRow) => !s.disabled && s.status === 'ok'

  // The three or four words that stand in for the panel while the row is shut.
  // The register prints "เปิดให้ 2 ที่" here, which is the right answer on a
  // page about servers; this room's only question is WHO, so it prints the
  // names and falls back to a count past two, where the names stop fitting on
  // one line beside the address.
  const ownerBadges = (s: MCPRow) => {
    const names = targetsOf(s).map(targetName)
    if (names.length === 0) return [{ text: t('settings.mcpForNobody'), warn: true }]
    if (names.length <= 2) return names.map((n) => ({ text: n, warn: false }))
    return [
      { text: names[0], warn: false },
      { text: t('capability.forMore', { n: String(names.length - 1) }), warn: false },
    ]
  }

  // The one fact this page exists to surface, and the one the old column could
  // not: whether the assistant a person actually talks to reaches any of this.
  //
  // **`for:` alone cannot answer it, and reading it that way was a false
  // sentence one measurement away from shipping.** A chat with no desk set is
  // the pre-modes full desk and carries EVERY server whatever `for:` says
  // (mode.go's `if m == nil { return true }`), which is the state the owner's
  // own machine was in: five servers pointed at nothing but agents, and 44
  // notion tools live in the session all the same. A banner saying "the
  // assistant cannot reach anything" would have been read by the one person it
  // was wrong about. Exactly the shape DESIGN.md §2 warns of — a value that is
  // true for more than one reason.
  //
  // So it is two facts, and the sentence needs both: the config carries nothing
  // for any desk, AND the registry actually running right now holds no MCP tool.
  // The second is measured, not inferred — ListTools is this session's own
  // registry, so when both hold, "cannot reach anything through MCP" is simply
  // what is true, whatever route the desk took to get there.
  const deskIds = $derived(targets.filter((x) => x.kind === 'desk').map((x) => x.id))
  const noDeskAnywhere = $derived(
    servers.length > 0 &&
    !tools.some((t) => t.source === 'mcp') &&
    !servers.some((s) => targetsOf(s).some((id) => deskIds.includes(id))),
  )

  // Placement, written the register's way — see the header note. Every helper
  // below mirrors Settings.svelte's by name so the two panels stay one thing.
  //
  // Flipping one switch sends the whole list back, because that is what the
  // engine stores: a per-target call would need the engine to merge, and two
  // places deciding what the list is now is how one of them ends up wrong.
  async function putTargets(s: MCPRow, label: string, next: string[]) {
    // The register says this with a disabled attribute alone, and that is enough
    // for a mouse. It is not enough for the rule: a switched-off server is
    // dropped by config.mcpServersFor before `for:` is ever read, so a write
    // here would persist a value the engine has already decided to ignore. The
    // guard lives in the writer so there is one place that knows it.
    if (s.disabled) return
    busy = label
    error = ''
    try {
      await SetMCPServerTargets(s.name, next)
      await load()
    } catch (err) {
      error = String(err)
    } finally {
      busy = ''
    }
  }

  const toggleTarget = (s: MCPRow, id: string) => {
    const current = targetsOf(s)
    return putTargets(s, 'target:' + s.name + ':' + id,
      current.includes(id) ? current.filter((x) => x !== id) : [...current, id])
  }

  // The three things a person standing on this row wants to do to a server and
  // could not, and the reason they could not was a rule that stopped being true.
  //
  // The room was built to refuse every edit, so that one config file would have
  // one editor. Then the placement panel came here and the room became the
  // second editor anyway — just one with its arms cut off, which is worse: it
  // makes a person hold a map of which half of the job lives on which page. So
  // the actions the register keeps on the row come with it (owner, 4 ก.ย. 2026).
  //
  // What deliberately does NOT come is the FORM. Address, key, env, cwd,
  // timeout, allowed tools: that is a second copy of eleven fields, which is the
  // duplication the original rule was actually about, and none of it is what
  // this room is for. แก้ไข opens the register and says so.
  async function runServer(label: string, fn: () => Promise<void>) {
    busy = label
    error = ''
    menuFor = ''
    try {
      await fn()
      await load()
    } catch (err) {
      error = String(err)
    } finally {
      busy = ''
    }
  }

  const pauseServer = (s: MCPRow) =>
    runServer('toggle:' + s.name, () => ToggleMCPServer(s.name, !s.disabled))
  const testServer = (s: MCPRow) =>
    runServer('test:' + s.name, async () => { await TestMCPServer(s.name) })
  const removeServer = (name: string) =>
    runServer('rm:' + name, () => RemoveMCPServer(name))

  // How many of a pack's skills are already on disk. The pack is the unit the
  // installer works in, so "added" cannot be a single name — a person who
  // deleted three of the fifty still has this pack, and a person who has
  // `code-review` from somewhere else does not.
  //
  // Matched case-insensitively against the folder name the installer writes,
  // which is what ListExternalSkills reports back.
  function installedOf(p: SkillPreset): number {
    const have = new Set(skills.map((s) => s.name.toLowerCase()))
    return p.installs.filter((n) => have.has(n.toLowerCase())).length
  }

  // One repository at a time, and the whole repository: InstallSkillFromGitHub
  // takes a URL and writes every skill folder in it, which is the reason the
  // card lists what arrives instead of naming one thing.
  //
  // It re-bootstraps the engine on the Go side, so `load()` afterwards is what
  // turns the card into เพิ่มแล้ว without a reload.
  async function installSkills(p: SkillPreset) {
    busy = p.name
    error = ''
    try {
      await InstallSkillFromGitHub(p.repo)
      await load()
    } catch (err) {
      error = String(err)
    } finally {
      busy = ''
    }
  }

  // Read the name into a plain local BEFORE closing the dialog. Written inline
  // first, as `{@const name = confirmRemove}` beside the dialog and a handler
  // that cleared confirmRemove and then passed `name` — and `{@const}` is a
  // getter over reactive state, not a snapshot, so by the time the call was made
  // it had already re-read the empty string. RemoveMCPServer('') deletes nothing
  // and reports success. The test that opens the dialog and presses through it
  // is what found this; a test that only checked the dialog appears would not
  // have.
  function removeConfirmed() {
    const name = confirmRemove
    confirmRemove = ''
    if (name) removeServer(name)
  }

  const editServer = () => {
    menuFor = ''
    toRegister('mcp')
  }

  const groupIds = (kind: string) => targets.filter((x) => x.kind === kind).map((x) => x.id)
  const groupOn = (s: MCPRow, kind: string) =>
    groupIds(kind).filter((id) => targetsOf(s).includes(id)).length
  const toggleGroup = (s: MCPRow, kind: string) => {
    const ids = groupIds(kind)
    const current = targetsOf(s)
    const allOn = ids.every((id) => current.includes(id))
    return putTargets(s, 'group:' + s.name + ':' + kind,
      allOn ? current.filter((id) => !ids.includes(id))
            : [...current, ...ids.filter((id) => !current.includes(id))])
  }

  // The glyphs the switches wear, taken from where each kind already keeps its
  // own — the sidebar's icon for a desk, the profile's `icon:` for an agent.
  // Same two lines as the register's, and deliberately so: an agent drawn one
  // way here and another way there is two people to whoever is reading.
  const deskIcon = (id: string): IconName =>
    NAV.find((n) => n.id === id)?.icon ?? (id === 'specialized' ? 'bot' : 'layoutList')
  // The whole face, not just the mark it holds: a profile may name its own
  // hair and glasses too, and an agent drawn with two thirds of what its owner
  // chose is the same "two people to whoever is reading" this comment already
  // warns about one line up.
  const agentFaceOf = (name: string) => faceOf(agents.find((x) => x.name === name))

  async function load() {
    const [m, k, tl, tg, ag] = await Promise.all([
      ListMCPServers(), ListExternalSkills(), ListTools(), PlacementTargets(),
      ListSubagentProfiles(),
    ])
    servers = m as MCPRow[]
    skills = k as SkillRow[]
    tools = tl as ToolRow[]
    targets = tg as TargetRow[]
    agents = (ag ?? []) as { name: string; icon?: string }[]
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
                <!-- The mark, from McpMark.svelte — the same component
                     ตั้งค่า › MCP draws its two lists with.
                     This comment used to argue against brand logos — "somebody
                     else's trademark in our binary" — which was never this
                     app's position: providerMarks.ts has shipped nineteen of
                     them since long before this room existed, with the licence
                     and the trademark line written above them. The argument
                     survives as the rule it should always have been: no FETCHED
                     logo, no file per server. mcpMarks.ts is inlined and
                     generated, same as its older twin. -->
                <McpMark name={p.name} size={26} />
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
          <!-- Said once, at the top, in words. Every switch under every row can
               be off for the desks and the page still reads as a healthy list of
               five servers — which is exactly what it did. -->
          {#if noDeskAnywhere}
            <div class="mcp-need-warn">
              <Icon name="alertTriangle" size={13} />
              <span class="mcp-need-txt">{t('capability.noDeskWarn')}</span>
            </div>
          {/if}
          <!-- The register's own row, not a lookalike (header note). Everything
               below is a class out of style.css that Settings' MCP panel already
               draws, so the two pages cannot drift into two designs. -->
          <div class="settings-card">
            {#each servers as s (s.name)}
              <div class="set-row reg-entry">
                <button
                  class="reg-head cap-srv-head"
                  aria-expanded={openRow === s.name}
                  onclick={() => (openRow = openRow === s.name ? '' : s.name)}
                >
                  <span class="reg-caret" class:open={openRow === s.name}>›</span>
                  <!-- The mark. A named thing in a gallery needs a face, and
                       this room draws four galleries; the ห้องสมุด card has
                       worn one since the room was built and the row it turns
                       into wore nothing, so the same server had a face on one
                       tab and none on the next. ตั้งค่า › MCP was the third
                       list with the same hole in it, which is what moved this
                       into a component. -->
                  <McpMark name={s.name} size={22} />
                  <span class="set-txt">
                    <span class="t">
                      <span class="dot" class:up={isUp(s)} class:off={s.disabled}></span>
                      {s.name}
                      <span class="mcp-badge">{s.url ? 'http' : 'stdio'}</span>
                      <!-- Off is a word, not just a grey dot (DESIGN.md §4) —
                           and it is the reason the switches below are dead, so
                           it has to be readable without opening the row. -->
                      {#if s.disabled}
                        <span class="mcp-badge mcp-badge-warn">{t('capability.serverOff')}</span>
                      {/if}
                      <!-- Absent rather than zero when nothing was ever
                           reported: a server that has not been reached yet has
                           no tool count, which is a different thing from having
                           no tools. -->
                      {#if s.tools > 0}
                        <span class="mcp-badge">{t('settings.mcpToolCount', { n: String(s.tools) })}</span>
                      {/if}
                      {#each ownerBadges(s) as b (b.text)}
                        <span class="mcp-badge" class:mcp-badge-warn={b.warn}>{b.text}</span>
                      {/each}
                    </span>
                    <span class="d cap-addr" title={s.url || (s.command ?? []).join(' ')}>
                      {s.url || (s.command ?? []).join(' ')}
                    </span>
                  </span>
                </button>

                <!-- The register's row actions, as a menu rather than four
                     buttons: the register is a settings page and can afford a
                     row of controls, while this row already carries a mark, a
                     status, three badges and an address. -->
                <div class="plus-menu-wrap cap-acts-menu">
                  <button
                    class="icobtn"
                    aria-label={t('capability.rowActions')}
                    aria-expanded={menuFor === s.name}
                    disabled={busy !== ''}
                    onclick={() => (menuFor = menuFor === s.name ? '' : s.name)}
                  >
                    <Icon name="ellipsisVertical" size={15} />
                  </button>
                  {#if menuFor === s.name}
                    <div class="plus-menu">
                      <button class="plus-menu-item" onclick={() => pauseServer(s)}>
                        <span class="ic"><Icon name={s.disabled ? 'plugZap' : 'plug'} size={14} /></span>
                        {s.disabled ? t('capability.serverResume') : t('capability.serverPause')}
                      </button>
                      <button class="plus-menu-item" disabled={s.disabled} onclick={() => testServer(s)}>
                        <span class="ic"><Icon name="refreshCw" size={14} /></span>
                        {t('settings.test')}
                      </button>
                      <button class="plus-menu-item" onclick={editServer}>
                        <span class="ic"><Icon name="pencil" size={14} /></span>
                        {t('capability.serverEdit')}
                      </button>
                      <button class="plus-menu-item danger" onclick={() => { menuFor = ''; confirmRemove = s.name }}>
                        <span class="ic"><Icon name="x" size={14} /></span>
                        {t('settings.remove')}
                      </button>
                    </div>
                  {/if}
                </div>

                {#if openRow === s.name}
                  <div class="mcp-targets">
                    <!-- A switched-off server carries nothing whoever it is
                         pointed at: config.mcpServersFor drops it before it
                         looks at `for:` at all. So the switches are dead, and a
                         dead control has to say why it is dead — otherwise this
                         panel takes a click and answers with nothing. -->
                    {#if s.disabled}
                      <div class="mcp-need-warn">
                        <Icon name="alertTriangle" size={13} />
                        <span class="mcp-need-txt">{t('capability.serverOffHint')}</span>
                        <button class="ctrl" onclick={() => toRegister('mcp')}>{t('capability.openRegister')}</button>
                      </div>
                    {/if}
                    <div class="d muted mcp-targets-hint">{t('settings.mcpForHint')}</div>
                    {#each ['desk', 'agent'] as place (place)}
                      {@const rows = targets.filter((x) => x.kind === place)}
                      {#if rows.length > 0}
                        {@const on = groupOn(s, place)}
                        <div class="mcp-targets-group">
                          <span class="eyebrow">
                            {place === 'desk' ? t('settings.mcpForDesks') : t('settings.mcpForAgents')}
                          </span>
                          <span class="mcp-group-count">{on}/{rows.length}</span>
                          <button class="mcp-group-all" disabled={busy !== '' || s.disabled} onclick={() => toggleGroup(s, place)}>
                            {on === rows.length ? t('settings.mcpForNone') : t('settings.mcpForAll')}
                          </button>
                        </div>
                        <!-- The register can say "โต๊ะ" bare, because a person
                             who reached three levels into Settings came looking
                             for it. This room is the announcement, so it is read
                             by someone meeting the word for the first time, and
                             a heading nobody can decode is a heading that is not
                             there. One line each, and only here. -->
                        <p class="cap-means">
                          {place === 'desk' ? t('capability.deskMeans') : t('capability.agentMeans')}
                        </p>
                        <!-- Chips, not the register's switch rows. The register
                             lays eleven of these out as a grid of full-width
                             cells, which puts every switch against the right
                             edge and its label against the left — on the
                             owner's window that is 200px of nothing between a
                             name and the control that belongs to it, and ten
                             rows tall for one server. A chip IS the control:
                             the label and the switch are one object, so there
                             is no gap to cross, and the same eleven fit in
                             four lines. What is on shows in colour, so the
                             answer to "who has this" is read rather than
                             counted. -->
                        <div class="cap-picks">
                          {#each rows as target (target.id)}
                            {@const isOn = targetsOf(s).includes(target.id)}
                            <button
                              class="cap-pick"
                              role="switch"
                              aria-checked={isOn}
                              title={target.detail ?? ''}
                              disabled={busy !== '' || s.disabled}
                              onclick={() => toggleTarget(s, target.id)}
                            >
                              {#if place === 'agent'}
                                <AgentFace name={target.name} {...agentFaceOf(target.name)} size={20} off={!isOn} />
                              {:else}
                                <span class="cap-pick-ic"><Icon name={deskIcon(target.id)} size={13} /></span>
                              {/if}
                              <span class="t">{target.name}</span>
                              <!-- The tick carries the state a second time, in
                                   shape rather than in colour: a chip that is
                                   only "the blue one" is a state you cannot see
                                   with the wrong eyes (DESIGN.md §4). -->
                              <span class="cap-pick-tick" aria-hidden="true">
                                {#if isOn}<Icon name="check" size={12} />{/if}
                              </span>
                            </button>
                          {/each}
                        </div>
                      {/if}
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
          <!-- NOT registerNote. That sentence hands editing, removing AND
               choosing who carries a server to the other page; one of those
               three now happens right here, so saying it whole would be false. -->
          <p class="cap-foot">
            {t('capability.mineNote')}
            <button class="linkish" onclick={() => toRegister('mcp')}>{t('capability.openRegister')}</button>
          </p>
        {/if}
      {/if}

      <!-- ================= สกิล ================= -->
      {#if kind === 'skills' && view === 'shelf'}
        <!-- This tab used to draw an honest empty state, and the note here said
             what it would take to fill it: "a curation job — pick the skills
             Aetox is good at and write a line each — not an engineering one".
             skillShelf.ts is that job done, on mcpShelf.ts's terms, so the
             empty state is gone rather than dressed up. What the older note
             refused is still refused: no card here was invented to make the
             tab look finished, and the three that exist each say what lands.

             The one shape this shelf has that the MCP one does not: a card is
             a repository, because InstallSkillFromGitHub cannot take less than
             one, so the count and the ✓ both come from the pack's own
             `installs` list. -->
        <p class="cap-note">{t('capability.skillShelfNote')}</p>
        <div class="cap-grid">
          {#each SKILL_PRESETS as p (p.name)}
            {@const got = installedOf(p)}
            <article class="cap-card" class:have={got === p.installs.length}>
              <div class="cap-card-head">
                <!-- Initials of owner and repo, not the first two letters of
                     either: three packs whose repos are `skills`,
                     `marketingskills` and `superpowers` gave sk / ma / su, and
                     the first of those says nothing about whose skills they
                     are. `ms` reads as the name on the card beside it. -->
                <span class="cap-mark" style="--px:26px; --h:{coverHue(p.name)}" aria-hidden="true">
                  {p.name.split('/').map((part) => part[0]).join('')}
                </span>
                <span class="cap-id">{p.name}</span>
                <span class="cap-tag">{t('capability.skillCount').replace('{n}', String(p.installs.length))}</span>
                <span class="cap-grow"></span>
                {#if got === p.installs.length}
                  <span class="cap-have"><Icon name="check" size={13} /> {t('capability.added')}</span>
                {:else}
                  <button class="ctrl" disabled={busy === p.name} onclick={() => installSkills(p)}>
                    {busy === p.name ? t('capability.adding') : t('capability.add')}
                  </button>
                {/if}
              </div>
              <!-- `why`, not `desc`, for the reason the MCP shelf gives above:
                   a person reading a shelf is asking what they are missing. -->
              <p class="cap-why">{p.why}</p>
              <!-- Said on the card rather than found out afterwards: pressing
                   this writes every one of them. A part-installed pack says so
                   too, because "เพิ่ม" on a card that is already half there
                   would be a lie about what the press does. -->
              <p class="cap-sub">
                {#if got > 0 && got < p.installs.length}
                  {t('capability.skillPartial').replace('{n}', String(got)).replace('{m}', String(p.installs.length))}
                {:else}
                  {t('capability.skillWrites').replace('{n}', String(p.installs.length)).replace('{kb}', String(p.kb))}
                {/if}
              </p>
            </article>
          {/each}
        </div>
        <p class="cap-foot">{t('capability.skillShelfFoot')}</p>
        <div class="cap-acts cap-acts-foot">
          <button class="ctrl" onclick={() => ask('settings.aiFindSkillPrompt')}>
            <Icon name="messageSquare" size={13} /> {t('settings.aiFindSkillTitle')}
          </button>
          <button class="ctrl" onclick={() => toRegister('skills')}>{t('settings.skillInstall')}</button>
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

<!-- Same gate the register puts a delete behind, and the same component: a
     removal that reads differently in two places is two promises about what
     the button does. -->
{#if confirmRemove}
  <ConfirmDialog
    title={t('settings.confirmMcpTitle')}
    message={t('settings.confirmMcpMessage')}
    detail={confirmRemove}
    confirmLabel={t('settings.remove')}
    onConfirm={removeConfirmed}
    onCancel={() => (confirmRemove = '')}
  />
{/if}

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

  /* The line under `why` that says what pressing the button writes. Dimmer
     than `why` on purpose: it is the receipt, not the pitch. */
  .cap-sub {
    margin: 0;
    color: var(--text-muted);
    font-size: 12px;
  }
  .cap-acts-foot { margin-top: 12px; }

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

  /* The ของคุณ half owns no layout of its own any more — it is the register's
     row and the register's placement panel, both read out of style.css. What is
     left here is the two marks that panel does not carry.

     The status dot: amber by default, because "configured but not reached yet"
     is the state a fresh row is in, and it is neither healthy nor off. It stays
     paired with words in the same row (the badges beside it) — a colour is not
     allowed to be the only difference between two states (DESIGN.md §4). */
  .dot {
    width: 7px; height: 7px; border-radius: 50%; flex: none;
    background: var(--status-warn);
  }
  .dot.up { background: var(--status-success); }
  .dot.off { background: var(--text-dim); }

  /* The ⋯ menu reuses .plus-menu-wrap / .plus-menu / .plus-menu-item out of
     style.css whole. Two overrides only, and both are about where it hangs:
     that menu was written to drop from a button at the left edge of a composer,
     and this one hangs off the right edge of a row. */
  /* One line, cut with an ellipsis, full text in the tooltip. kinocut's command
     is a 74-character absolute path; wrapped, it took a second line and pushed
     the ⋯ button off the row onto a line of its own, which made one row in five
     twice as tall as the rest for no reason a reader could see. The address is
     an identifier here, not something to read end to end. */
  .cap-addr {
    display: block;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* The chips the placement panel is made of. Everything else in this panel is
     still the register's own — .mcp-targets, the group heading, เลือกทั้งหมด —
     because the only part that was wrong was the control, not the frame. */
  .cap-picks {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .cap-pick {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    padding: 4px 10px 4px 5px;
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-full, 999px);
    background: var(--surface-panel);
    color: var(--text-muted);
    font: inherit;
    font-size: var(--fs-sm, 13px);
    cursor: pointer;
    transition: background .12s, color .12s, border-color .12s;
  }
  .cap-pick:hover:not(:disabled) { border-color: var(--border-strong); color: var(--text-primary); }
  .cap-pick:focus-visible { outline: 2px solid var(--interactive); outline-offset: 1px; }
  .cap-pick:disabled { opacity: .55; cursor: not-allowed; }
  /* On is the accent this app already uses for "the thing you chose", the same
     one .subseg wears under the tab you are standing on. */
  .cap-pick[aria-checked="true"] {
    color: var(--accent-bright);
    border-color: color-mix(in srgb, var(--accent) 45%, transparent);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
  }
  .cap-pick-ic {
    flex: none;
    display: grid;
    place-items: center;
    width: 20px;
    height: 20px;
    border-radius: var(--r-xs, 4px);
    background: var(--surface-raised);
    color: var(--text-dim);
  }
  .cap-pick[aria-checked="true"] .cap-pick-ic { color: var(--accent-bright); }
  .cap-pick-tick {
    flex: none;
    width: 12px;
    display: grid;
    place-items: center;
  }

  /* The address is nowrap so it can ellipsise, which gives .reg-head an
     intrinsic width as wide as the longest path on the machine — kinocut's is
     74 characters. .set-row wraps, and a flex line is broken on hypothetical
     size before anything is allowed to shrink, so on a narrow window that one
     row threw its ⋯ onto a line of its own. Basis 0 makes the row lay out from
     the space it has instead of from that path. */
  .cap-srv-head { flex: 1 1 0; min-width: 0; }

  .cap-acts-menu { flex: none; }
  .cap-acts-menu :global(.plus-menu) { left: auto; right: 0; }
  .cap-acts-menu :global(.plus-menu-item.danger) { color: var(--status-danger); }
  .cap-acts-menu :global(.plus-menu-item.danger:hover) { background: var(--status-danger-bg); }

  /* One line under โต๊ะ and under เอเจน, saying what the word means. The
     register does without it and should: by the time you are three levels into
     Settings you went looking for that word. This room is where a person meets
     it for the first time. */
  .cap-means {
    margin: -3px 0 8px;
    color: var(--text-dim);
    font-size: 12px;
    line-height: 1.5;
  }

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
  /* Columns, not a grid, and the difference is the whole reason this looks
     arranged rather than dropped.

     A grid puts these seven boxes on rows, and a row is as tall as the tallest
     box in it — so เว็บ (four lines of chips) set the height for อ่านสื่อ and
     งานโค้ด (one line each) beside it, and `align-items: start` turned that
     into two holes rather than two stretched boxes. Then การทำงานของผู้ช่วย,
     the biggest group, landed alone on the last row with two empty columns to
     its right. Three quarters of the page's lower half was nothing.

     The tempting fix is to reorder the groups so the tall ones pair up. That
     one is not available: this order is Go's own (internal/skill/category.go),
     the register upstairs prints the same sequence, and sorting by height would
     make the page's meaning depend on how many tools happened to be in a
     bucket that release.

     Column flow has no rows to be ragged about. Each box is as tall as its own
     contents, the browser balances the columns, and the order is preserved —
     read down the first column and on to the next, the way a newspaper is
     read. `break-inside: avoid` is what keeps a box whole across the break;
     without it a group's chips would split across two columns and the heading
     would be orphaned from half its own list. */
  .cap-cats {
    columns: 240px 3;
    column-gap: 10px;
  }
  .cap-cat {
    background: var(--surface-panel);
    border: 1px solid var(--border-subtle);
    border-radius: 10px;
    padding: 12px 13px 13px;
    /* Column layout has no row-gap, so the gap between stacked boxes is this
       margin. It matches column-gap so the spacing reads as one grid. */
    margin: 0 0 10px;
    break-inside: avoid;
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
