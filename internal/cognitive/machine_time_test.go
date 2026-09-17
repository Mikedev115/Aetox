package cognitive

import (
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
)

func TestCurrentMachineTimeIsFreshRequestContextNotHistory(t *testing.T) {
	messages := []model.Message{
		{Role: model.RoleSystem, Content: "system rules"},
		{Role: model.RoleUser, Content: "what time is it?"},
	}
	firstAt := time.Date(2026, time.September, 17, 9, 8, 7, 0, time.FixedZone("ICT", 7*60*60))
	secondAt := firstAt.Add(time.Minute)

	first := withCurrentMachineTime(messages, firstAt)
	second := withCurrentMachineTime(messages, secondAt)

	if !strings.Contains(first[0].Content, "Local time: 2026-09-17T09:08:07+07:00") {
		t.Fatalf("first request has no machine time: %q", first[0].Content)
	}
	if !strings.Contains(second[0].Content, "Local time: 2026-09-17T09:09:07+07:00") {
		t.Fatalf("second request did not refresh machine time: %q", second[0].Content)
	}
	if messages[0].Content != "system rules" {
		t.Fatalf("timestamp leaked into stored history: %q", messages[0].Content)
	}
}
