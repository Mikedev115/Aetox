package engine

// คลังสตูดิโอ — the studio's shelf of raw material, and the one tool that reads
// it. The engine half: the catalogue, the scan, the search. The folder dialog
// that starts a scan is a door and lives in screen_doors.go, by §248 A5's rule
// (dialog → a path → an engine binding).
//
// Two agents and nobody else. `video` dresses a scene it has already decided
// on; `editor` drops a sound or an overlay onto footage it has already cut.
// Both reach the shelf through `asset_find`, which is filed under
// deliverables (internal/skill/category.go) so it rides the `tools:` line of
// the agent that makes videos and the open list of the one that cuts them,
// and lands on no desk — a desk asked to write a memo has no use for a
// whoosh.
//
// The material is never fetched by Aetox and never copied wholesale. A scan
// records facts about files where they are; `asset_find use` copies ONE file
// into the project the moment an agent picks it, which is the same "material
// beside the scene" contract `video new` keeps for a template's own sounds.
// internal/assetlib's package comment has the licence reasoning behind the
// first half of that sentence.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/assetlib"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// assetFindToolName is the one name the shelf answers to in a tool block.
const assetFindToolName = "asset_find"

// StudioLibraryView is one shelf as the Settings page shows it: the counts
// and the size, never the rows. Thousands of rows have no place on a screen
// whose job is "which folders, and are they scanned".
type StudioLibraryView struct {
	ID string `json:"id"`
	// Name is what the card is called — the folder's own name, or the
	// bundled shelf's title. Root is the whole path, for the small line.
	Name    string         `json:"name"`
	Root    string         `json:"root"`
	Builtin bool           `json:"builtin"`
	License string         `json:"license,omitempty"`
	Source  string         `json:"source,omitempty"`
	Scanned string         `json:"scanned"`
	Bytes   int64          `json:"bytes"`
	Files   int            `json:"files"`
	Unread  int            `json:"unread"`
	Counts  map[string]int `json:"counts"`
	Missing bool           `json:"missing"`
}

// studioProgress is one step of a scan on the wire.
type studioProgress struct {
	ID    string `json:"id"`
	Root  string `json:"root"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

// studioDone is a scan's final word.
type studioDone struct {
	ID    string `json:"id,omitempty"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	// What the scan found, for the one card a person reads right after
	// pressing: files, bytes, and how many of each kind.
	Files  int            `json:"files,omitempty"`
	Bytes  int64          `json:"bytes,omitempty"`
	Counts map[string]int `json:"counts,omitempty"`
}

// studioScan is the one scan allowed in flight, for the reason
// capabilityInstall gives: two presses must not race two walks of the same
// folder into one catalogue.
type studioScan struct {
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// studioStore loads the catalogue. Every reader takes a fresh copy from disk
// rather than sharing one in memory, because the scan writes it from another
// goroutine and the file is small next to the material it describes.
func studioStore() (*assetlib.Store, string, error) {
	root, err := studioDataRoot()
	if err != nil {
		return nil, "", err
	}
	s, err := assetlib.Load(root)
	if err != nil {
		return nil, "", err
	}
	return s, root, nil
}

// studioShelves is every shelf an agent or the page can see: the bundled
// one(s) first, then the user's, in the order they were added. The bundled
// shelf is rebuilt from the binary each call and never enters the store, so
// the store stays a list of the user's own folders.
//
// A bundled shelf that cannot unpack (a read-only data root) is left out and
// logged nowhere louder than the page: the user's shelves still come back.
//
// The user's corrections (Store.Overrides) are applied here, so every reader
// — the agent's tool, the readiness row, the browser — sees the corrected
// kind and never sees a hidden row. The browser alone may ask for hidden rows
// back (studioShelvesAll) to let the user un-hide one.
//
// A shelf whose folder is gone is NOT among them (owner, 13 ก.ย. 2026:
// "ไม่มีก็ไม่ควรแสดงดิ"): a drive unplugged or a folder moved leaves its
// rows in the catalogue — that is what lets it come back without a rescan —
// but neither the agent nor the browser may be handed a file that cannot be
// opened. Only StudioLibraries (the Settings list, studioShelvesListed)
// still sees such a shelf, marked, so the person can see why and remove it.
func studioShelves() ([]*assetlib.Library, *assetlib.Store, string, error) {
	return studioShelvesWith(false, false)
}

// studioShelvesAll is studioShelves with hidden rows kept and marked.
func studioShelvesAll() ([]*assetlib.Library, *assetlib.Store, string, error) {
	return studioShelvesWith(true, false)
}

// studioShelvesListed is every shelf the user has added, present or not.
func studioShelvesListed() ([]*assetlib.Library, *assetlib.Store, string, error) {
	return studioShelvesWith(false, true)
}

func studioShelvesWith(keepHidden, keepMissing bool) ([]*assetlib.Library, *assetlib.Store, string, error) {
	s, root, err := studioStore()
	if err != nil {
		return nil, nil, "", err
	}
	builtin, _ := assetlib.Builtin(root)
	libs := make([]*assetlib.Library, 0, len(builtin)+len(s.Libraries))
	libs = append(libs, builtin...)
	for _, lib := range s.Libraries {
		if keepMissing || shelfPresent(lib) {
			libs = append(libs, lib)
		}
	}
	return s.Apply(libs, keepHidden), s, root, nil
}

// shelfPresent is whether a shelf's folder is where the catalogue says.
func shelfPresent(lib *assetlib.Library) bool {
	st, err := os.Stat(lib.Root)
	return err == nil && st.IsDir()
}

// StudioSetKind records the user's correction of one asset's kind ("" puts
// the scan's own guess back). Takes effect for the agent on its next call:
// asset_find reads the shelves fresh every time.
func (a *Engine) StudioSetKind(id, kind string) error {
	k := assetlib.Kind(strings.ToLower(strings.TrimSpace(kind)))
	if k != "" {
		known := false
		for _, want := range assetlib.Kinds {
			if want == k {
				known = true
			}
		}
		if !known {
			return fmt.Errorf("ไม่รู้จักชนิด %q", kind)
		}
	}
	s, root, err := studioStore()
	if err != nil {
		return err
	}
	s.SetKind(strings.TrimSpace(id), k)
	return s.Save(root)
}

// StudioSetHidden keeps one asset from the agent (and from the browser's
// default view), or gives it back.
func (a *Engine) StudioSetHidden(id string, hidden bool) error {
	s, root, err := studioStore()
	if err != nil {
		return err
	}
	s.SetHidden(strings.TrimSpace(id), hidden)
	return s.Save(root)
}

// StudioAssetPath is where one asset lives on this machine — the engine's
// half of the screen's RevealStudioAsset (screen_doors.go), which opens it
// in the file manager where the platform allows.
func (a *Engine) StudioAssetPath(id string) (string, error) {
	libs, _, _, err := studioShelvesAll()
	if err != nil {
		return "", err
	}
	lib, as, ok := assetlib.Find(libs, strings.TrimSpace(id))
	if !ok {
		return "", errors.New("ไม่มีวัตถุดิบนี้")
	}
	return lib.FullPath(*as), nil
}

// studioDataRoot is the one place the studio's files hang off.
func studioDataRoot() (string, error) { return config.DataRoot() }

// StudioLibraries lists every shelf, bundled first. Never nil (§34).
func (a *Engine) StudioLibraries() []StudioLibraryView {
	out := []StudioLibraryView{}
	libs, _, _, err := studioShelvesListed()
	if err != nil {
		return out
	}
	for _, lib := range libs {
		out = append(out, studioView(lib))
	}
	return out
}

func studioView(lib *assetlib.Library) StudioLibraryView {
	v := StudioLibraryView{
		ID:      lib.ID,
		Name:    lib.Name(),
		Root:    lib.Root,
		Builtin: lib.Builtin,
		License: lib.License,
		Source:  lib.Source,
		Bytes:   lib.Bytes,
		Files:   len(lib.Assets),
		Unread:  lib.Unread,
		Counts:  map[string]int{},
	}
	if !lib.Scanned.IsZero() {
		v.Scanned = lib.Scanned.Format(time.RFC3339)
	}
	for k, n := range lib.Counts() {
		v.Counts[string(k)] = n
	}
	// A shelf whose folder is gone (a drive unplugged, a folder renamed) is
	// still listed here, marked, so the user can see why the agent finds
	// nothing — and only here (studioShelves).
	v.Missing = !shelfPresent(lib)
	return v
}

// StudioScanning reports whether a scan is in flight, so a Settings page
// opened mid-scan draws the bar instead of the button.
func (a *Engine) StudioScanning() bool {
	a.studio.mu.Lock()
	defer a.studio.mu.Unlock()
	return a.studio.running
}

// AddStudioLibraryAt catalogues one folder and returns at once. Progress
// arrives as "studio:progress" events and the outcome as "studio:done".
// Returns true if this call started the work; false means one is already
// running, or the folder is not a folder.
//
// Rescanning is the same call with the same folder: the library id is a hash
// of the path, so the new catalogue replaces the old one.
func (a *Engine) AddStudioLibraryAt(dir string) (bool, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" || dir == "." {
		return false, errors.New("ยังไม่ได้เลือกโฟลเดอร์")
	}
	st, err := os.Stat(dir)
	if err != nil {
		return false, err
	}
	if !st.IsDir() {
		return false, fmt.Errorf("%s ไม่ใช่โฟลเดอร์", dir)
	}
	tools := assetlib.FFTools{FFmpeg: findProgram("ffmpeg"), FFprobe: findProgram("ffprobe")}
	if tools.FFprobe == "" {
		// Stills would still be read, but a shelf is mostly sound and clips,
		// and a catalogue with no lengths on it would be a worse thing to have
		// than a clear sentence about what is missing.
		return false, errors.New("ต้องมี ffmpeg (ffprobe) ก่อนจึงจะอ่านความยาวของเสียงและคลิปได้ — ติดตั้งได้จากหน้างานวิดีโอ")
	}

	a.studio.mu.Lock()
	if a.studio.running {
		a.studio.mu.Unlock()
		return false, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.studio.running = true
	a.studio.cancel = cancel
	a.studio.mu.Unlock()

	go a.runStudioScan(ctx, dir, tools)
	return true, nil
}

// CancelStudioScan stops the scan in flight, if any. The catalogue keeps
// whatever it held before the scan began.
func (a *Engine) CancelStudioScan() {
	a.studio.mu.Lock()
	defer a.studio.mu.Unlock()
	if a.studio.cancel != nil {
		a.studio.cancel()
	}
}

func (a *Engine) runStudioScan(ctx context.Context, dir string, tools assetlib.FFTools) {
	defer func() {
		a.studio.mu.Lock()
		a.studio.running = false
		a.studio.cancel = nil
		a.studio.mu.Unlock()
	}()

	// Every probe would otherwise be an event; on a shelf of several thousand
	// files that is several thousand messages into a bar with two hundred
	// pixels. Whole-percent changes only, plus the first and the last.
	last := -1
	lib, err := assetlib.Scan(ctx, dir, tools, func(p assetlib.Progress) {
		pct := 0
		if p.Total > 0 {
			pct = p.Done * 100 / p.Total
		}
		if pct == last && p.Done != p.Total {
			return
		}
		last = pct
		a.emitEvent("studio:progress", studioProgress{Root: dir, Done: p.Done, Total: p.Total})
	})
	if err != nil {
		a.emitEvent("studio:done", studioDone{OK: false, Error: err.Error()})
		return
	}
	s, root, err := studioStore()
	if err != nil {
		a.emitEvent("studio:done", studioDone{OK: false, Error: err.Error()})
		return
	}
	s.Put(lib)
	if err := s.Save(root); err != nil {
		a.emitEvent("studio:done", studioDone{OK: false, Error: err.Error()})
		return
	}
	counts := map[string]int{}
	for k, n := range lib.Counts() {
		counts[string(k)] = n
	}
	a.emitEvent("studio:done", studioDone{ID: lib.ID, OK: true, Files: len(lib.Assets), Bytes: lib.Bytes, Counts: counts})
}

// RescanStudioLibrary walks a shelf again, for a folder the user has added
// files to since. Same contract as AddStudioLibraryAt.
func (a *Engine) RescanStudioLibrary(id string) (bool, error) {
	s, _, err := studioStore()
	if err != nil {
		return false, err
	}
	lib, ok := s.Get(strings.TrimSpace(id))
	if !ok {
		// The bundled shelf is catalogued at build time; there is nothing to
		// rescan and the page does not offer it.
		return false, errors.New("ไม่มีคลังนี้")
	}
	return a.AddStudioLibraryAt(lib.Root)
}

// RemoveStudioLibrary forgets a shelf. The user's files are not touched, and
// the button that calls this must say so.
func (a *Engine) RemoveStudioLibrary(id string) ([]StudioLibraryView, error) {
	s, root, err := studioStore()
	if err != nil {
		return a.StudioLibraries(), err
	}
	if !s.Remove(strings.TrimSpace(id)) {
		return a.StudioLibraries(), errors.New("ไม่มีคลังนี้")
	}
	if err := s.Save(root); err != nil {
		return a.StudioLibraries(), err
	}
	return a.StudioLibraries(), nil
}

// StudioLibraryPath is the folder of one shelf — the screen's RevealStudioLibrary opens it; a shelf's folder in the file manager — the
// bundled one included, because "where did these sounds come from" is a fair
// question and the folder holds the licence beside them.
func (a *Engine) StudioLibraryPath(id string) (string, error) {
	libs, _, _, err := studioShelves()
	if err != nil {
		return "", err
	}
	for _, lib := range libs {
		if lib.ID == strings.TrimSpace(id) {
			return lib.Root, nil
		}
	}
	return "", errors.New("ไม่มีคลังนี้")
}

// studioReady is the readiness row for the shelf: "ok" with the count when
// something is catalogued, "optional" when nothing is — a video is made
// without a shelf every day, so an empty one is not a fault.
//
// With a shelf bundled in the binary the row is "ok" on every machine, and
// the number says how much is there — the bundled hundred alone, or the
// user's thousands on top.
func studioReady() ReadyRow {
	libs, _, _, err := studioShelves()
	if err != nil || len(libs) == 0 {
		return ReadyRow{ID: "assets", State: "optional"}
	}
	files := 0
	for _, lib := range libs {
		files += len(lib.Assets)
	}
	return ReadyRow{ID: "assets", State: "ok", Where: fmt.Sprintf("%d", files)}
}

// ---------------------------------------------------------------- the tool

// assetFindSkill is the shelf as the two video agents see it.
type assetFindSkill struct {
	app *Engine
}

func (*assetFindSkill) Name() string { return assetFindToolName }

func (*assetFindSkill) Description() string {
	return "ค้นคลังวัตถุดิบของสตูดิโอ — เสียงประกอบ overlay ฉากหลัง ไอคอน — แล้วหยิบไฟล์ที่เลือกเข้าโปรเจกต์"
}

func (s *assetFindSkill) ToolDefinition() model.ToolDefinition {
	// Signature only; the judgment is in Guidance, sent once on the first call
	// (internal/skill/guidance.go).
	return toolDef(assetFindToolName,
		"Search the studio's shelf of raw material (sound effects, music, overlays, backgrounds, icons, images) that the user has added, and copy one into the project.\n"+
			"`query` (text?, kind?, max_seconds?, alpha?, limit?) — find rows by words in their names and folders.\n"+
			"`use` (id, into) — copy one asset into the project at `into` (a folder relative to the project root) and get back the path to reference.\n"+
			"`summary` — how much of each kind is on the shelf.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{"type": "string", "enum": []string{"query", "use", "summary"}},
				"text":   map[string]any{"type": "string", "description": "`query`. Words to match, all of them; a prefix matches (\"whoo\" finds whoosh)."},
				"kind": map[string]any{
					"type":        "string",
					"enum":        kindNames(),
					"description": "`query`. sfx: short sound. music: a bed. overlay: has alpha or is green-screen. background: opaque loop. clip: other footage. icon: small still to place. image: other still.",
				},
				"max_seconds": map[string]any{"type": "number", "description": "`query`. Drop anything longer; stills always pass."},
				"alpha":       map[string]any{"type": "boolean", "description": "`query`. Only material with a real alpha channel — stricter than kind=overlay, which also admits green-screen footage that still needs keying."},
				"limit":       map[string]any{"type": "integer", "description": "`query`. Rows to return, default 20."},
				"id":          map[string]any{"type": "string", "description": "`use`. The id from a query row."},
				"into":        map[string]any{"type": "string", "description": "`use`. Folder inside the project to copy into, e.g. the scene's own folder or assets/."},
			},
			"required": []string{"action"},
		})
}

func kindNames() []string {
	out := make([]string, 0, len(assetlib.Kinds))
	for _, k := range assetlib.Kinds {
		out = append(out, string(k))
	}
	return out
}

// Guidance is what an agent needs once: when the shelf is the right place to
// look, and the two things about it that do not look like failures.
func (*assetFindSkill) Guidance(map[string]any) string {
	return strings.TrimSpace(`
The shelf is material the USER chose and put on this machine; Aetox did not pick it and cannot vouch for its licence. Use it the way you would use a file they attached.

Decide the piece first, then look. A search run to find out what to make is the shelf deciding the work. Arrive with a need — "a two-second whoosh for the title reveal", "a looping dark grid behind the chart" — and query for that; an empty shelf is an ordinary answer, and so is "nothing here fits", after which you build the piece without it.

Kind is inferred from names and probes and can be wrong. Every row carries the user's own folder name as its category; when the kind looks off, read that. A file with no duration is a still, or one the probe could not open.

"use" copies the file beside the scene and returns the path to write into the markup or hand to the editor. Copy, do not link to the shelf: a project that references a folder on a drive that is later unplugged renders a black box. Say which asset you took, by name, when you hand the piece back — the user knows their shelf and will want to know what you reached for.`)
}

func (s *assetFindSkill) ExecuteTool(ctx context.Context, args map[string]any) (skill.Output, error) {
	return s.run(ctx, args)
}

func (s *assetFindSkill) Execute(ctx context.Context, input skill.Input) (skill.Output, error) {
	return s.run(ctx, map[string]any(input))
}

func (s *assetFindSkill) run(_ context.Context, args map[string]any) (skill.Output, error) {
	action := strings.ToLower(strings.TrimSpace(argString(args, "action")))
	if action == "" {
		action = "query"
	}
	command := assetFindToolName + " " + action
	libs, _, _, err := studioShelves()
	if err != nil {
		return assetFailed(command, err), err
	}
	if len(libs) == 0 {
		// Only when even the bundled shelf could not unpack: the honest state
		// of that machine, said the way the renderer says it is not installed.
		return skill.Output{Name: assetFindToolName, Command: command, Success: true,
			Content: "คลังสตูดิโอยังว่าง — ผู้ใช้เพิ่มโฟลเดอร์วัตถุดิบได้ที่ ตั้งค่า → คลังสตูดิโอ ระหว่างนี้ทำงานต่อได้โดยไม่ใช้วัตถุดิบจากคลัง"}, nil
	}
	switch action {
	case "query":
		return s.query(libs, command, args)
	case "use":
		return s.use(libs, command, args)
	case "summary":
		return skill.Output{Name: assetFindToolName, Command: command, Success: true, Content: summaryText(libs)}, nil
	}
	err = fmt.Errorf("asset_find ไม่รู้จักคำสั่ง %q", action)
	return assetFailed(command, err), err
}

func (s *assetFindSkill) query(libs []*assetlib.Library, command string, args map[string]any) (skill.Output, error) {
	q := assetlib.Query{
		Text:       argString(args, "text"),
		Kind:       assetlib.Kind(strings.ToLower(strings.TrimSpace(argString(args, "kind")))),
		MaxSeconds: argFloat(args, "max_seconds"),
		NeedAlpha:  argBool(args, "alpha"),
		Limit:      int(argFloat(args, "limit")),
	}
	hits, total := assetlib.Search(libs, q)
	if len(hits) == 0 {
		return skill.Output{Name: assetFindToolName, Command: command, Success: true,
			Content: "ไม่มีวัตถุดิบที่ตรง — " + summaryText(libs)}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d of %d\n", len(hits), total)
	b.WriteString("id | name | kind | category | length | size | alpha\n")
	for _, a := range hits {
		length := "-"
		if a.Duration > 0 {
			length = fmt.Sprintf("%.1fs", a.Duration)
		}
		size := "-"
		if a.Width > 0 {
			size = fmt.Sprintf("%dx%d", a.Width, a.Height)
		}
		alpha := ""
		if a.HasAlpha {
			alpha = "alpha"
		}
		fmt.Fprintf(&b, "%s | %s | %s | %s | %s | %s | %s\n", a.ID, a.Name(), a.Kind, a.Category, length, size, alpha)
	}
	return skill.Output{Name: assetFindToolName, Command: command, Success: true, Content: strings.TrimRight(b.String(), "\n")}, nil
}

// use copies one asset into the project. The destination follows the same
// placement rule every file-producing tool keeps: inside the sandbox, or
// refused.
func (s *assetFindSkill) use(libs []*assetlib.Library, command string, args map[string]any) (skill.Output, error) {
	id := strings.TrimSpace(argString(args, "id"))
	into := strings.TrimSpace(argString(args, "into"))
	if id == "" || into == "" {
		err := errors.New("asset_find use ต้องบอกทั้ง id และ into")
		return assetFailed(command, err), err
	}
	lib, asset, ok := assetlib.Find(libs, id)
	if !ok {
		err := fmt.Errorf("ไม่มีวัตถุดิบ id %s ในคลัง — ค้นด้วย query ก่อน", id)
		return assetFailed(command, err), err
	}
	root := strings.TrimSpace(s.app.cur().cfg.SandboxRoot)
	if root == "" {
		err := errors.New("ยังไม่ได้เปิดโปรเจกต์")
		return assetFailed(command, err), err
	}
	destDir := into
	if !filepath.IsAbs(destDir) {
		destDir = filepath.Join(root, into)
	}
	destDir = filepath.Clean(destDir)
	rel, err := filepath.Rel(root, destDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		err := fmt.Errorf("ปลายทางต้องอยู่ในโปรเจกต์: %s", into)
		return assetFailed(command, err), err
	}
	src := lib.FullPath(*asset)
	dest := filepath.Join(destDir, filepath.Base(src))
	if err := copyFileInto(src, dest); err != nil {
		return assetFailed(command, err), err
	}
	relDest, _ := filepath.Rel(root, dest)
	report := fmt.Sprintf("copied %s → %s", asset.Name(), filepath.ToSlash(relDest))
	if asset.Duration > 0 {
		report += fmt.Sprintf(" (%.1fs)", asset.Duration)
	}
	if asset.HasAlpha {
		report += " alpha"
	}
	return skill.Output{Name: assetFindToolName, Command: command, Success: true, Content: report}, nil
}

func copyFileInto(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("อ่านไฟล์ต้นทางไม่ได้ (คลังยังอยู่ที่เดิมไหม): %w", err)
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dest)
		return err
	}
	return out.Close()
}

func summaryText(libs []*assetlib.Library) string {
	counts := assetlib.Summary(libs)
	var parts []string
	for _, k := range assetlib.Kinds {
		if n := counts[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", k, n))
		}
	}
	if len(parts) == 0 {
		return "คลังสตูดิโอว่าง"
	}
	return fmt.Sprintf("shelf: %s (%d shelves)", strings.Join(parts, ", "), len(libs))
}

func assetFailed(command string, err error) skill.Output {
	return skill.Output{Name: assetFindToolName, Command: command, Content: err.Error(), Success: false}
}
