package learned

import (
	"strings"
	"unicode"
)

// SameFact reports whether two proposed lines say the same thing — the same
// sentence with a word swapped, a full stop added, "replies" for "responses".
//
// It exists because the queue's dedup key was the body text, byte for byte,
// and the owner's store showed what that lets through (11 ก.ย.): "User
// communicates in Thai and expects replies in Thai" turned down, then
// "…primarily in Thai and expects responses in Thai" proposed, turned down,
// proposed again with a full stop, and "ผู้ใช้ใช้ภาษาไทยในการสนทนา" waiting on
// the page beside "ผู้ใช้ใช้ภาษาไทยในการสื่อสาร" already refused. The person
// reading the cards was being asked the same question in five spellings.
//
// Character trigrams rather than words, because Thai has no spaces to split
// on and the store is half Thai. Dice's coefficient over them, which is what
// the store's own pairs were measured against: every restatement above scored
// 0.61 to 1.00, and the closest pair of genuinely different facts scored 0.55
// — two lines about the same health check that asked for different things.
// The threshold sits in that gap. It is a number chosen from twenty-eight
// pairs on one machine, not a law; the test pins the pairs.
//
// Deliberately not what decides a proposal's fate on its own. An exact match
// is a duplicate; a near match is *reported* — to the tool's caller as which
// line it resembles and what the user said about it — so a model that meant
// something genuinely different can say so in words that differ.
func SameFact(a, b string) bool {
	return Similarity(a, b) >= sameFactThreshold
}

const sameFactThreshold = 0.6

// sameFactMinRunes is the length under which only an exact match counts. A
// trigram set drawn from eight characters is too small to tell "จำอันนี้"
// from "ไม่ต้องจำอันนี้" — they share every trigram of the shorter and score
// 0.63 — and a negation is exactly the word a short line turns on. Every
// restatement in the store that this exists to catch is a sentence.
const sameFactMinRunes = 16

// Similarity is SameFact's number, 0 to 1, exposed so a caller can rank
// several near matches and name the closest.
func Similarity(a, b string) float64 {
	ra, rb := factRunes(a), factRunes(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	if string(ra) == string(rb) {
		return 1
	}
	if len(ra) < sameFactMinRunes || len(rb) < sameFactMinRunes {
		return 0
	}
	ga, gb := trigrams(ra), trigrams(rb)
	shared, total := 0, 0
	for g, n := range ga {
		total += n
		if m, ok := gb[g]; ok {
			shared += min(n, m)
		}
	}
	for _, n := range gb {
		total += n
	}
	return 2 * float64(shared) / float64(total)
}

// factRunes is the comparable form: lowercased, letters, digits and combining
// marks only. Punctuation and spacing are what restatements differ in first.
func factRunes(s string) []rune {
	out := make([]rune, 0, len(s))
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) {
			out = append(out, r)
		}
	}
	return out
}

func trigrams(r []rune) map[string]int {
	out := map[string]int{}
	if len(r) < 3 {
		out[string(r)]++
		return out
	}
	for i := 0; i+3 <= len(r); i++ {
		out[string(r[i:i+3])]++
	}
	return out
}
