package skilllint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func one(files map[string]string) []Finding {
	return Lint(Skill{Name: "aetox-thing", Files: files})
}

func rules(fs []Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Rule)
	}
	return out
}

func has(fs []Finding, rule string, sev Severity) bool {
	for _, f := range fs {
		if f.Rule == rule && f.Severity == sev {
			return true
		}
	}
	return false
}

const clean = `---
name: aetox-thing
before: doing the thing, before the first line
description: ตอนผู้ใช้ขอให้ทำสิ่งนี้ และยังไม่มีแบบที่ตกลงกัน
---

# Thing

Say the rule. Hand a bug to ` + "`aetox-debug`" + `.

- One idea per line.
`

// The skill the rules were written for produces nothing: a linter that
// flags its own house style is a linter nobody keeps running.
func TestACleanSkillHasNoFindings(t *testing.T) {
	if fs := one(map[string]string{"SKILL.md": clean}); len(fs) != 0 {
		t.Fatalf("clean skill reported %v", fs)
	}
}

func TestFrontmatterIsHeldToTheFolder(t *testing.T) {
	fs := one(map[string]string{"SKILL.md": strings.Replace(clean, "name: aetox-thing", "name: aetox-other", 1)})
	if !has(fs, "frontmatter", Error) {
		t.Fatalf("a name that is not the folder's must be an error, got %v", rules(fs))
	}
	fs = one(map[string]string{"SKILL.md": "# no frontmatter\n\nbody\n"})
	if !has(fs, "frontmatter", Error) {
		t.Fatalf("no frontmatter must be an error, got %v", rules(fs))
	}
	fs = one(map[string]string{"SKILL.md": strings.Replace(clean, "description: ตอนผู้ใช้ขอให้ทำสิ่งนี้ และยังไม่มีแบบที่ตกลงกัน", "description: Thing layout, tokens and steps", 1)})
	if !has(fs, "description-moment", Note) {
		t.Fatalf("a description with no moment is a note, got %v", rules(fs))
	}
	if has(fs, "frontmatter", Error) {
		t.Fatalf("a present description is not a frontmatter error, got %v", fs)
	}
}

// A door the model is sent through and refused at is the shape §278's audit
// found by hand: `$ARGUMENTS` from claudekit, in a body and in a reference
// file. Inside inline code it is a token being explained, not a door.
func TestADeadDoorIsAnErrorUnlessItIsBeingExplained(t *testing.T) {
	fs := one(map[string]string{
		"SKILL.md":             clean,
		"references/update.md": "Run it.\n\n<args>$ARGUMENTS</args>\n",
	})
	if !has(fs, "dead-door", Error) {
		t.Fatalf("$ARGUMENTS in a reference must be an error, got %v", fs)
	}
	fs = one(map[string]string{"SKILL.md": clean + "\n- **`$ARGUMENTS`** is replaced by whatever follows the name.\n"})
	if has(fs, "dead-door", Error) {
		t.Fatalf("$ARGUMENTS inside inline code is documentation, got %v", fs)
	}
}

func TestANamedFileMustShip(t *testing.T) {
	fs := one(map[string]string{
		"SKILL.md":            clean + "\nRead `references/rules.md` and templates/x.md.\n",
		"references/other.md": "x",
	})
	if !has(fs, "named-file", Error) {
		t.Fatalf("a named file the folder does not ship must be an error, got %v", fs)
	}
	// templates/ is not a folder this skill ships at all, so templates/x.md
	// is a path in the user's project, not a door of the skill's own.
	for _, f := range fs {
		if f.Rule == "named-file" && strings.Contains(f.Message, "templates/") {
			t.Fatalf("a folder the skill does not ship is not its door: %v", f)
		}
	}
}

func TestLineRulesReadTheProseAndSkipTheCode(t *testing.T) {
	body := clean + `
- Try to keep it short.
- The model MUST announce itself.
- A line with an em dash — like this one.

` + "```\n- ideally this is code and MUST — be skipped\n```\n"
	fs := one(map[string]string{"SKILL.md": body})
	if !has(fs, "hedge", Warn) || !has(fs, "shout", Warn) || !has(fs, "em-dash", Warn) {
		t.Fatalf("expected hedge, shout and em-dash warnings, got %v", fs)
	}
	count := 0
	for _, f := range fs {
		if f.Rule == "hedge" || f.Rule == "shout" || f.Rule == "em-dash" {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("the fenced block must not be read: %d line findings, want 3: %v", count, fs)
	}
}

func TestAnUnclosedFenceIsAnError(t *testing.T) {
	fs := one(map[string]string{"SKILL.md": clean + "\n```js\nconst x = 1\n"})
	if !has(fs, "fence", Error) {
		t.Fatalf("an unclosed fence must be an error, got %v", rules(fs))
	}
}

func TestCRLFAndBOMAreErrors(t *testing.T) {
	fs := one(map[string]string{"SKILL.md": strings.ReplaceAll(clean, "\n", "\r\n")})
	if !has(fs, "crlf", Error) {
		t.Fatalf("CRLF must be an error, got %v", rules(fs))
	}
	fs = one(map[string]string{"SKILL.md": "\uFEFF" + clean})
	if !has(fs, "bom", Error) {
		t.Fatalf("a BOM must be an error, got %v", rules(fs))
	}
}

func TestLintAllowKeepsADeliberateChoice(t *testing.T) {
	body := clean + "\n<!-- lint-allow shout -->\n- The word MUST stays here on purpose.\n- ALWAYS here too. <!-- lint-allow shout -->\n- but not here: NEVER.\n"
	fs := one(map[string]string{"SKILL.md": body})
	n := 0
	for _, f := range fs {
		if f.Rule == "shout" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("two allowed lines and one not: want 1 shout, got %d in %v", n, fs)
	}
}

func TestAFragmentGetsTheLineRulesOnly(t *testing.T) {
	fs := LintFragment("## Added\n\n- Try to do it.\n")
	if !has(fs, "hedge", Warn) {
		t.Fatalf("a fragment's rule line is still read, got %v", fs)
	}
	for _, f := range fs {
		if f.Rule == "frontmatter" || f.Rule == "handoff" || f.Rule == "description-moment" {
			t.Fatalf("a fragment has no frontmatter to hold to: %v", f)
		}
	}
}

func TestSeverityFilterAndGate(t *testing.T) {
	fs := []Finding{{Rule: "a", Severity: Note}, {Rule: "b", Severity: Warn}, {Rule: "c", Severity: Error}}
	if got := AtLeast(fs, Warn); len(got) != 2 {
		t.Fatalf("AtLeast(Warn) = %v", got)
	}
	if !HasErrors(fs) || HasErrors(fs[:2]) {
		t.Fatal("HasErrors is the error gate")
	}
}

// The bundled shelf passes its own linter at the error level on every run.
// Warnings and notes are reported through `aetox skill lint`, not here: a
// build that fails on a note is a build people learn to ignore, and the
// error level is the one whose findings are broken doors, not taste.
func TestTheBundledShelfHasNoErrors(t *testing.T) {
	root := filepath.Join("..", "skill", "skills")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		dir := filepath.Join(root, e.Name())
		if !e.IsDir() || !IsSkillDir(dir) {
			continue
		}
		s, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		n++
		for _, f := range AtLeast(Lint(s), Error) {
			t.Errorf("%s: %s", e.Name(), f)
		}
	}
	if n < 30 {
		t.Fatalf("read %d bundled skills; the shelf has more than that, so the path is wrong", n)
	}
}
