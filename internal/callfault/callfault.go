// Package callfault marks errors whose whole remedy is in the call that
// caused them: a required argument was not given, an action word is not one
// the tool has, a tool was named that this seat does not hold.
//
// It is the third mark of its kind, and the one the first two left implicit.
// *exec.ExitError says a program ran and reported a result; statereport says
// the machine was in some state tonight. What stayed unmarked was "the tool
// refused this call and said why" — and for a year that was the conservative
// default, because the summarizer's reader treated every unmarked refusal as
// a sentence with a remedy worth offering. Since 2026-08-18 that reader offers
// nothing to memory; it raises problems for a developer. A refusal the model
// read and got past is not one of those. Owner, 11 ก.ย., looking at a card
// reporting `search` had failed four times with "action is required":
// *"ไม่ควรให้มันแจ้งเตือนอย่างเดียว แต่แจ้งเตือนอันที่เป็นปัญหาจริง ๆ"*. All
// twenty cards that page had ever raised were waved off, and ten of them were
// this shape — the model forgot a word, the tool said which word, the model
// supplied it on the next call.
//
// The line an author draws: mark the error when a different call from the
// same caller, with the same tool and the same machine, would have succeeded.
// A missing `action`, an `about` not given, `remove` without `old`, a tool
// name nothing answers to. Leave it unmarked when the call was well-formed and
// the tool still could not do the work — that is either the world
// (statereport) or the tool itself, and the second of those is exactly what
// the problems page exists to show.
//
// Same shape and same position in the import graph as statereport, for the
// same reason: the authors live in leaf packages the turn executor cannot be
// imported from.
package callfault

import (
	"errors"
	"fmt"
)

type faultErr struct{ err error }

func (e *faultErr) Error() string { return e.err.Error() }
func (e *faultErr) Unwrap() error { return e.err }

// New is errors.New for a refusal of the call itself.
func New(text string) error { return &faultErr{errors.New(text)} }

// Newf is fmt.Errorf for one. %w works as it does there.
func Newf(format string, a ...any) error { return &faultErr{fmt.Errorf(format, a...)} }

// Mark wraps an existing error as a caller fault; nil stays nil.
func Mark(err error) error {
	if err == nil {
		return nil
	}
	return &faultErr{err}
}

// Is reports whether err carries the mark anywhere in its chain.
func Is(err error) bool {
	var f *faultErr
	return errors.As(err, &f)
}
