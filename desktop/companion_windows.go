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
	"image"
	"math/rand/v2"
	"runtime"
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

	procSHQueryUserNotificationState = shell32.NewProc("SHQueryUserNotificationState")

	// Windows' timers tick at 15.6ms unless asked for better; a walk paced
	// by them is not smooth. Asked for 1ms only while something moves, and
	// given back when it stops — it costs power to keep.
	winmm               = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod   = winmm.NewProc("timeEndPeriod")
)

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

	// A press that moves less than this (logical px) before release is a
	// click (Companion.svelte CLICK_PX).
	companionClickPx = 4

	// The clock: frames while something is in motion, and while it only
	// breathes. Breathing is a 3.6s loop sampled at four phases; ten blended
	// frames a second between them is smooth, and the compositor's share of
	// each frame is what the idle cost mostly is.
	companionTimerID = 1
	frameActiveMs    = 16
	frameIdleMs      = 100
	// While the hand moves, frames are driven off the mouse messages
	// themselves (no faster than this) rather than the timer: WM_TIMER is
	// posted behind everything else in the queue, and a fast drag floods
	// the queue with moves, so a timer-paced walk stutters exactly when it
	// is looked at most closely.
	frameMinMs = 12

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
	sprites func() spriteSource
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
	button   string // "hide" or "mute" while a button is held
	hover    bool
	tracking bool

	nextBlink time.Time
	shutUntil time.Time
	interval  int
	lastFrame time.Time

	// Hidden while the shell says the user is in a full-screen app; checked
	// at fullScreenCheckSeconds.
	hidden    bool
	nextCheck time.Time

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
)

func openCompanionBody(x, y int, sprites func() spriteSource, on func(kind string, data map[string]any)) (companionBody, error) {
	w, err := openCompanionWindow(x, y, sprites, on)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// openCompanionWindow brings the body up with its figure's top-left at
// (x, y) physical pixels — a spot the caller remembered, or a negative pair
// meaning "you choose". Returns once the window exists or could not be
// created.
func openCompanionWindow(x, y int, sprites func() spriteSource, on func(kind string, data map[string]any)) (*companionWindow, error) {
	if sprites == nil {
		sprites = func() spriteSource { return nil }
	}
	if on == nil {
		on = func(string, map[string]any) {}
	}
	w := &companionWindow{ready: make(chan error, 1), gone: make(chan struct{}), x: x, y: y, sprites: sprites, on: on}
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
// past the last one seen is a click's reaction.
func (w *companionWindow) apply(s CompanionState) {
	w.do(func() {
		w.state = s
		if s.Hop > w.seen {
			w.seen = s.Hop
			w.scene.HopAt = time.Now()
		}
		w.frame()
	})
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

	companionClassOnce.Do(func() {
		name, _ := syscall.UTF16PtrFromString("AetoxCompanion")
		wc := wndClassExW{
			Size:      uint32(unsafe.Sizeof(wndClassExW{})),
			WndProc:   syscall.NewCallback(companionWndProc),
			ClassName: name,
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
	// before anything is drawn.
	w.setScale(96)
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
	w.on("bake", map[string]any{"scale": float64(w.dpi) / 96})
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
		w.comp = newComposer(w.sprites(), w.text, float64(dpi)/96)
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
			// A frame just drawn off a mouse move is not drawn again.
			if time.Since(w.lastFrame) >= frameMinMs*time.Millisecond {
				w.frame()
			}
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
			return 0
		case wmDPIChanged:
			w.dpiChanged(int(wparam >> 16 & 0xffff))
			return 0
		case wmDestroy:
			procKillTimer.Call(hwnd, companionTimerID)
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
	w.on("bake", map[string]any{"scale": float64(dpi) / 96})
	debuglog.Msg("companion: dpi now %d → canvas %dx%d, figure at %d,%d", dpi, w.w, w.h, fx, fy)
}

func cursorPos() winPoint {
	var p winPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

// buttonAt is which button the screen point is on, if any.
func (w *companionWindow) buttonAt(p winPoint) string {
	hide, mute := w.comp.buttonRects()
	q := image.Pt(int(p.X)-w.x, int(p.Y)-w.y)
	switch {
	case w.hover && q.In(hide):
		return "hide"
	case (w.hover || w.state.Muted) && q.In(mute):
		return "mute"
	}
	return ""
}

func (w *companionWindow) pressBegin() {
	c := cursorPos()
	debuglog.Msg("companion: press at %d,%d (hover=%v)", c.X, c.Y, w.hover)
	procSetCapture.Call(w.hwnd)
	if b := w.buttonAt(c); b != "" {
		w.button = b
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
	}
	// The hand's spot for the figure, kept on the desktop. At an edge the
	// grip is re-anchored to the clamped spot, so the way back starts the
	// moment the hand turns round (Companion.svelte onMove).
	f := w.comp.figureRect()
	tx, ty := clampFigure(int(c.X)-w.grabX, int(c.Y)-w.grabY, f.Dx(), f.Dy())
	w.grabX, w.grabY = int(c.X)-tx, int(c.Y)-ty
	now := time.Now()
	w.walk.move(float64(tx), float64(ty), now)
	if now.Sub(w.lastFrame) >= frameMinMs*time.Millisecond {
		w.frame()
	}
}

func (w *companionWindow) pressEnd() {
	debuglog.Msg("companion: release (pressing=%v moved=%v button=%q)", w.pressing, w.moved, w.button)
	if w.button != "" {
		b := w.button
		w.button = ""
		procReleaseCapture.Call()
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
	w.comp.setSprites(w.sprites())

	if w.pressing && w.walk != nil && w.moved {
		fx, fy, moved := w.walk.tick(now)
		if moved {
			w.placeFigure(int(fx), int(fy))
		}
		w.scene.Walking = w.walk.moving
		w.scene.Heading = w.walk.heading
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
	w.scene.Hover = w.hover && !(w.pressing && w.moved)
	w.scene.Shut = now.Before(w.shutUntil)
	w.scene.Flip = w.bubbleFlips()
	if w.scene.Pose == "" {
		w.scene.Pose = "idle"
	}

	used := w.comp.draw(w.canvas, w.scene, now)
	w.present(w.canvas, used)

	active := w.pressing || s.Cursor || now.Sub(w.scene.HopAt) < hopMs*time.Millisecond || now.Sub(w.comp.crossAt) < crossMs*time.Millisecond
	if active {
		w.setInterval(frameActiveMs)
	} else {
		w.setInterval(frameIdleMs)
	}
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
