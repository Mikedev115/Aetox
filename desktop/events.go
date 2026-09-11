package main

import wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

// emitEvent sends a frontend event from the screen's own side — the update
// notices, and in time every event the engine sends, relayed. Through a.emit
// when a test has installed one; wailsruntime.EventsEmit calls log.Fatalf off
// a real Wails context, which a test never has.
//
// A nil ctx is silently nothing rather than a crash: an event with no window
// to reach is not an error.
func (a *App) emitEvent(event string, data ...any) {
	if a.emit != nil {
		a.emit(event, data...)
		return
	}
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, event, data...)
}
