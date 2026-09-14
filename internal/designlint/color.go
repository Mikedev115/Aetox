package designlint

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// rgb is a colour the rules can reason about. Only the channels: alpha
// never decides whether a shadow is a glow or a stripe is coloured.
type rgb struct{ r, g, b float64 }

var (
	hexColorRe   = regexp.MustCompile(`#([0-9a-fA-F]{3,8})\b`)
	rgbFuncRe    = regexp.MustCompile(`(?i)rgba?\(\s*([0-9.]+)\s*[, ]\s*([0-9.]+)\s*[, ]\s*([0-9.]+)`)
	hslFuncRe    = regexp.MustCompile(`(?i)hsla?\(\s*([0-9.]+)(?:deg)?\s*[, ]\s*([0-9.]+)%?\s*[, ]\s*([0-9.]+)%?`)
	oklchFuncRe  = regexp.MustCompile(`(?i)oklch\(\s*([0-9.]+)%?\s+([0-9.]+)\s+([0-9.]+)`)
	varRefRe     = regexp.MustCompile(`var\(\s*(--[\w-]+)\s*(?:,\s*([^)]*))?\)`)
	customPropRe = regexp.MustCompile(`(--[\w-]+)\s*:\s*([^;{}]+)`)
	colorTokenRe = regexp.MustCompile(`(?i)#[0-9a-f]{3,8}\b|rgba?\([^)]*\)|hsla?\([^)]*\)|oklch\([^)]*\)|\b(?:transparent|currentcolor|black|white|gray|grey|silver|red|blue|green|purple|violet|indigo|orange|pink|cyan|teal|lime|yellow|magenta|navy|maroon|olive|aqua|fuchsia|gold|coral|crimson|tomato|salmon|orchid|plum|khaki|ivory|beige|tan|brown|chocolate|sienna|slategray|lightgray|darkgray|dimgray|whitesmoke|gainsboro)\b`)
)

// namedColors are the keywords a rule meets in practice; a name not here
// is treated as unknown, which no rule counts as chromatic.
var namedColors = map[string]rgb{
	"black": {0, 0, 0}, "white": {255, 255, 255}, "gray": {128, 128, 128}, "grey": {128, 128, 128},
	"silver": {192, 192, 192}, "lightgray": {211, 211, 211}, "darkgray": {169, 169, 169},
	"dimgray": {105, 105, 105}, "whitesmoke": {245, 245, 245}, "gainsboro": {220, 220, 220},
	"slategray": {112, 128, 144},
	"red":       {255, 0, 0}, "blue": {0, 0, 255}, "green": {0, 128, 0}, "purple": {128, 0, 128},
	"violet": {238, 130, 238}, "indigo": {75, 0, 130}, "orange": {255, 165, 0}, "pink": {255, 192, 203},
	"cyan": {0, 255, 255}, "aqua": {0, 255, 255}, "teal": {0, 128, 128}, "lime": {0, 255, 0},
	"yellow": {255, 255, 0}, "magenta": {255, 0, 255}, "fuchsia": {255, 0, 255}, "navy": {0, 0, 128},
	"maroon": {128, 0, 0}, "olive": {128, 128, 0}, "gold": {255, 215, 0}, "coral": {255, 127, 80},
	"crimson": {220, 20, 60}, "tomato": {255, 99, 71}, "salmon": {250, 128, 114}, "orchid": {218, 112, 214},
	"plum": {221, 160, 221}, "khaki": {240, 230, 140}, "ivory": {255, 255, 240}, "beige": {245, 245, 220},
	"tan": {210, 180, 140}, "brown": {165, 42, 42}, "chocolate": {210, 105, 30}, "sienna": {160, 82, 45},
}

// parseColor reads one colour token. oklch is converted through its own
// space so that a modern sheet's accent is as visible to the rules as a hex
// one; hsl likewise. Nil for anything else — a var() left unresolved, a
// color-mix(), a keyword this file does not know.
func parseColor(tok string) *rgb {
	t := strings.TrimSpace(strings.ToLower(tok))
	if m := hexColorRe.FindStringSubmatch(t); m != nil && strings.HasPrefix(t, "#") {
		h := m[1]
		switch len(h) {
		case 3, 4:
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		case 6, 8:
			h = h[:6]
		default:
			return nil
		}
		v, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return nil
		}
		return &rgb{float64(v >> 16), float64(v >> 8 & 0xff), float64(v & 0xff)}
	}
	if m := rgbFuncRe.FindStringSubmatch(t); m != nil {
		return &rgb{num(m[1]), num(m[2]), num(m[3])}
	}
	if m := hslFuncRe.FindStringSubmatch(t); m != nil {
		return hslToRGB(num(m[1]), num(m[2])/100, num(m[3])/100)
	}
	if m := oklchFuncRe.FindStringSubmatch(t); m != nil {
		l := num(m[1])
		if strings.Contains(m[0], "%") || l > 1 {
			l /= 100
		}
		return oklchToRGB(l, num(m[2]), num(m[3]))
	}
	if c, ok := namedColors[t]; ok {
		return &c
	}
	return nil
}

func num(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// chroma is the spread of the channels: impeccable's `has_chroma` at its
// default threshold of 30 is what separates a tinted neutral from a colour.
func (c rgb) chroma() float64 {
	return math.Max(c.r, math.Max(c.g, c.b)) - math.Min(c.r, math.Min(c.g, c.b))
}

func (c rgb) chromatic() bool { return c.chroma() >= 30 }

// hue in degrees, 0–360, the HSL reading.
func (c rgb) hue() float64 {
	r, g, b := c.r/255, c.g/255, c.b/255
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	d := mx - mn
	if d == 0 {
		return 0
	}
	var h float64
	switch mx {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h
}

func (c rgb) hex() string {
	return "#" + strconv.FormatInt(int64(c.r)<<16|int64(c.g)<<8|int64(c.b), 16)
}

func hslToRGB(h, s, l float64) *rgb {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return &rgb{(r + m) * 255, (g + m) * 255, (b + m) * 255}
}

// oklchToRGB is the standard OKLab → linear sRGB → sRGB path, clamped;
// precision past the rules' thresholds (a 30-point spread, a hue band) is
// not needed and not claimed.
func oklchToRGB(l, c, hDeg float64) *rgb {
	h := hDeg * math.Pi / 180
	a := c * math.Cos(h)
	bb := c * math.Sin(h)
	l_ := l + 0.3963377774*a + 0.2158037573*bb
	m_ := l - 0.1055613458*a - 0.0638541728*bb
	s_ := l - 0.0894841775*a - 1.2914855480*bb
	l3, m3, s3 := l_*l_*l_, m_*m_*m_, s_*s_*s_
	lr := 4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3
	lg := -1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3
	lb := -0.0041960863*l3 - 0.7034186147*m3 + 1.7076147010*s3
	gam := func(v float64) float64 {
		v = math.Max(0, math.Min(1, v))
		if v <= 0.0031308 {
			return 12.92 * v * 255
		}
		return (1.055*math.Pow(v, 1/2.4) - 0.055) * 255
	}
	return &rgb{gam(lr), gam(lg), gam(lb)}
}

// customProps collects `--name: value` declared anywhere in the file, first
// declaration wins, so a shadow written as `0 0 24px var(--accent)` is
// judged on the accent's colour. One file only: a token defined in another
// sheet is unresolved here, and an unresolved colour is never a finding.
func customProps(text string) map[string]string {
	props := make(map[string]string)
	for _, m := range customPropRe.FindAllStringSubmatch(text, -1) {
		if _, taken := props[m[1]]; !taken {
			props[m[1]] = strings.TrimSpace(m[2])
		}
	}
	return props
}

// resolveVars substitutes var() references from props, one level deep and
// then once more for a token that names another token; a var with no
// definition here keeps its fallback if it wrote one, else stays as is.
func resolveVars(value string, props map[string]string) string {
	for range 3 {
		if !strings.Contains(value, "var(") {
			break
		}
		value = varRefRe.ReplaceAllStringFunc(value, func(ref string) string {
			m := varRefRe.FindStringSubmatch(ref)
			if v, ok := props[m[1]]; ok {
				return v
			}
			if m[2] != "" {
				return strings.TrimSpace(m[2])
			}
			return ref
		})
	}
	return value
}

// colorsIn returns every colour token in a value that parses to a colour.
func colorsIn(value string) []rgb {
	var out []rgb
	for _, tok := range colorTokenRe.FindAllString(value, -1) {
		if c := parseColor(tok); c != nil {
			out = append(out, *c)
		}
	}
	return out
}

// splitLayers splits a shadow or gradient list on commas outside parens.
func splitLayers(value string) []string {
	var out []string
	depth := 0
	start := 0
	for i, r := range value {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(value[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(value[start:]))
	return out
}
