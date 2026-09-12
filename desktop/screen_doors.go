package main

// The doors that need a window: every native dialog the app opens and every
// "show it in the file manager" button, in one file (§248 A5).
//
// Each one is two halves. The half here asks the user something the screen's
// machine can answer — which folder, which file, where to save — or shows them
// something on that machine. The other half does the work, and lives with the
// engine: it takes a path, or hands back bytes, and never opens a dialog. The
// rule that split them: **dialog → a path on the engine's host → an engine
// binding**, and for exports **engine bytes → a dialog on the screen**.
//
// Today both halves are methods on one App in one process, and the split is
// invisible from the frontend, which keeps calling the names below. It exists
// so the engine half can move out of this package without dragging a window
// with it — and so that when the engine is on another machine, what the user
// picks here is understood to be a path THERE only where that is true
// (a folder to browse, a project to add), and to be bytes moving between two
// machines everywhere else (an export, an attachment).
//
// The reveal doors stay, in remote mode, the one thing this file cannot do: a
// folder on the engine's host cannot be opened in this machine's file manager.
// That will be a named error shown as it is, not a button that vanishes.

import (
	"os"
	"path/filepath"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------- dialogs

// ExportAgentPackage packs one worker and asks where to save the .zip.
// Returns the summary of what travelled, "" when the picker was dismissed —
// cancelling is not a failure and must not raise one, the same contract
// InstallSkillFromZip keeps.
func (a *App) ExportAgentPackage(name string) (string, error) {
	// Packed before the dialog opens: an agent that cannot be exported should
	// refuse while the user is still looking at the button they pressed.
	file, err := a.AgentPackageBytes(name)
	if err != nil {
		return "", err
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "ส่งออกเอเจน",
		DefaultFilename: file.Name,
		Filters:         []wailsruntime.FileFilter{{DisplayName: "Agent package (*.zip)", Pattern: "*.zip"}},
	})
	if err != nil || strings.TrimSpace(path) == "" {
		return "", err
	}
	if err := os.WriteFile(path, file.Data, 0o644); err != nil {
		return "", err
	}
	return file.Note, nil
}

// ExportSession asks where to save and writes one session there. Returns the
// path written, "" when the user closed the dialog — a cancel is a decision,
// not an error to report.
func (a *App) ExportSession(id, format string) (string, error) {
	// Rendered before the dialog opens: a session that cannot be exported
	// should refuse before asking where to put it.
	file, err := a.SessionExportBytes(id, format)
	if err != nil {
		return "", err
	}
	display := "Aetox chat (*.json)"
	if format == "markdown" {
		display = "Markdown (*.md)"
	}
	ext := filepath.Ext(file.Name)
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "ส่งออกบทสนทนา",
		DefaultFilename: file.Name,
		Filters:         []wailsruntime.FileFilter{{DisplayName: display, Pattern: "*" + ext}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, file.Data, 0o644)
}

// SavePicture asks where to save a picture the agent made and writes it
// there, byte for byte — see PictureBytes for why not through a canvas.
func (a *App) SavePicture(relPath string) (string, error) {
	file, err := a.PictureBytes(relPath)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(file.Name))
	if ext == "" {
		ext = ".png"
	}
	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "บันทึกภาพ",
		DefaultFilename: file.Name,
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "รูปภาพ (*" + ext + ")",
			Pattern:     "*" + ext,
		}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, file.Data, 0o644)
}

// ImportSession asks for an exported .json file and brings it in as a new
// session in the current project. Returns the new session's id, "" when the
// user closed the dialog.
func (a *App) ImportSession() (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "นำเข้าบทสนทนา",
		Filters: []wailsruntime.FileFilter{{DisplayName: "Aetox chat (*.json)", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	return a.ImportSessionFrom(path)
}

// PickPresetImage opens the native picker and, if the user chose a file,
// copies it in as that preset's cover. Returns the cover as a data URI so the
// card updates without re-reading the whole list.
func (a *App) PickPresetImage(name string) (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "เลือกรูปหน้าปกชุดคำสั่ง",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Images (*.png, *.jpg, *.jpeg, *.webp, *.gif, *.bmp)", Pattern: "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp"},
		},
	})
	if err != nil || strings.TrimSpace(path) == "" {
		return "", err
	}
	return a.SetPresetImageFrom(name, path)
}

// InstallSkillFromZip asks for a skill archive and installs it.
//
// The third road in, and the one that covers everything the other two do not: a
// skill that arrived by email, by download, as a release asset, or from someone
// who does not publish it on GitHub at all.
//
// Returns "" with no error when the picker was dismissed — cancelling is not a
// failure and must not raise one.
func (a *App) InstallSkillFromZip() (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "เลือกไฟล์ zip ของสกิล",
		Filters: []wailsruntime.FileFilter{{DisplayName: "Skill archive (*.zip)", Pattern: "*.zip"}},
	})
	if err != nil || strings.TrimSpace(path) == "" {
		return "", err
	}
	return a.InstallSkillsFromZipAt(path)
}

// AddSpaceContext asks for files and copies them into a project's context
// folder — see AddSpaceContextFiles for what that folder is.
func (a *App) AddSpaceContext(name string) ([]string, error) {
	picked, err := wailsruntime.OpenMultipleFilesDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "เพิ่มไฟล์บริบทของโปรเจกต์",
	})
	if err != nil {
		return nil, err
	}
	return a.AddSpaceContextFiles(name, picked)
}

// AddWorkspaceFolder asks the user for a folder and gives it the same rights
// the project folder has, for this project, until they remove it.
func (a *App) AddWorkspaceFolder() ([]WorkspaceFolder, error) {
	if !a.projectFocused {
		// Refused before the dialog, with the reason AddWorkspaceFolderAt gives.
		return a.AddWorkspaceFolderAt("")
	}
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "เพิ่มโฟลเดอร์เข้าโปรเจกต์นี้",
	})
	if err != nil {
		return a.WorkspaceFolders(), err
	}
	if strings.TrimSpace(dir) == "" {
		return a.WorkspaceFolders(), nil // cancelled
	}
	return a.AddWorkspaceFolderAt(dir)
}

// BrowseFolder asks for a folder and points the file tree at it. Returns the
// folder chosen, or what the tree was already showing when the dialog was
// dismissed.
func (a *App) BrowseFolder() (string, error) {
	if a.projectFocused {
		// Refused before the dialog, with the reason BrowseFolderAt gives.
		return a.BrowseFolderAt("")
	}
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Browse a folder",
	})
	if err != nil {
		return "", err
	}
	return a.BrowseFolderAt(dir)
}

// AddStudioLibrary asks the user which folder holds their video material and
// starts cataloguing it (studio_library.go). Returns false when the dialog was
// dismissed or a scan is already running — cancelling is not a failure.
func (a *App) AddStudioLibrary() (bool, error) {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "เพิ่มโฟลเดอร์วัตถุดิบเข้าคลังสตูดิโอ",
	})
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(dir) == "" {
		return false, nil // cancelled
	}
	return a.AddStudioLibraryAt(dir)
}

// ---------------------------------------------------------------- reveals

// reveal opens a path the engine answered with in this machine's file manager
// (or its default program, for a file): one implementation, in speech.go, on
// every platform.
func (a *App) reveal(path string, err error) error {
	if err != nil {
		return err
	}
	return a.revealInFileManager(path)
}

// OpenFileExternally opens a file of the open project with its default program.
func (a *App) OpenFileExternally(relPath string) error {
	return a.reveal(a.ProjectFilePath(relPath))
}

// OpenArtifact opens a produced file from the gallery. Separate from
// OpenFileExternally, which takes a path relative to the *open project's*
// sandbox root: an artifact is absolute and routinely belongs to another
// project's output folder — ArtifactPath bounds it by the gallery's own roots.
func (a *App) OpenArtifact(path string) error {
	return a.reveal(a.ArtifactPath(path))
}

// OpenMCPFolder reveals the folder holding mcp-servers.json.
func (a *App) OpenMCPFolder() error { return a.reveal(a.MCPFolderPath()) }

// OpenMemoryFolder reveals the memory directory.
func (a *App) OpenMemoryFolder() error { return a.reveal(a.MemoryFolderPath()) }

// OpenPromptsFolder reveals the prompts directory.
func (a *App) OpenPromptsFolder() error { return a.reveal(a.PromptsFolderPath()) }

// OpenSkillsFolder reveals the skills directory.
func (a *App) OpenSkillsFolder() error { return a.reveal(a.SkillsFolderPath()) }

// OpenSpaceFolder shows a project's folder — the answer to "where do I put
// the files?", given rather than described.
func (a *App) OpenSpaceFolder(name string) error { return a.reveal(a.SpaceFolderPath(name)) }

// OpenSubagentsFolder reveals the sub-agents' home.
func (a *App) OpenSubagentsFolder() error { return a.reveal(a.SubagentsFolderPath()) }

// OpenAgentsFolder reveals the agents' home — the office page's hiring door.
func (a *App) OpenAgentsFolder() error { return a.reveal(a.AgentsFolderPath()) }

// OpenAgentSkillsFolder reveals one agent's own skills shelf.
func (a *App) OpenAgentSkillsFolder(name string) error {
	return a.reveal(a.AgentSkillsFolderPath(name))
}

// OpenAgentHome reveals one agent's home directory.
func (a *App) OpenAgentHome(name string) error { return a.reveal(a.AgentHomePath(name)) }
