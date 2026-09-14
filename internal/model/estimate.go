package model

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/gif"  // ImageTokens reads attachment headers; the same set
	_ "image/jpeg" // loadPicture validates (png is registered by image_fit)
)

// PromptEstimate is a request's size as Aetox guesses it before the provider
// has counted: chars/4 for each part, attachments at their own rates.
//
// Three parts rather than one number because they do not err alike. System and
// Tools are the same bytes on every round of a session — the fixed floor — and
// Messages is the part that grows. The provider's tokenizer packs the tool
// block's JSON (repeated keys, braces) more tightly than Thai prose, so one
// ratio for the whole request is wrong twice over. Keeping the parts apart is
// what lets a later measurement say how far each was off (Engine's
// promptCalibration).
type PromptEstimate struct {
	System   int // the system prompt
	Tools    int // the tool definitions as sent
	Messages int // everything after the system prompt, attachments included
}

// Fixed is the part of the request that rides along unchanged with every
// message of a session.
func (e PromptEstimate) Fixed() int { return e.System + e.Tools }

// Total is the whole guessed request.
func (e PromptEstimate) Total() int { return e.System + e.Tools + e.Messages }

// IsZero reports an estimate nobody made — the value a Usage carries when the
// round that produced it did not send the conversation (an ephemeral title
// request, a probe).
func (e PromptEstimate) IsZero() bool { return e == PromptEstimate{} }

// EstimatePrompt guesses what one request costs from its bytes.
//
// Everything a message carries, not just Content: reasoning rides back out on
// the wire for providers that take it (openai_compatible resends it with the
// history), and an attached screenshot was the single biggest thing this used
// to count as zero — a pasted image read as a free message while costing more
// than the text around it.
func EstimatePrompt(msgs []Message, tools []ToolDefinition) PromptEstimate {
	est := func(chars int) int { return (chars + 3) / 4 }

	systemChars, msgChars, attachTokens := 0, 0, 0
	for i, m := range msgs {
		chars := len(m.Content) + len(m.ReasoningContent)
		for _, tc := range m.ToolCalls {
			chars += len(tc.Function.Arguments)
		}
		for _, img := range m.Images {
			attachTokens += ImageTokens(img)
		}
		// A document's real price is per page and nothing here can count
		// pages cheaply, so this is a floor — roughly one page — rather than
		// a guess dressed as a measurement. The measured total from the next
		// round absorbs the true cost either way.
		attachTokens += 1500 * len(m.Documents)
		if i == 0 && m.Role == RoleSystem {
			systemChars = chars
		} else {
			msgChars += chars
		}
	}

	toolChars := 0
	if len(tools) > 0 {
		if b, err := json.Marshal(tools); err == nil {
			toolChars = len(b)
		}
	}

	return PromptEstimate{
		System:   est(systemChars),
		Tools:    est(toolChars),
		Messages: est(msgChars) + attachTokens,
	}
}

// ImageTokens estimates what one attached image adds to the next request.
//
// Not bytes/4: a vision model prices an image by its pixels, not its file
// size, and the two disagree by an order of magnitude in both directions — a
// 100KB screenshot costs ~1.3k tokens, which bytes/4 would call 25k. The
// working rule is Anthropic's width×height/750, and providers downscale
// anything past ~1.6k tokens, hence the cap. DecodeConfig reads only the
// header, so this costs nothing per call.
//
// The flat fallback is for a format the header-read cannot parse (webp): the
// size of a typical screenshot, chosen over zero because zero is the exact lie
// this estimate exists to stop telling.
func ImageTokens(img Image) int {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(img.Data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 1500
	}
	tok := (cfg.Width*cfg.Height + 749) / 750
	if tok > 1600 {
		tok = 1600
	}
	return tok
}
