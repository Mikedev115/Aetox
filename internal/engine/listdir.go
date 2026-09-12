package engine

// The remote folder picker's two bindings (§248 phase 3). The screen's
// directory dialog opens this machine's disks; when the engine is on a
// host, the folder to open is on the host, and the only thing that can
// list it is the engine. These list — they never open: the choice comes
// back through OpenProjectPath as it always has.

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirEntry is one folder in a listing.
type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// Hidden is a dot-folder: shown greyed, still openable.
	Hidden bool `json:"hidden"`
}

// DirListing is a folder and the folders in it.
type DirListing struct {
	Path    string     `json:"path"`
	Parent  string     `json:"parent"`
	Entries []DirEntry `json:"entries"`
	// Truncated says the folder had more than dirListCap entries.
	Truncated bool `json:"truncated"`
}

// dirListCap is how many folders a listing carries: a home directory has
// dozens, a mount point can have thousands, and the picker is a list a
// person scrolls.
const dirListCap = 2000

// HomeDir is the engine's user's home on its machine — where the picker
// starts.
func (a *Engine) HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/"
	}
	return home
}

// ListDir is the folders under path (files are not listed — the picker
// picks a project folder), sorted, with the parent to go up to.
func (a *Engine) ListDir(path string) (DirListing, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "~" {
		path = a.HomeDir()
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return DirListing{}, errors.New("เปิดโฟลเดอร์นี้ไม่ได้")
	}
	if !info.IsDir() {
		return DirListing{}, errors.New("ไม่ใช่โฟลเดอร์")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return DirListing{}, errors.New("อ่านโฟลเดอร์นี้ไม่ได้")
	}
	out := DirListing{Path: path, Entries: []DirEntry{}}
	if parent := filepath.Dir(path); parent != path {
		out.Parent = parent
	}
	for _, e := range entries {
		if !e.IsDir() {
			// A symlink to a folder counts: that is how projects are laid
			// out on more than one host.
			if e.Type()&os.ModeSymlink == 0 {
				continue
			}
			st, err := os.Stat(filepath.Join(path, e.Name()))
			if err != nil || !st.IsDir() {
				continue
			}
		}
		if len(out.Entries) >= dirListCap {
			out.Truncated = true
			break
		}
		out.Entries = append(out.Entries, DirEntry{
			Name:   e.Name(),
			Path:   filepath.Join(path, e.Name()),
			Hidden: strings.HasPrefix(e.Name(), "."),
		})
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		if out.Entries[i].Hidden != out.Entries[j].Hidden {
			return !out.Entries[i].Hidden
		}
		return strings.ToLower(out.Entries[i].Name) < strings.ToLower(out.Entries[j].Name)
	})
	return out, nil
}
