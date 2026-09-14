package designlint

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// lineRule is one pattern over one line. `around` is the line with three
// neighbours either side, for the rules whose evidence is split across a
// block (border-radius on the line above the stripe); props are the file's
// custom properties, for the rules that need the colour behind a var().
type lineRule struct {
	id       string
	advisory bool
	// needs are lowercase substrings, any of which a line must contain for
	// the rule's patterns to run at all — a dozen regexes over every line of
	// a 3,000-line component is most of the cost of a check, and none of
	// them can match a line without its keyword.
	needs []string
	check func(line, around string, props map[string]string) []string
}

// lineRules in report order. Ids are impeccable's; the two that are not are
// commented where they are defined.
var lineRules = []lineRule{
	{id: "gradient-text", needs: []string{"clip"}, check: gradientText},
	{id: "dark-glow", needs: []string{"shadow"}, check: darkGlow},
	{id: "side-tab", needs: []string{"border", "shadow"}, check: sideTab},
	{id: "border-accent-on-rounded", needs: []string{"border"}, check: borderAccentOnRounded},
	{id: "ai-color-palette", needs: []string{"gradient", "from-"}, check: aiColorPalette},
	{id: "overused-font", needs: []string{"font"}, check: overusedFont},
	{id: "bounce-easing", needs: []string{"bounce", "animation", "cubic-bezier"}, check: bounceEasing},
	{id: "layout-transition", needs: []string{"transition"}, check: layoutTransition},
	{id: "broken-image", needs: []string{"<img"}, check: brokenImage},
	{id: "repeating-stripes-gradient", advisory: true, needs: []string{"repeating-"}, check: repeatingStripes},
	{id: "codex-grid-background", advisory: true, needs: []string{"background"}, check: gridBackground},
	{id: "tiny-text", advisory: true, needs: []string{"font-size"}, check: tinyText},
}

func (r lineRule) applies(lower string) bool {
	for _, n := range r.needs {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

var (
	bgClipTextRe   = regexp.MustCompile(`(?i)(?:-webkit-)?background-clip\s*:\s*text\b`)
	gradientWordRe = regexp.MustCompile(`(?i)gradient`)
	twClipTextRe   = regexp.MustCompile(`\bbg-clip-text\b`)
	twGradientRe   = regexp.MustCompile(`\bbg-gradient-to-`)
)

// gradientText: text filled with a gradient, the first tell on the list and
// the one every model reaches for on a hero heading. Emphasis is weight or
// size.
func gradientText(line, around string, _ map[string]string) []string {
	if bgClipTextRe.MatchString(line) && gradientWordRe.MatchString(around) {
		return []string{"background-clip: text + gradient"}
	}
	if twClipTextRe.MatchString(line) && twGradientRe.MatchString(around) {
		return []string{"bg-clip-text + bg-gradient"}
	}
	return nil
}

var (
	shadowDeclRe = regexp.MustCompile(`(?i)\b(box-shadow|text-shadow)\s*:\s*([^;{}]+)`)
	lengthRe     = regexp.MustCompile(`(-?[0-9.]+)(px|rem|em)?\b`)
)

// shadowLengths reads the lengths of one shadow layer in order — x, y, blur,
// spread — with colour functions removed first so an rgb() channel is not
// read as a length. rem and em at 16px, the browser default.
func shadowLengths(layer string) []float64 {
	stripped := colorTokenRe.ReplaceAllString(layer, " ")
	stripped = strings.ReplaceAll(strings.ToLower(stripped), "inset", " ")
	var out []float64
	for _, m := range lengthRe.FindAllStringSubmatch(stripped, -1) {
		v := num(m[1])
		if m[2] == "rem" || m[2] == "em" {
			v *= 16
		}
		out = append(out, v)
	}
	return out
}

// darkGlow: a coloured shadow with no offset and a wide blur is a halo, the
// decoration a model adds when it means "premium". impeccable also flags a
// coloured offset shadow on a dark page; that needs the page's root
// background, which one file rarely holds, so only the zero-offset form is
// read here.
func darkGlow(line, _ string, props map[string]string) []string {
	var out []string
	for _, m := range shadowDeclRe.FindAllStringSubmatch(line, -1) {
		value := resolveVars(m[2], props)
		for _, layer := range splitLayers(value) {
			colors := colorsIn(layer)
			if len(colors) == 0 || !colors[0].chromatic() {
				continue
			}
			vals := shadowLengths(layer)
			if len(vals) < 3 || vals[2] <= 4 {
				continue
			}
			if vals[0] == 0 && vals[1] == 0 {
				out = append(out, fmt.Sprintf("zero-offset %s glow (%s)", strings.ToLower(m[1]), colors[0].hex()))
				break
			}
		}
	}
	return out
}

var (
	sideBorderRe      = regexp.MustCompile(`(?i)\bborder-(?:left|right|inline-start|inline-end)\s*:\s*(\d+)px\s+solid\b([^;{}]*)`)
	sideBorderWidthRe = regexp.MustCompile(`(?i)\bborder-(?:left|right|inline-start|inline-end)-width\s*:\s*(\d+)px`)
	sideBorderTwRe    = regexp.MustCompile(`\bborder-[lrse]-(\d+)\b`)
	sideBorderJSRe    = regexp.MustCompile("border(?:Left|Right)\\s*[:=]\\s*[\"'`](\\d+)px\\s+solid")
	roundedTwRe       = regexp.MustCompile(`\brounded(?:-[\w-]+)?\b`)
	roundedNoneRe     = regexp.MustCompile(`\brounded-none\b`)
	borderRadiusRe    = regexp.MustCompile(`(?i)border-radius`)
	// The elements a side border belongs to: a quote, a nav rail, a code
	// block, a link's underline-as-border, an input, an inline span.
	safeElementRe = regexp.MustCompile(`(?i)<(?:blockquote|nav[\s>]|pre[\s>]|code[\s>]|a\s|input[\s>]|span[\s>])`)
)

func hasRounded(around string) bool {
	return roundedTwRe.MatchString(roundedNoneRe.ReplaceAllString(around, ""))
}

// sideTab: the coloured stripe down one edge of a card, list item or alert.
// Three pixels and up, two on a rounded box; a neutral stripe is a border,
// and a colour this file cannot resolve is not judged.
func sideTab(line, around string, props map[string]string) []string {
	var out []string
	if !safeElementRe.MatchString(around) {
		for _, m := range sideBorderRe.FindAllStringSubmatch(line, -1) {
			n := num(m[1])
			colors := colorsIn(resolveVars(m[2], props))
			if len(colors) == 0 || !colors[0].chromatic() {
				continue
			}
			if n >= 3 || (n >= 2 && borderRadiusRe.MatchString(around)) {
				out = append(out, strings.TrimSpace(m[0]))
			}
		}
		for _, m := range sideBorderWidthRe.FindAllStringSubmatch(line, -1) {
			if num(m[1]) >= 3 {
				out = append(out, m[0])
			}
		}
	}
	for _, m := range sideBorderTwRe.FindAllStringSubmatch(line, -1) {
		n := num(m[1])
		if n >= 4 || (n >= 2 && hasRounded(around)) {
			out = append(out, m[0])
		}
	}
	for _, m := range sideBorderJSRe.FindAllStringSubmatch(line, -1) {
		if num(m[1]) >= 3 {
			out = append(out, m[0])
		}
	}
	// The same stripe drawn with an inset shadow: `inset 4px 0 0 <colour>`.
	for _, m := range shadowDeclRe.FindAllStringSubmatch(line, -1) {
		for _, layer := range splitLayers(resolveVars(m[2], props)) {
			if !strings.Contains(strings.ToLower(layer), "inset") {
				continue
			}
			colors := colorsIn(layer)
			vals := shadowLengths(layer)
			if len(colors) == 0 || !colors[0].chromatic() || len(vals) < 3 {
				continue
			}
			if math.Abs(vals[0]) >= 3 && vals[1] == 0 && vals[2] == 0 {
				out = append(out, fmt.Sprintf("inset box-shadow %gpx stripe", math.Abs(vals[0])))
			}
		}
	}
	return out
}

var (
	accentBorderRe   = regexp.MustCompile(`(?i)\bborder-(?:top|bottom)\s*:\s*(\d+)px\s+solid\b`)
	accentBorderTwRe = regexp.MustCompile(`\bborder-[tb]-(\d+)\b`)
)

// borderAccentOnRounded: a thick top or bottom border on a rounded box — the
// stripe's cousin, the "accent" a card gets instead of a reason to exist.
func borderAccentOnRounded(line, around string, _ map[string]string) []string {
	var out []string
	for _, m := range accentBorderRe.FindAllStringSubmatch(line, -1) {
		if num(m[1]) >= 3 && borderRadiusRe.MatchString(around) {
			out = append(out, m[0])
		}
	}
	for _, m := range accentBorderTwRe.FindAllStringSubmatch(line, -1) {
		if num(m[1]) >= 1 && hasRounded(around) {
			out = append(out, m[0])
		}
	}
	return out
}

var (
	twFromPurpleRe = regexp.MustCompile(`\bfrom-(?:purple|violet|indigo)-\d+\b`)
	twToColorRe    = regexp.MustCompile(`\bto-(?:purple|violet|indigo|blue|cyan|pink|fuchsia)-\d+\b`)
)

// aiColorPalette: the purple-to-blue gradient. impeccable reads it off
// Tailwind classes (`from-violet-500 to-blue-500`); that form is kept, and
// beside it — ours — the CSS form, read from the hues of the colours in a
// gradient() value: one stop in the purple band (250°–290°) and another in
// blue, cyan or pink (180°–340°) is the palette whatever the syntax.
func aiColorPalette(line, around string, props map[string]string) []string {
	var out []string
	if twFromPurpleRe.MatchString(line) && twToColorRe.MatchString(around) {
		out = append(out, twFromPurpleRe.FindString(line)+" gradient")
	}
	if strings.Contains(strings.ToLower(line), "gradient(") {
		var purple, partner *rgb
		for _, c := range colorsIn(resolveVars(line, props)) {
			if !c.chromatic() {
				continue
			}
			h := c.hue()
			switch {
			case h >= 250 && h <= 290 && purple == nil:
				purple = &c
			case h >= 180 && h <= 340:
				partner = &c
			}
		}
		if purple != nil && partner != nil && purple.hex() != partner.hex() {
			out = append(out, fmt.Sprintf("purple-to-blue gradient (%s → %s)", purple.hex(), partner.hex()))
		}
	}
	return out
}

var (
	overusedFonts   = []string{"Inter", "Roboto", "Open Sans", "Lato", "Montserrat", "Arial", "Helvetica", "Fraunces", "Geist Sans", "Geist Mono", "Geist", "Mona Sans", "Plus Jakarta Sans", "Space Grotesk", "Recoleta", "Instrument Sans", "Instrument Serif"}
	overusedFontRe  = regexp.MustCompile(`(?i)font-family\s*:\s*['"]?(` + strings.Join(overusedFonts, "|") + `)\b`)
	googleFontsRe   = regexp.MustCompile(`(?i)fonts\.googleapis\.com/css2?\?[^"'\s)<>]*`)
	googleFamilyRe  = regexp.MustCompile(`(?i)family=([^&:]+)`)
	overusedFontSet = func() map[string]bool {
		m := make(map[string]bool)
		for _, f := range overusedFonts {
			m[strings.ToLower(f)] = true
		}
		return m
	}()
)

// overusedFont: the family named first is the family the page is set in,
// and these are the ones every model sets it in. A fallback later in the
// stack is not judged — this reads only what comes right after the colon.
func overusedFont(line, _ string, _ map[string]string) []string {
	var out []string
	for _, m := range overusedFontRe.FindAllStringSubmatch(line, -1) {
		out = append(out, "font-family: "+m[1])
	}
	for _, u := range googleFontsRe.FindAllString(line, -1) {
		for _, fam := range googleFamilyRe.FindAllStringSubmatch(u, -1) {
			name := strings.ReplaceAll(fam[1], "+", " ")
			if overusedFontSet[strings.ToLower(name)] {
				out = append(out, "Google Fonts: "+name)
				break
			}
		}
	}
	return out
}

var (
	twBounceRe    = regexp.MustCompile(`\banimate-bounce\b`)
	animBounceRe  = regexp.MustCompile(`(?i)\banimation(?:-name)?\s*:\s*([^;{}]*\b(?:bounce|elastic|wobble|jiggle|spring)[^;{}]*)`)
	motionTokenRe = regexp.MustCompile(`(?i)[\w-]*(?:bounce|elastic|wobble|jiggle|spring)[\w-]*`)
	cubicBezierRe = regexp.MustCompile(`cubic-bezier\(\s*([-0-9.]+)\s*,\s*([-0-9.]+)\s*,\s*([-0-9.]+)\s*,\s*([-0-9.]+)\s*\)`)
)

// bounceEasing: the overshoot — bounce, elastic, spring — that read as
// playful in 2014. Exponential ease-out from an already-visible default.
func bounceEasing(line, _ string, _ map[string]string) []string {
	var out []string
	if twBounceRe.MatchString(line) {
		out = append(out, "animate-bounce (Tailwind)")
	}
	for _, m := range animBounceRe.FindAllStringSubmatch(line, -1) {
		out = append(out, "animation: "+motionTokenRe.FindString(m[1]))
	}
	for _, m := range cubicBezierRe.FindAllStringSubmatch(line, -1) {
		y1, y2 := num(m[2]), num(m[4])
		if y1 < -0.1 || y1 > 1.1 || y2 < -0.1 || y2 > 1.1 {
			out = append(out, m[0])
		}
	}
	return out
}

var (
	transitionRe = regexp.MustCompile(`(?i)\btransition(?:-property)?\s*:\s*([^;{}]+)`)
	allWordRe    = regexp.MustCompile(`\ball\b`)
	layoutPropRe = regexp.MustCompile(`\b(?:(?:max|min)-)?(?:width|height)\b|\bpadding(?:-(?:top|right|bottom|left))?\b|\bmargin(?:-(?:top|right|bottom|left))?\b`)
)

// layoutTransition: animating width, height, padding or margin reflows the
// page on every frame; transform and opacity are the animatable pair.
// `transition: all` is left alone here — it is its own habit — the same
// reading impeccable makes.
func layoutTransition(line, _ string, _ map[string]string) []string {
	var out []string
	for _, m := range transitionRe.FindAllStringSubmatch(line, -1) {
		val := strings.ToLower(m[1])
		if allWordRe.MatchString(val) {
			continue
		}
		if found := layoutPropRe.FindAllString(val, -1); len(found) > 0 {
			out = append(out, "transition: "+strings.Join(found, ", "))
		}
	}
	return out
}

var (
	imgEmptySrcRe = regexp.MustCompile(`(?i)<img\b[^>]*\bsrc\s*=\s*(?:""|''|"\s+"|'\s+'|"#"|'#')`)
	imgTagRe      = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	srcAttrRe     = regexp.MustCompile(`(?i)\bsrc\s*=|\{src\}`)
)

// brokenImage: an image with no source, or an empty one — the placeholder
// a build leaves behind, and the first thing a user sees.
func brokenImage(line, _ string, _ map[string]string) []string {
	var out []string
	for _, m := range imgEmptySrcRe.FindAllString(line, -1) {
		out = append(out, clip(m, 100))
	}
	for _, tag := range imgTagRe.FindAllString(line, -1) {
		if !srcAttrRe.MatchString(tag) {
			out = append(out, clip(tag, 100))
		}
	}
	return out
}

var repeatingGradientRe = regexp.MustCompile(`(?i)repeating-linear-gradient\(`)

// repeatingStripes: stripes as texture. A background is a surface; stripes
// need a canvas, a map, a blueprint or a ruler under them.
func repeatingStripes(line, _ string, _ map[string]string) []string {
	if repeatingGradientRe.MatchString(line) {
		return []string{"repeating-linear-gradient as texture"}
	}
	return nil
}

var (
	bgDeclRe     = regexp.MustCompile(`(?i)\bbackground(?:-image)?\s*:\s*([^;{}]+)`)
	hairlineRe   = regexp.MustCompile(`(?i)\b[1-3]px\s*,\s*transparent\b|\btransparent\s+calc\(100%\s*-\s*[1-3]px\)`)
	bgSizePxRe   = regexp.MustCompile(`(?i)background-size\s*:[^;{}]*\b\d{1,3}px\b|/\s*\d{1,3}px\b`)
	linearGradRe = regexp.MustCompile(`(?i)linear-gradient\(`)
)

// gridBackground: the hairline grid — two crossed gradients tiled at a
// fixed size — that reads as "technical" without a thing measured on it.
func gridBackground(line, around string, _ map[string]string) []string {
	for _, m := range bgDeclRe.FindAllStringSubmatch(line, -1) {
		if len(linearGradRe.FindAllString(m[1], -1)) >= 2 && hairlineRe.MatchString(m[1]) && bgSizePxRe.MatchString(around) {
			return []string{"hairline grid background"}
		}
	}
	return nil
}

var fontSizeRe = regexp.MustCompile(`(?i)\bfont-size\s*:\s*([0-9.]+)(px|rem)\b`)

// tinyText: a size below ten pixels is below what a browser rule would let
// pass as body text at any context; impeccable reads the computed size in
// a page, which knows whether the element is a caption. This reads the
// declaration, which does not, so it is advisory.
func tinyText(line, _ string, _ map[string]string) []string {
	var out []string
	for _, m := range fontSizeRe.FindAllStringSubmatch(line, -1) {
		v := num(m[1])
		if m[2] == "rem" {
			v *= 16
		}
		if v > 0 && v < 10 {
			out = append(out, "font-size: "+m[1]+m[2])
		}
	}
	return out
}

// checkGlyphs is ours, from impeccable's craft floor rather than its
// detector: "Unicode glyphs or emoji standing in for an icon system" is a
// ban there that no scanner enforced. An emoji in the markup of a
// component is that — icons are drawn, in one stroke and weight. Advisory,
// because a chat transcript or a demo may quote one on purpose.
func checkGlyphs(line string) []string {
	var out []string
	for _, r := range line {
		if isEmoji(r) {
			out = append(out, fmt.Sprintf("emoji %q as a glyph", string(r)))
			break
		}
	}
	return out
}

func isEmoji(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1FAFF: // pictographs, emoticons, symbols, supplemental
		return true
	case r >= 0x2600 && r <= 0x27BF: // misc symbols, dingbats
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF: // flags
		return true
	}
	return false
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
