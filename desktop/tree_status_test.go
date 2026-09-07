package main

// The file tree wearing git's answer (owner, 7 ก.ย.: "แสดงสถานะเหมือน git ด้วยได้ไหม
// แบบไฟล์ไหนแก้อะไรยังไง").
//
// Against a real repository, like git_worktree_test.go and for the same reason:
// every claim here is about the difference between HEAD and the disk, and the
// interesting cases — a folder git never looked inside, a project that is a
// subfolder of its repo — are exactly the ones a fake would get wrong.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

func treeByPath(nodes []TreeNode) map[string]TreeNode {
	out := map[string]TreeNode{}
	for _, n := range nodes {
		out[n.Path] = n
	}
	return out
}

func TestProjectTreeMarksWhatChangedAndHowMuch(t *testing.T) {
	root, a := repoAt(t)
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("one\nTWO\nthree\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "fresh.txt"), []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("add: %v", err)
	}

	nodes := treeByPath(a.ProjectTree())
	if got := nodes["kept.txt"]; got.Status != "M" || got.Added != 1 || got.Removed != 1 {
		t.Errorf("kept.txt = %+v, want M +1 -1", got)
	}
	if got := nodes["fresh.txt"]; got.Status != "U" {
		t.Errorf("fresh.txt = %+v, want U", got)
	}
}

// A folder git has never seen is one row of status output — it does not list
// what is inside something it does not track — so the files under it have to be
// marked by where they are rather than by being named.
func TestProjectTreeMarksFilesInsideABrandNewFolder(t *testing.T) {
	root, a := repoAt(t)
	if err := os.MkdirAll(filepath.Join(root, "slides"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "slides", "deck.html"), []byte("<h1>x</h1>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodes := treeByPath(a.ProjectTree())
	if got := nodes["slides/deck.html"]; got.Status != "U" {
		t.Errorf("slides/deck.html = %+v, want U — its whole folder is new", got)
	}
	if got := nodes["slides"]; got.Status != "U" {
		t.Errorf("the folder itself = %+v, want U", got)
	}
}

// The tree spends its life collapsed. A mark that only ever landed on files said
// nothing at all in that state, which is the screen the owner was looking at.
func TestProjectTreeRollsStatusUpToFolders(t *testing.T) {
	root, a := repoAt(t)
	deep := filepath.Join(root, "backend", "app")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "main.go"), []byte("package main\n"), 0o644); err != nil {
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
	run("add", "backend/app/main.go")
	run("commit", "-m", "second")
	if err := os.WriteFile(filepath.Join(deep, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodes := treeByPath(a.ProjectTree())
	for _, dir := range []string{"backend", "backend/app"} {
		if got := nodes[dir]; got.Status != "M" {
			t.Errorf("%s = %+v, want M — a changed file is somewhere under it", dir, got)
		}
		if got := nodes[dir]; got.Added != 0 || got.Removed != 0 {
			t.Errorf("%s carries counts %+v — a folder says which kind, not how much", dir, got)
		}
	}
}

func TestRollUpStatusTakesTheStrongestMark(t *testing.T) {
	nodes := []TreeNode{
		{Path: "src", Kind: "dir"},
		{Path: "src/new.go", Kind: "file", Status: "U"},
		{Path: "src/old.go", Kind: "file", Status: "M"},
		{Path: "docs", Kind: "dir"},
		{Path: "docs/note.md", Kind: "file", Status: "U"},
		{Path: "clean", Kind: "dir"},
		{Path: "clean/same.txt", Kind: "file"},
	}
	rollUpStatus(nodes, nil)
	by := treeByPath(nodes)
	if by["src"].Status != "M" {
		t.Errorf("src = %q, want M — an edit outranks a new file", by["src"].Status)
	}
	if by["docs"].Status != "U" {
		t.Errorf("docs = %q, want U", by["docs"].Status)
	}
	if by["clean"].Status != "" {
		t.Errorf("clean = %q, want no mark", by["clean"].Status)
	}
}

// A file that was deleted has no row to wear a mark — the walk reads the disk.
// Its folder is the only place the tree can say it happened, and saying nothing
// would make a deletion the one change the panel hides completely.
func TestProjectTreeMarksTheFolderOfADeletedFile(t *testing.T) {
	root, a := repoAt(t)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "old.md"), []byte("gone soon\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "stays.md"), []byte("here\n"), 0o644); err != nil {
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
	run("add", "docs")
	run("commit", "-m", "docs")
	if err := os.Remove(filepath.Join(root, "docs", "old.md")); err != nil {
		t.Fatal(err)
	}

	nodes := treeByPath(a.ProjectTree())
	if got := nodes["docs"]; got.Status != "D" {
		t.Errorf("docs = %+v, want D — a file under it is gone", got)
	}
	if _, ghost := nodes["docs/old.md"]; ghost {
		t.Error("the deleted file got a row of its own, which opens onto nothing")
	}
}

// A project opened at a subfolder of its repository. Porcelain prints paths from
// the repository root, so every one of them used to miss the tree's own
// vocabulary and the panel simply showed nothing.
func TestProjectTreeMarksInsideASubfolderProject(t *testing.T) {
	root, _ := repoAt(t)
	sub := filepath.Join(root, "web")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "index.html"), []byte("one\ntwo\n"), 0o644); err != nil {
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
	run("add", "web/index.html")
	run("commit", "-m", "web")
	if err := os.WriteFile(filepath.Join(sub, "index.html"), []byte("one\nTWO\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The change outside the project must not leak into it.
	if err := os.WriteFile(filepath.Join(root, "kept.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inSub := seed(&App{cfg: config.Config{SandboxRoot: sub}, projectFocused: true}, newConversation())
	nodes := treeByPath(inSub.ProjectTree())
	if got := nodes["index.html"]; got.Status != "M" || got.Added != 1 || got.Removed != 1 {
		t.Errorf("index.html = %+v, want M +1 -1", got)
	}
	if _, leaked := nodes["kept.txt"]; leaked {
		t.Error("a file from outside the project appeared in its tree")
	}
}
