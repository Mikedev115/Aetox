# Browser-request wording — measured before/after, 2 Oct 2026

## Outcome

The approved wording removed unsolicited browser calls in all three Terra NX1 samples: before `[3, 2, 0]`, after `[0, 0, 0]`. Luna made no unsolicited browser calls in either arm, so these samples cannot demonstrate a reduction for Luna. All twelve NX1 implementations passed the same 16 hidden checks and have successful project-test and production-build receipts.

Explicitly requested browser work remained possible: receipt-audited success was Terra **2/3 → 3/3**, Luna **2/3 → 2/3**. The original scorer misses Luna's successful shell-based Playwright fallback in one after sample; raw results remain unchanged.

This supports the single prompt change, not a universal browser policy or guaranteed speed/token saving. No further prompt/runtime adjustment or model call was made while auditing these results.

## Change and conditions

Committed as `5a5b9e2c`, changing the browser passage and adding prompt regression tests:

> Complete and verify the requested work first. Use a browser for previews or detailed UI review only when asked.

- One contemporary runtime, frozen from the dirty tree at HEAD `0003adf1f7a0d0bb4699ddfce29d12f20ef78a98` on 2 Oct, 00:18 +07:00: 2,692 source/input files, one Go integration-test driver, two isolated coding-mode overrides. This is not certification of that committed tree.
- The old browser passage is the before arm. Only that passage differs in the after arm. Verification, installation and process-ownership instructions remain byte-identical.
- Fixed identity/thinking, installed skills, task fixtures/prompts and scorer inputs. The owner's Thai draft was unchanged. No live dev session or application executable was opened.
- `gpt-5.6-terra` / `gpt-6-luna`, Codex provider, medium, three repetitions per arm per task: **24 retained cells**, finished 03:27 +07:00. No Claude, CLI comparator, fallback model, silent retry or overwritten cell.
- NX1 arms ran sequentially with alternating arm order; different model groups were sequential. Browser-request pairs started concurrently. Durations are driver-process durations; scoring ran separately.
- Models chose their application implementation, dependencies and test suites. Package downloads/caches and provider latency were not controlled. Matched repetition numbers are not matched random seeds.
- Offline prompt verification: old passage failed the new policy guard; retained coding rules passed. After edit: mode package **59 pass**, offline candidate/receipt tests **6 pass**.

## NX1: complete-system task

All rows scored **16/16**, including the scorer's HTTP/HTML and functional checks. All have a completed driver turn; that status alone is not an implementation-quality score.

| Model | Sample | Before seconds | After seconds | Before rounds | After rounds | Unrequested browser calls before → after |
|:--|--:|--:|--:|--:|--:|:--|
| Terra | 1 | 886.942 | 741.210 | 24 | 32 | 3 → 0 |
| Terra | 2 | 802.647 | 658.896 | 21 | 30 | 2 → 0 |
| Terra | 3 | 755.457 | 628.100 | 45 | 31 | 0 → 0 |
| Luna | 1 | 1,356.694 | 1,115.086 | 130 | 94 | 0 → 0 |
| Luna | 2 | 714.682 | 480.530 | 57 | 37 | 0 → 0 |
| Luna | 3 | 782.389 | 845.966 | 77 | 70 | 0 → 0 |

| Model | Median seconds before → after | Range before | Range after |
|:--|:--|:--|:--|
| Terra | 802.647 → 658.896 | 755.457–886.942 | 628.100–741.210 |
| Luna | 782.389 → 845.966 | 714.682–1,356.694 | 480.530–1,115.086 |

Terra finished sooner in all three matched samples (127–146 seconds); its median decreased 17.9%. Luna finished sooner in two samples and later in one; its median increased 8.1%. Luna had no browser calls to eliminate in its baseline. These observations do not establish a browser-caused timing benefit for Luna or guarantee the Terra benefit on other work.

The five Terra-before browser calls were successful `page_check` invocations, with matched saved model receipts, across two samples. Four requested screenshots; two used `script: document.title`. The recorded browser scripts do not demonstrate clicked-flow acceptance. No browser-library candidate was found in the recorded NX1 shell commands in either arm; this is a command inventory, not a universal detector of arbitrary browser launches.

### Token and round measurements

Numbers below are **per-cell medians** across three samples. Input is cumulative across model requests and includes cached input. It is not a dollar bill.

| Model / arm | Cumulative input | Cached input | Uncached input | Output | Rounds |
|:--|--:|--:|--:|--:|--:|
| Terra before | 823,959 | 765,440 | 58,519 | 17,464 | 24 |
| Terra after | 920,526 | 868,352 | 53,037 | 17,667 | 31 |
| Luna before | 1,826,724 | 1,726,464 | 100,260 | 18,399 | 77 |
| Luna after | 1,911,544 | 1,833,472 | 78,072 | 16,643 | 70 |

Across all three NX1 cells per arm:

| Model / arm | Total input | Total cached input | Total uncached input | Total output | Total rounds |
|:--|--:|--:|--:|--:|--:|
| Terra before | 2,782,762 | 2,605,056 | 177,706 | 53,587 | 90 |
| Terra after | 2,768,078 | 2,603,520 | 164,558 | 51,953 | 93 |
| Luna before | 8,968,263 | 8,651,776 | 316,487 | 70,944 | 264 |
| Luna after | 6,151,419 | 5,911,552 | 239,867 | 50,806 | 201 |

Median cumulative input increased for both models despite lower median uncached input. Terra's median rounds increased. Luna's large before-r1 contributes heavily to totals. Do not describe these results as uniformly fewer rounds/input tokens or a measured billing saving.

### Verification retained, coverage not certified

Each NX1 sample has successful project test and build evidence, separate from the hidden scorer. Project tests vary because each implementation is generated independently. For example, the first successful project-suite receipts show Terra before 3/5/5 test cases and after 2/1/3; Luna before 4/1/1 and after a multi-check smoke script/4/1. One named test can contain many assertions; case counts are not comparable coverage scores.

The audit preserves failed attempts as well as later successes. It does not infer build success from a multi-command shell's final exit alone: before/Luna/r1 includes a combined command with tool `ok=1` but a `Build error occurred` output. Later standalone builds succeeded. Hidden checks passed, but this experiment has not certified complete branch coverage, visual quality, every UI flow, or the real five-minute scheduler wakeup.

## Explicit-browser control: small read-only task

The user prompt explicitly requests an available headless browser, page-load status and console/runtime errors. It prohibits file edits, installations and file reports. The HTML throws `BROWSER_CONTROL_EXPECTED` at runtime. A source read alone does not satisfy acceptance.

| Model | Raw success before → after | Receipt-audited success before → after | Browser attempted before → after | Fixture files unchanged |
|:--|:--|:--|:--|:--|
| Terra | 2/3 → 3/3 | 2/3 → 3/3 | 3/3 → 3/3 | All six |
| Luna | 2/3 → 1/3 | 2/3 → 2/3 | 3/3 → 3/3 | All six |

All **12 initial built-in browser attempts** failed to expose DevTools within about 20 seconds. Eight cells then retried the built-in tool successfully, one succeeded through shell-based Playwright, and three returned an honest blocker. This is a browser-startup failure, not evidence that the new wording makes the model refuse an explicit user request. The underlying startup cause, including any contribution from concurrent control pairs, is not established by these receipts.

The retained unsuccessful cells after receipt audit are:

- `BROWSER-REQUEST-before-terra-r1`
- `BROWSER-REQUEST-before-luna-r2`
- `BROWSER-REQUEST-after-luna-r3`

### Scorer fault — separate interpretation, no raw rewrite

`BROWSER-REQUEST-after-luna-r2` is raw-false for browser success because `positive_score` only indexes direct `page_check`/browser tools. It excludes shell execution even though the task permits any available headless browser.

Saved model-visible shell receipt `call_C21zpsskHLQDpyUasQvQTqzF` contains:

```text
exit 0
LOAD: success
TITLE: Browser request control
ERRORS:
pageerror: BROWSER_CONTROL_EXPECTED
```

The command uses installed Playwright, launches Chromium with `headless=True`, opens the task's `index.html`, captures page errors, and closes the browser. The final answer reports this runtime error and the untested scope. Fixture hashes are unchanged. This is a demonstrated success, not a relaxed assertion or source-marker guess.

Raw `result.json` files still show Luna after **1/3**. Receipt interpretation records **2/3** separately. Luna's recovery also includes an invalid Python attempt and a shell path-guard rejection; these failed attempts remain in the evidence. No cell was rerun to improve the result.

## Evidence and boundaries

Evidence root: `output/behavior-bench/browser-request-ab-20261002/`.

- `experiment.json`, `collection-seal.json`, `coding-before.md`, `coding-after.md`: frozen conditions and the isolated wording difference.
- `runner-completed.json`: 24/24 retained, no stop condition.
- `runs/<cell>/result.json`, `score-junit.xml`, `audit.json`, `store.json`, `turn-1.answer.txt`, `changes.patch`: raw outcomes, receipts, usage and changes.
- `audits/interpretation-v1.json`: per-group medians/ranges/totals, paired differences, browser and shell evidence, verification inventory, and SHA-256 fingerprints of its raw inputs. Does not modify scores or model workspaces.

Automatic checking after HTML writes is suppressed by the Go test driver; NX1 uses TSX and the control does not write HTML. Production automatic checking, tool-provided browser guidance and their policy interactions remain outside this measurement. The old 889-second NX1 run is motivation, not this experiment's baseline.

**Decision boundary:** the committed sentence has direct evidence of reducing Terra's unsolicited browser work while preserving the measured functional gate and explicit-request capability. Evidence for Luna is neutral on browser reduction. Do not generalize three samples into a complete browser-policy guarantee, a coverage guarantee or a billing claim.
