// What the assistant is doing right now, as one word the mascot can wear.
//
// The window already knows everything this needs — it just knows it in six
// places: the engine's phase phrase (agentStatus, Thai prose), the running
// tool rows, the answer streaming in, the reasoning streaming in, a question
// the model is blocked on, the composer's mic, the reader's audio. None of
// those is a pose, and the mascot must not read them itself: this function
// is the single seam, so that when a seventh signal appears (a plan step, a
// delegate's report) it is one more line here and nothing in the rig.
//
// Pure: inputs in, a pose id out, no store, no time. The caller decides how
// long a pose lingers (a success card that flashes for 200ms is not seen).
import type { PoseId } from './poses'

export type PresenceInput = {
  /** A turn is running. */
  awaiting: boolean
  /** Engine phase phrase — only consulted when nothing concrete is on screen. */
  status?: string
  /** Names of the tools currently running (ToolStep.name), outermost first. */
  running?: readonly string[]
  /** The answer's prose is arriving. */
  streaming?: boolean
  /** The model's reasoning is arriving. */
  reasoning?: boolean
  /** The model is blocked on an ask_user question. */
  asking?: boolean
  /** The composer's mic is recording. */
  mic?: boolean
  /** A reply is being read aloud. */
  speaking?: boolean
  /** The last turn ended and nobody has typed since. */
  justDone?: boolean
}

/** Tool name → what doing it looks like. Names are the ids `tool_runs` logs
 *  (the packed ones, not their per-action keys): a tool not listed here is
 *  drawn as typing, which is what most of them look like from the outside. */
export const TOOL_POSE: Record<string, PoseId> = {
  web_search: 'research',
  web_fetch: 'research',
  browser: 'research',
  read: 'reading',
  pdf_read: 'reading',
  media_read: 'reading',
  search: 'searchFiles',
  codebase: 'searchFiles',
  session_search: 'searchData',
  memory: 'searchData',
  skills_list: 'searchDocs',
  skill_view: 'searchDocs',
  github: 'searchDocs',
  change: 'typing',
  rename: 'typing',
  doc_write: 'typing',
  sheet_write: 'typing',
  shell: 'coding',
  git: 'coding',
  pr: 'coding',
  desk_terminal: 'coding',
  plan: 'planning',
  todo_write: 'planning',
  task: 'helping',
  ask_user: 'asking',
}

export function presenceOf(i: PresenceInput): PoseId {
  // The user's own actions come first: a mascot that keeps typing while the
  // user is talking to it is not listening.
  if (i.mic) return 'listening'
  if (i.speaking) return 'answering'
  if (i.asking) return 'asking'
  if (!i.awaiting) return i.justDone ? 'success' : 'idle'
  if (i.running && i.running.length) {
    // The innermost tool is the one whose work is visible; a `task` that
    // delegated to a `read` is reading right now.
    for (let k = i.running.length - 1; k >= 0; k--) {
      const pose = TOOL_POSE[i.running[k]]
      if (pose) return pose
    }
    return 'typing'
  }
  if (i.streaming) return 'answering'
  if (i.reasoning) return 'thinking'
  // Nothing concrete yet — the phrase before the first token.
  return 'thinking'
}

/** What the companion's bubble shows — only what the assistant SAYS, never
 *  what it runs (owner, 12 ก.ย.: "ไม่ต้องแสดงว่ามันรันคำสั่งอะไร อันนั้นเป็นไอคอนก็พอ
 *  แสดงแค่ตอนมันรายงาน"). The running tool is already the card beside the
 *  head; this is the sentence the model wrote between tools, the tail of the
 *  answer as it streams, or the question it is blocked on. */
export type ReportInput = {
  awaiting: boolean
  /** The model's latest narration between tool calls (a `note`/`said` row). */
  note?: string
  /** The answer arriving. */
  streamingText?: string
  /** The question the model is blocked on. */
  question?: string
}

/** Characters of the answer the bubble keeps while it streams. */
export const REPORT_TAIL = 64
/** Characters of a narration the bubble shows. */
export const REPORT_MAX = 120

export function reportOf(i: ReportInput): string {
  if (i.question) return clip(i.question, REPORT_MAX)
  if (i.streamingText) {
    const line = lastLine(i.streamingText)
    return line.length > REPORT_TAIL ? '…' + line.slice(-REPORT_TAIL) : line
  }
  if (!i.awaiting) return ''
  return i.note ? clip(lastLine(i.note), REPORT_MAX) : ''
}

function lastLine(text: string): string {
  const lines = text.replace(/[*_`#>]+/g, '').split('\n').map((l) => l.trim()).filter(Boolean)
  return lines[lines.length - 1] ?? ''
}
function clip(text: string, max: number): string {
  const t = text.trim()
  return t.length > max ? t.slice(0, max - 1) + '…' : t
}
