package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// askUserSkill lets the model pause mid-turn and ask the user to pick between
// concrete options (the Claude Code AskUserQuestion pattern). The tool blocks
// until the user clicks an option in the chat UI or the turn is canceled.
// One question in flight at a time — the tool loop is sequential anyway.
// conv is the chat this tool belongs to. Built per conversation in
// applyConfig, so a question raised by one chat's engine can only ever be
// asked, drawn and answered in that chat — the owner's instruction on
// 19 ส.ค.: *"ask_user หากมี ให้ขึ้นแจ้งเตือนที่เซสชั่นปกติเลยครับ"*.
type askUserSkill struct {
	app  *App
	conv *conversation
}

func (*askUserSkill) Name() string { return "ask_user" }

func (*askUserSkill) Description() string {
	return "ถามผู้ใช้ให้เลือกตัวเลือก แล้วรอคำตอบก่อนทำงานต่อ"
}

func (*askUserSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The question to ask, in the user's language. Ask only when genuinely blocked on a decision that is the user's to make.",
			},
			"options": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				// "plain strings" and "one question" are both here because both
				// were got wrong: the nested AskUserQuestion shape arrives
				// otherwise, and askArgs has to unpick it.
				"description": "2-4 short, mutually exclusive choices, as plain strings — not objects, and not a nested list of questions. The user can also type a free-text answer.",
			},
		},
		"required":             []string{"question", "options"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "ask_user",
			Description: "Ask the user ONE question and wait for the answer. Arguments are flat: a `question` string and an `options` array of plain strings. Use when clarifying ambiguous goals, choosing between concrete approaches, or when blocked on a decision only the user can make.",
			Parameters:  payload,
		},
	}
}

func (s *askUserSkill) ExecuteTool(ctx context.Context, args map[string]any) (skill.Output, error) {
	question, options := askArgs(args)
	return s.ask(ctx, question, options)
}

// askArgs reads the flat shape this tool declares — and the nested one models
// reach for anyway.
//
// Anthropic-trained models know AskUserQuestion, whose arguments nest:
// {questions:[{question, header, options:[{label, description}]}]}. Handed a
// field called "options", such a model puts that whole array in it, and what
// arrived was {"options":[{"options":["อยากรู้สเปก/ราคา MacBook Air...", ...]}]}
// — a question genuinely asked, with its choices, rejected for the packaging.
// Three times running, because "question is required" never named the shape it
// wanted (owner, 8 ก.ย. 2026).
//
// Refusing that costs the user an answer to a question they were never shown,
// so it is read rather than refused. The declared shape stays the flat one:
// this is a translation on the way in, not a second contract.
func askArgs(args map[string]any) (string, []string) {
	question := strings.TrimSpace(stringOf(args["question"]))
	options := optionLabels(args["options"])
	if question != "" && len(options) >= 2 {
		return question, options
	}
	// One level down, under whichever name the model used for the list.
	for _, key := range []string{"options", "questions"} {
		nestedQ, nestedOpts := nestedAsk(args[key])
		if question == "" {
			question = nestedQ
		}
		// Only when there is something down there: a flat list that is merely
		// too short must not be blanked by a lookup that found nothing.
		if len(options) < 2 && len(nestedOpts) > 0 {
			options = nestedOpts
		}
	}
	return question, options
}

// stringOf reads a string out of a JSON value and ignores anything else.
func stringOf(v any) string {
	s, _ := v.(string)
	return s
}

// optionLabels turns a JSON array into the strings the user will click. An
// entry is either a plain string (this tool's own shape) or an object carrying
// the label under one of the names the various question tools use — for
// AskUserQuestion's {label, description}, the label is the clickable half and
// the description is scaffolding this UI has nowhere to put.
func optionLabels(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		var label string
		switch t := item.(type) {
		case string:
			label = t
		case map[string]any:
			for _, key := range []string{"label", "text", "title", "option", "value", "answer"} {
				if label = strings.TrimSpace(stringOf(t[key])); label != "" {
					break
				}
			}
		}
		if label = strings.TrimSpace(label); label != "" {
			out = append(out, label)
		}
	}
	return out
}

// nestedAsk digs one level for a question packaged as a list of questions.
// Only the first is taken: this tool asks one thing at a time and the UI draws
// one card, so the rest are dropped rather than silently merged into one
// unanswerable prompt — the model re-asks whatever still matters once it has
// this answer.
func nestedAsk(v any) (string, []string) {
	raw, ok := v.([]any)
	if !ok {
		return "", nil
	}
	for _, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		question := strings.TrimSpace(stringOf(obj["question"]))
		if question == "" {
			// AskUserQuestion's `header` is a 12-character chip, not a
			// sentence — a poor question, but a far better card than none.
			question = strings.TrimSpace(stringOf(obj["header"]))
		}
		opts := optionLabels(obj["options"])
		if question != "" || len(opts) > 0 {
			return question, opts
		}
	}
	return "", nil
}

func (s *askUserSkill) Execute(ctx context.Context, input skill.Input) (skill.Output, error) {
	question, _ := input["question"].(string)
	return s.ask(ctx, strings.TrimSpace(question), nil)
}

func (s *askUserSkill) ask(ctx context.Context, question string, options []string) (skill.Output, error) {
	start := time.Now()
	fail := func(err error) (skill.Output, error) {
		return skill.Output{
			Name: "ask_user", Command: "ask_user", Success: false,
			Content: err.Error(), RawOutput: err.Error(), Stderr: err.Error(),
			DurationMs: time.Since(start).Milliseconds(),
		}, err
	}
	// Both messages spell the shape out. An error that only states what is
	// missing leaves a model that got the packaging wrong to retry the same
	// packaging, which is exactly what happened three times running.
	if question == "" {
		return fail(errors.New(`question is required — call it flat, one question per call: {"question": "...", "options": ["...", "..."]}`))
	}
	if len(options) < 2 {
		return fail(errors.New(`provide at least 2 options, as plain strings: {"question": "...", "options": ["ทำต่อ", "หยุดไว้ก่อน"]}`))
	}

	answerCh, err := s.app.beginUserQuestion(s.conv, question, options)
	if err != nil {
		return fail(err)
	}
	defer s.app.endUserQuestion(s.conv)

	select {
	case answer := <-answerCh:
		receipt := fmt.Sprintf("user chose: %s", answer)
		return skill.Output{
			Name: "ask_user", Command: "ask_user " + question, Success: true,
			Content: receipt, RawOutput: receipt,
			// The same answer again, as a field. `receipt` is a sentence written
			// for the model; this is for the transcript, which has to be able to
			// show what was asked and what was said back long after the card that
			// asked it is gone.
			Answer:     answer,
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	case <-ctx.Done():
		return fail(ctx.Err())
	}
}

// approvalAllow / approvalDeny are the exact option strings approveToolCall
// sends to the UI and compares the answer against. Anything else the user
// types back — including free text — reads as a refusal.
const (
	approvalAllow = "อนุญาต / Allow"
	approvalDeny  = "ปฏิเสธ / Deny"
)

// approveToolCall is the desktop's turn.ApprovalPromptFunc: the same in-chat
// question UI ask_user uses, asked by the engine instead of the model. It
// exists because the engine's default approver is a console y/N prompt, and a
// windowsgui build has no console — reading its stdin fails instantly, which
// is how every prompting tool call used to die with "read /dev/stdin: The
// handle is invalid" before anyone saw a question.
func (a *App) approveToolCall(conv *conversation, ctx context.Context, command, reason string) (bool, error) {
	reason = strings.TrimSpace(reason)
	question := fmt.Sprintf("ขออนุญาตรัน: `%s`", command)
	if reason != "" {
		question += fmt.Sprintf(" — %s", reason)
	}
	ch, err := a.beginUserQuestion(conv, question, []string{approvalAllow, approvalDeny})
	if err != nil {
		return false, err
	}
	defer a.endUserQuestion(conv)
	select {
	case answer := <-ch:
		return answer == approvalAllow, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// beginUserQuestion registers the single in-flight question and pushes it to
// the chat UI. Fails loudly if one is already pending.
func (a *App) beginUserQuestion(conv *conversation, question string, options []string) (chan string, error) {
	a.askMu.Lock()
	defer a.askMu.Unlock()
	if conv.askCh != nil {
		return nil, errors.New("another question is already awaiting the user")
	}
	conv.askCh = make(chan string, 1)
	// One question per chat, not one per app. Two conversations working at once
	// can each be waiting on their own, and neither sees the other's — which is
	// only true because the channel is the conversation's and the card is
	// addressed to it.
	a.emitEvent("ask:user", sessionEvent[map[string]any]{SessionID: conv.id, Data: map[string]any{
		"question": question,
		"options":  options,
	}})
	return conv.askCh, nil
}

func (a *App) endUserQuestion(conv *conversation) {
	a.askMu.Lock()
	defer a.askMu.Unlock()
	conv.askCh = nil
	a.emitEvent("ask:done", sessionEvent[any]{SessionID: conv.id})
}

// AnswerUserQuestion delivers the user's choice to the blocked ask_user tool
// call. No-op when nothing is pending (e.g. a stale click after cancel).
// The session is named rather than assumed. The card is drawn in the chat that
// raised it, so the click that answers it comes from that chat — and with two
// conversations able to be waiting at once, "the one on screen" would deliver
// one chat's answer into the other's blocked tool call.
func (a *App) AnswerUserQuestion(sessionID, answer string) {
	conv := a.convs.find(sessionID)
	if conv == nil && sessionID == a.cur().id {
		conv = a.cur()
	}
	if conv == nil {
		return
	}
	a.askMu.Lock()
	defer a.askMu.Unlock()
	if conv.askCh == nil {
		return
	}
	select {
	case conv.askCh <- strings.TrimSpace(answer):
	default: // already answered
	}
}

// todoWriteSkill is the Claude Code TodoWrite pattern: on long multi-step
// work the model maintains a task checklist the user can watch. Each call
// replaces the whole list; state lives in the frontend only.
// ponytail: not persisted with the session — store it in SessionMessage land
// if surviving a reload ever matters.
//
// conv is the chat whose plan this is, so the event can say so. It went out
// unstamped for as long as one chat could work at a time; with several, an
// unstamped list landed on whichever plan panel was on screen — a background
// chat re-planning overwrote the checklist the user was reading.
type todoWriteSkill struct {
	app  *App
	conv *conversation
}

func (*todoWriteSkill) Name() string { return "todo_write" }

func (*todoWriteSkill) Description() string {
	return "อัปเดตรายการสิ่งที่ต้องทำของงานปัจจุบันให้ผู้ใช้เห็น"
}

func (*todoWriteSkill) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"todos": map[string]any{
				"type":        "array",
				"description": "The full task list (replaces the previous list). Keep one item in_progress at a time; mark items completed as soon as they are done.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content":    map[string]any{"type": "string", "description": "Short task description in the user's language"},
						"status":     map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed"}},
						"activeForm": map[string]any{"type": "string", "description": "The same task worded as what is happening right now — \"reading the config\" rather than \"read the config\". Shown while the item is in_progress, in the user's language."},
					},
					"required":             []string{"content", "status"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"todos"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "todo_write",
			Description: "Maintain a visible task checklist for large or long-running work: call it when starting multi-step work, and again whenever a step's status changes. Sends the FULL list each call.",
			Parameters:  payload,
		},
	}
}

// todoItem is one checklist row as the frontend draws it. Package-level (it
// was local to ExecuteTool) so the stamped event type can name it.
type todoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"`
	// ActiveForm is what the row reads while the step is running. Optional:
	// a model that omits it leaves the UI showing Content, which is what
	// happened before this field existed.
	ActiveForm string `json:"activeForm,omitempty"`
}

func (s *todoWriteSkill) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	start := time.Now()
	raw, _ := args["todos"].([]any)
	items := make([]todoItem, 0, len(raw))
	for _, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		content, _ := m["content"].(string)
		status, _ := m["status"].(string)
		activeForm, _ := m["activeForm"].(string)
		content = strings.TrimSpace(content)
		activeForm = strings.TrimSpace(activeForm)
		switch status {
		case "pending", "in_progress", "completed":
		default:
			status = "pending"
		}
		if content == "" {
			continue
		}
		items = append(items, todoItem{Content: content, Status: status, ActiveForm: activeForm})
	}
	// Stamped with the conversation like every other agent event; nil conv is
	// the test seam and degrades to the old unstamped shape the frontend still
	// accepts.
	if s.app.ctx != nil || s.app.emit != nil {
		if s.conv != nil {
			s.app.emitEvent("todo:update", sessionEvent[[]todoItem]{SessionID: s.conv.id, Data: items})
		} else {
			s.app.emitEvent("todo:update", items)
		}
	}
	doneCount := 0
	for _, it := range items {
		if it.Status == "completed" {
			doneCount++
		}
	}
	receipt := fmt.Sprintf("todo list updated: %d items, %d completed", len(items), doneCount)
	return skill.Output{
		Name: "todo_write", Command: "todo_write", Success: true,
		Content: receipt, RawOutput: receipt,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (s *todoWriteSkill) Execute(ctx context.Context, input skill.Input) (skill.Output, error) {
	return s.ExecuteTool(ctx, map[string]any{"todos": input["todos"]})
}
