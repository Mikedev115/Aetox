package skill

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/nle"
	"github.com/Mikedev115/Aetox/internal/proc"
)

func planFrom(t *testing.T, doc string) cutfilePlan {
	t.Helper()
	var cf cutfileDoc
	if err := json.Unmarshal([]byte(doc), &cf); err != nil {
		t.Fatalf("fixture is not valid json: %v", err)
	}
	plan, err := readCutfilePlan(cf)
	if err != nil {
		t.Fatalf("readCutfilePlan() error = %v", err)
	}
	return plan
}

func TestPlanReadsTrimsAsClips(t *testing.T) {
	plan := planFrom(t, `{"name":"c","sources":[{"id":"raw","path":"media/raw.mp4"}],
		"ops":[{"op":"trim","src":"raw","start":2,"duration":4,"id":"a"},
		       {"op":"trim","src":"raw","start":10,"duration":1.5,"id":"b"}]}`)
	if len(plan.clips) != 2 {
		t.Fatalf("read %d clips, want 2", len(plan.clips))
	}
	if plan.clips[0].In != 2 || plan.clips[0].Out != 6 {
		t.Errorf("clip 1 = %v..%v, want 2..6", plan.clips[0].In, plan.clips[0].Out)
	}
	if plan.clips[1].In != 10 || plan.clips[1].Out != 11.5 {
		t.Errorf("clip 2 = %v..%v, want 10..11.5", plan.clips[1].In, plan.clips[1].Out)
	}
}

// The merge is the running order, and reading it wrong is an edit that plays
// backwards — which is why it is pinned rather than assumed from file order.
func TestMergeSetsTheOrder(t *testing.T) {
	plan := planFrom(t, `{"name":"c","sources":[{"id":"raw","path":"raw.mp4"}],
		"ops":[{"op":"trim","src":"raw","start":10,"duration":2,"id":"late"},
		       {"op":"trim","src":"raw","start":0,"duration":2,"id":"early"},
		       {"op":"merge","srcs":["early","late"]}]}`)
	if len(plan.clips) != 2 {
		t.Fatalf("read %d clips, want 2", len(plan.clips))
	}
	if plan.clips[0].Name != "early" || plan.clips[1].Name != "late" {
		t.Errorf("order = %s, %s; want early, late", plan.clips[0].Name, plan.clips[1].Name)
	}
}

// A JSON array decodes to []any, and stringSlice only accepts []string — the
// combination that would have made every merge silently keep no order at all.
func TestMergeReadsADecodedJSONArray(t *testing.T) {
	if got := cutfileStrings([]any{"a", "", "b", 7}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("cutfileStrings([]any{...}) = %v, want [a b]", got)
	}
	if got := cutfileStrings([]string{" a ", " "}); len(got) != 1 || got[0] != "a" {
		t.Errorf("cutfileStrings([]string{...}) = %v, want [a]", got)
	}
}

func TestMergeCanJoinAWholeSource(t *testing.T) {
	plan := planFrom(t, `{"name":"c","sources":[{"id":"one","path":"a.mp4"},{"id":"two","path":"b.mp4"}],
		"ops":[{"op":"merge","srcs":["one","two"]}]}`)
	if len(plan.clips) != 2 {
		t.Fatalf("read %d clips, want 2", len(plan.clips))
	}
	// Out stays zero here on purpose: the length is ffprobe's answer, and the
	// plan is read before any file is measured.
	if plan.clips[0].Source != "one" || plan.clips[0].Out != 0 {
		t.Errorf("whole-source clip = %+v, want source one with no end yet", plan.clips[0])
	}
}

func TestResizeSetsTheSequenceFrame(t *testing.T) {
	plan := planFrom(t, `{"name":"c","sources":[{"id":"raw","path":"raw.mp4"}],
		"ops":[{"op":"trim","src":"raw","start":0,"duration":2},{"op":"resize","width":1080,"height":1920}]}`)
	if plan.width != 1080 || plan.height != 1920 {
		t.Errorf("frame = %dx%d, want 1080x1920", plan.width, plan.height)
	}
}

// Work the spine cannot carry has to be named. Dropping it silently is the
// failure this whole tool is supposed to prevent.
func TestWhatCannotTravelIsNamed(t *testing.T) {
	plan := planFrom(t, `{"name":"c","sources":[{"id":"raw","path":"raw.mp4"}],
		"ops":[{"op":"trim","src":"raw","start":0,"duration":2},
		       {"op":"add_text","text":"hello"},
		       {"op":"burn_in","subtitles":"subs.srt"},
		       {"op":"crop","x":0,"y":0,"width":10,"height":10},
		       {"op":"crop","x":1,"y":1,"width":10,"height":10}]}`)
	joined := strings.Join(plan.unrepresented, " | ")
	for _, want := range []string{"add_text", "burn_in", "crop (x2)"} {
		if !strings.Contains(joined, want) {
			t.Errorf("unrepresented = %q, want it to name %q", joined, want)
		}
	}
	if len(plan.clips) != 1 {
		t.Errorf("read %d clips, want the one trim to survive", len(plan.clips))
	}
}

// A trim of an earlier step's output is a nested edit a flat spine cannot say.
// It has to be reported, not guessed at.
func TestChainedTrimIsReportedNotGuessed(t *testing.T) {
	plan := planFrom(t, `{"name":"c","sources":[{"id":"raw","path":"raw.mp4"}],
		"ops":[{"op":"trim","src":"raw","start":0,"duration":8,"id":"a"},
		       {"op":"trim","src":"a","start":1,"duration":2,"id":"b"}]}`)
	if len(plan.clips) != 1 || plan.clips[0].Name != "a" {
		t.Fatalf("clips = %+v, want only the trim of a real source", plan.clips)
	}
	if len(plan.unrepresented) == 0 || !strings.Contains(plan.unrepresented[0], "trim") {
		t.Errorf("unrepresented = %v, want the chained trim named", plan.unrepresented)
	}
}

func TestPlanRefusesACutfileWithNoSources(t *testing.T) {
	var cf cutfileDoc
	if err := json.Unmarshal([]byte(`{"name":"c","ops":[]}`), &cf); err != nil {
		t.Fatal(err)
	}
	if _, err := readCutfilePlan(cf); err == nil {
		t.Error("readCutfilePlan() accepted a cutfile with no sources")
	}
}

func TestParseRational(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want nle.Rate
	}{
		{"30/1", nle.Rate{Num: 30, Den: 1}},
		{"30000/1001", nle.Rate{Num: 30000, Den: 1001}},
		{"25", nle.Rate{Num: 25, Den: 1}},
		{"", nle.Rate{}},
		{"nonsense", nle.Rate{}},
	} {
		if got := parseRational(tc.in); got != tc.want {
			t.Errorf("parseRational(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

// The scaffold kinocut writes is YAML, and its own loader reads a filled-in
// one as an edit with no operations. The tool has to say which file to write.
func TestYAMLCutfileSaysWriteJSONInstead(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "cutfile.yaml")
	if err := os.WriteFile(path, []byte("name: \"c\"\nsources: []\nops: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &videoProjectSkill{root: root, files: NewFileState()}
	_, err := s.run(context.Background(), time.Now(), "cutfile.yaml")
	if err == nil {
		t.Fatal("a YAML cutfile was accepted")
	}
	if !strings.Contains(err.Error(), ".json") {
		t.Errorf("error = %q, want it to say to write .json", err)
	}
}

func TestRefusesASourceOutsideTheProject(t *testing.T) {
	root := t.TempDir()
	doc := `{"name":"c","sources":[{"id":"raw","path":"../../elsewhere.mp4"}],
		"ops":[{"op":"trim","src":"raw","start":0,"duration":1}]}`
	if err := os.WriteFile(filepath.Join(root, "cut.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	s := &videoProjectSkill{root: root, files: NewFileState()}
	if _, err := s.run(context.Background(), time.Now(), "cut.json"); err == nil {
		t.Error("a source above the project root was accepted")
	}
}

// End to end on a machine with the video toolchain: a real clip, a real
// cutfile, a real .fcpxml with the edit in it.
func TestWritesAProjectFromARealClip(t *testing.T) {
	ffmpeg := bundledBinary("ffmpeg", "ffmpeg")
	if err := exec.Command(ffmpeg, "-version").Run(); err != nil {
		t.Skipf("no ffmpeg on this machine: %v", err)
	}
	root := t.TempDir()
	clip := filepath.Join(root, "raw.mp4")
	// mpeg4 rather than h264: it is in every build, including the LGPL one
	// Aetox ships for reading media.
	build := exec.Command(ffmpeg, "-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=320x180:rate=30:duration=6",
		"-c:v", "mpeg4", clip)
	proc.HideConsole(build)
	if err := build.Run(); err != nil {
		t.Skipf("could not build a clip to edit: %v", err)
	}

	doc := `{"name":"rough","sources":[{"id":"raw","path":"raw.mp4"}],
		"ops":[{"op":"trim","src":"raw","start":1,"duration":2,"id":"open"},
		       {"op":"trim","src":"raw","start":4,"duration":1,"id":"close"},
		       {"op":"merge","srcs":["open","close"]},
		       {"op":"add_text","text":"left behind"}]}`
	if err := os.WriteFile(filepath.Join(root, "cut.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &videoProjectSkill{root: root, files: NewFileState()}
	out, err := s.run(context.Background(), time.Now(), "cut.json")
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	written, err := os.ReadFile(filepath.Join(root, "cut.fcpxml"))
	if err != nil {
		t.Fatalf("no project file was written: %v", err)
	}
	text := string(written)
	for _, want := range []string{"<!DOCTYPE fcpxml>", "<asset-clip", "raw.mp4", "file://"} {
		if !strings.Contains(text, want) {
			t.Errorf("project file does not contain %q", want)
		}
	}
	if got := strings.Count(text, "<asset-clip"); got != 2 {
		t.Errorf("project holds %d clips, want 2", got)
	}
	// 2s then 1s at 30fps is 90 frames, and the report has to say so in the
	// same numbers the file does.
	report := out.Content
	if !strings.Contains(report, "2 clip(s), 0:03.000 long") {
		t.Errorf("report does not give the total length as 3 seconds:\n%s", report)
	}
	if !strings.Contains(report, "add_text") {
		t.Errorf("report does not name the operation that was left behind:\n%s", report)
	}
}
