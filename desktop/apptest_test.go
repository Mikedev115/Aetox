package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/engine/remote"
	"github.com/Mikedev115/Aetox/internal/engine/rpc"
)

const testToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// newTestApp is the screen a test holds: the same client NewApp builds, on
// a wire to an engine in this process behind a loopback listener — never an
// in-process engine value, because there is no such path any more (§248
// decision 2). No window: a.ctx stays nil, so anything that would reach the
// Wails runtime reaches the test seams (emit, openDir) or nothing. The
// engine is shut down with the test, so the store it opened is closed
// before the temp dirs go.
func newTestApp(t *testing.T) *App {
	t.Helper()
	a, _ := newTestAppAt(t)
	return a
}

// newTestAppAt is newTestApp with the listener's address, for a test that
// wants the same engine to look like one on a host (onAHost).
func newTestAppAt(t *testing.T) (*App, string) {
	t.Helper()
	a := &App{}
	a.client = a.newClient()
	a.api = a.client
	srv := rpc.NewServer(testToken, engine.NewEngine)
	hs := httptest.NewServer(srv.Handler())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.client.Connect(ctx, "tcp", strings.TrimPrefix(hs.URL, "http://"), testToken); err != nil {
		hs.Close()
		t.Fatalf("connecting the test screen: %v", err)
	}
	if _, err := a.client.Hello(ctx, "test", (appScreen{a}).WindowTools(nil), []string{rpc.FeatureWindowTools, rpc.FeatureProviderProxy}); err != nil {
		hs.Close()
		t.Fatalf("hello: %v", err)
	}
	t.Cleanup(func() {
		a.client.Close()
		hs.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		engine.Shutdown(srv.Engine(), ctx)
	})
	return a, strings.TrimPrefix(hs.URL, "http://")
}

// onAHost makes the test's engine look like one on a host over ssh: the
// supervisor in remote mode, its wire the loopback listener the engine is
// really behind. Nothing else changes — which is the point: the doors must
// tell the two apart by this alone.
func onAHost(a *App, addr string) {
	a.engine = &localEngine{
		token:  testToken,
		target: engineTarget{mode: modeRemote, host: remote.Host{Name: "wsl", Target: "wsl"}},
		proc:   &engineProcess{network: "tcp", address: addr, remote: "wsl"},
		status: EngineStatus{State: engineConnected, Mode: modeRemote, Host: "wsl"},
	}
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
	// voice stands in for the engine's voice preferences, when set.
	voice *engine.VoiceSettings
}

func (e engineWith) VoiceSettings() engine.VoiceSettings {
	if e.voice != nil {
		return *e.voice
	}
	return e.API.VoiceSettings()
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
