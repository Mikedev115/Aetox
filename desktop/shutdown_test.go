package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// Closing the window used to be the one way to end a turn that wrote nothing
// down: the goroutine died with the process, and the question openTurn had
// stored sat alone forever (§219). The close now presses Stop on every turn
// and holds the window until each has written its ending.
func TestClosingTheAppStopsEveryTurnAndWaitsForItsEnding(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	onScreen := a.cur()
	// A second chat, working off screen: CancelTurn would not reach it, and a
	// close that only stopped the chat on screen would let this one die mute.
	offScreen := &conversation{id: newSessionID(), cfg: onScreen.cfg}
	a.convs.show(offScreen)
	a.convs.show(onScreen)

	var ended []string
	for _, conv := range []*conversation{onScreen, offScreen} {
		conv := conv
		if err := a.beginTurn(conv.id); err != nil {
			t.Fatalf("beginTurn(%s): %v", conv.id, err)
		}
		a.openTurn(conv, SessionMessage{Role: "user", Text: "งานยาว", Time: "09:00"})
		ctx, cancel := context.WithCancel(context.Background())
		if a.armTurnCancel(conv.id, ctx, cancel) {
			t.Fatal("a Stop was pending before anybody pressed one")
		}
		// The turn as SendMessage runs it: works until its context ends, then
		// writes its ending and closes — a few ticks later, like a real one.
		go func() {
			<-ctx.Done()
			time.Sleep(30 * time.Millisecond)
			a.appendFailedTurn(conv, SessionMessage{Role: "agent", Time: "09:00"}, a.closeReason(ctx.Err()))
			a.turnMu.Lock()
			ended = append(ended, conv.id)
			a.turnMu.Unlock()
			a.endTurn(conv.id)
		}()
	}

	if a.beforeClose(context.Background()) {
		t.Fatal("beforeClose prevented the close — the user asked for it")
	}
	if a.turnBusy() {
		t.Fatal("beforeClose returned with a turn still running")
	}
	a.turnMu.Lock()
	wrote := len(ended)
	a.turnMu.Unlock()
	if wrote != 2 {
		t.Fatalf("%d turn(s) wrote their ending, want both", wrote)
	}
	// And what each wrote says the app was closed — not that Stop was pressed.
	for _, conv := range []*conversation{onScreen, offScreen} {
		messages, err := a.SessionTranscript(conv.id)
		if err != nil {
			t.Fatalf("SessionTranscript: %v", err)
		}
		if len(messages) != 2 || messages[1].Role != "agent" {
			t.Fatalf("%s: stored %d messages; want the question and its ending", conv.id, len(messages))
		}
		last := messages[1]
		if !strings.Contains(last.ErrorText, interruptedTurnMarker) {
			t.Errorf("%s: error_text = %q, want the close marker", conv.id, last.ErrorText)
		}
		if !strings.HasSuffix(last.ErrorText, context.Canceled.Error()) {
			t.Errorf("%s: error_text = %q, want the cancel kept behind the marker so a Stop reader still recognises it", conv.id, last.ErrorText)
		}
	}
}

// A tool that ignores its context cannot hold the window hostage: the close
// waits for the grace and then goes, leaving the question for the next launch.
func TestClosingTheAppGivesUpOnATurnThatWillNotEnd(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	if err := a.beginTurn(a.cur().id); err != nil {
		t.Fatalf("beginTurn: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.armTurnCancel(a.cur().id, ctx, cancel)

	started := time.Now()
	if a.finishTurnsForClose(80 * time.Millisecond) {
		t.Fatal("reported every turn ended while one never did")
	}
	if waited := time.Since(started); waited < 80*time.Millisecond || waited > 2*time.Second {
		t.Errorf("waited %s, want about the grace", waited)
	}
	if ctx.Err() == nil {
		t.Error("the turn was never cancelled")
	}
}

// A Stop the user pressed before the close is still a Stop. The marker goes on
// a cancel only once the app is closing, and never on any other error.
func TestTheCloseMarkerGoesOnlyOnACancelDuringClose(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	before := a.closeReason(context.Canceled)
	if strings.Contains(before.Error(), interruptedTurnMarker) {
		t.Errorf("before closing: %q carries the marker — a Stop would read as a close", before)
	}
	a.closing.Store(true)
	if err := a.closeReason(nil); err != nil {
		t.Errorf("nil in, %v out", err)
	}
	quota := errors.New("codex: the free plan's limit is used up")
	if got := a.closeReason(quota); got != quota {
		t.Errorf("a quota error was rewritten to %q", got)
	}
	flattened := errors.New(`Post "https://provider/chat": context canceled`)
	got := a.closeReason(flattened)
	if !strings.HasPrefix(got.Error(), interruptedTurnMarker) || !errors.Is(got, flattened) {
		t.Errorf("flattened cancel became %q; want the marker in front and the original kept", got)
	}
	// Twice through does not stack markers.
	if again := a.closeReason(got); again != got {
		t.Errorf("second pass rewrote %q to %q", got, again)
	}
}

// The other road: the process ended with no time to write anything. The next
// launch closes every question left waiting, and only those.
func TestTheNextLaunchClosesQuestionsTheLastRunLeftWaiting(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	orphaned := a.cur()
	if !a.openTurn(orphaned, SessionMessage{Role: "user", Text: "ทำต่อให้ที", Time: "23:59"}) {
		t.Fatal("openTurn refused")
	}
	// A conversation that ended properly must not be touched.
	finished := &conversation{id: newSessionID(), cfg: orphaned.cfg}
	a.convs.show(finished)
	a.openTurn(finished, SessionMessage{Role: "user", Text: "สวัสดี", Time: "23:00"})
	a.appendTurn(finished, SessionMessage{Role: "user", Text: "สวัสดี", Time: "23:00"},
		SessionMessage{Role: "agent", Text: "สวัสดีครับ", Time: "23:00"})
	a.convs.show(orphaned)
	orphaned.turnOpened = false // the process that raised it is gone

	if closed := a.closeInterruptedTurns(); closed != 1 {
		t.Fatalf("closed %d turn(s), want exactly the orphaned one", closed)
	}

	messages, err := a.SessionTranscript(orphaned.id)
	if err != nil {
		t.Fatalf("SessionTranscript: %v", err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "agent" {
		t.Fatalf("stored %d messages; want the question and its ending", len(messages))
	}
	ending := messages[1]
	if ending.ErrorText != interruptedTurnMarker {
		t.Errorf("error_text = %q, want %q", ending.ErrorText, interruptedTurnMarker)
	}
	if ending.Text != "" {
		t.Errorf("text = %q, want nothing — the process wrote none", ending.Text)
	}
	if ending.Time != "23:59" {
		t.Errorf("time = %q, want the question's own, not the launch's", ending.Time)
	}

	untouched, _ := a.SessionTranscript(finished.id)
	if len(untouched) != 2 {
		t.Errorf("the finished chat has %d messages, want 2 — it was not waiting for anything", len(untouched))
	}
	// Second launch: nothing left to close.
	if closed := a.closeInterruptedTurns(); closed != 0 {
		t.Errorf("a second pass closed %d turn(s); the first left nothing waiting", closed)
	}
}

// A question with no answer yet is legitimate while a turn is answering it —
// a reload of the window mid-turn must never see it closed underneath.
func TestTheSweepRefusesToRunUnderALiveTurn(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	if err := a.beginTurn(a.cur().id); err != nil {
		t.Fatalf("beginTurn: %v", err)
	}
	a.openTurn(a.cur(), SessionMessage{Role: "user", Text: "กำลังตอบอยู่", Time: "10:00"})
	if closed := a.closeInterruptedTurns(); closed != 0 {
		t.Fatalf("closed %d turn(s) under a live turn", closed)
	}
	a.endTurn(a.cur().id)
}
