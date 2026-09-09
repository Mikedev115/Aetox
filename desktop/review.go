package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/learned"
	"github.com/Mikedev115/Aetox/internal/model"
)

// Post-session review: reflecting on finished or idle sessions to extract durable
// facts about the user and their preferences (Hermes ethos, docs/architecture/user-profile-and-habits-2026-09-06.md §4.3).
//
// Key principles:
// 1. Human-in-the-loop: Every fact is proposed into pending_changes (source='review');
//    nothing writes to disk without human approval.
// 2. Pre-emption: Can be cancelled immediately when the user sends a new message,
//    preventing queue contention on local models like Ollama.
// 3. User digest only: The model sees only the user's own messages, keeping cost and
//    tokens minimal (< 1 KB for typical sessions).

var (
	activeReviewMu     sync.Mutex
	activeReviewCancel context.CancelFunc
	reviewedSessions   sync.Map // sessionID -> bool
)

// cancelActiveReview terminates any ongoing background review immediately.
// Called from SendMessage before a live turn starts.
func (a *App) cancelActiveReview() {
	activeReviewMu.Lock()
	defer activeReviewMu.Unlock()
	if activeReviewCancel != nil {
		activeReviewCancel()
		activeReviewCancel = nil
	}
}

// setActiveReview arms the cancellation handle for an ongoing review.
func setActiveReview(cancel context.CancelFunc) {
	activeReviewMu.Lock()
	defer activeReviewMu.Unlock()
	if activeReviewCancel != nil {
		activeReviewCancel()
	}
	activeReviewCancel = cancel
}

// clearActiveReview clears the cancellation handle when a review completes.
func clearActiveReview() {
	activeReviewMu.Lock()
	defer activeReviewMu.Unlock()
	activeReviewCancel = nil
}

// ReviewFact is one extracted declarative fact about the user.
type ReviewFact struct {
	Text string `json:"text"`
	Why  string `json:"why"`
}

// sessionReviewer abstracts the LLM call for unit testing.
type sessionReviewer interface {
	Review(ctx context.Context, userMessages []string) ([]ReviewFact, error)
}

const sessionReviewInstructions = `You are an expert user-modeling reviewer for Aetox, applying the Hermes Agent ethos.
Your job is to review the user's messages in a conversation and extract durable, long-term facts about who the user is and how they want to be worked with.

Ask these two questions of the user's messages:
1. Has the user revealed things about themselves — persona, desires, preferences, role, or background?
2. Has the user expressed expectations about how you should behave, their work style, or ways they want you to operate?

Strict Rules:
- Write entries as DECLARATIVE FACTS about the user, never as imperative instructions to yourself:
  "User prefers concise answers and system settings over third-party tools" (✓)
  "Always answer concisely" (✗ - imperative phrasing gets re-read as a directive in later sessions that can override current requests).
- Skills come first: a preference about how to execute a specific procedure belongs in a skill, not user memory.
- If the user revealed nothing durable, return an empty facts array.
- Call the user_profile_proposals tool with the extracted facts.`

var sessionReviewTool = model.ToolDefinition{
	Type: "function",
	Function: model.ToolFunction{
		Name:        "user_profile_proposals",
		Description: "Propose durable declarative facts about the user for USER.md",
		Parameters: json.RawMessage(`{
			"type":"object",
			"properties":{
				"facts":{
					"type":"array",
					"items":{
						"type":"object",
						"properties":{
							"text":{"type":"string","description":"Declarative fact about the user (e.g. 'User prefers concise summaries before file changes')"},
							"why":{"type":"string","description":"Brief quote or evidence from the user's messages"}
						},
						"required":["text","why"]
					},
					"description":"List of durable facts about the user. Empty if nothing durable was revealed."
				}
			},
			"required":["facts"]
		}`),
	},
}

// appSessionReviewer calls the current active model using oneShotProvider.
type appSessionReviewer struct{ app *App }

func (r appSessionReviewer) Review(ctx context.Context, userMessages []string) ([]ReviewFact, error) {
	if len(userMessages) == 0 {
		return nil, nil
	}

	p, modelName, err := r.app.oneShotProvider()
	if err != nil {
		return nil, err
	}

	var digest strings.Builder
	for i, msg := range userMessages {
		digest.WriteString(fmt.Sprintf("User message %d: %s\n\n", i+1, msg))
	}

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: sessionReviewInstructions},
			{Role: model.RoleUser, Content: fmt.Sprintf("=== User Messages ===\n\n%s", digest.String())},
		},
		Tools:      []model.ToolDefinition{sessionReviewTool},
		ToolChoice: "required",
		MaxTokens:  1200,
	}

	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	var raw string
	if len(resp.ToolCalls) > 0 {
		raw = resp.ToolCalls[0].Function.Arguments
	} else {
		raw = extractJSONObject(resp.Text)
	}
	if raw == "" {
		return nil, nil
	}

	var out struct {
		Facts []ReviewFact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("failed to parse review facts: %w", err)
	}

	return out.Facts, nil
}

// getUserMessagesForSession fetches all user messages from SQLite for a session in order.
func getUserMessagesForSession(db *sql.DB, sessionID string) ([]string, error) {
	if db == nil || sessionID == "" {
		return nil, nil
	}

	var msgs []string
	err := eachRow(db, "review: reading user messages", `
		SELECT text FROM messages
		 WHERE session_id = ? AND role = 'user'
		 ORDER BY id ASC`,
		[]any{sessionID},
		func(rows *sql.Rows) error {
			var text string
			if err := rows.Scan(&text); err != nil {
				return err
			}
			text = strings.TrimSpace(text)
			if text != "" {
				msgs = append(msgs, text)
			}
			return nil
		},
	)
	return msgs, err
}

// runSessionReviewWith runs the review pipeline with a specified reviewer (real or fake).
func (a *App) runSessionReviewWith(ctx context.Context, reviewer sessionReviewer, sessionID string) (int, error) {
	if !learningEnabled() {
		return 0, fmt.Errorf("learning is switched off")
	}

	db, err := a.database()
	if err != nil {
		return 0, err
	}

	msgs, err := getUserMessagesForSession(db, sessionID)
	if err != nil {
		return 0, err
	}

	// Hermes rule: review needs at least 2 user turns to observe a pattern or preference
	if len(msgs) < 2 {
		return 0, nil
	}

	facts, err := reviewer.Review(ctx, msgs)
	if err != nil {
		return 0, err
	}

	proposed := 0
	for _, f := range facts {
		text := strings.TrimSpace(f.Text)
		if text == "" {
			continue
		}

		p := learned.Proposal{
			Kind:   kindMemory,
			Scope:  learned.UserScope, // Proposals land directly in USER.md (who the user is)
			Op:     learned.OpAdd,
			Body:   text,
			Reason: f.Why,
		}

		// Propose into pending_changes (source='review')
		res, err := a.proposeLearnedFromReview(p, sessionID)
		if err != nil {
			debuglog.Msg("review: propose failed for %q: %v", text, err)
			continue
		}
		if !res.Duplicate {
			proposed++
		}
	}

	reviewedSessions.Store(sessionID, true)
	if proposed > 0 {
		a.emitLearningChanged()
	}
	return proposed, nil
}

// proposeLearnedFromReview records a proposal with source='review'.
func (a *App) proposeLearnedFromReview(p learned.Proposal, sessionID string) (learned.Result, error) {
	db, err := a.database()
	if err != nil {
		return learned.Result{}, err
	}

	target := ""
	if path, err := learned.FileFor(p.Scope); err == nil {
		target = path
	}

	var existing int64
	err = db.QueryRow(`
		SELECT id FROM pending_changes
		 WHERE state = ? AND kind = ? AND scope = ? AND op = ? AND body = ?
		 LIMIT 1`,
		statePending, p.Kind, p.Scope, p.Op, p.Body).Scan(&existing)
	if err == nil {
		return learned.Result{ID: existing, Duplicate: true}, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	evidence := fmt.Sprintf("session:%s", sessionID)
	res, err := db.Exec(`
		INSERT INTO pending_changes(kind, scope, target, op, before, body, reason, evidence, source, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'review', ?, ?)`,
		p.Kind, p.Scope, target, p.Op, p.Before, p.Body, p.Reason, evidence, statePending, now)
	if err != nil {
		return learned.Result{}, err
	}
	id, _ := res.LastInsertId()
	return learned.Result{ID: id}, nil
}

var (
	idleTimersMu sync.Mutex
	idleTimers   = map[string]*time.Timer{}
)

// armSessionIdleReview arms a 5-minute idle timer to review the session after inactivity.
func (a *App) armSessionIdleReview(sessionID string) {
	if !learningEnabled() || !sessionReviewAuto() || sessionID == "" {
		return
	}
	idleTimersMu.Lock()
	defer idleTimersMu.Unlock()
	if t, ok := idleTimers[sessionID]; ok {
		t.Stop()
	}
	idleTimers[sessionID] = time.AfterFunc(5*time.Minute, func() {
		a.maybeReviewSession(sessionID)
	})
}

// maybeReviewSession triggers a background review when a session finishes/switches/idles.
func (a *App) maybeReviewSession(sessionID string) {
	if !learningEnabled() || !sessionReviewAuto() || sessionID == "" {
		return
	}

	// Skip if already reviewed in this process lifetime
	if _, done := reviewedSessions.Load(sessionID); done {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	setActiveReview(cancel)

	go func() {
		defer clearActiveReview()
		defer cancel()

		_, err := a.runSessionReviewWith(ctx, appSessionReviewer{a}, sessionID)
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			debuglog.Msg("maybeReviewSession: session %s review failed: %v", sessionID, err)
		}
	}()
}

// RunSessionReview allows manually triggering a session review (e.g. from UI button).
func (a *App) RunSessionReview(sessionID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return a.runSessionReviewWith(ctx, appSessionReviewer{a}, sessionID)
}

// sessionReviewAuto reads the stored preference.
func sessionReviewAuto() bool {
	pref, ok, _ := config.LoadModelPreference()
	if !ok {
		return false
	}
	return pref.SessionReviewAuto
}

// SessionReviewAuto reports the switch for Settings.
func (a *App) SessionReviewAuto() bool { return sessionReviewAuto() }

// SetSessionReviewAuto persists the switch for Settings.
func (a *App) SetSessionReviewAuto(on bool) error {
	return config.UpdateModelPreference(func(pref *config.ModelPreference) error {
		pref.SessionReviewAuto = on
		return nil
	})
}
