package engine

// The engine's lifecycle, as the window drives it (§248 B1).
//
// Package-level functions rather than exported methods, on purpose: every
// exported method of the screen's App is a Wails binding, and the desktop
// forwards the engine's exported methods to the frontend by generation. A
// startup hook or an asset middleware is not something the frontend calls,
// and an `http.Handler` has no shape on that wire — so these four stay off
// the method set the generator reads, and the window reaches them by name.

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/Mikedev115/Aetox/internal/config"
)

// Startup is the Wails OnStartup hook: the window's context, the remembered
// project and desk, the background sweeps.
func Startup(e *Engine, ctx context.Context) { e.startup(ctx) }

// BeforeClose is the Wails OnBeforeClose hook: every turn is stopped and
// written down before the window goes. Never prevents the close.
func BeforeClose(e *Engine, ctx context.Context) (prevent bool) { return e.beforeClose(ctx) }

// Shutdown is the Wails OnShutdown hook: the store, the MCP children, the
// language servers, a read-aloud in flight.
func Shutdown(e *Engine, ctx context.Context) { e.shutdown(ctx) }

// AssetMiddleware serves the two URL surfaces the panes load directly —
// /aetox-file/ for the open project's files and /aetox-tts/ for synthesized
// speech — in front of the frontend's own assets.
func AssetMiddleware(e *Engine, next http.Handler) http.Handler { return e.assetMiddleware(next) }

// WebviewUserDataDir returns where a WebView2 instance should store its
// profile (cache/cookies/IndexedDB) — always an explicit, Aetox-owned path
// under config.DataRoot() (ARCHITECTURE.md §14), never Wails'/go-webview2's
// own silent default (%AppData%\<exe-name>, which used to differ between the
// dev binary and the real one — two profiles for the same app). Empty return
// is only a last-resort fallback if DataRoot() itself fails.
func WebviewUserDataDir(name string) string {
	root, err := config.DataRoot()
	if err != nil || root == "" {
		return ""
	}
	return filepath.Join(root, "webview", name)
}

func webviewUserDataDir(name string) string { return WebviewUserDataDir(name) }
