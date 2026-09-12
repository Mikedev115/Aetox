package main

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"
)

// A sprite source of flat-coloured squares, one colour per key, so a test
// can tell which frame was drawn where.
type fakeSprites struct {
	size   int
	colors map[string]color.RGBA
	asked  []string
}

func (f *fakeSprites) frame(key string) *image.RGBA {
	f.asked = append(f.asked, key)
	c, ok := f.colors[key]
	if !ok {
		return nil
	}
	// Like a baked frame: the figure in the middle, the pad transparent;
	// an icon is a small full square.
	size, pad := f.size, f.size*cFigPad/cSprite
	if strings.HasPrefix(key, "icon-") {
		size, pad = cIcon, 0
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := pad; y < size-pad; y++ {
		for x := pad; x < size-pad; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
		}
	}
	return img
}

// A text painter that is 7px per rune, 17px per line, and paints the lines
// as solid fg bars — enough to see where text landed.
type fakeText struct{}

func (fakeText) measure(s string, _ int) int { return 7 * len([]rune(s)) }
func (fakeText) lineHeight(_ int) int        { return 17 }
func (fakeText) paint(lines []string, _, w, h int, bg, fg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = bg.R, bg.G, bg.B, 255
	}
	for y := 0; y < h && y/17 < len(lines); y++ {
		for x := 0; x < 7*len([]rune(lines[y/17])) && x < w; x++ {
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = fg.R, fg.G, fg.B, 255
		}
	}
	return img
}

var (
	red  = color.RGBA{255, 0, 0, 255}
	blue = color.RGBA{0, 0, 255, 255}
)

func newTestComposer(colors map[string]color.RGBA) (*composer, *fakeSprites, *image.RGBA) {
	sp := &fakeSprites{size: cSprite, colors: colors}
	c := newComposer(sp, fakeText{}, 1)
	w, h := c.canvasSize()
	return c, sp, image.NewRGBA(image.Rect(0, 0, w, h))
}

// The figure sits in the middle of the canvas; the phases of a loop are
// blended by where in the loop the moment falls.
func TestComposerBlendsThePhasesOfALoop(t *testing.T) {
	c, _, dst := newTestComposer(map[string]color.RGBA{
		"idle-p0-open": red, "idle-p1-open": blue, "idle-p2-open": red, "idle-p3-open": blue,
	})
	t0 := time.Unix(100, 0)
	c.draw(dst, companionScene{Pose: "idle"}, t0)
	f := c.figureRect()
	mid := dst.RGBAAt(f.Min.X+f.Dx()/2, f.Min.Y+f.Dy()/2)
	if mid != red {
		t.Fatalf("phase 0 at t=0: %+v", mid)
	}
	// halfway between phase 0 and 1 (3.6s loop → .45s)
	c.draw(dst, companionScene{Pose: "idle"}, t0.Add(450*time.Millisecond))
	mid = dst.RGBAAt(f.Min.X+f.Dx()/2, f.Min.Y+f.Dy()/2)
	if mid.R < 120 || mid.R > 136 || mid.B < 120 || mid.B > 136 {
		t.Fatalf("halfway blend: %+v", mid)
	}
	// outside the sprite the canvas is clear
	if px := dst.RGBAAt(2, 2); px.A != 0 {
		t.Fatalf("canvas corner not transparent: %+v", px)
	}
}

// A new pose crossfades from the old for crossMs, then stands alone.
func TestComposerCrossfadesAPoseChange(t *testing.T) {
	c, _, dst := newTestComposer(map[string]color.RGBA{
		"idle-p0-open": red, "idle-p1-open": red, "idle-p2-open": red, "idle-p3-open": red,
		"thinking-p0-open": blue, "thinking-p1-open": blue, "thinking-p2-open": blue, "thinking-p3-open": blue,
	})
	t0 := time.Unix(100, 0)
	c.draw(dst, companionScene{Pose: "idle"}, t0)
	c.draw(dst, companionScene{Pose: "thinking"}, t0.Add(time.Second))
	f := c.figureRect()
	if px := dst.RGBAAt(f.Min.X+10, f.Min.Y+10); px != red {
		t.Fatalf("the moment of the change still shows the old pose: %+v", px)
	}
	c.draw(dst, companionScene{Pose: "thinking"}, t0.Add(time.Second+crossMs*time.Millisecond/2))
	if px := dst.RGBAAt(f.Min.X+10, f.Min.Y+10); px.R < 100 || px.B < 100 {
		t.Fatalf("mid-crossfade is not a mix: %+v", px)
	}
	c.draw(dst, companionScene{Pose: "thinking"}, t0.Add(2*time.Second))
	if px := dst.RGBAAt(f.Min.X+10, f.Min.Y+10); px != blue {
		t.Fatalf("after the crossfade: %+v", px)
	}
}

// Walking is keyed by the rounded heading and timed from the drag; a blink
// shows the shut frame; a frame that is not baked falls back to idle.
func TestComposerPicksWalkBlinkAndFallback(t *testing.T) {
	c, sp, dst := newTestComposer(map[string]color.RGBA{
		"idle-p0-open": red, "idle-p0-shut": blue, "walk-t90-p0": blue,
	})
	t0 := time.Unix(100, 0)
	c.draw(dst, companionScene{Pose: "idle", Walking: true, Heading: 83}, t0)
	if !asked(sp.asked, "walk-t90-p0") {
		t.Fatalf("walk at 83° asked for %v", sp.asked)
	}
	sp.asked = nil
	c.draw(dst, companionScene{Pose: "idle", Shut: true}, t0)
	f := c.figureRect()
	if px := dst.RGBAAt(f.Min.X+10, f.Min.Y+10); px != blue {
		t.Fatalf("blink did not show the shut frame: %+v", px)
	}
	sp.asked = nil
	c.draw(dst, companionScene{Pose: "coding"}, t0)
	if px := dst.RGBAAt(f.Min.X+10, f.Min.Y+10); px != red {
		t.Fatalf("an unbaked pose did not fall back to idle: %+v", px)
	}
}

// The bubble is an opaque card to the left of the figure — to the right when
// flipped — sized to its wrapped text, and every pixel of the card is opaque
// so GDI text drawn on it survives the layered window.
func TestComposerDrawsTheBubbleBesideTheFigure(t *testing.T) {
	c, _, dst := newTestComposer(map[string]color.RGBA{"idle-p0-open": red})
	theme := CompanionTheme{Bg: "#202020", Fg: "rgb(240, 240, 240)", Border: "#404040"}
	s := companionScene{Pose: "idle", Theme: theme, Shown: "hello there", Words: []string{"hello", " ", "there"}}
	c.draw(dst, s, time.Unix(100, 0))
	f := c.figureRect()
	// the card ends cBubbleGap left of the figure and its bottom is
	// cBubbleBottom above the figure's bottom
	right := f.Min.X - cBubbleGap
	bottom := f.Max.Y - cBubbleBottom
	textW := 7 * len([]rune("hello there"))
	left := right - textW - 2*cBubblePadX
	top := bottom - 17 - 2*cBubblePadY
	inside := dst.RGBAAt(left+cBubblePadX+2, top+cBubblePadY+2)
	if inside.A != 255 || inside != (color.RGBA{240, 240, 240, 255}) {
		t.Fatalf("text pixel inside the card: %+v", inside)
	}
	// (the corners are rounded, so the check keeps clear of them)
	for y := top + 2; y < bottom-2; y++ {
		for x := left + cBubbleRadius; x < right-cBubbleRadius; x++ {
			if dst.RGBAAt(x, y).A != 255 {
				t.Fatalf("card pixel %d,%d not opaque", x, y)
			}
		}
	}
	for x := left + 2; x < right-2; x++ {
		if dst.RGBAAt(x, top+cBubbleRadius).A != 255 || dst.RGBAAt(x, bottom-cBubbleRadius).A != 255 {
			t.Fatalf("card column %d not opaque", x)
		}
	}
	if dst.RGBAAt(right+2, bottom-12-cArrow).A == 0 {
		t.Fatal("no arrow point towards the figure")
	}
	if dst.RGBAAt(left-3, top-3).A != 0 {
		t.Fatal("something drawn outside the card")
	}

	// flipped: the card starts cBubbleGap right of the figure
	s.Flip = true
	c.draw(dst, s, time.Unix(100, 0))
	if dst.RGBAAt(f.Max.X+cBubbleGap+cBubblePadX+2, top+cBubblePadY+2).A != 255 {
		t.Fatal("flipped card is not to the right of the figure")
	}
	if dst.RGBAAt(left+cBubblePadX+2, top+cBubblePadY+2).A != 0 {
		t.Fatal("flipped card still drawn on the left")
	}
}

// Words wrap greedily within the card's width; a word wider than the card is
// cut; the window's segmentation is used only when it adds up to the text.
func TestWrapWords(t *testing.T) {
	width := func(s string) int { return 7 * len([]rune(s)) }
	lines, widest := wrapWords([]string{"สวัสดี", "ครับ", " ", "mike"}, "สวัสดีครับ mike", 7*8, width)
	if strings.Join(lines, "|") != "สวัสดี|ครับ|mike" {
		t.Fatalf("thai wrap: %q", lines)
	}
	if widest != 7*6 {
		t.Fatalf("widest %d", widest)
	}
	lines, _ = wrapWords([]string{"stale", "words"}, "one two three", 7*8, width)
	if strings.Join(lines, "|") != "one two|three" {
		t.Fatalf("fallback wrap: %q", lines)
	}
	lines, _ = wrapWords(nil, "abcdefghijklmnop", 7*5, width)
	if strings.Join(lines, "|") != "abcde|fghij|klmno|p" {
		t.Fatalf("cut wrap: %q", lines)
	}
	lines, _ = wrapWords(nil, "a\nb", 100, width)
	if strings.Join(lines, "|") != "a|b" {
		t.Fatalf("newline wrap: %q", lines)
	}
}

// On hover the frame and both buttons are drawn; muted, only the speaker is.
func TestComposerFrameAndButtons(t *testing.T) {
	c, sp, dst := newTestComposer(map[string]color.RGBA{"idle-p0-open": red, "icon-x": {255, 255, 255, 255}, "icon-volume2": {255, 255, 255, 255}, "icon-volumeX": {255, 255, 255, 255}})
	sp.size = cSprite
	c.draw(dst, companionScene{Pose: "idle"}, time.Unix(100, 0))
	hide, mute := c.buttonRects()
	if dst.RGBAAt(hide.Min.X+11, hide.Min.Y+11).A != 0 || dst.RGBAAt(mute.Min.X+11, mute.Min.Y+11).A != 0 {
		t.Fatal("buttons drawn without hover")
	}
	c.draw(dst, companionScene{Pose: "idle", Hover: true}, time.Unix(100, 0))
	if dst.RGBAAt(hide.Min.X+11, hide.Min.Y+11).A == 0 || dst.RGBAAt(mute.Min.X+11, mute.Min.Y+11).A == 0 {
		t.Fatal("buttons missing on hover")
	}
	f := c.figureRect().Inset(-cFrameInset)
	dashes := 0
	for x := f.Min.X + cFrameRadius; x < f.Max.X-cFrameRadius; x++ {
		if dst.RGBAAt(x, f.Min.Y).A > 0 {
			dashes++
		}
	}
	if dashes == 0 || dashes > f.Dx()-2*cFrameRadius-4 {
		t.Fatalf("top edge is not dashed: %d of %d", dashes, f.Dx()-2*cFrameRadius)
	}
	sp.asked = nil
	c.draw(dst, companionScene{Pose: "idle", Muted: true}, time.Unix(100, 0))
	if dst.RGBAAt(mute.Min.X+11, mute.Min.Y+11).A == 0 || dst.RGBAAt(hide.Min.X+11, hide.Min.Y+11).A != 0 {
		t.Fatal("muted should show the speaker alone")
	}
	if !asked(sp.asked, "icon-volumeX") {
		t.Fatalf("muted icon not used: %v", sp.asked)
	}
}

func TestHopOffsetRisesAndLands(t *testing.T) {
	c := newComposer(&fakeSprites{}, fakeText{}, 2)
	at := time.Unix(100, 0)
	if c.hopOffset(time.Time{}, at) != 0 {
		t.Fatal("no hop should be no lift")
	}
	if got := c.hopOffset(at, at.Add(hopMs*30/100*time.Millisecond)); got != -8 {
		t.Fatalf("peak lift at 2x: %d", got)
	}
	if got := c.hopOffset(at, at.Add(hopMs*time.Millisecond)); got != 0 {
		t.Fatalf("after the hop: %d", got)
	}
}

func asked(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func TestParseColor(t *testing.T) {
	def := color.RGBA{1, 2, 3, 4}
	cases := map[string]color.RGBA{
		"#ff8000":              {255, 128, 0, 255},
		"#fff":                 {255, 255, 255, 255},
		"rgb(31, 31, 35)":      {31, 31, 35, 255},
		"rgba(31, 31, 35, .5)": {31, 31, 35, 128},
		"rgb(1 2 3 / 50%)":     {1, 2, 3, 128},
		"nonsense":             def,
		"":                     def,
	}
	for in, want := range cases {
		if got := parseColor(in, def); got != want {
			t.Errorf("parseColor(%q) = %+v, want %+v", in, got, want)
		}
	}
}

// Crossing to a monitor of another scale before a set is baked for it: the
// frames at hand are resampled to the new box rather than cut off, and the
// composer keeps its clock.
func TestComposerRescalesFramesUntilTheNewSetArrives(t *testing.T) {
	c, sp, dst := newTestComposer(map[string]color.RGBA{"idle-p0-open": red})
	sp.size = cSprite * 2 // frames baked at 200%
	t0 := time.Unix(100, 0)
	c.draw(dst, companionScene{Pose: "idle"}, t0)
	at := c.poseAt
	f := c.figureRect()
	if px := dst.RGBAAt(f.Max.X-3, f.Max.Y-3); px.R < 200 {
		t.Fatalf("a 2x frame was cut at the box's edge: %+v", px)
	}
	c.rescale(2)
	w, h := c.canvasSize()
	dst = image.NewRGBA(image.Rect(0, 0, w, h))
	c.draw(dst, companionScene{Pose: "idle"}, t0.Add(time.Second))
	if c.poseAt != at {
		t.Fatal("rescaling restarted the pose's clock")
	}
	f = c.figureRect()
	if px := dst.RGBAAt(f.Max.X-3, f.Max.Y-3); px != red {
		t.Fatalf("at the frames' own scale they are drawn as they are: %+v", px)
	}
}

// A body opened before any set is named draws nothing and does not fall
// over: the brain names the set a moment later.
func TestComposerDrawsNothingWithoutSprites(t *testing.T) {
	c := newComposer(nil, fakeText{}, 1)
	w, h := c.canvasSize()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	used := c.draw(dst, companionScene{Pose: "idle", Hover: true, Shown: "hi", Words: []string{"hi"}}, time.Unix(100, 0))
	if used.Empty() {
		// the bubble and frame still draw; only the figure is missing
		t.Fatal("nothing at all was drawn")
	}
	c.setSprites(nil)
	c.draw(dst, companionScene{Pose: "idle"}, time.Unix(101, 0))
}
