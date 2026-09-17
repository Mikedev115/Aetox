package botbridge

import "context"

// Message is one plain-text message after platform-specific filtering.  The
// callbacks keep wire details out of the engine and remain valid across a
// gateway reconnect because replies use the platform REST API.
type Message struct {
	Platform string
	Endpoint string
	Sender   string
	Text     string
	Reply    func(context.Context, string) error
	Typing   func(context.Context) error
}

type Handler func(context.Context, Message)

// Listen blocks until ctx ends or the platform connection fails.  The engine
// owns retry/backoff so connect, reconnect and disconnect have one lifecycle.
func Listen(ctx context.Context, platform string, handler Handler) error {
	switch platform {
	case Telegram:
		return listenTelegram(ctx, Token(platform), handler)
	case Discord:
		return listenDiscord(ctx, Token(platform), handler)
	default:
		return nil
	}
}
