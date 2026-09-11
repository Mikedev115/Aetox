//go:build windows

package main

import (
	"unsafe"
)

var procFlashWindowEx = user32.NewProc("FlashWindowEx")

// flashWinfo is FLASHWINFO. Field order and widths are the Win32 struct's, and
// cbSize has to be filled in or the call returns without doing anything.
type flashWinfo struct {
	cbSize    uint32
	hwnd      uintptr
	dwFlags   uint32
	uCount    uint32
	dwTimeout uint32
}

const (
	// FLASHW_ALL: the caption and the taskbar button both.
	flashwAll = 0x3
	// FLASHW_TIMERNOFG: keep flashing until the window comes to the
	// foreground. The one mode that means "until you look", which is the whole
	// thing being asked for — a fixed count would stop before a person on a
	// long call came back.
	flashwTimerNoFG = 0xC
)

// flashOwnWindow flashes the main window's taskbar button until it is brought
// to the front. Not when it already is: the frontend asks only when
// document.hasFocus() is false, and this is the second check for the case
// where the webview's idea of focus and the desktop's disagree.
func flashOwnWindow() {
	hwnd := findOwnMainWindow()
	if hwnd == 0 {
		return
	}
	if fg, _, _ := procGetForegroundWindow.Call(); fg == hwnd {
		return
	}
	info := flashWinfo{
		cbSize:  uint32(unsafe.Sizeof(flashWinfo{})),
		hwnd:    hwnd,
		dwFlags: flashwAll | flashwTimerNoFG,
	}
	procFlashWindowEx.Call(uintptr(unsafe.Pointer(&info)))
}
