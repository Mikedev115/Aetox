package main

// Posters for the shelf browser, rendered in the background and announced
// when ready — the "render" third of display / render / load
// (internal/assetlib/thumbs.go has the whole argument).
//
// The page asks once per page which posters exist (StudioThumbs). Those that
// do come back at once as URLs; those that do not are queued, and one worker
// renders them a batch at a time and emits "studio:thumbs" with the ids it
// finished, so tiles fill in as the frames land rather than the page waiting
// for all sixty. A poster rendered once is a file on disk forever after, so
// the second visit to a page costs nothing.

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Mikedev115/Aetox/internal/assetlib"
)

// studioThumbPrefix is the URL space for posters: /aetox-shelf/thumb/<id>.<jpg|png>
// Inside the shelf host's prefix so the middleware chain needs no new entry;
// "thumb" cannot collide with a library id, which is eight hex digits or
// "builtin-…".
const studioThumbPrefix = studioHostPrefix + "thumb/"

var thumbName = regexp.MustCompile(`^([0-9a-f]{12})\.(jpg|png)$`)

// studioThumbs is the render queue: one worker, a set of ids waiting, and the
// ids in flight so a page re-asking mid-render does not queue them twice.
type studioThumbs struct {
	mu      sync.Mutex
	pending map[string]bool
	running bool
}

// studioThumbURL is a poster's address, given its asset.
func studioThumbURL(a assetlib.Asset) string {
	return studioThumbPrefix + filepath.Base(assetlib.ThumbPath("", a))
}

// StudioThumbs answers which of the ids have a poster now, as id → URL, and
// queues the rest for rendering. Never nil (§34).
func (a *App) StudioThumbs(ids []string) map[string]string {
	out := map[string]string{}
	libs, _, root, err := studioShelves()
	if err != nil {
		return out
	}
	th := assetlib.Thumber{FFmpeg: findProgram("ffmpeg"), DataRoot: root}
	for _, id := range ids {
		_, as, ok := assetlib.Find(libs, id)
		if !ok || !assetlib.HasPoster(*as) {
			continue
		}
		if _, err := os.Stat(assetlib.ThumbPath(root, *as)); err == nil {
			out[id] = studioThumbURL(*as)
		}
	}
	missing := th.Missing(libs, ids)
	if len(missing) == 0 || th.FFmpeg == "" {
		return out
	}
	a.thumbs.mu.Lock()
	if a.thumbs.pending == nil {
		a.thumbs.pending = map[string]bool{}
	}
	for _, as := range missing {
		a.thumbs.pending[as.ID] = true
	}
	start := !a.thumbs.running
	a.thumbs.running = true
	a.thumbs.mu.Unlock()
	if start {
		go a.runThumbWorker(th)
	}
	return out
}

// runThumbWorker drains the queue in batches until it is empty. The shelves
// are re-read per round so a rescan mid-render is seen, not raced.
func (a *App) runThumbWorker(th assetlib.Thumber) {
	for {
		a.thumbs.mu.Lock()
		if len(a.thumbs.pending) == 0 {
			a.thumbs.running = false
			a.thumbs.mu.Unlock()
			return
		}
		ids := make([]string, 0, 16)
		for id := range a.thumbs.pending {
			ids = append(ids, id)
			delete(a.thumbs.pending, id)
			if len(ids) == 16 {
				break
			}
		}
		a.thumbs.mu.Unlock()

		libs, _, _, err := studioShelves()
		if err != nil {
			continue
		}
		done := th.Render(context.Background(), libs, th.Missing(libs, ids))
		urls := map[string]string{}
		for _, id := range done {
			if _, as, ok := assetlib.Find(libs, id); ok {
				urls[id] = studioThumbURL(*as)
			}
		}
		// Ids that failed to render are reported too, with no URL, so a tile
		// stops waiting and shows its kind's icon instead of a placeholder
		// that never resolves.
		for _, id := range ids {
			if _, ok := urls[id]; !ok {
				urls[id] = ""
			}
		}
		a.emitEvent("studio:thumbs", urls)
	}
}

// serveStudioThumb answers a poster request, or reports false for a path
// that is not one so the shelf host continues with its own lookup.
func serveStudioThumb(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, studioThumbPrefix) {
		return false
	}
	name := strings.TrimPrefix(r.URL.Path, studioThumbPrefix)
	if !thumbName.MatchString(name) {
		http.NotFound(w, r)
		return true
	}
	root, err := studioDataRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return true
	}
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(assetlib.ThumbDir), name))
	if err != nil {
		http.NotFound(w, r)
		return true
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return true
	}
	w.Header().Set("Content-Type", contentTypes[filepath.Ext(name)])
	// A poster is content-addressed by the asset id and rendered once; the
	// webview may keep it for the session.
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, name, info.ModTime(), f)
	return true
}
