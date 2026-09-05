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
//
// The first piece is the only one the listener can feel, and the first
// version of this file got it wrong in the language the app is written in.
// It cut every over-long sentence to the FULL window before asking how big
// the first piece was allowed to be — so an English reply, whose first
// sentence ends at a period, opened with a 59-rune piece, and a Thai reply,
// whose whole first paragraph is one sentence to this file, opened with 214.
// Measured on the owner's own reply on 5 ก.ย. 2569, while he was asking why
// ฟัง was slow. Now a sentence is cut to the limit of the piece being built,
// and the first piece's limit is FirstChunkRunes in Thai exactly as it is in
// English.
//
// Where a cut lands follows the owner's kiosk (AI-Robot-Guide,
// AvatarManager._splitTextIntoChunks): at whitespace, never mid-word — Thai
// writes spaces between phrases, which is a better place to pause than most.
// The mid-word cut is kept only for a run with no space to back off to.

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
	// rather than being spoken alone with a gap on either side, and a tail
	// shorter than this rides with the piece before it. It is also the most a
	// piece may run over its window by, for either reason.
	MinChunkRunes = 24
)

// Segment splits text into pieces to synthesize in order. Joining the result
// with spaces gives back the input with its whitespace collapsed — pieces are
// never dropped and never reordered.
func Segment(text string) []string { return segment(text, FirstChunkRunes, ChunkRunes) }

func segment(text string, first, max int) []string {
	out := []string{}
	var cur []rune
	// What the piece being built is held to: first until the first piece is
	// out, max for every one after it.
	limit := first
	flush := func() {
		if s := strings.TrimSpace(string(cur)); s != "" {
			out = append(out, s)
		}
		cur = nil
		limit = max
	}
	take := func(r []rune) {
		if len(cur) > 0 {
			cur = append(cur, ' ')
		}
		cur = append(cur, r...)
	}
	for _, atom := range sentences(text) {
		rest := []rune(atom)
		for len(rest) > 0 {
			// How much the piece being built can still take.
			room := limit - len(cur)
			if len(cur) > 0 {
				room-- // the joining space
			}
			switch {
			case len(rest) <= room, len(rest) < MinChunkRunes:
				// It fits — or it is the stray "ครับ", which overruns this
				// piece by less than a fragment rather than becoming one.
				take(rest)
				rest = nil
			case len(cur) >= MinChunkRunes && (len(rest) <= max || room < MinChunkRunes):
				// A sentence that fits in a piece of its own starts one; it is
				// not cut just to top this piece up. And a piece with less than
				// a fragment of room left is finished — nothing that small has
				// a good place to pause in it.
				flush()
			default:
				// Longer than any piece may be, or behind a piece too short to
				// stand alone: cut off what fits here and carry the rest.
				head, tail := cutAt(rest, room)
				take(head)
				rest = tail
			}
			if len(cur) >= limit {
				flush()
			}
		}
	}
	// A tail too short to be a piece of its own rides with the one before it,
	// when that one has the room to run over.
	if n := len(cur); n > 0 && n < MinChunkRunes && len(out) > 0 {
		last := []rune(out[len(out)-1])
		if len(last)+1+n <= max+MinChunkRunes {
			out[len(out)-1] = string(last) + " " + strings.TrimSpace(string(cur))
			cur = nil
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

// cutAt takes a head of at most window runes off r and returns it with the
// rest. It backs off to the last space in the second half of the window, so
// the cut lands between words, and only a run with no space in that range is
// cut mid-word. A tail too short to be a piece of its own stays with the
// head: the piece runs over by less than a fragment rather than leaving one
// behind.
func cutAt(r []rune, window int) (head, tail []rune) {
	if window < 1 {
		window = 1
	}
	if len(r) <= window {
		return r, nil
	}
	cut := window
	for k := window; k > window/2; k-- {
		if unicode.IsSpace(r[k]) {
			cut = k
			break
		}
	}
	if len(r)-cut < MinChunkRunes {
		return r, nil
	}
	return trimRunes(r[:cut]), trimRunes(r[cut:])
}

func trimRunes(r []rune) []rune {
	i, j := 0, len(r)
	for i < j && unicode.IsSpace(r[i]) {
		i++
	}
	for j > i && unicode.IsSpace(r[j-1]) {
		j--
	}
	return r[i:j]
}
