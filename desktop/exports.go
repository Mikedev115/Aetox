package main

// Where an export lands (§248 B1): the machine's Downloads folder, which is
// this machine's — the screen's — and never the engine's host. The engine
// decides what an export contains (engine.DeckExportFiles); this file decides
// where it goes, remembers what it wrote so OpenExport can open it, and
// nothing else.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// exports is every file the deck export wrote this session, and the test
// seam for where they go.
type exports struct {
	mu   sync.Mutex
	seen map[string]bool
	// root overrides Downloads. Test seam, and it earns its keep: without it
	// every run of the export tests would drop files into the developer's
	// real Downloads folder.
	root string
}

// ExportDeck writes a deck out in another format and answers with where it
// landed: <Downloads>/<name><ext>, or a folder of pictures under the deck's
// name. See engine.DeckExportFiles for what each format is.
func (a *App) ExportDeck(relPath, format string) (string, error) {
	export, err := a.api.DeckExportFiles(relPath, format)
	if err != nil {
		return "", err
	}
	target, err := a.freeDownloadPath(export.Base, export.Ext)
	if err != nil {
		return "", err
	}
	if export.Folder {
		dir := strings.TrimSuffix(target, export.Ext) // <name>.png -> <name>, a folder
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		for _, f := range export.Files {
			if err := writeFileAtomically(filepath.Join(dir, f.Name), f.Data); err != nil {
				return "", err
			}
		}
		target = dir
	} else {
		if len(export.Files) != 1 {
			return "", fmt.Errorf("การส่งออกได้ %d ไฟล์ แต่รอไฟล์เดียว", len(export.Files))
		}
		if err := writeFileAtomically(target, export.Files[0].Data); err != nil {
			return "", err
		}
	}
	a.rememberExport(target)
	return target, nil
}

// writeFileAtomically writes through a temp file in the same directory and
// renames, the way ooxml.WriteFile does. A half-written export that keeps the
// name of the last good one is worse than no export: it opens, it is wrong, and
// nothing says when it went wrong.
func writeFileAtomically(target string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".aetox-export-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, target); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// exportsDir is where an export lands: the machine's Downloads folder.
//
// Falls back to the home folder, then to the project, rather than failing — a
// machine with no Downloads is unusual but the export is still worth having, and
// the answer names wherever it actually went.
func (a *App) exportsDir() (string, error) {
	// Test seam, and it earns its keep: without it every run of the export
	// tests would drop files into the developer's real Downloads folder.
	if override := strings.TrimSpace(a.exports.root); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("หาโฟลเดอร์ของผู้ใช้ไม่เจอ")
	}
	downloads := filepath.Join(home, "Downloads")
	if info, err := os.Stat(downloads); err == nil && info.IsDir() {
		return downloads, nil
	}
	return home, nil
}

// freeDownloadPath is <Downloads>/<base><ext>, with a number appended if that
// name is taken.
//
// Overwriting would be the wrong default here in a way it is not inside the
// session folder: Downloads is shared with every other program, the file there
// may be one the user already sent to somebody, and an export that quietly
// replaced it would destroy something this app never made. Windows numbers
// duplicate downloads the same way.
func (a *App) freeDownloadPath(base, ext string) (string, error) {
	dir, err := a.exportsDir()
	if err != nil {
		return "", err
	}
	base = engine.SanitiseFileName(base)
	candidate := filepath.Join(dir, base+ext)
	for n := 2; n < 1000; n++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, n, ext))
	}
	return "", fmt.Errorf("มีไฟล์ชื่อนี้อยู่แล้วเป็นร้อยไฟล์ ลองเปลี่ยนชื่อเด็คดู")
}

// rememberExport records a file this session wrote, so OpenExport can open it.
//
// An export lands in Downloads, outside the project, which every other file
// binding here refuses by design (safeSandboxPath). Widening one of those to
// "anywhere" to serve one button would hand the frontend a way to open any path
// on the machine. A set of the paths this app actually wrote is the narrow
// version of the same permission: the only thing that can be opened is a file
// the user just asked to be made.
func (a *App) rememberExport(path string) {
	a.exports.mu.Lock()
	defer a.exports.mu.Unlock()
	if a.exports.seen == nil {
		a.exports.seen = map[string]bool{}
	}
	a.exports.seen[path] = true
}

// exportPath checks that a path is a file this session exported and is still
// there — what OpenExport (screen_doors.go) opens with whatever the OS uses.
func (a *App) exportPath(path string) (string, error) {
	a.exports.mu.Lock()
	known := a.exports.seen[path]
	a.exports.mu.Unlock()
	if !known {
		// Not "permission denied": from here it is the truth. Nothing else has
		// ever been offered to open, so a path that is not in the set is not a
		// path this button ever produced.
		return "", fmt.Errorf("ไฟล์นี้ไม่ได้มาจากการส่งออกในรอบนี้")
	}
	if _, err := os.Stat(path); err != nil {
		return "", engine.ErrFileGone
	}
	return path, nil
}
