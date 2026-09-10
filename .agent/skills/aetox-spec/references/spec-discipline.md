# Technical Specification Discipline Reference

Writing a formal specification is not documentation after the fact; it is the **synthesis phase** that freezes architectural decisions into an unambiguous blueprint before any code is sliced or executed.

---

## 1. Synthesis vs. Discovery

A common trap in AI-assisted workflows is reopening discovery questions during the specification phase.
- **Grill phase (`aetox-grill`)**: Open, exploratory, adversarial, questioning.
- **Spec phase (`aetox-spec`)**: Closed, synthesizing, contractual, precise.

The agent's role in `aetox-spec` is to:
1. Extract all decisions settled during grilling or discussion.
2. Read the codebase directly for exact package paths, type definitions, and dependencies.
3. Transform informal chat agreements into formal schemas, sequence diagrams, and interface signatures.
4. If a genuine blocker or unanswered architectural contradiction is discovered, halt and ask only that specific blocker. Never restart open-ended brainstorming.

---

## 2. The Non-Goals Rule

A specification without strict Non-Goals is an invitation to scope explosion.

- **Bad**: "We will build a caching system."
- **Good**:
  - **Goal**: In-memory LRU cache with TTL for query results up to 10MB total memory footprint.
  - **Non-Goal**: Distributed Redis sync across multiple instances (deferred to v2).
  - **Non-Goal**: Disk-backed cache persistence across application restarts.

Every non-goal protects the implementation phase from premature optimization and unneeded dependencies.

---

## 3. Data Contracts & State Transitions

Before designing methods or UI buttons, define the data contracts and state machines.

### Criteria for High-Quality Contracts:
- **Field Types**: Explicit Go/TypeScript types, including nullability, pointer semantics, and zero-value behavior.
- **Validation Rules**: Allowed ranges, required fields, regex patterns, and maximum payload sizes.
- **State Invariants**: What state combinations are illegal? (e.g. A task cannot be `Status: "completed"` with `CompletedAt: nil`).

---

## 4. Seams and Failure Modes

Every feature has internal and external seams. An AI agent implementing code will fail if the seams are not defined in the specification.

1. **System Seam**: Where does this code touch the database, file system, or network?
2. **Failure Counterpart**: What happens when the seam fails?
   - Timeout?
   - File permission denied?
   - Corrupted data?
   - Concurrent modification?

A complete spec specifies the exact error return type and user-facing message for each failure counterpart.

---

## 5. Storage Location & Versioning

- Store specs under `docs/specs/<feature-slug>.md`.
- Name files descriptively using lowercase kebab-case (e.g. `docs/specs/auth-token-refresh.md`).
- If an issue tracker is active (GitHub Issues, Linear), publish or reference the issue ID at the top of the spec.
