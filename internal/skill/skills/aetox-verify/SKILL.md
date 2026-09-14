---
name: aetox-verify
before: telling the user that something works, passes, builds, or is fixed
description: ตอนกำลังจะบอกผู้ใช้ว่าสิ่งใดใช้ได้ ผ่าน บิ้วได้ หรือแก้แล้ว ก่อน commit หรือเปิด PR
source: https://github.com/obra/superpowers (verification-before-completion), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Verify

No claim without fresh evidence. If the command that proves it was not run
in this turn, on this code, the claim is not made.

## The gate

Before saying done, fixed, passing, working:

1. Name the command that would prove it.
2. Run it, whole and fresh. Not the run from before the last edit.
3. Read all of it: the exit code, the failure count, the last lines.
4. Does the output say what you are about to say? If not, say what it does
   say, with the output.
5. Only then, the claim, with the evidence beside it.

## What proves what

| The claim | What proves it | What does not |
|---|---|---|
| Tests pass | the test command, zero failures | an earlier run; "should pass" |
| Build succeeds | the build command, exit 0 | the linter passing; the logs looking fine |
| Linter clean | the linter, zero errors | a partial run extrapolated |
| Bug fixed | the original symptom, reproduced, now gone | the code changed; the theory sounds right |
| Regression test works | seen red, then seen green | passing once |
| It works end to end | the thing, exercised, and its output read | each part passing on its own |

## The words that mean stop

"Should work now", "I am confident", "looks right", "the tests will pass":
each is a claim made before its command. Run the command instead; if it
cannot be run, say that it cannot and why, and say the claim is untested.

Partial verification is reported as partial: what was run, what was not,
and what is therefore unknown.
