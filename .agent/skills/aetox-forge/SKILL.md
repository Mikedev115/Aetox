---
name: aetox-forge
description: Implements tasks and tickets using strict Test-Driven Development (TDD) at pre-agreed seams. Enforces observable external behavior over internal coupling, bans tautological assertions, executes continuous test cycles, and passes completed work through code review with clean context boundaries.
source: https://github.com/Mikedev115 (aetox-forge - inspired by Matt Pocock implement & tdd)
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Forge (Test-Driven Implementation Execution)

Use this skill to implement features and changes from a specification or tickets using disciplined Test-Driven Development (TDD) at pre-agreed seams.

Inspired by Matt Pocock's `implement` and `tdd` methodologies: avoid unstructured leaps; enforce observable external behavior through public interfaces, one vertical test-slice at a time.

## Core Operational Principles

1. **Test Only at Pre-Agreed Seams**:
   Tests observe behavior through public interfaces without reaching inside. Never write tests against unexported internals or unconfirmed seams.
   - For detailed guidelines and anti-patterns, read `references/tdd-loop.md`.

2. **Strict Red-Green Loop (Vertical Cycle)**:
   - **Red**: Write a failing test first that matches the ticket's Acceptance Criteria. Run it to confirm failure.
   - **Green**: Implement the minimal production code necessary to make the test pass.
   - Do NOT dump all tests first and all implementation second (horizontal test slicing). Work in vertical cycles: one test ➡️ one minimal implementation ➡️ repeat.

3. **Ban Bad Tests**:
   - **No Implementation-Coupling**: Do not mock internal collaborators or verify private state.
   - **No Tautological Assertions**: Assertions must never recompute expected values using the code under test; they must come from independent ground truth.

4. **Testing Cadence**:
   - Run the single focused test file repeatedly during the loop.
   - Run typecheck / build regularly.
   - Run the entire project test suite once at the conclusion of the ticket.

5. **Post-Implementation Review & Clean Context**:
   - Deliver the walkthrough report using `templates/walkthrough-template.md`.
   - Route to `aetox-code-review` to inspect the diff before committing.
   - Flush or restart conversation context before advancing to the next ticket.

## Execution Workflow

1. **Intake the Ticket**:
   - Load the target ticket from `docs/specs/<feature-slug>/tickets/`.
   - Confirm all blockers are marked `Done`.
   - Identify the agreed test seam.

2. **Execute TDD Cycles**:
   - For each acceptance criterion:
     1. Write the failing test at the seam.
     2. Run the test command and verify it fails for the expected reason.
     3. Write the minimal code to satisfy the test.
     4. Verify test passes.

3. **Regression & Typecheck**:
   - Run the full package / project test suite to guarantee zero regression.

4. **Review & Walkthrough**:
   - Run `aetox-code-review` on the git diff.
   - Document results using `templates/walkthrough-template.md`.
   - Mark the ticket as `done` and identify the next frontier ticket.
