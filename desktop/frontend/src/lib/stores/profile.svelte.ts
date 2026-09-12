// The name the user typed for themselves — one store, because two rooms read
// it. The sidebar footer shows and edits it; the empty chat greets by it
// (Chat.svelte, start.*.headlineNamed). Until 12 ก.ย. 2026 the footer kept it
// in its own $state and nothing else could see it, which is how a name the
// owner had typed was "ตั้งไว้เสียเปล่า" — set and never used.
//
// Backed by the preference file, not localStorage — see
// config.ModelPreference.UserName for why it used to vanish between builds.
import { SetUserName, UserName } from '../../../wailsjs/go/main/App'

/** `loaded` is set the moment the first read starts, so two rooms mounting
 *  together ask the backend once; a test resets it to make the next mount
 *  read again. */
export const profile = $state({ name: '', loaded: false })

/** Read the saved name once the backend is up. Every room that shows the name
 *  calls this on mount and the first call does the reading; a backend that is
 *  not up yet, or a test with no binding, leaves the name empty rather than
 *  throwing. */
export async function loadProfileName(): Promise<void> {
  if (profile.loaded) return
  profile.loaded = true
  try {
    profile.name = ((await UserName()) ?? '').trim()
  } catch {
    /* backend not up yet — typing a name still saves it */
  }
}

/** Keep the trimmed name and write it through. The store updates first so
 *  every reader moves together; the write is fire-and-forget like the rest of
 *  the footer's preferences. */
export function saveProfileName(name: string): void {
  const next = name.trim()
  if (next === profile.name) return
  profile.name = next
  void SetUserName(next).catch(() => {})
}
