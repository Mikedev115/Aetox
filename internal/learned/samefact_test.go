package learned

import "testing"

// The pairs are the owner's own store on 11 ก.ย.: what was proposed again in
// other words after being turned down, and the closest two lines that were
// genuinely different. The threshold has to sit between them.
func TestSameFactSeesThroughRewording(t *testing.T) {
	same := [][2]string{
		{"User communicates in Thai and expects replies in Thai",
			"User communicates primarily in Thai and expects responses in Thai"},
		{"User communicates primarily in Thai and expects responses in Thai",
			"User communicates primarily in Thai and expects responses in Thai."},
		{"User communicates in Thai and expects Thai-language interaction",
			"User communicates in Thai and expects replies in Thai"},
		{"User's name is Tanyong (ตันหยง), born December 15, 2553 BE (2010)",
			"User's name is Tanyong (ตันหยง), born on December 15, 2553 BE (2010)."},
		{"ผู้ใช้ใช้ภาษาไทยในการสื่อสาร", "ผู้ใช้ใช้ภาษาไทยในการสนทนา"},
		{"ผู้ใช้ใช้ระบบปฏิบัติการ Windows และมีโฟลเดอร์ทำงานอยู่ที่ D:\\Aetox\\โต้ะทำงานเอกสาร Aetox",
			"ผู้ใช้ใช้ Windows และมีโฟลเดอร์ทำงานที่ D:\\Aetox\\โต้ะทำงานเอกสาร Aetox"},
	}
	for _, p := range same {
		if !SameFact(p[0], p[1]) {
			t.Errorf("should read as one fact (%.2f):\n  %s\n  %s", Similarity(p[0], p[1]), p[0], p[1])
		}
	}
	different := [][2]string{
		{"ผู้ใช้ต้องการให้ตรวจสอบสุขภาพเครื่อง (ดิสก์ หน่วยความจำ โหลด อุณหภูมิ โปรแกรมที่กินทรัพยากร)",
			"ผู้ใช้ต้องการให้สรุปผลการตรวจสอบเป็นสามส่วน คือ อะไรปกติ อะไรน่าห่วง และแนวทางแก้ไข"},
		{"User's laptop is an HP OMEN (RTX 5050, 115W max TGP) with strong cooling",
			"User's previous laptop was a Gigabyte G6 KF 2024 (RTX 4060 @ 75W)"},
		{"ผู้ใช้เป็นคนพัฒนา Aetox คนเดียว (GitHub Mikedev115)",
			"ผู้ใช้ติดตามราคาและคะแนน benchmark ของโมเดล AI"},
		{"", "ผู้ใช้ใช้ภาษาไทย"},
		// Short lines: a negation is one word, and one word is most of the
		// trigrams. Only an exact match counts under sameFactMinRunes.
		{"จำอันนี้", "ไม่ต้องจำอันนี้"},
	}
	for _, p := range different {
		if SameFact(p[0], p[1]) {
			t.Errorf("should read as two facts (%.2f):\n  %s\n  %s", Similarity(p[0], p[1]), p[0], p[1])
		}
	}
}

func TestSimilarityIgnoresCaseSpacingAndPunctuation(t *testing.T) {
	if s := Similarity("User Prefers  Short Answers.", "user prefers short answers"); s != 1 {
		t.Errorf("case, spacing and a full stop should not separate two lines: %.2f", s)
	}
}
