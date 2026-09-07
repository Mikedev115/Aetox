package main

// The two questions `video new` and `video render` answer wrongly when nobody
// is holding them to it: how big is this scene, and what is this tool allowed to
// ask the renderer for.
//
// Every test here was written against the code as it stood on 7 ก.ย. 2569 and
// fails on it. A regression test for a defect that has never been seen failing
// is a test of nothing.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The decoy the 240px floor could never have caught, and neither could a check
// against the page width: a hero exactly as wide as the page, 72px short of it,
// written above the rule that actually sets the canvas.
//
// Taken from `portfolio-home-organic`, which really does carry a 1440x828 block
// beside its 1440x900 body. On the shelf today the body rule happens to come
// first, so the old first-match-wins read it correctly; this is the same file
// with the two swapped, which is one library refresh away from being real.
func TestTheCanvasIsTheBodyRuleNotTheBiggestBoxAboveIt(t *testing.T) {
	const scene = `<!doctype html><html><head>
<meta name="viewport" content="width=1440">
<style>
  .hero-panel { width: 1440px; height: 828px; overflow: hidden; }
  .tile { width: 344px; height: 344px; }
  body { width: 1440px; height: 900px; overflow: hidden; margin: 0; }
</style></head><body></body></html>`

	w, h := videoStillCanvas(scene)
	if w != 1440 || h != 900 {
		t.Errorf("the canvas reads %dx%d, want 1440x900 — a hero won the frame", w, h)
	}
}

// No body rule and a viewport that names a width: the width still has to match
// the page's own declaration, so a sidebar cannot take the frame by being first.
func TestWithoutABodyRuleTheViewportWidthStillHasToMatch(t *testing.T) {
	const scene = `<!doctype html><html><head>
<meta name="viewport" content="width=1440">
<style>
  .sidebar { width: 260px; height: 900px; }
  .page { width: 1440px; height: 900px; }
</style></head><body></body></html>`

	w, h := videoStillCanvas(scene)
	if w != 1440 || h != 900 {
		t.Errorf("the canvas reads %dx%d, want 1440x900", w, h)
	}
}

// A scene that says neither is not a scene this can measure, and the answer to
// that is a sentence rather than a shrug. Until 8 ก.ย. 2569 "I gave it a frame"
// and "I could not" were both `false, nil`, so a project that was about to
// render at 1080x1920 and be cropped came back from `video new` reading exactly
// like one that had brought its own frame.
func TestAnUnmeasurableSceneSaysSoInsteadOfGoingQuiet(t *testing.T) {
	dir := t.TempDir()
	const scene = `<!doctype html><html><head>
<meta name="viewport" content="width=device-width">
<style>.chip { width: 80px; height: 24px; }</style>
</head><body><p>hello</p></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(scene), 0o644); err != nil {
		t.Fatal(err)
	}

	framed, unframed, err := videoEnsureCompositionRoot(dir, "mystery", 5)
	if err != nil {
		t.Fatal(err)
	}
	if framed {
		t.Fatal("a scene with no measurable canvas was framed anyway")
	}
	if unframed == "" {
		t.Fatal("a scene that could not be framed said nothing about it")
	}

	// And the file is left exactly as it arrived: this refuses to guess, it does
	// not half-write a frame.
	after, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != scene {
		t.Error("the scene was rewritten by a pass that reported it could not be framed")
	}
}

// A mounted layer is not the project. `grain-texture-hero` mounts `intro` at a
// literal 2.5 seconds and `structured-grid` mounts one at 1.86, both written
// ABOVE a root whose own duration is still `__VIDEO_DURATION__` — so reading the
// first element with the pair in it answered with a sub-composition unless the
// placeholder patcher had already run. This is that file's shape, unpatched.
func TestTheRootDurationIgnoresAMountedSubComposition(t *testing.T) {
	dir := t.TempDir()
	const project = `<!doctype html><html><body>
  <div id="intro-layer" data-composition-id="intro"
       data-composition-src="compositions/intro.html"
       data-start="0" data-duration="2.5"></div>
  <div id="main" data-composition-id="main-video"
       data-start="0" data-duration="10" data-width="1920" data-height="1080"></div>
</body></html>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(project), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := videoRootDuration(dir); got != 10 {
		t.Errorf("the project reads %g seconds, want 10 — a mounted layer answered for the whole", got)
	}
}

// What came out of a render, in the two shapes a render has. A png-sequence is a
// folder of frames, and a size check against a folder reads 0 on Windows, which
// would have called every successful frame export a render that produced
// nothing.
func TestWhatARenderProducedIsReadPerFormat(t *testing.T) {
	dir := t.TempDir()

	clip := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(clip, make([]byte, 2<<20), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := videoRenderProduced(clip, "mp4")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "MB") {
		t.Errorf("a clip reports %q, want a size in MB", got)
	}

	empty := filepath.Join(dir, "empty.mp4")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := videoRenderProduced(empty, "mp4"); err == nil {
		t.Error("an empty file passed as a finished render")
	}

	frames := filepath.Join(dir, "out-frames")
	if err := os.MkdirAll(frames, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := videoRenderProduced(frames, "png-sequence"); err == nil {
		t.Error("a folder with no frames in it passed as a finished sequence")
	}
	for _, name := range []string{"0001.png", "0002.png"} {
		if err := os.WriteFile(filepath.Join(frames, name), []byte{0}, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err = videoRenderProduced(frames, "png-sequence")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "2") {
		t.Errorf("a two-frame sequence reports %q", got)
	}
}

// The schema is what the model actually reads when it picks arguments, so it is
// where the truth about an argument has to be. Two claims here, and both were
// false on 7 ก.ย. 2569.
func TestTheSchemaSaysWhatTheRendererReallyTakes(t *testing.T) {
	var schema struct {
		Properties map[string]struct {
			Type        string   `json:"type"`
			Enum        []string `json:"enum"`
			Description string   `json:"description"`
		} `json:"properties"`
	}
	def := (&videoToolSkill{}).ToolDefinition()
	if err := json.Unmarshal(def.Function.Parameters, &schema); err != nil {
		t.Fatal(err)
	}

	// `seconds` cannot leave the schema while `new` needs it, so the schema has
	// to say whose it is. The prose line said "Length is not a render option"
	// while this block went on offering the parameter with nothing attached.
	if d := schema.Properties["seconds"].Description; !strings.Contains(d, "new") {
		t.Errorf("seconds does not say which action reads it: %q", d)
	}

	// Our own ceiling, not the engine's: `hyperframes render --help` documents
	// "24000/1001 for 23.976" and this said `"type": "integer"`, so every rate
	// film arrives at was unreachable.
	if got := schema.Properties["fps"].Type; got == "integer" {
		t.Error("fps is typed integer, which locks out the rational rates the engine documents")
	}

	// Forwarded four of the engine's forty options until 8 ก.ย. 2569. These two
	// are the ones that change what the work can be rather than how fast it
	// encodes.
	for name, want := range map[string][]string{
		"resolution": videoResolutions,
		"format":     videoFormats,
	} {
		got := schema.Properties[name].Enum
		if len(got) != len(want) {
			t.Errorf("%s offers %v, want %v", name, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s offers %v, want %v", name, got, want)
				break
			}
		}
	}
	if _, ok := schema.Properties["variables"]; !ok {
		t.Error("variables is not offered — one composition with different words has no way through the tool")
	}

	// 4K is the headline of the unlock: it is available on this machine today,
	// through a flag that leaves the composition's CSS untouched.
	if !strings.Contains(schema.Properties["resolution"].Description, "3840x2160") {
		t.Error("the resolution description does not name the 4K size it unlocks")
	}
}

// mp4 by name, and every other container by its own name. A webm written to a
// path ending .mp4 is a file this machine's players refuse.
func TestTheDefaultOutputNameFollowsTheFormat(t *testing.T) {
	for format, want := range map[string]string{
		"mp4":          ".mp4",
		"webm":         ".webm",
		"mov":          ".mov",
		"gif":          ".gif",
		"png-sequence": "-frames",
	} {
		if got := videoFormatSuffix[format]; got != want {
			t.Errorf("%s names its output %q, want %q", format, got, want)
		}
	}
}
