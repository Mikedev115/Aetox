//go:build windows

package main

import (
	"sync/atomic"
	"time"

	"github.com/wailsapp/go-webview2/pkg/edge"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// The main window's browser died; the app does not.
//
// WebView2 keeps the page in processes of its own, and the browser process
// among them can end for reasons that are nobody's fault in particular — the
// machine ran out of memory, a GPU driver reset, an antivirus took the
// runtime away mid-run, somebody ended it in Task Manager. Wails' answer to
// that event was a message box and os.Exit(-1) on the whole process
// (internal/frontend/desktop/windows/frontend.go, ProcessFailedCallback): the
// engine beside this window, holding every turn in flight, was taken down
// with it, and the log simply stopped. The owner's words, 16 ก.ย. 2026:
// "เว็บวิวบางทีมันเออเร่อ มันทำให้โปรแกรมปิดตัวลงเลย".
//
// The browser TABS already survived their own engine dying (browser_windows.go,
// ProcessFailedCallback: a dead engine is a tab to revive). This is the same
// answer for the window itself, through the fork's revival hook
// (third_party/go-webview2/pkg/edge/revive.go): the view is built again on the
// same window, the settings main.go asked Wails for are put back, and the page
// loads fresh — a reload, which the Go side has always outlived (SessionTranscript,
// TurnInFlight). The engine never learns anything happened.
//
// When the rebuild itself fails the hook says so, and Wails gets the event
// after all: a window with no view in it is worse than the dialog.

// installWebviewRevival hands the fork the hook. Before wails.Run, because
// the view is built inside it.
func installWebviewRevival(app *App) {
	edge.BrowserProcessExitedHook = app.reviveMainWebview
}

// reviveMainWebview is the hook: on the UI thread, after the event that
// reported the death has been dispatched.
func (a *App) reviveMainWebview(view *edge.Chromium) bool {
	n := atomic.AddInt64(&a.webviewRevivals, 1)
	debuglog.Msg("webview: the main window's browser process exited (revival %d); rebuilding the view", n)
	started := time.Now()
	if !view.Revive() {
		debuglog.Msg("webview: revival FAILED after %s; leaving the event to Wails", time.Since(started).Round(time.Millisecond))
		return false
	}
	// What setupChromium put on the old webview's settings object, put on the
	// new one. The values are the ones main.go's options resolve to inside
	// Wails: context menus and devtools only in a dev build, no zoom control,
	// pinch zoom on, no status bar, no browser accelerator keys (F5, Ctrl+F —
	// the app's own shortcuts own those).
	dev := a.ctx != nil && wailsruntime.Environment(a.ctx).BuildType == "dev"
	if settings, err := view.GetSettings(); err == nil {
		_ = settings.PutAreDefaultContextMenusEnabled(dev)
		_ = settings.PutAreDevToolsEnabled(dev)
		_ = settings.PutIsZoomControlEnabled(false)
		_ = settings.PutIsPinchZoomEnabled(true)
		_ = settings.PutIsStatusBarEnabled(false)
		_ = settings.PutAreBrowserAcceleratorKeysEnabled(false)
	} else {
		debuglog.Msg("webview: revived, but its settings could not be read: %v", err)
	}
	// main.go's BackgroundColour, so the moment before the page paints is the
	// app's dark ground and not a white flash.
	view.SetBackgroundColour(11, 15, 22, 255)
	debuglog.Msg("webview: revived in %s", time.Since(started).Round(time.Millisecond))
	return true
}
