package prompt

import (
	"strings"
	"testing"
)

// useShelf points the package at a fixture shelf for one test and puts back
// whatever was there. The claims live in a package-level seam now, so a test
// that set one and walked away would be a test that changed every prompt built
// after it.
func useShelf(t *testing.T, claims ...Read) {
	t.Helper()
	before, _ := shelf.Load().(func() []Read)
	shelf.Store(func() []Read { return claims })
	t.Cleanup(func() {
		if before == nil {
			shelf.Store(func() []Read { return nil })
			return
		}
		shelf.Store(before)
	})
}

// "Before X, read Y" is the sentence a model follows where the principle in
// capability() is ignored (§221). It comes from the machine's shelf, in the
// order the shelf gave, and only where skill_view is there to act on it.
func TestReadsNameTheSkillBeforeItsWork(t *testing.T) {
	useShelf(t,
		Read{Skill: "aetox-web-templates", Before: "writing a web page"},
		Read{Skill: "invoice", Before: "drafting an invoice"},
	)
	got := reads(Desk{Name: "assistant", Carries: func(string) bool { return true }})
	for _, want := range []string{
		"skill_view",
		`before writing a web page: skill_view "aetox-web-templates"`,
		`before drafting an invoice: skill_view "invoice"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("reads is missing %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "aetox-web-templates") > strings.Index(got, `"invoice"`) {
		t.Errorf("claims must keep the shelf's order:\n%s", got)
	}
}

func TestReadsAreSilentWithNothingToSay(t *testing.T) {
	useShelf(t)
	if got := reads(Desk{Carries: func(string) bool { return true }}); got != "" {
		t.Errorf("no claims must produce no layer, got %q", got)
	}

	useShelf(t, Read{Skill: "x", Before: "y"})
	noView := Desk{Carries: func(name string) bool { return name != "skill_view" }}
	if got := reads(noView); got != "" {
		t.Errorf("a desk without skill_view cannot act on a claim and must not hear it, got %q", got)
	}
}

// The layer is part of the built prompt, after the capability principle it
// makes concrete, and skipped with the rest of the tool instruction on a desk
// that carries nothing.
func TestBuiltPromptCarriesTheReads(t *testing.T) {
	useShelf(t, Read{Skill: "aetox-web-templates", Before: "writing a web page"})
	desk := Desk{Name: "assistant", Carries: func(string) bool { return true }}
	got := BuildForDesk(SurfaceDesktop, Scope{Root: t.TempDir()}, desk)
	if !strings.Contains(got, `skill_view "aetox-web-templates"`) {
		t.Fatalf("built prompt does not carry the read:\n%s", got)
	}
	if strings.Index(got, "skills_list") > strings.Index(got, `skill_view "aetox-web-templates"`) {
		t.Error("the claim must follow the principle it makes concrete")
	}
}

// The reason the claims left Desk on 2026-09-05. A shelf is a fact about the
// machine, so every way of asking for a prompt has to hear the same thing:
// both front ends, and a session that predates desks entirely. While it hung
// on the desk, the CLI — which builds no Desk — was told none of the claims,
// and desktop's full desk carried a line that prompt.Build did not, which
// desktop's own TestALegacySessionKeepsTheFullDeskAndTheSamePrompt forbids.
func TestEveryDoorHearsTheSameShelf(t *testing.T) {
	useShelf(t, Read{Skill: "aetox-web-templates", Before: "writing a web page"})
	root := t.TempDir()
	claim := `skill_view "aetox-web-templates"`

	full := Desk{Carries: func(string) bool { return true }}
	for _, c := range []struct {
		name string
		text string
	}{
		{"Build/desktop", Build(SurfaceDesktop, Scope{Root: root})},
		{"Build/cli", Build(SurfaceCLI, Scope{Root: root})},
		{"BuildForDesk/zero", BuildForDesk(SurfaceDesktop, Scope{Root: root}, Desk{})},
		{"BuildForDesk/full", BuildForDesk(SurfaceDesktop, Scope{Root: root}, full)},
	} {
		if !strings.Contains(c.text, claim) {
			t.Errorf("%s does not carry the shelf's claim", c.name)
		}
	}

	// The one that used to fail, stated here in the package that owns it: a
	// legacy session is the zero Desk, and its prompt must be Build's byte for
	// byte or an existing conversation pays for the upgrade in cache.
	if a, b := Build(SurfaceDesktop, Scope{Root: root}), BuildForDesk(SurfaceDesktop, Scope{Root: root}, Desk{}); a != b {
		t.Error("a zero Desk no longer produces prompt.Build's prompt")
	}
}
