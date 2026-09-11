package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
)

// solidPNG is a picture of one colour, the size asked for — what a screen
// that can draw hands back for a slide.
func solidPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func deckHTML(title string, slides ...string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><title>" + title + "</title></head><body>")
	for _, s := range slides {
		b.WriteString(`<section class="slide"><h1>` + s + `</h1><ul><li>ประเด็น</li></ul>` +
			`<aside class="notes">พูดช้า ๆ ตรงนี้</aside></section>`)
	}
	b.WriteString("</body></html>")
	return b.String()
}

// deckApp gives the engine a project of its own. Where an export lands is
// the screen's business now (desktop/exports.go), so nothing here can drop a
// file into the developer's real Downloads.
func deckApp(t *testing.T) (*Engine, string) {
	t.Helper()
	root := t.TempDir()
	return seed(&Engine{cfg: config.Config{SandboxRoot: root}}, newConversation()), root
}

func writeUnder(t *testing.T, root, rel, body string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func TestListDecksFindsDecksAndSkipsPages(t *testing.T) {
	a, root := deckApp(t)
	writeUnder(t, root, "output/20260819-010101.111/เสนอราคา.html", deckHTML("ก", "หนึ่ง", "สอง", "สาม"))
	writeUnder(t, root, "output/20260819-010101.111/index.html", "<html><body><section><h1>หน้าเว็บ</h1></section></body></html>")
	writeUnder(t, root, "output/20260819-010101.111/บันทึก.md", "# ไม่ใช่เด็ค")

	decks := a.ListDecks()
	if len(decks) != 1 {
		t.Fatalf("got %d decks, want 1: %+v", len(decks), decks)
	}
	if decks[0].Slides != 3 {
		t.Errorf("slides = %d, want 3", decks[0].Slides)
	}
	// The path has to be the one the file host and the pane both speak, or the
	// row lists fine and opens onto a blank iframe.
	if want := "output/20260819-010101.111/เสนอราคา.html"; decks[0].Path != want {
		t.Errorf("path = %q, want %q", decks[0].Path, want)
	}
	if filepath.IsAbs(decks[0].Path) {
		t.Errorf("path must be project-relative, got %q", decks[0].Path)
	}
	if decks[0].SessionID != "20260819-010101.111" {
		t.Errorf("sessionId = %q", decks[0].SessionID)
	}
}

// A deck loose in output/ predates per-session folders. It is still a deck; it
// just has no conversation to point back at, and the honest answer is "".
func TestDeckLooseInOutputHasNoSession(t *testing.T) {
	a, root := deckApp(t)
	writeUnder(t, root, "output/เก่า.html", deckHTML("ก", "หนึ่ง"))

	decks := a.ListDecks()
	if len(decks) != 1 {
		t.Fatalf("got %d decks, want 1", len(decks))
	}
	if decks[0].SessionID != "" {
		t.Errorf("sessionId = %q, want empty", decks[0].SessionID)
	}
}

// aged writes a deck and dates it, because every assertion about a range is an
// assertion about when the file was written.
func aged(t *testing.T, root, rel string, daysAgo int, slides ...string) {
	t.Helper()
	full := writeUnder(t, root, rel, deckHTML("ก", slides...))
	when := time.Now().AddDate(0, 0, -daysAgo).Add(-time.Hour)
	if err := os.Chtimes(full, when, when); err != nil {
		t.Fatal(err)
	}
}

// The room reloads on every finished turn, so listing without a bound charged a
// full read and a full HTML parse of every deck the workspace had ever produced
// to every one of those turns. The range is that bound, and this is the guard
// on it: what is outside the range must not be opened at all.
//
// Counted by what came back *and* by the fact that the old deck is not in it —
// a listing that read everything and then filtered would pass an assertion
// about rows and be the exact cost this replaced. The read is what the filter
// has to come before, which is why candidatesWithin takes deckFile and not Deck.
func TestTheWeekRangeLeavesOlderDecksUnopened(t *testing.T) {
	a, root := deckApp(t)
	aged(t, root, "output/s-new/สัปดาห์นี้.html", 1, "หนึ่ง", "สอง")
	aged(t, root, "output/s-mid/เดือนนี้.html", 14, "หนึ่ง")
	aged(t, root, "output/s-old/นานแล้ว.html", 200, "หนึ่ง")

	week := a.ListDecksIn(RangeWeek)
	if week.Range != RangeWeek {
		t.Errorf("range = %q, want %q", week.Range, RangeWeek)
	}
	if len(week.Decks) != 1 || week.Decks[0].Name != "สัปดาห์นี้.html" {
		t.Fatalf("week = %+v, want only this week's deck", week.Decks)
	}
	if week.Total != 1 {
		t.Errorf("total = %d, want 1 — with no cap it is exactly what is in range", week.Total)
	}

	month := a.ListDecksIn(RangeMonth)
	if len(month.Decks) != 2 {
		t.Fatalf("month = %+v, want two decks", month.Decks)
	}
	// Newest first, the order the room draws and the day headings depend on.
	if month.Decks[0].Name != "สัปดาห์นี้.html" {
		t.Errorf("month is not newest first: %+v", month.Decks)
	}

	if all := a.ListDecksIn(RangeAll); len(all.Decks) != 3 {
		t.Errorf("all = %d decks, want 3", len(all.Decks))
	}
}

// An empty week is indistinguishable from a broken room, so it widens — and it
// has to widen on the number of *decks*, not on the number of .html files the
// walk found. A week holding nothing but web pages is an empty week.
func TestAWeekOfNonDecksWidensRatherThanShowingNothing(t *testing.T) {
	a, root := deckApp(t)
	page := writeUnder(t, root, "output/s-new/หน้าเว็บ.html",
		"<html><body><section><h1>ไม่ใช่เด็ค</h1></section></body></html>")
	when := time.Now().AddDate(0, 0, -1)
	if err := os.Chtimes(page, when, when); err != nil {
		t.Fatal(err)
	}
	aged(t, root, "output/s-old/เด็คเก่า.html", 120, "หนึ่ง")

	got := a.ListDecksIn(RangeWeek)
	if len(got.Decks) != 1 || got.Decks[0].Name != "เด็คเก่า.html" {
		t.Fatalf("did not widen past a week of non-decks: %+v", got.Decks)
	}
	if got.Range != RangeAll {
		t.Errorf("range = %q, want %q — the picker has to say what is on screen", got.Range, RangeAll)
	}
}

// The widening chain must not pay for the same file twice. Without the memo,
// a week that widens to the month and then to everything reads the oldest decks
// three times over.
func TestWideningDoesNotRereadWhatItAlreadyOpened(t *testing.T) {
	a, root := deckApp(t)
	aged(t, root, "output/s-old/เก่า.html", 300, "หนึ่ง")

	got := a.ListDecksIn(RangeWeek)
	if len(got.Decks) != 1 || got.Range != RangeAll {
		t.Fatalf("got %+v range=%q", got.Decks, got.Range)
	}
	if got.Total != 1 {
		t.Errorf("total = %d, want 1", got.Total)
	}
}

func TestListDecksWithNoProjectIsEmptyNotNil(t *testing.T) {
	a := seed(&Engine{cfg: config.Config{SandboxRoot: ""}}, newConversation())
	// The binding crosses to JS, where a nil slice arrives as null and the room
	// would render `null.length`. Empty is the same answer without the crash.
	if got := a.ListDecks(); got == nil || len(got) != 0 {
		t.Errorf("ListDecks with no project = %#v, want an empty slice", got)
	}
}

func TestListDecksOnAFreshProjectIsEmpty(t *testing.T) {
	a, _ := deckApp(t) // no output folder at all
	if got := a.ListDecks(); len(got) != 0 {
		t.Errorf("a project that has produced nothing = %#v", got)
	}
}

// An export is named for the deck and holds its words; where it lands is the
// screen's business (desktop/exports.go), which is why nothing here has a
// Downloads folder. The words have to survive the trip: a .pptx that opens
// empty is the failure that reads as the feature being broken.
func TestExportIsNamedForTheDeckAndHoldsItsWords(t *testing.T) {
	a, root := deckApp(t)
	rel := "output/s1/เด็ค.html"
	writeUnder(t, root, rel, deckHTML("ก", "ต้นทุนลดลง ๔๐%", "สามอย่างที่เปลี่ยน"))

	export, err := a.DeckExportFiles(rel, "pptx")
	if err != nil {
		t.Fatalf("DeckExportFiles: %v", err)
	}
	if export.Base != "เด็ค" || export.Ext != ".pptx" || export.Folder {
		t.Errorf("export = %q %q folder=%v, want the deck's own name, .pptx, one file", export.Base, export.Ext, export.Folder)
	}
	if len(export.Files) != 1 || export.Files[0].Name != "เด็ค.pptx" {
		t.Fatalf("files = %+v, want the one .pptx under the deck's name", names(export))
	}
	// The deck itself does not move.
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Errorf("the deck was disturbed: %v", err)
	}

	parts := unzipPPTX(t, export.Files[0].Data)
	slide1 := parts["ppt/slides/slide1.xml"]
	if !strings.Contains(slide1, "ต้นทุนลดลง ๔๐%") {
		t.Errorf("slide 1 lost its title:\n%s", slide1)
	}
	if !strings.Contains(slide1, "ประเด็น") {
		t.Errorf("slide 1 lost its bullet")
	}
	if notes := parts["ppt/notesSlides/notesSlide1.xml"]; !strings.Contains(notes, "พูดช้า ๆ ตรงนี้") {
		t.Errorf("the speaker notes did not come across:\n%s", notes)
	}
	if _, ok := parts["ppt/slides/slide2.xml"]; !ok {
		t.Error("the second slide is missing")
	}
}

// names is what an export would write, by name.
func names(export DeckExport) []string {
	out := make([]string, 0, len(export.Files))
	for _, f := range export.Files {
		out = append(out, f.Name)
	}
	return out
}

// renderScreen is a window that can draw: it answers RenderDeck with one
// picture per slide (the same solid PNG each time), which is what the
// picture formats need and what testScreen refuses.
type renderScreen struct {
	testScreen
	slides int
	shot   []byte
	asked  []DeckRender
}

func (s *renderScreen) RenderDeck(_ context.Context, _ string, req DeckRender) (DeckRendered, error) {
	s.asked = append(s.asked, req)
	switch req.Kind {
	case "pdf":
		return DeckRendered{PDF: []byte("%PDF-1.7 fake")}, nil
	case "images":
		out := DeckRendered{}
		for i := 0; i < s.slides; i++ {
			out.Images = append(out.Images, s.shot)
		}
		return out, nil
	}
	return DeckRendered{}, fmt.Errorf("unexpected render %q", req.Kind)
}

// The picture formats are the screen's to draw — a deck is a file on the
// engine's host, and what it looks like only a browser can say. The engine
// asks, then packs what came back: one file per slide in a folder for the
// picture formats, and a frozen .pptx of the same pictures for pptx-img.
func TestPictureExportsAskTheScreenForThePixels(t *testing.T) {
	a, root := deckApp(t)
	rel := "output/s1/เด็ค.html"
	writeUnder(t, root, rel, deckHTML("ก", "หนึ่ง", "สอง", "สาม"))
	screen := &renderScreen{slides: 3, shot: solidPNG(t, 16, 9, color.Black)}
	a.screen = screen

	pdf, err := a.DeckExportFiles(rel, "pdf")
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	if pdf.Folder || len(pdf.Files) != 1 || pdf.Files[0].Name != "เด็ค.pdf" || string(pdf.Files[0].Data) != "%PDF-1.7 fake" {
		t.Errorf("pdf export = %v folder=%v, want the one PDF the screen drew", names(pdf), pdf.Folder)
	}

	png, err := a.DeckExportFiles(rel, "png")
	if err != nil {
		t.Fatalf("png: %v", err)
	}
	if !png.Folder || png.Ext != ".png" {
		t.Errorf("png export folder=%v ext=%q, want a folder of pictures", png.Folder, png.Ext)
	}
	// 01, 02, 03: a ten-slide deck sorts 01..10 in every file browser rather
	// than 1, 10, 2.
	if got := strings.Join(names(png), ","); got != "01.png,02.png,03.png" {
		t.Errorf("png files = %s", got)
	}

	frozen, err := a.DeckExportFiles(rel, "pptx-img")
	if err != nil {
		t.Fatalf("pptx-img: %v", err)
	}
	// Both pptx rows carry the same extension, and they are not
	// interchangeable: one is editable and plain, the other exact and frozen.
	// Sharing a name would have the second export silently replace the first.
	if frozen.Base != "เด็ค-img" || frozen.Ext != ".pptx" {
		t.Errorf("pptx-img = %q %q, want the deck's name with -img and .pptx", frozen.Base, frozen.Ext)
	}
	parts := unzipPPTX(t, frozen.Files[0].Data)
	if _, ok := parts["ppt/slides/slide3.xml"]; !ok {
		t.Error("the frozen deck lost a slide")
	}
	if notes := parts["ppt/notesSlides/notesSlide1.xml"]; !strings.Contains(notes, "พูดช้า ๆ ตรงนี้") {
		t.Errorf("the frozen deck dropped the presenter's notes:\n%s", notes)
	}

	// Every one of those went through the screen, with the deck as a file URL.
	if len(screen.asked) != 3 {
		t.Errorf("the screen was asked %d times, want once per export", len(screen.asked))
	}
}

// A deck may be called anything; a file may not.
func TestADecksNameIsMadeSafeForAFilename(t *testing.T) {
	if got := SanitiseFileName(`ราคา: Q4/2026 <ร่าง>`); strings.ContainsAny(got, `<>:"/\|?*`) {
		t.Errorf("sanitiseFileName left something Windows refuses: %q", got)
	}
	if got := SanitiseFileName("   "); got != "deck" {
		t.Errorf("an empty name = %q, want a usable fallback", got)
	}
	// The name is still the deck's. Sanitising must not turn it into a hash.
	if got := SanitiseFileName("เสนอราคา"); got != "เสนอราคา" {
		t.Errorf("a perfectly good Thai name was mangled to %q", got)
	}
}

func TestExportDeckRefusesAFormatItCannotWrite(t *testing.T) {
	a, root := deckApp(t)
	rel := "output/s1/เด็ค.html"
	writeUnder(t, root, rel, deckHTML("ก", "หนึ่ง"))

	// Formats nothing writes, and the empty string. Every row the menu actually
	// offers has left this list as it shipped — a format that is ready must
	// never be asserted here, because on a machine with no webview it would
	// fail for the wrong reason and go red the day one existed. What guards the
	// other direction is TestEveryReadyFormatActuallyWrites.
	for _, format := range []string{"odp", "key", "mp4", ""} {
		if _, err := a.DeckExportFiles(rel, format); err == nil {
			t.Errorf("DeckExportFiles(%q) succeeded, but nothing writes that yet", format)
		}
	}
}

// The menu and the writer read one list, so a row cannot say "ready" while
// the export refuses it, or the reverse. The formats that need pixels get
// them from a screen that can draw (renderScreen), so every ready row is
// written here rather than closed out by hand.
func TestEveryReadyFormatActuallyWrites(t *testing.T) {
	a, root := deckApp(t)
	rel := "output/s1/เด็ค.html"
	writeUnder(t, root, rel, deckHTML("ก", "หนึ่ง"))
	a.screen = &renderScreen{slides: 1, shot: solidPNG(t, 16, 9, color.White)}

	formats := a.DeckFormats()
	if len(formats) == 0 {
		t.Fatal("the export menu is empty")
	}
	ready := 0
	for _, f := range formats {
		if !f.Ready {
			continue
		}
		ready++
		out, err := a.DeckExportFiles(rel, f.ID)
		if err != nil {
			t.Errorf("%s is marked ready and refused: %v", f.ID, err)
			continue
		}
		if out.Ext != f.Ext {
			t.Errorf("%s wrote %q, which is not the %q the menu promised", f.ID, out.Ext, f.Ext)
		}
		if len(out.Files) == 0 {
			t.Errorf("%s wrote nothing", f.ID)
		}
	}
	if ready == 0 {
		t.Error("no format is ready, so the export button can never do anything")
	}
}

// The export webview is a browser tab, so it reaches the deck as a file:// URL
// rather than through the app's own /aetox-file/ host. A Thai filename or a
// space that arrives unescaped is a deck the engine cannot find, which prints
// as its own error page.
func TestFileURLEscapesEverySegment(t *testing.T) {
	got := FileURLForPath(`D:\งาน\output\s1\เสนอ ราคา.html`)
	if !strings.HasPrefix(got, "file:///") {
		t.Errorf("%q does not start with file:///", got)
	}
	if strings.Contains(got, " ") {
		t.Errorf("%q still contains a raw space", got)
	}
	if strings.Contains(got, `\`) {
		t.Errorf("%q still contains a backslash", got)
	}
	// The separators must survive: escaping the whole path at once turns them
	// into %2F and the engine gets one very long filename.
	if n := strings.Count(got, "/"); n < 5 {
		t.Errorf("%q lost its separators (%d slashes)", got, n)
	}
	if !strings.Contains(got, url.PathEscape("เสนอ ราคา.html")) {
		t.Errorf("%q does not carry the escaped filename", got)
	}
}

func TestExportDeckRefusesAPageWithNoSlides(t *testing.T) {
	a, root := deckApp(t)
	rel := "output/s1/index.html"
	writeUnder(t, root, rel, "<html><body><h1>หน้าเว็บธรรมดา</h1></body></html>")

	_, err := a.DeckExportFiles(rel, "pptx")
	if err == nil {
		t.Fatal("a page with no slides was exported")
	}
	// The remedy belongs in the message: whoever reads it has to know what to
	// change, and "no slides" alone does not say.
	if !strings.Contains(err.Error(), `<section class="slide">`) {
		t.Errorf("the error must say what a slide is: %v", err)
	}
}

func TestExportDeckStaysInsideTheProject(t *testing.T) {
	a, root := deckApp(t)
	writeUnder(t, root, "output/s1/เด็ค.html", deckHTML("ก", "หนึ่ง"))

	for _, escape := range []string{
		"../outside.html",
		"output/../../outside.html",
	} {
		if _, err := a.DeckExportFiles(escape, "pptx"); err == nil {
			t.Errorf("DeckExportFiles(%q) was allowed out of the project", escape)
		}
	}
}

// Gone is the one answer the window can translate (§133), so it has to be
// the sentinel and not a Win32 sentence.
func TestExportDeckOnAMissingFileSaysItIsGone(t *testing.T) {
	a, _ := deckApp(t)
	if _, err := a.DeckExportFiles("output/s1/หายไป.html", "pptx"); err != ErrFileGone {
		t.Fatalf("exporting a file that is not there = %v, want ErrFileGone", err)
	}
}

func unzipPPTX(t *testing.T, data []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("the export is not a readable package: %v", err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = string(body)
	}
	return out
}

// The skeleton the slides skill hands out has to survive the whole export path,
// not just the parser.
//
// internal/skill already checks that `internal/deck` reads three slides in it
// (bundled_slides_skeleton_test.go). That proves it is a deck; it does not prove
// it is a deck somebody can hand to PowerPoint, which is the thing a user
// actually does with it. Between the two sit deck.Slides and ooxml.BuildPPTX,
// and a skeleton that parses but exports empty would look exactly like a working
// one right up until the click.
//
// Runs here rather than in internal/skill because the export is the Engine's.
func TestTheSlidesSkeletonExportsToARealPPTX(t *testing.T) {
	skeleton := skeletonFromTheSlidesSkill(t)

	a, root := deckApp(t)
	rel := "output/s1/skeleton.html"
	writeUnder(t, root, rel, skeleton)

	export, err := a.DeckExportFiles(rel, "pptx")
	if err != nil {
		t.Fatalf("the documented skeleton does not export: %v", err)
	}
	data := export.Files[0].Data
	// A .pptx is a ZIP of XML parts. Anything this small is an empty shell.
	if len(data) < 4096 {
		t.Errorf("the export is %d bytes — too small to hold three slides", len(data))
	}

	slides := 0
	for name := range unzipPPTX(t, data) {
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			slides++
		}
	}
	// However many the skeleton shows — the point is that every one of them
	// arrives, not that the document keeps a particular number of them.
	if want := strings.Count(skeleton, `class="slide"`); slides != want {
		t.Errorf("the .pptx holds %d slides, the skeleton has %d", slides, want)
	}
}

// skeletonFromTheSlidesSkill reads the largest ```html block out of the bundled
// aetox-slides document — the block a model copies before it changes a word.
//
// Read off disk rather than through internal/skill, which exposes a bundled
// skill's body only to its own package. The path is the same folder //go:embed
// compiles in, so the two cannot disagree about which document this is.
func skeletonFromTheSlidesSkill(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "skill", "skills", "aetox-slides", "SKILL.md"))
	if err != nil {
		t.Fatalf("the slides skill is not where it is compiled in from: %v", err)
	}
	best, rest := "", string(raw)
	for {
		start := strings.Index(rest, "```html")
		if start < 0 {
			break
		}
		rest = rest[start+len("```html"):]
		end := strings.Index(rest, "```")
		if end < 0 {
			break
		}
		if block := rest[:end]; len(block) > len(best) {
			best = block
		}
		rest = rest[end:]
	}
	if best == "" {
		t.Fatal("the slides skill carries no html block — the skeleton is the part that gets copied")
	}
	return best
}

// The slide templates are not decks, and this asks the real ones.
//
// Every file in the aetox-slide-templates skill is a paste block: a <style> and
// one <section class="slide">, with the 1280x720 box and the palette left to the
// skeleton it gets pasted into. On 7 ก.ย. a session copied this whole repo into
// its output/ folder and the room listed forty-four of them, then rendered each
// one honestly — a heading against the left edge of a stage it never asked for.
//
// Written against the files on disk rather than a fragment made up here, for the
// reason skeletonFromTheSlidesSkill exists: the rule is only worth anything if
// it holds for the documents that actually shipped.
func TestSlideTemplatesDoNotListAsDecks(t *testing.T) {
	dir := filepath.Join("..", "skill", "skills", "aetox-slide-templates", "slides")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("the slide templates are not where they are compiled in from: %v", err)
	}

	a, root := deckApp(t)
	copied := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		writeUnder(t, root, "output/20260907-123107.699/templates/"+e.Name(), string(body))
		copied++
	}
	if copied == 0 {
		t.Fatal("no templates copied — the folder moved and this test stopped testing anything")
	}
	writeUnder(t, root, "output/20260907-123107.699/เสนอราคา.html", deckHTML("ก", "หนึ่ง", "สอง"))

	decks := a.ListDecks()
	if len(decks) != 1 {
		t.Fatalf("got %d decks from %d templates and one deck, want 1: %+v", len(decks), copied, decks)
	}
	if decks[0].Name != "เสนอราคา.html" {
		t.Errorf("listed %q, want the deck", decks[0].Name)
	}
}

// Deleting from the room, which is the second door onto the same act ผลงาน owns
// (COMPANY.md §6.7) rather than a second rule about it.
func TestDeleteDeckRemovesTheFile(t *testing.T) {
	a, root := deckApp(t)
	rel := "output/20260819-010101.111/เสนอราคา.html"
	full := writeUnder(t, root, rel, deckHTML("ก", "หนึ่ง"))

	if err := a.DeleteDeck(rel); err != nil {
		t.Fatalf("DeleteDeck: %v", err)
	}
	if _, err := os.Stat(full); !os.IsNotExist(err) {
		t.Errorf("the file is still there: %v", err)
	}
	if decks := a.ListDecks(); len(decks) != 0 {
		t.Errorf("the room still lists %d decks", len(decks))
	}
}

// The binding takes a project-relative path and calls os.Remove, and it is
// reachable from a click. output/ is the only folder the listing looks in, so it
// is the only folder this may empty — a deck path pointing at the source tree is
// a bug or an attack, and either way the answer is no.
func TestDeleteDeckRefusesOutsideOutput(t *testing.T) {
	a, root := deckApp(t)
	full := writeUnder(t, root, "src/index.html", deckHTML("ก", "หนึ่ง"))

	if err := a.DeleteDeck("src/index.html"); err == nil {
		t.Error("a file outside output/ was deleted")
	}
	if _, err := os.Stat(full); err != nil {
		t.Errorf("the file was removed anyway: %v", err)
	}
	if err := a.DeleteDeck("../outside.html"); err == nil {
		t.Error("a path climbing out of the project was accepted")
	}
}

// Two clicks arrive from a room that reloads on every finished turn, so the file
// can go between the listing and the second click. The user asked for it not to
// be there, and it is not there.
func TestDeleteDeckOnAFileAlreadyGoneIsFine(t *testing.T) {
	a, _ := deckApp(t)
	if err := a.DeleteDeck("output/20260819-010101.111/ไม่มีแล้ว.html"); err != nil {
		t.Errorf("DeleteDeck on a missing file: %v", err)
	}
}
