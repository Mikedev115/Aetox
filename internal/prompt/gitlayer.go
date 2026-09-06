package prompt

// The git snapshot folded into a focused session's prompt — see the call site
// in BuildWithReport for why it sits where it sits. Everything here is capped
// and labeled a snapshot: the prompt is built at bootstrap and does not track
// the tree, and a snapshot that presents itself as live would be the one kind
// of wrong this layer must never be. The git tool stays the live answer.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/proc"
)

const (
	// gitLayerTimeout bounds the two commands together. Bootstrap runs once per
	// session and already does file IO; two local git reads answer inside a
	// couple of hundred milliseconds, and a repository where they do not — a
	// network mount, a corrupt index — costs the layer, not the session.
	//
	// Measured 2026-09-06 on this repository, 127 dirty paths, warm cache:
	// `status --porcelain -z` took 112 ms, and the whole stat sweep that orders
	// it took 0.5 ms. The cost is the git process, not the work this file does
	// with what it returns — which is why widening the list below is nearly
	// free, and why nothing here tries to be clever about doing less of it.
	gitLayerTimeout = 2 * time.Second

	// gitLayerMaxDirtyBytes is how much of the uncommitted list is spent, in
	// bytes rather than in a count of files.
	//
	// It used to be twelve files. Twelve is a fine number for a tidy tree and a
	// bad one here: this owner's normal working state is over a hundred dirty
	// paths that sit for weeks, so twelve files chosen in git's own order meant
	// twelve paths that happened to sort early. Measured on 2026-09-06 the list
	// was ARCHITECTURE.md and eleven files starting "desktop/browser_", not one
	// of them the work in progress — while fourteen of the twenty most recently
	// touched paths, two of them brand-new files no commit has ever seen, were
	// invisible. The agent could not know what was in play, so it swept the
	// project to find out.
	//
	// A byte cap rather than a file cap because paths are not one size, and what
	// has to be protected is the prompt, which pays in bytes. Claude Code caps
	// the same block at 2,000 characters; this is a little under that, because
	// ordering by mtime means the paths that survive the cut are the ones that
	// matter, which was never true of the old cut.
	//
	// NOT covered by budget_test.go's 16,000-byte ceiling: that test builds
	// against a t.TempDir(), which is not a repository, so this layer is absent
	// from every number it has ever logged. TestGitLayerStaysWithinItsBudget
	// holds this one instead.
	gitLayerMaxDirtyBytes = 1500

	// gitLayerMaxBytes is the ceiling for the whole layer, held by
	// TestGitLayerStaysWithinItsBudget for the reason written above: this is
	// the only layer budget_test.go cannot weigh.
	//
	// It is larger than gitLayerMaxDirtyBytes by more than the branch line
	// costs, because the five recent commits are the other variable half and
	// this repository writes long Thai commit subjects — Thai spends three
	// bytes a character, so five of them run past what five English subjects
	// would.
	//
	// Measured 2026-09-06 against this repository: 2,860 B ≈ 715 tokens, of
	// which the capped dirty list is 1,500 and the five commit subjects are
	// most of the rest. Ceiling set at 3,200, about 12% of headroom, the same
	// shape of margin budget_test.go keeps.
	//
	// If this ever trips on the commit half rather than the dirty half, the fix
	// is to cap a subject's length, not to raise this number — the dirty list
	// is bounded and the commits are not, which is the asymmetry to close.
	gitLayerMaxBytes = 3200
)

// dirtyPath is one uncommitted entry, with the timestamp used to rank it.
type dirtyPath struct {
	// code is porcelain's two-column XY status, kept verbatim so the block
	// reads like `git status --short` and needs no key.
	code string
	path string
	// mod orders the list. Zero for an entry nothing could be stat'd for, which
	// sorts last: no timestamp is not evidence of an old change, it is an
	// absence of evidence, and the bottom is where absence belongs.
	mod time.Time
}

func gitLayer(root string) string {
	if info, err := os.Stat(filepath.Join(root, ".git")); err != nil || !info.IsDir() {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitLayerTimeout)
	defer cancel()
	run := func(args ...string) string {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
		proc.HideConsole(cmd)
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}

	branch := run("rev-parse", "--abbrev-ref", "HEAD")
	if branch == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## Git, as this session opened\n\n")
	b.WriteString("A snapshot, not a feed — the tree may have moved since; the git tool answers live.\n")
	fmt.Fprintf(&b, "Branch: %s\n", branch)
	writeDirtyPaths(&b, readDirtyPaths(ctx, root))
	if log := run("log", "--oneline", "-5"); log != "" {
		b.WriteString("Recent commits:\n")
		for _, line := range strings.Split(log, "\n") {
			b.WriteString("  " + strings.TrimRight(line, "\r") + "\n")
		}
	}
	return b.String()
}

// readDirtyPaths returns the working tree's uncommitted entries, newest first.
//
// Three flags carry the whole difference between this and the plain
// `status --porcelain` it replaces, and each one is load-bearing:
//
//   - -z separates records with NUL and, because it does, stops git quoting
//     paths at all. Without it a Thai filename arrives C-quoted as
//     "\340\270\243\340\270\261..." — which the old layer printed straight into
//     the prompt, handing the model octal escapes for a name it could neither
//     read nor quote back. It also broke the stat below, since that string is
//     not a path on disk.
//   - core.quotepath=false is redundant while -z is here, and is kept as the
//     belt to its braces: the day somebody swaps -z back out for readability,
//     the quoting bug must not silently return with it.
//   - --no-optional-locks keeps a plain status from taking index.lock to
//     refresh the index. It costs nothing measurable and it stops this read
//     from racing the owner's own git in the terminal next door.
func readDirtyPaths(ctx context.Context, root string) []dirtyPath {
	cmd := exec.CommandContext(ctx, "git", "-C", root,
		"-c", "core.quotepath=false", "--no-optional-locks",
		"status", "--porcelain", "-z")
	proc.HideConsole(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	records := strings.Split(string(out), "\x00")
	var dirty []dirtyPath
	for i := 0; i < len(records); i++ {
		rec := records[i]
		// "XY PATH" — anything shorter is the trailing empty record.
		if len(rec) < 4 {
			continue
		}
		entry := dirtyPath{code: rec[:2], path: rec[3:]}
		// A rename or copy spends two records: the new path here, the original
		// in the next one. Consume it, or the original is read as an entry of
		// its own wearing the following entry's status code.
		if entry.code[0] == 'R' || entry.code[0] == 'C' {
			i++
		}
		entry.mod = pathModTime(root, entry.path)
		dirty = append(dirty, entry)
	}

	// Stable, so entries sharing a timestamp keep git's own path order. Two
	// builds of the same tree then produce byte-identical prompts, which is
	// what the provider's prefix cache matches on.
	sort.SliceStable(dirty, func(i, j int) bool { return dirty[i].mod.After(dirty[j].mod) })
	return dirty
}

// pathModTime is when this entry last moved.
//
// The fallback is the point. A deleted file cannot be stat'd, and sorting every
// deletion to the bottom would hide the loudest change a tree can carry. Its
// parent directory, though, is stamped by the removal itself, on NTFS and on
// POSIX alike — so the directory answers a question the file no longer can, and
// answers it with roughly the right time rather than with nothing.
//
// The same fallback covers the untracked-directory entries porcelain folds into
// one row ending in "/": stat lands on the directory either way, and a
// directory's own mtime moves when files are added to or removed from it.
func pathModTime(root, path string) time.Time {
	full := filepath.Join(root, path)
	if info, err := os.Stat(full); err == nil {
		return info.ModTime()
	}
	if info, err := os.Stat(filepath.Dir(full)); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

// writeDirtyPaths spends gitLayerMaxDirtyBytes on the head of the list and says
// what it did with the rest.
//
// The overflow line names the tool rather than only counting what was dropped.
// A count alone tells the model that it does not have the whole picture and
// nothing about how to get it, which is a good way to be told to guess; Claude
// Code's equivalent block ends by naming the command, and it is right to.
func writeDirtyPaths(b *strings.Builder, dirty []dirtyPath) {
	if len(dirty) == 0 {
		b.WriteString("Working tree clean.\n")
		return
	}
	fmt.Fprintf(b, "Uncommitted (%d), most recently touched first:\n", len(dirty))
	spent := 0
	for i, entry := range dirty {
		line := "  " + entry.code + " " + entry.path + "\n"
		// Always render one, however long its path: a list that spent its whole
		// budget saying it had no room is worse than a list of one.
		if i > 0 && spent+len(line) > gitLayerMaxDirtyBytes {
			fmt.Fprintf(b, "  ... and %d more; run the git tool with action status for the whole list.\n", len(dirty)-i)
			return
		}
		b.WriteString(line)
		spent += len(line)
	}
}
