package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/learned"
	_ "modernc.org/sqlite"
)

type fakeReviewer struct {
	facts []ReviewFact
	err   error
	// seen is what the reviewer was shown, for the tests that check it was
	// shown the user's decisions.
	seen *reviewInput
}

func (f fakeReviewer) Review(ctx context.Context, in reviewInput) ([]ReviewFact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.seen != nil {
		*f.seen = in
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

// The measurement that changed the design (11 ก.ย.): fifty review proposals,
// two approved, the same fact refused in six wordings. The reviewer is now
// shown what the user already decided — and the door behind it refuses a
// restatement whether or not the model listened.
func TestReviewIsShownTheDecisionsAndCannotRestateThem(t *testing.T) {
	db := setupReviewTestDB(t)
	defer db.Close()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())

	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s-1', 'user', 'ผมพิมพ์ไทยนะครับ', '10:00')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s-1', 'user', 'ช่วยดู CI ให้หน่อย', '10:01')`)
	// What the user already said: one refusal, one line waiting.
	_, _ = db.Exec(`INSERT INTO pending_changes(kind, scope, op, body, source, state, created_at, decided_at)
		VALUES('memory', 'user:profile', 'add', 'User communicates in Thai and expects replies in Thai', 'review', 'rejected', '2026-09-09T10:00:00Z', '2026-09-09T11:00:00Z')`)
	_, _ = db.Exec(`INSERT INTO pending_changes(kind, scope, op, body, source, state, created_at)
		VALUES('memory', 'user:profile', 'add', 'User runs CI on GitHub Actions', 'agent', 'pending', '2026-09-10T10:00:00Z')`)

	app := &App{db: db}
	var seen reviewInput
	reviewer := fakeReviewer{
		seen: &seen,
		facts: []ReviewFact{
			// The refused fact in other words — the sixth spelling.
			{Text: "User communicates primarily in Thai and expects responses in Thai.", Why: "message 1"},
			// The waiting fact, word for word.
			{Text: "User runs CI on GitHub Actions", Why: "message 2"},
			// A genuinely new one.
			{Text: "User keeps the Aetox repository at D:\\Aetox\\Aetox", Why: "message 2"},
		},
	}
	n, err := app.runSessionReviewWith(context.Background(), reviewer, "s-1")
	if err != nil {
		t.Fatalf("runSessionReviewWith: %v", err)
	}
	if n != 1 {
		t.Errorf("proposed %d, want 1 — the restatement and the duplicate must not count", n)
	}
	if len(seen.Rejected) != 1 || len(seen.Pending) != 1 {
		t.Errorf("the reviewer was shown rejected=%v pending=%v; it must read both lists", seen.Rejected, seen.Pending)
	}
	digest := reviewDigest(seen)
	if !strings.Contains(digest, "Turned down by the user") || !strings.Contains(digest, "expects replies in Thai") {
		t.Errorf("the digest does not carry the refusal:\n%s", digest)
	}

	var pending int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pending_changes WHERE state = 'pending'`).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending != 2 {
		t.Errorf("%d rows waiting, want 2 — the old one and the new fact, never the restatement", pending)
	}
	var restated int
	_ = db.QueryRow(`SELECT COUNT(*) FROM pending_changes WHERE body LIKE 'User communicates primarily%'`).Scan(&restated)
	if restated != 0 {
		t.Error("a refused fact in other words reached the queue")
	}
}

// With nothing decided yet, the reviewer reads exactly the digest it read
// before the lists existed — the section headers appear only with content.
func TestReviewDigestIsUnchangedWhenNothingWasDecided(t *testing.T) {
	got := reviewDigest(reviewInput{Messages: []string{"สวัสดี", "ช่วยหน่อย"}})
	want := "=== User Messages ===\n\nUser message 1: สวัสดี\n\nUser message 2: ช่วยหน่อย\n\n"
	if got != want {
		t.Errorf("digest with nothing decided:\n%q\nwant\n%q", got, want)
	}
}
