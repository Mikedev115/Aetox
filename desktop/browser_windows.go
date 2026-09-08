package main

// The Windows half of the native browser: every tab is a WebView2 embedded in
// a Win32 child window over the dock's browser pane.
//
// Threading model: WebView2 is COM/STA — every webview lives on ONE dedicated
// OS thread that runs a Windows message pump. All operations are marshalled
// onto that thread via a command queue + PostThreadMessage(WM_APP) wake-up.
// This is the one platform that gets a thread of its own; GTK and Cocoa
// require their webviews on the app's main thread instead, which is why
// hostBackend.do is specified as asynchronous rather than "posts to the
// browser thread".

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/wailsapp/go-webview2/pkg/edge"
)

// bridgePost is how a page hands a message back to Go on this engine. See
// metaScript in browser.go.
const bridgePost = "window.chrome.webview.postMessage"

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procShowWindow       = user32.NewProc("ShowWindow")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procPostThreadMsgW   = user32.NewProc("PostThreadMessageW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")

	// Detaching a tab into a window of its own: reparent it out of the app,
	// give it a real frame, name it for the taskbar, and let the OS place it.
	// See win32Tab.detach.
	procSetParent         = user32.NewProc("SetParent")
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procSetWindowTextW    = user32.NewProc("SetWindowTextW")

	procGetWindowDpiAwarenessCtx = user32.NewProc("GetWindowDpiAwarenessContext")
	procSetThreadDpiAwarenessCtx = user32.NewProc("SetThreadDpiAwarenessContext")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")

	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")

	ole32              = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
)

const (
	wmApp      = 0x8000
	wsChild    = 0x40000000
	wsVisible  = 0x10000000
	wsClipSibl = 0x04000000
	wsPopup    = 0x80000000
	swHide     = 0

	// A window that is visible to the compositor and to nobody else: no taskbar
	// button, never activated, never stealing focus. Paired with a position
	// outside every monitor, this is the deck export's window (deck_render.go).
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000

	// A window of its own: title bar, border, minimise/maximise, resizable —
	// everything WS_CHILD is not. What a detached tab wears (win32Tab.detach).
	// No WS_EX_TOOLWINDOW here, unlike the export window: this one is a window
	// the user works in, so it belongs on the taskbar where they can find it
	// after alt-tabbing away.
	wsOverlappedWindow = 0x00CF0000
	gwlStyle           = ^uintptr(15) // GWL_STYLE, which is -16
	swpFrameChanged    = 0x0020
	swShowNormal       = 1
	swRestore          = 9
	wmSize             = 0x0005
	wmClose            = 0x0010

	coinitApartmentThreaded = 0x2

	// hwndTop + these SWP flags force the tab's WebView2 child window to the
	// top of the Z order: two separate WebView2 controllers in the same
	// top-level window each composite independently, so plain ShowWindow/
	// MoveWindow (no Z-order change) can leave the tab rendered behind the
	// app's own webview — invisible, even though it's really navigated and
	// painting.
	hwndTop = 0
	// The other end of the same fact: a tab sent to the BOTTOM of the Z order
	// is still visible to Windows and still painting, and simply has the app's
	// own webview drawn over it. That is what an export needs — composited, so
	// a capture has frames to capture, and unseen, so nothing flashes. See
	// sendBehind.
	hwndBottom    = 1
	swpNoMove     = 0x0002
	swpNoSize     = 0x0001
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
)

type winMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type wndClassExW struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

// win32Tab is one WebView2 in its own child window.
type win32Tab struct {
	hwnd     uintptr
	chromium *edge.Chromium
	// detached: this tab has left the panel and is a top-level window of its
	// own (detach). Two things stop happening the moment it is true — the pane
	// no longer says where the tab sits, and nothing hides it — and both are
	// enforced here as well as at the caller, because the pane is one window’s
	// opinion and this is the window itself.
	detached bool
	// onClosed fires when somebody closes the detached window’s frame. It is
	// the only way a tab can end that does not come from Aetox’s own strip, and
	// without it the × on that frame would destroy the window while the agent
	// went on holding a tab that no longer exists.
	onClosed func()
	// forget takes this tab out of the host’s hwnd index. Held as a closure
	// rather than a host pointer so a tab built by a test needs no host.
	forget func()
	// hostThread and reportErr are the tripwire, not the plumbing: see
	// requireHostThread.
	hostThread uint32
	reportErr  func(error)
}

// requireHostThread is CEF's CEF_REQUIRE_UI_THREAD, which is DCHECK(CefCurrentlyOn(TID_UI))
// at the top of every function that has a thread it must be on. Same idea, one
// difference that matters here.
//
// browserHost.onTab already makes the accident impossible at compile time —
// there is no view to reach without going through it. What a compiler cannot
// see is a tabView copied out of an onTab callback and used later, and nothing
// in Go will ever see that. CEF's answer covers exactly that gap, so the two
// together cover more than either alone: a wall for the reach, a tripwire for
// the stash.
//
// It does not panic. A crash in a shipping desktop app over a browser tab is
// not a trade worth making, and a log line is what §127.8 already proved nobody
// reads. So it reports through onEngineError — the channel built for precisely
// this in §128.4 — which means a wrong-thread call now names itself, in our own
// words, in the tool result, before WebView2's more cryptic refusal arrives.
//
// The call is then made anyway. The engine will refuse it; the point is that by
// then somebody has already been told why.
func (t *win32Tab) requireHostThread(what string) {
	if t.hostThread == 0 {
		return // a tab built by a test, with no host thread to be off
	}
	cur, _, _ := procGetCurrentThreadID.Call()
	if uint32(cur) == t.hostThread {
		return
	}
	err := fmt.Errorf("browser.%s was called from thread %d, not the webview's thread %d — WebView2 will refuse it", what, cur, t.hostThread)
	debuglog.Msg("%v", err)
	if t.reportErr != nil {
		t.reportErr(err)
	}
}

func (t *win32Tab) navigate(url string) {
	t.requireHostThread("navigate")
	t.chromium.Navigate(url)
}

func (t *win32Tab) eval(js string) {
	t.requireHostThread("eval")
	t.chromium.Eval(js)
}

func (t *win32Tab) setZoom(f float64) {
	t.requireHostThread("setZoom")
	t.chromium.PutZoomFactor(f)
}

func (t *win32Tab) openDevTools() {
	t.requireHostThread("openDevTools")
	t.chromium.OpenDevToolsWindow()
}

// capture adapts the vendored patch's result type to the portable one, so
// browser.go's tabView never names an engine. See third_party/go-webview2's
// AETOX-PATCH.md.
func (t *win32Tab) capture() <-chan shotResult {
	out := make(chan shotResult, 1)
	t.requireHostThread("capture")
	src := t.chromium.CapturePreview()
	go func() {
		r := <-src
		out <- shotResult{PNG: r.PNG, Err: r.Err}
	}()
	return out
}

// callEngine runs a Chrome DevTools Protocol method on this tab.
//
// The DevTools door rather than ICoreWebView2_7::PrintToPdf and friends: that
// route is four COM interfaces for one call (the webview's, the environment's,
// the settings', and a handler), where this is one method and one handler —
// both already patched into the vendored fork, and reusable for every later CDP
// question. See third_party/go-webview2/AETOX-PATCH.md.
//
// Nothing is decoded here. What "data" means differs per method, and the layer
// that asked is the layer that knows.
func (t *win32Tab) callEngine(method, paramsJSON string) <-chan engineReply {
	out := make(chan engineReply, 1)
	t.requireHostThread("callEngine")
	src := t.chromium.CallDevToolsProtocolMethod(method, paramsJSON)
	go func() {
		r := <-src
		out <- engineReply{JSON: r.JSON, Err: r.Err}
	}()
	return out
}

// sendBehind is what the deck export used to ask for and no longer can.
//
// It set the tab to the bottom of the Z order on the reasoning that a tab left
// under the app's own webview is "invisible, even though it's really navigated
// and painting" — a sentence this file records from experience. The reasoning
// was right about siblings and wrong about this: the app's webview is the
// PARENT's content, and a child window paints over its parent's client area
// whatever the Z order says. The export tab sat on top of the whole application
// for the length of every export.
//
// The export now gets its own top-level window instead (see openTab), so this
// has no caller. It stays because tabView declares it and because a future
// engine may have a Z order that does mean something here.
func (t *win32Tab) sendBehind() {
	t.requireHostThread("sendBehind")
	procSetWindowPos.Call(t.hwnd, hwndBottom, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
}

// setBounds moves and raises; it does not show. SWP_SHOWWINDOW rode along here
// until 7 ก.ย., which made every re-glue a show: the pane reflows on a window
// resize whether or not its tab is on screen, so resizing the app with
// settings open — or with the tab hidden for any other reason — surfaced the
// page over whatever was drawn there. Showing is setVisible's job, and the
// portable layer (BrowserSetBounds) refuses to surface a hidden tab anyway;
// this line just stops asking for it.
func (t *win32Tab) setBounds(x, y, w, h int) {
	t.requireHostThread("setBounds")
	// A detached window is where the user put it. The pane goes on measuring
	// its own rectangle and goes on reporting it — it has no idea this tab left
	// — so the refusal lives here, at the window, rather than depending on
	// every caller remembering.
	if t.detached {
		return
	}
	procSetWindowPos.Call(t.hwnd, hwndTop, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoActivate)
	t.chromium.Resize()
}

// detach takes this tab out of the app and gives it a window of its own.
//
// The owner's call, 8 ก.ย.: *"ไม่อยากให้ผูกอยู่แค่กับใน Aetox ... แต่ Aetox ก็ตามไป
// ควบคุมได้นะ"* — and the second half needs no code at all, which is the fact
// that makes this feature small. The agent talks to the WebView2 through CDP
// and the page bridge; neither has ever known where the HWND sits. read, click,
// type, capture and the ref marks all keep working on a window that is no
// longer anywhere near the app.
//
// It REPARENTS rather than rebuilding. A second window with the same URL is not
// the same window: the page would reload, losing the scroll position, the form
// half-filled, the session the page is holding in memory — and the refs from
// the last read would be pointing at a document that no longer exists.
//
// The Win32 dance is the standard one and every step is load-bearing:
// SetParent(0) makes it top-level, GWL_STYLE swaps WS_CHILD for a real frame,
// and SWP_FRAMECHANGED is what makes the window recalculate its non-client area
// — without it the frame is not drawn and the client rect is still the child's,
// so the page is laid out for a window that is not the one on screen.
//
// The export window (openTab) reaches the same place by being born there. This
// one has to arrive mid-life, because a tab is only worth detaching once there
// is something on it.
func (t *win32Tab) detach(title string, w, h int) {
	t.requireHostThread("detach")
	if t.detached {
		// Already out. Asking again is "bring it to me", which is the useful
		// reading of a second press and costs nothing to honour.
		procShowWindow.Call(t.hwnd, uintptr(swRestore))
		procSetWindowPos.Call(t.hwnd, hwndTop, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow|swpNoActivate)
		return
	}
	procSetParent.Call(t.hwnd, 0)
	procSetWindowLongPtrW.Call(t.hwnd, gwlStyle, uintptr(wsOverlappedWindow|wsVisible))
	// Named for the taskbar. A row of untitled windows is what alt-tab shows
	// otherwise, and the whole point of detaching is that the user goes and
	// finds this page later.
	if title != "" {
		if p, err := syscall.UTF16PtrFromString(title); err == nil {
			procSetWindowTextW.Call(t.hwnd, uintptr(unsafe.Pointer(p)))
		}
	}
	if w <= 0 || h <= 0 {
		w, h = detachedW, detachedH
	}
	x, y := detachedSpawn()
	procSetWindowPos.Call(t.hwnd, hwndTop, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpFrameChanged|swpShowWindow)
	procShowWindow.Call(t.hwnd, uintptr(swShowNormal))
	t.detached = true
	// After the frame, not before: the client rect the page is laid out in is
	// the one the new style produced.
	t.chromium.Resize()
}

// detachedW, detachedH and detachedSpawn are where a window that has just left
// the panel appears. Cascaded rather than centred, because the second one has
// to be findable when it lands on top of the first.
const (
	detachedW = 1100
	detachedH = 800
)

var detachedNth int

func detachedSpawn() (int, int) {
	const step = 28
	n := detachedNth % 8
	detachedNth++
	return 120 + n*step, 90 + n*step
}

func (t *win32Tab) setVisible(visible bool) {
	t.requireHostThread("setVisible")
	// Detached: raise it when asked, and never hide it. Hiding is the panel’s
	// idea of a tab that is not the current one, and this window has no current
	// tab to lose to — a capture that hid the user’s window to photograph it
	// would be absurd, and a session switch that hid it would be the panel
	// reaching into a window it gave away.
	if t.detached {
		if visible {
			procShowWindow.Call(t.hwnd, uintptr(swRestore))
			procSetWindowPos.Call(t.hwnd, hwndTop, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow|swpNoActivate)
		}
		return
	}
	if visible {
		procSetWindowPos.Call(t.hwnd, hwndTop, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow|swpNoActivate)
		return
	}
	procShowWindow.Call(t.hwnd, uintptr(swHide))
}

// destroy closes the controller, then the window. Close is what actually ends
// a webview — DestroyWindow alone left the controller to be reclaimed at
// process exit, which the old comment here called a ponytail. On a tab whose
// engine has already gone, Close is refused; the refusal is the expected
// answer and is not reported.
func (t *win32Tab) destroy() {
	t.requireHostThread("destroy")
	if t.forget != nil {
		t.forget()
	}
	if err := t.chromium.Close(); err != nil && !engineClosed(err) {
		debuglog.Msg("browser: closing a tab's webview: %v", err)
	}
	procDestroyWindow.Call(t.hwnd)
}

// errEngineProcessExited is what ProcessFailed says when the browser process
// behind a tab ends. Named because it reaches the agent's answer.
var errEngineProcessExited = errors.New("the browser engine's process exited")

// engineClosed reports whether err is the engine saying "this webview is
// closed" rather than complaining about the call it was given.
//
// WebView2 answers every call on a closed CoreWebView2 with
// HRESULT_FROM_WIN32(ERROR_INVALID_STATE) — the same code it returns after
// Close(), and after the browser process it lived in has exited. The RPC
// codes are the same fact seen from one layer down: the COM proxy found no
// server on the other end of the pipe.
//
// The message Windows renders for it — "The group or resource is not in the
// correct state to perform the requested operation" — is the sentence that sat
// in the log fifteen times on 6 ก.ย. while the tool reported a network
// problem. It is not a network problem, and it is not transient: no call on
// that webview will ever succeed again. The only recovery is a new one.
func engineClosed(err error) bool {
	var code syscall.Errno
	if !errors.As(err, &code) {
		return false
	}
	switch uint32(code) {
	case 0x139F, // ERROR_INVALID_STATE
		0x8007139F, // HRESULT_FROM_WIN32(ERROR_INVALID_STATE)
		0x80010108, // RPC_E_DISCONNECTED: the object invoked has disconnected from its clients
		0x800706BA, // HRESULT_FROM_WIN32(RPC_S_SERVER_UNAVAILABLE)
		0x800706BE: // HRESULT_FROM_WIN32(RPC_S_CALL_FAILED)
		return true
	}
	return false
}

type win32Host struct {
	mu       sync.Mutex
	cmds     []func()
	threadID uint32
	parent   uintptr
	attempt  *startAttempt
	started  bool
	class    *uint16
	// byHwnd is how the window procedure finds the tab a message is about.
	// It exists because a detached window has messages worth answering — it is
	// resized and closed by the user now, not by the pane — and the class’s
	// procedure is shared by every tab, so it is handed an HWND and nothing
	// else. Guarded by mu with the queue above; the procedure runs on the host
	// thread and nothing holds mu across a wait for that thread.
	wins map[uintptr]*win32Tab
}

// startAttempt is one try at bringing the host thread up: a latch that closes
// exactly once, and the reason it closed.
//
// It replaces a bare `ready chan struct{}` that only ever closed on the happy
// path. err is read only after done closes, so it needs no lock of its own.
type startAttempt struct {
	done chan struct{}
	err  error
}

func newAttempt() *startAttempt { return &startAttempt{done: make(chan struct{})} }

// browserStartBudget bounds the wait for the host thread.
//
// The thread is a CoInitializeEx, a DPI call and a RegisterClassExW — about a
// millisecond in practice, which browser.start logs every launch. The budget is
// not a latency target, it is the promise that a browser tool ENDS. Owner, 22
// ส.ค.: *"มันไม่ควรหยุดสิ ... เว้นแต่จะถูกกดหยุด"* — closing a tab or a panel is
// a user-side event the model is told about and works around, and the only
// thing that ends a turn is Stop.
// A var rather than a const only so the test that pins the bound can shorten
// it, the same way toolProgressInterval is reached in internal/model.
var browserStartBudget = 10 * time.Second

func newHostBackend() hostBackend { return &win32Host{attempt: newAttempt()} }

// start spins up the dedicated STA browser thread (idempotent).
func (h *win32Host) start() error {
	h.mu.Lock()
	if h.started {
		att := h.attempt
		h.mu.Unlock()
		return await(att)
	}
	h.started = true
	att := h.attempt
	h.mu.Unlock()

	parent := findOwnMainWindow()
	if parent == 0 {
		debuglog.Msg("browser.start: main window not found")
		// Released with the reason, not merely un-claimed.
		//
		// h.started goes up BEFORE this lookup runs, so a second binding
		// arriving during it takes the branch above and parks on this same
		// attempt — and this path used to abandon that channel without ever
		// closing it. Nothing woke those callers again, ever, and
		// browserHostLazy calls start() on EVERY binding: one failed lookup
		// could park the whole browser surface for the life of the process,
		// with no error, no timeout and nothing in the log. That is the shape
		// of "นิ่งค้างไปเลย".
		h.abandon(att, fmt.Errorf("main window not found"))
		return await(att)
	}
	debuglog.Msg("browser.start: parent hwnd=%#x (pid=%d)", parent, os.Getpid())
	h.parent = parent

	go h.run(att)
	if err := await(att); err != nil {
		return err
	}
	debuglog.Msg("browser.start: host thread ready (tid=%d)", h.threadID)
	return nil
}

// abandon ends a failed attempt: everyone parked on it wakes with the reason,
// and the next caller gets a fresh latch to try again on.
func (h *win32Host) abandon(att *startAttempt, err error) {
	h.mu.Lock()
	h.started = false
	if h.attempt == att {
		h.attempt = newAttempt()
	}
	h.mu.Unlock()
	att.err = err // before the close: nobody may read it until then
	close(att.done)
}

// await blocks until this attempt settles, and never longer than the budget.
func await(att *startAttempt) error {
	select {
	case <-att.done:
		return att.err
	case <-time.After(browserStartBudget):
		return fmt.Errorf("เบราว์เซอร์ยังไม่พร้อมใน %s", browserStartBudget)
	}
}

// findOwnMainWindow returns this process's visible top-level window (the wails
// main window). Never look it up by TITLE: FindWindowW("Aetox Desktop") matches
// any window that happens to carry that text — a browser tab showing the dev
// URL, explorer's taskbar thumbnail host, another instance — and a parent from
// a foreign process makes every CreateWindowExW child fail with "Access is
// denied", silently killing all browser tabs.
//
// ponytail: enumerating at all is what the port blueprint's rule 1 says not to
// do ("hold a direct handle from the toolkit"). Wails v2.13 exports no such
// handle, so retiring this needs a patch to the vendored Wails — planned with
// phase 3a, where GTK forces the question anyway. ARCHITECTURE.md §48.
func findOwnMainWindow() uintptr {
	self := uint32(os.Getpid())
	var found uintptr
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid != self {
			return 1 // keep enumerating
		}
		if vis, _, _ := procIsWindowVisible.Call(hwnd); vis == 0 {
			return 1
		}
		found = hwnd
		return 0 // stop
	})
	procEnumWindows.Call(cb, 0)
	return found
}

func (h *win32Host) run(att *startAttempt) {
	runtime.LockOSThread()
	procCoInitializeEx.Call(0, coinitApartmentThreaded)

	// Match the main window's DPI awareness context. Windows refuses to
	// create a child window whose thread runs under a different DPI context
	// than the parent — CreateWindowExW fails with ERROR_ACCESS_DENIED. A raw
	// goroutine thread starts on the process default, which does not
	// necessarily match the wails main window's per-monitor context.
	if ctx, _, _ := procGetWindowDpiAwarenessCtx.Call(h.parent); ctx != 0 {
		prev, _, _ := procSetThreadDpiAwarenessCtx.Call(ctx)
		debuglog.Msg("browser.run: thread DPI ctx set to parent's (prev=%#x)", prev)
	}

	tid, _, _ := procGetCurrentThreadID.Call()
	h.threadID = uint32(tid)

	// Child window class; all messages go to DefWindowProc — sizing is driven
	// explicitly from BrowserSetBounds.
	wndProc := syscall.NewCallback(func(hwnd, msg, wparam, lparam uintptr) uintptr {
		// Two messages a detached window gets and a child window never did.
		//
		// WM_SIZE: the user drags the frame, so the page has to be laid out again.
		// A child window was only ever resized by BrowserSetBounds, which called
		// Resize itself — the comment above this class used to say sizing is driven
		// explicitly, and that stopped being the whole truth the day a window could
		// be resized by somebody else.
		//
		// WM_CLOSE: the × on that frame. Answered rather than passed on, so the
		// window is not destroyed here — it goes back out through the same close
		// path a × in the strip takes, which is the one that tells the agent its
		// page is gone. Destroying first and reporting after would leave a moment
		// in which the tab is in the map and its window is not.
		switch msg {
		case wmSize:
			if t := h.tabByHwnd(hwnd); t != nil && t.chromium != nil {
				t.chromium.Resize()
			}
		case wmClose:
			if t := h.tabByHwnd(hwnd); t != nil && t.onClosed != nil {
				t.onClosed()
				return 0
			}
		}
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		return r
	})
	className, _ := syscall.UTF16PtrFromString("AetoxBrowserHost")
	h.class = className
	wc := wndClassExW{
		Size:      uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:   wndProc,
		ClassName: className,
	}
	atom, _, regErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	debuglog.Msg("browser.run: RegisterClassExW atom=%d err=%v", atom, regErr)

	close(att.done)

	var msg winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			return
		}
		h.drain()
		if msg.Message != wmApp {
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

// do queues fn onto the browser thread and wakes its pump. Asynchronous, as
// hostBackend.do requires.
func (h *win32Host) do(fn func()) {
	h.mu.Lock()
	h.cmds = append(h.cmds, fn)
	h.mu.Unlock()
	procPostThreadMsgW.Call(uintptr(h.threadID), wmApp, 0, 0)
}

func (h *win32Host) drain() {
	for {
		h.mu.Lock()
		if len(h.cmds) == 0 {
			h.mu.Unlock()
			return
		}
		fn := h.cmds[0]
		h.cmds = h.cmds[1:]
		h.mu.Unlock()
		fn()
	}
}

// openTab creates the child window + WebView2 for a tab. Already on the
// browser thread — browserHost.open calls this from inside do.
// offscreenTabID names the one tab that gets a window of its own rather than a
// child of the app's. Kept here rather than passed in because the reason is a
// Win32 fact, not a caller's preference — see below.
const offscreenTabID = exportTabID

// tabByHwnd answers the window procedure's one question. Nil for a window this
// host does not own, which includes every message that arrives after a tab has
// been destroyed and before its queue has drained.
func (h *win32Host) tabByHwnd(hwnd uintptr) *win32Tab {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.wins[hwnd]
}

func (h *win32Host) rememberWindow(hwnd uintptr, t *win32Tab) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.wins == nil {
		h.wins = map[uintptr]*win32Tab{}
	}
	h.wins[hwnd] = t
}

func (h *win32Host) forgetWindow(hwnd uintptr) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.wins, hwnd)
}

func (h *win32Host) openTab(id, url string, x, y, w, hgt int, cb tabCallbacks) tabView {
	// A child window ALWAYS paints over its parent's client area. Z-order only
	// orders siblings, and the app's own webview is not a sibling — it is the
	// parent's content. So there is no arrangement of a WS_CHILD window that is
	// composited and unseen, which is what the deck export needs: frames to
	// capture, and nothing on screen while it works. `sendBehind` was written
	// on the opposite assumption and could never have worked.
	//
	// Its own top-level window can be both. WS_POPUP at -32000 is visible to
	// the compositor — DWM keeps a redirection surface for it, so it paints —
	// and outside every monitor, so nobody sees it. WS_EX_TOOLWINDOW keeps it
	// off the taskbar and WS_EX_NOACTIVATE keeps it from stealing focus.
	style, exStyle, parent := uintptr(wsChild|wsVisible|wsClipSibl), uintptr(0), h.parent
	if id == offscreenTabID {
		style, exStyle, parent = wsPopup|wsVisible, wsExToolWindow|wsExNoActivate, 0
		x, y = -32000, -32000
	}
	hwnd, _, lastErr := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(h.class)),
		0,
		style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(hgt),
		parent, 0, 0, 0,
	)
	if hwnd == 0 {
		debuglog.Msg("browser.open(%s): CreateWindowExW FAILED: %v", id, lastErr)
		return nil
	}

	chromium := edge.NewChromium()
	chromium.DataPath = webviewUserDataDir("browser")
	if chromium.DataPath == "" {
		chromium.DataPath = filepath.Join(os.Getenv("AppData"), "aetox-browser")
	}
	chromium.SetErrorCallback(func(err error) {
		// default handler calls os.Exit(1) — never acceptable for a tab
		fmt.Fprintln(os.Stderr, "browser tab error:", err)
		debuglog.Msg("browser tab %s error: %v", id, err)
		// And out to the portable half, which is the part that was missing.
		// For a week this function was the entire fate of every engine
		// complaint: two lines nobody reads while the tool above answered
		// "page did not finish loading" and the agent guessed at the network.
		if cb.onEngineError != nil {
			cb.onEngineError(err)
		}
		// A refusal because the webview is CLOSED is not a complaint about
		// the call, it is the engine's only way of saying the browser
		// process behind this tab is gone. Reported separately, because the
		// answer to it is a new engine, not a better call. See engineClosed.
		if cb.onEngineGone != nil && engineClosed(err) {
			cb.onEngineGone(err)
		}
	})

	// The engine's own account of a process behind this tab ending. The
	// browser process is the one that matters: when it goes, this webview is
	// closed and stays closed, and until this handler existed nobody was told —
	// the tab sat in the host's map, black, answering every call with
	// ERROR_INVALID_STATE for the rest of the session (6 ก.ย.: twenty-two
	// minutes of "the browser engine refused what Aetox asked it to do").
	//
	// A renderer dying is a smaller event with a smaller answer: the engine is
	// fine, the page is not, and Reload puts the page back.
	chromium.ProcessFailedCallback = func(_ *edge.ICoreWebView2, args *edge.ICoreWebView2ProcessFailedEventArgs) {
		if args == nil {
			return
		}
		kind, err := args.GetProcessFailedKind()
		if err != nil {
			debuglog.Msg("browser tab %s: a process failed but the engine would not say which: %v", id, err)
			return
		}
		switch kind {
		case edge.COREWEBVIEW2_PROCESS_FAILED_KIND_BROWSER_PROCESS_EXITED:
			debuglog.Msg("browser tab %s: the browser process exited", id)
			if cb.onEngineGone != nil {
				cb.onEngineGone(errEngineProcessExited)
			}
		case edge.COREWEBVIEW2_PROCESS_FAILED_KIND_RENDER_PROCESS_EXITED:
			debuglog.Msg("browser tab %s: the page's renderer exited; reloading", id)
			chromium.Reload()
		case edge.COREWEBVIEW2_PROCESS_FAILED_KIND_RENDER_PROCESS_UNRESPONSIVE:
			debuglog.Msg("browser tab %s: the page's renderer is not responding", id)
		default:
			debuglog.Msg("browser tab %s: a helper process ended (kind %d); the engine restarts those itself", id, kind)
		}
	}

	view := &win32Tab{hwnd: hwnd, chromium: chromium, hostThread: h.threadID, reportErr: cb.onEngineError, onClosed: cb.onWindowClosed}
	// Indexed by window handle from here on: the class procedure is shared by
	// every tab and is handed an HWND, so this map is the only way a message
	// about a detached window finds the tab it belongs to.
	h.rememberWindow(hwnd, view)
	view.forget = func() { h.forgetWindow(hwnd) }

	chromium.MessageCallback = func(message string, _ *edge.ICoreWebView2, args *edge.ICoreWebView2WebMessageReceivedEventArgs) {
		source, _ := args.GetSource()
		cb.onMessage(message, source)
	}
	chromium.NavigationCompletedCallback = func(_ *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) {
		// An unreadable flag counts as success: the tab is usable either way,
		// and inventing a failure is worse than missing one.
		ok := true
		if args != nil {
			if success, err := args.GetIsSuccess(); err == nil {
				ok = success
			} else {
				debuglog.Msg("browser tab %s: navigation status unavailable: %v", id, err)
			}
		}
		cb.onNavDone(view, ok)
	}

	debuglog.Msg("browser.open(%s): embedding webview (dataPath=%s)", id, chromium.DataPath)
	if !chromium.Embed(hwnd) {
		debuglog.Msg("browser.open(%s): Embed FAILED", id)
		procDestroyWindow.Call(hwnd)
		return nil
	}
	debuglog.Msg("browser.open(%s): embed ok, navigating", id)
	chromium.Resize()
	view.setVisible(true)

	// Registered here and not a line earlier: Init reaches through to the
	// controller, which does not exist until Embed has returned true — Embed's
	// own last act is an Init call for exactly that reason. It still lands
	// before any page script, because it applies to documents created from now
	// on and the first Navigate is below. A page whose first statement is
	// confirm() would otherwise get the real blocking dialog and stop the tab
	// dead with nobody able to answer it. See dialogScript.
	chromium.Init(dialogScript())
	// Second Init rather than one concatenated script: AddScriptToExecuteOnDocumentCreated
	// is additive, and two scripts that answer two unrelated questions should
	// be readable as two. Same timing argument as the dialog's — a page whose
	// first statement throws is exactly the error worth recording, and a
	// recorder installed a line later would miss it. See browser_log.go.
	chromium.Init(logScript())

	if url != "" {
		chromium.Navigate(url)
	}
	return view
}
