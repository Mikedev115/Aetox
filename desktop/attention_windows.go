//go:build windows

package main

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"golang.org/x/sys/windows/registry"
)

var (
	procFlashWindowEx    = user32.NewProc("FlashWindowEx")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
)

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

// windowAway answers whether the user is somewhere other than this app: the
// foreground window belongs to another process, or there is none (the lock
// screen). Any window of our own in front — the main window, a file dialog it
// opened, a detached browser tab — is the user looking at us, and nothing is
// owed. No main window at all (a test process) is "not away": there is
// nothing to come back to.
func windowAway() bool {
	if findOwnMainWindow() == 0 {
		return false
	}
	fg, _, _ := procGetForegroundWindow.Call()
	if fg == 0 {
		return true
	}
	var pid uint32
	procGetWindowThreadProcessID.Call(fg, uintptr(unsafe.Pointer(&pid)))
	return pid != uint32(os.Getpid())
}

// flashOwnWindow flashes the main window's taskbar button until it is brought
// to the front. Not when it already is: RequestAttention asks only when the
// window is away, and this is the second check for the case where that answer
// and the desktop's disagree.
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

// ---------------------------------------------------------------------------
// The notification
// ---------------------------------------------------------------------------
//
// Shell_NotifyIcon's balloon, which Windows 10 and 11 draw as an ordinary
// notification in the corner and keep in the notification centre. Chosen over
// the WinRT toast API because it needs none of what that needs — no COM
// activation from Go, no package identity or Start-menu shortcut carrying an
// AppUserModelID, no PowerShell spawned per notification with its console
// flashing over the app (git_badge.go's lesson). It needs one thing: a tray
// icon to hang off. So the icon exists only while there is something the user
// has not seen, on a message-only window of its own thread, and is taken down
// the moment the main window is in front — the same "until you look" rule the
// flash follows. Deleting the icon also dismisses its notification, which is
// why it is not deleted any sooner: a person who comes back an hour later
// should still find it in the centre.

const (
	nimAdd        = 0
	nimModify     = 1
	nimDelete     = 2
	nimSetVersion = 4

	nifMessage = 0x01
	nifIcon    = 0x02
	nifTip     = 0x04
	nifInfo    = 0x10

	// NIIF_USER with NIIF_LARGE_ICON: the app's own icon on the notification,
	// from hBalloonIcon. NIIF_NOSOUND: the chime is the app's sound, and it is
	// a switch of its own — a second sound from the shell would ignore it.
	// NIIF_RESPECT_QUIET_TIME: none of this during the first hour after a
	// sign-in, which is what the flag means.
	niifUser             = 0x04
	niifNoSound          = 0x10
	niifLargeIcon        = 0x20
	niifRespectQuietTime = 0x80

	// NOTIFYICON_VERSION_4: the callback carries the event in LOWORD(lParam)
	// and NIN_* codes are delivered at all.
	notifyIconVersion4 = 4

	// WM_USER + n, the NIN_ events this cares about.
	ninBalloonUserClick = 0x0400 + 5

	// The tray's callback message, private to this window (wmApp is the work
	// wake-up, as it is on every other message loop in this package).
	wmAttentionTray = wmApp + 0x11

	// HWND_MESSAGE: a window that receives messages and is never shown.
	hwndMessage = ^uintptr(2)

	// IDI_APPLICATION, for a build with no icon resource of its own.
	idiApplication = 32512
)

// notifyIconDataW is NOTIFYICONDATAW at NOTIFYICON_VERSION_4. Field order and
// widths are the Win32 struct's; attention_layout_test.go pins the size.
type notifyIconDataW struct {
	CbSize           uint32
	Hwnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32 // a union with uTimeout, which nothing here sets
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

// attentionTray is the one tray icon and the thread that owns it.
type attentionTray struct {
	once  sync.Once
	ready chan error
	work  chan func()
	hwnd  uintptr // the message-only window; every Shell_NotifyIcon call names it
	small uintptr // HICON for the tray
	large uintptr // HICON for the notification itself
	// Only the tray thread reads or writes these two.
	shown    bool
	watching bool
}

var tray = &attentionTray{ready: make(chan error, 1), work: make(chan func(), 16)}

// showAttentionToast puts up one notification. Starting the thread the first
// time is part of the call; a thread that could not start is logged once and
// every later call is a silent success, which is the contract RequestAttention
// gives its callers.
func showAttentionToast(kind, title, text string) {
	tray.once.Do(func() {
		go tray.run()
		if err := <-tray.ready; err != nil {
			debuglog.Msg("attention: tray thread did not start: %v", err)
			tray.hwnd = 0
		}
	})
	if tray.hwnd == 0 {
		return
	}
	tray.post(func() { tray.show(kind, title, text) })
}

// post runs fn on the tray thread. Dropped rather than blocked when the queue
// is full: a burst of sixteen unanswered notifications is not a state worth
// stalling a turn's ending over.
func (t *attentionTray) post(fn func()) {
	select {
	case t.work <- fn:
		procPostMessageW.Call(t.hwnd, wmApp, 0, 0)
	default:
	}
}

func (t *attentionTray) drain() {
	for {
		select {
		case fn := <-t.work:
			fn()
		default:
			return
		}
	}
}

// run is the tray thread: a message-only window whose only messages are the
// tray callback and the work wake-up.
func (t *attentionTray) run() {
	runtime.LockOSThread()
	for _, p := range []*syscall.LazyProc{procShellNotifyIconW, procRegisterClassExW, procCreateWindowExW, procGetMessageW, procPostMessageW} {
		if err := p.Find(); err != nil {
			t.ready <- err
			return
		}
	}
	className, _ := syscall.UTF16PtrFromString("AetoxAttention")
	wc := wndClassExW{
		Size:      uint32(unsafe.Sizeof(wndClassExW{})),
		WndProc:   syscall.NewCallback(t.wndProc),
		ClassName: className,
	}
	if atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		t.ready <- err
		return
	}
	tip, _ := syscall.UTF16PtrFromString("Aetox")
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(tip)),
		0, 0, 0, 0, 0, hwndMessage, 0, 0, 0)
	if hwnd == 0 {
		t.ready <- err
		return
	}
	t.hwnd = hwnd
	t.small, t.large = ownIcons()
	t.ready <- nil

	var msg winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			return
		}
		t.drain()
		if msg.Message != wmApp {
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func (t *attentionTray) wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == wmAttentionTray {
		// Pressing the notification, or the icon itself: the person is asking
		// to come back, so bring the window up and take the icon down —
		// arriving is what the icon was waiting for.
		switch lParam & 0xFFFF {
		case ninBalloonUserClick, wmLButtonUp:
			if main := findOwnMainWindow(); main != 0 {
				_ = reachFocusWindow(main)
			}
			t.remove()
		}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// show adds the icon if it is not up and hangs the notification on it. On the
// tray thread.
func (t *attentionTray) show(kind, title, text string) {
	if !t.shown {
		add := t.data()
		add.UFlags = nifMessage | nifIcon | nifTip
		add.UCallbackMessage = wmAttentionTray
		add.HIcon = t.small
		copyUTF16(add.SzTip[:], "Aetox")
		if ok, _, err := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&add))); ok == 0 {
			debuglog.Msg("attention: NIM_ADD failed: %v", err)
			return
		}
		ver := t.data()
		ver.UVersion = notifyIconVersion4
		procShellNotifyIconW.Call(nimSetVersion, uintptr(unsafe.Pointer(&ver)))
		t.shown = true
		t.watch()
	}
	info := t.data()
	info.UFlags = nifInfo
	info.DwInfoFlags = niifUser | niifLargeIcon | niifNoSound | niifRespectQuietTime
	info.HBalloonIcon = t.large
	copyUTF16(info.SzInfoTitle[:], title)
	copyUTF16(info.SzInfo[:], text)
	if ok, _, err := procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&info))); ok == 0 {
		debuglog.Msg("attention: NIM_MODIFY (%s) failed: %v", kind, err)
	}
}

// remove takes the icon down, and its notifications with it. On the tray
// thread.
func (t *attentionTray) remove() {
	if !t.shown {
		return
	}
	del := t.data()
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&del)))
	t.shown = false
}

// watch takes the icon down once the user is back — polled, because the main
// window is wails's and its activation is not ours to hook. Half a second is
// well under the time it takes to read a notification and far above the cost
// of one GetForegroundWindow. One watcher at a time; it ends itself when the
// icon is gone, however it went.
func (t *attentionTray) watch() {
	if t.watching {
		return
	}
	t.watching = true
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			done := make(chan bool, 1)
			t.post(func() {
				if t.shown && !windowAway() {
					t.remove()
				}
				if !t.shown {
					t.watching = false
				}
				done <- !t.shown
			})
			select {
			case gone := <-done:
				if gone {
					return
				}
			case <-time.After(5 * time.Second):
				// The thread is not answering; nothing here can help it.
			}
		}
	}()
}

func (t *attentionTray) data() notifyIconDataW {
	return notifyIconDataW{
		CbSize: uint32(unsafe.Sizeof(notifyIconDataW{})),
		Hwnd:   t.hwnd,
		UID:    1,
	}
}

// ownIcons is the executable's own icon in the two sizes the shell asks for,
// falling back to the stock application icon for a build that carries none
// (a bare `go build` of the dev binary). Kept for the life of the process.
func ownIcons() (small, large uintptr) {
	if exe, err := os.Executable(); err == nil && procExtractIconExW.Find() == nil {
		if wide, err := syscall.UTF16PtrFromString(exe); err == nil {
			procExtractIconExW.Call(uintptr(unsafe.Pointer(wide)), 0,
				uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&small)), 1)
		}
	}
	if small == 0 || large == 0 {
		stock, _, _ := procLoadIconW.Call(0, idiApplication)
		if small == 0 {
			small = stock
		}
		if large == 0 {
			large = stock
		}
	}
	return small, large
}

// windowsNotificationsOff reads the master switch behind ตั้งค่า Windows ›
// ระบบ › การแจ้งเตือน. Off, Shell_NotifyIcon still answers success and draws
// nothing, so this is the only way the settings row can tell the truth. A
// missing value is the default, which is on.
func windowsNotificationsOff() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\PushNotifications`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("ToastEnabled")
	return err == nil && v == 0
}

// copyUTF16 writes s into a fixed NUL-terminated buffer, cut to fit. Cut on a
// code unit rather than a rune is fine for the shell: it renders what it can
// read and stops at the NUL.
func copyUTF16(dst []uint16, s string) {
	wide, err := syscall.UTF16FromString(s)
	if err != nil {
		wide = []uint16{0}
	}
	n := copy(dst[:len(dst)-1], wide)
	dst[n] = 0
}
