package subagent

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// A team is a roster: which agents a session hires from, and the desk they
// work at while it does (owner's call, 12 ก.ย. 2026 — DECISIONS §256). It is
// the unit the chat's picker offers instead of every agent on the machine,
// and the way an agent the user wrote reaches the code door without the
// star (§84) bending: a team at the coding desk is hired by the coding desk
// and hands its result back to the coding desk, which is hiring within a desk
// (§94.3), not across one.
//
// **A team is a list, not a home.** An agent stays one folder under
// agents/<name>/ with one memory and one job history (§94.1: a name has one
// owner), and a team names it — so the same agent can sit on a team at the
// office and on one at the coding desk, and what it learned at either follows
// it to the other, because it is the same worker. §94.1's other shape — a
// folder per desk — would have made "use my agent at the code door" mean
// "copy it out under another name", which is the request this exists to
// answer. What a list costs is a name that can go stale when the agent it
// names is deleted; that name is reported (Missing), never silently dropped,
// and never runs.
//
// **A team has one desk, and that desk's manifest is the ceiling** over
// every member while it works for this team (§94.2 unchanged). The desk is
// written once, in TEAM.md, so no agent file ever writes `desk:` (v0.9.3's
// rule, kept). A team at the coding desk therefore holds a shell — the first
// time an agent does — through the same gate the coding desk itself answers
// to; what §84 protected is still protected, because a document writer on an
// office team is still under the office ceiling.
//
// **The default team is not a file.** ทีมผู้ช่วย is the bundled agents plus
// every agent of the user's that no team names, so dropping a folder into
// agents/ still hires — nobody vanishes for not having been put on a list —
// and a machine that has never made a team behaves exactly as it did before
// teams existed. It cannot be edited (owner, 12 ก.ย.): a member the user does
// not want hired is switched off, not removed, and there is no file on disk
// to go stale.
//
// Home: <DataRoot>/teams/<name>/TEAM.md. A folder rather than a flat file for
// the reason an agent is one: room for what a team may come to hold.

// TeamDefinitionFile is the one file a team folder holds today.
const TeamDefinitionFile = "TEAM.md"

// DefaultTeam is the id of ทีมผู้ช่วย — empty, which is also what
// sessions.team stores for a session that never chose, so a conversation
// from before teams existed is on the default team by construction.
const DefaultTeam = ""

// Team is one roster as the rest of the app sees it.
type Team struct {
	// Name is the folder's; "" is the default team, which has no folder.
	Name string `json:"name"`
	// Desk is where the members work for this team — the ceiling.
	Desk        string `json:"desk"`
	Description string `json:"description"`
	// Members are the agents this team names that actually exist, in the
	// file's order; for the default team, the roster's order.
	Members []string `json:"members"`
	// Missing are names the file lists that no agent answers to — reported so
	// a deleted agent leaves a visible hole rather than a quietly shorter
	// list. They never run.
	Missing []string `json:"missing,omitempty"`
	Path    string   `json:"path,omitempty"`
	Default bool     `json:"default"`
	// Invalid is why this team cannot be hired from, in the user's language,
	// or "" for a healthy one. A team at a desk no team may sit at is the
	// case: it stays on the page with its reason, and no picker offers it.
	Invalid string `json:"invalid,omitempty"`
}

// teamDesks are the desks a team may sit at. The assistant desk is not one:
// it hires from the office (dispatch: specialized) and holds no roster of its
// own, so a team there would be a list nothing could read.
var teamDesks = []string{mode.Office, mode.Coding}

// TeamsDir returns <DataRoot>/teams — the teams' home (not created here).
func TeamsDir() (string, error) {
	root, err := config.DataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "teams"), nil
}

// Teams reports every team: the default first, then the user's, alphabetical.
// Read from disk on every call, like Chairs — a folder the user just made
// must be on the next list.
func Teams() []Team {
	users := userTeams()
	out := make([]Team, 0, len(users)+1)
	out = append(out, defaultTeam(users))
	return append(out, users...)
}

// TeamsAt reports the healthy teams whose members work at the named desk —
// what a picker at that desk offers.
func TeamsAt(desk string) []Team {
	desk = strings.ToLower(strings.TrimSpace(desk))
	var out []Team
	for _, t := range Teams() {
		if t.Desk == desk && t.Invalid == "" {
			out = append(out, t)
		}
	}
	return out
}

// LoadTeam returns the team named name — the default for "" — and false when
// no such team exists or the one that does cannot be hired from.
func LoadTeam(name string) (Team, bool) {
	name = strings.TrimSpace(name)
	for _, t := range Teams() {
		if t.Name == name && t.Invalid == "" {
			return t, true
		}
	}
	return Team{}, false
}

// Has reports whether the named agent is on this team. Case is forgiven, as
// it is everywhere a name is typed by hand (WorkersOff, `@name`).
func (t Team) Has(name string) bool {
	name = strings.TrimSpace(name)
	for _, m := range t.Members {
		if strings.EqualFold(m, name) {
			return true
		}
	}
	return false
}

// MemberProfiles resolves the members to their profiles, in the team's order.
// A name that stopped resolving between the read and now is skipped, which is
// the same answer Missing gives it.
func (t Team) MemberProfiles() []Profile {
	out := make([]Profile, 0, len(t.Members))
	for _, name := range t.Members {
		if p, ok := Load(name); ok && p.Desk != "" {
			out = append(out, p)
		}
	}
	return out
}

// defaultTeam builds ทีมผู้ช่วย: the bundled agents (and the user's edits
// of them), plus every agent of the user's that no team names. Computed from
// the roster and the user's teams rather than stored, so it cannot be out of
// date with either.
func defaultTeam(users []Team) Team {
	named := map[string]bool{}
	for _, t := range users {
		for _, m := range t.Members {
			named[strings.ToLower(m)] = true
		}
	}
	t := Team{Name: DefaultTeam, Desk: mode.Office, Default: true, Members: []string{}}
	for _, p := range Chairs(mode.Office) {
		if p.Builtin || p.Overrides || !named[strings.ToLower(p.Name)] {
			t.Members = append(t.Members, p.Name)
		}
	}
	return t
}

// userTeams reads every folder under the teams' home, alphabetical. A folder
// with no TEAM.md is not a team and is not reported: nothing was written
// there to explain.
func userTeams() []Team {
	dir, err := TeamsDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // no folder yet is the normal state
	}
	roster := Chairs(mode.Office)
	var out []Team
	for _, e := range entries {
		if !e.IsDir() || !validName(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name(), TeamDefinitionFile)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		t := parseTeam(e.Name(), string(raw), roster)
		t.Path = path
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// parseTeam reads one TEAM.md. The folder is the name, as it is for an agent.
// Members are matched against the roster case-insensitively and written back
// in the roster's spelling, so a hand-typed "Doc" hires doc and the id every
// other table keys on is the one that gets stored.
func parseTeam(name, raw string, roster []Profile) Team {
	fields, _, err := skill.ParseFrontmatter(raw)
	if err != nil {
		fields = map[string]string{}
	}
	t := Team{
		Name:        name,
		Desk:        strings.ToLower(strings.TrimSpace(fields["desk"])),
		Description: strings.TrimSpace(fields["description"]),
		Members:     []string{},
	}
	if t.Desk == "" {
		t.Desk = mode.Office
	}
	if !teamDeskAllowed(t.Desk) {
		t.Invalid = "ไฟล์นี้เขียน desk: " + t.Desk + " แต่ทีมนั่งได้เฉพาะโต๊ะ " +
			strings.Join(teamDesks, " หรือ ") + " — แก้บรรทัด desk แล้วทีมนี้จะกลับมาใช้ได้"
	}
	for _, part := range strings.Split(fields["members"], ",") {
		want := strings.TrimSpace(part)
		if want == "" {
			continue
		}
		found := ""
		for _, p := range roster {
			if strings.EqualFold(p.Name, want) {
				found = p.Name
				break
			}
		}
		switch {
		case found == "":
			t.Missing = append(t.Missing, want)
		case !t.Has(found):
			t.Members = append(t.Members, found)
		}
	}
	return t
}

func teamDeskAllowed(desk string) bool {
	for _, d := range teamDesks {
		if d == desk {
			return true
		}
	}
	return false
}

// SaveTeam writes a team's file — the one write door teams have. The default
// team is refused by name: it has no file, and a user who wants a bundled
// agent out of the assistant's reach switches it off instead.
//
// Members are checked at the door rather than left for the read to report as
// Missing: the read forgives a name that went stale after it was written,
// but a save is the moment the name was typed, and a typo is cheaper to
// refuse than to file.
func SaveTeam(name, desk, description string, members []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("ต้องตั้งชื่อทีมก่อน")
	}
	if len([]rune(name)) > 40 {
		return errors.New("ชื่อทีมยาวเกินไป (ไม่เกิน 40 ตัวอักษร)")
	}
	if !validName(name) {
		return errors.New(`ชื่อทีมห้ามมีช่องว่างหรืออักขระเหล่านี้: \ / : * ? " < > |`)
	}
	desk = strings.ToLower(strings.TrimSpace(desk))
	if desk == "" {
		desk = mode.Office
	}
	if !teamDeskAllowed(desk) {
		return errors.New("ทีมนั่งได้เฉพาะโต๊ะ " + strings.Join(teamDesks, " หรือ "))
	}
	roster := Chairs(mode.Office)
	kept := make([]string, 0, len(members))
	for _, want := range members {
		want = strings.TrimSpace(want)
		if want == "" {
			continue
		}
		found := ""
		for _, p := range roster {
			if strings.EqualFold(p.Name, want) {
				found = p.Name
				break
			}
		}
		if found == "" {
			return errors.New("ไม่มีเอเจนชื่อ " + want + " — ทีมใส่ได้เฉพาะเอเจนที่มีอยู่แล้ว")
		}
		dup := false
		for _, k := range kept {
			if k == found {
				dup = true
			}
		}
		if !dup {
			kept = append(kept, found)
		}
	}
	dir, err := TeamsDir()
	if err != nil {
		return err
	}
	home := filepath.Join(dir, name)
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("desk: " + desk + "\n")
	if d := strings.TrimSpace(description); d != "" {
		b.WriteString("description: " + strings.ReplaceAll(d, "\n", " ") + "\n")
	}
	b.WriteString("members: " + strings.Join(kept, ", ") + "\n")
	b.WriteString("---\n")
	return os.WriteFile(filepath.Join(home, TeamDefinitionFile), []byte(b.String()), 0o644)
}

// DeleteTeam removes a team's folder. The agents it named are untouched — the
// list is gone, the people are not — and the ones no other team names are
// back on the default team by the rule that put them there.
func DeleteTeam(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || !validName(name) {
		return errors.New("ชื่อทีมไม่ถูกต้อง")
	}
	dir, err := TeamsDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(dir, name))
}
