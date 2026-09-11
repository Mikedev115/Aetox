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
// 3. User digest, plus the user's decisions: the model sees the user's own
//    messages of the session, and what the user has already said yes and no to.
//
// The third principle used to end at "only": the design (§4.3 of the doc
// above) had the reviewer read nothing but the session, and the store measured
// what that costs (11 ก.ย.). Fifty proposals from this source, two approved.
// "User communicates in Thai" was proposed six times in six wordings and
// refused six times; the owner's GitHub handle, approved on the 8th, was
// proposed again on the 11th and refused for being there already. A reviewer
// that cannot see the profile cannot know what is in it, and one that cannot
// see the refusals cannot learn the bar — so every session it re-derived the
// same obvious facts from the same kind of evidence. It now reads USER.md as
// it stands, the lines waiting on the page, and the lines turned down, and is
// told what each list means. The mechanical door (queueMemoryProposal) still
// stands behind it for the restatements a model produces anyway.

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

// reviewInput is everything the reviewer reads: the session's user messages,
// and the three lists that say what has already been decided.
type reviewInput struct {
	Messages []string
	// Profile is USER.md as it stands — every line in it was approved. It is
	// the bar shown by example, and the list of what not to propose again.
	Profile string
	// Pending are the lines already waiting for a decision.
	Pending []string
	// Rejected are the lines the user turned down, newest first, capped. A
	// fact refused once is refused in every rewording.
	Rejected []string
}

// sessionReviewer abstracts the LLM call for unit testing.
type sessionReviewer interface {
	Review(ctx context.Context, in reviewInput) ([]ReviewFact, error)
}

// reviewRejectedShown caps the refusals the reviewer reads. The store had
// eighty-odd on 11 ก.ย.; the newest thirty are the current bar, and older ones
// are still caught by the door behind this pass if a model restates them.
const reviewRejectedShown = 30

const sessionReviewInstructions = `You are a User Profile Reviewer. Your job is to extract ONLY permanent user facts and enduring instructions from the conversation.

Rules:
1. Language: Must strictly match the language of the conversation.
2. Filter: Extract ONLY immutable facts (e.g. role, tech stack, environment) and explicit permanent rules (e.g. "always do X"). Write as declarative facts about the user.
3. Strictly ignore: Ephemeral/one-off tasks, emotions, personality quirks, and guesses. The language the user writes in, their tone, their typos and how long their messages are are never facts — the assistant already answers in the user's language.
4. Already decided: You are shown the profile as it stands (approved), what is waiting for a decision, and what the user turned down. Do not propose anything in those lists, nor the same fact in other words. What was approved shows the bar; what was turned down shows what falls under it.
5. Output: Call user_profile_proposals. If no NEW durable facts exist, return an empty array. Be conservative.`

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
							"text":{"type":"string","description":"Durable fact about the user in the conversation's language"},
							"why":{"type":"string","description":"Brief quote or evidence from the user's messages in the conversation's language"}
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

func (r appSessionReviewer) Review(ctx context.Context, in reviewInput) ([]ReviewFact, error) {
	if len(in.Messages) == 0 {
		return nil, nil
	}

	p, modelName, err := r.app.oneShotProvider()
	if err != nil {
		return nil, err
	}

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: sessionReviewInstructions},
			{Role: model.RoleUser, Content: reviewDigest(in)},
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

// reviewDigest is the user turn the reviewer reads: what was already decided
// first, so the session's messages are read against it, then the messages.
// Each decided list is written only when it has something in it — a fresh
// install's reviewer reads exactly the digest it read before the lists
// existed.
func reviewDigest(in reviewInput) string {
	var b strings.Builder
	if strings.TrimSpace(in.Profile) != "" {
		b.WriteString("=== Profile as it stands (approved by the user — do not propose these again) ===\n\n")
		b.WriteString(strings.TrimSpace(in.Profile))
		b.WriteString("\n\n")
	}
	writeList := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		b.WriteString("=== " + title + " ===\n\n")
		for _, line := range lines {
			b.WriteString("- " + line + "\n")
		}
		b.WriteString("\n")
	}
	writeList("Waiting for the user's decision (do not propose again)", in.Pending)
	writeList("Turned down by the user (do not propose again, in any wording)", in.Rejected)
	b.WriteString("=== User Messages ===\n\n")
	for i, msg := range in.Messages {
		b.WriteString(fmt.Sprintf("User message %d: %s\n\n", i+1, msg))
	}
	return b.String()
}

// decidedMemoryLines reads the queue's own record for the reviewer: what is
// waiting and what was refused, for the memory kind, newest first. Every
// scope, not only the profile's — a machine fact the user refused is the same
// signal about the bar whichever file it was headed for.
func (a *App) decidedMemoryLines(state string, limit int) []string {
	db, err := a.database()
	if err != nil {
		return nil
	}
	var out []string
	_ = eachRow(db, "review: reading decided lines", `
		SELECT body FROM pending_changes WHERE kind = ? AND state = ?
		  ORDER BY id DESC LIMIT `+fmt.Sprint(limit), []any{kindMemory, state},
		func(rows *sql.Rows) error {
			var body string
			if err := rows.Scan(&body); err != nil {
				return err
			}
			out = append(out, body)
			return nil
		})
	return out
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

	facts, err := reviewer.Review(ctx, reviewInput{
		Messages: msgs,
		Profile:  learned.Read(learned.UserScope),
		Pending:  a.decidedMemoryLines(statePending, 200),
		Rejected: a.decidedMemoryLines(stateRejected, reviewRejectedShown),
	})
	if err != nil {
		return 0, err
	}

	proposed := 0
	for _, f := range facts {
		text := strings.TrimSpace(f.Text)
		if text == "" {
			continue
		}

		// Don't queue proposals if memory scope has no room to apply them
		if learned.Full(learned.UserScope, len(text)+2) {
			debuglog.Msg("review: skipping proposal, %s is full: %q", learned.UserScope, text)
			continue
		}

		p := learned.Proposal{
			Kind:   kindMemory,
			Scope:  learned.UserScope, // Proposals land directly in USER.md (who the user is)
			Op:     learned.OpAdd,
			Body:   text,
			Reason: f.Why,
		}

		// Through the same door as the agent's own proposals, which is what
		// answers "already waiting" and "already decided" for both. A fact the
		// model restated despite the lists above stops here and is logged as
		// what it was, not counted as proposed.
		res, err := a.queueMemoryProposal(p, "review", "session:"+sessionID)
		if err != nil {
			debuglog.Msg("review: propose failed for %q: %v", text, err)
			continue
		}
		if res.Prior != nil {
			debuglog.Msg("review: %q restates #%d (%s), not queued", text, res.Prior.ID, res.Prior.State)
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
