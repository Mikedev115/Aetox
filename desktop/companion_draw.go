package main

// What the desktop body looks like, frame by frame.
//
// Companion.svelte draws the companion with CSS: the sprite, a bubble to its
// left (or right, at the left edge), a dashed frame with two round buttons
// when the pointer is near, a hop when clicked, a crossfade when the pose
// changes. This file draws the same things into an RGBA canvas for the
// window (companion_windows.go) to show, from the baked frames (bake.ts,
// companion_sprites.go) and the state the window reports (companion.go).
// The measurements are the component's, in logical pixels, scaled here.
//
// Pure: a canvas in, a canvas out. Text is the one thing Go cannot draw on
// its own — Thai needs a shaping engine — so it goes through a textPainter
// the platform provides (GDI on Windows, companion_text_windows.go) and a
// test can stand in for.

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
	"time"
)

// Layout, logical pixels (Companion.svelte's numbers).
const (
	cSprite  = 130 // bake.ts FIGURE * BOX / 64
	cFigure  = 104 // Companion.svelte SIZE
	cFigPad  = (cSprite - cFigure) / 2
	cCanvasW = 764
	cCanvasH = 180
	cFigTop  = 44 // room above for the bubble's rise, the hop and the buttons

	cBubbleMaxW   = 300
	cBubbleGap    = 10
	cBubblePadX   = 11
	cBubblePadY   = 8
	cBubbleRadius = 12
	cBubbleBottom = 30
	cArrow        = 6
	cFontPx       = 12
	cLineH        = 1.4

	cFrameInset  = 6
	cFrameRadius = 14
	cButton      = 22
	cButtonOver  = 12
	cIcon        = 11

	// Mascot.svelte CROSS_MS; Companion.svelte REACT/hop (.55s in mascot.css);
	// a blink is shut for about this long (ms-blink 95.5%→100% of 5.2s).
	crossMs = 240
	hopMs   = 550
	blinkMs = 140
	// Phases per loop and walk heading step: bake.ts PHASES, WALK_STEP.
	phases   = 4
	walkStep = 30
)

// CompanionTheme is the bubble's and the frame's colours, read by the window
// off its own stylesheet so the desktop matches the app's theme.
type CompanionTheme struct {
	Bg     string `json:"bg"`
	Fg     string `json:"fg"`
	Muted  string `json:"muted"`
	Border string `json:"border"`
	Accent string `json:"accent"`
}

// companionScene is everything one frame is drawn from.
type companionScene struct {
	Pose  string
	Theme CompanionTheme
	// The bubble: the text as shown so far, its words for wrapping (the
	// window segments them; Go cannot break Thai), and whether a cursor
	// blinks after it.
	Shown  string
	Words  []string
	Cursor bool
	// Local to the body.
	Hover   bool
	Muted   bool
	Flip    bool // bubble on the right: no room on the left
	Walking bool
	Heading float64
	HopAt   time.Time
	Shut    bool
}

// spriteSource is where frames come from (spriteStore).
type spriteSource interface {
	frame(key string) *image.RGBA
}

// noSprites is a source with nothing in it — what the body draws from until
// the window has named a set. A nil interface here once took the whole
// process down on the body's first frame (13 ก.ย. 2026).
type noSprites struct{}

func (noSprites) frame(string) *image.RGBA { return nil }

// textPainter draws and measures text at a pixel size.
type textPainter interface {
	// width of s in pixels at fontPx.
	measure(s string, fontPx int) int
	lineHeight(fontPx int) int
	// paint draws lines top-left on an opaque bg, returning an image of
	// exactly w×h.
	paint(lines []string, fontPx, w, h int, bg, fg color.RGBA) *image.RGBA
}

// composer draws frames and remembers what it takes to crossfade and loop:
// when the pose began and what it was before.
type composer struct {
	sprites spriteSource
	text    textPainter
	scale   float64

	pose     string
	poseAt   time.Time
	prevPose string
	prevAt   time.Time
	crossAt  time.Time
	walkAt   time.Time
	// Scratch for blends: the pose's two phases, the old pose's two during a
	// crossfade, and the crossfade itself — three, so none overwrites another.
	blend      *image.RGBA
	blendUnder *image.RGBA
	blendCross *image.RGBA
	// The part of the canvas the last frame touched: what the window shows,
	// and what the next frame has to clear.
	used image.Rectangle
	// The bubble's text, painted once per text and theme rather than per
	// frame — the one costly thing in a frame is the GDI round trip.
	textKey   string
	textLines []string
	textW     int
	textImg   *image.RGBA
	// Frames resampled to this scale, for the moment between crossing to a
	// monitor of another scale and the window baking a set for it: a frame
	// drawn at the wrong size was cut off at the edge of its box, and a
	// frame drawn a little blurry for a second is not (owner, 13 ก.ย. 2026:
	// "โยกไปจอต่างขนาดมันพัง").
	scaled map[string]*image.RGBA
}

// rescale moves the composer to another monitor's scale without losing its
// clock: the pose, its phase and a crossfade in flight carry on, so a drag
// across the seam between two monitors does not restart the walk (owner:
// "โยกไปโยกมามันกระตุก ๆ").
func (c *composer) rescale(scale float64) {
	if scale <= 0 || scale == c.scale {
		return
	}
	c.scale = scale
	c.textKey = ""
	c.textImg = nil
	c.blend, c.blendUnder, c.blendCross = nil, nil, nil
	c.scaled = nil
	c.used = image.Rectangle{}
}

// fit is a frame at the size the sprite box has at this scale: the frame
// itself when it was baked for this scale, a resampled copy otherwise.
func (c *composer) fit(key string, pic *image.RGBA) *image.RGBA {
	if pic == nil {
		return nil
	}
	want := c.spriteRect().Dx()
	if strings.HasPrefix(key, "icon-") {
		want = c.px(cIcon)
	}
	if pic.Bounds().Dx() == want {
		return pic
	}
	if c.scaled == nil {
		c.scaled = map[string]*image.RGBA{}
	}
	if s, ok := c.scaled[key]; ok && s.Bounds().Dx() == want {
		return s
	}
	if len(c.scaled) > 32 {
		c.scaled = map[string]*image.RGBA{}
	}
	s := resample(pic, want, want)
	c.scaled[key] = s
	return s
}

// resample scales a premultiplied RGBA picture to w×h, bilinear.
func resample(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == 0 || sh == 0 || w == 0 || h == 0 {
		return dst
	}
	for y := 0; y < h; y++ {
		fy := (float64(y)+0.5)*float64(sh)/float64(h) - 0.5
		y0 := int(math.Floor(fy))
		ty := fy - float64(y0)
		y1 := y0 + 1
		if y0 < 0 {
			y0, ty = 0, 0
		}
		if y1 >= sh {
			y1 = sh - 1
		}
		for x := 0; x < w; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(w) - 0.5
			x0 := int(math.Floor(fx))
			tx := fx - float64(x0)
			x1 := x0 + 1
			if x0 < 0 {
				x0, tx = 0, 0
			}
			if x1 >= sw {
				x1 = sw - 1
			}
			o := dst.PixOffset(x, y)
			a, b := src.PixOffset(x0, y0), src.PixOffset(x1, y0)
			cc, d := src.PixOffset(x0, y1), src.PixOffset(x1, y1)
			for k := 0; k < 4; k++ {
				top := float64(src.Pix[a+k])*(1-tx) + float64(src.Pix[b+k])*tx
				bot := float64(src.Pix[cc+k])*(1-tx) + float64(src.Pix[d+k])*tx
				dst.Pix[o+k] = uint8(top*(1-ty) + bot*ty + 0.5)
			}
		}
	}
	return dst
}

func newComposer(sprites spriteSource, text textPainter, scale float64) *composer {
	if scale <= 0 {
		scale = 1
	}
	if sprites == nil {
		sprites = noSprites{}
	}
	return &composer{sprites: sprites, text: text, scale: scale}
}

// setSprites swaps the source; nil means none.
func (c *composer) setSprites(s spriteSource) {
	if s == nil {
		s = noSprites{}
	}
	c.sprites = s
}

func (c *composer) px(logical float64) int { return int(math.Round(logical * c.scale)) }

// canvasSize is the window's size at this scale.
func (c *composer) canvasSize() (int, int) { return c.px(cCanvasW), c.px(cCanvasH) }

// figureRect is where the 104-logical figure sits on the canvas — what the
// clamp keeps on a monitor and what the pointer is "near".
func (c *composer) figureRect() image.Rectangle {
	x := c.px((cCanvasW - cFigure) / 2)
	y := c.px(cFigTop)
	return image.Rect(x, y, x+c.px(cFigure), y+c.px(cFigure))
}

// spriteRect is where the baked 130-logical picture goes.
func (c *composer) spriteRect() image.Rectangle {
	f := c.figureRect()
	p := c.px(cFigPad)
	return image.Rect(f.Min.X-p, f.Min.Y-p, f.Min.X-p+c.px(cSprite), f.Min.Y-p+c.px(cSprite))
}

// buttonRects are the hide (right) and mute (left) circles, drawn on hover.
func (c *composer) buttonRects() (hide, mute image.Rectangle) {
	f := c.figureRect()
	b, over := c.px(cButton), c.px(cButtonOver)
	hide = image.Rect(f.Max.X+over-b, f.Min.Y-over, f.Max.X+over, f.Min.Y-over+b)
	mute = image.Rect(f.Min.X-over, f.Min.Y-over, f.Min.X-over+b, f.Min.Y-over+b)
	return
}

// ---- the frame --------------------------------------------------------------

// draw paints one frame of the scene at `now` onto dst and returns the part
// of the canvas it used — the sprite, the bubble, the frame — which is all
// the window needs to show. Only what the previous frame used is cleared.
func (c *composer) draw(dst *image.RGBA, s companionScene, now time.Time) image.Rectangle {
	clearRect(dst, c.used)
	used := image.Rectangle{}
	if s.Pose != c.pose {
		c.prevPose, c.prevAt = c.pose, c.poseAt
		c.pose, c.poseAt = s.Pose, now
		c.crossAt = now
	}
	if s.Walking && c.walkAt.IsZero() {
		c.walkAt = now
	} else if !s.Walking {
		c.walkAt = time.Time{}
	}

	// the figure
	pic := c.picture(s, now, c.pose, c.poseAt, &c.blend)
	if age := now.Sub(c.crossAt); c.prevPose != "" && age < crossMs*time.Millisecond && !s.Walking {
		// The old drawing fades over the new one (Mascot.svelte {#key}).
		under := c.picture(s, now, c.prevPose, c.prevAt, &c.blendUnder)
		if under != nil && pic != nil {
			c.blendCross = lerpRGBA(c.blendCross, pic, under, 1-float64(age)/float64(crossMs*time.Millisecond))
			pic = c.blendCross
		} else if pic == nil {
			pic = under
		}
	}
	if pic != nil {
		r := c.spriteRect()
		r = r.Add(image.Pt(0, c.hopOffset(s.HopAt, now)))
		draw.Draw(dst, r, pic, pic.Bounds().Min, draw.Over)
		used = used.Union(r)
	}

	// the bubble
	if s.Shown != "" {
		used = used.Union(c.bubble(dst, s, now))
	}

	// the frame and its buttons
	if s.Hover || s.Muted {
		used = used.Union(c.frame(dst, s))
	}
	c.used = used.Intersect(dst.Bounds())
	return c.used
}

// clearRect zeroes a rectangle of the canvas.
func clearRect(dst *image.RGBA, r image.Rectangle) {
	r = r.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i := dst.PixOffset(r.Min.X, y)
		clear(dst.Pix[i : i+r.Dx()*4])
	}
}

// picture is the sprite for a pose at `now`: the two phases either side of
// the moment, blended — so four baked pictures read as a loop. The walk is
// keyed by heading and timed from the drag; the blink swaps the shut frame
// in; a frame not yet baked falls back to the nearest that is.
func (c *composer) picture(s companionScene, now time.Time, pose string, since time.Time, buf **image.RGBA) *image.RGBA {
	if s.Walking {
		h := walkHeadingKey(s.Heading, walkStep)
		i, f := c.phaseAt("walk", now.Sub(c.walkAt))
		ka, kb := walkKey(h, i), walkKey(h, (i+1)%phases)
		a := c.fit(ka, c.sprites.frame(ka))
		b := c.fit(kb, c.sprites.frame(kb))
		return pair(buf, a, b, f)
	}
	if s.Shut {
		if k := poseKey(pose, 0, "shut"); c.sprites.frame(k) != nil {
			return c.fit(k, c.sprites.frame(k))
		}
	}
	i, f := c.phaseAt(pose, now.Sub(since))
	ka, kb := poseKey(pose, i, "open"), poseKey(pose, (i+1)%phases, "open")
	a := c.fit(ka, c.sprites.frame(ka))
	b := c.fit(kb, c.sprites.frame(kb))
	if a == nil && b == nil {
		k := poseKey("idle", 0, "open")
		return c.fit(k, c.sprites.frame(k))
	}
	return pair(buf, a, b, f)
}

// pair blends two phases into *buf, or returns the one there is.
func pair(buf **image.RGBA, a, b *image.RGBA, f float64) *image.RGBA {
	switch {
	case a == nil:
		return b
	case b == nil || f <= 0.001:
		return a
	case f >= 0.999:
		return b
	}
	*buf = lerpRGBA(*buf, a, b, f)
	return *buf
}

// phaseAt is which baked phase a moment of a pose falls in and how far
// towards the next: the loop's period is bake.ts loopOf's. A one-shot pose
// (wake, startled) holds its last phase.
func (c *composer) phaseAt(pose string, elapsed time.Duration) (int, float64) {
	period, oneShot := loopPeriod(pose)
	if period <= 0 {
		return 0, 0
	}
	t := elapsed.Seconds()
	if oneShot && t >= period {
		return phases - 1, 0
	}
	u := math.Mod(t, period) / period * phases
	i := int(u) % phases
	return i, u - math.Floor(u)
}

// loopPeriod mirrors bake.ts loopOf: seconds per loop, and whether the pose
// plays once rather than looping.
func loopPeriod(pose string) (float64, bool) {
	switch pose {
	case "walk":
		return 0.62, false
	case "greeting":
		return 1.4, false
	case "typing", "coding", "searchData":
		return 0.36, false
	case "recharge":
		return 3.2, false
	case "wake":
		return 1.2, true
	case "startled":
		return 0.5, true
	}
	return 3.6, false
}

func poseKey(pose string, phase int, blink string) string {
	return pose + "-p" + strconv.Itoa(phase) + "-" + blink
}

func walkKey(heading, phase int) string {
	return "walk-t" + strconv.Itoa(heading) + "-p" + strconv.Itoa(phase)
}

// hopOffset is the vertical lift of a click's hop at `now`, in device pixels
// (mascot.css ms-hop: up 4px by 30%, back by 60%).
func (c *composer) hopOffset(at, now time.Time) int {
	if at.IsZero() {
		return 0
	}
	u := float64(now.Sub(at)) / float64(hopMs*time.Millisecond)
	if u < 0 || u >= 0.6 {
		return 0
	}
	var lift float64
	if u < 0.3 {
		lift = u / 0.3
	} else {
		lift = 1 - (u-0.3)/0.3
	}
	return -c.px(4 * lift)
}

// ---- the bubble ---------------------------------------------------------------

// bubble draws the report card beside the figure: the wrapped text on an
// opaque rounded card with a 1px border and a small arrow towards the
// figure, and a blinking bar after the text while it is still arriving.
func (c *composer) bubble(dst *image.RGBA, s companionScene, now time.Time) image.Rectangle {
	fontPx := c.px(cFontPx)
	lineH := c.text.lineHeight(fontPx)
	if lineH <= 0 {
		lineH = c.px(cFontPx * cLineH)
	}
	padX, padY := c.px(cBubblePadX), c.px(cBubblePadY)
	maxText := c.px(cBubbleMaxW) - 2*padX
	bg := parseColor(s.Theme.Bg, color.RGBA{0x1f, 0x1f, 0x23, 0xff})
	border := parseColor(s.Theme.Border, color.RGBA{0x22, 0x22, 0x26, 0xff})
	fg := parseColor(s.Theme.Fg, color.RGBA{0xf2, 0xf2, 0xf3, 0xff})

	// Wrapped and painted once per text: the key is everything the picture
	// depends on.
	key := s.Shown + "|" + s.Theme.Bg + "|" + s.Theme.Fg + "|" + strconv.Itoa(fontPx)
	if key != c.textKey {
		lines, textW := wrapWords(s.Words, s.Shown, maxText, func(t string) int { return c.text.measure(t, fontPx) })
		c.textKey, c.textLines, c.textW = key, lines, textW
		c.textImg = nil
		if len(lines) > 0 && textW > 0 {
			// Painted opaque on the card's colour: a layered window and
			// ClearType do not mix, so the glyphs go where alpha is 255.
			c.textImg = c.text.paint(lines, fontPx, textW, len(lines)*lineH, bg, fg)
		}
	}
	lines, textW := c.textLines, c.textW
	if s.Cursor {
		textW += c.px(2)
	}
	w := textW + 2*padX
	h := len(lines)*lineH + 2*padY
	fig := c.figureRect()
	var x int
	if s.Flip {
		x = fig.Max.X + c.px(cBubbleGap)
	} else {
		x = fig.Min.X - c.px(cBubbleGap) - w
	}
	y := fig.Max.Y - c.px(cBubbleBottom) - h
	card := image.Rect(x, y, x+w, y+h)

	fillRoundRect(dst, card, float64(c.px(cBubbleRadius)), bg, border, float64(c.scale))
	// the arrow: a small diamond on the edge facing the figure, its base
	// covered by the card so only the point shows
	arrow := c.px(cArrow)
	ay := card.Max.Y - c.px(12) - arrow
	if s.Flip {
		fillDiamond(dst, card.Min.X, ay, arrow, bg, border, float64(c.scale))
	} else {
		fillDiamond(dst, card.Max.X, ay, arrow, bg, border, float64(c.scale))
	}
	if c.textImg != nil {
		draw.Draw(dst, image.Rect(x+padX, y+padY, x+padX+c.textW, y+padY+len(lines)*lineH), c.textImg, image.Point{}, draw.Src)
	}
	if s.Cursor && len(lines) > 0 && (now.UnixMilli()/500)%2 == 0 {
		last := lines[len(lines)-1]
		cx := x + padX + c.text.measure(last, fontPx) + c.px(1)
		cy := y + padY + (len(lines)-1)*lineH
		fillRect(dst, image.Rect(cx, cy+c.px(2), cx+c.px(1), cy+lineH-c.px(2)), parseColor(s.Theme.Accent, color.RGBA{0x35, 0xd0, 0xe0, 0xff}))
	}
	return card.Inset(-arrow - 1)
}

// wrapWords lays words out greedily within maxW, measuring with `width`.
// The words are the window's segments of `shown` (Intl.Segmenter) and are
// used only if they add up to it; otherwise the text is broken at spaces,
// and failing that anywhere, so nothing is ever wider than the card.
func wrapWords(words []string, shown string, maxW int, width func(string) int) ([]string, int) {
	if strings.Join(words, "") != shown {
		words = strings.SplitAfter(shown, " ")
	}
	var lines []string
	line := ""
	flush := func() {
		lines = append(lines, strings.TrimRight(line, " "))
		line = ""
	}
	for _, w := range words {
		if w == "" {
			continue
		}
		if strings.Contains(w, "\n") {
			for i, part := range strings.Split(w, "\n") {
				if i > 0 {
					flush()
				}
				line += part
			}
			continue
		}
		if line != "" && width(line+strings.TrimRight(w, " ")) > maxW {
			flush()
			if strings.TrimSpace(w) == "" {
				continue
			}
		}
		// a single word wider than the card is cut where it must be
		for width(strings.TrimRight(line+w, " ")) > maxW && len(w) > 1 {
			cut := len(w) - 1
			for cut > 0 && width(line+w[:cut]) > maxW {
				cut--
			}
			if cut <= 0 {
				break
			}
			for cut > 0 && cut < len(w) && !isRuneStart(w[cut]) {
				cut--
			}
			line += w[:cut]
			flush()
			w = w[cut:]
		}
		line += w
	}
	if line != "" || len(lines) == 0 {
		flush()
	}
	widest := 0
	for _, l := range lines {
		if wl := width(l); wl > widest {
			widest = wl
		}
	}
	return lines, widest
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }

// ---- the frame ----------------------------------------------------------------

// frame draws the dashed border around the figure and the two buttons: ×
// (hide) top-right, the speaker (voice) top-left. Muted, the speaker button
// stays visible without the rest — a figure that will not talk should say so.
func (c *composer) frame(dst *image.RGBA, s companionScene) image.Rectangle {
	border := parseColor(s.Theme.Border, color.RGBA{0x22, 0x22, 0x26, 0xff})
	bg := parseColor(s.Theme.Bg, color.RGBA{0x1f, 0x1f, 0x23, 0xff})
	muted := parseColor(s.Theme.Muted, color.RGBA{0x7d, 0x80, 0x87, 0xff})
	hide, mute := c.buttonRects()
	used := mute.Inset(-1)
	if s.Hover {
		fig := c.figureRect().Inset(-c.px(cFrameInset))
		dashedRoundRect(dst, fig, float64(c.px(cFrameRadius)), border, float64(c.scale))
		c.button(dst, hide, "icon-x", bg, border, muted)
		used = used.Union(fig.Inset(-2)).Union(hide.Inset(-1))
	}
	icon := "icon-volume2"
	if s.Muted {
		icon = "icon-volumeX"
	}
	c.button(dst, mute, icon, bg, border, muted)
	return used
}

func (c *composer) button(dst *image.RGBA, r image.Rectangle, icon string, bg, border, ink color.RGBA) {
	fillRoundRect(dst, r, float64(r.Dx())/2, bg, border, float64(c.scale))
	if pic := c.fit(icon, c.sprites.frame(icon)); pic != nil {
		tinted := tint(pic, ink)
		at := image.Pt(r.Min.X+(r.Dx()-pic.Bounds().Dx())/2, r.Min.Y+(r.Dy()-pic.Bounds().Dy())/2)
		draw.Draw(dst, image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}, tinted, image.Point{}, draw.Over)
	}
}

// ---- pixels -------------------------------------------------------------------

// lerpRGBA blends premultiplied a towards b by f into buf (reused when the
// size fits).
func lerpRGBA(buf, a, b *image.RGBA, f float64) *image.RGBA {
	if buf == nil || buf.Bounds().Size() != a.Bounds().Size() {
		buf = image.NewRGBA(image.Rectangle{Max: a.Bounds().Size()})
	}
	if b == nil || b.Bounds().Size() != a.Bounds().Size() {
		copy(buf.Pix, a.Pix)
		return buf
	}
	fa := uint32(math.Round(f * 256))
	fb := 256 - fa
	for i := range buf.Pix {
		buf.Pix[i] = uint8((uint32(a.Pix[i])*fb + uint32(b.Pix[i])*fa) >> 8)
	}
	return buf
}

// tint colours a white-on-transparent picture: the alpha is the shape.
func tint(pic *image.RGBA, ink color.RGBA) *image.RGBA {
	out := image.NewRGBA(image.Rectangle{Max: pic.Bounds().Size()})
	for i := 0; i+3 < len(pic.Pix); i += 4 {
		a := uint32(pic.Pix[i+3])
		out.Pix[i] = uint8(uint32(ink.R) * a / 255)
		out.Pix[i+1] = uint8(uint32(ink.G) * a / 255)
		out.Pix[i+2] = uint8(uint32(ink.B) * a / 255)
		out.Pix[i+3] = uint8(a)
	}
	return out
}

func fillRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	draw.Draw(dst, r.Intersect(dst.Bounds()), image.NewUniform(c), image.Point{}, draw.Over)
}

// fillRoundRect paints an anti-aliased rounded rectangle with a 1-logical-px
// border, by signed distance: inside the border is fill, within a pixel of
// the edge is border, and the last pixel fades out.
func fillRoundRect(dst *image.RGBA, r image.Rectangle, radius float64, fill, border color.RGBA, scale float64) {
	shape := func(x, y float64) float64 { return roundRectDist(x, y, r, radius) }
	paintShape(dst, r.Inset(-1), shape, fill, border, scale)
}

func fillDiamond(dst *image.RGBA, cx, cy, half int, fill, border color.RGBA, scale float64) {
	r := image.Rect(cx-half, cy-half, cx+half, cy+half)
	fc := float64(cx)
	shape := func(x, y float64) float64 {
		// a square rotated 45°: L1 distance from the centre, minus half
		return (math.Abs(x-fc) + math.Abs(y-float64(cy))) - float64(half)
	}
	paintShape(dst, r.Inset(-1), shape, fill, border, scale)
}

func dashedRoundRect(dst *image.RGBA, r image.Rectangle, radius float64, border color.RGBA, scale float64) {
	dash := 4 * scale
	shape := func(x, y float64) float64 {
		d := math.Abs(roundRectDist(x, y, r, radius)+0.5*scale) - 0.5*scale
		// dashes along the perimeter, approximated by position: on for
		// dash, off for dash, using whichever axis runs along this edge
		if math.Mod(math.Floor((x+y)/dash), 2) == 1 {
			return 1
		}
		return d
	}
	paintShape(dst, r.Inset(-1), shape, color.RGBA{}, border, scale)
}

// roundRectDist is the signed distance from (x, y) to the rounded rect's
// edge; negative inside.
func roundRectDist(x, y float64, r image.Rectangle, radius float64) float64 {
	cx := (float64(r.Min.X) + float64(r.Max.X)) / 2
	cy := (float64(r.Min.Y) + float64(r.Max.Y)) / 2
	hw := float64(r.Dx())/2 - radius
	hh := float64(r.Dy())/2 - radius
	qx := math.Abs(x-cx) - hw
	qy := math.Abs(y-cy) - hh
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - radius
}

// paintShape composites a shape given by a signed distance: fill inside,
// border within one logical pixel of the edge, anti-aliased at the edge.
func paintShape(dst *image.RGBA, area image.Rectangle, dist func(x, y float64) float64, fill, border color.RGBA, scale float64) {
	area = area.Intersect(dst.Bounds())
	bw := 1 * scale
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			d := dist(float64(x)+0.5, float64(y)+0.5)
			if d > 0.5 {
				continue
			}
			cover := math.Min(1, 0.5-d)
			c := fill
			if d > -bw {
				c = border
			}
			blendAt(dst, x, y, c, cover)
		}
	}
}

// blendAt composites c at coverage a onto the premultiplied pixel.
func blendAt(dst *image.RGBA, x, y int, c color.RGBA, a float64) {
	if a <= 0 || c.A == 0 {
		return
	}
	i := dst.PixOffset(x, y)
	sa := a * float64(c.A) / 255
	sr, sg, sb := float64(c.R)*sa, float64(c.G)*sa, float64(c.B)*sa
	keep := 1 - sa
	dst.Pix[i] = uint8(math.Min(255, sr+float64(dst.Pix[i])*keep))
	dst.Pix[i+1] = uint8(math.Min(255, sg+float64(dst.Pix[i+1])*keep))
	dst.Pix[i+2] = uint8(math.Min(255, sb+float64(dst.Pix[i+2])*keep))
	dst.Pix[i+3] = uint8(math.Min(255, sa*255+float64(dst.Pix[i+3])*keep))
}

// parseColor reads what getComputedStyle hands over — "rgb(r, g, b)",
// "rgba(r, g, b, a)" or "#rrggbb" — and falls back to `def`.
func parseColor(s string, def color.RGBA) color.RGBA {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 || len(hex) == 8 {
			v, err := strconv.ParseUint(hex, 16, 32)
			if err == nil {
				if len(hex) == 6 {
					return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
				}
				return color.RGBA{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8), uint8(v)}
			}
		}
		return def
	}
	if i := strings.Index(s, "("); i > 0 && strings.HasSuffix(s, ")") {
		parts := strings.FieldsFunc(s[i+1:len(s)-1], func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
		if len(parts) >= 3 {
			var ch [4]float64
			ch[3] = 1
			for k := 0; k < len(parts) && k < 4; k++ {
				v, err := strconv.ParseFloat(strings.TrimSuffix(parts[k], "%"), 64)
				if err != nil {
					return def
				}
				if k == 3 && strings.HasSuffix(parts[k], "%") {
					v /= 100
				}
				ch[k] = v
			}
			return color.RGBA{uint8(ch[0]), uint8(ch[1]), uint8(ch[2]), uint8(math.Round(ch[3] * 255))}
		}
	}
	return def
}
