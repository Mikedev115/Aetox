<script lang="ts">
  // ความสามารถ: the MCP servers the assistant reaches, who carries each one,
  // and what else it could be reaching.
  //
  // **Since 13 ก.ย. 2026 the room is a rail of pages, ตั้งค่า's own frame.**
  // The owner read the 12 ก.ย. dashboard on his machine (five servers added,
  // none tested) and found it "ดูยาก ไม่พอ พางง": the servers were the last
  // thing on the page, under four tiles that contradicted each other ("ต่ออยู่
  // 5" over "ยังไม่เคยต่อ" ×5, "ต่อได้ 0 · ต่อไม่ได้ 0 · ทุกตัวปกติ") and nine
  // person cards of which six said "ยังไม่ได้ให้ MCP ตัวไหน". What he asked for
  // instead: "หน้าต่างย่อย เหมือนหน้าตั้งค่า" — a left rail, and one page per
  // question:
  //
  //   - **MCP server ของคุณ** — readiness only: is it connected, signed in,
  //     keyed, and which tools it offers. "ล็อคอิน ใส่คีย์ ดูรายละเอียด tool
  //     ต้องจบที่หน้านี้" and nothing here hands a server to anyone. The four
  //     tiles are one sentence whose clauses are drawn only when true.
  //   - **ตั้งค่า MCP ฝั่งผู้ช่วยและโค้ด** — a card per side, what it holds,
  //     one button that opens the picker for it. "มีหน้าที่แค่เลือกใช้".
  //   - **ตั้งค่า MCP สำหรับเอเจนเฉพาะ** — the same, a card per agent, plus
  //     what its own file says it needs and has not got.
  //   - **ห้องสมุด MCP** — unchanged.
  //
  // So the `for:` list has one writer (`put`) behind one picker, opened from
  // a target card on either placement page. The server's sheet no longer
  // carries a ใครได้ใช้ tab: that was the placement question on the
  // readiness page.
  //
  // **สกิล is the second heading on the rail (13 ก.ย.), three pages, moved
  // whole out of ตั้งค่า (DESIGN.md §1: one thing, one place to change it;
  // ตั้งค่า › สกิล left the menu the way ตั้งค่า › MCP did). Not four: the MCP
  // rail has a placement page per side because a server carries a `for:`
  // list and a skill does not — every desk keeps every shelf skill
  // (mode.go, Carries: "a skill is knowledge, not capability"), so a page
  // asking which desk gets which skill would have nothing to change.
  //
  //   - **สกิลของคุณ** — the shelf (~/.aetox/skills + what ships in the
  //     binary): is it there, does it read, add one, take one out. Three
  //     roads in on one sheet (GitHub, zip, by hand), as ตั้งค่า had them.
  //   - **ตั้งค่าสกิลสำหรับเอเจนเฉพาะ** — a card per agent with what is in
  //     its own folder (agents/<name>/skills), and one sheet where a shelf
  //     skill is ticked INTO that folder. A copy, not a pointer: an agent's
  //     knowledge is a folder in its home and nothing else (subagent/skills.go
  //     says why the shelf is the wrong place), so the tick is
  //     CopySkillToAgent and unticking is RemoveAgentSkill. What shipped with
  //     the profile is listed and cannot be removed here; a same-named folder
  //     replaces it, the shelf's own rule.
  //   - **ห้องสมุดสกิล** — the curated open-source packs (skillShelf.ts).
  //
  // The registers ตั้งค่า still holds (บัญชี เครื่องมือในตัว) are the next
  // headings, moved the same way. Until they move, the foot of ของคุณ links
  // เครื่องมือ.
  //
  // **This room is the ONE place an MCP server is handled.** Rebuilt 12 ก.ย.
  // 2026 after the owner read the previous version against the other rooms
  // (ทีมงาน, โปรเจกต์) and found it was the only room in the app that did
  // three things none of the others do:
  //
  //   1. It held three kinds of thing (MCP / สกิล / เครื่องมือในตัว) behind a
  //      pair of tab bars. ทีมงาน is agents; โปรเจกต์ is projects; no room
  //      switches subject. The lede still names all three — that was the
  //      announcing job the room was founded for (d90e7d60: "ประกาศด้วยประโยคนำ
  //      ไม่ใช่ป้ายที่ต้องกด") — but skills and built-in tools are registers
  //      that need no connecting, and they stay in ตั้งค่า, one link each.
  //   2. It borrowed the settings register's `.set-row.reg-entry` row for a
  //      room. DESIGN.md §1: borrow the frame, not the ornament — and the
  //      ornament of a settings page is a settings page. The card here is
  //      `.chair-card.agc`, the roster's own, with the mark at the roster's 38px.
  //   3. The same `for:` list had three editors: this room's panel, ตั้งค่า ›
  //      MCP's panel, and the MCP box on an agent's own settings page. One is
  //      left: the picker at the top of the sheet the gear on a card opens.
  //      ตั้งค่า › MCP is gone from the menu — not left as a page that points
  //      here (DESIGN.md §3) — and its eleven-field form is that same sheet.
  //
  // **ของคุณ is a dashboard, read-only; every change is on the sheet.** Three
  // shapes were built and taken out on 12 ก.ย. before this one: a grid of
  // servers by people ("ไม่ควรอยู่แบบนี้"), a row of chips per person (90 chips
  // for ten servers), and a drag-into-pocket board. Each put switches on the
  // page and each became a switchboard. What the owner asked for instead was
  // "แดชบอร์ดภาพรวม เห็นว่าตัวไหนถืออะไรอยู่ ส่วนตั้งค่า ทำหน้านี้ให้ดี": a page
  // that shows who holds what, and one sheet that is where anything changes.
  //
  //   - (12 ก.ย.) Four tiles, the two mains as a tier above the agents, a
  //     server showing the faces holding it. Replaced 13 ก.ย. by the rail
  //     above; what survived is the rule itself — the page shows, the sheet
  //     changes — and the agent card, which is now a page of its own.
  //
  // **A grid was tried and taken out the same afternoon.** Rows of servers by
  // columns of desks and agents answered both "this server, who?" and "this
  // agent, what?" at once, and the owner's reaction to it on the real page was
  // "ไม่ควรอยู่แบบนี้": a ten-column switchboard in the middle of a room. What
  // he asked for instead is the older shape, per server, behind a button — so
  // the picker is chips on the sheet, opened from the card, and opened for you
  // the moment a server is added (owner, 12 ก.ย.: "หลังจากเพิ่ม MCP ควรจะมี
  // แจ้งเตือนว่าไปตั้งค่า MCP ก่อน").
  //
  // **Two desks, named in Thai, and no third.** PlacementTargets sends the
  // engine's mode names (assistant / coding / specialized); the chips wear the
  // sidebar's words for the first two (NAV) and do not draw the third at all —
  // the office desk is what an agent's own chat runs on, and every agent is a
  // chip of its own already, so a chip for it was the same question twice.
  //
  // The 4 ก.ย. rule ("the form does not come to the room, only the register
  // has it") is kept in the only half that mattered: there is still exactly one
  // form. It just lives here now (owner, 12 ก.ย.: "หน้าตั้งค่าซ้ำซ้อนไป …
  // ให้มันครบจบที่เดียว").
  //
  // Nothing on the page is a lookalike. `.sec-head`, `.ag-reach`, `.ag-band`,
  // `.office-grid`, `.chair-*`, `.feed-filter .pill`, `.office-note` are the
  // roster's selectors read out of style.css; the sheet and its chips are the
  // shapes this room has that no other room does, and they are the only rules
  // in the style block below.
  import { onMount, tick } from 'svelte'
  import { pageWindow, nearViewport } from './pageWindow.svelte'
  import {
    ListMCPServers, SaveMCPServer, ListTools, ListExternalSkills, PlacementTargets,
    SetMCPServerTargets, ListSubagentProfiles,
    ToggleMCPServer, TestMCPServer, RemoveMCPServer, MCPConfigPath, OpenMCPFolder,
    StartMCPSignIn, CompleteMCPSignIn, CancelMCPSignIn, MCPSignInStatus,
    SkillsDir, SkillScanIssues, InstallSkillFromGitHub, InstallSkillFromZip, RemoveExternalSkill,
    RefreshSkills, OpenSkillsFolder, AgentSkills, OpenAgentSkillsFolder, CopySkillToAgent, RemoveAgentSkill,
  } from '../../wailsjs/go/main/App'
  import { config } from '../../wailsjs/go/models'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
  import { cockpit, openSettingsAt, startChatWith } from './stores/cockpit.svelte'
  import { t, type TKey } from './i18n.svelte'
  import Icon from './Icon.svelte'
  import ScopeMark from './ScopeMark.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import { lookOf } from './mascot/agentLook'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import { NAV } from './desks'
  import type { IconName } from './icons'
  import McpMark from './McpMark.svelte'
  import { scopeMeta } from './memoryScope'
  import { MCP_PRESETS, SHELF_GROUPS, needsPaste, presetConfig, type MCPPreset, type ShelfGroup } from './mcpShelf'
  import { SKILL_PRESETS, type SkillPreset } from './skillShelf'
  import { coverHue } from './coverHue'

  let { onClose }: { onClose: () => void } = $props()

  // What the engine reports per server. `err` is the sentence a failed connect
  // left behind; it was on the wire all along and this room never read it,
  // which is how "failed" and "never tried" wore the same dot.
  type MCPRow = {
    name: string; disabled: boolean; status: string; tools: number; tokens?: number; err?: string
    for?: string[]; url?: string; command?: string[]
    environment?: Record<string, string>; headers?: Record<string, string>
    cwd?: string; timeoutMs?: number; allowed?: string[]; toolList?: ToolCost[]
  }
  type TargetRow = { id: string; name: string; detail?: string; kind: string }
  type Profile = { name: string; description?: string; icon?: string; shell?: string; top?: string; face?: string; accent?: string; hue?: string; needs?: string[] }
  type ToolCost = { name: string; tokens: number }

  let servers = $state<MCPRow[]>([])
  let targets = $state<TargetRow[]>([])
  let agents = $state<Profile[]>([])
  let toolTotal = $state(0)
  // Whether this session's own registry holds any MCP tool right now. A chat
  // with no desk set carries EVERY server whatever `for:` says (mode.go), so
  // "the assistant cannot use any" read off `for:` alone would be a false
  // sentence on exactly the machine it was written for (the owner's).
  let mcpLive = $state(false)
  // The shelf, as ListExternalSkills reports it: bundled first, then the
  // user's folders. `before` is the moment a skill names for itself (§221).
  type SkillRow = { name: string; description: string; dir: string; bundled?: boolean; before?: string }
  let skills = $state<SkillRow[]>([])
  let skillsDir = $state('')
  let skillIssues = $state<string[]>([])
  // What each agent holds in its own folder, read when the agents page opens.
  type AgentSkill = { name: string; description: string; bundled: boolean }
  let agentSkills = $state<Record<string, AgentSkill[]>>({})
  let skillPick = $state('')            // the agent whose skill sheet is open
  let skillInstall = $state(false)      // the install sheet
  let skillInstallUrl = $state('')
  let skillInstallResult = $state('')
  let confirmSkill = $state<SkillRow | null>(null)
  let storedAt = $state('')
  let busy = $state('')
  let error = $state('')
  let loaded = $state(false)
  let confirmRemove = $state('')
  let shelfFilter = $state<'' | ShelfGroup>('')
  // Filtering is a new list, so the window starts over with it.
  const setShelfFilter = (g: '' | ShelfGroup) => { shelfFilter = g; winShelf.reset(); winSignIn.reset() }
  // The rail. Opens on ของคุณ when there is anything in it, on ห้องสมุด when
  // there is not: the room's founding point was that an empty register
  // announces nothing, and a full one is what a person came back for.
  type Page = 'mine' | 'desks' | 'agents' | 'shelf' | 'skills' | 'skagents' | 'skshelf'
  // Two headings, one per kind of thing; a new kind is a new heading, never
  // a tab. The MCP group has four rows and the skill group three — see the
  // note at the top for why the skill rail has no placement page per side.
  type Row = { id: Page; labelKey: TKey; icon: IconName }
  const RAIL: { labelKey: TKey; rows: Row[] }[] = [
    { labelKey: 'capability.navGroupReach', rows: [
      { id: 'mine', labelKey: 'capability.navMine', icon: 'plug' },
      { id: 'desks', labelKey: 'capability.navDesks', icon: 'sparkles' },
      { id: 'agents', labelKey: 'capability.navAgents', icon: 'bot' },
      { id: 'shelf', labelKey: 'capability.navShelf', icon: 'layoutList' },
    ] },
    { labelKey: 'capability.navGroupSkills', rows: [
      { id: 'skills', labelKey: 'capability.navSkills', icon: 'puzzle' },
      { id: 'skagents', labelKey: 'capability.navSkillAgents', icon: 'bot' },
      { id: 'skshelf', labelKey: 'capability.navSkillShelf', icon: 'layoutList' },
    ] },
  ]
  let page = $state<Page>('shelf')
  let pageSettled = false
  const goPage = (p: Page) => {
    page = p
    if (SKILL_PAGES.includes(p) && !skillsLoaded) void loadSkills()
    // A window belongs to a visit, not to the session: leaving a page and
    // coming back should not hand the reader 400 cards it did not ask for.
    winMine.reset(); winBundled.reset(); winShelf.reset(); winSignIn.reset(); winSkShelf.reset(); winServers.reset()
    if (p === 'mine' && loaded) void probeIdle()
    if (p === 'skagents') void loadAgentSkills()
  }
  // A window per long grid (pageWindow.svelte.ts): the DOM this room asks
  // for at once, rationed. 494 bundled skills is 13,700 nodes in one frame
  // otherwise (owner, 13 ก.ย.: "แบ่งโหลด ไม่งั้นแย่").
  const winMine = pageWindow()
  const winBundled = pageWindow()
  const winShelf = pageWindow()
  const winSignIn = pageWindow()
  const winSkShelf = pageWindow()
  const winServers = pageWindow()
  let signInPreset = $state<MCPPreset | null>(null)
  let signInURL = $state('')

  // Two loads, one per heading on the rail. The MCP half is a handful of
  // rows; the skill half walks every skill folder on disk (494 on the
  // owner's machine) and reads each SKILL.md, so it is read when a skill page
  // is first opened and again only after a skill action — not after every
  // MCP test, which `run` used to do eight calls at a time.
  const SKILL_PAGES: Page[] = ['skills', 'skagents', 'skshelf']
  const onSkillPage = () => SKILL_PAGES.includes(page)
  let skillsLoaded = false
  async function load() {
    const [m, tl, tg, ag, path] = await Promise.all([
      ListMCPServers(), ListTools(), PlacementTargets(), ListSubagentProfiles(), MCPConfigPath(),
    ])
    servers = (m ?? []) as MCPRow[]
    signInsLoaded = loadSignIns()
    toolTotal = (tl ?? []).length
    mcpLive = (tl ?? []).some((x: { source?: string }) => x.source === 'mcp')
    targets = (tg ?? []) as TargetRow[]
    agents = (ag ?? []) as Profile[]
    storedAt = path ?? ''
    loaded = true
    if (!pageSettled) { page = servers.length > 0 ? 'mine' : 'shelf'; pageSettled = true }
  }
  async function loadSkills() {
    const [k, sd, si] = await Promise.all([ListExternalSkills(), SkillsDir(), SkillScanIssues()])
    skills = (k ?? []) as SkillRow[]
    skillsDir = sd ?? ''
    skillIssues = si ?? []
    skillsLoaded = true
    if (page === 'skagents' || skillPick) await loadAgentSkills()
  }
  onMount(async () => {
    await load()
    void probeIdle()
    await arriveAt()
  })
  // Where another page asked this room to open (cockpit.capabilityIntent):
  // one of its pages, and on the two per-agent pages that agent's own sheet.
  // The agent editor's "เลือก MCP" / "จัดสกิล" doors use it — landing on the
  // room's front page and leaving the reader to find their agent among the
  // cards was the complaint (owner, 13 ก.ย. 2026: "กดปุ่มความสามารถแล้วควรพามา
  // หน้าตั้งค่า MCP สำหรับเอเจนเฉพาะสิครับ"). Consumed once.
  async function arriveAt() {
    const intent = cockpit.capabilityIntent
    if (!intent) return
    cockpit.capabilityIntent = null
    const pg = intent.page as Page
    if (!RAIL.some((g) => g.rows.some((r) => r.id === pg))) return
    pageSettled = true
    goPage(pg)
    if (SKILL_PAGES.includes(pg) && !skillsLoaded) await loadSkills()
    if (!intent.agent) return
    if (pg === 'agents') {
      const target = agentTargets.find((x) => x.name === intent.agent)
      if (target) await openPicker(target.id)
    } else if (pg === 'skagents') {
      if (agentTargets.some((x) => x.name === intent.agent)) skillPick = intent.agent
    }
  }

  // The probe. A server's tool count, its cost and its tool list live on the
  // server and arrive with a connect, so a page of never-connected servers
  // showed "ยังไม่เคยต่อ" five times and asked the owner to press ทดสอบ on
  // each ("ผู้ใช้ต้องไปดูทีละอันหรอมันถึงจะขึ้น", 13 ก.ย.). The room connects
  // them itself when ของคุณ opens: two at a time so five stdio servers do
  // not start five processes at once, each card saying it is being tested,
  // the grid refreshed as each one lands. Not a server that wants a login
  // (it would only 401), not one switched off, and never twice per visit;
  // ทดสอบทั้งหมด stays for a deliberate re-run.
  let probing = $state(new Set<string>())
  let probed = false
  let signInsLoaded: Promise<void> = Promise.resolve()
  async function probeIdle() {
    if (probed || page !== 'mine') return
    probed = true
    await signInsLoaded
    const queue = live.filter((x) => stateOf(x) === 'idle' && !x.toolList?.length && !wantsSignIn(x))
    if (queue.length === 0) return
    const worker = async () => {
      for (let x = queue.shift(); x; x = queue.shift()) {
        probing = new Set([...probing, x.name])
        try { await TestMCPServer(x.name) } catch { /* the card reads the engine's verdict on reload */ }
        probing = new Set([...probing].filter((n) => n !== x!.name))
        await load()
      }
    }
    await Promise.all([worker(), worker()])
  }

  async function run(label: string, fn: () => Promise<void>) {
    busy = label
    error = ''
    try {
      await fn()
      // An MCP action reloads the MCP half; a skill action, or one taken
      // while a skill page is open, the skill half as well.
      await load()
      if (onSkillPage() || skillPick || label.includes('skill')) await loadSkills()
    } catch (err) {
      error = String(err)
    } finally {
      busy = ''
    }
  }

  // ---- facts about a server -------------------------------------------------
  const OFFICE_DESK = 'specialized'
  const desks = $derived(targets.filter((x) => x.kind === 'desk' && x.id !== OFFICE_DESK))
  const agentTargets = $derived(targets.filter((x) => x.kind === 'agent'))
  const deskIds = $derived(desks.map((x) => x.id))
  const targetsOf = (s: MCPRow) => s.for ?? []
  // A desk's name is the sidebar's word for it, not the engine's mode id.
  const deskLabel = (id: string) => {
    const nav = NAV.find((n) => n.kind === 'desk' && n.id === id)
    return nav ? t(nav.labelKey) : id
  }
  const targetName = (id: string) => {
    const x = targets.find((y) => y.id === id)
    if (!x) return id.replace(/^agent:/, '')
    return x.kind === 'desk' ? deskLabel(x.id) : x.name
  }
  const onDesk = (s: MCPRow) => !s.disabled && targetsOf(s).some((id) => deskIds.includes(id))
  const inHand = $derived(servers.filter(onDesk))
  const presetOf = (name: string) => MCP_PRESETS.find((p) => p.name.toLowerCase() === name.toLowerCase())
  const address = (s: MCPRow) => s.url || (s.command ?? []).join(' ')
  const taken = (name: string) => servers.some((s) => s.name.toLowerCase() === name.toLowerCase())

  // Status is FOUR words, not a dot. The engine says idle / connected / failed
  // (internal/mcp/client.go) or "disabled"; the old room compared against 'ok',
  // which nothing ever sends, so its green never lit.
  type St = 'connected' | 'idle' | 'failed' | 'off'
  const stateOf = (s: MCPRow): St =>
    s.disabled ? 'off' : s.status === 'connected' ? 'connected' : s.status === 'failed' ? 'failed' : 'idle'
  const stateKey: Record<St, TKey> = {
    connected: 'capability.stConnected', idle: 'capability.stIdle',
    failed: 'capability.stFailed', off: 'capability.stOff',
  }

  // What a server costs on every message, said in the composer's own units.
  // Live for a connected server (the engine measured its tool block), and the
  // shelf's last measurement for one not added yet (mcpShelf.ts).
  const fmtTokens = (n: number) => (n >= 1000 ? (n / 1000).toFixed(1) + 'k' : String(n))
  const costLine = (tools: number, tokens: number | undefined) =>
    t('settings.mcpToolCount', { n: String(tools) }) +
    (tokens && tokens > 0 ? ' · ' + t('capability.tokensPerMsg', { n: fmtTokens(tokens) }) : '')


  // ---- the dashboard --------------------------------------------------------
  const live = $derived(servers.filter((x) => !x.disabled))
  const holds = (id: string) => live.filter((x) => targetsOf(x).includes(id))
  // What a server adds to every message. The engine's own measure once it
  // has connected; before that, the library's measurement for the same
  // server (mcpShelf.ts), marked ~ wherever it is shown. A cost that reads
  // "0" on a server that simply has not connected yet was the wrong number
  // (owner, 13 ก.ย.: "ควรแสดงโทเค็น … MCP นี้จะเพิ่มโทเค็นในระบบเท่าไหร่").
  const measured = (x: MCPRow) => !!x.tokens && x.tokens > 0
  const tokensOf = (x: MCPRow) => (measured(x) ? x.tokens! : (presetOf(x.name)?.tokens ?? 0))
  const toolsOf = (x: MCPRow) => (x.tools > 0 ? x.tools : (presetOf(x.name)?.toolCount ?? 0))
  const estimated = (x: MCPRow) => !measured(x) && !!presetOf(x.name)?.tokens
  const tokMark = (x: MCPRow) => (estimated(x) ? '~' : '') + fmtTokens(tokensOf(x))
  const tokensHeld = (id: string) => holds(id).reduce((a, x) => a + tokensOf(x), 0)
  const toolsHeld = (id: string) => holds(id).reduce((a, x) => a + toolsOf(x), 0)
  const heldEstimated = (id: string) => holds(id).some(estimated)
  const anyEstimated = $derived(live.some(estimated))
  const unmetNeeds = (agentName: string) =>
    live.filter((x) => !targetsOf(x).includes('agent:' + agentName) && needsServer(agentName, x.name)).map((x) => x.name)
  const heldByNobody = $derived(live.filter((x) => targetsOf(x).filter((id) => id !== OFFICE_DESK).length === 0))
  const failed = $derived(live.filter((x) => x.status === 'failed'))
  const connected = $derived(live.filter((x) => x.status === 'connected'))
  const idle = $derived(live.filter((x) => stateOf(x) === 'idle'))
  const off = $derived(servers.filter((x) => x.disabled))
  // Whether each oauth server (mcpShelf `oauth: true`) has a token on this
  // machine. Read after the list, never awaited by the cards: a slow oauth
  // store must not hold up the grid. Unknown reads as signed in, so the card
  // offers ทดสอบ (which will say 401) rather than a login it may not need.
  let signedIn = $state<Record<string, boolean>>({})
  // A server signs in through the browser when its config says so: a header
  // whose value is a `${connect:…}` reference. Read off the server rather
  // than its preset, so one added by address, or whose preset has since
  // left the shelf, keeps its เข้าสู่ระบบ.
  const isOAuth = (x: MCPRow) =>
    Object.values(x.headers ?? {}).some((v) => /\$\{connect:[^}]+\}/.test(v)) || !!presetOf(x.name)?.oauth
  async function loadSignIns() {
    const names = servers.filter(isOAuth).map((x) => x.name)
    const answers = await Promise.all(names.map((n) => MCPSignInStatus(n).then((st) => st.signed_in, () => true)))
    signedIn = Object.fromEntries(names.map((n, i) => [n, answers[i]]))
  }
  const wantsSignIn = (x: MCPRow) => isOAuth(x) && signedIn[x.name] === false
  const unsigned = $derived(live.filter(wantsSignIn))
  // What the rail counts on ของคุณ: servers that cannot be used as they are.
  const notReady = $derived(live.filter((x) => x.status === 'failed' || wantsSignIn(x)))
  // A login for a server already on the list, from its card. The same road
  // the library takes for a new oauth preset (addOAuth), without the save.
  async function signInServer(x: MCPRow) {
    if (!x.url) return
    busy = x.name
    error = ''
    try {
      signInPreset = presetOf(x.name) ?? { name: x.name, url: x.url, desc: '', why: '', group: 'apps', proven: false, oauth: true }
      const prompt = await StartMCPSignIn(x.name, x.url)
      signInURL = prompt.url
      if (prompt.url) BrowserOpenURL(prompt.url)
      await CompleteMCPSignIn(x.name)
      signInPreset = null
      await load()
    } catch (err) {
      error = String(err)
      signInPreset = null
    } finally {
      busy = ''
    }
  }
  // Every server that is not switched off, once each, in order. A sentence
  // per server names what is being tested (DESIGN.md §4).
  const testAll = () => run('testall:', async () => {
    for (const s of live) { busy = 'testall:' + s.name; await TestMCPServer(s.name) }
  })
  // A desk wears exactly what ตั้งค่า › การเรียนรู้ draws for it (memoryScope):
  // the same tile, the same tone (cyan for ผู้ช่วย, amber for โค้ด), the same
  // two words under the name. Owner, 12 ก.ย., with that page open: "แก้อันนี้
  // ให้เหมือนกันดีกว่า แบบใช้สีเหมือนกันจะได้ไม่งง". One identity per desk.
  const deskMeta = (id: string) => (id === 'assistant' ? scopeMeta('') : scopeMeta('mode:' + id))
  // An agent's role in three words: the profile's `description:` up to its
  // dash, with the "เอเจน" every profile opens with taken off. The full line is
  // on hover and on the roster, where it belongs.
  const roleOf = (x: TargetRow) => {
    const full = agents.find((a) => a.name === x.name)?.description ?? x.detail ?? ''
    return full.split(/\s[—–-]\s/)[0].replace(/^เอเจน/, '').trim()
  }
  const personDesc = (x: TargetRow) =>
    x.kind === 'desk'
      ? (() => { const nav = NAV.find((n) => n.kind === 'desk' && n.id === x.id); return nav ? t(nav.blurbKey) : (x.detail ?? '') })()
      : (agents.find((a) => a.name === x.name)?.description ?? x.detail ?? '')
  const agentsInNeed = $derived(agentTargets.filter((a) => unmetNeeds(a.name).length > 0))

  // ---- the picker -----------------------------------------------------------
  // Which agents declare `needs: mcp:<server>`, read off their own files. The
  // chip wears a wrench when that agent is not switched on for it — a healthy
  // server switched off for the one agent that cannot work without it is the
  // state nothing used to say.
  const needsServer = (agentName: string, server: string) =>
    (agents.find((a) => a.name === agentName)?.needs ?? []).some((entry) =>
      entry.split('|').some((alt) => alt.trim().toLowerCase() === 'mcp:' + server.toLowerCase()))

  const isOn = (s: MCPRow, id: string) => targetsOf(s).includes(id)

  // One writer. Every flip sends the whole list, because that is what the
  // engine stores; a switched-off server is refused here rather than written
  // and ignored (config.mcpServersFor drops it before reading `for:`).
  const put = (s: MCPRow, label: string, next: string[]) => {
    if (s.disabled) return Promise.resolve()
    return run(label, () => SetMCPServerTargets(s.name, next))
  }
  const toggleTarget = (s: MCPRow, id: string) =>
    put(s, 'target:' + s.name + ':' + id, isOn(s, id) ? targetsOf(s).filter((x) => x !== id) : [...targetsOf(s), id])
  // All or none of the live servers for one target: one write per server,
  // in order, inside one busy span, because the engine stores a list per
  // server and there is no call that writes them all.
  const toggleAllFor = (id: string) => run('all:' + id, async () => {
    const all = live.every((x) => isOn(x, id))
    for (const x of live) {
      if (all === isOn(x, id)) await SetMCPServerTargets(x.name, all ? targetsOf(x).filter((y) => y !== id) : [...targetsOf(x), id])
    }
  })

  const deskIcon = (id: string): IconName =>
    NAV.find((n) => n.id === id)?.icon ?? (id === 'specialized' ? 'bot' : 'layoutList')
  const agentLookOf = (name: string) => lookOf(agents.find((x) => x.name === name))

  // ---- row actions ----------------------------------------------------------
  const testServer = (s: MCPRow) => run('test:' + s.name, async () => { await TestMCPServer(s.name) })
  const pauseServer = (s: MCPRow) => run('toggle:' + s.name, () => ToggleMCPServer(s.name, !s.disabled))
  function removeConfirmed() {
    const name = confirmRemove
    confirmRemove = ''
    if (name) run('rm:' + name, async () => { await RemoveMCPServer(name); if (sheet?.original === name) sheet = null })
  }

  // ---- skills ---------------------------------------------------------------
  const shelfMine = $derived(skills.filter((x) => !x.bundled))
  const shelfBundled = $derived(skills.filter((x) => x.bundled))
  // The lettered tile ห้องสมุด gives a skill, `aetox-` dropped so twenty tiles
  // do not all read `ae`.
  const skillMark = (n: string) => n.replace(/^aetox-/, '').slice(0, 2)
  const shortSkill = (n: string) => n.replace(/^aetox-/, '')
  // A folder shown relative to the shelf: the shelf's path is printed once
  // on the band, and a card repeating it thirty times is a wall.
  const skillDirShort = (dir: string) => (skillsDir && dir.startsWith(skillsDir) ? '~' + dir.slice(skillsDir.length) : dir)
  async function loadAgentSkills() {
    const names = agentTargets.map((x) => x.name)
    const rows = await Promise.all(names.map((n) => AgentSkills(n)))
    const next: Record<string, AgentSkill[]> = {}
    names.forEach((n, i) => { next[n] = (rows[i] ?? []) as AgentSkill[] })
    agentSkills = next
  }
  const ownOf = (agent: string) => agentSkills[agent] ?? []
  const hasOwn = (agent: string, name: string) => ownOf(agent).some((x) => x.name.toLowerCase() === name.toLowerCase())
  // The tick: into the agent's folder, or out of it.
  const toggleAgentSkill = (agent: string, s: SkillRow) => run('agsk:' + agent + ':' + s.name, async () => {
    if (hasOwn(agent, s.name)) await RemoveAgentSkill(agent, s.name)
    else await CopySkillToAgent(agent, s.name)
  })
  function skillConfirmed() {
    const x = confirmSkill
    confirmSkill = null
    if (x) void run('rmsk:' + x.name, () => RemoveExternalSkill(x.name))
  }
  const refreshSkills = () => run('refresh-skills', () => RefreshSkills())
  // How many of a pack's skills are already on the shelf, matched the way the
  // installer names folders. The pack is the unit (skillShelf.ts).
  const installedOf = (p: SkillPreset) => {
    const have = new Set(skills.map((x) => x.name.toLowerCase()))
    return p.installs.filter((n) => have.has(n.toLowerCase())).length
  }
  const installPack = (p: SkillPreset) => run('pack:' + p.name, async () => {
    skillInstallResult = await InstallSkillFromGitHub(p.repo)
  })
  const installSkillURL = () => run('install-skill', async () => {
    skillInstallResult = await InstallSkillFromGitHub(skillInstallUrl.trim())
    skillInstallUrl = ''
  })
  // The picker is native; an empty report is a dismissed dialog, not a failure.
  const installSkillZip = () => run('install-zip', async () => {
    const report = await InstallSkillFromZip()
    if (report) skillInstallResult = report
  })
  async function askSkillAssistant() {
    onClose()
    await startChatWith(t('settings.aiFindSkillPrompt'))
  }

  // ---- the sheet: the one form ----------------------------------------------
  type Sheet = {
    original: string
    kind: 'stdio' | 'http'
    name: string; command: string; url: string
    envText: string; headersText: string; cwd: string; timeout: string
    // The allowlist. `null` is "everything the server offers", which is what
    // an empty `tools:` means to the engine; a Set is the names kept.
    picked: Set<string> | null
    needsKey: boolean
    // The preset's locale line saying where a pasted value comes from, or ''.
    hint: TKey | ''
  }
  type SheetTab = 'conn' | 'tools'
  let sheet = $state<Sheet | null>(null)
  let sheetTab = $state<SheetTab>('conn')
  // The target (desk id or agent:<name>) whose picker is open, or ''.
  let pickFor = $state('')
  let pickerEl = $state<HTMLElement | null>(null)
  const pickTarget = $derived(targets.find((x) => x.id === pickFor))
  async function openPicker(id: string) {
    pickFor = id
    error = ''
    await tick()
    pickerEl?.querySelector<HTMLElement>('[role=switch]')?.focus?.()
  }
  const closePicker = () => { pickFor = '' }
  function onPickerKey(e: KeyboardEvent) {
    if (e.key === 'Escape') { e.stopPropagation(); closePicker() }
  }
  // The server the last press on the library wrote. ของคุณ says so above the
  // grid and names the two placement pages, because a server that was just
  // added is one nobody has decided about yet — and this page does not decide.
  let justAdded = $state('')
  const landed = (name: string) => { justAdded = name; goPage('mine') }
  let sheetEl = $state<HTMLElement | null>(null)

  // The env / headers boxes grow with what is in them. Two rows was a cap:
  // one Windows path (HYPERFRAMES_BROWSER_PATH=C:/Users/…/chrome-headless-shell/…)
  // wrapped past it and the rest of the line was out of sight (owner, 13 ก.ย.).
  const linesFor = (text: string) => Math.max(4, text.split('\n').length + 1)
  // A preset's hint is one locale line: steps separated by ' · ', and the
  // values a person will type or see verbatim (a host, a URL, a command)
  // between backticks. That is the whole markup, so the locale stays a string
  // and the note can still set a URL apart from the sentence it sits in.
  const hintSteps = (text: string) => text.split(' · ').map((s) => s.trim()).filter(Boolean)
  const codeSegs = (step: string) => step.split('`')
  const mapToLines = (m: Record<string, string> | undefined, sep: string) =>
    Object.entries(m ?? {}).map(([k, v]) => `${k}${sep}${v}`).join('\n')
  function parseLines(text: string, sep: '=' | ':'): Record<string, string> {
    const out: Record<string, string> = {}
    for (const line of text.split('\n')) {
      const i = line.indexOf(sep)
      if (i <= 0) continue
      out[line.slice(0, i).trim()] = line.slice(i + 1).trim()
    }
    return out
  }

  // The tool list lives on the server and arrives with a connect. Opening
  // the tab on a server that has never connected starts one, so the tab
  // shows the list (or the reason there is none) rather than an instruction
  // to press a button at the other corner (owner, 13 ก.ย.: "ทำไมว่างเปล่า").
  // Not for a server that wants a login first: that connect would only 401.
  function openToolsTab() {
    sheetTab = 'tools'
    const row = sheetRow
    if (row && !row.disabled && stateOf(row) === 'idle' && !row.toolList?.length && !wantsSignIn(row) && busy === '' && !probing.has(row.name)) void testServer(row)
  }
  // Whether the ขั้นสูง fold on the conn tab is open. Closed on every open
  // unless the server already carries one of its two values.
  let advancedOpen = $state(false)
  async function openSheet(next: Sheet, tab: SheetTab = 'conn') {
    sheet = next
    sheetTab = tab
    advancedOpen = next.cwd.trim() !== '' || next.timeout.trim() !== ''
    error = ''
    await tick()
    sheetEl?.querySelector<HTMLElement>('input, select, textarea')?.focus?.()
  }
  const blankSheet = (): Sheet => ({
    original: '', kind: 'http', name: '', command: '', url: '',
    envText: '', headersText: '', cwd: '', timeout: '', picked: null, needsKey: false, hint: '',
  })
  const addServer = () => openSheet(blankSheet())
  const editServer = (s: MCPRow) => openSheet({
    original: s.name, kind: s.url ? 'http' : 'stdio', name: s.name,
    command: (s.command ?? []).join(' '), url: s.url ?? '',
    envText: mapToLines(s.environment, '='), headersText: mapToLines(s.headers, ': '),
    cwd: s.cwd ?? '', timeout: s.timeoutMs ? String(s.timeoutMs) : '',
    picked: s.allowed?.length ? new Set(s.allowed) : null, needsKey: false, hint: '',
  })
  const sheetValid = $derived(
    !!sheet && sheet.name.trim() !== '' && (sheet.kind === 'stdio' ? sheet.command.trim() !== '' : sheet.url.trim() !== ''),
  )
  const sheetRow = $derived(sheet ? servers.find((s) => s.name === sheet!.original) : undefined)
  // What the เครื่องมือ tab lists: the server's own list when it has connected
  // once, else the names already kept (all ticked), else nothing and a line
  // saying to press ทดสอบ. Tokens per row come from the engine's measure.
  const toolRows = $derived.by((): ToolCost[] => {
    if (!sheetRow) return []
    if (sheetRow.toolList?.length) return sheetRow.toolList
    return (sheetRow.allowed ?? []).map((name) => ({ name, tokens: 0 }))
  })
  const toolOn = (name: string) => !sheet?.picked || sheet.picked.has(name)
  const toolsOnCount = $derived(toolRows.filter((r) => toolOn(r.name)).length)
  const toolsOnTokens = $derived(toolRows.filter((r) => toolOn(r.name)).reduce((a, r) => a + r.tokens, 0))
  function toggleTool(name: string) {
    if (!sheet) return
    const next = new Set(sheet.picked ?? toolRows.map((r) => r.name))
    next.has(name) ? next.delete(name) : next.add(name)
    sheet.picked = next
  }
  function toggleAllTools() {
    if (!sheet) return
    sheet.picked = sheet.picked && sheet.picked.size < toolRows.length ? null : new Set()
  }

  // An auth header naming its scheme and carrying no credential can never
  // connect; refusing to save it is not a smaller failure than saving it and
  // reporting the server's 400 later (owner, 14 ส.ค., on github doing exactly
  // that). A ${env:X} / ${connect:x} reference IS a value.
  const AUTH_SCHEMES = ['bearer', 'basic', 'token', 'apikey']
  function credentiallessHeader(headers: Record<string, string>): string {
    for (const [key, value] of Object.entries(headers)) {
      const v = value.trim()
      if (/\$\{(env|connect):[^}]+\}/.test(v)) continue
      if (v === '' || AUTH_SCHEMES.includes(v.toLowerCase())) return key
    }
    return ''
  }

  const saveSheet = () => run('save', async () => {
    const f = sheet
    if (!f) return
    const headers = f.kind === 'http' ? parseLines(f.headersText, ':') : {}
    const empty = credentiallessHeader(headers)
    if (empty) throw new Error(t('settings.mcpHeaderNoValue', { header: empty }))
    const timeout = Number.parseInt(f.timeout, 10)
    await SaveMCPServer(f.original, new config.MCPServerConfig({
      name: f.name.trim(),
      command: f.kind === 'stdio' ? f.command.trim().split(/\s+/).filter(Boolean) : [],
      url: f.kind === 'http' ? f.url.trim() : '',
      environment: f.kind === 'stdio' ? parseLines(f.envText, '=') : {},
      headers,
      cwd: f.cwd.trim(),
      timeoutMs: timeout > 0 ? timeout : 0,
      // Always an array: the engine keeps its stored list when the field is
      // absent, which would make the allowlist unclearable from here.
      tools: f.picked ? [...f.picked] : [],
    }))
    sheet = null
    if (!f.original) landed(f.name.trim())
  })

  function closeSheet() {
    sheet = null
    justAdded = ''
  }
  function onSheetKey(e: KeyboardEvent) {
    if (e.key === 'Escape') { e.stopPropagation(); closeSheet() }
  }

  // ---- the library ----------------------------------------------------------
  const shelf = $derived(MCP_PRESETS.filter((p) => !taken(p.name) && (shelfFilter === '' || p.group === shelfFilter)))
  // Two bands, split by the one fact a person has to act on: whether the
  // server signs in through their own account. The band over the sign-in
  // rows used to be called "ยังไม่ได้ลองจริง" and the owner read it as "may
  // not work" (13 ก.ย.: "หมายถึงคำ ไม่ใช่เอา MCP ออก"); what is true of every
  // row there is the sign-in, so that is its name.
  const instant = $derived(shelf.filter((p) => !p.oauth))
  const signIn = $derived(shelf.filter((p) => p.oauth))
  const groupsPresent = $derived(SHELF_GROUPS.filter((g) => MCP_PRESETS.some((p) => p.group === g && !taken(p.name))))

  // A preset still waiting for a pasted token cannot be finished in one press,
  // so it opens the sheet with the header names filled in instead of saving
  // something that could never connect — or, for a program, with its env
  // lines filled in and the blanks left blank. An oauth preset goes through
  // the browser first (addOAuth); its header reads like a one-click one and
  // needsPaste alone would wave it through.
  async function add(p: MCPPreset) {
    if (p.oauth) return addOAuth(p)
    if (needsPaste(p.headers, p.env)) {
      return openSheet({
        ...blankSheet(), kind: p.url ? 'http' : 'stdio', name: p.name, url: p.url ?? '',
        command: (p.command ?? []).join(' '),
        headersText: (p.headers ?? []).map((h) => (h.includes(':') ? `${h} ` : `${h}: `)).join('\n'),
        envText: (p.env ?? []).join('\n'),
        needsKey: true, hint: p.hint ?? '',
      })
    }
    await run(p.name, async () => { await SaveMCPServer('', await presetConfig(p)) })
    landed(p.name)
  }

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
      landed(p.name)
    } catch (err) {
      error = String(err)
      signInPreset = null
    } finally {
      busy = ''
    }
  }
  // The one door out of the library that is not a card: a server this list
  // does not carry is the assistant's job to find (it reads the aetox-mcp
  // skill), and that is a chat, so the room closes behind it.
  async function askAssistant() {
    onClose()
    await startChatWith(t('capability.aiFindPrompt'))
  }
  const reopenSignIn = () => { if (signInURL) BrowserOpenURL(signInURL) }
  async function abandonSignIn(p: MCPPreset) {
    await CancelMCPSignIn(p.name)
    signInPreset = null
    signInURL = ''
    busy = ''
  }
</script>

<div class="settings-page cap-page">
  <!-- The rail: ตั้งค่า's own (.settings-nav), because the owner asked for
       "หน้าต่างย่อย เหมือนหน้าตั้งค่า" (13 ก.ย.) and DESIGN.md §1 lets a room
       borrow a frame. Four pages, one question each. The registers ตั้งค่า
       still holds (บัญชี สกิล เครื่องมือในตัว) move here whole when they
       move; they are not drawn as rows that point elsewhere (§3). -->
  <aside class="settings-nav">
    <button class="settings-back" onclick={onClose}><Icon name="arrowLeft" size={14} /> {t('settings.backToApp')}</button>
    <div class="cap-rail-title">{t('desk.capability')}</div>
    {#each RAIL as grp (grp.labelKey)}
      <div class="settings-group-label eyebrow">{t(grp.labelKey)}</div>
      {#each grp.rows as pg (pg.id)}
        <button class="settings-nav-item" class:active={page === pg.id} onclick={() => goPage(pg.id)}>
          <span class="ic"><Icon name={pg.icon} /></span> {t(pg.labelKey)}
          {#if pg.id === 'mine' && notReady.length > 0}
            <span class="nav-count" title={t('capability.navNotReady', { n: String(notReady.length) })}>{notReady.length}</span>
          {:else if pg.id === 'agents' && agentsInNeed.length > 0}
            <span class="nav-count" title={t('capability.navNeeds', { n: String(agentsInNeed.length) })}>{agentsInNeed.length}</span>
          {:else if pg.id === 'skills' && skillIssues.length > 0}
            <span class="nav-count" title={t('settings.skillIssues', { n: skillIssues.length })}>{skillIssues.length}</span>
          {/if}
        </button>
      {/each}
    {/each}
  </aside>

  <div class="settings-content">
    <div class="settings-inner wide">
      {#if error && !sheet && !pickFor}<div class="mset-error cap-error">{error}</div>{/if}

      <!-- The end of a windowed grid: a line saying what is held back, a
           button that opens it, and the same node the observer watches. Both
           doors, because a window only scrolling can open is one a keyboard
           cannot (DESIGN.md §4). -->
      {#snippet more(win: ReturnType<typeof pageWindow>, total: number)}
        {#if win.more(total)}
          <div class="cap-more" use:nearViewport={() => win.grow(total)}>
            <button class="ctrl" onclick={() => win.grow(total)}>
              {t('capability.showMore', { n: String(win.rest(total)) })}
            </button>
          </div>
        {/if}
      {/snippet}

      <!-- A server card says one thing: is it ready. Who uses it is the
           next two pages' question and is not drawn here (owner, 13 ก.ย.:
           "หน้านี้ ... ทำให้ MCP พร้อมใช้งานและลงทะเบียนกับระบบ"). -->
      {#snippet serverCard(s: MCPRow)}
        {@const st = stateOf(s)}
        {@const needsSignIn = wantsSignIn(s)}
        <div class="chair-card agc cap-srv" class:off={s.disabled}>
          <div class="chair-body">
            <div class="chair-who">
              <McpMark name={s.name} size={38} />
              <span class="chair-name"><span class="nm">{s.name}</span><span class="cap-tr">{s.url ? 'http' : 'stdio'}</span></span>
            </div>
            <p class="chair-desc" title={address(s)}>{presetOf(s.name)?.desc ?? address(s)}</p>
            <div class="chair-chips">
              <span class="cap-st {st}"><i></i>{t(stateKey[st])}</span>
              {#if st === 'connected' && s.tools > 0}<span class="cap-cost">{costLine(s.tools, s.tokens)}</span>{/if}
              {#if isOAuth(s) && !s.disabled}
                <span class="cap-st" class:connected={!needsSignIn} class:failed={needsSignIn}><i></i>{needsSignIn ? t('capability.notSignedIn') : t('settings.signedInAs')}</span>
              {/if}
            </div>
            {#if st === 'failed' && s.err}<div class="chair-stat cap-err">{s.err}</div>{/if}
          </div>
          <div class="chair-foot">
            {#if s.disabled}
              <button class="ctrl ctrl-primary cap-act" disabled={busy !== ''} onclick={() => pauseServer(s)}>
                <Icon name="plugZap" size={14} /><span class="t">{t('capability.serverResume')}</span>
              </button>
            {:else if signInPreset?.name === s.name}
              <span class="cap-signin-wait">{t('settings.signInWaiting')}</span>
              <button class="linklike" onclick={reopenSignIn}>{t('settings.signInOpenPage')}</button>
              <button class="ctrl" onclick={() => abandonSignIn(signInPreset!)}>{t('settings.signInCancel')}</button>
            {:else if needsSignIn}
              <!-- Readiness, in order: a server that wants a login cannot be
                   tested before it, so the login is the card's one action. -->
              <button class="ctrl ctrl-primary cap-act" disabled={busy !== ''} onclick={() => signInServer(s)}>
                <Icon name="globe" size={14} /><span class="t">{t('capability.signIn')}</span>
              </button>
            {:else}
              {@const testing = probing.has(s.name) || busy === 'test:' + s.name || busy === 'testall:' + s.name}
              <button class="ctrl ctrl-primary cap-act" class:testing disabled={busy !== '' || probing.has(s.name)} onclick={() => testServer(s)}>
                <Icon name="refreshCw" size={14} />
                <span class="t">{testing ? t('settings.testing') : (st === 'idle' ? t('settings.test') : t('capability.testAgain'))}</span>
              </button>
            {/if}
            <!-- The gear in a frame the size of the button beside it: a bare
                 13px glyph was too small to find ("ปุ่มตั้งค่า เล็กไป"). -->
            <button class="ctrl cap-gear tip-l" aria-label={t('capability.configure', { name: s.name })}
              data-tip={t('capability.configure', { name: s.name })} onclick={() => editServer(s)}>
              <Icon name="settings" size={16} />
            </button>
          </div>
        </div>
      {/snippet}

      <!-- A target card (a side, or an agent): what it holds, and the one
           button that opens the picker for it. Nothing on the card writes.
           Never drawn dim: `.agc.off` is the roster's "cannot work" and a grey
           mascot is a switched-off agent, but an agent with no MCP works fine
           (owner, 13 ก.ย.: "บางตัวแม้ไม่มีก็ไม่เกี่ยวสิ"). -->
      {#snippet targetCard(x: TargetRow)}
        {@const h = holds(x.id)}
        {@const miss = x.kind === 'agent' ? unmetNeeds(x.name) : []}
        {@const tok = tokensHeld(x.id)}
        <div class="chair-card agc cap-target" class:tier={x.kind === 'desk'}>
          <div class="chair-body">
            <div class="chair-who">
              {#if x.kind === 'agent'}
                <AgentMascot name={x.name} {...agentLookOf(x.name)} size={38} />
              {:else}
                {@const m = deskMeta(x.id)}
                <span class="mem-scope-ic cap-desk-ic mem-tone-{m.tone}" class:face={!!m.head}><ScopeMark meta={m} size={18} face={38} /></span>
              {/if}
              <span class="chair-name"><span class="nm">{x.kind === 'desk' ? deskMeta(x.id).label : x.name}</span></span>

            </div>
            <p class="chair-desc" title={personDesc(x)}>{x.kind === 'desk' ? deskMeta(x.id).audience : roleOf(x)}</p>
            <div class="cap-holds">
              {#each h as srv (srv.name)}
                <span class="cap-hold" title={presetOf(srv.name)?.desc ?? address(srv)}>
                  <McpMark name={srv.name} size={16} />{srv.name}
                  {#if tokensOf(srv) > 0}<span class="tk">{tokMark(srv)}</span>{/if}
                </span>
              {/each}
              {#each miss as n (n)}<span class="cap-hold need" title={t('capability.cellNeed')}>! {n}</span>{/each}
              {#if h.length === 0 && miss.length === 0}<span class="cap-hold-none">{t('capability.givenNoMcp')}</span>{/if}
            </div>
            {#if h.length > 0}
              <div class="cap-hold-total" class:est={heldEstimated(x.id)}>
                {t('capability.costTotal', { tools: String(toolsHeld(x.id)), n: (heldEstimated(x.id) ? '~' : '') + fmtTokens(tok) })}
              </div>
            {/if}
          </div>
          <div class="chair-foot">
            <!-- Primary only where the file says something is missing: the
                 eye goes to the one card that cannot work yet, not to seven
                 blue buttons. -->
            <button class="ctrl cap-act" class:ctrl-primary={miss.length > 0} disabled={busy !== ''} onclick={() => openPicker(x.id)}>
              <Icon name="plug" size={14} /><span class="t">{h.length ? t('capability.pickEdit') : t('capability.pickFor')}</span>
            </button>
          </div>
        </div>
      {/snippet}

      <!-- ================= MCP server ของคุณ ================= -->
      {#if page === 'mine'}
        <h2>{t('capability.navMine')}</h2>
        <p class="muted set-sub">{t('capability.mineLede')}</p>

        {#if loaded && servers.length === 0}
          <div class="office-grid">
            <div class="chair-card empty"><div class="chair-body">
              <p class="chair-desc"><b>{t('capability.noServers')}</b> {t('capability.noServersBody')}</p>
              <p><button class="ctrl" onclick={() => goPage('shelf')}>{t('capability.navShelf')}</button></p>
            </div></div>
          </div>
        {:else if loaded}
          <div class="sec-head">
            <!-- One sentence in place of the four tiles that stood here on
                 12 ก.ย. Each clause is a colour the cards already use for the
                 same word, and a clause that is not true is not drawn. -->
            <p class="ag-reach cap-line">
              <b>{t('capability.lineCount', { n: String(servers.length) })}</b>
              {#if probing.size > 0}
                <span class="sep">·</span><span class="w idle"><span class="spin"><Icon name="refreshCw" size={12} /></span> {t('capability.lineProbing', { n: String(probing.size) })}</span>
              {:else if idle.length === live.length && live.length > 0}
                <span class="sep">·</span><span class="w idle">{t('capability.lineIdleAll')}</span>
              {:else if idle.length > 0}
                <span class="sep">·</span><span class="w idle">{t('capability.lineIdle', { n: String(idle.length) })}</span>
              {/if}
              {#if failed.length > 0}
                <span class="sep">·</span><span class="w bad">{t('capability.lineFailed', { n: String(failed.length), names: failed.map((x) => x.name).join(', ') })}</span>
              {:else if idle.length === 0 && connected.length > 0}
                <span class="sep">·</span><span class="w ok">{t('capability.lineAllOk')}</span>
              {/if}
              {#if unsigned.length > 0}
                <span class="sep">·</span><span class="w warn">{t('capability.lineSignIn', { n: String(unsigned.length), names: unsigned.map((x) => x.name).join(', ') })}</span>
              {/if}
              {#if off.length > 0}
                <span class="sep">·</span><span class="w">{t('capability.lineOff', { n: String(off.length) })}</span>
              {/if}
            </p>
            {#if idle.length + failed.length > 0}
              <button class="ctrl" disabled={busy !== ''} onclick={testAll}>
                <Icon name="refreshCw" size={13} /> {busy.startsWith('testall:') ? t('capability.testingAll', { name: busy.slice(8) }) : t('capability.testAll')}
              </button>
            {/if}
            <button class="ctrl" disabled={busy !== ''} onclick={addServer}><Icon name="plus" size={13} /> {t('capability.addServer')}</button>
          </div>

          <!-- A server just added from the library: registered, and nobody
               has chosen it for anyone yet. The two doors are named here
               rather than opened, because this page is readiness only. -->
          {#if justAdded && servers.some((x) => x.name === justAdded)}
            <div class="cap-notice cap-added">
              <Icon name="check" size={13} />
              <span>{t('capability.justAdded', { name: justAdded })}</span>
              <button class="linklike" onclick={() => goPage('desks')}>{t('capability.navDesks')}</button>
              ·
              <button class="linklike" onclick={() => goPage('agents')}>{t('capability.navAgents')}</button>
              <button class="icobtn tiny" aria-label={t('settings.cancel')} onclick={() => (justAdded = '')}><Icon name="x" size={12} /></button>
            </div>
          {/if}

          <div class="office-grid">
            {#each winServers.take(servers) as s (s.name)}{@render serverCard(s)}{/each}
          </div>
          {@render more(winServers, servers.length)}
        {/if}
        <p class="office-note">{t('capability.mcpShelfFoot')}</p>
        <p class="office-note">
          {t('capability.toolsNote', { n: String(toolTotal) })}
          <button class="linklike" onclick={() => openSettingsAt('tools')}>{t('capability.openTools')}</button>
          {#if storedAt}
            · {t('capability.storedAt')} <span class="mono-dim">{storedAt}</span>
            <button class="linklike" onclick={() => OpenMCPFolder()}>{t('settings.skillsFolder')}</button>
          {/if}
        </p>
      {/if}

      <!-- ================= ตั้งค่า MCP ฝั่งผู้ช่วยและโค้ด ================= -->
      {#if page === 'desks'}
        <h2>{t('capability.navDesks')}</h2>
        <p class="muted set-sub">{t('capability.desksLede')}</p>
        {#if loaded}
          {#if servers.length === 0}
            <p class="office-note cap-pagenote">{t('capability.noServersForAgent')} <button class="linklike" onclick={() => goPage('shelf')}>{t('capability.navShelf')}</button></p>
          {:else if heldByNobody.length > 0}
            <p class="office-note cap-pagenote warn">{t('capability.lineNobody', { n: String(heldByNobody.length), names: heldByNobody.map((x) => x.name).join(', ') })}</p>
          {:else if inHand.length === 0 && mcpLive}
            <p class="office-note cap-pagenote">{t('capability.reachLoose')}</p>
          {/if}
          <div class="office-grid cap-tiergrid">
            {#each desks as x (x.id)}{@render targetCard(x)}{/each}
          </div>
        {/if}
        <p class="office-note">{t('capability.pickFoot')}{#if anyEstimated} {t('capability.costEstNote')}{/if}</p>
      {/if}

      <!-- ================= ตั้งค่า MCP สำหรับเอเจนเฉพาะ ================= -->
      {#if page === 'agents'}
        <h2>{t('capability.navAgents')}</h2>
        <p class="muted set-sub">{t('capability.agentsLede')}</p>
        {#if loaded && agentTargets.length === 0}
          <div class="office-grid">
            <div class="chair-card empty"><div class="chair-body"><p class="chair-desc">{t('capability.agentsNone')}</p></div></div>
          </div>
        {:else if loaded}
          <div class="office-grid">
            {#each agentTargets as x (x.id)}{@render targetCard(x)}{/each}
          </div>
        {/if}
        <p class="office-note">{t('capability.pickFoot')}{#if anyEstimated} {t('capability.costEstNote')}{/if}</p>
      {/if}

      <!-- ================= ห้องสมุด ================= -->
      {#if page === 'shelf'}
        <h2>{t('capability.navShelf')}</h2>
        <p class="muted set-sub">{t('capability.mcpShelfNote')}</p>
        {#if groupsPresent.length > 1}
          <div class="feed-filter cap-filter">
            <button class="pill" class:on={shelfFilter === ''} onclick={() => setShelfFilter('')}>{t('capability.libAll')}</button>
            {#each groupsPresent as g (g)}
              <button class="pill" class:on={shelfFilter === g} onclick={() => setShelfFilter(g)}>{t(('capability.group_' + g) as TKey)}</button>
            {/each}
          </div>
        {/if}
      {/if}

      {#snippet shelfCard(p: MCPPreset)}
        <article class="chair-card agc">
          <div class="chair-body">
            <div class="chair-who">
              <McpMark name={p.name} size={38} />
              <span class="chair-name"><span class="nm">{p.name}</span><span class="cap-tr">{p.url ? 'http' : 'stdio'}</span></span>
            </div>
            <p class="chair-desc">{p.desc}</p>
            {#if p.why}<p class="chair-desc cap-why">{p.why}</p>{/if}
            <div class="chair-chips">
              {#if p.toolCount}
                <span class="cap-cost" title={t('capability.measuredOn', { date: p.measured ?? '' })}>{costLine(p.toolCount, p.tokens)}</span>
              {/if}
              {#if needsPaste(p.headers, p.env)}<span class="chip">{t('capability.needsKey')}</span>{/if}
              {#if p.oauth}<span class="chip">{t('capability.needsSignIn')}</span>{/if}
              {#if !p.url}<span class="chip">{t('capability.onThisMachine')}</span>{/if}
            </div>
          </div>
          <div class="chair-foot">
            {#if signInPreset?.name === p.name}
              <span class="cap-signin-wait">{t('settings.signInWaiting')}</span>
              <button class="linklike" onclick={reopenSignIn}>{t('settings.signInOpenPage')}</button>
              <button class="ctrl" onclick={() => abandonSignIn(p)}>{t('settings.signInCancel')}</button>
            {:else}
              <button class="ctrl ctrl-primary cap-act" disabled={busy !== ''} onclick={() => add(p)}>
                <Icon name="plus" size={14} />
                <span class="t">{busy === p.name ? t('capability.adding') : t('capability.add')}</span>
              </button>
            {/if}
          </div>
        </article>
      {/snippet}

      {#if page === 'shelf'}
      {#if instant.length > 0}
        <div class="ag-band">
          <span class="lab">{t('capability.bandInstant')}</span><span class="n">{instant.length}</span>
          <span class="rule"></span><span class="say">{t('capability.bandInstantNote')}</span>
        </div>
        <div class="office-grid">
          {#each winShelf.take(instant) as p (p.name)}{@render shelfCard(p)}{/each}
        </div>
        {@render more(winShelf, instant.length)}
      {/if}
      {#if signIn.length > 0}
        <div class="ag-band" class:cap-sec={instant.length > 0}>
          <span class="lab">{t('capability.bandSignIn')}</span><span class="n">{signIn.length}</span>
          <span class="rule"></span><span class="say">{t('capability.bandSignInNote')}</span>
        </div>
        <div class="office-grid">
          {#each winSignIn.take(signIn) as p (p.name)}{@render shelfCard(p)}{/each}
        </div>
        {@render more(winSignIn, signIn.length)}
      {/if}
      {#if shelf.length === 0}
        <p class="office-note">{t('capability.shelfEmpty')}</p>
      {/if}
      <p class="office-note">
        {t('capability.libNote')}
        <button class="linklike" onclick={addServer}>{t('capability.addByAddress')}</button>
        · <button class="linklike" onclick={askAssistant}>{t('capability.aiFind')}</button>
      </p>
      <p class="office-note">{t('capability.mcpShelfFoot')}</p>
      {/if}

      <!-- A skill card says what it is and where it is. The bundled band says
           once what a bundled row used to repeat thirty times. -->
      {#snippet skillCard(s: SkillRow)}
        <div class="chair-card agc cap-srv cap-skill">
          <div class="chair-body">
            <div class="chair-who">
              <span class="cap-mark" style="--px:38px; --h:{coverHue(s.name)}" aria-hidden="true">{skillMark(s.name)}</span>
              <span class="chair-name"><span class="nm">{s.name}</span></span>
            </div>
            <p class="chair-desc" title={s.description}>{s.description || '—'}</p>
            <div class="chair-chips">
              {#if s.before}<span class="cap-cost cap-before" title={t('capability.skillBeforeTip')}><Icon name="clock" size={11} />{t('capability.skillBefore', { work: s.before })}</span>{/if}
              {#if !s.bundled}<span class="cap-cost mono-dim" title={s.dir}>{skillDirShort(s.dir)}</span>{/if}
            </div>
          </div>
          {#if !s.bundled}
            <div class="chair-foot">
              <button class="ctrl cap-act" onclick={() => OpenSkillsFolder()}><Icon name="folderOpen" size={14} /><span class="t">{t('settings.skillsFolder')}</span></button>
              <button class="icobtn tiny" aria-label={t('settings.remove')} disabled={busy !== ''} onclick={() => (confirmSkill = s)}><Icon name="trash" size={13} /></button>
            </div>
          {/if}
        </div>
      {/snippet}

      <!-- ================= สกิลของคุณ ================= -->
      {#if page === 'skills'}
        <h2>{t('capability.navSkills')}</h2>
        <p class="muted set-sub">{t('capability.skillsLede')}</p>
        {#if loaded}
          <div class="sec-head">
            <p class="ag-reach cap-line">
              <b>{t('settings.skillCount', { n: String(skills.length) })}</b>
              <span class="sep">·</span><span class="w idle">{t('capability.skillLineBundled', { n: String(shelfBundled.length) })}</span>
              <span class="sep">·</span><span class="w ok">{t('capability.skillLineMine', { n: String(shelfMine.length) })}</span>
              {#if skillIssues.length > 0}<span class="sep">·</span><span class="w bad">{t('capability.skillLineIssues', { n: String(skillIssues.length) })}</span>{/if}
            </p>
            <button class="ctrl" disabled={busy !== ''} onclick={refreshSkills}><Icon name="refreshCw" size={13} /> {busy === 'refresh-skills' ? t('settings.refreshing') : t('settings.refresh')}</button>
            <button class="ctrl" onclick={() => OpenSkillsFolder()}><Icon name="folderOpen" size={13} /> {t('settings.skillsFolder')}</button>
            <button class="ctrl ctrl-primary" onclick={() => (skillInstall = true)}><Icon name="plus" size={13} /> {t('capability.skillInstallTitle')}</button>
          </div>
          <!-- Files in the right folder that still did not appear: a broken
               SKILL.md looks exactly like a folder nobody is scanning. -->
          {#each skillIssues as issue, i (i)}
            <div class="cap-notice cap-issue">
              <Icon name="alertTriangle" size={13} />
              <span>{t('capability.skillIssueLine')} <span class="mono-dim">{issue}</span></span>
            </div>
          {/each}
          <div class="ag-band">
            <span class="lab">{t('capability.bandMine')}</span><span class="n">{shelfMine.length}</span>
            <span class="rule"></span><span class="say">{t('capability.bandMineNote', { dir: skillsDir })}</span>
          </div>
          {#if shelfMine.length === 0}
            <div class="office-grid">
              <div class="chair-card empty"><div class="chair-body">
                <p class="chair-desc">{t('settings.noSkills')}</p>
                <p><button class="ctrl" onclick={() => goPage('skshelf')}>{t('capability.navSkillShelf')}</button></p>
              </div></div>
            </div>
          {:else}
            <div class="office-grid">
              {#each winMine.take(shelfMine) as s (s.dir || s.name)}{@render skillCard(s)}{/each}
            </div>
            {@render more(winMine, shelfMine.length)}
          {/if}
          <div class="ag-band cap-sec">
            <span class="lab">{t('capability.bandBundled')}</span><span class="n">{shelfBundled.length}</span>
            <span class="rule"></span><span class="say">{t('capability.bandBundledNote')}</span>
          </div>
          <div class="office-grid">
            {#each winBundled.take(shelfBundled) as s (s.dir || s.name)}{@render skillCard(s)}{/each}
          </div>
          {@render more(winBundled, shelfBundled.length)}
          <p class="office-note foot">{t('capability.skillsFoot')}</p>
        {/if}
      {/if}

      <!-- ================= ตั้งค่าสกิลสำหรับเอเจนเฉพาะ ================= -->
      {#if page === 'skagents'}
        <h2>{t('capability.navSkillAgents')}</h2>
        <p class="muted set-sub">{t('capability.skillAgentsLede')}</p>
        {#if loaded && agentTargets.length === 0}
          <div class="office-grid">
            <div class="chair-card empty"><div class="chair-body"><p class="chair-desc">{t('capability.agentsNone')}</p></div></div>
          </div>
        {:else if loaded}
          <div class="office-grid">
            {#each agentTargets as x (x.id)}
              {@const own = ownOf(x.name)}
              <div class="chair-card agc cap-target cap-skagent">
                <div class="chair-body">
                  <div class="chair-who">
                    <AgentMascot name={x.name} {...agentLookOf(x.name)} size={38} />
                    <span class="chair-name"><span class="nm">{x.name}</span></span>
                  </div>
                  <p class="chair-desc" title={personDesc(x)}>{roleOf(x)}</p>
                  <div class="cap-holds">
                    {#each own as s (s.name)}
                      <span class="cap-hold" class:mine={!s.bundled} title={s.description}>
                        <span class="cap-mark" style="--px:16px; --h:{coverHue(s.name)}" aria-hidden="true">{skillMark(s.name)}</span>{shortSkill(s.name)}
                      </span>
                    {/each}
                    {#if own.length === 0}<span class="cap-hold-none">{t('capability.agentSkillsNone')}</span>{/if}
                  </div>
                  {#if own.length > 0}<div class="cap-hold-total">{t('capability.agentSkillsCount', { n: String(own.length) })}</div>{/if}
                </div>
                <div class="chair-foot">
                  <button class="ctrl cap-act" class:ctrl-primary={own.length === 0} disabled={busy !== ''} onclick={() => (skillPick = x.name)}>
                    <Icon name="puzzle" size={14} /><span class="t">{own.length ? t('capability.agentSkillEdit', { name: x.name }) : t('capability.agentSkillAdd', { name: x.name })}</span>
                  </button>
                  <button class="icobtn tiny" aria-label={t('settings.agentSkillsOpenFolder')} onclick={() => OpenAgentSkillsFolder(x.name)}><Icon name="folderOpen" size={13} /></button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
        <p class="office-note foot">{t('capability.skillAgentsFoot')}</p>
      {/if}

      <!-- ================= ห้องสมุดสกิล ================= -->
      {#if page === 'skshelf'}
        <h2>{t('capability.navSkillShelf')}</h2>
        <p class="muted set-sub">{t('settings.skillShelfNote')}</p>
        <div class="ag-band">
          <span class="lab">{t('capability.bandOpenSource')}</span><span class="n">{SKILL_PRESETS.length}</span>
          <span class="rule"></span><span class="say">{t('capability.bandOpenSourceNote')}</span>
        </div>
        <div class="office-grid">
          {#each winSkShelf.take(SKILL_PRESETS) as p (p.name)}
            {@const got = installedOf(p)}
            <article class="chair-card agc">
              <div class="chair-body">
                <div class="chair-who">
                  <span class="cap-mark" style="--px:38px; --h:{coverHue(p.name)}" aria-hidden="true">{p.name.split('/').map((part) => part[0]).join('')}</span>
                  <span class="chair-name"><span class="nm">{p.name}</span><span class="cap-tr">{p.licence}</span></span>
                </div>
                <p class="chair-desc">{p.desc}</p>
                <p class="chair-desc cap-why">{p.why}</p>
                <div class="chair-chips">
                  <span class="cap-cost">{t('settings.skillWrites', { n: String(p.installs.length), kb: String(p.kb) })}</span>
                  {#if got > 0 && got < p.installs.length}<span class="chip">{t('settings.skillPartial', { n: String(got), m: String(p.installs.length) })}</span>{/if}
                </div>
              </div>
              <div class="chair-foot">
                {#if got === p.installs.length}
                  <span class="cap-st connected"><i></i>{t('settings.skillAdded')}</span>
                {:else}
                  <button class="ctrl ctrl-primary cap-act" disabled={busy !== ''} onclick={() => installPack(p)}>
                    <Icon name="plus" size={14} /><span class="t">{busy === 'pack:' + p.name ? t('settings.installing') : t('capability.skillAddPack')}</span>
                  </button>
                {/if}
                <button class="icobtn tiny" aria-label={p.repo} onclick={() => BrowserOpenURL(p.repo)}><Icon name="externalLink" size={13} /></button>
              </div>
            </article>
          {/each}
        </div>
        {@render more(winSkShelf, SKILL_PRESETS.length)}
        {#if skillInstallResult}<pre class="skill-result cap-sec">{skillInstallResult}</pre>{/if}
        <p class="office-note cap-sec">
          {t('capability.skillLibNote')}
          <button class="linklike" onclick={() => (skillInstall = true)}>{t('capability.skillInstallLink')}</button>
          · <button class="linklike" onclick={askSkillAssistant}>{t('settings.aiFindSkillTitle')}</button>
        </p>
        <p class="office-note">{t('settings.skillShelfFoot')}</p>
      {/if}
    </div>
  </div>

  <!-- Both sheets sit inside the frame on purpose: style.css scopes .ctrl to
       .settings-page / .page-shell, and a sheet mounted beside the page drew
       native white inputs and bare buttons (owner, 13 ก.ย.: "CSS ไม่ครบ"). -->

<!-- The picker: "this one, which servers?" for a side or an agent. The one
     writer of `for:` in the app (`put`), opened from a target card on either
     placement page. A switched-off server is drawn and cannot be ticked. -->
{#if pickFor && pickTarget}
  <div class="cap-sheet-overlay" role="presentation" onkeydown={onPickerKey}>
    <button class="cap-sheet-backdrop" aria-label={t('settings.cancel')} onclick={closePicker}></button>
    <div class="cap-sheet cap-sheet-pick" role="dialog" aria-modal="true" aria-labelledby="cap-pick-title" bind:this={pickerEl}>
      <div class="cap-sheet-head">
        {#if pickTarget.kind === 'agent'}
          <AgentMascot name={pickTarget.name} {...agentLookOf(pickTarget.name)} size={30} />
        {:else}
          {@const m = deskMeta(pickTarget.id)}
          <span class="mem-scope-ic cap-desk-ic mem-tone-{m.tone}" class:face={!!m.head}><ScopeMark meta={m} size={16} face={30} /></span>
        {/if}
        <h3 id="cap-pick-title">{t('capability.pickTitle', { name: targetName(pickFor) })}</h3>
        <button class="icobtn" aria-label={t('settings.cancel')} onclick={closePicker}><Icon name="x" size={15} /></button>
      </div>
      <div class="cap-sheet-body">
        <p class="cap-means cap-pick-desc">{personDesc(pickTarget)}</p>
        {#if servers.length === 0}
          <p class="cap-means">{t('capability.noServersForAgent')}</p>
          <p><button class="ctrl" onclick={() => { closePicker(); goPage('shelf') }}>{t('capability.navShelf')}</button></p>
        {:else}
          <div class="cap-pickgroup">
            <span class="lab">{t('capability.navMine')}</span>
            <span class="cnt">{holds(pickFor).length}/{live.length}</span>
            <span class="rule"></span>
            <button class="linklike" disabled={busy !== ''} onclick={() => toggleAllFor(pickFor)}>
              {holds(pickFor).length === live.length ? t('capability.toolsNone') : t('capability.toolsAll')}
            </button>
          </div>
          <!-- One row per server, the roster's card width: name, what it is,
               what it adds per message, the tick. A chip was too small to
               carry the cost, and the cost is the thing to weigh here. -->
          <div class="cap-pickrows">
            {#each servers as s (s.name)}
              {@const on = isOn(s, pickFor)}
              {@const need = !on && pickTarget.kind === 'agent' && needsServer(pickTarget.name, s.name)}
              <button
                class="cap-pickrow" class:need class:dead={s.disabled}
                role="switch" aria-checked={on}
                title={s.disabled ? t('capability.cellDead') : need ? t('capability.cellNeed') : ''}
                disabled={busy !== '' || s.disabled}
                onclick={() => toggleTarget(s, pickFor)}
              >
                <McpMark name={s.name} size={28} />
                <span class="txt">
                  <span class="nm">{s.name}{#if need}<span class="needtag">{t('capability.needTag')}</span>{/if}</span>
                  <span class="d">{s.disabled ? t('capability.stOff') : (presetOf(s.name)?.desc ?? address(s))}</span>
                </span>
                <span class="cost">
                  {#if tokensOf(s) > 0}
                    <b>{tokMark(s)}</b><small>{t('capability.tokensUnit')}</small>
                    {#if toolsOf(s) > 0}<small>{t('settings.mcpToolCount', { n: String(toolsOf(s)) })}</small>{/if}
                  {:else}
                    <small>{t('capability.costUnknown')}</small>
                  {/if}
                </span>
                <span class="cap-pick-tick" aria-hidden="true">
                  {#if on}<Icon name="check" size={14} />{:else if need}<Icon name="wrench" size={12} />{/if}
                </span>
              </button>
            {/each}
          </div>
          <p class="cap-means cap-pick-sum" class:est={heldEstimated(pickFor)}>
            {t('capability.pickInstant', { name: targetName(pickFor), cost: t('capability.costTotal', { tools: String(toolsHeld(pickFor)), n: (heldEstimated(pickFor) ? '~' : '') + fmtTokens(tokensHeld(pickFor)) }) })}
            {#if anyEstimated}{t('capability.costEstNote')}{/if}
          </p>
        {/if}
        {#if error}<div class="mset-error">{error}</div>{/if}
      </div>
    </div>
  </div>
{/if}

<!-- The sheet. The register's form, whole, and the only copy of it. -->
{#if sheet}
  <div class="cap-sheet-overlay" role="presentation" onkeydown={onSheetKey}>
    <button class="cap-sheet-backdrop" aria-label={t('settings.cancel')} onclick={closeSheet}></button>
    <div class="cap-sheet" role="dialog" aria-modal="true" aria-labelledby="cap-sheet-title" bind:this={sheetEl}>
      <div class="cap-sheet-head">
        {#if sheet.original}<McpMark name={sheet.original} size={30} />{/if}
        <h3 id="cap-sheet-title">{sheet.original ? t('capability.sheetEdit', { name: sheet.original }) : t('capability.addServer')}</h3>
        <button class="icobtn" aria-label={t('settings.cancel')} onclick={closeSheet}><Icon name="x" size={15} /></button>
      </div>

      {#if sheetRow}
        {@const st = stateOf(sheetRow)}
        <div class="cap-sheet-status">
          <span class="cap-st {st}"><i></i>{t(stateKey[st])}</span>
          {#if sheetRow.tools > 0}<span class="cap-cost">{costLine(sheetRow.tools, sheetRow.tokens)}</span>{/if}
          {#if st === 'failed' && sheetRow.err}<span class="cap-err">{sheetRow.err}</span>{/if}
          <span class="grow"></span>
          {#if !sheetRow.disabled && wantsSignIn(sheetRow)}
            <button class="ctrl" disabled={busy !== ''} onclick={() => signInServer(sheetRow!)}>{t('capability.signIn')}</button>
          {:else if !sheetRow.disabled}
            <button class="ctrl" disabled={busy !== ''} onclick={() => testServer(sheetRow!)}>
              {busy === 'test:' + sheetRow.name ? t('settings.testing') : t('settings.test')}
            </button>
          {/if}
        </div>
      {/if}

      <div class="cap-tabs" role="tablist">
        <button role="tab" aria-selected={sheetTab === 'conn'} class:on={sheetTab === 'conn'} onclick={() => (sheetTab = 'conn')}>{t('capability.tabConn')}</button>
        <button role="tab" aria-selected={sheetTab === 'tools'} class:on={sheetTab === 'tools'} disabled={!sheetRow} onclick={openToolsTab}>
          {t('capability.tabTools')}{#if sheetRow && toolRows.length}<span class="c">{toolsOnCount}</span>{/if}
        </button>
      </div>

      <div class="cap-sheet-body">
        {#if sheetTab === 'tools' && sheetRow}
          {#if toolRows.length === 0}
            {@const st = stateOf(sheetRow)}
            <!-- No list yet. Say which of the four reasons, and put the way
                 out in the same place, not in the corner. -->
            <div class="cap-tools-empty">
              {#if busy === 'test:' + sheetRow.name || probing.has(sheetRow.name)}
                <span class="spin"><Icon name="refreshCw" size={14} /></span>
                <span>{t('capability.toolsFetching', { name: sheetRow.name })}</span>
              {:else if sheetRow.disabled}
                <span>{t('capability.cellDead')}</span>
                <button class="ctrl" disabled={busy !== ''} onclick={() => pauseServer(sheetRow!)}>{t('capability.serverResume')}</button>
              {:else if wantsSignIn(sheetRow)}
                <!-- The sign-in button and the error already sit in the
                     status row two lines up; the tab says why it is empty
                     and does not grow a second copy of either. -->
                <span>{t('capability.toolsNeedSignIn', { name: sheetRow.name })}</span>
              {:else if st === 'failed'}
                <span>{t('capability.toolsFailed', { name: sheetRow.name })}</span>
              {:else if st === 'connected'}
                <span>{t('capability.toolsNoneOffered', { name: sheetRow.name })}</span>
              {:else}
                <span>{t('capability.toolsNoneYet')}</span>
                <button class="ctrl ctrl-primary" disabled={busy !== ''} onclick={() => testServer(sheetRow!)}>{t('settings.test')}</button>
              {/if}
            </div>
          {:else}
            <div class="cap-toolhead">
              <span>{t('capability.toolsKept', { n: String(toolsOnCount), total: String(toolRows.length) })}{#if toolsOnTokens > 0}{' · '}{t('capability.tokensPerMsg', { n: fmtTokens(toolsOnTokens) })}{/if}</span>
              <span class="grow"></span>
              <button class="linklike" onclick={toggleAllTools}>{sheet.picked && sheet.picked.size < toolRows.length ? t('capability.toolsAll') : t('capability.toolsNone')}</button>
            </div>
            <div class="cap-toollist" role="group" aria-label={t('capability.tabTools')}>
              {#each toolRows as r (r.name)}
                <label class="cap-toolrow" class:on={toolOn(r.name)}>
                  <input type="checkbox" checked={toolOn(r.name)} onchange={() => toggleTool(r.name)} />
                  <span class="nm">{r.name}</span>
                  {#if r.tokens > 0}<span class="tk">~{fmtTokens(r.tokens)}</span>{/if}
                </label>
              {/each}
            </div>
            <p class="cap-means">{sheetRow.toolList?.length ? t('capability.toolsHint') : t('capability.toolsStoredHint')}</p>
          {/if}

        {:else}
          <div class="mset-keyrow">
            <select class="ctrl mcp-kind" bind:value={sheet.kind} disabled={!!sheet.original}>
              <option value="stdio">stdio</option>
              <option value="http">http</option>
            </select>
            <input class="ctrl key-input" placeholder={t('settings.mcpNamePlaceholder')} bind:value={sheet.name} />
          </div>
          {#if sheet.kind === 'stdio'}
            <input class="ctrl" placeholder={t('settings.mcpCommandPlaceholder')} bind:value={sheet.command} />
            <textarea class="ctrl mcp-lines" rows={linesFor(sheet.envText)} placeholder={t('settings.mcpEnvPlaceholder')} bind:value={sheet.envText}></textarea>
          {:else}
            <input class="ctrl" placeholder={t('settings.mcpUrlPlaceholder')} bind:value={sheet.url} />
            <textarea class="ctrl mcp-lines" rows={linesFor(sheet.headersText)} placeholder={t('settings.mcpHeadersPlaceholder')} bind:value={sheet.headersText}></textarea>
          {/if}
          <div class="d muted">{t('settings.mcpSecretHint')}</div>
          <!-- What a preset still needs from the person, as one note rather
               than three muted paragraphs stacked under the form (owner,
               13 ก.ย.: "ทำ CSS ดีๆหน่อย"): the title says which server and
               what kind of blank, the steps say where the values come from
               when the preset carries a hint, and the foot says nothing is
               saved until เพิ่ม is pressed. -->
          {#if sheet.needsKey}
            <div class="cap-keynote">
              <div class="cap-keynote-t">{t(sheet.kind === 'stdio' ? 'settings.mcpNeedsEnv' : 'settings.mcpNeedsKey', { name: sheet.name })}</div>
              {#if sheet.hint}
                <ol class="cap-keynote-steps">
                  {#each hintSteps(t(sheet.hint)) as step, i (i)}
                    <li>{#each codeSegs(step) as seg, j (j)}{#if j % 2}<code>{seg}</code>{:else}{seg}{/if}{/each}</li>
                  {/each}
                </ol>
              {:else}
                <p>{t(sheet.kind === 'stdio' ? 'settings.mcpNeedsEnvHow' : 'settings.mcpNeedsKeyHow')}</p>
              {/if}
              <div class="cap-keynote-f">{t('settings.mcpNothingSaved')}</div>
            </div>
          {/if}

          <!-- ขั้นสูง folds (owner, 13 ก.ย.: "ทำเป็นปุ่มซ่อนไว้ กดพับเปิดได้ก็พอ").
               It opens by itself when either field holds a value, so a stored
               setting is never out of sight behind a closed fold. -->
          <button type="button" class="cap-fold" aria-expanded={advancedOpen} onclick={() => (advancedOpen = !advancedOpen)}>
            <Icon name={advancedOpen ? 'chevronDown' : 'chevronRight'} size={13} />
            <span class="eyebrow">{t('capability.sheetAdvanced')}</span>
            {#if !advancedOpen && (sheet.cwd.trim() || sheet.timeout.trim())}
              <span class="cap-fold-set">{[sheet.cwd.trim() && t('settings.mcpCwd'), sheet.timeout.trim() && t('settings.mcpTimeout')].filter(Boolean).join(' · ')}</span>
            {/if}
          </button>
          {#if advancedOpen}
            <div class="cap-two">
              <label class="pp-field">
                <span class="eyebrow">{t('settings.mcpCwd')}</span>
                <input class="ctrl" placeholder={t('settings.mcpCwdPlaceholder')} bind:value={sheet.cwd} />
              </label>
              <label class="pp-field">
                <span class="eyebrow">{t('settings.mcpTimeout')}</span>
                <div class="mset-keyrow">
                  <input class="ctrl set-num" inputmode="numeric" placeholder="0" bind:value={sheet.timeout} />
                  <span class="muted set-unit">ms</span>
                </div>
              </label>
            </div>
            <span class="d muted">{t('settings.mcpTimeoutHint')}</span>
          {/if}
        {/if}
        {#if error}<div class="mset-error">{error}</div>{/if}
      </div>

      <div class="cap-sheet-foot">
        <button class="ctrl ctrl-primary" disabled={busy !== '' || !sheetValid} onclick={saveSheet}>
          {busy === 'save' ? t('settings.saving') : (sheet.original ? t('settings.save') : t('settings.add'))}
        </button>
        {#if sheetRow}
          <button class="ctrl" disabled={busy !== ''} onclick={() => pauseServer(sheetRow!)}>
            {sheetRow.disabled ? t('capability.serverResume') : t('capability.serverPause')}
          </button>
          <span class="grow"></span>
          <button class="ctrl ctrl-danger" disabled={busy !== ''} onclick={() => (confirmRemove = sheetRow!.name)}>{t('settings.remove')}</button>
        {/if}
      </div>
    </div>
  </div>
{/if}
</div>

<!-- The agent's skill sheet: what its folder holds, and the shelf to tick
     from. A tick is a copy into the folder (CopySkillToAgent); unticking a
     copy removes it. What shipped with the profile is listed, not tickable. -->
{#if skillPick}
  {@const target = agentTargets.find((x) => x.name === skillPick)}
  {@const own = ownOf(skillPick)}
  {@const shipped = own.filter((x) => x.bundled)}
  <div class="cap-sheet-overlay" role="presentation" onkeydown={(e) => { if (e.key === 'Escape') skillPick = '' }}>
    <button class="cap-sheet-backdrop" aria-label={t('settings.cancel')} onclick={() => (skillPick = '')}></button>
    <div class="cap-sheet cap-sheet-pick" role="dialog" aria-modal="true" aria-labelledby="cap-skill-title">
      <div class="cap-sheet-head">
        <AgentMascot name={skillPick} {...agentLookOf(skillPick)} size={30} />
        <h3 id="cap-skill-title">{t('capability.skillSheetTitle', { name: skillPick })}</h3>
        <button class="icobtn" aria-label={t('settings.cancel')} onclick={() => (skillPick = '')}><Icon name="x" size={15} /></button>
      </div>
      <div class="cap-sheet-body">
        {#if target}<p class="cap-means cap-pick-desc">{personDesc(target)}</p>{/if}
        <div class="cap-pickgroup">
          <span class="lab">{t('capability.skillOwnGroup')}</span><span class="cnt">{shipped.length}</span>
          <span class="rule"></span>
          <button class="linklike" onclick={() => OpenAgentSkillsFolder(skillPick)}>{t('settings.agentSkillsOpenFolder')}</button>
        </div>
        {#if shipped.length > 0}
          <div class="cap-picks">
            {#each shipped as s (s.name)}
              <span class="cap-hold" title={s.description}><span class="cap-mark" style="--px:16px; --h:{coverHue(s.name)}" aria-hidden="true">{skillMark(s.name)}</span>{shortSkill(s.name)}</span>
            {/each}
          </div>
          <p class="cap-means">{t('capability.skillOwnShipped')}</p>
        {:else}
          <p class="cap-means">{t('capability.skillOwnEmpty')}</p>
        {/if}
        <div class="cap-pickgroup">
          <span class="lab">{t('capability.skillCopyGroup')}</span>
          <span class="cnt">{own.filter((x) => !x.bundled).length}/{skills.length}</span>
          <span class="rule"></span>
        </div>
        <p class="cap-means">{t('capability.skillCopyNote', { name: skillPick })}</p>
        <div class="cap-pickrows">
          {#each skills as s (s.name)}
            {@const on = hasOwn(skillPick, s.name)}
            <button class="cap-pickrow" role="switch" aria-checked={on} disabled={busy !== ''} onclick={() => toggleAgentSkill(skillPick, s)}>
              <span class="cap-mark" style="--px:28px; --h:{coverHue(s.name)}" aria-hidden="true">{skillMark(s.name)}</span>
              <span class="txt">
                <span class="nm">{s.name}</span>
                <span class="d">{s.description}</span>
              </span>
              <span class="cost"><small>{s.bundled ? t('capability.bundled') : t('capability.skillOwnUser')}</small></span>
              <span class="cap-pick-tick" aria-hidden="true">{#if on}<Icon name="check" size={14} />{/if}</span>
            </button>
          {/each}
        </div>
        {#if error}<div class="mset-error">{error}</div>{/if}
      </div>
    </div>
  </div>
{/if}

<!-- The install sheet: the three roads onto the shelf, whole, as ตั้งค่า
     had them — GitHub, a zip, or a folder dropped by hand. -->
{#if skillInstall}
  <div class="cap-sheet-overlay" role="presentation" onkeydown={(e) => { if (e.key === 'Escape') skillInstall = false }}>
    <button class="cap-sheet-backdrop" aria-label={t('settings.cancel')} onclick={() => (skillInstall = false)}></button>
    <div class="cap-sheet" role="dialog" aria-modal="true" aria-labelledby="cap-skill-install">
      <div class="cap-sheet-head">
        <span class="cap-mark" style="--px:30px; --h:210" aria-hidden="true"><Icon name="plus" size={14} /></span>
        <h3 id="cap-skill-install">{t('capability.skillInstallTitle')}</h3>
        <button class="icobtn" aria-label={t('settings.cancel')} onclick={() => (skillInstall = false)}><Icon name="x" size={15} /></button>
      </div>
      <div class="cap-sheet-body">
        <div class="cap-pickgroup"><span class="lab">{t('settings.skillInstall')}</span><span class="rule"></span></div>
        <div class="mset-keyrow">
          <input
            class="ctrl key-input" placeholder={t('settings.skillInstallPlaceholder')}
            bind:value={skillInstallUrl}
            onkeydown={(e) => e.key === 'Enter' && skillInstallUrl.trim() && installSkillURL()}
          />
          <button class="ctrl ctrl-primary" disabled={busy !== '' || !skillInstallUrl.trim()} onclick={installSkillURL}>
            {busy === 'install-skill' ? t('settings.installing') : t('settings.install')}
          </button>
        </div>
        <p class="cap-means">{t('settings.skillInstallHint')}</p>
        <div class="cap-pickgroup"><span class="lab">{t('capability.skillFromZip')}</span><span class="rule"></span></div>
        <div class="mset-keyrow">
          <p class="cap-means eyebrow-grow">{t('settings.skillZipHint')}</p>
          <button class="ctrl" disabled={busy !== ''} onclick={installSkillZip}>
            <Icon name="upload" size={13} /> {busy === 'install-zip' ? t('settings.installing') : t('settings.skillZip')}
          </button>
        </div>
        <div class="cap-pickgroup"><span class="lab">{t('capability.skillByHand')}</span><span class="rule"></span></div>
        <p class="cap-means">{t('capability.skillByHandNote')} <span class="mono-dim">{skillsDir}</span></p>
        <div class="mset-keyrow"><button class="ctrl" onclick={() => OpenSkillsFolder()}><Icon name="folderOpen" size={13} /> {t('settings.skillsFolder')}</button></div>
        {#if skillInstallResult}<pre class="skill-result">{skillInstallResult}</pre>{/if}
        {#if error}<div class="mset-error">{error}</div>{/if}
      </div>
    </div>
  </div>
{/if}

{#if confirmSkill}
  <ConfirmDialog
    title={t('settings.confirmSkillTitle')}
    message={t('settings.confirmSkillMessage')}
    detail={confirmSkill.dir || confirmSkill.name}
    confirmLabel={t('settings.remove')}
    onConfirm={skillConfirmed}
    onCancel={() => (confirmSkill = null)}
  />
{/if}

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
  .cap-error { margin-bottom: 14px; }
  /* The rail's title, under the back button, where ตั้งค่า puts its search
     box: this rail has three rows and nothing to search. */
  .cap-rail-title { padding: 10px 10px 0; font-size: var(--fs-xl, 17px); font-weight: 600; color: var(--text-primary); }
  .cap-page .settings-nav .settings-group-label:first-of-type { padding-top: 22px; }
  .cap-page .settings-nav .settings-nav-item + .settings-group-label { margin-top: 18px; }
  .cap-filter { margin: -4px 0 12px; }
  .cap-sec { margin-top: 36px; }
  /* The brand tile on a card: style.css paints .cap-mark.logo on
     --surface-raised, which is the card's own ground, so the logo sat on
     nothing and only showed once hover lifted the card ("โลโก้ควรจะชัดตั้งแต่
     แรก ไม่ใช่ต้องรอเอาเมาส์ไปสัมผัส", 13 ก.ย.). A plate one step down, with
     its own edge, at every size the room draws it. */
  .cap-page .chair-card .cap-mark.logo { background: var(--surface-sunken); box-shadow: inset 0 0 0 1px var(--border-default); }
  .cap-tr {
    font-family: var(--mono); font-size: var(--fs-2xs); font-weight: 400;
    color: var(--text-dim); border: 1px solid var(--border-default);
    border-radius: var(--r-xs); padding: 0 4px; margin-left: 6px;
  }
  .cap-why { color: var(--text-muted); }
  /* Status as a word with a colour, in the same slot on every card. The dot
     repeats the colour in shape so it survives the wrong eyes (DESIGN.md §4). */
  .cap-st { display: inline-flex; align-items: center; gap: 6px; font-size: var(--fs-2xs); font-weight: 600; color: var(--text-muted); }
  .cap-st i { width: 7px; height: 7px; border-radius: 50%; background: var(--text-dim); display: inline-block; }
  .cap-st.connected { color: var(--status-success); } .cap-st.connected i { background: var(--status-success); }
  .cap-st.failed { color: var(--status-danger); } .cap-st.failed i { background: var(--status-danger); }
  .cap-st.off { color: var(--text-dim); } .cap-st.off i { background: transparent; border: 1.5px solid var(--text-dim); }
  .cap-cost { font-size: var(--fs-2xs); color: var(--text-muted); font-variant-numeric: tabular-nums; }
  .cap-err { color: var(--status-danger); font-family: var(--mono); word-break: break-word; }
  .cap-signin-wait { font-size: var(--fs-xs); color: var(--text-muted); }
  /* ---- skills ---- */
  .cap-before { display: inline-flex; align-items: center; gap: 4px; }
  .cap-issue { color: var(--status-warn); background: color-mix(in srgb, var(--status-warn) 10%, transparent); border-color: color-mix(in srgb, var(--status-warn) 35%, transparent); margin-bottom: 10px; flex-wrap: wrap; }
  .cap-issue + .ag-band { margin-top: 6px; }
  .cap-hold .cap-mark { font-size: 8px; }
  .cap-hold.mine { border-color: color-mix(in srgb, var(--accent) 45%, transparent); color: var(--accent-bright); }
  .skill-result.cap-sec { margin-bottom: 0; }

  /* The card's one action, at the foot's full width. .ctrl-primary is the
     app's own "the one action the row exists for" (style.css button
     hierarchy); the roster's grey .chair-talk was read as blending in. */
  .cap-act { flex: 1; min-width: 0; display: inline-flex; align-items: center; justify-content: center; gap: 7px; height: 32px; }
  .cap-act .t { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .cap-act.testing svg { animation: cap-spin .9s linear infinite; }
  .cap-line .spin { display: inline-flex; vertical-align: -2px; }
  .cap-line .spin svg { animation: cap-spin .9s linear infinite; }
  /* The gear: a framed square as tall as the action beside it. */
  .cap-gear { flex: none; width: 38px; height: 32px; padding: 0; display: grid; place-items: center; color: var(--text-muted); }
  .cap-gear:hover { color: var(--text-primary); border-color: var(--border-strong); }
  @media (prefers-reduced-motion: reduce) { .cap-act.testing svg, .cap-line .spin svg { animation: none; } }

  /* ---- ของคุณ ---- */
  /* One sentence in place of four tiles. Each clause is the colour the cards
     use for the same word, so the line and the grid agree. */
  .cap-line { margin: 0; line-height: 1.6; }
  .cap-line b { color: var(--text-primary); font-weight: 600; }
  .cap-line .sep { margin: 0 8px; color: var(--text-dim); }
  .cap-line .w.idle { color: var(--text-secondary); }
  .cap-line .w.ok { color: var(--status-success); }
  .cap-line .w.bad { color: var(--status-danger); }
  .cap-line .w.warn { color: var(--status-warn); }
  /* The just-added notice, above the grid, with the two doors in it. */
  .cap-added { margin-bottom: 14px; flex-wrap: wrap; }
  .cap-added .linklike { color: inherit; }
  .cap-added .icobtn { margin-left: auto; color: inherit; }
  /* A placement page's note above its cards. */
  .cap-pagenote { margin: -8px 0 16px; }
  .cap-pagenote.warn { color: var(--status-warn); }
  /* The two sides are a tier: two columns, larger name. */
  .cap-tiergrid { grid-template-columns: repeat(2, 1fr); }
  .cap-target.tier .chair-name { font-size: var(--fs-xl, 17px); }
  .cap-target.tier .cap-hold { font-size: var(--fs-xs); padding: 4px 10px 4px 4px; }
  @media (max-width: 820px) { .cap-tiergrid { grid-template-columns: 1fr; } }
  /* What a target holds, as chips on its card. */
  .cap-holds { display: flex; flex-wrap: wrap; gap: 5px; min-height: 24px; align-items: center; margin-top: 2px; }
  .cap-hold { display: inline-flex; align-items: center; gap: 5px; padding: 2px 8px 2px 3px; border-radius: var(--r-full, 999px); border: 1px solid var(--border-default); background: var(--surface-panel); font-size: var(--fs-2xs); color: var(--text-secondary); }
  .cap-hold.need { border-style: dashed; border-color: var(--status-warn); color: var(--status-warn); padding-left: 8px; }
  .cap-hold-none { font-size: var(--fs-2xs); color: var(--text-dim); padding: 3px 0; }
  /* The desk tile is ตั้งค่า › การเรียนรู้'s own (.mem-scope-ic, tone classes
     in style.css); only the size is this room's. */
  .cap-desk-ic { width: 30px; height: 30px; border-radius: 8px; }
  .cap-desk-ic.sm { width: 22px; height: 22px; border-radius: 6px; }
  .cap-desk-ic.xs { width: 18px; height: 18px; border-radius: 5px; }

  /* ---- the picker, on the sheet ---- */
  .cap-nobody { font-size: var(--fs-2xs); color: var(--status-warn); background: none; border: 0; padding: 0; font-family: inherit; cursor: pointer; text-decoration: underline; text-underline-offset: 2px; }
  .cap-pickbox { display: flex; flex-direction: column; gap: 8px; padding: 12px; border-radius: var(--r-lg); background: var(--surface-raised); border: 1px solid var(--border-subtle); }
  .cap-notice { display: flex; align-items: center; gap: 8px; padding: 8px 10px; border-radius: var(--r-md); font-size: var(--fs-xs); color: var(--status-success); background: var(--status-success-bg); border: 1px solid var(--status-success-border); }
  .cap-pickgroup { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
  .cap-pickgroup .lab { font-size: var(--fs-2xs); letter-spacing: .08em; text-transform: uppercase; color: var(--text-muted); font-weight: 600; }
  .cap-pickgroup .cnt { font-family: var(--mono); font-size: var(--fs-2xs); color: var(--text-dim); font-variant-numeric: tabular-nums; }
  .cap-pickgroup .rule { flex: 1; height: 1px; background: var(--border-subtle); }
  .cap-pickgroup .linklike { font-size: var(--fs-2xs); background: none; border: 0; padding: 0; font-family: inherit; }
  .cap-means { margin: -4px 0 2px; color: var(--text-dim); font-size: var(--fs-2xs); line-height: 1.5; }
  .cap-picks { display: flex; flex-wrap: wrap; gap: 6px; }
  /* A chip IS the control: label and switch are one object, so there is no
     gap to cross, and what is on shows in colour AND in the tick (DESIGN.md §4). */
  .cap-pick {
    display: inline-flex; align-items: center; gap: 7px; padding: 4px 10px 4px 5px;
    border: 1px solid var(--border-subtle); border-radius: var(--r-full, 999px);
    background: var(--surface-panel); color: var(--text-muted);
    font: inherit; font-size: var(--fs-sm); cursor: pointer;
    transition: background var(--dur-tint), color var(--dur-tint), border-color var(--dur-tint);
  }
  .cap-pick:hover:not(:disabled) { border-color: var(--border-strong); color: var(--text-primary); }
  .cap-pick:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 1px; }
  .cap-pick:disabled { opacity: .55; cursor: not-allowed; }
  .cap-pick[aria-checked="true"] {
    color: var(--accent-bright);
    border-color: color-mix(in srgb, var(--accent) 45%, transparent);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
  }
  .cap-pick.need { border-color: var(--status-warn); border-style: dashed; color: var(--status-warn); }
  .cap-pick.dead { opacity: .5; }
  .cap-sheet-pick { max-width: 560px; }
  /* Picker rows. The same states as the chip (colour AND tick), at a size
     that can hold a name, a line of what it is, and the cost. */
  .cap-pickrows { display: flex; flex-direction: column; gap: 6px; }
  .cap-pickrow {
    display: grid; grid-template-columns: auto 1fr auto 22px; align-items: center; gap: 12px;
    width: 100%; padding: 10px 12px; text-align: left;
    border: 1px solid var(--border-subtle); border-radius: var(--r-lg);
    background: var(--surface-panel); color: var(--text-secondary);
    font: inherit; cursor: pointer;
    transition: background var(--dur-tint), color var(--dur-tint), border-color var(--dur-tint);
  }
  .cap-pickrow:hover:not(:disabled) { border-color: var(--border-strong); color: var(--text-primary); }
  .cap-pickrow:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 1px; }
  .cap-pickrow:disabled { cursor: not-allowed; }
  .cap-pickrow.dead { opacity: .5; }
  .cap-pickrow[aria-checked="true"] {
    color: var(--text-primary);
    border-color: color-mix(in srgb, var(--accent) 45%, transparent);
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }
  .cap-pickrow[aria-checked="true"] .cap-pick-tick { color: var(--accent-bright); }
  .cap-pickrow.need { border-color: var(--status-warn); border-style: dashed; }
  .cap-pickrow .txt { display: flex; flex-direction: column; min-width: 0; gap: 2px; }
  .cap-pickrow .nm { font-size: var(--fs-md); font-weight: 600; color: inherit; display: flex; align-items: center; gap: 8px; }
  .cap-pickrow .needtag { font-size: var(--fs-2xs); font-weight: 500; color: var(--status-warn); border: 1px dashed var(--status-warn); border-radius: 999px; padding: 0 7px; }
  .cap-pickrow .d { font-size: var(--fs-2xs); color: var(--text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .cap-pickrow .cost { display: flex; flex-direction: column; align-items: flex-end; font-variant-numeric: tabular-nums; line-height: 1.2; }
  .cap-pickrow .cost b { font-size: var(--fs-md); font-weight: 600; color: var(--text-primary); font-family: var(--mono); }
  .cap-pickrow .cost small { font-size: var(--fs-2xs); color: var(--text-dim); }
  .cap-pick-sum b { color: var(--text-secondary); }
  .cap-hold .tk { font-family: var(--mono); font-size: var(--fs-2xs); color: var(--text-dim); margin-left: 2px; }
  .cap-hold-total { font-family: var(--mono); font-size: var(--fs-2xs); color: var(--text-muted); font-variant-numeric: tabular-nums; margin-top: 2px; }
  .cap-fold { display: inline-flex; align-items: center; gap: 6px; align-self: flex-start; background: none; border: 0; padding: 2px 0; margin-top: 4px; color: var(--text-muted); font: inherit; cursor: pointer; }
  .cap-fold:hover .eyebrow { color: var(--text-secondary); }
  .cap-fold:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 2px; border-radius: var(--r-xs); }
  .cap-fold-set { font-size: var(--fs-2xs); color: var(--text-dim); margin-left: 4px; }
  .cap-more { display: flex; justify-content: center; padding: 4px 0 8px; margin-top: -4px; }
  .cap-tools-empty { display: flex; flex-direction: column; align-items: flex-start; gap: 10px; padding: 14px; border: 1px dashed var(--border-default); border-radius: var(--r-lg); color: var(--text-muted); font-size: var(--fs-xs); line-height: 1.5; }
  .cap-tools-empty .spin svg { animation: cap-spin .8s linear infinite; }
  @keyframes cap-spin { to { transform: rotate(360deg); } }
  @media (prefers-reduced-motion: reduce) { .cap-tools-empty .spin svg { animation: none; } }
  .cap-pick-desc { margin-top: 0; }
  .cap-pick-ic { flex: none; display: grid; place-items: center; width: 20px; height: 20px; border-radius: var(--r-xs); background: var(--surface-raised); color: var(--text-dim); }
  .cap-pick[aria-checked="true"] .cap-pick-ic { color: var(--accent-bright); }
  .cap-pick.tier { font-size: var(--fs-md); padding: 6px 14px 6px 7px; border-color: var(--border-default); }
  .cap-pick.tier .cap-pick-ic { width: 26px; height: 26px; }
  .cap-pick-tick { flex: none; width: 12px; display: grid; place-items: center; }

  /* ---- the sheet ---- */
  .cap-sheet-overlay { position: fixed; inset: 0; z-index: 60; display: flex; justify-content: flex-end; }
  .cap-sheet-backdrop { position: absolute; inset: 0; width: 100%; height: 100%; background: var(--overlay-scrim); border: 0; cursor: default; }
  .cap-sheet {
    position: relative; width: min(420px, 100vw); height: 100%; overflow-y: auto;
    background: var(--surface-panel); border-left: 1px solid var(--border-default);
    display: flex; flex-direction: column; gap: 14px; padding: 18px 20px 24px;
    box-shadow: -20px 0 48px -16px rgba(0, 0, 0, .6);
    animation: cap-sheet-in var(--dur-arrive) var(--ease-arrive) both;
  }
  @keyframes cap-sheet-in { from { transform: translateX(24px); opacity: 0; } to { transform: none; opacity: 1; } }
  @media (prefers-reduced-motion: reduce) { .cap-sheet { animation: none; } }
  .cap-sheet-head { display: flex; align-items: center; gap: 10px; }
  .cap-sheet-status { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; font-size: var(--fs-2xs); }
  .cap-sheet-status .grow { flex: 1; }
  .cap-tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--border-subtle); }
  .cap-tabs button { font: inherit; font-size: var(--fs-sm); color: var(--text-muted); background: none; border: 0; padding: 8px 10px; border-bottom: 2px solid transparent; cursor: pointer; margin-bottom: -1px; }
  .cap-tabs button.on { color: var(--text-primary); font-weight: 500; border-bottom-color: var(--accent); }
  .cap-tabs button:disabled { opacity: .45; cursor: not-allowed; }
  .cap-tabs button .c { font-family: var(--mono); font-size: var(--fs-2xs); color: var(--text-dim); margin-left: 5px; }
  .cap-tabs button:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 1px; }
  .cap-two { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
  .cap-toolhead { display: flex; align-items: center; gap: 10px; font-size: var(--fs-2xs); color: var(--text-muted); }
  .cap-toolhead .grow { flex: 1; }
  .cap-toolhead .linklike { font-size: var(--fs-2xs); background: none; border: 0; padding: 0; font-family: inherit; }
  .cap-toollist { display: flex; flex-direction: column; }
  .cap-toolrow { display: flex; align-items: center; gap: 9px; padding: 6px 8px; border-radius: var(--r-md); font-size: var(--fs-xs); color: var(--text-secondary); cursor: pointer; }
  .cap-toolrow { transition: background var(--dur-tint); }
  .cap-toolrow:hover { background: var(--surface-raised); }
  .cap-toolrow input { margin: 0; accent-color: var(--interactive); }
  .cap-toolrow .nm { font-family: var(--mono); font-size: var(--fs-xs); color: var(--text-primary); flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; }
  .cap-toolrow:not(.on) .nm { color: var(--text-muted); }
  .cap-toolrow .tk { font-family: var(--mono); font-size: var(--fs-2xs); color: var(--text-dim); }
  .cap-sheet-head h3 { margin: 0; flex: 1; min-width: 0; font-size: var(--fs-lg); font-weight: 600; color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .cap-sheet-body { display: flex; flex-direction: column; gap: 10px; }
  .cap-sheet-body > input, .cap-sheet-body > textarea { width: 100%; }
  /* One entry per line, never wrapped mid-path: a long value scrolls sideways
     inside its own line, so the line count on screen is the entry count. */
  .cap-sheet-body > textarea.mcp-lines { min-height: 6.5em; line-height: 1.45; white-space: pre; overflow-x: auto; }
  /* The notes under the form read as notes, not as more form: smaller than
     the fields, with room between lines for Thai. */
  .cap-sheet-body > .d { font-size: var(--fs-xs); line-height: 1.55; }
  /* What a preset still needs: one bordered note with a title, numbered
     steps, and a foot — the shape a person can act on top to bottom. */
  .cap-keynote {
    display: flex; flex-direction: column; gap: 8px;
    padding: 10px 14px 10px 14px;
    border-left: 2px solid var(--accent); border-radius: 0 var(--r-sm) var(--r-sm) 0;
    background: var(--surface-sunken);
    font-size: var(--fs-xs); line-height: 1.55; color: var(--text-secondary);
  }
  .cap-keynote-t { font-size: var(--fs-sm); font-weight: 600; color: var(--text-primary); }
  .cap-keynote p { margin: 0; }
  .cap-keynote-steps { margin: 0; padding-left: 1.5em; display: flex; flex-direction: column; gap: 6px; }
  .cap-keynote-steps li { padding-left: 3px; }
  .cap-keynote-steps li::marker { color: var(--text-dim); font-variant-numeric: tabular-nums; }
  .cap-keynote code {
    font-family: var(--mono); font-size: 0.92em; padding: 1px 5px;
    border-radius: var(--r-xs); background: var(--surface-code); color: var(--text-code);
    overflow-wrap: anywhere;
  }
  .cap-keynote-f { font-size: var(--fs-2xs); color: var(--text-dim); }
  /* Under the form, not pinned to the bottom of a full-height sheet: pinned,
     the ลบ button sat a screen below the fields and read as absent. */
  .cap-sheet-foot { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; padding-top: 14px; border-top: 1px solid var(--border-subtle); }
</style>
