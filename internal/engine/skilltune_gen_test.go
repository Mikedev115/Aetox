package engine

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

type fakeDrafter struct {
	op, before, body, reason string
	calls                    int
	sawEvidence              string
	sawRefused               string
}

func (f *fakeDrafter) Draft(_ context.Context, _, _, evidence, refused string) (string, string, string, string, error) {
	f.calls++
	f.sawEvidence = evidence
	f.sawRefused = refused
	return f.op, f.before, f.body, f.reason, nil
}

type pendingRow struct {
	scope, op, before, body, source, evidence string
}

func pendingByKind(t *testing.T, a *Engine, kind string) []pendingRow {
	t.Helper()
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	rows, err := db.Query(
		`SELECT scope, op, before, body, source, evidence FROM pending_changes
		  WHERE kind = ? AND state = ? ORDER BY id`, kind, statePending)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []pendingRow
	for rows.Next() {
		var r pendingRow
		if err := rows.Scan(&r.scope, &r.op, &r.before, &r.body, &r.source, &r.evidence); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != (error)(nil) && err != sql.ErrNoRows {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// The whole stage-three pipeline without an LLM: three bad ratings flag a skill,
// the drafter (faked) proposes an edit, and it lands as a skill-kind proposal in
// the approval queue — grounded in the job rows, sourced to the optimizer, and
// nothing applied until a human approves.
func TestGeneratorQueuesADraftedSkillEdit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	a := newJobApp(t)
	for i := int64(1); i <= 3; i++ {
		skillJob(t, a, i, "aetox-slides", outcomeBad)
	}

	f := &fakeDrafter{op: "add", body: "OPTIMIZER-DRAFTED-LINE", reason: "เพราะโดน 👎 ซ้ำ"}
	a.generateSkillRefinements(context.Background(), f)

	rows := pendingByKind(t, a, kindSkill)
	if len(rows) != 1 {
		t.Fatalf("want one skill proposal, got %d", len(rows))
	}
	r := rows[0]
	if r.scope != "aetox-slides" || r.body != "OPTIMIZER-DRAFTED-LINE" {
		t.Errorf("proposal = %+v, want the drafted edit on aetox-slides", r)
	}
	if r.source != "optimizer" {
		t.Errorf("source = %q — the self-optimize loop must read distinct from the model deciding mid-task", r.source)
	}
	if !strings.HasPrefix(r.evidence, "jobs:") {
		t.Errorf("evidence should name the job rows it was drawn from: %q", r.evidence)
	}
	if !strings.Contains(f.sawEvidence, "ถาม:") {
		t.Errorf("the drafter was handed no grounded evidence: %q", f.sawEvidence)
	}

	// A second pass must not re-call the model or duplicate: the skill already
	// has a proposal waiting.
	a.generateSkillRefinements(context.Background(), f)
	if f.calls != 1 {
		t.Errorf("the model was called %d times; a skill with a pending proposal must not be re-drafted", f.calls)
	}
	if rows := pendingByKind(t, a, kindSkill); len(rows) != 1 {
		t.Errorf("a duplicate proposal was queued: %d rows", len(rows))
	}
}

// A drafter that returns an empty body is the "not the skill's fault" answer —
// nothing is queued.
func TestGeneratorQueuesNothingWhenTheDrafterDeclines(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	a := newJobApp(t)
	for i := int64(1); i <= 3; i++ {
		skillJob(t, a, i, "aetox-slides", outcomeBad)
	}
	a.generateSkillRefinements(context.Background(), &fakeDrafter{op: "add", body: ""})
	if rows := pendingByKind(t, a, kindSkill); len(rows) != 0 {
		t.Fatalf("an empty draft was queued anyway: %+v", rows)
	}
}

// The memory queue's finding, on this queue (DECISIONS §246): a refused edit
// used to be re-drafted on the very next pass from the same misfires, and a
// model asked twice answers in different words — so the user's no bought one
// pass of quiet and then a new card. A no stands until a bad rating arrives
// that is newer than it; and when one does, the drafter is shown what was
// refused, and the door refuses a restatement of it anyway.
func TestARefusedSkillEditIsNotRedraftedUntilSomethingNewHappens(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	a := newJobApp(t)
	for i := int64(1); i <= 3; i++ {
		skillJob(t, a, i, "aetox-slides", outcomeBad)
	}
	db, err := a.database()
	if err != nil {
		t.Fatal(err)
	}
	// Rated yesterday, refused today.
	if _, err := db.Exec(`UPDATE jobs SET time = '2026-09-10T09:00:00+07:00'`); err != nil {
		t.Fatal(err)
	}
	f := &fakeDrafter{op: "add", body: "เปิดอ่านเทมเพลตก่อนสร้างสไลด์ทุกครั้ง แล้วยึดสีจากเทมเพลตนั้น", reason: "โดน 👎"}
	a.generateSkillRefinements(context.Background(), f)
	rows := pendingByKind(t, a, kindSkill)
	if len(rows) != 1 {
		t.Fatalf("want one proposal, got %d", len(rows))
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM pending_changes WHERE kind = ? AND state = ?`, kindSkill, statePending).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := a.RejectPendingChange(id); err != nil {
		t.Fatalf("reject: %v", err)
	}

	// Same misfires, next pass: no model call, no card.
	f.body = "ก่อนสร้างสไลด์ ให้เปิดอ่านเทมเพลตก่อนทุกครั้ง และใช้สีของเทมเพลตนั้น"
	a.generateSkillRefinements(context.Background(), f)
	if f.calls != 1 {
		t.Errorf("the drafter was called %d times; a refusal with nothing new since must not be re-drafted", f.calls)
	}
	if rows := pendingByKind(t, a, kindSkill); len(rows) != 0 {
		t.Fatalf("a refused edit came back: %+v", rows)
	}

	// A new bad rating, rated after the refusal: the question is open again —
	// the drafter is called, shown the refusal, and its restatement of the
	// refused edit stops at the door.
	skillJob(t, a, 4, "aetox-slides", outcomeBad)
	if _, err := db.Exec(`UPDATE jobs SET time = '2099-01-01T00:00:00+07:00' WHERE message_id = 4`); err != nil {
		t.Fatal(err)
	}
	a.generateSkillRefinements(context.Background(), f)
	if f.calls != 2 {
		t.Errorf("the drafter was called %d times; new evidence must reopen the question", f.calls)
	}
	if !strings.Contains(f.sawRefused, "เปิดอ่านเทมเพลตก่อนสร้างสไลด์ทุกครั้ง") {
		t.Errorf("the drafter was not shown the refused edit: %q", f.sawRefused)
	}
	if rows := pendingByKind(t, a, kindSkill); len(rows) != 0 {
		t.Fatalf("a restatement of the refused edit was queued: %+v", rows)
	}

	// A genuinely different edit goes through.
	f.body = "ถ้าผู้ใช้ส่งโลโก้มา ให้วางไว้มุมขวาล่างของทุกสไลด์"
	a.generateSkillRefinements(context.Background(), f)
	if rows := pendingByKind(t, a, kindSkill); len(rows) != 1 {
		t.Fatalf("a different edit should be queued, got %d rows", len(rows))
	}
}
