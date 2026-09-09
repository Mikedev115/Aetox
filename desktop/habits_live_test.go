package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestLiveUserDatabaseHabits(t *testing.T) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		t.Skip("APPDATA not set")
	}
	dbPath := filepath.Join(appData, "aetox", "aetox.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Skipf("user db does not exist: %v", err)
	}

	// Open in read-only mode to guarantee zero mutations to the user's store
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath)+"?mode=ro")
	if err != nil {
		t.Fatalf("failed to open user db: %v", err)
	}
	defer db.Close()

	habits := detectRecurringRequests(db, 2, 200)
	t.Logf("Found %d recurring habit clusters in real user database:", len(habits))
	for i, h := range habits {
		t.Logf("  [%d] Count: %d | Last: %s | Text: %q | Normalized: %q",
			i+1, h.Count, h.LastAskedAt, h.Text, h.Normalized)
	}
}
