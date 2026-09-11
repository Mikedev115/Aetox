package main

// The engine as this window's child, for real: the binary built, started
// the way startup starts it, a turn run through the socket, the process
// killed under a running window and started again, the window closed and
// the child gone with it.

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	engineBinOnce sync.Once
	engineBin     string
	engineBinErr  error
)

// builtEngine is cmd/aetox-engine built once for the package.
func builtEngine(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go on PATH")
	}
	engineBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "aetox-engine-bin")
		if err != nil {
			engineBinErr = err
			return
		}
		engineBin = filepath.Join(dir, "aetox-engine")
		if runtime.GOOS == "windows" {
			engineBin += ".exe"
		}
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", engineBin, "../cmd/aetox-engine")
		if out, err := cmd.CombinedOutput(); err != nil {
			engineBinErr = err
			t.Logf("build: %s", out)
		}
	})
	if engineBinErr != nil {
		t.Fatalf("building the engine: %v", engineBinErr)
	}
	return engineBin
}

// liveApp is NewApp with its child running, under a context the test ends.
func liveApp(t *testing.T) (*App, *recorder, context.CancelFunc) {
	t.Helper()
	t.Setenv("AETOX_ENGINE", builtEngine(t))
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := NewApp()
	rec := captureEvents(a)
	ctx, cancel := context.WithCancel(context.Background())
	go a.engine.run(ctx)
	t.Cleanup(func() {
		cancel()
		select {
		case <-a.engine.stopped:
		case <-time.After(15 * time.Second):
			t.Error("the supervisor did not stop with the context")
		}
	})
	waitForState(t, a, engineConnected, 30*time.Second)
	return a, rec, cancel
}

func waitForState(t *testing.T, a *App, state string, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if a.EngineStatus().State == state {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the engine never reached %q; status is %+v", state, a.EngineStatus())
}

func TestTheWindowRunsATurnOnItsChildEngine(t *testing.T) {
	a, rec, _ := liveApp(t)
	st := a.EngineStatus()
	if st.PID == 0 || st.Address == "" {
		t.Fatalf("connected with no process behind it: %+v", st)
	}

	if _, err := a.api.OpenProjectPath(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.api.SwitchProvider("aetox"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.api.SwitchModel("aetox-render:test"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.api.SwitchApprovalMode("full-access"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.api.NewSession(); err != nil {
		t.Fatal(err)
	}
	reply, err := a.SendMessage("เทสๆ", "")
	if err != nil {
		t.Fatalf("SendMessage through the child: %v", err)
	}
	if !strings.Contains(reply.Text, "ทดสอบ Markdown") {
		t.Errorf("reply = %q", reply.Text)
	}
	// The events crossed the socket and reached the window's event door,
	// stamped with the session, as JSON the frontend reads as it always did.
	deliveries := 0
	for _, e := range rec.all() {
		if e.Name != "agent:chunk" || len(e.Data) == 0 {
			continue
		}
		raw, ok := e.Data[0].(json.RawMessage)
		if !ok {
			t.Fatalf("agent:chunk reached the window as %T, want the wire's JSON", e.Data[0])
		}
		var ch struct {
			SessionID string `json:"sessionId"`
			Data      struct {
				Text    string `json:"text"`
				Replace bool   `json:"replace"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &ch); err != nil {
			t.Fatal(err)
		}
		if ch.SessionID == "" {
			t.Error("a chunk with no session on it")
		}
		if ch.Data.Replace && strings.TrimSpace(ch.Data.Text) != "" {
			deliveries++
		}
	}
	if deliveries != 1 {
		t.Errorf("%d deliveries crossed, want exactly one", deliveries)
	}
	// The project's files are reachable under the window's own path.
	if _, _, _, ok := a.engineEndpoint(); !ok {
		t.Error("the file proxy has no endpoint while the engine is connected")
	}
}

// A crash restarts the engine; the window's bindings wait through it rather
// than failing; the chip counts it.
func TestACrashedEngineIsStartedAgainAndTheWindowCarriesOn(t *testing.T) {
	a, rec, _ := liveApp(t)
	pid := a.EngineStatus().PID
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatal(err)
	}
	// The supervisor notices — the wire drops, the redial finds no process.
	deadline := time.Now().Add(15 * time.Second)
	for a.EngineStatus().State == engineConnected && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	// A binding made while it is down waits for the new wire rather than
	// failing; one that was in flight when the wire dropped fails, which is
	// the honest answer for a call whose engine died under it.
	done := make(chan string, 1)
	go func() { done <- a.api.AppVersion() }()

	deadline = time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if st := a.EngineStatus(); st.State == engineConnected && st.Restarts >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	st := a.EngineStatus()
	if st.State != engineConnected || st.Restarts != 1 || st.PID == pid || st.PID == 0 {
		t.Errorf("after the kill, status = %+v; want one restart and a new process", st)
	}
	select {
	case v := <-done:
		if v == "" {
			t.Error("the binding made during the restart answered empty")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the binding made during the restart never answered")
	}
	states := []string{}
	for _, e := range rec.all() {
		if e.Name == "engine:status" && len(e.Data) == 1 {
			if st, ok := e.Data[0].(EngineStatus); ok {
				states = append(states, st.State)
			}
		}
	}
	joined := strings.Join(states, ",")
	if !strings.Contains(joined, engineRestarting) || !strings.HasSuffix(joined, engineConnected) {
		t.Errorf("status events = %v; want a restart announced and connected at the end", states)
	}
}

// Closing the window: the engine is asked to finish first, then told to
// leave, and it is gone before shutdown returns.
func TestClosingTheWindowTakesTheEngineWithIt(t *testing.T) {
	a, _, cancel := liveApp(t)
	pid := a.EngineStatus().PID
	if prevent := a.beforeClose(context.Background()); prevent {
		t.Error("beforeClose prevented the close")
	}
	cancel()
	a.shutdown(context.Background())
	select {
	case <-a.engine.stopped:
	case <-time.After(15 * time.Second):
		t.Fatal("the supervisor is still running after shutdown")
	}
	if alive(pid) {
		t.Errorf("engine %d is still running after the window shut down", pid)
	}
}

// alive reports whether a process still runs, the portable way: a signal
// of nothing on Unix, a handle that still opens on Windows.
func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		// FindProcess succeeds for any pid on Windows; Wait fails at once
		// for one that is gone or not ours. A process we did not start
		// cannot be waited on, so check by the exit state Kill reports.
		err := p.Signal(os.Kill)
		return err == nil
	}
	return p.Signal(nil) == nil
}
