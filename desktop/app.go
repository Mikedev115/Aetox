package main

// The screen (§248 B1).
//
// App is what Wails binds and what the frontend calls, and since Stage B it
// is a window and nothing else: every binding that is engine work forwards to
// the engine (engine_forwarders_gen.go, generated from the engine's exported
// methods), and every binding that needs a window is written here in
// desktop/. Today the engine is a value in this process; the field is what
// becomes a client of a socket in phase 2, and nothing above it has to change
// for that.

import (
	"context"
	"sync"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/tts"
)

type App struct {
	// api is the engine as the screen calls it: every forwarder
	// (engine_forwarders_gen.go), every window tool and every door here goes
	// through this interface and nothing else. In phase 2 it is the RPC
	// client; a test hands in an engine with one answer changed.
	api engine.API
	// eng is the same engine as a value in this process — what the lifecycle
	// hooks (engine/lifecycle.go) still take, and the last thing that knows
	// the engine is in-process. Goes with phase 2.
	eng *engine.Engine
	// ctx is the window's lifetime — what the Wails runtime is called with:
	// dialogs, window sizing, Quit. Nil until startup has run.
	ctx context.Context
	// openDir stands in for openInFileManager, the one door out to the OS file
	// manager, so a test can watch a reveal happen without a window appearing
	// on somebody's desk.
	openDir func(string) error
	// emit stands in for wailsruntime.EventsEmit — see emitEvent.
	emit func(event string, data ...any)

	staged stagedUpdate
	// exports is what the deck export wrote into Downloads this session
	// (exports.go), so OpenExport can open it and nothing else.
	exports exports

	// The browser tab host (browser.go) and the machine lock
	// (computer_guard.go): both act on this window's computer, which is why
	// they are the screen's and not the engine's.
	browsersMu sync.Mutex
	browsers   *browserHost
	// driving is which chat, if any, is currently driving programs on the
	// machine. On the App rather than on a conversation because the thing it
	// protects is the machine, and there is one of those: two chats clicking
	// in one window produce a state neither of them predicted. Held for the
	// length of an acting call, never for a session.
	driving screenLock

	// The reply's ฟัง button (speak.go, voice.go): heard where the window is,
	// so the synthesizer and the voices it picks from are the screen's.
	//
	// ttsVoiceMu guards ttsVoiceCache — the settings page enumerates voices
	// while a SpeakText on another goroutine resolves its default from the
	// same list.
	ttsVoiceMu sync.Mutex
	// ttsVoiceCache is the installed-voices list per TTS engine id, cached for
	// the process: enumerating costs a PowerShell run (~1s), and the set only
	// changes when the user installs a voice into Windows — a restart after
	// that is acceptable, a second of extra latency on every ฟัง press is not.
	ttsVoiceCache map[string][]tts.Voice
	// speakMu guards speakJobs — StartSpeech, StopSpeech, the reader goroutine
	// and the asset-server handler all reach it, and the handler runs on the
	// webview's own thread.
	speakMu sync.Mutex
	// speakJobs is every read-aloud in flight, by job id. Normally at most one
	// (a second press of ฟัง stops the first), but the map is the registry the
	// URL host authorizes against, so it is keyed rather than a single field:
	// a stopped job must become unfindable the instant it stops, and deleting
	// a key is that. See speak.go.
	speakJobs map[string]*speechJob
}

// NewApp builds the screen and the engine that talks to it.
func NewApp() *App {
	a := &App{}
	a.eng = engine.NewEngine(appScreen{a})
	a.api = a.eng
	return a
}

// The four Wails lifecycle hooks and the asset middleware, wired in main.go.
// Unexported, as they were: not bindings, not for the frontend.

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Before anything else: the window is created and centred by the time
	// startup runs, and shown only once the webview has content, so a window
	// bigger than the screen is corrected while nobody can see it move.
	a.fitToScreen()
	engine.Startup(a.eng, ctx)
	// The previous build's exe, renamed aside by a self-update, and the staging
	// download — this build is running, so by definition neither is needed.
	// Except the download the user has not restarted into yet; see
	// adoptStagedUpdate.
	go a.adoptStagedUpdate()
	// And the other end of the same feature: ask whether a newer build exists,
	// so the answer reaches the user without them going looking for it
	// (update_notify.go).
	go a.watchForUpdates()
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) { return engine.BeforeClose(a.eng, ctx) }

func (a *App) shutdown(ctx context.Context) {
	engine.Shutdown(a.eng, ctx)
	// A read-aloud in flight owns a temp folder and a synthesizer. Neither
	// survives the process, but the folder would (speak.go).
	a.stopAllSpeech()
}
