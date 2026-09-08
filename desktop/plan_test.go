package main

import (
	"context"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/mode"
)

// planApp is a conversation with a database and an id — the state a tool is
// always in when it runs for real, and the one a bare &App{} is not.
func planApp(t *testing.T) *planSkill {
	t.Helper()
	app := &App{}
	isolateUserDirs(t)
	closeDBOnCleanup(t, app)
	app.cur().id = newSessionID()
	return &planSkill{app: app, conv: app.cur()}
}

func call(t *testing.T, s *planSkill, args map[string]any) string {
	t.Helper()
	out, err := s.ExecuteTool(context.Background(), args)
	if err != nil {
		t.Fatalf("plan %v: %v", args["action"], err)
	}
	if !out.Success {
		t.Fatalf("plan %v did not succeed: %s", args["action"], out.Content)
	}
	return out.Content
}

func section(heading, body string) any {
	return map[string]any{"heading": heading, "body": body}
}

// The reason this tool exists: a revision costs the section that changed, not
// the document that did not.
func TestAmendLeavesTheSectionsNobodyNamedAlone(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write", "title": "ย้ายการรอไปเป็น 600 วิ", "sections": []any{
		section("What is there now", "waitMax is 60s"),
		section("What to change", "1. raise waitMax\n2. teach the guidance"),
		section("What could go wrong", "a turn held open for ten minutes"),
	}})

	call(t, s, map[string]any{"action": "amend", "sections": []any{
		section("What to change", "1. raise waitMax\n2. teach the guidance\n3. report progress"),
	}})

	plan, err := s.app.loadPlan(s.sessionID())
	if err != nil || plan == nil {
		t.Fatalf("the plan is gone: %v", err)
	}
	if len(plan.Sections) != 3 {
		t.Fatalf("amend changed the shape of the plan: %d sections", len(plan.Sections))
	}
	body := planMarkdown(*plan)
	if !strings.Contains(body, "3. report progress") {
		t.Error("the amended section did not take")
	}
	// The two nobody mentioned are the assertion. Before the tool, a revision
	// meant the model writing these out again — every byte of them, paid twice.
	if !strings.Contains(body, "waitMax is 60s") || !strings.Contains(body, "held open for ten minutes") {
		t.Errorf("amend dropped a section it was not asked about:\n%s", body)
	}
	if plan.Version != 2 {
		t.Errorf("version is %d, so the card cannot say this is a revision", plan.Version)
	}
}

// `Changed` rides on the event and is deliberately NOT stored, which is easy to
// read as an oversight: the card marks the sections a revision touched, and a
// chat reopened three days later should not still be highlighting them. It is a
// fact about the call, not about the plan.
func TestTheEventNamesWhatTheAmendTouched(t *testing.T) {
	s := planApp(t)
	var last Plan
	s.app.emit = func(event string, data ...any) {
		if event != "plan:update" || len(data) == 0 {
			return
		}
		if ev, ok := data[0].(sessionEvent[Plan]); ok {
			last = ev.Data
		}
	}
	call(t, s, map[string]any{"action": "write", "sections": []any{
		section("What is there now", "sixty seconds"),
		section("What to change", "1. first"),
	}})
	if len(last.Changed) != 0 {
		t.Errorf("a first draft marked sections as changed: %v", last.Changed)
	}
	call(t, s, map[string]any{"action": "amend", "sections": []any{
		section("What to change", "1. second"),
	}})
	if len(last.Changed) != 1 || last.Changed[0] != "What to change" {
		t.Fatalf("the event marks %v — the card would highlight the wrong section", last.Changed)
	}
	if last.Version != 2 {
		t.Errorf("the event says v%d", last.Version)
	}
	// The whole plan travels, not just the delta: the card draws from this, so
	// a window that missed an earlier event still ends up showing the truth.
	if len(last.Sections) != 2 {
		t.Errorf("the event carried %d sections; the card cannot draw a plan it was not given", len(last.Sections))
	}
	// And it is stamped, or a background chat's revision lands on the plan the
	// user is reading — the mistake §187 and §234 are both about.
	if last.Sections[1].Body != "1. second" {
		t.Errorf("the event carried a stale body: %q", last.Sections[1].Body)
	}
}

// A model that writes a heading in its own casing must amend the section it
// meant, not grow a second one beside it. Silent, cumulative, and invisible in
// any single answer: the plan simply has two "What to change" by the third pass.
func TestAHeadingInTheWrongCaseStillFindsItsSection(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write", "sections": []any{
		section("What to change", "1. first"),
	}})
	call(t, s, map[string]any{"action": "amend", "sections": []any{
		section("what TO change", "1. second"),
	}})

	plan, _ := s.app.loadPlan(s.sessionID())
	if len(plan.Sections) != 1 {
		t.Fatalf("a second section appeared: %+v", plan.Sections)
	}
	if plan.Sections[0].Heading != "What to change" {
		t.Errorf("the stored heading is %q rather than the canonical spelling", plan.Sections[0].Heading)
	}
	if plan.Sections[0].Body != "1. second" {
		t.Errorf("the amend did not land: %q", plan.Sections[0].Body)
	}
}

// An amend arrives out of order because it names only what moved. The stored
// plan is put back into the shape's order, because reading every plan the same
// way is what the dial was turned for (§106.11).
func TestSectionsAreStoredInTheShapesOrder(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write", "sections": []any{
		section("What you are unsure of", "whether the ceiling is enough"),
		section("What is there now", "sixty seconds"),
	}})

	plan, _ := s.app.loadPlan(s.sessionID())
	order := mode.PlanHeadings()
	last := -1
	for _, sec := range plan.Sections {
		at := indexOf(order, sec.Heading)
		if at < last {
			t.Fatalf("sections are out of the shape's order: %v", headingsOf(plan.Sections))
		}
		last = at
	}
}

// `amend` on a conversation with no plan is a write rather than a refusal: it
// is the honest reading of the call, and refusing would spend a round teaching
// a distinction the result does not depend on.
func TestAmendWithNothingToAmendWritesThePlan(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "amend", "sections": []any{
		section("What to change", "1. only this"),
	}})
	plan, _ := s.app.loadPlan(s.sessionID())
	if plan == nil || plan.Version != 1 {
		t.Fatalf("the plan was not written as a first draft: %+v", plan)
	}
}

// "No plan yet" and "the plan is empty" send a model to different places, and
// only one of them is true before anything has been written.
func TestReadingBeforeThereIsAPlanSaysSo(t *testing.T) {
	s := planApp(t)
	out := call(t, s, map[string]any{"action": "read"})
	if !strings.Contains(out, "no plan") {
		t.Errorf("the answer does not say the plan is missing: %q", out)
	}
}

// The receipt is not the plan, and that is the whole economy of this tool: the
// model wrote the plan once, and handing it back would put every byte of it
// into context a second time.
func TestTheReceiptDoesNotHandThePlanBack(t *testing.T) {
	s := planApp(t)
	body := "1. a step whose words should not come back"
	out := call(t, s, map[string]any{"action": "write", "sections": []any{
		section("What to change", body),
	}})
	if strings.Contains(out, body) {
		t.Errorf("the receipt repeated the plan:\n%s", out)
	}
	if !strings.Contains(out, "What to change") {
		t.Errorf("the receipt does not say what the plan holds:\n%s", out)
	}
	// Named rather than left to be inferred: a heading the model dropped by
	// accident is worth telling it about, and one it left out on purpose is not
	// worth refusing.
	if !strings.Contains(out, "Still empty") {
		t.Errorf("the receipt does not name the headings still unfilled:\n%s", out)
	}
}

// A failed plan call has to say why. Every error path used to return a receipt
// with nothing written on it, so the row the user saw was blank — found by
// tool_coverage_test.go, whose own report of the failure was blank for the same
// reason.
func TestARefusedPlanCallSaysWhy(t *testing.T) {
	s := planApp(t)
	out, err := s.ExecuteTool(context.Background(), map[string]any{"action": "sing"})
	if err == nil {
		t.Fatal("plan accepted an action it does not have")
	}
	if strings.TrimSpace(out.Content) == "" || strings.TrimSpace(out.Stderr) == "" {
		t.Errorf("the refusal is invisible: content=%q stderr=%q", out.Content, out.Stderr)
	}
}

// The plan outlives the turn that wrote it, which is what separates it from the
// todo list next door and what makes ลงมือ able to carry one out.
func TestThePlanSurvivesForTheSessionToReadBack(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write", "title": "the job", "sections": []any{
		section("What to change", "1. the step"),
	}})
	// A second skill over the same App and conversation is what a later turn —
	// or a later stance — looks like from here.
	later := &planSkill{app: s.app, conv: s.conv}
	out := call(t, later, map[string]any{"action": "read"})
	if !strings.Contains(out, "the job") || !strings.Contains(out, "1. the step") {
		t.Errorf("the plan did not come back:\n%s", out)
	}
}

func indexOf(list []string, want string) int {
	for i, s := range list {
		if s == want {
			return i
		}
	}
	return len(list)
}

func headingsOf(secs []PlanSection) []string {
	out := make([]string, 0, len(secs))
	for _, s := range secs {
		out = append(out, s.Heading)
	}
	return out
}

// A PLAN WITH NO STEPS HAS NO CHECKLIST AND NO BUTTON — which is what the owner
// saw on the first real plan of the day. The model had written its steps as a
// numbered list inside "What to change", a perfectly reasonable reading of a
// prompt that said "number the steps", and passed no `steps` at all. The card
// came back inert with nothing to say why.
//
// A feature that disappears when one parameter is omitted is worse than one that
// was never built, so the steps are recovered from the prose.
func TestStepsWrittenAsProseAreRecovered(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write", "title": "ทดสอบเบราว์เซอร์", "sections": []any{
		section("What is there now", "เปิด/คลิก/พิมพ์ได้แล้ว"),
		section("What to change", "1. ทดสอบ screenshot\n2. ทดสอบคลิกพิกัดพิกเซล\n3. ทดสอบ drag"),
		section("What could go wrong", "หน้าต่างอาจยังถูกซ่อน"),
	}})

	plan, _ := s.app.loadPlan(s.sessionID())
	if len(plan.Steps) != 3 {
		t.Fatalf("the checklist was not recovered: %+v", plan.Steps)
	}
	if plan.Steps[0].Text != "ทดสอบ screenshot" || plan.Steps[2].N != 3 {
		t.Errorf("recovered wrongly: %+v", plan.Steps)
	}
}

// Only from the section that describes the work. A numbered list under "What
// could go wrong" is a list of risks, and turning it into a checklist would hand
// the user a button that carries out the things they were warned about.
func TestOnlyTheWorkSectionBecomesSteps(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write", "sections": []any{
		section("What could go wrong", "1. หน้าต่างถูกซ่อน\n2. เว็บบล็อก automation\n3. DPI ไม่ตรง"),
	}})
	plan, _ := s.app.loadPlan(s.sessionID())
	if len(plan.Steps) != 0 {
		t.Fatalf("risks became work to carry out: %+v", plan.Steps)
	}
}

// What the call actually said always wins. Recovering over the top of it would
// be the tool deciding it knows better than its own arguments.
func TestNamedStepsAreNotOverriddenByTheProse(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write",
		"sections": []any{section("What to change", "1. อย่างหนึ่ง\n2. อย่างสอง\n3. อย่างสาม")},
		"steps":    []any{"ขั้นจริง"}})
	plan, _ := s.app.loadPlan(s.sessionID())
	if len(plan.Steps) != 1 || plan.Steps[0].Text != "ขั้นจริง" {
		t.Fatalf("the prose overrode what the call passed: %+v", plan.Steps)
	}
}

// One line that starts with a digit is a sentence, not a checklist. The cost of
// guessing wrong here is a card offering to carry out a paragraph.
func TestASingleNumberedLineIsNotAChecklist(t *testing.T) {
	s := planApp(t)
	call(t, s, map[string]any{"action": "write", "sections": []any{
		section("What to change", "1. อย่างเดียวเท่านั้น"),
	}})
	plan, _ := s.app.loadPlan(s.sessionID())
	if len(plan.Steps) != 0 {
		t.Errorf("a single line became a checklist: %+v", plan.Steps)
	}
}
