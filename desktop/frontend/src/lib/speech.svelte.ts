// The one voice of the window: everything read aloud goes through here.
//
// This is the ฟัง button's player, lifted out of Chat.svelte (12 ก.ย. 2026)
// the day the companion learned to speak. Two readers in one window would
// have been two voices at once — the owner's first worry ("เสียงชนกัน") —
// and Chat is unmounted whenever a page covers it, which killed a read the
// moment the user opened Settings. Up here there is one queue, one <audio>
// at a time, and it lives as long as the window does.
//
// The rules that keep it one voice:
//
//   - Newest wins. speak() stops whatever is playing before it starts, so
//     the ฟัง button, the companion, and a second press all replace rather
//     than pile up. Nothing is ever queued behind a read.
//   - Pieces play strictly in turn. The backend (desktop/speak.go) announces
//     one piece at a time and this plays them back to back; a piece that is
//     not ready yet is a pause in the reading, never a second voice over the
//     first. That is the answer to "เสียงโหลดไม่ทัน": the synthesizer runs a
//     little ahead of the listener and the listener waits, it does not skip.
//   - Report every piece as it starts. That report (SpeechPlaying) is what
//     releases the synthesizer to run one more piece ahead, so a player that
//     goes quiet stops the work rather than letting it run to the end of a
//     reply nobody is listening to.
//
// What a read is called (`key`) is the caller's business: Chat keys by
// message, the companion by turn. Errors go back to the caller that started
// the read — the ฟัง button shows them, the companion stays silent (owner:
// "ทำงานไม่ได้ก็เงียบไป").
import { StartSpeech, StopSpeech, SpeechPlaying } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { applySpeaker } from './audioDevices.svelte'
import { voice } from './mascot/voice.svelte'

type SpeechChunk = { job: string; seq: number; url: string; mime: string; last: boolean; error?: string }

export const speech = $state({
  /** The read being played, by its caller's key; '' = silent. */
  key: '',
  /** The read waiting on its first piece — a spinner, not yet a sound. */
  busyKey: '',
})

let job = ''                          // the backend read this queue belongs to
let current: HTMLAudioElement | null = null
let queue: HTMLAudioElement[] = []
let last = false                      // the last piece has arrived; nothing more is coming
let starting = false                  // StartSpeech is in flight and has not named the job yet
let held: SpeechChunk[] = []          // pieces that beat that name back here
let onError: ((msg: string) => void) | undefined
let wired = false
// Which read is current, as a number rather than as its key: two reads may
// share a key (the companion greets, the user's name lands, it greets again)
// and the first's StartSpeech can still be in flight when the second starts.
// Told apart by key alone, the first came back, saw its own key, and took
// the job — two readings of the greeting at once, heard as an echo on the
// first open (owner, 12 ก.ย.: "เสียงมันแบบก้องเหมือนถูกสร้างมาซ้อนกัน").
let generation = 0

/** Listen for pieces once, the first time anything is read. A read that was
 *  stopped is filtered out by its job id, not by tearing the listener down. */
function wire(): void {
  if (wired) return
  wired = true
  EventsOn('speech:chunk', (c: SpeechChunk) => {
    // A read is announced before its id gets back here: the backend starts
    // synthesizing the moment StartSpeech is called, and the first piece can
    // beat that call's own reply across the bridge. Holding those rather than
    // dropping them is the difference between a fast first word and a spinner
    // that never stops.
    if (starting && !job) {
      held.push(c)
      return
    }
    if (!job || c.job !== job) return
    accept(c)
  })
}

function accept(c: SpeechChunk): void {
  if (c.error) {
    const report = onError
    stopSpeech()
    report?.(c.error)
    return
  }
  if (c.last) last = true
  // preload='auto' is the prefetch: the piece after the one being spoken is
  // fetched from /aetox-tts/ while there is still audio playing over it,
  // which is what makes the seam between two pieces inaudible.
  const audio = new Audio(c.url)
  audio.preload = 'auto'
  audio.dataset.seq = String(c.seq)
  audio.load()
  queue.push(audio)
  if (!current) void playNext()
}

async function playNext(): Promise<void> {
  const audio = queue.shift()
  if (!audio) {
    // Out of pieces: either the read is over, or the next one is still being
    // made and the arriving chunk will call back in here.
    current = null
    if (last) stopSpeech()
    return
  }
  current = audio
  speech.busyKey = ''
  audio.onended = () => { void playNext() }
  // A piece that will not play is not a reason to stop the read — skip to
  // the next one, the way a dropped frame is skipped rather than fatal.
  audio.onerror = () => { void playNext() }
  if (job) void SpeechPlaying(job, Number(audio.dataset.seq ?? 0)).catch(() => {})
  // Routed here rather than at creation: setSinkId is a promise, and a piece
  // built while the one before it is still playing would race play().
  await applySpeaker(audio)
  try {
    await audio.play()
  } catch {
    stopSpeech()
  }
}

/** Silence, now. Also the normal end of a finished read. */
export function stopSpeech(): void {
  current?.pause()
  current = null
  // Dropping the src is what lets a queued fetch be abandoned rather than
  // run to completion for audio that will never be played.
  for (const queued of queue) queued.src = ''
  queue = []
  held = []
  last = false
  speech.key = ''
  speech.busyKey = ''
  voice.speaking = false
  onError = undefined
  generation++
  const ended = job
  job = ''
  // Both ends of the read close here: the backend cancels the synthesis in
  // flight and deletes the pieces. This is also the normal end of a finished
  // read — "the audio is over" and "the files can go" are one moment, and
  // this is the only side that knows it.
  if (ended) void StopSpeech(ended).catch(() => {})
}

/** Stop only if `key` is the read playing — a caller ending its own read
 *  without silencing somebody else's. */
export function stopSpeechIf(key: string): void {
  if (speech.key === key) stopSpeech()
}

/** Read `text` aloud as `key`, replacing whatever was playing. Errors — the
 *  engine refusing, a piece failing — go to `report`; a caller that passes
 *  none gets silence. */
export async function speak(key: string, text: string, report?: (msg: string) => void): Promise<void> {
  wire()
  stopSpeech()
  const said = speechText(text)
  if (!said) return
  onError = report
  speech.busyKey = key
  speech.key = key
  voice.speaking = true
  starting = true
  const mine = generation
  try {
    const started = await StartSpeech(said)
    // Stopped — or replaced, even by a read with the same key — while the
    // engine was being resolved. Close this read rather than start playing
    // it over the one that took its place.
    if (mine !== generation) {
      void StopSpeech(started).catch(() => {})
      return
    }
    job = started
    const early = held
    held = []
    for (const c of early) if (c.job === started) accept(c)
  } catch (err) {
    // A superseded read's failure is nobody's news: the read that replaced
    // it is the one playing, and stopping it here would be the echo's cousin.
    if (mine !== generation) return
    stopSpeech()
    report?.(String(err))
  } finally {
    // Only the read still current may lower the flag: a superseded read's
    // finally would otherwise drop the pieces the current one is holding.
    if (mine === generation) {
      starting = false
      held = []
    }
  }
}

/** What gets spoken: the reply without its markdown scaffolding. The text on
 *  screen renders those marks away; a voice that reads "ดอกจัน" out loud is
 *  reading the source, not the answer. */
export function speechText(md: string): string {
  return md
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`([^`]*)`/g, '$1')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/^[-*+]\s+/gm, '')
    .replace(/^>\s?/gm, '')
    .replace(/[*_~|]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}
