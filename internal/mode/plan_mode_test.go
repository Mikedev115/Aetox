package mode

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// TheDialOnlyEverNarrows is the guarantee everything else here rests on.
//
// COMPANY.md §6.3 freezes a session's desk so a context can never hold another
// desk's tools, and §106 is allowed to move the stance underneath that for
// exactly one reason: a stance only subtracts. The moment something can widen
// one, that reasoning is gone — so this is pinned as a fact about the type
// rather than left to the one caller that happens to obey it today.
func TestTheDialOnlyEverNarrows(t *testing.T) {
	for _, tc := range []struct {
		from, to Stance
		allowed  bool
	}{
		{StanceAct, StancePlan, true},
		{StanceAct, StanceConsult, true},
		{StancePlan, StanceConsult, true},
		{StancePlan, StanceAct, false},
		{StanceConsult, StanceAct, false},
		{StanceConsult, StancePlan, false},
		{StanceAct, StanceAct, false},
		{StancePlan, StancePlan, false},
	} {
		d := NewDial(tc.from, nil)
		err := d.Narrow(tc.to)
		if (err == nil) != tc.allowed {
			t.Fatalf("%s → %s: allowed=%v, err=%v", stanceID(tc.from), stanceID(tc.to), tc.allowed, err)
		}
		// A refusal for going the wrong WAY is asked for by identity rather than
		// by reading its sentence, because it is the one the rest of §106 rests
		// on: "already there" is a wasted round, and a widening is the guarantee
		// breaking. A caller that has to tell them apart — and this test is one
		// — must not be doing it by matching text.
		if wrongWay := !tc.allowed && tc.from != tc.to; wrongWay != errors.Is(err, ErrWiden) {
			t.Fatalf("%s → %s: ErrWiden=%v, want %v (err=%v)",
				stanceID(tc.from), stanceID(tc.to), errors.Is(err, ErrWiden), wrongWay, err)
		}
		want := tc.to
		if !tc.allowed {
			want = tc.from
		}
		if got := d.Stance(); got != want {
			t.Fatalf("%s → %s: dial left on %q, want %q",
				stanceID(tc.from), stanceID(tc.to), stanceID(got), stanceID(want))
		}
	}
}

// TheDialAnswersFiltersLive is why the switch can land mid-turn at all.
//
// The dispatcher's filters run at request time and read the dial rather than a
// copy of a stance, so a tool withheld half way through a turn is refused for
// the rest of it — on the same dispatcher, with no engine rebuilt and no round
// interrupted. Snapshot the value into a closure and this stops being true
// while everything still compiles, which is the failure this pins.
func TestTheDialAnswersFiltersLive(t *testing.T) {
	d := NewDial(StanceAct, nil)
	if !d.Carries("write", skill.SourceBuiltin) {
		t.Fatal("ลงมือ should carry write")
	}
	if err := d.Narrow(StancePlan); err != nil {
		t.Fatalf("narrow: %v", err)
	}
	if d.Carries("write", skill.SourceBuiltin) {
		t.Fatal("วางแผน must not carry write once the dial has moved")
	}
	if !d.Carries("read", skill.SourceBuiltin) {
		t.Fatal("วางแผน keeps every tool that only looks")
	}
}

// TestTheHostHearsAboutTheSwitch: the engine narrows itself, and three things
// outside it hold the same fact — the composer's picker, the sessions row and
// conv.stance. They are told through this callback and by nothing else, so a
// silent switch would leave the window drawing ลงมือ over an engine that had
// stopped handing out `write`.
func TestTheHostHearsAboutTheSwitch(t *testing.T) {
	var heard []Stance
	d := NewDial(StanceAct, func(s Stance) { heard = append(heard, s) })
	if err := d.Narrow(StancePlan); err != nil {
		t.Fatalf("narrow: %v", err)
	}
	// A refused switch must not be announced: the host would move a picker onto
	// a stance nothing is enforcing.
	_ = d.Narrow(StanceAct)
	if len(heard) != 1 || heard[0] != StancePlan {
		t.Fatalf("host heard %v, want exactly one plan", heard)
	}
}

// TestPlanModeHandsBackTheDirection pins the mechanism that replaces the system
// prompt for a mid-turn switch.
//
// The prompt is assembled once at bootstrap and is not rebuilt when the model
// narrows itself, so วางแผน's direction cannot arrive the way it does when the
// user works the picker. It comes back as the tool's own result instead, which
// lands later in the context than the prompt and therefore outweighs it
// (§106.4). Hand back a receipt instead and the session filters like วางแผน
// while behaving like ลงมือ — the one failure that looks like nothing at all.
func TestPlanModeHandsBackTheDirection(t *testing.T) {
	d := NewDial(StanceAct, nil)
	tool := NewPlanModeTool(d)
	out, err := tool.ExecuteTool(context.Background(), map[string]any{"why": "งานใหญ่ ขอวางแผนก่อน"})
	if err != nil {
		t.Fatalf("plan_mode: %v", err)
	}
	if d.Stance() != StancePlan {
		t.Fatalf("dial left on %q", stanceID(d.Stance()))
	}
	if !strings.Contains(out.RawOutput, StancePlan.Direction()) {
		t.Fatal("the model must be handed วางแผน's direction, not just a receipt")
	}
	// Two readers, two texts. What the user sees on the mode button is the
	// sentence the model wrote about their request, not a paragraph of
	// instructions addressed to the model.
	if !strings.Contains(out.Content, "งานใหญ่ ขอวางแผนก่อน") {
		t.Fatalf("the user's line is missing from %q", out.Content)
	}
	if strings.Contains(out.Content, StancePlan.Direction()) {
		t.Fatal("the direction is for the model; the button gets the reason")
	}
}

// TestPlanModeIsOnlyOnTheDeskThatCanUseIt.
//
// Nothing in the filters names this tool: วางแผน answers from an allow-list
// that does not include it, and คู่คิด carries nothing at all. So the one stance
// with somewhere to go is the only one holding it, out of machinery that was
// already there. A line added to planKeeps would break that quietly — the model
// would get a switch into the mode it is already in, and the refusal would be
// the only thing standing between that and a wasted round.
func TestPlanModeIsOnlyOnTheDeskThatCanUseIt(t *testing.T) {
	if !StanceAct.Carries("plan_mode", skill.SourceBuiltin) {
		t.Fatal("ลงมือ is the stance this tool exists for")
	}
	if StancePlan.Carries("plan_mode", skill.SourceBuiltin) {
		t.Fatal("วางแผน has nowhere to go and must not carry it")
	}
	if StanceConsult.Carries("plan_mode", skill.SourceBuiltin) {
		t.Fatal("คู่คิด carries no tools at all")
	}
}
