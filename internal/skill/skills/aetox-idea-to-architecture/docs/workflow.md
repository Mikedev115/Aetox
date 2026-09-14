# Workflow

Use this workflow to convert a raw idea into proposal artifacts without implying
an existing implementation.

## 1. Intake

Capture the raw idea in the user's language.

Separate:

- User-provided facts
- Stated goals
- Stated constraints
- Missing context

Record the initial idea scope so validation can detect proposal expansion.

Run a complexity scan and record grill depth:

- Low: single workflow, no payment, no authentication, no external integration,
  and no regulated data.
- Medium: 2-4 modules, simple integration, or basic roles.
- High: payment, authentication, multi-role access, regulated data,
  multi-service boundaries, or compliance.

Intake gate: pass only when the idea is restated, initial scope is recorded,
and user-provided facts are separated from unknowns.

## 2. Extract Intent

Identify:

- Primary outcome
- Users or actors
- Business goal
- System goal
- Constraints and non-goals
- Success signals

Intent gate: pass only when goals, actors, constraints, and success signals are
identified before questioning.

## 3. Problem Framing

Create a How Might We frame:

```txt
How might we [desired outcome] for [target user] without [constraint]?
```

The frame must identify:

- Target user
- Problem
- Constraint
- Success signal

Use explicit user-provided values when available. Use safe assumptions only
when they are reversible. Mark unsafe or boundary-changing gaps as `Unknown`
and ask focused questions before architecture proposal work.

Reject frames that are:

- Solution-first: embeds a specific implementation instead of the problem.
- Too broad: lacks a clear target user, outcome, or boundary.
- Constraint-free: lacks a real tension, limitation, or success signal.

## 4. Ask Questions

Ask only architecture-impacting questions. Prefer fewer, sharper questions.

When safe, each important question should include:

- Recommended default
- Why that default is useful
- Alternative option
- Impact if the user chooses differently
- Whether the recommendation requires approval

Use assumptions when the answer can be safely revised later. Pause for approval
when the answer changes boundaries, data ownership, compliance, cost, payment
flow, or security posture.

Use grill depth to set question volume:

- Low: ask 2-3 questions maximum.
- Medium: ask 4-6 questions.
- High: ask until all boundary-changing unknowns are resolved or explicitly
  deferred with approval.

If the user does not answer, proceed only with assumptions marked `Assumed`
and `Requires approval`; do not present the proposal as approved.

Question gate: pass only when architecture-impacting questions are asked, or
`None required` is written. Important questions should include a recommended
default unless the decision is unsafe to recommend.

## 5. Diverge Before Propose

Use this step only when the idea is fuzzy, broad, or solution-first. Skip it
when the user gives a clear direction.

Generate 2-4 `Proposed` product or system directions. Each direction must
include:

- Target user
- Core problem
- Value thesis
- Scope fit
- Major exclusion

Do not generate module structures, database designs, service boundaries, API
contracts, code structure, or implementation tasks.

## 6. Converge Before Architecture

Before architecture proposal work, select or recommend one direction.

Evaluate candidate directions by:

- User value
- Feasibility
- Differentiation
- Scope fit
- Architecture boundary risk
- Assumption severity: `Dealbreaker`, `Important`, or `Nice-to-have`

Keep the selected direction labeled `Proposed` until approved. Architecture
proposal work starts only after a selected proposed direction exists or the
user's direction is already clear.

## 7. State Assumptions

Document assumptions before proposing architecture.

Each assumption should include:

- Assumption
- Why it is being made
- Impact if wrong
- Whether approval is required

Assumption gate: pass only when every assumption has impact if wrong, approval
status, and severity where relevant.

## 8. Propose Architecture

Label every item as proposed.

Cover:

- Proposed modules
- Proposed boundaries
- Proposed integrations
- Proposed decision options
- Proposed tradeoffs

Proposal gate: pass only when modules, workflows, data models, integrations,
and boundaries are marked `Proposed` before mapping or drafting detail.

## 9. Map Workflows

Draft proposed workflows from trigger to outcome.

Include:

- Actor
- Trigger
- Steps
- System responsibilities
- Failure paths
- Open questions

## 10. Draft Data Model

Draft proposed entities and relationships.

Call out:

- Ownership
- Sensitive fields
- Audit needs
- Retention questions
- Reporting needs

## 11. Identify Risks

List risks as proposal risks, not existing defects.

Include:

- Risk
- Why it matters
- Mitigation
- Early validation step
- Owner or decision-maker when known

## 12. Validate

Run the validation gate from `SKILL.md` before Report.

Compare final proposal scope to the initial idea scope recorded at Intake. If
the proposal expanded, list what was added and mark each addition as
user-requested, justified by the idea, or requiring approval.

Also verify:

- How Might We frame is actionable, bounded, and not solution-embedded.
- Selected direction is labeled `Proposed`, not approved.
- Assumption severity exists where relevant.
- Not Doing / Proposed Exclusions list exists.
- No implementation planning leaked into output.

Use proposed exclusions to identify scope creep or intentional omissions.

## 13. Report

End with:

- What was produced
- What is assumed
- What needs approval
- What remains unknown
- Safe next actions
