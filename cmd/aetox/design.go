package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mikedev115/Aetox/internal/designlint"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// `aetox design [path ...]` runs the design check from a terminal, a CI step
// or a hook — the same rules the `codebase design` tool runs for the model
// (internal/designlint), over the current directory when no path is given.
// Exit 0 with nothing to report, 2 with findings, 1 when a path is missing:
// the shape `go vet` and every linter have, so a hook that wants to block
// on findings can, and one that only wants to pass them on reads stdout.
//
// A hook is where this earns its place. hooks.json can already run a
// command after every `change` and hand what it prints to the model
// (internal/hook, PostToolUse notes); this verb reads the edited path from
// the call the hook receives, so one entry makes every UI edit checked:
//
//	{"event": "PostToolUse", "matcher": "change", "command": "aetox design --hook"}
//
// `--hook` reads AETOX_TOOL_ARGS, checks the file it names if it is a UI
// file, and says nothing otherwise — a note that appears only when there is
// something to say is the only kind that stays read.
func runDesignCommand(args []string) (handled bool, code int) {
	if len(args) == 0 || strings.ToLower(strings.TrimSpace(args[0])) != "design" {
		return false, 0
	}
	rest := args[1:]
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true, 1
	}
	var paths []string
	hook := false
	for _, a := range rest {
		switch a {
		case "--hook":
			hook = true
		case "-h", "--help":
			fmt.Println("usage: aetox design [--hook] [path ...]\n  path   a UI file or a folder (default: the current directory)\n  --hook read the edited file from AETOX_TOOL_ARGS (for hooks.json) and stay silent when it is not a UI file")
			return true, 0
		default:
			paths = append(paths, a)
		}
	}
	if hook {
		var call struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal([]byte(os.Getenv("AETOX_TOOL_ARGS")), &call)
		if call.Path == "" || !designlint.IsUIFile(call.Path) {
			return true, 0
		}
		paths = []string{call.Path}
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	// Absolute paths are checked from their own folder so the report's paths
	// read from there; relative ones from the current directory.
	ctx, cancel := context.WithTimeout(context.Background(), skill.DesignCheckTimeBudget)
	defer cancel()
	var all []designlint.Finding
	read := 0
	for _, p := range paths {
		checkRoot, target := root, p
		if filepath.IsAbs(p) {
			checkRoot, target = filepath.Dir(p), filepath.Base(p)
		}
		findings, n, err := designlint.Check(ctx, designlint.Options{Root: checkRoot, Ignore: skill.RepoMapIgnores()}, []string{target})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		all = append(all, findings...)
		read += n
	}
	if len(all) == 0 {
		if !hook {
			fmt.Println(skill.RenderDesignFindings(nil, read))
		}
		return true, 0
	}
	fmt.Print(skill.RenderDesignFindings(all, read))
	return true, 2
}
