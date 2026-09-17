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
// Today both halves are methods on one Engine in one process, and the split is
// invisible from the frontend, which keeps calling the names below. It exists
// so the engine half can move out of this package without dragging a window
// with it — and so that when the engine is on another machine, what the user
// picks here is understood to be a path THERE only where that is true
// (a folder to browse, a project to add), and to be bytes moving between two
// machines everywhere else (an export, an attachment).
//
// With the engine on a host (§248 phase 4, host_files.go) the halves meet
// differently: a file picked here goes up the wire before its binding is
// called (onHost), a folder is picked THERE through the window's remote
// picker (pickHostDir), and a reveal — the one thing this file cannot do
// across two machines — answers with a named error carrying the path, not
// a button that vanishes. A file "opened with its program" is the exception:
// a copy is fetched and that is opened, and said to be a copy.

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// ---------------------------------------------------------------- dialogs

// ExportAgentPackage packs one worker and asks where to save the .zip.
// Returns the summary of what travelled, "" when the picker was dismissed —
// cancelling is not a failure and must not raise one, the same contract
// InstallSkillFromZip keeps.
func (a *App) ExportAgentPackage(name string) (string, error) {
	// Packed before the dialog opens: an agent that cannot be exported should
	// refuse while the user is still looking at the button they pressed.
	file, err := a.api.AgentPackageBytes(name)
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
	file, err := a.api.SessionExportBytes(id, format)
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
	file, err := a.api.PictureBytes(relPath)
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
	return a.withHostFile(path, 0, a.api.ImportSessionFrom)
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
	return a.withHostFile(path, 0, func(p string) (string, error) { return a.api.SetPresetImageFrom(name, p) })
}

// PickSpaceImage opens the native picker for the picture shown on a project
// card and header. The bytes live with the project on the engine's machine, so
// a remote engine takes the same host-file trip as attachments and context.
func (a *App) PickSpaceImage(name string) (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "เลือกรูปโปรเจกต์",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Images (*.png, *.jpg, *.jpeg, *.webp, *.gif, *.bmp)", Pattern: "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.bmp"},
		},
	})
	if err != nil || strings.TrimSpace(path) == "" {
		return "", err
	}
	return a.withHostFile(path, 4<<20, func(p string) (string, error) { return a.api.SetSpaceImageFrom(name, p) })
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
	return a.withHostFile(path, 0, a.api.InstallSkillsFromZipAt)
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
	return a.AddSpaceContextFiles(name, picked) // the screen's: the trip up, on a host
}

// AddWorkspaceFolder asks the user for a folder and gives it the same rights
// the project folder has, for this project, until they remove it.
func (a *App) AddWorkspaceFolder() ([]engine.WorkspaceFolder, error) {
	if !a.api.GetProjectStatus().Focused {
		// Refused before the dialog, with the reason AddWorkspaceFolderAt gives.
		return a.api.AddWorkspaceFolderAt("")
	}
	dir, err := a.pickHostDir("เพิ่มโฟลเดอร์เข้าโปรเจกต์นี้", "")
	if err != nil {
		return a.api.WorkspaceFolders(), err
	}
	if strings.TrimSpace(dir) == "" {
		return a.api.WorkspaceFolders(), nil // cancelled
	}
	return a.api.AddWorkspaceFolderAt(dir)
}

// OpenProjectFolder lets the user pick a real folder via the native OS dialog,
// then opens a chat in it (OpenProjectPath). A dismissed dialog answers with
// the project as it stands.
func (a *App) OpenProjectFolder() (engine.ProjectStatus, error) {
	dir, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Open Aetox Project Folder",
	})
	if err != nil {
		return engine.ProjectStatus{}, err
	}
	if strings.TrimSpace(dir) == "" {
		return a.api.GetProjectStatus(), nil
	}
	return a.api.OpenProjectPath(dir)
}

// PickCodeProjectsDir asks where the coding desk's new projects should go
// from now on and remembers the answer (engine.SetCodeProjectsDir). Answers
// with the folder now in force — the one chosen, or the one already set when
// the dialog was dismissed.
func (a *App) PickCodeProjectsDir() (string, error) {
	dir, err := a.pickHostDir("โฟลเดอร์ที่จะเก็บโปรเจกต์ใหม่", a.api.CodeProjectsDir())
	if err != nil {
		return a.api.CodeProjectsDir(), err
	}
	if strings.TrimSpace(dir) == "" {
		return a.api.CodeProjectsDir(), nil // cancelled
	}
	if err := a.api.SetCodeProjectsDir(dir); err != nil {
		return a.api.CodeProjectsDir(), err
	}
	return a.api.CodeProjectsDir(), nil
}

// BrowseFolder asks for a folder and points the file tree at it. Returns the
// folder chosen, or what the tree was already showing when the dialog was
// dismissed.
func (a *App) BrowseFolder() (string, error) {
	if a.api.GetProjectStatus().Focused {
		// Refused before the dialog, with the reason BrowseFolderAt gives.
		return a.api.BrowseFolderAt("")
	}
	dir, err := a.pickHostDir("Browse a folder", "")
	if err != nil {
		return "", err
	}
	return a.api.BrowseFolderAt(dir)
}

// AddStudioLibrary asks the user which folder holds their video material and
// starts cataloguing it (studio_library.go). Returns false when the dialog was
// dismissed or a scan is already running — cancelling is not a failure.
func (a *App) AddStudioLibrary() (bool, error) {
	dir, err := a.pickHostDir("เพิ่มโฟลเดอร์วัตถุดิบเข้าคลังสตูดิโอ", "")
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
// (or its default program, for a file): one implementation, on every platform.
// With the engine on a host the path is there and the file manager is here,
// and the door says so (errOnHost) — every caller shows the error it gets.
func (a *App) reveal(path string, err error) error {
	if err != nil {
		return err
	}
	if a.engineOnHost() {
		return a.errOnHost(path)
	}
	return a.revealInFileManager(path)
}

// revealInFileManager is every "open this in the file manager" button's last
// step, and the one place the OS door is opened. Routed through the App so a
// test can watch it happen without a window appearing on somebody's desk — see
// App.openDir.
func (a *App) revealInFileManager(path string) error {
	if a.openDir != nil {
		return a.openDir(path)
	}
	return openInFileManager(path)
}

// openInFileManager reveals a directory in the OS file manager. The one place
// every "open folder" button in the app goes through.
//
// Deliberately NOT wrapped in proc.HideConsole. That helper sets HideWindow and
// CREATE_NO_WINDOW so a background console process (git, a shell) does not flash
// a black box — but explorer.exe is a GUI program whose window is the entire
// point, and those flags suppress it. Every folder button in the app was hiding
// the window it had just asked for, which reads as the button doing nothing.
//
// explorer.exe also exits non-zero on success, so Start() (not Run()) is what
// this wants regardless: launch it and stop caring.
func openInFileManager(dir string) error {
	// proc-show-window: launching a GUI program — see the comment above and
	// TestEveryExecSiteHidesTheConsole. HideConsole here would hide the very
	// window this function exists to open.
	// proc-detached: the file manager belongs to the user, not to this call.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

// RevealStudioAsset opens the folder holding one shelf asset, with the file
// selected where the platform allows; RevealStudioLibrary opens a shelf's
// folder. Both are the screen's half of a pair: the engine says where
// (StudioAssetPath, StudioLibraryPath), this machine's file manager shows it.
func (a *App) RevealStudioAsset(id string) error {
	return a.reveal(a.api.StudioAssetPath(id))
}

func (a *App) RevealStudioLibrary(id string) error {
	return a.reveal(a.api.StudioLibraryPath(id))
}

// RevealSpeechModel shows the folder a scanned speech model sits in.
func (a *App) RevealSpeechModel(path string) error {
	return a.reveal(a.api.SpeechModelFolderPath(path))
}

// OpenSpeechModelDir opens one of the scanned model folders, creating Aetox's
// own if it does not exist yet — that is where a downloaded model is meant to go.
func (a *App) OpenSpeechModelDir(dir string) error { return a.reveal(a.api.SpeechModelDirPath(dir)) }

// OpenExport opens a file this session exported, with whatever the OS uses —
// the same door the file card uses, so an exported file opens exactly the way
// every other produced file in this app does.
func (a *App) OpenExport(path string) error { return a.reveal(a.exportPath(path)) }

// OpenFileExternally opens a file of the open project with its default
// program. On a host the file is fetched first and a copy is what opens
// (openHostFileCopy) — the one reveal that can cross the wire, because a
// file, unlike a folder, can be carried.
func (a *App) OpenFileExternally(relPath string) error {
	if a.engineOnHost() {
		if _, err := a.api.ProjectFilePath(relPath); err != nil {
			return err // the engine's own refusal — outside the sandbox, not there
		}
		return a.openHostFileCopy(relPath)
	}
	return a.reveal(a.api.ProjectFilePath(relPath))
}

// OpenArtifact opens a produced file from the gallery. Separate from
// OpenFileExternally, which takes a path relative to the *open project's*
// sandbox root: an artifact is absolute and routinely belongs to another
// project's output folder — ArtifactPath bounds it by the gallery's own roots.
func (a *App) OpenArtifact(path string) error {
	return a.reveal(a.api.ArtifactPath(path))
}

// OpenMCPFolder reveals the folder holding mcp-servers.json.
func (a *App) OpenMCPFolder() error { return a.reveal(a.api.MCPFolderPath()) }

// OpenMemoryFolder reveals the memory directory.
func (a *App) OpenMemoryFolder() error { return a.reveal(a.api.MemoryFolderPath()) }

// OpenPromptsFolder reveals the prompts directory.
func (a *App) OpenPromptsFolder() error { return a.reveal(a.api.PromptsFolderPath()) }

// OpenSkillsFolder reveals the skills directory.
func (a *App) OpenSkillsFolder() error { return a.reveal(a.api.SkillsFolderPath()) }

// OpenSpaceFolder shows a project's folder — the answer to "where do I put
// the files?", given rather than described.
func (a *App) OpenSpaceFolder(name string) error { return a.reveal(a.api.SpaceFolderPath(name)) }

// OpenSubagentsFolder reveals the sub-agents' home.
func (a *App) OpenSubagentsFolder() error { return a.reveal(a.api.SubagentsFolderPath()) }

// OpenAgentsFolder reveals the agents' home — the office page's hiring door.
func (a *App) OpenAgentsFolder() error { return a.reveal(a.api.AgentsFolderPath()) }

// OpenTeamsFolder reveals the teams' home — where a team made by hand goes.
func (a *App) OpenTeamsFolder() error { return a.reveal(a.api.TeamsFolderPath()) }

// OpenAgentSkillsFolder reveals one agent's own skills shelf.
func (a *App) OpenAgentSkillsFolder(name string) error {
	return a.reveal(a.api.AgentSkillsFolderPath(name))
}

// OpenAgentHome reveals one agent's home directory.
func (a *App) OpenAgentHome(name string) error { return a.reveal(a.api.AgentHomePath(name)) }
