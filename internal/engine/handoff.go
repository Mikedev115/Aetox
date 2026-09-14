package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/cognitive"
	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// "สรุปแล้วไปเริ่มแชทใหม่" (DECISIONS §282): compaction the user can see.
//
// Compaction already folds a long chat into a paragraph — silently, for the
// model, when the window is nearly full. The owner's ask was the same fold with
// the person in the loop: press a button, get the conversation as a short list
// of points, untick the ones that no longer matter, and land in a fresh chat
// that starts from exactly those. Two doors, because there is a decision
// between them: DraftHandoff writes the list, ContinueInNewSession opens the
// chat with whatever came back ticked.
//
// The new chat is born at the same station as the one it continues — desk,
// chair, team, project, stance, and the model dials — because "continue" means
// the same room with a cleaner table, not a different room. And it is born
// with a row: the handoff is the one kind of chat that has something to show
// before its first turn, and a chat whose only content lives in memory until
// the user types is a chat a restart forgets.

const (
	// handoffRole is the transcript role of the one row a continued chat
	// starts with. Every reader that keys on "user"/"agent" — the title,
	// habit synthesis, the review, regenerate — skips it by construction, and
	// the two that must not (the context rebuild, the window) name it.
	handoffRole = "handoff"
	// handoffFraming is what the model sees above the points: the same
	// device memory.Context.ReplaceWithSummary uses for a compaction summary,
	// so a handed-off chat reads to the model exactly like a compacted one.
	handoffFraming = "[Points the user chose to carry over from an earlier chat — treat as prior context, not as a question]\n"
	// handoffWait bounds the summarizer call. One completion with no tools;
	// the reconnect budgets inside completeWithReconnect are what usually
	// decide, and this is the wall behind them.
	handoffWait = 2 * time.Minute
)

// DraftHandoff writes the open chat as a list of points for the user to pick
// from. It costs one model call and changes nothing: the chat it summarizes is
// left exactly as it was, and the list lives only in the window until
// ContinueInNewSession is handed the picked ones.
func (a *Engine) DraftHandoff(sessionID string) ([]string, error) {
	conv := a.liveConversation(sessionID)
	if conv == nil {
		return nil, fmt.Errorf("เปิดแชทนั้นก่อนแล้วค่อยสรุป")
	}
	if a.turnRunningIn(conv.id) {
		return nil, fmt.Errorf("รอให้คำตอบนี้จบก่อน")
	}
	if conv.agent == nil {
		return nil, fmt.Errorf("ยังไม่มีโมเดลให้สรุป — ต่อโมเดลก่อน")
	}
	ctx, cancel := context.WithTimeout(context.Background(), handoffWait)
	defer cancel()
	points, err := conv.agent.HandoffPoints(ctx)
	if errors.Is(err, cognitive.ErrNothingToHandOff) {
		return nil, fmt.Errorf("ยังไม่มีอะไรให้สรุป — คุยกันก่อนสักหนึ่งรอบ")
	}
	if err != nil {
		return nil, fmt.Errorf("สรุปไม่สำเร็จ: %w", err)
	}
	return points, nil
}

// ContinueInNewSession opens a fresh chat at the same station as sessionID,
// carrying the given points as its first row, and returns the new chat's id.
// The engine's open conversation becomes the new one — the caller loads it
// the way it loads any session, and LoadSession attaches to the live engine
// this built rather than rebuilding it.
//
// The chat being left is left, not closed: it stays in the history under its
// own title and everything it held, and the new one links back to it.
func (a *Engine) ContinueInNewSession(sessionID string, points []string) (string, error) {
	points = cleanHandoffPoints(points)
	if len(points) == 0 {
		return "", fmt.Errorf("เลือกอย่างน้อยหนึ่งประเด็นที่จะพาไป")
	}
	prev := a.liveConversation(sessionID)
	if prev == nil || prev.id == "" {
		return "", fmt.Errorf("เปิดแชทนั้นก่อนแล้วค่อยไปต่อ")
	}
	if a.turnRunningIn(prev.id) {
		return "", fmt.Errorf("รอให้คำตอบนี้จบก่อน")
	}
	conv := a.conversationContinuing(prev)
	// prev.cfg, not a.cfg: the dials of the chat being continued, which is
	// what "the same room" means when the user has moved the model in this
	// chat and not in the app's default (§155).
	a.applyConfig(conv, prev.cfg)
	if err := a.bearHandoff(conv, points); err != nil {
		return "", err
	}
	a.showConversation(conv)
	// Leaving a chat is the moment its review runs, same as LoadSession.
	a.maybeReviewSession(prev.id)
	debuglog.Info("handoff", fmt.Sprintf("%s -> %s (%d points)", prev.id, conv.id, len(points)))
	return conv.id, nil
}

// conversationContinuing is a new conversation at prev's station, linked to
// it. startNewSession's shape with the station copied whole instead of the
// window's desk alone: a continuation keeps the project and the stance too,
// because they are part of what the user was in the middle of.
func (a *Engine) conversationContinuing(prev *conversation) *conversation {
	conv := newConversation()
	conv.id = newSessionID()
	// The dials and the project root come along before applyConfig lays them
	// over again: the row is written from conv.cfg, and a continuation in
	// another project's key would be a chat the sidebar never lists.
	conv.cfg = prev.cfg
	conv.desk, conv.chair, conv.team = prev.desk, prev.chair, prev.team
	conv.space, conv.stance = prev.space, prev.stance
	conv.continuedFrom = prev.id
	return conv
}

// bearHandoff gives a conversation its first row — the points — in the store
// and in its engine's context. Split from ContinueInNewSession so the store
// half can be tested on an engine with no model behind it, which is the
// state most of this package's tests run in.
//
// The row is written before the first turn, which openTurn normally does,
// and with the ORIGIN's title: a continued chat is named after what it
// continues, and the sidebar's mark (SessionMeta.ContinuedFrom) is what
// tells the two apart. openTurn's later upsert keeps both (ON CONFLICT
// touches neither title nor continued_from).
func (a *Engine) bearHandoff(conv *conversation, points []string) error {
	db, err := a.database()
	if err != nil {
		return err
	}
	now := time.Now()
	title := ""
	_ = db.QueryRow(`SELECT title FROM sessions WHERE id = ?`, conv.continuedFrom).Scan(&title)
	if err := upsertSessionRow(db, conv, title, now.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("บันทึกแชทใหม่ไม่ได้: %w", err)
	}
	row := SessionMessage{Role: handoffRole, Text: handoffNote(points), Time: now.Format("15:04")}
	res, err := db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES(?,?,?,?)`,
		conv.id, row.Role, row.Text, row.Time)
	if err != nil {
		return fmt.Errorf("บันทึกประเด็นไม่ได้: %w", err)
	}
	row.ID, _ = res.LastInsertId()
	row.Origin = &SessionOrigin{ID: conv.continuedFrom, Title: title}
	conv.transcript = []SessionMessage{row}
	if conv.agent != nil {
		conv.agent.RestoreHistory(transcriptToModelMessages(conv.transcript))
	}
	return nil
}

// handoffNote is the row's text: the points as a markdown list, which is
// what the window draws and what the model reads under handoffFraming.
func handoffNote(points []string) string {
	var b strings.Builder
	for _, p := range points {
		b.WriteString("- ")
		b.WriteString(p)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// cleanHandoffPoints drops blanks and trims the rest — the window sends what
// the user ticked, and a ticked blank is nothing to carry.
func cleanHandoffPoints(points []string) []string {
	out := make([]string, 0, len(points))
	for _, p := range points {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// sessionOrigin is the chat a session continues, for the card at the top of
// its transcript, or nil for a chat that started from nothing. The origin's
// title comes back empty when that chat has since been deleted; the link is
// still real and the window words the gap.
func (a *Engine) sessionOrigin(id string) *SessionOrigin {
	db, err := a.database()
	if err != nil {
		return nil
	}
	var from string
	if db.QueryRow(`SELECT continued_from FROM sessions WHERE id = ?`, id).Scan(&from) != nil || from == "" {
		return nil
	}
	origin := &SessionOrigin{ID: from}
	_ = db.QueryRow(`SELECT title FROM sessions WHERE id = ?`, from).Scan(&origin.Title)
	return origin
}

// liveConversation is the conversation this process holds for id — the open
// one included, which is the usual answer, since the button lives in the
// chat on screen.
func (a *Engine) liveConversation(id string) *conversation {
	if id = strings.TrimSpace(id); id == "" {
		return nil
	}
	// cur() first: it is what builds the manager on a zero Engine.
	if cur := a.cur(); cur.id == id {
		return cur
	}
	return a.convs.find(id)
}
