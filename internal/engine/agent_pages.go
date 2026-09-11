package engine

// The pages the agent opened, read back for the browser tab's start page
// (§248 B1: the browser is the window's, the record of where it has been is
// the engine's — it is rows in tool_runs).

import (
	"database/sql"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// ---------- reading those back ----------
//
// Every browser_open the agent has ever run is already on disk: recordToolRun
// writes one tool_runs row per call and nothing in the app has ever read one
// back. RecentAgentPages is that read, and it is what a new browser tab is made
// of — the tab opens showing where the agent has been rather than a blank slate
// (ARCHITECTURE.md §81).

// AgentPage is one page the agent opened, for the browser tab's start page.
type AgentPage struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Time  string `json:"time"` // RFC3339 — what the frontend's agoLabel() parses
}

const (
	// BrowserOpenedPrefix is shared by the sentence `open` writes
	// (BrowserOpenedLine) and the parser below so the two cannot drift
	// without the diff putting them side by side.
	BrowserOpenedPrefix = "เปิดแล้ว: "
	agentPageScanRows   = 200 // rows read before de-duplication
	agentPageMax        = 50
	agentPageDefault    = 24
)

// BrowserOpenedLine is the one place the sentence `open` writes is written.
// A function rather than an inline Sprintf so the round-trip test can call
// the real writer — sharing only the prefix constant left the test with its
// own copy of the format, which meant editing this line could not fail
// anything. The browser that says it is the window's (desktop/workbench.go);
// the sentence lives here, beside the parser that reads it back.
func BrowserOpenedLine(title, url string) string {
	return BrowserOpenedPrefix + BrowserPageRef(title, url)
}

// BrowserPageRef names a page, and is the one place any browser action spells
// out which page it is talking about.
//
// It exists because there were four spellings and they were added one at a
// time, each perfectly reasonable on its own: `open` said "Title (url)", `read`
// wrote a document header, and `click`/`type`/`capture` each invented a third
// and fourth. One fact, four renderings, in a file that already kept the
// prefix as a shared constant precisely so `open`'s sentence and the parser
// cannot drift apart. The lesson was already written down here; it just was
// not applied to the sentences added after it.
//
// `read` is the deliberate exception and stays a two-line markdown header. It
// is not a sentence referring to a page, it is the top of a document the model
// reads as a document, and folding it into this shape would make it worse to
// read to make it easier to count.
func BrowserPageRef(title, url string) string {
	title, url = strings.TrimSpace(title), strings.TrimSpace(url)
	switch {
	case title != "" && url != "":
		return fmt.Sprintf("%s (%s)", title, url)
	case url != "":
		return url // a page that has not told us its title yet
	default:
		return title // rare, and better than saying nothing
	}
}

// ParseBrowserOpened splits the line `open` writes back apart again.
//
// Parsing our own sentence rather than storing structured output is the
// deliberate half of this: RawOutput is what the *model* reads (see
// toolRunOutput in the turn executor), so making it JSON for the sake of a
// panel would change model-facing text to save a function. The round-trip test
// in workbench_agentpages_test.go is what keeps the pair honest.
func ParseBrowserOpened(output string) (title, pageURL string) {
	s := strings.TrimSpace(output)
	if !strings.HasPrefix(s, BrowserOpenedPrefix) {
		return "", "" // a failure line ("เปิดไม่สำเร็จ: …") or something else entirely
	}
	s = strings.TrimPrefix(s, BrowserOpenedPrefix)

	// A page with no title is written as the bare address, because
	// "เปิดแล้ว:  (https://x)" is a sentence with a hole in it. Both shapes are
	// BrowserPageRef's output and both have to come back apart here — this
	// parser and that writer are one contract with two halves, which is what
	// the shared prefix constant above has always been about.
	if !strings.HasSuffix(s, ")") {
		if urlSchemeRe.MatchString(s) || strings.HasPrefix(s, "file:///") {
			return "", s
		}
		return "", "" // truncated, or a sentence that is not this one
	}

	// LAST " (", so a page whose own title contains one survives.
	open := strings.LastIndex(s, " (")
	if open < 0 {
		return "", ""
	}
	return strings.TrimSpace(s[:open]), s[open+2 : len(s)-1]
}

// urlFromArgs is the fallback when the sentence above cannot be read: a format
// change then costs the title, never the whole list.
func urlFromArgs(args string) string {
	var a struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(args), &a); err != nil {
		return ""
	}
	u := strings.TrimSpace(a.URL)
	// browser_open also accepts a sandbox-relative path, and a row rebuilt from
	// one would navigate to https://out/report.html. Only an absolute URL is
	// recoverable from args alone.
	if !strings.Contains(u, "://") {
		return ""
	}
	return u
}

// LocalFileBehind answers with the file on disk a page URL is showing, or ""
// for a page that is not a local file or is one that has since gone. The
// browser's capture asks it too, to know whether a page is a deck.
//
// Unescaped first, then raw: a Thai filename may be percent-encoded on the way
// into the URL and is not on the way out of os.WriteFile.
func LocalFileBehind(pageURL string) string {
	if !strings.HasPrefix(pageURL, "file:///") {
		return ""
	}
	p := filepath.FromSlash(strings.TrimPrefix(pageURL, "file:///"))
	if unescaped, err := neturl.PathUnescape(p); err == nil {
		if _, err := os.Stat(unescaped); err == nil {
			return unescaped
		}
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// stillOpenable drops a local file that has since been deleted — session output
// folders age out, and a row that opens the engine's "not found" page is the
// dead end this surface exists to prevent. Remote pages are not checked: a 404
// still has back, reload and the address bar directly above it.
func stillOpenable(pageURL string) bool {
	return !strings.HasPrefix(pageURL, "file:///") || LocalFileBehind(pageURL) != ""
}

// RecentAgentPages returns the pages the agent opened, newest first, one row
// per URL. Machine-wide rather than per-project or per-session: a page the
// agent opened is not project data, and scoping it through sessions would hide
// the row the list exists for.
func (a *Engine) RecentAgentPages(limit int) []AgentPage {
	out := []AgentPage{}
	if limit <= 0 || limit > agentPageMax {
		limit = agentPageDefault
	}
	db, err := a.database()
	if err != nil {
		// Logged, not propagated, for the reason recordToolRun states: the
		// pane's "nothing here yet" line is the honest content of a database
		// that will not open, too.
		debuglog.Msg("agent pages: db unavailable: %v", err)
		return out
	}
	seen := map[string]bool{}
	_ = eachRow(db, "agent pages", `
		SELECT args, output, time FROM tool_runs
		 WHERE tool = 'browser_open' AND ok = 1
		 ORDER BY id DESC LIMIT ?`, []any{agentPageScanRows},
		func(rows *sql.Rows) error {
			var args, output, ts string
			if err := rows.Scan(&args, &output, &ts); err != nil {
				return err
			}
			title, pageURL := ParseBrowserOpened(output)
			if pageURL == "" {
				pageURL = urlFromArgs(args)
			}
			// Newest wins: the same page opened three times is one row, carrying
			// the most recent time.
			if pageURL == "" || seen[pageURL] || !stillOpenable(pageURL) {
				return nil
			}
			seen[pageURL] = true
			out = append(out, AgentPage{URL: pageURL, Title: title, Time: ts})
			if len(out) == limit {
				// A full page is not a failure, and rows.Err() is nil after a
				// caller-initiated stop.
				return errStopRows
			}
			return nil
		})
	return out
}
