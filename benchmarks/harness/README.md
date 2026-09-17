# Harness benchmark kit

This directory turns the protocol in
[`docs/architecture/harness-bench-2026-09-14.md`](../../docs/architecture/harness-bench-2026-09-14.md)
into repeatable fixtures. It does not weaken that document's publication rule:
the full claim still requires five tasks, three runs per side, every declared
harness pair, and the recorded versions and raw artifacts.

The first fixture is a controlled pilot for the database path added in
`e440aeb4`:

| Task | Shape | Objective scorer |
|---|---|---|
| `sqlite-email-migration` | Upgrade a Python stdlib SQLite application without rewriting migration history | Six hidden `unittest` cases covering fresh install, upgrade, backfill, canonical lookup, uniqueness, idempotency and collision rollback |

## Fairness boundary

- Both sides receive `tasks/<task>/PROMPT.md` byte-for-byte.
- Both use `gpt-5.6-luna` at `low` through the same ChatGPT account.
- Every run copies the same `base/` into a new Git repository.
- No follow-up message is sent.
- The hidden scorer is copied in only after the harness exits.
- Aetox is built from the recorded repository commit and runs with a fresh
  `AETOX_DATA_ROOT`. The existing encrypted Aetox Codex session is explicitly
  copied into that temporary root, which is deleted after the run. `USERPROFILE` points at an empty
  per-run directory while Aetox runs so user-installed skills do not enter the
  measurement; bundled skills remain.
- Codex runs `--ephemeral --ignore-user-config --ignore-rules`, without web
  search. It uses `workspace-write` plus approval policy `never`; this gives it
  unattended access to the task workspace without giving the benchmark agent
  the whole machine.
- Timeout is 20 minutes. A timeout is a scored failure, not a discarded run.

This is a pilot deviation from H7, not a publishable comparison: one task and
one run per side first proves the command line, authentication, artifact
capture and scorer. After that succeeds, run three rounds for this task. The
five-task/120-run publication bar remains unchanged.

## Run

From the repository root:

```powershell
.\scripts\harness-bench.ps1 -Harness aetox -Task sqlite-email-migration -Run 1 -UseExistingAetoxSession
.\scripts\harness-bench.ps1 -Harness codex -Task sqlite-email-migration -Run 1
```

`-UseExistingAetoxSession` is intentionally explicit: the runner copies only
the encrypted `oauth.json` that Aetox already owns into the temporary profile.
It does not import or rotate the official Codex CLI refresh token. No
credential is copied into the repository or result directory.

Artifacts land under `output/harness-bench/<run-id>/`:

- `stdout.txt`, `stderr.txt`, `last-message.txt`
- `changes.patch` and `changed-files.txt`
- `hidden-tests.txt`
- `result.json`, including versions, commit, duration, exit status, hidden-test
  counts and mechanical line metrics
- `workspace/`, retained for manual inspection

The runner refuses to overwrite an existing run directory. Choose another
`-Run` number or another `-ResultsRoot` rather than mixing attempts.

## Interpret

Hidden-test pass ratio is the primary result. Duration, rounds/tokens where the
harness reports them, changed files and line metrics explain the result but do
not overturn it. A pilot can prove the benchmark machinery and expose a
failure; it cannot establish general harness superiority.
