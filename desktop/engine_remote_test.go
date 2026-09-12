package main

// The window on a host: the supervisor pointed at a fake host (the ssh in
// internal/engine/remote/testdata/fakessh) with a real engine behind it,
// the road walked with its steps on the chip, the window reloaded once
// the engine answers, a dropped tunnel redialed, and the way back.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/update"
)

var (
	fakeSSHOnce sync.Once
	fakeSSHBin  string
	fakeSSHErr  error
)

// builtFakeSSH is the fake host, built once for the package.
func builtFakeSSH(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go on PATH")
	}
	fakeSSHOnce.Do(func() {
		dir, err := os.MkdirTemp("", "aetox-fakessh")
		if err != nil {
			fakeSSHErr = err
			return
		}
		fakeSSHBin = filepath.Join(dir, "fakessh")
		if runtime.GOOS == "windows" {
			fakeSSHBin += ".exe"
		}
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", fakeSSHBin, "../internal/engine/remote/testdata/fakessh")
		if out, err := cmd.CombinedOutput(); err != nil {
			fakeSSHErr = err
			t.Logf("build: %s", out)
		}
	})
	if fakeSSHErr != nil {
		t.Fatalf("building the fake ssh: %v", fakeSSHErr)
	}
	return fakeSSHBin
}

// remoteFixture is a window with a child engine, a fake host it can go
// to, and the engine binary laid out where engineBinaryFor looks for the
// Linux one — the fake host runs whatever bytes it is sent, so on this OS
// the "Linux engine" is this OS's.
func remoteFixture(t *testing.T) (a *App, home string, reloads *atomic.Int32) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("AETOX_FAKE_HOME", home)
	t.Setenv("AETOX_SSH", builtFakeSSH(t))
	linuxDir := t.TempDir()
	src, _ := os.ReadFile(builtEngine(t))
	if err := os.WriteFile(filepath.Join(linuxDir, update.EngineAssetName("linux", "amd64")), src, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETOX_ENGINE_LINUX_DIR", linuxDir)
	t.Cleanup(func() {
		b, err := os.ReadFile(filepath.Join(home, ".aetox", "server", "engine.pid"))
		if err != nil {
			return
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
			// Waited for, then a beat, before t.TempDir removes the binary
			// the fake host ran (remote_test.go says why).
			_, _ = p.Wait()
			time.Sleep(300 * time.Millisecond)
		}
	})
	a, _, _ = liveApp(t)
	reloads = &atomic.Int32{}
	a.reload = func() { reloads.Add(1) }
	return a, home, reloads
}

func waitFor(t *testing.T, within time.Duration, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("never: %s", what)
}

func TestTheWindowGoesToAHostAndComesBack(t *testing.T) {
	a, home, reloads := remoteFixture(t)
	localPID := a.EngineStatus().PID
	if localPID == 0 {
		t.Fatal("no child before the switch")
	}

	if err := a.SaveRemoteHost("box", "user@box", home); err != nil {
		t.Fatal(err)
	}
	hosts := a.RemoteHosts()
	if len(hosts.Hosts) != 1 || hosts.Hosts[0].Target != "user@box" || hosts.Active != "" {
		t.Fatalf("RemoteHosts = %+v", hosts)
	}
	if err := a.ConnectRemote("box"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 60*time.Second, "connected to the host", func() bool {
		st := a.EngineStatus()
		return st.State == engineConnected && st.Mode == modeRemote
	})
	st := a.EngineStatus()
	if st.Host != "box" || st.PID != 0 || !strings.HasPrefix(st.Address, "127.0.0.1:") {
		t.Errorf("status on the host = %+v", st)
	}
	if st.Restarts != 0 {
		t.Errorf("a switch counted as %d restarts", st.Restarts)
	}
	waitFor(t, 5*time.Second, "the window reloaded after the switch", func() bool { return reloads.Load() == 1 })
	// The engine on the far side is the one the fake host started, with
	// the project the host row names.
	if got := a.api.GetProjectStatus(); got.Path != home {
		t.Errorf("the host's engine has project %q; want %q", got.Path, home)
	}
	hosts = a.RemoteHosts()
	if hosts.Active != "box" || hosts.Hosts[0].Version == "" || hosts.Hosts[0].Arch != "amd64" {
		t.Errorf("after connecting, RemoteHosts = %+v", hosts)
	}
	// The token is on disk for the next window, and not in the view.
	c, err := loadScreenConfig()
	if err != nil {
		t.Fatal(err)
	}
	if h, _ := c.host("box"); h.Token == "" {
		t.Error("screen.json holds no token for the engine just started")
	}
	if err := a.DisconnectRemote(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 60*time.Second, "back on this machine", func() bool {
		st := a.EngineStatus()
		return st.State == engineConnected && st.Mode == modeLocal
	})
	if a.EngineStatus().PID == 0 {
		t.Error("back on this machine with no child of our own")
	}
	waitFor(t, 5*time.Second, "the window reloaded again", func() bool { return reloads.Load() == 2 })
	if got := a.RemoteHosts().Active; got != "" {
		t.Errorf("active host after disconnect = %q", got)
	}
	// The engine on the host is still there for the next connection.
	if _, err := os.Stat(filepath.Join(home, ".aetox", "server", "engine.pid")); err != nil {
		t.Error("the host's engine was stopped by a disconnect; it should be left to its idle clock")
	}
}

func TestADroppedTunnelIsOpenedAgainWithoutTouchingTheEngine(t *testing.T) {
	a, home, _ := remoteFixture(t)
	if err := a.SaveRemoteHost("box", "user@box", ""); err != nil {
		t.Fatal(err)
	}
	if err := a.ConnectRemote("box"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 60*time.Second, "connected to the host", func() bool {
		st := a.EngineStatus()
		return st.State == engineConnected && st.Mode == modeRemote
	})
	pidBefore, _ := os.ReadFile(filepath.Join(home, ".aetox", "server", "engine.pid"))
	before := a.api.AppVersion()

	// The network went away: the tunnel process dies.
	a.engine.mu.Lock()
	p := a.engine.proc
	a.engine.mu.Unlock()
	p.leave()
	waitFor(t, 60*time.Second, "connected again", func() bool {
		st := a.EngineStatus()
		return st.State == engineConnected && st.Mode == modeRemote && st.Restarts >= 1
	})
	if got := a.api.AppVersion(); got != before || got == "" {
		t.Errorf("AppVersion after the redial = %q; before %q", got, before)
	}
	pidAfter, _ := os.ReadFile(filepath.Join(home, ".aetox", "server", "engine.pid"))
	if string(pidBefore) != string(pidAfter) {
		t.Errorf("the engine on the host was replaced (%s -> %s); only the tunnel should have been", strings.TrimSpace(string(pidBefore)), strings.TrimSpace(string(pidAfter)))
	}
}

func TestAWindowStartsOnTheHostItWasLeftOn(t *testing.T) {
	a, home, _ := remoteFixture(t)
	if err := a.SaveRemoteHost("box", "user@box", ""); err != nil {
		t.Fatal(err)
	}
	if err := a.ConnectRemote("box"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 60*time.Second, "connected to the host", func() bool {
		st := a.EngineStatus()
		return st.State == engineConnected && st.Mode == modeRemote
	})
	// A second window, same DataRoot: it goes straight to the host.
	b := NewApp()
	captureEvents(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go b.engine.run(ctx)
	waitFor(t, 60*time.Second, "the next window on the host", func() bool {
		st := b.EngineStatus()
		return st.State == engineConnected && st.Mode == modeRemote && st.Host == "box"
	})
	cancel()
	<-b.engine.stopped
	_ = home
}

func TestAHostThatCannotBeReachedIsAFailureWithWords(t *testing.T) {
	a, _, _ := remoteFixture(t)
	t.Setenv("AETOX_FAKE_FAIL", "auth")
	if err := a.SaveRemoteHost("box", "user@box", ""); err != nil {
		t.Fatal(err)
	}
	if err := a.ConnectRemote("box"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 30*time.Second, "the failure", func() bool { return a.EngineStatus().State == engineFailed })
	st := a.EngineStatus()
	if !strings.Contains(st.Detail, "Permission denied") || st.Mode != modeRemote {
		t.Errorf("status = %+v; want ssh's words and the remote mode", st)
	}
	// The way back does not need the host.
	if err := a.DisconnectRemote(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 30*time.Second, "back on this machine", func() bool {
		st := a.EngineStatus()
		return st.State == engineConnected && st.Mode == modeLocal
	})
}

func TestForgettingTheActiveHostIsRefused(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := NewApp()
	if err := a.SaveRemoteHost("box", "user@box", ""); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveRemoteHost("", "-oProxyCommand=calc", ""); err == nil {
		t.Error("a target shaped like an option was saved")
	}
	c, _ := loadScreenConfig()
	c.ActiveHost = "box"
	if err := saveScreenConfig(c); err != nil {
		t.Fatal(err)
	}
	if err := a.ForgetRemoteHost("box"); err == nil {
		t.Error("the active host was forgotten")
	}
	c.ActiveHost = ""
	_ = saveScreenConfig(c)
	if err := a.ForgetRemoteHost("box"); err != nil {
		t.Fatal(err)
	}
	if len(a.RemoteHosts().Hosts) != 0 {
		t.Error("the host is still listed")
	}
}
