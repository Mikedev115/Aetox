package tts

// Cutting a reply into speakable pieces — the first half of reading a long
// answer aloud without making the listener wait for the whole thing.
//
// Nothing above this file used to know what a sentence was. Engine's contract
// is "text in, one audio file out", so a 4000-character reply was one call
// that returned nothing at all until the last word had been synthesized. The
// fix is not a faster engine; it is asking the engine for less at a time.
//
// The cut is by RUNE, never by byte: Thai is three bytes a character, and a
// byte-counted window would land inside one and hand the engine a replacement
// character to pronounce.
//
// Thai is also why the boundaries here are not the usual list. Thai writes no
// sentence-final period — a Thai paragraph ends its sentences with a space, or
// with nothing but a line break. So this looks for THREE kinds of break, in
// falling order of confidence: a line break, a sentence terminator followed by
// whitespace, and, only when a piece is still too long for either, a space.
// The last resort is a hard cut mid-word, which costs a pause in the wrong
// place and is still better than the wait it replaces.

import (
	"strings"
	"unicode"
)

// Chunk sizes, in runes.
//
// FirstChunkRunes is deliberately much smaller than the rest: it is the only
// number the listener can feel. Time-to-first-audio is the synthesis time of
// this many characters and nothing else, so it is short enough to come back
// fast and long enough to still be a phrase rather than a fragment. The pieces
// after it are sized for natural delivery instead — by the time one of them is
// needed, the one before it is already playing.
const (
	FirstChunkRunes = 90
	ChunkRunes      = 240
	// MinChunkRunes stops a stray "ครับ" or "OK!" from becoming a piece of its
	// own. A piece shorter than this takes the overflow from the next one
	// rather than being spoken alone with a gap on either side.
	MinChunkRunes = 24
)

// Segment splits text into pieces to synthesize in order. Joining the result
// with spaces gives back the input with its whitespace collapsed — pieces are
// never dropped and never reordered.
func Segment(text string) []string { return segment(text, FirstChunkRunes, ChunkRunes) }

func segment(text string, first, max int) []string {
	out := []string{}
	cur := []rune(nil)
	limit := first
	flush := func() {
		if s := strings.TrimSpace(string(cur)); s != "" {
			out = append(out, s)
		}
		cur = nil
		// Only the first piece gets the small window; everything after it is
		// synthesized while something else is playing.
		limit = max
	}
	for _, atom := range sentences(text) {
		for _, piece := range fit(atom, max) {
			r := []rune(piece)
			switch {
			case len(cur) == 0:
				cur = r
			// Too short to stand alone: take the overflow rather than flush a
			// fragment.
			case len(cur) < MinChunkRunes, len(cur)+1+len(r) <= limit:
				cur = append(append(cur, ' '), r...)
			default:
				flush()
				cur = r
			}
			if len(cur) >= limit {
				flush()
			}
		}
	}
	flush()
	return out
}

// sentences splits at the breaks worth trusting: every line break, and every
// sentence terminator that is followed by whitespace or by the end of the
// text. A terminator with no space after it is inside something — a decimal, a
// URL, a version number — and is not a boundary.
func sentences(text string) []string {
	out := []string{}
	runes := []rune(text)
	start := 0
	emit := func(end int) {
		if s := strings.TrimSpace(string(runes[start:end])); s != "" {
			out = append(out, s)
		}
	}
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\n' {
			emit(i)
			start = i + 1
			continue
		}
		if !isTerminator(runes[i]) {
			continue
		}
		// Run past a cluster — "?!", "...", "…" — so the whole thing stays
		// with the sentence it ends.
		j := i
		for j+1 < len(runes) && isTerminator(runes[j+1]) {
			j++
		}
		i = j
		if j+1 < len(runes) && !unicode.IsSpace(runes[j+1]) {
			continue
		}
		if runes[j] == '.' && looksAbbreviated(runes[start:j]) {
			continue
		}
		emit(j + 1)
		start = j + 1
	}
	emit(len(runes))
	return out
}

func isTerminator(r rune) bool {
	switch r {
	case '.', '!', '?', '…', '。', '！', '？':
		return true
	}
	return false
}

// looksAbbreviated guesses whether the period ending this text closes an
// abbreviation ("Dr.", "e.g.", "U.S.") rather than a sentence, by counting the
// letters in the last word: two or fewer is an abbreviation, more is a word.
//
// It is a guess, and it is allowed to be. Getting it wrong costs one pause in
// the wrong place inside a piece that was going to be spoken anyway — not a
// wrong word, not a dropped one. "etc." is the case it misses, knowingly: a
// list of exceptions would need maintaining in every language Aetox speaks.
func looksAbbreviated(upTo []rune) bool {
	letters := 0
	for i := len(upTo) - 1; i >= 0; i-- {
		r := upTo[i]
		if unicode.IsSpace(r) {
			break
		}
		if unicode.IsLetter(r) {
			letters++
		}
		if letters > 2 {
			return false
		}
	}
	return letters > 0 && letters <= 2
}

// fit cuts a single over-long sentence down to size. It backs off to the last
// space in the second half of the window, so an English break lands between
// words; Thai text with no space in that range falls through to the hard cut,
// which is the one place this package can put a pause inside a word.
func fit(s string, max int) []string {
	runes := []rune(s)
	if len(runes) <= max {
		return []string{s}
	}
	out := []string{}
	for len(runes) > max {
		cut := max
		for k := max; k > max/2; k-- {
			if unicode.IsSpace(runes[k]) {
				cut = k
				break
			}
		}
		if s := strings.TrimSpace(string(runes[:cut])); s != "" {
			out = append(out, s)
		}
		runes = runes[cut:]
	}
	if s := strings.TrimSpace(string(runes)); s != "" {
		out = append(out, s)
	}
	return out
}
