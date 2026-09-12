package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/engine"
)

// Server is the engine's end: it owns the engine, takes one screen at a
// time on the listener, answers the screen's binding calls through the
// generated dispatch table, and is the engine's Screen (ScreenPeer) — so
// events and the engine's own questions reach whichever screen is connected.
type Server struct {
	token    string
	peer     *ScreenPeer
	engine   *engine.Engine
	dataRoot string
	files    http.Handler
	shelf    http.Handler

	mu        sync.RWMutex
	handlers  map[string]Handler
	notifiers map[string]Notifier
}

// NewServer builds the engine behind the wire. build is engine.NewEngine
// (or a test's wrapper around it): the engine is built with the peer as its
// screen, which is the only way the two can meet, since each needs the
// other first.
func NewServer(token string, build func(engine.Screen) *engine.Engine) *Server {
	s := &Server{
		token:     token,
		peer:      newScreenPeer(),
		handlers:  map[string]Handler{},
		notifiers: map[string]Notifier{},
	}
	s.engine = build(s.peer)
	s.peer.server = s
	s.dataRoot, _ = config.DataRoot()
	s.files = engine.FileHandler(s.engine, FilePath)
	s.shelf = engine.ShelfHandler(s.engine, ShelfPath)
	s.Handle(MethodHello, s.hello)
	// The provider stream's two notifications land on the peer's streams.
	s.OnNotification(MethodProviderChunk, func(_ string, params json.RawMessage) { s.peer.streams.chunk(params) })
	s.OnNotification(MethodProviderClose, func(_ string, params json.RawMessage) { s.peer.streams.closed(params) })
	return s
}

// Engine is the engine this server fronts — for the process that runs it
// (startup, shutdown), never for a caller across the wire.
func (s *Server) Engine() *engine.Engine { return s.engine }

// Screen is the engine's view of whatever screen is connected.
func (s *Server) Screen() *ScreenPeer { return s.peer }

// Handle registers a method of the server's own, tried before the bindings.
func (s *Server) Handle(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// OnNotification registers a receiver for a screen→engine notification.
func (s *Server) OnNotification(method string, n Notifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifiers[method] = n
}

// Handler is the HTTP side of the listener: the WebSocket upgrade at
// RPCPath, and the open project's files at FilePath, behind the same token.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(RPCPath, s.accept)
	mux.Handle(FilePath, s.guarded(s.files))
	mux.Handle(ShelfPath, s.guarded(s.shelf))
	return mux
}

// Serve answers on the listener until ctx ends or the listener fails.
func (s *Server) Serve(ctx context.Context, l net.Listener) error {
	srv := &http.Server{Handler: s.Handler()}
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(l) }()
	select {
	case err := <-errs:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.peer.detachAll()
		return srv.Shutdown(shutdown)
	}
}

func (s *Server) accept(w http.ResponseWriter, r *http.Request) {
	conn, err := Accept(w, r, s.token, s.dispatch, s.notify)
	if err != nil {
		return
	}
	s.peer.attach(conn)
	go func() {
		<-conn.Done()
		s.peer.detach(conn)
	}()
}

func (s *Server) dispatch(ctx context.Context, method string, params json.RawMessage) (any, error) {
	s.mu.RLock()
	h := s.handlers[method]
	s.mu.RUnlock()
	if h != nil {
		return h(ctx, method, params)
	}
	if result, err, ok := dispatch(s.engine, method, params); ok {
		return result, err
	}
	return nil, ErrMethodNotFound
}

func (s *Server) notify(method string, params json.RawMessage) {
	s.mu.RLock()
	n := s.notifiers[method]
	s.mu.RUnlock()
	if n != nil {
		n(method, params)
	}
}

// decodeParams reads positional params into the method's arguments. Fewer
// values than arguments leaves the rest at their zero value — a frontend
// that omits a trailing argument today gets the same — and more are
// ignored; a value that does not fit its argument is an invalid-params
// error naming which.
func decodeParams(raw json.RawMessage, into ...any) error {
	var items []json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &items); err != nil {
			return &InvalidParams{Err: fmt.Errorf("params must be an array: %w", err)}
		}
	}
	for i, ptr := range into {
		if i >= len(items) || string(items[i]) == "null" {
			continue
		}
		if err := json.Unmarshal(items[i], ptr); err != nil {
			return &InvalidParams{Err: fmt.Errorf("argument %d: %w", i, err)}
		}
	}
	return nil
}
