package main

// The engine as a process: built, started the way the screen starts it,
// spoken to over its socket, and gone when the screen's stdin closes.

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine/rpc"
)

func buildEngine(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go on PATH")
	}
	bin := filepath.Join(t.TempDir(), "aetox-engine")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func TestTheEngineProcessServesAndLeavesWithItsScreen(t *testing.T) {
	bin := buildEngine(t)
	dataRoot := t.TempDir()
	cmd := exec.Command(bin, "serve", "--tcp", "127.0.0.1:0", "--token-stdin")
	cmd.Env = append(os.Environ(), "AETOX_DATA_ROOT="+dataRoot)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	t.Cleanup(func() {
		stdin.Close()
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
		}
	})

	token, err := rpc.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stdin.Write([]byte(token + "\n")); err != nil {
		t.Fatal(err)
	}

	// The one line on stdout says where it listens.
	lineCh := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		lineCh <- line
	}()
	var where struct {
		Network string `json:"network"`
		Address string `json:"address"`
		PID     int    `json:"pid"`
		Version string `json:"version"`
	}
	select {
	case line := <-lineCh:
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &where); err != nil {
			t.Fatalf("stdout line %q is not the listening line: %v", line, err)
		}
	case err := <-exited:
		t.Fatalf("the engine exited before it said where it listens: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("the engine never said where it listens")
	}
	if where.Network != "tcp" || where.Address == "" || where.PID == 0 {
		t.Fatalf("listening line = %+v", where)
	}

	// A screen: connect with the token, say hello, ask something.
	c := rpc.NewClient(rpc.ClientOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := c.Connect(ctx, where.Network, where.Address, token); err != nil {
		t.Fatalf("connect: %v", err)
	}
	hello, err := c.Hello(ctx, "test-screen", nil, nil)
	if err != nil {
		t.Fatalf("hello: %v", err)
	}
	if hello.PID != where.PID || hello.DataRoot != dataRoot {
		t.Errorf("hello = %+v, want the process on the stdout line at this data root", hello)
	}
	if v := c.AppVersion(); v != hello.Version {
		t.Errorf("AppVersion() = %q, hello said %q", v, hello.Version)
	}
	// The wrong token is refused by the process, not only by the library.
	bad := rpc.NewClient(rpc.ClientOptions{})
	if err := bad.Connect(ctx, where.Network, where.Address, "nope"); err == nil {
		t.Error("a wrong token connected to the running engine")
	}
	c.Close()

	// The screen going away — its stdin closed — is the engine's cue to leave.
	stdin.Close()
	select {
	case err := <-exited:
		if err != nil {
			t.Errorf("the engine exited with %v, want a clean exit", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the engine outlived its screen")
	}
	if b, err := os.ReadFile(filepath.Join(dataRoot, "logs", "engine.log")); err != nil || !strings.Contains(string(b), "listening on") {
		t.Errorf("engine.log does not say the process listened: %v\n%s", err, b)
	}
}

// The token is never an argument, and a start with no way to get one is
// refused before anything listens.
func TestTheEngineRefusesToStartWithoutAToken(t *testing.T) {
	bin := buildEngine(t)
	cmd := exec.Command(bin, "serve", "--tcp", "127.0.0.1:0")
	cmd.Env = append(os.Environ(), "AETOX_DATA_ROOT="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("started without a token:\n%s", out)
	}
	if !strings.Contains(string(out), "token") {
		t.Errorf("the refusal does not name the token:\n%s", out)
	}
}
