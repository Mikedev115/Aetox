package main

// The doors on a REAL host over the REAL ssh (host_files.go): a file from
// this machine attached to a project on the host, the trip cleared, and a
// file of that project opened here as a fetched copy. Opt-in, like
// remote.TestRemoteSmoke, because it needs a host:
//
//	AETOX_REMOTE_SMOKE=wsl AETOX_ENGINE_LINUX=<linux build of cmd/aetox-engine> \
//	  go test ./desktop -run TestRemoteSmokeHostFiles -v

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine/remote"
	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/version"
)

func TestRemoteSmokeHostFiles(t *testing.T) {
	target := strings.TrimSpace(os.Getenv("AETOX_REMOTE_SMOKE"))
	binary := strings.TrimSpace(os.Getenv("AETOX_ENGINE_LINUX"))
	if target == "" || binary == "" {
		t.Skip("AETOX_REMOTE_SMOKE=user@host and AETOX_ENGINE_LINUX=<engine> are not both set")
	}
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	d := &remote.Driver{
		Binary:   func(context.Context, string, func(done, total int64)) (string, error) { return binary, nil },
		Log:      t.Logf,
		IdleExit: "2m",
	}
	host := remote.Host{Name: "smoke", Target: target}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := d.Stop(ctx, host); err != nil {
		t.Fatalf("Stop (first contact): %v", err)
	}
	tun, err := d.Connect(ctx, &host, version.Current, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() {
		tun.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		d.Stop(ctx, host)
	})

	// The screen, as NewApp builds it, on the wire through the tunnel — and
	// told it is on a host the way the supervisor would tell it.
	a := &App{}
	a.client = a.newClient()
	a.api = a.client
	if err := a.client.Connect(ctx, "tcp", tun.Local, host.Token); err != nil {
		t.Fatalf("dial through the tunnel: %v", err)
	}
	t.Cleanup(func() { a.client.Close() })
	if _, err := a.client.Hello(ctx, version.Current, (appScreen{a}).WindowTools(nil), []string{rpc.FeatureWindowTools, rpc.FeatureProviderProxy}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	a.engine = &localEngine{
		token:  host.Token,
		target: engineTarget{mode: modeRemote, host: host},
		proc:   &engineProcess{network: "tcp", address: tun.Local, remote: host.Label()},
		status: EngineStatus{State: engineConnected, Mode: modeRemote, Host: host.Label()},
	}

	// A project on the host, made there.
	root := "/tmp/aetox-smoke-hostfiles"
	onHost := func(script string) string {
		t.Helper()
		out, err := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", target, script).CombinedOutput()
		if err != nil {
			t.Fatalf("ssh %s: %v\n%s", script, err, out)
		}
		return string(out)
	}
	onHost("rm -rf " + root + " && mkdir -p " + root + " && printf 'PDF plan' > " + root + "/plan.pdf")
	if _, err := a.api.OpenProjectPath(root); err != nil {
		t.Fatalf("OpenProjectPath on the host: %v", err)
	}

	// 1. A file from this machine, attached: it lands in the project THERE.
	local := filepath.Join(t.TempDir(), "screenshot.png")
	if err := os.WriteFile(local, []byte("\x89PNG smoke"), 0o644); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	rel, err := a.SaveChatImage(local)
	if err != nil {
		t.Fatalf("SaveChatImage over ssh: %v", err)
	}
	t.Logf("attached as %s in %s", rel, time.Since(started).Round(time.Millisecond))
	if got := onHost("cat " + root + "/" + rel); got != "\x89PNG smoke" {
		t.Errorf("on the host the attachment reads %q", got)
	}
	// The trip is cleared on the host: the engine's inbox is empty.
	if got := strings.TrimSpace(onHost("ls -A ~/.config/aetox/inbox 2>/dev/null; true")); got != "" {
		t.Errorf("the host's inbox still holds %q after the binding took its copy", got)
	}

	// 2. A folder on the host, asked through the window: the door waits,
	//    the "window" answers, the folder is added there.
	rec := captureEvents(a)
	done := make(chan error, 1)
	go func() {
		_, err := a.AddWorkspaceFolder()
		done <- err
	}()
	var ask hostDirAsk
	for deadline := time.Now().Add(5 * time.Second); ask.ID == "" && time.Now().Before(deadline); {
		for _, e := range rec.all() {
			if e.Name == "screen:pickdir" && len(e.Data) == 1 {
				ask = e.Data[0].(hostDirAsk)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ask.ID == "" {
		t.Fatal("no screen:pickdir for AddWorkspaceFolder on a host")
	}
	a.AnswerHostDir(ask.ID, "/tmp")
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("AddWorkspaceFolder: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("AddWorkspaceFolder did not return after the picker answered")
	}

	// 3. A reveal names the host; a file opens as a copy fetched here.
	var opened string
	a.openDir = func(p string) error { opened = p; return nil }
	if err := a.OpenSpaceFolder("no-such-space"); err == nil {
		t.Error("a folder that does not exist was revealed")
	}
	if err := a.OpenFileExternally("plan.pdf"); err != nil {
		t.Fatalf("OpenFileExternally over ssh: %v", err)
	}
	if opened == "" {
		t.Fatal("nothing was opened")
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(opened)) })
	if got, _ := os.ReadFile(opened); string(got) != "PDF plan" {
		t.Errorf("the fetched copy reads %q", got)
	}
	t.Logf("opened a copy at %s", opened)
}
