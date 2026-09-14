package model

import (
	"errors"
	"time"
)

// RateLimitError is a refusal that named the moment it lifts.
//
// A 429 has always ended the turn with a sentence — "the plus plan's limit is
// used up. It resets in 3 hours." — and left the next move to the user: come
// back later, press retry, and hope the tool loop that was eighteen results
// deep can be rebuilt. The owner runs Codex on a plan with a five-hour window
// and asked for the obvious thing instead (14 ก.ย. 2026): when the window is
// spent, say so, wait for it, and carry on where the turn was.
//
// That decision is not the provider's to make. Whether a turn may be held open
// for three hours depends on who is watching it — a window that can draw a
// countdown, or a delegate nobody is looking at — so the provider only states
// the fact, as this type, and cognitive.Agent decides what to do with it. The
// transport's own retry (retryTransport.maxWait) stays short and is unrelated:
// that one covers "slow down for a few seconds", this one covers "the plan is
// spent until four o'clock".
//
// Err is the sentence the turn would have ended with, kept whole so a caller
// that chooses not to wait shows exactly what it always has.
type RateLimitError struct {
	Provider string
	ResetAt  time.Time
	Err      error
}

func (e *RateLimitError) Error() string { return e.Err.Error() }
func (e *RateLimitError) Unwrap() error { return e.Err }

// RateLimitResetAt reports when the limit behind err lifts. False when err is
// not a rate limit, or is one that never said when — there is nothing to
// count down to, and the caller's only honest move is the sentence.
func RateLimitResetAt(err error) (time.Time, bool) {
	var limit *RateLimitError
	if !errors.As(err, &limit) || limit.ResetAt.IsZero() {
		return time.Time{}, false
	}
	return limit.ResetAt, true
}
