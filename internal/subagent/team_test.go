package subagent

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// The rules a team lives by (team.go, DECISIONS §256), each pinned where it
// would otherwise drift: the default team is computed and never a file, a
// list names people it does not own, and a member runs under the TEAM's desk.

func writeTeam(t *testing.T, name, body string) string {
	t.Helper()
	dir, err := TeamsDir()
	if err != nil {
		t.Fatalf("TeamsDir: %v", err)
	}
	path := filepath.Join(dir, name, TeamDefinitionFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func teamNamed(list []Team, name string) (Team, bool) {
	for _, tm := range list {
		if tm.Name == name {
			return tm, true
		}
	}
	return Team{}, false
}

// With no team folder at all, the machine behaves as it did before teams: one
// team, every agent on it, sitting in the office.
func TestAMachineWithNoTeamsHasTheDefaultTeamAndEveryoneIsOnIt(t *testing.T) {
	isolate(t)
	writeProfile(t, AgentsDir, "ผู้ช่วยขาย", "---\ndescription: ตอบลูกค้า\n---\nsell")

	teams := Teams()
	if len(teams) != 1 || !teams[0].Default || teams[0].Name != DefaultTeam {
		t.Fatalf("expected the default team alone, got %+v", teams)
	}
	def := teams[0]
	if def.Desk != mode.Office {
		t.Errorf("the default team sits at %q, not the office", def.Desk)
	}
	for _, want := range append([]string{"ผู้ช่วยขาย"}, profileNames(Chairs(mode.Office))...) {
		if !def.Has(want) {
			t.Errorf("%s is not on the default team: %v", want, def.Members)
		}
	}
	if def.Has("explore") {
		t.Error("a helper is on the default team — a team is about colleagues")
	}
}

// A user's agent that a team names leaves the default team; a bundled one
// never does — it is switched off, not removed (owner, 12 ก.ย.).
func TestAnAgentNamedByATeamLeavesTheDefaultTeamUnlessItIsBundled(t *testing.T) {
	isolate(t)
	writeProfile(t, AgentsDir, "reviewer2", "---\ndescription: อ่านโค้ด\n---\nreview")
	writeTeam(t, "ทีมโค้ด", "---\ndesk: coding\nmembers: reviewer2, doc\n---\n")

	def, ok := LoadTeam(DefaultTeam)
	if !ok {
		t.Fatal("the default team did not load")
	}
	if def.Has("reviewer2") {
		t.Error("reviewer2 is on a team of the user's and still on the default team")
	}
	if !def.Has("doc") {
		t.Error("doc left the default team for being named by another — a bundled agent cannot be removed from ทีมผู้ช่วย")
	}
	code, ok := LoadTeam("ทีมโค้ด")
	if !ok {
		t.Fatal("ทีมโค้ด did not load")
	}
	if code.Desk != mode.Coding || !code.Has("reviewer2") || !code.Has("doc") {
		t.Errorf("ทีมโค้ด read wrong: %+v", code)
	}
	// Deleting the team puts the agent back where the rule says it goes.
	if err := DeleteTeam("ทีมโค้ด"); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	if def, _ := LoadTeam(DefaultTeam); !def.Has("reviewer2") {
		t.Error("reviewer2 is on no team after its team was deleted")
	}
}

// A name the file lists that nobody answers to is reported, never dropped
// and never run. Case in a hand-typed name is forgiven and normalised to the
// roster's spelling — the id every other table keys on.
func TestAStaleOrMistypedMemberIsReportedNotSilentlyDropped(t *testing.T) {
	isolate(t)
	writeTeam(t, "เอกสาร", "---\nmembers: Doc, ghost, doc\n---\n")

	tm, ok := LoadTeam("เอกสาร")
	if !ok {
		t.Fatal("the team did not load")
	}
	if !slices.Equal(tm.Members, []string{"doc"}) {
		t.Errorf("members = %v, want [doc] (case normalised, duplicate folded)", tm.Members)
	}
	if !slices.Equal(tm.Missing, []string{"ghost"}) {
		t.Errorf("missing = %v, want [ghost]", tm.Missing)
	}
	if tm.Desk != mode.Office {
		t.Errorf("a team that names no desk sits at %q, not the office", tm.Desk)
	}
	if profiles := tm.MemberProfiles(); len(profiles) != 1 || profiles[0].Name != "doc" {
		t.Errorf("MemberProfiles = %v", profileNames(profiles))
	}
}

// A team at a desk no team may sit at stays on the page with its reason and
// is offered by no picker.
func TestATeamAtTheWrongDeskIsInvalidNotReinterpreted(t *testing.T) {
	isolate(t)
	writeTeam(t, "หลงทาง", "---\ndesk: assistant\nmembers: doc\n---\n")

	tm, ok := teamNamed(Teams(), "หลงทาง")
	if !ok {
		t.Fatal("an invalid team vanished from Teams() — the settings page could not explain it")
	}
	if tm.Invalid == "" || !strings.Contains(tm.Invalid, "desk") {
		t.Errorf("no reason on the invalid team: %+v", tm)
	}
	if _, ok := LoadTeam("หลงทาง"); ok {
		t.Error("LoadTeam handed out a team that cannot be hired from")
	}
	for _, desk := range []string{mode.Office, mode.Coding, "assistant"} {
		if _, ok := teamNamed(TeamsAt(desk), "หลงทาง"); ok {
			t.Errorf("TeamsAt(%s) offers the invalid team", desk)
		}
	}
}

// The write door: validated at the moment the name was typed.
func TestSaveTeamRefusesWhatTheReadWouldOnlyReport(t *testing.T) {
	isolate(t)
	writeProfile(t, AgentsDir, "sales", "---\ndescription: ขาย\n---\nsell")

	if err := SaveTeam("", mode.Office, "", []string{"doc"}); err == nil {
		t.Error("a nameless team was saved")
	}
	if err := SaveTeam(DefaultTeam, mode.Office, "", nil); err == nil {
		t.Error("the default team took a save — it has no file and cannot be edited")
	}
	if err := SaveTeam("a b", mode.Office, "", nil); err == nil {
		t.Error("a name with a space was saved")
	}
	if err := SaveTeam("ทีม", "assistant", "", []string{"doc"}); err == nil {
		t.Error("a team at the assistant desk was saved")
	}
	if err := SaveTeam("ทีม", mode.Coding, "", []string{"doc", "nobody"}); err == nil {
		t.Error("a member that does not exist was saved")
	}
	if err := SaveTeam("ทีม", mode.Coding, "", []string{"explore"}); err == nil {
		t.Error("a helper was put on a team")
	}
	if err := SaveTeam("ทีม", mode.Coding, "งานโค้ด\nสองบรรทัด", []string{"Sales", "doc", "sales"}); err != nil {
		t.Fatalf("SaveTeam: %v", err)
	}
	tm, ok := LoadTeam("ทีม")
	if !ok {
		t.Fatal("the saved team did not load")
	}
	if tm.Desk != mode.Coding || tm.Description != "งานโค้ด สองบรรทัด" {
		t.Errorf("saved team read back as %+v", tm)
	}
	if !slices.Equal(tm.Members, []string{"sales", "doc"}) {
		t.Errorf("members = %v, want [sales doc] in the order given, spelt as the roster spells them", tm.Members)
	}
}

// The whole point, at the tool: a coding-desk session with a coding team
// hires the member under the coding ceiling — shell included — while the
// same session refuses an agent the team does not name, and an office team
// hired from the assistant desk still runs under the office.
func TestATeamMemberRunsUnderTheTeamsDesk(t *testing.T) {
	isolate(t)
	writeProfile(t, AgentsDir, "fixer", "---\ndescription: แก้โค้ด\n---\nfix")
	writeTeam(t, "ทีมโค้ด", "---\ndesk: coding\nmembers: fixer\n---\n")
	code, _ := LoadTeam("ทีมโค้ด")
	coding, _ := mode.Load(mode.Coding)
	assistant, _ := mode.Load("assistant")
	fixer, _ := Load("fixer")
	doc, _ := Load("doc")

	tool := taskToolOf(t, TaskOptions{Desk: coding, Team: &code})
	ceiling, err := tool.reach(fixer)
	if err != nil {
		t.Fatalf("a coding team's member is refused from the coding desk: %v", err)
	}
	if ceiling.DeskName() != mode.Coding {
		t.Errorf("the member runs under %q, want the team's desk", ceiling.DeskName())
	}
	if !ceiling.Carries("shell", skill.SourceBuiltin) {
		t.Error("the coding ceiling does not carry shell — §94.2 says a coding-desk agent holds one")
	}
	if _, err := tool.reach(doc); err == nil || !strings.Contains(err.Error(), "team") {
		t.Errorf("an agent the team does not name was reached, or refused for the wrong reason: %v", err)
	}
	if names := profileNames(tool.available()); !slices.Contains(names, "fixer") || slices.Contains(names, "doc") {
		t.Errorf("the roster offered %v — it must be the team and the helpers, nobody else", names)
	}
	// Helpers are not the team's business.
	if explore, ok := Load("explore"); ok {
		if _, err := tool.reach(explore); err != nil {
			t.Errorf("a helper is refused on a team session: %v", err)
		}
	}

	// The default team, hired from the assistant desk: the office ceiling, as
	// before teams existed.
	def, _ := LoadTeam(DefaultTeam)
	tool = taskToolOf(t, TaskOptions{Desk: assistant, Team: &def})
	ceiling, err = tool.reach(doc)
	if err != nil {
		t.Fatalf("doc refused from the assistant desk on the default team: %v", err)
	}
	if ceiling.DeskName() != mode.Office {
		t.Errorf("doc runs under %q from the assistant desk, want the office", ceiling.DeskName())
	}
	if ceiling.Carries("shell", skill.SourceBuiltin) {
		t.Error("an office team member holds shell — the office ceiling did not apply")
	}

	// And the coding desk cannot reach an office team: the star has one center.
	tool = taskToolOf(t, TaskOptions{Desk: coding, Team: &def})
	if _, err := tool.reach(doc); err == nil {
		t.Error("the coding desk hired from an office team — §84 crossed")
	}

	// No team at all is the reach every host had before: doc from the
	// assistant desk, under the office.
	tool = taskToolOf(t, TaskOptions{Desk: assistant})
	if ceiling, err := tool.reach(doc); err != nil || ceiling.DeskName() != mode.Office {
		t.Errorf("nil team changed the old reach: %v %v", ceiling.DeskName(), err)
	}
}
