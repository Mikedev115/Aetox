package main

import (
	"sync"
	"testing"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// newTestApp is the screen a test holds: built the way NewApp builds it, with
// the engine talking back to this very App, and no window — a.ctx stays nil,
// so anything that would reach the Wails runtime reaches the test seams
// (emit, openDir) or nothing.
func newTestApp() *App {
	a := &App{}
	a.eng = engine.NewEngine(appScreen{a})
	a.api = a.eng
	return a
}

// engineWith is the real engine with one or two answers changed: what a
// screen test states as a fact about the engine — a turn is running, the
// marks are off — without reaching into a package it cannot see into.
// Everything not named here is answered by the engine newTestApp built.
type engineWith struct {
	engine.API
	turnRunning bool
	marksOff    bool
	// export stands in for the deck export's engine half, when set.
	export func(relPath, format string) (engine.DeckExport, error)
}

func (e engineWith) AnyTurnRunning() bool { return e.turnRunning }
func (e engineWith) PageMarksOn() bool    { return !e.marksOff }
func (e engineWith) DeckExportFiles(relPath, format string) (engine.DeckExport, error) {
	if e.export != nil {
		return e.export(relPath, format)
	}
	return e.API.DeckExportFiles(relPath, format)
}

// emitted is one event the recorder saw; recorder is the locked list of them.
// The screen's own copy of the engine tests' recorder, for the same reason:
// the update announces from goroutines, and the browser host drains on its
// own thread.
type emitted struct {
	Name string
	Data []any
}

type recorder struct {
	mu   sync.Mutex
	seen []emitted
}

func (r *recorder) add(e emitted) { r.mu.Lock(); r.seen = append(r.seen, e); r.mu.Unlock() }

func (r *recorder) len() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.seen) }

// all hands back a copy, so a caller ranging over it is not reading a slice a
// background emit is appending to.
func (r *recorder) all() []emitted {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]emitted(nil), r.seen...)
}

func (r *recorder) names() []string {
	all := r.all()
	out := make([]string, 0, len(all))
	for _, e := range all {
		out = append(out, e.Name)
	}
	return out
}

// captureEvents swaps the App's event seam for a recorder. It does not set
// a.ctx: some of these tests assert on an emit that must not happen, and
// giving the App a context would change what the code under test decides to
// fire.
func captureEvents(a *App) *recorder {
	rec := &recorder{}
	a.emit = func(event string, data ...any) { rec.add(emitted{Name: event, Data: data}) }
	return rec
}

// captureEmit is captureEvents for the tests that read names only.
func captureEmit(t *testing.T, a *App) *recorder {
	t.Helper()
	return captureEvents(a)
}
