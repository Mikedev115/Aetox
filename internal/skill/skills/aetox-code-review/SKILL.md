---
name: aetox-code-review
before: reviewing a diff, a PR, a commit or a change before it is merged
description: ตอนต้องรีวิว diff, PR, คอมมิต หรือการเปลี่ยนแปลงก่อน merge หรือถูกถามว่าโค้ดนี้ปลอดภัย/ถูกต้องไหม
source: aetox-architect (Step 6 Assess, Operating Rules 1-2, 16-19), adapted for a single change
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Code Review

A review judges a change *in the system it lands in*, not the lines of the
diff. The discipline is the architect's (`aetox-architect`), cut down to one
change: understand before judging, evidence before opinion, the smallest safe
correction, and nothing invented to look thorough.

## What is under review

The diff, PR, commit or files the user named. Nothing named: `git diff HEAD`
is the change, then the last commit if the tree is clean. Ask only when there
is no diff to find.

## Understand before judging

1. For each changed symbol whose contract or behavior may reach other code,
   use `codebase` with `action: impact` first. It identifies production
   callers, tests and generated or RPC surfaces; read only the referenced
   ranges needed to judge the change. When a language server cannot resolve
   the file, use targeted search and state that the caller set is inferred.
   If the verdict depends on how a changed entry point reaches another layer,
   use `action: trace` to prove that path, especially across generated, RPC,
   or frontend/backend boundaries. Do not trace every changed symbol.
2. Read the project's own way of doing the same thing elsewhere: error
   handling, naming, module layout, how data is reached. The change is judged
   against the project's dominant pattern and the framework's documented
   conventions, never against personal taste.
3. Run it. Build and test the touched packages before the verdict. A verdict
   with no run is `Inferred`, and says so.
4. Scan by default: the changed code and one evidence-backed hop out. If the
   change crosses a boundary or the area is unfamiliar, use `codebase` with
   `action: map` on the relevant folder; do not turn that trigger into a
   whole-module read. Open complete files only when they are small or the
   review genuinely depends on the whole file, and say so.

## What to look for

- Correctness: empty and null input, error paths that swallow or drop,
  off-by-one, concurrency, state that can now be reached twice.
- Security: input that reaches a shell, a file path, the network or a query;
  secrets in code; a check that moved and no longer guards what it did.
  When the change touches auth, secrets, crypto or a permission check, or
  the question is "can this be attacked", the security section of the
  report is `aetox-security`'s.
- Convention drift: the change does one way what the project does another
  way everywhere else.
- Flow conflicts: a second source of truth for the same state, a layer
  bypassed, a side effect crossing a boundary the structure claims to keep.
- Performance only where the change sits on a hot path and the cost is
  shown, not supposed.

## The shape of a finding

Every finding carries all five, or it is not a finding:

- Evidence: `file:line`, and what was observed there.
- Impact: what it breaks, slows, or makes unsafe to change.
- Severity: `Critical` (wrong, or contradicts the system's own flow) ·
  `High` (harms change safety now) · `Medium` (taxes future work) · `Low`
  (worth noting, not acting on).
- Confidence: `Direct` or `Inferred`, with `Verify first` when someone should
  confirm it before acting.
- Direction: the smallest safe correction, proposed, not applied. The code
  under review is not edited unless the user asks.

Not a finding: taste with no convention behind it; working code that is
merely different from how you would write it; hypothetical scaling or "best
practice" with no evidence of impact here; anything you cannot trace to a
line. `None identified` is a complete answer and beats an invented one.

## Report

Ordered by reader priority, not by the order you worked in:

1. Verdict: **merge** · **fix first** · **discuss** - one line saying why.
2. Critical and High findings, in the shape above.
3. Medium and Low, briefly.
4. What was run and what it said, failures included in their own words;
   what was not checked and why.

Short. A review is read by someone about to press a button.
