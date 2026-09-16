# Implement: `codebase.impact`

Implement one new read-only action, `impact`, inside the existing `codebase` tool pack.

## Goal

`codebase.impact(path, name)` answers the pre-change question:

> If this symbol changes, which source locations, tests, and confirmed generated/RPC surfaces are affected?

It is on-demand only. Do not add background indexing, a database, caching, startup work, or runtime tracing. A project that never invokes `impact` must pay no additional work.

## Required contract

Tool call:

```json
{
  "action": "impact",
  "path": "internal/engine/app.go",
  "name": "SendMessage"
}
```

`path` and `name` are required. Resolve the identifier through the same language-server path used by `codebase.symbol`; do not implement a second symbol resolver and do not fall back to grep when the language server is unavailable.

Use the existing truthful outcomes from `symbol` for:

- unsupported file type;
- language server unavailable;
- missing/escaped path;
- no resolved symbol.

## Scope and output

Output concise, source-grounded sections. Every reported reference/boundary must include its file and line.

```text
Impact: engine.Engine.SendMessage
Declared at internal/engine/app.go:2478

Production references (N)
- path:line

Test references (N)
- path:line

Other references (N)
- path:line

Generated/RPC surface evidence
- path:line

Related narrow checks
- go test ./internal/engine/...
```

Rules:

1. Reuse `lsp.Shared(s.root).Symbol(...)` and its declaration/reference evidence.
2. Partition references deterministically:
   - **Test references:** `*_test.go`, `*.test.ts`, `*.spec.ts`, `test_*.py`, and conventional test directories.
   - **Other references:** documentation, `testdata`, mock fixtures, and generated files.
   - **Production references:** every remaining source reference.
3. The generated/RPC section is evidence only, not a claim that every runtime call crosses a boundary. Include it only for references in clearly recognized generated/API/RPC files, such as the project's `*_gen.go`, `wailsjs/`, or known RPC generated files.
4. List related test files. Recommend a package-level narrow test command only when it can be derived reliably from the test file's package path. Otherwise write: `No narrow automated check was identified.`
5. Do **not** parse enclosing test functions and do **not** generate individual `go test -run` expressions. That is multi-language parser scope and is deliberately deferred.
6. Respect existing line/output truncation behavior.

## Files to change

1. `internal/skill/packed.go`
   - Add `impact` to the `codebase` pack's actions and action-name mapping.

2. `internal/skill/category.go`
   - Add `impact` as `CategoryCode`, alongside existing codebase action names.

3. `internal/mode/stance.go`
   - Add `impact` to `planKeeps`, alongside `diagnostics`, `symbol`, `repo_map`, and `design_check`.

4. `internal/skill/codebase_pack.go`
   - Wire `case "impact"` to `impactSkill`.
   - Add an `impact(path, name)` action description.
   - Ensure the tool schema includes `name` whenever either `symbol` **or** `impact` is allowed. A narrowed profile that permits only `impact` must still receive the required `name` property.

5. `internal/skill/impact.go`
   - Add the read-only implementation.
   - Share private symbol-resolution/output helpers with `symbol.go` where doing so avoids divergent LSP availability, sandbox, and path behavior. Do not change `symbol` behavior.

6. `internal/skill/impact_test.go`
   - Add focused tests.

7. Existing codebase-pack/token-budget tests
   - Update only their expected action/schema/block content to account for the real new action. Do not loosen assertions merely to make them pass.

## Required tests

Cover at least:

- pack registration, dispatch, narrowing, category, and planning-stance access;
- required `path`/`name` validation and sandbox path refusal;
- unsupported/unavailable language server returns the existing explicit not-checked outcome;
- LSP references are partitioned into production, tests, and other;
- recognized generated/RPC evidence is reported with an exact file and line;
- output is truncated through the established tool-output limit;
- existing `codebase` actions remain unchanged.

## Verification

1. Run the new narrow `TestImpact...` tests first.
2. Run the affected `internal/skill` test package.
3. Run the affected `internal/mode` tests.
4. Run diagnostics on every changed Go file.
5. Exercise `codebase.impact` on a real symbol such as `internal/engine/app.go` / `SendMessage`; confirm the output is evidence-based and does not claim unproven runtime flow.

## Explicit non-goals

Do not add or infer:

- a full call graph;
- event producer/consumer mapping;
- config/database ownership;
- Wails binding inference beyond clearly generated-file evidence;
- runtime traces, logs, or profiling;
- bug diagnosis, architecture recommendations, or source edits;
- new standalone tools or persistent project metadata;
- changes to `codebase.map`.
