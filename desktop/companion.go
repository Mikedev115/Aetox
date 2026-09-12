package main

// The assistant's presence, published for the body outside this window.
//
// The mascot that sits on the screen (frontend, lib/mascot/Companion.svelte)
// is drawn inside the app's own webview, and the owner wants it able to
// leave it (12 ก.ย. 2026: "ออกไปโลดแล่นบนจอส่วนอื่น เหมือน Codex") — without a
// second WebView2 ("กินแรมเยอะโดยไม่จำเป็น"). So the figure on the desktop is
// a window of Go's own (companion_windows.go), drawn from frames the app
// bakes (companion_sprites.go, lib/mascot/bake.ts), and this file is what
// the two sides share:
//
//   - the STATE: what the assistant is doing and saying right now, and what
//     it looks like. The window's presence (presence.ts) already knows; it
//     pushes here through SetCompanionState on every change and the body is
//     told at once. Nothing is computed twice — the body is a mirror of the
//     one presence the app has.
//   - the DOORS: OpenCompanionWindow / CloseCompanionWindow for the switch,
//     CompanionSpriteKeys / CompanionSprites for the frames, and the event
//     the body's clicks and drags come back on (companionInputEvent).
//
// An earlier layer (a83be484) served the state over a loopback HTTP feed to
// a page a second webview would load; that surface was decided against and
// the feed went with it — the state is in memory and the body is in process.

import (
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// CompanionPrefs is the look the user chose (ตั้งค่า › อวตาร): the same four
// dials avatarPrefs.svelte.ts keeps, every one a catalogue row id. Blank is
// that catalogue's default.
type CompanionPrefs struct {
	Shell  string `json:"shell"`
	Accent string `json:"accent"`
	Top    string `json:"top"`
	Face   string `json:"face"`
}

// CompanionState is one report from the window: the pose presence derived,
// the headline the bubble shows (empty when it is shut), whether the
// companion is switched on at all, and the look. Seq counts reports so a
// poller can tell "nothing changed" from "changed back".
//
// The desktop body (companion_windows.go) draws from the rest: the bubble's
// text as typed out so far and its words (Intl.Segmenter's — Go cannot break
// Thai), whether a cursor blinks after it, the theme's colours, the voice
// switch, and Hop, a count of clicks reacted to, so a reaction the brain
// decided on hops the body too.
type CompanionState struct {
	Pose   string         `json:"pose"`
	Report string         `json:"report"`
	On     bool           `json:"on"`
	Prefs  CompanionPrefs `json:"prefs"`
	Seq    int64          `json:"seq"`
	At     time.Time      `json:"at"`

	Shown  string         `json:"shown,omitempty"`
	Words  []string       `json:"words,omitempty"`
	Cursor bool           `json:"cursor,omitempty"`
	Theme  CompanionTheme `json:"theme"`
	Muted  bool           `json:"muted,omitempty"`
	Hop    int            `json:"hop,omitempty"`
	// Size is the figure's logical size, the user's (companionSetting size);
	// 0 is the default. The body scales its frames by it.
	Size int `json:"size,omitempty"`
}

type companionServer struct {
	mu    sync.Mutex
	state CompanionState
	// body is the companion's window on the desktop while there is one —
	// a Win32 layered window on Windows (companion_windows.go), nothing on
	// the other platforms yet. Nil when the companion is inside the app.
	body    companionBody
	opening bool
	// stores hold the baked frames, one per set the window has named — a
	// set is one look at one scale (companion_sprites.go). Every monitor's
	// scale the figure has visited stays resident, so crossing back is not
	// a reload; rig is the look's half of the latest hash, and a set of the
	// same look at another scale stands in while this scale's is baking.
	stores map[string]*spriteStore
	rig    string
}

// companionBody is the desktop window as this file needs to know it: it can
// be told the state and be closed. The platform files provide
// openCompanionBody, handing the body where frames come from and where the
// user's doings go.
type companionBody interface {
	apply(CompanionState)
	close()
}

// companionInputEvent is what the body's doings reach the window as:
// `kind` is click · dragStart · dragEnd · hide · mute · moved (x, y: the
// figure's top-left, physical px) · bake (scale: frames wanted at this
// scale) · resize (size: the figure's new logical size, dragged at the
// corner). The brain in Companion.svelte answers each.
const companionInputEvent = "companion:input"

// OpenCompanionWindow sends the companion out of the app window to (x, y)
// physical pixels on the desktop — negative for "wherever". False when the
// platform cannot, in which case the window keeps drawing its own; the
// reason is in the log, not the answer, because the only thing the switch
// can do with it is fall back.
func (a *App) OpenCompanionWindow(x, y int) bool {
	c := a.companion()
	c.mu.Lock()
	if c.body != nil || c.opening {
		c.mu.Unlock()
		return true
	}
	c.opening = true
	c.mu.Unlock()
	// Not under the lock: the body's thread asks for the sprite store while
	// it comes up, and that goes through the same lock.
	sprites := func(scale float64) spriteSource { return c.spritesAt(scale) }
	on := func(kind string, data map[string]any) {
		if data == nil {
			data = map[string]any{}
		}
		data["kind"] = kind
		a.emitEvent(companionInputEvent, data)
	}
	body, err := openCompanionBody(x, y, sprites, on)
	c.mu.Lock()
	c.opening = false
	if err != nil {
		c.mu.Unlock()
		debuglog.Msg("companion window: %v", err)
		return false
	}
	c.body = body
	state := c.state
	c.mu.Unlock()
	body.apply(state)
	return true
}

// CloseCompanionWindow takes the companion back in: the desktop window is
// destroyed and the app window draws it again. A no-op when there is none.
func (a *App) CloseCompanionWindow() {
	c := a.companion()
	c.mu.Lock()
	body := c.body
	c.body = nil
	c.mu.Unlock()
	if body != nil {
		body.close()
	}
}

func (a *App) companion() *companionServer {
	a.companionOnce.Do(func() { a.companionSrv = &companionServer{} })
	return a.companionSrv
}

// SetCompanionState is the window's report. Called on every change of pose,
// words, look or switch — cheap, in memory — and passed straight to the body
// when there is one; kept for the body that opens later when there is not.
func (a *App) SetCompanionState(s CompanionState) {
	c := a.companion()
	c.mu.Lock()
	defer c.mu.Unlock()
	s.Seq = c.state.Seq + 1
	s.At = time.Now()
	c.state = s
	if c.body != nil {
		c.body.apply(s)
	}
}
