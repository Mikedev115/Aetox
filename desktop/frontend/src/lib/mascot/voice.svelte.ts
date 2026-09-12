// The composer's mic and the reader's voice, as two booleans.
//
// The mascot must not reach into Chat or the player for them (presence.ts is
// the seam, and it takes booleans). Chat mirrors its mic state here in one
// effect; the window's one player (lib/speech.svelte.ts) sets `speaking` as
// a read starts and ends; the companion reads both. A store rather than
// props because the companion is mounted at the app's root, not under Chat —
// it sits over every desk, Chat is one of them.
export const voice = $state({
  /** The composer's mic is recording. */
  mic: false,
  /** Something is being read aloud — a reply's ฟัง button or the companion. */
  speaking: false,
})
