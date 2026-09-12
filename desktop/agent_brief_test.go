package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The links people paste for a brief, and the URL each one is read from:
// a GitHub file page becomes the raw file, a Drive share becomes the
// download, a Docs link becomes the document as text, and anything else
// goes through untouched (§256.5).
func TestABriefLinkIsTurnedIntoTheFilesOwnURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/mike/agents/blob/main/sales/AGENT.md":     "https://raw.githubusercontent.com/mike/agents/main/sales/AGENT.md",
		"https://github.com/mike/agents/raw/v2/brief.txt":             "https://raw.githubusercontent.com/mike/agents/v2/brief.txt",
		"https://gist.github.com/mike/0123abcd":                       "https://gist.github.com/mike/0123abcd/raw",
		"https://gist.github.com/mike/0123abcd/raw":                   "https://gist.github.com/mike/0123abcd/raw",
		"https://drive.google.com/file/d/1AbC_dEf/view?usp=sharing":   "https://drive.google.com/uc?export=download&id=1AbC_dEf",
		"https://drive.google.com/open?id=1AbC_dEf":                   "https://drive.google.com/uc?export=download&id=1AbC_dEf",
		"https://docs.google.com/document/d/1DoC_iD/edit?usp=sharing": "https://docs.google.com/document/d/1DoC_iD/export?format=txt",
		"https://raw.githubusercontent.com/mike/agents/main/a.md":     "https://raw.githubusercontent.com/mike/agents/main/a.md",
		"  https://example.com/brief.md  ":                            "https://example.com/brief.md",
	}
	for in, want := range cases {
		got, err := briefURL(in)
		if err != nil || got != want {
			t.Errorf("briefURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "brief.md", "ftp://x/y", "https://"} {
		if _, err := briefURL(bad); err == nil {
			t.Errorf("briefURL(%q) accepted", bad)
		}
	}
}

// What comes back is the text and nothing else: a markdown file lands as it
// is (CRLF and BOM straightened), a web page in place of the file is refused
// with a sentence that says what to paste instead, and so is a file over the
// cap or one that is not text.
func TestFetchingABriefHandsBackTextAndRefusesPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/brief.md":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("\xEF\xBB\xBF# บทบาท\r\nตอบลูกค้าเรื่องราคา\r\n\r\n"))
		case "/page":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<!DOCTYPE html><html><body>sign in</body></html>"))
		case "/sneaky":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("<html><body>drive scan</body></html>"))
		case "/big":
			_, _ = w.Write([]byte(strings.Repeat("x", briefMaxBytes+1)))
		case "/binary":
			_, _ = w.Write([]byte{0x89, 'P', 'N', 'G', 0, 1, 2})
		case "/private":
			w.WriteHeader(http.StatusForbidden)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	got, err := fetchBrief(ctx, srv.Client(), srv.URL+"/brief.md")
	if err != nil || got != "# บทบาท\nตอบลูกค้าเรื่องราคา\n" {
		t.Fatalf("brief.md: %q, %v", got, err)
	}
	for path, want := range map[string]string{
		"/page":    "หน้าเว็บ",
		"/sneaky":  "หน้าเว็บ",
		"/big":     "ใหญ่เกิน",
		"/binary":  "ไม่ใช่ไฟล์ข้อความ",
		"/private": "ไม่ได้เปิดสาธารณะ",
		"/gone":    "404",
	} {
		_, err := fetchBrief(ctx, srv.Client(), srv.URL+path)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want it to say %q", path, err, want)
		}
	}
}

// The file road reads the same way: the words, the cap, the text check.
func TestABriefFileIsReadAsText(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sales.md")
	if err := os.WriteFile(p, []byte("บทบาท: ขาย\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := readBriefFile(p); err != nil || got != "บทบาท: ขาย\n" {
		t.Fatalf("sales.md: %q, %v", got, err)
	}
	empty := filepath.Join(dir, "empty.txt")
	_ = os.WriteFile(empty, []byte("  \n"), 0o644)
	if _, err := readBriefFile(empty); err == nil || !strings.Contains(err.Error(), "ว่างเปล่า") {
		t.Errorf("empty: %v", err)
	}
	if _, err := readBriefFile(filepath.Join(dir, "missing.md")); err == nil {
		t.Error("a missing file was read")
	}
}
