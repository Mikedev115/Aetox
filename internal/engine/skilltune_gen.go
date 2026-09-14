package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/debuglog"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
	"github.com/Mikedev115/Aetox/internal/skilllint"
)

// Stage three of the self-optimize loop: turning a flagged misfire into a
// proposed skill edit (docs/architecture/self-optimize-loop-2026-08-26.md).
//
// The detector (skilltune.go) is deterministic; this is not — drafting an edit
// to prose needs a model to read the skill and the misfires and say what to
// change. That one non-deterministic step sits behind an interface so the
// pipeline around it is testable without an LLM, and behind human approval so a
// draft is only ever a proposal: nothing it produces touches a skill until a
// person clicks yes.

// skillEditDrafter reads a flagged skill and the results that got bad ratings,
// and proposes one edit to its SKILL.md. The Engine's real implementation calls a
// model; a test supplies a fake.
type skillEditDrafter interface {
	// refused is what the user already turned down for this skill, one edit
	// per line, so the drafter is not asked to rediscover them.
	Draft(ctx context.Context, skillName, skillText, evidence, refused string) (op, before, body, reason string, err error)
}

// generateSkillRefinements drafts an edit for each skill the detector flags and
// queues it for approval — skipping any skill that already has a proposal
// waiting, so a skill is not re-drafted and the model not re-called every pass.
func (a *Engine) generateSkillRefinements(ctx context.Context, drafter skillEditDrafter) {
	if !learningEnabled() || drafter == nil {
		return
	}
	// Through skillMisfires, not detectSkillMisfires: the Engine reader already
	// opens the database and supplies the live skill set, and this function
	// having its own copy of both meant the two could drift apart on which
	// names count as skills.
	for _, m := range a.skillMisfires() {
		if a.hasPendingSkillProposal(m.skill) || a.refusalStillStands(m) {
			continue
		}
		text, err := skill.Body(m.skill)
		if err != nil {
			continue
		}
		op, before, body, reason, err := drafter.Draft(ctx, m.skill, text, a.skillEvidence(m), a.refusedSkillEdits(m.skill))
		if err != nil {
			debuglog.Msg("skilltune: drafting %s failed: %v", m.skill, err)
			continue
		}
		if strings.TrimSpace(body) == "" {
			continue
		}
		a.proposeSkillEdit(m, op, before, body, reason)
	}
}

// autoTuneSkills lets tests turn the background drafter off: a unit test that
// rates a skill bad three times would otherwise trip maybeTuneSkills into a real
// model call. Set false in the test harness (newJobApp); true in the app.
var autoTuneSkills = true

// maybeTuneSkills is the trigger, called after every turn from recordJobs. It
// does the cheap deterministic detect on this goroutine and, only if a skill is
// actually flagged, spends a model call to draft — in the background, one at a
// time, so a chat turn is never slowed and two turns never draft at once. The
// proposal it produces waits for human approval like every other; nothing here
// changes a skill on its own.
func (a *Engine) maybeTuneSkills() {
	if !autoTuneSkills || !learningEnabled() || !skillTuneAuto() {
		return
	}
	db, err := a.database()
	if err != nil {
		return
	}
	names := map[string]bool{}
	for _, d := range skill.ListDiscovered(skill.DefaultDiscoveryPaths()) {
		if d.Name != "" {
			names[d.Name] = true
		}
	}
	if len(detectSkillMisfires(db, names, skillMisfireMinBad)) == 0 {
		return
	}
	if !a.skillTuneRunning.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer a.skillTuneRunning.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		a.generateSkillRefinements(ctx, appDrafter{a})
	}()
}

// RunSkillTuneup is the manual trigger for the settings page: it drafts now,
// whatever the auto switch says (the user asked for it), and reports how many
// new proposals it queued so the page can say something happened. Synchronous —
// the caller awaits one model call — and still only ever queues proposals.
func (a *Engine) RunSkillTuneup() (int, error) {
	if !learningEnabled() {
		return 0, fmt.Errorf("learning is switched off in settings")
	}
	before := a.countPending(kindSkill)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	a.generateSkillRefinements(ctx, appDrafter{a})
	return a.countPending(kindSkill) - before, nil
}

// hasPendingSkillProposal is the gate that keeps the model from being called for
// a skill whose edit is already waiting on the human.
func (a *Engine) hasPendingSkillProposal(skillName string) bool {
	db, err := a.database()
	if err != nil {
		return false
	}
	var id int64
	return db.QueryRow(
		`SELECT id FROM pending_changes WHERE kind = ? AND scope = ? AND state = ? LIMIT 1`,
		kindSkill, skillName, statePending).Scan(&id) == nil
}

// refusalStillStands is the other gate, and the one the first pass lacked
// (found 11 ก.ย. with the memory queue's twin, DECISIONS §246): a skill whose
// last edit the user turned down was re-drafted on the very next pass, from
// the same misfires, and a model asked the same question twice answers in
// different words — so the refusal bought one turn of quiet and a new card.
// A no stands until something new happens: the skill is left alone until a
// bad rating arrives that is newer than the refusal.
func (a *Engine) refusalStillStands(m skillMisfire) bool {
	db, err := a.database()
	if err != nil || len(m.jobIDs) == 0 {
		return false
	}
	var decidedAt string
	if db.QueryRow(
		`SELECT decided_at FROM pending_changes WHERE kind = ? AND scope = ? AND state <> ?
		  ORDER BY id DESC LIMIT 1`,
		kindSkill, m.skill, statePending).Scan(&decidedAt) != nil {
		return false
	}
	var latest string
	// A row per id rather than IN (?), which database/sql cannot expand; the
	// list is capped at a handful by the detector.
	for _, id := range m.jobIDs {
		var at string
		if db.QueryRow(`SELECT time FROM jobs WHERE id = ?`, id).Scan(&at) == nil && at > latest {
			latest = at
		}
	}
	// Only a refusal holds the line: an approved edit changed the file, and
	// misfires after it are a new question about a new skill.
	var state string
	_ = db.QueryRow(
		`SELECT state FROM pending_changes WHERE kind = ? AND scope = ? AND state <> ?
		  ORDER BY id DESC LIMIT 1`, kindSkill, m.skill, statePending).Scan(&state)
	return state == stateRejected && latest != "" && latest <= decidedAt
}

// refusedSkillEdits is what the user turned down for one skill, newest first,
// for the drafter to read before it drafts — the same list the memory reviewer
// reads for the same reason.
func (a *Engine) refusedSkillEdits(skillName string) string {
	db, err := a.database()
	if err != nil {
		return ""
	}
	var b strings.Builder
	_ = eachRow(db, "skilltune: reading refused edits", `
		SELECT op, body FROM pending_changes WHERE kind = ? AND scope = ? AND state = ?
		  ORDER BY id DESC LIMIT 10`, []any{kindSkill, skillName, stateRejected},
		func(rows *sql.Rows) error {
			var op, body string
			if err := rows.Scan(&op, &body); err != nil {
				return err
			}
			fmt.Fprintf(&b, "- [%s] %s\n", op, oneLine(body, 300))
			return nil
		})
	return strings.TrimRight(b.String(), "\n")
}

// skillEvidence is the misfires a proposal is grounded in — the request that got
// a bad answer, the answer, and how it was scored (👎 or a redo). A person reads
// it to judge whether the edit is warranted; the drafter reads it to know what
// went wrong. Capped: a handful of examples is enough to draft from and to read.
func (a *Engine) skillEvidence(m skillMisfire) string {
	db, err := a.database()
	if err != nil || len(m.jobIDs) == 0 {
		return ""
	}
	ids := m.jobIDs
	const maxExamples = 5
	if len(ids) > maxExamples {
		ids = ids[:maxExamples]
	}
	var b strings.Builder
	for _, id := range ids {
		var req, ans, src string
		if db.QueryRow(`SELECT request, answer, outcome_source FROM jobs WHERE id = ?`, id).
			Scan(&req, &ans, &src) != nil {
			continue
		}
		if src == "" {
			src = "👎"
		}
		fmt.Fprintf(&b, "- [%s] ถาม: %s\n  ตอบ: %s\n", src, oneLine(req, 220), oneLine(ans, 220))
	}
	return strings.TrimRight(b.String(), "\n")
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		s = string(r[:max]) + "…"
	}
	return s
}

// proposeSkillEdit queues one drafted edit as a skill-kind proposal. Mirrors the
// summarizer's proposeSystemIssue: its own INSERT with source "optimizer" — the
// system's self-optimize, kept distinct from "the model decided this mid-task"
// (source "agent") — deduped so the same edit is not queued twice.
func (a *Engine) proposeSkillEdit(m skillMisfire, op, before, body, reason string) bool {
	db, err := a.database()
	if err != nil {
		return false
	}
	// The identical edit in any state, as before — and, since §246, the same
	// edit in other words against what the user already answered.
	if prior, found, err := a.priorSkillDecision(m.skill, op, before, body); err != nil || found {
		if found {
			debuglog.Msg("skilltune: %s edit restates #%d (%s), not queued", m.skill, prior.ID, prior.State)
		}
		return false
	}
	if strings.TrimSpace(reason) == "" {
		reason = fmt.Sprintf("โดน 👎/ตอบใหม่ %d ครั้ง จาก %d ที่ให้คะแนน", m.bad, m.bad+m.good)
	}
	// The replacement text is read by the shelf's linter (internal/skilllint)
	// and what it finds rides on the card: the user judging the edit sees a
	// hedge or a dead door named, instead of finding it after approving.
	if worth := skilllint.AtLeast(skilllint.LintFragment(body), skilllint.Warn); len(worth) > 0 {
		reason = strings.TrimSpace(reason) + "\n\nตัวตรวจสกิล (aetox skill lint):\n" + strings.TrimSpace(skilllint.Render(worth))
	}
	if _, err := db.Exec(
		`INSERT INTO pending_changes(kind, scope, target, op, before, body, reason, evidence, source, state, created_at)
		 VALUES(?, ?, '', ?, ?, ?, ?, ?, 'optimizer', ?, ?)`,
		kindSkill, m.skill, op, before, body, reason, jobsEvidence(m.jobIDs), statePending,
		time.Now().Format(time.RFC3339)); err != nil {
		debuglog.Msg("skilltune: proposing edit failed: %v", err)
		return false
	}
	a.emitLearningChanged()
	return true
}

// jobsEvidence names the job rows a proposal was drawn from, capped — the same
// shape as summarize.go's evidenceFor, but labelled jobs, not tool_runs, because
// a skill's misfires are scored on the job, not on any one call.
func jobsEvidence(ids []int64) string {
	const maxListed = 20
	listed := ids
	if len(listed) > maxListed {
		listed = listed[:maxListed]
	}
	parts := make([]string, len(listed))
	for i, id := range listed {
		parts[i] = fmt.Sprint(id)
	}
	out := "jobs:" + strings.Join(parts, ",")
	if extra := len(ids) - maxListed; extra > 0 {
		out += fmt.Sprintf(" +%d", extra)
	}
	return out
}

// appDrafter is the real drafter: it asks the user's own current model to
// propose the edit, through the same one-shot Complete path the connection test
// uses. It is the only part of the loop that spends a model call, which is why
// the caller gates it on a skill not already having a proposal waiting.
type appDrafter struct{ app *Engine }

const skillDraftInstructions = `คุณคือผู้ดูแลสกิลของ Aetox. สกิลด้านล่างถูกผู้ใช้ให้คะแนนแย่ (👎 หรือกด "ตอบใหม่") ซ้ำหลายครั้ง.
อ่านหลักฐานแล้วเรียกเครื่องมือ skill_edit เพื่อเสนอการแก้ SKILL.md เพียง "จุดเดียว" ที่น่าจะลดความพลาด โดยยึดจากหลักฐาน ไม่ใช่การเดา.
แก้ให้น้อยที่สุดที่ได้ผล อย่ารื้อทั้งไฟล์. ถ้าปัญหาไม่ได้อยู่ที่สกิล ให้ body ว่างไว้ (ระบบจะไม่เสนออะไร).`

// skillEditTool is the structured-output channel, learned from how Hermes (and
// Claude Code's synthetic StructuredOutput tool) get a parseable result across
// providers: do not ask the model to "return JSON" — a model that wants to say
// "Looking at the pattern:…" as well will put both in one string. Force a single
// tool call instead, so the prose lands in the text channel and the tool's
// arguments stay pure JSON. One tool + ToolChoice "required" forces it on both
// wire formats (Anthropic maps "required"→any, OpenAI passes it through), so no
// per-provider branch is needed. Application stays deterministic (skill.Apply)
// and gated on human approval — the same split Hermes uses.
var skillEditTool = model.ToolDefinition{
	Type: "function",
	Function: model.ToolFunction{
		Name:        "skill_edit",
		Description: "เสนอการแก้ SKILL.md หนึ่งจุดที่จะลดความพลาดที่โดนให้คะแนนแย่",
		Parameters: json.RawMessage(`{
			"type":"object",
			"properties":{
				"op":{"type":"string","enum":["replace","add"],"description":"replace = แทนข้อความเดิมทั้งท่อน, add = เติมย่อหน้าใหม่ท้ายไฟล์"},
				"before":{"type":"string","description":"ข้อความเดิมทั้งท่อนที่จะแทน ต้องตรงเป๊ะกับใน SKILL.md (เว้นว่างถ้า op=add)"},
				"body":{"type":"string","description":"ข้อความใหม่ (เว้นว่างถ้าปัญหาไม่ได้อยู่ที่สกิล)"},
				"reason":{"type":"string","description":"เหตุผลสั้น ๆ ภาษาไทย"}
			},
			"required":["op","body","reason"]
		}`),
	},
}

func (d appDrafter) Draft(ctx context.Context, skillName, skillText, evidence, refused string) (op, before, body, reason string, err error) {
	p, modelName, err := d.app.oneShotProvider()
	if err != nil {
		return "", "", "", "", err
	}
	content := fmt.Sprintf(
		"%s\n\n=== สกิล: %s ===\n%s\n\n=== ผลลัพธ์ที่โดนให้คะแนนแย่ ===\n%s",
		skillDraftInstructions, skillName, skillText, evidence)
	// Only when there is something to say, so a skill with no history reads
	// the prompt it always read.
	if refused != "" {
		content += "\n\n=== การแก้ที่ผู้ใช้ไม่เอาแล้ว (อย่าเสนอสิ่งเดียวกันในสำนวนอื่น ถ้าไม่มีทางอื่นที่จะช่วย ให้ body ว่าง) ===\n" + refused
	}
	req := model.Request{
		Model:      modelName,
		Messages:   []model.Message{{Role: model.RoleUser, Content: content}},
		Tools:      []model.ToolDefinition{skillEditTool},
		ToolChoice: "required",
		MaxTokens:  1800,
	}
	// One retry: a model that preambled instead of calling the tool the first
	// time almost always calls it cleanly on a second ask (Hermes' parse-then-
	// retry-once recovers ~95% of these).
	op, before, body, reason, err = draftSkillEditOnce(ctx, p, req)
	if err == nil {
		return op, before, body, reason, nil
	}
	return draftSkillEditOnce(ctx, p, req)
}

// draftSkillEditOnce runs one forced-tool call and reads the edit off the tool
// arguments, falling back to a JSON object in the text for a provider that
// answered in prose despite the forcing.
func draftSkillEditOnce(ctx context.Context, p model.Provider, req model.Request) (op, before, body, reason string, err error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return "", "", "", "", err
	}
	var raw string
	if len(resp.ToolCalls) > 0 {
		raw = resp.ToolCalls[0].Function.Arguments
	} else {
		raw = extractJSONObject(resp.Text)
	}
	if raw == "" {
		return "", "", "", "", fmt.Errorf("the drafter returned neither a tool call nor JSON")
	}
	var out struct {
		Op     string `json:"op"`
		Before string `json:"before"`
		Body   string `json:"body"`
		Reason string `json:"reason"`
	}
	if json.Unmarshal([]byte(raw), &out) != nil {
		return "", "", "", "", fmt.Errorf("the drafter's arguments did not parse")
	}
	if out.Op == "" {
		out.Op = "add"
	}
	return out.Op, out.Before, out.Body, out.Reason, nil
}

// oneShotProvider builds a client on the user's current model, the same way
// TestProviderConnection does — endpoint, key and wire format from the live
// config, so the loop drafts with the model the user is actually paying for.
func (a *Engine) oneShotProvider() (model.Provider, string, error) {
	return a.oneShotProviderFor(a.cur())
}

// oneShotProviderFor is the same thing for a NAMED conversation, which is what
// any caller reached from a finished turn must use. `a.cur()` is the chat on
// screen — a cursor this codebase stopped trusting at §150 — and a turn that
// ended while the user was reading another conversation would otherwise be
// followed up on whatever model THAT chat happens to run. Same rule the turn
// path already follows: the conversation is handed over and held, never read
// back later.
func (a *Engine) oneShotProviderFor(conv *conversation) (model.Provider, string, error) {
	if conv == nil {
		conv = a.cur()
	}
	cfg := conv.cfg
	canonical := model.NormalizeProvider(cfg.ModelProvider)
	baseURL := resolveBaseURLForProvider(canonical)
	modelName := strings.TrimSpace(cfg.ModelName)
	if modelName == "" {
		modelName = a.defaultModel(canonical, baseURL)
	}
	if modelName == "" {
		return nil, "", fmt.Errorf("no model is configured to draft with")
	}
	// Signed by the screen's transport, like the chat's own provider: this
	// drafter runs on the engine side and holds no key (§248 A3).
	screen := a.screenOf()
	p, err := model.NewProvider(model.ProviderOptions{
		Provider:         canonical,
		Model:            modelName,
		BaseURL:          baseURL,
		Timeout:          60 * time.Second,
		WireFormat:       cfg.ModelWireFormat,
		Transport:        screen.ProviderTransport(canonical, cfg.ModelWireFormat),
		SignedInEndpoint: screen.ProviderEndpoint(canonical),
	})
	return p, modelName, err
}

// extractJSONObject pulls the first {...} block out of a model reply, so a model
// that wrapped its JSON in prose or a ```json fence is still read.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}
