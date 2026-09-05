package tts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingEngine stands in for a vendor: it writes a file with the text in it,
// counts the calls, and can be made slow or made to fail.
type countingEngine struct {
	calls atomic.Int64
	mime  string
	delay time.Duration
	fail  error
	// stagger makes later calls finish sooner, so pieces started together
	// come back out of order.
	stagger bool

	mu       sync.Mutex
	said     []string
	inflight int64
	peak     int64 // the most calls that were ever in progress at once
}

func (e *countingEngine) peakInflight() int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.peak
}

func (e *countingEngine) ID() string { return "counting" }

func (e *countingEngine) Mime() string {
	if e.mime == "" {
		return "audio/wav"
	}
	return e.mime
}

func (e *countingEngine) Voices(context.Context) ([]Voice, error) { return nil, nil }

func (e *countingEngine) Synthesize(ctx context.Context, text, outPath string) error {
	n := e.calls.Add(1)
	e.mu.Lock()
	e.said = append(e.said, text)
	e.inflight++
	if e.inflight > e.peak {
		e.peak = e.inflight
	}
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		e.inflight--
		e.mu.Unlock()
	}()
	delay := e.delay
	if e.stagger {
		delay = time.Duration(4-n%4) * 30 * time.Millisecond
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if e.fail != nil {
		return e.fail
	}
	return os.WriteFile(outPath, []byte(text), 0o600)
}

func drain(t *testing.T, ch <-chan Piece) []Piece {
	t.Helper()
	var out []Piece
	for p := range ch {
		out = append(out, p)
	}
	return out
}

const longThai = "ผมกำลังทดสอบระบบเสียงอ่านของแอปนี้ซึ่งต้องแบ่งข้อความออกเป็นชิ้นเล็กๆ ก่อนจะสังเคราะห์ทีละชิ้น "

func TestReadDeliversEveryPieceInOrderAndFlagsTheLast(t *testing.T) {
	eng := &countingEngine{}
	pieces := drain(t, Read(context.Background(), eng, strings.Repeat(longThai, 8), ReadOptions{Dir: t.TempDir()}))
	if len(pieces) < 3 {
		t.Fatalf("a long text produced %d pieces, want several", len(pieces))
	}
	for i, p := range pieces {
		if p.Err != nil {
			t.Fatalf("piece %d failed: %v", i, p.Err)
		}
		if p.Seq != i {
			t.Errorf("piece %d reports Seq %d", i, p.Seq)
		}
		if p.Last != (i == len(pieces)-1) {
			t.Errorf("piece %d has Last=%v", i, p.Last)
		}
		body, err := os.ReadFile(p.Path)
		if err != nil {
			t.Fatalf("piece %d has no file: %v", i, err)
		}
		if string(body) != p.Text {
			t.Errorf("piece %d file holds %q, want %q", i, body, p.Text)
		}
	}
	if got := int64(len(pieces)); eng.calls.Load() != got {
		t.Errorf("engine was called %d times for %d pieces", eng.calls.Load(), got)
	}
}

// The whole point of the look-ahead living with the caller: a caller that
// stops reading must stop the synthesizer within a piece or two, not let it
// run to the end of the reply. The channel's single slot is what enforces it.
func TestReadDoesNotRunAheadOfACallerThatStopsReading(t *testing.T) {
	eng := &countingEngine{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := Read(ctx, eng, strings.Repeat(longThai, 30), ReadOptions{Dir: t.TempDir()})
	<-ch // take exactly one piece, then behave like a listener still on it
	time.Sleep(150 * time.Millisecond)
	if n := eng.calls.Load(); n > 3 {
		t.Errorf("engine ran %d pieces ahead of a caller holding at the first", n)
	}
}

func TestCancelStopsTheReadMidSynthesis(t *testing.T) {
	eng := &countingEngine{delay: 30 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	ch := Read(ctx, eng, strings.Repeat(longThai, 8), ReadOptions{Dir: t.TempDir()})
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-ch:
		// Either the channel closed or an error piece came through — both are
		// "it stopped". What must not happen is waiting out the 30s delay.
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop the read")
	}
	if n := eng.calls.Load(); n != 1 {
		t.Errorf("engine was called %d times, want 1 — cancel should not start another piece", n)
	}
}

func TestEngineFailureEndsTheReadWithTheReason(t *testing.T) {
	eng := &countingEngine{fail: fmt.Errorf("ไม่มีเสียงในเครื่อง")}
	pieces := drain(t, Read(context.Background(), eng, strings.Repeat(longThai, 8), ReadOptions{Dir: t.TempDir()}))
	if len(pieces) != 1 {
		t.Fatalf("got %d pieces, want the failure and nothing after it", len(pieces))
	}
	if pieces[0].Err == nil || !strings.Contains(pieces[0].Err.Error(), "ไม่มีเสียงในเครื่อง") {
		t.Errorf("the engine's own reason did not reach the caller: %v", pieces[0].Err)
	}
}

func TestEmptyTextIsOneErrorPieceNotSilence(t *testing.T) {
	pieces := drain(t, Read(context.Background(), &countingEngine{}, "   \n ", ReadOptions{Dir: t.TempDir()}))
	if len(pieces) != 1 || pieces[0].Err == nil {
		t.Fatalf("got %+v, want a single piece carrying the reason", pieces)
	}
}

// Pressing ฟัง on the same reply twice must synthesize nothing the second
// time — the reason the cache exists at all.
func TestCacheMakesTheSecondReadFree(t *testing.T) {
	cache := t.TempDir()
	text := strings.Repeat(longThai, 6)
	opts := ReadOptions{Dir: t.TempDir(), CacheDir: cache, CacheKey: "windows|Pattara|"}

	first := &countingEngine{}
	a := drain(t, Read(context.Background(), first, text, opts))
	if first.calls.Load() == 0 {
		t.Fatal("the first read synthesized nothing")
	}

	second := &countingEngine{}
	opts.Dir = t.TempDir()
	b := drain(t, Read(context.Background(), second, text, opts))
	if second.calls.Load() != 0 {
		t.Errorf("the second read called the engine %d times, want 0", second.calls.Load())
	}
	if len(a) != len(b) {
		t.Fatalf("cached read produced %d pieces, first produced %d", len(b), len(a))
	}
	for i := range a {
		if a[i].Path != b[i].Path || b[i].Err != nil {
			t.Errorf("piece %d: cached read gave %q (%v), want %q", i, b[i].Path, b[i].Err, a[i].Path)
		}
	}
}

// A different voice is a different sound for the same words. Sharing a cache
// entry across voices is the one bug that would make the setting look broken.
func TestCacheKeySeparatesVoices(t *testing.T) {
	cache := t.TempDir()
	text := strings.Repeat(longThai, 4)

	one := &countingEngine{}
	drain(t, Read(context.Background(), one, text, ReadOptions{Dir: t.TempDir(), CacheDir: cache, CacheKey: "windows|Pattara|"}))
	two := &countingEngine{}
	drain(t, Read(context.Background(), two, text, ReadOptions{Dir: t.TempDir(), CacheDir: cache, CacheKey: "windows|Niwat|"}))

	if two.calls.Load() != one.calls.Load() {
		t.Errorf("second voice reused the first voice's audio: %d calls vs %d", two.calls.Load(), one.calls.Load())
	}
}

func TestCachedFilesAreNotHalfWritten(t *testing.T) {
	cache := t.TempDir()
	eng := &countingEngine{fail: fmt.Errorf("ล้มกลางคัน")}
	drain(t, Read(context.Background(), eng, strings.Repeat(longThai, 4), ReadOptions{Dir: t.TempDir(), CacheDir: cache, CacheKey: "k"}))
	entries, err := os.ReadDir(cache)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "part-") {
			t.Errorf("a failed synthesis left %q in the cache under a name a hit would trust", e.Name())
		}
	}
}

func TestPieceFilesAreNamedAfterWhatTheEngineWrote(t *testing.T) {
	cases := map[string]string{
		"audio/wav":           ".wav",
		"audio/mpeg":          ".mp3",
		"audio/ogg":           ".ogg",
		"application/octet":   ".audio",
		"audio/wav; codecs=1": ".wav",
	}
	for mime, want := range cases {
		if got := extForMime(mime); got != want {
			t.Errorf("extForMime(%q) = %q, want %q", mime, got, want)
		}
	}
	eng := &countingEngine{mime: "audio/mpeg"}
	pieces := drain(t, Read(context.Background(), eng, strings.Repeat(longThai, 4), ReadOptions{Dir: t.TempDir()}))
	if len(pieces) == 0 || filepath.Ext(pieces[0].Path) != ".mp3" {
		t.Errorf("an MP3 engine wrote %q", pieces[0].Path)
	}
}

func TestPruneCacheDropsTheColdestFirst(t *testing.T) {
	dir := t.TempDir()
	names := []string{"cold", "warm", "hot"}
	for i, n := range names {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, make([]byte, 400), 0o600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(time.Duration(i-len(names)) * time.Hour)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	pruneCache(dir, 900)
	if _, err := os.Stat(filepath.Join(dir, "cold")); err == nil {
		t.Error("the coldest file survived the prune")
	}
	if _, err := os.Stat(filepath.Join(dir, "hot")); err != nil {
		t.Error("the newest file was pruned")
	}
}

// The owner's kiosk fires every chunk's request the moment the text is cut.
// Here the same idea is bounded: with Parallel set, that many pieces are in
// the engine at once — no more, no fewer while there is work.
func TestParallelKeepsThatManyPiecesInTheEngine(t *testing.T) {
	eng := &countingEngine{delay: 120 * time.Millisecond}
	pieces := drain(t, Read(context.Background(), eng, strings.Repeat(longThai, 12), ReadOptions{Dir: t.TempDir(), Parallel: 3}))
	if len(pieces) < 6 {
		t.Fatalf("got %d pieces, want enough to fill the window several times", len(pieces))
	}
	for i, p := range pieces {
		if p.Err != nil {
			t.Fatalf("piece %d failed: %v", i, p.Err)
		}
	}
	if got := eng.peakInflight(); got != 3 {
		t.Errorf("the engine had at most %d pieces in progress at once, want 3", got)
	}
}

// Pieces leave in order even when the engine finishes them in another.
func TestParallelStillDeliversInOrder(t *testing.T) {
	eng := &countingEngine{stagger: true}
	pieces := drain(t, Read(context.Background(), eng, strings.Repeat(longThai, 12), ReadOptions{Dir: t.TempDir(), Parallel: 4}))
	want := Segment(strings.Repeat(longThai, 12))
	if len(pieces) != len(want) {
		t.Fatalf("got %d pieces, want %d", len(pieces), len(want))
	}
	for i, p := range pieces {
		if p.Seq != i || p.Text != want[i] {
			t.Errorf("piece %d is seq %d %.30q, want %.30q", i, p.Seq, p.Text, want[i])
		}
		body, err := os.ReadFile(p.Path)
		if err != nil || string(body) != p.Text {
			t.Errorf("piece %d file does not hold its own text (%v)", i, err)
		}
		if p.Last != (i == len(pieces)-1) {
			t.Errorf("piece %d has Last=%v", i, p.Last)
		}
	}
}

// The throttle survives parallelism: a slot frees when a piece is taken, not
// when it is made, so a caller that stops reading stops the engine within
// Parallel pieces.
func TestParallelStillStopsForACallerThatStopsReading(t *testing.T) {
	eng := &countingEngine{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := Read(ctx, eng, strings.Repeat(longThai, 30), ReadOptions{Dir: t.TempDir(), Parallel: 3})
	<-ch
	time.Sleep(150 * time.Millisecond)
	// One taken, one parked in the channel, three holding slots.
	if n := eng.calls.Load(); n > 5 {
		t.Errorf("engine ran %d pieces for a caller holding at the first, want at most 5", n)
	}
}

func TestCancelStopsEveryParallelPiece(t *testing.T) {
	eng := &countingEngine{delay: 30 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	ch := Read(ctx, eng, strings.Repeat(longThai, 12), ReadOptions{Dir: t.TempDir(), Parallel: 3})
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not stop the read")
	}
	if n := eng.calls.Load(); n != 3 {
		t.Errorf("engine was called %d times, want the 3 that were in flight and no more", n)
	}
}

// A piece that fails ends the read at its place in the order, not the moment
// it fails: what came before it is still delivered, whole.
func TestParallelFailureStillArrivesInOrder(t *testing.T) {
	eng := &countingEngine{fail: fmt.Errorf("ล้มกลางคัน")}
	pieces := drain(t, Read(context.Background(), eng, strings.Repeat(longThai, 12), ReadOptions{Dir: t.TempDir(), Parallel: 3}))
	if len(pieces) != 1 || pieces[0].Seq != 0 || pieces[0].Err == nil {
		t.Fatalf("got %+v, want the first piece carrying the failure and nothing after it", pieces)
	}
}
