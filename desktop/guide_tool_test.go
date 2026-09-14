package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Mikedev115/Aetox/internal/engine"
	"github.com/Mikedev115/Aetox/internal/skill"
)

type fakeSession struct {
	id string
}

func (f *fakeSession) ID() string   { return f.id }
func (f *fakeSession) Root() string { return "" }

var _ engine.Session = (*fakeSession)(nil)

func TestGuideSkillPressNotSafeReturnsError(t *testing.T) {
	app := &App{}
	sess := &fakeSession{id: "sess-guide-1"}
	s := newGuideSkill(app, sess)

	app.emit = func(event string, data ...any) {
		if event == "screen:guide" && len(data) > 0 {
			ask, ok := data[0].(guideAskEvent)
			if ok && ask.Action == "press" {
				go app.AnswerGuide(ask.ID, `{"ok":false,"error":"not safe: button cannot be pressed"}`)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.Execute(ctx, skill.Input{
		"action": "press",
		"id":     "chat.send",
	})
	if err == nil {
		t.Fatal("press safe:false did not return error, want not safe error")
	}
	if !strings.Contains(err.Error(), "not safe") {
		t.Errorf("press error %q does not mention 'not safe'", err.Error())
	}
}

func TestGuideSkillWhereDiffsVisibleOnSamePage(t *testing.T) {
	app := &App{}
	sess := &fakeSession{id: "sess-guide-diff"}
	s := newGuideSkill(app, sess)

	currentSnapshot := whereOutput{
		Page:    "chat",
		GuideAt: "chat.send",
		Visible: []guideElement{
			{ID: "chat.send", Name: "Send", Safe: false},
			{ID: "chat.input", Name: "Input", Safe: true},
		},
	}

	app.emit = func(event string, data ...any) {
		if event == "screen:guide" && len(data) > 0 {
			ask, ok := data[0].(guideAskEvent)
			if ok && ask.Action == "where" {
				bytes, _ := json.Marshal(currentSnapshot)
				go app.AnswerGuide(ask.ID, string(bytes))
			}
		}
	}

	ctx := context.Background()

	// 1. First where call: should return all visible elements
	out1, err := s.Execute(ctx, skill.Input{"action": "where"})
	if err != nil {
		t.Fatalf("where 1 failed: %v", err)
	}
	var res1 whereOutput
	if err := json.Unmarshal([]byte(out1.Content), &res1); err != nil {
		t.Fatalf("unmarshal where 1: %v", err)
	}
	if len(res1.Visible) != 2 {
		t.Errorf("where 1 visible count = %d, want 2", len(res1.Visible))
	}

	// 2. Second where call on same page: should return diff (0 new elements)
	out2, err := s.Execute(ctx, skill.Input{"action": "where"})
	if err != nil {
		t.Fatalf("where 2 failed: %v", err)
	}
	var res2 whereOutput
	if err := json.Unmarshal([]byte(out2.Content), &res2); err != nil {
		t.Fatalf("unmarshal where 2: %v", err)
	}
	if len(res2.Visible) != 0 {
		t.Errorf("where 2 (diff) visible count = %d, want 0", len(res2.Visible))
	}

	// 3. Third where call with full: true: should return full list
	out3, err := s.Execute(ctx, skill.Input{"action": "where", "full": true})
	if err != nil {
		t.Fatalf("where 3 failed: %v", err)
	}
	var res3 whereOutput
	if err := json.Unmarshal([]byte(out3.Content), &res3); err != nil {
		t.Fatalf("unmarshal where 3: %v", err)
	}
	if len(res3.Visible) != 2 {
		t.Errorf("where 3 (full) visible count = %d, want 2", len(res3.Visible))
	}
}

func TestGuideSkillContextCancel(t *testing.T) {
	app := &App{}
	sess := &fakeSession{id: "sess-guide-timeout"}
	s := newGuideSkill(app, sess)

	// Don't answer the askGuide call
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := s.Execute(ctx, skill.Input{"action": "where"})
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}

func TestGuideSkillToolDefinitionIncludesMapIndex(t *testing.T) {
	app := &App{}
	sess := &fakeSession{id: "sess-guide-index"}
	s := newGuideSkill(app, sess)

	indexJSON := `[
		{"id":"chat.send","name":"ส่งข้อความ","safe":false},
		{"id":"settings.rail.brain","name":"ต่อสมอง","safe":true}
	]`
	app.guideIndices.Store(sess.id, indexJSON)

	tool, ok := s.(skill.Tool)
	if !ok {
		t.Fatalf("guideSkill does not implement skill.Tool")
	}
	def := tool.ToolDefinition()
	desc := def.Function.Description
	if !strings.Contains(desc, "- chat.send (ส่งข้อความ) [NOT safe]") {
		t.Errorf("ToolDefinition missing chat.send index entry:\n%s", desc)
	}
	if !strings.Contains(desc, "- settings.rail.brain (ต่อสมอง) [safe]") {
		t.Errorf("ToolDefinition missing settings.rail.brain index entry:\n%s", desc)
	}
}
