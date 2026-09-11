package engine

// What the window's tool packs ask of the engine (§248 B1).
//
// The browser and the machine act on the screen's computer, and live there
// (desktop/browser_*.go, desktop/computer_*.go). But a screenshot they take
// belongs in the project, a path the model names is a path in the project,
// and "which chat is on screen" is the engine's to say — so those are twins
// here, and the packs call them the way the frontend calls any binding. When
// the engine is on another machine these are the calls that cross the
// socket, which is why each takes and answers plain values.

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/Mikedev115/Aetox/internal/skill"
)

// MaxDeckBytes bounds what the deck listing, the export and the browser's
// deck sniff will read: deciding costs a read and an HTML parse, and a deck
// with its pictures inline is megabytes.
const MaxDeckBytes = maxDeckBytes

// AnyTurnRunning reports whether any chat has a turn in flight — what a
// window closing its tabs asks before keeping the agent's.
func (a *Engine) AnyTurnRunning() bool { return a.anyTurnRunning() }

// PageMarksOn is the busy-signal preference the browser reads: whether the
// numbered marks are drawn on the page the agent is looking at.
func (a *Engine) PageMarksOn() bool { return !a.cfg.BusyPageMarksOff }

// ComputerControlChanged is what the screen tells the engine after the
// machine switch moved (SetComputerControlOn, desktop/computer_guard.go): the
// tool block of the chat on screen is rebuilt with or without the tool.
func (a *Engine) ComputerControlChanged() {
	if a.cur() != nil {
		a.applyConfig(a.cur(), a.cur().cfg)
	}
}

// normalizeWorkbenchURL turns what the agent asked for into something the
// browser can go to, or reports that it was not an address at all.
//
// A local file first, and only a file that actually exists: every other tool
// speaks sandbox-relative paths, so without this the model has to splice the
// root in by hand to look at what it just made, and "index.html" would fall
// through and navigate to https://index.html. It resolves through
// skill.PlacedPath rather than joining onto the root, because `write` steers a
// new relative file into the session output folder — the model asks for
// "index.html", the file is really at "aetox/output/<session>/index.html", and a
// plain root join finds nothing and degrades to a DNS lookup for a hostname
// called index.html.
//
// Everything else goes to resolveAddress, which is now the single place that
// tells a place from a question (see address.go). What is new is the second
// return: this used to stamp https:// onto ANYTHING left over, so `open("ยูทูป")`
// became a DNS failure that read like a broken website. The agent has
// web_search; being told to use it is a better answer than being quietly
// searched for, which would teach it that `open` is a search box.
func normalizeWorkbenchURL(input, sandboxRoot string, outputSubdir func() string) (resolved, query string) {
	if root := strings.TrimSpace(sandboxRoot); root != "" && !urlSchemeRe.MatchString(input) && !bareSchemeRe.MatchString(input) {
		placed := skill.PlacedPath(root, outputSubdir, input)
		if abs, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(placed))); err == nil {
			if _, statErr := os.Stat(abs); statErr == nil {
				return "file:///" + strings.ReplaceAll(abs, `\`, "/"), ""
			}
		}
	}
	addr := resolveAddress(input)
	return addr.URL, addr.Query
}

// ResolveWorkbenchURL is what the browser opens for the address the model
// wrote: a sandbox-relative file becomes a file URL steered through the
// output folder the way `write` places new files; words that are not an
// address come back as a query, for the pack to refuse with advice.
func (a *Engine) ResolveWorkbenchURL(input string) (resolved, query string) {
	return normalizeWorkbenchURL(input, a.cur().cfg.SandboxRoot, a.outputSubdir)
}

// handedOverFile turns a file:/// address back into the project-relative path
// `write` reported, when — and only when — it lies in this chat's own output
// folder. "" for any web address, any file outside the root, and any file of
// the project: none of those is a thing this chat made.
func handedOverFile(root, outputSubdir, url string) string {
	if !strings.HasPrefix(url, "file:///") || strings.TrimSpace(root) == "" {
		return ""
	}
	abs := filepath.FromSlash(strings.TrimPrefix(url, "file:///"))
	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if !handedOver(outputSubdir, rel) {
		return ""
	}
	return rel
}

// HandedOverFile is what the browser asks after an `open` landed: the
// project-relative path `write` reported, when the page is a file of this
// chat's own output folder, "" otherwise — the artifact flag is about what
// this chat made, never about what it showed.
func (a *Engine) HandedOverFile(fileURL string) string {
	return handedOverFile(a.cur().cfg.SandboxRoot, a.outputSubdir(), fileURL)
}

// SandboxFile resolves a path the way `open` does for a page — sandbox-
// relative, steered through the output folder the way `write` places new
// files — and refuses what every other tool refuses: a credential store, or
// anywhere outside the sandbox.
func (a *Engine) SandboxFile(request string) (string, error) {
	root := strings.TrimSpace(a.cur().cfg.SandboxRoot)
	if root == "" {
		return "", fmt.Errorf("no working folder is set")
	}
	placed := skill.PlacedPath(root, a.outputSubdir, request)
	return skill.SandboxFile(root, placed)
}

var browserShotSeq int64

// workSubdir is the one folder name the app creates for its own working files.
// English, like output/ above it: this is a real directory the user will meet in
// Explorer and quote into a shell, and the gallery translates it for the card
// rather than putting Thai in a path.
const workSubdir = "work"

// workFileDir is where a file the agent produced **while working** goes, as a
// sandbox-relative path: under the session's own output folder, in a
// subfolder that says what it is. The gallery reads the place as the fact —
// Artifact.Folder, one card per folder. Nothing new records what a file is:
// the place it was put says it, which is the only kind of record this page
// trusts, because the folder is the half the user can move and rename.
//
// The subfolder is added here rather than at the call site so that the next
// tool with a byproduct inherits it by using this function, which is the
// reason the function has a name at all.
func (a *Engine) workFileDir() string {
	session := strings.TrimSpace(a.cur().id)
	if session == "" {
		session = "unsaved" // a chat that has not been saved can still take a picture
	}
	return path.Join("output", session, workSubdir)
}

// SaveBrowserShot puts a picture the browser took in the work-file folder of
// the chat on screen and answers with the sandbox-relative path. marked names
// the file for what it is: the picture with the numbers on it is not the page
// — it is what the agent was looking at when it decided which thing to press,
// which is the one piece of evidence that answers "why did it click that" a
// day later. A gallery in which both are called page-4.png is a gallery where
// nobody can tell those two apart.
func (a *Engine) SaveBrowserShot(png []byte, marked bool) (string, error) {
	root := strings.TrimSpace(a.cur().cfg.SandboxRoot)
	if root == "" {
		return "", fmt.Errorf("no working folder is set")
	}
	name := fmt.Sprintf("page-%d.png", atomic.AddInt64(&browserShotSeq, 1))
	if marked {
		name = strings.TrimSuffix(name, ".png") + "-marks.png"
	}
	rel := path.Join(a.workFileDir(), name)
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, png, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}
