package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/Mikedev115/Aetox/internal/cliagent"
)

// ExternalCLIProvider stands in the engine for a provider whose turns are
// answered by a program on the user's machine (internal/cliagent).
//
// It is not a model client, and a chat on such a provider never reaches it for
// an answer: the desktop hands the turn to the engine before its own executor
// runs. What is left is the one thing every provider row is asked the same
// way — the connection test — and this answers it by asking the engine whether
// a turn could run at all. No request, no tokens: the settings page's
// ทดสอบการเชื่อมต่อ and the queued-switch preflight (§232) get to prove what
// the first turn would otherwise trip over.
//
// Anything else — the CLI build of Aetox, a path that forgot the desktop's
// branch — gets ErrExternalCLIOutsideDesktop rather than a silent nothing,
// because a turn that ran through here would be a turn that spent nothing and
// produced no answer.
type ExternalCLIProvider struct {
	provider string
	model    string
}

// ErrExternalCLIOutsideDesktop is the refusal above.
var ErrExternalCLIOutsideDesktop = errors.New("this provider answers only through the desktop's external engine")

// NewExternalCLIProvider builds the placeholder for a canonical provider id and
// the model name its catalog row uses.
func NewExternalCLIProvider(provider, model string) *ExternalCLIProvider {
	return &ExternalCLIProvider{provider: provider, model: model}
}

func (p *ExternalCLIProvider) Name() string { return p.provider }

// Complete answers the probe's one-token "ping" by asking the engine whether it
// is ready, and refuses everything else.
func (p *ExternalCLIProvider) Complete(ctx context.Context, req Request) (Response, error) {
	if !isProbeRequest(req) {
		return Response{}, ErrExternalCLIOutsideDesktop
	}
	engine, ok := cliagent.For(p.provider)
	if !ok {
		// A catalog row on this runtime with no engine behind it: a wiring
		// mistake in this program, said plainly rather than as a failed login.
		return Response{}, fmt.Errorf("ยังไม่มีเครื่องยนต์สำหรับ %q ในบิลด์นี้", p.provider)
	}
	if st := engine.Probe(ctx); !st.Ready {
		if st.Detail != "" {
			return Response{}, errors.New(st.Detail)
		}
		return Response{}, fmt.Errorf("%s ยังไม่พร้อมใช้งาน", engine.Label())
	}
	return Response{Provider: p.provider, Model: modelOr(req.Model, p.model), Text: "ok"}, nil
}

// SupportsReasoning: an engine's effort dial is the thinking level, and the
// catalog row says whether the row has one.
func (p *ExternalCLIProvider) SupportsReasoning() bool {
	_, canReason := providerReasoningCapability(p.provider)
	return canReason
}

// isProbeRequest recognises what probeProvider sends: one user message saying
// "ping", with room for a single token.
func isProbeRequest(req Request) bool {
	return req.MaxTokens == 1 &&
		len(req.Messages) == 1 &&
		req.Messages[0].Role == RoleUser &&
		req.Messages[0].Content == "ping"
}
