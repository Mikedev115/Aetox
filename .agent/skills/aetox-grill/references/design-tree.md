# Design Tree & Frontier Interview Reference

A plan or architecture is not a flat list of requirements. It is a **tree of interdependent decisions**.

---

## 1. The Design Tree Concept

Early architectural choices determine which subsequent questions even make sense.

```
                  [Root: Storage Architecture]
                           /         \
              [Option A: SQLite]   [Option B: Postgres]
                     /                      \
      [In-process locking]           [Connection pooling]
          /          \                   /          \
     [WAL mode]  [Rollback]         [PgBouncer]  [Direct TCP]
```

If you ask about connection pooling before knowing whether the user wants SQLite or Postgres, you waste time and cognitive load on irrelevant decisions.

---

## 2. The Frontier

The **frontier** is the active set of decisions whose prerequisites are already settled.
These are the questions you can ask *right now* without having to guess at answers you have not received yet.

### Rules of Frontier Progression:
1. **Never jump ahead**: A question that depends on another unsettled question belongs to a future round, never the current one.
2. **Batch the Frontier into One Round**: Present all current frontier questions together in one structured post.
3. **Wait for Answers**: Do not write code or speculate before the user responds to the active round.
4. **Recompute the Frontier**: As soon as the user responds, prune the unselected branches and advance the frontier down the selected branches.

---

## 3. Autonomous Fact-Finding vs Decision Routing

A critical failure mode in AI interviews is asking the user questions that exist in the repository.

| Question Type | Who Resolves It | How |
|---|---|---|
| **Facts** (What version of Go? What tables exist? Which packages are imported?) | **The Agent** | Use `search`, `read`, `list`, or `shell` to inspect the codebase directly. Never ask the user. |
| **Decisions** (Should we prioritize throughput or memory? Should we support SQLite or PostgreSQL?) | **The User** | Present the trade-offs, viable options, and your recommendation. |

If an investigation is needed to formulate a question:
- Investigate the codebase first.
- If the exploration is ongoing, do not block unrelated branches of the frontier. Ask the ready questions now.

---

## 4. Question & Recommendation Structure

Every question in a round must provide concrete options and an explicit, technically sound recommendation:

```markdown
❓ **Q1 - [Decision Title]**: [Context, trade-offs, and options A / B / C]

➡️ **Recommendation**: [Recommended option with concise technical rationale based on project constraints]

---

❓ **Q2 - [Decision Title]**: [Context and concrete choices]

➡️ **Recommendation**: [Recommended option with concise technical rationale]
```

Providing recommendations (`➡️`) reduces user fatigue and demonstrates deep technical understanding of the system's architecture.

---

## 5. Completion Criteria

The interview session concludes when:
1. The frontier is empty: Every critical path of the design tree has been navigated.
2. No implicit or unverified assumptions remain hidden.
3. The resulting plan or architecture is unambiguous and ready for surgical implementation.
