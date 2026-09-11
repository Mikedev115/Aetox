package skill

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// initGitRepo creates a minimal git repo with one commit in a temp dir.
func initGitRepo(t *testing.T) string {
	t.Helper()
	return initGitRepoIn(t, t.TempDir())
}

// initGitRepoIn is initGitRepo at a directory the test chose — a subfolder of
// a workspace, or a folder outside one.
func initGitRepoIn(t *testing.T, dir string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	return dir
}

func TestGitSkillStatus(t *testing.T) {
	dir := initGitRepo(t)
	s := &gitSkill{root: dir}
	out, err := s.Execute(context.Background(), Input{"args": []string{"status"}})
	if err != nil {
		t.Fatalf("git status: unexpected error: %v", err)
	}
	if !out.Success {
		t.Error("Success = false, want true")
	}
}

func TestGitSkillLog(t *testing.T) {
	dir := initGitRepo(t)
	s := &gitSkill{root: dir}
	out, err := s.Execute(context.Background(), Input{"args": []string{"log"}})
	if err != nil {
		t.Fatalf("git log: unexpected error: %v", err)
	}
	if !strings.Contains(out.Content, "init") {
		t.Errorf("Content = %q, want to contain commit message %q", out.Content, "init")
	}
}

func TestGitSkillRejectsUnsupportedAction(t *testing.T) {
	dir := initGitRepo(t)
	s := &gitSkill{root: dir}
	if _, err := s.Execute(context.Background(), Input{"args": []string{"push"}}); err == nil {
		t.Fatal("expected error for unsupported git action (push), got nil")
	}
	if _, err := s.Execute(context.Background(), Input{"args": []string{"commit"}}); err == nil {
		t.Fatal("expected error for unsupported git action (commit), got nil")
	}
}

func TestGitSkillBlocksUnsafeFlags(t *testing.T) {
	dir := initGitRepo(t)
	s := &gitSkill{root: dir}
	for _, args := range [][]string{
		{"status", "--git-dir=/etc"},
		{"status", "-C", "/etc"},
		{"log", "-c", "core.pager=evil"},
	} {
		if _, err := s.Execute(context.Background(), Input{"args": args}); err == nil {
			t.Errorf("args %v: expected unsafe-option error, got nil", args)
		}
	}
}

func TestGitSkillNotARepo(t *testing.T) {
	dir := t.TempDir() // no git init
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	s := &gitSkill{root: dir}
	if _, err := s.Execute(context.Background(), Input{"args": []string{"status"}}); err == nil {
		t.Fatal("expected error outside a git repository, got nil")
	}
}

// On Windows every `git diff` used to arrive with a "warning: LF will be
// replaced by CRLF" line per touched file glued to the front, because stderr
// was merged into stdout. The model reads that as part of the diff, and on a
// large change it can push the real content past the output line limit.
func TestGitExecuteCommandKeepsStderrOutOfSuccessfulOutput(t *testing.T) {
	name, args := "sh", []string{"-c", "echo OUT; echo ERR 1>&2"}
	if runtime.GOOS == "windows" {
		name, args = "cmd", []string{"/C", "echo OUT&echo ERR 1>&2"}
	}

	out, _, err := executeCommand(context.Background(), name, t.TempDir(), args...)
	if err != nil {
		t.Fatalf("executeCommand: unexpected error: %v", err)
	}
	if strings.Contains(out, "ERR") {
		t.Errorf("output = %q, want stderr excluded on success", out)
	}
	if !strings.Contains(out, "OUT") {
		t.Errorf("output = %q, want stdout kept", out)
	}

	// On failure stderr is the only place the reason lives, so it must appear.
	out, _, err = executeCommand(context.Background(), "git", t.TempDir(), "--no-such-flag")
	if err == nil {
		t.Fatal("expected an error from an invalid git flag, got nil")
	}
	if strings.TrimSpace(out) == "" || out == "(command failed)" {
		t.Errorf("output = %q, want git's stderr explaining the failure", out)
	}
}

// The desk is not a repository; the repository is a folder inside it. `path`
// takes git there, and its absence says how to get there.
func TestGitSkillPathRunsInsideWorkspace(t *testing.T) {
	root := t.TempDir()
	initGitRepoIn(t, filepath.Join(root, "repo"))
	s := &gitSkill{root: root}

	_, err := s.Execute(context.Background(), Input{"args": []string{"status"}})
	if err == nil {
		t.Fatal("status at a non-repository root: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "pass path") {
		t.Errorf("error = %q, want the remedy (pass path)", err)
	}

	out, err := s.Execute(context.Background(), Input{"args": []string{"log"}, "path": "repo"})
	if err != nil {
		t.Fatalf("git log with path: unexpected error: %v", err)
	}
	if !strings.Contains(out.Content, "init") {
		t.Errorf("Content = %q, want the repo's commit", out.Content)
	}
	if !strings.Contains(out.Command, "(in repo)") {
		t.Errorf("Command = %q, want it to name where the command ran", out.Command)
	}
}

// A repository outside the workspace is reachable only the way any file is:
// when the user added its folder. Refused otherwise — `path` is a door, not a
// hole.
func TestGitSkillPathOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	repo := initGitRepo(t)
	s := &gitSkill{root: root}

	if _, err := s.Execute(context.Background(), Input{"args": []string{"status"}, "path": repo}); err == nil {
		t.Fatal("path outside the workspace: expected refusal, got nil")
	}

	setSandboxPolicy(root, false, []string{repo}, nil)
	t.Cleanup(func() { setSandboxPolicy(root, false, nil, nil) })
	out, err := s.Execute(context.Background(), Input{"args": []string{"status"}, "path": repo})
	if err != nil {
		t.Fatalf("path in an added folder: unexpected error: %v", err)
	}
	if !out.Success {
		t.Error("Success = false, want true")
	}
}

// ExecuteTool is the model's door; path has to make it through.
func TestGitSkillExecuteToolPassesPath(t *testing.T) {
	root := t.TempDir()
	initGitRepoIn(t, filepath.Join(root, "repo"))
	s := &gitSkill{root: root}
	out, err := s.ExecuteTool(context.Background(), map[string]any{"action": "log", "path": "repo"})
	if err != nil {
		t.Fatalf("ExecuteTool with path: unexpected error: %v", err)
	}
	if !strings.Contains(out.Content, "init") {
		t.Errorf("Content = %q, want the repo's commit", out.Content)
	}
}
