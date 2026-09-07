package main

// Where a clip's length comes from, and what the tool says when it is not the
// caller's to choose.
//
// `seconds` was advertised on both actions and honoured by one of them, on four
// scenes out of fifty. An agent that asked for a 20-second clip from a 30-second
// scene got thirty seconds, reported twenty, and nothing anywhere said
// otherwise — including `video render`, whose signature carried `seconds` and
// whose argv never mentioned it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// The four that take a length, and the forty-six that do not, told apart by the
// markup rather than by a list of names kept somewhere else.
func TestOnlyTheAsAskedScenesCarryADurationSlot(t *testing.T) {
	for scene, want := range map[string]bool{
		"bold-portrait-title": true,
		"grain-texture-hero":  true,
		"playful-bounce":      true,
		"structured-grid":     true,
		"statement-title":     false,
		"product-launch-30s":  false,
		"airbnb-deck":         false,
		"minimal-hero":        false,
	} {
		dest := filepath.Join(t.TempDir(), "p")
		if _, err := videoCopyTemplate(scene, dest); err != nil {
			t.Fatalf("%s: %v", scene, err)
		}
		if got := videoHasDurationSlot(dest); got != want {
			t.Errorf("%s takes a length = %v, want %v", scene, got, want)
		}
	}
}

// The length actually on disk, read off the frame the renderer reads it off —
// including in a folder scene, where the root is a div that wraps beats which
// carry the same pair of attributes one level in.
func TestTheProjectsRealLengthIsReadFromItsRoot(t *testing.T) {
	for scene, want := range map[string]float64{
		"statement-title":    3.0,
		"minimal-hero":       3.8,
		"product-launch-30s": 30,
	} {
		dest := filepath.Join(t.TempDir(), "p")
		if _, err := videoCopyTemplate(scene, dest); err != nil {
			t.Fatalf("%s: %v", scene, err)
		}
		if got := videoRootDuration(dest); got != want {
			t.Errorf("%s is %g seconds on disk, want %g", scene, got, want)
		}
	}
}

// A framed still scene reports the length it was given, because that one really
// was the caller's to choose.
func TestAFramedStillSceneReportsTheLengthItWasGiven(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "cover")
	if _, err := videoCopyTemplate("social-cover-editorial", dest); err != nil {
		t.Fatal(err)
	}
	framed, unframed, err := videoEnsureCompositionRoot(dest, "cover", 7)
	if err != nil {
		t.Fatal(err)
	}
	if !framed {
		t.Fatalf("the cover was not framed: %s", unframed)
	}
	if got := videoRootDuration(dest); got != 7 {
		t.Errorf("the frame says %g seconds, want 7", got)
	}
}

// The sentence itself: a scene whose length is fixed says so, in the reply that
// created it, rather than at the end of a render nobody watched.
func TestNewSaysWhenTheAskedLengthWasNotUsed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", root)
	s := &videoToolSkill{app: seed(&App{cfg: config.Config{SandboxRoot: root}}, newConversation())}

	out, err := s.run(t.Context(), map[string]any{
		"action":   "new",
		"template": "statement-title",
		"path":     "a-title",
		"seconds":  20,
	})
	if err != nil {
		t.Fatalf("video new: %v", err)
	}
	if !strings.Contains(out.Content, "20") || !strings.Contains(out.Content, "data-duration") {
		t.Errorf("the reply does not say the asked length was unused:\n%s", out.Content)
	}
	if !strings.Contains(out.Content, "3") {
		t.Errorf("the reply does not say what the scene's real length is:\n%s", out.Content)
	}

	// And a scene that does take the length says nothing of the kind.
	out, err = s.run(t.Context(), map[string]any{
		"action":   "new",
		"template": "structured-grid",
		"path":     "a-grid",
		"seconds":  20,
	})
	if err != nil {
		t.Fatalf("video new: %v", err)
	}
	if strings.Contains(out.Content, "ไม่ได้ถูกใช้") {
		t.Errorf("an as-asked scene was told its length was ignored:\n%s", out.Content)
	}
	body, err := os.ReadFile(filepath.Join(root, "a-grid", "index.html"))
	if err == nil && strings.Contains(string(body), "__VIDEO_DURATION__") {
		t.Error("the duration slot was left unfilled")
	}
}

// And the signature stops advertising it, because a parameter that has been
// advertised gets tried. The properties map still carries `seconds` — `new`
// uses it — so this asserts the sentence the model reads, not the schema.
func TestRenderNoLongerAdvertisesALengthItCannotChange(t *testing.T) {
	s := &videoToolSkill{}
	def := s.ToolDefinition()
	desc := def.Function.Description
	for _, line := range strings.Split(desc, "\n") {
		if !strings.HasPrefix(line, "`render`") {
			continue
		}
		if strings.Contains(line, "seconds?") {
			t.Errorf("render still offers seconds, which nothing reads: %s", line)
		}
		if !strings.Contains(line, "data-duration") {
			t.Errorf("render does not say where length actually lives: %s", line)
		}
		return
	}
	t.Fatal("no render line in the tool description")
}
