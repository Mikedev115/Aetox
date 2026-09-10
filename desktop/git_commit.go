package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/proc"
)

// GitCommitGroup represents one atomic commit proposal suggested by AI.
type GitCommitGroup struct {
	Title   string   `json:"title"`
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

// GitCommitFiles stages and commits the given files with the specified message.
// If files is empty, it commits all changed files in the working tree.
func (a *App) GitCommitFiles(message string, files []string) error {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return errors.New("commit message cannot be empty")
	}
	root, ok := a.gitRoot()
	if !ok {
		return errors.New("no git repository focused")
	}

	ctx, cancel := a.gitContext()
	defer cancel()

	if len(files) == 0 {
		// Stage everything
		if _, err := gitOut(ctx, root, "add", "-A"); err != nil {
			return fmt.Errorf("git add -A: %w", err)
		}
	} else {
		// Reset any previously staged changes so only the requested files are committed
		if _, err := gitOut(ctx, root, "rev-parse", "--verify", "HEAD"); err == nil {
			_, _ = gitOut(ctx, root, "reset", "HEAD")
		}

		addArgs := append([]string{"add", "--"}, files...)
		if _, err := gitOut(ctx, root, addArgs...); err != nil {
			return fmt.Errorf("git add: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx, "git", "-C", root, "-c", "core.quotepath=false", "commit", "-m", trimmed)
	proc.HideConsole(cmd)
	proc.KillOnCancel(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" {
			return fmt.Errorf("git commit: %s", outStr)
		}
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

// GitSuggestCommitMessage uses the active model to generate a commit message for the given files.
func (a *App) GitSuggestCommitMessage(files []string) (string, error) {
	root, ok := a.gitRoot()
	if !ok {
		return "", errors.New("no git repository focused")
	}

	ctx, cancel := a.gitContext()
	defer cancel()

	p, modelName, err := a.oneShotProvider()
	if err != nil {
		return "", fmt.Errorf("provider unavailable: %w", err)
	}

	// Recent commit messages to mimic repository style and language
	recentLog, _ := gitOut(ctx, root, "log", "-n", "5", "--oneline")

	var diffSummary strings.Builder
	for _, f := range files {
		diffSummary.WriteString(fmt.Sprintf("\n--- %s ---\n", f))
		diff, _ := gitOut(ctx, root, "diff", "HEAD", "--", f)
		lines := strings.Split(diff, "\n")
		if len(lines) > 40 {
			diff = strings.Join(lines[:40], "\n") + "\n... (truncated)"
		}
		diffSummary.WriteString(diff)
	}

	prompt := fmt.Sprintf(`You are a Git commit message generator.
Write a single concise, high quality Conventional Commit message (e.g. feat(...), fix(...), style(...)) summarizing the code changes below.
Follow the language and phrasing style of the repository's recent commits if available.
Do NOT write markdown formatting, quotes, or explanations — output ONLY the commit message string itself.

Recent commits:
%s

Changes:
%s`, strings.TrimSpace(recentLog), diffSummary.String())

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleUser, Content: prompt},
		},
		MaxTokens: 120,
	}

	resp, err := p.Complete(ctx, req)
	if err != nil {
		return "", err
	}
	msg := strings.TrimSpace(resp.Text)
	msg = strings.Trim(msg, "`\"'")
	return msg, nil
}

const gitSplitToolJSON = `{
  "type": "function",
  "function": {
    "name": "propose_split_commits",
    "description": "Group changed files into atomic, cohesive Git commits.",
    "parameters": {
      "type": "object",
      "required": ["groups"],
      "properties": {
        "groups": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["title", "message", "files"],
            "properties": {
              "title": { "type": "string", "description": "Short descriptive topic title in Thai" },
              "message": { "type": "string", "description": "Conventional commit message (e.g. feat(memory): ...)" },
              "files": { "type": "array", "items": { "type": "string" }, "description": "Relative paths of files in this commit" }
            }
          }
        }
      }
    }
  }
}`

var gitSplitTool = func() model.ToolDefinition {
	var def model.ToolDefinition
	_ = json.Unmarshal([]byte(gitSplitToolJSON), &def)
	return def
}()

// isDangerousPath reports whether a file path points to a sensitive/secret or binary file
// that should never be automatically proposed for Git commits.
func isDangerousPath(path string) bool {
	p := filepath.ToSlash(strings.ToLower(strings.TrimSpace(path)))
	parts := strings.Split(p, "/")
	filename := parts[len(parts)-1]

	// 1. Environment & Secrets (.env)
	if filename == ".env" || (strings.HasPrefix(filename, ".env.") &&
		!strings.HasSuffix(filename, ".example") &&
		!strings.HasSuffix(filename, ".sample") &&
		!strings.HasSuffix(filename, ".template")) ||
		strings.HasSuffix(filename, ".env") {
		return true
	}

	// 2. Private Keys & Certificates
	for _, ext := range []string{".pem", ".key", ".pkcs12", ".pfx", ".p12"} {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	for _, keyName := range []string{"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519"} {
		if filename == keyName {
			return true
		}
	}

	// 3. Credentials & Tokens
	for _, cred := range []string{
		"credentials.json", "credentials.yaml", "credentials.yml",
		"service-account.json", "service_account.json",
		".npmrc", ".pypirc",
	} {
		if filename == cred {
			return true
		}
	}
	if strings.Contains(p, ".aws/credentials") {
		return true
	}

	// 4. Databases & Executables
	for _, ext := range []string{".sqlite", ".sqlite3", ".db", ".dump", ".sql.gz", ".sql.bak", ".exe", ".dll", ".so", ".dylib"} {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}

	return false
}

// GitSuggestSplitCommits analyzes the working tree changes, groups them by topic,
// and returns proposed atomic commits with suggested messages.
func (a *App) GitSuggestSplitCommits() ([]GitCommitGroup, error) {
	out := []GitCommitGroup{}
	tree := a.GitWorkingTree()
	if len(tree) == 0 {
		return out, nil
	}

	// Filter out sensitive or dangerous files from automatic split suggestions
	var safeTree []GitFileChange
	for _, f := range tree {
		if !isDangerousPath(f.Path) {
			safeTree = append(safeTree, f)
		}
	}
	tree = safeTree
	if len(tree) == 0 {
		return out, nil
	}

	root, ok := a.gitRoot()
	if !ok {
		return out, errors.New("no git repository focused")
	}

	// Single file fallback: no need to ask LLM for grouping
	if len(tree) == 1 {
		msg, _ := a.GitSuggestCommitMessage([]string{tree[0].Path})
		if msg == "" {
			msg = fmt.Sprintf("chore: update %s", tree[0].Path)
		}
		return []GitCommitGroup{{
			Title:   tree[0].Path,
			Message: msg,
			Files:   []string{tree[0].Path},
		}}, nil
	}

	ctx, cancel := a.gitContext()
	defer cancel()

	p, modelName, err := a.oneShotProvider()
	if err != nil {
		// Fallback to grouping by top-level directory or single group
		return fallbackSplitGroups(tree), nil
	}

	recentLog, _ := gitOut(ctx, root, "log", "-n", "8", "--oneline")

	var changesSummary strings.Builder
	for _, f := range tree {
		changesSummary.WriteString(fmt.Sprintf("- [%s] %s (+%d -%d)\n", f.Status, f.Path, f.Added, f.Removed))
	}

	// Also append brief diff snippets for non-binary files
	var diffSnippets strings.Builder
	for i, f := range tree {
		if i >= 15 {
			diffSnippets.WriteString(fmt.Sprintf("\n... and %d more files", len(tree)-15))
			break
		}
		diff, _ := gitOut(ctx, root, "diff", "HEAD", "--", f.Path)
		lines := strings.Split(diff, "\n")
		if len(lines) > 20 {
			diff = strings.Join(lines[:20], "\n") + "\n..."
		}
		if strings.TrimSpace(diff) != "" {
			diffSnippets.WriteString(fmt.Sprintf("\n=== Diff: %s ===\n%s\n", f.Path, diff))
		}
	}

	userContent := fmt.Sprintf(`Analyze the changed files and diffs in this repository.
Group them into logical, cohesive, atomic commits (Conventional Commits style).
Do NOT put all files in one commit if they belong to different features, fixes, or components.
Every changed file must be included in exactly one group.
Match the repository's recent commit style and language (e.g. Thai or English).

Recent repository commits:
%s

Changed files in working tree:
%s

Diff snippets:
%s`, strings.TrimSpace(recentLog), changesSummary.String(), diffSnippets.String())

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: "You are an expert software engineer and Git release manager. You organize uncommitted changes into clean, atomic commits."},
			{Role: model.RoleUser, Content: userContent},
		},
		Tools:      []model.ToolDefinition{gitSplitTool},
		ToolChoice: "required",
		MaxTokens:  1200,
	}

	resp, err := p.Complete(ctx, req)
	if err != nil {
		return fallbackSplitGroups(tree), nil
	}

	var rawArgs string
	if len(resp.ToolCalls) > 0 {
		rawArgs = resp.ToolCalls[0].Function.Arguments
	} else {
		rawArgs = extractJSONObject(resp.Text)
	}

	var parsed struct {
		Groups []GitCommitGroup `json:"groups"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &parsed); err != nil || len(parsed.Groups) == 0 {
		return fallbackSplitGroups(tree), nil
	}

	// Verify all returned files exist in tree
	treeMap := make(map[string]bool)
	for _, f := range tree {
		treeMap[f.Path] = true
	}

	claimed := make(map[string]bool)
	for _, g := range parsed.Groups {
		validFiles := []string{}
		for _, fp := range g.Files {
			if treeMap[fp] && !claimed[fp] {
				validFiles = append(validFiles, fp)
				claimed[fp] = true
			}
		}
		if len(validFiles) > 0 {
			out = append(out, GitCommitGroup{
				Title:   g.Title,
				Message: g.Message,
				Files:   validFiles,
			})
		}
	}

	// Any unassigned files get put into an "Other changes" group
	var leftovers []string
	for _, f := range tree {
		if !claimed[f.Path] {
			leftovers = append(leftovers, f.Path)
		}
	}
	if len(leftovers) > 0 {
		out = append(out, GitCommitGroup{
			Title:   "การเปลี่ยนแปลงอื่น ๆ",
			Message: "chore: update miscellaneous files",
			Files:   leftovers,
		})
	}

	return out, nil
}

func fallbackSplitGroups(tree []GitFileChange) []GitCommitGroup {
	byDir := make(map[string][]string)
	for _, f := range tree {
		parts := strings.Split(f.Path, "/")
		key := "root"
		if len(parts) > 1 {
			key = parts[0]
		}
		byDir[key] = append(byDir[key], f.Path)
	}

	var groups []GitCommitGroup
	for dir, files := range byDir {
		title := fmt.Sprintf("การเปลี่ยนแปลงใน %s", dir)
		if dir == "root" {
			title = "ไฟล์ใน root"
		}
		groups = append(groups, GitCommitGroup{
			Title:   title,
			Message: fmt.Sprintf("chore(%s): update files", dir),
			Files:   files,
		})
	}
	return groups
}
