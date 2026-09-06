package prompt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newGitRepo makes an empty repository with one commit in it, and returns both
// its root and a runner for further git commands. Every test below needs the
// same three lines, and a repository with no commit at all is a different
// animal — `rev-parse --abbrev-ref HEAD` answers nothing there, and gitLayer
// bails before it reads anything.
func newGitRepo(t *testing.T) (string, func(args ...string)) {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	git("init", "-b", "main")
	mustWrite(t, filepath.Join(root, "seed.txt"), "seed\n")
	git("add", "seed.txt")
	git("commit", "-m", "seed")
	return root, git
}

// touch stamps a path so ordering tests do not depend on how fast the machine
// running them writes files.
func touch(t *testing.T, path string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

// dirtyLines is the uncommitted block alone: the entries between the header and
// whatever follows them. Tests assert on order and on size, and both questions
// are about this section rather than about the layer's fixed furniture.
func dirtyLines(t *testing.T, layer string) []string {
	t.Helper()
	var out []string
	seen := false
	for _, line := range strings.Split(layer, "\n") {
		if strings.HasPrefix(line, "Uncommitted (") {
			seen = true
			continue
		}
		if !seen {
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			break
		}
		// The overflow line wears the same indent as an entry and is not one.
		// Counting it as an entry made the cap test accuse the code of
		// overspending by exactly the length of the line announcing the cap.
		if strings.HasPrefix(strings.TrimSpace(line), "...") {
			break
		}
		out = append(out, strings.TrimSpace(line))
	}
	return out
}

// The reason this file exists. Twelve paths in git's own order named whichever
// twelve sorted early; the list has to lead with what was touched last, or a
// tree with a hundred dirty files reports the alphabet instead of the work.
func TestGitLayerOrdersByMostRecentlyTouched(t *testing.T) {
	root, _ := newGitRepo(t)
	now := time.Now()
	for name, age := range map[string]time.Duration{
		"aaa_oldest.txt": 72 * time.Hour,
		"mmm_middle.txt": 2 * time.Hour,
		"zzz_newest.txt": time.Minute,
	} {
		path := filepath.Join(root, name)
		mustWrite(t, path, "dirty\n")
		touch(t, path, now.Add(-age))
	}

	lines := dirtyLines(t, gitLayer(root))
	var order []string
	for _, line := range lines {
		if i := strings.LastIndex(line, " "); i >= 0 {
			order = append(order, line[i+1:])
		}
	}
	want := []string{"zzz_newest.txt", "mmm_middle.txt", "aaa_oldest.txt"}
	if len(order) < len(want) {
		t.Fatalf("expected at least %d entries, got %v", len(want), order)
	}
	for i, name := range want {
		if order[i] != name {
			t.Errorf("entry %d is %q, want %q — full order %v", i, order[i], name, order)
		}
	}
}

// A deletion has no file left to stat. Ranking it by its parent directory keeps
// it near the top where a deletion belongs, instead of at the bottom with the
// entries nothing is known about.
func TestGitLayerRanksADeletionByItsParentDirectory(t *testing.T) {
	root, git := newGitRepo(t)
	stale := filepath.Join(root, "stale.txt")
	doomed := filepath.Join(root, "doomed.txt")
	mustWrite(t, stale, "one\n")
	mustWrite(t, doomed, "two\n")
	git("add", "stale.txt", "doomed.txt")
	git("commit", "-m", "two files")

	mustWrite(t, stale, "edited a long time ago\n")
	touch(t, stale, time.Now().Add(-48*time.Hour))
	if err := os.Remove(doomed); err != nil {
		t.Fatalf("remove: %v", err)
	}

	lines := dirtyLines(t, gitLayer(root))
	if len(lines) < 2 {
		t.Fatalf("expected both files as entries, got %v", lines)
	}
	if !strings.HasSuffix(lines[0], "doomed.txt") {
		t.Errorf("the deleted file should lead the list, got %v", lines)
	}
}

// The bug this fixes was silent: git C-quotes a non-ASCII path unless told not
// to, and the old layer printed that straight into the prompt. The model was
// handed octal escapes for a filename it could not read or quote back.
func TestGitLayerReadsNonAsciiPathsAsText(t *testing.T) {
	root, _ := newGitRepo(t)
	name := "รันเพื่อเช็คดูผล.md"
	mustWrite(t, filepath.Join(root, name), "ผล\n")

	layer := gitLayer(root)
	if strings.Contains(layer, `\340`) {
		t.Errorf("the layer carries C-quoted octal instead of the filename:\n%s", layer)
	}
	if !strings.Contains(layer, name) {
		t.Errorf("the layer does not name %q:\n%s", name, layer)
	}
}

// A hundred dirty paths is this owner's ordinary Monday. The list has to stop
// somewhere, stop by bytes because that is what the prompt pays in, and say how
// to get what it dropped.
func TestGitLayerCapsTheUncommittedListAndNamesTheWayOut(t *testing.T) {
	root, _ := newGitRepo(t)
	for i := 0; i < 200; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("some_moderately_long_filename_%03d.txt", i)), "x\n")
	}

	layer := gitLayer(root)
	spent := 0
	for _, line := range dirtyLines(t, layer) {
		spent += len(line) + 3 // the two-space indent and the newline
	}
	if spent > gitLayerMaxDirtyBytes {
		t.Errorf("uncommitted list spent %d B, cap is %d B", spent, gitLayerMaxDirtyBytes)
	}
	if !strings.Contains(layer, "more; run the git tool with action status") {
		t.Errorf("a truncated list must say how to get the rest:\n%s", layer)
	}
	if !strings.Contains(layer, "Uncommitted (200)") {
		t.Errorf("the header must still count every dirty path, not the ones shown:\n%s", layer)
	}
}

// budget_test.go's 16,000-byte ceiling cannot see this layer: it builds against
// a t.TempDir(), which is not a repository. This is the ceiling that holds it,
// and it is deliberately measured against THIS repository rather than a fixture
// — the layer's size is driven by real commit subjects and real path lengths,
// and a fixture would only ever confirm the size of the fixture.
func TestGitLayerStaysWithinItsBudget(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Skipf("cannot resolve the repository root: %v", err)
	}
	if info, err := os.Stat(filepath.Join(root, ".git")); err != nil || !info.IsDir() {
		t.Skip("not running inside a git checkout")
	}
	layer := gitLayer(root)
	if layer == "" {
		t.Skip("git unavailable")
	}
	t.Logf("git layer: %d B / ~%d tokens (ceiling %d B)", len(layer), len(layer)/4, gitLayerMaxBytes)
	if len(layer) > gitLayerMaxBytes {
		t.Errorf("the git layer is %d B, ceiling is %d B:\n%s", len(layer), gitLayerMaxBytes, layer)
	}
}
