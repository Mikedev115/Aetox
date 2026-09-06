package skill

import (
	"strings"
	"testing"
)

// A skill may name the work it must be read before (§221). The claim rides on
// the frontmatter beside name and description, and nowhere else: no desk file
// spells a skill's name to make the prompt say "read this first".
func TestDiscoverSkills_ReadsTheBeforeClaim(t *testing.T) {
	root := t.TempDir()
	writeSkillFixture(t, root, "invoice", "---\nname: invoice\nbefore:  drafting an invoice for a client \ndescription: Invoice layout\n---\nbody\n")
	writeSkillFixture(t, root, "quiet", "---\nname: quiet\ndescription: Makes no claim\n---\nbody\n")

	found := ListDiscovered([]string{root})
	byName := map[string]DiscoveredSkill{}
	for _, d := range found {
		byName[d.Name] = d
	}
	if got := byName["invoice"].Before; got != "drafting an invoice for a client" {
		t.Errorf("Before = %q, want the trimmed claim", got)
	}
	if got := byName["quiet"].Before; got != "" {
		t.Errorf("a skill with no before: line must carry none, got %q", got)
	}
}

// `aetox-web-templates` had the first claim on the shelf. The owner withdrew it
// on 2026-09-05, for the reason its own STANDARD.md opens with: a library that
// encodes the average does not help a model that is already good, it pins it to
// the average. A claim compels the lookup on every page, so a library below the
// model's own standard becomes a ceiling rather than a floor, and the
// measurement said it was below: 104 sections averaging 11.7KB against the slide
// shelf's 2.2KB, and a test suite that checks hygiene a from-scratch page
// already passes rather than any value a model cannot compute. The withdrawal
// was the decision; the `before:` line is still in that file at HEAD, so the
// shelf carries two claims until the two are reconciled.
//
// The note that stood here also said some skill would earn one, and
// `aetox-teach` is that skill. The test below records why, because the two cases
// look alike from a distance and are opposites underneath. What was withdrawn
// was a library competing with something the model already does well. What is
// claimed here cannot be computed at all: the names printed on this build's
// rooms and buttons, which settings page a thing is on, and the order a person
// learns them in. A model guessing there produced the answer of 22 ส.ค.,
// "ไฟล์ เชลล์ desk_open", three tool ids handed to a user who has never seen one
// and cannot type it.
//
// The moment is narrow on purpose: being asked what this app can do, or teaching
// somebody to use it. A claim earns its line by being both unguessable and rare,
// and the second half is as load-bearing as the first.
func TestTheTeachingSkillClaimsTheMomentItIsFor(t *testing.T) {
	var claim string
	for _, s := range bundledSkills() {
		if s.Name == "aetox-teach" {
			claim = s.Before
		}
	}
	if claim == "" {
		t.Fatal("aetox-teach ships without the claim that is the whole point of it")
	}
	// A moment in the user's words, not a topic and not a file name: reads()
	// prints it straight into the prompt as "before <this>".
	if !strings.Contains(claim, "what Aetox can do") {
		t.Errorf("the claim does not name the moment it is for: %q", claim)
	}
}
