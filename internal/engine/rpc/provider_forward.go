package rpc

// The screen's half of the provider proxy: provider.open answered by
// sending the request with the credential the screen holds, and the
// response streamed back a piece at a time.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
)

// Signer is what the screen makes for a provider in a wire format: the
// transport that puts the credential on (desktop/provider_forward.go's
// providerTransport). The proxy never sees the credential; it sees a
// transport.
type Signer func(provider, wireFormat string) model.Transport

// ServeProvider registers the screen's answers to the engine's provider
// requests on the client. Call before Connect.
func ServeProvider(c *Client, sign Signer) {
	f := &providerForwarder{client: c, sign: sign, inflight: map[string]context.CancelFunc{}, networks: map[time.Duration]*http.Transport{}}
	c.Handle(MethodProviderOpen, f.open)
	c.OnNotification(MethodProviderCancel, f.cancel)
}

type providerForwarder struct {
	client *Client
	sign   Signer

	mu       sync.Mutex
	inflight map[string]context.CancelFunc
	// networks are the transports the requests go out on, one per first-byte
	// budget the engine asks for (there are two: local and remote). Kept so
	// connections are reused across a session's calls, as in one process.
	networks map[time.Duration]*http.Transport
}

func (f *providerForwarder) network(headerTimeout time.Duration) *http.Transport {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.networks[headerTimeout]; ok {
		return t
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.ResponseHeaderTimeout = headerTimeout
	f.networks[headerTimeout] = t
	return t
}

func (f *providerForwarder) open(ctx context.Context, _ string, params json.RawMessage) (any, error) {
	var p OpenParams
	if err := decodeParams(params, &p); err != nil {
		return nil, err
	}
	if p.ID == "" || p.URL == "" {
		return nil, &InvalidParams{Err: errors.New("provider.open needs an id and a URL")}
	}
	// The stream's own context, under the connection's: the engine can
	// withdraw the request (provider.cancel), and the wire going away ends
	// every stream on it.
	sctx, cancel := context.WithCancel(ctx)
	f.mu.Lock()
	f.inflight[p.ID] = cancel
	f.mu.Unlock()
	finish := func() {
		f.mu.Lock()
		delete(f.inflight, p.ID)
		f.mu.Unlock()
		cancel()
	}

	req, err := http.NewRequestWithContext(sctx, p.Method, p.URL, bytes.NewReader(p.Body))
	if err != nil {
		finish()
		return nil, fmt.Errorf("provider.open: %w", err)
	}
	for name, values := range p.Headers {
		req.Header[name] = append([]string(nil), values...)
	}
	budget := time.Duration(p.HeaderTimeoutMs) * time.Millisecond
	if budget <= 0 {
		budget = 5 * time.Minute
	}
	var transport http.RoundTripper = f.network(budget)
	if f.sign != nil {
		if wrap := f.sign(p.Provider, p.WireFormat); wrap != nil {
			transport = wrap(transport)
		}
	}
	resp, err := (&http.Client{Transport: transport}).Do(req) //nolint:bodyclose // the body is handed to stream, which defers its Close — the response outlives this call by design
	if err != nil {
		finish()
		return nil, err
	}
	conn := f.client.Conn()
	go f.stream(p.ID, resp.Body, conn, finish)
	return OpenResult{Status: resp.StatusCode, Headers: resp.Header}, nil
}

// stream copies the body to the engine, chunkSize at a time, and says how
// it ended. The chunk is copied before it is queued: the buffer is reused
// for the next read, and the frame is encoded later on the writer.
func (f *providerForwarder) stream(id string, body io.ReadCloser, conn *Conn, finish func()) {
	defer finish()
	defer body.Close()
	buf := make([]byte, chunkSize)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if nerr := conn.Notify(MethodProviderChunk, ChunkParams{ID: id, Data: append([]byte(nil), buf[:n]...)}); nerr != nil {
				return // the wire is gone; nobody to tell
			}
		}
		if err != nil {
			close := CloseParams{ID: id}
			if !errors.Is(err, io.EOF) {
				close.Error = err.Error()
			}
			_ = conn.Notify(MethodProviderClose, close)
			return
		}
	}
}

func (f *providerForwarder) cancel(_ string, params json.RawMessage) {
	var p CancelParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	f.mu.Lock()
	cancel := f.inflight[p.ID]
	f.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
