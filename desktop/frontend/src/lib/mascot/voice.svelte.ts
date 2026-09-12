// The composer's mic and the reader's voice, as two booleans.
//
// Both live in Chat.svelte, where the buttons are, and the mascot must not
// reach into Chat for them (presence.ts is the seam, and it takes booleans).
// Chat mirrors its two states here in one effect; the companion reads them.
// A store rather than props because the companion is mounted at the app's
// root, not under Chat — it sits over every desk, Chat is one of them.
export const voice = $state({
  /** The composer's mic is recording. */
  mic: false,
  /** A reply is being read aloud. */
  speaking: false,
})
