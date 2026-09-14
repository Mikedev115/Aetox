// Where the engine is, as the window last heard (§248 phases 2 and 3).
// One store rather than each card reading the binding for itself: the
// status chip, the Settings page and the folder door all need the same
// answer to "is the engine on this machine or on a host", and a window on
// a host must open folders THERE — through the remote picker — not with
// the native dialog that browses these disks.

import { EngineStatus as engineStatusBinding, AnswerHostDir } from '../../../wailsjs/go/main/App'
import type { main } from '../../../wailsjs/go/models'

export type EngineMode = 'local' | 'remote' | 'attach'

export const engine = $state({
  status: {
    state: 'connected', detail: '', restarts: 0, pid: 0, address: '', mode: 'local', host: '',
  } as main.EngineStatus,
  /** An event has arrived; a later read of the binding must not overwrite it. */
  heard: false,
  /** The remote folder picker is up (RemoteDirPicker.svelte). */
  pickerOpen: false,
  /** A door on the Go side is waiting for a folder on the host (§248 phase
   *  4, host_files.go): the picker is up for it, titled as the dialog would
   *  have been, and AnswerHostDir carries the pick back by this id. */
  hostDirAsk: null as HostDirAsk | null,
})

/** One folder question from the screen's Go side — the screen:pickdir event. */
export type HostDirAsk = { id: string; title: string; start: string }

/** Answer the open folder question — a path, or '' for cancelled — and take
 *  the picker down. The id is read BEFORE the ask is cleared: the picker's
 *  own `{@const ask}` is a derived read, and clearing first left it null by
 *  the time the binding was called (seen live, 14 ก.ย. 2026). */
export async function answerHostDir(path: string): Promise<void> {
  const ask = engine.hostDirAsk
  if (!ask) return
  engine.hostDirAsk = null
  try {
    await AnswerHostDir(ask.id, path)
  } catch {
    // The door on the Go side lets the question go after its own wait; a
    // binding that failed here is the wire's to report.
  }
}

export function applyEngineStatus(st: main.EngineStatus): void {
  engine.heard = true
  engine.status = st
}

/** The state now, for a window that mounted after the event went by. */
export async function loadEngineStatus(): Promise<void> {
  try {
    const st = await engineStatusBinding()
    if (st && !engine.heard) engine.status = st
  } catch {
    // A binding that failed is a status of its own; the chip hears it.
  }
}

export function engineIsRemote(): boolean {
  return engine.status.mode === 'remote'
}

/** Reset — tests only. */
export function resetEngineStore(): void {
  engine.status = {
    state: 'connected', detail: '', restarts: 0, pid: 0, address: '', mode: 'local', host: '',
  } as main.EngineStatus
  engine.heard = false
  engine.pickerOpen = false
  engine.hostDirAsk = null
}
