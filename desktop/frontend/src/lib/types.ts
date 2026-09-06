// Cockpit view-model. These shapes are the whole contract between the UI and
// whatever feeds it — a mock today, the Go core via Wails bindings later.
// Components render CockpitState; they never know the source.

export type GitStatus = 'M' | 'U' | null

/** Why the local store could not be opened (Go: main.StoreFault).
 *
 *  `tooNew` is the case with a cure the user can act on — the file was migrated
 *  by a newer Aetox and this build is behind it — so it carries both schema
 *  numbers. `message` is the engine's own words, for a bug report rather than
 *  for the sidebar. */
export interface StoreFault {
  failed: boolean
  tooNew: boolean
  have: number
  known: number
  message: string
}

/** One state of the project this chat can be put back to (Go: main.RestorePoint).
 *  The label is the message that opened the turn it precedes — a list of times
 *  and tree hashes is a list nobody can pick from. */
export interface RestorePoint {
  id: string
  at: string
  label: string
}

export interface TreeNode {
  label: string
  path: string
  kind: 'dir' | 'file'
  depth: number
  open?: boolean
  status?: GitStatus
  icon?: string
}

export interface OpenFile {
  path: string
  content: string
}

export interface Session {
  id: string
  title: string
  ago: string
  /** RFC3339 stamp behind `ago`. Kept alongside the rendered label because the
   *  sidebar groups rows under วันนี้/เมื่อวาน/…, and a day boundary cannot be
   *  recovered from "3 วันที่แล้ว" — the two answer different questions. */
  updatedAt?: string
  active?: boolean
  snippet?: string
  /** Only set on the cross-project (global) history list. */
  projectName?: string
  /** The โปรเจกต์ this chat was held inside (§90). Only search results ever
   *  carry it: the plain lists drop those rows, because a project's chats
   *  belong to that project's own list. A row with this set was found by
   *  searching and says so. */
  space?: string
  /** The desk this conversation was held at (COMPANY.md §2). '' for the ones
   *  that predate desks — they belong to no desk and are shown unlabelled. */
  mode?: string
  /** Who the conversation was held with (§85): '' for the main assistant,
   *  an agent's name for a direct chat in the office. */
  agent?: string
  /** This chat exists but has never been answered, so the store has no row for
   *  it and no title to show. The lists draw it anyway — pressing "new chat"
   *  has to put something on screen, or the chat you just made is invisible
   *  until the first reply lands (owner, 26 ส.ค.). Replaced by the real row the
   *  moment the first turn is stored. */
  draft?: boolean
}

export interface RecentProject {
  key: string
  name: string
  path: string
  ago: string
  active?: boolean
  snippet?: string
}

/** One โปรเจกต์ at the storefront door (COMPANY.md §84) as the sidebar needs
 *  it: enough to draw a row and open it, and nothing more.
 *
 *  Deliberately not `main.Space`. The binding's type carries `path`,
 *  `contextPath` and the context file names, which are the โปรเจกต์ page's
 *  business — a sidebar row that held them would be a second place those
 *  answers live, and the one that goes stale first. The name IS the key here:
 *  a space exists because its folder exists (desktop/spaces.go), so there is no
 *  id to carry. */
export interface SpaceRow {
  name: string
  /** How many chats are filed in it. Not drawn on the row — see the sidebar —
   *  but it is what tells an empty project from a used one. */
  chats: number
  updatedAt: string
}

export interface ProjectInfo {
  name: string
  path: string
  branch: string
  /** false = "ไม่โฟกัสโปรเจกต์" mode — engine rooted at home, tools still fully usable. */
  focused: boolean
  extraBranches: number
  governanceFile: string
  governanceLoaded: boolean
}

export type ApprovalMode = 'ask' | 'unsafe-only' | 'full-access'

export interface ModelStatus {
  provider: string
  modelName: string
  thinkLevel: string
  contextUsed: number
  contextMax: number
  approval: ApprovalMode
  /** Active wire format for providers that speak more than one (e.g. DeepSeek's
   *  "anthropic" vs "openai-compatible"). Empty for single-format providers. */
  wireFormat: string
  /** Why `provider` is not actually answering — the engine fell back to the
   *  built-in aetox provider (e.g. LM Studio's server is not running). Empty
   *  when the named provider really is the one running. */
  warning: string
}

/** One labeled share of the context window; key: system | tools | messages | free. */
export interface ContextSlice {
  key: string
  tokens: number
}

/** One tool definition's weight inside the tool block (desktop/app.go
 *  ContextTool). The block is the biggest thing in a fresh chat and the meter
 *  used to say only its size, which is a number you can resent and cannot act
 *  on — so the row opens onto this list. */
export interface ContextTool {
  name: string
  /** Apportioned in Go so the rows sum to the tools slice exactly. A list you
   *  opened to check a number has to survive being added up. */
  tokens: number
  /** builtin | workbench | mcp — where the tool came from, which is the half
   *  that says whether anything can be done about it. */
  source: string
  /** The MCP server that bridged it, for mcp tools only. The one cost on this
   *  list the user can actually switch off. */
  server?: string
}

/** Composer context meter payload (GetContextBreakdown). */
export interface ContextBreakdown {
  usedTokens: number
  maxTokens: number
  slices: ContextSlice[]
  /** True once this session has actually sent a round, so usedTokens is the
   *  provider's own count. False means nothing has been sent and the figure is
   *  a forecast of the first request — showing that as "used" made a fresh chat
   *  look like a bill already run up. */
  measured: boolean
  /** How much of the last round's input the provider served from its prompt
   *  cache, at a fraction of full price. 0 when nothing hit or the provider
   *  does no cache accounting — the note is only shown when there is one. */
  cachedTokens: number
  /** Reclaimed by the compaction layers this session: old tool outputs swept
   *  (and the tokens they gave back) and history summaries. The sweep happens
   *  invisibly mid-turn, and a meter whose number drops with no line saying
   *  why reads as broken. Optional: absent from rounds before this shipped. */
  sweptItems?: number
  sweptTokens?: number
  summaries?: number
  /** The tools slice, broken out per definition and heaviest first. Absent on
   *  a chat with no tools at all, and on rounds from before this shipped. */
  tools?: ContextTool[]
}

export interface ChatMessage {
  role: 'user' | 'agent'
  text: string
  time: string
  /** The stored row this reply became. It is what a rating addresses, so a
   * bubble without one cannot be rated — which is the honest state for a turn
   * that failed and was persisted nowhere. */
  id?: number
  /** The verdict already given for this reply, so a reopened session shows the
   * thumb that was pressed rather than a blank pair. */
  rating?: 'good' | 'bad' | 'unknown'
  /** optional badge, e.g. "Thinking (low)" */
  tag?: string
  /** Images attached to this question, in the order they were attached — for
   * inline preview only (the model is handed their paths, in the suffix below).
   * A list because one question can carry a whole folder of screenshots; it was
   * a single field, and attaching a second picture quietly replaced the first. */
  images?: MessageImage[]
  /** Dragged-in files/browser tabs, as small cards on the bubble. Their content
   * is inlined into the text, so the card only names them. */
  contexts?: MessageContext[]
  /** User-attached videos/audio/documents, as the same chip. Only the path goes
   * to the model, so without these the bubble showed nothing at all. */
  files?: MessageFile[]
  /** The attachment marker lines that were appended to this message when it was
   * sent — the part the model reads and the bubble never shows. Kept so editing
   * the prose can re-send them: without it, fixing a typo would silently detach
   * the file the question is about. */
  attachSuffix?: string
  /** tool calls made during this turn, kept on the reply for a persistent timeline. */
  steps?: ToolStep[]
  /** Finished files this turn made for the user (see CockpitState.turnFiles).
   *  Rebuilt from the message's own parts on reload, so the open button survives
   *  a restart — it did not at first, and the file became unreachable from the
   *  answer that announced it. */
  producedFiles?: string[]
  /** Changes this turn asked to remember (CockpitState.turnProposals), rebuilt
   * from the message's own parts on reload. The bubble draws one card each, so
   * the decision is made in front of the work that suggested it instead of on a
   * Settings page the user has no reason to be looking at. */
  proposals?: number[]
  /** The turn as it happened — prose, thinking segments and tool calls in the
   * order they occurred. When present this is what the bubble draws, so
   * narration appears where it was said instead of collapsed into a panel.
   * Absent on user messages and on every turn from before the sequence existed,
   * which fall back to rendering `text`. */
  parts?: TurnPart[]
  /** the model's thinking for this reply — kept after the turn, collapsed by default. */
  reasoning?: string
  /** seconds the model spent thinking (first→last reasoning chunk). */
  thinkSecs?: number
  /** Every answer this question has had, when it was asked more than once.
   * text/reasoning/thinkSecs/steps above always mirror variants[activeVariant].
   * Fewer than two means no switcher is drawn. */
  variants?: MessageVariant[]
  activeVariant?: number
  /** Sent into a turn that was already running (Interject). The transcript
   *  draws it below the live block instead of above it, because that is where
   *  it happened — the array order is already right, so the flag is cleared
   *  the moment the turn ends and the bubble takes its ordinary place. */
  duringTurn?: boolean
  /** This bubble is a turn that never completed. failedText is exactly what was
   * sent — attachment marker lines and all — so a retry re-sends the same thing
   * rather than a reconstruction of it. */
  failed?: boolean
  failedText?: string
  /** How the failure is worded ("หยุดการทำงานแล้ว", the network sentence). It
   * is also glued onto `text`, which is what every reader that predates the
   * phase layout draws; a turn drawn from its own sequence needs it apart,
   * because the prose in `text` is already in the sequence and re-drawing the
   * whole string would say the model's last sentence twice. Composed, never
   * stored — the store keeps the error and the window keeps the wording, so
   * switching language re-words a failure that happened yesterday. */
  ending?: string
  /** The turn ended because the user pressed Stop — or closed the app while it
   * ran (§219), which is the same act on a bigger button. A subset of `failed`
   * — the bubble and the retry chip are the same ones — but not the same
   * event: the app did exactly what it was told, and painting that in the
   * danger colour told the owner his own Stop was a crash (25 ส.ค.). */
  stopped?: boolean
  /** Files put back before this answer was regenerated, named on the bubble:
   * an answer that quietly undid six files would be the worse surprise. */
  revertedFiles?: string[]
  /** A re-run that failed, shown under the answer it could not replace. The
   * previous answer is still the one on screen. */
  error?: string
}

/** One piece of an assistant turn — mirrors turn.TurnPart in Go.
 *
 * The shape a turn actually has: a provider streams prose, a tool call, more
 * prose. Collapsing that to one string is what put narration in a separate
 * panel and made the tool timeline impossible to store. */
export interface TurnPart {
  kind: 'text' | 'thinking' | 'tool'
  /** prose, on a 'text' part */
  text?: string
  /** On a 'text' part: the model had finished this as its answer, and an
   * interjection kept the turn going past it. Prose the user was already
   * reading, so a reopened session draws it as prose. */
  demoted?: boolean
  /** seconds a 'thinking' segment streamed */
  secs?: number
  tool?: ToolPartInfo
}

export interface ToolPartInfo {
  ref?: string
  name: string
  /** The action inside a packed tool, written down with the turn. Without it a
   * reopened session can only draw a `browser` row as "เปิดเว็บ" and a `change`
   * row as "แก้" — the pack's first action standing in for all of them — while
   * the live turn had said which one it actually was. Absent on turns stored
   * before this field existed. */
  act?: string
  subject?: string
  agent?: string
  brief?: string
  /** Which pile the worker on a `task` row is in: 'agent' (เอเจน) or 'helper'
   *  (ซับเอเจน). Stamped by the engine — the kind is decided by which home
   *  the profile file lives in, which only the engine can see. */
  agentKind?: string
  /** Whether this `task` row hired anybody. Written down with the turn, so a
   *  reopened session draws the same blocks the live one did. */
  delegation?: boolean
  ok: boolean
  error?: string
  secs?: number
  added?: number
  removed?: number
  /** The reading tools' readout — see ToolEvent.count/range. */
  count?: number
  range?: string
  /** Git-style unified hunks for what this call changed — the same format
   * `git diff` prints, built by the tool itself (internal/skill/hunk.go) rather
   * than asked of git, so a row expanded tomorrow still shows what THIS call
   * did instead of the working tree's current state. Empty on every tool that
   * writes no file. One line is not git's: `~N` means N further diff lines
   * exist and were dropped to keep the transcript bounded. */
  diff?: string
  /** Finished files this call made for the user. Unlike the live ToolEvent's
   *  copy, this one is written down with the message — which is what lets the
   *  open button still be there after a restart. */
  artifacts?: string[]
  /** On a `memory` call: the change it queued for approval. Written down like
   *  the artifacts above, so reopening the session brings the card back — still
   *  asking, or saying which way the decision went. */
  proposalId?: number
  /** On a `web_search` call: what it found, written down for the same reason
   *  the artifacts are. The card is the only record the user has of which
   *  sources an answer was built from, and a card that vanishes on restart
   *  leaves them with an answer and no way back to what it read. */
  links?: ToolLink[]
  /** On an `ask_user` call: what the user answered. The question card is drawn
   *  inside the live turn and is gone the moment it is answered, so this row is
   *  the only record the exchange leaves — and until this field it carried
   *  neither half of it. */
  answer?: string
}

/** One of the answers a question received. */
export interface MessageVariant {
  text: string
  reasoning?: string
  thinkSecs?: number
  /** The sequence THIS attempt produced (desktop/sessions.go SessionVariant).
   * Each try does its own work, so it is stored per variant — a variant that
   * carried only its text put one answer on screen above another answer's tool
   * calls the moment the user flipped between them. */
  parts?: TurnPart[]
  /** Live only: the timeline as the events built it. A variant read back from
   * the store has `parts` instead, and `stepsFromParts` is what turns one into
   * the other — the two must never be assumed to arrive together. */
  steps?: ToolStep[]
}

/** One tool call in the live per-turn timeline ("Using browser_read… 12s"). */
/** One background delegation as the engine's register reports it — mirrors
/** desktop/usage.go UsageRound, added up over one turn.
 *
 *  `in` counts cached input too, the same as the provider's own bill does.
 *  `cached` is the part of it that was served from the prompt cache, and
 *  cacheReported whether any provider in this turn accounts for one: false
 *  means nobody said, which is not the same as nothing hit, and the meter must
 *  stay silent about it rather than draw a zero it made up. */
export interface TurnSpend {
  in: number
  out: number
  cached: number
  cacheReported: boolean
  /** USD, summed over the rounds the engine could price. */
  cost: number
  /** How many spending rounds had no published rate. Money is drawn only while
   *  this is 0: half a bill presented as the bill is worse than no bill, and
   *  "this model is not in the price catalog" is not "this model is free". */
  unpriced: number
}

/** What a turn starts from. One function rather than the literal repeated at
 *  every reset, because a field added to TurnSpend must not silently keep its
 *  old value on three of the four paths that clear it. */
export function emptyTurnSpend(): TurnSpend {
  return { in: 0, out: 0, cached: 0, cacheReported: false, cost: 0, unpriced: 0 }
}

/** What this whole chat has spent (desktop/usage.go SessionSpend), read back
 *  from token_usage rather than accumulated here.
 *
 *  TurnSpend above is the live one and it is deliberately per-turn: it is reset
 *  by the next turn, and a webview reload has no memory of the last one at all.
 *  That is correct for the question it answers and useless for the other one a
 *  person asks in the same breath — what has this conversation cost — which is
 *  why refreshing the window looked like the bill being thrown away. Nothing
 *  was ever lost; every round was in the database, filed under the session id,
 *  and nothing read it back.
 *
 *  Same field names as TurnSpend on purpose: the panel draws both blocks with
 *  one row component, and two shapes saying the same thing is how the two
 *  blocks would drift apart. */
export interface SessionSpend extends TurnSpend {
  /** How many model rounds this chat has taken. Shown beside the total because
   *  it is the number that explains it: 20 rounds is a turn that ran a long
   *  tool loop, not a chat with twenty messages in it. */
  rounds: number
}

/** A chat with nothing spent yet. Not the same as "not loaded": a chat that has
 *  genuinely spent nothing draws no card at all, so both cases render the same
 *  and neither invents a zero the user could read as a measurement. */
export function emptySessionSpend(): SessionSpend {
  return { ...emptyTurnSpend(), rounds: 0 }
}

/** desktop/background_tasks.go BackgroundTask. The tray draws these. */
export interface BackgroundTask {
  id: string
  agent: string
  label: string
  /** RFC3339; the row's clock runs from it client-side. */
  startedAt: string
  toolCalls: number
  /** What this delegate ran on, and what it spent. Per-delegate answers the
   *  session total cannot give: a delegate's tokens land in the user's total
   *  untouched, so the total knows how much and nothing knew by whom. */
  model?: string
  tokens: number
  /** The same spend split into what the model read and what it wrote, live.
   *  Two numbers because they are two different problems and the brake is a
   *  different decision for each: input climbing is a transcript being re-sent
   *  every round, output climbing is a model that will not stop writing. */
  tokensIn: number
  tokensOut: number
  /** The share of tokensIn served from the provider's prompt cache, and whether
   *  the provider accounts for one at all. cacheReported false means unknown,
   *  NOT zero, and the card must draw nothing rather than a 0 it invented. */
  cachedIn: number
  cacheReported: boolean
  /** The declared job this belongs to, both absent for a delegate started on
   *  its own (internal/subagent/run.go). */
  run?: string
  phase?: string
  /** 'running' | 'queued' (asked for, waiting for one of the four slots)
   *  | 'waiting' (parked on a question) | 'done' | 'failed'
   *  | 'stopped' (the user ended it, which is neither of the last two) */
  state: string
  /** How long the delegation really took, present only once it has finished.
   *  While it runs the clock is still going, so the row counts from startedAt. */
  elapsedMs?: number
  /** What a waiting delegate is stuck on, absent otherwise. */
  question?: string
  /** A finished result somebody has already redeemed — the work is in the
   *  conversation now, so the tray drops the row. */
  collected: boolean
}

/** One declared job — mirrors desktop/background_tasks.go BackgroundRun. The
 *  card draws the phases in the order they were DECLARED, never in the order
 *  work arrived: a phase nobody has worked in yet is the row worth drawing. */
export interface BackgroundRun {
  id: string
  name: string
  brief?: string
  startedAt: string
  running: boolean
  tokens: number
  phases: BackgroundPhase[]
}

/** One stage of a run. `planned` is what the plan said and 0 when it did not
 *  say, which the card draws as a bare count — a denominator nobody promised is
 *  one the user would hold the work to. */
export interface BackgroundPhase {
  title: string
  planned: number
  done: number
  failed: number
  running: number
  waiting: number
  tokens: number
}

/** One result a `web_search` came back with — mirrors turn.ToolLink in Go.
 *
 * The list has always existed; it was formatted into the tool's text output and
 * handed to the model, which is a place a UI cannot read from. So the chat could
 * say that a search had run and how many seconds it took, and not one word about
 * what it found — while the model, three lines later, was writing about sources
 * the user had never been shown. This is that list travelling as data.
 */
export interface ToolLink {
  title: string
  url: string
  /** The agent went back and read this one in full — a later `web_fetch` or
   * `browser open` on the same URL. Not sent by the engine: it is a fact about
   * the REST of the turn, so the window works it out over the whole step list
   * (markOpenedLinks in lib/toolFace.ts) rather than a single event knowing
   * something it cannot. */
  opened?: boolean
}

/** One tool call/result as the engine sends it — mirrors turn.ToolEvent in Go. */
export interface ToolEvent {
  action: 'call' | 'result' | 'note' | 'thinking' | 'said'
  name: string
  /** The engine's tool-call id — how a row is recognized across updates. The
   * label cannot serve: it is empty of its subject on the early events. */
  ref?: string
  /** Set only on events from inside a sub-agent: the ref of the `task` call that
   * spawned it. The row is shown as that task's work rather than the agent's own. */
  parent?: string
  /** The delegation's own id ("task_1"), on every event from inside a sub-agent.
   * `parent` is the provider's call id — a different namespace — so this is the
   * only key that joins a live row to its task in the engine's register. */
  task?: string
  subject?: string
  /** The action inside a packed tool: `browser`'s open/read/click, `task`'s
   * start/collect. Packing put a dozen capabilities behind one name, so `name`
   * says "browser" for every one of them and this is the only field that says
   * which. Absent for a tool that is not packed. */
  act?: string
  /** Which browser tab this call is working. Stamped by the host on the way
   * past, not by the engine, which has never heard of a browser. It is what
   * lets ไฟบอกสถานะ point at one tab instead of lighting the whole panel
   * (busySignal.svelte.ts). Absent for every tool that is not the browser. */
  tab?: string
  ok?: boolean
  error?: string
  added?: number
  removed?: number
  /** The reading tools' readout, result events only: how much came back in the
   * tool's own unit (lines for read, matches for grep, files for glob), and
   * for read the 1-based line span it returned ("77-136") — so a row can say
   * WHICH slice was opened, not just which file. */
  count?: number
  range?: string
  /** The touched file's one-letter git state (M/A/U/D/R), host-stamped on
   * file-tool results. Absent for a clean file, outside a repo, and on tools
   * that touch no file — the badge marks the noteworthy only. */
  git?: string
  /** Errors the language server sees in a file this call changed — the
   * after-edit self-check's number, worn as a red "!N". */
  problems?: number
  /** What a `web_search` found: title and URL per result, in the order the
   * tool ranked them. Absent on every other tool. */
  links?: ToolLink[]
  /** Git-style unified hunks for what this call changed — the same format
   * `git diff` prints, built by the tool itself (internal/skill/hunk.go) rather
   * than asked of git, so a row expanded tomorrow still shows what THIS call
   * did instead of the working tree's current state. Empty on every tool that
   * writes no file. One line is not git's: `~N` means N further diff lines
   * exist and were dropped to keep the transcript bounded. */
  diff?: string
  /** Set only on the `task` call that opens a delegation: which sub-agent it is
   * handing the work to, and the whole brief it handed over. They are what make
   * a delegation render as a named block with its own steps inside, instead of
   * one more row reading "task". */
  agent?: string
  brief?: string
  /** With them, which pile that worker is in: 'agent' (เอเจน) or 'helper'
   * (ซับเอเจน). The chat counts the two apart; empty lands in the helper
   * pile, which is where every delegation lived before the split. */
  agentKind?: string
  /** Whether this row hired anybody. Only the engine can say: delegation is
   * packed under one tool name, so `task collect` — the agent waiting on a
   * delegate it started earlier — has the same label as the delegation it is
   * waiting for. Absent means nobody has said yet, which is what a row born
   * from the streaming announcement knows: the action is in the arguments,
   * and they are still being written. */
  delegation?: boolean
  /** On a result event: sandbox paths of finished files this call made for the
   * user — set only by tools whose whole output is a file (sheet_write and, in
   * time, the .pptx and .docx writers). `write` and `edit` deliberately leave
   * it empty, or a coding turn would print a card per edited source file. */
  artifacts?: string[]
  /** On a result event from `memory`: the change it queued for approval. The
   * chat draws it as a card under the answer, so what the agent wants to
   * remember is decided where it was proposed rather than in Settings. */
  proposalId?: number
  /** On a result event from `ask_user`: what the user answered. Travels beside
   * `subject`, which carries the question — the two halves of an exchange the
   * live card stops showing the instant it is answered. */
  answer?: string
  /** On a "note" event: the narration the model wrote alongside this round's
   * tool calls — its own words for what it is doing. On a "said" event: a whole
   * answer the model had finished writing when the user typed over it. */
  text?: string
  /** On a "thinking" event: how long that round's reasoning streamed. */
  secs?: number
}

export interface ToolStep {
  /** What this row is. Absent means a tool call, as every row was before
   * "note" (the model's narration between calls) and "thinking" (a reasoning
   * segment's duration) joined the timeline.
   *
   * "said" is not a row at all: it is an answer the model finished writing and
   * an interjection re-placed. It rides in this list because that is where the
   * order lives, and is drawn as markdown prose in the bubble — never inside
   * the tool timeline, and never counted as a tool. */
  kind?: 'note' | 'thinking' | 'said'
  /** The row as one string: `name subject`, which is what it was for a year and
   * what every row stored before this change is.
   *
   * Kept, not replaced. It is still the identity a row falls back to when the
   * engine sends no call id (`running()` in the store matches on it), it is
   * still what a `note`/`said` row's prose rides in, and a turn read out of the
   * database can only be as detailed as what was written down. The three fields
   * below are the same facts un-joined; the drawing prefers them and falls back
   * to this, so an old turn keeps its row and a new one gets a better one. */
  label: string
  /** The tool the engine named — `browser`, `change`, `read`. Split back out of
   * the label because joining it in threw away the thing the row most needed:
   * which FAMILY of work this is, which is what the icon and the colour say
   * (lib/toolFace.ts). Absent on rows born before this field existed. */
  name?: string
  /** The action inside a packed tool (ToolEvent.act). The whole reason §99 put
   * Act on the event: `browser` is twelve different sentences and `name` says
   * the same word for all of them. Absent for an unpacked tool, and on a row
   * rebuilt from a turn stored before ToolPart carried it. */
  act?: string
  /** The one argument worth reading — the path, the URL, the query. Held apart
   * from `name` so the row can draw it in its own ink: mono, dim for the
   * directory, bright for the file name at the end of it. */
  subject?: string
  /** ToolEvent.ref of the call this row is showing, when the engine sent one. */
  ref?: string
  /** Set when this row is a sub-agent's work, carrying the `task` call's ref. */
  parent?: string
  /** The delegation's id, for joining this row to the register's task. */
  task?: string
  state: 'run' | 'done' | 'err'
  /** Why it failed, straight from the engine's result event. Only on 'err'. */
  error?: string
  /** Lines a write or edit changed, for the "+9 -0" readout. */
  added?: number
  removed?: number
  /** The reading tools' side of that readout — see ToolEvent.count/range. */
  count?: number
  range?: string
  /** The touched file's git letter — see ToolEvent.git. */
  git?: string
  /** The self-check's error count — see ToolEvent.problems. */
  problems?: number
  /** What a `web_search` found — see ToolLink. Absent on every other tool, and
   * on a search from before the engine sent them. */
  links?: ToolLink[]
  /** Git-style unified hunks for what this call changed — the same format
   * `git diff` prints, built by the tool itself (internal/skill/hunk.go) rather
   * than asked of git, so a row expanded tomorrow still shows what THIS call
   * did instead of the working tree's current state. Empty on every tool that
   * writes no file. One line is not git's: `~N` means N further diff lines
   * exist and were dropped to keep the transcript bounded. */
  diff?: string
  /** On an `ask_user` row: what the user answered. The row's `subject` carries
   * the question, so the two together are the whole exchange — which is the
   * only place it survives, the card that asked it being drawn inside the live
   * turn and gone the moment it is answered. */
  answer?: string
  /** On a `task` row: the worker doing the job, and the brief it was given. */
  agent?: string
  brief?: string
  /** On a `task` row: 'agent' (เอเจน) or 'helper' (ซับเอเจน) — which of
   * the two count chips this delegation belongs to. Absent (old turns, engine
   * without the stamp) counts as a helper, the pre-split reading. */
  agentKind?: string
  /** ToolEvent.delegation, kept on the row: whether this `task` call hired
   * anybody. Absent while the arguments are still streaming, and on turns
   * stored before the engine said so. */
  delegation?: boolean
  startedAt: number
  /** seconds it took, filled in when the result arrives */
  secs?: number
}

/** A row in the timeline plus whatever ran underneath it. Only a `task` row ever
 * has children — a sub-agent's tool calls, which belong inside its block rather
 * than mixed into the main agent's list. */
export interface TimelineNode {
  step: ToolStep
  children: ToolStep[]
}

/** Fold the flat step list the engine produces into what the timeline draws.
 *
 * Delegation is the only nesting there is, so this is one pass: a row with no
 * parent is the agent's own, and a row carrying a parent belongs to the `task`
 * row with that ref. A child whose parent is not in this list (a persisted turn
 * trimmed oddly, an event that arrived first) is kept at the top level rather
 * than dropped — a visible row in the wrong place beats work that vanished.
 */
export function groupSteps(steps: ToolStep[]): TimelineNode[] {
  const byRef = new Map<string, TimelineNode>()
  const nodes: TimelineNode[] = []
  for (const step of steps) {
    if (!step.parent) {
      const node: TimelineNode = { step, children: [] }
      if (step.ref) byRef.set(step.ref, node)
      nodes.push(node)
      continue
    }
    const parent = byRef.get(step.parent)
    if (parent) parent.children.push(step)
    else nodes.push({ step, children: [] })
  }
  return nodes
}

/** True when a row hired a sub-agent.
 *
 * The engine's answer wins wherever it gave one, because only the engine reads
 * the arguments and the arguments are where the answer is: delegation is packed
 * under one tool name, and of its actions only `start` hires anyone. A `task
 * collect` — the agent waiting on a delegate it started earlier — carries the
 * same label as the delegation it is waiting for, so a row judged by its label
 * was drawn as a second worker beside the one it was waiting on, and counted as
 * one.
 *
 * The guess below is what a row falls back to while nobody has said yet: one
 * still streaming its arguments, or a turn stored before the engine said. It is
 * right about every delegation and wrong about the actions that are not one,
 * which is the correct way round for a fallback that only ever holds for a
 * moment. */
export function isDelegation(node: TimelineNode): boolean {
  // Narration is free text — it may well start with the word "task".
  if (node.step.kind) return false
  if (node.step.delegation !== undefined) return node.step.delegation
  return Boolean(node.step.agent) || node.children.length > 0 || node.step.label.startsWith('task ') || node.step.label === 'task'
}

/** An image attached in the composer, staged before send. */
export interface PendingImage {
  /** Sandbox-relative path — what the model/tools (e.g. image_ocr) operate on. */
  relPath: string
  /** data: URL for the composer's own thumbnail preview. */
  dataUrl: string
}

/** A non-image file attached from the composer's "+" — copied into the sandbox
 * and handed to the model as a path rather than inlined, because a clip is for
 * a tool to open, not for the message to carry. */
export interface PendingFile {
  /** Sandbox-relative path — what audio_transcribe / video_ocr / read operate on. */
  relPath: string
  /** Chip label — the original file name. */
  label: string
  /** Decides which tool the model is pointed at. */
  kind: 'audio' | 'video' | 'file'
}

/** A workbench tab (file or browser) dragged into the composer, staged before
 * send — its content is inlined into the message so the model reads it
 * directly, no tool call needed. */
export interface PendingContext {
  /** 'pick' is what the user pointed at in the browser, not the whole page. */
  kind: 'file' | 'browser' | 'pick'
  /** Chip label — file name, page title/URL, or the selector pointed at. */
  label: string
  content: string
}

/** One attached image as a sent message carries it. `dataUrl` is what the
 * bubble draws and never survives a reload; `relPath` is the sandbox path a
 * restored message reads the thumbnail back from, since a data URL has no
 * business in the history DB. */
export interface MessageImage {
  dataUrl?: string
  relPath?: string
}

/** One dragged-in file/browser tab as a sent message carries it. */
export interface MessageContext {
  label: string
  /** First lines of the inlined content, for the card in the bubble. */
  preview?: string
}

/** One attached clip/document as a sent message carries it. */
export interface MessageFile {
  label: string
  kind: PendingFile['kind']
}

export type StepStatus = 'done' | 'active' | 'wait'

export interface ChangeSummary {
  items: string[]
  footer: string
  badge: string
}

export interface TimelineStep {
  time: string
  title: string
  detail: string
  status: StepStatus
  change?: ChangeSummary
}

export interface TaskState {
  elapsed: string
  steps: TimelineStep[]
}

// DiffView / TestRun / ChangedFile lived here for the Review panel, which was
// removed on 2026-08-03. Two of its four sections — the diff and the test run —
// were never written to by anything and had shown an empty box since the day
// they shipped; the two that worked duplicated what the file tree already says
// with its M/U badges. It was a code-review surface in a product whose promise
// is finished work for people who do not read diffs.

/** One folder the user added to the focused project. It carries the same rights
 *  the project folder does — the list is the permission, so what is drawn here
 *  is exactly what the agent can reach. */
export interface ProjectFolder {
  path: string
  name: string
  /** No longer on disk. Kept on the list (an unplugged drive is not a decision
   *  to remove it) but drawn as unavailable, or it reads as reachable. */
  missing: boolean
}

/** A local runtime reading a model off disk before this turn can start.
 *
 *  `secs` is measured, and there is deliberately no percentage: neither Ollama
 *  (/api/ps) nor LM Studio (/api/v0/models) counts the bytes on their way into
 *  memory — they answer "resident" or "not resident" — so a bar here would be
 *  this app's guess wearing a measurement's clothes. null when nothing is
 *  loading (desktop/model_load.go). */
export interface ModelLoading {
  /** The wait is on. Carried as its own flag because a session with no model
   *  name pinned waits just as long as one with, and reading "is it loading"
   *  off the name would leave that wait undrawn. */
  loading: boolean
  provider: string
  /** '' when the session has not pinned one — the row then names no model. */
  model: string
  secs: number
}

/** One chat's live turn state, held while the window is showing another chat.
 *
 * The same fields CockpitState carries for the chat on screen — that is the
 * point: parking is a move, not a translation, so a field that exists in one
 * and not the other is a field that gets lost on the way. */
export interface ParkedTurn {
  chat: ChatMessage[]
  awaitingReply: boolean
  agentStatus: string
  toolSteps: ToolStep[]
  turnFiles: string[]
  turnProposals: number[]
  streamingText: string
  reasoningText: string
  modelLoading: ModelLoading | null
  ask: { question: string; options: string[] } | null
  todos: { content: string; status: 'pending' | 'in_progress' | 'completed' }[]
  /** Parked with the rest of the live state, because the meter belongs to the
   *  turn and the turn is what is being parked. Left on `cockpit` instead, it
   *  followed the user into the next chat and drew one conversation's spend
   *  under another's composer. */
  turnSpend: TurnSpend
  /** Stragglers — messages the engine handed back because its turn was already
   *  returning (agent:interjection-missed). Parked with the chat they were
   *  typed into: the queue was one module global, so a message missed in chat A
   *  while the user stood in chat B went out as B's next turn. */
  queued: string[]
}

export interface CockpitState {
  project: ProjectInfo
  /** Folders added to the focused project, in the order they were added. */
  projectFolders: ProjectFolder[]
  projects: RecentProject[]
  tree: TreeNode[]
  sessions: Session[]
  /** All chat history across every project, newest first — sidebar's global history layer.
   *  Chats held inside a โปรเจกต์ are not in it: they live in `spaceHistory`. */
  history: Session[]
  /** Why the history above could not be read, or null when it could.
   *
   *  It exists because an empty `history` used to mean two completely different
   *  things and said neither: "you have no chats" and "your chats are on disk
   *  and this build cannot open the file they are in". On 7 ก.ย. 2026 the owner
   *  got the second one — 77 sessions, none on screen — and read it as the
   *  first, which is the only reading the window offered. */
  historyFault: StoreFault | null
  /** The open chat's project's own chats (§90), newest first. Empty whenever
   *  the open chat is in no project — which is most of the time, and is what
   *  puts `history` back on screen. */
  spaceHistory: Session[]
  /** Every โปรเจกต์ the user has, newest first — the sidebar's own list (§84).
   *
   *  Held here rather than fetched by the sidebar the way the โปรเจกต์ page
   *  fetches it, because two lists of the same folders that refresh at
   *  different moments disagree in front of the user: the page can afford to
   *  read the disk when it opens, the rail is on screen while the disk is
   *  being changed. */
  spaces: SpaceRow[]
  model: ModelStatus
  chat: ChatMessage[]
  task: TaskState
  openFiles: OpenFile[]
  /** 'chat', 'settings', 'office', 'artifacts', or an open file's path —
   *  which surface the window currently shows. */
  activeView: string
  /** The desk the open session was created at, '' for the full desk. Read back
   *  from the engine rather than remembered here: a session is born at a desk
   *  and never moves, so this changes only when a different session is opened
   *  (COMPANY.md §6.3). */
  desk: string
  /** The agent the open session talks to directly (§85), '' for the main
   *  assistant. Same lifecycle as desk: fixed at birth, read back, never
   *  remembered independently. */
  chair: string
  /** The โปรเจกต์ the open session is being held inside (COMPANY.md §84), ''
   *  for a chat held outside every project. Same lifecycle as desk and chair:
   *  fixed when the session is born, read back from the engine when one is
   *  reopened. It is what the chat shows so a project chat does not look like
   *  any other chat — a session filed somewhere with nothing on screen saying
   *  so reads as the project having vanished. */
  space: string
  /** How the open session is being run (DECISIONS.md §106), '' for ลงมือ.
   *
   *  The one coordinate here with a different lifecycle, and that is the point
   *  of it: desk, chair and space are fixed when a session is born, while this
   *  is a dial the user turns mid-conversation. It only ever subtracts from the
   *  desk, so turning it cannot bring another desk's tools into this context —
   *  which is why it needs no new session and §6.3 still holds. */
  stance: string
  /** Every stance the engine implements, in picker order, ลงมือ first. Read
   *  from Go rather than listed here: which stances exist is the engine's
   *  answer, and a hardcoded copy is the one that goes stale. What each is
   *  called is a locale string — see `stance.*` in the locales. */
  stances: string[]
  /** True from the moment a message is sent until the reply (or an error) arrives. */
  awaitingReply: boolean
  /** Which conversation the turn in flight belongs to. '' when nothing is running.
   *
   *  The window used to answer this by looking at `active` — the engine's open
   *  session — which is the same id right up until the moment it matters. The
   *  question it is really asked is "is the chat I am about to open the one
   *  that is working?", and answering it from the engine's cursor meant the
   *  working chat could be handed back as a stored transcript with the live
   *  work missing. Stamped from the engine's own session id when the turn
   *  starts (the same value Go stamps the turn with), so the two cannot drift. */
  turnSession: string
  /** Files the last turn changed, from PendingUndo. Empty when there is nothing
   *  to undo — no snapshot yet, not a git repo, or the turn touched no file. */
  undoFiles: string[]
  /** Every point this chat can be taken back to, newest first, from
   *  RestorePoints. Empty until a turn has run. Refreshed beside undoFiles:
   *  they answer the same question at two depths, and a list that lagged one
   *  turn behind the chip would offer a point that is already the present. */
  restorePoints: RestorePoint[]
  /** Live turn-progress text from the Go engine's status reporter ("กำลังคิดคำตอบ...", etc), '' when idle. */
  agentStatus: string
  /** Tool calls of the turn in flight, appended live from agent:tool events. */
  toolSteps: ToolStep[]
  /** Tool calls of a sub-agent still working after the turn that started it
   *  ended. A delegate's life is the session's, not the turn's, so its steps
   *  keep arriving with nowhere in the live block to belong — toolSteps is
   *  cleared at both ends of every turn, so they would surface inside the NEXT
   *  turn's timeline as somebody else's work. Kept here instead, for the whole
   *  session, which is also the span the work itself now has. */
  backgroundSteps: ToolStep[]
  /** The tray's rows: this session's delegations as the ENGINE's register
   *  reports them (App.BackgroundTasks). Never derived from tool events — a
   *  `task` call completes the instant the work starts, so events cannot tell
   *  running from done. Refreshed by refreshBackgroundTasks. */
  backgroundTasks: BackgroundTask[]
  backgroundRuns: BackgroundRun[]
  /** What has been spent since this turn began, counted live off usage:round.
   *  Reset when a turn starts, not when one ends: a sub-agent outlives the turn
   *  that dispatched it, and the rounds it keeps spending afterwards are still
   *  this chat's bill. Zeroing at the end would hide exactly the case worth
   *  watching, which is spend continuing while nothing looks like it is going
   *  on. */
  turnSpend: TurnSpend
  /** What the chat on screen has spent in total, read back from the database
   *  (SessionSpend) rather than accumulated here.
   *
   *  Deliberately not derived from turnSpend. token_usage already holds every
   *  round of every turn this chat ever took, so the total is a query, and a
   *  running sum kept beside it would be a second answer that drifts the first
   *  time a round is missed — or double-counts the first time both are added.
   *  Re-read rather than incremented, on the open of a chat and after each
   *  round: recordTokenUsage writes the row before emitUsageRound announces it,
   *  so a refresh a round triggers already contains that round. */
  sessionSpend: SessionSpend
  /** Sandbox paths of finished files this turn produced — a spreadsheet, a deck,
   *  a document. Collected live from agent:tool results so the reply can show
   *  them as cards with an open button: the file panel is where you go looking,
   *  and a deliverable should not need looking for. Reset with toolSteps. */
  turnFiles: string[]
  /** Ids of the changes this turn asked to remember, collected live from the
   *  same tool results. The reply draws a card for each: memory is the one tool
   *  whose work does not happen until a person says yes, and a queue nobody is
   *  looking at is the same as no proposal. Reset with toolSteps. */
  turnProposals: number[]
  /** Reply text streamed so far this turn, appended live from agent:chunk events. '' when idle. */
  streamingText: string
  /** Model's reasoning/thinking tokens streamed so far this turn, from agent:reasoning events. '' when idle or the provider doesn't stream reasoning. */
  reasoningText: string
  /** The local runtime is loading this turn's model into memory — a wait that
   *  looks exactly like a hung app until something says so. null otherwise. */
  modelLoading: ModelLoading | null
  /** Images staged in the composer, not yet sent — in the order they were
   *  attached. A list, not a slot: attaching a second picture used to replace
   *  the first, so one question could only ever carry one. */
  pendingImages: PendingImage[]
  /** Why the last attempt to open a session from the history list failed, in
   *  the engine's own words. '' when the last one worked. */
  sessionError: string
  /** The live state of a chat that is working while the window looks at another.
   *
   *  Replaces `peek`, which held exactly one field of this (the messages) and
   *  only for reading. The window can hold several working chats now: the one
   *  on screen keeps its live state in the fields above, and every other one
   *  that has a turn in flight keeps its own here, keyed by session. Switching
   *  parks the outgoing chat's state and restores the incoming one's, so a
   *  timeline drawn in one conversation can never appear under another and a
   *  reply can never be appended to whichever chat happened to be open when it
   *  arrived.
   *
   *  Empty whenever nothing is working off screen, which is almost always. */
  parked: Record<string, ParkedTurn>
  /** Which chat the window is showing. The window's own fact, not the engine's
   *  — the engine is addressed by id and has no "current" of its own
   *  (desktop/conversation.go). Everything that means "the chat on screen"
   *  reads this. */
  openSession: string
  /** File/browser tabs dragged into the composer, staged before send. */
  pendingContexts: PendingContext[]
  /** Non-image files attached in the composer, staged before send. */
  pendingFiles: PendingFile[]
  /** Question the model is blocked on (ask_user tool), null when none. */
  ask: { question: string; options: string[] } | null
  /** The model's task checklist (todo_write tool), replaced wholesale each call. */
  todos: { content: string; status: 'pending' | 'in_progress' | 'completed' }[]
  /** Side work the agent flagged with suggest_task — chips the user can start or dismiss. */
  taskChips: TaskChip[]
  /** How many things the agent wants to remember and is waiting to be allowed to.
   * Surfaced as a mark on the way into settings: an approval queue nobody is
   * told about is one that never gets emptied, which would quietly turn
   * "nothing takes effect without you" into "nothing takes effect". */
  pendingLearned: number
  /** How many things have failed the same way more than once and are waiting
   * for the user to decide whether the developer should hear about them. Its
   * own number because it is its own question: this one lights the problems
   * row in settings and nothing else, while pendingLearned marks the gear. A
   * repeated failure is worth finding when you go looking, and is not worth
   * being interrupted for. */
  pendingIssues: number
  /** A one-shot request from another page for what Settings should open on:
   * the team page's configure/create doors land in the shared profile editor
   * this way. Carries the *kind* because it came from the roster — Settings
   * must never re-derive it from a file. Consumed and cleared on arrival. */
  settingsIntent: { section: string; agent?: string; createAgent?: boolean } | null
}

/** One suggested side task from the agent (suggest_task tool). */
export interface TaskChip {
  id: string
  title: string
  tldr: string
  prompt: string
  createdAt: string
}

/** A blank, well-formed state so the UI renders before the source hydrates. */
export function emptyCockpitState(): CockpitState {
  return {
    project: { name: '', path: '', branch: '', focused: false, extraBranches: 0, governanceFile: '', governanceLoaded: false },
    projectFolders: [],
    projects: [],
    tree: [],
    sessions: [],
    history: [],
    historyFault: null,
    spaceHistory: [],
    spaces: [],
    model: { provider: '', modelName: '', thinkLevel: '', contextUsed: 0, contextMax: 0, approval: 'ask', wireFormat: '', warning: '' },
    chat: [],
    task: { elapsed: '', steps: [] },
    turnFiles: [],
    turnProposals: [],
    openFiles: [],
    activeView: 'chat',
    desk: '',
    chair: '',
    space: '',
    stance: '',
    stances: [],
    awaitingReply: false,
    turnSession: '',
    undoFiles: [],
    restorePoints: [],
    agentStatus: '',
    toolSteps: [],
    backgroundSteps: [],
    backgroundTasks: [],
    backgroundRuns: [],
    turnSpend: emptyTurnSpend(),
    sessionSpend: emptySessionSpend(),
    streamingText: '',
    reasoningText: '',
    modelLoading: null,
    ask: null,
    todos: [],
    taskChips: [],
    pendingLearned: 0,
    pendingIssues: 0,
    settingsIntent: null,
    pendingImages: [],
    sessionError: '',
    parked: {},
    openSession: '',
    pendingContexts: [],
    pendingFiles: [],
  }
}
