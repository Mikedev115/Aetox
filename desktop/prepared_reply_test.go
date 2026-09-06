package main

import "testing"

// The gate is the whole feature's budget: everything offersChoice turns away
// costs nothing, and everything it lets through costs one model call. So the
// cases below are written as the two failures that matter — a turn that ended
// by asking something and prepared nothing, and an ordinary "done" turn that
// spent a call for a composer nobody was going to Tab.

func TestATurnThatEndsByAskingSomethingPreparesAReply(t *testing.T) {
	asked := []struct {
		name   string
		answer string
	}{
		{"thai question particle", "แก้ให้แล้วครับ สองจุด จะให้รันเทสต์ทั้งชุดเลยไหม"},
		{"thai fork with no question mark", "ทำได้สองทาง เอาแบบยิงรอบสอง หรือจะเพิ่มเครื่องมือให้โมเดลเรียกเอง"},
		{"thai which-one", "เจอสามไฟล์ที่เข้าข่าย จะเริ่มจากอันไหนดี"},
		{"question mark", "I can do this two ways. Which would you prefer?"},
		{"english offer without a mark", "Both work. Let me know if you want the second one."},
		{
			"a closing question under a list of what was done",
			"เรียบร้อยครับ\n- แก้ Chat.svelte\n- แก้ style.css\n- เพิ่มเทสต์\nเทสต์ผ่านทั้งชุด ให้คอมมิตเลยหรือจะดูดิฟฟ์ก่อน",
		},
	}
	for _, row := range asked {
		t.Run(row.name, func(t *testing.T) {
			if !offersChoice(row.answer) {
				t.Fatalf("offersChoice said no, so no wording is prepared and the user types the answer they were just handed:\n%s", row.answer)
			}
		})
	}
}

func TestAFinishedTurnWithNothingToDecidePreparesNothing(t *testing.T) {
	done := []struct {
		name   string
		answer string
	}{
		{"a plain report", "แก้ให้แล้วครับ Chat.svelte สองจุด เทสต์ผ่านทั้งชุด"},
		{"english report", "Done. Two files changed and the suite is green."},
		{"empty", ""},
		{"whitespace", "   \n  "},
		{
			"a question asked and answered long before the end",
			"คำถามคือไฟล์ไหนที่พัง? คำตอบคือ Chat.svelte บรรทัด 1863 ครับ " +
				"ตัวจับปุ่มไม่ได้ดัก Tab เลยไม่มีอะไรเกิดขึ้น ผมแก้ให้แล้วและรันเทสต์ผ่านหมด " +
				"ตอนนี้กด Tab แล้วรับคำได้ตามที่ออกแบบไว้ ไม่มีอะไรค้างอยู่ในสาขานี้ " +
				"ทั้งหมดอยู่ในคอมมิตเดียว เขียนเหตุผลไว้ในหัวไฟล์เรียบร้อย " +
				"ส่วนสไตล์ชีตแยกอีกไฟล์ตามที่โครงสร้างเดิมวางไว้ทุกประการ " +
				"ผลลัพธ์ตรงกับที่ตกลงกันไว้ตั้งแต่ต้นทุกข้อครับ ไม่มีส่วนไหนที่ต้องรื้อเพิ่ม",
		},
	}
	for _, row := range done {
		t.Run(row.name, func(t *testing.T) {
			if offersChoice(row.answer) {
				t.Fatalf("offersChoice said yes, so an ordinary turn spends a model call on wording nobody will take:\n%s", row.answer)
			}
		})
	}
}

// What comes back from the model lands in the user's own message box, so the
// cleaner is the last thing standing between a sloppy reply and a composer full
// of dim text somebody has to clear by hand.
func TestPreparedWordingIsCleanedBeforeItReachesTheComposer(t *testing.T) {
	got := cleanReplies([]string{
		"  เอาทางแรก  ",
		"เอาทางแรก", // the same wording twice would make the second Tab press look dead
		"",          // a Tab that types nothing
		"   ",       //
		"เอาทางสอง\nแล้วค่อยรันเทสต์ทีหลัง", // a plan, not a line
		"เอาทางสาม",
		"เอาทางสี่", // over the cap of three
	})
	want := []string{"เอาทางแรก", "เอาทางสอง", "เอาทางสาม"}
	if len(got) != len(want) {
		t.Fatalf("kept %d wording(s), want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Text != w {
			t.Fatalf("wording %d is %q, want %q", i, got[i].Text, w)
		}
	}
}

// A paragraph is not a message somebody was about to type, and dim text taller
// than the box it sits in is a bug you cannot see past.
func TestAWordingTooLongToBeAMessageIsDropped(t *testing.T) {
	long := make([]rune, 221)
	for i := range long {
		long[i] = 'ก'
	}
	got := cleanReplies([]string{string(long), "เอาอันนี้"})
	if len(got) != 1 || got[0].Text != "เอาอันนี้" {
		t.Fatalf("expected only the short wording to survive, got %+v", got)
	}
}

// Never nil (§34): the frontend reads .length off this the moment the event
// lands, and a nil slice crosses the Wails bridge as null.
func TestCleanRepliesIsNeverNil(t *testing.T) {
	if got := cleanReplies(nil); got == nil {
		t.Fatal("cleanReplies(nil) is nil — the composer reads .length off it")
	}
}
