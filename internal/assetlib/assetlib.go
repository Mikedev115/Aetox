// Package assetlib is the studio's shelf of raw material: the sound effects,
// overlays, motion backgrounds and animated icons that a video is dressed with
// once its shape has been decided.
//
// The material is the user's, and it stays where the user put it. Aetox keeps
// only a catalogue — one entry per file with the facts an agent needs before it
// can pick one without watching it: how long it runs, what size it is, whether
// it carries transparency, and which words its name and folders are made of.
// Those facts are read once, at scan time, by ffprobe, and a search afterwards
// touches no file at all.
//
// Why a catalogue and not a skill listing the files: the first shelf offered
// to Aetox held thirty gigabytes in forty folders (12 ก.ย. 2569). A skill is
// re-sent to the model on every round of every session, and the templates
// skill is already the largest thing an agent carries at seventy-five rows.
// Thousands of rows would be paid for on every message and read on none of
// them. A search tool costs one call for the twenty rows that matter.
//
// Why Aetox does not fetch the material itself: nothing in that thirty
// gigabytes came with a licence, and some of it (a commercial title-card pack,
// a folder of memes) plainly cannot be redistributed by anybody. Every
// download in internal/capability names whose program it is and under what
// terms; this package cannot say that about a stranger's Drive folder and so
// must not be the one putting it on a machine. The user fetches what they
// have the right to, points Aetox at the folder, and the rights stay theirs —
// exactly as they do for a file attached to a chat.
package assetlib

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Kind is what a file is FOR in a video, which is the question an agent asks.
// It is inferred, and the inference can be wrong, so Asset keeps the user's
// own folder name beside it for the agent to read when the kind looks off.
type Kind string

const (
	// KindSFX is a short sound: a whoosh, a click, a cash register.
	KindSFX Kind = "sfx"
	// KindMusic is a long sound meant to sit under a piece.
	KindMusic Kind = "music"
	// KindOverlay is a clip with transparency, or green-screen footage to be
	// keyed: something laid over the user's own picture.
	KindOverlay Kind = "overlay"
	// KindBackground is an opaque loop meant to sit under everything.
	KindBackground Kind = "background"
	// KindClip is opaque footage that is neither — a meme, a stock shot.
	KindClip Kind = "clip"
	// KindIcon is a small still meant to be placed, not filled: an icon, an
	// emoji, a logo, an arrow.
	KindIcon Kind = "icon"
	// KindImage is any other still.
	KindImage Kind = "image"
)

// Kinds is every kind, in the order a summary reads them out.
var Kinds = []Kind{KindSFX, KindMusic, KindOverlay, KindBackground, KindClip, KindIcon, KindImage}

// Asset is one file on the shelf.
type Asset struct {
	// ID is stable across rescans and across the user moving the whole
	// library: a hash of the path relative to the root, not of the bytes,
	// because hashing thirty gigabytes is not a scan anyone waits for.
	ID string `json:"id"`
	// Path is relative to the library root, forward slashes.
	Path string `json:"path"`
	Kind Kind   `json:"kind"`
	// Category is the first folder under the root, exactly as the user named
	// it. Not translated and not normalised: it is how they think of the
	// shelf, and it is the word to fall back on when Kind guessed wrong.
	Category string  `json:"category"`
	Duration float64 `json:"duration,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	HasAlpha bool    `json:"alpha,omitempty"`
	Bytes    int64   `json:"bytes"`
	// Tags are the words of the path, lowercased, for Search to match against.
	Tags []string `json:"tags"`
	// Hidden is set by Store.Apply on a row the user has hidden, when a caller
	// asked to keep such rows. Never written by a scan and never persisted
	// here — the store's Overrides are the record.
	Hidden bool `json:"hidden,omitempty"`
}

// Name is the file's own name without its extension — what a person calls it.
func (a Asset) Name() string {
	base := a.Path
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// Library is one root the user pointed at, and everything found under it.
type Library struct {
	ID      string    `json:"id"`
	Root    string    `json:"root"`
	Scanned time.Time `json:"scanned"`
	// Bytes is the whole shelf's size on disk, for the sentence on the screen.
	Bytes  int64   `json:"bytes"`
	Assets []Asset `json:"assets"`
	// Unread is how many media files the probe could not open. Kept as a
	// number rather than a list: a shelf with three broken files is still a
	// shelf, and the user has no use for the paths in Settings.
	Unread int `json:"unread,omitempty"`

	// Builtin marks a shelf that ships inside Aetox (builtin.go): not the
	// user's, not removable, and never written to the Store. Title, License
	// and Source are what its card says instead of a folder path — set only
	// for those, empty for a user's folder, which is called by its own name.
	Builtin bool   `json:"builtin,omitempty"`
	Title   string `json:"title,omitempty"`
	License string `json:"license,omitempty"`
	Source  string `json:"source,omitempty"`
}

// Name is what a shelf is called on screen: a bundled one by its title, a
// user's folder by its last path segment — "30GB+ Video Editing Assets", not
// the whole of the Downloads path, which is what the first page showed.
func (l *Library) Name() string {
	if l.Title != "" {
		return l.Title
	}
	return filepath.Base(l.Root)
}

// Counts is how many of each kind a library holds.
func (l *Library) Counts() map[Kind]int {
	out := map[Kind]int{}
	for _, a := range l.Assets {
		out[a.Kind]++
	}
	return out
}

// Probe is what a media file says about itself once opened.
type Probe struct {
	Duration float64
	Width    int
	Height   int
	// HasAlpha is whether the picture carries transparency, which for a
	// clip is the whole difference between an overlay and a background.
	HasAlpha bool
}

// Prober opens files and reads their facts. FFTools is the one this package
// ships; tests pass a ProbeFunc.
type Prober interface {
	Probe(ctx context.Context, path string) (Probe, error)
}

// BatchProber is a Prober that can read many files in one go, and a scan
// prefers it when it is offered.
//
// The reason is measured, not guessed: on the owner's machine (12 ก.ย. 2569)
// starting ffprobe cost 2–6 seconds per launch with the file itself taking
// milliseconds — Defender inspecting a freshly-fetched executable on every
// start, most likely — so 1,855 files came out at over an hour through one
// process each. One ffmpeg fed forty inputs reads forty headers in one start.
type BatchProber interface {
	Prober
	// ProbeMany returns one Probe and one error per path, in order.
	ProbeMany(ctx context.Context, paths []string) ([]Probe, []error)
}

// ProbeFunc is a Prober made of one function, for tests and for callers with
// their own reader.
type ProbeFunc func(ctx context.Context, path string) (Probe, error)

// Probe implements Prober.
func (f ProbeFunc) Probe(ctx context.Context, path string) (Probe, error) { return f(ctx, path) }

// batchSize is how many files one ProbeMany call is handed. Forty keeps a
// command line well inside Windows' 32K limit on the longest paths a shelf
// has shown, and keeps one failed batch's fallback to single probes cheap.
const batchSize = 40

// Progress is one step of a scan: files probed so far, of how many.
type Progress struct {
	Done  int
	Total int
}

// mediaExt is every extension a scan looks at. Anything else under the root
// — a readme, a licence, a preview jpg named "thumbnail" — is left alone,
// with one exception: stills ARE material here (icons, emojis, memes).
var mediaExt = map[string]string{
	".wav": "audio", ".mp3": "audio", ".ogg": "audio", ".m4a": "audio", ".aac": "audio", ".flac": "audio", ".aiff": "audio", ".aif": "audio",
	".mov": "video", ".webm": "video", ".mp4": "video", ".mkv": "video", ".avi": "video", ".m4v": "video", ".gif": "video",
	".png": "image", ".jpg": "image", ".jpeg": "image", ".webp": "image", ".svg": "image", ".bmp": "image", ".tif": "image", ".tiff": "image",
}

// scanWorkers is how many probes run at once. ffprobe on a local disk is
// bound by process start-up more than by reading, so a few in flight is the
// difference between a scan that takes minutes and one that takes an hour on
// a shelf of several thousand files; more than a few and a laptop's disk is
// the thing waiting instead.
const scanWorkers = 4

// Scan walks one root and builds its library. Every media file is probed;
// one that cannot be opened is still catalogued from its name and counted in
// Unread, because a shelf with a broken file on it is not a broken shelf.
//
// onProgress is called after every probe, and may be nil.
func Scan(ctx context.Context, root string, probe Prober, onProgress func(Progress)) (*Library, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		return nil, errors.New("ยังไม่ได้บอกว่าคลังอยู่โฟลเดอร์ไหน")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s ไม่ใช่โฟลเดอร์", root)
	}
	if probe == nil {
		return nil, errors.New("assetlib: probe is nil")
	}

	// Two passes, and the first is the cheap one: a walk that only counts,
	// so the progress bar has a denominator before the first slow probe.
	type found struct {
		rel   string
		bytes int64
	}
	var files []found
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // an unreadable subfolder is skipped, not fatal
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if _, ok := mediaExt[strings.ToLower(filepath.Ext(name))]; !ok {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		files = append(files, found{rel: filepath.ToSlash(rel), bytes: fi.Size()})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	lib := &Library{ID: libraryID(root), Root: root, Scanned: time.Now()}
	assets := make([]Asset, len(files))
	unread := make([]bool, len(files))

	// Work is handed out in batches of indices. A BatchProber gets the whole
	// batch in one call; a plain Prober gets it one file at a time, which is
	// the same loop with a batch of one.
	batcher, canBatch := probe.(BatchProber)
	size := 1
	if canBatch {
		size = batchSize
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		done int
		jobs = make(chan []int)
	)
	record := func(i int, p Probe, perr error) {
		f := files[i]
		assets[i] = classify(f.rel, f.bytes, p)
		unread[i] = perr != nil
		mu.Lock()
		done++
		n := done
		mu.Unlock()
		if onProgress != nil {
			onProgress(Progress{Done: n, Total: len(files)})
		}
	}
	for w := 0; w < scanWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range jobs {
				paths := make([]string, len(batch))
				for j, i := range batch {
					paths[j] = filepath.Join(root, filepath.FromSlash(files[i].rel))
				}
				if canBatch && len(batch) > 1 {
					probes, errs := batcher.ProbeMany(ctx, paths)
					for j, i := range batch {
						record(i, probes[j], errs[j])
					}
					continue
				}
				for j, i := range batch {
					p, perr := probe.Probe(ctx, paths[j])
					record(i, p, perr)
				}
			}
		}()
	}
	for start := 0; start < len(files) && ctx.Err() == nil; start += size {
		end := start + size
		if end > len(files) {
			end = len(files)
		}
		batch := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			batch = append(batch, i)
		}
		jobs <- batch
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for i, a := range assets {
		lib.Assets = append(lib.Assets, a)
		lib.Bytes += a.Bytes
		if unread[i] {
			lib.Unread++
		}
	}
	sort.Slice(lib.Assets, func(i, j int) bool { return lib.Assets[i].Path < lib.Assets[j].Path })
	return lib, nil
}

// classify turns a path and its probe into an asset. The probe may be the
// zero value when the file could not be opened, and the rules degrade to the
// extension and the words then.
func classify(rel string, bytes int64, p Probe) Asset {
	tags := words(rel)
	// The folders' words alone, for the rules where a file's own name lies:
	// an animated icon called "add-song.gif" is not a song, but a clip in a
	// folder called BACKGROUND MUSIC is (both from the first real shelf).
	dirWords := words(path.Dir(rel))
	anyOf := func(in []string, names ...string) bool {
		for _, t := range in {
			for _, n := range names {
				if t == n {
					return true
				}
			}
		}
		return false
	}
	has := func(names ...string) bool { return anyOf(tags, names...) }
	inDir := func(names ...string) bool { return anyOf(dirWords, names...) }
	a := Asset{
		ID:       assetID(rel),
		Path:     rel,
		Category: category(rel),
		Duration: p.Duration,
		Width:    p.Width,
		Height:   p.Height,
		HasAlpha: p.HasAlpha,
		Bytes:    bytes,
		Tags:     tags,
	}
	musicWords := []string{"music", "bgm", "song", "songs", "soundtrack", "track", "tracks", "loop", "loops"}
	kind := mediaExt[strings.ToLower(filepath.Ext(rel))]
	// A "video" with no picture is a sound in a video container — an .mp4 of
	// a song, which the first real shelf had a folder of. Judged as audio.
	if kind == "video" && p.Width == 0 && p.Duration > 0 {
		kind = "audio"
	}
	switch kind {
	case "audio":
		// Forty-five seconds is where a sound stops being an effect and starts
		// being a bed; the folder's own word wins when it has one.
		if has(musicWords...) || p.Duration > 45 {
			a.Kind = KindMusic
		} else {
			a.Kind = KindSFX
		}
		// An mp3's embedded cover art arrives from the probe as a video stream
		// with a size. It is not the asset, and a 1280x720 sound effect is a
		// row that makes an agent ask what it is looking at.
		a.Width, a.Height, a.HasAlpha = 0, 0, false
	case "video":
		switch {
		case inDir(musicWords...):
			// A music video in a folder called BACKGROUND MUSIC is a bed with a
			// picture attached, and the folder is the honest word for it. The
			// folder, not the file: "add-song.gif" is an icon.
			a.Kind = KindMusic
		case p.HasAlpha,
			has("green", "greenscreen", "chroma", "overlay", "overlays", "transition", "transitions", "burn", "leak", "leaks", "glitch", "particles"):
			// Green-screen footage has no alpha channel and is still made to be
			// laid over something; the name is the only thing that says so.
			a.Kind = KindOverlay
		case has("background", "backgrounds", "bg", "texture", "textures", "grid", "cloud", "clouds", "gradient", "gradients"):
			a.Kind = KindBackground
		default:
			a.Kind = KindClip
		}
	default:
		if has("icon", "icons", "emoji", "emojis", "logo", "logos", "arrow", "arrows", "sticker", "stickers", "letter", "letters") {
			a.Kind = KindIcon
		} else {
			a.Kind = KindImage
		}
	}
	return a
}

// category is the first folder under the root, or "" for a file at the root.
func category(rel string) string {
	if i := strings.Index(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return ""
}

// words breaks a relative path into lowercase search tokens: every folder
// name and the file name, split on anything that is not a letter or digit.
// "GREEN SCREEN EFFECTS/Cash_Register-02.mov" → green screen effects cash
// register 02 mov. Duplicates are dropped; order is kept so the file's own
// words come last and a reader can tell folder from file.
func words(rel string) []string {
	var out []string
	seen := map[string]bool{}
	fields := strings.FieldsFunc(strings.ToLower(rel), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, f := range fields {
		if len(f) < 2 && !unicode.IsDigit([]rune(f)[0]) {
			continue
		}
		if seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

func assetID(rel string) string {
	sum := sha1.Sum([]byte(strings.ToLower(filepath.ToSlash(rel))))
	return hex.EncodeToString(sum[:])[:12]
}

func libraryID(root string) string {
	sum := sha1.Sum([]byte(strings.ToLower(filepath.ToSlash(filepath.Clean(root)))))
	return hex.EncodeToString(sum[:])[:8]
}

// Query is one search. Every field is optional; an empty query lists the
// shelf in path order, which is what a caller asking "what is here" wants.
type Query struct {
	// Text is free words, matched against Tags. Every word must match.
	Text string
	Kind Kind
	// MaxSeconds drops anything longer; 0 means no limit. Stills (duration 0)
	// always pass, because a picture is as long as you show it.
	MaxSeconds float64
	// NeedAlpha keeps only material that can be laid over something.
	NeedAlpha bool
	// Limit caps the result; 0 means DefaultLimit.
	Limit int
}

// DefaultLimit is how many rows a search returns when nobody said. Twenty is
// what a model reads without skimming, and what the templates skill settled
// on for its own refusals.
const DefaultLimit = 20

// Search runs one query over several libraries and returns the best rows,
// with how many matched in all so a caller can say "20 of 340".
//
// Ranking is by how much of the path the words cover — a query of "cash" puts
// "MONEY SFX/cash.wav" above "ASSETS/cash-register-long-loop-v2.wav" — with
// shorter paths winning ties, so the file named for the thing outranks the
// one that merely mentions it.
func Search(libs []*Library, q Query) ([]Asset, int) {
	terms := words(q.Text)
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	type hit struct {
		a     Asset
		score float64
	}
	var hits []hit
	for _, lib := range libs {
		if lib == nil {
			continue
		}
		for _, a := range lib.Assets {
			if q.Kind != "" && a.Kind != q.Kind {
				continue
			}
			if q.NeedAlpha && !a.HasAlpha {
				continue
			}
			if q.MaxSeconds > 0 && a.Duration > q.MaxSeconds {
				continue
			}
			score, ok := match(a.Tags, terms)
			if !ok {
				continue
			}
			hits = append(hits, hit{a: a, score: score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].a.Path < hits[j].a.Path
	})
	total := len(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]Asset, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.a)
	}
	return out, total
}

// match reports whether every term is found in the tags — a term matches a
// tag it equals or is a prefix of, so "whoosh" finds "whooshes" — and scores
// the hit by the share of tags the terms account for.
func match(tags, terms []string) (float64, bool) {
	if len(terms) == 0 {
		return 0, true
	}
	matched := 0
	for _, term := range terms {
		found := false
		for _, tag := range tags {
			if tag == term || strings.HasPrefix(tag, term) {
				found = true
				break
			}
		}
		if !found {
			return 0, false
		}
		matched++
	}
	if len(tags) == 0 {
		return 0, true
	}
	return float64(matched) / float64(len(tags)), true
}

// Find returns one asset by id across the libraries, with the library it came
// from so a caller can resolve its full path.
func Find(libs []*Library, id string) (*Library, *Asset, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, lib := range libs {
		if lib == nil {
			continue
		}
		for i := range lib.Assets {
			if lib.Assets[i].ID == id {
				return lib, &lib.Assets[i], true
			}
		}
	}
	return nil, nil, false
}

// FullPath is where an asset actually lives on disk.
func (l *Library) FullPath(a Asset) string {
	return filepath.Join(l.Root, filepath.FromSlash(a.Path))
}

// ---------------------------------------------------------------- the store

// Store is every library the user has added, as saved on disk.
type Store struct {
	Libraries []*Library `json:"libraries"`
	// Overrides are the user's corrections to what a scan guessed, keyed by
	// asset id — which is a hash of the path, so a correction survives every
	// rescan and the library moving. Kept apart from the rows on purpose: a
	// rescan REPLACES a library's rows, and a correction folded into a row
	// would be lost the moment the folder was read again.
	Overrides map[string]Override `json:"overrides,omitempty"`
}

// Override is one correction: a kind the machine got wrong, a file that
// should not be offered to the agent at all (a Premiere extension's preview
// jpegs, say). Zero values mean "no correction" for that field.
type Override struct {
	Kind   Kind `json:"kind,omitempty"`
	Hidden bool `json:"hidden,omitempty"`
}

// SetKind records a corrected kind ("" clears it).
func (s *Store) SetKind(id string, k Kind) {
	o := s.override(id)
	o.Kind = k
	s.putOverride(id, o)
}

// SetHidden records whether an asset is kept from the agent and the browser.
func (s *Store) SetHidden(id string, hidden bool) {
	o := s.override(id)
	o.Hidden = hidden
	s.putOverride(id, o)
}

func (s *Store) override(id string) Override {
	if s.Overrides == nil {
		return Override{}
	}
	return s.Overrides[id]
}

func (s *Store) putOverride(id string, o Override) {
	if o == (Override{}) {
		delete(s.Overrides, id)
		return
	}
	if s.Overrides == nil {
		s.Overrides = map[string]Override{}
	}
	s.Overrides[id] = o
}

// Apply returns the libraries with the store's corrections laid over their
// rows: corrected kinds replaced, hidden rows dropped unless keepHidden, in
// which case they stay and carry Hidden so a browser can show them dimmed.
// The libraries given are not touched — the bundled shelf is rebuilt from
// the binary each call and must not carry state between calls.
func (s *Store) Apply(libs []*Library, keepHidden bool) []*Library {
	if len(s.Overrides) == 0 {
		return libs
	}
	out := make([]*Library, 0, len(libs))
	for _, lib := range libs {
		cp := *lib
		cp.Assets = make([]Asset, 0, len(lib.Assets))
		for _, a := range lib.Assets {
			o, ok := s.Overrides[a.ID]
			if !ok {
				cp.Assets = append(cp.Assets, a)
				continue
			}
			if o.Hidden && !keepHidden {
				continue
			}
			if o.Kind != "" {
				a.Kind = o.Kind
			}
			a.Hidden = o.Hidden
			cp.Assets = append(cp.Assets, a)
		}
		out = append(out, &cp)
	}
	return out
}

// storeFile is where the catalogue lives under the data root. Its own folder
// rather than a file beside tools/, because the studio will grow more than a
// catalogue (a style memory, a per-project pick list) and they belong together.
const storeFile = "studio/libraries.json"

// Load reads the store from under root. A missing file is an empty store,
// not an error: nothing has been added yet.
func Load(root string) (*Store, error) {
	path := filepath.Join(root, filepath.FromSlash(storeFile))
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Store{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("อ่านสารบัญคลังไม่ได้ (%s): %w", path, err)
	}
	return &s, nil
}

// Save writes the store under root, whole-file and atomically, so a crash
// mid-write leaves the previous catalogue rather than half of a new one.
func (s *Store) Save(root string) error {
	path := filepath.Join(root, filepath.FromSlash(storeFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Put adds a library or replaces the one with the same id (a rescan).
func (s *Store) Put(lib *Library) {
	for i, have := range s.Libraries {
		if have.ID == lib.ID {
			s.Libraries[i] = lib
			return
		}
	}
	s.Libraries = append(s.Libraries, lib)
}

// Remove drops a library from the catalogue. The files are the user's and are
// not touched — this forgets, it does not delete.
func (s *Store) Remove(id string) bool {
	for i, have := range s.Libraries {
		if have.ID == id {
			s.Libraries = append(s.Libraries[:i], s.Libraries[i+1:]...)
			return true
		}
	}
	return false
}

// Get returns one library by id.
func (s *Store) Get(id string) (*Library, bool) {
	for _, have := range s.Libraries {
		if have.ID == id {
			return have, true
		}
	}
	return nil, false
}

// Summary is one line per kind with a count, for a tool answer or a screen.
// Empty when the shelf is.
func Summary(libs []*Library) map[Kind]int {
	out := map[Kind]int{}
	for _, lib := range libs {
		if lib == nil {
			continue
		}
		for k, n := range lib.Counts() {
			out[k] += n
		}
	}
	return out
}
