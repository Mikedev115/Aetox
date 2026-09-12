package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/assetlib"
)

// A shelf of the user's own, catalogued straight into the isolated data root
// without a scan: the browser and the host read the catalogue, and the
// catalogue is what is under test here, not ffprobe.
func seedShelf(t *testing.T) (dataRoot string, lib *assetlib.Library) {
	t.Helper()
	dataRoot = t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", dataRoot)
	shelf := t.TempDir()
	files := map[string]string{
		"SFX/whoosh-01.wav":    "RIFF....",
		"SFX/whoosh-02.wav":    "RIFF....",
		"Backgrounds/grid.mp4": "....ftypisom",
		"Icons/rocket.png":     "\x89PNG",
	}
	for rel, body := range files {
		p := filepath.Join(shelf, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lib, err := assetlib.Scan(context.Background(), shelf, assetlib.ProbeFunc(func(context.Context, string) (assetlib.Probe, error) {
		return assetlib.Probe{Duration: 1, Width: 1920, Height: 1080}, nil
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := assetlib.Load(dataRoot)
	s.Put(lib)
	if err := s.Save(dataRoot); err != nil {
		t.Fatal(err)
	}
	return dataRoot, lib
}

func TestStudioAssetsPagesAndFiltersLikeTheAgentTool(t *testing.T) {
	seedShelf(t)
	a := &App{}

	// Everything: the bundled hundred plus the four seeded, two pages of 60.
	all := a.StudioAssets(StudioAssetQuery{})
	if all.Total != 104 || all.Pages != 2 || len(all.Rows) != 60 || all.Page != 1 {
		t.Fatalf("all: total %d pages %d rows %d page %d", all.Total, all.Pages, len(all.Rows), all.Page)
	}
	second := a.StudioAssets(StudioAssetQuery{Page: 2})
	if len(second.Rows) != 44 || second.Page != 2 {
		t.Errorf("page 2: %d rows, page %d", len(second.Rows), second.Page)
	}
	// Past the end lands on the last page rather than an empty one.
	if far := a.StudioAssets(StudioAssetQuery{Page: 9}); far.Page != 2 {
		t.Errorf("page 9 → %d", far.Page)
	}

	// Text + kind is the same Search the agent runs.
	whoosh := a.StudioAssets(StudioAssetQuery{Text: "whoosh", Kind: "sfx"})
	if whoosh.Total != 2 || whoosh.Rows[0].Ext != "wav" || whoosh.Rows[0].Library == "" {
		t.Errorf("whoosh: %+v", whoosh.Rows)
	}
	if !strings.HasPrefix(whoosh.Rows[0].URL, studioHostPrefix) || !strings.HasSuffix(whoosh.Rows[0].URL, ".wav") {
		t.Errorf("url: %s", whoosh.Rows[0].URL)
	}

	// The category list is the user's own folder names, counted before the
	// kind filter narrows the rows — so it still offers Icons on a sfx query.
	names := map[string]int{}
	for _, c := range whoosh.Categories {
		names[c.Name] = c.Count
	}
	if names["SFX"] != 2 || names["Icons"] != 1 || names["Interface Sounds"] != 100 {
		t.Errorf("categories: %v", names)
	}
	// A category filter narrows on its own.
	if icons := a.StudioAssets(StudioAssetQuery{Category: "Icons"}); icons.Total != 1 || icons.Rows[0].Kind != "icon" {
		t.Errorf("Icons: %+v", icons.Rows)
	}
}

func TestStudioHostServesByIdAndNothingElse(t *testing.T) {
	_, lib := seedShelf(t)
	a := &App{}
	var whoosh assetlib.Asset
	for _, as := range lib.Assets {
		if strings.HasSuffix(as.Path, "whoosh-01.wav") {
			whoosh = as
		}
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })
	get := func(target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		a.studioHost(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec
	}

	// A catalogued file, by library id and asset id, with the type the
	// <audio> element needs.
	rec := get(studioHostPrefix + lib.ID + "/" + whoosh.ID + ".wav")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "audio/wav" || rec.Body.String() != "RIFF...." {
		t.Errorf("wav: %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	// The bundled shelf answers the same way.
	builtin, _ := assetlib.Builtin(os.Getenv("AETOX_DATA_ROOT"))
	if rec := get(studioHostPrefix + builtin[0].ID + "/" + builtin[0].Assets[0].ID + ".ogg"); rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "audio/ogg" {
		t.Errorf("builtin: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}

	// No path ever reaches the disk: a path where an id should be is an
	// unknown id, and an unknown library is unknown.
	for _, bad := range []string{
		studioHostPrefix + lib.ID + "/../../etc/passwd",
		studioHostPrefix + lib.ID + "/SFX/whoosh-01.wav",
		studioHostPrefix + "nope/" + whoosh.ID + ".wav",
		studioHostPrefix + lib.ID + "/",
		studioHostPrefix,
	} {
		if rec := get(bad); rec.Code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", bad, rec.Code)
		}
	}
	// Anything outside the prefix passes through untouched.
	if rec := get("/aetox-file/x.png"); rec.Code != http.StatusTeapot {
		t.Errorf("foreign path: %d", rec.Code)
	}
}

// A correction made on the page is what the agent gets on its next call, and
// a hidden file is gone from the tool's answer while the browser can still
// ask to see it.
func TestStudioCorrectionsReachTheAgentTool(t *testing.T) {
	_, lib := seedShelf(t)
	a := &App{}
	var grid, rocket assetlib.Asset
	for _, as := range lib.Assets {
		switch {
		case strings.HasSuffix(as.Path, "grid.mp4"):
			grid = as
		case strings.HasSuffix(as.Path, "rocket.png"):
			rocket = as
		}
	}
	tool := &assetFindSkill{app: a}
	before, _ := tool.run(context.Background(), map[string]any{"action": "query", "text": "grid"})
	if !strings.Contains(before.Content, "| background |") {
		t.Fatalf("before: %s", before.Content)
	}

	if err := a.StudioSetKind(grid.ID, "overlay"); err != nil {
		t.Fatal(err)
	}
	if err := a.StudioSetHidden(rocket.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := a.StudioSetKind(grid.ID, "nonsense"); err == nil {
		t.Error("an unknown kind was accepted")
	}

	after, _ := tool.run(context.Background(), map[string]any{"action": "query", "text": "grid"})
	if !strings.Contains(after.Content, "| overlay |") {
		t.Errorf("agent did not see the corrected kind: %s", after.Content)
	}
	gone, _ := tool.run(context.Background(), map[string]any{"action": "query", "text": "rocket"})
	if strings.Contains(gone.Content, rocket.ID) {
		t.Errorf("hidden asset still offered to the agent: %s", gone.Content)
	}
	// `use` cannot reach a hidden file either.
	if out, err := tool.run(context.Background(), map[string]any{"action": "use", "id": rocket.ID, "into": "x"}); err == nil || out.Success {
		t.Error("use of a hidden asset succeeded")
	}

	// The browser's default view agrees; asking for hidden rows shows it marked.
	if page := a.StudioAssets(StudioAssetQuery{Text: "rocket"}); page.Total != 0 {
		t.Errorf("browser shows hidden by default: %+v", page.Rows)
	}
	page := a.StudioAssets(StudioAssetQuery{Text: "rocket", IncludeHidden: true})
	if page.Total != 1 || !page.Rows[0].Hidden {
		t.Errorf("includeHidden: %+v", page.Rows)
	}
	// And the readiness count drops by the hidden one.
	if row := studioReady(); row.Where != "103" {
		t.Errorf("readiness counts hidden rows: %s", row.Where)
	}
}
