package main

// The plan, edited by the person whose plan it is.
//
// Everything else about this document is written by the model: `plan write`
// makes it, `plan amend` changes a section, `plan step` marks a step. That is
// the right default and it is not enough. The owner, on 2026-09-08, in three
// words: *"ท่าน แก้เองด้วยมือ ควรทำได้"*.
//
// **The whole card as markdown, and the format is the one the tool already
// prints.** He chose that shape over per-section editing, and the objection to
// it — parsing a plan back out of prose is fragile — is answered by owning both
// ends: `planMarkdown` writes exactly what `parsePlanMarkdown` reads, so the
// round trip is a property of one file rather than a guess about what a person
// might type. `TestPlanMarkdownRoundTrips` is what keeps it that way; a change
// to either side that forgets the other fails there rather than in front of
// somebody who has just lost their plan.
//
// What a human types is not what a program prints, so the parser is generous in
// the directions that cost nothing:
//
//   - The checklist can be written `1.`, `- [x] 1.` or `[x] 1.` — a person
//     reaching for a markdown to-do list writes the second, and refusing it
//     would be the parser being right about its own output while wrong about
//     the user.
//   - Numbers are re-assigned on the way in. Somebody who inserts a step in the
//     middle should not have to renumber the rest, and a step's number is an
//     address this side owns anyway (parsePlanSteps).
//   - A step whose text is unchanged KEEPS ITS STATE. Editing the wording of
//     step 4 must not un-tick steps 1 through 3, and matching on text is what
//     makes that true without asking the user to preserve markers they did not
//     write.
//
// **The model is told at the start of the next turn, not now** (his call, and
// the right one). Interrupting a running turn to say the plan moved would have
// the model arguing with a document mid-thought; the next turn simply reads a
// note saying the plan was edited by hand, and the plan it reads is the new one.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// planStepLine matches a checklist row in every spelling a person might use.
//
//	[x] 3. text — note      the tool's own output
//	- [ ] 3. text           a markdown to-do list
//	3. text                 a plain numbered list
//
// The mark, the bullet and the number are each optional; the text is not.
var planStepLine = regexp.MustCompile(`^\s*(?:[-*]\s*)?(?:\[([ xX~!])\]\s*)?(?:(\d+)[.)]\s*)?(.+)$`)

// planHeadingLine matches a section heading as planMarkdown writes it, and as
// markdown writes it, because a person editing prose reaches for `##`.
var planHeadingLine = regexp.MustCompile(`^\s*(?:#{1,4}\s*)?\*\*(.+?)\*\*\s*$|^\s*#{2,4}\s+(.+?)\s*$`)

// parsePlanMarkdown reads an edited plan back.
//
// `old` is what was there before, and it is not optional: step states live
// nowhere in the text a person edits, so they are carried across by matching
// text. Without it every hand edit would silently un-tick the whole checklist.
func parsePlanMarkdown(src string, old *Plan) Plan {
	out := Plan{}
	if old != nil {
		out.Title = old.Title
		out.Version = old.Version
	}
	var (
		heading string
		body    []string
		steps   []PlanStep
		inSteps bool
	)
	flush := func() {
		if heading == "" {
			return
		}
		out.Sections = append(out.Sections, PlanSection{
			Heading: canonicalHeading(heading),
			Body:    strings.TrimSpace(strings.Join(body, "\n")),
		})
		heading, body = "", nil
	}

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "# ") && !strings.HasPrefix(t, "##") {
			flush()
			out.Title = strings.TrimSpace(strings.TrimPrefix(t, "# "))
			inSteps = false
			continue
		}
		if m := planHeadingLine.FindStringSubmatch(line); m != nil && strings.TrimSpace(line) != "" {
			name := m[1]
			if name == "" {
				name = m[2]
			}
			if strings.TrimSpace(name) != "" {
				flush()
				heading = strings.TrimSpace(name)
				inSteps = false
				continue
			}
		}
		// A checklist row. Recognised anywhere, because a person moving a step
		// around does not think about which heading it fell under — the steps
		// are their own list and the tool renders them below the prose.
		if st, ok := parseStepLine(line); ok {
			steps = append(steps, st)
			inSteps = true
			continue
		}
		if strings.TrimSpace(line) == "" && inSteps {
			continue
		}
		inSteps = false
		if heading != "" {
			body = append(body, line)
		}
	}
	flush()

	// Renumber, then carry the states across by text.
	for i := range steps {
		steps[i].N = i + 1
	}
	if old != nil {
		carryStepStates(steps, old.Steps)
	}
	out.Steps = steps
	out.Sections = orderPlanSections(out.Sections)
	return out
}

// parseStepLine reads one checklist row, and refuses anything that is plainly
// prose.
//
// The guard is the point. Without it every sentence in a section body would
// match `(.+)` and become a step, which is the failure that would make hand
// editing worse than useless: a plan that grows nine steps because somebody
// wrote a paragraph. A row qualifies only if it carried a MARK or a NUMBER —
// something the writer did on purpose.
func parseStepLine(line string) (PlanStep, bool) {
	m := planStepLine.FindStringSubmatch(line)
	if m == nil {
		return PlanStep{}, false
	}
	mark, num, text := m[1], m[2], strings.TrimSpace(m[3])
	if text == "" || (mark == "" && num == "") {
		return PlanStep{}, false
	}
	// A number alone is only a step when it opens the line as a list item. This
	// is already true of the regex, and the note is here because the alternative
	// reading — any line containing a number — is the one somebody will try to
	// "fix" it to later.
	if num != "" {
		if _, err := strconv.Atoi(num); err != nil {
			return PlanStep{}, false
		}
	}
	st := PlanStep{Text: text}
	// The note the tool renders after an em dash comes back off it. A person who
	// typed their own dash gets the same treatment, which is the reading they
	// meant: everything after it is about the step rather than part of its name.
	if i := strings.Index(text, " — "); i > 0 {
		st.Text = strings.TrimSpace(text[:i])
		st.Note = strings.TrimSpace(text[i+len(" — "):])
	}
	switch mark {
	case "x", "X":
		st.State = planStepDone
	case "~":
		st.State = planStepDoing
	case "!":
		st.State = planStepFailed
	}
	return st, true
}

// carryStepStates gives an edited step back the state it had, matched on text.
//
// Matched on text and not on number, because a number is exactly what a hand
// edit moves. Somebody inserting a step at the top would otherwise shift every
// state down one — the checklist would come back saying the wrong work was
// finished, which is worse than saying nothing.
//
// A state the person typed themselves wins: they wrote `[x]`, and overwriting
// it with what the row used to be would be the app arguing with the edit.
func carryStepStates(now []PlanStep, before []PlanStep) {
	was := map[string]PlanStep{}
	for _, b := range before {
		was[strings.TrimSpace(b.Text)] = b
	}
	for i := range now {
		if now[i].State != planStepTodo {
			continue
		}
		if b, ok := was[strings.TrimSpace(now[i].Text)]; ok {
			now[i].State = b.State
			if now[i].Note == "" {
				now[i].Note = b.Note
			}
		}
	}
}

// canonicalHeading folds a heading onto the shape's spelling, so a person who
// typed one in their own casing does not end up with a section the tool cannot
// find by name (parsePlanSections makes the same fold for the same reason).
func canonicalHeading(h string) string {
	for _, want := range planHeadingsOrder() {
		if strings.EqualFold(strings.TrimSpace(h), want) {
			return want
		}
	}
	return strings.TrimSpace(h)
}

// ---------------------------------------------------------------------------
// what the window calls
// ---------------------------------------------------------------------------

// SavePlanText stores a plan the user edited by hand, and answers with a
// refusal or "".
//
// The version goes UP, exactly as a model's amend does, because the card's
// revision mark is about the document changing and not about who changed it.
// What is recorded separately is that a person did it — `handEdited` is what the
// next turn is told about (planHandEditNote).
func (a *App) SavePlanText(sessionID, text string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "ห้องนี้ยังไม่มีบทสนทนา"
	}
	if strings.TrimSpace(text) == "" {
		// Refused rather than accepted as "delete the plan". Emptying the box
		// and saving is far more likely to be a slip than an intention, and the
		// plan is the one thing on screen that a run depends on.
		return "แผนว่างเปล่า — ถ้าจะลบแผน ให้บอกผู้ช่วยแทน"
	}
	old, err := a.loadPlan(sessionID)
	if err != nil {
		return err.Error()
	}
	plan := parsePlanMarkdown(text, old)
	if len(plan.Sections) == 0 && len(plan.Steps) == 0 {
		return "อ่านแผนจากข้อความนี้ไม่ออก — หัวข้อเขียนเป็น **ชื่อหัวข้อ** และขั้นตอนเป็น 1. 2. 3."
	}
	plan.Version = 1
	if old != nil {
		plan.Version = old.Version + 1
	}
	if err := a.savePlan(sessionID, plan); err != nil {
		return err.Error()
	}
	a.markHandEdited(sessionID)
	a.emitPlan(sessionID, plan)
	return ""
}

// handEdits remembers which conversations have a plan the user changed and
// whose model has not been told yet.
//
// In memory, and one flag rather than a log: what the next turn needs to know is
// that the plan moved under it, not how many times. Told at the START of the
// next turn (the owner's call) rather than mid-flight, because a model
// interrupted to be told a document changed argues with it in the middle of a
// thought.
func (a *App) markHandEdited(sessionID string) {
	a.goalRunSet().mu.Lock()
	defer a.goalRunSet().mu.Unlock()
	if a.handEdited == nil {
		a.handEdited = map[string]bool{}
	}
	a.handEdited[sessionID] = true
}

// planHandEditNote is the sentence folded into the next turn, or "" when the
// plan has not been touched by hand since the model last saw it.
//
// Consumed on read: the model is told once. A note that kept arriving would be
// a fact about the past re-announced as news every turn.
func (a *App) planHandEditNote(sessionID string) string {
	a.goalRunSet().mu.Lock()
	defer a.goalRunSet().mu.Unlock()
	if !a.handEdited[sessionID] {
		return ""
	}
	delete(a.handEdited, sessionID)
	plan, err := a.loadPlan(sessionID)
	if err != nil || plan == nil {
		return ""
	}
	return fmt.Sprintf("The user edited the plan by hand since your last turn (it is now v%d). "+
		"Read it with `plan` (action: read) before acting on anything you remember about it — "+
		"what you remember is the old one.", plan.Version)
}
