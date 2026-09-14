package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/learned"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/skilllint"
)

// HabitProposal represents an automated capability synthesized from repeated work.
type HabitProposal struct {
	Destination string `json:"destination"` // "skill", "user_profile", or "machine_memory"
	SkillName   string `json:"skill_name"`  // slug for skill if destination == "skill"
	// Extends names an installed skill this belongs in, and turns the proposal
	// from a new SKILL.md into a section appended to that one (op "add").
	// Added 2026-09-14 after the synthesizer, which never saw the shelf,
	// drafted a full skill for work a bundled one already covered — the
	// duplicate would not even have applied (skill.Apply refuses "create" on
	// an existing name), but the user was still asked to read and judge it.
	Extends string `json:"extends"`
	Title   string `json:"title"`  // short human summary for the proposal card
	Body    string `json:"body"`   // SKILL.md markdown or declarative fact
	Reason  string `json:"reason"` // evidence citing observed sessions
}

const habitSynthesisInstructions = `You are Aetox's Habit & Workflow Synthesizer.
Your role is to analyze user requests and the actions/tools executed across multiple sessions (or a single repeated session).
Your goal is to decide whether this repeated pattern should become an automated Skill or a durable Memory, and propose it cleanly.

Rules:
1. Language: Must strictly match the language of the conversation/user (e.g. Thai if user speaks Thai).
2. Decision Rules:
   - "skill": If this pattern represents a procedure, script, multi-step workflow, or recurring tool task (e.g. GPU check, git pull-and-test, data conversions).
     First read "Skills already installed" in the user message. If one of them already covers this work, set "extends" to its exact name and write in "body" only what that skill is missing — the steps, commands or pitfalls these sessions showed — as a section to append to it. Never draft a second skill for work an installed one covers.
     Otherwise draft a new SKILL.md with YAML frontmatter:
     ---
     name: <slug>
     description: <one line, under 80 characters, in the user's language, naming the moment this applies (ตอน... / when...), not a summary of the steps — the only line the agent sees before opening the skill, and a description that summarises the work is followed instead of the skill>
     ---
     then the body. As short as the procedure allows, shaped to the work — headings only where the work has parts, not a fixed outline. The commands and checks the sessions actually used. Rules, not a story about the sessions: no dates, no quoted chat, no narration of what happened.
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
				"extends": {
					"type": "string",
					"description": "Exact name of an installed skill (from 'Skills already installed') that this work belongs in; body is then the section to append to it. Empty for a new skill."
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
	app *Engine
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
	// The same shelf the main prompt lists (bootstrap.skillReads): a drafter
	// that cannot see what is installed can only ever propose a new skill.
	if index := installedSkillIndex(); index != "" {
		userPrompt += "\n\n=== Skills already installed ===\n" + index
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

// lintHabitDraft reads a proposal the way the shelf is read: a new skill as
// a whole SKILL.md under its slug, a section for an installed skill as a
// fragment (no frontmatter of its own to hold to).
func lintHabitDraft(prop *HabitProposal) []skilllint.Finding {
	if strings.TrimSpace(prop.Extends) != "" {
		return skilllint.LintFragment(prop.Body)
	}
	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(prop.SkillName), " ", "-"))
	return skilllint.Lint(skilllint.Skill{Name: slug, Files: map[string]string{"SKILL.md": prop.Body}})
}

// installedSkillIndex is the shelf as one line per skill, name and description,
// for the drafter's prompt. The same scan skills_list and the main prompt run,
// so the three cannot disagree about what is installed.
func installedSkillIndex() string {
	var b strings.Builder
	for _, d := range installedSkills() {
		b.WriteString("- " + d.Name)
		if desc := strings.Join(strings.Fields(d.Description), " "); desc != "" {
			b.WriteString(": " + oneLine(desc, 160))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// installedSkillNames maps a case-folded skill name to the name as installed,
// which is how skill_view resolves one (bundled_skills.go).
func installedSkillNames() map[string]string {
	out := map[string]string{}
	for _, d := range installedSkills() {
		out[strings.ToLower(d.Name)] = d.Name
	}
	return out
}

func installedSkills() []skill.DiscoveredSkill {
	found := skill.ListDiscovered(skill.DefaultDiscoveryPaths())
	sort.Slice(found, func(i, j int) bool { return found[i].Name < found[j].Name })
	return found
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

// synthesizeHabitForSessions runs synthesis over a set of sessions and queues the proposal.
func (a *Engine) synthesizeHabitForSessions(ctx context.Context, synthesizer habitSynthesizer, sessionIDs []string, normalizedKey, hint string) (*PendingChange, error) {
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
	// A drafted skill is read by the same linter the bundled shelf passes
	// (internal/skilllint) before anyone is asked to judge it. An error is a
	// broken door, so the drafter gets one more go with the findings in its
	// hint; what is still wrong after that travels on the card, in the
	// reason, where the user decides — a proposal is never dropped for it.
	if prop.Destination == "skill" {
		findings := lintHabitDraft(prop)
		if skilllint.HasErrors(findings) {
			retry := hint + "\n\n[ตัวตรวจสกิลพบข้อผิดพลาดในร่างก่อนหน้า แก้แล้วเสนอใหม่:\n" + skilllint.Render(skilllint.AtLeast(findings, skilllint.Error)) + "]"
			if again, err := synthesizer.Synthesize(ctx, digest, retry); err == nil && again != nil && strings.TrimSpace(again.Body) != "" && again.Destination == "skill" {
				prop = again
				findings = lintHabitDraft(prop)
			}
		}
		if worth := skilllint.AtLeast(findings, skilllint.Warn); len(worth) > 0 {
			prop.Reason = strings.TrimSpace(prop.Reason) + "\n\nตัวตรวจสกิล (aetox skill lint):\n" + strings.TrimSpace(skilllint.Render(worth))
		}
	}

	var kind string
	var scope string
	var op string
	target := ""

	switch prop.Destination {
	case "skill":
		kind = kindSkill
		installed := installedSkillNames()
		if extends := strings.ToLower(strings.TrimSpace(prop.Extends)); extends != "" {
			// The model named a skill to extend. Only a name that is really on
			// the shelf is honoured — a made-up one would queue an "add" that
			// approval could not apply.
			if name, ok := installed[extends]; ok {
				scope, target, op = name, name, learned.OpAdd
				break
			}
			debuglog.Msg("habit_synthesis: extends %q names no installed skill, drafting anew", prop.Extends)
		}
		skillSlug := strings.TrimSpace(prop.SkillName)
		if skillSlug == "" {
			skillSlug = "custom-workflow"
		}
		// Clean slug
		skillSlug = strings.ToLower(strings.ReplaceAll(skillSlug, " ", "-"))
		if name, ok := installed[skillSlug]; ok {
			// A new skill under a name that is taken. skill.Apply would refuse
			// it at approval, so the user would be asked to judge a proposal
			// that cannot land; not queued, and the log says which one.
			debuglog.Msg("habit_synthesis: %q is already installed, new skill not queued", name)
			return nil, nil
		}
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

	var id int64
	if kind == kindMemory {
		// A memory line goes through the one door every memory proposer uses,
		// which is what knows whether the user already answered it. This
		// writer used to insert directly and check nothing — three of its
		// lines were refused before the door could tell it so.
		res, err := a.queueMemoryProposal(learned.Proposal{
			Kind: kind, Scope: scope, Op: op, Body: change.Body, Reason: change.Reason,
		}, change.Source, "")
		if err != nil {
			return nil, fmt.Errorf("failed to record habit proposal: %w", err)
		}
		if res.Prior != nil {
			debuglog.Msg("habit_synthesis: %q restates #%d (%s), not queued", change.Body, res.Prior.ID, res.Prior.State)
			return nil, nil
		}
		id = res.ID
	} else {
		// A skill proposal is bound by the same rule at its own door: the
		// user's earlier answer about this skill stands, in any wording.
		if prior, found, err := a.priorSkillDecision(scope, op, "", change.Body); err != nil {
			return nil, fmt.Errorf("failed to record habit proposal: %w", err)
		} else if found {
			debuglog.Msg("habit_synthesis: skill %q restates #%d (%s), not queued", scope, prior.ID, prior.State)
			return nil, nil
		}
		res, err := db.Exec(`
			INSERT INTO pending_changes (kind, scope, target, op, body, reason, source, state, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			change.Kind, change.Scope, change.Target, change.Op, change.Body, change.Reason, change.Source, change.State, change.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to record habit proposal: %w", err)
		}
		id, _ = res.LastInsertId()
	}
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
func (a *Engine) maybeSynthesizeHabits() {
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

		_, err := a.synthesizeHabitForSessions(ctx, appHabitSynthesizer{a}, req.SessionIDs, req.Normalized, "")
		if err != nil {
			debuglog.Msg("habit_synthesis: failed for %q: %v", req.Normalized, err)
		}
	}(*targetReq)
}

// SynthesizeHabit is the manual/on-demand trigger exposed to Wails.
// Even if count < 3 or user explicitly requests it, this performs synthesis immediately.
func (a *Engine) SynthesizeHabit(sessionID string, hint string) (int64, error) {
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

	change, err := a.synthesizeHabitForSessions(ctx, appHabitSynthesizer{a}, sessionIDs, normKey, hint)
	if err != nil {
		return 0, err
	}
	if change == nil {
		return 0, nil
	}
	return change.ID, nil
}
