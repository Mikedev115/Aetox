package main

// The assistant's presence, published for a surface that is not this window.
//
// The mascot that sits on the screen (frontend, lib/mascot/Companion.svelte)
// lives inside the app's own webview and so cannot leave it. The owner wants
// it to (12 ก.ย. 2026: "ออกไปโลดแล่นบนจอส่วนอื่น เหมือน Codex"), and the way
// there is a second, transparent, always-on-top WebView2 window of its own —
// the same kind browser_windows.go already opens for the deck export. That
// window needs two things this file provides, and both stand on their own
// before the window exists:
//
//   1. A FEED: what the assistant is doing and saying right now, and what it
//      looks like. The window's presence (presence.ts) already knows; it
//      pushes here through SetCompanionState on every change, and this serves
//      the latest as JSON. Nothing is computed twice — the feed is a mirror of
//      the one presence the app has.
//   2. A PAGE: companion_page.html, generated from the same rig the app draws
//      with (`npm run mascots`, scripts/mascot-sheet.mjs), which polls the feed
//      and draws the mascot. Any surface that can show a web page can show
//      the assistant: the native window to come, or a plain browser tab today.
//
// The listener is loopback only and the path carries a token minted per run:
// a second WebView2 has no access to the app's own asset scheme, so this has
// to be a real port, and a real port on 127.0.0.1 is reachable by any process
// of the same user — the token keeps a stray local page from reading what the
// assistant last said, which is the one thing here worth keeping to ourselves.
// The remote server (remote.go) was not reused on purpose: it is parked, LAN-
// facing, and built around pairing, none of which a local companion wants.

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

//go:embed companion_page.html
var companionPage string

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
type CompanionState struct {
	Pose   string         `json:"pose"`
	Report string         `json:"report"`
	On     bool           `json:"on"`
	Prefs  CompanionPrefs `json:"prefs"`
	Seq    int64          `json:"seq"`
	At     time.Time      `json:"at"`
}

// CompanionFeedInfo is what the window (and a future native window) needs to
// reach the feed: where the page is, and whether anything is listening.
type CompanionFeedInfo struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
	Port    int    `json:"port"`
}

// companionPathPrefix is the URL space, token appended: /c/<token>/…
const companionPathPrefix = "/c/"

type companionServer struct {
	mu    sync.Mutex
	state CompanionState
	token string
	srv   *http.Server
	port  int
	// body is the companion's window on the desktop while there is one —
	// a Win32 layered window on Windows (companion_windows.go), nothing on
	// the other platforms yet. Nil when the companion is inside the app.
	body companionBody
	// store holds the baked frames of the current look and scale
	// (companion_sprites.go); nil until the window names a set.
	store *spriteStore
}

// companionBody is the desktop window as this file needs to know it: it can
// be closed, and (to come) shown a frame and asked where it is. The platform
// files provide openCompanionBody.
type companionBody interface {
	close()
}

// OpenCompanionWindow sends the companion out of the app window to (x, y)
// physical pixels on the desktop — negative for "wherever". False when the
// platform cannot, in which case the window keeps drawing its own; the
// reason is in the log, not the answer, because the only thing the switch
// can do with it is fall back.
func (a *App) OpenCompanionWindow(x, y int) bool {
	c := a.companion()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.body != nil {
		return true
	}
	body, err := openCompanionBody(x, y)
	if err != nil {
		debuglog.Msg("companion window: %v", err)
		return false
	}
	c.body = body
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
	a.companionOnce.Do(func() { a.companionSrv = &companionServer{token: mintCompanionToken()} })
	return a.companionSrv
}

func mintCompanionToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// A failed random read is a broken machine; a fixed token is still a
		// token, and the feed stays loopback-only regardless.
		return "aetox-companion"
	}
	return hex.EncodeToString(b)
}

// SetCompanionState is the window's report. Called on every change of pose,
// headline, look or switch — cheap, in memory, no listener needed for it to
// be recorded, so the feed has the truth the moment something starts to poll.
func (a *App) SetCompanionState(s CompanionState) {
	c := a.companion()
	c.mu.Lock()
	defer c.mu.Unlock()
	s.Seq = c.state.Seq + 1
	s.At = time.Now()
	c.state = s
}

// CompanionFeed brings the listener up if it is not, and says where it is.
func (a *App) CompanionFeed() CompanionFeedInfo {
	c := a.companion()
	if err := c.start(); err != nil {
		debuglog.Msg("companion feed: %v", err)
		return CompanionFeedInfo{}
	}
	return c.info()
}

// StopCompanionFeed closes the listener. The state is kept; a later
// CompanionFeed starts again on a fresh port with the same token.
func (a *App) StopCompanionFeed() CompanionFeedInfo {
	c := a.companion()
	c.stop()
	return c.info()
}

func (c *companionServer) info() CompanionFeedInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.srv == nil {
		return CompanionFeedInfo{}
	}
	return CompanionFeedInfo{Running: true, Port: c.port, URL: "http://127.0.0.1:" + strconv.Itoa(c.port) + companionPathPrefix + c.token + "/"}
}

func (c *companionServer) start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.srv != nil {
		return nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return errors.New("companion feed: listener has no TCP address")
	}
	c.port = addr.Port
	c.srv = &http.Server{Handler: c.handler(), ReadHeaderTimeout: 5 * time.Second}
	go func(srv *http.Server, ln net.Listener) {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			debuglog.Msg("companion feed stopped: %v", err)
		}
	}(c.srv, ln)
	debuglog.Msg("companion feed listening on 127.0.0.1:%d", c.port)
	return nil
}

func (c *companionServer) stop() {
	c.mu.Lock()
	srv := c.srv
	c.srv = nil
	c.port = 0
	c.mu.Unlock()
	if srv != nil {
		_ = srv.Close()
	}
}

// handler serves two things under /c/<token>/: the page at "" and the state
// at "state". Anything else, including a wrong token, is a 404 that says
// nothing about whether a token exists.
func (c *companionServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest, ok := strings.CutPrefix(r.URL.Path, companionPathPrefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		token, leaf, _ := strings.Cut(rest, "/")
		if subtle.ConstantTimeCompare([]byte(token), []byte(c.token)) != 1 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		switch leaf {
		case "":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(companionPage))
		case "state":
			c.mu.Lock()
			s := c.state
			c.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(s)
		default:
			http.NotFound(w, r)
		}
	})
}
