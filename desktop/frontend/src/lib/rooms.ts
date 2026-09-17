// The id of every page a room can be asked to open, owned in one place.
//
// These used to be private to the room that drew them — `type Page` inside
// Capability.svelte, a bare `string` for Settings' rail — and every caller
// that wanted to send somebody to a page named it as a free string. That is
// fine while the only callers are three buttons in the same file, and it broke
// the day the guide's map named 157 of them: the map asked for the capability
// room's `mcp` and `builtins` pages, neither of which has ever existed (they
// are `mine` and `tools`), and nothing said so. Capability.arriveAt drops an
// id it does not recognise on purpose — a room that threw on a stale intent
// would be worse — so the guide walked to the room, landed on whatever page
// was last open, and reported that the button was not on screen.
//
// A union, not a list of strings, because the failure has to be a compile
// error: `npm run check` names the caller and the bad id, before anyone runs
// the app. The room's own rail is typed against it too, so a page added to the
// rail without a line here is the same error from the other side.
//
// Adding a page: add the id here, then to the room's rail. Renaming one:
// change it here and the compiler lists every caller that has to follow —
// which is the whole point of the file.

// The list is the value and the type at once: a caller that has a string in
// hand at runtime — the guide's model naming a page, an intent read back out
// of storage — needs to ask whether it is real, and a second hand-kept copy
// for that check is the same drift one file up.

/** Pages of ห้องความสามารถ (Capability.svelte's rail), in rail order. */
export const CAPABILITY_PAGES = [
  'mine', 'desks', 'agents', 'shelf',
  'skills', 'skagents', 'skshelf', 'sktune',
  'tools',
  'prompts', 'habits',
  'computer',
  'connections',
] as const
export type CapabilityPage = (typeof CAPABILITY_PAGES)[number]

export function isCapabilityPage(s: string): s is CapabilityPage {
  return (CAPABILITY_PAGES as readonly string[]).includes(s)
}

/** Sections of the Settings rail (Settings.svelte's `sections`), in rail order. */
export const SETTINGS_SECTIONS = [
  'general', 'appearance', 'avatar', 'you', 'issues',
  'models', 'main', 'team', 'teams', 'agents', 'hands',
  'voice', 'image', 'studio', 'remote',
  'account', 'usage', 'about', 'sponsor',
] as const
export type SettingsSection = (typeof SETTINGS_SECTIONS)[number]

export function isSettingsSection(s: string): s is SettingsSection {
  return (SETTINGS_SECTIONS as readonly string[]).includes(s)
}


// ---------------------------------------------------------------- places

/**
 * Rooms that are not made of sections — one screen, one id.
 */
export const ROOM_IDS = ['chat', 'office', 'artifacts', 'projects', 'videowork', 'lines'] as const

/**
 * **The name of a place in this app**, and the app's one vocabulary for it.
 *
 * Four things speak it and none of them keeps its own list: the sign a page
 * stamps on itself (`data-guide-place`), the `page` on every row of the
 * guide's map, the door chains that lead from one place to another
 * (guide/path.ts), and the adapter that opens one outright (guide/pages.ts).
 *
 * That is the whole point of having it. The guide used to ask the app where it
 * was — `cockpit.activeView` — which answers "settings" and cannot say *which*
 * settings page, and which ties a feature that should float above the UI to
 * one store's internals. A place that names itself answers both: the guide
 * reads the sign, and knows nothing about how the room is built (owner,
 * 15 ก.ย. 2026: *"อาจจะต้องทำเหมือนป้ายไว้ ถ้าไม่มีป้ายมันก็ไม่รู้ว่าตอนนี้อยู่หน้าไหน
 * … ไกด์ไม่ผูกกับหน้าไหนเลย"*, and *"ระบบแมพของเราเอามาใช้กับตรงนี้ได้ไหม"* — yes,
 * and this is where the two meet).
 */
export type PageId =
  | (typeof ROOM_IDS)[number]
  | `settings.${SettingsSection}`
  | `capability.${CapabilityPage}`

export const PAGE_IDS: readonly PageId[] = [
  ...ROOM_IDS,
  ...SETTINGS_SECTIONS.map((s) => `settings.${s}` as const),
  ...CAPABILITY_PAGES.map((p) => `capability.${p}` as const),
]

export function isPageId(s: string): s is PageId {
  return (PAGE_IDS as readonly string[]).includes(s)
}

/** The room half of a place: `settings.models` → `settings`, `chat` → `chat`. */
export function roomOf(page: PageId): string {
  const dot = page.indexOf('.')
  return dot < 0 ? page : page.slice(0, dot)
}

/** The attribute a page stamps on itself so the guide can read where it is.
 *  One spelling, written down once — a second copy of this string is a bug
 *  nothing would report. */
export const PLACE_ATTR = 'data-guide-place'

/** A page can keep the same physical room while its working context changes.
 * Chat does this when the top-left door swaps ผู้ช่วย ↔ โค้ด. The guide watches
 * this second sign so a desk switch is a real state change, not a silent text
 * replacement inside a page still named `chat`. */
export const CONTEXT_ATTR = 'data-guide-context'
