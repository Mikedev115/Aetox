package engine

// The bindings behind the slides room: what decks this workspace has, and
// turning one into a file somebody else's program opens.
//
// A deck is an .html file carrying `<section class="slide">`
// (docs/architecture/html-deck-2026-08-19.md). The authoring format is HTML
// because that is what a model writes best and what a browser already renders,
// which leaves exporting as a separate step — and that separation is the whole
// reason this file exists. `slides_write`, which wrote a .pptx directly, was
// retired for it (§149).
//
// Everything here speaks **project-relative paths**, the same vocabulary the
// file host and the workbench panes use. That is not a style choice: the file
// host resolves against SandboxRoot (filehost.go), so a deck addressed any other
// way lists fine and then renders as a blank iframe. Listing something the
// viewer cannot open is worse than not listing it, so the listing is scoped to
// what the viewer can actually show.

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/deck"
	"github.com/Mikedev115/Aetox/internal/ooxml"
)

// A deck is HTML plus its pictures inline, so it is bigger than a source file
// and much smaller than a video. 40 MB is two dozen embedded screenshots and
// well short of anything that should be scanned on a listing.
const maxDeckBytes = 40 << 20

// Deck is one deck as the room lists it.
type Deck struct {
	// Path is project-relative, which is what SlidesPane and the file host both
	// take. Never absolute: see the file header.
	Path   string `json:"path"`
	Name   string `json:"name"`
	Slides int    `json:"slides"`
	// SessionID is the chat that made it, read off the output folder name, so
	// the room can group by conversation without a table recording it. Empty
	// for a deck sitting loose in output/.
	SessionID string `json:"sessionId,omitempty"`
	Modified  string `json:"modified"` // RFC3339
}

// DeckPage is one range of the room's list.
//
// The same three fields ArtifactPage carries, and deliberately the same: ผลงาน
// and this room are one question asked about two kinds of produced file, and a
// second shape for the answer is a second set of edge cases to get right.
// Range is the range actually served, which is not always the one asked for.
type DeckPage struct {
	Decks []Deck `json:"decks"`
	Range string `json:"range"`
	Total int    `json:"total"`
}

// deckFile is a candidate: everything the walk can know without opening it.
type deckFile struct {
	full     string
	rel      string
	name     string
	session  string
	modified time.Time
}

// deckCandidates is the half of the listing that costs a directory entry each.
//
// Name, size and modification time come off the walk; whether the file is a
// deck and how many slides are in it do not, and that split is the whole reason
// this is its own pass. Deciding costs a full read *and* an HTML parse — the
// package that does it says so and leaves the bound to this caller
// (internal/deck) — and the contract for pictures is that they are embedded, so
// a deck with a dozen screenshots in it is megabytes where a text deck is
// twenty kilobytes. Answering "which decks are recent" out of the walk means
// the reads are spent only on the ones about to be shown.
//
// Newest first, tie broken by path, so two decks written in the same second
// come back in the same order on every call.
func deckCandidates(root string) []deckFile {
	base := filepath.Join(root, outputDir)
	var out []deckFile
	_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil //lint:ignore nilerr an unreadable entry is skipped, not fatal
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext != ".html" && ext != ".htm" {
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil || info.Size() > maxDeckBytes {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		out = append(out, deckFile{
			full:     path,
			rel:      filepath.ToSlash(rel),
			name:     entry.Name(),
			session:  sessionOfOutputPath(base, path),
			modified: info.ModTime(),
		})
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].modified.Equal(out[j].modified) {
			return out[i].rel < out[j].rel
		}
		return out[i].modified.After(out[j].modified)
	})
	return out
}

// candidatesWithin cuts the walk down to one range before anything is opened.
//
// The cutoff comes from rangeCutoff, which within() uses for ผลงาน, so the line
// between "this week" and older is drawn once for both rooms and for the day
// headings the rows are then grouped under.
func candidatesWithin(all []deckFile, name string) []deckFile {
	cutoff, bounded := rangeCutoff(name)
	if !bounded {
		return all
	}
	out := make([]deckFile, 0, len(all))
	for _, c := range all {
		if !c.modified.Before(cutoff) {
			out = append(out, c)
		}
	}
	return out
}

// ListDecks returns every deck in this workspace, for a caller that wants the
// whole list and is prepared to pay for it.
func (a *Engine) ListDecks() []Deck { return a.ListDecksIn(RangeAll).Decks }

// ListDecksIn returns the decks inside one time range, newest first.
//
// Scoped to the open workspace rather than to every root the gallery sweeps,
// and that is the file-host constraint above rather than a product decision:
// a deck from another project cannot be rendered here, so offering it would be
// a row that opens onto nothing.
//
// **A range rather than everything**, which is ผลงาน's shape (artifacts.go) and
// matters more here than it does there. A row in the gallery costs a readdir
// entry and a stat; a row here costs the whole file read and parsed. The room
// reloads on every `agent:done` while it is open, so listing without a bound
// charged a full read of every deck the workspace had ever produced to every
// turn that finished — a cost with no ceiling, growing fastest for whoever uses
// the feature most. A week of decks is what the room opens on now.
//
// **It widens when the range it was given is empty**, on the deck count and not
// on the candidate count: a week holding three .html files that turn out to be
// web pages is an empty week, and a room that answered it with nothing would be
// indistinguishable from the feature being broken. Range comes back saying
// which one answered, so the picker can keep telling the truth.
//
// **No cap on top of the range.** ผลงาน has one because its rows are cheap
// enough to sweep in full and the cap is a backstop on drawing; here a cap
// would have to be applied before the reads to save anything, and a count taken
// before the reads counts .html files rather than decks. So the range is the
// only bound, Total is exactly how many decks are in it, and "ทั้งหมด" costs
// what it says it costs — asked for by name, not paid by default.
func (a *Engine) ListDecksIn(want string) DeckPage {
	root := strings.TrimSpace(a.cur().cfg.SandboxRoot)
	if root == "" {
		// No project open reads as no decks, which is the truth about it. The
		// unfocused workspace has its own SandboxRoot when it is the one open,
		// so this is genuinely "nowhere to look" rather than a missed case.
		return DeckPage{Decks: []Deck{}, Range: RangeAll}
	}

	candidates := deckCandidates(root)
	// Memo across the widening chain. A week that turns out to be empty falls
	// through to the month, and without this the files it already opened would
	// be opened again on the way past.
	read := map[string]*Deck{}
	rows := func(in []deckFile) []Deck {
		out := []Deck{}
		for _, c := range in {
			d, done := read[c.rel]
			if !done {
				d = readDeckRow(c)
				read[c.rel] = d
			}
			if d != nil {
				out = append(out, *d)
			}
		}
		return out
	}

	for _, name := range widenFrom(want) {
		found := rows(candidatesWithin(candidates, name))
		if len(found) == 0 && name != RangeAll {
			continue
		}
		return DeckPage{Decks: found, Range: name, Total: len(found)}
	}
	return DeckPage{Decks: []Deck{}, Range: RangeAll}
}

// readDeckRow opens one candidate and returns its row, or nil if the file is
// not a deck. nil rather than a zero Deck because the caller remembers the
// answer, and "already looked, it is not one" has to be tellable from "not
// looked at yet".
func readDeckRow(c deckFile) *Deck {
	source, err := os.ReadFile(c.full)
	if err != nil {
		return nil
	}
	// A fragment carrying the marker is not a deck (deck.Whole), and it is asked
	// first because it answers off the head of the file: a folder of slide
	// templates costs one tag each here rather than forty-one full parses.
	if !deck.Whole(source) {
		return nil
	}
	// The marker is counted rather than matched, so the row can say how many
	// slides are in there without the caller opening it. A page that is not a
	// deck costs one parse and is dropped.
	n := deck.Count(source)
	if n == 0 {
		return nil
	}
	return &Deck{
		Path:      c.rel,
		Name:      c.name,
		Slides:    n,
		SessionID: c.session,
		Modified:  c.modified.Format(time.RFC3339),
	}
}

// DeleteDeck removes one deck from the project.
//
// ผลงาน has been the only place a produced file dies (COMPANY.md §6.7), and this
// is the second door rather than a second rule: the room lists decks from the
// open project's output/ folder, and until now a row it showed could only be got
// rid of somewhere else, by finding the same file again in a gallery that groups
// by conversation. The room is where the person is already looking at the deck,
// which is the same argument the export button won.
//
// Bounded twice, and the second bound is the one that matters. safeSandboxPath
// keeps the path inside the open project; the output/ check keeps it inside the
// only folder the listing looks in. Without the second, a binding that takes a
// project-relative path and calls os.Remove is a door onto the user's source
// tree — and this one is reachable from a click.
//
// A deck that is already gone is a success: the caller asked for it not to be
// there, and it is not there. The room reloads after, so a file deleted from
// under it leaves the same way.
func (a *Engine) DeleteDeck(relPath string) error {
	root := strings.TrimSpace(a.cur().cfg.SandboxRoot)
	if root == "" {
		return fmt.Errorf("no project open")
	}
	full, err := safeSandboxPath(root, relPath)
	if err != nil {
		return err
	}
	base, err := filepath.Abs(filepath.Join(root, outputDir))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, full)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("ลบได้เฉพาะไฟล์ในโฟลเดอร์ผลงานเท่านั้น")
	}
	info, err := os.Stat(full)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("นี่เป็นโฟลเดอร์ ไม่ใช่ไฟล์เด็ค")
	}
	return os.Remove(full)
}

// FileURLForPath turns an absolute OS path into a file:// URL — the address
// the screen's renderer is handed (Screen.RenderDeck).
//
// The export webview is a browser tab, not the app's own webview, so it cannot
// resolve `/aetox-file/...` — that path is served by the Wails asset handler
// inside the main window (filehost.go). It reaches the deck the way any browser
// reaches a local file. Which works only because the contract requires pictures
// to be embedded: a deck referencing chart.png beside itself would load in the
// pane and print blank boxes.
func FileURLForPath(abs string) string {
	slashed := filepath.ToSlash(abs)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed // C:/x -> /C:/x, so the URL keeps three slashes
	}
	// Each segment escaped on its own, or a Thai filename or a space arrives at
	// the engine as a different path than the one on disk. PathEscape leaves
	// the separators alone, which is why the split is by hand.
	parts := strings.Split(slashed, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return "file://" + strings.Join(parts, "/")
}

// sessionOfOutputPath reads the chat id off the first folder under output/.
// A deck loose in output/ predates per-session folders and has none, which is
// answered as "" rather than guessed.
func sessionOfOutputPath(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// DeckFormat is one row in the export menu.
//
// Ready is here rather than in the pane because whether a format can be written
// is a fact about this binary, not about a button. The menu asks; it does not
// keep its own list. When PrintToPdf lands, one `true` here fills the row in
// without the frontend being touched, and there is never a moment where the two
// lists disagree about what works.
type DeckFormat struct {
	ID    string `json:"id"`
	Ext   string `json:"ext"`
	Ready bool   `json:"ready"`
}

// deckFormats is every format the export menu shows, in menu order.
//
// Ready is what the menu greys a row on, and nothing here may claim it without
// something behind it that writes: a row that looks available and then refuses
// is a lie the user only finds by clicking. TestEveryReadyFormatActuallyWrites
// is what holds that.
//
// The order is the order somebody wants them in: the editable copy, the one that
// looks exactly right, then the pictures for putting a slide in a chat.
var deckFormats = []DeckFormat{
	{ID: "pptx", Ext: ".pptx", Ready: true},
	{ID: "pptx-img", Ext: ".pptx", Ready: true},
	{ID: "pdf", Ext: ".pdf", Ready: true},
	{ID: "png", Ext: ".png", Ready: true},
	{ID: "jpg", Ext: ".jpg", Ready: true},
	{ID: "webp", Ext: ".webp", Ready: true},
}

// DeckFormats is the export menu, straight from the list above.
func (a *Engine) DeckFormats() []DeckFormat { return deckFormats }

// writableDeckFormat resolves an id to the extension it writes, and refuses
// anything this binary cannot actually produce.
func writableDeckFormat(id string) (string, bool) {
	for _, f := range deckFormats {
		if strings.EqualFold(f.ID, strings.TrimSpace(id)) {
			return f.Ext, f.Ready
		}
	}
	return "", false
}

// DeckExport is a deck written out in another format, as bytes: one file, or
// one picture per slide. Where it lands — the machine's Downloads folder — is
// the screen's (ExportDeck, desktop/exports.go); what it contains is decided
// here, from the deck in the project.
type DeckExport struct {
	// Base is the deck's own name, made safe for a file; Ext the extension.
	Base string `json:"base"`
	Ext  string `json:"ext"`
	// Files is the one file, or the slides one picture each — then Folder is
	// set and the names are "01.png", "02.png"…, so a ten-slide deck sorts
	// 01..10 in every file browser rather than 1, 10, 2.
	Folder bool         `json:"folder"`
	Files  []ExportFile `json:"files"`
}

// DeckExportFiles renders a deck in another format and answers with the bytes.
//
// The .pptx it produces is deliberately plainer than the HTML it came from:
// title, bullets, one picture, speaker notes, which is exactly what
// ooxml.BuildPPTX has always written. That gap is the trade the move to HTML
// was made to take — the deck people look at is the HTML, and the .pptx is the
// copy somebody can open in PowerPoint and edit. Making the .pptx match the
// HTML pixel for pixel would mean writing a PowerPoint, which is the thing this
// project has now twice decided not to do.
//
// **One caller, and that is the decision (§153).** For a day this had two: the
// export bar and a `deck_export` tool the model could call. The tool was removed
// because the deck itself never needed it — the deck is the HTML, the slides
// pane reads it, and exporting is the separate act of handing a copy to somebody
// else's program. That act has a button, on the screen where the person is
// already looking at the deck; a tool doing the same thing cost 139 tokens in
// every request of the busiest desk to save one click, and was never once
// called.
//
// The pixels come from the screen (Screen.RenderDeck): a deck is a file on
// the engine's host, and what it looks like is a question only a browser can
// answer.
func (a *Engine) DeckExportFiles(relPath, format string) (DeckExport, error) {
	root := strings.TrimSpace(a.cur().cfg.SandboxRoot)
	if root == "" {
		return DeckExport{}, fmt.Errorf("no project open")
	}
	ext, ready := writableDeckFormat(format)
	if !ready {
		// The same sentence for "no such format" and "listed but not ready
		// yet", because from here they are the same fact: nothing in this
		// binary writes it. The menu is what tells them apart, before the click.
		return DeckExport{}, fmt.Errorf("ยังส่งออกเป็น %s ไม่ได้", format)
	}
	full, err := safeSandboxPath(root, relPath)
	if err != nil {
		return DeckExport{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return DeckExport{}, ErrFileGone
	}
	if info.Size() > maxDeckBytes {
		return DeckExport{}, fmt.Errorf("ไฟล์นี้ใหญ่เกินกว่าจะส่งออก")
	}
	source, err := os.ReadFile(full)
	if err != nil {
		return DeckExport{}, err
	}

	// Under the deck's own name. The deck itself stays where it is.
	base := strings.TrimSuffix(filepath.Base(full), filepath.Ext(full))
	if strings.EqualFold(strings.TrimSpace(format), "pptx-img") {
		// Both pptx rows share an extension, so the picture one takes a suffix.
		// Without it the second export silently replaces the first, and the two
		// are not interchangeable: one is editable and plain, the other is exact
		// and frozen.
		base += "-img"
	}
	out := DeckExport{Base: base, Ext: ext}
	one := func(data []byte) DeckExport {
		out.Files = []ExportFile{{Name: base + ext, Data: data}}
		return out
	}
	fileURL := FileURLForPath(full)
	screen := a.screenOf()

	// The two formats are two different acts, and only one of them reads the
	// deck's structure. `.pptx` reduces the HTML to slides and rebuilds it in
	// OOXML, which is why it is editable and plainer than the original. `.pdf`
	// hands the untouched file to the engine that renders it on screen, which is
	// why it looks exactly like the deck and cannot be edited. Neither is the
	// better one; they answer different questions about the same deck.
	switch id := strings.ToLower(strings.TrimSpace(format)); id {
	case "pdf":
		rendered, err := screen.RenderDeck(context.Background(), fileURL, DeckRender{Kind: "pdf"})
		if err != nil {
			return DeckExport{}, err
		}
		return one(rendered.PDF), nil
	case "png", "jpg", "webp":
		// Pictures are one file per slide, so they land in a folder of their
		// own rather than scattering eight siblings beside the deck.
		rendered, err := screen.RenderDeck(context.Background(), fileURL, DeckRender{Kind: "images", Format: id})
		if err != nil {
			return DeckExport{}, err
		}
		out.Folder = true
		for i, shot := range rendered.Images {
			out.Files = append(out.Files, ExportFile{Name: fmt.Sprintf("%02d%s", i+1, ext), Data: shot})
		}
		return out, nil
	case "pptx-img":
		// The same pictures the .png export writes, one per slide, each covering
		// its whole slide. It is a .pptx that cannot be edited a word of — and
		// that is the trade, not a shortcoming: what it buys is a deck that
		// looks in PowerPoint exactly like the deck looks here, which the
		// editable one never can without pptx.go learning to lay out a slide.
		rendered, err := screen.RenderDeck(context.Background(), fileURL, DeckRender{Kind: "images", Format: "png"})
		if err != nil {
			return DeckExport{}, err
		}
		shots := rendered.Images
		slides, err := deck.Slides(source, filepath.Dir(full), root)
		if err != nil {
			return DeckExport{}, err
		}
		if len(slides) != len(shots) {
			// The reducer and the renderer both cut on section.slide, so this
			// cannot happen from a well-formed deck — and if it ever does, a
			// deck with the wrong notes on the wrong slide is worse than none.
			return DeckExport{}, fmt.Errorf("จำนวนสไลด์ไม่ตรงกัน (%d ภาพ, %d สไลด์)", len(shots), len(slides))
		}
		picture := make([]ooxml.Slide, len(shots))
		for i, shot := range shots {
			cfg, _, err := image.DecodeConfig(bytes.NewReader(shot))
			if err != nil {
				return DeckExport{}, err
			}
			picture[i] = ooxml.Slide{
				FullBleed: true,
				// Notes survive: they never render, so a picture deck keeps the
				// presenter's script instead of trading it away for fidelity.
				Notes: slides[i].Notes,
				Image: &ooxml.Picture{Ext: "png", Data: shot, WidthPx: cfg.Width, HeightPx: cfg.Height, AltText: slides[i].Title},
			}
		}
		parts, err := ooxml.BuildPPTX(picture)
		if err != nil {
			return DeckExport{}, err
		}
		data, err := ooxml.WritePackage(parts)
		if err != nil {
			return DeckExport{}, err
		}
		return one(data), nil
	default:
		// deck.Slides is authoritative about what a slide is; the frontend's
		// own check is a routing hint (see internal/deck/deck.go). So a file
		// that reached the room and still has nothing to export says so here,
		// naming what a slide is, rather than writing an empty deck.
		slides, err := deck.Slides(source, filepath.Dir(full), root)
		if err != nil {
			return DeckExport{}, err
		}
		parts, err := ooxml.BuildPPTX(slides)
		if err != nil {
			return DeckExport{}, err
		}
		data, err := ooxml.WritePackage(parts)
		if err != nil {
			return DeckExport{}, err
		}
		return one(data), nil
	}
}

// SanitiseFileName keeps a deck's own name while dropping what Windows will not
// accept in one. A deck may be called anything; a file may not. Exported for
// the screen, which names the file it writes into Downloads.
func SanitiseFileName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 0x20 {
			return '-'
		}
		return r
	}, name)
	cleaned = strings.Trim(strings.TrimSpace(cleaned), ".")
	if cleaned == "" {
		return "deck"
	}
	return cleaned
}
