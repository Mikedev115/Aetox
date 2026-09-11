package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// gitIn runs one git command in root and fails the test on any error.
func gitIn(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// commitFile writes path and commits it as message, so a test can build the
// history it is about in a line per commit.
func commitFile(t *testing.T, root, path, content, message string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	gitIn(t, root, "add", "--", path)
	gitIn(t, root, "commit", "-q", "-m", message)
}

func TestGitLogNewestFirstWithGitsOwnCounts(t *testing.T) {
	root, a := repoAt(t)
	commitFile(t, root, "kept.txt", "one\nTWO\nthree\nfour\n", "feat: second")
	commitFile(t, root, "fresh.txt", "a\nb\n", "docs: third")

	page := a.GitLog("", 50)
	if page.More {
		t.Error("More = true on a three-commit history read in one page")
	}
	if len(page.Commits) != 3 {
		t.Fatalf("GitLog = %d rows, want 3", len(page.Commits))
	}
	if got := page.Commits[0]; got.Subject != "docs: third" || got.Files != 1 || got.Added != 2 || got.Removed != 0 {
		t.Errorf("newest = %+v, want docs: third, 1 file +2 -0", got)
	}
	if got := page.Commits[1]; got.Subject != "feat: second" || got.Added != 2 || got.Removed != 1 {
		t.Errorf("second = %+v, want +2 -1 (TWO swapped, four added)", got)
	}
	if got := page.Commits[2]; got.Subject != "first" || got.Merge {
		t.Errorf("oldest = %+v, want the root commit, not a merge", got)
	}
	for _, c := range page.Commits {
		if len(c.Hash) != 40 || !strings.HasPrefix(c.Hash, c.Short) || c.Author != "test" || !strings.Contains(c.At, "T") {
			t.Errorf("row %+v: hash/short/author/date not as git reports them", c)
		}
	}
}

// Paging is by the hash the last page ended on, not by an offset, so the page
// after a page begins exactly one commit past it — with no row twice and none
// skipped — and the last page says there is no more.
func TestGitLogPagesByHashWithoutOverlap(t *testing.T) {
	root, a := repoAt(t)
	for i := 0; i < 5; i++ {
		commitFile(t, root, "kept.txt", strings.Repeat("x\n", i+2), "chore: "+string(rune('a'+i)))
	}
	// six commits: first + a..e

	seen := map[string]bool{}
	before := ""
	pages := 0
	for {
		page := a.GitLog(before, 4)
		pages++
		if len(page.Commits) == 0 {
			t.Fatalf("page %d came back empty", pages)
		}
		for _, c := range page.Commits {
			if seen[c.Hash] {
				t.Errorf("commit %s appeared on two pages", c.Short)
			}
			seen[c.Hash] = true
		}
		if !page.More {
			break
		}
		before = page.Commits[len(page.Commits)-1].Hash
		if pages > 5 {
			t.Fatal("paging never ended")
		}
	}
	if len(seen) != 6 || pages != 2 {
		t.Errorf("walked %d commits in %d pages, want 6 in 2", len(seen), pages)
	}
	// Past the root there is nothing, and nothing is the answer — not an error.
	last := ""
	for h := range seen {
		if page := a.GitLog(h, 4); len(page.Commits) == 0 {
			last = h
		}
	}
	if last == "" {
		t.Error("no commit reported an empty page after it; the root commit should")
	}
}

func TestGitLogRefusesWhatIsNotAHash(t *testing.T) {
	_, a := repoAt(t)
	if page := a.GitLog("--all", 50); len(page.Commits) != 0 || page.Commits == nil {
		t.Errorf("GitLog(\"--all\") = %+v, want an empty (non-nil) page", page)
	}
}

// Every "not applicable" answers with an empty page and never with nil — the
// frontend does .length on it (§34).
func TestGitLogEmptyOutsideAProjectOrRepo(t *testing.T) {
	root, a := repoAt(t)
	a.projectFocused = false
	if page := a.GitLog("", 50); page.Commits == nil || len(page.Commits) != 0 {
		t.Errorf("unfocused GitLog = %+v, want empty", page)
	}
	_ = root
	b := seed(&Engine{cfg: config.Config{SandboxRoot: t.TempDir()}, projectFocused: true}, newConversation())
	if page := b.GitLog("", 50); page.Commits == nil || len(page.Commits) != 0 {
		t.Errorf("GitLog outside a repo = %+v, want empty", page)
	}
	if got := b.GitCommitChanges("0123456789abcdef"); got == nil || len(got) != 0 {
		t.Errorf("GitCommitChanges outside a repo = %+v, want empty", got)
	}
}

func TestGitCommitChangesUsesTheWorkingTreeLetters(t *testing.T) {
	root, a := repoAt(t)
	commitFile(t, root, "gone.txt", "bye\n", "add gone")
	// One commit that modifies, adds and deletes.
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nTWO\nthree\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "new.txt"), []byte("n\n"), 0o644); err != nil {
		if mkErr := os.MkdirAll(filepath.Join(root, "sub"), 0o755); mkErr != nil {
			t.Fatalf("mkdir: %v", mkErr)
		}
		if err := os.WriteFile(filepath.Join(root, "sub", "new.txt"), []byte("n\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	gitIn(t, root, "rm", "-q", "gone.txt")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "refactor: all three")

	head := a.GitLog("", 1).Commits[0]
	rows := map[string]GitFileChange{}
	for _, f := range a.GitCommitChanges(head.Hash) {
		rows[f.Path] = f
	}
	if len(rows) != 3 {
		t.Fatalf("GitCommitChanges = %+v, want three rows", rows)
	}
	if got := rows["kept.txt"]; got.Status != "M" || got.Added != 1 || got.Removed != 1 {
		t.Errorf("kept.txt = %+v, want M +1 -1", got)
	}
	if got := rows["sub/new.txt"]; got.Status != "U" || got.Added != 1 {
		t.Errorf("sub/new.txt = %+v, want U +1", got)
	}
	if got := rows["gone.txt"]; got.Status != "D" || got.Removed != 1 {
		t.Errorf("gone.txt = %+v, want D -1", got)
	}
}

func TestGitCommitFileDiffIsTheSameHunksTheChatDraws(t *testing.T) {
	root, a := repoAt(t)
	commitFile(t, root, "kept.txt", "one\nTWO\nthree\n", "feat: swap")
	commitFile(t, root, "fresh.txt", "a\nb\n", "feat: fresh")
	page := a.GitLog("", 2).Commits
	swap, fresh := page[1], page[0]

	got := a.GitCommitFileDiff(swap.Hash, "kept.txt")
	if !strings.HasPrefix(got, "+++ kept.txt\n@@ -1,3 +1,3 @@") {
		t.Fatalf("GitCommitFileDiff = %q, want the path then the file's own hunk numbering", got)
	}
	if !strings.Contains(got, "-two") || !strings.Contains(got, "+TWO") {
		t.Errorf("diff =\n%s\nwant the swap of line 2", got)
	}
	// A file the commit added has no "before": everything in it is an addition.
	if got := a.GitCommitFileDiff(fresh.Hash, "fresh.txt"); !strings.Contains(got, "+a") || strings.Contains(got, "\n-") {
		t.Errorf("added file =\n%s\nwant every line added", got)
	}
	// And nothing for a file that commit never touched, a bad hash, or a path
	// that leaves the project.
	if got := a.GitCommitFileDiff(fresh.Hash, "kept.txt"); got != "" {
		t.Errorf("untouched file = %q, want empty", got)
	}
	if got := a.GitCommitFileDiff("HEAD", "kept.txt"); got != "" {
		t.Errorf("a ref is not a hash: %q", got)
	}
	if got := a.GitCommitFileDiff(swap.Hash, "../../etc/passwd"); got != "" {
		t.Errorf("escaped the project: %q", got)
	}
}

func TestParseGitLogHandlesMergesAndBodylessRecords(t *testing.T) {
	raw := "\x1e" + strings.Join([]string{"aaaa", "aaa", "me", "2026-09-12T10:00:00+07:00", "p1 p2", "Merge branch x"}, "\x1f") + "\n" +
		"\x1e" + strings.Join([]string{"bbbb", "bbb", "me", "2026-09-12T09:00:00+07:00", "p1", "fix: one"}, "\x1f") + "\n" +
		" 2 files changed, 3 insertions(+), 1 deletion(-)\n" +
		"\x1e" + strings.Join([]string{"cccc", "ccc", "me", "2026-09-12T08:00:00+07:00", "", "first"}, "\x1f") + "\n" +
		" 1 file changed, 1 insertion(+)\n"
	rows := parseGitLog(raw)
	if len(rows) != 3 {
		t.Fatalf("parseGitLog = %d rows, want 3", len(rows))
	}
	if !rows[0].Merge || rows[0].Files != 0 {
		t.Errorf("merge row = %+v, want Merge with no stat", rows[0])
	}
	if rows[1].Merge || rows[1].Files != 2 || rows[1].Added != 3 || rows[1].Removed != 1 {
		t.Errorf("fix row = %+v, want 2 files +3 -1", rows[1])
	}
	if rows[2].Files != 1 || rows[2].Added != 1 || rows[2].Removed != 0 {
		t.Errorf("root row = %+v, want 1 file +1 -0 (no deletion clause)", rows[2])
	}
}

func TestRenameTargetSpellings(t *testing.T) {
	for in, want := range map[string]string{
		"old.txt => new.txt":         "new.txt",
		"a/{b => c}/d.txt":           "a/c/d.txt",
		"src/{ => lib}/x.go":         "src/lib/x.go",
		"plain.txt":                  "",
		"desktop/{old.go => new.go}": "desktop/new.go",
	} {
		if got := renameTarget(in); got != want {
			t.Errorf("renameTarget(%q) = %q, want %q", in, got, want)
		}
	}
}
