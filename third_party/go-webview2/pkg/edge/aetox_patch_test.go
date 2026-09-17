package edge

import (
	"errors"
	"testing"
)

// AETOX PATCH verification. This is a real proof, not a mock: if the patch
// regressed and errorCallback still os.Exit(1)'d when a custom callback is
// installed, this test would take the whole `go test` process down with it —
// a failing test, loudly. Surviving to the assertions IS the guarantee that a
// browser tab's WebView2 error can no longer crash the app.
func TestCustomErrorCallbackDoesNotExit(t *testing.T) {
	e := NewChromium()

	var got error
	e.SetErrorCallback(func(err error) { got = err })
	if !e.customErrorCallback {
		t.Fatal("SetErrorCallback must flag customErrorCallback")
	}

	sentinel := errors.New("transient webview failure")
	e.errorCallback(sentinel) // upstream would os.Exit(1) here

	if got != sentinel {
		t.Fatalf("callback got %v, want the passed error", got)
	}
}

// The second half of the same guarantee, added with revive.go: once a view has
// come up, an error from it is a runtime complaint and never an end to the
// process. The Wails main window is the Chromium this matters for — it installs
// no callback of its own, so before this rule every ExecJS that landed between
// its browser process dying and the ProcessFailed event arriving took the whole
// app with it. Surviving to the end of this test is the proof: without the rule
// the call below is os.Exit(1).
func TestErrorAfterTheViewIsUpDoesNotExit(t *testing.T) {
	e := NewChromium()
	e.everUp = true

	e.errorCallback(errors.New("a call on a view whose browser process has just died"))
}

// Nothing to rebuild without a window and nothing to revive without a hook, and
// both are answered before any COM call is made: a caller that asks out of turn
// gets false rather than a crash (desktop/webview_revive_windows.go).
func TestReviveWithoutAWindowOrAHookAnswersFalse(t *testing.T) {
	e := NewChromium()
	if e.Revive() {
		t.Error("Revive answered true with no window to embed into")
	}
	if e.offerRevival(nil) {
		t.Error("offerRevival answered true with no hook installed")
	}

	BrowserProcessExitedHook = func(*Chromium) bool { return true }
	defer func() { BrowserProcessExitedHook = nil }()
	if e.offerRevival(nil) {
		t.Error("offerRevival answered true with a hook but no window to revive")
	}
}

// A tab's bounds are already physical pixels, but its raster still has to
// follow the monitor. These are independent WebView2 settings: switching the
// first to raw pixels is not permission to freeze the second. On a mixed-DPI
// desk that freeze resamples the whole page after a window move, showing strong
// colour fringes around Thai combining marks while the app's own file view is
// crisp.
//
// This fake vtable exercises the real COM-call boundary too. In v1.0.22 the
// BOOL setter passed a pointer instead of 0/1, so merely asserting that the
// caller supplied `true` would not have proved what WebView2 actually received.
func TestControllerUsesRawBoundsAndTracksMonitorScale(t *testing.T) {
	var boundsCalled, scaleCalled bool
	var boundsMode, detectScale uintptr
	controller3 := &ICoreWebView2Controller3{Vtbl: &ICoreWebView2Controller3Vtbl{
		PutBoundsMode: NewComProc(func(_ uintptr, mode uintptr) uintptr {
			boundsCalled = true
			boundsMode = mode
			return 0
		}),
		PutShouldDetectMonitorScaleChanges: NewComProc(func(_ uintptr, enabled uintptr) uintptr {
			scaleCalled = true
			detectScale = enabled
			return 0
		}),
	}}

	e := NewChromium()
	e.SetErrorCallback(func(err error) { t.Fatalf("configuring controller: %v", err) })
	e.configureController3(controller3)

	if !boundsCalled || boundsMode != uintptr(COREWEBVIEW2_BOUNDS_MODE_USE_RAW_PIXELS) {
		t.Errorf("bounds mode call = (%v, %d), want raw pixels", boundsCalled, boundsMode)
	}
	if !scaleCalled || detectScale != 1 {
		t.Errorf("monitor-scale call = (%v, %d), want BOOL TRUE by value", scaleCalled, detectScale)
	}
}
