package engine

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/skill"
)

func TestGuideDeskCarriesOnlyGuideTool(t *testing.T) {
	guideTool := &stubTool{name: "guide"}
	browserTool := &stubTool{name: "browser"}
	computerTool := &stubTool{name: computerToolName}

	lent := []skill.Skill{guideTool, browserTool, computerTool}
	a := bootDeskAppLending(t, "guide", lent)

	tools := toolNames(a)
	if len(tools) != 1 {
		t.Fatalf("guide desk has %d tools (%v), want exactly 1 ('guide')", len(tools), tools)
	}
	if tools[0] != "guide" {
		t.Errorf("guide desk tool is %q, want 'guide'", tools[0])
	}
}

func TestOtherDesksDoNotCarryGuideTool(t *testing.T) {
	guideTool := &stubTool{name: "guide"}
	lent := []skill.Skill{guideTool}

	for _, desk := range []string{"assistant", "coding"} {
		a := bootDeskAppLending(t, desk, lent)
		tools := toolNames(a)
		for _, tool := range tools {
			if tool == "guide" {
				t.Errorf("desk %q carried 'guide' tool, but guide must be exclusive to guide desk", desk)
			}
		}
	}
}

func TestGuideSessionsAreExcludedFromHistoryAndLists(t *testing.T) {
	a := bootDeskApp(t, "assistant")
	db, err := a.database()
	if err != nil {
		t.Fatalf("database: %v", err)
	}

	// Insert an ordinary session and a guide session
	normalID := "20260915-010101.001"
	guideID := "20260915-020202.002"
	pkey := projectKey(a.cur().cfg.SandboxRoot)

	_, err = db.Exec(`INSERT INTO sessions (id, title, created_at, updated_at, mode, project_key) VALUES (?, ?, datetime('now'), datetime('now'), ?, ?)`,
		normalID, "Normal Session", "assistant", pkey)
	if err != nil {
		t.Fatalf("insert normal session: %v", err)
	}
	_, err = db.Exec(`INSERT INTO sessions (id, title, created_at, updated_at, mode, project_key) VALUES (?, ?, datetime('now'), datetime('now'), ?, ?)`,
		guideID, "Guide Session", "guide", pkey)
	if err != nil {
		t.Fatalf("insert guide session: %v", err)
	}

	// 1. ListSessionsAt("")
	list := a.ListSessionsAt("")
	for _, s := range list {
		if s.ID == guideID || s.Mode == "guide" {
			t.Errorf("ListSessionsAt included guide session: %v", s)
		}
	}

	// 2. ListAllSessions()
	all := a.ListAllSessions()
	for _, s := range all {
		if s.ID == guideID || s.Mode == "guide" {
			t.Errorf("ListAllSessions included guide session: %v", s)
		}
	}

	// 3. ListSessionsForDoor (excluding coding)
	doorList := a.ListSessionsForDoor(DeskFilter{Desks: []string{"coding"}, Exclude: true})
	for _, s := range doorList {
		if s.ID == guideID || s.Mode == "guide" {
			t.Errorf("ListSessionsForDoor included guide session: %v", s)
		}
	}

	// 4. OpenDatabase cleans up guide sessions
	freshDB, err := a.openDatabase()
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer freshDB.Close()
	var count int
	_ = freshDB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE mode = 'guide'`).Scan(&count)
	if count != 0 {
		t.Errorf("openDatabase left %d guide sessions, want 0", count)
	}
	var normalCount int
	_ = freshDB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = ?`, normalID).Scan(&normalCount)
	if normalCount != 1 {
		t.Errorf("openDatabase deleted normal session, want 1, got %d", normalCount)
	}
}

// The guide's conversation is held beside the chat on screen, never in its
// place: opening one moves nothing, asking it by id runs a turn in it, and
// closing it leaves no row. NewSessionAt would have shown it — the cursor
// would have left the user's chat (guide.go says why that door is not used).
func TestGuideSessionIsHeldBesideTheChatNotShown(t *testing.T) {
	a := bootDeskApp(t, "assistant")
	before := a.cur()
	id, err := a.OpenGuideSession()
	if err != nil {
		t.Fatalf("OpenGuideSession: %v", err)
	}
	if id == "" {
		t.Fatal("OpenGuideSession answered with no id")
	}
	if a.cur() != before || a.cur().id == id {
		t.Fatalf("opening the guide moved the chat on screen to %q", a.cur().id)
	}
	conv := a.convs.find(id)
	if conv == nil {
		t.Fatal("the guide's conversation is not held")
	}
	if conv.desk == nil || conv.desk.DeskName() != "guide" {
		t.Fatalf("the guide sits at desk %v, want guide", conv.desk)
	}
	if _, err := a.SendToGuide("no-such-guide", "hi"); err == nil {
		t.Error("SendToGuide answered a guide that was never opened")
	}
	if _, err := a.SendToGuide(before.id, "hi"); err == nil {
		t.Error("SendToGuide ran a turn in the chat on screen")
	}
	if err := a.CloseGuideSession(id); err != nil {
		t.Fatalf("CloseGuideSession: %v", err)
	}
	if a.convs.find(id) != nil {
		t.Error("the guide's conversation is still held after close")
	}
	if a.cur() != before {
		t.Error("closing the guide moved the chat on screen")
	}
}
