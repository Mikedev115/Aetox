package engine

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/cognitive"
	"github.com/Mikedev115/Aetox/internal/model"
)

// A reopened chat is handed the memory its model actually had — the tool
// call and its result — not the two lines of prose around them. This is the
// cost the owner saw on 16 ก.ย. 2026: a transcript rebuild drops everything
// between question and answer, the model reads it all again, and the provider
// bills the whole history as a cache miss.
func TestReopenedChatGetsItsModelsMemoryBack(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.appendTurn(a.cur(),
		SessionMessage{Role: "user", Text: "อ่าน gate.py ให้หน่อย", Time: "10:00"},
		SessionMessage{Role: "agent", Text: "อ่านแล้ว มี 3 ฟังก์ชัน", Time: "10:01"})
	id := a.cur().id
	transcript, err := a.SessionTranscript(id)
	if err != nil {
		t.Fatal(err)
	}
	held := []model.Message{
		{Role: model.RoleUser, Content: "อ่าน gate.py ให้หน่อย"},
		{Role: model.RoleAssistant, ToolCalls: []model.ToolCall{{ID: "c1", Type: "function",
			Function: model.FunctionCall{Name: "read", Arguments: `{"path":"gate.py"}`}}}},
		{Role: model.RoleTool, ToolCallID: "c1", Name: "read", Content: "def a():..."},
		{Role: model.RoleAssistant, Content: "อ่านแล้ว มี 3 ฟังก์ชัน"},
	}
	db, _ := a.database()
	writeContext(db, id, lastMessageID(transcript), held)

	got := a.historyFor(id, transcript)
	if len(got) != 4 || got[2].Role != model.RoleTool || got[2].Content != "def a():..." {
		t.Fatalf("historyFor = %+v, want the four held messages with the tool result intact", got)
	}
	if got[1].ToolCalls[0].ID != "c1" {
		t.Errorf("tool call id lost on the way through the store: %+v", got[1])
	}
}

// A memory written after a turn the user has since edited away describes a
// conversation that no longer exists. The stamp refuses it, and the rebuild
// takes the transcript — the old loss, taken once, on the row that could not
// be trusted.
func TestAMemoryOlderThanTheTranscriptIsRefused(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.appendTurn(a.cur(),
		SessionMessage{Role: "user", Text: "one", Time: "10:00"},
		SessionMessage{Role: "agent", Text: "1", Time: "10:00"})
	id := a.cur().id
	first, _ := a.SessionTranscript(id)
	db, _ := a.database()
	writeContext(db, id, lastMessageID(first), []model.Message{
		{Role: model.RoleUser, Content: "one"},
		{Role: model.RoleTool, ToolCallID: "x", Content: "stale"},
		{Role: model.RoleAssistant, Content: "1"},
	})
	// The transcript moves on without the store being told.
	a.appendTurn(a.cur(),
		SessionMessage{Role: "user", Text: "two", Time: "10:01"},
		SessionMessage{Role: "agent", Text: "2", Time: "10:01"})
	second, _ := a.SessionTranscript(id)

	got := a.historyFor(id, second)
	for _, m := range got {
		if m.Role == model.RoleTool {
			t.Fatalf("a stale memory was handed back: %+v", got)
		}
	}
	if len(got) != 4 || got[3].Content != "2" {
		t.Fatalf("expected the transcript rebuild (4 rows ending in %q), got %+v", "2", got)
	}
}

// The app died inside a turn: the question was stored, the next launch closed
// it with a marker row. The memory written one turn earlier still describes
// what the rebuild would keep — the rebuild skips the failed pair too — so it
// is handed back rather than thrown away with the crash.
func TestAMemoryFromBeforeAnInterruptedTurnStillCounts(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.appendTurn(a.cur(),
		SessionMessage{Role: "user", Text: "one", Time: "10:00"},
		SessionMessage{Role: "agent", Text: "1", Time: "10:00"})
	id := a.cur().id
	settled, _ := a.SessionTranscript(id)
	db, _ := a.database()
	held := []model.Message{
		{Role: model.RoleUser, Content: "one"},
		{Role: model.RoleTool, ToolCallID: "x", Content: "kept"},
		{Role: model.RoleAssistant, Content: "1"},
	}
	writeContext(db, id, lastMessageID(settled), held)
	if !a.openTurn(a.cur(), SessionMessage{Role: "user", Text: "two", Time: "10:01"}) {
		t.Fatal("openTurn refused")
	}
	if n := a.closeInterruptedTurns(); n != 1 {
		t.Fatalf("closeInterruptedTurns = %d, want 1", n)
	}
	after, _ := a.SessionTranscript(id)
	if len(after) != 4 {
		t.Fatalf("transcript has %d rows, want 4 (the pair, the question, the marker)", len(after))
	}

	got := a.historyFor(id, after)
	if len(got) != 3 || got[1].Content != "kept" {
		t.Fatalf("historyFor = %+v, want the memory written before the interrupted turn", got)
	}
}

// Deleting a chat deletes its memory: that row holds every tool result
// verbatim, more of the chat than the chat itself.
func TestDeletingAChatDeletesItsMemory(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.appendTurn(a.cur(),
		SessionMessage{Role: "user", Text: "one", Time: "10:00"},
		SessionMessage{Role: "agent", Text: "1", Time: "10:00"})
	id := a.cur().id
	transcript, _ := a.SessionTranscript(id)
	db, _ := a.database()
	writeContext(db, id, lastMessageID(transcript), []model.Message{{Role: model.RoleUser, Content: "one"}})
	a.startNewSession()
	if err := a.DeleteSession(id); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM session_context WHERE session_id = ?`, id).Scan(&n)
	if n != 0 {
		t.Fatalf("session_context still holds %d row(s) for a deleted chat", n)
	}
}

// Through endTurn, the door every turn leaves by: what the engine holds at
// the end of a turn is what the store has a moment later, stamped with the
// row that turn wrote — so the next LoadSession, in this process or the
// next, is handed exactly it.
func TestEndTurnWritesWhatTheModelRemembers(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	conv := a.cur()
	conv.agent = cognitive.NewAgent(cognitive.AgentConfig{SystemPrompt: "test prompt"})
	user := SessionMessage{Role: "user", Text: "ดู note.txt", Time: "10:00"}
	agent := SessionMessage{Role: "agent", Text: "4,213 บาท", Time: "10:00"}
	agent.ID = a.appendTurn(conv, user, agent)
	conv.transcript = append(conv.transcript, user, agent)
	conv.agent.RestoreHistory([]model.Message{
		{Role: model.RoleUser, Content: "ดู note.txt"},
		{Role: model.RoleAssistant, ToolCalls: []model.ToolCall{{ID: "r1", Type: "function",
			Function: model.FunctionCall{Name: "read", Arguments: `{"path":"note.txt"}`}}}},
		{Role: model.RoleTool, ToolCallID: "r1", Name: "read", Content: "ต้นทุนรวม 4,213 บาท"},
		{Role: model.RoleAssistant, Content: "4,213 บาท"},
	})

	a.endTurn(conv.id)

	transcript, _ := a.SessionTranscript(conv.id)
	got := a.storedContext(conv.id, transcript)
	if len(got) != 4 || got[2].Content != "ต้นทุนรวม 4,213 บาท" {
		t.Fatalf("storedContext after endTurn = %+v, want the four messages the engine held, system prompt excluded", got)
	}
	if got[0].Role != model.RoleUser || got[0].Content != "ดู note.txt" {
		t.Errorf("the system prompt must not be in the row; first stored message = %+v", got[0])
	}
}
