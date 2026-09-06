package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseUsesOverrideDir(t *testing.T) {
	a := seed(&App{dbDir: t.TempDir()}, newConversation())
	closeDBOnCleanup(t, a)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database(): unexpected error: %v", err)
	}
	if db == nil {
		t.Fatal("database() returned nil *sql.DB with no error")
	}
	// Schema must have applied (sessions table exists) against the override dir.
	if _, err := db.Exec("SELECT 1 FROM sessions LIMIT 1"); err != nil {
		t.Errorf("sessions table not created in override dir: %v", err)
	}
}

func TestDatabaseSingleton(t *testing.T) {
	a := seed(&App{dbDir: t.TempDir()}, newConversation())
	closeDBOnCleanup(t, a)
	db1, err := a.database()
	if err != nil {
		t.Fatalf("database(): unexpected error: %v", err)
	}
	db2, _ := a.database()
	if db1 != db2 {
		t.Error("database() returned a different *sql.DB on second call, want the same singleton")
	}
}

// The 7 ก.ย. 2026 failure, from the outside: an installed build meets a file a
// newer one already migrated. The history it cannot read must not come back
// looking like a history that is empty — that is the whole difference between
// "อัปเดตแอป" and "ข้อมูลหายหมดแล้ว", and for as long as the list existed the
// user was shown the second one.
func TestStoreWrittenByANewerBuildIsReportedNotSilentlyEmpty(t *testing.T) {
	dir := t.TempDir()
	ahead := latestSchemaVersion() + 1
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(dir, "aetox.db")))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", ahead)); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	a := seed(&App{dbDir: dir}, newConversation())
	if _, err := a.database(); err == nil {
		t.Fatal("database() opened a store from a newer build, want it refused")
	}

	fault := a.HistoryFault()
	if !fault.Failed || !fault.TooNew {
		t.Fatalf("HistoryFault() = %+v, want Failed and TooNew both set", fault)
	}
	if fault.Have != ahead || fault.Known != latestSchemaVersion() {
		t.Errorf("HistoryFault() reported schema %d over %d, want %d over %d",
			fault.Have, fault.Known, ahead, latestSchemaVersion())
	}
	// The pairing the window depends on: the list is empty AND the fault says
	// why. An empty list on its own is the bug, not the report.
	if got := a.ListSessionsForDoor(DeskFilter{}); len(got) != 0 {
		t.Errorf("ListSessionsForDoor returned %d rows over an unopenable store", len(got))
	}
	if !a.dbFatal {
		t.Error("a newer schema was not marked fatal, so every list will re-attempt an open that cannot succeed")
	}
}

// A store that was busy for a moment — another instance still writing its way
// out of a shutdown — must not cost the window its history until it restarts.
// sync.Once made the first error the answer forever; this is the retry that
// replaced it.
func TestTransientOpenFailureIsRetriedRatherThanKeptForever(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	// A file where the data directory belongs: MkdirAll fails, the way a
	// locked or unwritable disk does, and nothing about it is permanent.
	if err := os.WriteFile(dir, []byte("in the way"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	a := seed(&App{dbDir: dir}, newConversation())
	closeDBOnCleanup(t, a)
	if _, err := a.database(); err == nil {
		t.Fatal("database() succeeded with a file where its directory belongs")
	}
	if a.dbFatal {
		t.Fatal("a blocked directory was marked fatal, want it retried once the disk lets go")
	}
	if fault := a.HistoryFault(); !fault.Failed || fault.TooNew {
		t.Errorf("HistoryFault() = %+v, want Failed without TooNew", fault)
	}

	if err := os.Remove(dir); err != nil {
		t.Fatalf("remove blocker: %v", err)
	}
	// The throttle is about not stalling the UI, not about the clock — this
	// test is asking whether the failure was kept, so it stands the throttle
	// down rather than sleeping through it.
	a.dbRetryAt = time.Time{}
	if _, err := a.database(); err != nil {
		t.Fatalf("database() after the obstruction cleared: %v", err)
	}
	if fault := a.HistoryFault(); fault.Failed {
		t.Errorf("HistoryFault() = %+v after the store opened, want no fault", fault)
	}
}

// The throttle itself: a store that cannot open must not make every list in the
// window wait out another open attempt.
func TestFailedOpenIsNotReattemptedOnEveryCall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	if err := os.WriteFile(dir, []byte("in the way"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	a := seed(&App{dbDir: dir}, newConversation())
	if _, err := a.database(); err == nil {
		t.Fatal("database() succeeded with a file where its directory belongs")
	}
	if err := os.Remove(dir); err != nil {
		t.Fatalf("remove blocker: %v", err)
	}
	if _, err := a.database(); err == nil {
		t.Fatal("database() re-opened inside the retry window, want the cached failure until it expires")
	}
}
