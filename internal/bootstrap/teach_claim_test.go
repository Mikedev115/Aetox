package bootstrap

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/prompt"
)

// The claim only does anything if it survives three seams: the frontmatter is
// parsed, the shelf this package registers finds a bundled skill and not only a
// folder on disk, and reads() prints it. Each of those is covered where it
// lives, and none of them proves the line reaches a real prompt.
//
// It is the line that decides what a user is told when they ask what this app
// can do, which is the first thing most of them ask and the one answer nobody
// gets a second chance at. So it is checked end to end, on the prompt a desk
// actually builds.
func TestTheTeachingClaimReachesABuiltPrompt(t *testing.T) {
	desk := prompt.Desk{
		Name:      "assistant",
		Direction: "This session is assistant work.",
		Carries:   func(string) bool { return true },
	}
	got := prompt.BuildForDesk(prompt.SurfaceDesktop, prompt.Scope{Root: t.TempDir()}, desk)
	if !strings.Contains(got, `skill_view "aetox-teach"`) {
		t.Errorf("a desk that can read skills is never told to read aetox-teach:\n%s", got)
	}
	// Printed as the moment, not as a topic. reads() writes "before <claim>",
	// and a claim that reads as a subject heading is one the model matches
	// against the wrong turns.
	if !strings.Contains(got, "before answering what Aetox can do") {
		t.Errorf("the claim did not reach the prompt as a moment:\n%s", got)
	}
}

// A desk with no skill_view cannot act on any claim, so reads() says nothing at
// all there. Worth pinning next to the test above: the two together are the
// whole contract, and a claim shown to a session that cannot open it is an
// instruction to call a tool that is not on the desk.
func TestADeskWithoutSkillViewHearsNoClaims(t *testing.T) {
	desk := prompt.Desk{
		Name:      "narrow",
		Direction: "This session is assistant work.",
		Carries:   func(name string) bool { return name != "skill_view" },
	}
	got := prompt.BuildForDesk(prompt.SurfaceDesktop, prompt.Scope{Root: t.TempDir()}, desk)
	if strings.Contains(got, "aetox-teach") {
		t.Errorf("a desk that cannot open a skill is told to open one:\n%s", got)
	}
}
