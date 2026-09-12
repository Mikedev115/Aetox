package main

// A team at the desktop (§256): the session's third coordinate, and what it
// changes about a real engine — who `task` reaches, which desk a chair chat
// sits at, and which pair of switches the reach reads.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/subagent"
)

// hire writes a user agent and a team naming it at the given desk, in the
// shapes the real editor and team page write.
func hire(t *testing.T, agent, team, desk string) {
	t.Helper()
	path, err := config.AgentDefinitionPath(agent)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\ndescription: ทดสอบ\n---\nwork"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := subagent.SaveTeam(team, desk, "", []string{agent}); err != nil {
		t.Fatalf("SaveTeam: %v", err)
	}
}

// A coding-desk session on a coding team offers `task` the team's member and
// nobody from the office; a session on the default team at the coding desk
// has no colleague to hand to at all — the star did not move.
func TestACodingTeamIsHiredFromTheCodingDeskAndOnlyThere(t *testing.T) {
	a := bootDeskApp(t, "")
	hire(t, "fixer", "ทีมโค้ด", mode.Coding)

	id, err := a.NewTeamSession(mode.Coding, "ทีมโค้ด")
	if err != nil {
		t.Fatalf("NewTeamSession: %v", err)
	}
	if a.cur().team != "ทีมโค้ด" || a.SessionTeam(id) != "ทีมโค้ด" {
		t.Fatalf("the session is not on its team: %q / %q", a.cur().team, a.SessionTeam(id))
	}
	if !slices.Contains(toolNames(a), "task") {
		t.Fatal("a coding session with a team carries no task")
	}
	// The reach the roster in the schema promises, read off the same tool
	// the model would call.
	agents := taskAgentEnum(t, a)
	if !slices.Contains(agents, "fixer") {
		t.Errorf("the coding team's member is not offered: %v", agents)
	}
	if slices.Contains(agents, "doc") {
		t.Errorf("an office agent is offered at the coding desk: %v", agents)
	}

	// No team at the coding desk: no roster, not an error — and nobody hired.
	if _, err := a.NewTeamSession(mode.Coding, subagent.NoTeam); err != nil {
		t.Fatalf("no team at the coding desk refused — it is 'no roster', not an error: %v", err)
	}
	for _, n := range taskAgentEnum(t, a) {
		if p, ok := subagent.Load(n); ok && p.Desk != "" {
			t.Errorf("the coding desk on no team can hire %s — §84 crossed", n)
		}
	}
}

// A chair sits at the coding desk only through a team at that desk, and the
// engine it gets holds what that desk holds — the first shell an agent has
// ever carried (§94.2). Reopening the session after the team is gone refuses,
// as it does for a deleted profile (§85).
func TestAChairAtTheCodingDeskNeedsItsTeamAndHoldsTheDesksTools(t *testing.T) {
	a := bootDeskApp(t, "")
	hire(t, "fixer", "ทีมโค้ด", mode.Coding)

	if _, err := a.NewChairSessionAt(mode.Coding, "doc", ""); err == nil {
		t.Error("an office agent no coding team names sat at the coding desk")
	}
	id, err := a.NewChairSessionAt(mode.Coding, "fixer", "")
	if err != nil {
		t.Fatalf("NewChairSessionAt(coding, fixer): %v", err)
	}
	if a.cur().chair != "fixer" || a.cur().desk.DeskName() != mode.Coding {
		t.Fatalf("seated at %q as %q", a.cur().desk.DeskName(), a.cur().chair)
	}
	got := toolNames(a)
	if !slices.Contains(got, "shell") {
		t.Errorf("a coding-team chair holds no shell — the ceiling is not the coding desk's: %v", got)
	}
	if slices.Contains(got, "doc_write") {
		t.Errorf("a coding-team chair carries the office's writers: %v", got)
	}
	sys := a.cur().agent.ContextMessages()[0].Content
	if !strings.Contains(sys, "work") {
		t.Error("the chair chat does not run on the chair's own prompt")
	}
	// A turn, so the session has a row to reopen from below.
	a.appendTurn(a.cur(),
		SessionMessage{Role: "user", Text: "x", Time: "00:00"},
		SessionMessage{Role: "agent", Text: "y", Time: "00:00"})

	// The same chair, opened from the office, is under the office ceiling.
	if _, err := a.NewChairSession("fixer"); err != nil {
		t.Fatalf("NewChairSession(fixer) at the office: %v", err)
	}
	if got := toolNames(a); slices.Contains(got, "diagnostics") {
		t.Errorf("an office chair chat carries the code group: %v", got)
	}

	// Gone team, refused reopen.
	if err := subagent.DeleteTeam("ทีมโค้ด"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoadSession(id); err == nil || !strings.Contains(err.Error(), "ทีม") {
		t.Errorf("a coding chair session whose team is gone reopened, or refused for the wrong reason: %v", err)
	}
}

// A new chat keeps the team the window was on when the desk can reach it,
// and drops to the default when it cannot — a blank page must never be
// refused for a roster the click did not ask about.
func TestANewChatCarriesTheTeamOnlyWhereItFits(t *testing.T) {
	a := bootDeskApp(t, "")
	hire(t, "fixer", "ทีมโค้ด", mode.Coding)
	if _, err := a.NewTeamSession(mode.Coding, "ทีมโค้ด"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.NewSessionAt(mode.Coding); err != nil || a.cur().team != "ทีมโค้ด" {
		t.Errorf("a new coding chat dropped its team: team=%q err=%v", a.cur().team, err)
	}
	// The assistant desk cannot carry a coding team, so the chat lands on the
	// desk's preferred team: the seeded ผู้ช่วยในคอมพิวเตอร์, which bootDeskApp's fresh
	// data root was given on its first roster read.
	if _, err := a.NewSessionAt("assistant"); err != nil || a.cur().team != subagent.SeedTeamName {
		t.Errorf("a new assistant chat did not land on the seeded team: team=%q err=%v", a.cur().team, err)
	}
	if _, err := a.NewTeamSession("assistant", "ทีมโค้ด"); err == nil {
		t.Error("the assistant desk opened a session on a coding team")
	}
	if _, err := a.NewTeamSession(mode.Coding, "no-such-team"); err == nil {
		t.Error("an unknown team opened a session")
	}
}

// The two door switches are the doors', and a member's reach is the team's:
// flipping a member off on one team leaves it in reach on another (12 ก.ย.:
// "จะไม่เหมารวมกันนะ"), the code door's switch moves nothing on the
// assistant's, and neither team has a master switch of its own (13 ก.ย.:
// "ทำเป็น 2 สวิตช์ข้างบน แยกฝั่งผู้ช่วยและฝั่งโค้ด").
func TestSwitchesArePerDoorAndMembersPerTeam(t *testing.T) {
	a := bootDeskApp(t, "")
	hire(t, "fixer", "ทีมโค้ด", mode.Coding)
	if err := subagent.SaveTeam("ทีมเอกสาร", mode.Office, "", []string{"fixer", "doc"}); err != nil {
		t.Fatal(err)
	}

	code := a.DelegateSwitches("ทีมโค้ด")
	if code.Team != "ทีมโค้ด" || code.Code.Off || code.Agents.Off {
		t.Errorf("a fresh machine has a door switched off: %+v / %+v", code.Agents, code.Code)
	}
	names := func(s DelegateSettings) []string {
		out := []string{}
		for _, w := range s.Agents.Workers {
			out = append(out, w.Name)
		}
		return out
	}
	if got := names(code); !slices.Equal(got, []string{"fixer"}) {
		t.Errorf("the coding team's rows are %v, want its member alone", got)
	}
	if got := names(a.DelegateSwitches(subagent.NoTeam)); len(got) != 0 {
		t.Errorf("no team has rows: %v", got)
	}

	docs := a.SetAgentOff("ทีมเอกสาร", "fixer", true)
	on := func(s DelegateSettings, name string) bool {
		for _, w := range s.Agents.Workers {
			if w.Name == name {
				return w.On
			}
		}
		return false
	}
	if on(docs, "fixer") {
		t.Error("fixer is still in reach on ทีมเอกสาร after being switched off there")
	}
	if !on(a.DelegateSwitches("ทีมโค้ด"), "fixer") {
		t.Error("switching fixer off on ทีมเอกสาร switched it off on ทีมโค้ด too — the switches were lumped together")
	}
	if a.cur().cfg.DelegateSet {
		t.Error("a team's member switch set DelegateSet — that flag guards the shipped door default, which a team's list is not")
	}

	// The code door's switch, and only the code door's.
	after := a.SetDelegateOff("code", true)
	if !after.Code.Off {
		t.Error("the code door's switch did not take")
	}
	if after.Agents.Off {
		t.Error("switching the code door off switched the assistant's door off")
	}
	// A coding session on a coding team now hands to nobody, and an assistant
	// session on an office team still does.
	if _, err := a.NewTeamSession(mode.Coding, "ทีมโค้ด"); err != nil {
		t.Fatal(err)
	}
	if got := taskAgentEnum(t, a); slices.Contains(got, "fixer") {
		t.Errorf("the code door is switched off and still offers %v", got)
	}
	if _, err := a.NewTeamSession("assistant", "ทีมเอกสาร"); err != nil {
		t.Fatal(err)
	}
	if got := taskAgentEnum(t, a); !slices.Contains(got, "doc") {
		t.Errorf("the assistant door is on and offers %v", got)
	}
	// The helpers half is the session's, whichever door.
	a.SetDelegateOff("helpers", true)
	if !a.DelegateSwitches("ทีมโค้ด").Helpers.Off {
		t.Error("the helpers switch is per team — hands are not on a roster")
	}
}

// A chat opened on no team on purpose (the picker's "ไม่ใช้ทีมช่วย") reopens
// on no team — and the rows from before teams existed, which say the same
// thing and meant the opposite, are put on the seeded team once by the
// migration, not on reopen: a chat that could hire everyone must not come
// back able to hire nobody, and a chat that chose nobody must not come back
// hiring the seed.
func TestNoTeamIsAChoiceAndPreTeamRowsJoinTheSeedOnce(t *testing.T) {
	a := bootDeskApp(t, "")
	if _, err := a.NewTeamSession("assistant", subagent.NoTeam); err != nil {
		t.Fatal(err)
	}
	id := a.cur().id
	a.appendTurn(a.cur(),
		SessionMessage{Role: "user", Text: "x", Time: "00:00"},
		SessionMessage{Role: "agent", Text: "y", Time: "00:00"})
	if a.SessionTeam(id) != subagent.NoTeam {
		t.Fatalf("the row was born with a team: %q", a.SessionTeam(id))
	}
	if _, err := a.NewSessionAt("assistant"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoadSession(id); err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if a.cur().team != subagent.NoTeam || a.SessionTeam(id) != subagent.NoTeam {
		t.Errorf("reopened on %q, row says %q — a chosen no-team must stay one", a.cur().team, a.SessionTeam(id))
	}

	// The same row, as a pre-team row would be: the migration step moves it.
	db, err := a.database()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := preTeamRowsJoinTheSeed(tx); err != nil {
		t.Fatalf("migration: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.SessionTeam(id) != subagent.SeedTeamName {
		t.Errorf("the migration left the row on %q, want the seed", a.SessionTeam(id))
	}
	// The migration runs at startup, before any chat is live; a reopen then
	// reads the row. Here the chat is still held, so it is let go first.
	a.convs.forget(id)
	if _, err := a.LoadSession(id); err != nil {
		t.Fatalf("LoadSession after the move: %v", err)
	}
	if a.cur().team != subagent.SeedTeamName {
		t.Errorf("reopened on %q after the move, want the seed", a.cur().team)
	}
}

// taskAgentEnum is the `agent` enum of the session's task tool — the roster
// the model is actually offered — decoded from the schema it would be sent.
func taskAgentEnum(t *testing.T, a *App) []string {
	t.Helper()
	for _, d := range a.deskTools().ToolDefinitions() {
		if d.Function.Name != "task" {
			continue
		}
		var schema struct {
			Properties struct {
				Agent struct {
					Enum []string `json:"enum"`
				} `json:"agent"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(d.Function.Parameters, &schema); err != nil {
			t.Fatalf("task schema: %v", err)
		}
		return schema.Properties.Agent.Enum
	}
	return nil
}
