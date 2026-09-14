package main

import (
	"log"
	"strings"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// wailsLog is the Wails runtime's logger, pointed at this app's own log.
//
// Wails writes to stdout by default, and a windowsgui build has no stdout:
// every line it ever wrote on an installed machine went nowhere. Most of
// those are noise, but not all — when the WebView2 browser process dies,
// Wails logs "WebView2Process failed with kind N", shows a message box, and
// calls os.Exit(-1) on the whole app (internal/frontend/desktop/windows/
// frontend.go, ProcessFailedCallback). That is a death with no crash file (it
// is an exit, not a panic), no Windows Error Report and no Event 1000 — the
// exact shape of the 14 ก.ย. 2026 incident, whose desktop log simply stopped.
// With this in place the log's last line names the kind.
//
// Wails also reaches for the standard library's log.Fatal in a dozen places
// (a WebView2 setting it could not put, a binding it could not serialise), so
// stdLogWriter sends the standard logger the same way; those lines are the
// last thing the process says before os.Exit(1).
type wailsLog struct{}

func (wailsLog) Print(m string)   { debuglog.Msg("wails: %s", m) }
func (wailsLog) Trace(m string)   { debuglog.Msg("wails: %s", m) }
func (wailsLog) Debug(m string)   { debuglog.Msg("wails: %s", m) }
func (wailsLog) Info(m string)    { debuglog.Msg("wails: %s", m) }
func (wailsLog) Warning(m string) { debuglog.Msg("wails: WARNING %s", m) }
func (wailsLog) Error(m string)   { debuglog.Msg("wails: ERROR %s", m) }
func (wailsLog) Fatal(m string)   { debuglog.Msg("wails: FATAL %s", m) }

// stdLogWriter is the standard library's log output, as debuglog lines.
type stdLogWriter struct{}

func (stdLogWriter) Write(p []byte) (int, error) {
	debuglog.Msg("log: %s", strings.TrimRight(string(p), "\r\n"))
	return len(p), nil
}

// routeStrayLogs points both loggers at debuglog. Before wails.Run, so the
// lines Wails writes while it is still building the window are kept too —
// in the ring until the file opens at startup, then replayed into it.
func routeStrayLogs() {
	log.SetFlags(0)
	log.SetOutput(stdLogWriter{})
}
