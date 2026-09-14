package lsp

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPathURIRoundTrip(t *testing.T) {
	// gopls rejects file://E:\a\b outright, and a drive letter without the
	// leading slash is read as a hostname — both fail silently as "no
	// diagnostics", which is indistinguishable from "your code is fine".
	for _, p := range []string{`E:\Aetox\Aetox\main.go`, "/home/u/main.go"} {
		uri := pathToURI(p)
		if !strings.HasPrefix(uri, "file:///") {
			t.Errorf("pathToURI(%q) = %q, want a file:/// URI", p, uri)
		}
		if got, want := uriToPath(uri), filepath.ToSlash(mustAbs(t, p)); got != want {
			t.Errorf("round trip of %q gave %q, want %q", p, got, want)
		}
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// "no server for this language" and "a server exists but is not here" must
// stay distinguishable — collapsing them lets an unchecked file be reported as
// a clean one, the most damaging wrong answer this tool can give.
func TestConfiguredIsNotAvailable(t *testing.T) {
	if !Configured("a.rs") {
		t.Error("rust is in the server table, Configured should say so")
	}
	if Configured("notes.txt") {
		t.Error("plain text has no server configured")
	}
	// Available may legitimately be either, depending on the machine — what it
	// must never do is panic or block, and under `go test` it never installs.
	_ = Available(context.Background(), "a.rs")
}

func TestUnsupportedAndMissingServersAreSilent(t *testing.T) {
	c := New(t.TempDir())
	defer c.Close()
	// A language with no server configured, and one configured but (almost
	// certainly) not installed. Both must be quiet: diagnostics are advice on
	// top of an edit that already succeeded.
	for _, name := range []string{"notes.txt", "main.rs"} {
		diags, err := c.Diagnose(context.Background(), name, time.Second)
		if err != nil {
			t.Errorf("%s returned an error: %v", name, err)
		}
		if len(diags) != 0 {
			t.Errorf("%s returned diagnostics: %v", name, diags)
		}
	}
}

// The real thing, against a real server. Skipped where gopls is absent so the
// suite stays green on a machine that never installed it.
func TestDiagnoseReportsRealErrorsFromGopls(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not installed")
	}
	root := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module scratch\n\ngo 1.21\n")
	write("broken.go", "package main\n\nfunc main() {\n\tx := 1\n}\n") // x declared and not used

	c := New(root)
	defer c.Close()

	diags, err := c.Diagnose(context.Background(), "broken.go", 20*time.Second)
	if err != nil {
		t.Fatalf("Diagnose failed: %v", err)
	}
	if len(diags) == 0 {
		t.Fatal("gopls reported nothing for a file with an unused variable")
	}
	found := false
	for _, d := range diags {
		if d.Line == 4 && strings.Contains(strings.ToLower(d.Message), "declared and not used") {
			found = true
		}
		if d.Line < 1 || d.Column < 1 {
			t.Errorf("positions must be 1-based, got %+v", d)
		}
	}
	if !found {
		t.Errorf("expected the unused-variable error on line 4, got: %v", diags)
	}
}

func TestParseReferencesCapsAndConverts(t *testing.T) {
	mk := func(n int) []byte {
		items := make([]string, n)
		for i := range items {
			items[i] = `{"uri":"file:///repo/a.go","range":{"start":{"line":` + strconv.Itoa(i) + `}}}`
		}
		return []byte("[" + strings.Join(items, ",") + "]")
	}
	refs, truncated := parseReferences(mk(3))
	if len(refs) != 3 || truncated {
		t.Fatalf("3 locations should come back whole, got %d truncated=%v", len(refs), truncated)
	}
	if refs[1].Line != 2 {
		t.Errorf("LSP lines are 0-based and ours are 1-based, got %d", refs[1].Line)
	}
	refs, truncated = parseReferences(mk(maxRefs + 20))
	if len(refs) != maxRefs || !truncated {
		t.Fatalf("the cap must bite AND say so, got %d truncated=%v", len(refs), truncated)
	}
	if refs, _ := parseReferences([]byte("null")); refs != nil {
		t.Error("a null answer is no references, not a panic")
	}
}

// The doc comment above a Go declaration begins with the symbol's own name,
// so "first occurrence" landed in prose for every documented function and the
// server answered with nothing (codebase symbol app.go CancelTurn, 14 ก.ย.
// 2026). The declaration must win; a comment is only ever the fallback.
func TestFindIdentifierPrefersCodeOverComment(t *testing.T) {
	src := strings.Join([]string{
		"package engine",
		"",
		"// CancelTurn aborts the chat turn in flight (the tool loop is unbounded, so",
		"// CancelTurn is the only way out).",
		"func (a *Engine) CancelTurn() {",
		"\ta.cancelTurn()",
		"}",
	}, "\n")
	line, ch, ok := findIdentifier(src, "CancelTurn")
	if !ok || line != 4 || ch != len("func (a *Engine) ") {
		t.Fatalf("want the declaration at line 4, got line=%d ch=%d ok=%v", line, ch, ok)
	}
	// Nothing but comments mention it: still found, so a name that lives only
	// in prose gets the old behaviour rather than "does not appear".
	line, _, ok = findIdentifier("# only Frobnicate here\nx = 1\n", "Frobnicate")
	if !ok || line != 0 {
		t.Fatalf("comment-only occurrence must still be found, got line=%d ok=%v", line, ok)
	}
	// Standalone still holds on the code sweep.
	if _, _, ok := findIdentifier("func Getter() {}\n", "Get"); ok {
		t.Error(`"Get" must not land inside "Getter"`)
	}
}

// A server that answers a request with a JSON-RPC error used to be read as an
// empty success: readLoop unmarshalled `result` alone. The reason gopls gave
// is the only diagnostic a user has when all three lookups fail, so it has to
// reach the caller as an error.
func TestCallSurfacesServerError(t *testing.T) {
	pr, pw := io.Pipe()
	// readLoop closes the conn when the pipe ends, and close() waits on the
	// server process — so the fake gets a real, already-finished one.
	cmd := exec.Command("go", "version")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skip("no go binary on PATH:", err)
	}
	cn := &conn{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(pr),
		diags:   map[string][]Diagnostic{},
		updated: make(chan string, 1),
		pending: map[int]chan reply{},
	}
	go cn.readLoop()
	answer := make(chan reply, 1)
	cn.mu.Lock()
	cn.pending[7] = answer
	cn.mu.Unlock()
	body := `{"jsonrpc":"2.0","id":7,"error":{"code":-32602,"message":"no identifier found"}}`
	go func() {
		_, _ = pw.Write([]byte("Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body))
		_ = pw.Close()
	}()
	select {
	case r := <-answer:
		if r.err == nil || !strings.Contains(r.err.Error(), "no identifier found") {
			t.Fatalf("server error must come through, got %v", r.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("reply never delivered")
	}
}
