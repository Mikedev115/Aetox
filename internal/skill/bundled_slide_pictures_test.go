package skill

// The picture half, and the table it is supposed to answer to.
//
// aetox-design-system's data/slide-backgrounds.csv has named a veil for every
// kind of picture slide since long before this folder existed: five in
// overlay_style, four in text_placement. For most of that time none of the nine
// had a line of markup anywhere, which is the same failure the colour
// treatments had and the same one slide-layouts.csv had before
// aetox-slide-templates was written at all — a table full of names the model
// reads, believes, and then has to invent the implementation of at write time,
// differently on every run.
//
// So the guard is the agreement itself, in both directions. A name in the table
// with no rule is a door onto nothing. A rule with no name in the table is a
// treatment nothing will ever select, because the model reads the table.
//
// The second test is the one that has already caught a real bug rather than a
// hypothetical one. full-bleed.html shipped its scrim as
// linear-gradient(0deg, rgba(5,6,8,.94) ...) — the stage colour written out by
// hand — and that was invisible for as long as every theme was dark. The moment
// paper and elevated existed it meant a black veil laid over the photograph
// with dark text printed on top, on a light slide. A scrim is not a colour; it
// is the stage at some opacity, and it has to be mixed from the token or it is
// only correct for the palette it was typed on.

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
)

const (
	overlayFile     = bundledSkillRoot + "/aetox-slide-templates/overlays/photo-overlays.css"
	backgroundsCSV  = bundledSkillRoot + "/aetox-design-system/data/slide-backgrounds.csv"
	colOverlayStyle = 2
	colTextPlace    = 3
)

func readBundled(t *testing.T, name string) string {
	t.Helper()
	raw, err := fs.ReadFile(bundledSkillFS, name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(raw)
}

// column pulls the distinct values of one column out of the shipped CSV.
func column(t *testing.T, csv string, idx int) []string {
	t.Helper()
	seen := map[string]bool{}
	var out []string
	for i, line := range strings.Split(strings.TrimSpace(csv), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		cells := strings.Split(line, ",")
		if len(cells) <= idx {
			t.Fatalf("row %d has %d columns, wanted at least %d: %q", i, len(cells), idx+1, line)
		}
		v := strings.TrimSpace(cells[idx])
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	if len(out) == 0 {
		t.Fatalf("column %d of the table is empty, so nothing below was actually checked", idx)
	}
	sort.Strings(out)
	return out
}

// classesWithPrefix lists the class names the stylesheet actually defines under
// one prefix, so the reverse direction is checked against rules rather than
// against a list written here.
func classesWithPrefix(css, prefix string) []string {
	seen := map[string]bool{}
	var out []string
	for _, chunk := range strings.Split(css, "."+prefix) {
		end := strings.IndexFunc(chunk, func(r rune) bool {
			return !(r == '-' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
		})
		if end <= 0 {
			continue
		}
		name := chunk[:end]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func TestEveryOverlayTheTableNamesHasARule(t *testing.T) {
	css := stripCSSComments(readBundled(t, overlayFile))
	table := readBundled(t, backgroundsCSV)

	for _, pair := range []struct {
		col    int
		prefix string
		what   string
	}{
		{colOverlayStyle, "ov-", "overlay_style"},
		{colTextPlace, "tp-", "text_placement"},
	} {
		named := column(t, table, pair.col)
		defined := map[string]bool{}
		for _, c := range classesWithPrefix(css, pair.prefix) {
			defined[c] = true
		}
		for _, n := range named {
			if !defined[n] {
				t.Errorf("slide-backgrounds.csv asks for %s %q and photo-overlays.css defines no .%s%s — the model reads the table, so that name is a door onto nothing",
					pair.what, n, pair.prefix, n)
			}
			delete(defined, n)
		}
		var extra []string
		for c := range defined {
			extra = append(extra, c)
		}
		sort.Strings(extra)
		for _, c := range extra {
			t.Errorf("photo-overlays.css defines .%s%s and slide-backgrounds.csv never names it in %s, so nothing will ever select it",
				pair.prefix, c, pair.what)
		}
	}
}

// A rule that lays a full-cover layer and paints it has to take its colour from
// the palette. Written out by hand it is correct for exactly the theme it was
// typed on, and silently wrong on the other five.
func TestAFullCoverScrimIsMixedFromTheTokens(t *testing.T) {
	sources := map[string]string{"overlays/photo-overlays.css": readBundled(t, overlayFile)}
	for name, body := range templateFiles(t) {
		sources["slides/"+name] = body
	}

	checked := 0
	for name, body := range sources {
		for _, block := range cssBlocks(stripCSSComments(body)) {
			flat := strings.ReplaceAll(block.decls, " ", "")
			if !strings.Contains(flat, "inset:0") {
				continue
			}
			paints := strings.Contains(flat, "gradient(") ||
				strings.Contains(flat, "background:rgba") ||
				strings.Contains(flat, "background:#")
			if !paints {
				continue
			}
			checked++
			if !strings.Contains(flat, "var(--stage)") && !strings.Contains(flat, "var(--accent)") {
				t.Errorf("%s: %q lays a scrim over the whole slide and never mixes it from --stage or --accent, so it is only right on the palette it was written for",
					name, strings.TrimSpace(block.selector))
			}
		}
	}
	if checked == 0 {
		t.Error("no full-cover scrim was found anywhere, so this test proved nothing — the matcher has gone stale")
	}
}

// The lesson the web templates paid for once: [class*="w-"] matched a substring
// anywhere in the attribute and styled show-panel and arrow-nav, neither of
// which is a width.
func TestOverlayClassesAreNotMatchedBySubstring(t *testing.T) {
	// Not space-stripped, and that is the whole subtlety: the correct selector
	// is [class*=" ov-"], whose leading space is *inside* the string and is the
	// only thing separating it from the broken one. Normalising whitespace here
	// turns the right answer into the wrong answer and fails a correct file,
	// which is what this check did on its first run.
	css := stripCSSComments(readBundled(t, overlayFile))
	for _, bad := range []string{`[class*="ov-"]`, `[class*='ov-']`} {
		if strings.Contains(css, bad) {
			t.Errorf(`photo-overlays.css selects with %s, which matches those letters anywhere in any class name — use :is([class^="ov-"],[class*=" ov-"])`, bad)
		}
	}
	if !strings.Contains(css, `[class^="ov-"]`) {
		t.Error(`photo-overlays.css no longer anchors its overlay selector with [class^="ov-"], so the check above is guarding nothing`)
	}
}

// The picture templates have to be findable from the index like every other
// layout, and the overlay file has to be findable at all.
func TestThePictureTemplatesAndOverlaysAreIndexed(t *testing.T) {
	body := bundledDoc(t, "aetox-slide-templates").body
	for _, want := range []string{
		"slides/visual-split.html",
		"slides/image-grid.html",
		"overlays/photo-overlays.css",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("%s ships and SKILL.md never names it", want)
		}
		if _, err := fs.ReadFile(bundledSkillFS, path.Join(bundledSkillRoot, "aetox-slide-templates", want)); err != nil {
			t.Errorf("SKILL.md names %s and it is not shipped: %v", want, err)
		}
	}
}

type cssBlock struct{ selector, decls string }

// cssBlocks splits a stylesheet, or the <style> inside a template, into
// selector/declaration pairs. Deliberately shallow: there is no nesting in any
// of these files, and a real parser here would be more machinery than the thing
// it guards.
func cssBlocks(css string) []cssBlock {
	var out []cssBlock
	rest := css
	sel := ""
	for {
		open := strings.Index(rest, "{")
		if open < 0 {
			return out
		}
		sel = rest[:open]
		if i := strings.LastIndexAny(sel, "}>"); i >= 0 {
			sel = sel[i+1:]
		}
		rest = rest[open+1:]
		close := strings.Index(rest, "}")
		if close < 0 {
			return out
		}
		out = append(out, cssBlock{selector: strings.TrimSpace(sel), decls: rest[:close]})
		rest = rest[close+1:]
	}
}
