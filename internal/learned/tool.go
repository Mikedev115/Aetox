package learned

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/callfault"
	"github.com/Mikedev115/Aetox/internal/model"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// Proposal is one change the agent wants made to what it has learned. It is
// not applied here — Propose queues it, and a human decides.
type Proposal struct {
	Kind   string // "memory" today; "skill" and "prompt" queue through the same door
	Scope  string
	Op     string
	Before string
	Body   string
	Reason string
}

// Result is what queuing produced, so the tool can tell the model the truth
// about what happened rather than always claiming success.
type Result struct {
	ID        int64
	Duplicate bool // the same fact was already waiting; ID is that row's
	// Prior is set when nothing was queued because the user already answered
	// this fact — approved it, or turned it down. The door remembers so the
	// model does not have to: measured 11 ก.ย., a fact refused on the 9th was
	// proposed again on the 10th in other words, and again on the 11th.
	Prior *Prior
}

// Prior is the earlier proposal a new one turned out to restate, and what the
// user said about it. State is one of the queue's own: "approved" means the
// line is memory now, "rejected" means the user said no to it.
type Prior struct {
	ID        int64
	State     string
	Body      string
	DecidedAt string
}

// Proposer is the door. Implemented by the desktop app against the
// pending_changes table; kept as an interface so this package never learns
// what a database is, and so a test can watch what the tool proposed without
// one.
type Proposer interface {
	Propose(Proposal) (Result, error)
}

// MemoryTool is the model-facing half. One instance per scope: the main
// agent's registry gets a MainScope one, and every delegate gets one bound to
// its own profile name, which is what makes "an agent learns only inside its
// own scope" true by construction rather than by the model's cooperation.
type MemoryTool struct {
	Scope    string
	Proposer Proposer
	// Project is the second destination: the ROOT PATH of the folder this
	// session is focused on, empty when none is (the unfocused session, the
	// pre-project hosts, every delegate). The path rather than a key so there
	// is one spelling of "which project" in the system — ProjectScope does the
	// rest.
	//
	// The desk the session was opened at cannot stand in for it, which is why
	// there is no desk destination here. โต๊ะโค้ด is the same desk in every
	// repository, so a decision made in one would arrive as advice in the
	// next; this is the axis where "we settled on X here, because Y" can be
	// kept without being carried anywhere it is not true.
	Project string
	// ProjectFirst is the desk's memory architecture (mode.MemoryRule, §184):
	// true means an unqualified line lands in Project's own file, and
	// `everywhere` is the explicit road back to the shared one. The desk
	// answers this rather than the model because the model demonstrably does
	// not: measured 25 ส.ค., 7 of the 11 memory calls ever made sent no
	// `where` at all, and "did not decide" was indistinguishable from
	// "decided on everywhere". False — the assistant's architecture, and
	// every session before desks declared one — keeps the shared file as the
	// default it always was. With no Project it is inert, so an unfocused
	// coding session falls back to shared rather than to a junk drawer.
	ProjectFirst bool
	// Desk is the name of the desk this session sits at, and with ProjectFirst
	// it is what makes that desk's memory ITS OWN (11 ก.ย.): a project-first
	// desk that names itself writes its cross-project lines into modes/<Desk>.md
	// rather than into the assistant's MEMORY.md, and its `where` offers
	// this-desk | this-project instead of everywhere | this-project.
	//
	// The owner's finding that asked for it: the four lines MEMORY.md held on
	// his machine were all the assistant's — a GPU, a Framer workflow, an OMEN
	// laptop — and not one was something โต๊ะโค้ด had ever used, yet every
	// coding session paid for them, and anything coding learned that was true
	// across repositories had nowhere of its own to go (§184 closed the desk
	// scope to writes because no model ever chose it; it was never offered as
	// the default). USER.md stays the one file every desk shares, because who
	// somebody is does not change with the room.
	//
	// Empty for the assistant, the CLI and every delegate, all of which keep
	// the shared file as their destination exactly as before.
	Desk string
}

// ownMemory reports whether this session's cross-project destination is the
// desk's own file rather than the assistant's shared one — see Desk.
func (t *MemoryTool) ownMemory() bool {
	return t.ProjectFirst && strings.TrimSpace(t.Desk) != ""
}

func (*MemoryTool) Name() string { return "memory" }

// forWorker reports whether this instance belongs to one worker rather than to
// the assistant, its desk or a project.
//
// It decides which words the model reads, and the words are the whole feature.
// Measured 31 ส.ค. on the owner's own machine: 557 tool calls made by agents,
// **none of them memory**, against 16 by the assistant — and no worker had a
// MEMORY.md at all. The wiring was complete the whole time. What every worker
// read was a description written for the assistant's job: "keep what you learn
// about this user", "what they tell you about themselves", "a fact the user
// states about themselves is already the evidence". A worker never speaks to
// the user. It takes a brief and returns a result, so the tool as described had
// no occasion to fire and correctly never fired — while the file it would have
// written is headed "What you have learned doing this job before".
//
// One tool, two jobs, two descriptions. The bar underneath is the same in both
// and always was: will this still be true, and still change what you do, on a
// day nobody remembers this conversation.
func (t *MemoryTool) forWorker() bool {
	scope := strings.TrimSpace(t.Scope)
	if scope == "" || scope == MainScope {
		return false
	}
	if _, isDesk := SplitModeScope(scope); isDesk {
		return false
	}
	_, isProject := SplitProjectScope(scope)
	return !isProject
}

// Destinations the `where` parameter can name, in the order they are offered.
// Which file an unqualified line lands in is the desk's own architecture
// (ProjectFirst), not a choice the model is offered; `this-desk` is offered
// only at a desk that keeps its own memory (Desk), where it is that desk's
// cross-project file and `everywhere` — the assistant's file — is not on the
// menu at all, because that desk no longer reads it.
const (
	whereEverywhere = "everywhere"
	whereDesk       = "this-desk"
	whereProject    = "this-project"
)

// What a line is ABOUT, which is a different question from where it goes and
// is now asked first.
//
// The two were one question until 6 ก.ย. and the machine had been answering
// them separately the whole time: of sixteen proposals, the seven about the
// user were approved four times and the eight about the machine zero times.
// One description asked the model to clear one bar; the person applying it held
// two. So the model is asked which of the two it is writing, and the answer
// picks the file — `user` always lands in the profile, whatever desk this is
// and whatever `where` says, because who somebody is cannot be true of one
// project only.
//
// `self` came 14 ก.ย. 2026. With two words the assistant's own file could only
// ever hold facts about the computer, and the owner, reading a day of
// proposals, saw the shape that made: *"มันจดทุกอย่างลงเกี่ยวกับผม แต่ไม่ค่อยเห็น
// มันจำอะไรเกี่ยวกับตัวเองเลย"*. What the work taught the assistant — an
// approach that landed or failed with this person, a habit it now keeps —
// had no word to be written under and so was never written. Same file as
// `machine` (the head's own, §266 "the assistant's" layer): the split that
// matters is user / not-user, and a third file would be a third page.
const (
	aboutUser    = "user"
	aboutMachine = "machine"
	aboutSelf    = "self"
)

func (t *MemoryTool) whereOptions() []string {
	out := []string{whereEverywhere}
	if t.ownMemory() {
		out = []string{whereDesk}
	}
	if t.Project != "" {
		out = append(out, whereProject)
	}
	return out
}

// whereDescription says what each destination is FOR, in one sentence each,
// and which of them an unsaid word means — because the default is the desk's
// call now, and a description that left it implicit would teach the assistant
// desk's default at every desk.
func (t *MemoryTool) whereDescription() string {
	b := strings.Builder{}
	projectFirst := t.ProjectFirst && t.Project != ""
	// It stopped saying "this user" on 6 ก.ย.: a fact about the user has its own
	// file now and reaches it whatever this says, so leaving the word here would
	// offer a destination that the same call has already overruled.
	everywhere := "Only read when about is self or machine. everywhere for a fact that is true of this computer " +
		"whatever you are working on."
	if t.ownMemory() {
		// This desk's own file: what its work taught that holds in every
		// repository, and that the assistant desk never pays for.
		everywhere = "Only read when about is self or machine. this-desk for something this kind of work taught you " +
			"that holds in every project — a tool this machine lacks, a convention you follow wherever " +
			"you write code. Only sessions at this desk read it."
	}
	if !projectFirst {
		everywhere = strings.Replace(everywhere, "everywhere ", "everywhere (default) ", 1)
		everywhere = strings.Replace(everywhere, "this-desk ", "this-desk (default) ", 1)
	}
	b.WriteString(everywhere)
	if t.Project != "" {
		project := " this-project for something that is true only in " + filepath.Base(t.Project) +
			" — what was decided here and why, a convention this codebase holds to. " +
			"It follows nobody into another project."
		if projectFirst {
			project = strings.Replace(project, "this-project ", "this-project (default) ", 1)
		}
		b.WriteString(project)
	}
	return b.String()
}

func (t *MemoryTool) Description() string {
	if t.forWorker() {
		return "Remember something durable this job taught you, or revise something already remembered. " +
			"The user approves it before it takes effect."
	}
	return "Remember something durable about this user or machine, or revise something already remembered. " +
		"The user approves it before it takes effect."
}

// ToolDefinition states what belongs in memory as a principle rather than a
// list of triggers: the failure this guards against is a memory that fills
// with restatements of the current task, and no enumeration of forbidden
// topics would prevent that. What separates a fact worth keeping from noise is
// whether it will still be true, and still change what the agent does, on a
// day nobody remembers this conversation.
//
// It was written entirely against that failure, and against the opposite one
// it was silent — so the opposite one is what shipped. Owner, 18 ส.ค.: the
// agent remembers only when told to remember, and a user saying who they are
// and what they are building goes straight past this tool. Two clauses did it,
// both aimed at the agent's own guesses and neither saying so. "Something you
// worked out" is a source test that the user's own sentence fails. "Anything
// you have not actually seen borne out" is an evidence test, and the agent
// waits for corroboration of a fact whose only possible source has just
// spoken.
//
// What generalises, and what the text below now says: the bar is whether a
// line will still be true and still matter, never where it came from. When the
// user states something about themselves, that IS the evidence — there is no
// later observation that would confirm it further. The cost sentence is what
// holds the other side; it always was.
//
// 6 ก.ย. added the second question — `about`, user or machine — and two
// sentences the file's own history had earned. The bar was one bar in this text
// and two bars in the person applying it (see UserScope for the count), so the
// model is now asked which of them it is clearing. The declarative-fact rule
// is the other: the only proposal this machine ever made in the imperative
// ("ก่อนสร้าง UI … ต้องเปิดอ่านสกิลก่อน") was refused, and it deserved to be —
// an order stored here is read in every later session and outranks whatever the
// user asks for then. Hermes states the same rule and states the reason better
// than a list of examples could, so it is stated rather than exemplified. Where
// that lesson does belong is the skill for that work, which is one more
// sentence and closes the only exit the refusal left open.
func (t *MemoryTool) ToolDefinition() model.ToolDefinition {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"op": map[string]any{
				"type":        "string",
				"enum":        []string{OpAdd, OpReplace, OpRemove},
				"description": "add a new line, replace an existing one, or remove one. Defaults to add.",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "The fact, in one sentence. Write it in the same language the user communicates in (e.g. Thai if the user speaks Thai). Required for add and replace.",
			},
			"old": map[string]any{
				"type":        "string",
				"description": "Distinctive words from the line being replaced or removed — enough to match it uniquely.",
			},
			"why": map[string]any{
				"type":        "string",
				"description": "What in this session showed you this, in the user's language. The user reads it when deciding whether to keep it.",
			},
		},
		"additionalProperties": false,
	}
	// A worker is never offered this and its block does not grow by a byte: it
	// does not talk to the user, so it has no profile to write and no evidence
	// to write one from. That is the same boundary §184.5 drew on the read side,
	// held here by the schema rather than by the model's cooperation — there is
	// no word a delegate's tool call can say that reaches USER.md.
	if !t.forWorker() {
		schema["properties"].(map[string]any)["about"] = map[string]any{
			"type": "string",
			"enum": []string{aboutUser, aboutMachine, aboutSelf},
			"description": "Required. user for an enduring fact about the person you are talking to — who they are, " +
				"what they are building, how they want to be worked with, what a request of theirs " +
				"reliably turns out to mean. self for what the work taught YOU — an approach that turned out " +
				"right or wrong with this person, a rule you now keep, what you found out about your own tools here. " +
				"machine for a permanent, global hardware or system constraint " +
				"that holds across all projects (e.g. global proxy, strict OS limit). NEVER use machine for local directory paths, " +
				"command output, or script workarounds.",
		}
		schema["required"] = []string{"about"}
	}
	// The parameter appears only for a session that has somewhere else to put a
	// line — a focused project. The tool block rides in every request, so an
	// option nobody can use is a bill with no benefit; a session with one
	// destination is offered no choice at all, byte-for-byte the block it
	// always sent.
	if where := t.whereOptions(); len(where) > 1 {
		schema["properties"].(map[string]any)["where"] = map[string]any{
			"type":        "string",
			"enum":        where,
			"description": t.whereDescription(),
		}
	}
	payload, _ := json.Marshal(schema)
	return model.ToolDefinition{
		Type: "function",
		Function: model.ToolFunction{
			Name:        "memory",
			Description: t.definitionText(),
			Parameters:  payload,
		},
	}
}

// definitionText is what the model reads about when to call this. Two texts,
// one bar - see forWorker for the measurement that split them.
func (t *MemoryTool) definitionText() string {
	if t.forWorker() {
		return "Skills come first. Memory is a narrow exception for enduring conventions that apply to EVERY future job of this kind. " +
			"Worth keeping: how this user's codebase or work is permanently structured — an enduring convention their files hold to, " +
			"or what a recurring request of theirs reliably turns out to mean — anything that will still be true next month and would change " +
			"how you do the same job again. " +
			"STRICTLY FORBIDDEN: Do NOT propose memories for transient script errors, one-off workarounds, directory paths, " +
			"local command output, or anything discoverable by reading files or tools at runtime. " +
			"Write both the fact and the reason in the user's language (the language they communicate with you in), " +
			"so they can review and approve it naturally. " +
			"This file is yours alone. Nobody else reads it, nothing you write reaches the assistant or another " +
			"worker, and you are the one who will pay for it: a remembered line costs context on every job you " +
			"are ever given again, so a wrong or idle one is paid for forever. " +
			"Nothing takes effect until the user approves it, and it reaches you at the start of the next job, " +
			"not this one."
	}
	return "Skills come first. Memory is a narrow exception for enduring facts that apply to EVERY session regardless of task. " +
		"Worth keeping in USER.md (about: user): what they tell you about themselves — who they are, what they are building, " +
		"how they want to be worked with. A fact the user states about themselves is already the evidence for it; " +
		"their name is not one of them, it reaches you in the prompt already. " +
		"Worth keeping in your own file (about: self): what this work taught you — an approach that landed or failed " +
		"with this person, a habit you now keep, what you learned about your own tools here. That file is what makes " +
		"you the same colleague next month; a session that only wrote about the user learned nothing itself. " +
		"Also in your own file (about: machine): only permanent, global environment constraints that will still be true next month " +
		"and would change what you do (e.g. hardware limits, global proxy). " +
		"STRICTLY FORBIDDEN: Do NOT propose memories for transient errors, command failures, script debugging, date/locale formatting quirks, " +
		"one-off workarounds, directory paths discovered during a task, or anything discoverable by running a command. " +
		"Never guess user intent or record speculative conclusions. " +
		"Write both the fact and the reason in the user's language (the language they communicate with you in), " +
		"so they can review and approve it naturally. " +
		"Write a declarative fact, never an instruction to yourself: \"they prefer short answers\", not \"always " +
		"answer briefly\" — an order kept here outranks what they ask you for next month. " +
		"How to do a kind of work belongs in a skill, not here. " +
		"A remembered line costs context on every request this agent ever makes again, so a wrong or " +
		"idle one is paid for forever. Nothing here takes effect until the user approves it, and it " +
		"reaches you at the start of the next session, not this one."
}

// Execute exists because skill.Skill requires it. Memory is model-only: a
// person editing what the agent learned does it in the file or the review
// list, both of which show them what is already there.
func (t *MemoryTool) Execute(_ context.Context, _ skill.Input) (skill.Output, error) {
	return skill.Output{}, fmt.Errorf("memory is called by the model; edit the memory folder directly to change it by hand")
}

func (t *MemoryTool) ExecuteTool(_ context.Context, args map[string]any) (skill.Output, error) {
	start := time.Now()
	fail := func(err error) (skill.Output, error) {
		return skill.Output{
			Name:       "memory",
			Content:    err.Error(),
			Command:    "memory",
			Success:    false,
			Stderr:     err.Error(),
			DurationMs: time.Since(start).Milliseconds(),
		}, err
	}
	// id travels with the receipt so the chat can draw the proposal where it was
	// made, with the decision on it. A duplicate carries the id of the row
	// already waiting: the second attempt is the same proposal, and the card
	// under this answer should be about that one rather than about nothing.
	ok := func(msg, command string, id int64) (skill.Output, error) {
		return skill.Output{
			Name:       "memory",
			Content:    msg,
			Command:    command,
			Success:    true,
			ProposalID: id,
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}

	if t.Proposer == nil {
		return fail(fmt.Errorf("memory is not available in this session"))
	}

	op := strings.TrimSpace(stringArg(args, "op"))
	if op == "" {
		op = OpAdd
	}
	text := strings.TrimSpace(stringArg(args, "text"))
	old := strings.TrimSpace(stringArg(args, "old"))
	why := strings.TrimSpace(stringArg(args, "why"))
	// The default destination is the desk's architecture, not the model's
	// guess (§184): a project-first desk lands an unsaid `where` in the
	// project's own file, everything else lands it in the shared one — the
	// scope that was the only one before this parameter existed. An invented
	// word is an unsaid word: it means the desk's default, never nowhere.
	scope := t.Scope
	if t.ownMemory() {
		// A desk with its own memory never lands a line in the assistant's
		// file: unfocused, its cross-project file is the floor.
		scope = ModeScope(t.Desk)
	}
	if t.ProjectFirst && t.Project != "" {
		scope = ProjectScope(t.Project)
	}
	// `about` is required and has no default, which is the opposite of `where`
	// one paragraph down, and the difference is who can answer. §184 moved
	// `where`'s default onto the desk because the desk knows: coding work
	// settles project decisions wherever it is done. Nothing but the model knows
	// whether the sentence it just wrote is about the person or about the
	// computer — a coding session learns both — so an unsaid word here would be
	// §184's own finding repeating itself: a parameter whose absence is
	// indistinguishable from one of its values is a bias, not a choice.
	//
	// Refused rather than guessed, and the refusal names both words, because the
	// model can act on it in the same turn: this is the door §139 opened for a
	// `replace` that names nothing. Every refusal of the call's shape below is
	// marked as the caller's (internal/callfault): the model can fix each one
	// on its next call, and the problems page has no business hearing of it.
	if !t.forWorker() {
		switch strings.TrimSpace(stringArg(args, "about")) {
		case aboutUser:
			// Whatever the desk's architecture says and whatever `where` says.
			// Who somebody is cannot be true of one project only, so there is no
			// destination left to choose.
			scope = UserScope
		case aboutMachine, aboutSelf:
			// Both land in the head's own file: what the computer is and what
			// the work taught are the two things that are true of this
			// assistant rather than of the person, and one file holds them.
			switch strings.TrimSpace(stringArg(args, "where")) {
			case whereEverywhere, whereDesk:
				// The cross-project destination, whichever word this desk was
				// offered for it. A word the desk was NOT offered still means
				// its cross-project file — `everywhere` at โต๊ะโค้ด is the
				// desk's own file, never the assistant's, because that desk
				// does not read the assistant's and a line sent there would be
				// a line nobody reads.
				scope = t.Scope
				if t.ownMemory() {
					scope = ModeScope(t.Desk)
				}
			case whereProject:
				if t.Project != "" {
					scope = ProjectScope(t.Project)
				}
			}
		case "":
			return fail(callfault.Newf(
				"about is required — %q for a fact about the person you are talking to, %q for what the work taught you, %q for a fact about this computer or setup",
				aboutUser, aboutSelf, aboutMachine))
		default:
			return fail(callfault.Newf("about must be %q, %q or %q", aboutUser, aboutSelf, aboutMachine))
		}
	}

	switch op {
	case OpAdd:
		if text == "" {
			return fail(callfault.New("text is required to remember something"))
		}
	case OpReplace:
		if text == "" || old == "" {
			return fail(callfault.New("replace needs both old (what to find) and text (what it becomes)"))
		}
	case OpRemove:
		if old == "" {
			return fail(callfault.New("remove needs old — distinctive words from the line to forget"))
		}
	default:
		return fail(callfault.Newf("unknown op %q — use add, replace or remove", op))
	}

	// Before the queue rather than before the file: a card the user cannot read
	// correctly must never be drawn for them to approve. See Screen for why this
	// is the only check of its kind here.
	if err := Screen(text); err != nil {
		return fail(err)
	}

	// Refused here rather than at approval: a proposal that cannot be applied
	// would sit in the user's review list looking like progress, and the agent
	// would never learn that it needs to consolidate.
	if op == OpAdd && Full(scope, len(text)+2) {
		// The limit is per file since the profile got its own, a quarter the
		// size, and it is the one the model will meet first. Naming the number
		// this scope actually has is the difference between "consolidate" and
		// "consolidate down to what".
		return fail(fmt.Errorf(
			"this memory is full (limit %d bytes) — replace or remove a line that has stopped being useful before adding another",
			MaxBytesFor(scope)))
	}

	// Same door, same reason, and the case that made it necessary is the
	// ordinary one rather than a rare one: the agent proposes a line, the user
	// corrects it two messages later, and the agent revises the line it just
	// proposed. But a proposal is not memory — nothing is, until somebody
	// approves it — so `old` names a line that is not in any file, and the
	// revision queued as a card that could never be approved. Its only exit was
	// ไม่เอา, which records the user refusing a line they had just asked for.
	//
	// What the model needs told is which of the two things is true, because the
	// answer changes what it should do next: the line is not there, so revising
	// it is not the move — adding it is.
	if op == OpReplace || op == OpRemove {
		if !Has(scope, old) {
			if op == OpRemove {
				return fail(fmt.Errorf("nothing remembered contains %q, so there is nothing to forget", old))
			}
			return fail(fmt.Errorf(
				"nothing remembered contains %q — a proposal you made earlier is not memory until the user approves it. Use add if this should be remembered on its own",
				old))
		}
	}

	res, err := t.Proposer.Propose(Proposal{
		Kind:   "memory",
		Scope:  scope,
		Op:     op,
		Before: old,
		Body:   text,
		Reason: why,
	})
	if err != nil {
		return fail(fmt.Errorf("could not queue this for approval: %w", err))
	}
	if res.Duplicate {
		return ok("Already waiting for the user to approve — not queued twice.", "memory "+op, res.ID)
	}
	// Told as a result rather than an error: the model did nothing wrong by
	// the tool's contract, and a failure here would be recorded as one and
	// read by the problems page. What it needs is the sentence the user
	// already answered and the answer, so the same question is not asked a
	// third time in a fourth spelling. No proposal id — there is no card to
	// draw under this answer, and the decided row is not this turn's.
	if res.Prior != nil {
		return ok(priorMessage(res.Prior), "memory "+op, 0)
	}
	return ok(
		"Queued for the user to approve. It does not affect this session; once approved it is there from the next one on.",
		"memory "+op, res.ID)
}

// priorMessage says what the user already decided about this fact, in words
// the model can act on. The rejected case is the one that matters: a line the
// user refused is refused in every rewording, and saying so is what stops the
// rewordings.
func priorMessage(p *Prior) string {
	when := p.DecidedAt
	if len(when) >= 10 {
		when = when[:10]
	}
	switch p.State {
	case "rejected":
		return fmt.Sprintf("Not queued. The user already turned this down (%s): %q. "+
			"Do not propose it again, or the same thing in other words — a line they refused once is refused.",
			when, p.Body)
	default:
		return fmt.Sprintf("Not queued. This is already remembered (approved %s): %q. "+
			"If it needs revising, use replace with old naming that line.", when, p.Body)
	}
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	s, _ := args[key].(string)
	return s
}
