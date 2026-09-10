---
name: aetox-grill
description: Relentlessly interview and stress-test a plan, architecture, or idea before building. Navigates the design tree in rounds along the frontier, provides concrete options with technical recommendations, investigates codebase facts autonomously, records Architecture Decision Records (ADRs), and establishes an unambiguous domain glossary.
source: https://github.com/Mikedev115 (aetox-grill - inspired by Matt Pocock grill-with-docs)
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Grill (Relentless Plan & Architecture Stress-Test)

Use this skill to relentlessly stress-test and sharpen any plan, technical design, feature proposal, or architectural decision before building. Inspired by the grilling and domain-modeling discipline: no assumptions left unaddressed, no premature implementation, and no guessing.

## Core Operational Principles

1. **Map as a Design Tree**:
   Every architecture is a tree of decisions. Early choices branch into downstream prerequisites. Never jump ahead to child nodes while root choices remain unsettled.
   - For complete navigation rules, read `references/design-tree.md`.

2. **Work in Rounds along the Frontier**:
   The **frontier** is the set of decisions whose prerequisites are already settled: the questions you can ask *now* without guessing at answers you have not heard yet. Ask the whole frontier in one structured round, then wait for the user.

3. **Finding Facts is YOUR Job, Never the User's**:
   When a question depends on what is in the repository (files, dependencies, database schema, configs, external APIs), go and inspect it yourself with file and shell tools. Never ask the user for facts you can look up. Only ask the user for **Decisions, Trade-offs, and Business Priorities**.

4. **Always Provide a Recommended Answer**:
   Do not dump raw open questions. For each question in the round, analyze the technical trade-offs and provide your concrete recommendation (`➡️ Recommendation`).

## Interview Round Format

Present each round cleanly:

```markdown
❓ **Q1 - [Decision Topic]**: [Context, problem statement, and concrete options A / B / C]

➡️ **Recommendation**: [Recommended choice with concise technical rationale]

---

❓ **Q2 - [Decision Topic]**: [Context, trade-offs, and options]

➡️ **Recommendation**: [Recommended choice and rationale]
```

## Advancing the Tree

1. When the user answers a round, settled decisions unblock downstream branches.
2. Recompute the frontier and ask the next round of questions.
3. The interview is complete when the frontier is empty: every critical branch visited, no hidden assumptions left.

## Architectural Decision Records (ADRs)

Only offer to record an ADR when all 3 qualification criteria hold:
1. **Hard to reverse**: The cost of changing later is high.
2. **Surprising without context**: Future maintainers will wonder why it was designed this way.
3. **Real trade-off**: There were valid alternative approaches and one was selected.

- When recording an ADR, follow `templates/adr-format.md`.
- Save under `docs/adr/NNNN-title.md`.

## Domain Vocabulary & Glossary

When domain concepts are ambiguous, overloaded, or contested (e.g. "account" vs "user", or "subscription" vs "license"):
- Challenge fuzzy terminology immediately before coding.
- Formulate a clean ubiquitous language dictionary.
- For formatting `CONTEXT.md` or multi-context `CONTEXT-MAP.md`, follow `templates/context-format.md`.
