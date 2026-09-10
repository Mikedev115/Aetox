package main

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Mikedev115/Aetox/internal/learned"
	_ "modernc.org/sqlite"
)

type fakeReviewer struct {
	facts []ReviewFact
	err   error
}

func (f fakeReviewer) Review(ctx context.Context, userMessages []string) ([]ReviewFact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return f.facts, f.err
}

func setupReviewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}

	schema := `
	CREATE TABLE messages (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  session_id TEXT NOT NULL,
	  role TEXT NOT NULL,
	  text TEXT NOT NULL,
	  time TEXT NOT NULL
	);
	CREATE TABLE pending_changes (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  kind TEXT NOT NULL,
	  scope TEXT NOT NULL,
	  target TEXT NOT NULL DEFAULT '',
	  op TEXT NOT NULL,
	  before TEXT NOT NULL DEFAULT '',
	  body TEXT NOT NULL,
	  reason TEXT NOT NULL DEFAULT '',
	  evidence TEXT NOT NULL DEFAULT '',
	  source TEXT NOT NULL DEFAULT '',
	  state TEXT NOT NULL DEFAULT 'pending',
	  created_at TEXT NOT NULL,
	  decided_at TEXT NOT NULL DEFAULT ''
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create test schema failed: %v", err)
	}
	return db
}

func TestGetUserMessagesForSession(t *testing.T) {
	db := setupReviewTestDB(t)
	defer db.Close()

	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s1', 'user', 'ข้อความแรก', '10:00')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s1', 'agent', 'ตอบรับ', '10:01')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s1', 'user', 'ข้อความสอง', '10:02')`)

	msgs, err := getUserMessagesForSession(db, "s1")
	if err != nil {
		t.Fatalf("getUserMessagesForSession failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 user messages, got %d", len(msgs))
	}
	if msgs[0] != "ข้อความแรก" || msgs[1] != "ข้อความสอง" {
		t.Errorf("unexpected messages: %v", msgs)
	}
}

func TestRunSessionReview(t *testing.T) {
	db := setupReviewTestDB(t)
	defer db.Close()

	// Seed 2 user messages
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('sess-abc', 'user', 'ผมทำงาน Full-stack dev นะ', '10:00')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('sess-abc', 'agent', 'รับทราบครับ', '10:01')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('sess-abc', 'user', 'ตอบแบบสรุปสั้นๆ พอ ไม่ต้องเกริ่น', '10:02')`)

	app := &App{db: db}

	reviewer := fakeReviewer{
		facts: []ReviewFact{
			{
				Text: "User is a full-stack developer",
				Why:  "Stated in message 1",
			},
			{
				Text: "User prefers concise summaries without preamble",
				Why:  "Stated in message 2",
			},
		},
	}

	ctx := context.Background()
	n, err := app.runSessionReviewWith(ctx, reviewer, "sess-abc")
	if err != nil {
		t.Fatalf("runSessionReviewWith failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 proposed facts, got %d", n)
	}

	// Verify rows in pending_changes
	rows, err := db.Query(`SELECT kind, scope, source, body, state FROM pending_changes WHERE evidence = 'session:sess-abc'`)
	if err != nil {
		t.Fatalf("query pending_changes failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var kind, scope, source, body, state string
		if err := rows.Scan(&kind, &scope, &source, &body, &state); err != nil {
			t.Fatalf("scan row failed: %v", err)
		}
		count++
		if kind != "memory" {
			t.Errorf("expected kind=memory, got %s", kind)
		}
		if scope != learned.UserScope {
			t.Errorf("expected scope=%s, got %s", learned.UserScope, scope)
		}
		if source != "review" {
			t.Errorf("expected source=review, got %s", source)
		}
		if state != "pending" {
			t.Errorf("expected state=pending, got %s", state)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 rows in pending_changes, got %d", count)
	}

	// Running review again should not create duplicate proposals
	n2, err := app.runSessionReviewWith(ctx, reviewer, "sess-abc")
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if n2 != 0 {
		t.Errorf("expected 0 new proposals on re-run, got %d", n2)
	}
}

func TestSessionReviewUnderTwoMessagesSkipped(t *testing.T) {
	db := setupReviewTestDB(t)
	defer db.Close()

	// Only 1 user message
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s-short', 'user', 'สวัสดี', '10:00')`)

	app := &App{db: db}
	reviewer := fakeReviewer{
		facts: []ReviewFact{{Text: "Something", Why: "Why"}},
	}

	n, err := app.runSessionReviewWith(context.Background(), reviewer, "s-short")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected review to be skipped (< 2 user messages), but got %d proposals", n)
	}
}

func TestSessionReviewPreemption(t *testing.T) {
	db := setupReviewTestDB(t)
	defer db.Close()

	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s-cancel', 'user', 'msg 1', '10:00')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s-cancel', 'user', 'msg 2', '10:01')`)

	app := &App{db: db}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel context simulating user sending message

	reviewer := fakeReviewer{
		facts: []ReviewFact{{Text: "Fact", Why: "Why"}},
	}

	_, err := app.runSessionReviewWith(ctx, reviewer, "s-cancel")
	if err == nil {
		t.Errorf("expected context cancellation error")
	}
}
