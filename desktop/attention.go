package main

// เรียกให้หัน — how the window reaches a person who is not looking at it.
//
// Owner, 12 ก.ย. 2026: *"เวลาเอเจนถาม มันไม่มีอะไรแจ้งเตือนเลย เซสชั่นนั้นจะนิ่งและ
// เงียบไป ... ส่วนใหญ่มันชอบเงียบแล้วค้างเองแทนที่จะแจ้งเตือน"*. Every signal
// the app had lived inside the window: a dot on a row in the sidebar, a card in
// one chat's transcript. A person in another program, or with the window
// minimised, or reading a different chat, was told nothing — and `ask_user`
// waits with no deadline (internal/turn/executor.go, noDeadlineTools), so a
// question nobody was told about is a turn that sits forever.
//
// Owner again, 13 ก.ย., a day after the first two signals shipped: *"เวลาเราทิ้ง
// ให้มันทำงานมันไม่มีแจ้งเตือนครับ คนไม่รู้"*. A taskbar button flashing and two
// quiet notes are signals a person has to already know about; the one signal
// everybody on Windows recognises as แจ้งเตือน is the notification in the
// corner, which §251 had left unbuilt. It is the third switch now.
//
// Three signals, all switches, all shipped on:
//
//   - **flash** — the taskbar button flashes until the window is brought to the
//     front. Asked of the operating system only when the window is NOT already
//     in front, because flashing the window somebody is looking at is noise.
//   - **chime** — a short tone, played by the window itself (WebAudio, no
//     asset). Go only holds the switch: a tone needs no OS call and the window
//     is where the event that earns it arrives.
//   - **toast** — a Windows notification naming the chat and what it wants
//     (attention_windows.go). Only when the window is not in front, and
//     pressing it brings the window back. It goes through Shell_NotifyIcon
//     rather than WinRT so it costs no COM, no package identity and no
//     PowerShell; the tray icon it needs exists only while there is something
//     unseen, and goes away the moment the window is in front again.
//
// Whether the window is in front is decided HERE, from GetForegroundWindow,
// and handed back to the window: the frontend used to decide it from
// document.hasFocus() and only then ask for the flash, so a webview that was
// wrong about its own focus kept every signal in. The window still has its own
// guess and uses whichever of the two says "away" — a signal one side missed
// is worse than one both sides raise.
//
// No in-window toast. The app has none by design (see SlidesPane.svelte's note:
// a notice that vanishes before it is read is worse than none), and what tells
// the user WHICH chat is asking is the row and the strip in the window.
//
// Same shape as the busy signal next door: a preference read straight from the
// file, no re-bootstrap, because none of this reaches the engine.

import (
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/engine"
)

// The three ids, spelled once.
const (
	attentionFlash = "flash"
	attentionChime = "chime"
	attentionToast = "toast"
)

// AttentionSignal reports the switches in the order they are shown.
func (a *App) AttentionSignal() []engine.BusyLayer {
	pref, _, _ := config.LoadModelPreference()
	toastNote := "การแจ้งเตือนที่มุมจอ บอกว่าแชตไหนถาม ขออนุญาต หรือทำงานเสร็จ ตอนหน้าต่างไม่ได้อยู่ข้างหน้า กดที่การแจ้งเตือนเพื่อกลับมา"
	if windowsNotificationsOff() {
		// The switch here is not the only one: with Windows' own master
		// switch off, nothing this app sends is ever drawn, and a row that
		// said "on" over that would be the owner's 13 ก.ย. complaint again —
		// found on the machine this was built on, where the test notification
		// went nowhere until the registry said why.
		toastNote += " — ตอนนี้ Windows ปิดการแจ้งเตือนไว้ทั้งเครื่อง จึงยังไม่ขึ้น เปิดได้ที่ ตั้งค่า Windows › ระบบ › การแจ้งเตือน"
	}
	return []engine.BusyLayer{
		{
			ID:    attentionToast,
			Label: "แจ้งเตือนของ Windows",
			Note:  toastNote,
			On:    !pref.AttentionToastOff,
		},
		{
			ID:    attentionFlash,
			Label: "กระพริบบนแถบงาน",
			Note:  "ปุ่มของ Aetox บนแถบงานกระพริบเมื่อเอเจนต์ถาม ขออนุญาต หรือทำงานเสร็จตอนหน้าต่างไม่ได้อยู่ข้างหน้า",
			On:    !pref.AttentionFlashOff,
		},
		{
			ID:    attentionChime,
			Label: "เสียงเตือน",
			Note:  "เสียงสั้น ๆ เมื่อเอเจนต์ถาม ขออนุญาต หรือทำงานเสร็จในแชตที่ไม่ได้เปิดอยู่",
			On:    !pref.AttentionChimeOff,
		},
	}
}

// SetAttentionSignal turns one switch on or off and writes it down. An unknown
// id is ignored rather than guessed at, the same rule SetBusyLayer follows.
func (a *App) SetAttentionSignal(id string, on bool) []engine.BusyLayer {
	switch strings.TrimSpace(id) {
	case attentionFlash:
		_ = config.UpdateModelPreference(func(pref *config.ModelPreference) error {
			pref.AttentionFlashOff = !on
			return nil
		})
	case attentionChime:
		_ = config.UpdateModelPreference(func(pref *config.ModelPreference) error {
			pref.AttentionChimeOff = !on
			return nil
		})
	case attentionToast:
		_ = config.UpdateModelPreference(func(pref *config.ModelPreference) error {
			pref.AttentionToastOff = !on
			return nil
		})
	}
	return a.AttentionSignal()
}

// RequestAttention asks the operating system to draw the user back to the
// window, and answers whether it had to: true when the window was not in
// front, which is the window's cue to sound the chime for a turn it would
// otherwise have thought was watched.
//
// `kind` is "ask" or "done"; `title` names the chat and `text` says what it
// wants, both already in the user's language — the notification is the one
// signal here that carries words. Every switch that is off, a window already
// in front, and a platform with no taskbar or notification centre are silent
// successes, because the caller has nothing to do about any of them.
func (a *App) RequestAttention(kind, title, text string) bool {
	if !windowAway() {
		return false
	}
	pref, _, _ := config.LoadModelPreference()
	if !pref.AttentionFlashOff {
		flashOwnWindow()
	}
	if !pref.AttentionToastOff {
		showAttentionToast(kind, title, text)
	}
	return true
}
