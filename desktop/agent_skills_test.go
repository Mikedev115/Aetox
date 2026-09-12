package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// ownOnly drops what shipped with the profile, leaving what is in the folder.
func ownOnly(in []AgentSkillInfo) []AgentSkillInfo {
	var out []AgentSkillInfo
	for _, s := range in {
		if !s.Bundled {
			out = append(out, s)
		}
	}
	return out
}

// A shelf skill ticked for an agent lands in that agent's own folder, whole,
// and the shelf keeps its copy. Ticked twice, the second is refused rather
// than overwriting what may by then be the user's edit.
func TestCopySkillToAgentLandsInHomeAndKeepsShelf(t *testing.T) {
	a := newSkillsTestApp(t)
	src := writeTestSkill(t, "invoice-th")
	if err := os.WriteFile(filepath.Join(src, "rules.md"), []byte("no VAT on exports\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := a.CopySkillToAgent("doc", "invoice-th"); err != nil {
		t.Fatalf("copy: %v", err)
	}
	home, _ := config.AgentSkillsPath("doc")
	for _, f := range []string{"SKILL.md", "rules.md"} {
		if _, err := os.Stat(filepath.Join(home, "invoice-th", f)); err != nil {
			t.Fatalf("%s missing from the agent's copy: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
		t.Fatalf("shelf copy gone after copying to agent: %v", err)
	}
	// doc ships three skills of its own; only the folder's contents are this
	// test's business.
	got := ownOnly(a.AgentSkills("doc"))
	if len(got) != 1 || got[0].Name != "invoice-th" {
		t.Fatalf("AgentSkills(doc) own = %+v, want the one copy", got)
	}

	if err := a.CopySkillToAgent("doc", "invoice-th"); err == nil {
		t.Fatal("second copy of the same name should be refused, not overwrite")
	}
	if err := a.CopySkillToAgent("doc", "no-such-skill"); err == nil {
		t.Fatal("copying a skill the shelf does not have should fail")
	}
}

// A bundled skill has no folder on disk; the copy reads it out of the binary
// and the agent gets a real folder it can edit.
func TestCopyBundledSkillToAgentReadsTheBinary(t *testing.T) {
	a := newSkillsTestApp(t)
	var bundled string
	for _, s := range a.ListExternalSkills() {
		if s.Bundled {
			bundled = s.Name
			break
		}
	}
	if bundled == "" {
		t.Skip("no bundled skills compiled in")
	}
	if err := a.CopySkillToAgent("doc", bundled); err != nil {
		t.Fatalf("copy bundled %s: %v", bundled, err)
	}
	home, _ := config.AgentSkillsPath("doc")
	if _, err := os.Stat(filepath.Join(home, bundled, "SKILL.md")); err != nil {
		t.Fatalf("bundled copy has no SKILL.md: %v", err)
	}
}

// Removal takes out only what is in the agent's folder, resolved by name there
// — a name that is not in the folder (a profile-shipped skill included) is an
// error, not a no-op that reports success.
func TestRemoveAgentSkillOnlyTouchesTheHome(t *testing.T) {
	a := newSkillsTestApp(t)
	writeTestSkill(t, "invoice-th")
	if err := a.CopySkillToAgent("doc", "invoice-th"); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveAgentSkill("doc", "INVOICE-TH"); err != nil {
		t.Fatalf("remove (case-folded): %v", err)
	}
	if got := ownOnly(a.AgentSkills("doc")); len(got) != 0 {
		t.Fatalf("still listed after remove: %+v", got)
	}
	if err := a.RemoveAgentSkill("doc", "invoice-th"); err == nil {
		t.Fatal("removing what is not there should be an error")
	}
}
