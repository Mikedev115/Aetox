package tts

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// The property the whole feature rests on: the first piece is small, so the
// wait before any sound at all does not grow with the length of the reply.
func TestFirstPieceIsShortWhateverTheLength(t *testing.T) {
	short := Segment("สวัสดีครับ วันนี้อากาศดีมาก ผมจะเล่าเรื่องยาวให้ฟัง")
	long := Segment(strings.Repeat("ประโยคนี้ยาวพอสมควรและมีเนื้อหาครบถ้วน ", 200))
	if len(short) == 0 || len(long) == 0 {
		t.Fatal("segmenting real text produced nothing")
	}
	if n := runeLen(long[0]); n > ChunkRunes {
		t.Errorf("first piece of a very long text is %d runes, want at most %d", n, ChunkRunes)
	}
	// The point is not that they are equal — it is that the long one did not
	// grow. A first piece the size of the reply is the bug this file exists for.
	if runeLen(long[0]) > 3*FirstChunkRunes {
		t.Errorf("first piece did not stay small: %d runes", runeLen(long[0]))
	}
}

func TestNoPieceExceedsTheWindow(t *testing.T) {
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 60) +
		strings.Repeat("ก", 900) + " จบ"
	for i, p := range Segment(text) {
		// One rune of slack: a piece may take an overflow to avoid leaving a
		// fragment behind, and the join adds a space.
		if n := runeLen(p); n > ChunkRunes+MinChunkRunes+1 {
			t.Errorf("piece %d is %d runes, want at most ~%d", i, n, ChunkRunes)
		}
	}
}

// Nothing is dropped and nothing is reordered: rejoining the pieces gives back
// the input with its whitespace collapsed.
func TestSegmentLosesNothing(t *testing.T) {
	cases := []string{
		"หนึ่ง สอง สาม",
		"One. Two! Three?\nFour\n\nFive",
		strings.Repeat("ทดสอบการอ่านออกเสียงข้อความภาษาไทยที่ยาวมาก ", 40),
		strings.Repeat("word ", 400),
	}
	for _, in := range cases {
		want := strings.Join(strings.Fields(in), " ")
		got := strings.Join(strings.Fields(strings.Join(Segment(in), " ")), " ")
		if got != want {
			t.Errorf("rejoined text differs\n in: %.60q\ngot: %.60q", want, got)
		}
	}
}

// Thai writes no sentence-final period. If the only boundaries this knew were
// terminators, a Thai paragraph would be one piece — the exact failure the
// segmenter exists to prevent, in the language the app is written in.
func TestThaiSplitsWithoutAnyPeriod(t *testing.T) {
	para := strings.Repeat("ผมกำลังทดสอบเสียงอ่านภาษาไทยของแอปนี้อยู่ตอนนี้ ", 12)
	got := Segment(para)
	if len(got) < 3 {
		t.Fatalf("Thai paragraph with no periods split into %d pieces, want several", len(got))
	}
}

func TestTerminatorInsideSomethingIsNotABoundary(t *testing.T) {
	for _, in := range []string{"ค่าคือ 3.14 ครับ", "ไปที่ example.com นะ", "รุ่น 2.15.0 ออกแล้ว"} {
		got := Segment(in)
		if len(got) != 1 {
			t.Errorf("Segment(%q) = %q, want one piece — the dot is not a sentence end", in, got)
		}
	}
}

func TestAbbreviationDoesNotEndASentence(t *testing.T) {
	got := Segment("Ask Dr. Smith about it.")
	if len(got) != 1 {
		t.Errorf("Segment split on an abbreviation: %q", got)
	}
}

// A hard cut is the last resort, and it must land between runes. A byte-sized
// window through Thai produces replacement characters for the engine to try to
// pronounce.
func TestHardCutStaysOnRuneBoundaries(t *testing.T) {
	for _, p := range Segment(strings.Repeat("ก", 2000)) {
		if !utf8.ValidString(p) {
			t.Fatalf("piece is not valid UTF-8: %q", p)
		}
		if strings.ContainsRune(p, utf8.RuneError) {
			t.Fatalf("piece contains a replacement character: %q", p)
		}
	}
}

func TestSegmentOfNothingIsEmptyNotNil(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\n\t"} {
		got := Segment(in)
		if got == nil {
			t.Errorf("Segment(%q) returned nil", in)
		}
		if len(got) != 0 {
			t.Errorf("Segment(%q) = %q, want no pieces", in, got)
		}
	}
}
