package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Mikedev115/Aetox/internal/skilllint"
)

// `aetox skill lint [--min note|warn|error] [path ...]` reads skills the way
// internal/skilllint does and prints what it finds, the same exit shape as
// `aetox design`: 0 clean, 2 findings at or above the threshold, 1 a path
// that is not a skill. A path is a skill folder (has SKILL.md) or a folder of
// them; no path means the bundled shelf when run from the repository, else
// the current directory.
//
// The bar the shelf is held to in the build is errors only
// (internal/skilllint's own test over internal/skill/skills); this verb shows
// the warnings and notes too, because a person deciding whether a skill is
// finished wants the whole reading, and a build wants a yes or a no.
func runSkillLintCommand(args []string) (handled bool, code int) {
	if len(args) < 2 || strings.ToLower(strings.TrimSpace(args[0])) != "skill" || strings.ToLower(strings.TrimSpace(args[1])) != "lint" {
		return false, 0
	}
	min := skilllint.Note
	var paths []string
	rest := args[2:]
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--min":
			if i+1 >= len(rest) {
				fmt.Fprintln(os.Stderr, "--min needs note, warn or error")
				return true, 1
			}
			i++
			switch rest[i] {
			case "note":
				min = skilllint.Note
			case "warn":
				min = skilllint.Warn
			case "error":
				min = skilllint.Error
			default:
				fmt.Fprintln(os.Stderr, "--min needs note, warn or error")
				return true, 1
			}
		case "-h", "--help":
			fmt.Println("usage: aetox skill lint [--min note|warn|error] [path ...]\n  path   a skill folder (has SKILL.md) or a folder of skill folders\n  --min  lowest severity to report and to count toward the exit code (default note)")
			return true, 0
		default:
			paths = append(paths, rest[i])
		}
	}
	if len(paths) == 0 {
		if skilllint.IsSkillDir(filepath.Join("internal", "skill", "skills", "aetox")) {
			paths = []string{filepath.Join("internal", "skill", "skills")}
		} else {
			paths = []string{"."}
		}
	}
	var dirs []string
	for _, p := range paths {
		if skilllint.IsSkillDir(p) {
			dirs = append(dirs, p)
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		found := 0
		for _, e := range entries {
			if e.IsDir() && skilllint.IsSkillDir(filepath.Join(p, e.Name())) {
				dirs = append(dirs, filepath.Join(p, e.Name()))
				found++
			}
		}
		if found == 0 {
			fmt.Fprintf(os.Stderr, "%s: not a skill folder and holds none\n", p)
			return true, 1
		}
	}
	sort.Strings(dirs)
	total := 0
	counts := map[skilllint.Severity]int{}
	for _, d := range dirs {
		s, err := skilllint.Load(d)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return true, 1
		}
		fs := skilllint.AtLeast(skilllint.Lint(s), min)
		if len(fs) == 0 {
			continue
		}
		fmt.Printf("%s\n", filepath.ToSlash(d))
		for _, f := range fs {
			counts[f.Severity]++
			fmt.Printf("  %s\n", f)
		}
		total += len(fs)
	}
	if total == 0 {
		fmt.Printf("%d skills read, nothing to report at %s or above\n", len(dirs), min)
		return true, 0
	}
	fmt.Printf("%d skills read: %d error, %d warn, %d note\n", len(dirs), counts[skilllint.Error], counts[skilllint.Warn], counts[skilllint.Note])
	return true, 2
}
