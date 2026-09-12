// Package cliagent is the seam for answering a chat with an agent program that
// already lives on the user's machine, instead of with Aetox's own turn loop.
//
// **It ships with no engines.** What is here is the contract and the registry:
// the shape an engine has to have, the events it reports a turn through, and
// the lookup the desktop makes before every turn. With nothing registered the
// lookup misses and Aetox behaves exactly as it did before this package
// existed — which is the point of building it early. The day an engine is
// worth having, it is one file that satisfies Engine plus one catalog row on
// RuntimeExternalCLI; nothing in the turn path, the timeline or the settings
// page has to be found and changed again.
//
// **Why a program and not an API.** Some agents are only reachable as the
// command the user already installed and signed into: the credential is the
// program's, the loop is the program's, and the vendor's terms may permit
// exactly that and nothing else. In that shape Aetox is the window — the
// bubble, the tool timeline, the Stop button, the transcript — and the program
// is the engine. The rule that comes with it, and the reason this contract has
// no credential in it anywhere: **an engine may read what the user set up for
// themselves and must never collect, store, forward or stand in for it.** An
// engine that wants a key gets it through Env, from the user's own store, for
// one child process.
//
// **What an engine owes the window.** Events as the work happens, not a
// transcript at the end: text as it streams, tool calls when they start and
// when they finish, and a Result that names what the turn came to. Everything
// below is the smallest set the chat window can draw — an engine that can say
// less simply reports less, and an engine that can say more says it through
// the tool events rather than by growing this struct.
package cliagent

import (
	"context"
	"crypto/rand"
	"fmt"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/proc"
)

// Engine is one external agent program Aetox can hand a turn to.
type Engine interface {
	// Provider is the canonical catalog id this engine answers for — the same
	// string the conversation's config carries, already normalized.
	Provider() string
	// Label is the engine's name for a settings row, in the vendor's own
	// spelling. Never a trademark Aetox has no right to use as its own.
	Label() string
	// Probe reports whether a turn could run right now: where the program is,
	// which version, and if it is not ready, the one sentence that says what
	// the user must do. Cheap enough to call from a settings page — no network,
	// no tokens — and safe to call concurrently.
	Probe(ctx context.Context) Status
	// Run performs one exchange. It reports progress through onEvent (which may
	// be nil, and is called from Run's own goroutine, in order) and returns
	// once the turn is over.
	//
	// Cancelling ctx must end the child and everything it started — this is the
	// Stop button — and Run then returns ctx.Err().
	Run(ctx context.Context, t Turn, onEvent func(Event)) (Result, error)
}

// Status is what Probe found.
type Status struct {
	// Path is the program Probe would run, "" when none was found.
	Path string `json:"path"`
	// Version is whatever the program calls itself, for a settings row.
	Version string `json:"version"`
	// Ready is the whole verdict: a turn handed over now would run.
	Ready bool `json:"ready"`
	// Detail is the sentence the user reads when Ready is false — what is
	// missing and what to do about it, in their language. It is what a failed
	// turn reports too, so the settings row and the chat cannot disagree.
	Detail string `json:"detail,omitempty"`
}

// Turn is one exchange's parameters.
type Turn struct {
	// Text is what the user said.
	Text string
	// Dir is the folder the engine's file tools should work in — the project
	// this chat is held in.
	Dir string
	// SessionID is the engine-side conversation id, and Resume says whether the
	// engine has seen it before. Aetox mints it (NewSessionID) and holds it on
	// the conversation; what the id MEANS is the engine's business, and so is
	// the history stored under it. This is the whole of what carries a chat
	// between turns: Aetox does not replay its own transcript into an engine
	// that is already keeping one.
	SessionID string
	Resume    bool
	// Model and Effort are the two dials the window offers, in whatever
	// spelling the engine's catalog row uses. Empty means the engine's default.
	Model  string
	Effort string
	// SystemNote is what Aetox adds to whatever system prompt the engine
	// already has: that the answer is being drawn in a desktop window with no
	// terminal behind it, and which desk the user opened. Never a replacement —
	// an engine's own prompt is what makes its own tools work.
	SystemNote string
	// Env is added to the child's environment as "KEY=value". It exists so a
	// credential the user typed into Aetox can reach a program that reads one
	// from the environment, for the length of one child process and no longer.
	Env []string
}

// EventKind is what one Event reports.
type EventKind string

const (
	// EventText is a fragment of the answer as it is written. The window draws
	// these into the live bubble and throws them away when the turn ends —
	// what is kept is EventAnswer.
	EventText EventKind = "text"
	// EventThinking is a fragment of the engine's reasoning, for the panel that
	// draws it. An engine that cannot show its reasoning sends none.
	EventThinking EventKind = "thinking"
	// EventAnswer is a finished block of prose, in the order the engine wrote
	// it. These are what the transcript keeps, so an engine that streams text
	// must also report the block it streamed.
	EventAnswer EventKind = "answer"
	// EventToolCall is a tool starting, EventToolResult the same tool ending.
	// Both carry Tool.Ref, which is what pairs them in the timeline.
	EventToolCall   EventKind = "tool-call"
	EventToolResult EventKind = "tool-result"
	// EventStatus is a line for the "what is it doing" indicator, "" to clear
	// it. Aetox's own words, not the engine's log.
	EventStatus EventKind = "status"
)

// Event is one thing that happened during a turn.
type Event struct {
	Kind EventKind
	// Text carries EventText / EventThinking / EventAnswer / EventStatus.
	Text string
	// Tool carries the two tool kinds.
	Tool Tool
	// Parent is the Ref of the tool call that caused this event, set only on
	// events from inside a delegate the engine spawned. Empty for the engine's
	// own work. The window uses it the same way it uses its own sub-agent
	// stamp: delegate events go under their row in the timeline and never into
	// the answer bubble.
	Parent string
}

// Tool is one tool call as the timeline draws it.
type Tool struct {
	// Ref is the engine's own id for the call, and it is what pairs a result
	// with its call. An engine that issues no ids must send none — the window
	// falls back to matching on the name.
	Ref string
	// Name is the tool's name in the engine's own vocabulary. It is shown as
	// it is: a made-up translation into Aetox's tool names would claim the
	// engine ran a tool Aetox has.
	Name string
	// Subject is the one argument worth reading in a list — the path, the
	// command, the pattern, the URL. Empty when the call takes nothing
	// nameable.
	Subject string
	// OK and Error are the result's verdict; both are ignored on a call event.
	OK    bool
	Error string
}

// Result is what a turn came to.
type Result struct {
	// Text is the engine's final answer. The window uses it when the engine
	// reported no EventAnswer at all, which is how a program that only prints
	// a result still works.
	Text string
	// SessionID is the id the engine actually used, which is the one to resume
	// next turn — an engine is free to hand back a different one from the one
	// it was given.
	SessionID string
	// Usage is what the turn cost in tokens, zero when the engine does not say.
	Usage Usage
	// CostUSD is the engine's own estimate, zero when it does not say. It is an
	// estimate: an engine reporting one is reporting what it believes, not a
	// bill.
	CostUSD  float64
	Duration time.Duration
	// IsError with ErrorText is the engine reporting that the RUN failed while
	// the program itself worked — a refused sign-in, a quota, a policy stop. A
	// program that could not be run at all is Run's returned error instead.
	IsError   bool
	ErrorText string
}

// Usage is a turn's token count.
type Usage struct {
	InputTokens int
	// CachedInputTokens is the part of InputTokens the engine says it served
	// from a prompt cache, and is counted inside it — never added to it.
	CachedInputTokens int
	OutputTokens      int
}

var registry struct {
	sync.RWMutex
	engines map[string]Engine
}

// Register adds an engine, keyed by its Provider(). Meant for an init in the
// engine's own file, so that adding one is adding a file.
//
// It panics on a duplicate or an empty provider, which are both mistakes a
// program makes about itself at startup rather than anything a user can cause.
func Register(e Engine) {
	if e == nil || e.Provider() == "" {
		panic("cliagent: engine with no provider")
	}
	registry.Lock()
	defer registry.Unlock()
	if registry.engines == nil {
		registry.engines = map[string]Engine{}
	}
	if _, taken := registry.engines[e.Provider()]; taken {
		panic("cliagent: two engines for provider " + e.Provider())
	}
	registry.engines[e.Provider()] = e
}

// For looks an engine up by canonical provider id. The caller normalizes —
// this package holds no catalog and cannot resolve an alias.
func For(provider string) (Engine, bool) {
	registry.RLock()
	defer registry.RUnlock()
	e, ok := registry.engines[provider]
	return e, ok
}

// UnregisterForTest takes an engine back out. Tests alone: the registry is
// written once at startup in real life, and nothing in the program has a
// reason to remove an engine — a provider row that stops being served would be
// a row that stops existing.
func UnregisterForTest(provider string) {
	registry.Lock()
	delete(registry.engines, provider)
	registry.Unlock()
}

// Providers lists what is registered, sorted. Empty is the normal state.
func Providers() []string {
	registry.RLock()
	defer registry.RUnlock()
	out := make([]string, 0, len(registry.engines))
	for name := range registry.engines {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Command builds the child process an engine runs, with the two things every
// such child on Windows needs: no console window of its own, and a kill that
// reaches the whole tree when the context ends. Engines should build their
// children through here rather than calling exec directly.
func Command(ctx context.Context, exe string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, exe, args...)
	proc.HideConsole(cmd)
	proc.KillOnCancel(cmd)
	return cmd
}

// NewSessionID mints the conversation id handed to an engine: a random UUID,
// which is the shape every agent program in reach asks for.
func NewSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// A time-seeded fallback rather than a panic: the id only has to be
		// unique on this machine.
		now := time.Now().UnixNano()
		for i := range b {
			b[i] = byte(now >> (8 * (i % 8)))
		}
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
