// Package mcp connects Aetox to external Model Context Protocol servers.
//
// Two transports: local/stdio (a subprocess speaking MCP over stdin/stdout,
// e.g. npx/uvx-based servers) and remote streamable HTTP (URL + optional
// static headers).
//
// This package knows nothing about where a header's value comes from, and that
// is a division of labour rather than a limitation: OAuth arrived on 2026-09-03
// (internal/oauth/mcpauth.go) and lands here as an ordinary
// `Authorization: Bearer ${connect:name}` header whose value resolveSecretRefs
// fetches — refreshing the token when it is close to expiry. Since 6 ก.ย. that
// fetch happens on every request (Server.HeaderSource) rather than once when
// the connection is built, because a token fetched once is a token that
// expires mid-session. A server that signs in and one that carries a pasted
// key still reach this file looking identical, which is the point.
//
// The transport, JSON-RPC framing, and initialize handshake come from the
// official github.com/modelcontextprotocol/go-sdk; this package owns only
// config, connection lifecycle, and (elsewhere) the skill.Tool adapter.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mikedev115/Aetox/internal/proc"
)

// Status reports where a server's connection stands. Connection failures are
// surfaced as StatusFailed rather than thrown, so a misconfigured server just
// drops out of the tool list instead of breaking the agent loop.
type Status string

const (
	StatusIdle      Status = "idle"      // configured, not connected yet (lazy)
	StatusConnected Status = "connected" // handshake succeeded, tools usable
	StatusFailed    Status = "failed"    // connect failed; see Client.Err
)

const defaultTimeout = 30 * time.Second

// Server is one configured MCP server. A non-empty URL selects the remote
// streamable-HTTP transport; otherwise Command is spawned as a local stdio
// subprocess. Fields mirror the config schema in MCP-SUPPORT-PLAN.md §4.
type Server struct {
	Name        string            // stable id; used as the tool-name prefix
	Command     []string          // local: argv0 + args
	Cwd         string            // local: working dir; caller resolves against sandbox root
	Environment map[string]string // local: merged over os.Environ()
	URL         string            // remote: streamable HTTP endpoint
	Headers     map[string]string // remote: static headers (e.g. Authorization)
	Timeout     time.Duration     // connect timeout; default 30s
	Disabled    bool              // configured but switched off; Manager skips it
	// Deferred means no desk carries this server — only named agents do — so
	// it is left out of the startup connect and brought up when one of them is
	// actually started.
	//
	// `for:` decided who *sees* a server's tools and never decided whether to
	// connect it, so a server placed on one agent was still spawned on every
	// launch by every user who had it configured, whether or not that agent was
	// ever spoken to. For a 90-tool remote server that is a handshake and a full
	// tool listing bought on the chance somebody might.
	//
	// Only agent-only servers qualify. A desk's server has to be there before
	// the first message, because a desk is what the user is already sitting at.
	Deferred bool
	// Tools, when non-empty, is the only tools accepted from this server —
	// everything else it offers is dropped before it ever reaches a registry.
	//
	// This exists for the server that is two products in one box. n8n-mcp is
	// the case that forced it: 7 of its tools answer a question Aetox cannot
	// (n8n publishes no node schemas), and 16 write workflows — which Aetox
	// already does, through its own tools, its own permission rules and its own
	// tool_runs log. Taking the whole server would put a second way to write in
	// front of the model, and the one it happened to pick would decide whether
	// the user's own record of what was done to their instance had the entry.
	//
	// Why an allowlist and not `deny:` on the profile, which already works: a
	// denial list is written against somebody else's catalogue, and the day
	// that server ships a seventeenth write tool it arrives switched on and
	// nothing says so. An allowlist is wrong loudly instead — see SkillTools,
	// which refuses to let a list that matches nothing pass as a server with
	// nothing to offer.
	//
	// Empty means take everything, so nothing changes for a server nobody has
	// written a list for, which is all of them today.
	Tools []string
	// HeaderSource, when set, is asked for the remote headers on EVERY request,
	// and Headers is then only what the settings page shows.
	//
	// It exists because of 6 ก.ย. 20:32. Headers was resolved once, when the
	// Manager was built at 20:27, and the canva OAuth token it resolved to had
	// five minutes left — outside the two-minute grace oauth.Token refreshes
	// within, so nothing refreshed it, and the header carried the same string
	// to the wire until the server said Unauthorized. A Manager is built once
	// and survives every re-bootstrap by design, so "resolved at construction"
	// meant "resolved at app launch" for a token whose whole point is that it
	// expires. Asked per request, the refresh happens inside the grace window
	// the way it does for every model provider, and a rotated key in .env
	// reaches the next call without a restart.
	HeaderSource func() map[string]string
	// EnvSource is the same for a local server's environment, asked at each
	// spawn — which is more than once now that a dead session reconnects.
	EnvSource func() map[string]string
}

// reconnectBackoff is how long a client sits on a dropped connection before
// the next caller is allowed to build a new one.
//
// A drop that reconnects instantly is a loop when the cause is still there:
// the desktop re-bootstraps its engine on every session switch, desk change
// and stance change — twelve times in twenty-five seconds on 6 ก.ย. — and each
// one registers every server. With a token the server keeps refusing, that
// would be an initialize handshake and a tools/list per switch, all refused.
// A caller inside the window gets the drop's own error back, immediately,
// which is the same answer at none of the cost. A var, not a const, so a test
// can shorten it.
var reconnectBackoff = 30 * time.Second

// Client wraps a single MCP server connection. Connect is lazy: the subprocess
// is not started until the first Tools/CallTool call, so servers that are
// configured but never used don't slow startup. Safe for concurrent use.
type Client struct {
	cfg Server

	mu      sync.Mutex
	session *mcpsdk.ClientSession
	status  Status
	lastErr error
	// procCancel and procPID belong to a local (stdio) server's subprocess and
	// are zero for an HTTP one. They are the client's own rather than the
	// caller's: ensure() holds whichever per-call context asked for a tool, and
	// binding the server to that would kill it when the first call returned.
	procCancel context.CancelFunc
	procPID    int
	toolCount  int // tools seen on the last successful Tools(); 0 until then
	// droppedAt is when the last live session was found dead (see drop), and
	// zero when the client has never lost one or has been Closed since.
	droppedAt time.Time
}

// New builds a Client for cfg without connecting.
func New(cfg Server) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	return &Client{cfg: cfg, status: StatusIdle}
}

// Name returns the server's configured id.
func (c *Client) Name() string { return c.cfg.Name }

// Deferred reports whether this server waits for the agent that needs it. See
// Server.Deferred.
func (c *Client) Deferred() bool { return c.cfg.Deferred }

// Command returns the server's configured argv (for status display).
func (c *Client) Command() []string { return c.cfg.Command }

// Status reports the current connection state.
func (c *Client) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// Err returns the last connection error, if the client is in StatusFailed.
func (c *Client) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastErr
}

// ensure connects on first use and caches the session. A prior failure is
// sticky — we don't respawn on every call, which would let a broken server
// stall each tool invocation. Close resets that so a reconfigured server can
// reconnect.
//
// A session that connected and later DIED is not a prior failure. It used to
// be treated as a live one: the cached pointer stayed set, every call went to
// a connection the SDK had already closed, and the client answered "client is
// closing" in 0ms for the rest of the app's life — status still "connected",
// tools gone from every registry built after the death (6 ก.ย., canva: 22
// minutes of it, on every re-bootstrap). drop() clears the pointer when the
// session ends, so the next caller here builds a new one; force skips the
// backoff for a caller that just watched its own call die (see withSession).
func (c *Client) ensure(ctx context.Context, force bool) (*mcpsdk.ClientSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil {
		return c.session, nil
	}
	if c.status == StatusFailed {
		return nil, c.lastErr
	}
	if !force && !c.droppedAt.IsZero() && time.Since(c.droppedAt) < reconnectBackoff {
		return nil, c.lastErr
	}
	// Both stay nil for an HTTP server, which has no process to tear down.
	var procCancel context.CancelFunc
	var localCmd *exec.Cmd

	var transport mcpsdk.Transport
	switch {
	case c.cfg.URL != "":
		transport = &mcpsdk.StreamableClientTransport{
			Endpoint:   c.cfg.URL,
			HTTPClient: headerHTTPClient(c.cfg.Headers, c.cfg.HeaderSource),
		}
	case len(c.cfg.Command) == 0 || c.cfg.Command[0] == "":
		c.status = StatusFailed
		c.lastErr = errors.New("mcp: server " + c.cfg.Name + " has no command or url")
		return nil, c.lastErr
	default:
		// The server lives as long as this Client does, so it gets a context of
		// its own — and a cancellable one, because that is the only kind
		// os/exec watches. proc.KillOnCancel installs cmd.Cancel, and with a
		// context that can never be done the command starts fine and the kill
		// silently never happens.
		//
		// It also has to be exec.CommandContext and not exec.Command: Start
		// refuses a command that carries a Cancel it was not built with, and the
		// Start here is inside the SDK's Connect, where the refusal would
		// surface as every local server failing to connect.
		procCtx, cancelProc := context.WithCancel(context.Background())
		cmd := exec.CommandContext(procCtx, c.cfg.Command[0], c.cfg.Command[1:]...)
		cmd.Dir = c.cfg.Cwd
		// The production desktop exe is a GUI app: without this, a console child
		// (npx→cmd.exe on Windows) pops a visible Windows Terminal window on spawn.
		proc.HideConsole(cmd)
		// CommandTransport.Close closes stdin, then SIGTERMs, then kills — all of
		// it aimed at the direct child. A server that forks (npx→node,
		// uvx→python) left its grandchildren running, and every mid-life
		// teardown hits this: Settings' Test button, adding or removing a server,
		// a project switch. Close() below does the polite shutdown first and
		// sweeps the tree after.
		proc.KillOnCancel(cmd)
		procCancel, localCmd = cancelProc, cmd
		extra := c.cfg.Environment
		if c.cfg.EnvSource != nil {
			extra = c.cfg.EnvSource()
		}
		if len(extra) > 0 {
			env := os.Environ()
			for k, v := range extra {
				env = append(env, k+"="+v)
			}
			cmd.Env = env
		}
		transport = &mcpsdk.CommandTransport{Command: cmd}
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "aetox", Version: "0"}, nil)

	// Bound the initialize handshake so a process that starts but never speaks
	// MCP can't hang the caller (and, via lazy connect, startup) indefinitely.
	connectCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		// A server that half-started still forked whatever it forked, and
		// nothing else holds a handle on it from here.
		if procCancel != nil {
			procCancel()
		}
		c.status = StatusFailed
		c.lastErr = err
		return nil, err
	}
	// Connect is where the SDK calls Start, so this is the first moment there is
	// a pid to remember. Close needs it after Wait has already reaped the
	// process, by which point cmd.Process is no longer a safe thing to read.
	if localCmd != nil && localCmd.Process != nil {
		c.procPID = localCmd.Process.Pid
	}
	c.procCancel = procCancel
	c.session = session
	c.status = StatusConnected
	c.lastErr = nil // a drop that was recovered from is no longer news
	c.droppedAt = time.Time{}
	// Somebody has to be listening for the end. Wait returns when the
	// connection closes for any reason — the server's process exiting, the
	// SDK failing the connection on a non-transient HTTP error (a 401 is one),
	// a network drop — and until this goroutine existed, nobody was.
	go func() { c.drop(session, session.Wait()) }()
	return session, nil
}

// drop forgets a session that has ended, so that the next call connects again
// instead of talking to a corpse. Only the session named is dropped: a newer
// one, built after this one died, is left alone.
//
// Status goes back to idle, not failed, on purpose. Failed is sticky and means
// "connecting does not work"; a session that worked and then ended says
// nothing of the kind. The drop's error is kept in lastErr so the settings page
// can show why it went, and reconnectBackoff keeps the retry from being a loop.
func (c *Client) drop(session *mcpsdk.ClientSession, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != session {
		return
	}
	failed := err != nil
	if err == nil {
		err = mcpsdk.ErrConnectionClosed
	}
	c.session = nil
	c.status = StatusIdle
	c.lastErr = fmt.Errorf("connection dropped: %w", err)
	c.toolCount = 0
	// The backoff is for a session that ended BADLY — the server refusing the
	// token, the process crashing — where the cause is likely still there.
	// One that ended cleanly (the server retired an idle session, say) can be
	// rebuilt by the very next caller.
	if failed {
		c.droppedAt = time.Now()
	} else {
		c.droppedAt = time.Time{}
	}
	// A local server whose connection ended has usually exited already; the
	// sweep is for the tree it may have left, and is harmless when it has not.
	c.releaseProcess()
}

// withSession runs op on the live session, connecting first if there is none,
// and retries ONCE — over a fresh connection — when op reports that the
// connection had already closed under it.
//
// The retry is for the caller that arrives between the death and the watcher
// in ensure noticing it; without it that caller pays for a drop it did not
// cause. It is not for a call the server refused: an error that is not
// ErrConnectionClosed comes straight back, and a second failure of the same
// kind is not retried, so a server that closes the connection on every call
// cannot make this spin.
func (c *Client) withSession(ctx context.Context, op func(*mcpsdk.ClientSession) error) error {
	session, err := c.ensure(ctx, false)
	if err != nil {
		return err
	}
	err = op(session)
	if err == nil || !errors.Is(err, mcpsdk.ErrConnectionClosed) {
		return err
	}
	c.drop(session, err)
	if session, err = c.ensure(ctx, true); err != nil {
		return err
	}
	return op(session)
}

// Tools lists the server's tools, connecting lazily. On connect failure it
// returns the error; callers treat that as "this server contributes no tools".
func (c *Client) Tools(ctx context.Context) ([]*mcpsdk.Tool, error) {
	var tools []*mcpsdk.Tool
	err := c.withSession(ctx, func(session *mcpsdk.ClientSession) error {
		tools = tools[:0]
		for tool, iterErr := range session.Tools(ctx, nil) {
			if iterErr != nil {
				return iterErr
			}
			tools = append(tools, tool)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.toolCount = len(tools)
	c.mu.Unlock()
	return tools, nil
}

// ToolCount reports how many tools the server exposed on the last successful
// Tools() enumeration (0 before the first one, or after Close).
func (c *Client) ToolCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.toolCount
}

// CallTool invokes one tool on the server, connecting lazily.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*mcpsdk.CallToolResult, error) {
	var out *mcpsdk.CallToolResult
	err := c.withSession(ctx, func(session *mcpsdk.ClientSession) (err error) {
		out, err = session.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Resources lists what the server offers to read — files, records, documents
// it is the authority on. Empty is the normal case: a server that only exposes
// tools declares no resources, and asking one that does not support them at all
// is an error, not a fact worth reporting.
func (c *Client) Resources(ctx context.Context) ([]*mcpsdk.Resource, error) {
	var out []*mcpsdk.Resource
	err := c.withSession(ctx, func(session *mcpsdk.ClientSession) error {
		out = out[:0]
		for resource, iterErr := range session.Resources(ctx, nil) {
			if iterErr != nil {
				return iterErr
			}
			out = append(out, resource)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReadResource fetches one resource's contents by URI.
func (c *Client) ReadResource(ctx context.Context, uri string) (*mcpsdk.ReadResourceResult, error) {
	var out *mcpsdk.ReadResourceResult
	err := c.withSession(ctx, func(session *mcpsdk.ClientSession) (err error) {
		out, err = session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: uri})
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Prompts lists the server's prompt templates — the workflows its author
// thought were worth naming.
func (c *Client) Prompts(ctx context.Context) ([]*mcpsdk.Prompt, error) {
	var out []*mcpsdk.Prompt
	err := c.withSession(ctx, func(session *mcpsdk.ClientSession) error {
		out = out[:0]
		for prompt, iterErr := range session.Prompts(ctx, nil) {
			if iterErr != nil {
				return iterErr
			}
			out = append(out, prompt)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetPrompt renders one prompt template with arguments filled in.
func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcpsdk.GetPromptResult, error) {
	var out *mcpsdk.GetPromptResult
	err := c.withSession(ctx, func(session *mcpsdk.ClientSession) (err error) {
		out, err = session.GetPrompt(ctx, &mcpsdk.GetPromptParams{Name: name, Arguments: args})
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Close terminates the subprocess if connected and resets to idle so a later
// call can reconnect. Safe to call when never connected.
//
// The order is the whole point. session.Close runs the shutdown the protocol
// prescribes — close stdin, wait, SIGTERM, wait, kill — and every step of it
// reaches only the process the SDK started. Then the sweep, because a server
// that forked (npx→node, uvx→python) has left descendants that no signal in
// that ladder was ever addressed to.
//
// KillTree rather than procCancel for the sweep: the SDK's Close calls Wait,
// and once Wait returns, os/exec has retired the goroutine that would have
// invoked cmd.Cancel — cancelling here would compile, run, and do nothing.
// procCancel is still called, for the release of the context itself.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		c.status = StatusIdle
		c.releaseProcess()
		return nil
	}
	err := c.session.Close()
	c.releaseProcess()
	c.session = nil
	c.status = StatusIdle
	c.lastErr = nil
	c.toolCount = 0
	// A deliberate close is not a drop: the next call may connect at once.
	// (The watcher this Close wakes finds c.session already nil and does
	// nothing — drop only acts on the session it was given.)
	c.droppedAt = time.Time{}
	return err
}

// releaseProcess sweeps a local server's tree and drops what named it. Caller
// holds c.mu.
func (c *Client) releaseProcess() {
	if c.procPID != 0 {
		proc.KillTree(c.procPID)
		c.procPID = 0
	}
	if c.procCancel != nil {
		c.procCancel()
		c.procCancel = nil
	}
}

// headerHTTPClient returns an http.Client that stamps headers onto every
// request (Authorization tokens etc.), or the default client when there are
// none. With a source, the headers are asked for per request — see
// Server.HeaderSource — and the static map is only the fallback for a source
// that answers nothing.
func headerHTTPClient(headers map[string]string, source func() map[string]string) *http.Client {
	if len(headers) == 0 && source == nil {
		return nil // transport falls back to http.DefaultClient
	}
	return &http.Client{Transport: headerRoundTripper{headers: headers, source: source}}
}

type headerRoundTripper struct {
	headers map[string]string
	source  func() map[string]string
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	headers := h.headers
	if h.source != nil {
		if live := h.source(); live != nil {
			headers = live
		}
	}
	// Per http.RoundTripper's contract, don't mutate the caller's request.
	req = req.Clone(req.Context())
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(req)
}
