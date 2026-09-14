package engine

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/mode"
)

// What the model is told about a plan, read off the real assembled prompt and
// tool block of the coding desk — the two stances side by side.
//
// Measured on 14 ก.ย. 2026 before this: วางแผน's 28k-character prompt spent
// ~6k on planning and the rest on paragraphs that did not know it was
// planning, two of which said the opposite ("do it now", "you must run the
// tests"), the whole skills shelf (42%), a second ask-first rule, and the plan
// headings spelled a second time in the tool block. Each is pinned here from
// the outside, the way the owner counted them.

// The `plan` tool's own description says where a plan goes — the rule that
// used to be a prompt paragraph — and does not spell the headings, which the
// schema's enum and วางแผน's direction already do.
func TestThePlanToolSaysWhereAPlanGoes(t *testing.T) {
	a := bootDeskApp(t, mode.Coding)
	var desc string
	for _, d := range a.deskTools().ToolDefinitions() {
		if d.Function.Name == "plan" {
			desc = d.Function.Description
		}
	}
	if desc == "" {
		t.Fatal("the coding desk carries no plan tool")
	}
	for _, want := range []string{"plan card", "never typed into the reply"} {
		if !strings.Contains(desc, want) {
			t.Errorf("the plan tool's description does not say %q — a plan asked for in ลงมือ ends up typed into the answer:\n%s", want, desc)
		}
	}
	if strings.Contains(desc, "Plan headings, in this order") || strings.Contains(desc, "Report headings:") {
		t.Errorf("the tool block spells the headings a second time:\n%s", desc)
	}
}

// Under วางแผน the coding desk's acting paragraphs, the skills index and the
// general ask-first rule are gone from the prompt; under ลงมือ every one of
// them is still there.
func TestAPlanningTurnAtTheCodingDeskHearsOneVoice(t *testing.T) {
	a := bootDeskApp(t, mode.Coding)
	acting := systemPrompt(t, a)
	if _, err := a.SetStance(mode.StancePlan.String()); err != nil {
		t.Fatal(err)
	}
	planning := systemPrompt(t, a)

	contradictions := []string{
		"A turn ends when the work is done",
		"Done means proven by execution",
		"before touching any code",
		"When asked to create something without enough of a brief",
		"Skills installed on this machine, one line each",
		"Keep it small: a viewBox",
	}
	for _, s := range contradictions {
		if !strings.Contains(acting, s) {
			t.Errorf("ลงมือ lost %q", s)
		}
		if strings.Contains(planning, s) {
			t.Errorf("วางแผน is still told %q", s)
		}
	}
	for _, kept := range []string{
		"This turn is planning work",
		"Make minimal, surgical changes",
		"This session is coding work",
	} {
		if !strings.Contains(planning, kept) {
			t.Errorf("วางแผน lost %q", kept)
		}
	}
	if len(planning) >= len(acting) {
		t.Errorf("the planning prompt (%d chars) is not shorter than the acting one (%d)", len(planning), len(acting))
	}
	t.Logf("coding desk prompt: ลงมือ %d chars, วางแผน %d chars", len(acting), len(planning))
}
