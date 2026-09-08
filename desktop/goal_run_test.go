package main

import (
	"context"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/mode"
)

func goalApp(t *testing.T) *planSkill {
	t.Helper()
	app := &App{}
	isolateUserDirs(t)
	closeDBOnCleanup(t, app)
	app.cur().id = newSessionID()
	return &planSkill{app: app, conv: app.cur()}
}

func writeRunnablePlan(t *testing.T, s *planSkill) {
	t.Helper()
	call(t, s, map[string]any{
		"action": "write",
		"title":  "ยกเพดานการรอ",
		"sections": []any{
			section("What to change", "see the steps"),
			section("How you will know it worked", "`go test ./desktop/` passes and the busy bar counts"),
		},
		"steps": []any{"raise waitMax", "wire toolCallDeadline", "report progress"},
	})
}

// GATE ONE, and the reason this mode is not the one §106.10 declined: the check
// that a run is finished is not a judgement about prose. A model that says "all
// done" with a step unmarked is sent back naming the step.
func TestARunIsNotFinishedWhileAStepIsUnmarked(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.goalRunSet().start(id)

	check := s.app.goalCheck(id)
	verdict := check("All three steps are complete.")
	if verdict == "" {
		t.Fatal("the turn was allowed to end with every step unmarked — the model's word was taken for it")
	}
	for _, want := range []string{"raise waitMax", "wire toolCallDeadline", "report progress"} {
		if !strings.Contains(verdict, want) {
			t.Errorf("the verdict does not name the open step %q:\n%s", want, verdict)
		}
	}
}

// A step marked through the tool is settled, and only the ones left are named.
// The verdict is a message that is re-sent with every later round of the turn,
// so naming settled work would be paid for repeatedly for nothing.
func TestTheVerdictNamesOnlyWhatIsLeft(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.goalRunSet().start(id)

	call(t, s, map[string]any{"action": "step", "n": 1, "state": "done"})
	call(t, s, map[string]any{"action": "step", "n": 2, "state": "done"})

	verdict := s.app.goalCheck(id)("done now")
	if strings.Contains(verdict, "raise waitMax") {
		t.Errorf("the verdict names a step that is already done:\n%s", verdict)
	}
	if !strings.Contains(verdict, "report progress") {
		t.Errorf("the verdict does not name the one step still open:\n%s", verdict)
	}
}

// `failed` settles a step. A step the model looked at and reported impossible is
// a finding the user needs, and sending the turn back over it forever would be
// the run refusing to accept an answer.
func TestAFailedStepSettlesRatherThanLooping(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.goalRunSet().start(id)

	call(t, s, map[string]any{"action": "step", "n": 1, "state": "done"})
	call(t, s, map[string]any{"action": "step", "n": 2, "state": "failed", "note": "the deadline is not reachable from here"})
	call(t, s, map[string]any{"action": "step", "n": 3, "state": "done"})

	// Gate one is satisfied, so what comes back is gate two — the plan's own
	// finish condition — and not another list of steps.
	verdict := s.app.goalCheck(id)("two done, one impossible")
	if strings.Contains(verdict, "not marked done") {
		t.Errorf("a failed step kept the run open:\n%s", verdict)
	}
	if !strings.Contains(verdict, "go test ./desktop/") {
		t.Errorf("gate two did not hand back the plan's finish condition:\n%s", verdict)
	}
}

// GATE TWO fires once and then never again. A gate that can fire repeatedly on
// prose is a gate that argues, and the ceiling — not this — is what handles a
// run that genuinely cannot finish.
func TestTheFinishConditionIsAskedOnce(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.goalRunSet().start(id)
	for n := 1; n <= 3; n++ {
		call(t, s, map[string]any{"action": "step", "n": n, "state": "done"})
	}

	check := s.app.goalCheck(id)
	if first := check("finished"); !strings.Contains(first, "go test ./desktop/") {
		t.Fatalf("the finish condition was not asked at all:\n%s", first)
	}
	if second := check("I ran it, it passes"); second != "" {
		t.Errorf("the finish condition was asked twice, so the run argues:\n%s", second)
	}
	// And the run is put down with the turn, so the card stops saying it is
	// working without the window having to ask.
	if s.app.goalRunSet().running(id) {
		t.Error("the run is still marked running after the turn was allowed to end")
	}
}

// A plan with no checkable steps is refused, and the refusal is the honest one:
// มุ่งเป้า with no written steps is "try harder for longer", which is exactly what
// §106.10 declined.
func TestARunRefusesAPlanWithNothingToCheck(t *testing.T) {
	s := goalApp(t)
	call(t, s, map[string]any{"action": "write", "sections": []any{
		section("What to change", "improve it somehow"),
	}})
	if got := s.app.StartPlanRun(s.sessionID()); got.Refusal == "" {
		t.Fatal("a plan with no steps was accepted as something to carry out")
	}
	if s.app.goalRunSet().running(s.sessionID()) {
		t.Error("a refused start left a run behind")
	}
}

func TestARunRefusesAConversationWithNoPlan(t *testing.T) {
	s := goalApp(t)
	if got := s.app.StartPlanRun(s.sessionID()); got.Refusal == "" {
		t.Fatal("a conversation with no plan was accepted as something to carry out")
	}
}

// Stopping the run from the window lets the turn END rather than cutting it off:
// the user pressed stop on the RUN, and the Stop button is what does the other
// thing. Cutting the answer here would lose work already done.
func TestStoppingTheRunLetsTheTurnFinish(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.goalRunSet().start(id)

	check := s.app.goalCheck(id)
	s.app.goalRunSet().stop(id)
	if verdict := check("stopping here"); verdict != "" {
		t.Errorf("a stopped run still sent the turn back:\n%s", verdict)
	}
}

// An amend that does not mention the steps must not wipe the marks a run has
// already made — the same mistake `amend` exists to stop it making with the
// sections, and a worse one: it would lose the record of work that happened.
func TestAmendingTheProseKeepsTheStepsAndTheirMarks(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	call(t, s, map[string]any{"action": "step", "n": 1, "state": "done"})

	call(t, s, map[string]any{"action": "amend", "sections": []any{
		section("What could go wrong", "a turn held open for ten minutes"),
	}})

	plan, _ := s.app.loadPlan(s.sessionID())
	if len(plan.Steps) != 3 {
		t.Fatalf("the amend left %d steps", len(plan.Steps))
	}
	if plan.Steps[0].State != planStepDone {
		t.Errorf("step 1 lost its mark, so a run would redo work that was already done")
	}
}

// The numbering belongs to this side. A model that numbered its own steps could
// renumber them in the next amend, and a run would then be marking something
// other than what it carried out.
func TestStepsAreNumberedByTheTool(t *testing.T) {
	s := goalApp(t)
	call(t, s, map[string]any{"action": "write", "sections": []any{section("What to change", "x")},
		"steps": []any{"first", "second"}})
	plan, _ := s.app.loadPlan(s.sessionID())
	if len(plan.Steps) != 2 || plan.Steps[0].N != 1 || plan.Steps[1].N != 2 {
		t.Fatalf("steps are numbered %+v", plan.Steps)
	}
	out, err := s.ExecuteTool(context.Background(), map[string]any{"action": "step", "n": 9, "state": "done"})
	if err == nil {
		t.Fatal("a step that does not exist was accepted")
	}
	if !strings.Contains(out.Content, "no step 9") {
		t.Errorf("the refusal does not say which step was missing: %q", out.Content)
	}
}

// `read` carries the marks, because it is what a later turn — or ลงมือ, or a
// later run — uses to find out where the work got to.
func TestReadingThePlanShowsWhatHasBeenDone(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	call(t, s, map[string]any{"action": "step", "n": 1, "state": "done"})
	call(t, s, map[string]any{"action": "step", "n": 2, "state": "failed", "note": "not reachable"})

	out := call(t, s, map[string]any{"action": "read"})
	if !strings.Contains(out, "[x] 1.") {
		t.Errorf("a finished step is not marked in the plan that comes back:\n%s", out)
	}
	if !strings.Contains(out, "[!] 2.") || !strings.Contains(out, "not reachable") {
		t.Errorf("a failed step lost its mark or its reason:\n%s", out)
	}
	if !strings.Contains(out, "[ ] 3.") {
		t.Errorf("an open step is not shown as open:\n%s", out)
	}
}

// THE PRESS CROSSES FROM PLANNING TO ACTING.
//
// Without it the button was a promise the session could not keep: วางแผน
// carries no write, no edit, no shell, and deliberately no `plan_step` — so a
// run started there could not touch anything, could not mark anything, and
// would be sent back by its own first gate until it hit the ceiling. The owner
// found it on the first real run, which is the useful kind of finding and the
// expensive kind to have shipped.
func TestStartingARunCrossesIntoActing(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	s.app.cur().stance = mode.StancePlan

	if got := s.app.StartPlanRun(s.sessionID()); got.Refusal != "" {
		t.Fatalf("the run was refused: %s", got.Refusal)
	}
	if got := s.app.cur().stance; got != mode.StanceAct {
		t.Fatalf("the session is still in %q — the run cannot write, run anything, or even mark a step", got)
	}
	// The tool it needs most is the one วางแผน is right to withhold: marking a
	// step done asserts that work happened.
	if !mode.StanceAct.AllowsAction("plan", "step") {
		t.Error("plan_step is not reachable in the stance the run just switched to")
	}
	if mode.StancePlan.AllowsAction("plan", "step") {
		t.Error("plan_step is reachable in วางแผน — a stance that changes nothing must not be able to say work happened")
	}
}

// A refused switch is a refused run. Starting one against a session mid-turn
// would aim it at an engine that is about to be replaced.
func TestARunIsNotStartedWhenTheStanceWillNotMove(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	s.app.cur().stance = mode.StancePlan
	// The busy state as the app records it, reached directly because this is
	// about the guard rather than about how a turn gets started.
	s.app.turnMu.Lock()
	if s.app.turns == nil {
		s.app.turns = map[string]*liveTurn{}
	}
	s.app.turns[s.sessionID()] = &liveTurn{}
	s.app.turnMu.Unlock()

	if got := s.app.StartPlanRun(s.sessionID()); got.Refusal == "" {
		t.Fatal("a run started while a turn was already running in the same chat")
	}
	if s.app.goalRunSet().running(s.sessionID()) {
		t.Error("a refused start left a run behind")
	}
}

// PAUSING IS NOT STOPPING, and the difference is where it takes effect: the
// check lets go, so the turn ends where it naturally would with everything it
// did already marked. Nothing is thrown away, which is the whole promise of a
// button called พัก.
func TestPausingLetsTheTurnEndWithoutLosingTheRun(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.goalRunSet().start(id)

	// One step done, two open: without the hold this would be sent back.
	call(t, s, map[string]any{"action": "step", "n": 1, "state": "done"})
	s.app.PausePlanRun(id)

	if v := s.app.goalCheck(id)("stopping here for now"); v != "" {
		t.Fatalf("a paused run still pushed the turn on:\n%s", v)
	}
	// The run is STILL THERE. A pause that ended the run would be a stop with a
	// gentler label.
	if !s.app.goalRunSet().running(id) {
		t.Fatal("pausing ended the run")
	}
	plan, _ := s.app.loadPlan(id)
	if plan.Steps[0].State != planStepDone {
		t.Error("work done before the pause was lost")
	}
}

func TestResumingPicksTheRunBackUp(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.goalRunSet().start(id)
	s.app.PausePlanRun(id)
	s.app.ResumePlanRun(id)

	if v := s.app.goalCheck(id)("done?"); v == "" {
		t.Fatal("a resumed run let the turn end — the hold was never lifted")
	}
}

// A BREAKPOINT IS A PAUSE YOU CAN SET IN ADVANCE. The pause button only works
// for somebody sitting in front of the screen, and this mode exists to be left
// alone; this is the control for "stop before you touch that one".
func TestABreakpointHoldsTheRunBeforeThatStep(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.SetPlanStepStop(id, 2, true)
	s.app.goalRunSet().start(id)
	call(t, s, map[string]any{"action": "step", "n": 1, "state": "done"})

	// Captured off the EVENT rather than off the row: `Paused` is true of a run
	// and not of a plan, so it is filled in on the way to the screen and stored
	// nowhere — a hold that survived a restart would be a card offering to
	// resume something that is not happening.
	var drawn Plan
	s.app.emit = func(event string, data ...any) {
		if ev, ok := data[0].(sessionEvent[Plan]); ok && event == "plan:update" {
			drawn = ev.Data
		}
	}
	if v := s.app.goalCheck(id)("carrying on"); v != "" {
		t.Fatalf("the run walked through a breakpoint:\n%s", v)
	}
	if drawn.Paused == "" {
		t.Error("the card cannot say why it stopped")
	}
	if !strings.Contains(drawn.Paused, "2") {
		t.Errorf("the card does not name the step it is waiting on: %q", drawn.Paused)
	}
	if !s.app.goalRunSet().running(id) {
		t.Error("a breakpoint ended the run rather than holding it")
	}
}

// Only the NEXT open step is looked at. A breakpoint further down the list is
// not this moment's business, and firing early would stop the run before the
// work the user was happy to have done unattended.
func TestABreakpointFurtherDownDoesNotFireYet(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.SetPlanStepStop(id, 3, true)
	s.app.goalRunSet().start(id)

	// Nothing done at all: the next open step is 1, which carries no breakpoint.
	if v := s.app.goalCheck(id)("done?"); v == "" {
		t.Fatal("a breakpoint on step 3 stopped the run before step 1")
	}
}

// ไปต่อ past a breakpoint clears it. Stopping again at the same step would read
// as the button not working — a breakpoint asks to be consulted once, it is not
// a wall.
func TestResumingPastABreakpointClearsIt(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()
	s.app.SetPlanStepStop(id, 1, true)
	s.app.goalRunSet().start(id)

	if v := s.app.goalCheck(id)("starting"); v != "" {
		t.Fatal("the breakpoint did not hold")
	}
	s.app.ResumePlanRun(id)
	plan, _ := s.app.loadPlan(id)
	if plan.Steps[0].Stop {
		t.Error("the breakpoint survived ไปต่อ, so the run stops here forever")
	}
	if v := s.app.goalCheck(id)("carrying on"); v == "" {
		t.Fatal("the run did not carry on after ไปต่อ")
	}
}

// A breakpoint belongs to the plan, not to the run: it should be there tomorrow
// and settable before anything starts, which is when somebody reading a fresh
// plan actually notices the step they want to see.
func TestABreakpointIsStoredWithThePlan(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	s.app.SetPlanStepStop(s.sessionID(), 2, true)

	plan, _ := s.app.loadPlan(s.sessionID())
	if !plan.Steps[1].Stop {
		t.Fatal("the breakpoint was not stored")
	}
	s.app.SetPlanStepStop(s.sessionID(), 2, false)
	plan, _ = s.app.loadPlan(s.sessionID())
	if plan.Steps[1].Stop {
		t.Error("the breakpoint could not be taken off")
	}
}

// PRESSING THE BUTTON HAS TO SEND SOMETHING.
//
// Starting a run switches the stance, registers the run and installs the goal
// check — and none of that makes anything happen, because the check only fires
// when a turn is about to END and there is no turn. The first real press left
// the bar at 0/5 with the light on, waiting for the user to type. A button that
// says "carry this out" and only arms the machinery is a button that lies.
//
// What comes back is a FLAG, not a sentence. The first version had this build a
// paragraph of Thai instruction in Go and hand it over to be sent as a user
// turn, which the owner refused on sight: screen text past the locale files, a
// prompt wearing a user turn, and re-sent with every later round of the run —
// in the one mode built around the fact that exactly that cost is quadratic.
func TestStartingARunTellsTheWindowToSend(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	got := s.app.StartPlanRun(s.sessionID())
	if got.Refusal != "" {
		t.Fatalf("refused: %s", got.Refusal)
	}
	if !got.Started {
		t.Fatal("nothing told the window to send, so the run would sit there doing nothing")
	}
}

// The instructions live where this repo puts instructions: guidance, delivered
// once with the first result and never again — not in a message that is re-sent
// on every round.
func TestTheStepInstructionsAreGuidanceRatherThanAMessage(t *testing.T) {
	s := planApp(t)
	g := s.Guidance(map[string]any{"action": "step"})
	if !strings.Contains(g, "the moment it is finished") {
		t.Errorf("nothing tells it to mark as it goes: %s", g)
	}
	if !strings.Contains(g, "failed") {
		t.Errorf("nothing tells it that failed is a real answer: %s", g)
	}
	if !strings.Contains(g, "narrate") {
		t.Errorf("nothing tells it to stop describing progress: %s", g)
	}
	// And the write half says where the checklist goes, which is the mistake
	// that made the first real plan of the day inert.
	if w := s.Guidance(map[string]any{"action": "write"}); !strings.Contains(w, "`steps`") {
		t.Errorf("the write guidance does not say where the checklist goes: %s", w)
	}
	// Nothing to say once about an action whose signature says it all gets "".
	if s.Guidance(map[string]any{"action": "sing"}) != "" {
		t.Error("guidance is returned for an action that does not exist")
	}
}
