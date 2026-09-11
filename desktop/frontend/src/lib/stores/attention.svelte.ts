// เรียกให้หัน — the two ways the window reaches a person who is not looking
// at it (desktop/attention.go holds the switches and the Win32 half).
//
// Owner, 12 ก.ย. 2026: *"เวลาเอเจนถาม มันไม่มีอะไรแจ้งเตือนเลย เซสชั่นนั้นจะนิ่ง
// และเงียบไป"*. Every signal the app had was drawn INSIDE the window — a dot
// on a sidebar row, a card in one chat's transcript — so a person in another
// program, or reading another chat, was told nothing, and `ask_user` waits
// with no deadline.
//
// Two signals:
//   - the **chime**, played here with WebAudio (no asset to ship, no file to
//     go missing) — a question always earns it, a finished turn only when it
//     finished somewhere the user was not looking;
//   - the **flash**, asked of Go, which flashes the taskbar button until the
//     window is brought to the front — only when the window is not focused,
//     because flashing the window somebody is looking at is noise.
//
// Deliberately no toast: the app has none (SlidesPane.svelte's note), and
// WHICH chat is asking is the sidebar row's and the topbar strip's to say.
// These two only say "come back".

import { AttentionSignal, RequestAttention, SetAttentionSignal } from '../../../wailsjs/go/main/App'
import type { engine } from '../../../wailsjs/go/models'

export const attention = $state<{ layers: engine.BusyLayer[]; loaded: boolean }>({
  layers: [],
  loaded: false,
})

/** Is this switch on? True before the first read lands — both ship on, and a
 * question that arrives before the settings were read still deserves a sound. */
export function attentionOn(id: 'flash' | 'chime'): boolean {
  const found = attention.layers.find((l) => l.id === id)
  return found ? found.on : true
}

export async function loadAttention(): Promise<void> {
  if (attention.loaded) return
  try {
    attention.layers = (await AttentionSignal()) ?? []
  } catch {
    // Unreadable preference: attentionOn's fallback is the shipped answer.
  }
  attention.loaded = true
}

export async function toggleAttention(id: string, on: boolean): Promise<void> {
  try {
    attention.layers = (await SetAttentionSignal(id, on)) ?? attention.layers
    attention.loaded = true
  } catch {
    // Left as it was — a switch that says it took and did not is worse.
  }
}

/** What happened, and therefore which sound. */
export type AttentionKind = 'ask' | 'done'

/** The window's own idea of whether anyone is looking at it. A function so a
 * test can stand in for the document. */
let focused: () => boolean = () => {
  try {
    return document.hasFocus()
  } catch {
    return true
  }
}

/** Test seam. */
export function setFocusProbe(probe: (() => boolean) | null): void {
  focused = probe ?? (() => { try { return document.hasFocus() } catch { return true } })
}

/** Wake the user. `onScreen` says whether the chat this is about is the one
 * they are reading — a finished turn in that chat needs nothing, its answer
 * just landed in front of them; a question does, because its card may be
 * below the fold or behind another page. */
export function attend(kind: AttentionKind, onScreen: boolean): void {
  const away = !focused()
  const wantChime = kind === 'ask' || !onScreen || away
  if (wantChime && attentionOn('chime')) chime(kind)
  if (away && attentionOn('flash')) {
    // Fire and forget: Go checks the foreground window again, and a flash
    // that could not be asked for has nothing to be told about.
    void RequestAttention().catch(() => {})
  }
}

/** Two short notes, rising for a question and falling for a finished turn —
 * the same vocabulary a phone uses, and the two are told apart without a
 * glance. Quiet on purpose: this is a nudge across the room, not an alarm. */
let audio: AudioContext | null = null
function chime(kind: AttentionKind): void {
  try {
    audio ??= new AudioContext()
    const ctx = audio
    if (ctx.state === 'suspended') void ctx.resume()
    const notes = kind === 'ask' ? [880, 1174.66] : [1046.5, 783.99]
    const at = ctx.currentTime
    notes.forEach((hz, i) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.value = hz
      const start = at + i * 0.14
      gain.gain.setValueAtTime(0, start)
      gain.gain.linearRampToValueAtTime(0.12, start + 0.015)
      gain.gain.exponentialRampToValueAtTime(0.0005, start + 0.16)
      osc.connect(gain).connect(ctx.destination)
      osc.start(start)
      osc.stop(start + 0.18)
    })
  } catch {
    // No audio device, or a webview that refuses to play before a gesture —
    // the flash and the marks in the window still say it.
  }
}
