package main

// The console's session: the engine in this process, at the coding desk,
// driven from a terminal. One shot (`aetox chat "goal"`) or a line loop; the
// same engine, the same desk, the same tools the desktop's โต๊ะโค้ด has, so
// what is measured here is Aetox and not a second harness with its name.
//
// The desk is not a flag. The console is the coding desk and nothing else
// (owner, 14 ก.ย. 2026: "CLI ทำออกมาเป็น coding อย่างเดียว ตั้งเป็นค่าเริ่มต้น") —
// the other desks are the window's, and a terminal that could open at any of
// them would need the window's pickers to say which.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	aetoxapp "github.com/Mikedev115/Aetox/internal/app"
	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/version"
)

// dials is what the command line may set on the session; "" leaves the
// engine's remembered choice alone.
type dials struct {
	Root     string
	Provider string
	Model    string
	BaseURL  string
	Think    string
	Approval string
}

type console struct {
	e      *engine.Engine
	screen *cliScreen
	// lines is the terminal, one line at a time, from the single goroutine
	// that reads stdin; nil once stdin has closed. One reader, because the
	// prompt and a question the engine is waiting on would otherwise race for
	// the same keystrokes.
	lines chan string
	// running is a turn in flight, for Ctrl-C to know whether it is stopping
	// the answer or the program.
	running atomic.Bool
	cancel  context.CancelFunc
}

type turnResult struct {
	reply engine.TurnReply
	err   error
}

// startConsole builds the engine, opens the project the terminal stands in
// and sits the session at the coding desk. The dials are applied after, in
// the order the window would apply them — provider before model before depth.
func startConsole(d dials, in io.Reader) (*console, error) {
	root := strings.TrimSpace(d.Root)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = wd
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	screen := newCLIScreen()
	e := engine.NewEngine(screen)
	// The chat app's own console lines go nowhere: stdout is the answer and
	// stderr the narration, and those lines belong to neither.
	engine.UseConsole(e, aetoxapp.Discard{})
	ctx, cancel := context.WithCancel(context.Background())
	engine.Startup(e, ctx)
	c := &console{e: e, screen: screen, lines: readLines(in), cancel: cancel}

	if _, err := e.OpenProjectPath(root); err != nil {
		c.close()
		return nil, fmt.Errorf("open %s: %w", root, err)
	}
	if _, err := e.NewSessionAt(mode.Coding); err != nil {
		c.close()
		return nil, err
	}
	if p := strings.TrimSpace(d.Provider); p != "" {
		if _, err := e.SwitchProvider(p); err != nil {
			c.close()
			return nil, err
		}
	}
	if u := strings.TrimSpace(d.BaseURL); u != "" {
		if _, err := e.SetProviderBaseURL(e.GetModelInfo().Provider, u); err != nil {
			c.close()
			return nil, err
		}
	}
	if m := strings.TrimSpace(d.Model); m != "" {
		if _, err := e.SwitchModel(m); err != nil {
			c.close()
			return nil, err
		}
	}
	if t := strings.TrimSpace(d.Think); t != "" {
		if _, err := e.SwitchThinkLevel(t); err != nil {
			c.close()
			return nil, err
		}
	}
	if a := strings.TrimSpace(d.Approval); a != "" {
		if _, err := e.SwitchApprovalMode(a); err != nil {
			c.close()
			return nil, err
		}
	}
	if info := e.GetModelInfo(); info.Warning != "" {
		fmt.Fprintf(os.Stderr, "warning: %s\n", info.Warning)
	}
	return c, nil
}

// readLines is the one stdin reader: every line, CR and LF stripped, until
// the terminal closes — then the channel does.
func readLines(in io.Reader) chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		r := bufio.NewReader(in)
		for {
			line, err := r.ReadString('\n')
			if line != "" {
				lines <- strings.TrimRight(line, "\r\n")
			}
			if err != nil {
				return
			}
		}
	}()
	return lines
}

func (c *console) close() {
	shutdown, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	engine.Shutdown(c.e, shutdown)
	c.cancel()
}

// status is the one line the banner and /status print: what is answering,
// how deep, under which gate, where.
func (c *console) status() string {
	info := c.e.GetModelInfo()
	label := formatModelModeLabel(info.Provider, info.ModelName, info.ThinkLevel)
	return fmt.Sprintf("%s · โต๊ะโค้ด · %s · %s", label, info.ApprovalMode, c.e.GetProjectStatus().Path)
}

// turn runs one message to its end and prints the answer, answering the
// engine's questions from the terminal on the way — an `ask_user` from the
// model, or a tool call under the ask gate. The turn runs beside this loop
// because the engine emits a question from the goroutine that is waiting on
// the answer. A closed stdin (a script with nothing more to say) answers with
// nothing: the model is told the user did not answer, and an approval so
// answered is a refusal.
func (c *console) turn(text string) error {
	c.running.Store(true)
	defer c.running.Store(false)
	done := make(chan turnResult, 1)
	go func() {
		reply, err := c.e.SendMessage(text, "")
		done <- turnResult{reply: reply, err: err}
	}()
	var pending *askEvent
	answer := func(a string) {
		c.e.AnswerUserQuestion(pending.SessionID, a)
		pending = nil
	}
	for {
		select {
		case r := <-done:
			if r.err != nil {
				return r.err
			}
			c.screen.finishTurn(r.reply.Text)
			return nil
		case ask := <-c.screen.asks:
			c.screen.breakPreview()
			fmt.Fprintf(os.Stderr, "\n? %s\n", ask.Question)
			for i, opt := range ask.Options {
				fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, opt)
			}
			pending = &ask
			if c.lines == nil {
				fmt.Fprintln(os.Stderr, "  (no answer — stdin closed)")
				answer("")
				continue
			}
			fmt.Fprint(os.Stderr, "> ")
		case line, ok := <-c.lines:
			if !ok {
				c.lines = nil
				if pending != nil {
					fmt.Fprintln(os.Stderr, "  (no answer — stdin closed)")
					answer("")
				}
				continue
			}
			line = strings.TrimSpace(line)
			if pending == nil {
				if line != "" {
					fmt.Fprintln(os.Stderr, "  (รอให้ตอบเสร็จก่อน)")
				}
				continue
			}
			if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(pending.Options) {
				line = pending.Options[n-1]
			}
			answer(line)
		}
	}
}

// interactive is the line loop. Ctrl-C during an answer stops the answer;
// Ctrl-C at the prompt, or /exit, or a closed stdin, ends the session.
func (c *console) interactive() error {
	fmt.Fprintf(os.Stderr, "Aetox %s — %s\n", version.Current, c.status())
	fmt.Fprintln(os.Stderr, "พิมพ์คำสั่งได้เลย · /help ดูคำสั่ง · /exit ออก")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	quit := make(chan struct{})
	go func() {
		for range interrupt {
			if c.running.Load() {
				c.e.CancelTurn()
				continue
			}
			close(quit)
			return
		}
	}()

	for {
		fmt.Fprint(os.Stderr, "\n› ")
		select {
		case <-quit:
			return nil
		case line, ok := <-c.lines:
			if !ok {
				return nil
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "/") {
				if c.command(line) {
					return nil
				}
				continue
			}
			if err := c.turn(line); err != nil {
				if errors.Is(err, context.Canceled) {
					fmt.Fprintln(os.Stderr, "  (หยุดแล้ว)")
					continue
				}
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
		}
	}
}

// command is the handful of slash commands a terminal needs. Returns true
// when the session should end.
func (c *console) command(line string) bool {
	name, arg, _ := strings.Cut(line, " ")
	arg = strings.TrimSpace(arg)
	report := func(info engine.ModelInfo, err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		if info.Warning != "" {
			fmt.Fprintf(os.Stderr, "warning: %s\n", info.Warning)
		}
		fmt.Fprintln(os.Stderr, c.status())
	}
	switch strings.ToLower(name) {
	case "/exit", "/quit", "/q":
		return true
	case "/help", "/?":
		fmt.Fprint(os.Stderr, slashHelp)
	case "/status":
		fmt.Fprintln(os.Stderr, c.status())
	case "/new":
		if _, err := c.e.NewSessionAt(mode.Coding); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "เริ่มแชทใหม่ที่โต๊ะโค้ด")
		}
	case "/provider":
		if arg == "" {
			fmt.Fprintln(os.Stderr, strings.Join(model.SupportedProviders(), " "))
			return false
		}
		report(c.e.SwitchProvider(arg))
	case "/model":
		if arg == "" {
			fmt.Fprintln(os.Stderr, "usage: /model <name>")
			return false
		}
		report(c.e.SwitchModel(arg))
	case "/think":
		if arg == "" {
			fmt.Fprintln(os.Stderr, strings.Join(c.e.SupportedThinkLevels(), " "))
			return false
		}
		report(c.e.SwitchThinkLevel(arg))
	case "/approval":
		if arg == "" {
			fmt.Fprintln(os.Stderr, "usage: /approval ask|unsafe-only|full-access")
			return false
		}
		report(c.e.SwitchApprovalMode(arg))
	default:
		fmt.Fprintf(os.Stderr, "ไม่รู้จัก %s — /help\n", name)
	}
	return false
}

const slashHelp = `  /status                 โมเดล ระดับคิด โหมดอนุมัติ และโฟลเดอร์ที่ทำงานอยู่
  /new                    เริ่มแชทใหม่ที่โต๊ะโค้ด
  /provider [ชื่อ]         เปลี่ยนผู้ให้บริการ (ไม่ใส่ชื่อ = ดูรายการ)
  /model <ชื่อ>            เปลี่ยนโมเดล
  /think [ระดับ]           เปลี่ยนระดับคิด (ไม่ใส่ = ดูรายการ)
  /approval <โหมด>         ask · unsafe-only · full-access
  /exit                   ออก
  Ctrl-C ระหว่างตอบ = หยุดคำตอบ · ที่พรอมต์ = ออก
`
