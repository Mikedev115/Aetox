package main

// The one way bytes from the open project reach the webview.
//
// Everything the file pane could show until now had to be carried across the
// Wails binding as a JavaScript value: text as a string, a workbook as JSON, an
// image as a base64 data: URL. That works while the file is small and it stops
// working the moment it is not — a data: URL costs about 4/3 its file size as a
// string that then lives in the DOM, which is why the image pane needed a 20MB
// ceiling and why a video could never be shown at all.
//
// A URL has none of that shape. The webview streams it the way a browser
// streams any other URL: only the bytes on screen are in memory, and the ones
// past them are fetched when asked for. That is what makes a 4GB video no
// harder than a 4MB one, and it is why this file exists rather than a
// ReadVideoDataURL next to ReadImageDataURL.
//
// The sandbox is not relaxed to get it. Every request resolves through the same
// safeSandboxPath every binding uses, so this adds a transport, not a right.

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// pictureExts are the extensions siblingByExtension will accept as the same
// picture under another name. Deliberately only pictures: this rule exists
// because image_make renames what it wrote, and nothing else in the app hands
// back a file under a different extension than the caller asked for.
var pictureExts = []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}

// isServableFile is "a file that exists and is not a directory". Named apart
// from the test helper of a similar name in tool_coverage_test.go, which is a
// closure factory rather than a predicate.
func isServableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// siblingByExtension answers the mismatch the app creates itself.
//
// image_make names the file after the BYTES the engine produced, not after
// what the caller asked for — a model writes `hero.png` because pictures are
// called that, and Pollinations answers with JPEG, so the file on disk is
// `hero.jpg`. That rule is right and stays: a .png that is really a JPEG opens
// nowhere. But the model then writes ![hero](hero.png) into its answer, naming
// the file it requested rather than the file it got, and the picture 404s
// (owner, 7 ก.ย., third image_make call — the .jpg answered 200 beside it).
//
// So when a picture is asked for by a name that does not exist, the same
// basename wearing another picture extension in the same folder is served
// instead. It only ever runs when the exact path missed, so a real hero.png
// sitting beside a real hero.jpg is never shadowed — and it will not turn a
// missing document into some other document, because both ends are pictures.
func siblingByExtension(full string) string {
	if _, err := os.Stat(full); err == nil {
		return full
	}
	ext := strings.ToLower(filepath.Ext(full))
	if !slices.Contains(pictureExts, ext) {
		return full
	}
	stem := strings.TrimSuffix(full, filepath.Ext(full))
	for _, candidate := range pictureExts {
		if candidate == ext {
			continue
		}
		alt := stem + candidate
		if info, err := os.Stat(alt); err == nil && !info.IsDir() {
			return alt
		}
	}
	return full
}

// fileHostPrefix is the URL space this owns. Everything outside it belongs to
// the app's own assets and is passed straight through.
const fileHostPrefix = "/aetox-file/"

// contentTypes pins the types the panes depend on rather than letting
// mime.TypeByExtension answer — which on Windows means asking the registry,
// where another program's install can have claimed .svg or .pdf and left a type
// behind that turns the pane into a download prompt. The few extensions that
// have a pane are worth being certain about; anything else falls through to
// http.ServeContent's own sniffing, which is correct for the general case.
var contentTypes = map[string]string{
	".pdf":  "application/pdf",
	".svg":  "image/svg+xml",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
	".bmp":  "image/bmp",
	".ico":  "image/x-icon",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mov":  "video/quicktime",
	".mkv":  "video/x-matroska",
	".avi":  "video/x-msvideo",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".m4a":  "audio/mp4",
	".flac": "audio/flac",
	".ogg":  "audio/ogg",
}

// fileHost serves files from the open project to the webview.
//
// Written as Wails middleware rather than an assetserver Handler because the
// Handler is only reached for requests the embedded assets could not answer,
// and "could not answer" is a fallback path shared with genuine 404s. A
// middleware sees every request first and can claim its own prefix outright —
// which also means the one rule that matters here is the early `next`: anything
// not addressed to this prefix must leave untouched, or the app's own HTML and
// scripts stop loading and the window comes up blank.
func (a *App) fileHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, fileHostPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		// r.URL.Path is already percent-decoded by net/http. Decoding it again
		// would corrupt any real % in a filename and, worse, let %252e%252e
		// through as ".." — the escape safeSandboxPath is there to stop.
		rel := strings.TrimPrefix(r.URL.Path, fileHostPrefix)
		if rel == "" {
			http.NotFound(w, r)
			return
		}

		root := strings.TrimSpace(a.cur().cfg.SandboxRoot)
		if root == "" {
			http.Error(w, "no project open", http.StatusNotFound)
			return
		}
		// A produced file has two names: the path the model ASKED for, and the
		// path placedWrite actually gave it — a session's output/<id> folder
		// when no project is focused. An <img> in an answer carries the first,
		// because the model writes ![cat](cat.jpg) out of the same intention
		// that made the call, not out of the receipt that says where the file
		// landed. So every picture a chat produced 404'd here (owner, 7 ก.ย.,
		// with a screenshot of the second image_make call).
		//
		// The file TOOLS already answer this with skill.PlacedPath: try the
		// literal path, fall back to the session's folder, and report the
		// original when neither exists. The same rule one layer up, and it
		// grants nothing new — both candidates still go through
		// safeSandboxPath under the same root.
		// Two candidate folders, each given the extension tolerance in turn.
		// They have to compose: a picture asked for as `hero.png` that was
		// written as `output/<id>/hero.jpg` misses on BOTH counts at once, and
		// resolving the folder first would leave the extension rule searching
		// the root while the file sits in the session's folder.
		//
		// PlacedWrite rather than PlacedPath: the second candidate is wanted as
		// a place to look, and PlacedPath only reports one that already holds
		// the exact name — which is the case that has just failed.
		full, err := safeSandboxPath(root, rel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		full = siblingByExtension(full)
		if _, statErr := os.Stat(full); statErr != nil {
			if placed := skill.PlacedWrite(a.outputSubdir, rel); placed != rel {
				if alt, altErr := safeSandboxPath(root, placed); altErr == nil {
					if alt = siblingByExtension(alt); isServableFile(alt) {
						full = alt
					}
				}
			}
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
		// A file the agent is still writing must not be cached under a URL that
		// never changes; the pane re-reads on every open by design (loadFileTab)
		// and a cached response would hand back the previous turn's bytes.
		w.Header().Set("Cache-Control", "no-store")

		// ServeContent, not io.Copy: it answers Range requests, which is the
		// whole reason a video can be scrubbed without downloading it first,
		// and it sets Last-Modified and Accept-Ranges to match.
		http.ServeContent(w, r, info.Name(), info.ModTime(), f)
	})
}
