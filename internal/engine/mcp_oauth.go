package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/Mikedev115/Aetox/internal/oauth"
)

// MCP server sign-in bindings — deliberately separate from the screen's
// StartSignIn/CompleteSignIn (desktop/oauth.go) rather than sharing them.
//
// Those exist for "sign in with your own subscription": CompleteSignIn hands
// back a ModelInfo and re-bootstraps the model engine, because that is what
// an AI-provider sign-in is for. An MCP server sign-in is not that — nothing
// about which model Aetox talks to changes when semgrep is connected, and
// what it needs afterward is a rebuildMCP() so the newly-stored credential
// actually gets resolved into a live server. Sharing one pair of bindings
// between the two would mean CompleteSignIn either re-bootstraps the engine
// for no reason on an MCP sign-in, or skips it for a real one — the same
// argument connect.go's own package comment makes for keeping this apart
// from oauth.Methods entirely: "a model sign-in buys thinking while a
// connection buys reach."
//
// And this one is the engine's where the provider sign-in is the screen's:
// the credential it stores is read by `${connect:}` on the host the MCP
// server runs on (§248 rule 5), so the store has to be that host's. The
// browser step needs a browser on that host, which a remote engine has none
// of — v1 does not support it there (design doc §9).
//
// The flow itself is generic (internal/oauth/mcpauth.go discovers everything
// from the server's own URL), so unlike StartSignIn there is no fixed
// registry of providers to check against here — any server name reaches the
// same discovery, and a server whose authorization server has no dynamic
// client registration fails with a specific, readable error rather than
// "unknown provider".

// SignInPrompt is what the UI shows while waiting. Which fields are filled
// depends on Kind: "device" fills UserCode and VerificationURI (type this code
// into that page), "browser" and "paste" fill URL. Shared with the screen's
// provider sign-in (desktop/oauth.go), which answers with the same shape.
type SignInPrompt struct {
	Provider        string `json:"provider"`
	Kind            string `json:"kind"`
	URL             string `json:"url"`
	UserCode        string `json:"user_code,omitempty"`
	VerificationURI string `json:"verification_uri,omitempty"`
}

type pendingSignIn struct {
	pending *oauth.Pending
	// ctx spans both calls: CompleteMCPSignIn blocks on it, so the cancel is
	// what actually unblocks a user who changed their mind.
	ctx    context.Context
	cancel context.CancelFunc
}

var pendingMCPSignIns = struct {
	sync.Mutex
	byServer map[string]*pendingSignIn
}{byServer: map[string]*pendingSignIn{}}

// StartMCPSignIn discovers the server's OAuth metadata, registers Aetox as a
// client, and returns what to show the user while the browser sign-in is in
// flight. Nothing is stored until CompleteMCPSignIn succeeds.
func (a *Engine) StartMCPSignIn(serverName, resourceURL string) (SignInPrompt, error) {
	// A second sign-in for the same server abandons the first, same reasoning
	// as CancelSignIn in oauth.go: clicking twice means giving up on the
	// first attempt, not wanting two listeners racing for one redirect.
	a.CancelMCPSignIn(serverName)

	ctx, cancel := context.WithCancel(context.Background())
	pending, err := oauth.StartMCPOAuth(ctx, serverName, resourceURL)
	if err != nil {
		cancel()
		return SignInPrompt{}, err
	}

	pendingMCPSignIns.Lock()
	pendingMCPSignIns.byServer[serverName] = &pendingSignIn{pending: pending, ctx: ctx, cancel: cancel}
	pendingMCPSignIns.Unlock()

	return SignInPrompt{Provider: serverName, Kind: "browser", URL: pending.URL}, nil
}

// CompleteMCPSignIn finishes what StartMCPSignIn began: it blocks until the
// browser redirect lands (or CancelMCPSignIn unblocks it), stores the
// credential, and rebuilds the MCP manager so the server that was waiting on
// this credential can actually connect on the next tool call.
func (a *Engine) CompleteMCPSignIn(serverName string) error {
	pendingMCPSignIns.Lock()
	entry := pendingMCPSignIns.byServer[serverName]
	pendingMCPSignIns.Unlock()
	if entry == nil {
		return fmt.Errorf("no sign-in in progress for %s", serverName)
	}

	err := oauth.FinishMCPOAuth(entry.ctx, entry.pending)

	pendingMCPSignIns.Lock()
	delete(pendingMCPSignIns.byServer, serverName)
	pendingMCPSignIns.Unlock()
	entry.cancel()
	entry.pending.Cancel()

	if err != nil {
		return err
	}
	a.rebuildMCP()
	return nil
}

// CancelMCPSignIn abandons an in-flight MCP sign-in. Safe to call when
// nothing is in progress.
func (a *Engine) CancelMCPSignIn(serverName string) {
	pendingMCPSignIns.Lock()
	entry := pendingMCPSignIns.byServer[serverName]
	delete(pendingMCPSignIns.byServer, serverName)
	pendingMCPSignIns.Unlock()

	if entry != nil {
		entry.cancel()
		entry.pending.Cancel()
	}
}

// MCPSignInStatus reports whether one MCP server has a stored OAuth
// credential, and as whom — the same shape oauth.StatusFor already gives
// Settings for AI providers, read from the same store, since ${connect:id}
// resolves an MCP server's credential from exactly there.
func (a *Engine) MCPSignInStatus(serverName string) oauth.Status {
	return oauth.StatusFor(serverName)
}
