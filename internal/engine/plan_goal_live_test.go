package engine

// วางแผน → มุ่งเป้า → ลงมือ-plans-itself, measured against a real model through the
// real engine, on a toy project in a temp dir so nothing the model writes lands
// anywhere that matters:
//
//	AETOX_LIVE=1 go test ./internal/engine/ -run TestLivePlanThenGoal -v -count=1 -timeout 60m
//
// What it measures is what the owner asked on 14 ก.ย. 2026 — *"เช็คโหมดวางแผนและ
// โหมดมุ่งสู่เป้าหมายด้วยครับ เหมือนเราจะจูนมันไม่ดี"* — as numbers rather than
// impressions: where in the turn `ask_user` fell, how many reads came before
// `plan write`, whether steps were marked as the work went or in one batch at
// the end, how much prose rode beside the tool calls, how often the goal check
// sent the turn back, and whether the run ended in a `plan report` row.
//
// Nothing here asserts a threshold. A live turn against a model is a
// measurement, and the reading is in the log; the one hard failure is a plan
// turn that changed the toy project, which is the promise วางแผน exists to keep.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/safety"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// toyProject writes a tiny Go module the brief is about. Add, Sub and Mul with
// tests; the brief asks for division, which leaves one real scope question open
// (what happens on zero) — the kind a plan should put to the user before it
// reads everything.
func toyProject(t *testing.T) string {
	t.Helper()
	// Not t.TempDir(): the model's own `go test` in the sandbox leaves the
	// build cache holding a file open on Windows for a moment, and TempDir's
	// cleanup then fails the test over a lock the run itself never caused.
	// Best-effort removal, and a directory left behind is a temp dir.
	dir, err := os.MkdirTemp("", "aetox-toycalc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	files := map[string]string{
		"go.mod": "module toycalc\n\ngo 1.22\n",
		"calc.go": "package toycalc\n\n// Add returns a + b.\nfunc Add(a, b int) int { return a + b }\n\n" +
			"// Sub returns a - b.\nfunc Sub(a, b int) int { return a - b }\n\n" +
			"// Mul returns a * b.\nfunc Mul(a, b int) int { return a * b }\n",
		"calc_test.go": "package toycalc\n\nimport \"testing\"\n\n" +
			"func TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"Add\")\n\t}\n}\n\n" +
			"func TestSub(t *testing.T) {\n\tif Sub(5, 3) != 2 {\n\t\tt.Fatal(\"Sub\")\n\t}\n}\n\n" +
			"func TestMul(t *testing.T) {\n\tif Mul(2, 3) != 6 {\n\t\tt.Fatal(\"Mul\")\n\t}\n}\n",
		"README.md": "# toycalc\n\nA calculator with Add, Sub and Mul. Run `go test ./...`.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// snapshot is every file under dir with its bytes, for the "changed nothing"
// check after a plan turn.
func treeSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		b, _ := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		out[rel] = string(b)
		return nil
	})
	return out
}

// turnTrace is one turn read off its parts, the way the transcript audit of
// 11 ก.ย. read the owner's own runs.
type turnTrace struct {
	seq         []string // one token per part: T (text), Td (demoted text), [step] etc.
	tools       int
	texts       int
	textChars   int
	demoted     int
	steps       []int // part index of every `plan step`
	asks        []int // part index of every ask_user
	writes      []int // part index of every `plan write`
	readsBefore int   // tool calls before the first plan write / ask
}

func trace(parts []turn.TurnPart) turnTrace {
	var tr turnTrace
	firstMark := -1
	for i, p := range parts {
		switch p.Kind {
		case turn.PartText:
			tr.texts++
			tr.textChars += len(p.Text)
			if p.Demoted {
				tr.demoted++
				tr.seq = append(tr.seq, "Td")
			} else {
				tr.seq = append(tr.seq, "T")
			}
		case turn.PartAsked:
			tr.seq = append(tr.seq, "<user>")
		case turn.PartTool:
			if p.Tool == nil {
				continue
			}
			tr.tools++
			name := p.Tool.Name
			switch {
			case name == "plan":
				tr.seq = append(tr.seq, "["+p.Tool.Act+"]")
				if p.Tool.Act == "step" {
					tr.steps = append(tr.steps, i)
				}
				if p.Tool.Act == "write" {
					tr.writes = append(tr.writes, i)
					if firstMark < 0 {
						firstMark = tr.tools - 1
					}
				}
			case name == "ask_user":
				tr.seq = append(tr.seq, "ASK")
				tr.asks = append(tr.asks, i)
				if firstMark < 0 {
					firstMark = tr.tools - 1
				}
			default:
				if len(name) > 2 {
					name = name[:2]
				}
				tr.seq = append(tr.seq, name)
			}
		}
	}
	if firstMark < 0 {
		firstMark = tr.tools
	}
	tr.readsBefore = firstMark
	return tr
}

func (tr turnTrace) String() string {
	return fmt.Sprintf("tools=%d texts=%d (%d chars, %d demoted) steps=%d asks=%d writes=%d tools-before-first-ask/write=%d\n    %s",
		tr.tools, tr.texts, tr.textChars, tr.demoted, len(tr.steps), len(tr.asks), len(tr.writes), tr.readsBefore, strings.Join(tr.seq, " "))
}

// stepSpread says whether the step marks were spread through the work or
// batched at the end: the share of tool calls that came AFTER the last
// non-plan tool call, i.e. how much of the run was already over when the
// first step was ticked.
func stepSpread(parts []turn.TurnPart, tr turnTrace) string {
	if len(tr.steps) == 0 {
		return "no step marked at all"
	}
	// Tool index (1-based) of the first step mark, over total tools.
	toolsBeforeFirstStep := 0
	for i, p := range parts {
		if i >= tr.steps[0] {
			break
		}
		if p.Kind == turn.PartTool && p.Tool != nil {
			toolsBeforeFirstStep++
		}
	}
	return fmt.Sprintf("first step marked after %d of %d tool calls (%.0f%% of the run already done)",
		toolsBeforeFirstStep, tr.tools, 100*float64(toolsBeforeFirstStep)/float64(max(tr.tools, 1)))
}

type liveUsage struct{ in, out, cached, rounds int64 }

func usageSince(t *testing.T, db *sql.DB, afterID int64) liveUsage {
	t.Helper()
	var u liveUsage
	rows, err := db.Query(`SELECT prompt_tokens, completion_tokens, cached_prompt_tokens FROM token_usage WHERE id > ?`, afterID)
	if err != nil {
		return u
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var in, out, cached sql.NullInt64
		_ = rows.Scan(&in, &out, &cached)
		u.in += in.Int64
		u.out += out.Int64
		u.cached += cached.Int64
		u.rounds++
	}
	if err := rows.Err(); err != nil {
		t.Logf("token_usage: %v", err)
	}
	return u
}

func lastUsageID(db *sql.DB) int64 {
	var id sql.NullInt64
	_ = db.QueryRow(`SELECT MAX(id) FROM token_usage`).Scan(&id)
	return id.Int64
}

// liveEngine is the team test's boot, with the stance and sandbox this one needs.
func liveEngine(t *testing.T, sandbox string, think string) (*Engine, *recorder) {
	t.Helper()
	key := liveDeepSeekKey(t)
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	screen := &signingScreen{key: key}
	rec := &recorder{}
	a := seed(&Engine{
		ctx:    context.Background(),
		emit:   func(name string, data ...any) { rec.add(emitted{name, data}) },
		dbDir:  t.TempDir(),
		screen: screen,
	}, &conversation{id: newSessionID()})
	bootRecorders.Store(a, rec)
	t.Cleanup(func() {
		bootRecorders.Delete(a)
		if a.db != nil {
			_ = a.db.Close()
		}
	})
	m, _ := mode.Load("coding")
	a.cur().desk = m
	a.applyConfig(a.cur(), config.Config{
		SandboxRoot:     sandbox,
		ModelProvider:   "deepseek",
		ModelName:       "deepseek-v4-flash",
		ModelTimeoutSec: 300,
		ThinkLevel:      think,
		ApprovalMode:    string(safety.ApprovalFullAccess),
	})
	return a, rec
}

// autoAnswer answers every ask_user with the first option (or a canned line)
// after a beat, so a plan that asks is not a plan that waits 30 minutes.
func autoAnswer(a *Engine, rec *recorder, canned string, asked *[]string) {
	prev := a.emit
	a.emit = func(name string, data ...any) {
		prev(name, data...)
		if name != "ask:user" || len(data) == 0 {
			return
		}
		se, ok := data[0].(SessionEvent[map[string]any])
		if !ok {
			return
		}
		q, _ := se.Data["question"].(string)
		opts, _ := se.Data["options"].([]string)
		*asked = append(*asked, fmt.Sprintf("%q options=%v", q, opts))
		answer := canned
		if answer == "" && len(opts) > 0 {
			answer = opts[0]
		}
		go func() {
			time.Sleep(1500 * time.Millisecond)
			a.AnswerUserQuestion(se.SessionID, answer)
		}()
	}
}

func probeTurn(t *testing.T, a *Engine, ask string, limit time.Duration) (TurnReply, time.Duration) {
	t.Helper()
	start := time.Now()
	done := make(chan struct{})
	var reply TurnReply
	var err error
	go func() {
		defer close(done)
		reply, err = a.SendMessage(ask, "")
	}()
	select {
	case <-done:
	case <-time.After(limit):
		t.Logf("turn ran past %s — cancelled", limit)
		a.CancelTurn()
		<-done
	}
	if err != nil {
		t.Logf("turn error: %v", err)
	}
	return reply, time.Since(start)
}

const toyBrief = "เพิ่มการหารให้เครื่องคิดเลขตัวนี้หน่อยครับ ให้ครบเหมือนตัวอื่น"

// longBrief is the run-sized job: enough steps (about eight) that "marked as
// it went" and "marked in one batch at the end" look different, and that a
// sentence per round adds up to something the trace can count.
const longBrief = "ทำให้เครื่องคิดเลขตัวนี้ครบชุดหน่อยครับ: เพิ่ม Div, Mod และ Pow พร้อมเทสต์ของแต่ละตัว, " +
	"เพิ่ม cmd/calc/main.go เป็น CLI รับ `calc <op> <a> <b>` แล้วพิมพ์ผล (มีเทสต์ด้วย), " +
	"และอัปเดต README ให้ตรงของจริงทั้งหมด"

func TestLivePlanThenGoal(t *testing.T) {
	sandbox := toyProject(t)
	before := treeSnapshot(t, sandbox)
	a, rec := liveEngine(t, sandbox, "high")
	var asked []string
	autoAnswer(a, rec, "", &asked)
	db, err := a.database()
	if err != nil {
		t.Fatal(err)
	}
	sid := a.cur().id

	// ---- 1. วางแผน -------------------------------------------------------
	if _, err := a.SetStance(mode.StancePlan.String()); err != nil {
		t.Fatal(err)
	}
	u0 := lastUsageID(db)
	reply, took := probeTurn(t, a, longBrief, 12*time.Minute)
	tr := trace(reply.Parts)
	u := usageSince(t, db, u0)
	t.Logf("PLAN TURN  %s  usage rounds=%d in=%d out=%d cached=%d\n  %s", took.Round(time.Second), u.rounds, u.in, u.out, u.cached, tr)
	t.Logf("PLAN ASKED: %d — %s", len(asked), strings.Join(asked, " | "))
	t.Logf("PLAN REPLY (%d chars): %s", len(reply.Text), reply.Text)
	if after := treeSnapshot(t, sandbox); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Errorf("วางแผน changed the project: before=%v after=%v", fileNames(before), fileNames(after))
	}
	plan, _ := a.loadPlan(sid)
	if plan == nil {
		t.Fatalf("no plan stored after the plan turn — the model typed it instead? reply: %s", reply.Text)
	}
	t.Logf("PLAN STORED v%d title=%q sections=%d steps=%d", plan.Version, plan.Title, len(plan.Sections), len(plan.Steps))
	for _, s := range plan.Sections {
		t.Logf("  ## %s (%d chars)\n%s", s.Heading, len(s.Body), indent(s.Body))
	}
	for _, s := range plan.Steps {
		t.Logf("  step %d: %s", s.N, s.Text)
	}

	// ---- 2. มุ่งเป้า -----------------------------------------------------
	asked = nil
	started := a.StartPlanRun(sid)
	if !started.Started {
		t.Fatalf("run refused: %s", started.Refusal)
	}
	// AETOX_PROBE_NO_BRIEF=1 runs the baseline: the run as it was before the
	// brief was handed over at the press, so the two can be read side by side.
	if os.Getenv("AETOX_PROBE_NO_BRIEF") == "1" {
		a.goalRunSet().get(sid).briefed = true
		t.Log("BASELINE: brief withheld")
	}
	u1 := lastUsageID(db)
	reply2, took2 := probeTurn(t, a, "ลงมือตามแผนนี้เลยครับ", 25*time.Minute)
	tr2 := trace(reply2.Parts)
	u = usageSince(t, db, u1)
	// A finished run is gone from the set, so the count of verdicts is read
	// off the transcript: every text the goal check demoted is one send-back.
	running := a.goalRunSet().get(sid) != nil
	t.Logf("GOAL RUN  %s  usage rounds=%d in=%d out=%d cached=%d  sentBack=%d stillRunning=%v\n  %s", took2.Round(time.Second), u.rounds, u.in, u.out, u.cached, tr2.demoted, running, tr2)
	t.Logf("GOAL STEP SPREAD: %s", stepSpread(reply2.Parts, tr2))
	opened := 0
	for _, p := range reply2.Parts {
		if p.Kind == turn.PartTool && p.Tool != nil && p.Tool.Name == "skill_view" {
			opened++
			t.Logf("  skill_view: %s", p.Tool.Subject)
		}
	}
	t.Logf("GOAL SKILLS OPENED: %d", opened)
	t.Logf("GOAL ASKED: %d — %s", len(asked), strings.Join(asked, " | "))
	t.Logf("GOAL REPLY (%d chars): %s", len(reply2.Text), reply2.Text)
	for i, p := range reply2.Parts {
		if p.Kind == turn.PartText {
			t.Logf("  text[%d] demoted=%v: %s", i, p.Demoted, clip(p.Text, 200))
		}
	}
	plan, _ = a.loadPlan(sid)
	for _, s := range plan.Steps {
		t.Logf("  step %d [%s]: %s — %s", s.N, orTodo(s.State), s.Text, s.Note)
	}
	reports, _ := a.loadPlanReports(sid)
	t.Logf("GOAL REPORTS: %d", len(reports))
	for _, r := range reports {
		b, _ := json.MarshalIndent(r, "  ", " ")
		t.Logf("  %s", b)
	}
	// Did the work actually land?
	t.Logf("GOAL go test in sandbox: %s", goTest(sandbox))
	after := treeSnapshot(t, sandbox)
	t.Logf("GOAL files now: %v", fileNames(after))
}

// The same brief in ลงมือ, fresh conversation: does the model plan on its own
// — `plan write`, steps marked — or just do the job?
func TestLiveActPlansItself(t *testing.T) {
	sandbox := toyProject(t)
	a, rec := liveEngine(t, sandbox, "high")
	var asked []string
	autoAnswer(a, rec, "", &asked)
	db, _ := a.database()
	sid := a.cur().id
	u0 := lastUsageID(db)
	reply, took := probeTurn(t, a, toyBrief, 20*time.Minute)
	tr := trace(reply.Parts)
	u := usageSince(t, db, u0)
	t.Logf("ACT TURN  %s  usage rounds=%d in=%d out=%d cached=%d\n  %s", took.Round(time.Second), u.rounds, u.in, u.out, u.cached, tr)
	t.Logf("ACT ASKED: %d — %s", len(asked), strings.Join(asked, " | "))
	t.Logf("ACT REPLY (%d chars): %s", len(reply.Text), reply.Text)
	for i, p := range reply.Parts {
		if p.Kind == turn.PartText {
			t.Logf("  text[%d] demoted=%v: %s", i, p.Demoted, clip(p.Text, 200))
		}
	}
	plan, _ := a.loadPlan(sid)
	if plan == nil {
		t.Logf("ACT PLAN: none written")
	} else {
		t.Logf("ACT PLAN: v%d steps=%d — %s", plan.Version, len(plan.Steps), stepSpread(reply.Parts, tr))
		for _, s := range plan.Steps {
			t.Logf("  step %d [%s]: %s", s.N, orTodo(s.State), s.Text)
		}
	}
	t.Logf("ACT go test in sandbox: %s", goTest(sandbox))
}

func fileNames(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func indent(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := range lines {
		lines[i] = "      " + lines[i]
	}
	return strings.Join(lines, "\n")
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " / ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func orTodo(state string) string {
	if state == "" {
		return "todo"
	}
	return state
}

func goTest(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "FAIL: " + err.Error() + "\n" + string(out)
	}
	return "ok\n" + string(out)
}
