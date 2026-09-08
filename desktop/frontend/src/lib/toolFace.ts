import type { IconName } from './icons'
import type { TKey } from './i18n.svelte'
import type { ToolStep } from './types'

/**
 * What a tool row SAYS it is, out of what the engine actually sent.
 *
 * The row used to be one string: `[name, subject].join(' ')`, which arrived on
 * screen as `canva_generate-design A modern tech poster announcing…`. That is a
 * log line. It reads as a log line for a reason that has nothing to do with
 * fonts — a log line is a record for whoever debugs it later, and it names the
 * function that ran. A UI row is for the person watching the work happen, and
 * what they are owed is what the agent is DOING, in words, with the thing it is
 * doing it to beside them.
 *
 * The facts to say that with have been on turn.ToolEvent the whole time — Name,
 * Act, Subject, Count, Range (ARCHITECTURE.md §99 put Act there precisely
 * because packing made Name stop being an answer: `browser` is twelve different
 * sentences, `change` is five). The window was joining them into a string and
 * dropping Act on the floor. This file is the join undone: three fields in, a
 * family / an icon / a verb out.
 *
 * It lives beside the components rather than inside Chat.svelte because both
 * halves of the transcript ask it the same question — a live row and a row
 * rebuilt from the database — and because a table this size is worth being able
 * to test without rendering a chat.
 */

/** The six kinds of work the timeline distinguishes, plus the tail.
 *
 * Six because six is what a glance can hold. The categories are drawn where a
 * reader's question changes: "is it looking or changing" (read vs write) is the
 * one that matters most and gets the two loudest colours; "is it leaving this
 * machine" (web, shell) is the next; media and delegation are their own because
 * neither looks like anything else that happens in a turn.
 *
 * `mcp` is the seventh, and it is not a seventh KIND of work — it is the honest
 * answer to a different question. A bridged tool's job is unknowable from here
 * (nobody can enumerate every server anyone will ever connect), but who it
 * belongs to is knowable exactly, because internal/mcp names every one of them
 * `server_tool`. So the row stops guessing at the work and says the provenance
 * instead, which is the fact a reader of that row actually wants: this did not
 * happen inside your project, it happened at Canva.
 *
 * `other` is what is left: `time`, `calc`, and a first-party tool added after
 * this table was last read. Neutral tile, no colour — the honest drawing of
 * "something ran". Inventing a colour for the unclassifiable would make the
 * palette say less, not more. */
export type ToolFamily = 'read' | 'write' | 'web' | 'shell' | 'media' | 'task' | 'mcp' | 'other'

/**
 * name → family, keyed on BOTH vocabularies on purpose.
 *
 * The packed names (`search`, `change`, `codebase`, `media_read`) are what a
 * live event carries; the per-action names underneath them (`grep`, `write`,
 * `symbol`, `image_ocr`) are what a turn stored before packing carries, what
 * `tool_runs` speaks, and what a CLI-side caller still uses. Both are here
 * because both reach this function, and a pack cannot straddle a family line
 * anyway — §99 draws the packs so that `search` reads only and `change` writes
 * only, which is exactly the line this table draws. That is not a coincidence:
 * it is the same rule, once for permissions and once for the eye.
 */
const FAMILY_OF: Record<string, ToolFamily> = {
  // Looking, in this machine.
  read: 'read',
  search: 'read',
  list: 'read',
  glob: 'read',
  grep: 'read',
  codebase: 'read',
  symbol: 'read',
  repo_map: 'read',
  diagnostics: 'read',
  session_search: 'read',
  skills_list: 'read',
  skill_view: 'read',
  // Changing it.
  change: 'write',
  write: 'write',
  edit: 'write',
  edits: 'write',
  append: 'write',
  delete: 'write',
  rename: 'write',
  apply: 'write',
  // Leaving it, over HTTP.
  web_search: 'web',
  web_fetch: 'web',
  browser: 'web',
  browser_open: 'web',
  browser_read: 'web',
  browser_click: 'web',
  browser_type: 'web',
  browser_capture: 'web',
  // Leaving it, as a process — and the two tools whose whole subject is the
  // repository's own history, which is the same kind of act to a reader: a
  // command against something outside the file they are looking at.
  //
  // Driving another program on this machine is filed here too, for the reason
  // its category is CategoryShell: to a reader, "reached out of this app and
  // did something to the machine" is one kind of act, whether the thing it
  // reached was a command line or somebody's window.
  computer: 'shell',
  computer_apps: 'shell',
  computer_read: 'shell',
  computer_capture: 'shell',
  computer_focus: 'shell',
  computer_click: 'shell',
  computer_type: 'shell',
  computer_close: 'shell',
  shell: 'shell',
  shell_output: 'shell',
  shell_kill: 'shell',
  shell_list: 'shell',
  desk_terminal: 'shell',
  git: 'shell',
  github: 'shell',
  pr: 'shell',
  // Files that are not text.
  media_read: 'media',
  image_ocr: 'media',
  video: 'media',
  video_ocr: 'media',
  audio_transcribe: 'media',
  pdf_read: 'media',
  doc_write: 'media',
  sheet_write: 'media',
  // Handing the work to somebody else.
  task: 'task',
  // The rest of the first-party set, listed so that the MCP split below cannot
  // reach them: every one of these carries an underscore, and `skills_list`
  // read as server `skills` doing `list` would be a bridged tool that is not
  // one. A name in this table is by definition ours.
  plugin_install: 'other',
  desk: 'other',
  desk_open: 'other',
  desk_list: 'other',
  desk_close: 'other',
  todo_write: 'other',
  ask_user: 'other',
  memory: 'other',
  n8n: 'other',
  windmill: 'other',
  time: 'other',
  calc: 'other',
}

/** A bridged tool's two halves, or null when the name is not one.
 *
 * internal/mcp/adapter.go builds every bridged name as `ToolPrefix(server) +
 * tool` — the server, sanitized to lowercase, then one underscore, then the
 * tool. So the split is exact rather than a guess, and it is the same rule the
 * naming side uses; reading it any other way here would be the second copy of a
 * rule that this codebase treats as debt.
 *
 * Guarded by the table above: a first-party name that happens to carry an
 * underscore (`web_search`, `shell_output`, `skills_list`) is claimed there
 * first and never reaches this. Which means the guard is the thing to keep
 * honest — a new first-party tool with an underscore and no table entry would
 * be drawn as somebody else's.
 */
export function mcpParts(name: string): { server: string; action: string } | null {
  const raw = (name ?? '').trim().toLowerCase()
  if (!raw || FAMILY_OF[raw] || VERB_OF[raw]) return null
  const cut = raw.indexOf('_')
  if (cut <= 0 || cut === raw.length - 1) return null
  return { server: raw.slice(0, cut), action: raw.slice(cut + 1) }
}

/** The bridged tool's action, said as words: `generate-design` -> `generate
 *  design`. The separators are there because a tool name has to match
 *  ^[A-Za-z0-9_-]+$ for the model APIs (adapter.go), which is a constraint on
 *  the wire and not on the row. */
export function humanizeAction(action: string): string {
  return (action ?? '').replace(/[-_]+/g, ' ').trim()
}

/** Which of the five chart slots a server's tile takes.
 *
 * Hashed from the name, so a server keeps its colour for life and two tools
 * from Canva always look like two tools from Canva — the same property
 * coverHue.ts gives a gallery, and the same reason: the colour is only useful
 * if it is stable. Chart slots rather than a free hue because theme.css says
 * the chart family is the one set a named theme may leave alone, so five
 * validated colours are available here without asking fourteen themes to
 * re-pick them.
 */
export function serverSlot(server: string): number {
  // Folded wide and narrowed once at the end, not reduced mod 5 on every
  // character. coverHue.ts reduces as it goes because its modulus is 360 and
  // the string keeps most of its information; at modulus 5 it does not — the
  // first five servers tried (canva, slack, figma, notion, github) put three of
  // them on the same colour, which is a palette that has stopped saying
  // anything. Same hash, one reduction.
  let h = 0
  for (const ch of server ?? '') h = (h * 31 + (ch.codePointAt(0) ?? 0)) % 1000003
  return (h % 5) + 1
}

/** One icon per family, and the reason it is per family rather than per tool.
 *
 * A tool row is read in a column of a dozen others. What the eye is being asked
 * is "which of these are the same kind of thing" — a question a shared glyph
 * answers at a glance and thirty distinct glyphs answer never. The verb beside
 * it is what says which particular tool ran.
 *
 * `search` (the magnifier) is deliberately NOT the read family's icon even
 * though half that family searches: the web-search card wears it as its own
 * head, and one glyph meaning two things in the same transcript is one glyph
 * meaning nothing. */
const ICON_OF: Record<ToolFamily, IconName> = {
  read: 'fileText',
  write: 'pencil',
  web: 'globe',
  shell: 'terminal',
  media: 'image',
  task: 'bot',
  // A plug, because that is what the row is saying: this ran somewhere else,
  // through something you connected. A wrench said "a tool ran", which is true
  // of every row in the list and therefore says nothing.
  mcp: 'plug',
  other: 'wrench',
}

/**
 * The verb, keyed on the act inside the pack where there is one.
 *
 * This is the table Act was added to ToolEvent for. `browser` alone can only be
 * drawn as "เว็บ", which is the row saying that the browser is busy and not one
 * thing more — the exact complaint §99's Act comment records. With the act in
 * hand the same row says เปิดเว็บ / อ่านหน้าเว็บ / คลิก / พิมพ์, which is what
 * somebody watching a browsing turn is actually trying to follow.
 *
 * Keyed `name:act` first, then the bare name. The bare-name entries are not a
 * fallback nobody hits: a turn read back from the database has no act at all
 * (turn.ToolPart did not carry one until this change), so every restored packed
 * row lands on them, and they have to be a true sentence on their own rather
 * than a placeholder.
 */
const VERB_OF: Record<string, TKey> = {
  // Reading and finding.
  read: 'tool.read',
  'search:list': 'tool.listFiles',
  'search:glob': 'tool.findFiles',
  'search:grep': 'tool.grep',
  search: 'tool.findFiles',
  list: 'tool.listFiles',
  glob: 'tool.findFiles',
  grep: 'tool.grep',
  'codebase:errors': 'tool.diagnose',
  'codebase:symbol': 'tool.symbol',
  'codebase:map': 'tool.repoMap',
  codebase: 'tool.symbol',
  diagnostics: 'tool.diagnose',
  symbol: 'tool.symbol',
  repo_map: 'tool.repoMap',
  session_search: 'tool.sessionSearch',
  // Writing. `append` shares edit's verb because that is what it is to a
  // reader — the file changed and did not appear — and the counts beside it
  // already say how much.
  'change:write': 'tool.write',
  'change:edit': 'tool.edit',
  'change:append': 'tool.edit',
  'change:batch': 'tool.editMany',
  'change:delete': 'tool.delete',
  change: 'tool.edit',
  write: 'tool.write',
  edit: 'tool.edit',
  edits: 'tool.editMany',
  append: 'tool.edit',
  delete: 'tool.delete',
  rename: 'tool.rename',
  // The web.
  web_search: 'tool.webSearch',
  web_fetch: 'tool.webFetch',
  'browser:open': 'tool.browserOpen',
  'browser:read': 'tool.browserRead',
  'browser:scroll': 'tool.browserRead',
  'browser:click': 'tool.browserClick',
  'browser:type': 'tool.browserType',
  'browser:capture': 'tool.browserCapture',
  'browser:tabs': 'tool.browserTabs',
  browser: 'tool.browserOpen',
  browser_open: 'tool.browserOpen',
  browser_read: 'tool.browserRead',
  browser_click: 'tool.browserClick',
  browser_type: 'tool.browserType',
  browser_capture: 'tool.browserCapture',
  // Programs on this machine. Both spellings of each, the same as the browser's
  // above: a packed call arrives as `computer:read` and the gates below it spell
  // the same act `computer_read`, and a row that knows only one of the two draws
  // the other as a third-party MCP server called "computer" (see mcpParts).
  'computer:list_apps': 'tool.computerApps',
  'computer:read': 'tool.computerRead',
  'computer:capture': 'tool.computerCapture',
  computer: 'tool.computerApps',
  computer_apps: 'tool.computerApps',
  computer_read: 'tool.computerRead',
  computer_capture: 'tool.computerCapture',
  'computer:focus': 'tool.computerFocus',
  'computer:click': 'tool.computerClick',
  'computer:type': 'tool.computerType',
  'computer:close': 'tool.computerClose',
  computer_focus: 'tool.computerFocus',
  computer_click: 'tool.computerClick',
  computer_type: 'tool.computerType',
  computer_close: 'tool.computerClose',
  // Commands and the repository.
  'shell:run': 'tool.shell',
  'shell:output': 'tool.shellOutput',
  'shell:kill': 'tool.shellKill',
  'shell:list': 'tool.shellList',
  shell: 'tool.shell',
  shell_output: 'tool.shellOutput',
  desk_terminal: 'tool.shell',
  git: 'tool.git',
  github: 'tool.github',
  pr: 'tool.pr',
  // Things that are not text.
  'media_read:image': 'tool.readImage',
  'media_read:video': 'tool.readVideo',
  'media_read:audio': 'tool.readAudio',
  media_read: 'tool.readImage',
  image_ocr: 'tool.readImage',
  video_ocr: 'tool.readVideo',
  audio_transcribe: 'tool.readAudio',
  pdf_read: 'tool.readPdf',
  video: 'tool.makeVideo',
  doc_write: 'tool.writeDoc',
  sheet_write: 'tool.writeSheet',
  // Delegation. `collect` is the agent sitting and waiting on somebody else,
  // which is a different thing to watch than hiring them, and until Act existed
  // the two rows were indistinguishable.
  'task:start': 'tool.delegate',
  'task:collect': 'tool.awaitDelegate',
  'task:plan': 'tool.plan',
  'task:answer': 'tool.answer',
  task: 'tool.delegate',
  // The tail worth naming anyway: small tools a reader sees often enough that
  // the raw identifier would be the only untranslated word on screen.
  memory: 'tool.memory',
  todo_write: 'tool.todo',
  ask_user: 'tool.askUser',
  time: 'tool.time',
  calc: 'tool.calc',
}

/** The name a row should be classified by: the tool's, lower-cased. */
function baseName(step: Pick<ToolStep, 'name' | 'label'>): string {
  // `name` is what the engine sent. `label` is the old single string, and its
  // first word is the tool name by construction — which is what keeps a turn
  // stored before this change from losing its icon.
  const raw = step.name || step.label.split(' ')[0] || ''
  return raw.trim().toLowerCase()
}

export function toolFamily(step: Pick<ToolStep, 'name' | 'label'>): ToolFamily {
  const name = baseName(step)
  return FAMILY_OF[name] ?? (mcpParts(name) ? 'mcp' : 'other')
}

/** The server a bridged tool came from, for the chip in front of the verb.
 *  Empty for everything first-party, which is what keeps that chip off the
 *  rows where it would only be repeating the app's own name back. */
export function toolServer(step: Pick<ToolStep, 'name' | 'label'>): string {
  return mcpParts(baseName(step))?.server ?? ''
}

export function toolIcon(family: ToolFamily): IconName {
  return ICON_OF[family]
}

/** The locale key for this row's verb, or '' when there is nothing better to
 *  say than the tool's own name. */
export function toolVerbKey(step: Pick<ToolStep, 'name' | 'act' | 'label'>): TKey | '' {
  const name = baseName(step)
  const act = (step.act ?? '').trim().toLowerCase()
  if (act && VERB_OF[`${name}:${act}`]) return VERB_OF[`${name}:${act}`]
  return VERB_OF[name] ?? ''
}

/** What a row shows when no verb is known.
 *
 * For a bridged tool this is the good case rather than the sad one: the server
 * half moves out to its own chip and what is left — `generate design`, `send
 * message`, `create issue` — is already a verb phrase, written by whoever built
 * the server. Nobody can translate it, and it does not need translating; it
 * needs the identifier taken off it, which is what this does.
 *
 * For anything else it is the tool's own name and, when the pack has one, the
 * act after it — so an unmapped first-party tool still says which of its
 * actions ran, which is more than the old label managed. */
export function toolFallbackVerb(step: Pick<ToolStep, 'name' | 'act' | 'label'>): string {
  const name = baseName(step)
  const parts = mcpParts(name)
  if (parts) return humanizeAction(parts.action)
  const act = (step.act ?? '').trim().toLowerCase()
  return act ? `${name} ${act}` : name
}

/** The thing this row is about, from whichever field still has it.
 *
 * `subject` is the engine's own answer and is always preferred. The fallback is
 * for a row that only ever had the joined label: its first word is the tool
 * name by construction, so everything after the first space is the subject that
 * was joined onto it. Without this a row from an older shape would draw its verb
 * and then nothing — which is a worse row than the log line it replaced, and the
 * whole point of keeping `label` was that the old rows keep working.
 */
export function toolSubject(step: Pick<ToolStep, 'subject' | 'label'>): string {
  if (step.subject) return step.subject
  const label = step.label ?? ''
  const gap = label.indexOf(' ')
  return gap < 0 ? '' : label.slice(gap + 1)
}

/**
 * The subject cut into the part that locates it and the part that names it.
 *
 * `internal/skill/web_search.go` is one string to a computer and two facts to a
 * reader: WHERE (dim, skippable, only read when the file name is ambiguous) and
 * WHAT (the thing the row is actually about). Drawing them in one ink makes the
 * eye scan sixty characters to find the last eight, on every row, all the way
 * down the turn.
 *
 * Split on the last separator only. A deeper split — colouring each segment —
 * was tried and is noise: nobody reads a path segment by segment, they read the
 * end of it. Returns an empty head for a subject with no separator, which is
 * most shell commands and every query, and those are simply all "name".
 */
export function splitSubject(subject: string): { head: string; tail: string } {
  const s = subject ?? ''
  // Only a path is cut, and whitespace is what says this is not one. A shell
  // command and a search query both contain slashes often enough — the first
  // row this was tried on read `go test ./internal/...`, and cutting at the last
  // slash dimmed "go test ./internal/" and lit up "..." , which is the emphasis
  // exactly inverted. A path is one token; a sentence is not, and a sentence has
  // no directory half to skip past.
  //
  // Paths with spaces in them exist and lose the split. That costs them one ink
  // instead of two; getting it wrong costs the row its meaning.
  if (/\s/.test(s)) return { head: '', tail: s }
  const cut = Math.max(s.lastIndexOf('/'), s.lastIndexOf('\\'))
  // A trailing separator has no name after it (a directory the model wrote as
  // `src/lib/`), so the whole thing is the name rather than a head with nothing
  // on the end of it.
  if (cut <= 0 || cut === s.length - 1) return { head: '', tail: s }
  return { head: s.slice(0, cut + 1), tail: s.slice(cut + 1) }
}

/** The host of a result URL, without the `www.` nobody reads.
 *
 * Falls back to the raw string rather than throwing: a link that arrived
 * malformed should still draw as SOMETHING under its title, and the title is
 * the part the reader is choosing by anyway. */
export function linkDomain(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return (url ?? '').replace(/^https?:\/\//, '').split('/')[0]
  }
}

/** Which of a search's results the agent went back and READ.
 *
 * The badge cannot be decided by the event that carries the links, because the
 * answer had not happened yet when that event fired: the search finds eight,
 * and two or three calls later a `web_fetch` opens two of them. It is a fact
 * about the turn, not about the call — so it is worked out over the whole step
 * list, from both ends of the transcript (a live turn's events and a turn
 * rebuilt out of the database), by the same function.
 *
 * Why it earns its place: a search that found eight and a search that found
 * eight and read two are different pieces of work, and the difference is
 * exactly what a reader checking an answer's sources wants to know. Without it
 * the card says what the internet has, not what the agent did.
 *
 * Matched on the URL with its trailing slash and #fragment removed, because
 * `web_fetch` is called with the URL the model copied out of the search result
 * and models tidy those on the way past. Mutates in place and returns the same
 * list, so a caller can drop it into a pipeline or use it for its effect.
 */
export function markOpenedLinks(steps: ToolStep[]): ToolStep[] {
  const opened = new Set<string>()
  for (const step of steps) {
    const name = baseName(step)
    const act = (step.act ?? '').trim().toLowerCase()
    // The two ways an agent reads a page in full. `browser` is admitted only on
    // the actions that actually load one: a `click` or a `type` carries a
    // selector in its subject, not a URL, and would never match anyway — but
    // listing the acts says which ones count rather than relying on that.
    const reads = name === 'web_fetch' ||
      (name === 'browser' && (act === '' || act === 'open' || act === 'read')) ||
      name === 'browser_open' || name === 'browser_read'
    if (!reads) continue
    const url = normalizeURL(step.subject ?? '')
    if (url) opened.add(url)
  }
  for (const step of steps) {
    for (const link of step.links ?? []) {
      link.opened = opened.has(normalizeURL(link.url)) || undefined
    }
  }
  return steps
}

function normalizeURL(url: string): string {
  const trimmed = (url ?? '').trim().toLowerCase()
  if (!trimmed) return ''
  return trimmed.split('#')[0].replace(/\/+$/, '')
}

/** The one or two letters a favicon-less result wears.
 *
 * Taken from the domain rather than the title on purpose: the reader is
 * scanning for "did this come from go.dev or from a content farm", and two
 * results from one site should carry the same mark even when their titles begin
 * differently. */
export function linkInitials(url: string): string {
  const host = linkDomain(url)
  const first = host.split('.')[0] || host
  return first.slice(0, 2).toUpperCase()
}
