package rpc

// The wire, before anything of the engine is on it: two ends over a real
// socket, calls both ways, answers out of order, notifications in order, a
// peer that dies. Every later commit of phase 2 stands on these.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const testToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// pair is a screen and an engine end over a loopback TCP listener, with the
// handlers the test gives each side. The engine end arrives on a channel
// because the listener hands it to us from its own goroutine.
func pair(t *testing.T, engine Handler, engineNotify Notifier, screen Handler, screenNotify Notifier) (screenEnd, engineEnd *Conn) {
	t.Helper()
	accepted := make(chan *Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != RPCPath {
			http.NotFound(w, r)
			return
		}
		c, err := Accept(w, r, testToken, engine, engineNotify)
		if err != nil {
			return
		}
		accepted <- c
	}))
	t.Cleanup(srv.Close)
	addr := strings.TrimPrefix(srv.URL, "http://")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := Dial(ctx, "tcp", addr, testToken, screen, screenNotify)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	select {
	case e := <-accepted:
		t.Cleanup(func() { s.Close(); e.Close() })
		return s, e
	case <-time.After(5 * time.Second):
		t.Fatal("the engine end never arrived")
		return nil, nil
	}
}

// echo answers with its params, after an optional delay named in them, so a
// test can make the first call finish last.
func echo(_ context.Context, method string, params json.RawMessage) (any, error) {
	var args []any
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &InvalidParams{Err: err}
	}
	switch method {
	case "echo":
		return args, nil
	case "sleep":
		ms, _ := args[0].(float64)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return args, nil
	case "fail":
		return nil, errors.New("file-gone")
	case "fail_kind":
		return nil, fmt.Errorf("wrapped: %w", context.Canceled)
	case "panic":
		panic("boom")
	case "nothing":
		return nil, nil
	}
	return nil, ErrMethodNotFound
}

func TestACallIsAnsweredWithItsResult(t *testing.T) {
	s, _ := pair(t, echo, nil, nil, nil)
	var got []any
	if err := s.Call(context.Background(), "echo", []any{"สวัสดี", 7}, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "สวัสดี" || got[1] != float64(7) {
		t.Errorf("echo = %v", got)
	}
}

// Two calls in flight, the first one slower: the answers come back to the
// right callers whatever order they arrive in. This is the property that
// lets SendMessage hold its id for a whole turn while CancelTurn goes by.
func TestConcurrentCallsAreMatchedByID(t *testing.T) {
	s, _ := pair(t, echo, nil, nil, nil)
	var wg sync.WaitGroup
	results := make([]string, 2)
	errs := make([]error, 2)
	for i, ms := range []int{300, 10} {
		wg.Add(1)
		go func(i, ms int) {
			defer wg.Done()
			var got []any
			errs[i] = s.Call(context.Background(), "sleep", []any{ms, fmt.Sprintf("call-%d", i)}, &got)
			if len(got) == 2 {
				results[i], _ = got[1].(string)
			}
		}(i, ms)
	}
	wg.Wait()
	for i := range results {
		if errs[i] != nil {
			t.Fatalf("call %d: %v", i, errs[i])
		}
		if results[i] != fmt.Sprintf("call-%d", i) {
			t.Errorf("call %d got %q — the answers crossed", i, results[i])
		}
	}
}

// Notifications keep the order they were sent in. An event stream that
// reordered agent:chunk would render a reply with its words shuffled.
func TestNotificationsArriveInOrder(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	done := make(chan struct{})
	const n = 500
	s, e := pair(t, nil, nil, nil, func(method string, params json.RawMessage) {
		mu.Lock()
		seen = append(seen, string(params))
		if len(seen) == n {
			close(done)
		}
		mu.Unlock()
	})
	_ = s
	for i := 0; i < n; i++ {
		if err := e.Notify("event", map[string]any{"i": i}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		mu.Lock()
		t.Fatalf("got %d of %d notifications", len(seen), n)
	}
	for i, p := range seen {
		if want := fmt.Sprintf(`{"i":%d}`, i); p != want {
			t.Fatalf("notification %d was %s — the order broke", i, p)
		}
	}
}

// The wire is two-way: the engine calls the screen while the screen's own
// call to the engine is still open. This is provider.open inside SendMessage.
func TestTheEngineCanCallBackWhileACallIsOpen(t *testing.T) {
	var engineEnd *Conn
	ready := make(chan struct{})
	engine := func(ctx context.Context, method string, params json.RawMessage) (any, error) {
		<-ready
		var signed string
		if err := engineEnd.Call(ctx, "screen.sign", []any{"request"}, &signed); err != nil {
			return nil, err
		}
		return "turn done with " + signed, nil
	}
	screen := func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method != "screen.sign" {
			return nil, ErrMethodNotFound
		}
		return "a signed request", nil
	}
	s, e := pair(t, engine, nil, screen, nil)
	engineEnd = e
	close(ready)
	var got string
	if err := s.Call(context.Background(), "SendMessage", []any{"hello"}, &got); err != nil {
		t.Fatal(err)
	}
	if got != "turn done with a signed request" {
		t.Errorf("got %q", got)
	}
}

// An error crosses as its own sentence — the frontend matches "file-gone" in
// err.message — and a registered kind keeps errors.Is working on this side.
func TestErrorsKeepTheirSentenceAndTheirKind(t *testing.T) {
	s, _ := pair(t, echo, nil, nil, nil)
	err := s.Call(context.Background(), "fail", nil, nil)
	if err == nil || err.Error() != "file-gone" {
		t.Fatalf("err = %v, want the engine's own sentence", err)
	}
	var wire *Error
	if !errors.As(err, &wire) || wire.Code != codeApp {
		t.Errorf("err = %#v, want a wire error with the app code", err)
	}

	err = s.Call(context.Background(), "fail_kind", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a canceled error lost its identity across the wire: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "wrapped:") {
		t.Errorf("the kind kept, the sentence lost: %v", err)
	}

	err = s.Call(context.Background(), "no_such", nil, nil)
	if !errors.Is(err, ErrMethodNotFound) {
		t.Errorf("an unknown method = %v, want ErrMethodNotFound", err)
	}

	err = s.Call(context.Background(), "echo", "not an array", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid params") {
		t.Errorf("unreadable params = %v, want an invalid-params error", err)
	}
}

// A handler that panics answers with an error and the connection lives on.
// The screen's turn must never take the engine down.
func TestAPanickingHandlerAnswersAnErrorAndStaysUp(t *testing.T) {
	s, _ := pair(t, echo, nil, nil, nil)
	err := s.Call(context.Background(), "panic", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "internal error") {
		t.Fatalf("panic came back as %v", err)
	}
	var got []any
	if err := s.Call(context.Background(), "echo", []any{"still here"}, &got); err != nil {
		t.Fatalf("the connection did not survive the panic: %v", err)
	}
}

// A call whose result is nothing still gets a response, or the caller would
// wait forever for a frame that never comes.
func TestAVoidCallStillAnswers(t *testing.T) {
	s, _ := pair(t, echo, nil, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Call(ctx, "nothing", nil, nil); err != nil {
		t.Fatal(err)
	}
}

// The caller's context ends its wait; the engine keeps working on the request
// and its late answer is dropped, not delivered to somebody else.
func TestACancelledCallStopsWaiting(t *testing.T) {
	s, _ := pair(t, echo, nil, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := s.Call(ctx, "sleep", []any{2000}, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	// And the connection is fine for the next one.
	var got []any
	if err := s.Call(context.Background(), "echo", []any{"next"}, &got); err != nil {
		t.Fatal(err)
	}
}

// When the peer goes, every open call ends with ErrDisconnected at once —
// not after a timeout, and not with a made-up answer.
func TestAClosedPeerFailsEveryOpenCall(t *testing.T) {
	s, e := pair(t, echo, nil, nil, nil)
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() { errs <- s.Call(context.Background(), "sleep", []any{5000}, nil) }()
	}
	time.Sleep(100 * time.Millisecond)
	e.Close()
	for i := 0; i < 3; i++ {
		select {
		case err := <-errs:
			if !errors.Is(err, ErrDisconnected) {
				t.Errorf("call ended with %v, want ErrDisconnected", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("a call outlived its connection")
		}
	}
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the screen end never noticed the engine closing")
	}
}

// The token is the whole admission: without it the upgrade is refused, and
// the refusal is told apart from a listener that is not there.
func TestTheTokenAdmitsAndRefuses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := Accept(w, r, testToken, echo, nil); err == nil {
			go func() { <-c.Done() }()
		}
	}))
	t.Cleanup(srv.Close)
	addr := strings.TrimPrefix(srv.URL, "http://")
	ctx := context.Background()
	if _, err := Dial(ctx, "tcp", addr, "wrong", nil, nil); err == nil || !strings.Contains(err.Error(), "refused the token") {
		t.Errorf("a wrong token dialed in: %v", err)
	}
	if _, err := Dial(ctx, "tcp", addr, "", nil, nil); err == nil {
		t.Error("no token dialed in")
	}
	c, err := Dial(ctx, "tcp", addr, testToken, nil, nil)
	if err != nil {
		t.Fatalf("the right token was refused: %v", err)
	}
	c.Close()
	if !Authorized(&http.Request{Header: http.Header{"Authorization": {"Bearer " + testToken}}}, testToken) {
		t.Error("Authorized refused the right header")
	}
	if Authorized(&http.Request{Header: http.Header{}}, "") {
		t.Error("an engine with no token admitted an empty one")
	}
}

// Listen clears a stale socket file and refuses a network it does not know.
func TestListenOpensUnixAndTCP(t *testing.T) {
	l, err := Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l.Close()
	if _, err := Listen("pipe", "x"); err == nil {
		t.Error("an unknown network was accepted")
	}
	var _ net.Listener = l
}
