package skill

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/imagegen"
)

// fakeDrawEngine stands in for a vendor. It writes whatever bytes it was given, so
// a test can hand back a real picture, a lie, or an error without a network.
type fakeDrawEngine struct {
	ext    string
	body   []byte
	err    error
	prompt string
	req    imagegen.Request
}

func (f *fakeDrawEngine) ID() string  { return "fake" }
func (f *fakeDrawEngine) Ext() string { return f.ext }
func (f *fakeDrawEngine) Generate(_ context.Context, prompt string, req imagegen.Request, outPath string) error {
	f.prompt, f.req = prompt, req
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(outPath, f.body, 0o644)
}

func oneTinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.RGBA{R: 200, G: 60, B: 40, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func drawSkill(t *testing.T, eng *fakeDrawEngine) (*imageMakeSkill, string) {
	t.Helper()
	root := t.TempDir()
	return &imageMakeSkill{
		root:      root,
		newEngine: func(imagegen.Options) (imagegen.Engine, error) { return eng, nil },
	}, root
}

func TestImageMakeWritesThePictureAndReportsIt(t *testing.T) {
	eng := &fakeDrawEngine{ext: ".png", body: oneTinyPNG(t)}
	s, root := drawSkill(t, eng)

	out, err := s.ExecuteTool(context.Background(), map[string]any{
		"prompt": "an orange cat on a cardboard box",
		"path":   "art/cat.png",
		"width":  float64(640),
		"height": float64(480),
	})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "art", "cat.png")); statErr != nil {
		t.Fatalf("the picture is not on disk: %v", statErr)
	}
	// The engine is handed the prompt and the size verbatim — nothing here
	// reinterprets what the model asked for.
	if eng.prompt != "an orange cat on a cardboard box" {
		t.Errorf("prompt reached the engine as %q", eng.prompt)
	}
	if eng.req.Width != 640 || eng.req.Height != 480 {
		t.Errorf("size reached the engine as %dx%d", eng.req.Width, eng.req.Height)
	}
	content := out.Content
	if !strings.Contains(content, "art/cat.png") {
		t.Errorf("the receipt does not name the file: %q", content)
	}
	// The dimensions come off the bytes, not off the request — proof the file
	// was actually looked at rather than assumed.
	if !strings.Contains(content, "3x2 px") {
		t.Errorf("the receipt does not carry the real dimensions: %q", content)
	}
	if !strings.Contains(content, "AI") {
		t.Errorf("the receipt does not say the picture was generated: %q", content)
	}
}

func TestImageMakeCorrectsAnExtensionTheEngineCannotHonour(t *testing.T) {
	// The model asks for .png because that is what pictures are called; the
	// engine only makes JPEG. A .png that is a JPEG opens nowhere.
	jpg := []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")
	eng := &fakeDrawEngine{ext: ".jpg", body: jpg}
	s, root := drawSkill(t, eng)

	out, err := s.ExecuteTool(context.Background(), map[string]any{"prompt": "x", "path": "hero.png"})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "hero.jpg")); statErr != nil {
		t.Fatalf("the corrected name is not on disk: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "hero.png")); statErr == nil {
		t.Error("the wrong name was written too")
	}
	if content := out.Content; !strings.Contains(content, "hero.jpg") || !strings.Contains(content, "นามสกุล") {
		t.Errorf("the receipt does not explain the rename: %q", content)
	}
}

func TestImageMakeRefusesAndRemovesSomethingThatIsNotAPicture(t *testing.T) {
	// The failure a keyless endpoint actually has, arriving one layer down:
	// bytes that are not a picture, already written to the destination.
	eng := &fakeDrawEngine{ext: ".jpg", body: []byte("<html>rate limited</html>")}
	s, root := drawSkill(t, eng)

	_, err := s.ExecuteTool(context.Background(), map[string]any{"prompt": "x", "path": "nope.jpg"})
	if err == nil {
		t.Fatal("an HTML body was accepted as a picture")
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Errorf("the impostor was left on disk: %d entries", len(entries))
	}
}

func TestImageMakeRefusesAnMP3EvenThoughTheSnifferKnowsIt(t *testing.T) {
	// sniffMediaKind also answers for sound. A picture tool that accepted an
	// MP3 because the sniffer recognised it would be a check that never says no.
	eng := &fakeDrawEngine{ext: ".jpg", body: []byte("ID3\x03\x00\x00\x00\x00\x00\x00mp3 body here")}
	s, _ := drawSkill(t, eng)

	if _, err := s.ExecuteTool(context.Background(), map[string]any{"prompt": "x", "path": "a.jpg"}); err == nil {
		t.Fatal("a sound file was accepted by the picture tool")
	}
}

func TestImageMakeNeedsBothAPromptAndAPath(t *testing.T) {
	s, _ := drawSkill(t, &fakeDrawEngine{ext: ".jpg", body: oneTinyPNG(t)})
	for _, args := range []map[string]any{
		{"prompt": "x"},
		{"path": "a.jpg"},
		{"prompt": "  ", "path": "a.jpg"},
	} {
		if _, err := s.ExecuteTool(context.Background(), args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestWithExtReplacesRatherThanAppends(t *testing.T) {
	for _, c := range []struct{ path, want string }{
		{"hero.png", "hero.jpg"},
		{"hero.jpg", "hero.jpg"},
		{"hero.JPG", "hero.JPG"}, // already right; casing is the user's
		{"hero", "hero.jpg"},
		{"art/a.b/hero.webp", "art/a.b/hero.jpg"},
	} {
		if got := withExt(c.path, ".jpg"); got != c.want {
			t.Errorf("withExt(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestTheBytesOverruleTheEnginesOwnDeclaration(t *testing.T) {
	// The case Ext() cannot cover: a row that declares PNG handed a JPEG,
	// which is exactly what the dall-e path does when it fetches whatever sits
	// at the URL it was given. The name must follow the bytes, not the claim.
	jpg := []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")
	eng := &fakeDrawEngine{ext: ".png", body: jpg}
	s, root := drawSkill(t, eng)

	out, err := s.ExecuteTool(context.Background(), map[string]any{"prompt": "x", "path": "hero.png"})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "hero.jpg")); statErr != nil {
		t.Fatalf("the name did not follow the bytes: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "hero.png")); statErr == nil {
		t.Error("the engine's wrong claim was left on disk as a second file")
	}
	if content := out.Content; !strings.Contains(content, "hero.jpg") {
		t.Errorf("the receipt names the wrong file: %q", content)
	}
}

func TestAnHonestEngineIsNotRenamedForNothing(t *testing.T) {
	// The other half of the rule: when the declaration and the bytes agree and
	// the caller asked for that extension, nothing is renamed and the receipt
	// says nothing about it.
	eng := &fakeDrawEngine{ext: ".png", body: oneTinyPNG(t)}
	s, _ := drawSkill(t, eng)

	out, err := s.ExecuteTool(context.Background(), map[string]any{"prompt": "x", "path": "a.png"})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	if strings.Contains(out.Content, "นามสกุล") {
		t.Errorf("a rename was reported when none happened: %q", out.Content)
	}
}

func TestTheReceiptHandsOverALineThatActuallyResolves(t *testing.T) {
	// A picture that was made and then only NAMED is a picture nobody sees —
	// the third real call answered with the path in inline code and no image.
	// The receipt now carries the finished markdown, and the two things the
	// model gets wrong writing it itself have to already be right in it: the
	// placed path, and the extension AFTER the byte-driven correction.
	jpg := []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")
	eng := &fakeDrawEngine{ext: ".png", body: jpg}
	s, _ := drawSkill(t, eng)

	out, err := s.ExecuteTool(context.Background(), map[string]any{
		"prompt": "an orange cat",
		"path":   "art/hero.png",
	})
	if err != nil {
		t.Fatalf("ExecuteTool: %v", err)
	}
	want := "![an orange cat](art/hero.jpg)"
	if !strings.Contains(out.Content, want) {
		t.Errorf("the receipt does not carry %q:\n%s", want, out.Content)
	}
	// And never the name that was asked for, which is the whole failure.
	if strings.Contains(out.Content, "](art/hero.png)") {
		t.Errorf("the receipt offers the stale name:\n%s", out.Content)
	}
}

func TestAltTextIsADescriptionNotAParagraph(t *testing.T) {
	// Alt text is read aloud and shown when the picture cannot load.
	long := strings.Repeat("a very detailed cyberpunk scene ", 12)
	got := altText(long)
	if len(got) > 80 {
		t.Errorf("alt text is %d chars, want it trimmed: %q", len(got), got)
	}
	if strings.HasSuffix(got, " ") {
		t.Errorf("alt text ends on a space: %q", got)
	}
	// A bracket would close the markdown alt early and spill the rest as prose.
	if strings.ContainsAny(altText("a [cat] in a box"), "[]") {
		t.Error("brackets survived into the alt text")
	}
	// Newlines and runs of spaces collapse — a prompt is often multi-line.
	if got := altText("  a cat\n  in   a box  "); got != "a cat in a box" {
		t.Errorf("altText did not collapse whitespace: %q", got)
	}
}
