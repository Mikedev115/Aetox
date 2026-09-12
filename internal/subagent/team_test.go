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

// A fresh machine gets one team written for it — ผู้ช่วยในคอมพิวเตอร์, the
// four shipped errands — and nobody else is on any team: the rest of the
// roster are ordinary specialists (owner, 13 ก.ย.: "ตัวอื่นให้เป็นเอเจนเฉพาะทาง
// ธรรมดา").
// The seed was ทีมเอเจน for a day (13 ก.ย.). A home that still has it under
// that name is renamed once — members and desk intact — and a home that
// already has a team of the new name is not touched.
func TestTheSeedsOldNameIsRenamedOnce(t *testing.T) {
	isolate(t)
	if err := SaveTeam(LegacySeedTeamName, mode.Office, "ของเก่า", []string{"doc", "sheet"}); err != nil {
		t.Fatal(err)
	}
	teams := Teams()
	if len(teams) != 1 || teams[0].Name != SeedTeamName {
		t.Fatalf("expected the old seed renamed to %q, got %+v", SeedTeamName, teams)
	}
	if !slices.Equal(teams[0].Members, []string{"doc", "sheet"}) || teams[0].Description != "ของเก่า" {
		t.Errorf("the rename lost the file: %+v", teams[0])
	}
	if _, ok := LoadTeam(LegacySeedTeamName); ok {
		t.Error("the old folder is still there")
	}

	// Both names present: nothing moves.
	isolate(t)
	if err := SaveTeam(SeedTeamName, mode.Office, "ใหม่", []string{"doc"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveTeam(LegacySeedTeamName, mode.Office, "เก่า", []string{"sheet"}); err != nil {
		t.Fatal(err)
	}
	if got := Teams(); len(got) != 2 {
		t.Fatalf("expected both teams kept, got %+v", got)
	}
}

func TestAFreshMachineIsSeededWithOneTeamOfFour(t *testing.T) {
	isolate(t)
	writeProfile(t, AgentsDir, "ผู้ช่วยขาย", "---\ndescription: ตอบลูกค้า\n---\nsell")

	teams := Teams()
	if len(teams) != 1 || teams[0].Name != SeedTeamName {
		t.Fatalf("expected the seeded team alone, got %+v", teams)
	}
	seed := teams[0]
	if seed.Desk != mode.Office {
		t.Errorf("the seed sits at %q, not the office", seed.Desk)
	}
	if !slices.Equal(seed.Members, seedMembers) {
		t.Errorf("the seed names %v, want %v", seed.Members, seedMembers)
	}
	for _, outside := range []string{"ผู้ช่วยขาย", "automation", "github", "editor", "explore"} {
		if seed.Has(outside) {
			t.Errorf("%s is on the seeded team — it should be an ordinary specialist", outside)
		}
	}
	if seed.Path == "" {
		t.Error("the seed is not a file — it has to be editable and deletable like any team")
	}
	if PreferredTeam(mode.Office) != SeedTeamName || PreferredTeam(mode.Coding) != NoTeam {
		t.Errorf("preferred: office %q coding %q", PreferredTeam(mode.Office), PreferredTeam(mode.Coding))
	}
}

// The seed is a team like any other: edited, it stays edited; deleted, it
// stays deleted — the home's existence is what says "seeded already".
func TestTheSeedIsAnOrdinaryTeamOnceWritten(t *testing.T) {
	isolate(t)
	Teams() // seeds
	if err := SaveTeam(SeedTeamName, mode.Office, "เหลือสองคน", []string{"doc", "sheet"}); err != nil {
		t.Fatalf("SaveTeam(seed): %v", err)
	}
	if tm, ok := LoadTeam(SeedTeamName); !ok || !slices.Equal(tm.Members, []string{"doc", "sheet"}) {
		t.Errorf("the edited seed read back as %+v", tm)
	}
	if err := DeleteTeam(SeedTeamName); err != nil {
		t.Fatalf("DeleteTeam(seed): %v", err)
	}
	if len(Teams()) != 0 {
		t.Errorf("the deleted seed came back: %+v", Teams())
	}
	if PreferredTeam(mode.Office) != NoTeam {
		t.Error("a machine with no team still prefers one")
	}
	if _, ok := LoadTeam(NoTeam); ok {
		t.Error("NoTeam loaded as a team")
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
	// doc is not on the team AND sits at a desk the coding desk never hands
	// to, so the refusal is the cross-desk one — the model's right next move
	// is "this belongs in another kind of session", not "add doc to the team".
	if _, err := tool.reach(doc); err == nil || !strings.Contains(err.Error(), "does not hand work") {
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
	// The seed, written by hand here: the teams home already exists (the
	// coding team above), so nothing seeds it.
	if err := SaveTeam(SeedTeamName, mode.Office, "", seedMembers); err != nil {
		t.Fatal(err)
	}
	def, _ := LoadTeam(SeedTeamName)
	tool = taskToolOf(t, TaskOptions{Desk: assistant, Team: &def})
	ceiling, err = tool.reach(doc)
	if err != nil {
		t.Fatalf("doc refused from the assistant desk on the seeded team: %v", err)
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

	// A chat on NO team — an empty roster, which is what the desktop hands
	// over for one — hires no colleague, and says where a team is chosen.
	none := Team{Name: NoTeam, Members: []string{}}
	tool = taskToolOf(t, TaskOptions{Desk: assistant, Team: &none})
	if _, err := tool.reach(doc); err == nil || !strings.Contains(err.Error(), "no team") {
		t.Errorf("a chat on no team hired doc, or refused for the wrong reason: %v", err)
	}
	if explore, ok := Load("explore"); ok {
		if _, err := tool.reach(explore); err != nil {
			t.Errorf("a chat on no team lost its helpers: %v", err)
		}
	}
}
