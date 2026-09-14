---
name: aetox-run-plan
before: carrying out a written plan or a set of tickets task by task, alone or through subagents
description: ตอนมีแผนหรือชุด ticket ที่เขียนไว้แล้ว และต้องลงมือทำทีละงานจนจบ ไม่ว่าจะทำเองหรือส่งให้ลูกมือ
source: https://github.com/obra/superpowers (executing-plans, writing-plans, subagent-driven-development), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Run Plan

A plan is run to the end, one task at a time, each task proven before the
next begins. The plan is followed exactly; what it does not settle is asked
about, not guessed.

## Before the first task

1. Read the whole plan critically. A gap that stops a task, an instruction
   you do not understand, a step that contradicts the code as it is: raise
   it now, before any work, and wait for the answer.
2. Confirm the workspace. Work on a branch or a worktree the user agreed to
   (`aetox-worktree`); never start on the main branch without being told to.
3. Turn the tasks into a visible checklist. One item per task, marked as it
   moves.

## What a task must carry

A task is runnable by someone with no context, or it is not ready:

- Files: exact paths to create, to modify (with the lines), and the test.
- Interfaces: what it consumes from earlier tasks and what it produces for
  later ones, as exact names and signatures.
- Steps of one action each: write the failing test, run it and watch it
  fail, write the least code that passes, run it, commit.
- Verification: the command to run and the output that means done.
- The plan's global constraints (versions, names, formats) apply to every
  task without being repeated in it.

A task missing these is not started; the missing part is written first.

## Running a task yourself

Follow the steps as written. Run every verification the task names and read
the output. Stop and ask when a step fails twice, a dependency is missing or
an instruction is unclear; do not force through a blocker, and do not
improve the plan silently while executing it.

## Running a task through a subagent

- A fresh subagent per task. Its brief is that task's text and the
  interfaces it touches, plus the global constraints, and nothing else: no
  session history, no summary of earlier tasks (a real dispatch was 42,000
  characters of pasted history around 400 of task).
- Exact values (numbers, strings, signatures) live in the brief, quoted
  verbatim; never make a subagent read the whole plan.
- One implementer at a time. Never two on code that may touch.
- The implementer does not dispatch subagents of its own, and does not
  review itself in place of the review below.
- It reports one of four: done, done with concerns, needs context, blocked.
  Concerns are read before moving on; missing context is supplied and the
  task re-sent; blocked is assessed by you, never answered by "try again"
  unchanged.

## Review every task, twice

After a task is reported done, a reviewer who did not write it gets the
brief, the report and the diff of that task (from the commit recorded before
dispatch, never `HEAD~1`), and returns two verdicts:

1. Spec compliance: does the change do what the task said, with the exact
   values the task named, and nothing the task did not ask for.
2. Quality: would `aetox-code-review` merge it.

Both are required. Findings go back to the implementer, one fix round at a
time, each re-run and re-reviewed on the fix's own diff. After three rounds
without closing, a fresh implementer on a stronger model; after five, stop
and bring the open findings to the user. Never widen the next task to
absorb what this one left open.

## When all tasks are done

One broad review of the whole branch against the spec, then
`aetox-finish-branch`. Report what was run and what it said, the tasks
completed, and anything left open with why.
