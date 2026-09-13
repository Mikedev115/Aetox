package remote

// The road to a host, driven against the fake host in testdata/fakessh:
// the sequence of steps, what goes up on stdin, what is read back, the
// tunnel's life — with a real engine process on the far end of it, so a
// Hello through the tunnel is the proof.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/version"
)

var (
	buildOnce sync.Once
	fakeSSH   string
	engineBin string
	buildErr  error
)

// built is testdata/fakessh and cmd/aetox-engine, built once for the package.
func built(t *testing.T) (ssh, engine string) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go on PATH")
	}
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "aetox-remote-bin")
		if err != nil {
			buildErr = err
			return
		}
		exe := ""
		if runtime.GOOS == "windows" {
			exe = ".exe"
		}
		fakeSSH = filepath.Join(dir, "fakessh"+exe)
		engineBin = filepath.Join(dir, "aetox-engine"+exe)
		for _, b := range [][2]string{{fakeSSH, "./testdata/fakessh"}, {engineBin, "../../../cmd/aetox-engine"}} {
			out, err := exec.Command("go", "build", "-buildvcs=false", "-o", b[0], b[1]).CombinedOutput()
			if err != nil {
				buildErr = err
				t.Logf("build %s: %s", b[1], out)
				return
			}
		}
	})
	if buildErr != nil {
		t.Fatalf("building: %v", buildErr)
	}
	return fakeSSH, engineBin
}

// fakeHost is one fake host's home, with the engine it may start killed
// when the test ends.
type fakeHost struct {
	home string
	log  string
}

func newFakeHost(t *testing.T) *fakeHost {
	t.Helper()
	h := &fakeHost{home: t.TempDir()}
	h.log = filepath.Join(h.home, "calls.log")
	t.Setenv("AETOX_FAKE_HOME", h.home)
	t.Setenv("AETOX_FAKE_LOG", h.log)
	t.Setenv("AETOX_SSH", "")
	t.Cleanup(h.killEngine)
	return h
}

func (h *fakeHost) killEngine() {
	b, err := os.ReadFile(filepath.Join(h.home, ".aetox", "server", "engine.pid"))
	if err != nil {
		return
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
		// Waited for, and then a beat: t.TempDir's cleanup runs next and
		// Windows keeps an exe locked for a moment after its process is
		// gone — "Access is denied" on the binary under load otherwise.
		_, _ = p.Wait()
		time.Sleep(300 * time.Millisecond)
	}
}

// calls is the steps the fake host saw, in order.
func (h *fakeHost) calls() []string {
	b, _ := os.ReadFile(h.log)
	return strings.Fields(string(b))
}

func (h *fakeHost) reset() { _ = os.Remove(h.log) }

func newDriver(t *testing.T) (*Driver, *fakeHost) {
	t.Helper()
	ssh, engine := built(t)
	h := newFakeHost(t)
	d := &Driver{
		SSH: ssh,
		Binary: func(ctx context.Context, arch string, progress func(done, total int64)) (string, error) {
			if arch != "amd64" {
				t.Errorf("Binary asked for arch %q; the fake host is x86_64", arch)
			}
			return engine, nil
		},
		Log: t.Logf,
	}
	return d, h
}

// hello dials the tunnel with an rpc client and says hello: the whole road
// in one round trip.
func hello(t *testing.T, tun *Tunnel, token string) rpc.HelloResult {
	t.Helper()
	c := rpc.NewClient(rpc.ClientOptions{})
	t.Cleanup(func() { _ = c.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := c.Connect(ctx, "tcp", tun.Local, token); err != nil {
		t.Fatalf("dialing through the tunnel: %v", err)
	}
	res, err := c.Hello(ctx, version.Current, nil, nil)
	if err != nil {
		t.Fatalf("hello through the tunnel: %v", err)
	}
	return res
}

func TestConnectInstallsStartsAndTunnelsOnAFreshHost(t *testing.T) {
	d, h := newDriver(t)
	host := Host{Name: "box", Target: "user@box", Root: h.home}
	var steps []string
	var uploaded int64
	tun, err := d.Connect(context.Background(), &host, version.Current, func(s Step) {
		if len(steps) == 0 || steps[len(steps)-1] != s.Name {
			steps = append(steps, s.Name)
		}
		if s.Name == "install" && s.Done > uploaded {
			uploaded = s.Done
		}
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer tun.Close()

	if want := []string{"probe", "download", "install", "start", "tunnel"}; strings.Join(steps, " ") != strings.Join(want, " ") {
		t.Errorf("steps = %v; want %v", steps, want)
	}
	if got := h.calls(); strings.Join(got, " ") != "probe install start tunnel" {
		t.Errorf("the host saw %v", got)
	}
	st, _ := os.Stat(engineBin)
	if uploaded != st.Size() {
		t.Errorf("install reported %d bytes; the binary is %d", uploaded, st.Size())
	}
	if host.Token == "" || host.Version != version.Current || host.Arch != "amd64" || host.LastUsed.IsZero() {
		t.Errorf("host after Connect = %+v; want token, version, arch and a time", host)
	}
	// The token went up on stdin and into a file only the user reads.
	tok, err := os.ReadFile(filepath.Join(h.home, ".aetox", "server", "token"))
	if err != nil || strings.TrimSpace(string(tok)) != host.Token {
		t.Errorf("token on the host = %q (%v); want %q", tok, err, host.Token)
	}
	res := hello(t, tun, host.Token)
	if res.Version != version.Current || res.PID == 0 {
		t.Errorf("hello through the tunnel = %+v", res)
	}
	if res.Root != h.home {
		t.Errorf("the engine opened %q; --root was %q", res.Root, h.home)
	}
}

func TestConnectReusesARunningEngineOfThisVersion(t *testing.T) {
	d, h := newDriver(t)
	host := Host{Target: "user@box"}
	tun, err := d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("first Connect: %v", err)
	}
	tun.Close()
	token := host.Token
	h.reset()

	tun, err = d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("second Connect: %v", err)
	}
	defer tun.Close()
	if got := h.calls(); strings.Join(got, " ") != "probe tunnel" {
		t.Errorf("the second road was %v; want probe and tunnel only", got)
	}
	if host.Token != token {
		t.Error("the token changed although the engine was kept")
	}
	hello(t, tun, host.Token)
}

func TestConnectReplacesAnEngineWhoseTokenIsLost(t *testing.T) {
	d, h := newDriver(t)
	host := Host{Target: "user@box"}
	tun, err := d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("first Connect: %v", err)
	}
	tun.Close()
	old := host.Token
	host.Token = "" // screen.json gone
	h.reset()

	tun, err = d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("second Connect: %v", err)
	}
	defer tun.Close()
	if got := h.calls(); strings.Join(got, " ") != "probe start tunnel" {
		t.Errorf("the road was %v; want probe, start (no install: same version), tunnel", got)
	}
	if host.Token == "" || host.Token == old {
		t.Error("a fresh token was expected for the fresh engine")
	}
	hello(t, tun, host.Token)
}

func TestConnectReplacesAnEngineOfAnotherVersion(t *testing.T) {
	d, h := newDriver(t)
	host := Host{Target: "user@box"}
	tun, err := d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("first Connect: %v", err)
	}
	tun.Close()
	// The host now believes it runs an older version.
	if err := os.WriteFile(filepath.Join(h.home, ".aetox", "server", "engine.version"), []byte("0.0.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h.reset()

	tun, err = d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("second Connect: %v", err)
	}
	defer tun.Close()
	if got := h.calls(); strings.Join(got, " ") != "probe start tunnel" {
		t.Errorf("the road was %v; want the old engine replaced", got)
	}
	hello(t, tun, host.Token)
}

func TestSSHFailuresCarryTheirOwnWords(t *testing.T) {
	d, _ := newDriver(t)
	t.Setenv("AETOX_FAKE_FAIL", "auth")
	host := Host{Target: "user@box"}
	_, err := d.Connect(context.Background(), &host, version.Current, nil)
	if err == nil || !strings.Contains(err.Error(), "Permission denied (publickey)") {
		t.Errorf("err = %v; want ssh's sentence", err)
	}
}

func TestAHostThatIsNotLinuxIsRefusedBeforeAnythingIsSent(t *testing.T) {
	d, h := newDriver(t)
	t.Setenv("AETOX_FAKE_UNAME", "Darwin arm64")
	host := Host{Target: "user@mac"}
	_, err := d.Connect(context.Background(), &host, version.Current, nil)
	if err == nil || !strings.Contains(err.Error(), "Linux") {
		t.Errorf("err = %v; want a refusal naming Linux", err)
	}
	if got := h.calls(); strings.Join(got, " ") != "probe" {
		t.Errorf("the host saw %v; want the probe only", got)
	}
}

func TestATargetThatLooksLikeAnOptionIsRefused(t *testing.T) {
	for _, bad := range []string{"", "-oProxyCommand=calc", "user@box extra", "-N"} {
		if err := CheckTarget(bad); err == nil {
			t.Errorf("CheckTarget(%q) accepted", bad)
		}
	}
	for _, ok := range []string{"user@box", "box", "u@192.168.1.10", "deploy@[fe80::1]", "user@host.example.com"} {
		if err := CheckTarget(ok); err != nil {
			t.Errorf("CheckTarget(%q) = %v", ok, err)
		}
	}
}

func TestATunnelThatDiesSaysSo(t *testing.T) {
	d, _ := newDriver(t)
	host := Host{Target: "user@box"}
	tun, err := d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	// The network went away: ssh is gone.
	_ = tun.cmd.Process.Kill()
	select {
	case <-tun.Exited():
	case <-time.After(10 * time.Second):
		t.Fatal("Exited did not close after the ssh process died")
	}
	if tun.Err() == nil {
		t.Error("a killed tunnel reports no error")
	}
}

func TestStopEndsTheEngineOnTheHost(t *testing.T) {
	d, h := newDriver(t)
	host := Host{Target: "user@box"}
	tun, err := d.Connect(context.Background(), &host, version.Current, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	tun.Close()
	if err := d.Stop(context.Background(), host); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	p, err := d.Probe(context.Background(), host, version.Current)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if p.Running {
		t.Error("the engine still answers after Stop")
	}
	if !p.Installed {
		t.Error("Stop removed the binary; it should only end the process")
	}
	_ = h
}

func TestScriptsQuoteThePathTheyCarry(t *testing.T) {
	s := startScript("1.2.3", `/home/u/it's here`, "30m")
	if !strings.Contains(s, `--root '/home/u/it'\''s here'`) {
		t.Errorf("root not single-quoted: %s", s)
	}
	w := wrap(`echo 'a'`)
	if w != `sh -c 'echo '\''a'\'''` {
		t.Errorf("wrap = %s", w)
	}
}

func TestParseProbeReadsWhatTheHostSays(t *testing.T) {
	p, err := parseProbe("Linux aarch64\ninstalled=yes\nrunning=yes\npid=4242\nversion=1.5.28\naddress={\"network\":\"tcp\",\"address\":\"127.0.0.1:40123\",\"pid\":4242,\"version\":\"1.5.28\"}\n")
	if err != nil {
		t.Fatal(err)
	}
	want := Probe{OS: "linux", Arch: "arm64", Installed: true, Running: true, PID: 4242, Version: "1.5.28", Address: "127.0.0.1:40123"}
	if p != want {
		t.Errorf("parseProbe = %+v; want %+v", p, want)
	}
	if _, err := parseProbe(""); err == nil {
		t.Error("an empty answer parsed")
	}
}

// ssh's own words come back when there are any; when there are none, the
// exit status is explained rather than quoted — Windows' ssh says nothing
// under LogLevel=ERROR for a refused connection, which is how "exit status
// 255" reached the owner's status card.
func TestSSHWordsExplainASilentExit(t *testing.T) {
	if got := sshWords("Warning: Permanently added 'x' (ED25519) to the list of known hosts.\nuser@x: Permission denied (publickey).\n", errors.New("exit status 255")); got != "user@x: Permission denied (publickey)." {
		t.Errorf("with words: %q", got)
	}
	if got := sshWords("", errors.New("exit status 255")); !strings.Contains(got, "255") || !strings.Contains(got, "ติดต่อเครื่องไม่ได้") {
		t.Errorf("silent 255: %q", got)
	}
	if got := sshWords("", errors.New("exit status 1")); got != "exit status 1" {
		t.Errorf("silent 1: %q", got)
	}
}
