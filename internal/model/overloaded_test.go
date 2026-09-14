package model

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The customer's log, 14 ก.ย. 2026: the Codex backend answered 200, opened the
// stream, and then wrote "Our servers are currently overloaded. Please try
// again later." into it. The transport's retry had already returned with the
// headers; what the runtime handed up was a sentence the turn could not tell
// from a refusal, so the turn ended. It is a typed overload now.
func TestResponsesTypesAnOverloadWrittenIntoTheStream(t *testing.T) {
	for name, event := range map[string]string{
		// The plain error event, code and message beside the type.
		"error, top level": `data: {"type":"error","code":"server_error","message":"Our servers are currently overloaded. Please try again later.","sequence_number":3}`,
		// The same event with the pair under an `error` key, which the Codex
		// backend has also been seen to send.
		"error, nested": `data: {"type":"error","error":{"code":"server_error","message":"Our servers are currently overloaded. Please try again later."}}`,
		// A response that got as far as failing.
		"response.failed": `data: {"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","message":"Our servers are currently overloaded. Please try again later."}}}`,
		// No code worth reading; the words alone say what it is.
		"words only": `data: {"type":"error","message":"Our servers are currently overloaded. Please try again later."}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := responsesServer(t, sseLines(`data: {"type":"response.output_text.delta","delta":"partial"}`, event), nil)
			defer server.Close()
			p, _ := NewResponsesProvider(ResponsesConfig{Provider: "codex", Model: "m", BaseURL: server.URL, APIKey: "tok"})
			_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
			if !IsProviderOverloaded(err) {
				t.Fatalf("err = %v, want it typed as the provider overloaded", err)
			}
			if OverloadedProvider(err) != "codex" {
				t.Errorf("provider = %q, want codex", OverloadedProvider(err))
			}
			// The sentence is kept whole: it is what the turn ends with once
			// the retries are spent, and the debug log's evidence either way.
			if !strings.Contains(err.Error(), "codex: Our servers are currently overloaded") {
				t.Errorf("err = %v, want the provider's own words under the type", err)
			}
		})
	}
}

// Not every error in a stream is the provider's fault, and retrying the ones
// that are the request's fault spends the user's quota three more times on
// the same refusal. These stay the plain sentence they always were.
func TestResponsesLeavesARefusalInTheStreamUntyped(t *testing.T) {
	for name, event := range map[string]string{
		"invalid request":  `data: {"type":"error","code":"invalid_request_error","message":"Invalid value for 'tools[0]'"}`,
		"context length":   `data: {"type":"response.failed","response":{"error":{"code":"context_length_exceeded","message":"This model's maximum context length is 200000 tokens"}}}`,
		"quota":            `data: {"type":"error","code":"insufficient_quota","message":"You exceeded your current quota"}`,
		"the old test's":   `data: {"type":"response.failed","response":{"error":{"message":"the model refused this input"}}}`,
		"incomplete stays": `data: {"type":"response.incomplete","response":{"status":"incomplete","error":{"message":"overloaded"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := responsesServer(t, sseLines(event), nil)
			defer server.Close()
			p, _ := NewResponsesProvider(ResponsesConfig{Provider: "codex", Model: "m", BaseURL: server.URL, APIKey: "tok"})
			_, err := p.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}})
			if err == nil {
				t.Fatal("the failure was swallowed")
			}
			if IsProviderOverloaded(err) {
				t.Errorf("err = %v was typed as an overload; retrying it would spend the quota on the same refusal", err)
			}
		})
	}
}

// Anthropic's 529 has a stream-side twin: an `error` event of type
// overloaded_error after the headers. Same fact, same type.
func TestAnthropicTypesAnOverloadWrittenIntoTheStream(t *testing.T) {
	for name, tc := range map[string]struct {
		event      string
		overloaded bool
	}{
		"overloaded_error": {`{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`, true},
		"api_error":        {`{"type":"error","error":{"type":"api_error","message":"Internal server error"}}`, true},
		"invalid_request":  {`{"type":"error","error":{"type":"invalid_request_error","message":"messages: roles must alternate"}}`, false},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-haiku-4-5\",\"usage\":{\"input_tokens\":4}}}\n\n"))
				_, _ = w.Write([]byte("data: " + tc.event + "\n\n"))
			}))
			defer server.Close()
			provider, err := NewAnthropicProvider(AnthropicConfig{Model: "claude-haiku-4-5", APIKey: "k", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.StreamComplete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "ping"}}}, nil, nil)
			if err == nil {
				t.Fatal("the stream error was swallowed")
			}
			if IsProviderOverloaded(err) != tc.overloaded {
				t.Errorf("err = %v; typed as overloaded = %v, want %v", err, !tc.overloaded, tc.overloaded)
			}
		})
	}
}

// The classifier's two halves: the provider's own code first, its words
// second, and neither for the failures no backoff can clear.
func TestOverloadedInStreamReadsCodeThenWords(t *testing.T) {
	for _, tc := range []struct {
		code, message string
		want          bool
	}{
		{"server_error", "", true},
		{"SERVER_ERROR", "", true},
		{"overloaded_error", "Overloaded", true},
		{"api_error", "Internal server error", true},
		{"", "Our servers are currently overloaded. Please try again later.", true},
		{"", "The engine is temporarily unavailable", true},
		{"rate_limit_exceeded", "Rate limit reached for gpt-4", false},
		{"insufficient_quota", "You exceeded your current quota", false},
		{"invalid_request_error", "messages: roles must alternate", false},
		{"", "the provider ended the response without saying why", false},
		{"", "", false},
	} {
		if got := overloadedInStream(tc.code, tc.message); got != tc.want {
			t.Errorf("overloadedInStream(%q, %q) = %v, want %v", tc.code, tc.message, got, tc.want)
		}
	}
}

// The type is a fact for cognitive.Agent and nothing else: a caller that only
// has the sentence still has the sentence, and errors.Is through it still
// finds whatever was underneath.
func TestProviderOverloadedErrorKeepsItsCause(t *testing.T) {
	cause := errors.New("codex: Our servers are currently overloaded. Please try again later.")
	err := &ProviderOverloadedError{Provider: "codex", Err: cause}
	if err.Error() != cause.Error() {
		t.Errorf("Error() = %q, want the cause's own words", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Error("the cause is not reachable through the type")
	}
	if IsProviderOverloaded(cause) || OverloadedProvider(cause) != "" {
		t.Error("a plain sentence must not read as an overload")
	}
}
