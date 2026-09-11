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
// Two signals, both switches, both shipped on:
//
//   - **flash** — the taskbar button flashes until the window is brought to the
//     front. Asked of the operating system only when the window is NOT already
//     in front, because flashing the window somebody is looking at is noise.
//     The frontend also checks document.hasFocus() before asking; the check
//     here is the one that holds when the webview is confused about focus.
//   - **chime** — a short tone, played by the window itself (WebAudio, no
//     asset). Go only holds the switch: a tone needs no OS call and the window
//     is where the event that earns it arrives.
//
// No toast. The app has none by design (see SlidesPane.svelte's note: a notice
// that vanishes before it is read is worse than none), and what tells the user
// WHICH chat is asking is the row and the strip in the window — the flash only
// says "come back".
//
// Same shape as the busy signal next door: a preference read straight from the
// file, no re-bootstrap, because none of this reaches the engine.

import (
	"github.com/Mikedev115/Aetox/internal/engine"
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
)

// The two ids, spelled once.
const (
	attentionFlash = "flash"
	attentionChime = "chime"
)

// AttentionSignal reports both switches in the order they are shown.
func (a *App) AttentionSignal() []engine.BusyLayer {
	pref, _, _ := config.LoadModelPreference()
	return []engine.BusyLayer{
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
	}
	return a.AttentionSignal()
}

// RequestAttention asks the operating system to draw the user back to the
// window. A no-op when the switch is off, when the window is already in front,
// or on a platform that has no taskbar to flash — every one of those is a
// silent success, because the caller has nothing to do about any of them.
func (a *App) RequestAttention() {
	pref, _, _ := config.LoadModelPreference()
	if pref.AttentionFlashOff {
		return
	}
	flashOwnWindow()
}
