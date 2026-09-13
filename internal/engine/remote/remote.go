// Package remote is the screen's side of an engine on another machine
// (§248 phase 3, design doc §5): the system ssh from PATH does the
// reaching, so ~/.ssh/config, the agent and jump hosts come for free, and
// this package only decides what to say to it.
//
// On the host everything the screen puts there lives under
// ~/.aetox/server: bin/<version>/aetox-engine, the token (0600, written by
// the screen on every start), and the engine's pid, address, version and
// stdio files. The engine's own DataRoot — the database, permissions,
// hooks, memory, modes — is ~/.config/aetox on that machine and stays per
// host (§248 decision 5 for the secrets it holds; settings follow the same
// rule in v1 and Settings says so).
//
// The engine listens on a loopback TCP port of the host's choosing and the
// screen reaches it through `ssh -N -L`. A unix socket on the host with
// streamlocal forwarding was the design's first choice and remains an
// improvement for later: it could not be proven from the machine this was
// built on, and the TCP road — the token keeps other users on the host out,
// the ssh channel is the encryption — is the fallback the design named for
// exactly that case.
//
// Every script this package sends begins with `: aetox-<step>`, a POSIX
// null command with the step's name as its argument. A person reading `ps`
// on the host sees what the screen is doing, and the fake host in
// testdata/fakessh dispatches on it.
package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/proc"
)

// Host is one machine the screen can put an engine on.
type Host struct {
	// Name is the label on screen; empty means Target.
	Name string `json:"name"`
	// Target is what ssh is told: user@host, or an alias from ~/.ssh/config.
	Target string `json:"target"`
	// Root is the project folder on the host to open, if any. Chosen there
	// through the remote picker once an engine is up.
	Root string `json:"root,omitempty"`
	// Token is the running engine's admission, minted by the screen on the
	// start that produced it. The file it lives in (screen.json) is atrest-
	// wrapped by the desktop; here it is a string like any other.
	Token string `json:"token,omitempty"`
	// Version and Arch are what was last put there — the sentence on the
	// Settings row, and what a probe compares against.
	Version  string    `json:"version,omitempty"`
	Arch     string    `json:"arch,omitempty"`
	LastUsed time.Time `json:"last_used,omitempty"`
	// Spec is the host as its engine described it the last time the window
	// was there (machine.Info.Line), for the row while it is not.
	Spec string `json:"spec,omitempty"`
}

// Label is the host's name on screen.
func (h Host) Label() string {
	if strings.TrimSpace(h.Name) != "" {
		return h.Name
	}
	return h.Target
}

// targetShape is what a target may look like: a user, a host name or
// address, a port — nothing ssh would read as an option. A target that
// starts with a dash IS an option to ssh, whatever else it contains, which
// is why the first character is pinned.
var targetShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._@%:\-\[\]]*$`)

// CheckTarget refuses a target ssh would misread.
func CheckTarget(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("ยังไม่ได้ระบุเครื่อง (user@host)")
	}
	if !targetShape.MatchString(target) {
		return fmt.Errorf("%q ไม่ใช่รูป user@host หรือชื่อจาก ~/.ssh/config", target)
	}
	return nil
}

var versionShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.\-+]*$`)

// Driver runs ssh for one screen.
type Driver struct {
	// SSH is the ssh program; empty means the first on PATH.
	SSH string
	// Binary answers with the engine binary to send to a host of this
	// architecture ("amd64", "arm64"): a local file path. Called only when
	// the host does not have this version.
	Binary func(ctx context.Context, arch string, progress func(done, total int64)) (string, error)
	// Log takes one line about what happened, for the screen's log.
	Log func(format string, args ...any)
	// IdleExit is the engine's --idle-exit on the host; empty means
	// DefaultIdleExit.
	IdleExit string
}

// DefaultIdleExit is how long an engine on a host waits for a screen before
// leaving: long enough to survive a laptop's sleep, short enough that a
// forgotten host does not carry a process forever.
const DefaultIdleExit = "30m"

// Step is one stage of Connect, for the status chip.
type Step struct {
	// Name is probe, download, install, start or tunnel.
	Name string
	// Done and Total carry a download's or upload's motion; zero otherwise.
	Done, Total int64
}

// Probe is what a host says about itself.
type Probe struct {
	OS, Arch  string
	Installed bool
	Running   bool
	PID       int
	Version   string
	Address   string
}

func (d *Driver) logf(format string, args ...any) {
	if d.Log != nil {
		d.Log(format, args...)
	}
}

func (d *Driver) idleExit() string {
	if d.IdleExit != "" {
		return d.IdleExit
	}
	return DefaultIdleExit
}

// SSHPath is the program to run: AETOX_SSH, the Driver's, or PATH's.
func (d *Driver) SSHPath() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("AETOX_SSH")); custom != "" {
		return custom, nil
	}
	if d.SSH != "" {
		return d.SSH, nil
	}
	p, err := exec.LookPath("ssh")
	if err != nil {
		return "", errors.New("ไม่พบ ssh ในเครื่องนี้ — Windows 10/11 มี OpenSSH Client ให้เปิดใน Settings › Optional features")
	}
	return p, nil
}

// baseOptions is every ssh call's: no prompt of any kind (there is no
// terminal to answer one — a key or an agent is the way in), a bounded
// connect, and a host key that is remembered on first sight and refused
// when it changes. accept-new is the choice a desktop app has to make:
// asking the user to compare fingerprints in a dialog is a ceremony almost
// nobody performs, and refusing every unknown host would make the first
// connection impossible from inside the app.
//
// The log level is ssh's default, not ERROR: Windows' OpenSSH reports a
// refused connection at INFO, so under ERROR a host that is simply off
// answered "exit status 255" and nothing else (the first morning after the
// first real host, 2026-09-13). The one line the default adds — the host
// key remembered on first sight — is worth having in the log anyway.
func baseOptions() []string {
	return []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
	}
}

// quote is POSIX single-quoting: safe for any byte but NUL.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// wrap is the command line ssh hands the host's login shell: `sh -c` with
// the script single-quoted, so it reads the same under bash, zsh, dash and
// fish alike — only sh has to be there, which POSIX promises.
func wrap(script string) string {
	return "sh -c " + quote(script)
}

// serverDir is the script-side name of the screen's directory on the host.
const serverDir = `d="$HOME/.aetox/server"`

func probeScript(version string) string {
	return `: aetox-probe; ` + serverDir + `; uname -sm; ` +
		`if [ -x "$d/bin/` + version + `/aetox-engine" ]; then echo installed=yes; else echo installed=no; fi; ` +
		`if [ -r "$d/engine.pid" ] && kill -0 "$(cat "$d/engine.pid")" 2>/dev/null; then ` +
		`echo running=yes; echo "pid=$(cat "$d/engine.pid")"; echo "version=$(cat "$d/engine.version" 2>/dev/null)"; echo "address=$(cat "$d/engine.addr" 2>/dev/null)"; ` +
		`else echo running=no; fi`
}

func installScript(version string) string {
	return `: aetox-install; set -e; d="$HOME/.aetox/server/bin/` + version + `"; mkdir -p "$d"; ` +
		`cat > "$d/aetox-engine.tmp"; chmod 755 "$d/aetox-engine.tmp"; mv -f "$d/aetox-engine.tmp" "$d/aetox-engine"; ` +
		`"$d/aetox-engine" version`
}

// startScript starts the engine detached — nohup, stdio on files, stdin
// from /dev/null — so the ssh session that started it can end. The token
// arrives on stdin and lands in a 0600 file before anything else runs; an
// engine already recorded as running is stopped first, because a start is
// asked for only when the one there is the wrong version or has a token
// the screen no longer knows.
func startScript(version, root, idle string) string {
	rootArg := ""
	if root != "" {
		rootArg = ` --root ` + quote(root)
	}
	return `: aetox-start; set -e; umask 077; ` + serverDir + `; mkdir -p "$d"; ` +
		`if [ -r "$d/engine.pid" ] && kill -0 "$(cat "$d/engine.pid")" 2>/dev/null; then kill "$(cat "$d/engine.pid")" 2>/dev/null || true; sleep 1; fi; ` +
		`cat > "$d/token"; rm -f "$d/engine.out" "$d/engine.addr" "$d/engine.pid"; cd "$HOME"; ` +
		`nohup "$d/bin/` + version + `/aetox-engine" serve --tcp 127.0.0.1:0 --token-file "$d/token" --idle-exit ` + idle + rootArg +
		` > "$d/engine.out" 2> "$d/engine.err" < /dev/null & pid=$!; ` +
		`i=0; while [ ! -s "$d/engine.out" ]; do ` +
		`if ! kill -0 "$pid" 2>/dev/null; then echo "engine exited:" >&2; cat "$d/engine.err" >&2; exit 1; fi; ` +
		`i=$((i+1)); if [ "$i" -gt 200 ]; then echo "engine did not listen in 20s" >&2; exit 1; fi; sleep 0.1; done; ` +
		`head -n 1 "$d/engine.out" > "$d/engine.addr"; echo "$pid" > "$d/engine.pid"; echo ` + version + ` > "$d/engine.version"; ` +
		`cat "$d/engine.addr"`
}

const stopScript = `: aetox-stop; ` + serverDir + `; if [ -r "$d/engine.pid" ]; then kill "$(cat "$d/engine.pid")" 2>/dev/null; rm -f "$d/engine.pid"; fi; true`

const tailScript = `: aetox-tail; ` + serverDir + `; echo "== engine.err"; tail -n 40 "$d/engine.err" 2>/dev/null; ` +
	`f=$(ls -t "${XDG_CONFIG_HOME:-$HOME/.config}/aetox/logs"/aetox-*.log 2>/dev/null | head -n 1); ` +
	`if [ -n "$f" ]; then echo "== $f"; tail -n 120 "$f"; fi; true`

// run is one ssh command on the host: the script, stdin if any, stdout
// back. A failure carries ssh's own words — "Permission denied (publickey)"
// is the sentence the user needs, not "exit status 255".
func (d *Driver) run(ctx context.Context, h Host, script string, stdin io.Reader) ([]byte, error) {
	if err := CheckTarget(h.Target); err != nil {
		return nil, err
	}
	ssh, err := d.SSHPath()
	if err != nil {
		return nil, err
	}
	args := append(baseOptions(), h.Target, wrap(script))
	cmd := exec.CommandContext(ctx, ssh, args...)
	proc.HideConsole(cmd)
	cmd.Stdin = stdin
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("ssh %s: %s", h.Target, sshWords(errb.String(), err))
	}
	return out.Bytes(), nil
}

// sshWords is the last thing ssh or the host said — or, when they said
// nothing, the exit status with what it usually means: 255 is ssh's own
// "could not get there", and a person reading the card needs the next
// thing to try, not the number.
func sshWords(stderr string, err error) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	if err != nil && strings.HasSuffix(err.Error(), "255") {
		return "ติดต่อเครื่องไม่ได้ (ssh ออกด้วยรหัส 255 โดยไม่บอกเหตุ) — เครื่องปิดอยู่ sshd ไม่ได้รัน หรือพอร์ตไม่ตรง; ลอง ssh ไปเองในเทอร์มินัลจะเห็นสาเหตุ"
	}
	if err != nil {
		return err.Error()
	}
	return "ssh ออกโดยไม่บอกเหตุ"
}

// Probe asks the host what it is and what is there.
func (d *Driver) Probe(ctx context.Context, h Host, version string) (Probe, error) {
	if !versionShape.MatchString(version) {
		return Probe{}, fmt.Errorf("version %q", version)
	}
	out, err := d.run(ctx, h, probeScript(version), nil)
	if err != nil {
		return Probe{}, err
	}
	return parseProbe(string(out))
}

func parseProbe(out string) (Probe, error) {
	var p Probe
	lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return p, errors.New("the host answered nothing")
	}
	sys := strings.Fields(lines[0])
	if len(sys) < 2 {
		return p, fmt.Errorf("uname said %q", lines[0])
	}
	p.OS = strings.ToLower(sys[0])
	switch sys[1] {
	case "x86_64", "amd64":
		p.Arch = "amd64"
	case "aarch64", "arm64":
		p.Arch = "arm64"
	default:
		p.Arch = sys[1]
	}
	for _, l := range lines[1:] {
		k, v, ok := strings.Cut(strings.TrimSpace(l), "=")
		if !ok {
			continue
		}
		switch k {
		case "installed":
			p.Installed = v == "yes"
		case "running":
			p.Running = v == "yes"
		case "pid":
			p.PID, _ = strconv.Atoi(v)
		case "version":
			p.Version = v
		case "address":
			p.Address = addressOf(v)
		}
	}
	return p, nil
}

// addressOf is the address in the engine's listening line, which is JSON
// — read with a string cut rather than a decoder so a stray line in the
// file cannot fail the probe.
func addressOf(line string) string {
	_, rest, ok := strings.Cut(line, `"address":"`)
	if !ok {
		return ""
	}
	addr, _, ok := strings.Cut(rest, `"`)
	if !ok {
		return ""
	}
	return addr
}

// Install streams the engine binary at path up to the host and checks it
// runs there.
func (d *Driver) Install(ctx context.Context, h Host, version, path string, progress func(done, total int64)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	var rd io.Reader = f
	if progress != nil {
		rd = &progressReader{r: f, total: st.Size(), report: progress}
	}
	out, err := d.run(ctx, h, installScript(version), rd)
	if err != nil {
		return fmt.Errorf("ติดตั้งเครื่องยนต์บน %s ไม่สำเร็จ: %w", h.Label(), err)
	}
	if got := strings.TrimSpace(string(out)); got != version {
		return fmt.Errorf("เครื่องยนต์ที่ส่งไป %s บอกเวอร์ชัน %q ไม่ใช่ %s", h.Label(), got, version)
	}
	d.logf("remote: installed engine %s on %s", version, h.Target)
	return nil
}

type progressReader struct {
	r      io.Reader
	done   int64
	total  int64
	report func(done, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.done += int64(n)
		p.report(p.done, p.total)
	}
	return n, err
}

// Start starts the engine on the host with this token and reports where it
// listens there (a loopback address on the host) and its pid.
func (d *Driver) Start(ctx context.Context, h Host, version, token string) (address string, pid int, err error) {
	if !versionShape.MatchString(version) {
		return "", 0, fmt.Errorf("version %q", version)
	}
	out, err := d.run(ctx, h, startScript(version, h.Root, d.idleExit()), strings.NewReader(token+"\n"))
	if err != nil {
		return "", 0, fmt.Errorf("เริ่มเครื่องยนต์บน %s ไม่สำเร็จ: %w", h.Label(), err)
	}
	line := strings.TrimSpace(string(out))
	address = addressOf(line)
	if address == "" {
		return "", 0, fmt.Errorf("เครื่องยนต์บน %s ไม่บอกว่าฟังที่ไหน (%q)", h.Label(), line)
	}
	if _, rest, ok := strings.Cut(line, `"pid":`); ok {
		n, _, _ := strings.Cut(rest, ",")
		pid, _ = strconv.Atoi(strings.TrimRight(strings.TrimSpace(n), "}"))
	}
	d.logf("remote: engine %s started on %s at %s (pid %d)", version, h.Target, address, pid)
	return address, pid, nil
}

// Stop ends the engine on the host. Not called when a window closes — the
// engine's idle clock does that — only when the user asks, or before a
// start replaces it.
func (d *Driver) Stop(ctx context.Context, h Host) error {
	_, err := d.run(ctx, h, stopScript, nil)
	return err
}

// Tail is the engine's stderr and the newest of its logs on the host, for
// the Settings page.
func (d *Driver) Tail(ctx context.Context, h Host) (string, error) {
	out, err := d.run(ctx, h, tailScript, nil)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Tunnel is one `ssh -N -L`: a loopback port here that is the engine's
// loopback port there, for as long as the process lives.
type Tunnel struct {
	// Local is the address the screen dials: 127.0.0.1:<port>.
	Local string
	cmd   *exec.Cmd
	// exited closes when ssh is gone; err is why.
	exited chan struct{}
	err    error
	stderr bytes.Buffer
	once   sync.Once
}

// Exited closes when the tunnel is gone, however it went.
func (t *Tunnel) Exited() <-chan struct{} { return t.exited }

// Err is why it went, after Exited.
func (t *Tunnel) Err() error {
	select {
	case <-t.exited:
		if t.err == nil {
			return nil
		}
		return fmt.Errorf("%s", sshWords(t.stderr.String(), t.err))
	default:
		return nil
	}
}

// Close ends the tunnel. The engine on the host is untouched.
func (t *Tunnel) Close() {
	t.once.Do(func() {
		if t.cmd != nil && t.cmd.Process != nil {
			_ = t.cmd.Process.Kill()
		}
	})
	<-t.exited
}

// tunnelWait is how long a tunnel may take to answer on its local port:
// key exchange and auth to a far host, with a jump host in between.
var tunnelWait = 25 * time.Second

// Tunnel opens the tunnel to remoteAddr, a loopback address on the host.
func (d *Driver) Tunnel(ctx context.Context, h Host, remoteAddr string) (*Tunnel, error) {
	if err := CheckTarget(h.Target); err != nil {
		return nil, err
	}
	ssh, err := d.SSHPath()
	if err != nil {
		return nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	local := "127.0.0.1:" + strconv.Itoa(port)
	args := append(baseOptions(),
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=15",
		"-o", "ServerAliveCountMax=3",
		"-L", local+":"+remoteAddr,
		h.Target,
	)
	// Not CommandContext: the tunnel outlives the call that opened it, and
	// its end is Close, or ssh's own.
	cmd := exec.Command(ssh, args...)
	proc.HideConsole(cmd)
	t := &Tunnel{Local: local, cmd: cmd, exited: make(chan struct{})}
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = &t.stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("เปิด ssh ไม่ได้: %w", err)
	}
	go func() {
		t.err = cmd.Wait()
		close(t.exited)
	}()
	// ssh listens on the local port once it is in; until then the dial is
	// refused. A process that leaves before that is a failure with words.
	deadline := time.Now().Add(tunnelWait)
	for {
		c, dialErr := net.DialTimeout("tcp", local, time.Second)
		if dialErr == nil {
			_ = c.Close()
			d.logf("remote: tunnel %s -> %s on %s (ssh pid %d)", local, remoteAddr, h.Target, cmd.Process.Pid)
			return t, nil
		}
		select {
		case <-t.exited:
			return nil, fmt.Errorf("ssh %s: %s", h.Target, sshWords(t.stderr.String(), t.err))
		case <-ctx.Done():
			t.Close()
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Close()
			return nil, fmt.Errorf("ssh %s: อุโมงค์ไม่ตอบใน %s", h.Target, tunnelWait)
		}
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

// Connect is the whole road: probe, install if this version is not there,
// start unless an engine of this version is running with a token we hold,
// tunnel. It writes what it learned back into h — token, version, arch,
// last use — for the caller to keep.
func (d *Driver) Connect(ctx context.Context, h *Host, version string, report func(Step)) (*Tunnel, error) {
	if report == nil {
		report = func(Step) {}
	}
	report(Step{Name: "probe"})
	p, err := d.Probe(ctx, *h, version)
	if err != nil {
		return nil, err
	}
	if p.OS != "linux" {
		return nil, fmt.Errorf("%s เป็น %s — รุ่นนี้รองรับเฉพาะ host ที่เป็น Linux", h.Label(), p.OS)
	}
	if p.Arch != "amd64" && p.Arch != "arm64" {
		return nil, fmt.Errorf("%s เป็น %s — มีเครื่องยนต์ให้เฉพาะ x86_64 กับ aarch64", h.Label(), p.Arch)
	}
	h.Arch = p.Arch

	address := p.Address
	reuse := p.Running && p.Version == version && h.Token != "" && address != ""
	if !reuse {
		if !p.Installed {
			if d.Binary == nil {
				return nil, errors.New("ไม่มีเครื่องยนต์สำหรับ Linux ให้ส่งไป")
			}
			report(Step{Name: "download"})
			path, err := d.Binary(ctx, p.Arch, func(done, total int64) {
				report(Step{Name: "download", Done: done, Total: total})
			})
			if err != nil {
				return nil, err
			}
			report(Step{Name: "install"})
			if err := d.Install(ctx, *h, version, path, func(done, total int64) {
				report(Step{Name: "install", Done: done, Total: total})
			}); err != nil {
				return nil, err
			}
		}
		token, err := rpc.NewToken()
		if err != nil {
			return nil, err
		}
		report(Step{Name: "start"})
		address, _, err = d.Start(ctx, *h, version, token)
		if err != nil {
			return nil, err
		}
		h.Token = token
		h.Version = version
	} else {
		d.logf("remote: engine %s already running on %s at %s", version, h.Target, address)
	}
	report(Step{Name: "tunnel"})
	t, err := d.Tunnel(ctx, *h, address)
	if err != nil {
		return nil, err
	}
	h.LastUsed = time.Now()
	return t, nil
}
