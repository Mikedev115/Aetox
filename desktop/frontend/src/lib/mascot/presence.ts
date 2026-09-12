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
import { toolFamily, type ToolFamily } from '../toolFace'

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
  /** A tool in this turn failed and the model is deciding what to do about
   *  it — the moment between the error and the next call. */
  failed?: boolean
}

/** Tool name → what doing it looks like, for the tools whose look is more
 *  specific than their family's. Names are the ids the engine sends on a
 *  ToolStep. Anything not here falls to FAMILY_POSE by way of toolFace's
 *  family table — the same table that picks the row's icon — so a tool the
 *  app grows tomorrow, or one bridged from an MCP server, is drawn as what
 *  its family looks like from the outside rather than as nothing. */
export const TOOL_POSE: Record<string, PoseId> = {
  // Reading one thing, as opposed to looking for it.
  read: 'reading',
  pdf_read: 'reading',
  media_read: 'reading',
  image_ocr: 'reading',
  video_ocr: 'reading',
  audio_transcribe: 'reading',
  // Looking through what was said and remembered.
  session_search: 'searchData',
  memory: 'searchData',
  // Looking through documentation.
  skills_list: 'searchDocs',
  skill_view: 'searchDocs',
  github: 'searchDocs',
  github_search: 'searchDocs',
  github_read_file: 'searchDocs',
  github_list_files: 'searchDocs',
  github_repo_summary: 'searchDocs',
  help: 'searchDocs',
  // Finding what is wrong.
  diagnostics: 'debugging',
  // Deciding what to do.
  plan: 'planning',
  plan_mode: 'planning',
  todo_write: 'planning',
  suggest_task: 'planning',
  // Making the thing that will be shown.
  doc_write: 'presenting',
  sheet_write: 'presenting',
  image_make: 'presenting',
  video_project: 'presenting',
  cutting_room: 'presenting',
  ask_user: 'asking',
}

/** What a family of tools looks like when the name says nothing more. */
export const FAMILY_POSE: Record<ToolFamily, PoseId> = {
  read: 'searchFiles',
  write: 'typing',
  web: 'research',
  shell: 'coding',
  media: 'reading',
  task: 'helping',
  mcp: 'typing',
  other: 'typing',
}

/** The pose for one running tool, by name. */
export function toolPose(name: string): PoseId {
  const key = name.trim().toLowerCase()
  return TOOL_POSE[key] ?? FAMILY_POSE[toolFamily({ name: key, label: key })]
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
    return toolPose(i.running[i.running.length - 1])
  }
  if (i.streaming) return 'answering'
  // Something went wrong a moment ago and nothing has replaced it yet: the
  // model is looking at the error, not merely thinking.
  if (i.failed) return 'debugging'
  if (i.reasoning) return 'thinking'
  // Nothing concrete yet — the phrase before the first token.
  return 'thinking'
}

/** Which way to face while walking, from the way it is moving on screen.
 *
 *  The head turn --t is an azimuth: 0 faces the viewer, +90 looks to the
 *  viewer's right, 180 turns its back. Walking down the screen is walking
 *  towards the viewer, so the heading is atan2(dx, dy) with y down — right is
 *  +90, up is 180, left is -90 — and a step that is too small to have a
 *  direction keeps the last one. The angle is unwrapped against the previous
 *  heading so a turn from 170 to -170 is a 20° turn, not a spin the long way
 *  round through the transition (owner, 12 ก.ย.: "ทำไมเดินไปแค่ข้างหน้า ขึ้น ลง
 *  ซ้าย ขวา"). */
export function walkTurn(dx: number, dy: number, prev: number, minPx = 3): number {
  if (Math.hypot(dx, dy) < minPx) return prev
  return nearAngle((Math.atan2(dx, dy) * 180) / Math.PI, prev)
}

/** The angle equal to `h` (mod 360) that is nearest `prev` — so stepping
 *  from `prev` towards it is the short way round. On the seam (180° apart)
 *  it goes the positive way. */
export function nearAngle(h: number, prev: number): number {
  const r = (((h - prev) % 360) + 360) % 360
  return prev + (r > 180 ? r - 360 : r)
}

/** What a card knows about an agent, as the cartoon faces spelled it: nothing,
 *  thinking, working, finished, failed. The delegate cards, the background
 *  panel and the settings preview all speak this five-word language (Chat's
 *  faceState, BackgroundWork's task.state), and each of them should keep
 *  speaking it — a card's business is whether the job is alive, not which of
 *  twenty poses that looks like. This is the one seam where the word becomes
 *  a pose, so the mascot can replace the cartoon person under every card with
 *  the import line as the only edit. */
export type FaceState = '' | 'think' | 'work' | 'done' | 'err'

export const FACE_STATE_POSE: Record<FaceState, PoseId> = {
  '': 'idle',
  think: 'thinking',
  work: 'typing',
  done: 'success',
  err: 'error',
}

export function poseOfFaceState(state: FaceState | undefined): PoseId {
  return FACE_STATE_POSE[state ?? ''] ?? 'idle'
}

/** What the companion's bubble shows — only what the assistant SAYS, never
 *  what it runs (owner, 12 ก.ย.: "ไม่ต้องแสดงว่ามันรันคำสั่งอะไร อันนั้นเป็นไอคอนก็พอ
 *  แสดงแค่ตอนมันรายงาน"), and of what it says only the HEADLINE — the first
 *  line of the piece, not its tail and not the whole of it ("แสดงแค่หัวข้อ ๆ
 *  แสดงแค่บน ๆ ของท่อนนั้น ๆ"). While a tool runs the bubble is shut and the
 *  card beside the head is the whole story. */
export type ReportInput = {
  awaiting: boolean
  /** A tool is running right now. */
  busy?: boolean
  /** The model's latest narration between tool calls (a `note`/`said` row). */
  note?: string
  /** The answer arriving. */
  streamingText?: string
  /** The question the model is blocked on. */
  question?: string
}

/** Characters of a headline the bubble shows before it is cut. */
export const REPORT_MAX = 96

export function reportOf(i: ReportInput): string {
  if (i.question) return clip(firstLine(i.question), REPORT_MAX)
  if (i.streamingText) return clip(firstLine(i.streamingText), REPORT_MAX)
  if (!i.awaiting || i.busy) return ''
  return i.note ? clip(firstLine(i.note), REPORT_MAX) : ''
}

/** The first line with words in it, markdown marks stripped. */
function firstLine(text: string): string {
  const lines = text.replace(/[*_`#>]+/g, '').split('\n').map((l) => l.trim()).filter(Boolean)
  return lines[0] ?? ''
}
function clip(text: string, max: number): string {
  const t = text.trim()
  return t.length > max ? t.slice(0, max - 1) + '…' : t
}
