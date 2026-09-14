package designlint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rules(t *testing.T, rel, src string) map[string][]Finding {
	t.Helper()
	out := make(map[string][]Finding)
	for _, f := range CheckSource(rel, []byte(src)) {
		out[f.Rule] = append(out[f.Rule], f)
	}
	return out
}

func want(t *testing.T, got map[string][]Finding, rule string, line int) {
	t.Helper()
	for _, f := range got[rule] {
		if f.Line == line {
			return
		}
	}
	t.Errorf("%s at line %d missing; got %+v", rule, line, got)
}

func none(t *testing.T, got map[string][]Finding, rule string) {
	t.Helper()
	if len(got[rule]) > 0 {
		t.Errorf("%s should not fire: %+v", rule, got[rule])
	}
}

// Each rule on the line it is for, and on the line it must let pass. The
// positives are the shapes measured on this app's own frontend on 14 ก.ย.
// 2569 (14 findings in 72 files, every one of them real); the negatives are
// the neighbours that looked alike.
func TestRulesOnCSS(t *testing.T) {
	got := rules(t, "a.css", strings.Join([]string{
		/* 1 */ ".hero h1 { background: linear-gradient(90deg, #f00, #00f); -webkit-background-clip: text; }",
		/* 2 */ ".plain { background-clip: padding-box; }",
		/* 3 */ ".node.done { box-shadow: 0 0 8px #10b981; }",
		/* 4 */ ".card { box-shadow: 0 2px 8px rgba(0,0,0,.2); }",
		/* 5 */ ".lift { box-shadow: 0 4px 12px #10b981; }",
		/* 6 */ ".callout { border-radius: 8px; border-left: 4px solid #6366f1; }",
		/* 7 */ ".rule { border-left: 4px solid #e5e7eb; }",
		/* 8 */ ".quiet { border-left: 3px solid var(--line); }",
		/* 9 */ ".stripe { box-shadow: inset 4px 0 0 #6366f1; }",
		/* 10 */ ".top { border-radius: 12px; border-top: 4px solid #f59e0b; }",
		/* 11 */ ".flat { border-top: 4px solid #f59e0b; }",
		/* 12 */ ".ai { background: linear-gradient(135deg, #8b5cf6 0%, #3b82f6 100%); }",
		/* 13 */ ".warm { background: linear-gradient(135deg, #f59e0b, #ef4444); }",
		/* 14 */ "body { font-family: Inter, system-ui, sans-serif; }",
		/* 15 */ "code { font-family: 'JetBrains Mono', monospace; }",
		/* 16 */ ".pop { animation: pop-in .4s cubic-bezier(.2, .9, .3, 1.3) both; }",
		/* 17 */ ".ease { transition: transform .2s cubic-bezier(.2, .8, .2, 1); }",
		/* 18 */ ".grow { transition: height .3s ease; }",
		/* 19 */ ".every { transition: all .3s ease; }",
		/* 20 */ ".move { transition: transform .3s, opacity .3s; }",
		/* 21 */ ".tex { background: repeating-linear-gradient(45deg, #000 0 2px, #fff 2px 4px); }",
		/* 22 */ ".grid { background-image: linear-gradient(#ddd 1px, transparent 1px), linear-gradient(90deg, #ddd 1px, transparent 1px); background-size: 24px 24px; }",
		/* 23 */ ".mark { font-size: 8px; }",
		/* 24 */ ".meta { font-size: 12px; }",
		/* 25 */ ".shake { animation: wobble 1s infinite; }",
	}, "\n"))

	want(t, got, "gradient-text", 1)
	if len(got["gradient-text"]) != 1 {
		t.Errorf("padding-box is not a text clip: %+v", got["gradient-text"])
	}
	want(t, got, "dark-glow", 3)
	for _, f := range got["dark-glow"] {
		if f.Line == 4 || f.Line == 5 {
			t.Errorf("a neutral or an offset shadow is not a glow: %+v", f)
		}
	}
	want(t, got, "side-tab", 6)
	want(t, got, "side-tab", 9)
	for _, f := range got["side-tab"] {
		if f.Line == 7 {
			t.Errorf("a neutral border is a border: %+v", f)
		}
		if f.Line == 8 {
			t.Errorf("a colour this file cannot resolve is not judged: %+v", f)
		}
	}
	want(t, got, "border-accent-on-rounded", 10)
	for _, f := range got["border-accent-on-rounded"] {
		if f.Line == 11 {
			t.Errorf("no radius, no accent-on-rounded: %+v", f)
		}
	}
	want(t, got, "ai-color-palette", 12)
	for _, f := range got["ai-color-palette"] {
		if f.Line == 13 {
			t.Errorf("orange to red is not the palette: %+v", f)
		}
	}
	want(t, got, "overused-font", 14)
	for _, f := range got["overused-font"] {
		if f.Line == 15 {
			t.Errorf("JetBrains Mono is not on the list: %+v", f)
		}
	}
	want(t, got, "bounce-easing", 16)
	want(t, got, "bounce-easing", 25)
	for _, f := range got["bounce-easing"] {
		if f.Line == 17 {
			t.Errorf("an ease-out inside [0,1] is not a bounce: %+v", f)
		}
	}
	want(t, got, "layout-transition", 18)
	for _, f := range got["layout-transition"] {
		if f.Line == 19 || f.Line == 20 {
			t.Errorf("`all` and transform/opacity are not layout transitions: %+v", f)
		}
	}
	want(t, got, "repeating-stripes-gradient", 21)
	want(t, got, "codex-grid-background", 22)
	want(t, got, "tiny-text", 23)
	for _, f := range got["tiny-text"] {
		if f.Line == 24 {
			t.Errorf("12px is not tiny: %+v", f)
		}
	}
}

// The glow written against a token: resolved from the file's own custom
// properties, one level and then one more.
func TestGlowResolvesTheFilesOwnTokens(t *testing.T) {
	got := rules(t, "t.css", ":root { --accent: #6366f1; --ring: var(--accent); }\n.x { box-shadow: 0 0 24px var(--ring); }\n.y { box-shadow: 0 0 24px var(--elsewhere); }\n")
	want(t, got, "dark-glow", 2)
	for _, f := range got["dark-glow"] {
		if f.Line == 3 {
			t.Errorf("a token defined in another file is unresolved, and unresolved is never a finding: %+v", f)
		}
	}
}

func TestMarkupRules(t *testing.T) {
	got := rules(t, "C.svelte", strings.Join([]string{
		/* 1 */ "<script>",
		/* 2 */ "  const label = '⚠️ careful'",
		/* 3 */ "  // border-left: 6px solid #f00 in a comment",
		/* 4 */ "</script>",
		/* 5 */ "<div class=\"rounded-lg border-l-4 border-indigo-500\">",
		/* 6 */ "  <span>⚠️ {warn}</span>",
		/* 7 */ "  <img src=\"\" alt=\"\">",
		/* 8 */ "  <img {src} alt=\"\">",
		/* 9 */ "  <img src={url} alt=\"\">",
		/* 10 */ "  <div class=\"bg-gradient-to-r from-violet-500 to-blue-500 bg-clip-text\">x</div>",
		/* 11 */ "</div>",
		/* 12 */ "<style>",
		/* 13 */ "  /* box-shadow: 0 0 30px #f0f — commented out */",
		/* 14 */ "  .k { animation: pulse 1s; }",
		/* 15 */ "</style>",
	}, "\n"))
	want(t, got, "side-tab", 5)
	want(t, got, "glyph-icon", 6)
	for _, f := range got["glyph-icon"] {
		if f.Line == 2 {
			t.Errorf("an emoji in a script string is not a glyph on screen: %+v", f)
		}
	}
	want(t, got, "broken-image", 7)
	for _, f := range got["broken-image"] {
		if f.Line == 8 || f.Line == 9 {
			t.Errorf("Svelte's {src} and src={x} are sources: %+v", f)
		}
	}
	want(t, got, "gradient-text", 10)
	want(t, got, "ai-color-palette", 10)
	for rule, fs := range got {
		for _, f := range fs {
			if f.Line == 3 || f.Line == 13 {
				t.Errorf("%s fired on a comment: %+v", rule, f)
			}
		}
	}
	none(t, got, "bounce-easing")
}

// `design-allow <rule>` on the line or the line above waives that finding
// and nothing else — the logo's pop-in overshoot is a choice, and this is
// how the file says so.
func TestDesignAllowWaivesOneFinding(t *testing.T) {
	got := rules(t, "w.css", strings.Join([]string{
		"/* design-allow bounce-easing */",
		".pop { animation: pop-in .4s cubic-bezier(.2, .9, .3, 1.3); }",
		".x { box-shadow: 0 0 20px #f00; } /* design-allow dark-glow */",
		".y { box-shadow: 0 0 20px #f00; } /* design-allow side-tab */",
		".z { box-shadow: 0 0 20px #f00; } /* design-allow all */",
	}, "\n"))
	none(t, got, "bounce-easing")
	if len(got["dark-glow"]) != 1 || got["dark-glow"][0].Line != 4 {
		t.Errorf("only the glow with no waiver for it should remain: %+v", got["dark-glow"])
	}
}

func TestCheckWalksUIFilesOnly(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("src/a.css", ".x { box-shadow: 0 0 20px #f00; }\n")
	write("src/b.svelte", "<div>fine</div>\n")
	write("src/c.go", "// box-shadow: 0 0 20px #f00\n")
	write("node_modules/x/y.css", ".x { box-shadow: 0 0 20px #f00; }\n")
	findings, read, err := Check(context.Background(), Options{Root: root, Ignore: map[string]bool{"node_modules": true}}, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if read != 2 {
		t.Errorf("read = %d, want the two UI files", read)
	}
	if len(findings) != 1 || findings[0].Path != "src/a.css" || findings[0].Line != 1 {
		t.Errorf("findings = %+v", findings)
	}
	if _, _, err := Check(context.Background(), Options{Root: root}, []string{"missing.css"}); err == nil {
		t.Error("a missing path must be an error, not a clean report")
	}
}
