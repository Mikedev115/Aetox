package rpc

// A file from the screen's machine lands where the engine can read it, is
// named as it was, is refused without the token, and goes when the screen
// says it is done with it (upload.go).

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// engineOnTheWire is a server with no project open and its listener's
// address; the data root is the test's.
func engineOnTheWire(t *testing.T) (addr, dataRoot string) {
	t.Helper()
	dataRoot = t.TempDir()
	t.Setenv("AETOX_DATA_ROOT", dataRoot)
	srv := NewServer(testToken, engine.NewEngine)
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		engine.Shutdown(srv.Engine(), ctx)
	})
	return strings.TrimPrefix(hs.URL, "http://"), dataRoot
}

func TestAnUploadLandsInTheEnginesInboxAndGoesWhenDiscarded(t *testing.T) {
	addr, dataRoot := engineOnTheWire(t)
	endpoint := func() (string, string, string, bool) { return "tcp", addr, testToken, true }

	local := filepath.Join(t.TempDir(), "ใบเสร็จ.pdf")
	if err := os.WriteFile(local, []byte("%PDF-1.4 hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	up, err := UploadFile(context.Background(), endpoint, local)
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if filepath.Base(up.Path) != "ใบเสร็จ.pdf" {
		t.Errorf("landed as %q, want the file's own name", up.Path)
	}
	if !strings.HasPrefix(up.Path, filepath.Join(dataRoot, inboxDir)) {
		t.Errorf("landed at %q, want under the engine's inbox", up.Path)
	}
	got, err := os.ReadFile(up.Path)
	if err != nil {
		t.Fatalf("reading the landing: %v", err)
	}
	if string(got) != "%PDF-1.4 hello" {
		t.Errorf("bytes differ: %q", got)
	}

	if err := Discard(context.Background(), endpoint, up.ID); err != nil {
		t.Fatalf("Discard: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(up.Path)); !os.IsNotExist(err) {
		t.Errorf("the landing is still there after Discard: %v", err)
	}
}

func TestAnUploadIsRefusedWithoutTheTokenAndWithoutAName(t *testing.T) {
	addr, _ := engineOnTheWire(t)
	put := func(token, name string) int {
		req, _ := http.NewRequest(http.MethodPut, "http://"+addr+UploadPath+"?name="+name, strings.NewReader("x"))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := put("", "a.txt"); code != http.StatusUnauthorized {
		t.Errorf("no token got %d, want 401", code)
	}
	if code := put(testToken, ""); code != http.StatusBadRequest {
		t.Errorf("no name got %d, want 400", code)
	}
	if code := put(testToken, "..%2F..%2Fescape.txt"); code != http.StatusOK {
		t.Errorf("a name with folders in it got %d, want 200 with the folders dropped", code)
	}
}

func TestAnUploadThatEndsEarlyLeavesNothingBehind(t *testing.T) {
	addr, dataRoot := engineOnTheWire(t)
	// A body shorter than its declared length, the way a tunnel that dropped
	// mid-clip looks from the engine's side.
	req, _ := http.NewRequest(http.MethodPut, "http://"+addr+UploadPath+"?name=clip.mp4", strings.NewReader("short"))
	req.ContentLength = 1 << 20
	req.Header.Set("Authorization", "Bearer "+testToken)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	// Either the client noticed (Go's transport refuses to send a body
	// shorter than ContentLength) or the engine did; both must leave the
	// inbox empty.
	entries, _ := os.ReadDir(filepath.Join(dataRoot, inboxDir))
	if len(entries) != 0 {
		t.Errorf("%d landings left behind by a body that ended early", len(entries))
	}
}

func TestSafeUploadNameKeepsTheNameAndDropsTheFolders(t *testing.T) {
	cases := map[string]string{
		`C:\Users\me\photo.png`: "photo.png",
		"/home/me/a b.txt":      "a b.txt",
		"plain.md":              "plain.md",
		"..":                    "",
		"":                      "",
		`dir\`:                  "",
	}
	for in, want := range cases {
		if got := safeUploadName(in); got != want {
			t.Errorf("safeUploadName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTheSweepTakesOnlyOldLandings(t *testing.T) {
	dataRoot := t.TempDir()
	old := filepath.Join(dataRoot, inboxDir, "0123456789abcdef")
	fresh := filepath.Join(dataRoot, inboxDir, "fedcba9876543210")
	for _, d := range []string{old, fresh} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	stale := time.Now().Add(-2 * inboxKeep)
	if err := os.Chtimes(old, stale, stale); err != nil {
		t.Fatal(err)
	}
	sweepInbox(dataRoot)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("a day-old landing survived the sweep")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("a fresh landing was swept")
	}
}
