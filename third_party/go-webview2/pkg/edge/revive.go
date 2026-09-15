//go:build windows

package edge

// AETOX PATCH: bringing a window's browser back after its process died.
//
// WebView2 runs the page in processes of its own, and the one that matters —
// the browser process — can end for reasons that have nothing to do with the
// app: out of memory, a GPU driver reset, an antivirus quarantining the
// runtime mid-run, Task Manager. The engine then raises ProcessFailed with
// COREWEBVIEW2_PROCESS_FAILED_KIND_BROWSER_PROCESS_EXITED, the controller and
// webview behind the window are dead for good, and Microsoft's guidance is
// to build a new one (docs: "the app has to recreate a new WebView to recover
// from this failure"). Upstream Wails answers the event with a message box
// and os.Exit(-1) on the whole process — every turn in flight, every unsaved
// state, gone with a dialog the user did not ask for.
//
// This file is the other answer. A host that wants the window to live installs
// BrowserProcessExitedHook; when the event names the browser process on a
// Chromium whose owner never installed an error callback (the Wails main
// window — a tab that did owns its own recovery, browser_windows.go) the hook
// is asked first, on the UI thread but after the event handler has returned,
// and a hook that took the event keeps it from the owner's callback.
//
// Revive is what the hook calls: the same Embed that built the view at startup,
// on the same window, followed by everything this package recorded being put on
// the old view — resource filters and the last navigation. Settings the owner
// put on the old webview's ICoreWebViewSettings lived on the object that died;
// putting them again is the hook's job, since only the owner knows them.

import (
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/wailsapp/go-webview2/internal/w32"
)

// BrowserProcessExitedHook is asked when a browser process died behind a
// Chromium with no error callback of its own. It runs on the view's UI thread,
// after the event handler returned. true means the hook took the event and the
// owner's ProcessFailedCallback is not told.
var BrowserProcessExitedHook func(e *Chromium) bool

type resourceFilter struct {
	filter string
	ctx    COREWEBVIEW2_WEB_RESOURCE_CONTEXT
}

// offerRevival is ProcessFailed's first stop. It answers true when the hook
// will be asked, which is decided here and now; the asking itself is deferred
// past the handler's return, as the WebView2 samples do, so the new view is
// not built inside the old one's event dispatch.
func (e *Chromium) offerRevival(args *ICoreWebView2ProcessFailedEventArgs) bool {
	if BrowserProcessExitedHook == nil || e.customErrorCallback || args == nil || e.hwnd == 0 {
		return false
	}
	kind, err := args.GetProcessFailedKind()
	if err != nil || kind != COREWEBVIEW2_PROCESS_FAILED_KIND_BROWSER_PROCESS_EXITED {
		return false
	}
	// The dead proxies stay where they are until Embed replaces them. A call
	// on one meanwhile — Wails resizing, focusing, running a script — comes
	// back as an HRESULT and goes through errorCallback, which no longer exits
	// once the view has been up; a nil there would be a panic in Focus.
	runLater(e.hwnd, func() {
		if !BrowserProcessExitedHook(e) && e.ProcessFailedCallback != nil {
			e.ProcessFailedCallback(nil, args)
		}
	})
	return true
}

// Revive builds a new browser behind the same window. false when the embed
// failed, in which case the window is a dead view and the caller decides.
func (e *Chromium) Revive() bool {
	if e.hwnd == 0 {
		return false
	}
	// The old proxies are not released: the process that served them is
	// gone, and a Release on one is a call into nothing. Embed overwrites
	// them when the new controller completes.
	atomic.StoreUintptr(&e.inited, 0)
	atomic.StoreUintptr(&e.embedFailed, 0)
	if !e.Embed(e.hwnd) {
		return false
	}
	// Every filter the owner added, added again — re-recorded by the call, so
	// the list is taken first and cleared.
	filters := e.resourceFilters
	e.resourceFilters = nil
	for _, f := range filters {
		e.AddWebResourceRequestedFilter(f.filter, f.ctx)
	}
	e.Resize()
	// The same hide/show Wails does on first navigation
	// (WebView2Feedback#1077): a fresh controller does not always paint.
	_ = e.Hide()
	_ = e.Show()
	if e.lastNavigate != "" {
		e.Navigate(e.lastNavigate)
	}
	return true
}

// runLater runs fn on hwnd's thread once the current message has been
// dispatched. A timer with its own procedure is dispatched by DispatchMessage
// without the window's procedure being involved, which is what makes this
// possible from a package that does not own the window.
//
// One callback for the life of the process: syscall.NewCallback takes a slot
// from a table of 2000 the runtime never frees (Aetox DECISIONS §295), so the
// procedure is registered once and finds its closure by timer id.
func runLater(hwnd uintptr, fn func()) {
	laterOnce.Do(func() { laterProc = syscall.NewCallback(laterTimerProc) })
	laterMu.Lock()
	laterSeq++
	id := laterSeq
	laterFns[id] = fn
	laterMu.Unlock()
	r, _, _ := w32.User32SetTimer.Call(hwnd, id, 0, laterProc)
	if r == 0 {
		laterMu.Lock()
		delete(laterFns, id)
		laterMu.Unlock()
		// No timer to wait for: the thread is this one, so run it here.
		fn()
	}
}

var (
	laterOnce sync.Once
	laterProc uintptr
	laterMu   sync.Mutex
	laterSeq  uintptr
	laterFns  = map[uintptr]func(){}
)

// laterTimerProc is TIMERPROC: (HWND, UINT message, UINT_PTR idEvent, DWORD time).
func laterTimerProc(hwnd uintptr, _ uint32, id uintptr, _ uint32) uintptr {
	_, _, _ = w32.User32KillTimer.Call(hwnd, id)
	laterMu.Lock()
	fn := laterFns[id]
	delete(laterFns, id)
	laterMu.Unlock()
	if fn != nil {
		fn()
	}
	return 0
}
