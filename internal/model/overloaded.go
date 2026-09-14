package model

import (
	"errors"
	"strings"
)

// ProviderOverloadedError is the provider saying, in its own words and after
// the headers were already sent, "not now".
//
// The 5xx family is retried by the transport (retryTransport) and named by
// providerDownError, and for a long time that looked like the whole of it. It
// is not: a streamed answer is HTTP 200 the moment its headers land, and a
// backend that sheds the request after that says so INSIDE the stream — an
// `error` or `response.failed` event on the Responses wire, an `error` event
// of type overloaded_error on Anthropic's. The transport has already returned
// by then, so its retry never sees it, and the runtime surfaced the event as a
// plain sentence, which the turn loop could not tell from "the request was
// wrong" and ended on.
//
// 14 ก.ย. 2026: a customer's worker on Codex met "Our servers are currently
// overloaded. Please try again later." four times in one hour, and every one
// ended the turn — no backoff at any layer, on the one failure whose whole
// meaning is "back off". The report that reached the owner read the ending as
// Aetox failing to hand the job to the worker at all, because the sentence the
// main model got back ("sub-agent failed: codex: …") gave it nothing else to
// say.
//
// So the runtimes now state this as a type, the way RateLimitError states a
// spent window, and cognitive.Agent decides what to do with it — ask again on
// a short backoff, on the same bookkeeping the dropped connection uses. Err is
// the sentence the turn used to end with, kept whole for when the retries are
// spent.
type ProviderOverloadedError struct {
	Provider string
	Err      error
}

func (e *ProviderOverloadedError) Error() string { return e.Err.Error() }
func (e *ProviderOverloadedError) Unwrap() error { return e.Err }

// IsProviderOverloaded reports whether err is the provider shedding load, as
// opposed to refusing the request or the plan.
func IsProviderOverloaded(err error) bool {
	var overloaded *ProviderOverloadedError
	return errors.As(err, &overloaded)
}

// OverloadedProvider names the provider behind an overload, or "" when err is
// not one.
func OverloadedProvider(err error) string {
	var overloaded *ProviderOverloadedError
	if !errors.As(err, &overloaded) {
		return ""
	}
	return overloaded.Provider
}

// overloadedInStream decides whether an error the provider wrote into its own
// stream is the momentary kind.
//
// The code is asked first, because it is the provider's own classification:
// OpenAI's server_error is "our side, transient", Anthropic's overloaded_error
// and api_error are the same two facts under their own names. The words are
// asked second, because the Codex backend has been seen to send the sentence
// with no code worth reading, and "overloaded" in a provider's own message is
// not ambiguous.
//
// What is deliberately NOT here: rate_limit_exceeded (a 429 by another route,
// and one the plan-window wait already knows how to read when it comes with a
// reset), insufficient_quota (no backoff clears it), invalid_request and the
// context-length family (repeating a wrong request only spends the quota
// again). A stream error with none of these marks stays what it always was — a
// sentence that ends the turn — because the cost of retrying a refusal three
// times is the user's money and thirty seconds of their patience, and this
// list is what keeps that from being the default.
func overloadedInStream(code, message string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "server_error", "overloaded", "overloaded_error", "api_error", "service_unavailable", "engine_overloaded":
		return true
	}
	lower := strings.ToLower(message)
	for _, mark := range []string{"overloaded", "try again later", "temporarily unavailable", "at capacity", "server is busy"} {
		if strings.Contains(lower, mark) {
			return true
		}
	}
	return false
}
