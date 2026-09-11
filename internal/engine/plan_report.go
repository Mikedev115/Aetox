package engine

// The closing report: the plan's "after".
//
// ## What was missing
//
// A plan has been a thing since §236 — a row, a card, a run with gates. What a
// run left behind was not: when the steps were settled and the finish condition
// answered, the model said so in its last message, the turn ended, and the
// words went wherever the transcript goes. The pane that lists what a chat
// produced (ArtifactsPane) could show the plan and could show nothing about
// whether it had been carried out, and the owner, looking at that pane on 12
// ก.ย., asked for two things in it: *"ควรแสดงแผน และรายงานก็พอ"*.
//
// Antigravity has the same pair and names them (docs/antigravity-study):
// implementation_plan.md before, walkthrough.md after. This is the after.
//
// ## The shape
//
// One report per RUN, not per plan. A plan is carried out in rounds — a run
// pauses at a breakpoint, the user reads what happened, presses ไปต่อ; or it is
// stopped, the plan amended, and run again — and each round is a piece of work
// somebody walked away from and needs told about. Keyed (session_id, run), and
// the run number is simply how many reports this conversation has plus one.
//
// The words are the model's, under three headings mode.ReportHeadings() owns,
// the way the plan's headings are mode.PlanHeadings()'s. The numbers around them
// are NOT the model's: which version of the plan this was, how many steps were
// done and how many failed, how long it took, how often the engine sent the
// turn back. They are copied off the plan and the run at the moment the report
// is written, so a report cannot claim a finish the checklist does not show.
//
// ## Why it is a gate rather than a suggestion
//
// A run is not finished until its report is written (goal_run.go, gate three).
// Mechanical, like gate one: a report exists or it does not, and nothing has to
// read it to know. Without the gate the report would be written when the model
// remembered, which is the failure the plan's own checklist was built to end —
// and a mode whose whole promise is "you can walk away" owes the person who
// walked away a note on the desk when they come back.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Mikedev115/Aetox/internal/mode"
	"github.com/Mikedev115/Aetox/internal/skill"
)

// PlanReport is one round's closing report, as stored and as drawn.
type PlanReport struct {
	// Run numbers the rounds of this conversation from 1. It is the report's
	// name on screen ("รอบที่ 2") and its half of the key.
	Run int `json:"run"`
	// PlanVersion is the plan as it stood when this was written — a later
	// amend changes the plan and not this row, so the card can say which
	// revision the run carried out.
	PlanVersion int           `json:"planVersion"`
	Title       string        `json:"title"`
	Sections    []PlanSection `json:"sections"`
	// The checklist as it stood, counted rather than copied. Done and Failed
	// are both "settled"; Total minus the two is what was still open when the
	// run stopped short.
	Done   int `json:"done"`
	Failed int `json:"failed"`
	Total  int `json:"total"`
	// ElapsedSecs and SentBack are the run's, when there was one — a report
	// written by hand outside a run has neither, and zero reads as "not
	// measured" rather than "instant".
	ElapsedSecs int `json:"elapsedSecs,omitempty"`
	SentBack    int `json:"sentBack,omitempty"`
	// Stopped is why the round ended before the plan did — the pause reason
	// (goalRun.paused) — and "" for a round that ran to the finish. The card
	// draws it as the round's badge, so a report written at a breakpoint does
	// not look like one written at the end.
	Stopped string `json:"stopped,omitempty"`
	At      string `json:"at"` // RFC3339
}

// report writes the closing report for the round in progress.
//
// It reads the plan for the numbers and the run for the clock, and refuses
// only when there is nothing to report on — no plan, or no sections in the
// call. A report with an empty heading is not refused: "What is left" being
// empty is a real answer, and the shape says so.
func (s *planSkill) report(start time.Time, args map[string]any) (skill.Output, error) {
	id := s.sessionID()
	if id == "" {
		return planFail(errors.New("a report belongs to a conversation, and this call has none"))
	}
	plan, err := s.app.loadPlan(id)
	if err != nil {
		return planFail(err)
	}
	if plan == nil {
		return planFail(errors.New("there is no plan in this conversation to report on"))
	}
	sections, err := parseReportSections(args["sections"])
	if err != nil {
		return planFail(err)
	}
	if len(sections) == 0 {
		return planFail(errors.New("plan report needs at least one section: " + strings.Join(mode.ReportHeadings(), " / ")))
	}

	existing, err := s.app.loadPlanReports(id)
	if err != nil {
		return planFail(err)
	}
	rep := PlanReport{
		Run:         len(existing) + 1,
		PlanVersion: plan.Version,
		Title:       plan.Title,
		Sections:    sections,
		Total:       len(plan.Steps),
		At:          time.Now().Format(time.RFC3339),
	}
	for _, st := range plan.Steps {
		switch st.State {
		case planStepDone:
			rep.Done++
		case planStepFailed:
			rep.Failed++
		}
	}
	if run := s.app.goalRunSet().get(id); run != nil {
		rep.SentBack = run.sentBack
		rep.Stopped = run.paused
		if t, err := time.Parse(time.RFC3339, run.startedAt); err == nil {
			rep.ElapsedSecs = int(time.Since(t).Seconds())
		}
		// The gate reads this (goal_run.go): the round has its report, and the
		// turn may end.
		run.reported = true
	}
	if err := s.app.savePlanReport(id, rep); err != nil {
		return planFail(err)
	}
	s.app.emitPlanReport(id, rep)

	// The receipt, like the plan's, is deliberately not the report. The user
	// can already see it, and the one thing the model must NOT do next is type
	// it again into the answer — the rule Antigravity states on every turn and
	// the arithmetic §236.4 gives for this mode in particular.
	out := fmt.Sprintf("report written for round %d (%d/%d steps done", rep.Run, rep.Done, rep.Total)
	if rep.Failed > 0 {
		out += fmt.Sprintf(", %d failed", rep.Failed)
	}
	out += ").\nThe user can see it under ชิ้นงาน. Do not repeat it in your answer — one line saying the work is done and where the report is, and stop."
	return skill.Output{
		Name: "plan", Command: "plan report", Success: true,
		Content: out, RawOutput: out, DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// parseReportSections is parsePlanSections against the report's own headings.
// Same fold, same reason: a model writing "what was done" against a stored
// "What was done" must land on the heading it meant.
func parseReportSections(raw any) ([]PlanSection, error) {
	if raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, errors.New("sections must be a list of {heading, body}")
	}
	canon := map[string]string{}
	for _, h := range mode.ReportHeadings() {
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
	// In the shape's order, whatever order they arrived in — the reason
	// orderPlanSections gives.
	order := mode.ReportHeadings()
	rank := func(h string) int {
		if i := slices.Index(order, h); i >= 0 {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(out, func(a, b PlanSection) int { return rank(a.Heading) - rank(b.Heading) })
	return out, nil
}

// reportMarkdown renders a report the way a copy off the card writes it out.
func reportMarkdown(r PlanReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — round %d\n\n", r.Title, r.Run)
	for _, sec := range r.Sections {
		b.WriteString("**" + sec.Heading + "**\n" + sec.Body + "\n\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// emitPlanReport puts the report in front of the user, stamped with its
// conversation for the reason emitPlan stamps the plan (§187, §234).
func (a *Engine) emitPlanReport(sessionID string, rep PlanReport) {
	if a.ctx == nil && a.emit == nil {
		return
	}
	a.emitEvent("plan:report", SessionEvent[PlanReport]{SessionID: sessionID, Data: rep})
}

// ---------------------------------------------------------------------------
// the rows
// ---------------------------------------------------------------------------

// loadPlanReports reads a conversation's reports, oldest round first. A
// conversation with none answers an empty list, never nil-and-error: "no
// reports yet" is the ordinary state of every plan that has not been run.
func (a *Engine) loadPlanReports(sessionID string) ([]PlanReport, error) {
	if strings.TrimSpace(sessionID) == "" {
		return []PlanReport{}, nil
	}
	db, err := a.database()
	if err != nil {
		return nil, err
	}
	return queryAll(db, "plan reports",
		`SELECT run, plan_version, title, sections, done, failed, total, elapsed_secs, sent_back, stopped, at
		   FROM plan_reports WHERE session_id=? ORDER BY run`, []any{sessionID},
		func(rows *sql.Rows) (PlanReport, error) {
			var r PlanReport
			var sections string
			if err := rows.Scan(&r.Run, &r.PlanVersion, &r.Title, &sections, &r.Done, &r.Failed, &r.Total,
				&r.ElapsedSecs, &r.SentBack, &r.Stopped, &r.At); err != nil {
				return r, err
			}
			// A row that will not parse keeps its numbers and loses its words —
			// the same recovery loadPlan makes, for the same reason: one bad row
			// must not make every later report unreadable.
			_ = json.Unmarshal([]byte(sections), &r.Sections)
			return r, nil
		})
}

func (a *Engine) savePlanReport(sessionID string, r PlanReport) error {
	db, err := a.database()
	if err != nil {
		return err
	}
	blob, err := json.Marshal(r.Sections)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	  INSERT INTO plan_reports
	    (session_id, run, plan_version, title, sections, done, failed, total, elapsed_secs, sent_back, stopped, at)
	  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	  ON CONFLICT(session_id, run) DO UPDATE SET
	    plan_version=excluded.plan_version, title=excluded.title, sections=excluded.sections,
	    done=excluded.done, failed=excluded.failed, total=excluded.total,
	    elapsed_secs=excluded.elapsed_secs, sent_back=excluded.sent_back,
	    stopped=excluded.stopped, at=excluded.at`,
		sessionID, r.Run, r.PlanVersion, r.Title, string(blob), r.Done, r.Failed, r.Total,
		r.ElapsedSecs, r.SentBack, r.Stopped, r.At)
	return err
}

// SessionPlanReports is the window's read, for the pane that lists what a
// chat produced. Oldest round first; the pane reverses.
func (a *Engine) SessionPlanReports(sessionID string) []PlanReport {
	reps, err := a.loadPlanReports(sessionID)
	if err != nil {
		return []PlanReport{}
	}
	return reps
}
