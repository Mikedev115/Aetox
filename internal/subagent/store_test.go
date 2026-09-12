package subagent

import (
	"slices"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// The helpers' write door is half open (12 ก.ย. 2026, after being shut on
// 2026-08-06): a bundled name may be shadowed, a new name may not, and
// deleting the shadow is the revert.
func TestSaveShadowsABundledHelperAndNothingElse(t *testing.T) {
	dir := isolate(t)

	if err := Save("explore", "---\ndescription: ของผม\nmodel: deepseek-v4\n---\nMine.\n"); err != nil {
		t.Fatalf("Save(explore): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "explore.md")); err != nil {
		t.Fatalf("the shadow was not written into the helpers' home: %v", err)
	}
	p, _ := Load("explore")
	if p.Prompt != "Mine." || p.Model != "deepseek-v4" || p.Builtin || !p.Overrides {
		t.Fatalf("the save did not take effect as a shadow: %+v", p)
	}
	if len(p.Tools) != 4 {
		t.Fatalf("the shadow changed explore's kit: %v", p.Tools)
	}

	// A name the app does not ship is refused, pointing at the team page.
	err := Save("backend", "---\ndescription: ของผม\n---\nMine.\n")
	if err == nil {
		t.Fatal("Save created a new helper")
	}
	if !strings.Contains(err.Error(), "เอเจน") {
		t.Errorf("the refusal does not point at the team page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "backend.md")); !os.IsNotExist(err) {
		t.Fatal("the refused save still wrote a file")
	}

	// Delete is the revert: the bundled explore is back, untouched.
	if err := Delete("explore"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "explore.md")); !os.IsNotExist(err) {
		t.Fatal("the shadow was not removed")
	}
	if p, _ := Load("explore"); !p.Builtin || p.Prompt == "Mine." {
		t.Fatalf("revert did not restore the bundled helper: %+v", p)
	}
	if err := Delete("explore"); err != nil {
		t.Fatalf("second Delete should be a no-op, got %v", err)
	}
}

// Name and body validation moved with the one write door that still exists.
func TestSaveAgentRejectsBadNames(t *testing.T) {
	isolate(t)
	for _, name := range []string{"", "   ", "..", "../evil", `sub\evil`, "has space", strings.Repeat("x", 41)} {
		if err := SaveAgent(name, "body"); err == nil {
			t.Errorf("SaveAgent(%q) was accepted", name)
		}
	}
	if err := SaveAgent("ok-name", "   "); err == nil {
		t.Error("empty body was accepted")
	}
}

// SetModel must edit one line and leave every other key — including ones this
// package does not read — exactly as written. For an agent the line lands in
// its AGENT.md; for a helper it lands in a shadow in the helpers' home — the
// one edit the owner asked for by name (12 ก.ย. 2026).
func TestSetModelEditsOneLine(t *testing.T) {
	isolate(t)

	if err := SetModel("doc", "deepseek-v4-flash"); err != nil {
		t.Fatalf("SetModel: %v", err)
	}
	raw, err := os.ReadFile(agentDefinition(t, "doc"))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "model: deepseek-v4-flash") {
		t.Errorf("model line missing:\n%s", text)
	}
	if strings.Count(text, "model:") != 1 {
		t.Errorf("model key appears %d times:\n%s", strings.Count(text, "model:"), text)
	}
	p, _ := Load("doc")
	if p.Model != "deepseek-v4-flash" || p.Desk != "specialized" {
		t.Fatalf("other fields damaged: %+v", p)
	}

	// Empty model = inherit: the line goes away instead of being set to "".
	if err := SetModel("doc", ""); err != nil {
		t.Fatalf("SetModel(clear): %v", err)
	}
	if p, _ := Load("doc"); p.Model != "" {
		t.Errorf("model = %q after clearing", p.Model)
	}

	// A helper is pinned through its own door: a shadow, the kit untouched.
	if err := SetModel("explore", "deepseek-v4-flash"); err != nil {
		t.Fatalf("SetModel(helper): %v", err)
	}
	if p, _ := Load("explore"); p.Model != "deepseek-v4-flash" || !p.Overrides || len(p.Tools) != 4 {
		t.Fatalf("helper pin: %+v", p)
	}
	if err := SetModel("explore", ""); err != nil {
		t.Fatalf("SetModel(helper, clear): %v", err)
	}
	if p, _ := Load("explore"); p.Model != "" {
		t.Errorf("helper model = %q after clearing", p.Model)
	}
}

func TestSetFrontmatterFieldOnUnusualInput(t *testing.T) {
	// No frontmatter at all: one gets added, body preserved.
	got := setFrontmatterField("just a prompt\n", "model", "m1")
	if !strings.HasPrefix(got, "---\nmodel: m1\n---\n") || !strings.Contains(got, "just a prompt") {
		t.Errorf("bare body: %q", got)
	}
	// Nothing to do, nothing changed.
	if got := setFrontmatterField("just a prompt\n", "model", ""); got != "just a prompt\n" {
		t.Errorf("no-op changed the file: %q", got)
	}
	// Unterminated frontmatter is left alone rather than guessed at.
	broken := "---\ndescription: x\nbody"
	if got := setFrontmatterField(broken, "model", "m1"); got != broken {
		t.Errorf("edited a broken file: %q", got)
	}
}

// The registry a sub-agent is handed is a copy holding only what it may use —
// and never `task`, which is what makes depth 1 structural.
func TestFilterRegistry(t *testing.T) {
	isolate(t)
	parent := skill.NewDefaultRegistry(skill.RegistryOptions{SandboxRoot: t.TempDir()})

	explore, _ := Load("explore")
	child := FilterRegistry(parent, explore, nil)
	// Two ENTRIES for four tools: `tools: grep, glob, list, read` reaches the
	// registry as `read` plus `search` narrowed to the three acts it named
	// (internal/skill/search_pack.go). This is the case Step 0's per-action
	// narrowing exists for, seen from the sub-agent side - a profile that names
	// actions must still get exactly those.
	if got := len(child.Names()); got != 2 {
		t.Fatalf("explore registry has %d entries, want 2: %v", got, child.Names())
	}
	for _, name := range explore.Tools {
		if _, ok := child.Get(name); ok {
			continue
		}
		// Not an entry of its own, so it has to be an act of one that is.
		found := false
		for _, entry := range child.Names() {
			if slices.Contains(skill.PackedActions(entry), name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q is neither a registry entry nor an act of one", name)
		}
	}
	// Source survives the copy, or the Tools panel would relabel every builtin.
	if src, ok := child.SourceOf("read"); !ok || src != skill.SourceBuiltin {
		t.Errorf("SourceOf(read) = %v, %v", src, ok)
	}

	general, _ := Load("general")
	generalRegistry := FilterRegistry(parent, general, nil)
	for _, name := range forcedDenials {
		if _, ok := generalRegistry.Get(name); ok {
			t.Errorf("%q reached a sub-agent's registry", name)
		}
	}
	if _, ok := generalRegistry.Get("shell"); !ok {
		t.Error("general lost shell, which it is supposed to inherit")
	}
	// Even inheriting everything, the copy is never the parent itself — a
	// sub-agent must not be able to mutate the session's registry.
	if generalRegistry == parent {
		t.Error("FilterRegistry handed back the parent registry")
	}

	denied := Profile{Deny: []string{"write", "edit"}}
	deniedRegistry := FilterRegistry(parent, denied, nil)
	for _, name := range denied.Deny {
		if _, ok := deniedRegistry.Get(name); ok {
			t.Errorf("a denied tool %q was handed over anyway", name)
		}
	}

	if FilterRegistry(nil, general, nil) != nil {
		t.Error("nil parent should stay nil")
	}
}
