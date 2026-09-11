package skill

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
	"time"

	"github.com/Mikedev115/Aetox/internal/callfault"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/proc"
	"github.com/Mikedev115/Aetox/internal/statereport"
)

type gitSkill struct {
	root string
}

func (*gitSkill) Name() string { return "git" }

func (*gitSkill) Description() string {
	return "รันคำสั่ง git แบบอ่านอย่างเดียวที่ปลอดภัย (status, log, branch, diff, show)"
}

// ToolDefinition hands git to the model, alongside shell and for the same
// reason: an agent that cannot read `git diff` cannot see what it just did.
//
// This one is the narrower half of that pair — `allowedGitReadActions` is a
// read-only allowlist and validateGitReadArgs blocks the options that would
// escape the workspace, so nothing here can mutate a repository. The enum is
// built from that same map rather than restated, or the schema drifts from the
// check the day an action is added; sorted because an unsorted enum reshuffles
// the tool payload on every turn and misses the provider's prefix cache (see
// Registry.Names).
func (*gitSkill) ToolDefinition() model.ToolDefinition {
	actions := make([]string, 0, len(allowedGitReadActions))
	for action := range allowedGitReadActions {
		actions = append(actions, action)
	}
	sort.Strings(actions)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        actions,
				"description": "Which read-only git command to run.",
			},
			"args": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Arguments for the action, one per element, e.g. [\"--stat\", \"HEAD~1\"]. Omit for the plain command.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Repository directory to run in, relative to the workspace. Omit for the workspace root.",
			},
		},
		"required":             []string{"action"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "git",
			Description: "Read a repository's git state: " + strings.Join(actions, ", ") + ". Read-only.",
			Parameters:  payload,
		},
	}
}

// Guidance carries what the block entry used to say about when and where —
// moved here the day `path` joined the signature, because the entry was
// already over the standard (block_standard_test.go) and a parameter is worth
// more in it than a sentence of habit.
func (*gitSkill) Guidance(map[string]any) string {
	return "Use diff to check what your own edits changed before reporting them done. " +
		"Committing, checking out or anything else that writes belongs in shell, where the user approves it.\n" +
		"When the workspace root is not the repository, pass path — a repository inside the workspace or one the " +
		"user added alongside it. -C and --git-dir are blocked; path is the way there."
}

func (s *gitSkill) ExecuteTool(ctx context.Context, args map[string]any) (Output, error) {
	action, _ := args["action"].(string)
	if strings.TrimSpace(action) == "" {
		err := callfault.New("action is required")
		return newToolOutput("git", "git", "", time.Now(), false, err), err
	}
	callArgs := append([]string{action}, anyStringSlice(args["args"])...)
	input := Input{"args": callArgs}
	if path, _ := args["path"].(string); strings.TrimSpace(path) != "" {
		input["path"] = path
	}
	return s.Execute(ctx, input)
}

func (s *gitSkill) Execute(ctx context.Context, input Input) (Output, error) {
	start := time.Now()
	if s == nil {
		err := errors.New("git skill unavailable")
		return newToolOutput("git", "git", "", start, false, err), err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := exec.LookPath("git"); err != nil {
		// Git being absent from PATH is this machine, not this call (statereport).
		err = statereport.New("git command not found in PATH")
		return newToolOutput("git", "git", "", start, false, err), err
	}

	args := stringSlice(input["args"])
	if len(args) == 0 {
		err := errors.New("usage: git <status|log|branch|diff|show> [args]")
		return newToolOutput("git", "git", "", start, false, err), err
	}

	action := strings.ToLower(strings.TrimSpace(args[0]))
	if _, ok := allowedGitReadActions[action]; !ok {
		err := fmt.Errorf("unsupported git action: %s", action)
		return newToolOutput("git", "git "+strings.Join(args, " "), "", start, false, err), err
	}

	actionArgs := args[1:]
	if err := validateGitReadArgs(action, actionArgs); err != nil {
		return newToolOutput("git", "git "+strings.Join(args, " "), "", start, false, err), err
	}
	commandText := "git " + strings.Join(args, " ")

	// Where the command runs. The workspace root unless `path` says otherwise,
	// and `path` goes through the one door every file tool uses
	// (resolveSandboxPath): inside the root, or a folder the user added, and
	// never a credential store. It exists because a reviewer on 11 ก.ย. sat at
	// a desk that was not a repository while the assistant it was checking
	// worked in one — `read`, `list` and `glob` all took a path and followed;
	// git had none, `-C` and `--git-dir` are blocked below, and the model tried
	// all three before giving up. A repository the user has opened is one the
	// user has opened; the tool that reads it should not need the desk moved.
	root, err := resolveSafeWorkspace(s.root)
	if err != nil {
		return newToolOutput("git", commandText, "", start, false, err), err
	}
	requestPath, _ := input["path"].(string)
	requestPath = strings.TrimSpace(requestPath)
	if requestPath != "" {
		commandText += " (in " + requestPath + ")"
		dir, err := resolveSandboxPath(s.root, requestPath)
		if err != nil {
			return newToolOutput("git", commandText, "", start, false, err), err
		}
		info, err := os.Stat(dir)
		if err != nil {
			err = callfault.Newf("path: %w", err)
			return newToolOutput("git", commandText, "", start, false, err), err
		}
		if !info.IsDir() {
			err = callfault.Newf("path is not a directory: %s", requestPath)
			return newToolOutput("git", commandText, "", start, false, err), err
		}
		root = dir
	}

	// And the same gate every other tool answers to (shell_sandbox.go). The
	// check above is a denylist of git options someone thought of; this one
	// asks the only question that generalises — does this argument name
	// somewhere outside the folders the user chose. Asked of the directory
	// the command runs in, so a relative argument means what git will take it
	// to mean; that directory was itself admitted by the same gate a moment
	// ago, so nothing under it is new ground.
	if err := guardArgs(root, actionArgs); err != nil {
		return newToolOutput("git", commandText, "", start, false, err), err
	}

	if err := ensureGitRepo(ctx, root); err != nil {
		if requestPath == "" {
			// The desk is not a repository. That is a fact about the desk, and
			// the remedy is in the call: say where the repository is.
			err = fmt.Errorf("%w; the workspace root is not a git repository, pass path to run in one elsewhere", err)
		}
		return newToolOutput("git", commandText, "", start, false, err), err
	}

	command := append([]string{action}, actionArgs...)
	output, capped, err := executeCommand(ctx, "git", root, command...)
	output = strings.TrimSpace(output)
	if output == "" {
		output = "(no output)"
	}
	output, truncated := limitLines(output, defaultToolOutputLineLimit)
	// A capped `git show` of a committed binary is one very long line, which
	// limitLines happily calls complete. Say so, or the model reasons about a
	// diff it only half received.
	if capped {
		truncated = true
		output += "\n... (output exceeded 1 MiB and was cut)"
	}
	result := newToolOutput("git", commandText, output, start, truncated, err)

	if err != nil {
		return result, err
	}

	return result, nil
}

var allowedGitReadActions = map[string]struct{}{
	"status": {},
	"log":    {},
	"branch": {},
	"diff":   {},
	"show":   {},
}

func validateGitReadArgs(action string, args []string) error {
	for _, rawArg := range args {
		arg := strings.TrimSpace(rawArg)
		if arg == "" {
			continue
		}
		lower := strings.ToLower(arg)

		switch {
		case lower == "--git-dir" || lower == "--work-tree" || strings.HasPrefix(lower, "--git-dir=") || strings.HasPrefix(lower, "--work-tree="):
			return fmt.Errorf("unsafe git option blocked: %s", arg)
		case lower == "-c" || strings.HasPrefix(lower, "-c"):
			return fmt.Errorf("unsafe git option blocked: %s", arg)
		case lower == "-C" || strings.HasPrefix(lower, "-C"):
			return fmt.Errorf("unsafe git option blocked: %s", arg)
		}

		if action != "status" && strings.Contains(lower, "/../") {
			return fmt.Errorf("unsupported path argument blocked: %s", arg)
		}
	}
	return nil
}

func resolveSafeWorkspace(root string) (string, error) {
	workspace := strings.TrimSpace(root)
	if workspace == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot detect current directory: %w", err)
		}
		workspace = cwd
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("invalid workspace path: %w", err)
	}

	stats, err := os.Stat(absWorkspace)
	if err != nil {
		return "", fmt.Errorf("workspace not available: %w", err)
	}
	if !stats.IsDir() {
		return "", fmt.Errorf("workspace is not a directory: %s", absWorkspace)
	}

	return absWorkspace, nil
}

func ensureGitRepo(ctx context.Context, workspace string) error {
	output, _, err := executeCommand(ctx, "git", workspace, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return errors.New(strings.TrimSpace(output))
	}
	if strings.TrimSpace(output) != "true" {
		return errors.New("not inside a git repository")
	}
	return nil
}

// executeCommand runs a git command and reports whether its output hit the
// byte cap, so callers can tell the model the result is partial.
func executeCommand(ctx context.Context, name, dir string, args ...string) (string, bool, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	proc.HideConsole(cmd)
	// Same cap as shell: `git log` with no range, `git diff` on a big change
	// or `git show` of a commit that added a binary are all unbounded output.
	//
	// stdout and stderr are kept apart, and stderr only surfaces on failure.
	// Merging them meant every `git diff` on Windows arrived with a "warning:
	// LF will be replaced by CRLF" line per touched file glued to the front —
	// pure noise that the model reads as part of the diff and that can push
	// the real content past the 220-line output limit on a large change.
	stdout, stderr := &cappedWriter{}, &cappedWriter{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.buf.String())
	if err != nil {
		if msg := strings.TrimSpace(stderr.buf.String()); msg != "" {
			out = strings.TrimSpace(out + "\n" + msg)
		}
		if out == "" {
			out = "(command failed)"
		}
		return out, stdout.dropped, err
	}
	return out, stdout.dropped, nil
}
