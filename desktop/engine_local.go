package main

// The engine as this window's child (§248 phase 2, design doc §4).
//
// The screen starts `aetox-engine serve` beside it, hands it a token on
// stdin — never on the command line — and keeps stdin open for as long as
// it lives, which is how the engine knows when the screen is gone. The
// socket is a unix socket under DataRoot on both systems; when that cannot
// be had, a loopback TCP port the engine reports on its stdout. Decision 2
// of §248 is that this is the ONLY way the screen reaches an engine: there
// is no in-process path, so every day of use is a day of the socket being
// tested, and an engine on the far end of an ssh tunnel (phase 3) is the
// same client with another address.
//
// What the supervisor here promises: the engine is running or the status
// chip says why not; a crash restarts it, with a backoff, and the third
// crash in a minute stops trying and says so; a wire that dropped while
// the process lives is redialed before the process is replaced; and every
// binding the frontend calls meanwhile waits for the wire (rpc.Client) so a
// restart is a pause, not a screen of errors.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/version"
)

// EngineStatus is what the status chip draws, and the engine:status event.
type EngineStatus struct {
	// State is starting, connected, reconnecting, restarting or failed.
	State string `json:"state"`
	// Detail is the sentence for a state that needs one: why it failed,
	// what it is waiting for.
	Detail string `json:"detail"`
	// Restarts is how many times the engine has been started again this
	// session — a number the person can see grow, which is the point.
	Restarts int `json:"restarts"`
	// PID and Address are the running engine's, for the log and the
	// curious; empty while there is none.
	PID     int    `json:"pid"`
	Address string `json:"address"`
}

const (
	engineStarting     = "starting"
	engineConnected    = "connected"
	engineReconnecting = "reconnecting"
	engineRestarting   = "restarting"
	engineFailed       = "failed"
)

// Bounds. maxRestartsPerMinute is the plan's number: a third crash inside
// a minute is a broken engine, and restarting it a fourth time would only
// spend the user's minute on the same crash.
const (
	maxRestartsPerMinute = 3
	spawnLineWait        = 20 * time.Second
	dialRetryWindow      = 10 * time.Second
	stopGrace            = 5 * time.Second
)

// localEngine is the child and the wire to it.
type localEngine struct {
	app    *App
	client *rpc.Client
	token  string

	mu       sync.Mutex
	proc     *engineProcess
	status   EngineStatus
	restarts []time.Time
	kick     chan struct{}
	stopped  chan struct{}
}

// engineProcess is one running child.
type engineProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	network string
	address string
	pid     int
	exited  chan struct{}
	err     error
}

func newLocalEngine(app *App, client *rpc.Client) *localEngine {
	return &localEngine{
		app:     app,
		client:  client,
		kick:    make(chan struct{}, 1),
		stopped: make(chan struct{}),
		status:  EngineStatus{State: engineStarting},
	}
}

// endpoint is where the engine listens now — what the file proxy asks per
// request (rpc.Endpoint).
func (e *localEngine) endpoint() (network, address, token string, ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.proc == nil {
		return "", "", "", false
	}
	return e.proc.network, e.proc.address, e.token, true
}

// EngineStatus is the chip's binding: the state now, for a frontend that
// mounted after the event went by.
func (a *App) EngineStatus() EngineStatus {
	if a.engine == nil {
		return EngineStatus{State: engineConnected}
	}
	a.engine.mu.Lock()
	defer a.engine.mu.Unlock()
	return a.engine.status
}

// RestartEngine is the chip's button: try again after a failure, or start
// over on demand. A supervisor that is between attempts takes it at once;
// one that has given up takes it as permission.
func (a *App) RestartEngine() {
	if a.engine == nil {
		return
	}
	select {
	case a.engine.kick <- struct{}{}:
	default:
	}
}

func (e *localEngine) setStatus(state, detail string) {
	e.mu.Lock()
	e.status.State, e.status.Detail = state, detail
	if e.proc != nil {
		e.status.PID, e.status.Address = e.proc.pid, e.proc.address
	} else {
		e.status.PID, e.status.Address = 0, ""
	}
	st := e.status
	e.mu.Unlock()
	debuglog.Msg("engine: %s %s", state, detail)
	e.app.emitEvent("engine:status", st)
}

// run is the supervisor's life, from startup until the window's context
// ends. It returns once the child is gone.
func (e *localEngine) run(ctx context.Context) {
	defer close(e.stopped)
	first := true
	for {
		if !first {
			if !e.mayRestart() {
				e.setStatus(engineFailed, "เครื่องยนต์ล้มสามครั้งในหนึ่งนาที — กดเริ่มใหม่เมื่อพร้อม")
				if !e.waitForKick(ctx) {
					return
				}
				e.mu.Lock()
				e.restarts = nil
				e.mu.Unlock()
			}
			e.setStatus(engineRestarting, "")
		}
		first = false

		p, err := e.spawn(ctx)
		if err != nil {
			e.setStatus(engineFailed, err.Error())
			if !e.waitForKick(ctx) {
				return
			}
			continue
		}
		e.mu.Lock()
		e.proc = p
		e.mu.Unlock()

		if err := e.connect(ctx, p); err != nil {
			e.stop(p)
			e.setStatus(engineFailed, "ต่อเครื่องยนต์ไม่ได้: "+err.Error())
			e.noteRestart()
			select {
			case <-time.After(e.backoff()):
			case <-ctx.Done():
				return
			}
			continue
		}
		e.setStatus(engineConnected, "")

		// Live: watch the wire and the process, whichever gives first.
		for alive := true; alive; {
			select {
			case <-ctx.Done():
				e.stop(p)
				return
			case <-e.kick:
				debuglog.Msg("engine: restart asked for")
				e.stop(p)
				alive = false
			case <-p.exited:
				e.setStatus(engineRestarting, fmt.Sprintf("เครื่องยนต์หยุดทำงาน (%s)", exitWord(p.err)))
				alive = false
			case <-e.client.Done():
				// The wire dropped but the process may live: redial before
				// replacing it, which keeps the turn in flight and the
				// sessions the engine holds.
				e.setStatus(engineReconnecting, "")
				if err := e.connect(ctx, p); err != nil {
					debuglog.Msg("engine: redial failed: %v", err)
					e.stop(p)
					alive = false
					break
				}
				e.setStatus(engineConnected, "")
			}
		}
		e.mu.Lock()
		e.proc = nil
		e.mu.Unlock()
		e.noteRestart()
		select {
		case <-time.After(e.backoff()):
		case <-ctx.Done():
			return
		}
	}
}

func (e *localEngine) waitForKick(ctx context.Context) bool {
	select {
	case <-e.kick:
		return true
	case <-ctx.Done():
		return false
	}
}

func (e *localEngine) noteRestart() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.restarts = append(e.restarts, time.Now())
	e.status.Restarts++
}

// mayRestart applies the three-per-minute rule.
func (e *localEngine) mayRestart() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := time.Now().Add(-time.Minute)
	recent := e.restarts[:0]
	for _, t := range e.restarts {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	e.restarts = recent
	return len(recent) < maxRestartsPerMinute
}

// backoff grows with the restarts of the last minute: 1s, 2s, 4s.
func (e *localEngine) backoff() time.Duration {
	e.mu.Lock()
	n := len(e.restarts)
	e.mu.Unlock()
	if n < 1 {
		n = 1
	}
	if n > 3 {
		n = 3
	}
	return time.Duration(1<<(n-1)) * time.Second
}

// spawn starts one child and reads where it listens.
func (e *localEngine) spawn(ctx context.Context) (*engineProcess, error) {
	bin, dir, prefix, err := engineCommand()
	if err != nil {
		return nil, err
	}
	dataRoot, err := config.DataRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return nil, err
	}
	// A unix socket first, on both systems (rpc.Listen says why); a child
	// that cannot listen on it says so and exits, and the second try asks
	// for a loopback port instead.
	socket := filepath.Join(dataRoot, fmt.Sprintf("engine-%d.sock", os.Getpid()))
	p, err := e.start(ctx, bin, dir, append(prefix, "serve", "--socket", socket, "--token-stdin"))
	if err == nil {
		return p, nil
	}
	debuglog.Msg("engine: unix socket start failed (%v); trying tcp", err)
	return e.start(ctx, bin, dir, append(prefix, "serve", "--tcp", "127.0.0.1:0", "--token-stdin"))
}

func (e *localEngine) start(ctx context.Context, bin, dir string, args []string) (*engineProcess, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	proc.HideConsole(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("เริ่ม %s ไม่ได้: %w", filepath.Base(bin), err)
	}
	p := &engineProcess{cmd: cmd, stdin: stdin, pid: cmd.Process.Pid, exited: make(chan struct{})}
	go func() {
		p.err = cmd.Wait()
		close(p.exited)
	}()
	// The engine's stderr is ours to keep: its own log is under DataRoot,
	// this is only what it says on the way in or out.
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			debuglog.Msg("engine[%d]: %s", p.pid, sc.Text())
		}
	}()
	if _, err := io.WriteString(stdin, e.token+"\n"); err != nil {
		e.stop(p)
		return nil, fmt.Errorf("handing the engine its token: %w", err)
	}
	// The one line on its stdout.
	lineCh := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		lineCh <- line
		// Anything after the line is not ours to read; drain so the child
		// never blocks on a full pipe.
		_, _ = io.Copy(io.Discard, stdout)
	}()
	var where struct {
		Network string `json:"network"`
		Address string `json:"address"`
	}
	select {
	case line := <-lineCh:
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &where); err != nil || where.Address == "" {
			e.stop(p)
			return nil, fmt.Errorf("the engine did not say where it listens (%q)", strings.TrimSpace(line))
		}
	case <-p.exited:
		return nil, fmt.Errorf("the engine exited before listening (%s)", exitWord(p.err))
	case <-time.After(spawnLineWait):
		e.stop(p)
		return nil, errors.New("the engine took too long to start")
	case <-ctx.Done():
		e.stop(p)
		return nil, ctx.Err()
	}
	p.network, p.address = where.Network, where.Address
	return p, nil
}

// connect dials the running child, retrying for a while: right after start
// the listener is up (it said so), but a redial after a dropped wire may
// meet a socket still being reopened.
func (e *localEngine) connect(ctx context.Context, p *engineProcess) error {
	deadline := time.Now().Add(dialRetryWindow)
	var last error
	for {
		dctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := e.client.Connect(dctx, p.network, p.address, e.token)
		cancel()
		if err == nil {
			break
		}
		last = err
		select {
		case <-p.exited:
			return fmt.Errorf("the engine exited while being dialed (%s)", exitWord(p.err))
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return last
		}
	}
	hctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	screen := appScreen{e.app}
	_, err := e.client.Hello(hctx, version.Current, screen.WindowTools(nil), []string{
		rpc.FeatureDialogs, rpc.FeatureFileManager, rpc.FeatureWindowTools, rpc.FeatureProviderProxy,
	})
	return err
}

// stop ends the child: stdin closed is its cue to leave on its own; a
// child that has not left in stopGrace is killed.
func (e *localEngine) stop(p *engineProcess) {
	if p == nil {
		return
	}
	_ = p.stdin.Close()
	select {
	case <-p.exited:
	case <-time.After(stopGrace):
		_ = p.cmd.Process.Kill()
		<-p.exited
	}
}

// shutdown is the window closing: the child told to leave, and waited for.
func (e *localEngine) shutdown() {
	e.mu.Lock()
	p := e.proc
	e.mu.Unlock()
	e.stop(p)
}

func exitWord(err error) string {
	if err == nil {
		return "exit 0"
	}
	return err.Error()
}

// engineCommand is how to start the engine on this machine: the binary
// beside this one, or — in a development tree, where nothing has been
// packaged — `go run` on the source, so `wails dev` needs no extra step.
// AETOX_ENGINE names a binary outright, for a build of the engine alone.
func engineCommand() (bin, dir string, prefix []string, err error) {
	if custom := strings.TrimSpace(os.Getenv("AETOX_ENGINE")); custom != "" {
		return custom, "", nil, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", "", nil, err
	}
	name := "aetox-engine"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	beside := filepath.Join(filepath.Dir(exe), name)
	if _, statErr := os.Stat(beside); statErr == nil {
		return beside, "", nil, nil
	}
	if root := moduleRootAbove(filepath.Dir(exe)); root != "" {
		if goBin, lookErr := exec.LookPath("go"); lookErr == nil {
			return goBin, root, []string{"run", "./cmd/aetox-engine"}, nil
		}
	}
	return "", "", nil, fmt.Errorf("ไม่พบ %s ข้างโปรแกรม (%s)", name, filepath.Dir(exe))
}

// moduleRootAbove walks up from dir to the folder holding this module's
// go.mod, "" when there is none — a packaged install has none.
func moduleRootAbove(dir string) string {
	for i := 0; i < 8; i++ {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(b), "module github.com/Mikedev115/Aetox") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}
