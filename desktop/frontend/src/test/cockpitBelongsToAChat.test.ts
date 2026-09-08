// Every field on `cockpit` has to have an answer to one question: when the user
// switches conversations, what happens to it?
//
// Three bugs have been the same missing answer. The desk file tab (§187) landed
// its tab on whichever chat was on screen. The undo chip rode across and offered
// to put back a file another conversation wrote. The wording Tab types rode
// across and answered a question asked somewhere the user was no longer looking
// (2026-09-08). Each was found by a person noticing, months apart, and each fix
// went in the same place — `arriveAt()`.
//
// The reason it kept happening is worth stating, because it is not carelessness.
// In all three the ARRIVING event was already guarded: it carries a sessionId
// and the store drops what is not the chat on screen. That guard is the door
// everybody inspects, and it is not the door the state comes through. The state
// was already sitting on `cockpit` when the switch happened, put there while the
// chat WAS on screen, and nothing took it off on the way out.
//
// So this test is not about correctness — a table cannot know whether a field is
// handled RIGHT. It is about a decision existing at all. Add a field to
// CockpitState and this test fails until somebody writes down which of the five
// kinds it is, which puts the question in front of the one person who can still
// answer it cheaply: whoever is adding it.
import { describe, it, expect } from 'vitest'
import { cockpit } from '../lib/stores/cockpit.svelte'

/** What a field does when the user opens another conversation. */
type Kind =
  /** Not a chat's at all — the app, the project, the window. Untouched. */
  | 'app'
  /** The composer in front of the user. Deliberately NOT per-chat: what the
   *  user typed follows the user, on the same reasoning a half-written email
   *  does not empty itself when you look at another one. The line against
   *  `prepared` below is that a wording is not what the user wrote — it is an
   *  answer to a question one chat asked, so it belongs to that chat. */
  | 'composer'
  /** Names which chat is which. Read and written BY the switch itself. */
  | 'identity'
  /** Dropped on the way in and asked for again, because the engine still knows
   *  it: an undo chip, the task chips, the bill, the desk, the stance. Cheap to
   *  re-read and always right. */
  | 'dropped'
  /** Carried by parkLive/restoreLive — live turn state, so a chat left working
   *  comes back mid-flight. Gated on `awaitingReply`, which is why it cannot
   *  hold anything that exists only AFTER a turn ends. */
  | 'parked-live'
  /** Held in a map of its own, because it is pushed once and stored nowhere:
   *  dropping it would destroy something that cannot be asked for again. */
  | 'parked-own'

const decided: Record<string, Kind> = {
  // ---- the app, the project, the window ----
  project: 'app', projectFolders: 'app', projects: 'app', tree: 'app',
  browseRoot: 'app', sessions: 'app', history: 'app', historyFault: 'app',
  spaceHistory: 'app', spaces: 'app', stances: 'app', activeView: 'app',
  openFiles: 'app', settingsIntent: 'app', pendingLearned: 'app',
  pendingIssues: 'app', backgroundTasks: 'app', backgroundRuns: 'app',
  backgroundSteps: 'app',

  // ---- which chat is which ----
  openSession: 'identity', turnSession: 'identity', parked: 'identity',
  // Not one chat's state but a map ABOUT chats, like `parked` above it, and
  // written by the switch itself: arriveAt clears the arriving chat's mark,
  // which is what "opened, therefore read" means. Nothing to park and nothing
  // to drop — the entries that matter are always other conversations'.
  unread: 'identity',

  // ---- the box in front of the user ----
  pendingImages: 'composer', pendingContexts: 'composer', pendingFiles: 'composer',

  // ---- dropped, then asked for again ----
  // Every one of these has an engine-side reader, which is what makes dropping
  // safe: sessionSpend/refreshSessionSpend, undoFiles/refreshUndo,
  // taskChips/refreshTaskChips, desk & chair & space & stance re-read at the
  // tail of every door.
  sessionSpend: 'dropped', undoFiles: 'dropped', taskChips: 'dropped',
  sessionError: 'dropped', desk: 'dropped', chair: 'dropped', space: 'dropped',
  stance: 'dropped', model: 'dropped', restorePoints: 'dropped',
  // The plan (desktop/plan.go). Dropped and re-read through `SessionPlan`,
  // because it is a row keyed by session id and the engine can always be asked
  // again — the same reasoning as the undo chip, and the opposite of `prepared`
  // below, which nothing can hand back.
  //
  // This row exists because the test demanded it: `plan` was added to
  // CockpitState and the suite failed until somebody chose. That is the whole
  // instrument working, on the first field added after it existed.
  plan: 'dropped',

  // ---- carried across, mid-flight ----
  chat: 'parked-live', awaitingReply: 'parked-live', agentStatus: 'parked-live',
  toolSteps: 'parked-live', turnFiles: 'parked-live', turnProposals: 'parked-live',
  streamingText: 'parked-live', reasoningText: 'parked-live',
  modelLoading: 'parked-live', ask: 'parked-live', todos: 'parked-live',
  turnSpend: 'parked-live', task: 'parked-live',
  // The takeover strip (computer tool). Parked rather than dropped, and the
  // two wrong answers are worth naming because both look right.
  //
  // Dropped would be wrong: nothing on the engine side can be asked "is this
  // chat driving something", so a strip dropped on the way out never comes
  // back, and the user returns to a chat that is holding their machine with
  // nothing on screen saying so.
  //
  // Left on cockpit would be worse: the strip would follow the user into
  // another conversation and tell them THAT chat was driving a window, which
  // is the exact shape of the three bugs at the top of this file.
  //
  // Parked is safe here for a reason that does not generalise: the strip only
  // exists between takeTheScreen and its deferred release, both inside one
  // tool call, so awaitingReply is true for the whole of its life and
  // parkLive's gate can never miss it.
  driving: 'parked-live',

  // ---- held in its own map ----
  // A prepared wording exists BECAUSE a turn ended, so it can never satisfy
  // parkLive's `awaitingReply` gate, and nothing re-reads it: it is pushed once
  // on `composer:prepared` and stored nowhere. Park is the only answer left.
  prepared: 'parked-own', preparedAt: 'parked-own',
}

describe('every piece of cockpit state knows whose chat it is', () => {
  it('has a decision written down for each field', () => {
    const undecided = Object.keys(cockpit).filter((k) => !(k in decided))
    expect(
      undecided,
      'A new field on CockpitState. Before this passes, decide what happens to it ' +
        'when the user opens another conversation, and add it to `decided` above. ' +
        'Three bugs have been this decision going unmade — see the note at the top ' +
        'of this file and arriveAt() in cockpit.svelte.ts.',
    ).toEqual([])
  })

  it('has no decision for a field that is gone', () => {
    const live = new Set(Object.keys(cockpit))
    expect(
      Object.keys(decided).filter((k) => !live.has(k)),
      'A field was removed but its decision was left behind. Delete the row.',
    ).toEqual([])
  })
})
