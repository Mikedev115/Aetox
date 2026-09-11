package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// repoAt builds a real repository with one committed file, because every claim
// this panel makes is about the difference between HEAD and the disk, and a
// fake of that is a fake of the whole feature.
func repoAt(t *testing.T) (string, *App) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	run("add", "kept.txt")
	run("commit", "-m", "first")

	a := seed(&App{cfg: config.Config{SandboxRoot: root}, projectFocused: true}, newConversation())
	return root, a
}

func TestGitWorkingTreeCountsWhatChanged(t *testing.T) {
	root, a := repoAt(t)
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nTWO\nthree\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "fresh.txt"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("add: %v", err)
	}

	rows := map[string]GitFileChange{}
	tree, err := a.GitWorkingTree()
	if err != nil {
		t.Fatalf("GitWorkingTree: %v", err)
	}
	for _, f := range tree {
		rows[f.Path] = f
	}
	if len(rows) != 2 {
		t.Fatalf("GitWorkingTree() = %+v, want two rows", rows)
	}
	if got := rows["kept.txt"]; got.Status != "M" || got.Added != 1 || got.Removed != 1 {
		t.Errorf("kept.txt = %+v, want M +1 -1", got)
	}
	// An untracked file has no numstat row to read, so its own length is the
	// addition — which is what git reports the moment it is added.
	if got := rows["fresh.txt"]; got.Status != "U" || got.Added != 2 || got.Removed != 0 {
		t.Errorf("fresh.txt = %+v, want U +2 -0", got)
	}
}

func TestGitWorkingTreeCleanTreeHasNoRows(t *testing.T) {
	_, a := repoAt(t)
	if got, err := a.GitWorkingTree(); err != nil || len(got) != 0 {
		t.Errorf("GitWorkingTree() on a clean tree = %+v, want empty", got)
	}
}

// Unfocused mode roams the machine, and home may well sit inside somebody's
// repository. That repository's status is not this window's business.
func TestGitWorkingTreeEmptyWhenUnfocused(t *testing.T) {
	root, a := repoAt(t)
	a.projectFocused = false
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if got, err := a.GitWorkingTree(); err != nil || len(got) != 0 {
		t.Errorf("GitWorkingTree() unfocused = %+v, want empty", got)
	}
}

func TestGitWorkingTreeOutsideARepo(t *testing.T) {
	a := seed(&App{cfg: config.Config{SandboxRoot: t.TempDir()}, projectFocused: true}, newConversation())
	if got, err := a.GitWorkingTree(); err != nil || len(got) != 0 {
		t.Errorf("GitWorkingTree() outside a repo = %+v, want empty", got)
	}
}

func TestGitFileDiffShowsTheChangedLine(t *testing.T) {
	root, a := repoAt(t)
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nTWO\nthree\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}

	got := a.GitFileDiff("kept.txt")
	if !strings.HasPrefix(got, "+++ kept.txt\n@@ ") {
		t.Fatalf("GitFileDiff = %q, want the path then a hunk", got)
	}
	if !strings.Contains(got, "-two") || !strings.Contains(got, "+TWO") {
		t.Errorf("GitFileDiff =\n%s\nwant the swap of line 2", got)
	}
	// Line-numbered from the file, which is the whole reason to show hunks
	// rather than the counts the row already carries.
	if !strings.Contains(got, "@@ -1,3 +1,3 @@") {
		t.Errorf("hunk header missing the file's own numbering:\n%s", got)
	}
}

// A file HEAD never had has no `git show` answer, and "" is exactly what that
// means: everything in it is an addition.
func TestGitFileDiffOnANewFileIsAllAdded(t *testing.T) {
	root, a := repoAt(t)
	if err := os.WriteFile(filepath.Join(root, "fresh.txt"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("add: %v", err)
	}
	got := a.GitFileDiff("fresh.txt")
	if !strings.Contains(got, "+a") || !strings.Contains(got, "+b") || strings.Contains(got, "\n-") {
		t.Errorf("GitFileDiff on a new file =\n%s\nwant every line added", got)
	}
}

// A binding is a public door. A path that climbs out of the project is refused
// here rather than trusted because of where it usually comes from.
func TestGitFileDiffRefusesAPathOutsideTheProject(t *testing.T) {
	_, a := repoAt(t)
	if got := a.GitFileDiff("../../../etc/passwd"); got != "" {
		t.Errorf("GitFileDiff escaped the project: %q", got)
	}
}

func TestGitFileDiffEmptyWhenNothingChanged(t *testing.T) {
	_, a := repoAt(t)
	if got := a.GitFileDiff("kept.txt"); got != "" {
		t.Errorf("GitFileDiff on an unchanged file = %q, want empty", got)
	}
}

// A screenshot dropped into the project is an untracked file with no lines in
// it, and it used to be read whole on every tick to be told so. Now the first
// eight kilobytes say "binary" and the answer is 0 — and a file past the cap
// is 0 without being read at all.
func TestGitWorkingTreeDoesNotCountBinaryOrHugeUntrackedFiles(t *testing.T) {
	root, a := repoAt(t)
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 4096)...)
	if err := os.WriteFile(filepath.Join(root, "shot.png"), png, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	big := strings.Repeat("line\n", (untrackedCountCap/5)+10)
	if err := os.WriteFile(filepath.Join(root, "dump.txt"), []byte(big), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "small.txt"), []byte("a\nb\nc"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	tree, err := a.GitWorkingTree()
	if err != nil {
		t.Fatalf("GitWorkingTree: %v", err)
	}
	rows := map[string]GitFileChange{}
	for _, f := range tree {
		rows[f.Path] = f
	}
	if got := rows["shot.png"]; got.Status != "U" || got.Added != 0 {
		t.Errorf("shot.png = %+v, want U +0 (binary has no lines)", got)
	}
	if got := rows["dump.txt"]; got.Status != "U" || got.Added != 0 {
		t.Errorf("dump.txt = %+v, want U +0 (past the cap, not read)", got)
	}
	// An unterminated last line still counts as one.
	if got := rows["small.txt"]; got.Added != 3 {
		t.Errorf("small.txt = %+v, want +3", got)
	}
}

// A read that ran out of its budget is an error, not a clean tree: the room
// keeps what it had and says git was slow, rather than telling the user that
// fifty-eight changed files are nothing.
func TestWorkingTreeReportsARunOutBudgetRatherThanACleanTree(t *testing.T) {
	root, _ := repoAt(t)
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rows, err := workingTree(ctx, root, true)
	if err == nil {
		t.Fatalf("workingTree on a spent context = %+v, nil; want an error", rows)
	}
	if !errors.Is(err, errGitSlow) {
		t.Errorf("err = %v, want errGitSlow", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %+v, want none beside the error", rows)
	}
}

// An index that another process left behind says a file is gone while the
// disk says it is here: porcelain prints `D ` and `??` for the same path. The
// pane keys its rows by path, and a duplicate key is a list that does not draw
// — so the two lines are one row, and the row says the file changed.
func TestGitWorkingTreeOneRowWhenIndexAndDiskDisagree(t *testing.T) {
	root, a := repoAt(t)
	gitIn(t, root, "rm", "-q", "--cached", "kept.txt")
	tree, err := a.GitWorkingTree()
	if err != nil {
		t.Fatalf("GitWorkingTree: %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("GitWorkingTree = %+v, want one row for kept.txt", tree)
	}
	if tree[0].Path != "kept.txt" || tree[0].Status != "M" {
		t.Errorf("row = %+v, want kept.txt M", tree[0])
	}
}
