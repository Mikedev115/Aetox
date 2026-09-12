package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGitCommitFilesSubset(t *testing.T) {
	root, a := repoAt(t)

	// Modify kept.txt
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nMODIFIED\nthree\n"), 0o644); err != nil {
		t.Fatalf("write kept.txt: %v", err)
	}

	// Create new file extra.txt
	if err := os.WriteFile(filepath.Join(root, "extra.txt"), []byte("new file\n"), 0o644); err != nil {
		t.Fatalf("write extra.txt: %v", err)
	}

	// Verify working tree has 2 files
	tree, _ := a.GitWorkingTree()
	if len(tree) != 2 {
		t.Fatalf("expected 2 files in working tree, got %d", len(tree))
	}

	// Commit ONLY kept.txt
	if err := a.GitCommitFiles("chore: update kept", []string{"kept.txt"}); err != nil {
		t.Fatalf("GitCommitFiles failed: %v", err)
	}

	// Working tree must now ONLY have extra.txt (kept.txt was committed, extra.txt was NOT lost!)
	remaining, _ := a.GitWorkingTree()
	if len(remaining) != 1 {
		t.Fatalf("expected 1 file remaining in working tree, got %d", len(remaining))
	}
	if remaining[0].Path != "extra.txt" {
		t.Errorf("expected extra.txt to remain uncommitted, got %s", remaining[0].Path)
	}

	// Commit extra.txt with empty message should fail
	if err := a.GitCommitFiles("   ", []string{"extra.txt"}); err == nil {
		t.Error("expected empty message to fail, got nil")
	}

	// Commit extra.txt
	if err := a.GitCommitFiles("feat: add extra", []string{"extra.txt"}); err != nil {
		t.Fatalf("GitCommitFiles extra failed: %v", err)
	}

	// Working tree must now be completely clean
	clean, _ := a.GitWorkingTree()
	if len(clean) != 0 {
		t.Errorf("expected clean working tree, got %d files", len(clean))
	}
}

func TestGitSuggestSplitCommitsEmpty(t *testing.T) {
	_, a := repoAt(t)

	// Clean tree should return empty slice (not nil)
	groups, err := a.GitSuggestSplitCommits()
	if err != nil {
		t.Fatalf("GitSuggestSplitCommits error: %v", err)
	}
	if groups == nil {
		t.Fatal("GitSuggestSplitCommits returned nil slice")
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups on clean tree, got %d", len(groups))
	}
}

// The chat's own attachments are the app's files under the project root, and
// thirteen of them rode into commits before this rule. Never proposed.
func TestIsDangerousPathHoldsOutTheAppsAttachments(t *testing.T) {
	for p, want := range map[string]bool{
		".aetox-attachments/20260912-041113.536/1789163035789-1.png": true,
		"sub/.aetox-attachments/x.png":                               true,
		"docs/attachments/x.png":                                     false,
		"kept.txt":                                                   false,
		".env":                                                       true,
	} {
		if got := isDangerousPath(p); got != want {
			t.Errorf("isDangerousPath(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestFallbackSplitGroups(t *testing.T) {
	tree := []GitFileChange{
		{Path: "desktop/git_commit.go", Status: "M"},
		{Path: "desktop/git_commit_test.go", Status: "U"},
		{Path: "docs/readme.md", Status: "M"},
		{Path: "root.txt", Status: "M"},
	}

	groups := fallbackSplitGroups(tree)
	if len(groups) != 3 {
		t.Fatalf("expected 3 directory groups, got %d", len(groups))
	}

	foundDesktop := false
	for _, g := range groups {
		if len(g.Files) == 2 && g.Files[0] == "desktop/git_commit.go" {
			foundDesktop = true
		}
	}
	if !foundDesktop {
		t.Errorf("expected desktop group with 2 files, got %+v", groups)
	}
}

// One git process for the whole tree, split per path — and the new file,
// which HEAD has no diff for, shown as an all-added file instead of nothing.
// A path with a space is the case the header split has to survive.
func TestGitDiffByFileOneProcessAndUntracked(t *testing.T) {
	root, a := repoAt(t)
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nMODIFIED\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "with space"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "with space", "a b.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", "with space/a b.txt")
	run("commit", "-m", "space")
	if err := os.WriteFile(filepath.Join(root, "with space", "a b.txt"), []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fresh.go"), []byte("package x\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tree, err := a.GitWorkingTree()
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 3 {
		t.Fatalf("tree: %+v", tree)
	}
	ctx, cancel := a.gitContext()
	defer cancel()
	diffs := gitDiffByFile(ctx, root, tree)

	if d := diffs["kept.txt"]; !strings.Contains(d, "+MODIFIED") || !strings.Contains(d, "-two") {
		t.Errorf("kept.txt diff:\n%s", d)
	}
	if d := diffs["kept.txt"]; strings.Contains(d, "\nindex ") {
		t.Errorf("index line not dropped:\n%s", d)
	}
	if d := diffs["with space/a b.txt"]; !strings.Contains(d, "+y") {
		t.Errorf("path with a space lost its diff: %q (have %v)", d, keys(diffs))
	}
	if d := diffs["fresh.go"]; !strings.HasPrefix(d, "(new file)\n+package x") {
		t.Errorf("untracked file not shown as added lines:\n%s", d)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestCleanCommitMessage(t *testing.T) {
	for in, want := range map[string]string{
		"```\nfeat(x): y\n\n- a\n```":        "feat(x): y\n\n- a",
		"```text\nfix(x): y\n```":            "fix(x): y",
		"\"docs: one line\"":                 "docs: one line",
		"Commit message:\nchore(a): b\n\n":   "chore(a): b",
		"feat(git): สอง\r\n\r\n- บรรทัด\r\n": "feat(git): สอง\n\n- บรรทัด",
	} {
		if got := cleanCommitMessage(in); got != want {
			t.Errorf("cleanCommitMessage(%q) = %q, want %q", in, got, want)
		}
	}
}

// The placeholder names its files and never says "update files".
func TestFallbackCommitMessageNamesTheFiles(t *testing.T) {
	msg := fallbackCommitMessage([]string{"desktop/a.go", "desktop/b.go"})
	if !strings.HasPrefix(msg, "chore(desktop): ") || strings.Contains(msg, "update files") {
		t.Errorf("subject: %q", msg)
	}
	if !strings.Contains(msg, "- desktop/a.go") || !strings.Contains(msg, "- desktop/b.go") {
		t.Errorf("files missing:\n%s", msg)
	}
	groups := fallbackSplitGroupsWithReason([]GitFileChange{{Path: "x.txt"}}, "no model")
	if groups[0].Source != gitSplitSourceFallback || groups[0].Reason != "no model" {
		t.Errorf("fallback not labelled: %+v", groups[0])
	}
}
