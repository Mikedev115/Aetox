package skill

// The design_check tool: internal/designlint behind the sandbox, the fourth
// act of the `codebase` pack — is this file broken (errors), what is this
// name (symbol), what shape is this project (map), and now: does this UI
// carry the tells of a page assembled by habit. Everything about WHERE it
// may look is decided here, the library never widens a path; everything
// about WHAT a finding is stays in the library, which imports nothing from
// this package (the repo_map split, for the same reason).
//
// CategoryCode, like the rest of the pack: the tells are in source files,
// and the assistant desk holds no source. An unfocused session may still
// check a single file it was pointed at; a folder check with no project
// focused would walk the user's home and is refused, the repo_map refusal.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/designlint"
	"github.com/Mikedev115/Aetox/internal/model"
)

type designCheckSkill struct {
	root string
	open bool
}

func (*designCheckSkill) Name() string { return "design_check" }

func (*designCheckSkill) Description() string {
	return "Mechanical design tells in UI source: gradient text, glow shadows, side stripes, the AI palette, overused fonts, bounce easing, layout transitions, broken images, emoji as icons"
}

func (*designCheckSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "A UI file (.svelte, .css, .html, .tsx, ...) or a folder to check every one inside; default: the whole project",
			},
		},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "design_check",
			Description: "Design tells in UI source, by rule and line, with no browser and no model",
			Parameters:  payload,
		},
	}
}

func (*designCheckSkill) Guidance(map[string]any) string {
	return "Each line is a pattern in the file, not a verdict on the design: a finding on a card is a " +
		"stripe or a glow to remove, a finding on a mascot's pop-in is a choice already made. " +
		"Fix what was habit; for what was chosen, put `design-allow <rule>` in a comment on that " +
		"line (or alone on the line above) and it stays out of the report. Advisory rules name a " +
		"habit rather than a defect. Run it again on the files you touched before saying the UI is done."
}

// DesignCheckTimeBudget bounds one check the way RepoMapTimeBudget bounds a
// map: every file is read from disk, and a folder that turned out to be
// something else should end the call, not the turn.
const DesignCheckTimeBudget = 15 * time.Second

func (s *designCheckSkill) Execute(ctx context.Context, input Input) (Output, error) {
	start := time.Now()
	if s == nil {
		err := errors.New("design_check skill unavailable")
		return newToolOutput("design_check", "design_check", "", start, false, err), err
	}
	sub := "."
	if args := stringSlice(input["args"]); len(args) > 0 {
		sub = strings.TrimSpace(strings.Join(args, " "))
	}
	command := "design_check"
	if sub != "." && sub != "" {
		command += " " + sub
	}
	if s.open && (sub == "." || sub == "") {
		err := errors.New("no project is focused: design_check walks a project folder, not the whole machine — focus a project, or name a file or folder")
		return newToolOutput("design_check", command, "", start, false, err), err
	}
	base, err := resolveSandboxPath(s.root, sub)
	if err != nil {
		return newToolOutput("design_check", command, "", start, false, err), err
	}
	// The library reports paths relative to its root and walks under it;
	// checking from the resolved path as root keeps a folder outside the
	// sandbox (a workspace folder the user added) reporting sane paths too.
	checkCtx, cancel := context.WithTimeout(ctx, DesignCheckTimeBudget)
	defer cancel()
	root, target := filepath.Dir(base), filepath.Base(base)
	if info, statErr := os.Stat(base); statErr == nil && info.IsDir() {
		root, target = base, "."
	}
	findings, read, err := designlint.Check(checkCtx, designlint.Options{Root: root, Ignore: RepoMapIgnores()}, []string{target})
	if err != nil {
		return newToolOutput("design_check", command, "", start, false, err), err
	}
	return newToolOutput("design_check", command, RenderDesignFindings(findings, read), start, false, nil), nil
}

// RenderDesignFindings is one line per finding, path:line first so the
// model's next call is a read at that line, and a first line that says how
// much was read — a clean report over zero files is not a clean UI.
// Exported for `aetox design` (cmd/aetox), so a terminal and the model read
// the same report.
func RenderDesignFindings(findings []designlint.Finding, read int) string {
	if len(findings) == 0 {
		return fmt.Sprintf("(no findings) — %d UI files read", read)
	}
	advisory := 0
	for _, f := range findings {
		if f.Advisory {
			advisory++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "design check — %d UI files, %d findings", read, len(findings))
	if advisory > 0 {
		fmt.Fprintf(&b, " (%d advisory)", advisory)
	}
	b.WriteString("\n")
	for _, f := range findings {
		tag := ""
		if f.Advisory {
			tag = " (advisory)"
		}
		fmt.Fprintf(&b, "%s:%d %s%s — %s\n", f.Path, f.Line, f.Rule, tag, f.Detail)
	}
	return b.String()
}

func (s *designCheckSkill) ExecuteTool(ctx context.Context, args map[string]any) (Output, error) {
	var callArgs []string
	if path, ok := args["path"].(string); ok && strings.TrimSpace(path) != "" {
		callArgs = append(callArgs, strings.TrimSpace(path))
	}
	return s.Execute(ctx, Input{"args": callArgs})
}
