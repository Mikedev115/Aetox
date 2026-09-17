package prompt

import "strings"

// Transport is the external chat surface carrying one conversation. It is
// deliberately separate from Surface: the engine is still the desktop engine
// with the same identity, tools and permissions; only the delivery context is
// different. The zero value is every ordinary Aetox conversation.
type Transport string

const (
	TransportTelegram Transport = "telegram"
	TransportDiscord  Transport = "discord"
)

// WithTransport appends the small, channel-specific layer to an already built
// system prompt. The main prompt assembler remains byte-for-byte unchanged for
// ordinary conversations, and this layer comes last so its delivery facts can
// correct desktop-only presentation advice without replacing identity or desk
// direction.
func WithTransport(base string, transport Transport) string {
	layer := transportLayer(transport)
	if layer == "" {
		return base
	}
	return strings.TrimRight(base, "\n") + "\n\n" + layer
}

func transportLayer(transport Transport) string {
	common := "You are still the same Aetox main assistant, with the same identity, conversation memory, tools and permissions; this channel does not create another assistant. " +
		"The person receives your answer in this external chat, not in the Aetox chat window. Ask necessary follow-up questions directly in your reply and do not rely on an Aetox-only picker, panel or button for the answer. " +
		"If work creates or changes a file, summarize the result here and name its path instead of claiming that the person can see a desktop panel."

	switch transport {
	case TransportTelegram:
		return "## Conversation channel: Telegram\n" +
			"This conversation is being carried through the connected Telegram bot. Telegram replies are sent as plain text without a Markdown parse mode, so use clear text and simple bullets rather than relying on rendered Markdown, HTML, SVG or LaTeX. " +
			common + " Commands such as /new and /help are handled by the channel before a message reaches you.\n"
	case TransportDiscord:
		return "## Conversation channel: Discord\n" +
			"This conversation is being carried through the connected Discord bot, either in a direct message or a server channel. Discord Markdown is available, but there is no Aetox desktop panel in this delivery path. " +
			common + " In a server, the bot mention is removed before the message reaches you. Commands such as /new and /help are handled by the channel before a message reaches you.\n"
	default:
		return ""
	}
}
