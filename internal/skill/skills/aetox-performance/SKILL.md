---
name: aetox-performance
before: finding why something is slow or making it faster, a build, a startup, a screen, a query, a request, a data load
description: ตอนมีอะไรช้า หรือถูกขอให้หาคอขวดและทำให้เร็วขึ้น บิ้ว เปิดแอป หน้าจอ คิวรี รีเควสต์ การโหลดข้อมูล รวมถึงชี้ว่าสถาปัตยกรรมการโหลดข้อมูลผิดตรงไหน
source: vercel-labs/agent-skills vercel-optimize and react-best-practices (MIT) for signal-before-source, the not-investigated list and the waterfall and bundle rules; Brendan Gregg's USE method for the system pass; this repository's BENCHMARK.md for the noise rule
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Performance

A bottleneck is a number at a line. A fix is the same number, smaller by
more than the measurement's own swing. No number, no finding; no second
number, no fix. Everything else is a hunch, and is labelled as one.

## First, the number and its noise

1. Say what "slow" is: which action, measured from where to where, and
   what fast enough would be. When the user cannot say, measure the thing
   they pointed at end to end and put the first number on it.
2. Measure the baseline five times or more, same command, same inputs.
   The spread is the noise. A later change smaller than the spread is not
   a result. This machine swung ±107% on one benchmark with nothing
   changed (BENCHMARK.md); the noise is measured before any fix is judged.
3. Profile before reading. The tool per stack is in `references/tools.md`.
   Read the top of the profile; the first thing to fix is the top entry,
   not the one that looks interesting.
4. Open source files only where the profile or the timing points. A grep
   for anti-patterns is the last resort, and its hits are leads, each one
   read and timed before it is called a cause.

## Where the time usually is

In the order it is usually found:

- Waiting, one after another: a request, a query, a file, a lock, each
  awaited before the next is started when they do not depend on each
  other. The signature is wall time far above CPU time.
- Doing it N times: a query per row, an IPC call per item, a file re-read
  per request, a parse of the same bytes per call.
- Doing too much: the whole table, file or tree loaded for one row; no
  pagination; a list that grew past what its O(n²) loop was written for.
- Doing it again: no cache where the input did not change; the same
  computation on every render or keystroke.
- Doing it in the wrong place: on the UI thread; at startup for something
  needed later; on every request for something static.
- The build: one package or step that takes the time, a generated file, a
  cgo dependency, a bundle that pulls a library for one function. The
  build tool has its own profile (`references/tools.md`); it is read the
  same way.

## When the architecture is the finding

Some slowness has no line to fix: the shape of the loading is the cost.
The signs, with what each looks like in a profile, are in
`references/data-loading.md`. The short list: fetch-on-render waterfalls
across components; one request per item; a startup that loads everything;
the same data fetched in three places with no shared cache; a store with
no index on the column every query filters by; data kept in the shape it
is written in and read a thousand times; a synchronous call per row across
a process boundary.

An architecture finding is written like any other, with its number, and
then names the shape that fixes it and what would change. It is not
rebuilt in the same turn: a reshape is a design change and goes through
`aetox-brainstorm` for the yes first.

## Fixing

- One change at a time, measured against the same baseline with the same
  command and the same number of runs. Kept only when it moved beyond the
  noise; otherwise reverted, and the report says it did not move.
- The order of fixes: do not do it · do it once · do it at once (in
  parallel) · do it later (defer) · do it faster. A fix from the first
  three usually beats any from the last.
- A fix that changes behaviour (a cache that can serve stale, a page size,
  a dropped field) is a design change and is asked about before it is
  made.
- A fix is not traded against correctness or clarity for a number nobody
  asked for.
- The improvement is stated as before → after with the noise beside it. An
  estimate for a fix not yet made is a range from the measured baseline,
  never "much faster".

## What is not a finding

- A pattern with no number: a loop that looks quadratic on twelve items.
- A micro-optimisation below the noise.
- Something on a path nobody runs, unless the user asked about it.
- A dependency that is large but loaded once and off the measured path.

## Report

1. One line: the bottleneck (number, where) and what was done (before →
   after, with the noise), or what is proposed.
2. Findings, each with: Evidence (the number, the profile line,
   `file:line`) · Share of the total · Cause · Fix, applied or proposed ·
   Result, measured.
3. What was measured and how to run it again.
4. What was not investigated, and why. A lead that was seen and not
   followed is listed here, not dropped.

## Hand-offs

- Slowness that is a bug (a leak, a retry loop, a lock held too long):
  `aetox-debug` for the cause, then back here for the number.
- Before saying it is faster: `aetox-verify`, the command run fresh.
- A reshape of how data is loaded: `aetox-brainstorm`, or
  `aetox-idea-to-architecture` when it is a new system.
- The change is otherwise under review: the verdict is
  `aetox-code-review`'s; this skill supplies the numbers.
