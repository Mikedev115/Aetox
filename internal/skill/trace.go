package skill

// The trace tool: internal/codeindex behind the sandbox, the sixth act of the
// `codebase` pack. `errors` asks whether this file is broken, `symbol` what a
// name is and who uses it, `impact` what changing it can disturb, `map` what
// shape the project has, `design` whether this UI was assembled by habit — and
// trace asks what a name CONNECTS to and which source line proves each hop.
//
// It exists for the one relationship the other five cannot see: a frontend call
// goes through a generated Wails binding into a Go method, and neither the
// language server nor the import map crosses a file nobody wrote by hand. What
// it returns is evidence, never a summary: every hop carries its path, line,
// relation, how strongly it was established, and which analyzer produced it.
//
// CategoryCode like the rest of the pack, and the open-sandbox refusal is
// repo_map's for the same reason: the unfocused root is the whole machine, and
// an index of "wherever the session stands" is a wrong answer ranked
// confidently.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/codeindex"
	"github.com/Mikedev115/Aetox/internal/model"
)

type traceSkill struct {
	root string
	open bool
}

func (*traceSkill) Name() string { return "trace" }

func (*traceSkill) Description() string {
	return "เดินตามความสัมพันธ์ของชื่อในโค้ด พร้อมหลักฐานไฟล์:บรรทัดทุกช่วง"
}

func (*traceSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Project-relative file where the name is written.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "The identifier to start from, exactly as written.",
			},
			"direction": map[string]any{
				"type":        "string",
				"enum":        []string{"callees", "callers", "between"},
				"description": "callees follows what this name reaches, callers which files reach it, between finds the path to targetPath/targetName.",
			},
			"depth": map[string]any{
				"type":        "integer",
				"description": "How many hops to follow. Default 3.",
			},
			"targetPath": map[string]any{
				"type":        "string",
				"description": "direction=between: project-relative file to reach.",
			},
			"targetName": map[string]any{
				"type":        "string",
				"description": "direction=between: the identifier to reach in targetPath.",
			},
		},
		"required":             []string{"path", "name"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name: "trace",
			Description: "Follow what a name connects to and see where each hop is written. " +
				"Crosses the generated Wails boundary from a frontend call to the Go method that answers it. " +
				"Every hop comes back with file:line, relation and how strongly it was established; " +
				"a hop that could not be established is reported as unknown, never dropped.",
			Parameters: payload,
		},
	}
}

func (s *traceSkill) Guidance(map[string]any) string {
	return "Use it when the question is how two places are connected — \"what does this call\", " +
		"\"who reaches this\", \"how does this button end up in that Go function\" — which symbol answers " +
		"with a reference list and grep cannot answer at all across a generated binding. " +
		"Depth is a fan-out and the default is the budget's ceiling: one hop is the file you named and " +
		"its own edges, and on a project that generates a binding layer between the frontend and the Go " +
		"method, the method is the third hop — two ends inside generated code. " +
		"A partial outcome means a budget was reached, not that the rest is unrelated."
}

func (s *traceSkill) Execute(ctx context.Context, input Input) (Output, error) {
	args := stringSlice(input["args"])
	if len(args) < 2 {
		err := errors.New("usage: trace <path> <name> [direction] [depth] [targetPath] [targetName]")
		return newToolOutput("trace", "trace", "", time.Now(), false, err), err
	}
	call := map[string]any{"path": args[0], "name": args[1]}
	if len(args) > 2 {
		call["direction"] = args[2]
	}
	if len(args) > 3 {
		call["depth"] = args[3]
	}
	if len(args) > 4 {
		call["targetPath"] = args[4]
	}
	if len(args) > 5 {
		call["targetName"] = args[5]
	}
	return s.ExecuteTool(ctx, call)
}

func (s *traceSkill) ExecuteTool(ctx context.Context, args map[string]any) (Output, error) {
	start := time.Now()
	if s == nil {
		err := errors.New("trace skill unavailable")
		return newToolOutput("trace", "trace", "", start, false, err), err
	}
	path, _ := args["path"].(string)
	name, _ := args["name"].(string)
	direction, _ := args["direction"].(string)
	targetPath, _ := args["targetPath"].(string)
	targetName, _ := args["targetName"].(string)
	path, name = strings.TrimSpace(path), strings.TrimSpace(name)
	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction == "" {
		direction = string(codeindex.DirectionCallees)
	}
	// 0 means "no opinion", and the traversal decides: the default is one
	// fact about the graph, and codeindex is where the graph is.
	depth := intArg(args["depth"])
	command := "trace " + path + " " + name
	if direction != string(codeindex.DirectionCallees) {
		command += " " + direction
	}
	if depth > 0 {
		command += fmt.Sprintf(" depth=%d", depth)
	}
	if strings.TrimSpace(targetPath) != "" {
		command += " -> " + strings.TrimSpace(targetPath)
		if strings.TrimSpace(targetName) != "" {
			command += " " + strings.TrimSpace(targetName)
		}
	}

	// Unfocused, there is no project to index — the same refusal repo_map
	// makes, worded for this question.
	if s.open {
		err := errors.New("no project is focused: trace follows references inside a project folder, not across the whole machine — focus a project first")
		return newToolOutput("trace", command, "", start, false, err), err
	}
	if path == "" || name == "" {
		err := errors.New("path and name are required")
		return newToolOutput("trace", command, "", start, false, err), err
	}
	abs, err := resolveSandboxPath(s.root, path)
	if err != nil {
		return newToolOutput("trace", command, "", start, false, err), err
	}
	rel, err := relToProjectRoot(s.root, abs)
	if err != nil {
		return newToolOutput("trace", command, "", start, false, err), err
	}
	targetRel := ""
	if strings.TrimSpace(targetPath) != "" {
		targetAbs, err := resolveSandboxPath(s.root, strings.TrimSpace(targetPath))
		if err != nil {
			return newToolOutput("trace", command, "", start, false, err), err
		}
		targetRel, err = relToProjectRoot(s.root, targetAbs)
		if err != nil {
			return newToolOutput("trace", command, "", start, false, err), err
		}
	}

	indexer, err := codeindex.NewIndexer(s.root, codeindex.DefaultBudget())
	if err != nil {
		return newToolOutput("trace", command, "", start, false, err), err
	}
	defer indexer.Close()
	result := indexer.Trace(ctx, codeindex.TraceRequest{
		Path:       rel,
		Name:       name,
		Direction:  codeindex.Direction(direction),
		Depth:      depth,
		TargetPath: targetRel,
		TargetName: strings.TrimSpace(targetName),
	})

	body := renderTrace(rel, name, codeindex.Direction(direction), targetRel, result)
	lines, truncated := limitLines(body, defaultToolOutputLineLimit)
	// A failure and a miss are both "this question could not be answered", and
	// both are reported as one: passing an unanswered question back as a
	// successful empty result is how "nothing is connected to this" gets read
	// over a file that does not exist.
	if result.Outcome == codeindex.OutcomeFailed || result.Outcome == codeindex.OutcomeUnavailable {
		err := errors.New(result.Reason)
		return newToolOutput("trace", command, lines, start, false, err), err
	}
	out := newToolOutput("trace", command, lines, start, truncated || result.Truncated, nil)
	// The timeline's readout, in this tool's own unit: hops, not lines.
	out.ResultCount = len(result.Facts)
	return out, nil
}

// renderTrace is the whole answer a model reads: the outcome first, because
// "partial" changes what the list means, then one block per hop with the two
// source anchors it runs between.
func renderTrace(path, name string, direction codeindex.Direction, target string, result codeindex.Result) string {
	var b strings.Builder
	// The direction is named before the hops, because it decides how every arrow
	// below reads: walked the other way, the same facts answer the other half of
	// the question. The command line that carries it belongs to the timeline —
	// the model is sent Content and nothing else — so an answer that does not
	// say which way it walked cannot be read.
	fmt.Fprintf(&b, "trace %s %s in %s", direction, name, path)
	if target != "" {
		fmt.Fprintf(&b, " -> %s", target)
	}
	// The depth the walk actually had, never the one that was asked for: a
	// header stating five hops over a walk of three is the same lie as a cap
	// that does not say it bit.
	fmt.Fprintf(&b, " (depth=%d)\n", result.Depth)
	fmt.Fprintf(&b, "outcome=%s facts=%d stale=%d reads=%d\n",
		result.Outcome, len(result.Facts), result.Stale, result.Reads)
	if result.Rebuilt {
		b.WriteString("index: built for this answer\n")
	}
	if result.Truncated {
		fmt.Fprintf(&b, "TRUNCATED: %s — the hops below are still valid, the rest was not walked\n", result.Reason)
	} else if result.Reason != "" {
		fmt.Fprintf(&b, "note: %s\n", result.Reason)
	}
	if result.Unknown > 0 {
		fmt.Fprintf(&b, "%d relationship(s) in this project could not be established and are not listed\n", result.Unknown)
	}
	if result.FileLevelHops > 0 {
		// An import is written BY a file and names no symbol. Following one
		// answers "these files are connected"; it is not evidence about the
		// name that was asked about, and a reader who cannot tell the two
		// apart will read a filename as an answer about a function.
		fmt.Fprintf(&b, "note: %d of these hop(s) are file-level (an import, marked by a destination with no name), not facts about %q\n",
			result.FileLevelHops, name)
	}
	if len(result.Facts) == 0 {
		b.WriteString("no hop was established\n")
		return b.String()
	}
	for index, fact := range result.Facts {
		fmt.Fprintf(&b, "%d. %s:%d", index+1, fact.From.Path, fact.From.Line)
		if fact.From.Name != "" {
			fmt.Fprintf(&b, " · %s", fact.From.Name)
		}
		fmt.Fprintf(&b, "\n   -> %s", fact.To.Path)
		// A hop whose destination is the whole file — an import — carries no
		// line, and printing one would put an unread number beside numbers
		// somebody actually looked at.
		if fact.To.Line > 0 {
			fmt.Fprintf(&b, ":%d", fact.To.Line)
		}
		if fact.To.Name != "" {
			fmt.Fprintf(&b, " · %s", fact.To.Name)
		}
		fmt.Fprintf(&b, " [%s · %s · %s]\n", fact.Relation, fact.Strength, fact.Producer)
	}
	return strings.TrimRight(b.String(), "\n")
}

func relToProjectRoot(root, abs string) (string, error) {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	slash := filepath.ToSlash(rel)
	if slash == ".." || strings.HasPrefix(slash, "../") {
		return "", fmt.Errorf("%s is outside the focused project root, which is what trace indexes", abs)
	}
	return slash, nil
}
