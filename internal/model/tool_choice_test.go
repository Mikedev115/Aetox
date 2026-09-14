package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

const okStreamFrame = "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

// opencode-go / muse-spark-1.3-contributor, 2026-09-15: every turn with tools
// answered 400 "Thinking mode does not support this tool_choice", at every
// effort, for two days. The same model on the same plan worked from opencode's
// own client, which never writes tool_choice. Aetox wrote "auto" on every
// turn — the spec's own default, so a field that changes nothing and can only
// be refused.
func TestAutoToolChoiceStaysOffTheWire(t *testing.T) {
	var seen atomic.Value
	provider := silentProvider(t, "opencode-go", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		_, has := payload["tool_choice"]
		seen.Store(has)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(okStreamFrame))
	})

	tools := []ToolDefinition{{Type: "function"}}
	if _, err := provider.StreamComplete(context.Background(),
		Request{Messages: []Message{{Role: RoleUser, Content: "ping"}}, Tools: tools, ToolChoice: "auto"}, nil, nil); err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	if seen.Load() == true {
		t.Error(`tool_choice:"auto" went on the wire; it is the default and the one field a thinking gate refuses`)
	}

	// "required" is a real constraint and still travels.
	if _, err := provider.StreamComplete(context.Background(),
		Request{Messages: []Message{{Role: RoleUser, Content: "ping"}}, Tools: tools, ToolChoice: "required"}, nil, nil); err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	if seen.Load() != true {
		t.Error(`tool_choice:"required" was dropped; only the default is implicit`)
	}
}

// The ladder's order matters: the gateway's sentence says both "thinking" and
// "tool_choice". Read reasoning-first, the replay drops an effort that was
// never the problem, sends the same tool_choice again, and shows the user the
// same 400 — which is exactly the report. The field the sentence names is the
// one to drop, and the effort stays.
func TestARefusedToolChoiceIsReplayedWithoutIt(t *testing.T) {
	var calls int32
	var second map[string]any
	// openai/gpt-5.2 rather than the reporting row: its effort table is in Go,
	// so reasoning_effort is on the wire here without a fetched catalog, and
	// the assertion below that the replay keeps it means something.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if atomic.AddInt32(&calls, 1) == 1 {
			if _, has := payload["reasoning_effort"]; !has {
				t.Error("the first request carried no reasoning_effort; the test cannot tell whether the replay kept it")
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"Thinking mode does not support this tool_choice"}}`))
			return
		}
		second = payload
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(okStreamFrame))
	}))
	t.Cleanup(server.Close)
	requireKey := false
	provider, err := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		Provider: "openai", Model: "gpt-5.2", BaseURL: server.URL, RequireAPIKey: &requireKey,
	})
	if err != nil {
		t.Fatalf("new provider failed: %v", err)
	}
	t.Cleanup(func() { refusedToolChoice.Delete("openai/gpt-5.2") })

	resp, err := provider.StreamComplete(context.Background(),
		Request{
			Messages:   []Message{{Role: RoleUser, Content: "ping"}},
			Tools:      []ToolDefinition{{Type: "function"}},
			ToolChoice: "required",
			Reasoning:  &ReasoningConfig{Effort: "high"},
		}, nil, nil)
	if err != nil {
		t.Fatalf("the replay did not rescue the turn: %v", err)
	}
	if resp.Text != "ok" {
		t.Errorf("text = %q, want the replay's answer", resp.Text)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls = %d, want one refusal and one replay", calls)
	}
	if _, has := second["tool_choice"]; has {
		t.Error("the replay still carried tool_choice, the field the 400 named")
	}
	if _, has := second["reasoning_effort"]; !has {
		t.Error("the replay dropped reasoning_effort, which the 400 never named")
	}
	if !provider.dropToolChoice("gpt-5.2") {
		t.Error("the refusal was not remembered; the next turn would pay for the same 400 again")
	}
}
