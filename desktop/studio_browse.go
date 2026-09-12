package main

// Browsing the studio's shelf from the Settings page: a paged, filtered view
// of the catalogue, and the URL space that lets the webview play a sound or
// show a clip straight from wherever the user keeps it.
//
// The page sees exactly what the agent sees. StudioAssets runs the same
// assetlib.Search that `asset_find query` runs, over the same shelves, so a
// person can type "whoosh" here and watch the rows the agent would get — the
// one honest way to judge whether the catalogue understood their folder.
//
// The shelf host is a third URL space beside the project's (/aetox-file/) and
// the reader's (/aetox-tts/). It is narrower than either: it serves by asset
// id, not by path, and an id resolves only through the catalogue. A request
// can therefore reach precisely the files a scan recorded and nothing beside
// them — no path arrives from the webview at all, so there is nothing for a
// traversal to traverse. That is what makes it acceptable to serve from
// outside the project sandbox, which the file host never does.

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mikedev115/Aetox/internal/assetlib"
)

// studioPerPage is how many rows one page of the browser holds. Sixty is
// what a grid of thumbnails shows without the eye giving up, and what sixty
// <video preload=metadata> tags cost the local file host without a stutter.
const studioPerPage = 60

// StudioAssetQuery is the page's question.
type StudioAssetQuery struct {
	Text      string `json:"text"`
	Kind      string `json:"kind"`
	Category  string `json:"category"`
	Library   string `json:"library"`
	AlphaOnly bool   `json:"alphaOnly"`
	// IncludeHidden shows rows the user hid, marked, so one can be un-hidden.
	IncludeHidden bool `json:"includeHidden"`
	// Page is 1-based.
	Page int `json:"page"`
}

// StudioAssetView is one row as the page draws it.
type StudioAssetView struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Kind     string  `json:"kind"`
	Category string  `json:"category"`
	Duration float64 `json:"duration"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Alpha    bool    `json:"alpha"`
	Bytes    int64   `json:"bytes"`
	// Hidden is the user's own doing (StudioSetHidden); only present when the
	// query asked for hidden rows.
	Hidden bool `json:"hidden"`
	// Ext is the lowercase extension without the dot, so the page can pick
	// <audio>, <video> or <img> without re-parsing the name.
	Ext string `json:"ext"`
	// URL is where the webview can fetch the bytes (studioHost).
	URL string `json:"url"`
	// Thumb is the poster's URL when one has been rendered, else "" — the
	// page then asks StudioThumbs, which renders it in the background.
	Thumb string `json:"thumb"`
	// Playable is whether the webview itself can decode this file for a
	// hover preview. False for ProRes .mov and the like, where the poster is
	// all a tile will show — and all it needs to.
	Playable bool `json:"playable"`
	// Library is the shelf's name, for the small line under a row when more
	// than one shelf is on the page.
	Library   string `json:"library"`
	LibraryID string `json:"libraryId"`
}

// StudioAssetPage is one page of answers.
type StudioAssetPage struct {
	Rows  []StudioAssetView `json:"rows"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Pages int               `json:"pages"`
	// Categories is every first-level folder across the shelves searched, with
	// counts, for the filter — answered here so the page needs one call.
	Categories []StudioCategory `json:"categories"`
}

// StudioCategory is one of the user's own folder names, with how many rows
// it holds.
type StudioCategory struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// StudioAssets answers one page of the browser. Never nil rows (§34).
func (a *App) StudioAssets(q StudioAssetQuery) StudioAssetPage {
	out := StudioAssetPage{Rows: []StudioAssetView{}, Categories: []StudioCategory{}, Page: 1, Pages: 1}
	libs, _, root, err := studioShelvesWith(q.IncludeHidden)
	if err != nil {
		return out
	}
	if lib := strings.TrimSpace(q.Library); lib != "" {
		var only []*assetlib.Library
		for _, l := range libs {
			if l.ID == lib {
				only = append(only, l)
			}
		}
		libs = only
	}

	// Categories are counted over the shelves in view, before the kind and
	// text filters, so the dropdown keeps offering folders a narrower query
	// has emptied — otherwise picking one filter hides the others.
	cats := map[string]int{}
	for _, l := range libs {
		for _, as := range l.Assets {
			if as.Category != "" {
				cats[as.Category]++
			}
		}
	}
	for name, n := range cats {
		out.Categories = append(out.Categories, StudioCategory{Name: name, Count: n})
	}
	sortCategories(out.Categories)

	// The whole match, then the page: Search caps by Limit, so ask for all
	// and cut here. Catalogues are thousands of rows, not millions.
	all, _ := assetlib.Search(libs, assetlib.Query{
		Text:      q.Text,
		Kind:      assetlib.Kind(strings.ToLower(strings.TrimSpace(q.Kind))),
		NeedAlpha: q.AlphaOnly,
		Limit:     1 << 20,
	})
	if cat := strings.TrimSpace(q.Category); cat != "" {
		kept := all[:0]
		for _, as := range all {
			if as.Category == cat {
				kept = append(kept, as)
			}
		}
		all = kept
	}
	out.Total = len(all)
	out.Pages = (len(all) + studioPerPage - 1) / studioPerPage
	if out.Pages == 0 {
		out.Pages = 1
	}
	out.Page = q.Page
	if out.Page < 1 {
		out.Page = 1
	}
	if out.Page > out.Pages {
		out.Page = out.Pages
	}
	start := (out.Page - 1) * studioPerPage
	end := start + studioPerPage
	if end > len(all) {
		end = len(all)
	}
	// Which shelf a row came from is not on the Asset — Search flattens the
	// shelves — so it is looked up again by id, which Find does in one pass.
	for _, as := range all[start:end] {
		lib, _, ok := assetlib.Find(libs, as.ID)
		if !ok {
			continue
		}
		out.Rows = append(out.Rows, studioAssetView(lib, as, root))
	}
	return out
}

// webviewPlays is what WebView2 (Chromium) decodes on its own. Everything
// else gets a poster and no live preview.
var webviewPlays = map[string]bool{"mp4": true, "webm": true, "m4v": true, "gif": true, "png": true, "jpg": true, "jpeg": true, "webp": true, "svg": true, "mp3": true, "wav": true, "ogg": true, "m4a": true, "aac": true, "flac": true}

func studioAssetView(lib *assetlib.Library, as assetlib.Asset, dataRoot string) StudioAssetView {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(as.Path)), ".")
	thumb := ""
	if assetlib.HasPoster(as) {
		if _, err := os.Stat(assetlib.ThumbPath(dataRoot, as)); err == nil {
			thumb = studioThumbURL(as)
		}
	}
	return StudioAssetView{
		Thumb:     thumb,
		Playable:  webviewPlays[ext],
		ID:        as.ID,
		Name:      as.Name(),
		Path:      as.Path,
		Kind:      string(as.Kind),
		Category:  as.Category,
		Duration:  as.Duration,
		Width:     as.Width,
		Height:    as.Height,
		Alpha:     as.HasAlpha,
		Bytes:     as.Bytes,
		Hidden:    as.Hidden,
		Ext:       ext,
		URL:       studioHostPrefix + lib.ID + "/" + as.ID + strings.ToLower(filepath.Ext(as.Path)),
		Library:   lib.Name(),
		LibraryID: lib.ID,
	}
}

func sortCategories(cs []StudioCategory) {
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0 && strings.ToLower(cs[j].Name) < strings.ToLower(cs[j-1].Name); j-- {
			cs[j], cs[j-1] = cs[j-1], cs[j]
		}
	}
}

// ---------------------------------------------------------------- the host

// studioHostPrefix is the URL space this owns: /aetox-shelf/<library>/<asset>.<ext>
//
// The extension is decoration for the webview's benefit — a <video> element
// and the content-type table both read it — and is ignored on the way in: the
// asset id alone decides which file answers.
const studioHostPrefix = "/aetox-shelf/"

// studioHost serves one catalogued file per request, by id.
func (a *App) studioHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, studioHostPrefix) {
			next.ServeHTTP(w, r)
			return
		}
		if serveStudioThumb(w, r) {
			return
		}
		rest := strings.TrimPrefix(r.URL.Path, studioHostPrefix)
		libID, assetID, ok := strings.Cut(rest, "/")
		if !ok || libID == "" || assetID == "" {
			http.NotFound(w, r)
			return
		}
		assetID = strings.TrimSuffix(assetID, filepath.Ext(assetID))

		libs, _, _, err := studioShelves()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var full string
		for _, lib := range libs {
			if lib.ID != libID {
				continue
			}
			if _, as, found := assetlib.Find([]*assetlib.Library{lib}, assetID); found {
				full = lib.FullPath(*as)
			}
			break
		}
		if full == "" {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		if ct := contentTypes[strings.ToLower(filepath.Ext(full))]; ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		// Unlike a produced file, a shelf file does not change under its URL,
		// so the webview may keep it: paging back to a grid should not
		// re-fetch sixty first frames.
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
}
