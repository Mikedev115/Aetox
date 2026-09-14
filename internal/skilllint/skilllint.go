// Package skilllint reads a skill the way the shelf's own rules say one should
// be written (internal/skill/skills/aetox-skill-writing) and reports where it
// falls short, without a browser and without a model.
//
// It exists because the shelf was audited by hand on 2026-09-14 (DECISIONS
// §278) and four faults had sat there for weeks that no eye had caught: a
// routing block for `$ARGUMENTS`, a command this app does not have; a step
// that named a retired tool; a best-practice line that contradicted its own
// file; a door skill with no bar of its own. Three of the four are shapes,
// and a shape is what a program finds on every commit for free.
//
// What it is not: a judge of whether a skill is *good*. Whether a model opens
// a skill at the right moment, and whether it follows the body, are measured
// by running the model, not by reading words (§278, "not done"). This file
// catches the writer's mistakes before the model meets them; it does not
// grade the writer's prose.
//
// Three severities, and the split is the promise a caller can build on:
//
//   - Error: the skill is broken as shipped — a model following it hits a
//     door that is not there, or the file will not parse. A bundled skill
//     with one of these fails the build; a drafted one is shown to the user
//     with the finding attached.
//   - Warn: a shape the skill-writing rules name as a known way to lose the
//     model (a hedge inside a rule, a shouted MUST, an em dash where the
//     prompt forbids one). Reported, never blocking.
//   - Note: worth a look, often right as it is (a long body, no hand-off,
//     a description whose first hundred characters name no moment).
//
// `<!-- lint-allow rule -->` on the offending line or the line above keeps a
// deliberate choice, the same shape as designlint's `design-allow`.
package skilllint

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Mikedev115/Aetox/internal/skill"
)

type Severity int

const (
	Note Severity = iota
	Warn
	Error
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warn:
		return "warn"
	}
	return "note"
}

type Finding struct {
	// File is the path inside the skill folder; "SKILL.md" for the main file.
	File     string
	Line     int // 1-based; 0 when the finding is about the file as a whole
	Rule     string
	Severity Severity
	Message  string
}

func (f Finding) String() string {
	where := f.File
	if f.Line > 0 {
		where = fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return fmt.Sprintf("%s %s [%s] %s", f.Severity, where, f.Rule, f.Message)
}

// Skill is what the linter reads: the folder's files by path, forward
// slashes, "SKILL.md" among them. Name is the folder name when known, so the
// frontmatter can be held to it; empty when a draft has no folder yet.
type Skill struct {
	Name  string
	Files map[string]string
}

// Lint runs every rule. Findings come back ordered by file, then line.
func Lint(s Skill) []Finding {
	var out []Finding
	main, ok := s.Files["SKILL.md"]
	if !ok {
		return []Finding{{File: "SKILL.md", Rule: "missing", Severity: Error, Message: "no SKILL.md in the skill"}}
	}
	out = append(out, lintMain(s, main)...)
	for path, text := range s.Files {
		if path == "SKILL.md" || !strings.HasSuffix(strings.ToLower(path), ".md") {
			continue
		}
		out = append(out, lintText(path, text, false)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			if out[i].File == "SKILL.md" {
				return true
			}
			if out[j].File == "SKILL.md" {
				return false
			}
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// LintFragment reads a piece of a skill that is not a whole file: a section a
// drafter wants appended, a replacement for one paragraph. The frontmatter
// and file-level rules do not apply; the line rules do.
func LintFragment(text string) []Finding {
	return lintText("fragment", text, false)
}

// HasErrors is the one question a gate asks.
func HasErrors(fs []Finding) bool {
	for _, f := range fs {
		if f.Severity == Error {
			return true
		}
	}
	return false
}

// AtLeast keeps findings of the given severity or worse.
func AtLeast(fs []Finding, min Severity) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Severity >= min {
			out = append(out, f)
		}
	}
	return out
}

// Render is the report as text, one finding per line, nothing when there is
// nothing to say.
func Render(fs []Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.String())
		b.WriteByte('\n')
	}
	return b.String()
}

var (
	slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// A moment cue in a description: the words that say *when*, Thai and
	// English. The rule in aetox-skill-writing is that the description says
	// when, not what, and that only the first hundred characters reach the
	// prompt; so the cue is looked for there.
	momentRe = regexp.MustCompile(`(?i)(ตอน|เมื่อ|ก่อน|หลังจาก|ถ้า|ทุกครั้งที่|ขณะ|\bwhen\b|\bbefore\b|\bafter\b|\bonce\b|\bwhile\b|\bif\b|\bany time\b|\bwhenever\b|\bthe moment\b)`)
	// A hedge inside a line that is shaped like a rule: a bullet, a numbered
	// step, or a bold lead. Prose may hedge; a rule that hedges is a rule the
	// model reads as optional, which is what the design shelf was found doing.
	hedgeRe   = regexp.MustCompile(`(?i)\b(if possible|when possible|where possible|ideally|try to|consider\b|you may want|aim for|good enough|if you can|as needed|when appropriate|if applicable)`)
	ruleLine  = regexp.MustCompile(`^\s*([-*]|\d+[.)])\s+|^\s*\*\*`)
	shoutRe   = regexp.MustCompile(`\b(MUST|NEVER|ALWAYS|CRITICAL|IMPORTANT)\b`)
	deadDoors = []*regexp.Regexp{
		regexp.MustCompile(`\$ARGUMENTS`),
		regexp.MustCompile(`\{subcommand\}`),
		regexp.MustCompile(`(?i)\bparse subcommand\b`),
	}
	namedFileRe = regexp.MustCompile(`(^|[^A-Za-z0-9._/-])([A-Za-z0-9._-]+/[A-Za-z0-9._/-]*[A-Za-z0-9._-]+\.[A-Za-z0-9]+)`)
	handoffRe   = regexp.MustCompile("`aetox-[a-z0-9-]+`")
	allowRe     = regexp.MustCompile(`<!--\s*lint-allow\s+([a-z-]+(?:\s*,\s*[a-z-]+)*)\s*-->`)
	fenceRe     = regexp.MustCompile("^\\s*(```|~~~)")
)

const (
	descriptionWindow = 100
	beforeMaxChars    = 160
	longBodyWords     = 1500
)

func lintMain(s Skill, text string) []Finding {
	var out []Finding
	fields, body, err := skill.ParseFrontmatter(text)
	if err != nil {
		out = append(out, Finding{File: "SKILL.md", Rule: "frontmatter", Severity: Error, Message: err.Error()})
		return append(out, lintText("SKILL.md", text, true)...)
	}
	if !strings.HasPrefix(strings.TrimLeft(text, "\n"), "---") {
		out = append(out, Finding{File: "SKILL.md", Rule: "frontmatter", Severity: Error, Message: "no frontmatter; name and description are how the shelf lists a skill"})
	}
	name := strings.TrimSpace(fields["name"])
	switch {
	case name == "":
		out = append(out, Finding{File: "SKILL.md", Rule: "frontmatter", Severity: Error, Message: "frontmatter has no name"})
	case !slugRe.MatchString(name):
		out = append(out, Finding{File: "SKILL.md", Rule: "frontmatter", Severity: Error, Message: fmt.Sprintf("name %q is not a lower-case slug", name)})
	case s.Name != "" && name != s.Name:
		out = append(out, Finding{File: "SKILL.md", Rule: "frontmatter", Severity: Error, Message: fmt.Sprintf("name %q does not match the folder %q", name, s.Name)})
	}
	desc := strings.TrimSpace(fields["description"])
	if desc == "" {
		out = append(out, Finding{File: "SKILL.md", Rule: "frontmatter", Severity: Error, Message: "frontmatter has no description; the shelf lists nothing for it"})
	} else {
		head := []rune(desc)
		if len(head) > descriptionWindow {
			head = head[:descriptionWindow]
		}
		if !momentRe.MatchString(string(head)) {
			out = append(out, Finding{File: "SKILL.md", Rule: "description-moment", Severity: Note,
				Message: "the first 100 characters of the description name no moment (ตอน / เมื่อ / when / before); a description that summarises the work is followed instead of the skill"})
		}
		if strings.Contains(desc, "—") {
			out = append(out, Finding{File: "SKILL.md", Line: lineOf(text, "description:"), Rule: "em-dash", Severity: Warn,
				Message: "em dash in the description, which is the one line every prompt carries"})
		}
	}
	if before, ok := fields["before"]; ok {
		before = strings.TrimSpace(before)
		switch {
		case before == "":
			out = append(out, Finding{File: "SKILL.md", Rule: "before-line", Severity: Warn, Message: "before: is present and empty"})
		case strings.Contains(before, "\n"):
			out = append(out, Finding{File: "SKILL.md", Rule: "before-line", Severity: Warn, Message: "before: spans lines; the prompt draws it as one clause"})
		case len([]rune(before)) > beforeMaxChars:
			out = append(out, Finding{File: "SKILL.md", Rule: "before-line", Severity: Warn, Message: fmt.Sprintf("before: is %d characters; one clause naming the work is the shape", len([]rune(before)))})
		}
	}
	out = append(out, lintText("SKILL.md", text, true)...)

	// Doors: a folder/file.ext the body names, whose folder the skill ships,
	// and which is not there. The same shape internal/skill's bundled test
	// checks, applied to any skill, including one that only exists as text.
	ships := map[string]bool{}
	for path := range s.Files {
		if i := strings.Index(path, "/"); i > 0 {
			ships[path[:i]] = true
		}
	}
	seen := map[string]bool{}
	for _, m := range namedFileRe.FindAllStringSubmatch(body, -1) {
		ref := m[2]
		folder := ref[:strings.Index(ref, "/")]
		if seen[ref] || !ships[folder] {
			continue
		}
		seen[ref] = true
		if _, ok := s.Files[ref]; !ok {
			out = append(out, Finding{File: "SKILL.md", Line: lineOf(text, ref), Rule: "named-file", Severity: Error,
				Message: fmt.Sprintf("names %s and does not ship it", ref)})
		}
	}
	if !handoffRe.MatchString(body) {
		out = append(out, Finding{File: "SKILL.md", Rule: "handoff", Severity: Note,
			Message: "names no other skill; the rules say to name the moment to hand to another skill, by its name"})
	}
	if n := len(strings.Fields(body)); n > longBodyWords {
		out = append(out, Finding{File: "SKILL.md", Rule: "long", Severity: Note,
			Message: fmt.Sprintf("body is %d words; a skill is paid for every time it is opened, and reference material belongs in a file beside it", n)})
	}
	return out
}

// lintText runs the line rules over one file. main says whether the text
// starts with frontmatter, whose lines are skipped by the prose rules.
func lintText(file, text string, main bool) []Finding {
	var out []Finding
	if strings.Contains(text, "\r\n") {
		out = append(out, Finding{File: file, Rule: "crlf", Severity: Error, Message: "CRLF line endings; the repository and the prompt are LF"})
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	if strings.HasPrefix(text, "\uFEFF") {
		out = append(out, Finding{File: file, Rule: "bom", Severity: Error, Message: "starts with a byte-order mark"})
	}
	lines := strings.Split(text, "\n")
	inFront := false
	frontDone := false
	fences := 0
	inFence := false
	for i, raw := range lines {
		n := i + 1
		line := raw
		if main && !frontDone {
			if i == 0 && strings.TrimSpace(line) == "---" {
				inFront = true
				continue
			}
			if inFront {
				if strings.TrimSpace(line) == "---" {
					inFront = false
					frontDone = true
				}
				continue
			}
			frontDone = true
		}
		if fenceRe.MatchString(line) {
			fences++
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		allowed := allowedRules(line, prev(lines, i))
		report := func(rule string, sev Severity, msg string) {
			if allowed[rule] || allowed["all"] {
				return
			}
			out = append(out, Finding{File: file, Line: n, Rule: rule, Severity: sev, Message: msg})
		}
		// A door is a place the model is sent; a mention inside inline code
		// is documentation of the token (aetox-prompts explains $ARGUMENTS
		// to a person writing a prompt file) and stays.
		for _, d := range deadDoors {
			if d.MatchString(stripInlineCode(line)) {
				report("dead-door", Error, "names a command or argument this app does not have ("+d.FindString(line)+")")
				break
			}
		}
		if strings.Contains(line, "—") {
			report("em-dash", Warn, "em dash; the prompt and everything the model reads are written without one (bf5d7e57)")
		}
		// The same courtesy for a word in backticks: a rule about the word
		// "MUST" mentions it, it does not shout it.
		prose := stripInlineCode(line)
		if ruleLine.MatchString(line) {
			if m := hedgeRe.FindString(prose); m != "" {
				report("hedge", Warn, fmt.Sprintf("a rule that hedges (%q) is read as optional; say the rule, or say when it does not apply", strings.TrimSpace(m)))
			}
		}
		if m := shoutRe.FindString(prose); m != "" {
			report("shout", Warn, fmt.Sprintf("%s in capitals; a model told it MUST tells the user that it must instead of doing the work", m))
		}
	}
	if fences%2 == 1 {
		out = append(out, Finding{File: file, Rule: "fence", Severity: Error, Message: "a code fence is opened and never closed; everything after it is read as code"})
	}
	return out
}

var inlineCodeRe = regexp.MustCompile("`[^`]*`")

func stripInlineCode(line string) string { return inlineCodeRe.ReplaceAllString(line, "") }

func prev(lines []string, i int) string {
	if i == 0 {
		return ""
	}
	return lines[i-1]
}

// allowedRules reads a lint-allow comment on the line itself, or on the line
// above when that line is nothing but the comment; a comment at the end of
// one rule does not reach the rule under it.
func allowedRules(line, previous string) map[string]bool {
	out := map[string]bool{}
	sources := []string{line}
	if p := strings.TrimSpace(previous); strings.HasPrefix(p, "<!--") && strings.HasSuffix(p, "-->") {
		sources = append(sources, p)
	}
	for _, src := range sources {
		for _, m := range allowRe.FindAllStringSubmatch(src, -1) {
			for _, r := range strings.Split(m[1], ",") {
				out[strings.TrimSpace(r)] = true
			}
		}
	}
	return out
}

func lineOf(text, needle string) int {
	i := strings.Index(text, needle)
	if i < 0 {
		return 0
	}
	return strings.Count(text[:i], "\n") + 1
}

// Load reads a skill folder from disk: every file's path (so named-file can
// see what ships) and the text of the markdown ones (so the line rules can
// read them). Anything else is carried by path only.
func Load(dir string) (Skill, error) {
	s := Skill{Name: filepath.Base(filepath.Clean(dir)), Files: map[string]string{}}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir && (d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(filepath.Ext(rel), ".md") {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			s.Files[rel] = string(b)
		} else {
			s.Files[rel] = ""
		}
		return nil
	})
	if err != nil {
		return Skill{}, err
	}
	if _, ok := s.Files["SKILL.md"]; !ok {
		return Skill{}, fmt.Errorf("%s: no SKILL.md", dir)
	}
	return s, nil
}

// IsSkillDir reports whether a folder carries a SKILL.md.
func IsSkillDir(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil && !st.IsDir()
}
