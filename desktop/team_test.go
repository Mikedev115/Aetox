package main

// A team at the desktop (§251): the session's third coordinate, and what it
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

	// An office team cannot be carried into the coding desk.
	if _, err := a.NewTeamSession(mode.Coding, subagent.DefaultTeam); err != nil {
		t.Fatalf("the default team at the coding desk refused — it is 'no roster', not an error: %v", err)
	}
	for _, n := range taskAgentEnum(t, a) {
		if p, ok := subagent.Load(n); ok && p.Desk != "" {
			t.Errorf("the coding desk on the default team can hire %s — §84 crossed", n)
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
	if _, err := a.NewSessionAt("assistant"); err != nil || a.cur().team != subagent.DefaultTeam {
		t.Errorf("a new assistant chat carried a coding team: team=%q err=%v", a.cur().team, err)
	}
	if _, err := a.NewTeamSession("assistant", "ทีมโค้ด"); err == nil {
		t.Error("the assistant desk opened a session on a coding team")
	}
	if _, err := a.NewTeamSession(mode.Coding, "no-such-team"); err == nil {
		t.Error("an unknown team opened a session")
	}
}

// Each team's switches are its own: flipping a member off on one team leaves
// it in reach on another, and a user team starts with delegation on while
// the default keeps whatever it had.
func TestSwitchesAreKeptPerTeam(t *testing.T) {
	a := bootDeskApp(t, "")
	hire(t, "fixer", "ทีมโค้ด", mode.Coding)
	if err := subagent.SaveTeam("ทีมเอกสาร", mode.Office, "", []string{"fixer", "doc"}); err != nil {
		t.Fatal(err)
	}

	code := a.DelegateSwitches("ทีมโค้ด")
	if code.Team != "ทีมโค้ด" || code.Agents.Off {
		t.Errorf("a user team does not start with delegation on: %+v", code.Agents)
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
	if got := names(a.DelegateSwitches(subagent.DefaultTeam)); slices.Contains(got, "fixer") || !slices.Contains(got, "doc") {
		t.Errorf("the default team's rows are %v — fixer is on a team of the user's, doc is bundled", got)
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
	after := a.SetDelegateOff("ทีมโค้ด", "agents", true)
	if !after.Agents.Off {
		t.Error("the coding team's delegation switch did not take")
	}
	if a.DelegateSwitches(subagent.DefaultTeam).Agents.Off != !a.cur().cfg.DelegateAgents {
		t.Error("a user team's switch moved the default team's")
	}
	if a.cur().cfg.DelegateSet {
		t.Error("a user team's switch set DelegateSet — that flag guards the shipped default, which no user team has")
	}
	// The helpers half is the session's, whichever team is asked.
	a.SetDelegateOff("ทีมโค้ด", "helpers", true)
	if !a.DelegateSwitches(subagent.DefaultTeam).Helpers.Off {
		t.Error("the helpers switch is per team — hands are not on a roster")
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
