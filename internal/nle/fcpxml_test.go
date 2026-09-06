package nle

import (
	"encoding/xml"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// absPath is a real absolute path for whichever machine the test runs on,
// with a space in it — the character that separates a URL built with net/url
// from one built by joining strings.
func absPath(name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(`D:\my footage`, name)
	}
	return filepath.Join("/home/me/my footage", name)
}

func twoClips() Timeline {
	src := Source{
		ID: "raw", Name: "raw.mp4", Path: absPath("raw.mp4"),
		Seconds: 12, Width: 1920, Height: 1080,
		Rate: Rate{Num: 30, Den: 1}, HasAudio: true, AudioChannels: 2,
	}
	return Timeline{
		Name: "rough cut", Width: 1920, Height: 1080,
		Rate:    Rate{Num: 30, Den: 1},
		Sources: []Source{src},
		Clips: []Clip{
			{Source: "raw", Name: "the answer", In: 2, Out: 6},
			{Source: "raw", Name: "the close", In: 10, Out: 11.5},
		},
	}
}

func TestPlaceIsExactFrames(t *testing.T) {
	placed, err := twoClips().Place()
	if err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	if len(placed) != 2 {
		t.Fatalf("placed %d clips, want 2", len(placed))
	}
	// 2s..6s at 30fps is frame 60 for 120 frames, landing at 0.
	if placed[0].InFrames != 60 || placed[0].LenFrames != 120 || placed[0].AtFrames != 0 {
		t.Errorf("clip 1 = in %d len %d at %d, want 60/120/0",
			placed[0].InFrames, placed[0].LenFrames, placed[0].AtFrames)
	}
	// 10s..11.5s is frame 300 for 45 frames, and it lands where clip 1 ended.
	if placed[1].InFrames != 300 || placed[1].LenFrames != 45 || placed[1].AtFrames != 120 {
		t.Errorf("clip 2 = in %d len %d at %d, want 300/45/120",
			placed[1].InFrames, placed[1].LenFrames, placed[1].AtFrames)
	}
}

func TestFCPXMLIsWellFormedAndDeclaresItself(t *testing.T) {
	out, err := twoClips().FCPXML()
	if err != nil {
		t.Fatalf("FCPXML() error = %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "<!DOCTYPE fcpxml>") {
		t.Error("no DOCTYPE — Final Cut and Resolve both look for it")
	}
	if !strings.Contains(text, `version="1.10"`) {
		t.Error("no fcpxml version attribute")
	}
	var doc fcpxml
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not well-formed XML: %v", err)
	}
	if got := len(doc.Library.Event.Project.Sequence.Spine.Clips); got != 2 {
		t.Fatalf("spine holds %d clips, want 2", got)
	}
}

// Every time in the file has to be a whole number of frames over one
// denominator. This is the property the whole package exists to keep.
func TestEveryTimeIsAWholeFrame(t *testing.T) {
	out, err := twoClips().FCPXML()
	if err != nil {
		t.Fatalf("FCPXML() error = %v", err)
	}
	var doc fcpxml
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	seq := doc.Library.Event.Project.Sequence
	// 30fps scales to 100/3000s, so every time is a multiple of 100 over 3000.
	if got := doc.Resources.Formats[0].FrameDuration; got != "100/3000s" {
		t.Errorf("frameDuration = %q, want 100/3000s", got)
	}
	for _, v := range []struct{ name, value string }{
		{"sequence duration", seq.Duration},
		{"clip 1 offset", seq.Spine.Clips[0].Offset},
		{"clip 1 start", seq.Spine.Clips[0].Start},
		{"clip 1 duration", seq.Spine.Clips[0].Duration},
		{"clip 2 offset", seq.Spine.Clips[1].Offset},
		{"clip 2 start", seq.Spine.Clips[1].Start},
		{"clip 2 duration", seq.Spine.Clips[1].Duration},
		{"asset duration", doc.Resources.Assets[0].Duration},
	} {
		if v.value == "0s" {
			continue
		}
		num, den, ok := strings.Cut(strings.TrimSuffix(v.value, "s"), "/")
		if !ok || den != "3000" {
			t.Errorf("%s = %q, want <n>/3000s", v.name, v.value)
			continue
		}
		if !strings.HasSuffix(num, "00") {
			t.Errorf("%s = %q, which is not a whole frame at 100/3000s", v.name, v.value)
		}
	}
	// 120 frames then 45: 165 frames total, 16500/3000s.
	if seq.Duration != "16500/3000s" {
		t.Errorf("sequence duration = %q, want 16500/3000s", seq.Duration)
	}
	if seq.Spine.Clips[1].Offset != "12000/3000s" {
		t.Errorf("clip 2 offset = %q, want 12000/3000s", seq.Spine.Clips[1].Offset)
	}
}

// 29.97 is the rate that catches a package that stored frame rates as floats.
func TestDropFrameRateStaysRational(t *testing.T) {
	tl := twoClips()
	tl.Rate = Rate{Num: 30000, Den: 1001}
	tl.Sources[0].Rate = Rate{Num: 30000, Den: 1001}
	out, err := tl.FCPXML()
	if err != nil {
		t.Fatalf("FCPXML() error = %v", err)
	}
	var doc fcpxml
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := doc.Resources.Formats[0].FrameDuration; got != "1001/30000s" {
		t.Errorf("frameDuration = %q, want 1001/30000s", got)
	}
	// 2s at 30000/1001 is frame 60 (59.94 rounded), so 60*1001 = 60060.
	if got := doc.Library.Event.Project.Sequence.Spine.Clips[0].Start; got != "60060/30000s" {
		t.Errorf("clip 1 start = %q, want 60060/30000s", got)
	}
}

func TestFileURLEncodesTheSpace(t *testing.T) {
	out, err := twoClips().FCPXML()
	if err != nil {
		t.Fatalf("FCPXML() error = %v", err)
	}
	var doc fcpxml
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	src := doc.Resources.Assets[0].MediaRep.Src
	if !strings.HasPrefix(src, "file:///") {
		t.Errorf("media-rep src = %q, want three slashes after file:", src)
	}
	if strings.Contains(src, " ") {
		t.Errorf("media-rep src = %q, still holds a raw space", src)
	}
	if !strings.Contains(src, "my%20footage") {
		t.Errorf("media-rep src = %q, want the space percent-encoded", src)
	}
	if strings.Contains(src, `\`) {
		t.Errorf("media-rep src = %q, still holds a backslash", src)
	}
}

// A source that is not the sequence's shape must not be declared as the
// sequence's shape, or the importer scales it without being asked.
func TestOffFormatSourceGetsItsOwnFormat(t *testing.T) {
	tl := twoClips()
	tl.Sources[0].Width, tl.Sources[0].Height = 3840, 2160
	out, err := tl.FCPXML()
	if err != nil {
		t.Fatalf("FCPXML() error = %v", err)
	}
	var doc fcpxml
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(doc.Resources.Formats) != 2 {
		t.Fatalf("declared %d formats, want 2 (sequence and source)", len(doc.Resources.Formats))
	}
	if doc.Resources.Assets[0].Format == doc.Library.Event.Project.Sequence.Format {
		t.Error("the 4K asset was declared in the 1080p sequence format")
	}
}

func TestAudioIsDeclaredOnlyWhenThereIsSome(t *testing.T) {
	tl := twoClips()
	tl.Sources[0].HasAudio = false
	out, err := tl.FCPXML()
	if err != nil {
		t.Fatalf("FCPXML() error = %v", err)
	}
	var doc fcpxml
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.Resources.Assets[0].HasAudio != "" {
		t.Error("a silent source was declared as having audio")
	}
}

func TestRefusesWhatIsNotAnEdit(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Timeline)
		want string
	}{
		{"no clips", func(t *Timeline) { t.Clips = nil }, "no clips"},
		{"unknown source", func(t *Timeline) { t.Clips[0].Source = "nope" }, "not declared"},
		{"backwards clip", func(t *Timeline) { t.Clips[0].In, t.Clips[0].Out = 6, 2 }, "before it starts"},
		{"no rate", func(t *Timeline) { t.Rate = Rate{} }, "frame rate"},
		{"no size", func(t *Timeline) { t.Width = 0 }, "frame size"},
		{"no name", func(t *Timeline) { t.Name = "  " }, "no name"},
		{"relative source", func(t *Timeline) { t.Sources[0].Path = "media/raw.mp4" }, "not absolute"},
		{"sub-frame clip", func(t *Timeline) { t.Clips[0].Out = t.Clips[0].In + 0.001 }, "shorter than one frame"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tl := twoClips()
			tc.mut(&tl)
			_, err := tl.FCPXML()
			if err == nil {
				t.Fatalf("FCPXML() returned no error, want one about %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// The report a user reads and the file they open must agree, which they only
// do if both come from Place.
func TestSecondsRoundTripThroughFrames(t *testing.T) {
	tl := twoClips()
	placed, err := tl.Place()
	if err != nil {
		t.Fatalf("Place() error = %v", err)
	}
	if got := tl.Seconds(placed[0].LenFrames); got != 4 {
		t.Errorf("clip 1 length = %v seconds, want 4", got)
	}
	if got := tl.Seconds(placed[1].AtFrames); got != 4 {
		t.Errorf("clip 2 lands at %v seconds, want 4", got)
	}
}
