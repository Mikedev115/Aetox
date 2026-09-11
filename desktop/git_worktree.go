package main

// The working tree, as a room you can open (§161.4).
//
// The chat timeline says what *this turn's* calls changed, hunk by hunk, under
// the row that made each one. That is the right answer to "what did you just
// do" and the wrong one to "where does my repository stand" — a turn is not a
// session, a session is not an afternoon, and by the third turn the honest
// answer to the second question lives nowhere in the window.
//
// So: one panel, the branch and the working tree at the top, a row per changed
// file with its own `+N -M`, and a diff behind each row. The diff is built by
// internal/skill's differ rather than by parsing `git diff` output, for one
// reason that is worth the extra call: the fold-out in the chat and the fold-out
// here then draw the *same format*, capped the same way, from the same code. Two
// renderers for one thing is how they drift.
//
// Every "not applicable" answers the same way — unfocused, not a repo, git not
// installed — because all three mean "there is no working tree to show", and a
// panel that distinguishes them would be reporting on itself rather than on the
// user's code.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// GitFileChange is one row of the panel.
type GitFileChange struct {
	Path string `json:"path"`
	// "M" changed, "U" untracked or newly added, "D" gone. The same three the
	// file tree's badges use, so a file means the same thing in both places.
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

// GitWorkingTree lists what the focused project's working tree has that HEAD
// does not, newest information first: status from `git status --porcelain`,
// line counts from `git diff --numstat HEAD`.
//
// Counts come from a second call rather than being computed here, because
// numstat is what git itself reports and a panel that disagreed with
// `git diff --stat` about the same file would be a bug nobody could explain.
// An untracked file has no numstat row — nothing to compare against — so its
// own line count is the addition, which is what git shows once it is added.
//
// **A read that failed is not a tree that is clean.** Outside a project or a
// repository the answer is an empty list and no error — there is nothing to
// show. A git that timed out or refused is an error, and the pane keeps the
// last tree it had rather than drawing "nothing changed" over fifty-eight
// files (owner, 12 ก.ย., on a machine where a git process was taking seconds:
// "ทำไมมันค้างแบบนี้"). The room polls, so the next read will say more.
func (a *App) GitWorkingTree() ([]GitFileChange, error) {
	out := []GitFileChange{}
	root, ok := a.gitRoot()
	if !ok {
		return out, nil
	}
	ctx, cancel := a.gitContext()
	defer cancel()
	rows, err := workingTree(ctx, root, true)
	if err != nil {
		return out, err
	}
	return append(out, rows...), nil
}

// workingTree is the one reader of `git status --porcelain` in this app.
//
// Three surfaces ask this question — the git room's rows, the summary strip's
// repo section, and the file tree's badges — and they used to ask it with three
// copies of the same parse. The letters were kept in step by comment ("the same
// three the file tree's badges use") rather than by code, which is exactly the
// arrangement that drifts on the day one of them learns something the others
// do not. This is that something: `countUntracked`, and the prefix below.
//
// **Paths come back relative to `root`.** Porcelain always prints them relative
// to the *repository* root, so a project opened at a subfolder of its repo got
// rows whose paths matched nothing it could show — no badge in the tree, and a
// summary listing paths that read as if they were somewhere else.
// `rev-parse --show-prefix` is where the project sits inside the repo, and
// anything outside it belongs to a part of the repository this window is not
// looking at.
//
// countUntracked is what an untracked file's `+N` costs: its whole content, read
// to be counted. Worth it for a panel of rows somebody is reading; not worth it
// for a tree that only needs to know the file is new, and where a folder full of
// unadded files would mean reading every one of them on every refresh.
//
// Two processes, not three: where the project sits inside its repository is
// remembered per root (cachedRepoPrefix), because this runs every two seconds
// while the git room is in front and again for every file-tree refresh.
// Measured on the owner's repository, 12 ก.ย.: rev-parse 175 ms, status 264 ms,
// numstat 342 ms — a fifth of the bill when git is quick, a third when each
// process takes a second to start, for a fact that never changes.
func workingTree(ctx context.Context, root string, countUntracked bool) ([]GitFileChange, error) {
	out := []GitFileChange{}
	status, err := gitOut(ctx, root, "status", "--porcelain")
	if err != nil {
		if ctx.Err() != nil {
			return out, errGitSlow
		}
		return out, err
	}
	prefix := cachedRepoPrefix(ctx, root)
	counts, err := numstat(ctx, root)
	if err != nil {
		return out, err
	}

	// One row per path. Porcelain prints a path twice when the index and the
	// disk disagree about whether it exists — `D ` for a file removed from the
	// index and `??` for the same file still on disk, which is what an index
	// left behind by another process looks like (seen 12 ก.ย.: six files, and
	// the pane's keyed list refusing to draw sixty-three rows over a duplicate
	// key). Two rows for one file is two truths; one row saying "M" is the fact.
	at := map[string]int{}
	for _, line := range strings.Split(strings.TrimRight(status, "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		code := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		// A rename arrives as "old -> new"; the new name is the one that exists.
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}
		if unquoted, err := strconv.Unquote(path); err == nil {
			path = unquoted
		} else {
			path = strings.Trim(path, `"`)
		}
		here, inside := underPrefix(path, prefix)
		if !inside {
			continue
		}

		row := GitFileChange{Path: here, Status: "M"}
		switch {
		case strings.Contains(code, "D"):
			row.Status = "D"
		case strings.Contains(code, "?"), strings.Contains(code, "A"):
			row.Status = "U"
		}
		if c, hit := counts[path]; hit {
			row.Added, row.Removed = c[0], c[1]
		} else if row.Status == "U" && countUntracked {
			row.Added = fileLineCount(filepath.Join(root, filepath.FromSlash(here)))
		}
		if i, seen := at[here]; seen {
			if out[i].Status != row.Status {
				out[i].Status = "M"
			}
			if row.Added+row.Removed > out[i].Added+out[i].Removed {
				out[i].Added, out[i].Removed = row.Added, row.Removed
			}
			continue
		}
		at[here] = len(out)
		out = append(out, row)
	}
	return out, nil
}

// errGitSlow is what a read that ran out of its budget says — the one error
// the room translates, because it is the one the user can do nothing about
// except wait.
var errGitSlow = errors.New("git took too long to answer")

// repoPrefix is where root sits inside its repository, as a forward-slashed
// path ending in "/" — empty when root is the repository itself, which is the
// ordinary case and costs one more git call to be sure of.
func repoPrefix(ctx context.Context, root string) string {
	raw, err := gitOut(ctx, root, "rev-parse", "--show-prefix")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(raw)
}

// cachedRepoPrefix is repoPrefix remembered per root. Where a project sits
// inside its repository does not change while the app runs, and every binding
// in this file shares one ten-second budget with the git it then runs — on a
// machine where a git process takes seconds to start (a scanner, a loaded
// disk), the lookup was the call that pushed the real one past the line.
var repoPrefixes sync.Map // root -> prefix

func cachedRepoPrefix(ctx context.Context, root string) string {
	if v, ok := repoPrefixes.Load(root); ok {
		return v.(string)
	}
	prefix := repoPrefix(ctx, root)
	if ctx.Err() == nil {
		repoPrefixes.Store(root, prefix)
	}
	return prefix
}

// underPrefix re-roots one porcelain path at the project, and says whether it
// belongs to the project at all.
//
// An untracked directory arrives as "sub/" and stays a directory: it is the one
// row that names a folder rather than a file, and the tree needs it that way to
// know every file under it is new.
func underPrefix(path, prefix string) (string, bool) {
	if prefix == "" {
		return path, true
	}
	if !strings.HasPrefix(path, prefix) {
		// The repository has changes elsewhere. True, and none of this
		// project's business.
		return "", false
	}
	return strings.TrimPrefix(path, prefix), true
}

// GitFileDiff is what one row unfolds into: the same git-style hunks the chat
// draws under an edit, for this file's difference from HEAD.
//
// Empty for anything that has no readable answer — outside a project, not a
// repo, a path that leaves the root, a binary file. The panel draws "no diff to
// show" rather than an error, because every one of those is an ordinary state
// of a working tree and none of them is the user's mistake.
func (a *App) GitFileDiff(path string) string {
	root, ok := a.gitRoot()
	if !ok {
		return ""
	}
	clean := strings.TrimSpace(path)
	if clean == "" {
		return ""
	}
	// The panel's own rows are the only source of these paths, but a binding is
	// a public door: a path that climbs out of the project is refused here
	// rather than trusted because of where it usually comes from.
	full := filepath.Join(root, filepath.FromSlash(clean))
	if rel, relErr := filepath.Rel(root, full); relErr != nil || strings.HasPrefix(rel, "..") {
		return ""
	}

	// HEAD's copy, or nothing when the file is new to git — `git show` fails
	// for a path HEAD never had, and "" is exactly what that means here.
	ctx, cancel := a.gitContext()
	defer cancel()
	before, _ := gitOut(ctx, root, "show", "HEAD:"+filepath.ToSlash(clean))

	after := ""
	if data, readErr := os.ReadFile(full); readErr == nil {
		if strings.ContainsRune(string(data), 0) {
			return "" // binary; hunks would be noise, not information
		}
		after = string(data)
	}
	return skill.FileDiff(clean, before, after)
}

// gitRoot is the project this panel reports on, and whether there is one at
// all. Unfocused mode has no project: home may well sit inside somebody's
// repository, and that repository's status is not this window's business.
func (a *App) gitRoot() (string, bool) {
	if !a.projectFocused {
		return "", false
	}
	root := a.cur().cfg.SandboxRoot
	if strings.TrimSpace(root) == "" {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return "", false
	}
	return root, true
}

// gitContext bounds every call this panel makes. The panel refreshes on a timer
// and again at the end of every turn, so a git that hangs — an index.lock left
// by another process, a repository on a network share — must not pile up
// processes that nothing on screen can reach. Ten seconds is far past any local
// `git status`, and a panel answering "nothing" beats an app leaking a git per
// refresh.
func (a *App) gitContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(a.engineCtx(), 10*time.Second)
}

func gitOut(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root, "-c", "core.quotepath=false"}, args...)...)
	proc.HideConsole(cmd)
	proc.KillOnCancel(cmd)
	raw, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// numstat maps path -> {added, removed} against HEAD, staged and unstaged
// together, which is what the working tree actually holds.
func numstat(ctx context.Context, root string) (map[string][2]int, error) {
	counts := map[string][2]int{}
	raw, err := gitOut(ctx, root, "diff", "--numstat", "HEAD")
	if err != nil {
		if ctx.Err() != nil {
			return counts, errGitSlow
		}
		// A repository with no HEAD yet — nothing committed — has no numstat
		// and that is not a failure: every row is an untracked file.
		return counts, nil
	}
	for _, line := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		cols := strings.SplitN(line, "\t", 3)
		if len(cols) != 3 {
			continue
		}
		// git writes "-" for both counts on a binary file. Zeroes are the right
		// answer there: there are no lines to have changed.
		added, _ := strconv.Atoi(cols[0])
		removed, _ := strconv.Atoi(cols[1])
		p := cols[2]
		if unquoted, err := strconv.Unquote(p); err == nil {
			p = unquoted
		} else {
			p = strings.Trim(p, `"`)
		}
		counts[p] = [2]int{added, removed}
	}
	return counts, nil
}

// untrackedCountCap is the most of an untracked file that is read to count its
// lines. Past it the answer is 0 rather than a slower answer: a file that size
// is a build product or a dump, and "+0" on its row costs nothing to be wrong
// about, where reading it whole on every two-second tick did.
const untrackedCountCap = 4 << 20

// fileLineCount is an untracked file's `+N`: its lines, since every one of
// them is an addition the moment it is added. A binary — a screenshot in
// .aetox-attachments, an .exe — has no lines and answers 0 after its first
// eight kilobytes rather than after all of them.
func fileLineCount(full string) int {
	f, err := os.Open(full)
	if err != nil {
		return 0
	}
	defer f.Close()
	if info, statErr := f.Stat(); statErr != nil || info.IsDir() || info.Size() == 0 || info.Size() > untrackedCountCap {
		return 0
	}
	head := make([]byte, 8<<10)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if bytes.IndexByte(head, 0) >= 0 {
		return 0
	}
	count := bytes.Count(head, []byte{'\n'})
	last := byte(0)
	if n > 0 {
		last = head[n-1]
	}
	rest, _ := io.ReadAll(f)
	if len(rest) > 0 {
		count += bytes.Count(rest, []byte{'\n'})
		last = rest[len(rest)-1]
	}
	if last != '\n' {
		count++
	}
	return count
}
