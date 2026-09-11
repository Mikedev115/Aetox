package main

import "testing"

// closeStoreAtEnd closes the app's store when the test ends.
//
// Needed by every test that builds an engine over a TempDir and never opens
// the store itself: since 11 ก.ย. the prompt reads the proposal ledger while
// it is built (appProposer.Ledger → prompt.Desk.Ledger), which opens aetox.db,
// and a handle left open makes Windows refuse the TempDir's removal at
// cleanup. The tests that already opened the store close it the same way.
func closeStoreAtEnd(t *testing.T, a *App) {
	t.Helper()
	t.Cleanup(func() {
		a.dbMu.Lock()
		defer a.dbMu.Unlock()
		if a.db != nil {
			_ = a.db.Close()
			a.db = nil
		}
	})
}
