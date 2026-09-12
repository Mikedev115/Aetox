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
// This file owns the window and the mouse. What the window SHOWS is composed
// elsewhere (companion_draw.go, to come): this file only asks for a frame and
// hands it to Windows. Until then it draws a placeholder — the proof that the
// window itself works, which is the risk this file retires.

import (
	"errors"
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
	"syscall"
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
)

const (
	wmDestroy        = 0x0002
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmCaptureChanged = 0x0215
	wmDPIChanged     = 0x02E0

	swpNoZOrder = 0x0004

	// Per-monitor v2: the window is told (WM_DPICHANGED) when it is dragged
	// onto a monitor with a different scale, and nothing is stretched for it.
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is the handle value -4.
	dpiAwarenessPerMonitorV2 = ^uintptr(3)

	monitorDefaultToNull    = 0
	monitorDefaultToNearest = 2

	// A press that moves less than this before release is a click.
	companionClickPx = 4
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

// companionWindow is the body: one layered window and the thread that owns
// it. Every field past `mu` belongs to that thread.
type companionWindow struct {
	mu       sync.Mutex
	cmds     []func()
	threadID uint32
	hwnd     uintptr
	ready    chan error
	gone     chan struct{}

	// Where it is and how big, in physical pixels of whichever monitor it is
	// on; dpi is that monitor's, read at creation and on every WM_DPICHANGED.
	x, y, w, h int
	dpi        int

	// A press in progress: where the cursor and the window were when it
	// began, so a move is a delta from there and not from the last event
	// (which would let a missed event walk the window off the hand).
	dragging    bool
	pressCursor winPoint
	pressOrigin winPoint
	moved       bool
	onClick     func()
	onMoved     func(x, y int)
	onDrag      func(bool)
}

var (
	companionClassOnce sync.Once
	companionClass     *uint16
)

func openCompanionBody(x, y int) (companionBody, error) {
	w, err := openCompanionWindow(x, y)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// openCompanionWindow brings the body up at (x, y) physical pixels — a spot
// the caller remembered, or a negative pair meaning "you choose". Returns
// once the window exists or could not be created.
func openCompanionWindow(x, y int) (*companionWindow, error) {
	w := &companionWindow{ready: make(chan error, 1), gone: make(chan struct{}), x: x, y: y}
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

	// Sized for the placeholder at the primary monitor's scale; the real
	// canvas (companion_draw.go) will size itself per monitor.
	w.dpi = 96
	w.w, w.h = 400, 220
	if w.x < 0 || w.y < 0 {
		w.x, w.y = 200, 200
	}
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

	if dpi, _, _ := procGetDpiForWindow.Call(hwnd); dpi != 0 {
		w.dpi = int(dpi)
	}
	w.resizeForDPI()
	w.paint()
	procShowWindow.Call(hwnd, swShowNoActivate)
	debuglog.Msg("companion: window %#x at %d,%d %dx%d dpi=%d", hwnd, w.x, w.y, w.w, w.h, w.dpi)
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

// resizeForDPI keeps the window the same logical size on every monitor.
func (w *companionWindow) resizeForDPI() {
	w.w = 400 * w.dpi / 96
	w.h = 220 * w.dpi / 96
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
	w := companionWindows.get(hwnd)
	if w != nil {
		switch msg {
		case wmLButtonDown:
			w.pressBegin()
			return 0
		case wmMouseMove:
			if w.dragging {
				w.pressMove()
			}
			return 0
		case wmLButtonUp:
			w.pressEnd()
			return 0
		case wmCaptureChanged:
			w.dragging = false
			return 0
		case wmDPIChanged:
			// HIWORD(wParam) is the new DPI. lParam carries a rect Windows
			// suggests; it is not read — reading a uintptr as a pointer is
			// what vet forbids, and the two cases have their own answers.
			//
			// Mid-drag the grab is kept: the point under the cursor stays
			// under the cursor, scaled with the window, so the window's
			// centre goes where the hand goes and does not hop back over the
			// edge it just crossed (which made the message come four times
			// in 20 ms, each undoing the last). Otherwise — the display
			// settings changed under a resting window — it is resized about
			// its own centre.
			old := w.dpi
			w.dpi = int(wparam >> 16 & 0xffff)
			if old == 0 {
				old = w.dpi
			}
			if w.dragging {
				offX := int(w.pressCursor.X-w.pressOrigin.X) * w.dpi / old
				offY := int(w.pressCursor.Y-w.pressOrigin.Y) * w.dpi / old
				w.pressOrigin = winPoint{w.pressCursor.X - int32(offX), w.pressCursor.Y - int32(offY)}
				w.resizeForDPI()
				w.paint()
				w.pressMove()
			} else {
				cx, cy := w.x+w.w/2, w.y+w.h/2
				w.resizeForDPI()
				w.paint()
				w.x, w.y = clampToWorkArea(cx-w.w/2, cy-w.h/2, w.w, w.h)
				procSetWindowPos.Call(hwnd, 0, uintptr(w.x), uintptr(w.y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
			}
			debuglog.Msg("companion: dpi now %d → %dx%d at %d,%d", w.dpi, w.w, w.h, w.x, w.y)
			return 0
		case wmDestroy:
			procPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return r
}

func cursorPos() winPoint {
	var p winPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

func (w *companionWindow) pressBegin() {
	procSetCapture.Call(w.hwnd)
	w.dragging = true
	w.moved = false
	w.pressCursor = cursorPos()
	var r winRect
	procGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)))
	w.pressOrigin = winPoint{r.Left, r.Top}
	w.x, w.y = int(r.Left), int(r.Top)
}

func (w *companionWindow) pressMove() {
	c := cursorPos()
	dx, dy := int(c.X-w.pressCursor.X), int(c.Y-w.pressCursor.Y)
	if !w.moved && (abs(dx) >= companionClickPx || abs(dy) >= companionClickPx) {
		w.moved = true
		if w.onDrag != nil {
			w.onDrag(true)
		}
	}
	if !w.moved {
		return
	}
	x, y := int(w.pressOrigin.X)+dx, int(w.pressOrigin.Y)+dy
	x, y = clampToWorkArea(x, y, w.w, w.h)
	if x == w.x && y == w.y {
		return
	}
	w.x, w.y = x, y
	procSetWindowPos.Call(w.hwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
}

func (w *companionWindow) pressEnd() {
	if !w.dragging {
		return
	}
	w.dragging = false
	procReleaseCapture.Call()
	if w.moved {
		if w.onDrag != nil {
			w.onDrag(false)
		}
		if w.onMoved != nil {
			w.onMoved(w.x, w.y)
		}
		return
	}
	if w.onClick != nil {
		w.onClick()
	}
}

// clampToWorkArea keeps the figure — the middle of the canvas, the part that
// is not transparent — on the desktop. Only the figure, on purpose: the
// canvas is wider than the figure so the bubble has room, and a canvas
// forced whole onto one monitor could never straddle the edge between two,
// which is what a drag from one to the other does halfway.
//
// "On the desktop" means all the monitors together. The figure is held
// inside the work area (the desktop less the taskbar) of the monitor its
// centre is nearest, but only at the edges where the desktop ends: an edge
// with another monitor beyond it is left open, or the centre could never
// reach that monitor and the figure would be stuck at the seam. Once the
// centre is across, the other monitor's own edges take over.
// SPI_GETWORKAREA (window_windows.go) only knows the primary; this asks per
// monitor.
func clampToWorkArea(x, y, w, h int) (int, int) {
	fw, fh := w*companionFigureShare/100, h*companionFigureShare/100
	mon, _, _ := procMonitorFromPoint.Call(packPoint(x+w/2, y+h/2), monitorDefaultToNearest)
	if mon == 0 {
		return x, y
	}
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
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
	return clampInner(x, y, w, h, fw, fh, mi.Work, open)
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

// companionFigureShare is how much of the canvas, centred, must stay on a
// monitor: the placeholder ball's extent for now, the mascot's box later.
const companionFigureShare = 64

// clampInner moves a w×h canvas at (x, y) the least distance that puts its
// centred iw×ih figure inside area — on the edges that are not open. open
// is left, top, right, bottom.
func clampInner(x, y, w, h, iw, ih int, area winRect, open [4]bool) (int, int) {
	ox, oy := (w-iw)/2, (h-ih)/2
	fx, fy := x+ox, y+oy
	if !open[0] {
		fx = max(fx, int(area.Left))
	}
	if !open[2] {
		fx = min(fx, int(area.Right)-iw)
	}
	if !open[1] {
		fy = max(fy, int(area.Top))
	}
	if !open[3] {
		fy = min(fy, int(area.Bottom)-ih)
	}
	return fx - ox, fy - oy
}

// paint composes the current frame and hands it to Windows.
func (w *companionWindow) paint() {
	w.present(placeholderFrame(w.w, w.h))
}

// present shows an RGBA frame (premultiplied, as image.RGBA is) as the whole
// window: it is copied into a DIB section — the one bitmap format
// UpdateLayeredWindow takes — as BGRA, and the window takes the frame's size.
func (w *companionWindow) present(img *image.RGBA) {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	if width == 0 || height == 0 {
		return
	}
	bmi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(width),
		Height:   -int32(height), // top-down, like img.Pix
		Planes:   1,
		BitCount: 32,
	}
	screen, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, screen)
	var bits unsafe.Pointer
	hbm, _, err := procCreateDIBSectionOv.Call(screen, uintptr(unsafe.Pointer(&bmi)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		debuglog.Msg("companion: CreateDIBSection failed: %v", err)
		return
	}
	defer procDeleteObject.Call(hbm)
	dst := unsafe.Slice((*byte)(bits), width*height*4)
	for y := 0; y < height; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+width*4]
		out := dst[y*width*4 : (y+1)*width*4]
		for i := 0; i < width*4; i += 4 {
			out[i], out[i+1], out[i+2], out[i+3] = row[i+2], row[i+1], row[i], row[i+3]
		}
	}
	mem, _, _ := procCreateCompatibleDC.Call(screen)
	defer procDeleteDC.Call(mem)
	old, _, _ := procSelectObject.Call(mem, hbm)
	defer procSelectObject.Call(mem, old)

	size := winPoint{int32(width), int32(height)}
	src := winPoint{}
	blend := blendFunction{BlendOp: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: acSrcAlpha}
	if ok, _, err := procUpdateLayeredWindow.Call(w.hwnd, 0, 0, uintptr(unsafe.Pointer(&size)), mem, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&blend)), ulwAlpha); ok == 0 {
		debuglog.Msg("companion: UpdateLayeredWindow failed: %v", err)
	}
	w.w, w.h = width, height
}

// placeholderFrame is the stand-in for the mascot until the sprites exist: a
// shaded ball in the middle of a transparent canvas, so that "is it really
// transparent, does a click beside it go through, does it survive a monitor
// with another scale" can each be answered by looking.
func placeholderFrame(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	r := float64(min(w, h)) * 0.32
	cx, cy := float64(w)/2, float64(h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			d := math.Hypot(dx, dy)
			if d > r+1 {
				continue
			}
			// Edge coverage for a soft rim, a highlight up-left, blue tint.
			a := math.Min(1, r+1-d)
			light := 1 - math.Hypot(dx+r*0.3, dy+r*0.3)/(r*1.6)
			light = math.Max(0.15, math.Min(1, light))
			cr, cg, cb := 60+150*light, 110+120*light, 240*math.Min(1, light+0.3)
			img.SetRGBA(x, y, color.RGBA{uint8(cr * a), uint8(cg * a), uint8(cb * a), uint8(255 * a)})
		}
	}
	return img
}
