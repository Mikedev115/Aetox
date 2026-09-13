// เรียกให้หัน — the ways the window reaches a person who is not looking at
// it (desktop/attention.go holds the switches and the Win32 half).
//
// Owner, 12 ก.ย. 2026: *"เวลาเอเจนถาม มันไม่มีอะไรแจ้งเตือนเลย เซสชั่นนั้นจะนิ่ง
// และเงียบไป"*. Every signal the app had was drawn INSIDE the window — a dot
// on a sidebar row, a card in one chat's transcript — so a person in another
// program, or reading another chat, was told nothing, and `ask_user` waits
// with no deadline. And 13 ก.ย., after the flash and the chime had shipped:
// *"เวลาเราทิ้งให้มันทำงานมันไม่มีแจ้งเตือนครับ คนไม่รู้"* — hence the third.
//
// Three signals:
//   - the **chime**, played here with WebAudio (no asset to ship, no file to
//     go missing) — a question always earns it, a finished turn only when it
//     finished somewhere the user was not looking;
//   - the **flash**, asked of Go, which flashes the taskbar button until the
//     window is brought to the front;
//   - the **notification**, also Go's: Windows' own, in the corner and in the
//     notification centre, naming the chat and what it wants, and bringing
//     the window back when pressed.
//
// Whether the window is "away" is Go's answer, from the desktop's foreground
// window, not this document's `hasFocus()`. It used to be this document's,
// and only when it said "not focused" was Go even asked — so a webview that
// was wrong about its own focus kept every signal in. Both are still read,
// and either saying "away" is enough: a signal one side missed is worse than
// one both sides raise.
//
// Deliberately no in-window toast: the app has none (SlidesPane.svelte's
// note), and WHICH chat is asking is the sidebar row's and the topbar strip's
// to say inside the window. The notification carries the name outside it.

import { AttentionSignal, RequestAttention, SetAttentionSignal } from '../../../wailsjs/go/main/App'
import type { engine } from '../../../wailsjs/go/models'
import { t } from '../i18n.svelte'

export const attention = $state<{ layers: engine.BusyLayer[]; loaded: boolean }>({
  layers: [],
  loaded: false,
})

/** Is this switch on? True before the first read lands — all ship on, and a
 * question that arrives before the settings were read still deserves a sound. */
export function attentionOn(id: 'flash' | 'chime' | 'toast'): boolean {
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

/** What happened, and therefore which sound and which words. */
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
 * below the fold or behind another page. `title` names the chat for the
 * notification; empty when it has none yet (a first turn).
 *
 * Go is asked every time. It holds the switches for the flash and the
 * notification, it alone can see the desktop's foreground window, and its
 * answer — was the user away? — is what decides the chime for a turn that
 * ended on screen. Fire-and-forget from the caller's point of view: a turn's
 * ending must not wait on a notification. */
export function attend(kind: AttentionKind, onScreen: boolean, title = ''): void {
  const name = title.trim() || t('topbar.askingUnnamed')
  const text = kind === 'ask' ? t('attention.askToast') : t('attention.doneToast')
  void RequestAttention(kind, name, text)
    .then((away) => away === true)
    .catch(() => false)
    .then((awayByOS) => {
      const away = awayByOS || !focused()
      if ((kind === 'ask' || !onScreen || away) && attentionOn('chime')) chime(kind)
    })
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
    // the flash, the notification and the marks in the window still say it.
  }
}
