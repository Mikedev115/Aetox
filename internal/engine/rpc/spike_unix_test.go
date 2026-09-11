package rpc

// The spike §248 phase 2 opens with: a unix socket on Windows.
//
// The design says one address family on both systems — AF_UNIX, which Go
// supports on Windows 10 1803 and later — and everything after it assumes
// that is true on the owner's machine and on the build agents. This test is
// the proof: listen on a socket file under a short path, serve the WebSocket
// on it, dial it, round-trip one call. If it ever fails on some Windows, the
// TCP fallback in transport.go is what the local child uses there, and this
// test is what says so.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAUnixSocketCarriesTheWireOnThisOS(t *testing.T) {
	// A short path on purpose: sun_path is ~107 bytes, and t.TempDir() on
	// Windows is already sixty of them. DataRoot is where the real one goes.
	dir, err := os.MkdirTemp("", "aetox-sock")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "engine.sock")

	// A stale file from a crashed engine must not block the next one.
	if err := os.WriteFile(path, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := Listen("unix", path)
	if err != nil {
		t.Fatalf("AF_UNIX is not available here: %v — the local child must use --tcp on this machine", err)
	}
	t.Cleanup(func() { l.Close() })

	accepted := make(chan *Conn, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Accept(w, r, testToken, echo, nil)
		if err != nil {
			return
		}
		accepted <- c
	})}
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := Dial(ctx, "unix", path, testToken, nil, nil)
	if err != nil {
		t.Fatalf("dial over the unix socket: %v", err)
	}
	defer s.Close()
	var got []any
	if err := s.Call(ctx, "echo", []any{"over a unix socket"}, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "over a unix socket" {
		t.Errorf("echo = %v", got)
	}
	select {
	case e := <-accepted:
		e.Close()
	case <-time.After(time.Second):
	}

	// And the file is a socket, not a regular file the listener overwrote.
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSocket == 0 {
		// Windows reports a unix socket as a reparse point rather than
		// ModeSocket; the dial above having worked is the proof that
		// matters, so this is a log line, not a failure.
		t.Logf("socket file mode: %v", info.Mode())
	}
	var _ net.Addr = l.Addr()
	_ = json.Valid
}
