package rpc

// Where the wire is plugged in.
//
// The engine listens on a unix socket under DataRoot — on Windows too: Go
// speaks AF_UNIX there since Windows 10 1803, and one address family on both
// systems is one code path to prove (spike_unix_test.go is that proof). The
// fallback is TCP on a loopback port, which is also what an ssh tunnel lands
// on in phase 3. Neither address is a secret, so neither is the security:
// a token the screen made and handed the engine on its stdin is, checked on
// every HTTP request the listener takes — the WebSocket upgrade and, in
// phase 2's later commits, /file/.

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

// RPCPath is where the WebSocket is upgraded on the engine's listener.
const RPCPath = "/rpc"

// NewToken is 32 random bytes as hex — what the screen makes for each engine
// it starts, and never writes anywhere but the engine's stdin (or, over ssh,
// a file only the user can read).
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Authorized reports whether a request carries the token, compared in
// constant time. A wrong token and a missing one are the same answer.
func Authorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer"))
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

// Listen opens the engine's door. network is "unix" or "tcp"; a unix socket
// path that a previous engine left behind is removed first, because a
// crashed process leaves its file and Listen would otherwise refuse the
// address forever. sun_path is short (about 107 bytes) so the socket lives
// directly under DataRoot, never under a deep temp path.
func Listen(network, address string) (net.Listener, error) {
	switch network {
	case "unix":
		if err := os.Remove(address); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("rpc: clearing the old socket %s: %w", address, err)
		}
	case "tcp":
	default:
		return nil, fmt.Errorf("rpc: unknown network %q (unix or tcp)", network)
	}
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, fmt.Errorf("rpc: listen %s %s: %w", network, address, err)
	}
	return l, nil
}

// Upgrader accepts the screen's WebSocket. No origin check: the listener is
// a unix socket or loopback, and the token is what admits a caller.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// Accept is the engine's side of a connection: it checks the token, upgrades
// the request, and answers with the Conn. The caller serves it until Done.
func Accept(w http.ResponseWriter, r *http.Request, token string, h Handler, n Notifier) (*Conn, error) {
	if !Authorized(r, token) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, errors.New("rpc: unauthorized")
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err // Upgrade has already written the response
	}
	return newConn(ws, "e", h, n), nil
}

// Dial is the screen's side: it connects to the engine's listener and opens
// the WebSocket with the token. network and address name the listener the
// way Listen was given them; over ssh the address is the tunnel's local
// port, and the tunnel is somebody else's business.
func Dial(ctx context.Context, network, address, token string, h Handler, n Notifier) (*Conn, error) {
	dialer := websocket.Dialer{
		NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, address)
		},
		HandshakeTimeout: writeWait,
	}
	// The URL's host is a placeholder: NetDialContext ignores it, and gorilla
	// needs one to build the handshake.
	ws, resp, err := dialer.DialContext(ctx, "ws://engine"+RPCPath, http.Header{"Authorization": {"Bearer " + token}})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("rpc: the engine refused the token")
		}
		return nil, fmt.Errorf("rpc: dial %s %s: %w", network, address, err)
	}
	return newConn(ws, "s", h, n), nil
}
