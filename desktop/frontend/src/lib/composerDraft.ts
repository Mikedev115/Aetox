// The composer's unsent text, filed under the chat it was typed in.
//
// One key, one slot: the draft is a convenience for the chat on screen, not a
// record. It lives here rather than inside Chat.svelte because the โปรเจกต์
// page hands a starter card into a chat it has just opened, and the only way
// to do that without a second channel is to file the text where the composer
// already looks on mount — Chat unmounts behind the page and mounts again
// when it closes, reading this slot for the session the engine is on.

export const DRAFT_KEY = 'aetox-composer-draft'

export type StoredDraft = { session: string; text: string }

/** Put text in the composer of `session`, to be found the next time it mounts.
 *  Storage that refuses is not an error: the chat opens empty, as it would have. */
export function stashDraft(session: string, text: string): void {
  try {
    localStorage.setItem(DRAFT_KEY, JSON.stringify({ session, text } satisfies StoredDraft))
  } catch {
    // Private mode, quota, or no storage at all — the draft is a convenience.
  }
}
