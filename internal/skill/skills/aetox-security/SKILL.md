---
name: aetox-security
before: auditing a system, a codebase, a change or a configuration for security, or answering whether it can be attacked
description: ตอนต้องตรวจความปลอดภัยของระบบ โค้ดเบส การเปลี่ยนแปลง หรือคอนฟิก หรือถูกถามว่าปลอดภัยไหม เจาะได้ไหม รวมถึงแอปที่มีเอเจน เครื่องมือ หรือ MCP
source: anthropics/claude-code-security-review (MIT) for the audit shape and the exclusion list; trailofbits/skills differential-review (CC BY-SA 4.0, method only, no text taken) for blame on removed guards and blast radius; OWASP Top 10 and OWASP Top 10 for Agentic Applications 2026 for the category lists
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Security

A security finding is an attack path someone can walk, shown at a line.
Everything else is hardening advice, and is labelled as such. The two are
never mixed in one list, because the reader is deciding whether to ship.

## Two sizes

**One change** (a diff, PR, commit): read the diff for anything that
touches auth, secrets, crypto, validation, a path, a shell, a query, the
network, or a permission check. Then one hop out: who calls the changed
code, and what reaches each new input. Use `codebase` with `action: impact`
for changed symbols before opening callers or tests; use targeted search only
when the language server cannot resolve the code. For every guard the diff
removes or loosens, `git log -S` on the removed line; a removed check is often
the fix for a bug that is about to come back. A refactor is high risk until
the callers show it changed nothing.

**A system** (a folder, a service, an app): do not read everything. Start
with `codebase` action `map` at the requested scope. If it is cut, map the
relevant subfolder instead of repeating the whole-project map. The map ranks
navigation candidates; it does not prove a trust boundary. Build the security
map in this order:

1. Entry points: HTTP handlers, CLI arguments, files read, messages
   consumed, tool calls the model can make, uploads.
2. Trust boundaries: where data from outside first arrives, and where it is
   first checked.
3. Privileged operations: shell, filesystem writes, network, database,
   money, credentials.
4. Secrets: where they are stored, loaded, logged.

Then walk each path from an entry point to a privileged operation. Depth
follows risk and size: in a small system, cover every such path without
automatically opening every file; up to 200 files, inspect entry points and
one evidence-backed hop; larger, inspect only the critical paths. Read ranges
around the evidence first. Use `codebase` with `action: trace` when a critical
path crosses generated, RPC, or frontend/backend boundaries; its hops are
evidence, not proof that validation is sufficient. Do not trace paths that
targeted source reads already establish. The report names what was not read.

## Before judging

1. Learn how this project already guards itself: its auth middleware, its
   validation layer, its escaping, its secret store. A finding is a place
   that skips the project's own guard, or the guard missing everywhere.
2. Trace, do not pattern-match. `exec` in a file is not a finding; input
   from outside reaching `exec` with no boundary between them is.
3. Run what is there. The tests on the touched code. A scanner if one is
   installed (`references/scanners.md`); a scanner's line is a lead, and
   the code is read before it becomes a finding. Installing a scanner is
   the user's decision; ask.
4. Reproduce only on a copy or a test target the user named, with nothing
   leaving the machine. An attack shown against a live system, or data sent
   to a third-party service to "check it", is not a review step.

## What counts

The category list with the questions to ask for each is
`references/checklist.md`. In one line each: injection (shell, SQL, path,
template, deserialisation); auth and authorization (bypass, another user's
object, privilege escalation, session); secrets and crypto (in source, in
history, weak algorithm, bad randomness, TLS verification off); data
exposure (secrets or personal data in logs, errors, debug endpoints); code
execution (eval, unsafe deserialise, dynamic load of a name from input);
supply chain (unpinned, known-vulnerable, install scripts); configuration
(fail-open defaults, permissive CORS, listening on all interfaces);
TOCTOU on a security decision.

An app with an agent, tools or MCP servers, this one included, has a
second list (the agentic section of the checklist): content the model
reads (a tool result, a fetched page, a file) that can steer it; a tool
reachable around its approval gate; a sandbox root escaped by `..` or a
symlink; a subagent with wider rights than its parent; a memory or skill
file that content the model read can write; a server installed from a name
alone.

## What is not a finding

Adapted from Anthropic's exclusion list, which exists because a review that
reports these is ignored along with everything else in it:

- Denial of service, rate limiting, CPU or memory exhaustion, unless the
  user asked about availability.
- Missing validation on a field nothing dangerous reaches.
- A weakness with no path from an input you can name.
- Style, "best practice", a header missing on an internal tool.
- A secret held in the platform's own store (keychain, DPAPI, a secrets
  manager). A secret in source or in the repository's history is.
- Anything you would not bet an attacker could do. It goes under "worth a
  look", not under findings.

## The shape of a finding

Every finding carries all five, or it is hardening advice:

- Evidence: `file:line`, and the path the input walks, from where to where.
- Attack: what an attacker sends or does, and what they get. One concrete
  scenario, not a category name.
- Severity: `Critical` (unauthenticated, and reaches code execution, other
  users' data, or money) · `High` (authenticated, or one more condition) ·
  `Medium` (needs an unlikely precondition or an insider) · `Low`.
- Confidence: `Direct` (traced end to end, or reproduced) · `Inferred`
  (plausible from reading, not traced), with `Verify first` when someone
  should confirm before acting.
- Fix: the smallest change that closes the path, in the project's own
  idiom. Proposed, not applied, unless the user asks. A fix is checked the
  way a bug fix is (`aetox-verify`): the attack that worked now fails.

A found secret is reported by its location and kind, never quoted.
`None identified` is a complete answer and beats an invented one.

## Report

Ordered for someone about to press a button:

1. One line: **ship** · **fix first** · **needs a deeper audit**, and why.
2. Critical and High findings, in the shape above.
3. Medium and Low, one line each.
4. Worth a look: hardening advice, labelled as advice.
5. What was read, what was run and what it said, what was not covered.

## Hand-offs

- The change is otherwise under review: the verdict belongs to
  `aetox-code-review`; this skill supplies its security section.
- A finding needs to be fixed: `aetox-debug` for the root cause,
  `aetox-verify` before saying it is closed.
- A system with no code yet: `aetox-idea-to-architecture` carries the
  risks section; this skill is not run on a proposal.
