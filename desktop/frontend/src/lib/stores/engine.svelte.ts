// Where the engine is, as the window last heard (§248 phases 2 and 3).
// One store rather than each card reading the binding for itself: the
// status chip, the Settings page and the folder door all need the same
// answer to "is the engine on this machine or on a host", and a window on
// a host must open folders THERE — through the remote picker — not with
// the native dialog that browses these disks.

import { EngineStatus as engineStatusBinding } from '../../../wailsjs/go/main/App'
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
})

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
}
