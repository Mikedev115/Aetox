package main

// What happens to work in flight when the app is closed (§219).
//
// Closing the window used to be the one way to end a turn that wrote nothing
// down. Stop writes the turn's ending (appendFailedTurn, `context canceled`);
// a quota or a dropped connection writes it; the window's X did not — the turn
// goroutine simply died with the process, mid-tool-call, and the question
// openTurn had stored sat alone forever. Reopen the chat and it read as if the
// agent had never answered: no ending, no partial text, no tool timeline, no
// ลองใหม่ chip. The owner's words (3 ก.ย.): "แทนที่จะถูกหยุดอย่างถูกวิธี ดันหายไปเลย".
//
// Two layers, because the process can end two ways.
//
// The polite way — X, Quit, the self-update's restart — passes through Wails'
// OnBeforeClose, which is beforeClose below: it presses Stop on everything
// (every conversation's turn and every delegate, not only the chat on screen),
// then holds the close for as long as the turns need to unwind and write their
// ending — bounded, so a tool that ignores its context cannot hold the window
// hostage. A turn ended this way records the marker below alongside the
// cancel, so the chat reads "the app was closed" rather than "you pressed
// Stop", which the user did not.
//
// The other way — a crash, a task kill, a power cut, or the bound above running
// out — writes nothing, so the next launch does: closeInterruptedTurns
// (sessions.go) finds every question still waiting for an answer and closes it
// with the same marker. Both roads end in the same row, so the window has one
// ending to draw.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// closeGrace is how long beforeClose holds the window for the turns to write
// their ending. Cancellation reaches a provider request, a shell, a parked
// question and a delegate through their contexts, so the ordinary case takes
// well under a second; the cap exists for the tool that does not listen.
const closeGrace = 5 * time.Second

// beforeClose is the Wails OnBeforeClose hook (wired in main.go). Never
// prevents the close — the user asked for it — but does not let it happen
// until the work in flight has been stopped and written down, or the grace has
// run out.
func (a *App) beforeClose(_ context.Context) (prevent bool) {
	a.finishTurnsForClose(closeGrace)
	return false
}

// finishTurnsForClose stops every turn and delegate in every conversation and
// waits, up to grace, for the turns to end. Reports whether they all did.
//
// Idempotent: the OnShutdown hook calls it too, so a close that reached
// shutdown without passing beforeClose (nothing in Wails does today) still
// ends the work before the store it would write to is closed.
func (a *App) finishTurnsForClose(grace time.Duration) bool {
	a.closing.Store(true)
	stopped := a.stopEverything()
	if !a.turnBusy() {
		return true
	}
	debuglog.Msg("close: %d turn(s) still running — waiting up to %s for them to write their ending", stopped, grace)
	deadline := time.Now().Add(grace)
	for a.turnBusy() {
		if time.Now().After(deadline) {
			debuglog.Msg("close: grace over with a turn still running; the next launch closes its question")
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return true
}

// stopEverything is CancelTurn for every conversation at once, and returns how
// many turns it found running.
//
// CancelTurn deliberately reaches only the chat on screen, because Stop is a
// button on that chat. Closing the app is not a statement about one chat; a
// turn working off screen dies with the process exactly like the one on it.
func (a *App) stopEverything() int {
	a.turnMu.Lock()
	defer a.turnMu.Unlock()
	for _, conv := range a.allConversations() {
		if conv.delegations != nil {
			if n := conv.delegations.StopAll(); n > 0 {
				debuglog.Msg("close: ended %d running sub-agent(s) in %s", n, conv.id)
			}
		}
		if conv.agent != nil {
			conv.agent.DrainInterjections()
		}
	}
	running := 0
	for _, live := range a.turns {
		running++
		if live.cancel != nil {
			live.cancel()
		} else {
			// Same window CancelTurn covers: between beginTurn and the cancel
			// func existing. armTurnCancel consumes the flag and cancels at once.
			live.stopEarly = true
		}
	}
	return running
}

// allConversations is every live conversation, the one on screen included —
// a.cur()'s manager, built on demand the same way.
func (a *App) allConversations() []*conversation {
	a.cur() // builds a.convs once, for the zero App the tests construct
	return a.convs.all()
}

// isClosing reports whether the app is on its way out — set by beforeClose
// before it cancels anything, so a turn that ends with `context canceled` from
// here on can tell that ending from the user's Stop.
func (a *App) isClosing() bool {
	return a.closing.Load()
}

// closeReason is the error a turn records when the app's own close is what
// cancelled it. The marker goes in front so the window's one predicate for
// "the app was closed" (cockpit.svelte.ts, wasInterrupted) reads it, and the
// cancel stays behind `%w` so everything that already recognises a Stop —
// errors.Is, the executor's suffix fallback, the retry chip — keeps working.
//
// Any other error passes through untouched: a quota that ran out a moment
// before the user closed the window ran out, and saying the close did it
// would be a lie about what happened.
func (a *App) closeReason(err error) error {
	if err == nil || !a.isClosing() || !endedByCancel(err) {
		return err
	}
	if strings.Contains(err.Error(), interruptedTurnMarker) {
		return err
	}
	return fmt.Errorf("%s: %w", interruptedTurnMarker, err)
}

// endedByCancel is the executor's own test for a turn ended by its context
// (internal/turn/executor.go): the sentinel by identity, or Go's one spelling
// of it at the end of an error that was flattened on its way up.
func endedByCancel(err error) bool {
	return errors.Is(err, context.Canceled) ||
		strings.HasSuffix(err.Error(), context.Canceled.Error())
}
