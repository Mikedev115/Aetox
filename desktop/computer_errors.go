package main

// The vocabulary of failure, and why this file exists at all.
//
// The tool this work replaces had exactly two error sentences. `computerScreenInfo`
// collapsed everything into "GetCursorPos failed: %v" and `sendInputBatch` into
// "SendInput delivered 0 of 4 events". Both are true and neither is actionable:
// they name the Win32 call that returned zero and say nothing about which of the
// four different situations produced it. The owner hit one of them live, the
// transcript could not distinguish them, and the commit that removed the tool
// says so in as many words — "deleted rather than debugged" (50584e5).
//
// So this is not error formatting. It is the difference between a tool that can
// be repaired from a transcript and one that can only be deleted. Every reach
// failure resolves to one of the sentences below, each of which tells the model
// (and through it the user) the one thing that changes what to do next:
//
//	the element moved      → read again, refs renumber
//	the control is off     → nothing to do here yet, look for what enables it
//	the control can't      → this control has no such verb; try another route
//	privilege              → Aetox is below that window; a human has to do it
//	no input desktop       → the screen is locked or a security prompt is up
//	the app stopped answering → the program is busy or hung, not absent
//
// Cross-platform on purpose, even though the only reach today is Windows: the
// classifier is a table, and a table that compiles everywhere can be tested
// everywhere. The Windows-only half is the mapping input in uia_windows.go.

import (
	"errors"
	"fmt"
	"strings"
)

// hresult carries a COM failure with its numeric code intact so the classifier
// below has something to reason about. Lives here rather than beside the COM
// code because the code is the input to a decision made here.
type hresult struct {
	code uint32
	what string
}

func (h hresult) Error() string { return fmt.Sprintf("%s: HRESULT %#08x", h.what, h.code) }

// Well-known HRESULTs this code reasons about by name rather than by number.
const (
	hrElementNotAvailable = 0x80040201 // UIA_E_ELEMENTNOTAVAILABLE
	hrElementNotEnabled   = 0x80040200 // UIA_E_ELEMENTNOTENABLED
	hrNoClickablePoint    = 0x80040202 // UIA_E_NOCLICKABLEPOINT
	hrNotSupported        = 0x80040204 // UIA_E_NOTSUPPORTED
	hrTimeout             = 0x80131505 // UIA_E_TIMEOUT
	hrAccessDenied        = 0x80070005 // E_ACCESSDENIED
	hrInvalidArg          = 0x80070057 // E_INVALIDARG
	hrNoInterface         = 0x80004002 // E_NOINTERFACE
	hrRPCUnavailable      = 0x800706ba // RPC_S_SERVER_UNAVAILABLE
	hrRPCCallFailed       = 0x800706be // RPC_S_CALL_FAILED
	hrCallRejected        = 0x80010001 // RPC_E_CALL_REJECTED
	hrServerNotResponding = 0x8001011f // RPC_E_SERVERCALL_RETRYLATER's neighbour
	hrClassNotRegistered  = 0x80040154 // REGDB_E_CLASSNOTREG
)

// Win32 last-error values the reach can produce.
const (
	winAccessDenied         = 5
	winInvalidWindowHandle  = 1400
	winInvalidThreadID      = 1444
	winNotEnoughQuota       = 1816
	winAccessDeniedByPolicy = 1260
)

// reachRefusal is a NO that the guard decided, not one Windows returned. It is
// a separate type because the two answer different questions and must never be
// worded alike: Windows refusing is a fact about the machine, and the guard
// refusing is a rule this project chose, which means it can name the rule and
// the way around it.
type reachRefusal struct {
	why  string
	hint string
}

func (r reachRefusal) Error() string {
	if r.hint == "" {
		return r.why
	}
	return r.why + " " + r.hint
}

func refuse(why, hint string) error { return reachRefusal{why: why, hint: hint} }

// win32Error is a failure a plain Win32 call reported, carrying the last-error
// value so the classifier can separate "no input desktop" from "wrong window".
type win32Error struct {
	call string
	code uintptr
}

func (w win32Error) Error() string {
	return fmt.Sprintf("%s failed (Windows error %d)", w.call, w.code)
}

// explainReach turns any error the reach can produce into one sentence a model
// can act on. Never returns the empty string, and never returns a bare "failed":
// the fallback still names the operation and the raw error, because an
// unclassified failure that says which call produced it is one bug report away
// from becoming a classified one.
func explainReach(op string, err error) string {
	if err == nil {
		return ""
	}

	var refusal reachRefusal
	if errors.As(err, &refusal) {
		return refusal.Error()
	}

	var hr hresult
	if errors.As(err, &hr) {
		if s := explainHRESULT(hr.code); s != "" {
			return s
		}
		return fmt.Sprintf("%s did not go through: Windows returned %#08x (%s). "+
			"This one is not yet classified — report it with what was on screen.",
			op, hr.code, hr.what)
	}

	var w32 win32Error
	if errors.As(err, &w32) {
		if s := explainWin32(w32.code); s != "" {
			return s
		}
		return fmt.Sprintf("%s did not go through: %s.", op, w32.Error())
	}

	return fmt.Sprintf("%s did not go through: %v", op, err)
}

func explainHRESULT(code uint32) string {
	switch code {
	case hrElementNotAvailable:
		return "That element is no longer in the window. The app has redrawn since the last read, " +
			"so read the window again — refs are renumbered by every read and the one you used belongs to the round before."
	case hrElementNotEnabled:
		return "That control is there but disabled right now. Nothing was clicked. " +
			"Read the window again and look for what has to happen first — an empty required field, an unchecked box, a step not finished."
	case hrNotSupported, hrNoInterface:
		return "That control does not accept this action. A label cannot be typed into and a static image cannot be pressed. " +
			"Read the window again and aim at the control that carries the verb you want."
	case hrNoClickablePoint:
		return "That element has no point that can be pressed — it is scrolled out of view or covered by another window. " +
			"Bring the window forward or scroll it into view first."
	case hrAccessDenied:
		return "Windows refused. That window belongs to a program running with higher privileges than Aetox, " +
			"and a program cannot drive one above it. Do this step by hand, or restart Aetox with the same privileges as that program."
	case hrTimeout, hrCallRejected, hrServerNotResponding:
		return "The app did not answer in time. It is busy or has stopped responding, not missing. " +
			"Wait for it to come back and read the window again."
	case hrRPCUnavailable, hrRPCCallFailed:
		return "The app closed while this was in flight. Nothing was done. List the windows again to see what is still open."
	case hrClassNotRegistered:
		return "Windows UI Automation is not available on this machine. This is an OS component, " +
			"so it usually means a stripped or heavily locked-down Windows install rather than a missing download."
	case hrInvalidArg:
		return "Windows rejected the request as malformed. This is a defect in Aetox rather than something on screen — report it."
	}
	return ""
}

func explainWin32(code uintptr) string {
	switch code {
	case winAccessDenied:
		// The single most important sentence in this file. This is the failure
		// the removed tool reported as "GetCursorPos failed", and the reason it
		// could not be told apart from a privilege problem.
		return "There is no reachable desktop to act on right now. The screen is locked, a Windows security prompt is up, " +
			"or the session is switched away. Nothing was clicked or typed. Unlock the screen and try again."
	case winInvalidWindowHandle:
		return "That window has closed since it was listed. List the windows again."
	case winInvalidThreadID:
		return "The window's program has exited. List the windows again."
	case winNotEnoughQuota:
		return "Windows would not accept that much input at once. Send it in smaller pieces."
	case winAccessDeniedByPolicy:
		return "A Windows policy on this machine blocks this. An administrator has restricted it; it is not something Aetox can work around."
	}
	return ""
}

// unclassified reports whether explainReach fell through to its raw fallback.
// The test uses it to hold the line that a new failure path must arrive with a
// sentence, and the tool uses it to mark a receipt worth reporting.
func unclassified(sentence string) bool {
	return strings.Contains(sentence, "not yet classified") ||
		strings.Contains(sentence, "did not go through: ")
}
