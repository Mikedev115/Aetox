package mcp

// A session that ends is forgotten, and a header that is a reference is read
// every time.
//
// Both tests are 6 ก.ย. 20:32 replayed against a server this test owns: an
// HTTP MCP server behind a gate that wants one exact bearer token. The canva
// token the app resolved at 20:27 expired at 20:32; the SDK failed the
// connection on the 401 and the client kept handing out the dead session for
// the next half hour, "connected" on the settings page and absent from every
// registry.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// gate is the token the server wants right now, changeable mid-test.
type gate struct {
	mu   sync.Mutex
	want string
	// seen counts requests that got through, so a test can tell "the token
	// changed and the server accepted it" from "the client never asked".
	seen int
}

func (g *gate) set(token string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.want = token
}

func (g *gate) passed() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.seen
}

// gatedServer is one echo tool over streamable HTTP, behind a bearer check.
func gatedServer(t *testing.T) (*httptest.Server, *gate) {
	t.Helper()
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "gated", Version: "1"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "echo", Description: "echoes text"},
		func(_ context.Context, _ *mcpsdk.CallToolRequest, args struct {
			Text string `json:"text"`
		}) (*mcpsdk.CallToolResult, any, error) {
			return &mcpsdk.CallToolResult{Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: args.Text}}}, nil, nil
		})
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return server }, nil)
	g := &gate{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		ok := r.Header.Get("Authorization") == "Bearer "+g.want
		if ok {
			g.seen++
		}
		g.mu.Unlock()
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts, g
}

// A token that rotates under a live connection reaches the wire on the very
// next request — no reconnect, no restart, nothing the user has to press.
func TestARotatedHeaderReachesTheNextRequest(t *testing.T) {
	ts, g := gatedServer(t)
	g.set("first")

	var mu sync.Mutex
	token := "first"
	c := New(Server{
		Name: "gated", URL: ts.URL, Timeout: 10 * time.Second,
		Headers: map[string]string{"Authorization": "Bearer stale-snapshot"},
		HeaderSource: func() map[string]string {
			mu.Lock()
			defer mu.Unlock()
			return map[string]string{"Authorization": "Bearer " + token}
		},
	})
	t.Cleanup(func() { c.Close() })

	ctx := context.Background()
	if _, err := c.Tools(ctx); err != nil {
		t.Fatalf("first Tools: %v", err)
	}
	before := g.passed()

	// The source and the server move together, the way oauth.Token and the
	// provider do when a refresh lands. The static Headers map still says
	// "stale-snapshot" and must not be what goes out.
	mu.Lock()
	token = "second"
	mu.Unlock()
	g.set("second")

	res, err := c.CallTool(ctx, "echo", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatalf("CallTool after rotation: %v", err)
	}
	if got := res.Content[0].(*mcpsdk.TextContent).Text; got != "hi" {
		t.Fatalf("echo = %q, want hi", got)
	}
	if g.passed() <= before {
		t.Fatal("the rotated token never reached the server")
	}
	if c.Status() != StatusConnected {
		t.Fatalf("status = %q after a rotation that needed no reconnect", c.Status())
	}
}

// A connection the server has failed is dropped, and the next caller — once
// the backoff has passed — gets a fresh one instead of the corpse.
func TestADeadSessionIsDroppedAndReconnected(t *testing.T) {
	old := reconnectBackoff
	reconnectBackoff = 50 * time.Millisecond
	t.Cleanup(func() { reconnectBackoff = old })

	ts, g := gatedServer(t)
	g.set("live")

	var mu sync.Mutex
	token := "live"
	c := New(Server{
		Name: "gated", URL: ts.URL, Timeout: 10 * time.Second,
		HeaderSource: func() map[string]string {
			mu.Lock()
			defer mu.Unlock()
			return map[string]string{"Authorization": "Bearer " + token}
		},
	})
	t.Cleanup(func() { c.Close() })
	ctx := context.Background()

	if _, err := c.Tools(ctx); err != nil {
		t.Fatalf("first Tools: %v", err)
	}

	// The server stops accepting what the client is sending — the token
	// expired on the server's clock and the store has not caught up.
	g.set("rotated")
	_, err := c.Tools(ctx)
	if err == nil {
		t.Fatal("Tools succeeded against a server that refuses the token")
	}
	if !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("the refusal must name itself, got: %v", err)
	}

	// The SDK fails the connection on a 401, the watcher sees it end, and
	// the client forgets it. This is the state the old code never left.
	deadline := time.Now().Add(5 * time.Second)
	for c.Status() == StatusConnected {
		if time.Now().After(deadline) {
			t.Fatalf("status still %q five seconds after the connection died", c.Status())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if c.Status() != StatusIdle {
		t.Fatalf("status = %q after a drop, want idle (failed is for a connect that never worked)", c.Status())
	}
	if c.Err() == nil || !strings.Contains(c.Err().Error(), "dropped") {
		t.Fatalf("Err() = %v, want the drop recorded for the settings page", c.Err())
	}

	// Inside the backoff the same error comes back at once, with no handshake
	// spent on a server that would refuse it anyway.
	before := g.passed()
	if _, err := c.Tools(ctx); err == nil {
		t.Fatal("Tools reconnected inside the backoff")
	}
	if g.passed() != before {
		t.Fatal("a call inside the backoff reached the server")
	}

	// The store catches up. After the backoff, the next caller connects again
	// — the way the first call of the day did — and the tools are back.
	mu.Lock()
	token = "rotated"
	mu.Unlock()
	time.Sleep(2 * reconnectBackoff)
	tools, err := c.Tools(ctx)
	if err != nil {
		t.Fatalf("Tools after the token caught up: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("tools = %+v, want the one echo", tools)
	}
	if c.Status() != StatusConnected {
		t.Fatalf("status = %q after reconnecting", c.Status())
	}
	if c.Err() != nil {
		t.Fatalf("Err() = %v after a clean reconnect, want nil", c.Err())
	}
}

// A caller that reaches a session in the moment between its death and the
// watcher noticing is served over a new connection, not handed the drop.
func TestACallOnAClosedSessionRetriesOnce(t *testing.T) {
	old := reconnectBackoff
	reconnectBackoff = time.Hour // the retry must not be waiting on this
	t.Cleanup(func() { reconnectBackoff = old })

	ts, g := gatedServer(t)
	g.set("live")
	c := New(Server{
		Name: "gated", URL: ts.URL, Timeout: 10 * time.Second,
		Headers: map[string]string{"Authorization": "Bearer live"},
	})
	t.Cleanup(func() { c.Close() })
	ctx := context.Background()

	session, err := c.ensure(ctx, false)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// The connection ends underneath the client — closed from the SDK's side,
	// as it would be after a failed request — without going through Close.
	if err := session.Close(); err != nil {
		t.Fatalf("closing the session out from under the client: %v", err)
	}
	// Race the watcher on purpose: whether it has run or not, the call below
	// must come back with tools.
	tools, err := c.Tools(ctx)
	if err != nil {
		t.Fatalf("Tools over a session that had closed: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(tools))
	}
}

// The stdio half of the same rule: a local server whose process dies is
// respawned by the next call rather than answered from its ghost.
func TestALocalServerThatDiesIsRespawned(t *testing.T) {
	old := reconnectBackoff
	reconnectBackoff = 50 * time.Millisecond
	t.Cleanup(func() { reconnectBackoff = old })

	bin := buildEchoServer(t)
	c := New(Server{Name: "echo", Command: []string{bin}})
	t.Cleanup(func() { c.Close() })
	ctx := context.Background()

	if _, err := c.Tools(ctx); err != nil {
		t.Fatalf("first Tools: %v", err)
	}
	c.mu.Lock()
	pid := c.procPID
	c.mu.Unlock()
	if pid == 0 {
		t.Fatal("no pid recorded for a connected local server")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("finding the server process: %v", err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatalf("killing the server process: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for c.Status() == StatusConnected {
		if time.Now().After(deadline) {
			t.Fatal("the client never noticed its server die")
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(2 * reconnectBackoff)

	res, err := c.CallTool(ctx, "echo", map[string]any{"text": "again"})
	if err != nil {
		t.Fatalf("CallTool after the server died: %v", err)
	}
	if got := res.Content[0].(*mcpsdk.TextContent).Text; got != "again" {
		t.Fatalf("echo = %q, want again", got)
	}
	c.mu.Lock()
	respawned := c.procPID
	c.mu.Unlock()
	if respawned == 0 || respawned == pid {
		t.Fatalf("pid after respawn = %d, want a new process (was %d)", respawned, pid)
	}
}
