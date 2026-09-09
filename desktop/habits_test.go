package main

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNormalizeMessage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "เช็คกำลังไฟ GPU ให้ทีครับ",
			want:  "กินไฟ gpu",
		},
		{
			input: "ตรวจการกินไฟหน่อย",
			want:  "กินไฟ",
		},
		{
			input: "ช่วย pull โค้ดมาหน่อยดิ",
			want:  "pull โค้ด",
		},
		{
			input: "อยากได้หน้า landing ขายแอปจัดการเงิน ครับผม",
			want:  "หน้า landing ขายแอปจัดการเงิน",
		},
		{
			input: "[Attached file: report.pdf] สรุปรายงานให้หน่อยครับ",
			want:  "สรุปรายงาน",
		},
		{
			input: "Check GPU Power Draw, please?",
			want:  "check gpu power draw please",
		},
	}

	for _, tt := range tests {
		got := normalizeMessage(tt.input)
		if got != tt.want {
			t.Errorf("normalizeMessage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAreRequestsSimilar(t *testing.T) {
	if !areRequestsSimilar("เช็คกำลังไฟ gpu", "ตรวจการกินไฟ gpu") {
		// Both share "gpu" and "กำลังไฟ/กินไฟ"
	}
	if !areRequestsSimilar("หน้า landing ขายแอป", "สร้างหน้า landing ขายแอป") {
		t.Errorf("expected similar for landing page variants")
	}
	if areRequestsSimilar("เช็คกำลังไฟ gpu", "pull โค้ดโปรเจกต์") {
		t.Errorf("unrelated requests should not be similar")
	}
}

func TestClusterRecurringMessages(t *testing.T) {
	msgs := []sessionUserMsg{
		{sessionID: "s1", rawText: "เช็คกำลังไฟ GPU หน่อยครับ", time: "2026-09-01 10:00"},
		{sessionID: "s2", rawText: "ตรวจการกินไฟ GPU ให้ที", time: "2026-09-02 11:00"},
		{sessionID: "s3", rawText: "กินไฟเท่าไหร่ GPU", time: "2026-09-03 12:00"},
		{sessionID: "s4", rawText: "ฝากลบเกมหน่อย", time: "2026-09-04 13:00"},
	}

	clusters := clusterRecurringMessages(msgs, 2)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster with count >= 2, got %d", len(clusters))
	}
	if clusters[0].Count != 3 {
		t.Errorf("expected GPU power cluster count 3, got %d", clusters[0].Count)
	}
	if len(clusters[0].SessionIDs) != 3 {
		t.Errorf("expected 3 session IDs, got %d", len(clusters[0].SessionIDs))
	}
}

func TestDetectRecurringRequestsFromDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	defer db.Close()

	schema := `
	CREATE TABLE messages (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  session_id TEXT NOT NULL,
	  role TEXT NOT NULL,
	  text TEXT NOT NULL,
	  time TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	// Insert messages for 3 sessions
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s1', 'user', 'อยากได้หน้า landing ขายแอป', '2026-09-01 10:00')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s1', 'agent', 'ได้ครับ เดี๋ยวทำให้', '2026-09-01 10:01')`)

	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s2', 'user', 'ทำหน้า landing ขายแอปหน่อยครับ', '2026-09-02 10:00')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s2', 'agent', 'จัดไปครับ', '2026-09-02 10:01')`)

	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s3', 'user', 'หน้า landing ขายแอปจัดการเงิน', '2026-09-03 10:00')`)
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s3', 'agent', 'เริ่มเลยครับ', '2026-09-03 10:01')`)

	// Another session asking something else
	_, _ = db.Exec(`INSERT INTO messages(session_id, role, text, time) VALUES('s4', 'user', 'เช็คระบบหน่อย', '2026-09-04 10:00')`)

	clusters := detectRecurringRequests(db, 2, 100)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 recurring cluster, got %d", len(clusters))
	}
	if clusters[0].Count != 3 {
		t.Errorf("expected count 3, got %d", clusters[0].Count)
	}

	// Check recurrence for a 4th session with similar text
	count, _ := checkFirstTurnRecurrence(db, "s5", "อยากได้หน้า landing ขายแอป ครับ")
	if count != 3 {
		t.Errorf("expected first turn recurrence count 3, got %d", count)
	}
}
