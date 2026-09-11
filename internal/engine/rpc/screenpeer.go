package rpc

import (
	"context"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// The methods the engine asks of the screen, by name on the wire. The
// screen registers a handler for each (desktop/screen_rpc.go); a test
// registers what it wants to answer.
const (
	MethodEvent            = "event"
	MethodAgentTab         = "screen.agentTab"
	MethodDefaultModel     = "screen.defaultModel"
	MethodProbe            = "screen.probe"
	MethodModelResident    = "screen.modelResident"
	MethodProviderEndpoint = "screen.providerEndpoint"
	MethodRenderDeck       = "screen.renderDeck"
)

// screenWait is how long the engine waits for a screen to be connected
// before a question that needs one fails with ErrNoScreen. The screen is
// the process that started this one, so its absence is a restart or a
// tunnel that dropped — long enough to ride out a reconnect, short enough
// that a turn does not sit on a dead question for good.
var screenWait = 60 * time.Second

// ScreenPeer is engine.Screen over the wire: whichever screen is connected
// right now, or nobody. Nobody is a real state (design doc §4): events go
// nowhere, a request that needs a screen waits screenWait for one and then
// fails as ErrNoScreen, and a screen that arrives later fills every blank.
type ScreenPeer struct {
	server *Server

	mu      sync.Mutex
	conn    *Conn
	arrived chan struct{} // closed when a screen is attached

	// streams is every provider response on its way from the screen
	// (provider_proxy.go).
	streams providerStreams
	// announced is what the screen said in hello (screen_tool.go).
	announced announced
}

func newScreenPeer() *ScreenPeer {
	return &ScreenPeer{arrived: make(chan struct{})}
}

// attach makes conn the screen. A screen already there is closed first:
// the new one is the reconnect, and two screens on one engine is two
// windows drawing one chat.
func (p *ScreenPeer) attach(conn *Conn) {
	p.mu.Lock()
	old := p.conn
	p.conn = conn
	close(p.arrived)
	p.mu.Unlock()
	if old != nil {
		old.Close()
	}
}

func (p *ScreenPeer) detach(conn *Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != conn {
		return // already replaced
	}
	p.conn = nil
	p.arrived = make(chan struct{})
}

func (p *ScreenPeer) detachAll() {
	p.mu.Lock()
	conn := p.conn
	p.conn = nil
	p.arrived = make(chan struct{})
	p.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
}

// current is the screen now, nil for nobody.
func (p *ScreenPeer) current() *Conn {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.conn
}

// Connected reports whether a screen is on the wire.
func (p *ScreenPeer) Connected() bool { return p.current() != nil }

// await is the screen, waiting up to screenWait for one to connect.
func (p *ScreenPeer) await(ctx context.Context) (*Conn, error) {
	deadline := time.NewTimer(screenWait)
	defer deadline.Stop()
	for {
		p.mu.Lock()
		conn, arrived := p.conn, p.arrived
		p.mu.Unlock()
		if conn != nil {
			return conn, nil
		}
		select {
		case <-arrived:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, ErrNoScreen
		}
	}
}

// call asks the screen, waiting for one if none is connected.
func (p *ScreenPeer) call(ctx context.Context, method string, params any, out any) error {
	conn, err := p.await(ctx)
	if err != nil {
		return err
	}
	return conn.Call(ctx, method, params, out)
}

// Emit sends an event to the screen. No screen, no event — the engine's
// state is what a reconnecting screen re-reads, not a replay of these.
func (p *ScreenPeer) Emit(event string, data any) {
	if conn := p.current(); conn != nil {
		_ = conn.Notify(MethodEvent, map[string]any{"name": event, "data": data})
	}
}

func (p *ScreenPeer) ProviderEndpoint(provider string) string {
	var out string
	if err := p.call(context.Background(), MethodProviderEndpoint, []any{provider}, &out); err != nil {
		return ""
	}
	return out
}

func (p *ScreenPeer) AgentTab() string {
	conn := p.current()
	if conn == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var out string
	if err := conn.Call(ctx, MethodAgentTab, nil, &out); err != nil {
		return ""
	}
	return out
}

func (p *ScreenPeer) DefaultModel(provider, baseURL string) string {
	var out string
	if err := p.call(context.Background(), MethodDefaultModel, []any{provider, baseURL}, &out); err != nil {
		return ""
	}
	return out
}

func (p *ScreenPeer) Probe(provider, modelName, baseURL, wireFormat string) (string, error) {
	var out string
	err := p.call(context.Background(), MethodProbe, []any{provider, modelName, baseURL, wireFormat}, &out)
	return out, err
}

func (p *ScreenPeer) ModelResident(provider, baseURL, modelName string) bool {
	var out bool
	if err := p.call(context.Background(), MethodModelResident, []any{provider, baseURL, modelName}, &out); err != nil {
		return false
	}
	return out
}

func (p *ScreenPeer) RenderDeck(ctx context.Context, fileURL string, req engine.DeckRender) (engine.DeckRendered, error) {
	var out engine.DeckRendered
	err := p.call(ctx, MethodRenderDeck, []any{fileURL, req}, &out)
	return out, err
}

var _ engine.Screen = (*ScreenPeer)(nil)
