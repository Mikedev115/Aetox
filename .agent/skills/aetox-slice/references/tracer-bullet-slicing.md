# Tracer-Bullet Task Slicing & Decomposition Reference

Decomposing a specification into executable tickets requires disciplined slicing. An AI agent fails when tasks are too broad, sliced horizontally, or lack clear blocking relationships.

---

## 1. Vertical Tracer Bullets vs. Horizontal Layers

### The Horizontal Antipattern
- Slicing by architectural layer (e.g. all DB tables, then all handlers, then all UI).
- Fails because nothing is verifiable until the very last ticket, and discrepancies discovered late require reworking every previous layer.

### The Tracer-Bullet Standard
- Each ticket cuts a narrow but **complete** vertical slice across all necessary layers (data, logic, interface, test).
- Every completed ticket delivers verifiable, demoable capability.
- Each ticket is sized to fit comfortably within a single fresh context window (the **Smart Zone**).

---

## 2. Prefactoring: "Make the Change Easy, Then Make the Easy Change"

Before implementing new behavior, evaluate whether existing code needs restructuring:
- If adding the feature is awkward because of existing coupling, schedule a **prefactoring ticket** first.
- Complete the prefactor, verify tests remain green, and only then implement the feature slice.

---

## 3. The Expand-Contract Pattern (Wide Refactors)

**Wide refactors are the explicit exception to vertical slicing.**
A wide refactor is a mechanical change (renaming a database column, altering a core shared interface) whose **blast radius** spans dozens or hundreds of call sites across the entire codebase.

Attempting to change everything in one vertical ticket breaks the build everywhere and guarantees merge hell.

### The Three-Stage Sequence:
1. **Expand**: Add the new interface or function alongside the old one. Both exist simultaneously. Nothing breaks; CI stays green.
2. **Migrate**: Migrate call sites in batches (grouped by package or directory). Each batch is an independent ticket blocked by the Expand ticket.
3. **Contract**: Once zero callers use the old form, delete the old implementation in a final cleanup ticket blocked by all migration batches.

---

## 4. Quizzing the User on Granularity & Blockers

Before publishing tickets, quiz the user with the proposed breakdown:
- **Show for each ticket**: Number, Title, Blocked by, and What it delivers.
- **Ask the user**:
  1. *Does the granularity feel right?* (Too coarse or too fine?)
  2. *Are the blocking edges correct?* (Does each ticket only depend on tickets that genuinely gate it?)
  3. *Should any tickets be merged or split further?*

Iterate on the breakdown until the user confirms the plan.

---

## 5. Storage & Frontier Execution

- **One File Per Ticket**: Write tickets as separate files under `docs/specs/<feature-slug>/tickets/<NN>-<slug>.md` (or `.scratch/<feature-slug>/issues/`).
- **Never publish a single combined file**: Combined files force the agent to ingest unrelated tickets, wasting context.
- **Work the Frontier**: Always pick the ticket whose blockers are all completed.
