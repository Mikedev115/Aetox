package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/cognitive"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
)

// composerLine is the exact string cockpit.svelte.ts appends for an attached
// image. Written out in full rather than built from a helper: if the composer's
// wording changes and this does not, the picture stops reaching the model and
// nothing else fails — so this literal is the contract, and this test is what
// notices when one side moves.
const composerLine = "\n\n[attachment: user-attached image — read it with image_ocr] .aetox-attachments/shot.png"

func newVisionApp(t *testing.T, provider, modelName string) (*Engine, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".aetox-attachments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// A real PNG header, so mime.TypeByExtension and any later sniffing agree.
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(dir, "shot.png"), png, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return seed(&Engine{cfg: config.Config{
		SandboxRoot:   root,
		ModelProvider: provider,
		ModelName:     modelName,
	}}, newConversation()), root
}

func TestVisionAttachmentsSendsThePictureToASightedModel(t *testing.T) {
	app, _ := newVisionApp(t, "openai", "gpt-4o")

	text, images := app.visionAttachments("why is this broken?" + composerLine)

	if len(images) != 1 {
		t.Fatalf("want one image attached, got %d", len(images))
	}
	if images[0].MediaType != "image/png" || len(images[0].Data) == 0 {
		t.Errorf("image = %+v, want a png with bytes", model.Image{MediaType: images[0].MediaType})
	}
	if strings.Contains(text, "image_ocr") {
		t.Errorf("text still tells the model to OCR a picture it is holding: %q", text)
	}
	// The path stays: the model needs the file's name to talk about it or edit it.
	if !strings.Contains(text, ".aetox-attachments/shot.png") {
		t.Errorf("text = %q, want the path kept", text)
	}
	if !strings.Contains(text, "why is this broken?") {
		t.Errorf("text = %q, want the question kept", text)
	}
}

// The fallback, and the reason image_ocr is not being deleted: a model with no
// vision must get exactly what it got before this feature existed.
func TestVisionAttachmentsLeavesABlindModelOnTheOCRPath(t *testing.T) {
	app, _ := newVisionApp(t, "deepseek", "deepseek-v4")
	original := "why is this broken?" + composerLine

	text, images := app.visionAttachments(original)

	if len(images) != 0 {
		t.Errorf("attached %d images to a model that cannot see", len(images))
	}
	if text != original {
		t.Errorf("text was rewritten for a blind model\n got: %q\nwant: %q", text, original)
	}
}

func TestVisionAttachmentsIgnoresAMessageWithNoAttachment(t *testing.T) {
	app, _ := newVisionApp(t, "openai", "gpt-4o")
	const plain = "just a question"

	text, images := app.visionAttachments(plain)

	if text != plain || images != nil {
		t.Errorf("visionAttachments(%q) = (%q, %v), want it untouched", plain, text, images)
	}
}

// A path that escapes the sandbox must not be read. Falling back to the OCR
// line rather than erroring is deliberate: image_ocr resolves paths through the
// same guard and will refuse it in terms the model already understands.
func TestVisionAttachmentsRefusesToEscapeTheSandbox(t *testing.T) {
	app, _ := newVisionApp(t, "openai", "gpt-4o")
	escaping := "look" + "\n\n[attachment: user-attached image — read it with image_ocr] ../../secrets.png"

	text, images := app.visionAttachments(escaping)

	if len(images) != 0 {
		t.Fatal("read an image from outside the sandbox root")
	}
	if text != escaping {
		t.Errorf("text = %q, want it left for the OCR path", text)
	}
}

// A missing file is the same story: no crash, no empty image, just the path
// that was already there.
func TestVisionAttachmentsSurvivesAMissingFile(t *testing.T) {
	app, _ := newVisionApp(t, "openai", "gpt-4o")
	gone := "look" + "\n\n[attachment: user-attached image — read it with image_ocr] .aetox-attachments/nope.png"

	text, images := app.visionAttachments(gone)

	if len(images) != 0 {
		t.Fatalf("attached %d images for a file that does not exist", len(images))
	}
	if text != gone {
		t.Errorf("text = %q, want it unchanged", text)
	}
}

// A tripwire, not a behaviour test.
//
// An attached image is capability-gated in both directions: the composer writes
// "read it with image_ocr", and visionAttachments above rewrites that line when
// ResolveVision says the model can look. A clip is not. The composer writes
// "read its speech with audio_transcribe, its on-screen text with video_ocr"
// and nothing ever revisits it — so a model that could watch the video itself
// would be handed an instruction to go OCR it instead, obey, and nobody would
// see a failure. The answer would just be worse.
//
// That asymmetry is correct *today* and only today, for one reason: there is no
// channel to send a clip through. model.Message carries Images and nothing
// else, so the skill route is the only route for every model Aetox has. The
// hint cannot be wrong because there is no alternative for it to be wrong
// about.
//
// This test fails the day that stops being true. When it does, the fix is not
// here — it is to gate the composer's clip hint the way the image hint is
// gated, and to give this file a sibling of visionAttachments to rewrite it.
func TestClipHintStaysUngatedOnlyWhileNoClipChannelExists(t *testing.T) {
	media := reflect.VisibleFields(reflect.TypeOf(model.Message{}))
	for _, f := range media {
		switch strings.ToLower(f.Name) {
		case "images":
			continue // the gated one — visionAttachments is its rewrite
		case "video", "videos", "audio", "audios", "clips", "media", "attachments", "files":
			t.Fatalf("model.Message gained %q: a clip can now reach a model directly, so the composer's "+
				"\"read it with video_ocr / audio_transcribe\" hint must be capability-gated like the image "+
				"hint is — see cockpit.svelte.ts sendUserMessage and visionAttachments in app.go", f.Name)
		}
	}
}

// A message typed INTO a running turn gets its attachments resolved exactly as
// a new turn's does.
//
// It did not, and the shape of the failure is why this test is here rather than
// in a comment: the picture was saved, the path was correct, the model could
// see, every piece worked — and the model was handed the OCR line anyway,
// because the only code that turns a path into bytes sat on the other branch.
// So it went looking for an OCR library, found none installed, and the owner
// watched a sighted model fail to read five screenshots (15 ก.ย. 2026:
// *"ทำไมโมเดลนี้มันอ่านภาพไม่ได้"*). Nothing was logged, nothing errored.
func TestInterjectionCarriesThePictureIntoTheRunningTurn(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir()) // no user presets to expand against
	app, _ := newVisionApp(t, "openai", "gpt-4o")
	app.cur().agent = cognitive.NewAgent(cognitive.AgentConfig{
		Provider: model.NewNoopProvider("aetox-tools:test"),
		Model:    "aetox-tools:test",
	})

	if err := app.Interject("นี่ครับ ยอดเท่าไหร่" + composerLine); err != nil {
		t.Fatalf("Interject: %v", err)
	}

	got := app.cur().agent.DrainInterjections()
	if len(got) != 1 {
		t.Fatalf("the running turn was handed %d message(s), want 1", len(got))
	}
	if len(got[0].Images) != 1 {
		t.Fatalf("images carried into the turn = %d, want 1 — the picture was dropped", len(got[0].Images))
	}
	// And the line is rewritten for the same reason it is on a new turn: a model
	// holding the picture must not also be told to go and OCR it.
	if strings.Contains(got[0].Text, "image_ocr") {
		t.Errorf("the OCR instruction survived beside the attached picture: %q", got[0].Text)
	}
	if !strings.Contains(got[0].Text, "included below") {
		t.Errorf("the marker was not rewritten: %q", got[0].Text)
	}
}

// The blind model keeps the path it can actually use. Same rule as a new turn,
// asked on this branch too: OCR is not a lesser path, it is the only path for a
// model with no eyes.
func TestInterjectionLeavesTheOCRLineForABlindModel(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	app, _ := newVisionApp(t, "ollama", "qwen3:8b")
	app.cur().agent = cognitive.NewAgent(cognitive.AgentConfig{
		Provider: model.NewNoopProvider("aetox-tools:test"),
		Model:    "aetox-tools:test",
	})

	if err := app.Interject("อ่านให้หน่อย" + composerLine); err != nil {
		t.Fatalf("Interject: %v", err)
	}

	got := app.cur().agent.DrainInterjections()
	if len(got) != 1 || len(got[0].Images) != 0 {
		t.Fatalf("a blind model was sent %d image(s)", len(got[0].Images))
	}
	if !strings.Contains(got[0].Text, "image_ocr") {
		t.Errorf("the OCR line it needs was taken away: %q", got[0].Text)
	}
}
