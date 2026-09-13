//go:build windows && (amd64 || arm64)

package main

import (
	"testing"
	"unsafe"
)

// NOTIFYICONDATAW at NOTIFYICON_VERSION_4 is 976 bytes on a 64-bit Windows.
// The shell reads cbSize to know which fields exist; a struct one padding
// slot out would be taken for an older version and the notification fields
// silently ignored — no error, no toast.
func TestNotifyIconDataLayout(t *testing.T) {
	if got := unsafe.Sizeof(notifyIconDataW{}); got != 976 {
		t.Fatalf("NOTIFYICONDATAW size = %d, want 976", got)
	}
	if off := unsafe.Offsetof(notifyIconDataW{}.SzInfo); off != 304 {
		t.Errorf("szInfo at %d, want 304", off)
	}
	if off := unsafe.Offsetof(notifyIconDataW{}.HBalloonIcon); off != 968 {
		t.Errorf("hBalloonIcon at %d, want 968", off)
	}
}

// The buffers are fixed and NUL-terminated; a title past 63 code units is cut,
// not overrun, and the cut is where the shell stops reading.
func TestCopyUTF16Cuts(t *testing.T) {
	var buf [8]uint16
	copyUTF16(buf[:], "ยาวเกินไปมาก")
	if buf[7] != 0 {
		t.Error("no NUL at the end")
	}
	if buf[6] == 0 {
		t.Error("cut short of the buffer")
	}
	copyUTF16(buf[:], "ab")
	if buf[0] != 'a' || buf[1] != 'b' || buf[2] != 0 {
		t.Errorf("short string not copied whole: %v", buf)
	}
}
