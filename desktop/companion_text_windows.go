//go:build windows

package main

// Text for the desktop body's bubble, drawn by GDI.
//
// Go has no shaping engine, and Thai needs one — tone marks stack, vowels
// sit before, above and below the consonant they belong to — so the bubble's
// words are painted by the OS, the way every other window's are, with the
// same Segoe UI the app's stylesheet names and Windows' own font linking to
// Leelawadee for Thai. The catch with a layered window is that GDI writes no
// alpha: the glyphs come out with A=0 and the compositor drops them. So each
// run of text is painted on an opaque bitmap in the card's own colour and
// copied onto the card with alpha forced to 255 — the card is opaque anyway,
// so nothing is lost (the reviewer's point, 12 ก.ย. 2026).

import (
	"image"
	"image/color"
	"sync"
	"syscall"
	"unsafe"
)

var (
	procCreateFontW           = gdi32.NewProc("CreateFontW")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")
	procTextOutW              = gdi32.NewProc("TextOutW")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procSetBkColor            = gdi32.NewProc("SetBkColor")
	procGetTextMetricsW       = gdi32.NewProc("GetTextMetricsW")
)

const (
	fwNormal            = 400
	defaultCharset      = 1
	outDefaultPrecis    = 0
	clipDefaultPrecis   = 0
	antialiasedQuality  = 4
	defaultPitchFFDont  = 0
	companionFontFace   = "Segoe UI"
	companionLineHeight = 1.4
)

type textMetricW struct {
	Height, Ascent, Descent, InternalLeading, ExternalLeading int32
	AveCharWidth, MaxCharWidth, Weight, Overhang              int32
	DigitizedAspectX, DigitizedAspectY                        int32
	FirstChar, LastChar, DefaultChar, BreakChar               uint16
	Italic, Underlined, StruckOut, PitchAndFamily, CharSet    byte
}

// gdiText is a textPainter over GDI. Fonts are made per pixel size and kept;
// a size is asked for over and over at one scale.
type gdiText struct {
	mu    sync.Mutex
	fonts map[int]uintptr
}

func newGDIText() *gdiText { return &gdiText{fonts: map[int]uintptr{}} }

func (g *gdiText) font(px int) uintptr {
	g.mu.Lock()
	defer g.mu.Unlock()
	if f, ok := g.fonts[px]; ok {
		return f
	}
	face, _ := syscall.UTF16PtrFromString(companionFontFace)
	// A negative height is the character height without internal leading —
	// the CSS font-size.
	f, _, _ := procCreateFontW.Call(uintptr(int32(-px)), 0, 0, 0, fwNormal, 0, 0, 0,
		defaultCharset, outDefaultPrecis, clipDefaultPrecis, antialiasedQuality, defaultPitchFFDont,
		uintptr(unsafe.Pointer(face)))
	g.fonts[px] = f
	return f
}

// withDC runs fn with a memory DC that has the font selected.
func (g *gdiText) withDC(px int, fn func(dc uintptr)) {
	dc, _, _ := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return
	}
	defer procDeleteDC.Call(dc)
	old, _, _ := procSelectObject.Call(dc, g.font(px))
	defer procSelectObject.Call(dc, old)
	fn(dc)
}

func (g *gdiText) measure(s string, px int) int {
	if s == "" {
		return 0
	}
	u := syscall.StringToUTF16(s)
	var size struct{ CX, CY int32 }
	g.withDC(px, func(dc uintptr) {
		procGetTextExtentPoint32W.Call(dc, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1), uintptr(unsafe.Pointer(&size)))
	})
	return int(size.CX)
}

func (g *gdiText) lineHeight(px int) int {
	return int(float64(px)*companionLineHeight + 0.5)
}

// paint draws the lines on an opaque w×h bitmap of bg, each line centred in
// its line-height, and returns it as RGBA with every alpha 255.
func (g *gdiText) paint(lines []string, px, w, h int, bg, fg color.RGBA) *image.RGBA {
	if w <= 0 || h <= 0 {
		return nil
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	g.withDC(px, func(dc uintptr) {
		bmi := bitmapInfoHeader{Size: uint32(unsafe.Sizeof(bitmapInfoHeader{})), Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 32}
		var bits unsafe.Pointer
		hbm, _, _ := procCreateDIBSectionOv.Call(dc, uintptr(unsafe.Pointer(&bmi)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
		if hbm == 0 || bits == nil {
			return
		}
		defer procDeleteObject.Call(hbm)
		oldBmp, _, _ := procSelectObject.Call(dc, hbm)
		defer procSelectObject.Call(dc, oldBmp)

		px4 := unsafe.Slice((*byte)(bits), w*h*4)
		for i := 0; i < len(px4); i += 4 {
			px4[i], px4[i+1], px4[i+2], px4[i+3] = bg.B, bg.G, bg.R, 255
		}
		procSetBkColor.Call(dc, colorref(bg))
		procSetTextColor.Call(dc, colorref(fg))
		var tm textMetricW
		procGetTextMetricsW.Call(dc, uintptr(unsafe.Pointer(&tm)))
		lh := g.lineHeight(px)
		for i, line := range lines {
			if line == "" {
				continue
			}
			u := syscall.StringToUTF16(line)
			y := i*lh + (lh-int(tm.Height))/2
			procTextOutW.Call(dc, 0, uintptr(y), uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)-1))
		}
		// GDI wrote BGR with whatever alpha; the card is opaque, so it is 255.
		for i := 0; i < len(px4); i += 4 {
			out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = px4[i+2], px4[i+1], px4[i], 255
		}
	})
	return out
}

// colorref is GDI's 0x00BBGGRR.
func colorref(c color.RGBA) uintptr {
	return uintptr(c.R) | uintptr(c.G)<<8 | uintptr(c.B)<<16
}
