# Architectural Decision Record (ADR) Format

ADRs live in `docs/adr/` and use sequential 4-digit numbering: `0001-slug.md`, `0002-slug.md`, etc.
Create the `docs/adr/` directory lazily: only when the first decision meeting the qualification criteria is reached.

---

## Qualification Criteria

Only offer to record an ADR when **all three** criteria are met:

1. **Hard to reverse**: The cost or complexity of changing your mind later is significant.
2. **Surprising without context**: A future maintainer will look at the code and ask "why on earth did they do it this way?".
3. **The result of a real trade-off**: There were genuine, viable alternatives and you picked one for specific technical or business reasons.

> If any of the three is missing, skip the ADR. Easy choices can simply be reversed, obvious choices surprise nobody, and one-sided choices have no alternatives worth recording.

### What Qualifies
- System boundary and macro-architecture decisions (e.g. event sourcing vs CRUD, monorepo vs polyrepo).
- Technology choices with high switching costs (database engine, authentication protocol, message queue).
- Deliberate deviations from common industry conventions (e.g. manual SQL over an ORM for query control).
- Non-obvious constraints (regulatory requirements, vendor API rate limits, hardware memory ceilings).

### What Does NOT Qualify
- Routine library choices that can be swapped in an afternoon.
- Trivial code formatting or folder layout preferences.
- Standard framework idioms that follow official documentation.
- Temporary hacks that are planned to be refactored immediately.

---

## ADR Template

```markdown
# ADR-NNNN: [Short Title of Decision]

- **Status**: [proposed | accepted | deprecated | superseded by ADR-XXXX]
- **Date**: YYYY-MM-DD

## Context

[1-3 paragraphs: What problem or architectural tension required a decision? What constraints, legacy code, or operational realities influenced the situation?]

## Decision

[Clear, declarative statement of the chosen direction. State what we WILL do and what we WILL NOT do.]

## Options Considered

- **Option A ([Chosen Option Name])**: [Summary of approach, why it was chosen]
- **Option B ([Alternative 1])**: [Why it was evaluated and why it was rejected]
- **Option C ([Alternative 2])**: [Why it was evaluated and why it was rejected]

## Consequences

- **Positive**: [What becomes easier, safer, or faster]
- **Negative / Costs**: [What becomes harder, new trade-offs introduced, or operational friction]
- **Mitigations**: [How the negative consequences are handled]
```

---

## Review & Superseding Rules

1. **Immutable History**: Once accepted, an ADR is not rewritten to reflect a new direction.
2. **Superseding**: When a previous decision is replaced, write a new ADR (e.g. `0015-use-postgres.md`), mark its status as `accepted`, and update the original ADR's status to `superseded by ADR-0015`.
