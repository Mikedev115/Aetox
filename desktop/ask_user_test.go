package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ask_user must reject calls that would render an unanswerable prompt.
func TestAskUserValidation(t *testing.T) {
	s := &askUserSkill{app: NewApp(), conv: newConversation()}

	if _, err := s.ExecuteTool(context.Background(), map[string]any{
		"options": []any{"a", "b"},
	}); err == nil || !strings.Contains(err.Error(), "question") {
		t.Fatalf("missing question must fail loudly, got %v", err)
	}

	if _, err := s.ExecuteTool(context.Background(), map[string]any{
		"question": "pick one",
		"options":  []any{"only"},
	}); err == nil || !strings.Contains(err.Error(), "2 options") {
		t.Fatalf("fewer than 2 options must fail loudly, got %v", err)
	}
}

// The full round-trip: the tool blocks until AnswerUserQuestion delivers the
// user's choice, then reports it to the model.
func TestAskUserAnswerRoundTrip(t *testing.T) {
	app := &App{}
	s := &askUserSkill{app: app, conv: app.cur()}

	type result struct {
		content string
		// The answer as a FIELD, which is the half the transcript reads. The
		// receipt below is a sentence written for the model; a row that had to
		// parse "user chose: B" back apart to say what was chosen would be
		// exactly the mistake ToolEvent.Links exists to have stopped.
		answer string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		out, err := s.ExecuteTool(context.Background(), map[string]any{
			"question": "which one?",
			"options":  []any{"A", "B"},
		})
		done <- result{out.Content, out.Answer, err}
	}()

	// Wait until the question is registered, then answer as the user would.
	deadline := time.After(2 * time.Second)
	for {
		app.askMu.Lock()
		pending := app.cur().askCh != nil
		app.askMu.Unlock()
		if pending {
			break
		}
		select {
		case <-deadline:
			t.Fatal("question was never registered")
		case <-time.After(5 * time.Millisecond):
		}
	}
	app.AnswerUserQuestion(app.cur().id, "B")

	r := <-done
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if !strings.Contains(r.content, "user chose: B") {
		t.Fatalf("answer must reach the model receipt, got %q", r.content)
	}
	// And the transcript's half of it. Without this the timeline row is the only
	// record the exchange leaves and it says neither what was asked nor what was
	// answered (owner, 7 ก.ย.).
	if r.answer != "B" {
		t.Fatalf("Output.Answer = %q, want %q — the row has no way to say what was chosen", r.answer, "B")
	}
	// The slot must be free again for the next question.
	app.askMu.Lock()
	defer app.askMu.Unlock()
	if app.cur().askCh != nil {
		t.Fatal("ask slot must be cleared after the answer")
	}
}

// Turn cancellation (Stop button) must unblock a waiting question.
func TestAskUserCancelUnblocks(t *testing.T) {
	app := &App{}
	s := &askUserSkill{app: app, conv: app.cur()}
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := s.ExecuteTool(ctx, map[string]any{
			"question": "still there?",
			"options":  []any{"A", "B"},
		})
		errCh <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("canceled question must return an error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not unblock the question")
	}
}

// Only one question may be in flight — a second concurrent ask fails loudly
// instead of silently queueing.
func TestAskUserSecondQuestionFailsWhilePending(t *testing.T) {
	app := &App{}
	if _, err := app.beginUserQuestion(app.cur(), "first", []string{"a", "b"}); err != nil {
		t.Fatalf("first question must register: %v", err)
	}
	defer app.endUserQuestion(app.cur())
	if _, err := app.beginUserQuestion(app.cur(), "second", []string{"a", "b"}); err == nil {
		t.Fatal("second concurrent question must fail")
	}
}

// A stale answer (after cancel/completion) must be a no-op, not a panic.
func TestAnswerUserQuestionNoPendingIsNoop(t *testing.T) {
	app := &App{}
	app.AnswerUserQuestion(app.cur().id, "stale click") // must not panic
}

// waitForPendingQuestion blocks until beginUserQuestion has registered a
// question, or fails the test after 2s.
func waitForPendingQuestion(t *testing.T, app *App) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		app.askMu.Lock()
		pending := app.cur().askCh != nil
		app.askMu.Unlock()
		if pending {
			return
		}
		select {
		case <-deadline:
			t.Fatal("question was never registered")
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// approveToolCall must approve only the exact allow option — the deny option
// and free text both read as refusal.
func TestApproveToolCallAllowAndDeny(t *testing.T) {
	for _, tc := range []struct {
		answer string
		want   bool
	}{
		{approvalAllow, true},
		{approvalDeny, false},
		{"y", false}, // free text is not consent
	} {
		app := &App{}
		type result struct {
			ok  bool
			err error
		}
		done := make(chan result, 1)
		go func() {
			ok, err := app.approveToolCall(app.cur(), context.Background(), "shell rm -rf x", "may delete state")
			done <- result{ok, err}
		}()
		waitForPendingQuestion(t, app)
		app.AnswerUserQuestion(app.cur().id, tc.answer)
		r := <-done
		if r.err != nil {
			t.Fatalf("answer %q: unexpected error: %v", tc.answer, r.err)
		}
		if r.ok != tc.want {
			t.Errorf("answer %q: approved = %v, want %v", tc.answer, r.ok, tc.want)
		}
	}
}

// Turn cancellation (Stop button) must unblock a waiting approval as a denial.
func TestApproveToolCallCancelDenies(t *testing.T) {
	app := &App{}
	ctx, cancel := context.WithCancel(context.Background())
	type result struct {
		ok  bool
		err error
	}
	done := make(chan result, 1)
	go func() {
		ok, err := app.approveToolCall(app.cur(), ctx, "shell sleep 999", "")
		done <- result{ok, err}
	}()
	waitForPendingQuestion(t, app)
	cancel()
	select {
	case r := <-done:
		if r.ok || r.err == nil {
			t.Fatalf("canceled approval must deny with an error, got ok=%v err=%v", r.ok, r.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not unblock the approval")
	}
}

// todo_write sanitizes junk input and reports honest counts.
func TestTodoWriteSanitizesAndCounts(t *testing.T) {
	s := &todoWriteSkill{app: &App{}}
	out, err := s.ExecuteTool(context.Background(), map[string]any{
		"todos": []any{
			map[string]any{"content": "task one", "status": "completed"},
			map[string]any{"content": "task two", "status": "in_progress"},
			map[string]any{"content": "task three", "status": "bogus-status"}, // → pending
			map[string]any{"content": "   ", "status": "pending"},             // dropped
			"not an object", // dropped
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Success {
		t.Fatal("todo_write must succeed")
	}
	if !strings.Contains(out.Content, "3 items, 1 completed") {
		t.Fatalf("receipt must count sanitized items, got %q", out.Content)
	}
}

// An empty list is valid — it clears the checklist.
func TestTodoWriteEmptyListClears(t *testing.T) {
	s := &todoWriteSkill{app: &App{}}
	out, err := s.ExecuteTool(context.Background(), map[string]any{"todos": []any{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.Content, "0 items, 0 completed") {
		t.Fatalf("empty list receipt wrong: %q", out.Content)
	}
}

// The shape a model actually sent, three calls running, each rejected for
// having no question — while carrying one.
//
// Anthropic-trained models know AskUserQuestion, whose arguments nest a list of
// questions, each with its own options. Handed a field named "options", the
// model put that list in it. Nothing was missing; it was one level down.
func TestAskUserReadsTheNestedQuestionShape(t *testing.T) {
	// Verbatim from the failure card (owner, 8 ก.ย. 2026), minus the truncation.
	question, options := askArgs(map[string]any{
		"options": []any{
			map[string]any{
				"question": "จะให้ช่วยเรื่อง MacBook Air แบบไหนดีครับ",
				"header":   "MacBook Air",
				"options": []any{
					"อยากรู้สเปก/ราคา MacBook Air รุ่นล่าสุด เทียบรุ่น",
					"กำลังจะซื้อ อยากได้คำแนะนำเลือกรุ่น",
				},
			},
		},
	})
	if question != "จะให้ช่วยเรื่อง MacBook Air แบบไหนดีครับ" {
		t.Fatalf("question not found one level down: %q", question)
	}
	if len(options) != 2 || options[0] != "อยากรู้สเปก/ราคา MacBook Air รุ่นล่าสุด เทียบรุ่น" {
		t.Fatalf("options not found one level down: %#v", options)
	}
}

// AskUserQuestion's own field names, used verbatim: `questions` at the top and
// {label, description} for each choice. The label is the clickable half.
func TestAskUserReadsLabelledOptions(t *testing.T) {
	question, options := askArgs(map[string]any{
		"questions": []any{
			map[string]any{
				"question": "ลบไฟล์เก่าทิ้งไหม",
				"options": []any{
					map[string]any{"label": "ลบเลย", "description": "กู้คืนไม่ได้"},
					map[string]any{"label": "เก็บไว้ก่อน", "description": "ย้ายไป .bak"},
				},
			},
		},
	})
	if question != "ลบไฟล์เก่าทิ้งไหม" {
		t.Fatalf("question: %q", question)
	}
	if len(options) != 2 || options[0] != "ลบเลย" || options[1] != "เก็บไว้ก่อน" {
		t.Fatalf("labels not unwrapped: %#v", options)
	}
}

// A question with only a `header` is a 12-character chip, not a sentence — a
// poor card, and still a far better one than a third identical rejection.
func TestAskUserFallsBackToTheHeader(t *testing.T) {
	question, options := askArgs(map[string]any{
		"options": []any{
			map[string]any{"header": "ธีม", "options": []any{"สว่าง", "มืด"}},
		},
	})
	if question != "ธีม" || len(options) != 2 {
		t.Fatalf("header fallback: %q %#v", question, options)
	}
}

// The flat shape is still the declared one, and unpicking the nested shape must
// not touch it.
func TestAskUserKeepsTheFlatShape(t *testing.T) {
	question, options := askArgs(map[string]any{
		"question": "เอาไหม",
		"options":  []any{" เอา ", "ไม่เอา", "   "},
	})
	if question != "เอาไหม" {
		t.Fatalf("question: %q", question)
	}
	// Trimmed, and the blank entry dropped rather than drawn as a dead button.
	if len(options) != 2 || options[0] != "เอา" {
		t.Fatalf("options: %#v", options)
	}
}

// Nothing usable is still a refusal — and now the refusal says what the shape
// should have been, which is the part that was missing.
func TestAskUserStillRefusesWhenThereIsNoQuestion(t *testing.T) {
	s := &askUserSkill{app: NewApp(), conv: newConversation()}
	_, err := s.ExecuteTool(context.Background(), map[string]any{
		"options": []any{map[string]any{"nothing": "useful"}},
	})
	if err == nil || !strings.Contains(err.Error(), `"question"`) {
		t.Fatalf("the error must name the shape it wants, got %v", err)
	}
}
