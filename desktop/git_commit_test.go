package main

import (
	"os"
	"path/filepath"
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
	tree := a.GitWorkingTree()
	if len(tree) != 2 {
		t.Fatalf("expected 2 files in working tree, got %d", len(tree))
	}

	// Commit ONLY kept.txt
	if err := a.GitCommitFiles("chore: update kept", []string{"kept.txt"}); err != nil {
		t.Fatalf("GitCommitFiles failed: %v", err)
	}

	// Working tree must now ONLY have extra.txt (kept.txt was committed, extra.txt was NOT lost!)
	remaining := a.GitWorkingTree()
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
	clean := a.GitWorkingTree()
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
