package main

// What `video new` can actually reach, and what it does to a scene that did not
// bring its own frame.
//
// Both halves are here because both were the same defect wearing two hats: the
// shelf's own index taught twenty-five names the tool refused, and the reason
// they were refused — a still scene has no composition root, so the renderer
// makes it 1080x1920 and crops it — was a real problem nobody had solved rather
// than a decision worth keeping.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/subagent"
)

// The four shelves hold seventy-five scenes and every one of them is reachable
// by the spelling SKILL.md prints. The count is pinned for the reason the
// bundled-profile counts are: a number that moves here should be a scene
// somebody added, never a shelf that stopped listing.
func TestEveryShelfInTheLibraryIsReachable(t *testing.T) {
	total := 0
	for _, shelf := range videoShelves {
		names, err := subagent.ListSkillDir(videoLibraryAgent, videoLibrarySkill, shelf)
		if err != nil {
			t.Fatalf("shelf %s does not list: %v", shelf, err)
		}
		if len(names) == 0 {
			t.Errorf("shelf %s is empty", shelf)
		}
		total += len(names)
	}
	if total != 75 {
		t.Errorf("the library holds %d scenes, want 75", total)
	}
	if len(videoShelves) != 4 {
		t.Errorf("%d shelves, want 4", len(videoShelves))
	}
}

// The spelling the shelf's own tables print, which is the spelling agents copy
// verbatim. Every one of these used to be a refusal.
func TestTheSpellingsSKILLMDPrintsAllResolve(t *testing.T) {
	for _, spelling := range []string{
		"graphic-scenes/social-cover-editorial.html",
		"graphic-scenes/social-cover-editorial",
		"social-cover-editorial",
		"slide-scenes/data-slide-organic.html",
		"web-scenes/saas-landing-minimal.html",
		"api-docs-editorial",
		"graphic-scenes/explainer-diagram-poster.html",
		"motion/minimal-hero.html",
		"minimal-hero",
		"motion/product-launch-30s/index.html",
		"product-launch-30s",
	} {
		dest := filepath.Join(t.TempDir(), "project")
		n, err := videoCopyTemplate(spelling, dest)
		if err != nil {
			t.Errorf("%s: %v", spelling, err)
			continue
		}
		if n == 0 {
			t.Errorf("%s copied nothing", spelling)
		}
		if _, err := os.Stat(filepath.Join(dest, "index.html")); err != nil {
			t.Errorf("%s left no index.html: %v", spelling, err)
		}
	}
}

// A shelf that does not exist and a path that goes deeper than a scene are
// still refused, and the refusal names the shelves rather than the seventy-five.
func TestAPathThatIsNotASceneIsStillRefused(t *testing.T) {
	for _, bad := range []string{
		"nowhere/minimal-hero",
		"motion/product-launch-30s/compositions/01-problem-type.html",
		"../../../etc/passwd",
	} {
		dest := filepath.Join(t.TempDir(), "project")
		if _, err := videoCopyTemplate(bad, dest); err == nil {
			t.Errorf("%s was accepted", bad)
		}
	}
}

// The canvas every still scene draws itself at, read out of the scene rather
// than measured by hand. These are the five frames the shelf actually uses.
func TestAStillScenesCanvasIsReadFromTheSceneItself(t *testing.T) {
	for _, want := range []struct {
		scene         string
		width, height int
	}{
		{"graphic-scenes/social-cover-editorial", 1200, 510},
		{"graphic-scenes/vertical-infographic-organic", 1080, 1920},
		{"graphic-scenes/explainer-diagram-poster", 1200, 1581},
		{"slide-scenes/data-slide-minimal", 1920, 1080},
		{"web-scenes/writing-tool-organic", 1440, 900},
	} {
		data, err := subagent.ReadSkillFile(videoLibraryAgent, videoLibrarySkill, want.scene+".html")
		if err != nil {
			t.Fatalf("%s: %v", want.scene, err)
		}
		w, h := videoStillCanvas(string(data))
		if w != want.width || h != want.height {
			t.Errorf("%s reads %dx%d, want %dx%d", want.scene, w, h, want.width, want.height)
		}
	}
}

// The frame a still scene gets on the way into a project, and the frame a
// motion scene does not get because it already has one.
func TestACopiedStillSceneGetsTheFrameTheRendererNeeds(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "cover")
	if _, err := videoCopyTemplate("social-cover-editorial", dest); err != nil {
		t.Fatal(err)
	}
	framed, unframed, err := videoEnsureCompositionRoot(dest, "cover", 6)
	if err != nil {
		t.Fatal(err)
	}
	if !framed {
		t.Fatalf("a still scene came away with no composition root: %s", unframed)
	}
	body, err := os.ReadFile(filepath.Join(dest, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-composition-id="cover"`,
		"data-no-timeline",
		`data-width="1200"`,
		`data-height="510"`,
		`data-duration="6"`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("the frame is missing %s", want)
		}
	}

	// Twice is once. A project reopened must not collect a second frame.
	again, againWhy, err := videoEnsureCompositionRoot(dest, "cover", 6)
	if err != nil {
		t.Fatal(err)
	}
	if again {
		t.Error("a scene that already had a frame was framed again")
	}
	// And it says nothing about it, because "you already had one" is not a
	// complaint. The reason string is reserved for a scene that has no frame
	// and could not be given one, which is the case that used to be silent.
	if againWhy != "" {
		t.Errorf("a scene that brought its own frame was reported as unframed: %s", againWhy)
	}

	// A motion scene decided its own frame; leave it exactly as it was.
	motion := filepath.Join(t.TempDir(), "hero")
	if _, err := videoCopyTemplate("minimal-hero", motion); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(motion, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if framed, why, err := videoEnsureCompositionRoot(motion, "hero", 6); err != nil || framed || why != "" {
		t.Fatalf("videoEnsureCompositionRoot on a motion scene = %v, %q, %v; want false, \"\", nil", framed, why, err)
	}
	after, _ := os.ReadFile(filepath.Join(motion, "index.html"))
	if string(before) != string(after) {
		t.Error("a motion scene was rewritten")
	}
}

// Every one of the twenty-five still scenes can be copied and framed. Written
// as a sweep rather than a sample because the canvas is read by a pattern, and
// a pattern that works on five files and not on the sixth fails silently — the
// project renders 1080x1920 and crops, with nothing in the output saying why.
func TestAllTwentyFiveStillScenesCanBeFramed(t *testing.T) {
	stills := 0
	for _, shelf := range videoShelves {
		if shelf == videoLibraryDir {
			continue
		}
		names, err := subagent.ListSkillDir(videoLibraryAgent, videoLibrarySkill, shelf)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			scene := strings.TrimSuffix(name, ".html")
			dest := filepath.Join(t.TempDir(), scene)
			if _, err := videoCopyTemplate(shelf+"/"+scene, dest); err != nil {
				t.Errorf("%s/%s: %v", shelf, scene, err)
				continue
			}
			framed, unframed, err := videoEnsureCompositionRoot(dest, scene, 5)
			if err != nil {
				t.Errorf("%s/%s: %v", shelf, scene, err)
				continue
			}
			if !framed {
				t.Errorf("%s/%s came away with no composition root: %s", shelf, scene, unframed)
				continue
			}
			stills++
		}
	}
	if stills != 25 {
		t.Errorf("framed %d still scenes, want 25", stills)
	}
}

// The blank start, and the sentence it gives when the renderer is not here.
//
// `blank` is not on any shelf: it is the engine's own empty composition, which
// is the only copy of the renderer's contract guaranteed to match the renderer
// installed beside it. With no bundle there is nothing to copy, and the answer
// has to be that sentence rather than a file we invented.
func TestTheBlankStartComesFromTheEngineOrSaysWhyNot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", root)

	dest := filepath.Join(t.TempDir(), "scratch")
	_, err := videoCopyTemplate("blank", dest)
	if err == nil {
		t.Fatal("blank produced a project with no renderer installed")
	}
	if !strings.Contains(err.Error(), "ตัวเรนเดอร์") {
		t.Errorf("the refusal does not say the renderer is missing: %v", err)
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("a refused blank left a folder behind")
	}
}
