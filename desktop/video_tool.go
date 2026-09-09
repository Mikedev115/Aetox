package main

// The agent's hands on the scene renderer.
//
// **Why this is a tool of ours and not somebody else's MCP server.** Every
// external program Aetox uses is reached the same way: a Go skill runs it and
// names its parameters after what they mean. `image_ocr` knows tesseract's
// flags, `video_ocr` knows ffmpeg's, `audio_transcribe` knows whisper's, and
// none of the three asks a model to remember a command line. Making a video was
// about to be the first exception — routed through kinocut, a Python MCP server
// whose own job is cutting footage that already exists, which then forwards to
// the same Hyperframes CLI this file calls directly. Owner, 30 ส.ค.: *"มันคนละ
// ส่วนกัน ทำไมถึงเอามาปนกัน"*. So: making a video talks to Hyperframes, cutting
// one talks to kinocut, and neither carries the other's runtime.
//
// **Three actions, because the job has three moves.** Take a scene out of the
// library and make it a project, look at it before spending the render, then
// render. That is the loop Hyperframes' own documentation describes, and the
// middle one is the one an agent skips and should not: a render is minutes of
// somebody's machine, and `check` opens the page once and reports the text that
// ran past the edge of the frame.
//
// **What this file does NOT do** is decide anything about the video. Length,
// wording, which scene, how many — those are the agent's judgement, and they
// arrive as parameters. This file resolves paths, runs one program, and reports
// what came back.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/statereport"
	"github.com/Mikedev115/Aetox/internal/subagent"
)

// videoToolName is the one name this pack answers to, spelled once for the same
// reason browserToolName is.
const videoToolName = "video"

const (
	// The agent whose shelf the scene library sits on, and the skill on it.
	// Named rather than searched: a library that could be found under any agent
	// is a library any agent could quietly render from.
	videoLibraryAgent = "video"
	videoLibrarySkill = "video-templates"

	// Where the motion scenes live inside that skill, and the shelf a bare name
	// is looked for on first.
	videoLibraryDir = "motion"

	// The name that means "do not copy anybody's scene". It is not on any
	// shelf: the file comes out of the renderer's own bundle, which is the one
	// blank composition guaranteed to match the engine that will render it.
	videoBlankTemplate = "blank"

	// What a still scene's frame says when nobody said. Only ever a
	// placeholder: a cover has no length of its own and the renderer needs a
	// number, so one is written and then reported, never written and left to be
	// discovered in the output.
	videoStillFallbackSeconds = 4.0

	// How many scene names one shelf contributes to a refusal before it says
	// how many more there are. The refusal exists to un-stick a caller who
	// guessed a name, not to be the inventory.
	videoNamesPerShelf = 20

	// A render is the longest thing this app runs on purpose. Bounded anyway,
	// because a scene that has stopped answering should end as an error rather
	// than as a chat that never comes back.
	videoRenderTimeout = 30 * time.Minute
	// Checking opens one browser session and seeks through the scene once.
	videoCheckTimeout = 5 * time.Minute
)

type videoToolSkill struct {
	app *App
	// actions this caller may use, nil for all of them. Set only by Narrow.
	actions []string
}

func (s *videoToolSkill) allowedActions() []string {
	if s == nil || len(s.actions) == 0 {
		out := make([]string, 0, len(skill.PackedCalls(videoToolName)))
		for _, call := range skill.PackedCalls(videoToolName) {
			out = append(out, call.Action)
		}
		return out
	}
	return s.actions
}

func (s *videoToolSkill) Actions() []string { return skill.PackedActions(videoToolName) }

// Narrow hands back a scene tool offering only the named actions — a copy, for
// the same shared-registry reason the browser's and the shell's are.
func (s *videoToolSkill) Narrow(named []string) skill.Skill {
	narrowed := *s
	want := map[string]bool{}
	for _, n := range named {
		want[strings.ToLower(strings.TrimSpace(n))] = true
	}
	var actions []string
	for _, call := range skill.PackedCalls(videoToolName) {
		if want[call.Permission] {
			actions = append(actions, call.Action)
		}
	}
	if len(actions) == 0 {
		return s // silence is the whole tool, not an empty one
	}
	narrowed.actions = actions
	return &narrowed
}

func (*videoToolSkill) Name() string { return videoToolName }

func (*videoToolSkill) Description() string {
	return "สร้างคลิปจากฉาก HTML — หยิบฉากจากคลังมาเป็นโปรเจกต์ ตรวจก่อนเรนเดอร์ แล้วเรนเดอร์เป็นไฟล์วิดีโอ"
}

func (s *videoToolSkill) ToolDefinition() model.ToolDefinition {
	allowed := s.allowedActions()
	// Signatures only. Every "when should I reach for this" sentence belongs in
	// Guidance, sent once on the first call — the same split browser_tool.go
	// made when its entry went from 766 tokens to a fifth of that.
	lines := map[string]string{
		"new":    "`new` (template, path, seconds?, footage?) — copy a scene out of the library into its own project folder, ready to edit. `template` is a scene name, `<shelf>/<name>` when two shelves could mean it, or `blank` for the renderer's own empty composition when no scene is the shape you need.",
		"check":  "`check` (path) — open the project once and report runtime errors, text that overflows the frame, and contrast failures.",
		"render": "`render` (path, output?, resolution?, format?, fps?, quality?, composition?, variables?, strict?, proof?) — render the project; a fresh check report rides along, and proof:true also reads the on-screen text back via OCR in the same reply. Length is not a render option: it is `data-duration` in the markup.",
	}
	var b strings.Builder
	b.WriteString("Build a video from an HTML scene. Actions:\n")
	for _, action := range allowed {
		b.WriteString(lines[action] + "\n")
	}
	b.WriteString("\nEvery path is relative to the project root. " +
		"Read the `video-templates` skill for what is in the library and what each scene needs.")

	// One schema for three actions, because `video` is one packed tool. That
	// means every property is visible to every action and the description is
	// where an action-shaped fact has to live — see `seconds`, which `render`
	// does not read and cannot stop being offered while `new` needs it. Saying
	// so in the schema is the honest version of "removed from the signature",
	// which is what the render line above claimed on its own until 8 ก.ย. 2569
	// while this block went on advertising the parameter to the model.
	return toolDef(videoToolName, b.String(), map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":      map[string]any{"type": "string", "enum": allowed},
			"template":    map[string]any{"type": "string"},
			"path":        map[string]any{"type": "string"},
			"output":      map[string]any{"type": "string"},
			"footage":     map[string]any{"type": "string"},
			"composition": map[string]any{"type": "string"},
			"seconds": map[string]any{
				"type":        "number",
				"description": "`new` only, and only for the scenes carrying __VIDEO_DURATION__; every other scene's length is `data-duration` in its markup. `render` has no length option and neither does the renderer.",
			},
			"fps": map[string]any{
				"type":        "string",
				"description": "`render`. Whole number (24, 25, 30, 50, 60, 120, 240) or ffmpeg rational as text — \"24000/1001\" for 23.976, \"30000/1001\" for 29.97, \"60000/1001\" for 59.94. Range 1-240; defaults to the composition's data-fps, else 30.",
			},
			"resolution": map[string]any{
				"type":        "string",
				"enum":        videoResolutions,
				"description": "`render`. Scales the capture without touching the scene: landscape 1920x1080, portrait 1080x1920, square 1080x1080, and the -4k of each (3840x2160, 2160x3840, 2160x2160). Must keep the composition's aspect ratio and scale by a whole multiple.",
			},
			"format": map[string]any{
				"type":        "string",
				"enum":        videoFormats,
				"description": "`render`. mp4 by default. mov and webm carry transparency for an overlay; png-sequence writes RGBA frames to a folder for After Effects, Nuke or Fusion; gif is small enough to paste into a pull request.",
			},
			"variables": map[string]any{
				"type":        "string",
				"description": "`render`. JSON object merged over the composition's data-composition-variables defaults, read inside the scene through window.__hyperframes.getVariables(). One composition, different words, without editing the markup between renders.",
			},
			"strict":  map[string]any{"type": "boolean", "description": "`render`. Fail the render on lint errors instead of producing the file anyway. Off by default, because half of what the layout pass reports on this library is the design of the scene."},
			"quality": map[string]any{"type": "string", "enum": []string{"draft", "standard", "high"}},
			"proof":   map[string]any{"type": "boolean"},
		},
		"required": []string{"action"},
	})
}

func (s *videoToolSkill) ExecuteTool(ctx context.Context, args map[string]any) (skill.Output, error) {
	return s.run(ctx, args)
}

func (s *videoToolSkill) Execute(ctx context.Context, input skill.Input) (skill.Output, error) {
	return s.run(ctx, map[string]any(input))
}

func (s *videoToolSkill) run(ctx context.Context, args map[string]any) (skill.Output, error) {
	action := strings.ToLower(strings.TrimSpace(argString(args, "action")))
	if action == "" {
		err := fmt.Errorf("video ต้องบอกว่าจะทำอะไร: %s", strings.Join(s.allowedActions(), ", "))
		return videoFailed("video", err), err
	}
	allowed := false
	for _, a := range s.allowedActions() {
		if a == action {
			allowed = true
			break
		}
	}
	if !allowed {
		err := fmt.Errorf("video %s ใช้ไม่ได้ที่นี่", action)
		return videoFailed("video "+action, err), err
	}
	switch action {
	case "new":
		return s.newProject(args)
	case "check":
		return s.check(ctx, args)
	case "render":
		return s.render(ctx, args)
	}
	err := fmt.Errorf("video ไม่รู้จักคำสั่ง %q", action)
	return videoFailed("video "+action, err), err
}

// ---------------------------------------------------------------------------
// new
// ---------------------------------------------------------------------------

// newProject copies one scene out of the library and leaves a folder the agent
// can edit and the renderer can read.
//
// A copy rather than the agent writing the files itself, and the reason is
// bytes: nine of the library's scenes carry mp3 and png beside their HTML, and
// a model asked to reproduce those is a model asked to invent them.
func (s *videoToolSkill) newProject(args map[string]any) (skill.Output, error) {
	template := strings.TrimSpace(argString(args, "template"))
	dest := strings.TrimSpace(argString(args, "path"))
	command := "video new " + template
	if template == "" || dest == "" {
		err := fmt.Errorf("video new ต้องบอกทั้ง template และ path")
		return videoFailed(command, err), err
	}
	root, err := s.sandbox()
	if err != nil {
		return videoFailed(command, err), err
	}
	// The same placement rule every file-producing tool follows (write.go): an
	// unfocused session's project lands in output/<session>, not at the root.
	// This tool skipped the rule, so every session's video work piled up in one
	// folder and a test in one chat globbed up the renders of another — the
	// receipt below echoes the placed path, and check/render resolve it back.
	dest = skill.PlacedWrite(s.app.outputSubdir, dest)
	full, err := safeSandboxPath(root, dest)
	if err != nil {
		return videoFailed(command, err), err
	}

	written, err := videoCopyTemplate(template, full)
	if err != nil {
		return videoFailed(command, err), err
	}

	// The two slots the library ships unfilled. Upstream fills them when it
	// scaffolds a project from a video file; nothing fills them on this path,
	// so a scene left with the placeholders in it renders with a duration of
	// the literal string "__VIDEO_DURATION__".
	asked := argFloat(args, "seconds")
	seconds := asked
	if seconds <= 0 {
		seconds = videoDefaultSeconds(full)
	}
	// Whether this scene is one of the four that take a length, asked before
	// the patcher fills the slot and the evidence disappears. The other 46 have
	// their length written into the markup by the author, and a `seconds` that
	// silently does nothing to them is the defect this answers.
	takesLength := videoHasDurationSlot(full)
	// Before the placeholders, because the frame it writes carries the duration
	// as a number and the patcher would otherwise have nothing to fill.
	framed, unframed, err := videoEnsureCompositionRoot(full, filepath.Base(dest), seconds)
	if err != nil {
		return videoFailed(command, err), err
	}
	filled, err := videoPatchPlaceholders(full, seconds, strings.TrimSpace(argString(args, "footage")))
	if err != nil {
		return videoFailed(command, err), err
	}

	localised, err := videoUseLocalGSAP(full)
	if err != nil {
		return videoFailed(command, err), err
	}

	report := fmt.Sprintf("วางฉาก %s ไว้ที่ %s แล้ว (%d ไฟล์", template, dest, written)
	if filled > 0 {
		report += fmt.Sprintf(", แทนค่าความยาว %.4g วินาที ใน %d ไฟล์", seconds, filled)
	}
	if localised > 0 {
		report += fmt.Sprintf(", ชี้ GSAP ไปที่ไฟล์ในเครื่องใน %d ไฟล์", localised)
	}
	if framed {
		report += fmt.Sprintf(", เติมกรอบคอมโพสิชันให้ตามขนาดที่ฉากวาดไว้เอง ยาว %.4g วินาที", videoRootDuration(full))
	}
	report += ")\n" + videoProjectInventory(full)
	// The length, said out loud when it was not the caller's to choose.
	//
	// `seconds` reaches four of the fifty motion scenes — the ones whose author
	// wrote `__VIDEO_DURATION__` because their keyframes were built to stretch.
	// On the other forty-six it did nothing and said nothing, so an agent asked
	// for a 20-second clip got a 30-second one, reported 20, and nobody found
	// out until somebody watched it. There is no flag on the renderer either:
	// duration lives in the markup and only there.
	if asked > 0 && !takesLength && !framed {
		if fixed := videoRootDuration(full); fixed > 0 && math.Abs(fixed-asked) > 0.05 {
			report += fmt.Sprintf("\nความยาวที่ขอมา %.4g วินาที ไม่ได้ถูกใช้ — ฉากนี้ยาว %.4g วินาทีเขียนไว้ในไฟล์"+
				"\nจะเปลี่ยนต้องแก้ `data-duration` บนธาตุรากพร้อมกับคีย์เฟรมที่วิ่งอยู่ในนั้น แก้อย่างเดียวได้ภาพค้างหรือท่าที่ถูกตัดกลางคัน"+
				"\nหรือเลือกฉากที่ยาวใกล้เคียงกว่านี้จากตารางใน SKILL.md ซึ่งบอกความยาวไว้ทุกแถว", asked, fixed)
		}
	}
	if framed {
		// Said out loud rather than left for the render to reveal. The still
		// shelves are for the frames around a video — a cover, an infographic,
		// an interface on screen where a real screenshot would leak real data —
		// and one rendered on its own is a video of a photograph.
		report += "\nฉากนี้ไม่มีการเคลื่อนไหว มันเป็นวัสดุหรือภาพนิ่ง ไม่ใช่คลิปที่ยืนได้ด้วยตัวเอง" +
			" — ถ้าจะให้เป็นงานส่ง ให้เอาไปประกอบในฉากที่เคลื่อนไหว หรือส่งเป็นภาพนิ่ง"
		if asked <= 0 {
			// The number above was ours because nobody gave one, and a scene
			// that does not move has none of its own to read. Said here so it
			// is a starting point rather than a decision made quietly.
			//
			// Read back off the frame that was just written rather than printed
			// from videoStillFallbackSeconds, which is what this said until
			// 8 ก.ย. 2569. The two agree only while no still scene ships a
			// `template.html-video.yaml`; the moment one does, videoDefaultSeconds
			// wins the write and the constant wins the sentence, and one report
			// states two different lengths for one file. Reporting a number the
			// tool did not use is the exact defect the rest of this block exists
			// to answer.
			report += fmt.Sprintf("\nความยาว %.4g วินาทีเป็นค่าตั้งต้นที่เครื่องมือใส่ให้ ไม่ใช่ของฉาก — เปลี่ยนที่ `data-duration` บนราก ได้ตามที่งานต้องการ", videoRootDuration(full))
		}
	}
	if unframed != "" {
		// The loud half of videoEnsureCompositionRoot's third answer. A project
		// with no composition root renders 1080x1920 whatever it drew, crops the
		// rest away, and comes back from `check` with an empty report that reads
		// like a pass — so the one moment this can be caught is here, before the
		// agent spends a render finding out.
		report += "\n**ฉากนี้ไม่มีกรอบคอมโพสิชัน และเติมให้ไม่ได้** — " + unframed +
			"\nถ้าปล่อยไว้ ตัวเรนเดอร์จะทำเป็น 1080x1920 แล้วครอปส่วนที่กว้างกว่านั้นทิ้งเงียบ ๆ และ `video check` จะไม่มีความยาวให้สุ่มเลย์เอาต์ รายงานจะว่างและอ่านเหมือนผ่าน" +
			"\nเติมเองบน `<body>`: `data-composition-id`, `data-width`, `data-height`, `data-start=\"0\"`, `data-duration` และ `data-no-timeline` ถ้าฉากไม่มีการเคลื่อนไหว"
	}
	report += "\nแก้ข้อความข้างบนให้เป็นของงานนี้ทั้งหมด (ที่เห็นคือสำเนาตัวอย่าง) แล้วเรียก video check ก่อนเรนเดอร์" +
		"\nฟอนต์ของฉากไม่มีชุดตัวอักษรไทย — ข้อความไทยจะออกมาเป็นฟอนต์ระบบ ถ้าไม่ใช่ที่ตั้งใจให้เพิ่มฟอนต์ไทยใน <link> เอง"
	return skill.Output{Name: "video_new", Command: command, Content: report, Success: true}, nil
}

// videoProjectInventory reads the freshly copied project so the agent does not
// have to. Measured 31 ส.ค. (session 164630): after `video new` the agent spent
// six calls of glob/list/read learning what the copy held, and eight
// `skill_view` calls before that reading sub-scenes in the library to find
// where the words live. The tool wrote every one of those files; it can say
// what is in them.
func videoProjectInventory(dir string) string {
	var b strings.Builder
	b.WriteString("ไฟล์ในโปรเจกต์ และข้อความที่อยู่ในแต่ละไฟล์:")
	var others []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "vendor/") {
			return nil // the GSAP copy `video new` itself put there
		}
		if !strings.EqualFold(filepath.Ext(path), ".html") {
			others = append(others, rel)
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		snippets := videoVisibleText(string(data), 10)
		if len(snippets) == 0 {
			b.WriteString("\n- " + rel + " — (ไม่มีข้อความ)")
			return nil
		}
		b.WriteString("\n- " + rel + ": " + strings.Join(snippets, " | "))
		return nil
	})
	if len(others) > 0 {
		b.WriteString("\n- ไฟล์ประกอบ: " + strings.Join(others, ", "))
	}
	return b.String()
}

// videoVisibleText is the words a browser would paint, pulled out of markup the
// cheap way: style, script and comments dropped, tags stripped, whitespace
// folded. Cheap is enough — this feeds an editor deciding what to rewrite, not
// a renderer deciding where it lands.
func videoVisibleText(markup string, max int) []string {
	for _, re := range videoInvisibleParts {
		markup = re.ReplaceAllString(markup, " ")
	}
	markup = videoTagRe.ReplaceAllString(markup, "\n")
	replacer := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&nbsp;", " ")
	var out []string
	for _, line := range strings.Split(markup, "\n") {
		line = strings.Join(strings.Fields(replacer.Replace(line)), " ")
		if line == "" {
			continue
		}
		if r := []rune(line); len(r) > 80 {
			line = string(r[:80]) + "…"
		}
		out = append(out, line)
		if len(out) == max {
			out = append(out, "…")
			break
		}
	}
	return out
}

var (
	videoInvisibleParts = []*regexp.Regexp{
		regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`),
		regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?s)<!--.*?-->`),
		regexp.MustCompile(`(?is)<(?:head|title)[^>]*>.*?</(?:head|title)>`),
	}
	videoTagRe = regexp.MustCompile(`<[^>]*>`)
)

// videoCopyTemplate resolves a library name to whichever shape it has.
//
// Thirteen of the twenty-two motion scenes are one self-contained file and nine
// are folders with their own sub-scenes and assets. Both become a project
// folder here, so nothing downstream has to know which kind it started as —
// that difference belongs to the library, not to the renderer.
//
// The shape is decided by asking the shelf, not by which copy fails. The old
// try-the-folder-then-fall-through order turned any folder-copy failure into
// "ไม่มีฉากชื่อ ... ในคลัง" — including the one where the destination already
// existed, which sent an agent retrying a template the refusal itself listed
// (tool_runs, 31 ส.ค. 16:49). A refusal has to name the caller's actual
// mistake, or it manufactures the next one.
// videoShelves are the library's four shelves, in the order a bare name is
// searched. No name appears on two of them, so the order decides nothing today
// — it is written down so that a name that ever does collide resolves the same
// way twice rather than by whichever read came back first.
//
// **Three of these were unreachable until 7 ก.ย. 2569, and the shelf's own
// index taught their names anyway.** `videoLibraryDir` was the only prefix
// stripped and anything with a `/` left in it was refused, so an agent that did
// what SKILL.md's own table says — `video new graphic-scenes/social-cover-
// editorial.html` — got told that a scene it had just read about was not a
// scene. Twenty-five of the seventy-five, which is every cover, infographic,
// data slide and interface mock on the shelf.
//
// The reason they were held back was real and is kept, one line down rather
// than as a locked door: a still scene rendered on its own is a video of a
// photograph. What that argument does not cover is the job SKILL.md actually
// describes them for — "a believable interface on screen where a real
// screenshot would leak real data" — which needs the file inside a project and
// had no way to get there. `product-launch-30s` ships three 1x1 transparent
// PNGs where its product screenshots go; the material to replace them was on
// the shelf the whole time and the tool would not hand it over.
var videoShelves = []string{videoLibraryDir, "graphic-scenes", "slide-scenes", "web-scenes"}

func videoCopyTemplate(template, dest string) (int, error) {
	// The path shapes are accepted, because WE teach them: the SKILL.md tables
	// spell every scene `<shelf>/<name>.html` or `motion/<name>/index.html`,
	// and agents copy that spelling verbatim into `template` — three times in
	// one evening (problem queue, 31 ส.ค.), each answered by a refusal that
	// blamed the caller for reading our own index. One unambiguous meaning,
	// so normalise instead of refusing.
	template = strings.TrimSuffix(strings.ReplaceAll(template, `\`, "/"), "/index.html")
	template = strings.TrimSuffix(template, ".html")
	shelves := videoShelves
	if shelf, name, found := strings.Cut(template, "/"); found {
		if !slices.Contains(videoShelves, shelf) || strings.ContainsAny(name, "/") {
			return 0, fmt.Errorf("template คือชื่อฉากในคลัง หรือ <ชั้น>/<ชื่อฉาก> — ชั้นที่มีคือ %s",
				strings.Join(videoShelves, ", "))
		}
		shelves, template = []string{shelf}, name
	}
	if _, err := os.Stat(dest); err == nil {
		return 0, fmt.Errorf("%s มีอยู่แล้ว", filepath.Base(dest))
	}
	if template == videoBlankTemplate {
		return videoCopyBlank(dest)
	}
	// A shelf that will not list is not a scene that does not exist, so the loop
	// keeps looking — but the reason the first one gave is kept, because if NO
	// shelf lists then the caller's name was never the problem and telling them
	// it was sends them to pick a different one, forever. This dropped the error
	// on the floor until 8 ก.ย. 2569 and answered a broken library with
	// "ไม่มีฉากชื่อ …", which is the failure the comment above this function
	// describes, committed by the function it describes.
	var shelfErr error
	listed := false
	for _, shelf := range shelves {
		names, err := subagent.ListSkillDir(videoLibraryAgent, videoLibrarySkill, shelf)
		if err != nil {
			if shelfErr == nil {
				shelfErr = err
			}
			continue
		}
		listed = true
		dir := shelf + "/" + template
		if slices.Contains(names, template) {
			return subagent.CopySkillDir(videoLibraryAgent, videoLibrarySkill, dir, dest)
		}
		if slices.Contains(names, template+".html") {
			data, err := subagent.ReadSkillFile(videoLibraryAgent, videoLibrarySkill, dir+".html")
			if err != nil {
				return 0, err
			}
			return videoWriteFlatScene(dest, data)
		}
	}
	if !listed {
		return 0, fmt.Errorf("เปิดคลังฉากไม่ได้เลยสักชั้น จึงยังไม่รู้ว่ามี %s หรือไม่: %w", template, shelfErr)
	}
	return 0, fmt.Errorf("ไม่มีฉากชื่อ %s ในคลัง — มีให้เลือกคือ %s", template, videoLibraryNames())
}

// videoCopyBlank writes the renderer's own empty composition, which is the
// answer to "start from nothing" that does not involve anybody guessing at the
// contract.
//
// **Why this exists.** Until 7 ก.ย. 2569 `video new` had no shape but "copy one
// of these", and the brief agreed with it — a scene written from nothing is a
// scene whose timing nobody has tested. Both were true, and together they meant
// every clip this office had made was one of fifty files with the words changed.
// On 9 ก.ย. 2569 the brief was turned around to match this action rather than
// fence it: the shape comes from the piece, the library is where you check
// whether that shape already exists, and a blank start is an ordinary answer.
//
// It comes out of the engine's bundle rather than being a file we wrote,
// because a blank composition is a statement about the renderer's contract —
// which `data-*` attributes a root needs, how a timeline registers — and the
// only copy of that statement guaranteed to match the engine installed here is
// the one shipped beside it. It arrives carrying `__VIDEO_DURATION__`,
// `__VIDEO_SRC__` and the GSAP CDN address, so the three steps `video new`
// already runs over a copied scene fill it in with nothing added.
func videoCopyBlank(dest string) (int, error) {
	src := hyperframesTemplate(videoBlankTemplate)
	if src == "" {
		return 0, fmt.Errorf("ฉากเปล่ามาจากตัวเรนเดอร์ ซึ่งเครื่องนี้ยังไม่ได้ติดตั้ง — หน้างานวิดีโอติดตั้งได้ในกดเดียว หรือเริ่มจากฉากในคลังไปก่อน")
	}
	return copyOSTree(src, dest)
}

// copyOSTree copies a folder on disk into dest, answering with how many files
// it wrote. dest must not exist, for the reason subagent.CopySkillDir refuses a
// folder that does: merging is how a half-finished project quietly inherits
// files from a different scene.
func copyOSTree(src, dest string) (int, error) {
	written := 0
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		written++
		return nil
	})
	return written, err
}

// videoWriteFlatScene turns one self-contained library file into a project
// folder.
func videoWriteFlatScene(dest string, data []byte) (int, error) {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, err
	}
	// index.html, because that is the name the renderer looks for in a project
	// directory. The library's flat scenes are named after themselves, which is
	// right on a shelf and wrong in a project.
	if err := os.WriteFile(filepath.Join(dest, "index.html"), data, 0o644); err != nil {
		return 0, err
	}
	return 1, nil
}

// videoLibraryNames is the shelf, spelled the way `template` has to be spelled.
//
// It is in the refusal rather than in the tool's description on purpose. The
// listing costs tokens on every request if it lives in the schema, and it is
// only ever needed by a caller that has already guessed wrong — which is any
// agent that does not carry the `video-templates` skill and therefore cannot
// take the description's advice to go and read it. Guess once, be told, get it
// right, instead of a loop of "no such scene" with nowhere to look.
func videoLibraryNames() string {
	var parts []string
	for _, shelf := range videoShelves {
		entries, err := subagent.ListSkillDir(videoLibraryAgent, videoLibrarySkill, shelf)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, strings.TrimSuffix(e, ".html"))
		}
		if len(names) == 0 {
			continue
		}
		// Capped per shelf, because this went from fifty names to seventy-five
		// when the other three shelves opened on 7 ก.ย. 2569 and it is printed
		// on every mistyped scene name. The point of the list is to get the
		// caller un-stuck on the shelf they meant, and twenty names does that;
		// the complete inventory is the SKILL.md tables, which the tail below
		// sends them to.
		if len(names) > videoNamesPerShelf {
			names = append(names[:videoNamesPerShelf:videoNamesPerShelf],
				fmt.Sprintf("… อีก %d ฉาก", len(names)-videoNamesPerShelf))
		}
		// Grouped by shelf rather than run together, because the four are not
		// interchangeable: one of them moves and three do not, and a caller
		// reading a flat list of seventy-five names cannot tell which is which.
		parts = append(parts, shelf+": "+strings.Join(names, ", "))
	}
	if len(parts) == 0 {
		return "เปิดสกิล video-templates ดูรายชื่อ"
	}
	parts = append(parts, videoBlankTemplate+": ฉากเปล่าของตัวเรนเดอร์ เมื่อไม่มีฉากไหนเป็นรูปทรงที่ต้องการ")
	parts = append(parts, "รายชื่อครบพร้อมความยาวและกรอบของทุกฉากอยู่ในตารางของสกิล video-templates")
	return strings.Join(parts, "\n")
}

// videoStillCanvas is the page frame a still scene draws itself at, taken from
// the CSS rule that declares the width the scene's own `<meta name="viewport">`
// says the page is.
//
// Read out of the file instead of measured by hand, and instead of editing
// twenty-five vendored files to carry a composition root. CREDITS.md promises
// those files are the source's own bytes, and this is the same split
// `__VIDEO_DURATION__` and the GSAP rewrite already use: the shelf keeps what
// arrived, the copy gets what the engine needs.
//
// **The selector is what decides, and the reason is that size alone does not.**
// This took the first rule declaring both a width and a height over 240px until
// 8 ก.ย. 2569, on the reasoning that 240 tells a page frame from a button. It
// does not, and the shelf already carries the counter-examples: swept on
// 8 ก.ย., `portfolio-home-organic` has a 1440x828 hero, `api-docs-editorial` a
// 260x900 sidebar, `portfolio-home-editorial` two 320x320 tiles and a 344x344,
// `vertical-infographic-organic` a 600x600 and a 400x400. All twenty-five
// measured correctly anyway, and only because the canvas rule happens to come
// first in every file — a property of how somebody wrote them, not one this
// code can hold on to across a library refresh. 1440x828 is the one to picture:
// 72px short of the real 1440x900, close enough to render and to pass `check`
// looking fine, and a hero is exactly as wide as the page it sits on, so no
// amount of comparing widths would have caught it either.
//
// What separates them is what the rule is FOR. The canvas is the page, so the
// rule that sets it names `body` — and swept over the shelf on 8 ก.ย., twenty-
// four of the twenty-five carry exactly one such rule and it is the canvas in
// every one. That is the answer: not the biggest box, not the first box, the
// box that is the document.
//
// **Two fallbacks under it, in order, for a file that does not say it that
// way.** The scene's own `<meta name="viewport" content="width=…">` next, which
// twenty-four of the twenty-five also carry and which at least forces a decoy to
// match the page width. Then first-rule-over-240, the old behaviour with the old
// risk, which is what `explainer-diagram-poster` lands on: it has no body rule
// and its viewport says `device-width`, which says nothing. Kept rather than
// refused because a poster measured wrong is recoverable and a poster refused is
// a shelf with a hole in it — and the outcome is no longer silent either way,
// because videoEnsureCompositionRoot now reports both the frame it wrote and its
// failure to write one.
var videoStillBlock = regexp.MustCompile(`(?s)([^{}]*)\{([^{}]*)\}`)
var videoStillWidth = regexp.MustCompile(`(?:^|[^-\w])width\s*:\s*(\d+)px`)
var videoStillHeight = regexp.MustCompile(`(?:^|[^-\w])height\s*:\s*(\d+)px`)
var videoStillBodySel = regexp.MustCompile(`(?i)(?:^|[\s,>+~])body\b`)
var videoViewportMeta = regexp.MustCompile(`(?i)<meta\b[^>]*name\s*=\s*["']?viewport["']?[^>]*>`)
var videoViewportWidth = regexp.MustCompile(`(?i)\bwidth\s*=\s*(\d+)`)

func videoStillCanvas(markup string) (width, height int) {
	// 0 when the scene declares no pixel width — `device-width`, or no meta at
	// all — which is the signal to fall through rather than to refuse.
	declared := 0
	if meta := videoViewportMeta.FindString(markup); meta != "" {
		if m := videoViewportWidth.FindStringSubmatch(meta); m != nil {
			declared, _ = strconv.Atoi(m[1])
		}
	}
	var body, matchesViewport, first [2]int
	for _, block := range videoStillBlock.FindAllStringSubmatch(markup, -1) {
		selector, rule := block[1], block[2]
		w := videoStillWidth.FindStringSubmatch(rule)
		h := videoStillHeight.FindStringSubmatch(rule)
		if w == nil || h == nil {
			continue
		}
		wv, _ := strconv.Atoi(w[1])
		hv, _ := strconv.Atoi(h[1])
		// The 240 floor is still here and it is no longer what does the work:
		// it only keeps an icon out of the last fallback, where nothing else can.
		if wv < 240 || hv < 240 {
			continue
		}
		if body == [2]int{} && videoStillBodySel.MatchString(selector) {
			body = [2]int{wv, hv}
		}
		if matchesViewport == [2]int{} && declared > 0 && wv == declared {
			matchesViewport = [2]int{wv, hv}
		}
		if first == [2]int{} {
			first = [2]int{wv, hv}
		}
	}
	for _, pick := range [][2]int{body, matchesViewport, first} {
		if pick != [2]int{} {
			return pick[0], pick[1]
		}
	}
	return 0, 0
}

// videoEnsureCompositionRoot gives a copied scene the frame the renderer needs,
// when the scene did not bring one.
//
// **What happens without it**, and it is the defect CREDITS.md already records
// for the thirteen single-file motion scenes: the renderer produces 1080x1920
// for every project that declares nothing, crops anything drawn wider, and says
// nothing about either. `video check` cannot sample a layout it has no duration
// for, so the report comes back empty and reads like a pass.
//
// Only ever adds. A scene that already carries `data-composition-id` anywhere
// is a scene whose author decided the frame, including all fifty on the motion
// shelf, and this leaves it exactly as it was.
//
// **Three answers, not two, and the third is the point.** `framed` is a frame
// written; empty `unframed` with `framed` false is a scene that brought its own
// and needed nothing. A non-empty `unframed` is the case that used to look
// exactly like the second one: a scene with no frame that could not be given
// one. Silence there is the whole defect this function exists to stop — the
// renderer takes an unframed project as 1080x1920, crops whatever is drawn
// wider, and `check` has no duration to sample, so the report comes back empty
// and reads like a pass. Returned as a sentence rather than logged, because the
// only party who can do anything about it is the agent reading the report.
func videoEnsureCompositionRoot(dir, name string, seconds float64) (framed bool, unframed string, err error) {
	index := filepath.Join(dir, "index.html")
	data, readErr := os.ReadFile(index)
	if readErr != nil {
		return false, "", nil // a folder scene mounts its own; nothing to frame
	}
	markup := string(data)
	if strings.Contains(markup, "data-composition-id") {
		return false, "", nil
	}
	// From here down the scene has no frame, so every exit owes a reason.
	width, height := videoStillCanvas(markup)
	if width == 0 || height == 0 {
		return false, "อ่านขนาดที่ฉากวาดตัวเองไม่ออก — ไม่มีทั้ง `<meta name=\"viewport\" content=\"width=…\">`" +
			" และกติกา CSS ที่บอกทั้งกว้างและสูงเป็นพิกเซล", nil
	}
	loc := regexp.MustCompile(`<body\b[^>]*>`).FindStringIndex(markup)
	if loc == nil {
		return false, "ไฟล์นี้ไม่มีแท็ก <body> ให้เติมกรอบ", nil
	}
	open := markup[loc[0]:loc[1]]
	if strings.HasSuffix(open, "/>") {
		return false, "แท็ก <body> ปิดตัวเอง เติมแอตทริบิวต์เข้าไปไม่ได้", nil
	}
	if seconds <= 0 {
		// A still scene has no length of its own to read — nothing moves, so
		// nobody wrote one — and the renderer will not start without a number.
		// So this is a placeholder, not a judgement about how long anybody
		// should look at a cover: `video new` says the number out loud in its
		// report, which is what keeps it a starting point the agent can argue
		// with rather than a decision made quietly on its behalf.
		seconds = videoStillFallbackSeconds
	}
	// data-no-timeline because these do not move: there is no paused timeline
	// for the renderer to seek, and saying so is what stops it waiting for one.
	attrs := fmt.Sprintf(` data-composition-id=%q data-no-timeline data-width="%d" data-height="%d" data-start="0" data-duration="%.4g"`,
		name, width, height, seconds)
	patched := markup[:loc[1]-1] + attrs + markup[loc[1]-1:]
	if writeErr := os.WriteFile(index, []byte(patched), 0o644); writeErr != nil {
		return false, "", writeErr
	}
	return true, "", nil
}

var (
	videoDurationSlot = "__VIDEO_DURATION__"
	videoFootageSlot  = "__VIDEO_SRC__"
	// The four shapes upstream strips when a scene has a footage slot and
	// nobody supplied footage. Copied from hyperframes' own patchVideoSrc
	// (Apache-2.0, HeyGen) rather than reasoned out, because a scene left
	// holding <video src="__VIDEO_SRC__"> renders a broken element rather than
	// nothing, and they already found that out.
	videoFootageStrip = []*regexp.Regexp{
		regexp.MustCompile(`(?s)<video[^>]*src="__VIDEO_SRC__"[^>]*>.*?</video>`),
		regexp.MustCompile(`<video[^>]*src="__VIDEO_SRC__"[^>]*>`),
		regexp.MustCompile(`(?s)<audio[^>]*src="__VIDEO_SRC__"[^>]*>.*?</audio>`),
		regexp.MustCompile(`<audio[^>]*src="__VIDEO_SRC__"[^>]*>`),
	}
	videoDefaultSecRe = regexp.MustCompile(`(?m)^\s*default_sec:\s*([0-9.]+)`)
)

// videoDefaultSeconds reads the length the scene's own manifest suggests.
//
// A regex rather than a YAML parser, and deliberately: one number is wanted out
// of a file this app never writes, and adding a YAML dependency to read it
// would be the larger change. A miss falls through to the caller's default,
// which is a number the agent was going to have to justify anyway.
func videoDefaultSeconds(dir string) float64 {
	data, err := os.ReadFile(filepath.Join(dir, "template.html-video.yaml"))
	if err != nil {
		return 0
	}
	m := videoDefaultSecRe.FindSubmatch(data)
	if m == nil {
		return 0
	}
	seconds, err := strconv.ParseFloat(string(m[1]), 64)
	if err != nil {
		return 0
	}
	return seconds
}

// videoHasDurationSlot reports whether any file in the project still carries
// `__VIDEO_DURATION__` — that is, whether this scene's author meant its length
// to be given rather than fixed.
//
// Asked of the copy rather than of a list of four scene names, because the
// answer is a property of the markup and a list would be a second place to keep
// it true.
func videoHasDurationSlot(dir string) bool {
	found := false
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found || !strings.EqualFold(filepath.Ext(path), ".html") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && strings.Contains(string(data), videoDurationSlot) {
			found = true
		}
		return nil
	})
	return found
}

// videoRootDuration is how long this project actually is, read off the frame
// the renderer will read it off.
//
// The first element carrying both `data-composition-id` and `data-duration`
// that is not itself a mounted sub-composition — a flat scene declares the pair
// on `<body>`, a folder scene on a `<div>` that wraps the beats, and the beats
// carry the same pair one level in.
//
// **`data-composition-src` is what tells the whole from its parts**, and
// document order is not. This filtered on nothing and took the first match
// until 8 ก.ย. 2569, on the reasoning that a root is written before the layers
// it wraps. Two of the thirteen folder scenes are the other way round:
// `grain-texture-hero` mounts `intro` at a literal `data-duration="2.5"` and
// `structured-grid` mounts one at `1.86`, both ahead of a root whose own
// duration is still `__VIDEO_DURATION__` at that moment — so the old rule was
// right only because videoPatchPlaceholders had already run and turned the root
// into a number the regex could see. An ordering dependency between two
// functions that nothing wrote down. Swept over all thirteen on 8 ก.ย.: the
// filter picks the right root in every one of them, before or after the patch,
// and recovers `bold-portrait-title` and `playful-bounce`, which answered 0.
//
// 0 when the project declares nothing, which is a different sentence from "it
// is zero seconds long" and is why the callers check.
var videoRootTag = regexp.MustCompile(`<[a-zA-Z][^>]*>`)
var videoRootDurationAttr = regexp.MustCompile(`data-duration="([0-9.]+)"`)

func videoRootDuration(dir string) float64 {
	data, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		return 0
	}
	for _, tag := range videoRootTag.FindAllString(string(data), -1) {
		if !strings.Contains(tag, "data-composition-id") {
			continue
		}
		if strings.Contains(tag, "data-composition-src") {
			continue // a layer this project mounts, not the project
		}
		m := videoRootDurationAttr.FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		seconds, convErr := strconv.ParseFloat(m[1], 64)
		if convErr != nil {
			continue
		}
		return seconds
	}
	return 0
}

// videoUseLocalGSAP points a copied project's script tags at the copy of GSAP
// on this machine, and answers with how many files it rewrote.
//
// **Why the rewrite happens here and not in the library.** Nine scenes load
// their animation library from a CDN, which means nine of the twenty-two need
// the network up at render time or they produce a still picture with nothing in
// the output saying why. The fix cannot be to edit those nine in place: the file
// that arrives on disk has to keep working when somebody opens it in a browser
// straight out of the shelf, and CREDITS.md promises the folders are
// byte-for-byte upstream's. So the library keeps the CDN address and the copy
// gets the local one — the same split `__VIDEO_DURATION__` already uses.
//
// **A missing GSAP is not an error.** It leaves the CDN address in place, which
// is exactly what upstream shipped and works whenever there is a network. The
// tool says what it did; it does not refuse to make a project because an
// optional download has not happened.
func videoUseLocalGSAP(dir string) (int, error) {
	local := gsapFile()
	if local == "" {
		return 0, nil
	}
	// One copy per project, beside the scene rather than inside it, so a
	// composition folder's own files are still only the ones upstream wrote.
	vendor := filepath.Join(dir, "vendor")
	copied := false
	rewritten := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		if !strings.Contains(text, gsapCDNURL) {
			return nil
		}
		if !copied {
			if err := os.MkdirAll(vendor, 0o755); err != nil {
				return err
			}
			blob, err := os.ReadFile(local)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(vendor, "gsap.min.js"), blob, 0o644); err != nil {
				return err
			}
			copied = true
		}
		// Root-relative from every file, including the ones inside
		// compositions/, and the word "including" is the whole fix.
		//
		// This was `filepath.Rel` from the loading file until 7 ก.ย. 2569, on
		// the reasoning that a sub-composition sits one level down and should
		// therefore climb one level up. It sits one level down **on disk**; the
		// renderer serves it with the project root as its base URL, so `../`
		// climbs above the root and `check` refuses it —
		// `invalid_parent_traversal_in_asset_path`, whose own fix line says
		// "compositions are served with the project root as their base URL, so
		// paths must be root-relative, not relative to the compositions/
		// directory."
		//
		// **What it cost.** Every folder scene that loads GSAP came out of
		// `video new` with one error per sub-composition and no way for the
		// agent to fix it — 8 on `product-launch-30s`, and 9 of the 13 folder
		// scenes affected, which is every rich scene on the shelf. The render
		// still worked, because the renderer rewrites the path against each
		// sub-composition's source. So the agent was handed a report that said
		// the good templates were broken and the flat ones were clean, and it
		// did what anybody would: it stopped reaching for the folders. Measured
		// 7 ก.ย. on a real copy — 8 errors before, `Check passed` after.
		href := "vendor/gsap.min.js"
		if err := os.WriteFile(path, []byte(strings.ReplaceAll(text, gsapCDNURL, href)), 0o644); err != nil {
			return err
		}
		rewritten++
		return nil
	})
	return rewritten, err
}

// videoPatchPlaceholders fills the two slots across every HTML file in the
// project and answers with how many files it touched.
func videoPatchPlaceholders(dir string, seconds float64, footage string) (int, error) {
	touched := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		if !strings.Contains(text, videoDurationSlot) && !strings.Contains(text, videoFootageSlot) {
			return nil
		}
		if footage != "" {
			text = strings.ReplaceAll(text, videoFootageSlot, footage)
		} else {
			for _, re := range videoFootageStrip {
				text = re.ReplaceAllString(text, "")
			}
		}
		if seconds > 0 {
			text = strings.ReplaceAll(text, videoDurationSlot, strconv.FormatFloat(seconds, 'g', -1, 64))
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return err
		}
		touched++
		return nil
	})
	return touched, err
}

// ---------------------------------------------------------------------------
// check and render
// ---------------------------------------------------------------------------

func (s *videoToolSkill) check(ctx context.Context, args map[string]any) (skill.Output, error) {
	dir := strings.TrimSpace(argString(args, "path"))
	command := "video check " + dir
	full, err := s.projectDir(dir)
	if err != nil {
		return videoFailed(command, err), err
	}
	// **Findings are the answer, not a failure.** `check` exits non-zero the
	// moment it has anything to say, and everything it has to say is on stdout —
	// so treating the exit code as an error threw the report away and handed the
	// agent the last line of stderr instead, which is a note about which fonts
	// were fetched. Measured 31 ส.ค.: `statement-title` came back as
	// "check ไม่สำเร็จ: Fetched 5 font face(s) for Space Grotesk", with five
	// layout findings on the floor.
	//
	// It is also not a gate. Half of what the layout pass reports on this
	// library is deliberate — tilted headline lines that overlap each other are
	// the design of the scene — so the judgement belongs to the agent reading
	// it, the way it belongs to a person looking at the frame. The report ends
	// with its own verdict, and videoTail keeps the end.
	report, ran, err := runHyperframesReport(ctx, videoCheckTimeout, "check", full)
	if !ran {
		return videoFailed(command, err), err
	}
	return skill.Output{Name: "video_check", Command: command, Content: report, Success: true}, nil
}

func (s *videoToolSkill) render(ctx context.Context, args map[string]any) (skill.Output, error) {
	dir := strings.TrimSpace(argString(args, "path"))
	command := "video render " + dir
	full, err := s.projectDir(dir)
	if err != nil {
		return videoFailed(command, err), err
	}

	argv := []string{"render", full}

	// The container, decided before the name, because it decides the name and
	// it decides what "there is a render" means at the end.
	format := strings.ToLower(strings.TrimSpace(argString(args, "format")))
	if format != "" && !slices.Contains(videoFormats, format) {
		err := fmt.Errorf("format %q ไม่มี — มี %s", format, strings.Join(videoFormats, ", "))
		return videoFailed(command, err), err
	}
	if format == "" {
		format = "mp4"
	}

	out := strings.TrimSpace(argString(args, "output"))
	if out == "" {
		// Beside the project, named after it. Upstream's own default is
		// renders/<name>.mp4 inside the project, which buries the one file the
		// user actually wants under the working folder.
		//
		// The suffix follows the format, because a webm written to a name
		// ending .mp4 is a file every player on this machine will refuse.
		out = filepath.Base(full) + videoFormatSuffix[format]
	}
	root, err := s.sandbox()
	if err != nil {
		return videoFailed(command, err), err
	}
	// Beside the project as projectDir RESOLVED it, not as the caller spelled
	// it — with the session placement above, those are different folders, and
	// the render must land next to the project it came from.
	outFull, err := safeSandboxPath(root, filepath.Join(filepath.Dir(skill.PlacedPath(root, s.app.outputSubdir, dir)), out))
	if err != nil {
		return videoFailed(command, err), err
	}
	argv = append(argv, "--output", outFull)

	// Passed as text, not as an integer, and that is the unlock rather than a
	// detail. `hyperframes render --help` takes "an integer (24, 25, 30, 50, 60,
	// 120, 240) or ffmpeg-style rational (30000/1001 for NTSC 29.97, 24000/1001
	// for 23.976, 60000/1001 for 59.94), range 1-240". This read argInt and the
	// tool schema said `"type": "integer"` until 8 ก.ย. 2569, so 23.976 — the
	// rate every piece of footage shot on film arrives at — was unreachable
	// through a ceiling this office had built and the engine had not. The engine
	// validates the range; repeating that here would be a second place to keep
	// the same number true.
	if fps := strings.TrimSpace(argString(args, "fps")); fps != "" && fps != "0" {
		argv = append(argv, "--fps", fps)
	}
	if quality := strings.TrimSpace(argString(args, "quality")); quality != "" {
		argv = append(argv, "--quality", quality)
	}
	if composition := strings.TrimSpace(argString(args, "composition")); composition != "" {
		argv = append(argv, "--composition", composition)
	}
	if format != "mp4" {
		argv = append(argv, "--format", format)
	}
	// **4K was available on this machine the whole time.** `--resolution` scales
	// the capture by raising Chrome's deviceScaleFactor — "the composition is
	// unchanged", in upstream's own words — so a 1920x1080 scene renders at
	// 3840x2160 without a byte of its CSS moving. `video render` forwarded four
	// of the engine's forty options until 8 ก.ย. 2569 and this was not one of
	// them, which made every clip this office has ever produced 1080p because
	// nobody could ask for anything else. The aspect ratio still has to match
	// the composition and the scale still has to be a whole multiple; those are
	// the engine's rules and it enforces them.
	if resolution := strings.ToLower(strings.TrimSpace(argString(args, "resolution"))); resolution != "" {
		argv = append(argv, "--resolution", resolution)
	}
	// One composition, different words. This is upstream's own answer to the
	// complaint that started this whole day — that every video came out the same
	// shape with the words changed — and it had no way through the tool: values
	// merged over the composition's `data-composition-variables` defaults and
	// read back inside the scene through `window.__hyperframes.getVariables()`.
	// Passed through as the caller wrote it, because it is JSON on both sides
	// and re-encoding it here would only add a place for it to change shape.
	if variables := strings.TrimSpace(argString(args, "variables")); variables != "" {
		argv = append(argv, "--variables", variables)
	}
	// Off by default, and it stays off by default for the reason the check
	// below already gives: half of what the layout pass reports on this library
	// is the design of the scene. What changes is that a caller who wants the
	// render to stop on a lint error can now say so, instead of the tool having
	// decided for everyone that it never should.
	if argBool(args, "strict") {
		argv = append(argv, "--strict")
	}
	// `seconds` was in this action's signature until 7 ก.ย. 2569 and was never
	// read — no flag was built from it, and the renderer has none to build:
	// `hyperframes render --help` offers output, resolution, format, fps,
	// quality, composition, variables and encoder settings, and nothing that
	// changes how long the piece runs. Duration is `data-duration` in the markup
	// and only there.
	//
	// Still in the schema, because one packed tool has one schema and `new`
	// needs it; what changed on 8 ก.ย. is that the schema now says whose
	// parameter it is, which is the claim the prose was making on its own. And
	// answered rather than ignored when a caller sends it here anyway, because a
	// parameter that has been advertised gets tried. It does not fail the
	// render: the file that comes out is the right file, and refusing to produce
	// it would punish the caller for our own old lie.
	var lengthNote string
	if asked := argFloat(args, "seconds"); asked > 0 {
		lengthNote = "\nseconds ไม่ใช่ตัวเลือกของ render และไม่ได้ถูกใช้ — ความยาวอยู่ที่ `data-duration` บนธาตุรากในมาร์กอัป"
		if fixed := videoRootDuration(full); fixed > 0 {
			lengthNote += fmt.Sprintf(" ซึ่งตอนนี้คือ %.4g วินาที", fixed)
		}
		lengthNote += "\nจะเปลี่ยนความยาวต้องแก้ค่านั้นพร้อมกับคีย์เฟรมที่วิ่งอยู่ในนั้น แล้วเรนเดอร์ใหม่"
	}

	// The check rides inside the render, and it gates nothing. Measured on the
	// clean run of 31 ส.ค. (session 192150): every clip pays a final
	// check-then-render as two adjacent model rounds, the second check warm at
	// ~9s and its round carrying ~90k of context. Folding it here makes
	// "edit → render" the whole closing move: the findings arrive beside the
	// file in one reply, and the agent that wants a look BEFORE spending the
	// render still has `video check` untouched. Not a gate, because half of
	// what the layout pass reports on this library is the design of the scene
	// (see check above) — a render blocked on a deliberate overlap would be
	// the tool overruling the craft. Best effort: a check that cannot run
	// costs the render nothing.
	checkReport, checkRan, _ := runHyperframesReport(ctx, videoCheckTimeout, "check", full)

	// lengthNote rides the failure too, and it did not until 8 ก.ย. 2569: it was
	// built here and appended only to the success report, so a caller who passed
	// `seconds` and got a failed render never learned the argument had been
	// ignored, and sent it again with the next attempt. A correction that only
	// arrives when nothing went wrong is a correction that never reaches the
	// person repeating the mistake.
	if _, err := runHyperframes(ctx, videoRenderTimeout, argv...); err != nil {
		failed := videoFailed(command, err)
		failed.Content += lengthNote
		return failed, err
	}
	produced, err := videoRenderProduced(outFull, format)
	if err != nil {
		// The renderer said it succeeded and there is nothing at the path it was
		// given. Reporting that as success is how a chat ends with a path nobody
		// can open.
		failed := videoFailed(command, err)
		failed.Content += lengthNote
		return failed, err
	}
	report := "เรนเดอร์เสร็จ: " + out + " (" + produced + ")"
	report += lengthNote
	if checkRan && strings.TrimSpace(checkReport) != "" {
		report += "\nรายงาน check ก่อนเรนเดอร์ (อ่านแบบเดียวกับ video check — ที่ตั้งใจซ้อนกันไม่ใช่ปัญหา):\n" + checkReport
	}

	// proof folds the read-back into the same reply. Rendering a draft and then
	// OCR-ing it were two model turns for one deterministic sequence — the
	// sequence belongs here, the judgement on what the letters say stays with
	// the agent. Best effort: a machine that cannot OCR still rendered.
	if argBool(args, "proof") {
		if format == "png-sequence" {
			// A folder of frames is not something OCRVideoFile can open, and
			// letting it try produces an error about a path rather than the
			// sentence the caller needs.
			report += "\nproof อ่านกลับจากไฟล์วิดีโอ ใช้กับ png-sequence ไม่ได้ — เรนเดอร์เป็น mp4 อีกรอบถ้าต้องการอ่านตัวอักษรกลับ"
		} else if text, ocrErr := skill.OCRVideoFile(ctx, outFull, 1); ocrErr != nil {
			report += "\nอ่านข้อความบนจอกลับไม่ได้: " + ocrErr.Error()
		} else {
			report += "\nข้อความบนจอ (OCR ทุก 1 วิ):\n" + text
		}
	}
	report += "\nเปิดให้ผู้ใช้ดูด้วย desk open"
	return skill.Output{Name: "video_render", Command: command, Content: report, Success: true}, nil
}

// videoFormats and videoFormatSuffix are the containers the engine writes, in
// the order `hyperframes render --help` lists them at 0.8.20. Named here rather
// than forwarded blindly so a typo comes back as a sentence naming the five
// instead of as a render that quietly produced an mp4 nobody asked for.
//
// The list is worth reading as capability rather than as file extensions: mov
// and webm carry an alpha channel, which is the difference between a title card
// and an overlay that can sit on somebody else's footage; png-sequence is the
// handoff to After Effects, Nuke and Fusion; gif is the one that can be pasted
// into a pull request. All four were unreachable until 8 ก.ย. 2569.
var videoFormats = []string{"mp4", "webm", "mov", "gif", "png-sequence"}

// videoResolutions are the engine's output presets at 0.8.20. Listed rather
// than free-form WxH because the engine takes presets and only presets, and an
// enum is the difference between the model reading the six sizes it can have
// and guessing at a seventh.
//
// The aliases upstream also accepts (1080p, 4k, uhd, square-1080p and the rest)
// are deliberately not here: six names that mean six sizes teaches the shape of
// the choice, and eleven names that mean six sizes teaches nothing extra.
var videoResolutions = []string{"landscape", "portrait", "square", "landscape-4k", "portrait-4k", "square-4k"}

var videoFormatSuffix = map[string]string{
	"mp4":  ".mp4",
	"webm": ".webm",
	"mov":  ".mov",
	"gif":  ".gif",
	// A directory of RGBA frames, so no suffix and no dot: a folder named
	// <project>.mp4 would be a lie told by the file browser.
	"png-sequence": "-frames",
}

// videoRenderProduced answers what the renderer left behind, or an error saying
// it left nothing.
//
// Two shapes, because png-sequence is a directory of frames and everything else
// is one file. A size check against a directory reads 0 on Windows and would
// have called every successful frame export a failed render.
func videoRenderProduced(path, format string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("เรนเดอร์จบแล้วแต่ไม่มีไฟล์ที่ %s", filepath.Base(path))
	}
	if format == "png-sequence" {
		if !info.IsDir() {
			return "", fmt.Errorf("png-sequence ต้องได้โฟลเดอร์เฟรม แต่ %s ไม่ใช่โฟลเดอร์", filepath.Base(path))
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil || len(entries) == 0 {
			return "", fmt.Errorf("เรนเดอร์จบแล้วแต่ไม่มีเฟรมใน %s", filepath.Base(path))
		}
		return fmt.Sprintf("%d เฟรม", len(entries)), nil
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("เรนเดอร์จบแล้วแต่ไฟล์ที่ %s ว่างเปล่า", filepath.Base(path))
	}
	return fmt.Sprintf("%.1f MB", float64(info.Size())/(1<<20)), nil
}

// projectDir resolves a project path and refuses one that is not a project.
//
// The existence check is not politeness. `render` on a directory with no
// index.html spends a browser launch to find out, and the sentence it comes
// back with is about a composition rather than about a wrong path.
func (s *videoToolSkill) projectDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("ต้องบอก path ของโปรเจกต์")
	}
	root, err := s.sandbox()
	if err != nil {
		return "", err
	}
	// The read side of newProject's placement: a caller repeating the short
	// name it originally asked for still finds the project the session folder
	// holds (write.go's PlacedPath — literal path wins, fallback only fires
	// when nothing is there).
	dir = skill.PlacedPath(root, s.app.outputSubdir, dir)
	full, err := safeSandboxPath(root, dir)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(full, "index.html")); err != nil {
		return "", fmt.Errorf("%s ไม่ใช่โปรเจกต์ฉาก — ไม่มี index.html อยู่ในนั้น", dir)
	}
	return full, nil
}

func (s *videoToolSkill) sandbox() (string, error) {
	root := strings.TrimSpace(s.app.cur().cfg.SandboxRoot)
	if root == "" {
		return "", fmt.Errorf("ยังไม่ได้เปิดโปรเจกต์")
	}
	return root, nil
}

// ---------------------------------------------------------------------------
// running the renderer
// ---------------------------------------------------------------------------

// runHyperframes runs one command and hands back what it said.
//
// The bundled Node running the bundled entry point, never a shim: npm's `.cmd`
// wrapper looks for `node` on PATH, and the whole point of a portable bundle is
// that nothing has to be on PATH for it to work.
func runHyperframes(ctx context.Context, budget time.Duration, argv ...string) (string, error) {
	stdout, stderr, err := spawnHyperframes(ctx, budget, argv...)
	if err != nil {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = strings.TrimSpace(stdout)
		}
		if detail == "" {
			detail = err.Error()
		}
		if statereport.Is(err) {
			return "", err
		}
		return "", fmt.Errorf("%s ไม่สำเร็จ: %s", argv[0], videoTail(detail))
	}
	answer := strings.TrimSpace(stdout)
	if answer == "" {
		answer = strings.TrimSpace(stderr)
	}
	return videoTail(answer), nil
}

// runHyperframesReport is the same run for a command whose non-zero exit means
// "I found something" rather than "I failed".
//
// `ran` is the question the caller actually has: did the program get to say
// anything. False means the renderer is missing, could not start, or ran out of
// its budget — and then err is the sentence to show. True means there is a
// report, whatever the exit code was.
func runHyperframesReport(ctx context.Context, budget time.Duration, argv ...string) (report string, ran bool, err error) {
	stdout, stderr, runErr := spawnHyperframes(ctx, budget, argv...)
	said := strings.TrimSpace(stdout)
	if said == "" {
		said = strings.TrimSpace(stderr)
	}
	if runErr != nil {
		var exit *exec.ExitError
		// An exit code with something written under it is the program
		// answering. Anything else — a missing renderer, a binary that would
		// not start, a deadline — is not.
		if !errors.As(runErr, &exit) || said == "" {
			return "", false, runErr
		}
	}
	return videoTail(said), true, nil
}

// spawnHyperframes runs the renderer once and hands back both streams.
func spawnHyperframes(ctx context.Context, budget time.Duration, argv ...string) (stdout, stderr string, err error) {
	node, entry := hyperframesParts()
	if node == "" {
		return "", "", statereport.New("ยังเรนเดอร์ไม่ได้ เพราะเครื่องนี้ยังไม่มีตัวเรนเดอร์ฉาก — เปิดหน้างานวิดีโอแล้วกดติดตั้ง")
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	cmd := exec.CommandContext(ctx, node, append([]string{entry}, argv...)...)
	proc.HideConsole(cmd)
	root, rootErr := config.DataRoot()
	cmd.Env = append(os.Environ(), videoEnvPairs(root, rootErr == nil)...)
	var out, errs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errs

	runErr := cmd.Run()
	// A budget that ran out is its own sentence: the exit error alone says
	// "signal: killed", which reads as a crash rather than as a wait nobody
	// wanted to keep waiting for.
	if ctx.Err() != nil {
		return out.String(), errs.String(), fmt.Errorf("%s ใช้เวลาเกิน %s จึงหยุดไว้", argv[0], budget)
	}
	return out.String(), errs.String(), runErr
}

// videoEnvPairs is hyperframesEnvironment in the shape exec wants.
func videoEnvPairs(root string, haveRoot bool) []string {
	env := hyperframesEnvironment(root, haveRoot, findProgram)
	pairs := make([]string, 0, len(env))
	for key, value := range env {
		pairs = append(pairs, key+"="+value)
	}
	return pairs
}

// videoTail keeps the end of a long answer rather than the start.
//
// A renderer's useful sentence is its last one — what failed, or where the file
// landed. Truncating from the front would keep the banner and drop the answer.
func videoTail(s string) string {
	const most = 4000
	if len(s) <= most {
		return s
	}
	return "…\n" + s[len(s)-most:]
}

func videoFailed(command string, err error) skill.Output {
	return skill.Output{
		Name:    "video",
		Command: command,
		Content: err.Error(),
		Stderr:  err.Error(),
		Success: false,
		// A missing renderer is a fact about this machine rather than about
		// anything the agent did, and the learning floor reads the difference.
		FromWorld: statereport.Is(err),
	}
}

// argString, argFloat and argInt read one loosely-typed argument. A model sends
// a number as a string as often as not, and refusing that is refusing the call
// over its punctuation.
func argString(args map[string]any, key string) string {
	switch v := args[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case json.Number:
		return v.String()
	}
	return ""
}

func argFloat(args map[string]any, key string) float64 {
	switch v := args[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0
		}
		return f
	}
	return 0
}

func argInt(args map[string]any, key string) int { return int(argFloat(args, key)) }

func argBool(args map[string]any, key string) bool {
	switch v := args[key].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	}
	return false
}
