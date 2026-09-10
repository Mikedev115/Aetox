package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/learned"
	"github.com/Mikedev115/Aetox/internal/model"
)

// HabitProposal represents an automated capability synthesized from repeated work.
type HabitProposal struct {
	Destination string `json:"destination"` // "skill", "user_profile", or "machine_memory"
	SkillName   string `json:"skill_name"`   // slug for skill if destination == "skill"
	Title       string `json:"title"`        // short human summary for the proposal card
	Body        string `json:"body"`         // SKILL.md markdown or declarative fact
	Reason      string `json:"reason"`       // evidence citing observed sessions
}

const habitSynthesisInstructions = `You are Aetox's Habit & Workflow Synthesizer.
Your role is to analyze user requests and the actions/tools executed across multiple sessions (or a single repeated session).
Your goal is to decide whether this repeated pattern should become an automated Skill or a durable Memory, and propose it cleanly.

Rules:
1. Language: Must strictly match the language of the conversation/user (e.g. Thai if user speaks Thai).
2. Decision Rules:
   - "skill": If this pattern represents a procedure, script, multi-step workflow, or recurring tool task (e.g. GPU check, git pull-and-test, data conversions).
     Draft a complete, self-contained SKILL.md content with YAML frontmatter:
     ---
     name: <slug>
     description: <short description in the user's language>
     ---
     # <Title>
     <Concise step-by-step instructions and command recipes>
   - "user_profile": Extract ONLY immutable facts (e.g. role, tech stack, environment) and explicit permanent working rules. Write as a declarative fact about the user in their language. Strictly ignore ephemeral/one-off tasks, emotions, personality quirks, and guesses.
   - "machine_memory": Permanent, non-negotiable hardware or environment constraint (e.g. hardware limits, global proxy). Write as a declarative fact in the user's language for MEMORY.md.
3. Be conservative: Propose only if the user explicitly asked or a genuinely durable, recurring pattern exists. If there is nothing durable or worth automating, do not call the tool.`

var habitSynthesisTool = model.ToolDefinition{
	Type: "function",
	Function: model.ToolFunction{
		Name:        "habit_proposal",
		Description: "Propose an automated skill, user profile memory, or environment memory based on observed habits",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"destination": {
					"type": "string",
					"enum": ["skill", "user_profile", "machine_memory"],
					"description": "Where this habit belongs: 'skill' for workflows/procedures, 'user_profile' for user preferences, 'machine_memory' for environment"
				},
				"skill_name": {
					"type": "string",
					"description": "Slug for skill name (e.g. check-gpu-power) if destination is skill"
				},
				"title": {
					"type": "string",
					"description": "User-facing summary headline in the conversation's language"
				},
				"body": {
					"type": "string",
					"description": "The SKILL.md content (for skill) or declarative fact in the conversation's language (for user_profile/machine_memory)"
				},
				"reason": {
					"type": "string",
					"description": "Evidence and reasoning citing what was observed across the sessions in the conversation's language"
				}
			},
			"required": ["destination", "title", "body", "reason"]
		}`),
	},
}

// habitSynthesizer abstracts the model call for testing.
type habitSynthesizer interface {
	Synthesize(ctx context.Context, digest, hint string) (*HabitProposal, error)
}

type appHabitSynthesizer struct {
	app *App
}

func (s appHabitSynthesizer) Synthesize(ctx context.Context, digest, hint string) (*HabitProposal, error) {
	p, modelName, err := s.app.oneShotProvider()
	if err != nil {
		return nil, err
	}

	userPrompt := fmt.Sprintf("=== Observed Sessions and Actions ===\n\n%s", digest)
	if hint != "" {
		userPrompt += fmt.Sprintf("\n\n[User Hint / Context: %s]", hint)
	}

	req := model.Request{
		Model: modelName,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: habitSynthesisInstructions},
			{Role: model.RoleUser, Content: userPrompt},
		},
		Tools:      []model.ToolDefinition{habitSynthesisTool},
		ToolChoice: "required",
		MaxTokens:  2000,
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

	var prop HabitProposal
	if err := json.Unmarshal([]byte(raw), &prop); err != nil {
		return nil, fmt.Errorf("failed to parse habit proposal: %w", err)
	}
	return &prop, nil
}

// buildSessionsDigest gathers messages and tool runs for the given session IDs.
func buildSessionsDigest(db *sql.DB, sessionIDs []string) (string, error) {
	if db == nil || len(sessionIDs) == 0 {
		return "", nil
	}

	var sb strings.Builder
	for i, sid := range sessionIDs {
		fmt.Fprintf(&sb, "--- Session %d (ID: %s) ---\n", i+1, sid)

		// 1. User messages
		var userMsgs []string
		_ = eachRow(db, "habit: reading user messages", `
			SELECT text FROM messages
			 WHERE session_id = ? AND role = 'user'
			 ORDER BY id ASC`,
			[]any{sid},
			func(rows *sql.Rows) error {
				var text string
				if err := rows.Scan(&text); err == nil && strings.TrimSpace(text) != "" {
					userMsgs = append(userMsgs, strings.TrimSpace(text))
				}
				return nil
			},
		)
		for _, m := range userMsgs {
			fmt.Fprintf(&sb, "User: %s\n", m)
		}

		// 2. Tool runs
		var actions []string
		_ = eachRow(db, "habit: reading tool runs", `
			SELECT tool, args, output FROM tool_runs
			 WHERE session_id = ?
			 ORDER BY id ASC LIMIT 10`,
			[]any{sid},
			func(rows *sql.Rows) error {
				var tool, args, output string
				if err := rows.Scan(&tool, &args, &output); err == nil {
					// Compact args & output for digest
					args = strings.ReplaceAll(args, "\n", " ")
					if len(args) > 100 {
						args = args[:100] + "..."
					}
					output = strings.ReplaceAll(output, "\n", " ")
					if len(output) > 120 {
						output = output[:120] + "..."
					}
					actions = append(actions, fmt.Sprintf("Action: %s(%s) -> %s", tool, args, output))
				}
				return nil
			},
		)
		for _, a := range actions {
			fmt.Fprintf(&sb, "%s\n", a)
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String()), nil
}

// SynthesizeHabitForSessions runs synthesis over a set of sessions and queues the proposal.
func (a *App) SynthesizeHabitForSessions(ctx context.Context, synthesizer habitSynthesizer, sessionIDs []string, normalizedKey, hint string) (*PendingChange, error) {
	if !learningEnabled() || synthesizer == nil {
		return nil, fmt.Errorf("learning is switched off")
	}

	db, err := a.database()
	if err != nil {
		return nil, err
	}

	digest, err := buildSessionsDigest(db, sessionIDs)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(digest) == "" && hint == "" {
		return nil, fmt.Errorf("no session content found for synthesis")
	}

	prop, err := synthesizer.Synthesize(ctx, digest, hint)
	if err != nil {
		return nil, err
	}
	if prop == nil || strings.TrimSpace(prop.Body) == "" {
		return nil, nil
	}

	var kind string
	var scope string
	var op string
	target := ""

	switch prop.Destination {
	case "skill":
		kind = kindSkill
		skillSlug := strings.TrimSpace(prop.SkillName)
		if skillSlug == "" {
			skillSlug = "custom-workflow"
		}
		// Clean slug
		skillSlug = strings.ToLower(strings.ReplaceAll(skillSlug, " ", "-"))
		scope = skillSlug
		target = skillSlug
		op = "create"
	case "machine_memory":
		kind = kindMemory
		scope = learned.MainScope
		op = learned.OpAdd
	default: // user_profile
		kind = kindMemory
		scope = learned.UserScope
		op = learned.OpAdd
	}

	reasonText := strings.TrimSpace(prop.Reason)
	if prop.Title != "" {
		reasonText = fmt.Sprintf("[%s] %s", prop.Title, reasonText)
	}

	change := PendingChange{
		Kind:      kind,
		Scope:     scope,
		Target:    target,
		Op:        op,
		Body:      strings.TrimSpace(prop.Body),
		Reason:    reasonText,
		Source:    "habit_synthesis",
		State:     statePending,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	res, err := db.Exec(`
		INSERT INTO pending_changes (kind, scope, target, op, body, reason, source, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		change.Kind, change.Scope, change.Target, change.Op, change.Body, change.Reason, change.Source, change.State, change.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record habit proposal: %w", err)
	}

	id, _ := res.LastInsertId()
	change.ID = id

	// Record in synthesized_habits if normalized key is provided
	if normalizedKey != "" {
		_, _ = db.Exec(`
			INSERT OR REPLACE INTO synthesized_habits (normalized, synthesized_at, proposal_id)
			VALUES (?, ?, ?)`,
			normalizedKey, time.Now().Format(time.RFC3339), id,
		)
	}

	a.emitLearningChanged()
	return &change, nil
}

// autoSynthesizeHabits lets tests turn the background habit synthesizer off.
var autoSynthesizeHabits = true

// maybeSynthesizeHabits is triggered post-turn from jobs.go.
func (a *App) maybeSynthesizeHabits() {
	if !autoSynthesizeHabits || !learningEnabled() {
		return
	}
	db, err := a.database()
	if err != nil {
		return
	}

	// 1. Detect recurring requests hitting threshold of 3
	recurring := detectRecurringRequests(db, 3, 100)
	if len(recurring) == 0 {
		return
	}

	// 2. Find first candidate that has not been synthesized yet
	var targetReq *RecurringRequest
	for i := range recurring {
		norm := recurring[i].Normalized
		var exists int
		if err := db.QueryRow(`SELECT 1 FROM synthesized_habits WHERE normalized = ?`, norm).Scan(&exists); err == sql.ErrNoRows {
			targetReq = &recurring[i]
			break
		}
	}
	if targetReq == nil {
		return
	}

	// Concurrency guard
	if !a.habitSynthesisRunning.CompareAndSwap(false, true) {
		return
	}

	go func(req RecurringRequest) {
		defer a.habitSynthesisRunning.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		_, err := a.SynthesizeHabitForSessions(ctx, appHabitSynthesizer{a}, req.SessionIDs, req.Normalized, "")
		if err != nil {
			debuglog.Msg("habit_synthesis: failed for %q: %v", req.Normalized, err)
		}
	}(*targetReq)
}

// SynthesizeHabit is the manual/on-demand trigger exposed to Wails.
// Even if count < 3 or user explicitly requests it, this performs synthesis immediately.
func (a *App) SynthesizeHabit(sessionID string, hint string) (int64, error) {
	if !learningEnabled() {
		return 0, fmt.Errorf("learning is switched off")
	}

	db, err := a.database()
	if err != nil {
		return 0, err
	}

	sessionIDs := []string{}
	if sessionID != "" {
		sessionIDs = append(sessionIDs, sessionID)
	}

	// If sessionID is provided, check if it belongs to any detected recurring cluster
	var normKey string
	if sessionID != "" {
		all := detectRecurringRequests(db, 2, 100)
		for _, req := range all {
			for _, sid := range req.SessionIDs {
				if sid == sessionID {
					sessionIDs = req.SessionIDs
					normKey = req.Normalized
					break
				}
			}
			if len(sessionIDs) > 1 {
				break
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	change, err := a.SynthesizeHabitForSessions(ctx, appHabitSynthesizer{a}, sessionIDs, normKey, hint)
	if err != nil {
		return 0, err
	}
	if change == nil {
		return 0, nil
	}
	return change.ID, nil
}
