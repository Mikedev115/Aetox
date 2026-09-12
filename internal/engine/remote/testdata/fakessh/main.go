// Command fakessh is a host that speaks the scripts internal/engine/remote
// sends, without a network or a Linux machine: the tests build it and hand
// it to the Driver as its ssh. It parses ssh's argv the way ssh would,
// dispatches on the `: aetox-<step>` marker each script begins with, and
// keeps its state under AETOX_FAKE_HOME the way the real host keeps it
// under $HOME — including a real engine process, started from the bytes
// that arrived on stdin, and a real TCP forwarder for `-N -L`.
//
// What it does not do is run the scripts. The scripts are POSIX sh and
// this runs on Windows; their faithfulness to a Linux host is what
// TestRemoteSmoke (AETOX_REMOTE_SMOKE=user@host) and the CI Linux job
// check. What this checks is everything on the screen's side of the wire:
// the sequence, the payloads on stdin, the parsing of what comes back, and
// the tunnel's life.
//
// Knobs, all environment:
//
//	AETOX_FAKE_HOME     the host's $HOME (required)
//	AETOX_FAKE_UNAME    what `uname -sm` answers (default "Linux x86_64")
//	AETOX_FAKE_FAIL     auth | hostkey — fail every call the way ssh would
//	AETOX_FAKE_LOG      a file that gets one line per call: the step name
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(255)
	}
}

func run(args []string) error {
	home := os.Getenv("AETOX_FAKE_HOME")
	if home == "" {
		return fmt.Errorf("AETOX_FAKE_HOME is not set")
	}
	switch os.Getenv("AETOX_FAKE_FAIL") {
	case "auth":
		return fmt.Errorf("user@box: Permission denied (publickey).")
	case "hostkey":
		return fmt.Errorf("Host key verification failed.")
	}

	var forward string
	var noCommand bool
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-o" || a == "-p" || a == "-i" || a == "-F":
			i++ // option with a value
		case a == "-L":
			i++
			if i < len(args) {
				forward = args[i]
			}
		case a == "-N":
			noCommand = true
		case strings.HasPrefix(a, "-"):
			// a flag without a value
		default:
			rest = args[i:]
			i = len(args)
		}
	}
	if len(rest) == 0 {
		return fmt.Errorf("no target")
	}
	target := rest[0]
	if strings.HasPrefix(target, "-") {
		return fmt.Errorf("target %q looks like an option", target)
	}
	if noCommand {
		if forward == "" {
			return fmt.Errorf("-N without -L")
		}
		logStep(home, "tunnel")
		return tunnel(forward)
	}
	// ssh joins the rest with spaces; the driver sends `sh -c '<script>'`.
	line := strings.Join(rest[1:], " ")
	script, ok := unwrap(line)
	if !ok {
		return fmt.Errorf("not `sh -c '<script>'`: %q", line)
	}
	step := stepOf(script)
	logStep(home, step)
	h := &host{home: home, server: filepath.Join(home, ".aetox", "server")}
	switch step {
	case "probe":
		return h.probe(script)
	case "install":
		return h.install(script)
	case "start":
		return h.start(script)
	case "stop":
		return h.stop()
	case "tail":
		return h.tail()
	default:
		return fmt.Errorf("unknown step %q", step)
	}
}

func logStep(home, step string) {
	if p := os.Getenv("AETOX_FAKE_LOG"); p != "" {
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintln(f, step)
			f.Close()
		}
	}
}

// unwrap undoes `sh -c '<script>'` with POSIX single quoting.
func unwrap(line string) (string, bool) {
	if !strings.HasPrefix(line, "sh -c '") || !strings.HasSuffix(line, "'") {
		return "", false
	}
	inner := line[len("sh -c '") : len(line)-1]
	return strings.ReplaceAll(inner, `'\''`, `'`), true
}

func stepOf(script string) string {
	first, _, _ := strings.Cut(script, ";")
	return strings.TrimPrefix(strings.TrimSpace(first), ": aetox-")
}

type host struct {
	home   string
	server string
}

var versionInScript = regexp.MustCompile(`/bin/([0-9A-Za-z.+-]+)`)

func (h *host) engineBinary(script string) (version, path string, err error) {
	m := versionInScript.FindStringSubmatch(script)
	if m == nil {
		return "", "", fmt.Errorf("no version in %q", script)
	}
	version = m[1]
	path = filepath.Join(h.server, "bin", version, "aetox-engine")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	return version, path, nil
}

// running is the engine the pid file names, if its address still answers.
func (h *host) running() (pid int, version, address string, ok bool) {
	b, err := os.ReadFile(filepath.Join(h.server, "engine.pid"))
	if err != nil {
		return 0, "", "", false
	}
	pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
	v, _ := os.ReadFile(filepath.Join(h.server, "engine.version"))
	line, _ := os.ReadFile(filepath.Join(h.server, "engine.addr"))
	address = addressOf(string(line))
	if address == "" {
		return pid, "", "", false
	}
	c, err := net.DialTimeout("tcp", address, 300*time.Millisecond)
	if err != nil {
		return pid, "", "", false
	}
	_ = c.Close()
	return pid, strings.TrimSpace(string(v)), address, true
}

func addressOf(line string) string {
	var l struct {
		Address string `json:"address"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(line)), &l)
	return l.Address
}

func (h *host) probe(script string) error {
	uname := os.Getenv("AETOX_FAKE_UNAME")
	if uname == "" {
		uname = "Linux x86_64"
	}
	fmt.Println(uname)
	_, bin, err := h.engineBinary(script)
	if err != nil {
		return err
	}
	if _, err := os.Stat(bin); err == nil {
		fmt.Println("installed=yes")
	} else {
		fmt.Println("installed=no")
	}
	if pid, version, _, ok := h.running(); ok {
		fmt.Println("running=yes")
		fmt.Printf("pid=%d\n", pid)
		fmt.Printf("version=%s\n", version)
		line, _ := os.ReadFile(filepath.Join(h.server, "engine.addr"))
		fmt.Printf("address=%s\n", strings.TrimSpace(string(line)))
	} else {
		fmt.Println("running=no")
	}
	return nil
}

func (h *host) install(script string) error {
	_, bin, err := h.engineBinary(script)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		return err
	}
	tmp := bin + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, os.Stdin); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, bin); err != nil {
		return err
	}
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		return fmt.Errorf("the installed engine does not run: %v", err)
	}
	fmt.Print(string(out))
	return nil
}

var rootInScript = regexp.MustCompile(`--root '((?:[^']|'\\'')*)'`)

func (h *host) start(script string) error {
	version, bin, err := h.engineBinary(script)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(h.server, 0o700); err != nil {
		return err
	}
	if pid, _, _, ok := h.running(); ok {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
		}
		time.Sleep(200 * time.Millisecond)
	}
	token, _ := io.ReadAll(os.Stdin)
	tokenFile := filepath.Join(h.server, "token")
	if err := os.WriteFile(tokenFile, token, 0o600); err != nil {
		return err
	}
	for _, name := range []string{"engine.out", "engine.addr", "engine.pid"} {
		_ = os.Remove(filepath.Join(h.server, name))
	}
	args := []string{"serve", "--tcp", "127.0.0.1:0", "--token-file", tokenFile}
	if m := rootInScript.FindStringSubmatch(script); m != nil {
		args = append(args, "--root", strings.ReplaceAll(m[1], `'\''`, `'`))
	}
	if _, rest, ok := strings.Cut(script, "--idle-exit "); ok {
		idle, _, _ := strings.Cut(rest, " ")
		args = append(args, "--idle-exit", idle)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = h.home
	cmd.Env = append(os.Environ(), "AETOX_DATA_ROOT="+filepath.Join(h.home, ".config", "aetox"))
	errFile, err := os.Create(filepath.Join(h.server, "engine.err"))
	if err != nil {
		return err
	}
	defer errFile.Close()
	cmd.Stderr = errFile
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	lineCh := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		lineCh <- line
		_, _ = io.Copy(io.Discard, stdout)
	}()
	var line string
	select {
	case line = <-lineCh:
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		return fmt.Errorf("engine did not listen in 20s")
	}
	if addressOf(line) == "" {
		_ = cmd.Process.Kill()
		return fmt.Errorf("engine exited: %s", strings.TrimSpace(line))
	}
	// The engine outlives this process, as it does on a host: nothing
	// waits for it, and the pid file is how the next call finds it.
	_ = os.WriteFile(filepath.Join(h.server, "engine.out"), []byte(line), 0o600)
	_ = os.WriteFile(filepath.Join(h.server, "engine.addr"), []byte(line), 0o600)
	_ = os.WriteFile(filepath.Join(h.server, "engine.pid"), []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o600)
	_ = os.WriteFile(filepath.Join(h.server, "engine.version"), []byte(version+"\n"), 0o600)
	go func() { _ = cmd.Wait() }()
	fmt.Print(line)
	return nil
}

func (h *host) stop() error {
	if pid, _, _, ok := h.running(); ok {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
		}
	}
	_ = os.Remove(filepath.Join(h.server, "engine.pid"))
	return nil
}

func (h *host) tail() error {
	fmt.Println("== engine.err")
	b, _ := os.ReadFile(filepath.Join(h.server, "engine.err"))
	fmt.Print(string(b))
	return nil
}

// tunnel is `-L local:remote` for as long as this process lives.
func tunnel(spec string) error {
	// 127.0.0.1:L:127.0.0.1:R
	parts := strings.Split(spec, ":")
	if len(parts) != 4 {
		return fmt.Errorf("forward spec %q", spec)
	}
	local := parts[0] + ":" + parts[1]
	remote := parts[2] + ":" + parts[3]
	l, err := net.Listen("tcp", local)
	if err != nil {
		return fmt.Errorf("bind %s: %v", local, err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go func() {
			defer c.Close()
			r, err := net.Dial("tcp", remote)
			if err != nil {
				return
			}
			defer r.Close()
			done := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(r, c); done <- struct{}{} }()
			go func() { _, _ = io.Copy(c, r); done <- struct{}{} }()
			<-done
		}()
	}
}
