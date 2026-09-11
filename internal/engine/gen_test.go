package engine

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

//go:generate go run ./gen -root ../..

// The generated files are the contract between the screen and the engine
// (§248 B1, phase 2): api_gen.go is every exported method of *Engine, the
// desktop's forwarders are every one of them the screen does not implement
// itself, and rpc's client and server are the same list across the socket. A
// method added or changed without regenerating is caught here, not on the
// wire.
func TestGeneratedFilesAreCurrent(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go on PATH")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	cmd := exec.Command("go", "run", "./internal/engine/gen", "-root", root, "-out", out)
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gen: %v\n%s", err, b)
	}
	for _, pair := range [][2]string{
		{filepath.Join(out, "api_gen.go"), filepath.Join(root, "internal", "engine", "api_gen.go")},
		{filepath.Join(out, "engine_forwarders_gen.go"), filepath.Join(root, "desktop", "engine_forwarders_gen.go")},
		{filepath.Join(out, "client_gen.go"), filepath.Join(root, "internal", "engine", "rpc", "client_gen.go")},
		{filepath.Join(out, "server_gen.go"), filepath.Join(root, "internal", "engine", "rpc", "server_gen.go")},
	} {
		fresh, err := os.ReadFile(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		committed, err := os.ReadFile(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(fresh, committed) {
			t.Errorf("%s is stale — run `go generate ./internal/engine`", pair[1])
		}
	}
}
