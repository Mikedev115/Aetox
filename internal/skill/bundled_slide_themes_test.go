package skill

// The themes have to be readable, and that is arithmetic rather than taste.
//
// aetox-slide-templates ships six palettes in themes/. Each is a :root block a
// deck pastes in whole, and swapping one changes every colour on every layout
// in the folder at once — which is the point of them, and also the way a single bad
// value goes wrong everywhere instead of on one slide. A layout that breaks is
// visible in the room the moment somebody looks at it. A palette that breaks is
// not: #767c86 caption text on a #f4f3ef stage looks fine on the laptop it was
// picked on and disappears off the back wall of the room it is projected in,
// and the person who cannot read it is never the person who chose it.
//
// So the numbers each theme file claims in its own comment are recomputed here
// from the shipped values, against every ground the theme's own --stage-bg can
// actually produce: each flat colour in the gradient, and each translucent stop
// composited over each of those, and then the panel composited over all of it.
// That is what made the pass worth running rather than eyeballing — it is the
// panel-over-the-bright-end-of-the-gradient ground, which nobody would think to
// sample by hand, that the house red failed on at 3.70:1 and that put
// --accent-text in the token set.
//
// The bars are WCAG 2.1 contrast minimums, chosen per role rather than one
// number for everything, because the roles are genuinely different sizes: a
// heading on these slides is never below 44px and is large text by any
// reading, while a 13px footer and a 16px kicker are not, whatever the room
// they end up projected in.
//
// --line is deliberately not in here. It is a hairline between table rows and a
// rail behind numbered steps: decoration, and on those rails the meaning is
// carried by the accent-bordered dot sitting on top of it, not by the rail. Held
// to 3:1 it stops being a hairline and starts being a border.

import (
	"io/fs"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const themeDir = bundledSkillRoot + "/aetox-slide-templates/themes"

// Every token a theme owes, and the reason the list is checked rather than
// assumed: a theme missing one does not fail loudly, it silently inherits
// whatever the skeleton shipped, so a light theme keeps one dark value and only
// that one slide looks wrong.
var themeTokenNames = []string{
	"accent", "accent-text", "accent-ink", "stage", "stage-bg",
	"line", "stroke", "text", "body", "muted", "panel",
}

// role -> the contrast it owes. Sizes are the skeleton's own.
var themeBars = map[string]float64{
	"text":        3.0, // h1 96px, h2 64px: large text under any reading
	"body":        4.5, // p 19-22px
	"muted":       4.5, // captions and footers, 13px
	"accent-text": 4.5, // the kicker at 16px, a tag at 14px
}

type themeColor struct{ r, g, b, a float64 }

var (
	hexRe  = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)
	rgbaRe = regexp.MustCompile(`rgba?\(([^)]*)\)`)
	tokRe  = regexp.MustCompile(`--([a-z-]+)\s*:\s*([^;]+);`)
)

func parseThemeColor(s string) (themeColor, bool) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") && len(s) >= 7 {
		v, err := strconv.ParseUint(s[1:7], 16, 32)
		if err != nil {
			return themeColor{}, false
		}
		return themeColor{float64(v >> 16 & 255), float64(v >> 8 & 255), float64(v & 255), 1}, true
	}
	m := rgbaRe.FindStringSubmatch(s)
	if m == nil {
		return themeColor{}, false
	}
	parts := strings.Split(m[1], ",")
	if len(parts) < 3 {
		return themeColor{}, false
	}
	var n [3]float64
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
		if err != nil {
			return themeColor{}, false
		}
		n[i] = v
	}
	a := 1.0
	if len(parts) > 3 {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err != nil {
			return themeColor{}, false
		}
		a = v
	}
	return themeColor{n[0], n[1], n[2], a}, true
}

// over paints fg onto an opaque ground, the way the browser does before any of
// this is looked at. Ratios computed against a translucent colour instead of
// against what it composites to are the usual way a checker agrees with itself
// and disagrees with the screen.
func over(fg, ground themeColor) themeColor {
	return themeColor{
		fg.r*fg.a + ground.r*(1-fg.a),
		fg.g*fg.a + ground.g*(1-fg.a),
		fg.b*fg.a + ground.b*(1-fg.a),
		1,
	}
}

func relLum(c themeColor) float64 {
	f := func(v float64) float64 {
		v /= 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*f(c.r) + 0.7152*f(c.g) + 0.0722*f(c.b)
}

func contrast(a, b themeColor) float64 {
	x, y := relLum(a), relLum(b)
	if x < y {
		x, y = y, x
	}
	return (x + 0.05) / (y + 0.05)
}

func themeFiles(t *testing.T) map[string]string {
	t.Helper()
	entries, err := fs.ReadDir(bundledSkillFS, themeDir)
	if err != nil {
		t.Fatalf("aetox-slide-templates ships no themes folder: %v", err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".css") {
			continue
		}
		raw, readErr := fs.ReadFile(bundledSkillFS, path.Join(themeDir, e.Name()))
		if readErr != nil {
			t.Fatalf("reading %s: %v", e.Name(), readErr)
		}
		out[e.Name()] = string(raw)
	}
	if len(out) == 0 {
		t.Fatal("aetox-slide-templates ships no themes at all")
	}
	return out
}

func themeTokens(css string) map[string]string {
	out := map[string]string{}
	for _, m := range tokRe.FindAllStringSubmatch(css, -1) {
		out[m[1]] = strings.TrimSpace(m[2])
	}
	return out
}

// themeGrounds is every colour a word in this theme can end up sitting on: the
// flat colours the stage is painted from, every translucent gradient stop
// composited over each of those, and then the panel composited over all of it.
func themeGrounds(tok map[string]string) []themeColor {
	var flats []themeColor
	for _, h := range hexRe.FindAllString(tok["stage-bg"], -1) {
		if c, ok := parseThemeColor(h); ok {
			flats = append(flats, c)
		}
	}
	if c, ok := parseThemeColor(tok["stage"]); ok {
		flats = append(flats, c)
	}
	grounds := append([]themeColor(nil), flats...)
	for _, stop := range rgbaRe.FindAllString(tok["stage-bg"], -1) {
		c, ok := parseThemeColor(stop)
		if !ok {
			continue
		}
		for _, f := range flats {
			grounds = append(grounds, over(c, f))
		}
	}
	if p, ok := parseThemeColor(tok["panel"]); ok {
		for _, g := range append([]themeColor(nil), grounds...) {
			grounds = append(grounds, over(p, g))
		}
	}
	return grounds
}

// A theme that forgets a token does not fail, it half-applies: the deck keeps
// one value from whatever palette it had before, and exactly one thing on the
// slide is the wrong colour.
func TestEverySlideThemeDefinesTheWholePalette(t *testing.T) {
	for name, css := range themeFiles(t) {
		tok := themeTokens(css)
		for _, want := range themeTokenNames {
			if strings.TrimSpace(tok[want]) == "" {
				t.Errorf("%s does not set --%s, so a deck using it keeps one colour from the palette it replaced", name, want)
			}
		}
	}
}

// A theme is a palette, not a stylesheet. The moment one carries a rule of its
// own it can win or lose against the layout depending on which was pasted
// first, and the promise that the order does not matter is gone.
func TestASlideThemeIsAPaletteAndNothingElse(t *testing.T) {
	for name, css := range themeFiles(t) {
		body := stripCSSComments(css)
		for _, sel := range []string{".slide", ".rise", "@media", "@keyframes", "@import"} {
			if strings.Contains(body, sel) {
				t.Errorf("%s carries %q — a theme is a :root block, so that pasting it before or after the layouts reads the same", name, sel)
			}
		}
		if n := strings.Count(body, "{"); n != 1 {
			t.Errorf("%s opens %d blocks, want exactly 1 (:root)", name, n)
		}
		if !strings.Contains(body, ":root") {
			t.Errorf("%s sets its tokens somewhere other than :root, so a deck cannot override it by pasting one later", name)
		}
	}
}

// The whole reason this file exists. Every role, against every ground the theme
// can put behind it.
func TestSlideThemesAreLegible(t *testing.T) {
	for name, css := range themeFiles(t) {
		tok := themeTokens(css)
		grounds := themeGrounds(tok)
		if len(grounds) == 0 {
			t.Errorf("%s: could not work out a single colour its stage paints, so nothing below was actually checked", name)
			continue
		}
		for role, bar := range themeBars {
			fg, ok := parseThemeColor(tok[role])
			if !ok {
				t.Errorf("%s: --%s is %q, which is not a colour this check can read", name, role, tok[role])
				continue
			}
			worst, at := math.Inf(1), themeColor{}
			for _, g := range grounds {
				if r := contrast(over(fg, g), g); r < worst {
					worst, at = r, g
				}
			}
			if worst < bar {
				t.Errorf("%s: --%s reads %.2f:1 against #%02x%02x%02x, want %.1f:1 — a reader at the back of the room loses it",
					name, role, worst, int(at.r), int(at.g), int(at.b), bar)
			}
		}
	}
}

// --accent-ink is the one token measured against the accent instead of against
// the stage, because it is the one that goes on top of it: a CTA, and the
// gradient-accent overlay, both put words on a field of the accent itself.
//
// The bar is 3:1 rather than 4.5:1 and that is a real limit, not a soft one.
// Nothing legible clears 4.5:1 on a saturated mid-red — measured, the shipped
// --accent-text lands between 1.00 and 1.35 there — so the treatments that use
// this token are headline-only and say so in their own files. Holding it to a
// bar it cannot meet would only push somebody to change the accent instead.
func TestTheInkOnAnAccentFieldIsLegible(t *testing.T) {
	for name, css := range themeFiles(t) {
		tok := themeTokens(css)
		accent, okA := parseThemeColor(tok["accent"])
		ink, okI := parseThemeColor(tok["accent-ink"])
		if !okA || !okI {
			t.Errorf("%s: --accent %q / --accent-ink %q is not a colour this check can read",
				name, tok["accent"], tok["accent-ink"])
			continue
		}
		ground := over(accent, themeColor{255, 255, 255, 1})
		if r := contrast(over(ink, ground), ground); r < 3.0 {
			t.Errorf("%s: --accent-ink reads %.2f:1 on --accent, want 3.0:1 — the words on a CTA or a gradient-accent slide sit on that field",
				name, r)
		}
	}
}

// --stroke is the line that draws something, and it is the one thin line in the
// palette that owes a contrast bar.
//
// --line right beside it owes none and says so: it is a hairline between table
// rows and the rail behind numbered steps, and pushed to 3:1 it stops being a
// hairline. The two were one token until the diagram templates arrived — sixteen
// of them landing at once, every box, arrow and axis stroked with --line — and
// on the shipped palettes that measured about 1.1:1. Not invisible on the
// machine it was drawn on. Gone from the back of a room, which is the only
// place a deck is ever read.
//
// 3:1 rather than 4.5:1 because WCAG asks that of a graphic that carries
// meaning rather than of text, and a 2px flowchart box is the former.
func TestSlideStrokesAreVisible(t *testing.T) {
	for name, css := range themeFiles(t) {
		tok := themeTokens(css)
		fg, ok := parseThemeColor(tok["stroke"])
		if !ok {
			t.Errorf("%s: --stroke is %q, which is not a colour this check can read", name, tok["stroke"])
			continue
		}
		worst, at := math.Inf(1), themeColor{}
		for _, g := range themeGrounds(tok) {
			if r := contrast(over(fg, g), g); r < worst {
				worst, at = r, g
			}
		}
		if worst < 3.0 {
			t.Errorf("%s: --stroke reads %.2f:1 against #%02x%02x%02x, want 3.0:1 — the boxes and arrows of a diagram are drawn with it",
				name, worst, int(at.r), int(at.g), int(at.b))
		}
	}
}

// The other half of that split, enforced where it actually goes wrong: a
// template reaching for the hairline when it meant the drawing. Checked on the
// SVG presentation attributes because that is where a diagram is drawn, and
// where all sixteen of them had it.
func TestADiagramIsNotDrawnWithTheHairline(t *testing.T) {
	for name, body := range templateFiles(t) {
		flat := strings.ReplaceAll(body, " ", "")
		for _, attr := range []string{`stroke="var(--line`, `fill="var(--line`} {
			if strings.Contains(flat, attr) {
				t.Errorf("%s: draws with %s… — --line is a hairline held to no contrast bar; a box, an arrow or an axis wants var(--stroke,var(--line,…))",
					name, attr)
			}
		}
	}
}

// The index and the folder, in both directions, for the same reason the layout
// table is checked that way: the model reads the table, never the folder.
func TestTheThemeIndexMatchesTheFolder(t *testing.T) {
	body := bundledDoc(t, "aetox-slide-templates").body
	for name := range themeFiles(t) {
		if !strings.Contains(body, "themes/"+name) {
			t.Errorf("themes/%s ships and SKILL.md never names it, so nothing will open it", name)
		}
	}
	for _, ref := range strings.Split(body, "`themes/") {
		end := strings.Index(ref, "`")
		if end <= 0 || !strings.HasSuffix(ref[:end], ".css") {
			continue
		}
		if _, ok := themeFiles(t)[ref[:end]]; !ok {
			t.Errorf("SKILL.md points at themes/%s, which is not shipped", ref[:end])
		}
	}
}

// stripCSSComments drops /* ... */ so the prose in a theme file, which is most
// of it and names the very selectors the check above forbids, is not mistaken
// for the file doing something.
func stripCSSComments(css string) string {
	var b strings.Builder
	for {
		i := strings.Index(css, "/*")
		if i < 0 {
			b.WriteString(css)
			return b.String()
		}
		b.WriteString(css[:i])
		rest := css[i+2:]
		j := strings.Index(rest, "*/")
		if j < 0 {
			return b.String()
		}
		css = rest[j+2:]
	}
}
