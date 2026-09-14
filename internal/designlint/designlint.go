// Package designlint reads UI source — stylesheets, Svelte and HTML files,
// JSX — for the mechanical tells of a page assembled by habit: gradient text,
// the zero-offset glow, the coloured side stripe on a card, the
// purple-to-blue gradient, the font every model reaches for, bouncing
// easing, a transition on a layout property, an image with no source, an
// emoji doing an icon's job. Nothing here needs a browser or a model: every
// rule is a pattern in the file, so the answer is the same every time it is
// asked, and it costs nothing to ask after every edit.
//
// The rule set is impeccable's (github.com/pbakaus/impeccable, Apache-2.0,
// v4.1.0) reduced to what a source file can prove on its own — its detector
// has 61 rules, most of them over a rendered page, and its per-edit hook
// runs a thirteen-rule "immediate tier" of mechanical, unambiguous problems;
// the rules below are the members of that tier and of its slop list that
// read from text, ported by hand from the Rust engine's regex matchers with
// the same ids so a finding here can be looked up there. Three differ and
// say so where they are defined in rules.go: `ai-color-palette` reads hues
// from CSS colours where theirs reads Tailwind class names, because this
// app's frontends write CSS; `tiny-text` reads the declaration where theirs
// reads the computed size in a page; `glyph-icon` is their craft-floor ban
// made mechanical, and is not in their detector at all.
//
// A library with no imports from internal/skill, like internal/repomap and
// for the same reason: the tool wrapper decides where it may look and how the
// result is shown; this package decides only what a finding is.
package designlint

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Finding is one thing worth a line in the report.
type Finding struct {
	// Path is repository-relative, forward slashes.
	Path string
	Line int
	// Rule is the id, impeccable's where the rule is theirs.
	Rule string
	// Detail is the matched text or a one-line reading of it.
	Detail string
	// Advisory marks a rule that names a habit rather than a defect: worth a
	// look, not worth blocking an edit on.
	Advisory bool
}

// Options carries what the host decides and Check must not.
type Options struct {
	// Root is the absolute directory paths are relative to. The caller has
	// already resolved and authorized it — this package never widens a path.
	Root string
	// Ignore lists directory names never entered, on top of dot-dirs which
	// are always skipped.
	Ignore map[string]bool
}

// maxFiles bounds a folder check. A frontend has hundreds of UI files and a
// check of all of them is a few hundred milliseconds; the cap is against a
// path that turned out to be a whole disk.
const maxFiles = 4000

// maxFileBytes skips a bundle or a generated sheet: not source someone
// maintains, and the one kind of file whose findings nobody can act on.
const maxFileBytes = 512 * 1024

// Check reads every UI file under each path (a file, or a folder walked) and
// returns the findings sorted by path and line, with how many files were
// read. Paths are relative to opts.Root. A path that is not a UI file is
// skipped and yields nothing; a missing path is an error, because a clean
// report on a file that does not exist is the one wrong answer.
func Check(ctx context.Context, opts Options, paths []string) ([]Finding, int, error) {
	var out []Finding
	read := 0
	for _, p := range paths {
		abs := filepath.Join(opts.Root, filepath.FromSlash(p))
		info, err := os.Stat(abs)
		if err != nil {
			return nil, read, fmt.Errorf("not found: %s", p)
		}
		if !info.IsDir() {
			if f, ok := checkFile(opts.Root, abs); ok {
				out = append(out, f...)
				read++
			}
			continue
		}
		walkErr := filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if name := d.Name(); path != abs && (strings.HasPrefix(name, ".") || opts.Ignore[name]) {
					return filepath.SkipDir
				}
				return nil
			}
			if read >= maxFiles {
				return filepath.SkipAll
			}
			if f, ok := checkFile(opts.Root, path); ok {
				out = append(out, f...)
				read++
			}
			return nil
		})
		if walkErr != nil {
			return nil, read, walkErr
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Line < out[j].Line
	})
	return out, read, nil
}

// checkFile reads one file if it is UI source and returns its findings; the
// bool says whether it counted as read.
func checkFile(root, abs string) ([]Finding, bool) {
	if kindOf(abs) == kindNone {
		return nil, false
	}
	info, err := os.Stat(abs)
	if err != nil || info.Size() > maxFileBytes {
		return nil, false
	}
	src, err := os.ReadFile(abs)
	if err != nil {
		return nil, false
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		rel = abs
	}
	return CheckSource(filepath.ToSlash(rel), src), true
}

// fileKind is how a file is read: a stylesheet is all CSS; markup carries
// CSS in <style> blocks and inline style attributes beside its elements.
type fileKind int

const (
	kindNone fileKind = iota
	kindCSS
	kindMarkup
)

func kindOf(path string) fileKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".css", ".scss", ".sass", ".less":
		return kindCSS
	case ".svelte", ".html", ".htm", ".vue", ".astro", ".jsx", ".tsx":
		return kindMarkup
	}
	return kindNone
}

// IsUIFile reports whether Check would read this path — the question a
// caller asks before spending a call on a file that cannot have findings.
func IsUIFile(path string) bool { return kindOf(path) != kindNone }

// CheckSource runs every rule over one file's text. rel is only carried into
// the findings. Exported so the wrapper can check a buffer it already holds,
// and so the tests hit the rules without a disk.
func CheckSource(rel string, src []byte) []Finding {
	kind := kindOf(rel)
	if kind == kindNone {
		return nil
	}
	text := blankComments(string(src), kind)
	lines := strings.Split(text, "\n")
	props := customProps(text)
	code := codeLines(lines, kind)

	var out []Finding
	add := func(line int, rule, detail string, advisory bool) {
		if allowed(lines, line-1, rule) {
			return
		}
		out = append(out, Finding{Path: rel, Line: line, Rule: rule, Detail: detail, Advisory: advisory})
	}

	for i, line := range lines {
		n := i + 1
		lower := strings.ToLower(line)
		around := ""
		for _, r := range lineRules {
			if !r.applies(lower) {
				continue
			}
			if around == "" {
				around = blockAround(lines, i)
			}
			for _, hit := range r.check(line, around, props) {
				add(n, r.id, hit, r.advisory)
			}
		}
		if !code[i] {
			for _, hit := range checkGlyphs(line) {
				add(n, "glyph-icon", hit, true)
			}
		}
	}

	// One report per rule per line: a shorthand that matches two patterns
	// of the same rule is one finding, not two.
	seen := make(map[string]bool)
	kept := out[:0]
	for _, f := range out {
		key := fmt.Sprintf("%d\x00%s", f.Line, f.Rule)
		if seen[key] {
			continue
		}
		seen[key] = true
		kept = append(kept, f)
	}
	return kept
}

// allowMarker is the inline waiver: `design-allow <rule>` (or `all`) in a
// comment on the finding's line or the line above keeps that finding out of
// the report — for the blockquote that really wants its stripe, the demo
// that shows the wrong thing on purpose. The marker survives blankComments
// so it is still there when the rules run.
const allowMarker = "design-allow"

func allowed(lines []string, i int, rule string) bool {
	for _, j := range []int{i, i - 1} {
		if j < 0 || j >= len(lines) {
			continue
		}
		idx := strings.Index(lines[j], allowMarker)
		if idx < 0 {
			continue
		}
		fields := strings.Fields(lines[j][idx+len(allowMarker):])
		if len(fields) == 0 || (fields[0] != "all" && fields[0] != rule) {
			continue
		}
		// A marker at the end of a declaration waives that declaration; only
		// a marker on a line of its own reaches the line below. Otherwise
		// `.x {…} /* design-allow dark-glow */` would also waive `.y` under it.
		if j == i {
			return true
		}
		rest := strings.TrimSpace(lines[j][:idx] + lines[j][idx+len(allowMarker)+len(fields[0])+1:])
		if rest == "" {
			return true
		}
	}
	return false
}

// blankComments replaces the inside of every /* */ and <!-- --> with spaces,
// and a line in markup that is only a // comment, so a rule never fires on
// commented-out CSS. Newlines survive, so line numbers do — and the
// design-allow marker with its argument survives, because the waiver is
// written in exactly the comment being blanked.
func blankComments(s string, kind fileKind) string {
	b := []byte(s)
	blank := func(from, to int) {
		keepFrom, keepTo := -1, -1
		if idx := strings.Index(string(b[from:to]), allowMarker); idx >= 0 {
			keepFrom = from + idx
			keepTo = keepFrom + len(allowMarker)
			for keepTo < to && b[keepTo] == ' ' {
				keepTo++
			}
			for keepTo < to && isRuleChar(b[keepTo]) {
				keepTo++
			}
		}
		for k := from; k < to; k++ {
			if b[k] == '\n' || (k >= keepFrom && k < keepTo) {
				continue
			}
			b[k] = ' '
		}
	}
	for _, pair := range [][2]string{{"/*", "*/"}, {"<!--", "-->"}} {
		open, close := []byte(pair[0]), []byte(pair[1])
		pos := 0
		for {
			start := bytes.Index(b[pos:], open)
			if start < 0 {
				break
			}
			start += pos
			end := bytes.Index(b[start+len(open):], close)
			if end < 0 {
				blank(start, len(b))
				break
			}
			end += start + len(open) + len(close)
			blank(start, end)
			pos = end
		}
	}
	if kind != kindMarkup {
		return string(b)
	}
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "//") && !strings.Contains(line, allowMarker) {
			lines[i] = strings.Repeat(" ", len(line))
		}
	}
	return strings.Join(lines, "\n")
}

func isRuleChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-'
}

// blockAround is the line with up to eight neighbours either side, cut at
// the braces of its own block: that is where border-radius sits when the
// stripe is on a card, and where the gradient sits when the clip is on
// text — and the block above, whose radius is not this block's, stops at
// its closing brace. Markup lines have no braces and get the plain window.
func blockAround(lines []string, i int) string {
	lo := i
	for k := i - 1; k >= 0 && i-k <= 8; k-- {
		if strings.Contains(lines[k], "}") {
			break
		}
		lo = k
		if strings.Contains(lines[k], "{") {
			break
		}
	}
	hi := i
	if !strings.Contains(lines[i], "}") {
		for k := i + 1; k < len(lines) && k-i <= 8; k++ {
			if strings.Contains(lines[k], "{") {
				break
			}
			hi = k
			if strings.Contains(lines[k], "}") {
				break
			}
		}
	}
	if strings.Contains(lines[i], "{") {
		lo = i
	}
	return strings.Join(lines[lo:hi+1], " ")
}

// codeLines marks the lines of a markup file that sit inside <script> or
// <style>, where an emoji is a string in code and not a glyph on screen. A
// stylesheet is code throughout.
func codeLines(lines []string, kind fileKind) []bool {
	code := make([]bool, len(lines))
	if kind != kindMarkup {
		for i := range code {
			code[i] = true
		}
		return code
	}
	inside := false
	for i, line := range lines {
		l := strings.ToLower(line)
		if strings.Contains(l, "<script") || strings.Contains(l, "<style") {
			inside = true
		}
		code[i] = inside
		if strings.Contains(l, "</script>") || strings.Contains(l, "</style>") {
			inside = false
		}
	}
	return code
}
