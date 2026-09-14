---
name: aetox-parallel
before: handing two or more independent problems to subagents at the same time, such as several failing test files with different causes
description: ตอนมีปัญหาอิสระตั้งแต่สองเรื่องขึ้นไป เช่น เทสต์หลายไฟล์พังคนละสาเหตุ หรือระบบย่อยหลายตัวเสียโดยไม่เกี่ยวกัน
source: https://github.com/obra/superpowers (dispatching-parallel-agents), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Parallel

One subagent per independent problem, all at once. Investigating three
unrelated failures in sequence spends three times the wall clock and drags
the first problem's context into the third.

## Is it independent

Parallel only when every one of these holds:

- Each problem can be understood without the others. A fix in one cannot
  change the outcome of another.
- No shared state: not the same files, not the same database, not the same
  running process.
- The whole system's state does not need to be held in one head to solve
  any of them.

Related failures, where fixing one might fix the rest, are one problem for
one agent. Two agents that would edit the same file are one agent.

## The brief each one gets

Focused, self-contained, specific about the return:

- Scope: one test file, one subsystem, one bug. Named.
- Goal: what "done" is, as the command that proves it.
- Constraints: do not change code outside the scope; do not touch the
  others' files; report, do not commit, unless told to.
- Return: what was found, what was changed, what the verification said.

All the context the agent needs is in the brief. It has no conversation to
read.

## Dispatch and integrate

Dispatch every one in the same turn; one per turn is sequential, whatever
it was called. When they return: read each report, check that the changes
do not conflict, run the full suite once on the combined result, and only
then call any of it fixed. A report that says "should work" is a report
without evidence; run it.
