# Technical Specification: [Feature Name]

- **Status**: [Draft | Ready for Review | Approved]
- **Date**: YYYY-MM-DD
- **Target Location**: `docs/specs/[feature-slug].md`
- **Related ADRs**: [e.g. ADR-0001 or None]

---

## 1. Problem Statement

[Describe the exact problem or limitation the user/system is facing, written strictly from the user's perspective. What fails, degrades, or is impossible today?]

---

## 2. Solution Overview

[Describe the proposed solution from the user's perspective. How does this resolve the problem, and what does the improved experience look like?]

---

## 3. User Stories

> **Rule**: Provide an extensive, exhaustive numbered list covering all actors, flows, edge cases, and failure scenarios.

1. **As a** [actor], **I want** [feature/action], **so that** [benefit].
2. **As a** [actor], **I want** [validation / error feedback], **so that** [recovery benefit].
3. **As a** [actor], **I want** [persistence / state preservation], **so that** [benefit].
[... Add all necessary user stories to exhaustively cover the feature ...]

---

## 4. Architectural Seams & Testing Boundaries

> **Rule**: Seams are the highest public boundaries where external behavior is observed without reaching into internals. The fewer seams across the system, the better (ideal is 1).

### Proposed Seams:
- **Primary Seam**: `[Interface / Public Method / API Endpoint]`
  - *Observation Point*: [Where callers invoke and observe results]
  - *Why this seam*: [Rationale for testing at this level]
- **Confirmation with User**: [ ] Confirmed by developer / user before slicing tickets.

---

## 5. Implementation Decisions

> **Rule**: Capture decisions, domain contracts, schemas, and state machines. Avoid ephemeral file paths or line numbers unless an inlined prototype snippet captures a data shape or state transition.

### 5.1 Modules & Interfaces
- **Target Modules**: [List of logical modules modified or introduced]
- **Key Contracts**: [Public interface signatures, data protocols, or API schemas defining the module boundary]

### 5.2 Schema & State Changes
- [Schema definitions, state machines, or data models]

### 5.3 Technical Clarifications & Trade-offs
- [Key trade-offs agreed during the grilling session]

---

## 6. Testing Decisions

- **Testing Philosophy**: Verify observable external behavior through public interfaces; never assert private methods, internal mocks, or recomputed tautologies.
- **Modules Under Test**: [Specific packages/services to test]
- **Prior Art in Codebase**: [Existing test files that serve as the gold standard pattern, e.g. `internal/skill/bundled_skills_test.go`]

---

## 7. Out of Scope (Strictly Enforced)

- ❌ [Out of scope item 1 - deferred or unneeded]
- ❌ [Out of scope item 2 - handled elsewhere]
- ❌ [Out of scope item 3 - speculative optimization]

---

## 8. Further Notes & References

- [Any additional operational notes, security considerations, or documentation references]
