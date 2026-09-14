---
name: aetox-worktree
before: starting feature work that must not disturb the user's current checkout, or running a plan that will make many commits
description: ตอนจะเริ่มงานฟีเจอร์ที่ไม่ควรไปยุ่งกับ checkout ปัจจุบันของผู้ใช้ หรือจะรันแผนที่มีหลายคอมมิต
source: https://github.com/obra/superpowers (using-git-worktrees), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Worktree

Work that will take many commits, or that must not touch what the user has
open, happens in a linked worktree on its own branch. Detect first, ask
second, create third.

## Detect

`git rev-parse --git-dir` and `git rev-parse --git-common-dir`: when they
differ, this is already a linked worktree (unless
`git rev-parse --show-superproject-working-tree` prints a path, which means a
submodule, not a worktree). Already isolated means do not make another; say
where you are and on which branch, and go to setup.

## Ask

Unless the user's instructions already say what they want, ask once:
"Set this up in an isolated worktree, so your current branch is untouched?"
No means work in place. A yes once is not a yes for the next job.

## Create

Where: the user's stated place; else an existing `.worktrees/` or
`worktrees/` at the project root; else `.worktrees/`. A project-local
folder must be ignored before anything is created in it: `git check-ignore`,
and if it is not, add it to `.gitignore` and commit that first, or the
worktree's contents end up in the repository.

Then `git worktree add <folder>/<branch> -b <branch>` and work from there. A
permission error means the sandbox refused; say so and work in place.

## Set up, then prove the baseline

Run the project's install (`npm install`, `go mod download`, `pip install
-r …`, whatever the files say) and the test suite once before changing
anything. A red baseline is reported before work starts, not discovered
after it, when it can no longer be told apart from the work.

## Leaving

The worktree is removed when the branch has been merged or pushed
(`aetox-finish-branch`), never before, and never with uncommitted work in
it.
