package engine

import (
	"github.com/Mikedev115/Aetox/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func emptyRepoAt(t *testing.T) (string, *Engine) {
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
	a := seed(&Engine{cfg: config.Config{SandboxRoot: root}, projectFocused: true}, newConversation())
	return root, a
}

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
	tree, err := a.GitWorkingTree()
	if err != nil {
		t.Fatalf("GitWorkingTree: %v", err)
	}
	if len(tree) != 2 {
		t.Fatalf("expected 2 files in working tree, got %d", len(tree))
	}

	// Commit ONLY kept.txt
	res, err := a.GitCommitFiles("chore: update kept", []string{"kept.txt"})
	if err != nil {
		t.Fatalf("GitCommitFiles failed: %v", err)
	}
	if res.Outcome != GitCommitOutcomeCommitted {
		t.Errorf("expected outcome committed, got %q", res.Outcome)
	}
	if res.Hash == "" {
		t.Errorf("expected non-empty commit hash on success")
	}

	// Working tree must now ONLY have extra.txt (kept.txt was committed, extra.txt was NOT lost!)
	remaining, err := a.GitWorkingTree()
	if err != nil {
		t.Fatalf("GitWorkingTree remaining: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 file remaining in working tree, got %d", len(remaining))
	}
	if remaining[0].Path != "extra.txt" {
		t.Errorf("expected extra.txt to remain uncommitted, got %s", remaining[0].Path)
	}

	// Commit extra.txt with empty message should fail
	if _, err := a.GitCommitFiles("   ", []string{"extra.txt"}); err == nil {
		t.Error("expected empty message to fail, got nil")
	}

	// Commit extra.txt
	resExtra, err := a.GitCommitFiles("feat: add extra", []string{"extra.txt"})
	if err != nil {
		t.Fatalf("GitCommitFiles extra failed: %v", err)
	}
	if resExtra.Outcome != GitCommitOutcomeCommitted {
		t.Errorf("expected outcome committed, got %q", resExtra.Outcome)
	}

	// Working tree must now be completely clean
	clean, err := a.GitWorkingTree()
	if err != nil {
		t.Fatalf("GitWorkingTree clean: %v", err)
	}
	if len(clean) != 0 {
		t.Errorf("expected clean working tree, got %d files", len(clean))
	}

	// Committing files with no changes should safely return outcome no_changes with nil error
	noChangeRes, err := a.GitCommitFiles("chore: no-op commit", []string{"kept.txt"})
	if err != nil {
		t.Fatalf("expected no-op GitCommitFiles to succeed without error, got: %v", err)
	}
	if noChangeRes.Outcome != GitCommitOutcomeNoChanges {
		t.Errorf("expected outcome no_changes, got %q", noChangeRes.Outcome)
	}
	if noChangeRes.Hash != "" {
		t.Errorf("expected empty hash for no_changes, got %q", noChangeRes.Hash)
	}
}

func TestGitCommitPreservesStagedIndex(t *testing.T) {
	root, a := repoAt(t)

	// User has a staged change on staged.txt
	if err := os.WriteFile(filepath.Join(root, "staged.txt"), []byte("staged content\n"), 0o644); err != nil {
		t.Fatalf("write staged.txt: %v", err)
	}
	runCmd := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	runCmd("add", "staged.txt")

	// User also has an untracked file
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("untracked content\n"), 0o644); err != nil {
		t.Fatalf("write untracked.txt: %v", err)
	}

	// User also has a modified kept.txt
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("kept modified\n"), 0o644); err != nil {
		t.Fatalf("write kept.txt: %v", err)
	}

	countBefore := runCmd("rev-list", "--count", "HEAD")

	// Commit ONLY kept.txt
	res, err := a.GitCommitFiles("chore: commit kept only", []string{"kept.txt"})
	if err != nil {
		t.Fatalf("GitCommitFiles failed: %v", err)
	}
	if res.Outcome != GitCommitOutcomeCommitted {
		t.Fatalf("expected committed, got %s", res.Outcome)
	}

	// Verify exactly 1 commit was created
	countAfter := runCmd("rev-list", "--count", "HEAD")
	if countAfter != "2" || countBefore != "1" {
		t.Fatalf("expected commit count to increase from 1 to 2, was %s -> %s", countBefore, countAfter)
	}

	// Verify staged.txt is STILL staged in the index!
	cachedDiff := runCmd("diff", "--cached", "--name-only")
	if cachedDiff != "staged.txt" {
		t.Errorf("expected staged.txt to remain staged, got cached diff: %q", cachedDiff)
	}

	// Verify untracked.txt is still untracked
	status := runCmd("status", "--porcelain")
	if !strings.Contains(status, "?? untracked.txt") {
		t.Errorf("expected untracked.txt to remain untracked, status:\n%s", status)
	}
	if !strings.Contains(status, "A  staged.txt") {
		t.Errorf("expected staged.txt to remain staged (A ), status:\n%s", status)
	}
}

func TestGitCommitUntrackedInSubsetPreservesIndex(t *testing.T) {
	root, a := repoAt(t)

	// User has a staged change on staged.txt
	if err := os.WriteFile(filepath.Join(root, "staged.txt"), []byte("staged content\n"), 0o644); err != nil {
		t.Fatalf("write staged.txt: %v", err)
	}
	runCmd := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	runCmd("add", "staged.txt")

	// User has untracked file new_feature.txt
	if err := os.WriteFile(filepath.Join(root, "new_feature.txt"), []byte("new feature\n"), 0o644); err != nil {
		t.Fatalf("write new_feature.txt: %v", err)
	}

	// Commit the untracked file new_feature.txt
	res, err := a.GitCommitFiles("feat: add new feature", []string{"new_feature.txt"})
	if err != nil {
		t.Fatalf("GitCommitFiles failed: %v", err)
	}
	if res.Outcome != GitCommitOutcomeCommitted {
		t.Fatalf("expected committed, got %s", res.Outcome)
	}

	// Verify staged.txt is STILL staged in the index!
	cachedDiff := runCmd("diff", "--cached", "--name-only")
	if cachedDiff != "staged.txt" {
		t.Errorf("expected staged.txt to remain staged, got cached diff: %q", cachedDiff)
	}

	// Verify new_feature.txt was committed
	log := runCmd("log", "-n", "1", "--name-only", "--oneline")
	if !strings.Contains(log, "new_feature.txt") {
		t.Errorf("expected commit to contain new_feature.txt, got log:\n%s", log)
	}
}

func TestGitCommitPartiallyStagedFilePreserved(t *testing.T) {
	root, a := repoAt(t)

	runCmd := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	// Modify line 1 of kept.txt and stage it
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one-staged\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("write kept.txt: %v", err)
	}
	runCmd("add", "kept.txt")

	// Modify line 3 of kept.txt (unstaged) -> now kept.txt is MM (partially staged)
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one-staged\ntwo\nthree-unstaged\n"), 0o644); err != nil {
		t.Fatalf("write kept.txt: %v", err)
	}

	// Create and modify another file other.txt
	if err := os.WriteFile(filepath.Join(root, "other.txt"), []byte("other content\n"), 0o644); err != nil {
		t.Fatalf("write other.txt: %v", err)
	}

	// Commit other.txt
	res, err := a.GitCommitFiles("feat: commit other", []string{"other.txt"})
	if err != nil {
		t.Fatalf("GitCommitFiles failed: %v", err)
	}
	if res.Outcome != GitCommitOutcomeCommitted {
		t.Fatalf("expected committed, got %s", res.Outcome)
	}

	// Verify kept.txt is STILL partially staged:
	// 1) Cached diff has one-staged
	cachedDiff := runCmd("diff", "--cached", "kept.txt")
	if !strings.Contains(cachedDiff, "+one-staged") || !strings.Contains(cachedDiff, "-one") {
		t.Errorf("expected staged hunk preserved in kept.txt, got:\n%s", cachedDiff)
	}
	if strings.Contains(cachedDiff, "three-unstaged") {
		t.Errorf("unstaged hunk leaked into cached diff:\n%s", cachedDiff)
	}

	// 2) Unstaged diff has three-unstaged
	unstagedDiff := runCmd("diff", "kept.txt")
	if !strings.Contains(unstagedDiff, "+three-unstaged") || !strings.Contains(unstagedDiff, "-three") {
		t.Errorf("expected unstaged hunk preserved in kept.txt, got:\n%s", unstagedDiff)
	}
}

func TestGitCommitRootCommitPreservesStaged(t *testing.T) {
	root, a := emptyRepoAt(t)

	runCmd := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	// Create first.txt and second.txt in empty repo
	if err := os.WriteFile(filepath.Join(root, "first.txt"), []byte("first\n"), 0o644); err != nil {
		t.Fatalf("write first.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "second.txt"), []byte("second\n"), 0o644); err != nil {
		t.Fatalf("write second.txt: %v", err)
	}

	// Stage second.txt
	runCmd("add", "second.txt")

	// Commit first.txt (which is untracked) as the root commit
	res, err := a.GitCommitFiles("initial: commit first", []string{"first.txt"})
	if err != nil {
		t.Fatalf("GitCommitFiles failed: %v", err)
	}
	if res.Outcome != GitCommitOutcomeCommitted {
		t.Fatalf("expected committed, got %s", res.Outcome)
	}

	// Verify second.txt is STILL staged in index!
	status := runCmd("status", "--porcelain")
	if !strings.Contains(status, "A  second.txt") {
		t.Errorf("expected second.txt to remain staged (A ), status:\n%s", status)
	}

	// Verify first.txt was committed
	log := runCmd("log", "--oneline")
	if !strings.Contains(log, "initial: commit first") {
		t.Errorf("expected initial commit in log, got:\n%s", log)
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

func TestGitCommitRecoversWhenGitReportsErrorButHeadChanged(t *testing.T) {
	root, a := repoAt(t)

	// Create a wrapper git script that executes git commands normally but returns exit code 1
	// specifically when the "commit" subcommand is executed (simulating a process abnormality
	// or hook failure after commit creation).
	tmpDir := t.TempDir()
	var wrapperPath string
	if runtime.GOOS == "windows" {
		wrapperPath = filepath.Join(tmpDir, "git-wrapper.bat")
		script := "@echo off\r\ngit %*\r\nset CODE=%ERRORLEVEL%\r\nfor %%a in (%*) do (\r\n  if \"%%a\"==\"commit\" exit /b 1\r\n)\r\nexit /b %CODE%\r\n"
		if err := os.WriteFile(wrapperPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write wrapper: %v", err)
		}
	} else {
		wrapperPath = filepath.Join(tmpDir, "git-wrapper.sh")
		script := "#!/bin/sh\ngit \"$@\"\ncode=$?\nfor arg in \"$@\"; do\n  if [ \"$arg\" = \"commit\" ]; then\n    exit 1\n  fi\ndone\nexit $code\n"
		if err := os.WriteFile(wrapperPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write wrapper: %v", err)
		}
	}

	oldGitBin := gitBin
	gitBin = wrapperPath
	t.Cleanup(func() {
		gitBin = oldGitBin
	})

	// Modify kept.txt
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nMODIFIED_WITH_HOOK\nthree\n"), 0o644); err != nil {
		t.Fatalf("write kept.txt: %v", err)
	}

	res, err := a.GitCommitFiles("chore: commit with failing hook", []string{"kept.txt"})
	if err != nil {
		t.Fatalf("expected commit to succeed despite post-commit hook failure, got err: %v", err)
	}
	if res.Outcome != GitCommitOutcomeCommitted {
		t.Fatalf("expected outcome committed, got %s", res.Outcome)
	}
	if res.Hash == "" {
		t.Fatal("expected non-empty commit hash")
	}
	if res.Warning == "" {
		t.Fatal("expected warning indicating git process reported abnormality")
	}

	// Verify HEAD actually updated in git
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	head := strings.TrimSpace(string(out))
	if head != res.Hash {
		t.Fatalf("expected HEAD %s to match result hash %s", head, res.Hash)
	}
}

func TestGitCommitReportsErrorWhenGitErrorsAndHeadUnchanged(t *testing.T) {
	root, a := repoAt(t)

	// Add failing pre-commit hook (which aborts before commit is made)
	hookDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	hookPath := filepath.Join(hookDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}
	_ = os.WriteFile(filepath.Join(hookDir, "pre-commit.bat"), []byte("@exit /b 1\r\n"), 0o755)

	headBeforeCmd := exec.Command("git", "rev-parse", "HEAD")
	headBeforeCmd.Dir = root
	outBefore, _ := headBeforeCmd.CombinedOutput()
	headBefore := strings.TrimSpace(string(outBefore))

	// Modify kept.txt
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nMODIFIED_BLOCKED\nthree\n"), 0o644); err != nil {
		t.Fatalf("write kept.txt: %v", err)
	}

	res, err := a.GitCommitFiles("chore: commit blocked by pre-commit", []string{"kept.txt"})
	if err == nil {
		t.Fatal("expected pre-commit hook failure to return error, got nil")
	}
	if res.Outcome == GitCommitOutcomeCommitted {
		t.Fatalf("expected outcome not committed, got %s", res.Outcome)
	}

	// Verify HEAD did not change
	headAfterCmd := exec.Command("git", "rev-parse", "HEAD")
	headAfterCmd.Dir = root
	outAfter, _ := headAfterCmd.CombinedOutput()
	headAfter := strings.TrimSpace(string(outAfter))
	if headAfter != headBefore {
		t.Fatalf("expected HEAD to remain %s, but changed to %s", headBefore, headAfter)
	}
}
