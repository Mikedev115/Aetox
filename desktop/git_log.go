package main

// The history, as a room (§250).
//
// The git room (§161.4) answers "where does my repository stand right now".
// This answers the one it cannot: "how did it get here" — the commits, newest
// first, each one openable down to the hunks. They are different questions with
// different rhythms: the working tree moves on every keystroke and is polled;
// history moves only when a commit lands, and is read once per page.
//
// **Everything here is bounded by what the user has looked at.** A page is one
// git process for fifty commits and nothing more — no per-commit work is done
// for a row nobody has opened. The files of a commit cost a second process the
// moment a row is expanded, and a file's hunks a third the moment that file is.
// The owner's brief for this room was the load: "อย่าให้มันหนักเกิน แบ่งโหลดด้วย".
//
// Diffs are built by internal/skill's differ from the two blobs, never by
// parsing `git show -p`, for the same reason the git room does it that way: a
// file's hunks must look and truncate identically under a chat row, under a
// working-tree row, and under a commit row. Two renderers for one thing is how
// they drift.

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// GitCommit is one row of the timeline.
type GitCommit struct {
	Hash    string `json:"hash"`
	Short   string `json:"short"`
	Author  string `json:"author"`
	At      string `json:"at"` // RFC 3339, author date — what "when" means to the person reading
	Subject string `json:"subject"`
	Merge   bool   `json:"merge"`
	Files   int    `json:"files"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

// GitLogPage is one page of the timeline and whether another follows it.
type GitLogPage struct {
	Commits []GitCommit `json:"commits"`
	More    bool        `json:"more"`
}

// gitLogPageMax is the most one call will read. The pane asks for fifty; the
// cap is here so a caller cannot turn one binding into the whole history.
const gitLogPageMax = 200

// GitLog reads one page of the focused project's history, newest first.
//
// `before` is the hash the previous page ended on, or "" for the first page;
// the page starts at that commit's parent. Paging by hash rather than by
// `--skip` means a commit landing between two calls shifts nothing — the second
// page still begins exactly where the first ended.
//
// One process per page: `--shortstat` puts a commit's file count and line
// totals under its own header, so the row's `+N -M` costs no second call.
// A project opened at a subfolder of its repository sees only the commits that
// touched that folder — the same `--show-prefix` rule the working tree uses,
// so the two rooms agree on what "this project" means.
func (a *App) GitLog(before string, limit int) GitLogPage {
	page := GitLogPage{Commits: []GitCommit{}}
	root, ok := a.gitRoot()
	if !ok {
		return page
	}
	if limit <= 0 || limit > gitLogPageMax {
		limit = 50
	}
	ctx, cancel := a.gitContext()
	defer cancel()

	// One more than asked, so "is there another page" is answered by the same
	// read rather than by a second one.
	args := []string{"log", "-n", strconv.Itoa(limit + 1), "--format=" + gitLogFormat, "--shortstat", "--date=iso-strict"}
	if b := strings.TrimSpace(before); b != "" {
		if !isHexHash(b) {
			return page
		}
		args = append(args, b+"^")
	}
	if prefix := cachedRepoPrefix(ctx, root); prefix != "" {
		args = append(args, "--", prefix)
	}
	raw, err := gitOut(ctx, root, args...)
	if err != nil {
		// `before^` on a root commit is the ordinary end of the story, not a
		// failure — and every other error also means "no more to show".
		return page
	}
	rows := parseGitLog(raw)
	if len(rows) > limit {
		rows = rows[:limit]
		page.More = true
	}
	page.Commits = append(page.Commits, rows...)
	return page
}

// gitLogFormat starts every record with a record separator so a body-less
// commit and a commit with a stat line parse the same way: split on the
// separator, first line is the header, whatever follows is the stat.
const gitLogFormat = "%x1e%H%x1f%h%x1f%an%x1f%aI%x1f%P%x1f%s"

var (
	statFiles   = regexp.MustCompile(`(\d+) files? changed`)
	statAdded   = regexp.MustCompile(`(\d+) insertions?\(\+\)`)
	statRemoved = regexp.MustCompile(`(\d+) deletions?\(-\)`)
	hexHash     = regexp.MustCompile(`^[0-9a-fA-F]{4,64}$`)
)

func isHexHash(s string) bool { return hexHash.MatchString(s) }

func parseGitLog(raw string) []GitCommit {
	out := []GitCommit{}
	for _, rec := range strings.Split(raw, "\x1e") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		head, stat, _ := strings.Cut(rec, "\n")
		cols := strings.Split(head, "\x1f")
		if len(cols) != 6 {
			continue
		}
		row := GitCommit{
			Hash:    cols[0],
			Short:   cols[1],
			Author:  cols[2],
			At:      cols[3],
			Merge:   len(strings.Fields(cols[4])) > 1,
			Subject: cols[5],
		}
		if m := statFiles.FindStringSubmatch(stat); m != nil {
			row.Files, _ = strconv.Atoi(m[1])
		}
		if m := statAdded.FindStringSubmatch(stat); m != nil {
			row.Added, _ = strconv.Atoi(m[1])
		}
		if m := statRemoved.FindStringSubmatch(stat); m != nil {
			row.Removed, _ = strconv.Atoi(m[1])
		}
		out = append(out, row)
	}
	return out
}

// GitCommitChanges is what one timeline row unfolds into: the files that
// commit touched, in the same shape the working-tree room uses for its rows,
// so a file reads the same in both. "U" is a file the commit added, "D" one it
// removed, "M" everything else — including a rename, which arrives under its
// new name.
//
// Fetched on expand, never with the page: a page of fifty commits is fifty
// more processes if their files come along, for a list nobody has opened.
func (a *App) GitCommitChanges(hash string) []GitFileChange {
	out := []GitFileChange{}
	root, ok := a.gitRoot()
	if !ok || !isHexHash(strings.TrimSpace(hash)) {
		return out
	}
	hash = strings.TrimSpace(hash)
	ctx, cancel := a.gitContext()
	defer cancel()
	prefix := cachedRepoPrefix(ctx, root)

	// `--raw` for the letter, `--numstat` for the counts, both from one
	// process: git prints the raw lines and then the numstat lines under one
	// commit, and the pane's ten-second budget is shared by every call this
	// binding makes — on a loaded machine a third process is the one that
	// starves. The numbers are still git's own, as the working tree room
	// insists on, never a parse of `-p`. Merges are diffed against their first
	// parent (`-m --first-parent`), which is what "what did this merge bring
	// in" means on a branch you are standing on.
	raw, err := gitOut(ctx, root, "show", "--format=", "--raw", "--numstat", "-m", "--first-parent", hash)
	if err != nil {
		return out
	}
	status := map[string]string{}
	counts := map[string][2]int{}
	order := []string{}
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		if strings.HasPrefix(line, ":") {
			// ":100644 100644 abc def M\tpath" — or "R100\told\tnew" for a
			// rename, whose last column is the name that exists.
			meta, rest, ok := strings.Cut(line, "\t")
			if !ok {
				continue
			}
			fields := strings.Fields(meta)
			if len(fields) < 5 {
				continue
			}
			cols := strings.Split(rest, "\t")
			here, inside := underPrefix(unquoteGitPath(cols[len(cols)-1]), prefix)
			if !inside {
				continue
			}
			letter := "M"
			switch fields[4][:1] {
			case "A":
				letter = "U"
			case "D":
				letter = "D"
			}
			if _, seen := status[here]; !seen {
				order = append(order, here)
			}
			status[here] = letter
			continue
		}
		cols := strings.SplitN(line, "\t", 3)
		if len(cols) != 3 {
			continue
		}
		added, _ := strconv.Atoi(cols[0])
		removed, _ := strconv.Atoi(cols[1])
		path := cols[2]
		// numstat writes a rename as "old => new" or "dir/{old => new}/f";
		// the new name is what the raw line keyed on.
		if strings.Contains(path, " => ") {
			path = renameTarget(path)
		}
		if here, inside := underPrefix(unquoteGitPath(path), prefix); inside {
			counts[here] = [2]int{added, removed}
		}
	}

	for _, p := range order {
		row := GitFileChange{Path: p, Status: status[p]}
		if c, hit := counts[p]; hit {
			row.Added, row.Removed = c[0], c[1]
		}
		out = append(out, row)
	}
	return out
}

// renameTarget turns numstat's rename spelling into the new path:
// "a/{b => c}/d" → "a/c/d", "old => new" → "new".
func renameTarget(p string) string {
	if open := strings.Index(p, "{"); open >= 0 {
		if closeIdx := strings.Index(p[open:], "}"); closeIdx >= 0 {
			inner := p[open+1 : open+closeIdx]
			_, after, _ := strings.Cut(inner, " => ")
			return p[:open] + after + p[open+closeIdx+1:]
		}
	}
	_, after, _ := strings.Cut(p, " => ")
	return after
}

func unquoteGitPath(p string) string {
	if unquoted, err := strconv.Unquote(p); err == nil {
		return unquoted
	}
	return strings.Trim(p, `"`)
}

// GitCommitFileDiff is one file of one commit, as the same hunks the chat
// draws: the blob before (`hash^:path`) against the blob after (`hash:path`),
// through skill.FileDiff. A file the commit added has no "before" and a file it
// removed has no "after"; both read as the empty side, which is what the
// differ draws as all-added or all-removed.
//
// Empty for anything without a readable answer — outside a project, a bad
// hash, a path that leaves the project, a binary blob — for the reason the
// working-tree room gives: every one of those is an ordinary state of a
// repository and none is the user's mistake.
func (a *App) GitCommitFileDiff(hash, path string) string {
	root, ok := a.gitRoot()
	if !ok {
		return ""
	}
	hash = strings.TrimSpace(hash)
	clean := strings.TrimSpace(path)
	if !isHexHash(hash) || clean == "" {
		return ""
	}
	full := filepath.Join(root, filepath.FromSlash(clean))
	if rel, relErr := filepath.Rel(root, full); relErr != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	ctx, cancel := a.gitContext()
	defer cancel()
	// "./" makes the path relative to the directory git was started in, so a
	// project below the repository root needs no prefix lookup here — one
	// process for both blobs and nothing else.
	spec := "./" + filepath.ToSlash(clean)
	blobs, err := gitBlobs(ctx, root, hash+"^:"+spec, hash+":"+spec)
	if err != nil {
		// A read that failed is not a file that is empty: drawing one as the
		// other would show every line removed for a file that merely took too
		// long to read. Nothing, and the pane's "no diff to show".
		return ""
	}
	before, after := blobs[0], blobs[1]
	if strings.ContainsRune(before, 0) || strings.ContainsRune(after, 0) {
		return "" // binary; hunks would be noise, not information
	}
	if before == "" && after == "" {
		return ""
	}
	return skill.FileDiff(clean, before, after)
}

// gitBlobs reads several blobs in one process through `cat-file --batch`,
// answering "" for a spec that names nothing — a file the commit added has no
// parent-side blob, one it removed has no commit-side blob — and an error only
// when the read itself failed, which the caller must not mistake for either.
func gitBlobs(ctx context.Context, root string, specs ...string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", root, "-c", "core.quotepath=false", "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(strings.Join(specs, "\n") + "\n")
	proc.HideConsole(cmd)
	proc.KillOnCancel(cmd)
	raw, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(specs))
	rest := string(raw)
	for range specs {
		header, tail, ok := strings.Cut(rest, "\n")
		if !ok {
			return nil, errors.New("cat-file: short read")
		}
		fields := strings.Fields(header)
		// "<spec> missing" — or "<spec> ambiguous"; either way, no blob.
		if len(fields) != 3 {
			out = append(out, "")
			rest = tail
			continue
		}
		size, convErr := strconv.Atoi(fields[2])
		if convErr != nil || size < 0 || size+1 > len(tail) {
			return nil, errors.New("cat-file: malformed header " + header)
		}
		out = append(out, tail[:size])
		rest = tail[size+1:] // the newline cat-file writes after every object
	}
	return out, nil
}
