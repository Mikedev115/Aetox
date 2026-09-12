package main

// The two writes on ตั้งค่าสกิลสำหรับเอเจนเฉพาะ (Capability.svelte): a shelf
// skill copied into one agent's own folder, and a copy taken out again.
//
// A copy, not a pointer. An agent's skill is a folder in its home
// (config.AgentSkillsPath) and nothing else — subagent/skills.go says why the
// shared shelf is the wrong place for a worker's knowledge, and a `for:` list
// on a shelf skill would put it back there under another name. So the page's
// tick is a file operation: the folder lands in agents/<name>/skills, the
// original on the shelf stays, and the agent reads its copy from then on.
// Editing the copy does not touch the shelf, which is the point of a copy.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// CopySkillToAgent puts the shelf skill called name into agent's own folder.
//
// Resolved by name off the same scan ของคุณ lists, so the page cannot offer a
// skill this cannot find; a bundled skill copies out of the binary through
// DiscoveredSkill.Folder, which is the only road it has. A name the agent
// already holds is refused rather than overwritten: the folder there may be
// the user's own edit of it, and "add" must never be a silent "replace".
func (a *App) CopySkillToAgent(agent, name string) error {
	src, ok := findShelfSkill(name)
	if !ok {
		return fmt.Errorf("ไม่พบสกิล %q บนชั้น", name)
	}
	fsys := src.Folder()
	if fsys == nil {
		return fmt.Errorf("สกิล %q ไม่มีโฟลเดอร์ให้ก๊อป", name)
	}
	home, err := config.AgentSkillsPath(agent)
	if err != nil {
		return err
	}
	dest := filepath.Join(home, folderNameOf(src))
	if _, statErr := os.Stat(dest); statErr == nil {
		return fmt.Errorf("%s มีสกิล %q อยู่แล้ว — แก้ในโฟลเดอร์ของเอเจนได้เลย", agent, src.Name)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return err
	}
	if err := os.CopyFS(dest, fsys); err != nil {
		// Half a skill is worse than none: a SKILL.md whose references/ never
		// arrived is the "document full of doors that open onto nothing" that
		// bundled_skills.go was written to prevent.
		_ = os.RemoveAll(dest)
		return err
	}
	return nil
}

// RemoveAgentSkill deletes one folder out of agent's own skills.
//
// Only a folder in the agent's home can go: a skill that ships with the
// profile has no folder there (AgentSkillInfo.Bundled), and the way to be rid
// of it is the same as on the shelf — a folder of the same name that says
// something else. Resolved by name against the home's own scan so the path
// deleted is always one under that home, never one the caller typed.
func (a *App) RemoveAgentSkill(agent, name string) error {
	home, err := config.AgentSkillsPath(agent)
	if err != nil {
		return err
	}
	own, _ := skill.ScopedSkills([]string{home})
	for _, s := range own {
		if strings.EqualFold(s.Name, name) && s.Dir != "" {
			return os.RemoveAll(s.Dir)
		}
	}
	return fmt.Errorf("สกิล %q ไม่ได้อยู่ในโฟลเดอร์ของ %s — ถ้าติดมากับเอเจน ลบไม่ได้", name, agent)
}

// findShelfSkill is the name lookup ของคุณ, RemoveExternalSkill and the copy
// above all mean: the shared shelf plus the bundled set, matched the way
// skill_view matches (case-folded).
func findShelfSkill(name string) (skill.DiscoveredSkill, bool) {
	for _, s := range skill.ListDiscovered(skill.DefaultDiscoveryPaths()) {
		if strings.EqualFold(s.Name, name) {
			return s, true
		}
	}
	return skill.DiscoveredSkill{}, false
}

// folderNameOf is the directory a copy is written under. A skill on disk keeps
// its folder's name — that is what the user typed and what an override of the
// same skill has to match; a bundled one has no folder, so its name serves.
func folderNameOf(s skill.DiscoveredSkill) string {
	if s.Dir != "" {
		return filepath.Base(s.Dir)
	}
	return s.Name
}
