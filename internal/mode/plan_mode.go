package mode

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// planModeTool is the assistant turning its own dial from ลงมือ down to
// วางแผน, in the middle of a turn, and carrying on under it.
//
// ## Why the model gets this when the user already has the picker
//
// The picker is a decision made BEFORE the request has been read; the assistant
// reads it first. The user types "จัดการเรื่อง X ให้ที" from ลงมือ because ลงมือ is
// where they always are, and one round in it is clear that this is a job to be
// thought through rather than started. Without this the only honest move is to
// say so and stop — spending a whole turn asking them to press a button they
// would have pressed themselves had they known.
//
// The other direction stays entirely theirs, and already works: a finished plan
// with ลงมือ pressed on it is ลงมือ (StartPlanRun, desktop/goal_run.go).
//
// ## One destination, and the two that are missing are missing on purpose
//
// **ลงมือ → วางแผน, and nothing else.**
//
//   - **Never wider.** Dial.Narrow refuses it, and that refusal is why this is
//     safe to ship. Everything COMPANY.md §6.3 guarantees about a frozen desk
//     survives a moving stance for exactly one reason — a stance only ever
//     subtracts — and a model that could widen its own would be the first thing
//     in this app able to hand ITSELF a tool, mid-turn, in a conversation the
//     user is watching run.
//   - **Never into คู่คิด**, which IS a legal narrowing and is still not offered.
//     คู่คิด is the stance for when the user wants the conversation instead of
//     the errand — a judgement about what THEY want, which is theirs to make.
//     An assistant that could put itself there would have a way of answering
//     "this needs looking up" with "let us just talk about it", and a tool the
//     model can reach when the work gets hard is a tool it will reach for when
//     the work gets hard. The one narrowing on offer costs it its writing tools
//     and hands it a document to produce; that is the opposite of an exit.
//
// One destination is also why there is no `to` parameter. A schema is paid for
// on every request this desk sends, and an enum with one member is a token
// spent to describe a choice that does not exist.
//
// ## No prompt layer, and that was the owner's call (9 ก.ย.)
//
// The first build of this had a paragraph in internal/prompt saying when to
// reach for it. He cut it — *"ฝังในชั้นของระบบดีกว่า คอนแท็คจะได้ไม่บวมเพิ่ม"* —
// so everything the model is told about this control has to fit in the
// definition below, which is one static block at the head of the request and
// therefore cached rather than re-read. That is a real constraint on the
// wording and not a note about where a file lives: if the description does not
// carry it, nothing does.
//
// ## What it hands back
//
// วางแผน's own Direction(), verbatim. The system prompt was assembled at
// bootstrap and is not rebuilt mid-turn, so the direction cannot arrive the way
// it does when the user works the picker — but a tool result lands LATER in the
// context than the prompt does, and later context outweighs earlier (§106.4).
// The same words, through the door that is open.
type planModeTool struct{ dial *Dial }

// NewPlanModeTool builds the switch over a live dial.
//
// A nil dial answers nil rather than a tool that fails on every call: an engine
// with nothing to turn should carry no switch at all. The one caller registers
// what it gets and skips a nil.
func NewPlanModeTool(dial *Dial) skill.Tool {
	if dial == nil {
		return nil
	}
	return &planModeTool{dial: dial}
}

func (*planModeTool) Name() string { return "plan_mode" }

func (*planModeTool) Description() string {
	return "เปลี่ยนเทิร์นนี้เป็นโหมดวางแผน แล้วทำงานต่อภายใต้โหมดนั้น"
}

func (*planModeTool) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"why": map[string]any{
				"type": "string",
				"description": "One short sentence, in the user's language, saying what about this " +
					"request needs planning first. They see it on the mode button.",
			},
		},
		"required":             []string{"why"},
		"additionalProperties": false,
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name: "plan_mode",
			// This is the ONLY place the model is told when to use this — there is
			// no prompt layer (see the type's note), so every sentence here is
			// carrying something.
			//
			//   1. "keep working" — without it the switch reads as an ending and
			//      the model hands the turn back, costing the user the round AND
			//      the work.
			//   2. "before you start" — a switch after three edits is a stance
			//      that arrives too late to mean anything.
			//   3. the last sentence — a control that makes the job easier is one
			//      a model reaches for when the job is hard, and "I have moved us
			//      to planning" is a far more comfortable thing to write than a
			//      finished piece of work.
			Description: "Switch this turn into planning mode and keep working under it. " +
				"Use it in the first round or two, before you start, when the request turns out to be too " +
				"big or too vague to begin without guessing and what the user actually needs first is a " +
				"plan they can approve. You keep every tool that reads and lose the ones that write, run " +
				"or delegate, and the turn ends in a written plan instead of the work. " +
				"It is one-way: it cannot give tools back, and returning to acting is the user's own " +
				"press. Do not use it to duck a hard job, and do not use it for an ordinary request you " +
				"could simply carry out.",
			Parameters: payload,
		},
	}
}

// Execute is the terminal spelling. Every built-in has one, and this one earns
// it twice over: a stance is otherwise a desktop picker, so a CLI session has
// no other way to reach the dial at all.
func (t *planModeTool) Execute(ctx context.Context, input skill.Input) (skill.Output, error) {
	words, _ := input["args"].([]string)
	return t.ExecuteTool(ctx, map[string]any{"why": strings.Join(words, " ")})
}

func (t *planModeTool) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	start := time.Now()
	why, _ := args["why"].(string)
	why = strings.TrimSpace(why)

	if err := t.dial.Narrow(StancePlan); err != nil {
		// Refused rather than crashed, so the model reads the reason and moves
		// on. There is one refusal it can actually meet — "already in plan",
		// from a session the user put there while this call was in flight — and
		// the right next move for that is to carry on planning, not to retry.
		return skill.Output{
			Name:       "plan_mode",
			Command:    "plan_mode",
			Content:    err.Error(),
			RawOutput:  err.Error(),
			Stderr:     err.Error(),
			Success:    false,
			DurationMs: time.Since(start).Milliseconds(),
		}, err
	}

	// Two readers, two texts, which is why this is built by hand rather than
	// through the shared helper. The user gets the sentence the model wrote
	// about their own request — it is what the mode button says when it moves.
	// The model gets วางแผน's direction, because that is what it now has to work
	// under and the prompt it booted with does not say it.
	shown := "วางแผน"
	if why != "" {
		shown += " — " + why
	}
	forModel := "Planning mode is on for the rest of this turn. Keep going under it; do not hand the " +
		"turn back to ask permission for the mode you just chose.\n\n" + StancePlan.Direction()
	return skill.Output{
		Name:       "plan_mode",
		Command:    "plan_mode",
		Content:    shown,
		RawOutput:  forModel,
		Success:    true,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}
