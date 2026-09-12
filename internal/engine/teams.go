package engine

import (
	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/subagent"
)

// Bindings for teams (DECISIONS §256): the roster a session hires from, drawn
// by the team page, the chat's picker and the code door's picker. Thin, like
// subagents.go — every rule about what a team is lives in internal/subagent,
// so no page can invent a second definition of one.

// TeamCard is one team as a page draws it: the team's own facts, and its
// members as chairs under the TEAM's desk — so a member's tool list on a
// coding team shows the shell it holds there, and the same agent on the
// default team shows none.
type TeamCard struct {
	// Name is the folder's.
	Name        string `json:"name"`
	Desk        string `json:"desk"`
	Description string `json:"description"`
	// Invalid is why the team cannot be hired from, "" when it can.
	Invalid string `json:"invalid,omitempty"`
	// Missing are names the file lists that no agent answers to.
	Missing []string `json:"missing"`
	Members []Chair  `json:"members"`
	Path    string   `json:"path,omitempty"`
}

// ListTeams reports the teams a desk can offer — those whose members work
// at it — or every team when desk is "". Read from disk per call, like
// ListChairs: a folder the user just made must be on the next list.
func (a *Engine) ListTeams(desk string) []TeamCard {
	used := a.chairActivity()
	var teams []subagent.Team
	if desk == "" {
		teams = subagent.Teams()
	} else {
		teams = subagent.TeamsAt(desk)
	}
	out := make([]TeamCard, 0, len(teams)) // never nil: §34
	for _, t := range teams {
		ceiling, _ := mode.Load(t.Desk)
		card := TeamCard{
			Name:        t.Name,
			Desk:        t.Desk,
			Description: t.Description,
			Invalid:     t.Invalid,
			Missing:     append([]string{}, t.Missing...),
			Members:     make([]Chair, 0, len(t.Members)),
			Path:        t.Path,
		}
		for _, p := range t.MemberProfiles() {
			card.Members = append(card.Members, a.chairCard(p, ceiling, used))
		}
		out = append(out, card)
	}
	return out
}

// SaveTeam writes one team — the team page's door. The rules (name, desk,
// members that exist) are the package's, and its refusals come back in the
// user's language.
func (a *Engine) SaveTeam(name, desk, description string, members []string) error {
	return subagent.SaveTeam(name, desk, description, members)
}

// DeleteTeam removes a team. The agents it named stay; the ones no other team
// names are back on ทีมผู้ช่วย by the rule that put them there.
func (a *Engine) DeleteTeam(name string) error {
	return subagent.DeleteTeam(name)
}

// TeamsFolderPath creates the teams' home if needed and answers with it — the
// same contract as AgentsFolderPath, so making a team by hand is "drop a
// folder with TEAM.md here".
func (a *Engine) TeamsFolderPath() (string, error) {
	return profileHome(subagent.TeamsDir)
}
