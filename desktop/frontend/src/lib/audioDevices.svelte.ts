// Which microphone the composer records from, and which speaker the ฟัง
// button plays out of. Both used to be whatever Windows had picked as its
// default, chosen nowhere and visible nowhere — which on a machine with four
// capture devices (a USB mic, a headset jack with nothing in it, the laptop
// array, and NVIDIA Broadcast's virtual mic) is a coin toss between a working
// recording and a silent one. The silent one comes back as "ไม่ได้ยินเสียงพูด
// ในบันทึก", an error about the user's voice for a fault in device routing
// (owner, 8 ก.ย. 2026).
//
// localStorage rather than the app DB on purpose: a device id is a fact about
// THIS machine's hardware, and it does not survive being carried to another
// one — the same reason the font stores live here.

const MIC_KEY = 'aetox-audio-input'
const SPEAKER_KEY = 'aetox-audio-output'

export type AudioDeviceRow = { id: string; label: string }

export const audioDevices = $state<{
  micId: string
  speakerId: string
  mics: AudioDeviceRow[]
  speakers: AudioDeviceRow[]
  /** enumerateDevices withholds labels until a capture permission has been
   *  granted once — false means the lists are placeholders, not names. */
  labelled: boolean
  /** Set when a remembered device was gone and the default was used instead:
   *  falling back is right, falling back silently is how this bug started. */
  fellBack: boolean
}>({ micId: '', speakerId: '', mics: [], speakers: [], labelled: false, fellBack: false })

// Read at module load, not from an init in main.ts: nothing here is painted,
// so there is no flash to get ahead of.
try {
  audioDevices.micId = localStorage.getItem(MIC_KEY) ?? ''
  audioDevices.speakerId = localStorage.getItem(SPEAKER_KEY) ?? ''
} catch {
  // Private mode or blocked site data — the defaults below are the answer.
}

function remember(key: string, id: string): void {
  try {
    if (id) localStorage.setItem(key, id)
    else localStorage.removeItem(key)
  } catch {
    // A pick that cannot be saved still applies for this run.
  }
}

export function setMicId(id: string): void {
  audioDevices.micId = id
  audioDevices.fellBack = false
  remember(MIC_KEY, id)
}

export function setSpeakerId(id: string): void {
  audioDevices.speakerId = id
  remember(SPEAKER_KEY, id)
}

/** Fill the two lists from the browser. Safe to call repeatedly — devices come
 *  and go with a USB plug, and the picker must not show a stale set. */
export async function refreshAudioDevices(): Promise<void> {
  if (!navigator.mediaDevices?.enumerateDevices) return
  const all = await navigator.mediaDevices.enumerateDevices()
  const rows = (kind: MediaDeviceKind, prefix: string): AudioDeviceRow[] =>
    all
      .filter((d) => d.kind === kind && d.deviceId)
      .map((d, i) => ({ id: d.deviceId, label: d.label || `${prefix} ${i + 1}` }))
  audioDevices.mics = rows('audioinput', 'Mic')
  audioDevices.speakers = rows('audiooutput', 'Speaker')
  // One unlabelled entry is enough to know the permission has not been given:
  // the browser blanks every label or none.
  audioDevices.labelled = all.some((d) => d.kind !== 'videoinput' && !!d.label)
}

/** Open the mic the user chose, or Windows' default if they chose nothing.
 *  `exact` rather than `ideal` deliberately: a remembered device that has been
 *  unplugged must be a fact this returns, not a quiet swap to a mic the user
 *  is not speaking into. The retry is the fallback, and it sets fellBack. */
export async function openMicStream(): Promise<MediaStream> {
  audioDevices.fellBack = false
  const id = audioDevices.micId
  if (id) {
    try {
      return await navigator.mediaDevices.getUserMedia({ audio: { deviceId: { exact: id } } })
    } catch (err) {
      // Denied is denied — only a device that is GONE earns the second try.
      if (err instanceof DOMException && (err.name === 'NotAllowedError' || err.name === 'SecurityError')) throw err
      audioDevices.fellBack = true
    }
  }
  return await navigator.mediaDevices.getUserMedia({ audio: true })
}

/** Point one <audio> at the chosen speaker before it plays. A no-op when
 *  nothing is chosen, and when the webview has no setSinkId to point with. */
export async function applySpeaker(audio: HTMLAudioElement): Promise<void> {
  const id = audioDevices.speakerId
  // Cast, not a lib bump: setSinkId is Chromium-only and the DOM typings this
  // project builds against do not carry it.
  const sinkable = audio as HTMLAudioElement & { setSinkId?: (id: string) => Promise<void> }
  if (!id || typeof sinkable.setSinkId !== 'function') return
  try {
    await sinkable.setSinkId(id)
  } catch {
    // A speaker that has gone away plays on the default one. Losing the
    // routing is worth less than losing the audio.
  }
}
