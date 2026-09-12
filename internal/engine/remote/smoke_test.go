package remote

// The road to a REAL host over the REAL ssh: the scripts as POSIX sh on a
// Linux machine, nohup and the pid file, the tunnel through sshd. Opt-in,
// because it needs a host:
//
//	AETOX_REMOTE_SMOKE=user@host go test ./internal/engine/remote -run TestRemoteSmoke -v
//
// The engine sent up is AETOX_ENGINE_LINUX (a linux build of
// cmd/aetox-engine for the host's architecture), or — when this test runs
// on Linux itself, as the CI job does against localhost — the engine built
// here. scripts/remote-smoke.sh sets a Linux machine up to be its own host.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/version"
)

func TestRemoteSmoke(t *testing.T) {
	target := strings.TrimSpace(os.Getenv("AETOX_REMOTE_SMOKE"))
	if target == "" {
		t.Skip("AETOX_REMOTE_SMOKE=user@host is not set")
	}
	binary := strings.TrimSpace(os.Getenv("AETOX_ENGINE_LINUX"))
	if binary == "" {
		if runtime.GOOS != "linux" {
			t.Skip("AETOX_ENGINE_LINUX is not set and this is not Linux")
		}
		dir := t.TempDir()
		binary = filepath.Join(dir, "aetox-engine")
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "../../../cmd/aetox-engine")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building the engine: %v\n%s", err, out)
		}
	}
	d := &Driver{
		Binary: func(ctx context.Context, arch string, progress func(done, total int64)) (string, error) {
			if arch != runtime.GOARCH && os.Getenv("AETOX_ENGINE_LINUX") == "" {
				t.Fatalf("the host is %s and the engine built here is %s", arch, runtime.GOARCH)
			}
			return binary, nil
		},
		Log:      t.Logf,
		IdleExit: "2m",
	}
	host := Host{Name: "smoke", Target: target}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// A clean slate on the host: whatever a previous run left is stopped.
	if err := d.Stop(ctx, host); err != nil {
		t.Fatalf("Stop (first contact): %v", err)
	}
	var steps []string
	tun, err := d.Connect(ctx, &host, version.Current, func(s Step) {
		if len(steps) == 0 || steps[len(steps)-1] != s.Name {
			steps = append(steps, s.Name)
		}
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Logf("steps: %v; engine %s on %s at %s", steps, host.Version, host.Arch, tun.Local)

	c := rpc.NewClient(rpc.ClientOptions{})
	defer c.Close()
	if err := c.Connect(ctx, "tcp", tun.Local, host.Token); err != nil {
		t.Fatalf("dial through the tunnel: %v", err)
	}
	res, err := c.Hello(ctx, version.Current, nil, nil)
	if err != nil {
		t.Fatalf("hello: %v", err)
	}
	if res.OS != "linux" || res.Version != version.Current {
		t.Errorf("hello = %+v", res)
	}
	// Two bindings the remote picker and the terminal live on.
	home := c.HomeDir()
	if !strings.HasPrefix(home, "/") {
		t.Errorf("HomeDir on the host = %q", home)
	}
	if _, err := c.ListDir(home); err != nil {
		t.Errorf("ListDir(%s): %v", home, err)
	}
	shells := c.TerminalShells()
	if len(shells) == 0 {
		t.Error("TerminalShells on the host answered none")
	}
	tail, err := d.Tail(ctx, host)
	if err != nil {
		t.Errorf("Tail: %v", err)
	}
	t.Logf("tail:\n%s", tail)

	// The tunnel closed leaves the engine; a second Connect finds it.
	tun.Close()
	p, err := d.Probe(ctx, host, version.Current)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !p.Running || p.Version != version.Current {
		t.Errorf("after the tunnel closed the host says %+v; want the engine still running", p)
	}
	if err := d.Stop(ctx, host); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if p, _ := d.Probe(ctx, host, version.Current); p.Running {
		t.Error("the engine survived Stop")
	}
}
