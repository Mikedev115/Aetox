package main

// The screen (§248 B1, phase 2).
//
// App is what Wails binds and what the frontend calls, and since Stage B it
// is a window and nothing else: every binding that is engine work forwards to
// the engine (engine_forwarders_gen.go, generated from the engine's exported
// methods), and every binding that needs a window is written here in
// desktop/. The engine is a process of its own (cmd/aetox-engine), started
// beside this one and reached over one socket (engine_local.go); `api` is
// the client on that socket, and nothing above it knows.

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/tts"
)

type App struct {
	// api is the engine as the screen calls it: every forwarder
	// (engine_forwarders_gen.go), every window tool and every door here goes
	// through this interface and nothing else. It is the RPC client; a test
	// hands in an engine with one answer changed (engineWith).
	api engine.API
	// client is that same client, as the pieces that speak to the engine
	// outside the bindings need it: the supervisor, the file proxy.
	client *rpc.Client
	// engine is the child process and the wire to it (engine_local.go). Nil
	// in a test, whose engine is in the same process behind a loopback
	// listener (apptest_test.go).
	engine *localEngine
	// ctx is the window's lifetime — what the Wails runtime is called with:
	// dialogs, window sizing, Quit. Nil until startup has run.
	ctx context.Context
	// openDir stands in for openInFileManager, the one door out to the OS file
	// manager, so a test can watch a reveal happen without a window appearing
	// on somebody's desk.
	openDir func(string) error
	// emit stands in for wailsruntime.EventsEmit — see emitEvent.
	emit func(event string, data ...any)
	// reload stands in for wailsruntime.WindowReloadApp — see reloadWindow.
	reload func()

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

// NewApp builds the screen and the client its engine will be reached
// through. The engine itself starts in startup, once the window has a
// lifetime; a binding called before it is up waits for the wire
// (rpc.Client) rather than failing.
func NewApp() *App {
	a := &App{}
	a.client = a.newClient()
	a.api = a.client
	token, err := rpc.NewToken()
	if err != nil {
		panic("aetox: no random token for the engine: " + err.Error())
	}
	a.engine = newLocalEngine(a, a.client)
	a.engine.token = token
	return a
}

// newClient is the wire's screen end, with this window's whole Screen
// served on it: events to the frontend, the provider signer, the window
// tools, the desk questions (rpc.ServeScreen).
func (a *App) newClient() *rpc.Client {
	c := rpc.NewClient(rpc.ClientOptions{
		OnEvent: func(name string, data json.RawMessage) { a.emitEvent(name, data) },
		OnFailure: func(method string, err error) {
			a.emitEvent("engine:status", EngineStatus{State: engineReconnecting, Detail: method + ": " + err.Error()})
		},
	})
	rpc.ServeScreen(c, appScreen{a})
	return c
}

// The four Wails lifecycle hooks and the asset middleware, wired in main.go.
// Unexported, as they were: not bindings, not for the frontend.

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Before anything else: the window is created and centred by the time
	// startup runs, and shown only once the webview has content, so a window
	// bigger than the screen is corrected while nobody can see it move.
	a.fitToScreen()
	// The screen's own log, named for the process: the engine writes its own
	// beside it (debuglog.InitAs says why the name matters).
	if dataRoot, err := config.DataRoot(); err == nil {
		debuglog.InitAs(dataRoot, "desktop")
	}
	// The engine, as a child of this window for as long as the window lives.
	go a.engine.run(ctx)
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

// beforeClose is the X, Quit and the update's restart: the engine is asked
// to stop every turn and write its ending before the window goes — the
// same grace the in-process engine took, across the wire, and bounded here
// too so a wire that is down cannot hold the window open.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if conn := a.client.Conn(); conn != nil {
		wait, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		_ = conn.Call(wait, "PrepareToClose", nil, nil)
	}
	return false
}

func (a *App) shutdown(context.Context) {
	// The child is told to leave — stdin closed is its cue — and waited for,
	// so the store it holds is closed before this process is gone.
	if a.engine != nil {
		a.engine.shutdown()
	}
	// A read-aloud in flight owns a temp folder and a synthesizer. Neither
	// survives the process, but the folder would (speak.go).
	a.stopAllSpeech()
}
