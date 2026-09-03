package main

// The one way synthesized speech reaches the webview.
//
// This is filehost.go's argument applied to the ฟัง button. A data: URL costs
// about 4/3 its file size as a string that then lives in the DOM, and SAPI
// writes uncompressed 16-bit WAV — a reply that takes five minutes to read
// aloud is roughly 9MB of audio and 13MB of string, built by copying the file
// into memory, then copying that into base64, then handing the result across
// the Wails binding. A URL has none of that shape: the audio element fetches
// it, holds the part it is playing, and drops the rest.
//
// The sandbox check that filehost.go needs has no counterpart here, and not
// because it was skipped. These files are not the user's; they were written by
// the synthesizer moments ago, and speechJob.files already records exactly
// which path belongs to which piece of which job. So the URL carries a job id
// and a piece number, both looked up in that map — a request cannot name a
// path at all, which is a stronger guarantee than validating one.

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ttsHostPrefix is the URL space this owns.
const ttsHostPrefix = "/aetox-tts/"

// assetMiddleware chains the two URL spaces the app claims in front of its own
// embedded assets. Order does not matter — the prefixes are disjoint — but the
// early `next` in each does: anything addressed to neither must leave
// untouched, or the app's own HTML stops loading and the window comes up blank.
func (a *App) assetMiddleware(next http.Handler) http.Handler {
	return a.fileHost(a.ttsHost(next))
}

// ttsHost serves one piece of one read: /aetox-tts/<job>/<seq>.<ext>
//
// The extension is decoration for anything that sniffs by name; the type comes
// from the engine that wrote the file, so a cloud vendor's MP3 is never served
// as the WAV the local engines produce.
func (a *App) ttsHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, ttsHostPrefix) {
			next.ServeHTTP(w, r)
			return
		}
		jobID, piece, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, ttsHostPrefix), "/")
		if !ok || jobID == "" {
			http.NotFound(w, r)
			return
		}
		seq, err := strconv.Atoi(strings.TrimSuffix(piece, filepath.Ext(piece)))
		if err != nil || seq < 0 {
			http.NotFound(w, r)
			return
		}
		path, mime, ok := a.speechChunkFile(jobID, seq)
		if !ok {
			// A stopped read deletes its files and forgets its pieces, so a
			// late request for one is ordinary, not an error worth a page.
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(path)
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
		if mime != "" {
			w.Header().Set("Content-Type", mime)
		}
		// A piece URL is unique to one read and dies with it; caching it would
		// only ever keep bytes alive past the file they describe.
		w.Header().Set("Cache-Control", "no-store")
		// ServeContent for the same reason the file pane wants it: it answers
		// Range requests, which is what lets the audio element start playing
		// before it holds the whole piece.
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
}
