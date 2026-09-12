package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func spritePNG(t *testing.T, c color.RGBA) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// A batch is kept on disk under its key, listed back, and decoded on demand
// as premultiplied RGBA; a second store on the same directory finds it
// without being handed anything.
func TestCompanionSpritesKeepFramesAcrossStores(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "abcdef01-175")
	s := newSpriteStore("abcdef01-175", dir)
	err := s.put([]CompanionFrame{
		{Key: "idle-p0-open", W: 4, H: 4, PNG: spritePNG(t, color.RGBA{R: 255, A: 128})},
		{Key: "walk-t90-p2", W: 4, H: 4, PNG: spritePNG(t, color.RGBA{B: 255, A: 255})},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "walk-t90-p2.png")); err != nil {
		t.Fatalf("frame not on disk: %v", err)
	}
	again := newSpriteStore("abcdef01-175", dir)
	if got := again.have(); len(got) != 2 || got[0] != "idle-p0-open" || got[1] != "walk-t90-p2" {
		t.Fatalf("keys after reopen: %v", got)
	}
	img := again.frame("idle-p0-open")
	if img == nil {
		t.Fatal("frame not decodable from disk")
	}
	if px := img.RGBAAt(1, 1); px.A != 128 || px.R < 126 || px.R > 129 {
		t.Fatalf("not premultiplied RGBA: %+v", px)
	}
	if again.frame("nothing-p0-open") != nil {
		t.Fatal("an unknown key drew something")
	}
}

// A bad key or a bad picture is refused by name; the rest of the batch stays.
func TestCompanionSpritesRefuseWhatCannotBeDrawn(t *testing.T) {
	s := newSpriteStore("abcdef01-100", "")
	err := s.put([]CompanionFrame{
		{Key: "../escape", PNG: spritePNG(t, color.RGBA{A: 255})},
		{Key: "idle-p1-open", PNG: "bm90IGEgcG5n"},
		{Key: "idle-p2-open", PNG: spritePNG(t, color.RGBA{A: 255})},
	})
	if err == nil {
		t.Fatal("bad frames were accepted")
	}
	if got := s.have(); len(got) != 1 || got[0] != "idle-p2-open" {
		t.Fatalf("kept: %v", got)
	}
}

// Decoded frames are bounded: past the limit the least recently drawn goes,
// and comes back from disk when asked for again.
func TestCompanionSpritesCacheIsBounded(t *testing.T) {
	dir := t.TempDir()
	s := newSpriteStore("abcdef01-100", dir)
	s.limit = 3
	var frames []CompanionFrame
	for _, k := range []string{"a-p0-open", "b-p0-open", "c-p0-open", "d-p0-open"} {
		frames = append(frames, CompanionFrame{Key: k, PNG: spritePNG(t, color.RGBA{A: 255})})
	}
	if err := s.put(frames); err != nil {
		t.Fatal(err)
	}
	if n := s.cached(); n != 3 {
		t.Fatalf("cached %d, want 3", n)
	}
	if s.frame("a-p0-open") == nil {
		t.Fatal("evicted frame did not come back from disk")
	}
	if n := s.cached(); n != 3 {
		t.Fatalf("cached %d after reload, want 3", n)
	}
}

// The bindings name a set by hash and refuse a hash that is not one, and a
// new hash starts a new store.
func TestCompanionSpriteBindingsFollowTheHash(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := newTestApp(t)
	if err := a.CompanionSprites("not a hash", nil); err == nil {
		t.Fatal("bad hash accepted")
	}
	if got := a.CompanionSpriteKeys("not a hash"); len(got) != 0 {
		t.Fatalf("bad hash listed %v", got)
	}
	if err := a.CompanionSprites("0123abcd-175", []CompanionFrame{{Key: "idle-p0-open", PNG: spritePNG(t, color.RGBA{A: 255})}}); err != nil {
		t.Fatal(err)
	}
	if got := a.CompanionSpriteKeys("0123abcd-175"); len(got) != 1 {
		t.Fatalf("keys: %v", got)
	}
	if got := a.CompanionSpriteKeys("0123abcd-100"); len(got) != 0 {
		t.Fatalf("another scale shares frames: %v", got)
	}
}

// Every scale's set stays resident, and a monitor whose set is still baking
// draws from the same look's other set (resampled by the composer) rather
// than from nothing — crossing monitors is not a reload.
func TestCompanionSpritesKeepEveryScaleAndFallBackAcrossThem(t *testing.T) {
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	a := &App{}
	c := a.companion()
	if err := a.CompanionSprites("0123abcd-175", []CompanionFrame{{Key: "idle-p0-open", PNG: spritePNG(t, color.RGBA{A: 255})}}); err != nil {
		t.Fatal(err)
	}
	// the window names the 100% set (empty so far) as the figure crosses
	a.CompanionSpriteKeys("0123abcd-100")
	if c.spritesAt(1).frame("idle-p0-open") == nil {
		t.Fatal("at 100% with nothing baked yet, the 175% frame should stand in")
	}
	if c.spritesAt(1.75).frame("idle-p0-open") == nil {
		t.Fatal("the 175% set was dropped when the 100% set was named")
	}
	if err := a.CompanionSprites("0123abcd-100", []CompanionFrame{{Key: "idle-p0-open", PNG: spritePNG(t, color.RGBA{R: 255, A: 255})}}); err != nil {
		t.Fatal(err)
	}
	if px := c.spritesAt(1).frame("idle-p0-open").RGBAAt(0, 0); px.R != 255 {
		t.Fatalf("once baked, the 100%% set's own frame is used: %+v", px)
	}
	// another look replaces the current one; the old look's sets are not fallbacks
	a.CompanionSpriteKeys("ffffffff-175")
	if c.spritesAt(1.75).frame("idle-p0-open") != nil {
		t.Fatal("a set of another look was used as a fallback")
	}
}
