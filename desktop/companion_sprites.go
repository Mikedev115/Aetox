package main

// The baked frames of the mascot, kept for the desktop body.
//
// The window bakes them (frontend lib/mascot/bake.ts: the app's own webview
// draws the mascot and hands over PNGs) and this keeps them, on disk under
// <DataRoot>/companion/<hash>/<key>.png so the next launch does not bake
// again, and in memory decoded — only the few frames on screen, because a
// set is ~170 pictures and decoding all of them would cost more than the
// WebView2 this replaces. The hash names one look at one scale from one
// version of the drawing (bake.ts setHash); frames from two hashes never
// meet, and a stale hash is just a directory nobody opens again.
//
// Keys are the window's (bake.ts keyOf) and are filename-safe by construction;
// anything else is refused rather than sanitised, so a key is always the
// file it names.

import (
	"bytes"
	"container/list"
	"encoding/base64"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
)

// CompanionFrame is one baked picture as the window hands it over: the key,
// its size in device pixels, the PNG in base64.
type CompanionFrame struct {
	Key string `json:"key"`
	W   int    `json:"w"`
	H   int    `json:"h"`
	PNG string `json:"png"`
}

var (
	companionKeyRe  = regexp.MustCompile(`^[A-Za-z0-9]+(?:-[A-Za-z0-9-]+)*$`)
	companionHashRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9]{2,4}$`)
)

// spriteStore is one hash's frames: a directory, and a bounded cache of
// decoded ones.
type spriteStore struct {
	mu    sync.Mutex
	hash  string
	dir   string
	keys  map[string]bool
	cache map[string]*list.Element
	lru   *list.List
	limit int
}

type spriteEntry struct {
	key string
	img *image.RGBA
}

// spriteCacheLimit is how many decoded frames stay in memory: the two of a
// crossfade, the neighbours of a loop, a blink, the icons — a dozen is a
// working set, sixteen leaves room.
const spriteCacheLimit = 16

// companionSpriteDir is where a hash's frames live; empty if the data root
// cannot be found, in which case frames live only in memory.
func companionSpriteDir(hash string) string {
	root, err := config.DataRoot()
	if err != nil || root == "" {
		return ""
	}
	return filepath.Join(root, "companion", hash)
}

func newSpriteStore(hash, dir string) *spriteStore {
	s := &spriteStore{hash: hash, dir: dir, keys: map[string]bool{}, cache: map[string]*list.Element{}, lru: list.New(), limit: spriteCacheLimit}
	if dir != "" {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if name, ok := cutPNG(e.Name()); ok && companionKeyRe.MatchString(name) {
					s.keys[name] = true
				}
			}
		}
	}
	return s
}

func cutPNG(name string) (string, bool) {
	const ext = ".png"
	if len(name) <= len(ext) || name[len(name)-len(ext):] != ext {
		return "", false
	}
	return name[:len(name)-len(ext)], true
}

// put keeps frames: on disk when there is a disk, and in the key set either
// way. A frame that fails to decode is refused — a picture that cannot be
// drawn is worse than a missing one, which the body asks for again.
func (s *spriteStore) put(frames []CompanionFrame) error {
	var firstErr error
	for _, f := range frames {
		if !companionKeyRe.MatchString(f.Key) {
			firstErr = errors.Join(firstErr, errors.New("companion sprite: bad key "+f.Key))
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(f.PNG)
		if err != nil {
			firstErr = errors.Join(firstErr, errors.New("companion sprite "+f.Key+": "+err.Error()))
			continue
		}
		img, err := decodeSprite(raw)
		if err != nil {
			firstErr = errors.Join(firstErr, errors.New("companion sprite "+f.Key+": "+err.Error()))
			continue
		}
		s.mu.Lock()
		if s.dir != "" {
			if err := os.MkdirAll(s.dir, 0o755); err == nil {
				if err := os.WriteFile(filepath.Join(s.dir, f.Key+".png"), raw, 0o644); err != nil {
					debuglog.Msg("companion sprite %s: not kept on disk: %v", f.Key, err)
				}
			}
		}
		s.keys[f.Key] = true
		s.remember(f.Key, img)
		s.mu.Unlock()
	}
	return firstErr
}

// have lists the keys this store can draw, sorted for a stable answer.
func (s *spriteStore) have() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.keys))
	for k := range s.keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (s *spriteStore) has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keys[key]
}

// frame is the decoded picture for a key, from the cache or from disk; nil
// if the store has no such frame.
func (s *spriteStore) frame(key string) *image.RGBA {
	s.mu.Lock()
	defer s.mu.Unlock()
	if el, ok := s.cache[key]; ok {
		s.lru.MoveToFront(el)
		return el.Value.(*spriteEntry).img
	}
	if !s.keys[key] || s.dir == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(s.dir, key+".png"))
	if err != nil {
		delete(s.keys, key)
		return nil
	}
	img, err := decodeSprite(raw)
	if err != nil {
		debuglog.Msg("companion sprite %s on disk is unreadable: %v", key, err)
		delete(s.keys, key)
		return nil
	}
	s.remember(key, img)
	return img
}

// remember caches a decoded frame, evicting the least recently drawn past
// the limit. Caller holds mu.
func (s *spriteStore) remember(key string, img *image.RGBA) {
	if el, ok := s.cache[key]; ok {
		el.Value.(*spriteEntry).img = img
		s.lru.MoveToFront(el)
		return
	}
	s.cache[key] = s.lru.PushFront(&spriteEntry{key: key, img: img})
	for s.lru.Len() > s.limit {
		last := s.lru.Back()
		s.lru.Remove(last)
		delete(s.cache, last.Value.(*spriteEntry).key)
	}
}

func (s *spriteStore) cached() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lru.Len()
}

// decodeSprite is a PNG as the compositor wants it: RGBA, premultiplied —
// image.RGBA's own convention, and UpdateLayeredWindow's.
func decodeSprite(raw []byte) (*image.RGBA, error) {
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba, nil
	}
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst, nil
}

// ---- the bindings ---------------------------------------------------------

// sprites is the store for a hash, made on first mention; a new hash (the
// look changed, the monitor's scale changed) replaces the old one, whose
// files stay on disk for the day the user switches back.
func (c *companionServer) sprites(hash string) *spriteStore {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.store != nil && c.store.hash == hash {
		return c.store
	}
	if !companionHashRe.MatchString(hash) {
		return nil
	}
	c.store = newSpriteStore(hash, companionSpriteDir(hash))
	return c.store
}

// CompanionSpriteKeys is what is already baked for a set, so the window
// bakes only what is missing. An unknown hash is an empty set, not an error.
func (a *App) CompanionSpriteKeys(hash string) []string {
	s := a.companion().sprites(hash)
	if s == nil {
		return []string{}
	}
	return s.have()
}

// CompanionSprites takes a batch of baked frames for a set. The error names
// the first frame that could not be kept; the rest of the batch is kept.
func (a *App) CompanionSprites(hash string, frames []CompanionFrame) error {
	s := a.companion().sprites(hash)
	if s == nil {
		return errors.New("companion sprites: bad set hash")
	}
	return s.put(frames)
}
