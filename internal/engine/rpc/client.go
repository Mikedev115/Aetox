package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// connectWait is how long a binding waits for the wire to be up before it
// fails with ErrDisconnected: the engine is starting, or restarting after a
// crash, and the frontend's first calls arrive before either is done. Long
// enough for a cold start and one restart's backoff; short enough that a
// binding never looks hung to the person who pressed the button.
var connectWait = 30 * time.Second

// Client is the screen's end: engine.API over the wire (client_gen.go), the
// engine's events as they arrive, and the handlers for what the engine asks
// of the screen — signing a request, running a window tool, rendering a deck
// — registered by the screen before it connects.
type Client struct {
	opts ClientOptions

	mu        sync.RWMutex
	conn      *Conn
	connected chan struct{} // closed while conn is live, replaced when it goes
	handlers  map[string]Handler
	notifiers map[string]Notifier
}

// ClientOptions is what the screen wants told.
type ClientOptions struct {
	// OnEvent receives every event the engine emits, in order, with the
	// payload as the engine encoded it — what the screen hands to
	// EventsEmit as-is.
	OnEvent func(name string, data json.RawMessage)
	// OnFailure is told when a binding with no error in its signature could
	// not reach the engine: the caller got a zero value, and this is the one
	// place that knows why (engine_status, phase 2's status chip).
	OnFailure func(method string, err error)
}

// NewClient prepares a client; Connect opens the wire.
func NewClient(opts ClientOptions) *Client {
	return &Client{opts: opts, connected: make(chan struct{}), handlers: map[string]Handler{}, notifiers: map[string]Notifier{}}
}

// Handle registers the screen's answer to one engine→screen method.
func (c *Client) Handle(method string, h Handler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[method] = h
}

// OnNotification registers the screen's receiver for one engine→screen
// notification other than "event" (the provider stream's cancel, say).
func (c *Client) OnNotification(method string, n Notifier) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notifiers[method] = n
}

// Connect dials the engine. Calling it again after the connection went —
// the engine restarted, the tunnel came back — puts the new wire under the
// same client: the handlers stay registered, and a binding that was
// waiting for a wire gets this one.
func (c *Client) Connect(ctx context.Context, network, address, token string) error {
	conn, err := Dial(ctx, network, address, token, c.dispatch, c.notify)
	if err != nil {
		return err
	}
	c.mu.Lock()
	old := c.conn
	c.conn = conn
	if old == nil {
		// Nobody was connected, so waiters are on this channel; a live old
		// connection means it is already closed and they never waited.
		close(c.connected)
	}
	c.mu.Unlock()
	if old != nil {
		old.Close()
	}
	go func() {
		<-conn.Done()
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
			c.connected = make(chan struct{})
		}
		c.mu.Unlock()
	}()
	return nil
}

// await is the live connection, waiting up to connectWait for one.
func (c *Client) await(ctx context.Context) (*Conn, error) {
	deadline := time.NewTimer(connectWait)
	defer deadline.Stop()
	for {
		c.mu.RLock()
		conn, connected := c.conn, c.connected
		c.mu.RUnlock()
		if conn != nil {
			return conn, nil
		}
		select {
		case <-connected:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, ErrDisconnected
		}
	}
}

// Conn is the connection underneath, for the pieces of the screen that
// speak to the engine outside the bindings (the provider forwarder).
func (c *Client) Conn() *Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// Done closes when the connection is over; Err says why. Before Connect,
// Done is a channel that never closes.
func (c *Client) Done() <-chan struct{} {
	if conn := c.Conn(); conn != nil {
		return conn.Done()
	}
	return make(chan struct{})
}

func (c *Client) Err() error {
	if conn := c.Conn(); conn != nil {
		return conn.Err()
	}
	return nil
}

func (c *Client) Close() error {
	if conn := c.Conn(); conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *Client) dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	c.mu.RLock()
	h := c.handlers[method]
	c.mu.RUnlock()
	if h == nil {
		return nil, ErrMethodNotFound
	}
	return h(ctx, method, params)
}

// eventParams is the "event" notification's payload.
type eventParams struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) notify(method string, params json.RawMessage) {
	if method == "event" {
		if c.opts.OnEvent == nil {
			return
		}
		var ev eventParams
		if err := json.Unmarshal(params, &ev); err != nil {
			debuglog.Msg("rpc: an event that could not be read: %v", err)
			return
		}
		c.opts.OnEvent(ev.Name, ev.Data)
		return
	}
	c.mu.RLock()
	n := c.notifiers[method]
	c.mu.RUnlock()
	if n != nil {
		n(method, params)
	}
}

// call is one binding across the wire. The bindings carry no context of
// their own — the frontend's call has none — so the wait is the connection's
// lifetime: a call outlives nothing but the wire it is on. With no wire yet
// it waits for one (await), so the frontend's first calls at launch and its
// calls through a restart are answered late rather than refused.
func (c *Client) call(method string, params []any, out any) error {
	conn, err := c.await(context.Background())
	if err != nil {
		return err
	}
	return conn.Call(conn.ctx, method, params, out)
}

// callN is call for a method that answers with several values, which the
// server sends as one array.
func (c *Client) callN(method string, params []any, outs ...any) error {
	var raw []json.RawMessage
	if err := c.call(method, params, &raw); err != nil {
		return err
	}
	if len(raw) != len(outs) {
		return fmt.Errorf("rpc: %s answered with %d values, want %d", method, len(raw), len(outs))
	}
	for i, r := range raw {
		if string(r) == "null" {
			continue
		}
		if err := json.Unmarshal(r, outs[i]); err != nil {
			return fmt.Errorf("rpc: %s value %d is unreadable: %w", method, i, err)
		}
	}
	return nil
}

// failed is where a binding with no error to return puts the wire's.
func (c *Client) failed(method string, err error) {
	debuglog.Msg("rpc: %s: %v", method, err)
	if c.opts.OnFailure != nil {
		c.opts.OnFailure(method, err)
	}
}
