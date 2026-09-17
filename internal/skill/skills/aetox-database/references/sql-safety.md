# SQL construction and execution safety

## Provenance before execution

Classify SQL as hard-coded repository SQL, user-authored SQL, or untrusted
generated/external SQL. Model output, pasted snippets, migration-generator
output and database content that becomes SQL are untrusted until inspected.
Never generate and execute opaque SQL in one step: first expose the exact
statement, resolved target and parameters, then inspect them.

This follows the provenance boundary used by Supabase's open-source safe SQL
execution skill:
https://github.com/supabase/supabase/blob/master/.agents/skills/safe-sql-execution/SKILL.md

## Values, identifiers and structure are different

- Bind every data value through the driver or ORM parameter API.
- Ordinary bind parameters do not quote table names, column names, sort
  direction or SQL keywords. Use a driver-provided identifier helper or map a
  strict allowlist of external choices to hard-coded identifiers.
- Do not build `IN` lists, filters, order clauses or migration statements with
  raw string interpolation.
- Keep credentials out of command lines and output. Prefer process-scoped
  environment injection supported by the repository without printing it.

## Read queries still need bounds

Select only required columns, add a narrow predicate and limit exploratory
results. Inspect an estimated plan before an executing plan. `EXPLAIN ANALYZE`
executes the statement; it is not a safe preview for a mutation and can still
be expensive for a read.

## Mutations carry assertions

Before `UPDATE` or `DELETE`, run the equivalent bounded `SELECT` and record the
expected count. Put the same predicate in the mutation, inspect affected rows,
and verify the postcondition. Reject an unexpected count rather than widening
the query until it runs.

Use database constraints for invariants and uniqueness, not only application
checks. For contention-sensitive writes prefer a unique constraint,
conditional update, upsert, compare-and-swap version or appropriate lock over
a read-then-write race.

## Transactions are short

Do not wait for user input, network calls, filesystem work or long computation
while holding a transaction. Set engine-appropriate statement/lock/busy
timeouts for rehearsals, keep work bounded, and define retry behavior for
serialization, deadlock or busy errors. A retry must be safe to repeat.

