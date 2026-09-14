package engine

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/model"
)

// A continued chat (§282) is born with a row, named after what it continues,
// linked to it, and holding the picked points as its first message — before
// its first turn, which is the only kind of chat that has anything to show
// then.
func TestAContinuedChatIsBornLinkedAndNamedAfterItsOrigin(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	prev := a.cur()
	question := SessionMessage{Role: "user", Text: "ทำหน้า landing ให้หน่อย", Time: "10:00"}
	if !a.openTurn(prev, question) {
		t.Fatal("openTurn refused")
	}
	a.appendTurn(prev, question, SessionMessage{Role: "agent", Text: "ได้เลย", Time: "10:01"})

	conv := a.conversationContinuing(prev)
	if conv.continuedFrom != prev.id || conv.id == prev.id || conv.id == "" {
		t.Fatalf("continuation = %+v, want a fresh id linked to %s", conv, prev.id)
	}
	points := []string{"กำลังทำหน้า landing", "ค้าง: หน้า pricing"}
	if err := a.bearHandoff(conv, points); err != nil {
		t.Fatalf("bearHandoff: %v", err)
	}

	// The row: origin's title, origin's id.
	rows := a.ListSessions()
	var born *SessionMeta
	for i := range rows {
		if rows[i].ID == conv.id {
			born = &rows[i]
		}
	}
	if born == nil {
		t.Fatal("the continued chat is not in the history list — it has to exist before its first turn")
	}
	if born.ContinuedFrom != prev.id {
		t.Errorf("ContinuedFrom = %q, want %q", born.ContinuedFrom, prev.id)
	}
	if born.Title != sessionTitleFrom(question.Text) {
		t.Errorf("title = %q, want the origin's %q", born.Title, sessionTitleFrom(question.Text))
	}

	// The transcript: one handoff row, the points as a list, the origin named.
	got, err := a.SessionTranscript(conv.id)
	if err != nil {
		t.Fatalf("SessionTranscript: %v", err)
	}
	if len(got) != 1 || got[0].Role != handoffRole {
		t.Fatalf("transcript = %+v, want exactly one handoff row", got)
	}
	if got[0].Text != "- กำลังทำหน้า landing\n- ค้าง: หน้า pricing" {
		t.Errorf("row text = %q", got[0].Text)
	}
	if got[0].Origin == nil || got[0].Origin.ID != prev.id || got[0].Origin.Title != sessionTitleFrom(question.Text) {
		t.Errorf("origin = %+v, want the chat it continues", got[0].Origin)
	}

	// What the model sees on a rebuild: the points in the user's slot, framed
	// as prior context — the same shape a compaction summary takes.
	msgs := transcriptToModelMessages(got)
	if len(msgs) != 1 || msgs[0].Role != model.RoleUser {
		t.Fatalf("context = %+v, want one user-role message", msgs)
	}
	if !strings.HasPrefix(msgs[0].Content, handoffFraming) || !strings.Contains(msgs[0].Content, "ค้าง: หน้า pricing") {
		t.Errorf("context content = %q, want the framing over the points", msgs[0].Content)
	}
}

// The first real turn in a continued chat must not rename it or unlink it:
// openTurn's upsert touches neither column.
func TestTheFirstTurnKeepsAContinuedChatsNameAndLink(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	prev := a.cur()
	first := SessionMessage{Role: "user", Text: "เรื่องเดิม", Time: "10:00"}
	a.openTurn(prev, first)
	a.appendTurn(prev, first, SessionMessage{Role: "agent", Text: "ok", Time: "10:01"})

	conv := a.conversationContinuing(prev)
	if err := a.bearHandoff(conv, []string{"point"}); err != nil {
		t.Fatalf("bearHandoff: %v", err)
	}
	next := SessionMessage{Role: "user", Text: "คำถามใหม่ในแชทที่ต่อมา", Time: "10:05"}
	if !a.openTurn(conv, next) {
		t.Fatal("openTurn refused")
	}
	a.appendTurn(conv, next, SessionMessage{Role: "agent", Text: "ตอบ", Time: "10:06"})

	for _, m := range a.ListSessions() {
		if m.ID != conv.id {
			continue
		}
		if m.Title != sessionTitleFrom(first.Text) {
			t.Errorf("title became %q — the first turn renamed the continued chat", m.Title)
		}
		if m.ContinuedFrom != prev.id {
			t.Errorf("ContinuedFrom became %q — the first turn unlinked it", m.ContinuedFrom)
		}
	}
	got, _ := a.SessionTranscript(conv.id)
	if len(got) != 3 || got[0].Role != handoffRole || got[1].Role != "user" || got[2].Role != "agent" {
		t.Fatalf("transcript roles = %v, want handoff,user,agent", rolesOf(got))
	}
	// And the pairing every downstream reader assumes still holds from the
	// first real turn on: a failed pair lookup never lands on the handoff row.
	if failedPairAt(got, 0) {
		t.Error("the handoff row was read as half of a failed turn")
	}
}

func TestContinueInNewSessionRefusesNothingToCarry(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	if _, err := a.ContinueInNewSession(a.cur().id, []string{"", "  "}); err == nil {
		t.Fatal("blank points must be refused before anything is written")
	}
	if _, err := a.ContinueInNewSession("no-such-session", []string{"point"}); err == nil {
		t.Fatal("a session this process does not hold must be refused")
	}
	if len(a.ListSessions()) != 0 {
		t.Error("a refused continuation must leave no row behind")
	}
}

func TestDraftHandoffNeedsAModel(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	if _, err := a.DraftHandoff(a.cur().id); err == nil || !strings.Contains(err.Error(), "โมเดล") {
		t.Fatalf("err = %v, want the no-model refusal", err)
	}
}

func rolesOf(messages []SessionMessage) []string {
	out := make([]string, len(messages))
	for i, m := range messages {
		out[i] = m.Role
	}
	return out
}
