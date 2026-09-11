package main

// The five ways a ref can miss, and the rule they all serve: never blame the
// window for something that was not the window's doing.
//
// browser_ref_miss_test.go pins the same five branches for the browser and
// states the reason: telling a model "the page changed" when in truth it never
// read the page, or used a number from a filtered round, sends it to re-read
// something that was never the problem. The tests below assert the same thing
// negatively — each branch must NOT say the window changed — because that is
// the wrong answer that was easy to give and hard to notice.

import (
	"strings"
	"testing"
)

func node(ref int, name string) reachNode {
	return reachNode{Ref: ref, Name: name, Role: "ปุ่ม", Enabled: true, RuntimeID: []int32{42, int32(ref)}}
}

func missReason(t *testing.T, r *reachRefs, hwnd uintptr, ref int) string {
	t.Helper()
	_, err := r.lookup(hwnd, ref)
	if err == nil {
		t.Fatalf("ref %d resolved when it should have missed", ref)
	}
	return explainReach("click", err)
}

func TestNothingHasBeenReadIsNamedAsSuch(t *testing.T) {
	var r reachRefs
	got := missReason(t, &r, 100, 1)
	if !strings.Contains(got, "ยังไม่ได้") {
		t.Errorf("a chat that never read anything is not told so: %q", got)
	}
	if strings.Contains(got, "เปลี่ยน") {
		t.Errorf("nothing had changed; the window is being blamed for a read that never happened: %q", got)
	}
}

func TestARefFromAnotherWindowIsNamedAsSuch(t *testing.T) {
	var r reachRefs
	r.remember(100, "Notepad", "", 2, []reachNode{node(1, "บันทึก"), node(2, "ยกเลิก")})
	got := missReason(t, &r, 200, 1)
	if !strings.Contains(got, "Notepad") {
		t.Errorf("the refusal does not name the window the ref actually belongs to: %q", got)
	}
	if strings.Contains(got, "เปลี่ยน") {
		t.Errorf("the ref was fine, it was aimed at the wrong window: %q", got)
	}
}

func TestAFilterThatFoundNothingIsNamedWithTheFilter(t *testing.T) {
	var r reachRefs
	r.remember(100, "Notepad", "ยืนยัน", 0, nil)
	got := missReason(t, &r, 100, 1)
	if !strings.Contains(got, "ยืนยัน") {
		t.Errorf("the refusal does not quote the filter that found nothing: %q", got)
	}
	if !strings.Contains(got, "filter") {
		t.Errorf("the refusal does not say to drop the filter: %q", got)
	}
}

func TestAWindowWithNothingActionableIsNamedAsSuch(t *testing.T) {
	var r reachRefs
	r.remember(100, "ตัวแสดงภาพ", "", 0, nil)
	got := missReason(t, &r, 100, 1)
	if strings.Contains(got, "filter") {
		t.Errorf("no filter was used; the refusal invents one: %q", got)
	}
	if !strings.Contains(got, "capture") {
		t.Errorf("a window with no controls should point at the tool that can still show it: %q", got)
	}
}

func TestANumberOutOfRangeSaysWhatTheRangeWas(t *testing.T) {
	var r reachRefs
	r.remember(100, "Notepad", "", 2, []reachNode{node(1, "บันทึก"), node(2, "ยกเลิก")})
	got := missReason(t, &r, 100, 7)
	if !strings.Contains(got, "1-2") {
		t.Errorf("the refusal does not say what the range was: %q", got)
	}
	// The renumbering rule, said where it is needed rather than in general.
	if !strings.Contains(got, "เริ่มที่ 1") {
		t.Errorf("the refusal does not explain that every read renumbers: %q", got)
	}
}

func TestAnOutOfRangeRefFromAFilteredReadSaysTheFilterToo(t *testing.T) {
	var r reachRefs
	r.remember(100, "Notepad", "บันทึก", 9, []reachNode{node(1, "บันทึก")})
	got := missReason(t, &r, 100, 4)
	if !strings.Contains(got, "บันทึก") {
		t.Errorf("the refusal does not quote the filter, which is why the range is short: %q", got)
	}
	if !strings.Contains(got, "1-1") {
		t.Errorf("the refusal does not say what the filtered range was: %q", got)
	}
}

func TestARefThatIsThereResolves(t *testing.T) {
	var r reachRefs
	r.remember(100, "Notepad", "", 2, []reachNode{node(1, "บันทึก"), node(2, "ยกเลิก")})
	got, err := r.lookup(100, 2)
	if err != nil {
		t.Fatalf("a valid ref was refused: %v", err)
	}
	if got.Name != "ยกเลิก" {
		t.Errorf("ref 2 resolved to %q", got.Name)
	}
	if len(got.RuntimeID) == 0 {
		t.Error("the resolved row carries no runtime id, so nothing downstream could find the element again")
	}
}

func TestASecondReadReplacesTheFirst(t *testing.T) {
	var r reachRefs
	r.remember(100, "Notepad", "", 3, []reachNode{node(1, "ก"), node(2, "ข"), node(3, "ค")})
	r.remember(100, "Notepad", "", 1, []reachNode{node(1, "ง")})

	got, err := r.lookup(100, 1)
	if err != nil || got.Name != "ง" {
		t.Fatalf("ref 1 still points at the old read: %v %q", err, got.Name)
	}
	// The stale half must be gone, not merely shadowed. A table that kept the
	// longer previous read would answer ref 3 with a control from a window
	// state that no longer exists, which is the worst possible outcome: a
	// confident click on the wrong thing.
	if _, err := r.lookup(100, 3); err == nil {
		t.Fatal("a ref from the previous read still resolves; the table grew instead of being replaced")
	}
}

func TestForgettingLeavesNoWindowBehind(t *testing.T) {
	var r reachRefs
	r.remember(100, "Notepad", "", 1, []reachNode{node(1, "ก")})
	r.forget()
	if hwnd, _ := r.window(); hwnd != 0 {
		t.Error("the table still names a window after being forgotten")
	}
	// And the diagnosis falls back to the honest first branch rather than to a
	// range from a read that is gone.
	got := missReason(t, &r, 100, 1)
	if !strings.Contains(got, "ยังไม่ได้") {
		t.Errorf("after forgetting, the miss is not reported as nothing-has-been-read: %q", got)
	}
}

func TestAPasswordFieldIsShownAsPresentButItsContentsAreNot(t *testing.T) {
	target := reachTarget{Exe: "app.exe", Title: "เข้าสู่ระบบ"}
	nodes := []reachNode{
		{Ref: 1, Role: "แก้ไข", Name: "ชื่อผู้ใช้", Value: "somebody", Enabled: true},
		{Ref: 2, Role: "แก้ไข", Name: "รหัสผ่าน", Value: "hunter2", Password: true, Enabled: true},
	}
	got := renderRead(target, nodes, len(nodes), "")

	// Named, so a model does not hunt for a field it can see is there.
	if !strings.Contains(got, "รหัสผ่าน") {
		t.Errorf("the password field is not shown at all: %q", got)
	}
	if !strings.Contains(got, "พิมพ์ไม่ได้") {
		t.Errorf("the password field is not marked as one that cannot be typed into: %q", got)
	}
	// Its contents never reach the model. §6.2 of the direction doc is about
	// never typing a credential; reading one back out is the same wound facing
	// the other way, and a transcript keeps what it is shown.
	if strings.Contains(got, "hunter2") {
		t.Fatal("the contents of a password field were rendered into the model's view")
	}
}

func TestATruncatedReadSaysItIsTruncated(t *testing.T) {
	target := reachTarget{Exe: "app.exe", Title: "รายการ"}
	nodes := []reachNode{node(1, "ก"), node(2, "ข")}
	got := renderRead(target, nodes, 400, "")
	if !strings.Contains(got, "400") {
		t.Errorf("a read that showed 2 of 400 rows does not say so: %q", got)
	}
	// A list that does not announce its own truncation gets treated as
	// complete, and a model then reports "there is no Save button" about a
	// window that has one on row 300.
	if !strings.Contains(got, "filter") {
		t.Errorf("the truncation note does not say how to narrow: %q", got)
	}
}
