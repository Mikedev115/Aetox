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
