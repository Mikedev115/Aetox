---
name: aetox-idea-to-architecture
before: designing or laying out a system, a product or a feature that has no code yet, from an idea or a goal
description: ตอนผู้ใช้มีไอเดีย ฟีเจอร์ หรือเป้าหมายที่ยังไม่มีโค้ดเลย และต้องการโครงสถาปัตยกรรมที่รีวิวได้; ระบบที่มีอยู่แล้วเป็นงานของ aetox-architect
source: https://github.com/aetox-skills/idea-to-architecture-agent
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Idea to Architecture

## Purpose

Turn raw ideas into reviewable architecture proposals. Preserve the user's
intent, expose uncertainty, and produce lean Markdown artifacts that a human can
approve, revise, or hand off.

## When to Use

Use this skill when the user has a raw idea, product concept, feature request,
business goal, or system goal without an existing implementation.

## When Not to Use

Do not use this skill to inspect an existing codebase, map an existing system,
create file responsibility maps, document current architecture, review
boundaries in implemented software, or claim any proposed architecture already
exists. Use `aetox-architect` for existing systems, mixed
existing/proposed work, architecture handoff, broader architecture mapping, or
boundary review.

## Core Rules

- Preserve user intent before optimizing or reframing it.
- Ask architecture-impacting questions first when answers materially change
  boundaries, data, risk, cost, compliance, or delivery sequence.
- When asking architecture-impacting questions, include a recommended default
  and short tradeoff when it is safe to recommend one.
- Continue with explicit assumptions when safe; stop for approval when an
  assumption could invalidate the architecture.
- Mark every module, workflow, data model, integration, and boundary as
  `Proposed` until approved.
- Separate user-provided facts, assumptions, proposed architecture, open
  questions, risks, and decisions requiring approval.
- Use Markdown as the documentation source of truth.
- Use Mermaid as editable diagram source; SVG is not required for this version.
- Keep outputs lean and proposal-focused.
- Do not pass a checkpoint gate until its required labels or explicit
  limitations are present.

## Checkpoint Gates

Use these gates to keep the proposal flow enforceable:

- Intake gate: the raw idea is restated, initial idea scope is recorded, and
  user-provided facts are separated from unknowns.
- Intent gate: goals, actors, constraints, and success signals are identified
  before questioning.
- Question gate: architecture-impacting questions are asked, or `None required`
  is written. Each important question includes a recommended default when safe,
  plus the tradeoff or approval risk.
- Assumption gate: every assumption is labeled with impact if wrong and
  approval status before proposing architecture. Add assumption severity where
  relevant.
- Proposal gate: modules, workflows, data models, integrations, and boundaries
  are marked `Proposed` before mapping or drafting detail.
- Validation gate: run the validation gate before Report.

## Problem Framing

After Intake, frame the idea as:

```txt
How might we [desired outcome] for [target user] without [constraint]?
```

The frame must identify target user, problem, constraint, and success signal.
Use `User-provided` values when stated directly. Use `Assumed` only when the
inference is safe and reversible. Mark unsafe or boundary-changing gaps as
`Unknown` and ask focused questions before architecture proposal work.

Treat these frames as not ready for architecture proposal:

- Solution-first: embeds a specific implementation instead of the problem.
- Too broad: does not constrain user, outcome, or boundary.
- Constraint-free: lacks a tradeoff, limitation, or success signal.

## Diverge Before Propose

Use bounded divergence only when the idea is fuzzy, broad, or solution-first.
Skip this step when the user gives a clear direction.

Generate 2-4 `Proposed` product or system directions. Each direction must state
target user, core problem, value thesis, scope fit, and major exclusion.

Do not generate module structures, database designs, service boundaries, API
contracts, code structure, or implementation tasks during divergence.

## Converge Before Architecture

Before architecture proposal work, select or recommend one direction.

Evaluate candidate directions by:

- User value
- Feasibility
- Differentiation
- Scope fit
- Architecture boundary risk
- Assumption severity: `Dealbreaker`, `Important`, or `Nice-to-have`

Keep the recommended direction labeled `Proposed` until the user approves it.
Architecture proposal work starts only after a selected proposed direction
exists or the user's direction is already clear.

## Implementation Boundary

This skill stops at reviewable architecture proposal work unless the user
explicitly approves the proposal and asks for implementation planning.

Do not generate implementation tasks, code structure, database migrations, API
contracts, file-level plans, or build steps before proposal approval.

Before implementation planning, output:

- Approved decisions
- Remaining assumptions
- Unknowns
- Safe next step

## Fast-Path Decision Tree

1. If the user mentions an existing repository, codebase, files, services,
   current architecture, boundary review, or mixed existing/proposed context,
   switch to `aetox-architect`.
2. At Intake, run a complexity scan and record grill depth:
   - Low: single workflow, no payment, no authentication, no external
     integration, and no regulated data. Ask 2-3 questions maximum.
   - Medium: 2-4 modules, simple integration, or basic roles. Ask 4-6
     questions.
   - High: payment, authentication, multi-role access, regulated data,
     multi-service boundaries, or compliance. Ask until boundary-changing
     unknowns are resolved or explicitly deferred with approval.
3. If the idea is underspecified but safe assumptions can unblock progress, use
   the grill depth question budget and proceed with clearly labeled
   assumptions.
   If the user does not answer, continue only with assumptions marked
   `Assumed` and `Requires approval`; do not present the proposal as approved.
4. If the idea involves regulated data, payments, security, identity, minors,
   medical/legal/financial stakes, or production migration, list
   approval-required decisions before proposing details.
5. If the user asks for only a small slice, produce the smallest relevant
   artifact instead of the full proposal set.
6. If the user asks for diagrams, provide Mermaid source that labels the
   content as proposed.

## Grill Loop Exit

Stop questioning when all boundary-changing unknowns are either resolved or
explicitly deferred with approval. State remaining assumptions clearly before
proceeding to proposal work. If the user does not answer, proceed only with
explicit assumptions that require approval.

## Workflow

Follow this sequence:

1. Intake: restate the raw idea, record initial idea scope, identify
   user-provided facts, separate unknowns, and run the complexity scan.
   Intake gate must pass before intent extraction.
2. Extract Intent: capture goals, actors, outcomes, constraints, and success
   signals, then create a How Might We problem frame. Intent gate must pass
   before questioning.
3. Ask Questions: ask architecture-impacting questions according to grill depth;
   separate must-answer from can-assume. Question gate must pass before
   assumptions.
4. Diverge Before Propose: if the idea is fuzzy, broad, or solution-first,
   generate 2-4 proposed product/system directions; otherwise skip.
5. Converge Before Architecture: evaluate candidate directions and keep the
   selected direction labeled `Proposed` until approved.
6. State Assumptions: label each assumption and explain why it is safe or needs
   approval. Assumption gate must pass before proposing architecture.
7. Propose Architecture: describe proposed modules, boundaries, integrations,
   and decision options. Proposal gate must pass before mapping workflows or
   drafting data detail.
8. Map Workflows: draft proposed user/system workflows from trigger to outcome.
9. Draft Data Model: list proposed entities, relationships, ownership, and
   sensitive data.
10. Identify Risks: document risks, unknowns, mitigations, and early validation steps.
11. Validate: run the validation gate below before Report.
12. Report: summarize artifacts, approval points, unknowns, and safe next actions.

## Validation Gate

Before finalizing, verify:

1. User intent preserved?
2. Assumptions visible?
3. Proposed architecture clearly marked as proposed?
4. Decisions requiring approval listed?
5. Handoff includes unknowns and safe next actions?
6. How Might We frame is actionable, bounded, and not solution-embedded?
7. Selected direction is labeled `Proposed`, not approved?
8. Assumption severity exists where relevant?
9. Not Doing / Proposed Exclusions list exists?
10. No implementation planning leaked into output?

Also compare the final proposal scope to the initial idea scope recorded at
Intake. If the proposal expanded, list what was added and mark each addition as
user-requested, justified by the idea, or requiring approval. Use the Not Doing
/ Proposed Exclusions list to confirm intentional omissions and scope creep.

## Suggested Artifacts

- `idea-brief.md` for intent, facts, assumptions, and questions.
- `architecture-proposal.md` for proposed system shape and decisions.
- `module-proposal.md` for proposed module boundaries.
- `workflow-proposal.md` for proposed workflows.
- `data-model-draft.md` for proposed entities and relationships.
- `decision-options.md` for alternatives requiring approval.
- `risk-register.md` for risks and validation steps.
- `ai-agent-notes.md` for handoff notes.
