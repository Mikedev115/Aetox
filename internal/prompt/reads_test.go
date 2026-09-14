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

// The other half of the shelf, since 2026-09-14: every skill by name and one
// line, whether or not it claims a moment. A skill without `before:` used to be
// invisible unless the model called skills_list, and it almost never did.
func TestReadsListEveryInstalledSkill(t *testing.T) {
	long := strings.Repeat("word ", 40) // 200 runes, well past the cap
	useShelf(t,
		Read{Skill: "aetox-web-templates", Description: "Page templates for a web site.", Before: "writing a web page"},
		Read{Skill: "invoice", Description: long},
		Read{Skill: "bare"},
	)
	got := reads(Desk{Name: "assistant", Carries: func(string) bool { return true }})
	for _, want := range []string{
		"- aetox-web-templates: Page templates for a web site.",
		"- invoice: word word",
		"- bare\n",
		`before writing a web page: skill_view "aetox-web-templates"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("reads is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, long) {
		t.Errorf("a long description must be clipped in the index:\n%s", got)
	}
	if strings.Contains(got, `skill_view "invoice"`) || strings.Contains(got, `skill_view "bare"`) {
		t.Errorf("a skill with no claim gets an index line, not a claim:\n%s", got)
	}
	// Index before claims: the model reads what is on the shelf, then which
	// of it decided its own moment.
	if strings.Index(got, "- invoice:") > strings.Index(got, "before writing a web page") {
		t.Errorf("the index must come before the claims:\n%s", got)
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

// A planning turn hears the claims and not the index. The index is every skill
// by name — 42% of the วางแผน prompt when measured on 14 ก.ย. 2026 — and it is
// there so a turn about to DO something can find its document; a plan reads
// what the before-lines name and has skills_list for the rest.
func TestAPlanningTurnHearsTheClaimsNotTheShelf(t *testing.T) {
	useShelf(t,
		Read{Skill: "aetox-grill", Description: "Interview a plan.", Before: "writing a plan whose scope is open"},
		Read{Skill: "invoice", Description: "Draft an invoice."},
	)
	planning := Desk{Name: "coding", Planning: true, Carries: func(string) bool { return true }}
	got := reads(planning)
	if !strings.Contains(got, `before writing a plan whose scope is open: skill_view "aetox-grill"`) {
		t.Errorf("the claim a plan acts on went missing:\n%s", got)
	}
	for _, index := range []string{"- invoice:", "- aetox-grill:", "Skills installed on this machine"} {
		if strings.Contains(got, index) {
			t.Errorf("a planning turn was handed the whole shelf (%q):\n%s", index, got)
		}
	}
	if !strings.Contains(got, "skills_list") {
		t.Errorf("a planning turn is not told where the rest of the shelf is:\n%s", got)
	}

	useShelf(t, Read{Skill: "invoice", Description: "Draft an invoice."})
	if got := reads(planning); got != "" {
		t.Errorf("no claims means nothing to say to a planning turn, got %q", got)
	}
}
