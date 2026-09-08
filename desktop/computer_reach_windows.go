//go:build windows

package main

// The Windows half of the reach: which windows exist, what is inside one, and
// what one looks like.
//
// Everything here answers a question the guard has already permitted. The one
// exception is reachListWindows, which runs before any grant exists — listing
// is how a user finds out what they might grant, and a list that only showed
// already-granted programs could never grow. What listing does NOT do is look
// inside: a title and a program name are what the Windows taskbar shows anyone
// walking past, and reading the tree needs the yes.
//
// Two Win32 habits worth naming, because both were paid for:
//
//   - **Every failing call carries GetLastError, not a bool.** The removed tool
//     checked `if r == 0` and reported "failed", which is how a locked screen
//     and a privilege problem became the same sentence. win32Error exists so
//     computer_errors.go can tell them apart.
//   - **The capture is of a WINDOW, never of the screen.** PrintWindow asks the
//     window to draw itself, so nothing behind or in front of it is in the
//     picture. That is a privacy property before it is a technical one — §6.4
//     of the direction doc forbids screen recording — and it also happens to
//     mean Aetox's own window can never appear in a picture Aetox took, which
//     is the prompt-injection loop Claude Desktop closes by excluding its own
//     window from screenshots.

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetWindowLongPtrW    = user32.NewProc("GetWindowLongPtrW")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procGetWindow            = user32.NewProc("GetWindow")
	procPrintWindow          = user32.NewProc("PrintWindow")
	procGetDC                = user32.NewProc("GetDC")
	procReleaseDC            = user32.NewProc("ReleaseDC")
	procIsIconic             = user32.NewProc("IsIconic")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
)

const (
	wsExToolWindow2 = 0x00000080
	gwOwner         = 4

	pwRenderFullContent = 0x00000002
	biRGB               = 0
	dibRGBColors        = 0
)

// GWL_EXSTYLE is negative, and a negative constant cannot be converted to
// uintptr at compile time. A var makes the conversion a runtime one, which is
// what every Win32 index like this needs.
var gwlExStyle = -20

type win32Rect struct{ Left, Top, Right, Bottom int32 }

// ---------------------------------------------------------------------------
// Listing
// ---------------------------------------------------------------------------

// reachListWindows returns the top-level windows a person could alt-tab to.
//
// The filter is the same one the taskbar uses, and it is a filter rather than a
// dump on purpose: a raw EnumWindows on a normal desktop returns several
// hundred handles, almost all of them invisible message-only windows with no
// title, and handing that to a model is handing it noise to hallucinate over.
func reachListWindows() ([]reachTarget, error) {
	self := int32(os.Getpid())
	var out []reachTarget
	var cbErr error

	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if vis, _, _ := procIsWindowVisible.Call(hwnd); vis == 0 {
			return 1
		}
		// Owned windows and tool windows are palettes, tooltips and dropdowns —
		// parts of another window rather than windows a person switches to.
		if owner, _, _ := procGetWindow.Call(hwnd, gwOwner); owner != 0 {
			return 1
		}
		if ex, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle)); ex&wsExToolWindow2 != 0 {
			return 1
		}
		title := windowTitle(hwnd)
		if title == "" {
			return 1
		}
		var pid uint32
		procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == 0 {
			return 1
		}
		out = append(out, reachTarget{
			HWND:  hwnd,
			PID:   int32(pid),
			Exe:   processImage(pid),
			Title: title,
		})
		return 1
	})
	if r, _, err := procEnumWindows.Call(cb, 0); r == 0 && len(out) == 0 {
		return nil, win32Error{call: "EnumWindows", code: errnoOf(err)}
	}
	if cbErr != nil {
		return nil, cbErr
	}

	// Aetox's own windows never appear. Not merely refused when aimed at —
	// absent, so a model never spends a turn discovering it may not.
	kept := out[:0]
	for _, w := range out {
		if w.PID == self {
			continue
		}
		kept = append(kept, w)
	}
	return kept, nil
}

func windowTitle(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	got, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if got == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:got])
}

// processImage answers the full path of the program that owns a pid, or "" when
// it cannot be read. Empty is a meaningful answer here rather than an error:
// appTier refuses an unidentified window, so a process this user may not query
// (a service, something at higher integrity) is refused by the same rule that
// refuses a nameless one, which is the correct outcome for both.
func processImage(pid uint32) string {
	const queryLimitedInformation = 0x1000
	h, err := windows.OpenProcess(queryLimitedInformation, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

func errnoOf(err error) uintptr {
	var e syscall.Errno
	if ok := asErrno(err, &e); ok {
		return uintptr(e)
	}
	return 0
}

func asErrno(err error, out *syscall.Errno) bool {
	if e, ok := err.(syscall.Errno); ok {
		*out = e
		return true
	}
	return false
}

// reachFindWindow re-resolves a window handle the model was given earlier. A
// window that closed between one call and the next is the ordinary case, not an
// exceptional one, so it gets a sentence rather than a stack trace.
func reachFindWindow(hwnd uintptr) (reachTarget, error) {
	if vis, _, _ := procIsWindowVisible.Call(hwnd); vis == 0 {
		return reachTarget{}, win32Error{call: "IsWindowVisible", code: winInvalidWindowHandle}
	}
	var pid uint32
	procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return reachTarget{}, win32Error{call: "GetWindowThreadProcessId", code: winInvalidWindowHandle}
	}
	return reachTarget{
		HWND:  hwnd,
		PID:   int32(pid),
		Exe:   processImage(pid),
		Title: windowTitle(hwnd),
	}, nil
}

// ---------------------------------------------------------------------------
// Reading
// ---------------------------------------------------------------------------

// reachReadWindow walks one window's accessibility tree and returns the parts a
// person could act on, numbered.
//
// The cap and the renumbering are both copied from browser_read, and for the
// same reasons its own file gives: a tree with no ceiling is a context window
// spent on a file dialog's 400 list items, and refs that survive a redraw are
// refs that point at the wrong thing. Every read renumbers from 1.
func reachReadWindow(hwnd uintptr, filter string, max int) ([]reachNode, int, error) {
	if max <= 0 {
		max = reachReadCap
	}
	var (
		nodes []reachNode
		total int
	)
	err := uia().do(func(a *iUIAutomation) error {
		root, err := a.elementFromHandle(hwnd)
		if err != nil {
			return err
		}
		defer root.release()

		cond, err := a.createTrueCondition()
		if err != nil {
			return err
		}
		defer cond.release()

		arr, err := root.findAll(scopeDescendants, cond)
		if err != nil {
			return err
		}
		defer arr.release()

		n, err := arr.length()
		if err != nil {
			return err
		}
		for i := int32(0); i < n; i++ {
			el, err := arr.at(i)
			if err != nil {
				return err
			}
			node, keep := describeElement(el, len(nodes)+1)
			el.release()
			if !keep {
				continue
			}
			if !matchesFilter(node, filter) {
				continue
			}
			total++
			if len(nodes) >= max {
				continue
			}
			node.Ref = len(nodes) + 1
			nodes = append(nodes, node)
		}
		return nil
	})
	return nodes, total, err
}

// describeElement turns one UIA element into a row, or says to skip it.
//
// Skipping is most of the work. An accessibility tree contains every pane,
// group and separator a framework felt like exposing, and none of them can be
// clicked or typed into. What survives is what carries a name a person could
// point at, or a verb a person could use.
func describeElement(el *iUIAutomationElement, nextRef int) (reachNode, bool) {
	offscreen, err := el.isOffscreen()
	if err != nil || offscreen {
		return reachNode{}, false
	}
	name, _ := el.name()
	role, _ := el.localizedControlType()
	ctrl, _ := el.controlType()

	interesting := reachActionable[ctrl]
	if name == "" && !interesting {
		return reachNode{}, false
	}
	// A text element with no name is furniture; one with a name is what the
	// window is saying, and a model reading a dialog needs to see it.
	if !interesting && ctrl != ctrlText && ctrl != ctrlDocument {
		return reachNode{}, false
	}

	node := reachNode{
		Ref:  nextRef,
		Name: name,
		Kind: ctrl,
		Role: role,
	}
	if id, err := el.runtimeID(); err == nil {
		node.RuntimeID = id
	}
	if enabled, err := el.isEnabled(); err == nil {
		node.Enabled = enabled
	} else {
		node.Enabled = true
	}
	if pw, err := el.isPassword(); err == nil {
		node.Password = pw
	}
	if ctrl == ctrlEdit || ctrl == ctrlDocument || ctrl == ctrlComboBox {
		// The current contents, so a model can tell an empty field from a full
		// one without typing into it to find out.
		if v, err := el.currentValue(); err == nil && !node.Password {
			node.Value = v
		}
	}
	return node, true
}

func matchesFilter(n reachNode, filter string) bool {
	f := normalizeFilter(filter)
	if f == "" {
		return true
	}
	return containsFold(n.Name, f) || containsFold(n.Role, f) || containsFold(n.Value, f)
}

// Control type ids this reach cares about. UIAutomationClient.h.
const (
	ctrlButton      = 50000
	ctrlCheckBox    = 50002
	ctrlComboBox    = 50003
	ctrlEdit        = 50004
	ctrlHyperlink   = 50005
	ctrlListItem    = 50007
	ctrlMenuItem    = 50011
	ctrlRadioButton = 50013
	ctrlSlider      = 50015
	ctrlTabItem     = 50019
	ctrlText        = 50020
	ctrlTreeItem    = 50024
	ctrlSplitButton = 50031
	ctrlDocument    = 50030
)

// reachActionable is the "a person could do something with this" set. Deliberately
// short: a list that admits panes and groups is a list that renumbers into the
// hundreds and buries the six controls that matter.
var reachActionable = map[int32]bool{
	ctrlButton: true, ctrlCheckBox: true, ctrlComboBox: true, ctrlEdit: true,
	ctrlHyperlink: true, ctrlListItem: true, ctrlMenuItem: true,
	ctrlRadioButton: true, ctrlSlider: true, ctrlTabItem: true,
	ctrlTreeItem: true, ctrlSplitButton: true, ctrlDocument: true,
}

// ---------------------------------------------------------------------------
// Capturing
// ---------------------------------------------------------------------------

// reachCaptureWindow photographs one window by asking it to draw itself.
//
// PW_RENDERFULLCONTENT is what makes this work on anything modern: without it
// a window that renders through DirectComposition (which is most of them now)
// prints as a blank rectangle — the same blank-image failure browser_shot.go
// documents for CapturePreview.
func reachCaptureWindow(hwnd uintptr) ([]byte, error) {
	if min, _, _ := procIsIconic.Call(hwnd); min != 0 {
		return nil, refuse(
			"หน้าต่างนี้ถูกย่อเก็บอยู่ จึงไม่มีอะไรให้ถ่าย",
			"ใช้ `focus` เพื่อเรียกมันขึ้นมาก่อน")
	}
	var r win32Rect
	if ok, _, err := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return nil, win32Error{call: "GetWindowRect", code: errnoOf(err)}
	}
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if w <= 0 || h <= 0 {
		return nil, win32Error{call: "GetWindowRect", code: winInvalidWindowHandle}
	}

	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		return nil, win32Error{call: "GetDC", code: winAccessDenied}
	}
	defer procReleaseDC.Call(0, screenDC)

	memDC, _, err := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return nil, win32Error{call: "CreateCompatibleDC", code: errnoOf(err)}
	}
	defer procDeleteDC.Call(memDC)

	// Top-down 32-bit DIB: negative height means row 0 is the top row, which is
	// the order image.RGBA wants, so the copy below is a straight walk.
	type bitmapInfoHeader struct {
		Size                    uint32
		Width, Height           int32
		Planes, BitCount        uint16
		Compression, SizeImage  uint32
		XPelsPerMeter, YPelsPer int32
		ClrUsed, ClrImportant   uint32
	}
	bi := struct {
		Header bitmapInfoHeader
		_      [3]uint32
	}{Header: bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       int32(w),
		Height:      int32(-h),
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}}

	var bits unsafe.Pointer
	bmp, _, err := procCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bi)),
		dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return nil, win32Error{call: "CreateDIBSection", code: errnoOf(err)}
	}
	defer procDeleteObject.Call(bmp)

	old, _, _ := procSelectObject.Call(memDC, bmp)
	defer procSelectObject.Call(memDC, old)

	if ok, _, err := procPrintWindow.Call(hwnd, memDC, pwRenderFullContent); ok == 0 {
		return nil, win32Error{call: "PrintWindow", code: errnoOf(err)}
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	src := unsafe.Slice((*byte)(bits), w*h*4)
	for i := 0; i < w*h; i++ {
		// The DIB is BGRA with an alpha channel most windows leave at zero.
		// Forcing it opaque is not a shortcut: a picture whose every pixel is
		// transparent is the blank image this function exists to avoid, and it
		// arrives looking exactly like a window that drew nothing.
		img.Pix[i*4+0] = src[i*4+2]
		img.Pix[i*4+1] = src[i*4+1]
		img.Pix[i*4+2] = src[i*4+0]
		img.Pix[i*4+3] = 0xff
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("the picture could not be encoded: %w", err)
	}
	return buf.Bytes(), nil
}
