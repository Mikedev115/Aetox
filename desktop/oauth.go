package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/oauth"
)

// Sign-in bindings — "use the plan you already pay for" instead of pasting a
// key. The flows themselves live in internal/oauth; everything here is the
// screen's half: hold the in-flight authorization between the two calls the
// UI makes, and tell the engine when a sign-in lands on the provider that is
// currently active. The store these write (oauth.json) is the screen's, like
// credentials.json: the engine never reads a token out of it (§248, engine
// deps_test.go), it asks the screen to sign (provider_forward.go).

// pendingSignIns holds authorizations between StartSignIn and CompleteSignIn.
// Package-level rather than a field on App because there is exactly one App
// per process.
var pendingSignIns = struct {
	sync.Mutex
	byProvider map[string]*pendingSignIn
}{byProvider: map[string]*pendingSignIn{}}

type pendingSignIn struct {
	pending *oauth.Pending
	// ctx spans both calls: CompleteSignIn blocks on it, so CancelSignIn's
	// cancel is what actually unblocks a user who changed their mind while a
	// device flow was still polling.
	ctx    context.Context
	cancel context.CancelFunc
}

// SignInMethods lists every provider Aetox can sign into, with the risk note
// the UI must show before the user commits.
func (a *App) SignInMethods() []oauth.Method {
	return oauth.Methods()
}

// SignInStatus reports whether one provider is signed in, and as whom. It
// never returns a token.
func (a *App) SignInStatus(providerName string) oauth.Status {
	return oauth.StatusFor(providerName)
}

// StartSignIn opens a sign-in and returns what to show the user. Nothing is
// stored until CompleteSignIn succeeds.
func (a *App) StartSignIn(providerName string) (engine.SignInPrompt, error) {
	canonical := model.NormalizeProvider(providerName)
	method, ok := oauth.MethodFor(canonical)
	if !ok {
		return engine.SignInPrompt{}, fmt.Errorf("%s has no sign-in — add an API key instead", canonical)
	}

	// Starting a second sign-in for the same provider abandons the first
	// rather than leaking its listener: the user clicking the button twice
	// means they gave up on the first attempt.
	a.CancelSignIn(canonical)

	ctx, cancel := context.WithCancel(context.Background())
	pending, err := oauth.Start(ctx, canonical)
	if err != nil {
		cancel()
		return engine.SignInPrompt{}, err
	}

	pendingSignIns.Lock()
	pendingSignIns.byProvider[canonical] = &pendingSignIn{pending: pending, ctx: ctx, cancel: cancel}
	pendingSignIns.Unlock()

	return engine.SignInPrompt{
		Provider:        canonical,
		Kind:            method.Kind,
		URL:             pending.URL,
		UserCode:        pending.UserCode,
		VerificationURI: pending.VerificationURI,
	}, nil
}

// CompleteSignIn finishes what StartSignIn began and stores the credential.
//
// It blocks: a device flow polls until the user approves in their browser, and
// a browser flow waits for the redirect. The UI shows the prompt and awaits
// this call; CancelSignIn is how the user backs out.
//
// pasted carries the code for providers that make the user copy one and is
// ignored by the rest.
func (a *App) CompleteSignIn(providerName, pasted string) (engine.ModelInfo, error) {
	canonical := model.NormalizeProvider(providerName)

	pendingSignIns.Lock()
	entry := pendingSignIns.byProvider[canonical]
	pendingSignIns.Unlock()
	if entry == nil {
		return engine.ModelInfo{}, fmt.Errorf("no %s sign-in in progress", canonical)
	}

	err := oauth.Finish(entry.ctx, entry.pending, strings.TrimSpace(pasted))

	pendingSignIns.Lock()
	delete(pendingSignIns.byProvider, canonical)
	pendingSignIns.Unlock()
	entry.cancel()
	entry.pending.Cancel()

	if err != nil {
		return engine.ModelInfo{}, err
	}
	return a.api.ProviderCredentialChanged(canonical)
}

// CancelSignIn abandons an in-flight sign-in and frees whatever it holds.
// Safe to call when nothing is in progress.
func (a *App) CancelSignIn(providerName string) {
	canonical := model.NormalizeProvider(providerName)

	pendingSignIns.Lock()
	entry := pendingSignIns.byProvider[canonical]
	delete(pendingSignIns.byProvider, canonical)
	pendingSignIns.Unlock()

	if entry != nil {
		entry.cancel()
		entry.pending.Cancel()
	}
}

// ImportableSignIns lists providers whose official CLI already holds a session
// on this machine, so Settings can offer "use the one you have" instead of a
// second authorization for the same account.
//
// Three entries: the Codex CLI writes a session Aetox can adopt, so someone
// already signed into it never authorizes the same ChatGPT account twice, and
// the Copilot and Kilo CLIs do the same for theirs. Settings hides the button on
// an empty list.
func (a *App) ImportableSignIns() []string {
	var out []string
	if oauth.CodexCLIAvailable() {
		out = append(out, "codex")
	}
	if oauth.CopilotCLIAvailable() {
		out = append(out, "github-copilot")
	}
	if oauth.KiloCLIAvailable() {
		out = append(out, "kilo")
	}
	return out
}

// ImportSignIn adopts that existing session. Explicit action only — the button
// says which tool it is reading from.
func (a *App) ImportSignIn(providerName string) (engine.ModelInfo, error) {
	canonical := model.NormalizeProvider(providerName)

	var err error
	switch canonical {
	case "codex":
		err = oauth.ImportCodexCLI()
	case "github-copilot":
		err = oauth.ImportCopilotCLI(context.Background())
	case "kilo":
		err = oauth.ImportKiloCLI(context.Background())
	default:
		return engine.ModelInfo{}, fmt.Errorf("%s has no session to import", canonical)
	}
	if err != nil {
		return engine.ModelInfo{}, err
	}
	return a.api.ProviderCredentialChanged(canonical)
}

// SignOut forgets a provider's credential. If it was the active provider the
// engine is re-bootstrapped, which is what surfaces "needs credentials" in the
// UI instead of failing on the user's next message.
func (a *App) SignOut(providerName string) (engine.ModelInfo, error) {
	canonical := model.NormalizeProvider(providerName)
	a.CancelSignIn(canonical)
	if err := oauth.Logout(canonical); err != nil {
		return engine.ModelInfo{}, err
	}
	return a.api.ProviderCredentialChanged(canonical)
}
