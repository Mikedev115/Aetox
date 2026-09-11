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
}

// NewApp builds the screen around a fresh engine.
func NewApp() *App {
	return &App{eng: engine.NewEngine()}
}

// The four Wails lifecycle hooks and the asset middleware, wired in main.go.
// Unexported, as they were: not bindings, not for the frontend.

func (a *App) startup(ctx context.Context) { engine.Startup(a.eng, ctx) }

func (a *App) beforeClose(ctx context.Context) (prevent bool) { return engine.BeforeClose(a.eng, ctx) }

func (a *App) shutdown(ctx context.Context) { engine.Shutdown(a.eng, ctx) }

func (a *App) assetMiddleware(next http.Handler) http.Handler {
	return engine.AssetMiddleware(a.eng, next)
}
