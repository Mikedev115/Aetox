package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// serve runs one request through the file host, with `next` recording whether
// the request was passed through instead of claimed.
func serve(t *testing.T, root, target string) (*http.Response, string, bool) {
	t.Helper()
	passed := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { passed = true })
	a := &App{}
	a.cur().cfg.SandboxRoot = root

	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	a.fileHost(next).ServeHTTP(rec, req)
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	return res, string(body), passed
}

// The rule the whole app depends on: anything not addressed to the prefix must
// leave untouched. Claiming it would take the app's own HTML with it and the
// window would come up blank.
func TestFileHostPassesThroughForeignPaths(t *testing.T) {
	for _, path := range []string{"/", "/index.html", "/assets/app.js", "/aetox-files/x"} {
		_, _, passed := serve(t, t.TempDir(), path)
		if !passed {
			t.Errorf("%s: claimed by the file host, want pass-through", path)
		}
	}
}

func TestFileHostServesProjectFile(t *testing.T) {
	root := t.TempDir()
	want := "<svg xmlns=\"http://www.w3.org/2000/svg\"/>"
	if err := os.WriteFile(filepath.Join(root, "mark.svg"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	res, body, passed := serve(t, root, "/aetox-file/mark.svg")
	if passed {
		t.Fatal("passed through, want served")
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
	// Pinned rather than sniffed: on Windows mime.TypeByExtension asks the
	// registry, where another program's install can leave .svg answering
	// something that turns the pane into a download prompt.
	if ct := res.Header.Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
	// Scrubbing a video without downloading it first is the whole point of
	// serving over HTTP rather than as a data: URL.
	if ar := res.Header.Get("Accept-Ranges"); ar != "bytes" {
		t.Errorf("Accept-Ranges = %q, want bytes", ar)
	}
}

// A file tab re-reads on every open because the agent rewrites the same path
// constantly. A cached response would hand back the previous turn's bytes under
// the right filename — the one failure the pane exists to prevent.
func TestFileHostDoesNotCache(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, _ := serve(t, root, "/aetox-file/a.png")
	if cc := res.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}

func TestFileHostServesRange(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "clip.mp4"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	a.cur().cfg.SandboxRoot = root
	req := httptest.NewRequest(http.MethodGet, "/aetox-file/clip.mp4", nil)
	req.Header.Set("Range", "bytes=4-6")
	rec := httptest.NewRecorder()
	a.fileHost(http.NotFoundHandler()).ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != "456" {
		t.Errorf("body = %q, want %q", body, "456")
	}
}

// This transport adds a way to reach files, not a right to reach them. Every
// request resolves through the same safeSandboxPath as every binding.
func TestFileHostRefusesOutsideProject(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "secret.txt")
	if err := os.WriteFile(outside, []byte("private"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(outside) })

	for _, target := range []string{
		"/aetox-file/../secret.txt",
		"/aetox-file/sub/../../secret.txt",
	} {
		res, body, _ := serve(t, root, target)
		if res.StatusCode == http.StatusOK {
			t.Errorf("%s: status 200, want refusal", target)
		}
		if strings.Contains(body, "private") {
			t.Errorf("%s: served a file outside the project", target)
		}
	}
}

// A path already decoded by net/http must not be decoded a second time, or
// %252e%252e arrives as ".." after the guard has already looked at it.
func TestFileHostDoesNotDoubleDecode(t *testing.T) {
	root := t.TempDir()
	res, body, _ := serve(t, root, "/aetox-file/%252e%252e/secret.txt")
	if res.StatusCode == http.StatusOK {
		t.Errorf("status 200, want refusal; body %q", body)
	}
}

func TestFileHostWithoutProject(t *testing.T) {
	res, _, passed := serve(t, "", "/aetox-file/a.png")
	if passed {
		t.Fatal("passed through, want claimed")
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}

func TestFileHostRefusesDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, _, _ := serve(t, root, "/aetox-file/docs")
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}

// A produced file has two names: the one the model asked for and the one
// placedWrite actually gave it. An <img> in an answer carries the first —
// ![cat](cat.jpg) is written out of the same intention that made the call, not
// out of the receipt that says where the file landed — so every picture a chat
// produced used to 404 here (owner, 7 ก.ย., first two image_make calls).
//
// The same fallback the file TOOLS already use (skill.PlacedPath), one layer up.
func TestFileHostFindsAFileInTheSessionsOwnFolder(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "output", "sess-1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "cat.jpg"), []byte("\xff\xd8\xffbytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	passed := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { passed = true })
	a := &App{}
	a.cur().cfg.SandboxRoot = root
	a.cur().id = "sess-1"

	rec := httptest.NewRecorder()
	a.fileHost(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/aetox-file/cat.jpg", nil))
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	if passed {
		t.Fatal("the request was passed through instead of served")
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("bare name = %d, want 200 — the session fallback did not run", res.StatusCode)
	}
	if !strings.HasPrefix(string(body), "\xff\xd8\xff") {
		t.Errorf("served the wrong bytes: %q", body)
	}
}

// The literal path still wins. A file that really is at the root must not be
// shadowed by one of the same name in a session folder.
func TestFileHostPrefersTheLiteralPath(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "output", "sess-1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cat.jpg"), []byte("at-the-root"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "cat.jpg"), []byte("in-the-folder"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	a.cur().cfg.SandboxRoot = root
	a.cur().id = "sess-1"
	rec := httptest.NewRecorder()
	a.fileHost(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/aetox-file/cat.jpg", nil))
	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) != "at-the-root" {
		t.Errorf("served %q, want the literal path to win", body)
	}
}

// image_make names the file after the bytes, so a model that asked for .png
// and got JPEG has a correct file on disk and a stale name in its own answer.
// The app made that mismatch; the app absorbs it.
func TestFileHostServesAPictureWhoseExtensionWasCorrected(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "output", "sess-1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "hero.jpg"), []byte("real-jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	a.cur().cfg.SandboxRoot = root
	a.cur().id = "sess-1"
	rec := httptest.NewRecorder()
	a.fileHost(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/aetox-file/hero.png", nil))
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("hero.png = %d, want 200 via the .jpg beside it", res.StatusCode)
	}
	if string(body) != "real-jpeg" {
		t.Errorf("served %q", body)
	}
}

// The exact name always wins, and the rule never reaches past pictures.
func TestExtensionFallbackIsNarrow(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("both.png", "the-png")
	write("both.jpg", "the-jpg")
	write("notes.md", "a document")

	a := &App{}
	a.cur().cfg.SandboxRoot = root
	get := func(path string) (int, string) {
		rec := httptest.NewRecorder()
		a.fileHost(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/aetox-file/"+path, nil))
		res := rec.Result()
		body, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(body)
	}

	// A real file is never shadowed by its sibling.
	if _, body := get("both.png"); body != "the-png" {
		t.Errorf("both.png served %q, want the exact file", body)
	}
	// A missing document does not become some other document.
	if code, _ := get("notes.txt"); code != http.StatusNotFound {
		t.Errorf("notes.txt = %d, want 404 — the rule is for pictures only", code)
	}
	// And a picture with no sibling at all is still a 404.
	if code, _ := get("nothing.png"); code != http.StatusNotFound {
		t.Errorf("nothing.png = %d, want 404", code)
	}
}
