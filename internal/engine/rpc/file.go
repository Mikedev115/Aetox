package rpc

// The project's files across the wire (§248 phase 2, design doc §4).
//
// The panes load a file by URL — /aetox-file/<rel> on the window's own
// origin — because a URL streams and a binding does not (engine/filehost.go
// says why). When the engine is another process the bytes are on its side,
// so the engine serves the same host at /file/ on its listener, behind the
// same token as /rpc, and the screen's /aetox-file/ becomes a reverse proxy
// onto it: the webview still fetches the window's origin, so a pane that
// reads the document it loaded (SlidesPane) still can, and Range requests —
// the reason a video can be scrubbed — pass through untouched.

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

// FilePath is where the engine's listener serves the open project's files.
const FilePath = "/file/"

// ShelfPath is where it serves the studio's shelf (engine.ShelfHandler).
const ShelfPath = "/shelf/"

var errNoEngine = errors.New("no engine is running")

// ScreenFilePrefix is the window's own path for them, the one the panes use.
const ScreenFilePrefix = "/aetox-file/"

// ScreenShelfPrefix is the window's path for the shelf — the one the engine's
// own thumbnail URLs carry, so the two must agree (engine.studioHostPrefix).
const ScreenShelfPrefix = "/aetox-shelf/"

// screenPrefixes maps each window path onto the engine's.
var screenPrefixes = [][2]string{{ScreenFilePrefix, FilePath}, {ScreenShelfPrefix, ShelfPath}}

// enginePath is the engine's path for a window path, "" for neither space.
func enginePath(p string) string {
	for _, pair := range screenPrefixes {
		if strings.HasPrefix(p, pair[0]) {
			return pair[1] + strings.TrimPrefix(p, pair[0])
		}
	}
	return ""
}

// guarded is one of the listener's HTTP spaces, token-checked.
func (s *Server) guarded(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !Authorized(r, s.token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Endpoint is where the engine listens right now, and the token for it —
// asked per request, because a local engine that restarted on a TCP port
// is on another port, and a screen that cached the first would be proxying
// into nothing. ok is false while no engine is up.
type Endpoint func() (network, address, token string, ok bool)

// FileProxy is the screen's /aetox-file/ and /aetox-shelf/: every request
// under them goes to the engine's /file/ or /shelf/ with the token on, and
// the answer — status, headers, body, a 206 for a Range — comes back as it
// was. Requests for anything else fall through to next, the way the
// in-process middleware did.
func FileProxy(endpoint Endpoint, next http.Handler) http.Handler {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		network, address, _, ok := endpoint()
		if !ok {
			return nil, errNoEngine
		}
		var d net.Dialer
		d.Timeout = 10 * time.Second
		return d.DialContext(ctx, network, address)
	}
	// The engine may take a while to answer a big file's first byte off a
	// cold disk; the body after it streams at whatever pace the disk has.
	transport.ResponseHeaderTimeout = time.Minute
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "http"
			pr.Out.URL.Host = "engine"
			pr.Out.URL.Path = enginePath(pr.In.URL.Path)
			pr.Out.URL.RawPath = ""
			pr.Out.Host = "engine"
			if _, _, token, ok := endpoint(); ok {
				pr.Out.Header.Set("Authorization", "Bearer "+token)
			}
		},
		Transport: transport,
		// A streamed response is flushed as it arrives, not when it ends:
		// a video's first seconds are playable before the rest has come.
		FlushInterval: -1,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "the engine did not answer: "+err.Error(), http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if enginePath(r.URL.Path) == "" {
			next.ServeHTTP(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	})
}
