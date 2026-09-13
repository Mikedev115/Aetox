package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mikedev115/Aetox/internal/config"
)

// A project the coding desk makes from the sidebar (owner, 14 ก.ย. 2026): press
// + on the projects heading, type a name, and a real folder exists and a chat
// is open in it.
//
// Not a Space. The assistant desk's projects (spaces.go) live inside the app's
// data folder, hidden, and carry a context/ subfolder the assistant reads —
// that is a folder of conversations. A coding project is the user's own
// source tree: it has to be somewhere they can open in an editor, clone into,
// and find in Explorer, so it goes under a visible folder in their home and
// nothing is put inside it. Two kinds, two places, and the owner asked for
// exactly that separation — คนละชั้นคนละตำแหน่ง — so they cannot be confused
// for one another.

// codeProjectsDirName is the folder under the home directory that holds these
// when the user has not chosen another (config.ModelPreference.CodeProjectsDir).
const codeProjectsDirName = "aetox-projects"

// CodeProjectsDir answers where the next created project will go: the folder
// the user chose, else <home>/aetox-projects. Never created here — the
// folder appears when the first project does.
func (a *Engine) CodeProjectsDir() string {
	return codeProjectsDir()
}

func codeProjectsDir() string {
	if pref, ok, err := config.LoadModelPreference(); err == nil && ok {
		if dir := strings.TrimSpace(pref.CodeProjectsDir); dir != "" {
			return filepath.Clean(dir)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return codeProjectsDirName
	}
	return filepath.Join(home, codeProjectsDirName)
}

// SetCodeProjectsDir remembers where created projects go from now on. Empty
// puts the default back. Projects already made stay where they are — this is
// the parent of the next one, not a move.
func (a *Engine) SetCodeProjectsDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir != "" {
		if !filepath.IsAbs(dir) {
			return fmt.Errorf("ต้องเป็นตำแหน่งเต็ม เช่น C:\\Users\\คุณ\\projects")
		}
		dir = filepath.Clean(dir)
	}
	return config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.CodeProjectsDir = dir
		return nil
	})
}

// CreateCodeProject makes <CodeProjectsDir>/<name> and answers with the path,
// which the screen then opens as a project (OpenProjectPath) — creating and
// opening are two calls because opening is a session switch the screen has to
// arrive at (cockpit.openProject), and a door that did both would put the
// switch behind a folder write it cannot see.
//
// Same name rules as a Space (spaceFolderName): the name becomes a folder on
// this machine, and the characters a folder cannot carry are the same on both
// desks. A name already taken is refused rather than reused — the user typed a
// name for something new, and opening whatever happened to be there would be
// a quiet substitution.
func (a *Engine) CreateCodeProject(name string) (string, error) {
	folder, err := spaceFolderName(name)
	if err != nil {
		return "", err
	}
	root := codeProjectsDir()
	path := filepath.Join(root, folder)
	if filepath.Dir(path) != filepath.Clean(root) {
		return "", fmt.Errorf("ชื่อโปรเจกต์ไม่ถูกต้อง")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return "", fmt.Errorf("มีโฟลเดอร์ชื่อนี้อยู่แล้วที่ %s", root)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("สร้างโฟลเดอร์ไม่ได้: %w", err)
	}
	return path, nil
}
