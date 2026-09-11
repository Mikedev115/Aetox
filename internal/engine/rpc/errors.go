package rpc

import (
	"context"
	"errors"
	"sync"
)

// Kinds is the registry of errors that keep their identity across the wire.
//
// A Go error is a value with an identity; on the wire it is a sentence. Most
// callers only ever read the sentence — the frontend matches "file-gone" in
// err.message and nothing else — but the screen's own Go code sometimes asks
// errors.Is, and the engine's sentinel is on the other side of a socket. So a
// sentinel registered here travels as a kind in the error's data, and the
// *Error the caller receives answers errors.Is for it. Registration is
// explicit and short: a kind is a contract the frontend can also read.
type kinds struct {
	mu   sync.RWMutex
	byID map[string]error
	ids  []registered
}

type registered struct {
	kind string
	err  error
}

var Kinds = &kinds{byID: map[string]error{}}

// Register names a sentinel. Later registrations of the same kind replace
// the earlier one, so a test can register a stand-in.
func (k *kinds) Register(kind string, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.byID[kind] = err
	for i := range k.ids {
		if k.ids[i].kind == kind {
			k.ids[i].err = err
			return
		}
	}
	k.ids = append(k.ids, registered{kind, err})
}

// kindOf answers with the registered kind an error is, or "".
func (k *kinds) kindOf(err error) string {
	if err == nil {
		return ""
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, r := range k.ids {
		if errors.Is(err, r.err) {
			return r.kind
		}
	}
	return ""
}

func (k *kinds) sentinel(kind string) error {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.byID[kind]
}

// The kinds every wire knows. Context errors first: a call the caller
// cancelled, or one the engine gave up on, has to come back as the same
// error it would have been in-process, or every `errors.Is(err,
// context.Canceled)` on the screen silently stops matching.
func init() {
	Kinds.Register("canceled", context.Canceled)
	Kinds.Register("deadline", context.DeadlineExceeded)
	Kinds.Register("no_screen", ErrNoScreen)
	Kinds.Register("disconnected", ErrDisconnected)
	Kinds.Register("method_not_found", ErrMethodNotFound)
}

// ErrNoScreen is the engine with nobody at the other end of the wire: a
// provider request that waited its whole budget for a screen to sign it, a
// window tool called with no window.
var ErrNoScreen = errors.New("no screen is connected")

// ErrDisconnected is a call that could not complete because the connection
// went away under it. Distinct from canceled: nobody chose this.
var ErrDisconnected = errors.New("the engine connection was lost")

// ErrMethodNotFound is a call for a method the other side does not have — a
// screen and an engine of different versions, or a typo in a generated file
// that the currency test would have caught.
var ErrMethodNotFound = errors.New("no such method")

// Is lets errors.Is(err, sentinel) hold across the wire for registered
// kinds, and for the *Error's own identity.
func (e *Error) Is(target error) bool {
	if e == nil {
		return false
	}
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code && e.Message == t.Message
	}
	kind := e.Kind()
	if kind == "" {
		return false
	}
	if s := Kinds.sentinel(kind); s != nil {
		return errors.Is(s, target)
	}
	return false
}

// toWire turns the error an engine method returned into what crosses the
// wire: the sentence, and the kind if it has one.
func toWire(err error) *Error {
	if err == nil {
		return nil
	}
	var wire *Error
	if errors.As(err, &wire) {
		// Already a wire error (a call forwarded through a second hop, or a
		// test) — pass it on as it is.
		return wire
	}
	out := &Error{Code: codeApp, Message: err.Error()}
	if kind := Kinds.kindOf(err); kind != "" {
		out.Data = &errorData{Kind: kind}
	}
	return out
}
