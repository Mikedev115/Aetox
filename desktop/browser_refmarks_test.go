package main

// Tests for the refs drawn onto a capture (browser_refmarks.go).
//
// The script is asserted as text, the way browser_read_test.go asserts
// textScript: there is no webview in a unit test, and the properties that
// matter here are properties of the script itself — where it mounts, what it
// reads, and above all what it does NOT do.

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

// The one rule the whole feature rests on: the numbers in the picture are the
// numbers `read` handed out, read back off the attribute. A script that
// assigned its own would produce the worst bug available here — the model reads
// the right number off the picture and presses a different control — and it
// would look correct on every page where DOM order and viewport order agree,
// which is most of them.
func TestRefMarksNeverAssignARefOfTheirOwn(t *testing.T) {
	js := refMarkScript("tok", refMarkCap)

	if strings.Contains(js, "setAttribute") {
		t.Error("the mark script writes an attribute; refs belong to textScript alone")
	}
	if !strings.Contains(js, `querySelectorAll('[data-aetox-ref]')`) {
		t.Error("the mark script does not read the refs that are already on the page")
	}
	if !strings.Contains(js, "getAttribute('data-aetox-ref')") {
		t.Error("the mark script does not read each element's own ref number")
	}
}

// The three rules browser_marks.go states for anything Aetox draws on somebody
// else's page, because a native WebView2 window composites above the app and
// nothing the app draws over it is visible.
func TestRefMarksObeyTheOverlayRules(t *testing.T) {
	js := refMarkScript("tok", refMarkCap)

	for _, want := range []string{
		"document.documentElement", // not body: pages put transforms on body
		"position:fixed",           // the layer is pinned to the viewport
		"pointer-events:none",      // the user can still use the page under it
		refMarkLayerID,             // one element holds everything
	} {
		if !strings.Contains(js, want) {
			t.Errorf("the mark script is missing %q", want)
		}
	}
	// innerHTML is what a page with Trusted Types enforced refuses, and this
	// has to work on exactly those pages. The word itself appears in the
	// script's own comment saying so, which is why this looks for the write.
	if strings.Contains(js, "innerHTML=") {
		t.Error("the mark script builds a node with innerHTML")
	}
	if !strings.Contains(js, "textContent=String(bx.ref)") {
		t.Error("the chip's number is not set as text")
	}
}

// Off-frame refs are stamped across the whole document; only the ones in the
// picture may be drawn, and the count of what was in view has to come back so
// the answer can say what the cap left out.
func TestRefMarksOnlyDrawWhatIsInTheFrame(t *testing.T) {
	js := refMarkScript("tok", 12)

	if !strings.Contains(js, "b.bottom<=0||b.top>=vh||b.right<=0||b.left>=vw") {
		t.Error("the mark script does not skip elements outside the viewport")
	}
	if !strings.Contains(js, "CAP=12") {
		t.Error("the cap the caller asked for did not reach the script")
	}
	if !strings.Contains(js, "marks:drawn,count:inview") {
		t.Error("the script does not report which refs it drew and how many were in view")
	}
	// By ref rather than by scan order, or the cap would keep whichever shadow
	// root happened to be walked first instead of the controls a read lists.
	if !strings.Contains(js, "boxes.sort(") {
		t.Error("the marks are not ordered by ref before the cap is applied")
	}
}

func TestClearRefMarksTakesTheLayerOff(t *testing.T) {
	js := clearRefMarksScript("tok")
	if !strings.Contains(js, refMarkLayerID) || !strings.Contains(js, "removeChild") {
		t.Errorf("the clear script does not remove the layer:\n%s", js)
	}
}

// The key is the half that makes a marked capture better than either answer
// that existed before: a picture with names, in one call, that cannot disagree
// with itself.
func TestRefMarkLegendNamesEveryNumberInThePicture(t *testing.T) {
	els := []browserElement{
		{Ref: 3, Tag: "input", Role: "textbox", Text: "ค้นหา", Focused: true},
		{Ref: 7, Tag: "button", Text: "ส่ง"},
		{Ref: 9, Tag: "a", Text: "หน้าถัดไป"},
	}
	got := refMarkLegend([]int{7, 3}, els, 2)

	for _, want := range []string{`[3] textbox "ค้นหา"`, `[7] button "ส่ง"`, "คีย์บอร์ดอยู่ที่นี่"} {
		if !strings.Contains(got, want) {
			t.Errorf("legend missing %q:\n%s", want, got)
		}
	}
	// Sorted, so the key reads in the order the numbers appear in a read.
	if strings.Index(got, "[3]") > strings.Index(got, "[7]") {
		t.Errorf("legend is not in ref order:\n%s", got)
	}
	// A ref that was not drawn is not in the picture, so it is not in its key.
	if strings.Contains(got, "[9]") {
		t.Errorf("legend names a ref that was never drawn:\n%s", got)
	}
}

// A crowded screen is cut, and the cut is reported: the model cannot tell a
// crowded picture from a complete one by looking at it.
func TestRefMarkLegendSaysWhatTheCapLeftOut(t *testing.T) {
	els := []browserElement{{Ref: 1, Tag: "button", Text: "หนึ่ง"}}
	got := refMarkLegend([]int{1}, els, 40)

	if !strings.Contains(got, "40") || !strings.Contains(got, "read") {
		t.Errorf("legend does not say how many were left unmarked:\n%s", got)
	}
}

// A page that paints its own content has no refs to draw, and the picture is
// then exactly the case coordinates exist for. Saying so is the difference
// between a model that switches to x,y and one that asks for marks again.
func TestRefMarkLegendOnAPageWithNothingToMark(t *testing.T) {
	got := refMarkLegend(nil, nil, 0)
	if !strings.Contains(got, "x,y") {
		t.Errorf("an unmarkable page is not pointed at the aim that still works:\n%s", got)
	}
}

// The two flags, and the two cases where what was asked for is not what should
// be done. Both give way with a sentence rather than silently.
func TestCaptureFramingRefusesThePairsThatCannotWork(t *testing.T) {
	t.Run("marks need eyes", func(t *testing.T) {
		full, marks, notes := captureFraming(false, true, false)
		if marks {
			t.Error("marks were drawn for a model that cannot see them")
		}
		if full {
			t.Error("full was turned on by a call that did not ask for it")
		}
		if len(notes) != 1 || !strings.Contains(notes[0], "read") {
			t.Errorf("notes = %v, want one line pointing at read", notes)
		}
	})

	t.Run("marks outrank full", func(t *testing.T) {
		full, marks, notes := captureFraming(true, true, true)
		if full {
			t.Error("a full-page picture was taken with marks on it — the labels would be unreadable")
		}
		if !marks {
			t.Error("marks gave way to full; the caller asked to press something")
		}
		if len(notes) != 1 {
			t.Errorf("notes = %v, want one line saying why the frame changed", notes)
		}
	})

	t.Run("neither flag conflicts", func(t *testing.T) {
		for _, tc := range []struct{ full, marks, sees bool }{
			{full: true, sees: true},
			{marks: true, sees: true},
			{sees: true},
			{full: true},
		} {
			full, marks, notes := captureFraming(tc.full, tc.marks, tc.sees)
			if full != tc.full || marks != tc.marks || len(notes) != 0 {
				t.Errorf("captureFraming(%+v) changed a request it had no reason to: full=%v marks=%v notes=%v",
					tc, full, marks, notes)
			}
		}
	})
}

// The file on disk is named for what it is. A gallery where the page and the
// page-with-numbers-on-it are both page-4.png is a gallery in which nobody can
// answer "what was the agent looking at when it clicked that".
func TestMarkedShotIsNamedForWhatItIs(t *testing.T) {
	a := seed(&App{cfg: config.Config{SandboxRoot: t.TempDir()}}, &conversation{id: "s1"})

	plain, err := a.writeBrowserShot([]byte("PNG"), false)
	if err != nil {
		t.Fatal(err)
	}
	marked, err := a.writeBrowserShot([]byte("PNG marked"), true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "-marks") {
		t.Errorf("plain shot = %q", plain)
	}
	if !strings.HasSuffix(marked, "-marks.png") {
		t.Errorf("marked shot = %q, want it named for the marks", marked)
	}
}
