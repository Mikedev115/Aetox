package rpc

// The provider request, carried to the screen to be signed (§248 rule 3,
// design doc §4).
//
// The engine holds no key, so it cannot send a model call itself. What it
// does instead is hand the whole HTTP request — method, URL, headers, body —
// to the screen as `provider.open`, and the screen does what its own
// signedTransport does in one process: put the credential on, send it, and
// stream the response back as `provider.chunk` notifications closed by
// `provider.close`. The engine sees status, headers and a body it reads like
// any other; it never sees what was put on the request.
//
// The proxy is a model.Transport, plugged in exactly where the network was:
// under retryTransport and idleTimeoutBody (internal/model/httpclient.go), so
// a 429 is retried by re-opening, a silent stream is caught by the same
// watchdog, and the quota headers reach NoteQuotas off the response the
// proxy builds. Nothing above the transport knows the request left the
// process.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
)

// The wire's names for the four messages of one request.
const (
	MethodProviderOpen   = "provider.open"   // engine → screen, a call
	MethodProviderChunk  = "provider.chunk"  // screen → engine, a notification
	MethodProviderClose  = "provider.close"  // screen → engine, a notification
	MethodProviderCancel = "provider.cancel" // engine → screen, a notification
)

// chunkQueue is how many pieces of a response may wait between the read
// loop and the engine's reader. Each is at most chunkSize, so this bounds
// the memory a slow consumer costs; a full queue holds the read loop, which
// is back-pressure on the screen and the honest answer to a stream nobody
// is reading.
const (
	chunkQueue = 256
	chunkSize  = 32 << 10
)

// OpenParams is provider.open's request: the HTTP request without its
// credential, and the wire format so the screen knows how a key is carried.
type OpenParams struct {
	ID         string              `json:"id"`
	Provider   string              `json:"provider"`
	WireFormat string              `json:"wireFormat"`
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	// HeaderTimeoutMs is how long the screen may wait for the response
	// headers — the engine's own first-byte budget, so the two sides agree.
	HeaderTimeoutMs int64 `json:"headerTimeoutMs"`
}

// OpenResult is the response's head; the body follows as chunks.
type OpenResult struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
}

// ChunkParams is one piece of the body.
type ChunkParams struct {
	ID   string `json:"id"`
	Data []byte `json:"data"`
}

// CloseParams ends the body: Error is "" for a clean end, the screen's read
// error otherwise.
type CloseParams struct {
	ID    string `json:"id"`
	Error string `json:"error,omitempty"`
}

// CancelParams withdraws a request the engine no longer wants.
type CancelParams struct {
	ID string `json:"id"`
}

// providerStreams is the engine's side of every response in flight, by id.
type providerStreams struct {
	mu      sync.Mutex
	next    uint64
	streams map[string]*providerStream
}

// providerStream is one response body on its way from the screen: chunks
// land in the queue from the read loop, and a goroutine of the stream's own
// feeds them to the pipe the engine reads. The goroutine is what keeps the
// read loop off the pipe — a pipe write blocks until the engine reads, and
// the engine may be busy with another stream.
type providerStream struct {
	id    string
	queue chan ChunkParams
	done  chan CloseParams
	pw    *io.PipeWriter
	pr    *io.PipeReader
	// finished closes when pump has returned: the body ended, or the
	// engine stopped reading it.
	finished chan struct{}
}

func (s *providerStreams) open() *providerStream {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	id := "p" + strconv.FormatUint(s.next, 10)
	pr, pw := io.Pipe()
	st := &providerStream{
		id:       id,
		queue:    make(chan ChunkParams, chunkQueue),
		done:     make(chan CloseParams, 1),
		pr:       pr,
		pw:       pw,
		finished: make(chan struct{}),
	}
	if s.streams == nil {
		s.streams = map[string]*providerStream{}
	}
	s.streams[id] = st
	return st
}

func (s *providerStreams) get(id string) *providerStream {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streams[id]
}

func (s *providerStreams) forget(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.streams, id)
}

// chunk and closed are what the server's notifiers call, on the read loop.
func (s *providerStreams) chunk(params json.RawMessage) {
	var p ChunkParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	if st := s.get(p.ID); st != nil {
		st.queue <- p
	}
}

func (s *providerStreams) closed(params json.RawMessage) {
	var p CloseParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	if st := s.get(p.ID); st != nil {
		s.forget(p.ID)
		st.done <- p
	}
}

// pump feeds the pipe until the screen closes the stream or the engine
// stops reading. Chunks already queued are written before a close is
// honoured, so the order the screen sent is the order the engine reads.
func (st *providerStream) pump() {
	defer close(st.finished)
	for {
		select {
		case c := <-st.queue:
			if _, err := st.pw.Write(c.Data); err != nil {
				// The engine closed the body: drain nothing more.
				return
			}
		case c := <-st.done:
			// Whatever the read loop queued before the close is still in
			// the queue: drain it first.
			for {
				select {
				case c2 := <-st.queue:
					if _, err := st.pw.Write(c2.Data); err != nil {
						return
					}
					continue
				default:
				}
				break
			}
			if c.Error != "" {
				st.pw.CloseWithError(fmt.Errorf("provider stream: %s", c.Error))
			} else {
				st.pw.Close()
			}
			return
		}
	}
}

// providerProxy is the transport: one RoundTrip is one provider.open.
type providerProxy struct {
	peer       *ScreenPeer
	provider   string
	wireFormat string
	// budget is how long to wait for the response head — the engine's own
	// first-byte budget, read off the network transport this proxy replaced.
	budget time.Duration
}

// ProviderTransport is the credential the engine does not hold: a transport
// that carries the request to the screen. The network transport it is
// handed is not dialed — the screen dials — but its first-byte budget is
// kept, so a provider that says nothing is given up on after the same wait
// it would have been in one process.
func (p *ScreenPeer) ProviderTransport(provider, wireFormat string) model.Transport {
	return func(network http.RoundTripper) http.RoundTripper {
		budget := 5 * time.Minute
		if t, ok := network.(*http.Transport); ok && t.ResponseHeaderTimeout > 0 {
			budget = t.ResponseHeaderTimeout
		}
		return &providerProxy{peer: p, provider: provider, wireFormat: wireFormat, budget: budget}
	}
}

func (x *providerProxy) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("provider proxy: reading the request: %w", err)
		}
		body = b
	}
	streams := &x.peer.streams
	st := streams.open()
	go st.pump()

	params := OpenParams{
		ID:              st.id,
		Provider:        x.provider,
		WireFormat:      x.wireFormat,
		Method:          req.Method,
		URL:             req.URL.String(),
		Headers:         req.Header.Clone(),
		Body:            body,
		HeaderTimeoutMs: x.budget.Milliseconds(),
	}
	// The head has the same budget the network would have had; the body has
	// the engine's idle watchdog above this transport, as before.
	ctx, cancel := context.WithTimeout(req.Context(), x.budget)
	var head OpenResult
	err := x.peer.call(ctx, MethodProviderOpen, []any{params}, &head)
	cancel()
	if err != nil {
		streams.forget(st.id)
		st.pw.CloseWithError(err)
		if ctx.Err() != nil && req.Context().Err() == nil {
			return nil, fmt.Errorf("provider proxy: no response head within %s: %w", x.budget, err)
		}
		return nil, err
	}
	// The engine giving up on the request — the Stop button, a timeout, a
	// body closed before its end — is told to the screen, which stops
	// reading and lets the provider's socket go. abandon is safe to reach
	// twice: the second time the stream is already forgotten.
	abandon := func(why error) {
		if streams.get(st.id) == nil {
			return
		}
		streams.forget(st.id)
		if conn := x.peer.current(); conn != nil {
			_ = conn.Notify(MethodProviderCancel, CancelParams{ID: st.id})
		}
		st.pr.CloseWithError(why)
	}
	go func() {
		select {
		case <-req.Context().Done():
			abandon(req.Context().Err())
		case <-st.finished:
		}
	}()
	resp := &http.Response{
		Status:     fmt.Sprintf("%d %s", head.Status, http.StatusText(head.Status)),
		StatusCode: head.Status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header(head.Headers),
		Body:       &streamBody{pr: st.pr, onClose: func() { abandon(io.ErrClosedPipe) }},
		Request:    req,
	}
	if resp.Header == nil {
		resp.Header = http.Header{}
	}
	return resp, nil
}

// streamBody is the response body the engine reads: the pipe, and on Close
// the stream abandoned — forgotten here, withdrawn on the screen — so a body
// the model client stopped reading is not read to its end for nobody.
type streamBody struct {
	pr      *io.PipeReader
	onClose func()
	once    sync.Once
}

func (b *streamBody) Read(p []byte) (int, error) { return b.pr.Read(p) }
func (b *streamBody) Close() error {
	b.once.Do(func() {
		b.onClose()
		_ = b.pr.Close()
	})
	return nil
}
