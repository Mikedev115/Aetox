package skill

import (
	"fmt"
	"strings"
)

// One tool call may carry 300 lines of content. Not because 300 is special,
// but because everything above it is a gamble against a number the model
// cannot see.
//
// The round's output ceiling is what actually decides whether a call survives,
// and it ranges from 512 tokens (a nearly full window) to 65,536 (DeepSeek V4)
// across the providers Aetox speaks to — see providerOutputCeiling and
// clampToWindow in internal/cognitive. A model cannot count its own tokens
// against that, and nothing in the wire tells it where the line is. It can
// count lines, which is the entire reason this cap is stated in lines.
//
// 300 against the ceiling met most often (8,192: DeepSeek V3, OpenRouter,
// Groq, and any provider we have no figure for):
//
//	code, ~12 tokens/line   →  ~3,600   room to spare
//	HTML, ~20 tokens/line   →  ~6,000   fits, getting tight
//	Thai prose, ~35/line    → ~10,500   over
//
// So the cap is not a promise that 300 lines always fits. Thai runs about one
// token per two characters (the same measurement clampToWindow is built on),
// and the ceiling is shared with reasoning and whatever the model says before
// the call. What the cap does is keep ordinary work — code and markup, which
// is most of what write is for — inside one call on every provider.
//
// It is a number the model is told, not a gate the content is refused at.
// Content that reaches the tool at all has parsed as JSON, which means the
// call survived the ceiling; the only failure the cap exists to prevent has
// by then not happened. Refusing anyway was measured on 2026-09-04 (§221): a
// 438-line page arrived whole after 134 seconds, was thrown away, and was
// bought back in three more rounds. So a call over the cap is written, and
// the result carries a note saying it was a gamble that happened to pay.
//
// Chosen over the tighter numbers considered (100, 250) because the two costs
// are not symmetric. Too low spends an extra round on every large file, for
// ever, and that was already decided against once: deepseekV4OutputTokenMax
// was raised to 64K precisely so a whole file need not be split, since every
// split resends the full context. Too high risks one cut-off round when a
// file genuinely overflows, and only then.
const contentLineCap = 300

// A line is counted at ordinary width: every started run of contentLineWidth
// characters is one line. The cap exists to stand in for tokens, and a line
// carries tokens in proportion to its length, not its count — so a cap that
// counted newlines was a cap the model could satisfy by removing them. It did.
// An outside tester (13 ก.ย., GPT-5.6 at low effort) got a server.ts of 9
// lines, the longest 1,822 characters, from a session that had been told
// "300 lines per call" and "lines are the one unit you can count": correct,
// under the cap, and unreadable. Counting width takes the incentive away, and
// says so where the number is stated, because the number is the only thing the
// model was ever optimising against.
//
// 120 rather than 80: the point is to stop a file being folded into a few
// enormous lines, not to fine a formatter's occasional long one. Ordinary code
// stays at one line per line; a 1,822-character line is sixteen.
const contentLineWidth = 120

// contentLines counts what the cap counts: lines at ordinary width. A file
// ending in a newline is not credited with a phantom last line, and an empty
// line is still one line.
func contentLines(s string) int {
	if s == "" {
		return 0
	}
	lines := 0
	for _, line := range strings.SplitAfter(s, "\n") {
		if line == "" {
			continue // the phantom after a trailing newline
		}
		width := len([]rune(strings.TrimSuffix(line, "\n")))
		lines += max(1, (width+contentLineWidth-1)/contentLineWidth)
	}
	return lines
}

// physicalLines is the newline count the model sees in its editor, for the
// note to name when it differs from what was counted.
func physicalLines(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		lines++
	}
	return lines
}

// contentLineCapNote is what a call over the cap is told, appended to a result
// that otherwise reads as success. Empty within the cap.
//
// It rides on every door that takes content, not just write's: a cap that only
// watched write would be routed around by one enormous append, which is the
// same content through a different name. The remedy is the same either way,
// and it is the reason the cap is affordable at all — append continues a file
// without re-sending it.
//
// When long lines are what put the call over, the note says so and names the
// remedy for that instead: the model that reads "send the first 300 lines"
// about a 9-line file learns nothing it can use.
func contentLineCapNote(field, content string) string {
	lines := contentLines(content)
	if lines <= contentLineCap {
		return ""
	}
	physical := physicalLines(content)
	if physical < lines {
		return fmt.Sprintf(
			"Note: %s was %d lines, but %d lines at ordinary width (a line counts once per %d characters), "+
				"over the %d-line guide for one call. It was written whole because it arrived intact, but "+
				"long lines cost the same output tokens as the short ones they replace, so packing code into "+
				"few lines does not make a call smaller, only unreadable. Lay code out the way its formatter "+
				"would, and split a long file with edit mode=append.",
			field, physical, lines, contentLineWidth, contentLineCap)
	}
	return fmt.Sprintf(
		"Note: %s was %d lines, over the %d-line guide for one call. It was written whole because it "+
			"arrived intact, but a call this size is a gamble on the round's output limit, and a losing one "+
			"is cut off mid-JSON and cannot run. Next time send the first %d lines, then continue the file "+
			"with edit mode=append.",
		field, lines, contentLineCap, contentLineCap)
}
