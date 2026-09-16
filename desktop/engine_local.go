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
// tested, and an engine on the far end of an ssh tunnel (phase 3,
// engine_remote.go) is the same client with another address.
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
	"github.com/Mikedev115/Aetox/internal/engine/remote"
	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/model"
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
	// Mode is local, remote or attach — where the engine is; Host is the
	// remote host's label when Mode is remote.
	Mode string `json:"mode"`
	Host string `json:"host"`
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

// localEngine is the supervisor: the engine — this window's child, an
// engine somebody else started, or one on a host over ssh — and the wire
// to it.
type localEngine struct {
	app    *App
	client *rpc.Client
	token  string
	driver *remote.Driver

	mu       sync.Mutex
	proc     *engineProcess
	status   EngineStatus
	restarts []time.Time
	kick     chan struct{}
	stopped  chan struct{}
	// target is what spawn starts next; switched is set when it changed
	// since the last connection, which is the window's cue to reload.
	target   engineTarget
	switched bool
	// hello is what the engine said of itself when this wire was opened —
	// its OS and hostname are how an attached engine (AETOX_ENGINE_ADDR) is
	// told apart from one on this machine (elsewhere).
	hello rpc.HelloResult
}

// engineTarget is where the next engine is.
type engineTarget struct {
	mode string // modeLocal or modeRemote
	host remote.Host
}

const (
	modeLocal  = "local"
	modeRemote = "remote"
	modeAttach = "attach"
)

// engineProcess is one running engine as this window holds it: a child
// process, or a tunnel to one elsewhere.
type engineProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	network string
	address string
	pid     int
	exited  <-chan struct{}
	err     error
	// leave ends what this window holds when it is not a child of its own:
	// a tunnel closed, the engine on the host left running.
	leave func()
	// remote is the host's label when this is a tunnel.
	remote string
}

func newLocalEngine(app *App, client *rpc.Client) *localEngine {
	e := &localEngine{
		app:     app,
		client:  client,
		kick:    make(chan struct{}, 1),
		stopped: make(chan struct{}),
		status:  EngineStatus{State: engineStarting, Mode: modeLocal},
		target:  engineTarget{mode: modeLocal},
	}
	e.driver = &remote.Driver{Binary: engineBinaryFor, Log: debuglog.Msg}
	// The host the window was on when it last closed, if it chose one.
	if c, err := loadScreenConfig(); err == nil && c.ActiveHost != "" {
		if h, ok := c.host(c.ActiveHost); ok {
			e.target = engineTarget{mode: modeRemote, host: h}
			e.status.Mode, e.status.Host = modeRemote, h.Label()
		}
	}
	return e
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
	switch {
	case e.target.mode == modeRemote:
		e.status.Mode, e.status.Host = modeRemote, e.target.host.Label()
	case strings.TrimSpace(os.Getenv("AETOX_ENGINE_ADDR")) != "":
		e.status.Mode, e.status.Host = modeAttach, ""
	default:
		e.status.Mode, e.status.Host = modeLocal, ""
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
			e.mu.Lock()
			switching := e.switched
			e.mu.Unlock()
			if switching {
				// Another engine on purpose, not the same one again: the
				// card should not open with "it stopped".
				e.setStatus(engineStarting, "")
			} else {
				e.setStatus(engineRestarting, "")
			}
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
		if e.takeSwitched() {
			// Another engine, another database: every store the frontend
			// holds is about the one before. Start it over.
			e.app.reloadWindow()
		}

		// Live: watch the wire and the process, whichever gives first.
		asked := false
		for alive := true; alive; {
			select {
			case <-ctx.Done():
				e.stop(p)
				return
			case <-e.kick:
				debuglog.Msg("engine: restart asked for")
				e.stop(p)
				alive, asked = false, true
			case <-p.exited:
				if p.remote != "" {
					// The tunnel, not the engine: the host's engine is
					// most likely still there, and the next spawn finds it.
					e.setStatus(engineReconnecting, "การเชื่อมต่อกับ "+p.remote+" ขาด กำลังต่อใหม่")
				} else {
					e.setStatus(engineRestarting, fmt.Sprintf("เครื่องยนต์หยุดทำงาน (%s)", exitWord(p.err)))
				}
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
		if asked {
			// A restart the user asked for, or a switch of engine: not a
			// crash, so neither the count nor the backoff.
			continue
		}
		e.noteRestart()
		select {
		case <-time.After(e.backoff()):
		case <-ctx.Done():
			return
		}
	}
}

// takeSwitched reports, once, that the target changed since the last
// connection.
func (e *localEngine) takeSwitched() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	was := e.switched
	e.switched = false
	return was
}

// retarget points the supervisor at another engine and kicks it: the one
// running is stopped (a child) or let go of (a tunnel), and the next spawn
// is the new target.
func (e *localEngine) retarget(t engineTarget) {
	e.mu.Lock()
	e.target = t
	e.switched = true
	e.restarts = nil
	e.mu.Unlock()
	select {
	case e.kick <- struct{}{}:
	default:
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

// spawn starts one child and reads where it listens — or reaches the
// engine the target names: one on a host over ssh (spawnRemote), or, with
// AETOX_ENGINE_ADDR set, one somebody else started (attach), the manual
// road to a host that the Settings page replaced in phase 3 and that
// remains for a host the page cannot describe.
func (e *localEngine) spawn(ctx context.Context) (*engineProcess, error) {
	e.mu.Lock()
	target := e.target
	e.mu.Unlock()
	if target.mode == modeRemote {
		return e.spawnRemote(ctx, target.host)
	}
	if addr := strings.TrimSpace(os.Getenv("AETOX_ENGINE_ADDR")); addr != "" {
		return e.attach(addr)
	}
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
	// proc-detached: the engine outlives any one call on purpose — its life is
	// the screen's, held through the stdin pipe below (EOF = the screen is
	// gone, and cmd/aetox-engine exits on it), and e.stop is the one hand that
	// ends it early. A context here would tie the process to whichever caller
	// happened to spawn it, which is the coupling §248 removed.
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
	exited := make(chan struct{})
	p := &engineProcess{cmd: cmd, stdin: stdin, pid: cmd.Process.Pid, exited: exited}
	go func() {
		p.err = cmd.Wait()
		close(exited)
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
	res, err := e.client.Hello(hctx, version.Current, screen.WindowTools(nil), []string{
		rpc.FeatureDialogs, rpc.FeatureFileManager, rpc.FeatureWindowTools, rpc.FeatureProviderProxy,
	})
	if err == nil {
		e.mu.Lock()
		e.hello = res
		e.mu.Unlock()
		// Push the screen's cached ChatGPT/Codex model capabilities across to
		// the engine. On a local child this is a harmless re-read of the same
		// file; on a remote host (§248 phase 3) this is what gives the Linux
		// engine the ladder the client already fetched, bridging §261.3's gap.
		if localRoot, err := config.DataRoot(); err == nil && localRoot != "" {
			if rows, err := model.LoadResponsesModelFacts(localRoot); err == nil && len(rows) > 0 {
				go func() {
					_ = e.client.SyncResponsesModelFacts(rows)
				}()
			}
		}
	}
	return err
}

// elsewhere reports whether the engine's disk is another machine's: a host
// over ssh always; an engine attached by hand (AETOX_ENGINE_ADDR) when its
// hello says another OS — first, because two machines can share a name:
// the owner's WSL and his Windows are both "Mikedev" (14 ก.ย. 2026) — or, on
// the same OS, another hostname. A child of this process never is.
func (e *localEngine) elsewhere() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.target.mode == modeRemote {
		return true
	}
	if strings.TrimSpace(os.Getenv("AETOX_ENGINE_ADDR")) == "" {
		return false
	}
	if e.hello.OS != "" && e.hello.OS != runtime.GOOS {
		return true
	}
	if e.hello.Hostname != "" {
		here, _ := os.Hostname()
		return !strings.EqualFold(e.hello.Hostname, here)
	}
	return false
}

// attach is spawn for an engine this window did not start: AETOX_ENGINE_ADDR
// names it as `tcp:127.0.0.1:7400` or `unix:/path/to/engine.sock`, and the
// token comes from AETOX_ENGINE_TOKEN or a file named by
// AETOX_ENGINE_TOKEN_FILE. Nothing is supervised but the wire: a dropped
// connection is redialed, and an engine that is gone is a status the chip
// shows until it is back. The manual road to a host over ssh — start the
// engine there, `ssh -L` the port here, point the window at it — which phase
// 3 turns into a Settings page.
func (e *localEngine) attach(addr string) (*engineProcess, error) {
	network, address, ok := strings.Cut(addr, ":")
	if !ok || (network != "tcp" && network != "unix") || address == "" {
		return nil, fmt.Errorf("AETOX_ENGINE_ADDR=%q — want tcp:host:port or unix:/path", addr)
	}
	token := strings.TrimSpace(os.Getenv("AETOX_ENGINE_TOKEN"))
	if file := strings.TrimSpace(os.Getenv("AETOX_ENGINE_TOKEN_FILE")); token == "" && file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("AETOX_ENGINE_TOKEN_FILE: %w", err)
		}
		token = strings.TrimSpace(string(b))
	}
	if token == "" {
		return nil, errors.New("AETOX_ENGINE_ADDR is set but no token: set AETOX_ENGINE_TOKEN or AETOX_ENGINE_TOKEN_FILE")
	}
	e.token = token
	// No process of ours: exited never closes, and stop has nothing to do.
	return &engineProcess{network: network, address: address, exited: make(chan struct{})}, nil
}

// stop ends the child: stdin closed is its cue to leave on its own; a
// child that has not left in stopGrace is killed. A tunnel is closed and
// the engine behind it left running; an attached engine is not ours to
// stop at all.
func (e *localEngine) stop(p *engineProcess) {
	if p == nil {
		return
	}
	if p.leave != nil {
		p.leave()
		return
	}
	if p.cmd == nil {
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
	root := moduleRootAbove(filepath.Dir(exe))
	if root != "" {
		if goBin, lookErr := exec.LookPath("go"); lookErr == nil {
			return goBin, root, []string{"run", "./cmd/aetox-engine"}, nil
		}
	}
	return "", "", nil, missingEngine(name, filepath.Dir(exe), root == "")
}

// missingEngine is the chip's sentence when the engine binary is not where
// the app is. In a packaged install that file arrived with the app — the
// installer and the zip both carry it — so its absence has one likely
// cause, and the sentence names it: an antivirus put it in quarantine.
// That is what happened on 2026-09-15 (Defender, Trojan:Script/Wacatac.C!ml
// on the v1.7.1 engine, §294), and the person on that machine read "ไม่พบ
// aetox-engine.exe" for ten minutes without a way to act on it. The path
// to Protection history and the two verbs there are the way to act; a
// reinstall is the other. A development tree without `go` on PATH keeps
// the bare sentence, since nothing was quarantined there.
func missingEngine(name, dir string, packaged bool) error {
	if !packaged {
		return fmt.Errorf("ไม่พบ %s ข้างโปรแกรม (%s)", name, dir)
	}
	where := "โปรแกรมป้องกันไวรัส"
	if runtime.GOOS == "windows" {
		where = "Windows Security › Protection history"
	}
	return fmt.Errorf("ไม่พบ %s ข้างโปรแกรม (%s) — ไฟล์นี้ติดตั้งมาพร้อมกัน จึงน่าจะถูกโปรแกรมป้องกันไวรัสกักไว้: เปิด %s ถ้าเห็น %s ให้กด Restore แล้ว Allow on device จากนั้นกดเริ่มใหม่ หรือติดตั้ง Aetox ซ้ำ", name, dir, where, name)
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
