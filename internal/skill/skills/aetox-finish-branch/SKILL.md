---
name: aetox-finish-branch
before: wrapping up a branch or worktree whose work is complete, when it has to be merged, pushed, or left for the user
description: ตอนงานบน branch หรือ worktree เสร็จแล้ว และต้องตัดสินว่าจะ merge เปิด PR หรือทิ้งไว้ให้ผู้ใช้จัดการ
source: https://github.com/obra/superpowers (finishing-a-development-branch), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Finish Branch

Finished work is verified, then the user chooses what happens to it. The
choice is theirs; the cleanup afterwards is yours.

## 1. Prove it is green

Run the full suite, the whole thing, now. Red means the menu does not
appear: report the failures and stop. Nothing is offered for merging on an
assumption that it passes.

## 2. Know where you are

`git rev-parse --git-dir` against `--git-common-dir` says whether this is a
linked worktree; `git branch --show-current` says whether there is a branch
at all. Record the worktree path now, before any `cd`. Find the base branch
the work forked from (the plan, the conversation, the branch's upstream); if
it is not known, ask, because merging into the wrong base is expensive to
undo.

## 3. Offer exactly these

On a branch:

1. Merge into `<base>` locally
2. Push and open a pull request
3. Leave the branch as it is; the user will handle it

On a detached HEAD, only 2 (as a new branch) and 3. Discarding the work is
never on the menu; it happens only when the user asks for it in so many
words. Wait for the answer.

## 4. Do the one they chose

- Merge: from the main checkout, update the base, merge the branch, run the
  suite on the merged result. Red on the merge means stop and leave
  everything in place; nothing has been pushed, so it is all recoverable.
  Green means delete the branch after cleanup.
- Pull request: push, open it with a body that says what changed and what
  was run, hand back the link.
- Leave it: say where the branch and worktree are and stop.

## 5. Clean up

A worktree you created is removed after a merge or a push, never with
uncommitted changes in it. A worktree the user or the harness created is
left alone.
