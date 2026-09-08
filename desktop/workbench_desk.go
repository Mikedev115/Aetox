package main

// The agent's own reach onto the desk.
//
// Until this file, the whole of that reach was `browser_open`: the agent could
// put a web page on the desk and nothing else. It could write a deck and say so,
// and the user still had to go and click the card themselves — on the surface
// whose entire job is showing what was produced. §87 widened the desk from four
// panes to six and left the agent holding one of them, which made the gap worse
// rather than better.
//
// Three verbs closed it, and they are deliberately thin. `open` does not know
// what a pane is — it hands the path to the same opener the file tree and the
// drop handler call, and `fileView()` decides. That is what keeps the desk able
// to learn a new file type without anything here, or in the model's head,
// changing. See docs/architecture/desk-file-panes-2026-08-06.md §6.
//
// They are one tool as of 2026-08-20 — `desk`, with `open`, `list` and `close`
// inside it (§99's packing, owner's call). `close` is new with the pack and is
// the verb that was missing: a desk that can be filled and never emptied buries
// the file the user was reading under the five the agent opened after it.
//
// `focus` is the fourth, 8 ก.ย. Burying was only half of that complaint: the
// other half is that nothing could be brought back up. Every kind of pane
// answered "show me that again" differently or not at all — see deskFocusSkill
// for what each one used to cost — and the mirror never said which tab the
// person was looking at, so the agent could not even tell it had taken the view
// away from them. Both halves are one field on DeskTab and one verb here.
//
// `desk_terminal`, below, is NOT in that pack, and the line is worth stating
// because it is the one that keeps the pack free. This pack is the SURFACE —
// put something on it, see what is on it, take something off. A terminal is a
// thing that lives on the surface with a back-and-forth of its own, which is
// what the browser is too, and the browser has always been its own pack. Kept
// out, every action in `desk` is CategoryAgent and every one of them only ever
// looks, so no desk and no stance has to cut inside it — and neither of them
// can (packed.go).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// ---------------------------------------------------------------------------
// what is on the desk
// ---------------------------------------------------------------------------

// DeskTab is one tab as the frontend sees it, reported here so the agent can
// read its own desk. The frontend pushes the whole list on every change
// (WorkbenchTabsChanged); nothing on the Go side tries to track it by watching
// the events it sends, which would drift the moment the user closed a tab.
type DeskTab struct {
	Kind string `json:"kind"` // terminal | browser | files | file | decks | cutroom | ...
	Name string `json:"name"`
	Path string `json:"path,omitempty"` // file tabs
	URL  string `json:"url,omitempty"`  // browser tabs
	// Mine reports that the agent opened this tab rather than the user. It is
	// the whole basis of the redaction below.
	Mine bool `json:"mine"`
	// ID is the tab's address in the strip — `web-agent-1` for a page the agent
	// opened, `web-3` for one the user did, `file-<path>`, or the bare name of a
	// singleton pane (`git`, `files`). It is here so `focus` has something exact
	// to aim at: a path names a file tab and nothing else, and until this
	// arrived a terminal or a git pane could not be named at all.
	//
	// Not a leak of anything §81 protects: an id says a tab exists, which the
	// redaction below already says out loud. What it does not say is where the
	// page went, and that is still withheld.
	ID string `json:"id,omitempty"`
	// Active is the one tab the user is actually looking at.
	//
	// The mirror carried the whole strip and never this, so the agent could see
	// what was on the desk and not what was in front of the person — it could
	// not tell that its own `desk open` had just taken the view away from
	// something they were reading. `browser tabs list` has marked its current
	// tab with a `*` since it was written; this is the same answer for the
	// surface that owns the question.
	Active bool `json:"active,omitempty"`
}

type deskState struct {
	mu   sync.Mutex
	tabs []DeskTab
}

// WorkbenchTabsChanged is called by the frontend whenever the tab strip changes.
//
// The session is named because the window keeps one workbench per chat and this
// is its mirror: an unnamed report lands on whichever conversation this side
// thinks is current, which is the chat on screen — right until a chat working in
// the background asks what is on its desk and is told about somebody else's.
//
// An empty id means a window that has not been told which chat it is on yet,
// which is the moment before the first load answers; the chat on screen is the
// only conversation there is then.
func (a *App) WorkbenchTabsChanged(sessionID string, tabs []DeskTab) {
	conv := a.convs.find(sessionID)
	if conv == nil {
		conv = a.cur()
	}
	conv.openTabs.mu.Lock()
	conv.openTabs.tabs = tabs
	conv.openTabs.mu.Unlock()
}

func deskTabsOf(conv *conversation) []DeskTab {
	conv.openTabs.mu.Lock()
	defer conv.openTabs.mu.Unlock()
	return append([]DeskTab(nil), conv.openTabs.tabs...)
}

// ---------------------------------------------------------------------------
// desk_open
// ---------------------------------------------------------------------------

// conv is the chat this tool was built for: a desk_open from a conversation
// working in the background opens against ITS project, not against whatever the
// window is showing. Same reason askUserSkill carries one.
type deskOpenSkill struct {
	app  *App
	conv *conversation
}

func (s *deskOpenSkill) open(path string) (skill.Output, error) {
	start := time.Now()
	out := skill.Output{Name: "desk_open", Command: "desk_open " + path}
	fail := func(err error) (skill.Output, error) {
		out.Content = "เปิดบนโต๊ะไม่สำเร็จ: " + err.Error()
		out.Stderr = err.Error()
		out.DurationMs = time.Since(start).Milliseconds()
		return out, err
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return fail(fmt.Errorf("path is required"))
	}
	if s.app.ctx == nil {
		return fail(fmt.Errorf("UI not ready"))
	}
	// Resolved here rather than left to the frontend: a tab that opens onto a
	// path that does not exist shows the user a card reading "file is gone",
	// which reads as the agent having lost the file it just made. The error
	// belongs in the turn, where the model can still do something about it.
	// This chat's root, not the app's. The skill is built per conversation
	// (workbenchSkills), so a tool call from a chat working in the background
	// resolves against the project THAT chat is in — App.cfg is only the
	// template a new chat is born from now (DECISIONS §155).
	root := strings.TrimSpace(s.conv.cfg.SandboxRoot)
	if root == "" {
		return fail(fmt.Errorf("no project open"))
	}
	// Resolved the way `browser open` resolves it (normalizeWorkbenchURL), and
	// for the same reason: `write` steers a new relative file into
	// output/<session> and its receipt echoes the path the model ASKED for. A
	// model that then hands that path here — which is exactly what this tool's
	// description tells it to do — was told the file does not exist, one line
	// after making it. Reported by the owner on 2026-08-20 against a deck it
	// had just written.
	//
	// PlacedPath is the one definition of that rule and the literal path still
	// wins, so a real file is never shadowed by a same-named artifact.
	placed := skill.PlacedPath(root, func() string { return s.app.outputSubdirOf(s.conv) }, path)
	full, err := safeSandboxPath(root, placed)
	if err != nil {
		return fail(err)
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return fail(fmt.Errorf("%s does not exist", path))
	}
	// From here on the placed path is the path: it is what the tab, ReadFile
	// and the file host all have to resolve, and none of them knows the rule.
	path = placed

	name := filepath.Base(filepath.FromSlash(path))
	// The session rides in the event because the window routes on it: a
	// desk_open from a chat working in the background must land on THAT
	// chat's saved desk, not on whichever desk is on screen (§187). Without
	// it the tab appeared in front of whoever happened to be looking, and
	// the on-screen session's next snapshot persisted the stray as its own.
	s.app.deskEvent(s.conv.id, "open-file", map[string]string{"path": path, "name": name})

	out.Success = true
	out.DurationMs = time.Since(start).Milliseconds()
	out.Content = deskOpenedLine(path)
	out.RawOutput = out.Content
	return out, nil
}

// deskOpenedLine is written once, here, so the round-trip test asserts the real
// sentence rather than its own copy of the format (same reasoning as
// browserOpenedLine).
func deskOpenedLine(path string) string {
	return fmt.Sprintf("วางไฟล์ %s บนโต๊ะแล้ว ผู้ใช้เห็นอยู่ตอนนี้", path)
}

// ---------------------------------------------------------------------------
// desk_terminal
// ---------------------------------------------------------------------------

// conv is the chat this tool was built for, same as deskOpenSkill and for
// the same reason: its terminal belongs on ITS desk (§187).
type deskTerminalSkill struct {
	app  *App
	conv *conversation
}

func (*deskTerminalSkill) Name() string { return "desk_terminal" }

func (*deskTerminalSkill) Description() string {
	return "เปิดเทอร์มินัลบนโต๊ะ แล้วพิมพ์คำสั่งให้ผู้ใช้เห็นสด ๆ"
}

func (*deskTerminalSkill) ToolDefinition() model.ToolDefinition {
	return toolDef("desk_terminal",
		"Open a terminal on the user's desk and optionally type a command into it. This is for the user to WATCH — the output streams into a real terminal in front of them, live. It is not how you read a command's output: `shell` is, and it is faster and gives you the text. Reach for this when being seen matters — a build or a server the user should watch run, or a session they will keep typing into themselves afterwards.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "Command to type and run. Omit to open an empty terminal for the user."},
			},
		})
}

func (s *deskTerminalSkill) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	cmd, _ := args["command"].(string)
	return s.run(cmd)
}

func (s *deskTerminalSkill) Execute(_ context.Context, input skill.Input) (skill.Output, error) {
	cmd, _ := input["command"].(string)
	return s.run(cmd)
}

// openDeskTerminal is the one way a terminal appears on the desk from the Go
// side: start the shell, tell the frontend to mount a pane on it, and type the
// command in if there is one. Extracted from the tool below on 2026-08-11,
// when the engine starter (engine_server.go) became its second caller — an
// engine starting in a window the user watches is this exact act, and a copy
// of these lines over there would be two ways to open the same terminal.
// sessionID is whose desk the pane belongs on — the calling chat's, or ""
// for a caller with no conversation (the engine starter's log terminal, which
// is deliberately for whoever is looking).
func (a *App) openDeskTerminal(sessionID, command string) (shellName string, err error) {
	if a.ctx == nil {
		return "", fmt.Errorf("UI not ready")
	}
	shells := a.TerminalShells()
	if len(shells) == 0 {
		return "", fmt.Errorf("no shell found on this machine")
	}
	// The first profile, not the user's saved default: that preference lives in
	// the frontend's localStorage and the Go side cannot see it. Worth knowing
	// rather than guessing at — if it starts mattering, the frontend has to send
	// it over, and this comment is where to start.
	sh := shells[0]

	id, err := a.TerminalStart(sh.Path, 80, 24)
	if err != nil {
		return "", err
	}
	// The session exists before the tab does, which is the reverse of the
	// browser's flow (there the frontend creates the window and Go polls for
	// it). It is simpler this way round: the id is already real, so the frontend
	// only has to mount a pane onto it, and there is nothing to wait for.
	// The path travels with the event, not only the name. The window saves the
	// desk it is drawing, and a terminal it cannot start again is a tab that
	// disappears when the app closes — which is what the whole panel used to do.
	a.deskEvent(sessionID, "open-terminal", map[string]string{"id": id, "name": sh.Name, "path": sh.Path})

	if command != "" {
		// A newline is what makes it run. Given to the PTY exactly as a keypress
		// would arrive, because that is what this is: the agent typing.
		if err := a.TerminalWrite(id, command+"\r"); err != nil {
			return "", err
		}
	}
	return sh.Name, nil
}

func (s *deskTerminalSkill) run(command string) (skill.Output, error) {
	start := time.Now()
	command = strings.TrimSpace(command)
	out := skill.Output{Name: "desk_terminal", Command: "desk_terminal " + command}
	fail := func(err error) (skill.Output, error) {
		out.Content = "เปิดเทอร์มินัลไม่สำเร็จ: " + err.Error()
		out.Stderr = err.Error()
		out.DurationMs = time.Since(start).Milliseconds()
		return out, err
	}

	name, err := s.app.openDeskTerminal(s.conv.id, command)
	if err != nil {
		return fail(err)
	}
	sh := ShellProfile{Name: name}

	out.Success = true
	out.DurationMs = time.Since(start).Milliseconds()
	if command == "" {
		out.Content = fmt.Sprintf("เปิดเทอร์มินัล %s บนโต๊ะแล้ว", sh.Name)
	} else {
		// Says plainly that the output is not coming back here, so a model
		// waiting for it stops waiting and uses shell instead next time.
		out.Content = fmt.Sprintf("เปิดเทอร์มินัล %s บนโต๊ะแล้วพิมพ์ `%s` ให้ ผู้ใช้กำลังเห็นผลสด ๆ — ผลลัพธ์ไม่ได้ส่งกลับมาที่นี่ ถ้าต้องอ่านผลเองให้ใช้ shell", sh.Name, command)
	}
	out.RawOutput = out.Content
	return out, nil
}

// ---------------------------------------------------------------------------
// desk_list
// ---------------------------------------------------------------------------

// conv is the chat whose desk this reports. Same reason askUserSkill and
// deskOpenSkill carry one: a tool built for a conversation answers about that
// conversation.
type deskListSkill struct {
	app  *App
	conv *conversation
}

func (s *deskListSkill) list() (skill.Output, error) {
	start := time.Now()
	tabs := deskTabsOf(s.conv)
	lines := describeDesk(tabs)
	content := "โต๊ะว่าง ยังไม่มีอะไรเปิดอยู่"
	if len(lines) > 0 {
		content = "บนโต๊ะตอนนี้:\n" + strings.Join(lines, "\n")
		// Spent only when a row is actually wearing the mark: a legend for a
		// symbol that is nowhere on the list is a line explaining nothing. An
		// engine talking to a window from before Active existed reports no
		// active tab at all, and gets the listing it has always had.
		if slices.ContainsFunc(tabs, func(t DeskTab) bool { return t.Active }) {
			content += "\n* คือแท็บที่ผู้ใช้กำลังเห็นอยู่ตอนนี้ ใช้ focus เพื่อสลับไปแท็บอื่น"
		}
	}
	return skill.Output{
		Name:       "desk_list",
		Command:    "desk_list",
		Success:    true,
		Content:    content,
		RawOutput:  content,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// describeDesk renders the tab list for the agent, with one redaction.
//
// §81 decided that the workbench's browsing history stays in localStorage
// rather than the tool_runs table, because that table is the agent's own
// searchable memory and putting the user's personal browsing in it "is a far
// bigger decision than a start-page list gets to make". A desk_list that
// returned page titles and URLs would hand over exactly that, through a
// different door.
//
// So: a page the agent opened reports fully — it already knows, it put it
// there. A page the user opened reports that it exists and nothing else.
func describeDesk(tabs []DeskTab) []string {
	lines := make([]string, 0, len(tabs))
	for _, t := range tabs {
		// The mark the browser's own listing uses, for the same fact. A row
		// carrying it is the row the user is looking at.
		mark := "-"
		if t.Active {
			mark = "*"
		}
		var row string
		switch {
		case t.Kind == "browser" && !t.Mine:
			row = "แท็บเบราว์เซอร์ที่ผู้ใช้เปิดเอง (ไม่เปิดเผยที่อยู่)"
		case t.Kind == "browser":
			row = fmt.Sprintf("เบราว์เซอร์: %s (%s)", t.Name, t.URL)
		case t.Kind == "file":
			row = fmt.Sprintf("ไฟล์: %s", t.Path)
		case t.Kind == "terminal":
			row = fmt.Sprintf("เทอร์มินัล: %s", t.Name)
		default:
			row = t.Name
		}
		// The address `focus` takes, printed for everything a path cannot name.
		// A file row already ends in the path, which is the address for it and
		// the one `open` and `close` have always taken — printing `file-<path>`
		// beside it would be the same tab named twice.
		if t.ID != "" && t.Kind != "file" {
			row += " [" + t.ID + "]"
		}
		lines = append(lines, mark+" "+row)
	}
	return lines
}

// deskTabLabel names a tab in a sentence the agent reads back, with §81's
// redaction applied once, here.
//
// It exists because `focus` answers in prose where `list` answers in rows, and
// the rule that a page the user opened never has its address repeated has to
// hold in both. A tab's NAME is the hostname for a browser tab (labelForUrl),
// so echoing the name of somebody's own tab is echoing where they went.
func deskTabLabel(t DeskTab) string {
	switch {
	case t.Kind == "browser" && !t.Mine:
		return "แท็บเบราว์เซอร์ของผู้ใช้"
	case t.Kind == "browser":
		return "เบราว์เซอร์ " + t.Name
	case t.Kind == "file":
		return "ไฟล์ " + t.Path
	case t.Kind == "terminal":
		return "เทอร์มินัล " + t.Name
	}
	return t.Name
}

// ---------------------------------------------------------------------------
// desk_close
// ---------------------------------------------------------------------------

type deskCloseSkill struct {
	app  *App
	conv *conversation
}

// close takes a file off the desk.
//
// The desk had no way to put anything down until now, which shows up the first
// time an agent opens five files in one turn and buries the one the user was
// reading. `browser` has closed its own tabs since it grew a `tabs` action, and
// a terminal is a session rather than a view — so this is deliberately about
// FILE tabs and says so, rather than becoming a third way to stop a shell.
//
// Only a tab the agent opened. §81's rule is that what the user is doing on
// their own machine is not the agent's to read; taking away a file they opened
// themselves is the same rule with a heavier hand, and the `Mine` flag that
// answers it is already on the wire for desk_list.
func (s *deskCloseSkill) close(path string) (skill.Output, error) {
	start := time.Now()
	out := skill.Output{Name: "desk_close", Command: "desk_close " + path}
	fail := func(err error) (skill.Output, error) {
		out.Content = "ปิดของบนโต๊ะไม่สำเร็จ: " + err.Error()
		out.Stderr = err.Error()
		out.DurationMs = time.Since(start).Milliseconds()
		return out, err
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return fail(fmt.Errorf("path is required"))
	}
	if s.app.ctx == nil {
		return fail(fmt.Errorf("UI not ready"))
	}
	// Resolved the same way desk_open resolves it, so the path that opened a
	// file also closes it. A model that had to remember which spelling it used
	// would be remembering an implementation detail of the output folder.
	root := strings.TrimSpace(s.conv.cfg.SandboxRoot)
	if root != "" {
		path = skill.PlacedPath(root, func() string { return s.app.outputSubdirOf(s.conv) }, path)
	}

	var found *DeskTab
	for _, tab := range deskTabsOf(s.conv) {
		if tab.Kind == "file" && tab.Path == path {
			found = &tab
			break
		}
	}
	if found == nil {
		return fail(fmt.Errorf("%s is not on the desk", path))
	}
	if !found.Mine {
		return fail(fmt.Errorf("%s was opened by the user, so it is not yours to close", path))
	}

	s.app.deskEvent(s.conv.id, "close-file", map[string]string{"path": path})

	out.Success = true
	out.DurationMs = time.Since(start).Milliseconds()
	out.Content = fmt.Sprintf("เก็บ %s ออกจากโต๊ะแล้ว", path)
	out.RawOutput = out.Content
	return out, nil
}

// ---------------------------------------------------------------------------
// desk_focus
// ---------------------------------------------------------------------------

// deskFocusSkill brings a tab that is ALREADY on the desk to the front.
//
// The verb the surface never had. "ขอดูอันนั้นอีกที" was answered one way per
// kind and, for most kinds, not at all: a browser tab had `browser tabs
// select`, a file had `desk open` a second time — which re-reads the file,
// rebuilds the pane and force-saves whatever the user had half-typed into it
// (FileEditor's onDestroy) — and a terminal, a git pane or the cutting room had
// nothing whatsoever. One question, three answers and a hole, which is what the
// owner named on 8 ก.ย.: *"ควบคุมได้ดีแค่เบราว์เซอร์ พอเป็นตัวอื่นกลับพัง ทั้งที่ควรจะ
// เป็นมาตรฐานเดียวกัน"*.
//
// It changes what the user SEES and nothing about what the agent may read, and
// that line is what lets it take a tab the user opened themselves — the one
// thing `close` refuses. Raising somebody's own page back into view is doing
// what they asked for; reading it stays refused where it was always refused
// (mustOwn, and describeDesk's redaction), and no address is handed over here.
//
// It knows nothing about panes, the same way `open` does not: it names a tab
// and the window makes it active. A desk that learns a seventh kind of pane
// gains this for free on the day it is written.
type deskFocusSkill struct {
	app  *App
	conv *conversation
}

// findDeskTab resolves what the model typed against what is on the desk.
//
// Four passes, widening: the id `list` prints, the file path `open` and `close`
// already take, the tab's exact name, then a name merely containing it. The
// order is the whole design — `git` is the id of one pane and a word inside the
// name of any file called git.md, and an address that resolves to a different
// tab depending on what else happens to be open is worse than no address. An
// ambiguous partial match is refused with the candidates named rather than
// picked between.
//
// **A browser tab the user opened is reachable by id and by nothing else.**
// Its name is the hostname (labelForUrl), so matching on names would answer
// "is mail.google.com open?" with a success or a failure — §81's redaction
// walked around through the back door, one guess at a time. An id says a tab
// exists, which `list` says already, and says nothing about where it went.
func findDeskTab(tabs []DeskTab, target, placed string) (DeskTab, error) {
	want := strings.ToLower(strings.TrimSpace(target))
	if want == "" {
		return DeskTab{}, fmt.Errorf("tab is required — use list to see what is on the desk")
	}
	for _, t := range tabs {
		if t.ID != "" && strings.EqualFold(t.ID, want) {
			return t, nil
		}
	}
	for _, t := range tabs {
		if t.Kind == "file" && t.Path != "" && (t.Path == placed || t.Path == target) {
			return t, nil
		}
	}
	named := func(t DeskTab) bool { return t.Kind != "browser" || t.Mine }
	for _, t := range tabs {
		if named(t) && strings.EqualFold(t.Name, want) {
			return t, nil
		}
	}
	var near []DeskTab
	for _, t := range tabs {
		if !named(t) {
			continue
		}
		if strings.Contains(strings.ToLower(t.Name), want) ||
			(t.Path != "" && strings.Contains(strings.ToLower(t.Path), want)) {
			near = append(near, t)
		}
	}
	switch len(near) {
	case 1:
		return near[0], nil
	case 0:
		return DeskTab{}, fmt.Errorf("ไม่มี %s อยู่บนโต๊ะ ใช้ list ดูว่ามีอะไรเปิดอยู่", target)
	}
	labels := make([]string, 0, len(near))
	for _, t := range near {
		labels = append(labels, deskTabLabel(t))
	}
	return DeskTab{}, fmt.Errorf("%s ตรงกับหลายแท็บ (%s) ระบุให้ชัดกว่านี้ หรือใช้ id จาก list",
		target, strings.Join(labels, ", "))
}

func (s *deskFocusSkill) focus(target string) (skill.Output, error) {
	start := time.Now()
	target = strings.TrimSpace(target)
	out := skill.Output{Name: "desk_focus", Command: "desk_focus " + target}
	fail := func(err error) (skill.Output, error) {
		out.Content = "สลับแท็บไม่สำเร็จ: " + err.Error()
		out.Stderr = err.Error()
		out.DurationMs = time.Since(start).Milliseconds()
		return out, err
	}

	if target == "" {
		return fail(fmt.Errorf("tab is required"))
	}
	if s.app.ctx == nil {
		return fail(fmt.Errorf("UI not ready"))
	}
	// Resolved the way `open` and `close` resolve it, so the path that put a
	// file on the desk is also the path that comes back to it.
	placed := target
	if root := strings.TrimSpace(s.conv.cfg.SandboxRoot); root != "" {
		placed = skill.PlacedPath(root, func() string { return s.app.outputSubdirOf(s.conv) }, target)
	}
	tab, err := findDeskTab(deskTabsOf(s.conv), target, placed)
	if err != nil {
		return fail(err)
	}
	// A window from before DeskTab carried an id reports neither, and only a
	// name match can have got us here. Said plainly rather than emitting an
	// event that names nothing and reporting a switch that never happened.
	if tab.ID == "" && tab.Path == "" {
		return fail(fmt.Errorf("หน้าต่างรุ่นนี้ยังไม่ได้บอกที่อยู่ของแท็บ อัปเดตแอปแล้วลองใหม่"))
	}

	out.Success = true
	defer func() {
		out.RawOutput = out.Content
		out.DurationMs = time.Since(start).Milliseconds()
	}()
	if tab.Active {
		// Not a failure and not an event: the desk is already showing it. Worth
		// saying which, because the model asked for something it turned out to
		// already have and the next thing it does should not be asking again.
		out.Content = fmt.Sprintf("%s อยู่หน้าจออยู่แล้ว ไม่ได้สลับอะไร", deskTabLabel(tab))
		return out, nil
	}

	// Both, and the frontend tries them in that order: `tab` is the address for
	// everything, `path` is what a window mid-upgrade can still resolve a file
	// tab from (`file-<path>` is how that id is built).
	s.app.deskEvent(s.conv.id, "focus-tab", map[string]string{"tab": tab.ID, "path": tab.Path})

	// The honest half. A chat working in the background parks its focus on its
	// OWN saved desk (§187) — the tab will be in front when its user opens that
	// chat, and claiming they can see it now would be a sentence the model then
	// repeats to somebody looking at a different screen. capture asks the same
	// question the same way.
	if s.conv.id != "" && s.conv.id != s.app.cur().id {
		out.Content = fmt.Sprintf("ตั้ง %s ให้เป็นแท็บหน้าสุดของโต๊ะแชตนี้แล้ว แชตนี้ไม่ได้อยู่บนจอ ผู้ใช้จะเห็นเมื่อเปิดแชตนี้", deskTabLabel(tab))
		return out, nil
	}
	out.Content = deskFocusedLine(tab)
	return out, nil
}

// deskFocusedLine is written once, here, for the same reason deskOpenedLine is:
// the round-trip test asserts the real sentence rather than its own copy.
func deskFocusedLine(t DeskTab) string {
	return fmt.Sprintf("สลับไปที่ %s แล้ว ผู้ใช้เห็นอยู่ตอนนี้", deskTabLabel(t))
}

// ---------------------------------------------------------------------------
// desk — one name, four rights
// ---------------------------------------------------------------------------

// deskSkill is the surface itself: put something on it, see what is on it, take
// something off, and move what is in front. Packed on 2026-08-20 by §99's
// mechanism, on the owner's call; `focus` joined it on 8 ก.ย. and joined it
// rather than becoming a tool of its own because it passes both of the pack's
// tests — CategoryAgent like the other three, and it changes nothing on the
// machine, so วางแผน still carries the whole tool.
//
// What is NOT in here is the design. `desk_terminal` stays its own tool and the
// browser has always been its own pack, because those are things that LIVE on
// the desk and carry their own back-and-forth — a terminal you keep typing into,
// a page you read and click. Leaving them out is what makes this pack free: all
// three actions are CategoryAgent, so no desk gains or loses one of them by
// accident, and all three only ever look, so วางแผน carries the whole tool. A
// pack whose members disagree on either can only be cut by name — neither the
// desk gate nor the stance can reach inside one (packed.go).
type deskSkill struct {
	app  *App
	conv *conversation
	// actions this caller may use, nil for all of them. Set only by Narrow.
	actions []string
}

func (s *deskSkill) allowedActions() []string {
	if s == nil || len(s.actions) == 0 {
		out := make([]string, 0, len(skill.PackedCalls("desk")))
		for _, call := range skill.PackedCalls("desk") {
			out = append(out, call.Action)
		}
		return out
	}
	return s.actions
}

func (s *deskSkill) Actions() []string { return skill.PackedActions("desk") }

// Narrow hands back a desk offering only the named actions — a copy, for the
// same shared-registry reason as the browser's and shell's.
func (s *deskSkill) Narrow(named []string) skill.Skill {
	narrowed := *s
	want := map[string]bool{}
	for _, n := range named {
		want[strings.ToLower(strings.TrimSpace(n))] = true
	}
	var actions []string
	for _, call := range skill.PackedCalls("desk") {
		if want[call.Permission] {
			actions = append(actions, call.Action)
		}
	}
	// Silence is the whole tool, not an empty one — §99.2's rule.
	if len(actions) == 0 {
		return s
	}
	narrowed.actions = actions
	return &narrowed
}

func (*deskSkill) Name() string { return "desk" }

func (*deskSkill) Description() string {
	return "โต๊ะทำงานของผู้ใช้ — วางไฟล์ให้ดู ดูว่ามีอะไรอยู่ และเก็บออก"
}

func (s *deskSkill) ToolDefinition() model.ToolDefinition {
	allowed := s.allowedActions()

	// SIGNATURES ONLY. Everything that used to explain when to reach for these
	// now lives in Guidance below and is sent once — the standard set in
	// internal/skill/block_standard_test.go.
	lines := map[string]string{
		"open":  "`open` (path) — put a file in front of the user.",
		"list":  "`list` — what is on the desk right now, and which tab they are looking at.",
		"close": "`close` (path) — take a file you opened back off.",
		"focus": "`focus` (tab) — bring something already on the desk back to the front.",
	}
	var b strings.Builder
	b.WriteString("The user's desk: the panel beside the chat where they see what you produce. Actions:\n")
	for _, action := range allowed {
		b.WriteString(lines[action] + "\n")
	}

	return toolDef("desk", b.String(), map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{"type": "string", "enum": allowed},
			"path":   map[string]any{"type": "string"},
			// `focus` takes anything `list` printed — an id in brackets, or the
			// path of a file. A second name rather than reusing `path` because
			// most of what it can aim at is not a file at all, and a parameter
			// called path holding `web-agent-1` teaches the wrong lesson about
			// what the desk is made of.
			"tab": map[string]any{"type": "string"},
		},
		"required": []string{"action"},
	})
}

// Guidance is what somebody needs to know once, keyed per action.
//
// `open` carries the deck anatomy, and that is the hole this closes. §149 found
// that the slide marker lived "in no prompt, no profile and no tool description"
// and fixed it for the agents that write documents; the assistant was still told
// only that slides are HTML (internal/mode/modes/assistant.md) and never what
// makes an HTML file a deck. On 2026-08-20 it duly wrote one carrying its own
// navigation buttons and its own stacking, and the room — which pages the file
// itself — could not drive it. The file was fine; nobody had told the writer
// what the room does with it.
func (s *deskSkill) Guidance(args map[string]any) string {
	switch strings.ToLower(strings.TrimSpace(str(args["action"]))) {
	case "open":
		return "Pass the path write/doc_write/sheet_write reported. The desk picks the pane — a picture as a picture, a PDF as a reader, a spreadsheet as a grid, anything else in the editor — so there is nothing to choose here.\n" +
			"A web page (.html without slides) opens here as source. To show it rendered, `browser open` takes the same path; that is where a page you built for someone belongs.\n" +
			"Slides: a deck is one .html file whose slides are <section class=\"slide\">, and do NOT build navigation into it — the room already pages, presents full-screen and exports .pptx/.pdf, so a deck that moves itself is one the room cannot drive. The `aetox-slides` skill is the full recipe; read it before writing a deck rather than after."
	case "list":
		return "The row marked `*` is the tab the user is looking at right now; `focus` moves it.\n" +
			"A page the user opened themselves reports only that it exists — never its address. That is not a gap to work around."
	case "close":
		return "Only a file you opened. Use it to clear something you put up that is finished with, not to tidy the user's desk for them."
	case "focus":
		return "Pass what `list` printed: the id in brackets, or a file's path. It only moves the view — a tab the user opened is yours to bring back up and still not yours to read.\n" +
			"For a file already on the desk this is the call, not `open` again: opening re-reads the file and rebuilds the pane, which throws away where they had scrolled to and anything they were typing.\n" +
			"Use it to give the desk back when you have finished showing something — the user was looking at a page before you put five files over it."
	}
	return ""
}

func (s *deskSkill) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	return s.run(args)
}

func (s *deskSkill) Execute(_ context.Context, input skill.Input) (skill.Output, error) {
	return s.run(map[string]any(input))
}

func (s *deskSkill) run(args map[string]any) (skill.Output, error) {
	action := strings.ToLower(strings.TrimSpace(str(args["action"])))
	if action == "" {
		action = "open" // the pack's fallback, and the call every habit makes
	}
	// Refused here as well as hidden from the description, because a
	// description is guidance and a gate is a gate.
	if !slices.Contains(s.allowedActions(), action) {
		return skill.Output{Name: "desk"}, fmt.Errorf("desk %s is not available here — this session may use: %s",
			action, strings.Join(s.allowedActions(), ", "))
	}

	switch action {
	case "open":
		return (&deskOpenSkill{app: s.app, conv: s.conv}).open(str(args["path"]))
	case "list":
		return (&deskListSkill{app: s.app, conv: s.conv}).list()
	case "close":
		return (&deskCloseSkill{app: s.app, conv: s.conv}).close(str(args["path"]))
	case "focus":
		// `path` is read as a fallback: a model that has just called `open` and
		// `close` with one has every reason to reach for it again, and refusing
		// that would be the tool being right about its own vocabulary at the
		// cost of the call working.
		target := str(args["tab"])
		if strings.TrimSpace(target) == "" {
			target = str(args["path"])
		}
		return (&deskFocusSkill{app: s.app, conv: s.conv}).focus(target)
	}
	return skill.Output{Name: "desk"}, fmt.Errorf("unknown desk action %q", action)
}
