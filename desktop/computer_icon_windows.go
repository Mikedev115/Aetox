//go:build windows

package main

// The program's own icon, pulled out of its executable.
//
// The owner asked for logos on the register (9 ก.ย., over a screenshot of the
// text-only list: *"เอาโลโก้มาด้วย"*), and a set of hand-drawn logos would not
// have answered it. The picker lists whatever the user happens to have open —
// their accounting program, their label printer, the thing their office wrote in
// 2011 — so any curated set is a set that covers Chrome and misses the program
// they actually wanted to allow. What every one of them does have is an icon
// inside its own .exe, which is the same picture Windows shows on the taskbar.
// The row and the taskbar then agree, which is the whole job: a person picking
// a program should recognise it at a glance rather than read a filename.
//
// Cached by path for the life of the process. Extracting one is a file read, an
// icon draw and a PNG encode; the settings page asks for every open window at
// once and refreshes on demand, so without a cache that is a dozen of them every
// time somebody presses refresh.

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	// shell32, not user32: ExtractIconEx is a shell function, and a LazyProc
	// that cannot find its export PANICS on the first call rather than failing.
	// Found by this file's own test on its first run, which is the cheapest
	// possible place to find it — in the app it would have been a crash the
	// moment somebody opened the settings page.
	shell32            = syscall.NewLazyDLL("shell32.dll")
	procExtractIconExW = shell32.NewProc("ExtractIconExW")

	procDestroyIcon = user32.NewProc("DestroyIcon")
	procDrawIconEx  = user32.NewProc("DrawIconEx")

	procCreateDIBSectionIcon = gdi32.NewProc("CreateDIBSection")
)

const (
	diNormal   = 0x0003
	iconPixels = 32 // what the row draws at; asking for more is bytes nobody sees
)

var (
	iconMu    sync.Mutex
	iconCache = map[string]string{}
)

// exeIconDataURL returns the program's icon as a `data:image/png;base64,...`
// string, or "" when there is nothing to show.
//
// Empty is a normal answer, not a failure: a program can legitimately carry no
// icon, and a path this user may not read (a service, something at higher
// integrity) is the same kind of nothing. The row draws a placeholder and says
// the same thing it said before; nobody is told a picture failed.
func exeIconDataURL(exePath string) string {
	key := strings.ToLower(strings.TrimSpace(exePath))
	if key == "" {
		return ""
	}
	iconMu.Lock()
	if got, ok := iconCache[key]; ok {
		iconMu.Unlock()
		return got
	}
	iconMu.Unlock()

	url := extractIconPNG(exePath)
	iconMu.Lock()
	iconCache[key] = url // "" is cached too: a second failure costs the same as the first
	iconMu.Unlock()
	return url
}

func extractIconPNG(exePath string) string {
	// Every export this function needs, checked before any of them is called.
	// LazyProc.Call panics on a missing export, and a decorative picture must
	// never be able to take the app down — a stripped or unusual Windows is a
	// machine with no icons on the list, not a machine that crashes.
	for _, p := range []*syscall.LazyProc{procExtractIconExW, procDestroyIcon, procDrawIconEx, procCreateDIBSectionIcon} {
		if p.Find() != nil {
			return ""
		}
	}
	wide, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return ""
	}
	var large, small uintptr
	// One icon, the large one. ExtractIconEx returns the count when asked for
	// none and the count of extracted when asked for some; anything but 1 here
	// means there was nothing usable to take.
	n, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(wide)), 0,
		uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&small)), 1)
	if small != 0 {
		defer procDestroyIcon.Call(small)
	}
	if n == 0 || large == 0 {
		return ""
	}
	defer procDestroyIcon.Call(large)

	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		return ""
	}
	defer procReleaseDC.Call(0, screenDC)
	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return ""
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
		Width:    iconPixels,
		Height:   -iconPixels, // top-down, so the copy below is a straight walk
		Planes:   1,
		BitCount: 32,
	}}

	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSectionIcon.Call(memDC, uintptr(unsafe.Pointer(&bi)),
		dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return ""
	}
	defer procDeleteObject.Call(bmp)
	old, _, _ := procSelectObject.Call(memDC, bmp)
	defer procSelectObject.Call(memDC, old)

	if ok, _, _ := procDrawIconEx.Call(memDC, 0, 0, large,
		iconPixels, iconPixels, 0, 0, diNormal); ok == 0 {
		return ""
	}

	img := image.NewRGBA(image.Rect(0, 0, iconPixels, iconPixels))
	src := unsafe.Slice((*byte)(bits), iconPixels*iconPixels*4)
	opaque := false
	for i := 0; i < iconPixels*iconPixels; i++ {
		a := src[i*4+3]
		if a != 0 {
			opaque = true
		}
		// BGRA to RGBA. NOT premultiplied here, unlike the overlay: this one
		// goes into a PNG for a browser, and PNG's alpha is straight.
		img.Pix[i*4+0] = src[i*4+2]
		img.Pix[i*4+1] = src[i*4+1]
		img.Pix[i*4+2] = src[i*4+0]
		img.Pix[i*4+3] = a
	}
	if !opaque {
		// Every pixel transparent. DrawIconEx said it succeeded and drew
		// nothing, which happens with some icon formats; a fully transparent
		// square in the row would read as a broken image rather than as no
		// icon, and those two want different drawings.
		return ""
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// knownProgramPaths is where the three rows that are not built yet keep their
// executables, so those rows can carry a real logo too.
//
// A short list of standard locations rather than a registry crawl, and a miss is
// answered with no icon rather than with a search: this is decoration on a row
// whose text already says everything that matters, and it must never be the
// slowest thing on the page.
var knownProgramPaths = map[string][]string{
	"chrome": {
		`%ProgramFiles%\Google\Chrome\Application\chrome.exe`,
		`%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe`,
		`%LocalAppData%\Google\Chrome\Application\chrome.exe`,
	},
	"msedge": {
		`%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe`,
		`%ProgramFiles%\Microsoft\Edge\Application\msedge.exe`,
	},
	"excel": {
		`%ProgramFiles%\Microsoft Office\root\Office16\EXCEL.EXE`,
		`%ProgramFiles(x86)%\Microsoft Office\root\Office16\EXCEL.EXE`,
		`%ProgramFiles%\Microsoft Office\Office16\EXCEL.EXE`,
	},
}

// ProgramIcon answers the settings page for a program named rather than found:
// the Chrome, Edge and Excel rows, which are on the page whether or not they are
// running. Empty when the program is not installed, which the row already knows
// how to draw.
func (a *App) ProgramIcon(name string) string {
	key := exeKey(name)
	if key == "" {
		return ""
	}
	// Running beats installed: the copy the user is actually looking at is the
	// one whose icon should be on the row.
	if windows, err := reachListWindows(); err == nil {
		for _, w := range windows {
			if exeKey(w.Exe) == key && w.Exe != "" {
				if url := exeIconDataURL(w.Exe); url != "" {
					return url
				}
			}
		}
	}
	for _, candidate := range knownProgramPaths[key] {
		path := expandWindowsVars(candidate)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if url := exeIconDataURL(path); url != "" {
			return url
		}
	}
	return ""
}

// expandWindowsVars resolves %NAME% the way a person writing that path meant it.
// os.ExpandEnv only knows $NAME, and rewriting the table into that spelling
// would make it unreadable to anyone checking it against their own machine.
func expandWindowsVars(path string) string {
	var out strings.Builder
	for {
		open := strings.Index(path, "%")
		if open < 0 {
			out.WriteString(path)
			return filepath.Clean(out.String())
		}
		close := strings.Index(path[open+1:], "%")
		if close < 0 {
			out.WriteString(path)
			return filepath.Clean(out.String())
		}
		name := path[open+1 : open+1+close]
		value := os.Getenv(name)
		if value == "" {
			return "" // a variable this machine does not set: not a path, not a guess
		}
		out.WriteString(path[:open])
		out.WriteString(value)
		path = path[open+close+2:]
	}
}
