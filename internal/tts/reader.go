package tts

// The second half of reading a long reply aloud: turning the pieces Segment
// cut into a stream of audio files, in order, a few at a time.
//
// This sits ABOVE Engine and does not change its contract. An engine still
// takes text and writes one file; it has no idea it is being fed a paragraph
// at a time, which is what let all eight vendors get this for free.
//
// What this file deliberately does NOT hold is how far ahead to run. A look-
// ahead window is a judgement about a listener — how much work is worth
// spending on audio nobody has asked to hear yet — and the listener lives in
// the UI. So the channel below is buffered by exactly one: one finished piece
// may park while the caller decides whether to take the next, and the caller's
// read rate is the throttle. desktop/speak.go is where the window is set,
// against what the webview reports it is actually playing. Same division as
// the voice-for-the-locale rule at the top of tts.go.
//
// How many pieces synthesize AT ONCE is the caller's call for the same reason
// (ReadOptions.Parallel). The model is the owner's kiosk (AI-Robot-Guide,
// AvatarManager.speak): every chunk's request goes out the moment the text is
// cut, and the player waits only for the chunk it is on — so a slow chunk
// costs one wait, not one wait per chunk behind it. Here it is bounded rather
// than all-at-once, and the throttle survives it: a slot is freed when its
// piece is TAKEN, not when the engine finishes it, so a caller that stops
// reading stops the engine within Parallel pieces instead of one. The pieces
// still leave in order whatever order the engine finished them in.
//
// When a voice fails a piece, the read climbs a ladder of stand-ins the caller
// handed it (ReadOptions.Fallbacks) instead of stopping — the kiosk again,
// which lists two voices per language and then another vendor and falls down
// the list until one answers. Which voices are on the ladder is the caller's
// policy. That a voice which failed one piece is not offered the next, and
// that a stop press climbs nothing, are this file's.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PieceTimeout bounds ONE piece, not the read. The old whole-message call had
// three minutes because SAPI on a wedged audio driver sits forever and the
// button had no other way home; the same reasoning applies per piece, where
// three minutes is now enormously generous rather than barely enough.
const PieceTimeout = 3 * time.Minute

// cacheBudget caps the shared chunk cache. Speech is disposable — the cache
// exists so a second press of ฟัง is instant, not so audio is kept — and a
// folder that grows without a ceiling is a bug that takes months to show up.
const cacheBudget = 512 << 20

// Piece is one synthesized chunk of a read. Err set means the read stopped
// there: the caller shows it and nothing further arrives.
type Piece struct {
	Seq  int
	Text string
	// Path is the audio file. It is NOT always inside ReadOptions.Dir — a
	// cache hit points straight at the cached file rather than copying it, so
	// the caller must serve the path it is given and must not assume it owns
	// what it is deleting.
	Path string
	// Mime is the type of what Path holds — the type of the engine that made
	// THIS piece, which after a fallback is not the engine Read was given.
	Mime string
	// Fallback is which voice made the piece: 0 the one Read was given, n the
	// nth entry of ReadOptions.Fallbacks.
	Fallback int
	Last     bool
	Err      error
}

// Fallback is a voice a read may fall back to when a piece fails on the one
// before it: an engine, and the cache key that names ITS configuration. A
// piece it makes is cached as its own voice, never as the one it stood in for
// — the wrong key here is the one way to hear the previous voice speaking.
type Fallback struct {
	Engine   Engine
	CacheKey string
}

// ReadOptions configures one read.
type ReadOptions struct {
	// Dir is where uncached pieces are written. The caller creates it and
	// deletes it when the read is over.
	Dir string
	// CacheDir, when set, is a shared folder of finished pieces so pressing
	// ฟัง on the same reply twice synthesizes nothing the second time. Empty
	// disables the cache.
	CacheDir string
	// CacheKey is everything about the engine's configuration that changes how
	// a given text sounds — vendor, voice, model. This package cannot derive
	// it: Engine deliberately does not expose the Options it was built from,
	// so the caller that built it passes the key. A wrong key here is the one
	// way to hear the previous voice speaking.
	CacheKey string
	// Parallel is how many pieces may be in the engine at once. Zero or one
	// synthesizes them one after another. The window is measured from what the
	// caller has taken, so the caller's read rate stays the throttle; how wide
	// it should be is the caller's judgement, like the look-ahead — a cloud
	// engine is latency-bound and gains from it, a local one is a process per
	// piece and gains less.
	Parallel int
	// Fallbacks names the voices to try, in order, when a piece fails on the
	// engine Read was given. Called once, on the first failure, never before:
	// building the list may mean enumerating another engine's voices, and a
	// read whose voice works should not pay for that. Nil, or an empty list,
	// means a failed piece ends the read the way it always did. Once a piece
	// has fallen back, the pieces started after it begin on the fallback
	// rather than retrying the voice that just failed; a piece already in
	// flight finishes on whatever it was started with.
	Fallbacks func(ctx context.Context) []Fallback
}

// ladder is what a read climbs when a piece fails: the voice it was given,
// then ReadOptions.Fallbacks, resolved once on the first failure.
type ladder struct {
	primary Fallback
	build   func(ctx context.Context) []Fallback
	once    sync.Once
	extra   []Fallback
	// level is the rung every piece started from now on begins at. It only
	// climbs: a voice that failed one piece is not offered the next one.
	level atomic.Int32
}

func (l *ladder) rung(ctx context.Context, n int) (Fallback, bool) {
	if n == 0 {
		return l.primary, true
	}
	l.once.Do(func() {
		if l.build != nil {
			l.extra = l.build(ctx)
		}
	})
	if n-1 < len(l.extra) {
		return l.extra[n-1], true
	}
	return Fallback{}, false
}

func (l *ladder) raise(n int) {
	for {
		cur := l.level.Load()
		if int32(n) <= cur || l.level.CompareAndSwap(cur, int32(n)) {
			return
		}
	}
}

// synthesize makes one piece, climbing the ladder as voices fail it.
func synthesize(ctx context.Context, l *ladder, i, n int, chunk string, opts ReadOptions) Piece {
	p := Piece{Seq: i, Text: chunk, Last: i == n-1}
	r := int(l.level.Load())
	voice, ok := l.rung(ctx, r)
	if !ok {
		// Cannot happen — the level only ever climbs to a rung that exists —
		// but a piece that reported success with no file behind it would be
		// worse than this line.
		p.Err = fmt.Errorf("ไม่มีเสียงให้อ่านชิ้นที่ %d", i+1)
		return p
	}
	var first error
	tried := 0
	for {
		tried++
		o := opts
		o.CacheKey = voice.CacheKey
		path, err := synthesizePiece(ctx, voice.Engine, chunk, extForMime(voice.Engine.Mime()), i, o)
		if err == nil {
			p.Path, p.Mime, p.Fallback = path, voice.Engine.Mime(), r
			return p
		}
		// A stop is a stop, not a reason to try another voice.
		if ctx.Err() != nil {
			p.Err = err
			return p
		}
		if first == nil {
			first = err
		}
		next, ok := l.rung(ctx, r+1)
		if !ok {
			// Out of voices. The first reason is the one the user can act on
			// — it is about the voice they picked.
			p.Err = first
			if tried > 1 {
				p.Err = fmt.Errorf("%w — เสียงสำรองอีก %d เสียงก็ไม่สำเร็จ", first, tried-1)
			}
			return p
		}
		// Climb, and take the pieces started after this one with us. Only
		// to a rung that exists: a level past the top of the ladder would
		// start the next piece on nothing.
		r++
		voice = next
		l.raise(r)
	}
}

// partSeq keeps two reads that start on the same millisecond from choosing the
// same temporary filename in the shared cache folder.
var partSeq atomic.Uint64

// Read segments text and synthesizes it piece by piece, sending each finished
// piece on the returned channel and closing it when the last one is through.
//
// Cancelling ctx stops it where it stands — mid-synthesis, not at the next
// piece boundary — which is what makes a stop button mean stop rather than
// "stop after the rest of this finishes".
func Read(ctx context.Context, eng Engine, text string, opts ReadOptions) <-chan Piece {
	out := make(chan Piece, 1)
	go func() {
		defer close(out)
		chunks := Segment(text)
		if len(chunks) == 0 {
			send(ctx, out, Piece{Last: true, Err: fmt.Errorf("ไม่มีข้อความให้อ่าน")})
			return
		}
		if opts.CacheDir != "" {
			if err := os.MkdirAll(opts.CacheDir, 0o755); err != nil {
				opts.CacheDir = "" // an unusable cache is not a reason to stay silent
			} else {
				go pruneCache(opts.CacheDir, cacheBudget)
			}
		}
		// Leaving early — a failed piece, a caller that cancelled — takes the
		// pieces still synthesizing with it, and the channel does not close
		// until they have gone: closed means no piece is still being written,
		// so a caller may delete Dir the moment it sees the close. Defers run
		// in reverse, so the cancel lands before the wait.
		var wg sync.WaitGroup
		defer wg.Wait()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		voices := &ladder{primary: Fallback{Engine: eng, CacheKey: opts.CacheKey}, build: opts.Fallbacks}
		width := opts.Parallel
		if width < 1 {
			width = 1
		}
		// slots bounds the pieces that are in flight or finished and not yet
		// taken. A slot is acquired to start a piece and freed when that piece
		// LEAVES on the channel — not when the engine finishes it — so the
		// caller's read rate is still the throttle: stop taking pieces and the
		// launcher below stops being let through.
		slots := make(chan struct{}, width)
		results := make([]chan Piece, len(chunks))
		for i := range results {
			results[i] = make(chan Piece, 1)
		}
		// The launcher is counted too, so a worker it starts after the wait
		// began is still one the wait sees.
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i, chunk := range chunks {
				select {
				case slots <- struct{}{}:
				case <-ctx.Done():
					return
				}
				wg.Add(1)
				go func(i int, chunk string) {
					defer wg.Done()
					results[i] <- synthesize(ctx, voices, i, len(chunks), chunk, opts)
				}(i, chunk)
			}
		}()
		// Out in order, whatever order the engine finished them in.
		for i := range chunks {
			var p Piece
			select {
			case p = <-results[i]:
			case <-ctx.Done():
				return
			}
			if !send(ctx, out, p) || p.Err != nil {
				return
			}
			<-slots
		}
	}()
	return out
}

func send(ctx context.Context, out chan<- Piece, p Piece) bool {
	select {
	case out <- p:
		return true
	case <-ctx.Done():
		return false
	}
}

// synthesizePiece writes one piece, through the cache when there is one.
//
// A cache hit returns the cached path directly rather than copying it into the
// job folder: the copy would be the same bytes under a second name, and the
// only reason to want one is an assumption — "everything I serve lives in my
// own folder" — that the caller is better off not making.
func synthesizePiece(ctx context.Context, eng Engine, text, ext string, seq int, opts ReadOptions) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, PieceTimeout)
	defer cancel()

	if opts.CacheDir == "" || opts.CacheKey == "" {
		path := filepath.Join(opts.Dir, fmt.Sprintf("%04d%s", seq, ext))
		if err := eng.Synthesize(ctx, text, path); err != nil {
			return "", err
		}
		return path, nil
	}

	final := filepath.Join(opts.CacheDir, pieceHash(opts.CacheKey, text)+ext)
	if info, err := os.Stat(final); err == nil && info.Size() > 0 {
		// Touch it so a piece that keeps being played is not the one the
		// pruner decides is coldest.
		now := time.Now()
		_ = os.Chtimes(final, now, now)
		return final, nil
	}
	// Synthesize to a private name and rename into place, so a half-written
	// file is never visible under the name a cache hit trusts.
	part := filepath.Join(opts.CacheDir, fmt.Sprintf("part-%d-%d%s", os.Getpid(), partSeq.Add(1), ext))
	if err := eng.Synthesize(ctx, text, part); err != nil {
		_ = os.Remove(part)
		return "", err
	}
	if err := os.Rename(part, final); err != nil {
		// Losing the rename loses the cache entry, not the audio.
		return part, nil
	}
	return final, nil
}

func pieceHash(key, text string) string {
	sum := sha256.Sum256([]byte(key + "\x00" + text))
	return hex.EncodeToString(sum[:16])
}

// extForMime names the file after what the engine actually wrote. The local
// engines write WAV and the cloud ones mostly write MP3; a .wav holding MP3 is
// the kind of thing that plays everywhere until the one place it does not.
func extForMime(mime string) string {
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(mime, ";", 2)[0])) {
	case "audio/wav", "audio/x-wav", "audio/wave":
		return ".wav"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "audio/opus":
		return ".opus"
	case "audio/flac":
		return ".flac"
	case "audio/aac", "audio/mp4", "audio/m4a":
		return ".m4a"
	default:
		return ".audio"
	}
}

// pruneCache deletes the coldest files until the folder is back under budget.
// Best-effort throughout: a cache that cannot be tidied is still a cache, and
// this runs on a goroutine nobody waits for.
func pruneCache(dir string, budget int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type item struct {
		path string
		size int64
		at   time.Time
	}
	var files []item
	var total int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, item{filepath.Join(dir, e.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	if total <= budget {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].at.Before(files[j].at) })
	for _, f := range files {
		if total <= budget {
			return
		}
		if os.Remove(f.path) == nil {
			total -= f.size
		}
	}
}
