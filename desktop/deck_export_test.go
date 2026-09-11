package main

// Where an export lands (exports.go) and how a deck is printed
// (deck_pdf.go): the screen's half of the deck export. What an export
// contains is the engine's (engine.DeckExportFiles, tested there), so the
// engine here is one that answers with a canned export.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// exportApp is a screen whose engine hands back the export it is given, with
// a Downloads folder of its own. The second half is not tidiness: without it
// every run of these tests would drop files into the developer's real
// Downloads.
func exportApp(t *testing.T, export engine.DeckExport, err error) *App {
	t.Helper()
	a := newTestApp(t)
	a.exports.root = t.TempDir()
	a.api = engineWith{API: a.api, export: func(string, string) (engine.DeckExport, error) { return export, err }}
	return a
}

func onePPTX(base string) engine.DeckExport {
	return engine.DeckExport{Base: base, Ext: ".pptx", Files: []engine.ExportFile{{Name: base + ".pptx", Data: []byte("PK fake pptx")}}}
}

// An export lands in the machine's Downloads folder, not beside the deck.
//
// It was beside the deck first, on the argument that this app already had one
// answer to "where do produced files go". That is right for what the agent
// produces and wrong for what a person asked for by pressing a button: an
// export is a file somebody is about to attach to an email, and Downloads is
// where every other program on the machine puts that. Owner's call. And it is
// THIS machine's Downloads — the screen's — never the engine's host (§248).
func TestExportLandsInDownloadsUnderTheDecksName(t *testing.T) {
	a := exportApp(t, onePPTX("เด็ค"), nil)

	out, err := a.ExportDeck("output/s1/เด็ค.html", "pptx")
	if err != nil {
		t.Fatalf("ExportDeck: %v", err)
	}
	if !filepath.IsAbs(out) {
		t.Fatalf("got %q, want an absolute path — it is outside the project", out)
	}
	if filepath.Dir(out) != a.exports.root {
		t.Errorf("landed in %q, want the downloads folder %q", filepath.Dir(out), a.exports.root)
	}
	if filepath.Base(out) != "เด็ค.pptx" {
		t.Errorf("named %q, want the deck's own name", filepath.Base(out))
	}
	if got, _ := os.ReadFile(out); string(got) != "PK fake pptx" {
		t.Errorf("the file holds %q, not the bytes the engine handed over", got)
	}
}

// Downloads is shared with every other program, and the file already there may
// be one the user has already sent to somebody. Overwriting it would destroy
// something this app never made.
func TestASecondExportDoesNotOverwriteTheFirst(t *testing.T) {
	a := exportApp(t, onePPTX("เด็ค"), nil)

	first, err := a.ExportDeck("output/s1/เด็ค.html", "pptx")
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.ExportDeck("output/s1/เด็ค.html", "pptx")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("both exports went to %q", first)
	}
	for _, p := range []string{first, second} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%q is not on disk: %v", p, err)
		}
	}
}

// The picture formats are one file per slide, so they land in a folder of
// their own under the deck's name rather than scattering eight siblings
// across Downloads.
func TestAPictureExportIsAFolder(t *testing.T) {
	a := exportApp(t, engine.DeckExport{Base: "เด็ค", Ext: ".png", Folder: true, Files: []engine.ExportFile{
		{Name: "01.png", Data: []byte("one")},
		{Name: "02.png", Data: []byte("two")},
	}}, nil)

	out, err := a.ExportDeck("output/s1/เด็ค.html", "png")
	if err != nil {
		t.Fatalf("ExportDeck: %v", err)
	}
	if filepath.Base(out) != "เด็ค" {
		t.Errorf("landed at %q, want a folder named for the deck", out)
	}
	for _, name := range []string{"01.png", "02.png"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("%s is not in the folder: %v", name, err)
		}
	}
}

// What the engine refuses, the screen refuses in the same words: nothing is
// written, nothing is remembered.
func TestExportPassesTheEnginesRefusalThrough(t *testing.T) {
	a := exportApp(t, engine.DeckExport{}, errors.New("ยังส่งออกเป็น odp ไม่ได้"))

	if _, err := a.ExportDeck("output/s1/เด็ค.html", "odp"); err == nil || !strings.Contains(err.Error(), "odp") {
		t.Fatalf("ExportDeck = %v, want the engine's refusal", err)
	}
	if entries, _ := os.ReadDir(a.exports.root); len(entries) != 0 {
		t.Errorf("a refused export still wrote %v", entries)
	}
}

// Only a file this session actually exported can be opened, because the opener
// takes an absolute path and everything else here refuses one on purpose.
func TestOpenExportRefusesAPathItNeverWrote(t *testing.T) {
	a := exportApp(t, onePPTX("เด็ค"), nil)
	if _, err := a.exportPath(filepath.Join(a.exports.root, "..", "somebody-elses.pptx")); err == nil {
		t.Fatal("a path this app never wrote was offered to open")
	}

	// And one it did write opens — until it is gone, which is the one answer
	// the window can translate (§133).
	out, err := a.ExportDeck("output/s1/เด็ค.html", "pptx")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := a.exportPath(out); err != nil || got != out {
		t.Errorf("exportPath(%q) = %q, %v", out, got, err)
	}
	os.Remove(out)
	if _, err := a.exportPath(out); err != engine.ErrFileGone {
		t.Errorf("exportPath on a removed export = %v, want ErrFileGone", err)
	}
}

// The print settings are the whole difference between a PDF that looks like the
// deck and one that looks like a bug report, and they are checkable here even
// though the printing is not.
func TestPrintSettingsKeepTheDeckLookingLikeItself(t *testing.T) {
	var params map[string]any
	// 0, 0 is "nothing measured" — the fallback path, which is the one this
	// test's numbers describe (deckPageWidthInches × deckPageHeightInches).
	if err := json.Unmarshal([]byte(deckPrintParams(0, 0)), &params); err != nil {
		t.Fatalf("the print parameters are not valid JSON: %v", err)
	}

	// Without this every slide with a colour behind it prints white. The file
	// still opens, so nothing anywhere says why it came out wrong.
	if params["printBackground"] != true {
		t.Error("printBackground is off, so every coloured slide would print white")
	}
	// With this on, Chromium stamps the file:// URL and today's date across
	// every slide.
	if params["displayHeaderFooter"] != false {
		t.Error("displayHeaderFooter is on, so every slide would carry a URL and a date")
	}
	for _, margin := range []string{"marginTop", "marginBottom", "marginLeft", "marginRight"} {
		if params[margin] != float64(0) {
			t.Errorf("%s is %v, but the slide IS the page: a margin shrinks the artwork inside its own paper", margin, params[margin])
		}
	}
	// 1280x720 CSS pixels at 96dpi, which is also ooxml's 12192000 EMU.
	if w := params["paperWidth"]; w != 13.333 {
		t.Errorf("paperWidth = %v, want 13.333in (1280px at 96dpi)", w)
	}
	if h := params["paperHeight"]; h != 7.5 {
		t.Errorf("paperHeight = %v, want 7.5in (720px at 96dpi)", h)
	}
}
