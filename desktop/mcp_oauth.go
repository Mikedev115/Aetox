package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/Mikedev115/Aetox/internal/oauth"
)

// MCP server sign-in bindings — deliberately separate from oauth.go's
// StartSignIn/CompleteSignIn rather than sharing them.
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
// The flow itself is generic (internal/oauth/mcpauth.go discovers everything
// from the server's own URL), so unlike StartSignIn there is no fixed
// registry of providers to check against here — any server name reaches the
// same discovery, and a server whose authorization server has no dynamic
// client registration fails with a specific, readable error rather than
// "unknown provider".

var pendingMCPSignIns = struct {
	sync.Mutex
	byServer map[string]*pendingSignIn
}{byServer: map[string]*pendingSignIn{}}

// StartMCPSignIn discovers the server's OAuth metadata, registers Aetox as a
// client, and returns what to show the user while the browser sign-in is in
// flight. Nothing is stored until CompleteMCPSignIn succeeds.
func (a *App) StartMCPSignIn(serverName, resourceURL string) (SignInPrompt, error) {
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
func (a *App) CompleteMCPSignIn(serverName string) error {
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
func (a *App) CancelMCPSignIn(serverName string) {
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
func (a *App) MCPSignInStatus(serverName string) oauth.Status {
	return oauth.StatusFor(serverName)
}
