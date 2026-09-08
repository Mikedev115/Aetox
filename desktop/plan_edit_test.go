package main

import (
	"strings"
	"testing"
)

// THE PROPERTY THE WHOLE DESIGN RESTS ON.
//
// The user edits the plan as markdown, and the objection to that shape is that
// parsing a document back out of prose is fragile. It is answered by owning both
// ends: planMarkdown writes exactly what parsePlanMarkdown reads. This test is
// what keeps that true — change either side and forget the other, and it fails
// here rather than in front of somebody who has just lost their plan.
func TestPlanMarkdownRoundTrips(t *testing.T) {
	original := Plan{
		Title: "ยกเพดานการรอของ browser เป็น 600 วินาที",
		Sections: []PlanSection{
			{Heading: "What is there now", Body: "`waitMax` = 60s\nและ `toolCallDeadline` ก็ตัดที่ 60s"},
			{Heading: "What to change", Body: "ดูขั้นตอนด้านล่าง"},
			{Heading: "How you will know it worked", Body: "`go test ./desktop/` ผ่าน"},
		},
		Steps: []PlanStep{
			{N: 1, Text: "ยก waitMax เป็น 600s", State: planStepDone},
			{N: 2, Text: "เพิ่ม browser_wait เข้า toolCallDeadline", State: planStepFailed, Note: "เดดไลน์ไปไม่ถึง"},
			{N: 3, Text: "รายงานความคืบหน้าทุก 5 วิ", State: planStepDoing},
			{N: 4, Text: "เขียนเทสต์ผูกเพดานสองตัว"},
		},
	}

	back := parsePlanMarkdown(planMarkdown(original), &original)

	if back.Title != original.Title {
		t.Errorf("title: %q -> %q", original.Title, back.Title)
	}
	if len(back.Sections) != len(original.Sections) {
		t.Fatalf("sections: %d -> %d (%v)", len(original.Sections), len(back.Sections), headingsOf(back.Sections))
	}
	for i, want := range original.Sections {
		if back.Sections[i].Heading != want.Heading {
			t.Errorf("section %d heading: %q -> %q", i, want.Heading, back.Sections[i].Heading)
		}
		if strings.TrimSpace(back.Sections[i].Body) != strings.TrimSpace(want.Body) {
			t.Errorf("section %q body:\n want %q\n  got %q", want.Heading, want.Body, back.Sections[i].Body)
		}
	}
	if len(back.Steps) != len(original.Steps) {
		t.Fatalf("steps: %d -> %d (%+v)", len(original.Steps), len(back.Steps), back.Steps)
	}
	for i, want := range original.Steps {
		got := back.Steps[i]
		if got.Text != want.Text || got.State != want.State || got.Note != want.Note || got.N != want.N {
			t.Errorf("step %d:\n want %+v\n  got %+v", i+1, want, got)
		}
	}
}

// A person writes a to-do list the way markdown writes one. Refusing that would
// be the parser being right about its own output and wrong about the user.
func TestAHandWrittenChecklistIsUnderstood(t *testing.T) {
	src := "# ทำให้เสร็จ\n\n" +
		"**What to change**\nตามนี้\n\n" +
		"- [x] 1. อ่านโค้ดเดิม\n" +
		"- [ ] 2. แก้ค่าคงที่\n" +
		"3. เขียนเทสต์\n"
	p := parsePlanMarkdown(src, nil)
	if len(p.Steps) != 3 {
		t.Fatalf("read %d steps from a hand-written list: %+v", len(p.Steps), p.Steps)
	}
	if p.Steps[0].State != planStepDone {
		t.Errorf("a ticked box was not read as done: %+v", p.Steps[0])
	}
	if p.Steps[2].Text != "เขียนเทสต์" {
		t.Errorf("a bare numbered line was not read as a step: %+v", p.Steps[2])
	}
}

// THE FAILURE THAT WOULD MAKE HAND EDITING WORSE THAN USELESS: prose becoming
// steps. A plan that grows nine steps because somebody wrote a paragraph is a
// plan มุ่งเป้า would then refuse to finish.
func TestProseDoesNotBecomeSteps(t *testing.T) {
	src := "**What could go wrong**\n" +
		"เทิร์นค้าง 10 นาทีโดยจอนิ่ง คนดูแยกไม่ออกจากการแขวน\n" +
		"ถ้าเกิดขึ้นจริงจะเสียงานที่ทำไปแล้ว\n\n" +
		"**What to change**\n" +
		"1. แก้ค่าคงที่\n"
	p := parsePlanMarkdown(src, nil)
	if len(p.Steps) != 1 {
		t.Fatalf("prose became steps: %+v", p.Steps)
	}
	body := sectionBody(p.Sections, "What could go wrong")
	if !strings.Contains(body, "จอนิ่ง") || !strings.Contains(body, "เสียงานที่ทำไปแล้ว") {
		t.Errorf("the prose lost lines on the way through:\n%q", body)
	}
}

// Editing the wording of one step must not un-tick the others. States live
// nowhere in the text a person edits, so they are carried across by matching
// text — and matched on TEXT rather than on number, because a number is exactly
// what a hand edit moves.
func TestEditingOneStepKeepsTheOthersMarks(t *testing.T) {
	old := Plan{Steps: []PlanStep{
		{N: 1, Text: "อ่านโค้ดเดิม", State: planStepDone},
		{N: 2, Text: "แก้ค่าคงที่", State: planStepDone},
		{N: 3, Text: "เขียนเทสต์"},
	}}
	// The user inserts a step at the TOP and rewords the last one. Numbers all
	// shift; nothing that was finished should come back open.
	src := "1. คุยกับเจ้าของก่อน\n2. อ่านโค้ดเดิม\n3. แก้ค่าคงที่\n4. เขียนเทสต์ให้ครบสองด้าน\n"
	p := parsePlanMarkdown(src, &old)

	if len(p.Steps) != 4 {
		t.Fatalf("steps: %+v", p.Steps)
	}
	if p.Steps[0].State != planStepTodo {
		t.Errorf("a step the user just added came back already done: %+v", p.Steps[0])
	}
	if p.Steps[1].State != planStepDone || p.Steps[2].State != planStepDone {
		t.Errorf("finished work was un-ticked by an edit above it: %+v", p.Steps)
	}
	if p.Steps[3].State != planStepTodo {
		t.Errorf("a reworded step kept a mark it should have lost: %+v", p.Steps[3])
	}
	// Renumbered on the way in: the user should not have to renumber by hand.
	for i, st := range p.Steps {
		if st.N != i+1 {
			t.Errorf("step %d is numbered %d", i+1, st.N)
		}
	}
}

// A mark the person typed themselves wins over what the row used to be —
// overwriting it would be the app arguing with the edit.
func TestATypedMarkBeatsTheOldState(t *testing.T) {
	old := Plan{Steps: []PlanStep{{N: 1, Text: "อ่านโค้ดเดิม", State: planStepDone}}}
	p := parsePlanMarkdown("[!] 1. อ่านโค้ดเดิม — ไฟล์หายไปแล้ว\n", &old)
	if len(p.Steps) != 1 || p.Steps[0].State != planStepFailed {
		t.Fatalf("the typed mark was overwritten: %+v", p.Steps)
	}
	if p.Steps[0].Note != "ไฟล์หายไปแล้ว" {
		t.Errorf("the reason after the dash was lost: %+v", p.Steps[0])
	}
}

// A heading typed in the user's own casing, or as markdown's `##`, still lands
// on the section the tool can find by name.
func TestAHeadingTypedLooselyStillLands(t *testing.T) {
	p := parsePlanMarkdown("## what TO change\n1. ทำ\n", nil)
	if len(p.Sections) != 1 || p.Sections[0].Heading != "What to change" {
		t.Fatalf("heading did not fold onto the shape: %+v", p.Sections)
	}
}

// Saving an empty box is refused rather than read as "delete the plan": it is
// far more likely a slip, and the plan is what a run depends on.
func TestSavingAnEmptyPlanIsRefused(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	if msg := s.app.SavePlanText(s.sessionID(), "   \n\n"); msg == "" {
		t.Fatal("an empty box was accepted")
	}
	plan, _ := s.app.loadPlan(s.sessionID())
	if plan == nil || len(plan.Steps) != 3 {
		t.Error("the refusal did not leave the plan alone")
	}
}

// The model is told ONCE, at the start of the next turn, and the plan it reads
// is the new one.
func TestTheModelIsToldOnceThatThePlanWasEdited(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	id := s.sessionID()

	if note := s.app.planHandEditNote(id); note != "" {
		t.Errorf("a plan nobody touched produced a note: %q", note)
	}
	if msg := s.app.SavePlanText(id, "**What to change**\nตามนี้\n\n1. ขั้นเดียว\n"); msg != "" {
		t.Fatalf("the save was refused: %s", msg)
	}
	note := s.app.planHandEditNote(id)
	if !strings.Contains(note, "edited the plan by hand") {
		t.Fatalf("the next turn is not told: %q", note)
	}
	if second := s.app.planHandEditNote(id); second != "" {
		t.Errorf("the note arrives every turn, so a fact about the past is news forever: %q", second)
	}
}

// A hand edit bumps the version, because the card's revision mark is about the
// document changing rather than about who changed it.
func TestAHandEditIsARevision(t *testing.T) {
	s := goalApp(t)
	writeRunnablePlan(t, s)
	before, _ := s.app.loadPlan(s.sessionID())
	s.app.SavePlanText(s.sessionID(), "**What to change**\nใหม่\n\n1. ขั้นเดียว\n")
	after, _ := s.app.loadPlan(s.sessionID())
	if after.Version != before.Version+1 {
		t.Errorf("version went %d -> %d", before.Version, after.Version)
	}
}
