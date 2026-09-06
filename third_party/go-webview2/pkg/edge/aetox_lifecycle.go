//go:build windows

package edge

// AETOX PATCH: the two lifecycle calls upstream declares and never binds.
//
// Same shape as the capture and DevTools patches: `corewebview2.go` and
// `ICoreWebView2Controller.go` carry the vtbl slots (`Reload`, `Close`) with no
// method on them. Aetox needs both for one reason — a tab whose engine has gone
// (desktop/browser.go, engineGone):
//
//   - Close lets a dead or discarded tab release its controller instead of
//     leaving it to be reclaimed at process exit. Before this, destroy() was
//     DestroyWindow alone, and the "ponytail" comment on it said as much.
//   - Reload is the one-line recovery for a renderer crash, which WebView2
//     reports through ProcessFailed and then shows its own error page for.
//
// Both return the engine's HRESULT as an error rather than routing through
// errorCallback, because the caller that closes a tab already knows the engine
// may be gone and does not want that fact reported as a new complaint.

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Close closes the WebView and releases the browser instance behind it. After
// this, every other call on the controller and its CoreWebView2 fails with
// ERROR_INVALID_STATE — which is also what they return when the browser
// process has gone away on its own, and why desktop/browser_windows.go treats
// that code as "the engine is closed" rather than as a transient.
func (i *ICoreWebView2Controller) Close() error {
	hr, _, _ := i.vtbl.Close.Call(uintptr(unsafe.Pointer(i)))
	if windows.Handle(hr) != windows.S_OK {
		return windows.Errno(hr)
	}
	return nil
}

// Reload reloads the current page — the engine's own F5.
func (i *ICoreWebView2) Reload() error {
	hr, _, _ := i.vtbl.Reload.Call(uintptr(unsafe.Pointer(i)))
	if windows.Handle(hr) != windows.S_OK {
		return windows.Errno(hr)
	}
	return nil
}

// Close closes this Chromium's controller. Idempotent, and safe on one whose
// engine has already gone: the refusal comes back as the error and nowhere
// else. Marks the instance as shutting down first so a queued Eval that lands
// afterwards is dropped rather than reported.
func (e *Chromium) Close() error {
	if e.controller == nil {
		return nil
	}
	e.shuttingDown = true
	return e.controller.Close()
}

// Reload asks the engine to reload the page in place. Errors go to the error
// callback like Navigate's do: a reload that the engine refuses is the same
// kind of fact as a navigation it refuses.
func (e *Chromium) Reload() {
	if e.webview == nil || e.shuttingDown {
		return
	}
	if err := e.webview.Reload(); err != nil {
		e.errorCallback(err)
	}
}
