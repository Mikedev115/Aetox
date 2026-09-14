package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/learned"
)

type fakeHabitSynthesizer struct {
	proposal  *HabitProposal
	err       error
	sawDigest string
	sawHint   string
}

func (f *fakeHabitSynthesizer) Synthesize(_ context.Context, digest, hint string) (*HabitProposal, error) {
	f.sawDigest = digest
	f.sawHint = hint
	return f.proposal, f.err
}

func TestBuildSessionsDigest(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}

	sessionID := "test-sess-1"
	_, err = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Session 1', datetime('now'), datetime('now'))`, sessionID)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO messages (session_id, role, text, time)
		VALUES (?, 'user', 'เช็คกำลังไฟ GPU ให้ทีครับ', datetime('now'))`,
		sessionID,
	)
	if err != nil {
		t.Fatalf("insert user msg: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO messages (session_id, role, text, time)
		VALUES (?, 'assistant', 'กำลังตรวจเช็คกำลังไฟ GPU ผ่าน nvidia-smi ครับ', datetime('now'))`,
		sessionID,
	)
	if err != nil {
		t.Fatalf("insert assistant msg: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO tool_runs (session_id, tool, args, output, time)
		VALUES (?, 'run_command', '{"command":"nvidia-smi"}', 'Power Draw: 120W', datetime('now'))`,
		sessionID,
	)
	if err != nil {
		t.Fatalf("insert tool run: %v", err)
	}

	digest, err := buildSessionsDigest(db, []string{sessionID})
	if err != nil {
		t.Fatalf("buildSessionsDigest: %v", err)
	}

	if !strings.Contains(digest, "เช็คกำลังไฟ GPU") {
		t.Errorf("digest missing user prompt: %s", digest)
	}
	if !strings.Contains(digest, "run_command") {
		t.Errorf("digest missing tool run: %s", digest)
	}
	if !strings.Contains(digest, "nvidia-smi") {
		t.Errorf("digest missing tool args: %s", digest)
	}
}

func TestSynthesizeHabitForSessions_RoutesToSkill(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}

	sess := "test-sess-skill"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'เช็คไฟ GPU', datetime('now'))`, sess)

	fake := &fakeHabitSynthesizer{
		proposal: &HabitProposal{
			Destination: "skill",
			SkillName:   "gpu-monitor",
			Title:       "GPU Power Monitor",
			Body:        "# GPU Monitor Skill\nUse nvidia-smi to monitor power.",
			Reason:      "ผู้ใช้สั่งเช็คกำลังไฟบ่อย",
		},
	}

	ctx := context.Background()
	normKey := "กินไฟ gpu"
	change, err := a.synthesizeHabitForSessions(ctx, fake, []string{sess}, normKey, "")
	if err != nil {
		t.Fatalf("synthesizeHabitForSessions failed: %v", err)
	}
	if change == nil {
		t.Fatalf("expected non-nil change")
	}

	if change.Kind != "skill" {
		t.Errorf("expected Kind == skill, got %q", change.Kind)
	}
	if change.Op != "create" {
		t.Errorf("expected Op == create, got %q", change.Op)
	}
	if change.Scope != "gpu-monitor" {
		t.Errorf("expected Scope == gpu-monitor, got %q", change.Scope)
	}
	if !strings.Contains(change.Body, "GPU Monitor Skill") {
		t.Errorf("expected skill body, got %q", change.Body)
	}
	if !strings.Contains(change.Reason, "GPU Power Monitor") {
		t.Errorf("expected title in reason, got %q", change.Reason)
	}

	// Verify synthesized_habits table recorded the normalized key
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM synthesized_habits WHERE normalized = ?`, normKey).Scan(&count); err != nil || count != 1 {
		t.Errorf("expected 1 row in synthesized_habits for %q, got %d, err: %v", normKey, count, err)
	}
}

// The drafter sees the shelf since 2026-09-14, and a proposal that names an
// installed skill becomes a section appended to it, never a second skill for
// the same work. "aetox" is bundled, so it is on every machine's shelf.
func TestSynthesizeHabitForSessions_ExtendsAnInstalledSkill(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	sess := "test-sess-extends"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'ถามเรื่องแอป', datetime('now'))`, sess)

	fake := &fakeHabitSynthesizer{
		proposal: &HabitProposal{
			Destination: "skill",
			SkillName:   "aetox-faq",
			Extends:     "Aetox", // case-folded like skill_view
			Title:       "คำถามที่ถามซ้ำ",
			Body:        "## คำถามที่พบบ่อย\n- ...",
			Reason:      "ถามซ้ำสามครั้ง",
		},
	}
	change, err := a.synthesizeHabitForSessions(context.Background(), fake, []string{sess}, "ถามเรื่องแอป", "")
	if err != nil {
		t.Fatalf("synthesizeHabitForSessions failed: %v", err)
	}
	if change == nil {
		t.Fatalf("expected a queued change")
	}
	if change.Op != "add" || change.Scope != "aetox" {
		t.Errorf("extending an installed skill must queue add on it, got op=%q scope=%q", change.Op, change.Scope)
	}
	if !strings.Contains(change.Body, "คำถามที่พบบ่อย") {
		t.Errorf("the body is the section to append, got %q", change.Body)
	}
}

// A new skill under a name the shelf already holds could never be applied
// (skill.Apply refuses create on an existing name), so it is not queued.
func TestSynthesizeHabitForSessions_DropsACreateOverAnInstalledName(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	sess := "test-sess-taken"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'x', datetime('now'))`, sess)

	fake := &fakeHabitSynthesizer{
		proposal: &HabitProposal{
			Destination: "skill",
			SkillName:   "aetox",
			Title:       "t",
			Body:        "---\nname: aetox\n---\n# again",
			Reason:      "r",
		},
	}
	change, err := a.synthesizeHabitForSessions(context.Background(), fake, []string{sess}, "x", "")
	if err != nil {
		t.Fatalf("synthesizeHabitForSessions failed: %v", err)
	}
	if change != nil {
		t.Errorf("a create over an installed name must not be queued, got %+v", change)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM pending_changes WHERE kind = 'skill' AND scope = 'aetox'`).Scan(&n)
	if n != 0 {
		t.Errorf("expected no pending row for aetox, got %d", n)
	}
}

// An "extends" naming nothing on the shelf is a hallucinated name; the draft
// falls back to a new skill rather than an add that approval cannot apply.
func TestSynthesizeHabitForSessions_UnknownExtendsDraftsAnew(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	sess := "test-sess-unknown"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'y', datetime('now'))`, sess)

	fake := &fakeHabitSynthesizer{
		proposal: &HabitProposal{
			Destination: "skill",
			SkillName:   "brand-new-thing",
			Extends:     "no-such-skill-on-this-shelf",
			Title:       "t",
			Body:        "---\nname: brand-new-thing\n---\n# new",
			Reason:      "r",
		},
	}
	change, err := a.synthesizeHabitForSessions(context.Background(), fake, []string{sess}, "y", "")
	if err != nil {
		t.Fatalf("synthesizeHabitForSessions failed: %v", err)
	}
	if change == nil || change.Op != "create" || change.Scope != "brand-new-thing" {
		t.Errorf("unknown extends must fall back to create, got %+v", change)
	}
}

func TestSynthesizeHabitForSessions_RoutesToUserProfile(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}

	sess := "test-sess-user"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'ไม่ต้องพูดยาว', datetime('now'))`, sess)

	fake := &fakeHabitSynthesizer{
		proposal: &HabitProposal{
			Destination: "user_profile",
			Title:       "User Preference",
			Body:        "- ผู้ใช้ต้องการคำตอบที่กระชับ ไม่ต้องอธิบายยืดยาว",
			Reason:      "ผู้ใช้ย้ำบ่อยเรื่องความกระชับ",
		},
	}

	ctx := context.Background()
	change, err := a.synthesizeHabitForSessions(ctx, fake, []string{sess}, "", "")
	if err != nil {
		t.Fatalf("synthesizeHabitForSessions failed: %v", err)
	}

	if change.Kind != "memory" {
		t.Errorf("expected Kind == memory, got %q", change.Kind)
	}
	if change.Scope != learned.UserScope {
		t.Errorf("expected Scope == %q, got %q", learned.UserScope, change.Scope)
	}
	if change.Op != learned.OpAdd {
		t.Errorf("expected Op == add, got %q", change.Op)
	}
}

func TestSynthesizeHabitForSessions_RoutesToMachineMemory(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}

	sess := "test-sess-machine"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'เครื่องนี้มี CUDA 12.4 นะ', datetime('now'))`, sess)

	fake := &fakeHabitSynthesizer{
		proposal: &HabitProposal{
			Destination: "machine_memory",
			Title:       "Machine Constraint",
			Body:        "- เครื่องนี้ติดตั้ง CUDA 12.4",
			Reason:      "เป็นข้อจำกัดถาวรของฮาร์ดแวร์/สภาพแวดล้อม",
		},
	}

	ctx := context.Background()
	change, err := a.synthesizeHabitForSessions(ctx, fake, []string{sess}, "", "")
	if err != nil {
		t.Fatalf("synthesizeHabitForSessions failed: %v", err)
	}

	if change.Kind != "memory" {
		t.Errorf("expected Kind == memory, got %q", change.Kind)
	}
	if change.Scope != learned.MainScope {
		t.Errorf("expected Scope == %q (main/machine), got %q", learned.MainScope, change.Scope)
	}
}

func TestSynthesizeHabit_OnDemandSingleSession(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}

	sess := "test-sess-ondemand"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'ทำสรุปรายงานสถิติยอดขาย', datetime('now'))`, sess)

	// Since a.SynthesizeHabit creates an appHabitSynthesizer that calls a model (which is nil or unavailable in unit test),
	// we verify that with fake/valid session, it correctly parses sessionID and calls synthesizeHabitForSessions.
	fake := &fakeHabitSynthesizer{
		proposal: &HabitProposal{
			Destination: "skill",
			SkillName:   "sales-report-summary",
			Title:       "Sales Report Summarizer",
			Body:        "# Sales Report Summary\nAnalyzes and summarizes sales metrics.",
			Reason:      "ผู้ใช้ต้องการสรุปรายงานเป็นประจำ",
		},
	}

	ctx := context.Background()
	change, err := a.synthesizeHabitForSessions(ctx, fake, []string{sess}, "", "ผู้ใช้ขอให้จำงานนี้เป็นสกิล")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if change == nil || change.ID == 0 {
		t.Fatalf("expected persisted PendingChange with ID > 0")
	}

	// Verify change was inserted into pending_changes
	prop := a.PendingChangeByID(change.ID)
	if prop.ID != change.ID {
		t.Errorf("expected pending change found with ID %d", change.ID)
	}
	if prop.Kind != "skill" {
		t.Errorf("expected Kind skill, got %s", prop.Kind)
	}
}

// A drafted skill is read by internal/skilllint before anyone judges it. An
// error sends the drafter back once with the findings in its hint; what is
// still wrong after that rides on the card, in the reason, and the proposal
// is queued anyway: the user is the judge, the linter is the note.
func TestSynthesizeHabitForSessions_LintsTheDraftAndSaysSoOnTheCard(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	sess := "test-sess-lint"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'เช็คไฟ GPU', datetime('now'))`, sess)

	fake := &countingHabitSynthesizer{proposal: &HabitProposal{
		Destination: "skill",
		SkillName:   "gpu-monitor",
		Title:       "GPU Power Monitor",
		Body:        "---\nname: gpu-monitor\ndescription: ตอนผู้ใช้ถามกำลังไฟ GPU\n---\n\nParse subcommand from $ARGUMENTS, then run nvidia-smi.\n",
		Reason:      "ผู้ใช้สั่งเช็คกำลังไฟบ่อย",
	}}
	change, err := a.synthesizeHabitForSessions(context.Background(), fake, []string{sess}, "กินไฟ gpu", "")
	if err != nil {
		t.Fatalf("synthesizeHabitForSessions failed: %v", err)
	}
	if change == nil {
		t.Fatal("a draft with a lint error is still queued for the user to judge")
	}
	if fake.calls != 2 {
		t.Fatalf("the drafter gets exactly one more go after an error, got %d calls", fake.calls)
	}
	if !strings.Contains(fake.hints[1], "dead-door") || !strings.Contains(fake.hints[1], "$ARGUMENTS") {
		t.Fatalf("the retry hint carries the finding, got %q", fake.hints[1])
	}
	if !strings.Contains(change.Reason, "ตัวตรวจสกิล") || !strings.Contains(change.Reason, "dead-door") {
		t.Fatalf("the card names what the linter found, got %q", change.Reason)
	}
	if !strings.Contains(change.Reason, "GPU Power Monitor") {
		t.Fatalf("the title still leads the reason, got %q", change.Reason)
	}
}

// A clean draft costs no second call and carries no note.
func TestSynthesizeHabitForSessions_ACleanDraftIsNotRetried(t *testing.T) {
	a := newJobApp(t)
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}
	sess := "test-sess-lint-clean"
	_, _ = db.Exec(`INSERT INTO sessions (id, project_key, title, created_at, updated_at) VALUES (?, 'p1', 'Test', datetime('now'), datetime('now'))`, sess)
	_, _ = db.Exec(`INSERT INTO messages (session_id, role, text, time) VALUES (?, 'user', 'เช็คไฟ GPU', datetime('now'))`, sess)

	fake := &countingHabitSynthesizer{proposal: &HabitProposal{
		Destination: "skill",
		SkillName:   "gpu-monitor",
		Title:       "GPU Power Monitor",
		Body:        "---\nname: gpu-monitor\ndescription: ตอนผู้ใช้ถามกำลังไฟ GPU\n---\n\nRun `nvidia-smi --query-gpu=power.draw --format=csv`. A driver fault goes to `aetox-debug`.\n",
		Reason:      "ผู้ใช้สั่งเช็คกำลังไฟบ่อย",
	}}
	change, err := a.synthesizeHabitForSessions(context.Background(), fake, []string{sess}, "กินไฟ gpu clean", "")
	if err != nil || change == nil {
		t.Fatalf("clean draft: change=%v err=%v", change, err)
	}
	if fake.calls != 1 {
		t.Fatalf("a clean draft is drafted once, got %d calls", fake.calls)
	}
	if strings.Contains(change.Reason, "ตัวตรวจสกิล") {
		t.Fatalf("a clean draft carries no linter note, got %q", change.Reason)
	}
}

type countingHabitSynthesizer struct {
	proposal *HabitProposal
	calls    int
	hints    []string
}

func (f *countingHabitSynthesizer) Synthesize(_ context.Context, _ string, hint string) (*HabitProposal, error) {
	f.calls++
	f.hints = append(f.hints, hint)
	p := *f.proposal
	return &p, nil
}
