package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/proc"
)

// GitCommitGroup is one proposed commit: the card's title, the files, and the
// message. Source says where the message came from — "model" when the model
// wrote it, "fallback" when nobody did and Message is the directory grouping's
// placeholder, "" while the model is still writing it and the pane is filling
// it in from the git:split:* events (see gitSplitWriteMessages).
type GitCommitGroup struct {
	Title   string   `json:"title"`
	Message string   `json:"message"`
	Files   []string `json:"files"`
	Source  string   `json:"source,omitempty"`
	// Reason is why Source is "fallback" — the provider was unavailable, the
	// model did not answer — shown on the card so a placeholder is never
	// mistaken for a message the model wrote.
	Reason string `json:"reason,omitempty"`
}

const (
	gitSplitSourceModel    = "model"
	gitSplitSourceFallback = "fallback"

	// The pane's events. chunk carries text as it streams, message the
	// finished message of one group, done the end of the run.
	gitSplitEventChunk   = "git:split:chunk"
	gitSplitEventMessage = "git:split:message"
	gitSplitEventDone    = "git:split:done"
)

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

// GitSuggestCommitMessage has the active model write the message for one
// commit made of the given files — the manual mode's ✨ button. The same
// writer as the smart split's, without the streaming.
func (a *App) GitSuggestCommitMessage(files []string) (string, error) {
	root, ok := a.gitRoot()
	if !ok {
		return "", errors.New("no git repository focused")
	}
	p, modelName, err := a.oneShotProvider()
	if err != nil {
		return "", fmt.Errorf("provider unavailable: %w", err)
	}

	tree, err := a.GitWorkingTree()
	if err != nil {
		return "", err
	}
	want := make(map[string]bool, len(files))
	for _, f := range files {
		want[f] = true
	}
	var chosen []GitFileChange
	for _, f := range tree {
		if want[f.Path] {
			chosen = append(chosen, f)
		}
	}
	if len(chosen) == 0 {
		return "", errors.New("none of the files has changes")
	}

	gctx, gcancel := a.gitContext()
	defer gcancel()
	recentLog, _ := gitOut(gctx, root, "log", "-n", "8", "--oneline")
	diffs := gitDiffByFile(gctx, root, chosen)

	// The model's own clock, not git's: gitContext is a 10 s budget for one
	// git process and a model writing a message is not one of those.
	msg, err := writeCommitMessage(a.engineCtx(), p, modelName, recentLog, chosen, diffs, nil)
	if err != nil {
		return "", err
	}
	return msg, nil
}

// GitSplitCancel ends the smart split's message-writing run in flight. The
// groups already on screen stay; the messages still unwritten stay empty.
func (a *App) GitSplitCancel() {
	a.gitSplitMu.Lock()
	defer a.gitSplitMu.Unlock()
	if a.gitSplitCancel != nil {
		a.gitSplitCancel()
		a.gitSplitCancel = nil
	}
}

// gitSplitBegin seats a new run: the previous one is cancelled, and the seat
// number lets a run that lost its seat notice before it emits.
func (a *App) gitSplitBegin() (context.Context, uint64) {
	a.gitSplitMu.Lock()
	defer a.gitSplitMu.Unlock()
	if a.gitSplitCancel != nil {
		a.gitSplitCancel()
	}
	ctx, cancel := context.WithCancel(a.engineCtx())
	a.gitSplitCancel = cancel
	a.gitSplitSeq++
	return ctx, a.gitSplitSeq
}

func (a *App) gitSplitSeated(seq uint64) bool {
	a.gitSplitMu.Lock()
	defer a.gitSplitMu.Unlock()
	return a.gitSplitSeq == seq
}

// gitSplitToolJSON is the grouping call's tool. Groups only — title and files.
// The message of each group is written afterwards, one group at a time with
// that group's whole diff in front of the model, and streamed to the pane;
// asking for all the messages inside one JSON answer is what made the old
// call slow enough to lose to its own clock, and thin enough to say nothing.
const gitSplitToolJSON = `{
  "type": "function",
  "function": {
    "name": "propose_split_commits",
    "description": "Group the changed files into atomic, cohesive Git commits. Title and files only; the commit messages are written separately.",
    "parameters": {
      "type": "object",
      "required": ["groups"],
      "properties": {
        "groups": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["title", "files"],
            "properties": {
              "title": { "type": "string", "description": "What this commit does, as a short noun phrase a reader of the list understands without opening it (at most 60 characters), in the language of the repository's recent commits" },
              "files": { "type": "array", "items": { "type": "string" }, "description": "Relative paths of the files in this commit, exactly as listed" }
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

	// 0. The app's own attachments (attachmentsDir): what the chat saved when
	// the user pasted a screenshot. Not the project's, never proposed.
	for _, seg := range parts {
		if seg == attachmentsDir {
			return true
		}
	}

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

// GitSuggestSplitCommits groups the working tree's changes into commits and
// returns the groups at once — titles and files, messages still empty. The
// messages are then written one group at a time on a goroutine and reach the
// pane as git:split:chunk / git:split:message / git:split:done, so the first
// card fills in while the model is still reading the second. No clock: the
// run ends when the model is done, when the user cancels (GitSplitCancel), or
// when the next click seats a new run.
//
// When there is no model to ask, the groups are the directory grouping with
// Source "fallback" and the reason on each — a placeholder the pane labels as
// one, never a message dressed up as the model's.
func (a *App) GitSuggestSplitCommits() ([]GitCommitGroup, error) {
	out := []GitCommitGroup{}
	tree, err := a.GitWorkingTree()
	if err != nil {
		return out, err
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

	p, modelName, err := a.oneShotProvider()
	if err != nil {
		a.GitSplitCancel()
		return fallbackSplitGroupsWithReason(tree, fmt.Sprintf("provider unavailable: %v", err)), nil
	}

	ctx, seq := a.gitSplitBegin()

	gctx, gcancel := a.gitContext()
	recentLog, _ := gitOut(gctx, root, "log", "-n", "8", "--oneline")
	diffs := gitDiffByFile(gctx, root, tree)
	gcancel()

	var groups []GitCommitGroup
	if len(tree) == 1 {
		// One file is one commit; nothing to group.
		groups = []GitCommitGroup{{Title: tree[0].Path, Files: []string{tree[0].Path}}}
	} else {
		groups, err = proposeSplitGroups(ctx, p, modelName, recentLog, tree, diffs)
		if err != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			// The model could not group (no tool calling, a garbled answer):
			// the directory grouping stands in, but the messages are still
			// the model's to write — that call is plain text and may well
			// work where the tool call did not.
			groups = fallbackSplitGroups(tree)
			for i := range groups {
				groups[i].Message = ""
				groups[i].Source = ""
			}
		}
	}

	go a.gitSplitWriteMessages(ctx, seq, p, modelName, recentLog, tree, diffs, groups)
	return groups, nil
}

// proposeSplitGroups is the grouping call: every changed file into exactly one
// group, titles in the repository's language, no messages yet.
func proposeSplitGroups(ctx context.Context, p model.Provider, modelName, recentLog string, tree []GitFileChange, diffs map[string]string) ([]GitCommitGroup, error) {
	var changesSummary strings.Builder
	for _, f := range tree {
		changesSummary.WriteString(fmt.Sprintf("- [%s] %s (+%d -%d)\n", f.Status, f.Path, f.Added, f.Removed))
	}

	// A glimpse of each diff is enough to tell what belongs together; the
	// message writer sees the whole thing later.
	var diffSnippets strings.Builder
	const snippetFiles, snippetLines = 30, 25
	for i, f := range tree {
		if i >= snippetFiles {
			diffSnippets.WriteString(fmt.Sprintf("\n... and %d more files", len(tree)-snippetFiles))
			break
		}
		d := clipLines(diffs[f.Path], snippetLines)
		if strings.TrimSpace(d) != "" {
			diffSnippets.WriteString(fmt.Sprintf("\n=== %s ===\n%s\n", f.Path, d))
		}
	}

	userContent := fmt.Sprintf(`Group the changed files of this repository into commits.

Rules:
- One commit per piece of work: a feature, a fix, a refactor, a docs change. Files that change for the same reason go together even when they sit in different folders; files that change for different reasons are split even when they sit in the same folder.
- Every changed file goes into exactly one group. Use the paths exactly as listed.
- A group's title says what that commit does, in the same language as the recent commits below (Thai commits → Thai title). Never "update files", "misc", "various changes".
- Do not write commit messages; they are written afterwards.

Recent commits (language and style reference):
%s

Changed files:
%s
Diff snippets:
%s`, strings.TrimSpace(recentLog), changesSummary.String(), diffSnippets.String())

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: "You are a careful software engineer preparing a clean commit history. You group uncommitted changes into atomic commits."},
			{Role: model.RoleUser, Content: userContent},
		},
		Tools:      []model.ToolDefinition{gitSplitTool},
		ToolChoice: "required",
		MaxTokens:  1500,
	}

	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
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
	if err := json.Unmarshal([]byte(rawArgs), &parsed); err != nil {
		return nil, fmt.Errorf("model answer is not the grouping: %w", err)
	}
	if len(parsed.Groups) == 0 {
		return nil, errors.New("model proposed no groups")
	}

	// Verify all returned files exist in tree
	treeMap := make(map[string]bool)
	for _, f := range tree {
		treeMap[f.Path] = true
	}

	var out []GitCommitGroup
	claimed := make(map[string]bool)
	for _, g := range parsed.Groups {
		validFiles := []string{}
		for _, fp := range g.Files {
			fp = filepath.ToSlash(strings.TrimSpace(fp))
			if treeMap[fp] && !claimed[fp] {
				validFiles = append(validFiles, fp)
				claimed[fp] = true
			}
		}
		if len(validFiles) > 0 {
			out = append(out, GitCommitGroup{
				Title: strings.TrimSpace(g.Title),
				Files: validFiles,
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
			Title: "ไฟล์ที่เหลือ",
			Files: leftovers,
		})
	}
	if len(out) == 0 {
		return nil, errors.New("model proposed no usable groups")
	}
	return out, nil
}

// gitSplitWriteMessages writes the groups' messages one after another and
// hands each to the pane the moment it is done. Sequential on purpose: the
// user reads the first card while the second is being written, and a local
// model asked for six messages at once answers none of them quickly.
func (a *App) gitSplitWriteMessages(ctx context.Context, seq uint64, p model.Provider, modelName, recentLog string, tree []GitFileChange, diffs map[string]string, groups []GitCommitGroup) {
	byPath := make(map[string]GitFileChange, len(tree))
	for _, f := range tree {
		byPath[f.Path] = f
	}
	emit := func(event string, data any) bool {
		if ctx.Err() != nil || !a.gitSplitSeated(seq) {
			return false
		}
		a.emitEvent(event, data)
		return true
	}

	for i, g := range groups {
		if ctx.Err() != nil {
			return
		}
		var files []GitFileChange
		for _, fp := range g.Files {
			if f, ok := byPath[fp]; ok {
				files = append(files, f)
			}
		}
		onChunk := func(text string) error {
			if !emit(gitSplitEventChunk, map[string]any{"index": i, "text": text}) {
				return ctx.Err()
			}
			return nil
		}
		msg, err := writeCommitMessage(ctx, p, modelName, recentLog, files, diffs, onChunk)
		if ctx.Err() != nil {
			return
		}
		payload := map[string]any{"index": i, "message": msg, "source": gitSplitSourceModel}
		if err != nil {
			payload["message"] = fallbackCommitMessage(g.Files)
			payload["source"] = gitSplitSourceFallback
			payload["reason"] = err.Error()
		}
		if !emit(gitSplitEventMessage, payload) {
			return
		}
	}
	emit(gitSplitEventDone, map[string]any{})
}

// writeCommitMessage has the model write the message of ONE commit from the
// whole diff of its files. onChunk, when given and the provider streams, sees
// the text as it arrives; the return is the finished, cleaned message.
func writeCommitMessage(ctx context.Context, p model.Provider, modelName, recentLog string, files []GitFileChange, diffs map[string]string, onChunk model.StreamChunkHandler) (string, error) {
	var list strings.Builder
	for _, f := range files {
		list.WriteString(fmt.Sprintf("- [%s] %s (+%d -%d)\n", f.Status, f.Path, f.Added, f.Removed))
	}

	// The whole diff, within reason: the writer must see what changed to say
	// what changed, but a 4,000-line generated file is not what it needs.
	var diff strings.Builder
	const perFile, total = 160, 700
	lines := 0
	for _, f := range files {
		d := diffs[f.Path]
		if strings.TrimSpace(d) == "" {
			continue
		}
		room := total - lines
		if room <= 0 {
			diff.WriteString(fmt.Sprintf("\n=== %s === (diff omitted, budget spent)\n", f.Path))
			continue
		}
		if room > perFile {
			room = perFile
		}
		d = clipLines(d, room)
		lines += strings.Count(d, "\n") + 1
		diff.WriteString(fmt.Sprintf("\n=== %s ===\n%s\n", f.Path, d))
	}

	userContent := fmt.Sprintf(`Write the commit message for this one commit.

Format:
- Line 1: type(scope): summary — Conventional Commits. type is one of feat, fix, refactor, perf, docs, test, chore, style, build, in English. scope is the part of the project touched (a package, folder or feature; short). summary is one sentence saying WHAT changed and its effect, at most 100 characters, no trailing period.
- Blank line.
- Then 1 to 5 lines, each starting with "- ", one per concrete change in this diff: which part, what it does now, and why when the diff shows it. Group by meaning, not by file; do not list file names.
- Write the summary and the lines in the same language as the recent commits below (Thai commits → Thai; keep identifiers, the type and the scope in English).
- Say what the change is. Never "update files", "misc changes", "various fixes", "improve code". A tiny change gets one line.
- Output the message only: no markdown fence, no quotes, no explanation before or after.

Recent commits (language and style reference):
%s

Files in this commit:
%s
Diff:
%s`, strings.TrimSpace(recentLog), list.String(), diff.String())

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: "You write Git commit messages. You are shown the diff of exactly the files that go into one commit, and you describe that commit."},
			{Role: model.RoleUser, Content: userContent},
		},
		MaxTokens: 700,
	}

	var streamed strings.Builder
	var resp model.Response
	var err error
	if sp, ok := p.(model.StreamingProvider); ok && onChunk != nil {
		resp, err = sp.StreamComplete(ctx, req, func(chunk string) error {
			streamed.WriteString(chunk)
			return onChunk(chunk)
		}, nil)
	} else {
		resp, err = p.Complete(ctx, req)
	}
	if err != nil {
		return "", err
	}
	text := resp.Text
	if strings.TrimSpace(text) == "" {
		text = streamed.String()
	}
	msg := cleanCommitMessage(text)
	if msg == "" {
		return "", errors.New("model wrote nothing")
	}
	return msg, nil
}

// cleanCommitMessage strips what a model wraps a message in — a code fence,
// quotes, a "Commit message:" label — and trims blank lines at both ends.
func cleanCommitMessage(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if nl := strings.IndexByte(s, '\n'); nl >= 0 && !strings.ContainsAny(s[:nl], " :(") {
			s = s[nl+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	s = strings.TrimSpace(s)
	for _, label := range []string{"Commit message:", "commit message:", "ข้อความคอมมิต:"} {
		s = strings.TrimSpace(strings.TrimPrefix(s, label))
	}
	if len(s) >= 2 && (s[0] == '"' || s[0] == '`') && s[len(s)-1] == s[0] && !strings.Contains(s, "\n") {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// gitDiffByFile is every file's diff against HEAD in ONE git process, split
// per path; untracked files, which HEAD has no diff for and the old loop
// showed as nothing at all, are read from disk as an all-added file. One
// process rather than one per file: git is 1–5 s a process on a loaded
// machine, and fifteen in a row was the whole budget before the model was
// even asked. Best effort — a path with no entry has an empty diff.
func gitDiffByFile(ctx context.Context, root string, tree []GitFileChange) map[string]string {
	out := make(map[string]string, len(tree))
	var tracked []string
	for _, f := range tree {
		if f.Status == "U" {
			out[f.Path] = untrackedAsDiff(filepath.Join(root, filepath.FromSlash(f.Path)))
			continue
		}
		tracked = append(tracked, f.Path)
	}
	if len(tracked) == 0 {
		return out
	}
	args := append([]string{"diff", "HEAD", "--no-renames", "--no-color", "--"}, tracked...)
	raw, err := gitOut(ctx, root, args...)
	if err != nil {
		return out
	}
	for _, chunk := range strings.Split("\n"+raw, "\ndiff --git ") {
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		head, body, _ := strings.Cut(chunk, "\n")
		// "a/P b/P": with --no-renames both halves are the same path, so it
		// is the first half of what is left after "a/" — which holds even
		// when P has spaces in it.
		rest := strings.TrimPrefix(head, "a/")
		if len(rest) < 3 {
			continue
		}
		path := rest[:(len(rest)-3)/2]
		if !strings.HasPrefix(rest[len(path):], " b/") {
			continue
		}
		// Drop the index/mode lines: tokens that say nothing about the change.
		var kept []string
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "new file mode") ||
				strings.HasPrefix(line, "deleted file mode") || strings.HasPrefix(line, "old mode") ||
				strings.HasPrefix(line, "new mode") {
				continue
			}
			kept = append(kept, line)
		}
		out[path] = strings.TrimRight(strings.Join(kept, "\n"), "\n")
	}
	return out
}

// untrackedAsDiff shows a new file the way a diff would — every line added —
// or says it is binary. The first lines only; a new file is usually a whole
// unit and its head says what it is.
func untrackedAsDiff(abs string) string {
	const head = 120
	data, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	if strings.IndexByte(string(data), 0) >= 0 {
		return fmt.Sprintf("(binary file, %d bytes)", len(data))
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	total := len(lines)
	if total > head {
		lines = lines[:head]
	}
	var b strings.Builder
	b.WriteString("(new file)\n")
	for _, l := range lines {
		b.WriteString("+")
		b.WriteString(l)
		b.WriteString("\n")
	}
	if total > head {
		b.WriteString(fmt.Sprintf("... (%d more lines)", total-head))
	}
	return strings.TrimRight(b.String(), "\n")
}

func clipLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-n)
}

// fallbackCommitMessage is the placeholder when nobody wrote a message: it
// names the files so the commit is at least honest about what it holds, and
// the pane labels it as the placeholder it is. Never "update files".
func fallbackCommitMessage(files []string) string {
	scope := "root"
	dirs := map[string]bool{}
	for _, f := range files {
		if i := strings.IndexByte(f, '/'); i > 0 {
			dirs[f[:i]] = true
		} else {
			dirs["root"] = true
		}
	}
	if len(dirs) == 1 {
		for d := range dirs {
			scope = d
		}
	} else {
		scope = fmt.Sprintf("%d folders", len(dirs))
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("chore(%s): แก้ %d ไฟล์ — ยังไม่มีคำอธิบาย\n", scope, len(files)))
	for _, f := range files {
		b.WriteString("\n- ")
		b.WriteString(f)
	}
	return b.String()
}

// fallbackSplitGroups is the directory grouping: what stands in when there is
// no model to ask. Deterministic order so the cards do not shuffle between
// two clicks.
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
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	var groups []GitCommitGroup
	for _, dir := range dirs {
		files := byDir[dir]
		title := fmt.Sprintf("การเปลี่ยนแปลงใน %s", dir)
		if dir == "root" {
			title = "ไฟล์ใน root"
		}
		groups = append(groups, GitCommitGroup{
			Title:   title,
			Message: fallbackCommitMessage(files),
			Files:   files,
			Source:  gitSplitSourceFallback,
		})
	}
	return groups
}

func fallbackSplitGroupsWithReason(tree []GitFileChange, reason string) []GitCommitGroup {
	groups := fallbackSplitGroups(tree)
	for i := range groups {
		groups[i].Reason = reason
	}
	return groups
}
