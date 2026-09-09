package main

// The session's fourth coordinate — how the turn runs (DECISIONS.md §106).
//
// Everything about a stance that is *policy* lives in internal/mode/stance.go:
// which stances exist, what each one withholds, what direction it folds into
// the prompt. This file is only the three things a window needs — read it, list
// them, set it — and the one behaviour that is genuinely the desktop's: setting
// it re-bootstraps in place instead of opening a new session.

import (
	"github.com/Mikedev115/Aetox/internal/mode"
)

// Stances reports every stance this build implements, in the order the picker
// draws them, ลงมือ first.
//
// Ids, not labels. What each one is called on screen is a locale string, the
// same split COMPANY.md §2 keeps between a desk's code name and its room's
// label — so the engine cannot dictate a word and a translation cannot invent
// a stance. A frontend that hardcoded the list instead would be a second answer
// to "which stances exist", and it would be the one that goes stale.
func (a *App) Stances() []string {
	all := mode.Stances()
	out := make([]string, 0, len(all))
	for _, s := range all {
		out = append(out, s.String())
	}
	return out
}

// Stance is how the open session is being run right now, "" for ลงมือ.
//
// Answered from the engine rather than from the sessions table, and that is not
// an optimisation: a session row is only written on its first turn, so the chat
// in front of you usually has no row at all. Reading "" back for it would draw
// ลงมือ under a composer that is standing in คู่คิด. SessionMode has the same
// note for the same reason.
func (a *App) Stance() string {
	return a.cur().stance.String()
}

// SetStance changes how the open session runs, and returns the stance actually
// in force — which is ลงมือ for any name this build does not implement, never
// an error. A picker offering a stance the engine has never heard of is a bug
// on the other side of the wire; refusing the call would leave the window
// showing a stance nothing is enforcing, which is the worse of the two.
//
// **This is the one coordinate that moves without opening a session** (§106).
// desk, chair and space are all fixed for a session's life because each would
// change what the context already holds; a stance only ever subtracts from the
// desk, so it cannot bring another desk's tools into this one and there is
// nothing to protect by starting over.
//
// The re-bootstrap is the whole implementation. applyConfig carries the
// outgoing agent's context into the new one — written for model switches, and
// exactly as true here — so the conversation survives, while the dispatcher's
// filter and the system prompt are both rebuilt from the new value. Two things
// that must agree about what this session carries, rebuilt together, from one
// field.
func (a *App) SetStance(name string) (string, error) {
	// Narrowed to this chat, not the app. Changing the stance rebuilds the
	// engine of the conversation on screen, carrying its context across — and a
	// turn running in THAT conversation would finish on an agent it was not
	// started with. A turn running in some other chat is no longer any of this
	// function's business, which is the difference per-conversation engines
	// make: the gate is around the thing being rewritten, not around the app.
	if a.turnRunningIn(a.cur().id) {
		return a.cur().stance.String(), errTurnBusy
	}
	next := mode.NormalizeStance(name)
	if next == a.cur().stance {
		return next.String(), nil
	}
	a.cur().stance = next
	a.applyConfig(a.cur(), a.cfg)
	a.persistStance()
	return next.String(), nil
}

// persistStance writes the current stance onto the open session's row, so
// reopening the conversation comes back the way it was left.
//
// Best-effort and silent, because the row legitimately may not exist yet: a
// session is only written on its first turn, and switching stance before saying
// anything is an ordinary thing to do. That case needs no repair — the INSERT
// in openTurn/appendTurn carries a.cur().stance with it, so the first message files
// the session with the stance it was actually held in.
func (a *App) persistStance() {
	if a.cur().id == "" {
		return
	}
	db, err := a.database()
	if err != nil {
		return
	}
	_, _ = db.Exec(`UPDATE sessions SET stance = ? WHERE id = ?`, a.cur().stance.String(), a.cur().id)
}

// stanceNarrowedByAgent is what happens when the ASSISTANT turns its own dial
// down mid-turn (internal/mode/plan_mode.go), handed to bootstrap as
// Options.OnStance. The engine has already narrowed by the time this runs — the
// dispatcher reads a live dial, which is the whole point — so this is not the
// switch. It is everything OUTSIDE the engine that holds a second copy of the
// same fact.
//
// Three of them, and each one is a bug if it is left out:
//
//   - conv.stance, or the next re-bootstrap (a model switch, a desk change)
//     reads the old value off the conversation and quietly widens the session
//     back without anybody asking for it.
//   - the sessions row, or reopening the chat comes back in a stance it was not
//     left in.
//   - the picker, or the composer goes on drawing ลงมือ over an engine that has
//     stopped handing out `write`. StartPlanRun already learned this one the
//     hard way (goal_run.go): the run worked and the screen lied about which
//     mode it was in.
//
// Keyed on the conversation the engine belongs to and NOT on a.cur(), which is
// the difference between this and SetStance. The user's press is aimed at the
// chat in front of them; this arrives from a turn that may be running in a chat
// they have since navigated away from, and writing a.cur() there would move the
// dial on somebody else's conversation.
//
// No re-bootstrap, deliberately. SetStance rebuilds the engine because the user
// can press the picker between turns and the prompt should be rebuilt with it;
// this fires INSIDE a turn, where replacing the agent would drop the round in
// flight. The prompt therefore stays as it was booted, and the stance's own
// direction reaches the model as the tool's result instead — see the tool.
func (a *App) stanceNarrowedByAgent(conv *conversation, next mode.Stance) {
	if conv == nil {
		return
	}
	// The two writes together, under the lock endTurn reads them both through.
	//
	// The stance is what the next bootstrap builds from; the flag is what makes
	// that bootstrap happen, at the boundary, because the engine keeps the
	// prompt it booted with for the rest of this turn and must not keep it any
	// longer (conversation.stanceRebuild). Split, endTurn could see the flag
	// without the stance and rebuild the session back into ลงมือ.
	//
	// This is the first write to conv.stance that does NOT come from the window
	// goroutine — SetStance is a click — so the lock is new here rather than a
	// habit copied. What still reads it without one is Stance(), which draws the
	// picker: a beat-late answer there costs nothing, because the event three
	// lines down is what actually moves the chip.
	a.turnMu.Lock()
	conv.stance = next
	conv.stanceRebuild = true
	a.turnMu.Unlock()
	if conv.id != "" {
		if db, err := a.database(); err == nil {
			_, _ = db.Exec(`UPDATE sessions SET stance = ? WHERE id = ?`, next.String(), conv.id)
		}
	}
	// Stamped with the session, like every other per-chat event (§187): a window
	// showing a different conversation must ignore it rather than redraw its own
	// composer from somebody else's turn.
	a.emitEvent("stance:update", sessionEvent[string]{SessionID: conv.id, Data: next.String()})
}
