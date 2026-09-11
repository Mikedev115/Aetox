// One place owns "is there a newer Aetox, and where is it up to".
//
// Three doors lead to the same act: the notice that appears by itself when the
// automatic check finds something (update_notify.go emits `update:available`),
// the button in Settings → About that the user pressed on purpose, and the
// version row in the profile menu — which is where somebody who never opens a
// settings page still finds out a new Aetox exists. Before this, About owned
// that state privately — so a second door meant a second copy of "downloading /
// 42% / ready / it failed", free to disagree with the first. All three now read
// and drive this: the running version, the answer to "is there a newer one",
// and how far the download got.
//
// The phases are two acts, not one, because they cost the user different
// things (internal/update's Stage/Restart): downloading is bandwidth and can
// happen while they work; restarting costs them whatever they were in the
// middle of. So `ready` is a real resting state — the download is verified
// and on disk, and stays there across launches (internal/update's Adopt) until
// the restart the user picks the moment for. What "on disk" buys differs by
// channel, and the card says which: on portable the exe already IS the new
// build, so closing the window tonight installs it too; on installer only the
// button does, because an installer has to run with the app closed.
//
// What this module does NOT decide: which action a channel deserves. Scoop gets
// its command, portable and installer get the one-click button, everything else
// gets the release page — internal/update already makes that call and puts it in
// Status.canAuto / .hint, and re-deriving it here would be the same duplication
// one layer down.

import {
  StageUpdate, RestartToUpdate, StagedUpdate, CheckForUpdate, AppVersion,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { main, update } from '../../wailsjs/go/models'
import { t } from './i18n.svelte'

/** idle → downloading → ready → (restarting). `error` can follow either act. */
export type UpdatePhase = 'idle' | 'downloading' | 'ready' | 'restarting' | 'error'

export const updater = $state<{
  /** The version this build is, straight from Go. Empty until asked — the menu
   *  shows a dash rather than guessing, because a version number the app made
   *  up is worse than one it has not fetched yet. */
  current: string
  /** The newest release the app knows about, or null if it never found one. */
  status: update.Status | null
  /** True when the standing offer arrived on its own (the daily check's
   *  `update:available`), false when the user went and asked. Only the first
   *  kind earns the card that covers the window: a check the user pressed is
   *  answered where they pressed it, and a card on top of that answer would be
   *  the same news twice. */
  announced: boolean
  /** A check is in flight. Its own flag rather than a phase, because checking
   *  and downloading can be true of the same minute — the phases below are
   *  about bytes, this is about a question. */
  checking: boolean
  /** Why the last check did not complete, or "". Separate from `error`: a
   *  check that could not reach GitHub says nothing about a download, and
   *  folding the two would let one overwrite the other's sentence. */
  checkError: string
  /** The user said "later". Holds for this run only — the notice is an offer,
   *  not a to-do item, and re-asking on the next launch is the whole point.
   *  Dismissing a *ready* update costs nothing: it is already on disk. */
  dismissed: boolean
  phase: UpdatePhase
  /** Bytes, as the download actually knows them. total is 0 when the server
   *  never said how big the file is. */
  done: number
  total: number
  /** The version waiting on disk once phase is 'ready'. */
  staged: string
  /** Which channel staged it — the one sentence on the ready card differs by
   *  it. On portable the exe on disk already IS the new build and closing the
   *  app installs it; on installer nothing has moved and only the restart
   *  button does. Kept beside `staged` rather than read off `status`, because
   *  an update adopted at launch is ready before any check has answered. */
  stagedChannel: string
  /** Empty unless something failed, in which case the build the user is
   *  running is still installed and untouched. */
  error: string
}>({
  current: '', status: null, announced: false, checking: false, checkError: '',
  dismissed: false, phase: 'idle', done: 0, total: 0, staged: '', stagedChannel: '', error: '',
})

/** How long the window waits for the Go side to actually close it after
 *  RestartToUpdate resolved. Quitting takes well under a second normally — the
 *  close hook holds it for at most five while turns write their ending — so a
 *  window still open after this is one the restart did not take, and a card
 *  stuck on "กำลังเปิดใหม่…" with no button is the worst thing it can show. */
const RESTART_GRACE_MS = 30_000

/** True when there is an offer to put in front of the user right now. */
export function updateOffered(): boolean {
  return !!updater.status?.available && !updater.dismissed
}

/** True when the offer arrived unasked and so still has to be carried to the
 *  user by the card. A door that shows the offer itself uses `updateOffered`. */
export function updateAnnounced(): boolean {
  return updateOffered() && updater.announced
}

/** 0–100, or -1 when nothing is downloading or the size is unknown. */
export function updatePct(): number {
  return updater.total > 0 ? Math.min(100, Math.round((updater.done / updater.total) * 100)) : -1
}

/** Which version this build is, for every door that names it. One Go call per
 *  run — the number cannot change while the process lives — and safe to call
 *  from anywhere, since the second caller finds it already answered. */
export function loadCurrentVersion(): void {
  if (updater.current) return
  void AppVersion().then((v) => { updater.current = v }).catch(() => {
    /* backend not up yet: the door shows a dash rather than a wrong number */
  })
}

/** Called once from App. Returns the unsubscriber Svelte's onMount expects. */
export function listenForUpdates(): () => void {
  loadCurrentVersion()

  const offAvailable = EventsOn('update:available', (st: update.Status) => {
    if (!st?.available) return
    updater.announced = true
    // A newer release than the one already on offer un-dismisses the notice:
    // "later" was said about a different version.
    if (updater.status && updater.status.latest !== st.latest) updater.dismissed = false
    updater.status = st
  })
  const offProgress = EventsOn('update:progress', (p: { done: number; total: number }) => {
    updater.done = p?.done ?? 0
    updater.total = p?.total ?? 0
  })
  // The Go side's own word on what is staged, sent when it changes without
  // the window having asked: the launch that adopted an installer a previous
  // run downloaded (and, with it, why the previous restart came back as the
  // same build), or a newer release making that installer stale.
  const offStaged = EventsOn('update:staged', (info: main.StagedInfo) => adoptStaged(info, true))
  // A webview reload (Vite HMR, or a crash of the frontend alone) leaves the Go
  // side holding a staged update this fresh page knows nothing about. Without
  // this the card would vanish and the user would be offered the same download
  // a second time — for a build already sitting on their disk.
  //
  // An empty answer here is NOT a drop: the Go side adopts off the startup
  // path, so this question can be asked before it has looked and answered
  // "nothing" a moment before `update:staged` says otherwise — and the reverse
  // order would clear a card the event just put up.
  void StagedUpdate().then((info) => adoptStaged(info, false)).catch(() => {
    /* nothing staged, or the binding is unavailable in a test: idle is right */
  })
  return () => {
    offAvailable()
    offProgress()
    offStaged()
  }
}

/** Takes the Go side's answer about a staged update into the phases. Only
 *  moves between idle and ready — a download or restart in flight owns the
 *  phase, and the answer will be asked for again when it ends. `emptyIsDrop`
 *  says whether an empty answer means "let go of what you have" (the event)
 *  or merely "nothing yet" (the reload query). */
function adoptStaged(info: main.StagedInfo | null | undefined, emptyIsDrop: boolean): void {
  if (!info) return
  if (info.version) {
    if (updater.phase !== 'idle') return
    updater.staged = info.version
    updater.stagedChannel = info.channel ?? ''
    updater.phase = 'ready'
    // The card can say what went wrong last time only if it has the words;
    // an adopted installer whose restart was declined at the UAC prompt is
    // the case that used to read as "อัปเดตแล้ววนอยู่ที่เดิม".
    if (info.installError) updater.error = info.installError
    // A ready update the user has not been told about yet is news, whichever
    // door — the card carries it, not a check that may never come back.
    updater.announced = true
    return
  }
  // Nothing staged any more: the installer the card was offering a restart
  // into has been overtaken (dropStaleStaged) — back to the offer, which
  // `update:available` fills in.
  if (emptyIsDrop && updater.phase === 'ready') {
    updater.staged = ''
    updater.stagedChannel = ''
    updater.phase = 'idle'
    updater.error = info.installError ?? ''
  }
}

export function dismissUpdate(): void {
  updater.dismissed = true
}

/** Ask GitHub now, because the user asked. Both doors that carry a button call
 *  this — Settings → About and the profile menu — so the answer they show is
 *  one answer, not two that can disagree by a release.
 *
 *  A failed check does NOT wipe a status that was already found: "could not
 *  reach GitHub just now" is not evidence that the release announced an hour
 *  ago stopped existing, and erasing it would take a real offer off the screen
 *  because the wifi dropped. */
export async function checkNow(): Promise<void> {
  if (updater.checking) return
  updater.checking = true
  updater.checkError = ''
  try {
    const st = await CheckForUpdate()
    // A different release than the one already waved off is news again — the
    // same rule the announcement follows, for the same reason.
    if (updater.status && updater.status.latest !== st.latest) updater.dismissed = false
    updater.status = st
    // Asked for, so answered where it was asked. See `announced`.
    updater.announced = false
    // Status.current is deliberately NOT copied into updater.current. It is
    // the same Go constant AppVersion returns, and a fact with two writers is
    // a fact that can be written twice differently — which is the whole reason
    // this module exists.
  } catch (err) {
    // Offline, rate-limited, a proxy eating the response: say so and change
    // nothing else. A failed check is not a broken app.
    updater.checkError = String(err)
  } finally {
    updater.checking = false
  }
}

/** Download and verify. Changes nothing the user can see when it finishes —
 *  the app is one restart away, and the restart is their call. */
export async function startDownload(): Promise<void> {
  if (updater.phase === 'downloading' || updater.phase === 'restarting') return
  updater.phase = 'downloading'
  updater.error = ''
  updater.done = 0
  updater.total = 0
  try {
    await StageUpdate()
    updater.staged = updater.status?.latest ?? ''
    updater.stagedChannel = updater.status?.channel ?? ''
    updater.phase = 'ready'
  } catch (err) {
    updater.error = String(err)
    updater.phase = 'error'
  }
}

/** Close this build and come back as the new one. The Go side quits the app a
 *  moment after this resolves, so `restarting` never has to be undone. */
export async function restartToUpdate(): Promise<void> {
  if (updater.phase === 'restarting') return
  updater.phase = 'restarting'
  updater.error = ''
  try {
    await RestartToUpdate()
  } catch (err) {
    // A turn is running, most likely. Nothing was lost — the new build is
    // still staged on disk — so this goes back to offering the restart, never
    // back to offering a download that has already happened.
    updater.error = String(err)
    updater.phase = 'ready'
    return
  }
  // Resolved means the waiter is up and the Go side will quit in a moment. If
  // this code is still running well past that moment, the quit did not
  // happen, and the user needs the button back rather than a spinner.
  setTimeout(() => {
    if (updater.phase !== 'restarting') return
    updater.error = t('update.restartStalled')
    updater.phase = 'ready'
  }, RESTART_GRACE_MS)
}
