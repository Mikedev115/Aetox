package main

// Looking at a folder, without moving in.
//
// The ไฟล์ tab shows nothing when no project is focused — ProjectTree refuses to
// walk a home directory — and its empty state offered one button: เปิดโฟลเดอร์.
// That button is `OpenProjectFolder`, the โค้ด door's act, and it does three
// things at once: marks the app focused, retargets the template every new chat
// is born into, and writes the folder into the recent-projects table for good.
//
// On the ผู้ช่วย door all three are wrong, and the third is the one the owner
// saw (7 ก.ย.): he picked a folder to look at and the app remembered it. The
// first is worse and says nothing on screen — the assistant's reach is
// `OpenSandbox: !projectFocused`, so a click meant as "show me this folder"
// narrowed a chat that roams the machine down to Downloads. The door's own spec
// says the opposite: *"ขอบเขต: ไม่ผูกโปรเจกต์ — ทั้งเครื่อง"* (DOOR-ASSISTANT.md).
//
// So browsing is its own act, and it is deliberately smaller than opening a
// project. It moves one string in memory. Nothing here touches the engine, the
// store or the session — which is what makes "ไม่จำ" true by construction
// rather than by everyone remembering not to persist it.

import (
	"fmt"
	"os"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// browseNodeCap is how many rows a browsed folder may produce. Past it the walk
// stops descending: four thousand rows is far more tree than anyone reads, and
// the folder somebody picks to look at can be their whole drive.
const browseNodeCap = 4000

// BrowseFolder asks for a folder and points the file tree at it. Returns the
// folder chosen, or "" when the dialog was dismissed.
//
// Refused while a project is focused: there the tree is showing the project,
// and a second root would be two answers to "what am I looking at". The door
// out of a project already exists (ClearProjectFocus) and is a deliberate act.
func (a *App) BrowseFolder() (string, error) {
	if a.projectFocused {
		return "", fmt.Errorf("มีโปรเจกต์เปิดอยู่แล้ว")
	}
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Browse a folder",
	})
	if err != nil {
		return "", err
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return a.browseRoot, nil // dismissed: keep looking at whatever it was
	}
	info, statErr := os.Stat(dir)
	if statErr != nil || !info.IsDir() {
		return "", fmt.Errorf("เปิดโฟลเดอร์นี้ไม่ได้")
	}
	a.browseRoot = dir
	return a.browseRoot, nil
}

// BrowseRoot is the folder the tree is looking at, or "" for none. The window
// asks on every workspace refresh, because it is the one thing about the tree
// that the tree's own rows do not say.
func (a *App) BrowseRoot() string {
	if a.projectFocused {
		return ""
	}
	return a.browseRoot
}

// StopBrowsing puts the tree back to empty.
func (a *App) StopBrowsing() { a.browseRoot = "" }

// takeProject marks the app as focused on a project, and drops the folder the
// tree was merely looking at.
//
// One call for the two lines because they are one fact: the tree's temporary
// root belongs to the mode that has no project, and carrying it across a focus
// switch is the "remembering" this file exists to prevent — one step quieter.
func (a *App) takeProject() {
	a.projectFocused = true
	a.browseRoot = ""
}
