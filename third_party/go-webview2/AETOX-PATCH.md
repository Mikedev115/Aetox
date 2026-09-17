# go-webview2 — local fork (Aetox patch)

Vendored copy of `github.com/wailsapp/go-webview2 v1.0.23`, wired via a
`replace` in the root `go.mod`. Same pattern as `third_party/conpty`.

## Why

Upstream `pkg/edge/chromium.go`'s error path is:

```go
func (e *Chromium) errorCallback(err error) {
	e.globalErrorCallback(err)
	os.Exit(1)          // always, even after SetErrorCallback
}
```

Aetox embeds one WebView2 **per browser tab** (`desktop/browser.go`). Any
single tab hitting a transient WebView2 failure — `ERROR_INVALID_STATE`
(0x8007139F) from RivaTuner/RTSS DLL injection, a GPU-driver hiccup, low
memory — routed through `errorCallback` and `os.Exit(1)`'d the **entire app**.
`SetErrorCallback` looked like it prevented this but only swapped the inner
callback; the `os.Exit` fired regardless.

## The patch (search `AETOX PATCH` in pkg/edge/chromium.go)

1. `SetErrorCallback` sets `customErrorCallback = true`.
2. `errorCallback` skips `os.Exit(1)` when a custom callback is installed —
   that callback owns recovery (Aetox logs it and lets the one tab fail).
   The default handler keeps exiting, and at the time of this patch that was
   read as "main-window behavior is unchanged" — the wails main window never
   calls `SetErrorCallback`, so it kept the exit. The sixth patch below is
   what that cost, and what closed it.
3. `CreateCoreWebView2ControllerCompleted` early-returns on failure instead of
   nil-dereferencing `controller` (upstream relied on the now-removed exit),
   sets `inited` to unblock `Embed`'s message loop, and flags `embedFailed`.
4. `Embed` returns `false` on `embedFailed`, so `desktop/browser.go` destroys
   the orphan child window instead of navigating a nil webview.

## A second, unrelated patch

`pkg/edge/ICoreWebView2NavigationCompletedEventArgs.go` binds `GetIsSuccess`.
Upstream declares the vtbl slot but no method, so every `NavigationCompleted`
looked identical whether the page loaded or Chrome rendered its own error page
— `browser_open` reported success over `ERR_FILE_NOT_FOUND`. The sibling
`pkg/webview2` copy of this interface already binds it; this is the same
binding, in the `edge` package's own idiom.

## A third patch: taking a picture of the page

`pkg/edge/capture.go` and `pkg/edge/ICoreWebView2CapturePreviewCompletedHandler.go`
add `Chromium.CapturePreview`. Same shape as the second patch — upstream
declares the vtbl slot and binds nothing, and the sibling `pkg/webview2` copy
already has both halves.

Aetox needs it for annotation (`desktop/browser_shot.go`): a mark drawn on a
page has to be a mark on *something*, and the only honest carrier of "ตรงบริเวณ
นี้" is the rendering itself. It is the engine's own capture rather than a
screen grab of the tab's window, so nothing floating above the window can end up
in the picture.

Two things there are deliberate and easy to undo by accident:

- **The bytes are read off the HGLOBAL** (`GetHGlobalFromStream` + `GlobalLock`),
  not through `IStream::Seek`/`Read`. This package's `IStream` binding covers
  only the two `ISequentialStream` slots, and reading the memory the stream is
  already backed by needs no further vtbl work.
- **`CapturePreview` returns a channel, and the caller must read it off the
  webview thread.** The completion handler is invoked by that thread's message
  pump, so waiting for it there is waiting for the thing that would deliver it.

## A fourth patch: the DevTools door

`pkg/edge/devtools.go` and `pkg/edge/ICoreWebView2CallDevToolsProtocolMethodCompletedHandler.go`
add `Chromium.CallDevToolsProtocolMethod`. Same shape as the second and third
patches, for the third time: upstream declares the vtbl slot
(`corewebview2.go`) and binds nothing, while the sibling `pkg/webview2` copy
already has both halves.

Aetox needs it for `Page.printToPDF`, so a deck written as HTML can be exported
as a PDF (`docs/architecture/html-deck-2026-08-19.md`). The alternative was
binding `ICoreWebView2_7` (or `_16`) plus `ICoreWebView2Environment6` plus
`ICoreWebView2PrintSettings` plus a fourth handler — four interfaces for one
call, where this is one method and one handler and answers every later CDP
question for free.

**One thing in the sibling copy is deliberately not copied.** It declares the
callback as:

```go
func ...Invoke(this *..., errorCode uintptr, result string) uintptr
```

COM passes an `LPCWSTR`. A Go string header is two words with a different
layout, so reading it that way takes the pointer as a length and whatever
follows on the stack as a data pointer. It has never crashed there only because
nothing in this tree calls it. The `edge` package's own ExecuteScript handler
has it right (`*uint16`), and that is the shape followed here.

**What this door does NOT open.** `Page.captureScreenshot` goes through the
compositor that a hidden webview never runs, so PNG export is still blocked —
see `desktop/browser_capture.go` and WebView2Feedback #1077 and #2983. Printing
is a different pipeline; Microsoft's own word for PrintToPdf is "silently".
That asymmetry is why `.pdf` ships and `.png` does not (`desktop/decks.go`).

**Still unverified at the time of writing.** Microsoft documents neither that
`Page.printToPDF` works over WebView2's CDP nor that it does not. This binding
is the cheap way to find out; if the engine refuses it, the fallback is
`ICoreWebView2_16::PrintToPdfStream` and the four interfaces above, for the same
result.

## A fifth patch: closing a webview, and reloading one

`pkg/edge/aetox_lifecycle.go` binds `ICoreWebView2Controller.Close` and
`ICoreWebView2.Reload`, and adds `Chromium.Close` / `Chromium.Reload` on top.
Same shape as the second, third and fourth patches: upstream declares both vtbl
slots and binds neither.

Aetox needs them for a tab whose engine has gone (`desktop/browser.go`,
`engineGone` / `revive`; DECISIONS §227). On 6 ก.ย. the browser process behind
an agent's tab exited while the app kept running; WebView2 answers every call on
a closed webview with `HRESULT_FROM_WIN32(ERROR_INVALID_STATE)`, the tab stayed
registered, and every browser tool call for twenty minutes was refused with the
same sentence. The fix listens to `ProcessFailed` (upstream already registers
the handler and exposes `ProcessFailedCallback`; Aetox had never set it) and
treats `ERROR_INVALID_STATE` from any call as "closed", then destroys the dead
view and creates a new one under the same tab.

- **`Close`** is what actually ends a webview. `win32Tab.destroy` used to be
  `DestroyWindow` alone, leaving the controller to be reclaimed at process exit.
  On a dead engine Close is refused with the same `ERROR_INVALID_STATE`; the
  caller expects that and does not report it.
- **`Reload`** is the one-line answer to `RENDER_PROCESS_EXITED`: the engine is
  fine, the page is not, and reloading puts it back without a new webview.
- Both return the HRESULT as an error rather than routing through
  `errorCallback`, because the code that closes a tab already knows the engine
  may be gone and must not have that fact re-reported as a fresh complaint.

## A sixth patch: bringing a window's browser back

`pkg/edge/revive.go` (and `User32SetTimer` / `User32KillTimer` in
`internal/w32/w32.go`) add `Chromium.Revive`, the hook that decides when it is
called, and the two lines in `chromium.go` that feed it.

WebView2 runs the page in processes of its own, and the browser process among
them can end for reasons that have nothing to do with the app: out of memory, a
GPU driver reset, an antivirus quarantining the runtime mid-run, Task Manager.
The engine then raises `ProcessFailed` with
`COREWEBVIEW2_PROCESS_FAILED_KIND_BROWSER_PROCESS_EXITED`, the controller and
webview behind the window are dead for good, and Microsoft's guidance is to
build a new one. Upstream's Wails host answers that event with a message box and
`os.Exit(-1)` **on the whole process** — every turn in flight and every unsaved
state gone with a dialog nobody asked for. The browser *tabs* had their own
answer (DECISIONS §227, and this file's fifth patch); the window itself did not.

- `BrowserProcessExitedHook` is asked first, on the UI thread but after the
event handler has returned, and only for a Chromium whose owner installed no
error callback of its own — that is the Wails main window. A tab that installs
one owns its own recovery, as before. A hook that takes the event keeps it from
the owner's `ProcessFailedCallback`, so the message box and the exit never
happen.
- `Revive` is the same `Embed` that built the view at startup, on the same
window, followed by everything this package recorded being put back on the new
view: every resource filter (`AddWebResourceRequestedFilter` now records as it
adds) and the last navigation (`Navigate` now records as it navigates). The
settings the owner put on the old webview's `ICoreWebView2Settings` lived on the
object that died, so putting them again is the hook's job — only the owner knows
them (`desktop/webview_revive_windows.go`).
- The dead proxies are **not** released. The process that served them is gone,
and a `Release` on one is a call into nothing; `Embed` overwrites them when the
new controller completes. A call that lands on one meanwhile comes back as an
HRESULT and goes through `errorCallback`, which is why that path had to stop
exiting first.
- `runLater` runs the hook past the handler's return, as the WebView2 samples
do, so the new view is not built inside the old one's event dispatch. It is a
`SetTimer` with its own procedure rather than a `WndProc` hook, because this
package does not own the window. **One callback for the life of the process:**
`syscall.NewCallback` takes a slot from a table of 2000 the runtime never frees
(DECISIONS §295), so the procedure is registered once under `sync.Once` and
finds its closure by timer id.

Two more exits in this file went with it, both the same class — a webview
complaint ending the app:

- `errorCallback` no longer exits once a view has come up (`everUp`, set by
  `CreateCoreWebView2ControllerCompleted`). The Wails main window installs no
  callback, so before this every `ExecJS` that landed between its browser
  process dying and the `ProcessFailed` event arriving took the app with it.
  Before a view has ever come up the exit stands: a WebView2 that cannot be
  built is a window that cannot open.
- `WebResourceRequested` routes a failed `GetRequest` through `errorCallback`
  instead of `log.Fatal`ing. Wails registers that filter for `*`, so one
  unreadable request was `os.Exit(1)` on the whole app.

The frame that hid all of this: a windowsgui build has no stdout. Upstream
writes those complaints with `fmt.Printf`/`fmt.Println` and exits, so on an
installed machine the app simply vanished — no crash file (an exit is not a
panic), no Windows Error Report, no Event 1000, and no last line anywhere. This
fork now uses the standard logger (`log.Printf`), which the host routes into its
own log (`desktop/wails_log.go`), so the last line before an exit names it.

## A seventh patch: a tab follows the monitor it is on

`pkg/edge/chromium.go` keeps raw-pixel bounds but enables WebView2's native
monitor-scale detection. Those settings are independent: raw bounds mean the
controller's rectangle is measured in physical pixels, while
`RasterizationScale` tells Chromium how densely to paint the content inside
that rectangle.

Upstream disables scale detection when it selects raw bounds, which is only
correct for a host that updates `RasterizationScale` itself. Aetox never did.
On the owner's mixed-DPI desk (175% laptop display and 100% external display),
the main Wails view followed a cross-monitor move but the separately embedded
browser tab kept its old raster scale. Windows resampled that stale surface;
the result was soft, colour-fringed web text, especially visible around Thai
vowels and tone marks, while files rendered in the main view stayed crisp.

WebView2 now owns monitor and text-scale changes for each tab. Raw bounds still
prevent a scale change from altering the child window's physical size. The
`TestControllerUsesRawBoundsAndTracksMonitorScale` fake-vtable test pins both
halves and checks the actual COM argument.

The v1.0.23 upstream boolean-ABI fix is part of the vendored baseline in both
`pkg/edge` and `pkg/webview2`: `PutShouldDetectMonitorScaleChanges` passes a
Win32 `BOOL` as integer 0/1, not a pointer to Go's one-byte `bool`. Without
that fix the setting was undefined at the COM boundary.

## Upgrading go-webview2

Re-copy the module, then re-apply the `AETOX PATCH` blocks in `chromium.go`
(the error path, `everUp`, the recording in `Navigate` and
`AddWebResourceRequestedFilter`, the `ProcessFailed` stop, and
`log.Printf` in place of `fmt.Printf` in `globalErrorHandler`, plus native
monitor-scale detection in `configureController3`), `revive.go`,
the `SetTimer`/`KillTimer` pair in `internal/w32/w32.go`, the `GetIsSuccess`
binding, the two capture files, the two DevTools files and
`aetox_lifecycle.go`. Keep the version in this note and the root `go.mod`
require in sync. `pkg/edge/aetox_patch_test.go` is the patch's own proof: it
fails loudly — by taking the test process with it — if the exit comes back.
