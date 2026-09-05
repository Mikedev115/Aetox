package tts

import (
	"strings"
	"testing"
	"unicode"
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
	if n := runeLen(long[0]); n > FirstChunkRunes {
		t.Errorf("first piece of a very long text is %d runes, want at most %d", n, FirstChunkRunes)
	}
}

// The owner's reply of 5 ก.ย. 2569, the one ฟัง opened with 214 runes of: a
// Thai paragraph is one sentence to this file, and the first cut has to land
// at the first piece's own limit, not the standing one.
func TestThaiParagraphOpensWithAShortPiece(t *testing.T) {
	para := "ได้ครับ ผมจะทำไฟล์ตัวอย่างแล้วเปิดให้ดูในเบราว์เซอร์ โดยจะแสดงพื้นฐานของเว็บสามส่วน เช่น CSS (จัดสี/จัดวาง), การใส่ฟอร์ม และการใช้ JavaScript เพื่อตอบสนองการคลิกของผู้ใช้ ซึ่งทั้งหมดนี้อยู่ในไฟล์เดียวเพื่อให้ดูง่าย และผมจะอธิบายทีละส่วนว่าทำหน้าที่อะไร เริ่มจากโครงสร้าง HTML ที่มีหัวเรื่อง ย่อหน้า และปุ่มหนึ่งปุ่ม"
	got := Segment(para)
	if len(got) < 2 {
		t.Fatalf("a %d-rune paragraph came back as %d piece(s)", runeLen(para), len(got))
	}
	if n := runeLen(got[0]); n > FirstChunkRunes {
		t.Errorf("first piece is %d runes, want at most %d: %q", n, FirstChunkRunes, got[0])
	}
	if n := runeLen(got[0]); n < FirstChunkRunes/2 {
		t.Errorf("first piece is %d runes — cut too eagerly to be a phrase: %q", n, got[0])
	}
}

// English was already right and must stay so: a first sentence shorter than
// the first window is the first piece, whole, with what fits behind it.
func TestEnglishFirstPieceIsItsFirstSentences(t *testing.T) {
	got := Segment("Sure. I will make a sample file and open it in the browser. It shows the three basics of a web page. Then the CSS that sets the background colour, the font and centres the content.")
	if want := "Sure. I will make a sample file and open it in the browser."; got[0] != want {
		t.Errorf("first piece = %q, want %q", got[0], want)
	}
}

// Cuts land at spaces (the owner's kiosk rule): a piece cut out of a Thai
// paragraph ends where the paragraph had a space, never inside a word.
func TestCutsLandBetweenWords(t *testing.T) {
	para := strings.Repeat("ผมกำลังทดสอบเสียงอ่าน ภาษาไทยของแอปนี้ อยู่ตอนนี้ ", 30)
	rest := para
	for i, p := range Segment(para) {
		if !strings.HasPrefix(rest, p) {
			t.Fatalf("piece %d is not the next run of the input: %q", i, p)
		}
		rest = rest[len(p):]
		if next, _ := utf8.DecodeRuneInString(rest); rest != "" && !unicode.IsSpace(next) {
			t.Errorf("piece %d ends mid-word: %q", i, p)
		}
		rest = strings.TrimLeft(rest, " \n")
	}
}

// After the first, pieces are sized for delivery: cutting the first piece
// short must not leave the rest of the paragraph in short pieces too.
func TestLaterPiecesStayFullSized(t *testing.T) {
	para := strings.Repeat("ผมกำลังทดสอบเสียงอ่าน ภาษาไทยของแอปนี้ อยู่ตอนนี้ ", 40)
	got := Segment(para)
	if len(got) < 4 {
		t.Fatalf("got %d pieces, want enough to have a middle", len(got))
	}
	for i := 1; i < len(got)-1; i++ {
		if n := runeLen(got[i]); n < ChunkRunes*3/4 {
			t.Errorf("piece %d is %d runes, want at least %d", i, n, ChunkRunes*3/4)
		}
	}
}

// No fragment is spoken alone: a short closing line rides with the piece
// before it rather than being a piece with a gap on either side.
func TestAShortTailIsNotAPieceOfItsOwn(t *testing.T) {
	text := strings.Repeat("ประโยคนี้ยาวพอสมควรและมีเนื้อหาครบถ้วน ", 30) + "\nขอบคุณครับ"
	got := Segment(text)
	last := got[len(got)-1]
	if runeLen(last) < MinChunkRunes {
		t.Errorf("last piece is a fragment: %q", last)
	}
	if !strings.HasSuffix(last, "ขอบคุณครับ") {
		t.Errorf("the closing line went missing: %q", last)
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
		strings.Repeat("ประโยคนี้ยาวพอสมควรและมีเนื้อหาครบถ้วน ", 30) + "\nขอบคุณครับ",
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
