---
name: aetox-architect
before: mapping or documenting a system that already exists, or judging a change that reaches across several modules
description: ตอนต้องเข้าใจระบบที่มีอยู่ก่อนลงมือ ให้สำรวจหลักฐานจริง กางขอบเขต โมดูล และการไหล ประเมินหนี้กับความเสี่ยง แล้วสร้างเอกสารสถาปัตยกรรมเท่าที่จำเป็น ใช้กับการส่งต่องาน การเปลี่ยนข้ามหลายโมดูล หรือการทานแผนด้วยโค้ดจริง ไม่ใช้กับไอเดียดิบที่ยังไม่มีระบบ
source: https://github.com/Mikedev115 (senior-architect-agent)
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Architect

Understand an existing system before recommending or changing its architecture.
The output is evidence-backed architecture judgment, not a decorative diagram.

Core flow:

```txt
Intake -> Inspect -> Classify -> Question -> Map -> Assess -> Document -> Validate -> Report
```

## Routing

- Use this skill when existing code, configuration, documentation, tests, or
  user-provided system facts can be inspected.
- For a raw idea with no implemented system, route to
  `aetox-idea-to-architecture`. If it is unavailable, explain the boundary and
  proceed only when the user accepts that every designed element will be
  labeled `Proposed`.
- For a small implementation fix with no architecture impact, report
  `No architecture pass required`, give the reason and any uncertainty, then
  stop this workflow.

## Non-negotiable rules

1. Inspect before concluding or designing.
2. Keep confirmed facts, inferences, assumptions, proposals, unknowns, risks,
   and decisions distinct.
3. Never present a proposal as existing implementation.
4. Judge against documented framework conventions and the project's dominant
   patterns, not personal taste.
5. Every finding needs evidence, impact, severity, confidence, and the smallest
   safe proposed correction.
6. Tool results guide inspection; critical claims still require direct source
   evidence. When they conflict, direct evidence wins.
7. Use the smallest safe pass and artifact set. Promotion requires a real
   trigger, evidence, and the risk of staying smaller.

## Pass level

- **Scan:** bounded, exploratory, or low-risk work. Produce one compact note.
- **Focus:** one module, workflow, subsystem, or clear boundary. Produce one to
  three artifacts.
- **Full:** whole-system or future-agent handoff work, unclear ownership, three
  or more interacting modules, persistence, external integrations, payment,
  authentication, security, deployment, or a major workflow change.

Default to Scan. Inspect narrowly before promoting when the right level is
unclear. Record the selected level, trigger, scope, and output path.

## Step 1: Intake

Identify the user's goal, available evidence, constraints, requested output,
and the narrowest useful scope. Reuse current architecture docs, ADRs, handoff
notes, and Mermaid sources when they are relevant. Do not start with an
architecture conclusion.

## Step 2: Inspect

For an unfamiliar repository, call `codebase` with `action: map` once at the
selected scope. It is a navigation map for choosing evidence, not the final
architecture map. If it is cut, map the relevant subfolder rather than mapping
the whole project again. Skip the map when the exact file and location are
already known or a current map is already in the conversation.

Then narrow deliberately:

- Use `action: symbol` to identify an unfamiliar code name.
- Use `action: impact` when callers, tests, or generated and RPC boundaries
  matter.
- Use `action: trace` when the architecture question is how one place reaches
  another, especially across generated, RPC, or frontend/backend boundaries.
  Do not make it an automatic follow-up to the repository map.
- Use search for literal text, messages, config keys, and formats the language
  server cannot name.
- Open the ranges those tools identify. Read a complete file only when it is
  small or the claim depends on the whole file.
- If `codebase` is unavailable, preserve the same order with tree, targeted
  search, and ranged reads.

Inspect only evidence needed for the selected scope: entry points, relevant
modules, data ownership, interfaces, tests, package and build configuration,
deployment signals, and existing architecture material. The list is not an
instruction to open every matching file. Record unavailable or skipped areas
instead of guessing.

Load `rules/inspection-rules.md` only when the evidence boundary or inspection
budget is unclear.

## Step 3: Classify

Classify only observed areas: frontend, backend, database, background or AI
processes, external services, infrastructure, shared modules, tests, and
unknown areas. Mark absent evidence as `Not observed`. Do not produce the final
architecture map before classification; the Step 2 repository map is only an
inspection aid.

## Step 4: Question

List architecture-impacting unknowns, or write `None identified`. Ask only
questions whose answer could change a boundary, responsibility, flow,
integration, deployment, security posture, recommendation, or documentation
scope. Find repository-answerable facts yourself. Continue with an explicit
inference when safe; never hide uncertainty in confident prose.

If later mapping or validation exposes an architecture-changing unknown,
return here before final conclusions. Load `rules/question-rules.md` when
blocking and non-blocking uncertainty is hard to separate.

## Step 5: Map

Create only the views needed to explain the selected scope: overview, system
boundary, module relationships, data flow, workflow, or file responsibilities.
Use Mermaid only when requested, required for handoff, or clearer than text.
Split large diagrams by question and keep Mermaid as the source of truth for
any generated SVG.

Every component and relationship must trace to inspected files, user-provided
facts, or an explicit `Inferred`, `Assumed`, `Proposed`, `Unknown`, or
`Unverified` label. `codebase` map, symbol, impact, and trace results may reveal
ranked files, incoming references, callers, tests, and evidence-backed paths
across generated boundaries, but can be capped or incomplete. Cross-check
critical relationships with direct ranged reads.

Evidence strength:

- `Direct`: confirmed by source or user-provided fact.
- `Inferred`: derived from evidence but not directly confirmed.
- `Assumed`: an explicit working premise.
- `Unverified`: not sourced yet; verify, label, or remove before a decision.

Add `Verify first: Yes` when future work must confirm a claim before relying on
it. Load `rules/diagram-rules.md` for non-trivial diagrams.

## Step 6: Assess

Assess only what was mapped. Look for architecture debt, responsibility leaks,
framework or project convention drift, and flow conflicts such as multiple
sources of truth, circular dependencies, dead paths, or layer bypasses.

For each finding record:

- Evidence and observed behavior
- Impact on this system
- Severity: `Critical`, `High`, `Medium`, or `Low`
- Confidence: `Direct` or `Inferred`, plus `Verify first: Yes` when needed
- Smallest safe correction, labeled `Proposed` and `Requires approval`

Fan-in, zero-consumer results, cycles, and graph edges are inspection signals,
not findings by themselves. Do not file style preferences, unfamiliar working
code, or hypothetical scale concerns without present impact. Write
`None identified` rather than inventing findings.

Load `rules/assessment-rules.md` when producing a debt register or judging
ambiguous graph signals. Load `docs/anti-patterns.md` when reviewing whether an
architecture output is trustworthy.

## Step 7: Document

Prefer updating a useful current document over adding another. Use the
templates in `templates/` only for outputs the task actually needs. Scan gets
one compact note; Focus gets one to three artifacts; Full gets a larger package
only when its trigger justifies it. State why before exceeding the budget.

The Scan note contains pass level and reason, scope, evidence checked, skipped
areas, facts, inferences or assumptions, findings, questions, risks, and safe
next actions. Load `rules/documentation-rules.md` or
`rules/anti-overengineering-rules.md` only when choosing artifacts or depth is
not obvious. Load `rules/agent-handoff-rules.md` only for a future-agent
handoff.

## Step 8: Validate

Before reporting, answer three gates:

1. **Traceability:** Does every important claim have a source or evidence label?
2. **Scope:** Does final scope match intake? If it expanded, why and was it
   approved?
3. **Handoff:** Are unknowns, risks, and safe next actions explicit, using
   `None identified` where appropriate?

Also confirm that proposals remain proposals, diagrams match the written map,
pass promotions are justified, and the artifact budget was respected.

## Step 9: Report

In Scan Mode, the compact note is the report. Do not repeat it in a larger
format.

For Focus and Full, lead with three to five lines containing any Critical or
High findings with evidence and the primary recommendation. If none exist, say
so and lead with the recommendation. Give deeper detail only when requested or
required by a checkpoint: scope and evidence, pass level, facts, inferences,
all findings, proposals, questions, risks, approvals, validation, artifacts,
and next steps.

Keep operational work concise. Read `docs/philosophy.md` only when the user asks
why the skill works this way or when revising its principles.
