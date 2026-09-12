package main

import "testing"

// SetCompanionState is the window's report and the body's truth: each report
// is kept whole, numbered, and handed to the body on the desktop if there is
// one.
func TestCompanionStateIsKeptNumberedAndHandedToTheBody(t *testing.T) {
	a := &App{}
	body := &recordingBody{}
	a.companion().body = body
	a.SetCompanionState(CompanionState{Pose: "thinking", Shown: "สวัสดี", Words: []string{"สวัสดี"}, Cursor: true, Hop: 2})
	a.SetCompanionState(CompanionState{Pose: "idle"})
	if len(body.got) != 2 {
		t.Fatalf("body was told %d times, want 2", len(body.got))
	}
	first, second := body.got[0], body.got[1]
	if first.Seq != 1 || second.Seq != 2 {
		t.Fatalf("seq %d, %d — want 1, 2", first.Seq, second.Seq)
	}
	if first.Pose != "thinking" || first.Shown != "สวัสดี" || !first.Cursor || first.Hop != 2 || first.At.IsZero() {
		t.Fatalf("first report not kept whole: %+v", first)
	}
	if got := a.companion().state; got.Seq != 2 || got.Pose != "idle" {
		t.Fatalf("latest state: %+v", got)
	}
}

// Without a body the report is simply kept, for the body that opens later.
func TestCompanionStateWaitsForABody(t *testing.T) {
	a := &App{}
	a.SetCompanionState(CompanionState{Pose: "coding"})
	if got := a.companion().state; got.Pose != "coding" || got.Seq != 1 {
		t.Fatalf("state without a body: %+v", got)
	}
}

type recordingBody struct{ got []CompanionState }

func (r *recordingBody) apply(s CompanionState) { r.got = append(r.got, s) }
func (r *recordingBody) close()                 {}
