package imagegen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogPutsTheKeylessRowFirst(t *testing.T) {
	got := Catalog()
	if len(got) == 0 {
		t.Fatal("catalog is empty")
	}
	if !got[0].Default {
		t.Fatalf("first row %q is not the default", got[0].ID)
	}
	// The default has to be the one that works on a machine with nothing set
	// up, or the first picture a user asks for fails on an error about a key.
	if len(got[0].Binaries) != 0 || len(got[0].InstallCommand) != 0 {
		t.Errorf("default row %q asks for an install; it must not", got[0].ID)
	}
	if !strings.Contains(got[0].Install, "เซิร์ฟเวอร์") {
		t.Errorf("cloud row %q does not tell the user the prompt leaves the machine: %q", got[0].ID, got[0].Install)
	}
}

func TestLookupEmptyIsTheDefaultAndUnknownIsRefused(t *testing.T) {
	d, ok := Lookup("")
	if !ok || !d.Default {
		t.Fatalf("empty id did not resolve to the default: %+v ok=%v", d, ok)
	}
	if _, ok := Lookup("midjourney"); ok {
		t.Error("an unknown vendor resolved")
	}
	if _, err := New(Options{Engine: "midjourney"}); err == nil {
		t.Error("New accepted an unknown vendor")
	}
}

func TestPinnedModelResolution(t *testing.T) {
	desc := Descriptor{Label: "x", Models: []string{"flux", "turbo"}}

	if got, err := resolveNamedModel(desc, ""); err != nil || got != "flux" {
		t.Errorf("unpinned should take the first: got %q err=%v", got, err)
	}
	if got, err := resolveNamedModel(desc, "turbo"); err != nil || got != "turbo" {
		t.Errorf("a known pin should stand: got %q err=%v", got, err)
	}
	// A name the built-in roster has never heard of is not wrong: the picker
	// prefers the installed model catalog, which knows newer names than this
	// file does. Refusing it would mean refusing a choice the app itself drew.
	if got, err := resolveNamedModel(desc, "flux-2-pro"); err != nil || got != "flux-2-pro" {
		t.Errorf("an off-roster pin should pass through: got %q err=%v", got, err)
	}
	// A vendor with no model concept and a pin anyway is a setting that has
	// silently stopped applying, which is the one case worth shouting about.
	if _, err := resolveNamedModel(Descriptor{Label: "y"}, "anything"); err == nil {
		t.Error("a pin on a vendor with no models was accepted silently")
	}
}

func TestOnlyAnImageBodyIsAccepted(t *testing.T) {
	for _, ok := range []string{"image/jpeg", "image/png", "IMAGE/JPEG; charset=binary"} {
		if err := mustBeImage(ok); err != nil {
			t.Errorf("mustBeImage(%q) rejected a picture: %v", ok, err)
		}
	}
	// The case this exists for: a free endpoint's rate-limit notice, served
	// 200 with an HTML body, written to disk under .jpg and opening nowhere.
	for _, bad := range []string{"text/html", "application/json", ""} {
		if err := mustBeImage(bad); err == nil {
			t.Errorf("mustBeImage(%q) accepted a non-picture", bad)
		}
	}
}

// serve stands in for the endpoint. It records the request it was given so the
// test can assert on the URL that was actually built.
func serve(t *testing.T, status int, contentType, body string) (*pollinations, *url.URL) {
	t.Helper()
	var seen *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		seen = &u
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &pollinations{base: srv.URL + "/prompt/", model: "flux", client: srv.Client()}, seen
}

func TestGenerateWritesThePictureAndAsksForWhatItWasTold(t *testing.T) {
	eng, _ := serve(t, http.StatusOK, "image/jpeg", "\xff\xd8\xffnot-really-a-jpeg")
	out := filepath.Join(t.TempDir(), "cat"+eng.Ext())

	err := eng.Generate(context.Background(), "แมวส้ม นั่งบนกล่อง", Request{Width: 640, Height: 480, Seed: 7}, out)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("nothing was written: %v", err)
	}
	if !strings.HasPrefix(string(got), "\xff\xd8\xff") {
		t.Errorf("the body did not land intact: %q", got)
	}

	// Re-run against a recorder we can read after the call.
	var seen *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := *r.URL
		seen = &u
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("\xff\xd8\xff"))
	}))
	defer srv.Close()
	eng2 := &pollinations{base: srv.URL + "/prompt/", model: "turbo", client: srv.Client()}
	if err := eng2.Generate(context.Background(), "a b", Request{Width: 640, Height: 480, Seed: 7}, filepath.Join(t.TempDir(), "x.jpg")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if seen == nil {
		t.Fatal("the endpoint was never called")
	}
	q := seen.Query()
	for k, want := range map[string]string{"model": "turbo", "width": "640", "height": "480", "seed": "7", "nologo": "true"} {
		if q.Get(k) != want {
			t.Errorf("query %s = %q, want %q (full: %s)", k, q.Get(k), want, seen.RawQuery)
		}
	}
	// PathEscape, not QueryEscape: a "+" here would be a literal plus in the
	// picture, not the space the user typed.
	if strings.Contains(seen.Path, "+") {
		t.Errorf("a space was escaped as + in the path: %q", seen.Path)
	}
	if seen.Path != "/prompt/a b" {
		t.Errorf("prompt path = %q, want %q", seen.Path, "/prompt/a b")
	}
}

func TestGenerateLeavesNoFileWhenTheAnswerIsNotAPicture(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "nope.jpg")

	// 200 with an HTML body — the failure mode a keyless endpoint actually has.
	eng, _ := serve(t, http.StatusOK, "text/html", "<html>rate limited</html>")
	if err := eng.Generate(context.Background(), "x", Request{}, out); err == nil {
		t.Fatal("an HTML body was accepted as a picture")
	}
	assertEmpty(t, dir)

	// And an ordinary failure status.
	eng2, _ := serve(t, http.StatusTooManyRequests, "application/json", `{"error":"slow down"}`)
	if err := eng2.Generate(context.Background(), "x", Request{}, out); err == nil {
		t.Fatal("a 429 was accepted")
	} else if !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "slow down") {
		t.Errorf("the vendor's own words did not reach the user: %v", err)
	}
	assertEmpty(t, dir)
}

func TestAnEmptyPromptIsRefusedBeforeTheNetwork(t *testing.T) {
	eng := &pollinations{base: "http://127.0.0.1:0/prompt/", client: http.DefaultClient}
	if err := eng.Generate(context.Background(), "   ", Request{}, filepath.Join(t.TempDir(), "x.jpg")); err == nil {
		t.Fatal("an empty prompt reached the network")
	}
}

// assertEmpty is the half-written-file guard: a failed Generate must leave the
// directory as it found it, temp file included.
func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("a failed Generate left files behind: %v", names)
	}
}

func TestDirOfKeepsTheTempBesideItsDestination(t *testing.T) {
	// Both separators, because this runs on Windows and the sandbox hands
	// paths through in whichever shape the caller had.
	for path, want := range map[string]string{
		`D:\a\b\c.jpg`: `D:\a\b`,
		"/a/b/c.jpg":   "/a/b",
		"c.jpg":        ".",
	} {
		if got := dirOf(path); got != want {
			t.Errorf("dirOf(%q) = %q, want %q", path, got, want)
		}
	}
}
