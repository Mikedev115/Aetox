//go:build windows

package main

// The companion's body on the desktop: a window of its own, outside the app.
//
// Wails v2 has one window and no way to open a second, so a mascot drawn
// inside it can never leave it. The owner wants it to ("ไปไหนมาไหนบนจอเราได้ …
// ลากไปจอไหนก็ได้เหมือน Codex") and wants no second WebView2 for it ("กินแรม
// เยอะโดยไม่จำเป็น แค่ตัวเล็ก ๆ ตัวเดียว"). What every desktop pet uses instead
// is here: a Win32 *layered* window — `UpdateLayeredWindow` hands Windows a
// premultiplied ARGB bitmap and the compositor does the rest. Per-pixel
// transparency, and the part that makes it livable: a click on a pixel with
// alpha 0 goes through to whatever is behind, by the OS's own hit-test, so
// the window can be generously sized and still not stand in anyone's way.
// Tool window (no taskbar button), never activated (never steals focus),
// topmost. A few megabytes, no extra process.
//
// The window lives on a thread of its own with its own message pump, the
// same shape as win32Host.run in browser_windows.go and deliberately not the
// same thread: that one exists for WebView2 and COM and is only started when
// an agent opens a browser tab; this one must not wake it up. Commands from
// other goroutines reach the thread through `do`, which queues and posts
// WM_APP, exactly as the browser host does.
//
// This file owns the window, the mouse and the clock. What the window SHOWS
// is composed by companion_draw.go from the baked frames; what the assistant
// is doing and saying arrives from the app window through apply (companion.go
// SetCompanionState); what the user does to the body — click, drag, the two
// buttons — goes back the same way, through `on`. The brain stays in the app
// window; this is a body.

import (
	"errors"
	"fmt"
	"image"
	"math/rand/v2"
	"os"
	"runtime"
	"sort"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// Only what the package does not declare already: the layered-window calls
// and constants live with the driving overlay (computer_overlay_windows.go),
// the DCs and rects with the reach tools, the rest with browser_windows.go.
var (
	procSetCapture       = user32.NewProc("SetCapture")
	procReleaseCapture   = user32.NewProc("ReleaseCapture")
	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW  = user32.NewProc("GetMonitorInfoW")
	procGetDpiForWindow  = user32.NewProc("GetDpiForWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procTrackMouseEvent  = user32.NewProc("TrackMouseEvent")
	procSetTimer         = user32.NewProc("SetTimer")
	procKillTimer        = user32.NewProc("KillTimer")
	procLoadCursorW      = user32.NewProc("LoadCursorW")
	procSetCursor        = user32.NewProc("SetCursor")

	procSetThreadPriority = kernel32.NewProc("SetThreadPriority")
	procGetCurrentThread  = kernel32.NewProc("GetCurrentThread")

	procSHQueryUserNotificationState = shell32.NewProc("SHQueryUserNotificationState")

	// Windows' timers tick at 15.6ms unless asked for better; a walk paced
	// by them is not smooth. Asked for 1ms only while something moves, and
	// given back when it stops — it costs power to keep.
	winmm               = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod   = winmm.NewProc("timeEndPeriod")

	// DwmFlush waits for the compositor's next frame: called after a
	// presented frame while the figure moves, it paces the body to the
	// screen's own refresh, so no two updates land in one composition and
	// none is skipped — the difference between 60 frames and 60 smooth ones.
	dwmapi       = syscall.NewLazyDLL("dwmapi.dll")
	procDwmFlush = dwmapi.NewProc("DwmFlush")

	// The monitor's own vertical blank, through the kernel-mode thunks GDI
	// exports: DwmFlush waits for the compositor's frame, which on a desktop
	// of mixed refresh rates is the primary's — 60Hz here, while the second
	// monitor runs 144 (owner, 13 ก.ย. 2026: "เอา Hz เท่าจอไม่ได้หรอ"). Waiting
	// on the blank of the monitor the figure is on gives it that monitor's
	// rate.
	procCreateDCW                       = gdi32.NewProc("CreateDCW")
	procD3DKMTOpenAdapterFromHdc        = gdi32.NewProc("D3DKMTOpenAdapterFromHdc")
	procD3DKMTCloseAdapter              = gdi32.NewProc("D3DKMTCloseAdapter")
	procD3DKMTWaitForVerticalBlankEvent = gdi32.NewProc("D3DKMTWaitForVerticalBlankEvent")
	procGetMonitorInfoExW               = user32.NewProc("GetMonitorInfoW")
)

type monitorInfoEx struct {
	monitorInfo
	Device [32]uint16
}

type d3dkmtOpenAdapterFromHdc struct {
	HDC           uintptr
	Adapter       uint32
	AdapterLuid   [2]uint32
	VidPnSourceID uint32
}

type d3dkmtWaitForVerticalBlankEvent struct {
	Adapter       uint32
	Device        uint32
	VidPnSourceID uint32
}

// vblankWaiter waits for one monitor's vertical blank.
type vblankWaiter struct {
	device  string
	adapter uint32
	source  uint32
}

// openVBlank opens the adapter behind the monitor at the point; nil when
// the thunks are not there (a remote desktop, an odd driver), in which case
// the caller falls back to DwmFlush.
func openVBlank(x, y int) *vblankWaiter {
	mi := monitorInfoEx{}
	mi.Size = uint32(unsafe.Sizeof(mi))
	mon, _, _ := procMonitorFromPoint.Call(packPoint(x, y), monitorDefaultToNearest)
	if mon == 0 {
		return nil
	}
	if ok, _, _ := procGetMonitorInfoExW.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return nil
	}
	device := syscall.UTF16ToString(mi.Device[:])
	display, _ := syscall.UTF16PtrFromString("DISPLAY")
	dev, _ := syscall.UTF16PtrFromString(device)
	hdc, _, _ := procCreateDCW.Call(uintptr(unsafe.Pointer(display)), uintptr(unsafe.Pointer(dev)), 0, 0)
	if hdc == 0 {
		return nil
	}
	defer procDeleteDC.Call(hdc)
	open := d3dkmtOpenAdapterFromHdc{HDC: hdc}
	if status, _, _ := procD3DKMTOpenAdapterFromHdc.Call(uintptr(unsafe.Pointer(&open))); status != 0 {
		return nil
	}
	return &vblankWaiter{device: device, adapter: open.Adapter, source: open.VidPnSourceID}
}

// wait blocks until the monitor's next vertical blank; false if it cannot.
func (v *vblankWaiter) wait() bool {
	arg := d3dkmtWaitForVerticalBlankEvent{Adapter: v.adapter, VidPnSourceID: v.source}
	status, _, _ := procD3DKMTWaitForVerticalBlankEvent.Call(uintptr(unsafe.Pointer(&arg)))
	return status == 0
}

func (v *vblankWaiter) close() {
	arg := struct{ Adapter uint32 }{v.adapter}
	procD3DKMTCloseAdapter.Call(uintptr(unsafe.Pointer(&arg)))
}

const (
	wmDestroy        = 0x0002
	wmTimer          = 0x0113
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmMouseLeave     = 0x02A3
	wmCaptureChanged = 0x0215
	wmDPIChanged     = 0x02E0

	swpNoZOrder = 0x0004

	// Per-monitor v2: the window is told (WM_DPICHANGED) when it is dragged
	// onto a monitor with a different scale, and nothing is stretched for it.
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is the handle value -4.
	dpiAwarenessPerMonitorV2 = ^uintptr(3)

	monitorDefaultToNull    = 0
	monitorDefaultToNearest = 2

	tmeLeave = 0x00000002

	wmSetCursor = 0x0020
	// A window class with no cursor shows whatever was last set — on a
	// thread that looks busy, the arrow with the spinning ring, which reads
	// as "loading" on a figure that is not (owner, 13 ก.ย. 2026: "เอาเมาส์
	// ไปแตะมันขึ้นโหลดตลอด"). The arrow is the class's; over the figure the
	// hand says "you can hold this", as cursor: grab does in the app.
	idcArrow    = 32512
	idcHand     = 32649
	idcSizeNWSE = 32642

	// The body's thread runs above normal: its frames have a compositor
	// deadline every 16.7ms, and the app's other threads (the Go runtime's,
	// the webview's) have none.
	threadPriorityAboveNormal = 1

	// A press that moves less than this (logical px) before release is a
	// click (Companion.svelte CLICK_PX).
	companionClickPx = 4

	// The clock: frames while something is in motion, and while it only
	// breathes. Breathing is a 3.6s loop sampled at four phases; ten blended
	// frames a second between them is smooth, and the compositor's share of
	// each frame is what the idle cost mostly is.
	companionTimerID = 1
	// While the hand moves the body runs a render loop: the timer is set to
	// 1ms — WM_TIMER is the lowest-priority message, delivered only once the
	// queue holds no input, so every mouse move is seen first — and each
	// frame ends by waiting for the compositor (DwmFlush), which is what
	// paces it: one frame per vblank, drawn from wherever the hand is by
	// then. Neither the mouse's cadence (8ms, beating against a 16.7ms
	// screen) nor the timer's sets the pace. A frame posted to the queue by
	// the loop itself was tried first and starved the input entirely —
	// posted messages come before input (13 ก.ย. 2026: "เลื่อนไม่ได้ ติด ๆ").
	// Measured on the owner's drags: frames driven off the moves reached
	// the screen at p95 20ms.
	frameActiveMs = 1
	frameIdleMs   = 100

	// A blink comes every 4.4–7s (mascot.css --ms-blink, off the hue) and is
	// shut for blinkMs.
	blinkMinMs = 4400
	blinkMaxMs = 7000

	// QUERY_USER_NOTIFICATION_STATE: the shell's own word on whether the
	// user is watching something that should not be covered — a game, a
	// film, a presentation. Asked once a second; a topmost window over a
	// full-screen app is exactly the thing a desktop pet must not be.
	qunsBusy               = 2
	qunsRunningD3DFull     = 3
	qunsPresentationMode   = 4
	fullScreenCheckSeconds = 1
)

type bitmapInfoHeader struct {
	Size          uint32
	Width, Height int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPels, YPels  int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type blendFunction struct {
	BlendOp, BlendFlags, SourceConstantAlpha, AlphaFormat byte
}

type monitorInfo struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
}

type trackMouseEvent struct {
	Size      uint32
	Flags     uint32
	Hwnd      uintptr
	HoverTime uint32
}

// companionWindow is the body: one layered window and the thread that owns
// it. Every field past `gone` belongs to that thread.
type companionWindow struct {
	mu       sync.Mutex
	cmds     []func()
	threadID uint32
	hwnd     uintptr
	ready    chan error
	gone     chan struct{}

	// Where the frames come from — the store of whatever set the window
	// last baked (companion_sprites.go); nil until it has — and where the
	// user's doings go.
	sprites func(scale float64) spriteSource
	on      func(kind string, data map[string]any)

	// Where it is and how big, in physical pixels of whichever monitor it is
	// on; dpi is that monitor's, read at creation and on every WM_DPICHANGED.
	x, y, w, h int
	dpi        int
	comp       *composer
	text       *gdiText
	canvas     *image.RGBA
	// The part of the canvas on screen: the window is exactly this big and
	// sits at (x, y) + used.Min — a sprite and two buttons most of the time,
	// the bubble's width only while there is a bubble. Everything else in
	// the canvas is never handed to the compositor at all.
	used image.Rectangle

	// What the app window last said, and what the body adds to it.
	state CompanionState
	scene companionScene
	seen  int // the last Hop count acted on

	// A press in progress. The figure is what is dragged (walker), the
	// window follows it; grab is where in the figure the hand took hold.
	pressing bool
	moved    bool
	pressAt  winPoint
	grabX    int
	grabY    int
	walk     *walker
	warmed   int    // the heading key whose neighbours were last warmed
	button   string // "hide", "mute" or "grip" while one is held
	// A resize in progress: the figure's size when the grip was taken; and
	// the size asked for at opening.
	resizing bool
	size0    int
	size     int
	hover    bool
	tracking bool

	nextBlink time.Time
	shutUntil time.Time
	interval  int
	lastFrame time.Time
	// Frame timing over one drag, logged at its end: the numbers behind
	// "is it smooth" (owner, 13 ก.ย. 2026: "เอาตัวเลขก่อนและหลังให้ดู").
	stats frameStats

	// Hidden while the shell says the user is in a full-screen app; checked
	// at fullScreenCheckSeconds.
	hidden    bool
	nextCheck time.Time

	// The vertical blank of the monitor the figure is on, opened while the
	// loop runs at its active rate and reopened when the figure crosses to
	// another monitor; nil falls back to DwmFlush.
	vblank       *vblankWaiter
	vblankDevice string

	// The bitmap handed to UpdateLayeredWindow, kept between frames and
	// remade only when the shown region changes size — a DIB section and a
	// DC per frame were most of a frame's cost.
	dib companionDIB
}

type companionDIB struct {
	w, h int
	hbm  uintptr
	mem  uintptr
	bits []byte
}

// ensure has the DIB at w×h, remaking it if the size changed.
func (d *companionDIB) ensure(screen uintptr, w, h int) bool {
	if d.hbm != 0 && d.w == w && d.h == h {
		return true
	}
	d.release()
	bmi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(w),
		Height:   -int32(h), // top-down, like image.RGBA
		Planes:   1,
		BitCount: 32,
	}
	var bits unsafe.Pointer
	hbm, _, err := procCreateDIBSectionOv.Call(screen, uintptr(unsafe.Pointer(&bmi)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		debuglog.Msg("companion: CreateDIBSection failed: %v", err)
		return false
	}
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	if mem == 0 {
		procDeleteObject.Call(hbm)
		return false
	}
	procSelectObject.Call(mem, hbm)
	d.w, d.h, d.hbm, d.mem = w, h, hbm, mem
	d.bits = unsafe.Slice((*byte)(bits), w*h*4)
	return true
}

func (d *companionDIB) release() {
	if d.mem != 0 {
		procDeleteDC.Call(d.mem)
	}
	if d.hbm != 0 {
		procDeleteObject.Call(d.hbm)
	}
	*d = companionDIB{}
}

var (
	companionClassOnce sync.Once
	companionClass     *uint16

	// AETOX_COMPANION_PACING=off turns the pacing off — frames only from the
	// timer, no DwmFlush — so the numbers with and without can be read from
	// one build. A diagnostic, not a setting.
	companionPacing = os.Getenv("AETOX_COMPANION_PACING") != "off"
)

// frameStats is one drag's worth of frame timing.
type frameStats struct {
	on        bool
	intervals []float64 // ms between frames reaching the screen
	work      []float64 // ms to compose a frame
	present   []float64 // ms UpdateLayeredWindow took
	flush     []float64 // ms DwmFlush waited
	decodes   int       // PNGs decoded on this thread, mid-frame
	last      time.Time
}

func (f *frameStats) begin(now time.Time) {
	*f = frameStats{on: true, last: now}
}

func (f *frameStats) frame(now time.Time, work, present, flush time.Duration) {
	if !f.on {
		return
	}
	if !f.last.IsZero() {
		f.intervals = append(f.intervals, float64(now.Sub(f.last).Microseconds())/1000)
	}
	f.last = now
	f.work = append(f.work, float64(work.Microseconds())/1000)
	f.present = append(f.present, float64(present.Microseconds())/1000)
	f.flush = append(f.flush, float64(flush.Microseconds())/1000)
}

// summary is "n frames, interval mean/p95/max ms, work mean/max ms".
func (f *frameStats) summary() string {
	f.on = false
	if len(f.intervals) == 0 {
		return "no frames"
	}
	pct := func(v []float64, p float64) float64 {
		c := append([]float64(nil), v...)
		sort.Float64s(c)
		return c[min(len(c)-1, int(float64(len(c))*p))]
	}
	mean := func(v []float64) float64 {
		t := 0.0
		for _, x := range v {
			t += x
		}
		return t / float64(len(v))
	}
	over := 0
	for _, x := range f.intervals {
		if x > 25 {
			over++
		}
	}
	return fmt.Sprintf("%d frames · interval mean %.1f p95 %.1f max %.1f ms (%d gaps >25ms) · compose mean %.2f max %.2f · present mean %.2f max %.2f · flush mean %.1f max %.1f ms · mid-frame decodes %d",
		len(f.intervals)+1, mean(f.intervals), pct(f.intervals, 0.95), pct(f.intervals, 1), over,
		mean(f.work), pct(f.work, 1), mean(f.present), pct(f.present, 1), mean(f.flush), pct(f.flush, 1), f.decodes)
}

func openCompanionBody(x, y, size int, sprites func(scale float64) spriteSource, on func(kind string, data map[string]any)) (companionBody, error) {
	w, err := openCompanionWindow(x, y, size, sprites, on)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// openCompanionWindow brings the body up with its figure's top-left at
// (x, y) physical pixels — a spot the caller remembered, or a negative pair
// meaning "you choose". Returns once the window exists or could not be
// created.
func openCompanionWindow(x, y, size int, sprites func(scale float64) spriteSource, on func(kind string, data map[string]any)) (*companionWindow, error) {
	if sprites == nil {
		sprites = func(float64) spriteSource { return noSprites{} }
	}
	if on == nil {
		on = func(string, map[string]any) {}
	}
	w := &companionWindow{ready: make(chan error, 1), gone: make(chan struct{}), x: x, y: y, size: size, sprites: sprites, on: on}
	go w.run()
	if err := <-w.ready; err != nil {
		return nil, err
	}
	return w, nil
}

// do queues fn onto the window's thread and wakes its pump. Asynchronous.
func (w *companionWindow) do(fn func()) {
	w.mu.Lock()
	w.cmds = append(w.cmds, fn)
	tid := w.threadID
	w.mu.Unlock()
	if tid != 0 {
		procPostThreadMsgW.Call(uintptr(tid), wmApp, 0, 0)
	}
}

func (w *companionWindow) drain() {
	for {
		w.mu.Lock()
		if len(w.cmds) == 0 {
			w.mu.Unlock()
			return
		}
		fn := w.cmds[0]
		w.cmds = w.cmds[1:]
		w.mu.Unlock()
		fn()
	}
}

// close destroys the window and waits for its thread to end.
func (w *companionWindow) close() {
	w.do(func() {
		if w.hwnd != 0 {
			procDestroyWindow.Call(w.hwnd)
		}
	})
	<-w.gone
}

// apply is the app window's report: what to be and what to say. A Hop count
// past the last one seen is a click's reaction; a Size other than the
// figure's resizes it.
func (w *companionWindow) apply(s CompanionState) {
	w.do(func() {
		w.state = s
		if s.Hop > w.seen {
			w.seen = s.Hop
			w.scene.HopAt = time.Now()
		}
		if !w.resizing {
			w.setFigure(s.Size, true)
		}
		w.frame()
	})
}

// setFigure sizes the figure about its top-left: the canvas is remade for
// it, and — once the size is settled — the app window is asked for frames
// baked at it; until they come the ones at hand are resampled.
func (w *companionWindow) setFigure(size int, bake bool) {
	if size <= 0 {
		size = cFigureDefault
	}
	if size == w.comp.figure {
		return
	}
	fx, fy := w.figureAt()
	w.comp.setFigure(size)
	w.w, w.h = w.comp.canvasSize()
	w.canvas = image.NewRGBA(image.Rect(0, 0, w.w, w.h))
	f := w.comp.figureRect()
	w.x, w.y = fx-f.Min.X, fy-f.Min.Y
	if bake {
		w.on("bake", map[string]any{"scale": w.comp.spriteScale()})
	}
}

// figureAt is where the figure's top-left is on the screen.
func (w *companionWindow) figureAt() (int, int) {
	f := w.comp.figureRect()
	return w.x + f.Min.X, w.y + f.Min.Y
}

// placeFigure puts the figure's top-left at (fx, fy), keeping it on a
// monitor, and moves the window to suit.
func (w *companionWindow) placeFigure(fx, fy int) {
	f := w.comp.figureRect()
	fx, fy = clampFigure(fx, fy, f.Dx(), f.Dy())
	// The window itself moves with the next present, which places it.
	w.x, w.y = fx-f.Min.X, fy-f.Min.Y
}

func (w *companionWindow) run() {
	runtime.LockOSThread()
	defer close(w.gone)
	procSetThreadDpiAwarenessCtx.Call(dpiAwarenessPerMonitorV2)

	tid, _, _ := procGetCurrentThreadID.Call()
	w.mu.Lock()
	w.threadID = uint32(tid)
	w.mu.Unlock()

	if h, _, _ := procGetCurrentThread.Call(); h != 0 {
		procSetThreadPriority.Call(h, threadPriorityAboveNormal)
	}
	companionClassOnce.Do(func() {
		name, _ := syscall.UTF16PtrFromString("AetoxCompanion")
		arrow, _, _ := procLoadCursorW.Call(0, idcArrow)
		wc := wndClassExW{
			Size:      uint32(unsafe.Sizeof(wndClassExW{})),
			WndProc:   syscall.NewCallback(companionWndProc),
			ClassName: name,
			Cursor:    arrow,
		}
		if atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
			debuglog.Msg("companion: RegisterClassExW failed: %v", err)
			return
		}
		companionClass = name
	})
	if companionClass == nil {
		w.ready <- errors.New("companion: window class not registered")
		return
	}

	w.text = newGDIText()
	// Created at the primary monitor's scale; the read below corrects it
	// before anything is drawn. The figure's size is the caller's from the
	// start, so the first frame and the first bake are at it.
	w.setScale(96)
	w.comp.setFigure(w.size)
	w.w, w.h = w.comp.canvasSize()
	w.canvas = image.NewRGBA(image.Rect(0, 0, w.w, w.h))
	fx, fy := w.x, w.y
	chosen := fx < 0 || fy < 0
	if chosen {
		fx, fy = defaultFigureSpot(w.comp)
	}
	f := w.comp.figureRect()
	w.x, w.y = fx-f.Min.X, fy-f.Min.Y
	hwnd, _, err := procCreateWindowExW.Call(
		wsExLayered|wsExToolWindow|wsExNoActivate|wsExTopmost,
		uintptr(unsafe.Pointer(companionClass)),
		0,
		wsPopup,
		uintptr(w.x), uintptr(w.y), uintptr(w.w), uintptr(w.h),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		w.ready <- errors.New("companion: CreateWindowExW: " + err.Error())
		return
	}
	w.hwnd = hwnd
	companionWindows.remember(hwnd, w)
	defer companionWindows.forget(hwnd)

	if dpi, _, _ := procGetDpiForWindow.Call(hwnd); dpi != 0 && int(dpi) != w.dpi {
		w.setScale(int(dpi))
		if chosen {
			// The corner is measured in the figure's own size, which was
			// not known until now.
			fx, fy = defaultFigureSpot(w.comp)
		}
		w.placeFigure(fx, fy)
	}
	w.nextBlink = time.Now().Add(blinkWait())
	w.on("bake", map[string]any{"scale": w.comp.spriteScale()})
	w.frame()
	procShowWindow.Call(hwnd, swShowNoActivate)
	w.setInterval(frameIdleMs)
	debuglog.Msg("companion: window %#x figure at %d,%d canvas %dx%d dpi=%d", hwnd, fx, fy, w.w, w.h, w.dpi)
	w.ready <- nil

	var msg winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			return
		}
		w.drain()
		if msg.Message != wmApp {
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

// setScale sizes the canvas for a monitor's dpi: the composer's logical
// layout at that scale, and a canvas to draw it on.
func (w *companionWindow) setScale(dpi int) {
	w.dpi = dpi
	if w.comp == nil {
		w.comp = newComposer(w.sprites(float64(dpi)/96), w.text, float64(dpi)/96)
	} else {
		w.comp.rescale(float64(dpi) / 96)
	}
	w.w, w.h = w.comp.canvasSize()
	w.canvas = image.NewRGBA(image.Rect(0, 0, w.w, w.h))
}

// defaultFigureSpot is bottom-right of the primary monitor's work area, the
// corner the eye rests in last (Companion.svelte seed).
func defaultFigureSpot(c *composer) (int, int) {
	f := c.figureRect()
	mi, ok := monitorNear(0, 0)
	if !ok {
		return 200, 200
	}
	margin := c.px(28)
	return int(mi.Work.Right) - f.Dx() - margin, int(mi.Work.Bottom) - f.Dy() - margin
}

func (w *companionWindow) setInterval(ms int) {
	if w.interval == ms {
		return
	}
	if ms == frameActiveMs {
		procTimeBeginPeriod.Call(1)
	} else if w.interval == frameActiveMs {
		procTimeEndPeriod.Call(1)
	}
	w.interval = ms
	if ms == frameActiveMs && !companionPacing {
		ms = 16 // no compositor wait to pace it: the timer alone does
	}
	procSetTimer.Call(w.hwnd, companionTimerID, uintptr(ms), 0)
}

func blinkWait() time.Duration {
	return time.Duration(blinkMinMs+rand.IntN(blinkMaxMs-blinkMinMs)) * time.Millisecond
}

// The class procedure is shared by every window of the class and is handed an
// HWND, so this index is how a message finds its window.
type companionIndex struct {
	mu sync.Mutex
	m  map[uintptr]*companionWindow
}

var companionWindows = companionIndex{m: map[uintptr]*companionWindow{}}

func (r *companionIndex) remember(hwnd uintptr, w *companionWindow) {
	r.mu.Lock()
	r.m[hwnd] = w
	r.mu.Unlock()
}

func (r *companionIndex) forget(hwnd uintptr) {
	r.mu.Lock()
	delete(r.m, hwnd)
	r.mu.Unlock()
}

func (r *companionIndex) get(hwnd uintptr) *companionWindow {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.m[hwnd]
}

func companionWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	// A fault in the body must not take the app with it: this procedure is
	// the only Go frame under the Windows call, so this is where a panic is
	// caught, logged, and turned into "this message did nothing".
	defer func() {
		if r := recover(); r != nil {
			debuglog.Msg("companion: recovered in message %#x: %v", msg, r)
		}
	}()
	w := companionWindows.get(hwnd)
	if w != nil {
		switch msg {
		case wmTimer:
			w.frame()
			return 0
		case wmLButtonDown:
			w.pressBegin()
			return 0
		case wmMouseMove:
			w.mouseMove()
			return 0
		case wmLButtonUp:
			w.pressEnd()
			return 0
		case wmSetCursor:
			// The hand over the figure, the corner arrows over the grip, the
			// arrow over the buttons.
			c := cursorPos()
			switch w.buttonAt(c) {
			case "":
				if !w.onFigure(c) {
					break
				}
				if hand, _, _ := procLoadCursorW.Call(0, idcHand); hand != 0 {
					procSetCursor.Call(hand)
					return 1
				}
			case "grip":
				if cur, _, _ := procLoadCursorW.Call(0, idcSizeNWSE); cur != 0 {
					procSetCursor.Call(cur)
					return 1
				}
			}
		case wmMouseLeave:
			w.tracking = false
			if !w.pressing {
				w.hover = false
				w.frame()
			}
			return 0
		case wmCaptureChanged:
			if w.pressing {
				w.pressing = false
				w.walk = nil
			}
			if w.resizing {
				w.resizing = false
				w.button = ""
			}
			return 0
		case wmDPIChanged:
			w.dpiChanged(int(wparam >> 16 & 0xffff))
			return 0
		case wmDestroy:
			procKillTimer.Call(hwnd, companionTimerID)
			if w.vblank != nil {
				w.vblank.close()
				w.vblank = nil
			}
			if w.interval == frameActiveMs {
				procTimeEndPeriod.Call(1)
			}
			w.dib.release()
			procPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return r
}

// dpiChanged is the window crossing to a monitor of another scale. The
// canvas is rebuilt at the new scale around the figure's spot — mid-drag,
// around the hand — and the app window is asked for frames at that scale.
// lParam's suggested rect is not read: reading a uintptr as a pointer is
// what vet forbids, and the figure's own spot is a better anchor anyway.
func (w *companionWindow) dpiChanged(dpi int) {
	if dpi == 0 || dpi == w.dpi {
		return
	}
	old := w.dpi
	fx, fy := w.figureAt()
	w.setScale(dpi)
	if w.pressing && w.walk != nil {
		w.grabX = w.grabX * dpi / old
		w.grabY = w.grabY * dpi / old
		w.walk.scale = float64(dpi) / 96
		c := cursorPos()
		fx, fy = int(c.X)-w.grabX, int(c.Y)-w.grabY
		w.walk.x, w.walk.y = float64(fx), float64(fy)
		w.walk.targetX, w.walk.targetY = float64(fx), float64(fy)
	}
	f := w.comp.figureRect()
	w.x, w.y = fx-f.Min.X, fy-f.Min.Y
	w.frame()
	w.on("bake", map[string]any{"scale": w.comp.spriteScale()})
	debuglog.Msg("companion: dpi now %d → canvas %dx%d, figure at %d,%d", dpi, w.w, w.h, fx, fy)
}

func cursorPos() winPoint {
	var p winPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

// onFigure is whether the screen point is on the figure's own pixels — the
// drawn ones, not the frame's touchable ground around them.
func (w *companionWindow) onFigure(p winPoint) bool {
	q := image.Pt(int(p.X)-w.x, int(p.Y)-w.y)
	if !q.In(w.comp.spriteRect()) || !q.In(w.canvas.Bounds()) {
		return false
	}
	return w.canvas.Pix[w.canvas.PixOffset(q.X, q.Y)+3] > 1
}

// buttonAt is which control the screen point is on, if any: a button, or
// the corner grip.
func (w *companionWindow) buttonAt(p winPoint) string {
	hide, mute := w.comp.buttonRects()
	q := image.Pt(int(p.X)-w.x, int(p.Y)-w.y)
	switch {
	case w.hover && q.In(hide):
		return "hide"
	case (w.hover || w.state.Muted) && q.In(mute):
		return "mute"
	case w.hover && q.In(w.comp.gripRect()):
		return "grip"
	}
	return ""
}

func (w *companionWindow) pressBegin() {
	c := cursorPos()
	if !w.onFigure(c) && w.buttonAt(c) == "" {
		// The frame's ground: hoverable, not holdable (the app's .grab is
		// the figure alone).
		return
	}
	procSetCapture.Call(w.hwnd)
	if b := w.buttonAt(c); b != "" {
		w.button = b
		if b == "grip" {
			w.resizing = true
			w.pressAt = c
			w.size0 = w.comp.figure
			w.setInterval(frameActiveMs)
		}
		return
	}
	w.pressing = true
	w.moved = false
	w.pressAt = c
	fx, fy := w.figureAt()
	w.grabX, w.grabY = int(c.X)-fx, int(c.Y)-fy
	// Start walking from where the head already is: the pose's rest turn.
	w.walk = newWalker(float64(fx), float64(fy), restTurn(w.state.Pose), float64(w.dpi)/96, time.Now())
}

func (w *companionWindow) mouseMove() {
	c := cursorPos()
	if !w.tracking {
		tme := trackMouseEvent{Size: uint32(unsafe.Sizeof(trackMouseEvent{})), Flags: tmeLeave, Hwnd: w.hwnd}
		procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
		w.tracking = true
	}
	if !w.hover {
		w.hover = true
		w.frame()
	}
	if w.resizing {
		// The corner follows the hand: the larger of the two deltas, in
		// logical pixels, added to the size it started at.
		d := max(int(c.X-w.pressAt.X), int(c.Y-w.pressAt.Y))
		w.setFigure(w.size0+int(float64(d)*96/float64(w.dpi)), false)
		return
	}
	if !w.pressing || w.walk == nil {
		return
	}
	dx, dy := int(c.X-w.pressAt.X), int(c.Y-w.pressAt.Y)
	click := w.comp.px(companionClickPx)
	if !w.moved && abs(dx) < click && abs(dy) < click {
		return
	}
	if !w.moved {
		w.moved = true
		w.on("dragStart", nil)
		w.setInterval(frameActiveMs)
		w.warmWalk(w.walk.heading)
		w.warmed = walkHeadingKey(w.walk.heading, walkStep)
		w.stats.begin(time.Now())
	}
	// The hand's spot for the figure, kept on the desktop. At an edge the
	// grip is re-anchored to the clamped spot, so the way back starts the
	// moment the hand turns round (Companion.svelte onMove).
	f := w.comp.figureRect()
	tx, ty := clampFigure(int(c.X)-w.grabX, int(c.Y)-w.grabY, f.Dx(), f.Dy())
	w.grabX, w.grabY = int(c.X)-tx, int(c.Y)-ty
	w.walk.move(float64(tx), float64(ty), time.Now())
}

func (w *companionWindow) pressEnd() {
	if w.button != "" {
		b := w.button
		w.button = ""
		procReleaseCapture.Call()
		if b == "grip" {
			w.resizing = false
			w.on("resize", map[string]any{"size": w.comp.figure})
			w.on("bake", map[string]any{"scale": w.comp.spriteScale()})
			w.frame()
			return
		}
		if w.buttonAt(cursorPos()) == b {
			w.on(b, nil)
		}
		return
	}
	if !w.pressing {
		return
	}
	w.pressing = false
	procReleaseCapture.Call()
	if w.moved && w.walk != nil {
		fx, fy, _ := w.walk.end()
		w.placeFigure(int(fx), int(fy))
		w.walk = nil
		w.scene.Walking = false
		debuglog.Msg("companion: drag %s (pacing %v, dpi %d, %s vblank=%v)", w.stats.summary(), companionPacing, w.dpi, w.vblankDevice, w.vblank != nil)
		w.on("dragEnd", nil)
		x, y := w.figureAt()
		w.on("moved", map[string]any{"x": x, "y": y})
		w.frame()
		return
	}
	w.walk = nil
	w.on("click", nil)
	w.scene.HopAt = time.Now()
	w.frame()
}

// decodesSoFar is how many PNGs the current sources have read from disk.
func (w *companionWindow) decodesSoFar() int {
	n := 0
	var count func(src spriteSource)
	count = func(src spriteSource) {
		switch v := src.(type) {
		case *spriteStore:
			n += v.decoded()
		case chainedSprites:
			for _, c := range v {
				count(c)
			}
		}
	}
	count(w.sprites(w.comp.spriteScale()))
	return n
}

// warmWalk has the walk's frames near a heading decoded before a step needs
// them: this heading and the two either side, every phase — 40 frames, the
// most a turn can reach within a few frames (walkTurnDegS). Called at the
// first step and again whenever the walk crosses into another heading key,
// so the frames a turn is about to need are being read while the current
// ones draw. Asynchronous; the decode runs outside the store's lock.
func (w *companionWindow) warmWalk(heading float64) {
	src := w.sprites(w.comp.spriteScale())
	ws, ok := src.(interface{ warm([]string) })
	if !ok {
		if chain, isChain := src.(chainedSprites); isChain && len(chain) > 0 {
			ws, ok = chain[0].(interface{ warm([]string) })
		}
	}
	if !ok {
		return
	}
	h := walkHeadingKey(heading, walkStep)
	keys := make([]string, 0, 5*walkPhases)
	for _, d := range []int{0, walkStep, -walkStep, 2 * walkStep, -2 * walkStep} {
		hk := walkHeadingKey(float64(h+d), walkStep)
		for p := 0; p < walkPhases; p++ {
			keys = append(keys, walkKey(hk, p))
		}
	}
	ws.warm(keys)
}

// restTurn is the pose's own head turn (poses.ts), for the walk to start
// from; unknown poses face front.
func restTurn(pose string) float64 {
	switch pose {
	case "idle", "typing":
		return 14
	case "greeting", "wink":
		return 12
	case "thinking":
		return -12
	case "reading", "coding", "wake":
		return 6
	case "research", "answering", "debugging", "presenting":
		return 10
	case "asking", "planning":
		return -6
	case "helping":
		return 8
	case "listening", "error":
		return -8
	case "recharge":
		return 18
	}
	return 0
}

// frame is one tick: the drag advanced, the blink scheduled, the scene
// composed and shown.
func (w *companionWindow) frame() {
	if w.hwnd == 0 || w.comp == nil {
		return
	}
	now := time.Now()
	w.lastFrame = now
	if now.After(w.nextCheck) {
		w.nextCheck = now.Add(fullScreenCheckSeconds * time.Second)
		if covered := userIsFullScreen(); covered != w.hidden {
			w.hidden = covered
			if covered {
				procShowWindow.Call(w.hwnd, swHide)
			} else {
				procShowWindow.Call(w.hwnd, swShowNoActivate)
			}
		}
	}
	if w.hidden {
		return
	}
	w.comp.setSprites(w.sprites(w.comp.spriteScale()))

	if w.pressing && w.walk != nil && w.moved {
		fx, fy, moved := w.walk.tick(now)
		if moved {
			w.placeFigure(int(fx), int(fy))
		}
		w.scene.Walking = w.walk.moving
		w.scene.Heading = w.walk.heading
		if hk := walkHeadingKey(w.walk.heading, walkStep); hk != w.warmed {
			w.warmed = hk
			w.warmWalk(w.walk.heading)
		}
	}

	if now.After(w.nextBlink) {
		w.shutUntil = now.Add(blinkMs * time.Millisecond)
		w.nextBlink = now.Add(blinkWait())
	}
	s := w.state
	w.scene.Pose = s.Pose
	w.scene.Theme = s.Theme
	w.scene.Shown = s.Shown
	w.scene.Words = s.Words
	w.scene.Cursor = s.Cursor
	w.scene.Muted = s.Muted
	// The frame and its buttons are for holding, not for the way there:
	// hidden while dragging (Companion.svelte .dragging .frame).
	w.scene.Hover = (w.hover && !(w.pressing && w.moved)) || w.resizing
	w.scene.Shut = now.Before(w.shutUntil)
	w.scene.Flip = w.bubbleFlips()
	if w.scene.Pose == "" {
		w.scene.Pose = "idle"
	}

	t0 := time.Now()
	before := w.decodesSoFar()
	used := w.comp.draw(w.canvas, w.scene, now)
	t1 := time.Now()
	w.stats.decodes += w.decodesSoFar() - before
	w.present(w.canvas, used)
	t2 := time.Now()
	// Whenever the loop runs at its active rate — a drag, a hop, a cursor,
	// a crossfade — the wait for the monitor's blank is what paces it (the
	// compositor's frame if the monitor's blank cannot be had); without a
	// wait the 1ms timer would draw a thousand frames a second.
	if companionPacing && w.interval == frameActiveMs {
		if v := w.vblankFor(); v == nil || !v.wait() {
			procDwmFlush.Call()
		}
	}
	// The interval is measured after the flush — when the frame reached
	// the screen — which is what the eye sees, not when it was started.
	w.stats.frame(time.Now(), t1.Sub(t0), t2.Sub(t1), time.Since(t2))

	active := w.pressing || w.resizing || s.Cursor || now.Sub(w.scene.HopAt) < hopMs*time.Millisecond || now.Sub(w.comp.crossAt) < crossMs*time.Millisecond
	if active {
		w.setInterval(frameActiveMs)
	} else {
		w.setInterval(frameIdleMs)
	}
}

// vblankFor is the waiter for the monitor under the figure's centre,
// reopened when that monitor changes.
func (w *companionWindow) vblankFor() *vblankWaiter {
	fx, fy := w.figureAt()
	f := w.comp.figureRect()
	mi := monitorInfoEx{}
	mi.Size = uint32(unsafe.Sizeof(mi))
	mon, _, _ := procMonitorFromPoint.Call(packPoint(fx+f.Dx()/2, fy+f.Dy()/2), monitorDefaultToNearest)
	if mon == 0 {
		return w.vblank
	}
	if ok, _, _ := procGetMonitorInfoExW.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return w.vblank
	}
	device := syscall.UTF16ToString(mi.Device[:])
	if w.vblank != nil && w.vblankDevice == device {
		return w.vblank
	}
	if w.vblank != nil {
		w.vblank.close()
	}
	w.vblank = openVBlank(fx+f.Dx()/2, fy+f.Dy()/2)
	w.vblankDevice = device
	if w.vblank == nil {
		debuglog.Msg("companion: no vertical blank for %s; pacing by the compositor", device)
	}
	return w.vblank
}

// userIsFullScreen is whether the shell would hold back a notification right
// now: a full-screen game or film, or a presentation.
func userIsFullScreen() bool {
	var state uint32
	if hr, _, _ := procSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&state))); hr != 0 {
		return false
	}
	return state == qunsBusy || state == qunsRunningD3DFull || state == qunsPresentationMode
}

// bubbleFlips says whether the bubble goes on the right: when the card would
// not fit on the left of the figure within its monitor's work area.
func (w *companionWindow) bubbleFlips() bool {
	fx, fy := w.figureAt()
	f := w.comp.figureRect()
	mi, ok := monitorNear(fx+f.Dx()/2, fy+f.Dy()/2)
	if !ok {
		return false
	}
	return fx-w.comp.px(cBubbleMaxW+cBubbleGap) < int(mi.Work.Left)
}

// clampFigure keeps the figure — the part that is not transparent — on the
// desktop. Only the figure, on purpose: the canvas is wider than the figure
// so the bubble has room, and a canvas forced whole onto one monitor could
// never straddle the edge between two, which is what a drag from one to the
// other does halfway.
//
// "On the desktop" means all the monitors together. The figure is held
// inside the work area (the desktop less the taskbar) of the monitor its
// centre is nearest, but only at the edges where the desktop ends: an edge
// with another monitor beyond it is left open, or the centre could never
// reach that monitor and the figure would be stuck at the seam. Once the
// centre is across, the other monitor's own edges take over. SPI_GETWORKAREA
// (window_windows.go) only knows the primary; this asks per monitor.
func clampFigure(x, y, fw, fh int) (int, int) {
	mi, ok := monitorNear(x+fw/2, y+fh/2)
	if !ok {
		return x, y
	}
	m := mi.Monitor
	cy, cx := int(m.Top+m.Bottom)/2, int(m.Left+m.Right)/2
	open := [4]bool{
		monitorAt(int(m.Left)-1, cy), // left
		monitorAt(cx, int(m.Top)-1),  // top
		monitorAt(int(m.Right), cy),  // right
		monitorAt(cx, int(m.Bottom)), // bottom
	}
	return clampRect(x, y, fw, fh, mi.Work, open)
}

func monitorNear(x, y int) (monitorInfo, bool) {
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	mon, _, _ := procMonitorFromPoint.Call(packPoint(x, y), monitorDefaultToNearest)
	if mon == 0 {
		return mi, false
	}
	if ok, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return mi, false
	}
	return mi, true
}

// monitorAt reports whether any monitor covers the point.
func monitorAt(x, y int) bool {
	mon, _, _ := procMonitorFromPoint.Call(packPoint(x, y), monitorDefaultToNull)
	return mon != 0
}

// packPoint is a POINT passed by value: two int32 in one 64-bit argument.
func packPoint(x, y int) uintptr {
	return uintptr(uint32(int32(x))) | uintptr(uint32(int32(y)))<<32
}

// clampRect moves a w×h rect at (x, y) the least distance that puts it
// inside area — on the edges that are not open. open is left, top, right,
// bottom.
func clampRect(x, y, w, h int, area winRect, open [4]bool) (int, int) {
	if !open[0] {
		x = max(x, int(area.Left))
	}
	if !open[2] {
		x = min(x, int(area.Right)-w)
	}
	if !open[1] {
		y = max(y, int(area.Top))
	}
	if !open[3] {
		y = min(y, int(area.Bottom)-h)
	}
	return x, y
}

// present shows the used part of an RGBA frame (premultiplied, as image.RGBA
// is) as the whole window: those rows are copied into the DIB — the one
// bitmap format UpdateLayeredWindow takes — as BGRA, and the window is
// moved and sized to the region in the same call.
func (w *companionWindow) present(img *image.RGBA, used image.Rectangle) {
	used = used.Intersect(img.Bounds())
	if used.Empty() {
		// Nothing to show yet (no frames baked): a single clear pixel keeps
		// the window alive and invisible.
		used = image.Rect(0, 0, 1, 1)
	}
	width, height := used.Dx(), used.Dy()
	w.used = used
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	if !w.dib.ensure(screen, width, height) {
		return
	}
	dst := w.dib.bits
	for y := 0; y < height; y++ {
		i := img.PixOffset(used.Min.X, used.Min.Y+y)
		row := img.Pix[i : i+width*4]
		out := dst[y*width*4 : (y+1)*width*4]
		for k := 0; k < width*4; k += 4 {
			out[k], out[k+1], out[k+2], out[k+3] = row[k+2], row[k+1], row[k], row[k+3]
		}
	}
	at := winPoint{int32(w.x + used.Min.X), int32(w.y + used.Min.Y)}
	size := winPoint{int32(width), int32(height)}
	src := winPoint{}
	blend := blendFunction{BlendOp: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: acSrcAlpha}
	if ok, _, err := procUpdateLayeredWindow.Call(w.hwnd, 0, uintptr(unsafe.Pointer(&at)), uintptr(unsafe.Pointer(&size)), w.dib.mem, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&blend)), ulwAlpha); ok == 0 {
		debuglog.Msg("companion: UpdateLayeredWindow failed: %v", err)
	}
}
