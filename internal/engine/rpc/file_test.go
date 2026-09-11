package rpc

// The project's files across the wire: served by the engine behind the
// token, proxied by the screen under the window's own path, with Range —
// the reason a video scrubs — passing through both.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// projectWithFile is an engine on the wire with a project open and one
// file in it, and the address of its listener.
func projectWithFile(t *testing.T) (addr string, content string) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	srv := NewServer(testToken, engine.NewEngine)
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		engine.Shutdown(srv.Engine(), ctx)
	})
	root := t.TempDir()
	content = strings.Repeat("0123456789", 100)
	if err := os.WriteFile(filepath.Join(root, "clip.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.Engine().OpenProjectPath(root); err != nil {
		t.Fatal(err)
	}
	return strings.TrimPrefix(hs.URL, "http://"), content
}

func TestTheEngineServesProjectFilesBehindTheToken(t *testing.T) {
	addr, content := projectWithFile(t)
	get := func(token, rng string) *http.Response {
		req, _ := http.NewRequest("GET", "http://"+addr+FilePath+"clip.txt", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}
	if resp := get("", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no token got %d, want 401", resp.StatusCode)
	}
	if resp := get("wrong", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("a wrong token got %d, want 401", resp.StatusCode)
	}
	resp := get(testToken, "")
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != content {
		t.Errorf("body differs (%d bytes)", len(body))
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Error("a project file must not be cached under a URL that never changes")
	}
	part := get(testToken, "bytes=10-19")
	if part.StatusCode != http.StatusPartialContent {
		t.Fatalf("a Range request got %d, want 206", part.StatusCode)
	}
	slice, _ := io.ReadAll(part.Body)
	if string(slice) != "0123456789" {
		t.Errorf("range body = %q", slice)
	}
	// Outside the project is refused, and a file that is not there is 404.
	if resp := get(testToken, ""); resp.StatusCode != 200 {
		t.Fatal("sanity")
	}
	req, _ := http.NewRequest("GET", "http://"+addr+FilePath+"../outside.txt", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	if resp, err := http.DefaultClient.Do(req); err == nil && resp.StatusCode == 200 {
		t.Error("a path outside the project was served")
	}
}

// The screen's /aetox-file/ reaches the same bytes through the proxy, with
// the token added on the way and the Range answered by the engine.
func TestTheScreenProxiesItsFilePathOntoTheEngine(t *testing.T) {
	addr, content := projectWithFile(t)
	fallthrough404 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	fixed := func(network, address, token string) Endpoint {
		return func() (string, string, string, bool) { return network, address, token, true }
	}
	screen := httptest.NewServer(FileProxy(fixed("tcp", addr, testToken), fallthrough404))
	t.Cleanup(screen.Close)

	resp, err := http.Get(screen.URL + ScreenFilePrefix + "clip.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != content {
		t.Errorf("body differs through the proxy (%d bytes)", len(body))
	}

	req, _ := http.NewRequest("GET", screen.URL+ScreenFilePrefix+"clip.txt", nil)
	req.Header.Set("Range", "bytes=0-4")
	part, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer part.Body.Close()
	if part.StatusCode != http.StatusPartialContent {
		t.Errorf("a Range through the proxy got %d, want 206", part.StatusCode)
	}
	if slice, _ := io.ReadAll(part.Body); string(slice) != "01234" {
		t.Errorf("range body through the proxy = %q", slice)
	}

	// Anything not under the prefix is the window's own business.
	other, err := http.Get(screen.URL + "/index.html")
	if err != nil {
		t.Fatal(err)
	}
	other.Body.Close()
	if other.StatusCode != http.StatusNotFound {
		t.Errorf("a request outside the prefix got %d from the proxy, want the fallthrough's 404", other.StatusCode)
	}

	// An engine that is not there is a 502 with a sentence, not a hang.
	dead := httptest.NewServer(FileProxy(fixed("tcp", "127.0.0.1:1", testToken), fallthrough404))
	t.Cleanup(dead.Close)
	gone, err := http.Get(dead.URL + ScreenFilePrefix + "clip.txt")
	if err != nil {
		t.Fatal(err)
	}
	gone.Body.Close()
	if gone.StatusCode != http.StatusBadGateway {
		t.Errorf("a dead engine got %d, want 502", gone.StatusCode)
	}
}
