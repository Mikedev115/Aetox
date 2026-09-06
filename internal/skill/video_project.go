package skill

// video_project turns a Cutfile into a project file an editing program opens.
//
// **The hole it fills.** The editor agent can cut and it can render, and until
// now both roads ended at an mp4 — the one form of an edit nobody can argue
// with. kinocut's Cutfile covers the person who will argue with it in a text
// editor. This covers the person whose editor is DaVinci Resolve: same
// decisions, same file on disk as the source of truth, a second door out.
//
// **It reads the Cutfile rather than taking its own arguments** so there is one
// account of the edit and not two. A cut list that renders differently from the
// cut list that was exported is worse than having no exporter.
//
// The XML is internal/nle's job; this file's job is the three things that
// package cannot know: where the file is allowed to be, how long the sources
// actually are, and which of the Cutfile's operations survive the trip.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/nle"
	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/statereport"
)

type videoProjectSkill struct {
	root  string
	files *FileState
}

func (*videoProjectSkill) Name() string { return "video_project" }

func (*videoProjectSkill) Description() string {
	return "แปลงรายการตัด (Cutfile) เป็นไฟล์โปรเจกต์ FCPXML ที่เปิดแก้ต่อได้ใน Resolve, Premiere หรือ Final Cut"
}

// ToolDefinition carries existence and signature only; everything a caller
// needs to know once is in Guidance below (block standard, guidance.go).
//
// There is no `output` parameter, and that is a decision rather than an
// omission: the cutfile's own folder is the workspace its media paths are
// relative to, so the project belongs beside it under the same name, and the
// parameter that would let it be put somewhere else cost more of the block
// than the choice was worth.
func (*videoProjectSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"cutfile": map[string]any{
				"type":        "string",
				"description": "Path to the cutfile .json",
			},
		},
		"required":             []string{"cutfile"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "video_project",
			Description: "Write a cutfile's edit as an FCPXML project file an editing program can open.",
			Parameters:  payload,
		},
	}
}

// Guidance is the judgment layer: what this is for, what it does not carry,
// and the failure that does not look like one.
func (*videoProjectSkill) Guidance(map[string]any) string {
	return strings.Join([]string{
		"The file lands beside the cutfile, named after it, and opens in DaVinci Resolve, Premiere Pro and Final Cut.",
		"It carries the cut list and only the cut list: which piece of which source, in what order.",
		"Crops, titles, burnt-in subtitles and every audio decision stay outside it — the report names what was left behind, and the user has to be told, because in their editor it looks like work that was never done rather than work that did not travel.",
		"Media is referenced by absolute path, so moving the footage breaks the project.",
		"Write the edit as .json. kinocut's own YAML scaffold reads a filled-in file as an edit with no operations and does not complain.",
	}, " ")
}

func (s *videoProjectSkill) Execute(ctx context.Context, input Input) (Output, error) {
	start := time.Now()
	args := stringSlice(input["args"])
	if len(args) == 0 {
		err := errors.New("usage: video_project <cutfile.json>")
		return newToolOutput("video_project", "video_project", "", start, false, err), err
	}
	return s.run(ctx, start, args[0])
}

func (s *videoProjectSkill) ExecuteTool(ctx context.Context, args map[string]any) (Output, error) {
	cutfile, _ := args["cutfile"].(string)
	cutfile = strings.TrimSpace(cutfile)
	if cutfile == "" {
		err := errors.New("cutfile is required")
		return newToolOutput("video_project", "video_project", "", time.Now(), false, err), err
	}
	return s.run(ctx, time.Now(), cutfile)
}

func (s *videoProjectSkill) run(ctx context.Context, start time.Time, cutfileArg string) (Output, error) {
	command := "video_project " + cutfileArg
	fail := func(err error) (Output, error) {
		return newToolOutput("video_project", command, "", start, false, err), err
	}

	cutfilePath, err := resolveSandboxPath(s.root, cutfileArg)
	if err != nil {
		return fail(err)
	}
	raw, err := os.ReadFile(cutfilePath)
	if err != nil {
		return fail(err)
	}
	var cf cutfileDoc
	if err := json.Unmarshal(raw, &cf); err != nil {
		// The scaffold kinocut writes is YAML, and its own loader reads a
		// filled-in one as an empty edit without complaining. Saying which
		// file to write instead is the whole of the fix.
		if strings.HasSuffix(strings.ToLower(cutfilePath), ".yaml") || strings.HasSuffix(strings.ToLower(cutfilePath), ".yml") {
			return fail(fmt.Errorf("%s is YAML; write the edit as a .json cutfile instead — "+
				"kinocut's YAML support is the empty scaffold only, and a filled-in one is read as an edit with no operations", cutfileArg))
		}
		return fail(fmt.Errorf("%s is not a valid cutfile: %w", cutfileArg, err))
	}
	if strings.TrimSpace(cf.Name) == "" {
		cf.Name = strings.TrimSuffix(filepath.Base(cutfilePath), filepath.Ext(cutfilePath))
	}

	plan, err := readCutfilePlan(cf)
	if err != nil {
		return fail(err)
	}

	baseDir := filepath.Dir(cutfilePath)
	sources := make([]nle.Source, 0, len(plan.sourceOrder))
	for _, id := range plan.sourceOrder {
		abs, err := s.resolveRelativeToCutfile(baseDir, plan.sources[id])
		if err != nil {
			return fail(err)
		}
		probed, err := probeSource(ctx, abs)
		if err != nil {
			return fail(err)
		}
		probed.ID = id
		sources = append(sources, probed)
	}

	timeline := nle.Timeline{
		Name:    cf.Name,
		Width:   plan.width,
		Height:  plan.height,
		Sources: sources,
		Clips:   plan.clips,
	}
	// The sequence takes the first source's shape unless the cutfile resized
	// it — an edit that never says what it is going to be is the size of the
	// footage it was cut from.
	if len(sources) > 0 {
		timeline.Rate = sources[0].Rate
		if timeline.Width == 0 || timeline.Height == 0 {
			timeline.Width, timeline.Height = sources[0].Width, sources[0].Height
		}
	}
	// A cutfile with no trim at all is still an edit: every source end to end,
	// in the order it was declared.
	if len(timeline.Clips) == 0 {
		for _, src := range sources {
			timeline.Clips = append(timeline.Clips, nle.Clip{
				Source: src.ID, Name: src.Name, In: 0, Out: src.Seconds,
			})
		}
	}
	// A source joined whole by a merge had no end until ffprobe supplied one:
	// the plan is read before any file is measured, on purpose, so that a
	// cutfile that cannot be built at all fails before spending a probe.
	lengths := map[string]float64{}
	for _, src := range sources {
		lengths[src.ID] = src.Seconds
	}
	for i := range timeline.Clips {
		if timeline.Clips[i].Out == 0 {
			timeline.Clips[i].Out = lengths[timeline.Clips[i].Source]
		}
	}

	document, err := timeline.FCPXML()
	if err != nil {
		return fail(err)
	}
	placed, err := timeline.Place()
	if err != nil {
		return fail(err)
	}

	outArg := strings.TrimSuffix(cutfileArg, filepath.Ext(cutfileArg)) + ".fcpxml"
	outPath, err := resolveSandboxPath(s.root, outArg)
	if err != nil {
		return fail(err)
	}
	if err := s.files.guardStale(outPath, outArg); err != nil {
		return fail(err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fail(err)
	}
	if err := os.WriteFile(outPath, document, 0o644); err != nil {
		return fail(err)
	}
	s.files.Note(outPath)

	report := renderProjectReport(outArg, timeline, placed, plan.unrepresented)
	truncated, wasTruncated := limitLines(report, defaultToolOutputLineLimit)
	return newToolOutput("video_project", command, truncated, start, wasTruncated, nil), nil
}

// resolveRelativeToCutfile applies the Cutfile's own rule — source paths are
// relative to the folder the cutfile sits in — and then hands the result back
// through the sandbox guard every other tool uses, so the rule cannot be used
// to step outside the project.
func (s *videoProjectSkill) resolveRelativeToCutfile(baseDir, sourcePath string) (string, error) {
	joined := sourcePath
	if !filepath.IsAbs(joined) {
		joined = filepath.Join(baseDir, filepath.FromSlash(sourcePath))
	}
	rel, err := filepath.Rel(s.root, joined)
	if err != nil {
		return "", fmt.Errorf("source %s is not inside this project", sourcePath)
	}
	return resolveSandboxPath(s.root, rel)
}

// cutfileDoc is the part of kinocut's Cutfile this tool reads. Nothing here
// validates the rest of it: `video_cutfile_validate` owns that question, and a
// second opinion on it in Go would be a second thing to keep in step.
type cutfileDoc struct {
	Name    string           `json:"name"`
	Sources []cutfileSource  `json:"sources"`
	Ops     []map[string]any `json:"ops"`
}

type cutfileSource struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type cutfilePlan struct {
	sources       map[string]string
	sourceOrder   []string
	clips         []nle.Clip
	width, height int
	unrepresented []string
}

// readCutfilePlan reads the edit out of a cutfile's operations.
//
// **What survives the trip, and why the rest does not.** FCPXML as this package
// writes it is a spine of cuts: which piece of which source, in what order.
// `trim` is that, `merge` is the order, and `resize` is the frame the sequence
// is cut for. `crop`, `add_text`, `burn_in`, `convert` and `composite_layers`
// are all real work that this file cannot carry, and the answer to that is to
// name them in the report rather than to write half of them and let the user
// discover the other half is missing in their editor.
func readCutfilePlan(cf cutfileDoc) (cutfilePlan, error) {
	plan := cutfilePlan{sources: map[string]string{}}
	for _, s := range cf.Sources {
		id := strings.TrimSpace(s.ID)
		if id == "" || strings.TrimSpace(s.Path) == "" {
			return plan, fmt.Errorf("a cutfile source is missing its id or path")
		}
		if _, dup := plan.sources[id]; dup {
			return plan, fmt.Errorf("two cutfile sources share the id %q", id)
		}
		plan.sources[id] = s.Path
		plan.sourceOrder = append(plan.sourceOrder, id)
	}
	if len(plan.sourceOrder) == 0 {
		return plan, errors.New("the cutfile declares no sources, so there is nothing to build a project from")
	}

	// Step ids that produced a clip, so a later merge can put them in order.
	fromStep := map[string]nle.Clip{}
	var order []string
	lastRef := plan.sourceOrder[0]
	dropped := map[string]int{}

	for i, op := range cf.Ops {
		name, _ := op["op"].(string)
		id, _ := op["id"].(string)
		if strings.TrimSpace(id) == "" {
			id = fmt.Sprintf("op%d", i+1)
		}
		switch name {
		case "trim":
			src, _ := op["src"].(string)
			if strings.TrimSpace(src) == "" {
				src = lastRef
			}
			if _, known := plan.sources[src]; !known {
				// A trim of an earlier step's output is a nested edit this
				// spine has no way to say. Naming it beats guessing at it.
				dropped["trim (of another step's output)"]++
				continue
			}
			startSec, okStart := cutfileNumber(op["start"])
			durSec, okDur := cutfileNumber(op["duration"])
			if !okStart {
				startSec = 0
			}
			if !okDur {
				return plan, fmt.Errorf("ops[%d] is a trim with no duration, so its clip has no length", i)
			}
			clip := nle.Clip{Source: src, Name: id, In: startSec, Out: startSec + durSec}
			fromStep[id] = clip
			order = append(order, id)
			lastRef = id
		case "merge":
			srcs := cutfileStrings(op["srcs"])
			if len(srcs) == 0 {
				continue
			}
			var reordered []string
			for _, ref := range srcs {
				if _, ok := fromStep[ref]; ok {
					reordered = append(reordered, ref)
					continue
				}
				if _, ok := plan.sources[ref]; ok {
					// A whole untrimmed source joined into the edit.
					stepID := "whole:" + ref
					fromStep[stepID] = nle.Clip{Source: ref, Name: ref, In: 0, Out: 0}
					reordered = append(reordered, stepID)
				}
			}
			if len(reordered) > 0 {
				order = reordered
			}
			lastRef = id
		case "resize":
			if w, ok := cutfileNumber(op["width"]); ok {
				plan.width = int(w)
			}
			if h, ok := cutfileNumber(op["height"]); ok {
				plan.height = int(h)
			}
			lastRef = id
		case "probe", "":
			// A probe produces no media and changes no picture.
		default:
			dropped[name]++
			lastRef = id
		}
	}

	for _, id := range order {
		plan.clips = append(plan.clips, fromStep[id])
	}
	for name, n := range dropped {
		if n == 1 {
			plan.unrepresented = append(plan.unrepresented, name)
			continue
		}
		plan.unrepresented = append(plan.unrepresented, fmt.Sprintf("%s (x%d)", name, n))
	}
	sort.Strings(plan.unrepresented)
	return plan, nil
}

// cutfileStrings reads a JSON list of names. `stringSlice` is the wrong tool
// here and silently so: it accepts []string, which is what a CLI argument
// arrives as, while a decoded JSON array is []any and would come back empty —
// a merge that quietly kept no order at all.
func cutfileStrings(v any) []string {
	var out []string
	switch list := v.(type) {
	case []string:
		out = append(out, list...)
	case []any:
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	}
	kept := out[:0]
	for _, s := range out {
		if s = strings.TrimSpace(s); s != "" {
			kept = append(kept, s)
		}
	}
	return kept
}

// cutfileNumber accepts what JSON gives back for a number a person typed, and
// the string form too: a cutfile is written by hand as often as by a tool.
func cutfileNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	}
	return 0, false
}

// probeSource asks ffprobe what a file actually is. Nothing here is guessed
// from a filename: an edit written against a duration nobody measured is an
// edit with a clip hanging off the end of its source.
func probeSource(ctx context.Context, path string) (nle.Source, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, bundledBinary("ffmpeg", "ffprobe"),
		"-v", "error", "-show_streams", "-show_format", "-of", "json", path)
	proc.HideConsole(cmd)
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nle.Source{}, missingFFmpegError()
		}
		return nle.Source{}, fmt.Errorf("ffprobe could not read %s: %w", filepath.Base(path), err)
	}
	var probed struct {
		Streams []struct {
			CodecType   string `json:"codec_type"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
			RFrameRate  string `json:"r_frame_rate"`
			Channels    int    `json:"channels"`
			Duration    string `json:"duration"`
			AvgFrameRat string `json:"avg_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &probed); err != nil {
		return nle.Source{}, fmt.Errorf("ffprobe answered something unreadable about %s: %w", filepath.Base(path), err)
	}

	src := nle.Source{Name: filepath.Base(path), Path: path}
	for _, st := range probed.Streams {
		switch st.CodecType {
		case "video":
			if src.Width == 0 {
				src.Width, src.Height = st.Width, st.Height
				rate := st.RFrameRate
				if rate == "" || rate == "0/0" {
					rate = st.AvgFrameRat
				}
				src.Rate = parseRational(rate)
			}
		case "audio":
			src.HasAudio = true
			if st.Channels > src.AudioChannels {
				src.AudioChannels = st.Channels
			}
		}
	}
	if secs, err := strconv.ParseFloat(strings.TrimSpace(probed.Format.Duration), 64); err == nil {
		src.Seconds = secs
	}
	if src.Width == 0 || src.Height == 0 {
		return nle.Source{}, statereport.New(fmt.Sprintf("ไม่พบภาพในไฟล์ %s จึงทำเป็นโปรเจกต์ตัดต่อไม่ได้", filepath.Base(path)))
	}
	if !src.Rate.Valid() {
		return nle.Source{}, statereport.New(fmt.Sprintf("อ่านเฟรมเรตของ %s ไม่ได้ ไฟล์โปรเจกต์ต้องรู้เฟรมเรตถึงจะวางคัตให้ตรงเฟรมได้", filepath.Base(path)))
	}
	return src, nil
}

// parseRational reads ffprobe's "30/1" or "30000/1001" without turning it into
// a float on the way past. 29.97 is not a number; 30000/1001 is.
func parseRational(s string) nle.Rate {
	num, den, ok := strings.Cut(strings.TrimSpace(s), "/")
	if !ok {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return nle.Rate{Num: n, Den: 1}
		}
		return nle.Rate{}
	}
	n, errN := strconv.Atoi(strings.TrimSpace(num))
	d, errD := strconv.Atoi(strings.TrimSpace(den))
	if errN != nil || errD != nil {
		return nle.Rate{}
	}
	return nle.Rate{Num: n, Den: d}
}

// renderProjectReport says what was written in the terms the agent has to
// repeat to the user: real timecodes, taken from the frame arithmetic rather
// than from the seconds that were asked for.
func renderProjectReport(outPath string, t nle.Timeline, placed []nle.PlacedClip, unrepresented []string) string {
	var b strings.Builder
	var total int64
	for _, p := range placed {
		total = p.AtFrames + p.LenFrames
	}
	fmt.Fprintf(&b, "wrote %s — FCPXML, opens in DaVinci Resolve, Premiere Pro and Final Cut\n", outPath)
	fmt.Fprintf(&b, "%dx%d at %s fps, %d clip(s), %s long\n\n",
		t.Width, t.Height, trimFPS(t.Rate.FPS()), len(placed), timecode(t.Seconds(total)))
	for i, p := range placed {
		fmt.Fprintf(&b, "%d. %s — %s from %s for %s, lands at %s\n",
			i+1, p.Name, p.Source,
			timecode(t.Seconds(p.InFrames)),
			timecode(t.Seconds(p.LenFrames)),
			timecode(t.Seconds(p.AtFrames)))
	}
	if len(unrepresented) > 0 {
		fmt.Fprintf(&b, "\nnot carried into the project file: %s\n", strings.Join(unrepresented, ", "))
		b.WriteString("the project holds the cut list only — say so rather than letting the user find out in their editor\n")
	}
	b.WriteString("\nthe media is referenced by absolute path, so the project breaks if the footage moves\n")
	return b.String()
}

func timecode(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	minutes := int(seconds) / 60
	rest := seconds - float64(minutes*60)
	return fmt.Sprintf("%d:%06.3f", minutes, rest)
}

func trimFPS(fps float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.3f", fps), "0"), ".")
}
