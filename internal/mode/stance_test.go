package mode

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// The zero value has to be the default, because three separate things rely on
// it saying so without translating: the sessions.stance column default, a
// caller that never sets bootstrap.Options.Stance, and every session written
// before the column existed.
func TestTheZeroStanceIsTheOneThatChangesNothing(t *testing.T) {
	var s Stance
	if s != StanceAct {
		t.Fatalf("the zero Stance must be StanceAct, got %q", s)
	}
	if s.String() != "" {
		t.Errorf("StanceAct must store as the empty string, got %q", s.String())
	}
	if !s.AllowsTool("write") {
		t.Error("ลงมือ must leave every tool on the desk — it is today's behaviour, unchanged")
	}
	if s.Direction() != "" {
		t.Error("ลงมือ must add nothing to the prompt: it changes nothing, so it has nothing to say")
	}
}

func TestConsultCarriesNoToolDefinitions(t *testing.T) {
	s := StanceConsult
	for _, name := range []string{"write", "edit", "shell", "browser", "calc", "task", "ask_user", "memory"} {
		if s.AllowsTool(name) {
			t.Errorf("คู่คิด must carry no tools; it kept %q", name)
		}
	}
	for _, source := range []skill.Source{skill.SourceBuiltin, skill.SourceWorkbench, skill.SourceMCP} {
		if s.Carries("anything", source) {
			t.Errorf("คู่คิด must carry nothing from source %v", source)
		}
	}
	if s.Direction() == "" {
		t.Error("a stance that changes the engine has to say so in the prompt")
	}
}

// The one thing no stance takes away. A skill contributes no tool definition to
// begin with (Dispatcher.ToolDefinitions skips SourceSkill), so withholding it
// would buy nothing — and it would break the `/skill-name` command the user
// typed on purpose. Same sentence Mode.Carries already holds.
func TestNoStanceTakesASkillAway(t *testing.T) {
	for _, s := range Stances() {
		if !s.Carries("tax-invoice", skill.SourceSkill) {
			t.Errorf("stance %q dropped a skill: a skill costs no tokens and /skill-name must work everywhere", s)
		}
	}
}

// A stance may only subtract. Nothing in the type can express "put this back",
// and this is the test that says so out loud — the property COMPANY.md §6.3
// rests on once the dial can move mid-session.
func TestEveryStanceIsASubsetOfTheDefault(t *testing.T) {
	names := []string{"write", "edit", "shell", "browser", "calc", "task", "diagnostics"}
	for _, s := range Stances() {
		for _, name := range names {
			if s.AllowsTool(name) && !StanceAct.AllowsTool(name) {
				t.Fatalf("stance %q handed back %q — a stance must never widen a desk", s, name)
			}
		}
	}
}

// A name from a later build — a session left in วางแผน and reopened after a
// downgrade — must come back as ลงมือ. Failing the other way opens a
// conversation that silently carries nothing and gives no reason why.
func TestAnUnknownStanceFallsBackToTheOneThatWithholdsNothing(t *testing.T) {
	for _, name := range []string{"goal", "ผิด", "CONSULT-ish", "  "} {
		if got := NormalizeStance(name); got != StanceAct {
			t.Errorf("NormalizeStance(%q) = %q, want ลงมือ", name, got)
		}
	}
	if got := NormalizeStance("  CONSULT "); got != StanceConsult {
		t.Errorf("NormalizeStance must trim and lowercase a real name, got %q", got)
	}
	if got := NormalizeStance("plan"); got != StancePlan {
		t.Errorf("วางแผน is implemented now and must survive normalization, got %q", got)
	}
}

// วางแผน is the stance that looks at everything and changes nothing, so the
// split has to hold in both directions: every reading tool kept, every writing,
// running or delegating one gone.
func TestPlanKeepsWhatOnlyLooksAndDropsWhatChanges(t *testing.T) {
	s := StancePlan
	for _, name := range []string{
		"read", "list", "glob", "grep", "web_search", "web_fetch",
		"pdf_read", "image_ocr", "diagnostics", "symbol", "github",
		"n8n_workflow_read", "todo_write", "ask_user", "calc", "skills_list",
	} {
		if !s.AllowsTool(name) {
			t.Errorf("วางแผน dropped %q — it only reads, and a plan written without looking is a guess", name)
		}
	}
	for _, name := range []string{
		"write", "edit", "edits", "delete", "notebook_edit",
		"shell", "shell_kill", "git", "desk_terminal",
		"doc_write", "sheet_write",
		"n8n_workflow_create", "n8n_server_start", "windmill_flow_update",
		"task", "plugin_install", "memory", "browser",
	} {
		if s.AllowsTool(name) {
			t.Errorf("วางแผน kept %q — the whole promise of this stance is that it changes nothing", name)
		}
	}
}

// An allow-list, and this is the test that says why. A tool nobody has
// classified must be withheld here, so that adding one next month cannot
// quietly hand a writing tool to the mode that promises not to write.
func TestPlanWithholdsAToolNobodyHasClassified(t *testing.T) {
	if StancePlan.AllowsTool("some_tool_added_next_month") {
		t.Error("วางแผน must fail closed on an unknown tool — a deny-list would fail the other way, silently")
	}
}

// An MCP tool is withheld without being looked at: `jira_search` and
// `jira_create_issue` are the same kind of string to this package, so there is
// no honest way to tell them apart from here.
func TestPlanWithholdsMCPBecauseItCannotTellReadFromWrite(t *testing.T) {
	if StancePlan.Carries("jira_search", skill.SourceMCP) {
		t.Error("วางแผน must withhold MCP: a promise this code cannot keep is worse than a missing tool")
	}
	if !StanceAct.Carries("jira_search", skill.SourceMCP) {
		t.Error("ลงมือ must still carry MCP — it withholds nothing")
	}
}

// ลงมือ leads because it is the way back: a control you can enter and cannot
// leave is not a switch (§106.3).
func TestThePickerOpensWithTheWayBack(t *testing.T) {
	all := Stances()
	if len(all) < 2 {
		t.Fatalf("a switch needs somewhere to go and somewhere to return to, got %d", len(all))
	}
	if all[0] != StanceAct {
		t.Errorf("ลงมือ must be first in picker order, got %q", all[0])
	}
	// Stances() must hand back a copy — a caller that sorts or truncates the
	// result must not be able to edit the set every session is normalized
	// against.
	all[0] = "tampered"
	if Stances()[0] != StanceAct {
		t.Error("Stances() leaked its backing array; a caller can now rewrite the set")
	}
}

// §44.0 and COMPANY.md §6.1: a manifest says what the work is, never who the
// assistant is. A stance is a manifest with no file, and is held to the line
// the same way.
func TestAStanceDirectionDescribesTheWorkAndNotTheWorker(t *testing.T) {
	for _, s := range Stances() {
		d := strings.ToLower(s.Direction())
		if d == "" {
			continue
		}
		for _, forbidden := range []string{"you are a", "you are an", "act as", "your persona", "your role is"} {
			if strings.Contains(d, forbidden) {
				t.Errorf("stance %q says %q — that is identity, and identity has one home (§44.0)", s, forbidden)
			}
		}
	}
}

// วางแผน never asked the user anything, and the owner paid for that in whole
// turns: a plan built around a guess, then rebuilt once the guess was named
// (2026-09-08).
//
// Four things were suppressing the question at once, and this test holds the
// two that live in this file. `ask_user` was on the allow-list the whole time —
// what was missing was any sentence telling the stance it had it, and one at
// the end telling it not to end a turn with a question, which was meant about
// permission and read as a rule about questions.
func TestPlanAsksAboutTheWorkBeforeItReads(t *testing.T) {
	d := StancePlan.Direction()
	if !strings.Contains(d, "ask_user") {
		t.Error("the plan direction never names the tool it is meant to ask with")
	}
	// The order is the mechanism, not the manners. Asked after the reading, a
	// question arrives with the budget already spent on the wrong branch.
	ask := strings.Index(d, "BEFORE the deep reading")
	if ask < 0 {
		t.Fatal("nothing tells the stance to ask before it reads")
	}
	// Anchored on the first sentence that demands a plan. What the test is about
	// is the ORDER, so it moves with the wording rather than pinning it — and
	// the wording stays surface-agnostic on purpose: HOW a plan reaches the
	// screen (a `plan` tool where there is one, a fence where there is not) is
	// internal/prompt.planCard's question, never this one (§106.12).
	if shape := strings.Index(d, "Give the plan under these headings"); shape < ask {
		t.Error("the plan is demanded before the question is invited — the question then arrives too late to change anything")
	}
	// The sentence that used to close this direction. It said "End with the
	// plan, not with a question", meaning do not beg to leave the stance, and a
	// model in the last-and-loudest position of a prompt read it as "never ask".
	if strings.Contains(d, "not with a question") {
		t.Error("the closing sentence forbids questions again — say permission, and say it about permission only")
	}
}

// A plan that already exists is amended, not written out again from the top.
//
// Every turn in this stance used to be turn one: the direction demanded a whole
// plan under every heading with no branch for "this one already exists", so a
// user correcting a single step paid for the entire document a second time,
// plus the reading that produced it.
func TestPlanAmendsThePlanItAlreadyHas(t *testing.T) {
	d := strings.ToLower(StancePlan.Direction())
	for _, want := range []string{"amends it", "already read"} {
		if !strings.Contains(d, want) {
			t.Errorf("the plan direction says nothing about %q — a revision costs a whole rewrite without it", want)
		}
	}
}

// A plan whose finish cannot be checked is a wish with numbered steps — and it
// is what มุ่งเป้า would have to aim at, so the heading is load-bearing twice.
func TestThePlanSaysHowItWillBeKnownToHaveWorked(t *testing.T) {
	var found bool
	for _, h := range PlanHeadings() {
		if strings.Contains(strings.ToLower(h), "know it worked") {
			found = true
		}
	}
	if !found {
		t.Errorf("no heading asks for a finish condition: %v", PlanHeadings())
	}
	// The prompt is built from the same list, so the two cannot drift inside
	// this file — but only while the direction actually renders it.
	d := StancePlan.Direction()
	for _, h := range PlanHeadings() {
		if !strings.Contains(d, h) {
			t.Errorf("heading %q is in the shape and not in the prompt", h)
		}
	}
}
