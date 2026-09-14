package cognitive

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
)

// The handoff is compaction the user can see (§282): the whole transcript
// goes to the summarizer with no tool block, and a list comes back that the
// window can put checkboxes beside.
func TestHandoffPointsSendsTheWholeTranscriptAndReadsBackAList(t *testing.T) {
	provider := &toolLoopProvider{responses: []model.Response{
		{Text: "- กำลังทำหน้า landing ด้วย Go\n- ตัดสินใจใช้ SQLite ไม่ใช่ Postgres\n\n- ค้าง: หน้า pricing ยังไม่ได้เขียน"},
	}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxChars: 50000, SystemPrompt: "sys"})
	agent.RestoreHistory([]model.Message{
		{Role: model.RoleUser, Content: "q0 ทำหน้า landing"},
		{Role: model.RoleAssistant, Content: "a0 ได้เลย"},
		{Role: model.RoleUser, Content: "q1 ใช้ SQLite นะ"},
		{Role: model.RoleAssistant, Content: "a1 รับทราบ"},
	})

	points, err := agent.HandoffPoints(context.Background())
	if err != nil {
		t.Fatalf("HandoffPoints: %v", err)
	}
	want := []string{"กำลังทำหน้า landing ด้วย Go", "ตัดสินใจใช้ SQLite ไม่ใช่ Postgres", "ค้าง: หน้า pricing ยังไม่ได้เขียน"}
	if strings.Join(points, "|") != strings.Join(want, "|") {
		t.Fatalf("points = %q, want %q", points, want)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("expected one summarizer call, got %d", len(provider.requests))
	}
	req := provider.requests[0]
	if len(req.Tools) != 0 {
		t.Fatal("the summarizer must be called with no tool block")
	}
	if !strings.Contains(req.Messages[0].Content, "fresh chat") {
		t.Fatalf("system message is not the handoff prompt: %q", req.Messages[0].Content)
	}
	transcript := req.Messages[1].Content
	for _, turnText := range []string{"q0 ", "a0 ", "q1 ", "a1 "} {
		if !strings.Contains(transcript, turnText) {
			t.Errorf("transcript is missing %q — the WHOLE conversation goes, not compaction's old half", turnText)
		}
	}
	if strings.Contains(transcript, "sys") && strings.Contains(transcript, "--- system ---") {
		t.Error("the system prompt is the assistant's brief, not something the user said; it must not be summarized")
	}
}

func TestHandoffPointsRefusesAnEmptyConversation(t *testing.T) {
	provider := &toolLoopProvider{responses: []model.Response{{Text: "- nothing"}}}
	agent := NewAgent(AgentConfig{Provider: provider, Model: "test-model", MaxChars: 50000, SystemPrompt: "sys"})
	if _, err := agent.HandoffPoints(context.Background()); !errors.Is(err, ErrNothingToHandOff) {
		t.Fatalf("err = %v, want ErrNothingToHandOff", err)
	}
	if len(provider.requests) != 0 {
		t.Fatal("an empty conversation must not cost a model call")
	}
}

func TestParseHandoffPointsAcceptsTheMarkersModelsActuallyUse(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"dashes", "- one\n- two", []string{"one", "two"}},
		{"numbers", "1. one\n2) two\n10. ten", []string{"one", "two", "ten"}},
		{"bullets and stars", "• one\n* two", []string{"one", "two"}},
		{"preamble is dropped when there are bullets", "Here are the points:\n- one\n- two", []string{"one", "two"}},
		{"no markers at all keeps every line", "one\ntwo", []string{"one", "two"}},
		{"blank", "\n\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseHandoffPoints(c.text)
			if strings.Join(got, "|") != strings.Join(c.want, "|") {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseHandoffPointsCapsTheList(t *testing.T) {
	var b strings.Builder
	for i := 0; i < handoffMaxPoints+5; i++ {
		b.WriteString("- point\n")
	}
	if got := len(parseHandoffPoints(b.String())); got != handoffMaxPoints {
		t.Fatalf("kept %d points, want the cap of %d", got, handoffMaxPoints)
	}
}
