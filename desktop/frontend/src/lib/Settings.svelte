<script lang="ts">
  import { onMount, tick } from 'svelte'
  import { theme, applyTheme, THEMES, type ThemeName } from './theme.svelte'
  import { editorFont, applyEditorFontSize } from './editorFont.svelte'
  import { chatFont, applyChatFontSize } from './chatFont.svelte'
  import { uiFont, applyUiFont, UI_FONTS, type UiFontName } from './uiFont.svelte'
  import { editorTheme, setBuiltinEditorTheme, setAutoEditorTheme, importThemeFile } from './editorTheme.svelte'
  import { treeFont, applyTreeFontSize } from './treeFont.svelte'
  import { systemZoom, applySystemZoom, SYSTEM_BASE_PX } from './systemFont.svelte'
  import { typeScale, applyTypeScale, TYPE_SCALES, type TypeScaleName } from './typeScale.svelte'
  import { i18n, t, setLocale, localeNames, type Locale, type TKey } from './i18n.svelte'
  import { AGENT_TEMPLATES, templateBody } from './agentTemplates'
  import { GALLERY_GROUPS, GALLERY_ROLES, GALLERY_SKIP_KEY, galleryBody, galleryMatches } from './agentGallery'
  import { audioDevices, refreshAudioDevices, setMicId, setSpeakerId, applySpeaker } from './audioDevices.svelte'
  import ConfirmDialog from './ConfirmDialog.svelte'
  import RemoteEngine from './RemoteEngine.svelte'
  import ProviderMark from './ProviderMark.svelte'
  import ProviderAccount from './ProviderAccount.svelte'
  import AgentMascot from './mascot/AgentMascot.svelte'
  import Mascot from './mascot/Mascot.svelte'
  import { HEADS, headOptions, type HeadId } from './mascot/avatarPrefs.svelte'
  import RankPip from './RankPip.svelte'
  import RankedFace from './RankedFace.svelte'
  import type { PoseId } from './mascot/poses'
  import ScopeMark from './ScopeMark.svelte'
  import { guide } from './guide/guideState.svelte'
  import AvatarSettings from './mascot/AvatarSettings.svelte'
  import TeamSettings from './TeamSettings.svelte'
  import { avatarText } from './mascot/avatarText'
  // The mascot's own catalogues, so the pickers below offer exactly what the
  // drawing can draw. Anything hand-listed here instead would be a second
  // catalogue to keep in step, which is the bug this page once had.
  import { lookOf, AGENT_BADGES } from './mascot/agentLook'
  import { SHELL, ACCENT } from './mascot/palette'
  import { FACE, TOP } from './mascot/parts'
  import { personas } from './mascot/avatarPrefs.svelte'
  import Icon from './Icon.svelte'
  import McpMark from './McpMark.svelte'
  import { coverHue } from './coverHue'
  import { armFirstRunReplay } from './firstRun'
  import { scopeLabel, scopeMeta, USER_SCOPE, MAIN_SCOPE } from './memoryScope'
  import { openTour } from './tourState.svelte'
  import { setShell } from './shell.svelte'
  import { attention, loadAttention, toggleAttention } from './stores/attention.svelte'
  import type { IconName } from './icons'
  import { NAV, deskLabelKey } from './desks'
  // The shelf and everything that turns one of its entries into a saved server.
  // It used to be written out in this file; ห้องความสามารถ reads the same list,
  // and a preset table with two copies goes stale on one of them (mcpShelf.ts).
  import { presetConfig, presetFor } from './mcpShelf'
  import { STUDIO_SOURCES } from './studioSources'
  import StudioBrowser from './StudioBrowser.svelte'
  import StudioSourcesSheet from './StudioSourcesSheet.svelte'
  import {
    SupportedProviders, HasAPIKey, APIKeyHint, RequiresAPIKey, AcceptsAPIKey, ProviderAccountFor, TerminalShells,
    ListModelsForProvider, SupportedThinkLevelsFor, ProviderBaseURL, ProviderBaseURLIsCustom, ProviderAPIKeyURL, ProviderReady, PriceModels,
    ProviderWireFormats, TestProviderConnection,
    EnabledProviders, SetProviderEnabled,
    CustomProviders, AddCustomProvider, RemoveCustomProvider,
    ListMCPServers, SaveMCPServer,
    DelegateSwitches, SetDelegateOff, SetAgentOff,
    PlacementTargets, SetMCPServerTargets,
    ListSpeechModels, SetSpeechModel, SpeechStatus, RevealSpeechModel, SpeechModelDirs, OpenSpeechModelDir,
    ListSpeechEngines, SetSpeechEngine, ListTTSEngines, SetTTSEngine, ListTTSVoices, SetTTSVoice, TTSStatus, SpeakText,
    ListImageEngines, SetImageEngine, SetImageModelName, ImageStatus,
    SetSpeechModelName, SetTTSModelName,
    InstallVoiceEngine,
    UsageStats,
    ModelPriceSource,
    ListSubagentProfiles, ReadSubagentProfile, SaveSubagentProfile, SaveAgentProfile, PickAgentBrief, FetchAgentBrief,
    DeleteSubagentProfile, SetSubagentModel, OpenAgentsFolder, OpenAgentSkillsFolder, ListChairs,
    ListExternalSkills, CopySkillToAgent,
    AgentSkills, AgentNeeds, OpenAgentHome,
    ChairStarters, SaveChairStarters, ChairStartersFile, DeskStarters, SaveDeskStarters, HeadName, SetHeadName,
    SignInMethods, SignInStatus, StartSignIn, CancelSignIn, ImportableSignIns,
    AppVersion, AppCredit, RecentDebugLog,
    LearningEnabled, SetLearningEnabled, ListPendingChanges, ListDecidedChanges, ListModes, MemoryScopeInfo,
    ReadDeskFile, SaveDeskFile, ResetDeskFile,
    SessionReviewAuto, SetSessionReviewAuto, RunSessionReview,
    PreparedReplyOn, SetPreparedReplyOn,
    ApprovePendingChange, ApprovePendingChangeTo, RejectPendingChange, LearnedEntries, LearnedScopeInfos, ConsolidateMemory, ApplyMemoryLines, SaveLearnedEntry, AddLearnedEntry, MoveLearnedEntry, OpenMemoryFolder,
    ForgetMemoryScope, AdoptMemoryScope, RecentProjects,
    ListSystemIssues, MarkIssueReported, ListDecidedIssues,
    AccountStatus, StartAccountSignIn, CompleteAccountSignIn, CancelAccountSignIn,
    AccountSignOut, AccountRefresh,
    StudioLibraries, StudioScanning, AddStudioLibrary, RescanStudioLibrary, RemoveStudioLibrary, RevealStudioLibrary, CancelStudioScan,
  } from '../../wailsjs/go/main/App'
  import { BrowserOpenURL, EventsOn } from '../../wailsjs/runtime/runtime'
  // Deliberately alongside the issue button rather than instead of it: an issue
  // carries the version and the log, the group carries the half-formed question
  // that is not a bug report yet.
  import { COMMUNITY_URL, PAGE_URL, YOUTUBE_URL } from './links'
  import promptPayQR from '../assets/images/promptpay-qr.png'
  import { config, engine, main, subagent, type mode } from '../../wailsjs/go/models'
  import { cockpit, openCapabilityAt, startChatWith, newChairSession, setActiveView, switchProvider, switchModel, submitAPIKey, switchApprovalMode, switchWireFormat, setProviderBaseURL, retryActiveProvider, completeSignIn, signOutProvider, importSignIn, SETTINGS_SECTION_KEY } from './stores/cockpit.svelte'
  import {
    identity, loadIdentityFiles, openIdentityFile, saveIdentityFile,
    createIdentityFile, deleteIdentityFile, closeIdentityFile,
  } from './identity.svelte'
  import { profile, loadProfileName, saveProfileName } from './stores/profile.svelte'
  import { updater, updatePct, startDownload, restartToUpdate, checkNow } from './selfUpdate.svelte'

  let { onClose }: { onClose: () => void } = $props()

  // ---------- Destructive actions ----------
  // One gate for everything that cannot be undone. This page used to have two
  // different answers to the same question: Skills, Prompts and Sub-agents
  // armed on the first click and deleted on the second, while MCP servers,
  // providers and identity files deleted on the first click with no warning at
  // all. Learning "it asks first" from one page and then losing a configured
  // MCP server on the next is the worst of both.
  type PendingConfirm = {
    title: string
    message: string
    /** The exact name/path being destroyed — shown verbatim for checking. */
    detail?: string
    confirmLabel: string
    run: () => void
  }
  let pendingConfirm = $state<PendingConfirm | null>(null)

  function askConfirm(req: PendingConfirm) {
    pendingConfirm = req
  }

  function runPendingConfirm() {
    const req = pendingConfirm
    pendingConfirm = null
    req?.run()
  }

  // Leaving a full-page editor with unsaved work is the same class of loss as a
  // delete — the work is gone and nothing says so — so it goes through the same
  // gate. Dirty is measured against a snapshot taken when the editor opened,
  // not against a field-by-field comparison: the sub-agent editor has seven
  // drafts and the diff only ever gets asked one question.
  function guardUnsaved(dirty: boolean, leave: () => void) {
    if (!dirty) { leave(); return }
    askConfirm({
      title: t('settings.unsavedTitle'),
      message: t('settings.unsavedMessage'),
      confirmLabel: t('settings.unsavedAction'),
      run: leave,
    })
  }

  // ---------- AI identity ----------
  // The files that ride along in a head's every system prompt (one folder per
  // head, config.IdentityDirFor). Drawn on ตัวหลัก › ตัวตน since 14 ก.ย. 2026
  // (owner: "คำสั่งประจำตัวพวกนี้ผูกกับเอเจนหลัก … แยกกันทั้งสองตัว เอาไว้ที่ส่วน
  // ตัวตน") — it was คำสั่งประจำตัว in the menu, one shared set, and before
  // that the sidebar. The store (identity.svelte.ts) holds one head at a time.
  const identityDirty = $derived(identity.draft !== identity.saved)
  const recommendedIdentityTemplates: {
    name: string
    icon: IconName
    descKey: TKey
    // null: the + button makes the file blank. context.md is the person's own
    // words about themselves (owner, 14 ก.ย.: "ควรจะโล่งเป็นค่าเริ่มต้น") — a
    // scaffold of bullets there is someone else's idea of what to say.
    tplKey: TKey | null
  }[] = [
    { name: 'identity.md', icon: 'sparkles', descKey: 'settings.identityDescIdentity', tplKey: 'identity.tplIdentity' },
    { name: 'thinking.md', icon: 'brain', descKey: 'settings.identityDescThinking', tplKey: 'identity.tplThinking' },
    // context.md was on เกี่ยวกับคุณ for a morning (14 ก.ย.); the owner put
    // it back with the other three: "ไม่ควรไปอยู่เกี่ยวกับคุณ มันควรผูกกับเอเจน".
    { name: 'context.md', icon: 'fileText', descKey: 'settings.identityDescContext', tplKey: null },
    // skills.md left the list 14 ก.ย. 2026 (owner: "ไม่มีประโยชน์ชัดเลยหน้านี้"):
    // a fourth always-on file nobody could say the purpose of. A hand-made
    // one on disk is still listed below, like any other.
  ]
  // The template "+" writes is the head's own: the assistant's identity is a
  // friend and a personal helper, the coder's an engineer beside you, and each
  // thinks the way its desk works (owner, 14 ก.ย.: "identity.md หน้าผู้ช่วย คือ
  // เป็นเพื่อน ผู้ช่วยส่วนตัว … กระชับ ไม่ต้องใส่เหตุผล"). Short on purpose: every
  // line here is folded into every request.
  const tplFor = (item: { name: string; tplKey: TKey | null }): string => {
    if (item.tplKey === null) return ''
    if (identity.head === 'coding' && item.name === 'identity.md') return t('identity.tplIdentityCoding')
    if (identity.head === 'coding' && item.name === 'thinking.md') return t('identity.tplThinkingCoding')
    return t(item.tplKey)
  }
  // Files a person made by hand (the old "เพิ่มไฟล์คำสั่งใหม่" box, gone on the
  // owner's word the same day): still listed while they exist, so a file on
  // disk is never invisible, but nothing here makes a new one.
  const customIdentityFiles = $derived(
    (identity.files || []).filter((f) => !recommendedIdentityTemplates.some((r) => r.name === f.name)),
  )

  const removeIdentityFile = (name: string) => askConfirm({
    title: t('settings.confirmIdentityTitle'),
    message: t('settings.confirmIdentityMessage'),
    detail: name,
    confirmLabel: t('settings.confirmDeleteAction'),
    run: () => deleteIdentityFile(name),
  })

  const approvalOptions = [
    { value: 'ask', label: t('chat.approvalAsk') },
    { value: 'unsafe-only', label: t('chat.approvalUnsafeOnly') },
    { value: 'full-access', label: t('chat.approvalFullAccess') },
  ]

  // ---------- General: default shell ----------
  let shells = $state<{ name: string; path: string }[]>([])
  let defaultShell = $state(localStorage.getItem('defaultShell') ?? '')
  // Whether a turn that ends by asking something writes the user's reply for
  // them (desktop/prepared_reply.go). Ships on, so the honest initial value is
  // on: a switch drawn off for the moment before Go answers reads as a feature
  // that is disabled rather than one that is loading.
  let preparedOn = $state(true)

  function saveDefaultShell() {
    localStorage.setItem('defaultShell', defaultShell)
  }

  // ---------- General: replay the first run ----------
  // The first screen anyone sees is the one nobody who works on Aetox can see:
  // this machine has been onboarded, has a key, and the wizard's own shortcuts
  // make sure of it. This puts the window back into that state so it can be
  // looked at. Through the same gate as a delete, because a window that
  // reloads and forgets your theme without warning is the same surprise even
  // when nothing is destroyed.
  const replayFirstRun = () => askConfirm({
    title: t('settings.firstRunConfirmTitle'),
    message: t('settings.firstRunConfirmMessage'),
    confirmLabel: t('settings.firstRunAction'),
    run: () => {
      armFirstRunReplay()
      window.location.reload()
    },
  })

  // ---------- Appearance: code theme import ----------
  let themeImportError = $state('')

  async function onThemeFileChosen(e: Event) {
    const file = (e.currentTarget as HTMLInputElement).files?.[0]
    if (!file) return
    themeImportError = ''
    try {
      await importThemeFile(file)
    } catch (err) {
      themeImportError = t('settings.importThemeError', { err: String(err) })
    }
    ;(e.currentTarget as HTMLInputElement).value = ''
  }

  // ---------- Model settings ----------
  // acceptsKey is not the negation of requiresKey: Codex requires credentials
  // and takes no key, because the only key a user could paste belongs to a
  // different host and a different bill.
  // ready is null until the engine answers. A local runtime cannot be known to
  // be up without asking it, and the dot used to be painted green on the
  // strength of "needs no key" — which is why LM Studio looked connected on a
  // card that said no models were found. Unknown must look like unknown.
  type ProviderRow = {
    name: string; requiresKey: boolean; acceptsKey: boolean; hasKey: boolean
    // The masked tail of the key that would actually be sent, or "" when
    // there is none. hasKey answers whether one exists; this answers which
    // one, which is the question a blank field could not.
    keyHint: string
    ready: boolean | null
  }

  let providers = $state<ProviderRow[]>([])
  let enabledNames = $state<string[]>([])
  let showAddProvider = $state(false)
  // The endpoints the user added themselves (AddCustomProvider). A row in
  // here is theirs to delete outright — key and all — where a catalog row's
  // X only hides it. The draft is the form for the next one; it opens in
  // the detail pane because the 190px sidebar has no room for three fields.
  let customNames = $state<Set<string>>(new Set())
  let customDraftOpen = $state(false)
  let customDraft = $state({ name: '', baseURL: '', apiKey: '' })
  let customDraftError = $state('')
  let selected = $state('')
  let baseURL = $state('')
  let wireFormats = $state<string[]>([])
  let models = $state<string[]>([])
  let loadingModels = $state(false)
  let keyDraft = $state('')
  let showKey = $state(false)
  // Where this provider's key is actually created. Empty for the rows that have
  // no such page (a local runtime, a sign-in provider), which is also the
  // signal not to draw the link.
  let keyPageURL = $state('')
  let customModel = $state('')
  // The model list is unusable without these on any hosted aggregator: 411
  // rows, alphabetical, so reaching "deepseek" means scrolling past every
  // aion-labs build first.
  let modelFilter = $state('')
  let freeOnly = $state(false)
  type ModelListing = {
    model: string; input: number; output: number
    priced: boolean; free: boolean; context: number
  }
  let priced = $state<Record<string, ModelListing>>({})
  // Where the price column came from, and when. A bare "$0.14 / $0.28" beside
  // a model name is a claim Aetox is not in a position to make: the figures are
  // models.dev's published list, copied verbatim, and on 2026-08-28 its
  // DeepSeek rows disagreed with DeepSeek's own page by two to four times. The
  // stats page has carried this qualification since it shipped; the picker,
  // which is where the number is actually read, carried none.
  let priceSource = $state<{ name: string; fetched: string }>({ name: '', fetched: '' })
  const priceSourceLine = $derived(
    priceSource.name && Object.values(priced).some((p) => p.priced)
      ? t('settings.priceSource', {
          source: priceSource.name,
          date: new Date(priceSource.fetched).toLocaleDateString(undefined, {
            year: 'numeric', month: 'short', day: 'numeric',
          }),
        })
      : '',
  )

  // Cheapest first once prices are known, because a price nobody can sort by is
  // a number to look at rather than one to decide with. Ties and unpriced rows
  // keep the provider's own order, which is the only order they have.
  const visibleModels = $derived.by(() => {
    const needle = modelFilter.trim().toLowerCase()
    const rows = models.filter((m) => {
      if (needle && !m.toLowerCase().includes(needle)) return false
      if (freeOnly && !priced[m]?.free) return false
      return true
    })
    return rows.sort((a, b) => {
      const pa = priced[a], pb = priced[b]
      if (pa?.priced && pb?.priced && pa.input !== pb.input) return pa.input - pb.input
      if (pa?.priced !== pb?.priced) return pa?.priced ? -1 : 1
      return 0
    })
  })
  const freeCount = $derived(models.filter((m) => priced[m]?.free).length)

  // ---------- Sign-in (use the plan you already pay for) ----------
  type SignInMethod = { provider: string; label: string; kind: string; risk: string; note: string }
  type SignInPrompt = { provider: string; kind: string; url: string; user_code?: string; verification_uri?: string }

  let signInMethods = $state<SignInMethod[]>([])
  let signedIn = $state<Record<string, { signed_in: boolean; label?: string; account?: string }>>({})
  // The authorization currently on screen. Only one at a time: the flow blocks
  // on the user, and two half-finished sign-ins is a state nobody can reason
  // about.
  let signInPrompt = $state<SignInPrompt | null>(null)
  let signInCode = $state('')
  let signInError = $state('')
  // Providers whose official CLI is already signed in on this machine, so the
  // user can adopt that session instead of authorizing the same account twice.
  let importable = $state<string[]>([])

  // ---------- Connections ----------
  // The register of accounts and self-run services left this file 14 ก.ย.
  // 2026 for ห้องความสามารถ (Capability.svelte): a connection is a reach out
  // of the app, the room's subject. Sign-in above stays — it buys thinking,
  // not reach.

  const signInProviderNames = $derived(new Set(signInMethods.map((m) => m.provider)))
  const signInMethod = $derived(signInMethods.find((m) => m.provider === selected) ?? null)
  const signInStatus = $derived(signedIn[selected] ?? null)
  let busy = $state('')
  let errorMsg = $state('')

  const selectedRow = $derived(providers.find((p) => p.name === selected))
  // Fetched per provider rather than for all of them at once: opening a card is
  // what asks, so a user who never opens Settings never spends a round trip on
  // a balance nobody is looking at.
  let accounts = $state<Record<string, any>>({})
  const account = $derived(accounts[selected] ?? null)

  async function loadAccount(name: string) {
    if (!name) return
    try {
      accounts = { ...accounts, [name]: await ProviderAccountFor(name) }
    } catch {
      // A provider that cannot answer leaves its card without the line, which
      // is the same as never having asked. Nothing else on the page depends
      // on it, so this must not surface as a page-level failure.
    }
  }

  async function refreshAccount() {
    busy = 'account'
    await loadAccount(selected)
    busy = ''
  }
  const isActiveProvider = $derived(cockpit.model.provider === selected)
  const enabledRows = $derived(providers.filter((p) => enabledNames.includes(p.name)))
  const addableRows = $derived(providers.filter((p) => !enabledNames.includes(p.name)))
  // Split, because the two kinds ask for completely different things: one wants
  // a browser and the plan you already pay for, the other wants a key you have
  // to go find. Mixing them in one alphabetical list hid the sign-ins.
  const addableSignIn = $derived(addableRows.filter((p) => signInProviderNames.has(p.name)))
  const addableKeyed = $derived(addableRows.filter((p) => !signInProviderNames.has(p.name)))
  // Only meaningful while this provider is the active one — otherwise nothing
  // has been bootstrapped for it yet, so show what would be the default.
  const currentWireFormat = $derived(isActiveProvider ? cockpit.model.wireFormat : (wireFormats[0] ?? ''))

  // Whether the first load finished, and why it didn't. Without this the whole
  // page was one unguarded await chain: a single throw from TerminalShells()
  // left providers, sign-in, MCP and skills all unloaded, and the user got a
  // blank Settings page with nothing saying anything had gone wrong.
  let booting = $state(true)
  let bootError = $state('')

  async function bootSettings() {
    booting = true
    bootError = ''
    try {
      // Three independent groups, run together rather than in a queue. They
      // were sequential, which made the tool list — needed by the sub-agent
      // editor — the last thing to arrive after every provider round-trip, so
      // opening a sub-agent quickly could find it still empty. Only the
      // provider chain has an internal order.
      await Promise.all([
        (async () => {
          shells = await TerminalShells()
          if (!shells.some((s) => s.path === defaultShell)) defaultShell = shells[0]?.path ?? ''
        })(),
        (async () => { preparedOn = await PreparedReplyOn() })(),
        (async () => { learningOn = await LearningEnabled() })(),
        loadAttention(),
        loadMCP(),
        loadSpeech(),
        (async () => {
          await refreshProviders()
          await refreshEnabledProviders()
          await refreshSignIn()
          await selectProvider(cockpit.model.provider || enabledRows[0]?.name || providers[0]?.name || '')
        })(),
      ])
    } catch (err) {
      bootError = String(err)
    } finally {
      booting = false
    }
  }

  onMount(bootSettings)

  // The team page's two doors land here: configure-on-a-card and the create
  // form both open this page with the editor already holding the right
  // profile — and the right *kind*, which the intent carries because it came
  // from the roster, not from re-reading any file. Consumed once and cleared:
  // an intent that survived into the next plain visit would reopen an editor
  // nobody asked for.
  //
  // The section itself is already `active` from the first frame (see the
  // $state below) — only the editor has to wait for the disk. While it does,
  // the 'team' pane draws a skeleton, not its "the list moved" notice: that
  // notice with its button sat on screen for a second and a half on every
  // gear press and read as the page having landed somewhere wrong (owner,
  // 13 ก.ย. 2026: "กดแล้วโหลดแปลกๆ"). The wait is one round trip for the row,
  // not loadAgents()' three in a row — the $effect on `active` is already
  // running that one, and the editor needs only the row plus its own file.
  let intentPending = $state(cockpit.settingsIntent !== null)
  onMount(async () => {
    await takeIntent()
  })
  // An intent set while the page is already open — the guide walking from
  // one section to another (lib/guide/pages.ts) — is taken the same way. The
  // page used to read it once, at mount, and a section written to the store
  // after that was a page that did not move.
  $effect(() => {
    if (cockpit.settingsIntent && !intentPending) {
      intentPending = true
      void takeIntent()
    }
  })
  async function takeIntent() {
    const intent = cockpit.settingsIntent
    if (!intent) {
      intentPending = false
      return
    }
    cockpit.settingsIntent = null
    try {
      openSection(intent.section)
      if (intent.head) openHead(intent.head)
      if (intent.section !== 'team') return
      if (intent.createAgent) {
        newAgent('agent')
        return
      }
      if (intent.agent) {
        subagents = await ListSubagentProfiles()
        const row = subagents.find((a) => a.name === intent.agent)
        if (row) await openAgent(row, 'agent')
        if (intent.tab) agentTab = intent.tab as AgentTab
        // An intent naming an agent comes from the roster page (a card's
        // lock), so the editor's back button walks back there.
        agentBack = 'office'
      }
    } finally {
      intentPending = false
    }
  }

  // ---------- About ----------
  // Kept out of bootSettings on purpose. The version is a constant the Go side
  // always has, and folding it into that Promise.all would let a stumble here
  // take the whole Settings page down with it for nothing.
  let appVersion = $state('')
  let appCredit = $state('')
  let hintCopied = $state(false)
  // Neither the answer NOR the act lives here. The same update can be started
  // from the notice the automatic check raises and from the version row in the
  // profile menu, and three private copies of "there is a v1.5.9 / 42% /
  // restarting / here is what went wrong" are three answers to one question
  // waiting to disagree. selfUpdate.svelte owns all of it; this page is a view.
  //
  // The check moved there on 2026-08-26, when the profile menu became the third
  // door: a menu that had asked GitHub itself would have been free to show a
  // release this page had never heard of, which is the same debt one layer up.
  const updateStatus = $derived(updater.status)
  const updateChecking = $derived(updater.checking)
  const updateError = $derived(updater.checkError)
  onMount(() => {
    void (async () => {
      try {
        appVersion = await AppVersion()
        appCredit = await AppCredit()
      } catch {
        /* the About page shows a dash rather than an error */
      }
    })()
  })

  const CHANNEL_LABELS: Record<string, TKey> = {
    scoop: 'settings.aboutChannelScoop',
    installer: 'settings.aboutChannelInstaller',
    portable: 'settings.aboutChannelPortable',
    store: 'settings.aboutChannelStore',
    unknown: 'settings.aboutChannelUnknown',
  }

  async function copyUpgradeHint(command: string) {
    try {
      await navigator.clipboard.writeText(command)
      hintCopied = true
      setTimeout(() => (hintCopied = false), 1500)
    } catch {
      /* clipboard blocked — the command is on screen to be typed */
    }
  }

  async function refreshProviders() {
    const names = await SupportedProviders()
    providers = await Promise.all(names.map(async (name) => ({
      name,
      requiresKey: await RequiresAPIKey(name),
      acceptsKey: await AcceptsAPIKey(name),
      hasKey: await HasAPIKey(name),
      keyHint: await APIKeyHint(name),
      ready: null,
    })))
    // Readiness is asked for separately and not awaited with the rest: proving
    // a local runtime is up means opening a connection to it, and a dead port
    // costs a timeout. The list must draw immediately and fill in, rather than
    // hold the whole page for the one provider that is switched off.
    for (const row of providers) {
      ProviderReady(row.name)
        .then((ready) => { row.ready = ready })
        .catch(() => { row.ready = false })
    }
  }

  async function refreshEnabledProviders() {
    enabledNames = await EnabledProviders()
    customNames = new Set(((await CustomProviders()) ?? []).map((r) => r.id))
  }

  async function refreshSignIn() {
    signInMethods = (await SignInMethods()) ?? []
    const entries = await Promise.all(
      signInMethods.map(async (m) => [m.provider, await SignInStatus(m.provider)] as const),
    )
    signedIn = Object.fromEntries(entries)
    importable = (await ImportableSignIns()) ?? []
  }

  // Two calls, not one: the first returns what to show the user (a code to
  // type, a page to visit), the second blocks until they finish. Device and
  // browser flows chain straight into the wait; only the paste flow stops here
  // for input.
  async function startSignIn() {
    const method = signInMethod
    if (!method) return
    signInError = ''
    signInCode = ''
    try {
      signInPrompt = await StartSignIn(method.provider)
    } catch (e) {
      signInError = String(e)
      return
    }
    if (signInPrompt.url) BrowserOpenURL(signInPrompt.url)
    if (method.kind !== 'paste') await finishSignIn()
  }

  async function finishSignIn() {
    const prompt = signInPrompt
    if (!prompt) return
    busy = 'signin'
    signInError = ''
    try {
      await completeSignIn(prompt.provider, signInCode.trim())
      signInPrompt = null
      signInCode = ''
      await refreshSignIn()
      await refreshProviders()
      // Re-select to pick up the model list, which was unreachable until now.
      await selectProvider(prompt.provider)
    } catch (e) {
      signInError = String(e)
    } finally {
      busy = ''
    }
  }

  async function abortSignIn() {
    const prompt = signInPrompt
    signInPrompt = null
    signInCode = ''
    signInError = ''
    if (prompt) await CancelSignIn(prompt.provider)
  }

  // The Aetox account, which is a different sign-in from the ones above: those
  // decide who pays for a request, this one is who you are to Aetox itself.
  let aetoxAccount = $state<main.AccountState | null>(null)
  let aetoxBusy = $state(false)
  let aetoxError = $state('')

  async function loadAetoxAccount() {
    aetoxAccount = await AccountStatus()
  }

  // Same two-call shape as a provider sign-in, and for the same reason: the
  // first call hands back a URL to open, the second blocks until the browser
  // comes back. Nothing is stored unless the second one succeeds.
  async function aetoxSignIn(provider: string) {
    aetoxError = ''
    aetoxBusy = true
    try {
      const url = await StartAccountSignIn(provider)
      if (url) BrowserOpenURL(url)
      aetoxAccount = await CompleteAccountSignIn()
    } catch (e) {
      aetoxError = String(e)
      await loadAetoxAccount()
    } finally {
      aetoxBusy = false
    }
  }

  async function aetoxAbort() {
    await CancelAccountSignIn()
    aetoxBusy = false
    aetoxError = ''
  }

  async function aetoxSignOut() {
    aetoxError = ''
    aetoxBusy = true
    try {
      await AccountSignOut()
    } catch (e) {
      // The local half of a sign-out always happened, so this is a warning
      // about the server, not a failure to sign out. The card says so.
      aetoxError = t('settings.accountSignOutPartial')
    } finally {
      await loadAetoxAccount()
      aetoxBusy = false
    }
  }

  async function aetoxCheck() {
    aetoxError = ''
    aetoxBusy = true
    try {
      aetoxAccount = await AccountRefresh()
    } catch (e) {
      aetoxError = String(e)
      await loadAetoxAccount()
    } finally {
      aetoxBusy = false
    }
  }

  const doImport = (name: string) => run('import:' + name, async () => {
    await importSignIn(name)
    await refreshSignIn()
    await refreshProviders()
    await selectProvider(name)
  })

  const doSignOut = (name: string) => run('signout:' + name, async () => {
    await signOutProvider(name)
    await refreshSignIn()
    await refreshProviders()
  })

  const addProvider = (name: string) => run('enable:' + name, async () => {
    enabledNames = await SetProviderEnabled(name, true)
    showAddProvider = false
    await selectProvider(name)
  })

  // customDraftFrom names the card the form was opened from — the "+" under
  // its Base URL — so the endpoint on that card is what the form starts
  // with, and its saved key (which this side only ever sees the tail of)
  // is copied by the engine when no new one is typed. '' from the sidebar.
  let customDraftFrom = $state('')
  function openCustomDraft(from = '') {
    showAddProvider = false
    customDraftError = ''
    customDraftFrom = from
    customDraft = from
      ? { name: '', baseURL: baseURLDraft.trim() || baseURL, apiKey: keyDraft.trim() }
      : { name: '', baseURL: '', apiKey: '' }
    customDraftOpen = true
  }
  // The card the form copies a key from has one to copy.
  const customDraftCopiesKey = $derived(
    customDraftFrom !== '' && customDraft.apiKey.trim() === '' && (providers.find((p) => p.name === customDraftFrom)?.hasKey ?? false))

  const submitCustomDraft = () => run('custom:add', async () => {
    customDraftError = ''
    try {
      const id = await AddCustomProvider(customDraft.name, customDraft.baseURL, customDraft.apiKey, customDraftFrom)
      customDraftOpen = false
      customDraft = { name: '', baseURL: '', apiKey: '' }
      // The new row has to exist in `providers` before selectProvider can
      // find it, so the full list is re-read rather than just the enabled set.
      await refreshProviders()
      await refreshEnabledProviders()
      await selectProvider(id)
    } catch (e) {
      // Shown in the form, next to the fields that caused it, rather than in
      // the card's general error slot the form has replaced.
      customDraftError = String(e).replace(/^Error:\s*/, '')
    }
  })

  const removeCustomProvider = (name: string) => askConfirm({
    title: t('settings.confirmCustomTitle'),
    message: cockpit.model.provider === name
      ? t('settings.confirmCustomMessage') + ' ' + t('settings.confirmProviderActive')
      : t('settings.confirmCustomMessage'),
    detail: name,
    confirmLabel: t('settings.remove'),
    run: () => run('disable:' + name, async () => {
      const wasActiveEngine = cockpit.model.provider === name
      enabledNames = await RemoveCustomProvider(name)
      await refreshProviders()
      await refreshEnabledProviders()
      if (selected === name) await selectProvider(enabledNames[0] ?? '')
      // Same rule as removeProvider below: the engine cannot keep running on
      // a row that no longer exists.
      if (wasActiveEngine) await switchProvider('aetox')
    }),
  })

  const removeProvider = (name: string) => askConfirm({
    title: t('settings.confirmProviderTitle'),
    // Removing the running provider moves the engine as a side effect, which
    // is exactly the kind of thing a confirm exists to say out loud.
    message: cockpit.model.provider === name
      ? t('settings.confirmProviderMessage') + ' ' + t('settings.confirmProviderActive')
      : t('settings.confirmProviderMessage'),
    detail: name,
    confirmLabel: t('settings.remove'),
    run: () => run('disable:' + name, async () => {
      const wasActiveEngine = cockpit.model.provider === name
      enabledNames = await SetProviderEnabled(name, false)
      if (selected === name) await selectProvider(enabledNames[0] ?? '')
      // Removing the provider Aetox is actually running on must move the engine
      // too — otherwise it keeps running unlisted while the picker shows a
      // provider that's no longer selectable. Falls back to aetox (Aetox's own
      // built-in engine, always available, needs no key) rather than an
      // arbitrary "next" provider, since that's the deliberate safe default.
      if (wasActiveEngine) await switchProvider('aetox')
    }),
  })

  async function selectProvider(name: string) {
    if (!name) return
    // Walking away from a half-finished sign-in must release the listener it
    // opened, not leave it waiting for a redirect nobody will send.
    if (signInPrompt && signInPrompt.provider !== name) await abortSignIn()
    const switching = selected !== name
    selected = name
    errorMsg = ''
    if (switching) keyDraft = ''
    connTesting = {}
    connResult = {}
    baseURL = await ProviderBaseURL(name)
    baseURLDraft = baseURL
    baseURLIsCustom = await ProviderBaseURLIsCustom(name)
    keyPageURL = await ProviderAPIKeyURL(name)
    // Not awaited: a slow provider must not hold up the rest of the card.
    loadAccount(name)
    wireFormats = await ProviderWireFormats(name)
    loadingModels = true
    models = []
    modelFilter = ''
    freeOnly = false
    priced = {}
    try {
      const res = await ListModelsForProvider(name)
      models = Array.isArray(res) ? res : []
      // Prices for the list that is on screen, not for a list fetched again:
      // OpenRouter alone returns 411 models and a second discovery call could
      // answer differently. Not awaited — the names are usable before the
      // money arrives.
      if (models.length) {
        PriceModels(name, models)
          .then((rows) => {
            const next: Record<string, ModelListing> = {}
            for (const r of rows ?? []) next[r.model] = r
            priced = next
          })
          .catch(() => { priced = {} })
        ModelPriceSource()
          .then((src) => { priceSource = { name: src?.name ?? '', fetched: src?.fetched ?? '' } })
          .catch(() => { priceSource = { name: '', fetched: '' } })
      }
      // Discovery just proved this endpoint answers. If the engine is still on
      // the fallback from a switch made while it was down, this is the moment
      // it can get off — otherwise the warning sits there next to the model
      // list that disproves it.
      if (models.length > 0 && cockpit.model.warning) await retryActiveProvider()
    } finally {
      loadingModels = false
    }
  }

  async function run(label: string, fn: () => Promise<void>) {
    busy = label
    errorMsg = ''
    try {
      await fn()
    } catch (err) {
      errorMsg = String(err)
    } finally {
      busy = ''
    }
  }

  const useProvider = () => run('provider', async () => {
    await switchProvider(selected)
  })

  const useModel = (m: string) => run(m, async () => {
    if (!isActiveProvider) await switchProvider(selected)
    await switchModel(m)
  })

  // Runtime identifiers ("anthropic", "openai-compatible") aren't meant for
  // display; map to a short human label. Falls back to the raw value for any
  // future format this list doesn't know about yet.
  function wireFormatLabel(format: string): string {
    switch (format) {
      case 'anthropic': return 'Anthropic'
      case 'openai-compatible': return 'OpenAI'
      case 'responses': return 'Responses'
      default: return format
    }
  }

  const useFormat = (fmt: string) => run('format:' + fmt, async () => {
    if (!isActiveProvider) await switchProvider(selected)
    await switchWireFormat(fmt)
  })

  // Connection test: a real 1-token completion through the chat path, run per
  // model so a model can be proven before switching to it.
  //
  // Deliberately outside run(). That helper takes the page's single `busy`
  // lock, which is the right shape for switching a provider or saving a key —
  // one of those at a time is the only sane number. A probe is not one of
  // those: it changes nothing, it waits however long the provider waits, and
  // holding the whole page while it does meant a list of twelve models could
  // only be checked twelve waits deep, one after another. Owner, 22 ส.ค.:
  // "กดแล้วไม่ต้องรอ ไปกดอันอื่นได้เลย".
  //
  // Two records rather than two strings, for the same reason: one string holds
  // one answer, so testing a second model erased the first one's result — the
  // comparison you were running the tests to make.
  //
  // It also leaves errorMsg alone. A failed probe belongs under the row that
  // failed, where it stays; the page-level banner holds one message, and with
  // several probes in flight the last failure would speak for all of them.
  let connTesting = $state<Record<string, boolean>>({})
  let connResult = $state<Record<string, string>>({}) // per model: 'ok:…' | 'err:…'
  const testConnection = async (name: string) => {
    if (connTesting[name]) return
    // Which provider asked. A probe can now outlive the page state it started
    // in, so a late answer must not land under a row belonging to whatever
    // provider has been clicked into since. Only possible because these run
    // concurrently — the lock used to make it unreachable.
    const asked = selected
    connTesting[name] = true
    delete connResult[name]
    try {
      const ok = 'ok:' + await TestProviderConnection(asked, name)
      if (asked === selected) connResult[name] = ok
    } catch (err) {
      if (asked === selected) connResult[name] = 'err:' + String(err)
    } finally {
      delete connTesting[name]
    }
  }

  // Endpoint override. Saving '' clears it — that is the reset, so the button
  // is enabled on an empty box rather than treated as "nothing to save".
  let baseURLDraft = $state('')
  let baseURLIsCustom = $state(false)
  const saveBaseURL = (value: string) => run('baseUrl', async () => {
    await setProviderBaseURL(selected, value)
    if (keyDraft.trim()) {
      await submitAPIKey(selected, keyDraft.trim())
      keyDraft = ''
    }
    await refreshProviders()
    await selectProvider(selected)
  })

  const saveKey = () => run('key', async () => {
    if (baseURLDraft.trim() !== baseURL) {
      await setProviderBaseURL(selected, baseURLDraft.trim())
    }
    const key = keyDraft.trim()
    if (!key) return
    await submitAPIKey(selected, key)
    keyDraft = ''
    await refreshProviders()
    await selectProvider(selected)
  })

  // ---------- MCP servers ----------
  type MCPRow = {
    name: string; command?: string[]; url?: string
    environment?: Record<string, string>; headers?: Record<string, string>
    cwd?: string; timeoutMs?: number
    disabled: boolean; status: string; tools: number; err?: string
    // The allowlist, if one was written. `tools` above is the count the server
    // offers; this is which of them are taken. Absent means all.
    allowed?: string[]
    // Who carries this server's tools. Absent from the row means the engine
    // sent nothing, which is not the same as "nobody" — treated as [] here and
    // shown as attached nowhere.
    for?: string[]
  }
  // `detail` is the desk's own description — a paragraph, which is why it is a
  // tooltip and `name` is what the chip prints.
  type MCPTargetRow = { id: string; name: string; detail?: string; kind: string }
  let mcpServers = $state<MCPRow[]>([])
  // Everywhere a server can be pointed, from the engine — the desks and the
  // team that actually exist. Not a list typed in here, so hiring an agent
  // puts it on these switches without this page being edited.
  let mcpTargets = $state<MCPTargetRow[]>([])
  let mcpBusy = $state('')
  let mcpError = $state('')

  async function loadMCP() {
    mcpServers = await ListMCPServers()
    mcpTargets = await PlacementTargets()
    mcpLoaded = true
  }

  async function runMCP(label: string, fn: () => Promise<void>) {
    mcpBusy = label
    mcpError = ''
    try {
      await fn()
    } catch (err) {
      mcpError = String(err)
    } finally {
      mcpBusy = ''
    }
  }

  // Install the server an agent says it is missing, and place the agent on it,
  // from the agent's own card.
  //
  // **The door used to lead to a page rather than to the fix.** An agent that
  // declares `needs: mcp:firecrawl` sent the user to the MCP section, where the
  // thing they needed sat in a shelf of seven with nothing saying which one it
  // was — owner, 2026-08-14: *"คนเขาไม่รู้หรอกอันไหนเราทำไว้เพื่อตัวไหน"*. That
  // is true, and it is the whole complaint: not the clicking, the matching.
  //
  // **Why this is a button and not a default.** Installing a bundled agent's
  // server at startup would wire a fresh install to a third party the user never
  // chose — for firecrawl, an outbound dependency on mcp.firecrawl.dev before
  // anyone has opened the agent once. It also inverts needs.go's one rule:
  // `needs:` declares and never grants, `for:` grants and is the only thing that
  // does. Pressed here, the user is standing in front of the agent that wants
  // it, with the reason on screen — which is the same one click, and still
  // their decision. What is removed is the matching, not the consent.
  //
  // Only for a preset that connects with no key. One that wants a token pasted
  // cannot be finished in one press, so it keeps the door to the page, where the
  // form is waiting with the header names already filled in.
  const installNeeded = (o: subagent.Need) => runMCP('need:' + o.id, async () => {
    const p = presetFor(o.id)
    if (!p) return
    await SaveMCPServer('', await presetConfig(p))
    await loadMCP()
    // Placing it is the half that makes the need met — installing alone leaves
    // the card saying "unplaced", which from the user's side is the same button
    // having done nothing.
    //
    // The agent and nobody else, replacing what the add just wrote rather than
    // adding to it. A plain add lands on the general desks now
    // (config.MCPDefaultDesks, applied in SaveMCPServer), which is right for a
    // server the user picked off the shelf and wrong for this one: pressed
    // here, the answer to "who is this for" is on screen — the agent whose card
    // it is — and the line above about keeping an agent's server off the main
    // assistant's tool block is the whole reason `for:` exists. Appending would
    // have handed firecrawl to the assistant as a side effect of meeting
    // research's need.
    if (agentMCPId) {
      const row = mcpServers.find((s) => s.name === p.name)
      if (row) {
        await SetMCPServerTargets(p.name, [agentMCPId])
        await loadMCP()
      }
    }
    if (agentReachFor) agentNeeds = await AgentNeeds(agentReachFor)
  })

  // ---------- Speech ----------
  // The tool register (what the assistant runs) left for ห้องความสามารถ on
  // 14 ก.ย. 2026; what stays is the setting audio_transcribe and the mic run
  // on, which is a choice and not a list.
  let speechOpen = $state(false)

  // ---------- Speech model (what audio_transcribe runs on) ----------
  // Models differ by an order of magnitude in size and accuracy, and a machine
  // can hold several — including ones Ollama or LM Studio already downloaded.
  // Without this the engine just took whichever it found first.
  type SpeechRow = { path: string; name: string; sizeMB: number; store: string; managed: boolean; active: boolean }
  let speechModels = $state<SpeechRow[]>([])
  let speechStatus = $state('') // engine's own reason it cannot run; '' means ready
  let speechBusy = $state(false)
  let speechError = $state('')

  let speechDirs = $state<{ path: string; label: string }[]>([])

  // Below the state it reads, not above it: $derived is lazy so the old
  // ordering worked at runtime, but it put speechModels in its own temporal
  // dead zone as far as the compiler was concerned.
  const activeSpeechLabel = $derived(
    speechModels.find((m) => m.active)?.name ?? t('settings.speechAuto'),
  )

  async function loadSpeech() {
    speechModels = await ListSpeechModels()
    speechStatus = await SpeechStatus()
    speechDirs = await SpeechModelDirs()
  }

  // '' pins nothing, which is how the user gets back to auto-discovery.
  async function pickSpeechModel(path: string) {
    speechBusy = true
    speechError = ''
    try {
      await SetSpeechModel(path)
      await loadSpeech()
      speechOpen = false // the choice is made; leaving it open just covers the page
    } catch (err) {
      speechError = String(err) // stays open so the reason is readable
    } finally {
      speechBusy = false
    }
  }

  // ---------- Picture page (image_make's vendor and model) ----------
  //
  // Half of the voice page's shape and none of its extra questions: there is no
  // voice to choose, no model FILE on disk to point at, and no local engine to
  // install — every vendor here is an HTTP call. What is left is the two picks
  // and the one status line.
  let imageEngines = $state<EngineRow[]>([])
  let imageStatus = $state('') // the vendor's own reason it cannot run; '' means ready
  let imagePageBusy = $state(false)
  let imagePageError = $state('')

  const activeImageEngine = $derived(imageEngines.find((e) => e.active))

  // Same shape, same reason as the voice card's picks below: the two selects
  // read state that every load re-asserts, so a pick that does not land
  // (parked mid-turn, or refused) cannot leave the control claiming it did.
  let imagePick = $state('')
  let imageModelPick = $state('')

  async function syncImagePicks() {
    await tick() // the options are rendered from the rows this load replaced
    imagePick = imageEngines.find((e) => e.active)?.id ?? ''
    imageModelPick = imageEngines.find((e) => e.active)?.activeModel ?? ''
  }

  // One click from "this vendor has no key" to the box that takes one.
  async function goToProviderKey(provider: string) {
    openSection('models')
    await selectProvider(provider)
  }

  async function loadImagePage() {
    try {
      imageEngines = await ListImageEngines()
      imageStatus = await ImageStatus()
    } finally {
      await syncImagePicks()
    }
  }

  async function pickImageEngine(id: string) {
    imagePick = id
    imagePageBusy = true
    imagePageError = ''
    try {
      await SetImageEngine(id)
      await loadImagePage()
    } catch (err) {
      imagePageError = String(err)
      await syncImagePicks()
    } finally {
      imagePageBusy = false
    }
  }

  async function pickImageModelName(name: string) {
    imageModelPick = name
    imagePageBusy = true
    imagePageError = ''
    try {
      await SetImageModelName(name)
      await loadImagePage()
    } catch (err) {
      imagePageError = String(err)
      await syncImagePicks()
    } finally {
      imagePageBusy = false
    }
  }

  // ---------- คลังสตูดิโอ (desktop/studio_library.go) ----------
  // Folders the user pointed at, with counts — never the rows. The scan runs
  // in the engine and reports over two events, the same shape the capability
  // downloads use; the page only draws what arrives.
  let studioLibs = $state<engine.StudioLibraryView[]>([])
  let studioScanning = $state(false)
  let studioProgress = $state<{ root: string; done: number; total: number } | null>(null)
  let studioError = $state('')
  // What the last scan found, shown once as a card until dismissed — the
  // one moment a person wants to know whether the app understood their
  // folder, and the old page said nothing at all right then.
  let studioResult = $state<{ files: number; bytes: number; counts: Record<string, number> } | null>(null)
  // The browser (StudioBrowser.svelte) is open when this is set: a kind from
  // a tile, a shelf from a card, or both empty for everything. Null is closed.
  let studioBrowse = $state<{ kind: string; library: string } | null>(null)
  // The "หาวัตถุดิบเพิ่ม" sheet (StudioSourcesSheet.svelte).
  let studioSourcesOpen = $state(false)
  // The shelf is for two agents, and the page should hand you to them —
  // otherwise a person who has just stocked it goes looking for where the
  // work happens. Same three moves VideoWork.start makes: leave the page,
  // show the chat, boot the chair; the view moves first so the click is
  // not a dead click while the session comes up.
  async function talkToVideoAgent(agent: 'video' | 'editor') {
    onClose()
    setActiveView('chat')
    await newChairSession(agent)
  }
  // A tab picks the kind and keeps whatever shelf filter is on; a card's
  // "ดูของ" picks the shelf and shows every kind of it. Pressing the open tab
  // again does nothing — a tab row always has one open.
  function browseStudio(kind = '', library = '') {
    studioBrowse = { kind, library }
  }
  function browseKind(k: string) { browseStudio(k, studioBrowse?.library ?? '') }

  // True until the first answer, so the page draws placeholder cards rather
  // than an empty strip that then jumps when the shelves arrive.
  let studioLoading = $state(true)
  async function loadStudio() {
    try {
      studioLibs = (await StudioLibraries()) ?? []
      studioScanning = await StudioScanning()
      // The browser opens on the first kind that holds anything, so the
      // page reads as tabs over content from the first frame — a row of
      // seven numbers with nothing under them read as a dashboard (owner,
      // 12 ก.ย.: "ไม่รู้เลยว่ากดได้").
      if (!studioBrowse) {
        const first = STUDIO_KINDS.find((k) => studioTotals[k] > 0)
        if (first) studioBrowse = { kind: first, library: '' }
      }
    } catch (err) {
      studioError = String(err)
    } finally {
      studioLoading = false
    }
  }

  $effect(() => {
    const offProgress = EventsOn('studio:progress', (p: { root: string; done: number; total: number }) => {
      studioScanning = true
      studioProgress = p
    })
    const offDone = EventsOn('studio:done', (d: { ok: boolean; error?: string; files?: number; bytes?: number; counts?: Record<string, number> }) => {
      studioScanning = false
      studioProgress = null
      if (!d.ok) studioError = d.error ?? t('settings.studioScanFailed')
      else studioResult = { files: d.files ?? 0, bytes: d.bytes ?? 0, counts: d.counts ?? {} }
      void loadStudio()
    })
    return () => { offProgress(); offDone() }
  })

  async function addStudioFolder() {
    studioError = ''
    try {
      // false is a dismissed dialog or a scan already running — neither is news.
      if (await AddStudioLibrary()) studioScanning = true
    } catch (err) {
      studioError = String(err)
    }
  }

  async function rescanStudio(id: string) {
    studioError = ''
    try {
      if (await RescanStudioLibrary(id)) studioScanning = true
    } catch (err) {
      studioError = String(err)
    }
  }

  // Forgetting a shelf is not deleting it, and the dialog says so: the one
  // fear a person has pressing this is that thirty gigabytes goes with it.
  function removeStudio(lib: engine.StudioLibraryView) {
    askConfirm({
      title: t('settings.studioRemoveTitle'),
      message: lib.root,
      detail: t('settings.studioRemoveDetail'),
      confirmLabel: t('settings.studioRemove'),
      run: async () => {
        try {
          studioLibs = (await RemoveStudioLibrary(lib.id)) ?? []
        } catch (err) {
          studioError = String(err)
        }
      },
    })
  }

  const GB = 1024 * 1024 * 1024
  const gb = (bytes: number) => bytes >= GB ? `${(bytes / GB).toFixed(1)} GB` : `${Math.max(1, Math.round(bytes / (1024 * 1024)))} MB`
  const STUDIO_KINDS = ['sfx', 'music', 'overlay', 'background', 'clip', 'icon', 'image'] as const
  const STUDIO_KIND_ICON: Record<string, IconName> = { sfx: 'volume2', music: 'headphones', overlay: 'sparkles', background: 'monitor', clip: 'clapperboard', icon: 'puzzle', image: 'image' }
  const studioKindLabel = (k: string) => t(`settings.studioKind.${k}` as TKey)
  // Seven tiles over every shelf. A kind with nothing in it stays on the
  // strip, dimmed: the strip is also the list of what the shelf can hold.
  const studioTotals = $derived.by(() => {
    const out: Record<string, number> = {}
    for (const k of STUDIO_KINDS) out[k] = 0
    for (const lib of studioLibs) for (const k of STUDIO_KINDS) out[k] += lib.counts?.[k] ?? 0
    return out
  })
  const studioFolderName = (root: string) => root.replace(/[\\/]+$/, '').split(/[\\/]/).pop() ?? root
  // A source counts as imported when a shelf's folder carries its name —
  // a Drive download keeps the folder's name, so this is the honest match.
  const studioImported = (folder?: string) => !!folder && studioLibs.some((l) => l.name.toLowerCase() === folder.toLowerCase())
  // Chips per kind for one shelf or one result, zeros left out.
  function studioKindChips(counts: Record<string, number> | undefined): { k: string; n: number }[] {
    return STUDIO_KINDS.map((k) => ({ k, n: counts?.[k] ?? 0 })).filter((c) => c.n > 0)
  }
  // One line per shelf: "1,204 files · 12.3 GB · sfx 340 · overlay 88 …", kinds
  // with nothing in them left out, because a zero is not information.
  function studioLine(lib: engine.StudioLibraryView): string {
    const parts = [t('settings.studioFiles', { n: lib.files.toLocaleString() }), gb(lib.bytes)]
    for (const k of STUDIO_KINDS) {
      const n = lib.counts?.[k] ?? 0
      if (n > 0) parts.push(`${t(`settings.studioKind.${k}` as TKey)} ${n.toLocaleString()}`)
    }
    if (lib.unread > 0) parts.push(t('settings.studioUnread', { n: lib.unread }))
    return parts.join(' · ')
  }

  // ---------- Voice page (STT + TTS, both vendor-switchable) ----------
  // Two catalogs rendered as two pickers: internal/stt for the mic and
  // audio_transcribe, internal/tts for reading replies aloud. A new vendor is
  // a catalog entry in Go — nothing on this page changes.
  type EngineRow = { id: string; label: string; install: string; active: boolean; hasModels: boolean; installCommand: string[]; models: string[]; activeModel: string }
  type TtsVoiceRow = { id: string; name: string; lang: string; gender: string; active: boolean }
  let sttEngines = $state<EngineRow[]>([])
  let ttsEngines = $state<EngineRow[]>([])
  let ttsVoicesList = $state<TtsVoiceRow[]>([])
  let ttsStatus = $state('') // TTS engine's own reason it cannot run; '' means ready
  let voicePageBusy = $state(false)
  let voicePageError = $state('')
  let ttsPreviewing = $state(false)
  let ttsPreviewAudio: HTMLAudioElement | null = null

  const activeSttEngine = $derived(sttEngines.find((e) => e.active))
  const activeTtsEngine = $derived(ttsEngines.find((e) => e.active))

  // The five selects on this card read their own state rather than the row
  // lists, and syncVoicePicks re-asserts it after every call. A one-way value={derived} cannot correct a pick that did not
  // land: the derived id comes back UNCHANGED, so Svelte has nothing to
  // update and the DOM keeps the option the user clicked. That is not
  // hypothetical — applyConfig parks a config change while a turn is in
  // flight, so a vendor switched mid-answer leaves the page claiming a vendor
  // the app is not on, with the model row below it still showing the old
  // one's files (found from a screenshot, 2026-09-08).
  let sttPick = $state('')
  let sttModelPick = $state('')
  let ttsPick = $state('')
  let ttsModelPick = $state('')
  let ttsVoicePick = $state('')

  // After a tick, deliberately: the option lists these five selects choose
  // from are rendered from the rows this same load just replaced, and a value
  // written into a select whose options do not exist yet is dropped by the DOM
  // with nothing to re-run it.
  async function syncVoicePicks() {
    await tick()
    sttPick = sttEngines.find((e) => e.active)?.id ?? ''
    sttModelPick = sttEngines.find((e) => e.active)?.activeModel ?? ''
    ttsPick = ttsEngines.find((e) => e.active)?.id ?? ''
    ttsModelPick = ttsEngines.find((e) => e.active)?.activeModel ?? ''
    ttsVoicePick = ttsVoicesList.find((v) => v.active)?.id ?? ''
  }

  async function loadVoicePage() {
    // Hardware first and separately: a webview that cannot enumerate devices
    // is no reason for the engine pickers below to stay empty.
    try {
      await refreshAudioDevices()
    } catch {
      // Lists stay as they are; the rows fall back to "ตัวที่ Windows ตั้งไว้".
    }
    // finally, not the happy path: whatever half of this load succeeded, the
    // five controls must end up showing what the app is actually on.
    try {
      sttEngines = await ListSpeechEngines()
      ttsEngines = await ListTTSEngines()
      ttsStatus = await TTSStatus()
      await loadSpeech()
      try {
        ttsVoicesList = await ListTTSVoices()
      } catch (err) {
        // No voices is not a page failure — the status line already carries the
        // engine's reason; keep whichever message is more specific.
        ttsVoicesList = []
        if (!ttsStatus) ttsStatus = String(err)
      }
    } finally {
      await syncVoicePicks()
    }
  }

  async function voiceAction(fn: () => Promise<void>) {
    voicePageBusy = true
    voicePageError = ''
    try {
      await fn()
      await loadVoicePage()
    } catch (err) {
      voicePageError = String(err)
      // A refused pick must not stay on screen either: the message says what
      // went wrong, and the control goes back to what the app is on.
      await syncVoicePicks()
    } finally {
      voicePageBusy = false
    }
  }

  // Each pick writes the clicked value into the select's own state before the
  // call, so that syncVoicePicks writing the app's answer back afterwards is a
  // CHANGE — which is the only thing that makes Svelte touch the DOM again.
  const pickSttEngine = (id: string) => { sttPick = id; return voiceAction(() => SetSpeechEngine(id)) }
  const pickTtsEngine = (id: string) => { ttsPick = id; return voiceAction(() => SetTTSEngine(id)) }
  const pickTtsVoice = (id: string) => { ttsVoicePick = id; return voiceAction(() => SetTTSVoice(id)) }
  const pickSttModelName = (name: string) => { sttModelPick = name; return voiceAction(() => SetSpeechModelName(name)) }
  const pickTtsModelName = (name: string) => { ttsModelPick = name; return voiceAction(() => SetTTSModelName(name)) }

  // ---------- ติดตั้ง engine จากในแอป ----------
  // The command on screen IS the command that runs: rows carry the catalog's
  // own argv (VoiceEngineInfo.installCommand) for display, and the button
  // sends back only (side, id) — InstallVoiceEngine re-reads the catalog, so
  // the webview cannot compose a command. The tail is pip's latest line, one
  // line tall on purpose: a chatty install must not move the page.
  let voiceInstallBusy = $state<'' | 'stt' | 'tts'>('')
  let voiceInstallTail = $state('')
  let voiceInstallDone = $state<Record<string, string>>({}) // side -> label just installed
  let voiceInstallFail = $state<Record<string, string>>({}) // side -> the verdict sentence

  async function installVoiceEngine(side: 'stt' | 'tts', eng: EngineRow) {
    voiceInstallBusy = side
    voiceInstallTail = ''
    voiceInstallDone = { ...voiceInstallDone, [side]: '' }
    voiceInstallFail = { ...voiceInstallFail, [side]: '' }
    const off = EventsOn('voice:install', (p: { side?: string; line?: string }) => {
      if (p?.side === side && p.line) voiceInstallTail = p.line
    })
    try {
      await InstallVoiceEngine(side, eng.id)
      voiceInstallDone = { ...voiceInstallDone, [side]: eng.label }
      // The page re-checks for itself — the green line below is the only
      // celebration; the red status clearing is the proof.
      await loadVoicePage()
    } catch (err) {
      voiceInstallFail = { ...voiceInstallFail, [side]: String(err) }
    } finally {
      off()
      voiceInstallBusy = ''
      voiceInstallTail = ''
    }
  }

  let voiceCmdCopied = $state('')
  async function copyVoiceCommand(side: string, argv: string[]) {
    try {
      await navigator.clipboard.writeText(argv.join(' '))
      voiceCmdCopied = side
      setTimeout(() => (voiceCmdCopied = ''), 1500)
    } catch {
      /* clipboard blocked — the command is on screen to be typed */
    }
  }

  // ลองฟัง — one short sentence through the exact path the chat's ฟัง button
  // takes, so what this proves is what the user will get.
  async function previewTts() {
    if (ttsPreviewing) {
      ttsPreviewAudio?.pause()
      ttsPreviewAudio = null
      ttsPreviewing = false
      return
    }
    voicePageError = ''
    ttsPreviewing = true
    try {
      const url = await SpeakText(t('settings.ttsPreviewText'))
      const audio = new Audio(url)
      // Through the picked speaker too, or ลองฟัง would be proving a path the
      // chat does not take.
      await applySpeaker(audio)
      ttsPreviewAudio = audio
      audio.onended = () => { ttsPreviewing = false; ttsPreviewAudio = null }
      audio.onerror = () => { ttsPreviewing = false; ttsPreviewAudio = null }
      await audio.play()
    } catch (err) {
      voicePageError = String(err)
      ttsPreviewing = false
      ttsPreviewAudio = null
    }
  }


  // ---------- Usage stats ----------
  // cacheRows counts the calls whose provider reported cache accounting at all.
  // Zero means "no cache to report" (a local runtime), which must render as an
  // em dash — a 0% hit rate would be a claim the provider never made.
  // provider is half of what a row IS, not a decoration on it: usageByModel
  // groups by model AND provider, because the same model id is sold per token
  // by one company and included in a subscription by another.
  // These used to be hand-written copies of the Go structs, and a copy of a
  // shape is a second place answering the same question: `cost`, `pricedCalls`
  // and `pricesFetched` were added to usage.go and never here, so the markup
  // below read three fields the local type said did not exist. The `as Usage`
  // cast on the call is what let it compile anyway. The generated bindings are
  // the one description of what the Go side returns; use them.
  type UsageRow = engine.UsageRow
  type Usage = engine.UsageStats

  let usage = $state<Usage | null>(null)
  let usageError = $state('')
  let usagePeriod = $state<'today' | 'week' | 'all'>('week')

  // null is a third answer, not an empty one: nobody has asked yet, the effect
  // below has not run, or the engine is still walking the history. It covers
  // the frame before the load starts as well as the load itself, which a
  // busy-flag set inside loadUsage would not. A reload with data already on
  // screen keeps the data — a skeleton flashed over numbers the user is
  // reading would be motion that says nothing.
  const usagePending = $derived(!usage && !usageError)

  async function loadUsage() {
    usageError = ''
    try {
      usage = await UsageStats()
    } catch (err) {
      usageError = String(err)
    }
  }

  const fmtTokens = (n: number) => n.toLocaleString('en-US')
  // Headline numbers reach eight digits; the cards need the shape, not the digits.
  const fmtCompact = (n: number) =>
    n >= 1e9 ? (n / 1e9).toFixed(1) + 'B'
    : n >= 1e6 ? (n / 1e6).toFixed(1) + 'M'
    : n >= 1e4 ? Math.round(n / 1e3) + 'K'
    : n.toLocaleString('en-US')
  const pct = (part: number, whole: number) => (whole > 0 ? Math.round((part / whole) * 100) : 0)
  const dayKey = (d: Date) =>
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

  // `?? []` because a Go nil slice arrives as JSON null, and a period with no
  // rows (today, before the first call of the day) used to do exactly that —
  // `.length` on it threw mid-render and the period buttons looked dead.
  const usageRows = $derived((usage ? usage[usagePeriod] : null) ?? [])
  const usageTotal = (r: UsageRow) => r.promptTokens + r.completionTokens
  const periodTotal = $derived(usageRows.reduce((sum, r) => sum + usageTotal(r), 0))

  // Colour follows the entity, not its row number: the top five all-time
  // models take the five slots and keep them, so switching the period filter
  // never repaints the models that survive it. The tail shares one mute slot —
  // a sixth hue could not stay distinguishable under colour-vision deficiency.
  // Hue belongs to the model and not to the row: the chart aggregates on the
  // model alone (usageByDay groups on it), and two rows of one model are one
  // model on two bills. Deduped by name, so the five slots hold five models
  // rather than four and a repeat.
  const allModels = $derived([...new Set((usage?.all ?? []).map((r) => r.model))])
  const topModels = $derived(allModels.slice(0, 5))
  const seriesOf = $derived.by(() => {
    const map = new Map<string, number>()
    const sorted = [...topModels].sort()
    sorted.forEach((model, i) => map.set(model, i + 1))
    return map
  })
  const slotOf = (model: string) => seriesOf.get(model) ?? 0

  // A row is a (provider, model) pair, and only the pair identifies it. The
  // name stopped being an identity the day the provider column arrived (db.go
  // migration 15): every model used on both sides of that line now has two
  // rows, one under the provider that served it and one under the blank the
  // older rows carry. Keyed on the name those two are one row claimed twice,
  // and Svelte refuses a keyed list it cannot tell apart — it throws, the
  // section never renders, and what the user sees is a sidebar entry that does
  // nothing rather than a table that looks wrong.
  // The pair itself is the key, rather than the two halves glued with a
  // separator: a separator has to be a character neither half can contain, and
  // the honest candidates are control characters that do not belong typed into
  // a source file. Fourteen rows do not need the cheaper spelling.
  const rowKey = (r: UsageRow) => JSON.stringify([r.provider, r.model])

  // Round a maximum up to a clean axis top, so the ticks read 0 / 250K / 500K
  // instead of 0 / 231,904 / 463,808.
  const niceMax = (value: number) => {
    if (value <= 0) return 1
    const mag = Math.pow(10, Math.floor(Math.log10(value)))
    for (const step of [1, 1.5, 2, 2.5, 3, 4, 5, 7.5]) {
      if (value <= step * mag) return step * mag
    }
    return 10 * mag
  }

  // Every day in the window gets a column, including the empty ones. Plotting
  // only the days that have data turns a month into four fat blocks and quietly
  // rescales the x-axis — the gaps ARE the story on a usage chart.
  const CHART_DAYS = 30

  // A column carries two encodings at once: hue is the model, fill is where the
  // tokens came from. Stacking is kind-outer, model-inner, so the hit|miss and
  // in|out boundaries land at the same depth in every column and can be read
  // straight across — the model split then reads as hue inside each band.
  //
  // 'raw' is input from a model that reported no cache accounting that day. It
  // is its own band on purpose: folding it into miss would claim a cache the
  // provider never said it had, which is the same lie the table renders as "—".
  type Kind = 'hit' | 'miss' | 'raw' | 'out'
  const KINDS: Kind[] = ['hit', 'miss', 'raw', 'out']
  // Same words the headline card already uses for the same split — a second
  // vocabulary for hit/miss would make the two read as different measurements.
  const kindLabel: Record<Kind, string> = $derived({
    hit: t('settings.usageHit'),
    miss: t('settings.usageMiss'),
    raw: t('settings.usageInput'),
    out: t('settings.usageOutput'),
  })

  const dailyChart = $derived.by(() => {
    if (!usage) return null
    // day -> kind -> model -> tokens
    const byDay = new Map<string, Map<Kind, Map<string, number>>>()
    const add = (day: string, kind: Kind, model: string, value: number) => {
      if (value <= 0) return
      let kinds = byDay.get(day)
      if (!kinds) { kinds = new Map(); byDay.set(day, kinds) }
      let models = kinds.get(kind)
      if (!models) { models = new Map(); kinds.set(kind, models) }
      models.set(model, (models.get(model) ?? 0) + value)
    }
    for (const p of usage.daily) {
      if (p.cacheRows > 0) {
        add(p.day, 'hit', p.model, Math.min(p.cachedTokens, p.promptTokens))
        add(p.day, 'miss', p.model, p.promptTokens - p.cachedTokens)
      } else {
        add(p.day, 'raw', p.model, p.promptTokens)
      }
      add(p.day, 'out', p.model, p.completionTokens)
    }
    if (byDay.size === 0) return null

    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const days = []
    for (let i = CHART_DAYS - 1; i >= 0; i--) {
      const d = new Date(today)
      d.setDate(d.getDate() - i)
      const key = dayKey(d)
      const kinds = byDay.get(key)
      const parts: { kind: Kind; model: string; value: number }[] = []
      const byKind = {} as Record<Kind, number>
      const byModel = new Map<string, number>()
      for (const kind of KINDS) {
        const models = [...(kinds?.get(kind) ?? new Map<string, number>())]
          // Stack in slot order so a model sits at the same depth every column.
          .sort((a, b) => slotOf(a[0]) - slotOf(b[0]))
        byKind[kind] = models.reduce((s, [, value]) => s + value, 0)
        for (const [model, value] of models) {
          parts.push({ kind, model, value })
          byModel.set(model, (byModel.get(model) ?? 0) + value)
        }
      }
      const models = [...byModel].sort((a, b) => slotOf(a[0]) - slotOf(b[0]))
      days.push({ day: key, total: parts.reduce((s, p) => s + p.value, 0), parts, byKind, models })
    }
    const max = niceMax(Math.max(...days.map((d) => d.total)))
    // Four gridlines top-down, the last being the baseline.
    const ticks = [1, 0.75, 0.5, 0.25, 0].map((f) => ({ frac: f, value: Math.round(max * f) }))
    return { days, max, ticks }
  })

  // Five x-labels evenly spaced; more collide at this width.
  const chartXLabels = $derived.by(() => {
    const days = dailyChart?.days ?? []
    if (days.length === 0) return []
    const every = Math.max(1, Math.round(days.length / 5))
    return days.map((d, i) => (i % every === 0 || i === days.length - 1 ? d.day.slice(5) : ''))
  })

  let hoverDay = $state<number | null>(null)
  const hoveredColumn = $derived(hoverDay === null ? null : (dailyChart?.days[hoverDay] ?? null))

  // 26 whole weeks ending with the current one. Cells past today are rendered
  // blank rather than as zero-activity days that have not happened yet.
  const heatmap = $derived.by(() => {
    const totals = new Map<string, number>()
    for (const p of usage?.heatmap ?? []) {
      totals.set(p.day, (totals.get(p.day) ?? 0) + p.promptTokens + p.completionTokens)
    }
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const end = new Date(today)
    end.setDate(end.getDate() + (6 - end.getDay()))
    const cells: { day: string; value: number; future: boolean }[] = []
    for (let i = 26 * 7 - 1; i >= 0; i--) {
      const d = new Date(end)
      d.setDate(d.getDate() - i)
      const key = dayKey(d)
      cells.push({ day: key, value: totals.get(key) ?? 0, future: d > today })
    }
    const max = Math.max(1, ...cells.map((c) => c.value))
    const weeks: (typeof cells)[] = []
    for (let w = 0; w < 26; w++) weeks.push(cells.slice(w * 7, w * 7 + 7))
    return { weeks, max }
  })
  const heatLevel = (value: number, max: number) => (value <= 0 ? 0 : Math.min(4, Math.ceil((value / max) * 4)))

  $effect(() => {
    if (active === 'usage') void loadUsage()
  })

  // ---------- Prompt presets ----------
  // Left this file 14 ก.ย. 2026 for ห้องความสามารถ › ชุดคำสั่ง
  // (Capability.svelte), the editor and the gallery whole.

  // ---------- Sub-agents (ARCHITECTURE.md §44) ----------
  // Only sub-agents live here. The main agent is the assistant — one identity,
  // configured by the identity files — and is not chosen from a list (§44.0).
  type SubagentRow = {
    name: string; description: string; model?: string; provider?: string; think?: string
    tools?: string[]; deny?: string[]; steps?: number; prompt: string
    path?: string; builtin: boolean; overrides?: boolean; invalid?: string; notice?: string; icon?: string
    // The look a profile may name for itself (profile.go). Blank on almost
    // every row and blank is the answer: the drawing derives it from the name
    // (agentLook.ts). Carried so that a page which SHOWS an agent shows the
    // one its owner chose — an override that only the settings editor
    // honoured would be one robot with two looks.
    shell?: string; top?: string; face?: string; accent?: string; hue?: string
    // Already resolved by the backend (applyHomeRules fills the default), which
    // is why the editor shows this rather than the raw `desk:` it keeps: the
    // default is a constant in internal/mode and spelling it again here is how
    // the screen ends up naming a desk the engine does not use.
    desk?: string
    // What the profile declares it needs, verbatim — `mcp:firecrawl`, or an
    // alternation like `connection:n8n | mcp:windmill`. Carried so the MCP
    // shelf can say which agents asked for a server without restating it
    // (agentsNeeding); the *state* of a need is the engine's answer, not this.
    needs?: string[]
  }
  // Searching the roster. Name and description both, because half the time the
  // thing a person remembers about an agent is what it does rather than what it
  // is called. No filter control beside it: this page is already split into
  // "yours" and "built-in", which is the same question a filter would ask and
  // answers it without a click.
  let agentQuery = $state('')
  const matchesQuery = (a: SubagentRow) => {
    const q = agentQuery.trim().toLowerCase()
    if (!q) return true
    return a.name.toLowerCase().includes(q) || (a.description ?? '').toLowerCase().includes(q)
  }
  // Every profile, both kinds — kept whole because the shared editor opens
  // agents here too (the team page's doors land on it with a name). What the
  // *lists* below draw is only the sub-agents: the agents' roster is the team
  // page, and one profile on two rosters is the overlap this split ended.
  let subagents = $state<SubagentRow[]>([])
  // Split by who wrote it, which is what a user actually asks of this page. A
  // file of yours that shadows a bundled one counts as yours — it IS your file —
  // and carries a badge saying so, because deleting it reverts rather than removes.
  // One split, applied twice: which kind, then who wrote it. Both pages read
  // these — the markup is shared (profileListPane), so a rule added here lands
  // on both without either page knowing about the other.
  const teamRows = $derived({
    mine: subagents.filter((a) => !a.builtin && chairNames.has(a.name)),
    builtin: subagents.filter((a) => a.builtin && chairNames.has(a.name)),
  })
  // A helper has no "yours" pile: the bundled set is the whole set, and a
  // file of yours over one of them is a shadow that stays in its place on the
  // one list, wearing the ทับของแอป chip (12 ก.ย. 2026 — the helpers' door
  // reopened for the model, the prompt and the look; profile.go
  // limitHelperShadow is the limit).
  const helperRows = $derived({
    mine: [] as SubagentRow[],
    builtin: subagents.filter((a) => (a.builtin || a.overrides) && !chairNames.has(a.name)),
  })
  // Which profiles are agents — asked of ListChairs, the same answer the team
  // page draws, never re-derived from a file's fields here.
  let chairNames = $state(new Set<string>())
  // null = the list. Anything else = the editor on that profile's raw file.
  let agentEditing = $state<SubagentRow | null>(null)
  let agentDraftName = $state('')
  // The .md file is still `--- key: value ---` plus a role prompt underneath —
  // that has not changed, and SaveSubagentProfile still only ever receives
  // that same text. What changed is that the editor stopped asking a person to
  // read and hand-edit it: each frontmatter key gets its own field below.
  // agentDraftModel is one of those fields (ag-sec agentSecBrain) and has been
  // since the editor grew its five sections. It was ALSO a dropdown repeated
  // down the list until 31 ส.ค., which is the copy that went: inheriting is the
  // rule and a pin is the exception, and a control drawn on every card made the
  // one agent that is pinned look exactly like the six that are not. The list
  // says so with a chip now, and the pin is set where the rest of what this
  // agent thinks with already lives.
  let agentDraftDescription = $state('')
  let agentDraftModel = $state('')
  // Which provider the model above lives at (owner, 12 ก.ย.: "ควรเลือกได้แม้แต่
  // ผู้ให้บริการ"). '' = the chat's, which is what every file said before.
  let agentDraftProvider = $state('')
  // How deep this agent thinks (owner, 13 ก.ย.: "ทำให้เราปรับระดับความคิดได้").
  // '' = the chat's level, which is what every file said before; anything else
  // is one of the levels the provider/model above actually has (agentThinkLevels).
  let agentDraftThink = $state('')
  let agentDraftTools = $state<string[]>([])
  let agentDraftDeny = $state<string[]>([])
  let agentDraftSteps = $state('')
  let agentDraftIcon = $state('')
  // The rest of the look — the mascot's identity dials (MASCOT.md §2). Same
  // rule as the icon and the same default: '' means "derive it", which is
  // what every profile nobody has opened says, and what the roster has always
  // drawn: the assistant's template in the hue the name gives.
  let agentDraftShell = $state('')
  let agentDraftTop = $state('')
  let agentDraftFace = $state('')
  // The colour is one dial with two spellings in the file: `accent:` names an
  // ACCENT row (a hue and how much of it — the same list the avatar page
  // offers, which is what lets a persona be handed over whole), and the older
  // `hue:` is a degree at full colour. The picker writes `accent:` and clears
  // `hue:`; a file that carries a degree keeps it until a colour is picked.
  // Both are kept as the string the file carries rather than parsed, so what
  // the editor holds is what the .md says — including a value somebody
  // hand-wrote that this build does not offer. lookOf() is the one place a
  // hue becomes a number, and it refuses anything that is not 0..360.
  let agentDraftAccent = $state('')
  let agentDraftHue = $state('')
  // What the STARTER CARD picker offers — a hand-picked subset of the app's
  // marks, not all of them: forty icons is a wall to scan and most of them mean
  // nothing on a card. Adding one is adding a name here.
  //
  // This list used to serve the agent's own icon too, and that was the bug: a
  // card's icon is drawn as itself, an agent's is drawn on the mascot's ears,
  // and the two rows answer different questions. The badge row reads
  // AGENT_BADGES (agentLook.ts), every one of which the ear can wear.
  const AGENT_ICONS: IconName[] = [
    'layoutList', 'fileText', 'chartColumn', 'fileCode', 'terminal', 'globe',
    'search', 'brain', 'palette', 'clapperboard', 'headphones', 'package',
    'compass', 'puzzle', 'bot',
  ]
  // What the preview draws from, which is what the ROSTER will draw from: the
  // name, and whatever of the face has been chosen. Before a name is typed
  // there is nothing to derive from, so it borrows the name field's own
  // placeholder rather than showing an empty tile — and it changes under you as
  // you type, which is the truest thing this page can say about where a face
  // comes from.
  const facePreviewName = $derived(agentDraftName.trim() || 'backend')
  const faceIsAuto = $derived(!agentDraftIcon && !agentDraftShell && !agentDraftTop && !agentDraftFace && !agentDraftAccent && !agentDraftHue)
  // The draft as one object, so the preview and every cell of every row below
  // are fed by the same call the roster is fed by. A row that built its own
  // overrides would be a second reading of the same six fields.
  const draftFace = $derived(lookOf({
    icon: agentDraftIcon, shell: agentDraftShell, top: agentDraftTop, face: agentDraftFace, accent: agentDraftAccent, hue: agentDraftHue,
  }))
  const resetFace = () => {
    agentDraftIcon = ''
    agentDraftShell = ''
    agentDraftTop = ''
    agentDraftFace = ''
    agentDraftAccent = ''
    agentDraftHue = ''
  }
  // A colour picked here is an accent; the degree the file may still carry
  // would otherwise win over it (rig.ts: a hue is full colour, and first).
  const pickAccent = (id: string) => {
    agentDraftAccent = id
    agentDraftHue = ''
  }
  // Wear a persona the user saved on the avatar page — this is what the slots
  // are for (avatarPrefs.svelte.ts): an agent designed here can put on a look
  // designed there, all four dials at once. The badge stays the agent's own;
  // a persona is a look, not an identity.
  const wearPersona = (i: number) => {
    const p = personas.slots[i]
    if (!p) return
    agentDraftShell = p.shell
    agentDraftTop = p.top
    agentDraftFace = p.face
    pickAccent(p.accent)
  }
  const wearsPersona = (i: number) => {
    const p = personas.slots[i]
    return !!p && p.shell === agentDraftShell && p.top === agentDraftTop && p.face === agentDraftFace && p.accent === agentDraftAccent && !agentDraftHue
  }

  let showFaceContexts = $state(false)
  let previewFaceState = $state<'idle' | 'think' | 'work' | 'done' | 'err'>('idle')
  let previewFaceOff = $state(false)

  let agentDraftPrompt = $state('')
  // The three roads into the role field beside typing (§256.5, owner 13 ก.ย.:
  // "เอาแบบเปิดไฟล์ + วางลิงก์แล้วดึง + เทมเพลต"): a file on this machine, a
  // link to a file kept on GitHub or Google Drive, a template with blanks.
  // Each lands TEXT in the field and nothing else — the file or link is where
  // the words came from, not something the agent keeps pointing at. A field
  // with words in it already asks before they are replaced.
  let agentFillLink = $state('')
  let agentFillBusy = $state(false)
  let agentFillError = $state('')
  function placeBrief(text: string) {
    agentFillError = ''
    if (agentDraftPrompt.trim() === '') {
      agentDraftPrompt = text
      agentBodyOpen = true
      return
    }
    askConfirm({
      title: t('settings.agentFillReplaceTitle'),
      message: t('settings.agentFillReplaceMessage'),
      confirmLabel: t('settings.agentFillReplaceAction'),
      run: () => { agentDraftPrompt = text; agentBodyOpen = true },
    })
  }
  async function fillFromFile() {
    if (agentFillBusy) return
    agentFillBusy = true
    agentFillError = ''
    try {
      const text = await PickAgentBrief()
      if (text) placeBrief(text)
    } catch (err) {
      agentFillError = String(err)
    } finally {
      agentFillBusy = false
    }
  }
  async function fillFromLink() {
    const link = agentFillLink.trim()
    if (!link || agentFillBusy) return
    agentFillBusy = true
    agentFillError = ''
    try {
      placeBrief(await FetchAgentBrief(link))
      agentFillLink = ''
    } catch (err) {
      agentFillError = String(err)
    } finally {
      agentFillBusy = false
    }
  }
  function fillFromTemplate(id: string) {
    const tp = AGENT_TEMPLATES.find((x) => x.id === id)
    if (tp) placeBrief(templateBody(tp, i18n.locale))
  }
  // The gallery (§284, owner 14 ก.ย.: "ตอนเปิดหน้าสร้างเอเจนใหม่ให้แสดงเทมเพลต
  // ก่อน … แล้วให้กดกาได้ว่าฉันไม่ต้องการเทมเพลต"). A NEW agent's editor opens
  // on it in place of the form — the four shapes and the forty-two roles,
  // grouped — unless the person ticked the box, which is kept in this
  // browser and not in any file: it is a preference about this screen, not
  // about any agent. The เทมเพลต button under the role field opens the same
  // sheet later, for anyone, so the tick costs nothing but the first look.
  //
  // `first` is which way out the sheet offers: opened on a new agent, the
  // way out is "no template, start blank"; opened from the button over a
  // form that already has words, it is "back to the form". Same sheet, one
  // label that tells the truth about what leaving does.
  let agentGallery = $state(false)
  let agentGalleryFirst = $state(false)
  let agentGalleryQuery = $state('')
  let agentGallerySkip = $state(false)
  let agentGalleryBusy = $state('')
  function readGallerySkip(): boolean {
    try { return localStorage.getItem(GALLERY_SKIP_KEY) === '1' } catch { return false }
  }
  function setGallerySkip(on: boolean) {
    agentGallerySkip = on
    try { on ? localStorage.setItem(GALLERY_SKIP_KEY, '1') : localStorage.removeItem(GALLERY_SKIP_KEY) } catch { /* private window */ }
  }
  function openGallery(first: boolean) {
    agentGalleryFirst = first
    agentGalleryQuery = ''
    agentFillError = ''
    agentGallerySkip = readGallerySkip()
    agentGallery = true
  }
  // A shape lands the brief only (blanks to fill, no description to give);
  // a role lands the brief, the card's line as the description, and its id
  // as the name when there is none yet — three fields from one press, all
  // of them still the draft's to change before Save.
  function pickShape(id: string) {
    fillFromTemplate(id)
    agentGallery = false
  }
  async function pickRole(id: string) {
    const r = GALLERY_ROLES.find((x) => x.id === id)
    if (!r || agentGalleryBusy) return
    agentGalleryBusy = id
    try {
      const body = await galleryBody(id)
      if (!agentDraftName.trim()) agentDraftName = id
      if (!agentDraftDescription.trim()) agentDraftDescription = r.desc
      placeBrief(body)
      agentGallery = false
    } catch {
      agentFillError = t('settings.galleryLoadFail')
    } finally {
      agentGalleryBusy = ''
    }
  }
  // The role is the whole of what an agent is, and for a bundled one that is a
  // hundred lines of prose — opening the editor to change a model meant
  // scrolling past all of it to reach anything else on the page. So it is a
  // preview until asked for.
  //
  // Focus opens it, always: collapsed is a reading state, and the moment a
  // caret goes in, the person is writing and cannot write what they cannot see.
  // The threshold is lines rather than characters because what is being
  // measured is how far the rest of the page got pushed down.
  let agentBodyOpen = $state(false)
  const agentBodyLines = $derived(agentDraftPrompt.split('\n').length)
  const agentBodyLong = $derived(agentBodyLines > 16)
  let agentBusy = $state('')
  // The delegation switches, and the measured cost of having them on.
  //
  // Re-read after every flip rather than patched locally, because flipping one
  // re-bootstraps the engine and the block's size is the thing this page is
  // showing. A local edit would be a number that agreed with the switch and
  // disagreed with the request that gets sent.
  let delegate = $state<engine.DelegateSettings | null>(null)
  let delegateBusy = $state('')
  // The DEFAULT team's switches (§256): this page is the shipped roster's
  // settings, and a user team's switches live on its card in ทีมเอเจน.
  async function loadDelegate() {
    try {
      delegate = await DelegateSwitches('')
    } catch {
      delegate = null // the switches are simply absent rather than the page failing
    }
  }
  async function toggleDelegate(kind: 'agents' | 'helpers') {
    if (!delegate || delegateBusy) return
    delegateBusy = kind
    try {
      delegate = await SetDelegateOff(kind, delegate[kind].off === false)
    } finally {
      delegateBusy = ''
    }
  }
  // One worker looked up across both blocks, and its own kind decides which
  // switch grays it out. The row snippet is shared by the two pages and is
  // handed a worker rather than a page, so asking the row is the only way that
  // cannot disagree with where it was drawn.
  function reachOf(name: string): { on: boolean; off: boolean } | null {
    if (!delegate) return null
    for (const block of [delegate.agents, delegate.helpers]) {
      // An older engine answers a no-team chat with `workers: null`; a page
      // must not fall over on the shape of a build it did not ship with.
      const w = (block.workers ?? []).find((x) => x.name === name)
      if (w) return { on: w.on, off: block.off }
    }
    return null
  }
  async function toggleAgentReach(name: string, on: boolean) {
    if (delegateBusy) return
    delegateBusy = name
    try {
      delegate = await SetAgentOff('', name, on)
    } finally {
      delegateBusy = ''
    }
  }
  let agentError = $state('')
  // Read from the file, shown, and written back untouched. Not `Draft` because
  // nothing here edits them — see AgentFields for why they exist at all.
  let agentKeptDesk = $state('')
  let agentKeptNeeds = $state<string[]>([])

  // ---------- What one agent reaches and knows ----------
  //
  // Three panels below the tool picker, and three different kinds of fact, which
  // is why they are three boxes rather than one list (owner, 10 ส.ค.):
  //
  //   เครื่องมือ  subtracts from the shared set — allow/deny over what every
  //              agent starts with.
  //   MCP        adds, and only to this one. A server pointed at an agent
  //              skips the profile's allow-list entirely and reaches past the
  //              desk's ceiling (internal/subagent/store.go).
  //   สกิล       adds, and only to this one. Attached *after* the filter and
  //              outside it, deliberately (internal/subagent/skills.go).
  //
  // Reading them as one box was the complaint, and it was right: the page was
  // describing one mechanism where the engine has three.
  let agentSkills = $state<engine.AgentSkillInfo[]>([])
  let agentNeeds = $state<subagent.Requirement[]>([])
  const unmetAgentNeeds = $derived(agentNeeds.filter((r) => !r.met).length)
  // Its own memory (MEMORY.md in its folder) and its own opening (STARTERS.md),
  // both read-only here. Neither is edited on this page — memory is approved on
  // the Learning page and the opening is a file — but an agent whose page never
  // mentions them is a page that quietly claims they do not exist.
  let agentMemory = $state<string[]>([])
  let agentNewMemoryText = $state('')
  let agentAddingMemory = $state(false)
  let agentMemorySaving = $state(false)
  let agentMemoryError = $state('')
  // The delegate's memory as the head page draws one (deskHead + memRow):
  // one block, the face on it, the meter, the same rows. Its meter comes
  // from MemoryScopeInfo, because LearnedScopeInfos does not list delegates.
  let agentMemInfo = $state<{ bytes: number; maxBytes: number; full: boolean }>({ bytes: 0, maxBytes: 0, full: false })
  const agentGroup = $derived<MemoryGroup>({
    scope: agentDraftName.trim(), lines: agentMemory, orphan: false,
    bytes: agentMemInfo.bytes, maxBytes: agentMemInfo.maxBytes, full: agentMemInfo.full, projectsUnder: false,
    look: agentEditing ? lookOf(agentEditing) : undefined,
  })
  async function loadAgentMemInfo(name: string) {
    try {
      const info = await MemoryScopeInfo(name.trim())
      if (agentReachFor === name) agentMemInfo = { bytes: info.bytes ?? 0, maxBytes: info.maxBytes ?? 0, full: !!info.full }
    } catch { /* the meter is a nicety; the rows still draw */ }
  }

  async function addAgentMemory(name: string, text: string) {
    if (!text.trim() || !name.trim()) return
    agentMemorySaving = true
    agentMemoryError = ''
    try {
      await AddLearnedEntry(name.trim(), text.trim())
      agentNewMemoryText = ''
      agentAddingMemory = false
      agentMemory = await LearnedEntries(name.trim())
    } catch (err) {
      agentMemoryError = String(err)
    } finally {
      agentMemorySaving = false
    }
  }

  async function saveAgentMemoryItem(name: string, index: number, text: string) {
    agentMemorySaving = true
    agentMemoryError = ''
    try {
      await SaveLearnedEntry(name.trim(), index, text)
      cancelMemoryEdit()
      agentMemory = await LearnedEntries(name.trim())
      void loadAgentMemInfo(name)
    } catch (err) {
      agentMemoryError = String(err)
    } finally {
      agentMemorySaving = false
    }
  }
  let agentStarters = $state<subagent.StarterSet | null>(null)
  let agentReachFor = $state('') // whose panels are loaded, so a stale answer cannot land on the next agent
  // Which of the panels have their answer. Each list is empty for the tick
  // between opening the editor and the disk answering, and an empty list drawn
  // as "ยังไม่มีสกิลของตัวเอง" for that tick is a false sentence the reader
  // sees flash (owner, 13 ก.ย. 2026: "ตอนแรกมันมีแว๊บนึงไม่แสดง"). Until the
  // answer lands the box draws a skeleton row instead; and each answer lands
  // on its own, so the skills do not wait for the slowest read in the batch.
  let agentSkillsReady = $state(false)
  let agentMemoryReady = $state(false)
  let mcpLoaded = $state(false)

  // ---------- The opening, as a form ----------
  //
  // A growable list, floored at four and ceilinged at the pool the engine will
  // return (maxStarters in internal/subagent/starters.go).
  //
  // It was four fixed rows, for a reason that stopped being true: the window
  // drew every card a file held, so a fifth row could never appear on screen
  // and an Add button would have been a dead control. The window now deals four
  // out of a pool (starters.ts), so an agent the user hired can hold more than
  // it shows — and a form capped at four would be the one thing standing
  // between them and that. The floor stays because a pool below a full hand
  // deals a widow into the grid; the ceiling is the engine's, so the form
  // cannot accept a card that would be silently dropped on read.
  const STARTER_MIN = 4
  const STARTER_MAX = 24
  const blankStarter = () => ({ title: '', prompt: '', icon: '' })
  const blankStarters = () => Array.from({ length: STARTER_MIN }, blankStarter)
  // The questions, as a list: one row is the old single headline, more rows
  // are the file's several headings, one drawn at random per opening (owner,
  // 14 ก.ย. 2026: "อยากให้คำนี้เพิ่มได้หลายแบบหรือสุ่มได้"). Never fewer than one
  // row on screen; blank rows are dropped on save.
  let startersHeadlines = $state<string[]>([''])
  const startersHeadlinesClean = () => startersHeadlines.map((h) => h.trim()).filter(Boolean)
  const addHeadlineRow = () => { if (startersHeadlines.length < 12) startersHeadlines = [...startersHeadlines, ''] }
  const removeHeadlineRow = (i: number) => {
    startersHeadlines = startersHeadlines.length > 1 ? startersHeadlines.filter((_, k) => k !== i) : ['']
  }
  let startersCards = $state(blankStarters())
  let startersFile = $state('')
  let startersBusy = $state(false)
  let startersError = $state('')
  // What was on screen when it was last read or saved, so the Save button
  // answers "is there anything to write" rather than "is this form non-empty".
  let startersSnapshot = $state('')
  // True while the cards shown are the bundled agent's rather than the user's.
  // Saving turns that into a file of your own — the same "copy it out to change
  // it" shape as AGENT.md, said before the click instead of after.
  let startersInherited = $state(false)

  const startersKey = () => JSON.stringify([startersHeadlinesClean(), startersCards])
  const startersDirty = $derived(startersKey() !== startersSnapshot)
  const startersEmpty = $derived(
    startersHeadlinesClean().length === 0 && startersCards.every((c) => !c.title.trim() && !c.prompt.trim()),
  )
  // The filename, asked of the engine rather than assembled here: which of
  // STARTERS.md / STARTERS.<lang>.md is written follows the same rule the
  // reader resolves by, and two places spelling it is two places to get it
  // wrong (config.AgentStartersName).
  const agentStartersFile = $derived(startersFile)

  function fillStarters(set: subagent.StarterSet | null) {
    const heads = (set?.headlines ?? []).filter(Boolean)
    startersHeadlines = heads.length ? heads : [set?.headline ?? '']
    // The trailing space after a colon is put back by the reader and must not
    // come back through the form as an edit nobody made.
    const cards = (set?.cards ?? []).slice(0, STARTER_MAX).map((c) => ({
      title: c.title,
      prompt: (c.prompt ?? '').trimEnd(),
      icon: c.icon ?? '',
    }))
    // Always at least the floor, so an agent with two cards still shows a form
    // that can be filled to a usable pool rather than a form that ends early.
    while (cards.length < STARTER_MIN) cards.push(blankStarter())
    startersCards = cards
    startersSnapshot = startersKey()
    startersError = ''
  }

  const canAddStarter = $derived(startersCards.length < STARTER_MAX)
  function addStarterRow() {
    if (canAddStarter) startersCards = [...startersCards, blankStarter()]
  }
  // Removing below the floor is not offered rather than refused: a button that
  // is there and says no is worse than a button that is not there.
  function removeStarterRow(i: number) {
    if (startersCards.length > STARTER_MIN) {
      startersCards = startersCards.filter((_, k) => k !== i)
    } else {
      startersCards[i] = blankStarter()
    }
  }

  // Whose opening the form is editing: a worker's (STARTERS.md in its home)
  // or a main desk's (modes/<desk>/STARTERS.md, §266). One form, one set of
  // state, two files — the same rows in both places, never two forms.
  type StartersOwner = { kind: 'chair'; name: string } | { kind: 'desk'; head: HeadId }
  let startersOwner = $state<StartersOwner>({ kind: 'chair', name: '' })
  const readStarters = (o: StartersOwner) =>
    o.kind === 'desk' ? DeskStarters(o.head, i18n.locale) : ChairStarters(o.name, i18n.locale)
  const writeStarters = (o: StartersOwner, set: subagent.StarterSet) =>
    o.kind === 'desk' ? SaveDeskStarters(o.head, i18n.locale, set) : SaveChairStarters(o.name, i18n.locale, set)
  async function loadHeadStarters(h: HeadId) {
    const [set, file] = await Promise.all([DeskStarters(h, i18n.locale), ChairStartersFile(i18n.locale)])
    if (mainHead !== h) return
    startersFile = 'modes/' + h + '/' + file
    startersInherited = false
    fillStarters(set)
  }

  const saveStarters = () => runStarters(async () => {
    await writeStarters(startersOwner, subagent.StarterSet.createFrom({
      headline: startersHeadlinesClean()[0] ?? '',
      headlines: startersHeadlinesClean(),
      cards: startersCards
        .filter((c) => c.title.trim() && c.prompt.trim())
        .map((c) => ({ title: c.title.trim(), prompt: c.prompt.trim(), icon: c.icon.trim() })),
    }))
    // Read back rather than assumed: the engine caps, trims and refuses, and
    // the form must end up showing what is now in the file.
    fillStarters(await readStarters(startersOwner))
    startersInherited = false
  })

  // Clearing is "I do not want my own opening" — the file goes, and whatever
  // was underneath it answers again: the shipped cards, or the ordinary four.
  const clearStarters = () => runStarters(async () => {
    await writeStarters(startersOwner, subagent.StarterSet.createFrom({ headline: '', cards: [] }))
    const back = await readStarters(startersOwner)
    fillStarters(back)
    startersInherited = (back.cards ?? []).length > 0 || !!back.headline
  })

  async function runStarters(fn: () => Promise<void>) {
    startersBusy = true
    startersError = ''
    try {
      await fn()
    } catch (e) {
      startersError = String(e)
    } finally {
      startersBusy = false
    }
  }

  // The delegate's own memory file and its meter, on their own because a
  // ลูกมือ has the ความจำ tab too (fddd0b15) and none of the reach: with the
  // load inside loadAgentReach, its tab opened on a page that never asked
  // for the file (owner, 14 ก.ย. 2026: "ทำไมโล่งแบบนี้").
  function loadAgentMemory(name: string): Promise<void> {
    agentReachFor = name
    agentMemory = []
    agentMemoryReady = false
    void loadAgentMemInfo(name)
    return LearnedEntries(name).then((memory) => {
      if (agentReachFor !== name) return
      agentMemory = memory
      agentMemoryReady = true
    })
  }

  async function loadAgentReach(name: string) {
    const memoryP = loadAgentMemory(name)
    agentSkills = []
    agentNeeds = []
    agentStarters = null
    agentSkillsReady = false
    // The MCP register is the source for "which servers is this agent on", and
    // it is already loaded for its own page. Asked for here too, because the
    // editor can be the first page opened in a session.
    const skillsP = AgentSkills(name).then((skills) => {
      if (agentReachFor !== name) return // the user moved on while the disk was being read
      agentSkills = skills
      agentSkillsReady = true
    })
    startersOwner = { kind: 'chair', name }
    const [needs, starters, file] = await Promise.all([
      AgentNeeds(name),
      ChairStarters(name, i18n.locale),
      ChairStartersFile(i18n.locale),
      skillsP,
      memoryP,
      mcpLoaded ? Promise.resolve() : loadMCP(),
    ])
    if (agentReachFor !== name) return
    agentNeeds = needs
    agentStarters = starters
    startersFile = file
    // Whether these cards are already this agent's own is not a question the
    // reader can answer — a bundled agent's opening looks identical to one you
    // wrote. Only a profile with a file of its own can be showing its own.
    startersInherited = ((starters.cards ?? []).length > 0 || !!starters.headline) && agentEditing?.path === ''
    fillStarters(starters)
  }

  // Which servers this agent carries, read off the register rather than stored
  // twice: `for:` on the server is the only thing that grants (needs.go), so a
  // second list here would be a second answer to a question that has one.
  //
  // The owner id comes from PlacementTargets, never from pasting "agent:" in
  // front of a name. config.MCPAgentPrefix carries a warning about exactly that
  // — three places say the prefix, and one of them spelling it by hand is a
  // switch that silently stops matching. Empty for an agent with no file yet,
  // which is what the panel's "save first" state is for.
  const agentMCPId = $derived(
    mcpTargets.find((x) => x.kind === 'agent' && x.name === agentDraftName.trim())?.id ?? '',
  )
  // Every enabled server, not only the ones already ticked: the panel answers
  // "what does this one carry" and "what could it" in one read, and a list of
  // just the ticked ones is a list you cannot add to.
  const agentServers = $derived(
    agentMCPId ? mcpServers.filter((s) => !s.disabled).filter((s) => (s.for ?? []).includes(agentMCPId)) : [],
  )
  const agentServerCount = $derived(agentServers.length)
  // A NEW agent has no placement id until its file exists, so its MCP tab
  // cannot write `for:` as it goes. It ticks instead (owner, 13 ก.ย. 2026:
  // "ตอนสร้างทำให้เลือกได้ว่าจะเลือก MCP ไหนที่มีในระบบ"), and Save places the
  // ticked servers right after the file is written — through the one writer
  // the room uses (SetMCPServerTargets), so it is a delay, not a second store.
  let agentNewMCP = $state<string[]>([])
  const liveServers = $derived(mcpServers.filter((s) => !s.disabled))
  function toggleNewMCP(name: string) {
    agentNewMCP = agentNewMCP.includes(name) ? agentNewMCP.filter((n) => n !== name) : [...agentNewMCP, name]
  }
  // A SAVED agent's switch writes at once, through the room's one writer —
  // the whole `for:` list of that server sent back with this agent added or
  // removed. Since 13 ก.ย. 2026 this box edits again (owner: "ทำไมมันไม่แสดง
  // MCP" of a box that listed only what was already placed): the room's
  // picker and this list are the same switch on the same call, not two
  // stores, and the page a person is standing on when they ask "what can
  // this one use" is this one.
  const isOnAgent = (srv: MCPRow) => !!agentMCPId && (srv.for ?? []).includes(agentMCPId)
  const toggleAgentMCP = (srv: MCPRow) => runMCP('target:' + srv.name, async () => {
    if (!agentMCPId) return
    const cur = srv.for ?? []
    await SetMCPServerTargets(srv.name, isOnAgent(srv) ? cur.filter((x) => x !== agentMCPId) : [...cur, agentMCPId])
    await loadMCP()
    if (agentReachFor) agentNeeds = await AgentNeeds(agentReachFor)
  })
  // The same shape for skills on a NEW agent: its folder does not exist until
  // Save, so the shelf is ticked here and Save copies each tick in
  // (CopySkillToAgent — the room's sheet's own call). A saved agent's skills
  // are edited on the room's sheet, which lists the whole shelf against what
  // the folder holds; this box lists the folder.
  type ShelfSkill = { name: string; description: string; bundled?: boolean }
  let shelfSkills = $state<ShelfSkill[]>([])
  let shelfLoaded = $state(false)
  let agentNewSkills = $state<string[]>([])
  let shelfQuery = $state('')
  const shelfShown = $derived.by(() => {
    const q = shelfQuery.trim().toLowerCase()
    return q ? shelfSkills.filter((x) => x.name.toLowerCase().includes(q) || x.description.toLowerCase().includes(q)) : shelfSkills
  })
  async function loadShelf() {
    try {
      shelfSkills = ((await ListExternalSkills()) ?? []) as ShelfSkill[]
    } catch {
      shelfSkills = []
    }
    shelfLoaded = true
  }
  function toggleNewSkill(name: string) {
    agentNewSkills = agentNewSkills.includes(name) ? agentNewSkills.filter((n) => n !== name) : [...agentNewSkills, name]
  }

  // Toggling here writes the same `for:` list the MCP page writes, through the
  // same call. It applies at once and does not wait for Save — the panel says
  // so, because a switch inside a form with a Save button is otherwise read as
  // part of the draft.

  // The model dropdown offers the models of the provider the draft names, or
  // the chat's when it names none — a pin to a model the list does not hold
  // still shows (as its own option) rather than silently reading as "inherit".
  let agentModels = $state<string[]>([])
  // Asked per provider, on the provider changing: a list fetched once for the
  // chat's provider would offer OpenAI's models under a DeepSeek pick.
  async function loadAgentModels(provider: string) {
    try {
      agentModels = await ListModelsForProvider(provider || cockpit.model.provider)
    } catch {
      agentModels = [] // no key / offline: the dropdown still offers "inherit"
    }
  }
  const pickAgentProvider = (provider: string) => {
    agentDraftProvider = provider
    // A model pinned under one provider is not a model at another.
    agentDraftModel = ''
    void loadAgentModels(provider)
  }
  // The thinking levels the draft's provider/model has — asked of the engine
  // rather than typed here, because the ladder is per model (Codex's sol has
  // ultra, luna stops at max; a local runtime has no dial at all) and the
  // engine already owns that table for the chat's own picker. Empty means no
  // dial: the control then says so instead of offering levels the request
  // would silently drop. Re-asked whenever provider or model changes.
  let agentThinkLevels = $state<string[]>([])
  $effect(() => {
    if (!agentEditing) return
    const provider = agentDraftProvider, model = agentDraftModel
    SupportedThinkLevelsFor(provider, model)
      .then((levels) => { agentThinkLevels = levels })
      .catch(() => { agentThinkLevels = [] })
  })

  async function loadAgents() {
    subagents = await ListSubagentProfiles()
    // Asked, not worked out from the profile's `desk`. Which profiles sit in
    // the office is decided in one place (ListChairs), and a second reading of
    // the same rule here is a second answer waiting to disagree with the page
    // that actually draws the roster.
    try {
      const roster = await ListChairs()
      chairNames = new Set(roster.map((c) => c.name))
    } catch {
      chairNames = new Set() // engine not up: rows just carry no room label
    }
    await loadAgentModels('')
  }

  async function runAgent(label: string, fn: () => Promise<void>) {
    agentBusy = label
    agentError = ''
    try {
      await fn()
    } catch (err) {
      agentError = String(err)
    } finally {
      agentBusy = ''
    }
  }

  // The dropdown on a row: '' means inherit whatever model the chat is on.
  const pinModel = (name: string, model: string) => runAgent('model:' + name, async () => {
    await SetSubagentModel(name, model)
    await loadAgents()
  })

  // The agent file is as long as the role you wrote, so a fixed box means
  // scrolling a small window inside a page that has room to spare. This grows
  // the field to its content instead; `min-height` in the CSS is still the
  // floor, so a short file looks exactly as it did before.
  //
  // Takes the text as its parameter rather than listening to `input` alone:
  // switching to another sub-agent replaces the value without any keystroke,
  // and a field left at the previous file's height is the bug this fixes.
  function autogrow(node: HTMLTextAreaElement, _value: string) {
    const fit = () => {
      // Collapse first: scrollHeight of an already-tall box reports the box,
      // not the text, so without this the field can only ever grow.
      node.style.height = 'auto'
      node.style.height = node.scrollHeight + 'px'
    }
    fit()
    node.addEventListener('input', fit)
    return {
      update: () => fit(),
      destroy: () => node.removeEventListener('input', fit),
    }
  }

  // Editing opens the raw .md — including for a bundled profile, where saving
  // writes your own copy over it (the engine already prefers user files).
  // Everything the editor can change, in one string. Compared against the value
  // captured when the editor opened (and re-captured on save) to answer the one
  // question the Back button needs answered.
  const agentDraftKey = () => JSON.stringify([
    agentDraftName, agentDraftDescription, agentDraftModel, agentDraftProvider, agentDraftThink,
    agentDraftTools, agentDraftDeny, agentDraftSteps, agentDraftPrompt,
    agentDraftIcon, agentDraftShell, agentDraftTop, agentDraftFace, agentDraftAccent, agentDraftHue,
  ])
  let agentSnapshot = ''

  // Which kind the editor is holding — เอเจน or ซับเอเจน. Set by the door
  // the editor was opened through and never by reading the file: the same rule
  // the storage layer lives by (a file's home is its kind), carried up. Saving
  // goes out through the matching door, so an edit cannot change what
  // something is as a side effect of where a button happened to be.
  let agentEditKind = $state<'agent' | 'helper'>('helper')
  // Where an AGENT's editor closes onto. Its own list (ตั้งค่า › เอเจนเฉพาะทาง),
  // normally; the roster page when a card's lock sent the person here to
  // finish setting someone up (AgentLock's intent), since that is the page
  // they were on; the ทีมเอเจน page when its form's "+ เอเจน" opened the
  // editor, so the person lands back on the draft they left, with the new
  // agent ticked.
  let agentBack = $state<'list' | 'office' | 'teams'>('list')
  type AgentTab = 'identity' | 'avatar' | 'brain' | 'reach' | 'knowledge' | 'opening' | 'memory'
  let agentTab = $state<AgentTab>('identity')

  const openAgent = (a: SubagentRow, kind?: 'agent' | 'helper') => runAgent('open:' + a.name, async () => {
    void loadLearning()
    const parsed = parseAgentFile(await ReadSubagentProfile(a.name))
    agentDraftName = a.name
    agentDraftDescription = parsed.description
    agentDraftModel = parsed.model
    agentDraftProvider = parsed.provider
    agentDraftThink = parsed.think
    if (parsed.provider) void loadAgentModels(parsed.provider)
    agentDraftTools = parsed.tools
    agentDraftDeny = parsed.deny
    // No `steps:` line means no ceiling (§110), so the box opens ticked. Seeded
    // rather than inferred from a blank field, so clearing the number to type a
    // new one does not disable the field under the user's cursor.
    agentDraftSteps = parsed.steps.trim() || STEPS_UNLIMITED
    agentDraftIcon = parsed.icon
    agentDraftShell = parsed.shell
    agentDraftTop = parsed.top
    agentDraftFace = parsed.face
    agentDraftAccent = parsed.accent
    agentDraftHue = parsed.hue
    agentKeptDesk = parsed.desk
    agentKeptNeeds = parsed.needs
    agentDraftPrompt = parsed.body
    // Every open starts collapsed, including the second open of the same agent:
    // the state belongs to this reading of the page, not to the file.
    agentBodyOpen = false
    agentTab = 'identity'
    agentGallery = false
    agentEditing = a
    // A row opened from this page answers from the roster the page already
    // asked for (ListChairs) — never from the file's own fields.
    agentEditKind = kind ?? (chairNames.has(a.name) ? 'agent' : 'helper')
    agentSnapshot = agentDraftKey()
    // The three panels that describe an agent's reach and knowledge. Fetched
    // after the fields are in, and not awaited by the editor: a slow disk scan
    // must not hold up the form the user came to type in.
    if (agentEditKind === 'agent') void loadAgentReach(a.name)
    else void loadAgentMemory(a.name)
  })

  function newAgent(kind: 'agent' | 'helper' = 'helper') {
    agentEditing = { name: '', description: '', prompt: '', builtin: false }
    agentDraftName = ''
    agentDraftDescription = ''
    agentDraftModel = ''
    agentDraftProvider = ''
    agentDraftThink = ''
    agentDraftTools = []
    agentDraftDeny = []
    agentDraftSteps = STEPS_UNLIMITED // a new worker starts uncapped, like every shipped one
    agentDraftIcon = ''
    agentDraftShell = ''
    agentDraftTop = ''
    agentDraftFace = ''
    agentDraftAccent = ''
    agentDraftHue = ''
    agentKeptDesk = ''
    agentKeptNeeds = []
    // Empty, with the guidance as a PLACEHOLDER. Until 13 ก.ย. the guidance was
    // the field's VALUE, and the Save button accepted it — so an agent made in
    // a hurry shipped with "มันไม่เห็นประวัติแชท คืนแค่ผลลัพธ์" as its whole
    // brief, and then behaved exactly like that in a chat someone had walked
    // into to talk (the report that started §256). A placeholder cannot be
    // saved; the guard on Save (an empty role) does the rest.
    agentDraftPrompt = ''
    agentBodyOpen = true // a new agent is opened to be written in, not read
    agentTab = 'identity'
    agentError = ''
    agentEditKind = kind
    agentSkills = []
    agentNeeds = []
    agentNewMCP = []
    agentNewSkills = []
    shelfQuery = ''
    if (!mcpLoaded) void loadMCP() // the tick lists need the register and the shelf
    if (!shelfLoaded) void loadShelf()
    agentSnapshot = agentDraftKey()
    // The sheet before the form (§284) — for an AGENT only. A helper is a
    // delegate the assistant hands work to, and none of the shelf's roles
    // are written for that seat.
    if (kind === 'agent' && !readGallerySkip()) openGallery(true)
    else agentGallery = false
  }

  // Leaving the editor. A ซับเอเจน's editor closes onto its own list; an
  // AGENT's closes onto wherever it was opened from (agentBack), and the
  // door resets to the list so the next open from this page stays here.
  function leaveAgentEditor(saved = '') {
    agentEditing = null
    if (agentEditKind !== 'agent') return
    const back = agentBack
    agentBack = 'list'
    if (back === 'teams') {
      cockpit.settingsIntent = { section: 'teams', agent: saved || undefined }
      openSection('teams')
    } else if (back === 'office') {
      setShell('assistant')
      setActiveView('office')
    }
  }
  // ทีมเอเจน's form asked for an agent that is not on the roster yet.
  function newAgentFromTeams() {
    agentBack = 'teams'
    openSection('team')
    newAgent('agent')
  }
  const closeAgentEditor = () =>
    guardUnsaved(agentDraftKey() !== agentSnapshot, leaveAgentEditor)

  const saveAgent = () => runAgent('save', async () => {
    const body = serializeAgentFile({
      description: agentDraftDescription,
      model: agentDraftModel,
      provider: agentDraftProvider,
      think: agentDraftThink,
      tools: agentDraftTools,
      deny: agentDraftDeny,
      steps: agentDraftSteps,
      icon: agentDraftIcon,
      shell: agentDraftShell,
      top: agentDraftTop,
      face: agentDraftFace,
      accent: agentDraftAccent,
      hue: agentDraftHue,
      desk: agentKeptDesk,
      needs: agentKeptNeeds,
      body: agentDraftPrompt,
    })
    // Out through the door that matches the kind — the backend refuses a name
    // the other kind owns, so the wrong door is an error message, never a file
    // in the wrong home.
    if (agentEditKind === 'agent') await SaveAgentProfile(agentDraftName.trim(), body)
    else await SaveSubagentProfile(agentDraftName.trim(), body)
    await placeNewAgentMCP(agentDraftName.trim())
    await copyNewAgentSkills(agentDraftName.trim())
    await loadAgents()
    leaveAgentEditor(agentDraftName.trim())
  })
  // The new agent's ticked skills, copied into the folder Save just made.
  async function copyNewAgentSkills(name: string) {
    if (agentEditKind !== 'agent' || agentNewSkills.length === 0) return
    const picks = agentNewSkills
    agentNewSkills = []
    for (const sk of picks) await CopySkillToAgent(name, sk)
  }
  // The new agent's ticked servers, placed once its file — and so its
  // placement id — exists. Each server's whole `for:` list is sent back with
  // this agent added, which is what the engine stores per server.
  async function placeNewAgentMCP(name: string) {
    if (agentEditKind !== 'agent' || agentNewMCP.length === 0) return
    const picks = agentNewMCP
    agentNewMCP = []
    await loadMCP()
    const id = mcpTargets.find((x) => x.kind === 'agent' && x.name === name)?.id
    if (!id) return
    for (const srv of picks) {
      const row = mcpServers.find((x) => x.name === srv && !x.disabled)
      if (row && !(row.for ?? []).includes(id)) await SetMCPServerTargets(srv, [...(row.for ?? []), id])
    }
    await loadMCP()
  }

  // Two different actions behind one button: deleting a profile the user wrote,
  // versus dropping an override so a built-in goes back to how it shipped. They
  // lose different things, so they say different things.
  const deleteAgent = () => {
    const reverting = agentEditing?.overrides === true
    askConfirm({
      title: reverting ? t('settings.confirmAgentRevertTitle') : t('settings.confirmAgentTitle'),
      message: reverting ? t('settings.confirmAgentRevertMessage') : t('settings.confirmAgentMessage'),
      detail: agentEditing?.path || agentDraftName.trim(),
      confirmLabel: reverting ? t('settings.confirmAgentRevertAction') : t('settings.confirmDeleteAction'),
      run: () => runAgent('delete', async () => {
        await DeleteSubagentProfile(agentDraftName.trim())
        await loadAgents()
        leaveAgentEditor()
      }),
    })
  }

  // There is no tool badge on this row any more. It said "all tools" or a
  // count, and after 31 ส.ค. every agent holds the same kit — so it was one
  // word repeated down a column, saying nothing about the agent it sat on.
  // What is left is the deny badge below, which appears only when this agent
  // refuses something, because only a refusal tells you what it will not do.

  const denyTip = (a: SubagentRow) => t('settings.agentDenyTip', { list: (a.deny ?? []).join(', ') })

  // What you may put in `tools:`/`deny:`. The editor is a raw .md field, so the
  // question it leaves you with is "what are the names?" — asking the running
  // registry beats a list written down here that drifts the day a tool is added.
  //
  // AGENT_FORCED_DENIALS mirrors subagent.forcedDenials: names a sub-agent never
  // gets no matter what the file says. Listing them as available would be a lie
  // the user only discovers after saving.
  const AGENT_FORCED_DENIALS = ['task', 'task_result', 'task_answer', 'task_plan', 'help', 'ask_user', 'todo_write']
  // Mirrors subagent.stepsUnlimitedKeyword. The frontmatter carries a word
  // rather than a sentinel number because the file is hand-editable.
  const STEPS_UNLIMITED = 'unlimited'

  // `desk` and `needs` are here to be *kept*, not to be edited. Neither had a
  // field, and neither survived a save: opening github or automation and
  // pressing Save wrote a shadow with `needs:` gone, so the agent quietly
  // stopped declaring what it cannot work without and the notice it carries in
  // its own prompt (subagent.PromptFor) went with it. `desk:` was the same
  // silent loss with a worse ending — an agent on a named desk fell back to the
  // office ceiling. An editor must not delete what it does not draw.
  type AgentFields = {
    description: string; model: string; provider: string; think: string; tools: string[]; deny: string[]; steps: string; icon: string
    shell: string; top: string; face: string; accent: string; hue: string; desk: string; needs: string[]; body: string
  }

  // Mirrors internal/subagent/profile.go's parse(): a leading `---`-fenced block
  // of `key: value` lines, then the role prompt underneath. Duplicated here
  // rather than asked of the backend because this is purely a display choice —
  // the file format itself has not changed, so there's nothing to add to the
  // Go side for it. Falls back to treating the whole thing as the prompt when
  // there's no recognizable frontmatter, so a hand-edited or malformed file is
  // never silently emptied under the user.
  function parseAgentFile(raw: string): AgentFields {
    const asPromptOnly = {
      description: '', model: '', provider: '', think: '', tools: [] as string[], deny: [] as string[],
      steps: '', icon: '', shell: '', top: '', face: '', accent: '', hue: '', desk: '',
      needs: [] as string[], body: raw.trim(),
    }
    const normalized = raw.replace(/\r\n/g, '\n').replace(/^\n+/, '')
    if (!normalized.startsWith('---\n')) return asPromptOnly
    const rest = normalized.slice(4)
    const end = rest.indexOf('\n---')
    if (end < 0) return asPromptOnly
    const fields: Record<string, string> = {}
    for (const line of rest.slice(0, end).split('\n')) {
      const t = line.trim()
      const i = t.indexOf(':')
      if (i < 0) continue
      const key = t.slice(0, i).trim().toLowerCase()
      if (key) fields[key] = t.slice(i + 1).trim().replace(/^["']+|["']+$/g, '')
    }
    const list = (v?: string) => (v ?? '').split(',').map((s) => s.trim().toLowerCase()).filter(Boolean)
    return {
      description: fields.description ?? '',
      model: fields.model ?? '',
      // A provider name is an id from the catalog, lowercase already.
      provider: (fields.provider ?? '').trim().toLowerCase(),
      // A level name (low, high, ultra…): lowercase, as the engine reads it.
      think: (fields.think ?? '').trim().toLowerCase(),
      tools: list(fields.tools),
      deny: list(fields.deny),
      steps: (fields.steps ?? '').trim(),
      icon: (fields.icon ?? '').trim(),
      // Not lowercased: these name a row in a catalogue by its id, and some
      // ids are camelCase ('laptopTerm'). Lowercasing here would turn a look
      // somebody chose into the default one, silently, on the way in.
      // `hair:` and `accessory:` are not read: they were the cartoon face's
      // (gone 12 ก.ย. 2026), nothing draws them, and a line the editor does
      // not draw is dropped on the next save — that is the migration.
      shell: (fields.shell ?? '').trim(),
      top: (fields.top ?? '').trim(),
      face: (fields.face ?? '').trim(),
      accent: (fields.accent ?? '').trim(),
      hue: (fields.hue ?? '').trim(),
      desk: (fields.desk ?? '').trim().toLowerCase(),
      // Not lowercased and not split on anything but the comma: an entry may
      // carry alternatives ("connection:n8n | connection:windmill"), which the
      // engine splits on `|` itself (subagent.alternatives). Touching the
      // inside of an entry here would be this editor deciding something the
      // author wrote down.
      needs: (fields.needs ?? '').split(',').map((s) => s.trim()).filter(Boolean),
      body: rest.slice(end + 4).trim(),
    }
  }

  // The inverse of parseAgentFile. What SaveSubagentProfile receives here is
  // exactly what ReadSubagentProfile would hand back for it afterwards — the
  // backend never has to know the editor stopped showing it the raw text.
  function serializeAgentFile(f: AgentFields): string {
    const lines = ['---', `description: ${f.description.trim()}`]
    if (f.model.trim()) lines.push(`model: ${f.model.trim()}`)
    if (f.provider.trim()) lines.push(`provider: ${f.provider.trim()}`)
    if (f.think.trim()) lines.push(`think: ${f.think.trim()}`)
    if (f.tools.length) lines.push(`tools: ${f.tools.join(', ')}`)
    if (f.deny.length) lines.push(`deny: ${f.deny.join(', ')}`)
    // Written back exactly as they were read. The editor shows both and edits
    // neither: what an agent requires and which desk it sits at are the
    // author's statements about the job, and a form that cannot express them
    // must at least not swallow them.
    if (f.desk) lines.push(`desk: ${f.desk}`)
    if (f.needs.length) lines.push(`needs: ${f.needs.join(', ')}`)
    // The mark this agent wears on the roster. Written only when the user chose
    // one: an absent field means the roster derives it from what the agent
    // makes, which is the right answer for every profile nobody has opened.
    if (f.icon.trim()) lines.push(`icon: ${f.icon.trim()}`)
    // The same rule for the rest of the look: written only when chosen, absent
    // when derived. A file full of lines restating the default is a file whose
    // defaults can never change again.
    if (f.shell.trim()) lines.push(`shell: ${f.shell.trim()}`)
    if (f.top.trim()) lines.push(`top: ${f.top.trim()}`)
    if (f.face.trim()) lines.push(`face: ${f.face.trim()}`)
    if (f.accent.trim()) lines.push(`accent: ${f.accent.trim()}`)
    if (f.hue.trim()) lines.push(`hue: ${f.hue.trim()}`)
    // The keyword, not a number. Leaving the line out would mean the same thing
    // today (the default is no ceiling since §110), but writing it says so in
    // the file, where the next person to read it is looking.
    if (f.steps.trim().toLowerCase() === STEPS_UNLIMITED) {
      lines.push(`steps: ${STEPS_UNLIMITED}`)
    } else {
      const steps = parseInt(f.steps, 10)
      if (Number.isFinite(steps) && steps > 0) lines.push(`steps: ${steps}`)
    }
    lines.push('---', '', f.body.trim())
    return lines.join('\n')
  }

  // ---------- Sub-agent tool permissions ----------
  // There is no control here any more, and the engine's rule is the reason.
  // It resolves one question per tool (internal/subagent/profile.go
  // AllowsTool): a forced denial always wins, then `deny:`, then an empty
  // allow-list means everything, then membership. On 31 ส.ค. every bundled
  // agent's allow-list went away — an agent's kit is its desk's now — so the
  // first half of that rule had nothing left to say on any screen, and `deny:`
  // alone did not earn a seventy-row picker to reach it.
  //
  // `agentDraftDeny` and `agentDraftTools` survive as passthrough: parsed in,
  // written back out unchanged, so an editor that no longer offers to write
  // either line cannot silently eat one somebody wrote by hand.

  // ---------- Step limit ----------
  const agentStepsUnlimited = $derived(agentDraftSteps.trim().toLowerCase() === STEPS_UNLIMITED)

  // Ticking remembers nothing, unticking leaves an empty box for a number: the
  // field is disabled while unlimited is on, so whatever was in it is not
  // something the user can see or was looking at. An empty box saves no
  // `steps:` line at all, which is the same "no ceiling" by a quieter route.
  function toggleStepsUnlimited() {
    agentDraftSteps = agentStepsUnlimited ? '' : STEPS_UNLIMITED
  }

  $effect(() => {
    if (active === 'agents' || active === 'team') void loadAgents()
    // Both pages, since each now carries its own switch: 'team' is เอเจน and
    // 'agents' is ซับเอเจน (see the render below). Loading on one page only is
    // how the ซับเอเจน page ended up with rows it could not explain.
    if (active === 'team' || active === 'agents') void loadDelegate()
  })

  // ---------- เกี่ยวกับคุณ ----------
  // The person's name and USER.md. context.md sat here for a morning
  // (14 ก.ย. 2026) and went back to the identity set on ตัวหลัก › ตัวตน the
  // same day, on the owner's word: it is the head's standing brief, not the
  // person's data.
  let youName = $state('')
  $effect(() => {
    if (active === 'you') {
      void loadLearning()
      void loadProfileName().then(() => { youName = profile.name })
    }
  })

  // ---------- ตัวหลัก ----------
  // Two heads, one page each, the agent editor's shape. What a head owns
  // today: its face (avatarPrefs), its desk file (modes/<desk>.md, read here
  // for the card's line), and its memory — MAIN_SCOPE for the assistant,
  // mode:coding for the coder, with the project files under whichever desk
  // hosts them. The rest of the tabs are doors to where each thing is still
  // edited (avatar, models, the capability room) until those move in.
  let mainHead = $state<HeadId | null>(null)
  // No avatar tab (owner, 14 ก.ย.: "จะไม่มีอวตาร เพราะมันมีอยู่แล้ว") — the
  // avatar page is the one place a look is chosen, and a tab that only
  // doored there was a tab. No brain tab either (owner, 14 ก.ย.: "เอาหน้าสมอง
  // ออก เพราะมันอิงกับตอนผู้ใช้เลือกอยู่แล้ว") — the model is the chat header's
  // pick, one for both desks, and a tab that only restated it was a tab.
  // The agent editor's own tabs, one per thing a head owns (owner, 14 ก.ย.:
  // "ทำให้มันเหมือนหน้าตั้งค่าเอเจน … สกิลไปไหน เปิดบทสนทนาไปไหน"): MCP with the
  // switches on the page, the shelf it sees, its opening, its memory.
  type MainTab = 'identity' | 'mcp' | 'skills' | 'opening' | 'memory'
  let mainTab = $state<MainTab>('identity')
  // The desk's own file (modes/<head>.md), whole — frontmatter and direction
  // — edited here since 14 ก.ย. 2026 (owner: "เอา modes/coding.md มาแสดงให้
  // คนปรับแต่งได้ … ปุ่มคืนค่าเริ่มต้นได้เสมอ ก่อนคืนค่าให้ถามยืนยัน"). A save is
  // the user's copy shadowing the bundled file, the shadowing a hand-written
  // modes/<name>.md always had; คืนค่าเริ่มต้น removes that copy. Read when
  // a session starts, so an edit reaches the next chat, and the page says so.
  let deskFile = $state<engine.DeskFile | null>(null)
  let deskDraft = $state('')
  let deskOpen = $state(false)
  let deskBusy = $state(false)
  let deskError = $state('')
  let deskMsg = $state('')
  const deskDirty = $derived(deskFile !== null && deskDraft !== deskFile.text)
  async function loadDeskFile(h: HeadId) {
    try {
      deskError = ''
      deskFile = await ReadDeskFile(h)
      deskDraft = deskFile.text
    } catch (err) {
      deskFile = null
      deskError = String(err)
    }
  }
  async function saveDeskFile(h: HeadId) {
    deskBusy = true
    try {
      deskError = ''
      await SaveDeskFile(h, deskDraft)
      await loadDeskFile(h)
      await loadModes()
      deskMsg = t('settings.mainDeskApplies')
    } catch (err) {
      deskError = String(err)
    } finally {
      deskBusy = false
    }
  }
  const askResetDeskFile = (h: HeadId) => askConfirm({
    title: t('settings.mainDeskResetTitle'),
    message: t('settings.mainDeskResetMessage', { name: headLabel(h) }),
    detail: `modes/${h}.md`,
    confirmLabel: t('settings.mainDeskReset'),
    run: () => void resetDeskFile(h),
  })
  async function resetDeskFile(h: HeadId) {
    deskBusy = true
    try {
      deskError = ''
      await ResetDeskFile(h)
      await loadDeskFile(h)
      await loadModes()
      deskMsg = t('settings.mainDeskApplies')
    } catch (err) {
      deskError = String(err)
    } finally {
      deskBusy = false
    }
  }
  let modes = $state<mode.Mode[]>([])
  async function loadModes() {
    try { modes = (await ListModes()) ?? [] } catch { modes = [] }
  }
  const headScope = (h: HeadId) => (h === 'assistant' ? MAIN_SCOPE : 'mode:coding')
  const headLabel = (h: HeadId) => { const k = deskLabelKey(h); return k ? t(k) : h }
  // What the head calls itself (config.ModelPreference.HeadNames), "" for
  // Aetox — its own field on ตัวตน (owner, 14 ก.ย. 2026: "ชื่อควรจะเป็นชื่อที่
  // เปลี่ยนได้"). Written on change like the person's name, and it reaches the
  // open chat's prompt at once (Engine.SetHeadName). The card and the page
  // title wear it; the desk's word stays as the badge.
  let headNames = $state<Record<HeadId, string>>({ assistant: '', coding: '' })
  let headNameDraft = $state('')
  const headShown = (h: HeadId) => headNames[h] || headLabel(h)
  async function loadHeadNames() {
    try {
      const [a, c] = await Promise.all([HeadName('assistant'), HeadName('coding')])
      headNames = { assistant: a ?? '', coding: c ?? '' }
    } catch { /* the label stands in */ }
  }
  async function saveHeadName(h: HeadId) {
    const next = headNameDraft.trim()
    if (next === (headNames[h] ?? '')) return
    try {
      await SetHeadName(h, next)
      headNames = { ...headNames, [h]: next }
    } catch (err) {
      learningError = String(err)
    }
  }
  const headDesc = (h: HeadId) => modes.find((m) => m.name === h)?.description ?? ''
  const headGroup = (h: HeadId) => memoryGroups.find((g) => g.scope === headScope(h)) ?? emptyGroup(headScope(h))
  // Every file a head answers for: its own, plus the projects when it is
  // the desk that hosts them (projectsUnder, from Go).
  const headScopes = (h: HeadId): string[] =>
    [headScope(h), ...(projectsHost === headScope(h) ? projectGroups.map((g) => g.scope) : [])]
  const headPending = (h: HeadId) => pendingChanges.filter((c) => headScopes(h).includes(c.scope))
  // Where each waiting proposal is DECIDED, so the rail's count sits on the row
  // that opens it. The engine hands over one number, and for a day it sat
  // whole on ตัวหลัก: a "1" there with nothing on the page to match, because
  // the item was the person's and lived on เกี่ยวกับคุณ (owner, 14 ก.ย.:
  // "ขึ้น 1 แจ้งตลอด แต่ไม่บอกว่าที่ไหน"). Split by the scope's audience —
  // the person's, a delegate's (พนักงาน if it holds a chair, ลูกมือ
  // otherwise), everything else the heads'. The engine's number stays the
  // truth: anything it counts that the list has not placed yet (the list is a
  // call away at mount) stays on ตัวหลัก rather than vanishing.
  const railPending = $derived.by(() => {
    const n = { main: 0, you: 0, team: 0, agents: 0 }
    for (const c of pendingChanges) {
      const tone = scopeMeta(c.scope).tone
      if (tone === 'user') n.you++
      else if (tone === 'agent') n[chairNames.has(c.scope.trim()) ? 'team' : 'agents']++
      else n.main++
    }
    n.main = Math.max(n.main, cockpit.pendingLearned - n.you - n.team - n.agents)
    return n
  })
  // The list behind that split, kept level with the engine's count: the count
  // moves on learning:changed, and a rail that only re-read the list when a
  // section opened would put a new item on the wrong row until then.
  $effect(() => {
    void cockpit.pendingLearned
    void (async () => {
      try {
        const rows = await ListPendingChanges()
        pendingChanges = rows
        if (chairNames.size === 0 && rows.some((c) => scopeMeta(c.scope).tone === 'agent')) {
          chairNames = new Set((await ListChairs()).map((c) => c.name))
        }
      } catch { /* the engine is not up: the row keeps the count it has */ }
    })()
  })
  const headDecided = (h: HeadId) => decidedChanges.filter((c) => headScopes(h).includes(c.scope))
  // A delegate's proposals: its scope is its bare name (memoryScope.ts).
  const agentPendingFor = (name: string) => pendingChanges.filter((c) => c.kind !== 'skill' && c.scope === name.trim())

  // The face at the head of a profile page is alive, not a tile: it breathes,
  // follows the pointer, and a click gets the companion's reaction — one of
  // the same four, with the same hop, then back (owner, 14 ก.ย. 2026: "ทำให้
  // มันกดเล่นได้ด้วยนะ ไม่ใช่ยืนนิ่ง"). One face per page, so the roster rule
  // (still, MASCOT.md §3.7) does not apply here.
  const HERO_REACTIONS: PoseId[] = ['greeting', 'cheer', 'helping', 'wink']
  const HERO_REACT_MS = 1600
  let heroPose = $state<PoseId | undefined>(undefined)
  let heroHop = $state(false)
  let heroTimer: ReturnType<typeof setTimeout> | undefined
  function pokeHero() {
    const next = HERO_REACTIONS[Math.floor(Math.random() * HERO_REACTIONS.length)]
    heroPose = next === heroPose ? HERO_REACTIONS[(HERO_REACTIONS.indexOf(next) + 1) % HERO_REACTIONS.length] : next
    heroHop = false
    requestAnimationFrame(() => (heroHop = true))
    clearTimeout(heroTimer)
    heroTimer = setTimeout(() => {
      heroPose = undefined
      heroHop = false
    }, HERO_REACT_MS)
  }
  $effect(() => () => clearTimeout(heroTimer))
  // A desk's switch on a server: the desk's id in that server's `for:` list,
  // written through the room's one writer (SetMCPServerTargets) the way the
  // agent box does — the room's picker and this switch are one call, not two
  // stores. PlacementTargets names the desks by their mode name, so the id
  // is the head's own name.
  const isOnHead = (srv: MCPRow, h: HeadId) => (srv.for ?? []).includes(h)
  const toggleHeadMCP = (srv: MCPRow, h: HeadId) => runMCP('target:' + srv.name, async () => {
    const cur = srv.for ?? []
    await SetMCPServerTargets(srv.name, isOnHead(srv, h) ? cur.filter((x) => x !== h) : [...cur, h])
    await loadMCP()
  })
  const headServerCount = (h: HeadId) => liveServers.filter((s) => isOnHead(s, h)).length
  // The shelf, as a desk sees it: all of it (mode.go: a skill is knowledge,
  // not capability, so no desk is ever without one). Listed here so the page
  // answers "what does this one know" without a trip to the room.
  const openHead = (h: HeadId) => {
    headNameDraft = headNames[h] ?? ''
    mainHead = h
    mainTab = 'identity'
    startersOwner = { kind: 'desk', head: h }
    void loadHeadStarters(h)
    if (!mcpLoaded) void loadMCP()
    if (!shelfLoaded) void loadShelf()
    deskOpen = false
    deskMsg = ''
    closeIdentityFile()
    void loadDeskFile(h)
    void loadIdentityFiles(h)
  }
  $effect(() => {
    if (active === 'main') {
      void loadLearning()
      void loadHeadNames()
      void loadModes()
    }
  })

  // ---------- Learning ----------
  //
  // This page exists because the agent proposing things is only half the
  // design. Without somewhere to see what it wants to remember, why, and what
  // it already remembers, "the agent learns" is indistinguishable from "the
  // agent changes itself" — and the second one is what nobody should have to
  // take on trust.
  let learningOn = $state(true)
  let sessionReviewAutoOn = $state(false)
  let sessionReviewBusy = $state(false)
  let sessionReviewMsg = $state('')
  // Habits (the requests typed again and again) left for ห้องความสามารถ ›
  // ชุดคำสั่ง on 14 ก.ย. 2026; the one tab left is memory.
  let pendingChanges = $state<engine.PendingChange[]>([])
  let decidedChanges = $state<engine.PendingChange[]>([])
  // The decided list is a record, not a queue: nothing is waiting on it and the
  // reason it is kept at all is so "why does it think that?" can be answered
  // months later. Twenty rows of it sat open above everything else on this page
  // (owner, 2026-08-14: "ตอนนี้มันรกเกิน"), which is a lot of screen for
  // something nobody came here to read.
  //
  // Four, then a count. Same shape the sidebar's project groups already use —
  // preview, then a button — and the count is there because "ดูเพิ่มเติม" on its
  // own does not say whether it hides two rows or two hundred.
  const DECIDED_PREVIEW = 4
  let decidedExpanded = $state(false)
  let learningError = $state('')
  let learningBusy = $state(0)
  // One row per remembered line. The file is a bullet list and always was —
  // showing it as one <pre> was the app describing the file rather than being
  // a way into it, so the only way to fix a line was to go and open the folder.
  // One group per scope that holds anything: the main agent's file, each desk's,
  // each project's. It was the main agent's alone until a desk and a project
  // could be the destination — and a line approved into a file this page could
  // not show is a line only the folder knows about.
  type MemoryGroup = {
    scope: string; lines: string[]; orphan: boolean
    bytes: number; maxBytes: number; full: boolean; projectsUnder: boolean
    // A delegate's own dress, so its block wears the face its card does.
    look?: ReturnType<typeof lookOf>
  }
  let memoryGroups = $state<MemoryGroup[]>([])
  // The projects the store still knows, for the orphan group's ย้ายไปที่…
  // picker. Loaded with the memory list because the two are one question:
  // which of these files can a session still arrive at, and where else could
  // an orphaned one go.
  let knownProjects = $state<{ name: string; rootPath: string }[]>([])
  // Which orphan group has its move picker open (by scope), '' for none.
  let adoptOpen = $state('')
  let memoryScopeError = $state('')
  // Which row is open for editing, and in which scope. -1 for none. One at a
  // time: these lines are short, and a page of open textareas is a form nobody
  // knows the state of. The scope rides along because the same index exists in
  // every group, and the save has to reach the right file.
  let memoryEditing = $state(-1)
  let memoryEditingScope = $state('')
  let memoryDraft = $state('')
  let memorySaving = $state(false)

  // The other queue, and deliberately not one variable of the block above:
  // nothing here is a proposal, nothing here can be approved, and nothing here
  // ends up in a file. Sharing state would be the same conflation this split
  // exists to undo (docs/architecture/system-problems-vs-learning-2026-08-18.md).
  let systemIssues = $state<engine.PendingChange[]>([])
  let decidedIssues = $state<engine.PendingChange[]>([])
  let issuesExpanded = $state(false)
  let issuesError = $state('')
  let issuesBusy = $state(0)

  async function loadIssues() {
    try {
      issuesError = ''
      systemIssues = await ListSystemIssues()
      decidedIssues = await ListDecidedIssues(20)
    } catch (err) {
      issuesError = String(err)
    }
  }

  // Take one problem to the assistant instead of to the developer.
  //
  // Whose fault a repeated failure is — this machine, Aetox, or the agent's own
  // way of calling the tool — is exactly what the user is being asked to judge
  // with nothing to go on, and it is a question the assistant can go and answer.
  // So the message names all three possibilities rather than asserting one.
  //
  // It goes in as the user's own visible message (owner's requirement, and the
  // honest shape): the problem is the first thing in the new chat, in words they
  // can read and edit the follow-up to. Nothing is decided by asking — the row
  // stays waiting, because the answer may send it either way.
  //
  // The evidence row ids are deliberately left out. They are for the GitHub
  // form, where somebody can query the database; in a chat they are noise, and
  // the agent can find the runs from the sentence itself (session_search reads
  // tool_runs).
  function consultPrompt(c: engine.PendingChange): string {
    return t('settings.issuesConsultPrompt', { body: c.body, reason: c.reason })
  }

  async function consultIssue(c: engine.PendingChange) {
    onClose()
    await startChatWith(consultPrompt(c))
  }

  // Reporting is the About page's door with this cluster written into the body:
  // same URL builder, same prefill, same "the user reads the whole thing on
  // GitHub and presses send themselves". A second door would be a second
  // privacy story to keep true, and this one is already written down.
  //
  // Marked reported after the form opens, never before — and marked even though
  // the user may read it and close the tab. What this side can honestly record
  // is that the problem was carried out the door, which is exactly what stops
  // it sitting here asking again.
  async function reportIssue(c: engine.PendingChange) {
    issuesBusy = c.id
    try {
      issuesError = ''
      await openIssueForm('problem', c)
      await MarkIssueReported(c.id)
      await loadIssues()
    } catch (err) {
      issuesError = String(err)
    } finally {
      issuesBusy = 0
    }
  }

  async function dismissIssue(id: number) {
    issuesBusy = id
    try {
      issuesError = ''
      await RejectPendingChange(id)
      await loadIssues()
    } catch (err) {
      issuesError = String(err)
    } finally {
      issuesBusy = 0
    }
  }

  async function loadLearning() {
    try {
      learningError = ''
      learningOn = await LearningEnabled()
      sessionReviewAutoOn = await SessionReviewAuto()
      pendingChanges = await ListPendingChanges()
      decidedChanges = await ListDecidedChanges(20)
      const scopes = await LearnedScopeInfos()
      memoryGroups = await Promise.all(
        scopes.map(async (info) => ({
          scope: info.scope, orphan: info.orphan, lines: await LearnedEntries(info.scope),
          bytes: info.bytes ?? 0, maxBytes: info.maxBytes ?? 0, full: !!info.full, projectsUnder: !!info.projectsUnder,
        })),
      )
      // Offered as move targets only when an orphan needs one — but loaded
      // here so the picker opens filled rather than after a spinner.
      if (memoryGroups.some((g) => g.orphan)) {
        knownProjects = (await RecentProjects()).map((p: { name: string; rootPath: string }) => ({ name: p.name, rootPath: p.rootPath }))
      }
    } catch (err) {
      learningError = String(err)
    }
  }

  async function adoptScope(scope: string, rootPath: string) {
    memoryScopeError = ''
    try {
      await AdoptMemoryScope(scope, rootPath)
      adoptOpen = ''
      await loadLearning()
    } catch (err) {
      memoryScopeError = String(err)
    }
  }

  async function forgetScope(scope: string) {
    memoryScopeError = ''
    try {
      await ForgetMemoryScope(scope)
      await loadLearning()
    } catch (err) {
      memoryScopeError = String(err)
    }
  }

  function startMemoryEdit(scope: string, i: number) {
    memoryEditing = i
    memoryEditingScope = scope
    memoryDraft = memoryGroups.find((g) => g.scope === scope)?.lines[i] ?? ''
  }

  function isEditing(scope: string, i: number): boolean {
    return memoryEditing === i && memoryEditingScope === scope
  }

  function cancelMemoryEdit() {
    memoryEditing = -1
    memoryEditingScope = ''
    memoryDraft = ''
  }

  // Saving and forgetting are the same write with a different body — an empty
  // one removes the line (learned.EditEntry). Keeping them one call means the
  // two paths cannot drift on which row they address.
  async function commitMemory(scope: string, index: number, text: string) {
    memorySaving = true
    try {
      learningError = ''
      await SaveLearnedEntry(scope, index, text)
      cancelMemoryEdit()
      // Re-read rather than patching the array: the row positions the next edit
      // sends have to be the file's, and a delete moves every line below it.
      await loadLearning()
      // A delegate's rows are the same snippet on its own page; its list and
      // meter are not in memoryGroups, so they are re-read here.
      if (agentReachFor && scope === agentReachFor) {
        agentMemory = await LearnedEntries(scope)
        void loadAgentMemInfo(scope)
      }
    } catch (err) {
      learningError = String(err)
    } finally {
      memorySaving = false
    }
  }

  function onMemoryKeydown(e: KeyboardEvent, scope: string, index: number) {
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation() // Escape also closes Settings; one press closes one layer
      cancelMemoryEdit()
      return
    }
    // Enter saves, Shift+Enter breaks the line. A remembered line is one
    // sentence, so the common key does the common thing.
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (memoryDraft.trim()) void commitMemory(scope, index, memoryDraft)
    }
  }

  const USER_LINE_PATTERN = /^(user['’s\s]|ผู้ใช้|คุณ\b|who\s+the\s+user)/i
  function isUserLine(line: string): boolean {
    return USER_LINE_PATTERN.test(line.trim())
  }

  const emptyGroup = (scope: string): MemoryGroup =>
    ({ scope, lines: [], orphan: false, bytes: 0, maxBytes: 0, full: false, projectsUnder: false })
  // One queue, two pages: what is about the person is decided on เกี่ยวกับคุณ,
  // the rest here. A sub-agent's proposal stays here too — its page has the
  // file, not the queue.
  const youPending = $derived(pendingChanges.filter((c) => c.scope === USER_SCOPE))
  const youDecided = $derived(decidedChanges.filter((c) => c.scope === USER_SCOPE))
  const userMemoryGroup = $derived(memoryGroups.find((g) => g.scope === USER_SCOPE) ?? emptyGroup(USER_SCOPE))
  // One block per desk (11 ก.ย.): the assistant's shared file first, then every
  // desk that keeps its own — the Go side lists those even while empty, so a
  // room is drawn before anything is in it. Project files hang under the desk
  // whose sessions write them (projectsUnder), not in a list of their own:
  // a person looking for "what did coding learn" is owed one place to look.
  const deskGroups = $derived(
    memoryGroups.filter((g) => g.scope === MAIN_SCOPE || g.scope.startsWith('mode:'))
      .sort((a, b) => (a.scope === MAIN_SCOPE ? -1 : b.scope === MAIN_SCOPE ? 1 : 0)),
  )
  const projectGroups = $derived(memoryGroups.filter((g) => g.scope.startsWith('project:')))
  const projectsHost = $derived(deskGroups.find((g) => g.projectsUnder)?.scope ?? '')
  const userLinesInMain = $derived(
    memoryGroups.find((g) => g.scope === MAIN_SCOPE)?.lines.filter(isUserLine) ?? []
  )
  // Where a line or a proposal can be sent instead: every file the page draws,
  // minus the one it is in and minus a folder no session can reach.
  function moveTargets(from: string): string[] {
    const all = [USER_SCOPE, ...deskGroups.map((g) => g.scope), ...projectGroups.filter((g) => !g.orphan).map((g) => g.scope)]
    return all.filter((s, i) => s !== from && all.indexOf(s) === i)
  }
  // The open "ย้ายไปที่…" menu, one at a time: a row's scope and index, or a
  // proposal's id. Closed by choosing, by Escape, or by clicking elsewhere.
  let moveOpen = $state('')
  function moveKey(scope: string, i: number) { return `${scope}#${i}` }
  function toggleMove(key: string) { moveOpen = moveOpen === key ? '' : key }
  function closeMoveOnOutside(e: MouseEvent) {
    if (moveOpen && !(e.target as HTMLElement)?.closest?.('.mem-move')) moveOpen = ''
  }
  $effect(() => {
    if (!moveOpen) return
    document.addEventListener('click', closeMoveOnOutside, true)
    return () => document.removeEventListener('click', closeMoveOnOutside, true)
  })
  // Capacity, as the meter draws it. The Go side answers "full" for one more
  // short line (learned.Full) — the header counts against the ceiling, so a
  // bytes/max ratio alone would read 95% as room that is not there.
  function capPct(g: MemoryGroup): number {
    return g.maxBytes > 0 ? Math.min(100, Math.round((g.bytes / g.maxBytes) * 100)) : 0
  }
  function capTone(g: MemoryGroup): 'ok' | 'near' | 'full' {
    return g.full ? 'full' : capPct(g) >= 80 ? 'near' : 'ok'
  }
  // The verb for a proposal, in the user's language — the raw op ("add") was
  // the database's own enum in the middle of a Thai sentence. The card in the
  // chat said this first (MemoryCard); the page says the same.
  function opAsk(c: engine.PendingChange): string {
    if (c.kind === 'skill') return c.op === 'create' ? t('chat.skillCreateAsk') : t('chat.skillTuneAsk')
    return c.op === 'remove' ? t('settings.learningOpRemove')
      : c.op === 'replace' ? t('settings.learningOpReplace')
      : t('settings.learningOpAdd')
  }
  // Only a NEW line can be kept somewhere else: a replace or a remove names a
  // line that lives in one file (ApprovePendingChangeTo refuses the rest).
  function canRedirect(c: engine.PendingChange): boolean {
    return c.kind === 'memory' && c.op === 'add'
  }
  async function decideChangeTo(id: number, scope: string) {
    moveOpen = ''
    learningBusy = id
    try {
      learningError = ''
      await ApprovePendingChangeTo(id, scope)
      await loadLearning()
    } catch (err) {
      learningError = String(err)
    } finally {
      learningBusy = 0
    }
  }

  // A move that failed says so IN THE BLOCK it was tried from. The first
  // cut sent the error to the page's top — off-screen from the row the
  // user pressed, so a full destination (USER.md at 4,086 of 4,096 bytes,
  // owner's screenshot 11 ก.ย.) read as a button that did nothing.
  let moveError = $state<{ scope: string; text: string } | null>(null)
  function isFull(scope: string): boolean {
    return memoryGroups.find((g) => g.scope === scope)?.full ?? false
  }
  function moveFailure(toScope: string, err: unknown): string {
    return String(err).includes('is full')
      ? t('settings.memoryTargetFull', { name: scopeLabel(toScope) })
      : String(err)
  }
  // "ให้ผู้ช่วยช่วยสรุป" (11 ก.ย.): when a file is full, the model drafts a
  // shorter list and the user reads it beside the current one before anything
  // is written. One draft open at a time, in the block it belongs to.
  let consolidation = $state<engine.MemoryConsolidation | null>(null)
  let consolidating = $state('')
  let consolidateError = $state<{ scope: string; text: string } | null>(null)
  async function consolidate(scope: string) {
    if (consolidating) return
    consolidating = scope
    consolidateError = null
    consolidation = null
    try {
      consolidation = await ConsolidateMemory(scope)
    } catch (err) {
      // The one failure a person can do something about is named in their
      // language: the model came back longer twice (the owner's first live
      // run, 4,862 bytes for a 3,691-byte file). Anything else is the
      // provider's own words.
      consolidateError = {
        scope,
        text: String(err).includes('not shorter') ? t('settings.memoryConsolidateNotShorter') : String(err),
      }
    } finally {
      consolidating = ''
    }
  }
  async function applyConsolidation() {
    const draft = consolidation
    if (!draft) return
    memorySaving = true
    try {
      await ApplyMemoryLines(draft.scope, draft.after)
      consolidation = null
      await loadLearning()
    } catch (err) {
      consolidateError = { scope: draft.scope, text: String(err) }
    } finally {
      memorySaving = false
    }
  }
  async function moveMemory(fromScope: string, toScope: string, index: number) {
    memorySaving = true
    moveError = null
    try {
      learningError = ''
      await MoveLearnedEntry(fromScope, toScope, index)
      cancelMemoryEdit()
      await loadLearning()
    } catch (err) {
      moveError = { scope: fromScope, text: moveFailure(toScope, err) }
    } finally {
      memorySaving = false
    }
  }

  let migrateBusy = $state(false)
  async function quickMigrateUserLines() {
    if (migrateBusy) return
    migrateBusy = true
    moveError = null
    try {
      learningError = ''
      const mainGroup = memoryGroups.find((g) => g.scope === MAIN_SCOPE)
      if (!mainGroup) return
      // Move backwards so row indices in MainScope don't shift
      for (let i = mainGroup.lines.length - 1; i >= 0; i--) {
        if (isUserLine(mainGroup.lines[i])) {
          await MoveLearnedEntry(MAIN_SCOPE, USER_SCOPE, i)
        }
      }
    } catch (err) {
      moveError = { scope: MAIN_SCOPE, text: moveFailure(USER_SCOPE, err) }
    } finally {
      migrateBusy = false
      // Whatever moved before a failure has moved; show the file as it is.
      await loadLearning()
    }
  }

  async function toggleLearning() {
    try {
      await SetLearningEnabled(!learningOn)
      await loadLearning()
    } catch (err) {
      learningError = String(err)
    }
  }

  async function toggleSessionReviewAuto() {
    try {
      await SetSessionReviewAuto(!sessionReviewAutoOn)
      await loadLearning()
    } catch (err) {
      learningError = String(err)
    }
  }

  async function runSessionReviewNow() {
    sessionReviewBusy = true
    sessionReviewMsg = ''
    try {
      const n = await RunSessionReview('')
      if (n > 0) {
        sessionReviewMsg = t('settings.sessionReviewRanFound', { count: String(n) })
      } else {
        sessionReviewMsg = t('settings.sessionReviewNoFacts')
      }
      await loadLearning()
    } catch (err) {
      learningError = String(err)
    } finally {
      sessionReviewBusy = false
    }
  }

  // Off means no turn ever spends a call preparing wording, whatever it ended
  // with. Written straight through rather than optimistically: this switch
  // decides whether money is spent, and a checkbox that moves before the write
  // lands is a checkbox that can lie about that.
  async function togglePreparedReply() {
    try {
      await SetPreparedReplyOn(!preparedOn)
      preparedOn = await PreparedReplyOn()
    } catch {
      // Preference file unwritable — leave the switch reading what Go last said
      // rather than showing a state nothing persisted.
    }
  }

  // การใช้คอมพิวเตอร์ left this file 14 ก.ย. 2026 for ห้องความสามารถ
  // (Capability.svelte): a reach out of the app is the same kind of thing as
  // an MCP server, and every such thing is one room.

  async function decideChange(id: number, approve: boolean) {
    learningBusy = id
    try {
      learningError = ''
      if (approve) await ApprovePendingChange(id)
      else await RejectPendingChange(id)
      await loadLearning()
    } catch (err) {
      // Shown rather than swallowed: an approval that could not be applied
      // leaves the proposal in the list, and a button that appears to do
      // nothing is how a user concludes the feature is broken.
      learningError = String(err)
    } finally {
      learningBusy = 0
    }
  }

  // Whose memory a proposal is for. Empty scope is the assistant itself; a
  // named one is a sub-agent, a desk or a project, and saying which matters —
  // it is the difference between "everything you ask it", "one job it does",
  // and "only in this folder". Shared with the card in the chat (memoryScope):
  // the same proposal is judged in both places and must not be labelled two
  // different ways.

  $effect(() => {
    if (active === 'voice') void loadVoicePage()
    if (active === 'image') void loadImagePage()
    if (active === 'studio') void loadStudio()
  })

  $effect(() => {
    // Reads from disk only, so opening the page costs nothing and works with
    // no network. Asking the server is the separate "check again" button.
    if (active === 'account') void loadAetoxAccount()
  })

  // Asked once, before any page is drawn, because the answer decides whether
  // the account page is in the nav at all. False in every shipped build today:
  // nothing is deployed for it to talk to, and a settings page that offers a
  // sign-in reaching nothing is the placeholder the 1.0.0 bar forbids.
  $effect(() => {
    void loadAetoxAccount()
  })

  $effect(() => {
    if (active === 'issues') void loadIssues()
  })

  // ---------- Nav ----------
  // `terms` is what the page is actually about, not just what it is called.
  // Search used to match the nav label alone, so "font" and "ธีม" — two of the
  // most likely things anyone types into a settings search — found nothing,
  // even though the Appearance page has five font controls on it. The terms are
  // the page's own setting titles, so they translate with everything else.
  type NavItem = { id: string; label: string; icon: IconName; terms: string[] }
  const sections: { group: string; items: NavItem[] }[] = $derived([
    { group: t('settings.groupPersonal'), items: [
      { id: 'general', label: t('settings.general'), icon: 'slidersHorizontal',
        terms: [t('settings.shellTitle'), t('settings.approvalTitle'), t('settings.firstRunTitle')] },
      { id: 'appearance', label: t('settings.appearance'), icon: 'palette',
        terms: [
          t('settings.languageTitle'), t('settings.themeTitle'), t('settings.uiFontTitle'),
          t('settings.typeScaleTitle'), t('settings.systemZoomTitle'), t('settings.editorFontTitle'),
          t('settings.chatFontTitle'), t('settings.treeFontTitle'), t('settings.codeThemeTitle'),
        ] },
      // The assistant's mascot: what it looks like and whether it sits on the
      // screen. Its words live in mascot/avatarText.ts until the locale files
      // are free (see that file).
      { id: 'avatar', label: avatarText(i18n.locale).title, icon: 'bot',
        terms: [avatarText(i18n.locale).onScreen, avatarText(i18n.locale).shell, avatarText(i18n.locale).hue] },
      // เกี่ยวกับคุณ (14 ก.ย. 2026): the one layer every desk and every agent
      // reads the same — who the user is. Your name, USER.md and the session
      // review that writes it, moved here out of การเรียนรู้ so a person's
      // own data is never mixed with what the assistant is (that is ตัวหลัก)
      // or what it worked out on its own.
      { id: 'you', label: t('settings.you'), icon: 'circleUser',
        terms: [t('settings.youName'), t('settings.learningUserSection'), t('settings.sessionReviewTitle'), 'USER.md'] },
      // คำสั่งประจำตัว left this menu 14 ก.ย. 2026 for ตัวหลัก › ตัวตน: the
      // four files are what a head IS, one set per head, and a head's page
      // is where that is edited. Its search terms went to the ตัวหลัก row.
      // การเรียนรู้ left this menu 14 ก.ย. 2026: what it held has three homes now
      // (เกี่ยวกับคุณ, ตัวหลัก, ห้องความสามารถ › ชุดคำสั่ง) and its one switch
      // is under ทั่วไป with the other switches of the system.
      // ปรับสกิลอัตโนมัติ left this menu 14 ก.ย. 2026 for the สกิล heading of
      // ห้องความสามารถ (Capability.svelte): a queue of edits to a shelf skill
      // lives beside the shelf.
      // Next to learning because that is where these used to arrive, and the
      // adjacency is the point: what Aetox worked out about you, and what keeps
      // going wrong, are two different things that spent a year in one queue
      // (docs/architecture/system-problems-vs-learning-2026-08-18.md).
      { id: 'issues', label: t('settings.issues'), icon: 'alertTriangle',
        terms: [t('settings.issuesReport'), t('settings.aboutReport')] },
    ]},
    { group: t('settings.groupModels'), items: [
      { id: 'models', label: t('settings.modelSettings'), icon: 'brain',
        terms: [t('settings.providers'), t('settings.apiKeyLabel'), t('settings.baseUrl'), t('settings.signInLabel'), t('settings.modelList')] },
    ]},
    // The company, as its own group (owner, 14 ก.ย. 2026: "ตัวหลักถึงลูกมือ
    // ควรจะแยกหัวข้อเป็นของตัวเองเลย"): the four rows that are people — the
    // heads, the staff, the hands, the teams they form — under one heading,
    // apart from the model they run on. The order is the ranks' order.
    { group: t('settings.groupAgents'), items: [
      // ตัวหลัก (14 ก.ย. 2026): the two heads a person actually talks to, each
      // with a page of its own in the shape every agent already has — the
      // feedback that started this was "ไม่รู้ว่าตัวหลักปรับแต่งได้", and a thing
      // with no page is a thing that looks unconfigurable. Beside เอเจนเฉพาะทาง
      // on purpose (owner: "B ดีสุด จำง่าย"): main, specialists, helpers, teams.
      { id: 'main', label: t('settings.mainHeads'), icon: 'userRound',
        terms: [t('desk.assistant'), t('desk.coding'), t('settings.learningAssistantSection'), t('settings.identity'), 'MEMORY.md', 'modes/coding.md', 'identity.md', 'thinking.md', 'context.md'] },
      // The people you talk to, then the helpers the assistant runs, then the
      // teams that group the first kind. Configuring an agent lives here again
      // since 13 ก.ย. 2026: for a day (12 ก.ย., "เอาเอเจนออกจากหน้าตั้งค่า")
      // this row was gone and the editor was reached only through a gear on
      // the roster page — a room with no door in the rail, and three pages
      // pointing at each other to explain where the list went. The roster page
      // is for talking; this row is for everything else about a person.
      { id: 'team', label: t('settings.team'), icon: 'bot',
        terms: [t('settings.teamNew'), t('settings.agentsFolder'), t('settings.subagentsMine'), t('settings.subagentsBuiltin')] },
      { id: 'agents', label: t('settings.subagents'), icon: 'puzzle',
        terms: [t('settings.subagentsMine'), t('settings.subagentsBuiltin')] },
      // Teams at the foot of the group, beside agents and never inside the
      // agent editor (§256): a roster is not a person, and the owner asked for
      // the two pages apart and this one last ("เพิ่มตั้งค่าทีมเอเจนที่ข้างล่าง").
      { id: 'teams', label: t('settings.teams'), icon: 'users',
        terms: [t('office.newTeam'), t('settings.teamSideAssistant'), t('settings.teamSideCode')] },
    ]},
    { group: t('settings.groupTools'), items: [
      // เครื่องมือ left this menu 14 ก.ย. 2026 for the เครื่องมือในตัว heading of
      // ห้องความสามารถ (Capability.svelte), the way สกิล and MCP did: what the
      // assistant can reach is one room. Voice stays — it is a setting, not a
      // register.
      { id: 'voice', label: t('settings.voice'), icon: 'mic',
        terms: ['TTS', 'STT', 'whisper', t('settings.sttHeading'), t('settings.ttsHeading'), t('settings.speechModel'), t('settings.audioInput'), t('settings.audioOutput')] },
      // Beside เสียง and not in its own group: both pages configure ONE tool
      // each — that one audio_transcribe and the ฟัง button, this one
      // image_make — so they are the same kind of row and belong together.
      { id: 'image', label: t('settings.image'), icon: 'image',
        terms: ['Pollinations', 'DALL-E', 'gpt-image', 'Grok', 'Gemini', 'image_make', t('settings.imageEngine')] },
      // The studio's shelf of raw material for the two video agents. Beside
      // เสียง and สร้างภาพ because it is the same kind of row — one page for
      // one thing the agent reaches for — and not on งานวิดีโอ, which asks
      // "make or cut?" and should not also be a file manager.
      { id: 'studio', label: t('settings.studio'), icon: 'clapperboard',
        terms: ['SFX', 'overlay', 'asset_find', t('settings.studioAdd'), t('settings.studioSources')] },
      // การเชื่อมต่อ and การใช้คอมพิวเตอร์ left this menu 14 ก.ย. 2026 for
      // headings of their own in ห้องความสามารถ (Capability.svelte), the way
      // MCP, สกิล and เครื่องมือ did: a reach out of the app is what that room
      // is for.
      // เครื่องระยะไกล — the engine on another machine over ssh (§248 phase
      // 3). Stays: it moves the whole engine to another machine, which is
      // where the work happens rather than what the assistant can reach.
      { id: 'remote', label: t('settings.remote'), icon: 'server',
        terms: ['ssh', 'Linux', t('settings.remoteConnect'), t('settings.remoteAdd')] },
      // ชุดคำสั่ง left this menu 14 ก.ย. 2026 for a heading of its own in
      // ห้องความสามารถ (Capability.svelte): a preset is the one thing on that
      // rail the user writes themselves, and it took the Habits tab of
      // การเรียนรู้ with it as its second row.
    ]},
    { group: t('settings.groupAbout'), items: [
      // First in this group and nowhere near the model sign-ins: those decide
      // which account pays for a request, this one is who you are to Aetox.
      // It is in About rather than up top because it configures nothing the
      // app does today.
      // Left out entirely when this build has no id server, which is every
      // shipped build today. Not greyed out and not marked "soon": there is
      // nothing behind it yet, and a row that opens onto nothing is worse than
      // a row that is not there.
      ...(aetoxAccount?.configured
        ? [{ id: 'account', label: t('settings.account'), icon: 'circleUser',
             terms: ['GitHub', 'Google', t('settings.accountSignOut')] } satisfies NavItem]
        : []),
      // Usage lives here rather than under Tools: it is a report about the app,
      // not a thing to configure, which is the same kind of page as About.
      { id: 'usage', label: t('settings.usage'), icon: 'chartColumn',
        terms: [t('settings.usageByModel'), t('settings.usageTotalTokens'), t('settings.usageCacheHitRate')] },
      { id: 'about', label: t('settings.about'), icon: 'package',
        terms: [t('settings.aboutVersion'), t('settings.aboutCheck'), t('settings.aboutReport'), t('settings.aboutFeedback')] },
      { id: 'sponsor', label: t('settings.sponsor'), icon: 'heart', terms: ['PromptPay', 'GitHub'] },
    ]},
  ])

  const SPONSOR_URL = 'https://github.com/Mikedev115/Aetox/blob/main/SPONSOR.md'
  // The repository, not the marketing site: the site is one page now, and the
  // place a supporter actually wants to land — the code, the releases, the
  // issues, the name on the commits — is here.
  const SITE_URL = 'https://github.com/Mikedev115/Aetox'
  // The one link that is right on every channel, whether or not a check ran.
  const RELEASES_URL = 'https://github.com/Mikedev115/Aetox/releases'
  const ISSUES_URL = 'https://github.com/Mikedev115/Aetox/issues'

  // What the user has to say about Aetox goes to the developer — as a GitHub
  // issue the user submits, not as anything the app sends. Two rows, one
  // destination: a problem report and plain feedback differ only in the hint
  // they open with, because a suggestion forced through a bug template arrives
  // apologising for not being a bug. The URL prefills the new-issue form with
  // the two facts the user should not have to hunt for (version, OS);
  // everything else is theirs to write, on a page where they read the whole
  // message before pressing send, signed in as themselves. This is the whole
  // privacy story: the last reader before anything leaves the machine is the
  // person it belongs to.
  //
  // Deliberately NOT the learning loop's door. That queue is for lessons about
  // this user and this machine; "Aetox is broken" and "I wish Aetox did X" are
  // facts about the product, and the two kinds of report must never share a
  // path (see summarize.go on state reports, the third kind, which goes
  // nowhere at all).
  // `cluster` is one row from the problems room: the same form, opened with the
  // failure already written into it. It goes above the blank lines, not below
  // the ---, because it is the subject of the report and the user is about to
  // write around it. Everything else on this path is unchanged, which is the
  // point — one door, one prefill, one story about what leaves the machine.
  async function openIssueForm(kind: 'problem' | 'feedback', cluster?: engine.PendingChange): Promise<void> {
    const ua = navigator.userAgent
    const os = ua.includes('Windows') ? 'Windows' : ua.includes('Mac') ? 'macOS' : 'Linux'
    const version = (appVersion ? 'v' + appVersion : t('settings.aboutReportUnknown'))
      + (updateStatus?.channel ? ` (${updateStatus.channel})` : '')
    const hint = kind === 'problem' ? t('settings.aboutReportBodyHint') : t('settings.aboutFeedbackBodyHint')
    const lines = [`<!-- ${hint} -->`]
    if (cluster) {
      lines.push('', cluster.body)
      if (cluster.reason) lines.push('', cluster.reason)
      if (cluster.evidence) lines.push('', cluster.evidence)
    }
    lines.push(
      '', '', '---',
      `${t('settings.aboutReportVersion')}: ${version}`,
      `${t('settings.aboutReportOS')}: ${os}`,
    )
    if (kind === 'problem') {
      // The evidence: what the app most recently complained about internally
      // (already secret-scrubbed at the moment each line was written). Only
      // the problem door carries it — feedback needs no logs. Trimmed from
      // the OLD end when over budget: a prefill URL has a length limit, and
      // the newest lines are the ones about whatever just went wrong. Folded
      // in <details> and labelled deletable, because it is the user's form.
      try {
        let log = (await RecentDebugLog()) ?? []
        while (log.length > 0 && !issueURLFits(lines, log)) log = log.slice(1)
        const text = log.join('\n')
        if (text) {
          lines.push('', `<details><summary>${t('settings.aboutReportLogTitle')}</summary>`, '', '```', text, '```', '</details>')
        }
      } catch {
        // No log is not a reason to block a report.
      }
    }
    // Trimmed from the end if the report itself is over budget even with no log
    // at all. A cluster body plus a stack trace can do it, and the alternative
    // is the button doing nothing, which is what it did before.
    let body = lines.join('\n')
    while (body.length > 0 && issueURL(body).length > ISSUE_URL_BUDGET) {
      body = body.slice(0, -256)
    }
    BrowserOpenURL(issueURL(body))
  }

  // encodeURIComponent leaves !~*'() alone — they are "unreserved marks" it is
  // specified not to touch. Wails refuses to open a URL containing any of
  // !~*() as a shell-metacharacter risk (ValidateAndSanitizeURL), and it
  // refuses by logging to a place no user sees and returning, so the button
  // did nothing and said nothing. One ordinary parenthesis was enough, and the
  // problem bodies are full of them: "เกิด 3 ครั้ง (ตัวอย่างล่าสุด...)".
  //
  // ' is encoded too. Wails does not object to it, but a body that is
  // percent-encoded except for one character is a rule with an exception to
  // remember, and there is nothing to gain by keeping it readable in a URL.
  const encodeIssueBody = (s: string) =>
    encodeURIComponent(s).replace(/[!~*'()]/g, (c) => '%' + c.charCodeAt(0).toString(16).toUpperCase())

  const issueURL = (body: string) => `${ISSUES_URL}/new?body=${encodeIssueBody(body)}`

  // ISSUE_URL_BUDGET is the whole URL, encoded, and it exists because the old
  // budget counted the wrong thing and the button silently did nothing.
  //
  // The cap used to be 4000 characters of raw log text. Thai is three bytes per
  // character in UTF-8 and each byte becomes three characters again once
  // percent-encoded, so a 4000-character Thai log is ~36,000 characters of URL.
  // Windows caps a command line at 32,767, and `BrowserOpenURL` hands the URL to
  // `rundll32` — over that it fails, and Wails logs the failure to a place no
  // user sees rather than returning it, so pressing แจ้งปัญหานี้ produced no
  // window, no error, nothing (measured 2026-08-18: 4,251 raw characters became
  // a 35,659-character URL).
  //
  // Was 8000, which is over GitHub's own ceiling, not under it. Measured
  // 2026-09-15 against issues/new with curl, URL length in characters:
  //   signed in   : 6,934 opens · 7,012 answers 500 "Whoops, something went
  //                 wrong" — the report from v1.6.3 portable, where a full log
  //                 put the URL at ~8,000 every time.
  //   signed out  : the sign-in bounce carries the URL again as return_to
  //                 (encoded once more, so ~1.6x longer) and GitHub drops it
  //                 above 4,534 — the user signs in and lands on an empty
  //                 form, with no sign anything was lost.
  //   above 8,000 : 414 / 502 from the edge.
  // 4000 clears both, and the log is what gives: the cluster and the version
  // are written first and the log is trimmed from its old end to fit.
  const ISSUE_URL_BUDGET = 4000

  function issueURLFits(head: string[], log: string[]): boolean {
    const body = [...head, '', '<details><summary>x</summary>', '', '```', log.join('\n'), '```', '</details>'].join('\n')
    return issueURL(body).length <= ISSUE_URL_BUDGET
  }

  // Which page is open survives an F5. Same reasoning as the chat/settings view
  // itself (see setActiveView in stores/cockpit.svelte.ts): sessionStorage, not
  // localStorage, because reopening the app should always start from the top —
  // but a reload during a run should not throw away where you were. Reloading
  // while three pages deep into MCP config and landing back on General is a
  // small thing that happens every single time.
  // Imported rather than spelled again: a room can send the user straight to a
  // section (openSettingsAt), and two spellings of this key would fail silently
  // and look like the page ignoring where it was told to go.
  const SECTION_KEY = SETTINGS_SECTION_KEY
  // Every id the rail can draw. Five of them (issues, voice, image, studio,
  // remote) were missing here for as long as they had existed: a door that
  // said openSettingsAt('voice') landed on ทั่วไป, quietly, because the
  // fallback below is the right answer for a page that was DELETED and the
  // wrong one for a page that was merely forgotten. Found 14 ก.ย. 2026 when
  // the tool register's door to เสียง moved rooms and got a test that opens
  // it the way a user does.
  const SECTION_IDS = new Set(['general', 'appearance', 'avatar', 'you', 'issues', 'models', 'main', 'team', 'teams', 'agents', 'voice', 'image', 'studio', 'remote', 'account', 'usage', 'about', 'sponsor'])

  function restoredSection(): string {
    try {
      const saved = sessionStorage.getItem(SECTION_KEY)
      // Validated, not trusted: a page that was removed since the value was
      // written would otherwise render nothing at all.
      if (saved && SECTION_IDS.has(saved)) return saved
    } catch {
      /* storage unavailable — start where a fresh open would */
    }
    return 'general'
  }

  // Seeded from the intent when there is one, so the page's first frame is
  // the section the gear asked for — not last visit's page for a tick, then
  // the right one. The intent is consumed in onMount above; at construction it
  // is still there to read.
  let active = $state(cockpit.settingsIntent?.section ?? restoredSection())
  let query = $state('')

  // Which memory group the learning page should take the reader to. Set only
  // by a door that names one — the page itself lists every scope and marks
  // none, which is right when you walked in through the sidebar and wrong when
  // you arrived from one agent's card asking about that agent.
  // null, not '': the assistant's own scope IS the empty string, and a
  // default of '' marked its heading as "the one you came for" on every visit.


  // The one scroller every section shares. Bound so a section change can put
  // the reader back at the top of it.
  let contentEl = $state<HTMLDivElement | null>(null)

  function openSection(id: string) {
    // เอเจนเฉพาะทาง and ลูกมือ share one editor pane (agentEditorPane). With an
    // editor open on one, the rail row of the other changed `active` and
    // nothing else — the pane stayed, and the click read as nothing (owner,
    // 14 ก.ย. 2026: "กดเมนูหน้าอื่น ๆ หรือเอเจนเฉพาะทาง มันกดหน้าลูกมือไม่ได้"). A
    // rail click means the list, so the editor closes first — through the
    // same unsaved guard the back button has, and to the list rather than to
    // wherever the editor was opened from.
    if ((id === 'agents' || id === 'team') && agentEditing !== null) {
      guardUnsaved(agentDraftKey() !== agentSnapshot, () => {
        agentEditing = null
        agentBack = 'list'
        showSection(id)
      })
      return
    }
    showSection(id)
  }
  function showSection(id: string) {
    active = id
    // Every page starts at its own top. Without this, a click made from the
    // bottom of a long section (รูปลักษณ์ carries eleven controls) keeps the
    // pane's scroll offset and lands the next section mid-list with its
    // heading off-screen above — which does not read as a new page, it reads
    // as the button having done nothing (owner, 8 ก.ย. 2026, of the สกิล row:
    // "กดเมนูแล้วหน้าไม่เปลี่ยน").
    // scrollTop rather than scrollTo({behavior:'smooth'}): the page under the
    // scroll has been replaced, so there is nothing to travel past.
    if (contentEl) contentEl.scrollTop = 0
    try {
      sessionStorage.setItem(SECTION_KEY, id)
    } catch {
      /* storage unavailable — the page just won't be remembered */
    }
  }

  const filteredSections = $derived.by(() => {
    const q = query.trim().toLowerCase()
    if (!q) return sections
    return sections
      .map((g) => ({
        ...g,
        items: g.items.filter((it) =>
          it.label.toLowerCase().includes(q) || it.terms.some((term) => term.toLowerCase().includes(q)),
        ),
      }))
      .filter((g) => g.items.length > 0)
  })

  const noSearchResults = $derived(query.trim() !== '' && filteredSections.length === 0)
</script>

<!-- Starting a self-hosted engine, and asking whether it is up.
     Two questions this page could not answer before, and both belong to it: a
     row that says "not connected" was unable to do the one thing that fixes
     that, and telling somebody to go and find a terminal from the screen they
     are already on is the complaint this closes.

     Deliberately separate from ตรวจสอบ. That button asks whether the KEY works;
     these ask whether the PROGRAM is running. Told apart they are two obvious
     fixes; run together they were one confusing failure, because a dead server
     and a dead key both come back as "could not connect".

     The command is the user's own, typed once and remembered. Nothing about
     where n8n or Windmill lives is written in this codebase — a guess would be
     wrong for everyone it was not written for — and the precedent is an MCP
     stdio server, which has always been a command in a config file. -->
<!-- One memory file's heading (11 ก.ย.): whose it is, who reads it, the file,
     and how full it is. The meter is the part that was missing — a full
     profile was a fact only the tool knew, refusing proposals and skipping
     the session review with nothing on this page saying so. -->
{#snippet pendingRow(c: engine.PendingChange)}
  {@const meta = scopeMeta(c.scope)}
  <div class="learn-row">
    <div class="learn-main">
      <!-- The verb, then whose file, then who will read it: the last is
           the decision actually being made (memoryScope.ts). -->
      <div class="learn-head">
        <span class="learn-verb">{opAsk(c)}</span>
        {#if c.kind === 'skill'}
          <span class="learn-scope">{c.scope}</span>
        {:else}
          <span class="learn-scope mem-tone-{meta.tone}"><ScopeMark {meta} size={11} face={16} /> {meta.label}</span>
          <span class="learn-aud">{meta.audience}</span>
        {/if}
      </div>
      {#if c.before}
        <!-- What it replaces, shown next to what it becomes: approving a
             change without seeing what it overwrites is not a decision. -->
        <div class="learn-before" class:learn-doc={c.kind === 'skill'}>{c.before}</div>
      {/if}
      <div class="learn-body" class:learn-doc={c.kind === 'skill'}>{c.body}</div>
      {#if c.reason}<div class="learn-why">{c.reason}</div>{/if}
    </div>
    <div class="learn-actions">
      <button type="button" class="ctrl ctrl-primary" disabled={learningBusy === c.id}
        onclick={() => decideChange(c.id, true)}>{t('settings.learningApprove')}</button>
      <button type="button" class="ctrl" disabled={learningBusy === c.id}
        onclick={() => decideChange(c.id, false)}>{t('settings.learningReject')}</button>
      {#if canRedirect(c)}
        <!-- "เก็บที่อื่น": the second half of the decision. Without it the
             only way to correct a destination was to refuse the line and
             hope it was proposed again from the right desk. -->
        <div class="mem-move">
          <button type="button" class="ctrl" disabled={learningBusy === c.id}
            aria-expanded={moveOpen === `pending#${c.id}`}
            onclick={() => toggleMove(`pending#${c.id}`)}>
            {t('settings.learningKeepElsewhere')} <Icon name="chevronDown" size={11} />
          </button>
          {#if moveOpen === `pending#${c.id}`}
            {@render moveMenu(t('settings.learningKeepIn'), moveTargets(c.scope), '', (to) => decideChangeTo(c.id, to))}
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/snippet}

{#snippet decidedRow(c: engine.PendingChange)}
  {@const meta = scopeMeta(c.scope)}
  <div class="learn-row past">
    <div class="learn-main">
      <div class="learn-head">
        <span class="learn-op" class:rejected={c.state === 'rejected'}>
          {c.state === 'approved' ? t('settings.learningStateApproved') : t('settings.learningStateRejected')}
        </span>
        {#if c.kind === 'skill'}
          <span class="learn-scope">{c.scope}</span>
        {:else}
          <span class="learn-scope mem-tone-{meta.tone}"><ScopeMark {meta} size={11} face={16} /> {meta.label}</span>
        {/if}
        <span class="learn-when">{c.decidedAt.slice(0, 10)}</span>
      </div>
      <div class="learn-body" class:learn-doc={c.kind === 'skill'}>{c.body}</div>
    </div>
  </div>
{/snippet}

{#snippet deskHead(g: MemoryGroup)}
  {@const meta = scopeMeta(g.scope)}
  {@const tone = capTone(g)}
  <div class="mem-scope mem-tone-{meta.tone}" data-mem-scope={g.scope}>
    <span class="mem-scope-ic" class:face={!!meta.head || (meta.tone === 'agent' && !!g.look)}><ScopeMark {meta} size={14} face={30} look={g.look} /></span>
    <span class="mem-scope-name">{meta.label}</span>
    <span class="learn-aud">{meta.audience}</span>
    <span class="mem-badge-file">{meta.file}</span>
    {#if g.orphan}
      <!-- The folder this file is keyed to moved or was deleted, so no session
           can ever read it again — a fact only this label states, because on
           disk the file looks exactly like a live one (§186). A label needs
           its exits: move the lines to the project the folder became, or let
           them go. -->
      <span class="mem-orphan">{t('settings.memoryOrphan')}</span>
      <span class="mem-orphan-actions">
        {#if knownProjects.length > 0}
          <button
            type="button" class="ctrl tiny"
            onclick={() => { adoptOpen = adoptOpen === g.scope ? '' : g.scope }}
          >{t('settings.memoryOrphanMove')}</button>
        {/if}
        <button
          type="button" class="ctrl tiny mem-forget"
          onclick={() => forgetScope(g.scope)}
        >{t('settings.memoryOrphanDelete')}</button>
      </span>
    {/if}
    {#if g.maxBytes > 0}
      <span class="mem-cap mem-cap-{tone}">
        <span class="mem-cap-bar"><i style="width:{capPct(g)}%"></i></span>
        <span class="mem-cap-num">{g.bytes.toLocaleString('en-US')} / {g.maxBytes.toLocaleString('en-US')} B</span>
      </span>
    {/if}
  </div>
  {#if tone !== 'ok'}
    <div class="mem-cap-note mem-cap-{tone}">
      <Icon name="alertTriangle" size={13} />
      <span>{tone === 'full' ? t('settings.memoryFull') : t('settings.memoryNearFull')}</span>
      <button type="button" class="ctrl tiny mem-consolidate" disabled={!!consolidating || g.lines.length < 2}
        onclick={() => consolidate(g.scope)}>
        <Icon name="sparkles" size={12} />
        {consolidating === g.scope ? t('settings.memoryConsolidating') : t('settings.memoryConsolidate')}
      </button>
    </div>
  {/if}
  {#if consolidateError?.scope === g.scope}
    <div class="mem-move-error"><Icon name="alertTriangle" size={13} /><span>{consolidateError.text}</span></div>
  {/if}
  {#if consolidation?.scope === g.scope}
    <!-- The draft, beside what it replaces. Read here, applied here; a list
         the user has not read is not one the app writes. -->
    <div class="mem-draft">
      <div class="mem-draft-cols">
        <div class="mem-draft-col">
          <div class="mem-draft-h">{t('settings.memoryDraftBefore')} <span class="mono">{g.bytes.toLocaleString('en-US')} B · {t('settings.memoryLines', { count: String(consolidation.before.length) })}</span></div>
          {#each consolidation.before as line, i (i)}<p class="mem-draft-line was">{line}</p>{/each}
        </div>
        <div class="mem-draft-col">
          <div class="mem-draft-h">{t('settings.memoryDraftAfter')} <span class="mono">{consolidation.bytes.toLocaleString('en-US')} B · {t('settings.memoryLines', { count: String(consolidation.after.length) })}</span></div>
          {#each consolidation.after as line, i (i)}<p class="mem-draft-line">{line}</p>{/each}
        </div>
      </div>
      {#if consolidation.note}<div class="mem-draft-note">{consolidation.note}</div>{/if}
      <div class="mem-draft-actions">
        <span class="mem-draft-hint">{t('settings.memoryDraftHint')}</span>
        <button type="button" class="ctrl" disabled={memorySaving} onclick={() => (consolidation = null)}>{t('settings.learningReject')}</button>
        <button type="button" class="ctrl ctrl-primary" disabled={memorySaving} onclick={applyConsolidation}>{t('settings.memoryDraftApply')}</button>
      </div>
    </div>
  {/if}
{/snippet}

<!-- One remembered line. Keyed by index rather than by text: two remembered
     lines can be byte-identical, and the index is also what the save
     addresses. The move button opens a menu of every other file rather than
     one fixed destination — a line can belong to any desk now. -->
{#snippet memRow(g: MemoryGroup, line: string, i: number)}
  <div class="mem-row" class:editing={isEditing(g.scope, i)}>
    {#if isEditing(g.scope, i)}
      <!-- svelte-ignore a11y_autofocus -->
      <textarea
        data-guide="settings.you.about_input"
        class="mem-input" rows="2" autofocus
        bind:value={memoryDraft}
        onkeydown={(e) => onMemoryKeydown(e, g.scope, i)}
      ></textarea>
      <div class="mem-actions">
        <button
          data-guide="settings.you.save"
          type="button" class="ctrl ctrl-primary"
          disabled={memorySaving || !memoryDraft.trim()}
          onclick={() => commitMemory(g.scope, i, memoryDraft)}
        >{t('settings.learningMemorySave')}</button>
        <button type="button" class="ctrl" disabled={memorySaving} onclick={cancelMemoryEdit}
        >{t('settings.learningMemoryCancel')}</button>
      </div>
    {:else}
      <p class="mem-text">{line}</p>
      <div class="mem-actions">
        <button
          type="button" class="icobtn tiny tip-l" aria-label={t('settings.learningMemoryEdit')}
          data-tip={t('settings.learningMemoryEdit')} disabled={memorySaving}
          onclick={() => startMemoryEdit(g.scope, i)}
        ><Icon name="pencil" size={13} /></button>
        <div class="mem-move">
          <button
            type="button" class="icobtn tiny tip-l mem-action-move"
            class:open={moveOpen === moveKey(g.scope, i)}
            aria-label={t('settings.learningMoveTo')} aria-expanded={moveOpen === moveKey(g.scope, i)}
            data-tip={t('settings.learningMoveTo')} disabled={memorySaving}
            onclick={() => toggleMove(moveKey(g.scope, i))}
          ><Icon name="arrowRight" size={13} /></button>
          {#if moveOpen === moveKey(g.scope, i)}
            {@render moveMenu(
              t('settings.learningMoveTo'), moveTargets(g.scope),
              g.scope === MAIN_SCOPE && isUserLine(line) ? USER_SCOPE : '',
              (to) => { moveOpen = ''; void moveMemory(g.scope, to, i) },
            )}
          {/if}
        </div>
        <!-- No confirm: the line is one sentence the agent wrote, the file is
             plain markdown the user owns, and a dialog for every tidy-up is
             what makes a list nobody tidies. -->
        <button
          type="button" class="icobtn tiny tip-l mem-forget" aria-label={t('settings.learningMemoryForget')}
          data-tip={t('settings.learningMemoryForget')} disabled={memorySaving}
          onclick={() => commitMemory(g.scope, i, '')}
        ><Icon name="x" size={13} /></button>
      </div>
    {/if}
  </div>
{/snippet}

<!-- Where else a line can go. Every entry says who would read it there, in
     the same tone the rest of the page uses for that file — the menu is the
     one place the whole map is visible at once. -->
{#snippet moveMenu(title: string, targets: string[], recommended: string, pick: (scope: string) => void)}
  <div class="mem-menu" role="menu">
    <div class="mem-menu-h">{title}</div>
    {#each targets as to (to)}
      {@const m = scopeMeta(to)}
      <button type="button" class="mem-menu-i" class:rec={to === recommended && !isFull(to)} class:full={isFull(to)}
        role="menuitem" disabled={isFull(to)} onclick={() => pick(to)}>
        <span class="learn-scope mem-tone-{m.tone}"><ScopeMark meta={m} size={11} face={16} /> {m.label}</span>
        <small>{isFull(to) ? t('settings.memoryFullShort') : `${to === recommended ? `${t('settings.learningMoveRecommended')} · ` : ''}${m.audience}`}</small>
      </button>
    {/each}
  </div>
{/snippet}

{#snippet voiceInstall(side: 'stt' | 'tts', eng: EngineRow | undefined, status: string)}
  <!-- The ติดตั้ง row (อนุมัติ 1 ก.ย.): shown only while the engine's own
       status says something is missing — a working engine needs no
       instructions, so success is this row disappearing and one green line
       standing in its place. -->
  {#if eng?.installCommand?.length && status}
    <div class="voice-install-row">
      <code class="voice-install-cmd">$ {eng.installCommand.join(' ')}</code>
      <button class="ctrl" disabled={voiceInstallBusy !== ''} onclick={() => copyVoiceCommand(side, eng.installCommand)}>
        {voiceCmdCopied === side ? t('settings.voiceInstallCopied') : t('settings.voiceInstallCopy')}
      </button>
      <button class="ctrl ctrl-primary" disabled={voiceInstallBusy !== ''} onclick={() => installVoiceEngine(side, eng)}>
        {voiceInstallBusy === side ? t('settings.installing') : t('settings.voiceInstallRun')}
      </button>
    </div>
    {#if voiceInstallBusy === side && voiceInstallTail}
      <div class="d voice-install-tail">{voiceInstallTail}</div>
    {/if}
  {/if}
  {#if voiceInstallFail[side]}
    <div class="d voice-install-fail">{voiceInstallFail[side]}</div>
  {/if}
  {#if voiceInstallDone[side]}
    <div class="d voice-install-ok">{t('settings.voiceInstalled', { name: voiceInstallDone[side] })}</div>
  {/if}
{/snippet}

<!-- ซับเอเจน only. The two pages used to share this markup, and the sharing was
     right while both were lists of files; they stopped being the same kind of
     thing the moment เอเจน became people you pick by face (agentCard above).
     A helper is not picked at all — it is part of the system, nobody chooses
     one, and a row is the honest shape for an inventory you read and close. -->
{#snippet profileRow(a: SubagentRow)}
  <!-- A helper is the same card one weight lighter: smaller mark, no door of
       its own, no chat. The set is fixed and the page reads (owner, 6 ส.ค.),
       so what a teammate's foot holds, this one simply does not have — and the
       difference in weight is what says which level of the company you are
       looking at, without a sentence having to say it. -->
  <div class="chair-card agc helper" class:off={reachOf(a.name) && !(reachOf(a.name)!.on && !reachOf(a.name)!.off)}>
    <div class="chair-body">
      <div class="chair-who">
        <!-- The same face as an เอเจน, at the same size (owner, 1 ก.ย.:
             *"อยากได้ UI เหมือนๆกัน"*). The mark here used to be one weight
             lighter than a teammate's, on the reasoning that the difference in
             weight said which level of the company you were looking at. The
             owner's call is that the levels are said by the page you are on —
             this one is headed ซับเอเจน and its rows carry no chat button —
             and that a second visual language for the same kind of thing costs
             more than it explains. -->
        <RankedFace tier="helper" size={44}><AgentMascot name={a.name} {...lookOf(a)} size={44} /></RankedFace>
        <span class="chair-name" title={a.path || 'built-in:' + a.name}>{a.name}</span>
        {#if delegate}
          {@const w = reachOf(a.name)}
          {#if w}
            <label class="mswitch" title={t('settings.agentReachTip')}>
              <input
                type="checkbox" checked={w.on && !w.off}
                disabled={w.off || delegateBusy !== ''}
                aria-label={t('settings.agentReach')}
                onchange={() => toggleAgentReach(a.name, w.on)}
              />
              <span></span>
            </label>
          {/if}
        {/if}
        <!-- The same cog the เอเจน card wears, opening the same editor through
             the helper door — which shows only what a helper's owner may
             change: the model and its ceiling (สมอง), the prompt and the
             description (ตัวตน, name locked), the look (อวตาร). No reach tab:
             the kit is the system's, and profile.go keeps it so whatever a
             hand-edited file says. -->
        <button
          class="icobtn tiny tip-l" disabled={agentBusy !== ''}
          aria-label={t('settings.agentConfigure')} data-tip={t('settings.agentConfigure')}
          onclick={() => openAgent(a, 'helper')}
        >
          <Icon name="settings" size={14} />
        </button>
      </div>
      <div class="d">{a.description || '—'}</div>
      <!-- A file that cannot run says why, where its owner will look — never a
           silent reinterpretation, never a card that just vanishes (the file is
           still on the user's disk). -->
      {#if a.invalid}<div class="d ag-invalid">{a.invalid}</div>{/if}
      <!-- Not fatal, and deliberately a different colour: this file runs, it is
           just doing something its author probably did not mean. -->
      {#if a.notice}<div class="d ag-notice">{a.notice}</div>{/if}
      <!-- Only the facts that differ between one helper and the next. The steps
           badge was drawn on every one of them reading "ไม่จำกัดรอบ", which is
           the same word four times down a column of four — and so was the
           source, `built-in:<name>` on every bundled card under a heading that
           already says มากับแอป, in a colour that vanished (owner, 12 ก.ย.:
           "แทบจะกลืนกับพื้นหลัง", then "ทำให้มันตรงๆสิ"). Gone: the name IS the
           file's name, the group says whose file it is, and the full path stays
           on the name's hover, exactly as the เอเจน card above does it. The one
           thing about the file worth a chip is the same one that card wears —
           a file of yours shadowing a bundled one. -->
      <div class="chair-chips">
        {#if a.overrides}<span class="chip mine">{t('settings.agentOverrides')}</span>{/if}
        {#if a.model || a.provider}<span class="chip">{[a.provider, a.model].filter(Boolean).join(' · ')}</span>{/if}
        {#if a.think}<span class="chip" title={t('settings.agentThinkTip')}>{t('settings.agentThinkChip', { level: a.think })}</span>{/if}
        {#if a.deny && a.deny.length > 0}<span class="chip deny" title={denyTip(a)}>{t('settings.agentDenyCount', { n: a.deny.length })}</span>{/if}
        {#if (a.steps ?? 0) > 0}
          <span class="chip" title={t('settings.agentStepsTip', { n: a.steps ?? 0 })}>{t('settings.agentSteps', { n: a.steps ?? 0 })}</span>
        {/if}
      </div>
    </div>
  </div>
{/snippet}

<!-- A teammate is a row again (owner, 12 ส.ค.), and this reverses the 10 ส.ค.
     call that made it a card. Both had a point and the row is written to keep
     the card's: what was wrong with the ORIGINAL row is that the name sat in a
     line of five grey tags, so there was nothing to recognise anyone by; what a
     grid of 240px cards then cost is that a team no longer fits on a screen,
     and each card spent a border, a colour band and a foot saying what a
     divider says for free.
     So the name leads, at the size the card gave it, with its face beside it
     and the tags trailing behind; the description gets its own line; and the
     two settings you change without opening anything sit on the right, out of
     the column the eye scans. The roster on the team page keeps the cards —
     there you are picking a person to talk to, here you are finding a file to
     configure, and those are different acts (§85).

     The file path stays in the name's tooltip, as it has since the card: on a
     row it was a fourth line of dim monospace under every entry, answering a
     question nobody asks while scanning. -->
{#snippet agentRow(a: SubagentRow)}
  <div class="chair-card agc" class:off={reachOf(a.name) && !(reachOf(a.name)!.on && !reachOf(a.name)!.off)}>
    <div class="chair-body">
      <div class="chair-who">
        <!-- The เอเจน page draws the same face the roster does, at the same
             size, because it is the same person seen from a different act
             (§85). The ซับเอเจน page one snippet up keeps the glyph mark on
             purpose: a helper is the assistant's own hands, and a face would
             invite the question of how to hire one. -->
        <!-- ยศ on the face's corner (RankedFace, style E — owner 14 ก.ย.):
             beside the name it fought the name for the row's width, under it
             the card grew; on the face it costs nothing and travels with the
             face everywhere. The two lists draw the same card on purpose;
             the emblem plus the page heading say which level this is. -->
        <!-- 48 for a พนักงาน, 44 for a ลูกมือ (profileRow), the heads larger
             still: the ranks read in the faces before the word is read
             (owner, 14 ก.ย.: "ทำให้พนักงานตัวใหญ่ขึ้นอีกหน่อย"). -->
        <RankedFace tier="agent" size={48}>
          <AgentMascot
            name={a.name}
            {...lookOf(a)}
            size={48}
            off={!!reachOf(a.name) && !(reachOf(a.name)!.on && !reachOf(a.name)!.off)}
          />
        </RankedFace>
        <span class="chair-name" title={a.path || 'built-in:' + a.name}>{a.name}</span>
        <div class="ag-actions">
      <!-- Whether the MAIN assistant may hand this one work. Not whether the
           agent exists: the user still opens a chat with it from the composer
           and still writes @name, and no switch on this page reaches those. That
           is what the tooltip says, and why the control carries no "เปิด/ปิด"
           wording of its own — "off" would read as "gone" while the agent is
           standing right there.

           The app's switch (style.css .mswitch), in the same shape the MCP shelf
           uses one row below: a row whose left half is a name and a description,
           and whose right half is the switch followed by the row's action
           buttons. It was a `.ctrl` chip that lit up when on, which is a state
           drawn as a button — the thing the owner sent back twice on 2026-08-20.
           A chip that is on and a chip that is merely hoverable look the same
           until you learn the colour; a switch does not have to be learned.

           Disabled, not hidden, while this kind is switched off entirely. A card
           that vanished would leave somebody wondering where their agent went; a
           card that is cooled under the switch above it explains itself. -->
          {#if delegate}
            {@const w = reachOf(a.name)}
            {#if w}
              <label class="mswitch" title={t('settings.agentReachTip')}>
                <input
                  type="checkbox" checked={w.on && !w.off}
                  disabled={w.off || delegateBusy !== ''}
                  aria-label={t('settings.agentReach')}
                  onchange={() => toggleAgentReach(a.name, w.on)}
                />
                <span></span>
              </label>
            {/if}
          {/if}
          <!-- A cog, not a labelled bar across the foot. On the roster the
               equivalent control says "คุยกับ doc" in words, and it earns the
               row because walking in to talk is what that page is FOR (owner,
               30 ส.ค.). Here the errand is configuring, the page is already the
               settings page, and a full-width button repeating the word cost
               every card a row of its own — which is what made the deck too
               tall to scan (owner, 31 ส.ค.: "ขนาดมันใหญ่ไป"). -->
          <button
            class="icobtn tiny tip-l" disabled={agentBusy !== ''}
            aria-label={t('settings.agentConfigure')} data-tip={t('settings.agentConfigure')}
            onclick={() => openAgent(a, 'agent')}
          >
            <Icon name="settings" size={14} />
          </button>
        </div>
      </div>
      <div class="d">{a.description || '—'}</div>
      <!-- A file that cannot run says why, where its owner will look — never a
           silent reinterpretation, never a card that just vanishes (the file is
           still on the user's disk). -->
      {#if a.invalid}<div class="d ag-invalid">{a.invalid}</div>{/if}
      <!-- Not fatal, and deliberately a different colour: this file runs, it is
           just doing something its author probably did not mean. -->
      {#if a.notice}<div class="d ag-notice">{a.notice}</div>{/if}
      <!-- Only what DIFFERS from the ordinary. The steps badge used to be drawn
           on every card — "ไม่จำกัดรอบ" seven times down a list of seven — which
           is a badge in the best slot on the card saying nothing at all. It is
           drawn now only when there is a real ceiling (§110: 0 is an absent
           `steps:` and a negative is the keyword; both mean no ceiling).
           The pinned model joins it for the same reason: inherit is the rule, a
           pin is the exception, and only the exception is worth a slot. The pin
           itself moved into the editor behind the gear, where the rest of what
           this agent thinks with already lives — a dropdown repeated down the
           column made the seven that inherit look exactly like the one that
           does not. -->
      <div class="chair-chips">
        {#if a.overrides}<span class="chip mine">{t('settings.agentOverrides')}</span>{/if}
        {#if a.deny && a.deny.length > 0}<span class="chip deny" title={denyTip(a)}>{t('settings.agentDenyCount', { n: a.deny.length })}</span>{/if}
        {#if (a.steps ?? 0) > 0}
          <span class="chip" title={t('settings.agentStepsTip', { n: a.steps ?? 0 })}>{t('settings.agentSteps', { n: a.steps ?? 0 })}</span>
        {/if}
        {#if a.model || a.provider}<span class="chip">{[a.provider, a.model].filter(Boolean).join(' · ')}</span>{/if}
        {#if a.think}<span class="chip" title={t('settings.agentThinkTip')}>{t('settings.agentThinkChip', { level: a.think })}</span>{/if}
      </div>
    </div>
  </div>
{/snippet}

{#snippet profileListPane(kind: 'agent' | 'helper')}
  {@const isAgent = kind === 'agent'}
  {@const rows = isAgent ? teamRows : helperRows}
  <h2>{isAgent ? t('settings.team') : t('settings.subagents')}</h2>
  <p class="muted set-sub">{isAgent ? t('settings.teamDesc') : t('settings.subagentsDesc')}</p>

  {#if isAgent}
    <div class="pp-bar">
      <button class="ctrl" onclick={() => loadAgents()}>{t('settings.refresh')}</button>
      <button class="ctrl" onclick={() => OpenAgentsFolder()}>{t('settings.agentsFolder')}</button>
      <div class="pp-bar-gap"></div>
      <!-- The door, where the eye lands first — the same shape and place as
           ทีมเอเจน's สร้างทีม (owner, 13 ก.ย. 2026: "ทำปุ่มเพิ่มเอเจนเฉพาะทาง
           ให้ชัดหน่อย … พื้นหลังสีเดียวกับสร้างทีม"). It took the slot of the
           "ไปหน้าเอเจนเฉพาะทาง" button, which the sentence above already says
           and the rail's row already offers. -->
      <button class="ctrl ctrl-primary" data-guide="office.new_agent_btn" onclick={() => newAgent(kind)}><Icon name="plus" size={14} /> {t('settings.teamNew')}</button>
    </div>
  {/if}
  {#if agentError}<div class="mset-error">{agentError}</div>{/if}

  <!-- This page's own switch. Delegation ships off, so this page is where
       somebody turns it on.

       Drawn on BOTH pages since 2026-08-20. It used to be `{#if isAgent}`, one
       switch on the เอเจน page governing both kinds — so somebody who opened
       the ซับเอเจน page while it was off found every row greyed out and nothing
       on the page explaining who had done it.

       **The token figures came off this row the same day, on the owner's call:
       "เลขตรงนี้เหมือนจะบั๊ค ๆ … เปิดไม่เปิดก็พอละ".** They were not wrong and
       that is what made them worse. Each switch showed its MARGINAL cost — what
       flipping it changes with the other switch left alone — so the number on
       this row moved when you touched the OTHER one, and swung sevenfold doing
       it: with เอเจน off, turning ซับเอเจน off removes the whole `task` tool
       (~599) where it would otherwise remove ~81. A figure that is honest,
       unpredictable from where the user is standing, and sitting next to a
       second figure with a different meaning (the whole block) reads as broken.
       A switch nobody can predict is worse than a switch with no number on it.

       If a cost belongs anywhere it is one number in one place, on a page about
       what the assistant is carrying — not a per-switch figure the reader has
       to hold two switches in their head to interpret. -->
  <!-- The assistant door's master switch left this page 14 ก.ย. 2026 (owner:
       "ตอนนี้เป็นระบบทีมเอเจนแล้ว ถ้าเลือกทีมคือส่งงานให้ทีมได้แน่นอน"): the
       chat's team picker holds the one switch, under the team it applies to
       (StationPick.svelte), and a second copy here was the drift §83 names.
       The helpers' switch stays: a ลูกมือ is not a team member, and this is
       its only home. -->
  {#if delegate && !isAgent}
    {@const side = isAgent
      ? { kind: 'agents' as const, reach: delegate.agents, label: 'settings.delegateAgents' as const, on: 'settings.delegateAgentsOn' as const, off: 'settings.delegateAgentsOff' as const }
      : { kind: 'helpers' as const, reach: delegate.helpers, label: 'settings.delegateHelpers' as const, on: 'settings.delegateHelpersOn' as const, off: 'settings.delegateHelpersOff' as const }}
    <div class="settings-card reach-card">
      <div class="set-row">
        <div class="set-txt">
          <div class="t">{t(side.label)}</div>
          <div class="d">
            {t(side.reach.off ? side.off : side.on, { n: side.reach.tokens.toLocaleString() })}
          </div>
        </div>
        <div class="ag-actions">
          <!-- The app's switch (style.css .mswitch), the same one the learning
               page and the MCP shelf wear. It was a `.ctrl` button reading
               "เปิด"/"ปิด" for an afternoon, which is the one control shape this
               settings page does not use for an on/off state: a button labelled
               "ปิด" is read twice, once as "it is off" and once as "press to
               turn it off", and which one it means depends on knowing the
               convention. A switch is the state and the control at once, and
               the owner asked for the standard one (20 ส.ค.).

               A real checkbox inside a label, not a role="switch" button: this
               row is not itself a button, so the checkbox drives the face and
               brings the keyboard and the screen reader with it for free. -->
          <label class="mswitch">
            <input
              type="checkbox" checked={!side.reach.off} disabled={delegateBusy !== ''}
              aria-label={t(side.label)}
              onchange={() => toggleDelegate(side.kind)}
            />
            <span></span>
          </label>
        </div>
      </div>
    </div>
  {/if}

  <!-- Drawn once there is enough to search. A box over three rows is furniture
       that explains nothing; over thirty it is the only way back to the one you
       meant. -->
  {#if rows.mine.length + rows.builtin.length > 6}
    <label class="ag-search">
      <Icon name="search" size={13} />
      <input bind:value={agentQuery} placeholder={t('settings.agentSearch')} />
    </label>
  {/if}

  {#if isAgent}
    <!-- Two grids, split by who wrote it — the question this page is actually
         asked. Built-ins are second because a fresh install has only those and
         the interesting list is the one you grow.
         The group heading is a bare label rather than a card wrapping the grid:
         a card around cards is a box around boxes, and the border did nothing
         the gap between the two groups was not already saying. -->
    {#each [{ id: 'mine', rows: rows.mine.filter(matchesQuery), label: t('settings.subagentsMine'), hint: t('settings.teamMineHint') },
            { id: 'builtin', rows: rows.builtin.filter(matchesQuery), label: t('settings.subagentsBuiltin'), hint: t('settings.subagentsBuiltinHint') }] as group (group.id)}
      <!-- Count beside the title, the sentence pushed to the right margin: the
           left edge is the column being scanned, and a hint under the heading
           put prose between every group and its first name. -->
      <div class="group-head">
        <span class="group-title">{group.label}</span>
        <span class="group-count">{group.rows.length}</span>
        <span class="group-hint">{group.hint}</span>
      </div>
      <!-- A deck, not a card holding rows. The card wrapping the list was the
           one boundary the group heading above it was already drawing, and a
           box around boxes is what made the borderless card impossible to read
           the first two times it was tried. -->
      <div class="office-grid">
        {#each group.rows as a (a.name)}{@render agentRow(a)}{/each}
      </div>
      {#if group.rows.length === 0}
        <p class="muted set-sub ag-empty">
          {agentQuery.trim() ? t('settings.agentNoMatches') : t('settings.teamNoneOfMine')}
        </p>
      {/if}
    {/each}
    <p class="muted set-sub">{t('settings.agentsHint')}</p>
  {:else}
    <!-- The helpers are part of the system (owner's call, 2026-08-06): the
         bundled set is the whole set, so there is no create here and "yours"
         cannot exist as a pile. Since 12 ก.ย. 2026 each one opens in the editor
         within limits (model, prompt, look) — a shadow file the owner may
         revert — and stays in its place on this one list. -->
    <div class="group-head">
      <span class="group-title">{t('settings.subagentsBuiltin')}</span>
      <span class="group-count">{rows.builtin.length}</span>
      <span class="group-hint">{t('settings.helpersFixedHint')}</span>
    </div>
    <div class="office-grid">
      {#each rows.builtin as a (a.name)}{@render profileRow(a)}{/each}
    </div>
    <p class="muted set-sub">{t('settings.helpersFoot')}</p>
  {/if}
{/snippet}

<!-- The one head every "this one's page" wears — ผู้ช่วย and โค้ด (the head
     page), a พนักงาน, a ลูกมือ (agentEditorPane). Owner, 14 ก.ย. 2026: "ทั้ง 3
     หน้านี้ต้อง UI มาตรฐานเดียวกัน แค่แต่ละหน้าอาจจะมีเมนูไม่เหมือนกัน". So the
     shape is fixed here and the pages differ only in what the bar above
     carries and which tabs follow: the face at one size with its rank on the
     corner, the name as the page's title with its badge, one line of what it
     does. There is no h2 above it any more — the name IS the title, and the
     heading-plus-subtitle-plus-bar-plus-face stack was what made the top
     feel tight (owner: "อึดอัดไปหน่อยข้างบน"). -->
{#snippet profileHero(tier: 'head' | 'agent' | 'helper', name: string, badge: string, desc: string, head: HeadId | null = null)}
  <div data-guide="settings.head.hero" class="pf-hero" data-tier={tier}>
    <button data-guide="settings.head.rank" type="button" class="pf-face" onclick={pokeHero} title={t('settings.heroPoke')} aria-label={t('settings.heroPoke')}>
      <RankedFace {tier} size={80}>
        {#if tier === 'head' && head}
          <Mascot {...headOptions(head)} pose={heroPose ?? 'idle'} size={80} hop={heroHop} look />
        {:else}
          <AgentMascot name={facePreviewName} {...draftFace} size={80} still={false} pose={heroPose} hop={heroHop} look />
        {/if}
      </RankedFace>
    </button>
    <div class="pf-who">
      <h2 class="pf-name">{name} <span class="badge on">{badge}</span></h2>
      <p class="pf-desc muted" class:empty={!desc}>{desc || t('settings.agentDescriptionPlaceholder')}</p>
    </div>
  </div>
{/snippet}

<!-- One editor for both kinds. Which kind it is holding was decided by the
     door it was opened through (agentEditKind), never by reading the file —
     that is the same rule the storage layer lives by, carried up. -->
{#snippet agentEditorPane()}
  {#if agentEditing !== null}
    <div class="pp-bar pf-bar">
      <button class="ctrl" onclick={closeAgentEditor}><Icon name="arrowLeft" size={14} /> {t('settings.agentBack')}</button>
      <div class="pp-bar-gap"></div>
      {#if !agentEditing.builtin && agentEditing.name}
        <button data-guide="settings.head.delete" class="ctrl ctrl-danger" disabled={agentBusy !== ''} onclick={deleteAgent}>
          {agentEditing.overrides ? t('settings.agentRevert') : t('settings.remove')}
        </button>
      {/if}
      <button data-guide="settings.head.save" class="ctrl ctrl-primary" disabled={agentBusy !== '' || !agentDraftName.trim() || !agentDraftPrompt.trim()} onclick={saveAgent}>
        {agentBusy === 'save' ? t('settings.saving') : t('settings.promptSave')}
      </button>
    </div>

    <!-- The draft, not the file: the name and the line under it follow what
         is typed on ตัวตน, and the face follows อวตาร, so the head of the page
         is what the card will be once saved. A new one is headed by the
         placeholder name until it has one. -->
    {@render profileHero(
      agentEditKind === 'agent' ? 'agent' : 'helper',
      agentDraftName.trim() || t('settings.agentNewName'),
      agentEditKind === 'agent' ? t('rank.agent') : t('rank.helper'),
      agentDraftDescription.trim(),
    )}

    {#if agentEditing.builtin}
      <p class="muted set-sub">{t('settings.agentOverrideNote')}</p>
    {/if}
    {#if agentError}<div class="mset-error">{agentError}</div>{/if}

    {#if agentGallery}
      {@render agentGallerySheet()}
    {:else}
    <!-- Five groups organised as a clean segmented tab bar (sub-tabs)
         instead of a monolithic vertical scroller. Each tab covers one topic:
         identity, brain, reach, knowledge, and starters. -->
    <div class="ag-tabs-bar">
      <div class="seg" role="tablist" aria-label={t('settings.editAgentTitle')}>
        <button
          type="button" role="tab" id="ag-tab-identity" aria-controls="ag-panel-identity"
          aria-selected={agentTab === 'identity'}
          class:on={agentTab === 'identity'} onclick={() => (agentTab = 'identity')}
        >
          <Icon name="userRound" size={14} />
          <span>{t('settings.agentSecIdentity')}</span>
        </button>
        <!-- The look, on its own tab (owner, 12 ก.ย.: "ควรทำหน้าอวตารแยก") —
             for both kinds: a helper is drawn on the delegate card too. -->
        <button
          type="button" role="tab" id="ag-tab-avatar" aria-controls="ag-panel-avatar"
          aria-selected={agentTab === 'avatar'}
          class:on={agentTab === 'avatar'} onclick={() => (agentTab = 'avatar')}
        >
          <Icon name="bot" size={14} />
          <span>{t('settings.agentSecAvatar')}</span>
        </button>
        <button
          type="button" role="tab" id="ag-tab-brain" aria-controls="ag-panel-brain"
          aria-selected={agentTab === 'brain'}
          class:on={agentTab === 'brain'} onclick={() => (agentTab = 'brain')}
        >
          <Icon name="brain" size={14} />
          <span>{t('settings.agentSecBrain')}</span>
        </button>
        {#if agentEditKind === 'agent'}
          <button
            data-guide="settings.head.tab.tools"
            type="button" role="tab" id="ag-tab-reach" aria-controls="ag-panel-reach"
            aria-selected={agentTab === 'reach'}
            class:on={agentTab === 'reach'} onclick={() => (agentTab = 'reach')}
          >
            <Icon name="plug" size={14} />
            <span>{t('settings.mainSecMcp')}</span>
            {#if unmetAgentNeeds > 0}
              <span class="ag-count ag-count-warn">{unmetAgentNeeds}</span>
            {/if}
          </button>
          <!-- Drawn for a NEW agent too. Its skills, memory and opening live
               in a folder that exists only once it is saved, so the two tabs
               were hidden until then — and a strip missing two tabs read as
               the page being unfinished (owner, 13 ก.ย. 2026: "ทำไมมีไม่ครบ").
               The tabs stay; the panels say what to do first. -->
            <button
              type="button" role="tab" id="ag-tab-knowledge" aria-controls="ag-panel-knowledge"
              aria-selected={agentTab === 'knowledge'}
              class:on={agentTab === 'knowledge'} onclick={() => (agentTab = 'knowledge')}
            >
              <Icon name="puzzle" size={14} />
              <span>{t('settings.mainSecSkills')}</span>
            </button>
            <button
              type="button" role="tab" id="ag-tab-opening" aria-controls="ag-panel-opening"
              aria-selected={agentTab === 'opening'}
              class:on={agentTab === 'opening'} onclick={() => (agentTab = 'opening')}
            >
              <Icon name="messageSquare" size={14} />
              <span>{t('settings.agentSecOpening')}</span>
            </button>
        {/if}
        <!-- ความจำ: its own tab for every kind (owner, 14 ก.ย. 2026: "เอเจน
             เฉพาะทางควรจะมีหน้าต่างความจำของตัวเอง มันหายไปไหน"). It sat under
             สกิล, which a ลูกมือ never even had a tab for. Only a saved profile
             has a file to show; a new one gets the tab and a save-first card. -->
        <button
          type="button" role="tab" id="ag-tab-memory" aria-controls="ag-panel-memory"
          aria-selected={agentTab === 'memory'}
          class:on={agentTab === 'memory'} onclick={() => (agentTab = 'memory')}
        >
          <Icon name="brain" size={14} />
          <span>{t('settings.mainSecMemory')}</span>
        </button>
      </div>
    </div>

    <!-- ── ตัวตน ── -->
    <div role="tabpanel" id="ag-panel-identity" aria-labelledby="ag-tab-identity"
      class="ag-tab-panel" class:on={agentTab === 'identity' || (agentEditKind !== 'agent' && agentTab !== 'brain' && agentTab !== 'avatar' && agentTab !== 'memory')}>
      <div class="settings-card">
        <div class="card-form pp-edit">
          <label class="pp-field">
            <span class="eyebrow">{t('settings.agentName')}</span>
            <input class="ctrl" bind:value={agentDraftName} placeholder="backend" disabled={agentEditing.name !== ''} />
          </label>
          <label class="pp-field">
            <span class="eyebrow">{t('settings.agentDescription')}</span>
            <input class="ctrl" bind:value={agentDraftDescription} placeholder={t('settings.agentDescriptionPlaceholder')} />
          </label>
          <!-- A div rather than the `label` every other field uses: the toggle is
               a button, and a button inside a label puts the caret in the textarea
               on every click of it. -->
          <div class="pp-field">
            <div class="ag-bodyhead">
              <span class="eyebrow">{t('settings.agentBody')}</span>
              {#if agentBodyLong}
                <button type="button" class="ag-bodymore" onclick={() => (agentBodyOpen = !agentBodyOpen)}>
                  {agentBodyOpen ? t('settings.agentBodyLess') : t('settings.agentBodyMore', { n: agentBodyLines })}
                </button>
              {/if}
            </div>
            <div class="ag-bodywrap" class:collapsed={agentBodyLong && !agentBodyOpen}>
              <textarea
                class="ctrl ag-body" bind:value={agentDraftPrompt} spellcheck="false"
                placeholder={t('settings.agentStarter')}
                use:autogrow={agentDraftPrompt}
                onfocus={() => (agentBodyOpen = true)}
              ></textarea>
            </div>
            <span class="d muted">{t('settings.agentBodyHint')}</span>
            <!-- Where the words can come from besides the keyboard: a file, a
                 link, a template — house controls (.ctrl), one row. The
                 select resets itself so the same template can be picked
                 twice; it is a menu, not a value. -->
            <div class="ag-fill">
              <button type="button" class="ctrl" onclick={fillFromFile} disabled={agentFillBusy}><Icon name="folderOpen" size={13} /> {t('settings.agentFillFile')}</button>
              <button type="button" class="ctrl ag-fill-tpl" onclick={() => openGallery(false)} disabled={agentFillBusy}><Icon name="layoutList" size={13} /> {t('settings.agentFillTemplate')}</button>
              <input class="ctrl key-input ag-fill-link" type="text" bind:value={agentFillLink} spellcheck="false"
                placeholder={t('settings.agentFillLinkPlaceholder')} disabled={agentFillBusy}
                onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); fillFromLink() } }} />
              <button type="button" class="ctrl" onclick={fillFromLink} disabled={agentFillBusy || !agentFillLink.trim()}><Icon name="download" size={13} /> {t('settings.agentFillFetch')}</button>
            </div>
            {#if agentFillError}<div class="mset-error">{agentFillError}</div>{/if}
          </div>
        </div>
      </div>
    </div>

    <!-- ── อวตาร ── the mascot this agent is, everywhere in the app. Its own
         tab (owner, 12 ก.ย. 2026: "ควรทำหน้าอวตารแยก เพิ่มนะครับ และทำให้มันใช้
         อวตารที่บันทึกไว้ได้"): the figure the way the avatar page shows the
         assistant's, the six persona slots as cards you press, and under them
         the badge and the four identity dials. Same draft, same save — a tab
         is where it is read, not a second place it is kept. -->
    <div role="tabpanel" id="ag-panel-avatar" aria-labelledby="ag-tab-avatar"
      class="ag-tab-panel" class:on={agentTab === 'avatar'}>
      <div class="settings-card">
        <div class="card-form pp-edit">
          <div class="pp-field">
            <div class="ag-face-header">
              <span class="eyebrow">{t('settings.agentLook')}</span>
              <button
                type="button"
                class="ctrl tiny"
                onclick={() => (showFaceContexts = !showFaceContexts)}
                aria-expanded={showFaceContexts}
                style="display:inline-flex; align-items:center; gap:5px; cursor:pointer;"
              >
                <Icon name="eye" size={13} />
                <span>{showFaceContexts ? t('settings.agentLookHideContexts') : t('settings.agentLookShowContexts')}</span>
              </button>
            </div>
            <!-- The one being faced: big, breathing. Everything else on this
                 tab is still. -->
            <div class="ag-face ag-avatar-stage">
              <!-- The rank on the stage too (owner, 14 ก.ย.: "ตอนสร้างเอเจน …
                   เลือกอวตาร มียศบอกเลยนะคืออะไร"): the emblem on the face
                   as it will be everywhere, and the word beside the name
                   here, where there is room to say it. -->
              <RankedFace tier={agentEditKind === 'agent' ? 'agent' : 'helper'} size={168}>
                <AgentMascot name={facePreviewName} {...draftFace} size={168} still={false} />
              </RankedFace>
              <div class="ag-face-say">
                <b class="ag-avatar-name">{facePreviewName} <RankPip tier={agentEditKind === 'agent' ? 'agent' : 'helper'} size="md" /></b>
                <span class="d muted">{faceIsAuto ? t('settings.agentLookAutoHint') : t('settings.agentLookHint')}</span>
                {#if !faceIsAuto}
                  <button type="button" class="ag-face-reset" onclick={resetFace}>
                    {t('settings.agentLookReset')}
                  </button>
                {/if}
              </div>
            </div>

            {#if showFaceContexts}
              <div class="ag-contexts-panel">
                <div class="ag-contexts-head">
                  <span class="ag-contexts-title">{t('settings.agentLookContextsTitle')}</span>
                  <span class="d muted" style="font-size:var(--fs-2xs);">{t('settings.agentLookContextsDesc')}</span>
                </div>

                <div class="ag-contexts-grid">
                  <!-- 1. Office & Team (38px) -->
                  <div class="ag-context-card">
                    <div class="ag-context-label">
                      <Icon name="userRound" size={14} />
                      <span>{t('settings.agentLookContextOffice')}</span>
                    </div>
                    <div class="ag-context-sample">
                      <RankedFace tier={agentEditKind === 'agent' ? 'agent' : 'helper'} size={38}><AgentMascot name={facePreviewName} {...draftFace} size={38} off={previewFaceOff} /></RankedFace>
                      <div style="display:flex; flex-direction:column; gap:2px; min-width:0;">
                        <span style="font-weight:600; font-size:var(--fs-sm); color:var(--text-primary);">{facePreviewName}</span>
                        <span class="d muted" style="font-size:var(--fs-2xs);">{previewFaceOff ? t('settings.agentLookStateOff') : t('settings.agentLookStateIdle')}</span>
                      </div>
                    </div>
                    <div class="ag-context-states">
                      <button
                        type="button"
                        class="ag-context-btn"
                        class:on={!previewFaceOff}
                        onclick={() => (previewFaceOff = false)}
                      >
                        {t('settings.agentLookStateIdle')}
                      </button>
                      <button
                        type="button"
                        class="ag-context-btn"
                        class:on={previewFaceOff}
                        onclick={() => (previewFaceOff = true)}
                      >
                        {t('settings.agentLookStateOff')}
                      </button>
                    </div>
                  </div>

                  <!-- 2. Chat & Tasks (34px) -->
                  <div class="ag-context-card">
                    <div class="ag-context-label">
                      <Icon name="messageSquare" size={14} />
                      <span>{t('settings.agentLookContextChat')}</span>
                    </div>
                    <div class="ag-context-sample">
                      <AgentMascot
                        name={facePreviewName}
                        {...draftFace}
                        size={34}
                        state={previewFaceState === 'idle' ? '' : previewFaceState}
                      />
                      <div style="display:flex; flex-direction:column; gap:2px; min-width:0;">
                        <span style="font-weight:600; font-size:var(--fs-sm); color:var(--text-primary);">{facePreviewName}</span>
                        <span class="d muted" style="font-size:var(--fs-2xs);">
                          {#if previewFaceState === 'idle'}
                            {t('settings.agentLookStateIdle')}
                          {:else if previewFaceState === 'think'}
                            {t('settings.agentLookStateThinking')}
                          {:else if previewFaceState === 'work'}
                            {t('settings.agentLookStateWorking')}
                          {:else if previewFaceState === 'done'}
                            {t('settings.agentLookStateDone')}
                          {:else if previewFaceState === 'err'}
                            {t('settings.agentLookStateError')}
                          {/if}
                        </span>
                      </div>
                    </div>
                    <div class="ag-context-states">
                      {#each [
                        { id: 'idle', label: t('settings.agentLookStateIdle') },
                        { id: 'think', label: t('settings.agentLookStateThinking') },
                        { id: 'work', label: t('settings.agentLookStateWorking') },
                        { id: 'done', label: t('settings.agentLookStateDone') },
                        { id: 'err', label: t('settings.agentLookStateError') },
                      ] as st (st.id)}
                        <button
                          type="button"
                          class="ag-context-btn"
                          class:on={previewFaceState === st.id}
                          onclick={() => (previewFaceState = st.id as any)}
                        >
                          {st.label}
                        </button>
                      {/each}
                    </div>
                  </div>

                  <!-- 3. Composer & Mention (20px) -->
                  <div class="ag-context-card">
                    <div class="ag-context-label">
                      <Icon name="terminal" size={14} />
                      <span>{t('settings.agentLookContextComposer')}</span>
                    </div>
                    <div class="ag-context-sample" style="align-items:center;">
                      <div class="ag-context-chip-composer">
                        <AgentMascot name={facePreviewName} {...draftFace} size={20} />
                        <span>@{facePreviewName}</span>
                      </div>
                    </div>
                    <span class="d muted" style="font-size:var(--fs-2xs);">{t('settings.agentLookComposerHint')}</span>
                  </div>
                </div>
              </div>
            {/if}
          </div>

          <!-- The saved personas, as the avatar page draws them: a card per
               look, worn by THIS agent (its own badge on the ears), and ใช้
               puts all four dials on the draft at once. With none saved, one
               dashed card says so — where a look would go is half the
               invitation to make one. -->
          <div class="pp-field">
            <span class="eyebrow">{t('settings.agentPersonas')}</span>
            <div class="ag-slots">
              {#each personas.slots as slot, i (i)}
                <div class="ag-slot" class:worn={wearsPersona(i)}>
                  <AgentMascot name={facePreviewName} icon={agentDraftIcon || undefined} shell={slot.shell} top={slot.top} face={slot.face} accent={slot.accent} size={56} />
                  <div class="ag-slot-meta">
                    <b>{t('settings.agentPersonaUse', { n: i + 1 })}</b>
                    <span>{wearsPersona(i) ? avatarText(i18n.locale).worn : ''}</span>
                  </div>
                  <button type="button" class="ctrl tiny pri" disabled={wearsPersona(i)} aria-label={`${t('settings.agentPersonaUse', { n: i + 1 })} — ${avatarText(i18n.locale).use}`} onclick={() => wearPersona(i)}>{avatarText(i18n.locale).use}</button>
                </div>
              {:else}
                <div class="ag-slot empty">
                  <div class="ag-slot-empty"><Icon name="bot" size={18} /></div>
                  <div class="ag-slot-meta"><span>{avatarText(i18n.locale).personaEmpty}</span></div>
                </div>
              {/each}
            </div>
            <span class="d muted">
              {t('settings.agentPersonasWhere')}
              <button type="button" class="ag-face-reset" onclick={() => openSection('avatar')}>{avatarText(i18n.locale).title}</button>
            </span>
          </div>

          <!-- The badge on the ears. Glyphs rather than robots: forty heads
               differing by one small mark on the ear is a wall of near-identical
               tiles, and the mark itself is what the eye can tell apart at this
               size. The preview above is where the outcome is read. Every glyph
               here is one the app's buttons already wear (ICONS), which is the
               point of putting it on an ear: a person who sees `search` there
               has seen it on the tool that does it. -->
          <div class="pp-field">
            <span class="eyebrow">{t('settings.agentBadge')}</span>
            <div class="ag-icons">
              <button type="button" class="ag-icon" class:on={agentDraftIcon === ''}
                title={t('settings.agentIconAuto')} aria-label={t('settings.agentIconAuto')}
                onclick={() => (agentDraftIcon = '')}>
                <Icon name="sparkles" size={16} />
              </button>
              {#each AGENT_BADGES as name (name)}
                <button type="button" class="ag-icon" class:on={agentDraftIcon === name}
                  title={name} aria-label={name} onclick={() => (agentDraftIcon = name)}>
                  <Icon name={name} size={16} />
                </button>
              {/each}
            </div>
            <span class="d muted">{agentDraftIcon === '' ? t('settings.agentIconAutoHint') : agentDraftIcon}</span>
          </div>

          <!-- The four identity dials ARE drawn as robots, and for the opposite
               reason to the badge row: the difference between two shells is the
               whole body, so a swatch of the part alone would be a shape nobody
               recognises. Each cell is the whole outcome, wearing whatever was
               picked in the other rows, so no cell here is a guess — and every
               cell is still: twenty of these breathing together is a page that
               stutters (MASCOT.md §3.7). -->
          <div class="pp-field">
            <span class="eyebrow">{t('settings.agentShell')}</span>
            <div class="ag-parts">
              <button type="button" class="ag-part" class:on={agentDraftShell === ''}
                title={t('settings.agentIconAuto')} aria-label={t('settings.agentIconAuto')}
                onclick={() => (agentDraftShell = '')}>
                <AgentMascot name={facePreviewName} {...draftFace} shell={undefined} size={44} />
              </button>
              {#each SHELL as sh (sh.id)}
                <button type="button" class="ag-part" class:on={agentDraftShell === sh.id}
                  title={sh.label} aria-label={sh.label} onclick={() => (agentDraftShell = sh.id)}>
                  <AgentMascot name={facePreviewName} {...draftFace} shell={sh.id} size={44} />
                </button>
              {/each}
            </div>
          </div>

          <!-- Colour, and the one row where the cell could have been a plain
               swatch. It is a robot for the same reason the rows around it are:
               the accent moves the cap, the ears, the soles and the light on the
               screen, all at once, and a square of one colour would be a promise
               about a quarter of what changes. The first cell is the colour the
               name gives — every agent's, before anyone chose. -->
          <div class="pp-field">
            <span class="eyebrow">{t('settings.agentAccent')}</span>
            <div class="ag-parts">
              <button type="button" class="ag-part" class:on={agentDraftAccent === '' && agentDraftHue === ''}
                title={t('settings.agentIconAuto')} aria-label={t('settings.agentIconAuto')}
                onclick={() => { agentDraftAccent = ''; agentDraftHue = '' }}>
                <AgentMascot name={facePreviewName} {...draftFace} accent={undefined} hue={undefined} size={44} />
              </button>
              {#each ACCENT as a (a.id)}
                <button type="button" class="ag-part" class:on={agentDraftAccent === a.id && agentDraftHue === ''}
                  title={a.label} aria-label={a.label} onclick={() => pickAccent(a.id)}>
                  <AgentMascot name={facePreviewName} {...draftFace} accent={a.id} hue={undefined} size={44} />
                </button>
              {/each}
            </div>
            {#if agentDraftHue}
              <span class="d muted">{t('settings.agentHueKept', { deg: agentDraftHue })}</span>
            {/if}
          </div>

          <div class="pp-field">
            <span class="eyebrow">{t('settings.agentTop')}</span>
            <div class="ag-parts">
              <button type="button" class="ag-part" class:on={agentDraftTop === ''}
                title={t('settings.agentIconAuto')} aria-label={t('settings.agentIconAuto')}
                onclick={() => (agentDraftTop = '')}>
                <AgentMascot name={facePreviewName} {...draftFace} top={undefined} size={44} />
              </button>
              {#each TOP as tp (tp.id)}
                <button type="button" class="ag-part" class:on={agentDraftTop === tp.id}
                  title={tp.label} aria-label={tp.label} onclick={() => (agentDraftTop = tp.id)}>
                  <AgentMascot name={facePreviewName} {...draftFace} top={tp.id} size={44} />
                </button>
              {/each}
            </div>
          </div>

          <!-- The resting face: only the identity rows of FACE. The others are
               what a pose lights (thinking, happy, the wink) and a roster tile
               never chooses those — presence does. -->
          <div class="pp-field">
            <span class="eyebrow">{t('settings.agentRestFace')}</span>
            <div class="ag-parts">
              <button type="button" class="ag-part" class:on={agentDraftFace === ''}
                title={t('settings.agentIconAuto')} aria-label={t('settings.agentIconAuto')}
                onclick={() => (agentDraftFace = '')}>
                <AgentMascot name={facePreviewName} {...draftFace} face={undefined} size={44} />
              </button>
              {#each FACE.filter((f) => f.identity) as f (f.id)}
                <button type="button" class="ag-part" class:on={agentDraftFace === f.id}
                  title={f.label} aria-label={f.label} onclick={() => (agentDraftFace = f.id)}>
                  <AgentMascot name={facePreviewName} {...draftFace} face={f.id} size={44} />
                </button>
              {/each}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── สมอง ── which model answers, and how long it may go on for. -->
    <div role="tabpanel" id="ag-panel-brain" aria-labelledby="ag-tab-brain"
      class="ag-tab-panel" class:on={agentTab === 'brain'}>
      <div class="settings-card">
        <div class="card-form pp-edit">
          <!-- The three that decide which brain answers, in the shape the
               chat's own picker uses (.mm-row, style.css): label left, control
               right, one under another. Owner's call, 13 ก.ย.: "แสดงเป็น
               ดรอบแบบนี้เลยดีกว่า … เหมือนหน้าแชททำอ่ะ" — they are one decision
               read together ("this agent thinks HERE, on THIS, THIS deep"),
               and three stacked full-width fields with a paragraph under each
               made it read as three unrelated settings.

               The provider rows are การตั้งค่าโมเดล's own — enabledRows, the
               catalogue's providers the user switched on, in the catalogue's
               order — not the raw enabled list, which can still name a
               provider the catalogue dropped (owner: "ควรอิง Providers ที่เปิด
               ไว้หน้าตั้งค่าโมเดล"). One that is off has no key to sign with.
               Whatever the file names is offered too, so a pin to a provider
               since switched off still reads as itself. -->
          <div class="pp-field ag-brain">
            <div class="mm-row">
              <span class="lbl">{t('settings.agentProviderPick')}</span>
              <select class="ctrl" value={agentDraftProvider} onchange={(e) => pickAgentProvider(e.currentTarget.value)}>
                <option value="">{t('settings.agentProviderInherit')}</option>
                {#each enabledRows as p (p.name)}<option value={p.name}>{p.name}</option>{/each}
                {#if agentDraftProvider && !enabledRows.some((p) => p.name === agentDraftProvider)}
                  <option value={agentDraftProvider}>{agentDraftProvider}</option>
                {/if}
              </select>
            </div>
            <div class="mm-row">
              <span class="lbl">{t('settings.agentModelPick')}</span>
              <select class="ctrl" bind:value={agentDraftModel}>
                <option value="">{agentDraftProvider ? t('settings.agentModelProviderDefault') : t('settings.agentModelInherit')}</option>
                {#each agentModels as m}<option value={m}>{m}</option>{/each}
                {#if agentDraftModel && !agentModels.includes(agentDraftModel)}
                  <option value={agentDraftModel}>{agentDraftModel}</option>
                {/if}
              </select>
            </div>
            <!-- The depth dial belongs to the model above, so the row exists
                 only when that model can actually think — the same test the
                 chat's menu makes on this row, and for the same reason: a
                 dropdown for a setting the model does not have is a control
                 that reads as broken. It sat there greyed out on every local
                 runtime until now (owner: "ตัวไหนคิดไม่ได้ก็ซ่อน").

                 The second half of the test is why a file is never silently
                 eaten: a profile carrying `think:` keeps its row even on a
                 model with no dial, so switching the model pin to a local
                 runtime to look at something does not drop a line the author
                 wrote. The warning under the group then says it will not be
                 honoured there. -->
            {#if agentThinkLevels.length > 0 || agentDraftThink}
              <div class="mm-row">
                <span class="lbl">{t('settings.agentThinkPick')}</span>
                <select class="ctrl" bind:value={agentDraftThink}>
                  <option value="">{t('settings.agentThinkInherit')}</option>
                  {#each agentThinkLevels as lvl (lvl)}<option value={lvl}>{lvl}</option>{/each}
                  {#if agentDraftThink && !agentThinkLevels.includes(agentDraftThink)}
                    <option value={agentDraftThink}>{agentDraftThink}</option>
                  {/if}
                </select>
              </div>
            {/if}
            <!-- One line for the group, not one under each row. What all three
                 share is the only thing a reader has to be told — blank means
                 the chat's — and saying it three times is what turned the
                 panel into a wall. The stale warning replaces it only when
                 there is something actually wrong to report. -->
            <span class="d muted">
              {agentDraftThink && agentThinkLevels.length === 0
                ? t('settings.agentThinkStale')
                : t('settings.agentBrainHint')}
            </span>
          </div>

          <div class="pp-field">
            <span class="eyebrow">{t('settings.agentStepsField')}</span>
            <div class="ag-steprow">
              <input
                class="ctrl ag-steps" bind:value={agentDraftSteps} inputmode="numeric" placeholder="40"
                disabled={agentStepsUnlimited}
                aria-label={t('settings.agentStepsField')}
              />
              <label class="ag-check">
                <span class="mswitch">
                  <input type="checkbox" checked={agentStepsUnlimited} onchange={toggleStepsUnlimited} />
                  <span></span>
                </span>
                {t('settings.agentStepsUnlimited')}
              </label>
            </div>
            <span class="d muted">
              {agentStepsUnlimited ? t('settings.agentStepsUnlimitedWarn') : t('settings.agentStepsFieldHint')}
            </span>
          </div>
        </div>
      </div>
    </div>

    {#if agentEditKind === 'agent'}
      <!-- ── MCP สำหรับเอเจน ── the servers placed on this agent, and its needs. -->
      <div role="tabpanel" id="ag-panel-reach" aria-labelledby="ag-tab-reach"
        class="ag-tab-panel" class:on={agentTab === 'reach'}>
        <!-- No โต๊ะที่สังกัด card (owner, 13 ก.ย. 2026: "เอาโต๊ะที่สังกัดออกเลย
             ไม่ต้องแสดง"). The desk is a ceiling the engine applies, not a
             thing this form lets anyone change; a read-only row naming a
             code-side id told the reader about the file, not about the agent. -->
        {@render agentMCPBox()}
        {@render agentNeedsBox()}
      </div>

      <!-- ── สกิลเฉพาะสำหรับเอเจน ── this agent's own skills and memory. -->
      <div role="tabpanel" id="ag-panel-knowledge" aria-labelledby="ag-tab-knowledge"
        class="ag-tab-panel" class:on={agentTab === 'knowledge'}>
        {@render agentSkillsBox()}
      </div>

      <!-- ── เปิดบทสนทนา ── conversation starter cards. -->
      <div role="tabpanel" id="ag-panel-opening" aria-labelledby="ag-tab-opening"
        class="ag-tab-panel" class:on={agentTab === 'opening'}>
        {#if agentEditing.name}
          {@render agentStartersBox()}
        {:else}
          {@render saveFirstCard('messageSquare', t('settings.agentSaveFirstOpening'))}
        {/if}
      </div>
    {/if}

    <!-- ── ความจำ ── this profile's own file, the head page's block. Outside
         the agent-only block above: a ลูกมือ has the tab (fddd0b15), so it has
         the panel. -->
    <div role="tabpanel" id="ag-panel-memory" aria-labelledby="ag-tab-memory"
      class="ag-tab-panel" class:on={agentTab === 'memory'}>
      {#if agentEditing.name}
        {@render agentMemoryBox()}
      {:else}
        {@render saveFirstCard('brain', t('settings.agentMemorySaveFirst'))}
      {/if}
    </div>
    {/if}
  {/if}
{/snippet}

<!-- The template sheet (§284). Shapes first — they are the ones a person
     fills rather than edits, and the smallest — then the roles in their
     groups. The search box narrows the roles only; four shapes need no
     search. The way out is one button whose label says what leaving does
     (see agentGalleryFirst), and the tick beside it is the only thing on
     this sheet that outlives the press. -->
{#snippet agentGallerySheet()}
  {@const q = agentGalleryQuery.trim()}
  <div class="settings-card ag-gallery" data-testid="agent-gallery">
    <div class="card-form">
      <div class="ag-gallery-head">
        <div>
          <div class="ag-gallery-title">{t('settings.galleryTitle')}</div>
          <div class="d muted">{t('settings.galleryLead')}</div>
        </div>
        <input class="ctrl key-input ag-gallery-search" type="search" bind:value={agentGalleryQuery}
          placeholder={t('settings.gallerySearch')} aria-label={t('settings.gallerySearch')} spellcheck="false" />
      </div>
      {#if agentFillError}<div class="mset-error">{agentFillError}</div>{/if}

      {#if !q}
        <div class="ag-gallery-group">
          <h4 class="ag-gallery-group-title">{t('settings.galleryGroupShape')}</h4>
          <div class="ag-gallery-grid shapes">
            {#each AGENT_TEMPLATES as tp (tp.id)}
              <button type="button" class="ag-gallery-card shape" onclick={() => pickShape(tp.id)}>
                <span class="ag-gallery-card-title">{t(tp.title)}</span>
              </button>
            {/each}
          </div>
        </div>
      {/if}

      {#each GALLERY_GROUPS as g (g.id)}
        {@const roles = GALLERY_ROLES.filter((r) => r.group === g.id && galleryMatches(r, q))}
        {#if roles.length}
          <div class="ag-gallery-group">
            <h4 class="ag-gallery-group-title">{t(g.title)}</h4>
            <div class="ag-gallery-grid">
              {#each roles as r (r.id)}
                <button type="button" class="ag-gallery-card" class:busy={agentGalleryBusy === r.id}
                  disabled={agentGalleryBusy !== ''} onclick={() => pickRole(r.id)}>
                  <span class="ag-gallery-card-title">{r.title}</span>
                  <span class="ag-gallery-card-desc">{r.desc}</span>
                  <span class="ag-gallery-card-meta">{t('settings.galleryWords', { n: (Math.round(r.words / 100) * 100).toLocaleString('en') })}</span>
                </button>
              {/each}
            </div>
          </div>
        {/if}
      {/each}
      {#if q && !GALLERY_ROLES.some((r) => galleryMatches(r, q))}
        <div class="d muted">{t('settings.galleryNone')}</div>
      {/if}

      <div class="ag-gallery-foot">
        <button type="button" class="ctrl" onclick={() => (agentGallery = false)}>
          {agentGalleryFirst ? t('settings.galleryBlank') : t('settings.galleryBack')}
        </button>
        <label class="ag-gallery-skip">
          <input type="checkbox" checked={agentGallerySkip} onchange={(e) => setGallerySkip(e.currentTarget.checked)} />
          <span>{t('settings.gallerySkip')}</span>
        </label>
        <span class="ag-gallery-foot-gap"></span>
        <span class="d muted">{t('settings.galleryEnglishNote')} · {t('settings.galleryCredit')}</span>
      </div>
    </div>
  </div>
{/snippet}

<!-- The servers pointed at this one agent. A box of its own, beside เครื่องมือ
     rather than inside it, because it is the opposite operation: `for:` on a
     server ADDS, skipping this profile's allow-list and reaching past the
     desk's ceiling (internal/subagent/store.go). Reading the two as one list
     was the complaint, and the complaint was a true statement about the code.

     On 12 ก.ย. 2026 this box stopped editing — it was the third editor of the
     same `for:` list (with ตั้งค่า › MCP and the room), and the owner's word
     for that was ซ้ำซ้อน. On 13 ก.ย. it came back as a list of every live
     server with a switch (owner: "ทำไมมันไม่แสดง MCP" of a box that listed
     only what was already placed): the switch and the room's picker are one
     switch on one call (SetMCPServerTargets), so it is one editor drawn in
     two places, not two stores. The door is for what this box cannot do —
     adding, testing, signing in to a server — and opens the room's own page.
     A NEW agent, with no file to place anything on yet, ticks here and Save
     places (placeNewAgentMCP). -->
{#snippet agentMCPBox()}
  {@const saved = !!agentEditing?.name}
  <div class="settings-card">
    <div class="set-row">
      <span class="ag-rowicon"><Icon name="plug" size={15} /></span>
      <div class="set-txt">
        <div class="t">{t('settings.agentMCPTitle')} {#if mcpLoaded}<span class="ag-count">{saved ? agentServerCount : agentNewMCP.length}</span>{/if}</div>
        <div class="d">{t('settings.agentMCPHint')}</div>
      </div>
      <!-- Where a server is added, tested and signed in: the room's own page.
           Placing is done here; the door is for everything else. -->
      <button class="ctrl" onclick={() => openCapabilityAt('mine')}>{t('capability.navMine')} <Icon name="arrowRight" size={13} /></button>
    </div>
    {#if mcpError}<div class="mset-error">{mcpError}</div>{/if}
    {#if !mcpLoaded}
      {@render waitRow()}
    {:else}
      {#each liveServers as srv (srv.name)}
        {@const on = saved ? isOnAgent(srv) : agentNewMCP.includes(srv.name)}
        <label class="set-row ag-reachrow" class:on>
          <McpMark name={srv.name} size={26} />
          <div class="set-txt">
            <div class="t">{srv.name}</div>
            <div class="d">{srv.tools > 0 ? t('settings.agentMCPTools', { n: srv.tools }) : (srv.url || (srv.command ?? []).join(' '))}</div>
          </div>
          <span class="mswitch">
            <input type="checkbox" checked={on} disabled={saved && (mcpBusy !== '' || !agentMCPId)} aria-label={srv.name}
              onchange={() => (saved ? toggleAgentMCP(srv) : toggleNewMCP(srv.name))} />
            <span></span>
          </span>
        </label>
      {/each}
      <div class="set-row"><div class="set-txt"><div class="d">
        {liveServers.length === 0 ? t('settings.agentMCPNoneInSystem') : (saved ? t('settings.agentMCPLiveHint') : t('settings.agentMCPNewHint'))}
      </div></div></div>
    {/if}
  </div>
{/snippet}

<!-- What the agent said it cannot work without, and where each of those stands.
     The engine has computed this since needs.go was written and only ever
     folded it into the agent's own prompt — so an agent that could not work
     said so in the chat, while the page you fix it on showed nothing.

     One row per *requirement*, not per thing. `needs: connection:n8n |
     connection:windmill` is one requirement — an automation engine — and either
     answers it. Drawn as two flat rows it read as "n8n is required" beside a
     second demand for a product the user had deliberately not installed. -->
{#snippet agentNeedsBox()}
  {#if agentKeptNeeds.length > 0}
    {@const unmet = agentNeeds.filter((r) => !r.met).length}
    <div class="settings-card">
      <div class="card-form">
        <div class="eyebrow">
          {t('settings.agentNeedsTitle')}
          {#if unmet > 0}<span class="ag-count ag-count-warn">{unmet}</span>{/if}
        </div>
        <div class="d muted">{t('settings.agentNeedsHint')}</div>
        {#if mcpError}<div class="mset-error">{mcpError}</div>{/if}
      </div>
      {#each agentNeeds as req (req.entry)}
        {@const options = req.options ?? []}
        <div class="ag-need" class:met={req.met}>
          <!-- The requirement's own line. With one option it is that option's
               name; with more it says the choice out loud, because "either of
               these" is the fact the flat list was losing. -->
          <div class="ag-need-head">
            <span class="ag-rowicon" class:ag-rowicon-warn={!req.met}>
              <Icon name={req.met ? 'check' : (options[0]?.kind === 'connection' ? 'globe' : 'plug')} size={15} />
            </span>
            <div class="set-txt">
              <div class="t">
                {options.length > 1
                  ? options.map((o) => o.label).join(' / ')
                  : (options[0]?.label ?? req.entry)}
              </div>
              <div class="d">
                {#if req.met}{t('settings.agentNeedMet')}
                {:else if options.length > 1}{t('settings.agentNeedEitherOf')}
                {:else}{t(`settings.agentNeedReason_${options[0]?.reason ?? 'unknown'}` as TKey)}{/if}
              </div>
            </div>
            {#if !req.met && options.length === 1}
              {@render needDoor(options[0])}
            {/if}
          </div>

          <!-- With a choice, each way of answering it gets its own line and its
               own state — "อันไหนเปิดอยู่ก็บอกว่าเปิด". The door beside each
               goes where THAT one is switched on, which a single shared button
               could never do. -->
          {#if options.length > 1}
            {#each options as o (o.kind + ':' + o.id)}
              <div class="ag-need-opt">
                <span class="ag-need-dot" class:on={!o.reason}></span>
                <div class="set-txt">
                  <div class="t">{o.label}</div>
                  <div class="d">
                    {o.reason ? t(`settings.agentNeedReason_${o.reason}` as TKey) : t('settings.agentNeedOptionOn')}
                  </div>
                </div>
                {#if o.reason}{@render needDoor(o)}{/if}
              </div>
            {/each}
          {/if}
        </div>
      {/each}
    </div>
  {/if}
{/snippet}

<!-- Where one unmet option is actually switched on. A server this agent is not
     placed on is fixed in the MCP box a few rows up, so that one says so
     instead of sending the user to a page they just came from. -->
{#snippet needDoor(o: subagent.Need)}
  {#if o.kind === 'mcp' && o.reason === 'unplaced'}
    <span class="d muted ag-need-here">{t('settings.agentNeedFixHere')}</span>
  {:else if o.kind === 'mcp' && o.reason === 'missing' && presetFor(o.id)}
    <!-- The server this agent was written against, installed and placed from
         here. Anywhere else and the user has to know which of seven presets
         belongs to which agent, which is the thing they cannot know. -->
    <button class="ctrl ctrl-primary ctrl-icon" disabled={mcpBusy !== ''} onclick={() => installNeeded(o)}>
      {mcpBusy === 'need:' + o.id ? t('settings.agentNeedInstalling') : t('settings.agentNeedInstall')}
    </button>
  {:else}
    <button class="ctrl" onclick={() => (o.kind === 'connection' ? openCapabilityAt('connections') : setActiveView('capability'))}>
      {o.kind === 'connection' ? t('settings.agentNeedConnect') : t('settings.agentNeedServer')}
      <Icon name="arrowRight" size={13} />
    </button>
  {/if}
{/snippet}

<!-- Its own shelf. Reads and does not edit, because a skill is a folder: the
     honest control is the one that opens it. The list is drawn here (13 ก.ย.,
     "ทำไมมันไม่แสดง"); adding, removing and copying in from the shared shelf
     stay on ห้องความสามารถ › ตั้งค่าสกิลสำหรับเอเจนเฉพาะ, which the door opens
     on this agent's sheet — one editor, as §253 asked. -->
{#snippet agentSkillsBox()}
  {@const saved = !!agentEditing?.name}
  <div class="settings-card">
    <div class="set-row">
      <span class="ag-rowicon"><Icon name="puzzle" size={15} /></span>
      <div class="set-txt">
        <div class="t">{t('settings.agentSkillsTitle')} {#if saved ? agentSkillsReady : shelfLoaded}<span class="ag-count">{saved ? agentSkills.length : agentNewSkills.length}</span>{/if}</div>
        <div class="d">{t('settings.agentSkillsHint')}</div>
      </div>
      {#if saved}
        <button class="ctrl ctrl-icon" title={t('settings.agentSkillsOpenFolder')} aria-label={t('settings.agentSkillsOpenFolder')} onclick={() => OpenAgentSkillsFolder(agentDraftName.trim())}><Icon name="folderOpen" size={14} /></button>
        <button class="ctrl" onclick={() => openCapabilityAt('skagents', agentDraftName.trim())}>{t('capability.navSkillAgents')} <Icon name="arrowRight" size={13} /></button>
      {:else}
        <!-- The shelf itself — installing, the folder, what did not read —
             is the room's สกิลของคุณ page; ticking off it is done here. -->
        <button class="ctrl" onclick={() => openCapabilityAt('skills')}>{t('capability.navSkills')} <Icon name="arrowRight" size={13} /></button>
      {/if}
    </div>
    {#if saved}
      {#each agentSkills as sk (sk.name)}
        <div class="set-row">
          <span class="cap-mark" style="--px:26px; --h:{coverHue(sk.name)}" aria-hidden="true">{sk.name.replace(/^aetox-/, '').slice(0, 2)}</span>
          <div class="set-txt">
            <div class="t">{sk.name.replace(/^aetox-/, '')} {#if sk.bundled}<span class="badge on">{t('office.builtin')}</span>{/if}</div>
            {#if sk.description}<div class="d clamp2">{sk.description}</div>{/if}
          </div>
        </div>
      {/each}
      {#if !agentSkillsReady}
        {@render waitRow()}
      {:else if agentSkills.length === 0}
        <div class="set-row"><div class="set-txt"><div class="d">{t('settings.agentSkillsNone')}</div></div></div>
      {/if}
    {:else if !shelfLoaded}
      {@render waitRow()}
    {:else}
      {#if shelfSkills.length > 6}
        <div class="set-row">
          <label class="ag-search">
            <Icon name="search" size={13} />
            <input bind:value={shelfQuery} placeholder={t('settings.agentSkillsSearch')} />
          </label>
        </div>
      {/if}
      {#each shelfShown as sk (sk.name)}
        {@const on = agentNewSkills.includes(sk.name)}
        <label class="set-row ag-reachrow" class:on>
          <span class="cap-mark" style="--px:26px; --h:{coverHue(sk.name)}" aria-hidden="true">{sk.name.replace(/^aetox-/, '').slice(0, 2)}</span>
          <div class="set-txt">
            <div class="t">{sk.name.replace(/^aetox-/, '')} {#if sk.bundled}<span class="badge on">{t('capability.bundled')}</span>{/if}</div>
            {#if sk.description}<div class="d clamp2">{sk.description}</div>{/if}
          </div>
          <span class="mswitch">
            <input type="checkbox" checked={on} aria-label={sk.name} onchange={() => toggleNewSkill(sk.name)} />
            <span></span>
          </span>
        </label>
      {/each}
      <div class="set-row"><div class="set-txt"><div class="d">
        {shelfSkills.length === 0 ? t('settings.agentSkillsNoneOnShelf') : (shelfShown.length === 0 ? t('settings.agentNoMatches') : t('settings.agentSkillsNewHint'))}
      </div></div></div>
    {/if}
  </div>
{/snippet}

<!-- The tick before a list's answer lands, drawn as the shape of a row and not
     as the empty state: "ยังไม่มี…" is a sentence, and a sentence that is
     untrue for 200 ms is still seen. -->
<!-- A new agent's folder does not exist until Save; what lives in it cannot
     be edited before then. Said on the tab, in the row shape the tab will
     have once it can. -->
{#snippet saveFirstCard(icon: IconName, text: string)}
  <div class="settings-card">
    <div class="set-row">
      <span class="ag-rowicon"><Icon name={icon} size={15} /></span>
      <div class="set-txt">
        <div class="t">{t('settings.agentSaveFirstTitle')}</div>
        <div class="d">{text}</div>
      </div>
      <button class="ctrl ctrl-primary" disabled={agentBusy !== '' || !agentDraftName.trim() || !agentDraftPrompt.trim()} onclick={saveAgent}>{t('settings.save')}</button>
    </div>
  </div>
{/snippet}

{#snippet waitRow()}
  <div class="set-row mset-skeleton ag-waitrow" aria-label={t('settings.loading')}>
    <span class="sk sk-line short"></span>
  </div>
{/snippet}

<!-- Memory is stored in this agent's own folder (agents/<name>/MEMORY.md).
     Users can view, inline-edit, delete, add entries, or open the folder directly. -->
{#snippet agentMemoryBox()}
  <!-- The head page's block (deskHead + memRow), so a delegate's memory
       reads like ผู้ช่วย's and โค้ด's: the face, who reads it, the file, the
       meter, one row per line (owner, 14 ก.ย. 2026: "พนักงานหรือเอเจนทุกตัวควร
       จะมีความจำแยกแบบนี้ CSS มาตรฐานเดียวกัน"). The queue the delegate
       proposes into it sits above, as it does on the head page. -->
  <h3 class="set-h3">{t('settings.mainMemoryOwn', { name: agentDraftName.trim() })}</h3>
  <p class="muted set-sub">{t('settings.agentMemoryHint')}</p>
  {#if agentPendingFor(agentDraftName).length > 0}
    <div class="settings-card">
      {#each agentPendingFor(agentDraftName) as c (c.id)}{@render pendingRow(c)}{/each}
    </div>
  {/if}
  <div class="settings-card mem-desk">
    {@render deskHead(agentGroup)}
    {#if agentMemoryError}
      <div class="mem-move-error"><Icon name="alertTriangle" size={13} /><span>{agentMemoryError}</span></div>
    {/if}
    {#if moveError?.scope === agentGroup.scope}
      <div class="mem-move-error"><Icon name="alertTriangle" size={13} /><span>{moveError.text}</span></div>
    {/if}
    {#if !agentMemoryReady}
      {@render waitRow()}
    {:else}
      {#each agentMemory as line, i (i)}
        {@render memRow(agentGroup, line, i)}
      {/each}
      {#if agentMemory.length === 0 && !agentAddingMemory}
        <div class="empty">{t('settings.agentMemoryNone')}</div>
      {/if}
    {/if}

    {#if agentAddingMemory}
      <div class="mem-row editing">
        <!-- svelte-ignore a11y_autofocus -->
        <textarea
          class="mem-input" rows="2" autofocus
          placeholder={t('settings.agentMemoryAddPlaceholder')}
          bind:value={agentNewMemoryText}
          onkeydown={(e) => {
            if (e.key === 'Escape') { agentAddingMemory = false; agentNewMemoryText = '' }
            else if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              if (agentNewMemoryText.trim()) void addAgentMemory(agentDraftName, agentNewMemoryText)
            }
          }}
        ></textarea>
        <div class="mem-actions">
          <button
            type="button" class="ctrl ctrl-primary"
            disabled={agentMemorySaving || !agentNewMemoryText.trim()}
            onclick={() => addAgentMemory(agentDraftName, agentNewMemoryText)}
          >{t('settings.learningMemorySave')}</button>
          <button
            type="button" class="ctrl" disabled={agentMemorySaving}
            onclick={() => { agentAddingMemory = false; agentNewMemoryText = '' }}
          >{t('settings.learningMemoryCancel')}</button>
        </div>
      </div>
    {/if}

    <div class="set-row learn-foot">
      <div class="set-txt">
        <div class="d muted">{t('settings.agentMemoryFolderHint')}</div>
      </div>
      <div style="display:flex; gap:8px;">
        {#if !agentAddingMemory}
          <button
            type="button" class="ctrl ctrl-primary"
            disabled={!agentEditing?.name}
            onclick={() => { agentAddingMemory = true; agentNewMemoryText = '' }}
          >
            <Icon name="plus" size={13} /> {t('settings.agentMemoryAddBtn')}
          </button>
        {/if}
        <button
          type="button" class="ctrl"
          disabled={!agentEditing?.name}
          onclick={() => OpenAgentHome(agentDraftName.trim())}
        >
          <Icon name="folderOpen" size={13} /> {t('settings.agentOpenHomeFolder')}
        </button>
      </div>
    </div>
  </div>
{/snippet}

<!-- How this agent opens a conversation (STARTERS.md in its folder), edited
     here rather than only in a text editor (owner, 10 ส.ค.).
     A form over the file, not a replacement for it: hand-editing stays exactly
     as valid, which is why what Save writes is a heading and a list and
     nothing a person would not have typed themselves.
     A growable list. It was four fixed rows while the window drew every card a
     file held; now the window deals four out of a pool, so the form has to be
     able to build a pool deeper than the grid. Floored at four so nothing can
     be saved that deals a widow, ceilinged at what the engine will read back. -->
{#snippet agentStartersBox()}
  <div class="settings-card">
    <div class="card-form pp-edit">
      <div class="pp-field">
        <div class="pp-bodyhead">
          <span class="eyebrow eyebrow-grow">{t('settings.agentStartersTitle')}</span>
          <span class="d muted mono-dim">{agentStartersFile}</span>
        </div>
        <div class="d muted">{t('settings.agentStartersHint')}</div>
      </div>

      <div class="pp-field">
        <div class="ag-starter-head">
          <span class="eyebrow eyebrow-grow">{t('settings.agentStartersHeadline')}</span>
          <span class="d muted">{t('settings.agentStartersHeadlinesHint')}</span>
        </div>
        {#each startersHeadlines as _, i (i)}
          <div class="ag-starter-row">
            <input class="ctrl" bind:value={startersHeadlines[i]} placeholder={t('settings.agentStartersHeadlinePlaceholder')} aria-label={t('settings.agentStartersHeadline')} />
            <button class="ag-headline-drop" title={t('settings.agentStarterRemove')} aria-label={t('settings.agentStarterRemove')} onclick={() => removeHeadlineRow(i)}>
              <Icon name="x" size={13} />
            </button>
          </div>
        {/each}
        {#if startersHeadlines.length < 12}
          <button class="ctrl ag-headline-add" onclick={addHeadlineRow}>
            <Icon name="plus" size={13} />
            <span>{t('settings.agentStartersHeadlineAdd')}</span>
          </button>
        {/if}
      </div>

      {#each startersCards as card, i (i)}
        <div class="pp-field ag-starter">
          <div class="ag-starter-head">
            <span class="eyebrow eyebrow-grow">{t('settings.agentStarterCard', { n: i + 1 })}</span>
            <button
              class="ag-starter-drop"
              title={t('settings.agentStarterRemove')}
              aria-label={t('settings.agentStarterRemove')}
              onclick={() => removeStarterRow(i)}
            >
              <Icon name="x" size={13} />
            </button>
          </div>
          <div class="ag-starter-row">
            <input class="ctrl" bind:value={card.title} placeholder={t('settings.agentStarterTitlePlaceholder')} />
            <select class="ctrl ag-starter-icon" bind:value={card.icon} aria-label={t('settings.agentIcon')}>
              <option value="">{t('settings.agentStarterNoIcon')}</option>
              {#each AGENT_ICONS as name (name)}<option value={name}>{name}</option>{/each}
            </select>
          </div>
          <input class="ctrl" bind:value={card.prompt} placeholder={t('settings.agentStarterPromptPlaceholder')} />
        </div>
      {/each}

      {#if canAddStarter}
        <button class="ctrl ag-starter-add" onclick={addStarterRow}>
          <Icon name="plus" size={13} />
          <span>{t('settings.agentStarterAdd')}</span>
        </button>
      {/if}

      <!-- The trailing-colon rule is the author's, not a quirk to discover: a
           prompt that ends in ":" is the deliberate half-sentence the user
           finishes in the composer. The pool line beside it is the other thing
           an author cannot see from here: the grid draws four of these. -->
      <div class="d muted">{t('settings.agentStartersColonHint')}</div>
      <div class="d muted">{t('settings.agentStartersPoolHint', { shown: 4, held: startersCards.length })}</div>

      {#if startersError}<div class="mset-error">{startersError}</div>{/if}

      <div class="pp-bar">
        <span class="d muted">
          {startersInherited ? t('settings.agentStartersInherited') : t('settings.agentStartersOwn')}
        </span>
        <div class="pp-bar-gap"></div>
        <button class="ctrl" disabled={startersBusy || startersEmpty} onclick={clearStarters}>
          {t('settings.agentStartersClear')}
        </button>
        <button class="ctrl ctrl-primary" disabled={startersBusy || !startersDirty} onclick={saveStarters}>
          {startersBusy ? t('settings.saving') : t('settings.agentStartersSave')}
        </button>
      </div>
    </div>
  </div>
{/snippet}

{#snippet railItemContent(it: { id: string; label: string; icon: any })}
  <span class="ic"><Icon name={it.icon} /></span> {it.label}
  <!-- The rank's bars on the three rows that are levels of the company
       (owner, 14 ก.ย. 2026: "ในหน้าเมนู ทำสัญลักษณ์ยศแปะไว้ด้วย"): the
       same emblem the faces wear, so the rail reads as the roster. -->
  {#if it.id === 'main' || it.id === 'team' || it.id === 'agents'}
    <span class="nav-rank"><RankPip tier={it.id === 'main' ? 'head' : it.id === 'team' ? 'agent' : 'helper'} word={false} /></span>
  {/if}
  <!-- The queue's count, on the row where each item is decided
       (railPending): ตัวหลัก for the heads' and the projects', เกี่ยวกับคุณ
       for the person's, พนักงาน / ลูกมือ for a delegate's. Until 14 ก.ย.
       2026 the whole number sat on ตัวหลัก and pointed at nothing. -->
  {#if (it.id === 'main' || it.id === 'you' || it.id === 'team' || it.id === 'agents') && railPending[it.id] > 0}
    <span class="nav-count" title={t('settings.learningWaiting', { count: String(railPending[it.id]) })}>
      {railPending[it.id]}
    </span>
  {/if}
  <!-- The same mark, and only here: the gear in the sidebar stays the
       learning queue's alone. A problem is worth finding when you come
       looking and is not worth being pulled out of a chat for. -->
  {#if it.id === 'issues' && cockpit.pendingIssues > 0}
    <span class="nav-count" title={t('settings.issuesWaiting', { count: String(cockpit.pendingIssues) })}>
      {cockpit.pendingIssues}
    </span>
  {/if}
{/snippet}

<div class="settings-page">
  <aside class="settings-nav">
    <button data-guide="settings.back" class="settings-back" onclick={onClose}><Icon name="arrowLeft" size={14} /> {t('settings.backToApp')}</button>
    <input data-guide="settings.search" class="settings-search" placeholder={t('settings.searchPlaceholder')} bind:value={query} />
    {#each filteredSections as g}
      <div class="settings-group-label eyebrow">{g.group}</div>
      {#each g.items as it}
        {#if it.id === 'general'}
          <button data-guide="settings.rail.general" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'appearance'}
          <button data-guide="settings.rail.appearance" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'avatar'}
          <button data-guide="settings.rail.avatar" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'you'}
          <button data-guide="settings.rail.you" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'issues'}
          <button data-guide="settings.rail.issues" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'models'}
          <button data-guide="settings.rail.brain" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'main'}
          <button data-guide="settings.rail.heads" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'teams'}
          <button data-guide="settings.rail.teams" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'team'}
          <button data-guide="settings.rail.agents" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'agents'}
          <button data-guide="settings.rail.hands" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'voice'}
          <button data-guide="settings.rail.voice" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'image'}
          <button data-guide="settings.rail.image" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'studio'}
          <button data-guide="settings.rail.studio" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'remote'}
          <button data-guide="settings.rail.remote" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'account'}
          <button data-guide="settings.rail.account" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'usage'}
          <button data-guide="settings.rail.usage" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'about'}
          <button data-guide="settings.rail.about" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else if it.id === 'sponsor'}
          <button data-guide="settings.rail.sponsor" class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {:else}
          <button class="settings-nav-item" class:active={active === it.id} onclick={() => openSection(it.id)}>{@render railItemContent(it)}</button>
        {/if}
      {/each}
    {/each}
    {#if noSearchResults}
      <div class="settings-nav-empty">{t('settings.searchNoResults', { q: query.trim() })}</div>
    {/if}
  </aside>

  <div class="settings-content" bind:this={contentEl}>
    <div class="settings-inner" style:--content-max={active === 'usage' ? '960px' : null}>
    {#if bootError}
      <!-- The whole page used to be one unguarded await chain, so a backend
           that wasn't up yet produced a blank page and no explanation. -->
      <div class="settings-banner">
        <div class="set-txt">
          <div class="t">{t('settings.bootErrorTitle')}</div>
          <div class="d">{t('settings.bootErrorHint')}</div>
          <div class="d mono-dim">{bootError}</div>
        </div>
        <button class="ctrl ctrl-primary" disabled={booting} onclick={bootSettings}>
          {booting ? t('settings.loading') : t('settings.retry')}
        </button>
      </div>
    {/if}
    {#if active === 'general'}
      <h2>{t('settings.general')}</h2>
      <p class="muted set-sub">{t('settings.generalDesc')}</p>

      <div class="group-head">
        <span class="group-title">{t('settings.groupTerminal')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.shellTitle')}</div>
            <div class="d">{t('settings.shellDesc')}</div>
          </div>
          {#if shells.length === 0}
            <span class="muted">{t('settings.noShells')}</span>
          {:else}
            <select class="ctrl" bind:value={defaultShell} onchange={saveDefaultShell}>
              {#each shells as s}
                <option value={s.path}>{s.name}</option>
              {/each}
            </select>
          {/if}
        </div>
      </div>

      <div class="group-head">
        <span class="group-title">{t('settings.groupSafety')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.approvalTitle')}</div>
            <div class="d">{t('settings.approvalDesc')}</div>
          </div>
          <div class="seg-ctrl" role="radiogroup" aria-label={t('settings.approvalTitle')}>
            {#each approvalOptions as opt}
              <button
                type="button"
                class="seg-btn"
                class:selected={cockpit.model.approval === opt.value}
                onclick={() => switchApprovalMode(opt.value)}
              >
                {opt.label}
              </button>
            {/each}
          </div>
        </div>
      </div>

      <div class="group-head">
        <span class="group-title">{t('settings.groupBehavior')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.preparedReplyTitle')}</div>
            <div class="d">{t('settings.preparedReplyDesc')}</div>
          </div>
          <label class="mswitch" aria-label={t('settings.preparedReplyTitle')}>
            <input type="checkbox" checked={preparedOn} onchange={togglePreparedReply} />
            <span></span>
          </label>
        </div>
        <!-- The one switch การเรียนรู้ had: whether Aetox records outcomes and
             proposes anything at all, every scope at once. A system switch,
             so it sits with the system's others (14 ก.ย. 2026). -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.learningEnabled')}</div>
            <div class="d">{t('settings.learningEnabledHint')}</div>
          </div>
          <label class="mswitch" aria-label={t('settings.learningEnabled')}>
            <input type="checkbox" checked={learningOn} onchange={toggleLearning} />
            <span></span>
          </label>
        </div>
      </div>

      <!-- เรียกให้หัน (desktop/attention.go). The words come from Go, the way
           the busy signal's do: the id is ours and the words are the
           product's, and a second table of names here would be a second place
           for them to drift from what the switch actually does. -->
      <div class="group-head">
        <span class="group-title">{t('settings.groupAttention')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="d">{t('settings.attentionHint')}</div>
          </div>
        </div>
        {#each attention.layers as layer (layer.id)}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{layer.label}</div>
              <div class="d">{layer.note}</div>
            </div>
            <label class="mswitch" aria-label={layer.label}>
              <input type="checkbox" checked={layer.on} onchange={() => void toggleAttention(layer.id, !layer.on)} />
              <span></span>
            </label>
          </div>
        {/each}
      </div>

      <div class="group-head">
        <span class="group-title">{t('settings.groupSystem')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.firstRunTitle')}</div>
            <div class="d">{t('settings.firstRunDesc')}</div>
          </div>
          <button class="ctrl" onclick={replayFirstRun}>{t('settings.firstRunAction')}</button>
        </div>
      </div>
    {:else if active === 'appearance'}
      <h2>{t('settings.appearance')}</h2>
      <!-- Every zone carries a heading, including the first. One unlabelled card
           above three labelled ones reads as an oversight, not as an intro. -->
      <div class="group-head">
        <span class="group-title">{t('settings.zoneLook')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.languageTitle')}</div>
            <div class="d">{t('settings.languageDesc')}</div>
          </div>
          <select data-guide="settings.general.language" class="ctrl" value={i18n.locale} onchange={(e) => setLocale(e.currentTarget.value as Locale)}>
            {#each Object.entries(localeNames) as [code, name]}
              <option value={code}>{name}</option>
            {/each}
          </select>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.themeTitle')}</div>
            <div class="d">{t('settings.themeDesc')}</div>
          </div>
          <select data-guide="settings.general.theme" class="ctrl" value={theme.name} onchange={(e) => applyTheme(e.currentTarget.value as ThemeName)}>
            {#each THEMES as th}
              <option value={th.value}>{th.label}</option>
            {/each}
          </select>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.uiFontTitle')}</div>
            <div class="d">{t('settings.uiFontDesc')}</div>
          </div>
          <select class="ctrl" value={uiFont.name} onchange={(e) => applyUiFont(e.currentTarget.value as UiFontName)}>
            {#each UI_FONTS as f}
              <option value={f.value}>{t(f.labelKey)}</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="group-head">
        <span class="group-title">{t('settings.zoneTextSize')}</span>
      </div>
      <div class="settings-card">
        <!-- Text size sits above overall size on purpose: it is the one people
             actually come here for, and reading the two in this order is what
             makes the difference between them land. -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.typeScaleTitle')}</div>
            <div class="d">{t('settings.typeScaleDesc')}</div>
          </div>
          <div data-guide="settings.general.font_scale" class="seg-ctrl">
            {#each TYPE_SCALES as s (s.value)}
              <button
                type="button" class="seg-btn" class:selected={typeScale.name === s.value}
                onclick={() => applyTypeScale(s.value as TypeScaleName)}
              >{t(s.labelKey)}</button>
            {/each}
          </div>
        </div>
        <!-- Three steps of the scale at once. A single sample line cannot show
             what a scale does — the thing being chosen is the gap between the
             heading and the caption, not any one size. -->
        <div class="set-row type-preview">
          <div class="tsp-heading">{t('settings.typeScalePreviewHeading')}</div>
          <div class="tsp-body">{t('settings.typeScalePreviewBody')}</div>
          <div class="tsp-caption">{t('settings.typeScalePreviewCaption')}</div>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.systemZoomTitle')}</div>
            <div class="d">{t('settings.systemZoomDesc')}</div>
          </div>
          <input
            class="ctrl set-num" type="number" min="12" max="20" step="0.5"
            value={Math.round(systemZoom.value * SYSTEM_BASE_PX * typeScale.scale * 10) / 10}
            onchange={(e) => applySystemZoom(parseFloat(e.currentTarget.value) / (SYSTEM_BASE_PX * typeScale.scale))}
          />
          <span class="muted set-unit">px</span>
        </div>
      </div>

      <div class="group-head">
        <span class="group-title">{t('settings.zonePaneSizes')}</span>
        <span class="group-count">{t('settings.zonePaneSizesHint')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.editorFontTitle')}</div>
            <div class="d">{t('settings.editorFontDesc')}</div>
          </div>
          <input
            class="ctrl set-num" type="number" min="10" max="24" step="0.5"
            value={editorFont.size}
            onchange={(e) => applyEditorFontSize(parseFloat(e.currentTarget.value))}
          />
          <span class="muted set-unit">px</span>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.chatFontTitle')}</div>
            <div class="d">{t('settings.chatFontDesc')}</div>
          </div>
          <input
            class="ctrl set-num" type="number" min="11" max="22" step="0.5"
            value={chatFont.size}
            onchange={(e) => applyChatFontSize(parseFloat(e.currentTarget.value))}
          />
          <span class="muted set-unit">px</span>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.treeFontTitle')}</div>
            <div class="d">{t('settings.treeFontDesc')}</div>
          </div>
          <input
            class="ctrl set-num" type="number" min="11" max="18" step="0.5"
            value={treeFont.size}
            onchange={(e) => applyTreeFontSize(parseFloat(e.currentTarget.value))}
          />
          <span class="muted set-unit">px</span>
        </div>
      </div>

      <div class="group-head">
        <span class="group-title">{t('settings.zoneCode')}</span>
      </div>
      <div class="settings-card">
        <div class="set-row">
          <span class="muted set-unit">px</span>
          <div class="set-txt">
            <div class="t">{t('settings.codeThemeTitle')}</div>
            <div class="d">{t('settings.codeThemeDesc')}</div>
          </div>
          <select class="ctrl" value={editorTheme.choice} onchange={(e) => {
            const v = e.currentTarget.value
            if (v === 'auto') setAutoEditorTheme()
            else if (v === 'vs-dark' || v === 'vs') setBuiltinEditorTheme(v)
          }}>
            <option value="auto">{t('settings.codeThemeAuto')}</option>
            <option value="vs-dark">{t('settings.codeThemeDark')}</option>
          <span class="muted set-unit">px</span>
            <option value="vs">{t('settings.codeThemeLight')}</option>
            {#if editorTheme.importedName}
              <option value="imported">{editorTheme.importedName}</option>
            {/if}
          </select>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.importThemeTitle')}</div>
            <div class="d">{t('settings.importThemeDesc')}</div>
          </div>
          <label class="ctrl">
            {t('settings.importThemeButton')}
            <input type="file" accept=".json,application/json" style="display:none" onchange={onThemeFileChosen} />
          </label>
        </div>
        {#if themeImportError}<div class="mset-error">{themeImportError}</div>{/if}
      </div>
    {:else if active === 'models'}
      <h2>{t('settings.modelSettings')}</h2>
      <p class="muted set-sub">{t('settings.modelsDesc')}</p>

      <div class="settings-card mset">
        <aside class="mset-side">
          <div class="settings-group-label eyebrow">{t('settings.providers')}</div>
          {#each enabledRows as p (p.name)}
            <div class="mset-prov-row">
              <!-- Closing the add form here, on the click, and not inside
                   selectProvider: the boot also selects a row, and a form
                   opened while the page was still loading must survive it. -->
              {#if p.name === 'ollama'}
                <button data-guide="settings.brain.provider.ollama" class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {:else if p.name === 'openai'}
                <button data-guide="settings.brain.provider.openai" class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {:else if p.name === 'anthropic'}
                <button data-guide="settings.brain.provider.anthropic" class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {:else if p.name === 'deepseek'}
                <button data-guide="settings.brain.provider.deepseek" class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {:else if p.name === 'google'}
                <button data-guide="settings.brain.provider.google" class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {:else if p.name === 'groq'}
                <button data-guide="settings.brain.provider.groq" class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {:else if p.name === 'openrouter'}
                <button data-guide="settings.brain.provider.openrouter" class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {:else}
                <button class="mset-prov" class:selected={selected === p.name} onclick={() => { customDraftOpen = false; selectProvider(p.name) }}>
                  <ProviderMark name={p.name} size={15} />
                  <span class="mset-prov-name">{p.name}</span>
                  <span class="dot" class:green={p.ready === true} class:unknown={p.ready === null} title={p.ready === null ? t('settings.providerChecking') : p.ready ? t('settings.providerReady') : t('settings.providerNotReady')}></span>
                </button>
              {/if}
              {#if customNames.has(p.name)}
                <button class="icobtn tiny" disabled={busy === 'disable:' + p.name}
                  aria-label={t('settings.remove')} onclick={() => removeCustomProvider(p.name)}><Icon name="x" size={13} /></button>
              {:else if enabledRows.length > 1}
                <button class="icobtn tiny" disabled={busy === 'disable:' + p.name}
                  aria-label={t('settings.remove')} onclick={() => removeProvider(p.name)}><Icon name="x" size={13} /></button>
              {/if}
            </div>
          {/each}

          <button data-guide="settings.brain.add_provider" class="mset-prov mset-add-toggle" onclick={() => (showAddProvider = !showAddProvider)}>
            <Icon name="plus" size={14} /> {t('settings.addProvider')}
          </button>
          {#if showAddProvider}
            <div class="mset-add-list">
              {#if addableSignIn.length > 0}
                <div class="mset-add-group">{t('settings.groupSignIn')}</div>
                {#each addableSignIn as p (p.name)}
                  <button class="mset-prov" disabled={busy === 'enable:' + p.name} onclick={() => addProvider(p.name)}>
                    <ProviderMark name={p.name} size={15} />
                    <span class="mset-prov-name">{p.name}</span>
                    <span class="dot">{busy === 'enable:' + p.name ? '…' : '+'}</span>
                  </button>
                {/each}
              {/if}
              {#if addableKeyed.length > 0}
                <div class="mset-add-group">{t('settings.groupApiKey')}</div>
                {#each addableKeyed as p (p.name)}
                  <button class="mset-prov" disabled={busy === 'enable:' + p.name} onclick={() => addProvider(p.name)}>
                    <ProviderMark name={p.name} size={15} />
                    <span class="mset-prov-name">{p.name}</span>
                    <span class="dot">{busy === 'enable:' + p.name ? '…' : '+'}</span>
                  </button>
                {/each}
              {/if}
              <!-- Always offered, even when every catalog row is enabled:
                   this is the one entry that can be added more than once. -->
              <div class="mset-add-group">{t('settings.groupCustom')}</div>
              <button class="mset-prov" onclick={() => openCustomDraft()}>
                <Icon name="plugZap" size={15} />
                <span class="mset-prov-name">{t('settings.customEndpoint')}</span>
                <span class="dot">+</span>
              </button>
            </div>
          {/if}
        </aside>

        <div class="mset-detail">
          {#if customDraftOpen}
            <div class="mset-head">
              <Icon name="plugZap" size={22} />
              <span class="mset-name">{t('settings.customEndpointTitle')}</span>
            </div>
            <p class="muted set-hint">{t('settings.customEndpointDesc')}</p>
            <div class="mset-field">
              <div class="eyebrow">{t('settings.customNameLabel')}</div>
              <div class="muted set-hint">{t('settings.customNameHint')}</div>
              <!-- svelte-ignore a11y_autofocus -->
              <input class="ctrl key-input" placeholder="deepseek-2" bind:value={customDraft.name} autofocus />
            </div>
            <div class="mset-field">
              <div class="eyebrow">{t('settings.baseUrl')}</div>
              <input class="ctrl key-input" placeholder="https://api.example.com/v1" bind:value={customDraft.baseURL}
                onkeydown={(e) => e.key === 'Enter' && submitCustomDraft()} />
            </div>
            <div class="mset-field">
              <div class="eyebrow">{t('settings.apiKeyLabel')}</div>
              <div class="muted set-hint">
                {customDraftCopiesKey ? t('settings.customKeyCopied', { provider: customDraftFrom }) : t('settings.customKeyHint')}
              </div>
              <input class="ctrl key-input" type="password" autocomplete="off" bind:value={customDraft.apiKey}
                onkeydown={(e) => e.key === 'Enter' && submitCustomDraft()} />
            </div>
            {#if customDraftError}
              <div class="mset-error">{customDraftError}</div>
            {/if}
            <div class="mset-keyrow">
              <button class="ctrl ctrl-primary"
                disabled={busy !== '' || customDraft.name.trim() === '' || customDraft.baseURL.trim() === ''}
                onclick={submitCustomDraft}>
                {busy === 'custom:add' ? t('settings.saving') : t('settings.customAdd')}
              </button>
              <button class="ctrl" disabled={busy !== ''} onclick={() => (customDraftOpen = false)}>{t('settings.cancel')}</button>
            </div>
          {:else if selectedRow}
            <div data-guide="settings.brain.hero" class="mset-head">
              <ProviderMark name={selected} size={22} />
              <span class="mset-name">{selected}</span>
              {#if customNames.has(selected)}
                <span class="badge custom" title={t('settings.customBadgeTitle')}>{t('settings.customBadge')}</span>
              {/if}
              {#if isActiveProvider}
                <span class="badge on">{t('settings.active')}</span>
              {:else}
                <button class="ctrl ctrl-primary" disabled={busy !== ''} onclick={useProvider}>
                  {busy === 'provider' ? t('settings.switching') : t('settings.useThisProvider')}
                </button>
              {/if}
            </div>

            {#if isActiveProvider && cockpit.model.warning}
              <!-- Without this the "Active" badge above claims a provider the
                   engine never reached (LM Studio with its server off). -->
              <div class="conn-test">{t('chat.providerFallback')} · {cockpit.model.warning}</div>
            {/if}

            {#if account}
              <div class="mset-acct">
                <ProviderAccount {account} />
                {#if account.balance?.hasAmount || account.quotaFetched}
                  <button class="ctrl tiny" disabled={busy === 'account'} onclick={refreshAccount}>
                    <Icon name="refreshCw" size={13} /> {t('settings.refreshBalance')}
                  </button>
                {/if}
              </div>
            {/if}

            <div class="mset-field">
              <div class="eyebrow">{t('settings.baseUrl')}</div>
              <div class="muted set-hint">{t('settings.baseUrlDesc')}</div>
              <div class="mset-keyrow">
                <input
                  class="ctrl key-input" placeholder={baseURL || 'http://localhost:1234/v1'}
                  bind:value={baseURLDraft}
                  onkeydown={(e) => e.key === 'Enter' && saveBaseURL(baseURLDraft)}
                />
                <button class="ctrl ctrl-primary" disabled={busy !== '' || baseURLDraft.trim() === baseURL} onclick={() => saveBaseURL(baseURLDraft)}>
                  {busy === 'baseUrl' ? t('settings.saving') : t('settings.save')}
                </button>
                {#if baseURLIsCustom}
                  <button class="ctrl" disabled={busy !== ''} onclick={() => saveBaseURL('')}>{t('settings.baseUrlReset')}</button>
                {/if}
              </div>
              {#if selected === 'openai-compatible' || customNames.has(selected)}
                <!-- The owner's "+": this card holds one endpoint at a time,
                     and typing a second one over it is how the first got
                     lost. Saving the card as a row of its own keeps both. -->
                <button class="ctrl mset-save-as" disabled={busy !== ''} onclick={() => openCustomDraft(selected)}>
                  <Icon name="plus" size={13} /> {t('settings.customSaveAs')}
                </button>
              {/if}
            </div>

            {#if wireFormats.length > 1}
              <div class="mset-field">
                <div class="eyebrow">{t('settings.wireFormat')}</div>
                <div class="muted set-hint">{t('settings.wireFormatDesc')}</div>
                <div class="mset-keyrow">
                  {#each wireFormats as fmt}
                    {#if currentWireFormat === fmt}
                      <span class="badge on">{wireFormatLabel(fmt)}</span>
                    {:else}
                      <button class="ctrl" disabled={busy !== ''} onclick={() => useFormat(fmt)}>
                        {busy === 'format:' + fmt ? t('settings.switching') : wireFormatLabel(fmt)}
                      </button>
                    {/if}
                  {/each}
                </div>
              </div>
            {/if}

            {#if signInMethod}
              <div class="mset-field">
                <div class="eyebrow">{t('settings.signInLabel')}</div>

                {#if signInStatus?.signed_in}
                  <div class="mset-keyrow">
                    <span class="badge on">{signInStatus.label || t('settings.signedInAs')}</span>
                    <button class="ctrl" disabled={busy !== ''} onclick={() => doSignOut(signInMethod.provider)}>
                      {busy === 'signout:' + signInMethod.provider ? '…' : t('settings.signOut')}
                    </button>
                  </div>
                {:else if signInPrompt}
                  {@const prompt = signInPrompt}
                  <div class="signin-flow">
                    {#if prompt.kind === 'device'}
                      <div class="muted">{t('settings.signInDeviceStep')}</div>
                      <div class="signin-code">{prompt.user_code}</div>
                      <div class="mset-keyrow">
                        <button class="ctrl" onclick={() => BrowserOpenURL(prompt.verification_uri || prompt.url)}>
                          {t('settings.signInOpenPage')}
                        </button>
                        <button class="ctrl" onclick={abortSignIn}>{t('settings.signInCancel')}</button>
                      </div>
                      <div class="muted">{t('settings.signInWaiting')}</div>
                    {:else if prompt.kind === 'paste'}
                      <div class="muted">{t('settings.signInPasteStep')}</div>
                      <div class="mset-keyrow">
                        <input
                          class="ctrl key-input" type="password"
                          placeholder={t('settings.signInPastePlaceholder')}
                          bind:value={signInCode}
                          onkeydown={(e) => e.key === 'Enter' && finishSignIn()}
                        />
                        <button class="ctrl" disabled={busy === 'signin' || !signInCode.trim()} onclick={finishSignIn}>
                          {busy === 'signin' ? '…' : t('settings.signInSubmit')}
                        </button>
                        <button class="ctrl" onclick={abortSignIn}>{t('settings.signInCancel')}</button>
                      </div>
                    {:else}
                      <div class="muted">{t('settings.signInWaiting')}</div>
                      <div class="mset-keyrow">
                        <button class="ctrl" onclick={() => BrowserOpenURL(prompt.url)}>
                          {t('settings.signInOpenPage')}
                        </button>
                        <button class="ctrl" onclick={abortSignIn}>{t('settings.signInCancel')}</button>
                      </div>
                    {/if}
                  </div>
                {:else}
                  <div class="mset-keyrow">
                    <button class="ctrl ctrl-primary" disabled={busy !== ''} onclick={startSignIn}>
                      {t('settings.signInWith', { label: signInMethod.label })}
                    </button>
                    {#if importable.includes(signInMethod.provider)}
                      <button class="ctrl" disabled={busy !== ''} onclick={() => doImport(signInMethod.provider)}>
                        {busy === 'import:' + signInMethod.provider ? '…' : t('settings.signInImport')}
                      </button>
                    {/if}
                  </div>
                  <div class="muted">{signInMethod.note}</div>
                  {#if signInMethod.risk === 'restricted'}
                    <div class="signin-warn">{t('settings.signInRestricted')}</div>
                  {/if}
                {/if}

                {#if signInError}
                  <div class="conn-test">{signInError}</div>
                {/if}
              </div>
            {/if}

            {#if selectedRow.requiresKey && selectedRow.acceptsKey}
              <div class="mset-field">
                <div class="eyebrow eyebrow-row">
                  <span>{signInMethod ? t('settings.signInOrKey') : t('settings.apiKeyLabel')}</span>
                  <!-- The card asks for a key; every provider hides the page
                       that issues one somewhere different. Drawn only when the
                       catalog knows a page for this row. -->
                  {#if keyPageURL}
                    <button class="keylink" onclick={() => BrowserOpenURL(keyPageURL)}>
                      {t('settings.getKey')}<Icon name="externalLink" size={12} />
                    </button>
                  {/if}
                </div>
                <div class="mset-keyrow">
                  <input
                    class="ctrl key-input" type={showKey ? 'text' : 'password'}
                    placeholder={selectedRow.hasKey
                      ? (selectedRow.keyHint
                          ? t('settings.keySetHintPlaceholder', { hint: selectedRow.keyHint })
                          : t('settings.keySetPlaceholder'))
                      : t('settings.pasteKeyPlaceholder')}
                    bind:value={keyDraft}
                    onkeydown={(e) => e.key === 'Enter' && saveKey()}
                  />
                  <button class="icobtn tiny" aria-label={t('settings.showKey')} onclick={() => (showKey = !showKey)}><Icon name="eye" size={14} /></button>
                  <button class="ctrl ctrl-primary" disabled={busy === 'key' || !keyDraft.trim()} onclick={saveKey}>
                    {busy === 'key' ? t('settings.saving') : t('settings.save')}
                  </button>
                </div>
              </div>
            {/if}

            <div class="mset-field">
              <div class="eyebrow">{t('settings.modelList')}</div>
              {#if loadingModels}
                <div class="muted">{t('settings.loadingModels')}</div>
              {:else if models.length === 0}
                <div class="muted">{t('settings.noModels')}</div>
              {:else}
                <!-- Only where the list is long enough to be worth searching.
                     Six rows do not need a filter above them. -->
                {#if models.length > 8}
                  <div class="mlist-tools">
                    <input
                      class="ctrl mlist-search" type="search"
                      placeholder={t('settings.filterModels', { n: String(models.length) })}
                      bind:value={modelFilter}
                    />
                    {#if freeCount > 0}
                      <button
                        class="conn-chip" class:on={freeOnly}
                        aria-pressed={freeOnly} onclick={() => (freeOnly = !freeOnly)}
                      >{t('settings.freeOnly')} {freeCount}</button>
                    {/if}
                  </div>
                {/if}
                {#if visibleModels.length === 0}
                  <div class="muted">{t('settings.noModelsMatch')}</div>
                {/if}
                {#each visibleModels as m}
                  <div class="mrow">
                    <span class="mname">{m}</span>
                    <!-- What the row costs, or a dash. Never a zero: on a list
                         this long a zero reads as "free" and the user would
                         act on it, and 84% coverage means the other 16% are
                         genuinely unknown rather than cheap. -->
                    {#if priced[m]?.free}
                      <span class="mprice free">{t('settings.priceFree')}</span>
                    {:else if priced[m]?.priced}
                      <span class="mprice" title={t('settings.pricePerMillion')}>
                        ${priced[m].input} / ${priced[m].output}
                      </span>
                    {:else}
                      <span class="mprice dim">—</span>
                    {/if}
                    <button
                      data-guide="settings.brain.test_connection"
                      class="icobtn tiny" title={t('settings.testConnection')} aria-label={t('settings.testConnection')}
                      disabled={connTesting[m]} onclick={() => testConnection(m)}
                    >{#if connTesting[m]}…{:else}<Icon name="plugZap" size={14} />{/if}</button>
                    {#if isActiveProvider && cockpit.model.modelName === m}
                      <span class="badge on">{t('settings.inUse')}</span>
                    {:else}
                      <button class="ctrl" disabled={busy !== ''} onclick={() => useModel(m)}>
                        {busy === m ? t('settings.switching') : t('settings.use')}
                      </button>
                    {/if}
                  </div>
                  {#if connResult[m]}
                    <div class="conn-test" class:ok={connResult[m].startsWith('ok:')}>
                      {#if connResult[m].startsWith('ok:')}
                        <Icon name="check" size={13} /> {t('settings.connOk')}: {connResult[m].slice(3)}
                      {:else}
                        <Icon name="x" size={13} /> {connResult[m].slice(4)}
                      {/if}
                    </div>
                  {/if}
                {/each}
              {/if}
              {#if priceSourceLine}
                <div class="mlist-source">{priceSourceLine}</div>
              {/if}
              <div class="mset-keyrow">
                <input
                  class="ctrl key-input" placeholder={t('settings.customModelPlaceholder')}
                  bind:value={customModel}
                  onkeydown={(e) => e.key === 'Enter' && customModel.trim() && useModel(customModel.trim())}
                />
                <button class="ctrl" disabled={busy !== '' || !customModel.trim()} onclick={() => useModel(customModel.trim())}>{t('settings.use')}</button>
              </div>
            </div>

            {#if errorMsg}
              <div class="mset-error">{errorMsg}</div>
            {/if}
          {:else if booting}
            <!-- refreshProviders fans out one IPC round-trip per provider, so
                 this pane sat completely blank on every open. A skeleton says
                 "coming"; nothing says "broken". -->
            <div class="mset-skeleton" aria-label={t('settings.loading')}>
              <span class="sk sk-head"></span>
              <span class="sk sk-line"></span>
              <span class="sk sk-line short"></span>
              <span class="sk sk-block"></span>
            </div>
          {:else}
            <div class="mset-empty muted">{t('settings.noProviderSelected')}</div>
          {/if}
        </div>
      </div>
    {:else if active === 'voice'}
      <h2>{t('settings.voice')}</h2>
      <p class="muted set-sub">{t('settings.voiceDesc')}</p>

      {#if voicePageError}<div class="mset-error">{voicePageError}</div>{/if}

      <div class="group-head"><span class="group-title">{t('settings.sttHeading')}</span></div>
      <div class="settings-card">
        <!-- The hardware end of the chain, and first because it is the end
             that fails silently. getUserMedia({audio:true}) takes whatever
             Windows calls default; on a machine with a headset jack holding
             nothing and NVIDIA Broadcast's virtual mic in the list, that is a
             coin toss between a recording and 30 seconds of silence — which
             comes back worded as if the user had mumbled (owner, 8 ก.ย. 2026). -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.audioInput')}</div>
            <div class="d">{audioDevices.labelled ? t('settings.audioInputDesc') : t('settings.audioNamesHidden')}</div>
          </div>
          <select data-guide="settings.voice.device_select" class="ctrl" value={audioDevices.micId} onchange={(e) => setMicId(e.currentTarget.value)}>
            <option value="">{t('settings.audioDefault')}</option>
            {#each audioDevices.mics as d (d.id)}<option value={d.id}>{d.label}</option>{/each}
          </select>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.sttEngine')}</div>
            <div class="d">{t('settings.sttEngineDesc')}</div>
            {#if speechStatus && voiceInstallBusy !== 'stt'}
              <!-- The engine's own sentence — it already names the missing
                   piece and how to get it. Hidden while the install it asks
                   for is running: "missing" and "installing" cannot both be
                   true on one screen. -->
              <div class="d mset-error">{speechStatus}</div>
            {/if}
            {@render voiceInstall('stt', activeSttEngine, speechStatus)}
          </div>
          <select data-guide="settings.voice.switch" class="ctrl" disabled={voicePageBusy} value={sttPick} onchange={(e) => pickSttEngine(e.currentTarget.value)}>
            {#each sttEngines as eng (eng.id)}<option value={eng.id}>{eng.label}</option>{/each}
          </select>
        </div>
        <!-- Hidden for a vendor that stores its own weights by name
             (hasModels=false): a file picker over no files is a control over
             nothing. Shown while the list is still loading, so the page does
             not jump when the answer arrives. -->
        {#if activeSttEngine?.hasModels !== false}
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.speechModel')}</div>
            <div class="d">{t('settings.speechModelDesc')}</div>
          </div>
          <!-- A dropdown, not an expanding section: picking a model must not
               shove the rest of the page down. -->
          <div class="tool-setting">
            <button class="ctrl" disabled={speechBusy} onclick={() => (speechOpen = !speechOpen)}>
              {activeSpeechLabel} <Icon name={speechOpen ? 'chevronUp' : 'chevronDown'} size={13} />
            </button>
            {#if speechOpen}
              <button
                class="drop-backdrop"
                aria-label={t('settings.close')}
                onclick={() => (speechOpen = false)}
              ></button>
              <div class="rowdrop-list">
                {#if speechModels.length === 0}
                  <div class="rowdrop-note muted">{t('settings.speechNoModels')}</div>
                {:else}
                  <button
                    class="rowdrop-opt"
                    class:selected={speechModels.every((m) => !m.active)}
                    onclick={() => pickSpeechModel('')}
                  >
                    <div class="t">{t('settings.speechAuto')}</div>
                    <div class="sub">{t('settings.speechAutoDesc')}</div>
                  </button>
                  {#each speechModels as m (m.path)}
                    <div class="rowdrop-row">
                      <button
                        class="rowdrop-opt"
                        class:selected={m.active}
                        onclick={() => pickSpeechModel(m.path)}
                      >
                        <div class="t">{m.name}</div>
                        <div class="sub">{m.sizeMB} MB · {m.store}</div>
                      </button>
                      <!-- data-tip, not title: the app has its own tooltip and
                           the native one is slow and unstyleable. The path is
                           what the tip is for. -->
                      <button
                        class="rowdrop-reveal"
                        data-tip={m.path}
                        aria-label={t('settings.speechOpenFolder')}
                        onclick={() => RevealSpeechModel(m.path)}
                      ><Icon name="folderOpen" size={14} /></button>
                    </div>
                  {/each}
                {/if}
                {#if speechError}<div class="rowdrop-note mset-error">{speechError}</div>{/if}

                <!-- Where the scan looks. Without it a missing model is a dead
                     end; with it, it is "put the file in one of these". -->
                <div class="rowdrop-sep"></div>
                <div class="rowdrop-note muted">{t('settings.speechScanned')}</div>
                {#each speechDirs as d (d.path)}
                  <button class="rowdrop-opt rowdrop-dir" onclick={() => OpenSpeechModelDir(d.path)}>
                    <Icon name="folderOpen" size={13} /> {d.label}
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        </div>
        {/if}
        <!-- The NAMED-model pick, for vendors whose models are API names
             rather than files (whisper-1 vs gpt-4o-transcribe). Drawn only
             when the vendor really offers more than one — a picker with a
             single entry is not a choice. -->
        {#if (activeSttEngine?.models?.length ?? 0) > 1}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.voiceModel')}</div>
              <div class="d">{t('settings.voiceModelDesc')}</div>
            </div>
            <select class="ctrl" disabled={voicePageBusy} value={sttModelPick} onchange={(e) => pickSttModelName(e.currentTarget.value)}>
              {#each activeSttEngine?.models ?? [] as m (m)}<option value={m}>{m}</option>{/each}
            </select>
          </div>
        {/if}
      </div>

      <div class="group-head"><span class="group-title">{t('settings.ttsHeading')}</span></div>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.ttsEngine')}</div>
            <div class="d">{t('settings.ttsEngineDesc')}</div>
            {#if ttsStatus && voiceInstallBusy !== 'tts'}<div class="d mset-error">{ttsStatus}</div>{/if}
            {@render voiceInstall('tts', activeTtsEngine, ttsStatus)}
          </div>
          <select class="ctrl" disabled={voicePageBusy} value={ttsPick} onchange={(e) => pickTtsEngine(e.currentTarget.value)}>
            {#each ttsEngines as eng (eng.id)}<option value={eng.id}>{eng.label}</option>{/each}
          </select>
        </div>
        <!-- Same rule as the STT side: only vendors with a real choice of
             named models get this row. -->
        {#if (activeTtsEngine?.models?.length ?? 0) > 1}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.voiceModel')}</div>
              <div class="d">{t('settings.voiceModelDesc')}</div>
            </div>
            <select class="ctrl" disabled={voicePageBusy} value={ttsModelPick} onchange={(e) => pickTtsModelName(e.currentTarget.value)}>
              {#each activeTtsEngine?.models ?? [] as m (m)}<option value={m}>{m}</option>{/each}
            </select>
          </div>
        {/if}
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.ttsVoice')}</div>
            <div class="d">{t('settings.ttsVoiceDesc')}</div>
          </div>
          <!-- ลองฟัง runs the exact path the chat's ฟัง button takes, so what
               it proves is what the user will get. -->
          <button data-guide="settings.voice.test_play" class="ctrl" disabled={voicePageBusy || !!ttsStatus} onclick={previewTts}>
            {ttsPreviewing ? t('settings.ttsPreviewStop') : t('settings.ttsPreview')}
          </button>
          <select class="ctrl" disabled={voicePageBusy || ttsVoicesList.length === 0} value={ttsVoicePick} onchange={(e) => pickTtsVoice(e.currentTarget.value)}>
            <option value="">{t('settings.ttsVoiceAuto')}</option>
            {#each ttsVoicesList as v (v.id)}<option value={v.id}>{v.name}{v.lang ? ` (${v.lang})` : ''}</option>{/each}
          </select>
        </div>
        <!-- Last on this card for the same reason the mic is first: read
             top to bottom, each card is the signal's own path. -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.audioOutput')}</div>
            <div class="d">{audioDevices.labelled ? t('settings.audioOutputDesc') : t('settings.audioNamesHidden')}</div>
          </div>
          <select class="ctrl" value={audioDevices.speakerId} onchange={(e) => setSpeakerId(e.currentTarget.value)}>
            <option value="">{t('settings.audioDefault')}</option>
            {#each audioDevices.speakers as d (d.id)}<option value={d.id}>{d.label}</option>{/each}
          </select>
        </div>
      </div>
    {:else if active === 'image'}
      <h2>{t('settings.image')}</h2>
      <p class="muted set-sub">{t('settings.imageDesc')}</p>

      {#if imagePageError}<div class="mset-error">{imagePageError}</div>{/if}

      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.imageEngine')}</div>
            <!-- ONE line under the title, and it is the vendor's own, not a
                 standing paragraph of advice. There were two: this row carried
                 a generic sentence about keys AND the vendor's note under it,
                 stacked over a dropdown whose label already said which of them
                 needs a key (owner, 7 ก.ย.: "เอาแค่ผู้ให้บริการก็พอ เขียน
                 รายละเอียดสะยาวเลย"). The generic one is gone; internal/imagegen
                 shortened the other to a line. -->
            {#if activeImageEngine?.install}<div class="d">{activeImageEngine.install}</div>{/if}
            <!-- Why the picked vendor cannot run, which for every cloud row
                 here is a missing key. The engine states the FACT and this
                 states the way out — as a button, because a sentence spelling
                 out a path the app can simply walk you down is a worse version
                 of the same thing (owner, 7 ก.ย.).
                 The imagegen row id and the provider id are deliberately the
                 same string, which is what lets one click land on the right
                 provider rather than on the models page in general. -->
            {#if imageStatus}
              <div class="d mset-error">{imageStatus}</div>
              <!-- The same click lands on the same provider row either way;
                   only the verb differs. The codex row has no key to add —
                   it rides the ChatGPT sign-in, and its row on the models
                   page is where that sign-in is made. -->
              {#if activeImageEngine && activeImageEngine.id !== 'pollinations'}
                <button class="ctrl ctrl-icon" onclick={() => goToProviderKey(activeImageEngine!.id)}>
                  <Icon name="brain" size={13} /> {t(activeImageEngine.id === 'codex' ? 'settings.imageSignIn' : 'settings.imageAddKey')}
                </button>
              {/if}
            {/if}
          </div>
          <select class="ctrl" disabled={imagePageBusy} value={imagePick} onchange={(e) => pickImageEngine(e.currentTarget.value)}>
            {#each imageEngines as eng (eng.id)}<option value={eng.id}>{eng.label}</option>{/each}
          </select>
        </div>
        <!-- Same rule the two voice pickers follow: a vendor with one model has
             no choice to offer, and a dropdown with one entry is not a choice. -->
        {#if (activeImageEngine?.models?.length ?? 0) > 1}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.imageModel')}</div>
              <div class="d">{t('settings.imageModelDesc')}</div>
            </div>
            <select class="ctrl" disabled={imagePageBusy} value={imageModelPick} onchange={(e) => pickImageModelName(e.currentTarget.value)}>
              {#each activeImageEngine?.models ?? [] as m (m)}<option value={m}>{m}</option>{/each}
            </select>
          </div>
        {/if}
      </div>

    {:else if active === 'studio'}
      <h2>{t('settings.studio')}</h2>
      <p class="muted set-sub">{t('settings.studioDesc')}</p>

      <!-- Where the material goes: the two agents that use it, one press away. -->
      <div class="studio-agents">
        <button class="ctrl ctrl-primary" onclick={() => talkToVideoAgent('video')}><Icon name="clapperboard" size={14} /> {t('settings.studioTalkVideo')}</button>
        <button class="ctrl" onclick={() => talkToVideoAgent('editor')}><Icon name="scissors" size={14} /> {t('settings.studioTalkEditor')}</button>
        <span class="d">{t('settings.studioTalkNote')}</span>
      </div>

      {#if studioError}<div class="mset-error">{studioError}</div>{/if}

      <!-- The app's own segmented tab bar (.ag-tabs-bar .seg, the one the
           agent editor uses), one tab per kind. The first draft drew seven
           counts in seven boxes and the owner read it as a dashboard; the
           second drew its own tab shapes and they overran the row. The
           standard control is the answer to both. Empty kinds stay, dimmed
           and unpressable: the row is also the list of what a shelf can hold. -->
      <div class="ag-tabs-bar studio-tabs-bar">
        <div class="seg studio-seg" role="tablist" aria-label={t('settings.studioKindsLabel')}>
          {#if studioLoading}
            {#each STUDIO_KINDS as k (k)}<button type="button" class="skeleton" disabled aria-hidden="true"><span class="sk sk-k"></span></button>{/each}
          {:else}
            {#each STUDIO_KINDS as k (k)}
              {@const on = studioBrowse?.kind === k}
              <button type="button" role="tab" aria-selected={on} class:on class:zero={studioTotals[k] === 0}
                disabled={studioTotals[k] === 0} onclick={() => browseKind(k)}
                title={studioTotals[k] === 0 ? t('settings.studioKindEmpty', { kind: studioKindLabel(k) }) : t('settings.studioBrowseKind', { kind: studioKindLabel(k) })}>
                <Icon name={STUDIO_KIND_ICON[k]} size={14} />
                <span class="studio-tab-label">{studioKindLabel(k)}</span>
                <span class="ag-count studio-tab-count">{studioTotals[k].toLocaleString()}</span>
              </button>
            {/each}
          {/if}
        </div>
      </div>
      {#if studioBrowse}
        <StudioBrowser kind={studioBrowse.kind} library={studioBrowse.library} onClose={() => (studioBrowse = null)} onClearLibrary={() => browseStudio(studioBrowse?.kind ?? '', '')} />
      {:else if !studioLoading}
        <!-- Closed on purpose: say where the content went, in the panel's place. -->
        <button class="studio-reopen" onclick={() => browseStudio(STUDIO_KINDS.find((k) => studioTotals[k] > 0) ?? '')}>{t('settings.studioReopen')}</button>
      {/if}
      <div class="ag-band studio-band">
        <span class="lab">{t('settings.studioShelves')}</span><span class="n">{studioLibs.length}</span>
        <span class="rule"></span>
        <button class="ctrl" onclick={() => (studioSourcesOpen = true)}><Icon name="globe" size={14} /> {t('settings.studioSources')}</button>
        <button class="ctrl ctrl-primary" disabled={studioScanning} onclick={addStudioFolder}><Icon name="plus" size={14} /> {t('settings.studioAdd')}</button>
      </div>

      {#if studioResult}
        <!-- Said once, right after the scan: what the app made of the folder. -->
        <div class="studio-result">
          <Icon name="check" size={16} />
          <div class="body">
            <div class="t">{t('settings.studioResult', { n: studioResult.files.toLocaleString(), size: gb(studioResult.bytes) })}</div>
            <div class="chair-chips">
              {#each studioKindChips(studioResult.counts) as c (c.k)}<span class="chip">{studioKindLabel(c.k)} {c.n.toLocaleString()}</span>{/each}
            </div>
          </div>
          <button class="ctrl" onclick={() => (studioResult = null)}>{t('settings.studioClose')}</button>
        </div>
      {/if}

      <div class="office-grid studio-grid">
        {#if studioLoading}
          {#each [0, 1] as i (i)}<article class="chair-card agc studio-card skeleton" aria-hidden="true"><div class="chair-body"><span class="sk sk-title"></span><span class="sk sk-line"></span><span class="sk sk-chips"></span></div></article>{/each}
        {/if}
        {#if studioScanning}
          <article class="chair-card agc studio-card studio-scanning">
            <div class="chair-body">
              <div class="chair-who">
                <span class="cap-mark logo studio-mark"><Icon name="folderOpen" size={20} /></span>
                <span class="chair-name"><span class="nm">{studioProgress ? studioFolderName(studioProgress.root) : t('settings.studioScanning')}</span></span>
              </div>
              <div class="studio-bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={studioProgress && studioProgress.total ? Math.round(studioProgress.done * 100 / studioProgress.total) : 0}>
                <span style:width={studioProgress && studioProgress.total ? `${studioProgress.done * 100 / studioProgress.total}%` : '0%'}></span>
              </div>
              <div class="d">
                {#if studioProgress}{t('settings.studioScanning')} · {studioProgress.done.toLocaleString()} / {studioProgress.total.toLocaleString()}{:else}{t('settings.studioCounting')}{/if}
              </div>
            </div>
            <div class="chair-foot">
              <button class="ctrl" onclick={() => void CancelStudioScan()}>{t('settings.studioCancel')}</button>
            </div>
          </article>
        {/if}
        {#each studioLibs as lib (lib.id)}
          <article class="chair-card agc studio-card" class:off={lib.missing}>
            <div class="chair-body">
              <div class="chair-who">
                <span class="cap-mark logo studio-mark" class:builtin={lib.builtin}><Icon name={lib.builtin ? 'clapperboard' : 'folderOpen'} size={20} /></span>
                <span class="chair-name">
                  <span class="nm">{lib.name}</span>
                  <span class="studio-tr">{lib.builtin ? t('settings.studioBuiltin') : t('settings.studioYours')}</span>
                </span>
              </div>
              {#if !lib.builtin}<p class="chair-desc studio-path" title={lib.root}>{lib.root}</p>{/if}
              {#if lib.builtin}<p class="chair-desc">{t('settings.studioBuiltinDesc')}</p>{/if}
              <div class="chair-chips">
                <span class="chip">{t('settings.studioFiles', { n: lib.files.toLocaleString() })}</span>
                <span class="chip">{gb(lib.bytes)}</span>
                {#each studioKindChips(lib.counts) as c (c.k)}<span class="chip">{studioKindLabel(c.k)} {c.n.toLocaleString()}</span>{/each}
                {#if lib.license}<span class="chip mine">{lib.license}</span>{/if}
                {#if lib.unread > 0}<span class="chip deny">{t('settings.studioUnread', { n: lib.unread })}</span>{/if}
              </div>
              <div class="chair-stat" class:studio-ok={!lib.missing}>
                {#if lib.missing}{t('settings.studioMissing')}{:else}{t('settings.studioReady')}{/if}
              </div>
            </div>
            <div class="chair-foot">
              <button class="ctrl" class:ctrl-primary={studioBrowse?.library === lib.id} disabled={lib.missing || lib.files === 0} onclick={() => browseStudio('', lib.id)}><Icon name="search" size={13} /> {t('settings.studioBrowse')}</button>
              <button class="ctrl ctrl-icon" title={t('settings.studioReveal')} aria-label={t('settings.studioReveal')} onclick={() => void RevealStudioLibrary(lib.id)}><Icon name="folderOpen" size={14} /></button>
              {#if lib.builtin}
                {#if lib.source}<button class="linklike" onclick={() => BrowserOpenURL(lib.source ?? '')}>{t('settings.studioSourcePage')}</button>{/if}
              {:else}
                <button class="ctrl" disabled={studioScanning} onclick={() => rescanStudio(lib.id)}>{t('settings.studioRescan')}</button>
                <button class="ctrl ctrl-danger" disabled={studioScanning} onclick={() => removeStudio(lib)}>{t('settings.studioRemove')}</button>
              {/if}
            </div>
          </article>
        {/each}
      </div>

      {#if studioSourcesOpen}
        <StudioSourcesSheet imported={studioImported} scanning={studioScanning} onAddFolder={addStudioFolder} onClose={() => (studioSourcesOpen = false)} />
      {/if}

    {:else if active === 'team' || active === 'agents'}
      {@const kind = active === 'team' ? 'agent' : 'helper'}
      {#if agentEditing !== null}
        {@render agentEditorPane()}
      {:else if kind === 'helper'}
        {@render profileListPane(kind)}
      {:else if intentPending}
        <!-- A gear was pressed and its editor is still being read off the disk.
             The frame that a heading and a paragraph would fill is the frame
             the editor is about to take, so the wait draws as a shape, not as
             a page saying something else. -->
        <div class="mset-skeleton" aria-label={t('settings.loading')}>
          <span class="sk sk-head"></span>
          <span class="sk sk-line"></span>
          <span class="sk sk-line short"></span>
          <span class="sk sk-block"></span>
        </div>
      {:else}
        {@render profileListPane(kind)}
      {/if}

    {:else if active === 'avatar'}
      <AvatarSettings />
    {:else if active === 'teams'}
      <TeamSettings onNewAgent={newAgentFromTeams} />
    {:else if active === 'main'}
      {#if mainHead === null}
        <h2>{t('settings.mainHeads')}</h2>
        <p class="muted set-sub">{t('settings.mainHeadsDesc')}</p>
        <!-- The agent list's card (agentRow), the head's own face in it. Two,
             never a third: the office desk is what an agent's own chat runs
             on and every agent is a card of its own already. Two cards on a
             page of their own get the page: one column each, the face at the
             roster's stage size, the desk's sentence unclamped (owner,
             14 ก.ย.: "มี 2 ตัว ทำตัวใหญ่กว่านี้ … อยากให้หน้านี้เห็นแค่ 2 ตัว").
             The rank rides on the face's corner (RankedFace, style E) the way
             it does on every other face in the app. No memory count on the
             card: the number said nothing a person acts on from here, and
             the memory tab is one click in. -->
        <div class="office-grid main-grid">
          {#each HEADS as h (h)}
            <div class="chair-card agc main-card" role="button" tabindex="0"
              onclick={() => openHead(h)} onkeydown={(e) => { if (e.key === 'Enter') openHead(h) }}>
              <div class="chair-body">
                <div class="chair-who">
                  <RankedFace tier="head" size={72}><Mascot {...headOptions(h)} pose="idle" size={72} still /></RankedFace>
                  <span class="chair-name">{headShown(h)}{#if headNames[h]} <span class="badge on">{headLabel(h)}</span>{/if}</span>
                  <div class="ag-actions">
                    <button class="icobtn tiny tip-l" aria-label={t('settings.agentConfigure')} data-tip={t('settings.agentConfigure')}
                      onclick={(e) => { e.stopPropagation(); openHead(h) }}>
                      <Icon name="settings" size={14} />
                    </button>
                  </div>
                </div>
                <div class="d">{headDesc(h)}</div>
                {#if headPending(h).length > 0}
                  <div class="chips">
                    <span class="tag main-tag-warn">{t('settings.mainPendingN', { n: headPending(h).length })}</span>
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
        <p class="office-note">
          {t('settings.mainSharedNote')}
          <button type="button" class="linklike" onclick={() => openSection('you')}>{t('settings.you')}</button>
        </p>
      {:else}
        {@const h = mainHead}
        {@const g = headGroup(h)}
        <!-- The same shell as a พนักงาน's or a ลูกมือ's page (profileHero):
             the bar, the head, the tabs. Only the bar's right side and the
             tab set are this page's own. -->
        <div class="pp-bar pf-bar">
          <button class="ctrl" onclick={() => (mainHead = null)}><Icon name="arrowLeft" size={14} /> {t('settings.agentBack')}</button>
          <div class="pp-bar-gap"></div>
          <button data-guide="settings.head.switch_desk" class="ctrl" onclick={() => openHead(h === 'assistant' ? 'coding' : 'assistant')}>
            {t('settings.mainGoOther', { name: headLabel(h === 'assistant' ? 'coding' : 'assistant') })} <Icon name="arrowRight" size={12} />
          </button>
        </div>
        {@render profileHero('head', headShown(h), t('settings.mainHeadBadge'), headDesc(h), h)}
        {#if learningError}<div class="mset-error">{learningError}</div>{/if}

        <div class="ag-tabs-bar">
          <div class="seg" role="tablist" aria-label={t('settings.mainEditTitle', { name: headLabel(h) })}>
            {#each [
              ['identity', 'userRound', t('settings.agentSecIdentity')],
              ['mcp', 'plug', t('settings.mainSecMcp')],
              ['skills', 'puzzle', t('settings.mainSecSkills')],
              ['opening', 'messageSquare', t('settings.agentSecOpening')],
              ['memory', 'brain', t('settings.mainSecMemory')],
            ] as [id, icon, label] (id)}
              {#if id === 'identity'}
                <button data-guide="settings.head.tab.identity" type="button" role="tab" aria-selected={mainTab === id} class:on={mainTab === id} onclick={() => (mainTab = id as MainTab)}>
                  <Icon name={icon as IconName} size={14} /><span>{label}</span>
                </button>
              {:else if id === 'mcp'}
                <button data-guide="settings.head.tab.mcp" type="button" role="tab" aria-selected={mainTab === id} class:on={mainTab === id} onclick={() => (mainTab = id as MainTab)}>
                  <Icon name={icon as IconName} size={14} /><span>{label}</span>
                </button>
              {:else if id === 'skills'}
                <button data-guide="settings.head.tab.skills" type="button" role="tab" aria-selected={mainTab === id} class:on={mainTab === id} onclick={() => (mainTab = id as MainTab)}>
                  <Icon name={icon as IconName} size={14} /><span>{label}</span>
                </button>
              {:else if id === 'opening'}
                <button data-guide="settings.head.tab.dialogue" type="button" role="tab" aria-selected={mainTab === id} class:on={mainTab === id} onclick={() => (mainTab = id as MainTab)}>
                  <Icon name={icon as IconName} size={14} /><span>{label}</span>
                </button>
              {:else if id === 'memory'}
                <button data-guide="settings.head.tab.memory" type="button" role="tab" aria-selected={mainTab === id} class:on={mainTab === id} onclick={() => (mainTab = id as MainTab)}>
                  <Icon name={icon as IconName} size={14} /><span>{label}</span>
                  {#if headPending(h).length > 0}<span class="ag-count ag-count-warn">{headPending(h).length}</span>{/if}
                </button>
              {/if}
            {/each}
          </div>
        </div>

        <!-- ตัวตน: the desk file (modes/<head>.md, whole, with a way back to
             the bundled one), then the head's own identity files — one folder
             per head (config.IdentityDirFor), moved whole from ตั้งค่า ›
             คำสั่งประจำตัว on 14 ก.ย. 2026. Both editors are a row until asked
             for; the "add a file" box did not come (owner: "เอา เพิ่มไฟล์
             คำสั่งใหม่ ออก"). -->
        <div class="ag-tab-panel" class:on={mainTab === 'identity'}>
          <div class="settings-card">
            <div class="set-row">
              <div class="set-txt">
                <div class="t">{t('settings.mainHeadName')}</div>
                <div class="d">{t('settings.mainHeadNameHint', { desk: headLabel(h) })}</div>
              </div>
              <!-- A button, not save-on-blur (owner, 14 ก.ย. 2026: "ควรจะมี กดบันทึก
                   ด้วยปุ่ม"): the name reaches every open chat's prompt the
                   moment it lands, so landing is a deliberate press. -->
              <div class="chair-name-edit">
                <input class="ctrl key-input" placeholder="Aetox" bind:value={headNameDraft}
                  onkeydown={(e) => { if (e.key === 'Enter') saveHeadName(h) }} aria-label={t('settings.mainHeadName')} />
                <button data-guide="settings.head.save" type="button" class="ctrl ctrl-primary" disabled={headNameDraft.trim() === (headNames[h] ?? '')}
                  onclick={() => saveHeadName(h)}>{t('settings.save')}</button>
              </div>
            </div>
          </div>
          <h3 class="set-h3">{t('settings.mainDeskFile')}</h3>
          <p class="muted set-sub">{t('settings.mainDeskFileHint')}</p>
          <div class="settings-card">
            <div class="set-row">
              <div class="set-txt">
                <div class="t"><span class="mono-dim you-file">modes/{h}.md</span>
                  {#if deskFile?.overrides}<span class="badge on">{t('settings.mainDeskOverrides')}</span>{/if}
                </div>
                <div class="d">{t('settings.mainDeskWhat')}</div>
              </div>
              <div class="set-ctrl" style="display:flex; align-items:center; gap:8px;">
                {#if deskFile?.overrides}
                  <button data-guide="settings.head.reset" type="button" class="ctrl" disabled={deskBusy} onclick={() => askResetDeskFile(h)}>
                    <Icon name="rotateCw" size={13} /> {t('settings.mainDeskReset')}
                  </button>
                {/if}
                <button type="button" class="ctrl" class:ctrl-primary={deskOpen} disabled={deskFile === null} onclick={() => (deskOpen = !deskOpen)}>
                  <Icon name="pencil" size={13} /> {t('settings.identityEditBtn')}
                </button>
              </div>
            </div>
            {#if deskError}<div class="mset-error you-inline-error">{deskError}</div>{/if}
            {#if deskMsg && !deskOpen}<div class="d muted you-inline-note">{deskMsg}</div>{/if}
          </div>
          {#if deskOpen && deskFile}
            <div class="settings-card card-form">
              <textarea class="identity-input desk-file" bind:value={deskDraft} spellcheck="false"
                aria-label={`modes/${h}.md`}></textarea>
              <div class="you-save">
                <span class="d muted">{deskMsg || t('settings.mainDeskEditHint')}</span>
                <button type="button" class="ctrl ctrl-primary" disabled={!deskDirty || deskBusy} onclick={() => saveDeskFile(h)}>
                  {deskBusy ? t('settings.saving') : t('settings.save')}
                </button>
              </div>
            </div>
          {/if}

          <h3 class="set-h3">{t('settings.identity')}</h3>
          <p class="muted set-sub">{t('settings.mainIdentityHint', { name: headLabel(h) })}</p>
          <div class="settings-card">
            {#each recommendedIdentityTemplates as item (item.name)}
              {@const exists = (identity.files || []).some((f) => f.name === item.name)}
              {@const isActive = identity.activeName === item.name}
              <div class="set-row">
                <div class="set-txt">
                  <div class="t"><span class="mono-dim you-file">{item.name}</span>
                    {#if isActive}<span class="badge on">{t('settings.identityEditingNow')}</span>{/if}
                  </div>
                  <div class="d">{t(item.descKey, { name: headLabel(h) })}</div>
                </div>
                <div class="set-ctrl">
                  {#if exists}
                    <button type="button" class="ctrl" class:ctrl-primary={isActive} onclick={() => (isActive ? closeIdentityFile() : openIdentityFile(item.name))}>
                      <Icon name="pencil" size={13} />
                      {t('settings.identityEditBtn')}
                    </button>
                  {:else}
                    <button type="button" class="ctrl" onclick={() => createIdentityFile(item.name, tplFor(item))}>
                      <Icon name="plus" size={13} />
                      {t('settings.identityCreateBtn')}
                    </button>
                  {/if}
                </div>
              </div>
            {/each}
            {#each customIdentityFiles as f (f.name)}
              {@const isActive = identity.activeName === f.name}
              <div class="set-row">
                <div class="set-txt">
                  <div class="t"><span class="mono-dim you-file">{f.name}</span></div>
                  <div class="d">{t('settings.identityCustomFiles')}</div>
                </div>
                <div class="set-ctrl" style="display:flex; align-items:center; gap:8px;">
                  <button type="button" class="ctrl" class:ctrl-primary={isActive} onclick={() => (isActive ? closeIdentityFile() : openIdentityFile(f.name))}>
                    <Icon name="pencil" size={13} />
                    {t('settings.identityEditBtn')}
                  </button>
                  <button type="button" class="ctrl" style="color:var(--status-danger);" aria-label={t('settings.remove')} onclick={() => removeIdentityFile(f.name)}>
                    <Icon name="trash" size={13} />
                  </button>
                </div>
              </div>
            {/each}
          </div>

          {#if identity.activeName}
            <div class="group-head" style="display:flex; justify-content:space-between; align-items:center;">
              <div style="display:flex; align-items:baseline; gap:8px;">
                <span class="group-title">{t('settings.identityEditing', { name: identity.activeName })}</span>
                {#if identityDirty}
                  <span class="group-count" style="color:var(--status-warning, #e3b341); font-weight:600;">{t('settings.identityUnsaved')}</span>
                {:else}
                  <span class="group-count" style="color:var(--text-dim);">{t('settings.identitySaved')}</span>
                {/if}
              </div>
              <button type="button" class="ctrl" style="color:var(--status-danger);" onclick={() => removeIdentityFile(identity.activeName)}>
                <Icon name="trash" size={13} />
                {t('settings.remove')}
              </button>
            </div>
            <div class="settings-card card-form">
              <textarea class="identity-input" placeholder={t('settings.identityPlaceholder')} bind:value={identity.draft}
                aria-label={identity.activeName}></textarea>
              <div style="display:flex; justify-content:flex-end;">
                <button type="button" class="ctrl identity-save ctrl-primary" disabled={!identityDirty || identity.saving} onclick={saveIdentityFile}>
                  {identity.saving ? t('settings.saving') : t('settings.save')}
                </button>
              </div>
            </div>
          {/if}
        </div>

        <!-- ตั้งค่า MCP: the agent box's shape — every live server with this
             desk's switch on it, writing at once through the room's one
             writer. The room's picker is the same switch on the same call. -->
        <div class="ag-tab-panel" class:on={mainTab === 'mcp'}>
          <div class="settings-card">
            <div class="set-row">
              <span class="ag-rowicon"><Icon name="plug" size={15} /></span>
              <div class="set-txt">
                <div class="t">{t('settings.mainMcpTitle', { name: headLabel(h) })} {#if mcpLoaded}<span class="ag-count">{headServerCount(h)}</span>{/if}</div>
                <div class="d">{t('settings.mainMcpHint')}</div>
              </div>
              <button class="ctrl" onclick={() => openCapabilityAt('mine')}>{t('capability.navMine')} <Icon name="arrowRight" size={13} /></button>
            </div>
            {#if mcpError}<div class="mset-error">{mcpError}</div>{/if}
            {#if !mcpLoaded}
              {@render waitRow()}
            {:else}
              {#each liveServers as srv (srv.name)}
                {@const on = isOnHead(srv, h)}
                <label class="set-row ag-reachrow" class:on>
                  <McpMark name={srv.name} size={26} />
                  <div class="set-txt">
                    <div class="t">{srv.name}</div>
                    <div class="d">{srv.tools > 0 ? t('settings.agentMCPTools', { n: srv.tools }) : (srv.url || (srv.command ?? []).join(' '))}</div>
                  </div>
                  <span class="mswitch">
                    <input type="checkbox" checked={on} disabled={mcpBusy !== ''} aria-label={srv.name} onchange={() => toggleHeadMCP(srv, h)} />
                    <span></span>
                  </span>
                </label>
              {/each}
              <div class="set-row"><div class="set-txt"><div class="d">
                {liveServers.length === 0 ? t('settings.agentMCPNoneInSystem') : t('settings.agentMCPLiveHint')}
              </div></div></div>
            {/if}
          </div>
          <div class="settings-card">
            <div class="set-row">
              <div class="set-txt"><div class="t">{t('capability.navTools')}</div><div class="d">{t('settings.mainReachTools')}</div></div>
              <button type="button" class="ctrl" onclick={() => openCapabilityAt('tools')}><Icon name="wrench" size={13} /> {t('settings.mainReachOpen')}</button>
            </div>
          </div>
        </div>

        <!-- สกิล: what this desk knows — the whole shelf, because every desk
             carries every skill (mode.go). Listed, not ticked: there is no
             per-desk list to edit, and the room is where the shelf changes. -->
        <div class="ag-tab-panel" class:on={mainTab === 'skills'}>
          <div class="settings-card">
            <div class="set-row">
              <span class="ag-rowicon"><Icon name="puzzle" size={15} /></span>
              <div class="set-txt">
                <div class="t">{t('settings.mainSkillsTitle', { name: headLabel(h) })} {#if shelfLoaded}<span class="ag-count">{shelfSkills.length}</span>{/if}</div>
                <div class="d">{t('settings.mainSkillsHint')}</div>
              </div>
              <button class="ctrl" onclick={() => openCapabilityAt('skills')}>{t('capability.navSkills')} <Icon name="arrowRight" size={13} /></button>
            </div>
            {#if !shelfLoaded}
              {@render waitRow()}
            {:else}
              <!-- The agent skills box's own rows, exactly (owner, 14 ก.ย.:
                   "ควรจะเป็นมาตรฐานเดียวกันทั้งหมด … CSS แบบนี้ดีกว่า"): the mark,
                   the name with มากับแอป on a bundled one, two lines of what
                   it does; yours first, and the same search over six. -->
              {#if shelfSkills.length > 6}
                <div class="set-row">
                  <label class="ag-search">
                    <Icon name="search" size={13} />
                    <input bind:value={shelfQuery} placeholder={t('settings.agentSkillsSearch')} />
                  </label>
                </div>
              {/if}
              {#each [...shelfShown.filter((s) => !s.bundled), ...shelfShown.filter((s) => s.bundled)] as sk (sk.name)}
                <div class="set-row">
                  <span class="cap-mark" style="--px:26px; --h:{coverHue(sk.name)}" aria-hidden="true">{sk.name.replace(/^aetox-/, '').slice(0, 2)}</span>
                  <div class="set-txt">
                    <div class="t">{sk.name.replace(/^aetox-/, '')} {#if sk.bundled}<span class="badge on">{t('office.builtin')}</span>{/if}</div>
                    {#if sk.description}<div class="d clamp2">{sk.description}</div>{/if}
                  </div>
                </div>
              {/each}
              {#if shelfSkills.length === 0 || shelfShown.length === 0}
                <div class="set-row"><div class="set-txt"><div class="d">
                  {shelfSkills.length === 0 ? t('settings.agentSkillsNoneOnShelf') : t('settings.agentNoMatches')}
                </div></div></div>
              {/if}
            {/if}
          </div>
        </div>

        <!-- เปิดบทสนทนา: the same form a worker's opening is edited on, aimed
             at modes/<desk>/STARTERS.md. {ชื่อ} in the headline is where the
             person's name goes (Chat.svelte withName). -->
        <div class="ag-tab-panel" class:on={mainTab === 'opening'}>
          <p class="muted set-sub" style="margin-top:0">{t('settings.mainOpeningHint')}</p>
          {@render agentStartersBox()}
        </div>

        <!-- ความจำ: this head's file, the projects it hosts, its queue and
             its history — moved whole from การเรียนรู้. -->
        <div class="ag-tab-panel" class:on={mainTab === 'memory'}>
          {#if headPending(h).length > 0}
            <h3 class="set-h3">{t('settings.learningPending')}</h3>
            <p class="muted set-sub">{t('settings.learningPendingHint')}</p>
            <div class="settings-card">
              {#each headPending(h) as c (c.id)}{@render pendingRow(c)}{/each}
            </div>
          {/if}
          <!-- Two layers, drawn as two (owner, 14 ก.ย.: "โค้ดควรมีความจำของตัวเอง
               แยกชั้นกับความจำในโปรเจกต์อีกที"): what this head learned across
               every project is one card under its own title; what a project
               taught it is a card per project under a second title, beneath.
               They were one card with the projects nested inside, which read
               as one memory with sub-folders. -->
          <h3 class="set-h3">{t('settings.mainMemoryOwn', { name: headLabel(h) })}</h3>
          <p class="muted set-sub">{t('settings.mainMemoryOwnHint')}</p>
          {#each [g] as group (group.scope)}
            <div class="settings-card mem-desk">
              {@render deskHead(group)}
              {#if moveError?.scope === group.scope}
                <div class="mem-move-error"><Icon name="alertTriangle" size={13} /><span>{moveError.text}</span></div>
              {/if}
              {#each group.lines as line, i (i)}
                {@render memRow(group, line, i)}
              {/each}
              {#if group.lines.length === 0}
                <div class="empty">{group.projectsUnder ? t('settings.memoryDeskEmpty') : t('settings.learningAssistantEmpty')}</div>
              {/if}
            </div>
          {/each}

          {#if g.scope === projectsHost}
            <h3 class="set-h3">{t('settings.mainMemoryProjects')}</h3>
            <p class="muted set-sub">{t('settings.mainMemoryProjectsHint')}</p>
            {#each projectGroups as project (project.scope)}
              <div class="settings-card mem-desk mem-project">
                {@render deskHead(project)}
                {#if project.orphan && adoptOpen === project.scope}
                  <div class="mem-adopt">
                    {#each knownProjects as p (p.rootPath)}
                      <button type="button" class="ctrl tiny" onclick={() => adoptScope(project.scope, p.rootPath)}>{p.name}</button>
                    {/each}
                  </div>
                {/if}
                {#if moveError?.scope === project.scope}
                  <div class="mem-move-error"><Icon name="alertTriangle" size={13} /><span>{moveError.text}</span></div>
                {/if}
                {#each project.lines as line, i (i)}
                  {@render memRow(project, line, i)}
                {/each}
                {#if project.lines.length === 0}
                  <div class="empty">{t('settings.learningAssistantEmpty')}</div>
                {/if}
              </div>
            {/each}
            {#if projectGroups.length === 0}
              <div class="settings-card"><div class="empty">{t('settings.mainMemoryProjectsEmpty')}</div></div>
            {/if}
          {/if}
          {#if memoryScopeError}
            <div class="set-error">{memoryScopeError}</div>
          {/if}
          <div class="settings-card">
            <div class="set-row learn-foot">
              <div class="set-txt"><div class="d muted">{t('settings.mainMemoryFolderHint')}</div></div>
              <button type="button" class="ctrl" onclick={() => OpenMemoryFolder()}>
                <Icon name="folderOpen" size={13} /> {t('settings.learningOpenFolder')}
              </button>
            </div>
          </div>
          {#if headDecided(h).length > 0}
            <h3 class="set-h3">{t('settings.learningHistory')}</h3>
            <p class="muted set-sub">{t('settings.learningHistoryHint')}</p>
            <div class="settings-card">
              {#each decidedExpanded ? headDecided(h) : headDecided(h).slice(0, DECIDED_PREVIEW) as c (c.id)}{@render decidedRow(c)}{/each}
              {#if headDecided(h).length > DECIDED_PREVIEW}
                <button type="button" class="learn-more" onclick={() => (decidedExpanded = !decidedExpanded)}>
                  <Icon name={decidedExpanded ? 'chevronUp' : 'chevronDown'} size={12} />
                  {decidedExpanded
                    ? t('settings.learningHistoryLess')
                    : t('settings.learningHistoryMore', { n: headDecided(h).length - DECIDED_PREVIEW })}
                </button>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
    {:else if active === 'you'}
      <h2>{t('settings.you')}</h2>
      <p class="muted set-sub">{t('settings.youDesc')}</p>
      {#if learningError}<div class="mset-error">{learningError}</div>{/if}

      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.youName')}</div>
            <div class="d">{t('settings.youNameHint')}</div>
          </div>
          <!-- The footer's own store, written the footer's own way (on
               change, fire-and-forget): one name, two doors. -->
          <input data-guide="settings.you.name_input" class="ctrl key-input" placeholder={t('settings.youNamePlaceholder')} bind:value={youName}
            onchange={() => saveProfileName(youName)} aria-label={t('settings.youName')} />
        </div>
      </div>

      <!-- The session review writes USER.md, so its switch sits with the
           file it writes — moved from การเรียนรู้ 14 ก.ย. 2026. -->
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.sessionReviewTitle')}</div>
            <div class="d">{t('settings.sessionReviewHint')}</div>
          </div>
          <label class="mswitch">
            <input type="checkbox" checked={sessionReviewAutoOn} onchange={toggleSessionReviewAuto} />
            <span></span>
          </label>
        </div>
        <div class="set-row">
          <div class="set-txt">
            {#if sessionReviewMsg}<div class="d" style="color:var(--accent)">{sessionReviewMsg}</div>{/if}
          </div>
          <button type="button" class="ctrl" disabled={sessionReviewBusy} onclick={runSessionReviewNow}>
            {sessionReviewBusy ? t('settings.sessionReviewRunning') : t('settings.sessionReviewNow')}
          </button>
        </div>
      </div>

      {#if youPending.length > 0}
        <h3 class="set-h3">{t('settings.learningPending')}</h3>
        <p class="muted set-sub">{t('settings.learningPendingHint')}</p>
        <div class="settings-card">
          {#each youPending as c (c.id)}{@render pendingRow(c)}{/each}
        </div>
      {/if}

      {@const userMeta = scopeMeta(USER_SCOPE)}
      <h3 class="set-h3 mem-header-split">
        <span>{t('settings.learningUserSection')}</span>
        <span class="learn-scope mem-tone-user"><Icon name="circleUser" size={11} /> {userMeta.audience}</span>
        <span class="mem-badge-file">{userMeta.file}</span>
      </h3>
      <p class="muted set-sub">{t('settings.learningUserSectionHint')}</p>
      <!-- Lines about the person that landed in the assistant's file: offered
           here, where they will land, rather than where they are. -->
      {#if userLinesInMain.length > 0}
        <div class="mem-quick-banner you-banner">
          <div class="mem-quick-txt">
            <Icon name="sparkles" size={15} />
            <span>{t('settings.learningQuickMigrateNotice', { count: String(userLinesInMain.length) })}</span>
          </div>
          <button
            type="button"
            class="ctrl tiny ctrl-primary"
            disabled={memorySaving || migrateBusy || isFull(USER_SCOPE)}
            title={isFull(USER_SCOPE) ? t('settings.memoryTargetFull', { name: scopeLabel(USER_SCOPE) }) : undefined}
            onclick={quickMigrateUserLines}
          >
            {t('settings.learningQuickMigrateAction', { count: String(userLinesInMain.length) })}
          </button>
        </div>
      {/if}
      <div data-guide="settings.you.about_input" class="settings-card mem-desk">
        {@render deskHead(userMemoryGroup)}
        {#if moveError?.scope === USER_SCOPE}
          <div class="mem-move-error"><Icon name="alertTriangle" size={13} /><span>{moveError.text}</span></div>
        {/if}
        {#if userMemoryGroup.lines.length > 0}
          {#each userMemoryGroup.lines as line, i (i)}
            {@render memRow(userMemoryGroup, line, i)}
          {/each}
        {:else}
          <div class="empty mem-empty-user">{t('settings.learningUserEmpty')}</div>
        {/if}
        <div class="set-row learn-foot">
          <button type="button" class="ctrl" onclick={() => OpenMemoryFolder()}>
            <Icon name="folderOpen" size={13} /> {t('settings.learningOpenFolder')}
          </button>
        </div>
      </div>

      {#if youDecided.length > 0}
        <h3 class="set-h3">{t('settings.learningHistory')}</h3>
        <p class="muted set-sub">{t('settings.learningHistoryHint')}</p>
        <div class="settings-card">
          {#each youDecided as c (c.id)}{@render decidedRow(c)}{/each}
        </div>
      {/if}

    {:else if active === 'issues'}
      <h2>{t('settings.issues')}</h2>
      <p class="muted set-sub">{t('settings.issuesDesc')}</p>
      <!-- Said here rather than only in the privacy policy, because this is the
           page where the button is. The pre-filled body carries the tool's own
           arguments, and those are the user's file paths -- a report is one
           click from publishing "output/20260823/xiaomi-17t-pro/index.html" to
           a public tracker. Nothing is sent until they submit on GitHub, so the
           fix is telling them what they are about to look at, not hiding it. -->
      <p class="muted set-sub">{t('settings.issuesReportNote')}</p>

      {#if issuesError}<div class="mset-error">{issuesError}</div>{/if}

      <!-- On the page somebody opens when something is wrong. Not instead of
           the issue button below: an issue carries the version and the log,
           the group carries the thing you cannot describe well enough to file
           yet. -->
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.community')}</div>
            <div class="d">{t('settings.communityDesc')}</div>
          </div>
          <button class="ctrl" onclick={() => BrowserOpenURL(COMMUNITY_URL)}>{t('settings.communityOpen')}</button>
        </div>
      </div>

      <div class="settings-card">
        {#each systemIssues as c (c.id)}
          <div class="learn-row">
            <div class="learn-main">
              <div class="learn-head">
                <span class="learn-scope">{scopeLabel(c.scope)}</span>
              </div>
              <div class="learn-body">{c.body}</div>
              {#if c.reason}<div class="learn-why">{c.reason}</div>{/if}
            </div>
            <!-- Asking comes first, and is the primary, because it is the only
                 one of the three that answers the question the other two need
                 answered. Reporting a problem that turns out to be this
                 machine's wastes the developer's time and the user's; waving
                 off one that turns out to be a real bug loses it. -->
            <div class="learn-actions">
              <button type="button" class="ctrl ctrl-primary" disabled={issuesBusy === c.id}
                onclick={() => consultIssue(c)}>{t('settings.issuesConsult')}</button>
              <button type="button" class="ctrl" disabled={issuesBusy === c.id}
                onclick={() => reportIssue(c)}>{t('settings.issuesReport')}</button>
              <button type="button" class="ctrl" disabled={issuesBusy === c.id}
                onclick={() => dismissIssue(c.id)}>{t('settings.issuesDismiss')}</button>
            </div>
          </div>
        {/each}
        {#if systemIssues.length === 0}
          <div class="empty">{t('settings.issuesEmpty')}</div>
        {/if}
      </div>

      <!-- Where a decided row went. The rows were never deleted — only invisible,
           which read as destroyed. Same shape as the learning room's record:
           four, then a count. -->
      {#if decidedIssues.length > 0}
        <h3 class="set-h3">{t('settings.issuesHistory')}</h3>
        <p class="muted set-sub">{t('settings.issuesHistoryHint')}</p>
        <div class="settings-card">
          {#each (issuesExpanded ? decidedIssues : decidedIssues.slice(0, DECIDED_PREVIEW)) as c (c.id)}
            <div class="learn-row past">
              <div class="learn-main">
                <div class="learn-head">
                  <span class="learn-scope">{scopeLabel(c.scope)}</span>
                  <span class="learn-op" class:rejected={c.state !== 'reported'}>
                    {c.state === 'reported' ? t('settings.issuesStateReported') : t('settings.issuesStateDismissed')}
                  </span>
                </div>
                <div class="learn-body">{c.body}</div>
              </div>
            </div>
          {/each}
          {#if decidedIssues.length > DECIDED_PREVIEW}
            <button type="button" class="learn-more" onclick={() => (issuesExpanded = !issuesExpanded)}>
              <Icon name={issuesExpanded ? 'chevronUp' : 'chevronDown'} size={12} />
              {issuesExpanded
                ? t('settings.learningHistoryLess')
                : t('settings.learningHistoryMore', { n: decidedIssues.length - DECIDED_PREVIEW })}
            </button>
          {/if}
        </div>
      {/if}
    {:else if active === 'usage'}
      <h2>{t('settings.usage')}</h2>
      <p class="muted set-sub">{t('settings.usageDesc')}</p>

      {#if usageError}<div class="mset-error">{usageError}</div>{/if}

      <!-- Three states, and the middle one used to be missing. UsageStats walks
           the whole history to build the chart, the heatmap and the streak, and
           that is half a second on this machine's own database and longer while
           the engine is writing to it. All of that time the page said "ยังไม่มี
           ข้อมูลการใช้งาน" — a sentence about the data, printed before anybody
           had asked the data anything.
           The placeholder is the page's own grid rather than a spinner, so the
           numbers land in the boxes that were already holding their place and
           nothing jumps. Shapes and pulse are the model pane's (.sk, sk-pulse);
           only the geometry is this layout's. -->
      {#if usagePending}
        <div class="stat-cards usage-sk" aria-busy="true" aria-label={t('settings.loading')}>
          {#each ['wide', 'wide', 'wide', '', '', '', ''] as w, i (i)}
            <div class="stat-card {w}">
              <span class="sk sk-eyebrow"></span>
              <span class="sk sk-figure"></span>
              <span class="sk sk-line short"></span>
            </div>
          {/each}
        </div>
        <div class="settings-card wide-card usage-sk" aria-hidden="true">
          <div class="card-form"><span class="sk sk-eyebrow"></span><span class="sk sk-chart"></span></div>
        </div>
        <div class="settings-card wide-card usage-sk" aria-hidden="true">
          <div class="card-form"><span class="sk sk-eyebrow"></span><span class="sk sk-heat"></span></div>
        </div>
      {:else if usage && usage.totals.calls > 0}
        {@const tot = usage.totals}
        <div class="stat-cards">
          <div class="stat-card wide">
            <div class="eyebrow">{t('settings.usageTotalTokens')}</div>
            <div class="stat-big">{fmtCompact(tot.promptTokens + tot.completionTokens)}</div>
            <div class="stat-split" aria-hidden="true">
              <span class="seg in" style="flex:{Math.max(tot.promptTokens, 1)}"></span>
              <span class="seg out" style="flex:{Math.max(tot.completionTokens, 1)}"></span>
            </div>
            <div class="stat-legend">
              <span><i class="dot in"></i>{t('settings.usageInput')} {fmtCompact(tot.promptTokens)}</span>
              <span><i class="dot out"></i>{t('settings.usageOutput')} {fmtCompact(tot.completionTokens)}</span>
            </div>
          </div>

          <div class="stat-card wide">
            <div class="eyebrow">{t('settings.usageCacheHitRate')}</div>
            {#if tot.cacheRows === 0}
              <div class="stat-big dim">—</div>
              <div class="stat-sub">{t('settings.usageCacheUnreported')}</div>
            {:else}
              <div class="stat-big">{pct(tot.cachedTokens, tot.promptTokens)}<span class="unit">%</span></div>
              <div class="stat-split" aria-hidden="true">
                <span class="seg hit" style="flex:{Math.max(tot.cachedTokens, 1)}"></span>
                <span class="seg miss" style="flex:{Math.max(tot.uncachedTokens, 1)}"></span>
              </div>
              <div class="stat-legend">
                <span><i class="dot hit"></i>{t('settings.usageHit')} {fmtCompact(tot.cachedTokens)}</span>
                <span><i class="dot miss"></i>{t('settings.usageMiss')} {fmtCompact(tot.uncachedTokens)}</span>
              </div>
            {/if}
          </div>

          <!-- The counts were always here; what was missing was a price to
               multiply them by, so "why did my balance drain" could only be
               answered in tokens. Three states, and the difference between the
               last two matters: priced, partly priced (say how much of it the
               figure covers, or a total built from half the models reads as
               the bill), and no catalog at all (a dash, never a zero — zero
               reads as "this was free"). -->
          <div class="stat-card wide">
            <div class="eyebrow">{t('settings.usageCost')}</div>
            {#if !tot.pricesFetched || tot.pricedCalls === 0}
              <div class="stat-big dim">—</div>
              <div class="stat-sub">{t('settings.usageCostUnknown')}</div>
            {:else}
              <div class="stat-big"><span class="unit">$</span>{tot.cost.toFixed(2)}</div>
              <div class="stat-sub">
                {t('settings.usageCostEstimate')}
                {#if tot.pricedCalls < tot.calls}
                  · {t('settings.usageCostPartial', {
                    priced: fmtCompact(tot.pricedCalls), total: fmtCompact(tot.calls),
                  })}
                {/if}
              </div>
            {/if}
          </div>

          <div class="stat-card">
            <div class="eyebrow">{t('settings.usageCalls')}</div>
            <div class="stat-big">{fmtCompact(tot.calls)}</div>
            <div class="stat-sub">{t('settings.usageMessages')} {fmtTokens(tot.messages)}</div>
          </div>

          <div class="stat-card">
            <div class="eyebrow">{t('settings.usageSessions')}</div>
            <div class="stat-big">{fmtCompact(tot.sessions)}</div>
            <div class="stat-sub">{t('settings.usageActiveDays')} {tot.activeDays}</div>
          </div>

          <div class="stat-card">
            <div class="eyebrow">{t('settings.usageStreak')}</div>
            <div class="stat-big">{tot.currentStreak}<span class="unit">{t('settings.usageDaysUnit')}</span></div>
            <div class="stat-sub">{t('settings.usageActiveDays')} {tot.activeDays}</div>
          </div>

          <div class="stat-card">
            <div class="eyebrow">{t('settings.usageTopModel')}</div>
            <div class="stat-model">{tot.topModel || '—'}</div>
            <div class="stat-sub">{tot.topModelShare}% {t('settings.usageOfTokens')}</div>
          </div>
        </div>

        {#if dailyChart}
          <div class="settings-card wide-card">
            <div class="card-form">
              <div class="chart-head">
                <div class="eyebrow">{t('settings.usagePerDay')}</div>
                <!-- two keys, because the bar carries two encodings: hue names
                     the model, fill names where the tokens came from -->
                <div class="chart-legend">
                  {#each topModels as model (model)}
                    <span><i class="dot s{slotOf(model)}"></i>{model}</span>
                  {/each}
                  {#if allModels.length > 5}
                    <span><i class="dot s0"></i>{t('settings.usageOther')}</span>
                  {/if}
                </div>
              </div>
              <div class="chart-head">
                <div class="chart-legend kind-legend">
                  {#each KINDS as kind (kind)}
                    <span title={kind === 'raw' ? t('settings.usageCacheUnreported') : ''}>
                      <i class="dot k-{kind}"></i>{kindLabel[kind]}
                    </span>
                  {/each}
                </div>
              </div>

              <div class="chart-body">
                <div class="chart-y" aria-hidden="true">
                  {#each dailyChart.ticks as tick (tick.frac)}
                    <span>{fmtCompact(tick.value)}</span>
                  {/each}
                </div>
                <div class="chart-plot" role="img" aria-label={t('settings.usagePerDay')}>
                  {#each dailyChart.ticks as tick (tick.frac)}
                    <div class="chart-gridline" style="bottom:{tick.frac * 100}%"></div>
                  {/each}
                  <!-- svelte-ignore a11y_no_static_element_interactions -->
                  <div class="daychart" onpointerleave={() => (hoverDay = null)}>
                    {#each dailyChart.days as d, i (d.day)}
                      <div
                        class="daycol"
                        class:on={hoverDay === i}
                        class:idle={d.total === 0}
                        onpointerenter={() => (hoverDay = i)}
                      >
                        <!-- idle days get their baseline tick from CSS; an inline
                             height:0 here would win and erase it -->
                        <div class="daybar" style={d.total === 0 ? '' : `height:${Math.max(2, (d.total / dailyChart.max) * 100)}%`}>
                          {#each d.parts as part (part.kind + part.model)}
                            <span class="k-{part.kind} s{slotOf(part.model)}" style="flex:{part.value}"></span>
                          {/each}
                        </div>
                      </div>
                    {/each}
                  </div>
                  {#if hoveredColumn && hoverDay !== null}
                    <div
                      class="chart-tip"
                      style="left:{((hoverDay + 0.5) / dailyChart.days.length) * 100}%; bottom:{Math.min(88, (hoveredColumn.total / dailyChart.max) * 100 + 6)}%"
                    >
                      <div class="tip-day">{hoveredColumn.day}</div>
                      {#if hoveredColumn.total === 0}
                        <div class="tip-row muted">{t('settings.usageNoActivity')}</div>
                      {:else}
                        {#each KINDS as kind (kind)}
                          {#if hoveredColumn.byKind[kind] > 0}
                            <div class="tip-row">
                              <i class="dot k-{kind}"></i>{kindLabel[kind]}
                              <span class="val">{fmtTokens(hoveredColumn.byKind[kind])}</span>
                            </div>
                          {/if}
                        {/each}
                        <div class="tip-sep"></div>
                        {#each hoveredColumn.models as [model, value] (model)}
                          <div class="tip-row">
                            <i class="dot s{slotOf(model)}"></i>{model}
                            <span class="val">{fmtTokens(value)}</span>
                          </div>
                        {/each}
                      {/if}
                    </div>
                  {/if}
                </div>
                <div></div>
                <div class="chart-x" aria-hidden="true">
                  {#each chartXLabels as label, i (i)}<span>{label}</span>{/each}
                </div>
              </div>
            </div>
          </div>
        {/if}

        <div class="settings-card wide-card">
          <div class="card-form">
            <div class="eyebrow">{t('settings.usageHeatmap')}</div>
            <div class="heatmap">
              {#each heatmap.weeks as week, w (w)}
                <div class="heat-week">
                  {#each week as cell (cell.day)}
                    <span
                      class="heat-cell l{cell.future ? 'x' : heatLevel(cell.value, heatmap.max)}"
                      title={cell.future ? '' : `${cell.day} · ${fmtTokens(cell.value)}`}
                    ></span>
                  {/each}
                </div>
              {/each}
            </div>
            <div class="chart-legend heat-scale">
              <span>{t('settings.usageLess')}</span>
              <i class="heat-cell l0"></i><i class="heat-cell l1"></i><i class="heat-cell l2"></i>
              <i class="heat-cell l3"></i><i class="heat-cell l4"></i>
              <span>{t('settings.usageMore')}</span>
            </div>
          </div>
        </div>
      {/if}

      <div class="settings-card wide-card">
        <div class="card-form">
          <div class="usage-toolbar">
            <div class="eyebrow">{t('settings.usageByModel')}</div>
            <div class="seg-ctrl">
              {#each [
                { id: 'today', label: t('settings.usageToday') },
                { id: 'week', label: t('settings.usageWeek') },
                { id: 'all', label: t('settings.usageAll') },
              ] as opt (opt.id)}
                <button
                  type="button"
                  class="seg-btn"
                  class:selected={usagePeriod === opt.id}
                  onclick={() => (usagePeriod = opt.id as typeof usagePeriod)}
                >{opt.label}</button>
              {/each}
            </div>
          </div>
        </div>
        {#if usagePending}
          {#each [0, 1, 2, 3] as i (i)}
            <div class="set-row usage-sk" aria-hidden="true"><span class="sk sk-line"></span></div>
          {/each}
        {:else if usageRows.length === 0}
          <!-- Reached only once the engine has actually answered, which is what
               makes this sentence true when it is printed. -->
          <div class="set-row"><div class="muted">{t('settings.usageEmpty')}</div></div>
        {:else}
          <div class="set-row usage-head">
            <div class="u-model">{t('settings.usageModel')}</div>
            <div class="u-num">{t('settings.usageInput')}</div>
            <div class="u-num">{t('settings.usageCached')}</div>
            <div class="u-num">{t('settings.usageOutput')}</div>
            <div class="u-num sm">{t('settings.usageCalls')}</div>
            <div class="u-num sm">{t('settings.usageAvgCall')}</div>
          </div>
          {#each usageRows as r (rowKey(r))}
            <div class="set-row usage-row">
              <div class="u-model">
                <i class="dot s{slotOf(r.model)}"></i>{r.model}
                <!-- Who served it, wherever anybody wrote it down. Two rows of
                     one model are two bills, and drawing them as the same name
                     twice with different numbers is a table nobody can read. -->
                {#if r.provider}<span class="u-by">{r.provider}</span>{/if}
                <span class="u-share" style="width:{pct(usageTotal(r), periodTotal)}%"></span>
              </div>
              <div class="u-num">{fmtTokens(r.promptTokens)}</div>
              <div class="u-num">
                {#if r.cacheRows === 0}
                  <span class="dim" title={t('settings.usageCacheUnreported')}>—</span>
                {:else}
                  {pct(r.cachedTokens, r.promptTokens)}%
                  <span class="u-sub">{fmtCompact(r.cachedTokens)}</span>
                {/if}
              </div>
              <div class="u-num">{fmtTokens(r.completionTokens)}</div>
              <div class="u-num sm">{fmtTokens(r.calls)}</div>
              <div class="u-num sm">{fmtCompact(Math.round(usageTotal(r) / Math.max(r.calls, 1)))}</div>
            </div>
          {/each}
        {/if}
      </div>
    {:else if active === 'remote'}
      <RemoteEngine />
    {:else if active === 'account'}
      <h2>{t('settings.account')}</h2>
      <p class="muted set-sub">{t('settings.accountDesc')}</p>

      {#if aetoxError}<div class="mset-error">{aetoxError}</div>{/if}

      <div class="settings-card">
        {#if aetoxAccount?.signed_in}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{aetoxAccount.display}</div>
              {#if aetoxAccount.user?.email && aetoxAccount.user.email !== aetoxAccount.display}
                <div class="d">{aetoxAccount.user.email}</div>
              {/if}
            </div>
            <button class="ctrl" disabled={aetoxBusy} onclick={aetoxCheck}>{t('settings.accountRefresh')}</button>
            <button class="ctrl" disabled={aetoxBusy} onclick={aetoxSignOut}>{t('settings.accountSignOut')}</button>
          </div>
        {:else if aetoxBusy}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.accountWaiting')}</div>
              <div class="d">{t('settings.accountWaitingDesc')}</div>
            </div>
            <button class="ctrl" onclick={aetoxAbort}>{t('settings.accountCancel')}</button>
          </div>
        {:else}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.accountSignedOut')}</div>
              <div class="d">{t('settings.accountSignedOutDesc')}</div>
            </div>
            {#each aetoxAccount?.providers ?? [] as door}
              <button class="ctrl" onclick={() => aetoxSignIn(door)}>
                {t('settings.accountWith', { provider: door === 'github' ? 'GitHub' : 'Google' })}
              </button>
            {/each}
          </div>
          <!-- Said on the page rather than discovered after signing in. The
               store this account is for does not exist yet, and a button that
               implies a locked feature would be the lie. -->
          <div class="set-row">
            <div class="set-txt">
              <div class="d">{t('settings.accountUnlocks')}</div>
            </div>
          </div>
        {/if}

        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.accountServer')}</div>
            <div class="d">{t('settings.accountServerDesc')}</div>
            <div class="d">{aetoxAccount?.server ?? ''}</div>
          </div>
        </div>
      </div>

    {:else if active === 'about'}
      <h2>{t('settings.about')}</h2>
      <div class="settings-card">
        <!-- The tour's own door, first on the page it names ("ดูการแนะนำนี้อีกได้ที่
             ตั้งค่า › เกี่ยวกับ"): closes settings so it plays over the app. -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.tourTitle')}</div>
            <div class="d">{t('settings.tourDesc')}</div>
          </div>
          <div style="display:flex;gap:6px">
            <button class="ctrl" data-guide="settings.about.guide_btn" title={t('account.guideTip')}
                    onclick={() => { onClose(); guide.start() }}>{t('account.guide')}</button>
            <button data-guide="settings.about.tour_btn" class="ctrl" onclick={() => { openTour(); onClose() }}>{t('settings.tourAction')}</button>
          </div>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.aboutVersion')}</div>
            <div data-guide="settings.about.version_info" class="d">
              {appVersion ? 'v' + appVersion : '—'}
              {#if updateStatus}
                · {t(CHANNEL_LABELS[updateStatus.channel] ?? 'settings.aboutChannelUnknown')}
              {/if}
              {#if updateStatus?.checkedAt}
                · {t('settings.aboutLastChecked', { when: new Date(updateStatus.checkedAt).toLocaleString() })}
              {/if}
            </div>
          </div>
          <button data-guide="settings.about.update_btn" class="ctrl" disabled={updateChecking} onclick={checkNow}>
            {updateChecking ? t('settings.aboutChecking') : t('settings.aboutCheck')}
          </button>
        </div>

        <!-- Four outcomes, four different sentences. "Switched off" is not a
             failure and must not read like one, and a failed check must never
             leave the impression that something in the app broke. -->
        {#if updateError}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.aboutCheckFailed')}</div>
              <div class="d">{t('settings.aboutCheckFailedHint')}</div>
            </div>
          </div>
        {:else if updateStatus?.disabled}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.aboutCheckOff')}</div>
              <div class="d">{t('settings.aboutCheckOffHint', { env: 'AETOX_DISABLE_UPDATE_CHECK' })}</div>
            </div>
          </div>
        {:else if updateStatus?.available}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.aboutNewVersion', { version: updateStatus.latest })}</div>
              <!-- Three endings, one action each. canAuto is the VS Code shape
                   split in two (§107): download and verify now, restart when
                   the user says. Scoop installed us, so Scoop upgrades us:
                   Aetox never writes into someone else's package directory.
                   Everything left gets the release page, which is always a
                   correct answer. -->
              <div class="d">
                {updateStatus.canAuto
                  ? t('settings.aboutAutoHint')
                  : updateStatus.hint ? t('settings.aboutRunCommand') : t('settings.aboutDownloadHint')}
              </div>
              {#if !updateStatus.canAuto && updateStatus.hint}
                <code class="about-cmd">{updateStatus.hint}</code>
              {/if}
              {#if updater.error}
                <div class="mset-error">{t('settings.aboutUpdateFailed', { err: updater.error })}</div>
              {/if}
            </div>
            {#if updateStatus.canAuto}
              <!-- The same two acts the card offers, because they are the same
                   two acts. This page is a second view of one state, never a
                   second copy of it (selfUpdate.svelte). -->
              {#if updater.phase === 'ready'}
                <button class="ctrl ctrl-primary" onclick={restartToUpdate}>
                  {t('settings.aboutRestartToUpdate')}
                </button>
              {:else}
                <button class="ctrl" disabled={updater.phase === 'downloading' || updater.phase === 'restarting'} onclick={startDownload}>
                  {updater.phase === 'restarting'
                    ? t('settings.aboutRestarting')
                    : updater.phase === 'downloading'
                      ? (updatePct() >= 0 ? t('settings.aboutDownloadingPct', { pct: String(updatePct()) }) : t('settings.aboutDownloading'))
                      : t('settings.aboutUpdateNow')}
                </button>
              {/if}
            {:else if updateStatus.hint}
              <button class="ctrl" onclick={() => copyUpgradeHint(updateStatus!.hint)}>
                {hintCopied ? t('settings.aboutCopied') : t('settings.aboutCopy')}
              </button>
            {:else}
              <button class="ctrl" onclick={() => BrowserOpenURL(updateStatus!.url)}>
                {t('settings.aboutOpenRelease')}
              </button>
            {/if}
          </div>
        {:else if updateStatus}
          <div class="set-row">
            <div class="set-txt">
              <div class="t">{t('settings.aboutUpToDate')}</div>
            </div>
          </div>
        {/if}

        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.aboutReleaseNotes')}</div>
            <div class="d">{RELEASES_URL}</div>
          </div>
          <button data-guide="settings.about.github_btn" class="ctrl" onclick={() => BrowserOpenURL(RELEASES_URL)}>{t('settings.aboutOpenRelease')}</button>
        </div>

        <!-- Three ways to follow along, as one row rather than three.
             They are the same kind of thing — a place to go and look — and a
             full-width row each would have taken half the page to say so, with
             a fourth already coming. The icons are what distinguishes them, so
             the buttons carry both the mark and the word: an icon-only row is a
             guessing game, and these three are not guessable from a glyph. -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.follow')}</div>
            <div class="d">{t('settings.followDesc')}</div>
          </div>
          <div class="follow-row">
            <button class="ctrl" onclick={() => BrowserOpenURL(COMMUNITY_URL)}>
              <Icon name="facebook" size={14} /> {t('settings.followGroup')}
            </button>
            <button class="ctrl" onclick={() => BrowserOpenURL(PAGE_URL)}>
              <Icon name="facebook" size={14} /> {t('settings.followPage')}
            </button>
            <button class="ctrl" onclick={() => BrowserOpenURL(YOUTUBE_URL)}>
              <Icon name="youtube" size={14} /> {t('settings.followYoutube')}
            </button>
          </div>
        </div>

        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.aboutReport')}</div>
            <div class="d">{t('settings.aboutReportDesc')}</div>
          </div>
          <button class="ctrl" onclick={() => openIssueForm('problem')}>{t('settings.aboutReportOpen')}</button>
        </div>

        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.aboutFeedback')}</div>
            <div class="d">{t('settings.aboutFeedbackDesc')}</div>
          </div>
          <button class="ctrl" onclick={() => openIssueForm('feedback')}>{t('settings.aboutFeedbackOpen')}</button>
        </div>
      </div>
    {:else if active === 'sponsor'}
      <h2>{t('settings.sponsor')}</h2>
      <div class="settings-card">
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.sponsorIntro')}</div>
            <div class="d">{t('settings.sponsorDesc')}</div>
          </div>
          <button class="ctrl" onclick={() => BrowserOpenURL(SITE_URL)}>{t('settings.sponsorOpenSite')}</button>
          <button class="ctrl" onclick={() => BrowserOpenURL(SPONSOR_URL)}>{t('settings.sponsorOpenGitHub')}</button>
        </div>
        <div class="set-row">
          <div class="set-txt">
            <div class="t">PromptPay</div>
            <div class="d">{t('settings.sponsorScanHint')}</div>
          </div>
        </div>
        <div class="set-row sponsor-center">
          <img src={promptPayQR} alt="PromptPay QR" class="sponsor-qr" />
        </div>
        <!-- What a supporter gets, said before the contact row that acts on it.
             The credit is a promise the project can actually keep: a name on a
             page it controls, kept rather than counted once. -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.sponsorCredit')}</div>
            <div class="d">{t('settings.sponsorCreditDesc')}</div>
          </div>
        </div>
        <!-- The same GitHub door the problem and feedback rows use. A donation
             is anonymous by construction — PromptPay tells the developer a
             transfer happened, never who to thank — so the supporter has to be
             the one who says. Asking to stay anonymous is offered in the same
             breath, because a name on a page is not everyone's idea of thanks. -->
        <div class="set-row">
          <div class="set-txt">
            <div class="t">{t('settings.sponsorContact')}</div>
            <div class="d">{t('settings.sponsorContactDesc')}</div>
          </div>
          <button class="ctrl" onclick={() => BrowserOpenURL(ISSUES_URL + '/new')}>{t('settings.sponsorContactOpen')}</button>
        </div>
        <!-- Attribution, not decoration: this is the one place in the running app
             that names who wrote it and where it came from. Untranslated on
             purpose — a name, a licence id and a URL read the same in every
             language, and a translated copyright line is a mistranslated one.
             It comes from version.Credit through AppCredit rather than being
             written out here: a literal was a second place naming the licence,
             and on 2026-08-19 it named the old one (§148). -->
        <div data-guide="settings.about.license_btn" class="set-row">
          <div class="set-txt">
            <div class="t">Aetox</div>
            <div class="d">{appCredit}</div>
          </div>
        </div>
      </div>
    {/if}
    </div>
  </div>
</div>

{#if pendingConfirm}
  {@const req = pendingConfirm}
  <ConfirmDialog
    title={req.title}
    message={req.message}
    detail={req.detail ?? ''}
    confirmLabel={req.confirmLabel}
    onConfirm={runPendingConfirm}
    onCancel={() => (pendingConfirm = null)}
  />
{/if}
