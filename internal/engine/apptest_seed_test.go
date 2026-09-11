package engine

// seed puts a conversation on the Engine's screen, for tests that used to build
// one by setting Engine fields directly.
//
// Those fields moved into `conversation` on 2026-08-19 (desktop/conversation.go)
// because none of them was ever a property of the app. The tests that set them
// were saying "an app whose open chat is this" all along; this says it in the
// words the code now uses, and nothing about what they assert changed.
func seed(a *Engine, conv *conversation) *Engine {
	// The chat inherits the app's config unless the test gave it one of its own.
	// That is what production does — a conversation is built from the template
	// in Engine.cfg (applyConfig) — and it keeps every test that sets `cfg:` on the
	// Engine literal saying what it has always said, now that the readers ask the
	// conversation rather than the app (DECISIONS §155).
	if conv.cfg.SandboxRoot == "" && conv.cfg.ModelProvider == "" {
		conv.cfg = a.cfg
	}
	a.convs = newConversations()
	a.convs.show(conv)
	return a
}
