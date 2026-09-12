package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeRelease serves one release's files by tag — the engine binary, a
// checksums.txt that lists it, and the signature over that — under a
// throwaway key installed as the release key for the test.
func fakeRelease(t *testing.T, version string, engine []byte, listEngine bool) (hits *atomic.Int32) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	orig := releasePublicKey
	releasePublicKey = base64.StdEncoding.EncodeToString(pub)
	name := EngineAssetName("linux", "amd64")
	sum := sha256.Sum256(engine)
	sums := "deadbeef  aetox-windows-amd64-portable.zip\n"
	if listEngine {
		sums += hex.EncodeToString(sum[:]) + "  " + name + "\n"
	}
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(sums))) + "\n"
	hits = &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := "/v" + version + "/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		switch strings.TrimPrefix(r.URL.Path, prefix) {
		case checksumsName:
			_, _ = w.Write([]byte(sums))
		case signatureName:
			_, _ = w.Write([]byte(sig))
		case name:
			hits.Add(1)
			_, _ = w.Write(engine)
		default:
			http.NotFound(w, r)
		}
	}))
	origBase := releaseDownloadBase
	releaseDownloadBase = srv.URL + "/"
	t.Cleanup(func() {
		srv.Close()
		releasePublicKey = orig
		releaseDownloadBase = origBase
	})
	return hits
}

func TestFetchEngineDownloadsVerifiesAndKeeps(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	engine := []byte("ELF but not really")
	hits := fakeRelease(t, "9.9.9", engine, true)

	var last int64
	path, err := FetchEngine(context.Background(), "9.9.9", "linux", "amd64", func(done, total int64) { last = done })
	if err != nil {
		t.Fatalf("FetchEngine: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(engine) {
		t.Fatalf("on disk %q (%v); want the release's bytes", got, err)
	}
	if last != int64(len(engine)) {
		t.Errorf("progress ended at %d; want %d", last, len(engine))
	}
	if !strings.HasSuffix(path, EngineAssetName("linux", "amd64")) {
		t.Errorf("path = %s; want the asset's own name", path)
	}

	// Second time: the cached copy, re-hashed, no download.
	again, err := FetchEngine(context.Background(), "9.9.9", "linux", "amd64", nil)
	if err != nil || again != path {
		t.Fatalf("second FetchEngine = %s, %v", again, err)
	}
	if hits.Load() != 1 {
		t.Errorf("the binary was downloaded %d times; want once", hits.Load())
	}

	// A cached copy that no longer matches is fetched again, not trusted.
	if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := FetchEngine(context.Background(), "9.9.9", "linux", "amd64", nil); err != nil {
		t.Fatalf("third FetchEngine: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != string(engine) {
		t.Error("the tampered copy was kept")
	}
	if hits.Load() != 2 {
		t.Errorf("the binary was downloaded %d times; want twice", hits.Load())
	}
}

func TestFetchEngineRefusesAReleaseWithoutTheAsset(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	fakeRelease(t, "9.9.9", []byte("x"), false)
	_, err := FetchEngine(context.Background(), "9.9.9", "linux", "amd64", nil)
	if err == nil || !strings.Contains(err.Error(), "Linux") {
		t.Errorf("err = %v; want a refusal naming the missing Linux engine", err)
	}
}

func TestFetchEngineRefusesABadSignature(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	fakeRelease(t, "9.9.9", []byte("x"), true)
	// Another key than the one that signed.
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	releasePublicKey = base64.StdEncoding.EncodeToString(pub)
	_, err := FetchEngine(context.Background(), "9.9.9", "linux", "amd64", nil)
	if err == nil || !strings.Contains(err.Error(), "ลายเซ็น") {
		t.Errorf("err = %v; want the signature refusal", err)
	}
}
