---
name: aetox-database
before: writing or changing a database schema, migration, ORM model, SQL query, seed, backfill, or running a database command
description: ก่อนเขียนหรือแก้ฐานข้อมูล SQL ของโปรเจกต์บนเครื่อง ตั้งแต่ schema, ORM, query, migration, seed และ backfill ให้หา engine/เครื่องมือ/ฐานเป้าหมายจริง ใช้ฐานชั่วคราว ตรวจ SQL ที่ generate และพิสูจน์ผลโดยไม่เผลอแตะฐานภายนอก
source: Supabase agent-skills (MIT), with PostgreSQL, SQLite, MySQL and Prisma official documentation
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Database

This first version covers relational/SQL databases and is local-first.
“Local” is a property of the resolved database
target, not of an environment-variable name or a developer-looking command.
Prove the target before a command can connect; when it is not a file, socket,
loopback address, disposable test instance, or a repository-defined local
container, stop before execution.

## Load only what the task needs

This document is the working contract and is enough when repository evidence
already establishes the target, migration path and engine behavior. Do not
open a reference merely because its category matches the task. Open one only
when it resolves a question that remains after inspecting the project:

- `references/local-workflow.md` when the resolved target or safe rehearsal
  path is still unclear.
- `references/migrations.md` when ordering, compatibility, rollback, backfill
  or idempotency is not established by the repository's migration system.
- `references/sql-safety.md` when constructing or executing ad-hoc or generated
  SQL whose parameter, identifier or mutation boundary is still unclear.
- Exactly one engine note when that engine is present:
  `references/sqlite.md`, `references/postgres.md`, or
  `references/mysql.md`, and only when the decision depends on engine-specific
  behavior.

If several independent references are necessary, issue their `skill_view`
calls together in one reply; do not spend a model round waiting for one before
opening the next.

If the repository uses another engine, derive its rules from that engine's
official documentation and keep the workflow here: establish the target,
inspect first, use the project's migration system, rehearse on a disposable
database, and verify the resulting state.

## Invariants

- Detect the engine, version, driver or ORM, migration history, and resolved
  target from the repository. Never choose a stack because it is familiar.
- Read current schema and migration state before changing either. Existing
  conventions outrank a new abstraction.
- Add a migration; do not rewrite or delete one that may already have run.
- Generate or preview SQL before applying it. Treat generated SQL as untrusted
  until the exact statements and exact target have been inspected.
- Rehearse mutations against a new disposable database or a safe copy. A
  non-disposable local database gets a restorable backup before destructive or
  irreversible work.
- Prefer data-preserving, forward-compatible steps. Use expand → migrate data
  → switch readers and writers → contract for incompatible changes.
- Bind data values. Identifiers use the driver or ORM's identifier facility,
  or a strict allowlist; never interpolate them as ordinary strings.
- Keep transactions short and account for the engine's actual DDL and locking
  semantics. “Wrapped in a transaction” is not evidence of reversibility.
- Do not print credentials or a full connection URL. Report the redacted
  target, migration or statements applied, checks run, and observed result.

## Completion

A database change is complete only when a clean disposable database can be
built through the repository's normal path, the intended schema/data
postconditions are checked, and the relevant application tests pass. If the
real target could not be proven local, report the inspected plan and the
unresolved target instead of running it.

## Hand-offs

- A slow query whose result is already correct goes to `aetox-performance` for
  a baseline and query plan; this skill supplies the engine rules.
- A wrong result, intermittent transaction failure or migration crash goes to
  `aetox-debug` for root cause before changing schema or data.
- A threat-model or injection audit goes to `aetox-security`; this skill's
  parameter and target rules remain the execution boundary.
