package main

// The plan as a thing, rather than as a message that happened to contain one.
//
// ## What was wrong
//
// วางแผน has produced a plan since §106.9 and drawn it as a card since §106.12,
// and in between those two the plan itself was never anywhere. It was prose in
// one assistant message, wrapped in a fence so the chat could put a border round
// it. Nothing could point at it, so nothing could change it — and "change step
// 3" with no step 3 to change means the only available revision is the whole
// document, written again, off readings taken again.
//
// The owner put it plainly on 2026-09-08: the assistant asks nothing and writes
// the plan from scratch every time. The audit agreed with him — 70% of every
// byte of plan ever written on his machine was written over a plan that already
// existed (TOKEN-AUDIT.md, PLAN REWRITES).
//
// ## The shape
//
// One plan per conversation, in a row of its own (db.go, migration v20). Three
// actions on one packed tool:
//
//   - `write`  — the plan, first time. Replaces whatever was there.
//   - `amend`  — named sections only. The rest stands, and is not re-sent.
//   - `read`   — the plan as it stands now.
//
// `amend` is the entire point and the only one of the three that saves
// anything. A revision costs the section that changed instead of the document
// that did not.
//
// ## Why `read` is not withheld anywhere
//
// The other two belong to วางแผน. `read` belongs to every stance, and that is
// the second thing this file buys: switching to ลงมือ used to leave the plan
// behind as transcript text, so the acting session read it back out of the
// conversation and interpreted it again. Now it asks. มุ่งเป้า, when it is
// built, aims at the same row.
//
// ## The headings are not this file's to invent
//
// They come from mode.PlanHeadings(), which is where the shape has lived since
// §106.11 and which had lost its last caller when the `plan` sub-agent profile
// was deleted. Reading them from there rather than restating them is what makes
// the tool, the prompt and the card agree by construction — and it puts the
// §106.11 seam back on its feet with a real caller behind it.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// PlanSection is one heading and what is under it, as stored and as drawn.
//
// Heading is matched case-insensitively on the way in and stored in the
// canonical spelling mode.PlanHeadings() gives, so a model that writes "what to
// change" amends the section it meant rather than adding a second one beside it.
type PlanSection struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

// The states a step can be in. `failed` settles a step as much as `done` does:
// a step the model looked at and reported impossible is a finding the user
// needs, and a run that sent the turn back over it forever would be refusing to
// accept an answer.
const (
	planStepTodo   = ""
	planStepDoing  = "doing"
	planStepDone   = "done"
	planStepFailed = "failed"
)

// finishHeading is the section มุ่งเป้า checks against. Named here rather than
// spelled at the two call sites, and taken from the shape rather than invented:
// mode.PlanHeadings() is the one place the wording lives (§106.11).
var finishHeading = planFinishHeading()

func planFinishHeading() string {
	for _, h := range mode.PlanHeadings() {
		if strings.Contains(strings.ToLower(h), "know it worked") {
			return h
		}
	}
	return ""
}

// PlanStep is one numbered step of the plan, with somewhere to record that it
// has been carried out.
//
// **Steps are structured rather than parsed out of the "What to change" body,
// and that is what makes มุ่งเป้า's first gate mechanical.** A markdown list is
// prose: to know whether step 3 is done you would have to read it, which is a
// judgement, which is the thing §106.10 said the checker must not be. A row with
// a state on it is a fact.
type PlanStep struct {
	N     int    `json:"n"`
	Text  string `json:"text"`
	State string `json:"state,omitempty"`
	// Note is why a step failed, or anything worth carrying beside a done one.
	// The card draws it under the step; nothing else reads it.
	Note string `json:"note,omitempty"`
	// Stop is a breakpoint the USER set: carry the plan out up to here and then
	// wait for me.
	//
	// It exists because the pause button only works for somebody sitting in
	// front of the screen, and this mode's whole purpose is being able to walk
	// away. A breakpoint is a pause you can set in advance, on the step you
	// already know you want to see before it happens.
	//
	// Stored with the plan rather than with the run, and that is deliberate: it
	// is a fact about this step of this plan — "do not do this one without me" —
	// and it should still be there tomorrow, and for the next run.
	Stop bool `json:"stop,omitempty"`
}

// Plan is a conversation's plan as the frontend draws it.
type Plan struct {
	Title    string        `json:"title"`
	Sections []PlanSection `json:"sections"`
	// Steps is the plan's own checklist, and it is what a run is measured
	// against. Empty for a plan written without one, which is a plan มุ่งเป้า
	// refuses to carry out — see StartPlanRun for why that refusal is the
	// honest answer rather than an inconvenience.
	Steps []PlanStep `json:"steps,omitempty"`
	// Version counts the writes, starting at 1. It is what lets the card say
	// which revision this is rather than redrawing silently — a plan that
	// changes with no mark on it reads as a plan the user misremembered.
	Version int `json:"version"`
	// Changed names the sections this call touched, empty on a first write. The
	// card marks them, which is the visible half of amending: a revision the
	// user cannot see the shape of is a revision they have to re-read in full,
	// which is the cost this whole change is about.
	Changed []string `json:"changed,omitempty"`
	Updated string   `json:"updated"`
	// SentBack counts the times the engine refused to let the turn end because
	// the plan was not finished (goal_run.go).
	//
	// On screen because it is the ONE part of มุ่งเป้า the user cannot otherwise
	// see: the gates fire between the model speaking and the turn ending, so a
	// run that was pushed back four times looks exactly like one that sailed
	// through. A number in the run bar is the cheapest honest way to say so —
	// cheaper than the rail that was the other candidate, and it costs nothing
	// per round because it is state rather than a sentence.
	SentBack int `json:"sentBack,omitempty"`
	// StartedAt is when the run began, RFC3339, empty when none is running. The
	// bar counts up from it in the window rather than the Go side pushing a
	// clock: a timer that ticks over an event stream is a stream of events that
	// say nothing.
	StartedAt string `json:"startedAt,omitempty"`
	// Paused says the run is holding, having finished what it was doing. Not a
	// stop: nothing is lost, the turn simply ended where it stood, and ไปต่อ
	// picks it up. Empty string when it is not paused; otherwise why.
	Paused string `json:"paused,omitempty"`
	// Running says this conversation is carrying the plan out right now
	// (goal_run.go). Not stored: a run is a turn in progress, and one that
	// survived a restart would be a window offering to stop something that is
	// not happening. Filled in on the way to the screen.
	Running bool `json:"running"`
}

// planSkill is the pack. Actions are `plan_write`, `plan_amend` and `plan_read`
// once unpacked — the names every gate and every stance judges by (packed.go).
type planSkill struct {
	app  *App
	conv *conversation
}

func (*planSkill) Name() string { return "plan" }

func (*planSkill) Description() string {
	return "เขียน แก้ และอ่านแผนของบทสนทนานี้"
}

func (s *planSkill) ToolDefinition() model.ToolDefinition {
	headings := mode.PlanHeadings()

	// SIGNATURES ONLY, the standard this repo holds every tool block to: what a
	// plan IS belongs to the stance's own direction, which the model already has
	// whenever these tools are on the desk, and repeating it here would be the
	// shape written down in a second place — the debt §106.11 exists to avoid.
	var b strings.Builder
	b.WriteString("This conversation's plan, kept as one document rather than retyped. Actions:\n")
	b.WriteString("`write` (title, sections, steps) — the plan, first time. Replaces any plan already here.\n")
	b.WriteString("`amend` (sections, note?) — change only the sections you name. Everything else stands, so do not re-send it.\n")
	b.WriteString("`read` — the plan as it stands now.\n")
	b.WriteString("`step` (n, state: doing|done|failed, note?) — mark one step as you carry it out.\n")
	b.WriteString("Headings, in this order: " + strings.Join(headings, " / ") + "\n")

	section := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"heading": map[string]any{"type": "string", "enum": headings},
			"body":    map[string]any{"type": "string", "description": "Markdown. Steps as a numbered list, so a later amend can name one."},
		},
		"required":             []string{"heading", "body"},
		"additionalProperties": false,
	}
	return toolDef("plan", b.String(), map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"write", "amend", "read", "step"}},
			"steps": map[string]any{
				"type":        "array",
				"description": "write only: the plan's checklist, in order. One thing to carry out per entry.",
				"items":       map[string]any{"type": "string"},
			},
			"n":        map[string]any{"type": "integer", "description": "step only: which step, as numbered by read."},
			"state":    map[string]any{"type": "string", "enum": []string{"doing", "done", "failed"}},
			"title":    map[string]any{"type": "string", "description": "The job in one line."},
			"sections": map[string]any{"type": "array", "items": section},
			"note":     map[string]any{"type": "string", "description": "amend only: one line saying what moved and why."},
		},
		"required":             []string{"action"},
		"additionalProperties": false,
	})
}

// planFail is the one way this tool reports a refusal.
//
// Every error return used to hand back a bare `skill.Output{Name: "plan"}`
// beside the error, which is a receipt with nothing written on it: the UI draws
// a failed tool row from Stderr and Content, so the user saw a plan call fail
// and was told nothing about why. Caught by tool_coverage_test.go, whose report
// of the failure was itself blank — the test could not say what went wrong
// because the tool did not say.
func planFail(err error) (skill.Output, error) {
	return skill.Output{
		Name: "plan", Command: "plan", Success: false,
		Content: err.Error(), Stderr: err.Error(), RawOutput: err.Error(),
	}, err
}

// Guidance is what somebody needs to know once about working a plan — the
// judgment layer, delivered with the first result of each action and never
// again (internal/skill/guidance.go).
//
// **This is where the instructions for carrying out a plan live, and the reason
// is arithmetic.** The first version of the run button sent them as a user turn:
// "work one step at a time, mark each with `plan step`, do not narrate". A user
// turn is re-sent with every later round of the same turn, so a paragraph of
// instruction was being paid for on every round — of the one mode built around
// the fact that exactly this cost is quadratic. Here it is sent once.
//
// Per action, as the interface intends: a session that only ever reads a plan
// has no use for what `step` is for.
func (s *planSkill) Guidance(args map[string]any) string {
	switch strings.ToLower(strings.TrimSpace(str(args["action"]))) {
	case "write":
		return "The checklist goes in `steps`, as its own list — never as a numbered list inside a " +
			"section. Steps are what the user presses a button to have carried out and what gets ticked " +
			"off one at a time, so a plan that writes them as prose comes back as a document nobody can " +
			"run. Under the section about the work, say what the work IS; put the things to do in `steps`.\n" +
			"One step is one thing somebody can finish and say so. \"Make it better\" is not a step; " +
			"\"raise waitMax to 600\" is."
	case "amend":
		return "Send only the sections that changed. Everything you do not name stands exactly as it was, " +
			"which is the whole reason this takes sections rather than a document — re-sending a section " +
			"nobody asked about costs what writing it cost the first time.\n" +
			"An amend that names no `steps` keeps the ones there are, with their marks. Send `steps` only " +
			"to replace the checklist outright, which throws away the record of what has been done."
	case "step":
		return "Mark each step the moment it is finished, not in a batch at the end: the user is watching " +
			"the checklist, and a run that marks nothing looks like a run that is doing nothing.\n" +
			"`failed` is a real answer and settles the step as `done` does — a step you looked at and " +
			"found impossible is a finding the user needs, with the reason in `note`. Leaving it open " +
			"instead means the work is never finished.\n" +
			"While a plan is being carried out you do not need to narrate progress in words. The " +
			"checklist says it, and a turn that both marks and describes has written it twice."
	case "read":
		return "The marks come back with it, so this is how a later turn — or a later stance — finds out " +
			"where the work got to rather than starting from what it remembers."
	}
	return ""
}

func (s *planSkill) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	return s.run(args)
}

func (s *planSkill) Execute(_ context.Context, input skill.Input) (skill.Output, error) {
	return s.run(map[string]any(input))
}

func (s *planSkill) run(args map[string]any) (skill.Output, error) {
	start := time.Now()
	action := strings.ToLower(strings.TrimSpace(str(args["action"])))
	if action == "" {
		// The fallback a habit reaches for. `read` rather than `write`, because
		// guessing wrong here has to be the harmless direction: a `read` that
		// was meant as a `write` costs a round, and a `write` that was meant as
		// a `read` overwrites the plan with whatever happened to be in `args`.
		action = "read"
	}

	switch action {
	case "read":
		return s.read(start)
	case "write", "amend":
		return s.save(start, action, args)
	case "step":
		return s.step(start, args)
	}
	return planFail(fmt.Errorf("plan %s is not an action — use write, amend or read", action))
}

func (s *planSkill) sessionID() string {
	if s.conv == nil {
		return ""
	}
	return s.conv.id
}

func (s *planSkill) read(start time.Time) (skill.Output, error) {
	plan, err := s.app.loadPlan(s.sessionID())
	if err != nil {
		return planFail(err)
	}
	if plan == nil {
		// Not an error, and the wording matters: "no plan yet" and "the plan is
		// empty" send a model to different places, and only one of them is true
		// here. The same distinction browser `wait` draws between a page that
		// has not said it yet and a page that never will.
		out := "there is no plan in this conversation yet."
		return skill.Output{
			Name: "plan", Command: "plan read", Success: true,
			Content: out, RawOutput: out, DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}
	body := planMarkdown(*plan)
	return skill.Output{
		Name: "plan", Command: "plan read", Success: true,
		Content: body, RawOutput: body, DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (s *planSkill) save(start time.Time, action string, args map[string]any) (skill.Output, error) {
	id := s.sessionID()
	if id == "" {
		return planFail(errors.New("a plan belongs to a conversation, and this call has none"))
	}
	incoming, err := parsePlanSections(args["sections"])
	if err != nil {
		return planFail(err)
	}
	if len(incoming) == 0 {
		return planFail(fmt.Errorf("plan %s needs at least one section", action))
	}

	existing, err := s.app.loadPlan(id)
	if err != nil {
		return planFail(err)
	}

	plan := Plan{Title: strings.TrimSpace(str(args["title"])), Steps: parsePlanSteps(args["steps"])}
	// **A PLAN WITH NO STEPS HAS NO CHECKLIST AND NO BUTTON, and that is what
	// the owner saw on the first real plan of the day.** The model had written
	// its steps as a numbered list inside "What to change" — which is a perfectly
	// reasonable reading of a prompt that says "number the steps" — and passed no
	// `steps` at all. The card came back inert: no checklist, no way to run it,
	// nothing to say why.
	//
	// So they are recovered from the prose. The parser is the one hand editing
	// already uses (plan_edit.go), which is the whole reason this is safe rather
	// than clever: it takes a line only when it carries a MARK or a NUMBER,
	// which `TestProseDoesNotBecomeSteps` pins, so a paragraph does not become
	// nine steps.
	//
	// Only from "What to change", and only when the call named none. Reading
	// every section would turn a numbered list in "What could go wrong" into work
	// somebody has to carry out; overriding what the call said would be this
	// function deciding it knows better than its own arguments.
	if len(plan.Steps) == 0 {
		plan.Steps = stepsFromProse(incoming)
	}
	switch {
	case action == "write" || existing == nil:
		// `amend` with nothing to amend is treated as a write rather than
		// refused. It is the honest reading of the call — the model is handing
		// over sections for a plan that does not exist — and refusing it would
		// spend a round teaching a distinction the result does not depend on.
		plan.Sections = incoming
		plan.Version = 1
	default:
		plan.Sections = mergePlanSections(existing.Sections, incoming)
		plan.Version = existing.Version + 1
		if plan.Title == "" {
			plan.Title = existing.Title
		}
		// An amend that names no steps keeps the ones there are, WITH their
		// states. Re-sending the checklist to change one heading would throw
		// away every mark a run has made — the same mistake amend exists to
		// stop it making with the sections.
		if len(plan.Steps) == 0 {
			plan.Steps = existing.Steps
		}
		for _, sec := range incoming {
			plan.Changed = append(plan.Changed, sec.Heading)
		}
	}
	plan.Sections = orderPlanSections(plan.Sections)
	plan.Updated = time.Now().Format(time.RFC3339)

	if err := s.app.savePlan(id, plan); err != nil {
		return planFail(err)
	}
	// The card is drawn from this, the same way the checklist is drawn from
	// `todo:update` (ask_user.go): the tool holds the state, the window draws
	// it, and the model never spends output tokens on the picture. That is what
	// makes an amend cheap — before this, the only way a plan reached the screen
	// was the model typing the whole of it again.
	s.app.emitPlan(id, plan)

	receipt := planReceipt(action, plan, strings.TrimSpace(str(args["note"])))
	return skill.Output{
		Name: "plan", Command: "plan " + action, Success: true,
		Content: receipt, RawOutput: receipt, DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// step marks one item of the checklist as the work goes.
//
// It is the half of the plan that มุ่งเป้า reads (goal_run.go), and it is
// deliberately one step per call rather than a whole list: a model handed "send
// the entire state each time" sends the entire state, which is todo_write's
// bargain and the reason a checklist costs its own size on every update. A step
// is a fact that has just become true, and the call should be the size of that
// fact.
func (s *planSkill) step(start time.Time, args map[string]any) (skill.Output, error) {
	id := s.sessionID()
	plan, err := s.app.loadPlan(id)
	if err != nil {
		return planFail(err)
	}
	if plan == nil || len(plan.Steps) == 0 {
		return planFail(errors.New("there is no plan with steps in this conversation to mark"))
	}
	n := intArg(args["n"])
	state := strings.ToLower(strings.TrimSpace(str(args["state"])))
	switch state {
	case planStepDoing, planStepDone, planStepFailed:
	default:
		return planFail(fmt.Errorf("state %q is not doing, done or failed", state))
	}
	found := false
	for i := range plan.Steps {
		if plan.Steps[i].N == n {
			plan.Steps[i].State = state
			plan.Steps[i].Note = strings.TrimSpace(str(args["note"]))
			found = true
		}
	}
	if !found {
		return planFail(fmt.Errorf("this plan has no step %d — read it to see the numbering", n))
	}
	if err := s.app.savePlan(id, *plan); err != nil {
		return planFail(err)
	}
	s.app.emitPlan(id, *plan)

	// The receipt says what is LEFT, which is the one thing the caller does not
	// already know. A step marked done is not news to whoever just marked it,
	// and handing the checklist back would be the cost this tool exists to end.
	left := unfinishedSteps(plan.Steps)
	out := fmt.Sprintf("step %d marked %s. %d of %d still open.", n, state, len(left), len(plan.Steps))
	if len(left) == 0 {
		out += " Every step is settled."
	}
	return skill.Output{
		Name: "plan", Command: "plan step", Success: true,
		Content: out, RawOutput: out, DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// emitPlan puts the plan in front of the user, stamped with the conversation it
// belongs to so a chat working in the background cannot redraw the card
// somebody else is reading (§187, §234).
//
// Running is filled in here rather than stored: it is true of a turn, not of a
// plan (goal_run.go).
func (a *App) emitPlan(sessionID string, plan Plan) {
	if a.ctx == nil && a.emit == nil {
		return
	}
	if run := a.goalRunSet().get(sessionID); run != nil {
		plan.Running = true
		plan.SentBack = run.sentBack
		plan.StartedAt = run.startedAt
		plan.Paused = run.paused
	}
	a.emitEvent("plan:update", sessionEvent[Plan]{SessionID: sessionID, Data: plan})
}

// planReceipt is what goes back into the conversation, and it is deliberately
// not the plan.
//
// Handing the whole document back would put every byte of it into context on
// every write — the model would then have written the plan once and paid for it
// twice, which is the shape of waste this tool exists to end. What it needs to
// know is that the write landed, what the plan holds now, and that the user can
// already see it.
func planReceipt(action string, plan Plan, note string) string {
	var b strings.Builder
	if action == "amend" {
		fmt.Fprintf(&b, "plan amended (v%d). Changed: %s.", plan.Version, strings.Join(plan.Changed, ", "))
		if note != "" {
			b.WriteString(" " + note)
		}
	} else {
		fmt.Fprintf(&b, "plan written (v%d).", plan.Version)
	}
	headings := make([]string, 0, len(plan.Sections))
	for _, sec := range plan.Sections {
		headings = append(headings, sec.Heading)
	}
	b.WriteString("\nIt now holds: " + strings.Join(headings, ", ") + ".")
	b.WriteString("\nThe user can see it. Do not repeat it in your answer — say what changed, in a line, and stop.")
	if missing := missingHeadings(plan.Sections); len(missing) > 0 {
		b.WriteString("\nStill empty: " + strings.Join(missing, ", ") + ".")
	}
	return b.String()
}

// missingHeadings names the parts of the shape this plan has not filled in.
//
// Reported rather than refused. A small job's plan can leave a heading out and
// still be the correct plan — the stance says so in as many words — and a tool
// that rejected it would be enforcing a rule the prompt deliberately does not
// have. Saying which are empty is the useful half: a model that dropped one by
// accident is told, and one that meant to is not stopped.
func missingHeadings(have []PlanSection) []string {
	var out []string
	for _, want := range mode.PlanHeadings() {
		if !slices.ContainsFunc(have, func(s PlanSection) bool { return s.Heading == want }) {
			out = append(out, want)
		}
	}
	return out
}

// mergePlanSections replaces by heading and appends what is new, leaving
// everything the caller did not name exactly as it was. That is `amend`.
func mergePlanSections(existing, incoming []PlanSection) []PlanSection {
	out := slices.Clone(existing)
	for _, sec := range incoming {
		i := slices.IndexFunc(out, func(o PlanSection) bool { return o.Heading == sec.Heading })
		if i >= 0 {
			out[i] = sec
			continue
		}
		out = append(out, sec)
	}
	return out
}

// orderPlanSections puts the plan back into the order of the shape, whatever
// order the sections arrived in.
//
// A plan the user reads the same way every time is the thing the dial was
// turned for (§106.11), and an amend naturally arrives out of order — it names
// only what moved. Sorting on the way in rather than on the way out means the
// stored row is already right for every later reader, the card included.
//
// A heading the shape does not know keeps its place at the end rather than
// being dropped. The schema's enum already makes it unlikely, and throwing away
// something a model wrote because this build has not heard of the heading is
// the wrong way to be strict.
func orderPlanSections(in []PlanSection) []PlanSection {
	order := mode.PlanHeadings()
	rank := func(h string) int {
		if i := slices.Index(order, h); i >= 0 {
			return i
		}
		return len(order)
	}
	out := slices.Clone(in)
	slices.SortStableFunc(out, func(a, b PlanSection) int { return rank(a.Heading) - rank(b.Heading) })
	return out
}

// parsePlanSections reads sections off whatever the provider handed over, and
// canonicalises the headings.
//
// The case fold is not politeness. `amend` finds its section by name, so a model
// writing "What To Change" against a stored "What to change" would silently add
// a second section rather than replace the first — a plan that grows a duplicate
// heading every revision, with no error anywhere.
func parsePlanSections(raw any) ([]PlanSection, error) {
	if raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, errors.New("sections must be a list of {heading, body}")
	}
	canon := map[string]string{}
	for _, h := range mode.PlanHeadings() {
		canon[strings.ToLower(h)] = h
	}
	out := make([]PlanSection, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		heading := strings.TrimSpace(str(m["heading"]))
		body := strings.TrimSpace(str(m["body"]))
		if heading == "" || body == "" {
			continue
		}
		if c, known := canon[strings.ToLower(heading)]; known {
			heading = c
		}
		out = append(out, PlanSection{Heading: heading, Body: body})
	}
	return out, nil
}

// stepsFromProse lifts a numbered list out of the section that describes the
// work, for a call that wrote its steps as prose instead of passing them.
//
// A recovery and not a feature: the schema still asks for `steps`, the prompt
// still says so, and this is what keeps the card working on the turn where both
// of those are ignored. Silent by design — a model told off for the shape of a
// plan it wrote spends the next turn apologising instead of planning.
func stepsFromProse(sections []PlanSection) []PlanStep {
	body := sectionBody(sections, stepsHeading)
	if body == "" {
		return nil
	}
	var out []PlanStep
	for _, line := range strings.Split(body, "\n") {
		st, ok := parseStepLine(line)
		if !ok {
			continue
		}
		st.N = len(out) + 1
		out = append(out, st)
	}
	// One step is a list of one, which is a plan somebody wrote in a hurry and
	// not a sentence that happened to start with a digit. Two is where a
	// numbered list starts being a numbered list, and the cost of guessing wrong
	// here is a card offering to carry out a paragraph.
	if len(out) < 2 {
		return nil
	}
	return out
}

// stepsHeading is the section the work itself is described in, taken from the
// shape rather than spelled here (§106.11).
var stepsHeading = planStepsHeading()

func planStepsHeading() string {
	for _, h := range mode.PlanHeadings() {
		if strings.Contains(strings.ToLower(h), "to change") {
			return h
		}
	}
	return ""
}

// parsePlanSteps numbers the checklist as it arrives.
//
// The model sends text; the numbers are this side's, because they are the
// address that plan step and every verdict use. A model numbering its own steps
// could renumber them in the next amend, and a run would then be marking
// something other than what it carried out.
func parsePlanSteps(raw any) []PlanStep {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]PlanStep, 0, len(list))
	for _, item := range list {
		text := strings.TrimSpace(str(item))
		if text == "" {
			continue
		}
		out = append(out, PlanStep{N: len(out) + 1, Text: text})
	}
	return out
}

// planHeadingsOrder is mode.PlanHeadings() by another name, so plan_edit.go can
// fold a hand-typed heading onto the shape without importing internal/mode for
// one call. One source, still (§106.11).
func planHeadingsOrder() []string { return mode.PlanHeadings() }

// planMarkdown renders the plan the way `read` hands it over and the way a copy
// off the card writes it out: the title as a heading, each section under its
// own. One renderer, so the two cannot say the same plan differently.
func planMarkdown(p Plan) string {
	var b strings.Builder
	if p.Title != "" {
		b.WriteString("# " + p.Title + "\n\n")
	}
	for _, sec := range p.Sections {
		b.WriteString("**" + sec.Heading + "**\n" + sec.Body + "\n\n")
	}
	// The checklist with its states, because read is what a later turn — or a
	// later stance — uses to find out where the work got to. A plan handed back
	// without them would say what to do and never what has been done.
	for _, st := range p.Steps {
		mark := "[ ]"
		switch st.State {
		case planStepDone:
			mark = "[x]"
		case planStepDoing:
			mark = "[~]"
		case planStepFailed:
			mark = "[!]"
		}
		fmt.Fprintf(&b, "%s %d. %s", mark, st.N, st.Text)
		if st.Note != "" {
			b.WriteString(" — " + st.Note)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ---------------------------------------------------------------------------
// the row
// ---------------------------------------------------------------------------

// loadPlan reads a conversation's plan, or nil when it has none.
//
// nil rather than an empty Plan, and the callers depend on the difference: "no
// plan yet" and "a plan with nothing in it" are different answers to `read`, and
// `amend` treats the first as a write and the second as a merge.
func (a *App) loadPlan(sessionID string) (*Plan, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, nil
	}
	db, err := a.database()
	if err != nil {
		return nil, err
	}
	var title, sections, steps, updated string
	var version int
	err = db.QueryRow(
		`SELECT title, sections, steps, version, updated FROM plans WHERE session_id=?`, sessionID,
	).Scan(&title, &sections, &steps, &version, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var secs []PlanSection
	// A row that will not parse is reported as no plan rather than as a failure.
	// The alternative is a conversation that can never write a plan again
	// because it once wrote one this build cannot read, and `write` replaces the
	// row wholesale — so the honest recovery is to let it.
	if err := json.Unmarshal([]byte(sections), &secs); err != nil {
		return nil, nil
	}
	var st []PlanStep
	// Same recovery as the sections above: steps that will not parse read as
	// none rather than as a failure, and write replaces the row.
	_ = json.Unmarshal([]byte(steps), &st)
	return &Plan{Title: title, Sections: secs, Steps: st, Version: version, Updated: updated}, nil
}

func (a *App) savePlan(sessionID string, p Plan) error {
	db, err := a.database()
	if err != nil {
		return err
	}
	blob, err := json.Marshal(p.Sections)
	if err != nil {
		return err
	}
	steps, err := json.Marshal(p.Steps)
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`
	  INSERT INTO plans (session_id, title, sections, steps, version, created, updated)
	  VALUES (?, ?, ?, ?, ?, ?, ?)
	  ON CONFLICT(session_id) DO UPDATE SET
	    title=excluded.title, sections=excluded.sections, steps=excluded.steps,
	    version=excluded.version, updated=excluded.updated`,
		sessionID, p.Title, string(blob), string(steps), p.Version, now, now)
	return err
}

// SessionPlan is the window's own read, for drawing the card when a chat is
// reopened. Empty title and no sections means the conversation has no plan.
func (a *App) SessionPlan(sessionID string) Plan {
	p, err := a.loadPlan(sessionID)
	if err != nil || p == nil {
		return Plan{}
	}
	return *p
}
