package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Console interface {
	Print(msg any)
	Printf(format string, args ...any)
	Println(msg ...any)
	Errorf(format string, args ...any)
	ReadLine() (string, error)
}

type StdIO struct {
	in     *bufio.Reader
	out    io.Writer
	errOut io.Writer
}

func NewStdIO() *StdIO {
	return &StdIO{
		in:     bufio.NewReader(os.Stdin),
		out:    os.Stdout,
		errOut: os.Stderr,
	}
}

func (c *StdIO) Print(msg any) {
	_, _ = fmt.Fprint(c.out, msg)
}

func (c *StdIO) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(c.out, format, args...)
}

func (c *StdIO) Println(msg ...any) {
	_, _ = fmt.Fprintln(c.out, msg...)
}

func (c *StdIO) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(c.errOut, format, args...)
}

func (c *StdIO) ReadLine() (string, error) {
	line, err := c.in.ReadString('\n')
	if err != nil {
		if len(line) == 0 {
			return "", err
		}
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r\n"))
		if trimmed == "" {
			return "", nil
		}
		return trimmed, nil
	}

	return strings.TrimSpace(strings.TrimSuffix(line, "\r\n")), nil
}

// Discard is a console that prints nothing and has nothing to read: the
// engine process's (cmd/aetox-engine, §248 phase 2), whose stdout carries
// one protocol line and whose user is at the far end of a socket. Every
// word the engine has for a person goes through the screen's events, not
// here.
type Discard struct{}

func (Discard) Print(any)                 {}
func (Discard) Printf(string, ...any)     {}
func (Discard) Println(...any)            {}
func (Discard) Errorf(string, ...any)     {}
func (Discard) ReadLine() (string, error) { return "", io.EOF }

var _ Console = Discard{}
