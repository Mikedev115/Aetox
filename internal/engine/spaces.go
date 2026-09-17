package engine

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Mikedev115/Aetox/internal/bootstrap"
	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/mode"
)

// A โปรเจกต์ at the storefront door: a named folder that groups chats and
// carries a few files into every session held inside it (COMPANY.md §84).
//
// **`space` in code, โปรเจกต์ on screen, and the two words are not the same
// word by accident.** The workshop door already has a โปรเจกต์: a folder on the
// user's disk that the sandbox is rooted in (`projectKey`, the `projects`
// table, `Engine.projectFocused`). That one is a fence — rooting the engine in it
// is the entire point. This one is the opposite: it is a folder for
// conversations and it moves no wall, the assistant still reaches the whole
// machine. Two meanings needed two identifiers, and taking the free one for the
// new concept beats renaming a shipped schema. The mapping is written down in
// COMPANY.md §8 so the second word stays a translation instead of becoming a
// drift.
//
// **The folder is the truth.** A space exists because `<DataRoot>/project/<name>`
// exists — there is no table of spaces to fall out of step with the disk, the
// same rule ผลงาน already follows (§84). A user who makes the folder by hand has
// made a project; one who deletes it has deleted one, and nothing here has to be
// told. The only thing the database holds is which space a session was held in,
// which is a fact about the session and lives on the session's own row.
type Space struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Image        string   `json:"image"` // project picture as a data URI, empty for the generated monogram
	Path         string   `json:"path"`
	ContextPath  string   `json:"contextPath"`
	ContextFiles []string `json:"contextFiles"`
	// ContextModified is when each context file last changed, RFC3339, keyed by
	// the name in ContextFiles. A second field rather than ContextFiles turning
	// into a struct: the list is what the prompt and the tests read, and a file
	// name is the whole answer there. The date is the page's — a row of five
	// names says nothing about which one is the stale one.
	ContextModified map[string]string `json:"contextModified"`
	Chats           int               `json:"chats"`
	UpdatedAt       string            `json:"updatedAt"`
}

// contextDirName is the folder inside a space that holds what every session in
// it should start knowing. Its own folder rather than loose files at the top,
// so a space can grow other folders later without the context becoming "the
// files that happen to not be in a subfolder".
const contextDirName = "context"

// Metadata is deliberately a small hidden file inside the project's own
// folder. The folder remains the record that the project exists, while the
// sentence the owner wrote travels with that folder when it is backed up or
// moved. The picture sits beside it so neither is mistaken for assistant
// context and offered to every chat.
const (
	spaceMetaFile            = ".aetox.json"
	spaceImageBase           = ".aetox-cover"
	maxSpaceImageBytes       = 4 << 20
	maxSpaceDescriptionRunes = 240
)

var spaceImageExts = []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp"}

type spaceMetadata struct {
	Description string `json:"description,omitempty"`
}

// spacesRoot is <DataRoot>/project — the owner named this path, and it is
// deliberately beside modes/, agents/ and subagents/ rather than inside the
// database: everything the user is meant to be able to open, edit and back up
// by hand lives as files under DataRoot (ARCHITECTURE.md §14).
func spacesRoot() (string, error) {
	root, err := config.DataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "project"), nil
}

// spaceFolderName is the name the user typed, once it has been checked that a
// folder can be called that.
//
// It **refuses** rather than repairs. Stripping the characters a filesystem
// cannot hold is the tempting version and it is wrong twice over: "a/b" would
// become a folder called "ab" that the user never named and cannot find by the
// name on their screen, and "../escape" would become "..escape", a name that
// looks like it did something and did not. The promise of putting projects on
// disk is that the folder is the one they named — so a name that cannot be a
// folder is an error with the reason in it, not a name quietly turned into a
// different one.
//
// Surrounding whitespace is the exception, and only because it is invisible: a
// user who typed a trailing space did not name their project that.
func spaceFolderName(name string) (string, error) {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return "", fmt.Errorf("ตั้งชื่อโปรเจกต์ก่อนนะครับ")
	}
	if len([]rune(cleaned)) > 64 {
		return "", fmt.Errorf("ชื่อโปรเจกต์ยาวเกินไป (ไม่เกิน 64 ตัวอักษร)")
	}
	if cleaned == "." || cleaned == ".." {
		return "", fmt.Errorf("ชื่อนี้ใช้เป็นชื่อโฟลเดอร์ไม่ได้")
	}
	for _, r := range cleaned {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("ชื่อโปรเจกต์มีอักขระที่ใช้เป็นชื่อโฟลเดอร์ไม่ได้")
		}
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return "", fmt.Errorf("ชื่อโปรเจกต์ใช้ %q ไม่ได้ เพราะมันเป็นชื่อโฟลเดอร์จริงบนเครื่อง", string(r))
		}
	}
	// Windows stores neither, and stores them by silently dropping the
	// character — the folder would exist under a name that is not the one on
	// screen, which is the failure this whole function is about.
	if strings.HasSuffix(cleaned, ".") {
		return "", fmt.Errorf("ชื่อโปรเจกต์ลงท้ายด้วยจุดไม่ได้")
	}
	// CON, PRN, NUL and friends are devices on Windows, not names. Creating one
	// fails in a way that reads like a bug in Aetox rather than a name that
	// cannot be used.
	base := strings.ToUpper(cleaned)
	if dot := strings.IndexByte(base, '.'); dot >= 0 {
		base = base[:dot]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return "", fmt.Errorf("%q เป็นชื่อที่ Windows จองไว้ ตั้งเป็นชื่อโฟลเดอร์ไม่ได้", cleaned)
	}
	return cleaned, nil
}

// spacePath resolves a space's folder and refuses anything that is not directly
// inside the spaces root. The name arrives from the frontend, so this is the
// gate, not the caller.
func spacePath(name string) (string, error) {
	folder, err := spaceFolderName(name)
	if err != nil {
		return "", err
	}
	root, err := spacesRoot()
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, folder)
	if filepath.Dir(path) != filepath.Clean(root) {
		return "", fmt.Errorf("ชื่อโปรเจกต์ไม่ถูกต้อง")
	}
	return path, nil
}

// Spaces lists every โปรเจกต์, newest activity first. Reading the disk rather
// than a table is the point (see the type comment) — and it is why this answers
// correctly for a folder the user created in Explorer a second ago.
func (a *Engine) Spaces() []Space {
	out := []Space{}
	root, err := spacesRoot()
	if err != nil {
		return out
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return out // no spaces folder yet is not an error, it is zero projects
	}
	counts := a.spaceChatCounts()
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		out = append(out, a.describeSpace(filepath.Join(root, entry.Name()), entry.Name(), counts[entry.Name()]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func (a *Engine) describeSpace(path, name string, chats int) Space {
	space := Space{
		Name:            name,
		Path:            path,
		ContextPath:     filepath.Join(path, contextDirName),
		ContextFiles:    []string{},
		ContextModified: map[string]string{},
		Chats:           chats,
	}
	space.Description = readSpaceMetadata(path).Description
	space.Image = readSpaceImage(path)
	if info, err := os.Stat(path); err == nil {
		space.UpdatedAt = info.ModTime().Format(time.RFC3339)
	}
	if entries, err := os.ReadDir(space.ContextPath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			space.ContextFiles = append(space.ContextFiles, entry.Name())
			// Best effort: a file whose stat fails is still listed, just undated.
			if info, err := entry.Info(); err == nil {
				space.ContextModified[entry.Name()] = info.ModTime().Format(time.RFC3339)
			}
		}
		sort.Strings(space.ContextFiles)
	}
	return space
}

func readSpaceMetadata(path string) spaceMetadata {
	var meta spaceMetadata
	data, err := os.ReadFile(filepath.Join(path, spaceMetaFile))
	if err == nil {
		_ = json.Unmarshal(data, &meta)
	}
	meta.Description = strings.TrimSpace(meta.Description)
	return meta
}

func writeSpaceMetadata(path string, meta spaceMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(path, spaceMetaFile), data, 0o644)
}

func spaceImagePath(path string) (string, bool) {
	for _, ext := range spaceImageExts {
		candidate := filepath.Join(path, spaceImageBase+ext)
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, true
		}
	}
	return "", false
}

func readSpaceImage(path string) string {
	imagePath, ok := spaceImagePath(path)
	if !ok {
		return ""
	}
	info, err := os.Stat(imagePath)
	if err != nil || info.Size() > maxSpaceImageBytes {
		return ""
	}
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return ""
	}
	mimeType := mime.TypeByExtension(filepath.Ext(imagePath))
	if mimeType == "" {
		mimeType = "image/png"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func touchSpaceFolder(path string) {
	now := time.Now()
	_ = os.Chtimes(path, now, now)
}

// UpdateSpaceDescription changes the one user-owned sentence shown on the
// project card and header. It does not rename the folder or touch its chats.
func (a *Engine) UpdateSpaceDescription(name, description string) (Space, error) {
	path, err := spacePath(name)
	if err != nil {
		return Space{}, err
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return Space{}, fmt.Errorf("ยังไม่มีโปรเจกต์ชื่อนี้")
	}
	description = strings.TrimSpace(description)
	if len([]rune(description)) > maxSpaceDescriptionRunes {
		return Space{}, fmt.Errorf("คำอธิบายยาวเกินไป (ไม่เกิน %d ตัวอักษร)", maxSpaceDescriptionRunes)
	}
	if err := writeSpaceMetadata(path, spaceMetadata{Description: description}); err != nil {
		return Space{}, err
	}
	touchSpaceFolder(path)
	return a.describeSpace(path, filepath.Base(path), a.spaceChatCounts()[filepath.Base(path)]), nil
}

// SetSpaceImageFrom copies a square project picture into the project's own
// folder and returns the data URI the UI can paint immediately. The original
// is never changed, and replacing a picture removes the previous format so a
// stale .jpg cannot win over a newer .png on the next read.
func (a *Engine) SetSpaceImageFrom(name, sourcePath string) (string, error) {
	path, err := spacePath(name)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return "", fmt.Errorf("ยังไม่มีโปรเจกต์ชื่อนี้")
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(sourcePath)))
	allowed := false
	for _, candidate := range spaceImageExts {
		if ext == candidate {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("ไฟล์รูปต้องเป็นนามสกุล %s", strings.Join(spaceImageExts, " "))
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	if len(data) > maxSpaceImageBytes {
		return "", fmt.Errorf("รูปใหญ่เกิน %d MB — ย่อก่อนแล้วลองใหม่", maxSpaceImageBytes>>20)
	}
	target := filepath.Join(path, spaceImageBase+ext)
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	// Write first so a failed replacement leaves the previous picture intact.
	// Once the new bytes are safely present, remove every other supported
	// extension; otherwise the deterministic reader could find an older format.
	for _, candidateExt := range spaceImageExts {
		candidate := filepath.Join(path, spaceImageBase+candidateExt)
		if candidate == target {
			continue
		}
		if err := os.Remove(candidate); err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	touchSpaceFolder(path)
	return readSpaceImage(path), nil
}

// RemoveSpaceImage restores the generated colour-and-monogram fallback.
func (a *Engine) RemoveSpaceImage(name string) error {
	path, err := spacePath(name)
	if err != nil {
		return err
	}
	if imagePath, ok := spaceImagePath(path); ok {
		if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	touchSpaceFolder(path)
	return nil
}

// spaceChatCounts is one query for the whole page rather than one per row.
func (a *Engine) spaceChatCounts() map[string]int {
	counts := map[string]int{}
	db, err := a.database()
	if err != nil {
		return counts
	}
	_ = eachRow(db, "spaces: chat counts",
		`SELECT space, COUNT(*) FROM sessions WHERE space <> '' GROUP BY space`, nil,
		func(rows *sql.Rows) error {
			var name string
			var n int
			if err := rows.Scan(&name, &n); err != nil {
				return err
			}
			counts[name] = n
			return nil
		})
	return counts
}

// CreateSpace makes the folder and the context folder inside it.
//
// Both, in one call, on purpose: a space whose context folder appears only once
// something is put in it is a space where the user has to be told where to put
// things. The folder being there *is* the instruction.
func (a *Engine) CreateSpace(name string) (Space, error) {
	path, err := spacePath(name)
	if err != nil {
		return Space{}, err
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return Space{}, fmt.Errorf("มีโปรเจกต์ชื่อนี้อยู่แล้ว")
	}
	if err := os.MkdirAll(filepath.Join(path, contextDirName), 0o755); err != nil {
		return Space{}, err
	}
	return a.describeSpace(path, filepath.Base(path), 0), nil
}

// DeleteSpace removes a โปรเจกต์ by removing its folder — the folder is the
// only record that the project exists, so there is nothing else to clean up
// (see the type comment).
//
// What it takes with it is the context folder: those are copies Aetox made when
// the files were added, so the originals the user picked from are untouched,
// and the page says so before it asks.
//
// What it deliberately does NOT take is the chats. A session's row holds the
// name of the space it was held in, and a name that no longer resolves already
// means "held outside every project" (resolvedSpace) — so the conversations
// stay readable in the sidebar and simply leave the room with it. Deleting a
// folder of context must not be a way to lose work; ผลงาน and the chat history
// are the only places anything dies (COMPANY.md §6.7).
//
// A project that is already gone is a success: the user asked for it not to be
// there, and it is not there.
func (a *Engine) DeleteSpace(name string) error {
	path, err := spacePath(name)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("ชื่อนี้ไม่ใช่โฟลเดอร์ของโปรเจกต์")
	}
	return os.RemoveAll(path)
}

// SpaceFolderPath is where a project's files live — the answer to "where do I
// put the files?", which OpenSpaceFolder (screen_doors.go) then shows rather
// than describes.
func (a *Engine) SpaceFolderPath(name string) (string, error) {
	path, err := spacePath(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("ยังไม่มีโฟลเดอร์ของโปรเจกต์นี้")
	}
	return path, nil
}

// AddSpaceContext copies files the user picks into the project's context
// folder, and answers with the folder's new contents.
//
// Copied, not linked. A link would make the project's context depend on a file
// staying where it was on a machine the user reorganises — and the promise this
// room makes is that the project *carries* its material. What is in the folder
// is what every chat in the project knows about, and that has to survive the
// original being moved into a different folder tomorrow.
//
// Returns the file list rather than nothing so the page redraws from the disk
// it just changed, instead of from what the frontend assumes happened.
//
// AddSpaceContextFiles is the engine's half of AddSpaceContext
// (screen_doors.go): the files named are on this host, and so is the project.
func (a *Engine) AddSpaceContextFiles(name string, picked []string) ([]string, error) {
	path, err := spacePath(name)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("ยังไม่มีโปรเจกต์ชื่อนี้")
	}
	contextDir := filepath.Join(path, contextDirName)
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return nil, err
	}
	for _, source := range picked {
		if err := copyIntoContext(source, contextDir); err != nil {
			return nil, err
		}
	}
	return a.describeSpace(path, filepath.Base(path), 0).ContextFiles, nil
}

// copyIntoContext copies one picked file in, never overwriting: a second
// "รายงาน.pdf" becomes "รายงาน (2).pdf" rather than replacing the one already
// there. The user picked a file to add — silently losing the previous one is
// not something they asked for, and it would be unrecoverable.
func copyIntoContext(source, contextDir string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	base := filepath.Base(source)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	target := filepath.Join(contextDir, base)
	for n := 2; ; n++ {
		if _, err := os.Stat(target); err != nil {
			break
		}
		target = filepath.Join(contextDir, fmt.Sprintf("%s (%d)%s", stem, n, ext))
	}
	return os.WriteFile(target, data, 0o644)
}

// RemoveSpaceContext deletes one file out of a project's context folder.
//
// The name is a bare filename and is checked to resolve inside that folder —
// it arrives from the frontend, and the frontend is not the gate. Deleting is
// the user's own gesture on their own file (the page asks twice, as ผลงาน
// does); nothing here is reachable by the model.
func (a *Engine) RemoveSpaceContext(name, file string) ([]string, error) {
	path, err := spacePath(name)
	if err != nil {
		return nil, err
	}
	contextDir := filepath.Join(path, contextDirName)
	target := filepath.Join(contextDir, filepath.Base(strings.TrimSpace(file)))
	if filepath.Dir(target) != filepath.Clean(contextDir) || filepath.Base(target) == "" {
		return nil, fmt.Errorf("ไฟล์นี้ไม่ได้อยู่ในโฟลเดอร์บริบทของโปรเจกต์")
	}
	if err := os.Remove(target); err != nil {
		return nil, err
	}
	return a.describeSpace(path, filepath.Base(path), 0).ContextFiles, nil
}

// NewSessionInSpace opens a chat inside a space, at the assistant desk.
//
// The space is held on the Engine and written onto the session's row when the
// first message lands, exactly as the desk and the agent are (§83, §85): a
// session is born inside a project and stays there. Nothing about the sandbox
// moves — that is what separates this from the workshop's project and it is the
// promise the room's own description makes.
func (a *Engine) NewSessionInSpace(name string) (string, error) {
	path, err := spacePath(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("ยังไม่มีโปรเจกต์ชื่อนี้")
	}
	id, err := a.NewSessionAt(mode.Default)
	if err != nil {
		return "", err
	}
	a.cur().space = filepath.Base(path)
	// Re-bootstrap so the system prompt is built with the project in it.
	// NewSessionAt already did one, and it ran before this line — without a
	// second the assistant would be told about the project one message late,
	// which is the message where it matters most.
	a.rebuildCurrentConversation()
	return id, nil
}

// resolvedSpace answers what a stored space name means now: itself if the
// folder is still there, "" if it is not. The disk is the only record that a
// project exists, so a name that no longer resolves is a project that no longer
// exists — and telling the assistant it is working inside one that is gone
// would have it looking for a context folder nobody can put anything into.
func (a *Engine) resolvedSpace(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	path, err := spacePath(name)
	if err != nil {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return filepath.Base(path)
}

// SessionsInSpace is the space's own chat history, newest first.
func (a *Engine) SessionsInSpace(name string) []SessionMeta {
	out := []SessionMeta{}
	folder, err := spaceFolderName(name)
	if err != nil {
		return out
	}
	db, dbErr := a.database()
	if dbErr != nil {
		return out
	}
	// Snippet is the assistant's last words, which no other list carries: the
	// sidebar's rows are one line and the title has it. The project page draws
	// two, because there a row is one of five conversations that all begin
	// "ช่วยผม…" and the title alone cannot say which one wrote the post.
	// Clipped in SQL rather than shipping whole answers for a list of 200.
	out, _ = queryAll(db, "spaces: sessions", `
		SELECT s.id, s.title, s.updated_at, s.mode, s.agent, s.continued_from,
		       COALESCE((SELECT substr(m.text, 1, 200) FROM messages m
		                 WHERE m.session_id = s.id AND m.role = 'agent'
		                 ORDER BY m.id DESC LIMIT 1), '')
		FROM sessions s
		WHERE s.project_key = ? AND s.space = ? ORDER BY s.updated_at DESC LIMIT 200`,
		[]any{projectKey(a.cur().cfg.SandboxRoot), folder},
		func(rows *sql.Rows) (SessionMeta, error) {
			var m SessionMeta
			err := rows.Scan(&m.ID, &m.Title, &m.UpdatedAt, &m.Mode, &m.Agent, &m.ContinuedFrom, &m.Snippet)
			return m, err
		})
	return out
}

// CurrentSpace is which project this session is being held in, "" for a chat
// held outside every project.
func (a *Engine) CurrentSpace() string { return a.cur().space }

// spaceContextForPrompt is what the system prompt names: where the project
// keeps its files and which files those are — never their contents.
//
// Naming them costs a line and buys the only thing the assistant is missing,
// which is knowing they exist; it reads the ones a question needs with the tools
// it already has. See prompt.workingIn for why the other design — pasting them
// in — makes the assistant worse at everything else.
//
// It takes the conversation rather than reading a.cur(), which is what it did
// until 30 ส.ค. and was wrong for the reason conversation.go states about cur()
// in as many words: it is the chat the WINDOW is looking at, and the one thing
// the turn path may not ask for. applyConfig is handed its conversation, and
// endTurn calls applyConfig for a chat that is deliberately not the open one —
// a config change parked mid-turn — so a project chat finishing its work in the
// background was rebuilt with whichever project happened to be on screen, or
// with none. Its prompt then named the wrong folder and the wrong files, or
// dropped the project layer entirely, silently and only for background work.
//
// Nothing else read it, which is why nobody saw it: every other caller was on
// screen, so cur() and the conversation were the same object.
func (a *Engine) spaceContextForPrompt(conv *conversation) bootstrap.SpaceContext {
	if conv == nil || conv.space == "" {
		return bootstrap.SpaceContext{}
	}
	path, err := spacePath(conv.space)
	if err != nil {
		return bootstrap.SpaceContext{}
	}
	space := a.describeSpace(path, conv.space, 0)
	return bootstrap.SpaceContext{Path: space.ContextPath, Files: space.ContextFiles}
}
