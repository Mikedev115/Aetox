package assetlib

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeProbe answers from the file name so a test never needs ffprobe: a name
// containing "long" runs 90s, "alpha" carries transparency, "broken" fails.
func fakeProbe(_ context.Context, path string) (Probe, error) {
	name := strings.ToLower(filepath.Base(path))
	if strings.Contains(name, "broken") {
		return Probe{}, os.ErrInvalid
	}
	p := Probe{Duration: 2, Width: 1920, Height: 1080}
	if strings.Contains(name, "long") {
		p.Duration = 90
	}
	if strings.Contains(name, "alpha") {
		p.HasAlpha = true
	}
	return p, nil
}

func shelf(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := []string{
		"71 SFX PACK/whoosh-01.wav",
		"71 SFX PACK/Cash_Register.wav",
		"MONEY SFX/cash-long-loop.wav",
		"overlays/film-burn-alpha.mov",
		"GREEN SCREEN EFFECTS/explosion.mp4",
		"MOTION BACKGROUNDS/grid-blue.mp4",
		"101 VIRAL MEMES/this-is-fine.mp4",
		"Animated Icons/rocket.png",
		"Human still images for/person-01.jpg",
		"README.txt",
		".hidden/secret.wav",
		"ASSETS/broken.mp4",
	}
	for _, f := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestScanClassifiesByKindAndSkipsWhatIsNotMaterial(t *testing.T) {
	root := shelf(t)
	var last Progress
	lib, err := Scan(context.Background(), root, ProbeFunc(fakeProbe), func(p Progress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	// README.txt and .hidden/ are not on the shelf; the broken file is, and is
	// counted as unread rather than dropped.
	if len(lib.Assets) != 10 {
		t.Fatalf("assets = %d, want 10: %+v", len(lib.Assets), paths(lib.Assets))
	}
	if lib.Unread != 1 {
		t.Errorf("unread = %d, want 1", lib.Unread)
	}
	if last.Done != last.Total || last.Total != 10 {
		t.Errorf("progress ended at %+v", last)
	}
	want := map[string]Kind{
		"71 SFX PACK/whoosh-01.wav":            KindSFX,
		"MONEY SFX/cash-long-loop.wav":         KindMusic,
		"overlays/film-burn-alpha.mov":         KindOverlay,
		"GREEN SCREEN EFFECTS/explosion.mp4":   KindOverlay,
		"MOTION BACKGROUNDS/grid-blue.mp4":     KindBackground,
		"101 VIRAL MEMES/this-is-fine.mp4":     KindClip,
		"Animated Icons/rocket.png":            KindIcon,
		"Human still images for/person-01.jpg": KindImage,
	}
	for _, a := range lib.Assets {
		if k, ok := want[a.Path]; ok && a.Kind != k {
			t.Errorf("%s: kind = %s, want %s", a.Path, a.Kind, k)
		}
		if a.Category == "" {
			t.Errorf("%s: no category", a.Path)
		}
	}
}

func TestSearchRanksTheFileNamedForTheThingFirst(t *testing.T) {
	root := shelf(t)
	lib, err := Scan(context.Background(), root, ProbeFunc(fakeProbe), nil)
	if err != nil {
		t.Fatal(err)
	}
	hits, total := Search([]*Library{lib}, Query{Text: "cash"})
	if total != 2 || len(hits) != 2 {
		t.Fatalf("cash: %d hits (%d total): %v", len(hits), total, paths(hits))
	}
	if hits[0].Path != "71 SFX PACK/Cash_Register.wav" {
		t.Errorf("first hit = %s, want the short SFX name", hits[0].Path)
	}

	// Kind and length filter together: the 90s loop is music, not an effect.
	hits, _ = Search([]*Library{lib}, Query{Text: "cash", Kind: KindSFX, MaxSeconds: 5})
	if len(hits) != 1 || hits[0].Kind != KindSFX {
		t.Errorf("sfx cash: %v", paths(hits))
	}

	// NeedAlpha is stricter than KindOverlay: green-screen footage is an
	// overlay by name but has no alpha channel.
	hits, _ = Search([]*Library{lib}, Query{Kind: KindOverlay})
	if len(hits) != 2 {
		t.Errorf("overlays: %v", paths(hits))
	}
	hits, _ = Search([]*Library{lib}, Query{NeedAlpha: true})
	if len(hits) != 1 || !strings.Contains(hits[0].Path, "alpha") {
		t.Errorf("alpha: %v", paths(hits))
	}

	// Every word must match: a term that hits nothing empties the result.
	if hits, _ = Search([]*Library{lib}, Query{Text: "cash unicorn"}); len(hits) != 0 {
		t.Errorf("cash unicorn: %v", paths(hits))
	}

	// Prefix match: "whoo" finds "whoosh".
	if hits, _ = Search([]*Library{lib}, Query{Text: "whoo"}); len(hits) != 1 {
		t.Errorf("whoo: %v", paths(hits))
	}
}

func TestIDsSurviveMovingTheLibrary(t *testing.T) {
	a := classify("71 SFX PACK/whoosh-01.wav", 1, Probe{})
	b := classify("71 sfx pack/WHOOSH-01.wav", 1, Probe{})
	if a.ID != b.ID {
		t.Errorf("id changed with case: %s vs %s", a.ID, b.ID)
	}
	if libraryID(`D:\Assets`) != libraryID(`d:/assets/`) {
		t.Error("library id should not depend on case or slashes")
	}
}

func TestStoreRoundTripsAndForgetsWithoutDeleting(t *testing.T) {
	root := shelf(t)
	lib, err := Scan(context.Background(), root, ProbeFunc(fakeProbe), nil)
	if err != nil {
		t.Fatal(err)
	}
	data := t.TempDir()
	s, err := Load(data)
	if err != nil || len(s.Libraries) != 0 {
		t.Fatalf("empty load: %v %d", err, len(s.Libraries))
	}
	s.Put(lib)
	s.Put(lib) // a rescan replaces, never duplicates
	if len(s.Libraries) != 1 {
		t.Fatalf("libraries = %d after two puts", len(s.Libraries))
	}
	if err := s.Save(data); err != nil {
		t.Fatal(err)
	}
	again, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := again.Get(lib.ID)
	if !ok || len(got.Assets) != len(lib.Assets) || got.Root != root {
		t.Fatalf("round trip lost the library: %+v", got)
	}
	_, a, ok := Find(again.Libraries, lib.Assets[0].ID)
	if !ok || a.Path != lib.Assets[0].Path {
		t.Errorf("find by id: %v %+v", ok, a)
	}
	if !again.Remove(lib.ID) || len(again.Libraries) != 0 {
		t.Error("remove")
	}
	// The user's files are untouched by forgetting the shelf.
	if _, err := os.Stat(filepath.Join(root, "71 SFX PACK", "whoosh-01.wav")); err != nil {
		t.Errorf("remove touched the user's files: %v", err)
	}
}

func TestPixFmtHasAlpha(t *testing.T) {
	for f, want := range map[string]bool{
		"yuv420p": false, "yuva420p": true, "rgba": true, "argb": true, "bgra": true,
		"rgb24": false, "pal8": false, "gbrap": true, "ya8": true, "yuv444p10le": false,
	} {
		if got := pixFmtHasAlpha(f); got != want {
			t.Errorf("%s: %v", f, got)
		}
	}
}

func paths(as []Asset) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.Path)
	}
	return out
}

func TestParseBannerReadsEveryInputOrNothing(t *testing.T) {
	banner := "[aist#0:0/pcm_s16le @ 0x1] Guessed Channel Layout: stereo\n" +
		"Input #0, wav, from 'a.wav':\n" +
		"  Duration: 00:00:03.40, bitrate: 1411 kb/s\n" +
		"  Stream #0:0: Audio: pcm_s16le ([1][0][0][0] / 0x0001), 44100 Hz, stereo, s16, 1411 kb/s\n" +
		"Input #1, gif, from 'b.gif':\n" +
		"  Duration: 00:00:05.03, start: 0.000000, bitrate: 797 kb/s\n" +
		"  Stream #1:0: Video: gif, bgra, 400x400, 25 fps, 33.33 tbr, 100 tbn\n" +
		"Input #2, mov,mp4,m4a,3gp,3g2,mj2, from 'c.mov':\n" +
		"  Metadata:\n" +
		"    major_brand     : qt  \n" +
		"  Duration: 00:01:02.50, start: 0.000000, bitrate: 45000 kb/s\n" +
		"  Stream #2:0[0x1](eng): Video: prores (ap4h / 0x68347061), yuva444p12le(bt709, progressive), 1920x1080, 44800 kb/s, SAR 1:1 DAR 16:9, 29.97 fps\n" +
		"  Stream #2:1[0x2](eng): Audio: pcm_s16le, 48000 Hz, stereo, s16, 1536 kb/s\n" +
		"At least one output file must be specified\n"
	got := parseBanner(banner, 3)
	if got == nil {
		t.Fatal("banner with all three inputs parsed as incomplete")
	}
	if got[0].Duration != 3.4 || got[0].Width != 0 {
		t.Errorf("wav: %+v", got[0])
	}
	if got[1].Duration != 5.03 || got[1].Width != 400 || got[1].Height != 400 || !got[1].HasAlpha {
		t.Errorf("gif: %+v", got[1])
	}
	if got[2].Duration != 62.5 || got[2].Width != 1920 || got[2].Height != 1080 || !got[2].HasAlpha {
		t.Errorf("mov: %+v", got[2])
	}
	// A run that aborted on input #1 shows only input #0: not an answer.
	if parseBanner(banner, 4) != nil {
		t.Error("banner short of an input should parse as nil")
	}
}

// batchProbe counts how many times it was asked, to prove Scan hands a
// BatchProber batches rather than files.
type batchProbe struct {
	calls int
}

func (b *batchProbe) Probe(ctx context.Context, path string) (Probe, error) {
	return fakeProbe(ctx, path)
}
func (b *batchProbe) ProbeMany(ctx context.Context, paths []string) ([]Probe, []error) {
	b.calls++
	ps := make([]Probe, len(paths))
	es := make([]error, len(paths))
	for i, p := range paths {
		ps[i], es[i] = fakeProbe(ctx, p)
	}
	return ps, es
}

func TestScanHandsABatchProberBatches(t *testing.T) {
	root := shelf(t)
	b := &batchProbe{}
	lib, err := Scan(context.Background(), root, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Assets) != 10 || lib.Unread != 1 {
		t.Fatalf("batched scan changed the answer: %d assets, %d unread", len(lib.Assets), lib.Unread)
	}
	if b.calls != 1 {
		t.Errorf("ten files should be one batch, got %d calls", b.calls)
	}
}

func TestBuiltinShelfUnpacksOnceAndIsNeverStored(t *testing.T) {
	data := t.TempDir()
	libs, err := Builtin(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(libs) != 1 || !libs[0].Builtin || libs[0].ID != "builtin-interface-sounds" {
		t.Fatalf("builtin shelves: %+v", libs)
	}
	lib := libs[0]
	if len(lib.Assets) != 100 || lib.Name() != "Kenney Interface Sounds" || lib.License != "CC0" {
		t.Errorf("shelf = %d assets, name %q, licence %q", len(lib.Assets), lib.Name(), lib.License)
	}
	// Every catalogued file is really on disk, and the licence travelled.
	for _, a := range lib.Assets[:3] {
		if _, err := os.Stat(lib.FullPath(a)); err != nil {
			t.Errorf("%s not unpacked: %v", a.Path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(lib.Root, "License.txt")); err != nil {
		t.Error("License.txt did not travel with the sounds")
	}
	// The catalogue stays in the binary, not on disk.
	if _, err := os.Stat(filepath.Join(lib.Root, "catalog.json")); err == nil {
		t.Error("catalog.json should not be unpacked")
	}
	// Searchable like any shelf: a click is a click.
	hits, _ := Search(libs, Query{Text: "click", Kind: KindSFX})
	if len(hits) != 5 {
		t.Errorf("click: %v", paths(hits))
	}
	// Second call is a no-op unpack (marker present) and the same shelf.
	again, err := Builtin(data)
	if err != nil || len(again) != 1 || again[0].Root != lib.Root {
		t.Errorf("second Builtin: %v %+v", err, again)
	}
	// The store on disk knows nothing about it.
	s, _ := Load(data)
	if len(s.Libraries) != 0 {
		t.Error("builtin shelf leaked into the store")
	}
}

func TestAFileNamedSongIsNotMusicUnlessItsFolderSaysSo(t *testing.T) {
	icon := classify("ANIMATED LOGOS 2/wired-flat-1093-add-song.gif", 1, Probe{Duration: 3, Width: 400, Height: 400, HasAlpha: true})
	if icon.Kind != KindOverlay {
		t.Errorf("add-song.gif with alpha: %s, want overlay", icon.Kind)
	}
	bed := classify("REEL BACKGROUND MUSIC BY SLIC MEDIA/Armani White - BILLIE EILISH.mp4", 1, Probe{Duration: 180, Width: 1080, Height: 1920})
	if bed.Kind != KindMusic {
		t.Errorf("music folder: %s, want music", bed.Kind)
	}
	// Sounds still read their own name: a podcast bed is a bed.
	pod := classify("71 SFX PACK/Podcast Background Music.mp3", 1, Probe{Duration: 303})
	if pod.Kind != KindMusic {
		t.Errorf("podcast music: %s", pod.Kind)
	}
}

func TestPostersAreForPicturesOnlyAndSeekPastTheBlackFirstFrame(t *testing.T) {
	clip := Asset{ID: "abcdefabcdef", Path: "x/clip.mp4", Duration: 12}
	short := Asset{ID: "abcdefabcdee", Path: "x/blip.mp4", Duration: 0.5}
	still := Asset{ID: "abcdefabcded", Path: "x/icon.png", HasAlpha: true}
	sound := Asset{ID: "abcdefabcdec", Path: "x/whoosh.wav", Duration: 1}
	if !HasPoster(clip) || !HasPoster(still) || HasPoster(sound) {
		t.Error("posters are for clips and stills, never sounds")
	}
	if posterSeek(clip) != 2 || posterSeek(short) != 0 || posterSeek(still) != 0 {
		t.Errorf("seeks: clip %v short %v still %v", posterSeek(clip), posterSeek(short), posterSeek(still))
	}
	if filepath.Ext(ThumbPath("r", still)) != ".png" || filepath.Ext(ThumbPath("r", clip)) != ".jpg" {
		t.Error("alpha posters are PNG, the rest JPEG")
	}
	root := t.TempDir()
	lib := &Library{ID: "l", Root: root, Assets: []Asset{clip, still, sound}}
	th := Thumber{DataRoot: root} // no ffmpeg: nothing renders, nothing crashes
	missing := th.Missing([]*Library{lib}, []string{clip.ID, still.ID, sound.ID, "nope"})
	if len(missing) != 2 {
		t.Errorf("missing = %d, want the clip and the still", len(missing))
	}
	if done := th.Render(context.Background(), []*Library{lib}, missing); len(done) != 0 {
		t.Errorf("rendered without ffmpeg: %v", done)
	}
	// A poster already on disk is not missing.
	os.MkdirAll(filepath.Dir(ThumbPath(root, clip)), 0o755)
	os.WriteFile(ThumbPath(root, clip), []byte("jpg"), 0o644)
	if missing := th.Missing([]*Library{lib}, []string{clip.ID, still.ID}); len(missing) != 1 || missing[0].ID != still.ID {
		t.Errorf("missing after render: %+v", missing)
	}
}

func TestOverridesSurviveARescanAndHideFromTheAgent(t *testing.T) {
	root := shelf(t)
	lib, _ := Scan(context.Background(), root, ProbeFunc(fakeProbe), nil)
	data := t.TempDir()
	s, _ := Load(data)
	s.Put(lib)

	// The scan called the meme an opaque clip; the user says overlay. And the
	// person still is a preview jpeg they never want offered.
	var meme, person Asset
	for _, a := range lib.Assets {
		switch {
		case strings.Contains(a.Path, "this-is-fine"):
			meme = a
		case strings.Contains(a.Path, "person-01"):
			person = a
		}
	}
	s.SetKind(meme.ID, KindOverlay)
	s.SetHidden(person.ID, true)
	if err := s.Save(data); err != nil {
		t.Fatal(err)
	}

	// A rescan replaces the rows; the corrections are elsewhere and stay.
	again, _ := Scan(context.Background(), root, ProbeFunc(fakeProbe), nil)
	s2, _ := Load(data)
	s2.Put(again)
	shown := s2.Apply(s2.Libraries, false)
	_, a, ok := Find(shown, meme.ID)
	if !ok || a.Kind != KindOverlay {
		t.Errorf("kind override lost across rescan: %+v", a)
	}
	if _, _, ok := Find(shown, person.ID); ok {
		t.Error("hidden asset still offered")
	}
	if hits, _ := Search(shown, Query{Kind: KindOverlay}); len(hits) != 3 {
		t.Errorf("overlay search after override: %v", paths(hits))
	}

	// A browser asking to see hidden rows gets them, marked.
	all := s2.Apply(s2.Libraries, true)
	if _, a, ok := Find(all, person.ID); !ok || !a.Hidden {
		t.Errorf("hidden row not shown to a keepHidden caller: %v %+v", ok, a)
	}
	// The originals were not touched.
	if _, a, _ := Find(s2.Libraries, meme.ID); a.Kind != KindClip {
		t.Error("Apply mutated the library it was given")
	}
	// Clearing puts the row back as scanned.
	s2.SetKind(meme.ID, "")
	s2.SetHidden(person.ID, false)
	if len(s2.Overrides) != 0 {
		t.Errorf("cleared overrides linger: %v", s2.Overrides)
	}
}
