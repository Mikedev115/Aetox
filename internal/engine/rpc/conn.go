package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// Timings. A ping every pingEvery; a peer that has said nothing — no frame,
// no pong — for pongWait is gone. writeWait bounds a single frame's write so
// a peer that stopped reading cannot hold the writer forever; the frame
// after it finds the connection closed.
const (
	pingEvery = 15 * time.Second
	pongWait  = 30 * time.Second
	writeWait = 10 * time.Second
	// outQueue is how many frames may wait for the writer. Events are the
	// bulk of it (agent:chunk runs at ~100/s), and a queue this deep is a
	// second or two of a busy turn; a slower peer than that is back-pressure
	// on the engine, which is the honest answer.
	outQueue = 512
)

// Handler answers a request from the other side. A method it does not
// have returns ErrMethodNotFound; params it cannot read return an
// *InvalidParams; anything else is the method's own error, carried across
// as its sentence and kind.
type Handler func(ctx context.Context, method string, params json.RawMessage) (any, error)

// Notifier receives a notification. It runs on the read loop, in order,
// so it must not block on the other side answering — hand slow work off.
type Notifier func(method string, params json.RawMessage)

// InvalidParams is a request whose params did not decode.
type InvalidParams struct{ Err error }

func (e *InvalidParams) Error() string { return "invalid params: " + e.Err.Error() }
func (e *InvalidParams) Unwrap() error { return e.Err }

// Conn is one end of the wire: a WebSocket with a single writer, a single
// reader, and a request dispatched on its own goroutine — so a call that
// takes a whole turn (SendMessage) holds nothing up, and CancelTurn,
// AnswerUserQuestion and every event pass it by.
type Conn struct {
	ws     *websocket.Conn
	side   string
	handle Handler
	notify Notifier

	out     chan []byte
	pending map[string]chan *message
	mu      sync.Mutex
	nextID  uint64

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
	err    error
	wg     sync.WaitGroup
}

// newConn wraps an open WebSocket and starts its loops. side is this end's
// id prefix: "s" for the screen, "e" for the engine.
func newConn(ws *websocket.Conn, side string, h Handler, n Notifier) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Conn{
		ws:      ws,
		side:    side,
		handle:  h,
		notify:  n,
		out:     make(chan []byte, outQueue),
		pending: map[string]chan *message{},
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	if c.handle == nil {
		c.handle = func(context.Context, string, json.RawMessage) (any, error) { return nil, ErrMethodNotFound }
	}
	if c.notify == nil {
		c.notify = func(string, json.RawMessage) {}
	}
	ws.SetReadLimit(64 << 20) // a deck's pictures, base64: generous, not unbounded
	_ = ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { return ws.SetReadDeadline(time.Now().Add(pongWait)) })
	c.wg.Add(3)
	go c.readLoop()
	go c.writeLoop()
	go c.pingLoop()
	return c
}

// Call sends a request and waits for its answer, decoded into out (which may
// be nil for a call whose result is not wanted). ctx ends the wait — the
// request is then forgotten here, not withdrawn there — and the connection
// closing ends it with ErrDisconnected.
func (c *Conn) Call(ctx context.Context, method string, params any, out any) error {
	raw, err := encodeParams(params)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.nextID++
	id := c.side + strconv.FormatUint(c.nextID, 10)
	reply := make(chan *message, 1)
	c.pending[id] = reply
	c.mu.Unlock()
	// Registered before sending, never after: a fast peer can answer before
	// the sending goroutine gets back to the map.
	if err := c.enqueue(message{JSONRPC: "2.0", ID: &id, Method: method, Params: raw}); err != nil {
		c.forget(id)
		return err
	}
	select {
	case m, ok := <-reply:
		if !ok {
			return ErrDisconnected
		}
		if m.Error != nil {
			return m.Error
		}
		if out == nil || len(m.Result) == 0 || string(m.Result) == "null" {
			return nil
		}
		if err := json.Unmarshal(m.Result, out); err != nil {
			return fmt.Errorf("rpc: %s answered with something unreadable: %w", method, err)
		}
		return nil
	case <-ctx.Done():
		c.forget(id)
		return ctx.Err()
	case <-c.done:
		c.forget(id)
		return ErrDisconnected
	}
}

// Notify sends a message that expects no answer. Frames go out in the order
// Notify was called from one goroutine, which is what an event stream needs.
func (c *Conn) Notify(method string, params any) error {
	raw, err := encodeParams(params)
	if err != nil {
		return err
	}
	return c.enqueue(message{JSONRPC: "2.0", Method: method, Params: raw})
}

// Done closes when the connection is over, for whatever reason; Err says
// which. A clean Close answers nil.
func (c *Conn) Done() <-chan struct{} { return c.done }

func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Close ends the connection: a close frame to the peer when the socket still
// takes one, then everything waiting gets ErrDisconnected.
func (c *Conn) Close() error {
	c.closeWith(nil)
	return nil
}

func (c *Conn) closeWith(err error) {
	c.once.Do(func() {
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
		c.cancel()
		// Best effort, and on purpose not through the writer: the writer may
		// be the one that failed.
		_ = c.ws.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		_ = c.ws.Close()
		c.mu.Lock()
		for id, ch := range c.pending {
			delete(c.pending, id)
			close(ch)
		}
		c.mu.Unlock()
		close(c.done)
	})
}

func (c *Conn) forget(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *Conn) enqueue(m message) error {
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("rpc: encoding message: %w", err)
	}
	select {
	case <-c.done:
		return ErrDisconnected
	default:
	}
	select {
	case c.out <- body:
		return nil
	case <-c.done:
		return ErrDisconnected
	}
}

func (c *Conn) writeLoop() {
	defer c.wg.Done()
	for {
		select {
		case body := <-c.out:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.TextMessage, body); err != nil {
				c.closeWith(fmt.Errorf("rpc: write: %w", err))
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Conn) pingLoop() {
	defer c.wg.Done()
	t := time.NewTicker(pingEvery)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				c.closeWith(fmt.Errorf("rpc: ping: %w", err))
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Conn) readLoop() {
	defer c.wg.Done()
	for {
		_, body, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.closeWith(nil)
			} else {
				c.closeWith(fmt.Errorf("rpc: read: %w", err))
			}
			return
		}
		// Any frame is proof of life, not only a pong.
		_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
		var m message
		if err := json.Unmarshal(body, &m); err != nil {
			// A frame that is not JSON-RPC at all: answer the way the spec
			// says, keep the connection. Nothing of ours sends one.
			_ = c.enqueue(message{JSONRPC: "2.0", Error: &Error{Code: codeParse, Message: "parse error: " + err.Error()}})
			continue
		}
		switch {
		case m.isRequest():
			go c.serve(&m)
		case m.isNotification():
			c.notify(m.Method, m.Params)
		case m.isResponse():
			c.mu.Lock()
			ch, ok := c.pending[*m.ID]
			delete(c.pending, *m.ID)
			c.mu.Unlock()
			if ok {
				ch <- &m
			}
			// An answer nobody waits for — the caller gave up — is dropped.
		default:
			id := m.ID
			_ = c.enqueue(message{JSONRPC: "2.0", ID: id, Error: &Error{Code: codeInvalidRequest, Message: "invalid request"}})
		}
	}
}

// serve answers one request on its own goroutine. A handler that panics
// answers with an internal error rather than taking the process down: the
// screen's turn must never end the engine, and the other way round.
func (c *Conn) serve(m *message) {
	var (
		result any
		err    error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				debuglog.Msg("rpc: %s panicked: %v\n%s", m.Method, r, debug.Stack())
				err = &Error{Code: codeInternal, Message: fmt.Sprintf("%s: internal error: %v", m.Method, r)}
			}
		}()
		result, err = c.handle(c.ctx, m.Method, m.Params)
	}()
	reply := message{JSONRPC: "2.0", ID: m.ID}
	switch {
	case err == nil:
		raw, encErr := json.Marshal(result)
		if encErr != nil {
			reply.Error = &Error{Code: codeInternal, Message: fmt.Sprintf("%s: encoding result: %v", m.Method, encErr)}
		} else {
			reply.Result = raw
			if len(raw) == 0 {
				reply.Result = nullResult
			}
		}
	case errors.Is(err, ErrMethodNotFound):
		reply.Error = &Error{Code: codeMethodNotFound, Message: "no such method: " + m.Method, Data: &errorData{Kind: "method_not_found"}}
	default:
		var bad *InvalidParams
		if errors.As(err, &bad) {
			reply.Error = &Error{Code: codeInvalidParams, Message: m.Method + ": " + bad.Error()}
		} else {
			reply.Error = toWire(err)
		}
	}
	_ = c.enqueue(reply)
}
