// Command aetox-engine is the engine as a process of its own (§248 phase 2):
// everything the desktop app does that is not a window, listening on one
// socket for the screen that started it.
//
//	aetox-engine serve --socket <path> | --tcp 127.0.0.1:0
//	                   [--root <project>] [--token-stdin | --token-file <path>]
//	                   [--idle-exit 30m]
//
// The token is the whole admission to the socket. It comes on stdin (the
// screen that spawned this process writes it and then holds stdin open, so
// EOF there means the screen is gone and this process exits with it) or
// from a file only the user can read (the remote host, phase 3). It is never
// an argument: argv is what `ps` shows to everyone.
//
// The one line this prints on stdout is where it listens, as JSON, so a
// screen that asked for --tcp 127.0.0.1:0 learns the port. Everything else
// goes to <DataRoot>/logs/engine.log — a file of its own, so local mode
// never has two processes writing one log.
//
// A binary of its own rather than `aetox serve`: the release workflow ships
// no CLI on purpose (§30), a fixed file name is what `pkill` and
// ~/.aetox/server/<version>/ need, and the CLI's own future is to become a
// second screen on this engine.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	aetoxapp "github.com/Mikedev115/Aetox/internal/app"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/engine/rpc"
	"github.com/Mikedev115/Aetox/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		if err := serve(os.Args[2:]); err != nil {
			log.Printf("aetox-engine: %v", err)
			fmt.Fprintln(os.Stderr, "aetox-engine:", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Println(version.Current)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: aetox-engine serve (--socket <path> | --tcp <addr>) [--root <project>] [--token-stdin | --token-file <path>] [--idle-exit <duration>]")
}

// listening is the one line on stdout.
type listening struct {
	Network string `json:"network"`
	Address string `json:"address"`
	PID     int    `json:"pid"`
	Version string `json:"version"`
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	socket := fs.String("socket", "", "listen on this unix socket (removed first if stale)")
	tcp := fs.String("tcp", "", "listen on this TCP address instead, e.g. 127.0.0.1:0")
	root := fs.String("root", "", "open this project folder after startup")
	tokenStdin := fs.Bool("token-stdin", false, "read the token as the first line of stdin, and exit when stdin closes")
	tokenFile := fs.String("token-file", "", "read the token from this file")
	idle := fs.Duration("idle-exit", 0, "exit after this long with no screen connected (0 = never)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var network, address string
	switch {
	case *socket != "" && *tcp != "":
		return errors.New("--socket and --tcp are one or the other")
	case *socket != "":
		network, address = "unix", *socket
	case *tcp != "":
		network, address = "tcp", *tcp
	default:
		return errors.New("--socket <path> or --tcp <addr> is required")
	}

	dataRoot, err := config.DataRoot()
	if err != nil {
		return err
	}
	closeLog := openLog(dataRoot)
	defer closeLog()

	token, stdinGone, err := readToken(*tokenStdin, *tokenFile)
	if err != nil {
		return err
	}

	srv := rpc.NewServer(token, func(screen engine.Screen) *engine.Engine {
		e := engine.NewEngine(screen)
		engine.UseConsole(e, aetoxapp.Discard{})
		return e
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Startup(srv.Engine(), ctx)
	if *root != "" {
		if _, err := srv.Engine().OpenProjectPath(*root); err != nil {
			log.Printf("--root %s: %v", *root, err)
		}
	}

	l, err := rpc.Listen(network, address)
	if err != nil {
		return err
	}
	if network == "unix" {
		defer os.Remove(address)
	}
	// The one line for the screen: where this process listens. Flushed
	// before serving so a screen waiting on the pipe never waits on a buffer.
	line, _ := json.Marshal(listening{Network: network, Address: l.Addr().String(), PID: os.Getpid(), Version: version.Current})
	fmt.Println(string(line))
	log.Printf("listening on %s %s (pid %d, version %s)", network, l.Addr(), os.Getpid(), version.Current)

	// Three ways out: a signal, the screen's stdin closing, the idle clock.
	stop := make(chan string, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case s := <-sigs:
			stop <- "signal " + s.String()
		case <-stdinGone:
			stop <- "the screen closed stdin"
		case <-ctx.Done():
		}
	}()
	if *idle > 0 {
		go watchIdle(ctx, srv, *idle, stop)
	}

	served := make(chan error, 1)
	go func() { served <- srv.Serve(ctx, l) }()
	var why string
	select {
	case why = <-stop:
	case err := <-served:
		if err != nil {
			return err
		}
		why = "the listener closed"
	}
	log.Printf("stopping: %s", why)
	cancel()
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	engine.Shutdown(srv.Engine(), shutdown)
	select {
	case <-served:
	case <-time.After(3 * time.Second):
	}
	log.Printf("stopped")
	return nil
}

// readToken is the token, and — when it came on stdin — a channel that
// closes when stdin does.
func readToken(fromStdin bool, file string) (string, <-chan struct{}, error) {
	gone := make(chan struct{})
	switch {
	case fromStdin && file != "":
		return "", nil, errors.New("--token-stdin and --token-file are one or the other")
	case file != "":
		b, err := os.ReadFile(file)
		if err != nil {
			return "", nil, fmt.Errorf("reading the token: %w", err)
		}
		token := strings.TrimSpace(string(b))
		if token == "" {
			return "", nil, errors.New("the token file is empty")
		}
		return token, gone, nil
	case fromStdin:
		r := bufio.NewReader(os.Stdin)
		line, err := r.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", nil, fmt.Errorf("reading the token from stdin: %w", err)
		}
		token := strings.TrimSpace(line)
		if token == "" {
			return "", nil, errors.New("no token on stdin")
		}
		// The screen holds stdin open for as long as it lives; the read
		// returns only when it is gone, and that is the signal to leave.
		go func() {
			defer close(gone)
			_, _ = io.Copy(io.Discard, r)
		}()
		return token, gone, nil
	default:
		return "", nil, errors.New("a token is required: --token-stdin or --token-file <path>")
	}
}

// watchIdle stops the process after `after` with no screen connected. A
// screen that comes and goes resets the clock each time it goes.
func watchIdle(ctx context.Context, srv *rpc.Server, after time.Duration, stop chan<- string) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	alone := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			if srv.Screen().Connected() {
				alone = now
				continue
			}
			if now.Sub(alone) >= after {
				stop <- fmt.Sprintf("no screen for %s", after)
				return
			}
		}
	}
}

// openLog sends the process's own lines to <DataRoot>/logs/engine.log,
// appended, and answers with the file's closer. A log that cannot be
// opened is a log on stderr, not a refusal to start.
func openLog(dataRoot string) func() {
	dir := filepath.Join(dataRoot, "logs")
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(filepath.Join(dir, "engine.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		log.SetOutput(os.Stderr)
		return func() {}
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	return func() { _ = f.Close() }
}
