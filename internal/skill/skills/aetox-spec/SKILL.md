---
name: aetox-spec
description: Synthesizes design decisions, architecture agreements, and requirements from grilling or discussion into a formal technical specification (SPEC / RFC). Establishes the highest-level testing seam, generates an exhaustive list of user stories, defines domain contracts and state transitions, and enforces strict out-of-scope boundaries without speculative re-interviewing.
source: https://github.com/Mikedev115 (aetox-spec - inspired by Matt Pocock to-spec)
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Spec (Technical Specification & Blueprint Generator)

Use this skill to convert discussions, design decisions, and architectural agreements into a formal, rigorous Technical Specification (`SPEC.md`).

Inspired by the engineering discipline popularized by Matt Pocock's `to-spec` workflow: when an initiative is too large or critical to build blindly, freeze the decisions into an unambiguous blueprint before slicing tickets or writing code.

## Core Operational Principles

1. **Synthesize, Do Not Re-Interview**:
   The requirement-gathering and grilling phase is already done. Your task is pure synthesis, structuring, and formalization. Do not ask speculative questions unless a fundamental contradiction or critical blocker is discovered.
   - For detailed principles, read `references/spec-discipline.md`.

2. **Propose and Confirm Architectural Seams**:
   Identify the highest-level public seam where the feature will be observed and tested without reaching into internals. The fewer seams across the codebase, the better (ideal is one). **Always check with the user that the proposed seams match their expectations.**

3. **Extensive User Stories**:
   Draft an exhaustive, numbered list of User Stories (`As an <actor>, I want <feature>, so that <benefit>`) covering every actor, happy path, validation failure, and recovery flow.

4. **Implementation & Testing Decisions (No Brittle Snippets)**:
   Document interfaces, data models, state machines, and prior art for tests. Avoid fragile, temporary file paths and line numbers that go stale. If an inlined prototype snippet captures a schema or state machine, inline only the decision-rich structure.

5. **Explicit Out of Scope**:
   Declare what is strictly out of scope to protect downstream implementation from scope explosion.

## Execution Workflow

1. **Inspect Context & Codebase**:
   - Inspect the current conversation for settled decisions.
   - Review related ADRs in `docs/adr/` and glossary definitions in `CONTEXT.md`.
   - Identify prior art in the codebase for patterns and tests.

2. **Propose Seams to User**:
   - Present the primary observation seam and verify alignment with the user.

3. **Synthesize the Specification**:
   - Draft the document following `templates/spec-template.md`.
   - Ensure all sections (Problem Statement, Solution, User Stories, Seams, Implementation Decisions, Testing Decisions, Out of Scope) are filled with high-precision detail.

4. **Persist the Specification**:
   - Create the directory `docs/specs/` if needed.
   - Save the file as `docs/specs/<feature-slug>.md`.
   - Present the saved specification link to the user along with a concise executive summary.

5. **Next Phase Routing**:
   - Once the specification is confirmed, recommend advancing to `aetox-slice` to break the spec into actionable tracer-bullet tickets.
