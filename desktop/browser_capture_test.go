package main

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
)

// capture asks whose tab it is before it asks the engine for a picture, so a
// session with no page open is told so instead of waiting on a webview.
func TestCaptureRefusesWithNoPageOpen(t *testing.T) {
	a := newTestApp(t)
	a.browsers = &browserHost{app: a, tabs: map[string]*browserTab{}}

	out, err := (&browserCaptureSkill{app: a}).capture(t.Context(), false, false)
	if err == nil {
		t.Fatal("capture answered without a page open")
	}
	if out.Success {
		t.Error("the output claims success")
	}
	if len(out.Images) != 0 {
		t.Error("a failed capture still handed the model an image")
	}
}

// pagePNG is a capture of a given shape. Only the header is ever read, so the
// pixels are left as whatever image.NewRGBA starts with.
func pagePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("encoding the test capture: %v", err)
	}
	return buf.Bytes()
}

// A full-page capture of a fifteen-slide deck came back 1280 x 10800 on
// 30 ส.ค. Every provider downsizes a picture before the model reads it, so a
// column that shape arrives with nothing in it legible — and the model, told
// nothing, reported on slides it could not see.
func TestTallCaptureSaysItCannotBeRead(t *testing.T) {
	note := tallStripNote(pagePNG(t, 1280, 10800))
	if note == "" {
		t.Fatal("a 1280x10800 capture must say the text in it will not be readable")
	}
	if !strings.Contains(note, "10800") || !strings.Contains(note, "1280") {
		t.Errorf("the note should carry the measurements it is about, got %q", note)
	}
}

// The ordinary capture says nothing extra. A note on every screenshot is a note
// nobody reads.
func TestOrdinaryCaptureCarriesNoStripNote(t *testing.T) {
	if note := tallStripNote(pagePNG(t, 1280, 720)); note != "" {
		t.Errorf("a viewport capture got %q, want no note", note)
	}
	if note := tallStripNote(pagePNG(t, 573, 871)); note != "" {
		t.Errorf("a tall-ish page capture got %q, want no note", note)
	}
	if note := tallStripNote([]byte("not a png")); note != "" {
		t.Errorf("unreadable bytes got %q, want no note", note)
	}
}

// A pixel in the picture has to become a point on the page, or a model that
// sees a cell cannot press it. The multiplier is measured against the fitted
// image the model actually sees, and a full-page picture says its y is not a
// viewport coordinate.
func TestCaptureSaysHowItsPixelsMapToThePage(t *testing.T) {
	png := pagePNG(t, 2560, 1400) // a 1280×700 viewport at DPR 2
	view := browserActResult{VW: 1280, VH: 700, DPR: 2}
	note := captureScaleNote(view, png, false)
	for _, want := range []string{"viewport 1280×700", "ภาพ 2560×1400", "DPR 2", "× 0.5"} {
		if !strings.Contains(note, want) {
			t.Errorf("scale note missing %q: %s", want, note)
		}
	}
	if strings.Contains(note, "full") || strings.Contains(note, "ทั้งเอกสาร") {
		t.Errorf("a viewport capture must not warn about document offsets: %s", note)
	}
	full := captureScaleNote(view, png, true)
	if !strings.Contains(full, "ทั้งเอกสาร") {
		t.Errorf("a full capture must say its y is a document offset: %s", full)
	}
	if captureScaleNote(view, []byte("not a png"), false) != "" {
		t.Error("an unreadable picture gets no note rather than a wrong one")
	}
}
