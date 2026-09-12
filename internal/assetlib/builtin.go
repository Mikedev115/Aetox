package assetlib

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// The shelf that ships inside Aetox, so the studio is never empty and the two
// video agents have a sound to reach for on the first day.
//
// Kenney's Interface Sounds (kenney.nl, 1.0, 2020): one hundred short UI
// sounds — click, select, confirmation, error, open/close, switch, tick —
// under CC0, which is the one licence that lets this file exist. Everything
// else the studio ever holds is the user's and arrives through a scan; this is
// the only material Aetox itself puts on a machine, and it is 0.9MB because
// the sounds an explainer or a product demo wants are all short.
//
// Chosen over the first candidate, a folder of seventy effects from the
// owner's 30GB Drive pack (12 ก.ย. 2569): that folder's file names traced to
// Epidemic Sound, Mixkit, Pixabay and YouTube rips, none of which may be
// redistributed. The pack the owner listened to and picked instead was this
// one; Impact Sounds (also CC0) was the runner-up and is not bundled.
//
// The catalogue travels beside the files, probed once at vendoring time by
// scratch/builtincat, so the app never runs ffprobe over its own shelf — a
// process start cost seconds on the owner's machine, and a hundred of them on
// first launch is not a first launch anyone waits through.

//go:embed builtin/interface-sounds
var builtinFS embed.FS

// builtinPacks is every shelf inside the binary. One today; the loop below
// is written for the list so a second needs one more row and no new code.
var builtinPacks = []struct {
	// dir is the folder under builtin/ in builtinFS.
	dir string
	// id is the library id, fixed rather than hashed from a path: it has to
	// survive the data root moving and never collide with a user's folder.
	id string
	// version goes into the on-disk folder name, so a newer pack unpacks
	// beside the old one instead of half over it.
	version string
	// title is what the shelf card is called; a proper noun, so not in the
	// locale files, the same call capability.Component.Title makes.
	title   string
	license string
	source  string
}{
	{dir: "interface-sounds", id: "builtin-interface-sounds", version: "1.0", title: "Kenney Interface Sounds", license: "CC0", source: "https://kenney.nl/assets/interface-sounds"},
}

// BuiltinRoot is where the bundled shelves are unpacked, under the data root.
const BuiltinRoot = "studio/builtin"

// Builtin unpacks the bundled shelves under dataRoot (once per version) and
// returns them as libraries. Never persisted in the Store — they are rebuilt
// from the binary on every call, so a newer Aetox brings a newer shelf and
// nothing on disk can go stale against it.
//
// An unpack that fails leaves that shelf out with the error returned, and
// the others still come back: a read-only data root should cost the user the
// bundled sounds, not the shelf they added themselves.
func Builtin(dataRoot string) ([]*Library, error) {
	var libs []*Library
	var firstErr error
	for _, p := range builtinPacks {
		lib, err := unpackBuiltin(dataRoot, p.dir, p.id, p.version, p.title, p.license, p.source)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("builtin %s: %w", p.dir, err)
			}
			continue
		}
		libs = append(libs, lib)
	}
	return libs, firstErr
}

func unpackBuiltin(dataRoot, dir, id, version, title, license, source string) (*Library, error) {
	src := path.Join("builtin", dir)
	dest := filepath.Join(dataRoot, filepath.FromSlash(BuiltinRoot), dir+"-"+version)
	marker := filepath.Join(dest, ".unpacked-"+version)

	if _, err := os.Stat(marker); err != nil {
		if err := fs.WalkDir(builtinFS, src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel := strings.TrimPrefix(strings.TrimPrefix(p, src), "/")
			if d.IsDir() {
				return os.MkdirAll(filepath.Join(dest, filepath.FromSlash(rel)), 0o755)
			}
			if path.Base(p) == "catalog.json" {
				return nil // read from the binary below, never from disk
			}
			data, err := builtinFS.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dest, filepath.FromSlash(rel)), data, 0o644)
		}); err != nil {
			return nil, err
		}
		if err := os.WriteFile(marker, nil, 0o644); err != nil {
			return nil, err
		}
	}

	raw, err := builtinFS.ReadFile(path.Join(src, "catalog.json"))
	if err != nil {
		return nil, err
	}
	var assets []Asset
	if err := json.Unmarshal(raw, &assets); err != nil {
		return nil, fmt.Errorf("catalog.json: %w", err)
	}
	lib := &Library{
		ID:      id,
		Root:    dest,
		Scanned: time.Time{}, // never scanned: the catalogue is authored
		Assets:  assets,
		Builtin: true,
		Title:   title,
		License: license,
		Source:  source,
	}
	for _, a := range assets {
		lib.Bytes += a.Bytes
	}
	return lib, nil
}
