package skill

// image_make draws a picture from a description and puts it in the workspace.
//
// The sibling of media_read's `image` action, and deliberately not one of its
// actions: that pack is "reading a file a model cannot read by itself", and
// every gate it sits behind was drawn for reads. This one WRITES — it is out of
// วางแผน's keep list, it needs the sandbox's write path, and it is a different
// grant. Folding it into the pack would have quietly given a read permission
// the power to create files.
//
// This file knows nothing about Pollinations, diffusion, or any vendor's wire
// format: picking one, resolving its model and turning a prompt into bytes all
// live in internal/imagegen (ARCHITECTURE.md §33), the same way audio_transcribe
// knows nothing about whisper. What is left here is the part that is genuinely
// this skill's — placing the file, naming it truthfully, and refusing to hand
// back something that is not a picture.
//
// **The extension is not the caller's to get wrong.** A model asked for a
// picture will name it `hero.png` because that is what pictures are called, and
// the engine that answers may only make JPEG. media_fetch warns about that
// mismatch because it did not choose the bytes; here we did, so the name is
// corrected to match them and the receipt says so. A .png that is a JPEG is the
// same silent breakage media_fetch exists to catch, one door over.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/imagegen"
	"github.com/Mikedev115/Aetox/internal/model"
)

type imageMakeSkill struct {
	root         string
	outputSubdir func() string
	files        *FileState
	images       imagegen.Options
	// newEngine is swapped in tests to exercise this file without reaching the
	// network; production always builds from the catalog.
	newEngine func(imagegen.Options) (imagegen.Engine, error)
}

func (*imageMakeSkill) Name() string { return "image_make" }

func (*imageMakeSkill) Description() string {
	return "วาดรูปจากคำบรรยาย แล้วบันทึกเป็นไฟล์ในโปรเจกต์ (ใช้เอนจินสร้างภาพที่ผู้ใช้เลือกไว้ในหน้า ภาพ)"
}

func (*imageMakeSkill) ToolDefinition() model.ToolDefinition {
	// Bare types and no per-property prose: the block entry carries what the
	// tool IS and what to pass it, and everything else is sent once from
	// Guidance below (block_standard_test.go, 80-token ceiling).
	//
	// `seed` is deliberately absent even though the engine takes one. It is the
	// least-used knob here and additionalProperties:false means every advertised
	// property is paid for on every request that carries this tool — the ceiling
	// is a budget, and reproducing an earlier picture is not what the budget is
	// best spent on. ExecuteTool still reads it, so it costs nothing to put back.
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{"type": "string"},
			"path":   map[string]any{"type": "string"},
			"width":  map[string]any{"type": "integer"},
			"height": map[string]any{"type": "integer"},
		},
		"required":             []string{"prompt", "path"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "image_make",
			Description: "Draw a picture from a description.",
			Parameters:  payload,
		},
	}
}

// Guidance carries the judgment the block entry may not: what this is not, how
// to write a prompt a picture model can read, and the duty that comes with a
// picture nobody photographed.
func (*imageMakeSkill) Guidance(map[string]any) string {
	return "image_make draws something that does not exist yet — an illustration, a texture, a placeholder. " +
		"It does not edit an existing picture and does not search for one: a real photograph comes from " +
		"web_search then media_fetch, which is also the route a published piece of work needs. Write the " +
		"prompt in English for the widest model support and describe subject, composition and style — a " +
		"picture model reads it literally and cannot ask a follow-up. width and height are pixels; omit them for the engine's own defaults. The file extension is " +
		"corrected to whatever the engine really produced, so the path you get back may not be the path " +
		"you asked for. Tell the user the picture was generated — it carries no licence and no provenance."
}

func (s *imageMakeSkill) Execute(ctx context.Context, input Input) (Output, error) {
	args := stringSlice(input["args"])
	if len(args) < 2 {
		err := errors.New("usage: image_make <path> <prompt...>")
		return newToolOutput("image_make", "image_make", "", time.Now(), false, err), err
	}
	// Path first here, prompt taking the rest: a prompt is many words and a
	// path is one, so any other order cannot be split unambiguously.
	return s.draw(ctx, strings.Join(args[1:], " "), args[0], imagegen.Request{})
}

func (s *imageMakeSkill) ExecuteTool(ctx context.Context, args map[string]any) (Output, error) {
	prompt, _ := args["prompt"].(string)
	path, _ := args["path"].(string)
	req := imagegen.Request{
		Width:  intArg(args["width"]),
		Height: intArg(args["height"]),
		// Read but not advertised — see ToolDefinition. A caller that knows the
		// name (the CLI path, a future control) still gets it.
		Seed: intArg(args["seed"]),
	}
	return s.draw(ctx, strings.TrimSpace(prompt), strings.TrimSpace(path), req)
}

func (s *imageMakeSkill) draw(ctx context.Context, prompt, requestPath string, req imagegen.Request) (Output, error) {
	start := time.Now()
	command := "image_make " + requestPath
	fail := func(err error) (Output, error) {
		return newToolOutput("image_make", command, "", start, false, err), err
	}

	if prompt == "" || requestPath == "" {
		return fail(errors.New("usage: image_make <path> <prompt...>"))
	}

	build := s.newEngine
	if build == nil {
		build = imagegen.New
	}
	engine, err := build(s.images)
	if err != nil {
		return fail(err)
	}

	// The name is settled before anything is written, so the file that appears
	// and the file named in the receipt cannot differ. See the file comment for
	// why the extension is corrected rather than warned about.
	// A first guess at the name, from the engine's own declaration, so the file
	// is created under something plausible. The bytes correct it after the
	// write — see below.
	original := requestPath
	requestPath = withExt(requestPath, engine.Ext())
	renamed := false

	// The same placement rule write, sheet_write and media_fetch follow
	// (write.go): a relative path in an unfocused session lands in the
	// session's output folder, and the receipt names where it really went.
	placed := placedWrite(s.outputSubdir, requestPath)
	targetPath, err := resolveSandboxPath(s.root, placed)
	if err != nil {
		return fail(err)
	}
	if err := ensureWriteDir(targetPath); err != nil {
		return fail(err)
	}
	if err := s.files.guardStale(targetPath, placed); err != nil {
		return fail(err)
	}

	if err := engine.Generate(ctx, prompt, req, targetPath); err != nil {
		return fail(err)
	}

	// Judged by its own first bytes, never by what the engine promised — the
	// rule media_fetch is built on, and the reason it is worth repeating here
	// is that an engine writing straight to a path is exactly the arrangement
	// that could leave a plausible-looking file nobody checked.
	body, err := os.ReadFile(targetPath)
	if err != nil {
		return fail(err)
	}
	kind := sniffMediaKind(body)
	if kind == "" || !isPictureKind(kind) {
		// Take the impostor back off the disk. Leaving it would put a file
		// card in the chat for something that opens nowhere.
		_ = os.Remove(targetPath)
		return fail(fmt.Errorf("%s เขียนไฟล์ออกมาแล้วแต่ไม่ใช่รูป — ลบทิ้งแล้ว ไม่ได้บันทึกอะไรไว้", engine.ID()))
	}

	// And now the bytes get the last word on the NAME too. Ext() was only ever
	// a hint, and with four vendors behind this it is four separate guesses —
	// the dall-e path hands back whatever sits at a URL, and a row declaring
	// PNG can perfectly well be given a JPEG. Correcting it here means no
	// engine can produce a file whose extension lies, however wrong its own
	// declaration was.
	if want := extFor(kind); want != "" && !strings.EqualFold(pathExt(targetPath), want) {
		corrected := strings.TrimSuffix(targetPath, pathExt(targetPath)) + want
		if renameErr := os.Rename(targetPath, corrected); renameErr != nil {
			return fail(renameErr)
		}
		targetPath = corrected
		placed = withExt(placed, want)
	}
	// Noted after the rename, so the record names the file that exists.
	s.files.Note(targetPath)
	renamed = !strings.EqualFold(pathExt(original), pathExt(placed))

	report := fmt.Sprintf("วาดแล้ว %s (%s, %s", placed, kind, humanBytes(len(body)))
	if w, h, ok := imageDims(kind, body); ok {
		report += fmt.Sprintf(", %dx%d px", w, h)
	}
	report += ")"
	if renamed {
		report += fmt.Sprintf("\nนามสกุลถูกเปลี่ยนจาก %s เป็น %s ให้ตรงกับไบต์ที่ได้มาจริง", pathExt(original), pathExt(placed))
	}
	if placed != requestPath {
		report += onDiskNote(s.root, targetPath)
	}
	report += "\nรูปนี้สร้างด้วย AI — บอกผู้ใช้ด้วยถ้ามันจะถูกเอาไปใช้ที่อื่น"
	return newToolOutput("image_make", command, report, start, false, nil), nil
}

// isPictureKind keeps this tool to still images. sniffMediaKind also answers
// for sound, and a picture tool that accepted an MP3 because the sniffer
// recognised it would be a check that never says no.
// extFor is the extension that matches what sniffMediaKind saw. Empty for
// anything that is not a still picture — isPictureKind has already refused
// those, so an empty answer means "leave the name alone".
func extFor(kind string) string {
	switch kind {
	case "png":
		return ".png"
	case "jpg":
		return ".jpg"
	case "gif":
		return ".gif"
	case "webp":
		return ".webp"
	}
	return ""
}

func isPictureKind(kind string) bool {
	switch kind {
	case "png", "jpg", "gif", "webp":
		return true
	}
	return false
}

// withExt puts want on path, replacing whatever extension was there. A path
// that already ends in want is returned untouched, including its own casing.
func withExt(path, want string) string {
	if want == "" {
		return path
	}
	have := pathExt(path)
	if strings.EqualFold(have, want) {
		return path
	}
	return strings.TrimSuffix(path, have) + want
}
