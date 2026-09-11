// Package rpc is the wire between the screen and the engine (§248 phase 2).
//
// One WebSocket per screen↔engine pair, JSON-RPC 2.0 in both directions: the
// screen calls the engine's bindings (method = the Go name, params positional,
// the way the frontend already calls them), the engine sends the screen its
// events as notifications and calls back for the few things only a window
// can do — sign a provider request, run a browser action, render a deck. In
// tests the socket is an in-memory listener; in local mode it is a unix
// socket under DataRoot; over ssh it is the same socket at the far end of a
// tunnel. The framing is the same in all three, which is the whole point.
//
// Design doc §4 is the record of the message shapes; what is written here is
// what the code does.
package rpc

import (
	"encoding/json"
	"fmt"
)

// message is one JSON-RPC 2.0 object, one per text frame. A request has an
// id and a method; a notification a method and no id; a response an id and
// a result or an error. Ids are strings with the sender's side as prefix —
// "s12" from the screen, "e7" from the engine — so the two id spaces cannot
// collide on a wire that carries requests both ways.
type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *string         `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

func (m *message) isRequest() bool      { return m.Method != "" && m.ID != nil }
func (m *message) isNotification() bool { return m.Method != "" && m.ID == nil }
func (m *message) isResponse() bool     { return m.Method == "" && m.ID != nil }

// The JSON-RPC error codes this wire uses. Everything an engine method
// returns as a Go error is codeApp, with the error's own words as the message
// and its kind, when it has one, in the data (errors.go).
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternal       = -32603
	codeApp            = -32000
)

// Error is what crosses the wire when a call fails, and what the caller
// gets back. Message is the Go error's text verbatim: the frontend reads
// `err.message` today and matches sentences like "file-gone" in it, and a
// wire that paraphrased would break every one of those.
type Error struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    *errorData `json:"data,omitempty"`
}

type errorData struct {
	// Kind names a sentinel the Go side can still test with errors.Is —
	// see Kinds in errors.go.
	Kind string `json:"kind,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Kind is the sentinel's name, "" when the error was not one of the
// registered kinds.
func (e *Error) Kind() string {
	if e == nil || e.Data == nil {
		return ""
	}
	return e.Data.Kind
}

// nullResult is the result of a call that answers nothing: present, so the
// frame is a response and not mistaken for a notification.
var nullResult = json.RawMessage("null")

// encodeParams turns positional arguments into the params array. nil is sent
// as an empty array rather than omitted, so a method with no arguments and a
// method whose arguments failed to encode never look the same on the wire.
func encodeParams(params any) (json.RawMessage, error) {
	if params == nil {
		return json.RawMessage("[]"), nil
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("rpc: encoding params: %w", err)
	}
	return raw, nil
}
