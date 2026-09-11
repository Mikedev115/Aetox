package engine

import (
	"errors"
	"strings"
	"testing"
)

// Restarting kills the process, and the process is where the turn lives — so
// the refusal is the same one every session switch gets. The sentence is not:
// this one arrives on the update card, where advice about switching chats would
// read as the update itself having broken.
func TestReadyToRestartRefusesMidTurnInItsOwnWords(t *testing.T) {
	a := &Engine{}
	if err := a.ReadyToRestart(); err != nil {
		t.Fatalf("ReadyToRestart() = %v on an idle engine", err)
	}
	if err := a.beginTurn(a.cur().id); err != nil {
		t.Fatalf("beginTurn() = %v", err)
	}
	defer a.endTurn(a.cur().id)

	err := a.ReadyToRestart()
	if err == nil {
		t.Fatal("ReadyToRestart() = nil while a turn is running — the screen would kill the turn with the process")
	}
	if !errors.Is(err, errTurnBusyUpdate) {
		t.Errorf("err = %v, want the update-specific refusal", err)
	}
	if strings.Contains(err.Error(), "สลับแชท") {
		t.Error("the refusal points at a door the user is not standing in")
	}
}
