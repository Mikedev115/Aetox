package main

// มุ่งเป้า — carrying out a plan until it is actually done.
//
// §106.10 declined this mode on 2026-08-14, and the owner reopened it on
// 2026-09-08. It is not the build that was declined, and the difference is worth
// stating before any of the code below makes sense.
//
// ## What that decision named as non-optional, and where each part lives now
//
//   - **A goal pinned so a long run cannot drift off it** — the stored plan
//     (plan.go). The user pins it by pressing "carry out this plan" ON the plan,
//     so nothing has to be re-typed and nothing can be re-interpreted.
//   - **A finish condition that can be CHECKED rather than felt** — the plan's
//     own "How you will know it worked" heading, added the same day for this.
//   - **A checker that is not the model that just declared victory** — below.
//   - **A hard ceiling on steps and tokens** — `MaxToolCalls`, which already
//     bounded the loop and bounds it whatever keeps it alive.
//
// ## The checker is mechanical first, and that is the whole trick
//
// The obvious build is a second model call: ask a reviewer whether the work is
// done. That is what §106.10 costed at "a verification call every round" and
// declined over, and it has a worse problem than price — a model asked whether
// another model finished is still a model being asked to judge prose.
//
// The first gate here is not a judgement at all. **Every step of the plan has a
// state in the database, and a run is not finished while a step is unmarked.**
// A model that says "all done" with step 3 untouched is sent back naming step 3.
// It cannot talk its way past that, and the only way to fake it is to call
// `plan step` on work it did not do — a deliberate false statement in a tool
// call, which is a far higher bar than a confident closing sentence.
//
// The second gate is the finish condition, which is prose and does need
// judgement. It is handed back to the model ONCE per run, as the plan's own
// words: you said this is how we would know it worked — say how you verified it.
// A criterion the model wrote in an earlier turn and had approved is not the
// same thing as a model marking its own homework in this one.
//
// Once, and then never again, because a gate that can fire repeatedly on prose
// is a gate that argues. The ceiling is what handles a run that genuinely cannot
// finish, and it is the one part that must not be left to judgement.
//
// ## What it does not do
//
// It does not narrate. The owner's constraint while this was being built:
// *"ทำให้มันรายงานให้น้อย… พูดแค่ส่วนสำคัญน้อยสุดยิ่งดี"*, and he is right for a
// sharper reason than output tokens — narration written alongside a round's tool
// calls goes into the conversation and is re-sent with every later round of the
// same turn, so the cost of narrating is quadratic in the length of the run.
// มุ่งเป้า is the only mode designed for long runs, which is why nothing else has
// ever noticed. Progress goes on the SCREEN, as step states on the plan card;
// the conversation carries only what changes what happens next.

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/mode"
)

// goalRun is one conversation carrying out its plan.
//
// Held in memory rather than in the database, and the difference from the plan
// itself is deliberate: a plan is a document meant to outlive the turn that
// wrote it, and a run is a turn in progress. A run that survived a restart would
// be a window offering to stop something that is not happening.
type goalRun struct {
	sessionID string
	// finishAsked records that the second gate has fired. Once per run — see the
	// note above about a gate that argues.
	finishAsked bool
	// sentBack counts the verdicts issued, for the card's own run bar and for
	// nothing else — the ceiling is MaxToolCalls and is not this. It is the one
	// part of this mode the user cannot otherwise see: the gates fire between
	// the model speaking and the turn ending, so a run pushed back four times
	// looks exactly like one that sailed through.
	sentBack int
	// startedAt is RFC3339, for the bar to count up from. The window keeps the
	// clock; pushing one from here would be an event stream that says nothing.
	startedAt string
	// paused is why the run is holding, and "" when it is not.
	//
	// **Pausing is not stopping, and the difference is where it takes effect.**
	// A stop would have to cut a turn mid-round and lose whatever was in flight.
	// This instead makes the goal check LET GO: the turn ends where it naturally
	// would, having finished what it was doing, and the run stays on the card
	// waiting to be told to go on. Nothing is thrown away, which is the whole
	// promise of a button called พัก.
	paused string
}

// goalRuns is the set of conversations currently carrying out a plan.
//
// Keyed by session for the reason everything in this app is (§187, §234): a run
// belongs to one conversation, and a window showing another one must not offer
// to stop it.
type goalRuns struct {
	mu   sync.Mutex
	runs map[string]*goalRun
}

func newGoalRuns() *goalRuns { return &goalRuns{runs: map[string]*goalRun{}} }

// goalRunSet builds the set on first use.
//
// Once, and not a bare nil check, for the reason a.cur() gives one screen over:
// most of this package's tests construct a zero App, and two goroutines finding
// a nil field at the same time each build their own set and then disagree about
// which conversations are running.
func (a *App) goalRunSet() *goalRuns {
	a.goalRunsOnce.Do(func() {
		if a.goalRunsSet == nil {
			a.goalRunsSet = newGoalRuns()
		}
	})
	return a.goalRunsSet
}

func (g *goalRuns) start(sessionID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.runs[sessionID] = &goalRun{sessionID: sessionID, startedAt: time.Now().Format(time.RFC3339)}
}

func (g *goalRuns) stop(sessionID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.runs, sessionID)
}

func (g *goalRuns) get(sessionID string) *goalRun {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.runs[sessionID]
}

func (g *goalRuns) running(sessionID string) bool { return g.get(sessionID) != nil }

// goalCheck is the question the turn loop asks when it is about to end
// (turn.TurnOptions.OnGoalCheck). It returns the verdict to keep working on, or
// "" to let the turn finish.
//
// Reads the plan fresh every time rather than holding a copy: the model has been
// marking steps through the tool while this run has been going, so a copy taken
// at the start would be checking against the plan as it was before any of the
// work happened.
func (a *App) goalCheck(sessionID string) func(string) string {
	return func(answer string) string {
		run := a.goalRunSet().get(sessionID)
		if run == nil {
			// Stopped from the window mid-turn. Let the turn end — the user
			// pressed stop on the run, not on the turn, and cutting the answer
			// off would lose work that has already been done.
			return ""
		}
		plan, err := a.loadPlan(sessionID)
		if err != nil || plan == nil {
			// No plan to carry out is not a failure to report at the model; it
			// is a run that should never have started. End the turn.
			a.goalRunSet().stop(sessionID)
			return ""
		}

		// HOLDING. Asked before either gate, because a paused run is not a run
		// that failed a check — it is one nobody is asking to continue. Letting
		// the turn end is the entire mechanism: what was in flight finished, and
		// the run sits on the card until ไปต่อ.
		if run.paused != "" {
			a.emitPlan(sessionID, *plan)
			return ""
		}

		// GATE ONE, mechanical. Nothing here is a judgement about the work.
		if left := unfinishedSteps(plan.Steps); len(left) > 0 {
			// A BREAKPOINT ON THE NEXT STEP HOLDS THE RUN instead of pushing it
			// forward. Checked here rather than anywhere else because this is
			// the one moment the run is between steps: the model has stopped,
			// everything it did is marked, and the next thing has not started.
			//
			// Only the NEXT open step is looked at. A breakpoint further down
			// the list is not this moment's business, and firing on it early
			// would stop the run before the work the user was happy to have
			// done unattended.
			if left[0].Stop {
				run.paused = fmt.Sprintf("รอก่อนทำข้อ %d", left[0].N)
				a.emitPlan(sessionID, *plan)
				return ""
			}
			run.sentBack++
			a.emitPlan(sessionID, *plan)
			return unfinishedVerdict(left)
		}

		// GATE TWO, the finish condition, once.
		if !run.finishAsked {
			run.finishAsked = true
			if cond := sectionBody(plan.Sections, finishHeading); cond != "" {
				run.sentBack++
				a.emitPlan(sessionID, *plan)
				return "Every step of the plan is marked done. Before this turn ends, the plan's own " +
					"finish condition has to be met, and these are its words:\n\n" + cond + "\n\n" +
					"Check it — actually run or read whatever settles it — and say in one or two lines " +
					"what you checked and what came back. If it does not hold, the work is not finished: " +
					"mark the step that has to be redone and carry on. Do not restate the plan."
			}
		}

		// Done. The run ends with the turn, so the card stops saying it is
		// working without the window having to ask.
		a.goalRunSet().stop(sessionID)
		a.emitPlan(sessionID, *plan)
		return ""
	}
}

// unfinishedVerdict is what a run that is not done is told.
//
// It names the steps and says nothing else. A verdict is a message in the
// conversation and is re-sent with every later round of this turn, so its length
// is paid for repeatedly — the same arithmetic that keeps this mode from
// narrating (see the note at the top).
func unfinishedVerdict(left []PlanStep) string {
	var b strings.Builder
	b.WriteString("Not finished. These steps of the plan are not marked done:\n")
	for _, s := range left {
		fmt.Fprintf(&b, "%d. %s\n", s.N, s.Text)
	}
	b.WriteString("Carry on, and mark each one with `plan` (action: step) as you finish it. " +
		"If a step turns out to be wrong or impossible, mark it failed with a reason rather than leaving it open.")
	return b.String()
}

func unfinishedSteps(steps []PlanStep) []PlanStep {
	var out []PlanStep
	for _, s := range steps {
		// `failed` counts as settled. A step the model has looked at and
		// reported impossible is a finding the user needs, and sending the turn
		// back over it forever would be the run refusing to accept an answer.
		if s.State != planStepDone && s.State != planStepFailed {
			out = append(out, s)
		}
	}
	return out
}

func sectionBody(sections []PlanSection, heading string) string {
	for _, s := range sections {
		if strings.EqualFold(s.Heading, heading) {
			return strings.TrimSpace(s.Body)
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// what the window calls
// ---------------------------------------------------------------------------

// PlanRunStart is what the button gets back: why it could not start, and
// whether it did.
//
// **`Started` exists because pressing the button has to SEND something, and
// finding that out cost a real run.** Starting a run switches the stance,
// registers the run and installs the goal check — and none of that makes
// anything happen, because the check only fires when a turn is about to END and
// there is no turn. The bar sat at 0/5 with the light on, waiting for the user
// to type. A button that says "carry this out" and only arms the machinery is a
// button that lies.
//
// What it does NOT carry is the message. That was the first version and the
// owner refused it on sight, correctly: a Thai sentence built in Go, saying
// "work one step at a time, mark each with plan step, do not narrate" — three
// separate debts in one string.
//
//   - Screen text hardcoded past the locale files, so an English user gets Thai.
//   - A PROMPT in a user turn, when this repo has one home for telling a model
//     how to work a tool.
//   - Re-sent with every later round of the run, because a user turn is. A
//     paragraph of instruction, paid for on every round, in the one mode built
//     around the fact that exactly this cost is quadratic.
//
// The words are now in the two places they belong: what the user "says" is a
// locale string in the window, short, the way somebody would actually type it —
// and what the model is told is planSkill.Guidance (plan.go), which this repo's
// own machinery delivers ONCE, with the first result, and never again.
type PlanRunStart struct {
	Refusal string `json:"refusal,omitempty"`
	Started bool   `json:"started,omitempty"`
}

// StartPlanRun begins carrying out this conversation's plan — the button on the
// plan card.
//
// Refused rather than silently ignored when there is nothing to carry out, and
// the two refusals say different things because they send the user to different
// places: no plan at all, and a plan with no steps in it.
func (a *App) StartPlanRun(sessionID string) PlanRunStart {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return PlanRunStart{Refusal: "ห้องนี้ยังไม่มีบทสนทนา"}
	}
	plan, err := a.loadPlan(sessionID)
	if err != nil {
		return PlanRunStart{Refusal: err.Error()}
	}
	if plan == nil {
		return PlanRunStart{Refusal: "ยังไม่มีแผนในห้องนี้"}
	}
	if len(plan.Steps) == 0 {
		// The honest message: มุ่งเป้า with no checkable steps is "try harder for
		// longer", which is the thing §106.10 declined.
		return PlanRunStart{Refusal: "แผนนี้ยังไม่มีขั้นตอนที่ตรวจได้ ให้วางแผนใหม่โดยให้แต่ละขั้นเป็นข้อ"}
	}
	// **THE PRESS CROSSES FROM PLANNING TO ACTING, and it has to.**
	//
	// Without this the button was a promise the session could not keep: วางแผน
	// carries no `write`, no `edit`, no `shell`, no `task` — and deliberately no
	// `plan_step` either, because marking a step done asserts that work
	// happened and that is the one thing this stance must never be able to say.
	// A run started there could not touch anything, could not mark anything, and
	// would be sent back by its own first gate until it hit the ceiling. The
	// owner found it on the first real run: *"มันควรจะลงมือทำจริง ไม่เห็นสวิชไป
	// ลงมืออ่ะ"*.
	//
	// It is not a patch on that hole, it is what the button always meant.
	// มุ่งเป้า is วางแผน's ACTING leg (§236): the press pins the goal and walks
	// through the door in the same motion, which is ประตูส่งไม้ (COMPANY.md §8)
	// one room smaller — a plan handed over, in this conversation, to the stance
	// that can carry it out.
	//
	// Switched through SetStance rather than by writing the field, so the engine
	// is rebuilt and the prompt and the dispatcher are both told. Its refusal
	// while a turn is running is kept, and returned as this call's refusal: a
	// run started against a session mid-turn would be aimed at an engine that is
	// about to be replaced.
	if stance, err := a.SetStance(mode.StanceAct.String()); err != nil {
		return PlanRunStart{Refusal: err.Error()}
	} else if mode.NormalizeStance(stance) != mode.StanceAct {
		return PlanRunStart{Refusal: "เปลี่ยนไปโหมดลงมือไม่สำเร็จ"}
	}
	a.goalRunSet().start(sessionID)
	if a.cur() != nil && a.cur().chat != nil {
		a.cur().chat.SetGoalCheck(a.goalCheck(sessionID))
	}
	a.emitPlan(sessionID, *plan)
	return PlanRunStart{Started: true}
}

// StopPlanRun puts the run down. The turn it is in keeps going — the user
// stopped the RUN, not the turn, and the Stop button is the one that does the
// other thing.
func (a *App) StopPlanRun(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	a.goalRunSet().stop(sessionID)
	if a.cur() != nil && a.cur().chat != nil {
		a.cur().chat.SetGoalCheck(nil)
	}
	if plan, err := a.loadPlan(sessionID); err == nil && plan != nil {
		a.emitPlan(sessionID, *plan)
	}
}

// PausePlanRun holds the run at the end of what it is doing.
//
// Not a stop, and the button says so: the turn in flight finishes, every mark it
// made stands, and the run stays on the card. It is the control for "wait, let
// me think" — which is the moment this mode is otherwise worst at, because it is
// built to be left alone.
func (a *App) PausePlanRun(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if run := a.goalRunSet().get(sessionID); run != nil && run.paused == "" {
		run.paused = "พักตามที่สั่ง"
	}
	if plan, err := a.loadPlan(sessionID); err == nil && plan != nil {
		a.emitPlan(sessionID, *plan)
	}
}

// ResumePlanRun lets a held run go on.
//
// It clears the hold and, when the hold came from a breakpoint, clears that too
// — otherwise pressing ไปต่อ would stop again at the same step, which reads as
// the button not working. A breakpoint is a request to be asked once, not a
// wall.
func (a *App) ResumePlanRun(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	run := a.goalRunSet().get(sessionID)
	if run == nil {
		return
	}
	run.paused = ""
	plan, err := a.loadPlan(sessionID)
	if err != nil || plan == nil {
		return
	}
	if left := unfinishedSteps(plan.Steps); len(left) > 0 && left[0].Stop {
		for i := range plan.Steps {
			if plan.Steps[i].N == left[0].N {
				plan.Steps[i].Stop = false
			}
		}
		_ = a.savePlan(sessionID, *plan)
	}
	a.emitPlan(sessionID, *plan)
}

// SetPlanStepStop puts a breakpoint on a step, or takes one off.
//
// On the PLAN rather than on the run, so it survives the run ending and is there
// for the next one — and so it can be set before a run starts, which is when
// somebody reading a fresh plan actually notices the step they want to see.
func (a *App) SetPlanStepStop(sessionID string, n int, on bool) {
	sessionID = strings.TrimSpace(sessionID)
	plan, err := a.loadPlan(sessionID)
	if err != nil || plan == nil {
		return
	}
	for i := range plan.Steps {
		if plan.Steps[i].N == n {
			plan.Steps[i].Stop = on
		}
	}
	if err := a.savePlan(sessionID, *plan); err != nil {
		return
	}
	a.emitPlan(sessionID, *plan)
}

// PlanRunning reports whether this conversation is carrying out its plan, for a
// window that has just opened it.
func (a *App) PlanRunning(sessionID string) bool {
	return a.goalRunSet().running(strings.TrimSpace(sessionID))
}
