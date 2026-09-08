//go:build windows

package main

// The light around the screen, and why the strip in the chat was not enough.
//
// The takeover banner (Chat.svelte, `.driving-strip`) is drawn inside Aetox's
// own window. The whole point of the foreground model the owner chose is that
// the window being driven comes to the FRONT — over Aetox. So at the one moment
// the banner matters, it is behind the thing it is warning about. The owner put
// it plainly, watching this run: *"ขอแสงวิวับที่ทำไว้อ่ะมาครอบจอด้วย ไม่งั้นไม่รู้"*.
//
// So the signal has to live above every window, which means a window of its own:
// layered (per-pixel alpha), transparent to the mouse, never activated, always
// on top, covering the whole virtual desktop. It draws a band around the edge of
// the screen with a bright head travelling round it.
//
// It is the beam from style.css, moved outside the app. Same idea, same job,
// same reason it is a travelling light rather than a static border: a border
// that does not move reads as part of the screen within about two seconds, and
// what this has to say is "right now", continuously, for as long as it lasts.
// The gradient is built in Go instead of CSS because there is no browser out
// here, and drawing it with GDI is a few dozen lines against the alternative of
// a second WebView2 with a transparency story of its own.
//
// Four things it must never do, each of which is one window style:
//
//	WS_EX_TRANSPARENT   every click passes through to the window underneath
//	WS_EX_NOACTIVATE    it never takes focus, so it cannot steal a keystroke
//	WS_EX_TOOLWINDOW    it is not in alt-tab and not on the taskbar
//	WS_EX_TOPMOST       it is above the window being driven, which is the point
//
// And one thing the code must never do: block. It runs on its own thread with
// its own message pump, and the acting call only ever sets a flag.

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

var (
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
	procPeekMessageW        = user32.NewProc("PeekMessageW")
	procUpdateLayeredWindow = user32.NewProc("UpdateLayeredWindow")

	procCreateDIBSectionOv = gdi32.NewProc("CreateDIBSection")
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79

	// Only what browser_windows.go does not already declare. That file owns the
	// package's Win32 constant block; wsPopup, swHide, wsExToolWindow,
	// wsExNoActivate, swpNoMove, swpNoSize and swpNoActivate all come from there.
	wsExLayered     = 0x00080000
	wsExTransparent = 0x00000020
	wsExTopmost     = 0x00000008

	swShowNoActivate = 4

	ulwAlpha   = 0x00000002
	acSrcOver  = 0x00
	acSrcAlpha = 0x01
	pmRemove   = 0x0001
)

// hwndTopmost is HWND_TOPMOST, which is (HWND)-1 rather than a small number
// like hwndTop and hwndBottom next door — the difference between "above the
// windows it is stacked with" and "above every window that is not itself
// topmost", and only the second one is any use for a warning.
var hwndTopmost = ^uintptr(0)

// overlayBand is how thick the light is, in pixels. Thin enough to be a frame
// rather than a curtain: this has to be unmissable in peripheral vision and
// must not obscure the corner of a window somebody is reading.
const overlayBand = 9

// overlayFPS is deliberately modest. The light is a signal, not an animation to
// admire, and this thread redraws a full-screen bitmap on every frame — at 60
// this would be a measurable share of a core for as long as the agent works.
const overlayFPS = 24

// overlayLinger is how long the light stays after an action finishes.
//
// Without it the light is a strobe. An acting call is often under a second, and
// a model working through a dialog fires several in a row, so lighting and
// unlighting per action would flash the edge of the screen four times in six
// seconds — which reads as a fault rather than as a state, and is exactly the
// "light show" the owner had removed from the delegation cards and the sidebar
// (style.css, the beam block). One steady light across a run of actions says
// the true thing: the agent is driving, still.
const overlayLinger = 2500 * time.Millisecond

type overlayCmd struct {
	on bool
	// at is where Aetox's own pointer should be, in virtual-screen pixels, and
	// (0,0) means "do not draw one". See overlayPointer.
	at  winPoint
	aim bool
}

type screenOverlay struct {
	mu      sync.Mutex
	started bool
	cmds    chan overlayCmd
}

var overlay = &screenOverlay{cmds: make(chan overlayCmd, 4)}

// show, hide and point are what the tool calls. All return as soon as the thread
// has the message; an acting call must never wait on a decoration.
func (o *screenOverlay) show() { o.send(overlayCmd{on: true}) }
func (o *screenOverlay) hide() { o.send(overlayCmd{on: false}) }

// point puts Aetox's own pointer on a control, in virtual-screen pixels.
//
// **The real mouse never moves.** That is the whole design and it is a better
// answer than the one the owner asked for by name — Codex on macOS gives each
// agent its own cursor context, which Windows has no equivalent of, and moving
// the real one would take the pointer out of the user's hand mid-sentence. What
// this draws instead is a second pointer, in the same light as the frame, at the
// element being pressed: the user sees WHERE Aetox is working and keeps their
// own mouse the whole time.
//
// It is also honest in a way a hijacked cursor could not be. Nothing here is
// aimed by coordinate — Invoke and SetValue go through the control — so this
// pointer is a report of what was pressed, not the mechanism that pressed it.
func (o *screenOverlay) point(x, y int) {
	o.send(overlayCmd{on: true, aim: true, at: winPoint{X: int32(x), Y: int32(y)}})
}

func (o *screenOverlay) send(cmd overlayCmd) {
	o.mu.Lock()
	if !o.started {
		o.started = true
		go o.run()
	}
	o.mu.Unlock()
	select {
	case o.cmds <- cmd:
	default:
		// The thread is mid-frame with a command already queued. Dropping is
		// right: the queued one is newer than nothing and the next action will
		// send again. A blocked acting call would be worse than a late light.
	}
}

func (o *screenOverlay) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hwnd, err := createOverlayWindow()
	if err != nil {
		debuglog.Msg("computer.overlay: %v", err)
		return
	}
	defer procDestroyWindow.Call(hwnd)

	x, y, w, h := virtualScreen()
	lit := false
	var goDarkAt time.Time
	phase := 0.0
	// Aetox's own pointer: where it was, where it is going, and how far along.
	// aimAt < 0 means there is none to draw.
	var aimFrom, aimTo, aimNow winPoint
	aimAt := -1.0
	tick := time.NewTicker(time.Second / overlayFPS)
	defer tick.Stop()

	var msg winMsg
	for {
		// Drain the pump so the window stays responsive to the system even
		// though it answers nothing itself. PeekMessage rather than GetMessage:
		// GetMessage blocks, and this loop has a clock of its own to keep.
		for {
			r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), hwnd, 0, 0, pmRemove)
			if r == 0 {
				break
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}

		select {
		case cmd := <-o.cmds:
			if cmd.aim {
				// Where it is going, so the pointer travels rather than
				// teleporting. A pointer that jumps says "something happened
				// somewhere"; one that moves says which thing, and the eye
				// follows a moving object without being asked to.
				if aimAt < 0 {
					// The first aim of a run has nowhere to travel from, and
					// sliding in from the top-left corner would draw a line
					// across the user's screen for no reason.
					aimNow = cmd.at
				}
				aimFrom, aimTo, aimAt = aimNow, cmd.at, 0
			}
			if !cmd.on {
				// Not off now: off in a moment, unless something asks for it
				// again first. See overlayLinger.
				goDarkAt = time.Now().Add(overlayLinger)
				break
			}
			goDarkAt = time.Time{}
			if lit {
				break
			}
			lit = true
			if lit {
				// Re-read the screen every time it lights up rather than once at
				// startup: a monitor plugged in or a resolution changed between
				// two actions would otherwise leave the frame around a rectangle
				// that is no longer the screen.
				x, y, w, h = virtualScreen()
				procSetWindowPos.Call(hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoActivate)
				procShowWindow.Call(hwnd, swShowNoActivate)
			}
		case <-tick.C:
			if !lit {
				continue
			}
			if !goDarkAt.IsZero() && time.Now().After(goDarkAt) {
				lit = false
				goDarkAt = time.Time{}
				aimAt = -1 // the pointer belongs to the run, not to the screen
				procShowWindow.Call(hwnd, uintptr(swHide))
				continue
			}
			phase += 1.0 / (overlayFPS * 3.0) // one lap every three seconds, as the beam does
			if phase >= 1 {
				phase -= 1
			}
			// Topmost is reasserted on every frame, not claimed once. Another
			// application going full-screen or raising its own topmost window
			// would otherwise put itself over the warning, which is the one
			// failure this whole file exists to prevent.
			procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoSize|swpNoMove|swpNoActivate)
			if aimAt >= 0 && aimAt < 1 {
				aimAt += 4.0 / overlayFPS // a quarter second to cross
				if aimAt > 1 {
					aimAt = 1
				}
				aimNow = winPoint{
					X: aimFrom.X + int32(float64(aimTo.X-aimFrom.X)*ease(aimAt)),
					Y: aimFrom.Y + int32(float64(aimTo.Y-aimFrom.Y)*ease(aimAt)),
				}
			}
			pointer := winPoint{X: -1, Y: -1}
			if aimAt >= 0 {
				// Into the bitmap's own coordinates: the window covers the
				// virtual screen, whose origin is negative on a layout with a
				// monitor to the left of the primary one.
				pointer = winPoint{X: aimNow.X - int32(x), Y: aimNow.Y - int32(y)}
			}
			paintOverlay(hwnd, x, y, w, h, phase, pointer)
		}
	}
}

func virtualScreen() (x, y, w, h int) {
	vx, _, _ := procGetSystemMetrics.Call(smXVirtualScreen)
	vy, _, _ := procGetSystemMetrics.Call(smYVirtualScreen)
	vw, _, _ := procGetSystemMetrics.Call(smCXVirtualScreen)
	vh, _, _ := procGetSystemMetrics.Call(smCYVirtualScreen)
	return int(int32(vx)), int(int32(vy)), int(int32(vw)), int(int32(vh))
}

func createOverlayWindow() (uintptr, error) {
	className, err := syscall.UTF16PtrFromString("AetoxDrivingOverlay")
	if err != nil {
		return 0, err
	}
	wndProc := syscall.NewCallback(func(hwnd, msg, wparam, lparam uintptr) uintptr {
		r, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		return r
	})
	wc := wndClassExW{
		Size:      uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:   wndProc,
		ClassName: className,
	}
	// A class already registered by a previous run of this thread is fine; the
	// create below is what actually has to succeed.
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	x, y, w, h := virtualScreen()
	hwnd, _, callErr := procCreateWindowExW.Call(
		uintptr(wsExLayered|wsExTransparent|wsExNoActivate|wsExTopmost|wsExToolWindow),
		uintptr(unsafe.Pointer(className)),
		0,
		uintptr(wsPopup),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		0, 0, 0, 0)
	if hwnd == 0 {
		return 0, win32Error{call: "CreateWindowExW(overlay)", code: errnoOf(callErr)}
	}
	return hwnd, nil
}

// paintOverlay draws one frame: a band around the edge whose brightness sweeps
// round the perimeter.
//
// Per-pixel alpha through UpdateLayeredWindow, which is the only way to have a
// window that is genuinely see-through in the middle and solid at the edge. The
// bitmap is premultiplied, which is not optional: Windows composites it as-is,
// and unpremultiplied colour shows up as a bright halo around every soft edge.
// ease is the shape of the pointer's travel: quick away, slow into place. A
// linear slide reads as a machine dragging something; this reads as an arm.
func ease(t float64) float64 { return 1 - math.Pow(1-t, 3) }

func paintOverlay(hwnd uintptr, x, y, w, h int, phase float64, pointer winPoint) {
	if w <= 0 || h <= 0 {
		return
	}
	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		return
	}
	defer procReleaseDC.Call(0, screenDC)
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return
	}
	defer procDeleteDC.Call(memDC)

	type bmiHeader struct {
		Size                    uint32
		Width, Height           int32
		Planes, BitCount        uint16
		Compression, SizeImage  uint32
		XPelsPerMeter, YPelsPer int32
		ClrUsed, ClrImportant   uint32
	}
	bi := struct {
		Header bmiHeader
		_      [3]uint32
	}{Header: bmiHeader{
		Size:     uint32(unsafe.Sizeof(bmiHeader{})),
		Width:    int32(w),
		Height:   int32(-h), // top-down
		Planes:   1,
		BitCount: 32,
	}}

	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSectionOv.Call(memDC, uintptr(unsafe.Pointer(&bi)),
		dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return
	}
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(memDC, bmp)
	defer procSelectObject.Call(memDC, old)

	buf := unsafe.Slice((*byte)(bits), w*h*4)
	drawBand(buf, w, h, phase)
	if pointer.X >= 0 {
		drawPointer(buf, w, h, int(pointer.X), int(pointer.Y))
	}

	src := winPoint{}
	dst := winPoint{X: int32(x), Y: int32(y)}
	size := struct{ CX, CY int32 }{int32(w), int32(h)}
	blend := struct {
		Op, Flags, Alpha, Format byte
	}{acSrcOver, 0, 255, acSrcAlpha}

	procUpdateLayeredWindow.Call(hwnd, screenDC,
		uintptr(unsafe.Pointer(&dst)), uintptr(unsafe.Pointer(&size)),
		memDC, uintptr(unsafe.Pointer(&src)), 0,
		uintptr(unsafe.Pointer(&blend)), ulwAlpha)
}

// overlayHue is the light's colour, and it is the app's own accent rather than
// a warning red. Being driven is the state the user asked for, not a fault —
// red would say something went wrong every single time something went right.
var overlayHue = color.RGBA{R: 0x6E, G: 0x9B, B: 0xFF}

// drawBand fills the edge band. Every pixel's alpha is the product of two
// falloffs: how deep into the band it is, and how far round the perimeter it is
// from the travelling head.
func drawBand(px []byte, w, h int, phase float64) {
	for i := range px {
		px[i] = 0
	}
	perim := float64(2 * (w + h)) // clockwise from the top-left corner
	if perim == 0 {
		return
	}
	head := phase * perim

	set := func(x, y int, depth int) {
		if x < 0 || y < 0 || x >= w || y >= h {
			return
		}
		// Where this pixel sits along the perimeter, so the head can be a
		// distance rather than a shape.
		var along float64
		switch {
		case y < overlayBand: // top edge, left to right
			along = float64(x)
		case x >= w-overlayBand: // right edge, top to bottom
			along = float64(w) + float64(y)
		case y >= h-overlayBand: // bottom edge, right to left
			along = float64(w+h) + float64(w-x)
		default: // left edge, bottom to top
			along = float64(2*w+h) + float64(h-y)
		}
		d := math.Abs(along - head)
		if d > perim/2 {
			d = perim - d // the lap wraps; the head is close to both ends at once
		}

		// A long tail behind a bright head, the same asymmetry the CSS beam
		// uses to say which way it is travelling. Two heads, half a lap apart,
		// so a wide screen is never dark along a whole edge.
		lead := math.Exp(-d / (perim * 0.06))
		opposite := math.Abs(d - perim/2)
		if opposite > perim/2 {
			opposite = perim - opposite
		}
		lead += 0.55 * math.Exp(-opposite/(perim*0.06))

		// Fade into the screen across the band's depth, so the inner edge has
		// no hard line on it.
		fade := 1 - float64(depth)/float64(overlayBand)
		a := (0.20 + 0.80*math.Min(lead, 1)) * fade
		if a <= 0 {
			return
		}
		if a > 1 {
			a = 1
		}
		alpha := byte(a * 255)
		o := (y*w + x) * 4
		// Premultiplied BGRA.
		px[o+0] = byte(float64(overlayHue.B) * a)
		px[o+1] = byte(float64(overlayHue.G) * a)
		px[o+2] = byte(float64(overlayHue.R) * a)
		px[o+3] = alpha
	}

	for d := 0; d < overlayBand; d++ {
		for x := d; x < w-d; x++ {
			set(x, d, d)
			set(x, h-1-d, d)
		}
		for y := d; y < h-d; y++ {
			set(d, y, d)
			set(w-1-d, y, d)
		}
	}
}

// overlayFrame renders one frame into an image, for the test. Same call the
// paint path makes, so what a test looks at is what the screen gets.
func overlayFrame(w, h int, phase float64) *image.RGBA {
	px := make([]byte, w*h*4)
	drawBand(px, w, h, phase)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		img.Pix[i*4+0] = px[i*4+2]
		img.Pix[i*4+1] = px[i*4+1]
		img.Pix[i*4+2] = px[i*4+0]
		img.Pix[i*4+3] = px[i*4+3]
	}
	return img
}

// drawPointer puts Aetox's own arrow on the screen, at cx,cy in the bitmap's
// own coordinates.
//
// A ring rather than a copy of the system cursor, and the reason is not taste.
// Two identical arrows on one screen is a user hunting for which one is theirs;
// a glowing ring around the thing being pressed cannot be mistaken for a mouse
// and says the more useful thing besides — not "the pointer is here" but "THIS
// is what is being pressed". It is the click ring browser_marks.go draws on a
// page, drawn on the screen instead, for the same reason and to the same rule:
// it points at the control without covering it.
func drawPointer(px []byte, w, h, cx, cy int) {
	const (
		ring  = 16 // radius of the ring itself
		soft  = 5  // how far the glow reaches past it
		inner = 3  // the dot at the centre
	)
	for dy := -ring - soft; dy <= ring+soft; dy++ {
		y := cy + dy
		if y < 0 || y >= h {
			continue
		}
		for dx := -ring - soft; dx <= ring+soft; dx++ {
			x := cx + dx
			if x < 0 || x >= w {
				continue
			}
			d := math.Hypot(float64(dx), float64(dy))

			// Two shapes in one pass: a soft ring at radius `ring`, and a solid
			// dot at the centre so the exact point is not a guess.
			a := math.Exp(-math.Abs(d-ring) / float64(soft) * 2)
			if d <= inner {
				a = 1
			}
			if a < 0.02 {
				continue
			}
			if a > 1 {
				a = 1
			}

			o := (y*w + x) * 4
			// Drawn OVER whatever the band already put here rather than
			// replacing it: the pointer can sit on the edge of the screen, and
			// a pointer that punched a hole in the frame would look like a
			// rendering fault at exactly the moment the frame matters.
			over(px[o:o+4], overlayHue, a)
		}
	}
}

// over composites one premultiplied pixel on top of another. Premultiplied on
// both sides, because that is what UpdateLayeredWindow reads and mixing the two
// conventions is the halo bug the band's own comment warns about.
func over(dst []byte, c color.RGBA, a float64) {
	sb, sg, sr := float64(c.B)*a, float64(c.G)*a, float64(c.R)*a
	keep := 1 - a
	dst[0] = clamp8(sb + float64(dst[0])*keep)
	dst[1] = clamp8(sg + float64(dst[1])*keep)
	dst[2] = clamp8(sr + float64(dst[2])*keep)
	dst[3] = clamp8(a*255 + float64(dst[3])*keep)
}

func clamp8(v float64) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v)
}
