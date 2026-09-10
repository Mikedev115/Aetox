---
name: aetox-slice
description: Decomposes a technical specification or plan into vertical tracer-bullet tickets. Calculates directed dependency graphs (DAG), schedules prefactoring first, applies expand-contract sequencing for wide refactors, quizzes the user on ticket granularity, and emits atomic per-ticket files sized for optimal agent context precision.
source: https://github.com/Mikedev115 (aetox-slice - inspired by Matt Pocock to-tickets)
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Slice (Tracer-Bullet Task Decomposition)

Use this skill to decompose a formal specification (`docs/specs/<feature-slug>.md`) or an approved plan into a set of **tickets**: tracer-bullet vertical slices, each declaring the tickets that **block** it.

Inspired by Matt Pocock's `to-tickets` methodology: large tasks cause AI failure; slicing into verifiable, self-contained vertical slices keeps execution inside the agent's high-precision Smart Zone.

## Core Operational Principles

1. **Vertical Tracer Bullets over Layer Cakes**:
   Each slice cuts a narrow but complete path through all required layers (schema, logic, UI, tests). A completed slice is demoable or verifiable on its own.
   - For detailed principles, read `references/tracer-bullet-slicing.md`.

2. **Prefactoring First**:
   "Make the change easy, then make the easy change." If existing code structure resists the new feature, schedule a prefactoring ticket first.

3. **Expand-Contract for Wide Refactors**:
   For changes with a large blast radius across many files, do not attempt an all-at-once edit. Structure as:
   - **Expand**: Introduce the new form beside the old.
   - **Migrate**: Convert call sites in independent batches.
   - **Contract**: Remove the old form once all callers are migrated.

4. **Quiz the User Before Finalizing**:
   Always present the proposed tickets to the user: show Title, Blocked by, and What it delivers. Ask if the granularity is right, if blockers are genuine, and if any tickets should be merged or split.

5. **One File per Ticket**:
   Write each ticket as its own standalone file (`<NN>-<slug>.md`). Never emit a single bloated combined file.

## Execution Workflow

1. **Gather Context**:
   - Inspect the feature specification in `docs/specs/<feature-slug>.md`.
   - Inspect existing codebase conventions and `CONTEXT.md`.

2. **Draft Slices & Blocking Edges**:
   - Number tickets sequentially starting at `01`.
   - Formulate explicit `Blocked by` and `Blocks` edges.

3. **Quiz the User**:
   - Present the numbered breakdown.
   - Wait for user feedback on granularity and dependencies.

4. **Publish the Tickets**:
   - Create `docs/specs/<feature-slug>/tickets/`.
   - Write each ticket using `templates/ticket-template.md`.
   - Generate `docs/specs/<feature-slug>/TICKETS.md` as an index of the DAG.

5. **Next Phase Routing**:
   - Advance to `aetox-forge` to begin Test-Driven Development on the first unblocked ticket at the frontier.
