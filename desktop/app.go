package main

// The screen (§248 B1).
//
// App is what Wails binds and what the frontend calls, and since Stage B it
// is a window and nothing else: every binding that is engine work forwards to
// the engine (engine_forwarders_gen.go, generated from the engine's exported
// methods), and every binding that needs a window is written here in
// desktop/. Today the engine is a value in this process; the field is what
// becomes a client of a socket in phase 2, and nothing above it has to change
// for that.

import (
	"context"
	"net/http"

	"github.com/Mikedev115/Aetox/internal/engine"
)

type App struct {
	eng *engine.Engine
	// ctx is the window's lifetime — what the Wails runtime is called with:
	// dialogs, window sizing, Quit. Nil until startup has run.
	ctx context.Context
	// openDir stands in for openInFileManager, the one door out to the OS file
	// manager, so a test can watch a reveal happen without a window appearing
	// on somebody's desk.
	openDir func(string) error
	// emit stands in for wailsruntime.EventsEmit — see emitEvent.
	emit func(event string, data ...any)

	staged stagedUpdate
}

// NewApp builds the screen and the engine that talks to it.
func NewApp() *App {
	a := &App{}
	a.eng = engine.NewEngine(appScreen{a})
	return a
}

// The four Wails lifecycle hooks and the asset middleware, wired in main.go.
// Unexported, as they were: not bindings, not for the frontend.

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Before anything else: the window is created and centred by the time
	// startup runs, and shown only once the webview has content, so a window
	// bigger than the screen is corrected while nobody can see it move.
	a.fitToScreen()
	engine.Startup(a.eng, ctx)
	// The previous build's exe, renamed aside by a self-update, and the staging
	// download — this build is running, so by definition neither is needed.
	// Except the download the user has not restarted into yet; see
	// adoptStagedUpdate.
	go a.adoptStagedUpdate()
	// And the other end of the same feature: ask whether a newer build exists,
	// so the answer reaches the user without them going looking for it
	// (update_notify.go).
	go a.watchForUpdates()
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) { return engine.BeforeClose(a.eng, ctx) }

func (a *App) shutdown(ctx context.Context) { engine.Shutdown(a.eng, ctx) }

func (a *App) assetMiddleware(next http.Handler) http.Handler {
	return engine.AssetMiddleware(a.eng, next)
}
